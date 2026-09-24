\set ON_ERROR_STOP on
-- Sólo prueba de lógica local: el consumidor AD3 aquí es un stub TEST-ONLY.
CREATE SCHEMA vec_prueba_d7c;
GRANT USAGE ON SCHEMA vec_prueba_d7c,vec_prueba_d7 TO vec_personal_d7_ejecutor;
CREATE FUNCTION vec_prueba_d7c.material(p_modo text,p_solicitud text DEFAULT '',p_clave text DEFAULT '018f1d1e-1234-4abc-8def-0123456789ab',p_fecha date DEFAULT current_date)
RETURNS text LANGUAGE plpgsql STABLE AS $f$
DECLARE propio boolean:=p_modo IN ('solicitar','consultar');
BEGIN
 RETURN jsonb_build_object(
  'asignacion_ref',CASE WHEN p_modo IN ('solicitar','confirmar') THEN 'ads_abcdefghijklmnopqrstuv' ELSE '' END,
  'campos_a_revisar',CASE WHEN p_modo='solicitar' THEN '["centro_ref"]'::jsonb ELSE '[]'::jsonb END,
  'clave_idempotencia',CASE WHEN p_modo='consultar' THEN '' ELSE p_clave END,
  'detalle_solicitado','','empleado_ref','emp_abcdefghijklmnopqrstuv',
  'esquema','vec.personal.rectificacion-dietas.v1','fecha_referencia',p_fecha::text,
  'identidad',jsonb_build_object('actor_ref',CASE WHEN propio THEN 'actor-sujeto' ELSE 'actor-administrativo' END,
    'contexto_actor_ref','ctx-a','contexto_version','1','cuenta_ref','cuenta-a',
    'cuenta_version','1','empleado_ref',CASE WHEN propio THEN 'emp_abcdefghijklmnopqrstuv' ELSE 'emp_bbbbbbbbbbbbbbbbbbbbbb' END,
    'perfil_ref','perfil-a','perfil_version','1',
    'persona_ref',CASE WHEN propio THEN 'per_abcdefghijklmnopqrstuv' ELSE 'per_bbbbbbbbbbbbbbbbbbbbbb' END,
    'persona_version','1'),
  'motivo_revision',CASE p_modo WHEN 'solicitar' THEN 'Revisar centro' WHEN 'rechazar' THEN 'No procede' WHEN 'confirmar' THEN 'Confirmar centro' ELSE '' END,
  'operacion',p_modo,'persona_ref','per_abcdefghijklmnopqrstuv',
  'relacion_ref','rel_abcdefghijklmnopqrstuv','solicitud_ref',p_solicitud,
  'unidad_ref','unidad-sintetica',
  'version_esperada',CASE WHEN p_modo IN ('solicitar','confirmar') THEN 1 ELSE 0 END)::text;
END $f$;
CREATE FUNCTION vec_prueba_d7c.capacidad(p_modo text) RETURNS bytea LANGUAGE sql IMMUTABLE AS $f$
 SELECT convert_to(jsonb_build_object(
  'audiencia_consumo',CASE p_modo WHEN 'solicitar' THEN 'vec_personal.asignacion_dietas.rectificacion.solicitar.v1'
   WHEN 'consultar' THEN 'vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1'
   ELSE 'vec_personal.asignacion_dietas.rectificacion.resolver.v1' END,
  'operacion',CASE p_modo WHEN 'solicitar' THEN 'personal.asignacion_dietas.rectificacion.solicitar'
   WHEN 'consultar' THEN 'personal.asignacion_dietas.rectificacion.propia.consultar'
   ELSE 'personal.asignacion_dietas.rectificacion.resolver' END)::text,'UTF8')
$f$;
CREATE FUNCTION vec_prueba_d7c.decision(p_modo text,p_solicitud text DEFAULT '',p_clave text DEFAULT '018f1d1e-1234-4abc-8def-0123456789ab',p_fecha date DEFAULT current_date)
RETURNS bytea LANGUAGE plpgsql STABLE AS $f$
DECLARE mat text:=vec_prueba_d7c.material(p_modo,p_solicitud,p_clave,p_fecha);
 h text; recurso text; accion text; finalidad text;
