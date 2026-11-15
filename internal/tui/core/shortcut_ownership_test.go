package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Keep shortcut definitions in core so widgets cannot drift from shared bindings.
func TestShortcutDefinitionsBelongToCore(t *testing.T) {
	t.Parallel()

	files := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == filepath.Join("..", "core") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(files, path, nil, 0)
		if err != nil {
			return err
		}
		owners := map[string]string{}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if importPath != "github.com/jadegopher/trabbit/internal/tui/core" && importPath != "charm.land/bubbles/v2/key" {
				continue
			}
			name := filepath.Base(importPath)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			owners[name] = filepath.Base(importPath)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			owner, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			if owners[owner.Name] == "core" && selector.Sel.Name == "NewShortcut" ||
				owners[owner.Name] == "key" && (selector.Sel.Name == "NewBinding" || selector.Sel.Name == "WithKeys") {
				t.Errorf("%s: use a shared core shortcut helper", files.Position(call.Pos()))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
