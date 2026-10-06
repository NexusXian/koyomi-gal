package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"backend/internal/company/model"
	galgameModel "backend/internal/galgame/model"
	galgameRepo "backend/internal/galgame/repository"
	"backend/internal/testutil"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func TestCompanyRepositoryUniquenessAndAtomicAliases(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	r := NewCompanyRepository(db)
	ctx := context.Background()
	c := &model.Company{Name: "Studio", Slug: "studio", Status: model.StatusActive}
	if err := r.Save(ctx, c, []string{"Alias"}); err != nil {
		t.Fatal(err)
	}
	created, err := r.FindByID(ctx, c.ID, false)
	if err != nil || created == nil || len(created.Aliases) != 1 || created.Aliases[0].CreatedAt.IsZero() {
		t.Fatalf("created alias must have a timestamp: %+v %v", created, err)
	}
	assertConstraint := func(err error, name string) {
		t.Helper()
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.ConstraintName != name {
			t.Fatalf("want constraint %s, got %v", name, err)
		}
	}
	assertConstraint(r.Save(ctx, &model.Company{Name: "Duplicate", Slug: "studio", Status: model.StatusActive}, nil), "companies_slug_unique")
	c.Name = "Must roll back"
	assertConstraint(r.Save(ctx, c, []string{"Duplicate", "Duplicate"}), "company_aliases_unique")
	found, err := r.FindByID(ctx, c.ID, true)
	if err != nil || found.Name != "Studio" || len(found.Aliases) != 1 || found.Aliases[0].Alias != "Alias" {
		t.Fatalf("aliases and metadata must roll back: %+v %v", found, err)
	}
	if err := r.Save(ctx, &model.Company{Name: "Other", Slug: "other", Status: model.StatusActive}, []string{"Alias"}); err != nil {
		t.Fatalf("aliases may be shared by companies: %v", err)
	}
}

func TestCompanyRepositoryCountsFiltersPaginationAndDeletion(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	r := NewCompanyRepository(db)
	ctx := context.Background()
	parent := &model.Company{Name: "Alpha", Slug: "alpha", Country: "JP", Status: model.StatusActive}
	if err := r.Save(ctx, parent, []string{"Alias_100%", "Second alias"}); err != nil {
		t.Fatal(err)
	}
	child := &model.Company{Name: "Beta", Slug: "beta", ParentCompanyID: &parent.ID, Status: model.StatusClosed}
	if err := r.Save(ctx, child, nil); err != nil {
		t.Fatal(err)
	}
	games := []galgameModel.Galgame{{Title: "Published", Slug: "published", Status: galgameModel.GalgameStatusPublished}, {Title: "Draft", Slug: "draft", Status: galgameModel.GalgameStatusPending}}
	if err := db.Create(&games).Error; err != nil {
		t.Fatal(err)
	}
	for _, game := range games {
		bindings, err := r.ReplaceGameCompanies(ctx, game.ID, []model.GameCompany{{CompanyID: parent.ID, Role: model.RoleDeveloper}, {CompanyID: parent.ID, Role: model.RolePublisher}})
		if err != nil {
			t.Fatal(err)
		}
		if len(bindings) != 2 || bindings[0].CreatedAt.IsZero() || bindings[1].CreatedAt.IsZero() {
			t.Fatalf("created game relations must have timestamps: %+v", bindings)
		}
	}
	public, err := r.FindBySlug(ctx, "alpha", true)
	if err != nil || public.GameCount != 1 || public.DeveloperGameCount != 1 || public.PublisherGameCount != 1 || len(public.Children) != 1 {
		t.Fatalf("public counts/children: %+v %v", public, err)
	}
	admin, err := r.FindByID(ctx, parent.ID, false)
	if err != nil || admin.GameCount != 2 {
		t.Fatalf("admin count: %+v %v", admin, err)
	}
	for _, tc := range []struct {
		options ListOptions
		total   int64
		id      uint
	}{
		{ListOptions{Keyword: "alias_100%"}, 1, parent.ID},
		{ListOptions{Keyword: "aliasX100x"}, 0, 0},
		{ListOptions{Country: "JP", Status: model.StatusActive}, 1, parent.ID},
		{ListOptions{Status: model.StatusClosed}, 1, child.ID},
		{ListOptions{Sort: "name", Page: 2}, 2, child.ID},
		{ListOptions{Sort: "game_count"}, 2, parent.ID},
	} {
		tc.options.PublishedOnly = true
		tc.options.PageSize = 1
		if tc.options.Page == 0 {
			tc.options.Page = 1
		}
		items, total, err := r.List(ctx, tc.options)
		if err != nil || total != tc.total {
			t.Fatalf("list %+v: total=%d err=%v", tc.options, total, err)
		}
		if tc.id != 0 && (len(items) != 1 || items[0].ID != tc.id) {
			t.Fatalf("list order: %+v", items)
		}
		if tc.total == 0 && len(items) != 0 {
			t.Fatalf("unexpected items: %+v", items)
		}
	}
	list, total, err := galgameRepo.NewGalgameRepository(db).ListPublished(ctx, galgameRepo.GalgameListOptions{CompanyID: &parent.ID, Page: 1, Limit: 20})
	if err != nil || total != 1 || len(list) != 1 || list[0].ID != games[0].ID {
		t.Fatalf("EXISTS must deduplicate and exclude draft: %+v %d %v", list, total, err)
	}
	detail, err := galgameRepo.NewGalgameRepository(db).FindPublishedByID(ctx, games[0].ID)
	if err != nil || len(detail.Companies) != 2 || detail.Companies[0].Company.Name != parent.Name {
		t.Fatalf("detail companies: %+v %v", detail, err)
	}
	if err := r.Delete(ctx, parent.ID); err != nil {
		t.Fatal(err)
	}
	child, err = r.FindByID(ctx, child.ID, true)
	if err != nil || child.ParentCompanyID != nil || child.ParentCompany != nil {
		t.Fatalf("child must become parentless: %+v %v", child, err)
	}
	var count int64
	for _, table := range []string{"company_aliases", "game_companies"} {
		if err := db.Table(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("cascade %s: count=%d err=%v", table, count, err)
		}
	}
	if err := db.Model(&galgameModel.Galgame{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("games must survive: %d %v", count, err)
	}
	if err := r.Delete(ctx, parent.ID); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("missing deletion: %v", err)
	}
}

