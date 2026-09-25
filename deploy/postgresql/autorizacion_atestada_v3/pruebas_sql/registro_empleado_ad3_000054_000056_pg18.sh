#!/usr/bin/env bash
# Ensayo estructural PG18.4 desechable de AD3-54 (registro de empleado B2),
# AD3-55 (catálogos B2) y AD3-56 (lista de empleados del organismo) sobre la postimagen de main: núcleo 051/052 con stub
# sintético y AD3-53, 59, 61, 70 y 80 reales, en el orden en que la principal
# ya las tiene. Comprueba ROLLBACK sin rastro, COMMIT, ACL nominal, que las
# reescrituras previas del núcleo se conservan, rechazo de segunda aplicación
# y de operación no nominal, y persistencia tras reinicio. No acredita una
# decisión V3 firmada ni la cadena íntegra.
set -euo pipefail
repo_dir=$(CDPATH='' cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
motor=${VEC_CONTENEDOR_MOTOR:-docker}
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor="vec-b2-ad3-54-56-${BASHPID}"
ad3="$repo_dir/deploy/postgresql/autorizacion_atestada_v3"
mig="$ad3/migraciones"
limpiar() { "$motor" rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT
psql_admin() { "$motor" exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres -d postgres -o /dev/null; }
valor() { "$motor" exec "$contenedor" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -d postgres -c "$1"; }
archivo() { psql_admin < "$1"; }
fallo() { printf 'AD3-54/56: %s\n' "$*" >&2; exit 1; }
ok() { printf 'OK %s\n' "$*"; }
esperar() {
  for _ in {1..80}; do
    if valor 'SELECT 1' >/dev/null 2>&1; then return 0; fi
    sleep 0.25
  done
  fallo 'PostgreSQL no quedó disponible'
}
"$motor" run -d --rm --network none --name "$contenedor" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
esperar
[[ $(valor "SELECT current_setting('server_version_num')") == 180004 ]] || fallo 'se requiere PostgreSQL 18.4'

archivo "$ad3/pruebas_sql/organizacion_historica_ad3_000051_stub.sql"
psql_admin <<'SQL'
CREATE ROLE vec_cronos_v1_propietario NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_cronos_v1_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_dietas_propietario NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_dietas_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_personal_d7_ejecutor NOLOGIN NOBYPASSRLS;
CREATE ROLE vec_b2_prueba_ajeno NOLOGIN NOBYPASSRLS;
SQL
for numero in 000051 000052 000053 000059 000061 000070 000080; do
  ruta=$(find "$mig" -maxdepth 1 -name "${numero}_*.up.sql" | sort)
  [[ -n $ruta && $(printf '%s\n' "$ruta" | wc -l) -eq 1 ]] || fallo "falta una única AD3 $numero"
  archivo "$ruta"
done
ok 'postimagen main: AD3-51/52 (stub), 53, 59, 61, 70 y 80 instaladas'

nucleo='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
f54='vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
f55='vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
m54="$mig/000054_consumidor_registro_empleado_b2.up.sql"
m55="$mig/000055_consumidor_catalogos_registro_empleado.up.sql"
f56='vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
m56="$mig/000056_consumidor_empleados_registro_b2.up.sql"
huella() { valor "SELECT md5(pg_get_functiondef('$nucleo'::regprocedure))||':'||(SELECT md5(pg_get_constraintdef(oid)) FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check')"; }

if archivo "$m55" 2>/dev/null; then fallo 'AD3-55 aceptada sin AD3-54'; fi
ok 'AD3-55 rechazada sin AD3-54'
antes=$(huella)
sed '$s/^COMMIT;$/ROLLBACK;/' "$m54" | psql_admin
[[ $(huella) == "$antes" && $(valor "SELECT to_regprocedure('$f54') IS NULL") == t ]] || fallo 'ROLLBACK de AD3-54 dejó rastro'
ok 'ROLLBACK de AD3-54 sin rastro en núcleo, audiencias ni fachada'
archivo "$m54"
antes=$(huella)
sed '$s/^COMMIT;$/ROLLBACK;/' "$m55" | psql_admin
[[ $(huella) == "$antes" && $(valor "SELECT to_regprocedure('$f55') IS NULL") == t ]] || fallo 'ROLLBACK de AD3-55 dejó rastro'
ok 'ROLLBACK de AD3-55 sin rastro'
if archivo "$m56" 2>/dev/null; then fallo 'AD3-56 aceptada sin AD3-55'; fi
ok 'AD3-56 rechazada sin AD3-55'
archivo "$m55"
antes=$(huella)
sed '$s/^COMMIT;$/ROLLBACK;/' "$m56" | psql_admin
[[ $(huella) == "$antes" && $(valor "SELECT to_regprocedure('$f56') IS NULL") == t ]] || fallo 'ROLLBACK de AD3-56 dejó rastro'
ok 'ROLLBACK de AD3-56 sin rastro'
archivo "$m56"
ok 'COMMIT de AD3-54, AD3-55 y AD3-56'

# Las reescrituras previas del núcleo se conservan y la nueva aparece una vez.
for perfil in importacion_organizacion_historica_personal cronos_saldo_propio cronos_permiso_solicitar bandeja_dietas \
              competencias_asignacion_dietas_personal revisor_documento_dietas registro_empleado_b2; do
  n=$(valor "SELECT (length(d)-length(replace(d,'IS DISTINCT FROM ''$perfil''','')))/length('IS DISTINCT FROM ''$perfil''') FROM pg_get_functiondef('$nucleo'::regprocedure) d")
  [[ $n == 1 ]] || fallo "exclusión de $perfil aparece $n veces"
done
ok 'exclusiones previas conservadas y registro_empleado_b2 única'
for operacion in alta.registrar hecho.registrar ficha.consultar vacantes.consultar catalogo.publicar catalogo.retirar catalogo.consultar empleados.consultar; do
  [[ $(valor "SELECT strpos(pg_get_functiondef('$nucleo'::regprocedure),'personal.registro_empleado.$operacion')>0") == t ]] || fallo "contrato $operacion ausente"
done
for audiencia in alta.v1 hecho.v1 ficha.v1 vacantes.v1 catalogo.publicar.v1 catalogo.retirar.v1 catalogo.consultar.v1 empleados.v1; do
  [[ $(valor "SELECT strpos(pg_get_constraintdef(oid),'vec_personal.registro_empleado.$audiencia')>0 AND strpos(pg_get_constraintdef(oid),'vec_cronos_v1.permiso_propio.solicitar.v1')>0 FROM pg_constraint WHERE conname='clave_capacidad_version_audiencia_consumo_check'") == t ]] || fallo "audiencia $audiencia"
done
ok 'audiencias B2 añadidas sin perder las previas'
for firma in "$f54" "$f55" "$f56"; do
  [[ $(valor "SELECT has_function_privilege('vec_personal_propietario','$firma','EXECUTE'),has_function_privilege('vec_personal_ejecutor','$firma','EXECUTE'),has_function_privilege('vec_b2_prueba_ajeno','$firma','EXECUTE'),has_function_privilege('vec_dietas_propietario','$firma','EXECUTE')") == 't|f|f|f' ]] || fallo "ACL divergente en $firma"
  [[ $(valor "SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid='$firma'::regprocedure AND a.grantee NOT IN (p.proowner,'vec_personal_propietario'::regrole)") == 0 ]] || fallo "ACL abierta en $firma"
  [[ $(valor "SELECT prosecdef AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole FROM pg_proc WHERE oid='$firma'::regprocedure") == t ]] || fallo "definidora $firma"
done
ok 'ACL nominal: solo vec_personal_propietario ejecuta las tres fachadas'
psql_admin <<'SQL'
DO $neg$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(
   convert_to('{"operacion":"personal.registro_empleado.catalogo.borrar"}','UTF8'),
   convert_to('{"modulo_id":"personal"}','UTF8'),'x'::bytea,'x'::bytea,1,1,
   'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'operación no nominal aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(
   convert_to('{"operacion":"personal.registro_empleado.baja.registrar"}','UTF8'),
   convert_to('{"modulo_id":"personal"}','UTF8'),'x'::bytea,'x'::bytea,1,1,
   'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'operación no nominal aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.consumir_empleados_registro_b2_v3_atestada(
   convert_to('{"operacion":"personal.registro_empleado.ficha.consultar","audiencia_consumo":"vec_personal.registro_empleado.ficha.v1"}','UTF8'),
   convert_to('{"modulo_id":"personal","accion":"personal.registro_empleado.ficha.consultar"}','UTF8'),'x'::bytea,'x'::bytea,1,1,
   'x'::bytea,'x'::bytea,'x'::bytea,'x'::bytea);
  RAISE EXCEPTION 'ficha aceptada por la fachada de empleados';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $neg$;
SQL
ok 'operaciones no nominales denegadas con 42501'
antes=$(huella)
if archivo "$m54" 2>/dev/null; then fallo 'segunda aplicación de AD3-54 aceptada'; fi
if archivo "$m55" 2>/dev/null; then fallo 'segunda aplicación de AD3-55 aceptada'; fi
if archivo "$m56" 2>/dev/null; then fallo 'segunda aplicación de AD3-56 aceptada'; fi
[[ $(huella) == "$antes" ]] || fallo 'segunda aplicación alteró el núcleo'
ok 'segunda aplicación rechazada sin cambios'
"$motor" restart "$contenedor" >/dev/null
esperar
[[ $(huella) == "$antes" && $(valor "SELECT to_regprocedure('$f54') IS NOT NULL AND to_regprocedure('$f55') IS NOT NULL AND to_regprocedure('$f56') IS NOT NULL") == t ]] || fallo 'estado perdido tras reinicio'
ok 'núcleo, audiencias y fachadas persisten tras reinicio'
printf 'PG18.4: AD3-54/55/56 sobre postimagen main (51/52 stub; 53/59/61/70/80 reales): ROLLBACK, COMMIT, ACL, negativos y reinicio; NO acredita decisión V3 firmada.\n'
