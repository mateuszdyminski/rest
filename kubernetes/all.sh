#!/usr/bin/env bash

# Create namespace
kubectl create -f namespaces/rest-namespace.yml

# Create limits
kubectl create -f limits/rest-limits.yml

# Create deployments
kubectl create -f deployments/rest-api.yml

# Create service
kubectl create -f services/api.yml
