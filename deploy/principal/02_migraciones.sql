\set ON_ERROR_STOP on
-- Orden por dependencias: AD3-000044 -> Bolsa-000011 -> AD3-000045 -> Bolsa-000012 -> AD3-000046 -> Bolsa-000013.
-- Los seis cambios forman una sola transacción; finalizar vale ROLLBACK en el ensayo.
BEGIN;
-- INICIO deploy/postgresql/autorizacion_atestada_v3/migraciones/000044_consumidor_borrador_llamamiento.up.sql
-- AD3-000044. Extiende estructuralmente el núcleo V3 posterior a AD3-32 y
-- AD3-43. No usa autohuellas: preserva las guardas previas y comprueba las
-- fachadas antecedentes con su firma nominal.
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000044',0));
DO $precondicion$
DECLARE
 f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND prokind='f' AND provolatile='v' AND pg_get_function_identity_arguments(oid)='p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea' AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
    THEN
  RAISE EXCEPTION 'estructura V3 posterior a AD3-32/43 incompatible para B-BACK-01' USING ERRCODE='55000';
 END IF;
END $precondicion$;
DO $nucleo$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; esperada text; actual text; reconstruida text;
 metadata jsonb; deps jsonb; acl aclitem[]; propietario oid; configuracion text[]; es_definidora boolean;
 marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 exclusion_ancla text := $ct$               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_participaciones_propias_bolsa'$ct$;
 exclusion_extension text := exclusion_ancla||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''creacion_borrador_llamamiento_interno_bolsa''\n               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_borrador_llamamiento_interno_bolsa''';
 runtime_ancla text := $g$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa')$g$;
 runtime_extension text := $g$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa')$g$;
 extension text := $p$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'creacion_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_borradores_llamamiento_interno')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'borrador_llamamiento_interno'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno')
$p$;
BEGIN
 SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
   INTO STRICT original,metadata,acl,propietario,configuracion,es_definidora
   FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,exclusion_ancla,''))<>length(exclusion_ancla)
    OR length(original)-length(replace(original,runtime_ancla,''))<>length(runtime_ancla)
    OR strpos(original,'creacion_borrador_llamamiento_interno_bolsa')<>0
    OR strpos(original,'consulta_participaciones_propias_bolsa')=0
    OR strpos(original,'despacho_correo_llamamiento_ct')=0
    OR strpos(original,'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva')=0 THEN
  RAISE EXCEPTION 'núcleo V3 AD3-32/43 no admite extensión B-BACK-01' USING ERRCODE='55000';
 END IF;
 -- Sólo se admiten tres inserciones nominales; las guardas AD3-32/43 y las
 -- restantes extensiones del núcleo se conservan byte a byte.
 esperada:=replace(original,exclusion_ancla,exclusion_extension);
 esperada:=replace(esperada,runtime_ancla,runtime_extension);
 esperada:=replace(esperada,marca,extension||marca);
 EXECUTE esperada;
 SELECT pg_get_functiondef(p.oid) INTO STRICT actual FROM pg_proc p WHERE p.oid=f;
 reconstruida:=replace(actual,extension||marca,marca);
 reconstruida:=replace(reconstruida,runtime_extension,runtime_ancla);
 reconstruida:=replace(reconstruida,exclusion_extension,exclusion_ancla);
 IF actual IS DISTINCT FROM esperada OR reconstruida IS DISTINCT FROM original THEN
  RAISE EXCEPTION 'B-BACK-01 modificó el núcleo fuera de sus tres extensiones' USING ERRCODE='55000';
 END IF;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM configuracion
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM es_definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN RAISE EXCEPTION 'B-BACK-01 alteró metadatos, ACL, configuración o dependencias V3' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE definicion text; nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT definicion FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c';
 IF strpos(definicion,'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1')<>0 OR strpos(definicion,'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1')<>0 OR strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(definicion,3)<>']))' THEN RAISE EXCEPTION 'audiencias incompatibles para B-BACK-01 V3' USING ERRCODE='55000'; END IF;
 nueva:=left(definicion,length(definicion)-3)||', ''vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1''::text, ''vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B-BACK-01 crear inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'borrador_llamamiento_interno' OR d->>'finalidad' IS DISTINCT FROM 'gestion_borradores_llamamiento_interno' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'material B-BACK-01 crear rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('creacion_borrador_llamamiento_interno_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'B-BACK-01 crear requiere consumo nuevo' USING ERRCODE='42501'; END IF; RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true; END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B-BACK-01 consulta inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'borrador_llamamiento_interno' OR d->>'finalidad' IS DISTINCT FROM 'consulta_borrador_llamamiento_interno' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'material B-BACK-01 consulta rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('consulta_borrador_llamamiento_interno_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'B-BACK-01 consulta requiere consumo nuevo' USING ERRCODE='42501'; END IF; RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true; END $f$;

ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario;
-- Cierre efectivo frente a ACL por defecto hostiles: las únicas autoridades
-- que pueden ejecutar estas fachadas son AD3 y el propietario técnico Bolsa.
DO $acl_cerrada$
DECLARE funcion regprocedure; a record;
BEGIN
 FOREACH funcion IN ARRAY ARRAY['vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_propietario'::regrole LOOP
   IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
   ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
  END LOOP;
 END LOOP;
END $acl_cerrada$;
-- FIN deploy/postgresql/autorizacion_atestada_v3/migraciones/000044_consumidor_borrador_llamamiento.up.sql
-- INICIO deploy/postgresql/bolsa_llamamientos/migraciones/000011_borrador_llamamiento.up.sql
-- Candidata inédita local. Si la canónica ya tuviera 000011 instalada, esta
-- segregación se entrega como migración aditiva numerada allí; no se reaplica
-- ni se modifica esa historia desde este archivo.
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000011',0));

-- B-BACK-01 solo conserva un borrador interno. No contiene candidato,
-- contacto, plazo, envío, disponibilidad, elegibilidad ni resultado.
DO $precondicion$
BEGIN
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.borrador_llamamiento_interno') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_registrador_frontera' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND rolinherit AND NOT rolreplication AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles miembro ON miembro.oid=m.member WHERE miembro.rolname='vec_bolsa_llamamientos_registrador_frontera')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE r.oid<>'vec_bolsa_llamamientos_registrador_frontera'::regrole AND pg_has_role('vec_bolsa_llamamientos_registrador_frontera',r.oid,'MEMBER'))
    OR has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','USAGE,CREATE')
    OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class WHERE relowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_proc WHERE proowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_type WHERE typowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND ((c.relkind='S' AND has_sequence_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'USAGE,SELECT,UPDATE')) OR (c.relkind IN ('r','p','v','m','f') AND (has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))))
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_bolsa_llamamientos' AND has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',p.oid,'EXECUTE')) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='dependencia o estado incompatible para borrador interno';
 END IF;
END $precondicion$;

CREATE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(p jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_typeof(p)='object' AND p ?& ARRAY['resumen']
   AND (p-'resumen')='{}'::jsonb
   AND jsonb_typeof(p->'resumen')='string'
   AND octet_length(convert_to(p->>'resumen','UTF8')) BETWEEN 3 AND 2000
   AND p->>'resumen'=btrim(p->>'resumen')
   AND p->>'resumen' !~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email|tel[eé]fono)([^[:alpha:]]|$)'
   AND position('@' IN p->>'resumen')=0
   AND position(chr(8232) IN p->>'resumen')=0
   AND position(chr(8233) IN p->>'resumen')=0
$f$;

CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno (
 borrador_ref text PRIMARY KEY,
 propietario_ref text NOT NULL,
 unidad_ref text NOT NULL,
 ambito_ref text NOT NULL,
 clave_idempotencia text NOT NULL,
 contenido_canonico jsonb NOT NULL,
 huella_comando_sha256 text NOT NULL,
 estado text NOT NULL DEFAULT 'borrador_interno',
 version bigint NOT NULL DEFAULT 1,
 decision_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 creada_en timestamptz(6) NOT NULL,
 CONSTRAINT borrador_llamamiento_ref_check CHECK (borrador_ref ~ '^borrador-llamamiento:alta:[0-9a-f]{64}$'),
 CONSTRAINT borrador_llamamiento_propietario_check CHECK (propietario_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 CONSTRAINT borrador_llamamiento_referencias_check CHECK (unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' AND ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' AND clave_idempotencia ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'),
 CONSTRAINT borrador_llamamiento_huella_check CHECK (huella_comando_sha256 ~ '^[0-9a-f]{64}$' AND huella_comando_sha256<>repeat('0',64)),
 CONSTRAINT borrador_llamamiento_estado_version_check CHECK (estado='borrador_interno' AND version=1),
 CONSTRAINT borrador_llamamiento_contenido_check CHECK (vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(contenido_canonico)),
 UNIQUE(propietario_ref,unidad_ref,clave_idempotencia)
);
CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_historia (
 historia_ref text PRIMARY KEY,
 borrador_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.borrador_llamamiento_interno(borrador_ref),
 secuencia bigint NOT NULL CHECK (secuencia>0),
 anterior_sha256 text NOT NULL CHECK (anterior_sha256 ~ '^[0-9a-f]{64}$'),
 hecho_canonico bytea NOT NULL CHECK (octet_length(hecho_canonico) BETWEEN 1 AND 16384),
 huella_sha256 text NOT NULL CHECK (huella_sha256=encode(sha256(hecho_canonico),'hex')),
 UNIQUE(borrador_ref,secuencia)
);
CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_auditoria (
 auditoria_ref text PRIMARY KEY,
 borrador_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.borrador_llamamiento_interno(borrador_ref),
 accion text NOT NULL CHECK (accion IN ('crear','consultar')),
 decision_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(borrador_ref,accion,decision_ref)
);
CREATE TABLE vec_bolsa_llamamientos.borrador_llamamiento_outbox (
 evento_ref text PRIMARY KEY,
 borrador_ref text NOT NULL UNIQUE REFERENCES vec_bolsa_llamamientos.borrador_llamamiento_interno(borrador_ref),
 evento_canonico bytea NOT NULL CHECK (octet_length(evento_canonico) BETWEEN 1 AND 16384),
 huella_sha256 text NOT NULL CHECK (huella_sha256=encode(sha256(evento_canonico),'hex')),
 emitido_en timestamptz(6) NOT NULL
);
-- Bitácora E06 de intentos fallidos/indeterminados. No es historia del
-- agregado: no contiene cuerpo, clave, referencia de recurso ni material AD3.
CREATE TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento (
 intento_ref text PRIMARY KEY,
 correlacion_ref text NOT NULL CHECK (correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'),
 accion text NOT NULL CHECK (accion IN ('crear','consultar')),
 ruta_clase text NOT NULL CHECK (ruta_clase IN ('coleccion','detalle')),
 actor_ref text CHECK (actor_ref IS NULL OR actor_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 resultado text NOT NULL CHECK (resultado IN ('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado')),
 registrada_en timestamptz(6) NOT NULL
);

CREATE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='historia de borrador llamamiento inmutable'; END $f$;
CREATE TRIGGER borrador_llamamiento_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_interno FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER borrador_llamamiento_historia_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_historia FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER borrador_llamamiento_auditoria_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_auditoria FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER borrador_llamamiento_outbox_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.borrador_llamamiento_outbox FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();
CREATE TRIGGER bitacora_intento_borrador_llamamiento_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion();

ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_interno FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_historia FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_auditoria ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_auditoria FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.borrador_llamamiento_outbox FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento FORCE ROW LEVEL SECURITY;
CREATE POLICY borrador_llamamiento_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_interno TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY borrador_llamamiento_historia_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_historia TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY borrador_llamamiento_auditoria_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_auditoria TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY borrador_llamamiento_outbox_solo_propietario ON vec_bolsa_llamamientos.borrador_llamamiento_outbox TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');
CREATE POLICY bitacora_intento_borrador_llamamiento_solo_propietario ON vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario');

CREATE FUNCTION vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user OR
    NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_bolsa_llamamientos\_%' ESCAPE '\'
           AND r.rolname <> 'vec_bolsa_llamamientos_ejecutor'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_contratacion_temporal\_%' ESCAPE '\'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='runtime de borrador llamamiento rechazado';
 END IF;
END $f$;

-- El registrador de frontera opera después de un rechazo o rollback de negocio.
-- Su identidad técnica es distinta del ejecutor y no obtiene funciones del agregado.
CREATE FUNCTION vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento()
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user OR
    NOT pg_has_role(session_user,'vec_bolsa_llamamientos_registrador_frontera','MEMBER') OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_bolsa_llamamientos\_%' ESCAPE '\'
           AND r.rolname <> 'vec_bolsa_llamamientos_registrador_frontera'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    EXISTS (
        SELECT 1 FROM pg_roles r
         WHERE r.rolname LIKE 'vec_contratacion_temporal\_%' ESCAPE '\'
           AND pg_has_role(session_user,r.oid,'MEMBER')
    ) OR
    current_setting('transaction_isolation')<>'serializable' OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='runtime de registrador de frontera rechazado';
 END IF;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(
 p_comando bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(borrador_ref text,recibo_ref text,huella_comando_sha256 text,reintento_idempotente boolean,registrado_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; contenido jsonb; canon text; h text; h_contexto text; preimagen_contexto text; ref text; consumo record; existente record; ahora timestamptz; hecho bytea; evento bytea;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento();
 IF p_comando IS NULL OR octet_length(p_comando) NOT BETWEEN 1 AND 16384 THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador inválido'; END IF;
 BEGIN c:=convert_from(p_comando,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador inválido'; END;
 IF jsonb_typeof(c)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>6 OR NOT c ?& ARRAY['esquema','propietario_ref','unidad_ref','ambito_ref','clave_idempotencia','contenido'] OR c->>'esquema'<>'vec.bolsa.llamamiento.borrador-interno.crear.v1' OR jsonb_typeof(c->'propietario_ref')<>'string' OR c->>'propietario_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR jsonb_typeof(c->'unidad_ref')<>'string' OR c->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR jsonb_typeof(c->'ambito_ref')<>'string' OR c->>'ambito_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR jsonb_typeof(c->'clave_idempotencia')<>'string' OR c->>'clave_idempotencia' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador inválido'; END IF;
 contenido:=c->'contenido';
 IF vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(contenido) IS NOT TRUE THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='contenido de borrador inválido'; END IF;
 canon:=format('{"esquema":%s,"propietario_ref":%s,"unidad_ref":%s,"ambito_ref":%s,"clave_idempotencia":%s,"contenido":{"resumen":%s}}',to_json(c->>'esquema')::text,to_json(c->>'propietario_ref')::text,to_json(c->>'unidad_ref')::text,to_json(c->>'ambito_ref')::text,to_json(c->>'clave_idempotencia')::text,to_json(contenido->>'resumen')::text);
 IF p_comando IS DISTINCT FROM convert_to(canon,'UTF8') THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='comando de borrador no canónico'; END IF;
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex'); ref:='borrador-llamamiento:alta:'||h;
 PERFORM pg_advisory_xact_lock(hashtextextended('borrador-llamamiento:'||(c->>'propietario_ref')||':'||(c->>'unidad_ref')||':'||(c->>'clave_idempotencia'),0));
 SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='autorización de borrador no ligada'; END;
 preimagen_contexto:=format('{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{}}',to_json(c->>'ambito_ref')::text,to_json(c->>'unidad_ref')::text);
 h_contexto:=encode(sha256(convert_to(preimagen_contexto,'UTF8')),'hex');
 IF consumo.efecto_ref IS DISTINCT FROM ref OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM c->>'propietario_ref'
    OR d->>'accion' IS DISTINCT FROM 'bolsa.llamamiento.borrador_interno.crear'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'borrador_llamamiento_interno'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_borradores_llamamiento_interno'
    OR d->>'recurso_ref' IS DISTINCT FROM ref
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h_contexto
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM h_contexto THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='autorización de borrador no ligada'; END IF;
 SELECT * INTO existente FROM vec_bolsa_llamamientos.borrador_llamamiento_interno WHERE propietario_ref=c->>'propietario_ref' AND unidad_ref=c->>'unidad_ref' AND clave_idempotencia=c->>'clave_idempotencia' FOR SHARE;
 IF FOUND THEN
  IF existente.huella_comando_sha256 IS DISTINCT FROM h THEN RAISE EXCEPTION USING ERRCODE='VBL01',MESSAGE='clave de borrador divergente'; END IF;
  RETURN QUERY SELECT existente.borrador_ref,existente.recibo_ref,existente.huella_comando_sha256,true,existente.creada_en; RETURN;
 END IF;
 ahora:=clock_timestamp();
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_interno VALUES(ref,c->>'propietario_ref',c->>'unidad_ref',c->>'ambito_ref',c->>'clave_idempotencia',contenido,h,'borrador_interno',1,consumo.decision_ref,'recibo:'||translate(h,'0123456789','ghijklmnop'),ahora);
 hecho:=convert_to(jsonb_build_object('esquema','vec.bolsa.llamamiento.borrador.historia.v1','borrador_ref',ref,'accion','crear','version',1,'decision_ref',consumo.decision_ref,'registrada_en',ahora)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_historia VALUES('historia:'||translate(h,'0123456789','ghijklmnop'),ref,1,repeat('0',64),hecho,encode(sha256(hecho),'hex'));
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_auditoria VALUES('auditoria:'||translate(h,'0123456789','ghijklmnop'),ref,'crear',consumo.decision_ref,ahora);
 evento:=convert_to(jsonb_build_object('esquema','vec.bolsa.llamamiento.borrador.outbox.v1','tipo','bolsa.llamamiento.borrador_interno.creado','borrador_ref',ref,'recibo_ref','recibo:'||translate(h,'0123456789','ghijklmnop'),'emitido_en',ahora)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_outbox VALUES('evento:'||translate(h,'0123456789','ghijklmnop'),ref,evento,encode(sha256(evento),'hex'),ahora);
 RETURN QUERY SELECT ref,'recibo:'||translate(h,'0123456789','ghijklmnop'),h,false,ahora;
END $f$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(
 p_consulta bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(borrador_ref text,propietario_ref text,unidad_ref text,ambito_ref text,contenido_canonico jsonb,estado text,version bigint,huella_comando_sha256 text,recibo_ref text,creada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE b record; consumo record; c jsonb; d jsonb; h_contexto text; preimagen_contexto text; ahora timestamptz;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento();
 IF p_consulta IS NULL OR octet_length(p_consulta) NOT BETWEEN 1 AND 4096 THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='consulta de borrador inválida'; END IF;
 BEGIN c:=convert_from(p_consulta,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='consulta de borrador inválida'; END;
 IF jsonb_typeof(c)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(c))<>5 OR NOT c ?& ARRAY['esquema','borrador_ref','propietario_ref','unidad_ref','ambito_ref'] OR c->>'esquema'<>'vec.bolsa.llamamiento.borrador-interno-consulta.v1' OR c->>'borrador_ref' !~ '^borrador-llamamiento:alta:[0-9a-f]{64}$' OR c->>'propietario_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$' OR c->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR c->>'ambito_ref' !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='consulta de borrador inválida'; END IF;
 SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de borrador no autorizada'; END;
 preimagen_contexto:=format('{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{}}',to_json(c->>'ambito_ref')::text,to_json(c->>'unidad_ref')::text);
 h_contexto:=encode(sha256(convert_to(preimagen_contexto,'UTF8')),'hex');
 IF consumo.efecto_ref IS DISTINCT FROM c->>'borrador_ref' OR consumo.consumo_nuevo IS NOT TRUE
    OR d->>'principal_id' IS DISTINCT FROM c->>'propietario_ref'
    OR d->>'accion' IS DISTINCT FROM 'bolsa.llamamiento.borrador_interno.consultar'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'borrador_llamamiento_interno'
    OR d->>'finalidad' IS DISTINCT FROM 'consulta_borrador_llamamiento_interno'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'borrador_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h_contexto
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM h_contexto THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consulta de borrador no autorizada'; END IF;
 SELECT bi.* INTO b FROM vec_bolsa_llamamientos.borrador_llamamiento_interno bi WHERE bi.borrador_ref=c->>'borrador_ref' AND bi.propietario_ref=c->>'propietario_ref' AND bi.unidad_ref=c->>'unidad_ref' AND bi.ambito_ref=c->>'ambito_ref' FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='borrador no disponible'; END IF;
 ahora:=clock_timestamp(); INSERT INTO vec_bolsa_llamamientos.borrador_llamamiento_auditoria (auditoria_ref,borrador_ref,accion,decision_ref,registrada_en) VALUES ('auditoria:lectura:'||translate(encode(sha256(convert_to((c->>'borrador_ref')||':'||consumo.decision_ref,'UTF8')),'hex'),'0123456789','ghijklmnop'),c->>'borrador_ref','consultar',consumo.decision_ref,ahora);
 RETURN QUERY SELECT b.borrador_ref,b.propietario_ref,b.unidad_ref,b.ambito_ref,b.contenido_canonico,b.estado,b.version,b.huella_comando_sha256,b.recibo_ref,b.creada_en;
END $f$;

-- Firma: registrar_intento_borrador_llamamiento_v1(text,text,text,text,text).
-- Se invoca en una operación posterior a una transacción fallida; por eso no
-- tiene FK ni toca agregado, historia, auditoría de éxito ni outbox.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(
 p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text
) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'
    OR p_accion NOT IN ('crear','consultar') OR p_ruta_clase NOT IN ('coleccion','detalle')
    OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$')
    OR p_resultado NOT IN ('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado') THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='intento de borrador inválido';
 END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento (intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en)
 VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;

REVOKE ALL ON vec_bolsa_llamamientos.borrador_llamamiento_interno,vec_bolsa_llamamientos.borrador_llamamiento_historia,vec_bolsa_llamamientos.borrador_llamamiento_auditoria,vec_bolsa_llamamientos.borrador_llamamiento_outbox,vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_registrador_frontera;
-- Las ACL históricas no se revocan ni normalizan aquí: una concesión residual
-- al registrador, incluso por PUBLIC o membresía, rechaza la candidata.
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(jsonb),vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion(),vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento(),vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento(),vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.borrador_llamamiento_contenido_valido(jsonb),vec_bolsa_llamamientos.borrador_llamamiento_rechazar_mutacion(),vec_bolsa_llamamientos.exigir_runtime_borrador_llamamiento(),vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento(),vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_registrador_frontera;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) TO vec_bolsa_llamamientos_registrador_frontera;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_ejecutor;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text) FROM vec_bolsa_llamamientos_registrador_frontera;
-- Las ACL por defecto pueden haber sido ampliadas por una instalación anfitriona.
-- RLS no cubre TRUNCATE, REFERENCES ni TRIGGER: retirar toda concesión que no
-- pertenezca al propietario técnico, y para las fachadas solo conservar ejecutor.
DO $acl_cerrada$
DECLARE objeto regclass; funcion regprocedure; a record;
BEGIN
 FOREACH objeto IN ARRAY ARRAY['vec_bolsa_llamamientos.borrador_llamamiento_interno'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_historia'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_auditoria'::regclass,'vec_bolsa_llamamientos.borrador_llamamiento_outbox'::regclass,'vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass] LOOP
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x WHERE c.oid=objeto AND x.grantee<>c.relowner LOOP
   IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON TABLE %s FROM PUBLIC',objeto::text);
   ELSE EXECUTE format('REVOKE ALL ON TABLE %s FROM %I',objeto::text,pg_get_userbyid(a.grantee)); END IF;
  END LOOP;
 END LOOP;
 FOREACH funcion IN ARRAY ARRAY['vec_bolsa_llamamientos.guardar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_bolsa_llamamientos.consultar_borrador_llamamiento_interno_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole LOOP
   IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
   ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
  END LOOP;
 END LOOP;
 funcion:='vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)'::regprocedure;
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_registrador_frontera'::regrole LOOP
  IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
  ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
 END LOOP;
END $acl_cerrada$;

-- Antes del COMMIT se comprueban privilegios efectivos, no solo ACL directas.
-- La única capacidad visible del grupo es la fachada exacta de bitácora.
DO $segregacion_efectiva$
DECLARE f regprocedure := 'vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','USAGE')
    OR has_schema_privilege('vec_bolsa_llamamientos_registrador_frontera','vec_bolsa_llamamientos','CREATE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE r.oid<>'vec_bolsa_llamamientos_registrador_frontera'::regrole AND pg_has_role('vec_bolsa_llamamientos_registrador_frontera',r.oid,'MEMBER'))
    OR EXISTS (SELECT 1 FROM pg_namespace WHERE nspowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class WHERE relowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_proc WHERE proowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_type WHERE typowner='vec_bolsa_llamamientos_registrador_frontera'::regrole)
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_bolsa_llamamientos' AND ((c.relkind='S' AND has_sequence_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'USAGE,SELECT,UPDATE')) OR (c.relkind IN ('r','p','v','m','f') AND (has_table_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege('vec_bolsa_llamamientos_registrador_frontera',c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))))
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_bolsa_llamamientos' AND p.oid<>f AND has_function_privilege('vec_bolsa_llamamientos_registrador_frontera',p.oid,'EXECUTE')) THEN
  RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='segregación efectiva del registrador de frontera incompatible';
 END IF;
END $segregacion_efectiva$;
-- FIN deploy/postgresql/bolsa_llamamientos/migraciones/000011_borrador_llamamiento.up.sql
-- INICIO deploy/postgresql/autorizacion_atestada_v3/migraciones/000045_consumidor_situacion_participacion.up.sql
-- AD3-000045. Extensión nominal B2 sobre la estructura de AD3-000044;
-- conserva las guardas anteriores sin exigir una autohuella del núcleo.
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000045',0));

DO $precondicion$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND prokind='f' AND provolatile='v' AND pg_get_function_identity_arguments(oid)='p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea' AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']) THEN
  RAISE EXCEPTION 'estructura AD3-000044 incompatible para B2' USING ERRCODE='55000';
 END IF;
END $precondicion$;

DO $nucleo$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; esperada text; actual text; reconstruida text; metadata jsonb; deps jsonb; acl aclitem[]; propietario oid; configuracion text[]; es_definidora boolean;
 marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 exclusion_pre text := $x$               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'$x$;
 exclusion_post text := exclusion_pre||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''situacion_participacion_bolsa''';
 runtime_pre text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa')$x$;
 runtime_post text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa')$x$;
 extension text := $p$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.situacion_participacion.cambiar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_situacion_participacion')