BEGIN
 h:=encode(sha256(convert_to(mat,'UTF8')),'hex');
 recurso:='{"ambitos":{"empleado_ref":"emp_abcdefghijklmnopqrstuv","persona_ref":"per_abcdefghijklmnopqrstuv","relacion_ref":"rel_abcdefghijklmnopqrstuv","unidad_ref":"unidad-sintetica"},"atributos":{"fecha_referencia":"'||p_fecha::text||'","material_sha256":"'||h||'","operacion":"'||p_modo||'"}}';
 accion:=CASE p_modo WHEN 'solicitar' THEN 'personal.asignacion_dietas.rectificacion.solicitar'
   WHEN 'consultar' THEN 'personal.asignacion_dietas.rectificacion.propia.consultar'
   ELSE 'personal.asignacion_dietas.rectificacion.resolver' END;
 finalidad:=CASE p_modo WHEN 'solicitar' THEN 'solicitar_rectificacion_dietas'
   WHEN 'consultar' THEN 'consultar_rectificacion_dietas_propia'
   ELSE 'resolver_rectificacion_dietas' END;
 RETURN convert_to(jsonb_build_object('concedida','true','accion',accion,
  'modulo_id','personal','tipo_recurso','rectificacion_asignacion_dietas',
  'recurso_ref','rel_abcdefghijklmnopqrstuv','finalidad',finalidad,
  'obligaciones','[]'::jsonb,
  'principal_id',CASE WHEN p_modo IN ('rechazar','confirmar') THEN 'actor-administrativo' ELSE 'actor-sujeto' END,
  'perfil_activo_ref','perfil-a',
  'contexto_recurso_huella_sha256',encode(sha256(convert_to(recurso,'UTF8')),'hex'),
  'campos_permitidos','["asignacion_ref","auditoria_ad3_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado","recibo_ref","registrada_en","solicitud_ref","version_origen"]'::jsonb)::text,'UTF8');
END $f$;
CREATE FUNCTION vec_prueba_d7c.material_correccion(p_solicitud text,p_clave text)
RETURNS text LANGUAGE sql STABLE AS $f$
 SELECT jsonb_set(jsonb_set(jsonb_set(
  vec_prueba_d7.material('corregir',p_clave,'centro-corregido',1::smallint)::jsonb,
  '{operacion}','"corregir"'::jsonb),
  '{motivo_revision}','"Confirmar centro"'::jsonb),
  '{procedencia_acto_ref}',to_jsonb(p_solicitud))::text
$f$;
CREATE FUNCTION vec_prueba_d7c.decision_correccion(p_solicitud text,p_clave text)
RETURNS bytea LANGUAGE plpgsql STABLE AS $f$
DECLARE mat text:=vec_prueba_d7c.material_correccion(p_solicitud,p_clave);
 h text; recurso text;
BEGIN
 h:=encode(sha256(convert_to(mat,'UTF8')),'hex');
 recurso:='{"ambitos":{"empleado_ref":"emp_abcdefghijklmnopqrstuv","persona_ref":"per_abcdefghijklmnopqrstuv","relacion_ref":"rel_abcdefghijklmnopqrstuv","unidad_ref":"unidad-sintetica"},"atributos":{"fecha_referencia":"'||current_date::text||'","material_sha256":"'||h||'","operacion":"corregir"}}';
 RETURN convert_to(jsonb_build_object(
  'concedida','true','accion','personal.asignacion_dietas.corregir',
  'modulo_id','personal','tipo_recurso','asignacion_dietas',
  'recurso_ref','rel_abcdefghijklmnopqrstuv','finalidad','corregir_asignacion_dietas',
  'obligaciones','[]'::jsonb,'principal_id','actor-administrativo',
  'decision_ref','dec_corr_d7c',
  'perfil_activo_ref','perfil-a',
  'contexto_recurso_huella_sha256',encode(sha256(convert_to(recurso,'UTF8')),'hex'),
  'campos_permitidos','["administrativo_persona_ref","asignacion_ref","auditoria_ad3_ref","centro_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado_local","grupo_dieta","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","unidad_ref","version","vigente_desde"]'::jsonb)::text,'UTF8');
END $f$;
CREATE FUNCTION vec_prueba_d7c.material_competentes()
RETURNS text LANGUAGE sql STABLE AS $f$
 SELECT jsonb_build_object('esquema','vec.personal.rectificaciones-dietas.competentes.v1',
  'fecha_referencia',current_date::text,
  'identidad',(vec_prueba_d7.material('corregir')::jsonb)->'identidad')::text
