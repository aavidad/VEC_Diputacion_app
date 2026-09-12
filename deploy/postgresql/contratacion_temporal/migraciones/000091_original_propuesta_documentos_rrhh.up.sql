\set ON_ERROR_STOP on
-- CT91: lectura explícita del original de propuesta v7 desde su historia 7/8/9.
-- No modifica la consulta ordinaria, CT69, publicaciones ni datos de negocio.
-- Requiere CT44/45, CT61/69 y CT86. No depende de CT88/89.
-- La autorización sigue siendo consultar el detalle v7, con consumo V3 nuevo,
-- auditoría y Recibo V2 en la misma transacción serializable del adaptador.
-- No hay DOWN sobre la historia conservada; la retirada exige otro corte.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000091:original-propuesta',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

CREATE FUNCTION vec_contratacion_temporal.materializar_original_propuesta_rrhh_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_corte_global numeric
) RETURNS vec_contratacion_temporal.materializacion_detalle_rrhh_v1
LANGUAGE plpgsql STABLE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s' SET idle_in_transaction_session_timeout='6s'
AS $original$
DECLARE
 actual vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
 original vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
 paso vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
 resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
 primero vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
 version_actual numeric;
 publicacion vec_contratacion_temporal.publicacion_version_rrhh%ROWTYPE;
 historia vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 propuesta vec_contratacion_temporal.propuesta_formalizacion%ROWTYPE;
 resolucion vec_contratacion_temporal.resolucion_formalizacion%ROWTYPE;
 anotacion vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1%ROWTYPE;
 anterior jsonb;
 agregado jsonb;
 ultimo jsonb;
 numero integer;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
  OR p_consulta.version_observada IS DISTINCT FROM 7::numeric THEN
  RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
 END IF;
 -- El ámbito se aplica a la última publicación, nunca a una candidata antigua.
 -- Cero conserva la semántica instalada de última versión al corte.
 actual:=vec_contratacion_temporal.materializar_detalle_rrhh_v1(
  p_alcance,ROW(p_consulta.expediente_ref,0)::vec_contratacion_temporal.consulta_detalle_rrhh_v1,p_corte_global);
 resumen:=(actual.detalle).resumen;
 version_actual:=resumen.version;
 IF version_actual NOT IN (7,8,9) OR NOT EXISTS(
  SELECT 1 FROM vec_contratacion_temporal.expediente_integral_actual e
  WHERE e.expediente_ref=p_consulta.expediente_ref AND e.version=version_actual
 ) THEN
  RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
 END IF;

 FOR numero IN 7..version_actual::integer LOOP
  SELECT p.* INTO STRICT publicacion
   FROM vec_contratacion_temporal.publicacion_version_rrhh p
   WHERE p.expediente_ref=p_consulta.expediente_ref AND p.version=numero
    AND p.corte_global<=p_corte_global;
  -- Reutiliza la proyección y todas las guardas publicación/agregado/ámbito.
  -- El corte histórico se deriva aquí, no se recibe del navegador ni del pool.
  paso:=vec_contratacion_temporal.materializar_detalle_rrhh_v1(
   p_alcance,ROW(p_consulta.expediente_ref,numero)::vec_contratacion_temporal.consulta_detalle_rrhh_v1,
   publicacion.corte_global);
  resumen:=(paso.detalle).resumen;
  SELECT h.* INTO STRICT historia
   FROM vec_contratacion_temporal.expediente_version_integral h
   WHERE h.expediente_ref=p_consulta.expediente_ref AND h.version=numero;
  agregado:=historia.agregado_json;
  ultimo:=agregado->'actuaciones'->-1;
  IF resumen.version IS DISTINCT FROM numero::numeric
   OR historia.agregado_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(agregado::text,'UTF8')),'hex')
   OR historia.fase_clave IS DISTINCT FROM 'nombramiento'
   OR historia.estado IS DISTINCT FROM 'en_curso'
   OR (ultimo->>'secuencia')::numeric IS DISTINCT FROM numero::numeric
   OR (ultimo->>'version_expediente')::numeric IS DISTINCT FROM numero::numeric THEN
   RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
  END IF;
  IF numero=7 THEN
   original:=paso;
   primero:=resumen;
   SELECT p.* INTO STRICT propuesta FROM vec_contratacion_temporal.propuesta_formalizacion p
    WHERE p.expediente_ref=p_consulta.expediente_ref
     AND p.organizacion_ref=p_alcance.organizacion_ref
     AND p.propuesta_ref=historia.operacion_ref AND p.version_resultante=7;
   IF historia.origen_version IS DISTINCT FROM 'propuesta_formalizacion_o6'
    OR ultimo->>'accion_clave' IS DISTINCT FROM 'registrar_propuesta_formalizacion'
    OR ultimo->>'recibo_ref' IS DISTINCT FROM propuesta.recibo_ref
    OR propuesta.recibo_json->>'PropuestaRef' IS DISTINCT FROM propuesta.propuesta_ref
    OR propuesta.recibo_json->>'ReciboLocalRef' IS DISTINCT FROM propuesta.recibo_ref THEN
    RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
   END IF;
  ELSE
   -- La propuesta y cada sucesora conservan exactamente ámbito y definición.
   -- Sólo versión, instante y una actuación añadida distinguen las postimágenes.
   IF resumen.organizacion_ref IS DISTINCT FROM primero.organizacion_ref
    OR resumen.centro_ref IS DISTINCT FROM primero.centro_ref
    OR resumen.unidad_ref IS DISTINCT FROM primero.unidad_ref
    OR resumen.flujo_ref IS DISTINCT FROM primero.flujo_ref
    OR resumen.flujo_version IS DISTINCT FROM primero.flujo_version
    OR resumen.flujo_huella_sha256 IS DISTINCT FROM primero.flujo_huella_sha256
    OR (agregado-ARRAY['version','actualizado_en','actuaciones'])
       IS DISTINCT FROM (anterior-ARRAY['version','actualizado_en','actuaciones'])
    OR ((agregado->'actuaciones')-(-1)) IS DISTINCT FROM anterior->'actuaciones' THEN
    RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
   END IF;
   IF numero=8 THEN
    SELECT r.* INTO STRICT resolucion FROM vec_contratacion_temporal.resolucion_formalizacion r
     WHERE r.expediente_ref=p_consulta.expediente_ref AND r.organizacion_ref=p_alcance.organizacion_ref
      AND r.propuesta_ref=propuesta.propuesta_ref AND r.version_resultante=8;
    IF historia.origen_version IS DISTINCT FROM 'resolucion_formalizacion_o6'
     OR historia.operacion_ref IS DISTINCT FROM resolucion.resolucion_ref
     OR ultimo->>'accion_clave' IS DISTINCT FROM 'registrar_resolucion_formalizacion'
     OR ultimo->>'recibo_ref' IS DISTINCT FROM resolucion.recibo_ref
     OR resolucion.recibo_json->>'ResolucionRef' IS DISTINCT FROM resolucion.resolucion_ref
     OR resolucion.recibo_json->>'ReciboRef' IS DISTINCT FROM resolucion.recibo_ref THEN
     RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
    END IF;
   ELSE
    SELECT a.* INTO STRICT anotacion FROM vec_contratacion_temporal.anotacion_administrativa_incorporacion_v1 a
     WHERE a.expediente_ref=p_consulta.expediente_ref AND a.organizacion_ref=p_alcance.organizacion_ref
      AND a.version_anterior=8 AND a.version_resultante=9;
    IF historia.origen_version IS DISTINCT FROM 'anotacion_administrativa_ct86'
     OR historia.operacion_ref IS DISTINCT FROM anotacion.recibo_ref
     OR ultimo->>'accion_clave' IS DISTINCT FROM 'contratacion_temporal.anotacion_administrativa.registrar'
     OR ultimo->>'recibo_ref' IS DISTINCT FROM anotacion.recibo_ref
     OR ultimo->'seguimiento_original' IS DISTINCT FROM anotacion.seguimiento_original
     OR anotacion.recibo_json->>'recibo_ref' IS DISTINCT FROM anotacion.recibo_ref
     OR anotacion.recibo_json->>'expediente_ref' IS DISTINCT FROM p_consulta.expediente_ref THEN
     RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
    END IF;
   END IF;
  END IF;
  anterior:=agregado;
 END LOOP;
 RETURN original;
