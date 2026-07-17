#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -f "$ROOT_DIR/agent-python/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/agent-python/.env"
  set +a
fi

: "${DEEPSEEK_API_KEY:?Set DEEPSEEK_API_KEY in $ROOT_DIR/agent-python/.env}"

NETWORK_NAME="${AGENT_NETWORK_NAME:-agent-network}"
PYTHON_CONTAINER="agent-app"
GO_CONTAINER="agent-go"
WEB_CONTAINER="agent-web"
ES_CONTAINER="es"
WEB_PORT="${WEB_PORT:-8081}"
PYTHON_DEBUG_PORT="${PYTHON_DEBUG_PORT:-8080}"

if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  docker network create "$NETWORK_NAME" >/dev/null
fi

# Elasticsearch is stateful and managed independently. Attach the existing
# container to the application network without restarting or recreating it.
if ! docker container inspect "$ES_CONTAINER" >/dev/null 2>&1; then
  echo "Elasticsearch container '$ES_CONTAINER' is not running or does not exist." >&2
  exit 1
fi
if [[ "$(docker inspect -f '{{.State.Running}}' "$ES_CONTAINER")" != "true" ]]; then
  echo "Elasticsearch container '$ES_CONTAINER' exists but is not running." >&2
  exit 1
fi
if ! docker network inspect -f '{{range .Containers}}{{.Name}} {{end}}' "$NETWORK_NAME" | grep -qw "$ES_CONTAINER"; then
  docker network connect "$NETWORK_NAME" "$ES_CONTAINER"
fi

docker build -t agent-python:latest "$ROOT_DIR/agent-python"
docker build -t agent-go:latest "$ROOT_DIR/agent-go"
docker build -t agent-web:latest "$ROOT_DIR/agent-web"

# Only recreate the three stateless application containers. Elasticsearch and
# Redis remain independently managed and their data is untouched.
docker rm -f "$WEB_CONTAINER" "$GO_CONTAINER" "$PYTHON_CONTAINER" >/dev/null 2>&1 || true

docker run -d \
  --name "$PYTHON_CONTAINER" \
  --network "$NETWORK_NAME" \
  --restart unless-stopped \
  -p "${PYTHON_DEBUG_PORT}:8765" \
  -e DEEPSEEK_API_KEY="$DEEPSEEK_API_KEY" \
  -e DEEPSEEK_API_URL="${DEEPSEEK_API_URL:-https://api.deepseek.com/v1/chat/completions}" \
  -e DEEPSEEK_MODEL="${DEEPSEEK_MODEL:-deepseek-chat}" \
  agent-python:latest

docker run -d \
  --name "$GO_CONTAINER" \
  --network "$NETWORK_NAME" \
  --restart unless-stopped \
  -v "$ROOT_DIR/agent-go/config.json:/app/config.json:ro" \
  agent-go:latest

docker run -d \
  --name "$WEB_CONTAINER" \
  --network "$NETWORK_NAME" \
  --restart unless-stopped \
  -p "${WEB_PORT}:80" \
  agent-web:latest

echo "Waiting for the web service..."
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${WEB_PORT}/health" >/dev/null 2>&1; then
    echo "Customer-service UI: http://$(hostname -I | awk '{print $1}'):${WEB_PORT}"
    echo "Local UI:           http://127.0.0.1:${WEB_PORT}"
    echo "Logs: docker logs -f ${GO_CONTAINER}"
    exit 0
  fi
  sleep 1
done

echo "Web service did not become healthy. Recent logs:" >&2
docker logs --tail 50 "$GO_CONTAINER" >&2 || true
exit 1
