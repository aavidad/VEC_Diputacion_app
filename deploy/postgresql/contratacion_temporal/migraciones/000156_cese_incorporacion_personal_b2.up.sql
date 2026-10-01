\set ON_ERROR_STOP on
-- CT156: incorporación/cese B2 hacia el histórico B13 y retorno B45.
-- CT no decide disponibilidad ni lee tablas Bolsa/Personal. Bolsa13 debe
-- consumir la incorporación y el cese antes de Bolsa45. No acredita baja
-- Personal, firma, eficacia administrativa ni incorporación real.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000156',0));
DO $pre$
BEGIN
 IF to_regclass('vec_contratacion_temporal.origen_incorporacion_personal_b2_v1') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.leer_ceses_bolsa_v1(bigint,text,integer)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.origen_publicacion_bolsa_ct156(text,text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT156: CT155/CT129 requeridas o ya instalada' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2 i
 JOIN vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 b
 ON b.organizacion_ref=i.organizacion_ref AND b.expediente_ref=i.expediente_ref)
 THEN RAISE EXCEPTION 'CT156: historia con orígenes incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;
-- Rellenar SOLO con la preimagen final posterior a CT155. Los NULL impiden
-- instalar CT156 con definiciones, ACL o dependencias de un clon intermedio.
DO $preimagen_consumidores$
DECLARE x record;p record;acl_sha text;efectiva_sha text;dep_sha text;shdep_sha text;
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('leer_contratos_bolsa_v1(bigint,text,integer)','8c5b5e08083d4dcd17d7c4e4dd2572d41260b8f711d18228637d97c4a6a2ae32','742ae0083bdb71bad9d375d794e2574fff3105b5e9d29e65041ec96d3e005fba',ARRAY['search_path=pg_catalog','TimeZone=UTC']::text[],'dcf5f8d75fd8215c95882934ba547ceac7633a0bfa6abbbf529168f90c162cda','7ad94585991db40b728799f4c3730af4f1ee6479c77324644660f0f6a7d27b8f','500280fa5d41d1efc1548a507898217068e5d2bbbb938ec79018a8941cac63b4','93d1e226d0b9bedec88916939e5e7451c260efca6b259034b3c84150712d1a85'),
  ('leer_ceses_bolsa_v1(bigint,text,integer)','47761fdf9a31d9f9a7a3fb7f97616fd0ac22782aba64c60dfd26e2098d78bb08','854dd3a360b6b041b41528b149fad64302e34e639cb7eae33c3a36dad7687a20',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']::text[],'dcf5f8d75fd8215c95882934ba547ceac7633a0bfa6abbbf529168f90c162cda','7ad94585991db40b728799f4c3730af4f1ee6479c77324644660f0f6a7d27b8f','0a4744f34f31a3aa1ca22c4181b039c17d7e5f82a02eebdb72af5e6c3563f0e8','05a29be840f8136ae897e2cffd289d3209935aa13b4b7877b788997267c4edc5'),
  ('verificar_cese_publicado_bolsa_v1(text,text,bigint)','22e44539371daf1fbb66a63a28362f292d3b2411a2ca7fc9a218e7202f546a1d','89162932aeae938aeb25a3b954baf9e3ea1531443f47ccf540f370cb647ed512',ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC']::text[],'29852ef9a9bbe4fb1bc8fed64ab13508b458a0b86347433d77ff71c7b521051e','8fdfa240d6514e3904a5b69082da6cad46e68f9a1ec06917a272114364cfbdef','92a369c0202ccd660a0b57bc73d8275b5d048fbee2cc5af9ca9a064b7885665c','670ea696738c84cd1886eb901cac7d61bfc3f4ad44bc9022bb5c42b702cabab2'),
  ('registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)','69db43aa4ec2563d3f7e35e474805a8ca50ea78a584051ea472f3b99849df82e','44dab34a4359acc0982096b3d26e9583570c69b60fc17c68030afe53ea0686f7',ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s','statement_timeout=5s']::text[],'dcf5f8d75fd8215c95882934ba547ceac7633a0bfa6abbbf529168f90c162cda','7ad94585991db40b728799f4c3730af4f1ee6479c77324644660f0f6a7d27b8f','b38d64961772bf3484c8a5b602abc2d6e3c473b536b9a830dbdf26f2c2d78bd7','6769e2e3e3187c8111093f35969f7b0d385b1d838d81dfee7a7b3047320f6176')
 ) v(firma,def_sha,src_sha,config,acl_sha,efectiva_sha,dep_sha,shdep_sha) LOOP
  IF x.def_sha IS NULL OR x.src_sha IS NULL OR x.config IS NULL OR x.acl_sha IS NULL
     OR x.efectiva_sha IS NULL OR x.dep_sha IS NULL OR x.shdep_sha IS NULL THEN
   RAISE EXCEPTION 'CT156: falta preimagen final de %',x.firma USING ERRCODE='55000';
  END IF;
  SELECT q.oid,q.pronamespace,q.proowner,q.prosecdef,q.proconfig,
         pg_get_functiondef(q.oid) AS def,q.prosrc INTO STRICT p
  FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object(
   'grantee',CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,
   'grantor',pg_get_userbyid(a.grantor),'privilege_type',a.privilege_type,
   'is_grantable',a.is_grantable) ORDER BY a.grantee,a.grantor,a.privilege_type,a.is_grantable),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO acl_sha FROM aclexplode(coalesce((SELECT proacl FROM pg_proc WHERE oid=p.oid),
                                         acldefault('f',p.proowner))) a;
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_object(
   'rolname',r.rolname,'execute',has_function_privilege(r.oid,p.oid,'EXECUTE'),
   'schema_usage',has_schema_privilege(r.oid,p.pronamespace,'USAGE')) ORDER BY r.rolname),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO efectiva_sha FROM pg_roles r;
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(d) ORDER BY
   d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO dep_sha FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid;
  SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(d) ORDER BY
   d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype,d.dbid),'[]'::jsonb)::text,'UTF8')),'hex')
  INTO shdep_sha FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=p.oid;
  IF p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
     OR p.prosecdef IS DISTINCT FROM true OR p.proconfig IS DISTINCT FROM x.config
     OR encode(sha256(convert_to(p.def,'UTF8')),'hex') IS DISTINCT FROM x.def_sha
     OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM x.src_sha
     OR acl_sha IS DISTINCT FROM x.acl_sha OR efectiva_sha IS DISTINCT FROM x.efectiva_sha
     OR dep_sha IS DISTINCT FROM x.dep_sha OR shdep_sha IS DISTINCT FROM x.shdep_sha THEN
   RAISE EXCEPTION 'CT156: preimagen incompatible: %',x.firma USING ERRCODE='55000';
  END IF;
 END LOOP;
