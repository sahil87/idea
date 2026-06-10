package idea

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Idea represents a single backlog item.
type Idea struct {
	ID   string `json:"id"`
	Date string `json:"date"`
	Text string `json:"text"`
	Done bool   `json:"-"`
}

// Status returns "open" or "done".
func (i Idea) Status() string {
	if i.Done {
		return "done"
	}
	return "open"
}

// StatusCheck returns "x" for done, " " for open.
func (i Idea) StatusCheck() string {
	if i.Done {
		return "x"
	}
	return " "
}

// MarshalJSON customizes JSON output to include status field.
func (i Idea) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID     string `json:"id"`
		Date   string `json:"date"`
		Status string `json:"status"`
		Text   string `json:"text"`
	}{
		ID:     i.ID,
		Date:   i.Date,
		Status: i.Status(),
		Text:   i.Text,
	})
}

// lineRegex matches idea lines leniently (lenient on read, canonical on write).
// Accepted input variants:
//   - bullet marker: -, *, or +
//   - arbitrary leading whitespace (spaces or tabs)
//   - the "YYYY-MM-DD: " date segment is OPTIONAL
//
// Examples that all parse:
//   - [ ] [a7k2] 2025-06-15: Add dark mode   (canonical)
//   - [ ] [a7k2] Add dark mode               (dateless)
//   - [x] [a7k2] indented + star bullet
//
// The [ ]/[x] checkbox plus the 4-char [id] are the anchors that keep
// false-positive matching of genuine prose low. The date group is non-capturing
// so the date itself remains optional without a second regex.
var lineRegex = regexp.MustCompile(`^\s*[-*+] \[([ x])\] \[([a-z0-9]{4})\] (?:(\d{4}-\d{2}-\d{2}): )?(.+)$`)

// shapeBPrefixRegex detects a legacy "Shape B" second bracket immediately
// following the id, e.g. the text portion of `- [ ] [id] [DEV-1011] date: text`.
// Such lines stay inert pass-through: the [issue_ids] slot is owned by external
// consumers (fab-kit's /fab-new), so idea must neither parse nor rewrite them.
var shapeBPrefixRegex = regexp.MustCompile(`^\[[^\]]*\]`)

var idChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// idRegex validates that an ID is exactly 4 lowercase alphanumeric characters.
var idRegex = regexp.MustCompile(`^[a-z0-9]{4}$`)

// dateRegex validates the YYYY-MM-DD date format.
var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateID checks that id matches the expected 4-char lowercase alphanumeric format.
func ValidateID(id string) error {
	if !idRegex.MatchString(id) {
		return fmt.Errorf("invalid ID %q: must be exactly 4 lowercase alphanumeric characters", id)
	}
	return nil
}

// ValidateDate checks that date matches YYYY-MM-DD format.
func ValidateDate(date string) error {
	if !dateRegex.MatchString(date) {
		return fmt.Errorf("invalid date %q: must be in YYYY-MM-DD format", date)
	}
	return nil
}

