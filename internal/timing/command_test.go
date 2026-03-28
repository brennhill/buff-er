package timing

import "testing"

func TestExtractPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"cargo build --release", "cargo build"},
		{"npm test", "npm test"},
		{"docker compose up -d", "docker compose"},
		{"go test ./...", "go test"},
		{"make", "make"},
		{"ENV=prod cargo build", "cargo build"},
		{"FOO=bar BAZ=qux npm test", "npm test"},
		{"cd /foo && npm test", "npm test"},
		{"git status && cargo build --release", "cargo build"},
		{"echo hello | grep hello", "grep hello"},
		{"ls -la || echo fail", "echo fail"},
		{"npm run build; npm test", "npm test"},
		{"", ""},
		{"   ", ""},
		{`docker compose -f "my file.yml" up`, "docker compose"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractPattern(tt.input)
			if got != tt.want {
				t.Errorf("ExtractPattern(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
