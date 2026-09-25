#!/usr/bin/env bash
# Ensayo de CT121 (circuito del sucesor también tras una expiración
# confirmada) en PostgreSQL 18.4 desechable sobre la estructura real
# restaurada de la principal (volcado con datos sintéticos que conserva un
# aviso de sucesor tras una renuncia). Instala CT110, CT111 y CT119; después
# CT121: ROLLBACK sin rastro, UP, doble UP, DOWN que restaura exactamente las
# siete funciones, doble DOWN, UP; la lectura del antecedente del aviso del
# sucesor tras una expiración (admitida con CT121 y rechazada sin ella);
# reinicio y DOWN protegido con historia.
# Uso: probar_ct121_sucesor_tras_expiracion_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-ct121-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-ct121-$$"
datos="/dev/shm/$nombre"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" >/dev/null 2>&1 || rm -rf "$datos" 2>/dev/null || true
}
trap limpiar EXIT
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
  salida=$(cat "$@" | docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres 2>&1) || { printf '%s\n' "$salida" | tail -5 >&2; exit 1; }
  grep -q "$marca" <<<"$salida" || { echo "FALLO: falta $marca" >&2; exit 1; }
  ok "$(grep -c '^ok$' <<<"$salida") comprobaciones hasta $marca"
}

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
pruebas=$repo/deploy/postgresql/contratacion_temporal/pruebas_sql
m=$ct/000121_sucesor_tras_expiracion
echo '== CT121 sin CT119 se niega'
falla_con "$m.up.sql" 'CT121 requiere CT119'; ok 'dependencia CT119 exigida'
echo '== Cadena previa: CT110, CT111 y CT119'
for f in "$ct/000110_fase_desde_cuadro_rrhh.up.sql" "$ct/000111_plazo_respuesta_llamamiento.up.sql" "$ct/000119_continuacion_tras_expiracion.up.sql"; do
  run <"$f"
done
huella="SELECT md5(string_agg(pg_get_functiondef(oid)||coalesce(proacl::text,'')||coalesce(array_to_string(proconfig,','),''),'' ORDER BY proname)) FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND proname IN ('registrar_comunicacion_llamamiento_local_v1','registrar_respuesta_recibida_rrhh_v1','consultar_justificante_respuesta_recibida_rrhh_v1','registrar_resolucion_manual_respuesta_rrhh_v1','registrar_propuesta_formalizacion_v1','registrar_propuesta_formalizacion_v2','leer_expediente_aviso_confirmado_v1')"
admitidas="SELECT count(*) FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND prosrc LIKE '%solicitud_json->>''Respuesta'' IN (''renuncia'',''expiracion_gobernada'')%' AND proname<>'continuar_llamamiento_rrhh_v2'"
antes=$(escalar "$huella")
echo '== CT121: ROLLBACK, UP, doble UP, DOWN, doble DOWN, UP'
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | run
[[ $(escalar "$huella") == "$antes" ]] || { echo 'FALLO: ROLLBACK dejó cambios' >&2; exit 1; }; ok 'ROLLBACK sin rastro'
run <"$m.up.sql"; despues=$(escalar "$huella")
[[ $(escalar "$admitidas") == 7 ]] || { echo 'FALLO: UP no amplió las siete funciones' >&2; exit 1; }; ok 'UP amplía las siete funciones'
falla_con "$m.up.sql" 'CT121 ya instalada'; ok 'doble UP rechazado'
run <"$m.down.sql"
[[ $(escalar "$huella") == "$antes" ]] || { echo 'FALLO: DOWN no restaura las funciones' >&2; exit 1; }; ok 'DOWN restaura definición, configuración y ACL'
falla_con "$m.down.sql" 'CT121 no instalada'; ok 'doble DOWN rechazado'
echo '== Sin CT121 el antecedente de expiración no se admite'
prueba 'CT121 OK' <(printf '\\set esperado no\n'; cat "$pruebas/ct121_sucesor_tras_expiracion.sql")
run <"$m.up.sql"
[[ $(escalar "$huella") == "$despues" ]] || { echo 'FALLO: UP tras DOWN no reproduce' >&2; exit 1; }; ok 'UP tras DOWN reproduce'
echo '== Con CT121 el aviso del sucesor admite la continuación tras expiración'
prueba 'CT121 OK' <(printf '\\set esperado si\n'; cat "$pruebas/ct121_sucesor_tras_expiracion.sql")

echo '== Historia: la continuación del aviso conservado queda tras una expiración'
run <<'SQL'
BEGIN;
SET LOCAL session_replication_role = replica;
UPDATE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
   SET solicitud_json=jsonb_set(solicitud_json,'{Respuesta}','"expiracion_gobernada"'),
       material_json=jsonb_set(material_json,'{Solicitud,Respuesta}','"expiracion_gobernada"'),
       material=jsonb_set(material::jsonb,'{Solicitud,Respuesta}','"expiracion_gobernada"')::text,
       material_huella_sha256=encode(sha256(convert_to(jsonb_set(material::jsonb,'{Solicitud,Respuesta}','"expiracion_gobernada"')::text,'UTF8')),'hex'),
       recibo_json=jsonb_set(jsonb_set(recibo_json,'{Solicitud,Respuesta}','"expiracion_gobernada"'),'{IntencionSiguiente,Solicitud,Respuesta}','"expiracion_gobernada"'),
       comando_siguiente_json=jsonb_set(comando_siguiente_json,'{justificante_ref}','null'),
       estado_plazo='expirado', justificante_ref=NULL, contacto_ref='contacto:ct121:efectivo'
 WHERE resolucion_ref='resolucion:0c5fdea4-be11-4bdd-bd0b-5bc035dd9ae0';
COMMIT;
SQL
echo '== Reinicio de PostgreSQL'
docker restart "$nombre" >/dev/null
for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
sleep 1
[[ $(escalar "$huella") == "$despues" && $(escalar "$admitidas") == 7 ]] || { echo 'FALLO: funciones tras reinicio' >&2; exit 1; }; ok 'funciones ampliadas tras reinicio'
falla_con "$m.down.sql" 'reversión denegada'; ok 'DOWN con historia rechazado'
echo 'CT121 verificada'
