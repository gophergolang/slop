package parser

import "github.com/vibeguard/vibeguard/internal/ir"

// decodeSteps converts the wire-shape rawStep into the typed Step interface.
// Unknown step types are silently dropped — `vibeguard validate` is the
// authoritative complainer.
func decodeSteps(in []rawStep) []ir.Step {
	out := make([]ir.Step, 0, len(in))
	for _, s := range in {
		if step := decodeOne(s); step != nil {
			out = append(out, step)
		}
	}
	return out
}

func decodeOne(s rawStep) ir.Step {
	switch s.Type {
	case "validate":
		return ir.ValidateStep{Schema: s.Schema}.WithName(s.Name)
	case "load":
		return ir.LoadStep{Entity: s.Entity, IDPath: s.IDPath, OutputVar: s.OutputVar}.WithName(s.Name)
	case "authorize":
		return ir.AuthorizeStep{Roles: s.Roles, Condition: s.Condition}.WithName(s.Name)
	case "external_call":
		return ir.ExternalCallStep{
			Service:        s.Service,
			Action:         s.Action,
			PromptTemplate: s.PromptTemplate,
			OutputVar:      s.OutputVar,
		}.WithName(s.Name)
	case "update":
		return ir.UpdateStep{Entity: s.Entity, Fields: s.Fields}.WithName(s.Name)
	case "create":
		return ir.CreateStep{Entity: s.Entity, Fields: s.Fields}.WithName(s.Name)
	case "delete":
		return ir.DeleteStep{Entity: s.Entity, IDPath: s.IDPath}.WithName(s.Name)
	case "query":
		return ir.QueryStep{Entity: s.Entity, Where: s.Where, OutputVar: s.OutputVar}.WithName(s.Name)
	case "emit_event", "emit":
		return ir.EmitEventStep{Event: s.Event, Payload: s.Payload}.WithName(s.Name)
	case "consume":
		return ir.ConsumeStep{Subject: s.Subject, Queue: s.Queue}.WithName(s.Name)
	case "log":
		return ir.LogStep{Level: s.Level, Message: s.Message}.WithName(s.Name)
	case "return":
		return ir.ReturnStep{Status: s.Status, Body: s.Body}.WithName(s.Name)
	case "transaction":
		// Inner steps not yet supported in raw shape; placeholder to keep
		// validators happy.
		return ir.TransactionStep{}.WithName(s.Name)
	}
	return nil
}
