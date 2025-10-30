package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	baoapi "github.com/openbao/openbao/api/v2"
)

// Simple example demonstrating creating an OpenBao API client with mutual TLS authentication.
// It expects environment variables or flags pointing to CA, client cert, and key files.
func main() {
	// Connection / security flags
	addr := flag.String("addr", envDefault("OPENBAO_ADDR", "https://localhost:8200"), "Base URL of the OpenBao server (include scheme, e.g. https://host:8200)")
	caPath := flag.String("ca", envDefault("OPENBAO_CA_CERT", "/certs/ca.pem"), "Path to CA certificate file")
	certPath := flag.String("cert", envDefault("OPENBAO_CLIENT_CERT", "/certs/client.crt"), "Path to client certificate file")
	keyPath := flag.String("key", envDefault("OPENBAO_CLIENT_KEY", "/certs/client.key"), "Path to client private key file")

	// Operation flags
	op := flag.String("op", envDefault("OPENBAO_OPERATION", "health"), "Operation: health | create-namespace | delete-namespace | list-namespaces | list-keys | create-key | delete-key | rotate-key | ensure-key-all-namespaces")
	namespace := flag.String("namespace", envDefault("OPENBAO_NAMESPACE", ""), "Target namespace for key operations (omit for all namespaces when supported)")
	keyName := flag.String("key-name", envDefault("OPENBAO_KEY_NAME", "example"), "Transit key name for key operations")
	transitPath := flag.String("transit", envDefault("OPENBAO_TRANSIT_PATH", "transit"), "Path where transit engine is mounted")
	keyType := flag.String("key-type", envDefault("OPENBAO_KEY_TYPE", "aes256-gcm96"), "Transit key type when creating keys")
	token := flag.String("token", envDefault("OPENBAO_TOKEN", ""), "Bao root or client token for authenticated requests")
	// Secret (KV) operations flags
	secretMount := flag.String("secret-mount", envDefault("OPENBAO_SECRET_MOUNT", "secret"), "KV secret engine mount path (v2)")
	secretName := flag.String("secret-name", envDefault("OPENBAO_SECRET_NAME", "demo"), "Secret name for KV operations")
	secretData := flag.String("secret-data", envDefault("OPENBAO_SECRET_DATA", "foo=bar"), "Secret data for create/update in key=value[,key=value] format")
	insecure := flag.Bool("insecure", envDefault("OPENBAO_INSECURE", "") != "", "If set (or OPENBAO_INSECURE is present) skip TLS certificate verification. DEV ONLY!")
	serverMode := flag.Bool("server", envDefault("OPENBAO_SERVER_MODE", "") != "", "Run as REST API server if true or OPENBAO_SERVER_MODE is set")
	listenAddr := flag.String("listen", envDefault("OPENBAO_LISTEN", ":8080"), "Listen address for REST server (e.g. :8080)")
	flag.Parse()

	tlsConfig, err := buildTLSConfig(*caPath, *certPath, *keyPath)
	if err != nil {
		log.Fatalf("failed to build TLS config: %v", err)
	}
	if *insecure {
		log.Printf("WARNING: insecure TLS mode enabled - certificate verification is skipped")
		tlsConfig.InsecureSkipVerify = true //nolint:gosec
	}

	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	cfg := baoapi.DefaultConfig()
	cfg.Address = *addr
	cfg.HttpClient = httpClient
	client, err := baoapi.NewClient(cfg)
	if err != nil {
		log.Fatalf("failed to create OpenBao client: %v", err)
	}
	if *token != "" {
		client.SetToken(*token)
	}

	// Add a simple timeout context via http client if desired (not changing library internals)
	httpClient.Timeout = 30 * time.Second

	if *serverMode {
		log.Printf("Starting REST API server on %s (Bao addr=%s)", *listenAddr, *addr)
		app := &apiServer{baoAddr: *addr, transitPath: *transitPath, defaultKeyType: *keyType, tlsConfig: tlsConfig, httpClient: httpClient, token: *token}
		if err := app.serve(*listenAddr); err != nil {
			log.Fatalf("server error: %v", err)
		}
		return
	}

	// CLI single-operation mode
	switch *op {
	case "health":
		if err := doHealth(client); err != nil {
			log.Fatalf("health request failed: %v", err)
		}
	case "create-namespace":
		mustNamespace(*namespace)
		if err := createNamespace(client, *namespace); err != nil {
			log.Fatalf("create namespace: %v", err)
		}
		log.Printf("namespace %q created", *namespace)
	case "delete-namespace":
		mustNamespace(*namespace)
		if err := deleteNamespace(client, *namespace); err != nil {
			log.Fatalf("delete namespace: %v", err)
		}
		log.Printf("namespace %q deleted", *namespace)
	case "list-namespaces":
		nss, err := listNamespaces(client)
		if err != nil {
			log.Fatalf("list namespaces: %v", err)
		}
		for _, ns := range nss {
			fmt.Println(ns)
		}
	case "list-keys":
		if *namespace != "" {
			client.SetNamespace(*namespace)
			keys, err := listTransitKeys(client, *transitPath)
			if err != nil {
				log.Fatalf("list keys in namespace %q: %v", *namespace, err)
			}
			for _, k := range keys {
				fmt.Printf("%s/%s\n", *namespace, k)
			}
		} else {
			nss, err := listNamespaces(client)
			if err != nil {
				log.Fatalf("list namespaces: %v", err)
			}
			for _, ns := range nss {
				client.SetNamespace(ns)
				keys, err := listTransitKeys(client, *transitPath)
				if err != nil {
					log.Printf("warn: list keys in %s failed: %v", ns, err)
					continue
				}
				for _, k := range keys {
					fmt.Printf("%s/%s\n", ns, k)
				}
			}
		}
	case "create-key":
		mustNamespace(*namespace)
		client.SetNamespace(*namespace)
		if err := createTransitKey(client, *transitPath, *keyName, *keyType); err != nil {
			log.Fatalf("create key: %v", err)
		}
		log.Printf("key %q created in namespace %q", *keyName, *namespace)
	case "delete-key":
		mustNamespace(*namespace)
		client.SetNamespace(*namespace)
		if err := deleteTransitKey(client, *transitPath, *keyName); err != nil {
			log.Fatalf("delete key: %v", err)
		}
		log.Printf("key %q deleted in namespace %q", *keyName, *namespace)
	case "rotate-key":
		mustNamespace(*namespace)
		client.SetNamespace(*namespace)
		if err := rotateTransitKey(client, *transitPath, *keyName); err != nil {
			log.Fatalf("rotate key: %v", err)
		}
		log.Printf("key %q rotated in namespace %q", *keyName, *namespace)
	case "ensure-key-all-namespaces":
		nss, err := listNamespaces(client)
		if err != nil {
			log.Fatalf("list namespaces: %v", err)
		}
		for _, ns := range nss {
			client.SetNamespace(ns)
			if err := ensureTransitKey(client, *transitPath, *keyName, *keyType); err != nil {
				log.Printf("namespace %s: ensure key failed: %v", ns, err)
			} else {
				log.Printf("namespace %s: key %s ensured", ns, *keyName)
			}
		}
	case "enable-kv":
		mustNamespace(*namespace)
		client.SetNamespace(*namespace)
		if err := enableKVIfNeeded(client, *secretMount); err != nil {
			log.Fatalf("enable kv: %v", err)
		}
		log.Printf("kv engine enabled at %s in namespace %s", *secretMount, *namespace)
	case "put-secret":
		mustNamespace(*namespace)
		client.SetNamespace(*namespace)
		dataMap := parseKVPairs(*secretData)
		if err := putKVSecret(client, *secretMount, *secretName, dataMap); err != nil {
			log.Fatalf("put secret: %v", err)
		}
		log.Printf("secret %s written (%d fields) in namespace %s", *secretName, len(dataMap), *namespace)
	case "read-secret":
		mustNamespace(*namespace)
		client.SetNamespace(*namespace)
		m, err := getKVSecret(client, *secretMount, *secretName)
		if err != nil {
			log.Fatalf("read secret: %v", err)
		}
		enc, _ := json.MarshalIndent(m, "", "  ")
		fmt.Println(string(enc))
	case "delete-secret":
		mustNamespace(*namespace)
		client.SetNamespace(*namespace)
		if err := deleteKVSecret(client, *secretMount, *secretName); err != nil {
			log.Fatalf("delete secret: %v", err)
		}
		log.Printf("secret %s deleted in namespace %s", *secretName, *namespace)
	default:
		log.Fatalf("unknown operation: %s", *op)
	}
}

