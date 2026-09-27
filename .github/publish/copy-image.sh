#!/usr/bin/env bash
# Copyright 2026 DevTheNet Labs.
# SPDX-License-Identifier: MIT
set -euo pipefail
[[ "$IMAGE_SHA" =~ ^[a-f0-9]{40}$ ]]
[[ "$IMAGE_DIGEST" =~ ^sha256:[a-f0-9]{64}$ ]]
registry=377946145366.dkr.ecr.us-east-1.amazonaws.com
case "$IMAGE_KIND" in
  runtime) repository=patchy/previews/patchy-preview-demo; tag="sha-$IMAGE_SHA" ;;
  agent) repository=patchy/app-envs/patchy-preview-demo; tag=toolchain-v1 ;;
  *) exit 1 ;;
esac
# A different image at an immutable tag is a hard failure, never an overwrite.
if existing=$(aws ecr describe-images --repository-name "$repository" --image-ids "imageTag=$tag" \
    --query 'imageDetails[0].imageDigest' --output text 2> "$RUNNER_TEMP/ecr-describe-error"); then
  test "$existing" = "$IMAGE_DIGEST"
  echo "Image already published with the expected digest."
  exit 0
else
  grep -q '(ImageNotFoundException)' "$RUNNER_TEMP/ecr-describe-error" || { cat "$RUNNER_TEMP/ecr-describe-error" >&2; exit 1; }
fi
umask 077
authfile="$RUNNER_TEMP/ecr-auth.json"
trap 'rm -f -- "$authfile"' EXIT
aws ecr get-login-password | skopeo login --authfile "$authfile" --username AWS --password-stdin "$registry"
skopeo copy --preserve-digests --authfile "$authfile" "oci:$RUNNER_TEMP/validated" "docker://$registry/$repository:$tag"
actual=$(aws ecr describe-images --repository-name "$repository" --image-ids "imageTag=$tag" \
  --query 'imageDetails[0].imageDigest' --output text)
test "$actual" = "$IMAGE_DIGEST"
printf 'Published %s/%s:%s@%s\n' "$registry" "$repository" "$tag" "$actual"
