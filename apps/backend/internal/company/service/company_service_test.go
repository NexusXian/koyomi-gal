package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"backend/internal/company/dto"
	"backend/internal/company/model"
	"backend/internal/company/repository"
	galgameDTO "backend/internal/galgame/dto"
	galgameModel "backend/internal/galgame/model"
	galgameRepo "backend/internal/galgame/repository"
	"backend/internal/testutil"
)

func TestCompanyServiceRejectsInvalidInputWithoutDatabase(t *testing.T) {
	s := NewCompanyService(nil, nil)
	ctx := context.Background()
	valid := dto.CompanyRequest{Name: "Studio", Slug: "studio", Status: model.StatusActive}
	if _, err := s.Save(ctx, 0, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nil metadata: %v", err)
	}
	for _, mutate := range []func(*dto.CompanyRequest){
		func(r *dto.CompanyRequest) { r.Name = " " },
		func(r *dto.CompanyRequest) { r.Slug = " " },
		func(r *dto.CompanyRequest) { r.Status = "unknown" },
		func(r *dto.CompanyRequest) { zero := uint(0); r.ParentCompanyID = &zero },
		func(r *dto.CompanyRequest) { date := "2026-02-30"; r.FoundedDate = &date },
		func(r *dto.CompanyRequest) { date := "2026-1-01"; r.ClosedDate = &date },
		func(r *dto.CompanyRequest) { date := "0000-01-01"; r.ClosedDate = &date },
		func(r *dto.CompanyRequest) {
			founded, closed := "2020-01-01", "2019-01-01"
			r.FoundedDate = &founded
			r.ClosedDate = &closed
		},
	} {
		r := valid
		mutate(&r)
		if _, err := s.Save(ctx, 0, &r); !errors.Is(err, ErrInvalidInput) && !errors.Is(err, ErrInvalidDate) {
			t.Fatalf("invalid metadata %+v: %v", r, err)
		}
	}
	for _, req := range []*dto.ReplaceGameCompaniesRequest{
		nil, {}, {Companies: []dto.GameCompanyInput{{CompanyID: 1, Role: "invalid"}}},
		{Companies: []dto.GameCompanyInput{{CompanyID: 0, Role: model.RoleDeveloper}}},
	} {
		if _, err := s.ReplaceGameCompanies(ctx, 1, req); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("invalid binding %+v: %v", req, err)
		}
	}
	duplicate := &dto.ReplaceGameCompaniesRequest{Companies: []dto.GameCompanyInput{{CompanyID: 1, Role: model.RoleDeveloper}, {CompanyID: 1, Role: model.RoleDeveloper}}}
	if _, err := s.ReplaceGameCompanies(ctx, 1, duplicate); !errors.Is(err, ErrDuplicateBinding) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := s.ReplaceGameCompanies(ctx, 0, &dto.ReplaceGameCompaniesRequest{Companies: []dto.GameCompanyInput{}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("zero game ID: %v", err)
	}
	for _, query := range []dto.CompanyQuery{{Page: -1}, {Page: 1000001}, {PageSize: -1}, {PageSize: 101}, {Sort: "id"}, {Status: "unknown"}} {
		if _, err := s.List(ctx, query, true); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("invalid list query: %v", err)
		}
	}
	for _, query := range []dto.CompanyGamesQuery{{Page: -1}, {PageSize: 101}, {Role: "invalid"}, {Sort: "id"}} {
		if _, err := s.ListGames(ctx, "studio", query); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("invalid games query: %v", err)
		}
	}
}

func TestCompanyServiceConcurrentSlugConflict(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	s := NewCompanyService(repository.NewCompanyRepository(db), galgameRepo.NewGalgameRepository(db))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	errs := make([]error, 8)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = s.Save(ctx, 0, &dto.CompanyRequest{Name: "Studio", Slug: " STUDIO ", Status: model.StatusActive, Aliases: []string{"Alias"}})
		}()
	}
	close(start)
	wg.Wait()
	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrSlugExists) {
			t.Fatalf("concurrent conflict must be a domain error: %v", err)
		}
	}
	data, err := s.List(ctx, dto.CompanyQuery{}, false)
	if err != nil || successes != 1 || data.Total != 1 || len(data.Items) != 1 || !reflect.DeepEqual(data.Items[0].Aliases, []string{"Alias"}) {
		t.Fatalf("one atomic create must succeed: successes=%d data=%+v err=%v", successes, data, err)
	}
}

func TestCompanyServiceRejectsUnreachableSlugs(t *testing.T) {
	s := NewCompanyService(nil, nil)
	for _, slug := range []string{
		"a/b", " a/b ", `a\b`, ".", " . ", "..", " .. ",
		"a\x00b", "a\tb", "a\nb", "a\rb", "a\u007fb", "a\u0085b", "a\u009fb",
	} {
		t.Run(slug, func(t *testing.T) {
			for _, id := range []uint{0, 1} {
				_, err := s.Save(context.Background(), id, &dto.CompanyRequest{Name: "Studio", Slug: slug, Status: model.StatusActive})
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("id=%d slug=%q: want ErrInvalidInput, got %v", id, slug, err)
				}
			}
		})
	}
}

