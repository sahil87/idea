package idea

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- Parsing Tests ---

func TestParseLine_ValidOpen(t *testing.T) {
	line := "- [ ] [a7k2] 2025-06-15: Add dark mode to settings page"
	idea, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected valid parse")
	}
	if idea.ID != "a7k2" {
		t.Errorf("ID = %q, want a7k2", idea.ID)
	}
	if idea.Date != "2025-06-15" {
		t.Errorf("Date = %q, want 2025-06-15", idea.Date)
	}
	if idea.Text != "Add dark mode to settings page" {
		t.Errorf("Text = %q, want 'Add dark mode to settings page'", idea.Text)
	}
	if idea.Done {
		t.Error("Done = true, want false")
	}
}

func TestParseLine_ValidDone(t *testing.T) {
	line := "- [x] [e5f6] 2025-06-08: Fix login redirect bug"
	idea, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected valid parse")
	}
	if idea.ID != "e5f6" {
		t.Errorf("ID = %q, want e5f6", idea.ID)
	}
	if idea.Date != "2025-06-08" {
		t.Errorf("Date = %q, want 2025-06-08", idea.Date)
	}
	if idea.Text != "Fix login redirect bug" {
		t.Errorf("Text = %q, want 'Fix login redirect bug'", idea.Text)
	}
	if !idea.Done {
		t.Error("Done = false, want true")
	}
}

func TestParseLine_Invalid(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"empty", ""},
		{"random text", "some random text"},
		{"header", "# Backlog"},
		{"blank", "   "},
		{"bad checkbox", "- [y] [a7k2] 2025-06-15: Text"},
		{"missing id brackets", "- [ ] a7k2 2025-06-15: Text"},
		{"short id", "- [ ] [a7k] 2025-06-15: Text"},
		{"long id", "- [ ] [a7k2x] 2025-06-15: Text"},
		{"uppercase id", "- [ ] [A7K2] 2025-06-15: Text"},
		// Note: a malformed date (e.g. "2025-6-15:") is no longer "invalid" — with
		// the optional-date relaxation it simply parses as a DATELESS idea whose
		// text begins with the malformed-date string. See TestParseLine_Lenient.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseLine(tt.line)
			if ok {
				t.Errorf("ParseLine(%q) should return false", tt.line)
			}
		})
	}
}

func TestFormatLine_Open(t *testing.T) {
	i := Idea{ID: "a7k2", Date: "2025-06-15", Text: "Add dark mode", Done: false}
	got := FormatLine(i)
	want := "- [ ] [a7k2] 2025-06-15: Add dark mode"
	if got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestFormatLine_Done(t *testing.T) {
	i := Idea{ID: "e5f6", Date: "2025-06-08", Text: "Fix bug", Done: true}
	got := FormatLine(i)
	want := "- [x] [e5f6] 2025-06-08: Fix bug"
	if got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	lines := []string{
		"- [ ] [a7k2] 2025-06-15: Add dark mode to settings page",
		"- [x] [e5f6] 2025-06-08: Fix login redirect bug",
	}
	for _, line := range lines {
		idea, ok := ParseLine(line)
		if !ok {
			t.Fatalf("failed to parse %q", line)
		}
		got := FormatLine(idea)
		if got != line {
			t.Errorf("round-trip failed: got %q, want %q", got, line)
		}
	}
}

// --- Query Matching Tests ---

func TestMatch_ByID(t *testing.T) {
	i := Idea{ID: "a7k2", Text: "Add dark mode"}
	if !Match("a7k2", i) {
		t.Error("expected match by ID")
	}
	if Match("c3d4", i) {
		t.Error("expected no match for wrong ID")
	}
}

func TestMatch_ByText(t *testing.T) {
	i := Idea{ID: "a7k2", Text: "Add dark mode"}
	if !Match("dark", i) {
		t.Error("expected match by text substring")
	}
}

func TestMatch_CaseInsensitive(t *testing.T) {
	i := Idea{ID: "a7k2", Text: "Add dark mode"}
	if !Match("DARK", i) {
		t.Error("expected case-insensitive match on text")
	}
	if !Match("A7K2", i) {
		t.Error("expected case-insensitive match on ID")
	}
}

func TestFindAll(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode", Done: false},
		{ID: "c3d4", Text: "Add light mode", Done: false},
		{ID: "e5f6", Text: "Fix redirect", Done: true},
	}

	tests := []struct {
		name   string
		query  string
		filter FilterKind
		want   int
	}{
		{"match both by mode", "mode", FilterAll, 2},
		{"match one by dark", "dark", FilterAll, 1},
		{"no match", "nonexistent", FilterAll, 0},
		{"filter open", "dark", FilterOpen, 1},
		{"filter done", "", FilterDone, 1},
		{"all with empty query", "", FilterAll, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindAll(tt.query, ideas, tt.filter)
			if len(result) != tt.want {
				t.Errorf("FindAll(%q, filter=%d) = %d results, want %d", tt.query, tt.filter, len(result), tt.want)
			}
		})
	}
}

func TestRequireSingle_OneMatch(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode"},
		{ID: "c3d4", Text: "Fix redirect"},
	}
	i, idx, err := RequireSingle("a7k2", ideas, FilterAll)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.ID != "a7k2" {
		t.Errorf("ID = %q, want a7k2", i.ID)
	}
	if idx != 0 {
		t.Errorf("idx = %d, want 0", idx)
	}
}

func TestRequireSingle_NoMatch(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode"},
	}
	_, _, err := RequireSingle("nonexistent", ideas, FilterAll)
	if err == nil {
		t.Fatal("expected error for no match")
	}
	if !strings.Contains(err.Error(), "No idea matching") {
		t.Errorf("error = %q, want 'No idea matching'", err.Error())
	}
}

