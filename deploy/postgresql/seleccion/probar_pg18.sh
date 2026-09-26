#!/usr/bin/env bash
# Ensayo de Selección 000001 con AD3-89 y AD3-90 en PostgreSQL 18.4 desechable
# sobre la estructura real restaurada de la principal (volcado sintético).
# 1. Lleva el volcado al estado de main (AD3-82…88, CT y Bolsa que no trae).
# 2. Instala en orden roles de Selección, AD3-89, AD3-90 y Selección 000001:
#    cada una con ROLLBACK sin rastro, UP, detección y doble UP rechazado.
# 3. Ida y vuelta DOWN/UP de las tres sin historia (núcleo AD3 idéntico).
# 4. ACL con roles reales: el LOGIN de ensayo (miembro solo del ejecutor) no
#    lee tablas ni llama funciones internas ni la fachada V3; otro LOGIN no
#    ejecuta nada; la fachada real rechaza material no atestado.
# 5. Recorrido con dobles de las fachadas: borrador, idempotencia, versión
#    obsoleta, plazo, requisito, ajena, presentación, justificante, RRHH.
# 6. Reinicio de PostgreSQL: la repetición devuelve el mismo recibo, y los
#    DOWN con historia se niegan.
# Uso: probar_pg18.sh GLOBALS_SQL VOLCADO_PG_DUMP
# El contenedor usa --rm, sin red ni volúmenes anónimos; sus datos viven en
# /dev/shm/vec-pg-seleccion-<pid> y se borran al terminar.
set -Eeuo pipefail

repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
globales=${1:?falta el volcado de roles (pg_dumpall --globals-only)}
volcado=${2:?falta el volcado de la base (pg_dump -Fc)}
[[ -s $globales && -s $volcado ]] || { echo 'Faltan los volcados' >&2; exit 2; }

