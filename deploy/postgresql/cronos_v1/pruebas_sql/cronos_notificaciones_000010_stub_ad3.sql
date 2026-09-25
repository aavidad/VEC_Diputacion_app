\set ON_ERROR_STOP on
-- SOLO para el arnés desechable de cronos_v1 000010. Fachadas AD3-58 de
-- prueba sobre el esquema que crea el stub de 000007 (y usa su tabla de
-- nonces): no verifican MAC, COSE ni gobierno; imitan el contrato de salida,
-- la guarda de sesión del ejecutor y el rechazo de replay por nonce. No es
-- migración.
DO $stub$
DECLARE fachada text; audiencia text;
BEGIN
 FOR fachada,audiencia IN VALUES
   ('registrar_y_consumir_cronos_notificacion_v3_atestada','vec_cronos_v1.notificacion_propia.registrar.v1'),
   ('consumir_cronos_notificaciones_propio_v3_atestada','vec_cronos_v1.notificaciones_propio.consultar.v1'),
   ('consumir_cronos_notificaciones_bandeja_v3_atestada','vec_cronos_v1.notificaciones_bandeja.consultar.v1'),
   ('registrar_y_consumir_cronos_atencion_notificacion_v3_atestada','vec_cronos_v1.notificacion.atender.v1') LOOP
  EXECUTE format($f$
CREATE FUNCTION vec_autorizacion_atestada_v3.%I(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $c$
DECLARE c jsonb; nuevo boolean;
BEGIN
 IF NOT pg_has_role(session_user,'vec_cronos_v1_ejecutor','MEMBER') THEN
   RAISE EXCEPTION 'stub AD3: sesión ajena' USING ERRCODE='42501';
 END IF;
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 IF c->>'audiencia_consumo' IS DISTINCT FROM %L THEN
   RAISE EXCEPTION 'stub AD3: audiencia' USING ERRCODE='42501';
 END IF;
 INSERT INTO vec_autorizacion_atestada_v3.stub_consumo VALUES(c->>'nonce',%L) ON CONFLICT DO NOTHING RETURNING true INTO nuevo;
 RETURN QUERY SELECT c->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',
   encode(sha256(convert_to(c->>'nonce','UTF8')),'hex'),'auditoria:stub:'||(c->>'nonce'),clock_timestamp(),coalesce(nuevo,false);
END $c$;$f$,fachada,audiencia,audiencia);
  EXECUTE format('REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC',fachada);
  EXECUTE format('GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_cronos_v1_propietario',fachada);
 END LOOP;
END $stub$;
