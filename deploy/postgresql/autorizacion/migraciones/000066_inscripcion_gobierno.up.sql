\set ON_ERROR_STOP on
-- AUT66: estructura inmutable y puerta nominal del gobierno de inscripción.
-- No publica roles, asignaciones ni concesiones. AUT68 incorpora el efecto
-- tras el consumidor AD231 y el mantenimiento ADMIN AUT67.
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
 OR pg_catalog.to_regprocedure('vec_autorizacion.canon_version_rol_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.canon_asignacion_perfil_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.json_cadena_canonica_go_admin_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.exigir_rol_ordinario_gobierno_v1(jsonb,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(jsonb)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.propuesta_version_rol_bolsa_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.catalogo_accion_nominal_v1') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_externa') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.propuesta_version_inscripcion_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT66: preimagen AUT62/AUT65 incompatible' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE TABLE vec_autorizacion.propuesta_version_inscripcion_v1(
 propuesta_ref text PRIMARY KEY CHECK(propuesta_ref ~ '^propuesta_admin:[0-9a-f]{32}$'),
 material bytea NOT NULL CHECK(pg_catalog.octet_length(material) BETWEEN 1 AND 196608),
 material_sha256 text NOT NULL CHECK(material_sha256=pg_catalog.encode(pg_catalog.sha256(material),'hex')),
 solicitud bytea NOT NULL CHECK(pg_catalog.octet_length(solicitud) BETWEEN 1 AND 262144),
 perfil_objetivo text NOT NULL CHECK(perfil_objetivo IN ('externo','empleado','rrhh')),
 proponente_persona_ref text NOT NULL,proponente_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 catalogo_ref text NOT NULL,catalogo_version integer NOT NULL,catalogo_sha256 text NOT NULL,
 version_rol_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,caduca_en timestamptz(6) NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,
 CHECK(pg_catalog.isfinite(creada_en) AND pg_catalog.isfinite(caduca_en)
  AND caduca_en>creada_en AND caduca_en<=creada_en+interval '1 day'),
 FOREIGN KEY(catalogo_ref,catalogo_version,catalogo_sha256)
  REFERENCES vec_autorizacion.registro_catalogo_acciones_admin_v1(catalogo_ref,version,huella_sha256)
);
CREATE TABLE vec_autorizacion.cierre_version_inscripcion_v1(
 propuesta_ref text PRIMARY KEY REFERENCES vec_autorizacion.propuesta_version_inscripcion_v1(propuesta_ref),
 operacion_ref text NOT NULL UNIQUE CHECK(operacion_ref ~ '^cierre_admin:[0-9a-f]{32}$'),
 solicitud bytea NOT NULL CHECK(pg_catalog.octet_length(solicitud) BETWEEN 1 AND 65536),
 version_rol_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 rol_sha256 text NOT NULL CHECK(rol_sha256 ~ '^[0-9a-f]{64}$'),
 control_sha256 text NOT NULL CHECK(control_sha256 ~ '^[0-9a-f]{64}$'),
 aprobador_persona_ref text NOT NULL,aprobador_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,
 confirmado_en timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(confirmado_en))
);
CREATE TABLE vec_autorizacion.outbox_version_inscripcion_v1(
 operacion_ref text PRIMARY KEY,
 evento text NOT NULL CHECK(evento IN('version_propuesta','version_publicada')),
 auditoria_ref text NOT NULL,creada_en timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(creada_en))
);

ALTER TABLE vec_autorizacion.propuesta_version_inscripcion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.propuesta_version_inscripcion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.propuesta_version_inscripcion_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.propuesta_version_inscripcion_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.propuesta_version_inscripcion_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.propuesta_version_inscripcion_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.propuesta_version_inscripcion_v1 FROM PUBLIC;

ALTER TABLE vec_autorizacion.cierre_version_inscripcion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.cierre_version_inscripcion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.cierre_version_inscripcion_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.cierre_version_inscripcion_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.cierre_version_inscripcion_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.cierre_version_inscripcion_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.cierre_version_inscripcion_v1 FROM PUBLIC;

ALTER TABLE vec_autorizacion.outbox_version_inscripcion_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.outbox_version_inscripcion_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.outbox_version_inscripcion_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.outbox_version_inscripcion_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.outbox_version_inscripcion_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.outbox_version_inscripcion_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.outbox_version_inscripcion_v1 FROM PUBLIC;

