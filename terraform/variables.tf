variable "region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-1"
}

variable "app_name" {
  description = "Name of the application"
  type        = string
  default     = "go-ethereum"
}

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "geth-cluster"
}

variable "docker_image" {
  description = "Docker image to deploy"
  type        = string
  default     = "mtotovski/go-ethereum-devnet"
}

variable "app_replicas" {
  description = "Number of replicas for the application"
  type        = number
  default     = 1
}