// Health check helper
func doHealth(client *baoapi.Client) error {
	req := client.NewRequest("GET", "/v1/sys/health")
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	fmt.Printf("StatusCode: %d\nBody: %s\n", resp.StatusCode, string(b))
	return nil
}

// Namespace operations
func createNamespace(client *baoapi.Client, name string) error {
	req := client.NewRequest("POST", "/v1/sys/namespaces/"+name)
	req.Body = bytes.NewBufferString("{}")
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError("create namespace", resp)
	}
	return nil
}

func deleteNamespace(client *baoapi.Client, name string) error {
	req := client.NewRequest("DELETE", "/v1/sys/namespaces/"+name)
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError("delete namespace", resp)
	}
	return nil
}

func listNamespaces(client *baoapi.Client) ([]string, error) {
	// Properly set list=true via query params (avoid encoding '?' in path)
	req := client.NewRequest("GET", "/v1/sys/namespaces")
	if req.Params != nil { // Vault/OpenBao client sets this map
		req.Params.Set("list", "true")
	}
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		// Treat unsupported path / enterprise-only errors as single root namespace
		if strings.Contains(err.Error(), "unsupported path") || strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "permission denied") {
			return []string{""}, nil
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 || resp.StatusCode == 405 { // unsupported operation
		return []string{""}, nil
	}
	if resp.StatusCode >= 300 {
		return nil, statusError("list namespaces", resp)
	}
	var raw map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	var out []string
	if data, ok := raw["data"].(map[string]any); ok {
		if keys, ok := data["keys"].([]any); ok {
			for _, k := range keys {
				if s, ok := k.(string); ok {
					out = append(out, strings.TrimSuffix(s, "/"))
				}
			}
		}
		if ns, ok := data["namespaces"].([]any); ok {
			for _, k := range ns {
				if s, ok := k.(string); ok {
					out = append(out, strings.TrimSuffix(s, "/"))
				}
			}
		}
	}
	return out, nil
}

