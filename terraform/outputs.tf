output "cluster_endpoint" {
  description = "Endpoint for EKS control plane"
  value       = module.eks.cluster_endpoint
}

output "cluster_security_group_id" {
  description = "Security group ID attached to the EKS cluster"
  value       = module.eks.cluster_security_group_id
}

output "cluster_name" {
  description = "Kubernetes Cluster Name"
  value       = module.eks.cluster_id
}

# output "load_balancer_hostname" {
#   description = "Hostname of the load balancer"
#   value       = kubernetes_service.app.status.0.load_balancer.0.ingress.0.hostname
# }