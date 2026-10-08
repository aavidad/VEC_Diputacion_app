\set ON_ERROR_STOP on
-- AUT63: publicación nominal B1 y avance CAS de asignaciones en una transacción.
-- Depende de AUT62 y AD227; AUT64 publica previamente las concesiones ADMIN.
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
 OR to_regprocedure('vec_autorizacion.validar_plan_version_rol_bolsa_v1(jsonb,boolean)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_version_rol_bolsa_v1(jsonb)') IS NULL
 OR to_regrole('vec_admin_version_rol_bolsa_ejecutor') IS NULL
 OR to_regprocedure('vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT63: preimagen AUT62/AD227 incompatible' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.concesiones_version_rol_bolsa_v1()
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT '[{"accion":"administracion.perfiles.version_bolsa.proponer","modulo_id":"administracion","tipo_recurso":"definicion_rol","finalidades":["gobierno_definiciones_perfiles"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]},{"accion":"administracion.perfiles.version_bolsa.aprobar","modulo_id":"administracion","tipo_recurso":"propuesta_definicion_rol","finalidades":["gobierno_definiciones_perfiles"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]}]'::jsonb
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.concesiones_version_rol_bolsa_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion.administradores_version_rol_bolsa_v1()
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
 AND r.documento->'concesiones' @> vec_autorizacion.concesiones_version_rol_bolsa_v1();
 RETURN salida;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.administradores_version_rol_bolsa_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.comprobar_postimagen_version_rol_bolsa_v1(p_propuesta text)
RETURNS void LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c record;r record;cv record;t jsonb;a record;s record;
BEGIN
 SELECT * INTO STRICT c FROM vec_autorizacion.cierre_version_rol_bolsa_v1 WHERE propuesta_ref=p_propuesta;
 SELECT * INTO STRICT r FROM vec_autorizacion.version_rol WHERE version_rol_ref=c.version_rol_ref FOR SHARE;
 SELECT x.* INTO STRICT cv FROM vec_autorizacion.control_vigencia_version_rol x
 WHERE x.version_rol_ref=c.version_rol_ref AND x.revision=1 FOR SHARE;
 IF r.huella_sha256 IS DISTINCT FROM c.rol_sha256 OR r.documento IS DISTINCT FROM c.resultado#>'{recibo,version_rol}'
 OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR cv.revision<>1 OR cv.estado<>'habilitada' OR cv.huella_sha256 IS DISTINCT FROM c.control_sha256
 OR cv.documento IS DISTINCT FROM c.resultado#>'{recibo,control_posterior}'
 THEN RAISE EXCEPTION 'AUT63: replay sin postimagen vigente' USING ERRCODE='40001';END IF;
 FOR t IN SELECT value FROM jsonb_array_elements(c.resultado#>'{recibo,asignaciones}') x LOOP
  SELECT * INTO STRICT a FROM vec_autorizacion.asignacion_perfil
   WHERE asignacion_ref='asignacion:'||(t#>>'{posterior,asignacion_id}')||':v'||(t#>>'{posterior,version}') FOR SHARE;
  SELECT * INTO STRICT s FROM vec_autorizacion.sello_efecto_admin_tx_v1
   WHERE asignacion_ref=a.asignacion_ref FOR SHARE;
  IF a.documento IS DISTINCT FROM t->'posterior'
  OR a.huella_sha256 IS DISTINCT FROM t->>'posterior_sha256'
  OR s.operacion_ref IS DISTINCT FROM c.operacion_ref
  OR s.auditoria_ref IS DISTINCT FROM c.auditoria_ref
  THEN RAISE EXCEPTION 'AUT63: replay sin asignación histórica' USING ERRCODE='40001';END IF;
 END LOOP;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.comprobar_postimagen_version_rol_bolsa_v1(text) FROM PUBLIC;

-- La correlación corresponde a cada acceso V3, no al material semántico de la
-- operación. Los demás campos del sobre deben seguir siendo idénticos.
CREATE FUNCTION vec_autorizacion.solicitud_replay_version_rol_bolsa_v1(p_original bytea,p_actual text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT (convert_from(p_original,'UTF8')::jsonb-'correlacion_ref')
  IS NOT DISTINCT FROM (p_actual::jsonb-'correlacion_ref')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.solicitud_replay_version_rol_bolsa_v1(bytea,text) FROM PUBLIC;

-- El recibo almacenado conserva el primer acceso; la respuesta acredita el
-- consumo V3 de esta consulta sin reescribir la historia de la propuesta.
CREATE FUNCTION vec_autorizacion.resultado_propuesta_acceso_version_rol_bolsa_v1(
 p_original jsonb,p_auditoria_actual text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF jsonb_typeof(p_original) IS DISTINCT FROM 'object'
 OR (p_original->>'auditoria_acceso_ref' ~ '^aud_v3_[0-9a-f]{32}$') IS NOT TRUE
 OR (p_auditoria_actual ~ '^aud_v3_[0-9a-f]{32}$') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT63: auditoria de propuesta invalida' USING ERRCODE='42501';END IF;
 RETURN p_original||jsonb_build_object('auditoria_acceso_ref',p_auditoria_actual);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resultado_propuesta_acceso_version_rol_bolsa_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_version_rol_bolsa_v1(
 p_cierre boolean,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC'
SET lock_timeout='2s' AS $f$
DECLARE m jsonb;mat jsonb;plan jsonb;d jsonb;c jsonb;motivo jsonb;entrada jsonb;efectivos jsonb;
 prop vec_autorizacion.propuesta_version_rol_bolsa_v1;prev vec_autorizacion.cierre_version_rol_bolsa_v1;
 x record;a record;amb jsonb;ambcanon text;ctx text;huella text;material_sha text;
 persona text;perfil text;asignacion text;recurso text;op text;accion text;audiencia text;
 ahora timestamptz(6);caduca timestamptz(6);fecha text;rol jsonb;control jsonb;rol_sha text;control_sha text;
 resultado jsonb;recibo jsonb;acto text;recibo_ref text;claves text[];replay boolean:=false;
 t jsonb;ant record;puntero record;doc jsonb;destino_ref text;destino_sha text;asigs jsonb:='[]'::jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 OR p_cierre IS NULL OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 196608
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'AUT63: contexto invalido' USING ERRCODE='42501';END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 m:=p_material::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_path_exists(m,'$.** ? (@ == null)')
 THEN RAISE EXCEPTION 'AUT63: envelope invalido' USING ERRCODE='22023';END IF;
 SELECT array_agg(k ORDER BY k COLLATE "C") INTO claves FROM jsonb_object_keys(m) k;
 IF NOT p_cierre THEN
  IF claves IS DISTINCT FROM ARRAY['correlacion_ref','esquema','material_canon','material_sha256','plan_sha256']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_version_rol_bolsa_propuesta_v1'
  OR jsonb_typeof(m->'material_canon') IS DISTINCT FROM 'string' OR octet_length(m->>'material_canon') NOT BETWEEN 1 AND 131072
  THEN RAISE EXCEPTION 'AUT63: propuesta invalida' USING ERRCODE='22023';END IF;
  mat:=(m->>'material_canon')::jsonb;plan:=mat->'Plan';
  IF vec_autorizacion.canon_version_rol_bolsa_v1(mat,'material') IS DISTINCT FROM m->>'material_canon'
  OR encode(sha256(convert_to(m->>'material_canon','UTF8')),'hex') IS DISTINCT FROM m->>'material_sha256'
  OR encode(sha256(convert_to(vec_autorizacion.canon_version_rol_bolsa_v1(plan,'plan'),'UTF8')),'hex') IS DISTINCT FROM m->>'plan_sha256'
  OR mat->>'OperacionRef' !~ '^propuesta_admin:[0-9a-f]{32}$'
  THEN RAISE EXCEPTION 'AUT63: canon o huella divergente' USING ERRCODE='22023';END IF;
  persona:=mat->>'ProponentePersonaRef';perfil:=mat->>'PerfilActivoRef';asignacion:=mat->>'AsignacionPerfilRef';
  recurso:=plan->>'version_rol_objetivo_ref';op:=mat->>'OperacionRef';motivo:=plan->'motivo';
  accion:='administracion.perfiles.version_bolsa.proponer';audiencia:='vec_autorizacion.versionar_rol_bolsa.propuesta.v1';
 ELSE
  IF claves IS DISTINCT FROM ARRAY['actor_perfil_ref','actor_persona_ref','asignacion_ref','correlacion_ref','decision','esquema','motivo','operacion_ref','propuesta_huella_sha256','propuesta_ref']::text[]
  OR m->>'esquema' IS DISTINCT FROM 'administracion_version_rol_bolsa_cierre_v1'
  OR m->>'decision' IS DISTINCT FROM 'aprobada' OR m->>'operacion_ref' !~ '^cierre_admin:[0-9a-f]{32}$'
  OR m->>'propuesta_ref' !~ '^propuesta_admin:[0-9a-f]{32}$' OR m->>'propuesta_huella_sha256' !~ '^[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'AUT63: cierre invalido' USING ERRCODE='22023';END IF;
  persona:=m->>'actor_persona_ref';perfil:=m->>'actor_perfil_ref';asignacion:=m->>'asignacion_ref';
  recurso:=m->>'propuesta_ref';op:=m->>'operacion_ref';motivo:=m->'motivo';
  accion:='administracion.perfiles.version_bolsa.aprobar';audiencia:='vec_autorizacion.versionar_rol_bolsa.cierre.v1';
 END IF;
 PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(motivo,'motivo');
 IF (persona ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE OR (perfil ~ '^prf_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
 OR m->>'correlacion_ref' IS NULL OR m->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR (convert_from(p_motivo,'UTF8')::jsonb)->'referencia' IS DISTINCT FROM motivo
 OR d->>'principal_id' IS DISTINCT FROM persona OR d->>'perfil_activo_ref' IS DISTINCT FROM perfil
 OR d->>'asignacion_ref' IS DISTINCT FROM asignacion OR d->>'accion' IS DISTINCT FROM accion
 OR d->>'correlacion_ref' IS DISTINCT FROM m->>'correlacion_ref'
 OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
 OR vec_autorizacion.acreditar_version_rol_bolsa_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT63: actor o decision ajenos' USING ERRCODE='42501';END IF;
 SELECT q.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil q WHERE q.asignacion_ref=asignacion;
 IF jsonb_typeof(a.documento->'ambitos') IS DISTINCT FROM 'array'
 OR (SELECT count(*) FROM jsonb_array_elements(a.documento->'ambitos'))<>2
 OR (SELECT count(DISTINCT b->>'clave') FROM jsonb_array_elements(a.documento->'ambitos') b)<>2
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(a.documento->'ambitos') b
  WHERE b->>'clave' NOT IN('organizacion_ref','unidad_ref')
  OR (CASE WHEN jsonb_typeof(b->'valores')='array' THEN jsonb_array_length(b->'valores')=1 ELSE false END) IS NOT TRUE)
 THEN RAISE EXCEPTION 'AUT63: ambitos no unitarios' USING ERRCODE='42501';END IF;
 SELECT jsonb_object_agg(b->>'clave',b#>>'{valores,0}') INTO amb FROM jsonb_array_elements(a.documento->'ambitos') b;
 IF (SELECT count(*) FROM jsonb_object_keys(amb))<>2 OR NOT amb ?& ARRAY['organizacion_ref','unidad_ref']
 THEN RAISE EXCEPTION 'AUT63: ambitos divergentes' USING ERRCODE='42501';END IF;
 ambcanon:='{"organizacion_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'organizacion_ref')
  ||',"unidad_ref":'||vec_autorizacion.json_cadena_canonica_go_admin_v1(amb->>'unidad_ref')||'}';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 ctx:='{"ambitos":'||ambcanon||',"atributos":{"material_sha256":"'||material_sha||'"}}';
 huella:=encode(sha256(convert_to(ctx,'UTF8')),'hex');
 IF d->>'recurso_ref' IS DISTINCT FROM recurso OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
 OR c->>'efecto_ref' IS DISTINCT FROM recurso OR c->>'huella_efecto_sha256' IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT63: recurso no ligado al material' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(
 p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM recurso OR x.huella_efecto_sha256 IS DISTINCT FROM huella
 THEN RAISE EXCEPTION 'AUT63: consumo divergente' USING ERRCODE='42501';END IF;
 efectivos:=vec_autorizacion.administradores_version_rol_bolsa_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 THEN RAISE EXCEPTION 'AUT63: dos ADMIN actuales necesarios' USING ERRCODE='42501';END IF;
 IF p_cierre THEN
  SELECT * INTO STRICT prop FROM vec_autorizacion.propuesta_version_rol_bolsa_v1 WHERE propuesta_ref=recurso FOR SHARE;
  mat:=convert_from(prop.material,'UTF8')::jsonb;plan:=mat->'Plan';
  IF prop.material_sha256 IS DISTINCT FROM m->>'propuesta_huella_sha256'
  OR prop.proponente_persona_ref=persona OR prop.proponente_perfil_ref=perfil
  OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref
   AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref)
  THEN RAISE EXCEPTION 'AUT63: cierre no independiente o proponente revocado' USING ERRCODE='42501';END IF;
 ELSE
  SELECT * INTO prop FROM vec_autorizacion.propuesta_version_rol_bolsa_v1 WHERE propuesta_ref=op FOR SHARE;
  IF FOUND THEN
   IF vec_autorizacion.solicitud_replay_version_rol_bolsa_v1(prop.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT63: replay de propuesta divergente' USING ERRCODE='23505';END IF;
   replay:=true;
  END IF;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:version-rol-bolsa:'||(plan#>>'{definicion_nueva,rol_id}'),0));
 IF p_cierre THEN
  SELECT * INTO prev FROM vec_autorizacion.cierre_version_rol_bolsa_v1 WHERE propuesta_ref=prop.propuesta_ref;
  IF FOUND THEN
   IF prev.operacion_ref IS DISTINCT FROM op
   OR vec_autorizacion.solicitud_replay_version_rol_bolsa_v1(prev.solicitud,p_material) IS NOT TRUE
   THEN RAISE EXCEPTION 'AUT63: propuesta cerrada con otro material' USING ERRCODE='23505';END IF;
   PERFORM vec_autorizacion.comprobar_postimagen_version_rol_bolsa_v1(prop.propuesta_ref);
   replay:=true;
  END IF;
 END IF;
 entrada:=vec_autorizacion.validar_plan_version_rol_bolsa_v1(plan,NOT replay);
 IF p_cierre THEN
  IF replay THEN
   resultado:=prev.resultado||jsonb_build_object('auditoria_acceso_ref',x.auditoria_ref);
  ELSE
   ahora:=clock_timestamp();
   IF ahora>=prop.caduca_en OR EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref=prop.version_rol_ref)
   THEN RAISE EXCEPTION 'AUT63: propuesta caducada o versión ocupada' USING ERRCODE='40001';END IF;
   fecha:=vec_autorizacion.fecha_canonica_go_admin_v1(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
   acto:='acto_admin:'||substr(op,14,32);recibo_ref:='recibo_admin:'||substr(op,14,32);
   rol:=plan->'definicion_nueva'||jsonb_build_object('estado','publicada','publicada_por',persona,'publicada_en',fecha,'retirada_en','0001-01-01T00:00:00Z');
   rol_sha:=encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(rol),'UTF8')),'hex');
   control:=jsonb_build_object('version_rol_ref',prop.version_rol_ref,'revision',1,'estado','habilitada','actualizado_por',persona,'actualizado_en',fecha);
   control_sha:=encode(sha256(convert_to(vec_autorizacion.canon_control_rol_admin_v1(control),'UTF8')),'hex');
   INSERT INTO vec_autorizacion.version_rol(version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
    VALUES(prop.version_rol_ref,rol->>'rol_id',(rol->>'version')::bigint,rol_sha,ahora,rol);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol(version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
    VALUES(prop.version_rol_ref,1,'habilitada',control_sha,ahora,control);
   INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual(version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
    VALUES(prop.version_rol_ref,1,ahora,persona,acto);
   FOR t IN SELECT value FROM jsonb_array_elements(plan->'asignaciones') WITH ORDINALITY x(value,n) ORDER BY n LOOP
    SELECT * INTO STRICT puntero FROM vec_autorizacion.asignacion_perfil_actual
     WHERE perfil_activo_ref=t#>>'{documento,perfil_activo_ref}' FOR UPDATE;
    IF puntero.asignacion_ref IS DISTINCT FROM t->>'asignacion_ref'
    THEN RAISE EXCEPTION 'AUT63: puntero CAS divergente' USING ERRCODE='40001';END IF;
    SELECT * INTO STRICT ant FROM vec_autorizacion.asignacion_perfil
     WHERE asignacion_ref=t->>'asignacion_ref' FOR SHARE;
    IF ant.documento IS DISTINCT FROM t->'documento'
    OR ant.huella_sha256 IS DISTINCT FROM t->>'huella_sha256'
    OR ant.documento->>'estado' IS DISTINCT FROM 'activa'
    OR ahora<(ant.documento->>'vigente_desde')::timestamptz
    OR ahora>=(ant.documento->>'vigente_hasta')::timestamptz
    THEN RAISE EXCEPTION 'AUT63: asignación CAS divergente' USING ERRCODE='40001';END IF;
    destino_ref:='asignacion:'||ant.asignacion_id||':v'||(ant.version+1)::text;
    doc:=ant.documento||jsonb_build_object('version',ant.version+1,'version_rol_ref',prop.version_rol_ref,
     'emitida_por',persona,'emitida_en',fecha,'vigente_desde',fecha);
    destino_sha:=encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(doc),'UTF8')),'hex');
    INSERT INTO vec_autorizacion.asignacion_perfil(
     asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,version_rol_ref,huella_sha256,emitida_en,documento)
    VALUES(destino_ref,ant.asignacion_id,ant.version+1,ant.perfil_activo_ref,ant.principal_id,
     prop.version_rol_ref,destino_sha,ahora,doc);
    INSERT INTO vec_autorizacion.sello_efecto_admin_tx_v1
     VALUES(destino_ref,txid_current(),op,x.auditoria_ref);
    UPDATE vec_autorizacion.asignacion_perfil_actual SET asignacion_ref=destino_ref,
     actualizada_en=ahora,actualizada_por=persona,acto_ref=acto
     WHERE perfil_activo_ref=ant.perfil_activo_ref AND asignacion_ref=ant.asignacion_ref;
    IF NOT FOUND THEN RAISE EXCEPTION 'AUT63: puntero CAS perdido' USING ERRCODE='40001';END IF;
    asigs:=asigs||jsonb_build_array(jsonb_build_object('anterior',ant.documento,'posterior',doc,
     'anterior_sha256',ant.huella_sha256,'posterior_sha256',destino_sha));
   END LOOP;
   recibo:=jsonb_build_object('acto_ref',acto,'recibo_ref',recibo_ref,'actor_persona_ref',persona,'perfil_activo_ref',perfil,
    'asignacion_perfil_ref',asignacion,'correlacion_ref',m->>'correlacion_ref','motivo',motivo,'auditoria_ref',x.auditoria_ref,'version_rol',rol,'control_posterior',control,'asignaciones',asigs);
   resultado:=jsonb_build_object('operacion_ref',op,'material_canon',convert_from(prop.material,'UTF8'),'propuesta_huella_sha256',prop.material_sha256,
    'decision','aprobada','confirmado_en',ahora,'auditoria_acceso_ref',x.auditoria_ref,'recibo',recibo);
   INSERT INTO vec_autorizacion.cierre_version_rol_bolsa_v1 VALUES(prop.propuesta_ref,op,convert_to(p_material,'UTF8'),prop.version_rol_ref,rol_sha,control_sha,persona,perfil,asignacion,x.auditoria_ref,resultado,ahora);
   INSERT INTO vec_autorizacion.outbox_version_rol_bolsa_v1 VALUES(op,'version_publicada',x.auditoria_ref,ahora);
  END IF;
 ELSE
  IF replay THEN
   IF EXISTS(SELECT 1 FROM vec_autorizacion.cierre_version_rol_bolsa_v1 WHERE propuesta_ref=op) THEN
    PERFORM vec_autorizacion.comprobar_postimagen_version_rol_bolsa_v1(op);
   END IF;
   resultado:=vec_autorizacion.resultado_propuesta_acceso_version_rol_bolsa_v1(prop.resultado,x.auditoria_ref);
  ELSE
   IF EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref=plan->>'version_rol_objetivo_ref')
   THEN RAISE EXCEPTION 'AUT63: versión objetivo ocupada' USING ERRCODE='40001';END IF;
   ahora:=clock_timestamp();
   caduca:=LEAST(ahora+interval '1 day',(a.documento->>'vigente_hasta')::timestamptz,
    NULLIF(entrada->>'vigente_hasta','0001-01-01T00:00:00Z')::timestamptz);
   resultado:=jsonb_build_object('material_canon',m->>'material_canon','huella_sha256',m->>'material_sha256',
    'caduca_en',caduca,'auditoria_acceso_ref',x.auditoria_ref);
   INSERT INTO vec_autorizacion.propuesta_version_rol_bolsa_v1 VALUES(op,convert_to(m->>'material_canon','UTF8'),m->>'material_sha256',convert_to(p_material,'UTF8'),
    persona,perfil,asignacion,plan->>'catalogo_ref',(plan->>'catalogo_version')::integer,plan->>'catalogo_huella_sha256',recurso,ahora,caduca,x.auditoria_ref,resultado);
   INSERT INTO vec_autorizacion.outbox_version_rol_bolsa_v1 VALUES(op,'version_propuesta',x.auditoria_ref,ahora);
  END IF;
 END IF;
 -- La autorización queda revalidada bajo los bloqueos y la barrera de auditoría.
 IF vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.acreditar_version_rol_bolsa_v1(d) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT63: autorizacion final no vigente' USING ERRCODE='42501';END IF;
 PERFORM vec_autorizacion.validar_plan_version_rol_bolsa_v1(plan,NOT replay AND NOT p_cierre);
 efectivos:=vec_autorizacion.administradores_version_rol_bolsa_v1();
 IF (SELECT count(DISTINCT v->>'persona_ref') FROM jsonb_array_elements(efectivos) v)<2
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=persona AND v->>'perfil_ref'=perfil AND v->>'asignacion_ref'=asignacion)
 OR (p_cierre AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(efectivos) v WHERE v->>'persona_ref'=prop.proponente_persona_ref AND v->>'perfil_ref'=prop.proponente_perfil_ref AND v->>'asignacion_ref'=prop.asignacion_ref))
 OR (p_cierre AND NOT replay AND clock_timestamp()>=prop.caduca_en)
 THEN RAISE EXCEPTION 'AUT63: doble control final no vigente' USING ERRCODE='42501';END IF;
 RETURN resultado||jsonb_build_object('estado','permitido','replay',replay);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_version_rol_bolsa_v1(boolean,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

-- El subbloque de cada fachada revierte consumo y escrituras al fallar. El
-- intento se asienta después, en la misma transacción que confirma el caller.
-- No convierte un fallo de auditoría en una respuesta denegada sin recibo.
CREATE FUNCTION vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(
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
 codigo:='version_rol_bolsa_'||estado;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_intento_version_rol_bolsa_v1(
  jsonb_build_object('tipo_registro','intento_version_rol_bolsa',
   'evento_ref','evento_'||replace(gen_random_uuid()::text,'-',''),
   'operador_login',session_user::text,'solicitud_sha256',h,
   'accion',p_accion,'recurso_ref','solicitud_version_rol_bolsa:'||substr(h,1,32),
   'resultado',estado,'motivo_ref',codigo,'proceso','postgresql',
   'canal','operacion_tecnica_privada','finalidad_ref','gobierno_definiciones_perfiles',
   'correlacion_ref',corr));
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'auditoria_intento',
  jsonb_build_object('auditoria_ref',a.auditoria_ref,'secuencia',a.secuencia,
   'huella_sha256',a.huella_sha256,'correlacion_ref',a.correlacion_ref,
   'registrada_en',a.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.proponer_version_rol_bolsa_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_version_rol_bolsa_v1(false,p_material,p_capacidad,p_decision,
   p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(p_material,
   'administracion.perfiles.version_bolsa.proponer',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.cerrar_version_rol_bolsa_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET row_security=on
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE resultado jsonb;fallo text;
BEGIN
 BEGIN
  resultado:=vec_autorizacion.aplicar_version_rol_bolsa_v1(true,p_material,p_capacidad,p_decision,
   p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
  RETURN vec_autorizacion.registrar_fallo_version_rol_bolsa_v1(p_material,
   'administracion.perfiles.version_bolsa.aprobar',fallo);
 END;
 RETURN resultado;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cerrar_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_version_rol_bolsa_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.proponer_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.cerrar_version_rol_bolsa_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion.resolver_rol_administrable_v1(text)
 TO vec_admin_version_rol_bolsa_ejecutor;
RESET ROLE;
COMMIT;
