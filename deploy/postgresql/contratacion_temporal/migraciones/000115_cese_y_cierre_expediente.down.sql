\set ON_ERROR_STOP on
-- CT115 DOWN: solo sin historia. Si algún expediente tiene una versión de
-- cese o de cierre, la reversión se niega: la historia es de solo adición.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000115',0));
LOCK TABLE vec_contratacion_temporal.expediente_version_integral IN SHARE ROW EXCLUSIVE MODE;

DO $pre$
DECLARE v_origen text;
BEGIN
    IF current_user<>'vec_contratacion_temporal_propietario'
       OR to_regclass('vec_contratacion_temporal.cese_nombramiento_v1') IS NULL
       OR to_regclass('vec_contratacion_temporal.cierre_expediente_v1') IS NULL THEN
        RAISE EXCEPTION 'CT115 DOWN: estado incompatible' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_version_integral
               WHERE origen_version IN ('cese_nombramiento_ct115','cierre_expediente_ct115'))
       OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.outbox_expediente_integral
                  WHERE tipo_evento IN ('ct.cese.v1','ct.cierre_expediente.v1')) THEN
        RAISE EXCEPTION 'CT115 DOWN: no admitido con historia de cese o cierre' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    IF strpos(v_origen,', ''cese_nombramiento_ct115''::text, ''cierre_expediente_ct115''::text')=0 THEN
        RAISE EXCEPTION 'CT115 DOWN: origen de versión no localizado' USING ERRCODE='55000';
    END IF;
END
$pre$;

-- Restaura exactamente la lectura de CT113 (solo incorporaciones).
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(p_desde_en timestamp with time zone, p_desde_ref text, p_limite integer)
 RETURNS TABLE(evento_ref text, evento jsonb, huella_sha256 text, origen_ref text, origen_creada_en timestamp with time zone)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET "TimeZone" TO 'UTC'
AS $function$
BEGIN
    IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
       OR (p_desde_en IS NULL) <> (p_desde_ref IS NULL)
       OR (p_desde_en IS NOT NULL AND NOT pg_catalog.isfinite(p_desde_en))
       OR pg_catalog.octet_length(p_desde_ref) > 512 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'lectura de contratos para Bolsa inválida';
    END IF;
    RETURN QUERY
    WITH base AS (
        SELECT o.outbox_ref, o.creada_en,
               'evento:ct:contrato-bolsa:' || pg_catalog.encode(pg_catalog.sha256(
                   pg_catalog.convert_to('incorporacion' || pg_catalog.chr(31) || o.outbox_ref, 'UTF8')
               ), 'hex') AS ref,
               pg_catalog.jsonb_build_object(
                   'esquema', 'vec.contratacion-temporal.contrato-bolsa.v1',
                   'tipo', 'incorporacion',
                   'origen_ref', o.outbox_ref,
                   'organizacion_ref', r.organizacion_ref,
                   'expediente_ref', r.expediente_ref,
                   'llamamiento_ref', p.llamamiento_ref,
                   'inicio', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,desde}')::timestamptz),
                   'fin_previsto', vec_contratacion_temporal.instante_contrato_bolsa_v1(
                       (r.material_json #>> '{Confirmacion,PeriodoIncorporacion,hasta}')::timestamptz),
                   'modalidad_clave', e.agregado_json #>> '{analisis,modalidad_clave}',
                   'categoria_ref', e.agregado_json #>> '{analisis,categoria_ref}',
                   'causa_clave', e.agregado_json #>> '{analisis,causa_clave}',
                   'ocurrido_en', vec_contratacion_temporal.instante_contrato_bolsa_v1(r.registrada_en)
               ) AS cuerpo
          FROM vec_contratacion_temporal.incorporacion_outbox_v2 o
          JOIN vec_contratacion_temporal.incorporacion_registro_v2 r
            ON r.recibo_ref = o.recibo_ref AND r.outbox_ref = o.outbox_ref
          JOIN vec_contratacion_temporal.propuesta_formalizacion p
            ON p.organizacion_ref = r.organizacion_ref AND p.expediente_ref = r.expediente_ref
          JOIN vec_contratacion_temporal.expediente_version_integral e
            ON e.expediente_ref = r.expediente_ref AND e.version = r.version_expediente
         WHERE p_desde_en IS NULL OR (o.creada_en, o.outbox_ref) > (p_desde_en, p_desde_ref)
         ORDER BY o.creada_en, o.outbox_ref
         LIMIT p_limite
    )
    SELECT b.ref, b.cuerpo || pg_catalog.jsonb_build_object('evento_ref', b.ref),
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               (b.cuerpo || pg_catalog.jsonb_build_object('evento_ref', b.ref))::text, 'UTF8')), 'hex'),
           b.outbox_ref, b.creada_en
      FROM base b
     ORDER BY b.creada_en, b.outbox_ref;
