#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de Dietas 000010 (D6: corregir y reenviar
# un documento devuelto), sin red. Borra el contenedor al salir.
#
# Uso: devolucion_000010_pg18.sh
#
# Instala la cadena real de Dietas 000001-000009 sobre Personal 000007 real y
# fachadas AD3/Personal de prueba (solo en este contenedor), y comprueba:
# ROLLBACK sin rastro; COMMIT con dueño, ACL y SECURITY DEFINER; repetición del
# UP rechazada sin cambios; devolución vigente proyectada con etapa, motivo,
# versión y fecha a la titular, también en corrección y en la versión exacta;
# ninguna devolución tras el reenvío; eliminación rechazada tras entrar en el
# circuito, admitida en un borrador que nunca salió y cerrada sin titular;
# mismo resultado tras reiniciar PostgreSQL; DOWN rechazado con historia de
# corrección; DOWN exacto a la preimagen sin historia; y reinstalación.
set -Eeuo pipefail
dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(CDPATH='' cd -- "$dir/../../../.." && pwd)
pg=$repo/deploy/postgresql
mig=$pg/dietas_borradores/migraciones
up=$mig/000010_devolucion_reenvio_comision.up.sql
down=$mig/000010_devolucion_reenvio_comision.down.sql
motor=${MOTOR_CONTENEDORES:-docker}
C="vec-dietas-000010-$$"
limpiar() { "$motor" rm -f "$C" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

fallo() { echo "FALLO: $*" >&2; exit 1; }
ok() { echo "OK $*"; }
sql() { "$motor" exec -i "$C" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres "$@"; }
val() { "$motor" exec "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
# Error SQLSTATE de una sentencia, o «sin_error».
estado_sql() {
  { "$motor" exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U postgres -d postgres 2>&1 || true; } <<SQL | sed -n 's/^ERROR: *\([0-9A-Z]\{5\}\):.*/\1/p;s/^FIN$/sin_error/p' | head -1
\set VERBOSITY verbose
$1
\echo FIN
SQL
}
# Proyección vista por la titular desde el login ejecutor de prueba.
titular() {
  "$motor" exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_dietas -d postgres <<SQL
BEGIN ISOLATION LEVEL SERIALIZABLE;
\o /dev/null
SELECT set_config('vec.dietas.persona_ref','per_TTTTTTTTTTTTTTTTTTTTTT',true);
\o
SELECT $1;
COMMIT;
SQL
}
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
huella_vec() {
  val "SELECT md5(string_agg(x,'|' ORDER BY x)) FROM (
    SELECT 'f:'||p.oid::regprocedure::text||':'||md5(p.prosrc)||':'||coalesce(p.proacl::text,'')||':'||p.prosecdef x
      FROM pg_proc p WHERE p.pronamespace='vec_dietas'::regnamespace
    UNION ALL SELECT 'g:'||t.tgname FROM pg_trigger t WHERE t.tgrelid='vec_dietas.comision_revision'::regclass
    UNION ALL SELECT 't:'||c.relname||':'||coalesce(c.relacl::text,'')||':'||c.relrowsecurity||c.relforcerowsecurity
      FROM pg_class c WHERE c.relnamespace='vec_dietas'::regnamespace AND c.relkind='r') s"
}
filas() {
  val "SELECT (SELECT count(*) FROM vec_dietas.comision_revision)||'|'||(SELECT count(*) FROM vec_dietas.recibo_operacion_comision)||'|'||(SELECT count(*) FROM vec_dietas.historia_operacion_comision)"
}
dev() { titular "vec_dietas.proyectar_comision_v2('dco_DDDDDDDDDDDDDDDDDDDDDD',true)->'comision'->'devolucion'"; }
exacta() { titular "vec_dietas.proyectar_revision_exacta_v2('dco_DDDDDDDDDDDDDDDDDDDDDD',$1)->'comision'->'devolucion'"; }
esperada='{"etapa": "autorizacion", "motivo": "Falta el justificante del taxi", "version": 3, "devuelta_en": "2026-09-23T09:00:00.123456Z"}'

"$motor" run -d --rm --network none --name "$C" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
esperar
[[ $(val 'SHOW server_version_num') == 180004 ]] || fallo 'no es PostgreSQL 18.4'

for f in "$pg/dietas_borradores/roles_up.sql" "$pg/personal/roles_up.sql" \
         "$dir/preparar_entorno_stub_ad3.sql" "$dir/preparar_entorno_stub_000009.sql" \
         "$pg/personal/migraciones/000007_relacion_empleado_dietas.up.sql"; do
  sql -o /dev/null < "$f" || fallo "preparación $(basename "$f")"
done
for n in 000001_borrador_comision_durable 000002_tarifas_provisionales 000003_consulta_tarifas_provisionales \
         000004_calculo_comision 000005_auditoria_frontera 000006_documento_comision \
         000007_circuito_comision 000008_revision_circuito_comision 000009_otros_gastos_justificados; do
  sql -o /dev/null < "$mig/$n.up.sql" || fallo "Dietas $n"
done
ok 'cadena real Dietas 000001-000009 instalada'

previa=$(huella_vec)
sed 's/^COMMIT;$/ROLLBACK;/' "$up" | sql -o /dev/null || fallo 'ensayo ROLLBACK'
[[ $(huella_vec) == "$previa" ]] || fallo 'ROLLBACK deja rastro'
ok 'ROLLBACK de 000010 sin rastro'

sql -o /dev/null < "$up" || fallo 'COMMIT 000010'
nueva=$(huella_vec)
[[ $nueva != "$previa" ]] || fallo 'COMMIT sin efecto'
[[ $(val "SELECT count(*) FROM pg_proc p WHERE p.oid IN ('vec_dietas.proyectar_comision_v2(text,boolean)'::regprocedure,'vec_dietas.proyectar_revision_exacta_v2(text,bigint)'::regprocedure)
   AND p.prosecdef AND p.proowner='vec_dietas_propietario'::regrole AND 'search_path=pg_catalog'=ANY(p.proconfig)
   AND p.proacl::text='{vec_dietas_propietario=X/vec_dietas_propietario}'") == 2 ]] || fallo 'proyecciones con dueño, ACL o entorno alterados'
[[ $(val "SELECT bool_and(NOT p.prosecdef AND p.proowner='vec_dietas_propietario'::regrole
   AND NOT has_function_privilege('vec_dietas_ejecutor',p.oid,'EXECUTE') AND NOT has_function_privilege('public',p.oid,'EXECUTE'))
   FROM pg_proc p WHERE p.oid IN ('vec_dietas.devolucion_vigente_comision_v1(text,bigint)'::regprocedure,'vec_dietas.impedir_eliminar_tras_circuito_v1()'::regprocedure)") == t ]] \
  || fallo 'funciones nuevas con privilegios de más'
ok 'COMMIT de 000010: dueño, ACL, entorno y SECURITY DEFINER conservados'

if sql -o /dev/null < "$up" 2>/dev/null; then fallo 'UP repetido aceptado'; fi
[[ $(huella_vec) == "$nueva" ]] || fallo 'UP repetido cambió el estado'
ok 'UP repetido rechazado sin cambios'

sql -o /dev/null < "$dir/devolucion_000010_preparar.sql" || fallo 'historia sintética'
[[ $(dev) == "$esperada" ]] || fallo "devolución vigente: $(dev)"
[[ $(titular "vec_dietas.proyectar_comision_v2('dco_DDDDDDDDDDDDDDDDDDDDDD',true)->'comision'->>'estado'") == devuelta ]] || fallo 'estado devuelta'
[[ $(exacta 3) == "$esperada" ]] || fallo "devolución en la versión exacta: $(exacta 3)"
[[ -z $(exacta 2) ]] || fallo 'la versión enviada proyecta devolución'
[[ -z $(titular "vec_dietas.proyectar_comision_v2('dco_BBBBBBBBBBBBBBBBBBBBBB',true)->'comision'->'devolucion'") ]] || fallo 'un borrador nunca enviado proyecta devolución'
ok 'la titular ve la devolución vigente con etapa, motivo, versión y fecha; nada en lo enviado ni en borradores nuevos'

# Eliminación: rechazada tras el circuito (también sin titular), admitida en
# un borrador que nunca salió. Todo dentro de transacciones que se deshacen.
antes=$(filas)
regla=$(val "SELECT regla_ref FROM vec_dietas.regla_devengo_provisional LIMIT 1")
# La fila se da explícita: con RLS sin titular un INSERT…SELECT no insertaría nada.
elim() { printf "BEGIN;%s INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,registrada_en) VALUES('%s',%s,'eliminado',DATE '2026-09-23',DATE '2026-09-23','08:00','12:00','Visita sintética','[\"GR1\",\"GR2\"]','{}','{}','%s',clock_timestamp()); ROLLBACK;" "$1" "$2" "$3" "$regla"; }
[[ $(estado_sql "$(elim '' dco_DDDDDDDDDDDDDDDDDDDDDD 4)") == PD005 ]] || fallo 'eliminación de un documento devuelto'
[[ $(estado_sql "$(elim " SET LOCAL ROLE vec_dietas_propietario; SELECT set_config('vec.dietas.persona_ref','per_TTTTTTTTTTTTTTTTTTTTTT',true);" dco_DDDDDDDDDDDDDDDDDDDDDD 4)") == PD005 ]] || fallo 'eliminación por el propietario con titular'
[[ $(estado_sql "$(elim ' SET LOCAL ROLE vec_dietas_propietario;' dco_DDDDDDDDDDDDDDDDDDDDDD 4)") == PD004 ]] || fallo 'eliminación sin titular no falla cerrado'
[[ $(estado_sql "$(elim '' dco_BBBBBBBBBBBBBBBBBBBBBB 3)") == sin_error ]] || fallo 'un borrador nunca enviado no se puede eliminar'
[[ $(filas) == "$antes" ]] || fallo 'los ensayos de eliminación dejaron filas'
ok 'eliminación rechazada tras entrar en el circuito y sin titular; admitida en un borrador nuevo'

# Corrección (v4, borrador desde devuelta): la devolución sigue a la vista.
sql -o /dev/null <<'SQL' || fallo 'corrección sintética'
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,registrada_en)
SELECT comision_ref,4,'borrador',fecha_inicio,fecha_fin,hora_inicio,hora_fin,'Visita sintética corregida',codigos_ruta,calculo,documento,regla_ref,TIMESTAMPTZ '2026-09-23 10:00:00+00'
FROM vec_dietas.comision_revision WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=3;
INSERT INTO vec_dietas.recibo_operacion_comision
SELECT 'rcd_00000000-0000-4000-8000-000000000004',comision_ref,4,'editar','claveCorreccionSint001','{}',repeat('c',64),'dec',repeat('d',64),'aud','act_titular',persona_ref,regla_ref,regla_huella_sha256,TIMESTAMPTZ '2026-09-23 10:00:00+00'
FROM vec_dietas.recibo_operacion_comision WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=3;
INSERT INTO vec_dietas.historia_operacion_comision VALUES
 ('hdi_sintetico_04','dco_DDDDDDDDDDDDDDDDDDDDDD',4,'rcd_00000000-0000-4000-8000-000000000004','devuelta','borrador','editar',NULL,'act_titular',TIMESTAMPTZ '2026-09-23 10:00:00+00');
COMMIT;
SQL
[[ $(dev) == "$esperada" && $(exacta 4) == "$esperada" ]] || fallo "devolución en corrección: $(dev)"
[[ $(titular "vec_dietas.proyectar_comision_v2('dco_DDDDDDDDDDDDDDDDDDDDDD',true)->'comision'->>'motivo'") == 'Visita sintética corregida' ]] || fallo 'versión corregida'
[[ $(estado_sql "$(elim '' dco_DDDDDDDDDDDDDDDDDDDDDD 5)") == PD005 ]] || fallo 'eliminación de un documento en corrección'
ok 'en corrección (borrador desde devuelta) la devolución sigue visible y el documento no se elimina'

con_arnes=$(huella_vec)
"$motor" restart "$C" >/dev/null
esperar
[[ $(dev) == "$esperada" && $(exacta 3) == "$esperada" && $(huella_vec) == "$con_arnes" ]] || fallo 'resultado distinto tras reiniciar'
ok 'mismo resultado tras reiniciar PostgreSQL'

if sql -o /dev/null < "$down" 2>/dev/null; then fallo 'DOWN atravesó una corrección registrada'; fi
[[ $(huella_vec) == "$con_arnes" ]] || fallo 'DOWN rechazado dejó cambios'
ok 'DOWN rechazado con una corrección registrada'

# Reenvío (v5): vuelve a revisión y la devolución deja de estar vigente,
# aunque la historia la conserva en las versiones 3 y 4.
sql -o /dev/null <<'SQL' || fallo 'reenvío sintético'
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,registrada_en)
SELECT comision_ref,5,'enviado_pendiente_revision',fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,TIMESTAMPTZ '2026-09-23 11:00:00+00'
FROM vec_dietas.comision_revision WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=4;
INSERT INTO vec_dietas.recibo_operacion_comision
SELECT 'rcd_00000000-0000-4000-8000-000000000005',comision_ref,5,'enviar','claveReenvioSintetic01','{}',repeat('c',64),'dec',repeat('d',64),'aud','act_titular',persona_ref,regla_ref,regla_huella_sha256,TIMESTAMPTZ '2026-09-23 11:00:00+00'
FROM vec_dietas.recibo_operacion_comision WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=3;
INSERT INTO vec_dietas.historia_operacion_comision VALUES
 ('hdi_sintetico_05','dco_DDDDDDDDDDDDDDDDDDDDDD',5,'rcd_00000000-0000-4000-8000-000000000005','borrador','enviado_pendiente_revision','enviar',NULL,'act_titular',TIMESTAMPTZ '2026-09-23 11:00:00+00');
COMMIT;
SQL
[[ -z $(dev) && -z $(exacta 5) ]] || fallo "devolución tras el reenvío: $(dev)"
[[ $(exacta 3) == "$esperada" && $(exacta 4) == "$esperada" ]] || fallo 'la historia perdió la devolución'
ok 'tras el reenvío no hay devolución vigente; las versiones 3 y 4 la conservan'

sql -o /dev/null <<'SQL' || fallo 'retirar historia sintética'
BEGIN;
SET LOCAL session_replication_role=replica;
DELETE FROM vec_dietas.historia_operacion_comision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.recibo_operacion_comision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.comision_revision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.numero_documento_comision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.borrador_comision WHERE referencia IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
COMMIT;
REVOKE EXECUTE ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean),
 vec_dietas.proyectar_revision_exacta_v2(text,bigint) FROM vec_prueba_dietas;
SQL

sql -o /dev/null < "$down" || fallo 'DOWN sin historia'
[[ $(huella_vec) == "$previa" ]] || fallo 'DOWN no restaura la preimagen exacta'
ok 'DOWN restaura la preimagen exacta'
sql -o /dev/null < "$up" || fallo 'reinstalación'
[[ $(huella_vec) == "$nueva" ]] || fallo 'reinstalación distinta'
ok 'reinstalación idéntica tras DOWN'
echo 'OK ensayo Dietas 000010 en PostgreSQL 18.4 desechable'
