package parser

import "gopkg.in/yaml.v3"

// Aliases so the polymorphic decoder in raw.go doesn't drag the gopkg.in
// import into a public API surface.

type yamlNode = yaml.Node

const (
	yamlScalarKind   = yaml.ScalarNode
	yamlSequenceKind = yaml.SequenceNode
)
