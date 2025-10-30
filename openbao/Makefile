# OpenBao development & integration Makefile (local mTLS test harness)
# Located in openbao/ so you can run: (cd openbao && make bao-test-all)
# Targets:
#   make build            - build the openbao-tool binary
#   make bao-cert-gen     - generate CA/server/client certs (SAN: localhost, 127.0.0.1)
#   make bao-server       - start OpenBao server with config (foreground)
#   make bao-init         - initialize & unseal (single share/threshold) using init-unseal.sh
#   make bao-health       - run health op via openbao-tool with mTLS
#   make bao-test-all     - automated full workflow (certs -> server -> init -> health)
#   make bao-stop         - stop background server started by bao-test-all
#   make bao-clean        - remove certs, data, init output
# Environment overrides (example): OPENBAO_ADDR=https://127.0.0.1:8301 make bao-test-all

# ---------------------------------------------------------------------------
# Variables (override with environment)
# ---------------------------------------------------------------------------
OPENBAO_ADDR            ?= https://127.0.0.1:8200
OPENBAO_CONFIG          ?= openbao-dev.hcl
OPENBAO_DATA_DIR        ?= openbao-data
CERT_DIR                ?= certs
OPENBAO_INIT_SHARES     ?= 1
OPENBAO_INIT_THRESHOLD  ?= 1
OPENBAO_TOOL_BIN        ?= openbao-tool
OPENBAO_TOKEN_FILE      ?= openbao-init-output.json
SERVER_PID_FILE         ?= .openbao-server.pid
CURL_SILENT             ?= -s

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
.PHONY: build
build:
	@echo "[build] Building $(OPENBAO_TOOL_BIN)";
	go build -o $(OPENBAO_TOOL_BIN) .

# ---------------------------------------------------------------------------
# Certificate generation (CA + server + client)
# ---------------------------------------------------------------------------
.PHONY: bao-cert-gen
bao-cert-gen:
	@echo "[cert-gen] Generating certificates in $(CERT_DIR)"
	@mkdir -p $(CERT_DIR)
	@openssl req -x509 -new -nodes -days 365 -subj "/CN=OpenBaoDevCA" -keyout $(CERT_DIR)/ca.key -out $(CERT_DIR)/ca.pem
	@printf '%s\n' '[ req ]' 'default_bits = 2048' 'prompt = no' 'default_md = sha256' 'req_extensions = req_ext' 'distinguished_name = dn' '[ dn ]' 'CN = 127.0.0.1' '[ req_ext ]' 'subjectAltName = @alt_names' '[ alt_names ]' 'DNS.1 = localhost' 'IP.1  = 127.0.0.1' > $(CERT_DIR)/server.cnf
	@openssl req -new -nodes -keyout $(CERT_DIR)/server.key -out $(CERT_DIR)/server.csr -config $(CERT_DIR)/server.cnf
	@openssl x509 -req -in $(CERT_DIR)/server.csr -CA $(CERT_DIR)/ca.pem -CAkey $(CERT_DIR)/ca.key -CAcreateserial -days 365 -out $(CERT_DIR)/server.crt -extensions req_ext -extfile $(CERT_DIR)/server.cnf
	@openssl req -new -nodes -subj "/CN=dev-client" -keyout $(CERT_DIR)/client.key -out $(CERT_DIR)/client.csr
	@openssl x509 -req -in $(CERT_DIR)/client.csr -CA $(CERT_DIR)/ca.pem -CAkey $(CERT_DIR)/ca.key -days 365 -out $(CERT_DIR)/client.crt
	@openssl verify -CAfile $(CERT_DIR)/ca.pem $(CERT_DIR)/server.crt $(CERT_DIR)/client.crt
	@echo "[cert-gen] Complete"

# ---------------------------------------------------------------------------
# Server start (foreground or background via test-all)
# ---------------------------------------------------------------------------
.PHONY: bao-server
bao-server: $(OPENBAO_CONFIG)
	@echo "[server] Starting OpenBao with config $(OPENBAO_CONFIG)";
	mkdir -p $(OPENBAO_DATA_DIR);
	bao server -config=$(OPENBAO_CONFIG)

