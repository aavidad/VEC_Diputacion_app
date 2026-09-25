#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de AD3-75 y Dietas 000011 (D6: la
# devolución solo se devuelve cuando la decisión V3 la concede), sobre una
# base VEC restaurada con el núcleo AD3 real y sin red. Borra el contenedor
# al salir.
#
# Uso: campo_devolucion_000011_pg18.sh BASE.dump ROLES.sql
#   BASE.dump  pg_dump -Fc de una base VEC sintética con AD3-48 instalada, un
#              alta CT consumida y sin Dietas 000001; se instala encima la
#              cadena real hasta Dietas 000010, AD3-59, AD3-53 y AD3-80.
#   ROLES.sql  pg_dumpall --roles-only de la misma instancia; las contraseñas
#              se descartan antes de entrar al contenedor.
#
# Comprueba, con las fachadas AD3 REALES y un login exacto miembro solo del
# ejecutor Dietas: antes de AD3-75 ninguna lista con la devolución pasa y la
# lista de Dietas 000006 de la titular tampoco (divergencia de AD3-59);
# AD3-75 con ROLLBACK sin rastro, preimagen alterada rechazada, COMMIT y
# repetición rechazada; después, cada fachada admite solo su lista base o la
# base con la devolución, y rechaza la lista antigua, una parcial, otra
# desordenada y otra ampliada; login ajeno y doble siguen fuera y ninguna
# sonda consume. Dietas 000011 con ROLLBACK, preimagen alterada, COMMIT,
# repetición, DOWN exacto y reinstalación. Con dobles de consumo SOLO en este
# contenedor: detalle y listado reales de la titular (que antes de 000011
# fallaban con 42883 y 42702) sin la devolución si la
# decisión no la concede (aunque se falsee la variable de transacción) y con
# ella si la concede; listas ampliadas o parciales rechazadas por Dietas;
# respuesta de mutación (proyección exacta) según la variable que fija la
# autorización; cotejo del borrado propio; y lectura del revisor con y sin
# el campo.
set -Eeuo pipefail
base=${1:?falta BASE.dump}
roles=${2:?falta ROLES.sql}
[[ -s $base && -s $roles ]] || { echo 'Base o roles vacíos' >&2; exit 2; }
dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(CDPATH='' cd -- "$dir/../../../.." && pwd)
pg=$repo/deploy/postgresql
m75=$pg/autorizacion_atestada_v3/migraciones/000075_campo_devolucion_documento_dietas.up.sql
up11=$pg/dietas_borradores/migraciones/000011_campo_devolucion_decision.up.sql
down11=$pg/dietas_borradores/migraciones/000011_campo_devolucion_decision.down.sql
motor=${MOTOR_CONTENEDORES:-docker}
tmp=$(mktemp -d)
C="vec-dietas-000011-$$"
limpiar() { "$motor" rm -f "$C" >/dev/null 2>&1 || true; rm -rf -- "$tmp"; }
trap limpiar EXIT INT TERM
sed -E "s/ PASSWORD '[^']*'//; s/ PASSWORD [^ ;]+//" "$roles" | grep -vE '^(CREATE|ALTER) ROLE postgres( |;)|^\\(un)?restrict' > "$tmp/roles.sql"
if grep -qi 'password' "$tmp/roles.sql"; then echo 'No se pudieron retirar las contraseñas' >&2; exit 2; fi

