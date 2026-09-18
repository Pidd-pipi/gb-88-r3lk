package repository

import "testing"

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		wantOK  bool
		wantID  string
	}{
		{"exact match", "/api/users", "/api/users", true, ""},
		{"param match", "/api/users/:id", "/api/users/42", true, "42"},
		{"mismatch length", "/api/users/:id", "/api/users/42/extra", false, ""},
		{"static mismatch", "/api/users", "/api/orders", false, ""},
		{"nested params", "/api/orgs/:orgId/users/:id", "/api/orgs/7/users/9", true, "9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, ok := matchPath(tt.pattern, tt.path)
			if ok != tt.wantOK {
				t.Fatalf("matchPath(%q, %q) ok = %v, want %v", tt.pattern, tt.path, ok, tt.wantOK)
			}
			if tt.wantOK && tt.wantID != "" && params["id"] != tt.wantID {
				t.Fatalf("matchPath(%q, %q) id = %q, want %q", tt.pattern, tt.path, params["id"], tt.wantID)
			}
		})
	}
}
