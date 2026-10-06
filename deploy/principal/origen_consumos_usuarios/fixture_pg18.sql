\set ON_ERROR_STOP on
-- Fixture estructural sintético para probar el guion en PostgreSQL 18.4
-- desechable. La tabla, la política y el resolutor AD172 se cargan después,
-- copiados literalmente de su migración por probar_pg18.sh.
CREATE ROLE vec_autorizacion_atestada_v3_propietario NOLOGIN;
CREATE ROLE vec_usuarios_ejecutor_interno NOLOGIN;
CREATE ROLE vec_usuarios_ejecutor_externo NOLOGIN;
CREATE ROLE vec_pref508a_i_ue LOGIN;
CREATE ROLE vec_pref508a_e_ue LOGIN;
CREATE ROLE vec_adm_lector_sintetico LOGIN;
GRANT vec_usuarios_ejecutor_interno TO vec_pref508a_i_ue WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_usuarios_ejecutor_externo TO vec_pref508a_e_ue WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_pref508a_i_ue, vec_pref508a_e_ue;

SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'mutación rechazada' USING ERRCODE = '55000'; END $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'truncado rechazado' USING ERRCODE = '55000'; END $f$;

DO $f$
DECLARE literales text; BEGIN
 SELECT string_agg(quote_literal('vec_usuarios.'||f||'.'||a||'.'||s||'.v1')||','||quote_literal('vec.'||f||'.'||a), ',')
   INTO literales
   FROM (VALUES ('preferencias','consultar'),('preferencias','actualizar'),('correos','consultar'),
                ('correos','anadir'),('correos','reenviar'),('correos','verificar'),('correos','activar'),
                ('correos','retirar'),('imagen','consultar'),('imagen','actualizar')) x(f,a),
        (VALUES ('interna_corporativa'),('externa_personal')) y(s);
 EXECUTE format($t$CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version (
   audiencia_consumo text CONSTRAINT clave_capacidad_version_audiencia_consumo_check
   CHECK (audiencia_consumo = ANY (ARRAY[%s, 'vec.admin.usuarios.consultar.v1'])))$t$,
   (SELECT string_agg(quote_literal('vec_usuarios.'||f||'.'||a||'.'||s||'.v1'), ',')
      FROM (VALUES ('preferencias','consultar'),('preferencias','actualizar'),('correos','consultar'),
                ('correos','anadir'),('correos','reenviar'),('correos','verificar'),('correos','activar'),
                ('correos','retirar'),('imagen','consultar'),('imagen','actualizar')) x(f,a),
           (VALUES ('interna_corporativa'),('externa_personal')) y(s)));
 -- Sustituto del núcleo: solo importa su texto (exige el origen y nombra
 -- audiencias y operaciones), igual que lo coteja el guion.
 EXECUTE format($t$CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RETURNS void
   LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $c$
   BEGIN
     IF $1 = ANY (ARRAY[%s]) THEN NULL; END IF;
     PERFORM vec_autorizacion_atestada_v3.resolver_origen_consumo_v1('a','b','c');
   END $c$$t$, literales);
END $f$;
RESET ROLE;
