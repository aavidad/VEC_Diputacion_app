\set ON_ERROR_STOP on
-- AD155: dos decisiones nominales. El integrante presenta documentación;
-- RRHH puede rechazarla con recibo sin cambiar la situación de la bolsa.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000155',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $nucleo$
DECLARE
 f regprocedure := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text := $x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'portal_candidato_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.presentar_solicitud_documental'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participaciones_candidato'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_participaciones_propias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.solicitudes_documentales.resolver'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.solicitudes_documentales.resolver.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'solicitud_documental_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_situacion_participacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'situacion_participacion_bolsa'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.solicitudes_documentales.consultar_rrhh'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_llamamientos.solicitudes_documentales.consultar_rrhh.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'participacion_bolsa'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_situacion_participacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
 meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
BEGIN
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'PARO clave=AD155.preimagen, actual=%/%/%/%, esperado=true/true/true/false',
   current_user='vec_autorizacion_atestada_v3_propietario',
   pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
   pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL,
   pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
   USING ERRCODE='55000';
 END IF;
 SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc p WHERE p.oid=f;
 SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=f;
 IF propietario IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
    OR NOT definidora
    -- La principal conserva search_path con pg_temp desde AD139/140; se admite y se preserva.
    OR (config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
        AND config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,''))<>pg_catalog.length(marca)
    OR pg_catalog.strpos(original,'portal_candidato_bolsa')=0
    OR pg_catalog.strpos(original,'situacion_participacion_bolsa')=0
    OR pg_catalog.strpos(original,'presentar_solicitud_documental')<>0 THEN
  RAISE EXCEPTION 'PARO clave=AD155.nucleo, actual=%/%/%/%, esperado=true/true/1/false',
   propietario='vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole,
   definidora,
   (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,marca,'')))/pg_catalog.length(marca),
   pg_catalog.strpos(original,'presentar_solicitud_documental')<>0 USING ERRCODE='55000';
 END IF;
 nuevo:=pg_catalog.replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR pg_catalog.replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::pg_catalog.regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'PARO clave=AD155.definicion_y_metadata, actual=%, esperado=%',
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(actual||
     coalesce((SELECT proacl::text FROM pg_catalog.pg_proc WHERE oid=f),'')||
     coalesce((SELECT proconfig::text FROM pg_catalog.pg_proc WHERE oid=f),''),'UTF8')),'hex'),
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(nuevo||coalesce(acl::text,'')||coalesce(config::text,''),'UTF8')),'hex')
   USING ERRCODE='55000';
 END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; a text;
BEGIN
 SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::pg_catalog.regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF pg_catalog.strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR pg_catalog.right(d,3)<>']))' THEN
  RAISE EXCEPTION 'PARO clave=AD155.forma_audiencias, actual=%/%, esperado=true/true',
   pg_catalog.strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')=1,
   pg_catalog.right(d,3)=']))' USING ERRCODE='55000';
 END IF;
 FOREACH a IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1',
  'vec_bolsa_llamamientos.solicitudes_documentales.resolver.v1',
  'vec_bolsa_llamamientos.solicitudes_documentales.consultar_rrhh.v1'
 ] LOOP
  IF pg_catalog.strpos(d,pg_catalog.quote_literal(a))<>0 THEN
   RAISE EXCEPTION 'PARO clave=audiencia_AD155_%, actual=presente, esperado=ausente',a USING ERRCODE='55000'; END IF;
  d:=pg_catalog.left(d,pg_catalog.length(d)-3)||', '||pg_catalog.quote_literal(a)||'::text]))';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||d;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(
 p_operacion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; perfil text; audiencia text; tipo text; finalidad text;
BEGIN
 BEGIN c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb; d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD155 material inválido' USING ERRCODE='22023'; END;
 IF p_operacion='bolsa.participaciones_propias.presentar_solicitud_documental' THEN
  perfil:='portal_candidato_bolsa'; audiencia:='vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1';
  tipo:='participaciones_candidato'; finalidad:='gestion_participaciones_propias';
  IF coalesce(d->>'recurso_ref','') !~ '^mi-bolsa:can_[A-Za-z0-9_-]{22,128}$' THEN RAISE EXCEPTION 'AD155 recurso ajeno' USING ERRCODE='42501'; END IF;
 ELSIF p_operacion='bolsa.solicitudes_documentales.resolver' THEN
  perfil:='situacion_participacion_bolsa'; audiencia:='vec_bolsa_llamamientos.solicitudes_documentales.resolver.v1';
  tipo:='solicitud_documental_bolsa'; finalidad:='gestion_situacion_participacion';
  IF coalesce(d->>'recurso_ref','') !~ '^solicitud-documental:[0-9a-f]{64}$' THEN RAISE EXCEPTION 'AD155 recurso ajeno' USING ERRCODE='42501'; END IF;
 ELSIF p_operacion='bolsa.solicitudes_documentales.consultar_rrhh' THEN
  perfil:='situacion_participacion_bolsa'; audiencia:='vec_bolsa_llamamientos.solicitudes_documentales.consultar_rrhh.v1';
  tipo:='participacion_bolsa'; finalidad:='gestion_situacion_participacion';
  IF coalesce(d->>'recurso_ref','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN RAISE EXCEPTION 'AD155 recurso ajeno' USING ERRCODE='42501'; END IF;
 ELSE RAISE EXCEPTION 'AD155 operación ajena' USING ERRCODE='42501'; END IF;
 IF c->>'operacion' IS DISTINCT FROM p_operacion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR d->>'accion' IS DISTINCT FROM p_operacion OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM tipo OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD155 decisión ajena' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD155 consumo anterior' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

DO $acl$
DECLARE f pg_catalog.regprocedure := 'vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
 x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_catalog.pg_proc p
 CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
 WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(x.grantee)) END);
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
  TO vec_bolsa_llamamientos_propietario;
 IF NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE') THEN
  RAISE EXCEPTION 'PARO clave=AD155.ejecutor_Bolsa, actual=false, esperado=true' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
