BINARY   := pulumi-resource-atlas
MODULE   := github.com/supabase/pulumi-atlas
CMD      := ./cmd/pulumi-resource-atlas

GOFLAGS  ?=
VERSION  ?= $(shell git describe --tags --match "v*" 2>/dev/null || echo "v0.0.0-dev")
LDFLAGS  := -s -w -X $(MODULE)/provider.Version=$(VERSION)

.PHONY: build test vet lint schema sdk clean

build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

schema: build
	pulumi package get-schema ./$(BINARY) > schema.json

sdk: build
	pulumi package gen-sdk ./$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf sdk/
