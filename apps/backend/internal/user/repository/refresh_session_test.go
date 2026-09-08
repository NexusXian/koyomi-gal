package repository

import "testing"

func TestParseRefreshSessionValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  RefreshSession
		valid bool
	}{
		{name: "legacy value", value: "42", want: RefreshSession{UserID: 42}, valid: true},
		{name: "versioned value", value: "42:7", want: RefreshSession{UserID: 42, AuthVersion: 7}, valid: true},
		{name: "missing version", value: "42:"},
		{name: "invalid user", value: "0:1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRefreshSessionValue(tt.value)
			if tt.valid && err != nil {
				t.Fatalf("expected valid session, got %v", err)
			}
			if !tt.valid && err == nil {
				t.Fatal("expected invalid session")
			}
			if got != tt.want {
				t.Fatalf("session = %+v, want %+v", got, tt.want)
			}
		})
	}
}