END $preimagen_consumidores$;
-- Sin reescribir filas: NULL conserva posición 0 en cualquier origen previo.
ALTER TABLE vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 ADD COLUMN transaccion_publicacion xid8;
ALTER TABLE vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 ALTER COLUMN transaccion_publicacion SET DEFAULT pg_current_xact_id();
CREATE INDEX origen_personal_b2_publicacion_ct156 ON vec_contratacion_temporal.origen_incorporacion_personal_b2_v1((coalesce(transaccion_publicacion,'0'::xid8)),outbox_ref);
-- CT75 conserva el instante original, con su zona y microsegundos. B2
-- confirma fechas civiles: su contrato de intercambio usa medianoche UTC.
CREATE FUNCTION vec_contratacion_temporal.origen_publicacion_bolsa_ct156(p_org text,p_exp text,p_recibo text,p_protocolo text)
RETURNS TABLE(organizacion_ref text,expediente_ref text,recibo_ref text,inicio_instante timestamptz,relacion_ref text)
LANGUAGE sql STABLE SET search_path=pg_catalog SET row_security='on' AS $f$
 SELECT i.organizacion_ref,i.expediente_ref,i.recibo_ref,
        (i.material_json#>>'{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz,
        i.material_json#>>'{Confirmacion,ResultadoPersonal,relacion_ref}'
 FROM vec_contratacion_temporal.incorporacion_registro_v2 i
 WHERE p_protocolo='ejercicio_v2' AND i.organizacion_ref=p_org AND i.expediente_ref=p_exp AND i.recibo_ref=p_recibo
 UNION ALL
 SELECT i.organizacion_ref,i.expediente_ref,i.recibo_ref,
        (p.material_json#>>'{material,desde}')::date::timestamp AT TIME ZONE 'UTC',i.relacion_ref
 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i
 JOIN vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 p
   ON p.organizacion_ref=i.organizacion_ref AND p.expediente_ref=i.expediente_ref AND p.plan_ref=i.plan_ref
 WHERE p_protocolo='personal_b2_v1' AND i.organizacion_ref=p_org AND i.expediente_ref=p_exp AND i.recibo_ref=p_recibo;
$f$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.origen_publicacion_bolsa_ct156(text,text,text,text) FROM PUBLIC;
DO $acl$
DECLARE g record; f oid:='vec_contratacion_temporal.origen_publicacion_bolsa_ct156(text,text,text,text)'::regprocedure;
BEGIN
 FOR g IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
 WHERE p.oid=f AND x.grantee<>p.proowner LOOP
 EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::regprocedure,CASE WHEN g.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(g.grantee)) END);
 END LOOP;
