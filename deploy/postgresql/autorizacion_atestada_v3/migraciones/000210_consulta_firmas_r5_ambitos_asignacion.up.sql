\set ON_ERROR_STOP on
-- AD210: huella del recurso de la consulta y de la recuperación R5 V2 con los
-- ámbitos de la asignación de quien consulta (opción B; para la unidad, opción
-- A del 06/10: la del paso del plan, que CT186 comprueba con CC10).
--
-- huella_recurso_consulta_firmas_r5_ct_v1 lee con
-- vec_autorizacion.ambitos_asignacion_firma_ct_v1 (AD206) la asignación que
-- nombra la decisión y calcula la huella canónica del recurso (atributo
-- material_sha256) con su organización y, si la tiene, su unidad. La
-- organización debe ser la del material. La unidad de la asignación debe ser
-- exactamente UnidadRef del material (ausente si la asignación no la tiene) y
-- sólo se admite en la vía VEC. Una asignación sin unidad y un material sin
-- UnidadRef producen la misma huella que antes. Requiere AD206. Sólo la
-- ejecuta el propietario CT. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000210',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD210: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
     'vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text)','EXECUTE')
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(text,bytea)') IS NOT NULL
 OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
 THEN RAISE EXCEPTION 'AD210: PARO clave=preimagen actual=incompatible esperado=AD206_sin_AD210' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- Huella canónica del recurso, idéntica a la del PDP en Go: claves ordenadas,
-- sin espacios. Los valores pasan expresiones sin «<>&» ni escapes, así que
-- la representación JSON coincide byte a byte.
CREATE FUNCTION vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(p_solicitud text,p_decision bytea)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s jsonb;d jsonb;amb jsonb;org text;uni text;ambitos text;
BEGIN
 IF p_solicitud IS NULL OR p_decision IS NULL
  OR pg_catalog.octet_length(p_solicitud) NOT BETWEEN 2 AND 4096
  OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN
  RAISE EXCEPTION 'AD210 material inválido' USING ERRCODE='22023'; END IF;
 s:=p_solicitud::jsonb; d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 amb:=vec_autorizacion.ambitos_asignacion_firma_ct_v1(d->>'asignacion_ref',d->>'asignacion_huella_sha256',
  d->>'principal_id',d->>'perfil_activo_ref',d->>'version_rol_ref');
 org:=amb->>'organizacion_ref'; uni:=amb->>'unidad_ref';
 IF amb IS NULL OR org IS DISTINCT FROM s->>'OrganizacionRef'
  OR (org ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
  -- La unidad del material es exactamente la de la asignación (o ninguna).
  OR (CASE WHEN uni IS NULL THEN s->'UnidadRef' IS DISTINCT FROM 'null'::jsonb
      ELSE pg_catalog.jsonb_typeof(s->'UnidadRef') IS DISTINCT FROM 'string' OR uni IS DISTINCT FROM s->>'UnidadRef'
       OR s->>'Via' IS DISTINCT FROM 'certificado_vec'
       OR (uni ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE END) THEN
  RAISE EXCEPTION 'AD210 ámbitos de la asignación no admitidos' USING ERRCODE='42501'; END IF;
 ambitos:='"organizacion_ref":"'||org||'"'||CASE WHEN uni IS NULL THEN '' ELSE ',"unidad_ref":"'||uni||'"' END;
 RETURN pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{"ambitos":{'||ambitos||
  '},"atributos":{"material_sha256":"'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_solicitud,'UTF8')),'hex')||'"}}','UTF8')),'hex');
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'AD210 material inválido' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(text,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(text,bytea) TO vec_contratacion_temporal_propietario;
RESET ROLE;

DO $post$
DECLARE h regprocedure:='vec_autorizacion_atestada_v3.huella_recurso_consulta_firmas_r5_ct_v1(text,bytea)'::regprocedure;
BEGIN
 IF (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=h) IS DISTINCT FROM
    ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
          'vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
 OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=h)<>'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=h) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD210: PARO clave=postimagen actual=divergente esperado=huella_para_propietario_CT' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
