#!/bin/bash
# This script generates a bearer token for the knuu service account.
# It is used to authenticate the knuu service account to the Kubernetes API server.
# The token is used to test things manually. 
# It requries ~/.kube/config to be set up.

# Variables
SERVICE_ACCOUNT_NAME="knuu-service-account"
NAMESPACE="default"
ROLE_BINDING_NAME="knuu-rolebinding"

cleanup() {
  echo "Cleaning up resources..."
  kubectl delete clusterrolebinding $ROLE_BINDING_NAME --ignore-not-found
  kubectl delete serviceaccount $SERVICE_ACCOUNT_NAME --namespace=$NAMESPACE --ignore-not-found
}
trap cleanup EXIT

# Create ServiceAccount
echo "Creating ServiceAccount..."
kubectl delete serviceaccount $SERVICE_ACCOUNT_NAME --namespace=$NAMESPACE --ignore-not-found
kubectl create serviceaccount $SERVICE_ACCOUNT_NAME --namespace=$NAMESPACE

# Bind the cluster-admin role to the ServiceAccount
echo "Binding cluster-admin role..."
kubectl delete clusterrolebinding $ROLE_BINDING_NAME --ignore-not-found
kubectl create clusterrolebinding $ROLE_BINDING_NAME \
  --clusterrole=cluster-admin \
  --serviceaccount=$NAMESPACE:$SERVICE_ACCOUNT_NAME 2>/dev/null || \
kubectl replace clusterrolebinding $ROLE_BINDING_NAME \
  --clusterrole=cluster-admin \
  --serviceaccount=$NAMESPACE:$SERVICE_ACCOUNT_NAME

# Generate token
TOKEN=$(kubectl create token $SERVICE_ACCOUNT_NAME --namespace=$NAMESPACE)
if [ -z "$TOKEN" ]; then
  echo "Failed to generate token!"
  exit 1
fi

# Get API server URL
API_SERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')

# Get CA Certificate
CA_CERT=$(kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 --decode)

# Export variables
export K8S_HOST=$API_SERVER
export K8S_AUTH_TOKEN=$TOKEN
export K8S_CA_CERT="$CA_CERT"

# Output
echo "============================="
echo "Bearer Token: $TOKEN"
echo "API Server: $API_SERVER"
echo "CA Certificate: $CA_CERT"
echo "============================="

# Cleanup happens automatically on script exit.

# Example how to run the tests using the auth token:
# . ./scripts/auth_token.sh && go test -v ./e2e/basic/
