\set ON_ERROR_STOP on
-- Bolsa 000030: el replay de una solicitud o respuesta del portal exige una
-- decisión viva y nueva, como B2. Se ejecuta como superusuario sobre los
-- datos de revision/datos.sql y termina en ROLLBACK. TEST-ONLY: sustituye
-- dentro de la transacción el consumidor AD3 del portal por un doble que
-- devuelve «consumo no nuevo» si la capacidad lleva "repetida": true.
BEGIN;
SET LOCAL timezone = 'UTC';
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
 RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
 LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:'||gen_random_uuid(), convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref', repeat('a',64), repeat('b',64), 'auditoria:doble', clock_timestamp(),
        coalesce((convert_from(p_capacidad,'UTF8')::jsonb->>'repetida')::boolean, false) IS NOT TRUE
$f$;
SET LOCAL session_replication_role = replica;
INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(participacion_ref, candidato_ref, acta_ref, instantanea_ref, version_instantanea, registrada_en)
VALUES ('participacion:rev:1', 'can_revision_portal_000000000001', 'acta:rev', 'instantanea:rev', 1, now() - interval '1 day');
SET LOCAL session_replication_role = origin;
CREATE FUNCTION pg_temp.material(accion text, repetida boolean, OUT cap bytea, OUT decision bytea, OUT ctx bytea) LANGUAGE sql AS $f$
 SELECT convert_to(jsonb_build_object('efecto_ref','mi-bolsa:can_revision_portal_000000000001','operacion',accion,'repetida',repetida)::text,'UTF8'),
        convert_to(jsonb_build_object('recurso_ref','mi-bolsa:can_revision_portal_000000000001','accion',accion)::text,'UTF8'),
        convert_to('{"vinculos":[{"tipo":"candidato","estado":"activo","referencia":"can_revision_portal_000000000001"}]}','UTF8')
$f$;
CREATE FUNCTION pg_temp.solicitar(repetida boolean) RETURNS TABLE(reutilizada boolean, solicitud_ref text, recibo_ref text, registrada_en timestamptz) LANGUAGE sql AS $f$
 SELECT s.* FROM pg_temp.material('bolsa.participaciones_propias.solicitar_reactivacion', repetida) m,
  vec_bolsa_llamamientos.solicitar_portal_candidato_v1('solicitud-portal:'||repeat('7',64), 'recibo:solicitud-portal:'||repeat('7',64),
   'can_revision_portal_000000000001', 'bolsa:rev', 'reactivacion', NULL, NULL, ARRAY['disponible'], 'vec.bolsa.reglas:1:b29.portal_candidato',
   'clave-revision-portal-1', '2026-09-25T10:00:00Z', m.cap, m.decision, NULL, m.ctx, 1, 1, NULL, NULL, NULL, NULL) s
$f$;
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $prueba$ DECLARE r record; BEGIN
 SELECT * INTO STRICT r FROM pg_temp.solicitar(false);
 IF r.reutilizada THEN RAISE EXCEPTION 'B30R: primera solicitud reutilizada'; END IF;
 -- Replay con una decisión ya consumida: 42501, sin devolver el recibo.
 BEGIN
  PERFORM pg_temp.solicitar(true);
  RAISE EXCEPTION 'B30R: replay sin decisión viva aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- Replay con una decisión nueva: el mismo recibo.
 SELECT * INTO STRICT r FROM pg_temp.solicitar(false);
 IF NOT r.reutilizada OR r.recibo_ref <> 'recibo:solicitud-portal:'||repeat('7',64) THEN RAISE EXCEPTION 'B30R: replay con decisión nueva %', r; END IF;
END $prueba$;
RESET ROLE;
SELECT 'OK B30 replay con decisión viva' AS resultado;
ROLLBACK;
