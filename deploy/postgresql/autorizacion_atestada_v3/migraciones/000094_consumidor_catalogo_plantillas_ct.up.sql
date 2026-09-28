\set ON_ERROR_STOP on
-- AD3-94: consumo nominal del catálogo de plantillas de Contratación.
-- La decisión atestada sigue siendo la única autoridad; el perfil nuevo
-- limita exactamente audiencia, operación, recurso, finalidad y módulo.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000094',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; propietario oid; config text[]; acl aclitem[];
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'catalogo_plantillas_ct'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas.v1'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.plantillas_documentos.consultar','contratacion_temporal.plantillas_documentos.editar','contratacion_temporal.plantillas_documentos.publicar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_plantillas_contratacion_temporal'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestionar_catalogo_plantillas_contratacion_temporal'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM CASE WHEN c->>'operacion'='contratacion_temporal.plantillas_documentos.consultar'
      THEN '["borrador","editor_de_esta_version","publicado"]'::jsonb ELSE '["catalogo","recibo"]'::jsonb END
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_ejecutor' AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-94: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.proowner,p.proconfig,p.proacl INTO STRICT original,propietario,config,acl FROM pg_proc p WHERE p.oid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
    OR strpos(original,'catalogo_plantillas_ct')<>0
 THEN RAISE EXCEPTION 'AD3-94: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
 THEN RAISE EXCEPTION 'AD3-94: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_contratacion_temporal.confirmar_alta_atestada.v1')=0
    OR strpos(d,'''vec_contratacion_temporal.catalogo_plantillas.v1''')<>0
 THEN RAISE EXCEPTION 'AD3-94: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', ''vec_contratacion_temporal.catalogo_plantillas.v1''::text';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-94: material inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.catalogo_plantillas.v1'
    OR c->>'operacion' <> ALL (ARRAY['contratacion_temporal.plantillas_documentos.consultar','contratacion_temporal.plantillas_documentos.editar','contratacion_temporal.plantillas_documentos.publicar'])
    OR c->>'operacion' IS NULL
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_plantillas_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_catalogo_plantillas_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR c->>'efecto_ref' IS DISTINCT FROM d->>'recurso_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM CASE WHEN c->>'operacion'='contratacion_temporal.plantillas_documentos.consultar'
         THEN '["borrador","editor_de_esta_version","publicado"]'::jsonb ELSE '["catalogo","recibo"]'::jsonb END
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-94: decisión de plantillas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'catalogo_plantillas_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-94: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
  LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
  WHERE p.oid=f AND x.grantee<>p.proowner AND x.grantee<>'vec_contratacion_temporal_propietario'::regrole LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
 END LOOP;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
               WHERE p.oid=f AND (x.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole)
                                  OR x.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'AD3-94: ACL de fachada incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
