package service

import (
	"strings"
	"testing"

	"github.com/mockhub/mockhub/internal/util"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name string
		tpl  string
	}{
		{"simple name+email", `{"name":"{{name.fullName}}","email":"{{internet.email}}"}`},
		{"repeat list", `{"data":[{{repeat 3}}{"id":{{number.int 1 99}},"name":"{{name.firstName}}"}{{endrepeat}}]}`},
		{"uuid and date", `{"uuid":"{{uuid.uuid}}","ts":"{{date.timestamp}}"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := util.RenderTemplate(tt.tpl)
			if err != nil {
				t.Fatalf("RenderTemplate(%q) error: %v", tt.tpl, err)
			}
			if out == "" {
				t.Fatalf("RenderTemplate(%q) returned empty", tt.tpl)
			}
			if strings.Contains(out, "{{") {
				t.Fatalf("RenderTemplate(%q) left unreplaced tokens: %q", tt.tpl, out)
			}
		})
	}
}

func TestJSONBody(t *testing.T) {
	if v := JSONBody([]byte(`{"a":1}`)); v == nil {
		t.Fatal("JSONBody should parse a JSON object")
	}
	if v := JSONBody(nil); v != nil {
		t.Fatalf("JSONBody(nil) = %v, want nil", v)
	}
	if v := JSONBody([]byte("plain")); v != "plain" {
		t.Fatalf("JSONBody(plain) = %v, want 'plain'", v)
	}
}
