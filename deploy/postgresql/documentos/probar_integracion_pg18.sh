#!/usr/bin/env bash
set -euo pipefail

base_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_dir=$(CDPATH= cd -- "$base_dir/../../.." && pwd)
container="vec-b5-doc-$$-${RANDOM}"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT

docker run -d --rm --name "$container" -e POSTGRES_HOST_AUTH_METHOD=trust postgres:18.4-alpine >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -c 'SELECT 1' >/dev/null 2>&1; then
  sleep 0.3
  if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
 fi
 sleep 0.3
done

ad3="$repo_dir/deploy/postgresql/autorizacion_atestada_v3"
docker cp "$ad3/pruebas_sql/organizacion_historica_ad3_000051_stub.sql" "$container:/tmp/stub.sql"
docker cp "$ad3/migraciones/000051_consumidor_organizacion_historica.up.sql" "$container:/tmp/000051.sql"
docker cp "$ad3/migraciones/000052_consumidor_importacion_organizacion.up.sql" "$container:/tmp/000052.sql"
docker cp "$ad3/migraciones/000060_documentos_comunes.up.sql" "$container:/tmp/000060.sql"
docker cp "$ad3/migraciones/000062_replay_documentos_comunes.up.sql" "$container:/tmp/000062.sql"
docker cp "$base_dir/roles_up.sql" "$container:/tmp/roles.sql"
docker cp "$base_dir/migraciones/000001_documentos_comunes.up.sql" "$container:/tmp/documentos.sql"
docker cp "$base_dir/migraciones/000002_replay_autorizado.up.sql" "$container:/tmp/documentos2.sql"
docker cp "$base_dir/pruebas_sql/replay_ad3_62_sintetico.sql" "$container:/tmp/replay_ad3_62.sql"

psql_pg() { docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres -f "$1"; }
psql_pg /tmp/stub.sql
psql_pg /tmp/000051.sql
psql_pg /tmp/000052.sql
psql_pg /tmp/roles.sql

docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/000060.sql >/tmp/000060.rollback.sql"
psql_pg /tmp/000060.rollback.sql
test "$(docker exec "$container" psql -X -qAt -U postgres -c "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL")" = t
psql_pg /tmp/000060.sql
docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres <<'SQL'
CREATE SCHEMA vec_autorizacion;
CREATE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)
RETURNS timestamptz LANGUAGE sql AS $$ SELECT clock_timestamp() $$;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric) TO vec_autorizacion_atestada_v3_propietario;
SQL
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/000062.sql >/tmp/000062.rollback.sql"
psql_pg /tmp/000062.rollback.sql
test "$(docker exec "$container" psql -X -qAt -U postgres -c "SELECT to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL")" = t
psql_pg /tmp/000062.sql

docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/documentos.sql >/tmp/documentos.rollback.sql"
psql_pg /tmp/documentos.rollback.sql
test "$(docker exec "$container" psql -X -qAt -U postgres -c "SELECT to_regclass('vec_documentos.documento') IS NULL")" = t
psql_pg /tmp/documentos.sql
docker exec "$container" sh -c "sed '\$s/^COMMIT;/ROLLBACK;/' /tmp/documentos2.sql >/tmp/documentos2.rollback.sql"
psql_pg /tmp/documentos2.rollback.sql
test "$(docker exec "$container" psql -X -qAt -U postgres -c "SELECT to_regprocedure('vec_documentos.confirmar_alta_v2(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL")" = t
psql_pg /tmp/documentos2.sql

docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres <<'SQL'
CREATE ROLE vec_documentos_ensayo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_documentos_ejecutor TO vec_documentos_ensayo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
DO $checks$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['documento','preparacion_notificacion','auditoria_operacion','outbox'] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=('vec_documentos.'||t)::regclass AND relrowsecurity AND relforcerowsecurity)
     OR has_table_privilege('vec_documentos_ensayo','vec_documentos.'||t,'SELECT')
     OR has_table_privilege('vec_documentos_ensayo','vec_documentos.'||t,'INSERT')
  THEN RAISE EXCEPTION 'RLS o ACL abiertos: %',t; END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid='vec_documentos.confirmar_alta_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure AND a.grantee=0)
    OR has_function_privilege('vec_documentos_ensayo','vec_documentos.consumir_v3_v1(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_documentos_ensayo','vec_documentos.confirmar_alta_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'ACL documental incompatible'; END IF;
 IF EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname IN ('vec_documentos','vec_autorizacion_atestada_v3') AND p.proname IN
  ('confirmar_alta_v1','listar_expediente_v1','obtener_original_v1','preparar_notificacion_v1','consumir_operacion_documentos_v3_atestada')
  AND (NOT p.prosecdef OR NOT ('search_path=pg_catalog'=ANY(p.proconfig))))
 THEN RAISE EXCEPTION 'función sensible sin SECURITY DEFINER/search_path fijo'; END IF;
