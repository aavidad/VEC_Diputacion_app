\set ON_ERROR_STOP on
-- B2: alta y hechos de RRHH. Cada invocación requiere consumo V3 nuevo.
-- El llamador abre SERIALIZABLE READ WRITE y reintenta la transacción entera
-- ante 40001. No ejecutar contra la base histórica hasta instalar dependencias.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000019:escritura-b2',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.registro_empleado_b2_recibo') IS NOT NULL
    OR to_regclass('vec_personal.relacion_servicio_historia') IS NULL
    OR to_regclass('vec_personal.ocupacion_empleado_historia') IS NULL
    OR to_regclass('vec_personal.servicio_reconocido_historia') IS NULL
    OR to_regclass('vec_personal.situacion_empleado_historia') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)') IS NULL
    OR to_regprocedure('vec_personal.publicar_proyeccion_empleado_persona_v1(text,bigint,text,text,text,timestamptz,timestamptz,text,text,bigint,text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)','EXECUTE')
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'Personal 000019: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_personal.registro_empleado_b2_recibo (
 idempotencia_ref uuid PRIMARY KEY,
 operacion text NOT NULL CHECK(operacion IN ('alta','hecho')),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 actor_ref text NOT NULL,
 efecto_ref text NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^perrec_[0-9a-f]{32}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 relacion_ref text NOT NULL CHECK(relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'),
 proyeccion_ref text CHECK(proyeccion_ref ~ '^pep_[A-Za-z0-9_-]{22,128}$'),
 hecho_ref text CHECK(hecho_ref ~ '^(rel|ocu|srv|sit)_[A-Za-z0-9_-]{22,128}$'),
 tipo text NOT NULL CHECK(tipo IN ('alta','relacion','ocupacion','servicio','situacion')),
 version bigint NOT NULL CHECK(version>0),
 decision_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL,
 registrado_en timestamptz(6) NOT NULL,
 catalogo_snapshot jsonb NOT NULL CHECK(pg_catalog.jsonb_typeof(catalogo_snapshot)='object'),
 acreditacion_persona_version numeric,
 acreditacion_procedencia_ref text,
 acreditacion_procedencia_version numeric,
 acreditacion_procedencia_huella_sha256 text,
 eficacia_administrativa boolean NOT NULL DEFAULT false CHECK (eficacia_administrativa=false),
 firma_oficial boolean NOT NULL DEFAULT false CHECK (firma_oficial=false),
 CHECK ((operacion='alta' AND acreditacion_persona_version>0
    AND acreditacion_procedencia_ref IS NOT NULL
    AND acreditacion_procedencia_version>0
    AND acreditacion_procedencia_huella_sha256 ~ '^[0-9a-f]{64}$')
    OR (operacion='hecho' AND acreditacion_persona_version IS NULL
    AND acreditacion_procedencia_ref IS NULL
    AND acreditacion_procedencia_version IS NULL
    AND acreditacion_procedencia_huella_sha256 IS NULL)),
 CHECK((operacion='alta' AND tipo='alta' AND proyeccion_ref IS NOT NULL AND hecho_ref IS NULL)
    OR (operacion='hecho' AND tipo<>'alta' AND proyeccion_ref IS NULL AND hecho_ref IS NOT NULL))
);
ALTER TABLE vec_personal.registro_empleado_b2_recibo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.registro_empleado_b2_recibo FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.registro_empleado_b2_recibo
 FOR ALL TO vec_personal_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.registro_empleado_b2_recibo
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.registro_empleado_b2_recibo
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
REVOKE ALL ON TABLE vec_personal.registro_empleado_b2_recibo FROM PUBLIC,vec_personal_ejecutor;

-- 000019 se instala antes del gobierno 000020. Esta fachada no concede una
-- operación hasta que el validador nominal publicado exista en runtime.
CREATE FUNCTION vec_personal.validar_catalogo_acto_b2_interna(
 p_organismo text,p_tipo text,p_entrada jsonb,p_fecha date
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $catalogo$
DECLARE v_version integer; v_snapshot jsonb;
BEGIN
 IF pg_catalog.to_regprocedure('vec_personal.validar_entrada_registro_empleado_v1(text,text,text,integer,date)') IS NULL THEN
  RAISE EXCEPTION 'catálogo de Personal no instalado' USING ERRCODE='42501'; END IF;
 IF p_organismo IS NULL OR p_organismo !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR p_tipo NOT IN ('regimen','modalidad','situacion','clase_servicio')
    OR pg_catalog.jsonb_typeof(p_entrada) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT pg_catalog.jsonb_object_keys(p_entrada) ORDER BY 1) IS DISTINCT FROM ARRAY['ref','version']
    OR p_entrada->>'ref' IS NULL OR p_entrada->>'ref'=''
    OR p_fecha IS NULL OR NOT pg_catalog.isfinite(p_fecha) THEN
  RAISE EXCEPTION 'entrada de catálogo inválida' USING ERRCODE='22023'; END IF;
 BEGIN v_version:=(p_entrada->>'version')::integer;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'versión de catálogo inválida' USING ERRCODE='22023'; END;
 IF v_version IS NULL OR v_version<1 THEN
  RAISE EXCEPTION 'versión de catálogo inválida' USING ERRCODE='22023'; END IF;
 v_snapshot:=vec_personal.validar_entrada_registro_empleado_v1(p_organismo,p_tipo,p_entrada->>'ref',v_version,p_fecha);
 IF v_snapshot IS NULL OR pg_catalog.jsonb_typeof(v_snapshot) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT pg_catalog.jsonb_object_keys(v_snapshot) ORDER BY 1) IS DISTINCT FROM
       ARRAY['denominacion','estado','huella_sha256','organismo_ref','ref','revision','tipo','version','vigente_desde','vigente_hasta']
    OR v_snapshot->>'organismo_ref' IS DISTINCT FROM p_organismo
    OR v_snapshot->>'tipo' IS DISTINCT FROM p_tipo
    OR v_snapshot->>'ref' IS DISTINCT FROM p_entrada->>'ref'
    OR v_snapshot->>'version' IS DISTINCT FROM v_version::text
    OR v_snapshot->>'estado' IS DISTINCT FROM 'publicada'
    OR v_snapshot->>'huella_sha256' !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'catálogo de Personal devolvió snapshot divergente' USING ERRCODE='55000'; END IF;
 RETURN v_snapshot;
