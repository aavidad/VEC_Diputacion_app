\set ON_ERROR_STOP on
-- AUT68: propuesta y publicación nominal de versiones de inscripción.
-- Depende de AUT66, AUT67, AD231 y CA39; no siembra permisos al instalar.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regclass('vec_autorizacion.propuesta_version_inscripcion_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.cierre_version_inscripcion_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.validar_plan_version_inscripcion_v1(jsonb,boolean)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.acreditar_version_inscripcion_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)') IS NULL
 OR pg_catalog.has_schema_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1','USAGE') IS NOT TRUE
 OR pg_catalog.has_schema_privilege('vec_autorizacion_propietario','vec_personal','USAGE') IS NOT TRUE
 OR pg_catalog.has_schema_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3','USAGE') IS NOT TRUE
 OR pg_catalog.has_function_privilege('vec_autorizacion_propietario',
  'vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)','EXECUTE') IS NOT TRUE
 OR pg_catalog.has_function_privilege('vec_autorizacion_propietario',
  'vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)','EXECUTE') IS NOT TRUE
 OR pg_catalog.has_function_privilege('vec_autorizacion_propietario',
  'vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)','EXECUTE') IS NOT TRUE
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
  CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,
   pg_catalog.acldefault('f',p.proowner))) acl
  WHERE p.oid='vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)'::pg_catalog.regprocedure
   AND acl.grantee=0 AND acl.privilege_type='EXECUTE')
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_version_inscripcion_v1(jsonb)') IS NULL
 OR pg_catalog.has_function_privilege('vec_autorizacion_propietario',
  'vec_autorizacion_atestada_v3.consumir_version_inscripcion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 OR pg_catalog.has_function_privilege('vec_autorizacion_propietario',
  'vec_autorizacion_atestada_v3.registrar_intento_version_inscripcion_v1(jsonb)','EXECUTE') IS NOT TRUE
 OR pg_catalog.to_regrole('vec_admin_version_inscripcion_ejecutor') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT68: dependencias AUT66 AUT67 AD231 CA39 ausentes' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.administradores_version_inscripcion_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE efectivos jsonb;salida jsonb;concesiones jsonb;
BEGIN
 concesiones:=pg_catalog.jsonb_build_array(
  pg_catalog.jsonb_build_object('accion','administracion.perfiles.version_inscripcion.proponer',
   'modulo_id','administracion','tipo_recurso','definicion_rol',
   'finalidades',pg_catalog.jsonb_build_array('gobierno_definiciones_perfiles'),
   'garantia_minima','alto','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb),
  pg_catalog.jsonb_build_object('accion','administracion.perfiles.version_inscripcion.aprobar',
   'modulo_id','administracion','tipo_recurso','propuesta_definicion_rol',
   'finalidades',pg_catalog.jsonb_build_array('gobierno_definiciones_perfiles'),
   'garantia_minima','alto','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb));
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 SELECT COALESCE(pg_catalog.jsonb_agg(x),'[]'::jsonb) INTO salida
 FROM pg_catalog.jsonb_array_elements(efectivos) x
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=x->>'asignacion_ref'
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 m USING(version_rol_ref)
 WHERE r.rol_id='administracion_perfiles' AND m.categoria_administrativa='aplicacion'
 AND m.tipo_perfil='fijo_sistema' AND m.version_rol_huella_sha256=r.huella_sha256
 AND r.documento->'concesiones' @> concesiones
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(concesiones) c
  WHERE NOT EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n
   WHERE n.version_rol_ref=r.version_rol_ref
    AND n.accion_ref='accion:'||(c->>'accion')
    AND n.fuente_ref=r.version_rol_ref AND n.fuente_version=r.version
    AND n.fuente_huella_sha256=r.huella_sha256
    AND n.concesion=c AND n.clase_control='administrador_aplicacion'
    AND n.dimensiones_ambito='["organizacion_ref"]'::jsonb
    AND pg_catalog.clock_timestamp()>=n.vigente_desde
    AND (n.vigente_hasta IS NULL OR pg_catalog.clock_timestamp()<n.vigente_hasta)));
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.administradores_version_inscripcion_v1() FROM PUBLIC;

-- CA39 da el vínculo positivo de persona/perfil interno con empleado. La
-- proyección enviada en el plan debe ser exactamente la acreditada y vigente.
CREATE FUNCTION vec_autorizacion.acreditar_cambios_empleado_inscripcion_v1(p jsonb)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE cambio jsonb;v jsonb;pr jsonb;ahora timestamptz;
BEGIN
 IF p->>'perfil_objetivo'<>'empleado' THEN RETURN;END IF;
 IF pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(text,text)') IS NULL
 THEN RAISE EXCEPTION 'AUT68: fuente de empleado ausente' USING ERRCODE='42501';END IF;
 ahora:=pg_catalog.clock_timestamp();
 FOR cambio IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'asignaciones') LOOP
  v:=vec_contexto_actor_v1.acreditar_empleado_perfil_interno_inscripcion_v1(
   cambio->>'principal_id',cambio->>'perfil_activo_ref');
  pr:=cambio->'proyeccion_empleado';
  IF v->>'estado' IS DISTINCT FROM 'acreditado'
  OR v->>'persona_ref' IS DISTINCT FROM cambio->>'principal_id'
  OR v->>'perfil_ref' IS DISTINCT FROM cambio->>'perfil_activo_ref'
  OR v->>'empleado_ref' IS DISTINCT FROM cambio->>'empleado_ref'
  OR v->>'proyeccion_ref' IS DISTINCT FROM pr->>'proyeccion_ref'
  OR v->>'version' IS DISTINCT FROM pr->>'version'
  OR v->>'procedencia_ref' IS DISTINCT FROM pr->>'procedencia_ref'
  OR v->>'procedencia_version' IS DISTINCT FROM pr->>'procedencia_version'
  OR v->>'procedencia_huella_sha256' IS DISTINCT FROM pr->>'procedencia_huella_sha256'
  OR pg_catalog.jsonb_typeof(v->'vigente_desde') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(v->'vigente_hasta') IS DISTINCT FROM 'string'
  OR ahora<(v->>'vigente_desde')::timestamptz
  OR ahora>=(v->>'vigente_hasta')::timestamptz
  OR (cambio->>'vigente_desde')::timestamptz<(v->>'vigente_desde')::timestamptz
  OR (cambio->>'vigente_hasta')::timestamptz>(v->>'vigente_hasta')::timestamptz
  THEN RAISE EXCEPTION 'AUT68: empleado/perfil/procedencia no acreditados' USING ERRCODE='42501';END IF;
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_cambios_empleado_inscripcion_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.acreditar_cambios_externos_inscripcion_v1(p jsonb)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE cambio jsonb;
BEGIN
 IF p->>'perfil_objetivo'<>'externo' THEN RETURN;END IF;
 FOR cambio IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'asignaciones') LOOP
  IF vec_contexto_actor_v1.acreditar_candidato_externo_v1(
   cambio->>'principal_id',cambio->>'perfil_activo_ref',cambio->>'candidato_ref') IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT68: candidato externo no acreditado' USING ERRCODE='42501';END IF;
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_cambios_externos_inscripcion_v1(jsonb) FROM PUBLIC;