fallo() { echo "FALLO: $*" >&2; exit 1; }
ok() { echo "OK $*"; }
sql() { "$motor" exec -i "$C" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
val() { "$motor" exec "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
esperar() {
  for _ in $(seq 1 120); do
    if "$motor" exec "$C" psql -X -qAt -U postgres -c 'SELECT 1' >/dev/null 2>&1; then
      sleep 0.5
      "$motor" exec "$C" psql -X -qAt -U postgres -c 'SELECT 1' >/dev/null 2>&1 && return 0
    fi
    sleep 0.3
  done
  fallo 'PostgreSQL no disponible'
}
nucleo="vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
firma_ad3() {
  val "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||md5(pg_get_constraintdef(c.oid))||
   (SELECT md5(string_agg(p.oid::regprocedure::text||coalesce(p.proacl::text,'')||md5(p.prosrc)||p.prosecdef||coalesce(p.proconfig::text,''),'|' ORDER BY p.oid::regprocedure::text))
      FROM pg_proc p WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace)
   FROM pg_constraint c WHERE c.conname='clave_capacidad_version_audiencia_consumo_check'"
}
firma_dietas() {
  val "SELECT md5(string_agg(p.oid::regprocedure::text||':'||md5(p.prosrc)||':'||coalesce(p.proacl::text,'')||':'||p.prosecdef
     ||':'||p.proowner::regrole::text||':'||coalesce(p.proconfig::text,''),'|' ORDER BY p.oid::regprocedure::text))
   FROM pg_proc p WHERE p.pronamespace='vec_dietas'::regnamespace"
}
consumos() {
  val "SELECT (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)||'|'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.atestacion_decision_v3)||'|'||
   (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)"
}

"$motor" run -d --rm --network none --name "$C" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
esperar
[[ $(val 'SHOW server_version_num') == 180004 ]] || fallo 'no es PostgreSQL 18.4'
sql -o /dev/null < "$tmp/roles.sql" || fallo 'roles'
"$motor" exec -i "$C" pg_restore -U postgres -d postgres < "$base" >/dev/null 2>&1 || true
[[ $(val "SELECT to_regprocedure('$nucleo') IS NOT NULL AND to_regclass('vec_dietas.borrador_comision') IS NULL") == t ]] \
  || fallo 'la base no tiene el núcleo AD3 o ya tiene Dietas 000001'
ok 'base VEC restaurada sin Dietas 000001'

# Roles NOLOGIN de cronos_v1 000001, preimagen de AD3-53.
sql -o /dev/null <<'SQL' || fallo 'roles de cronos_v1 000001'
DO $$ BEGIN
 IF to_regrole('vec_cronos_v1_propietario') IS NULL THEN CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; END IF;
 IF to_regrole('vec_cronos_v1_ejecutor') IS NULL THEN CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; END IF;
END $$;
SQL
cadena=(
 dietas_borradores/roles_up.sql
 personal/migraciones/000007_relacion_empleado_dietas.up.sql
 autorizacion_atestada_v3/migraciones/000049_consumidor_personal_dietas.up.sql
 autorizacion_atestada_v3/migraciones/000050_acceso_rutas_dietas.up.sql
 personal/migraciones/000008_consulta_relaciones_propias_dietas.up.sql
 personal/migraciones/000009_asignacion_dietas.up.sql
 autorizacion_atestada_v3/migraciones/000051_consumidor_organizacion_historica.up.sql
 autorizacion_atestada_v3/migraciones/000052_consumidor_importacion_organizacion.up.sql
 dietas_borradores/migraciones/000001_borrador_comision_durable.up.sql
 dietas_borradores/migraciones/000002_tarifas_provisionales.up.sql
 dietas_borradores/migraciones/000003_consulta_tarifas_provisionales.up.sql
 dietas_borradores/migraciones/000004_calculo_comision.up.sql
 dietas_borradores/migraciones/000005_auditoria_frontera.up.sql
 autorizacion_atestada_v3/migraciones/000059_consumidor_documento_dietas.up.sql
 personal/migraciones/000012_asignacion_dietas.up.sql
 personal/migraciones/000013_auditoria_frontera_asignacion_dietas.up.sql
 dietas_borradores/migraciones/000006_documento_comision.up.sql
 dietas_borradores/migraciones/000007_circuito_comision.up.sql
 autorizacion_atestada_v3/migraciones/000053_consumidores_cronos_empleado.up.sql
 autorizacion_atestada_v3/migraciones/000080_consumidor_revisor_documento_dietas.up.sql
 dietas_borradores/migraciones/000008_revision_circuito_comision.up.sql
 dietas_borradores/migraciones/000009_otros_gastos_justificados.up.sql
 dietas_borradores/migraciones/000010_devolucion_reenvio_comision.up.sql
)
for f in "${cadena[@]}"; do
  sql -o /dev/null < "$pg/$f" 2>"$tmp/err" || { cat "$tmp/err" >&2; fallo "cadena: $f"; }
