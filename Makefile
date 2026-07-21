TFGEN    = pulumi-tfgen-ripe-atlas
PROVIDER = pulumi-resource-ripe-atlas
BINDIR   = bin

.PHONY: build generate install clean sync-sdk

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

PLATFORM_PULUMI ?= ../platform/pulumi

sync-sdk: generate
	rsync -a --delete \
		--exclude=node_modules \
		--exclude=package-lock.json \
		sdk/nodejs/ $(PLATFORM_PULUMI)/ripe-atlas-sdk/
	sed -i '' 's/"version": "[^"]*"/"version": "0.0.1"/' \
		$(PLATFORM_PULUMI)/ripe-atlas-sdk/package.json

install: build
	mkdir -p "$(HOME)/.pulumi/plugins/resource-ripe-atlas-v0.0.1"
	cp $(BINDIR)/$(PROVIDER) "$(HOME)/.pulumi/plugins/resource-ripe-atlas-v0.0.1/$(PROVIDER)"

clean:
	rm -rf $(BINDIR) sdk/go sdk/nodejs sdk/python
