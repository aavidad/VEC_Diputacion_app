#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de AD3-60/62 y Documentos 000001-000003
# sobre una base VEC restaurada con el núcleo AD3 real, junto a las AD3 de main
# que reescriben el mismo núcleo (53, 54-56, 59, 61, 70, 80). Sin red; borra el
# contenedor al salir.
#
# Uso: probar_cadena_real_pg18.sh BASE.dump ROLES.sql [main-60|60-main]
#   BASE.dump  pg_dump -Fc de una base VEC sintética con AD3-48 instalada y sin
#              Dietas 000001 (la misma preimagen del ensayo Dietas 000008).
#   ROLES.sql  pg_dumpall --roles-only de la misma instancia; se descartan las
#              contraseñas antes de entrar al contenedor.
#   ORDEN      main-60 (por omisión): las AD3 de main ya instaladas y después
#              AD3-60/62, que es el orden real de despliegue; 60-main: orden
#              numérico, con 60/62 antes de 61/70/80.
#
# Comprueba: ROLLBACK sin rastro, COMMIT y repetición rechazada de cada
# migración de Documentos; núcleo con cada exclusión una sola vez; fachada
# AD3-60 real con LOGIN exacto hasta la clave de capacidad y rechazo con dos
# grupos o login ajeno; ACL y RLS del esquema documental; y persistencia de
# funciones, ACL y núcleo tras reiniciar PostgreSQL. No acredita COSE real: la
# base no tiene clave publicada para vec_documentos.operacion.v1.
set -Eeuo pipefail
base=${1:?falta BASE.dump}
roles=${2:?falta ROLES.sql}
orden=${3:-main-60}
[[ $orden == main-60 || $orden == 60-main ]] || { echo 'ORDEN: main-60 o 60-main' >&2; exit 2; }
[[ -s $base && -s $roles ]] || { echo 'Base o roles vacíos' >&2; exit 2; }
dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(CDPATH='' cd -- "$dir/../../.." && pwd)
pg=$repo/deploy/postgresql
tmp=$(mktemp -d)
C="vec-b5-cadena-$$"
limpiar() { docker rm -f "$C" >/dev/null 2>&1 || true; rm -rf -- "$tmp"; }
trap limpiar EXIT INT TERM
sed -E "s/ PASSWORD '[^']*'//; s/ PASSWORD [^ ;]+//" "$roles" | grep -vE '^(CREATE|ALTER) ROLE postgres( |;)|^\\(un)?restrict' > "$tmp/roles.sql"
if grep -qi 'password' "$tmp/roles.sql"; then echo 'No se pudieron retirar las contraseñas' >&2; exit 2; fi

fallo() { echo "FALLO: $*" >&2; exit 1; }
ok() { echo "OK $*"; }
sql() { docker exec -i "$C" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
val() { docker exec "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
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
nucleo="vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
fachada="vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
firma() {
  val "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef(c.oid))||
   (SELECT md5(coalesce(string_agg(p.oid::regprocedure::text||coalesce(p.proacl::text,'')||md5(p.prosrc),'|' ORDER BY p.oid::regprocedure::text),''))
      FROM pg_proc p WHERE p.pronamespace IN ('vec_autorizacion_atestada_v3'::regnamespace)
         OR p.pronamespace=to_regnamespace('vec_documentos'))||
   (SELECT count(*) FROM pg_class WHERE relnamespace=to_regnamespace('vec_documentos'))
   FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
}
existe() {
  local k=${1%%:*} o=${1#*:}
  case $k in
    r) val "SELECT to_regrole('$o') IS NOT NULL" ;;
    e) val "SELECT to_regnamespace('$o') IS NOT NULL" ;;
    t) val "SELECT to_regclass('$o') IS NOT NULL" ;;
    f) val "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace=to_regnamespace('${o%%.*}') AND proname='${o#*.}')" ;;
    p) val "SELECT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=to_regclass('${o%.*}') AND polname='${o##*.}')" ;;
    *) fallo "marcador desconocido: $1" ;;
  esac
}
# instalar FICHERO MARCA: ROLLBACK sin rastro, COMMIT, repetición rechazada.
instalar() {
  local f=$pg/$1 marca=$2 antes
  [[ $(existe "$marca") == f ]] || fallo "$1: $marca ya existía"
  antes=$(firma)
  sed '$s/^COMMIT;$/ROLLBACK;/' "$f" | sql -o /dev/null 2>"$tmp/err" || { cat "$tmp/err" >&2; fallo "ROLLBACK de $1"; }
  [[ $(firma) == "$antes" && $(existe "$marca") == f ]] || fallo "ROLLBACK de $1 dejó rastro"
  sql -o /dev/null < "$f" 2>"$tmp/err" || { cat "$tmp/err" >&2; fallo "COMMIT de $1"; }
  [[ $(existe "$marca") == t ]] || fallo "$1 no creó $marca"
  antes=$(firma)
  if sql -o /dev/null < "$f" 2>/dev/null; then fallo "segunda aplicación de $1 aceptada"; fi
  [[ $(firma) == "$antes" ]] || fallo "la segunda aplicación de $1 alteró el estado"
  ok "$(basename "$1"): ROLLBACK sin rastro, COMMIT y repetición rechazada"
}

