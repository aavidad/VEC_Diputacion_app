\set ON_ERROR_STOP on
-- CC10: predicado de sólo lectura sobre el plan nominal de firma CT publicado
-- y vigente: ¿tiene un paso de ese circuito (huella), documento y orden, en
-- esa organización y esa unidad? Lo usa CT186 para ligar la unidad de la
-- consulta y la recuperación R5 V2 a la del paso (opción A, dirección 06/10),
-- con el mismo criterio de publicación vigente que CC7/CC8 y la misma
-- vigencia de entrada que CC7. No devuelve el documento ni ninguna entrada:
-- sólo verdadero o falso. Sin publicación vigente, falso. Sin acceso LOGIN:
-- sólo el propietario CT. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_catalogos_configurables:migracion:000010',0));
DO $pre$
BEGIN
 IF current_user<>'vec_catalogos_configurables_propietario'
    OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.validar_plan_nominal_firma_v1(bytea,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.comprobar_publicacion_plan_nominal_firma_v1(text,bigint,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)') IS NOT NULL
    OR pg_catalog.to_regrole('vec_contratacion_temporal_propietario') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.plan_firma_control')
      AND c.relowner=current_user::pg_catalog.regrole AND c.relrowsecurity AND c.relforcerowsecurity)
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_catalogos_configurables.plan_firma_publicacion')
      AND c.relowner=current_user::pg_catalog.regrole AND c.relrowsecurity AND c.relforcerowsecurity) THEN
  RAISE EXCEPTION 'CC10: PARO clave=preimagen actual=incompatible esperado=CC7_CC8_sin_CC10' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(
 p_circuito_sha256 text,p_documento text,p_paso_orden integer,p_organizacion_ref text,p_unidad_ref text)
RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='10s' SET TimeZone='UTC' AS $f$
DECLARE control vec_catalogos_configurables.plan_firma_control%ROWTYPE;
 publicacion vec_catalogos_configurables.plan_firma_publicacion%ROWTYPE; documento jsonb; n integer; ahora timestamptz;
BEGIN
 IF (p_circuito_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
    OR (p_documento ~ '^[a-z][a-z0-9_]{1,63}$') IS NOT TRUE
    OR p_paso_orden IS NULL OR p_paso_orden NOT BETWEEN 1 AND 99
    OR (p_organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
    OR (p_unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'CC10: paso de plan inválido' USING ERRCODE='22023'; END IF;
 -- La única publicación vigente del módulo, con el criterio de CC7/CC8: una
 -- versión publicada que ninguna posterior del mismo catálogo (publicada o
 -- retirada) ha sustituido. Publicar la v2 no retira la v1, así que puede
 -- haber dos filas «publicado»; vale la última. Dos catálogos vigentes a la
 -- vez, o ninguno, no dan un plan: falso.
 SELECT count(*) INTO n FROM vec_catalogos_configurables.plan_firma_control x
  WHERE x.modulo_id='contratacion_temporal' AND x.estado='publicado'
    AND NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control y
      WHERE y.modulo_id='contratacion_temporal' AND y.catalogo_id=x.catalogo_id
        AND y.version>x.version AND y.estado IN ('publicado','retirado'));
 IF n<>1 THEN RETURN false; END IF;
 SELECT * INTO STRICT control FROM vec_catalogos_configurables.plan_firma_control x
  WHERE x.modulo_id='contratacion_temporal' AND x.estado='publicado'
    AND NOT EXISTS(SELECT 1 FROM vec_catalogos_configurables.plan_firma_control y
      WHERE y.modulo_id='contratacion_temporal' AND y.catalogo_id=x.catalogo_id
        AND y.version>x.version AND y.estado IN ('publicado','retirado'));
 SELECT * INTO publicacion FROM vec_catalogos_configurables.plan_firma_publicacion p
  WHERE p.catalogo_id=control.catalogo_id AND p.version=control.version;
 IF NOT FOUND OR publicacion.publicacion_sha256 IS DISTINCT FROM control.publicacion_sha256
    OR publicacion.revision IS DISTINCT FROM control.publicacion_revision
    OR pg_catalog.encode(pg_catalog.sha256(publicacion.canonico_exacto),'hex') IS DISTINCT FROM control.publicacion_sha256
    OR publicacion.publicada_en>pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'CC10: publicación original incoherente' USING ERRCODE='55000'; END IF;
 documento:=vec_catalogos_configurables.validar_plan_nominal_firma_v1(publicacion.canonico_exacto,control.publicacion_sha256);
 IF documento->>'id' IS DISTINCT FROM control.catalogo_id
    OR documento->>'version' IS DISTINCT FROM control.version::text
    OR documento->>'estado' IS DISTINCT FROM 'publicado'
    OR (documento->>'publicado_en')::timestamptz IS DISTINCT FROM publicacion.publicada_en
    OR documento->>'publicado_por' IS DISTINCT FROM publicacion.publicado_por
    OR documento->>'aprobacion_ref' IS DISTINCT FROM publicacion.aprobacion_ref THEN
  RAISE EXCEPTION 'CC10: publicación ajena' USING ERRCODE='55000'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(documento->'entradas') e
  WHERE e.value->'atributos'->>'esquema'='ct.plan-competencia-firma.v2'
    AND (e.value->>'vigente_desde')::timestamptz<=ahora
    AND (e.value->>'vigente_hasta'='0001-01-01T00:00:00Z' OR (e.value->>'vigente_hasta')::timestamptz>ahora)
    AND e.value->'atributos'->>'circuito_sha256'=p_circuito_sha256
    AND e.value->'atributos'->>'documento'=p_documento
    AND e.value->'atributos'->>'paso_orden'=p_paso_orden::text
    AND e.value->'atributos'->>'organizacion_ref'=p_organizacion_ref
    AND e.value->'atributos'->>'unidad_ref'=p_unidad_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)
 TO vec_contratacion_temporal_propietario;
DO $post$
DECLARE f regprocedure:='vec_catalogos_configurables.paso_plan_firma_con_unidad_v1(text,text,integer,text,text)'::regprocedure;
BEGIN
 IF (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM
    ARRAY['vec_catalogos_configurables_propietario=X/vec_catalogos_configurables_propietario',
          'vec_contratacion_temporal_propietario=X/vec_catalogos_configurables_propietario']::aclitem[]
  OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS NOT TRUE THEN
  RAISE EXCEPTION 'CC10: PARO clave=ACL actual=incompatible esperado=propietario_CT_EXECUTE' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
