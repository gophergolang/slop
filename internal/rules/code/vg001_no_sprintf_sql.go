package code

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// VG001NoSprintfSQL flags fmt.Sprintf calls whose result is passed to
// db.Exec / db.Query / db.QueryRow. A real implementation would use SSA
// taint analysis; this AST-level detector catches the obvious shape:
//
//	q := fmt.Sprintf("SELECT ... %s ...", userInput)
//	db.Exec(ctx, q)
//
// and the equally common inline form:
//
//	db.Exec(ctx, fmt.Sprintf(...), args...)
var VG001NoSprintfSQL = &analysis.Analyzer{
	Name: "VG001",
	Doc:  "report fmt.Sprintf calls used to build SQL passed to db.Exec/Query/QueryRow",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, file := range pass.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if !isDBSinkCall(call) {
					return true
				}
				// Inspect the SQL argument (typically index 1; index 0 is ctx).
				if len(call.Args) < 2 {
					return true
				}
				if isSprintfish(call.Args[1]) {
					pass.Reportf(call.Args[1].Pos(), "VG001: SQL string built via fmt.Sprintf or string concatenation; use parameterized arguments")
				}
				return true
			})
		}
		return nil, nil
	},
}

func isDBSinkCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	method := sel.Sel.Name
	switch method {
	case "Exec", "Query", "QueryRow":
		return true
	}
	return false
}

func isSprintfish(e ast.Expr) bool {
	switch v := e.(type) {
	case *ast.CallExpr:
		if sel, ok := v.Fun.(*ast.SelectorExpr); ok {
			id, ok := sel.X.(*ast.Ident)
			return ok && id.Name == "fmt" && (sel.Sel.Name == "Sprintf" || sel.Sel.Name == "Sprint")
		}
	case *ast.BinaryExpr:
		if v.Op.String() == "+" && containsString(v) {
			return true
		}
	}
	return false
}

func containsString(e *ast.BinaryExpr) bool {
	for _, side := range []ast.Expr{e.X, e.Y} {
		if lit, ok := side.(*ast.BasicLit); ok && strings.HasPrefix(lit.Value, `"`) {
			return true
		}
	}
	return false
}
