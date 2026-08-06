#!/usr/bin/env bash
# SOBER Deploy: Automates Build -> Push -> Deploy for Fly.io Nix containers
set -e

# Configuration
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
HOST_NAME=$(basename "$DIR")
APP_NAME="sober-$HOST_NAME"
IMAGE_ATTR="$HOST_NAME-image"
REGISTRY="registry.fly.io/$APP_NAME:latest"

echo "🦉 [SOBER] Starting deployment for: $HOST_NAME ($APP_NAME)"

# 1. Build via nixbuild.net (Offload CPU/RAM work)
echo "🔨 1/3: Building $IMAGE_ATTR via nixbuild.net..."
(cd "$DIR/../../.." && nix build .#$IMAGE_ATTR --verbose --option max-jobs 0)

# 2. Push via Skopeo (Daemonless)
echo "🚀 2/3: Pushing image to Fly registry..."
nix run nixpkgs#skopeo --extra-experimental-features 'nix-command flakes' -- copy \
  docker-archive:$(readlink -f "$DIR/../../../result") \
  docker://$REGISTRY \
  --dest-creds x:$(fly tokens create deploy -x 30m -q)

# 3. Deploy
echo "🚢 3/3: Executing fly deploy..."
cd "$DIR"
fly deploy --image $REGISTRY

echo "✅ [SOBER] Deployment of $HOST_NAME complete!"