func TestCompanyServiceAllowsUnicodeAndSpacesInSlugs(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	s := NewCompanyService(repository.NewCompanyRepository(db), galgameRepo.NewGalgameRepository(db))
	ctx := context.Background()
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"  日本 スタジオ  ", "日本 スタジオ"},
		{" 中文 Studio ", "中文 studio"},
		{"École Café", "école café"},
		{"СТУДИЯ Игры", "студия игры"},
		{"Studio\u00a0Name", "studio\u00a0name"},
		{"studio.v2", "studio.v2"},
		{"...", "..."},
	} {
		t.Run(tc.input, func(t *testing.T) {
			req := &dto.CompanyRequest{Name: "Studio", Slug: tc.input, Status: model.StatusActive}
			c, err := s.Save(ctx, 0, req)
			if err != nil || c.Slug != tc.want {
				t.Fatalf("create slug=%q: %+v %v", tc.input, c, err)
			}
			req.Name = "Updated"
			updated, err := s.Save(ctx, c.ID, req)
			if err != nil || updated.Slug != tc.want || updated.Name != "Updated" {
				t.Fatalf("update slug=%q: %+v %v", tc.input, updated, err)
			}
			found, err := s.GetBySlug(ctx, tc.want)
			if err != nil || found.ID != c.ID {
				t.Fatalf("lookup slug=%q: %+v %v", tc.want, found, err)
			}
		})
	}
}

func TestCompanyServiceCreateUpdateSlugAliasesAndCycles(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	s := NewCompanyService(repository.NewCompanyRepository(db), galgameRepo.NewGalgameRepository(db))
	ctx := context.Background()
	founded := "1998-01-01"
	req := &dto.CompanyRequest{Name: " Studio ", Slug: " STUDIO ", NameJa: "スタジオ", NameZh: "工作室", NameEn: "Studio", Description: "Description",
		LogoURL: "https://example.com/logo.png", BannerURL: "https://example.com/banner.png", WebsiteURL: "https://example.com", Country: "JP",
		FoundedDate: &founded, Status: model.StatusActive, Aliases: []string{" Alias ", "Alias", "", "Other"}}
	c, err := s.Save(ctx, 0, req)
	if err != nil || c.ID == 0 || c.Name != "Studio" || c.Slug != "studio" || c.CreatedAt.IsZero() || c.UpdatedAt.IsZero() {
		t.Fatalf("create: %+v %v", c, err)
	}
	r := dto.NewCompanyResponse(c)
	if !reflect.DeepEqual(r.Aliases, []string{"Alias", "Other"}) || r.FoundedDate == nil || *r.FoundedDate != founded || r.ClosedDate != nil || r.ParentCompany != nil || r.Children == nil {
		t.Fatalf("response contract: %+v", r)
	}
	if _, err := s.Save(ctx, 0, req); !errors.Is(err, ErrSlugExists) {
		t.Fatalf("duplicate normalized slug: %v", err)
	}
	childReq := &dto.CompanyRequest{Name: "Child", Slug: "child", Status: model.StatusInactive, ParentCompanyID: &c.ID}
	child, err := s.Save(ctx, 0, childReq)
	if err != nil {
		t.Fatal(err)
	}
	grandchild, err := s.Save(ctx, 0, &dto.CompanyRequest{Name: "Grandchild", Slug: "grandchild", Status: model.StatusClosed, ParentCompanyID: &child.ID})
	if err != nil {
		t.Fatal(err)
	}
	req.ParentCompanyID = &c.ID
	if _, err := s.Save(ctx, c.ID, req); !errors.Is(err, ErrParentCycle) {
		t.Fatalf("self cycle: %v", err)
	}
	req.ParentCompanyID = &grandchild.ID
	if _, err := s.Save(ctx, c.ID, req); !errors.Is(err, ErrParentCycle) {
		t.Fatalf("deep cycle: %v", err)
	}
	unknown := uint(10000)
	req.ParentCompanyID = &unknown
	if _, err := s.Save(ctx, c.ID, req); !errors.Is(err, ErrParentNotFound) {
		t.Fatalf("unknown parent: %v", err)
	}
	req.ParentCompanyID = nil
	childReq.Slug = "studio"
	if _, err := s.Save(ctx, child.ID, childReq); !errors.Is(err, ErrSlugExists) {
		t.Fatalf("update slug collision: %v", err)
	}
	if _, err := s.Save(ctx, unknown, req); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("update missing: %v", err)
	}
	c, err = s.Save(ctx, c.ID, &dto.CompanyRequest{Name: "Updated", Slug: "updated", Status: model.StatusClosed})
	if err != nil || len(c.Aliases) != 0 || c.FoundedDate != nil || c.Country != "" || c.Name != "Updated" {
		t.Fatalf("full replacement: %+v %v", c, err)
	}
	child, err = s.Get(ctx, child.ID)
	if err != nil || child.ParentCompany == nil || child.ParentCompany.Slug != "updated" {
		t.Fatalf("parent summary: %+v %v", child, err)
	}
	if _, err := s.GetBySlug(ctx, "studio"); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("old slug lookup: %v", err)
	}
	data, err := s.List(ctx, dto.CompanyQuery{}, true)
	if err != nil || data.Page != 1 || data.PageSize != 20 || data.Total != 3 {
		t.Fatalf("default pagination: %+v %v", data, err)
	}
	data, err = s.List(ctx, dto.CompanyQuery{Page: 4, PageSize: 1}, true)
	if err != nil || data.Total != 3 || data.Items == nil || len(data.Items) != 0 {
		t.Fatalf("empty page: %+v %v", data, err)
	}
}

