#!/bin/bash
# Script to create Gitea secret for Porch authentication

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Creating Gitea Secret for Porch${NC}"
echo "========================================"
echo ""

# Prompt for credentials
read -p "Enter Gitea username [nephio]: " USERNAME
USERNAME=${USERNAME:-nephio}

read -sp "Enter Gitea password: " PASSWORD
echo ""

read -sp "Enter Gitea access token: " TOKEN
echo ""

# Validate inputs
if [ -z "$PASSWORD" ]; then
    echo -e "${RED}Error: Password cannot be empty${NC}"
    exit 1
fi

if [ -z "$TOKEN" ]; then
    echo -e "${RED}Error: Token cannot be empty${NC}"
    exit 1
fi

# Base64 encode values
USERNAME_B64=$(echo -n "$USERNAME" | base64)
PASSWORD_B64=$(echo -n "$PASSWORD" | base64)
TOKEN_B64=$(echo -n "$TOKEN" | base64)

# Create secret YAML
cat > /tmp/gitea-secret.yaml <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: gitea-secret
  namespace: default
type: kubernetes.io/basic-auth
data:
  username: $USERNAME_B64
  password: $PASSWORD_B64
  bearerToken: $TOKEN_B64
EOF

echo ""
echo -e "${YELLOW}Secret YAML created at /tmp/gitea-secret.yaml${NC}"
echo ""

# Apply secret
read -p "Apply secret to cluster? (y/n): " APPLY
if [ "$APPLY" = "y" ] || [ "$APPLY" = "Y" ]; then
    kubectl apply -f /tmp/gitea-secret.yaml
    echo -e "${GREEN}Secret created successfully!${NC}"
    echo ""
    
    # Ask to restart Porch controllers
    read -p "Restart Porch controllers to pick up new secret? (y/n): " RESTART
    if [ "$RESTART" = "y" ] || [ "$RESTART" = "Y" ]; then
        kubectl rollout restart deployment/porch-controllers -n porch-system
        echo -e "${GREEN}Porch controllers restarted!${NC}"
        echo ""
        echo "Waiting for rollout to complete..."
        kubectl rollout status deployment/porch-controllers -n porch-system --timeout=60s
    fi
else
    echo -e "${YELLOW}Secret not applied. You can apply it manually with:${NC}"
    echo "  kubectl apply -f /tmp/gitea-secret.yaml"
fi

echo ""
echo -e "${GREEN}Done!${NC}"
