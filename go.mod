module github.com/vibeguard/vibeguard

go 1.25.0

require (
	github.com/owenrumney/go-sarif/v2 v2.3.3
	github.com/vibeguard/platform v0.0.0-00010101000000-000000000000
	golang.org/x/tools v0.44.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
)

replace github.com/vibeguard/platform => ./platform
