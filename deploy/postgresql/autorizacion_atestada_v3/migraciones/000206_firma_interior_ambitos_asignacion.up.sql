\set ON_ERROR_STOP on
-- AD206: la huella del recurso de la decisión interior de firma V2 lleva los
-- ámbitos de la asignación de quien actúa (opción B, dirección 05/10).
--
-- El PDP común exige que el recurso tenga exactamente las dimensiones de la
-- asignación. AD170 calculaba la huella sólo con la organización, así que un
-- firmante con asignación de organización y unidad (los cargos de AUT53
-- asignados por lote) nunca obtenía una decisión válida. Ahora:
--   * vec_autorizacion.ambitos_asignacion_firma_ct_v1 lee, con FOR SHARE, la
--     asignación ACTUAL que nombra la decisión (referencia y huella), de ese
--     principal, perfil activo y versión de rol, activa y vigente, y devuelve
--     su organización y, si la tiene, su unidad (un valor cada una; ninguna
--     otra dimensión). Sólo la ejecuta el propietario AD.
--   * huella_recurso_firma_interior_ct_v1 calcula la huella canónica del
--     recurso con esos ámbitos: la organización debe ser la del material y la
--     unidad sólo se admite en la vía VEC y debe ser UnidadFirmanteRef (la del
--     paso del plan). La usan AD170 v3 y, por CT181, CT172 v3 y CT176 v3.
--   * registrar_y_consumir_firma_descriptor_ct_v3_atestada es AD170 v2 con un
--     solo cambio medido: la huella sale de la función anterior.
-- Una asignación sin unidad produce la misma huella que antes. Requiere
-- AD170 con su definición exacta. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000206',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD206: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
     'vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
     IS DISTINCT FROM 'ddc3f92e5bd43572a2ee4d0a169c2d5e8a0b44ad471a96e19c908563c881e738'
 THEN RAISE EXCEPTION 'AD206: PARO clave=AD170 actual=distinto esperado=definicion_medida' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(text,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_actual') IS NULL
 OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
 THEN RAISE EXCEPTION 'AD206: PARO clave=preimagen actual=incompatible esperado=AD170_sin_AD206' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
-- Ámbitos de la asignación actual que nombra la decisión. NULL si no es la
-- actual, no coincide su huella, principal, perfil o versión de rol, no está
-- activa y vigente, o sus ámbitos no son organización y, como mucho, unidad,
-- con un valor cada una. Una lectura por clave única; no escribe nada.
CREATE FUNCTION vec_autorizacion.ambitos_asignacion_firma_ct_v1(
 p_asignacion_ref text,p_asignacion_huella text,p_principal text,p_perfil text,p_version_rol text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE a record;n integer;n_validas integer;org text;uni text;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR p_asignacion_ref IS NULL OR p_asignacion_huella IS NULL OR p_principal IS NULL
  OR p_perfil IS NULL OR p_version_rol IS NULL THEN RETURN NULL; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE p.asignacion_ref=p_asignacion_ref FOR SHARE OF p,x;
 IF NOT FOUND OR a.huella_sha256 IS DISTINCT FROM p_asignacion_huella
  OR a.principal_id IS DISTINCT FROM p_principal OR a.perfil_activo_ref IS DISTINCT FROM p_perfil
  OR a.version_rol_ref IS DISTINCT FROM p_version_rol OR a.documento->>'estado' IS DISTINCT FROM 'activa'
  OR (pg_catalog.clock_timestamp()>=(a.documento->>'vigente_desde')::timestamptz
      AND pg_catalog.clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz) IS NOT TRUE
  OR pg_catalog.jsonb_typeof(a.documento->'ambitos') IS DISTINCT FROM 'array' THEN RETURN NULL; END IF;
 SELECT count(*),
  count(*) FILTER (WHERE e->>'clave' IN('organizacion_ref','unidad_ref')
   AND pg_catalog.jsonb_typeof(e->'valores')='array' AND pg_catalog.jsonb_array_length(e->'valores')=1
   AND pg_catalog.jsonb_typeof(e->'valores'->0)='string'),
  min(e->'valores'->>0) FILTER (WHERE e->>'clave'='organizacion_ref'),
  min(e->'valores'->>0) FILTER (WHERE e->>'clave'='unidad_ref')
 INTO n,n_validas,org,uni FROM pg_catalog.jsonb_array_elements(a.documento->'ambitos') e;
 IF n<>n_validas OR n NOT IN(1,2) OR org IS NULL OR (n=2 AND uni IS NULL) THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object('organizacion_ref',org,'unidad_ref',uni);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text) TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- Huella canónica del recurso interior, idéntica a la del PDP en Go: claves
-- ordenadas, sin espacios. Los valores pasan expresiones sin «<>&» ni
-- escapes, así que la representación JSON coincide byte a byte.
CREATE FUNCTION vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(
 p_solicitud text,p_descriptor bytea,p_decision bytea)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s jsonb;d jsonb;amb jsonb;org text;uni text;ambitos text;
BEGIN
 IF p_solicitud IS NULL OR p_descriptor IS NULL OR p_decision IS NULL
  OR pg_catalog.octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
  OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN
  RAISE EXCEPTION 'AD206 material inválido' USING ERRCODE='22023'; END IF;
 s:=p_solicitud::jsonb; d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 amb:=vec_autorizacion.ambitos_asignacion_firma_ct_v1(d->>'asignacion_ref',d->>'asignacion_huella_sha256',
  d->>'principal_id',d->>'perfil_activo_ref',d->>'version_rol_ref');
 org:=amb->>'organizacion_ref'; uni:=amb->>'unidad_ref';
 IF amb IS NULL OR org IS DISTINCT FROM s->>'OrganizacionRef'
  OR (org ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
  OR (uni IS NOT NULL AND (s->>'Via' IS DISTINCT FROM 'certificado_vec'
      OR uni IS DISTINCT FROM s->>'UnidadFirmanteRef'
      OR (uni ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE)) THEN
  RAISE EXCEPTION 'AD206 ámbitos de la asignación no admitidos' USING ERRCODE='42501'; END IF;
 ambitos:='"organizacion_ref":"'||org||'"'||CASE WHEN uni IS NULL THEN '' ELSE ',"unidad_ref":"'||uni||'"' END;
 RETURN pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{"ambitos":{'||ambitos||
  '},"atributos":{"descriptor_firma_sha256":"'||pg_catalog.encode(pg_catalog.sha256(p_descriptor),'hex')||
  '","material_sha256":"'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_solicitud,'UTF8')),'hex')||'"}}','UTF8')),'hex');
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'AD206 material inválido' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(text,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(text,bytea,bytea) TO vec_contratacion_temporal_propietario;

-- AD170 v3: la definición medida con dos sustituciones exactas (nombre y
-- huella). Se comprueba que cada marca aparece una vez.
DO $v3$
DECLARE original text;nueva text;
 viejo_nombre text:='registrar_y_consumir_firma_descriptor_ct_v2_atestada(';
 nuevo_nombre text:='registrar_y_consumir_firma_descriptor_ct_v3_atestada(';
 viejo_ctx text:=E' contexto_h:=encode(sha256(convert_to(''{"ambitos":{"organizacion_ref":"''||(s->>''OrganizacionRef'')||\n  ''"},"atributos":{"descriptor_firma_sha256":"''||descriptor_h||''","material_sha256":"''||material_h||''"}}'',''UTF8'')),''hex'');';
 nuevo_ctx text:=' contexto_h:=vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(p_solicitud,p_descriptor,p_decision);';
BEGIN
 original:=pg_catalog.pg_get_functiondef('vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure);
 IF (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,viejo_nombre,'')))/pg_catalog.length(viejo_nombre)<>1
 OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,viejo_ctx,'')))/pg_catalog.length(viejo_ctx)<>1
 THEN RAISE EXCEPTION 'AD206: PARO clave=marcas_AD170 actual=no_unicas esperado=una_vez' USING ERRCODE='55000'; END IF;
 nueva:=pg_catalog.replace(pg_catalog.replace(original,viejo_nombre,nuevo_nombre),viejo_ctx,nuevo_ctx);
 EXECUTE nueva;
