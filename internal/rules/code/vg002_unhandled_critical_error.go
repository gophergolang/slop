package code

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// VG002UnhandledCriticalError flags expression statements whose call returns
// an error and isn't checked, when the call targets a critical sink:
// db.Exec, events.Publisher.Publish, http.ResponseWriter.Write, gin.Context.JSON.
//
// This is intentionally narrower than `errcheck` — vibeguard wants loud
// failures on the four sinks the master prompt singles out, without imposing
// a project-wide errcheck regime.
var VG002UnhandledCriticalError = &analysis.Analyzer{
	Name: "VG002",
	Doc:  "report unhandled errors on critical sinks (db.Exec, events.Publish, http.Write, gin.JSON)",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, file := range pass.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				stmt, ok := n.(*ast.ExprStmt)
				if !ok {
					return true
				}
				call, ok := stmt.X.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				method := sel.Sel.Name
				if !isCriticalSink(method) {
					return true
				}
				if returnsError(pass, call) {
					pass.Reportf(call.Pos(), "VG002: critical sink %q returns error which is not checked", method)
				}
				return true
			})
		}
		return nil, nil
	},
}

func isCriticalSink(method string) bool {
	switch method {
	case "Exec", "Publish", "Write", "JSON", "EnqueueTx":
		return true
	}
	return false
}

func returnsError(pass *analysis.Pass, call *ast.CallExpr) bool {
	if pass.TypesInfo == nil {
		return false
	}
	t := pass.TypesInfo.TypeOf(call)
	if t == nil {
		return false
	}
	if named, ok := t.(*types.Named); ok && named.Obj() != nil && named.Obj().Name() == "error" {
		return true
	}
	if tup, ok := t.(*types.Tuple); ok {
		for i := 0; i < tup.Len(); i++ {
			if isErrorType(tup.At(i).Type()) {
				return true
			}
		}
	}
	return isErrorType(t)
}

func isErrorType(t types.Type) bool {
	if named, ok := t.(*types.Named); ok && named.Obj() != nil {
		return named.Obj().Name() == "error" || named.Obj().Pkg() == nil
	}
	if iface, ok := t.Underlying().(*types.Interface); ok {
		return iface.NumMethods() == 1 && iface.Method(0).Name() == "Error"
	}
	return false
}