func TestRequireSingle_MultipleMatches(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode"},
		{ID: "c3d4", Text: "Add light mode"},
	}
	_, _, err := RequireSingle("mode", ideas, FilterAll)
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "Multiple matches") {
		t.Errorf("error = %q, want 'Multiple matches'", err.Error())
	}
	if !strings.Contains(err.Error(), "Be more specific") {
		t.Errorf("error should contain disambiguation guidance")
	}
}

// --- File Operations Tests ---

func writeBacklog(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "backlog.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFile_MixedContent(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Add dark mode
- [x] [e5f6] 2025-06-08: Fix bug

Some footer text
`
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(f.ideas) != 2 {
		t.Fatalf("ideas count = %d, want 2", len(f.ideas))
	}
	if f.ideas[0].ID != "a7k2" {
		t.Errorf("first idea ID = %q, want a7k2", f.ideas[0].ID)
	}
	if f.ideas[1].ID != "e5f6" {
		t.Errorf("second idea ID = %q, want e5f6", f.ideas[1].ID)
	}
}

func TestLoadFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeBacklog(t, dir, "")

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(f.ideas) != 0 {
		t.Errorf("ideas count = %d, want 0", len(f.ideas))
	}
}

func TestLoadFile_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.md")
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSaveFile_PreservesNonIdeaLines(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Add dark mode

Some footer text
`
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	// Modify the idea
	f.ideas[0].Text = "Add dark mode with toggle"
	if _, err := SaveFile(f, path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "# Backlog") {
		t.Error("header line missing after save")
	}
	if !strings.Contains(result, "Some footer text") {
		t.Error("footer line missing after save")
	}
	if !strings.Contains(result, "Add dark mode with toggle") {
		t.Error("modified idea text missing after save")
	}
}

func TestResolveFilePath(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		env     string
		wantSfx string
	}{
		{"default", "", "", "fab/backlog.md"},
		{"flag override", "custom/ideas.md", "", "custom/ideas.md"},
		{"env override", "", "env/ideas.md", "env/ideas.md"},
		{"flag beats env", "flag/ideas.md", "env/ideas.md", "flag/ideas.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("IDEAS_FILE", tt.env)
			} else {
				t.Setenv("IDEAS_FILE", "")
			}
			got := ResolveFilePath("/repo", tt.flag)
			want := filepath.Join("/repo", tt.wantSfx)
			if got != want {
				t.Errorf("ResolveFilePath = %q, want %q", got, want)
			}
		})
	}
}

// --- CRUD Operations Tests ---

func TestAdd_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")

	i, err := Add(path, "Build search feature", "", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(i.ID) != 4 {
		t.Errorf("ID length = %d, want 4", len(i.ID))
	}
	if i.Date == "" {
		t.Error("Date should not be empty")
	}
	if i.Text != "Build search feature" {
		t.Errorf("Text = %q, want 'Build search feature'", i.Text)
	}
	if i.Done {
		t.Error("new idea should not be done")
	}

	// Verify file content
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Build search feature") {
		t.Error("file should contain the idea text")
	}
}

func TestAdd_CustomIDAndDate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")

	i, err := Add(path, "My idea", "ab12", "2025-01-01")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if i.ID != "ab12" {
		t.Errorf("ID = %q, want ab12", i.ID)
	}
	if i.Date != "2025-01-01" {
		t.Errorf("Date = %q, want 2025-01-01", i.Date)
	}
}

func TestAdd_IDCollision(t *testing.T) {
	dir := t.TempDir()
	path := writeBacklog(t, dir, "- [ ] [ab12] 2025-06-15: Existing idea\n")

	_, err := Add(path, "New idea", "ab12", "")
	if err == nil {
		t.Fatal("expected error for ID collision")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want 'already exists'", err.Error())
	}
}

func TestAdd_AutoCreateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "backlog.md")

	_, err := Add(path, "New idea in new file", "", "2025-06-15")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should have been auto-created")
	}
}

func TestAdd_EmptyText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")

	_, err := Add(path, "", "", "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestList_OpenOnly(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
- [ ] [c3d4] 2025-06-10: Open two
- [x] [e5f6] 2025-06-08: Done one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterOpen, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 2 {
		t.Fatalf("count = %d, want 2", len(ideas))
	}
	for _, i := range ideas {
		if i.Done {
			t.Error("open filter returned a done idea")
		}
	}
}

func TestList_DoneOnly(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
- [x] [e5f6] 2025-06-08: Done one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterDone, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 1 {
		t.Fatalf("count = %d, want 1", len(ideas))
	}
	if !ideas[0].Done {
		t.Error("done filter returned an open idea")
	}
}

func TestList_All(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
- [x] [e5f6] 2025-06-08: Done one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 2 {
		t.Fatalf("count = %d, want 2", len(ideas))
	}
}

func TestList_SortByID(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [c3d4] 2025-06-10: Second
- [ ] [a1b2] 2025-06-15: First
- [ ] [e5f6] 2025-06-08: Third
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "id", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ideas[0].ID != "a1b2" || ideas[1].ID != "c3d4" || ideas[2].ID != "e5f6" {
		t.Errorf("sort by id: got %s, %s, %s", ideas[0].ID, ideas[1].ID, ideas[2].ID)
	}
}

func TestList_SortByDate(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [c3d4] 2025-06-15: Third date
- [ ] [a1b2] 2025-06-08: First date
- [ ] [e5f6] 2025-06-10: Second date
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ideas[0].Date != "2025-06-08" || ideas[1].Date != "2025-06-10" || ideas[2].Date != "2025-06-15" {
		t.Errorf("sort by date: got %s, %s, %s", ideas[0].Date, ideas[1].Date, ideas[2].Date)
	}
}

