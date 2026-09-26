\set ON_ERROR_STOP on
-- AD3-88: consumidores nominales de la incorporación acreditada de
-- Contratación temporal (dudas 11 y 12 de RRHH). El único llamante es el
-- propietario de CT (CT 000124):
--  * confirmación de GINPIX por RRHH: perfil y audiencia propios;
--  * confirmación de la incorporación por el centro y su bandeja: perfil
--    propio con la audiencia del material de las peticiones de centro (la
--    misma que usa AD3-23), sin audiencia nueva.
-- Se instala en serie con cualquier otra reescritura del núcleo: toma el
-- cerrojo común antes de leer su preimagen. Sin DOWN tras historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000088',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'confirmacion_ginpix_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmacion_ginpix.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.ginpix.confirmar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'confirmacion_ginpix_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'confirmar_ginpix_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_centro_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.confirmar_alta_atestada.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion.confirmar_centro','contratacion_temporal.incorporacion.consultar_centro'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incorporacion_centro_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_peticion_centro'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario'
                   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_ejecutor' AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-88: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps
 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 -- Los perfiles de CT pasan por el bloque general del núcleo, que exige una
 -- sesión miembro del ejecutor CT. La extensión solo fija operación,
 -- audiencia, recurso y finalidad; ninguna versión previa puede existir.
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'modificacion_tras_nombramiento_ct')=0
    OR strpos(original,'peticion_centro')=0
    OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
    OR strpos(original,'confirmacion_ginpix_ct')<>0
    OR strpos(original,'incorporacion_centro_ct')<>0
 THEN RAISE EXCEPTION 'AD3-88: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-88: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; audiencia text;
 lista text[]:=ARRAY['vec_contratacion_temporal.confirmacion_ginpix.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_contratacion_temporal.modificacion_tras_nombramiento.v1')=0
    OR strpos(d,'vec_contratacion_temporal.confirmar_alta_atestada.v1')=0
 THEN RAISE EXCEPTION 'AD3-88: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3);
 FOREACH audiencia IN ARRAY lista LOOP
  IF strpos(d,quote_literal(audiencia))<>0 THEN
   RAISE EXCEPTION 'AD3-88: audiencia ya registrada' USING ERRCODE='55000'; END IF;
  nueva:=nueva||', '||quote_literal(audiencia)||'::text';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
   DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

-- GINPIX: el recurso es el expediente nombrado con incorporación acreditada.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-88: material de GINPIX inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.ginpix.confirmar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.confirmacion_ginpix.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'confirmacion_ginpix_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'confirmar_ginpix_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-88: confirmación de GINPIX denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'confirmacion_ginpix_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-88: la confirmación de GINPIX requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

-- Centro: confirmación (recurso = expediente) y bandeja (recurso = centro).
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-88: material del centro inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR NOT (c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion.confirmar_centro','contratacion_temporal.incorporacion.consultar_centro']))
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.confirmar_alta_atestada.v1'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'incorporacion_centro_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_peticion_centro'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-88: operación del centro denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'incorporacion_centro_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-88: la operación del centro requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

DO $acl$
DECLARE f regprocedure; x record; permitido oid:='vec_contratacion_temporal_propietario'::regrole::oid;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_consumir_confirmacion_ginpix_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.registrar_y_consumir_incorporacion_centro_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  -- También las ACL por defecto: ningún rol conserva acceso por haber sido
  -- destinatario predeterminado del propietario.
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_propietario',f::text);
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
  THEN RAISE EXCEPTION 'AD3-88: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
  FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
   FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
   IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
      OR (x.grantee=permitido AND x.is_grantable)
   THEN RAISE EXCEPTION 'AD3-88: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