done
ok 'cadena real instalada: Dietas 000001-000010, AD3-49..53, AD3-59, AD3-80, Personal 7-9 y 12-13'

# Listas: base de Dietas 000006/000008 y la misma con la devolución.
MUT='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'
MUT_DEV='["comision.calculo","comision.centro_ref","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.unidad_ref","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.regla_huella_sha256","recibo.regla_ref","recibo.repeticion","recibo.version"]'
MUT_59='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'
CON=$(python3 -c 'import json,sys; b=json.loads(sys.argv[1]); print(json.dumps(sorted(b+["items."+c for c in b]+["siguiente_cursor"]),separators=(",",":")))' "$MUT")
CON_DEV=$(python3 -c 'import json,sys; b=json.loads(sys.argv[1]); print(json.dumps(sorted(b+["items."+c for c in b]+["siguiente_cursor"]),separators=(",",":")))' "$MUT_DEV")
CON_59=$(python3 -c 'import json,sys; b=json.loads(sys.argv[1]); print(json.dumps(sorted(b+["items."+c for c in b]+["siguiente_cursor"]),separators=(",",":")))' "$MUT_59")
CON_PARCIAL=$(python3 -c 'import json,sys; b=json.loads(sys.argv[1]); print(json.dumps(sorted(b+["comision.devolucion"]),separators=(",",":")))' "$CON")
REV='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'
REV_DEV='["comision.calculo","comision.codigos_ruta","comision.devolucion","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.hora_fin","comision.hora_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.rutas","comision.vehiculo_propio","comision.version","resultado"]'
mas() { python3 -c 'import json,sys; print(json.dumps(sorted(json.loads(sys.argv[1])+[sys.argv[2]]),separators=(",",":")))' "$1" "$2"; }
al_final() { python3 -c 'import json,sys; l=json.loads(sys.argv[1]); l.remove(sys.argv[2]); print(json.dumps(l+[sys.argv[2]],separators=(",",":")))' "$1" "$2"; }
[[ $(python3 -c 'import json,sys; print(len(json.loads(sys.argv[1])),len(json.loads(sys.argv[2])),len(json.loads(sys.argv[3])))' "$CON" "$CON_DEV" "$CON_59") == '45 47 37' ]] \
  || fallo 'listas de consulta mal construidas'

