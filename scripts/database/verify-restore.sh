#!/bin/sh
set -eu

use_sudo=false
if ! docker info >/dev/null 2>&1; then
  if sudo -n docker info >/dev/null 2>&1; then
    use_sudo=true
  else
    echo "Docker indisponivel para o usuario atual e via sudo sem senha." >&2
    exit 1
  fi
fi

docker_run() {
  if [ "$use_sudo" = true ]; then
    sudo -n docker "$@"
  else
    docker "$@"
  fi
}

if [ "$#" -ne 1 ]; then
  echo "Uso: $0 caminho/do/backup.dump" >&2
  exit 2
fi

dump_file="$1"
if [ ! -s "$dump_file" ]; then
  echo "Arquivo de backup inexistente ou vazio: $dump_file" >&2
  exit 1
fi

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
container="sobraai-restore-check-$timestamp"
image="${POSTGRES_IMAGE:-postgres:15-alpine}"
password="restore-check-only"

cleanup() {
	case "$container" in
    sobraai-restore-check-*) docker_run rm --force --volumes "$container" >/dev/null 2>&1 || true ;;
    *) echo "Recusando remover container inesperado: $container" >&2 ;;
  esac
}
trap cleanup EXIT INT TERM

echo "Iniciando PostgreSQL temporario sem volume..."
docker_run run --detach \
	--name "$container" \
	--tmpfs /var/lib/postgresql/data:rw,noexec,nosuid,size=256m \
  --env POSTGRES_PASSWORD="$password" \
  --env POSTGRES_DB=restore_test \
  "$image" >/dev/null

attempt=0
until docker_run exec "$container" pg_isready --username postgres --dbname restore_test >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 30 ]; then
    echo "PostgreSQL temporario nao ficou pronto." >&2
    exit 1
  fi
  sleep 1
done

docker_run cp "$dump_file" "$container:/tmp/backup.dump" >/dev/null
docker_run exec "$container" pg_restore \
  --username postgres \
  --dbname restore_test \
	--no-owner \
	--no-privileges \
	--exit-on-error \
	--single-transaction \
  /tmp/backup.dump

echo "Tabelas restauradas:"
docker_run exec "$container" psql --username postgres --dbname restore_test --tuples-only --command \
  "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;"

echo "Contagens principais:"
docker_run exec "$container" psql --username postgres --dbname restore_test --tuples-only --command \
  "SELECT 'users', COUNT(*) FROM users UNION ALL SELECT 'expenses', COUNT(*) FROM expenses UNION ALL SELECT 'incomes', COUNT(*) FROM incomes UNION ALL SELECT 'app_versions', COUNT(*) FROM app_versions;"

echo "Restauracao validada em banco descartavel. O container temporario sera removido agora."
