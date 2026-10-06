package migrations_test

import (
	"os"
	"strings"
	"testing"

	companyModel "backend/internal/company/model"
	galgameModel "backend/internal/galgame/model"
	"backend/internal/testutil"
)

func TestCompanyMigrationBackfillAndRollback(t *testing.T) {
	testutil.SkipWithoutPostgres(t)
	db := testutil.NewPostgres(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	apply := func(path string) {
		t.Helper()
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Exec(string(sql)).Error; err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	apply("000051_create_companies.down.sql")
	developers := []galgameModel.Developer{
		{ID: 42, Name: "Studio", OriginalName: "スタジオ", Slug: "Legacy 開発 Studio", Description: "About", LogoURL: "logo", Website: "website"},
		{ID: 43, Name: "Other", Slug: "other"},
	}
	if err := tx.Create(&developers).Error; err != nil {
		t.Fatal(err)
	}
	games := []galgameModel.Galgame{
		{Title: "Linked", Slug: "linked", DeveloperID: &developers[0].ID},
		{Title: "Unlinked", Slug: "unlinked"},
	}
	if err := tx.Create(&games).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("INSERT INTO roles (name, code) VALUES ('Admin', 'admin'), ('Super admin', 'super_admin'), ('User', 'user')").Error; err != nil {
		t.Fatal(err)
	}
	apply("000051_create_companies.up.sql")
	var company companyModel.Company
	if err := tx.Preload("Aliases").First(&company, developers[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if company.Name != developers[0].Name || company.NameJa != developers[0].OriginalName || company.Slug != developers[0].Slug || company.Description != developers[0].Description || company.LogoURL != developers[0].LogoURL || company.WebsiteURL != developers[0].Website || !company.CreatedAt.Equal(developers[0].CreatedAt) || company.Status != companyModel.StatusActive || len(company.Aliases) != 1 || company.Aliases[0].Alias != developers[0].OriginalName {
		t.Fatalf("developer metadata must be backfilled: %+v", company)
	}
	if company.Aliases[0].CreatedAt.IsZero() {
		t.Fatal("backfilled aliases must receive created_at defaults")
	}
	var bindings []companyModel.GameCompany
	if err := tx.Find(&bindings).Error; err != nil || len(bindings) != 1 || bindings[0].GameID != games[0].ID || bindings[0].CompanyID != developers[0].ID || bindings[0].Role != companyModel.RoleDeveloper {
		t.Fatalf("developer links must be backfilled once: %+v %v", bindings, err)
	}
	if bindings[0].CreatedAt.IsZero() {
		t.Fatal("backfilled game relations must receive created_at defaults")
	}
	next := companyModel.Company{Name: "Next", Slug: "next", Status: companyModel.StatusActive}
	if err := tx.Create(&next).Error; err != nil || next.ID != 44 {
		t.Fatalf("company sequence must advance past developer IDs: %d %v", next.ID, err)
	}
	var count int64
	if err := tx.Raw(`SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name IN ('company_aliases', 'game_companies')
AND column_name = 'created_at' AND data_type = 'timestamp with time zone'
AND is_nullable = 'NO' AND column_default = 'now()'`).Scan(&count).Error; err != nil || count != 2 {
		t.Fatalf("aliases and relations must have NOT NULL created_at with defaults: %d %v", count, err)
	}
	for _, index := range []struct {
		name       string
		definition string
	}{
		{"idx_company_aliases_alias_trgm", "USING gin (alias gin_trgm_ops)"},
		{"idx_companies_name_trgm", "USING gin (name gin_trgm_ops)"},
		{"idx_companies_name_ja_trgm", "USING gin (name_ja gin_trgm_ops)"},
		{"idx_companies_name_zh_trgm", "USING gin (name_zh gin_trgm_ops)"},
		{"idx_companies_name_en_trgm", "USING gin (name_en gin_trgm_ops)"},
		{"idx_companies_name", "USING btree (name)"},
	} {
		var definition string
		if err := tx.Raw("SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = ?", index.name).Scan(&definition).Error; err != nil || !strings.Contains(definition, index.definition) {
			t.Fatalf("index %s: want %q, got %q: %v", index.name, index.definition, definition, err)
		}
	}
	if err := tx.Table("role_permissions AS rp").Joins("JOIN permissions AS p ON p.id = rp.permission_id").Where("p.code = ?", "company:manage").Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("only admin roles receive company:manage: %d %v", count, err)
	}
	apply("000051_create_companies.down.sql")
	if err := tx.Table("pg_extension").Where("extname = ?", "pg_trgm").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("rollback must retain shared pg_trgm extension: %d %v", count, err)
	}
	var linked galgameModel.Galgame
	if err := tx.Preload("Developer").First(&linked, games[0].ID).Error; err != nil || linked.Developer == nil || linked.Developer.ID != developers[0].ID {
		t.Fatalf("rollback must preserve legacy game/developer links: %+v %v", linked, err)
	}
	if err := tx.Table("permissions").Where("code = ?", "company:manage").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("rollback must remove company permission: %d %v", count, err)
	}
}
