package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

const qsServerModule = "github.com/FangcunMount/qs-server"

func TestComponentBaseDoesNotImportQSServer(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath := strings.Trim(imported.Path.Value, `"`)
			if strings.HasPrefix(importPath, qsServerModule) {
				rel, _ := filepath.Rel(repoRoot, path)
				t.Fatalf("%s imports %s; component-base must not depend on qs-server", rel, importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
