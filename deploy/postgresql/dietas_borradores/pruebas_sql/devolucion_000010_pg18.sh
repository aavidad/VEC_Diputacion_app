#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de Dietas 000010 (D6: corregir y reenviar
# un documento devuelto), sin red. Borra el contenedor al salir.
#
# Uso: devolucion_000010_pg18.sh
#
# Instala la cadena real de Dietas 000001-000009 sobre Personal 000007 real y
# fachadas AD3/Personal de prueba (solo en este contenedor), y comprueba:
# ROLLBACK sin rastro; COMMIT con dueño, ACL, entorno y SECURITY DEFINER;
# repetición del UP rechazada sin cambios; devolución vigente proyectada con
# etapa, motivo, versión y fecha a la titular, también en corrección y en la
# versión exacta; ninguna devolución tras el reenvío; eliminación rechazada
# tras entrar en el circuito, admitida en un borrador que nunca salió y
# cerrada sin titular, también por mutar_comision_propia_v2('borrar') con la
# RLS de la titular; quien revisa el reenvío ve la devolución anterior sin
# identidad del revisor; un segundo ciclo devuelta→reenvío→devuelta proyecta
# el motivo nuevo y conserva el antiguo en su versión, y los mutantes M2
# (sin «NOT EXISTS enviar») y M2+M6 (además «ORDER BY ASC») mueren; mismo
# resultado tras reiniciar PostgreSQL; DOWN rechazado con historia de
# corrección; DOWN exacto a la preimagen sin historia; y reinstalación.
#
# Las fachadas AD3 y la revalidación Personal de la asignación son dobles
# TEST-ONLY de este contenedor (devolucion_000010_fachadas.sql); la relación
# Personal 000007 y todas las funciones Dietas son las reales.
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
    SELECT 'f:'||p.oid::regprocedure::text||':'||md5(p.prosrc)||':'||coalesce(p.proacl::text,'')||':'||p.prosecdef
      ||':'||p.proowner::regrole::text||':'||coalesce(p.proconfig::text,'') x
      FROM pg_proc p WHERE p.pronamespace='vec_dietas'::regnamespace
    UNION ALL SELECT 'g:'||t.tgname FROM pg_trigger t WHERE t.tgrelid='vec_dietas.comision_revision'::regclass
    UNION ALL SELECT 't:'||c.relname||':'||coalesce(c.relacl::text,'')||':'||c.relrowsecurity||c.relforcerowsecurity
      FROM pg_class c WHERE c.relnamespace='vec_dietas'::regnamespace AND c.relkind='r') s"
}
filas() {
  val "SELECT (SELECT count(*) FROM vec_dietas.comision_revision)||'|'||(SELECT count(*) FROM vec_dietas.recibo_operacion_comision)||'|'||(SELECT count(*) FROM vec_dietas.historia_operacion_comision)"
}
# Valor (primera línea) o SQLSTATE de una consulta del login ejecutor de
# prueba dentro de una transacción SERIALIZABLE que siempre se deshace.
como_ejecutor() {
  { "$motor" exec -i "$C" psql -X -qAt -v ON_ERROR_STOP=1 -U vec_prueba_dietas -d postgres 2>&1 || true; } <<SQL | sed -n 's/^ERROR: *\([0-9A-Z]\{5\}\):.*/\1/p;/^ERROR/!{/^$/!p}' | head -1
\set VERBOSITY verbose
BEGIN ISOLATION LEVEL SERIALIZABLE;
$1
ROLLBACK;
SQL
}
# Borrado propio real (mutar_comision_propia_v2) con la RLS de la titular.
borrar_propio() {
  como_ejecutor "SELECT vec_dietas.mutar_comision_propia_v2(s.material,s.capacidad,s.decision,convert_to('borrar','UTF8'),s.contexto,1,1,'\\x00','\\x00','\\x00','\\x00')->'comision'->>'estado' FROM prueba_000010.sellar_borrado('$1',$2,'$3') s;"
}
# Documento del circuito real (consultar_documento_circuito_v1) leído por
# quien revisa en su etapa.
consultar_revisor() {
  como_ejecutor "SELECT vec_dietas.consultar_documento_circuito_v1(s.material,s.capacidad,s.decision,convert_to('consultar','UTF8'),s.contexto,1,1,'\\x00','\\x00','\\x00','\\x00')$4 FROM prueba_000010.sellar_consulta('dco_DDDDDDDDDDDDDDDDDDDDDD','revision','U01','$1','$2','$3') s;"
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

# La precondición exige ACL y entorno exactos de las funciones sustituidas.
sql -o /dev/null -c "GRANT EXECUTE ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) TO vec_dietas_ejecutor" || fallo 'alterar ACL'
if sql -o /dev/null < "$up" 2>/dev/null; then fallo 'UP aceptado con ACL alterada'; fi
sql -o /dev/null -c "REVOKE EXECUTE ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) FROM vec_dietas_ejecutor" || fallo 'restaurar ACL'
sql -o /dev/null -c "ALTER FUNCTION vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SET lock_timeout='3s'" || fallo 'alterar entorno'
if sql -o /dev/null < "$up" 2>/dev/null; then fallo 'UP aceptado con entorno alterado'; fi
sql -o /dev/null -c "ALTER FUNCTION vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) SET lock_timeout='2s'" || fallo 'restaurar entorno'
ok 'UP rechazado con ACL o entorno alterados en la preimagen'

previa=$(huella_vec)
sed 's/^COMMIT;$/ROLLBACK;/' "$up" | sql -o /dev/null || fallo 'ensayo ROLLBACK'
[[ $(huella_vec) == "$previa" ]] || fallo 'ROLLBACK deja rastro'
ok 'ROLLBACK de 000010 sin rastro'

sql -o /dev/null < "$up" || fallo 'COMMIT 000010'
nueva=$(huella_vec)
[[ $nueva != "$previa" ]] || fallo 'COMMIT sin efecto'
[[ $(val "SELECT count(*) FROM pg_proc p WHERE p.oid IN ('vec_dietas.proyectar_comision_v2(text,boolean)'::regprocedure,'vec_dietas.proyectar_revision_exacta_v2(text,bigint)'::regprocedure)
   AND p.prosecdef AND p.proowner='vec_dietas_propietario'::regrole
   AND p.proconfig::text='{search_path=pg_catalog,row_security=on,TimeZone=UTC}'
   AND p.proacl::text='{vec_dietas_propietario=X/vec_dietas_propietario}'") == 2 ]] || fallo 'proyecciones con dueño, ACL o entorno alterados'
[[ $(val "SELECT count(*) FROM pg_proc p WHERE p.oid='vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
   AND p.prosecdef AND p.proowner='vec_dietas_propietario'::regrole
   AND p.proconfig::text='{search_path=pg_catalog,row_security=on,TimeZone=UTC,lock_timeout=2s}'
   AND p.proacl::text='{vec_dietas_propietario=X/vec_dietas_propietario,vec_dietas_ejecutor=X/vec_dietas_propietario}'") == 1 ]] || fallo 'documento del circuito con dueño, ACL o entorno alterados'
[[ $(val "SELECT bool_and(NOT p.prosecdef AND p.proowner='vec_dietas_propietario'::regrole
   AND p.proacl::text='{vec_dietas_propietario=X/vec_dietas_propietario}' AND p.proconfig::text='{search_path=pg_catalog}')
   FROM pg_proc p WHERE p.oid IN ('vec_dietas.devolucion_vigente_comision_v1(text,bigint)'::regprocedure,'vec_dietas.devolucion_anterior_comision_v1(text,bigint)'::regprocedure,'vec_dietas.impedir_eliminar_tras_circuito_v1()'::regprocedure)") == t ]] \
  || fallo 'funciones nuevas con privilegios de más'
ok 'COMMIT de 000010: dueño, ACL, entorno y SECURITY DEFINER conservados'

if sql -o /dev/null < "$up" 2>/dev/null; then fallo 'UP repetido aceptado'; fi
[[ $(huella_vec) == "$nueva" ]] || fallo 'UP repetido cambió el estado'
ok 'UP repetido rechazado sin cambios'

sql -o /dev/null < "$dir/devolucion_000010_preparar.sql" || fallo 'historia sintética'
sql -o /dev/null < "$dir/devolucion_000010_fachadas.sql" || fallo 'fachadas de prueba'
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

# Borrado por la operación nominal real: la titular elimina un borrador que
# nunca salió; el documento en corrección falla PD005 aunque su última
# versión sea «borrador». Todo se deshace.
antes=$(filas)
[[ $(borrar_propio dco_BBBBBBBBBBBBBBBBBBBBBB 2 claveBorradoNuevo0001) == eliminado ]] || fallo "borrado propio de un borrador nuevo: $(borrar_propio dco_BBBBBBBBBBBBBBBBBBBBBB 2 claveBorradoNuevo0002)"
[[ $(borrar_propio dco_DDDDDDDDDDDDDDDDDDDDDD 4 claveBorradoCorregido01) == PD005 ]] || fallo "borrado propio de un documento en corrección: $(borrar_propio dco_DDDDDDDDDDDDDDDDDDDDDD 4 claveBorradoCorregido02)"
[[ $(filas) == "$antes" ]] || fallo 'los borrados propios dejaron filas'
ok 'mutar_comision_propia_v2(borrar) con RLS de la titular: borrador nuevo eliminado; en corrección PD005'

con_arnes=$(huella_vec)
"$motor" restart "$C" >/dev/null
esperar
[[ $(dev) == "$esperada" && $(exacta 3) == "$esperada" && $(huella_vec) == "$con_arnes" ]] || fallo 'resultado distinto tras reiniciar'
ok 'mismo resultado tras reiniciar PostgreSQL'

if sql -o /dev/null < "$down" 2>/dev/null; then fallo 'DOWN atravesó una corrección registrada'; fi
[[ $(huella_vec) == "$con_arnes" ]] || fallo 'DOWN rechazado dejó cambios'
ok 'DOWN rechazado con una corrección registrada'

# Reenvío (v5): vuelve a revisión del administrativo con su asignación y
# su cola; la devolución deja de estar vigente, aunque la historia la
# conserva en las versiones 3 y 4.
sql -o /dev/null <<'SQL' || fallo 'reenvío sintético'
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,
 asignacion_ref,asignacion_version,grupo_dieta,centro_ref,administrativo_persona_ref,responsable_persona_ref,registrada_en)
SELECT comision_ref,5,'enviado_pendiente_revision',fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,
 'ads_AAAAAAAAAAAAAAAAAAAAAA',1,2,'C01','per_AAAAAAAAAAAAAAAAAAAAAA','per_SSSSSSSSSSSSSSSSSSSSSS',TIMESTAMPTZ '2026-09-23 11:00:00+00'
FROM vec_dietas.comision_revision WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=4;
INSERT INTO vec_dietas.recibo_operacion_comision
SELECT 'rcd_00000000-0000-4000-8000-000000000005',comision_ref,5,'enviar','claveReenvioSintetic01','{}',repeat('c',64),'dec',repeat('d',64),'aud','act_titular',persona_ref,regla_ref,regla_huella_sha256,TIMESTAMPTZ '2026-09-23 11:00:00+00'
FROM vec_dietas.recibo_operacion_comision WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=3;
INSERT INTO vec_dietas.historia_operacion_comision VALUES
 ('hdi_sintetico_05','dco_DDDDDDDDDDDDDDDDDDDDDD',5,'rcd_00000000-0000-4000-8000-000000000005','borrador','enviado_pendiente_revision','enviar',NULL,'act_titular',TIMESTAMPTZ '2026-09-23 11:00:00+00');
INSERT INTO vec_dietas.cola_circuito_comision VALUES
 ('dco_DDDDDDDDDDDDDDDDDDDDDD',5,'revision','per_TTTTTTTTTTTTTTTTTTTTTT','U01','ads_AAAAAAAAAAAAAAAAAAAAAA',1,'per_AAAAAAAAAAAAAAAAAAAAAA',DATE '2026-09-23',DATE '2026-09-23',TIMESTAMPTZ '2026-09-23 11:00:00+00');
COMMIT;
SQL
[[ -z $(dev) && -z $(exacta 5) ]] || fallo "devolución tras el reenvío: $(dev)"
[[ $(exacta 3) == "$esperada" && $(exacta 4) == "$esperada" ]] || fallo 'la historia perdió la devolución'
ok 'tras el reenvío no hay devolución vigente; las versiones 3 y 4 la conservan'

# Quien revisa el reenvío ve la devolución anterior (etapa, motivo, versión y
# fecha) por la función nominal real, sin la identidad de quien devolvió; la
# titular no lee su documento como revisora.
[[ $(consultar_revisor per_AAAAAAAAAAAAAAAAAAAAAA act_admin revision01 "->'comision'->'devolucion'") == "$esperada" ]] \
  || fallo "devolución anterior al revisor: $(consultar_revisor per_AAAAAAAAAAAAAAAAAAAAAA act_admin revision02 "->'comision'->'devolucion'")"
[[ $(consultar_revisor per_AAAAAAAAAAAAAAAAAAAAAA act_admin revision03 "->'comision'->>'version'") == 5 ]] || fallo 'versión del reenvío al revisor'
[[ $(consultar_revisor per_AAAAAAAAAAAAAAAAAAAAAA act_admin revision04 "::text LIKE '%act_%'") == f ]] || fallo 'el documento del revisor expone actores'
[[ $(consultar_revisor per_TTTTTTTTTTTTTTTTTTTTTT per_TTTTTTTTTTTTTTTTTTTTTT revision05 "->>'resultado'") == no_encontrado ]] || fallo 'la titular revisa lo suyo'
ok 'quien revisa el reenvío ve la devolución anterior sin identidad del revisor previo'

# Segundo ciclo: revisión aprueba (v6), autorización vuelve a devolver con
# otro motivo (v7). La devolución vigente es la nueva y la versión 4
# conserva la antigua.
sql -o /dev/null <<'SQL' || fallo 'segundo ciclo sintético'
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,
 asignacion_ref,asignacion_version,grupo_dieta,centro_ref,administrativo_persona_ref,responsable_persona_ref,registrada_en)
SELECT comision_ref,x.v,x.estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,
 asignacion_ref,asignacion_version,grupo_dieta,centro_ref,administrativo_persona_ref,responsable_persona_ref,x.en
FROM vec_dietas.comision_revision,(VALUES (6,'pendiente_autorizacion',TIMESTAMPTZ '2026-09-24 08:00:00+00'),
 (7,'devuelta',TIMESTAMPTZ '2026-09-24 09:00:00+00')) x(v,estado,en)
WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=5;
INSERT INTO vec_dietas.recibo_operacion_comision
SELECT x.recibo,comision_ref,x.v,x.op,x.clave,'{}',repeat('c',64),'dec',repeat('d',64),'aud',x.actor,persona_ref,regla_ref,regla_huella_sha256,x.en
FROM vec_dietas.recibo_operacion_comision,(VALUES
 ('rcd_00000000-0000-4000-8000-000000000006',6,'circuito_revision','claveRevisionSint00002','act_admin',TIMESTAMPTZ '2026-09-24 08:00:00+00'),
 ('rcd_00000000-0000-4000-8000-000000000007',7,'circuito_autorizacion','claveDevolucionSint0002','act_responsable',TIMESTAMPTZ '2026-09-24 09:00:00+00')) x(recibo,v,op,clave,actor,en)
WHERE comision_ref='dco_DDDDDDDDDDDDDDDDDDDDDD' AND version=3;
INSERT INTO vec_dietas.historia_operacion_comision VALUES
 ('hdi_sintetico_06','dco_DDDDDDDDDDDDDDDDDDDDDD',6,'rcd_00000000-0000-4000-8000-000000000006','enviado_pendiente_revision','pendiente_autorizacion','circuito_revision',NULL,'act_admin',TIMESTAMPTZ '2026-09-24 08:00:00+00'),
 ('hdi_sintetico_07','dco_DDDDDDDDDDDDDDDDDDDDDD',7,'rcd_00000000-0000-4000-8000-000000000007','pendiente_autorizacion','devuelta','circuito_autorizacion','Falta la firma del parte de viaje','act_responsable',TIMESTAMPTZ '2026-09-24 09:00:00.654321+00');
COMMIT;
SQL
segunda='{"etapa": "autorizacion", "motivo": "Falta la firma del parte de viaje", "version": 7, "devuelta_en": "2026-09-24T09:00:00.654321Z"}'
ciclo_correcto() {
  [[ $(dev) == "$segunda" && $(exacta 7) == "$segunda" && $(exacta 4) == "$esperada" && $(exacta 3) == "$esperada" &&
     -z $(exacta 5) && -z $(exacta 6) ]]
}
ciclo_correcto || fallo "segundo ciclo: vigente $(dev); v4 $(exacta 4); v5 $(exacta 5)"
ok 'segundo ciclo: vigente y v7 con el motivo nuevo; v3 y v4 con el antiguo; nada en v5 y v6'

# Mutantes de devolucion_vigente_comision_v1 sobre la historia de dos ciclos.
# Se aplican con pg_get_functiondef y se restaura la definición exacta.
original=$(val "SELECT encode(convert_to(pg_get_functiondef('vec_dietas.devolucion_vigente_comision_v1(text,bigint)'::regprocedure),'UTF8'),'base64')")
antes_mutantes=$(huella_vec)
mutante() {
  sql -o /dev/null <<SQL || fallo "aplicar mutante $1"
DO \$m\$
DECLARE def text:=pg_get_functiondef('vec_dietas.devolucion_vigente_comision_v1(text,bigint)'::regprocedure); orig text;
BEGIN
 orig:=def;
 IF '$1' LIKE '%M2%' THEN
  def:=regexp_replace(def,'\s+AND NOT EXISTS \(SELECT 1 FROM vec_dietas\.historia_operacion_comision e.*?e\.version<=p_hasta\)','','s');
 END IF;
 IF '$1' LIKE '%M6%' THEN def:=replace(def,'ORDER BY h.version DESC','ORDER BY h.version ASC'); END IF;
 IF def=orig OR ('$1' LIKE '%M2%' AND def LIKE '%NOT EXISTS%') OR ('$1' LIKE '%M6%' AND def NOT LIKE '%h.version ASC%') THEN
  RAISE EXCEPTION 'mutante $1 no aplicado'; END IF;
 EXECUTE def;
END \$m\$;
SQL
}
restaurar() {
  sql -o /dev/null -c "DO \$r\$ BEGIN EXECUTE convert_from(decode('$original','base64'),'UTF8'); END \$r\$" || fallo 'restaurar la función original'
  [[ $(huella_vec) == "$antes_mutantes" ]] || fallo 'la restauración tras un mutante no es exacta'
}
mutante M2+M6
if ciclo_correcto; then restaurar; fallo 'el mutante M2+M6 sobrevive'; fi
restaurar
# Por separado son equivalentes en una historia válida: tras un reenvío solo
# se vuelve a «devuelta» o «borrador» con una devolución nueva, que es la
# de versión mayor (M2) y la única sin reenvío posterior (M6).
aislados=''
for m in M2 M6; do
  mutante "$m"
  if ciclo_correcto; then aislados+=" $m=equivalente"; else aislados+=" $m=muere"; fi
  restaurar
done
ok "mutante M2+M6 muerto; aislados:$aislados"

sql -o /dev/null <<'SQL' || fallo 'retirar historia sintética'
BEGIN;
SET LOCAL session_replication_role=replica;
DELETE FROM vec_dietas.cola_circuito_comision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.historia_operacion_comision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.recibo_operacion_comision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.comision_revision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.numero_documento_comision WHERE comision_ref IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
DELETE FROM vec_dietas.borrador_comision WHERE referencia IN ('dco_DDDDDDDDDDDDDDDDDDDDDD','dco_BBBBBBBBBBBBBBBBBBBBBB');
COMMIT;
REVOKE EXECUTE ON FUNCTION vec_dietas.proyectar_comision_v2(text,boolean),
 vec_dietas.proyectar_revision_exacta_v2(text,bigint) FROM vec_prueba_dietas;
SQL

# DOWN exige también dueño, SECURITY DEFINER, ACL y entorno exactos.
sin_historia=$(huella_vec)
for alterar in "GRANT EXECUTE ON FUNCTION vec_dietas.proyectar_revision_exacta_v2(text,bigint) TO vec_dietas_ejecutor|REVOKE EXECUTE ON FUNCTION vec_dietas.proyectar_revision_exacta_v2(text,bigint) FROM vec_dietas_ejecutor" \
               "ALTER FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) SECURITY INVOKER|ALTER FUNCTION vec_dietas.proyectar_comision_v2(text,boolean) SECURITY DEFINER" \
               "ALTER FUNCTION vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO postgres|ALTER FUNCTION vec_dietas.consultar_documento_circuito_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario"; do
  sql -o /dev/null -c "${alterar%%|*}" || fallo "alterar: ${alterar%%|*}"
  if sql -o /dev/null < "$down" 2>/dev/null; then fallo "DOWN aceptado tras: ${alterar%%|*}"; fi
  sql -o /dev/null -c "${alterar#*|}" || fallo "restaurar: ${alterar#*|}"
done
[[ $(huella_vec) == "$sin_historia" ]] || fallo 'las alteraciones del DOWN no se restauraron'
ok 'DOWN rechazado con ACL, SECURITY DEFINER o dueño alterados'
sql -o /dev/null < "$down" || fallo 'DOWN sin historia'
[[ $(huella_vec) == "$previa" ]] || fallo 'DOWN no restaura la preimagen exacta'
ok 'DOWN restaura la preimagen exacta'
sql -o /dev/null < "$up" || fallo 'reinstalación'
[[ $(huella_vec) == "$nueva" ]] || fallo 'reinstalación distinta'
ok 'reinstalación idéntica tras DOWN'
echo 'OK ensayo Dietas 000010 en PostgreSQL 18.4 desechable'
