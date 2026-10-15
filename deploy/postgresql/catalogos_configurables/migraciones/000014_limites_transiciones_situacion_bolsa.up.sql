\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_catalogos_configurables:migracion:000014',0));

DO $pre$
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
    OR to_regclass('vec_catalogos_configurables.publicacion') IS NULL
    OR to_regclass('vec_catalogos_configurables.entrada_publicada') IS NULL
    OR to_regclass('vec_catalogos_configurables.categoria_control') IS NULL
    OR to_regprocedure('vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(integer,text)') IS NOT NULL
    OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
 THEN RAISE EXCEPTION 'CC14: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El acto de activación es la publicación ordinaria del catálogo común,
-- con dos aprobaciones, versión, huella, recibo e historia. Esta lectura no
-- publica, activa ni modifica contenido: solo acepta la entrada vigente
-- y habilitada del perfil normativo fijo de Bolsa.
CREATE FUNCTION vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(
 p_version integer,p_huella text) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_entrada jsonb; v_pares text[]; v_total integer;
BEGIN
 IF p_version IS NULL OR p_version<1 OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'CC14: perfil normativo inválido' USING ERRCODE='22023';
 END IF;
 SELECT e.definicion INTO v_entrada
 FROM vec_catalogos_configurables.publicacion p
 JOIN vec_catalogos_configurables.entrada_publicada e
   ON e.catalogo_id=p.catalogo_id AND e.version=p.version AND e.huella_sha256=p.huella_sha256
 JOIN vec_catalogos_configurables.categoria_control c
   ON c.categoria_id=e.categoria_id AND c.catalogo_id=e.catalogo_id
   AND c.version=e.version AND c.huella_sha256=e.huella_sha256
 WHERE p.catalogo_id='bolsa.transiciones_situacion' AND p.version=p_version
   AND p.huella_sha256=p_huella AND e.categoria_id='bolsa.transiciones_situacion.limites'
   AND c.estado='habilitada';
 IF v_entrada IS NULL OR coalesce(v_entrada->>'unidad_ref','') !~ '^[A-Za-z0-9:_-]{3,160}$'
    OR coalesce(v_entrada->>'ambito_ref','') !~ '^[A-Za-z0-9:_-]{3,160}$'
    OR jsonb_typeof(v_entrada->'transiciones') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'CC14: perfil normativo no publicado o inválido' USING ERRCODE='55000';
 END IF;
 IF jsonb_array_length(v_entrada->'transiciones') NOT BETWEEN 1 AND 56 THEN
  RAISE EXCEPTION 'CC14: perfil normativo no publicado o inválido' USING ERRCODE='55000';
 END IF;
 SELECT count(*),array_agg(valor ORDER BY ordinal) INTO v_total,v_pares
 FROM jsonb_array_elements_text(v_entrada->'transiciones') WITH ORDINALITY AS t(valor,ordinal);
 IF v_total IS DISTINCT FROM jsonb_array_length(v_entrada->'transiciones')
    OR EXISTS (SELECT 1 FROM unnest(v_pares) x WHERE x IS NULL OR octet_length(x)>128) THEN
  RAISE EXCEPTION 'CC14: límites normativos inválidos' USING ERRCODE='22023';
 END IF;
 RETURN jsonb_build_object('catalogo_ref','bolsa.transiciones_situacion:'||p_version,
  'catalogo_version',p_version,'catalogo_sha256',p_huella,
  'unidad_ref',v_entrada->>'unidad_ref','ambito_ref',v_entrada->>'ambito_ref',
  'transiciones',to_jsonb(v_pares));
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(integer,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_catalogos_configurables TO vec_bolsa_llamamientos_propietario;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(integer,text)
 TO vec_bolsa_llamamientos_propietario;
DO $acl$
DECLARE f regprocedure:='vec_catalogos_configurables.leer_limites_transiciones_situacion_bolsa_v1(integer,text)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_catalogos_configurables_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_bolsa_llamamientos_propietario','vec_catalogos_configurables','USAGE')
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                WHERE p.oid=f AND a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole::oid))
 THEN RAISE EXCEPTION 'CC14: ACL de lectura incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
