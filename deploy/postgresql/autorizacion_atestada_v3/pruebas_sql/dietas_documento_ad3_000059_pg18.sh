#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de AD3-59 sobre la PREIMAGEN REAL del
# núcleo hasta AD3-52: una base VEC restaurada, no un stub. Recorre los dos
# órdenes con AD3-53 (59→53 y 53→59) en contenedores separados y sin red.
#
# Uso: dietas_documento_ad3_000059_pg18.sh BASE.dump ROLES.sql
#   BASE.dump  pg_dump -Fc de una base VEC con AD3-52 instalada, sin AD3-53
#              ni AD3-59 (clon sintético; nunca datos reales).
#   ROLES.sql  pg_dumpall --roles-only de la misma instancia; las contraseñas
#              se descartan antes de entrar al contenedor.
# AD3-53 se lee de VEC_AD3_53_REF (por defecto la rama de Cronos) solo para el
# entorno del ensayo; no se copia a esta rama.
#
# Comprueba, en cada orden: ROLLBACK sin rastro; COMMIT; segunda aplicación
# rechazada sin cambios; un consumo CT real repetido antes y después (mismo
# recibo, sin filas nuevas); las cinco fachadas de Dietas y las cuatro de D7
# superan guarda, prevalidación y ligadura con su login exacto y se detienen
# en la clave de capacidad; un login con dos grupos y un login de otro módulo
# se rechazan en la guarda. Los contenedores se borran al salir.
set -Eeuo pipefail
base=${1:?falta BASE.dump (pg_dump -Fc con AD3-52)}
roles=${2:?falta ROLES.sql (pg_dumpall --roles-only)}
[[ -s $base && -s $roles ]] || { echo 'Base o roles vacíos' >&2; exit 2; }
dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(CDPATH='' cd -- "$dir/../../../.." && pwd)
mig=$repo/deploy/postgresql/autorizacion_atestada_v3/migraciones
m59=$mig/000059_consumidor_documento_dietas.up.sql
ref53=${VEC_AD3_53_REF:-origin/trabajo/cronos-montaje-20260925}
tmp=$(mktemp -d)
contenedores=()
limpiar() {
  for c in "${contenedores[@]}"; do docker rm -f "$c" >/dev/null 2>&1 || true; done
  rm -rf -- "$tmp"
}
trap limpiar EXIT INT TERM
git -C "$repo" show "$ref53:deploy/postgresql/autorizacion_atestada_v3/migraciones/000053_consumidores_cronos_empleado.up.sql" > "$tmp/53.sql"
sed -E "s/ PASSWORD '[^']*'//; s/ PASSWORD [^ ;]+//" "$roles" | grep -vE '^(CREATE|ALTER) ROLE postgres( |;)' > "$tmp/roles.sql"
if grep -qi 'password' "$tmp/roles.sql"; then echo 'No se pudieron retirar las contraseñas' >&2; exit 2; fi