nombre="vec-pg-seleccion-$$"
datos="/dev/shm/$nombre"
socket="/dev/shm/$nombre-socket"
limpiar() {
  docker rm -f "$nombre" >/dev/null 2>&1 || true
  docker run --rm -v "/dev/shm:/limpiar" postgres:18.4 rm -rf "/limpiar/$nombre" "/limpiar/$nombre-socket" >/dev/null 2>&1 || rm -rf "$datos" "$socket" 2>/dev/null || true
}
trap limpiar EXIT
mkdir -p "$datos" "$socket"
chmod 1777 "$socket"
docker run -d --rm --network none --name "$nombre" -e POSTGRES_HOST_AUTH_METHOD=trust \
  -v "$datos:/var/lib/postgresql" -v "$socket:/var/run/postgresql" postgres:18.4 >/dev/null
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
run() { docker exec -i "$nombre" psql -X -q -o /dev/null -v ON_ERROR_STOP=1 -U postgres -d postgres; }
escalar() { docker exec "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
como() { docker exec -i "$nombre" psql -X -At -v ON_ERROR_STOP=1 -U "$1" -d postgres; }
ok() { printf 'OK %s\n' "$1"; }
falla() { echo "FALLO: $1" >&2; exit 1; }
falla_con() { # $1 fichero, $2 texto esperado en el error
  local salida
  if salida=$(run <"$1" 2>&1); then falla "se esperaba rechazo de $(basename "$1")"; fi
  grep -q "$2" <<<"$salida" || falla "rechazo inesperado de $(basename "$1"): $salida"
}
reiniciar() {
  docker restart "$nombre" >/dev/null
  for _ in $(seq 1 120); do docker exec "$nombre" pg_isready -q -U postgres && break; sleep 0.5; done
  sleep 1
}
[[ $(escalar 'SHOW server_version') == 18.4* ]] || { echo 'PostgreSQL 18.4 no disponible' >&2; exit 2; }

echo '== Restaurando roles y estructura real'
docker exec -i "$nombre" psql -X -q -U postgres -d postgres <"$globales" >/dev/null 2>&1 || true
docker exec -i "$nombre" pg_restore -U postgres -d postgres <"$volcado" >/dev/null 2>&1 || true
[[ $(escalar "SELECT to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NOT NULL") == t ]] || { echo 'Restauración incompleta' >&2; exit 2; }

ad3=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
ct=$repo/deploy/postgresql/contratacion_temporal/migraciones
bolsa=$repo/deploy/postgresql/bolsa_llamamientos/migraciones
sel=$repo/deploy/postgresql/seleccion
echo '== Estado de main: migraciones que el volcado no trae'
for m in 000082 000083 000084 000085 000086 000087 000088; do run <"$(ls "$ad3"/${m}_*.up.sql)"; done; ok 'AD3-82…88'
for m in 000110 000111 000113 000115 000116 000118 000119 000120 000121; do run <"$(ls "$ct"/${m}_*.up.sql)"; done; ok 'CT110…121 (sin CT117)'
for m in 000010 000019 000021 000022 000023 000024 000025 000026 000028 000029 000030 000031 000032 000033 000034 000035 000037 000039 000040 000041; do
  run <"$(ls "$bolsa"/${m}_*.up.sql)"
done; ok 'Bolsa 010, 019 y 021…041'

nucleo="vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
huella_ad3() {
  escalar "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef((SELECT oid FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check')))||(SELECT nspacl::text FROM pg_namespace WHERE nspname='vec_autorizacion_atestada_v3')"
}
declare -A detecta=(
  [roles]="SELECT count(*)=3 FROM pg_roles WHERE rolname IN ('vec_seleccion_propietario','vec_seleccion_migrador','vec_seleccion_ejecutor')"
  [ad3_89]="SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
  [ad3_90]="SELECT to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL"
  [sel_1]="SELECT to_regclass('vec_seleccion.solicitud') IS NOT NULL"
)
declare -A fichero=(
  [ad3_89]="$ad3/000089_consumidor_solicitud_propia_seleccion"
  [ad3_90]="$ad3/000090_consumidor_consulta_solicitudes_seleccion"
  [sel_1]="$sel/migraciones/000001_solicitudes_participacion"
)
for m in roles ad3_89 ad3_90 sel_1; do [[ $(escalar "${detecta[$m]}") == f ]] || falla "$m presente antes de instalar"; done
ok 'nada de Selección presente al empezar'
h0=$(huella_ad3)

echo '== Instalación en orden, con ROLLBACK previo sin rastro'
run <"$sel/roles_up.sql"; [[ $(escalar "${detecta[roles]}") == t ]] || falla 'roles'
falla_con "$sel/roles_up.sql" 'ya existen roles'
ok 'roles de Selección (doble ejecución rechazada)'
for m in ad3_89 ad3_90 sel_1; do
  sed 's/^COMMIT;$/ROLLBACK;/' "${fichero[$m]}.up.sql" | run
  [[ $(escalar "${detecta[$m]}") == f ]] || falla "ROLLBACK de $m dejó rastro"
  run <"${fichero[$m]}.up.sql"
  [[ $(escalar "${detecta[$m]}") == t ]] || falla "$m no detectada tras UP"
  if run <"${fichero[$m]}.up.sql" >/dev/null 2>&1; then falla "doble UP de $m aceptado"; fi
  ok "$m: ROLLBACK sin rastro, UP, detección y doble UP rechazado"
done

echo '== Ida y vuelta DOWN/UP sin historia'
run <"${fichero[sel_1]}.down.sql"; run <"${fichero[ad3_90]}.down.sql"; run <"${fichero[ad3_89]}.down.sql"
[[ $(huella_ad3) == "$h0" ]] || falla 'el núcleo AD3 no vuelve a su estado previo'
for m in ad3_89 ad3_90 sel_1; do run <"${fichero[$m]}.up.sql"; done
ok 'DOWN/UP de Selección 000001, AD3-90 y AD3-89: núcleo, audiencias y ACL idénticos'
falla_con "${fichero[ad3_89]}.down.sql" 'AD3-90 sigue instalada'
falla_con "${fichero[ad3_90]}.down.sql" 'Selección 000001 sigue instalada'
ok 'DOWN fuera de orden rechazados'

echo '== ACL con roles reales'
escalar "CREATE ROLE vec_prueba_seleccion LOGIN; GRANT vec_seleccion_ejecutor TO vec_prueba_seleccion; CREATE ROLE vec_prueba_ajeno LOGIN;" >/dev/null
for consulta in 'SELECT count(*) FROM vec_seleccion.solicitud' 'SELECT count(*) FROM vec_seleccion.convocatoria_publicada' \
  "SELECT vec_seleccion.convocatoria_vigente_interna_v1('x')" \
  "SELECT * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_solicitud_propia_seleccion_v3_atestada('\\x00','\\x00','\\x00','\\x00',1,1,'\\x00','\\x00','\\x00','\\x00')"; do
  if como vec_prueba_seleccion <<<"$consulta" >/dev/null 2>&1; then falla "el ejecutor accede a: $consulta"; fi
done
if como vec_prueba_ajeno <<<'SELECT vec_seleccion.convocatorias_publicadas_v1()' >/dev/null 2>&1; then falla 'un LOGIN ajeno ejecuta Selección'; fi
[[ $(como vec_prueba_seleccion <<<'SELECT vec_seleccion.convocatorias_publicadas_v1()') == '[]' ]] || falla 'el ejecutor no lee convocatorias'
salida=$(como vec_prueba_seleccion <<<"SET default_transaction_isolation='serializable';
SELECT * FROM vec_seleccion.publicar_convocatoria_v1('proceso:publico:ensayo-acl','Ensayo',now()-interval '1 day',now()+interval '1 day','{\"turnos\":[],\"requisitos\":[],\"baremo\":{},\"numeracion\":{\"patron\":\"{anio}/SOL-{numero}\",\"ancho\":6}}',repeat('e',64),'2026-09-01T00:00:00Z');
SELECT * FROM vec_seleccion.guardar_borrador_propio_v1('per_AAAAAAAAAAAAAAAAAAAAAA','proceso:publico:ensayo-acl',1,0,'clave-acl-001',repeat('a',64),'sol_AAAAAAAAAAAAAAAAAAAAAAAAAA','libre','k','\\x000102030405060708090a0b','\\x00112233445566778899aabbccddeeff00',repeat('b',64),'***5678*','[]','[]',0,
 convert_to('{\"operacion\":\"seleccion.solicitudes_propias.guardar_borrador\",\"efecto_ref\":\"mis-solicitudes:per_AAAAAAAAAAAAAAAAAAAAAA\",\"audiencia_consumo\":\"vec_seleccion.solicitudes_propias.guardar_borrador.v1\"}','UTF8'),
 convert_to('{\"accion\":\"seleccion.solicitudes_propias.guardar_borrador\",\"recurso_ref\":\"mis-solicitudes:per_AAAAAAAAAAAAAAAAAAAAAA\",\"modulo_id\":\"seleccion\",\"finalidad\":\"gestion_solicitudes_propias\",\"tipo_recurso\":\"solicitudes_propias_seleccion\",\"campos_permitidos\":[],\"obligaciones\":[]}','UTF8'),
 '\\x00',convert_to('{\"persona_ref\":\"per_AAAAAAAAAAAAAAAAAAAAAA\"}','UTF8'),1,1,'\\x00','\\x00','\\x00','\\x00');" 2>&1 || true)
grep -Eq 'ERROR' <<<"$salida" || falla "la fachada real aceptó material no atestado: $salida"
[[ $(escalar 'SELECT count(*) FROM vec_seleccion.solicitud') == 0 ]] || falla 'material rechazado dejó rastro'
ok 'ejecutor sin tablas, internas ni fachada V3; LOGIN ajeno sin acceso; material no atestado rechazado sin rastro'

echo '== Recorrido con dobles de AD3-89/AD3-90'
run <"$sel/pruebas_sql/dobles_ad3_89_90.sql"
salida=$(como vec_prueba_seleccion <"$sel/pruebas_sql/recorrido_solicitudes.sql" 2>&1) || { printf '%s\n' "$salida" | tail -5 >&2; exit 1; }
grep -q '^fin_recorrido$' <<<"$salida" || falla 'recorrido incompleto'
ok "$(grep -c '^ok$' <<<"$salida") comprobaciones del recorrido"
[[ $(escalar "SELECT count(*) FILTER (WHERE tipo='listado')||'/'||count(*) FILTER (WHERE tipo='detalle') FROM vec_seleccion.acceso_rrhh") == 3/1 ]] || falla 'accesos RRHH no auditados'
[[ $(escalar 'SELECT count(*) FROM vec_seleccion.outbox') == 2 ]] || falla 'outbox'
ok 'accesos de RRHH auditados y dos eventos en la outbox'
if escalar "SET ROLE vec_seleccion_propietario; UPDATE vec_seleccion.solicitud SET creada_en=now()" >/dev/null 2>&1; then falla 'historia mutable'; fi
if escalar "SET ROLE vec_seleccion_propietario; DELETE FROM vec_seleccion.presentacion" >/dev/null 2>&1; then falla 'presentación borrable'; fi
ok 'historia de solo adición (UPDATE y DELETE rechazados)'
antes=$(escalar "SELECT string_agg(numero_justificante||'|'||recibo_ref||'|'||presentada_en,',' ORDER BY presentacion_id) FROM vec_seleccion.presentacion")

echo '== Reinicio de PostgreSQL'
reiniciar
for m in ad3_89 ad3_90 sel_1; do [[ $(escalar "${detecta[$m]}") == t ]] || falla "$m no detectada tras reiniciar"; done
[[ $(escalar "SELECT string_agg(numero_justificante||'|'||recibo_ref||'|'||presentada_en,',' ORDER BY presentacion_id) FROM vec_seleccion.presentacion") == "$antes" ]] || falla 'presentaciones alteradas tras reiniciar'
repetida=$(como vec_prueba_seleccion <<SQL
SET default_transaction_isolation='serializable';
SELECT reutilizada||'|'||numero_justificante||'|'||recibo_ref||'|'||presentada_en FROM vec_seleccion.presentar_solicitud_propia_v1('per_AAAAAAAAAAAAAAAAAAAAAA',
 (SELECT solicitud_ref FROM jsonb_to_recordset(vec_seleccion.listar_solicitudes_propias_v1('per_AAAAAAAAAAAAAAAAAAAAAA',
   convert_to('{"operacion":"seleccion.solicitudes_propias.consultar","efecto_ref":"mis-solicitudes:per_AAAAAAAAAAAAAAAAAAAAAA"}','UTF8'),
   convert_to('{"accion":"seleccion.solicitudes_propias.consultar","recurso_ref":"mis-solicitudes:per_AAAAAAAAAAAAAAAAAAAAAA"}','UTF8'),
   '\x00',convert_to('{"persona_ref":"per_AAAAAAAAAAAAAAAAAAAAAA"}','UTF8'),1,1,'\x00','\x00','\x00','\x00')) AS x(solicitud_ref text)),
 4,'clave-pres-A4',encode(sha256(convert_to('p5','UTF8')),'hex'),'recibo:seleccion-presentacion:'||encode(sha256(convert_to((SELECT 'x') ,'UTF8')),'hex'),
 convert_to('{"operacion":"seleccion.solicitudes_propias.presentar","efecto_ref":"mis-solicitudes:per_AAAAAAAAAAAAAAAAAAAAAA"}','UTF8'),
 convert_to('{"accion":"seleccion.solicitudes_propias.presentar","recurso_ref":"mis-solicitudes:per_AAAAAAAAAAAAAAAAAAAAAA"}','UTF8'),
 '\x00',convert_to('{"persona_ref":"per_AAAAAAAAAAAAAAAAAAAAAA"}','UTF8'),1,1,'\x00','\x00','\x00','\x00');
SQL
)
primera=$(escalar "SELECT 'true|'||numero_justificante||'|'||recibo_ref||'|'||presentada_en FROM vec_seleccion.presentacion ORDER BY presentacion_id LIMIT 1")
[[ $(tail -1 <<<"$repetida") == "$primera" ]] || falla "la repetición tras reiniciar no devuelve el mismo recibo: $repetida / $primera"
[[ $(escalar 'SELECT count(*) FROM vec_seleccion.presentacion') == 2 ]] || falla 'la repetición duplicó la presentación'
ok 'tras reiniciar: mismas presentaciones y la repetición devuelve el mismo justificante y recibo'
falla_con "${fichero[sel_1]}.down.sql" 'no admitido con solicitudes'
ok 'DOWN de Selección 000001 rechazado con historia'
echo 'ENSAYO SELECCION COMPLETO'
