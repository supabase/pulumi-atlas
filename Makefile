TFGEN    = pulumi-tfgen-ripe-atlas
PROVIDER = pulumi-resource-ripe-atlas
BINDIR   = bin

# Disable workspace mode: this module uses replace directives in go.mod to
# reference ../atlasctl and ../terraform-provider-ripe-atlas locally. The
# go.work in this directory is for those sibling repos, not this module.
# export GOWORK = off

.PHONY: build generate install clean

build:
	go build -o $(BINDIR)/$(TFGEN)    ./provider/cmd/$(TFGEN)
	go build -o $(BINDIR)/$(PROVIDER) ./provider/cmd/$(PROVIDER)

generate: $(BINDIR)/$(TFGEN)
	./$(BINDIR)/$(TFGEN) schema   --out provider/cmd/$(PROVIDER)
	./$(BINDIR)/$(TFGEN) go       --out sdk/go
	./$(BINDIR)/$(TFGEN) nodejs   --out sdk/nodejs
	./$(BINDIR)/$(TFGEN) python   --out sdk/python

$(BINDIR)/$(TFGEN):
	go build -o $(BINDIR)/$(TFGEN) ./provider/cmd/$(TFGEN)

install: build
	mkdir -p "$(HOME)/.pulumi/plugins/resource-ripe-atlas-v0.0.1"
	cp $(BINDIR)/$(PROVIDER) "$(HOME)/.pulumi/plugins/resource-ripe-atlas-v0.0.1/$(PROVIDER)"

clean:
	rm -rf $(BINDIR) sdk/go sdk/nodejs sdk/python