-- Canon Go de un solo plan. El orden de campos coincide con los tipos de
-- dominio; jsonb::text jamás se usa como representación de huella.
CREATE FUNCTION vec_autorizacion.canon_version_inscripcion_v1(p jsonb,t text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE campos text[];tipos text[];opc text[]:=ARRAY[]::text[];k text;v jsonb;
 s text;salida text:='';pieza jsonb;lista text;i integer;
BEGIN
 CASE t
 WHEN 'material' THEN
  campos:=ARRAY['OperacionRef','ProponentePersonaRef','PerfilActivoRef','AsignacionPerfilRef','Plan'];
  tipos:=ARRAY['string','string','string','string','plan'];
 WHEN 'plan' THEN
  campos:=ARRAY['operacion','perfil_objetivo','catalogo_ref','catalogo_version',
   'catalogo_huella_sha256','version_rol_objetivo_ref','base','definicion_nueva',
   'selecciones','asignaciones','motivo','referencia_acto'];
  tipos:=ARRAY['string','string','string','int','string','string','base',
   'definicion','selecciones','asignaciones','motivo','referencia_acto'];
  opc:=ARRAY['base','referencia_acto'];
 WHEN 'base' THEN
  campos:=ARRAY['rol','control_vigencia','tipo_perfil'];
  tipos:=ARRAY['rol','control','string'];
 WHEN 'definicion' THEN
  campos:=ARRAY['rol_id','version','nombre','concesiones'];
  tipos:=ARRAY['string','int','nombre','concesiones'];
 WHEN 'cambio' THEN
  campos:=ARRAY['almacen','modo','asignacion_id','principal_id','perfil_activo_ref',
   'ambitos','vigente_desde','vigente_hasta','candidato_ref','empleado_ref',
   'proyeccion_empleado','fuente_unidad','anterior'];
  tipos:=ARRAY['string','string','string','string','string','ambitos','fecha','fecha',
   'string','string','proyeccion','fuente_unidad','anterior'];
  opc:=ARRAY['candidato_ref','empleado_ref','proyeccion_empleado','fuente_unidad','anterior'];
 WHEN 'proyeccion' THEN
  campos:=ARRAY['proyeccion_ref','version','procedencia_ref','procedencia_version',
   'procedencia_huella_sha256'];
  tipos:=ARRAY['string','uint','string','uint','string'];
 WHEN 'fuente_unidad' THEN
  campos:=ARRAY['referencia','version','huella_sha256'];
  tipos:=ARRAY['string','uint','string'];
 WHEN 'anterior' THEN
  campos:=ARRAY['asignacion_ref','huella_sha256','documento'];
  tipos:=ARRAY['string','string','asignacion'];
 ELSE RAISE EXCEPTION 'AUT66: tipo canon desconocido' USING ERRCODE='22023';
 END CASE;
 IF pg_catalog.jsonb_typeof(p) IS DISTINCT FROM 'object'
 OR pg_catalog.octet_length(p::text)>196608
 OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_object_keys(p) x WHERE NOT x=ANY(campos))
 THEN RAISE EXCEPTION 'AUT66: objeto canon inválido' USING ERRCODE='22023';END IF;
 FOR i IN 1..pg_catalog.cardinality(campos) LOOP
  k:=campos[i];v:=p->k;s:=NULL;
  IF k=ANY(opc) AND v IS NULL THEN CONTINUE;END IF;
  IF v IS NULL OR v='null'::jsonb
  THEN RAISE EXCEPTION 'AUT66: campo canon ausente' USING ERRCODE='22023';END IF;
  IF tipos[i] IN('string','nombre','fecha','referencia_acto') THEN
   IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'string'
    OR pg_catalog.octet_length(v#>>'{}') NOT BETWEEN 1 AND
      (CASE WHEN tipos[i]='referencia_acto' THEN 1024 ELSE 512 END)
    OR (tipos[i] NOT IN('nombre','referencia_acto')
       AND vec_autorizacion.texto_positivo_valido(v#>>'{}',512) IS NOT TRUE)
    OR (tipos[i]='nombre' AND ((v#>>'{}')<>pg_catalog.btrim(v#>>'{}')
       OR (v#>>'{}') ~ '[[:cntrl:]]'))
    OR (tipos[i]='referencia_acto' AND
       (pg_catalog.char_length(v#>>'{}')>256
        OR (v#>>'{}')<>pg_catalog.btrim(v#>>'{}')
        OR (v#>>'{}') ~ '^[[:space:]]|[[:space:]]$'
        OR (v#>>'{}') ~ '[[:cntrl:]]'
        OR pg_catalog.strpos(v#>>'{}',pg_catalog.chr(8232))>0
        OR pg_catalog.strpos(v#>>'{}',pg_catalog.chr(8233))>0))
   THEN RAISE EXCEPTION 'AUT66: cadena canon inválida' USING ERRCODE='22023';END IF;
   IF tipos[i]='fecha' THEN
    s:=vec_autorizacion.fecha_canonica_go_admin_v1(v#>>'{}',false);
   ELSE s:=v#>>'{}';END IF;
   s:=vec_autorizacion.json_cadena_canonica_go_admin_v1(s);
  ELSIF tipos[i] IN('int','uint') THEN
   IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'number'
    OR (v#>>'{}') !~ (CASE WHEN tipos[i]='int' THEN '^[1-9][0-9]{0,9}$'
      ELSE '^[1-9][0-9]{0,19}$' END)
    OR (v#>>'{}')::numeric > (CASE WHEN tipos[i]='int' THEN 2147483647::numeric
      ELSE 18446744073709551615::numeric END)
   THEN RAISE EXCEPTION 'AUT66: entero canon inválido' USING ERRCODE='22023';END IF;
   s:=v#>>'{}';
  ELSIF tipos[i] IN('selecciones','asignaciones','concesiones','ambitos') THEN
   IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'array'
    OR pg_catalog.jsonb_array_length(v) NOT BETWEEN 1 AND
      (CASE tipos[i] WHEN 'selecciones' THEN 6 WHEN 'asignaciones' THEN 16
       WHEN 'ambitos' THEN 8 ELSE 512 END)
    OR (tipos[i]='selecciones' AND pg_catalog.jsonb_array_length(v)<>
       (CASE WHEN p->>'perfil_objetivo'='rrhh' THEN 6 ELSE 5 END))
   THEN RAISE EXCEPTION 'AUT66: lista canon inválida' USING ERRCODE='22023';END IF;
   lista:='';
   FOR pieza IN SELECT value FROM pg_catalog.jsonb_array_elements(v) WITH ORDINALITY x(value,n) ORDER BY n LOOP
    IF lista<>'' THEN lista:=lista||',';END IF;
    lista:=lista||CASE tipos[i]
     WHEN 'selecciones' THEN vec_autorizacion.canon_gobierno_rol_nuevo_v1(pieza,'seleccion')
     WHEN 'asignaciones' THEN vec_autorizacion.canon_version_inscripcion_v1(pieza,'cambio')
     WHEN 'concesiones' THEN vec_autorizacion.objeto_canonico_go_admin_v1(pieza,'concesion')
     ELSE vec_autorizacion.objeto_canonico_go_admin_v1(pieza,'ambito') END;
   END LOOP;
   s:='['||lista||']';
  ELSIF tipos[i]='rol' THEN s:=vec_autorizacion.canon_version_rol_admin_v1(v);
  ELSIF tipos[i]='control' THEN s:=vec_autorizacion.canon_control_rol_admin_v1(v);
  ELSIF tipos[i]='asignacion' THEN s:=vec_autorizacion.canon_asignacion_perfil_admin_v1(v);
  ELSIF tipos[i]='motivo' THEN s:=vec_autorizacion.canon_gobierno_rol_nuevo_v1(v,'motivo');
  ELSE s:=vec_autorizacion.canon_version_inscripcion_v1(v,tipos[i]);END IF;
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(k)||':'||s;
 END LOOP;
 RETURN '{'||salida||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_version_inscripcion_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.acciones_perfil_inscripcion_v1(p_perfil text)
RETURNS text[] LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT CASE p_perfil
 WHEN 'externo' THEN ARRAY[
  'bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
  'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar',
  'bolsa.inscripcion.presentar']::text[]
 WHEN 'empleado' THEN ARRAY[
  'bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar',
  'bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar',
  'bolsa.inscripcion.presentar']::text[]
 WHEN 'rrhh' THEN ARRAY[
  'bolsa.inscripcion.rrhh.convocatorias.listar',
  'bolsa.inscripcion.rrhh.listar','bolsa.inscripcion.rrhh.consultar',
  'bolsa.inscripcion.rrhh.motivos','bolsa.inscripcion.rrhh.decidir',
  'bolsa.inscripcion.rrhh.incorporar']::text[]
 ELSE NULL::text[] END
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acciones_perfil_inscripcion_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.concesion_inscripcion_exacta_v1(c jsonb,p_perfil text)
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT pg_catalog.jsonb_typeof(c)='object'
 AND c->>'modulo_id'='bolsa' AND c->>'garantia_minima'='alto'
 AND vec_autorizacion.texto_positivo_valido(c->>'tipo_recurso',128) IS TRUE
 AND pg_catalog.strpos(c->>'tipo_recurso','*')=0
 AND COALESCE(c->'obligaciones','[]'::jsonb)='[]'::jsonb
 AND pg_catalog.jsonb_typeof(COALESCE(c->'campos_permitidos','[]'::jsonb))='array'
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements_text(
  COALESCE(c->'campos_permitidos','[]'::jsonb)) z(campo)
  WHERE pg_catalog.strpos(z.campo,'*')>0)
 AND (CASE WHEN c->>'accion' IN('bolsa.inscripcion.presentar',
   'bolsa.inscripcion.rrhh.decidir','bolsa.inscripcion.rrhh.incorporar')
   THEN COALESCE(c->'campos_permitidos','[]'::jsonb)='[]'::jsonb
   ELSE pg_catalog.jsonb_array_length(COALESCE(c->'campos_permitidos','[]'::jsonb))>0 END)
 AND (c->>'accion'<>'bolsa.inscripcion.rrhh.convocatorias.listar'
  OR c->'campos_permitidos'='["convocatorias[].convocatoria_ref", "convocatorias[].titulo", "convocatorias[].categorias_resumen", "convocatorias[].plazo_fin", "convocatorias[].estado_publicacion", "total", "cursor_siguiente"]'::jsonb)
 AND c->'finalidades'=pg_catalog.jsonb_build_array((CASE c->>'accion'
  WHEN 'bolsa.inscripcion.convocatorias.listar' THEN 'consulta_convocatoria_abierta'
  WHEN 'bolsa.inscripcion.convocatoria.consultar' THEN 'consulta_convocatoria_abierta'
  WHEN 'bolsa.inscripcion.propias.listar' THEN 'consulta_inscripcion_propia'
  WHEN 'bolsa.inscripcion.propia.consultar' THEN 'consulta_inscripcion_propia'
  WHEN 'bolsa.inscripcion.presentar' THEN 'presentar_inscripcion'
  WHEN 'bolsa.inscripcion.rrhh.listar' THEN 'consulta_inscripcion_rrhh'
  WHEN 'bolsa.inscripcion.rrhh.consultar' THEN 'consulta_inscripcion_rrhh'
  WHEN 'bolsa.inscripcion.rrhh.convocatorias.listar' THEN 'consulta_convocatorias_gestion_rrhh'
  WHEN 'bolsa.inscripcion.rrhh.motivos' THEN 'consulta_motivos_inscripcion_rrhh'
  WHEN 'bolsa.inscripcion.rrhh.decidir' THEN 'revisar_inscripcion'
  WHEN 'bolsa.inscripcion.rrhh.incorporar' THEN 'incorporar_inscripcion'
  ELSE NULL END))
 AND c->>'tipo_recurso'=(CASE
  WHEN p_perfil='externo' AND c->>'accion' IN
   ('bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar')
   THEN 'convocatoria_inscripcion'
  WHEN p_perfil='externo' AND c->>'accion' IN
   ('bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar')
   THEN 'solicitud_inscripcion'
  WHEN p_perfil='externo' AND c->>'accion'='bolsa.inscripcion.presentar'
   THEN 'inscripcion_convocatoria'
  WHEN p_perfil='empleado' AND c->>'accion' IN
   ('bolsa.inscripcion.convocatorias.listar','bolsa.inscripcion.convocatoria.consultar')
   THEN 'convocatoria_inscripcion_empleado'
  WHEN p_perfil='empleado' AND c->>'accion' IN
   ('bolsa.inscripcion.propias.listar','bolsa.inscripcion.propia.consultar')
   THEN 'solicitud_inscripcion_empleado'
  WHEN p_perfil='empleado' AND c->>'accion'='bolsa.inscripcion.presentar'
   THEN 'inscripcion_convocatoria_empleado'
  WHEN p_perfil='rrhh' AND c->>'accion'='bolsa.inscripcion.rrhh.motivos'
   THEN 'motivos_inscripcion'
  WHEN p_perfil='rrhh' AND c->>'accion'='bolsa.inscripcion.rrhh.convocatorias.listar'
   THEN 'conjunto_gestion_inscripcion'
  WHEN p_perfil='rrhh' AND c->>'accion' IN
   ('bolsa.inscripcion.rrhh.listar','bolsa.inscripcion.rrhh.consultar',
    'bolsa.inscripcion.rrhh.decidir','bolsa.inscripcion.rrhh.incorporar')
   THEN 'solicitud_inscripcion'
  ELSE NULL END);
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.concesion_inscripcion_exacta_v1(jsonb,text) FROM PUBLIC;

-- El catálogo aprobado es la única fuente de cada concesión. Hay cinco
-- selecciones propias o seis RRHH; las anteriores se conservan byte a byte.
-- La validación de empleado por CA39 y el CAS final pertenecen a AUT68.
CREATE FUNCTION vec_autorizacion.validar_plan_version_inscripcion_v1(p jsonb,p_primera boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE catalogo jsonb;base jsonb;nueva jsonb;seleccion jsonb;cambio jsonb;entrada jsonb;
 r record;cv record;a record;actual record;esperadas text[];
 rolid text;perfil text;modo text;almacen text;ahora timestamptz;
 nuevas jsonb:='[]'::jsonb;vistas text[]:=ARRAY[]::text[];aid text;anterior jsonb;
 dimensiones text[];entrada_dimensiones text[];cambio_dimensiones text[];
BEGIN
 PERFORM vec_autorizacion.canon_version_inscripcion_v1(p,'plan');
 perfil:=p->>'perfil_objetivo';rolid:=p#>>'{definicion_nueva,rol_id}';
 IF p_primera IS NULL OR p->>'operacion' NOT IN('crear','versionar')
 OR perfil NOT IN('externo','empleado','rrhh')
 OR rolid IS DISTINCT FROM (CASE perfil
  WHEN 'externo' THEN 'candidato_bolsa_portal_historial_propio_desarrollo'
  WHEN 'empleado' THEN 'empleado_bolsa_inscripcion_desarrollo'
  ELSE 'tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo' END)
 OR p->>'operacion' IS DISTINCT FROM (CASE perfil WHEN 'empleado' THEN 'crear' ELSE 'versionar' END)
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM
  'rol:'||rolid||':v'||(p#>>'{definicion_nueva,version}')
 OR pg_catalog.jsonb_array_length(p->'selecciones')<>
  pg_catalog.cardinality(vec_autorizacion.acciones_perfil_inscripcion_v1(perfil))
 OR pg_catalog.jsonb_array_length(p->'asignaciones') NOT BETWEEN 1 AND 16
 THEN RAISE EXCEPTION 'AUT66: plan de perfil inválido' USING ERRCODE='22023';END IF;
 IF (perfil='empleado' AND p ? 'base') OR (perfil<>'empleado' AND NOT p ? 'base')
 OR (perfil='empleado' AND p#>>'{definicion_nueva,version}'<>'1')
 OR (perfil<>'empleado' AND (p#>>'{definicion_nueva,version}')::integer<>
  (p#>>'{base,rol,version}')::integer+1)
 OR (perfil<>'empleado' AND p#>>'{base,rol,rol_id}' IS DISTINCT FROM rolid)
 OR (perfil<>'empleado' AND p#>>'{definicion_nueva,nombre}' IS DISTINCT FROM p#>>'{base,rol,nombre}')
 THEN RAISE EXCEPTION 'AUT66: base de rol inválida' USING ERRCODE='22023';END IF;
 IF p_primera THEN
  SELECT * INTO r FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR r.version::text IS DISTINCT FROM p->>'catalogo_version'
   OR r.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT66: cabeza de catálogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 catalogo:=pg_catalog.convert_from(vec_autorizacion.resolver_catalogo_acciones_administracion_v1(
  p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256'),'UTF8')::jsonb;
 ahora:=pg_catalog.clock_timestamp();
 IF p_primera AND (ahora<(catalogo->>'vigente_desde')::timestamptz OR
  (catalogo->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
   AND ahora>=(catalogo->>'vigente_hasta')::timestamptz))
 THEN RAISE EXCEPTION 'AUT66: catálogo no vigente' USING ERRCODE='42501';END IF;
 esperadas:=vec_autorizacion.acciones_perfil_inscripcion_v1(perfil);
 IF esperadas IS NULL THEN RAISE EXCEPTION 'AUT66: perfil desconocido' USING ERRCODE='42501';END IF;
 FOR seleccion IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'selecciones')
  WITH ORDINALITY x(value,n) ORDER BY n LOOP
  SELECT value INTO entrada FROM pg_catalog.jsonb_array_elements(catalogo->'entradas')
   WHERE value->>'referencia'=seleccion->>'entrada_ref';
  IF NOT FOUND OR entrada->>'version' IS DISTINCT FROM seleccion->>'entrada_version'
   OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    vec_autorizacion.canon_gobierno_rol_nuevo_v1(entrada,'entrada'),'UTF8')),'hex')
    IS DISTINCT FROM seleccion->>'entrada_huella_sha256'
   OR entrada->>'clase_control' IS DISTINCT FROM 'ordinario'
   OR entrada#>>'{concesion,modulo_id}' IS DISTINCT FROM 'bolsa'
   OR entrada#>>'{concesion,garantia_minima}' IS DISTINCT FROM 'alto'
   OR COALESCE(entrada#>'{concesion,obligaciones}','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
   OR vec_autorizacion.concesion_inscripcion_exacta_v1(entrada->'concesion',perfil) IS NOT TRUE
   OR (entrada#>>'{concesion,accion}'=ANY(esperadas)) IS NOT TRUE
   OR entrada#>>'{concesion,accion}'=ANY(vistas)
   OR (p_primera AND (ahora<(entrada->>'vigente_desde')::timestamptz OR
    (entrada->>'vigente_hasta'<>'0001-01-01T00:00:00Z'
     AND ahora>=(entrada->>'vigente_hasta')::timestamptz)))
  THEN RAISE EXCEPTION 'AUT66: descriptor ausente o divergente' USING ERRCODE='42501';END IF;
  SELECT pg_catalog.array_agg(x ORDER BY x) INTO entrada_dimensiones
   FROM pg_catalog.jsonb_array_elements_text(entrada->'dimensiones_ambito') z(x);
  IF entrada_dimensiones IS NULL OR pg_catalog.cardinality(entrada_dimensiones)=0
   OR (dimensiones IS NOT NULL AND entrada_dimensiones IS DISTINCT FROM dimensiones)
   OR entrada_dimensiones IS DISTINCT FROM (CASE perfil
    WHEN 'externo' THEN ARRAY['candidato_ref']::text[]
    WHEN 'empleado' THEN ARRAY['empleado_ref']::text[]
    ELSE ARRAY['ambito_ref','unidad_ref']::text[] END)
  THEN RAISE EXCEPTION 'AUT66: dimensiones de descriptores divergentes' USING ERRCODE='42501';END IF;
  dimensiones:=entrada_dimensiones;
  IF perfil='empleado' AND NOT EXISTS(
   SELECT 1 FROM pg_catalog.jsonb_array_elements(catalogo->'entradas') par(e)
   WHERE par.e->>'clase_control'='ordinario'
    AND par.e->'dimensiones_ambito'='["candidato_ref"]'::jsonb
    AND par.e->'concesion'=pg_catalog.jsonb_set(entrada->'concesion','{tipo_recurso}',
      pg_catalog.to_jsonb(pg_catalog.left(entrada#>>'{concesion,tipo_recurso}',
       pg_catalog.length(entrada#>>'{concesion,tipo_recurso}')-9)))
    AND vec_autorizacion.concesion_inscripcion_exacta_v1(par.e->'concesion','externo') IS TRUE
    AND (NOT p_primera OR (ahora>=(par.e->>'vigente_desde')::timestamptz
      AND (par.e->>'vigente_hasta'='0001-01-01T00:00:00Z'
       OR ahora<(par.e->>'vigente_hasta')::timestamptz))))
  THEN RAISE EXCEPTION 'AUT66: descriptor externo par ausente' USING ERRCODE='42501';END IF;
  vistas:=pg_catalog.array_append(vistas,entrada#>>'{concesion,accion}');
  nuevas:=nuevas||pg_catalog.jsonb_build_array(
   vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
    pg_catalog.jsonb_build_object('concesiones',pg_catalog.jsonb_build_array(entrada->'concesion')))
    #>'{concesiones,0}');
  IF perfil='empleado' THEN
   PERFORM vec_autorizacion.exigir_rol_ordinario_gobierno_v1(
    entrada,rolid,p#>>'{definicion_nueva,nombre}');
  END IF;
 END LOOP;
 IF vistas IS DISTINCT FROM esperadas THEN
  RAISE EXCEPTION 'AUT66: conjunto de acciones incompleto' USING ERRCODE='42501';END IF;
 nueva:=p->'definicion_nueva';
 IF perfil='empleado' THEN
  IF nueva->'concesiones' IS DISTINCT FROM nuevas
   OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=rolid))
  THEN RAISE EXCEPTION 'AUT66: rol empleado ocupado o concesiones divergentes' USING ERRCODE='40001';END IF;
 ELSE
  base:=p->'base';
  SELECT * INTO r FROM vec_autorizacion.version_rol
   WHERE version_rol_ref='rol:'||rolid||':v'||(base#>>'{rol,version}') FOR SHARE;
  IF NOT FOUND OR r.rol_id IS DISTINCT FROM rolid OR r.documento->>'estado'<>'publicada'
   OR r.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
   OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(r.documento)
      IS DISTINCT FROM base->'rol'
   OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
      WHERE z.rol_id=rolid AND z.version>r.version))
  THEN RAISE EXCEPTION 'AUT66: rol base divergente' USING ERRCODE='40001';END IF;
  SELECT x.* INTO cv FROM vec_autorizacion.control_vigencia_version_rol_actual q
   JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
   WHERE q.version_rol_ref=r.version_rol_ref FOR SHARE OF q,x;
  IF NOT FOUND OR cv.estado<>'habilitada'
   OR cv.documento IS DISTINCT FROM base->'control_vigencia'
   OR cv.huella_sha256 IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
    vec_autorizacion.canon_control_rol_admin_v1(cv.documento),'UTF8')),'hex')
   OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(catalogo->'perfiles') z
     WHERE z->'rol'=base->'rol' AND z->'control_vigencia'=base->'control_vigencia'
       AND z->>'tipo_perfil'=base->>'tipo_perfil')
   OR nueva->'concesiones' IS DISTINCT FROM (base#>'{rol,concesiones}')||nuevas
   OR EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(base#>'{rol,concesiones}') x
     WHERE x->>'accion'=ANY(vistas))
   OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol
      WHERE version_rol_ref=p->>'version_rol_objetivo_ref'))
  THEN RAISE EXCEPTION 'AUT66: concesiones o control base divergentes' USING ERRCODE='40001';END IF;
 END IF;
 vistas:=ARRAY[]::text[];
 FOR cambio IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'asignaciones')
  WITH ORDINALITY x(value,n) ORDER BY n LOOP
  modo:=cambio->>'modo';almacen:=cambio->>'almacen';aid:=cambio->>'asignacion_id';
  IF aid=ANY(vistas) OR vec_autorizacion.texto_positivo_valido(aid,512) IS NOT TRUE
   OR cambio->>'principal_id' !~ '^per_[A-Za-z0-9_-]{22,128}$'
   OR cambio->>'perfil_activo_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
   OR (cambio->>'vigente_desde')::timestamptz>ahora
   OR (cambio->>'vigente_hasta')::timestamptz<=ahora
  THEN RAISE EXCEPTION 'AUT66: asignación no vigente o repetida' USING ERRCODE='22023';END IF;
  SELECT pg_catalog.array_agg(e.value->>'clave' ORDER BY e.value->>'clave')
   INTO cambio_dimensiones FROM pg_catalog.jsonb_array_elements(cambio->'ambitos') e(value);
  IF cambio_dimensiones IS DISTINCT FROM dimensiones
  THEN RAISE EXCEPTION 'AUT66: ámbitos de asignación y catálogo divergentes' USING ERRCODE='42501';END IF;
  vistas:=pg_catalog.array_append(vistas,aid);
  IF perfil='externo' THEN
   IF almacen<>'externo' OR modo<>'alta' OR cambio ? 'anterior'
    OR cambio->>'candidato_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR cambio ? 'empleado_ref' OR cambio ? 'proyeccion_empleado' OR cambio ? 'fuente_unidad'
    OR cambio->'ambitos' IS DISTINCT FROM pg_catalog.jsonb_build_array(
      pg_catalog.jsonb_build_object('clave','candidato_ref',
       'valores',pg_catalog.jsonb_build_array(cambio->>'candidato_ref')))
    OR (p_primera AND vec_contexto_actor_v1.acreditar_candidato_externo_v1(
      cambio->>'principal_id',cambio->>'perfil_activo_ref',cambio->>'candidato_ref') IS NOT TRUE)
    OR (p_primera AND (EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa x
      WHERE x.asignacion_id=aid OR x.perfil_activo_ref=cambio->>'perfil_activo_ref')
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual_externa x
      WHERE x.perfil_activo_ref=cambio->>'perfil_activo_ref')
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil x WHERE x.asignacion_id=aid)))
   THEN RAISE EXCEPTION 'AUT66: alta externa no acreditada' USING ERRCODE='42501';END IF;
  ELSIF perfil='empleado' THEN
   IF almacen<>'normal' OR modo<>'alta' OR cambio ? 'anterior'
    OR cambio ? 'candidato_ref' OR cambio ? 'fuente_unidad' OR NOT cambio ? 'empleado_ref'
    OR NOT cambio ? 'proyeccion_empleado'
    OR cambio->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR cambio#>>'{proyeccion_empleado,proyeccion_ref}' !~ '^pep_[A-Za-z0-9_-]{22,128}$'
    OR cambio#>>'{proyeccion_empleado,procedencia_ref}' !~ '^prc_[A-Za-z0-9_-]{22,128}$'
    OR cambio#>>'{proyeccion_empleado,procedencia_huella_sha256}' !~ '^[0-9a-f]{64}$'
    OR cambio->'ambitos' IS DISTINCT FROM pg_catalog.jsonb_build_array(
      pg_catalog.jsonb_build_object('clave','empleado_ref',
       'valores',pg_catalog.jsonb_build_array(cambio->>'empleado_ref')))
    OR (p_primera AND (EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil x
      WHERE x.asignacion_id=aid OR x.perfil_activo_ref=cambio->>'perfil_activo_ref')
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual x
      WHERE x.perfil_activo_ref=cambio->>'perfil_activo_ref')
    OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa x WHERE x.asignacion_id=aid)))
   THEN RAISE EXCEPTION 'AUT66: alta empleado no nominal' USING ERRCODE='42501';END IF;
  ELSE
   IF almacen<>'normal' OR modo<>'avance' OR NOT cambio ? 'anterior'
    OR cambio ? 'candidato_ref' OR cambio ? 'empleado_ref' OR cambio ? 'proyeccion_empleado'
    OR NOT cambio ? 'fuente_unidad'
    OR cambio#>>'{fuente_unidad,referencia}' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR cambio#>>'{fuente_unidad,version}' !~ '^[1-9][0-9]{0,9}$'
    OR (cambio#>>'{fuente_unidad,version}')::numeric>2147483647
    OR cambio#>>'{fuente_unidad,huella_sha256}' !~ '^[0-9a-f]{64}$'
    OR cambio#>>'{fuente_unidad,huella_sha256}'=pg_catalog.repeat('0',64)
   THEN RAISE EXCEPTION 'AUT66: avance RRHH no nominal' USING ERRCODE='22023';END IF;
   anterior:=cambio->'anterior';
   IF anterior->>'asignacion_ref' IS DISTINCT FROM
     'asignacion:'||aid||':v'||(anterior#>>'{documento,version}')
    OR anterior#>>'{documento,asignacion_id}' IS DISTINCT FROM aid
    OR anterior#>>'{documento,principal_id}' IS DISTINCT FROM cambio->>'principal_id'
    OR anterior#>>'{documento,perfil_activo_ref}' IS DISTINCT FROM cambio->>'perfil_activo_ref'
    OR anterior#>'{documento,ambitos}' IS DISTINCT FROM cambio->'ambitos'
    OR (anterior#>>'{documento,vigente_desde}')::timestamptz
       IS DISTINCT FROM (cambio->>'vigente_desde')::timestamptz
    OR (anterior#>>'{documento,vigente_hasta}')::timestamptz
       IS DISTINCT FROM (cambio->>'vigente_hasta')::timestamptz
    OR anterior#>>'{documento,version_rol_ref}' NOT LIKE 'rol:'||rolid||':v%'
    OR anterior#>>'{documento,estado}' IS DISTINCT FROM 'activa'
    OR anterior->>'huella_sha256' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      vec_autorizacion.canon_asignacion_perfil_admin_v1(anterior->'documento'),'UTF8')),'hex')
   THEN RAISE EXCEPTION 'AUT66: preimagen RRHH divergente' USING ERRCODE='40001';END IF;
   SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil x
    WHERE x.asignacion_ref=anterior->>'asignacion_ref' FOR SHARE;
   SELECT q.* INTO actual FROM vec_autorizacion.asignacion_perfil_actual q
    WHERE q.perfil_activo_ref=cambio->>'perfil_activo_ref' FOR SHARE;
   IF a.asignacion_ref IS NULL OR a.huella_sha256 IS DISTINCT FROM anterior->>'huella_sha256'
    OR a.documento IS DISTINCT FROM anterior->'documento'
    OR (p_primera AND actual.asignacion_ref IS DISTINCT FROM a.asignacion_ref)
    OR (p_primera AND (ahora<(a.documento->>'vigente_desde')::timestamptz
     OR ahora>=(a.documento->>'vigente_hasta')::timestamptz
     OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil h
       WHERE h.asignacion_id=aid AND h.documento->>'estado'='revocada')))
   THEN RAISE EXCEPTION 'AUT66: CAS RRHH divergente' USING ERRCODE='40001';END IF;
  END IF;
 END LOOP;
 IF perfil IN('externo','empleado') AND pg_catalog.cardinality(vistas)<>1
 THEN RAISE EXCEPTION 'AUT66: alta de perfil debe ser única' USING ERRCODE='22023';END IF;
 RETURN pg_catalog.jsonb_build_object('perfil_objetivo',perfil,'rol_id',rolid,
  'version_rol_objetivo_ref',p->>'version_rol_objetivo_ref','concesiones_nuevas',nuevas);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_plan_version_inscripcion_v1(jsonb,boolean) FROM PUBLIC;

-- Segunda puerta para AD231: el ADMIN de aplicación ha de ser nominal,
-- vigente y titular de la concesión exacta publicada en su propia versión.
CREATE FUNCTION vec_autorizacion.acreditar_version_inscripcion_v1(d jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE r record;cv record;a record;meta record;c jsonb;accion text;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR d->>'accion' NOT IN('administracion.perfiles.version_inscripcion.proponer',
                         'administracion.perfiles.version_inscripcion.aprobar')
 OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'tipo_recurso' IS DISTINCT FROM (CASE d->>'accion'
  WHEN 'administracion.perfiles.version_inscripcion.proponer' THEN 'definicion_rol'
  ELSE 'propuesta_definicion_rol' END)
 OR d->>'finalidad' IS DISTINCT FROM 'gobierno_definiciones_perfiles'
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RETURN false;END IF;
 accion:=d->>'accion';
 c:=pg_catalog.jsonb_build_object('accion',accion,'modulo_id','administracion',
  'tipo_recurso',d->>'tipo_recurso','finalidades',pg_catalog.jsonb_build_array('gobierno_definiciones_perfiles'),
  'garantia_minima','alto','campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb);
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=d->>'version_rol_ref' FOR SHARE;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO cv FROM vec_autorizacion.control_vigencia_version_rol_actual q
 JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
 WHERE q.version_rol_ref=r.version_rol_ref FOR SHARE OF q,x;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual q
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE q.perfil_activo_ref=d->>'perfil_activo_ref' AND q.asignacion_ref=d->>'asignacion_ref'
 FOR SHARE OF q,x;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x
 WHERE x.version_rol_ref=r.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false;END IF;
 ahora:=pg_catalog.clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.documento->>'estado'='publicada'
 AND r.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 AND cv.estado='habilitada' AND cv.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  vec_autorizacion.canon_control_rol_admin_v1(cv.documento),'UTF8')),'hex')
 AND r.huella_sha256=d->>'version_rol_huella_sha256'
 AND cv.revision::text=d->>'control_vigencia_version_rol_revision'
 AND cv.huella_sha256=d->>'control_vigencia_version_rol_huella_sha256'
 AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema'
 AND meta.version_rol_huella_sha256=r.huella_sha256 AND meta.fuente_ref=r.version_rol_ref
 AND meta.fuente_version=r.version AND meta.fuente_huella_sha256=r.huella_sha256
 AND a.principal_id=d->>'principal_id' AND a.version_rol_ref=r.version_rol_ref
 AND a.huella_sha256=d->>'asignacion_huella_sha256'
 AND a.documento->>'estado'='activa' AND ahora>=(a.documento->>'vigente_desde')::timestamptz
 AND ahora<(a.documento->>'vigente_hasta')::timestamptz
 AND pg_catalog.jsonb_array_length(a.documento->'ambitos')=2
 AND (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_array_elements(a.documento->'ambitos') b
  WHERE b->>'clave' IN('organizacion_ref','unidad_ref')
   AND pg_catalog.jsonb_array_length(b->'valores')=1)=2
 AND (SELECT pg_catalog.count(DISTINCT b->>'clave')
  FROM pg_catalog.jsonb_array_elements(a.documento->'ambitos') b)=2
 AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n
  WHERE n.version_rol_ref=r.version_rol_ref AND n.accion_ref='accion:'||accion
  AND n.fuente_ref=r.version_rol_ref AND n.fuente_version=r.version
  AND n.fuente_huella_sha256=r.huella_sha256 AND n.concesion=c
  AND n.clase_control='administrador_aplicacion'
  AND n.dimensiones_ambito='["organizacion_ref"]'::jsonb
  AND ahora>=n.vigente_desde AND (n.vigente_hasta IS NULL OR ahora<n.vigente_hasta))
 AND EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(r.documento->'concesiones') x WHERE x=c);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_version_inscripcion_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_version_inscripcion_v1(jsonb)
 TO vec_autorizacion_atestada_v3_propietario;

RESET ROLE;
COMMIT;
