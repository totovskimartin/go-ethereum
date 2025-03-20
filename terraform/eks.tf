provider "kubernetes" {
  host                   = aws_eks_cluster.example.endpoint
  cluster_ca_certificate = base64decode(aws_eks_cluster.example.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.example.token
}

data "aws_eks_cluster_auth" "cluster" {
  name = aws_eks_cluster.cluster.name
}

resource "kubernetes_deployment" "go_ethereum" {
  metadata {
    name      = "go-ethereum"
    namespace = "geth-app"
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app = "go-ethereum"
      }
    }

    template {
      metadata {
        labels = {
          app = "go-ethereum"
        }
      }
      spec {
        container {
          image = "var.image_repo"
          name  = "var.app_name"

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