docker run -d --rm --network none --name "$C" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
esperar
[[ $(val 'SHOW server_version_num') == 180004 ]] || fallo 'no es PostgreSQL 18.4'
sql -o /dev/null < "$tmp/roles.sql" || fallo 'roles'
docker exec -i "$C" pg_restore -U postgres -d postgres < "$base" >/dev/null 2>&1 || true
[[ $(val "SELECT to_regprocedure('$nucleo') IS NOT NULL AND to_regclass('vec_dietas.borrador_comision') IS NULL AND to_regnamespace('vec_documentos') IS NULL") == t ]] \
  || fallo 'la base no tiene el núcleo AD3, ya tiene Dietas 000001 o ya tiene Documentos'
ok 'base VEC restaurada sin Dietas 000001 ni Documentos'

# Cadena previa idéntica a la del ensayo Dietas 000008 (hasta AD3-59).
cadena=(
 'dietas_borradores/roles_up.sql e:vec_dietas'
 'personal/migraciones/000007_relacion_empleado_dietas.up.sql t:vec_personal.relacion_empleado_dietas'
 'autorizacion_atestada_v3/migraciones/000049_consumidor_personal_dietas.up.sql f:vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada'
 'autorizacion_atestada_v3/migraciones/000050_acceso_rutas_dietas.up.sql f:vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada'
 'personal/migraciones/000008_consulta_relaciones_propias_dietas.up.sql t:vec_personal.recibo_consulta_relacion_propia_dietas'
 'personal/migraciones/000009_asignacion_dietas.up.sql t:vec_personal.asignacion_dietas'
 'autorizacion_atestada_v3/migraciones/000051_consumidor_organizacion_historica.up.sql f:vec_autorizacion_atestada_v3.consumir_consulta_organizacion_v3_atestada'
 'autorizacion_atestada_v3/migraciones/000052_consumidor_importacion_organizacion.up.sql f:vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada'
 'dietas_borradores/migraciones/000001_borrador_comision_durable.up.sql t:vec_dietas.borrador_comision'
 'dietas_borradores/migraciones/000002_tarifas_provisionales.up.sql t:vec_dietas.version_tarifa_provisional'
 'dietas_borradores/migraciones/000003_consulta_tarifas_provisionales.up.sql p:vec_dietas.version_tarifa_provisional.lectura_ejecutor_tarifas'
 'dietas_borradores/migraciones/000004_calculo_comision.up.sql t:vec_dietas.calculo_comision'
 'dietas_borradores/migraciones/000005_auditoria_frontera.up.sql t:vec_dietas.auditoria_frontera_comision'
 'autorizacion_atestada_v3/migraciones/000059_consumidor_documento_dietas.up.sql f:vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada'
 'personal/migraciones/000012_asignacion_dietas.up.sql t:vec_personal.recibo_asignacion_dietas'
 'personal/migraciones/000013_auditoria_frontera_asignacion_dietas.up.sql t:vec_personal.auditoria_frontera_asignacion_dietas'
 'dietas_borradores/migraciones/000006_documento_comision.up.sql t:vec_dietas.comision_revision'
 'dietas_borradores/migraciones/000007_circuito_comision.up.sql t:vec_dietas.cola_circuito_comision'
)
for paso in "${cadena[@]}"; do
  f=${paso% *} marca=${paso##* }
  if [[ $(existe "$marca") == t ]]; then continue; fi
  sql -o /dev/null < "$pg/$f" 2>"$tmp/err" || { cat "$tmp/err" >&2; fallo "cadena: $f"; }
  [[ $(existe "$marca") == t ]] || fallo "cadena: $f no creó $marca"
done
ok 'cadena previa instalada hasta AD3-59 y Dietas 000007'
sql -o /dev/null <<'SQL' || fallo 'roles de cronos_v1 000001'
DO $$ BEGIN
 IF to_regrole('vec_cronos_v1_propietario') IS NULL THEN CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; END IF;
 IF to_regrole('vec_cronos_v1_ejecutor') IS NULL THEN CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; END IF;
END $$;
SQL

ad3=autorizacion_atestada_v3/migraciones
main_antes=(
 "$ad3/000053_consumidores_cronos_empleado.up.sql f:vec_autorizacion_atestada_v3.consumir_cronos_saldo_propio_v3_atestada"
 "$ad3/000054_consumidor_registro_empleado_b2.up.sql f:vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada"
)
main_despues=(
 "$ad3/000070_consumidores_cronos_movimientos_permisos.up.sql f:vec_autorizacion_atestada_v3.consumir_cronos_movimientos_propio_v3_atestada"
 "$ad3/000080_consumidor_revisor_documento_dietas.up.sql f:vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada"
)
documentos() {
  instalar documentos/roles_up.sql e:vec_documentos
  instalar "$ad3/000060_documentos_comunes.up.sql" f:vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada
  instalar documentos/migraciones/000001_documentos_comunes.up.sql t:vec_documentos.documento
  [[ $(val "SELECT to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NOT NULL") == t ]] \
    || fallo 'la base no tiene la revalidación viva que exige AD3-62'
  instalar "$ad3/000062_replay_documentos_comunes.up.sql" f:vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada
  instalar documentos/migraciones/000002_replay_autorizado.up.sql f:vec_documentos.confirmar_alta_v2
  instalar documentos/migraciones/000003_custodia_externa.up.sql t:vec_documentos.referencia_externa
}
case $orden in
  main-60) for p in "${main_antes[@]}" "${main_despues[@]}"; do instalar "${p% *}" "${p##* }"; done; documentos ;;
  60-main) for p in "${main_antes[@]}"; do instalar "${p% *}" "${p##* }"; done; documentos
           for p in "${main_despues[@]}"; do instalar "${p% *}" "${p##* }"; done ;;
