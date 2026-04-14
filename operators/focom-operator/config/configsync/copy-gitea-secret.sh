#!/bin/bash

#  Copyright 2026 The Nephio Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# Script to copy gitea-secret from default namespace to config-management-system
# This allows ConfigSync to authenticate to the Git repository

set -e

echo "Copying gitea-secret from default to config-management-system namespace..."

# Get the secret from default namespace and recreate it in config-management-system
kubectl get secret gitea-secret -n default -o yaml | \
  sed 's/namespace: default/namespace: config-management-system/' | \
  kubectl apply -f -

echo "✅ Secret copied successfully!"
echo ""
echo "To verify:"
echo "  kubectl get secret gitea-secret -n config-management-system"
