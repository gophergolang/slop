package lint

import (
	"io"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

// writeSARIF emits a SARIF 2.1.0 document. GitHub PR annotations consume
// SARIF natively when uploaded with github/codeql-action/upload-sarif.
func writeSARIF(w io.Writer, findings []Finding) error {
	report, err := sarif.New(sarif.Version210)
	if err != nil {
		return err
	}
	run := sarif.NewRunWithInformationURI("vibeguard-lint", "https://github.com/gophergolang/slop")
	for _, f := range findings {
		level := "warning"
		switch f.Severity {
		case "error":
			level = "error"
		case "info":
			level = "note"
		}
		run.AddDistinctArtifact(f.File)
		result := sarif.NewRuleResult(f.RuleID).
			WithLevel(level).
			WithMessage(sarif.NewTextMessage(f.Message))
		loc := sarif.NewLocationWithPhysicalLocation(
			sarif.NewPhysicalLocation().
				WithArtifactLocation(sarif.NewSimpleArtifactLocation(f.File)).
				WithRegion(sarif.NewRegion().WithStartLine(f.Line).WithStartColumn(f.Col)),
		)
		result.AddLocation(loc)
		run.AddResult(result)
	}
	report.AddRun(run)
	return report.Write(w)
}
