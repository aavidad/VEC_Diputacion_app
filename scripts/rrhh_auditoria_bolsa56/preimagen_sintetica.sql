\set ON_ERROR_STOP on
-- Solo para el contenedor desechable. El consumidor AD3 es un doble explícito:
-- permite ejercitar la proyección B48/B56, no acredita firmas ni concesiones V3.
CREATE ROLE vec_bolsa_llamamientos_propietario NOLOGIN;
CREATE ROLE vec_bolsa_llamamientos_ejecutor NOLOGIN;
CREATE ROLE vec_bolsa_llamamientos_migrador NOLOGIN;
CREATE ROLE vec_b56_rrhh LOGIN INHERIT;
CREATE ROLE vec_b56_sin_permiso LOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO vec_b56_rrhh;
CREATE SCHEMA vec_bolsa_llamamientos AUTHORIZATION vec_bolsa_llamamientos_propietario;
CREATE SCHEMA vec_autorizacion_atestada_v3;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
GRANT CONNECT ON DATABASE postgres TO vec_b56_rrhh,vec_b56_sin_permiso;
SET ROLE vec_bolsa_llamamientos_propietario;
CREATE TABLE vec_bolsa_llamamientos.situacion_participacion (
 participacion_ref text NOT NULL, desde timestamptz NOT NULL, situacion text NOT NULL,
 fecha_disponible timestamptz, motivo text NOT NULL, actor text NOT NULL,
 recibo_ref text NOT NULL, registrada_en timestamptz NOT NULL,
 PRIMARY KEY(participacion_ref,desde), UNIQUE(participacion_ref,recibo_ref)
);
CREATE TABLE vec_bolsa_llamamientos.operacion_situacion_participacion (
 participacion_ref text NOT NULL, desde timestamptz NOT NULL, operacion text NOT NULL,
 PRIMARY KEY(participacion_ref,desde)
);
CREATE TABLE vec_bolsa_llamamientos.datos_contacto_participacion (
 participacion_ref text NOT NULL, version bigint NOT NULL, recibo_ref text NOT NULL,
 motivo text NOT NULL, actor text NOT NULL, registrada_en timestamptz NOT NULL,
 PRIMARY KEY(participacion_ref,version), UNIQUE(participacion_ref,recibo_ref)
);
CREATE TABLE vec_bolsa_llamamientos.traza_valor_participacion (
 participacion_ref text NOT NULL, recibo_ref text NOT NULL, campo text NOT NULL,
 valor_anterior text, valor_nuevo text, actor text NOT NULL, registrada_en timestamptz NOT NULL,
 PRIMARY KEY(participacion_ref,recibo_ref,campo)
);
RESET ROLE;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(efecto_ref text,huella_efecto_sha256 text,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'operacion' IS DISTINCT FROM 'vec.auditoria.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'material de prueba incoherente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT c->>'efecto_ref',c->>'huella_efecto_sha256',true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario;