func TestCompanyServiceGameBindingsLookupAndDetail(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	gamesRepo := galgameRepo.NewGalgameRepository(db)
	s := NewCompanyService(repository.NewCompanyRepository(db), gamesRepo)
	ctx := context.Background()
	c, err := s.Save(ctx, 0, &dto.CompanyRequest{Name: "Studio", Slug: "studio", Status: model.StatusActive})
	if err != nil {
		t.Fatal(err)
	}
	games := []galgameModel.Galgame{
		{Title: "Alpha", Slug: "alpha", Status: galgameModel.GalgameStatusPublished, RatingAverage: 9},
		{Title: "Beta", Slug: "beta", Status: galgameModel.GalgameStatusPublished, RatingAverage: 8},
		{Title: "Draft", Slug: "draft", Status: galgameModel.GalgameStatusPending},
	}
	if err := db.Create(&games).Error; err != nil {
		t.Fatal(err)
	}
	for _, g := range games {
		result, err := s.ReplaceGameCompanies(ctx, g.ID, &dto.ReplaceGameCompaniesRequest{Companies: []dto.GameCompanyInput{{CompanyID: c.ID, Role: model.RoleDeveloper}, {CompanyID: c.ID, Role: model.RolePublisher}}})
		if err != nil || len(result) != 2 || result[0].Name != "Studio" || result[0].Slug != "studio" {
			t.Fatalf("multiple roles: %+v %v", result, err)
		}
	}
	if _, err := s.ReplaceGameCompanies(ctx, games[0].ID, &dto.ReplaceGameCompaniesRequest{Companies: []dto.GameCompanyInput{{CompanyID: c.ID + 1000, Role: model.RoleDeveloper}}}); !errors.Is(err, ErrUnknownCompanies) {
		t.Fatalf("unknown company: %v", err)
	}
	if _, err := s.ReplaceGameCompanies(ctx, 10000, &dto.ReplaceGameCompaniesRequest{Companies: []dto.GameCompanyInput{}}); !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("missing game: %v", err)
	}
	for _, tc := range []struct {
		query dto.CompanyGamesQuery
		total int64
		id    uint
	}{
		{dto.CompanyGamesQuery{Sort: "rating", PageSize: 1}, 2, games[0].ID},
		{dto.CompanyGamesQuery{Sort: "rating", Page: 2, PageSize: 1}, 2, games[1].ID},
		{dto.CompanyGamesQuery{Keyword: "Alpha", Role: model.RolePublisher}, 1, games[0].ID},
		{dto.CompanyGamesQuery{Role: model.RoleBrand}, 0, 0},
	} {
		data, err := s.ListGames(ctx, "studio", tc.query)
		if err != nil || data.Total != tc.total || data.Items == nil {
			t.Fatalf("games %+v: %+v %v", tc.query, data, err)
		}
		if tc.id != 0 && (len(data.Items) != 1 || data.Items[0].ID != tc.id) {
			t.Fatalf("game pagination: %+v", data)
		}
	}
	if _, err := s.ListGames(ctx, "missing", dto.CompanyGamesQuery{}); !errors.Is(err, ErrCompanyNotFound) {
		t.Fatalf("missing company games: %v", err)
	}
	for _, public := range []bool{true, false} {
		var g *galgameModel.Galgame
		if public {
			g, err = gamesRepo.FindPublishedByID(ctx, games[0].ID)
		} else {
			g, err = gamesRepo.FindByID(ctx, games[0].ID)
		}
		if err != nil {
			t.Fatal(err)
		}
		response := galgameDTO.NewGalgameResponse(g)
		if len(response.Companies) != 2 {
			t.Fatalf("public=%t detail companies: %+v", public, response.Companies)
		}
	}
	result, err := s.ReplaceGameCompanies(ctx, games[0].ID, &dto.ReplaceGameCompaniesRequest{Companies: []dto.GameCompanyInput{}})
	if err != nil || result == nil || len(result) != 0 {
		t.Fatalf("clear bindings: %+v %v", result, err)
	}
	g, err := gamesRepo.FindPublishedByID(ctx, games[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(galgameDTO.NewGalgameResponse(g))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if companies, ok := payload["companies"].([]any); !ok || len(companies) != 0 {
		t.Fatalf("companies must be an empty array: %s", encoded)
	}
}
