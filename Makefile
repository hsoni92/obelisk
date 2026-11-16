.PHONY: build-cli build-controller helm-lint dev-ui test-go dev dev-cluster install-air check-go test-vm vm-up vm-install vm-smoke vm-destroy

GO ?= go
export PATH := $(shell $(GO) env GOPATH 2>/dev/null)/bin:$(PATH)

GO_PKGS := ./cmd/... ./internal/... ./pkg/...

check-go:
	@chmod +x install/check-go.sh
	@./install/check-go.sh

test-go: check-go
	$(GO) test $(GO_PKGS)

build-cli: check-go
	$(GO) build -o bin/obelisk ./cmd/obelisk

build-controller: check-go
	$(GO) build -o bin/obelisk-controller ./cmd/obelisk-controller

helm-lint:
	helm lint deploy/helm/platform
	helm lint deploy/helm/services/postgres
	helm lint deploy/helm/services/redis
	helm lint deploy/helm/services/rustfs
	helm lint deploy/helm/services/mysql
	helm lint deploy/helm/services/mongodb
	helm lint deploy/helm/services/qdrant
	helm lint deploy/helm/services/searxng

dev-ui:
	cd web/admin && npm run dev

dev-k3d-up:
	chmod +x install/k3d-dev.sh
	./install/k3d-dev.sh

dev-k3d-down:
	k3d cluster delete obelisk || true

# :8081 is often taken on dev machines (e.g. other Docker stacks); override with OBELISK_GATEWAY_LISTEN.
GATEWAY_LISTEN ?= :18081
MAIL_GATEWAY_LISTEN ?= :18082

dev-api: build-controller
	test -f "$(CURDIR)/dev/kubeconfig" || $(MAKE) dev-k3d-up
	KUBECONFIG="$${KUBECONFIG:-$(CURDIR)/dev/kubeconfig}" \
	OBELISK_CONFIG="$${OBELISK_CONFIG:-$(CURDIR)/dev/obelisk.yaml}" \
	OBELISK_ROOT="$(CURDIR)" \
	OBELISK_RECONCILE=cluster \
	OBELISK_CREDENTIALS_DIR="$${OBELISK_CREDENTIALS_DIR:-$(CURDIR)/dev/credentials}" \
	./bin/obelisk-controller -listen :8080 -gateway-listen $(GATEWAY_LISTEN) -mail-gateway-listen $(MAIL_GATEWAY_LISTEN)

install-air: check-go
	$(GO) install github.com/air-verse/air@latest
	@echo "air installed to $$($(GO) env GOPATH)/bin — ensure that directory is on your PATH"

dev: check-go
	chmod +x install/k3d-dev.sh scripts/dev.sh scripts/run-controller-dev.sh
	./scripts/dev.sh

dev-cluster: check-go
	chmod +x install/k3d-dev.sh scripts/dev.sh scripts/run-controller-dev.sh
	OBELISK_DEV_CLUSTER=1 ./scripts/dev.sh --cluster

test-vm:
	chmod +x tests/integration/multipass/harness.sh tests/integration/vm/*.sh install/build-images.sh
	./tests/integration/multipass/harness.sh all

vm-up:
	chmod +x tests/integration/multipass/harness.sh
	./tests/integration/multipass/harness.sh up

vm-install:
	chmod +x tests/integration/multipass/harness.sh tests/integration/vm/*.sh
	./tests/integration/multipass/harness.sh install

vm-smoke:
	chmod +x tests/integration/multipass/harness.sh tests/integration/vm/*.sh
	./tests/integration/multipass/harness.sh smoke

vm-destroy:
	./tests/integration/multipass/harness.sh destroy