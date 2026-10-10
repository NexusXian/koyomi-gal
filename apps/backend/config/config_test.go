package config

import "testing"

func TestLoadNextMoe(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		baseURL string
		want    *NextMoe
		wantErr bool
	}{
		{name: "disabled without key", key: "", baseURL: "not a url"},
		{name: "default base url", key: "nmk_x", want: &NextMoe{APIKey: "nmk_x", BaseURL: defaultNextMoeBaseURL}},
		{name: "trims trailing slash", key: "nmk_x", baseURL: "https://api.example.test/v2/", want: &NextMoe{APIKey: "nmk_x", BaseURL: "https://api.example.test/v2"}},
		{name: "loopback http allowed", key: "nmk_x", baseURL: "http://127.0.0.1:9000/v2", want: &NextMoe{APIKey: "nmk_x", BaseURL: "http://127.0.0.1:9000/v2"}},
		{name: "remote http rejected", key: "nmk_x", baseURL: "http://api.example.test/v2", wantErr: true},
		{name: "query rejected", key: "nmk_x", baseURL: "https://api.example.test/v2?x=1", wantErr: true},
		{name: "credentials rejected", key: "nmk_x", baseURL: "https://u:p@api.example.test/v2", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NEXTMOE_API_KEY", tc.key)
			t.Setenv("NEXTMOE_API_BASE_URL", tc.baseURL)
			got, err := loadNextMoe()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