func TestList_Reverse(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a1b2] 2025-06-08: First
- [ ] [c3d4] 2025-06-10: Second
- [ ] [e5f6] 2025-06-15: Third
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "id", true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ideas[0].ID != "e5f6" || ideas[1].ID != "c3d4" || ideas[2].ID != "a1b2" {
		t.Errorf("reverse sort by id: got %s, %s, %s", ideas[0].ID, ideas[1].ID, ideas[2].ID)
	}
}

func TestList_JSON(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	data, err := json.Marshal(ideas)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("JSON array len = %d, want 1", len(parsed))
	}
	obj := parsed[0]
	if obj["id"] != "a7k2" {
		t.Errorf("JSON id = %v, want a7k2", obj["id"])
	}
	if obj["status"] != "open" {
		t.Errorf("JSON status = %v, want open", obj["status"])
	}
	if obj["date"] != "2025-06-15" {
		t.Errorf("JSON date = %v, want 2025-06-15", obj["date"])
	}
	if obj["text"] != "Add dark mode" {
		t.Errorf("JSON text = %v, want 'Add dark mode'", obj["text"])
	}
}

func TestShow_SingleMatch(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Fix redirect
`
	path := writeBacklog(t, dir, content)

	i, err := Show(path, "a7k2")
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if i.ID != "a7k2" {
		t.Errorf("ID = %q, want a7k2", i.ID)
	}
}

func TestShow_NoMatch(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, err := Show(path, "nonexistent")
	if err == nil {
		t.Fatal("expected error for no match")
	}
	if !strings.Contains(err.Error(), "No idea matching 'nonexistent'") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestShow_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Add light mode
`
	path := writeBacklog(t, dir, content)

	_, err := Show(path, "mode")
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "Multiple matches") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestDone_MarkOpen(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, _, err := Done(path, "a7k2")
	if err != nil {
		t.Fatalf("Done: %v", err)
	}
	if !i.Done {
		t.Error("idea should be done after Done()")
	}

	// Verify file was updated
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [x] [a7k2]") {
		t.Error("file should contain done marker")
	}
}

func TestDone_AlreadyDone(t *testing.T) {
	dir := t.TempDir()
	content := `- [x] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, _, err := Done(path, "a7k2")
	if err == nil {
		t.Fatal("expected error when marking already-done idea as done")
	}
	if !strings.Contains(err.Error(), "No idea matching") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestReopen_MarkDone(t *testing.T) {
	dir := t.TempDir()
	content := `- [x] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, _, err := Reopen(path, "a7k2")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if i.Done {
		t.Error("idea should be open after Reopen()")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [ ] [a7k2]") {
		t.Error("file should contain open marker")
	}
}

