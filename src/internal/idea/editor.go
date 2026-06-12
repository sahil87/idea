package idea

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// defaultEditor is the last-resort editor when neither $VISUAL nor $EDITOR
// is set — the conventional Unix fallback (git uses the same chain).
const defaultEditor = "vi"

// editorTempPattern names the temp file handed to the editor; the .md
// extension buys markdown syntax highlighting in most editors.
const editorTempPattern = "idea-edit-*.md"

// ResolveEditor returns the editor command to launch: $VISUAL if non-empty,
// else $EDITOR if non-empty, else vi. The returned value is a command
// string that may contain arguments (e.g. "code --wait"); EditInEditor
// runs it through the shell so multi-word values work.
func ResolveEditor() string {
	if v := os.Getenv("VISUAL"); v != "" {
		return v
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return defaultEditor
}

// EditInEditor round-trips text through the user's editor: it seeds a temp
// file with text (real decoded form — raw newlines, raw backslashes), opens
// the editor resolved by ResolveEditor on it with inherited stdio, and on a
// clean (zero) exit returns the buffer post-processed for persistence:
// CR-normalized, with exactly one trailing LF stripped. Editors
// conventionally append a final newline, so stripping one means a one-line
// idea round-trips without gaining a newline; deliberate extra trailing
// blank lines beyond that final LF are kept as content.
//
// unchanged reports whether the session left the text as-is. It compares
// both the stripped buffer and the pre-strip (CR-normalized only) buffer
// against text: the stripped comparison makes an editor-appended final LF a
// no-op for ordinary text, while the pre-strip comparison makes an untouched
// buffer a no-op even when text itself ends in an LF (which the strip would
// otherwise eat). A deliberate edit of "a\n" to "a" still registers as
// changed, since neither form of the buffer matches the original.
//
// A non-zero editor exit (or a failure to launch) returns an error and the
// caller must persist nothing. There is no TTY gate: a TTY-requiring editor
// (vi) fails fast on its own with a non-zero exit in non-interactive
// contexts, while a script $EDITOR stays viable headlessly.
func EditInEditor(text string) (edited string, unchanged bool, err error) {
	tmp, err := os.CreateTemp("", editorTempPattern)
	if err != nil {
		return "", false, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		return "", false, fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", false, fmt.Errorf("close temp file: %w", err)
	}

	editor := ResolveEditor()
	// Launch through the shell so multi-word editor values work; the path
	// rides as a positional parameter to sidestep quoting.
	cmd := exec.Command("sh", "-c", editor+` "$1"`, "sh", tmpPath)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", false, fmt.Errorf("editor (%s) failed: %w — idea unchanged", editor, err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", false, fmt.Errorf("read edited temp file: %w", err)
	}
	preStrip := normalizeCR(string(data))
	stripped := strings.TrimSuffix(preStrip, "\n")
	return stripped, stripped == text || preStrip == text, nil
}