# Sondas de las fachadas AD3 reales con el login exacto del ejecutor Dietas.
sql -o /dev/null < "$pg/autorizacion_atestada_v3/pruebas_sql/dietas_documento_ad3_000059_sondas.sql" || fallo 'sondas AD3-59'
sql -o /dev/null < "$dir/campo_devolucion_000011_sondas.sql" || fallo 'sondas AD3-75'
sql -o /dev/null <<'SQL' || fallo 'logins de sonda'
CREATE ROLE vec_prueba75_dietas LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba75_dietas WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba75_doble LOGIN INHERIT NOBYPASSRLS;
GRANT vec_dietas_ejecutor TO vec_prueba75_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_personal_d7_ejecutor TO vec_prueba75_doble WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE vec_prueba75_d7 LOGIN INHERIT NOBYPASSRLS;
GRANT vec_personal_d7_ejecutor TO vec_prueba75_d7 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT USAGE ON SCHEMA prueba59 TO vec_prueba75_dietas, vec_prueba75_doble, vec_prueba75_d7;
GRANT EXECUTE ON FUNCTION prueba59.consumir_dietas(text,text) TO vec_prueba75_dietas, vec_prueba75_doble, vec_prueba75_d7;
SQL
# sonda FACHADA LISTA [LOGIN]: resultado de la fachada real con esa lista.
n_sonda=0
sonda() {
  local fachada=$1 campos=$2 login=${3:-vec_prueba75_dietas} op aud tipo fin
  case $fachada in
    documento) op=dietas.borrador.propio.editar aud=vec_dietas.borrador_propio.editar.v1 tipo=comision_borrador fin=editar_borrador_propio ;;
    documento_consulta) op=dietas.documento.propio.consultar aud=vec_dietas.documento_propio.consultar.v1 tipo=comision_borrador fin=consultar_documento_propio_dietas ;;
    revisor_documento) op=dietas.circuito.documento.consultar aud=vec_dietas.circuito.documento.consultar.v1 tipo=documento_dietas fin=revisar_documento_circuito_dietas ;;
  esac
  n_sonda=$((n_sonda+1))
  val "SELECT prueba59.preparar_campos('s$n_sonda','$op','$aud','$tipo','$fin','$campos'::jsonb)" >/dev/null || fallo "material de sonda $fachada"
  { "$motor" exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U "$login" -d postgres 2>&1 || true; } <<SQL | sed -n 's/^.*ERROR: *//p;/|/p' | head -1
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT * FROM prueba59.consumir_dietas('registrar_y_consumir_dietas_${fachada}_v3_atestada','s$n_sonda');
COMMIT;
SQL
}
admitida='capacidad VEC-AD-3 rechazada'
den_doc='AD3-59: operación documento Dietas denegada'
den_con='AD3-59: consulta documento Dietas denegada'
den_rev='AD3-80: lectura del revisor Dietas denegada'
espera() { local r; r=$(sonda "$1" "$2" "${4:-vec_prueba75_dietas}"); [[ $r == "$3" ]] || fallo "$1 con $5: «$r», se esperaba «$3»"; }
filas=$(consumos)

espera documento "$MUT_DEV" "$den_doc" '' 'devolución antes de AD3-75'
espera documento_consulta "$CON_DEV" "$den_con" '' 'devolución antes de AD3-75'
espera revisor_documento "$REV_DEV" "$den_rev" '' 'devolución antes de AD3-75'
espera documento_consulta "$CON" "$den_con" '' 'lista de Dietas 000006 antes de AD3-75'
espera documento "$MUT" "$den_doc" '' 'lista de Dietas 000006 antes de AD3-75'
espera documento_consulta "$CON_59" "$admitida" '' 'lista antigua de AD3-59'
espera revisor_documento "$REV" "$admitida" '' 'lista base del revisor'
ok 'antes de AD3-75: ninguna lista con devolución pasa y la titular solo admite la lista antigua, que Dietas rechaza'

antes=$(firma_ad3)
sed '$s/^COMMIT;$/ROLLBACK;/' "$m75" | sql -o /dev/null || fallo 'AD3-75 en ROLLBACK'
[[ $(firma_ad3) == "$antes" ]] || fallo 'ROLLBACK de AD3-75 dejó rastro'
ok 'ROLLBACK de AD3-75 sin rastro en núcleo, audiencias, funciones ni ACL'
fachada_revisor='vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_revisor_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
sql -o /dev/null -c "GRANT EXECUTE ON FUNCTION $fachada_revisor TO vec_personal_propietario" || fallo 'abrir ACL'
if sql -o /dev/null < "$m75" 2>/dev/null; then fallo 'AD3-75 aceptada con ACL abierta'; fi
sql -o /dev/null -c "REVOKE EXECUTE ON FUNCTION $fachada_revisor FROM vec_personal_propietario" || fallo 'cerrar ACL'
[[ $(firma_ad3) == "$antes" ]] || fallo 'la preimagen alterada dejó rastro'
ok 'AD3-75 rechazada con la ACL de una fachada abierta, sin rastro'
sql -o /dev/null < "$m75" || fallo 'AD3-75 COMMIT'
despues=$(firma_ad3)
[[ $despues != "$antes" ]] || fallo 'AD3-75 sin efecto'
if sql -o /dev/null < "$m75" 2>/dev/null; then fallo 'segunda aplicación de AD3-75 aceptada'; fi
[[ $(firma_ad3) == "$despues" ]] || fallo 'la segunda aplicación alteró el estado'
ok 'AD3-75 confirmada; repetición rechazada sin cambios'

