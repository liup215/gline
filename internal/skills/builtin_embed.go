package skills

import (
	"embed"
	"fmt"
	"io"
	"os"
)

//go:embed builtin/*.yaml
var builtinFS embed.FS

// writeBuiltinSkill copies an embedded built-in skill file to dest on disk.
func writeBuiltinSkill(name, dest string) error {
	src := "builtin/" + name
	f, err := builtinFS.Open(src)
	if err != nil {
		return fmt.Errorf("open embedded %s: %w", src, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read embedded %s: %w", src, err)
	}

	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return nil
}
