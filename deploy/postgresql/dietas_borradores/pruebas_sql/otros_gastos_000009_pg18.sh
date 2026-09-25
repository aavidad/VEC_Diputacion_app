#!/usr/bin/env bash
# Ensayo PostgreSQL 18.4 desechable de Dietas 000009 (D5: otros medios y
# gastos con justificante), sin red. Borra el contenedor al salir.
#
# Uso: otros_gastos_000009_pg18.sh
#
# Instala la cadena real de Dietas 000001-000008 sobre Personal 000007 real y
# fachadas AD3/Personal de prueba (solo en este contenedor), y comprueba:
# ROLLBACK sin rastro; COMMIT con dueño, ACL, RLS e inmutabilidad; repetición
# del UP rechazada sin cambios; validación positiva y negativa de líneas D5;
# mismo resultado tras reiniciar PostgreSQL; DOWN rechazado con una línea D5
# guardada; DOWN exacto a la preimagen sin historia; y reinstalación.
set -Eeuo pipefail
dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
repo=$(CDPATH='' cd -- "$dir/../../../.." && pwd)
pg=$repo/deploy/postgresql
mig=$pg/dietas_borradores/migraciones
motor=${MOTOR_CONTENEDORES:-docker}
C="vec-dietas-000009-$$"
limpiar() { "$motor" rm -f "$C" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

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
fn="vec_dietas.validar_documento_v2(jsonb)"
huella_fn() { val "SELECT md5(p.prosrc)||':'||r.rolname||':'||coalesce(p.proacl::text,'')||':'||p.prosecdef FROM pg_proc p JOIN pg_roles r ON r.oid=p.proowner WHERE p.oid='$fn'::regprocedure"; }
huella_vec() {
  val "SELECT md5(string_agg(x,'|' ORDER BY x)) FROM (
    SELECT 'f:'||p.oid::regprocedure::text||':'||md5(p.prosrc)||':'||coalesce(p.proacl::text,'') x
      FROM pg_proc p WHERE p.pronamespace='vec_dietas'::regnamespace
    UNION ALL SELECT 't:'||c.relname||':'||coalesce(c.relacl::text,'')||':'||c.relrowsecurity||c.relforcerowsecurity
      FROM pg_class c WHERE c.relnamespace='vec_dietas'::regnamespace AND c.relkind='r') s"
}

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
         000007_circuito_comision 000008_revision_circuito_comision; do
  sql -o /dev/null < "$mig/$n.up.sql" || fallo "Dietas $n"
done
ok 'cadena real Dietas 000001-000008 instalada'

previa_fn=$(huella_fn)
previa_vec=$(huella_vec)
[[ ${previa_fn%%:*} == $(grep -o "md5(p.prosrc)='[0-9a-f]*'" "$mig/000009_otros_gastos_justificados.up.sql" | cut -d"'" -f2) ]] \
  || fallo "la preimagen de 000009 no coincide con 000006 (${previa_fn%%:*})"
ok 'preimagen de validar_documento_v2 igual a 000006'

sed 's/^COMMIT;$/ROLLBACK;/' "$mig/000009_otros_gastos_justificados.up.sql" | sql -o /dev/null || fallo 'ensayo ROLLBACK'
[[ $(huella_vec) == "$previa_vec" ]] || fallo 'ROLLBACK deja rastro'
ok 'ROLLBACK de 000009 sin rastro'

