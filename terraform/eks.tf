module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.34.0"

  cluster_name                             = local.cluster_name
  cluster_version                          = "1.31"
  cluster_endpoint_public_access           = true
  enable_cluster_creator_admin_permissions = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  eks_managed_node_group_defaults = {
    ami_type = "AL2_x86_64"
  }

  eks_managed_node_groups = {
    default = {
      min_size       = 1
      max_size       = 2
      desired_size   = 1
      instance_types = ["t3.small"]
    }
  }

  tags = {
    Environment = "dev"
    Terraform   = "true"
  }
}

resource "kubernetes_deployment" "app" {
  metadata {
    name = var.app_name
    labels = {
      app = var.app_name
    }
  }

  spec {
    replicas = var.app_replicas

    selector {
      match_labels = {
        app = var.app_name
      }
    }

    template {
      metadata {
        labels = {
          app = var.app_name
        }
      }

      spec {
        container {
          image = var.docker_image
          name  = var.app_name

          port {
            container_port = 8545
          }
        }
      }
    }
  }

  depends_on = [
    module.eks,
  ]
}

resource "kubernetes_service" "app" {
  metadata {
    name = "geth-service"
  }
  spec {
    selector = {
      app = kubernetes_deployment.app.metadata[0].labels.app
    }
    port {
      port        = 8545
      target_port = 8545
    }
    type = "NodePort" # hitting 'client rate limiter Wait returned an error: context deadline exceeded' using LoadBalancer
  }
}