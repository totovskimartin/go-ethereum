variable "region" {
  description = "The AWS region to create resources in"
  default     = "eu-west-1"
}

variable "cluster_name" {
  description = "The name of the EKS cluster"
  default     = "go-ethereum-eks-cluster"
}

variable "node_instance_type" {
  description = "EC2 instance type for EKS nodes"
  default     = "t2.micro"
}

variable "image_repo" {
  description = "Container image repository"
  type        = string
  default     = "mtotovski/go-ethereum"
}

variable "image_tag" {
  description = "Container image tag"
  type        = string
  default     = "contracts-deployed"
}

variable "namespace" {
  default = "default"
}

variable "app_name" {
  default = "geth-app"
}

variable "replicas" {
  default = 1
}