esac

comprobar_nucleo() {
  local def excl
  def=$(val "SELECT pg_get_functiondef('$nucleo'::regprocedure)")
  excl=$(grep -oE "p_perfil_mutacion IS DISTINCT FROM '[a-z0-9_]+'" <<<"$def" | sed -E "s/.*'(.*)'/\1/")
  [[ -z $(sort <<<"$excl" | uniq -d) ]] || fallo "exclusiones repetidas en el núcleo: $(sort <<<"$excl" | uniq -d | tr '\n' ' ')"
  for e in operacion_documentos_comunes circuito_dietas revisor_documento_dietas registro_empleado_b2 \
           importacion_organizacion_historica_personal cronos_saldo_propio cronos_movimientos_propio; do
    [[ $(grep -cx "$e" <<<"$excl") == 1 ]] || fallo "exclusión $e ausente o repetida"
  done
  [[ $(val "SELECT pg_get_constraintdef(oid) LIKE '%vec_documentos.operacion.v1%' FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'") == t ]] \
    || fallo 'audiencia documental ausente'
  wc -l <<<"$excl"
}
n=$(comprobar_nucleo)
ok "núcleo con AD3-53/54/59/60/70/80 (orden $orden): $n exclusiones, cada una una sola vez"

# Fachada AD3-60 real con material derivado de un alta CT consumida de la base.
sql -o /dev/null < "$pg/autorizacion_atestada_v3/pruebas_sql/dietas_documento_ad3_000059_sondas.sql" || fallo 'sondas'
sql -o /dev/null <<'SQL' || fallo 'sonda documental'
CREATE FUNCTION prueba59.preparar_documentos(p_caso text,p_operacion text,p_tipo text,p_finalidad text,p_campos jsonb) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE b prueba59.material; c jsonb; d jsonb; decision bytea; efecto text; huella text;
BEGIN
 SELECT * INTO STRICT b FROM prueba59.material WHERE caso='ct_alta';
 efecto:='doc:00000000-0000-4000-8000-'||lpad(substr(md5(p_caso),1,12),12,'0');
 huella:=encode(sha256(convert_to(efecto,'UTF8')),'hex');
 d:=convert_from(b.decision,'UTF8')::jsonb
   || jsonb_build_object('decision_ref','decision:prueba60:'||p_caso,'accion',p_operacion,
      'modulo_id','documentos','tipo_recurso',p_tipo,'finalidad',p_finalidad,'recurso_ref',efecto,
      'contexto_recurso_huella_sha256',huella,'campos_permitidos',p_campos,'obligaciones','[]'::jsonb);
 decision:=convert_to(d::text,'UTF8');
 c:=convert_from(b.capacidad,'UTF8')::jsonb
   || jsonb_build_object('decision_ref',d->>'decision_ref','operacion',p_operacion,
      'audiencia_consumo','vec_documentos.operacion.v1','efecto_ref',efecto,'huella_efecto_sha256',huella,
      'huella_decision_sha256',encode(sha256(decision),'hex'),
      'nonce',encode(sha256(convert_to('nonce:prueba60:'||p_caso,'UTF8')),'hex'));
 INSERT INTO prueba59.material VALUES (p_caso,
  vec_autorizacion_atestada_v3.capacidad_canonica(c),decision,b.motivo,b.contexto,
  b.persona_version,b.perfil_version,b.payload,b.sobre,b.evidencia,b.raiz);