-- Personal31 acredita la relación organización/unidad con revisión y huella
-- de fuente exactas. Su barrera de generación queda bloqueada hasta COMMIT.
CREATE FUNCTION vec_autorizacion.acreditar_cambios_rrhh_inscripcion_v1(p jsonb,p_org text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE cambio jsonb;unidad text;fuente jsonb;selector jsonb;acuse jsonb;
 salida jsonb:='{}'::jsonb;
BEGIN
 IF p->>'perfil_objetivo'<>'rrhh' THEN RETURN salida;END IF;
 IF p_org IS NULL OR p_org !~ '^[a-z][a-z0-9_:-]{2,127}$'
 THEN RAISE EXCEPTION 'AUT68: organización ADMIN inválida' USING ERRCODE='42501';END IF;
 FOR cambio IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'asignaciones') LOOP
  SELECT e->'valores'->>0 INTO unidad FROM pg_catalog.jsonb_array_elements(cambio->'ambitos') e
   WHERE e->>'clave'='unidad_ref';
  fuente:=cambio->'fuente_unidad';
  IF unidad IS NULL OR pg_catalog.jsonb_typeof(fuente) IS DISTINCT FROM 'object'
   OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(fuente))<>3
   OR NOT fuente ?& ARRAY['referencia','version','huella_sha256']
  THEN RAISE EXCEPTION 'AUT68: fuente de unidad RRHH ausente' USING ERRCODE='42501';END IF;
  selector:=pg_catalog.jsonb_build_object('dimension','unidad_ref',
   'valores',pg_catalog.jsonb_build_array(unidad),'fuente',fuente);
  acuse:=vec_personal.cotejar_unidad_bootstrap_admin_v1(p_org,selector,
   (cambio->>'vigente_hasta')::timestamptz);
  IF acuse->>'esquema' IS DISTINCT FROM 'vec.personal.unidad-bootstrap-admin.v1'
   OR acuse->>'organizacion_ref' IS DISTINCT FROM p_org
   OR acuse->>'unidad_ref' IS DISTINCT FROM unidad
   OR acuse->'fuente' IS DISTINCT FROM fuente
  OR (acuse->>'valida_hasta')::timestamptz
      IS DISTINCT FROM (cambio->>'vigente_hasta')::timestamptz
  THEN RAISE EXCEPTION 'AUT68: unidad RRHH ajena a organización' USING ERRCODE='42501';END IF;
  salida:=salida||pg_catalog.jsonb_build_object(cambio->>'asignacion_id',acuse);
 END LOOP;
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_cambios_rrhh_inscripcion_v1(jsonb,text) FROM PUBLIC;