espera documento "$MUT" "$admitida" '' 'lista base de la titular'
espera documento "$MUT_DEV" "$admitida" '' 'lista con devolución de la titular'
espera documento "$MUT_59" "$den_doc" '' 'lista antigua de AD3-59'
espera documento "$(mas "$MUT_DEV" comision.administrativo_persona_ref)" "$den_doc" '' 'lista ampliada'
espera documento "$(al_final "$MUT_DEV" comision.devolucion)" "$den_doc" '' 'lista desordenada'
ok 'mutación de la titular: base o base+devolución; antigua, ampliada y desordenada denegadas'
espera documento_consulta "$CON" "$admitida" '' 'lista base de la consulta'
espera documento_consulta "$CON_DEV" "$admitida" '' 'lista con devolución de la consulta'
espera documento_consulta "$CON_59" "$den_con" '' 'lista antigua de AD3-59'
espera documento_consulta "$CON_PARCIAL" "$den_con" '' 'devolución sin items.comision.devolucion'
espera documento_consulta "$(mas "$CON_DEV" items.comision.asignacion_ref)" "$den_con" '' 'lista ampliada'
ok 'consulta de la titular: base o base+devolución (detalle e items); antigua, parcial y ampliada denegadas'
espera revisor_documento "$REV" "$admitida" '' 'lista base del revisor'
espera revisor_documento "$REV_DEV" "$admitida" '' 'lista con devolución del revisor'
espera revisor_documento "$(mas "$REV_DEV" comision.relacion_ref)" "$den_rev" '' 'lista ampliada'
espera revisor_documento "$(al_final "$REV_DEV" comision.devolucion)" "$den_rev" '' 'lista desordenada'
ok 'lectura del revisor: base o base+devolución; ampliada y desordenada denegadas'
espera revisor_documento "$REV_DEV" 'consumo VEC-AD-3 rechazado' vec_prueba75_doble 'login con dos grupos'
espera documento_consulta "$CON_DEV" 'consumo VEC-AD-3 rechazado' vec_prueba75_d7 'login ajeno'
[[ $(consumos) == "$filas" ]] || fallo 'las sondas crearon consumos'
ok 'login doble y ajeno siguen rechazados; ninguna sonda consumió'

