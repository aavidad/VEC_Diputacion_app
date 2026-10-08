\set ON_ERROR_STOP on
-- AUT62. Estructura inmutable y canon de la versión B1. No concede permisos
-- ni publica roles. AUT63 instala el efecto después del consumidor AD227.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion.canon_version_rol_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.canon_asignacion_perfil_admin_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text)') IS NULL
 OR to_regprocedure('vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)') IS NULL
 OR to_regclass('vec_autorizacion.propuesta_version_rol_bolsa_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT62: preimagen AUT61 incompatible' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE TABLE vec_autorizacion.propuesta_version_rol_bolsa_v1(
 propuesta_ref text PRIMARY KEY CHECK(propuesta_ref ~ '^propuesta_admin:[0-9a-f]{32}$'),
 material bytea NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 131072),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(material),'hex')),
 solicitud bytea NOT NULL CHECK(octet_length(solicitud) BETWEEN 1 AND 196608),
 proponente_persona_ref text NOT NULL,proponente_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 catalogo_ref text NOT NULL,catalogo_version integer NOT NULL,catalogo_sha256 text NOT NULL,
 version_rol_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,caduca_en timestamptz(6) NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,
 CHECK(isfinite(creada_en) AND isfinite(caduca_en) AND caduca_en>creada_en AND caduca_en<=creada_en+interval '1 day'),
 FOREIGN KEY(catalogo_ref,catalogo_version,catalogo_sha256)
 REFERENCES vec_autorizacion.registro_catalogo_acciones_admin_v1(catalogo_ref,version,huella_sha256)
);
CREATE TABLE vec_autorizacion.cierre_version_rol_bolsa_v1(
 propuesta_ref text PRIMARY KEY REFERENCES vec_autorizacion.propuesta_version_rol_bolsa_v1(propuesta_ref),
 operacion_ref text NOT NULL UNIQUE CHECK(operacion_ref ~ '^cierre_admin:[0-9a-f]{32}$'),
 solicitud bytea NOT NULL CHECK(octet_length(solicitud) BETWEEN 1 AND 65536),
 version_rol_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 rol_sha256 text NOT NULL CHECK(rol_sha256 ~ '^[0-9a-f]{64}$'),
 control_sha256 text NOT NULL CHECK(control_sha256 ~ '^[0-9a-f]{64}$'),
 aprobador_persona_ref text NOT NULL,aprobador_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,confirmado_en timestamptz(6) NOT NULL CHECK(isfinite(confirmado_en))
);
CREATE TABLE vec_autorizacion.outbox_version_rol_bolsa_v1(
 operacion_ref text PRIMARY KEY,evento text NOT NULL CHECK(evento IN('version_propuesta','version_publicada')),
 auditoria_ref text NOT NULL,creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en))
);
ALTER TABLE vec_autorizacion.propuesta_version_rol_bolsa_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.propuesta_version_rol_bolsa_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.propuesta_version_rol_bolsa_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.propuesta_version_rol_bolsa_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.propuesta_version_rol_bolsa_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.propuesta_version_rol_bolsa_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.propuesta_version_rol_bolsa_v1 FROM PUBLIC;

ALTER TABLE vec_autorizacion.cierre_version_rol_bolsa_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.cierre_version_rol_bolsa_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.cierre_version_rol_bolsa_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.cierre_version_rol_bolsa_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.cierre_version_rol_bolsa_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.cierre_version_rol_bolsa_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.cierre_version_rol_bolsa_v1 FROM PUBLIC;

ALTER TABLE vec_autorizacion.outbox_version_rol_bolsa_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.outbox_version_rol_bolsa_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_autorizacion.outbox_version_rol_bolsa_v1
 TO vec_autorizacion_propietario USING(current_user='vec_autorizacion_propietario')
 WITH CHECK(current_user='vec_autorizacion_propietario');
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.outbox_version_rol_bolsa_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.outbox_version_rol_bolsa_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable();
REVOKE ALL ON TABLE vec_autorizacion.outbox_version_rol_bolsa_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion.outbox_version_rol_bolsa_v1 FROM PUBLIC;