$p$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,metadata,acl,propietario,configuracion,es_definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,marca,''))<>length(marca) OR length(original)-length(replace(original,exclusion_pre,''))<>length(exclusion_pre) OR length(original)-length(replace(original,runtime_pre,''))<>length(runtime_pre) OR strpos(original,'creacion_borrador_llamamiento_interno_bolsa')=0 OR strpos(original,'consulta_participaciones_propias_bolsa')=0 OR strpos(original,'despacho_correo_llamamiento_ct')=0 OR strpos(original,'situacion_participacion_bolsa')<>0 THEN RAISE EXCEPTION 'núcleo AD3-000044 no admite extensión B2' USING ERRCODE='55000'; END IF;
 esperada:=replace(original,exclusion_pre,exclusion_post); esperada:=replace(esperada,runtime_pre,runtime_post); esperada:=replace(esperada,marca,extension||marca); EXECUTE esperada;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 reconstruida:=replace(actual,extension||marca,marca); reconstruida:=replace(reconstruida,runtime_post,runtime_pre); reconstruida:=replace(reconstruida,exclusion_post,exclusion_pre);
 IF actual IS DISTINCT FROM esperada OR reconstruida IS DISTINCT FROM original OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM configuracion OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM es_definidora OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN RAISE EXCEPTION 'B2 alteró el núcleo AD3 fuera de su extensión nominal' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$ DECLARE definicion text; nueva text; BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT definicion FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c';
 IF strpos(definicion,'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1')<>0 OR strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(definicion,3)<>']))' THEN RAISE EXCEPTION 'audiencias incompatibles para B2' USING ERRCODE='55000'; END IF;
 nueva:=left(definicion,length(definicion)-3)||', ''vec_bolsa_llamamientos.situacion_participacion.cambiar.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B2 inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.situacion_participacion.cambiar.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'material B2 rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('situacion_participacion_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'B2 requiere consumo nuevo' USING ERRCODE='42501'; END IF; RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true; END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario;
DO $acl_cerrada$
DECLARE funcion regprocedure := 'vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_propietario'::regrole LOOP
  IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
  ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
 END LOOP;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',funcion,'EXECUTE') OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.privilege_type='EXECUTE' AND (x.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole) OR x.is_grantable AND x.grantee='vec_bolsa_llamamientos_propietario'::regrole)) THEN RAISE EXCEPTION 'ACL B2 incompatible' USING ERRCODE='55000'; END IF;
END $acl_cerrada$;
-- FIN deploy/postgresql/autorizacion_atestada_v3/migraciones/000045_consumidor_situacion_participacion.up.sql
-- INICIO deploy/postgresql/bolsa_llamamientos/migraciones/000012_situacion_participacion.up.sql
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000012', 0));

-- La bitácora segregada existente conserva también los fallos de frontera B2;
-- no recibe referencias de participación, cuerpo, material V3 ni idempotencia.
DO $precondicion_bitacora$
DECLARE accion text; ruta text;
BEGIN
 SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check' AND contype='c';
 SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check' AND contype='c';
 IF accion<>$$CHECK (accion = ANY (ARRAY['crear'::text, 'consultar'::text]))$$ OR ruta<>$$CHECK (ruta_clase = ANY (ARRAY['coleccion'::text, 'detalle'::text]))$$ THEN RAISE EXCEPTION 'bitácora de frontera incompatible con B2' USING ERRCODE='55000'; END IF;
END $precondicion_bitacora$;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check CHECK (accion IN ('crear','consultar','cambiar_situacion'));
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check;
ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check CHECK (ruta_clase IN ('coleccion','detalle','situacion'));
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_ref text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();
 IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$'
    OR p_accion NOT IN ('crear','consultar','cambiar_situacion') OR p_ruta_clase NOT IN ('coleccion','detalle','situacion')
    OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$')
    OR p_resultado NOT IN ('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado') THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='intento de frontera Bolsa inválido';
 END IF;
 v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');
 INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento (intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en)
 VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp());