// Transit key operations
func listTransitKeys(client *baoapi.Client, transitPath string) ([]string, error) {
	// Ensure transit engine is mounted before listing
	_ = ensureTransitMounted(client, transitPath)
	req := client.NewRequest("GET", fmt.Sprintf("/v1/%s/keys", strings.TrimPrefix(transitPath, "/")))
	if req.Params != nil {
		req.Params.Set("list", "true")
	}
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		// For missing mount or unsupported path, return empty list
		if strings.Contains(err.Error(), "unsupported path") || strings.Contains(err.Error(), "404") {
			return []string{}, nil
		}
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return []string{}, nil
	}
	if resp.StatusCode >= 300 {
		return nil, statusError("list keys", resp)
	}
	var raw map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	var out []string
	if data, ok := raw["data"].(map[string]any); ok {
		if keys, ok := data["keys"].([]any); ok {
			for _, k := range keys {
				if s, ok := k.(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	return out, nil
}

func createTransitKey(client *baoapi.Client, transitPath, keyName, keyType string) error {
	if err := ensureTransitMounted(client, transitPath); err != nil {
		return err
	}
	req := client.NewRequest("POST", fmt.Sprintf("/v1/%s/keys/%s", strings.TrimPrefix(transitPath, "/"), keyName))
	// Include deletion_allowed in create body (if server ignores here we'll set via config endpoint next)
	body, _ := json.Marshal(map[string]any{"type": keyType, "deletion_allowed": true})
	req.Body = bytes.NewBuffer(body)
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError("create key", resp)
	}
	// Enable deletion of the key (transit requires deletion_allowed=true via config before deletion)
	if err := configureTransitKey(client, transitPath, keyName, map[string]any{"deletion_allowed": true}); err != nil {
		log.Printf("warn: set deletion_allowed for key %s failed: %v", keyName, err)
	}
	return nil
}

func ensureTransitKey(client *baoapi.Client, transitPath, keyName, keyType string) error {
	if err := ensureTransitMounted(client, transitPath); err != nil {
		return err
	}
	keys, err := listTransitKeys(client, transitPath)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if k == keyName {
			return nil
		}
	}
	return createTransitKey(client, transitPath, keyName, keyType)
}

func deleteTransitKey(client *baoapi.Client, transitPath, keyName string) error {
	if err := ensureTransitMounted(client, transitPath); err != nil { // delete will still give 404 if key not present
		return err
	}
	req := client.NewRequest("DELETE", fmt.Sprintf("/v1/%s/keys/%s", strings.TrimPrefix(transitPath, "/"), keyName))
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError("delete key", resp)
	}
	return nil
}

