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

# Script to clean up all PackageRevisions from focom-resources repository

echo "=========================================="
echo "FOCOM Operator - Cleanup Script"
echo "=========================================="
echo ""

NAMESPACE="${PORCH_NAMESPACE:-default}"
REPOSITORY="${PORCH_REPOSITORY:-focom-resources}"

echo "Configuration:"
echo "  Namespace: $NAMESPACE"
echo "  Repository: $REPOSITORY"
echo ""

echo "1. Listing PackageRevisions to delete..."
echo "---"
PACKAGE_REVISIONS=$(kubectl get packagerevisions -n $NAMESPACE -o json | \
  jq -r ".items[] | select(.spec.repository == \"$REPOSITORY\") | .metadata.name")

if [ -z "$PACKAGE_REVISIONS" ]; then
  echo "No PackageRevisions found for repository '$REPOSITORY'"
  echo ""
  echo "All PackageRevisions in namespace '$NAMESPACE':"
  kubectl get packagerevisions -n $NAMESPACE
  exit 0
fi

echo "Found PackageRevisions:"
echo "$PACKAGE_REVISIONS"
echo ""

read -p "Do you want to delete these PackageRevisions? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
  echo "Cleanup cancelled."
  exit 0
fi

echo ""
echo "2. Deleting PackageRevisions..."
echo "---"

for PR in $PACKAGE_REVISIONS; do
  echo "Processing: $PR"
  
  # Get the lifecycle state
  LIFECYCLE=$(kubectl get packagerevision -n $NAMESPACE $PR -o jsonpath='{.spec.lifecycle}')
  echo "  Current lifecycle: $LIFECYCLE"
  
  # If Published, need to propose deletion first
  if [ "$LIFECYCLE" = "Published" ]; then
    echo "  Proposing deletion (Published -> DeletionProposed)..."
    kubectl patch packagerevision -n $NAMESPACE $PR --type=merge -p '{"spec":{"lifecycle":"DeletionProposed"}}'
    if [ $? -ne 0 ]; then
      echo "  ✗ Failed to propose deletion"
      continue
    fi
    sleep 1
  fi
  
  # Now delete
  echo "  Deleting..."
  kubectl delete packagerevision -n $NAMESPACE $PR
  if [ $? -eq 0 ]; then
    echo "  ✓ Deleted successfully"
  else
    echo "  ✗ Failed to delete"
  fi
  echo ""
done

echo ""
echo "3. Verifying cleanup..."
echo "---"
REMAINING=$(kubectl get packagerevisions -n $NAMESPACE -o json | \
  jq -r ".items[] | select(.spec.repository == \"$REPOSITORY\") | .metadata.name")

if [ -z "$REMAINING" ]; then
  echo "✓ All PackageRevisions deleted successfully!"
else
  echo "⚠ Some PackageRevisions still remain:"
  echo "$REMAINING"
fi

echo ""
echo "4. Checking Git repository status..."
echo "---"
echo "Note: The Git repository may still contain files."
echo "Porch should clean them up, but you may need to verify manually."
echo ""

echo "=========================================="
echo "Cleanup Complete!"
echo "=========================================="
echo ""
echo "You can now start fresh with the demo."
echo "Use the Postman collection to create new resources."
echo ""