END $f$;

-- B2, Petición RRHH p.2: histórico append-only de la situación de cada participación.
CREATE TABLE vec_bolsa_llamamientos.situacion_participacion (
    participacion_ref text NOT NULL,
    situacion text NOT NULL CHECK (situacion IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde')),
    desde timestamptz(6) NOT NULL,
    hasta timestamptz(6),
    fecha_disponible timestamptz(6),
    motivo text NOT NULL CHECK (octet_length(motivo) BETWEEN 1 AND 1000 AND motivo = btrim(motivo)),
    actor text NOT NULL,
    registrada_en timestamptz(6) NOT NULL,
    clave_idempotencia text NOT NULL CHECK (octet_length(clave_idempotencia) BETWEEN 1 AND 256 AND clave_idempotencia = btrim(clave_idempotencia)),
    recibo_ref text NOT NULL CHECK (octet_length(recibo_ref) BETWEEN 1 AND 256 AND recibo_ref = btrim(recibo_ref)),
    PRIMARY KEY (participacion_ref, desde),
    UNIQUE (participacion_ref, clave_idempotencia),
    UNIQUE (recibo_ref),
    CHECK (hasta IS NULL OR hasta = desde),
    CHECK ((situacion = 'disponible_desde' AND fecha_disponible IS NOT NULL AND fecha_disponible > desde) OR (situacion <> 'disponible_desde' AND fecha_disponible IS NULL))
);
CREATE INDEX situacion_participacion_vigente ON vec_bolsa_llamamientos.situacion_participacion(participacion_ref, desde DESC);
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.situacion_participacion FORCE ROW LEVEL SECURITY;
CREATE POLICY situacion_participacion_solo_propietario ON vec_bolsa_llamamientos.situacion_participacion TO vec_bolsa_llamamientos_propietario USING (current_user = 'vec_bolsa_llamamientos_propietario') WITH CHECK (current_user = 'vec_bolsa_llamamientos_propietario');
REVOKE ALL ON vec_bolsa_llamamientos.situacion_participacion FROM PUBLIC;
CREATE TRIGGER situacion_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.situacion_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- Las participaciones ya constituidas, y las que constituya el comando
-- existente después de esta migración, nacen disponibles sin otro dato.
INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
SELECT e.participacion_ref,'disponible',c.confirmada_en,NULL,NULL,'Constitución de bolsa','sistema:constitucion',c.confirmada_en,'constitucion:' || e.participacion_ref,'recibo:situacion:constitucion:' || e.participacion_ref
  FROM vec_bolsa_llamamientos.constitucion_entrada e
  JOIN vec_bolsa_llamamientos.constitucion c ON c.instantanea_ref=e.instantanea_ref AND c.version_instantanea=e.version_instantanea;

CREATE FUNCTION vec_bolsa_llamamientos.iniciar_situacion_participacion_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v_desde timestamptz;
BEGIN
 SELECT c.confirmada_en INTO v_desde FROM vec_bolsa_llamamientos.constitucion c WHERE c.instantanea_ref=NEW.instantanea_ref AND c.version_instantanea=NEW.version_instantanea;
 IF v_desde IS NULL THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='constitucion inexistente'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(NEW.participacion_ref,'disponible',v_desde,NULL,NULL,'Constitución de bolsa','sistema:constitucion',v_desde,'constitucion:' || NEW.participacion_ref,'recibo:situacion:constitucion:' || NEW.participacion_ref);
 RETURN NEW;
END $f$;
CREATE TRIGGER constitucion_entrada_inicia_situacion_participacion
AFTER INSERT ON vec_bolsa_llamamientos.constitucion_entrada
FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.iniciar_situacion_participacion_v1();

CREATE FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(p_bolsa_ref text,p_participacion_ref text, p_situacion text, p_desde timestamptz, p_fecha_disponible timestamptz, p_motivo text, p_actor text, p_clave_idempotencia text, p_recibo_ref text, p_registrada_en timestamptz,p_capacidad bytea,p_decision bytea,p_motivo_autorizacion bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean, recibo_ref text, situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE anterior record; consumo record; decision jsonb;
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario' OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_situacion NOT IN ('disponible','no_disponible','trabajando','pendiente_incorporacion','renuncia','excluido','disponible_desde') OR p_desde IS NULL OR p_motivo IS NULL OR p_motivo<>btrim(p_motivo) OR octet_length(p_motivo) NOT BETWEEN 1 AND 1000 OR p_actor IS NULL OR p_clave_idempotencia IS NULL OR p_clave_idempotencia<>btrim(p_clave_idempotencia) OR octet_length(p_clave_idempotencia) NOT BETWEEN 1 AND 256 OR p_recibo_ref IS NULL OR p_recibo_ref<>btrim(p_recibo_ref) OR octet_length(p_recibo_ref) NOT BETWEEN 1 AND 256 OR p_registrada_en IS NULL OR (p_situacion='disponible_desde' AND p_fecha_disponible IS NULL) OR (p_situacion<>'disponible_desde' AND p_fecha_disponible IS NOT NULL) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='situacion invalida'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea) WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN RAISE EXCEPTION USING ERRCODE='23503',MESSAGE='participacion ajena a la bolsa'; END IF;
 SELECT * INTO anterior FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p_participacion_ref ORDER BY desde DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23503', MESSAGE='participacion inexistente'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 BEGIN decision:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END;
 IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE OR decision->>'principal_id' IS DISTINCT FROM p_actor OR decision->>'accion' IS DISTINCT FROM 'bolsa.situacion_participacion.cambiar' OR decision->>'modulo_id' IS DISTINCT FROM 'bolsa' OR decision->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR decision->>'finalidad' IS DISTINCT FROM 'gestion_situacion_participacion' OR decision->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR consumo.huella_efecto_sha256 IS DISTINCT FROM decision->>'contexto_recurso_huella_sha256' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='situacion no autorizada'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia AND (s.situacion<>p_situacion OR s.motivo<>p_motivo OR s.fecha_disponible IS DISTINCT FROM p_fecha_disponible)) THEN RAISE EXCEPTION USING ERRCODE='VBS01', MESSAGE='clave idempotente reutilizada con otro comando'; END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia) THEN RETURN QUERY SELECT true, s.recibo_ref, s.situacion, s.desde, s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia; RETURN; END IF;
 IF p_desde < anterior.desde THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='desde anterior a la situacion vigente'; END IF;
 IF p_situacion='disponible_desde' AND p_fecha_disponible<=p_registrada_en THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='fecha disponible no futura'; END IF;
 IF NOT ((anterior.situacion='disponible' AND p_situacion IN ('no_disponible','pendiente_incorporacion','renuncia','excluido')) OR (anterior.situacion='no_disponible' AND p_situacion IN ('disponible','excluido')) OR (anterior.situacion='pendiente_incorporacion' AND p_situacion IN ('trabajando','disponible','renuncia','excluido')) OR (anterior.situacion='trabajando' AND p_situacion IN ('disponible','disponible_desde','excluido')) OR (anterior.situacion='disponible_desde' AND p_situacion IN ('disponible','excluido')) OR (anterior.situacion='renuncia' AND p_situacion IN ('disponible','excluido'))) THEN RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='transicion de situacion invalida'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref) VALUES(p_participacion_ref,p_situacion,p_desde,NULL,p_fecha_disponible,p_motivo,p_actor,p_registrada_en,p_clave_idempotencia,p_recibo_ref);
 RETURN QUERY SELECT false,p_recibo_ref,p_situacion,p_desde,p_fecha_disponible;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.recuperar_situacion_participacion_v1(p_participacion_ref text,p_clave_idempotencia text)
