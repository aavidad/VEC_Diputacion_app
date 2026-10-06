\set ON_ERROR_STOP on
-- AD209: la huella del recurso de la decisión EXTERIOR de firma V2 (la del
-- plan publicado, AD177) lleva los ámbitos de la asignación de quien actúa,
-- igual que la interior desde AD206 (opción B, dirección 05/10).
--
-- Con AD206 un firmante cuya asignación tiene organización y unidad obtiene la
-- decisión interior, pero la exterior seguía con la huella sólo de
-- organización: el PDP común exige exactamente las dimensiones de la
-- asignación, así que nunca obtenía la exterior. Ahora:
--   * huella_recurso_plan_firma_ct_v1 calcula la huella canónica del recurso
--     exterior (atributos material_sha256 y plan_firma_sha256) con los ámbitos
--     que devuelve vec_autorizacion.ambitos_asignacion_firma_ct_v1 (AD206) para
--     la asignación que nombra la decisión. Mismas reglas que la interior: la
--     organización es la del material y la unidad sólo se admite en la vía VEC
--     y debe ser UnidadFirmanteRef (la del paso del plan).
--   * consumir_plan_firma_ct_v3_atestada es AD177 v2 con un solo cambio
--     medido: la huella sale de la función anterior.
-- Una asignación sin unidad produce la misma huella que antes. Requiere AD177
-- con su definición exacta y AD206. CT185 la usa y cierra la v2 para el
-- propietario CT. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000209',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD209: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
     'vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'UTF8')),'hex')
     IS DISTINCT FROM '1844f8de343db8090f39b4317607f8ceb24622f26c979a7edb58100383423f96'
 THEN RAISE EXCEPTION 'AD209: PARO clave=AD177 actual=distinto esperado=definicion_medida' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(text,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_atestada_v3_propietario',
     'vec_autorizacion.ambitos_asignacion_firma_ct_v1(text,text,text,text,text)','EXECUTE')
 OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
 THEN RAISE EXCEPTION 'AD209: PARO clave=preimagen actual=incompatible esperado=AD177_AD206_sin_AD209' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- Huella canónica del recurso exterior, idéntica a la del PDP en Go: claves
-- ordenadas, sin espacios. Los valores pasan expresiones sin «<>&» ni
-- escapes, así que la representación JSON coincide byte a byte.
CREATE FUNCTION vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(
 p_solicitud text,p_envoltorio bytea,p_decision bytea)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s jsonb;d jsonb;amb jsonb;org text;uni text;ambitos text;
BEGIN
 IF p_solicitud IS NULL OR p_envoltorio IS NULL OR p_decision IS NULL
  OR pg_catalog.octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
  OR pg_catalog.octet_length(p_envoltorio) NOT BETWEEN 2 AND 65536
  OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN
  RAISE EXCEPTION 'AD209 material inválido' USING ERRCODE='22023'; END IF;
 s:=p_solicitud::jsonb; d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 amb:=vec_autorizacion.ambitos_asignacion_firma_ct_v1(d->>'asignacion_ref',d->>'asignacion_huella_sha256',
  d->>'principal_id',d->>'perfil_activo_ref',d->>'version_rol_ref');
 org:=amb->>'organizacion_ref'; uni:=amb->>'unidad_ref';
 IF amb IS NULL OR org IS DISTINCT FROM s->>'OrganizacionRef'
  OR (org ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
  OR (uni IS NOT NULL AND (s->>'Via' IS DISTINCT FROM 'certificado_vec'
      OR uni IS DISTINCT FROM s->>'UnidadFirmanteRef'
      OR (uni ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE)) THEN
  RAISE EXCEPTION 'AD209 ámbitos de la asignación no admitidos' USING ERRCODE='42501'; END IF;
 ambitos:='"organizacion_ref":"'||org||'"'||CASE WHEN uni IS NULL THEN '' ELSE ',"unidad_ref":"'||uni||'"' END;
 RETURN pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('{"ambitos":{'||ambitos||
  '},"atributos":{"material_sha256":"'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_solicitud,'UTF8')),'hex')||
  '","plan_firma_sha256":"'||pg_catalog.encode(pg_catalog.sha256(p_envoltorio),'hex')||'"}}','UTF8')),'hex');
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'AD209 material inválido' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(text,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(text,bytea,bytea) TO vec_contratacion_temporal_propietario;

-- AD177 v3: la definición medida con dos sustituciones exactas (nombre y
-- huella). Se comprueba que cada marca aparece una vez.
DO $v3$
DECLARE original text;nueva text;
 viejo_nombre text:='consumir_plan_firma_ct_v2_atestada(';
 nuevo_nombre text:='consumir_plan_firma_ct_v3_atestada(';
 viejo_ctx text:=E' contexto_sha:=encode(sha256(convert_to(''{"ambitos":{"organizacion_ref":"''||(s->>''OrganizacionRef'')||\n  ''"},"atributos":{"material_sha256":"''||material_sha||''","plan_firma_sha256":"''||plan_sha||''"}}'',''UTF8'')),''hex'');';
 nuevo_ctx text:=' contexto_sha:=vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(p_solicitud,p_envoltorio,p_decision);';
BEGIN
 original:=pg_catalog.pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure);
 IF (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,viejo_nombre,'')))/pg_catalog.length(viejo_nombre)<>1
 OR (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,viejo_ctx,'')))/pg_catalog.length(viejo_ctx)<>1
 THEN RAISE EXCEPTION 'AD209: PARO clave=marcas_AD177 actual=no_unicas esperado=una_vez' USING ERRCODE='55000'; END IF;
 nueva:=pg_catalog.replace(pg_catalog.replace(original,viejo_nombre,nuevo_nombre),viejo_ctx,nuevo_ctx);
 EXECUTE nueva;
END $v3$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_propietario;
RESET ROLE;

DO $post$
DECLARE v2 regprocedure:='vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v3 regprocedure:='vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v3_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 h regprocedure:='vec_autorizacion_atestada_v3.huella_recurso_plan_firma_ct_v1(text,bytea,bytea)'::regprocedure;
 acl aclitem[]:=ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
  'vec_contratacion_temporal_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[];
BEGIN
 -- v3 y v2 comparten propietario, seguridad y configuración; v3 y la huella
 -- sólo para el propietario AD y el propietario CT.
 IF (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=v3)
    IS DISTINCT FROM (SELECT row(proowner,prosecdef,proconfig,provolatile,proparallel) FROM pg_catalog.pg_proc WHERE oid=v2)
 OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(v3),'huella_recurso_plan_firma_ct_v1(p_solicitud,p_envoltorio,p_decision)')=0
 OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=v3) IS DISTINCT FROM acl
 OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=h) IS DISTINCT FROM acl
 OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=h)<>'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=h) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD209: PARO clave=postimagen actual=divergente esperado=v3_ACL_y_funciones' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
