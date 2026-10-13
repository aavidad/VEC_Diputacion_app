\set ON_ERROR_STOP on
-- CT214: prueba de correspondencia en la entrada ejecutable CT185 v4.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000214',0));
DO $pre$
DECLARE v4 regprocedure; huella_actual text; version_pg integer;
BEGIN
 version_pg:=pg_catalog.current_setting('server_version_num')::integer;
 IF version_pg NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CT214: PARO clave=server_version_num actual=% esperado=180000..189999',version_pg USING ERRCODE='55000';
 END IF;
 IF pg_catalog.to_regclass('vec_contratacion_temporal.firma_correspondencia_organizacion_v1') IS NOT NULL THEN
  RAISE EXCEPTION 'CT214: PARO clave=tabla_correspondencia actual=presente esperado=ausente' USING ERRCODE='55000';
 END IF;
 IF pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)') IS NULL THEN
  RAISE EXCEPTION 'CT214: PARO clave=resolver_CA45 actual=ausente esperado=presente' USING ERRCODE='55000';
 END IF;
 IF pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
       'vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(text)','EXECUTE') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT214: PARO clave=execute_CA45 actual=false esperado=true' USING ERRCODE='55000';
 END IF;
 v4:=pg_catalog.to_regprocedure('vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 huella_actual:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(v4),'UTF8')),'hex');
 IF huella_actual IS DISTINCT FROM '1d97318b9300965ac9d6d3a9ece0a77f3ad41b3a8965a6c8c824477e6682f328' THEN
  RAISE EXCEPTION 'CT214: PARO clave=v4_preimagen actual=% esperado=1d97318b9300965ac9d6d3a9ece0a77f3ad41b3a8965a6c8c824477e6682f328',COALESCE(huella_actual,'ausente') USING ERRCODE='55000';
 END IF;
END $pre$;
CREATE TABLE vec_contratacion_temporal.firma_correspondencia_organizacion_v1(
 firma_ref text PRIMARY KEY REFERENCES vec_contratacion_temporal.firma_documento_v1(firma_ref),
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.firma_documento_v1(recibo_ref),
 organizacion_operativa_ref text NOT NULL,
 organizacion_canonica_ref text NOT NULL,
 correspondencia_ref text NOT NULL,
 version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 9007199254740991),
 prueba jsonb NOT NULL,
 confirmada_en timestamptz(6) NOT NULL,
 CHECK((jsonb_typeof(prueba)='object' AND prueba->>'esquema'='vec.organizacion.correspondencia.v1'
  AND prueba->>'organizacion_operativa_ref'=organizacion_operativa_ref
  AND prueba->>'organizacion_canonica_ref'=organizacion_canonica_ref
  AND prueba->>'correspondencia_ref'=correspondencia_ref AND (prueba->>'version')::numeric=version
  AND prueba->>'estado'='vigente' AND prueba->>'huella_sha256' ~ '^[0-9a-f]{64}$') IS TRUE)
);
ALTER TABLE vec_contratacion_temporal.firma_correspondencia_organizacion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.firma_correspondencia_organizacion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_contratacion_temporal.firma_correspondencia_organizacion_v1
 TO vec_contratacion_temporal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_contratacion_temporal.firma_correspondencia_organizacion_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contratacion_temporal.firma_correspondencia_organizacion_v1 FROM PUBLIC;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_contratacion_temporal.firma_correspondencia_organizacion_v1 FOR EACH STATEMENT
 EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