RETURNS TABLE(recibo_ref text,situacion text,desde timestamptz,fecha_disponible timestamptz,motivo text)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT s.recibo_ref,s.situacion,s.desde,s.fecha_disponible,s.motivo FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref AND s.clave_idempotencia=p_clave_idempotencia
$f$;
CREATE FUNCTION vec_bolsa_llamamientos.leer_situacion_participacion_v1(p_participacion_ref text)
RETURNS TABLE(situacion text, desde timestamptz, fecha_disponible timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT s.situacion,s.desde,s.fecha_disponible FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p_participacion_ref ORDER BY s.desde DESC LIMIT 1
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_situacion_participacion_v1(text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.recuperar_situacion_participacion_v1(text,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.iniciar_situacion_participacion_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_situacion_participacion_v1(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_situacion_participacion_v1(text) TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.recuperar_situacion_participacion_v1(text,text) TO vec_bolsa_llamamientos_ejecutor;
-- FIN deploy/postgresql/bolsa_llamamientos/migraciones/000012_situacion_participacion.up.sql
-- INICIO deploy/postgresql/autorizacion_atestada_v3/migraciones/000046_consumidor_contacto_participacion.up.sql
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario; SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s'; SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000046',0)); DO $precondicion$ DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; BEGIN  IF current_user<>'vec_autorizacion_atestada_v3_propietario'     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL     OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL     OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND prokind='f' AND provolatile='v' AND pg_get_function_identity_arguments(oid)='p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea' AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']) THEN   RAISE EXCEPTION 'estructura AD3-000044 incompatible para B3' USING ERRCODE='55000';  END IF; END $precondicion$; DO $nucleo$ DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;  original text; esperada text; actual text; reconstruida text; metadata jsonb; deps jsonb; acl aclitem[]; propietario oid; configuracion text[]; es_definidora boolean;  marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';  exclusion_pre text := $x$               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'                AND p_perfil_mutacion IS DISTINCT FROM 'situacion_participacion_bolsa'$x$;  exclusion_post text := exclusion_pre||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''contacto_participacion_bolsa''';  runtime_pre text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'                OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa')$x$;  runtime_post text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'                OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'                OR p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa')$x$;  extension text := $p$           OR (  p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'  AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.registrar.v1'  AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'  AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'  AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'  AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'  AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_contactos_participacion')  OR (p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'  AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.consultar.v1'  AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.consultar'  AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.consultar'  AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'  AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'  AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_contactos_participacion') $p$; BEGIN  SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,metadata,acl,propietario,configuracion,es_definidora FROM pg_proc p WHERE p.oid=f;  SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;  IF length(original)-length(replace(original,marca,''))<>length(marca) OR length(original)-length(replace(original,exclusion_pre,''))<>length(exclusion_pre) OR length(original)-length(replace(original,runtime_pre,''))<>length(runtime_pre) OR strpos(original,'creacion_borrador_llamamiento_interno_bolsa')=0 OR strpos(original,'consulta_participaciones_propias_bolsa')=0 OR strpos(original,'despacho_correo_llamamiento_ct')=0 OR strpos(original,'situacion_participacion_bolsa')=0 OR strpos(original,'contacto_participacion_bolsa')<>0 THEN RAISE EXCEPTION 'núcleo AD3-000045 no admite extensión B3' USING ERRCODE='55000'; END IF;  esperada:=replace(original,exclusion_pre,exclusion_post); esperada:=replace(esperada,runtime_pre,runtime_post); esperada:=replace(esperada,marca,extension||marca); EXECUTE esperada;  SELECT pg_get_functiondef(f) INTO STRICT actual;  reconstruida:=replace(actual,extension||marca,marca); reconstruida:=replace(reconstruida,runtime_post,runtime_pre); reconstruida:=replace(reconstruida,exclusion_post,exclusion_pre);  IF actual IS DISTINCT FROM esperada OR reconstruida IS DISTINCT FROM original OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM configuracion OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM es_definidora OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN RAISE EXCEPTION 'B3 alteró el núcleo AD3 fuera de su extensión nominal' USING ERRCODE='55000'; END IF; END $nucleo$; LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE; DO $audiencia$ DECLARE definicion text; nueva text; BEGIN  SELECT pg_get_constraintdef(c.oid,true) INTO STRICT definicion FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c';  IF strpos(definicion,'vec_bolsa_llamamientos.contacto_participacion.registrar.v1')<>0 OR strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(definicion,3)<>']))' THEN RAISE EXCEPTION 'audiencias incompatibles para B3' USING ERRCODE='55000'; END IF;  nueva:=left(definicion,length(definicion)-3)||', ''vec_bolsa_llamamientos.contacto_participacion.registrar.v1''::text, ''vec_bolsa_llamamientos.contacto_participacion.consultar.v1''::text]))';  ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;  EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva; END $audiencia$; CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$ DECLARE c jsonb; d jsonb; x record; BEGIN  BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B3 inválido' USING ERRCODE='22023'; END;  IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.registrar.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'material B3 rechazado' USING ERRCODE='42501'; END IF;  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('contacto_participacion_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);  IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'B3 requiere consumo nuevo' USING ERRCODE='42501'; END IF; RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true; END $f$; CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$ DECLARE c jsonb; d jsonb; x record; BEGIN  BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material consulta B3 inválido' USING ERRCODE='22023'; END;  IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.consultar.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.contacto_participacion.consultar' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'consulta_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'consulta B3 rechazada' USING ERRCODE='42501'; END IF;  SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('contacto_participacion_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);  IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'consulta B3 requiere consumo nuevo' USING ERRCODE='42501'; END IF; RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true; END $f$; ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario; ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario; REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC; GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario; GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario; REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC; GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario; REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario; REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario; DO $acl_cerrada$ DECLARE funcion regprocedure := 'vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; a record; BEGIN  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_propietario'::regrole LOOP   IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);   ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;  END LOOP;  IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',funcion,'EXECUTE') OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.privilege_type='EXECUTE' AND (x.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole) OR x.is_grantable AND x.grantee='vec_bolsa_llamamientos_propietario'::regrole)) THEN RAISE EXCEPTION 'ACL B3 incompatible' USING ERRCODE='55000'; END IF; END $acl_cerrada$; DO $acl_consulta$ DECLARE funcion regprocedure := 'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; a record; BEGIN  FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_propietario'::regrole LOOP IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text); ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF; END LOOP;  IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',funcion,'EXECUTE') THEN RAISE EXCEPTION 'ACL consulta B3 incompatible' USING ERRCODE='55000'; END IF; END $acl_consulta$;
-- FIN deploy/postgresql/autorizacion_atestada_v3/migraciones/000046_consumidor_contacto_participacion.up.sql
-- INICIO deploy/postgresql/bolsa_llamamientos/migraciones/000013_contacto_participacion.up.sql
SET LOCAL ROLE vec_bolsa_llamamientos_propietario; SET LOCAL search_path=pg_catalog; SET LOCAL timezone='UTC'; SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000013',0)); DO $b$ DECLARE accion text; ruta text; BEGIN  SELECT pg_get_constraintdef(oid,true) INTO STRICT accion FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_accion_check';  SELECT pg_get_constraintdef(oid,true) INTO STRICT ruta FROM pg_constraint WHERE conrelid='vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento'::regclass AND conname='bitacora_intento_borrador_llamamiento_ruta_clase_check';  IF strpos(accion,'''cambiar_situacion''::text')=0 OR strpos(accion,'''registrar_contacto''::text')<>0 OR strpos(ruta,'''situacion''::text')=0 OR strpos(ruta,'''contactos''::text')<>0 THEN RAISE EXCEPTION 'bitácora incompatible con B3' USING ERRCODE='55000'; END IF; END $b$; ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check; ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_accion_check CHECK(accion IN('crear','consultar','cambiar_situacion','registrar_contacto')); ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento DROP CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check; ALTER TABLE vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento ADD CONSTRAINT bitacora_intento_borrador_llamamiento_ruta_clase_check CHECK(ruta_clase IN('coleccion','detalle','situacion','contactos')); CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.registrar_intento_borrador_llamamiento_v1(p_correlacion_ref text,p_accion text,p_ruta_clase text,p_actor_ref text,p_resultado text) RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$ DECLARE v_ref text; BEGIN  PERFORM vec_bolsa_llamamientos.exigir_runtime_registrador_frontera_borrador_llamamiento();  IF p_correlacion_ref IS NULL OR p_correlacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$' OR p_accion NOT IN('crear','consultar','cambiar_situacion','registrar_contacto') OR p_ruta_clase NOT IN('coleccion','detalle','situacion','contactos') OR (p_actor_ref IS NOT NULL AND p_actor_ref !~ '^per_[A-Za-z0-9_-]{22,128}$') OR p_resultado NOT IN('autenticacion_requerida','acceso_denegado','recurso_no_disponible','infraestructura_no_disponible','resultado_indeterminado') THEN RAISE EXCEPTION 'intento de frontera Bolsa inválido' USING ERRCODE='22023'; END IF;  v_ref:='intento:'||translate(encode(sha256(convert_to(p_correlacion_ref||':'||p_accion||':'||p_ruta_clase||':'||coalesce(p_actor_ref,'')||':'||p_resultado||':'||clock_timestamp()::text,'UTF8')),'hex'),'0123456789','ghijklmnop');  INSERT INTO vec_bolsa_llamamientos.bitacora_intento_borrador_llamamiento(intento_ref,correlacion_ref,accion,ruta_clase,actor_ref,resultado,registrada_en) VALUES(v_ref,p_correlacion_ref,p_accion,p_ruta_clase,p_actor_ref,p_resultado,clock_timestamp()); END $f$; DO $p$ BEGIN  IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL OR to_regclass('vec_bolsa_llamamientos.contacto_participacion') IS NOT NULL THEN RAISE EXCEPTION 'precondición B3 incompatible' USING ERRCODE='55000'; END IF; END $p$; CREATE TABLE vec_bolsa_llamamientos.contacto_participacion(  contacto_ref text PRIMARY KEY,  bolsa_ref text NOT NULL,  participacion_ref text NOT NULL,  llamamiento_ref text,  canal text NOT NULL CHECK(canal IN('telefono','correo','sms','presencial','otro')),  instante timestamptz(6) NOT NULL,  actor text NOT NULL CHECK(actor ~ '^per_[A-Za-z0-9_-]{22,128}$'),  resultado text NOT NULL CHECK(resultado IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro')),  anotacion text NOT NULL CHECK(anotacion=btrim(anotacion) AND octet_length(anotacion) BETWEEN 1 AND 1000 AND anotacion !~* '(^|[^[:alpha:]])(dni|nie|nif|pasaporte|email|teléfono)([^[:alpha:]]|$)' AND strpos(anotacion,'@')=0),  clave_idempotencia text NOT NULL,  recibo_ref text NOT NULL UNIQUE,  registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),  UNIQUE(participacion_ref,clave_idempotencia) ); CREATE INDEX contacto_participacion_participacion_fecha ON vec_bolsa_llamamientos.contacto_participacion(participacion_ref,instante DESC,contacto_ref DESC); CREATE INDEX contacto_participacion_bolsa_fecha ON vec_bolsa_llamamientos.contacto_participacion(bolsa_ref,instante DESC,contacto_ref DESC); ALTER TABLE vec_bolsa_llamamientos.contacto_participacion ENABLE ROW LEVEL SECURITY; ALTER TABLE vec_bolsa_llamamientos.contacto_participacion FORCE ROW LEVEL SECURITY; CREATE POLICY contacto_participacion_solo_propietario ON vec_bolsa_llamamientos.contacto_participacion TO vec_bolsa_llamamientos_propietario USING(current_user='vec_bolsa_llamamientos_propietario') WITH CHECK(current_user='vec_bolsa_llamamientos_propietario'); REVOKE ALL ON vec_bolsa_llamamientos.contacto_participacion FROM PUBLIC; CREATE TRIGGER contacto_participacion_inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.contacto_participacion FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion(); CREATE FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v1(p_contacto_ref text,p_bolsa_ref text,p_participacion_ref text,p_llamamiento_ref text,p_canal text,p_instante timestamptz,p_actor text,p_resultado text,p_anotacion text,p_clave text,p_recibo text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS TABLE(reutilizado boolean,recibo_ref text,contacto_ref text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$ DECLARE consumo record; d jsonb; anterior vec_bolsa_llamamientos.contacto_participacion%ROWTYPE; BEGIN  IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_contacto_ref IS NULL OR p_bolsa_ref IS NULL OR p_participacion_ref IS NULL OR p_canal NOT IN('telefono','correo','sms','presencial','otro') OR p_instante IS NULL OR p_actor !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_resultado NOT IN('contactado','no_contesta','buzon','acepta','rechaza','aplazado','otro') OR p_anotacion IS NULL OR p_anotacion<>btrim(p_anotacion) OR octet_length(p_anotacion) NOT BETWEEN 1 AND 1000 OR p_clave IS NULL OR p_clave<>btrim(p_clave) OR octet_length(p_clave) NOT BETWEEN 1 AND 256 OR p_recibo IS NULL THEN RAISE EXCEPTION 'contacto inválido' USING ERRCODE='22023'; END IF;  IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion_entrada e JOIN vec_bolsa_llamamientos.constitucion c USING(instantanea_ref,version_instantanea) WHERE e.participacion_ref=p_participacion_ref AND c.bolsa_ref=p_bolsa_ref) THEN RAISE EXCEPTION 'participación ajena' USING ERRCODE='23503'; END IF;  IF p_llamamiento_ref IS NOT NULL AND NOT EXISTS(   SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l   JOIN vec_bolsa_llamamientos.integracion_desarrollo o USING(operacion_ref)   WHERE l.llamamiento_ref=p_llamamiento_ref AND l.bolsa_ref=p_bolsa_ref    AND convert_from(o.registro_canonico,'UTF8')::jsonb#>>'{propuesta,participacion_seleccionada_ref}'=p_participacion_ref  ) THEN RAISE EXCEPTION 'llamamiento ajeno' USING ERRCODE='23503'; END IF;  SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);  d:=convert_from(p_decision,'UTF8')::jsonb;  IF consumo.efecto_ref IS DISTINCT FROM p_participacion_ref OR consumo.consumo_nuevo IS NOT TRUE OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'contacto no autorizado' USING ERRCODE='42501'; END IF;  SELECT * INTO anterior FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.participacion_ref=p_participacion_ref AND c.clave_idempotencia=p_clave FOR SHARE;  IF FOUND THEN   IF anterior.bolsa_ref<>p_bolsa_ref OR anterior.llamamiento_ref IS DISTINCT FROM p_llamamiento_ref OR anterior.canal<>p_canal OR anterior.instante<>p_instante OR anterior.actor<>p_actor OR anterior.resultado<>p_resultado OR anterior.anotacion<>p_anotacion THEN RAISE EXCEPTION 'clave idempotente divergente' USING ERRCODE='VBC01'; END IF;   RETURN QUERY SELECT true,anterior.recibo_ref,anterior.contacto_ref; RETURN;  END IF;  INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref,bolsa_ref,participacion_ref,llamamiento_ref,canal,instante,actor,resultado,anotacion,clave_idempotencia,recibo_ref) VALUES(p_contacto_ref,p_bolsa_ref,p_participacion_ref,p_llamamiento_ref,p_canal,p_instante,p_actor,p_resultado,p_anotacion,p_clave,p_recibo);  RETURN QUERY SELECT false,p_recibo,p_contacto_ref; END $f$; CREATE FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v1(p_bolsa_ref text,p_participacion_ref text,p_cursor text,p_limite integer,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS TABLE(contacto_ref text,bolsa_ref text,participacion_ref text,llamamiento_ref text,canal text,instante timestamptz,actor text,resultado text,anotacion text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$ DECLARE consumo record; d jsonb; BEGIN  IF current_user<>'vec_bolsa_llamamientos_propietario' OR p_bolsa_ref IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN RAISE EXCEPTION 'consulta contacto inválida' USING ERRCODE='22023'; END IF;  SELECT * INTO consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_contacto_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);  d:=convert_from(p_decision,'UTF8')::jsonb;  IF consumo.efecto_ref IS DISTINCT FROM coalesce(p_participacion_ref,p_bolsa_ref) OR consumo.consumo_nuevo IS NOT TRUE OR d->>'accion' IS DISTINCT FROM 'bolsa.contacto_participacion.consultar' OR d->>'finalidad' IS DISTINCT FROM 'consulta_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM coalesce(p_participacion_ref,p_bolsa_ref) OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'consulta contacto no autorizada' USING ERRCODE='42501'; END IF;  RETURN QUERY SELECT c.contacto_ref,c.bolsa_ref,c.participacion_ref,c.llamamiento_ref,c.canal,c.instante,c.actor,c.resultado,c.anotacion FROM vec_bolsa_llamamientos.contacto_participacion c WHERE c.bolsa_ref=p_bolsa_ref AND (p_participacion_ref IS NULL OR c.participacion_ref=p_participacion_ref) AND (p_cursor IS NULL OR (c.instante,c.contacto_ref)<(SELECT x.instante,x.contacto_ref FROM vec_bolsa_llamamientos.contacto_participacion x WHERE x.contacto_ref=p_cursor)) ORDER BY c.instante DESC,c.contacto_ref DESC LIMIT p_limite; END $f$; REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC; REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v1(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC; GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_contacto_participacion_v1(text,text,text,text,text,timestamptz,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor; GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.listar_contactos_participacion_v1(text,text,text,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
-- FIN deploy/postgresql/bolsa_llamamientos/migraciones/000013_contacto_participacion.up.sql
:finalizar;