# Dietas 000011.
previa=$(firma_dietas)
sed '$s/^COMMIT;$/ROLLBACK;/' "$up11" | sql -o /dev/null || fallo '000011 en ROLLBACK'
[[ $(firma_dietas) == "$previa" ]] || fallo 'ROLLBACK de 000011 dejó rastro'
sql -o /dev/null -c "GRANT EXECUTE ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) TO vec_dietas_ejecutor" || fallo 'abrir ACL Dietas'
if sql -o /dev/null < "$up11" 2>/dev/null; then fallo '000011 aceptada con ACL abierta'; fi
sql -o /dev/null -c "REVOKE EXECUTE ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) FROM vec_dietas_ejecutor" || fallo 'cerrar ACL Dietas'
[[ $(firma_dietas) == "$previa" ]] || fallo 'la preimagen alterada dejó rastro en Dietas'
ok '000011: ROLLBACK sin rastro y preimagen con ACL abierta rechazada'
sql -o /dev/null < "$up11" || fallo '000011 COMMIT'
nueva=$(firma_dietas)
[[ $nueva != "$previa" ]] || fallo '000011 sin efecto'
if sql -o /dev/null < "$up11" 2>/dev/null; then fallo 'segunda aplicación de 000011 aceptada'; fi
[[ $(firma_dietas) == "$nueva" ]] || fallo 'la segunda aplicación de 000011 alteró el estado'
sql -o /dev/null < "$down11" || fallo 'DOWN de 000011'
[[ $(firma_dietas) == "$previa" ]] || fallo 'DOWN de 000011 no devuelve la preimagen exacta'
if sql -o /dev/null < "$down11" 2>/dev/null; then fallo 'DOWN repetido aceptado'; fi
sql -o /dev/null < "$up11" || fallo 'reinstalación de 000011'
[[ $(firma_dietas) == "$nueva" ]] || fallo 'la reinstalación difiere de la primera'
ok '000011: COMMIT, repetición rechazada, DOWN a la preimagen exacta y reinstalación idéntica'

# A partir de aquí, dobles de consumo solo en este contenedor.
sql -o /dev/null < "$dir/campo_devolucion_000011_dobles.sql" || fallo 'dobles'
sql -o /dev/null < "$dir/devolucion_000010_preparar.sql" || fallo 'historia sintética'
# Valor (primera línea) o SQLSTATE de una consulta del login de prueba en una
# transacción SERIALIZABLE que siempre se deshace.
como_ejecutor() {
  { "$motor" exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_dietas -d postgres 2>&1 || true; } <<SQL | sed -n 's/^ERROR: *\([0-9A-Z]\{5\}\):.*/\1/p;/^ERROR/!{/^$/!p}' | head -1
\set VERBOSITY verbose
BEGIN ISOLATION LEVEL SERIALIZABLE;
\o /dev/null
${2:-}
\o
$1
ROLLBACK;
SQL
}
n=0
titular() { # OPERACION REF CAMPOS EXPRESION [PREVIO]
  n=$((n+1))
  como_ejecutor "SELECT vec_dietas.consultar_comisiones_propias_v2(s.material,s.capacidad,s.decision,convert_to('consultar','UTF8'),s.contexto,1,1,'\\x00','\\x00','\\x00','\\x00')$4 FROM prueba_000011.sellar_titular('$1','$2','$3'::jsonb,'t$n') s;" "${5:-}"
}
revisor() { # CAMPOS EXPRESION
  n=$((n+1))
  como_ejecutor "SELECT vec_dietas.consultar_documento_circuito_v1(s.material,s.capacidad,s.decision,convert_to('consultar','UTF8'),s.contexto,1,1,'\\x00','\\x00','\\x00','\\x00')$2 FROM prueba_000011.sellar_revisor('dco_DDDDDDDDDDDDDDDDDDDDDD','revision','U01','per_AAAAAAAAAAAAAAAAAAAAAA','act_admin','$1'::jsonb,'r$n') s;"
}
cotejo() { # CAMPOS
  n=$((n+1))
  como_ejecutor "SELECT vec_dietas.cotejar_recurso_documento_v2(s.material,s.capacidad,s.decision,s.contexto) FROM prueba_000011.sellar_titular('borrar','dco_BBBBBBBBBBBBBBBBBBBBBB','$1'::jsonb,'c$n') s;"
}
exacta() { # VALOR_VARIABLE (vacío: sin fijarla)
  local valor=$1 previo="SELECT set_config('vec.dietas.persona_ref','per_TTTTTTTTTTTTTTTTTTTTTT',true);"
  [[ -z $valor ]] || previo+=" SELECT set_config('vec.dietas.campo_devolucion','${valor}',true);"
  como_ejecutor "SELECT vec_dietas.proyectar_revision_exacta_v2('dco_DDDDDDDDDDDDDDDDDDDDDD',3)->'comision'->'devolucion';" "$previo"
}
esperada='{"etapa": "autorizacion", "motivo": "Falta el justificante del taxi", "version": 3, "devuelta_en": "2026-09-23T09:00:00.123456Z"}'
DOC=dco_DDDDDDDDDDDDDDDDDDDDDD

