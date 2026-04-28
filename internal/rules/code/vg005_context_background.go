package code

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// VG005ContextBackgroundInRequest flags context.Background() / context.TODO()
// calls inside functions whose first parameter is *gin.Context, http.ResponseWriter,
// or *http.Request — request handlers must propagate the incoming request
// context for cancellation, deadlines, and tracing.
var VG005ContextBackgroundInRequest = &analysis.Analyzer{
	Name: "VG005",
	Doc:  "report context.Background()/TODO() calls inside HTTP request handlers",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, file := range pass.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}
				if !isRequestHandler(fn) {
					return true
				}
				ast.Inspect(fn, func(inner ast.Node) bool {
					call, ok := inner.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					id, ok := sel.X.(*ast.Ident)
					if !ok || id.Name != "context" {
						return true
					}
					if sel.Sel.Name == "Background" || sel.Sel.Name == "TODO" {
						pass.Reportf(call.Pos(), "VG005: handler uses context.%s; propagate the request context instead", sel.Sel.Name)
					}
					return true
				})
				return false
			})
		}
		return nil, nil
	},
}

func isRequestHandler(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}
	if len(fn.Type.Params.List) == 0 {
		return false
	}
	first := fn.Type.Params.List[0].Type
	switch t := first.(type) {
	case *ast.StarExpr:
		if sel, ok := t.X.(*ast.SelectorExpr); ok {
			id, _ := sel.X.(*ast.Ident)
			return id != nil && id.Name == "gin" && sel.Sel.Name == "Context"
		}
	case *ast.SelectorExpr:
		id, _ := t.X.(*ast.Ident)
		return id != nil && id.Name == "http" && (t.Sel.Name == "ResponseWriter" || t.Sel.Name == "Request")
	}
	return false
}
