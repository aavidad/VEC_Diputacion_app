#!/usr/bin/env bash
set -euo pipefail

# Testigo de un solo uso. Requiere un directorio y un contenedor propios nuevos;
# conserva ambos ante un fallo para permitir el diagnóstico y nunca toca cidonia.
raiz_repo=$(git rev-parse --show-toplevel)
[[ $# -eq 2 ]]
base=$1
contenedor=$2
[[ $base =~ ^/dev/shm/vec-rpt-testigo-v3-[a-z0-9-]+$ ]]
[[ $contenedor =~ ^vec-rpt-testigo-v3-[a-z0-9-]+$ ]]
[[ ${base##*/} == "$contenedor" ]]
archivo=/home/alberto/.local/state/vec-clon/estado-cidonia-20260929-hito1.tgz
huella_h1=d1c2e38a85f872e83b6ba9eca6b660b7a5d2ff5d39d4aca612a81ec30a496e5b
fuente_h6=742127f5
toolchain=/home/alberto/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.linux-amd64

[[ $raiz_repo == /home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/codexd-rpt-testigo-v3-20260930 ]]
git merge-base --is-ancestor 90ec895b8c007ae392d34c83456900412f9dbaa4 HEAD
for ruta in deploy/postgresql/catalogos_configurables/migraciones/000001_autoridad_categorias.up.sql \
            deploy/postgresql/catalogos_configurables/migraciones/000002_lecturas_nominales.up.sql \
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000117_lecturas_categorias_rpt.up.sql; do
  [[ $(git rev-parse "HEAD:$ruta") == $(git rev-parse "90ec895b8c007ae392d34c83456900412f9dbaa4:$ruta") ]]
done
[[ ! -e $base && -f $archivo && -d $toolchain ]]
printf '%s  %s\n' "$huella_h1" "$archivo" | sha256sum -c --status
if docker container inspect "$contenedor" >/dev/null 2>&1; then
  echo 'RPT-V3-FALLO: contenedor propio ya existente' >&2
  exit 1
fi
mkdir -p "$base/build"
sha256sum "$archivo" > "$base/build/h1_sha256.txt"
tar -xzf "$archivo" -C "$base"
docker run -d --pull never --name "$contenedor" --network none --memory 4g --cpus 2 \
  --pids-limit 128 --shm-size 256m \
  --mount "type=bind,src=$base,dst=/var/lib/postgresql" \
  --tmpfs /tmp:rw,nosuid,nodev,size=32m \
  --tmpfs /var/run/postgresql:rw,nosuid,nodev,size=16m,uid=999,gid=999 \
  postgres:18.4 >/dev/null
for _ in {1..100}; do
  if docker exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres; then break; fi
  sleep 0.2
done
docker exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres

aplicar() {
  docker exec -i --user postgres "$contenedor" \
    psql -X -q -v ON_ERROR_STOP=1 -d postgres < "$1" >/dev/null
}
aplicar_git() {
  git -C "$raiz_repo" show "$fuente_h6:$1" |
    docker exec -i --user postgres "$contenedor" \
      psql -X -q -v ON_ERROR_STOP=1 -d postgres >/dev/null
}
for lista in /home/alberto/Trabajo/hito3/paquete/lista_sql_h3.txt \
             /home/alberto/Trabajo/hito4/lista_sql_h4.fuente.txt \
             /home/alberto/Trabajo/hito6/lista_sql_h6.fuente.txt; do
  while IFS= read -r ruta; do
    [[ $ruta =~ ^deploy/postgresql/[a-zA-Z0-9_./-]+\.sql$ && $ruta != *..* ]]
    aplicar_git "$ruta"
  done < "$lista"
done
aplicar "$raiz_repo/deploy/postgresql/catalogos_configurables/roles_up.sql"
aplicar "$raiz_repo/deploy/postgresql/catalogos_configurables/migraciones/000001_autoridad_categorias.up.sql"
aplicar "$raiz_repo/deploy/postgresql/catalogos_configurables/migraciones/000002_lecturas_nominales.up.sql"
aplicar "$raiz_repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000117_lecturas_categorias_rpt.up.sql"

docker exec --user postgres "$contenedor" psql -X -qAt -d postgres -c \
  "SELECT datacl FROM pg_catalog.pg_database WHERE datname=current_database()" \
  > "$base/build/datacl_antes.txt"
inventario_acl() {
  docker exec --user postgres "$contenedor" psql -X -qAt -d postgres -c "
    SELECT 'public_tipos_contexto',count(*) FROM pg_catalog.pg_type t
      JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
      CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(t.typacl,pg_catalog.acldefault('T',t.typowner))) a
     WHERE n.nspname='vec_contexto_actor_v1' AND t.typtype='c' AND a.grantee=0
    UNION ALL
    SELECT 'runtime_tipos_contexto',count(*) FROM pg_catalog.pg_type t
      JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
      CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(t.typacl,pg_catalog.acldefault('T',t.typowner))) a
     WHERE n.nspname='vec_contexto_actor_v1' AND t.typtype='c'
       AND a.grantee='vec_contexto_actor_v1_runtime'::regrole
    UNION ALL
    SELECT 'revalidador_dependencias',count(*) FROM pg_catalog.pg_shdepend
     WHERE refclassid='pg_catalog.pg_authid'::regclass
       AND refobjid='vec_identidad_sesiones_v1_revalidador'::regrole"
}
inventario_acl > "$base/build/acl_antes.txt"
docker exec -i --user postgres "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -d postgres >/dev/null <<'SQL'
GRANT CONNECT ON DATABASE postgres TO vec_contexto_actor_v1_runtime;
GRANT CONNECT ON DATABASE postgres TO vec_identidad_sesiones_v1_revalidador;
REVOKE TEMPORARY ON DATABASE postgres FROM PUBLIC;
DO $do$
DECLARE tipo text;
BEGIN
  FOR tipo IN SELECT pg_catalog.format('%I.%I',n.nspname,t.typname)
    FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid=t.typnamespace
   WHERE n.nspname='vec_contexto_actor_v1' AND t.typtype='c'
  LOOP
    EXECUTE 'REVOKE ALL ON TYPE '||tipo||' FROM PUBLIC,vec_contexto_actor_v1_runtime';
  END LOOP;
END $do$;
SQL
docker exec --user postgres "$contenedor" psql -X -qAt -d postgres -c \
  "SELECT datacl FROM pg_catalog.pg_database WHERE datname=current_database()" \
  > "$base/build/datacl_despues.txt"
inventario_acl > "$base/build/acl_despues.txt"
echo 'RPT-V3-ACL-CLON-ADAPTADA'

umask 077
cat > "$base/build/dsn.json" <<'JSON'
{"admin":"host=/var/run/postgresql dbname=postgres user=postgres sslmode=disable","contexto":"host=/var/run/postgresql dbname=postgres user=vec_rpt_testigo_contexto sslmode=disable","runtime":"host=/var/run/postgresql dbname=postgres user=vec_rpt_testigo_ct_ejecutor sslmode=disable","revalidacion":"host=/var/run/postgresql dbname=postgres user=vec_rpt_testigo_revalidador sslmode=disable"}
JSON
docker run --rm --pull never --network none --read-only --cpus 2 --memory 4g --pids-limit 128 \
  --mount "type=bind,src=$toolchain,dst=/toolchain,readonly" \
  --mount "type=bind,src=$raiz_repo,dst=/src,readonly" \
  --mount type=bind,src=/home/alberto/go/pkg/mod,dst=/go/pkg/mod,readonly \
  --mount "type=bind,src=$base/build,dst=/scratch" \
  --tmpfs /tmp:rw,nosuid,nodev,size=256m --workdir /src \
  --env GOPROXY=off --env GOSUMDB=off --env GOTOOLCHAIN=local \
  --env GOCACHE=/scratch/cache --env GOPATH=/go \
  --entrypoint /toolchain/bin/go golang:1.25-bookworm build -mod=readonly -p 2 \
  -o /scratch/lecturas-v3 \
  deploy/postgresql/catalogos_configurables/pruebas_sql/lecturas_v3_pg18.go
docker exec -i "$contenedor" sh -c \
  'cat > /var/lib/postgresql/18/lecturas-v3 && chown postgres:postgres /var/lib/postgresql/18/lecturas-v3 && chmod 700 /var/lib/postgresql/18/lecturas-v3' \
  < "$base/build/lecturas-v3"
docker exec -i "$contenedor" sh -c \
  'cat > /tmp/dsn.json && chown postgres:postgres /tmp/dsn.json && chmod 600 /tmp/dsn.json' \
  < "$base/build/dsn.json"
docker exec --user postgres "$contenedor" /var/lib/postgresql/18/lecturas-v3