r=$(titular detalle $DOC "$CON" "->>'resultado'")
[[ $r == concedido ]] || fallo "detalle con la lista base: $r"
[[ -z $(titular detalle $DOC "$CON" "->'comision'->'devolucion'") ]] || fallo 'detalle sin el campo devuelve la devolución'
[[ $(titular detalle $DOC "$CON" "->'comision'->>'estado'") == devuelta ]] || fallo 'detalle sin el campo: estado'
[[ $(titular detalle $DOC "$CON_DEV" "->'comision'->'devolucion'") == "$esperada" ]] || fallo "detalle con el campo: $(titular detalle $DOC "$CON_DEV" "->'comision'->'devolucion'")"
[[ -z $(titular detalle $DOC "$CON" "->'comision'->'devolucion'" "SELECT set_config('vec.dietas.campo_devolucion','concedido',true);") ]] \
  || fallo 'la variable falseada antes de la consulta abre la devolución'
ok 'detalle de la titular: devolución solo con el campo concedido; la variable falseada no la abre'
r=$(titular lista '' "$CON" "->'items'")
[[ $r == *dco_DDDDDDDDDDDDDDDDDDDDDD* && $r != *devolucion* ]] || fallo "listado sin el campo: $r"
r=$(titular lista '' "$CON_DEV" "->'items'")
[[ $(python3 -c 'import json,sys; i=[x for x in json.loads(sys.argv[1]) if x["comision"]["referencia"]==sys.argv[2]]; print(json.dumps(i[0]["comision"].get("devolucion"),sort_keys=True))' "$r" $DOC) \
   == $(python3 -c 'import json,sys; print(json.dumps(json.loads(sys.argv[1]),sort_keys=True))' "$esperada") ]] || fallo "listado con el campo: $r"
[[ $(python3 -c 'import json,sys; print(sum("devolucion" in x["comision"] for x in json.loads(sys.argv[1])))' "$r") == 1 ]] || fallo 'devolución en un borrador nunca enviado'
ok 'listado de la titular: devolución en su item solo con el campo concedido'
for lista in "$CON_59" "$CON_PARCIAL" "$(mas "$CON_DEV" items.comision.asignacion_ref)"; do
  [[ $(titular detalle $DOC "$lista" "->>'resultado'") == PD003 ]] || fallo "Dietas admite una lista no admitida: $lista"
done
ok 'Dietas rechaza (PD003) la lista antigua, la parcial y la ampliada'

[[ -z $(exacta '') ]] || fallo 'proyección exacta sin variable devuelve la devolución'
[[ -z $(exacta denegado) ]] || fallo 'proyección exacta denegada devuelve la devolución'
[[ -z $(exacta si) ]] || fallo 'proyección exacta con valor desconocido devuelve la devolución'
[[ $(exacta concedido) == "$esperada" ]] || fallo "proyección exacta concedida: $(exacta concedido)"
[[ $(cotejo "$MUT") == t && $(cotejo "$MUT_DEV") == t ]] || fallo 'cotejo del borrado con la lista base o con devolución'
[[ $(cotejo "$MUT_59") == f && $(cotejo "$(mas "$MUT_DEV" comision.administrativo_persona_ref)") == f && $(cotejo "$(al_final "$MUT_DEV" comision.devolucion)") == f ]] \
  || fallo 'cotejo del borrado con lista antigua, ampliada o desordenada'
ok 'respuesta de mutación: la proyección exacta solo lleva la devolución con «concedido»; cotejo del borrado base o base+devolución'