-- La primera aprobación revalida la vigencia temporal justo antes de
-- devolver. La cabeza sigue bloqueada desde validar_plan(...,true).
CREATE FUNCTION vec_autorizacion.revalidar_catalogo_cierre_inscripcion_v1(p jsonb)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE h record;catalogo jsonb;seleccion jsonb;entrada jsonb;ahora timestamptz;
BEGIN
 SELECT * INTO h FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
  WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
 IF NOT FOUND OR h.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR h.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
 THEN RAISE EXCEPTION 'AUT68: cabeza de catálogo cambió' USING ERRCODE='40001';END IF;
 catalogo:=pg_catalog.convert_from(vec_autorizacion.resolver_catalogo_acciones_administracion_v1(
  p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256'),'UTF8')::jsonb;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora<(catalogo->>'vigente_desde')::timestamptz
  OR (catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
   AND ahora>=(catalogo->>'vigente_hasta')::timestamptz)
 THEN RAISE EXCEPTION 'AUT68: catálogo vencido' USING ERRCODE='42501';END IF;
 FOR seleccion IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'selecciones') LOOP
  SELECT value INTO entrada FROM pg_catalog.jsonb_array_elements(catalogo->'entradas')
   WHERE value->>'referencia'=seleccion->>'entrada_ref';
  IF NOT FOUND OR entrada->>'version' IS DISTINCT FROM seleccion->>'entrada_version'
   OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    vec_autorizacion.canon_gobierno_rol_nuevo_v1(entrada,'entrada'),'UTF8')),'hex')
    IS DISTINCT FROM seleccion->>'entrada_huella_sha256'
   OR ahora<(entrada->>'vigente_desde')::timestamptz
   OR (entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
    AND ahora>=(entrada->>'vigente_hasta')::timestamptz)
  THEN RAISE EXCEPTION 'AUT68: descriptor vencido o divergente' USING ERRCODE='42501';END IF;
  IF p->>'perfil_objetivo'='empleado' AND NOT EXISTS(
   SELECT 1 FROM pg_catalog.jsonb_array_elements(catalogo->'entradas') par(e)
   WHERE par.e->>'clase_control'='ordinario'
    AND par.e->'dimensiones_ambito'='["candidato_ref"]'::jsonb
    AND par.e->'concesion'=pg_catalog.jsonb_set(entrada->'concesion','{tipo_recurso}',
     pg_catalog.to_jsonb(pg_catalog.left(entrada#>>'{concesion,tipo_recurso}',
      pg_catalog.length(entrada#>>'{concesion,tipo_recurso}')-9)))
    AND vec_autorizacion.concesion_inscripcion_exacta_v1(par.e->'concesion','externo') IS TRUE
    AND ahora>=(par.e->>'vigente_desde')::timestamptz
    AND (par.e->>'vigente_hasta'='0001-01-01T00:00:00Z'
     OR ahora<(par.e->>'vigente_hasta')::timestamptz))
  THEN RAISE EXCEPTION 'AUT68: descriptor externo par vencido' USING ERRCODE='42501';END IF;
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_catalogo_cierre_inscripcion_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.solicitud_replay_version_inscripcion_v1(original bytea,actual text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT (pg_catalog.convert_from(original,'UTF8')::jsonb-'correlacion_ref')
  IS NOT DISTINCT FROM (actual::jsonb-'correlacion_ref')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.solicitud_replay_version_inscripcion_v1(bytea,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.comprobar_postimagen_version_inscripcion_v1(p_propuesta text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE c record;r record;cv record;prop record;p jsonb;cambio jsonb;a record;
 comprobante jsonb;unidad_acuse jsonb;aprobador_asg record;
 tabla_externa boolean;i integer:=0;unidad text;org text;
BEGIN
 SELECT * INTO STRICT c FROM vec_autorizacion.cierre_version_inscripcion_v1 WHERE propuesta_ref=p_propuesta;
 SELECT * INTO STRICT prop FROM vec_autorizacion.propuesta_version_inscripcion_v1 WHERE propuesta_ref=p_propuesta;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref=c.version_rol_ref FOR SHARE;
 SELECT * INTO STRICT cv FROM vec_autorizacion.control_vigencia_version_rol
  WHERE version_rol_ref=c.version_rol_ref AND revision=1 FOR SHARE;
 IF r.huella_sha256 IS DISTINCT FROM c.rol_sha256
 OR r.documento IS DISTINCT FROM c.resultado#>'{recibo,version_rol}'
 OR cv.huella_sha256 IS DISTINCT FROM c.control_sha256 OR cv.estado<>'habilitada'
 OR cv.documento IS DISTINCT FROM c.resultado#>'{recibo,control_posterior}'
 THEN RAISE EXCEPTION 'AUT68: postimagen de rol divergente' USING ERRCODE='40001';END IF;
 p:=pg_catalog.convert_from(prop.material,'UTF8')::jsonb->'Plan';
 SELECT * INTO STRICT aprobador_asg FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=c.asignacion_ref FOR SHARE;
 SELECT e->'valores'->>0 INTO org FROM pg_catalog.jsonb_array_elements(
  aprobador_asg.documento->'ambitos') e WHERE e->>'clave'='organizacion_ref';
 FOR cambio IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'asignaciones') LOOP
  comprobante:=(c.resultado#>'{recibo,asignaciones}')->i;
  unidad_acuse:=comprobante->'unidad_acuse';
  tabla_externa:=cambio->>'almacen'='externo';
  IF tabla_externa THEN
   SELECT * INTO a FROM vec_autorizacion.asignacion_perfil_externa
    WHERE asignacion_ref='asignacion:'||cambio->>'asignacion_id'||':v1' FOR SHARE;
  ELSE
   SELECT * INTO a FROM vec_autorizacion.asignacion_perfil
    WHERE asignacion_ref='asignacion:'||cambio->>'asignacion_id'||':v'||
     (CASE WHEN cambio->>'modo'='avance' THEN ((cambio#>>'{anterior,documento,version}')::bigint+1)::text ELSE '1' END) FOR SHARE;
  END IF;
  IF NOT FOUND OR a.version_rol_ref IS DISTINCT FROM c.version_rol_ref
   OR a.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
     vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
   OR comprobante->>'posterior_ref' IS DISTINCT FROM a.asignacion_ref
   OR comprobante->>'posterior_sha256' IS DISTINCT FROM a.huella_sha256
   OR comprobante->'posterior_documento' IS DISTINCT FROM a.documento
   OR (CASE WHEN cambio->>'modo'='avance' THEN
      comprobante->>'anterior_ref' IS DISTINCT FROM cambio#>>'{anterior,asignacion_ref}'
      OR comprobante->>'anterior_sha256' IS DISTINCT FROM cambio#>>'{anterior,huella_sha256}'
     ELSE comprobante ? 'anterior_ref' OR comprobante ? 'anterior_sha256' END)
   OR a.documento->>'asignacion_id' IS DISTINCT FROM cambio->>'asignacion_id'
   OR a.documento->>'version' IS DISTINCT FROM
    (CASE WHEN cambio->>'modo'='avance' THEN ((cambio#>>'{anterior,documento,version}')::bigint+1)::text ELSE '1' END)
   OR a.documento->>'principal_id' IS DISTINCT FROM cambio->>'principal_id'
   OR a.documento->>'perfil_activo_ref' IS DISTINCT FROM cambio->>'perfil_activo_ref'
   OR a.documento->>'estado' IS DISTINCT FROM 'activa'
   OR a.documento->'ambitos' IS DISTINCT FROM cambio->'ambitos'
   OR a.documento->>'vigente_desde' IS DISTINCT FROM cambio->>'vigente_desde'
   OR a.documento->>'vigente_hasta' IS DISTINCT FROM cambio->>'vigente_hasta'
  THEN RAISE EXCEPTION 'AUT68: postimagen de asignación divergente' USING ERRCODE='40001';END IF;
  IF p->>'perfil_objetivo'='rrhh' THEN
   SELECT e->'valores'->>0 INTO unidad FROM pg_catalog.jsonb_array_elements(cambio->'ambitos') e
    WHERE e->>'clave'='unidad_ref';
   IF unidad_acuse->>'esquema' IS DISTINCT FROM 'vec.personal.unidad-bootstrap-admin.v1'
    OR unidad_acuse->>'organizacion_ref' IS DISTINCT FROM org
    OR unidad_acuse->>'unidad_ref' IS DISTINCT FROM unidad
    OR unidad_acuse->'fuente' IS DISTINCT FROM cambio->'fuente_unidad'
    OR (unidad_acuse->>'valida_hasta')::timestamptz
       IS DISTINCT FROM (cambio->>'vigente_hasta')::timestamptz
   THEN RAISE EXCEPTION 'AUT68: acuse histórico de unidad divergente' USING ERRCODE='40001';END IF;
  ELSIF comprobante ? 'unidad_acuse' THEN
   RAISE EXCEPTION 'AUT68: acuse de unidad ajeno al perfil' USING ERRCODE='40001';
  END IF;
  i:=i+1;
 END LOOP;
 IF i<>pg_catalog.jsonb_array_length(c.resultado#>'{recibo,asignaciones}')
 THEN RAISE EXCEPTION 'AUT68: recibo de asignaciones divergente' USING ERRCODE='40001';END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.comprobar_postimagen_version_inscripcion_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_version_inscripcion_v1(
 p_cierre boolean,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='30s' AS $f$
DECLARE m jsonb;mat jsonb;plan jsonb;d jsonb;cap jsonb;motivo jsonb;administradores jsonb;
 prop vec_autorizacion.propuesta_version_inscripcion_v1;
 prev vec_autorizacion.cierre_version_inscripcion_v1;
 x record;admin_asg record;prop_asg record;cambio jsonb;asg jsonb;pre jsonb;ant record;puntero record;
 amb jsonb;ambcanon text;ctx text;huella text;material_sha text;rol jsonb;control jsonb;
 persona text;perfil text;asignacion text;recurso text;op text;accion text;audiencia text;
 ahora timestamptz(6);caduca timestamptz(6);fecha text;rol_sha text;control_sha text;
 aid text;aref text;asha text;acto text;recibo_ref text;resultado jsonb;recibo jsonb;
 asignaciones_emitidas jsonb:='[]'::jsonb;rrhh_acuses jsonb:='{}'::jsonb;
 claves text[];replay boolean:=false;externa boolean;version_nueva bigint;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 OR pg_catalog.current_setting('role')<>'none'
 OR p_cierre IS NULL OR p_material IS NULL
 OR pg_catalog.octet_length(p_material) NOT BETWEEN 1 AND
   (CASE WHEN p_cierre THEN 65536 ELSE 262144 END)
 OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_motivo IS NULL OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'AUT68: contexto inválido' USING ERRCODE='42501';END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 m:=p_material::jsonb;d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 cap:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
 IF pg_catalog.jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_path_exists(m,'$.** ? (@ == null)')
 THEN RAISE EXCEPTION 'AUT68: sobre inválido' USING ERRCODE='22023';END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO claves FROM pg_catalog.jsonb_object_keys(m) k;
 IF NOT p_cierre THEN
  IF claves IS DISTINCT FROM ARRAY['correlacion_ref','esquema','material_canon','material_sha256','plan_sha256']::text[]
  OR pg_catalog.jsonb_typeof(m->'correlacion_ref') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(m->'esquema') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(m->'material_sha256') IS DISTINCT FROM 'string'
  OR pg_catalog.jsonb_typeof(m->'plan_sha256') IS DISTINCT FROM 'string'
  OR m->>'esquema' IS DISTINCT FROM 'administracion_version_inscripcion_propuesta_v1'
  OR pg_catalog.jsonb_typeof(m->'material_canon') IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(m->>'material_canon') NOT BETWEEN 1 AND 196608
  THEN RAISE EXCEPTION 'AUT68: propuesta inválida' USING ERRCODE='22023';END IF;
  mat:=(m->>'material_canon')::jsonb;plan:=mat->'Plan';
  IF vec_autorizacion.canon_version_inscripcion_v1(mat,'material') IS DISTINCT FROM m->>'material_canon'
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(m->>'material_canon','UTF8')),'hex') IS DISTINCT FROM m->>'material_sha256'
  OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   vec_autorizacion.canon_version_inscripcion_v1(plan,'plan'),'UTF8')),'hex') IS DISTINCT FROM m->>'plan_sha256'
  OR mat->>'OperacionRef' !~ '^propuesta_admin:[0-9a-f]{32}$'
  THEN RAISE EXCEPTION 'AUT68: canon o huella divergente' USING ERRCODE='22023';END IF;
  persona:=mat->>'ProponentePersonaRef';perfil:=mat->>'PerfilActivoRef';
  asignacion:=mat->>'AsignacionPerfilRef';recurso:=plan->>'version_rol_objetivo_ref';
  op:=mat->>'OperacionRef';motivo:=plan->'motivo';
  accion:='administracion.perfiles.version_inscripcion.proponer';
  audiencia:='vec_autorizacion.version_inscripcion.propuesta.v1';
 ELSE
  IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','asignacion_ref',
   'correlacion_ref','decision','esquema','motivo','operacion_ref',
   'propuesta_huella_sha256','propuesta_ref']::text[]
  OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_object_keys(m) k
   WHERE k<>'motivo' AND pg_catalog.jsonb_typeof(m->k) IS DISTINCT FROM 'string')
  OR pg_catalog.jsonb_typeof(m->'motivo') IS DISTINCT FROM 'object'
  OR m->>'esquema' IS DISTINCT FROM 'administracion_version_inscripcion_cierre_v1'
  OR m->>'decision' IS DISTINCT FROM 'aprobada'
  OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$'
  OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$'
  OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AUT68: cierre inválido' USING ERRCODE='22023';END IF;
  persona:=m->>'actor_persona_ref';perfil:=m->>'actor_perfil_ref';
  asignacion:=m->>'asignacion_ref';recurso:=m->>'propuesta_ref';
  op:=m->>'operacion_ref';motivo:=m->'motivo';
  accion:='administracion.perfiles.version_inscripcion.aprobar';
  audiencia:='vec_autorizacion.version_inscripcion.cierre.v1';
  SELECT * INTO STRICT prop FROM vec_autorizacion.propuesta_version_inscripcion_v1
   WHERE propuesta_ref=recurso FOR SHARE;
  mat:=pg_catalog.convert_from(prop.material,'UTF8')::jsonb;plan:=mat->'Plan';
 END IF;
 PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(motivo,'motivo');
 IF (persona ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
 OR (perfil ~ '^prf_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
 OR m->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR (pg_catalog.convert_from(p_motivo,'UTF8')::jsonb)->'referencia' IS DISTINCT FROM motivo
 OR d->>'principal_id' IS DISTINCT FROM persona
 OR d->>'perfil_activo_ref' IS DISTINCT FROM perfil
 OR d->>'asignacion_ref' IS DISTINCT FROM asignacion
 OR d->>'accion' IS DISTINCT FROM accion
 OR d->>'correlacion_ref' IS DISTINCT FROM m->>'correlacion_ref'
 OR cap->>'audiencia_consumo' IS DISTINCT FROM audiencia
 OR vec_autorizacion.acreditar_version_inscripcion_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT68: actor o decisión ajenos' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT admin_asg FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=asignacion FOR SHARE;
 IF pg_catalog.jsonb_typeof(admin_asg.documento->'ambitos') IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_array_length(admin_asg.documento->'ambitos')<>2
 OR (SELECT pg_catalog.count(DISTINCT b->>'clave') FROM pg_catalog.jsonb_array_elements(
  admin_asg.documento->'ambitos') b)<>2
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(admin_asg.documento->'ambitos') b
  WHERE b->>'clave' NOT IN('organizacion_ref','unidad_ref')
   OR pg_catalog.jsonb_typeof(b->'valores') IS DISTINCT FROM 'array'
   OR pg_catalog.jsonb_array_length(b->'valores')<>1)
 THEN RAISE EXCEPTION 'AUT68: ámbitos ADMIN inválidos' USING ERRCODE='42501';END IF;
 SELECT pg_catalog.jsonb_object_agg(b->>'clave',b#>>'{valores,0}') INTO amb
 FROM pg_catalog.jsonb_array_elements(admin_asg.documento->'ambitos') b;
 IF p_cierre THEN
  SELECT * INTO STRICT prop_asg FROM vec_autorizacion.asignacion_perfil
   WHERE asignacion_ref=prop.asignacion_ref FOR SHARE;
  IF prop_asg.principal_id IS DISTINCT FROM prop.proponente_persona_ref
   OR prop_asg.perfil_activo_ref IS DISTINCT FROM prop.proponente_perfil_ref
   OR prop_asg.documento->'ambitos'->0 IS NULL
   OR NOT prop_asg.documento->'ambitos' @> pg_catalog.jsonb_build_array(
    pg_catalog.jsonb_build_object('clave','organizacion_ref',
     'valores',pg_catalog.jsonb_build_array(amb->>'organizacion_ref')))
  THEN RAISE EXCEPTION 'AUT68: organización de ADMIN distintos' USING ERRCODE='42501';END IF;
 END IF;
 rrhh_acuses:=vec_autorizacion.acreditar_cambios_rrhh_inscripcion_v1(
  plan,amb->>'organizacion_ref');
 ambcanon:='{"organizacion_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'organizacion_ref')
  ||',"unidad_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'unidad_ref')||'}';
 material_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_material,'UTF8')),'hex');
 ctx:='{"ambitos":'||ambcanon||',"atributos":{"material_sha256":"'||material_sha||'"}}';
 huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(ctx,'UTF8')),'hex');
 IF d->>'recurso_ref' IS DISTINCT FROM recurso
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
 OR cap->>'efecto_ref' IS DISTINCT FROM recurso
 OR cap->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT68: recurso o material no ligados' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_version_inscripcion_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM recurso
 OR x.huella_efecto_sha256 IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT68: consumo V3 divergente' USING ERRCODE='42501';END IF;
 administradores:=vec_autorizacion.administradores_version_inscripcion_v1();
 IF (SELECT pg_catalog.count(DISTINCT z->>'persona_ref') FROM pg_catalog.jsonb_array_elements(administradores) z)<2
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(administradores) z
  WHERE z->>'persona_ref'=persona AND z->>'perfil_ref'=perfil AND z->>'asignacion_ref'=asignacion)
 THEN RAISE EXCEPTION 'AUT68: dos ADMIN actuales necesarios' USING ERRCODE='42501';END IF;
 IF p_cierre THEN
  IF prop.material_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256'
  OR prop.proponente_persona_ref=persona OR prop.proponente_perfil_ref=perfil
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(administradores) z
   WHERE z->>'persona_ref'=prop.proponente_persona_ref
    AND z->>'perfil_ref'=prop.proponente_perfil_ref
    AND z->>'asignacion_ref'=prop.asignacion_ref)
  THEN RAISE EXCEPTION 'AUT68: cierre no independiente' USING ERRCODE='42501';END IF;
 ELSE
  SELECT * INTO prop FROM vec_autorizacion.propuesta_version_inscripcion_v1
   WHERE propuesta_ref=op FOR SHARE;
  IF FOUND THEN
   IF vec_autorizacion.solicitud_replay_version_inscripcion_v1(prop.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT68: propuesta repetida con otro material' USING ERRCODE='23505';END IF;
   replay:=true;
  END IF;
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec:admin:version-inscripcion:'||(plan#>>'{definicion_nueva,rol_id}'),0));
 IF p_cierre THEN
  SELECT * INTO prev FROM vec_autorizacion.cierre_version_inscripcion_v1
   WHERE propuesta_ref=prop.propuesta_ref;
  IF FOUND THEN
   IF prev.operacion_ref IS DISTINCT FROM op
   OR vec_autorizacion.solicitud_replay_version_inscripcion_v1(prev.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT68: cierre repetido con otro material' USING ERRCODE='23505';END IF;
   PERFORM vec_autorizacion.comprobar_postimagen_version_inscripcion_v1(prop.propuesta_ref);
   replay:=true;
  END IF;
 END IF;
 PERFORM vec_autorizacion.validar_plan_version_inscripcion_v1(plan,NOT replay);
 PERFORM vec_autorizacion.acreditar_cambios_empleado_inscripcion_v1(plan);
 PERFORM vec_autorizacion.acreditar_cambios_externos_inscripcion_v1(plan);
 IF p_cierre THEN
  IF replay THEN
   resultado:=prev.resultado||pg_catalog.jsonb_build_object('auditoria_acceso_ref',x.auditoria_ref);
  ELSE
   ahora:=pg_catalog.clock_timestamp();
   IF ahora>=prop.caduca_en
   THEN RAISE EXCEPTION 'AUT68: propuesta caducada' USING ERRCODE='40001';END IF;
   fecha:=vec_autorizacion.fecha_canonica_go_admin_v1(
    pg_catalog.to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),false);
   acto:='acto_admin:'||pg_catalog.substr(op,14,32);
   recibo_ref:='recibo_admin:'||pg_catalog.substr(op,14,32);
   rol:=plan->'definicion_nueva'||pg_catalog.jsonb_build_object(
    'estado','publicada','publicada_por',persona,'publicada_en',fecha,
    'retirada_en','0001-01-01T00:00:00Z');
   rol_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    vec_autorizacion.canon_version_rol_admin_v1(rol),'UTF8')),'hex');
   control:=pg_catalog.jsonb_build_object('version_rol_ref',prop.version_rol_ref,
    'revision',1,'estado','habilitada','actualizado_por',persona,'actualizado_en',fecha);
   control_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex');
   INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
    VALUES(prop.version_rol_ref,rol->>'rol_id',(rol->>'version')::bigint,rol_sha,ahora,rol);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol(
    version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
    VALUES(prop.version_rol_ref,1,'habilitada',control_sha,ahora,control);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(
    version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
    VALUES(prop.version_rol_ref,1,ahora,persona,acto);
   FOR cambio IN SELECT value FROM pg_catalog.jsonb_array_elements(plan->'asignaciones')
    WITH ORDINALITY z(value,n) ORDER BY n LOOP
    aid:=cambio->>'asignacion_id';externa:=cambio->>'almacen'='externo';
    pre:=cambio->'anterior';
    version_nueva:=CASE WHEN cambio->>'modo'='avance'
     THEN (pre#>>'{documento,version}')::bigint+1 ELSE 1 END;
    aref:='asignacion:'||aid||':v'||version_nueva;
    asg:=pg_catalog.jsonb_build_object('asignacion_id',aid,'version',version_nueva,
     'perfil_activo_ref',cambio->>'perfil_activo_ref','principal_id',cambio->>'principal_id',
     'version_rol_ref',prop.version_rol_ref,'estado','activa','ambitos',cambio->'ambitos',
     'vigente_desde',cambio->>'vigente_desde','vigente_hasta',cambio->>'vigente_hasta',
     'emitida_por',persona,'emitida_en',fecha,'revocada_en','0001-01-01T00:00:00Z');
    asha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
     vec_autorizacion.canon_asignacion_perfil_admin_v1(asg),'UTF8')),'hex');
    asignaciones_emitidas:=asignaciones_emitidas||pg_catalog.jsonb_build_array(
     pg_catalog.jsonb_build_object('posterior_ref',aref,
      'posterior_sha256',asha,'posterior_documento',asg)
     ||CASE WHEN cambio->>'modo'='avance' THEN pg_catalog.jsonb_build_object(
      'anterior_ref',pre->>'asignacion_ref','anterior_sha256',pre->>'huella_sha256',
      'unidad_acuse',rrhh_acuses->aid)
      ELSE '{}'::jsonb END);
    IF cambio->>'modo'='alta' THEN
     PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec:admin:asignacion-inscripcion:'||aid,0));
     IF EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id=aid)
      OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_id=aid)
      OR (externa AND (EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa
        WHERE perfil_activo_ref=cambio->>'perfil_activo_ref')
       OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa
        WHERE perfil_activo_ref=cambio->>'perfil_activo_ref')))
      OR (NOT externa AND (EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil
        WHERE perfil_activo_ref=cambio->>'perfil_activo_ref')
       OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual
        WHERE perfil_activo_ref=cambio->>'perfil_activo_ref')))
     THEN RAISE EXCEPTION 'AUT68: alta nominal ocupada' USING ERRCODE='40001';END IF;
    END IF;
    IF externa THEN
     INSERT INTO vec_autorizacion.asignacion_perfil_externa(
      asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
      version_rol_ref,huella_sha256,emitida_en,documento)
      VALUES(aref,aid,version_nueva,cambio->>'perfil_activo_ref',
       cambio->>'principal_id',prop.version_rol_ref,asha,ahora,asg);
     INSERT INTO vec_autorizacion.asignacion_perfil_actual_externa(
      perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
      VALUES(cambio->>'perfil_activo_ref',aref,ahora,persona,acto);
    ELSE
     IF cambio->>'modo'='avance' THEN
      SELECT * INTO STRICT ant FROM vec_autorizacion.asignacion_perfil
       WHERE asignacion_ref=pre->>'asignacion_ref' FOR SHARE;
      SELECT * INTO STRICT puntero FROM vec_autorizacion.asignacion_perfil_actual
       WHERE perfil_activo_ref=cambio->>'perfil_activo_ref' FOR UPDATE;
      IF puntero.asignacion_ref IS DISTINCT FROM ant.asignacion_ref
       OR ant.documento IS DISTINCT FROM pre->'documento'
       OR ant.huella_sha256 IS DISTINCT FROM pre->>'huella_sha256'
       OR ant.asignacion_id IS DISTINCT FROM aid
       OR ant.principal_id IS DISTINCT FROM cambio->>'principal_id'
       OR ant.perfil_activo_ref IS DISTINCT FROM cambio->>'perfil_activo_ref'
       OR ant.documento->'ambitos' IS DISTINCT FROM cambio->'ambitos'
       OR (ant.documento->>'vigente_desde')::timestamptz
          IS DISTINCT FROM (cambio->>'vigente_desde')::timestamptz
       OR (ant.documento->>'vigente_hasta')::timestamptz
          IS DISTINCT FROM (cambio->>'vigente_hasta')::timestamptz
      THEN RAISE EXCEPTION 'AUT68: CAS nominal divergente' USING ERRCODE='40001';END IF;
     END IF;
     INSERT INTO vec_autorizacion.asignacion_perfil(
      asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
      version_rol_ref,huella_sha256,emitida_en,documento)
      VALUES(aref,aid,version_nueva,cambio->>'perfil_activo_ref',
       cambio->>'principal_id',prop.version_rol_ref,asha,ahora,asg);
     IF cambio->>'modo'='avance' THEN
      UPDATE vec_autorizacion.asignacion_perfil_actual
       SET asignacion_ref=aref,actualizada_en=ahora,actualizada_por=persona,acto_ref=acto
       WHERE perfil_activo_ref=cambio->>'perfil_activo_ref'
        AND asignacion_ref=pre->>'asignacion_ref';
      IF NOT FOUND THEN RAISE EXCEPTION 'AUT68: CAS perdió puntero' USING ERRCODE='40001';END IF;
     ELSE
      INSERT INTO vec_autorizacion.asignacion_perfil_actual(
       perfil_activo_ref,asignacion_ref,actualizada_en,actualizada_por,acto_ref)
       VALUES(cambio->>'perfil_activo_ref',aref,ahora,persona,acto);
     END IF;
    END IF;
   END LOOP;
   recibo:=pg_catalog.jsonb_build_object('acto_ref',acto,'recibo_ref',recibo_ref,
    'actor_persona_ref',persona,'perfil_activo_ref',perfil,'asignacion_perfil_ref',asignacion,
    'correlacion_ref',m->>'correlacion_ref','motivo',motivo,
    'auditoria_ref',x.auditoria_ref,'version_rol',rol,'control_posterior',control,
    'asignaciones',asignaciones_emitidas);
   resultado:=pg_catalog.jsonb_build_object('operacion_ref',op,
    'material_canon',pg_catalog.convert_from(prop.material,'UTF8'),
    'propuesta_huella_sha256',prop.material_sha256,'decision','aprobada',
    'confirmado_en',ahora,'auditoria_acceso_ref',x.auditoria_ref,'recibo',recibo);
   INSERT INTO vec_autorizacion.cierre_version_inscripcion_v1 VALUES(
    prop.propuesta_ref,op,pg_catalog.convert_to(p_material,'UTF8'),
    prop.version_rol_ref,rol_sha,control_sha,persona,perfil,asignacion,
    x.auditoria_ref,resultado,ahora);
   INSERT INTO vec_autorizacion.outbox_version_inscripcion_v1
    VALUES(op,'version_publicada',x.auditoria_ref,ahora);
  END IF;
 ELSE
  IF replay THEN
   IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_version_inscripcion_v1
    WHERE propuesta_ref=op) THEN
    PERFORM vec_autorizacion.comprobar_postimagen_version_inscripcion_v1(op);
   ELSIF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol
    WHERE version_rol_ref=plan->>'version_rol_objetivo_ref') THEN
    RAISE EXCEPTION 'AUT68: versión objetivo ocupada por otra propuesta' USING ERRCODE='40001';
   END IF;
   resultado:=prop.resultado||pg_catalog.jsonb_build_object('auditoria_acceso_ref',x.auditoria_ref);
  ELSE
   ahora:=pg_catalog.clock_timestamp();
   caduca:=LEAST(ahora+interval '1 day',
    (admin_asg.documento->>'vigente_hasta')::timestamptz);
   resultado:=pg_catalog.jsonb_build_object('material_canon',m->>'material_canon',
    'huella_sha256',m->>'material_sha256','caduca_en',caduca,
    'auditoria_acceso_ref',x.auditoria_ref);
   INSERT INTO vec_autorizacion.propuesta_version_inscripcion_v1 VALUES(
    op,pg_catalog.convert_to(m->>'material_canon','UTF8'),m->>'material_sha256',
    pg_catalog.convert_to(p_material,'UTF8'),plan->>'perfil_objetivo',
    persona,perfil,asignacion,plan->>'catalogo_ref',
    (plan->>'catalogo_version')::integer,plan->>'catalogo_huella_sha256',
    recurso,ahora,caduca,x.auditoria_ref,resultado);
   INSERT INTO vec_autorizacion.outbox_version_inscripcion_v1
    VALUES(op,'version_propuesta',x.auditoria_ref,ahora);
  END IF;
 END IF;
 IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
  p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_version_inscripcion_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT68: decisión final revocada' USING ERRCODE='42501';END IF;
 PERFORM vec_autorizacion.validar_plan_version_inscripcion_v1(
  plan,CASE WHEN p_cierre THEN false ELSE NOT replay END);
 PERFORM vec_autorizacion.acreditar_cambios_empleado_inscripcion_v1(plan);
 PERFORM vec_autorizacion.acreditar_cambios_externos_inscripcion_v1(plan);
 PERFORM vec_autorizacion.acreditar_cambios_rrhh_inscripcion_v1(plan,amb->>'organizacion_ref');
 administradores:=vec_autorizacion.administradores_version_inscripcion_v1();
 IF (SELECT pg_catalog.count(DISTINCT z->>'persona_ref') FROM pg_catalog.jsonb_array_elements(administradores) z)<2
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(administradores) z
  WHERE z->>'persona_ref'=persona AND z->>'perfil_ref'=perfil AND z->>'asignacion_ref'=asignacion)
 OR (p_cierre AND NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(administradores) z
  WHERE z->>'persona_ref'=prop.proponente_persona_ref
   AND z->>'perfil_ref'=prop.proponente_perfil_ref
   AND z->>'asignacion_ref'=prop.asignacion_ref))
 OR (p_cierre AND NOT replay AND pg_catalog.clock_timestamp()>=prop.caduca_en)
 THEN RAISE EXCEPTION 'AUT68: doble control final no vigente' USING ERRCODE='42501';END IF;
 IF p_cierre AND NOT replay THEN
  PERFORM vec_autorizacion.revalidar_catalogo_cierre_inscripcion_v1(plan);
 END IF;
 RETURN resultado||pg_catalog.jsonb_build_object('estado','permitido','replay',replay);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_version_inscripcion_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

