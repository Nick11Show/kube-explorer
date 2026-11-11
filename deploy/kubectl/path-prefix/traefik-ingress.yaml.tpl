# Note: please replace the host first
# To use sslip.io: https://sslip.io/
# To get your public IP: curl ipinfo.io/ip
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kube-explorer-ingress
  namespace: kube-explorer
  annotations:
    traefik.ingress.kubernetes.io/router.middlewares: kube-explorer-prefix@kubernetescrd,kube-explorer-add-header@kubernetescrd
spec:
  rules:
  - host: "${MY_IP}.sslip.io"  # Replace with your actual domain
    http:
      paths:
      - path: /kube-explorer
        pathType: Prefix
        backend:
          service:
            name: kube-explorer
            port:
              name: http
---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: prefix
  namespace: kube-explorer
spec:
  stripPrefix:
    prefixes:
      - /kube-explorer
---
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: add-header
  namespace: kube-explorer
spec:
  headers:
    customRequestHeaders:
      X-Forwarded-Prefix: "/kube-explorer" # Adds