-- Cada marca se exige una sola vez en la definición literal instalada de CT185.
DO $v4$
DECLARE t text; viejo text; nuevo text; v4 regprocedure:=
 'vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 t:=pg_catalog.pg_get_functiondef(v4);
 viejo:=$old_path$SET search_path TO 'pg_catalog'$old_path$;
 nuevo:=$new_path$SET search_path TO 'pg_catalog', 'pg_temp'$new_path$;
 IF pg_catalog.length(viejo)=0 OR (pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / pg_catalog.length(viejo)<>1 THEN
  RAISE EXCEPTION 'CT214: PARO clave=marca_v4 actual=% esperado=1 detalle=%',(pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / NULLIF(pg_catalog.length(viejo),0),pg_catalog.left(viejo,80) USING ERRCODE='55000'; END IF;
 t:=pg_catalog.replace(t,viejo,nuevo);
 viejo:=$old_decl$ cat text; ver bigint; pub text; clave text; decision_h text;$old_decl$;
 nuevo:=$new_decl$ cat text; ver bigint; pub text; clave text; decision_h text;
 correspondencia jsonb; prueba_correspondencia jsonb; prueba_historica record;$new_decl$;
 IF pg_catalog.length(viejo)=0 OR (pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / pg_catalog.length(viejo)<>1 THEN
  RAISE EXCEPTION 'CT214: PARO clave=marca_v4 actual=% esperado=1 detalle=%',(pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / NULLIF(pg_catalog.length(viejo),0),pg_catalog.left(viejo,80) USING ERRCODE='55000'; END IF;
 t:=pg_catalog.replace(t,viejo,nuevo);
 viejo:=$old_call$ recibo:=vec_contratacion_temporal.registrar_firma_verificada_v3($old_call$;
 nuevo:=$new_call$ -- CA45 resuelve bajo bloqueo compartido la misma organización operativa
 -- que permanece en el recurso, la asignación, el original y el pin.
 correspondencia:=vec_contexto_actor_v1.resolver_correspondencia_organizacion_v1(s->>'OrganizacionRef');
 IF jsonb_typeof(correspondencia) IS DISTINCT FROM 'object'
    OR correspondencia->>'organizacion_operativa_ref' IS DISTINCT FROM s->>'OrganizacionRef'
    OR (correspondencia->>'organizacion_canonica_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'CT214 correspondencia no ligada al efecto' USING ERRCODE='42501'; END IF;
 prueba_correspondencia:=NULLIF(correspondencia->'prueba','null'::jsonb);
 IF prueba_correspondencia IS NOT NULL AND (
    jsonb_typeof(prueba_correspondencia) IS DISTINCT FROM 'object'
    OR prueba_correspondencia->>'esquema' IS DISTINCT FROM 'vec.organizacion.correspondencia.v1'
    OR prueba_correspondencia->>'estado' IS DISTINCT FROM 'vigente'
    OR prueba_correspondencia->>'organizacion_operativa_ref' IS DISTINCT FROM s->>'OrganizacionRef'
    OR prueba_correspondencia->>'organizacion_canonica_ref' IS DISTINCT FROM correspondencia->>'organizacion_canonica_ref'
    OR (prueba_correspondencia->>'correspondencia_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
    OR (prueba_correspondencia->>'version' ~ '^[1-9][0-9]{0,15}$') IS NOT TRUE
    OR (prueba_correspondencia->>'version')::numeric>9007199254740991
    OR (prueba_correspondencia->>'huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE) THEN
  RAISE EXCEPTION 'CT214 prueba de correspondencia inválida' USING ERRCODE='42501'; END IF;
 recibo:=vec_contratacion_temporal.registrar_firma_verificada_v3($new_call$;
 IF pg_catalog.length(viejo)=0 OR (pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / pg_catalog.length(viejo)<>1 THEN
  RAISE EXCEPTION 'CT214: PARO clave=marca_v4 actual=% esperado=1 detalle=%',(pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / NULLIF(pg_catalog.length(viejo),0),pg_catalog.left(viejo,80) USING ERRCODE='55000'; END IF;
 t:=pg_catalog.replace(t,viejo,nuevo);
 viejo:=$old_end$ -- CC7 acaba de bloquear el control.$old_end$;
 nuevo:=$new_end$ -- La prueba se escribe sólo con una firma nueva y en la misma transacción.
 -- Una firma histórica sin prueba se recupera sin fabricar evidencia retroactiva.
 IF recibo->'YaRegistrada'='false'::jsonb THEN
  IF prueba_correspondencia IS NOT NULL THEN
   INSERT INTO vec_contratacion_temporal.firma_correspondencia_organizacion_v1(
    firma_ref,recibo_ref,organizacion_operativa_ref,organizacion_canonica_ref,
    correspondencia_ref,version,prueba,confirmada_en)
   VALUES(original.firma_ref,original.recibo_ref,s->>'OrganizacionRef',
    correspondencia->>'organizacion_canonica_ref',prueba_correspondencia->>'correspondencia_ref',
    (prueba_correspondencia->>'version')::numeric,prueba_correspondencia,original.registrada_en);
  END IF;
 ELSE
  SELECT * INTO prueba_historica
   FROM vec_contratacion_temporal.firma_correspondencia_organizacion_v1
   WHERE firma_ref=original.firma_ref FOR KEY SHARE;
  IF FOUND AND (
     prueba_correspondencia IS NULL
     OR prueba_historica.recibo_ref IS DISTINCT FROM original.recibo_ref
     OR prueba_historica.organizacion_operativa_ref IS DISTINCT FROM original.organizacion_ref
     OR prueba_historica.organizacion_canonica_ref IS DISTINCT FROM correspondencia->>'organizacion_canonica_ref'
     OR prueba_historica.correspondencia_ref IS DISTINCT FROM prueba_correspondencia->>'correspondencia_ref'
     OR prueba_historica.version IS DISTINCT FROM (prueba_correspondencia->>'version')::numeric
     OR prueba_historica.prueba IS DISTINCT FROM prueba_correspondencia
     OR prueba_historica.confirmada_en IS DISTINCT FROM original.registrada_en) THEN
   RAISE EXCEPTION 'CT214 prueba histórica divergente' USING ERRCODE='42501'; END IF;
 END IF;
 -- CC7 acaba de bloquear el control.$new_end$;
 IF pg_catalog.length(viejo)=0 OR (pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / pg_catalog.length(viejo)<>1 THEN
  RAISE EXCEPTION 'CT214: PARO clave=marca_v4 actual=% esperado=1 detalle=%',(pg_catalog.length(t)-pg_catalog.length(pg_catalog.replace(t,viejo,''))) / NULLIF(pg_catalog.length(viejo),0),pg_catalog.left(viejo,80) USING ERRCODE='55000'; END IF;
 t:=pg_catalog.replace(t,viejo,nuevo);
 EXECUTE t;
END $v4$;
-- CREATE OR REPLACE conserva propietario y ACL de CT185. La comprobación
-- posterior impide que la ruta ejecutable derive hacia otra firma o consumidor.
DO $post$
DECLARE v4 regprocedure:=
 'vec_contratacion_temporal.registrar_firma_con_plan_v4(text,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 def text;
BEGIN
 def:=pg_catalog.pg_get_functiondef(v4);
 IF (SELECT pg_catalog.pg_get_userbyid(proowner) FROM pg_catalog.pg_proc WHERE oid=v4)
    IS DISTINCT FROM 'vec_contratacion_temporal_propietario'
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=v4) IS DISTINCT FROM true
 OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=v4) IS DISTINCT FROM
    ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=v4) IS DISTINCT FROM
    ARRAY['vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
          'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
 OR pg_catalog.strpos(def,'consumir_plan_firma_ct_v3_atestada(')=0
 OR pg_catalog.strpos(def,'registrar_firma_verificada_v3(')=0
 OR pg_catalog.strpos(def,'firma_correspondencia_organizacion_v1(')=0
 OR pg_catalog.strpos(def,'IF recibo->''YaRegistrada''=''false''::jsonb THEN')=0 THEN
  RAISE EXCEPTION 'CT214: PARO clave=v4_postimagen_conforme actual=false esperado=true' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
