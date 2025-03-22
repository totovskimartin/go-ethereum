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

          # resources {
          #   limits = {
          #     cpu    = "500m"
          #     memory = "512Mi"
          #   }
          #   requests = {
          #     cpu    = "100m"
          #     memory = "128Mi"
          #   }
          # }
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