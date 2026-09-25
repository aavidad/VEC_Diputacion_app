#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de AD3-80 y Dietas 000008 sobre una base
# VEC restaurada (núcleo AD3 real), sin red. Borra el contenedor al salir.
#
# Uso: circuito_000008_pg18.sh BASE.dump ROLES.sql [53-80|80-53]
#   BASE.dump  pg_dump -Fc de una base VEC sintética con AD3-48 instalada y sin
#              Dietas 000001 (puede traer ya parte de la cadena: cada paso se
#              salta solo si existe el objeto que crea); se instala encima
#              la cadena hasta Dietas 000007.
#   ROLES.sql  pg_dumpall --roles-only de la misma instancia; las contraseñas
#              se descartan antes de entrar al contenedor.
#   ORDEN      orden de AD3-53 (Cronos) respecto de AD3-80; por omisión 53-80.
#
# Comprueba: ROLLBACK sin rastro, COMMIT y repetición rechazada de AD3-80 y de
# 000008; la fachada AD3-80 real con su LOGIN exacto supera guarda y ligadura y
# se detiene en la clave de capacidad; ACL y SECURITY DEFINER de 000008; y,
# con dobles de consumo SOLO en este contenedor, lectura del revisor,
# separación de personas, decisión, idempotencia, conflicto de clave, mismo
# recibo tras reiniciar PostgreSQL y DOWN rechazado con historia.
set -Eeuo pipefail
base=${1:?falta BASE.dump}
roles=${2:?falta ROLES.sql}
orden=${3:-53-80}
[[ $orden == 53-80 || $orden == 80-53 ]] || { echo 'ORDEN: 53-80 o 80-53' >&2; exit 2; }
[[ -s $base && -s $roles ]] || { echo 'Base o roles vacíos' >&2; exit 2; }
dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(CDPATH='' cd -- "$dir/../../../.." && pwd)
pg=$repo/deploy/postgresql
tmp=$(mktemp -d)
C="vec-dietas-000008-$$"
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
firma() {
  val "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef(c.oid))||
   (SELECT md5(string_agg(p.oid::regprocedure::text||coalesce(p.proacl::text,'')||md5(p.prosrc),'|' ORDER BY p.oid::regprocedure::text))
      FROM pg_proc p WHERE p.pronamespace IN ('vec_autorizacion_atestada_v3'::regnamespace,'vec_dietas'::regnamespace))
   FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
}
# Invoca prueba8.invocar con el LOGIN del ejecutor; devuelve JSON o el error.
invocar() {
  { docker exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba8_dietas -d postgres 2>&1 || true; } <<SQL | sed -n 's/^.*ERROR: *//p;/^{/p' | head -1
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT prueba8.invocar($1);
COMMIT;
SQL
}
campo() { python3 -c 'import json,sys; d=json.loads(sys.argv[1]); [d:=d[k] for k in sys.argv[2].split(".")]; print(d)' "$1" "$2"; }

docker run -d --rm --network none --name "$C" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
esperar
[[ $(val 'SHOW server_version_num') == 180004 ]] || fallo 'no es PostgreSQL 18.4'
sql -o /dev/null < "$tmp/roles.sql" || fallo 'roles'
docker exec -i "$C" pg_restore -U postgres -d postgres < "$base" >/dev/null 2>&1 || true
[[ $(val "SELECT to_regprocedure('$nucleo') IS NOT NULL AND to_regclass('vec_dietas.borrador_comision') IS NULL") == t ]] \
  || fallo 'la base no tiene el núcleo AD3 o ya tiene Dietas 000001'
ok 'base VEC restaurada sin Dietas 000001'

# Cada paso se identifica por un objeto concreto que crea. Si la base ya trae
# ese objeto (volcado con parte de la cadena), el paso se salta; si no, se
# aplica y cualquier error aborta el ensayo: nunca se decide por el texto.
#   r:nombre       rol          e:esquema     t:esquema.tabla
#   f:esquema.fn   función      p:esquema.tabla.política
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
  if [[ $(existe "$marca") == t ]]; then echo "·  $f ya presente ($marca)"; continue; fi
  sql -o /dev/null < "$pg/$f" 2>"$tmp/err" || { cat "$tmp/err" >&2; fallo "cadena: $f"; }
  [[ $(existe "$marca") == t ]] || fallo "cadena: $f no creó $marca"
done
ok 'cadena instalada hasta Dietas 000007 y AD3-59'