func rotateTransitKey(client *baoapi.Client, transitPath, keyName string) error {
	if err := ensureTransitMounted(client, transitPath); err != nil {
		return err
	}
	req := client.NewRequest("POST", fmt.Sprintf("/v1/%s/keys/%s/rotate", strings.TrimPrefix(transitPath, "/"), keyName))
	req.Body = bytes.NewBufferString("{}")
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError("rotate key", resp)
	}
	return nil
}

// configureTransitKey updates key configuration (e.g., delete_allowed) after creation.
func configureTransitKey(client *baoapi.Client, transitPath, keyName string, cfg map[string]any) error {
	if err := ensureTransitMounted(client, transitPath); err != nil {
		return err
	}
	req := client.NewRequest("POST", fmt.Sprintf("/v1/%s/keys/%s/config", strings.TrimPrefix(transitPath, "/"), keyName))
	body, _ := json.Marshal(cfg)
	req.Body = bytes.NewBuffer(body)
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError("configure key", resp)
	}
	return nil
}

// Get transit key metadata/info
func getTransitKeyInfo(client *baoapi.Client, transitPath, keyName string) (map[string]any, error) {
	if err := ensureTransitMounted(client, transitPath); err != nil {
		return nil, err
	}
	req := client.NewRequest("GET", fmt.Sprintf("/v1/%s/keys/%s", strings.TrimPrefix(transitPath, "/"), keyName))
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, statusError("get key info (not found)", resp)
	}
	if resp.StatusCode >= 300 {
		return nil, statusError("get key info", resp)
	}
	var raw map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	// Vault/OpenBao transit key read returns {data:{...meta...}}
	if data, ok := raw["data"].(map[string]any); ok {
		return data, nil
	}
	return raw, nil
}

// KV (v2) secret engine operations
func enableKVIfNeeded(client *baoapi.Client, mount string) error {
	// Check mounts
	req := client.NewRequest("GET", "/v1/sys/mounts")
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var mounts map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&mounts)
	if _, exists := mounts[strings.Trim(mount, "/")+"/"]; exists {
		return nil // already mounted
	}
	// Enable KV v2
	enableReq := client.NewRequest("POST", fmt.Sprintf("/v1/sys/mounts/%s", strings.TrimPrefix(mount, "/")))
	body, _ := json.Marshal(map[string]any{"type": "kv", "options": map[string]string{"version": "2"}})
	enableReq.Body = bytes.NewBuffer(body)
	enableResp, err := client.RawRequest(enableReq) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer enableResp.Body.Close()
	if enableResp.StatusCode >= 300 {
		return statusError("enable kv", enableResp)
	}
	return nil
}

func putKVSecret(client *baoapi.Client, mount, name string, data map[string]string) error {
	path := fmt.Sprintf("/v1/%s/data/%s", strings.TrimPrefix(mount, "/"), name)
	payload := map[string]any{"data": data}
	b, _ := json.Marshal(payload)
	req := client.NewRequest("POST", path)
	req.Body = bytes.NewBuffer(b)
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return statusError("put secret", resp)
	}
	return nil
}

func getKVSecret(client *baoapi.Client, mount, name string) (map[string]string, error) {
	path := fmt.Sprintf("/v1/%s/data/%s", strings.TrimPrefix(mount, "/"), name)
	req := client.NewRequest("GET", path)
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, statusError("get secret", resp)
	}
	var raw map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	out := map[string]string{}
	if data, ok := raw["data"].(map[string]any); ok {
		if inner, ok := data["data"].(map[string]any); ok {
			for k, v := range inner {
				if s, ok := v.(string); ok {
					out[k] = s
				}
			}
		}
	}
	return out, nil
}

func deleteKVSecret(client *baoapi.Client, mount, name string) error {
	// Delete metadata (KV v2)
	path := fmt.Sprintf("/v1/%s/metadata/%s", strings.TrimPrefix(mount, "/"), name)
	req := client.NewRequest("DELETE", path)
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != 404 { // 404 is fine (already deleted)
		return statusError("delete secret", resp)
	}
	return nil
}

// Parse key=value[,key=value] into map
func parseKVPairs(s string) map[string]string {
	m := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		m[kv[0]] = kv[1]
	}
	return m
}

// Helpers
func mustNamespace(ns string) {
	if ns == "" {
		log.Fatal("--namespace (or OPENBAO_NAMESPACE) is required for this operation")
	}
}

