-- Dobles de los consumidores AD3 de la política de ofertas (B47 y B51) para
-- ensayar Bolsa 000058: solo firmas y retorno. La criptografía real se ensaya
-- en su propio esquema.
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
DO $f$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['registrar_y_consumir_politica_ofertas_bolsa_v3_atestada','consumir_consulta_politica_ofertas_bolsa_v3_atestada'] LOOP
  EXECUTE format($s$CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.%I(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
   RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
   LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $c$
   SELECT 'decision:'||gen_random_uuid(),convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
          repeat('a',64),repeat('b',64),'auditoria:'||gen_random_uuid(),clock_timestamp(),true
   $c$ $s$, n);
  EXECUTE format('ALTER FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario', n);
  EXECUTE format('REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC', n);
  EXECUTE format('GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_propietario', n);
 END LOOP;
END $f$;
