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
REDIS_CONTAINER="${REDIS_CONTAINER_NAME:-agent-redis}"
WEB_PORT="${WEB_PORT:-8081}"
PYTHON_DEBUG_PORT="${PYTHON_DEBUG_PORT:-8080}"
JAEGER_CONTAINER="agent-jaeger"
COLLECTOR_CONTAINER="agent-collector"
JAEGER_PORT="${JAEGER_PORT:-16686}"

# Fixed names match the existing initialization commands. Never delete or rename volumes.
ensure_volume() {
  if ! docker volume inspect "$1" >/dev/null 2>&1; then
    docker volume create "$1" >/dev/null
  fi
}

check_mount() {
  local container="$1" volume="$2" destination="$3" actual count
  if docker container inspect "$container" >/dev/null 2>&1; then
    actual="$(docker inspect -f "{{range .Mounts}}{{if eq .Destination \"$destination\"}}{{.Type}}:{{.Name}}{{end}}{{end}}" "$container")"
    if [[ "$actual" != "volume:$volume" ]]; then
      echo "$container must use existing volume $volume at $destination; found '$actual'. Refusing to switch data storage." >&2
      exit 1
    fi
    count="$(docker inspect -f '{{range .Mounts}}{{if eq .Type "volume"}}volume {{end}}{{end}}' "$container" | wc -w)"
    if (( count > 1 )); then
      echo "$container has additional volumes. Review its mounts before deployment; no volumes were removed." >&2
      exit 1
    fi
  fi
}

start_existing() {
  local container="$1"
  if [[ "$(docker inspect -f '{{.State.Running}}' "$container")" != "true" ]]; then
    docker start "$container" >/dev/null
  fi
  if ! docker network inspect -f '{{range .Containers}}{{.Name}} {{end}}' "$NETWORK_NAME" | grep -qw "$container"; then
    docker network connect "$NETWORK_NAME" "$container"
  fi
}

command -v docker >/dev/null
command -v curl >/dev/null
docker info >/dev/null

# Validate existing data mounts before creating any new storage.
check_mount "$ES_CONTAINER" es_data /usr/share/elasticsearch/data
check_mount "$MYSQL_CONTAINER" mysql_data /var/lib/mysql
check_mount "$REDIS_CONTAINER" redis_data /data
check_mount "$JAEGER_CONTAINER" agent_jaeger_data /badger
if docker container inspect "$COLLECTOR_CONTAINER" >/dev/null 2>&1; then
  collector_volumes="$(docker inspect -f '{{range .Mounts}}{{if eq .Type "volume"}}volume {{end}}{{end}}' "$COLLECTOR_CONTAINER")"
  if [[ -n "$collector_volumes" ]]; then
    echo "Collector should not use data volumes; review its mounts before deployment." >&2
    exit 1
  fi
fi

if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  docker network create "$NETWORK_NAME" >/dev/null
fi

ensure_volume es_data
ensure_volume mysql_data
ensure_volume redis_data
ensure_volume agent_jaeger_data

if ! docker container inspect "$ES_CONTAINER" >/dev/null 2>&1; then
  docker run -d --name "$ES_CONTAINER" --network "$NETWORK_NAME" \
    --restart unless-stopped --memory=4g --memory-reservation=3g \
    -p 9200:9200 -p 9300:9300 \
    -e discovery.type=single-node -e ES_JAVA_OPTS='-Xms2g -Xmx2g' \
    -e xpack.security.enabled=true -e xpack.security.http.ssl.enabled=false \
    -e ELASTIC_PASSWORD="${ELASTIC_PASSWORD:-elastic}" \
    -v es_data:/usr/share/elasticsearch/data \
    docker.elastic.co/elasticsearch/elasticsearch:8.11.0 >/dev/null
fi
if ! docker container inspect "$MYSQL_CONTAINER" >/dev/null 2>&1; then
  docker run -d --name "$MYSQL_CONTAINER" --network "$NETWORK_NAME" \
    --network-alias mysql --restart unless-stopped --memory=1g --memory-reservation=512m \
    -p 3306:3306 -e MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-mysql-root-password}" \
    -e MYSQL_DATABASE=agent -e MYSQL_USER="${MYSQL_USER:-agent_user}" \
    -e MYSQL_PASSWORD="${MYSQL_PASSWORD:-agent-password}" \
    -v mysql_data:/var/lib/mysql mysql:8.0 >/dev/null
fi
if ! docker container inspect "$REDIS_CONTAINER" >/dev/null 2>&1; then
  docker run -d --name "$REDIS_CONTAINER" --network "$NETWORK_NAME" \
    --restart unless-stopped -p 6379:6379 -v redis_data:/data redis:7-alpine \
    redis-server --appendonly yes --requirepass "${REDIS_PASSWORD:-redis}" >/dev/null