func TestReopen_AlreadyOpen(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, _, err := Reopen(path, "a7k2")
	if err == nil {
		t.Fatal("expected error when reopening already-open idea")
	}
	if !strings.Contains(err.Error(), "No idea matching") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestEdit_TextOnly(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, _, err := Edit(path, "a7k2", "Add dark mode with toggle", "", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if i.Text != "Add dark mode with toggle" {
		t.Errorf("Text = %q", i.Text)
	}
	if i.ID != "a7k2" {
		t.Error("ID should be preserved")
	}
	if i.Date != "2025-06-15" {
		t.Error("Date should be preserved")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Add dark mode with toggle") {
		t.Error("file should contain updated text")
	}
}

func TestEdit_WithID_NoCollision(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, _, err := Edit(path, "a7k2", "Same text", "z9y8", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if i.ID != "z9y8" {
		t.Errorf("ID = %q, want z9y8", i.ID)
	}
}

func TestEdit_WithID_Collision(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [z9y8] 2025-06-10: Other idea
`
	path := writeBacklog(t, dir, content)

	_, _, err := Edit(path, "a7k2", "Text", "z9y8", "")
	if err == nil {
		t.Fatal("expected error for ID collision")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestEdit_WithDate(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, _, err := Edit(path, "a7k2", "Same text", "", "2025-12-01")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if i.Date != "2025-12-01" {
		t.Errorf("Date = %q, want 2025-12-01", i.Date)
	}
}

func TestRm_WithForce(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Fix redirect
`
	path := writeBacklog(t, dir, content)

	removed, _, err := Rm(path, "a7k2", true)
	if err != nil {
		t.Fatalf("Rm: %v", err)
	}
	if removed.ID != "a7k2" {
		t.Errorf("removed ID = %q, want a7k2", removed.ID)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "a7k2") {
		t.Error("file should not contain removed idea")
	}
	if !strings.Contains(string(data), "c3d4") {
		t.Error("file should still contain other idea")
	}
}

func TestRm_WithoutForce(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, _, err := Rm(path, "a7k2", false)
	if err == nil {
		t.Fatal("expected error without --force")
	}
	if !strings.Contains(err.Error(), "Use --force") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestRm_PreservesNonIdeaLines(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Fix redirect

Footer
`
	path := writeBacklog(t, dir, content)

	_, _, err := Rm(path, "a7k2", true)
	if err != nil {
		t.Fatalf("Rm: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "# Backlog") {
		t.Error("header should be preserved")
	}
	if !strings.Contains(result, "Footer") {
		t.Error("footer should be preserved")
	}
	if !strings.Contains(result, "c3d4") {
		t.Error("other idea should be preserved")
	}
}

// --- JSON Output Tests ---

func TestIdeaJSON(t *testing.T) {
	i := Idea{ID: "a7k2", Date: "2025-06-15", Text: "Add dark mode", Done: false}
	data, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if obj["id"] != "a7k2" {
		t.Errorf("id = %v", obj["id"])
	}
	if obj["date"] != "2025-06-15" {
		t.Errorf("date = %v", obj["date"])
	}
	if obj["status"] != "open" {
		t.Errorf("status = %v", obj["status"])
	}
	if obj["text"] != "Add dark mode" {
		t.Errorf("text = %v", obj["text"])
	}
}

func TestIdeaJSON_Done(t *testing.T) {
	i := Idea{ID: "e5f6", Date: "2025-06-08", Text: "Fix bug", Done: true}
	data, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var obj map[string]interface{}
	json.Unmarshal(data, &obj)
	if obj["status"] != "done" {
		t.Errorf("status = %v, want done", obj["status"])
	}
}

// --- Repo Root Tests ---

func TestMainRepoRoot_ReturnsPath(t *testing.T) {
	root, err := MainRepoRoot()
	if err != nil {
		t.Fatalf("MainRepoRoot: %v", err)
	}
	if root == "" {
		t.Error("MainRepoRoot returned empty string")
	}
}

func TestWorktreeRoot_ReturnsPath(t *testing.T) {
	root, err := WorktreeRoot()
	if err != nil {
		t.Fatalf("WorktreeRoot: %v", err)
	}
	if root == "" {
		t.Error("WorktreeRoot returned empty string")
	}
}

// --- Edge Case Tests ---

func TestAdd_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Existing idea
`
	path := writeBacklog(t, dir, content)

	_, err := Add(path, "New idea", "b1c2", "2025-07-01")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "# Backlog") {
		t.Error("header should be preserved")
	}
	if !strings.Contains(result, "a7k2") {
		t.Error("existing idea should be preserved")
	}
	if !strings.Contains(result, "b1c2") {
		t.Error("new idea should be present")
	}
}

func TestDone_PreservesOtherIdeas(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: First
- [ ] [c3d4] 2025-06-10: Second
`
	path := writeBacklog(t, dir, content)

	_, _, err := Done(path, "a7k2")
	if err != nil {
		t.Fatalf("Done: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "- [x] [a7k2]") {
		t.Error("first idea should be marked done")
	}
	if !strings.Contains(result, "- [ ] [c3d4]") {
		t.Error("second idea should remain open")
	}
}

func TestEdit_PreservesStatus(t *testing.T) {
	dir := t.TempDir()
	content := `- [x] [a7k2] 2025-06-15: Done idea
`
	path := writeBacklog(t, dir, content)

	i, _, err := Edit(path, "a7k2", "Updated done idea", "", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if !i.Done {
		t.Error("status should be preserved as done")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [x] [a7k2]") {
		t.Error("done marker should be preserved in file")
	}
}

func TestAdd_FileWithoutTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")
	// Existing file has NO trailing newline.
	if err := os.WriteFile(path, []byte("- [ ] [a7k2] 2025-06-15: Existing"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Add(path, "New idea", "b1c2", "2025-07-01"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "- [ ] [a7k2] 2025-06-15: Existing\n- [ ] [b1c2]") {
		t.Errorf("new entry should start on a fresh line, got:\n%s", got)
	}
}

func TestLoadFile_PreservesTrailingBlankLines(t *testing.T) {
	dir := t.TempDir()
	// Two trailing newlines means there's a blank line at EOF.
	content := "- [ ] [a7k2] 2025-06-15: Add dark mode\n\n"
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if _, err := SaveFile(f, path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != content {
		t.Errorf("round-trip should preserve trailing blank lines\ngot:  %q\nwant: %q", string(data), content)
	}
}

func TestEdit_EmptyTextRejected(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, _, err := Edit(path, "a7k2", "", "", "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	if !strings.Contains(err.Error(), "text is required") {
		t.Errorf("error = %q, want 'text is required'", err.Error())
	}
}

func TestList_InvalidSortField(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, err := List(path, FilterAll, "data", false)
	if err == nil {
		t.Fatal("expected error for invalid sort field")
	}
	if !strings.Contains(err.Error(), "invalid sort field") {
		t.Errorf("error = %q, want 'invalid sort field'", err.Error())
	}
}

func TestList_EmptyResult(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterDone, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 0 {
		t.Errorf("count = %d, want 0", len(ideas))
	}
}

// --- Lenient Parser Tests (resilient backlog parser) ---

// TestParseLine_Lenient covers the accepted input variants: optional date,
// variant bullets (- * +), leading whitespace, and the canonical regression.
func TestParseLine_Lenient(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantID   string
		wantDate string
		wantText string
		wantDone bool
	}{
		{"canonical open (regression)", "- [ ] [a7k2] 2025-06-15: Add dark mode", "a7k2", "2025-06-15", "Add dark mode", false},
		{"canonical done (regression)", "- [x] [e5f6] 2025-06-08: Fix bug", "e5f6", "2025-06-08", "Fix bug", true},
		{"dateless open", "- [ ] [rk7t] Tune the reporter", "rk7t", "", "Tune the reporter", false},
		{"dateless done", "- [x] [rk7t] Tune the reporter", "rk7t", "", "Tune the reporter", true},
		{"star bullet", "* [ ] [a7k2] do a thing", "a7k2", "", "do a thing", false},
		{"plus bullet", "+ [x] [a7k2] do a thing", "a7k2", "", "do a thing", true},
		{"star bullet dated", "* [ ] [a7k2] 2025-06-15: dated star", "a7k2", "2025-06-15", "dated star", false},
		{"leading spaces", "  - [ ] [a7k2] indented idea", "a7k2", "", "indented idea", false},
		{"leading tab", "\t- [ ] [a7k2] tab indented", "a7k2", "", "tab indented", false},
		{"leading spaces dated", "    - [x] [a7k2] 2025-06-15: deep indent dated", "a7k2", "2025-06-15", "deep indent dated", true},
		{"malformed date is text", "- [ ] [a7k2] 2025-6-15: Text", "a7k2", "", "2025-6-15: Text", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idea, ok := ParseLine(tt.line)
			if !ok {
				t.Fatalf("ParseLine(%q) = ok=false, want true", tt.line)
			}
			if idea.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", idea.ID, tt.wantID)
			}
			if idea.Date != tt.wantDate {
				t.Errorf("Date = %q, want %q", idea.Date, tt.wantDate)
			}
			if idea.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", idea.Text, tt.wantText)
			}
			if idea.Done != tt.wantDone {
				t.Errorf("Done = %v, want %v", idea.Done, tt.wantDone)
			}
		})
	}
}

// TestParseLine_DatelessAndDatedMatchModuloDate pins R1: a dateless line and its
// dated counterpart parse to the same Idea except for Date.
func TestParseLine_DatelessAndDatedMatchModuloDate(t *testing.T) {
	dated, ok1 := ParseLine("- [ ] [rk7t] 2026-06-10: Tune the README-extraction reporter")
	dateless, ok2 := ParseLine("- [ ] [rk7t] Tune the README-extraction reporter")
	if !ok1 || !ok2 {
		t.Fatalf("both lines should parse: dated=%v dateless=%v", ok1, ok2)
	}
	if dated.ID != dateless.ID || dated.Text != dateless.Text || dated.Done != dateless.Done {
		t.Errorf("ideas differ beyond Date: dated=%+v dateless=%+v", dated, dateless)
	}
	if dated.Date != "2026-06-10" {
		t.Errorf("dated.Date = %q, want 2026-06-10", dated.Date)
	}
	if dateless.Date != "" {
		t.Errorf("dateless.Date = %q, want empty", dateless.Date)
	}
}

// TestParseLine_PrecisionGuard pins R6 + R7: Shape B lines and genuine non-idea
// prose are NOT parsed as ideas under the relaxed regex.
func TestParseLine_PrecisionGuard(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		// Shape B (second bracket) — must stay inert pass-through (R6).
		{"shape B dated", "- [ ] [ni3o] [DEV-1011] 2026-02-12: Capture more metrics"},
		{"shape B dateless", "- [ ] [ni3o] [DEV-1011] Capture more metrics"},
		{"shape B star bullet", "* [x] [ni3o] [DEV-1011] 2026-02-12: done one"},
		// Genuine non-idea prose — missing the checkbox/id anchors (R7).
		{"header", "# Backlog"},
		{"prose", "Some footer text"},
		{"plain bullet", "- a plain bullet"},
		{"checkbox no id", "- [ ] no id here"},
		{"id too long", "- [ ] [toolong5] text"},
		{"id too short", "- [ ] [abc] text"},
		{"uppercase id", "- [ ] [ABCD] text"},
		{"blank", "   "},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := ParseLine(tt.line); ok {
				t.Errorf("ParseLine(%q) = ok=true, want false", tt.line)
			}
		})
	}
}

// TestParseLine_DatedBracketPrefixedTextParses guards R6's precision: the Shape B
// guard must apply only to DATELESS matches. A genuine canonical line whose
// description happens to begin with a bracket (date present) must still parse,
// not be dropped as pass-through.
func TestParseLine_DatedBracketPrefixedTextParses(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantDate string
		wantText string
	}{
		{"dated bracket text", "- [ ] [a7k2] 2025-06-15: [TODO] Add dark mode", "2025-06-15", "[TODO] Add dark mode"},
		{"dated bracket text done", "- [x] [e5f6] 2025-06-08: [WIP] refactor parser", "2025-06-08", "[WIP] refactor parser"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idea, ok := ParseLine(tt.line)
			if !ok {
				t.Fatalf("ParseLine(%q) = ok=false, want true (dated bracket-prefixed text must parse)", tt.line)
			}
			if idea.Date != tt.wantDate {
				t.Errorf("Date = %q, want %q", idea.Date, tt.wantDate)
			}
			if idea.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", idea.Text, tt.wantText)
			}
		})
	}
}

// TestLoadFile_CRLF pins R4: CRLF lines parse, \r never leaks into Text, and
// output is LF-only after a save.
func TestLoadFile_CRLF(t *testing.T) {
	dir := t.TempDir()
	// CRLF endings throughout.
	content := "# Backlog\r\n\r\n- [ ] [a7k2] 2025-06-15: Add dark mode\r\n- [x] [e5f6] 2025-06-08: Fix bug\r\n"
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(f.ideas) != 2 {
		t.Fatalf("ideas count = %d, want 2", len(f.ideas))
	}
	if f.ideas[0].Text != "Add dark mode" {
		t.Errorf("first idea Text = %q, want 'Add dark mode' (no trailing \\r)", f.ideas[0].Text)
	}
	if strings.Contains(f.ideas[0].Text, "\r") || strings.Contains(f.ideas[1].Text, "\r") {
		t.Error("carriage return leaked into idea Text")
	}

	// Save and assert output is LF-only.
	if _, err := SaveFile(f, path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "\r") {
		t.Errorf("output still contains CRLF; want LF-only:\n%q", string(data))
	}
}

// TestSaveFile_BackfillsDatelessDate pins R9 + R10's count: a dateless idea gets
// today's date on save and SaveFile reports the backfill count.
func TestSaveFile_BackfillsDatelessDate(t *testing.T) {
	dir := t.TempDir()
	content := "- [ ] [rk7t] Tune the reporter\n- [ ] [a7k2] 2025-06-15: Already dated\n- [ ] [c3d4] Another dateless\n"
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	// Capture today before the code under test stamps the backfill date, so a
	// midnight rollover between save and assertion cannot flake the test.
	today := time.Now().Format("2006-01-02")
	backfilled, err := SaveFile(f, path)
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	if backfilled != 2 {
		t.Errorf("backfilled = %d, want 2", backfilled)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "- [ ] [rk7t] "+today+": Tune the reporter") {
		t.Errorf("rk7t should be stamped with today (%s):\n%s", today, got)
	}
	if !strings.Contains(got, "- [ ] [c3d4] "+today+": Another dateless") {
		t.Errorf("c3d4 should be stamped with today (%s):\n%s", today, got)
	}
	// Already-dated idea keeps its date.
	if !strings.Contains(got, "- [ ] [a7k2] 2025-06-15: Already dated") {
		t.Errorf("a7k2 date should be unchanged:\n%s", got)
	}
}

// TestSaveFile_NoBackfillReturnsZero pins R10's suppression branch: when no
// dateless ideas exist, SaveFile reports a zero count.
func TestSaveFile_NoBackfillReturnsZero(t *testing.T) {
	dir := t.TempDir()
	content := "- [ ] [a7k2] 2025-06-15: Already dated\n"
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	backfilled, err := SaveFile(f, path)
	if err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	if backfilled != 0 {
		t.Errorf("backfilled = %d, want 0", backfilled)
	}
}

// TestDone_BackfillsAndCanonicalizes pins the worked example (R8 + R9 + R10):
// a dateless line is stamped with today and canonicalized on `done`.
func TestDone_BackfillsAndCanonicalizes(t *testing.T) {
	dir := t.TempDir()
	content := "- [ ] [rk7t] Tune the README-extraction reporter\n"
	path := writeBacklog(t, dir, content)

	// Capture today before Done backfills/saves, so a midnight rollover between
	// the save and the assertions cannot flake the test.
	today := time.Now().Format("2006-01-02")
	i, backfilled, err := Done(path, "rk7t")
	if err != nil {
		t.Fatalf("Done: %v", err)
	}
	if backfilled != 1 {
		t.Errorf("backfilled = %d, want 1", backfilled)
	}
	if i.Date != today {
		t.Errorf("returned idea Date = %q, want today %q", i.Date, today)
	}
	data, _ := os.ReadFile(path)
	want := "- [x] [rk7t] " + today + ": Tune the README-extraction reporter\n"
	if string(data) != want {
		t.Errorf("file after done:\ngot:  %q\nwant: %q", string(data), want)
	}
}

// TestEdit_BackfillsDatelessDate pins R9 on the edit path.
func TestEdit_BackfillsDatelessDate(t *testing.T) {
	dir := t.TempDir()
	content := "- [ ] [rk7t] original text\n"
	path := writeBacklog(t, dir, content)

	// Capture today before Edit backfills/saves, so a midnight rollover between
	// the save and the assertion cannot flake the test.
	today := time.Now().Format("2006-01-02")
	i, backfilled, err := Edit(path, "rk7t", "new text", "", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if backfilled != 1 {
		t.Errorf("backfilled = %d, want 1", backfilled)
	}
	if i.Date != today {
		t.Errorf("Date = %q, want today %q", i.Date, today)
	}
}

// TestSaveFile_CanonicalizesVariants pins R8: variant bullets, indentation, and
// CRLF among recognized idea lines are all canonicalized on the first mutating
// save, while non-idea lines are preserved.
func TestSaveFile_CanonicalizesVariants(t *testing.T) {
	dir := t.TempDir()
	// Mix of * and + bullets, indentation; CRLF on the star line.
	content := "# Backlog\n\n* [ ] [a7k2] 2025-06-15: star bullet\r\n  + [x] [e5f6] 2025-06-08: indented plus\nSome prose\n"
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(f.ideas) != 2 {
		t.Fatalf("ideas count = %d, want 2", len(f.ideas))
	}
	if _, err := SaveFile(f, path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	// Recognized idea lines canonicalized to "- " bullet, no indent, LF.
	if !strings.Contains(got, "- [ ] [a7k2] 2025-06-15: star bullet\n") {
		t.Errorf("star bullet not canonicalized:\n%q", got)
	}
	if !strings.Contains(got, "- [x] [e5f6] 2025-06-08: indented plus\n") {
		t.Errorf("indented plus bullet not canonicalized:\n%q", got)
	}
	if strings.Contains(got, "\r") {
		t.Errorf("output should be LF-only:\n%q", got)
	}
	if strings.Contains(got, "* [") || strings.Contains(got, "+ [") || strings.Contains(got, "  - [") {
		t.Errorf("variant bullets/indentation survived save:\n%q", got)
	}
	// Non-idea lines preserved.
	if !strings.Contains(got, "# Backlog") || !strings.Contains(got, "Some prose") {
		t.Errorf("non-idea lines not preserved:\n%q", got)
	}
}

// TestSaveFile_ShapeBPreservedVerbatim pins R6: a Shape B second-bracket line is
// neither parsed nor rewritten on a mutating save of a different idea.
func TestSaveFile_ShapeBPreservedVerbatim(t *testing.T) {
	dir := t.TempDir()
	shapeB := "- [ ] [ni3o] [DEV-1011] 2026-02-12: Capture more metrics"
	content := shapeB + "\n- [ ] [a7k2] 2025-06-15: Real idea\n"
	path := writeBacklog(t, dir, content)

	// Mutate a different (real) idea, forcing a save.
	_, _, err := Done(path, "a7k2")
	if err != nil {
		t.Fatalf("Done: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, shapeB) {
		t.Errorf("Shape B line not preserved verbatim:\nwant line: %q\ngot file:\n%q", shapeB, got)
	}
	if !strings.Contains(got, "- [x] [a7k2] 2025-06-15: Real idea") {
		t.Errorf("real idea should be marked done:\n%q", got)
	}
}

// TestSaveFile_PreservesNonIdeaLinesByteForByte pins R11: headers, prose, and
// blank lines survive a mutating save byte-for-byte (only the mutated idea line
// changes; the other idea lines are already canonical so the rest is identical).
func TestSaveFile_PreservesNonIdeaLinesByteForByte(t *testing.T) {
	dir := t.TempDir()
	content := "# Backlog\n\nIntro prose paragraph.\n\n- [ ] [a7k2] 2025-06-15: First\n- [ ] [c3d4] 2025-06-10: Second\n\nFooter line\n"
	path := writeBacklog(t, dir, content)

	_, _, err := Done(path, "a7k2")
	if err != nil {
		t.Fatalf("Done: %v", err)
	}

	want := "# Backlog\n\nIntro prose paragraph.\n\n- [x] [a7k2] 2025-06-15: First\n- [ ] [c3d4] 2025-06-10: Second\n\nFooter line\n"
	data, _ := os.ReadFile(path)
	if string(data) != want {
		t.Errorf("non-idea content not preserved byte-for-byte\ngot:  %q\nwant: %q", string(data), want)
	}
}

// --- Escape/Unescape Tests (escape-on-write, unescape-on-display) ---

func TestEscapeText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text untouched", "Add dark mode", "Add dark mode"},
		{"LF becomes literal backslash-n", "first\nsecond", `first\nsecond`},
		{
			"multi-paragraph with idea-looking line",
			"first line\n\nsecond paragraph\n- [ ] looks like a task",
			`first line\n\nsecond paragraph\n- [ ] looks like a task`,
		},
		{"backslash doubles", `C:\new`, `C:\\new`},
		{"doubled backslash quadruples", `a\\b`, `a\\\\b`},
		{"trailing lone backslash doubles", `trailing\`, `trailing\\`},
		{"literal backslash-n input", `a\nb`, `a\\nb`},
		{"CRLF normalizes then escapes", "a\r\nb", `a\nb`},
		{"lone CR normalizes then escapes", "a\rb", `a\nb`},
		{"mixed CRLF and CR", "a\r\nb\rc", `a\nb\nc`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EscapeText(tt.in)
			if got != tt.want {
				t.Errorf("EscapeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if strings.ContainsAny(got, "\n\r") {
				t.Errorf("EscapeText(%q) = %q contains raw LF/CR", tt.in, got)
			}
		})
	}
}

func TestUnescapeText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text untouched", "Add dark mode", "Add dark mode"},
		{"literal backslash-n becomes LF", `first\nsecond`, "first\nsecond"},
		{"doubled backslash halves", `C:\\new`, `C:\new`},
		{"escaped backslash before n stays literal", `a\\nb`, `a\nb`},
		{"unrecognized escape passes through verbatim", `a\b`, `a\b`},
		{"trailing lone backslash passes through verbatim", `trailing\`, `trailing\`},
		{"legacy literal backslash-n reinterprets as newline", `C:\new`, "C:\new"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnescapeText(tt.in)
			if got != tt.want {
				t.Errorf("UnescapeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestEscapeUnescapeRoundTrip pins the round-trip law:
// UnescapeText(EscapeText(x)) == x for any CR-free x; for x containing CR it
// equals the CR-normalized form (CR->LF is the only deliberate loss).
func TestEscapeUnescapeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string // "" means: expect the input back unchanged
	}{
		{"plain text", "Add dark mode to settings page", ""},
		{"multi-paragraph", "first line\n\nsecond paragraph\nthird", ""},
		{"windows path", `C:\new`, ""},
		{"doubled backslash", `a\\b`, ""},
		{"trailing lone backslash", `trailing\`, ""},
		{"literal backslash-n text", `a\nb`, ""},
		{"backslash-heavy mix", `\\n vs \n and \`, ""},
		{"idea-looking line", "- [ ] looks like a task", ""},
		{"newline-only text", "\n", ""},
		{"CRLF normalizes", "a\r\nb", "a\nb"},
		{"lone CR normalizes", "a\rb", "a\nb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want
			if want == "" {
				want = tt.in
			}
			got := UnescapeText(EscapeText(tt.in))
			if got != want {
				t.Errorf("UnescapeText(EscapeText(%q)) = %q, want %q", tt.in, got, want)
			}
		})
	}
}

func TestFormatLine_EscapesMultilineText(t *testing.T) {
	i := Idea{ID: "a7k2", Date: "2026-06-10", Text: "first line\n\nsecond paragraph\n- [ ] looks like a task"}
	got := FormatLine(i)
	want := `- [ ] [a7k2] 2026-06-10: first line\n\nsecond paragraph\n- [ ] looks like a task`
	if got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
	if strings.Contains(got, "\n") {
		t.Errorf("FormatLine output contains a raw newline: %q", got)
	}
}

func TestParseLine_UnescapesText(t *testing.T) {
	line := `- [ ] [a7k2] 2026-06-10: first line\n\nsecond paragraph`
	i, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected valid parse")
	}
	want := "first line\n\nsecond paragraph"
	if i.Text != want {
		t.Errorf("Text = %q, want %q", i.Text, want)
	}
}

// TestParseFormatRoundTrip_Escaped pins that an escaped on-disk line survives
// parse -> format byte-identical (no churn on already-canonical lines).
func TestParseFormatRoundTrip_Escaped(t *testing.T) {
	lines := []string{
		`- [ ] [a7k2] 2026-06-10: first\n\nsecond paragraph`,
		`- [x] [e5f6] 2026-06-10: path C:\\new here`,
		`- [ ] [c3d4] 2026-06-10: plain single-line text`,
	}
	for _, line := range lines {
		i, ok := ParseLine(line)
		if !ok {
			t.Fatalf("failed to parse %q", line)
		}
		if got := FormatLine(i); got != line {
			t.Errorf("round-trip changed line:\ngot:  %q\nwant: %q", got, line)
		}
	}
}

func TestDisplayLine_RendersRealNewlines(t *testing.T) {
	i := Idea{ID: "a7k2", Date: "2026-06-10", Text: "first line\n\nsecond paragraph", Done: false}
	got := DisplayLine(i)
	want := "- [ ] [a7k2] 2026-06-10: first line\n\nsecond paragraph"
	if got != want {
		t.Errorf("DisplayLine = %q, want %q", got, want)
	}
}

// TestAdd_MultilineText pins the core fix: a multiline add lands as exactly
// one physical line, parses as exactly one idea (no phantom from the embedded
// idea-looking line), and round-trips the full text.
func TestAdd_MultilineText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")
	text := "first line\n\nsecond paragraph\n- [ ] looks like a task"

	i, err := Add(path, text, "", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if i.Text != text {
		t.Errorf("in-memory Text = %q, want %q", i.Text, text)
	}

	data, _ := os.ReadFile(path)
	content := strings.TrimSuffix(string(data), "\n")
	if lines := strings.Split(content, "\n"); len(lines) != 1 {
		t.Fatalf("file has %d physical lines, want 1:\n%q", len(lines), string(data))
	}

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(f.ideas) != 1 {
		t.Fatalf("parsed %d ideas, want 1 (phantom idea?)", len(f.ideas))
	}
	if f.ideas[0].Text != text {
		t.Errorf("reloaded Text = %q, want %q", f.ideas[0].Text, text)
	}
}

func TestAdd_NormalizesCR(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")

	i, err := Add(path, "a\r\nb\rc", "", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if i.Text != "a\nb\nc" {
		t.Errorf("Text = %q, want %q", i.Text, "a\nb\nc")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `a\nb\nc`) {
		t.Errorf("file should contain escaped normalized text, got %q", string(data))
	}
	if strings.Contains(string(data), "\r") {
		t.Errorf("raw CR reached the file: %q", string(data))
	}
}

func TestEdit_MultilineText(t *testing.T) {
	dir := t.TempDir()
	content := "# Backlog\n\n- [ ] [a7k2] 2025-06-15: old text\n\nFooter\n"
	path := writeBacklog(t, dir, content)
	text := "new first\n\nnew second paragraph"

	i, _, err := Edit(path, "a7k2", text, "", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if i.Text != text {
		t.Errorf("in-memory Text = %q, want %q", i.Text, text)
	}

	want := "# Backlog\n\n" + `- [ ] [a7k2] 2025-06-15: new first\n\nnew second paragraph` + "\n\nFooter\n"
	data, _ := os.ReadFile(path)
	if string(data) != want {
		t.Errorf("file after multiline edit:\ngot:  %q\nwant: %q", string(data), want)
	}
}

// TestRm_MultilineIdea_NoOrphans pins the orphaning fix: removing a multiline
// idea removes everything — no continuation residue is left in the file.
func TestRm_MultilineIdea_NoOrphans(t *testing.T) {
	dir := t.TempDir()
	original := "# Backlog\n"
	path := writeBacklog(t, dir, original)

	if _, err := Add(path, "first line\n\nsecond paragraph\n- [ ] looks like a task", "ab12", ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, _, err := Rm(path, "ab12", true); err != nil {
		t.Fatalf("Rm: %v", err)
	}

	data, _ := os.ReadFile(path)
	got := string(data)
	if got != original {
		t.Errorf("file after rm = %q, want %q (no orphaned residue)", got, original)
	}
}

// TestLegacyBackslash_NormalizeOnWrite pins the legacy policy: lone-backslash
// text reads verbatim, re-serializes doubled on the first mutating save, and
// the second save is byte-stable (no further churn).
func TestLegacyBackslash_NormalizeOnWrite(t *testing.T) {
	dir := t.TempDir()
	path := writeBacklog(t, dir, `- [ ] [a7k2] 2025-06-15: path a\b here`+"\n")

	// Read: unrecognized escape passes through verbatim.
	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if want := `path a\b here`; f.ideas[0].Text != want {
		t.Errorf("legacy Text = %q, want %q", f.ideas[0].Text, want)
	}

	// First mutating save: on-disk encoding canonicalizes to doubled backslash.
	if _, _, err := Done(path, "a7k2"); err != nil {
		t.Fatalf("Done: %v", err)
	}
	data, _ := os.ReadFile(path)
	first := string(data)
	if want := `- [x] [a7k2] 2025-06-15: path a\\b here` + "\n"; first != want {
		t.Errorf("after first save:\ngot:  %q\nwant: %q", first, want)
	}

	// Content unchanged: still reads back as the same real text.
	f, err = LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if want := `path a\b here`; f.ideas[0].Text != want {
		t.Errorf("reloaded Text = %q, want %q", f.ideas[0].Text, want)
	}

	// Second save: byte-stable, no further churn.
	if _, _, err := Reopen(path, "a7k2"); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if _, _, err := Done(path, "a7k2"); err != nil {
		t.Fatalf("Done: %v", err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != first {
		t.Errorf("second save churned the file:\ngot:  %q\nwant: %q", string(data), first)
	}
}

// TestIdeaJSON_MultilineText pins that the JSON text field carries real
// newlines (JSON itself encodes them), since Idea.Text holds real text.
func TestIdeaJSON_MultilineText(t *testing.T) {
	i := Idea{ID: "a7k2", Date: "2026-06-10", Text: "first\nsecond", Done: false}
	data, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Text != "first\nsecond" {
		t.Errorf("decoded text = %q, want %q", decoded.Text, "first\nsecond")
	}
}