END $v3$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
RESET ROLE;

DO $post$
DECLARE v2 regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v3 regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_firma_descriptor_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 h regprocedure:='vec_autorizacion_atestada_v3.huella_recurso_firma_interior_ct_v1(text,bytea,bytea)'::regprocedure;
 a regprocedure:='vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text)'::regprocedure;
BEGIN
 -- v3 y v2 comparten propietario, seguridad y configuración; v3 sólo para el propietario CT.
 IF (SELECT row(proowner,prosecdef,proconfig,provolatile) FROM pg_catalog.pg_proc WHERE oid=v3)
    IS DISTINCT FROM (SELECT row(proowner,prosecdef,proconfig,provolatile) FROM pg_catalog.pg_proc WHERE oid=v2)
 OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(v3),'huella_recurso_firma_interior_ct_v1(p_solicitud,p_descriptor,p_decision)')=0
 OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=v3) IS DISTINCT FROM
    ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
          'vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
 OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=h) IS DISTINCT FROM
    ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
          'vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
 OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=a) IS DISTINCT FROM
    ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario',
          'vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario']::aclitem[]
 OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=a)<>'vec_autorizacion_propietario'::regrole
 THEN RAISE EXCEPTION 'AD206: PARO clave=postimagen actual=divergente esperado=v3_ACL_y_funciones' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
