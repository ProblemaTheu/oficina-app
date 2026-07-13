output "nome" {
  description = "Nome do cluster"
  value       = kind_cluster.this.name
}

output "contexto_kubectl" {
  description = "Nome do contexto no kubeconfig"
  value       = "kind-${kind_cluster.this.name}"
}

output "endpoint" {
  description = "Endpoint da API do cluster"
  value       = kind_cluster.this.endpoint
}

output "client_certificate" {
  description = "Certificado do cliente (PEM)"
  value       = kind_cluster.this.client_certificate
  sensitive   = true
}

output "client_key" {
  description = "Chave do cliente (PEM)"
  value       = kind_cluster.this.client_key
  sensitive   = true
}

output "cluster_ca_certificate" {
  description = "CA do cluster (PEM)"
  value       = kind_cluster.this.cluster_ca_certificate
  sensitive   = true
}
