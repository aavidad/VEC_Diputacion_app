#!/usr/bin/env bash
# Carga las proyecciones de medida únicamente en un contenedor desechable propio.
set -euo pipefail
umask 077

contenedor=${1:?Uso: cargar.sh CONTENEDOR PROPIETARIO}
propietario=${2:?Falta el valor de la etiqueta vec.propietario del contenedor}
directorio=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)

docker_local() { docker --host unix:///var/run/docker.sock "$@"; }
observado=$(docker_local inspect --format '{{index .Config.Labels "vec.propietario"}}|{{.HostConfig.AutoRemove}}|{{.HostConfig.NetworkMode}}|{{.HostConfig.RestartPolicy.Name}}' "$contenedor")
esperado="$propietario|true|none|no"
if [[ "$observado" != "$esperado" ]]; then
    printf 'clave=contenedor_banco esperado=%s observado=%s\n' "$esperado" "$observado" >&2
    exit 2
fi

for archivo in 01_ct_volumen.sql 02_ct_estados.sql 03_ct_urgencias.sql \
    04_ct_instantanea.sql 05_ct_cronologia.sql 06_bolsa_volumen.sql 07_recuentos.sql; do
    printf 'Carga: %s\n' "$archivo"
    docker_local exec -i "$contenedor" psql -X -v ON_ERROR_STOP=1 \
        -U postgres -d postgres < "$directorio/$archivo"
done
