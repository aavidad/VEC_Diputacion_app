\set ON_ERROR_STOP on
-- AD199: consumidor nominal propio de la descarga de borradores RRHH de
-- Contratación temporal (acción contratacion_temporal.borrador_rrhh.descargar,
-- audiencia vec_contratacion_temporal.borrador_rrhh.descargar.v1) para CT177.
-- Añade el perfil 'descarga_borrador' al núcleo de consultas RRHH
-- (consumir_consulta_rrhh_v3_interna), que ya exige la sesión del consultor
-- RRHH de CT; no toca el núcleo de mutaciones. El recurso es el expediente y la
-- huella de su contexto liga el tipo de borrador, el formato y el SHA256 del
-- archivo, que CT177 recalcula antes de consumir. No concede permisos.
-- Preimagen del núcleo RRHH medida en la copia fría H10-30 con las listas de
-- main del 05/10 (sin cambios desde AD140). El CHECK de audiencias se amplía
-- de forma aditiva y verificada, sin fijar su huella: así no depende del orden
-- de instalación con otros consumidores que también lo amplían.
-- Una sola vez; sin DOWN. Orden: AD199 -> CT177.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000199',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(bytea)') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario' AND NOT rolcanlogin AND NOT rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_consultor_rrhh' AND NOT rolcanlogin AND NOT rolsuper)
 THEN RAISE EXCEPTION 'AD199: PARO clave=preimagen esperado=PG18_nucleo_RRHH_sin_AD199 actual=incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo_rrhh$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;meta jsonb;deps jsonb;h text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
               p_perfil_consulta = 'descarga_borrador'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.borrador_rrhh.descargar.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.borrador_rrhh.descargar'
               AND d ->> 'accion' IS NOT DISTINCT FROM c ->> 'operacion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'expediente_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM c ->> 'efecto_ref'
               AND d ->> 'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c ->> 'huella_efecto_sha256'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$x$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD199: PARO clave=nucleo_rrhh esperado=presente actual=ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'd193f9c128e9036c25e485ecab778c59f10ffbb6cd1ede1dd0a1ac28ae19d683'
 THEN RAISE EXCEPTION 'AD199: PARO clave=nucleo_rrhh_def_SHA actual=% esperado=d193f9c128e9036c25e485ecab778c59f10ffbb6cd1ede1dd0a1ac28ae19d683',h USING ERRCODE='55000'; END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM '3b6a43533fc9311ffed7f8728a473de79eb22260869ad82f78c41f43a6db7397'
 THEN RAISE EXCEPTION 'AD199: PARO clave=nucleo_rrhh_src_SHA actual=% esperado=3b6a43533fc9311ffed7f8728a473de79eb22260869ad82f78c41f43a6db7397',h USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.provolatile='v' AND p.proparallel='u' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 THEN RAISE EXCEPTION 'AD199: PARO clave=nucleo_rrhh_metadatos esperado=propietario_privado actual=incompatible' USING ERRCODE='55000'; END IF;
 IF length(original)-length(replace(original,marca,''))<>length(marca)
 OR strpos(original,'descarga_borrador')<>0
 OR strpos(original,'vec_contratacion_temporal_consultor_rrhh')=0
 THEN RAISE EXCEPTION 'AD199: PARO clave=marcas esperado=marca_unica_sin_perfil actual=incompatible' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
  INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 nueva:=replace(original,marca,extension||marca);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva
 OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD199: PARO clave=postimagen_nucleo_rrhh esperado=extension_minima_metadatos_intactos actual=divergente' USING ERRCODE='55000'; END IF;
END $nucleo_rrhh$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 OR strpos(anterior,'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1')=0
 OR strpos(anterior,'vec_contratacion_temporal.borrador_rrhh.descargar.v1')<>0
 THEN RAISE EXCEPTION 'AD199: PARO clave=CHECK_audiencias esperado=con_detalle_sin_descarga actual=incompatible' USING ERRCODE='55000'; END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo = ''vec_contratacion_temporal.borrador_rrhh.descargar.v1'')';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid,false),'vec_contratacion_temporal.borrador_rrhh.descargar.v1')>0
   AND strpos(pg_get_constraintdef(c.oid,false),'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1')>0)
 THEN RAISE EXCEPTION 'AD199: PARO clave=CHECK_postimagen esperado=audiencia_anadida actual=ausente' USING ERRCODE='55000'; END IF;
END $audiencias$;

-- Fachada de consumo: sólo la ejecuta el propietario de CT desde CT177. El
-- núcleo RRHH comprueba la sesión del consultor, la firma, el gobierno, la
-- revocación y la vigencia, y deja el asiento en la auditoría común.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,auditoria_huella_sha256 text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 -- Una transacción mal configurada es un fallo técnico, no una denegación.
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' THEN
  RAISE EXCEPTION 'AD199: transacción de descarga no admitida' USING ERRCODE='55000';
 END IF;
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR p_capacidad IS NULL OR vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
    OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 THEN
  RAISE EXCEPTION 'AD199: descarga de borrador denegada' USING ERRCODE='42501';
 END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'AD199: material de descarga inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_contratacion_temporal.borrador_rrhh.descargar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'contratacion_temporal.borrador_rrhh.descargar'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'expediente_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitacion_expediente_contratacion_temporal'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS NULL
    OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
  RAISE EXCEPTION 'AD199: descarga de borrador denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(
  'descarga_borrador',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR x.consumo_huella_sha256 IS NULL OR x.consumo_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR x.auditoria_ref IS DISTINCT FROM 'aud_v3_'||substr(x.consumo_huella_sha256,1,32)
    OR x.consumida_en IS NULL OR NOT isfinite(x.consumida_en) THEN
  RAISE EXCEPTION 'AD199: consumo de descarga divergente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,
  x.auditoria_ref,x.auditoria_huella_sha256,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_descarga_borrador_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR NOT has_function_privilege('vec_contratacion_temporal_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_contratacion_temporal_consultor_rrhh',f,'EXECUTE')
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_propietario'::regrole) OR a.privilege_type<>'EXECUTE'
     OR a.is_grantable OR a.grantor<>p.proowner))
 THEN RAISE EXCEPTION 'AD199: PARO clave=ACL esperado=propietario_AD_y_propietario_CT actual=ampliada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
