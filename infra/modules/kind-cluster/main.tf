# Cluster Kubernetes local com kind (Kubernetes in Docker).
# O provider embute o kind — só precisa do Docker rodando na máquina.
resource "kind_cluster" "this" {
  name           = var.nome
  wait_for_ready = true

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    node {
      role = "control-plane"
    }
  }
}
