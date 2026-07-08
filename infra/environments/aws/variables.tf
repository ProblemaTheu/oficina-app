variable "projeto" {
  description = "Nome do projeto (prefixo dos recursos)"
  type        = string
  default     = "tech-challenge"
}

variable "regiao" {
  description = "Região AWS"
  type        = string
  default     = "us-east-1"
}

variable "vpc_cidr" {
  description = "CIDR da VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "eks_versao" {
  description = "Versão do Kubernetes no EKS"
  type        = string
  default     = "1.31"
}

variable "eks_instance_types" {
  description = "Tipos de instância dos nós do EKS"
  type        = list(string)
  default     = ["t3.medium"]
}

variable "eks_min_nodes" {
  description = "Mínimo de nós no node group"
  type        = number
  default     = 2
}

variable "eks_max_nodes" {
  description = "Máximo de nós no node group"
  type        = number
  default     = 3
}

variable "db_instance_class" {
  description = "Classe da instância RDS"
  type        = string
  default     = "db.t3.micro"
}

variable "db_nome" {
  description = "Nome do banco de dados"
  type        = string
  default     = "tech_challenge_db"
}

variable "db_usuario" {
  description = "Usuário master do RDS"
  type        = string
  default     = "postgres"
}
