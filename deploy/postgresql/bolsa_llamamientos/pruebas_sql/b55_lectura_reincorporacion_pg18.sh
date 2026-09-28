#!/usr/bin/env bash
# B55 en PG18 sintético: B46 real y doble mínimo de AD3-101 (el núcleo AD3 se revisa aparte).
set -Eeuo pipefail
repo=$(git rev-parse --show-toplevel)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-alpine}
contenedor=vec-b55-$$
datos=/dev/shm/$contenedor
limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if [[ -d $datos ]]; then
    docker run --rm --pull never -v "$datos:/d" --entrypoint rm "$imagen" -rf /d/18 >/dev/null 2>&1 || true
    rmdir "$datos" 2>/dev/null || true
  fi
}
trap limpiar EXIT
mkdir -p "$datos"
docker run -d --rm --pull never --network none --name "$contenedor" \
  -v "$datos:/var/lib/postgresql" -e POSTGRES_HOST_AUTH_METHOD=trust "$imagen" >/dev/null
for _ in $(seq 1 80); do docker exec "$contenedor" pg_isready -q -U postgres >/dev/null 2>&1 && break; sleep 0.5; done
sleep 2 # el servidor temporal de initdb puede responder antes del arranque definitivo
psql_pg() { docker exec -i "$contenedor" psql -X -q -v ON_ERROR_STOP=1 -U postgres "$@"; }
base=$repo/deploy/postgresql/bolsa_llamamientos
psql_pg < "$base/pruebas_sql/reincorporacion_titular/preimagen.sql" >/dev/null
psql_pg < "$base/migraciones/000046_reincorporacion_titular_ct.up.sql" >/dev/null
psql_pg <<'SQL' >/dev/null
CREATE ROLE vec_bolsa_llamamientos_migrador NOLOGIN;
CREATE ROLE vec_b55_rrhh LOGIN INHERIT;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b55_rrhh;
CREATE TABLE prueba_reincorporacion.revocacion(revocada boolean NOT NULL);
INSERT INTO prueba_reincorporacion.revocacion VALUES(false);
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 IF (SELECT revocada FROM prueba_reincorporacion.revocacion)
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.reincorporacion_titular.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'material no autorizado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:'||gen_random_uuid()::text,c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('b',64),
  'auditoria:'||gen_random_uuid()::text,clock_timestamp(),true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_reincorporacion_titular_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
SQL
m=$base/migraciones/000055_lectura_reincorporacion_titular_v3
sed 's/^COMMIT;$/ROLLBACK;/' "$m.up.sql" | psql_pg >/dev/null
[[ $(psql_pg -Atc "SELECT to_regprocedure('vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL") == t ]]
psql_pg < "$m.up.sql" >/dev/null
if psql_pg < "$m.up.sql" >/dev/null 2>&1; then echo 'B55 doble UP aceptado' >&2; exit 1; fi
psql_pg < "$m.down.sql" >/dev/null
psql_pg < "$m.up.sql" >/dev/null
[[ $(psql_pg -Atc "SELECT c.relrowsecurity::text||':'||c.relforcerowsecurity::text FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3'::regclass") == 'true:true' || $(psql_pg -Atc "SELECT c.relrowsecurity::text||':'||c.relforcerowsecurity::text FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3'::regclass") == 't:t' ]]
[[ $(psql_pg -Atc "SELECT count(*) FROM pg_policies WHERE schemaname='vec_bolsa_llamamientos' AND tablename='reincorporacion_titular_lectura_v3' AND policyname='reincorporacion_titular_lectura_v3_solo_propietario' AND roles='{vec_bolsa_llamamientos_propietario}'") == 1 ]]
# Una inserción confirmada después de la sonda de DOWN jamás debe perderse.
# La transacción escritora mantiene RowExclusive mientras DOWN espera el bloqueo.
psql_pg <<'SQL' >/dev/null &
BEGIN;
INSERT INTO vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3
 (decision_ref,auditoria_ref,participacion_ref,actor_ref,filas_devueltas,consultada_en)
 VALUES ('decision:carrera','auditoria:carrera','participacion:uno','per_abcdefghijklmnopqrstuv',0,clock_timestamp());
SELECT pg_sleep(4);
COMMIT;
SQL
pid_escritura=$!
en_barrera=false
for _ in $(seq 1 80); do
  if [[ $(psql_pg -Atc "SELECT count(*) FROM pg_stat_activity WHERE query='SELECT pg_sleep(4);' AND state='active'") == 1 ]]; then en_barrera=true; break; fi
  sleep 0.05