END $checks$;
SQL
psql_pg /tmp/replay_ad3_62.sql

# Solo para probar la lógica documental: sustituir en ESTA base desechable la
# fachada AD3-62 por un recibo sintético idempotente. No es una prueba COSE.
docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U postgres <<'SQL'
CREATE TABLE public.ensayo_documentos_b5(preimagen bytea NOT NULL,capacidad bytea NOT NULL,
 decision bytea NOT NULL,auth jsonb NOT NULL,objeto jsonb NOT NULL,recibo jsonb);
GRANT SELECT,INSERT,UPDATE ON public.ensayo_documentos_b5 TO vec_documentos_ensayo;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.ensayo_consumo_documentos(
 decision_ref text PRIMARY KEY, capacidad bytea NOT NULL, decision bytea NOT NULL,
 efecto_ref text NOT NULL, huella_efecto_sha256 text NOT NULL);
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
 previo record; nuevo boolean:=false;
BEGIN
 SELECT * INTO previo FROM vec_autorizacion_atestada_v3.ensayo_consumo_documentos ec
  WHERE ec.decision_ref=d->>'decision_ref' FOR UPDATE;
 IF FOUND THEN
  IF previo.capacidad IS DISTINCT FROM p_capacidad OR previo.decision IS DISTINCT FROM p_decision
     OR previo.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
     OR previo.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
  THEN RAISE EXCEPTION 'conflicto sintético AD3' USING ERRCODE='23505'; END IF;
 ELSE
  INSERT INTO vec_autorizacion_atestada_v3.ensayo_consumo_documentos VALUES
   (d->>'decision_ref',p_capacidad,p_decision,c->>'efecto_ref',c->>'huella_efecto_sha256');
  nuevo:=true;
 END IF;
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('b',64),'audit:synthetic',clock_timestamp(),nuevo;
END $f$;
RESET ROLE;
SQL