fi
if ! docker container inspect "$JAEGER_CONTAINER" >/dev/null 2>&1; then
  docker run --rm --user 0:0 -v agent_jaeger_data:/badger alpine:3.21 \
    sh -c 'mkdir -p /badger/data /badger/key && chown 10001:10001 /badger /badger/data /badger/key'
  docker run -d --name "$JAEGER_CONTAINER" --network "$NETWORK_NAME" \
    --restart unless-stopped --user 10001:10001 --tmpfs /tmp \
    -p "${JAEGER_PORT}:16686" -e COLLECTOR_OTLP_ENABLED=true \
    -e SPAN_STORAGE_TYPE=badger -e BADGER_EPHEMERAL=false \
    -e BADGER_DIRECTORY_VALUE=/badger/data -e BADGER_DIRECTORY_KEY=/badger/key \
    -e BADGER_SPAN_STORE_TTL=168h -e BADGER_CONSISTENCY=true \
    -v agent_jaeger_data:/badger jaegertracing/all-in-one:1.76.0 >/dev/null
fi
if ! docker container inspect "$COLLECTOR_CONTAINER" >/dev/null 2>&1; then
  docker run -d --name "$COLLECTOR_CONTAINER" --network "$NETWORK_NAME" \
    --restart unless-stopped otel/opentelemetry-collector:0.135.0 \
    --config='yaml:
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318
processors:
  batch: {}
exporters:
  otlphttp:
    endpoint: http://agent-jaeger:4318
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp]
' >/dev/null
fi
for dependency in "$ES_CONTAINER" "$MYSQL_CONTAINER" "$REDIS_CONTAINER" "$JAEGER_CONTAINER" "$COLLECTOR_CONTAINER"; do
  start_existing "$dependency"
done

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

echo "Waiting for Redis..."
REDIS_READY=false
for _ in $(seq 1 60); do
  if docker exec "$REDIS_CONTAINER" redis-cli -a "${REDIS_PASSWORD:-redis}" ping 2>/dev/null | grep -q PONG; then
    REDIS_READY=true
    break
  fi
  sleep 2
done
if [[ "$REDIS_READY" != "true" ]]; then
  echo "Redis did not become ready in time." >&2
  docker logs --tail 50 "$REDIS_CONTAINER" >&2 || true
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

echo "Waiting for Jaeger and Collector..."
OBSERVABILITY_READY=false
for _ in $(seq 1 30); do
  if curl --max-time 3 -fsS "http://127.0.0.1:${JAEGER_PORT}/api/services" >/dev/null 2>&1 && \
     docker run --rm --network "$NETWORK_NAME" alpine:3.21 \
       wget -T 3 -q -O /dev/null --header='Content-Type: application/json' \
       --post-data='{"resourceSpans":[]}' http://agent-collector:4318/v1/traces >/dev/null 2>&1; then
    OBSERVABILITY_READY=true
    break
  fi
  sleep 2
done
if [[ "$OBSERVABILITY_READY" != "true" ]]; then
  echo "Jaeger or Collector did not become ready." >&2
  docker logs --tail 30 "$JAEGER_CONTAINER" >&2 || true
  docker logs --tail 30 "$COLLECTOR_CONTAINER" >&2 || true
  exit 1
fi

docker build -t agent-python:latest "$ROOT_DIR/agent-python"
docker build -t agent-go:latest "$ROOT_DIR/agent-go"
docker build -t agent-web:latest "$ROOT_DIR/agent-web"

# Only recreate application containers; dependencies and data volumes are reused.
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
  -e OTEL_EXPORTER_OTLP_ENDPOINT=http://agent-collector:4318 \
  -e OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf \
  agent-python:latest

docker run -d \
  --name "$GO_CONTAINER" \
  --network "$NETWORK_NAME" \
  --restart unless-stopped \
  -e MYSQL_DSN="${MYSQL_DSN:-${MYSQL_USER:-agent_user}:${MYSQL_PASSWORD:-agent-password}@tcp(${MYSQL_CONTAINER}:3306)/agent?parseTime=true&charset=utf8mb4&loc=UTC}" \
  -e ELASTIC_PASSWORD="${ELASTIC_PASSWORD:-elastic}" \
  -e AUTH_SIGNING_SECRET="${AUTH_SIGNING_SECRET:-demo-only-change-this-signing-secret-32-chars}" \
  -e REDIS_ADDRESS="${REDIS_CONTAINER}:6379" \
  -e REDIS_PASSWORD="${REDIS_PASSWORD:-redis}" \
  -e LOG_LEVEL="${GO_LOG_LEVEL:-INFO}" \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=http://agent-collector:4318 \
  -e OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf \
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
    echo "Jaeger UI:          http://$(hostname -I | awk '{print $1}'):${JAEGER_PORT}"
    echo "Logs: docker logs -f ${GO_CONTAINER}"
    exit 0
  fi
  sleep 1
done

echo "Web service did not become healthy. Recent logs:" >&2
docker logs --tail 50 "$GO_CONTAINER" >&2 || true
exit 1