fallo() { echo "FALLO [$orden]: $*" >&2; exit 1; }
ok() { echo "OK [$orden] $*"; }
sql() { docker exec -i "$C" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
val() { docker exec "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
nucleo="vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
firma() {
  val "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef(c.oid))||
   (SELECT md5(string_agg(p.oid::regprocedure::text||coalesce(p.proacl::text,'')||md5(p.prosrc),'|' ORDER BY p.oid::regprocedure::text))
      FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace)
   FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
}
consumos() {
  val "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)||'|'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.atestacion_decision_v3)||'|'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)"
}
# Llama a una fachada con el login indicado y devuelve la fila o el mensaje.
llamar() {
  { docker exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U "$1" -d postgres 2>&1 || true; } <<SQL | sed -n 's/^.*ERROR: *//p;/|/p' | head -1
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM prueba59.consumir_$2('$3','$4');
COMMIT;
SQL
}
esperar() {
  for _ in $(seq 1 120); do
    if docker exec "$C" psql -X -qAt -U postgres -c 'SELECT 1' >/dev/null 2>&1; then
      sleep 0.5
      docker exec "$C" psql -X -qAt -U postgres -c 'SELECT 1' >/dev/null 2>&1 && return 0
    fi
    sleep 0.3
  done
  fallo 'PostgreSQL no disponible'
}
aplicar53() {
  sql -o /dev/null -c 'CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOBYPASSRLS; CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOBYPASSRLS'
  sql -o /dev/null < "$tmp/53.sql" || fallo 'AD3-53'
  ok "AD3-53 de $ref53 aplicada"
}

# fachada|llamador|operación|audiencia|módulo|recurso|finalidad
fachadas=(
 'registrar_y_consumir_dietas_documento_v3_atestada|dietas|dietas.borrador.propio.editar|vec_dietas.borrador_propio.editar.v1|dietas|comision_borrador|editar_borrador_propio'
 'registrar_y_consumir_dietas_documento_consulta_v3_atestada|dietas|dietas.documento.propio.consultar|vec_dietas.documento_propio.consultar.v1|dietas|comision_borrador|consultar_documento_propio_dietas'
 'registrar_y_consumir_dietas_prelectura_v3_atestada|dietas|dietas.circuito.preleer|vec_dietas.circuito.preleer.v1|dietas|documento_dietas|preleer_competencia_circuito_dietas'
 'registrar_y_consumir_dietas_circuito_v3_atestada|dietas|dietas.documento.revisar|vec_dietas.documento.revisar.v1|dietas|documento_dietas|revisar_documento_dietas'
 'registrar_y_consumir_dietas_bandeja_v3_atestada|dietas|dietas.bandeja.revision.consultar|vec_dietas.bandeja.revision.consultar.v1|dietas|bandeja_dietas|consultar_bandeja_revision_dietas'
 'registrar_y_consumir_consulta_asignacion_dietas_v3_atestada|personal|personal.asignacion_dietas.consultar|vec_personal.asignacion_dietas.consultar.v1|personal|asignacion_dietas|preparar_borrador_dietas'
 'registrar_y_consumir_correccion_asignacion_dietas_v3_atestada|personal|personal.asignacion_dietas.corregir|vec_personal.asignacion_dietas.corregir.v1|personal|asignacion_dietas|corregir_asignacion_dietas'
 'registrar_y_consumir_correccion_grupo_dieta_v3_atestada|personal|personal.asignacion_dietas.grupo_corregir|vec_personal.asignacion_dietas.grupo_corregir.v1|personal|asignacion_dietas|corregir_grupo_dieta'
 'registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada|personal|personal.asignacion_dietas.registrar_inicial|vec_personal.asignacion_dietas.registrar_inicial.v1|personal|asignacion_dietas|registrar_asignacion_dietas_inicial'
)
perfiles=(documento_dietas_mutacion documento_dietas_consulta consulta_asignacion_dietas_personal
 correccion_asignacion_dietas_personal alta_inicial_asignacion_dietas_personal prelectura_dietas
 circuito_dietas bandeja_dietas)

ensayar() {
  orden=$1
  C="vec-ad3-59-${orden}-$$"
  contenedores+=("$C")
  docker run -d --rm --network none --name "$C" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
  esperar
  [[ $(val 'SHOW server_version_num') == 180004 ]] || fallo 'no es PostgreSQL 18.4'
  sql -o /dev/null < "$tmp/roles.sql" || fallo 'roles'
  docker exec -i "$C" pg_restore -U postgres -d postgres --exit-on-error --single-transaction < "$base" || fallo 'restauración'
  [[ $(val "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
     AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
     AND to_regprocedure('vec_autorizacion_atestada_v3.consumir_cronos_saldo_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
     AND strpos(pg_get_functiondef('$nucleo'::regprocedure),'importacion_organizacion_historica_personal')>0
     AND strpos(pg_get_functiondef('$nucleo'::regprocedure),'documento_dietas_mutacion')=0") == t ]] \
    || fallo 'la base no es la preimagen AD3-52 sin AD3-53/59'
  ok 'preimagen real hasta AD3-52 restaurada'

  sql -o /dev/null < "$dir/dietas_documento_ad3_000059_sondas.sql" || fallo 'sondas'
  sql -o /dev/null <<'SQL' || fallo 'login CT'
CREATE ROLE vec_prueba59_ct LOGIN INHERIT NOBYPASSRLS;
GRANT vec_contratacion_temporal_ejecutor TO vec_prueba59_ct WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba59 TO vec_prueba59_ct;
GRANT EXECUTE ON FUNCTION prueba59.consumir_ct(text,text) TO vec_prueba59_ct;
SQL
  local ct_antes ct_despues filas
  ct_antes=$(llamar vec_prueba59_ct ct registrar_y_consumir_decision_v3_atestada ct_alta)
  [[ $ct_antes == decision:*'|f' ]] || fallo "consumo CT previo: $ct_antes"
  filas=$(consumos)
  ok "consumo CT antes de AD3-59: ${ct_antes%%|*}, consumo_nuevo=f"

  [[ $orden == 53-59 ]] && aplicar53
  local antes
  antes=$(firma)
  sed '$s/^COMMIT;$/ROLLBACK;/' "$m59" | sql -o /dev/null || fallo 'AD3-59 en ROLLBACK'
  [[ $(firma) == "$antes" ]] || fallo 'ROLLBACK de AD3-59 dejó rastro'
  ok 'ROLLBACK de AD3-59 sin rastro en núcleo, audiencias, funciones ni ACL'
  sql -o /dev/null < "$m59" || fallo 'AD3-59 COMMIT'
  ok 'AD3-59 confirmada'
  [[ $orden == 59-53 ]] && aplicar53
  antes=$(firma)
  if sql -o /dev/null < "$m59" 2>/dev/null; then fallo 'segunda aplicación de AD3-59 aceptada'; fi
  [[ $(firma) == "$antes" ]] || fallo 'la segunda aplicación alteró el estado'
  ok 'segunda aplicación de AD3-59 rechazada sin cambios'

  local def p
  def=$(val "SELECT pg_get_functiondef('$nucleo'::regprocedure)")
  for p in "${perfiles[@]}" cronos_marcaje_propio cronos_saldo_propio importacion_organizacion_historica_personal; do
    [[ $(grep -c "IS DISTINCT FROM '$p'" <<<"$def") == 1 ]] || fallo "exclusión de $p no única"
  done
  grep -q "g.rolname='vec_dietas_ejecutor'" <<<"$def" || fallo 'guarda Dietas sin cotejo por nombre'
  grep -q "'vec_cronos_v1_ejecutor'" <<<"$def" || fallo 'guarda Cronos perdida'
  ok 'núcleo con exclusiones únicas de Dietas, D7, Cronos y AD3-52'

  ct_despues=$(llamar vec_prueba59_ct ct registrar_y_consumir_decision_v3_atestada ct_alta)
  [[ $ct_despues == "$ct_antes" ]] || fallo "CT tras AD3-59 distinto: $ct_despues"
  [[ $(consumos) == "$filas" ]] || fallo 'el consumo CT repetido creó filas'
  ok 'consumo CT tras AD3-59/53 idéntico y sin filas nuevas'

  sql -o /dev/null <<'SQL' || fallo 'logins Dietas y D7'
CREATE ROLE vec_personal_d7_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_prueba59_dietas LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba59_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba59_d7 LOGIN INHERIT NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba59_d7 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba59_doble LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba59_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_personal_d7_ejecutor TO vec_prueba59_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba59 TO vec_prueba59_dietas, vec_prueba59_d7, vec_prueba59_doble;
GRANT EXECUTE ON FUNCTION prueba59.consumir_dietas(text,text) TO vec_prueba59_dietas, vec_prueba59_d7, vec_prueba59_doble;
GRANT EXECUTE ON FUNCTION prueba59.consumir_personal(text,text) TO vec_prueba59_dietas, vec_prueba59_d7, vec_prueba59_doble;
SQL
  local linea fachada llamador op aud modulo tipo fin exacto ajeno r i=0
  for linea in "${fachadas[@]}"; do
    IFS='|' read -r fachada llamador op aud modulo tipo fin <<<"$linea"
    i=$((i+1))
    sql -o /dev/null -c "SELECT prueba59.preparar('f$i','$fachada','$op','$aud','$modulo','$tipo','$fin')" || fallo "material $fachada"
    [[ $(val "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
       WHERE p.oid='vec_autorizacion_atestada_v3.$fachada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
         AND a.grantee<>p.proowner") == 1 ]] || fallo "ACL de $fachada"
    if [[ $llamador == dietas ]]; then exacto=vec_prueba59_dietas ajeno=vec_prueba59_d7; else exacto=vec_prueba59_d7 ajeno=vec_prueba59_dietas; fi
    r=$(llamar "$exacto" "$llamador" "$fachada" "f$i")
    [[ $r == 'capacidad VEC-AD-3 rechazada' ]] || fallo "$fachada con $exacto: $r"
    r=$(llamar vec_prueba59_doble "$llamador" "$fachada" "f$i")
    [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "$fachada con dos grupos: $r"
    r=$(llamar "$ajeno" "$llamador" "$fachada" "f$i")
    [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "$fachada con login ajeno: $r"
    ok "$fachada: login exacto supera guarda y ligadura; dos grupos y login ajeno rechazados"
  done
  sql -o /dev/null -c "UPDATE prueba59.material SET motivo=motivo||'\x00'::bytea WHERE caso='f1'"
  r=$(llamar vec_prueba59_dietas dietas registrar_y_consumir_dietas_documento_v3_atestada f1)
  [[ $r == 'ligadura VEC-AD-3 inválida' ]] || fallo "ligadura alterada aceptada: $r"
  [[ $(consumos) == "$filas" ]] || fallo 'las sondas crearon consumos'
  ok 'material alterado se detiene en la ligadura; ninguna sonda consumió'
}

ensayar 59-53
ensayar 53-59
echo 'PG18.4: AD3-59 sobre preimagen real hasta AD3-52, en ambos órdenes con AD3-53: ROLLBACK, COMMIT, repetición, CT, Dietas y D7 verificados.'
