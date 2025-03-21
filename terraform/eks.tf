provider "kubernetes" {
  host                   = aws_eks_cluster.geth-cluster.endpoint
  cluster_ca_certificate = base64decode(aws_eks_cluster.geth-cluster.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.geth-cluster.token
}

data "aws_eks_cluster_auth" "geth-cluster" {
  name = aws_eks_cluster.geth-cluster.name
}

resource "kubernetes_deployment" "geth" {
  metadata {
    name      = var.app_name
    namespace = var.namespace
  }

  spec {
    replicas = 1

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
          image = var.image_repo
          name  = var.app_name

          port {
            container_port = 8545
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "app_service" {
  metadata {
    name      = "${var.app_name}-svc"
    namespace = var.namespace
    annotations = {
      "service.beta.kubernetes.io/aws-load-balancer-type" = "nlb"
    }
  }

  spec {
    selector = {
      app = var.app_name
    }

    type = "LoadBalancer"

    port {
      port        = 8545
      target_port = 8545
    }
  }
}
