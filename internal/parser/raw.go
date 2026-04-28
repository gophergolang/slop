package parser

// rawApplication mirrors the on-disk YAML shape. Decoding happens in two
// phases: first into these raw structs, then into the typed IR. This split
// lets the IR carry resolved pointers (Entity.Module, CRUD.UpdateFields,
// Relationship.Resolved) without polluting the YAML schema.

type rawApplication struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   rawMetadata `yaml:"metadata"`
	Spec       rawSpec     `yaml:"spec"`
}

type rawMetadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Compliance  []string `yaml:"compliance"`
}

type rawSpec struct {
	Global  *rawGlobal  `yaml:"global"`
	Modules []rawModule `yaml:"modules"`
}

type rawGlobal struct {
	MultiTenancy rawMultiTenancy `yaml:"multi_tenancy"`
}

type rawMultiTenancy struct {
	Enabled       bool   `yaml:"enabled"`
	TenantIDField string `yaml:"tenant_id_field"`
	Isolation     string `yaml:"isolation"`
}

type rawModule struct {
	Name         string           `yaml:"name"`
	Type         string           `yaml:"type"`
	Entities     []rawEntity      `yaml:"entities"`
	Integrations []rawIntegration `yaml:"integrations"`
	Policies     rawPolicies      `yaml:"policies"`
	Events       []rawEvent       `yaml:"events"`
}

type rawEntity struct {
	Name          string            `yaml:"name"`
	Table         string            `yaml:"table"`
	Sensitivity   string            `yaml:"sensitivity"`
	Fields        []rawField        `yaml:"fields"`
	Relationships []rawRelationship `yaml:"relationships"`
	CRUD          rawCRUD           `yaml:"crud"`
	API           rawAPI            `yaml:"api"`
	BusinessRules []string          `yaml:"business_rules"`
}

type rawField struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Primary  bool     `yaml:"primary"`
	Nullable bool     `yaml:"nullable"`
	Unique   bool     `yaml:"unique"`
	Validate []string `yaml:"validate"`
	Values   []string `yaml:"values"`
	Default  string   `yaml:"default"`
	DB       struct {
		Index bool `yaml:"index"`
	} `yaml:"db"`
}

type rawRelationship struct {
	To         string `yaml:"to"`
	Type       string `yaml:"type"`
	ForeignKey string `yaml:"foreign_key"`
}

type rawCRUD struct {
	Create     bool       `yaml:"create"`
	Read       bool       `yaml:"read"`
	List       bool       `yaml:"list"`
	Update     updateSpec `yaml:"update"`
	Delete     bool       `yaml:"delete"`
	SoftDelete bool       `yaml:"soft_delete"`
}

// updateSpec accepts either a bool (false = disable update entirely; true =
// allow updates to all non-primary fields) or a []string (whitelist of
// settable column names).
type updateSpec struct {
	Enabled bool
	Fields  []string
}

// UnmarshalYAML decodes the polymorphic update field.
func (u *updateSpec) UnmarshalYAML(value *yamlNode) error {
	if value == nil {
		return nil
	}
	switch value.Kind {
	case yamlScalarKind:
		var b bool
		if err := value.Decode(&b); err == nil {
			u.Enabled = b
			return nil
		}
		return nil
	case yamlSequenceKind:
		var s []string
		if err := value.Decode(&s); err != nil {
			return err
		}
		u.Fields = s
		u.Enabled = len(s) > 0
		return nil
	}
	return nil
}

type rawAPI struct {
	BasePath        string              `yaml:"base_path"`
	AuthRequired    bool                `yaml:"auth_required"`
	RolesAllowed    []string            `yaml:"roles_allowed"`
	RateLimit       string              `yaml:"rate_limit"`
	CustomEndpoints []rawCustomEndpoint `yaml:"custom_endpoints"`
}

type rawCustomEndpoint struct {
	Path         string   `yaml:"path"`
	Method       string   `yaml:"method"`
	Description  string   `yaml:"description"`
	Request      string   `yaml:"request"`
	Response     string   `yaml:"response"`
	AuthRequired bool     `yaml:"auth_required"`
	Logic        rawLogic `yaml:"logic"`
}

type rawLogic struct {
	Description string    `yaml:"description"`
	Steps       []rawStep `yaml:"steps"`
}

type rawStep struct {
	Name           string            `yaml:"name"`
	Type           string            `yaml:"type"`
	Schema         string            `yaml:"schema"`
	Entity         string            `yaml:"entity"`
	IDPath         string            `yaml:"id_path"`
	OutputVar      string            `yaml:"output_var"`
	Roles          []string          `yaml:"roles"`
	Condition      string            `yaml:"condition"`
	Service        string            `yaml:"service"`
	Action         string            `yaml:"action"`
	PromptTemplate string            `yaml:"prompt_template"`
	Fields         map[string]string `yaml:"fields"`
	Event          string            `yaml:"event"`
	Payload        map[string]string `yaml:"payload"`
	Status         int               `yaml:"status"`
	Body           string            `yaml:"body"`
	Subject        string            `yaml:"subject"`
	Queue          string            `yaml:"queue"`
	Where          string            `yaml:"where"`
	MaxAttempts    int               `yaml:"max_attempts"`
	Backoff        string            `yaml:"backoff"`
	Key            string            `yaml:"key"`
	TTL            string            `yaml:"ttl"`
	Level          string            `yaml:"level"`
	Message        string            `yaml:"message"`
	For            string            `yaml:"for"`
}

type rawIntegration struct {
	Name     string            `yaml:"name"`
	Config   map[string]string `yaml:"config"`
	Features []string          `yaml:"features"`
}

type rawPolicies struct {
	RowLevelSecurity []rawRLS `yaml:"row_level_security"`
}

type rawRLS struct {
	Entity    string   `yaml:"entity"`
	Condition string   `yaml:"condition"`
	ApplyTo   []string `yaml:"apply_to"`
}

type rawEvent struct {
	Name      string `yaml:"name"`
	Trigger   string `yaml:"trigger"`
	Entity    string `yaml:"entity"`
	Condition string `yaml:"condition"`
	PublishTo string `yaml:"publish_to"`
}
