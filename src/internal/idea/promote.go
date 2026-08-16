package idea

import (
	"errors"
	"fmt"
	"io/fs"
)

// Promote moves a single matching idea from the source backlog to the
// destination backlog, preserving its ID, date, and status verbatim. The
// query resolves against the source via the shared matcher (RequireSingle
// with FilterAll — done ideas are promotable too).
//
// Write ordering is destination first, then source: the idea is appended to
// the destination and saved, and only after that write succeeds is it removed
// from the source and the source saved. A crash between the two writes leaves
// the idea duplicated (present in both files), never lost. Each file gets
// exactly one canonical write through the LoadFile/SaveFile seam, so
// normalize-on-write and date backfill apply to both as usual; the returned
// counts are the per-file backfill totals (see Done for their meaning), and
// the returned Idea is the promoted idea as it landed in the destination
// (a previously-dateless idea carries its newly stamped date).
//
// A destination idea with the same ID refuses the move with an error naming
// the ID — the ID is never silently re-minted, because external references
// (e.g. fab change folders) may key on it — and neither file is modified. A
// missing destination file is not an error: it loads as an empty backlog and
// is created on write (the atomicWriteFile MkdirAll seam). srcPath and
// dstPath are expected to be distinct files; the same-path no-op is the cmd
// layer's concern.
func Promote(srcPath, dstPath, query string) (Idea, int, int, error) {
	src, err := LoadFile(srcPath)
	if err != nil {
		return Idea{}, 0, 0, err
	}

	idea, idx, err := RequireSingle(query, src.ideas, FilterAll)
	if err != nil {
		return Idea{}, 0, 0, err
	}

	// A missing destination is an empty backlog, not an error (same posture
	// as Add's append path: the file is created on write).
	dst, err := LoadFile(dstPath)
	if errors.Is(err, fs.ErrNotExist) {
		dst = &File{}
	} else if err != nil {
		return Idea{}, 0, 0, err
	}

	// Refuse on ID collision in the destination — parsed ideas only, the same
	// accepted blind spot as checkIDCollision (a 4-char bracket inside an
	// unparseable line is invisible).
	for _, d := range dst.ideas {
		if d.ID == idea.ID {
			return Idea{}, 0, 0, fmt.Errorf("ID '%s' already exists in %s — resolve the collision manually (edit or rm one side), then retry", idea.ID, dstPath)
		}
	}

	dst.lines = append(dst.lines, FormatLine(idea))
	dst.ideaIndices = append(dst.ideaIndices, len(dst.lines)-1)
	dst.ideas = append(dst.ideas, idea)

	dstBackfilled, err := SaveFile(dst, dstPath)
	if err != nil {
		return Idea{}, 0, 0, err
	}

	// The destination now holds the idea; only then remove it from the source.
	removeIdeaAt(src, idx)
	srcBackfilled, err := SaveFile(src, srcPath)
	if err != nil {
		return Idea{}, 0, 0, err
	}

	// Return the idea as it landed in the destination (date backfilled).
	return dst.ideas[len(dst.ideas)-1], srcBackfilled, dstBackfilled, nil
}
