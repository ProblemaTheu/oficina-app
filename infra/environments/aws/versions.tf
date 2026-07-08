terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
  }

  # State local por enquanto. Antes de usar em equipe, migrar para S3:
  # backend "s3" {
  #   bucket         = "tech-challenge-tfstate"
  #   key            = "aws/terraform.tfstate"
  #   region         = "us-east-1"
  #   dynamodb_table = "tech-challenge-tflock"
  # }
}

provider "aws" {
  region = var.regiao

  default_tags {
    tags = {
      Projeto    = var.projeto
      Gerenciado = "terraform"
    }
  }
}
