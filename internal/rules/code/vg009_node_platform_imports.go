package code

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// VG009NodePlatformImports flags any file under an `internal/<module>/nodes/`
// directory — except the package's `nodes.go` boilerplate — that imports
// platform packages (`platform/db`, `platform/events`, `platform/llm`)
// directly. Node bodies must reach those subsystems via the `Deps` argument
// the wrapper hands them, so the wrapper retains its monopoly on tenant
// binding, request-context propagation, and observability.
//
// The `nodes/nodes.go` file is allowed to import these packages because that
// is where the `Deps` struct is *defined*; everything else must depend on
// `Deps` (declared in the same package, no extra import needed).
var VG009NodePlatformImports = &analysis.Analyzer{
	Name: "VG009",
	Doc:  "report platform/{db,events,llm} imports inside node bodies — use Deps instead",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, file := range pass.Files {
			fname := pass.Fset.File(file.Pos()).Name()
			if !isNodeStubFile(fname) {
				continue
			}
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if !forbiddenNodeImport(path) {
					continue
				}
				pass.Reportf(imp.Pos(), "VG009: node body imports %q directly; use the Deps argument the wrapper passes in", path)
			}
		}
		return nil, nil
	},
}

func isNodeStubFile(path string) bool {
	dir := filepath.ToSlash(filepath.Dir(path))
	if !strings.HasSuffix(dir, "/nodes") {
		return false
	}
	return filepath.Base(path) != "nodes.go"
}

func forbiddenNodeImport(path string) bool {
	switch path {
	case "github.com/vibeguard/platform/db",
		"github.com/vibeguard/platform/events",
		"github.com/vibeguard/platform/llm":
		return true
	}
	return false
}