done
[[ $en_barrera == true ]] || { echo 'B55: no alcanzó barrera de escritura concurrente' >&2; exit 1; }
psql_pg < "$m.down.sql" >/dev/null 2>&1 &
pid_down=$!
wait "$pid_escritura"
if wait "$pid_down"; then echo 'B55 DOWN borró historia concurrente' >&2; exit 1; fi
[[ $(psql_pg -Atc "SELECT count(*) FROM vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3") == 1 ]]
psql_pg <<'SQL' >/dev/null
DO $acl$ BEGIN
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR has_table_privilege('vec_bolsa_llamamientos_ejecutor','vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3','SELECT')
 THEN RAISE EXCEPTION 'B55 ACL incorrecta'; END IF;
END $acl$;
SET ROLE vec_bolsa_llamamientos_ejecutor;
SELECT * FROM vec_bolsa_llamamientos.registrar_reincorporacion_titular_ct_v1('evento:ct:reincorporacion:uno',repeat('1',64),10);
RESET ROLE;
SET SESSION AUTHORIZATION vec_b55_rrhh;
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $test$
DECLARE cap bytea; dec bytea; n int; denegada boolean; actor text:='per_abcdefghijklmnopqrstuv';
BEGIN
 cap:=convert_to('{"operacion":"bolsa.reincorporacion_titular.consultar","audiencia_consumo":"vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1","efecto_ref":"participacion:uno","huella_efecto_sha256":"'||repeat('a',64)||'"}','UTF8');
 dec:=convert_to('{"principal_id":"'||actor||'","accion":"bolsa.reincorporacion_titular.consultar","modulo_id":"bolsa","tipo_recurso":"participacion_bolsa","finalidad":"consulta_reincorporacion_titular","recurso_ref":"participacion:uno","contexto_recurso_huella_sha256":"'||repeat('a',64)||'","campos_permitidos":["reincorporaciones_titular"],"obligaciones":[]}','UTF8');
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2('participacion:uno',actor,cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF n<>1 THEN RAISE EXCEPTION 'B55: ficha exacta no devuelta'; END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2('participacion:dos',actor,replace(convert_from(cap,'UTF8'),'participacion:uno','participacion:dos')::bytea,replace(convert_from(dec,'UTF8'),'participacion:uno','participacion:dos')::bytea,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 IF n<>0 THEN RAISE EXCEPTION 'B55: participación ajena expuesta'; END IF;
 denegada:=false;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2('participacion:uno',actor,cap,
   replace(convert_from(dec,'UTF8'),'bolsa.reincorporacion_titular.consultar','bolsa.situacion_participacion.cambiar')::bytea,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'B55: permiso de mutación aceptado'; END IF;
 denegada:=false;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2('participacion:uno',actor,cap,
   replace(convert_from(dec,'UTF8'),'["reincorporaciones_titular"]','[]')::bytea,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'B55: campos vacíos aceptados'; END IF;
 denegada:=false;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2('participacion:uno','per_zzzzzzzzzzzzzzzzzzzzzz',cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'B55: actor ajeno aceptado'; END IF;
END $test$;
COMMIT;
RESET SESSION AUTHORIZATION;
DO $hist$ BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3)<>3 THEN RAISE EXCEPTION 'B55: lecturas no conservadas'; END IF;
END $hist$;
UPDATE prueba_reincorporacion.revocacion SET revocada=true;
SET SESSION AUTHORIZATION vec_b55_rrhh;
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $test$
DECLARE cap bytea; dec bytea; denegada boolean:=false; actor text:='per_abcdefghijklmnopqrstuv';
BEGIN
 cap:=convert_to('{"operacion":"bolsa.reincorporacion_titular.consultar","audiencia_consumo":"vec_bolsa_llamamientos.reincorporacion_titular.consultar.v1","efecto_ref":"participacion:uno","huella_efecto_sha256":"'||repeat('a',64)||'"}','UTF8');
 dec:=convert_to('{"principal_id":"'||actor||'","accion":"bolsa.reincorporacion_titular.consultar","modulo_id":"bolsa","tipo_recurso":"participacion_bolsa","finalidad":"consulta_reincorporacion_titular","recurso_ref":"participacion:uno","contexto_recurso_huella_sha256":"'||repeat('a',64)||'","campos_permitidos":["reincorporaciones_titular"],"obligaciones":[]}','UTF8');
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2('participacion:uno',actor,cap,dec,'\x00','\x00',1,1,'\x00','\x00','\x00','\x00');
 EXCEPTION WHEN insufficient_privilege THEN denegada:=true; END;
 IF NOT denegada THEN RAISE EXCEPTION 'B55: revocación aceptada'; END IF;
END $test$;
COMMIT;
RESET SESSION AUTHORIZATION;
DO $hist$ BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.reincorporacion_titular_lectura_v3)<>3 THEN RAISE EXCEPTION 'B55: denegación añadió historia'; END IF;
END $hist$;
SQL
if psql_pg < "$m.down.sql" >/dev/null 2>&1; then echo 'B55 DOWN borró historia' >&2; exit 1; fi
echo 'B55 PG18: ROLLBACK, UP, doble UP, DOWN/UP sin historia, lectura exacta, permisos y revocación OK'
