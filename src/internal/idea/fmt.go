package idea

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// adoptRegex matches a bare markdown checkbox line lacking the 4-char [id]
// anchor — the adoption-candidate shape for "idea fmt". It mirrors lineRegex's
// lenient surface (-/*/+ bullets, leading whitespace) and additionally accepts
// an uppercase [X] checkbox. A line is only a candidate when ParseLine has
// already rejected it, so managed idea lines never reach this regex.
var adoptRegex = regexp.MustCompile(`^\s*[-*+] \[([ xX])\] (.+)$`)

// FmtResult reports what Fmt changed — or, in check mode, would change.
type FmtResult struct {
	// Adopted holds the newly adopted ideas, in file order.
	Adopted []Idea
	// Normalized counts pre-existing managed idea lines whose canonical form
	// differs from their raw on-disk text (adopted lines are not counted).
	Normalized int
	// Backfilled counts pre-existing managed dateless lines stamped with
	// today's date (adopted lines carry today's date but count as Adopted).
	Backfilled int
	// Changed reports whether the rebuilt content differs from the on-disk
	// bytes. It drives both the write (skipped when false) and the --check
	// exit code.
	Changed bool
}

// Fmt rewrites the backlog at path into canonical form: every recognized idea
// line is regenerated via the existing render machinery (variant bullets,
// indentation, CRLF endings, dateless lines, and legacy lone backslashes all
// canonicalize), and bare checkbox lines without an [id] anchor are adopted as
// managed ideas (see adoptBareCheckboxes). Non-idea, non-candidate content
// passes through verbatim.
//
// Fmt is idempotent: when the rebuilt content is byte-identical to the file,
// the write is skipped entirely (no mtime churn). With check set, Fmt never
// writes — it only reports what a real run would do. Fmt writes nothing to
// stderr; counts flow up in FmtResult and the cmd layer prints (Constitution
// IV split).
func Fmt(path string, check bool) (FmtResult, error) {
	var res FmtResult

	data, err := os.ReadFile(path)
	if err != nil {
		return res, err
	}
	f := parseContent(string(data))
	if len(f.lines) == 0 {
		// Empty (0-byte) file: nothing to canonicalize, and inventing a
		// trailing newline would violate verbatim preservation.
		return res, nil
	}

	// Count what canonicalization will touch among the pre-existing managed
	// lines, comparing each regenerated line against its raw on-disk text.
	today := time.Now().Format("2006-01-02")
	for i, lineIdx := range f.ideaIndices {
		stamped := f.ideas[i]
		if stamped.Date == "" {
			stamped.Date = today
			res.Backfilled++
		}
		if FormatLine(stamped) != f.lines[lineIdx] {
			res.Normalized++
		}
	}

	res.Adopted, err = adoptBareCheckboxes(f, today)
	if err != nil {
		return res, err
	}

	content, _ := render(f)
	res.Changed = content != string(data)
	if check || !res.Changed {
		return res, nil
	}
	if err := atomicWriteFile(path, []byte(content), 0644); err != nil {
		return res, err
	}
	return res, nil
}

// adoptBareCheckboxes converts bare checkbox lines (adoption candidates) into
// managed ideas, merging them into f.ideas/f.ideaIndices in file order, and
// returns the adopted ideas.
//
// A line is an adoption candidate iff it did not parse as an idea (it sits in
// a non-idea slot) AND it matches adoptRegex AND its whitespace-trimmed text
// neither is blank nor begins with a [...] bracket (the precision guard
// keeping Shape B and bracket-metadata lines like `- [ ] [DEV-1011] ...`
// inert — trimming first so extra spaces cannot defeat the guard). Each
// adopted idea gets a fresh 4-char ID unique against both the IDs already in
// the file and IDs assigned earlier in the same pass, today's date, its
// checked state preserved ([x]/[X] -> done), and its trimmed text taken as
// real text (it is escaped on write via FormatLine like every other idea).
func adoptBareCheckboxes(f *File, today string) ([]Idea, error) {
	used := make(map[string]bool, len(f.ideas))
	for _, i := range f.ideas {
		used[i.ID] = true
	}
	ideaAt := make(map[int]bool, len(f.ideaIndices))
	for _, lineIdx := range f.ideaIndices {
		ideaAt[lineIdx] = true
	}

	var adopted []Idea
	var adoptedIndices []int
	for lineIdx, line := range f.lines {
		if ideaAt[lineIdx] {
			continue
		}
		m := adoptRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		// The guard and the adopted text both use the whitespace-trimmed
		// capture: extra spaces between the checkbox and a bracket must not
		// defeat the guard (e.g. `- [ ]  [DEV-1011] x`), and leading/trailing
		// spaces are checkbox surface formatting, not content.
		text := strings.TrimSpace(m[2])
		// Precision guard: bracket-led text ([DEV-1011], [TODO], Shape B
		// remnants) stays verbatim pass-through — err toward preservation.
		// Blank text (whitespace-only checkbox) is likewise not adopted.
		if text == "" || shapeBPrefixRegex.MatchString(text) {
			continue
		}
		id, err := generateUniqueIDInSet(used, 10)
		if err != nil {
			return nil, err
		}
		used[id] = true
		adopted = append(adopted, Idea{
			ID:   id,
			Date: today,
			Text: text,
			Done: m[1] == "x" || m[1] == "X",
		})
		adoptedIndices = append(adoptedIndices, lineIdx)
	}
	if len(adopted) == 0 {
		return nil, nil
	}

	// Merge the adopted ideas into the existing (sorted-by-line) slices so
	// f.ideas stays in file order for render's rebuild.
	mergedIdeas := make([]Idea, 0, len(f.ideas)+len(adopted))
	mergedIndices := make([]int, 0, len(f.ideaIndices)+len(adoptedIndices))
	e, a := 0, 0
	for e < len(f.ideas) || a < len(adopted) {
		if a >= len(adopted) || (e < len(f.ideas) && f.ideaIndices[e] < adoptedIndices[a]) {
			mergedIdeas = append(mergedIdeas, f.ideas[e])
			mergedIndices = append(mergedIndices, f.ideaIndices[e])
			e++
		} else {
			mergedIdeas = append(mergedIdeas, adopted[a])
			mergedIndices = append(mergedIndices, adoptedIndices[a])
			a++
		}
	}
	f.ideas = mergedIdeas
	f.ideaIndices = mergedIndices

	return adopted, nil
}

// generateUniqueIDInSet generates a random 4-char ID not present in used.
// Unlike generateUniqueID (which checks the on-disk file), uniqueness is
// checked against an in-memory set so IDs assigned earlier in the same fmt
// pass cannot collide before anything is written.
func generateUniqueIDInSet(used map[string]bool, maxRetries int) (string, error) {
	for i := 0; i < maxRetries; i++ {
		id, err := generateRandomID()
		if err != nil {
			return "", err
		}
		if !used[id] {
			return id, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique ID after %d attempts", maxRetries)
}
