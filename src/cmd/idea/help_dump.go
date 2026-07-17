package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// helpNode is one node in the serialized CLI help tree. The JSON shape is a
// frozen cross-repo contract shared across a 7-tool rollout; the canonical
// reference sample is sahil87/shll.ai help/wt.json. Field names, ordering, and
// the recursive "commands" array (empty array for leaves, never null) are part
// of that contract — do not reorder or rename.
type helpNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Short    string     `json:"short"`
	Usage    string     `json:"usage"`
	Text     string     `json:"text"`
	Commands []helpNode `json:"commands"`
}

// helpDump is the top-level envelope wrapping the help tree. The envelope is
// exactly {tool, version, schema_version, root} — it deliberately does NOT carry
// a captured_at field: the capture timestamp is owned by shll.ai's puller (a tool
// cannot know its own capture time), per the toolkit help-dump standard.
type helpDump struct {
	Tool          string   `json:"tool"`
	Version       string   `json:"version"`
	SchemaVersion int      `json:"schema_version"`
	Root          helpNode `json:"root"`
}

// helpSchemaVersion is the contract schema version emitted in the envelope.
const helpSchemaVersion = 1

// longOrShort returns the command's full description for the "text" field:
// Long if set, otherwise the one-line Short. Both may be empty for some
// commands; callers guard that case.
func longOrShort(c *cobra.Command) string {
	if c.Long != "" {
		return c.Long
	}
	return c.Short
}

// buildNode recursively serializes a command and its visible children into a
// helpNode. It walks the live cobra tree programmatically (never regex on -h),
// filtering out hidden commands and cobra's auto-generated "completion"/"help"
// subcommands. The "completion"/"help" filter, combined with the Hidden filter,
// also excludes the help-dump command itself.
func buildNode(cmd *cobra.Command) helpNode {
	// Cobra adds the default "-h, --help" flag (and, on a versioned command,
	// "-v, --version") lazily inside Execute(). help-dump walks the child
	// commands without executing them, so those flags are not yet registered
	// and UsageString() would omit them — diverging from the byte-for-byte -h
	// output the contract requires. Initialize them explicitly before reading
	// UsageString(). Both methods are idempotent and side-effect-free beyond
	// flag registration: InitDefaultHelpFlag() always adds "-h, --help";
	// InitDefaultVersionFlag() materializes "-v, --version" only when
	// cmd.Version != "" (so it adds the flag on the versioned root and is a
	// no-op on subcommands).
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()

	// text reproduces what "idea <cmd> -h" prints: the Long (or Short)
	// description, a blank line, then the Usage:/Flags: blocks. UsageString()
	// does not include Long/Short, so concatenate explicitly. When both Long
	// and Short are empty, emit just UsageString() with no leading blanks.
	text := cmd.UsageString()
	if desc := longOrShort(cmd); desc != "" {
		text = desc + "\n\n" + cmd.UsageString()
	}

	n := helpNode{
		Name:     cmd.Name(),
		Path:     cmd.CommandPath(),
		Short:    cmd.Short,
		Usage:    cmd.UseLine(),
		Text:     text,
		Commands: []helpNode{}, // never nil → JSON "[]" for leaves
	}

	for _, c := range cmd.Commands() {
		if c.Hidden || c.Name() == "completion" || c.Name() == "help" {
			continue
		}
		n.Commands = append(n.Commands, buildNode(c))
	}

	return n
}

// helpDumpCmd builds the hidden "help-dump" subcommand. It emits the CLI help
// tree as JSON to stdout for build tooling (the shll.ai command-reference
// publisher consumes it). It is hidden so it never appears in user-facing help
// nor in its own dump output.
func helpDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "help-dump",
		Short:  "Emit the CLI help tree as JSON (build tooling)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			dump := helpDump{
				Tool:          "idea",
				Version:       root.Version,
				SchemaVersion: helpSchemaVersion,
				Root:          buildNode(root),
			}

			out, err := json.MarshalIndent(dump, "", "  ")
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", out); err != nil {
				return err
			}
			return nil
		},
	}
}