END $f$;
CREATE FUNCTION prueba59.consumir_documentos(p_caso text)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $c$
DECLARE m prueba59.material;
BEGIN
 SELECT * INTO STRICT m FROM prueba59.material WHERE caso=p_caso;
 RETURN QUERY SELECT * FROM vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(
  m.capacidad,m.decision,m.motivo,m.contexto,m.persona_version,m.perfil_version,m.payload,m.sobre,m.evidencia,m.raiz);
END $c$;
REVOKE ALL ON FUNCTION prueba59.consumir_documentos(text) FROM PUBLIC;
ALTER FUNCTION prueba59.consumir_documentos(text) OWNER TO vec_documentos_propietario;
GRANT USAGE ON SCHEMA prueba59 TO vec_documentos_propietario;
GRANT SELECT ON prueba59.material TO vec_documentos_propietario;
SELECT prueba59.preparar_documentos('alta','documentos.generado.alta','documento_generado','alta_documento_generado','["documento","recibo"]');
SELECT prueba59.preparar_documentos('externa','documentos.externo.registrar','documento_externo','registrar_documento_externo','["documento","recibo"]');
SELECT prueba59.preparar_documentos('mal','documentos.generado.alta','documento_generado','alta_documento_generado','["documento"]');
CREATE ROLE vec_prueba60_documentos LOGIN INHERIT NOBYPASSRLS;
GRANT vec_documentos_ejecutor TO vec_prueba60_documentos WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba60_doble LOGIN INHERIT NOBYPASSRLS;
GRANT vec_documentos_ejecutor TO vec_prueba60_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_dietas_ejecutor TO vec_prueba60_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba60_dietas LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba60_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba59 TO vec_prueba60_documentos, vec_prueba60_doble, vec_prueba60_dietas;
GRANT EXECUTE ON FUNCTION prueba59.consumir_documentos(text) TO vec_prueba60_documentos, vec_prueba60_doble, vec_prueba60_dietas;
SQL
sonda() {
  { docker exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U "$1" -d postgres 2>&1 || true; } <<SQL | sed -n 's/^.*ERROR: *//p;/|/p' | head -1
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM prueba59.consumir_documentos('$2');
COMMIT;
SQL
}
fachada_real() {
  local r
  r=$(sonda vec_prueba60_documentos alta); [[ $r == 'capacidad VEC-AD-3 rechazada' ]] || fallo "AD3-60 alta con login exacto: $r"
  r=$(sonda vec_prueba60_documentos externa); [[ $r == 'capacidad VEC-AD-3 rechazada' ]] || fallo "AD3-60 externa con login exacto: $r"
  r=$(sonda vec_prueba60_documentos mal); [[ $r == 'AD3-60: operación documental denegada' ]] || fallo "AD3-60 con campos ajenos: $r"
  r=$(sonda vec_prueba60_doble alta); [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "AD3-60 con dos grupos: $r"
  r=$(sonda vec_prueba60_dietas alta); [[ $r == 'consumo VEC-AD-3 rechazado' ]] || fallo "AD3-60 con login ajeno: $r"
  [[ $(val "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
     WHERE p.oid='$fachada'::regprocedure AND a.grantee<>p.proowner AND a.grantee<>'vec_documentos_propietario'::regrole") == 0 ]] \
    || fallo 'ACL de la fachada AD3-60'
}
fachada_real
ok 'fachada AD3-60 real: login exacto hasta la clave (alta y externa); campos ajenos, dos grupos y login ajeno rechazados; ACL cerrada'

acl_documental() {
  [[ $(val "SELECT bool_and(c.relrowsecurity AND c.relforcerowsecurity
      AND NOT has_table_privilege('vec_documentos_ejecutor',c.oid,'SELECT')
      AND NOT has_table_privilege('vec_documentos_ejecutor',c.oid,'INSERT')
      AND NOT has_table_privilege('vec_documentos_ejecutor',c.oid,'UPDATE')
      AND NOT has_table_privilege('vec_documentos_ejecutor',c.oid,'DELETE'))
     FROM pg_class c WHERE c.relnamespace='vec_documentos'::regnamespace AND c.relkind='r'") == t ]] || fallo 'RLS o ACL de tablas documentales'
  [[ $(val "SELECT bool_and(p.prosecdef AND 'search_path=pg_catalog'=ANY(p.proconfig) AND NOT has_function_privilege('public',p.oid,'EXECUTE'))
     FROM pg_proc p WHERE p.pronamespace='vec_documentos'::regnamespace AND has_function_privilege('vec_documentos_ejecutor',p.oid,'EXECUTE')") == t ]] \
    || fallo 'función documental ejecutable sin SECURITY DEFINER, search_path fijo o cerrada a PUBLIC'
  [[ $(val "SELECT has_function_privilege('vec_documentos_ejecutor','vec_documentos.registrar_referencia_externa_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')") == t ]] \
    || fallo 'el ejecutor no puede registrar referencias externas'
}
acl_documental
ok 'Documentos: tablas con RLS forzada y sin DML directo del ejecutor; funciones SECURITY DEFINER cerradas a PUBLIC'

antes=$(firma)
docker restart "$C" >/dev/null
esperar
[[ $(firma) == "$antes" ]] || fallo 'el reinicio alteró núcleo, fachadas o esquema'
comprobar_nucleo >/dev/null
fachada_real
acl_documental
ok 'tras reiniciar PostgreSQL: mismo núcleo, fachada y ACL; la sonda vuelve a detenerse en la clave'
echo "PG18.4 ($orden): AD3-60/62 y Documentos 000001-000003 sobre núcleo AD3 real con AD3-53/54/59/70/80. NO acredita cadena COSE con clave documental publicada."
