terraform {
  required_version = ">= 1.5"

  required_providers {
    kind = {
      source  = "tehcyx/kind"
      version = "~> 0.9"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17"
    }
  }

  # State local (padrão) — suficiente para o ambiente de desenvolvimento.
  # Evolução futura (AWS): backend S3 com lock em DynamoDB, ex.:
  # backend "s3" {
  #   bucket         = "tech-challenge-tfstate"
  #   key            = "local/terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "tech-challenge-tflock"
  # }
}
