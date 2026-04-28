// Package code is the registry of master-prompt-derived static analyzers.
//
// Each analyzer enforces one of the master prompt's iron rules. Analyzers
// receive a *analysis.Pass and emit diagnostics via Pass.Reportf — the
// standard golang.org/x/tools/go/analysis contract.
//
// To add a rule: write one file, expose `var Analyzer = &analysis.Analyzer{}`,
// append to AllAnalyzers().
package code

import "golang.org/x/tools/go/analysis"

// AllAnalyzers returns every registered analyzer in stable order.
func AllAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		VG001NoSprintfSQL,
		VG002UnhandledCriticalError,
		VG005ContextBackgroundInRequest,
		VG006EditedGeneratedCode,
		VG009NodePlatformImports,
	}
}