docker exec -i "$container" psql -X -q -v ON_ERROR_STOP=1 -U vec_documentos_ensayo -d postgres <<'SQL'
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $test$
DECLARE p bytea; h text; c bytea; d bytea; a jsonb; o jsonb; r jsonb;
BEGIN
 p:=convert_to('{"accion":"documentos.generado.alta","id":"doc:00000000-0000-4000-8000-000000000001","clave_idempotencia":"idem:00000000-0000-4000-8000-000000000001","modulo_id":"dietas","expediente_ref":"exp:00000000-0000-4000-8000-000000000001","tipo_ref":"tipo:00000000-0000-4000-8000-000000000001","version":1,"mime":"application/pdf","tamano":3,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","politica_ref":"pol:00000000-0000-4000-8000-000000000001","version_politica":1,"huella_politica_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","proteccion":"conservacion","conservacion_hasta":"2030-01-01T00:00:00Z"}','UTF8');
 h:=encode(sha256(p),'hex');
 c:=convert_to(jsonb_build_object('audiencia_consumo','vec_documentos.operacion.v1','operacion','documentos.generado.alta','efecto_ref','doc:00000000-0000-4000-8000-000000000001','huella_efecto_sha256',h)::text,'UTF8');
 d:=convert_to(jsonb_build_object('accion','documentos.generado.alta','modulo_id','documentos','tipo_recurso','documento_generado','finalidad','alta_documento_generado','campos_permitidos',jsonb_build_array('documento','recibo'),'obligaciones','[]'::jsonb,'recurso_ref','doc:00000000-0000-4000-8000-000000000001','contexto_recurso_huella_sha256',h,'principal_id','per:00000000-0000-4000-8000-000000000001','perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001','correlacion_ref','corr:00000000-0000-4000-8000-000000000001','decision_ref','decision:00000000-0000-4000-8000-000000000001')::text,'UTF8');
 a:=jsonb_build_object('accion','documentos.generado.alta','finalidad','alta_documento_generado','recurso_ref','doc:00000000-0000-4000-8000-000000000001','ambito_ref','exp:00000000-0000-4000-8000-000000000001','principal_id','per:00000000-0000-4000-8000-000000000001','perfil_activo_ref','perfil:00000000-0000-4000-8000-000000000001','correlacion_ref','corr:00000000-0000-4000-8000-000000000001');
 o:=jsonb_build_object('objeto_ref','obj:00000000-0000-4000-8000-000000000001','objeto_version','ov1','conector_ref','s3_ensayo','recibo_objeto_ref','recibo:00000000-0000-4000-8000-000000000001','recibo_objeto_huella_sha256',repeat('d',64),'retenido_hasta','2031-01-01T00:00:00Z','inmovilizado',false,'mime','application/pdf','tamano',3,'huella_sha256',repeat('a',64));
 INSERT INTO public.ensayo_documentos_b5(preimagen,capacidad,decision,auth,objeto) VALUES(p,c,d,a,o);
 SELECT vec_documentos.confirmar_alta_v2(p,o,a,c,d,'\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea) INTO r;
 IF r->>'estado_firma'<>'pendiente_proveedor' OR r->>'numero_vec' !~ '^VEC-[0-9]{4}-[0-9]+$' THEN RAISE EXCEPTION 'alta sintética documental incoherente'; END IF;
 UPDATE public.ensayo_documentos_b5 SET recibo=r;
 o:=jsonb_set(o,'{retenido_hasta}','"2029-01-01T00:00:00Z"'::jsonb);
 BEGIN
  PERFORM vec_documentos.confirmar_alta_v2(p,o,a,c,d,'\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'retención insuficiente aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $test$;
COMMIT;
SQL
test "$(docker exec "$container" psql -X -qAt -U postgres -c "SELECT count(*)=1 FROM vec_documentos.documento WHERE estado_firma='pendiente_proveedor' AND objeto_retenido_hasta>=conservacion_hasta")" = t

docker exec -i "$container" sh -c 'cat >/tmp/replay_documentos.sql' <<'SQL'
\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
DO $replay$
DECLARE f record; r jsonb; otra_clave bytea;
BEGIN
 SELECT * INTO STRICT f FROM public.ensayo_documentos_b5;
 SELECT vec_documentos.confirmar_alta_v2(f.preimagen,f.objeto,f.auth,f.capacidad,f.decision,
  '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea) INTO r;
 IF r IS DISTINCT FROM f.recibo THEN RAISE EXCEPTION 'replay cambió recibo'; END IF;
 otra_clave:=convert_to(replace(convert_from(f.preimagen,'UTF8'),
  'idem:00000000-0000-4000-8000-000000000001','idem:00000000-0000-4000-8000-000000000002'),'UTF8');
 BEGIN
  PERFORM vec_documentos.confirmar_alta_v2(otra_clave,f.objeto,f.auth,f.capacidad,f.decision,
   '\x01'::bytea,'\x01'::bytea,1,1,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea,'\x01'::bytea);
  RAISE EXCEPTION 'otra clave con mismo V3 aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $replay$;
COMMIT;
SQL
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U vec_documentos_ensayo -d postgres -f /tmp/replay_documentos.sql
test "$(docker exec "$container" psql -X -qAt -U postgres -c "SELECT (SELECT count(*) FROM vec_documentos.documento)=1 AND (SELECT count(*) FROM vec_documentos.outbox)=1 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.ensayo_consumo_documentos)=1")" = t

docker restart "$container" >/dev/null
for _ in $(seq 1 60); do
 if docker exec "$container" psql -X -qAt -v ON_ERROR_STOP=1 -U postgres -c 'SELECT 1' >/dev/null 2>&1; then break; fi
 sleep 0.3
done
docker exec "$container" psql -X -q -v ON_ERROR_STOP=1 -U vec_documentos_ensayo -d postgres -f /tmp/replay_documentos.sql
test "$(docker exec "$container" psql -X -qAt -U postgres -c "SELECT to_regclass('vec_documentos.documento') IS NOT NULL AND (SELECT count(*) FROM vec_documentos.documento)=1 AND (SELECT count(*) FROM vec_documentos.outbox)=1 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.ensayo_consumo_documentos)=1")" = t
printf 'PG18.4: AD3-60/62 y Documentos-1/2 ROLLBACK/COMMIT, ACL, RLS, replay exacto y reinicio compatibles sobre preimagen sintética; NO acredita cadena V3 real.\n'