EXCEPTION
 WHEN SQLSTATE '40001' OR SQLSTATE '40P01' OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
 WHEN OTHERS THEN RAISE EXCEPTION 'original de propuesta no disponible' USING ERRCODE='42501';
END
$original$;

-- Derivación limitada de las envolventes verificadas: las guardas, el consumo,
-- el contexto autorizado v7 y el cierre del recibo se conservan byte a byte.
-- Las preimágenes exactas impiden copiar silenciosamente una versión distinta.
DO $envolventes$
DECLARE
 fuente text; destino text; llamada text; llamada_nueva text; huella text;
 firma text; ddl text; nuevo text; f oid; creada oid;
 p pg_proc%ROWTYPE; q pg_proc%ROWTYPE;
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR getdatabaseencoding()<>'UTF8'
  OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=propietario)
  OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CT91: propietario incompatible' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=to_regprocedure(
   'vec_contratacion_temporal.materializar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)')
  AND proowner=propietario AND prosecdef AND provolatile='s' AND proparallel='u'
  AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='d7ba4ebd03ef8ddd10ca2b464547011a7508ffae401e1611c35810409323dfad') THEN
  RAISE EXCEPTION 'CT91: materializador CT69 incompatible' USING ERRCODE='55000';
 END IF;
 FOR fuente,destino,llamada,llamada_nueva,huella,firma IN VALUES
  ('motor_consultar_detalle_rrhh_v1','motor_consultar_original_propuesta_rrhh_v1',
   'materializar_detalle_rrhh_v1','materializar_original_propuesta_rrhh_v1',
   'f85bdf81cdc3e30b0b1acf0af60252d61fcdeed20aef4031a5cbe6312c1af4d0',
   '(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'),
  ('consultar_detalle_rrhh_atestado_v1','consultar_original_propuesta_rrhh_atestado_v1',
   'motor_consultar_detalle_rrhh_v1','motor_consultar_original_propuesta_rrhh_v1',
   '7b7d6c4a419262d54ddb7f2a096e5a4d6e1c962bc58e3027546e221717ed1814',
   '(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
 LOOP
  f:=to_regprocedure('vec_contratacion_temporal.'||fuente||firma);
  IF f IS NULL OR to_regprocedure('vec_contratacion_temporal.'||destino||firma) IS NOT NULL THEN
   RAISE EXCEPTION 'CT91: dependencia ausente o destino existente' USING ERRCODE='55000';
  END IF;
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>propietario OR NOT p.prosecdef OR p.provolatile<>'v' OR p.proparallel<>'u'
   OR p.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
   OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC',
     'lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM huella
   OR (length(p.prosrc)-length(replace(p.prosrc,llamada,'')))/length(llamada)<>1 THEN
   RAISE EXCEPTION 'CT91: envolvente previa incompatible' USING ERRCODE='55000';
  END IF;
  ddl:=pg_get_functiondef(f);
  IF (length(ddl)-length(replace(ddl,p.prosrc,'')))<>length(p.prosrc)
   OR (length(ddl)-length(replace(ddl,'FUNCTION vec_contratacion_temporal.'||fuente||'(','')))
      <>length('FUNCTION vec_contratacion_temporal.'||fuente||'(') THEN
   RAISE EXCEPTION 'CT91: cabecera no unívoca' USING ERRCODE='55000';
  END IF;
  nuevo:=replace(p.prosrc,llamada,llamada_nueva);
  ddl:=replace(ddl,p.prosrc,nuevo);
  ddl:=replace(ddl,'FUNCTION vec_contratacion_temporal.'||fuente||'(',
                   'FUNCTION vec_contratacion_temporal.'||destino||'(');
  EXECUTE ddl;
  creada:=to_regprocedure('vec_contratacion_temporal.'||destino||firma);
  SELECT * INTO STRICT q FROM pg_proc WHERE oid=creada;
  IF q.prosrc IS DISTINCT FROM nuevo
   OR (to_jsonb(q)-ARRAY['oid','proname','prosrc','proacl'])
      IS DISTINCT FROM (to_jsonb(p)-ARRAY['oid','proname','prosrc','proacl'])
   OR (SELECT to_jsonb(z) FROM pg_proc z WHERE oid=f) IS DISTINCT FROM to_jsonb(p) THEN
   RAISE EXCEPTION 'CT91: derivación alteró atributos u original' USING ERRCODE='55000';
  END IF;
 END LOOP;
END
$envolventes$;

-- Revoca incluso ACL heredadas por privilegios predeterminados, sólo en las
-- tres funciones nuevas. No concede acceso a tablas, tipos, motores o roles.
DO $acl$
DECLARE f regprocedure; permiso record; propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 lector oid:='vec_contratacion_temporal_consultor_rrhh'::regrole;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_contratacion_temporal.materializar_original_propuesta_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,numeric)'::regprocedure,
  'vec_contratacion_temporal.motor_consultar_original_propuesta_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'::regprocedure,
  'vec_contratacion_temporal.consultar_original_propuesta_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
 ] LOOP
  FOR permiso IN SELECT DISTINCT a.grantee FROM pg_proc p,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>propietario
  LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f,
    CASE WHEN permiso.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(permiso.grantee)) END);
  END LOOP;
  IF (SELECT proname FROM pg_proc WHERE oid=f)='consultar_original_propuesta_rrhh_atestado_v1' THEN
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_contratacion_temporal_consultor_rrhh',f);
  END IF;
  IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>propietario
    OR (a.grantee<>propietario AND NOT (a.grantee=lector AND p.proname='consultar_original_propuesta_rrhh_atestado_v1')))) THEN
   RAISE EXCEPTION 'CT91: ACL nueva incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
END
$acl$;
COMMENT ON FUNCTION vec_contratacion_temporal.consultar_original_propuesta_rrhh_atestado_v1(
 vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
IS 'Original de propuesta v7: lectura autorizada y auditada, historia exacta 7/8/9 y ámbito actual y original. No firma ni altera documentos.';
COMMIT;