-- La excepción revierte consumo y efecto del subbloque. Sólo entonces se
-- asienta el intento fallido en la corriente común de auditoría de AD231.
CREATE FUNCTION vec_autorizacion.registrar_fallo_version_inscripcion_v1(
 p_material text,p_accion text,p_sqlstate text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;h text;corr text;estado text;codigo text;a record;
BEGIN
 h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  COALESCE(p_material,''),'UTF8')),'hex');
 corr:='correlacion_'||pg_catalog.substr(h,1,32);
 BEGIN
  m:=p_material::jsonb;
  IF pg_catalog.jsonb_typeof(m)='object'
   AND m->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$'
  THEN corr:=m->>'correlacion_ref';END IF;
 EXCEPTION WHEN OTHERS THEN NULL;
 END;
 estado:=CASE WHEN p_sqlstate IN('42501','22023','23505','P0002')
  THEN 'denegado' ELSE 'error' END;
 codigo:='version_inscripcion_'||estado;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_intento_version_inscripcion_v1(
  pg_catalog.jsonb_build_object('tipo_registro','intento_version_inscripcion',
   'evento_ref','evento_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'solicitud_sha256',h,
   'accion',p_accion,'recurso_ref','solicitud_version_inscripcion:'||pg_catalog.substr(h,1,32),
   'resultado',estado,'motivo_ref',codigo,'proceso','postgresql',
   'canal','operacion_tecnica_privada','finalidad_ref','gobierno_definiciones_perfiles',
   'correlacion_ref',corr));
 RETURN pg_catalog.jsonb_build_object('estado',estado,'codigo',codigo,
  'auditoria_intento',pg_catalog.jsonb_build_object(
   'auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
   'huella_sha256',a.huella_sha256,'correlacion_ref',a.correlacion_ref,
   'registrada_en',a.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_fallo_version_inscripcion_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.proponer_version_inscripcion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='35s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_version_inscripcion_v1(false,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_version_inscripcion_v1(p_material,
   'administracion.perfiles.version_inscripcion.proponer',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.cerrar_version_inscripcion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='35s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_version_inscripcion_v1(true,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_version_inscripcion_v1(p_material,
   'administracion.perfiles.version_inscripcion.aprobar',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cerrar_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_version_inscripcion_ejecutor;
GRANT EXECUTE ON FUNCTION
 vec_autorizacion.proponer_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.cerrar_version_inscripcion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_admin_version_inscripcion_ejecutor;
RESET ROLE;
COMMIT;
