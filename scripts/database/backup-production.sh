#!/bin/sh
set -eu
umask 077

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

container="${POSTGRES_CONTAINER:-app_financeiro_db}"
database="${POSTGRES_DB:-app_financeiro}"
user="${POSTGRES_USER:-postgres}"
backup_dir="${1:-migration-backup}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
dump_file="$backup_dir/app_financeiro-$timestamp.dump"
schema_file="$backup_dir/app_financeiro-schema-$timestamp.sql"
list_file="$backup_dir/app_financeiro-$timestamp.list"

cleanup() {
  rm -f -- "$dump_file.tmp" "$schema_file.tmp" "$list_file.tmp"
}
trap cleanup EXIT INT TERM

mkdir -p "$backup_dir"

if ! docker_run inspect "$container" >/dev/null 2>&1; then
  echo "Container PostgreSQL nao encontrado: $container" >&2
  exit 1
fi

echo "Gerando backup completo em formato custom..."
docker_run exec "$container" pg_dump --username "$user" --dbname "$database" --format custom --no-owner --no-privileges > "$dump_file.tmp"
test -s "$dump_file.tmp"
mv "$dump_file.tmp" "$dump_file"

echo "Gerando copia do schema para revisao..."
docker_run exec "$container" pg_dump --username "$user" --dbname "$database" --schema-only --no-owner --no-privileges > "$schema_file.tmp"
test -s "$schema_file.tmp"
mv "$schema_file.tmp" "$schema_file"

if [ ! -s "$dump_file" ] || [ ! -s "$schema_file" ]; then
  echo "Backup invalido: um dos arquivos foi criado vazio." >&2
  exit 1
fi

docker_run exec -i "$container" pg_restore --list < "$dump_file" > "$list_file.tmp"
test -s "$list_file.tmp"
mv "$list_file.tmp" "$list_file"
if [ ! -s "$list_file" ]; then
  echo "Backup invalido: pg_restore nao conseguiu listar o conteudo." >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$dump_file" "$schema_file" "$list_file" > "$backup_dir/checksums-$timestamp.sha256"
fi

chmod 600 "$dump_file" "$schema_file" "$list_file" "$backup_dir/checksums-$timestamp.sha256" 2>/dev/null || true
trap - EXIT INT TERM

echo "Backup criado e validado:"
echo "  Dados:  $dump_file"
echo "  Schema: $schema_file"
echo "  Lista:  $list_file"
echo "O proximo passo obrigatorio e restaurar o arquivo .dump em um PostgreSQL descartavel."
