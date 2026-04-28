package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/vibeguard/vibeguard/internal/parser"
)

func runIR(args []string) {
	if len(args) == 0 {
		die("usage: vibeguard ir dump -f <file>")
	}
	switch args[0] {
	case "dump":
		runIRDump(args[1:])
	default:
		die("vibeguard ir: unknown subcommand %q (try 'dump')", args[0])
	}
}

func runIRDump(args []string) {
	fs := flag.NewFlagSet("ir dump", flag.ExitOnError)
	file := fs.String("f", "vibeguard.yaml", "path to declaration")
	format := fs.String("format", "yaml", "output format: yaml | json")
	_ = fs.Parse(args)

	data, err := os.ReadFile(*file)
	if err != nil {
		die("read %s: %v", *file, err)
	}
	app, err := parser.Parse(data)
	if err != nil {
		die("parse: %v", err)
	}
	switch *format {
	case "json":
		out, _ := json.MarshalIndent(app, "", "  ")
		fmt.Println(string(out))
	default:
		// summary form — readable
		fmt.Printf("Application: %s (apiVersion=%s, version=%s)\n", app.Metadata.Name, app.APIVersion, app.Metadata.Version)
		if len(app.Metadata.Compliance) > 0 {
			fmt.Printf("  compliance: %v\n", app.Metadata.Compliance)
		}
		if app.Global.MultiTenancy.Enabled {
			fmt.Printf("  multi_tenancy: enabled (isolation=%s, tenant_id_field=%s)\n",
				app.Global.MultiTenancy.Isolation, app.Global.MultiTenancy.TenantIDField)
		}
		fmt.Println()
		for _, mod := range app.Modules {
			fmt.Printf("module %s (type=%s)\n", mod.Name, mod.Type)
			for _, ent := range mod.Entities {
				fmt.Printf("  entity %s (table=%s, sensitivity=%s)\n", ent.Name, ent.Table, ent.Sensitivity)
				crud := []string{}
				if ent.CRUD.Create {
					crud = append(crud, "create")
				}
				if ent.CRUD.Read {
					crud = append(crud, "read")
				}
				if ent.CRUD.List {
					crud = append(crud, "list")
				}
				if len(ent.CRUD.UpdateFields) > 0 {
					updateFields := []string{}
					for _, f := range ent.CRUD.UpdateFields {
						updateFields = append(updateFields, f.Name)
					}
					crud = append(crud, fmt.Sprintf("update[%v]", updateFields))
				}
				if ent.CRUD.Delete {
					crud = append(crud, "delete")
				}
				if ent.SoftDelete {
					crud = append(crud, "soft_delete")
				}
				fmt.Printf("    crud: %v\n", crud)
				fmt.Printf("    fields (%d): ", len(ent.Fields))
				for i, f := range ent.Fields {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%s:%s", f.Name, f.Type)
				}
				fmt.Println()
				if len(ent.API.CustomEndpoints) > 0 {
					for _, ep := range ent.API.CustomEndpoints {
						fmt.Printf("    custom: %s %s — %d steps\n", ep.Method, ep.Path, len(ep.Logic.Steps))
						for _, st := range ep.Logic.Steps {
							fmt.Printf("      - %T %s\n", st, st.StepName())
						}
					}
				}
			}
			if len(mod.Events) > 0 {
				fmt.Printf("  events: ")
				for i, e := range mod.Events {
					if i > 0 {
						fmt.Print(", ")
					}
					fmt.Printf("%s(%s)", e.Name, e.Trigger)
				}
				fmt.Println()
			}
		}
	}
}