END $acl$;
DO $parches$
DECLARE x record; p record; def text; meta jsonb; config_esperada text[];
        dependencias jsonb; compartidas jsonb; permisos_efectivos jsonb;
BEGIN
 FOR x IN SELECT * FROM (VALUES
('leer_contratos_bolsa_v1(bigint,text,integer)',$old$          JOIN vec_contratacion_temporal.incorporacion_registro_v2 r ON r.recibo_ref = c.incorporacion_ref$old$,$new$          CROSS JOIN LATERAL vec_contratacion_temporal.origen_publicacion_bolsa_ct156(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref),c.incorporacion_protocolo) r$new$),
('leer_contratos_bolsa_v1(bigint,text,integer)',$old$                   'llamamiento_ref', c.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),$old$,$new$                   'llamamiento_ref', c.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(r.inicio_instante),$new$),
('leer_contratos_bolsa_v1(bigint,text,integer)',$old$    ), base AS (
        SELECT * FROM incorporaciones UNION ALL SELECT * FROM ceses$old$,$new$    ), incorporaciones_b2 AS (
        SELECT i.outbox_ref AS origen,
               vec_contratacion_temporal.posicion_contrato_bolsa_v1(i.transaccion_publicacion) AS posicion,
               i.registrada_en AS creada,
               'evento:ct:contrato-bolsa:'||encode(sha256(convert_to('incorporacion'||chr(31)||i.outbox_ref,'UTF8')),'hex') AS ref,
               jsonb_build_object(
                   'esquema','vec.contratacion-temporal.contrato-bolsa.v1','tipo','incorporacion',
                   'origen_ref',i.outbox_ref,'organizacion_ref',i.organizacion_ref,'expediente_ref',i.expediente_ref,
                   'llamamiento_ref',p.material_json#>>'{material,bolsa,llamamiento_ref}',
                   'inicio',vec_contratacion_temporal.instante_contrato_bolsa_v1((p.material_json#>>'{material,desde}')::date::timestamp AT TIME ZONE 'UTC'),
                   'fin_previsto',vec_contratacion_temporal.instante_contrato_bolsa_v1(nullif(p.material_json#>>'{material,hasta}','')::date::timestamp AT TIME ZONE 'UTC'),
                   'modalidad_clave',e.agregado_json#>>'{analisis,modalidad_clave}',
                   'categoria_ref',e.agregado_json#>>'{analisis,categoria_ref}',
                   'causa_clave',e.agregado_json#>>'{analisis,causa_clave}',
                   'ocurrido_en',vec_contratacion_temporal.instante_contrato_bolsa_v1(i.registrada_en)) AS cuerpo
        FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 i
        JOIN vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 p
          ON p.organizacion_ref=i.organizacion_ref AND p.expediente_ref=i.expediente_ref AND p.plan_ref=i.plan_ref
        JOIN vec_contratacion_temporal.expediente_version_integral e
          ON e.expediente_ref=p.expediente_ref AND e.version=p.version_expediente
        JOIN vec_contratacion_temporal.outbox_expediente_integral o
          ON o.evento_ref=i.outbox_ref AND o.expediente_ref=i.expediente_ref AND o.version_expediente=p.version_expediente
         AND o.tipo_evento='ct.incorporacion-personal.v1'
        WHERE p.material_json#>>'{material,bolsa,llamamiento_ref}' IS NOT NULL
          AND (coalesce(i.transaccion_publicacion,'0'::xid8)<pg_snapshot_xmin(pg_current_snapshot())
               OR i.transaccion_publicacion=pg_current_xact_id_if_assigned())
          AND (p_desde_posicion IS NULL
               OR (vec_contratacion_temporal.posicion_contrato_bolsa_v1(i.transaccion_publicacion),i.outbox_ref)>(p_desde_posicion,p_desde_ref))
        ORDER BY 2,1 LIMIT p_limite
    ), base AS (
        SELECT * FROM incorporaciones UNION ALL SELECT * FROM ceses UNION ALL SELECT * FROM incorporaciones_b2$new$),
('leer_ceses_bolsa_v1(bigint,text,integer)',$old$SELECT c.*, i.material_json,$old$,$new$SELECT c.*, i.inicio_instante, i.relacion_ref,$new$),
('leer_ceses_bolsa_v1(bigint,text,integer)',$old$          JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref$old$,$new$          CROSS JOIN LATERAL vec_contratacion_temporal.origen_publicacion_bolsa_ct156(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref),c.incorporacion_protocolo) i$new$),
('leer_ceses_bolsa_v1(bigint,text,integer)',$old$vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (c.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz)$old$,$new$vec_contratacion_temporal.instante_contrato_bolsa_v1(c.inicio_instante)$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$SELECT c.*, i.material_json,$old$,$new$SELECT c.*, i.inicio_instante, i.relacion_ref,$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$          JOIN vec_contratacion_temporal.incorporacion_registro_v2 i ON i.recibo_ref=c.incorporacion_ref$old$,$new$          CROSS JOIN LATERAL vec_contratacion_temporal.origen_publicacion_bolsa_ct156(c.organizacion_ref,c.expediente_ref,coalesce(c.incorporacion_ref,c.incorporacion_b2_ref),c.incorporacion_protocolo) i$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (x.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz)$old$,$new$vec_contratacion_temporal.instante_contrato_bolsa_v1(x.inicio_instante)$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$           x.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}',$old$,$new$           x.relacion_ref,$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$x.recibo_ref,x.incorporacion_ref,x.llamamiento_ref,$old$,$new$x.recibo_ref,coalesce(x.incorporacion_ref,x.incorporacion_b2_ref),x.llamamiento_ref,$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$       AND x.material_json #>> '{Confirmacion,ResultadoPersonal,relacion_ref}' ~ '^ref:[0-9a-f]{64}$'$old$,$new$       AND ((x.incorporacion_protocolo='ejercicio_v2' AND x.relacion_ref ~ '^ref:[0-9a-f]{64}$')
            OR (x.incorporacion_protocolo='personal_b2_v1' AND x.relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'))$new$),
('verificar_cese_publicado_bolsa_v1(text,text,bigint)',$old$'incorporacion_ref',x.incorporacion_ref,$old$,$new$'incorporacion_ref',coalesce(x.incorporacion_ref,x.incorporacion_b2_ref),$new$),
('registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)',$old$ IF NOT FOUND THEN RAISE EXCEPTION 'CT75: expediente no disponible' USING ERRCODE='55000'; END IF;$old$,$new$ IF NOT FOUND THEN RAISE EXCEPTION 'CT75: expediente no disponible' USING ERRCODE='55000'; END IF;
 -- CT156: exclusión recíproca. Ambas fachadas exigen SERIALIZABLE,
 -- bloquean expediente y leen el origen contrario antes de escribir;
 -- SSI rechaza también la carrera con instantánea anterior (40001).
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.plan_incorporacion_personal_b2_v1 b
   WHERE b.organizacion_ref=preparacion->>'OrganizacionRef' AND b.expediente_ref=s->>'expediente_ref')
 OR EXISTS(SELECT 1 FROM vec_contratacion_temporal.origen_incorporacion_personal_b2_v1 b
   WHERE b.organizacion_ref=preparacion->>'OrganizacionRef' AND b.expediente_ref=s->>'expediente_ref')
 THEN RAISE EXCEPTION 'CT156: expediente con origen B2' USING ERRCODE='55000'; END IF;$new$)
 ) v(firma,anterior,nuevo) LOOP
 SELECT pg_get_functiondef(q.oid) AS def,to_jsonb(q)-'prosrc' AS meta,
        q.proconfig,q.prosecdef INTO STRICT p
 FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma) AND q.proowner=current_user::regrole;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
  d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO dependencias FROM pg_depend d WHERE d.classid='pg_proc'::regclass
  AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
  d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype,d.dbid),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass
  AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
 SELECT coalesce(jsonb_agg(jsonb_build_object(
  'rolname',r.rolname,'execute',has_function_privilege(r.oid,q.oid,'EXECUTE'),
  'schema_usage',has_schema_privilege(r.oid,q.pronamespace,'USAGE')) ORDER BY r.rolname),'[]'::jsonb)
 INTO permisos_efectivos FROM pg_roles r CROSS JOIN pg_proc q
 WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma);
 IF NOT p.prosecdef OR p.proconfig IS NULL
    OR (SELECT count(*) FROM unnest(p.proconfig) c WHERE c LIKE 'search_path=%')<>1
    OR NOT p.proconfig && ARRAY['search_path=pg_catalog','search_path=pg_catalog, pg_temp'] THEN
  RAISE EXCEPTION 'CT156: entorno heredado incompatible: %',x.firma USING ERRCODE='55000';
 END IF;
 config_esperada:=array_replace(p.proconfig,'search_path=pg_catalog','search_path=pg_catalog, pg_temp');
 meta:=jsonb_set(p.meta,'{proconfig}',to_jsonb(config_esperada));
 IF p.proconfig @> ARRAY['search_path=pg_catalog'] THEN
  EXECUTE format('ALTER FUNCTION %s SET search_path TO pg_catalog, pg_temp',
                 to_regprocedure('vec_contratacion_temporal.'||x.firma));
 END IF;
 SELECT pg_get_functiondef(q.oid) INTO STRICT def FROM pg_proc q
  WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)
    AND q.proconfig IS NOT DISTINCT FROM config_esperada;
 IF length(def)-length(replace(def,x.anterior,''))<>length(x.anterior)
 THEN RAISE EXCEPTION 'CT156: preimagen incompatible: %',x.firma USING ERRCODE='55000'; END IF;
 def:=replace(def,x.anterior,x.nuevo); EXECUTE def;
 IF (SELECT pg_get_functiondef(q.oid) FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM def
 OR (SELECT to_jsonb(q)-'prosrc' FROM pg_proc q WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
     d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass
       AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM dependencias
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY
     d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype,d.dbid),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass
       AND d.objid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM compartidas
 OR (SELECT coalesce(jsonb_agg(jsonb_build_object(
     'rolname',r.rolname,'execute',has_function_privilege(r.oid,q.oid,'EXECUTE'),
     'schema_usage',has_schema_privilege(r.oid,q.pronamespace,'USAGE')) ORDER BY r.rolname),'[]'::jsonb)
     FROM pg_roles r CROSS JOIN pg_proc q
     WHERE q.oid=to_regprocedure('vec_contratacion_temporal.'||x.firma)) IS DISTINCT FROM permisos_efectivos
 THEN RAISE EXCEPTION 'CT156: firma/OID/ACL/metadatos alterados: %',x.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $parches$;
COMMIT;
