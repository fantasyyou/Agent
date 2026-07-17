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
MYSQL_CONTAINER="${MYSQL_CONTAINER_NAME:-mysql}"
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

# MySQL is stateful and independently managed, like Elasticsearch.
if ! docker container inspect "$MYSQL_CONTAINER" >/dev/null 2>&1; then
  echo "MySQL container '$MYSQL_CONTAINER' does not exist. Create it using 虚拟机初始化操作.md." >&2
  exit 1
fi
if [[ "$(docker inspect -f '{{.State.Running}}' "$MYSQL_CONTAINER")" != "true" ]]; then
  echo "MySQL container '$MYSQL_CONTAINER' exists but is not running." >&2
  exit 1
fi
if ! docker network inspect -f '{{range .Containers}}{{.Name}} {{end}}' "$NETWORK_NAME" | grep -qw "$MYSQL_CONTAINER"; then
  docker network connect "$NETWORK_NAME" "$MYSQL_CONTAINER"
fi

echo "Waiting for MySQL..."
MYSQL_READY=false
for _ in $(seq 1 60); do
  if docker exec "$MYSQL_CONTAINER" mysqladmin ping -uroot -p"${MYSQL_ROOT_PASSWORD:-mysql-root-password}" --silent >/dev/null 2>&1; then
    MYSQL_READY=true
    break
  fi
  sleep 2
done
if [[ "$MYSQL_READY" != "true" ]]; then
  echo "MySQL did not become ready in time." >&2
  docker logs --tail 50 "$MYSQL_CONTAINER" >&2 || true
  exit 1
fi

echo "Waiting for Elasticsearch port 9200..."
ES_READY=false
for _ in $(seq 1 60); do
  if docker exec "$ES_CONTAINER" bash -c '</dev/tcp/127.0.0.1/9200' >/dev/null 2>&1; then
    ES_READY=true
    break
  fi
  sleep 2
done
if [[ "$ES_READY" != "true" ]]; then
  echo "Elasticsearch did not open port 9200 in time." >&2
  docker logs --tail 50 "$ES_CONTAINER" >&2 || true
  exit 1
fi

docker build -t agent-python:latest "$ROOT_DIR/agent-python"
docker build -t agent-go:latest "$ROOT_DIR/agent-go"
docker build -t agent-web:latest "$ROOT_DIR/agent-web"

# Only recreate the three stateless application containers. MySQL and
# Elasticsearch remain independently managed and their data is untouched.
docker rm -f "$WEB_CONTAINER" "$GO_CONTAINER" "$PYTHON_CONTAINER" >/dev/null 2>&1 || true

docker run -d \
  --name "$PYTHON_CONTAINER" \
  --network "$NETWORK_NAME" \
  --restart unless-stopped \
  -p "${PYTHON_DEBUG_PORT}:8765" \
  -e DEEPSEEK_API_KEY="$DEEPSEEK_API_KEY" \
  -e DEEPSEEK_API_URL="${DEEPSEEK_API_URL:-https://api.deepseek.com/v1/chat/completions}" \
  -e DEEPSEEK_MODEL="${DEEPSEEK_MODEL:-deepseek-chat}" \
  -e LOG_LEVEL="${PYTHON_LOG_LEVEL:-INFO}" \
  agent-python:latest

docker run -d \
  --name "$GO_CONTAINER" \
  --network "$NETWORK_NAME" \
  --restart unless-stopped \
  -e MYSQL_DSN="${MYSQL_DSN:-agent_user:agent-password@tcp(mysql:3306)/agent?parseTime=true&charset=utf8mb4&loc=UTC}" \
  -e AUTH_SIGNING_SECRET="${AUTH_SIGNING_SECRET:-demo-only-change-this-signing-secret-32-chars}" \
  -e LOG_LEVEL="${GO_LOG_LEVEL:-INFO}" \
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