func TestCompanyRepositoryReplacementRollsBack(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	r := NewCompanyRepository(db)
	ctx := context.Background()
	c := &model.Company{Name: "Studio", Slug: "studio", Status: model.StatusActive}
	if err := r.Save(ctx, c, nil); err != nil {
		t.Fatal(err)
	}
	game := &galgameModel.Galgame{Title: "Game", Slug: "game"}
	if err := db.Create(game).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReplaceGameCompanies(ctx, game.ID, []model.GameCompany{{CompanyID: c.ID, Role: model.RoleDeveloper}}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		bindings []model.GameCompany
	}{
		{"unknown", []model.GameCompany{{CompanyID: c.ID + 1000, Role: model.RoleDeveloper}}},
		{"role", []model.GameCompany{{CompanyID: c.ID, Role: "invalid"}}},
		{"duplicate", []model.GameCompany{{CompanyID: c.ID, Role: model.RolePublisher}, {CompanyID: c.ID, Role: model.RolePublisher}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := r.ReplaceGameCompanies(ctx, game.ID, tc.bindings); err == nil {
				t.Fatal("expected rejection")
			}
			var bindings []model.GameCompany
			if err := db.Where("game_id = ?", game.ID).Find(&bindings).Error; err != nil || len(bindings) != 1 || bindings[0].Role != model.RoleDeveloper {
				t.Fatalf("replacement must roll back: %+v %v", bindings, err)
			}
		})
	}
	if _, err := r.ReplaceGameCompanies(ctx, game.ID+1000, nil); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("missing game: %v", err)
	}
	if result, err := r.ReplaceGameCompanies(ctx, game.ID, nil); err != nil || len(result) != 0 {
		t.Fatalf("clear bindings: %+v %v", result, err)
	}
}

func TestCompanyRepositoryConcurrentParentCycles(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	r := NewCompanyRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	companies := []*model.Company{{Name: "A", Slug: "a", Status: model.StatusActive}, {Name: "B", Slug: "b", Status: model.StatusActive}}
	for _, c := range companies {
		if err := r.Save(ctx, c, nil); err != nil {
			t.Fatal(err)
		}
	}
	companies[0].ParentCompanyID = &companies[1].ID
	companies[1].ParentCompanyID = &companies[0].ID
	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range companies {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; errs[i] = r.Save(ctx, companies[i], nil) }()
	}
	close(start)
	wg.Wait()
	if !((errs[0] == nil && errors.Is(errs[1], ErrParentCycle)) || (errs[1] == nil && errors.Is(errs[0], ErrParentCycle))) {
		t.Fatalf("one reparent must fail: %v", errs)
	}
}