# ---------------------------------------------------------------------------
# Init & Unseal
# ---------------------------------------------------------------------------
.PHONY: bao-init
bao-init: init-unseal.sh
	@echo "[init] Initializing & unsealing (shares=$(OPENBAO_INIT_SHARES) threshold=$(OPENBAO_INIT_THRESHOLD))";
	OPENBAO_ADDR=$(OPENBAO_ADDR) OPENBAO_INIT_SHARES=$(OPENBAO_INIT_SHARES) OPENBAO_INIT_THRESHOLD=$(OPENBAO_INIT_THRESHOLD) OPENBAO_CA_CERT=$(CERT_DIR)/ca.pem OPENBAO_CLIENT_CERT=$(CERT_DIR)/client.crt OPENBAO_CLIENT_KEY=$(CERT_DIR)/client.key ./init-unseal.sh;
	@echo "[init] Output saved to $(OPENBAO_TOKEN_FILE)"

# ---------------------------------------------------------------------------
# Health check via tool
# ---------------------------------------------------------------------------
.PHONY: bao-health
bao-health:
	@echo "[health] Checking /v1/sys/health";
	@TOKEN=$$(jq -r '.root_token // empty' $(OPENBAO_TOKEN_FILE) 2>/dev/null); ./$(OPENBAO_TOOL_BIN) --op=health --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$${TOKEN} || { echo "[health] FAILED"; exit 1; };
	@echo "[health] OK"

# ---------------------------------------------------------------------------
# Direct curl health check (mTLS). Demonstrates proper client cert usage.
# ---------------------------------------------------------------------------
.PHONY: bao-curl-health
bao-curl-health:
	@echo "[curl-health] Using client certificate to query /v1/sys/health";
	@TOKEN=$$(jq -r '.root_token // empty' $(OPENBAO_TOKEN_FILE) 2>/dev/null); \
	if [ -z "$$TOKEN" ]; then echo "[curl-health] WARNING: no token (server may return 200 if health unauthenticated allowed, else 403)"; fi; \
	curl -s --fail --cert $(CERT_DIR)/client.crt --key $(CERT_DIR)/client.key --cacert $(CERT_DIR)/ca.pem -H "X-Vault-Token: $$TOKEN" $(OPENBAO_ADDR)/v1/sys/health | jq '.' || { echo "[curl-health] FAILED"; exit 1; }; \
	echo "[curl-health] OK"

# ---------------------------------------------------------------------------
# Full workflow (background server)
# ---------------------------------------------------------------------------
.PHONY: bao-test-all
bao-test-all: build bao-cert-gen
	@echo "[test-all] Starting server with automatic cleanup (trap)";
	@set -e; \
	  make -s bao-stop >/dev/null 2>&1 || true; \
	  mkdir -p $(OPENBAO_DATA_DIR); \
	  ( bao server -config=$(OPENBAO_CONFIG) > .openbao-server.log 2>&1 & echo $$! > $(SERVER_PID_FILE) ); \
	  trap 'STATUS=$$?; echo "[test-all] Cleaning up (status=$$STATUS)"; make -s bao-stop; if [ "$$STATUS" -eq 0 ]; then echo "[test-all] Removing data/certs/token/log/binary"; rm -rf $(OPENBAO_DATA_DIR) $(CERT_DIR) $(OPENBAO_TOKEN_FILE) .openbao-server.log $(OPENBAO_TOOL_BIN); fi; exit $$STATUS' EXIT INT TERM; \
	  echo "[test-all] Waiting for server TLS handshake (health endpoint)"; \
	  for i in $$(seq 1 30); do \
	    if curl -s --cert $(CERT_DIR)/client.crt --key $(CERT_DIR)/client.key --cacert $(CERT_DIR)/ca.pem $(OPENBAO_ADDR)/v1/sys/health >/dev/null 2>&1; then break; fi; \
	    sleep 1; \
	  done; \
	  echo "[test-all] Running init & unseal"; \
	  make -s bao-init; \
	  echo "[test-all] Tool health"; \
	  make -s bao-health; \
	  echo "[test-all] Curl health"; \
	  make -s bao-curl-health; \
	  echo "[test-all] Namespace + KV secret lifecycle"; \
	  TOKEN=$$(jq -r '.root_token' $(OPENBAO_TOKEN_FILE)); \
	  NS=temp-$$RANDOM; echo "[test-all] Namespace=$$NS"; \
	  ./$(OPENBAO_TOOL_BIN) --op=create-namespace --namespace=$$NS --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	  ./$(OPENBAO_TOOL_BIN) --op=enable-kv --namespace=$$NS --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	  ./$(OPENBAO_TOOL_BIN) --op=put-secret --namespace=$$NS --secret-name=demo --secret-data='alpha=one,beta=two' --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	  SECRET_JSON=$$(./$(OPENBAO_TOOL_BIN) --op=read-secret --namespace=$$NS --secret-name=demo --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN); echo $$SECRET_JSON | jq '.'; \
	  echo $$SECRET_JSON | jq -e '.alpha == "one" and .beta == "two"' >/dev/null || { echo "[test-all] Secret validation failed"; exit 1; }; \
	  ./$(OPENBAO_TOOL_BIN) --op=delete-secret --namespace=$$NS --secret-name=demo --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	  ./$(OPENBAO_TOOL_BIN) --op=delete-namespace --namespace=$$NS --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	  echo "[test-all] Namespace + secret lifecycle OK"; \
	  echo "[test-all] SUCCESS";

