output "vpc_id" {
  description = "ID da VPC"
  value       = module.vpc.vpc_id
}

output "eks_cluster_name" {
  description = "Nome do cluster EKS (para aws eks update-kubeconfig)"
  value       = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  description = "Endpoint da API do EKS"
  value       = module.eks.cluster_endpoint
}

output "rds_endpoint" {
  description = "Endpoint do RDS — usar como DB_HOST no overlay k8s/overlays/aws"
  value       = module.rds.db_instance_endpoint
}

output "db_password" {
  description = "Senha gerada do banco (usar no Secret oficina-secrets)"
  value       = random_password.db.result
  sensitive   = true
}
