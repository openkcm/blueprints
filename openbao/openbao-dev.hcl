listener "tcp" {
  address = "127.0.0.1:8200"
  tls_cert_file = "certs/server.crt"
  tls_key_file  = "certs/server.key"
  # Require and verify client certificates for mTLS auth testing
  tls_client_ca_file = "certs/ca.pem"
  tls_require_and_verify_client_cert = true
}
api_addr = "https://127.0.0.1:8200"

# Local single-node storage backend (required). For a quick dev setup you can use "file".
storage "file" {
  path = "openbao-data"
}

# If you prefer embedded raft instead of file, comment the file stanza above and use:
# storage "raft" {
#   path    = "openbao-raft"
#   node_id = "node1"
# }

ui             = true

