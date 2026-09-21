\set ON_ERROR_STOP on
-- AD3-000044. Extiende estructuralmente el núcleo V3 posterior a AD3-32 y
-- AD3-43. No usa autohuellas: preserva las guardas previas y comprueba las
-- fachadas antecedentes con su firma nominal.
BEGIN;
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
COMMIT;