-- Serialización del tipo Go específico de B1. Cada documento anidado usa el
-- canon AUT24, por lo que la huella no depende de jsonb::text ni del orden
-- que PostgreSQL da a las claves de un objeto.
CREATE FUNCTION vec_autorizacion.canon_version_rol_bolsa_v1(p jsonb,t text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE s text:='';salida text:='';e jsonb;i integer;partes text[]:=ARRAY[]::text[];claves text[];
BEGIN
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>131072
 THEN RAISE EXCEPTION 'AUT62: canon invalido' USING ERRCODE='22023';END IF;
 CASE t
 WHEN 'material' THEN
  claves:=ARRAY['OperacionRef','ProponentePersonaRef','PerfilActivoRef','AsignacionPerfilRef','Plan'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'OperacionRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'ProponentePersonaRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'PerfilActivoRef'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'AsignacionPerfilRef'),
   vec_autorizacion.canon_version_rol_bolsa_v1(p->'Plan','plan')];
 WHEN 'plan' THEN
  claves:=ARRAY['operacion','catalogo_ref','catalogo_version','catalogo_huella_sha256',
   'version_rol_objetivo_ref','base','definicion_nueva','seleccion','asignaciones','motivo','referencia_acto'];
  IF p->>'operacion' IS DISTINCT FROM 'versionar'
   OR jsonb_typeof(p->'asignaciones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'asignaciones') NOT BETWEEN 1 AND 16
   OR (p->>'catalogo_version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT62: plan invalido' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.canon_version_rol_bolsa_v1(e,'asignacion');
  END LOOP;
  s:='['||s||']';
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'operacion'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_ref'),
   p->>'catalogo_version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'catalogo_huella_sha256'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'version_rol_objetivo_ref'),
   vec_autorizacion.canon_version_rol_bolsa_v1(p->'base','base'),
   vec_autorizacion.canon_version_rol_bolsa_v1(p->'definicion_nueva','definicion'),
   vec_autorizacion.canon_gobierno_rol_nuevo_v1(p->'seleccion','seleccion'),
   s,
   vec_autorizacion.canon_gobierno_rol_nuevo_v1(p->'motivo','motivo'),
   CASE WHEN coalesce(p->>'referencia_acto','')='' THEN NULL
    ELSE vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'referencia_acto') END];
 WHEN 'base' THEN
  claves:=ARRAY['rol','control_vigencia','tipo_perfil'];
  partes:=ARRAY[
   vec_autorizacion.canon_version_rol_admin_v1(p->'rol'),
   vec_autorizacion.canon_control_rol_admin_v1(p->'control_vigencia'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'tipo_perfil')];
 WHEN 'definicion' THEN
  claves:=ARRAY['rol_id','version','nombre','concesiones'];
  IF jsonb_typeof(p->'concesiones') IS DISTINCT FROM 'array'
   OR jsonb_array_length(p->'concesiones') NOT BETWEEN 2 AND 512
   OR (p->>'version') !~ '^[1-9][0-9]{0,8}$'
  THEN RAISE EXCEPTION 'AUT62: definicion invalida' USING ERRCODE='22023';END IF;
  FOR e IN SELECT value FROM jsonb_array_elements(p->'concesiones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
   IF s<>'' THEN s:=s||',';END IF;
   s:=s||vec_autorizacion.objeto_canonico_go_admin_v1(e,'concesion');
  END LOOP;
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'rol_id'),
   p->>'version',
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'nombre'),
   '['||s||']'];
 WHEN 'asignacion' THEN
  claves:=ARRAY['asignacion_ref','huella_sha256','documento'];
  partes:=ARRAY[
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'asignacion_ref'),
   vec_autorizacion.json_cadena_canonica_go_admin_v1(p->>'huella_sha256'),
   vec_autorizacion.canon_asignacion_perfil_admin_v1(p->'documento')];
 ELSE RAISE EXCEPTION 'AUT62: tipo canon desconocido' USING ERRCODE='22023';
 END CASE;
 IF t='plan' AND partes[11] IS NULL THEN
  IF p ? 'referencia_acto' THEN
   RAISE EXCEPTION 'AUT62: referencia de acto vacía' USING ERRCODE='22023';END IF;
  claves:=claves[1:10];partes:=partes[1:10];
 END IF;
 IF (SELECT count(*) FROM jsonb_object_keys(p))<>cardinality(claves)
 OR NOT (p ?& claves)
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(p) x WHERE NOT x=ANY(claves))
 OR array_position(partes,NULL) IS NOT NULL
 THEN RAISE EXCEPTION 'AUT62: campos canon divergentes' USING ERRCODE='22023';END IF;
 FOR i IN 1..cardinality(claves) LOOP
  IF partes[i] IS NULL THEN CONTINUE;END IF;
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(claves[i])||':'||partes[i];
 END LOOP;
 RETURN '{'||salida||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_version_rol_bolsa_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.acreditar_version_rol_bolsa_v1(d jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE r record;cv record;a record;meta record;c jsonb;accion text;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR d->>'accion' NOT IN('administracion.perfiles.version_bolsa.proponer','administracion.perfiles.version_bolsa.aprobar')
 OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'tipo_recurso' IS DISTINCT FROM (CASE d->>'accion'
  WHEN 'administracion.perfiles.version_bolsa.proponer' THEN 'definicion_rol' ELSE 'propuesta_definicion_rol' END)
 OR d->>'finalidad' IS DISTINCT FROM 'gobierno_definiciones_perfiles'
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RETURN false;END IF;
 accion:=d->>'accion';
 c:=jsonb_build_object('accion',accion,'modulo_id','administracion',
  'tipo_recurso',d->>'tipo_recurso','finalidades',jsonb_build_array('gobierno_definiciones_perfiles'),
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
 ahora:=clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.documento->>'estado'='publicada'
 AND r.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 AND cv.estado='habilitada' AND cv.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(cv.documento),'UTF8')),'hex')
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
 AND jsonb_array_length(a.documento->'ambitos')=2
 AND (SELECT count(*) FROM jsonb_array_elements(a.documento->'ambitos') b
  WHERE b->>'clave' IN('organizacion_ref','unidad_ref') AND jsonb_array_length(b->'valores')=1)=2
 AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n
  WHERE n.version_rol_ref=r.version_rol_ref AND n.accion_ref='accion:'||accion
  AND n.fuente_ref=r.version_rol_ref AND n.fuente_version=r.version
  AND n.fuente_huella_sha256=r.huella_sha256 AND n.concesion=c
  AND n.clase_control='administrador_aplicacion'
  AND n.dimensiones_ambito='["organizacion_ref"]'::jsonb
  AND ahora>=n.vigente_desde AND (n.vigente_hasta IS NULL OR ahora<n.vigente_hasta))
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') x WHERE x=c);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_version_rol_bolsa_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_version_rol_bolsa_v1(jsonb)
 TO vec_autorizacion_atestada_v3_propietario;

