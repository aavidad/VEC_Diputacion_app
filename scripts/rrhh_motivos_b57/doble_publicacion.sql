\set ON_ERROR_STOP on
-- SOLO contenedor PG18 desechable: sustituye la verificación criptográfica de
-- AD3-102 para recorrer B57 con material sintético. AD3-102 se prueba aparte
-- con su núcleo real; este doble no acredita firma, revocación ni concesión.
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_causas_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.publicar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.publicar'
    OR c->>'efecto_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'doble AD3-102: material incoherente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:doble'::text,c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('a',64),
  'auditoria:doble'::text,clock_timestamp(),true;
END $f$;
RESET ROLE;

-- AD3-104/105 se sustituyen solo en este contenedor desechable. Cada doble
-- conserva la audiencia, operación, recurso y contexto que B57 comprueba.
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_propuesta_causas_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.proponer.v1'
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.proponer'
    OR c->>'efecto_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'doble AD3-104: material incoherente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:doble:104',c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('c',64),'auditoria:doble:104',clock_timestamp(),true;
END $f$;
RESET ROLE;

SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_propuesta_causas_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb:=convert_from(p_capacidad,'UTF8')::jsonb; d jsonb:=convert_from(p_decision,'UTF8')::jsonb;
BEGIN
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.propuesta.consultar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.propuesta.consultar'
    OR c->>'efecto_ref' !~ '^propuesta:causa:[0-9a-f]{64}$'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'doble AD3-105: material incoherente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:doble:105',c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('d',64),'auditoria:doble:105',clock_timestamp(),true;
END $f$;
RESET ROLE;

SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_catalogo_causas_bolsa_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb;
BEGIN
 c:=convert_from(p_capacidad,'UTF8')::jsonb;
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.causas_participacion.consultar.v1'
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.causas_participacion.consultar'
    OR c->>'efecto_ref' IS DISTINCT FROM 'vec.bolsa.causas_participacion'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'doble AD3-103: material incoherente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT 'decision:doble'::text,c->>'efecto_ref',c->>'huella_efecto_sha256',repeat('b',64),
  'auditoria:doble'::text,clock_timestamp(),true;
END $f$;
RESET ROLE;