sql -o /dev/null < "$mig/000009_otros_gastos_justificados.up.sql" || fallo 'COMMIT 000009'
nueva_fn=$(huella_fn)
[[ ${nueva_fn#*:} == "${previa_fn#*:}" ]] || fallo "dueño, ACL o SECURITY DEFINER cambiados: $nueva_fn"
[[ ${nueva_fn%%:*} != "${previa_fn%%:*}" ]] || fallo 'cuerpo sin cambiar'
[[ ${nueva_fn%%:*} == $(grep -o "md5(p.prosrc)='[0-9a-f]*'" "$mig/000009_otros_gastos_justificados.down.sql" | cut -d"'" -f2) ]] \
  || fallo "el DOWN no reconoce el cuerpo instalado (${nueva_fn%%:*})"
[[ $(val "SELECT count(*) FROM vec_dietas.tipo_otro_gasto_provisional") == 10 ]] || fallo 'catálogo incompleto'
[[ $(val "SELECT bool_and(relrowsecurity AND relforcerowsecurity) FROM pg_class WHERE oid IN ('vec_dietas.catalogo_otros_gastos_provisional'::regclass,'vec_dietas.tipo_otro_gasto_provisional'::regclass)") == t ]] || fallo 'RLS no forzada'
[[ $(val "SELECT has_table_privilege('vec_dietas_ejecutor','vec_dietas.tipo_otro_gasto_provisional','SELECT') OR has_table_privilege('vec_dietas_ejecutor','vec_dietas.catalogo_otros_gastos_provisional','SELECT') OR has_function_privilege('vec_dietas_ejecutor','$fn','EXECUTE')") == f ]] || fallo 'el ejecutor accede al catálogo o a la validación'
ok 'COMMIT de 000009: dueño, ACL, RLS y catálogo'

if sql -o /dev/null < "$mig/000009_otros_gastos_justificados.up.sql" 2>/dev/null; then fallo 'UP repetido aceptado'; fi
[[ $(huella_fn) == "$nueva_fn" && $(val "SELECT count(*) FROM vec_dietas.tipo_otro_gasto_provisional") == 10 ]] || fallo 'UP repetido cambió el estado'
ok 'UP repetido rechazado sin cambios'

if val "UPDATE vec_dietas.tipo_otro_gasto_provisional SET clase='otro_gasto' WHERE codigo='taxi'" >/dev/null 2>&1; then fallo 'catálogo mutable'; fi
if val "DELETE FROM vec_dietas.catalogo_otros_gastos_provisional" >/dev/null 2>&1; then fallo 'catálogo borrable'; fi
ok 'catálogo inmutable'

sql < "$dir/otros_gastos_000009.sql" || fallo 'casos de validación D5'
ok 'casos D5 positivos y negativos'

"$motor" restart "$C" >/dev/null
esperar
[[ $(huella_fn) == "$nueva_fn" ]] || fallo 'la función cambió tras reiniciar'
sql < "$dir/otros_gastos_000009.sql" || fallo 'casos D5 tras reiniciar'
ok 'mismo resultado tras reiniciar PostgreSQL'

# Una revisión con línea D5 guardada protege el catálogo: el DOWN se rechaza.
sql -o /dev/null <<'SQL' || fallo 'preparar historia D5'
BEGIN;
SET LOCAL session_replication_role=replica;
INSERT INTO vec_dietas.borrador_comision(referencia,persona_ref,empleado_ref,relacion_ref,unidad_ref,relacion_version,procedencia_acto_ref,fuente_ref,fuente_version,fecha_inicio,fecha_fin,motivo,codigos_ruta,clave_idempotencia,huella_semantica_sha256,creada_en)
VALUES ('dco_HHHHHHHHHHHHHHHHHHHHHH','per_AAAAAAAAAAAAAAAAAAAAAA','emp_BBBBBBBBBBBBBBBBBBBBBB','rel_CCCCCCCCCCCCCCCCCCCCCC','U1',1,'acto','fuente',1,DATE '2026-09-23',DATE '2026-09-23','Visita sintética','["GR1","GR2"]','clave_historia_d5_0001',repeat('a',64),clock_timestamp());
INSERT INTO vec_dietas.comision_revision(comision_ref,version,estado,fecha_inicio,fecha_fin,hora_inicio,hora_fin,motivo,codigos_ruta,calculo,documento,regla_ref,registrada_en)
VALUES ('dco_HHHHHHHHHHHHHHHHHHHHHH',2,'borrador',DATE '2026-09-23',DATE '2026-09-23','08:00','12:00','Visita sintética','["GR1","GR2"]','{}',
 '{"lineas":[{"tipo":"otro_medio","tipo_gasto":"taxi","catalogo_version":"provisional:otros-gastos:20260925"}]}',
 (SELECT regla_ref FROM vec_dietas.regla_devengo_provisional LIMIT 1),clock_timestamp());
COMMIT;
SQL
if sql -o /dev/null < "$mig/000009_otros_gastos_justificados.down.sql" 2>/dev/null; then fallo 'DOWN atravesó una línea D5 guardada'; fi
[[ $(huella_fn) == "$nueva_fn" ]] || fallo 'DOWN rechazado dejó cambios'
ok 'DOWN rechazado con línea D5 guardada'
sql -o /dev/null <<'SQL' || fallo 'retirar historia sintética'
BEGIN;
SET LOCAL session_replication_role=replica;
DELETE FROM vec_dietas.comision_revision WHERE comision_ref='dco_HHHHHHHHHHHHHHHHHHHHHH';
DELETE FROM vec_dietas.borrador_comision WHERE referencia='dco_HHHHHHHHHHHHHHHHHHHHHH';
COMMIT;
SQL

sql -o /dev/null < "$mig/000009_otros_gastos_justificados.down.sql" || fallo 'DOWN sin historia'
[[ $(huella_fn) == "$previa_fn" && $(huella_vec) == "$previa_vec" ]] || fallo 'DOWN no restaura la preimagen exacta'
ok 'DOWN restaura la preimagen exacta'
sql -o /dev/null < "$mig/000009_otros_gastos_justificados.up.sql" || fallo 'reinstalación'
[[ $(huella_fn) == "$nueva_fn" ]] || fallo 'reinstalación distinta'
ok 'reinstalación idéntica tras DOWN'
echo 'OK ensayo Dietas 000009 en PostgreSQL 18.4 desechable'
