#!/usr/bin/env bash
# Ensayo focal de Bolsa 000046 en PostgreSQL 18 desechable. El verificador CT
# se sustituye por una preimagen de firma idéntica; la integración CT real se
# ensaya separadamente al componer los dos módulos.
set -Eeuo pipefail
repo=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm}
contenedor=vec-pg-reincorporacion-$$
datos=/dev/shm/vec-pg-reincorporacion-$$
trabajo=$(mktemp -d /dev/shm/vec-reincorporacion-XXXXXX)
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d "$datos" ]]; then docker run --rm -v "$datos":/d --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true; fi
  rm -rf "$datos" "$trabajo"
}
trap limpiar EXIT
mkdir -p "$datos"
clave=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
docker run --detach --rm --name "$contenedor" -v "$datos":/var/lib/postgresql -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
for _ in $(seq 1 60); do docker exec "$contenedor" pg_isready -U postgres >/dev/null 2>&1 && break; sleep 1; done
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
base=$repo/deploy/postgresql/bolsa_llamamientos
psql_pg < "$base/pruebas_sql/reincorporacion_titular/preimagen.sql" >/dev/null
m=$base/migraciones/000046_reincorporacion_titular_ct
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
[[ $(psql_pg -tAc "SELECT to_regclass('vec_bolsa_llamamientos.reincorporacion_titular_ct') IS NULL") == t ]] || { echo 'ROLLBACK dejó objetos'; exit 1; }
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'doble UP aceptado'; exit 1; fi
psql_pg < "$m.down.sql" >/dev/null
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'doble DOWN aceptado'; exit 1; fi
psql_pg < "$m.up.sql" >/dev/null
echo 'ROLLBACK, UP/DOWN/UP y doble aplicación: OK'

psql_pg <<'SQL'
DO $test$
DECLARE f text;
BEGIN
 IF NOT (SELECT relrowsecurity AND relforcerowsecurity FROM pg_class
          WHERE oid='vec_bolsa_llamamientos.reincorporacion_titular_ct'::regclass) THEN
  RAISE EXCEPTION 'RLS FORCE ausente';
 END IF;
 IF has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.reincorporacion_titular_ct','SELECT,INSERT,UPDATE,DELETE') THEN
  RAISE EXCEPTION 'ejecutor lee tabla';
 END IF;
 FOREACH f IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(text,text,bigint)',
  'vec_bolsa_llamamientos.cursor_reincorporaciones_titular_ct_v1()',
  'vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  IF NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_proc p, aclexplode(p.proacl) a WHERE p.oid=f::regprocedure AND a.grantee=0) THEN
   RAISE EXCEPTION 'ACL incorrecta: %',f;
  END IF;
 END LOOP;
END $test$;
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $test$
DECLARE v record; n integer;
BEGIN
 SELECT * INTO v FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
  'evento:ct:reincorporacion:uno',repeat('1',64),10);
 IF v.reutilizada OR v.estado<>'cese_aplicado' OR v.disponible_desde<>DATE '2027-02-01' THEN
  RAISE EXCEPTION 'retorno acreditado no enlazado al cese';
 END IF;
 SELECT * INTO v FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
  'evento:ct:reincorporacion:uno',repeat('1',64),10);
 IF NOT v.reutilizada OR v.estado<>'cese_aplicado' THEN RAISE EXCEPTION 'replay divergente'; END IF;
 SELECT * INTO v FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
  'evento:ct:reincorporacion:dos',repeat('2',64),11);
 IF v.estado<>'pendiente_cese' OR v.disponible_desde IS NOT NULL THEN RAISE EXCEPTION 'cese ausente aplicado'; END IF;
 SELECT * INTO v FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
  'evento:ct:reincorporacion:tres',repeat('3',64),12);
 IF v.estado<>'cese_incompatible' OR v.disponible_desde IS NOT NULL THEN RAISE EXCEPTION 'cese ajeno aplicado'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1('evento:ct:reincorporacion:uno',repeat('4',64),10);
  RAISE EXCEPTION 'huella forjada aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.cursor_reincorporaciones_titular_ct_v1() c
 WHERE c.origen_posicion=12;
 IF n<>1 THEN RAISE EXCEPTION 'cursor incorrecto'; END IF;
END $test$;
DO $test$
DECLARE d bytea; n integer; rechazado boolean := false;
BEGIN
 d:=convert_to(jsonb_build_object('principal_id','actor:rrhh','accion','bolsa.situacion_participacion.cambiar',
  'modulo_id','bolsa','tipo_recurso','participacion_bolsa','finalidad','gestion_situacion_participacion',
  'recurso_ref','participacion:uno','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'contexto_recurso_huella_sha256',repeat('a',64))::text,'UTF8');
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(
  'participacion:uno','actor:rrhh','\x00'::bytea,d,'\x00'::bytea,'\x00'::bytea,1,1,
  convert_to('participacion:uno','UTF8'),convert_to(repeat('a',64),'UTF8'),'\x00'::bytea,'\x00'::bytea) r
 WHERE r.evento_ref='evento:ct:reincorporacion:uno' AND r.estado='cese_aplicado'
   AND r.disponible_desde=DATE '2027-02-01' AND r.regla_version=1 AND r.regla_huella_sha256=repeat('a',64);
 IF n<>1 THEN RAISE EXCEPTION 'ficha autorizada sin retorno y regla'; END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(
  'participacion:uno','actor:rrhh','\x00'::bytea,d,'\x00'::bytea,'\x00'::bytea,1,1,
  convert_to('participacion:uno','UTF8'),convert_to(repeat('a',64),'UTF8'),'\x00'::bytea,'\x00'::bytea);
 IF n<>1 THEN RAISE EXCEPTION 'ficha expuso retorno con cese incompatible'; END IF;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(
   'participacion:uno','actor:ajeno','\x00'::bytea,d,'\x00'::bytea,'\x00'::bytea,1,1,
   convert_to('participacion:uno','UTF8'),convert_to(repeat('a',64),'UTF8'),'\x00'::bytea,'\x00'::bytea);
 EXCEPTION WHEN insufficient_privilege THEN rechazado:=true; END;
 IF NOT rechazado THEN RAISE EXCEPTION 'lectura ajena aceptada'; END IF;
END $test$;
RESET ROLE;
SET ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.restriccion_cese_bolsa VALUES
 ('evento:ct:contrato-bolsa:'||repeat('2',64),'evento:ct:cese:dos','can_1234567890123456789012','relacion:dos',
  'recibo:ct:cese:dos',DATE '2026-09-02',DATE '2027-02-02',1);
RESET ROLE;
SET ROLE vec_bolsa_llamamientos_ejecutor;
DO $test$
DECLARE v record;
BEGIN
 SELECT * INTO v FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1(
  'evento:ct:reincorporacion:dos',repeat('2',64),11);
 IF NOT v.reutilizada OR v.estado<>'cese_aplicado' OR v.disponible_desde<>DATE '2027-02-02' THEN
  RAISE EXCEPTION 'cese tardío no reconciliado';
 END IF;
END $test$;
RESET ROLE;
DO $test$
DECLARE n integer;
BEGIN
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.reincorporacion_titular_ct;
 IF n<>3 THEN RAISE EXCEPTION 'historia duplicada'; END IF;
END $test$;
SQL
echo 'Origen, replay, cese tardío/incompatible, cursor y ACL: OK'

if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'DOWN borró historia'; exit 1; fi
echo 'DOWN con historia rechazado: OK'
printf 'PG18 Bolsa 000046 focal: OK\n'