# Corrección (v4) y reenvío (v5) a la revisión del administrativo.
sql -o /dev/null <<'SQL' || fallo 'reenvío sintético'
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,
 asignacion_ref,asignacion_version,grupo_dieta,centro_ref,administrativo_persona_ref,responsable_persona_ref,registrada_en)
SELECT comision_ref,v.version,v.estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,
 'ads_AAAAAAAAAAAAAAAAAAAAAA',1,2,'C01','per_AAAAAAAAAAAAAAAAAAAAAA','per_SSSSSSSSSSSSSSSSSSSSSS',v.en
FROM vec_dietas.comision_revision,(VALUES (4,'borrador',TIMESTAMPTZ '2026-09-23 10:00:00+00'),
 (5,'enviado_pendiente_revision',TIMESTAMPTZ '2026-09-23 11:00:00+00')) v(version,estado,en)
WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND comision_revision.version=3;
INSERT INTO vec_dietas.recibo_operacion_comision
SELECT 'rcd_00000000-0000-4000-8000-00000000000'||v.version,comision_ref,v.version,v.op,v.clave,'{}',repeat('c',64),'dec',repeat('d',64),'aud','act_titular',persona_ref,regla_ref,regla_huella_sha256,v.en
FROM vec_dietas.recibo_operacion_comision,(VALUES (4,'editar','claveCorreccionSint001',TIMESTAMPTZ '2026-09-23 10:00:00+00'),
 (5,'enviar','claveReenvioSintetic01',TIMESTAMPTZ '2026-09-23 11:00:00+00')) v(version,op,clave,en)
WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND recibo_operacion_comision.version=3;
INSERT INTO vec_dietas.historia_operacion_comision VALUES
 ('hdi_sintetico_04','dco_DDDDDDDDDDDDDDDDDDDDDD',4,'rcd_00000000-0000-4000-8000-000000000004','devuelta','borrador','editar',NULL,'act_titular',TIMESTAMPTZ '2026-09-23 10:00:00+00'),
 ('hdi_sintetico_05','dco_DDDDDDDDDDDDDDDDDDDDDD',5,'rcd_00000000-0000-4000-8000-000000000005','borrador','enviado_pendiente_revision','enviar',NULL,'act_titular',TIMESTAMPTZ '2026-09-23 11:00:00+00');
INSERT INTO vec_dietas.cola_circuito_comision VALUES
 ('dco_DDDDDDDDDDDDDDDDDDDDDD',5,'revision','per_TTTTTTTTTTTTTTTTTTTTTT','U01','ads_AAAAAAAAAAAAAAAAAAAAAA',1,'per_AAAAAAAAAAAAAAAAAAAAAA',DATE '2026-09-23',DATE '2026-09-23',TIMESTAMPTZ '2026-09-23 11:00:00+00');
COMMIT;
SQL
[[ $(revisor "$REV" "->>'resultado'") == concedido ]] || fallo "revisor con la lista base: $(revisor "$REV" "::text")"
[[ $(revisor "$REV" "->'comision'->>'version'") == 5 ]] || fallo 'versión del reenvío al revisor'
[[ -z $(revisor "$REV" "->'comision'->'devolucion'") ]] || fallo 'el revisor sin el campo recibe la devolución'
[[ $(revisor "$REV_DEV" "->'comision'->'devolucion'") == "$esperada" ]] || fallo "revisor con el campo: $(revisor "$REV_DEV" "::text")"
[[ $(revisor "$(mas "$REV_DEV" comision.relacion_ref)" "->>'resultado'") == PD003 ]] || fallo 'Dietas admite una lista ampliada del revisor'
ok 'lectura del revisor: devolución anterior solo con el campo concedido; lista ampliada rechazada'
echo 'PG18.4: AD3-75 y Dietas 000011 verificados sobre núcleo AD3 real y cadena Dietas hasta 000010.'
