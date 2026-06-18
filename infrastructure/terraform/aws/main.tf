terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
  }

  backend "s3" {
    bucket         = "titan-os-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "Titan OS"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

# VPC and Networking
module "vpc" {
  source = "./modules/vpc"

  name              = "titan-os"
  cidr              = "10.0.0.0/16"
  availability_zones = data.aws_availability_zones.available.names
  public_subnets    = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  private_subnets   = ["10.0.11.0/24", "10.0.12.0/24", "10.0.13.0/24"]
  database_subnets  = ["10.0.21.0/24", "10.0.22.0/24", "10.0.23.0/24"]

  enable_nat_gateway = true
  single_nat_gateway = false
  enable_vpn_gateway = true
}

# EKS Cluster
module "eks" {
  source = "./modules/eks"

  cluster_name           = "titan-os-${var.environment}"
  cluster_version        = "1.27"
  vpc_id                 = module.vpc.vpc_id
  subnet_ids             = module.vpc.private_subnet_ids

  node_groups = {
    general = {
      instance_types = ["t3.xlarge"]
      desired_size   = 3
      min_size       = 2
      max_size       = 10
      disk_size      = 100
    }
    memory = {
      instance_types = ["r5.2xlarge"]
      desired_size   = 2
      min_size       = 1
      max_size       = 5
      disk_size      = 150
    }
  }
}

# RDS PostgreSQL
module "rds" {
  source = "./modules/rds"

  identifier     = "titan-os-${var.environment}"
  engine         = "postgres"
  engine_version = "15.3"
  instance_class = "db.r5.xlarge"

  allocated_storage = 100
  storage_encrypted = true

  db_name  = "titan_db"
  username = "titan"
  password = random_password.db_password.result

  vpc_security_group_ids = [aws_security_group.rds.id]
  db_subnet_group_name   = aws_db_subnet_group.default.name

  backup_retention_period = 30
  skip_final_snapshot     = false
  final_snapshot_identifier = "titan-os-final-snapshot-${data.aws_caller_identity.current.account_id}"
}

# ElastiCache Redis
module "redis" {
  source = "./modules/redis"

  cluster_id           = "titan-os-${var.environment}"
  engine_version       = "7.2"
  node_type            = "cache.r6g.xlarge"
  num_cache_nodes      = 3
  parameter_group_name = "default.redis7.2"

  subnet_group_name = aws_elasticache_subnet_group.default.name
  security_group_ids = [aws_security_group.redis.id]

  automatic_failover_enabled = true
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
}

# MSK Kafka Cluster
module "msk" {
  source = "./modules/msk"

  cluster_name           = "titan-os-${var.environment}"
  kafka_version          = "3.5.0"
  number_of_broker_nodes = 3
  broker_node_group_info = {
    instance_type   = "kafka.m6g.xlarge"
    client_subnets  = module.vpc.private_subnet_ids
    security_groups = [aws_security_group.msk.id]
    storage_info = {
      ebs_volume_size = 500
    }
  }

  encryption_info = {
    encryption_at_rest = {
      enabled = true
    }
    encryption_in_transit = {
      client_broker = "TLS"
      in_cluster    = true
    }
  }
}

# S3 Bucket for artifacts
resource "aws_s3_bucket" "artifacts" {
  bucket = "titan-os-artifacts-${data.aws_caller_identity.current.account_id}"
}

resource "aws_s3_bucket_versioning" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "titan" {
  name              = "/aws/eks/titan-os-${var.environment}"
  retention_in_days = 30
}

# Generate random password for database
resource "random_password" "db_password" {
  length  = 32
  special = true
}

# Data sources
data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

# Outputs
output "eks_cluster_id" {
  value       = module.eks.cluster_id
  description = "EKS Cluster ID"
}

output "eks_cluster_endpoint" {
  value       = module.eks.cluster_endpoint
  description = "EKS Cluster endpoint"
}

output "rds_endpoint" {
  value       = module.rds.db_instance_endpoint
  description = "RDS endpoint"
  sensitive   = true
}

output "redis_endpoint" {
  value       = module.redis.configuration_endpoint_address
  description = "Redis endpoint"
}

output "msk_bootstrap_servers" {
  value       = module.msk.bootstrap_servers_tls
  description = "MSK bootstrap servers"
}

output "s3_bucket_name" {
  value       = aws_s3_bucket.artifacts.id
  description = "S3 bucket for artifacts"
}