m53=$pg/autorizacion_atestada_v3/migraciones/000053_consumidores_cronos_empleado.up.sql
m80=$pg/autorizacion_atestada_v3/migraciones/000080_consumidor_revisor_documento_dietas.up.sql
m8=$pg/dietas_borradores/migraciones/000008_revision_circuito_comision.up.sql
# AD3-53 (Cronos) reescribe el mismo núcleo: se ensaya antes y después de AD3-80.
# Su preimagen son los dos roles NOLOGIN de cronos_v1 000001, con sus atributos.
sql -o /dev/null <<'SQL' || fallo 'roles de cronos_v1 000001'
DO $$ BEGIN
 IF to_regrole('vec_cronos_v1_propietario') IS NULL THEN CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; END IF;
 IF to_regrole('vec_cronos_v1_ejecutor') IS NULL THEN CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; END IF;
END $$;
SQL
case $orden in
  53-80) migs=("$m53" "$m80" "$m8") ;;
  80-53) migs=("$m80" "$m53" "$m8") ;;
esac
for m in "${migs[@]}"; do
  antes=$(firma)
  sed '$s/^COMMIT;$/ROLLBACK;/' "$m" | sql -o /dev/null || fallo "ROLLBACK de $(basename "$m")"
  [[ $(firma) == "$antes" ]] || fallo "ROLLBACK de $(basename "$m") dejó rastro"
  sql -o /dev/null < "$m" || fallo "COMMIT de $(basename "$m")"
  antes=$(firma)
  if sql -o /dev/null < "$m" 2>/dev/null; then fallo "segunda aplicación de $(basename "$m") aceptada"; fi
  [[ $(firma) == "$antes" ]] || fallo "la segunda aplicación de $(basename "$m") alteró el estado"
  ok "$(basename "$m"): ROLLBACK sin rastro, COMMIT y repetición rechazada"
done
def=$(val "SELECT pg_get_functiondef('$nucleo'::regprocedure)")
excl=$(grep -oE "p_perfil_mutacion IS DISTINCT FROM '[a-z_]+'" <<<"$def" | sed -E "s/.*'(.*)'/\1/")
[[ -z $(sort <<<"$excl" | uniq -d) ]] || fallo "exclusiones repetidas en el núcleo: $(sort <<<"$excl" | uniq -d | tr '\n' ' ')"
for e in circuito_dietas revisor_documento_dietas importacion_organizacion_historica_personal \
         cronos_marcaje_propio cronos_marcaje_remoto_disponibilidad cronos_marcaje_remoto_recibo cronos_saldo_propio; do
  [[ $(grep -cx "$e" <<<"$excl") == 1 ]] || fallo "exclusión $e ausente o repetida"
done
ok "núcleo con AD3-59, AD3-53 y AD3-80 (orden $orden): $(wc -l <<<"$excl") exclusiones, cada una una sola vez"

# Fachada AD3-80 real: login exacto hasta la clave; dos grupos y ajeno fuera.
sql -o /dev/null < "$repo/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/dietas_documento_ad3_000059_sondas.sql" || fallo 'sondas'
sql -o /dev/null <<'SQL' || fallo 'logins de sonda'
CREATE ROLE vec_prueba80_dietas LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba80_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba80_doble LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba80_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_personal_d7_ejecutor TO vec_prueba80_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba80_d7 LOGIN INHERIT NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba80_d7 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba59 TO vec_prueba80_dietas, vec_prueba80_doble, vec_prueba80_d7;
GRANT EXECUTE ON FUNCTION prueba59.consumir_dietas(text,text) TO vec_prueba80_dietas, vec_prueba80_doble, vec_prueba80_d7;
SELECT prueba59.preparar('r80','registrar_y_consumir_dietas_revisor_documento_v3_atestada',
 'dietas.circuito.documento.consultar','vec_dietas.circuito.documento.consultar.v1','dietas',
 'documento_dietas','revisar_documento_circuito_dietas');