func statusError(action string, resp *baoapi.Response) error {
	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("%s: status=%d body=%s", action, resp.StatusCode, string(b))
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// buildTLSConfig loads CA cert, client cert and key and constructs a *tls.Config for mTLS.
func buildTLSConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	caData, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to append CA cert to pool")
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load client cert/key: %w", err)
	}

	return &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// --- REST API server implementation ---
type apiServer struct {
	baoAddr        string
	transitPath    string
	defaultKeyType string
	tlsConfig      *tls.Config
	httpClient     *http.Client
	token          string
}

func (s *apiServer) newClient(namespace string) (*baoapi.Client, error) {
	cfg := baoapi.DefaultConfig()
	cfg.Address = s.baoAddr
	cfg.HttpClient = s.httpClient
	c, err := baoapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	if namespace != "" {
		c.SetNamespace(namespace)
	}
	if s.token != "" {
		c.SetToken(s.token)
	}
	return c, nil
}

func (s *apiServer) serve(listen string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/namespaces", s.handleNamespacesCollection)
	mux.HandleFunc("/namespaces/", s.handleNamespaceItem) // /namespaces/{name}
	mux.HandleFunc("/keys", s.handleKeysCollection)       // list or create
	mux.HandleFunc("/keys/", s.handleKeysItem)            // /keys/{namespace}/{name}[(/rotate|)]
	mux.HandleFunc("/ensure-key-all-namespaces", s.handleEnsureAllNamespaces)
	mux.HandleFunc("/secrets", s.handleSecretsCollection) // POST write, GET read (requires namespace & name)
	mux.HandleFunc("/secrets/", s.handleSecretsItem)      // DELETE /secrets/{namespace}/{name}

	srv := &http.Server{Addr: listen, Handler: logMiddleware(mux)}
	return srv.ListenAndServe()
}

// Handlers
func (s *apiServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	c, err := s.newClient("")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := doHealth(c); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *apiServer) handleNamespacesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c, err := s.newClient("")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		nss, err := listNamespaces(c)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"namespaces": nss})
	case http.MethodPost:
		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		if payload.Name == "" {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("name is required"))
			return
		}
		c, err := s.newClient("")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := createNamespace(c, payload.Name); err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"created": payload.Name})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *apiServer) handleNamespaceItem(w http.ResponseWriter, r *http.Request) {
	name := path.Base(r.URL.Path)
	if name == "namespaces" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		c, err := s.newClient("")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := deleteNamespace(c, name); err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *apiServer) handleKeysCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ns := r.URL.Query().Get("namespace")
		c, err := s.newClient(ns)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if ns == "" {
			// list keys across all namespaces
			baseClient, err := s.newClient("")
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err)
				return
			}
			nss, err := listNamespaces(baseClient)
			if err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			result := map[string][]string{}
			for _, n := range nss {
				nc, err := s.newClient(n)
				if err != nil {
					continue
				}
				keys, err := listTransitKeys(nc, s.transitPath)
				if err != nil {
					continue
				}
				result[n] = keys
			}
			writeJSON(w, http.StatusOK, result)
			return
		}
		keys, err := listTransitKeys(c, s.transitPath)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"namespace": ns, "keys": keys})
	case http.MethodPost:
		var payload struct {
			Namespace   string `json:"namespace"`
			Name        string `json:"name"`
			Type        string `json:"type"`
			AllowDelete *bool  `json:"allow_delete"` // maps to deletion_allowed
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		// Allow empty namespace to target root namespace
		if payload.Name == "" {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("name required"))
			return
		}
		if payload.Type == "" {
			payload.Type = s.defaultKeyType
		}
		c, err := s.newClient(payload.Namespace)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := createTransitKey(c, s.transitPath, payload.Name, payload.Type); err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		// Optionally override delete_allowed if provided
		if payload.AllowDelete != nil {
			if err := configureTransitKey(c, s.transitPath, payload.Name, map[string]any{"deletion_allowed": *payload.AllowDelete}); err != nil {
				log.Printf("warn: override deletion_allowed for %s failed: %v", payload.Name, err)
			}
		}
		writeJSON(w, http.StatusCreated, map[string]string{"created": payload.Name, "namespace": payload.Namespace})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// ensureTransitMounted enables the transit engine at the given path if not already mounted.
