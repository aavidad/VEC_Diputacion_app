\set ON_ERROR_STOP on
-- Dobles de forma para B59 (Usuarios 000008) en la prueba de superficies: el
-- consumo V3 de la lectura de avisos y la fachada de ContextoActor que da la
-- persona de una referencia de candidato. No acreditan COSE ni identidad.
CREATE ROLE vec_contexto_actor_v1_propietario NOLOGIN NOBYPASSRLS;
CREATE SCHEMA vec_contexto_actor_v1 AUTHORIZATION vec_contexto_actor_v1_propietario;
SET ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(p_candidato_ref text)
RETURNS text LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT CASE WHEN p_candidato_ref='can_ABCDEFGHIJKLMNOPQRSTUV' THEN 'per_ABCDEFGHIJKLMNOPQRSTUV' END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_usuarios_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(text) TO vec_usuarios_propietario;
RESET ROLE;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF NOT pg_has_role(session_user,'vec_usuarios_ejecutor_interno','MEMBER')
 THEN RAISE EXCEPTION 'avisos: sesión denegada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3(decision_ref,decision_canonica) VALUES(d->>'decision_ref',p_decision);
 RETURN QUERY SELECT d->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',encode(sha256(p_decision),'hex'),
  'aud_'||replace(gen_random_uuid()::text,'-',''),clock_timestamp(),true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_usuarios_propietario;
RESET ROLE;
-- Llamada de RRHH con material y huella como en Go (canonico.RecursoCorreoAvisos).
-- p_esquema elige dónde está la fachada: vec_usuarios antes de 000010 y
-- vec_usuarios_correos_avisos después.
CREATE FUNCTION public.probar_avisos(p_esquema text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE m text; h text; c bytea; d bytea; r jsonb;
BEGIN
 m:='{"esquema":"vec.usuarios.correo-avisos-llamamiento.v1","superficie":"interna_corporativa","finalidad_ref":"gestion_llamamientos_bolsa","bolsa_ref":"bolsa:prueba:superficie","unidad_ref":"unidad:rrhh","ambito_ref":"ambito:bolsa","llamamiento_ref":"llamamiento:'||repeat('ab',32)||'","candidato_ref":"can_ABCDEFGHIJKLMNOPQRSTUV"}';
 h:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:rrhh"},"atributos":{"material_sha256":"'||encode(sha256(convert_to(m,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 c:=convert_to(jsonb_build_object('audiencia_consumo','vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1','operacion','llamamiento.emitir.v1',
  'efecto_ref','bolsa:prueba:superficie','huella_efecto_sha256',h)::text,'UTF8');
 d:=convert_to(jsonb_build_object('decision_ref','dec_'||replace(gen_random_uuid()::text,'-',''),'accion','llamamiento.emitir.v1','modulo_id','bolsa',
  'tipo_recurso','bolsa_constituida','finalidad','gestion_llamamientos_bolsa','recurso_ref','bolsa:prueba:superficie','concedida',true,
  'contexto_recurso_huella_sha256',h,'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'))::text,'UTF8');
 EXECUTE format('SELECT %I.correo_activo_avisos_llamamiento_v1($1,$2,$3,$4,$4,1,1,$4,$4,$4,$4)',p_esquema) INTO r USING m,c,d,'\x00'::bytea;
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION public.probar_avisos(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.probar_avisos(text) TO vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