-- Valida el catálogo aprobado y la preimagen durable. Para replay sólo se
-- coteja la publicación histórica; una versión posterior no borra su recibo.
CREATE FUNCTION vec_autorizacion.validar_plan_version_rol_bolsa_v1(p jsonb,p_primera boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE c jsonb;e jsonb;seleccion jsonb;base jsonb;nueva jsonb;h record;r record;cv record;a record;
 t jsonb;vistas text[]:=ARRAY[]::text[];ahora timestamptz;original jsonb;
 concesion_b1 jsonb;
BEGIN
 PERFORM vec_autorizacion.canon_version_rol_bolsa_v1(p,'plan');
 IF p_primera IS NULL OR p->>'operacion' IS DISTINCT FROM 'versionar'
 OR p#>>'{definicion_nueva,rol_id}' IS DISTINCT FROM 'tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo'
 OR p#>>'{base,rol,rol_id}' IS DISTINCT FROM p#>>'{definicion_nueva,rol_id}'
 OR (p#>>'{definicion_nueva,version}')::bigint<>(p#>>'{base,rol,version}')::bigint+1
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM
  'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v'||(p#>>'{definicion_nueva,version}')
 OR p#>>'{base,tipo_perfil}' IS DISTINCT FROM 'administrable'
 THEN RAISE EXCEPTION 'AUT62: versión B1 inválida' USING ERRCODE='22023';END IF;
 base:=p->'base';nueva:=p->'definicion_nueva';seleccion:=p->'seleccion';
 IF p_primera THEN
  SELECT * INTO h FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1
   WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR h.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR h.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT62: cabeza catálogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 c:=convert_from(vec_autorizacion.resolver_catalogo_acciones_administracion_v1(
  p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256'),'UTF8')::jsonb;
 SELECT value INTO e FROM jsonb_array_elements(c->'entradas')
  WHERE value->>'referencia'=seleccion->>'entrada_ref';
 IF NOT FOUND THEN RAISE EXCEPTION 'AUT62: descriptor B1 ausente' USING ERRCODE='42501';END IF;
 concesion_b1:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(
  jsonb_build_object('concesiones',jsonb_build_array(e->'concesion')))#>'{concesiones,0}';
 IF e->>'version' IS DISTINCT FROM seleccion->>'entrada_version'
 OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'entrada'),'UTF8')),'hex')
  IS DISTINCT FROM seleccion->>'entrada_huella_sha256'
 OR concesion_b1 IS DISTINCT FROM nueva#>'{concesiones,-1}'
 OR e#>>'{concesion,accion}' IS DISTINCT FROM 'bolsa.carga_convoca.confirmar'
 OR e#>>'{concesion,modulo_id}' IS DISTINCT FROM 'bolsa'
 OR e#>>'{concesion,tipo_recurso}' IS DISTINCT FROM 'carga_convoca'
 OR e#>'{concesion,finalidades}' IS DISTINCT FROM '["carga_bolsa_convoca"]'::jsonb
 OR e#>>'{concesion,garantia_minima}' IS DISTINCT FROM 'alto'
 OR coalesce(e#>'{concesion,campos_permitidos}','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
 OR coalesce(e#>'{concesion,obligaciones}','[]'::jsonb) IS DISTINCT FROM '[]'::jsonb
 OR e->>'clase_control' IS DISTINCT FROM 'ordinario'
 THEN RAISE EXCEPTION 'AUT62: descriptor B1 divergente' USING ERRCODE='42501';END IF;
 ahora:=clock_timestamp();
 IF p_primera AND (ahora<(c->>'vigente_desde')::timestamptz
 OR (c->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND ahora>=(c->>'vigente_hasta')::timestamptz)
 OR ahora<(e->>'vigente_desde')::timestamptz
 OR (e->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND ahora>=(e->>'vigente_hasta')::timestamptz))
 THEN RAISE EXCEPTION 'AUT62: descriptor caducado' USING ERRCODE='42501';END IF;
 SELECT * INTO r FROM vec_autorizacion.version_rol
  WHERE version_rol_ref='rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v'||(base#>>'{rol,version}')
  FOR SHARE;
 IF NOT FOUND OR r.rol_id IS DISTINCT FROM 'tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo'
 OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(r.documento) IS DISTINCT FROM base->'rol'
 OR (p_primera AND EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.rol_id=r.rol_id AND z.version>r.version))
 THEN RAISE EXCEPTION 'AUT62: base de rol divergente' USING ERRCODE='40001';END IF;
 SELECT x.* INTO cv FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=r.version_rol_ref
 AND x.revision=(base#>>'{control_vigencia,revision}')::numeric FOR SHARE;
 IF NOT FOUND OR cv.estado IS DISTINCT FROM 'habilitada'
 OR cv.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(cv.documento),'UTF8')),'hex')
 OR cv.documento IS DISTINCT FROM base->'control_vigencia'
 OR (p_primera AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual q
  WHERE q.version_rol_ref=cv.version_rol_ref AND q.revision=cv.revision))
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(c->'perfiles') q
  WHERE q->'rol'=base->'rol' AND q->'control_vigencia'=base->'control_vigencia'
   AND q->>'tipo_perfil'='administrable')
 THEN RAISE EXCEPTION 'AUT62: control o catálogo base divergente' USING ERRCODE='40001';END IF;
 IF nueva->>'nombre' IS DISTINCT FROM base#>>'{rol,nombre}'
 OR nueva->'concesiones' IS DISTINCT FROM (base#>'{rol,concesiones}')||jsonb_build_array(concesion_b1)
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(base#>'{rol,concesiones}') x
   WHERE x#>>'{accion}'='bolsa.carga_convoca.confirmar')
 OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z WHERE z.version_rol_ref=p->>'version_rol_objetivo_ref' AND p_primera)
 THEN RAISE EXCEPTION 'AUT62: concesiones no preservadas' USING ERRCODE='40001';END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(p->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
  IF t->>'asignacion_ref'=ANY(vistas) THEN RAISE EXCEPTION 'AUT62: asignación duplicada' USING ERRCODE='22023';END IF;
  vistas:=array_append(vistas,t->>'asignacion_ref');
  original:=t->'documento';
  IF t->>'asignacion_ref' IS DISTINCT FROM 'asignacion:'||(original->>'asignacion_id')||':v'||(original->>'version')
  OR t->>'huella_sha256' IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(original),'UTF8')),'hex')
  OR original->>'version_rol_ref' NOT LIKE 'rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v%'
  OR original->>'estado' IS DISTINCT FROM 'activa'
  THEN RAISE EXCEPTION 'AUT62: selección de asignación inválida' USING ERRCODE='22023';END IF;
  SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil x
   WHERE x.asignacion_ref=t->>'asignacion_ref' FOR SHARE;
  IF NOT FOUND OR a.huella_sha256 IS DISTINCT FROM t->>'huella_sha256'
  OR a.documento IS DISTINCT FROM original OR a.principal_id IS DISTINCT FROM original->>'principal_id'
  OR a.perfil_activo_ref IS DISTINCT FROM original->>'perfil_activo_ref'
  OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol z
   WHERE z.version_rol_ref=a.version_rol_ref AND z.rol_id=r.rol_id AND z.version<=r.version)
  THEN RAISE EXCEPTION 'AUT62: asignación histórica divergente' USING ERRCODE='40001';END IF;
  IF p_primera AND (NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q
   WHERE q.perfil_activo_ref=a.perfil_activo_ref AND q.asignacion_ref=a.asignacion_ref)
   OR clock_timestamp()<(a.documento->>'vigente_desde')::timestamptz
   OR clock_timestamp()>=(a.documento->>'vigente_hasta')::timestamptz
   OR EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil h
     WHERE h.asignacion_id=a.asignacion_id AND h.documento->>'estado'='revocada'))
  THEN RAISE EXCEPTION 'AUT62: asignación ya no actual o vigente' USING ERRCODE='40001';END IF;
 END LOOP;
 RETURN e;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_plan_version_rol_bolsa_v1(jsonb,boolean) FROM PUBLIC;

RESET ROLE;
COMMIT;