SQL
sonda() {
  { docker exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U "$1" -d postgres 2>&1 || true; } <<'SQL' | sed -n 's/^.*ERROR: *//p;/|/p' | head -1
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM prueba59.consumir_dietas('registrar_y_consumir_dietas_revisor_documento_v3_atestada','r80');
COMMIT;
SQL
}
[[ $(sonda vec_prueba80_dietas) == 'capacidad VEC-AD-3 rechazada' ]] || fallo "AD3-80 con login exacto: $(sonda vec_prueba80_dietas)"
[[ $(sonda vec_prueba80_doble) == 'consumo VEC-AD-3 rechazado' ]] || fallo 'AD3-80 con dos grupos'
[[ $(sonda vec_prueba80_d7) == 'consumo VEC-AD-3 rechazado' ]] || fallo 'AD3-80 con login ajeno'
[[ $(val "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a
   WHERE p.oid='vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
     AND a.grantee<>p.proowner AND a.grantee<>'vec_dietas_propietario'::regrole") == 0 ]] || fallo 'ACL de la fachada AD3-80'
ok 'fachada AD3-80 real: login exacto hasta la clave; dos grupos y ajeno rechazados; ACL cerrada'

firmas="'vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea)','vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'"
[[ $(val "SELECT bool_and(p.prosecdef AND p.proowner='vec_dietas_propietario'::regrole
     AND 'search_path=pg_catalog'=ANY(p.proconfig)
     AND NOT has_function_privilege('public',p.oid,'EXECUTE')
     AND NOT has_function_privilege('vec_dietas_registrador_frontera',p.oid,'EXECUTE'))
   FROM pg_proc p WHERE p.oid::regprocedure::text IN ($firmas)") == t ]] || fallo 'propiedad, entorno o PUBLIC de 000008'
[[ $(val "SELECT has_function_privilege('vec_dietas_ejecutor','vec_dietas.decidir_comision_v2(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
   AND has_function_privilege('vec_dietas_ejecutor','vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
   AND NOT has_function_privilege('vec_dietas_ejecutor','vec_dietas.cotejar_efecto_circuito_v2(text,bytea,bytea,bytea)','EXECUTE')
   AND NOT has_function_privilege('vec_dietas_ejecutor','vec_dietas.decidir_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
   AND NOT has_function_privilege('vec_dietas_ejecutor','vec_dietas.preleer_circuito_comision_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
   AND NOT has_table_privilege('vec_dietas_ejecutor','vec_dietas.cola_circuito_comision','SELECT')
   AND (SELECT relforcerowsecurity FROM pg_class WHERE oid='vec_dietas.cola_circuito_comision'::regclass)") == t ]] \
  || fallo 'ACL del ejecutor en 000008'
ok '000008: SECURITY DEFINER con search_path fijo, sin PUBLIC; ejecutor solo con v2 y lectura del revisor'

# A partir de aquí, dobles de consumo solo en este contenedor.
sql -o /dev/null < "$dir/circuito_000008_preparar.sql" || fallo 'preparación funcional'
r=$(invocar "'consultar_documento_circuito_v1','T','consultar_documento','revision',NULL,NULL,NULL,NULL,'n01'")
[[ $(campo "$r" resultado) == no_encontrado ]] || fallo "la titular lee como revisora: $r"
r=$(invocar "'consultar_documento_circuito_v1','R','consultar_documento','revision',NULL,NULL,NULL,NULL,'n02'")
[[ $(campo "$r" resultado) == no_encontrado ]] || fallo "el responsable lee la revisión administrativa: $r"
r=$(invocar "'consultar_documento_circuito_v1','A','consultar_documento','revision',NULL,NULL,NULL,NULL,'n03'")
[[ $(campo "$r" resultado) == concedido && $(campo "$r" comision.documento.lineas) == *just:ensayo:01* ]] || fallo "lectura del administrativo: $r"
[[ $r != *relacion_ref* && $r != *administrativo_persona_ref* ]] || fallo 'la lectura del revisor expone datos de asignación'
r=$(invocar "'consultar_documento_circuito_v1','A','consultar_documento','revision',NULL,NULL,NULL,NULL,'n03'")
[[ $r == *'consumo AD3 de revisión incompatible'* ]] || fallo "consumo repetido aceptado: $r"
ok 'lectura del revisor: solo el administrativo competente, con justificantes y sin asignación; nonce de un solo uso'

r=$(invocar "'decidir_comision_v2','T','decidir','revision','aprobar','','claveDecisionTitular01',2,'n10'")
[[ $r == *'separación de funciones Dietas'* ]] || fallo "la titular revisa lo suyo: $r"
r=$(invocar "'decidir_comision_v2','A','decidir','revision','aprobar','','claveDecisionRevision1',2,'n11'")
[[ $(campo "$r" comision.estado) == pendiente_autorizacion && $(campo "$r" recibo.repeticion) == False ]] || fallo "elevación: $r"
recibo=$(campo "$r" recibo.referencia); en=$(campo "$r" recibo.registrado_en)
filas() { val "SELECT (SELECT count(*) FROM vec_dietas.comision_revision)||'|'||(SELECT count(*) FROM vec_dietas.recibo_operacion_comision)||'|'||(SELECT count(*) FROM vec_dietas.historia_operacion_comision)||'|'||(SELECT count(*) FROM vec_dietas.outbox_comision)||'|'||(SELECT count(*) FROM vec_dietas.cola_circuito_comision)"; }
tras=$(filas)
[[ $tras == '2|2|1|1|2' ]] || fallo "efectos de la elevación: $tras"
r=$(invocar "'decidir_comision_v2','A','decidir','revision','aprobar','','claveDecisionRevision1',2,'n12'")
[[ $(campo "$r" recibo.referencia) == "$recibo" && $(campo "$r" recibo.registrado_en) == "$en" && $(campo "$r" recibo.repeticion) == True ]] || fallo "repetición: $r"
[[ $(filas) == "$tras" ]] || fallo 'la repetición creó filas'
r=$(invocar "'decidir_comision_v2','A','decidir','revision','aprobar','otro motivo','claveDecisionRevision1',2,'n13'")
[[ $r == *'conflicto de idempotencia Dietas'* ]] || fallo "clave reutilizada con otro contenido: $r"
r=$(invocar "'decidir_comision_v2','A','decidir','autorizacion','aprobar','','claveAutorizaAdmin0001',3,'n14'")
[[ $r == *'separación de funciones Dietas'* ]] || fallo "el administrativo autoriza tras revisar: $r"
ok 'decisión: estado, recibo, historia, outbox y cola en una transacción; repetición idéntica; conflicto y separación rechazados'

docker restart "$C" >/dev/null
esperar
r=$(invocar "'decidir_comision_v2','A','decidir','revision','aprobar','','claveDecisionRevision1',2,'n15'")
[[ $(campo "$r" recibo.referencia) == "$recibo" && $(campo "$r" recibo.registrado_en) == "$en" && $(campo "$r" recibo.repeticion) == True ]] || fallo "recibo tras reinicio: $r"
[[ $(filas) == "$tras" ]] || fallo 'el reinicio o la repetición crearon filas'
ok 'tras reiniciar PostgreSQL la misma decisión devuelve el mismo recibo y fecha, sin filas nuevas'

r=$(invocar "'decidir_comision_v2','R','decidir','autorizacion','devolver','no','claveDevolucionResp001',3,'n16'")
[[ $r == *'decisión Dietas inválida'* ]] || fallo "devolución sin motivo suficiente: $r"
r=$(invocar "'decidir_comision_v2','R','decidir','autorizacion','devolver','Falta el justificante del aparcamiento','claveDevolucionResp001',3,'n17'")
[[ $(campo "$r" comision.estado) == devuelta ]] || fallo "devolución del responsable: $r"
[[ $(val "SELECT destinatario_persona_ref FROM vec_dietas.outbox_comision WHERE tipo='circuito_devuelta'") == per_titularSintetico000000001 ]] || fallo 'aviso de devolución'
[[ $(val "SELECT motivo FROM vec_dietas.historia_operacion_comision WHERE estado_nuevo='devuelta'") == 'Falta el justificante del aparcamiento' ]] || fallo 'motivo de devolución en la historia'
[[ $(val "SELECT count(*) FROM vec_dietas.cola_circuito_comision WHERE version=4") == 0 ]] || fallo 'una devolución abrió cola'
r=$(invocar "'decidir_comision_v2','R','decidir','autorizacion','aprobar','','claveAutorizaResp00001',3,'n18'")
[[ $r == *'versión Dietas desactualizada'* || $r == *'asignación Dietas incompatible'* ]] || fallo "decisión sobre versión superada: $r"
ok 'devolución con motivo al titular, sin cola nueva; motivo corto y versión superada rechazados'

if sql -o /dev/null < "$pg/dietas_borradores/migraciones/000008_revision_circuito_comision.down.sql" 2>/dev/null; then fallo 'DOWN aceptó historia'; fi
[[ $(filas) == '3|3|2|2|2' ]] || fallo 'el DOWN rechazado alteró la historia'
ok 'DOWN de 000008 rechazado con decisiones registradas'
echo "PG18.4 ($orden): AD3-53, AD3-80 y Dietas 000008 verificados sobre núcleo AD3 real y cadena Dietas hasta 000007."