# ---------------------------------------------------------------------------
# Namespace + KV secret CRUD test (creates temp namespace, secret, verifies lifecycle)
# ---------------------------------------------------------------------------
.PHONY: bao-ns-secrets-test
bao-ns-secrets-test: build bao-cert-gen
	@echo "[ns-secrets-test] Starting server"; \
	set -e; \
	make -s bao-stop >/dev/null 2>&1 || true; \
	mkdir -p $(OPENBAO_DATA_DIR); \
	( bao server -config=$(OPENBAO_CONFIG) > .openbao-server.log 2>&1 & echo $$! > $(SERVER_PID_FILE) ); \
	trap 'STATUS=$$?; echo "[ns-secrets-test] Cleanup (status=$$STATUS)"; make -s bao-stop; if [ "$$STATUS" -eq 0 ]; then echo "[ns-secrets-test] Removing data/certs/token/log/binary"; rm -rf $(OPENBAO_DATA_DIR) $(CERT_DIR) $(OPENBAO_TOKEN_FILE) .openbao-server.log $(OPENBAO_TOOL_BIN); else rm -rf $(OPENBAO_DATA_DIR); fi; exit $$STATUS' EXIT INT TERM; \
	echo "[ns-secrets-test] Waiting for server readiness"; \
	for i in $$(seq 1 30); do \
	  if curl -s --cert $(CERT_DIR)/client.crt --key $(CERT_DIR)/client.key --cacert $(CERT_DIR)/ca.pem $(OPENBAO_ADDR)/v1/sys/init >/dev/null 2>&1; then break; fi; \
	  sleep 1; \
	done; \
	make -s bao-init; \
	NS=temp-$$RANDOM; echo "[ns-secrets-test] Namespace=$$NS"; \
	TOKEN=$$(jq -r '.root_token' $(OPENBAO_TOKEN_FILE)); \
	./$(OPENBAO_TOOL_BIN) --op=create-namespace --namespace=$$NS --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	./$(OPENBAO_TOOL_BIN) --op=enable-kv --namespace=$$NS --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	./$(OPENBAO_TOOL_BIN) --op=put-secret --namespace=$$NS --secret-name=demo --secret-data='foo=bar,baz=qux' --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	SECRET_JSON=$$(./$(OPENBAO_TOOL_BIN) --op=read-secret --namespace=$$NS --secret-name=demo --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN); echo $$SECRET_JSON | jq '.'; \
	echo $$SECRET_JSON | jq -e '.foo == "bar" and .baz == "qux"' >/dev/null || { echo "[ns-secrets-test] Secret data mismatch"; exit 1; }; \
	./$(OPENBAO_TOOL_BIN) --op=delete-secret --namespace=$$NS --secret-name=demo --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	./$(OPENBAO_TOOL_BIN) --op=delete-namespace --namespace=$$NS --addr=$(OPENBAO_ADDR) --ca=$(CERT_DIR)/ca.pem --cert=$(CERT_DIR)/client.crt --key=$(CERT_DIR)/client.key --token=$$TOKEN; \
	echo "[ns-secrets-test] SUCCESS";
