output "contexto_kubectl" {
  description = "Contexto para usar com kubectl (kind exporta automaticamente para ~/.kube/config)"
  value       = module.cluster.contexto_kubectl
}

output "endpoint" {
  description = "Endpoint da API do cluster"
  value       = module.cluster.endpoint
}
