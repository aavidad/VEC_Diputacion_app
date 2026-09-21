\set ON_ERROR_STOP on
-- AD3-000046. Extensión nominal B3 sobre la estructura de AD3-000045;
-- conserva las guardas anteriores sin exigir una autohuella del núcleo.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000046',0));

DO $precondicion$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_creacion_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_borrador_llamamiento_interno_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_resultado_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_mi_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef AND prokind='f' AND provolatile='v' AND pg_get_function_identity_arguments(oid)='p_perfil_mutacion text, p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea' AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']) THEN
  RAISE EXCEPTION 'estructura AD3-000044 incompatible para B3' USING ERRCODE='55000';
 END IF;
END $precondicion$;

DO $nucleo$
DECLARE f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; esperada text; actual text; reconstruida text; metadata jsonb; deps jsonb; acl aclitem[]; propietario oid; configuracion text[]; es_definidora boolean;
 marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 exclusion_pre text := $x$               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               AND p_perfil_mutacion IS DISTINCT FROM 'situacion_participacion_bolsa'$x$;
 exclusion_post text := exclusion_pre||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''contacto_participacion_bolsa''';
 runtime_pre text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa')$x$;
 runtime_post text := $x$               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_borrador_llamamiento_interno_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa')$x$;
 extension text := $p$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'contacto_participacion_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.registrar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'
 AND d->>'accion' IS NOT DISTINCT FROM 'bolsa.contacto_participacion.registrar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_contactos_participacion')
$p$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,metadata,acl,propietario,configuracion,es_definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF length(original)-length(replace(original,marca,''))<>length(marca) OR length(original)-length(replace(original,exclusion_pre,''))<>length(exclusion_pre) OR length(original)-length(replace(original,runtime_pre,''))<>length(runtime_pre) OR strpos(original,'creacion_borrador_llamamiento_interno_bolsa')=0 OR strpos(original,'consulta_participaciones_propias_bolsa')=0 OR strpos(original,'despacho_correo_llamamiento_ct')=0 OR strpos(original,'situacion_participacion_bolsa')=0 OR strpos(original,'contacto_participacion_bolsa')<>0 THEN RAISE EXCEPTION 'núcleo AD3-000045 no admite extensión B3' USING ERRCODE='55000'; END IF;
 esperada:=replace(original,exclusion_pre,exclusion_post); esperada:=replace(esperada,runtime_pre,runtime_post); esperada:=replace(esperada,marca,extension||marca); EXECUTE esperada;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 reconstruida:=replace(actual,extension||marca,marca); reconstruida:=replace(reconstruida,runtime_post,runtime_pre); reconstruida:=replace(reconstruida,exclusion_post,exclusion_pre);
 IF actual IS DISTINCT FROM esperada OR reconstruida IS DISTINCT FROM original OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM configuracion OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM es_definidora OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN RAISE EXCEPTION 'B3 alteró el núcleo AD3 fuera de su extensión nominal' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$ DECLARE definicion text; nueva text; BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT definicion FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c';
 IF strpos(definicion,'vec_bolsa_llamamientos.contacto_participacion.registrar.v1')<>0 OR strpos(definicion,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(definicion,3)<>']))' THEN RAISE EXCEPTION 'audiencias incompatibles para B3' USING ERRCODE='55000'; END IF;
 nueva:=left(definicion,length(definicion)-3)||', ''vec_bolsa_llamamientos.contacto_participacion.registrar.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material B3 inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.contacto_participacion.registrar.v1' OR c->>'operacion' IS DISTINCT FROM 'bolsa.contacto_participacion.registrar' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'participacion_bolsa' OR d->>'finalidad' IS DISTINCT FROM 'gestion_contactos_participacion' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'material B3 rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('contacto_participacion_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'B3 requiere consumo nuevo' USING ERRCODE='42501'; END IF; RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true; END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
REVOKE GRANT OPTION FOR EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_bolsa_llamamientos_propietario;
DO $acl_cerrada$
DECLARE funcion regprocedure := 'vec_autorizacion_atestada_v3.registrar_y_consumir_contacto_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.grantee<>p.proowner AND x.grantee<>'vec_bolsa_llamamientos_propietario'::regrole LOOP
  IF a.grantee=0 THEN EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',funcion::text);
  ELSE EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',funcion::text,pg_get_userbyid(a.grantee)); END IF;
 END LOOP;
 IF NOT has_function_privilege('vec_bolsa_llamamientos_propietario',funcion,'EXECUTE') OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=funcion AND x.privilege_type='EXECUTE' AND (x.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole) OR x.is_grantable AND x.grantee='vec_bolsa_llamamientos_propietario'::regrole)) THEN RAISE EXCEPTION 'ACL B3 incompatible' USING ERRCODE='55000'; END IF;
END $acl_cerrada$;
COMMIT;
