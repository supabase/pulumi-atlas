TFGEN    = pulumi-tfgen-ripe-atlas
PROVIDER = pulumi-resource-ripe-atlas
BINDIR   = bin
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.1")
LDFLAGS  := -X github.com/supabase/pulumi-atlas/provider/version.Version=$(VERSION)

.PHONY: build tfgen generate install clean sync-sdk

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(TFGEN)    ./provider/cmd/$(TFGEN)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(PROVIDER) ./provider/cmd/$(PROVIDER)

generate: tfgen
	GOWORK=off ./$(BINDIR)/$(TFGEN) schema   --out provider/cmd/$(PROVIDER)
	GOWORK=off ./$(BINDIR)/$(TFGEN) go       --out sdk/go
	GOWORK=off ./$(BINDIR)/$(TFGEN) nodejs   --out sdk/nodejs
	GOWORK=off ./$(BINDIR)/$(TFGEN) python   --out sdk/python
	GOWORK=off ./$(BINDIR)/$(TFGEN) dotnet   --out sdk/dotnet

# Phony so tfgen is always rebuilt against current sources. go build's cache makes
# this a no-op when nothing changed; a file target would reuse a stale binary.
tfgen:
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(TFGEN) ./provider/cmd/$(TFGEN)

PLATFORM_PULUMI ?= ../platform/pulumi

sync-sdk: generate
	rsync -a --delete \
		--exclude=node_modules \
		--exclude=package-lock.json \
		sdk/nodejs/ $(PLATFORM_PULUMI)/ripe-atlas-sdk/
	sed -i '' 's/"version": "[^"]*"/"version": "$(VERSION)"/' \
		$(PLATFORM_PULUMI)/ripe-atlas-sdk/package.json

install: build
	mkdir -p "$(HOME)/.pulumi/plugins/resource-ripe-atlas-v0.0.1"
	cp $(BINDIR)/$(PROVIDER) "$(HOME)/.pulumi/plugins/resource-ripe-atlas-v0.0.1/$(PROVIDER)"

clean:
	rm -rf $(BINDIR) sdk/go sdk/nodejs sdk/python
