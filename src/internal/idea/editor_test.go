package idea

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveEditor(t *testing.T) {
	tests := []struct {
		name   string
		visual string
		editor string
		want   string
	}{
		{"VISUAL wins over EDITOR", "code --wait", "vim", "code --wait"},
		{"EDITOR used when VISUAL unset", "", "vim", "vim"},
		{"vi fallback when both unset", "", "", "vi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)
			if got := ResolveEditor(); got != tt.want {
				t.Errorf("ResolveEditor() = %q, want %q", got, tt.want)
			}
		})
	}
}

// fakeEditor writes an executable shell script into a temp dir and returns
// its path. The script body sees the buffer path as $1 (shifted when the
// editor value carries extra arguments).
func fakeEditor(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-editor.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEditInEditor(t *testing.T) {
	tests := []struct {
		name          string
		body          string // fake $EDITOR script body
		editorArg     string // extra words appended to the EDITOR value
		input         string
		want          string
		wantUnchanged bool
		wantErr       bool
	}{
		{
			name:  "rewritten buffer returned with trailing LF stripped",
			body:  `printf 'new text\n' > "$1"`,
			input: "old text",
			want:  "new text",
		},
		{
			name:  "exactly one trailing LF stripped — extra blank line is content",
			body:  `printf 'kept\n\n' > "$1"`,
			input: "x",
			want:  "kept\n",
		},
		{
			name:          "untouched buffer returns the seeded decoded text",
			body:          `:`,
			input:         "same text\nsecond line",
			want:          "same text\nsecond line",
			wantUnchanged: true,
		},
		{
			name:          "editor-appended trailing LF on a one-liner is unchanged",
			body:          `printf '\n' >> "$1"`,
			input:         "x",
			want:          "x",
			wantUnchanged: true,
		},
		{
			name:          "untouched LF-terminated text is unchanged via the pre-strip buffer",
			body:          `:`,
			input:         "a\n",
			want:          "a", // strip-one still applies to the returned text
			wantUnchanged: true,
		},
		{
			name:  "deliberate edit of LF-terminated text to bare text is a change",
			body:  `printf 'a' > "$1"`,
			input: "a\n",
			want:  "a",
		},
		{
			name:  "buffer gaining a trailing blank line is a change keeping one LF",
			body:  `printf 'a\n\n' > "$1"`,
			input: "a",
			want:  "a\n",
		},
		{
			name:  "CRLF in buffer normalized to LF",
			body:  `printf 'a\r\nb' > "$1"`,
			input: "x",
			want:  "a\nb",
		},
		{
			name:      "multi-word editor value runs through the shell",
			body:      `printf 'multiword ok\n' > "$2"`, // $1 is the extra flag
			editorArg: " --flag",
			input:     "x",
			want:      "multiword ok",
		},
		{
			name:    "non-zero editor exit returns an error",
			body:    `exit 3`,
			input:   "x",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", "")
			t.Setenv("EDITOR", fakeEditor(t, tt.body)+tt.editorArg)

			got, unchanged, err := EditInEditor(tt.input)
			if tt.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), "idea unchanged") {
					t.Errorf("error %q should mention the idea is unchanged", err)
				}
				return
			}
			if got != tt.want {
				t.Errorf("EditInEditor(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if unchanged != tt.wantUnchanged {
				t.Errorf("EditInEditor(%q) unchanged = %v, want %v", tt.input, unchanged, tt.wantUnchanged)
			}
		})
	}
}

// TestEditInEditor_RemovesTempFile pins the temp-file contract: the buffer
// path matches idea-edit-*.md and the file is gone after the round trip.
func TestEditInEditor_RemovesTempFile(t *testing.T) {
	side := filepath.Join(t.TempDir(), "side")
	t.Setenv("VISUAL", "")
	t.Setenv("SIDE", side)
	t.Setenv("EDITOR", fakeEditor(t, `printf '%s' "$1" > "$SIDE"`))

	if _, _, err := EditInEditor("x"); err != nil {
		t.Fatalf("EditInEditor failed: %v", err)
	}

	b, err := os.ReadFile(side)
	if err != nil {
		t.Fatalf("read side file: %v", err)
	}
	tmpPath := string(b)
	base := filepath.Base(tmpPath)
	if !strings.HasPrefix(base, "idea-edit-") || !strings.HasSuffix(base, ".md") {
		t.Errorf("temp file name %q does not match idea-edit-*.md", base)
	}
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file %q still exists after EditInEditor returned (stat err = %v)", tmpPath, err)
	}
}
