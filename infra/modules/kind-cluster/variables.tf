variable "nome" {
  description = "Nome do cluster kind (o contexto kubectl será kind-<nome>)"
  type        = string
  default     = "oficina"
}