END $catalogo$;
REVOKE ALL ON FUNCTION vec_personal.validar_catalogo_acto_b2_interna(text,text,jsonb,date)
 FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.construir_catalogo_acto_b2_interna(
 p_tipo text,p_organismo text,p_material jsonb,p_fecha date
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $catalogo$
DECLARE r jsonb:=p_material->'regimen'; mo jsonb:=p_material->'modalidad';
 si jsonb:=p_material->'situacion'; cs jsonb:=p_material->'clase_servicio';
 vacio jsonb:='{"ref":"","version":0}'::jsonb;
BEGIN
 IF p_tipo IN ('alta','relacion') THEN
  IF p_tipo='relacion' AND (si IS DISTINCT FROM vacio OR cs IS DISTINCT FROM vacio
     OR p_material->>'clase_ocupacion' IS DISTINCT FROM '') THEN
   RAISE EXCEPTION 'catálogos no aplicables en relación' USING ERRCODE='22023'; END IF;
  RETURN pg_catalog.jsonb_build_object(
   'regimen',vec_personal.validar_catalogo_acto_b2_interna(p_organismo,'regimen',r,p_fecha),
   'modalidad',vec_personal.validar_catalogo_acto_b2_interna(p_organismo,'modalidad',mo,p_fecha));
 ELSIF p_tipo='ocupacion' THEN
  IF r IS DISTINCT FROM vacio OR si IS DISTINCT FROM vacio OR cs IS DISTINCT FROM vacio
     OR p_material->>'clase_ocupacion' NOT IN ('titular','provisional','temporal','reserva') THEN
   RAISE EXCEPTION 'catálogos no aplicables en ocupación' USING ERRCODE='22023'; END IF;
  RETURN pg_catalog.jsonb_build_object(
   'modalidad',vec_personal.validar_catalogo_acto_b2_interna(p_organismo,'modalidad',mo,p_fecha));
 ELSIF p_tipo='servicio' THEN
  IF r IS DISTINCT FROM vacio OR mo IS DISTINCT FROM vacio OR si IS DISTINCT FROM vacio
     OR p_material->>'clase_ocupacion' IS DISTINCT FROM '' THEN
   RAISE EXCEPTION 'catálogos no aplicables en servicio' USING ERRCODE='22023'; END IF;
  RETURN pg_catalog.jsonb_build_object(
   'clase_servicio',vec_personal.validar_catalogo_acto_b2_interna(p_organismo,'clase_servicio',cs,p_fecha));
 ELSIF p_tipo='situacion' THEN
  IF r IS DISTINCT FROM vacio OR mo IS DISTINCT FROM vacio OR cs IS DISTINCT FROM vacio
     OR p_material->>'clase_ocupacion' IS DISTINCT FROM '' THEN
   RAISE EXCEPTION 'catálogos no aplicables en situación' USING ERRCODE='22023'; END IF;
  RETURN pg_catalog.jsonb_build_object(
   'situacion',vec_personal.validar_catalogo_acto_b2_interna(p_organismo,'situacion',si,p_fecha));
 END IF;
 RAISE EXCEPTION 'tipo de hecho sin catálogo' USING ERRCODE='22023';
END $catalogo$;
REVOKE ALL ON FUNCTION vec_personal.construir_catalogo_acto_b2_interna(text,text,jsonb,date)
 FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.registrar_acto_empleado_b2_interna(
 p_operacion text,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $fn$
DECLARE
 m jsonb; c jsonb; d jsonb; x jsonb; a jsonb; proc jsonb;
 v_regimen jsonb; v_modalidad jsonb; v_situacion jsonb; v_clase_servicio jsonb; v_catalogo jsonb;
 v_consumo record; v_acreditacion record; v_proyeccion record; v_estado_persona record; v_previo vec_personal.registro_empleado_b2_recibo%ROWTYPE;
 v_sha text; v_recurso text; v_recurso_sha text; v_efecto text; v_accion text; v_audiencia text;
 v_tipo text; v_persona text; v_empleado text; v_relacion text; v_hecho text; v_pep text;
 v_recibo text; v_ahora timestamptz(6); v_desde date; v_hasta date; v_revision integer;
 v_pep_proc text; v_b1_version numeric; v_b1_ref text; v_b1_proc_version numeric; v_b1_huella text;
 v_org text; v_unidad text; v_plaza record; v_puesto record; v_plaza_uuid uuid; v_puesto_uuid uuid;
 v_cap_desde timestamptz; v_cap_hasta timestamptz; v_dec_hasta timestamptz;
 v_plaza_version text; v_puesto_version text; v_clave uuid; v_esperada bigint; v_rel_esperada bigint;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR pg_catalog.current_setting('TimeZone')<>'UTC'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_operacion NOT IN ('alta','hecho') OR p_material IS NULL
    OR pg_catalog.octet_length(p_material) NOT BETWEEN 1 AND 16384
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'registro de empleado denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
  d:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  x:=pg_catalog.convert_from(p_contexto,'UTF8')::jsonb;
  a:=m->'actor'; proc:=m->'procedencia';
  v_regimen:=m->'regimen'; v_modalidad:=m->'modalidad';
  v_situacion:=m->'situacion'; v_clase_servicio:=m->'clase_servicio';
  v_clave:=(proc->>'idempotencia_ref')::uuid;
  v_esperada:=(m->>'revision_esperada')::bigint;
  v_rel_esperada:=(m->>'relacion_version_esperada')::bigint;
  v_desde:=(m->>'vigente_desde')::date;
  v_hasta:=NULLIF(m->>'vigente_hasta','')::date;
  v_cap_desde:=(c->>'emitida_en')::timestamptz;
  v_cap_hasta:=(c->>'expira_en')::timestamptz;
  v_dec_hasta:=(d->>'valida_hasta')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de empleado inválido' USING ERRCODE='22023'; END;
 IF pg_catalog.jsonb_typeof(m)<>'object' OR pg_catalog.jsonb_typeof(a)<>'object'
    OR pg_catalog.jsonb_typeof(proc)<>'object'
    OR pg_catalog.jsonb_typeof(v_regimen) IS DISTINCT FROM 'object'
    OR pg_catalog.jsonb_typeof(v_modalidad) IS DISTINCT FROM 'object'
    OR ARRAY(SELECT pg_catalog.jsonb_object_keys(v_regimen) ORDER BY 1) IS DISTINCT FROM ARRAY['ref','version']
    OR ARRAY(SELECT pg_catalog.jsonb_object_keys(v_modalidad) ORDER BY 1) IS DISTINCT FROM ARRAY['ref','version']
    OR ARRAY(SELECT pg_catalog.jsonb_object_keys(a) ORDER BY 1) IS DISTINCT FROM
       ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','perfil_ref','perfil_version','persona_ref','persona_version']
    OR ARRAY(SELECT pg_catalog.jsonb_object_keys(proc) ORDER BY 1) IS DISTINCT FROM
       ARRAY['acto_ref','fuente_huella_sha256','fuente_ref','fuente_version','idempotencia_ref']
    OR proc->>'acto_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR proc->>'fuente_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR proc->>'fuente_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR (proc->>'fuente_version')::bigint<1
    OR v_clave::text !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR v_desde IS NULL OR NOT pg_catalog.isfinite(v_desde)
    OR (v_hasta IS NOT NULL AND (NOT pg_catalog.isfinite(v_hasta) OR v_hasta<=v_desde))
    OR a->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR a->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR a->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR a->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR a->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR a->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR a->>'persona_ref' IS DISTINCT FROM x->>'persona_ref'
    OR a->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR a->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR x->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR v_cap_desde IS NULL OR v_cap_hasta IS NULL OR v_dec_hasta IS NULL
    OR NOT pg_catalog.isfinite(v_cap_desde) OR NOT pg_catalog.isfinite(v_cap_hasta)
    OR NOT pg_catalog.isfinite(v_dec_hasta)
    OR v_cap_hasta<=v_cap_desde OR v_dec_hasta<=v_cap_desde
    OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta'
    OR d->>'principal_id' IS DISTINCT FROM a->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM a->>'perfil_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
   RAISE EXCEPTION 'identidad o procedencia divergente' USING ERRCODE='42501';
 END IF;
 IF pg_catalog.to_regprocedure('vec_personal.validar_entrada_registro_empleado_v1(text,text,text,integer,date)') IS NULL THEN
  RAISE EXCEPTION 'catálogo de Personal no instalado' USING ERRCODE='42501'; END IF;
 IF p_operacion='alta' THEN
  IF ARRAY(SELECT pg_catalog.jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
    'actor','esquema','modalidad','operacion','organismo_ref',
    'persona_ref','procedencia','regimen','unidad_ref','version_esperada','vigente_desde','vigente_hasta']
     OR m->>'esquema' IS DISTINCT FROM 'vec.personal.registro-empleado-b2.alta.v1'
     OR m->>'operacion' IS DISTINCT FROM 'alta'
     OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
     OR m->>'organismo_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
     OR m->>'unidad_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
     OR m->>'version_esperada' IS DISTINCT FROM '0' THEN
   RAISE EXCEPTION 'alta de empleado inválida' USING ERRCODE='22023';
  END IF;
  v_efecto:=m->>'persona_ref'; v_tipo:='alta';
  v_accion:='personal.registro_empleado.alta.registrar';
  v_audiencia:='vec_personal.registro_empleado.alta.v1';
  IF d->>'tipo_recurso' IS DISTINCT FROM 'alta_empleado_rrhh'
     OR d->>'finalidad' IS DISTINCT FROM 'registrar_empleado'
     OR d->'campos_permitidos' IS DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb THEN
   RAISE EXCEPTION 'alta sin permiso nominal' USING ERRCODE='42501'; END IF;
 ELSE
  IF ARRAY(SELECT pg_catalog.jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
    'actor','clase_ocupacion','clase_servicio','dias_reconocidos','empleado_ref','esquema','estado',
    'modalidad','operacion','organismo_ref','periodo_desde','periodo_hasta','plaza_ref','procedencia','puesto_ref',
    'regimen','relacion_ref','relacion_version_esperada','revision_esperada','situacion','tipo','unidad_ref','version_plaza_ref',
    'version_puesto_ref','vigente_desde','vigente_hasta']
     OR m->>'esquema' IS DISTINCT FROM 'vec.personal.registro-empleado-b2.hecho.v1'
     OR m->>'operacion' IS DISTINCT FROM 'hecho'
     OR m->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
     OR pg_catalog.jsonb_typeof(v_situacion) IS DISTINCT FROM 'object'
     OR pg_catalog.jsonb_typeof(v_clase_servicio) IS DISTINCT FROM 'object'
     OR ARRAY(SELECT pg_catalog.jsonb_object_keys(v_situacion) ORDER BY 1) IS DISTINCT FROM ARRAY['ref','version']
     OR ARRAY(SELECT pg_catalog.jsonb_object_keys(v_clase_servicio) ORDER BY 1) IS DISTINCT FROM ARRAY['ref','version']
     OR m->>'organismo_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
     OR v_esperada IS NULL OR v_esperada<1 OR v_esperada>2147483647
     OR v_rel_esperada IS NULL OR v_rel_esperada<0 OR v_rel_esperada>2147483647
     OR m->>'tipo' NOT IN ('relacion','ocupacion','servicio','situacion') THEN
   RAISE EXCEPTION 'hecho de empleado inválido' USING ERRCODE='22023'; END IF;
  v_efecto:=m->>'empleado_ref'; v_tipo:=m->>'tipo';
  v_accion:='personal.registro_empleado.hecho.registrar';
  v_audiencia:='vec_personal.registro_empleado.hecho.v1';
  IF d->>'tipo_recurso' IS DISTINCT FROM 'hecho_empleado_rrhh'
     OR d->>'finalidad' IS DISTINCT FROM 'registrar_hecho_empleado'
     OR d->'campos_permitidos' IS DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb THEN
   RAISE EXCEPTION 'hecho sin permiso nominal' USING ERRCODE='42501'; END IF;
 END IF;
 v_org:=m->>'organismo_ref';
 v_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_material,'UTF8')),'hex');
 v_recurso:='{"ambitos":{"objetivo_ref":'||pg_catalog.to_jsonb(v_efecto)::text||',"organismo_ref":'||pg_catalog.to_jsonb(v_org)::text||'},"atributos":{"material_sha256":"'||v_sha||'","operacion":'||pg_catalog.to_jsonb(p_operacion)::text||'}}';
 v_recurso_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_recurso,'UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM v_accion OR c->>'audiencia_consumo' IS DISTINCT FROM v_audiencia
    OR c->>'efecto_ref' IS DISTINCT FROM v_efecto
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
    OR d->>'accion' IS DISTINCT FROM v_accion OR d->>'recurso_ref' IS DISTINCT FROM v_efecto
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha THEN
  RAISE EXCEPTION 'selector o huella V3 divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM v_efecto
    OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_recurso_sha
    OR v_consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'consumo V3 divergente' USING ERRCODE='42501'; END IF;
 IF v_consumo.consumida_en<v_cap_desde OR v_consumo.consumida_en>=v_cap_hasta
    OR v_consumo.consumida_en>=v_dec_hasta THEN
  RAISE EXCEPTION 'consumo V3 fuera de vigencia' USING ERRCODE='42501'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:registro-b2:idempotencia:'||v_clave::text,0));
 SELECT * INTO v_previo FROM vec_personal.registro_empleado_b2_recibo WHERE idempotencia_ref=v_clave;
 IF FOUND THEN
  IF v_previo.operacion IS DISTINCT FROM p_operacion OR v_previo.material_sha256 IS DISTINCT FROM v_sha
     OR v_previo.actor_ref IS DISTINCT FROM a->>'actor_ref' OR v_previo.efecto_ref IS DISTINCT FROM v_efecto THEN
   RAISE EXCEPTION 'clave de registro reutilizada con material distinto' USING ERRCODE='23505'; END IF;
  IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta THEN
   RAISE EXCEPTION 'replay V3 caducado' USING ERRCODE='42501'; END IF;
  RETURN pg_catalog.jsonb_build_object('recibo',
   pg_catalog.jsonb_build_object('recibo_ref',v_previo.recibo_ref,'empleado_ref',v_previo.empleado_ref,
    'relacion_ref',v_previo.relacion_ref,'tipo',v_previo.tipo,'version',v_previo.version,
    'registrado_en',v_previo.registrado_en,
    'decision_ref',v_previo.decision_ref,'efecto_ref',v_previo.efecto_ref,
    'eficacia_administrativa',false,'firma_oficial',false,
    'consumo_huella_sha256',v_previo.consumo_huella_sha256,'auditoria_ref',v_previo.auditoria_ref)
   || CASE WHEN p_operacion='alta' THEN pg_catalog.jsonb_build_object('proyeccion_ref',v_previo.proyeccion_ref)
      ELSE pg_catalog.jsonb_build_object('hecho_ref',v_previo.hecho_ref) END,
   'acceso_actual',pg_catalog.jsonb_build_object('decision_ref',v_consumo.decision_ref,
    'efecto_ref',v_efecto,'consultada_en',v_consumo.consumida_en,'estado_replay','replay',
    'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
    'auditoria_ref',v_consumo.auditoria_ref));
 END IF;
 v_ahora:=v_consumo.consumida_en;
 v_recibo:='perrec_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 IF p_operacion='alta' THEN
  v_persona:=m->>'persona_ref';
  SELECT * INTO v_acreditacion FROM vec_contexto_actor_v1.acreditar_persona_tercero_v1(v_persona);
  IF NOT FOUND THEN RAISE EXCEPTION 'persona ausente' USING ERRCODE='P0002'; END IF;
  IF v_acreditacion.persona_ref IS DISTINCT FROM v_persona
     OR v_acreditacion.persona_version IS NULL OR v_acreditacion.persona_version<1
     OR v_acreditacion.procedencia_ref IS NULL
     OR v_acreditacion.procedencia_huella_sha256 IS NULL
     OR v_acreditacion.procedencia_huella_sha256 !~ '^[0-9a-f]{64}$'
     OR v_acreditacion.procedencia_version IS NULL OR v_acreditacion.procedencia_version<1
     OR v_acreditacion.vigente_desde IS NULL OR v_acreditacion.vigente_hasta IS NULL
     OR v_ahora < v_acreditacion.vigente_desde OR v_ahora>=v_acreditacion.vigente_hasta THEN
    RAISE EXCEPTION 'acreditación de persona caducada o modificada' USING ERRCODE='42501'; END IF;
  v_b1_version:=v_acreditacion.persona_version;
  v_b1_ref:=v_acreditacion.procedencia_ref;
  v_b1_proc_version:=v_acreditacion.procedencia_version;
  v_b1_huella:=v_acreditacion.procedencia_huella_sha256;
  -- La proyección nace del acto gobernado de alta Personal. B1 solo acredita
  -- persona objetivo y su sello permanece separado en el recibo durable.
  v_pep_proc:='prc_'||pg_catalog.substr(pg_catalog.encode(pg_catalog.sha256(
    pg_catalog.convert_to(v_clave::text||'|'||(proc->>'acto_ref')||'|'||v_sha,'UTF8')),'hex'),1,32);
  PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:registro-b2:alta-persona:'||v_persona,0));
  PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(v_persona);
  SELECT * INTO STRICT v_estado_persona FROM vec_personal.resolver_empleado_canonico_persona_v1(v_persona,pg_catalog.clock_timestamp());
  IF v_estado_persona.resultado<>'sin_empleado' THEN
    RAISE EXCEPTION 'persona con alta de empleado existente' USING ERRCODE='23505'; END IF;
  IF EXISTS (SELECT 1 FROM vec_personal.proyeccion_empleado_persona_historia h
     WHERE h.persona_ref=v_persona) THEN
    RAISE EXCEPTION 'persona con registro histórico de empleado' USING ERRCODE='23505'; END IF;
  v_empleado:='emp_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
  v_relacion:='rel_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
  v_pep:='pep_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
  v_unidad:=m->>'unidad_ref';
  v_catalogo:=vec_personal.construir_catalogo_acto_b2_interna('alta',v_org,m,v_desde);
  IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta
     OR pg_catalog.clock_timestamp()>=v_acreditacion.vigente_hasta THEN
   RAISE EXCEPTION 'alta caducada antes de escritura' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_personal.relacion_servicio_historia(
    relacion_ref,revision,persona_ref,empleado_ref,organismo_ref,unidad_ref,regimen_ref,modalidad_ref,
    estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,
    fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
  VALUES(v_relacion,1,v_persona,v_empleado,v_org,v_unidad,v_regimen->>'ref',v_modalidad->>'ref',
    'vigente',v_desde,v_hasta,v_ahora,proc->>'acto_ref',proc->>'fuente_ref',
    (proc->>'fuente_version')::bigint,proc->>'fuente_huella_sha256',v_consumo.decision_ref,v_consumo.auditoria_ref,v_catalogo);
  IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta
     OR pg_catalog.clock_timestamp()>=v_acreditacion.vigente_hasta THEN
   RAISE EXCEPTION 'proyección caducada antes de publicación' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT v_proyeccion FROM vec_personal.publicar_proyeccion_empleado_persona_v1(
    v_pep,1,v_persona,v_empleado,'activa',v_ahora,
    v_acreditacion.vigente_hasta,NULL,v_pep_proc,
    (proc->>'fuente_version')::bigint,v_sha);
  v_revision:=1;
 ELSE
  v_empleado:=m->>'empleado_ref';
  SELECT h.persona_ref INTO v_persona FROM (
    SELECT DISTINCT ON (p.proyeccion_ref) p.*
      FROM vec_personal.proyeccion_empleado_persona_historia p
     WHERE p.empleado_ref=v_empleado ORDER BY p.proyeccion_ref,p.version DESC
   ) h
   WHERE h.estado='activa' AND h.vigente_desde<=v_ahora AND v_ahora<h.vigente_hasta;
  IF v_persona IS NULL THEN RAISE EXCEPTION 'empleado ausente' USING ERRCODE='P7404'; END IF;
  PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(v_persona);
  SELECT * INTO STRICT v_estado_persona FROM vec_personal.resolver_empleado_canonico_persona_v1(
    v_persona,pg_catalog.clock_timestamp());
  IF v_estado_persona.resultado IS DISTINCT FROM 'empleado'
     OR v_estado_persona.empleado_ref IS DISTINCT FROM v_empleado THEN
    RAISE EXCEPTION 'proyección de empleado no vigente' USING ERRCODE='42501'; END IF;
  IF v_tipo='relacion' THEN
   IF m->>'unidad_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
      OR m->>'estado' NOT IN ('vigente','suspendida','finalizada') THEN
    RAISE EXCEPTION 'relación nueva inválida' USING ERRCODE='22023'; END IF;
   -- Una relación nueva declara su organismo propio y queda ligada al
   -- ámbito exacto V3 por la huella de recurso, incluso en empleados multi-org.
   IF m->>'relacion_ref'='' AND v_rel_esperada=0 AND v_esperada=1 THEN
    v_relacion:='rel_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
    v_revision:=1;
   ELSIF m->>'relacion_ref' ~ '^rel_[A-Za-z0-9_-]{22,128}$' AND v_rel_esperada>0
       AND v_esperada=v_rel_esperada+1 THEN
    v_relacion:=m->>'relacion_ref';
    SELECT h.revision,h.organismo_ref,h.persona_ref INTO v_plaza
      FROM vec_personal.relacion_servicio_historia h
      WHERE h.relacion_ref=v_relacion AND h.empleado_ref=v_empleado
      ORDER BY h.revision DESC LIMIT 1;
    IF NOT FOUND THEN RAISE EXCEPTION 'relación no encontrada' USING ERRCODE='P7404'; END IF;
    IF v_plaza.revision<>v_rel_esperada OR v_plaza.persona_ref IS DISTINCT FROM v_persona THEN
      RAISE EXCEPTION 'revisión de relación divergente' USING ERRCODE='23505'; END IF;
    IF v_plaza.organismo_ref IS DISTINCT FROM v_org THEN
     RAISE EXCEPTION 'organismo de relación fuera de ámbito' USING ERRCODE='42501'; END IF;
    v_revision:=v_esperada::integer;
   ELSE RAISE EXCEPTION 'selección de relación inválida' USING ERRCODE='22023'; END IF;
   v_hecho:=v_relacion;
   v_catalogo:=vec_personal.construir_catalogo_acto_b2_interna('relacion',v_org,m,v_desde);
   IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta THEN
    RAISE EXCEPTION 'relación caducada antes de escritura' USING ERRCODE='42501'; END IF;
   INSERT INTO vec_personal.relacion_servicio_historia(
     relacion_ref,revision,persona_ref,empleado_ref,organismo_ref,unidad_ref,regimen_ref,
     modalidad_ref,estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,
     fuente_version,fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
   VALUES(v_relacion,v_revision,v_persona,v_empleado,v_org,m->>'unidad_ref',v_regimen->>'ref',
     v_modalidad->>'ref',m->>'estado',v_desde,v_hasta,v_ahora,proc->>'acto_ref',
     proc->>'fuente_ref',(proc->>'fuente_version')::bigint,proc->>'fuente_huella_sha256',
     v_consumo.decision_ref,v_consumo.auditoria_ref,v_catalogo);
  ELSE
   v_relacion:=m->>'relacion_ref';
   IF v_relacion !~ '^rel_[A-Za-z0-9_-]{22,128}$' THEN
    RAISE EXCEPTION 'selector de relación inválido' USING ERRCODE='22023'; END IF;
   SELECT h.revision,h.organismo_ref,h.unidad_ref,h.vigente_desde,h.vigente_hasta,h.estado
     INTO v_plaza FROM vec_personal.relacion_servicio_historia h
     WHERE h.relacion_ref=v_relacion AND h.empleado_ref=v_empleado
     ORDER BY h.revision DESC LIMIT 1;
   IF NOT FOUND THEN RAISE EXCEPTION 'relación no encontrada' USING ERRCODE='P7404'; END IF;
   IF v_plaza.revision<>v_rel_esperada OR v_esperada<>1 THEN
    RAISE EXCEPTION 'revisión de relación divergente' USING ERRCODE='23505'; END IF;
   IF v_plaza.organismo_ref IS DISTINCT FROM v_org THEN
    RAISE EXCEPTION 'organismo de relación fuera de ámbito' USING ERRCODE='42501'; END IF;
   IF v_plaza.estado<>'vigente' OR v_desde<v_plaza.vigente_desde
      OR (v_plaza.vigente_hasta IS NOT NULL AND v_desde>=v_plaza.vigente_hasta) THEN
    RAISE EXCEPTION 'relación fuera de vigencia' USING ERRCODE='42501'; END IF;
   v_org:=v_plaza.organismo_ref;
   v_catalogo:=vec_personal.construir_catalogo_acto_b2_interna(v_tipo,v_org,m,v_desde);
   IF v_tipo='ocupacion' THEN
    BEGIN
     v_plaza_uuid:=NULLIF(pg_catalog.replace(m->>'plaza_ref','plaza:',''),'')::uuid;
     v_puesto_uuid:=NULLIF(pg_catalog.replace(m->>'puesto_ref','puesto:',''),'')::uuid;
    EXCEPTION WHEN others THEN RAISE EXCEPTION 'referencia de plaza o puesto inválida' USING ERRCODE='22023'; END;
    SELECT p.* INTO v_plaza FROM vec_personal.plaza_plantilla_historia p
     WHERE p.plaza_ref=v_plaza_uuid ORDER BY p.revision DESC LIMIT 1;
    IF NOT FOUND OR v_plaza.organismo_ref IS DISTINCT FROM v_org
       OR v_plaza.unidad_ref IS DISTINCT FROM m->>'unidad_ref'
       OR v_plaza.estado_estructural<>'vigente' OR v_plaza.retirado
       OR v_plaza.dotacion_presupuestaria<>'acreditada' THEN
     RAISE EXCEPTION 'plaza no acreditada' USING ERRCODE='42501'; END IF;
    v_plaza_version:='plantilla:'||v_plaza.plantilla_version_ref::text;
    IF m->>'version_plaza_ref' IS DISTINCT FROM v_plaza_version THEN
     RAISE EXCEPTION 'versión de plaza divergente' USING ERRCODE='23505'; END IF;
    IF v_puesto_uuid IS NOT NULL THEN
     SELECT p.*,t.rpt_version_ref INTO v_puesto FROM vec_personal.puesto_rpt_historia p
      JOIN vec_personal.puesto_tipo_historia t ON t.tipo_ref=p.tipo_ref AND t.revision=p.tipo_revision
      WHERE p.puesto_ref=v_puesto_uuid ORDER BY p.revision DESC LIMIT 1;
     IF NOT FOUND OR v_puesto.organismo_ref IS DISTINCT FROM v_org
        OR v_puesto.estado_estructural<>'vigente' OR v_puesto.retirado THEN
      RAISE EXCEPTION 'puesto no acreditado' USING ERRCODE='42501'; END IF;
     v_puesto_version:='rpt:'||v_puesto.rpt_version_ref::text;
     IF m->>'version_puesto_ref' IS DISTINCT FROM v_puesto_version THEN
      RAISE EXCEPTION 'versión de puesto divergente' USING ERRCODE='23505'; END IF;
    ELSIF m->>'version_puesto_ref'<>'' THEN
     RAISE EXCEPTION 'versión de puesto sin puesto' USING ERRCODE='22023'; END IF;
    IF m->>'clase_ocupacion' NOT IN ('titular','provisional','temporal','reserva') THEN
      RAISE EXCEPTION 'ocupación inválida' USING ERRCODE='22023'; END IF;
    v_hecho:='ocu_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
    IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta THEN
     RAISE EXCEPTION 'ocupación caducada antes de escritura' USING ERRCODE='42501'; END IF;
    INSERT INTO vec_personal.ocupacion_empleado_historia(
      ocupacion_ref,revision,relacion_ref,relacion_revision,empleado_ref,organismo_ref,
      unidad_ref,plaza_ref,plaza_revision,puesto_ref,puesto_revision,clase,modalidad_ref,
      estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,
      fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
    VALUES(v_hecho,1,v_relacion,v_rel_esperada::integer,v_empleado,v_org,m->>'unidad_ref',
      v_plaza_uuid,v_plaza.revision,v_puesto_uuid,CASE WHEN v_puesto_uuid IS NULL THEN NULL ELSE v_puesto.revision END,
      m->>'clase_ocupacion',v_modalidad->>'ref','vigente',v_desde,v_hasta,v_ahora,
      proc->>'acto_ref',proc->>'fuente_ref',(proc->>'fuente_version')::bigint,
      proc->>'fuente_huella_sha256',v_consumo.decision_ref,v_consumo.auditoria_ref,v_catalogo);
   ELSIF v_tipo='servicio' THEN
    IF m->>'estado' NOT IN ('declarado','comprobado','reconocido')
       OR (m->>'dias_reconocidos')::bigint NOT BETWEEN 0 AND 2147483647
       OR (m->>'periodo_desde')::date >= (m->>'periodo_hasta')::date THEN
      RAISE EXCEPTION 'servicio inválido' USING ERRCODE='22023'; END IF;
    v_hecho:='srv_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
    IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta THEN
     RAISE EXCEPTION 'servicio caducado antes de escritura' USING ERRCODE='42501'; END IF;
    INSERT INTO vec_personal.servicio_reconocido_historia(
      servicio_ref,revision,relacion_ref,relacion_revision,empleado_ref,organismo_ref,
      clase_ref,dias_reconocidos,periodo_desde,periodo_hasta,estado,vigente_desde,
      vigente_hasta,conocido_desde,acto_ref,fuente_ref,fuente_version,
      fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
    VALUES(v_hecho,1,v_relacion,v_rel_esperada::integer,v_empleado,v_org,v_clase_servicio->>'ref',
      (m->>'dias_reconocidos')::integer,(m->>'periodo_desde')::date,(m->>'periodo_hasta')::date,
      m->>'estado',v_desde,v_hasta,v_ahora,proc->>'acto_ref',proc->>'fuente_ref',
      (proc->>'fuente_version')::bigint,proc->>'fuente_huella_sha256',v_consumo.decision_ref,v_consumo.auditoria_ref,v_catalogo);
   ELSE
    IF m->>'estado' NOT IN ('vigente','finalizada','rectificada') THEN
      RAISE EXCEPTION 'situación inválida' USING ERRCODE='22023'; END IF;
    v_hecho:='sit_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
    IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta THEN
     RAISE EXCEPTION 'situación caducada antes de escritura' USING ERRCODE='42501'; END IF;
    INSERT INTO vec_personal.situacion_empleado_historia(
      situacion_ref,revision,relacion_ref,relacion_revision,empleado_ref,organismo_ref,
      situacion_codigo,estado,vigente_desde,vigente_hasta,conocido_desde,acto_ref,
      fuente_ref,fuente_version,fuente_huella_sha256,decision_ref,auditoria_ref,catalogo_snapshot)
    VALUES(v_hecho,1,v_relacion,v_rel_esperada::integer,v_empleado,v_org,v_situacion->>'ref',
      m->>'estado',v_desde,v_hasta,v_ahora,proc->>'acto_ref',proc->>'fuente_ref',
      (proc->>'fuente_version')::bigint,proc->>'fuente_huella_sha256',
      v_consumo.decision_ref,v_consumo.auditoria_ref,v_catalogo);
   END IF;
   v_revision:=1;
  END IF;
 END IF;
 IF pg_catalog.clock_timestamp()>=v_cap_hasta OR pg_catalog.clock_timestamp()>=v_dec_hasta THEN
  RAISE EXCEPTION 'recibo caducado antes de escritura' USING ERRCODE='42501'; END IF;
 IF p_operacion='alta' THEN
  IF pg_catalog.clock_timestamp()>=v_acreditacion.vigente_hasta THEN
   RAISE EXCEPTION 'persona caducada antes de recibo' USING ERRCODE='42501'; END IF;
 END IF;
 INSERT INTO vec_personal.registro_empleado_b2_recibo(
   idempotencia_ref,operacion,material_sha256,actor_ref,efecto_ref,recibo_ref,empleado_ref,
   relacion_ref,proyeccion_ref,hecho_ref,tipo,version,decision_ref,consumo_huella_sha256,
   auditoria_ref,registrado_en,catalogo_snapshot,acreditacion_persona_version,acreditacion_procedencia_ref,
   acreditacion_procedencia_version,acreditacion_procedencia_huella_sha256)
 VALUES(v_clave,p_operacion,v_sha,a->>'actor_ref',v_efecto,v_recibo,v_empleado,
   v_relacion,v_pep,v_hecho,v_tipo,v_revision,v_consumo.decision_ref,
   v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_ahora,v_catalogo,
   v_b1_version,v_b1_ref,v_b1_proc_version,v_b1_huella);
 RETURN pg_catalog.jsonb_build_object('recibo',
   pg_catalog.jsonb_build_object('recibo_ref',v_recibo,'empleado_ref',v_empleado,
    'relacion_ref',v_relacion,'tipo',v_tipo,'version',v_revision,
    'registrado_en',v_ahora,
    'decision_ref',v_consumo.decision_ref,'efecto_ref',v_efecto,
    'eficacia_administrativa',false,'firma_oficial',false,
    'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
    'auditoria_ref',v_consumo.auditoria_ref)
   || CASE WHEN p_operacion='alta' THEN pg_catalog.jsonb_build_object('proyeccion_ref',v_pep)
      ELSE pg_catalog.jsonb_build_object('hecho_ref',v_hecho) END,
   'acceso_actual',pg_catalog.jsonb_build_object('decision_ref',v_consumo.decision_ref,
    'efecto_ref',v_efecto,'consultada_en',v_consumo.consumida_en,'estado_replay','registrado',
    'consumo_huella_sha256',v_consumo.consumo_huella_sha256,
    'auditoria_ref',v_consumo.auditoria_ref));
END $fn$;
REVOKE ALL ON FUNCTION vec_personal.registrar_acto_empleado_b2_interna(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.registrar_empleado_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $fn$
 SELECT vec_personal.registrar_acto_empleado_b2_interna('alta',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11);
$fn$;
CREATE FUNCTION vec_personal.registrar_hecho_empleado_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $fn$
 SELECT vec_personal.registrar_acto_empleado_b2_interna('hecho',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11);
$fn$;
REVOKE ALL ON FUNCTION vec_personal.registrar_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_personal.registrar_hecho_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_hecho_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMIT;
