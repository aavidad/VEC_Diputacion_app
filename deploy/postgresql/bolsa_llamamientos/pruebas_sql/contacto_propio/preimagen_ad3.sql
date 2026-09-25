-- Preimagen mínima del núcleo AD3 v3 para ensayar AD3-84 y AD3-86 reales en
-- PostgreSQL 18 efímero: tabla de claves con la lista única de audiencias y
-- un doble del núcleo que conserva, línea a línea, las marcas que las
-- extensiones localizan (exclusión, revalidación y cierre de la lista de
-- formas admitidas). El doble no verifica firmas: acepta el material que
-- casa con alguna forma admitida y devuelve un consumo nuevo, salvo que la
-- capacidad lleve «repetida». La criptografía real se prueba en su esquema.
\set ON_ERROR_STOP on
DROP FUNCTION IF EXISTS vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version(
 clave_ref text PRIMARY KEY,
 audiencia_consumo text NOT NULL,
 CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo = ANY (ARRAY['vec.bolsa.mi-bolsa.v1'::text, 'vec_contratacion_temporal.despacho_correo_llamamiento.v1'::text]))
);
ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version OWNER TO vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(p_perfil_mutacion text,p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,p_contexto_actor_canonico bytea,p_persona_version numeric,p_perfil_version numeric,p_payload_vec_ad_3 bytea,p_sobre_cose_sign1 bytea,p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 -- Formas previas que Bolsa 000004-000006 exigen ver en el núcleo:
 -- bolsa.llamamiento.aceptacion_rrhh.registrar bolsa.llamamiento.renuncia_rrhh.registrar bolsa.llamamiento.siguiente.abrir
 c := convert_from(p_capacidad_canonica,'UTF8')::jsonb; d := convert_from(p_decision_canonica,'UTF8')::jsonb;
 IF p_perfil_mutacion IS NULL OR (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_participaciones_propias_bolsa'
 ) THEN RAISE EXCEPTION 'núcleo doble: perfil no admitido' USING ERRCODE='42501'; END IF;
 IF (p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               OR p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa') AND c ? 'caducada' THEN
  RAISE EXCEPTION 'núcleo doble: revalidación fallida' USING ERRCODE='42501';
 END IF;
 IF NOT (
           (p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_participaciones_propias_bolsa' AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.participaciones_propias.consultar')
       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1' THEN
  RAISE EXCEPTION 'núcleo doble: material rechazado' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT 'decision:'||md5(p_capacidad_canonica||p_decision_canonica), c->>'efecto_ref', c->>'huella_efecto_sha256', repeat('b',64),
   'aud_v3_'||md5(p_capacidad_canonica), clock_timestamp(), NOT (c ? 'repetida');
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_llamamientos_propietario;
