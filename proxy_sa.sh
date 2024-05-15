#!/bin/bash

# Step 1: Create a new namespace
kubectl create namespace my-traefik-namespace

kubectl create serviceaccount traefik-service-account -n my-traefik-namespace


# Step 2: Deploy Traefik with a specific Service Account and Roles
helm repo add traefik https://helm.traefik.io/traefik
helm repo update
helm install traefik traefik/traefik \
  --namespace my-traefik-namespace \
  --create-namespace \
  --set="additionalArguments={--api.insecure=true,--providers.kubernetesIngress,--providers.kubernetesCRD}" \
  --set serviceAccount.create=true \
  --set serviceAccount.name=traefik-service-account \
  --set rbac.enabled=true

# Wait for Traefik to be ready
echo "Waiting for Traefik to be ready..."
kubectl wait --namespace my-traefik-namespace --for=condition=ready pod --selector="app.kubernetes.io/name=traefik" --timeout=90s
# Step 3: Deploy a dummy webserver with its own Service Account and Role
## Create Service Account for dummy webserver
kubectl create serviceaccount dummy-webserver-account -n my-traefik-namespace

## Create Role and RoleBinding for dummy webserver
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: my-traefik-namespace
  name: webserver-role
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: webserver-role-binding
  namespace: my-traefik-namespace
subjects:
- kind: ServiceAccount
  name: dummy-webserver-account
  namespace: my-traefik-namespace
roleRef:
  kind: Role
  name: webserver-role
  apiGroup: rbac.authorization.k8s.io
EOF

## Deploy dummy webserver using the created Service Account
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dummy-webserver
  namespace: my-traefik-namespace
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dummy-webserver
  template:
    metadata:
      labels:
        app: dummy-webserver
    spec:
      serviceAccountName: dummy-webserver-account
      containers:
      - name: webserver
        image: nginx
        ports:
        - containerPort: 80
EOF

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: dummy-webserver
  namespace: my-traefik-namespace
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
  selector:
    app: dummy-webserver
EOF


# Step 4: Configure Ingress to route traffic to the dummy webserver
## Middleware
cat <<EOF | kubectl apply -f -
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: strip-dummy
  namespace: my-traefik-namespace
spec:
  stripPrefix:
    prefixes:
      - "/dummy"
EOF

## IngressRoute
cat <<EOF | kubectl apply -f -
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: dummy-webserver-route
  namespace: my-traefik-namespace
spec:
  entryPoints:
    - web
  routes:
    - match: PathPrefix(\`/dummy\`)
      kind: Rule
      services:
        - name: dummy-webserver
          port: 80
      middlewares:
        - name: strip-dummy
EOF

# Step 5: Get the public IP address of the Traefik LoadBalancer
sleep 10
TRAFFIK_IP=$(kubectl get svc traefik -n my-traefik-namespace -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Traefik Public IP: $TRAFFIK_IP"

# Step 6: Write a curl command that sends a request to the dummy webserver through the proxy
echo "Sending request to dummy webserver through Traefik..."
curl http://$TRAFFIK_IP/dummy
