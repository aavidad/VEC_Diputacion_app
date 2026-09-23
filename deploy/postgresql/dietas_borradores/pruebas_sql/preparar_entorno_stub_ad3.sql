\set ON_ERROR_STOP on
-- Sólo para el arnés desechable. Personal 000007 real se instala antes de
-- Dietas; esta fachada AD3 TEST-ONLY no es migración ni consumidor instalable.
CREATE ROLE vec_prueba_dietas LOGIN INHERIT;
GRANT vec_dietas_ejecutor TO vec_prueba_dietas;
CREATE SCHEMA vec_autorizacion_atestada_v3;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_dietas_propietario;
CREATE TABLE vec_autorizacion_atestada_v3.stub_consumo (nonce text PRIMARY KEY);
GRANT INSERT,SELECT ON vec_autorizacion_atestada_v3.stub_consumo TO vec_dietas_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$
DECLARE c jsonb; d jsonb; ctx jsonb; insertada boolean;
BEGIN
 IF $1 IS NULL OR $2 IS NULL OR $3 IS NULL OR $4 IS NULL OR $5 IS NULL OR $6 IS NULL
    OR $7 IS NULL OR $8 IS NULL OR $9 IS NULL OR $10 IS NULL THEN
   RAISE EXCEPTION 'stub AD3: material incompleto' USING ERRCODE='22023';
 END IF;
 c:=convert_from($1,'UTF8')::jsonb;
 d:=convert_from($2,'UTF8')::jsonb;
 ctx:=convert_from($4,'UTF8')::jsonb;
 IF c->>'esquema' <> 'vec.autorizacion.capacidad-registro-consumo-atestado.v3'
    OR c->>'nonce' IS NULL OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref'
    OR c->>'contexto_ref' IS NULL OR ctx->>'esquema' <> 'vec.contexto-actor.vinculado.v2' THEN
   RAISE EXCEPTION 'stub AD3: decisión/capacidad/contexto incompatibles' USING ERRCODE='PD003';
 END IF;
 INSERT INTO vec_autorizacion_atestada_v3.stub_consumo VALUES(c->>'nonce') ON CONFLICT DO NOTHING RETURNING true INTO insertada;
 RETURN QUERY SELECT c->>'decision_ref',c->>'efecto_ref',c->>'huella_efecto_sha256',encode(sha256(convert_to(c->>'nonce','UTF8')),'hex'),'aad_stub_'||substr(c->>'nonce',1,16),clock_timestamp(),coalesce(insertada,false);
END $$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_propietario;