$f$;
CREATE FUNCTION vec_prueba_d7c.capacidad_competentes()
RETURNS bytea LANGUAGE sql IMMUTABLE AS $f$
 SELECT convert_to(jsonb_build_object(
  'audiencia_consumo','vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1',
  'operacion','personal.asignacion_dietas.rectificacion.competente.consultar')::text,'UTF8')
$f$;
CREATE FUNCTION vec_prueba_d7c.decision_competentes()
RETURNS bytea LANGUAGE plpgsql STABLE AS $f$
DECLARE mat text:=vec_prueba_d7c.material_competentes(); h text; recurso text;
BEGIN
 h:=encode(sha256(convert_to(mat,'UTF8')),'hex');
 recurso:='{"ambitos":{"persona_ref":"per_bbbbbbbbbbbbbbbbbbbbbb"},"atributos":{"fecha_referencia":"'||current_date::text||'","material_sha256":"'||h||'","operacion":"lista"}}';
 RETURN convert_to(jsonb_build_object(
  'concedida','true','accion','personal.asignacion_dietas.rectificacion.competente.consultar',
  'modulo_id','personal','tipo_recurso','rectificaciones_competentes_dietas',
  'recurso_ref','per_bbbbbbbbbbbbbbbbbbbbbb',
  'finalidad','consultar_rectificaciones_dietas_competentes',
  'obligaciones','[]'::jsonb,'principal_id','actor-administrativo',
  'perfil_activo_ref','perfil-a',
  'contexto_recurso_huella_sha256',encode(sha256(convert_to(recurso,'UTF8')),'hex'),
  'campos_permitidos','["administrativo_persona_ref","asignacion_actual","asignacion_ref","auditoria_ad3_ref","campos_a_revisar","cardinalidad","centro_ref","consultada_en","consumo_huella_sha256","decision_ref","detalle_solicitado","efecto_ref","empleado_ref","estado","fecha_referencia","grupo_dieta","motivo_revision","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","solicitud_ref","solicitudes","unidad_ref","version","version_origen","vigente_desde"]'::jsonb)::text,'UTF8');
END $f$;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA vec_prueba_d7c,vec_prueba_d7 TO vec_personal_d7_ejecutor;

BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
INSERT INTO vec_personal.asignacion_dietas VALUES(
 'ads_abcdefghijklmnopqrstuv','rel_abcdefghijklmnopqrstuv',
 'per_abcdefghijklmnopqrstuv','unidad-sintetica',1,'centro-sintetico',
 'per_bbbbbbbbbbbbbbbbbbbbbb','per_cccccccccccccccccccccc',1,
 DATE '2026-01-01','alta sintética','acto-sintetico','actor-sintetico',clock_timestamp());
COMMIT;

