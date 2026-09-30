#!/usr/bin/env bash
set -euo pipefail
umask 077

# Testigo de un solo uso. Todas las rutas privadas se reciben en un JSON externo.
# Cada ejecución exige un directorio y un contenedor propios nuevos.
raiz_repo=$(git rev-parse --show-toplevel)
[[ $# -eq 3 ]]
config=$1
base=$2
contenedor=$3
python3 - "$config" <<'PY'
import os,pathlib,stat,subprocess,sys
ruta=pathlib.Path(sys.argv[1])
if not ruta.is_absolute() or not ruta.is_file() or ruta.is_symlink():
    raise SystemExit('configuración privada inválida')
if str(ruta) != str(ruta.resolve(strict=True)):
    raise SystemExit('ruta de configuración no canónica')
archivo=os.stat(ruta,follow_symlinks=False)
padre=os.stat(ruta.parent,follow_symlinks=False)
if archivo.st_uid!=os.getuid() or stat.S_IMODE(archivo.st_mode)!=0o600:
    raise SystemExit('configuración sin propietario o modo 0600')
if padre.st_uid!=os.getuid() or stat.S_IMODE(padre.st_mode)!=0o700:
    raise SystemExit('directorio de configuración sin propietario o modo 0700')
for ancestro in (ruta.parent,*ruta.parent.parents):
    if os.path.lexists(ancestro/'.git') or (
        (ancestro/'HEAD').is_file() and (ancestro/'objects').is_dir()
    ):
        raise SystemExit('configuración dentro de un repositorio Git')
entorno={k:v for k,v in os.environ.items() if not k.startswith('GIT_')}
entorno.update({'LC_ALL':'C','GIT_CONFIG_NOSYSTEM':'1',
                'GIT_CONFIG_GLOBAL':'/dev/null',
                'GIT_DISCOVERY_ACROSS_FILESYSTEM':'1'})
diagnostico=subprocess.run(
    ['git','-c','safe.directory=*','-C',str(ruta.parent),'rev-parse','--git-dir'],
    env=entorno,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
if diagnostico.returncode==0:
    raise SystemExit('configuración dentro de un repositorio Git')
if diagnostico.returncode!=128 or not diagnostico.stderr.startswith(
    'fatal: not a git repository'
):
    raise SystemExit('no se pudo descartar un repositorio Git')
PY
[[ $base =~ ^/dev/shm/vec-rpt-testigo-v3-[a-z0-9-]+$ ]]
[[ $contenedor =~ ^vec-rpt-testigo-v3-[a-z0-9-]+$ ]]
[[ $(basename "$base") == "$contenedor" ]]
[[ ! -e $base ]]
mkdir -m 700 "$base"
mkdir -m 700 "$base/build" "$base/source" "$base/inputs" "$base/docker-config"

configurar() {
  python3 - "$config" "$1" <<'PY'
import json,sys
with open(sys.argv[1],encoding='utf-8') as fuente:
    dato=json.load(fuente).get(sys.argv[2])
if not isinstance(dato,str) or not dato or any(c in dato for c in ('\0','\r','\n')):
    raise SystemExit('configuración privada inválida')
print(dato)
PY
}
revisado=$(configurar reviewed_sha)
archivo=$(configurar h1_path)
lista_h3=$(configurar h3_list_path)
lista_h4=$(configurar h4_list_path)
lista_h6=$(configurar h6_list_path)
toolchain=$(configurar toolchain_path)
modcache=$(configurar modcache_path)
huella_h1=d1c2e38a85f872e83b6ba9eca6b660b7a5d2ff5d39d4aca612a81ec30a496e5b
huella_h3=28fcbcd55d8f0c6c5112f8381460273764b9145e1ce32bc77f1fe58a243dd8c5
huella_h4=8b816ea3c1d8a06b25c3822327c6d0053bb4f14daad49acfa785786f5bc4e5dd
huella_h6=0bbdaf8d55b7facbe66d89139c4416f5b82544302f4cfb8428f2f35eb0f46322
fuente_h6=742127f5feb8a45985556c59e6d043bf6b237518

[[ $revisado =~ ^[0-9a-f]{40}$ ]]
[[ $(git -C "$raiz_repo" rev-parse HEAD) == "$revisado" ]]
[[ $(git -C "$raiz_repo" hash-object "$raiz_repo/deploy/postgresql/catalogos_configurables/pruebas_sql/lecturas_v3_pg18.sh") == $(git -C "$raiz_repo" rev-parse "$revisado:deploy/postgresql/catalogos_configurables/pruebas_sql/lecturas_v3_pg18.sh") ]]
git -C "$raiz_repo" merge-base --is-ancestor 90ec895b8c007ae392d34c83456900412f9dbaa4 "$revisado"
for ruta in deploy/postgresql/catalogos_configurables/migraciones/000001_autoridad_categorias.up.sql \
            deploy/postgresql/catalogos_configurables/migraciones/000002_lecturas_nominales.up.sql \
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000117_lecturas_categorias_rpt.up.sql; do
  [[ $(git -C "$raiz_repo" rev-parse "$revisado:$ruta") == $(git -C "$raiz_repo" rev-parse "90ec895b8c007ae392d34c83456900412f9dbaa4:$ruta") ]]
done
[[ -f $archivo && ! -L $archivo && -d $toolchain && -d $modcache ]]
printf '%s  %s\n' "$huella_h1" "$archivo" | sha256sum -c --status

# Las listas privadas se copian antes de usarlas; una huella distinta o un
# archivo repetido cierra el ensayo antes de arrancar PostgreSQL.
cp -- "$lista_h3" "$base/inputs/h3.txt"
cp -- "$lista_h4" "$base/inputs/h4.txt"
cp -- "$lista_h6" "$base/inputs/h6.txt"
printf '%s  %s\n' "$huella_h3" "$base/inputs/h3.txt" | sha256sum -c --status
printf '%s  %s\n' "$huella_h4" "$base/inputs/h4.txt" | sha256sum -c --status
printf '%s  %s\n' "$huella_h6" "$base/inputs/h6.txt" | sha256sum -c --status
declare -A visto=()
listas=("$base/inputs/h3.txt" "$base/inputs/h4.txt" "$base/inputs/h6.txt")
esperadas=(8 9 21)
for indice in 0 1 2; do
  cuenta=0
  while IFS= read -r ruta || [[ -n $ruta ]]; do
    [[ $ruta =~ ^deploy/postgresql/[a-zA-Z0-9_./-]+\.sql$ && $ruta != *..* ]]
    [[ -z ${visto[$ruta]+x} ]]
    visto[$ruta]=1
    git -C "$raiz_repo" cat-file -e "$fuente_h6:$ruta"
    cuenta=$((cuenta+1))
  done < "${listas[$indice]}"
  [[ $cuenta -eq ${esperadas[$indice]} ]]
done
[[ ${#visto[@]} -eq 38 ]]

[[ -S /var/run/docker.sock ]]
docker_local() {
  env -u DOCKER_HOST -u DOCKER_CONTEXT -u DOCKER_CONFIG -u DOCKER_TLS_VERIFY -u DOCKER_CERT_PATH \
    docker --config "$base/docker-config" --host unix:///var/run/docker.sock "$@"
}
if docker_local container inspect "$contenedor" >/dev/null 2>&1; then
  echo 'RPT-V3-FALLO: contenedor propio ya existente' >&2
  exit 1
fi
git -C "$raiz_repo" archive "$revisado" | tar -x -C "$base/source"
sha256sum "$archivo" > "$base/build/h1_sha256.txt"
tar -xzf "$archivo" -C "$base"
docker_local run -d --pull never --name "$contenedor" --network none --memory 4g --cpus 2 \
  --pids-limit 128 --shm-size 256m \
  --mount "type=bind,src=$base,dst=/var/lib/postgresql" \
  --tmpfs /tmp:rw,nosuid,nodev,size=32m \
  --tmpfs /var/run/postgresql:rw,nosuid,nodev,size=16m,uid=999,gid=999 \
  postgres:18.4 >/dev/null
for _ in {1..100}; do
  if docker_local exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres; then break; fi
  sleep 0.2
done
docker_local exec "$contenedor" pg_isready -q -h /var/run/postgresql -U postgres

aplicar() {
  docker_local exec -i --user postgres "$contenedor" \
    psql -X -q -v ON_ERROR_STOP=1 -d postgres < "$1" >/dev/null
}
aplicar_git() {
  git -C "$raiz_repo" show "$fuente_h6:$1" |
    docker_local exec -i --user postgres "$contenedor" \
      psql -X -q -v ON_ERROR_STOP=1 -d postgres >/dev/null
}
for lista in "$base/inputs/h3.txt" "$base/inputs/h4.txt" "$base/inputs/h6.txt"; do
  while IFS= read -r ruta || [[ -n $ruta ]]; do
    aplicar_git "$ruta"
  done < "$lista"
done
aplicar "$base/source/deploy/postgresql/catalogos_configurables/roles_up.sql"
aplicar "$base/source/deploy/postgresql/catalogos_configurables/migraciones/000001_autoridad_categorias.up.sql"
aplicar "$base/source/deploy/postgresql/catalogos_configurables/migraciones/000002_lecturas_nominales.up.sql"
aplicar "$base/source/deploy/postgresql/autorizacion_atestada_v3/migraciones/000117_lecturas_categorias_rpt.up.sql"

docker_local exec --user postgres "$contenedor" psql -X -qAt -d postgres -c \
  "SELECT datacl FROM pg_catalog.pg_database WHERE datname=current_database()" \
  > "$base/build/datacl_antes.txt"
inventario_acl() {
  docker_local exec --user postgres "$contenedor" psql -X -qAt -d postgres -c "
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
docker_local exec -i --user postgres "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -d postgres >/dev/null <<'SQL'
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
docker_local exec --user postgres "$contenedor" psql -X -qAt -d postgres -c \
  "SELECT datacl FROM pg_catalog.pg_database WHERE datname=current_database()" \
  > "$base/build/datacl_despues.txt"
inventario_acl > "$base/build/acl_despues.txt"
echo 'RPT-V3-ACL-CLON-ADAPTADA'

cat > "$base/build/dsn.json" <<'JSON'
{"admin":"host=/var/run/postgresql dbname=postgres user=postgres sslmode=disable","contexto":"host=/var/run/postgresql dbname=postgres user=vec_rpt_testigo_contexto sslmode=disable","runtime":"host=/var/run/postgresql dbname=postgres user=vec_rpt_testigo_ct_ejecutor sslmode=disable","revalidacion":"host=/var/run/postgresql dbname=postgres user=vec_rpt_testigo_revalidador sslmode=disable"}
JSON
docker_local run --rm --pull never --network none --read-only --cpus 2 --memory 4g --pids-limit 128 \
  --mount "type=bind,src=$toolchain,dst=/toolchain,readonly" \
  --mount "type=bind,src=$base/source,dst=/src,readonly" \
  --mount "type=bind,src=$modcache,dst=/go/pkg/mod,readonly" \
  --mount "type=bind,src=$base/build,dst=/scratch" \
  --tmpfs /tmp:rw,nosuid,nodev,size=256m --workdir /src \
  --env GOPROXY=off --env GOSUMDB=off --env GOTOOLCHAIN=local \
  --env GOCACHE=/scratch/cache --env GOPATH=/go \
  --entrypoint /bin/sh golang:1.25-bookworm -ceu \
  '/toolchain/bin/go mod verify && /toolchain/bin/go build -buildvcs=false -mod=readonly -p 2 -o /scratch/lecturas-v3 deploy/postgresql/catalogos_configurables/pruebas_sql/lecturas_v3_pg18.go'
docker_local exec -i "$contenedor" sh -c \
  'umask 077; cat > /var/lib/postgresql/18/lecturas-v3 && chown postgres:postgres /var/lib/postgresql/18/lecturas-v3 && chmod 700 /var/lib/postgresql/18/lecturas-v3' \
  < "$base/build/lecturas-v3"
docker_local exec -i "$contenedor" sh -c \
  'umask 077; cat > /tmp/dsn.json && chown postgres:postgres /tmp/dsn.json && chmod 600 /tmp/dsn.json' \
  < "$base/build/dsn.json"
docker_local exec --user postgres "$contenedor" /var/lib/postgresql/18/lecturas-v3
