DIST_DIR=dist
DIST_TARGETS=$(addprefix dist/,$(BINARIES))
UNAME := $(shell uname -m)

install-goreleaser: ## Install goreleaser
	@if ! [ -x "$$(command -v goreleaser)" ]; then \
		echo "goreleaser not found, installing..."; \
		$(GOCMD) install github.com/goreleaser/goreleaser/v2@latest; \
	fi

goreleaser-check: install-goreleaser ## Check gorelease configuration is correct
	@goreleaser check	

$(DIST_DIR): install-goreleaser goreleaser-check ## Build binaries for goreleaser
	@goreleaser build --snapshot --clean

$(DIST_TARGETS): install-goreleaser goreleaser-check ## Build specific binaries for goreleaser
	@goreleaser build --snapshot --clean --id $(@F)

GO_KUBE_RELEASE_BINARIES = $(foreach binary,$(BINARIES),$(HOME)/gokube/$(binary))

$(HOME)/gokube: ## Create gokube directory
	@if [ ! -d $(HOME)/gokube ]; then mkdir -p $(HOME)/gokube; fi

$(GO_KUBE_RELEASE_BINARIES): $(HOME)/gokube ## Copy binaries to gokube
	@echo $(@F) $(basename $(@F))
	@if [ "$(UNAME)" == "x86_64" ]; then \
		cp $(DIST_DIR)/$(@F)_linux_amd64_v1/$(@F) $(HOME)/gokube/; \
		printf "Copied linux amd64 binary to $(HOME)/gokube\n"; \
	else \
		cp $(DIST_DIR)/$(@F)_linux_arm64_v8.0/$(@F) $(HOME)/gokube/; \
		printf "Copied linux arm64 binary to $(HOME)/gokube\n"; \
	fi

install-dist: dist $(GO_KUBE_RELEASE_BINARIES) ## Create distributions and copy to gokube directory

# Lima commands for VMs
LIMA_VMS = master worker1 worker2
LIMA_INIT_TARGETS = $(addprefix init/,$(LIMA_VMS))
LIMA_CREATE_TARGETS = $(addprefix create/,$(LIMA_VMS))
LIMA_START_TARGETS = $(addprefix start/,$(LIMA_VMS))
LIMA_STOP_TARGETS = $(addprefix stop/,$(LIMA_VMS))
LIMA_DELETE_TARGETS = $(addprefix delete/,$(LIMA_VMS))
LIMA_SHELL_TARGETS = $(addprefix shell/,$(LIMA_VMS))

$(LIMA_CREATE_TARGETS): $(GO_KUBE_RELEASE_BINARIES) ## Create Lima VM
	@limactl start --name=$(@F) workbench/debian-12.yaml --log-level error --tty=false
	@printf "Lima instance '$(@F)' created and started\n"

$(LIMA_START_TARGETS): ## Start Lima VM
	@limactl start --name=$(@F) --log-level error --tty=false
	@printf "Lima instance '$(@F)' started\n"

$(LIMA_STOP_TARGETS): ## Stop Lima VM
	@limactl stop -f --log-level error $(@F)
	@printf "Lima instance '$(@F)' stopped\n"

$(LIMA_DELETE_TARGETS): ## Delete Lima VM
	@limactl delete --log-level error $(@F)
	@printf "Lima instance '$(@F)' deleted\n"

$(LIMA_SHELL_TARGETS): ## Go to shell of Lima VM
	@printf "Entering Lima instance '$(@F)' shell\n"
	@limactl shell --workdir $(HOME) --log-level error $(@F)

lima/install: ## Install Lima
	brew install lima

lima/init-vms: $(LIMA_CREATE_TARGETS) $(LIMA_STOP_TARGETS) ## Init Lima VMs

lima/start-vms: $(LIMA_START_TARGETS) ## Start all Lima VMs

lima/run: ### Run the project
	@process-compose -f process-compose-lima.yml up

lima/clean: $(LIMA_STOP_TARGETS) $(LIMA_DELETE_TARGETS) # Cleanup all lima vms