func ensureTransitMounted(client *baoapi.Client, mount string) error {
	if mount == "" {
		mount = "transit"
	}
	req := client.NewRequest("GET", "/v1/sys/mounts")
	resp, err := client.RawRequest(req) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var mounts map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&mounts)
	pathKey := strings.Trim(mount, "/") + "/"
	if _, ok := mounts[pathKey]; ok {
		return nil // already mounted
	}
	enableReq := client.NewRequest("POST", fmt.Sprintf("/v1/sys/mounts/%s", strings.TrimPrefix(mount, "/")))
	body, _ := json.Marshal(map[string]any{"type": "transit"})
	enableReq.Body = bytes.NewBuffer(body)
	enableResp, err := client.RawRequest(enableReq) //nolint:staticcheck
	if err != nil {
		return err
	}
	defer enableResp.Body.Close()
	if enableResp.StatusCode >= 300 {
		return statusError("enable transit", enableResp)
	}
	return nil
}

func (s *apiServer) handleKeysItem(w http.ResponseWriter, r *http.Request) {
	// Expect /keys/{namespace}/{name} or /keys/{namespace}/{name}/rotate
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/keys/"), "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ns, name := parts[0], parts[1]
	rotate := len(parts) == 3 && parts[2] == "rotate"
	c, err := s.newClient(ns)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if rotate {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := deleteTransitKey(c, s.transitPath, name); err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": name, "namespace": ns})
	case http.MethodPost:
		if rotate {
			if err := rotateTransitKey(c, s.transitPath, name); err != nil {
				writeErr(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"rotated": name, "namespace": ns})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case http.MethodGet:
		if rotate { // GET /rotate not supported
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		info, err := getTransitKeyInfo(c, s.transitPath, name)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"namespace": ns, "name": name, "info": info})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *apiServer) handleEnsureAllNamespaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}
	if payload.Name == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("name required"))
		return
	}
	if payload.Type == "" {
		payload.Type = s.defaultKeyType
	}
	baseClient, err := s.newClient("")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	nss, err := listNamespaces(baseClient)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	ensured := []string{}
	failed := map[string]string{}
	for _, ns := range nss {
		c, err := s.newClient(ns)
		if err != nil {
			failed[ns] = err.Error()
			continue
		}
		if err := ensureTransitKey(c, s.transitPath, payload.Name, payload.Type); err != nil {
			failed[ns] = err.Error()
			continue
		}
		ensured = append(ensured, ns)
	}
	writeJSON(w, http.StatusOK, map[string]any{"key": payload.Name, "ensured": ensured, "failed": failed})
}

// Secrets collection: for simplicity, GET requires namespace & name query params to read one secret.
// POST writes (upsert) a secret with JSON {namespace,name,data:{k:v}}
func (s *apiServer) handleSecretsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ns := r.URL.Query().Get("namespace")
		name := r.URL.Query().Get("name")
		mount := r.URL.Query().Get("mount")
		if mount == "" {
			mount = "secret"
		}
		if ns == "" || name == "" {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("namespace and name query params required"))
			return
		}
		c, err := s.newClient(ns)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		// ensure mount exists (best effort)
		_ = enableKVIfNeeded(c, mount)
		m, err := getKVSecret(c, mount, name)
		if err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"namespace": ns, "name": name, "data": m})
	case http.MethodPost:
		var payload struct {
			Namespace string            `json:"namespace"`
			Name      string            `json:"name"`
			Mount     string            `json:"mount"`
			Data      map[string]string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
		if payload.Namespace == "" || payload.Name == "" {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("namespace and name required"))
			return
		}
		if payload.Mount == "" {
			payload.Mount = "secret"
		}
		if payload.Data == nil {
			payload.Data = map[string]string{}
		}
		c, err := s.newClient(payload.Namespace)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if err := enableKVIfNeeded(c, payload.Mount); err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		if err := putKVSecret(c, payload.Mount, payload.Name, payload.Data); err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"written": payload.Name, "namespace": payload.Namespace, "mount": payload.Mount})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Secrets item: DELETE /secrets/{namespace}/{name}
func (s *apiServer) handleSecretsItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/secrets/"), "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ns, name := parts[0], parts[1]
	mount := r.URL.Query().Get("mount")
	if mount == "" {
		mount = "secret"
	}
	c, err := s.newClient(ns)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		_ = enableKVIfNeeded(c, mount) // ignore error; delete may still work
		if err := deleteKVSecret(c, mount, name); err != nil {
			writeErr(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": name, "namespace": ns})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Middleware & helpers
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
