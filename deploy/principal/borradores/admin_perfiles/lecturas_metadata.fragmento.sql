-- Fragmento privado para ensamblar en AUT24, antes de las siete fachadas.
-- No es una migración autónoma: no contiene BEGIN/COMMIT ni crea concesiones.
-- Dependencias: CA20, AUT24 y sus tablas/funciones privadas.
-- La metadata vinculacion_asignacion_admin_v1 permanece propiedad de AUT24;
-- este fragmento no la recrea ni amplía el documento canónico de asignación.
-- La fachada lectora debe consumir material V3 nuevo y revalidar el actor IS12
-- antes de llamar a proyectar_lectura_admin_interna_v1 en la misma transacción.
-- Las etiquetas visibles se localizan en Go/UI; CA no publica nombres.

RESET ROLE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.buscar_personas_admin_interna_v1(
 p_busqueda text,p_cursor text,p_limite integer
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE datos jsonb; siguiente text; cantidad integer; ahora timestamptz:=clock_timestamp();
BEGIN
 IF p_limite IS DISTINCT FROM 50 OR p_busqueda IS NULL
 OR octet_length(p_busqueda)>80 OR p_busqueda !~ '^per_[A-Za-z0-9_-]*$'
 OR (coalesce(p_cursor,'')<>'' AND vec_contexto_actor_v1.referencia_valida(p_cursor,'per_') IS NOT TRUE) THEN
  RAISE EXCEPTION 'AUT24: busqueda nominal invalida' USING ERRCODE='22023';
 END IF;
 -- Comparación literal de prefijo: '_' y '%' nunca son comodines de búsqueda.
 WITH pagina AS (
  SELECT x.persona_ref FROM vec_contexto_actor_v1.persona_actual a
  JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version)
  WHERE left(x.persona_ref,length(p_busqueda))=p_busqueda
   AND (coalesce(p_cursor,'')='' OR x.persona_ref COLLATE "C">p_cursor COLLATE "C")
   AND x.estado='activo' AND x.procedencia_autoridad='autoridad_maestra_acreditada'
   AND ahora>=x.vigente_desde AND ahora<x.vigente_hasta
  ORDER BY x.persona_ref COLLATE "C" LIMIT 51
 ), numerada AS (
  SELECT persona_ref,row_number() OVER(ORDER BY persona_ref COLLATE "C") n FROM pagina
 ) SELECT coalesce(jsonb_agg(jsonb_build_object('persona_ref',persona_ref,
    'nombre','','unidad_nombre','','unidad_clave_i18n','') ORDER BY persona_ref COLLATE "C")
    FILTER(WHERE n<=50),'[]'::jsonb),count(*),max(persona_ref) FILTER(WHERE n=50)
  INTO datos,cantidad,siguiente FROM numerada;
 RETURN jsonb_build_object('personas',datos,'siguiente_cursor',CASE WHEN cantidad>50 THEN siguiente ELSE '' END);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.buscar_personas_admin_interna_v1(text,text,integer)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.buscar_personas_admin_interna_v1(text,text,integer)
 TO vec_autorizacion_propietario;