// textEscaper converts real idea text to its persisted single-line form.
// Exactly two escape sequences exist: backslash (U+005C) -> `\\` and
// LF (U+000A) -> `\n`. The single left-to-right pass cannot double-process:
// the literal backslashes it emits are never re-examined.
var textEscaper = strings.NewReplacer(`\`, `\\`, "\n", `\n`)

// textUnescaper is the exact inverse for text produced by textEscaper:
// `\\` -> backslash, `\n` -> LF. The two patterns differ in their second
// byte, so at most one matches at any position — the single pass is
// deterministic. Bytes matching neither pattern are copied through verbatim,
// which implements the lenient legacy policy: an unrecognized escape (e.g.
// `\b`) and a trailing lone `\` pass through unchanged.
var textUnescaper = strings.NewReplacer(`\\`, `\`, `\n`, "\n")

// normalizeCR canonicalizes line endings inside idea text: CRLF -> LF first,
// then any remaining lone CR -> LF. No raw CR ever reaches the backlog file;
// this CR->LF normalization is the only deliberate loss in the escape
// round-trip (see EscapeText).
func normalizeCR(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// EscapeText converts real idea text to the persisted single-line form:
// CR normalization (CRLF -> LF, lone CR -> LF) followed by escaping
// (`\` -> `\\`, LF -> `\n`). The result contains no raw LF or CR, so a
// persisted idea line is always exactly one physical line.
//
// Round-trip law: UnescapeText(EscapeText(x)) == x for any CR-free x;
// for x containing CR it equals the CR-normalized form of x.
func EscapeText(s string) string {
	return textEscaper.Replace(normalizeCR(s))
}

// UnescapeText converts persisted idea text back to its real form in a
// single left-to-right scan: `\\` -> `\`, `\n` -> LF. A backslash followed
// by any other character — and a trailing lone backslash — pass through
// verbatim, so legacy text written before the escape convention reads
// unchanged (it canonicalizes to doubled backslashes on the next mutating
// save via FormatLine, the normalize-on-write precedent).
func UnescapeText(s string) string {
	return textUnescaper.Replace(s)
}

// ParseLine parses a single backlog line into an Idea.
// Returns the parsed idea and true if the line is valid, or a zero Idea and false.
//
// ParseLine is pure: it never stamps a date. A dateless line parses with
// Date == "". Date backfill happens at the save seam (SaveFile), which keeps
// reads (list/show) and MarshalJSON faithful to the on-disk content.
func ParseLine(line string) (Idea, bool) {
	m := lineRegex.FindStringSubmatch(line)
	if m == nil {
		return Idea{}, false
	}
	// Precision guard: a legacy "Shape B" line (`- [ ] [id] [DEV-1011] date: text`)
	// matches the relaxed regex with the second bracket captured into the text
	// group. Reject it so it stays inert pass-through — the [issue_ids] slot is
	// owned by external consumers, and idea must not parse or rewrite these lines.
	//
	// The guard only applies to DATELESS matches (m[3] == ""): a Shape B line
	// cannot have a parsed date because the second bracket sits between the id and
	// the date segment, so the date never reaches m[3]. Restricting to m[3] == ""
	// means a genuine canonical line whose description happens to start with a
	// bracket (e.g. `- [ ] [a7k2] 2025-06-15: [TODO] do thing`) still parses —
	// its date is captured into m[3], so it is not mistaken for a Shape B slot.
	if m[3] == "" && shapeBPrefixRegex.MatchString(m[4]) {
		return Idea{}, false
	}
	return Idea{
		ID:   m[2],
		Date: m[3],               // "" when the optional date segment is absent
		Text: UnescapeText(m[4]), // in-memory Idea.Text always holds real text
		Done: m[1] == "x",
	}, true
}

// formatLineWith renders the canonical line shape around the given text
// representation. It is the single home of the format string, shared by
// FormatLine (escaped, persisted form) and DisplayLine (real-text form).
func formatLineWith(i Idea, text string) string {
	return fmt.Sprintf("- [%s] [%s] %s: %s", i.StatusCheck(), i.ID, i.Date, text)
}

// FormatLine serializes an Idea back to the markdown line format.
// The text is escaped (see EscapeText), so the output is always exactly one
// physical line — every write path (Add's append, SaveFile's rebuild,
// confirmations, RequireSingle's match listing) inherits the guarantee here.
func FormatLine(i Idea) string {
	return formatLineWith(i, EscapeText(i.Text))
}

// DisplayLine renders an Idea in the canonical line shape with its real
// (unescaped) text, so multiline ideas show their continuation lines below
// the prefix line. Used for human-facing display (idea show); machine-facing
// output keeps the escaped single-line FormatLine form.
func DisplayLine(i Idea) string {
	return formatLineWith(i, i.Text)
}

// MainRepoRoot returns the main worktree's root directory.
// It runs: git rev-parse --path-format=absolute --git-common-dir
// and returns the parent of that directory. This always resolves to the
// main worktree regardless of which worktree the command is run from.
func MainRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository")
	}
	gitDir := strings.TrimSpace(string(out))
	return filepath.Dir(gitDir), nil
}

// WorktreeRoot returns the current worktree's root directory.
// It runs: git rev-parse --show-toplevel
// In the main worktree this returns the same path as MainRepoRoot.
// In a linked worktree this returns the worktree's own root.
func WorktreeRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

// File represents a loaded backlog file, preserving non-idea lines.
type File struct {
	// lines stores every line in order. Non-idea lines are stored as-is.
	// Idea lines are stored as empty strings (their content comes from ideas).
	lines []string
	// ideaIndices maps from ideas slice index to lines slice index.
	ideaIndices []int
	// ideas holds the parsed ideas in file order.
	ideas []Idea
}

// LoadFile reads and parses a backlog file.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f := &File{}
	content := string(data)
	if content == "" {
		return f, nil
	}

	// Trim a single trailing newline (the canonical EOF newline) so that a
	// well-formed file does not produce a spurious empty last line. Multiple
	// trailing newlines are preserved verbatim per Constitution Principle I.
	if strings.HasSuffix(content, "\n") {
		content = content[:len(content)-1]
	}
	rawLines := strings.Split(content, "\n")

	for i, line := range rawLines {
		// Strip a trailing carriage return so CRLF files parse identically to
		// LF files. The \r is part of canonicalization — recognized idea lines
		// are regenerated LF-only on save, and a non-idea line that merely had a
		// CRLF ending is stored without its \r (output is always LF; non-idea
		// *content* is otherwise preserved verbatim per Constitution I).
		line = strings.TrimSuffix(line, "\r")
		if idea, ok := ParseLine(line); ok {
			f.lines = append(f.lines, "") // placeholder
			f.ideaIndices = append(f.ideaIndices, i)
			f.ideas = append(f.ideas, idea)
		} else {
			f.lines = append(f.lines, line)
		}
	}

	return f, nil
}

// SaveFile writes the backlog file, reconstructing from preserved lines and ideas.
// It returns the number of dateless ideas whose date was backfilled to today
// (callers surface this count as an advisory stderr notice; see cmd/idea).
//
// Canonical on write: every recognized idea line is regenerated via FormatLine
// (- bullet, no indentation, date present, LF endings), so the first mutating
// save normalizes all variant/dateless/CRLF idea lines at once. Non-idea lines
// pass through unchanged. Any idea with an empty Date is stamped with today's
// date (time.Now().Format("2006-01-02")) before serialization, keeping the
// in-memory Idea — and therefore MarshalJSON — consistent with what is written.
//
// The write is atomic: content is written to a temp file in the same directory
// and then renamed over the target path, so a crash mid-write cannot leave the
// backlog (the source of truth) partially written or empty.
func SaveFile(f *File, path string) (int, error) {
	backfilled := 0
	today := time.Now().Format("2006-01-02")
	for i := range f.ideas {
		if f.ideas[i].Date == "" {
			f.ideas[i].Date = today
			backfilled++
		}
	}

	// Rebuild lines
	result := make([]string, len(f.lines))
	copy(result, f.lines)

	for i, idx := range f.ideaIndices {
		result[idx] = FormatLine(f.ideas[i])
	}

	content := strings.Join(result, "\n") + "\n"
	if err := atomicWriteFile(path, []byte(content), 0644); err != nil {
		return 0, err
	}
	return backfilled, nil
}

// atomicWriteFile writes data to path atomically by writing to a temp file
// in the same directory and renaming it over the destination. The temp file
// is cleaned up on any error path before returning.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".idea-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// ResolveFilePath determines the backlog file path.
// Priority: flagValue > IDEAS_FILE env > default (fab/backlog.md).
// The result is relative to repoRoot.
func ResolveFilePath(repoRoot, flagValue string) string {
	if flagValue != "" {
		return filepath.Join(repoRoot, flagValue)
	}
	if env := os.Getenv("IDEAS_FILE"); env != "" {
		return filepath.Join(repoRoot, env)
	}
	return filepath.Join(repoRoot, "fab", "backlog.md")
}

// FilterKind specifies which ideas to include.
type FilterKind int

const (
	FilterOpen FilterKind = iota
	FilterDone
	FilterAll
)

// Match returns true if query is a case-insensitive substring of id or text.
func Match(query string, idea Idea) bool {
	q := strings.ToLower(query)
	return strings.Contains(strings.ToLower(idea.ID), q) ||
		strings.Contains(strings.ToLower(idea.Text), q)
}

// FindAll returns all ideas matching the query and filter.
func FindAll(query string, ideas []Idea, filter FilterKind) []Idea {
	var result []Idea
	for _, idea := range ideas {
		if !matchesFilter(idea, filter) {
			continue
		}
		if query == "" || Match(query, idea) {
			result = append(result, idea)
		}
	}
	return result
}

// RequireSingle finds exactly one matching idea. Returns the idea and its
// index in the original ideas slice. Errors if 0 or >1 matches.
func RequireSingle(query string, ideas []Idea, filter FilterKind) (Idea, int, error) {
	var matches []Idea
	var indices []int
	for i, idea := range ideas {
		if !matchesFilter(idea, filter) {
			continue
		}
		if Match(query, idea) {
			matches = append(matches, idea)
			indices = append(indices, i)
		}
	}

	if len(matches) == 0 {
		return Idea{}, -1, fmt.Errorf("No idea matching '%s'", query)
	}
	if len(matches) > 1 {
		var lines []string
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  %s", FormatLine(m)))
		}
		return Idea{}, -1, fmt.Errorf("Multiple matches:\n%s\n\nBe more specific or use the exact ID.", strings.Join(lines, "\n"))
	}
	return matches[0], indices[0], nil
}

func matchesFilter(idea Idea, filter FilterKind) bool {
	switch filter {
	case FilterOpen:
		return !idea.Done
	case FilterDone:
		return idea.Done
	default:
		return true
	}
}

// generateRandomID generates a random 4-char alphanumeric ID.
func generateRandomID() (string, error) {
	b := make([]byte, 4)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(idChars))))
		if err != nil {
			return "", err
		}
		b[i] = idChars[n.Int64()]
	}
	return string(b), nil
}

// Add appends a new idea to the backlog file. Creates the file and parent
// directories if they don't exist.
func Add(path, text, customID, customDate string) (Idea, error) {
	if text == "" {
		return Idea{}, fmt.Errorf("text is required")
	}

	// Normalize line endings up front so the in-memory Idea.Text equals what
	// round-trips from disk (FormatLine escapes on write; CR never survives).
	text = normalizeCR(text)

	// Validate custom ID format if provided
	if customID != "" {
		if err := ValidateID(customID); err != nil {
			return Idea{}, err
		}
	}

	// Validate custom date format if provided
	if customDate != "" {
		if err := ValidateDate(customDate); err != nil {
			return Idea{}, err
		}
	}

	// Determine ID
	id := customID
	if id == "" {
		var err error
		id, err = generateUniqueID(path, 10)
		if err != nil {
			return Idea{}, err
		}
	} else {
		// Check for collision
		if err := checkIDCollision(path, id); err != nil {
			return Idea{}, err
		}
	}

	// Determine date
	date := customDate
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	idea := Idea{
		ID:   id,
		Date: date,
		Text: text,
		Done: false,
	}

	// Auto-create file and dirs if missing
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return Idea{}, fmt.Errorf("create directories: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return Idea{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	// If the existing file does not end with a newline, prepend one so the
	// new entry starts on a fresh line instead of being glued to the previous
	// content. Determine this by stat-ing the file and reading the last byte.
	if needsLeadingNewline, err := lastByteIsNewline(path); err == nil && !needsLeadingNewline {
		if _, err := f.WriteString("\n"); err != nil {
			return Idea{}, fmt.Errorf("write separator: %w", err)
		}
	}

	_, err = fmt.Fprintln(f, FormatLine(idea))
	if err != nil {
		return Idea{}, fmt.Errorf("write idea: %w", err)
	}

	return idea, nil
}

// lastByteIsNewline returns true when the file at path is empty or its last
// byte is '\n'. It returns false when the file ends with any other byte.
// An error is returned only when the file cannot be inspected at all.
func lastByteIsNewline(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Size() == 0 {
		return true, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 1)
	if _, err := f.ReadAt(buf, info.Size()-1); err != nil {
		return false, err
	}
	return buf[0] == '\n', nil
}

func generateUniqueID(path string, maxRetries int) (string, error) {
	for i := 0; i < maxRetries; i++ {
		id, err := generateRandomID()
		if err != nil {
			return "", err
		}
		err = checkIDCollision(path, id)
		if err == nil {
			return id, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique ID after %d attempts", maxRetries)
}

func checkIDCollision(path, id string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist yet, no collision
		return nil
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if idea, ok := ParseLine(line); ok {
			if idea.ID == id {
				return fmt.Errorf("ID '%s' already exists", id)
			}
		}
	}
	return nil
}

// List returns ideas filtered, sorted, and optionally formatted as JSON.
// sortField must be "id" or "date"; any other value is rejected so typos like
// `--sort=data` fail loudly instead of silently falling through to date order.
func List(path string, filter FilterKind, sortField string, reverse bool) ([]Idea, error) {
	if sortField != "id" && sortField != "date" {
		return nil, fmt.Errorf("invalid sort field %q: must be 'id' or 'date'", sortField)
	}

	f, err := LoadFile(path)
	if err != nil {
		return nil, err
	}

	var result []Idea
	for _, idea := range f.ideas {
		if matchesFilter(idea, filter) {
			result = append(result, idea)
		}
	}

	// Sort
	sort.SliceStable(result, func(i, j int) bool {
		switch sortField {
		case "id":
			return result[i].ID < result[j].ID
		default: // "date"
			return result[i].Date < result[j].Date
		}
	})

	if reverse {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	return result, nil
}

// Show finds a single idea matching the query.
func Show(path, query string) (Idea, error) {
	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, err
	}

	idea, _, err := RequireSingle(query, f.ideas, FilterAll)
	if err != nil {
		return Idea{}, err
	}
	return idea, nil
}

// Done marks a single matching open idea as done. The returned count is the
// number of previously-dateless ideas whose date was backfilled to today on save
// (a side effect of normalize-on-write); the cmd layer surfaces it on stderr.
func Done(path, query string) (Idea, int, error) {
	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, 0, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterOpen)
	if err != nil {
		return Idea{}, 0, err
	}

	f.ideas[idx].Done = true
	backfilled, err := SaveFile(f, path)
	if err != nil {
		return Idea{}, 0, err
	}
	return f.ideas[idx], backfilled, nil
}

// Reopen marks a single matching done idea as open. See Done for the count.
func Reopen(path, query string) (Idea, int, error) {
	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, 0, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterDone)
	if err != nil {
		return Idea{}, 0, err
	}

	f.ideas[idx].Done = false
	backfilled, err := SaveFile(f, path)
	if err != nil {
		return Idea{}, 0, err
	}
	return f.ideas[idx], backfilled, nil
}

// Edit modifies a single matching idea's text, and optionally its ID and date.
// See Done for the meaning of the returned count.
func Edit(path, query, newText, newID, newDate string) (Idea, int, error) {
	if newText == "" {
		return Idea{}, 0, fmt.Errorf("text is required")
	}

	// Normalize line endings up front so the in-memory Idea.Text equals what
	// round-trips from disk (FormatLine escapes on write; CR never survives).
	newText = normalizeCR(newText)

	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, 0, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterAll)
	if err != nil {
		return Idea{}, 0, err
	}

	// Validate new ID format if provided
	if newID != "" {
		if err := ValidateID(newID); err != nil {
			return Idea{}, 0, err
		}
	}

	// Validate new date format if provided
	if newDate != "" {
		if err := ValidateDate(newDate); err != nil {
			return Idea{}, 0, err
		}
	}

	// Check ID collision if changing
	if newID != "" && newID != f.ideas[idx].ID {
		for i, idea := range f.ideas {
			if i != idx && idea.ID == newID {
				return Idea{}, 0, fmt.Errorf("ID '%s' already exists", newID)
			}
		}
		f.ideas[idx].ID = newID
	}

	if newDate != "" {
		f.ideas[idx].Date = newDate
	}

	f.ideas[idx].Text = newText

	backfilled, err := SaveFile(f, path)
	if err != nil {
		return Idea{}, 0, err
	}
	return f.ideas[idx], backfilled, nil
}

// Rm removes a single matching idea from the file. See Done for the count.
func Rm(path, query string, force bool) (Idea, int, error) {
	if !force {
		return Idea{}, 0, fmt.Errorf("Use --force to confirm deletion")
	}

	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, 0, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterAll)
	if err != nil {
		return Idea{}, 0, err
	}

	removed := f.ideas[idx]

	// Remove the idea line from lines and update indices
	lineIdx := f.ideaIndices[idx]
	f.lines = append(f.lines[:lineIdx], f.lines[lineIdx+1:]...)

	// Remove from ideas and ideaIndices
	f.ideas = append(f.ideas[:idx], f.ideas[idx+1:]...)
	f.ideaIndices = append(f.ideaIndices[:idx], f.ideaIndices[idx+1:]...)

	// Adjust line indices for ideas that come after the removed line
	for i := idx; i < len(f.ideaIndices); i++ {
		f.ideaIndices[i]--
	}

	backfilled, err := SaveFile(f, path)
	if err != nil {
		return Idea{}, 0, err
	}
	return removed, backfilled, nil
}