# Stop background server
# ---------------------------------------------------------------------------
.PHONY: bao-stop
bao-stop:
	@if [ -f $(SERVER_PID_FILE) ]; then \
	  PID=$$(cat $(SERVER_PID_FILE)); \
	  echo "[stop] Stopping server PID $$PID"; \
	  kill $$PID || true; \
	  rm -f $(SERVER_PID_FILE); \
	else \
	  echo "[stop] No PID file"; \
	fi

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------
.PHONY: bao-clean
bao-clean: bao-stop
	@echo "[clean] Removing certs/data/output";
	@rm -rf $(CERT_DIR) $(OPENBAO_DATA_DIR) $(OPENBAO_TOKEN_FILE) $(CERT_DIR)/*.srl server.cnf *.srl 2>/dev/null || true;
	@echo "[clean] Done"

# Convenience alias
.PHONY: test
test: bao-test-all

# ---------------------------------------------------------------------------
# Kubernetes REST API test via port-forward
# Requires: kubectl context pointing at cluster with deployment & service from deploy.yaml
# Performs: port-forward, then namespace + transit key lifecycle through REST endpoints
# ---------------------------------------------------------------------------
.PHONY: bao-k8s-rest-test
bao-k8s-rest-test:
	@echo "[k8s-rest-test] Starting port-forward to service/openbao-ns-key-manager";
	@set -e; \
	  NS=openbao; SVC=openbao-ns-key-manager; LOCAL_PORT=$$(bash -c 'for p in $$(seq 18080 18120); do lsof -i :$$p >/dev/null 2>&1 || { echo $$p; break; }; done'); \
	  if [ -z "$$LOCAL_PORT" ]; then echo "[k8s-rest-test] No free local port in range"; exit 1; fi; \
	  echo "[k8s-rest-test] Using local port $$LOCAL_PORT"; \
	  ( kubectl -n $$NS port-forward svc/$$SVC $$LOCAL_PORT:80 > .pf.log 2>&1 & echo $$! > .pf.pid ); \
	  trap 'STATUS=$$?; echo "[k8s-rest-test] Cleanup (status=$$STATUS)"; if [ -f .pf.pid ]; then kill $$(cat .pf.pid) 2>/dev/null || true; rm -f .pf.pid .pf.log; fi; exit $$STATUS' EXIT INT TERM; \
	  echo "[k8s-rest-test] Waiting for /health"; \
	  for i in $$(seq 1 40); do curl -s http://127.0.0.1:$$LOCAL_PORT/health >/dev/null 2>&1 && break; sleep 1; done; \
	  curl -s --fail http://127.0.0.1:$$LOCAL_PORT/health | jq -e '.status == "ok"' >/dev/null || { echo "[k8s-rest-test] Health failed"; exit 1; }; \
	  NS_NAME=temp-$$RANDOM; echo "[k8s-rest-test] Namespace $$NS_NAME"; \
	  echo '{"name":"'$$NS_NAME'"}' | curl -s --fail -X POST -H 'Content-Type: application/json' --data @- http://127.0.0.1:$$LOCAL_PORT/namespaces | jq '.'; \
	  echo "[k8s-rest-test] Ensuring transit key example"; \
	  curl -s -X POST -H 'Content-Type: application/json' -d '{"name":"example","type":"aes256-gcm96"}' http://127.0.0.1:$$LOCAL_PORT/ensure-key-all-namespaces | jq '.'; \
	  echo "[k8s-rest-test] Creating key in namespace"; \
	  curl -s --fail -X POST -H 'Content-Type: application/json' -d '{"namespace":"'$$NS_NAME'","name":"demo","type":"aes256-gcm96"}' http://127.0.0.1:$$LOCAL_PORT/keys | jq '.'; \
	  echo "[k8s-rest-test] Listing keys in namespace"; \
	  curl -s --fail 'http://127.0.0.1:'$$LOCAL_PORT'/keys?namespace='$$NS_NAME | jq '.'; \
	  echo "[k8s-rest-test] Listing keys across all namespaces"; \
	  curl -s --fail http://127.0.0.1:$$LOCAL_PORT/keys | jq '.'; \
	  echo "[k8s-rest-test] Fetching key metadata"; \
	  curl -s --fail http://127.0.0.1:$$LOCAL_PORT/keys/$$NS_NAME/demo | jq '.'; \
	  echo "[k8s-rest-test] Rotating key"; \
	  curl -s --fail -X POST http://127.0.0.1:$$LOCAL_PORT/keys/$$NS_NAME/demo/rotate | jq '.'; \
	  echo "[k8s-rest-test] Deleting key"; \
	  curl -s --fail -X DELETE http://127.0.0.1:$$LOCAL_PORT/keys/$$NS_NAME/demo | jq '.'; \
	  echo "[k8s-rest-test] Deleting namespace"; \
	  curl -s --fail -X DELETE http://127.0.0.1:$$LOCAL_PORT/namespaces/$$NS_NAME | jq '.'; \
	  echo "[k8s-rest-test] SUCCESS";
