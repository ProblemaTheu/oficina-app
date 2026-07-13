# Ambiente local: cluster kind + metrics-server (pré-requisito do HPA).
#
# Uso:
#   terraform init
#   terraform apply
#   ../../..//scripts/k8s-local-deploy.sh   # deploy da aplicação no cluster

module "cluster" {
  source = "../../modules/kind-cluster"

  nome = var.nome_cluster
}

provider "helm" {
  kubernetes {
    host                   = module.cluster.endpoint
    client_certificate     = module.cluster.client_certificate
    client_key             = module.cluster.client_key
    cluster_ca_certificate = module.cluster.cluster_ca_certificate
  }
}

# metrics-server oficial via Helm. A flag --kubelet-insecure-tls é
# necessária no kind (kubelets usam certificado autoassinado).
resource "helm_release" "metrics_server" {
  name       = "metrics-server"
  repository = "https://kubernetes-sigs.github.io/metrics-server/"
  chart      = "metrics-server"
  version    = "3.12.2"
  namespace  = "kube-system"

  set {
    name  = "args[0]"
    value = "--kubelet-insecure-tls"
  }

  depends_on = [module.cluster]
}

variable "nome_cluster" {
  description = "Nome do cluster kind"
  type        = string
  default     = "oficina"
}
