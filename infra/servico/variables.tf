variable "regiao" {
  description = "Região AWS"
  type        = string
  default     = "us-east-1"
}

variable "ambiente" {
  description = "Ambiente único (corte 10 da Fase 3)"
  type        = string
  default     = "prod"
}

# Mantido da Fase 3 (F4-6.3 passo 4): trocar o namespace trocaria o Service e,
# com ele, o DNS do CLB para o qual o API Gateway aponta.
variable "namespace" {
  description = "Namespace do OS Service no EKS (criado pelo oficina-infra-k8s)"
  type        = string
  default     = "oficina-prod"
}

variable "service_account" {
  description = "ServiceAccount do Deployment (k8s/base/serviceaccount.yaml)"
  type        = string
  default     = "api"
}

variable "alerta_email" {
  description = "E-mail que recebe o alarme da DLQ (vazio = só o tópico SNS, sem assinatura)"
  type        = string
  default     = ""
}
