#!/usr/bin/env bash
# Ensayo de CT121 con la cadena real tras una expiración, en PostgreSQL 18.4
# desechable sobre la estructura real restaurada (volcado con datos
# sintéticos). Instala CT110, CT111 y CT119; rebobina el expediente B a su
# primer aviso con dobles de las fachadas AD3 (ct121_circuito_fixture.sql);
# registra con las funciones reales el contacto y la expiración (CT111) y la
# continuación (CT119), y comprueba que sin CT121 el aviso del sucesor se
# rechaza. Instala CT121 sobre esa historia (CT119 ya no se retira) y recorre
# aviso, respuesta, justificante, resolución y propuesta del sucesor, con
# replays; reinicia PostgreSQL, repite los mismos materiales (mismo recibo,
# ninguna fila nueva) y comprueba que el DOWN de CT121 se niega con historia.
# VEC_CT121C_CONSERVAR=1 deja el contenedor para depurar (borrarlo a mano).
# Uso: probar_ct121_circuito_sucesor_expiracion_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct121c-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct121c-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
[[ -n ${VEC_CT121C_CONSERVAR:-} ]] || trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" postgres:18.4 >/dev/null
esperar() {
  for _ in $(seq 1 240); do
    if docker logs "$nombre" 2>&1 | grep -q 'PostgreSQL init process complete' && docker exec "$nombre" pg_isready -q -U postgres; then return 0; fi
    sleep 0.5
  done
  return 1
}
esperar
if [[ -n $(docker inspect --format '{{range .Mounts}}{{if eq .Type "volume"}}{{.Name}}{{end}}{{end}}' "$nombre") ]]; then
  echo 'volumen anónimo inesperado' >&2; exit 65
fi
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }
ok() { printf 'OK %s\n' "$1"; }
falla_con() {
  if salida=$(run <"$1" 2>&1); then echo "FALLO: se esperaba rechazo de $(basename "$1")" >&2; exit 1; fi
  grep -q "$2" <<<"$salida" || { echo "FALLO: rechazo inesperado de $(basename "$1"): $salida" >&2; exit 1; }
}
prueba() {
  local marca=$1 salida; shift
  salida=$(docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres <"$1" 2>&1) || { printf '%s\n' "$salida" | tail -8 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
echo '== Cadena previa: CT110, CT111 y CT119'
for f in 000110_fase_desde_cuadro_rrhh 000111_plazo_respuesta_llamamiento 000119_continuacion_tras_expiracion; do
  run <"$ct/$f.up.sql"
done
echo '== Fixture: dobles AD3 y expediente B rebobinado a su primer aviso'
prueba 'fixture CT121 OK' "$pruebas/ct121_circuito_fixture.sql"
echo '== Sin CT121: expiración y continuación reales; el aviso del sucesor se rechaza'
prueba 'CT121 antecedente OK' "$pruebas/ct121_circuito_antecedente.sql"
echo '== CT121 sobre esa historia'
run <"$ct/000121_sucesor_tras_expiracion.up.sql"
falla_con "$ct/000119_continuacion_tras_expiracion.down.sql" 'CT121 sigue instalada'; ok 'DOWN de CT119 rechazado con CT121 instalada'
echo '== Circuito real del sucesor tras la expiración'
prueba 'CT121 circuito OK' "$pruebas/ct121_circuito_sucesor.sql"

echo '== Reinicio de PostgreSQL'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
prueba 'CT121 reinicio OK' "$pruebas/ct121_circuito_reinicio.sql"
falla_con "$ct/000121_sucesor_tras_expiracion.down.sql" 'reversión denegada'; ok 'DOWN de CT121 rechazado con historia'
echo 'CT121 circuito verificado'
