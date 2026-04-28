.PHONY: generate build test

generate:
	go run ./cmd/vibeguard -f sample_vibeguard.yaml
	@echo "✅ Full project generated with Platform SDK + Kubernetes + NATS"

build:
	go build -o bin/vibeguard ./cmd/vibeguard

test:
	go test ./... -cover

clean:
	rm -rf bin/ my-app/ team-task-saas/