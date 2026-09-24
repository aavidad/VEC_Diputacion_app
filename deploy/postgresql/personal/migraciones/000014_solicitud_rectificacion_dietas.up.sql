\set ON_ERROR_STOP on
-- D7c. Contrato preparado para consumidores AD3 nominales. Su ausencia cierra
-- las operaciones; esta migración no concede al empleado corrección de datos.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000014:rectificacion-dietas:v1',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.asignacion_dietas') IS NULL
    OR to_regprocedure('vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NULL
    OR to_regprocedure('vec_personal.consultar_competencias_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regprocedure('vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(text,text,text,text,text,text,text,smallint)') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_registrador_frontera' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR to_regclass('vec_personal.solicitud_rectificacion_dietas') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_d7_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_propietario' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'Personal 000014: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_personal.solicitud_rectificacion_dietas (
 solicitud_ref text PRIMARY KEY CHECK(solicitud_ref~'^srd_[0-9a-f]{32}$'),
 relacion_ref text NOT NULL,
 persona_ref text NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 unidad_ref text NOT NULL,
 asignacion_ref text NOT NULL,
 version_origen bigint NOT NULL CHECK(version_origen>0),
 fecha_referencia date NOT NULL,
 campos_a_revisar jsonb NOT NULL CHECK(jsonb_typeof(campos_a_revisar)='array' AND jsonb_array_length(campos_a_revisar) BETWEEN 1 AND 4),
 detalle_solicitado text NOT NULL CHECK((detalle_solicitado='' OR length(detalle_solicitado) BETWEEN 3 AND 500) AND detalle_solicitado !~ '[[:cntrl:]]' AND detalle_solicitado !~ 'per_[A-Za-z0-9_-]{22,128}'),
 motivo_revision text NOT NULL CHECK(length(motivo_revision) BETWEEN 3 AND 500 AND motivo_revision !~ '[[:cntrl:]]' AND motivo_revision !~ 'per_[A-Za-z0-9_-]{22,128}'),
 clave_idempotencia text NOT NULL CHECK(clave_idempotencia~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 huella_semantica_sha256 text NOT NULL CHECK(huella_semantica_sha256~'^[0-9a-f]{64}$'),
 actor_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(relacion_ref,clave_idempotencia),
 FOREIGN KEY(asignacion_ref) REFERENCES vec_personal.asignacion_dietas(asignacion_ref),
 FOREIGN KEY(relacion_ref,persona_ref,unidad_ref) REFERENCES vec_personal.relacion_empleado_dietas(relacion_ref,persona_ref,unidad_ref)
);
CREATE TABLE vec_personal.evento_rectificacion_dietas (
 evento_ref text PRIMARY KEY CHECK(evento_ref~'^erd_[0-9a-f]{32}$'),
 solicitud_ref text NOT NULL REFERENCES vec_personal.solicitud_rectificacion_dietas(solicitud_ref),
 tipo text NOT NULL CHECK(tipo IN ('solicitada','consultada','rechazada','confirmada')),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref~'^rrd_[0-9a-f]{32}$'),
 decision_ref text NOT NULL,
 efecto_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ad3_ref text NOT NULL,
 actor_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 clave_idempotencia text,
 huella_semantica_sha256 text,
 recibo_asignacion_ref text REFERENCES vec_personal.recibo_asignacion_dietas(referencia),
 asignacion_nueva_ref text REFERENCES vec_personal.asignacion_dietas(asignacion_ref),
 version_nueva bigint,
 huella_correccion_sha256 text,
 CHECK((tipo IN ('rechazada','confirmada'))=(clave_idempotencia IS NOT NULL AND huella_semantica_sha256 IS NOT NULL)),
 CHECK((tipo='confirmada')=(recibo_asignacion_ref IS NOT NULL AND asignacion_nueva_ref IS NOT NULL AND version_nueva IS NOT NULL AND huella_correccion_sha256 IS NOT NULL)),
 CHECK(version_nueva IS NULL OR version_nueva>0),
 CHECK(huella_correccion_sha256 IS NULL OR huella_correccion_sha256~'^[0-9a-f]{64}$'),
 CHECK(clave_idempotencia IS NULL OR clave_idempotencia~'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 CHECK(huella_semantica_sha256 IS NULL OR huella_semantica_sha256~'^[0-9a-f]{64}$')
);
CREATE UNIQUE INDEX evento_rectificacion_solicitud_unica_idx
 ON vec_personal.evento_rectificacion_dietas(solicitud_ref) WHERE tipo='solicitada';
CREATE UNIQUE INDEX evento_rectificacion_terminal_unico_idx
 ON vec_personal.evento_rectificacion_dietas(solicitud_ref)
 WHERE tipo IN ('rechazada','confirmada');
CREATE UNIQUE INDEX evento_rectificacion_rechazo_clave_idx
 ON vec_personal.evento_rectificacion_dietas(solicitud_ref,clave_idempotencia)
 WHERE tipo IN ('rechazada','confirmada');
CREATE TABLE vec_personal.auditoria_frontera_rectificacion_dietas (
 evento_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 correlacion_ref text NOT NULL CHECK(correlacion_ref='corr_no_disponible' OR correlacion_ref~'^corr_[0-9a-f]{32}$'),
 motivo text NOT NULL CHECK(motivo IN ('peticion_invalida','autenticacion_requerida','acceso_denegado','no_encontrada','metodo_no_permitido','representacion_no_admitida','conflicto','dependencia_no_disponible')),
 superficie text NOT NULL CHECK(superficie='api.personal.solicitudes_rectificacion_dietas'),
 ruta text NOT NULL CHECK(ruta IN ('/api/vec/personal/solicitudes-rectificacion-dietas','/api/vec/personal/solicitudes-rectificacion-dietas/competentes')),
 accion text NOT NULL CHECK(accion IN ('solicitar','consultar','consultar_competentes','confirmar','rechazar','metodo_no_admitido')),
 actor_ref text CHECK(actor_ref IS NULL OR (length(actor_ref) BETWEEN 1 AND 512 AND actor_ref~'^[A-Za-z0-9:_-]+$')),
 recurso_ref text CHECK(recurso_ref IS NULL OR recurso_ref~'^(rel_[A-Za-z0-9_-]{22,128}|srd_[0-9a-f]{32})$'),
 estado_http integer NOT NULL CHECK(estado_http IN (400,401,403,404,405,406,409,503)),
 registrada_en timestamptz(6) NOT NULL,
 CHECK(motivo<>'autenticacion_requerida' OR actor_ref IS NULL),
 CHECK((motivo='peticion_invalida' AND estado_http=400)
  OR (motivo='autenticacion_requerida' AND estado_http=401)
  OR (motivo='acceso_denegado' AND estado_http=403)
  OR (motivo='no_encontrada' AND estado_http=404)
  OR (motivo='metodo_no_permitido' AND estado_http=405)
  OR (motivo='representacion_no_admitida' AND estado_http=406)
  OR (motivo='conflicto' AND estado_http=409)
  OR (motivo='dependencia_no_disponible' AND estado_http=503)),
 CHECK(accion='metodo_no_admitido' OR
  ((accion='consultar_competentes')=(ruta='/api/vec/personal/solicitudes-rectificacion-dietas/competentes')))
);
CREATE TABLE vec_personal.recibo_consulta_competentes_rectificacion_dietas (
 referencia text PRIMARY KEY CHECK(referencia~'^rrd_[0-9a-f]{32}$'),
 actor_persona_ref text NOT NULL CHECK(actor_persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 decision_ref text NOT NULL, efecto_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ad3_ref text NOT NULL,
 fecha_referencia date NOT NULL,
 cardinalidad integer NOT NULL CHECK(cardinalidad BETWEEN 0 AND 50),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL
);
CREATE INDEX auditoria_frontera_rectificacion_correlacion_idx ON vec_personal.auditoria_frontera_rectificacion_dietas(correlacion_ref,evento_id);
CREATE INDEX solicitud_rectificacion_dietas_pendientes_idx ON vec_personal.solicitud_rectificacion_dietas(relacion_ref,persona_ref,unidad_ref,version_origen);
ALTER TABLE vec_personal.solicitud_rectificacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.solicitud_rectificacion_dietas FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.evento_rectificacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.evento_rectificacion_dietas FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.auditoria_frontera_rectificacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.auditoria_frontera_rectificacion_dietas FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_consulta_competentes_rectificacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_consulta_competentes_rectificacion_dietas FORCE ROW LEVEL SECURITY;
CREATE POLICY solicitud_contextual ON vec_personal.solicitud_rectificacion_dietas FOR ALL TO vec_personal_propietario
 USING(persona_ref=current_setting('vec.dietas.persona_ref',true) AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL)
 WITH CHECK(persona_ref=current_setting('vec.dietas.persona_ref',true) AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL);
CREATE POLICY evento_contextual ON vec_personal.evento_rectificacion_dietas FOR ALL TO vec_personal_propietario
 USING(EXISTS(SELECT 1 FROM vec_personal.solicitud_rectificacion_dietas s WHERE s.solicitud_ref=evento_rectificacion_dietas.solicitud_ref AND s.persona_ref=current_setting('vec.dietas.persona_ref',true)))
 WITH CHECK(EXISTS(SELECT 1 FROM vec_personal.solicitud_rectificacion_dietas s WHERE s.solicitud_ref=evento_rectificacion_dietas.solicitud_ref AND s.persona_ref=current_setting('vec.dietas.persona_ref',true)));
CREATE POLICY auditoria_frontera_rectificacion_propietario ON vec_personal.auditoria_frontera_rectificacion_dietas
 TO vec_personal_propietario USING(true) WITH CHECK(true);
CREATE POLICY recibo_competentes_actor ON vec_personal.recibo_consulta_competentes_rectificacion_dietas
 TO vec_personal_propietario
 USING(actor_persona_ref=current_setting('vec.dietas.competencias_actor_persona_ref',true))
 WITH CHECK(actor_persona_ref=current_setting('vec.dietas.competencias_actor_persona_ref',true));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.solicitud_rectificacion_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.solicitud_rectificacion_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.evento_rectificacion_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.evento_rectificacion_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.auditoria_frontera_rectificacion_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.auditoria_frontera_rectificacion_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_consulta_competentes_rectificacion_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_consulta_competentes_rectificacion_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();

CREATE FUNCTION vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(
 p_correlacion_ref text,p_motivo text,p_superficie text,p_ruta text,
 p_accion text,p_actor_ref text,p_recurso_ref text,p_estado_http integer)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members membresia
      WHERE membresia.member=session_user::regrole
        AND membresia.roleid='vec_personal_registrador_frontera'::regrole
        AND membresia.inherit_option AND NOT membresia.set_option AND NOT membresia.admin_option)
    OR EXISTS(SELECT 1 FROM pg_roles rol
      WHERE rol.oid<>session_user::regrole
        AND rol.oid<>'vec_personal_registrador_frontera'::regrole
        AND pg_has_role(session_user,rol.oid,'MEMBER'))
 THEN RAISE EXCEPTION 'registrador de rectificación inválido' USING ERRCODE='42501'; END IF;
 IF p_correlacion_ref IS NULL
    OR (p_correlacion_ref<>'corr_no_disponible' AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
    OR p_motivo IS NULL OR p_ruta IS NULL OR p_accion IS NULL OR p_estado_http IS NULL
    OR p_motivo NOT IN ('peticion_invalida','autenticacion_requerida','acceso_denegado','no_encontrada','metodo_no_permitido','representacion_no_admitida','conflicto','dependencia_no_disponible')
    OR p_superficie IS DISTINCT FROM 'api.personal.solicitudes_rectificacion_dietas'
    OR p_ruta NOT IN ('/api/vec/personal/solicitudes-rectificacion-dietas','/api/vec/personal/solicitudes-rectificacion-dietas/competentes')
    OR p_accion NOT IN ('solicitar','consultar','consultar_competentes','confirmar','rechazar','metodo_no_admitido')
    OR (p_accion<>'metodo_no_admitido' AND
       ((p_accion='consultar_competentes') IS DISTINCT FROM
        (p_ruta='/api/vec/personal/solicitudes-rectificacion-dietas/competentes')))
    OR NOT ((p_motivo='peticion_invalida' AND p_estado_http=400)
       OR (p_motivo='autenticacion_requerida' AND p_estado_http=401)
       OR (p_motivo='acceso_denegado' AND p_estado_http=403)
       OR (p_motivo='no_encontrada' AND p_estado_http=404)
       OR (p_motivo='metodo_no_permitido' AND p_estado_http=405)
       OR (p_motivo='representacion_no_admitida' AND p_estado_http=406)
       OR (p_motivo='conflicto' AND p_estado_http=409)
       OR (p_motivo='dependencia_no_disponible' AND p_estado_http=503))
    OR (p_motivo='autenticacion_requerida' AND p_actor_ref IS NOT NULL)
    OR (p_actor_ref IS NOT NULL AND (length(p_actor_ref) NOT BETWEEN 1 AND 512 OR p_actor_ref !~ '^[A-Za-z0-9:_-]+$'))
    OR (p_recurso_ref IS NOT NULL AND p_recurso_ref !~ '^(rel_[A-Za-z0-9_-]{22,128}|srd_[0-9a-f]{32})$')
 THEN RAISE EXCEPTION 'auditoría de rectificación inválida' USING ERRCODE='22023'; END IF;
 INSERT INTO vec_personal.auditoria_frontera_rectificacion_dietas(
  correlacion_ref,motivo,superficie,ruta,accion,actor_ref,recurso_ref,estado_http,registrada_en)
 VALUES(p_correlacion_ref,p_motivo,p_superficie,p_ruta,p_accion,p_actor_ref,
  p_recurso_ref,p_estado_http,clock_timestamp());
 RETURN true;
END $f$;

-- El consumidor AD3 único comprueba las tres acciones nominales. Confirmar
-- consume además la concesión de corrección D7 en la misma transacción.
CREATE FUNCTION vec_personal.ejecutar_rectificacion_dietas_interna_v1(
 p_modo text,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,
 p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 p_material_correccion text DEFAULT NULL,p_capacidad_correccion bytea DEFAULT NULL,
 p_decision_correccion bytea DEFAULT NULL,p_motivo_correccion bytea DEFAULT NULL,
 p_contexto_correccion bytea DEFAULT NULL,p_persona_version_correccion numeric DEFAULT NULL,
 p_perfil_version_correccion numeric DEFAULT NULL,p_payload_correccion bytea DEFAULT NULL,
 p_sobre_correccion bytea DEFAULT NULL,p_evidencia_correccion bytea DEFAULT NULL,
 p_raiz_correccion bytea DEFAULT NULL)
RETURNS TABLE(solicitud_ref text,recibo_ref text,estado text,registrada_en timestamptz,
 asignacion_ref text,version_origen bigint,decision_ref text,efecto_ref text,
 consumo_huella_sha256 text,auditoria_ad3_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; i jsonb; c jsonb; d jsonb; x jsonb; campos jsonb; mc jsonb;
 z record; cor record; r vec_personal.relacion_empleado_dietas%ROWTYPE;
 a vec_personal.asignacion_dietas%ROWTYPE;
 s vec_personal.solicitud_rectificacion_dietas%ROWTYPE;
 ev vec_personal.evento_rectificacion_dietas%ROWTYPE;
 actor_persona text; actor_empleado text; sujeto_persona text; sujeto_empleado text;
 relacion text; unidad text; clave text; material_sha text; recurso_sha text;
 amb text; atr text; accion text; audiencia text; finalidad text;
 fecha date; hoy date; ahora timestamptz(6); nueva_ref text; nuevo_recibo text;
 cor_recibo_ref text; cor_asignacion_ref text; cor_version bigint;
 validador regprocedure;
 campos_salida jsonb:='["asignacion_ref","auditoria_ad3_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado","recibo_ref","registrada_en","solicitud_ref","version_origen"]'::jsonb;
BEGIN
 IF p_modo NOT IN ('solicitar','consultar','rechazar','confirmar')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_d7_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL
    OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL
    OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
    OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'rectificación Personal rechazada' USING ERRCODE='42501'; END IF;
 IF p_modo='confirmar' AND (p_material_correccion IS NULL OR p_capacidad_correccion IS NULL
    OR p_decision_correccion IS NULL OR p_motivo_correccion IS NULL
    OR p_contexto_correccion IS NULL OR p_persona_version_correccion IS NULL
    OR p_perfil_version_correccion IS NULL OR p_payload_correccion IS NULL
    OR p_sobre_correccion IS NULL OR p_evidencia_correccion IS NULL
    OR p_raiz_correccion IS NULL)
 THEN RAISE EXCEPTION 'confirmación requiere corrección D7 completa' USING ERRCODE='55000';
 END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material de rectificación inválido' USING ERRCODE='22023';
 END;
 i:=m->'identidad'; campos:=m->'campos_a_revisar';
 IF jsonb_typeof(m)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
       ARRAY['asignacion_ref','campos_a_revisar','clave_idempotencia',
       'detalle_solicitado','empleado_ref','esquema','fecha_referencia',
       'identidad','motivo_revision','operacion','persona_ref','relacion_ref',
       'solicitud_ref','unidad_ref','version_esperada']
    OR m->>'esquema'<>'vec.personal.rectificacion-dietas.v1'
    OR m->>'operacion' IS DISTINCT FROM p_modo
    OR m->>'relacion_ref' !~ '^rel_[A-Za-z0-9_-]{22,128}$'
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR m->>'fecha_referencia' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR jsonb_typeof(i)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(i) ORDER BY 1) IS DISTINCT FROM
       ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref',
       'cuenta_version','empleado_ref','perfil_ref','perfil_version',
       'persona_ref','persona_version']
 THEN RAISE EXCEPTION 'material de rectificación no canónico' USING ERRCODE='22023'; END IF;
 relacion:=m->>'relacion_ref'; sujeto_persona:=m->>'persona_ref';
 sujeto_empleado:=m->>'empleado_ref'; unidad:=m->>'unidad_ref';
 IF unidad IS NULL OR length(unidad) NOT BETWEEN 1 AND 256
 THEN RAISE EXCEPTION 'unidad inválida' USING ERRCODE='22023'; END IF;
 fecha:=(m->>'fecha_referencia')::date;
 hoy:=(clock_timestamp() AT TIME ZONE 'UTC')::date;
 IF p_modo='consultar' AND fecha IS DISTINCT FROM hoy
 THEN RAISE EXCEPTION 'consulta histórica no disponible' USING ERRCODE='P7201'; END IF;
 SELECT e.valor->>'referencia' INTO actor_empleado
 FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
 WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
   AND e.valor->>'estado'='activo';
 actor_persona:=x->>'persona_ref';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 amb:='{"empleado_ref":'||to_jsonb(sujeto_empleado)::text||
      ',"persona_ref":'||to_jsonb(sujeto_persona)::text||
      ',"relacion_ref":'||to_jsonb(relacion)::text||
      ',"unidad_ref":'||to_jsonb(unidad)::text||'}';
 atr:='{"fecha_referencia":'||to_jsonb(fecha::text)::text||
      ',"material_sha256":'||to_jsonb(material_sha)::text||
      ',"operacion":'||to_jsonb(p_modo)::text||'}';
 recurso_sha:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 accion:=CASE p_modo WHEN 'solicitar' THEN 'personal.asignacion_dietas.rectificacion.solicitar'
   WHEN 'consultar' THEN 'personal.asignacion_dietas.rectificacion.propia.consultar'
   ELSE 'personal.asignacion_dietas.rectificacion.resolver' END;
 audiencia:=CASE p_modo WHEN 'solicitar' THEN 'vec_personal.asignacion_dietas.rectificacion.solicitar.v1'
   WHEN 'consultar' THEN 'vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1'
   ELSE 'vec_personal.asignacion_dietas.rectificacion.resolver.v1' END;
 finalidad:=CASE p_modo WHEN 'solicitar' THEN 'solicitar_rectificacion_dietas'
   WHEN 'consultar' THEN 'consultar_rectificacion_dietas_propia'
   ELSE 'resolver_rectificacion_dietas' END;
 IF x->>'esquema'<>'vec.contexto-actor.vinculado.v2'
    OR actor_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR actor_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR (SELECT count(*) FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
        WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
          AND e.valor->>'estado'='activo')<>1
    OR i->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR i->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR i->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR i->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR i->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR i->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR i->>'persona_ref' IS DISTINCT FROM actor_persona
    OR i->>'empleado_ref' IS DISTINCT FROM actor_empleado
    OR i->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR i->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR coalesce((x->>'persona_version')::numeric,0)<>p_persona_version
    OR coalesce((x->>'perfil_version')::numeric,0)<>p_perfil_version
    OR (p_modo IN ('solicitar','consultar') AND
        (sujeto_persona IS DISTINCT FROM actor_persona OR sujeto_empleado IS DISTINCT FROM actor_empleado))
    OR (p_modo='rechazar' AND sujeto_persona IS NOT DISTINCT FROM actor_persona)
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'operacion' IS DISTINCT FROM accion
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'accion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'rectificacion_asignacion_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM relacion
    OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM campos_salida
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
 THEN RAISE EXCEPTION 'capacidad de rectificación no corresponde' USING ERRCODE='42501'; END IF;
 IF p_modo='consultar' THEN
  IF m->'solicitud_ref' IS DISTINCT FROM '""'::jsonb
     OR m->'asignacion_ref' IS DISTINCT FROM '""'::jsonb
     OR m->'version_esperada' IS DISTINCT FROM '0'::jsonb
     OR m->'clave_idempotencia' IS DISTINCT FROM '""'::jsonb
     OR m->'campos_a_revisar' IS DISTINCT FROM '[]'::jsonb
     OR m->'detalle_solicitado' IS DISTINCT FROM '""'::jsonb
     OR m->'motivo_revision' IS DISTINCT FROM '""'::jsonb
  THEN RAISE EXCEPTION 'consulta no canónica' USING ERRCODE='22023'; END IF;
 ELSIF p_modo='solicitar' THEN
  IF m->'solicitud_ref' IS DISTINCT FROM '""'::jsonb
     OR m->>'asignacion_ref' !~ '^ads_[A-Za-z0-9_-]{22,128}$'
     OR coalesce((m->>'version_esperada')::bigint,0)<1
     OR m->>'clave_idempotencia' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
     OR jsonb_typeof(campos)<>'array' OR jsonb_array_length(campos) NOT BETWEEN 1 AND 4
     OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(campos) v(campo)
               WHERE v.campo NOT IN ('centro_ref','unidad_ref','administrativo_persona_ref','responsable_persona_ref'))
     OR (SELECT count(DISTINCT v.campo) FROM jsonb_array_elements_text(campos) v(campo))<>jsonb_array_length(campos)
     OR (SELECT jsonb_agg(v.campo ORDER BY v.campo) FROM jsonb_array_elements_text(campos) v(campo)) IS DISTINCT FROM campos
     OR jsonb_typeof(m->'detalle_solicitado')<>'string'
     OR ((m->>'detalle_solicitado'<>'') AND length(coalesce(m->>'detalle_solicitado','')) NOT BETWEEN 3 AND 500)
     OR m->>'detalle_solicitado' ~ '[[:cntrl:]]|per_[A-Za-z0-9_-]{22,128}'
     OR length(coalesce(m->>'motivo_revision','')) NOT BETWEEN 3 AND 500
     OR m->>'motivo_revision' ~ '[[:cntrl:]]|per_[A-Za-z0-9_-]{22,128}'
  THEN RAISE EXCEPTION 'solicitud no canónica' USING ERRCODE='22023'; END IF;
 ELSE
  IF m->>'solicitud_ref' !~ '^srd_[0-9a-f]{32}$'
     OR (p_modo='rechazar' AND m->'asignacion_ref' IS DISTINCT FROM '""'::jsonb)
     OR (p_modo='rechazar' AND m->'version_esperada' IS DISTINCT FROM '0'::jsonb)
     OR (p_modo='confirmar' AND m->>'asignacion_ref' !~ '^ads_[A-Za-z0-9_-]{22,128}$')
     OR (p_modo='confirmar' AND coalesce((m->>'version_esperada')::bigint,0)<1)
     OR m->>'clave_idempotencia' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
     OR m->'campos_a_revisar' IS DISTINCT FROM '[]'::jsonb
     OR m->'detalle_solicitado' IS DISTINCT FROM '""'::jsonb
     OR length(coalesce(m->>'motivo_revision','')) NOT BETWEEN 3 AND 500
     OR m->>'motivo_revision' ~ '[[:cntrl:]]|per_[A-Za-z0-9_-]{22,128}'
  THEN RAISE EXCEPTION 'resolución no canónica' USING ERRCODE='22023'; END IF;
 END IF;
 PERFORM set_config('vec.dietas.persona_ref',sujeto_persona,true);
 SELECT * INTO STRICT z FROM vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF z.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'rectificación requiere consumo fresco' USING ERRCODE='P0573';
 END IF;
 SELECT * INTO r FROM vec_personal.relacion_empleado_dietas v
 WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
   AND v.empleado_ref=sujeto_empleado AND v.unidad_ref=unidad
   AND v.estado='activa' AND v.desde<=hoy AND (v.hasta IS NULL OR hoy<v.hasta)
 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'relación Personal no vigente' USING ERRCODE='P7201'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:rectificacion-dietas:'||relacion,0));
 IF p_modo='consultar' THEN
  FOR s IN SELECT * FROM vec_personal.solicitud_rectificacion_dietas v
     WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
       AND v.unidad_ref=unidad ORDER BY v.registrada_en DESC LIMIT 1
  LOOP
   SELECT * INTO ev FROM vec_personal.evento_rectificacion_dietas v
   WHERE v.solicitud_ref=s.solicitud_ref AND v.tipo IN ('rechazada','confirmada');
   ahora:=clock_timestamp();
   nuevo_recibo:='rrd_'||substr(encode(sha256(convert_to(s.solicitud_ref||'|'||z.consumo_huella_sha256||'|'||ahora::text,'UTF8')),'hex'),1,32);
   INSERT INTO vec_personal.evento_rectificacion_dietas VALUES(
    'erd_'||substr(encode(sha256(convert_to(nuevo_recibo,'UTF8')),'hex'),1,32),
    s.solicitud_ref,'consultada',nuevo_recibo,z.decision_ref,z.efecto_ref,
    z.consumo_huella_sha256,z.auditoria_ref,i->>'actor_ref',ahora,NULL,NULL,NULL,NULL,NULL,NULL);
   RETURN QUERY SELECT s.solicitud_ref,nuevo_recibo,
    CASE ev.tipo WHEN 'rechazada' THEN 'rechazada' WHEN 'confirmada' THEN 'confirmada' ELSE 'pendiente' END::text,
    ahora,s.asignacion_ref,s.version_origen,
    z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,z.auditoria_ref;
  END LOOP;
  RETURN;
 END IF;
 IF p_modo='solicitar' THEN
  clave:=m->>'clave_idempotencia';
  SELECT * INTO s FROM vec_personal.solicitud_rectificacion_dietas v
  WHERE v.relacion_ref=relacion AND v.clave_idempotencia=clave FOR SHARE;
  IF FOUND THEN
   IF s.huella_semantica_sha256 IS DISTINCT FROM material_sha
      OR s.persona_ref IS DISTINCT FROM sujeto_persona
   THEN RAISE EXCEPTION 'conflicto de idempotencia' USING ERRCODE='P7204'; END IF;
   SELECT * INTO ev FROM vec_personal.evento_rectificacion_dietas v
   WHERE v.solicitud_ref=s.solicitud_ref AND v.tipo='solicitada';
   RETURN QUERY SELECT s.solicitud_ref,ev.recibo_ref,'replay_confirmado'::text,
    ev.registrada_en,s.asignacion_ref,s.version_origen,
    z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,z.auditoria_ref;
   RETURN;
  END IF;
  IF fecha IS DISTINCT FROM hoy
  THEN RAISE EXCEPTION 'solicitud histórica no disponible' USING ERRCODE='P7201'; END IF;
  SELECT * INTO a FROM vec_personal.asignacion_dietas v
  WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
    AND v.unidad_ref=unidad AND v.vigente_desde<=hoy
  ORDER BY v.version DESC LIMIT 1 FOR SHARE;
  IF NOT FOUND OR a.asignacion_ref IS DISTINCT FROM m->>'asignacion_ref'
     OR a.version IS DISTINCT FROM (m->>'version_esperada')::bigint
  THEN RAISE EXCEPTION 'asignación Personal no vigente' USING ERRCODE='P7203'; END IF;
  IF EXISTS(SELECT 1 FROM vec_personal.solicitud_rectificacion_dietas v
   WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
     AND v.unidad_ref=unidad AND NOT EXISTS(
      SELECT 1 FROM vec_personal.evento_rectificacion_dietas e
      WHERE e.solicitud_ref=v.solicitud_ref AND e.tipo IN ('rechazada','confirmada')))
  THEN RAISE EXCEPTION 'ya existe solicitud pendiente' USING ERRCODE='P7203'; END IF;
  ahora:=clock_timestamp();
  nueva_ref:='srd_'||substr(encode(sha256(convert_to(relacion||'|'||clave||'|'||material_sha,'UTF8')),'hex'),1,32);
  nuevo_recibo:='rrd_'||substr(encode(sha256(convert_to(nueva_ref||'|'||z.consumo_huella_sha256,'UTF8')),'hex'),1,32);
  INSERT INTO vec_personal.solicitud_rectificacion_dietas VALUES(
   nueva_ref,relacion,sujeto_persona,unidad,a.asignacion_ref,a.version,fecha,
   campos,m->>'detalle_solicitado',m->>'motivo_revision',clave,material_sha,
   i->>'actor_ref',ahora);
  INSERT INTO vec_personal.evento_rectificacion_dietas VALUES(
   'erd_'||substr(encode(sha256(convert_to(nuevo_recibo,'UTF8')),'hex'),1,32),
   nueva_ref,'solicitada',nuevo_recibo,z.decision_ref,z.efecto_ref,
   z.consumo_huella_sha256,z.auditoria_ref,i->>'actor_ref',ahora,NULL,NULL,NULL,NULL,NULL,NULL);
  RETURN QUERY SELECT nueva_ref,nuevo_recibo,'pendiente'::text,ahora,
   a.asignacion_ref,a.version,z.decision_ref,z.efecto_ref,
   z.consumo_huella_sha256,z.auditoria_ref;
  RETURN;
 END IF;
 SELECT * INTO s FROM vec_personal.solicitud_rectificacion_dietas v
 WHERE v.solicitud_ref=m->>'solicitud_ref' AND v.relacion_ref=relacion
   AND v.persona_ref=sujeto_persona AND v.unidad_ref=unidad FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'solicitud no disponible' USING ERRCODE='P7201'; END IF;
 SELECT * INTO a FROM vec_personal.asignacion_dietas v
 WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
   AND v.unidad_ref=unidad AND v.vigente_desde<=hoy
 ORDER BY v.version DESC LIMIT 1 FOR SHARE;
 SELECT * INTO ev FROM vec_personal.evento_rectificacion_dietas v
 WHERE v.solicitud_ref=s.solicitud_ref AND v.tipo IN ('rechazada','confirmada');
 IF FOUND THEN
  IF ev.tipo IS DISTINCT FROM (CASE p_modo WHEN 'confirmar' THEN 'confirmada' ELSE 'rechazada' END)
     OR ev.actor_ref IS DISTINCT FROM i->>'actor_ref'
     OR ev.clave_idempotencia IS DISTINCT FROM m->>'clave_idempotencia'
  THEN RAISE EXCEPTION 'conflicto de idempotencia' USING ERRCODE='P7204'; END IF;
  IF ev.huella_semantica_sha256 IS DISTINCT FROM material_sha
     OR (p_modo='confirmar' AND ev.huella_correccion_sha256 IS DISTINCT FROM
      encode(sha256(convert_to(p_material_correccion,'UTF8')),'hex'))
  THEN RAISE EXCEPTION 'conflicto de idempotencia' USING ERRCODE='P7204'; END IF;
  RETURN QUERY SELECT s.solicitud_ref,ev.recibo_ref,'replay_confirmado'::text,
   ev.registrada_en,s.asignacion_ref,s.version_origen,z.decision_ref,z.efecto_ref,
   z.consumo_huella_sha256,z.auditoria_ref;
  RETURN;
 END IF;
 IF fecha IS DISTINCT FROM hoy
 THEN RAISE EXCEPTION 'resolución histórica no disponible' USING ERRCODE='P7201'; END IF;
 IF p_modo='confirmar' THEN
  validador:=to_regprocedure('vec_personal.validar_destino_asignacion_dietas_v1(text,text,text,text,text,text,date)');
  IF validador IS NULL OR NOT EXISTS(
   SELECT 1 FROM pg_proc p WHERE p.oid=validador
     AND p.proowner='vec_personal_propietario'::regrole
     AND p.prosecdef
     AND p.proconfig @> ARRAY['search_path=pg_catalog']::text[]
     AND NOT EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
      WHERE x.grantee=0 OR x.grantee='vec_personal_d7_ejecutor'::regrole)
  ) THEN RAISE EXCEPTION 'fuente gobernada de destino Personal no disponible' USING ERRCODE='55000'; END IF;
 END IF;
 IF a.asignacion_ref IS NULL OR a.administrativo_persona_ref IS DISTINCT FROM actor_persona
    OR actor_persona IS NOT DISTINCT FROM sujeto_persona
 THEN RAISE EXCEPTION 'administrativo no competente' USING ERRCODE='42501'; END IF;
 IF p_modo='confirmar' THEN
  IF s.asignacion_ref IS DISTINCT FROM m->>'asignacion_ref'
     OR s.version_origen IS DISTINCT FROM (m->>'version_esperada')::bigint
     OR a.asignacion_ref IS DISTINCT FROM s.asignacion_ref
     OR a.version IS DISTINCT FROM s.version_origen
  THEN RAISE EXCEPTION 'versión de origen no vigente' USING ERRCODE='P7203'; END IF;
  IF s.campos_a_revisar ? 'unidad_ref' THEN
   RAISE EXCEPTION 'corrección de unidad requiere autoridad de relación Personal' USING ERRCODE='55000';
  END IF;
  BEGIN mc:=p_material_correccion::jsonb;
  EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de corrección inválido' USING ERRCODE='22023'; END;
  IF jsonb_typeof(mc)<>'object'
     OR mc->>'esquema'<>'vec.personal.asignacion-dietas.v1'
     OR mc->>'operacion'<>'corregir'
     OR mc->>'relacion_ref' IS DISTINCT FROM relacion
     OR mc->>'persona_ref' IS DISTINCT FROM sujeto_persona
     OR mc->>'empleado_ref' IS DISTINCT FROM sujeto_empleado
     OR mc->>'unidad_ref' IS DISTINCT FROM unidad
     OR mc->>'fecha_referencia' IS DISTINCT FROM m->>'fecha_referencia'
     OR mc->>'version_esperada' IS DISTINCT FROM s.version_origen::text
     OR mc->>'clave_idempotencia' IS DISTINCT FROM m->>'clave_idempotencia'
     OR mc->>'procedencia_acto_ref' IS DISTINCT FROM s.solicitud_ref
     OR mc->>'motivo_revision' IS DISTINCT FROM m->>'motivo_revision'
     OR mc->'identidad' IS DISTINCT FROM i
     OR p_persona_version_correccion IS DISTINCT FROM p_persona_version
     OR p_perfil_version_correccion IS DISTINCT FROM p_perfil_version
     OR p_contexto_correccion IS DISTINCT FROM p_contexto
     OR mc->>'grupo_dieta' IS DISTINCT FROM a.grupo_dieta::text
  THEN RAISE EXCEPTION 'corrección no enlazada a solicitud' USING ERRCODE='42501'; END IF;
  IF vec_personal.validar_destino_asignacion_dietas_v1(
    relacion,sujeto_persona,unidad,mc->>'centro_ref',
    mc->>'administrativo_persona_ref',mc->>'responsable_persona_ref',fecha)
    IS DISTINCT FROM true
  THEN RAISE EXCEPTION 'destino de asignación no acreditado' USING ERRCODE='42501'; END IF;
  SELECT * INTO STRICT cor FROM vec_personal.corregir_asignacion_dietas_v1(
   p_material_correccion,p_capacidad_correccion,p_decision_correccion,
   p_motivo_correccion,p_contexto_correccion,p_persona_version_correccion,
   p_perfil_version_correccion,p_payload_correccion,p_sobre_correccion,
   p_evidencia_correccion,p_raiz_correccion);
  IF cor.estado_local IS DISTINCT FROM 'registrada'
     OR cor.version IS DISTINCT FROM a.version+1
     OR cor.relacion_ref IS DISTINCT FROM relacion
     OR cor.persona_ref IS DISTINCT FROM sujeto_persona
     OR cor.unidad_ref IS DISTINCT FROM unidad
     OR cor.grupo_dieta IS DISTINCT FROM a.grupo_dieta
     OR (cor.centro_ref IS DISTINCT FROM a.centro_ref AND NOT (s.campos_a_revisar ? 'centro_ref'))
     OR (cor.administrativo_persona_ref IS DISTINCT FROM a.administrativo_persona_ref AND NOT (s.campos_a_revisar ? 'administrativo_persona_ref'))
     OR (cor.responsable_persona_ref IS DISTINCT FROM a.responsable_persona_ref AND NOT (s.campos_a_revisar ? 'responsable_persona_ref'))
     OR (cor.centro_ref IS NOT DISTINCT FROM a.centro_ref
         AND cor.administrativo_persona_ref IS NOT DISTINCT FROM a.administrativo_persona_ref
         AND cor.responsable_persona_ref IS NOT DISTINCT FROM a.responsable_persona_ref)
  THEN RAISE EXCEPTION 'corrección no corresponde a los campos solicitados' USING ERRCODE='42501'; END IF;
  cor_recibo_ref:=cor.recibo_ref;
  cor_asignacion_ref:=cor.asignacion_ref;
  cor_version:=cor.version;
 END IF;
 ahora:=clock_timestamp();
 nuevo_recibo:='rrd_'||substr(encode(sha256(convert_to(s.solicitud_ref||'|'||z.consumo_huella_sha256,'UTF8')),'hex'),1,32);
 INSERT INTO vec_personal.evento_rectificacion_dietas VALUES(
  'erd_'||substr(encode(sha256(convert_to(nuevo_recibo,'UTF8')),'hex'),1,32),
  s.solicitud_ref,CASE p_modo WHEN 'confirmar' THEN 'confirmada' ELSE 'rechazada' END,
  nuevo_recibo,z.decision_ref,z.efecto_ref,
  z.consumo_huella_sha256,z.auditoria_ref,i->>'actor_ref',ahora,
  m->>'clave_idempotencia',material_sha,
  cor_recibo_ref,cor_asignacion_ref,cor_version,
  CASE WHEN p_modo='confirmar' THEN encode(sha256(convert_to(p_material_correccion,'UTF8')),'hex') ELSE NULL END);
 RETURN QUERY SELECT s.solicitud_ref,nuevo_recibo,
  CASE p_modo WHEN 'confirmar' THEN 'confirmada' ELSE 'rechazada' END::text,ahora,
  s.asignacion_ref,s.version_origen,z.decision_ref,z.efecto_ref,
  z.consumo_huella_sha256,z.auditoria_ref;
END $f$;

CREATE FUNCTION vec_personal.solicitar_rectificacion_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(solicitud_ref text,recibo_ref text,estado text,registrada_en timestamptz,
 asignacion_ref text,version_origen bigint,decision_ref text,efecto_ref text,
 consumo_huella_sha256 text,auditoria_ad3_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_rectificacion_dietas_interna_v1(
 'solicitar',p_material,p_capacidad,p_decision,p_motivo,p_contexto,
 p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_personal.consultar_rectificacion_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(solicitud_ref text,recibo_ref text,estado text,registrada_en timestamptz,
 asignacion_ref text,version_origen bigint,decision_ref text,efecto_ref text,
 consumo_huella_sha256 text,auditoria_ad3_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_rectificacion_dietas_interna_v1(
 'consultar',p_material,p_capacidad,p_decision,p_motivo,p_contexto,
 p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_personal.resolver_rectificacion_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(solicitud_ref text,recibo_ref text,estado text,registrada_en timestamptz,
 asignacion_ref text,version_origen bigint,decision_ref text,efecto_ref text,
 consumo_huella_sha256 text,auditoria_ad3_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_rectificacion_dietas_interna_v1(
 'rechazar',
 p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
 p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_personal.confirmar_rectificacion_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea,
 p_material_correccion text,p_capacidad_correccion bytea,p_decision_correccion bytea,
 p_motivo_correccion bytea,p_contexto_correccion bytea,
 p_persona_version_correccion numeric,p_perfil_version_correccion numeric,
 p_payload_correccion bytea,p_sobre_correccion bytea,
 p_evidencia_correccion bytea,p_raiz_correccion bytea)
RETURNS TABLE(solicitud_ref text,recibo_ref text,estado text,registrada_en timestamptz,
 asignacion_ref text,version_origen bigint,decision_ref text,efecto_ref text,
 consumo_huella_sha256 text,auditoria_ad3_ref text)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_rectificacion_dietas_interna_v1(
 'confirmar',p_material,p_capacidad,p_decision,p_motivo,p_contexto,
 p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz,
 p_material_correccion,p_capacidad_correccion,p_decision_correccion,
 p_motivo_correccion,p_contexto_correccion,p_persona_version_correccion,
 p_perfil_version_correccion,p_payload_correccion,p_sobre_correccion,
 p_evidencia_correccion,p_raiz_correccion)
$f$;

-- Lista competente acotada. El material no contiene sujeto, unidad ni
-- solicitud elegidos por el cliente: Personal deriva candidaturas del actor.
CREATE FUNCTION vec_personal.consultar_rectificaciones_competentes_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(recibo_ref text,decision_ref text,efecto_ref text,
 consumo_huella_sha256 text,auditoria_ad3_ref text,consultada_en timestamptz,
 cardinalidad integer,solicitudes json)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE
 m jsonb; i jsonb; c jsonb; d jsonb; x jsonb;
 actor_persona text; actor_empleado text; fecha date; material_sha text;
 amb text; atr text; recurso_sha text; z record; candidato record;
 actual vec_personal.asignacion_dietas%ROWTYPE;
 relacion vec_personal.relacion_empleado_dietas%ROWTYPE;
 solicitud vec_personal.solicitud_rectificacion_dietas%ROWTYPE;
 lista jsonb:='[]'::jsonb; numero integer:=0; candidatos integer:=0;
 ahora timestamptz(6); ref text;
 campos jsonb:='["administrativo_persona_ref","asignacion_actual","asignacion_ref","auditoria_ad3_ref","campos_a_revisar","cardinalidad","centro_ref","consultada_en","consumo_huella_sha256","decision_ref","detalle_solicitado","efecto_ref","empleado_ref","estado","fecha_referencia","grupo_dieta","motivo_revision","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","solicitud_ref","solicitudes","unidad_ref","version","version_origen","vigente_desde"]'::jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_d7_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL
    OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL
    OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
    OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'consulta competente rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material de consulta competente inválido' USING ERRCODE='22023';
 END;
 i:=m->'identidad';
 IF jsonb_typeof(m)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
       ARRAY['esquema','fecha_referencia','identidad']
    OR m->>'esquema'<>'vec.personal.rectificaciones-dietas.competentes.v1'
    OR m->>'fecha_referencia' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR jsonb_typeof(i)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(i) ORDER BY 1) IS DISTINCT FROM
       ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref',
        'cuenta_version','empleado_ref','perfil_ref','perfil_version',
        'persona_ref','persona_version']
 THEN RAISE EXCEPTION 'material competente no canónico' USING ERRCODE='22023'; END IF;
 fecha:=(m->>'fecha_referencia')::date;
 IF fecha IS DISTINCT FROM (clock_timestamp() AT TIME ZONE 'UTC')::date
 THEN RAISE EXCEPTION 'lista histórica no disponible' USING ERRCODE='P7201'; END IF;
 actor_persona:=x->>'persona_ref';
 SELECT e.valor->>'referencia' INTO actor_empleado
 FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
 WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
   AND e.valor->>'estado'='activo';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 amb:='{"persona_ref":'||to_jsonb(actor_persona)::text||'}';
 atr:='{"fecha_referencia":'||to_jsonb(fecha::text)::text||
      ',"material_sha256":'||to_jsonb(material_sha)::text||',"operacion":"lista"}';
 recurso_sha:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 IF x->>'esquema'<>'vec.contexto-actor.vinculado.v2'
    OR actor_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR actor_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR (SELECT count(*) FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
      WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
        AND e.valor->>'estado'='activo')<>1
    OR i->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR i->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR i->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR i->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR i->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR i->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR i->>'persona_ref' IS DISTINCT FROM actor_persona
    OR i->>'empleado_ref' IS DISTINCT FROM actor_empleado
    OR i->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR i->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR coalesce((x->>'persona_version')::numeric,0)<>p_persona_version
    OR coalesce((x->>'perfil_version')::numeric,0)<>p_perfil_version
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'personal.asignacion_dietas.rectificacion.competente.consultar'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'accion' IS DISTINCT FROM 'personal.asignacion_dietas.rectificacion.competente.consultar'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'rectificaciones_competentes_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM actor_persona
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_rectificaciones_dietas_competentes'
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
 THEN RAISE EXCEPTION 'capacidad competente no corresponde' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.dietas.competencias_actor_persona_ref',actor_persona,true);
 SELECT * INTO STRICT z FROM vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF z.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'consulta competente requiere consumo fresco' USING ERRCODE='P0573'; END IF;
 FOR candidato IN
  SELECT DISTINCT ON (a.relacion_ref) a.relacion_ref,a.persona_ref,a.unidad_ref
  FROM vec_personal.asignacion_dietas a
  WHERE a.administrativo_persona_ref=actor_persona
  ORDER BY a.relacion_ref,a.version DESC LIMIT 1001
 LOOP
  candidatos:=candidatos+1;
  IF candidatos>1000 THEN RAISE EXCEPTION 'demasiadas relaciones candidatas' USING ERRCODE='P7202'; END IF;
  PERFORM set_config('vec.dietas.persona_ref',candidato.persona_ref,true);
  SELECT * INTO actual FROM vec_personal.asignacion_dietas a
  WHERE a.relacion_ref=candidato.relacion_ref AND a.persona_ref=candidato.persona_ref
    AND a.unidad_ref=candidato.unidad_ref AND a.vigente_desde<=fecha
  ORDER BY a.version DESC LIMIT 1;
  IF NOT FOUND OR actual.administrativo_persona_ref IS DISTINCT FROM actor_persona
  THEN CONTINUE; END IF;
  SELECT * INTO relacion FROM vec_personal.relacion_empleado_dietas r
  WHERE r.relacion_ref=actual.relacion_ref AND r.persona_ref=actual.persona_ref
    AND r.unidad_ref=actual.unidad_ref AND r.estado='activa'
    AND r.desde<=fecha AND (r.hasta IS NULL OR fecha<r.hasta);
  IF NOT FOUND THEN CONTINUE; END IF;
  FOR solicitud IN SELECT * FROM vec_personal.solicitud_rectificacion_dietas s
    WHERE s.relacion_ref=actual.relacion_ref AND s.persona_ref=actual.persona_ref
      AND s.unidad_ref=actual.unidad_ref
      AND NOT EXISTS(SELECT 1 FROM vec_personal.evento_rectificacion_dietas e
       WHERE e.solicitud_ref=s.solicitud_ref AND e.tipo IN ('rechazada','confirmada'))
    ORDER BY s.registrada_en,s.solicitud_ref
  LOOP
   numero:=numero+1;
   IF numero>50 THEN RAISE EXCEPTION 'lista competente supera 50 pendientes' USING ERRCODE='P7202'; END IF;
   lista:=lista||jsonb_build_array(jsonb_build_object(
    'solicitud_ref',solicitud.solicitud_ref,'estado','pendiente',
    'persona_ref',solicitud.persona_ref,'empleado_ref',relacion.empleado_ref,
    'relacion_ref',solicitud.relacion_ref,'unidad_ref',solicitud.unidad_ref,
    'asignacion_ref',solicitud.asignacion_ref,'version_origen',solicitud.version_origen,
    'fecha_referencia',solicitud.fecha_referencia,
    'campos_a_revisar',solicitud.campos_a_revisar,
    'motivo_revision',solicitud.motivo_revision,
    'detalle_solicitado',solicitud.detalle_solicitado,
    'registrada_en',solicitud.registrada_en,
    'asignacion_actual',jsonb_build_object(
      'centro_ref',actual.centro_ref,
      'administrativo_persona_ref',actual.administrativo_persona_ref,
      'responsable_persona_ref',actual.responsable_persona_ref,
      'grupo_dieta',actual.grupo_dieta,'vigente_desde',actual.vigente_desde,
      'version',actual.version,'asignacion_ref',actual.asignacion_ref)));
  END LOOP;
 END LOOP;
 ahora:=clock_timestamp();
 ref:='rrd_'||substr(encode(sha256(convert_to(z.consumo_huella_sha256||'|'||material_sha||'|'||ahora::text,'UTF8')),'hex'),1,32);
 INSERT INTO vec_personal.recibo_consulta_competentes_rectificacion_dietas VALUES(
  ref,actor_persona,z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,
  z.auditoria_ref,fecha,numero,material_sha,ahora);
 RETURN QUERY SELECT ref,z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,
  z.auditoria_ref,ahora,numero,lista::json;
END $f$;

REVOKE ALL ON vec_personal.solicitud_rectificacion_dietas,vec_personal.evento_rectificacion_dietas,
 vec_personal.auditoria_frontera_rectificacion_dietas,
 vec_personal.recibo_consulta_competentes_rectificacion_dietas
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_personal_registrador_frontera,vec_dietas_ejecutor,vec_dietas_propietario;
REVOKE ALL ON SEQUENCE vec_personal.auditoria_frontera_rectificacion_dietas_evento_id_seq
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_personal_registrador_frontera;
REVOKE ALL ON FUNCTION vec_personal.ejecutar_rectificacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.solicitar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.consultar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.resolver_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.confirmar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.consultar_rectificaciones_competentes_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_dietas_ejecutor,vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(text,text,text,text,text,text,text,integer)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_personal_migrador,vec_personal_registrador_frontera;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_auditoria_frontera_rectificacion_dietas_v1(text,text,text,text,text,text,text,integer)
 TO vec_personal_registrador_frontera;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_d7_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.solicitar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.consultar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.resolver_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.confirmar_rectificacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.consultar_rectificaciones_competentes_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_d7_ejecutor;
COMMIT;