func TestCompanyRepositorySearchEveryNameAndLiteralAliases(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	r := NewCompanyRepository(db)
	ctx := context.Background()
	c := &model.Company{Name: "Studio", NameJa: "スタジオ", NameZh: "工作室", NameEn: "English Name", Slug: "studio", Status: model.StatusActive}
	if err := r.Save(ctx, c, []string{`Path\100%_Alias`, "Alias one", "Alias two", "O'Brien"}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		keyword string
		total   int64
	}{
		{"STUDIO", 1}, {"スタジオ", 1}, {"工作", 1}, {"english", 1},
		{`Path\100%_Alias`, 1}, {"100%_", 1}, {"Alias", 1}, {"O'Brien", 1},
		{"PathX100YYAlias", 0}, {"' OR 1=1 --", 0},
	} {
		items, total, err := r.List(ctx, ListOptions{Keyword: tc.keyword, Page: 1, PageSize: 20, PublishedOnly: true})
		if err != nil || total != tc.total || int64(len(items)) != tc.total {
			t.Fatalf("keyword=%q: items=%+v total=%d err=%v", tc.keyword, items, total, err)
		}
	}
}

func TestCompanyRepositoryConcurrentWholeReplacement(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	r := NewCompanyRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	companies := []*model.Company{{Name: "A", Slug: "a", Status: model.StatusActive}, {Name: "B", Slug: "b", Status: model.StatusActive}}
	for _, c := range companies {
		if err := r.Save(ctx, c, nil); err != nil {
			t.Fatal(err)
		}
	}
	game := galgameModel.Galgame{Title: "Game", Slug: "game"}
	if err := db.Create(&game).Error; err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range companies {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := r.ReplaceGameCompanies(ctx, game.ID, []model.GameCompany{
				{CompanyID: companies[i].ID, Role: model.RoleDeveloper},
				{CompanyID: companies[i].ID, Role: model.RolePublisher},
			})
			errs[i] = err
			if err == nil && (len(result) != 2 || result[0].CompanyID != companies[i].ID || result[1].CompanyID != companies[i].ID) {
				errs[i] = fmt.Errorf("replacement returned mixed bindings: %+v", result)
			}
		}()
	}
	close(start)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var final []model.GameCompany
	if err := db.Where("game_id = ?", game.ID).Order("role").Find(&final).Error; err != nil || len(final) != 2 || final[0].CompanyID != final[1].CompanyID || final[0].Role != model.RoleDeveloper || final[1].Role != model.RolePublisher {
		t.Fatalf("final bindings must be one whole replacement: %+v %v", final, err)
	}
}

func TestCompanyRepositoryListUsesBatchedQueries(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	r := NewCompanyRepository(db)
	ctx := context.Background()
	parent := &model.Company{Name: "Parent", Slug: "parent", Status: model.StatusActive}
	if err := r.Save(ctx, parent, nil); err != nil {
		t.Fatal(err)
	}
	for i := range 12 {
		c := &model.Company{Name: fmt.Sprintf("Studio %02d", i), Slug: fmt.Sprintf("studio-%d", i), Status: model.StatusActive, ParentCompanyID: &parent.ID}
		if err := r.Save(ctx, c, []string{"Alias"}); err != nil {
			t.Fatal(err)
		}
	}
	var queries []string
	if err := db.Callback().Query().After("gorm:query").Register("company_test_queries", func(tx *gorm.DB) {
		if !tx.DryRun {
			queries = append(queries, tx.Statement.SQL.String())
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove("company_test_queries") })
	items, total, err := r.List(ctx, ListOptions{Page: 1, PageSize: 20, PublishedOnly: true})
	if err != nil || len(items) != 13 || total != 13 {
		t.Fatalf("list: %d %d %v", len(items), total, err)
	}
	if len(queries) > 5 {
		t.Fatalf("expected batched count, companies, aliases, parent and children queries, got %d: %v", len(queries), queries)
	}
}
