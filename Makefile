.PHONY: build-cli build-controller helm-lint dev-ui test-go

GO_PKGS := ./cmd/... ./internal/... ./pkg/...

test-go:
	go test $(GO_PKGS)

build-cli:
	go build -o bin/obelisk ./cmd/obelisk

build-controller:
	go build -o bin/obelisk-controller ./cmd/obelisk-controller

helm-lint:
	helm lint deploy/helm/platform
	helm lint deploy/helm/services/postgres
	helm lint deploy/helm/services/redis
	helm lint deploy/helm/services/rustfs
	helm lint deploy/helm/services/mysql
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

dev: build-controller
	@echo "Run in two terminals: make dev-api && make dev-ui"