-- Devuelve solo referencias/versiones/procedencias técnicas a AUT owner.
-- No permite elegir una cuenta ajena: los contextos proceden del vínculo CA.
CREATE FUNCTION vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(p_persona text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE pe record; contextos jsonb; perfiles jsonb; ahora timestamptz:=clock_timestamp();
BEGIN
 IF vec_contexto_actor_v1.referencia_valida(p_persona,'per_') IS NOT TRUE THEN
  RAISE EXCEPTION 'AUT24: persona nominal invalida' USING ERRCODE='22023';
 END IF;
 SELECT x.* INTO pe FROM vec_contexto_actor_v1.persona_actual a
 JOIN vec_contexto_actor_v1.persona_versiones x USING(persona_ref,version)
 WHERE a.persona_ref=p_persona AND x.estado='activo'
 AND x.procedencia_autoridad='autoridad_maestra_acreditada'
 AND ahora>=x.vigente_desde AND ahora<x.vigente_hasta;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: persona no disponible' USING ERRCODE='P0002'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.perfil_ref),'[]'::jsonb) INTO perfiles
 FROM (SELECT x.perfil_ref,x.version,x.persona_ref,x.estado,x.vigente_desde,x.vigente_hasta,
  x.procedencia_ref,x.procedencia_version,x.procedencia_huella_sha256,x.procedencia_autoridad
  FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version)
  WHERE x.persona_ref=p_persona AND x.procedencia_autoridad='autoridad_maestra_acreditada'
  ORDER BY x.perfil_ref LIMIT 50) t;
 SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.cuenta_ref,t.vinculo_ref),'[]'::jsonb) INTO contextos
 FROM (SELECT c.cuenta_ref,c.version AS cuenta_version,c.procedencia_ref,
  c.procedencia_version,c.procedencia_huella_sha256,c.vigente_hasta AS cuenta_vigente_hasta,
  v.vinculo_ref,v.version AS vinculo_version,v.perfil_ref,p.version AS perfil_version
  FROM vec_contexto_actor_v1.vinculo_contexto_actual a
  JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c ON c.cuenta_ref=ca.cuenta_ref AND c.version=ca.version
  JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
  JOIN vec_contexto_actor_v1.perfil_versiones p ON p.perfil_ref=pa.perfil_ref AND p.version=pa.version
  WHERE v.persona_ref=p_persona AND p.persona_ref=p_persona
  AND v.estado='activo' AND c.estado='activo' AND p.estado='activo'
  AND v.procedencia_autoridad='autoridad_maestra_acreditada'
  AND c.procedencia_autoridad='autoridad_maestra_acreditada'
  AND p.procedencia_autoridad='autoridad_maestra_acreditada'
  AND ahora>=GREATEST(v.vigente_desde,c.vigente_desde,p.vigente_desde)
  AND ahora<LEAST(v.vigente_hasta,c.vigente_hasta,p.vigente_hasta)
  ORDER BY c.cuenta_ref,v.vinculo_ref LIMIT 51) t;
 -- Cardinalidad excesiva no produce una selección arbitraria de cuenta.
 IF jsonb_array_length(contextos)>50 THEN contextos:='[]'::jsonb; END IF;
 RETURN jsonb_build_object('persona',to_jsonb(pe),'perfiles',perfiles,'contextos',contextos);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(text)
 TO vec_autorizacion_propietario;
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Prepara una alta con fuente actual. No inventa motivo, procedencia ni unidad.
-- Es privada y no se devuelve como acto disponible hasta ligar un motivo ADMIN.
-- La baja permanece cerrada: CA20 exige una procedencia NUEVA de revocación.
CREATE FUNCTION vec_autorizacion.objetivo_alta_admin_lectura_interna_v1(m jsonb,p_rol text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE fuente jsonb; cuenta jsonb; cfg record; o jsonb; acto jsonb; pi jsonb;
 unidad text; cantidad integer; fecha timestamptz; revision bigint; efectivos jsonb;
BEGIN
 IF m->>'actor_persona_ref' IS NOT DISTINCT FROM m->>'persona_ref' THEN RETURN NULL; END IF;
 SELECT a.* INTO cfg FROM vec_autorizacion.rol_administrable_exacto_v1 a
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
 WHERE a.version_rol_ref=p_rol AND a.huella_sha256=r.huella_sha256
 AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
 AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta;
 IF NOT FOUND THEN RETURN NULL; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x
  WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN RETURN NULL; END IF;
 IF cfg.clase<>'ordinario' AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)<2 THEN RETURN NULL; END IF;
 fuente:=vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(m->>'persona_ref');
 SELECT count(DISTINCT x->>'cuenta_ref') INTO cantidad FROM jsonb_array_elements(fuente->'contextos') x;
 IF cantidad<>1 THEN RETURN NULL; END IF;
 SELECT x INTO cuenta FROM jsonb_array_elements(fuente->'contextos') x ORDER BY x->>'vinculo_ref' LIMIT 1;
 IF cfg.clase='administrador' AND (vec_identidad_sesiones_v1.estado_admin_interno_v1(m->>'persona_ref',cuenta->>'cuenta_ref') IS NULL
  OR EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x WHERE x->>'persona_ref'=m->>'persona_ref')) THEN RETURN NULL; END IF;
 -- No usa el ámbito aportado por el cliente ni una configuración ambiental.
 IF cfg.unidad_requerida THEN
  SELECT count(DISTINCT u.unidad),min(u.unidad) INTO cantidad,unidad
  FROM vec_autorizacion.asignacion_perfil_actual ac
  JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  CROSS JOIN LATERAL jsonb_array_elements(a.documento->'ambitos') ab
  CROSS JOIN LATERAL jsonb_array_elements_text(ab->'valores') u(unidad)
  WHERE a.principal_id=m->>'persona_ref' AND a.documento->>'estado'='activa'
  AND ab->>'clave'='unidad' AND clock_timestamp()>=(a.documento->>'vigente_desde')::timestamptz
  AND clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz;
  IF cantidad<>1 OR vec_autorizacion.texto_positivo_valido(unidad,512) IS NOT TRUE THEN RETURN NULL; END IF;
 END IF;
 SELECT c.revision INTO revision FROM vec_autorizacion.control_continuidad_admin c WHERE control_id AND bootstrap_estado='consumido';
 IF NOT FOUND THEN RETURN NULL; END IF;
 fecha:=LEAST(cfg.vigente_hasta,(cuenta->>'cuenta_vigente_hasta')::timestamptz,(fuente#>>'{persona,vigente_hasta}')::timestamptz);
 IF fecha<=clock_timestamp() THEN RETURN NULL; END IF;
 o:=jsonb_build_object('cuenta_ref',cuenta->>'cuenta_ref','cuenta_version',cuenta->'cuenta_version',
  'persona_ref',m->>'persona_ref','persona_version',fuente#>'{persona,version}',
  'perfil_ref','prf_'||replace(gen_random_uuid()::text,'-',''),'perfil_version',0,
  'vinculo_ref','vca_'||replace(gen_random_uuid()::text,'-',''),'vinculo_version',0,
  'huella_sha256','','revision_continuidad',revision,'procedencia_ref',cuenta->>'procedencia_ref',
  'procedencia_version',cuenta->'procedencia_version','procedencia_huella_sha256',cuenta->>'procedencia_huella_sha256',
  'vigente_hasta',fecha);
 acto:=jsonb_build_object('operacion','otorgar','rol_version_ref',p_rol,'objetivo',o);
 IF unidad IS NOT NULL THEN acto:=acto||jsonb_build_object('unidad_ref',unidad); END IF;
 pi:=vec_autorizacion.preimagen_cambio_admin_interna_v1(acto);
 o:=jsonb_set(o,'{huella_sha256}',to_jsonb(encode(sha256(convert_to(pi::text,'UTF8')),'hex')));
 -- La unidad viaja en el DTO de lectura; al escribir está en el material raíz.
 IF unidad IS NOT NULL THEN o:=o||jsonb_build_object('unidad_ref',unidad); END IF;
 RETURN o;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.objetivo_alta_admin_lectura_interna_v1(jsonb,text)
 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;

CREATE FUNCTION vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(p_propuesta text,m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE p record; efectivos jsonb; continuidad record; vigente boolean; independiente boolean; motivos jsonb:='[]'::jsonb;
BEGIN
 SELECT * INTO p FROM vec_autorizacion.propuesta_perfil_sensible WHERE propuesta_ref=p_propuesta;
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT24: propuesta no disponible' USING ERRCODE='P0002'; END IF;
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 SELECT * INTO STRICT continuidad FROM vec_autorizacion.control_continuidad_admin WHERE control_id;
 independiente:=m->>'actor_persona_ref' IS DISTINCT FROM p.proponente_persona_ref
  AND m->>'actor_persona_ref' IS DISTINCT FROM p.objetivo_persona_ref;
 vigente:=continuidad.bootstrap_estado='consumido' AND continuidad.revision=p.revision_continuidad_esperada
  AND clock_timestamp()<p.caduca_en
  AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.cierre_propuesta_perfil_sensible WHERE propuesta_ref=p.propuesta_ref)
  AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(efectivos) x)>=2
  AND EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) x
   WHERE x->>'persona_ref'=p.proponente_persona_ref AND x->>'perfil_ref'=p.proponente_perfil_ref)
  AND EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 a
   JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
   WHERE a.version_rol_ref=p.version_rol_ref AND a.huella_sha256=r.huella_sha256
    AND r.documento->>'estado'='publicada' AND cv.estado='habilitada'
    AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta);
 -- AUT24 no publica una vinculación nominal de motivos ADMIN de cierre.
 -- Una lista de todas las entradas del catálogo no acredita su admisión aquí.
 RETURN jsonb_build_object('propuesta_ref',p.propuesta_ref,'proponente_persona_ref',p.proponente_persona_ref,
  'objetivo_persona_ref',p.objetivo_persona_ref,'objetivo_nombre','','rol_version_ref',p.version_rol_ref,
  'operacion',p.operacion,'huella_sha256',p.huella_sha256,'caduca_en',p.caduca_en,
  'puede_cerrar',independiente AND vigente AND jsonb_array_length(motivos)>0,'motivos_cierre',motivos);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(text,jsonb)
 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;

