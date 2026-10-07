\set ON_ERROR_STOP on
-- AUT60: propuesta y aprobación por dos ADMIN actuales de un RolID nuevo v1.
-- AUT58 conserva la autoridad del descriptor; AUT59 concede las dos acciones;
-- AD220 consume V3 y audita intentos. Publica sólo la definición/control;
-- la clasificación asignable requiere el plan aprobado de AUT49 y queda fuera.
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
 OR to_regprocedure('vec_autorizacion.resolver_catalogo_acciones_administracion_v1(text,integer,text)') IS NULL
 OR to_regclass('vec_autorizacion.cabeza_catalogo_acciones_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion.concesiones_gobierno_definiciones_admin_v1()') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(jsonb)') IS NULL
 OR to_regrole('vec_admin_gobierno_roles_ejecutor') IS NULL
 OR to_regclass('vec_autorizacion.propuesta_gobierno_rol_nuevo_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT60: preimagen AUT58 AUT59 AD220 incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE TABLE vec_autorizacion.propuesta_gobierno_rol_nuevo_v1(
 propuesta_ref text PRIMARY KEY CHECK(propuesta_ref ~ '^propuesta_admin:[0-9a-f]{32}$'),
 material bytea NOT NULL CHECK(octet_length(material) BETWEEN 1 AND 60000),
 material_sha256 text NOT NULL CHECK(material_sha256=encode(sha256(material),'hex')),
 solicitud bytea NOT NULL CHECK(octet_length(solicitud) BETWEEN 1 AND 65536),
 proponente_persona_ref text NOT NULL,proponente_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 catalogo_ref text NOT NULL,catalogo_version integer NOT NULL,catalogo_sha256 text NOT NULL,
 version_rol_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,caduca_en timestamptz(6) NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,
 CHECK(isfinite(creada_en) AND isfinite(caduca_en) AND caduca_en>creada_en AND caduca_en<=creada_en+interval '1 day'),
 FOREIGN KEY(catalogo_ref,catalogo_version,catalogo_sha256)
 REFERENCES vec_autorizacion.registro_catalogo_acciones_admin_v1(catalogo_ref,version,huella_sha256)
);
CREATE TABLE vec_autorizacion.cierre_gobierno_rol_nuevo_v1(
 propuesta_ref text PRIMARY KEY REFERENCES vec_autorizacion.propuesta_gobierno_rol_nuevo_v1(propuesta_ref),
 operacion_ref text NOT NULL UNIQUE CHECK(operacion_ref ~ '^cierre_admin:[0-9a-f]{32}$'),
 solicitud bytea NOT NULL CHECK(octet_length(solicitud) BETWEEN 1 AND 65536),
 version_rol_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.version_rol(version_rol_ref),
 rol_sha256 text NOT NULL CHECK(rol_sha256 ~ '^[0-9a-f]{64}$'),
 control_sha256 text NOT NULL CHECK(control_sha256 ~ '^[0-9a-f]{64}$'),
 aprobador_persona_ref text NOT NULL,aprobador_perfil_ref text NOT NULL,asignacion_ref text NOT NULL,
 auditoria_ref text NOT NULL,resultado jsonb NOT NULL,confirmado_en timestamptz(6) NOT NULL CHECK(isfinite(confirmado_en))
);
CREATE TABLE vec_autorizacion.outbox_gobierno_rol_nuevo_v1(
 operacion_ref text PRIMARY KEY,evento text NOT NULL CHECK(evento IN('definicion_propuesta','definicion_publicada')),
 auditoria_ref text NOT NULL,creada_en timestamptz(6) NOT NULL CHECK(isfinite(creada_en))
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['propuesta_gobierno_rol_nuevo_v1','cierre_gobierno_rol_nuevo_v1','outbox_gobierno_rol_nuevo_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC',t);
 END LOOP;
END $tablas$;

-- Serialización cerrada de los structs Go. No se confunde jsonb::text con
-- json.Marshal: se conservan orden, omitempty, escape HTML y fecha cero.
CREATE FUNCTION vec_autorizacion.canon_gobierno_rol_nuevo_v1(p jsonb,t text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE campos text[];tipos text[];opc text[]:=ARRAY[]::text[];i integer;k text;v jsonb;s text;salida text:='';
BEGIN
 CASE t
 WHEN 'material' THEN campos:=ARRAY['OperacionRef','ProponentePersonaRef','PerfilActivoRef','AsignacionPerfilRef','Plan'];tipos:=ARRAY['string','string','string','string','plan'];
 WHEN 'plan' THEN campos:=ARRAY['operacion','catalogo_ref','catalogo_version','catalogo_huella_sha256','version_rol_objetivo_ref','definicion_nueva','selecciones','motivo','referencia_acto'];tipos:=ARRAY['string','string','int','string','string','definicion','seleccion[]','motivo','string'];opc:=ARRAY['referencia_acto'];
 WHEN 'definicion' THEN campos:=ARRAY['rol_id','version','nombre','concesiones'];tipos:=ARRAY['string','int','nombre','concesion[]'];
 WHEN 'seleccion' THEN campos:=ARRAY['entrada_ref','entrada_version','entrada_huella_sha256'];tipos:=ARRAY['string','int64','string'];
 WHEN 'motivo' THEN campos:=ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'];tipos:=ARRAY['string','int','string','string'];
 WHEN 'entrada' THEN campos:=ARRAY['referencia','version','fuente_ref','fuente_version','fuente_huella_sha256','concesion','dimensiones_ambito','clase_control','vigente_desde','vigente_hasta'];tipos:=ARRAY['string','int64','string','int','string','concesion','strings','string','fecha','fecha'];
 ELSE RAISE EXCEPTION 'AUT60: canon desconocido' USING ERRCODE='22023';
 END CASE;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>60000
 OR EXISTS(SELECT 1 FROM jsonb_object_keys(p) x WHERE NOT x=ANY(campos))
 THEN RAISE EXCEPTION 'AUT60: objeto canon invalido' USING ERRCODE='22023'; END IF;
 FOR i IN 1..cardinality(campos) LOOP
  k:=campos[i];v:=p->k;s:=NULL;
  IF k=ANY(opc) AND (v IS NULL OR v='""'::jsonb) THEN CONTINUE; END IF;
  IF v IS NULL OR v='null'::jsonb THEN RAISE EXCEPTION 'AUT60: campo canon ausente' USING ERRCODE='22023'; END IF;
  IF tipos[i] IN('string','nombre','fecha') THEN
   IF jsonb_typeof(v) IS DISTINCT FROM 'string' OR octet_length(v#>>'{}') NOT BETWEEN 1 AND 512
   OR (tipos[i]<>'nombre' AND (vec_autorizacion.texto_positivo_valido(v#>>'{}',512) IS NOT TRUE
    OR (v#>>'{}') COLLATE "C" ~ '[^!-~]'))
   OR (tipos[i]='nombre' AND ((v#>>'{}')<>btrim(v#>>'{}') OR (v#>>'{}') ~ '[[:cntrl:]]'))
   THEN RAISE EXCEPTION 'AUT60: cadena canon invalida' USING ERRCODE='22023'; END IF;
   IF tipos[i]='fecha' THEN s:=vec_autorizacion.fecha_canonica_go_admin_v1(v#>>'{}',k='vigente_hasta');ELSE s:=v#>>'{}';END IF;
   s:=vec_autorizacion.json_cadena_canonica_go_admin_v1(s);
  ELSIF tipos[i] IN('int','int64') THEN
   IF jsonb_typeof(v) IS DISTINCT FROM 'number' OR (v#>>'{}') !~ '^[1-9][0-9]{0,18}$'
   OR (v#>>'{}')::numeric>(CASE WHEN tipos[i]='int' THEN 2147483647::numeric ELSE 9223372036854775807::numeric END)
   THEN RAISE EXCEPTION 'AUT60: entero canon invalido' USING ERRCODE='22023'; END IF;s:=v#>>'{}';
  ELSIF tipos[i] IN('seleccion[]','concesion[]') THEN
   IF jsonb_typeof(v) IS DISTINCT FROM 'array' OR jsonb_array_length(v)<>1 THEN RAISE EXCEPTION 'AUT60: se requiere una concesion' USING ERRCODE='22023';END IF;
   IF tipos[i]='concesion[]' THEN s:='['||vec_autorizacion.objeto_canonico_go_admin_v1(v->0,'concesion')||']';
   ELSE s:='['||vec_autorizacion.canon_gobierno_rol_nuevo_v1(v->0,'seleccion')||']';END IF;
  ELSIF tipos[i]='concesion' THEN s:=vec_autorizacion.objeto_canonico_go_admin_v1(v,'concesion');
  ELSIF tipos[i]='strings' THEN s:=vec_autorizacion.array_cadenas_canonico_go_admin_v1(v);
  ELSE s:=vec_autorizacion.canon_gobierno_rol_nuevo_v1(v,tipos[i]);END IF;
  IF salida<>'' THEN salida:=salida||',';END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(k)||':'||s;
 END LOOP;
 RETURN '{'||salida||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_gobierno_rol_nuevo_v1(jsonb,text) FROM PUBLIC;

-- Puerta positiva de Aplicación: versión actual, clasificación, concesión y
-- catálogo nominal coincidentes, sesión administrativa real y ámbitos exactos.
CREATE FUNCTION vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r record;cv record;a record;meta record;c jsonb;tipo text;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR d->>'accion' NOT IN('administracion.perfiles.definicion.proponer','administracion.perfiles.definicion.aprobar')
 OR d->>'accion' IS NULL OR d->>'modulo_id' IS DISTINCT FROM 'administracion'
 OR d->>'finalidad' IS DISTINCT FROM 'gobierno_definiciones_perfiles'
 OR d->>'garantia_minima' IS DISTINCT FROM 'alto' OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 OR vec_identidad_sesiones_v1.acreditar_operador_cargos_ct_v1(d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RETURN false;END IF;
 tipo:=CASE d->>'accion' WHEN 'administracion.perfiles.definicion.proponer' THEN 'definicion_rol' ELSE 'propuesta_definicion_rol' END;
 IF d->>'tipo_recurso' IS DISTINCT FROM tipo THEN RETURN false;END IF;
 SELECT value INTO STRICT c FROM jsonb_array_elements(vec_autorizacion.concesiones_gobierno_definiciones_admin_v1())
 WHERE value->>'accion'=d->>'accion';
 SELECT x.* INTO r FROM vec_autorizacion.version_rol x WHERE x.version_rol_ref=d->>'version_rol_ref' FOR SHARE;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO cv FROM vec_autorizacion.control_vigencia_version_rol_actual q
 JOIN vec_autorizacion.control_vigencia_version_rol x USING(version_rol_ref,revision)
 WHERE q.version_rol_ref=r.version_rol_ref FOR SHARE OF q,x;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual q
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE q.perfil_activo_ref=d->>'perfil_activo_ref' AND q.asignacion_ref=d->>'asignacion_ref' FOR SHARE OF q,x;
 IF NOT FOUND THEN RETURN false;END IF;
 SELECT x.* INTO meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE x.version_rol_ref=r.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false;END IF;
 ahora:=clock_timestamp();
 RETURN r.rol_id='administracion_perfiles' AND r.documento->>'estado'='publicada'
 AND r.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 AND cv.estado='habilitada' AND cv.huella_sha256=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(cv.documento),'UTF8')),'hex')
 AND r.huella_sha256=d->>'version_rol_huella_sha256' AND cv.revision::text=d->>'control_vigencia_version_rol_revision'
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
 AND EXISTS(SELECT 1 FROM vec_autorizacion.catalogo_accion_nominal_v1 n WHERE n.version_rol_ref=r.version_rol_ref
  AND n.accion_ref='accion:'||(d->>'accion') AND n.fuente_ref=r.version_rol_ref AND n.fuente_version=r.version
  AND n.fuente_huella_sha256=r.huella_sha256 AND n.concesion=c AND n.clase_control='administrador_aplicacion'
  AND n.dimensiones_ambito='["organizacion_ref"]'::jsonb
  AND ahora>=n.vigente_desde AND (n.vigente_hasta IS NULL OR ahora<n.vigente_hasta))
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(r.documento->'concesiones') x WHERE x=c);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(jsonb) TO vec_autorizacion_atestada_v3_propietario;

-- El primer efecto exige cabeza vigente; el replay coteja el descriptor
-- histórico íntegro y la postimagen original aun si la cabeza ya avanzó.
CREATE FUNCTION vec_autorizacion.validar_plan_rol_nuevo_v1(p jsonb,p_exigir_cabeza boolean)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE c jsonb;e jsonb;s jsonb;canon text;h record;ahora timestamptz;
BEGIN
 canon:=vec_autorizacion.canon_gobierno_rol_nuevo_v1(p,'plan');
 IF p_exigir_cabeza IS NULL OR p->>'operacion' IS DISTINCT FROM 'crear' OR p#>>'{definicion_nueva,version}' IS DISTINCT FROM '1'
 OR p->>'version_rol_objetivo_ref' IS DISTINCT FROM 'rol:'||(p#>>'{definicion_nueva,rol_id}')||':v1'
 OR p#>>'{definicion_nueva,rol_id}' !~ '^[a-z][a-z0-9_]{2,63}$'
 THEN RAISE EXCEPTION 'AUT60: solo RolID nuevo v1' USING ERRCODE='22023';END IF;
 IF p_exigir_cabeza THEN
  SELECT * INTO h FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1 WHERE catalogo_ref=p->>'catalogo_ref' FOR SHARE;
  IF NOT FOUND OR h.version::text IS DISTINCT FROM p->>'catalogo_version'
  OR h.huella_sha256 IS DISTINCT FROM p->>'catalogo_huella_sha256'
  THEN RAISE EXCEPTION 'AUT60: cabeza catalogo divergente' USING ERRCODE='40001';END IF;
 END IF;
 c:=convert_from(vec_autorizacion.resolver_catalogo_acciones_administracion_v1(p->>'catalogo_ref',(p->>'catalogo_version')::integer,p->>'catalogo_huella_sha256'),'UTF8')::jsonb;
 s:=p#>'{selecciones,0}';
 SELECT value INTO STRICT e FROM jsonb_array_elements(c->'entradas') WHERE value->>'referencia'=s->>'entrada_ref';
 ahora:=clock_timestamp();
 IF e->>'version' IS DISTINCT FROM s->>'entrada_version'
 OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(e,'entrada'),'UTF8')),'hex') IS DISTINCT FROM s->>'entrada_huella_sha256'
 OR vec_autorizacion.objeto_canonico_go_admin_v1(e->'concesion','concesion') IS DISTINCT FROM vec_autorizacion.objeto_canonico_go_admin_v1(p#>'{definicion_nueva,concesiones,0}','concesion')
 OR (p_exigir_cabeza AND ahora<(c->>'vigente_desde')::timestamptz)
 OR (p_exigir_cabeza AND c->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND ahora>=(c->>'vigente_hasta')::timestamptz)
 OR (p_exigir_cabeza AND ahora<(e->>'vigente_desde')::timestamptz)
 OR (p_exigir_cabeza AND e->>'vigente_hasta'<>'0001-01-01T00:00:00Z' AND ahora>=(e->>'vigente_hasta')::timestamptz)
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(c->'perfiles') q WHERE q#>>'{rol,rol_id}'=p#>>'{definicion_nueva,rol_id}')
 THEN RAISE EXCEPTION 'AUT60: descriptor no vigente o divergente' USING ERRCODE='42501';END IF;
 RETURN e;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.administradores_gobierno_rol_nuevo_v1()
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE efectivos jsonb;salida jsonb;
BEGIN
 efectivos:=vec_autorizacion.administradores_efectivos_internos_v1();
 SELECT coalesce(jsonb_agg(x),'[]'::jsonb) INTO salida
 FROM jsonb_array_elements(efectivos) x
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=x->>'asignacion_ref'
 JOIN vec_autorizacion.version_rol r USING(version_rol_ref)
 JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 m USING(version_rol_ref)
 WHERE r.rol_id='administracion_perfiles' AND m.categoria_administrativa='aplicacion' AND m.tipo_perfil='fijo_sistema'
 AND m.version_rol_huella_sha256=r.huella_sha256
 AND r.documento->'concesiones' @> vec_autorizacion.concesiones_gobierno_definiciones_admin_v1();
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.administradores_gobierno_rol_nuevo_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.comprobar_postimagen_rol_nuevo_v1(p_propuesta text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c record;r record;cv record;
BEGIN
 SELECT * INTO STRICT c FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 WHERE propuesta_ref=p_propuesta;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref=c.version_rol_ref FOR SHARE;
 SELECT x.* INTO STRICT cv FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=c.version_rol_ref AND x.revision=1 FOR SHARE;
 IF r.huella_sha256 IS DISTINCT FROM c.rol_sha256 OR r.documento IS DISTINCT FROM c.resultado#>'{recibo,version_rol}'
 OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR cv.revision<>1 OR cv.estado<>'habilitada' OR cv.huella_sha256 IS DISTINCT FROM c.control_sha256
 OR cv.documento IS DISTINCT FROM c.resultado#>'{recibo,control_posterior}'
 THEN RAISE EXCEPTION 'AUT60: replay sin postimagen vigente' USING ERRCODE='40001';END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.comprobar_postimagen_rol_nuevo_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(
 p_cierre boolean,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC'
SET lock_timeout='2s' AS $f$
DECLARE m jsonb;mat jsonb;plan jsonb;d jsonb;c jsonb;motivo jsonb;entrada jsonb;efectivos jsonb;
 prop vec_autorizacion.propuesta_gobierno_rol_nuevo_v1;prev vec_autorizacion.cierre_gobierno_rol_nuevo_v1;
 x record;a record;amb jsonb;ambcanon text;ctx text;huella text;material_sha text;
 persona text;perfil text;asignacion text;recurso text;op text;accion text;audiencia text;
 ahora timestamptz(6);caduca timestamptz(6);fecha text;rol jsonb;control jsonb;rol_sha text;control_sha text;
 resultado jsonb;recibo jsonb;acto text;recibo_ref text;claves text[];replay boolean:=false;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 OR p_cierre IS NULL OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 65536
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'AUT60: contexto invalido' USING ERRCODE='42501';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_path_exists(m,'$.** ? (@ == null)')
 THEN RAISE EXCEPTION 'AUT60: envelope invalido' USING ERRCODE='22023';END IF;
 SELECT array_agg(k ORDER BY k COLLATE "C") INTO claves FROM jsonb_object_keys(m) k;
 IF NOT p_cierre THEN
  IF claves IS DISTINCT FROM ARRAY['correlacion_ref','esquema','material_canon','material_sha256','plan_sha256']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_gobierno_rol_nuevo_propuesta_v1'
  OR jsonb_typeof(m->'material_canon') IS DISTINCT FROM 'string' OR octet_length(m->>'material_canon') NOT BETWEEN 1 AND 60000
  THEN RAISE EXCEPTION 'AUT60: propuesta invalida' USING ERRCODE='22023';END IF;
  mat:=(m->>'material_canon')::jsonb;plan:=mat->'Plan';
  IF vec_autorizacion.canon_gobierno_rol_nuevo_v1(mat,'material') IS DISTINCT FROM m->>'material_canon'
  OR encode(sha256(convert_to(m->>'material_canon','UTF8')),'hex') IS DISTINCT FROM m->>'material_sha256'
  OR encode(sha256(convert_to(vec_autorizacion.canon_gobierno_rol_nuevo_v1(plan,'plan'),'UTF8')),'hex') IS DISTINCT FROM m->>'plan_sha256'
  OR mat->>'OperacionRef' !~ '^propuesta_admin:[0-9a-f]{32}$'
  THEN RAISE EXCEPTION 'AUT60: canon o huella divergente' USING ERRCODE='22023';END IF;
  persona:=mat->>'ProponentePersonaRef';perfil:=mat->>'PerfilActivoRef';asignacion:=mat->>'AsignacionPerfilRef';
  recurso:=plan->>'version_rol_objetivo_ref';op:=mat->>'OperacionRef';motivo:=plan->'motivo';
  accion:='administracion.perfiles.definicion.proponer';audiencia:='vec_autorizacion.gobierno_rol_nuevo.propuesta.v1';
 ELSE
  IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','asignacion_ref','correlacion_ref','decision','esquema','motivo','operacion_ref','propuesta_huella_sha256','propuesta_ref']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_gobierno_rol_nuevo_cierre_v1'
  OR m->>'decision' IS DISTINCT FROM 'aprobada' OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$'
  OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AUT60: cierre invalido' USING ERRCODE='22023';END IF;
  persona:=m->>'actor_persona_ref';perfil:=m->>'actor_perfil_ref';asignacion:=m->>'asignacion_ref';
  recurso:=m->>'propuesta_ref';op:=m->>'operacion_ref';motivo:=m->'motivo';
  accion:='administracion.perfiles.definicion.aprobar';audiencia:='vec_autorizacion.gobierno_rol_nuevo.cierre.v1';
 END IF;
 PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(motivo,'motivo');
 IF (persona ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE OR (perfil ~ '^prf_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
 OR m->>'correlacion_ref' IS NULL OR m->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR (convert_from(p_motivo,'UTF8')::jsonb)->'referencia' IS DISTINCT FROM motivo
 OR d->>'principal_id' IS DISTINCT FROM persona OR d->>'perfil_activo_ref' IS DISTINCT FROM perfil
 OR d->>'asignacion_ref' IS DISTINCT FROM asignacion OR d->>'accion' IS DISTINCT FROM accion
 OR d->>'correlacion_ref' IS DISTINCT FROM m->>'correlacion_ref'
 OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
 OR vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT60: actor o decision ajenos' USING ERRCODE='42501';END IF;
 SELECT q.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil q WHERE q.asignacion_ref=asignacion;
 IF jsonb_typeof(a.documento->'ambitos') IS DISTINCT FROM 'array'
 OR (SELECT count(*) FROM jsonb_array_elements(a.documento->'ambitos'))<>2
 OR (SELECT count(DISTINCT b->>'clave') FROM jsonb_array_elements(a.documento->'ambitos') b)<>2
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') b
  WHERE b->>'clave' NOT IN('organizacion_ref','unidad_ref')
  OR (CASE WHEN jsonb_typeof(b->'valores')='array' THEN jsonb_array_length(b->'valores')=1 ELSE false END) IS NOT TRUE)
 THEN RAISE EXCEPTION 'AUT60: ambitos no unitarios' USING ERRCODE='42501';END IF;
 SELECT jsonb_object_agg(b->>'clave',b#>>'{valores,0}') INTO amb FROM jsonb_array_elements(a.documento->'ambitos') b;
 IF (SELECT count(*) FROM jsonb_object_keys(amb))<>2 OR NOT amb ?& ARRAY['organizacion_ref','unidad_ref']
 THEN RAISE EXCEPTION 'AUT60: ambitos divergentes' USING ERRCODE='42501';END IF;
 ambcanon:='{"organizacion_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'organizacion_ref')
  ||',"unidad_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'unidad_ref')||'}';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:='{"ambitos":'||ambcanon||',"atributos":{"material_sha256":"'||material_sha||'"}}';
 huella:=encode(sha256(convert_to(ctx,'UTF8')),'hex');
 IF d->>'recurso_ref' IS DISTINCT FROM recurso OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
 OR c->>'efecto_ref' IS DISTINCT FROM recurso OR c->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT60: recurso no ligado al material' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM recurso OR x.huella_efecto_sha256 IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT60: consumo divergente' USING ERRCODE='42501';END IF;
 efectivos:=vec_autorizacion.administradores_gobierno_rol_nuevo_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 THEN RAISE EXCEPTION 'AUT60: dos ADMIN actuales necesarios' USING ERRCODE='42501';END IF;
 IF p_cierre THEN
  SELECT * INTO STRICT prop FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 WHERE propuesta_ref=recurso FOR SHARE;
  mat:=convert_from(prop.material,'UTF8')::jsonb;plan:=mat->'Plan';
  IF prop.material_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256'
  OR prop.proponente_persona_ref=persona OR prop.proponente_perfil_ref=perfil
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref
   AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref)
  THEN RAISE EXCEPTION 'AUT60: cierre no independiente o proponente revocado' USING ERRCODE='42501';END IF;
 ELSE
  SELECT * INTO prop FROM vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 WHERE propuesta_ref=op FOR SHARE;
  IF FOUND THEN
   IF prop.solicitud IS DISTINCT FROM convert_to(p_material,'UTF8') THEN RAISE EXCEPTION 'AUT60: replay de propuesta divergente' USING ERRCODE='23505';END IF;
   replay:=true;
  END IF;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:rol-nuevo:'||(plan#>>'{definicion_nueva,rol_id}'),0));
 IF p_cierre THEN
  SELECT * INTO prev FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 WHERE propuesta_ref=prop.propuesta_ref;
  IF FOUND THEN
   IF prev.operacion_ref IS DISTINCT FROM op OR prev.solicitud IS DISTINCT FROM convert_to(p_material,'UTF8')
   THEN RAISE EXCEPTION 'AUT60: propuesta cerrada con otro material' USING ERRCODE='23505';END IF;
   PERFORM vec_autorizacion.comprobar_postimagen_rol_nuevo_v1(prop.propuesta_ref);
   replay:=true;
  END IF;
 END IF;
 entrada:=vec_autorizacion.validar_plan_rol_nuevo_v1(plan,NOT replay);
 IF p_cierre THEN
  IF replay THEN
   resultado:=prev.resultado||jsonb_build_object('auditoria_acceso_ref',x.auditoria_ref);
  ELSE
   ahora:=clock_timestamp();
   IF ahora>=prop.caduca_en OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=plan#>>'{definicion_nueva,rol_id}')
   THEN RAISE EXCEPTION 'AUT60: propuesta caducada o RolID ocupado' USING ERRCODE='40001';END IF;
   fecha:=vec_autorizacion.fecha_canonica_go_admin_v1(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
   acto:='acto_admin:'||substr(op,14,32);recibo_ref:='recibo_admin:'||substr(op,14,32);
   rol:=plan->'definicion_nueva'||jsonb_build_object('estado','publicada','publicada_por',persona,'publicada_en',fecha,'retirada_en','0001-01-01T00:00:00Z');
   rol_sha:=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol),'UTF8')),'hex');
   control:=jsonb_build_object('version_rol_ref',prop.version_rol_ref,'revision',1,'estado','habilitada','actualizado_por',persona,'actualizado_en',fecha);
   control_sha:=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex');
   INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
    VALUES(prop.version_rol_ref,rol->>'rol_id',1,rol_sha,ahora,rol);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
    VALUES(prop.version_rol_ref,1,'habilitada',control_sha,ahora,control);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
    VALUES(prop.version_rol_ref,1,ahora,persona,acto);
   recibo:=jsonb_build_object('acto_ref',acto,'recibo_ref',recibo_ref,'actor_persona_ref',persona,'perfil_activo_ref',perfil,
    'asignacion_perfil_ref',asignacion,'correlacion_ref',m->>'correlacion_ref','motivo',motivo,'auditoria_ref',x.auditoria_ref,'version_rol',rol,'control_posterior',control);
   resultado:=jsonb_build_object('operacion_ref',op,'material_canon',convert_from(prop.material,'UTF8'),'propuesta_huella_sha256',prop.material_sha256,
    'decision','aprobada','confirmado_en',ahora,'auditoria_acceso_ref',x.auditoria_ref,'recibo',recibo);
   INSERT INTO vec_autorizacion.cierre_gobierno_rol_nuevo_v1 VALUES(prop.propuesta_ref,op,convert_to(p_material,'UTF8'),prop.version_rol_ref,rol_sha,control_sha,persona,perfil,asignacion,x.auditoria_ref,resultado,ahora);
   INSERT INTO vec_autorizacion.outbox_gobierno_rol_nuevo_v1 VALUES(op,'definicion_publicada',x.auditoria_ref,ahora);
  END IF;
 ELSE
  IF replay THEN
   IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_gobierno_rol_nuevo_v1 WHERE propuesta_ref=op) THEN
    PERFORM vec_autorizacion.comprobar_postimagen_rol_nuevo_v1(op);
   ELSIF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=plan#>>'{definicion_nueva,rol_id}') THEN
    RAISE EXCEPTION 'AUT60: RolID ocupado' USING ERRCODE='40001';END IF;
   resultado:=prop.resultado;
  ELSE
   IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id=plan#>>'{definicion_nueva,rol_id}')
   THEN RAISE EXCEPTION 'AUT60: RolID debe ser nuevo' USING ERRCODE='40001';END IF;
   ahora:=clock_timestamp();
   caduca:=LEAST(ahora+interval '1 day',(a.documento->>'vigente_hasta')::timestamptz,
    NULLIF(entrada->>'vigente_hasta','0001-01-01T00:00:00Z')::timestamptz);
   resultado:=jsonb_build_object('material_canon',m->>'material_canon','huella_sha256',m->>'material_sha256','caduca_en',caduca);
   INSERT INTO vec_autorizacion.propuesta_gobierno_rol_nuevo_v1 VALUES(op,convert_to(m->>'material_canon','UTF8'),m->>'material_sha256',convert_to(p_material,'UTF8'),
    persona,perfil,asignacion,plan->>'catalogo_ref',(plan->>'catalogo_version')::integer,plan->>'catalogo_huella_sha256',recurso,ahora,caduca,x.auditoria_ref,resultado);
   INSERT INTO vec_autorizacion.outbox_gobierno_rol_nuevo_v1 VALUES(op,'definicion_propuesta',x.auditoria_ref,ahora);
  END IF;
 END IF;
 -- La autorización queda revalidada bajo los bloqueos y la barrera de auditoría.
 IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT60: autorizacion final no vigente' USING ERRCODE='42501';END IF;
 PERFORM vec_autorizacion.validar_plan_rol_nuevo_v1(plan,NOT replay);
 efectivos:=vec_autorizacion.administradores_gobierno_rol_nuevo_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 OR (p_cierre AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref))
 OR (p_cierre AND NOT replay AND clock_timestamp()>=prop.caduca_en)
 THEN RAISE EXCEPTION 'AUT60: doble control final no vigente' USING ERRCODE='42501';END IF;
 RETURN resultado||jsonb_build_object('estado','permitido','replay',replay);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

-- El subbloque de cada fachada revierte consumo y escrituras al fallar. El
-- intento se asienta después, en la misma transacción que confirma el caller.
-- No convierte un fallo de auditoría en una respuesta denegada sin recibo.
CREATE FUNCTION vec_autorizacion.registrar_fallo_gobierno_rol_nuevo_v1(
 p_material text,p_accion text,p_sqlstate text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on AS $f$
DECLARE m jsonb;h text;corr text;estado text;codigo text;a record;
BEGIN
 h:=encode(sha256(convert_to(coalesce(p_material,''),'UTF8')),'hex');
 corr:='correlacion_'||substr(h,1,32);
 BEGIN
  m:=p_material::jsonb;
  IF jsonb_typeof(m)='object' AND m->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$'
  THEN corr:=m->>'correlacion_ref';END IF;
 EXCEPTION WHEN OTHERS THEN NULL;
 END;
 estado:=CASE WHEN p_sqlstate IN('42501','22023','23505','P0002') THEN 'denegado' ELSE 'error' END;
 codigo:='gobierno_rol_nuevo_'||estado;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(
  jsonb_build_object('tipo_registro','intento_gobierno_rol_nuevo',
   'evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'solicitud_sha256',h,
   'accion',p_accion,'recurso_ref','solicitud_gobierno_rol_nuevo:'||substr(h,1,32),
   'resultado',estado,'motivo_ref',codigo,'proceso','postgresql',
   'canal','operacion_tecnica_privada','finalidad_ref','gobierno_definiciones_perfiles',
   'correlacion_ref',corr));
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'auditoria_intento',
  jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
   'huella_sha256',a.huella_sha256,'correlacion_ref',a.correlacion_ref,
   'registrada_en',a.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_fallo_gobierno_rol_nuevo_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.proponer_gobierno_rol_nuevo_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(false,p_material,p_capacidad,p_decision,
   p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_gobierno_rol_nuevo_v1(p_material,
   'administracion.perfiles.definicion.proponer',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proponer_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_gobierno_rol_nuevo_v1(true,p_material,p_capacidad,p_decision,
   p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_gobierno_rol_nuevo_v1(p_material,
   'administracion.perfiles.definicion.aprobar',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_gobierno_roles_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.proponer_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.cerrar_gobierno_rol_nuevo_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_admin_gobierno_roles_ejecutor;
RESET ROLE;
COMMIT;