END
$function$;
DO $ct113$
BEGIN
    IF (SELECT md5(prosrc) FROM pg_proc WHERE oid='vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz,text,integer)'::regprocedure)
       IS DISTINCT FROM 'ecb89f46ebaf84a53bf52ee7ea4e2b93' THEN
        RAISE EXCEPTION 'CT115 DOWN: no se restauró exactamente CT113' USING ERRCODE='55000';
    END IF;
END
$ct113$;
COMMENT ON FUNCTION vec_contratacion_temporal.leer_contratos_bolsa_v1(timestamptz, text, integer) IS
    'CT113: publica a Bolsa, desde el outbox CT75, las incorporaciones de expedientes cubiertos por llamamiento; solo referencias opacas, fechas y claves.';

DROP FUNCTION vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(text,text);
DROP FUNCTION vec_contratacion_temporal.confirmar_cierre_expediente_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_cierre_expediente_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.confirmar_cese_nombramiento_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.preparar_cese_nombramiento_v1(jsonb);
DROP FUNCTION vec_contratacion_temporal.resultado_cierre_ct115(vec_contratacion_temporal.cierre_expediente_v1,text);
DROP FUNCTION vec_contratacion_temporal.resultado_cese_ct115(vec_contratacion_temporal.cese_nombramiento_v1);
DROP FUNCTION vec_contratacion_temporal.incorporacion_expediente_ct115(text,text);
DROP FUNCTION vec_contratacion_temporal.validar_confirmacion_ct115(jsonb,text,text,text,text,text,bytea,bytea,numeric,numeric);
DROP FUNCTION vec_contratacion_temporal.validar_preparacion_ct115(jsonb,text,text,text);
DROP FUNCTION vec_contratacion_temporal.validar_material_cierre_ct115(jsonb);
DROP FUNCTION vec_contratacion_temporal.validar_material_cese_ct115(jsonb);
DROP FUNCTION vec_contratacion_temporal.exigir_sesion_ct115(boolean);
DROP TABLE vec_contratacion_temporal.cierre_expediente_v1;
DROP TABLE vec_contratacion_temporal.cese_nombramiento_v1;
DROP FUNCTION vec_contratacion_temporal.referencia_valida_ct115(text);
DROP FUNCTION vec_contratacion_temporal.huella_contexto_go_ct115(jsonb,jsonb);
DROP FUNCTION vec_contratacion_temporal.mapa_go_ct115(jsonb);
DROP FUNCTION vec_contratacion_temporal.inicio_incorporacion_ct115(jsonb);
DROP FUNCTION vec_contratacion_temporal.texto_valido_ct115(text,integer,boolean);

DO $origen$
DECLARE v_origen text;
BEGIN
    SELECT pg_get_constraintdef(oid) INTO STRICT v_origen FROM pg_constraint
     WHERE conrelid='vec_contratacion_temporal.expediente_version_integral'::regclass
       AND conname='expediente_version_integral_origen_version_check';
    ALTER TABLE vec_contratacion_temporal.expediente_version_integral
        DROP CONSTRAINT expediente_version_integral_origen_version_check;
    EXECUTE 'ALTER TABLE vec_contratacion_temporal.expediente_version_integral ADD CONSTRAINT expediente_version_integral_origen_version_check '
        ||replace(v_origen,', ''cese_nombramiento_ct115''::text, ''cierre_expediente_ct115''::text','');
END
$origen$;
COMMIT;