-- Sin fuente de destino, una solicitud puede existir pero confirmar no tiene
-- efecto. El ROLLBACK conserva la base de este ensayo limpia para el puente.
SET SESSION AUTHORIZATION vec_prueba_d7c_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $sin_fuente$
DECLARE r record;
BEGIN
 SELECT * INTO r FROM vec_personal.solicitar_rectificacion_dietas_v1(
  vec_prueba_d7c.material('solicitar'),vec_prueba_d7c.capacidad('solicitar'),
  vec_prueba_d7c.decision('solicitar'),'m',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
 BEGIN
  PERFORM vec_personal.confirmar_rectificacion_dietas_v1(
   vec_prueba_d7c.material('confirmar',r.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   vec_prueba_d7c.capacidad('confirmar'),
   vec_prueba_d7c.decision('confirmar',r.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r',
   vec_prueba_d7c.material_correccion(r.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   vec_prueba_d7.capacidad('corregir','corr-d7c-000000000000'),
   vec_prueba_d7c.decision_correccion(r.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'sin fuente confirmó';
 EXCEPTION WHEN SQLSTATE '55000' THEN
  IF SQLERRM<>'fuente gobernada de destino Personal no disponible'
  THEN RAISE; END IF;
 END;
END $sin_fuente$;
ROLLBACK;
RESET SESSION AUTHORIZATION;

-- TEST-ONLY: autoriza un único destino sintético; no acredita catálogo ni competencia.
CREATE FUNCTION vec_personal.validar_destino_asignacion_dietas_v1(
 text,text,text,text,text,text,date) RETURNS boolean
LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT $1='rel_abcdefghijklmnopqrstuv'
    AND $2='per_abcdefghijklmnopqrstuv' AND $3='unidad-sintetica'
    AND $4='centro-corregido'
    AND $5='per_bbbbbbbbbbbbbbbbbbbbbb'
    AND $6='per_cccccccccccccccccccccc'
    AND $7=current_date
$f$;
ALTER FUNCTION vec_personal.validar_destino_asignacion_dietas_v1(text,text,text,text,text,text,date)
 OWNER TO vec_personal_propietario;
REVOKE ALL ON FUNCTION vec_personal.validar_destino_asignacion_dietas_v1(text,text,text,text,text,text,date)
 FROM PUBLIC,vec_personal_d7_ejecutor,vec_personal_ejecutor;

SET SESSION AUTHORIZATION vec_prueba_d7c_personal;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $recorrido$
DECLARE a record; b record; c record; d record; l record;
BEGIN
 BEGIN
  PERFORM vec_personal.solicitar_rectificacion_dietas_v1(
   vec_prueba_d7c.material('solicitar','','018f1d1e-1234-4abc-8def-0123456789a0',current_date-1),
   vec_prueba_d7c.capacidad('solicitar'),
   vec_prueba_d7c.decision('solicitar','','018f1d1e-1234-4abc-8def-0123456789a0',current_date-1),
   'm',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'alta histórica aceptada';
 EXCEPTION WHEN SQLSTATE 'P7201' THEN NULL; END;
 SELECT * INTO a FROM vec_personal.solicitar_rectificacion_dietas_v1(
  vec_prueba_d7c.material('solicitar'),vec_prueba_d7c.capacidad('solicitar'),
  vec_prueba_d7c.decision('solicitar'),'m',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
 IF a.estado<>'pendiente' OR a.version_origen<>1 THEN RAISE EXCEPTION 'solicitud no registrada'; END IF;
 SELECT * INTO l FROM vec_personal.consultar_rectificaciones_competentes_dietas_v1(
  vec_prueba_d7c.material_competentes(),vec_prueba_d7c.capacidad_competentes(),
  vec_prueba_d7c.decision_competentes(),'m',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
 IF l.cardinalidad<>1 OR l.recibo_ref !~ '^rrd_[0-9a-f]{32}$'
    OR l.solicitudes::jsonb->0->>'solicitud_ref' IS DISTINCT FROM a.solicitud_ref
    OR l.solicitudes::jsonb->0->'asignacion_actual'->>'centro_ref' IS DISTINCT FROM 'centro-sintetico'
 THEN RAISE EXCEPTION 'lectura competente'; END IF;
 SELECT * INTO b FROM vec_personal.solicitar_rectificacion_dietas_v1(
  vec_prueba_d7c.material('solicitar'),vec_prueba_d7c.capacidad('solicitar'),
  vec_prueba_d7c.decision('solicitar'),'m',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
 IF b.estado<>'replay_confirmado' OR b.recibo_ref<>a.recibo_ref THEN RAISE EXCEPTION 'replay solicitud'; END IF;
 SELECT * INTO c FROM vec_personal.consultar_rectificacion_dietas_v1(
  vec_prueba_d7c.material('consultar'),vec_prueba_d7c.capacidad('consultar'),
  vec_prueba_d7c.decision('consultar'),'m',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
 IF c.estado<>'pendiente' OR c.solicitud_ref<>a.solicitud_ref OR c.recibo_ref=a.recibo_ref
 THEN RAISE EXCEPTION 'consulta propia'; END IF;
 SELECT * INTO d FROM vec_personal.resolver_rectificacion_dietas_v1(
  vec_prueba_d7c.material('rechazar',a.solicitud_ref),vec_prueba_d7c.capacidad('rechazar'),
  vec_prueba_d7c.decision('rechazar',a.solicitud_ref),'m',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
 IF d.estado<>'rechazada' OR d.solicitud_ref<>a.solicitud_ref THEN RAISE EXCEPTION 'rechazo'; END IF;
 SELECT * INTO b FROM vec_personal.resolver_rectificacion_dietas_v1(
  vec_prueba_d7c.material('rechazar',a.solicitud_ref),vec_prueba_d7c.capacidad('rechazar'),
  vec_prueba_d7c.decision('rechazar',a.solicitud_ref),'m',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
 IF b.estado<>'replay_confirmado' OR b.recibo_ref<>d.recibo_ref THEN RAISE EXCEPTION 'replay rechazo'; END IF;
 SELECT * INTO l FROM vec_personal.consultar_rectificaciones_competentes_dietas_v1(
  vec_prueba_d7c.material_competentes(),vec_prueba_d7c.capacidad_competentes(),
  vec_prueba_d7c.decision_competentes(),'m',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
 IF l.cardinalidad<>0 OR l.solicitudes::jsonb IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'lectura competente vacía'; END IF;
 SELECT * INTO a FROM vec_personal.solicitar_rectificacion_dietas_v1(
  vec_prueba_d7c.material('solicitar','','018f1d1e-1234-4abc-8def-0123456789ac'),
  vec_prueba_d7c.capacidad('solicitar'),
  vec_prueba_d7c.decision('solicitar','','018f1d1e-1234-4abc-8def-0123456789ac'),
  'm',vec_prueba_d7.contexto('consultar'),1,1,'p','s','e','r');
 BEGIN
  PERFORM vec_personal.confirmar_rectificacion_dietas_v1(
   vec_prueba_d7c.material('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   vec_prueba_d7c.capacidad('confirmar'),
   vec_prueba_d7c.decision('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r',
   jsonb_set(vec_prueba_d7c.material_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad')::jsonb,
    '{administrativo_persona_ref}',to_jsonb('per_dddddddddddddddddddddd'::text))::text,
   vec_prueba_d7.capacidad('corregir','corr-d7c-000000000000'),
   vec_prueba_d7c.decision_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'validador sintético aceptó destinatario no acreditado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT * INTO d FROM vec_personal.confirmar_rectificacion_dietas_v1(
  vec_prueba_d7c.material('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  vec_prueba_d7c.capacidad('confirmar'),
  vec_prueba_d7c.decision('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r',
  vec_prueba_d7c.material_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  vec_prueba_d7.capacidad('corregir','corr-d7c-000000000001'),
  vec_prueba_d7c.decision_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
 IF d.estado<>'confirmada' OR d.solicitud_ref<>a.solicitud_ref
 THEN RAISE EXCEPTION 'confirmación atómica'; END IF;
 SELECT * INTO b FROM vec_personal.confirmar_rectificacion_dietas_v1(
  vec_prueba_d7c.material('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  vec_prueba_d7c.capacidad('confirmar'),
  vec_prueba_d7c.decision('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r',
  vec_prueba_d7c.material_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  vec_prueba_d7.capacidad('corregir','corr-d7c-000000000002'),
  vec_prueba_d7c.decision_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
  'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
 IF b.estado<>'replay_confirmado' OR b.recibo_ref<>d.recibo_ref
 THEN RAISE EXCEPTION 'replay confirmación'; END IF;
 BEGIN
  PERFORM vec_personal.confirmar_rectificacion_dietas_v1(
   vec_prueba_d7c.material('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   vec_prueba_d7c.capacidad('confirmar'),
   vec_prueba_d7c.decision('confirmar',a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r',
   jsonb_set(vec_prueba_d7c.material_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad')::jsonb,
    '{centro_ref}',to_jsonb('otro-centro'::text))::text,
   vec_prueba_d7.capacidad('corregir','corr-d7c-000000000003'),
   vec_prueba_d7c.decision_correccion(a.solicitud_ref,'018f1d1e-1234-4abc-8def-0123456789ad'),
   'm',vec_prueba_d7.contexto('correccion'),1,1,'p','s','e','r');
  RAISE EXCEPTION 'replay de corrección distinta aceptado';
 EXCEPTION WHEN SQLSTATE 'P7204' THEN NULL; END;
END $recorrido$;
COMMIT;
RESET SESSION AUTHORIZATION;

BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SELECT set_config('vec.dietas.persona_ref','per_abcdefghijklmnopqrstuv',true);
DO $conteo$
BEGIN
 PERFORM set_config('vec.dietas.competencias_actor_persona_ref','per_bbbbbbbbbbbbbbbbbbbbbb',true);
 IF (SELECT count(*) FROM vec_personal.solicitud_rectificacion_dietas)<>2
    OR (SELECT count(*) FROM vec_personal.evento_rectificacion_dietas)<>5
    OR (SELECT count(*) FROM vec_personal.evento_rectificacion_dietas WHERE tipo='confirmada' AND recibo_asignacion_ref IS NOT NULL AND asignacion_nueva_ref IS NOT NULL AND version_nueva=2)<>1
    OR (SELECT count(*) FROM vec_personal.asignacion_dietas)<>2
    OR (SELECT count(*) FROM vec_personal.recibo_consulta_competentes_rectificacion_dietas)<>2
 THEN RAISE EXCEPTION 'historia rectificación duplicada'; END IF;
END $conteo$;
COMMIT;