CREATE FUNCTION vec_autorizacion.proyectar_lectura_admin_interna_v1(m jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE fuente jsonb; datos jsonb; historia jsonb; acciones jsonb; recibo jsonb;
BEGIN
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR m->>'esquema' IS DISTINCT FROM 'administracion_perfiles_lectura_v1'
 OR m->>'limite' IS DISTINCT FROM '50' OR jsonb_path_exists(m,'$.** ? (@ == null)') THEN
  RAISE EXCEPTION 'AUT24: proyeccion nominal invalida' USING ERRCODE='22023';
 END IF;
 -- Defensa interna adicional; solo las siete fachadas tienen EXECUTE runtime.
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x
 WHERE x->>'persona_ref'=m->>'actor_persona_ref' AND x->>'perfil_ref'=m->>'actor_perfil_ref') THEN
  RAISE EXCEPTION 'AUT24: lector no efectivo' USING ERRCODE='42501';
 END IF;
 CASE m->>'consulta'
 WHEN 'capacidades' THEN
  SELECT vr.documento INTO STRICT fuente FROM vec_autorizacion.asignacion_perfil_actual ac
  JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
  JOIN vec_autorizacion.version_rol vr USING(version_rol_ref)
  JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
  JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
  WHERE a.perfil_activo_ref=m->>'actor_perfil_ref' AND a.principal_id=m->>'actor_persona_ref'
   AND a.documento->>'estado'='activa' AND vr.documento->>'estado'='publicada' AND cv.estado='habilitada'
   AND clock_timestamp()>=(a.documento->>'vigente_desde')::timestamptz
   AND clock_timestamp()<(a.documento->>'vigente_hasta')::timestamptz;
  WITH concesiones AS (SELECT x->>'accion' accion FROM jsonb_array_elements(fuente->'concesiones') x
   WHERE x->>'modulo_id'='administracion' AND x->'finalidades' ? 'gestion_perfiles'),
  disponibles AS (
   SELECT 'consultar' codigo WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.consultar')
   UNION ALL SELECT 'aplicar_ordinario' WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.otorgar')
    AND EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.revocar')
   UNION ALL SELECT 'proponer' WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.proponer')
    AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x)>=2
   UNION ALL SELECT 'cerrar_propuesta' WHERE EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.aprobar')
    AND EXISTS(SELECT 1 FROM concesiones WHERE accion='administracion.perfiles.rechazar')
    AND (SELECT count(DISTINCT x->>'persona_ref') FROM jsonb_array_elements(vec_autorizacion.administradores_efectivos_internos_v1()) x)>=2
  ) SELECT coalesce(jsonb_agg(codigo ORDER BY codigo),'[]'::jsonb) INTO acciones FROM disponibles;
  RETURN jsonb_build_object('version','v1','actor_persona_ref',m->>'actor_persona_ref','acciones',acciones);
 WHEN 'buscar_personas' THEN
  RETURN vec_contexto_actor_v1.buscar_personas_admin_interna_v1(m->>'busqueda',coalesce(m->>'cursor',''),50);
 WHEN 'consultar_persona' THEN
  fuente:=vec_contexto_actor_v1.metadatos_persona_admin_interna_v1(m->>'persona_ref');
  SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.perfil_ref),'[]'::jsonb) INTO datos
  FROM (SELECT a.perfil_activo_ref AS perfil_ref,a.version_rol_ref AS rol_version_ref,
   p->>'estado' AS estado,(p->>'version')::numeric AS version,p->>'vigente_hasta' AS vigente_hasta
   FROM vec_autorizacion.asignacion_perfil_actual ac
   JOIN vec_autorizacion.asignacion_perfil a USING(asignacion_ref)
   JOIN LATERAL jsonb_array_elements(fuente->'perfiles') p ON p->>'perfil_ref'=a.perfil_activo_ref
   WHERE a.principal_id=m->>'persona_ref' ORDER BY a.perfil_activo_ref LIMIT 50) t;
  SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.confirmado_en DESC,t.acto_ref),'[]'::jsonb) INTO historia
  FROM (SELECT rr->>'acto_ref' AS acto_ref,coalesce(material->>'operacion',propuesta.operacion) AS operacion,
   rr->>'estado_posterior' AS estado,(rr->>'confirmado_en')::timestamptz AS confirmado_en
   FROM vec_autorizacion.registro_acto_admin_v1 a
   CROSS JOIN LATERAL (SELECT convert_from(a.material,'UTF8')::jsonb AS material,
    CASE WHEN jsonb_typeof(a.resultado->'recibo')='object' THEN a.resultado->'recibo' ELSE a.resultado END AS rr) proyeccion
   LEFT JOIN vec_autorizacion.propuesta_perfil_sensible propuesta
    ON propuesta.propuesta_ref=material->>'propuesta_ref'
   WHERE rr->>'objetivo_persona_ref'=m->>'persona_ref' AND rr ? 'acto_ref'
   AND coalesce(material->>'operacion',propuesta.operacion) IN ('otorgar','revocar')
   ORDER BY (rr->>'confirmado_en')::timestamptz DESC,rr->>'acto_ref' LIMIT 50) t;
  RETURN jsonb_build_object('persona_ref',m->>'persona_ref','nombre','','unidad_nombre','',
   'perfiles',datos,'actos_disponibles','[]'::jsonb,'historia',historia);
 WHEN 'listar_roles' THEN
  SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.version_ref),'[]'::jsonb) INTO datos
  FROM (SELECT a.version_rol_ref AS version_ref,a.clase,vr.documento->>'nombre' AS clave_i18n,
   ''::text AS etiqueta,a.huella_sha256 FROM vec_autorizacion.rol_administrable_exacto_v1 a
   JOIN vec_autorizacion.version_rol vr USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
   JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
   WHERE a.huella_sha256=vr.huella_sha256 AND vr.documento->>'estado'='publicada' AND cv.estado='habilitada'
   AND clock_timestamp()>=a.vigente_desde AND clock_timestamp()<a.vigente_hasta
   ORDER BY a.version_rol_ref LIMIT 50) t;
  RETURN jsonb_build_object('roles',datos);
 WHEN 'listar_propuestas' THEN
  SELECT coalesce(jsonb_agg(vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(t.propuesta_ref,m)
   ORDER BY t.creada_en DESC,t.propuesta_ref),'[]'::jsonb) INTO datos
  FROM (SELECT p.propuesta_ref,p.creada_en FROM vec_autorizacion.propuesta_perfil_sensible p
   WHERE NOT EXISTS(SELECT 1 FROM vec_autorizacion.cierre_propuesta_perfil_sensible c WHERE c.propuesta_ref=p.propuesta_ref)
   ORDER BY p.creada_en DESC,p.propuesta_ref LIMIT 50) t;
  RETURN jsonb_build_object('propuestas',datos);
 WHEN 'consultar_propuesta' THEN
  RETURN vec_autorizacion.proyectar_propuesta_admin_lectura_interna_v1(m->>'propuesta_ref',m);
 WHEN 'consultar_recibo' THEN
  SELECT t.rr INTO STRICT recibo FROM vec_autorizacion.registro_acto_admin_v1 a
   CROSS JOIN LATERAL (SELECT CASE WHEN jsonb_typeof(a.resultado->'recibo')='object'
    THEN a.resultado->'recibo' ELSE a.resultado END AS rr) t
   WHERE t.rr->>'recibo_ref'=m->>'recibo_ref' AND t.rr ? 'acto_ref';
  -- STRICT rechaza una referencia ausente (P0002) o ambigua.
  RETURN recibo;
 ELSE RAISE EXCEPTION 'AUT24: consulta desconocida' USING ERRCODE='22023';
 END CASE;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proyectar_lectura_admin_interna_v1(jsonb)
 FROM PUBLIC,vec_admin_perfiles_ejecutor,vec_autorizacion_fuente;
-- Ningún GRANT al runtime: el ensamblador publica únicamente siete wrappers.
