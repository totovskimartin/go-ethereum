resource "random_string" "suffix" {
  length  = 4
  special = false
}

locals {
  cluster_name = "${var.cluster_name}-${random_string.suffix.result}"
}

data "aws_eks_cluster" "cluster" {
  name       = module.eks.cluster_name
  depends_on = [module.eks]
}

data "aws_eks_cluster_auth" "cluster" {
  name       = module.eks.cluster_name
  depends_on = [module.eks]
}

terraform {
  backend "s3" {
    bucket         = "go-ethereum-mtotovski-tfstate"
    key            = "tfstate/terraform.tfstate"
    region         = "eu-west-1"
    dynamodb_table = "terraform-state-locks"
    encrypt        = true
  }
}