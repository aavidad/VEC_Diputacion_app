#!/usr/bin/env bash
# Fuente: clon local sintético. Sólo pg_dump de lectura; la migración y el
# ensayo se aplican en otro contenedor desechable PG18, sin red ni publicación.
set -euo pipefail
umask 077
if [[ ${1:-} != --clon-local && ${1:-} != --snapshot-privado || -z ${2:-} ]]; then
    printf '%s\n' 'Uso: probar_raiz_externa_123_pg18.sh --clon-local <nombre_contenedor_clon>' >&2
    exit 2
fi
fuente=$2
if [[ $1 == --clon-local ]]; then
    [[ $fuente == vec-* && $fuente != *cidonia* ]] || exit 2
else
    [[ $fuente == /dev/shm/vec-ad123-* && -r $fuente/roles.sql && -r $fuente/estado.dump ]] || exit 2
fi
raiz=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.." && pwd)
trabajo=$(mktemp -d /dev/shm/vec-ad123-ensayo-XXXXXX)
contenedor=vec-ad123-ensayo-$$
limpiar() {
    docker stop "$contenedor" >/dev/null 2>&1 || true
    docker run --rm --network none -v "$trabajo:/d" alpine sh -c 'find /d -mindepth 1 -delete' >/dev/null
    rmdir "$trabajo"
}
trap limpiar EXIT
# Nunca transportar contraseñas de roles al ensayo.
if [[ $1 == --clon-local ]]; then
    docker exec "$fuente" pg_dumpall -U postgres --roles-only --no-role-passwords > "$trabajo/roles.sql"
    docker exec "$fuente" pg_dump -U postgres -d postgres -Fc > "$trabajo/estado.dump"
else
    cp -- "$fuente/roles.sql" "$trabajo/roles.sql"
    cp -- "$fuente/estado.dump" "$trabajo/estado.dump"
fi
python3 - "$trabajo/roles.sql" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);s=p.read_text();p.write_text(s.replace('CREATE ROLE postgres;\n',''))
PY
mkdir "$trabajo/datos"
docker run --rm --network none -v "$trabajo/datos:/d" alpine chown 999:999 /d
docker run -d --rm --name "$contenedor" --network none --cpus 4 --memory 2g --pids-limit 128 \
    -e POSTGRES_HOST_AUTH_METHOD=trust -v "$trabajo/datos:/var/lib/postgresql" postgres:18.4 >/dev/null
for ((i=0;i<60;i++)); do
    if docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1; then break; fi
    sleep 1
done
docker exec "$contenedor" pg_isready -U postgres >/dev/null
docker exec -i "$contenedor" psql -U postgres -d postgres -X -q -v ON_ERROR_STOP=1 < "$trabajo/roles.sql" > "$trabajo/roles.log" 2>&1
docker exec -i "$contenedor" pg_restore -U postgres -d postgres --exit-on-error < "$trabajo/estado.dump" > "$trabajo/restaurar.log" 2>&1
if [[ ${3:-} == --antes-aut21-ad122 && -n ${4:-} ]]; then
    for sql in deploy/postgresql/autorizacion/migraciones/000021_clausura_externa_tipos_temporales.up.sql deploy/postgresql/autorizacion_atestada_v3/migraciones/000122_clausura_externa_tipos_temporales.up.sql; do
        git -C "$raiz" show "$4:$sql" > "$trabajo/dependencia.sql"
        docker exec -i "$contenedor" psql -U postgres -d postgres -X -q -v ON_ERROR_STOP=1 < "$trabajo/dependencia.sql"
    done
fi
docker exec -i "$contenedor" psql -U postgres -d postgres -X -q -v ON_ERROR_STOP=1 \
    < "$raiz/deploy/postgresql/autorizacion_atestada_v3/migraciones/000123_raiz_externa_propia.up.sql"
docker exec -i "$contenedor" psql -U postgres -d postgres -X -q -v ON_ERROR_STOP=1 \
    < "$raiz/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad3_123_raiz_externa.sql"
printf '%s\n' ENSAYO-OK
