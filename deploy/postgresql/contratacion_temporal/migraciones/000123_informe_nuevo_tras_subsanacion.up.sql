\set ON_ERROR_STOP on
-- CT123: informe jurídico nuevo tras subsanar un reparo de Intervención
-- (duda 5 de RRHH: «desfavorable: vuelve a la unidad gestora; tras subsanar,
-- informe nuevo y nueva firma»). CT51 solo emite el informe en la versión 4;
-- CT123 añade la emisión durante la subsanación sin editar CT51 ni CT93:
--   * antecedente: subsanación de la unidad en incidencia, fiscalización
--     desfavorable con su retorno pendiente y la subsanación de ese retorno
--     registrada (se comprueba con antecedente_refiscalizacion_v1 de CT93);
--     un solo informe nuevo por retorno;
--   * mismo material, recibo, documento de desarrollo y consumidor AD3-9 que
--     CT51; la actuación no sale de la subsanación, cita el retorno y el
--     informe nuevo identifica en «sustituye» el informe que se fiscalizó;
--   * la nueva fiscalización de CT93 toma el informe vigente del expediente,
--     así que queda ligada al informe nuevo y lo aporta a su autorización.
-- Exigir o no el informe nuevo lo decide el catálogo (regla c19). La
-- aplicación lo publica al arrancar en politica_informe_tras_subsanacion
-- (solo adición; publicar lo mismo no añade historia) y la nueva
-- fiscalización de CT93 lo comprueba también en la base: con la política
-- vigente «exige», rechaza fiscalizar de nuevo mientras el informe nuevo de
-- ese retorno no se haya emitido. CT93 no se edita: su confirmación recibe
-- esa comprobación desde aquí, con la preimagen exacta verificada. Sin
-- ninguna política publicada rige la conducta de siempre.
-- Añade inicio_ronda_informe_nuevo_v1: la versión desde la que se vuelve a
-- firmar el informe (segunda ronda del circuito de firma).
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000123', 0));

DO $prevalidacion$
DECLARE v_def text; f regprocedure;
BEGIN
    IF pg_catalog.to_regprocedure('vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.informe_nuevo_admisible_ct123(jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(text,text)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.informe_nuevo_exigido_ct123()') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text)') IS NOT NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.politica_informe_tras_subsanacion') IS NOT NULL THEN
        RAISE EXCEPTION 'CT123 ya instalada' USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NULL THEN
        RAISE EXCEPTION 'CT123: falta CT103' USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.to_regclass('vec_contratacion_temporal.reserva_informe_juridico') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.documento_informe_juridico_desarrollo') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.terminal_informe_juridico') IS NULL
       OR pg_catalog.to_regclass('vec_contratacion_temporal.reserva_subsanacion_reparos') IS NULL THEN
        RAISE EXCEPTION 'CT123: faltan CT51 o CT92' USING ERRCODE = '55000';
    END IF;
    FOREACH f IN ARRAY ARRAY[
        pg_catalog.to_regprocedure('vec_contratacion_temporal.antecedente_refiscalizacion_v1(jsonb)'),
        pg_catalog.to_regprocedure('vec_contratacion_temporal.recibo_informe_juridico_v1(text)'),
        pg_catalog.to_regprocedure('vec_contratacion_temporal.informe_juridico_claves_exactas_v1(jsonb,text[])'),
        pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_informe_juridico_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
    ] LOOP
        IF f IS NULL OR NOT pg_catalog.has_function_privilege(current_user, f, 'EXECUTE') THEN
            RAISE EXCEPTION 'CT123: faltan CT51, CT93 o AD3-9' USING ERRCODE = '55000';
        END IF;
    END LOOP;
    SELECT pg_catalog.pg_get_constraintdef(oid) INTO v_def
      FROM pg_catalog.pg_constraint
     WHERE conrelid = 'vec_contratacion_temporal.reserva_informe_juridico'::regclass
       AND conname = 'reserva_informe_juridico_version_expediente_check'
       AND contype = 'c' AND convalidated;
    IF v_def IS DISTINCT FROM 'CHECK ((version_expediente = (4)::numeric))' THEN
        RAISE EXCEPTION 'CT123: preimagen de reserva de informe incompatible' USING ERRCODE = '55000';
    END IF;
END
$prevalidacion$;

ALTER TABLE vec_contratacion_temporal.reserva_informe_juridico
    DROP CONSTRAINT reserva_informe_juridico_version_expediente_check,
    ADD CONSTRAINT reserva_informe_juridico_version_expediente_check
        CHECK (version_expediente = 4 OR version_expediente BETWEEN 7 AND 9007199254740990);

-- El expediente admite informe nuevo: subsanación en incidencia, reparo
-- desfavorable pendiente, con informe y sin informe nuevo para ese retorno.
-- La subsanación registrada la comprueba antecedente_refiscalizacion_v1.
CREATE FUNCTION vec_contratacion_temporal.informe_nuevo_admisible_ct123(p_expediente jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF pg_catalog.jsonb_typeof(p_expediente) IS DISTINCT FROM 'object'
       OR coalesce(p_expediente ->> 'version', '') !~ '^[1-9][0-9]{0,15}$' THEN
        RETURN false;
    END IF;
    RETURN (p_expediente ->> 'version')::numeric BETWEEN 7 AND 9007199254740990
       AND p_expediente ->> 'fase_actual' IS NOT DISTINCT FROM 'subsanacion_unidad'
       AND p_expediente ->> 'estado_actual' IS NOT DISTINCT FROM 'incidencia'
       AND pg_catalog.jsonb_typeof(p_expediente -> 'asignacion') IS NOT DISTINCT FROM 'object'
       AND pg_catalog.jsonb_typeof(p_expediente -> 'informe_juridico') IS NOT DISTINCT FROM 'object'
       AND p_expediente #>> '{fiscalizacion,resultado}' IS NOT DISTINCT FROM 'desfavorable'
       AND p_expediente #>> '{fiscalizacion,retorno,estado}' IS NOT DISTINCT FROM 'pendiente'
       AND coalesce(p_expediente #>> '{fiscalizacion,retorno,retorno_ref}', '') <> ''
       AND p_expediente #>> '{informe_juridico,sustituye,retorno_ref}' IS DISTINCT FROM
           p_expediente #>> '{fiscalizacion,retorno,retorno_ref}';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(
    p_operacion jsonb
)
RETURNS TABLE (
    resultado text, expediente_json text, reserva_ref text, informe_ref text,
    documento_ref text, recibo_ref text, auditoria_ref text, evento_ref text,
    ambito_hmac text, huella_peticion_hmac text, organizacion_ref text,
    expediente_ref text, version_expediente bigint, actor_ref text,
    perfil_ref text, estado text, recibo_json jsonb
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_activo jsonb := p_operacion #> '{sellos_hmac,activo}';
    v_retenido jsonb;
    v_reserva vec_contratacion_temporal.reserva_informe_juridico%ROWTYPE;
    v_actual record;
    v_encontrada boolean := false;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'on'
       OR pg_catalog.pg_column_size(p_operacion) > 65536
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion, ARRAY['actor_ref','esquema','expediente_ref',
           'operacion','organizacion_ref','perfil_ref','referencias_candidatas',
           'sellos_hmac','version_expediente'])
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.preparar-informe-juridico.v1'
       OR p_operacion ->> 'operacion' <> 'preparar'
       OR coalesce(p_operacion ->> 'version_expediente', '') !~ '^[1-9][0-9]{0,15}$'
       OR (p_operacion ->> 'version_expediente')::numeric NOT BETWEEN 7 AND 9007199254740990
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion -> 'referencias_candidatas',
           ARRAY['auditoria_ref','documento_ref','evento_ref','informe_ref',
                 'recibo_ref','reserva_ref'])
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion -> 'sellos_hmac', ARRAY['activo','retenidos'])
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           v_activo, ARRAY['ambito_hmac','generacion','huella_peticion_hmac'])
       OR pg_catalog.jsonb_typeof(p_operacion #> '{sellos_hmac,retenidos}')
          <> 'array'
       OR pg_catalog.jsonb_array_length(
           p_operacion #> '{sellos_hmac,retenidos}') > 16
       OR coalesce(v_activo ->> 'ambito_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]informe-juridico[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR coalesce(v_activo ->> 'huella_peticion_hmac', '') !~
          '^hmac-sha256:vec[.]contratacion-temporal[.]informe-juridico[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(
                 p_operacion #> '{sellos_hmac,retenidos}') e(valor)
            WHERE NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
                      e.valor,
                      ARRAY['ambito_hmac','generacion','huella_peticion_hmac'])
               OR coalesce(e.valor ->> 'ambito_hmac', '') !~
                  '^hmac-sha256:vec[.]contratacion-temporal[.]informe-juridico[.]ambito/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
               OR coalesce(e.valor ->> 'huella_peticion_hmac', '') !~
                  '^hmac-sha256:vec[.]contratacion-temporal[.]informe-juridico[.]peticion/v[1-9][0-9]{0,8}:[0-9a-f]{64}$'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_each_text(
                 p_operacion -> 'referencias_candidatas') c
            WHERE c.value !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'preparación de informe jurídico no autorizada';
    END IF;

    SELECT r.* INTO v_reserva
      FROM vec_contratacion_temporal.reserva_informe_juridico r
     WHERE r.ambito_hmac = v_activo ->> 'ambito_hmac';
    v_encontrada := FOUND;
    IF NOT v_encontrada THEN
        FOR v_retenido IN
            SELECT valor
              FROM pg_catalog.jsonb_array_elements(
                  p_operacion #> '{sellos_hmac,retenidos}'
              ) AS e(valor)
        LOOP
            SELECT r.* INTO v_reserva
              FROM vec_contratacion_temporal.reserva_informe_juridico r
             WHERE r.ambito_hmac = v_retenido ->> 'ambito_hmac';
            IF FOUND THEN
                v_encontrada := true;
                EXIT;
            END IF;
        END LOOP;
    END IF;

    IF v_encontrada THEN
        IF v_reserva.huella_peticion_hmac IS DISTINCT FROM
              coalesce(v_retenido ->> 'huella_peticion_hmac',
                       v_activo ->> 'huella_peticion_hmac')
           OR v_reserva.operacion IS DISTINCT FROM p_operacion ->> 'operacion'
           OR v_reserva.organizacion_ref IS DISTINCT FROM
              p_operacion ->> 'organizacion_ref'
           OR v_reserva.expediente_ref IS DISTINCT FROM
              p_operacion ->> 'expediente_ref'
           OR v_reserva.version_expediente IS DISTINCT FROM
              (p_operacion ->> 'version_expediente')::numeric
           OR v_reserva.actor_ref IS DISTINCT FROM p_operacion ->> 'actor_ref'
           OR v_reserva.perfil_ref IS DISTINCT FROM p_operacion ->> 'perfil_ref' THEN
            resultado := 'idempotencia_reutilizada';
        ELSIF v_reserva.estado = 'confirmada' THEN
            resultado := 'confirmada';
        ELSE
            resultado := 'reutilizada';
        END IF;
    ELSE
        SELECT a.version, v.agregado_json
          INTO STRICT v_actual
         FROM vec_contratacion_temporal.expediente_integral_actual a
          JOIN vec_contratacion_temporal.expediente_version_integral v
            USING (expediente_ref, version)
         WHERE a.expediente_ref = p_operacion ->> 'expediente_ref';
        IF v_actual.version <> (p_operacion ->> 'version_expediente')::numeric
           OR v_actual.agregado_json ->> 'organizacion_ref' <>
              p_operacion ->> 'organizacion_ref'
           OR NOT vec_contratacion_temporal.informe_nuevo_admisible_ct123(
               v_actual.agregado_json) THEN
            RAISE EXCEPTION USING ERRCODE = '40001',
                MESSAGE = 'expediente no disponible para informe jurídico';
        END IF;
        v_reserva.ambito_hmac := v_activo ->> 'ambito_hmac';
        v_reserva.huella_peticion_hmac :=
            v_activo ->> 'huella_peticion_hmac';
        v_reserva.operacion := 'preparar';
        v_reserva.organizacion_ref := p_operacion ->> 'organizacion_ref';
        v_reserva.expediente_ref := p_operacion ->> 'expediente_ref';
        v_reserva.version_expediente := v_actual.version;
        v_reserva.actor_ref := p_operacion ->> 'actor_ref';
        v_reserva.perfil_ref := p_operacion ->> 'perfil_ref';
        v_reserva.reserva_ref :=
            p_operacion #>> '{referencias_candidatas,reserva_ref}';
        v_reserva.informe_ref :=
            p_operacion #>> '{referencias_candidatas,informe_ref}';
        v_reserva.documento_ref :=
            p_operacion #>> '{referencias_candidatas,documento_ref}';
        v_reserva.recibo_ref :=
            p_operacion #>> '{referencias_candidatas,recibo_ref}';
        v_reserva.auditoria_ref :=
            p_operacion #>> '{referencias_candidatas,auditoria_ref}';
        v_reserva.evento_ref :=
            p_operacion #>> '{referencias_candidatas,evento_ref}';
        v_reserva.expediente_anterior_json := v_actual.agregado_json;
        v_reserva.estado := 'reservada';
        resultado := 'reservada';
    END IF;

    IF resultado <> 'idempotencia_reutilizada' AND v_reserva.estado <> 'confirmada' THEN
        PERFORM vec_contratacion_temporal.antecedente_refiscalizacion_v1(
            v_reserva.expediente_anterior_json);
    END IF;
    RETURN QUERY SELECT resultado, v_reserva.expediente_anterior_json::text,
        v_reserva.reserva_ref, v_reserva.informe_ref, v_reserva.documento_ref,
        v_reserva.recibo_ref, v_reserva.auditoria_ref, v_reserva.evento_ref,
        v_reserva.ambito_hmac, v_reserva.huella_peticion_hmac,
        v_reserva.organizacion_ref, v_reserva.expediente_ref,
        v_reserva.version_expediente::bigint, v_reserva.actor_ref,
        v_reserva.perfil_ref, v_reserva.estado,
        CASE WHEN v_reserva.estado = 'confirmada'
             THEN vec_contratacion_temporal.recibo_informe_juridico_v1(
                      v_reserva.ambito_hmac)
        END;
EXCEPTION
    WHEN invalid_text_representation OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'preparación de informe jurídico inválida';
END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(
    p_operacion jsonb,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE (recibo_json jsonb)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    r vec_contratacion_temporal.reserva_informe_juridico%ROWTYPE;
    v_actual record;
    v_consumo record;
    v_decision jsonb;
    v_ahora timestamptz(6);
    v_carga_huella text;
    v_agregado_huella text;
    v_prueba bytea;
    v_payload_evento bytea;
    v_anterior text;
    v_secuencia numeric;
    v_actuacion jsonb;
    v_informe jsonb;
    v_expediente_esperado jsonb;
    v_borrador_material bytea;
    v_borrador_huella text;
    v_normativas_json text;
    v_anexos_json text;
    v_paquete_json text;
    v_paquete_huella text;
    v_contenido_esperado text;
    v_contexto_canonico bytea;
    v_contexto_huella text;
    v_anexos_borrador jsonb;
    v_elemento record;
    v_reserva_persistida boolean := false;
    v_version numeric;
    v_secuencia_actuacion numeric;
    v_retorno text;
    v_sustituye jsonb;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.pg_column_size(p_operacion) > 3145728
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion, ARRAY['actor_ref','actuacion','ambito_idempotencia_hmac',
           'autorizacion','borrador',
           'configuracion','documento','esquema','expediente_anterior',
           'expediente_ref','expediente_siguiente','huella_peticion_hmac',
           'instante_efecto','operacion','organizacion_ref','perfil_ref',
           'referencias','reserva_ref','version_anterior'])
       OR p_operacion ->> 'esquema' <>
          'vec.contratacion-temporal.confirmar-informe-juridico.v1'
       OR p_operacion ->> 'operacion' <> 'preparar'
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion -> 'referencias',
           ARRAY['auditoria_ref','documento_ref','evento_ref','informe_ref',
                 'recibo_ref','reserva_ref'])
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion -> 'documento',
           ARRAY['contenido_desarrollo','documento_ref','formato',
                 'huella_documento_sha256','huella_paquete_sha256',
                 'nombre','version_documento'])
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion -> 'configuracion',
           ARRAY['accion','anexos','definicion_huella_sha256',
                 'definicion_ref','definicion_version','evaluada_en',
                 'finalidad','plantilla','referencias_normativas',
                 'unidad_ejecutora_ref','valida_hasta'])
       OR NOT vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion -> 'autorizacion',
           ARRAY['accion','contexto_recurso_huella_sha256',
                 'decision_canonica_hex','decision_huella_sha256',
                 'decision_ref','finalidad','motivo_canonico_hex',
                 'perfil_activo_ref','perfil_version','persona_version',
                 'principal_id','recurso_ref']) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'confirmación de informe jurídico no autorizada';
    END IF;
    BEGIN
        v_decision := pg_catalog.convert_from(p_decision, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'decisión de informe jurídico inválida';
    END;
    v_anexos_borrador := coalesce(
        nullif(p_operacion #> '{borrador,anexos}', 'null'::jsonb),
        '[]'::jsonb);

    IF vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion -> 'borrador',
           ARRAY['anexos','canon','expediente_ref','huella_sha256',
                 'plantilla','referencias_normativas',
                 'version_esperada_expediente']) IS DISTINCT FROM TRUE
       OR vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion #> '{borrador,canon}',
           ARRAY['esquema','version_esquema']) IS DISTINCT FROM TRUE
       OR vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
           p_operacion #> '{borrador,plantilla}',
           ARRAY['huella_sha256','plantilla_ref','version'])
          IS DISTINCT FROM TRUE
       OR pg_catalog.jsonb_typeof(
              p_operacion #> '{borrador,referencias_normativas}')
          IS DISTINCT FROM 'array'
       OR coalesce(pg_catalog.jsonb_typeof(
              p_operacion #> '{borrador,anexos}'), '')
          NOT IN ('array', 'null')
       OR pg_catalog.jsonb_array_length(
              p_operacion #> '{borrador,referencias_normativas}')
          NOT BETWEEN 1 AND 64
       OR pg_catalog.jsonb_array_length(v_anexos_borrador) > 64
       OR coalesce(p_operacion #>> '{borrador,expediente_ref}', '')
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion #>> '{borrador,plantilla,plantilla_ref}', '')
          !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR coalesce(p_operacion #>> '{borrador,plantilla,version}', '')
          !~ '^[1-9][0-9]{0,15}$'
       OR (p_operacion #>> '{borrador,plantilla,version}')::numeric >
          9007199254740991::numeric
       OR coalesce(
              p_operacion #>> '{borrador,plantilla,huella_sha256}', '')
          !~ '^[0-9a-f]{64}$'
       OR p_operacion #>> '{borrador,plantilla,huella_sha256}' =
          pg_catalog.repeat('0', 64)
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(
                 p_operacion #> '{borrador,referencias_normativas}'
             ) AS n(valor)
            WHERE vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
                      n.valor, ARRAY['huella_sha256','norma_ref','version'])
                  IS DISTINCT FROM TRUE
               OR coalesce(n.valor ->> 'norma_ref', '')
                  !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR coalesce(n.valor ->> 'version', '')
                  !~ '^[1-9][0-9]{0,15}$'
               OR (n.valor ->> 'version')::numeric >
                  9007199254740991::numeric
               OR coalesce(n.valor ->> 'huella_sha256', '')
                  !~ '^[0-9a-f]{64}$'
               OR n.valor ->> 'huella_sha256' = pg_catalog.repeat('0', 64)
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(v_anexos_borrador)
                  AS a(valor)
            WHERE vec_contratacion_temporal.informe_juridico_claves_exactas_v1(
                      a.valor,
                      ARRAY['documento_ref','huella_sha256',
                            'version_documento']) IS DISTINCT FROM TRUE
               OR coalesce(a.valor ->> 'documento_ref', '')
                  !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
               OR coalesce(a.valor ->> 'version_documento', '')
                  !~ '^[1-9][0-9]{0,15}$'
               OR (a.valor ->> 'version_documento')::numeric >
                  9007199254740991::numeric
               OR coalesce(a.valor ->> 'huella_sha256', '')
                  !~ '^[0-9a-f]{64}$'
               OR a.valor ->> 'huella_sha256' = pg_catalog.repeat('0', 64)
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(
                 p_operacion #> '{borrador,referencias_normativas}'
             ) WITH ORDINALITY AS actual(valor, posicion)
             JOIN pg_catalog.jsonb_array_elements(
                 p_operacion #> '{borrador,referencias_normativas}'
             ) WITH ORDINALITY AS anterior(valor, posicion)
               ON anterior.posicion + 1 = actual.posicion
            WHERE anterior.valor ->> 'norma_ref' >=
                  actual.valor ->> 'norma_ref'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_array_elements(v_anexos_borrador)
                  WITH ORDINALITY AS actual(valor, posicion)
             JOIN pg_catalog.jsonb_array_elements(v_anexos_borrador)
                  WITH ORDINALITY AS anterior(valor, posicion)
               ON anterior.posicion + 1 = actual.posicion
            WHERE anterior.valor ->> 'documento_ref' >
                  actual.valor ->> 'documento_ref'
               OR (anterior.valor ->> 'documento_ref' =
                   actual.valor ->> 'documento_ref'
                   AND (anterior.valor ->> 'version_documento')::numeric >=
                       (actual.valor ->> 'version_documento')::numeric)
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'borrador de informe jurídico inválido';
    END IF;

    v_borrador_material :=
        pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
            p_operacion #>> '{borrador,canon,esquema}', 'UTF8'))) ||
        pg_catalog.convert_to(
            p_operacion #>> '{borrador,canon,esquema}', 'UTF8') ||
        pg_catalog.int2send(
            (p_operacion #>> '{borrador,canon,version_esquema}')::smallint) ||
        pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
            p_operacion #>> '{borrador,expediente_ref}', 'UTF8'))) ||
        pg_catalog.convert_to(
            p_operacion #>> '{borrador,expediente_ref}', 'UTF8') ||
        pg_catalog.int8send(
            (p_operacion #>> '{borrador,version_esperada_expediente}')::bigint) ||
        pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
            p_operacion #>> '{borrador,plantilla,plantilla_ref}', 'UTF8'))) ||
        pg_catalog.convert_to(
            p_operacion #>> '{borrador,plantilla,plantilla_ref}', 'UTF8') ||
        pg_catalog.int8send(
            (p_operacion #>> '{borrador,plantilla,version}')::bigint) ||
        pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
            p_operacion #>> '{borrador,plantilla,huella_sha256}', 'UTF8'))) ||
        pg_catalog.convert_to(
            p_operacion #>> '{borrador,plantilla,huella_sha256}', 'UTF8') ||
        pg_catalog.int2send(pg_catalog.jsonb_array_length(
            p_operacion #> '{borrador,referencias_normativas}')::smallint);
    FOR v_elemento IN
        SELECT n.valor
          FROM pg_catalog.jsonb_array_elements(
              p_operacion #> '{borrador,referencias_normativas}'
          ) WITH ORDINALITY AS n(valor, posicion)
         ORDER BY n.posicion
    LOOP
        v_borrador_material := v_borrador_material ||
            pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
                v_elemento.valor ->> 'norma_ref', 'UTF8'))) ||
            pg_catalog.convert_to(v_elemento.valor ->> 'norma_ref', 'UTF8') ||
            pg_catalog.int8send((v_elemento.valor ->> 'version')::bigint) ||
            pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
                v_elemento.valor ->> 'huella_sha256', 'UTF8'))) ||
            pg_catalog.convert_to(
                v_elemento.valor ->> 'huella_sha256', 'UTF8');
    END LOOP;
    v_borrador_material := v_borrador_material ||
        pg_catalog.int2send(pg_catalog.jsonb_array_length(
            v_anexos_borrador)::smallint);
    FOR v_elemento IN
        SELECT a.valor
          FROM pg_catalog.jsonb_array_elements(v_anexos_borrador)
               WITH ORDINALITY AS a(valor, posicion)
         ORDER BY a.posicion
    LOOP
        v_borrador_material := v_borrador_material ||
            pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
                v_elemento.valor ->> 'documento_ref', 'UTF8'))) ||
            pg_catalog.convert_to(
                v_elemento.valor ->> 'documento_ref', 'UTF8') ||
            pg_catalog.int8send(
                (v_elemento.valor ->> 'version_documento')::bigint) ||
            pg_catalog.int4send(pg_catalog.octet_length(pg_catalog.convert_to(
                v_elemento.valor ->> 'huella_sha256', 'UTF8'))) ||
            pg_catalog.convert_to(
                v_elemento.valor ->> 'huella_sha256', 'UTF8');
    END LOOP;
    v_borrador_huella := pg_catalog.encode(
        pg_catalog.sha256(v_borrador_material), 'hex');

    SELECT pg_catalog.string_agg(
               '{"norma_ref":"' || (n.valor ->> 'norma_ref') ||
               '","version":' || (n.valor ->> 'version') ||
               ',"huella_sha256":"' || (n.valor ->> 'huella_sha256') || '"}',
               ',' ORDER BY n.posicion)
      INTO v_normativas_json
      FROM pg_catalog.jsonb_array_elements(
          p_operacion #> '{borrador,referencias_normativas}'
      ) WITH ORDINALITY AS n(valor, posicion);
    SELECT coalesce(pg_catalog.string_agg(
               '{"documento_ref":"' || (a.valor ->> 'documento_ref') ||
               '","version_documento":' ||
               (a.valor ->> 'version_documento') ||
               ',"huella_sha256":"' || (a.valor ->> 'huella_sha256') || '"}',
               ',' ORDER BY a.posicion), '')
      INTO v_anexos_json
      FROM pg_catalog.jsonb_array_elements(v_anexos_borrador)
           WITH ORDINALITY AS a(valor, posicion);
    v_paquete_json :=
        '{"esquema":"vec.dipgra.contratacion-temporal.informe-juridico.paquete-datos"' ||
        ',"version_esquema":1,"expediente_ref":"' ||
        (p_operacion #>> '{borrador,expediente_ref}') ||
        '","version_expediente":' ||
        (p_operacion #>> '{borrador,version_esperada_expediente}') ||
        ',"plantilla":{"plantilla_ref":"' ||
        (p_operacion #>> '{borrador,plantilla,plantilla_ref}') ||
        '","version":' ||
        (p_operacion #>> '{borrador,plantilla,version}') ||
        ',"huella_sha256":"' ||
        (p_operacion #>> '{borrador,plantilla,huella_sha256}') ||
        '"},"referencias_normativas":[' || v_normativas_json ||
        '],"anexos":[' || v_anexos_json ||
        '],"huella_borrador_sha256":"' || v_borrador_huella || '"}';
    v_paquete_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_paquete_json, 'UTF8')), 'hex');
    v_contenido_esperado :=
        'DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA' ||
        chr(10) || chr(10) || 'INFORME JURIDICO PROVISIONAL' || chr(10) ||
        'Pendiente de revision y firma.' || chr(10) || chr(10) ||
        'Datos juridicos canónicos:' || chr(10) || v_paquete_json || chr(10);

    v_carga_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(p_operacion::text, 'UTF8')), 'hex');
    SELECT x.* INTO r
      FROM vec_contratacion_temporal.reserva_informe_juridico x
     WHERE x.ambito_hmac = p_operacion ->> 'ambito_idempotencia_hmac'
     FOR UPDATE;
    v_reserva_persistida := FOUND;
    IF v_reserva_persistida AND (
       r.huella_peticion_hmac <> p_operacion ->> 'huella_peticion_hmac'
       OR r.operacion <> p_operacion ->> 'operacion'
       OR r.organizacion_ref <> p_operacion ->> 'organizacion_ref'
       OR r.expediente_ref <> p_operacion ->> 'expediente_ref'
       OR r.version_expediente <>
          (p_operacion ->> 'version_anterior')::numeric
       OR r.actor_ref <> p_operacion ->> 'actor_ref'
       OR r.perfil_ref <> p_operacion ->> 'perfil_ref') THEN
        RAISE EXCEPTION USING ERRCODE = '23505',
            MESSAGE = 'idempotencia de informe jurídico reutilizada';
    END IF;
    IF r.estado = 'confirmada' THEN
        RETURN QUERY SELECT
            vec_contratacion_temporal.recibo_informe_juridico_v1(r.ambito_hmac);
        RETURN;
    END IF;
    IF NOT v_reserva_persistida THEN
        r.ambito_hmac := p_operacion ->> 'ambito_idempotencia_hmac';
        r.huella_peticion_hmac := p_operacion ->> 'huella_peticion_hmac';
        r.operacion := 'preparar';
        r.organizacion_ref := p_operacion ->> 'organizacion_ref';
        r.expediente_ref := p_operacion ->> 'expediente_ref';
        r.version_expediente :=
            (p_operacion ->> 'version_anterior')::numeric;
        r.actor_ref := p_operacion ->> 'actor_ref';
        r.perfil_ref := p_operacion ->> 'perfil_ref';
        r.reserva_ref := p_operacion ->> 'reserva_ref';
        r.informe_ref := p_operacion #>> '{referencias,informe_ref}';
        r.documento_ref := p_operacion #>> '{referencias,documento_ref}';
        r.recibo_ref := p_operacion #>> '{referencias,recibo_ref}';
        r.auditoria_ref := p_operacion #>> '{referencias,auditoria_ref}';
        r.evento_ref := p_operacion #>> '{referencias,evento_ref}';
        r.expediente_anterior_json := p_operacion -> 'expediente_anterior';
        r.estado := 'reservada';
    END IF;

    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp());
    IF r.operacion <> 'preparar'
       OR r.reserva_ref <> p_operacion ->> 'reserva_ref'
       OR r.reserva_ref <> p_operacion #>> '{referencias,reserva_ref}'
       OR r.informe_ref <> p_operacion #>> '{referencias,informe_ref}'
       OR r.documento_ref <> p_operacion #>> '{referencias,documento_ref}'
       OR r.recibo_ref <> p_operacion #>> '{referencias,recibo_ref}'
       OR r.auditoria_ref <> p_operacion #>> '{referencias,auditoria_ref}'
       OR r.evento_ref <> p_operacion #>> '{referencias,evento_ref}'
       OR r.organizacion_ref <> p_operacion ->> 'organizacion_ref'
       OR r.expediente_ref <> p_operacion ->> 'expediente_ref'
       OR r.version_expediente <> (p_operacion ->> 'version_anterior')::numeric
       OR r.actor_ref <> p_operacion ->> 'actor_ref'
       OR r.perfil_ref <> p_operacion ->> 'perfil_ref'
       OR r.expediente_anterior_json <> p_operacion -> 'expediente_anterior'
       OR p_operacion #>> '{configuracion,accion}' <>
          'contratacion_temporal.informe_juridico.generar'
       OR p_operacion #>> '{configuracion,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR p_operacion #>> '{configuracion,unidad_ejecutora_ref}' <>
          r.expediente_anterior_json #>> '{asignacion,unidad_ref}'
       OR (p_operacion #>> '{configuracion,definicion_version}')::numeric
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_operacion #>> '{configuracion,definicion_huella_sha256}'
          !~ '^[0-9a-f]{64}$'
       OR (p_operacion #>> '{configuracion,evaluada_en}')::timestamptz >
          (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora < (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora >= (p_operacion #>> '{configuracion,valida_hasta}')::timestamptz
       OR p_operacion #>> '{borrador,canon,esquema}' <>
          'vec.dipgra.contratacion-temporal.informe-juridico.borrador'
       OR p_operacion #>> '{borrador,canon,version_esquema}' <> '1'
       OR p_operacion #>> '{borrador,expediente_ref}' <> r.expediente_ref
       OR (p_operacion #>> '{borrador,version_esperada_expediente}')::numeric <>
          r.version_expediente
       OR p_operacion #> '{borrador,plantilla}' <>
          p_operacion #> '{configuracion,plantilla}'
       OR p_operacion #> '{borrador,referencias_normativas}' <>
          p_operacion #> '{configuracion,referencias_normativas}'
       OR p_operacion #> '{borrador,anexos}' <>
          p_operacion #> '{configuracion,anexos}'
       OR p_operacion #>> '{borrador,huella_sha256}' <> v_borrador_huella
       OR p_operacion #>> '{documento,documento_ref}' <> r.documento_ref
       OR (p_operacion #>> '{documento,version_documento}')::numeric
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_operacion #>> '{documento,formato}' <>
          'text/plain; charset=utf-8'
       OR p_operacion #>> '{documento,nombre}' <>
          pg_catalog.btrim(p_operacion #>> '{documento,nombre}')
       OR pg_catalog.octet_length(
          p_operacion #>> '{documento,nombre}') NOT BETWEEN 1 AND 512
       OR p_operacion #>> '{documento,nombre}' <>
          'informe-juridico-desarrollo.txt'
       OR pg_catalog.octet_length(
          p_operacion #>> '{documento,contenido_desarrollo}')
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.strpos(
          p_operacion #>> '{documento,contenido_desarrollo}',
          'DOCUMENTO DE DESARROLLO') = 0
       OR p_operacion #>> '{documento,huella_documento_sha256}' <>
          pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
              p_operacion #>> '{documento,contenido_desarrollo}', 'UTF8')), 'hex')
       OR p_operacion #>> '{documento,contenido_desarrollo}' <>
          v_contenido_esperado
       OR p_operacion #>> '{documento,huella_paquete_sha256}' <>
          v_paquete_huella
       OR p_operacion #>> '{autorizacion,accion}' <>
          'contratacion_temporal.informe_juridico.generar'
       OR p_operacion #>> '{autorizacion,finalidad}' <>
          'gestionar_contratacion_temporal'
       OR p_operacion #>> '{autorizacion,principal_id}' <> r.actor_ref
       OR p_operacion #>> '{autorizacion,perfil_activo_ref}' <> r.perfil_ref
       OR p_operacion #>> '{autorizacion,recurso_ref}' <> r.expediente_ref
       OR p_operacion #>> '{autorizacion,decision_canonica_hex}' <>
          pg_catalog.encode(p_decision, 'hex')
       OR p_operacion #>> '{autorizacion,motivo_canonico_hex}' <>
          pg_catalog.encode(p_motivo, 'hex')
       OR (p_operacion #>> '{autorizacion,persona_version}')::numeric <>
          p_persona_version
       OR (p_operacion #>> '{autorizacion,perfil_version}')::numeric <>
          p_perfil_version
       OR p_operacion #>> '{autorizacion,decision_huella_sha256}' <>
          pg_catalog.encode(pg_catalog.sha256(p_decision), 'hex')
       OR v_decision ->> 'principal_id' <> r.actor_ref
       OR v_decision ->> 'perfil_activo_ref' <> r.perfil_ref
       OR v_decision ->> 'recurso_ref' <> r.expediente_ref
       OR v_decision ->> 'accion' <>
          'contratacion_temporal.informe_juridico.generar'
       OR v_decision ->> 'modulo_id' <> 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' <>
          'informe_juridico_contratacion_temporal'
       OR v_decision ->> 'finalidad' <> 'gestionar_contratacion_temporal'
       OR v_decision ->> 'decision_ref' <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR v_decision ->> 'contexto_recurso_huella_sha256' <>
          p_operacion #>> '{autorizacion,contexto_recurso_huella_sha256}' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'coordenadas de informe jurídico divergentes';
    END IF;

    SELECT v.* INTO STRICT v_actual
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        USING (expediente_ref, version)
     WHERE a.expediente_ref = r.expediente_ref
     FOR UPDATE OF a, v;
    IF v_actual.version <> r.version_expediente
       OR v_actual.agregado_json <> r.expediente_anterior_json
       OR NOT vec_contratacion_temporal.informe_nuevo_admisible_ct123(
           v_actual.agregado_json) THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS de informe jurídico perdido';
    END IF;
    PERFORM vec_contratacion_temporal.antecedente_refiscalizacion_v1(
        v_actual.agregado_json);
    v_version := r.version_expediente + 1;
    v_secuencia_actuacion := pg_catalog.jsonb_array_length(
        v_actual.agregado_json -> 'actuaciones') + 1;
    v_retorno := v_actual.agregado_json #>> '{fiscalizacion,retorno,retorno_ref}';
    v_sustituye := pg_catalog.jsonb_build_object(
        'informe_ref', v_actual.agregado_json #>> '{informe_juridico,informe_ref}',
        'documento_ref', v_actual.agregado_json #>> '{informe_juridico,documento_ref}',
        'retorno_ref', v_retorno
    );

    v_actuacion := pg_catalog.jsonb_build_object(
        'secuencia', v_secuencia_actuacion,
        'version_expediente', v_version,
        'accion_clave', 'contratacion_temporal.informe_juridico.generar',
        'actor_ref', r.actor_ref,
        'unidad_ref', p_operacion #>> '{configuracion,unidad_ejecutora_ref}',
        'recibo_ref', r.recibo_ref,
        'realizada_en', p_operacion -> 'instante_efecto',
        'fase_origen', v_actual.agregado_json -> 'fase_actual',
        'fase_destino', 'subsanacion_unidad',
        'estado_origen', v_actual.agregado_json -> 'estado_actual',
        'estado_destino', 'incidencia',
        'documentos_ref', pg_catalog.jsonb_build_array(r.documento_ref),
        'retorno_ref', v_retorno
    );
    v_informe := pg_catalog.jsonb_build_object(
        'borrador', p_operacion -> 'borrador',
        'informe_ref', r.informe_ref,
        'documento_ref', r.documento_ref,
        'version_documento',
            (p_operacion #>> '{documento,version_documento}')::numeric,
        'huella_documento_sha256',
            p_operacion #>> '{documento,huella_documento_sha256}',
        'emitido_en', p_operacion -> 'instante_efecto',
        'sustituye', v_sustituye,
        'actuacion_registro', pg_catalog.jsonb_build_object(
            'secuencia', v_secuencia_actuacion,
            'version_expediente', v_version,
            'accion_clave', 'contratacion_temporal.informe_juridico.generar',
            'fase_destino', 'subsanacion_unidad',
            'recibo_ref', r.recibo_ref,
            'informe_ref', r.informe_ref,
            'documento_ref', r.documento_ref,
            'version_documento',
                (p_operacion #>> '{documento,version_documento}')::numeric,
            'huella_documento_sha256',
                p_operacion #>> '{documento,huella_documento_sha256}',
            'huella_borrador_sha256',
                p_operacion #>> '{borrador,huella_sha256}'
        )
    );
    v_expediente_esperado := v_actual.agregado_json ||
        pg_catalog.jsonb_build_object(
            'version', v_version,
            'fase_actual', 'subsanacion_unidad',
            'estado_actual', 'incidencia',
            'actualizado_en', p_operacion -> 'instante_efecto',
            'actuaciones', (v_actual.agregado_json -> 'actuaciones') ||
                pg_catalog.jsonb_build_array(v_actuacion),
            'informe_juridico', v_informe
        );
    IF v_secuencia_actuacion <> v_version
       OR p_operacion -> 'actuacion' IS DISTINCT FROM v_actuacion
       OR p_operacion -> 'expediente_siguiente' IS DISTINCT FROM
          v_expediente_esperado THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'proyección de informe jurídico divergente';
    END IF;

    v_contexto_canonico := pg_catalog.convert_to(
        '{"ambitos":{"estado_previo":"' ||
        (v_actual.agregado_json ->> 'estado_actual') ||
        '","expediente_ref":"' || r.expediente_ref ||
        '","fase_previa":"' ||
        (v_actual.agregado_json ->> 'fase_actual') ||
        '","organizacion_ref":"' || r.organizacion_ref ||
        '"},"atributos":{"ambito_idempotencia_hmac":"' || r.ambito_hmac ||
        '","borrador_huella_sha256":"' || v_borrador_huella ||
        '","configuracion_huella_sha256":"' ||
        (p_operacion #>> '{configuracion,definicion_huella_sha256}') ||
        '","configuracion_ref":"' ||
        (p_operacion #>> '{configuracion,definicion_ref}') ||
        '","configuracion_version":"' ||
        (p_operacion #>> '{configuracion,definicion_version}') ||
        '","huella_peticion_hmac":"' || r.huella_peticion_hmac ||
        '","plantilla_huella_sha256":"' ||
        (p_operacion #>> '{configuracion,plantilla,huella_sha256}') ||
        '","plantilla_ref":"' ||
        (p_operacion #>> '{configuracion,plantilla,plantilla_ref}') ||
        '","plantilla_version":"' ||
        (p_operacion #>> '{configuracion,plantilla,version}') ||
        '","version_expediente":"' || r.version_expediente::text || '"}}',
        'UTF8');
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_canonico), 'hex');
    IF p_operacion #>> '{autorizacion,contexto_recurso_huella_sha256}' <>
          v_contexto_huella
       OR v_decision ->> 'contexto_recurso_huella_sha256' <>
          v_contexto_huella THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'contexto autorizado de informe jurídico divergente';
    END IF;

    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3
           .registrar_y_consumir_informe_juridico_v3_atestada(
          p_capacidad, p_decision, p_motivo, p_contexto,
          p_persona_version, p_perfil_version, p_payload, p_sobre,
          p_evidencia, p_raiz
      );
    IF v_consumo.decision_ref <>
          p_operacion #>> '{autorizacion,decision_ref}'
       OR v_consumo.efecto_ref <> r.expediente_ref
       OR v_consumo.huella_efecto_sha256 <>
          v_contexto_huella
       OR coalesce(v_consumo.auditoria_ref, '') !~
          '^aud_v3_[0-9a-f]{32}$'
       OR v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consumo de autorización de informe jurídico divergente';
    END IF;
    v_ahora := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp());
    IF v_ahora < (p_operacion ->> 'instante_efecto')::timestamptz
       OR v_ahora >=
          (p_operacion #>> '{configuracion,valida_hasta}')::timestamptz THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'vigencia de informe jurídico agotada';
    END IF;

    IF NOT v_reserva_persistida THEN
        INSERT INTO vec_contratacion_temporal.reserva_informe_juridico (
            ambito_hmac, huella_peticion_hmac, operacion, organizacion_ref,
            expediente_ref, version_expediente, actor_ref, perfil_ref,
            reserva_ref, informe_ref, documento_ref, recibo_ref,
            auditoria_ref, evento_ref, expediente_anterior_json, reservada_en
        ) VALUES (
            r.ambito_hmac, r.huella_peticion_hmac, r.operacion,
            r.organizacion_ref, r.expediente_ref, r.version_expediente,
            r.actor_ref, r.perfil_ref, r.reserva_ref, r.informe_ref,
            r.documento_ref, r.recibo_ref, r.auditoria_ref, r.evento_ref,
            r.expediente_anterior_json, v_ahora
        );
    END IF;

    INSERT INTO vec_contratacion_temporal.documento_informe_juridico_desarrollo (
        documento_ref, informe_ref, ambito_hmac, version_documento,
        formato, nombre, huella_documento_sha256, huella_paquete_sha256,
        contenido_desarrollo, creada_en
    ) VALUES (
        r.documento_ref, r.informe_ref, r.ambito_hmac,
        (p_operacion #>> '{documento,version_documento}')::numeric,
        p_operacion #>> '{documento,formato}',
        p_operacion #>> '{documento,nombre}',
        p_operacion #>> '{documento,huella_documento_sha256}',
        p_operacion #>> '{documento,huella_paquete_sha256}',
        p_operacion #>> '{documento,contenido_desarrollo}', v_ahora
    );

    v_agregado_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            (p_operacion -> 'expediente_siguiente')::text, 'UTF8')), 'hex');
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-EXPEDIENTE-INFORME-JURIDICO-V1' || chr(10) ||
        r.expediente_ref || chr(10) || v_version::text || chr(10) || v_agregado_huella ||
        chr(10) || r.reserva_ref || chr(10) || r.recibo_ref || chr(10) ||
        v_consumo.decision_ref || chr(10) || v_ahora::text, 'UTF8');
    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref, version, agregado_json, agregado_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, flujo_ref, flujo_version,
        flujo_huella_sha256, fase_clave, estado, origen_version,
        operacion_ref, registrada_en
    ) VALUES (
        r.expediente_ref, v_version, p_operacion -> 'expediente_siguiente',
        v_agregado_huella, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        v_actual.flujo_ref, v_actual.flujo_version,
        v_actual.flujo_huella_sha256, 'subsanacion_unidad', 'incidencia',
        'informe_juridico_o5', r.reserva_ref, v_ahora
    );
    UPDATE vec_contratacion_temporal.expediente_integral_actual
       SET version = v_version, actualizada_en = v_ahora, operacion_ref = r.reserva_ref
     WHERE expediente_ref = r.expediente_ref AND version = r.version_expediente;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'CAS final de informe jurídico perdido';
    END IF;

    v_prueba := pg_catalog.convert_to(
        'VEC-CT-ACTUACION-INFORME-JURIDICO-V1' || chr(10) ||
        (p_operacion -> 'actuacion')::text || chr(10) || r.recibo_ref ||
        chr(10) || v_ahora::text, 'UTF8');
    INSERT INTO vec_contratacion_temporal.actuacion_expediente_integral (
        expediente_ref, secuencia, version_expediente, operacion_ref,
        recibo_ref, actuacion_json, actuacion_json_huella_sha256,
        prueba_canonica, prueba_huella_sha256, registrada_en
    ) VALUES (
        r.expediente_ref, v_secuencia_actuacion, v_version, r.reserva_ref, r.recibo_ref,
        p_operacion -> 'actuacion', pg_catalog.encode(pg_catalog.sha256(
            pg_catalog.convert_to(
                (p_operacion -> 'actuacion')::text, 'UTF8')), 'hex'),
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'), v_ahora
    );

    SELECT secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia, v_anterior
      FROM vec_contratacion_temporal.control_cadenas_expediente_integral
     WHERE control_id
     FOR UPDATE;
    IF v_secuencia >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE = '22003',
            MESSAGE = 'límite de outbox alcanzado';
    END IF;
    v_secuencia := v_secuencia + 1;
    v_payload_evento := pg_catalog.convert_to(pg_catalog.jsonb_build_object(
        'esquema', 'vec.contratacion-temporal.informe-juridico-emitido.v1',
        'expediente_ref', r.expediente_ref,
        'version_resultante', v_version,
        'informe_ref', r.informe_ref,
        'documento_ref', r.documento_ref,
        'recibo_ref', r.recibo_ref
    )::text, 'UTF8');
    INSERT INTO vec_contratacion_temporal.outbox_expediente_integral (
        evento_ref, secuencia, operacion_ref, expediente_ref,
        version_expediente, tipo_evento, payload_canonico,
        payload_huella_sha256, anterior_sha256, huella_sha256, registrada_en
    ) VALUES (
        r.evento_ref, v_secuencia, r.reserva_ref, r.expediente_ref, v_version,
        'contratacion_temporal.informe_juridico_emitido', v_payload_evento,
        pg_catalog.encode(pg_catalog.sha256(v_payload_evento), 'hex'),
        v_anterior, pg_catalog.encode(pg_catalog.sha256(
            v_anterior::bytea || v_payload_evento), 'hex'), v_ahora
    );
    UPDATE vec_contratacion_temporal.control_cadenas_expediente_integral
       SET secuencia_outbox = v_secuencia,
           cabeza_outbox_sha256 = pg_catalog.encode(pg_catalog.sha256(
               v_anterior::bytea || v_payload_evento), 'hex'),
           actualizada_en = v_ahora
     WHERE control_id;

    INSERT INTO vec_contratacion_temporal.terminal_informe_juridico (
        ambito_hmac, huella_peticion_hmac, decision_ref,
        decision_huella_sha256, consumo_huella_sha256, auditoria_ref,
        documento_ref,
        huella_borrador_sha256, configuracion_ref, configuracion_version,
        configuracion_huella_sha256, carga_huella_sha256, confirmada_en
    ) VALUES (
        r.ambito_hmac, r.huella_peticion_hmac, v_consumo.decision_ref,
        p_operacion #>> '{autorizacion,decision_huella_sha256}',
        v_consumo.consumo_huella_sha256, v_consumo.auditoria_ref,
        r.documento_ref,
        p_operacion #>> '{borrador,huella_sha256}',
        p_operacion #>> '{configuracion,definicion_ref}',
        (p_operacion #>> '{configuracion,definicion_version}')::numeric,
        p_operacion #>> '{configuracion,definicion_huella_sha256}',
        v_carga_huella, v_ahora
    );
    UPDATE vec_contratacion_temporal.reserva_informe_juridico
       SET estado = 'confirmada', confirmada_en = v_ahora
     WHERE ambito_hmac = r.ambito_hmac AND estado = 'reservada';
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = '40001',
            MESSAGE = 'reserva de informe jurídico perdida';
    END IF;
    RETURN QUERY SELECT
        vec_contratacion_temporal.recibo_informe_juridico_v1(r.ambito_hmac);
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow OR
         numeric_value_out_of_range THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'entrada de informe jurídico inválida';
END
$funcion$;
-- Versión del expediente en la que se emitió el informe nuevo vigente (0 si
-- el informe vigente es el inicial). Las firmas registradas antes de esa
-- versión son de la ronda anterior del circuito. Misma frontera que la
-- consulta de firmas de CT118: sin datos personales y solo de la
-- organización del contexto.
CREATE FUNCTION vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(
    p_organizacion_ref text, p_expediente_ref text
) RETURNS numeric
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog SET row_security = on SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE v_agregado jsonb;
BEGIN
    IF current_user <> 'vec_contratacion_temporal_propietario'
       OR session_user = current_user
       OR NOT pg_has_role(session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
       OR pg_has_role(session_user, 'vec_contratacion_temporal_migrador', 'MEMBER') THEN
        RAISE EXCEPTION 'consulta de ronda de firma denegada' USING ERRCODE = '42501';
    END IF;
    IF p_organizacion_ref IS NULL OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_expediente_ref IS NULL OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION 'consulta de ronda de firma inválida' USING ERRCODE = '22023';
    END IF;
    SELECT v.agregado_json INTO v_agregado
      FROM vec_contratacion_temporal.expediente_integral_actual a
      JOIN vec_contratacion_temporal.expediente_version_integral v
        ON v.expediente_ref = a.expediente_ref AND v.version = a.version
     WHERE a.expediente_ref = p_expediente_ref;
    IF NOT FOUND THEN
        RETURN 0;
    END IF;
    IF v_agregado ->> 'organizacion_ref' IS DISTINCT FROM p_organizacion_ref THEN
        RAISE EXCEPTION 'consulta de ronda de firma denegada' USING ERRCODE = '42501';
    END IF;
    IF pg_catalog.jsonb_typeof(v_agregado #> '{informe_juridico,sustituye}') IS DISTINCT FROM 'object'
       OR coalesce(v_agregado #>> '{informe_juridico,actuacion_registro,version_expediente}', '')
          !~ '^[1-9][0-9]{0,15}$' THEN
        RETURN 0;
    END IF;
    RETURN (v_agregado #>> '{informe_juridico,actuacion_registro,version_expediente}')::numeric;
END
$funcion$;

-- ============================================================ POLÍTICA
-- Política de informe nuevo tras subsanar publicada por la aplicación al
-- arrancar desde el catálogo. Solo adición: cada versión es inmutable y la
-- vigente es la de mayor número. Nadie más que el propietario la lee.
CREATE TABLE vec_contratacion_temporal.politica_informe_tras_subsanacion (
    version bigint PRIMARY KEY,
    exige_informe_nuevo boolean NOT NULL,
    fuente_ref text NOT NULL,
    publicado_por text NOT NULL,
    publicado_en timestamptz(6) NOT NULL,
    CHECK (version BETWEEN 1 AND 9007199254740991),
    CHECK (fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
    CHECK (pg_catalog.length(publicado_por) BETWEEN 1 AND 63),
    CHECK (publicado_en = pg_catalog.date_trunc('microseconds', publicado_en))
);
CREATE TRIGGER politica_informe_tras_subsanacion_inmutable
BEFORE UPDATE OR DELETE
ON vec_contratacion_temporal.politica_informe_tras_subsanacion
FOR EACH ROW
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
CREATE TRIGGER politica_informe_tras_subsanacion_no_truncar
BEFORE TRUNCATE
ON vec_contratacion_temporal.politica_informe_tras_subsanacion
FOR EACH STATEMENT
EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1();
ALTER TABLE vec_contratacion_temporal.politica_informe_tras_subsanacion
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contratacion_temporal.politica_informe_tras_subsanacion
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_total
    ON vec_contratacion_temporal.politica_informe_tras_subsanacion
    TO vec_contratacion_temporal_propietario
    USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_contratacion_temporal.politica_informe_tras_subsanacion
FROM PUBLIC,
    vec_contratacion_temporal_migrador,
    vec_contratacion_temporal_ejecutor,
    vec_contratacion_temporal_gobernador;

-- La llama el gobernador al arrancar. Si coincide con la vigente (o, sin
-- ninguna, pide la conducta de siempre: no exigir) devuelve «vigente» sin
-- escribir; si difiere, añade la versión siguiente y devuelve «publicada».
CREATE FUNCTION vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(
    p_exige_informe_nuevo boolean,
    p_fuente_ref text
)
RETURNS TABLE (resultado text, version bigint)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_actual record;
    v_hay boolean;
BEGIN
    IF session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_gobernador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER') THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'publicación de política de informe nuevo no autorizada';
    END IF;
    IF p_exige_informe_nuevo IS NULL OR p_fuente_ref IS NULL
       OR p_fuente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'política de informe nuevo no válida';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_contratacion_temporal:politica_informe_tras_subsanacion', 0));
    SELECT p.version, p.exige_informe_nuevo INTO v_actual
      FROM vec_contratacion_temporal.politica_informe_tras_subsanacion p
     ORDER BY p.version DESC
     LIMIT 1;
    v_hay := FOUND;
    IF (v_hay AND v_actual.exige_informe_nuevo = p_exige_informe_nuevo)
       OR (NOT v_hay AND NOT p_exige_informe_nuevo) THEN
        RETURN QUERY SELECT 'vigente'::text,
            CASE WHEN v_hay THEN v_actual.version ELSE 0::bigint END;
        RETURN;
    END IF;
    RETURN QUERY
    INSERT INTO vec_contratacion_temporal.politica_informe_tras_subsanacion AS p (
        version, exige_informe_nuevo, fuente_ref, publicado_por, publicado_en
    ) VALUES (
        CASE WHEN v_hay THEN v_actual.version + 1 ELSE 1 END,
        p_exige_informe_nuevo, p_fuente_ref, session_user::text,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    )
    RETURNING 'publicada'::text, p.version;
END
$funcion$;

-- Política vigente: sin ninguna publicada, no se exige (conducta de siempre).
-- Solo la usa la confirmación de CT93, que se ejecuta como propietario.
CREATE FUNCTION vec_contratacion_temporal.informe_nuevo_exigido_ct123()
RETURNS boolean LANGUAGE sql STABLE SET search_path = pg_catalog
AS $funcion$
    SELECT coalesce((
        SELECT p.exige_informe_nuevo
          FROM vec_contratacion_temporal.politica_informe_tras_subsanacion p
         ORDER BY p.version DESC
         LIMIT 1), false)
$funcion$;

-- La confirmación de la nueva fiscalización (CT93, también vía la fachada
-- v2 de CT120) recibe la comprobación justo después de leer el antecedente.
-- Se exige la preimagen exacta con un único punto de inserción; el cuerpo
-- resultante, los permisos, el propietario y la configuración se verifican.
DO $refiscalizacion$
DECLARE
    v_ancla text := E'    v_antecedente := vec_contratacion_temporal.antecedente_refiscalizacion_v1(v_actual.agregado_json);\n';
    v_insercion text := E'    -- CT123: con la política publicada que exige informe nuevo tras\n    -- subsanar, no se fiscaliza de nuevo hasta emitirlo para este retorno.\n    IF vec_contratacion_temporal.informe_nuevo_exigido_ct123()\n       AND vec_contratacion_temporal.informe_nuevo_admisible_ct123(v_actual.agregado_json) THEN\n        RAISE EXCEPTION USING ERRCODE = ''55000'',\n            MESSAGE = ''informe jurídico nuevo tras subsanación pendiente'';\n    END IF;\n';
    v_antes record; v_despues record; v_definicion text;
BEGIN
    SELECT p.oid, pg_catalog.pg_get_functiondef(p.oid) AS definicion, p.proacl AS acl,
           p.proowner AS propietario, p.proconfig AS configuracion, p.prosecdef AS definidor
      INTO v_antes FROM pg_catalog.pg_proc p
     WHERE p.oid = pg_catalog.to_regprocedure('vec_contratacion_temporal.confirmar_fiscalizacion_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole AND p.prosecdef;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'CT123: falta la confirmación de CT93' USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.strpos(v_antes.definicion, 'ct123') <> 0
       OR pg_catalog.length(v_antes.definicion)
          - pg_catalog.length(pg_catalog.replace(v_antes.definicion, v_ancla, ''))
          <> pg_catalog.length(v_ancla) THEN
        RAISE EXCEPTION 'CT123: preimagen de la confirmación de CT93 incompatible' USING ERRCODE = '55000';
    END IF;
    v_definicion := pg_catalog.replace(v_antes.definicion, v_ancla, v_ancla || v_insercion);
    EXECUTE v_definicion;
    SELECT pg_catalog.pg_get_functiondef(p.oid) AS definicion, p.proacl AS acl,
           p.proowner AS propietario, p.proconfig AS configuracion, p.prosecdef AS definidor
      INTO STRICT v_despues FROM pg_catalog.pg_proc p WHERE p.oid = v_antes.oid;
    IF v_despues.definicion IS DISTINCT FROM v_definicion
       OR pg_catalog.replace(v_despues.definicion, v_ancla || v_insercion, v_ancla) IS DISTINCT FROM v_antes.definicion
       OR v_despues.acl IS DISTINCT FROM v_antes.acl
       OR v_despues.propietario IS DISTINCT FROM v_antes.propietario
       OR v_despues.configuracion IS DISTINCT FROM v_antes.configuracion
       OR v_despues.definidor IS NOT TRUE THEN
        RAISE EXCEPTION 'CT123: confirmación de CT93 alterada' USING ERRCODE = '55000';
    END IF;
END
$refiscalizacion$;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text),
    vec_contratacion_temporal.informe_nuevo_exigido_ct123()
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text)
TO vec_contratacion_temporal_gobernador;

REVOKE ALL ON FUNCTION
    vec_contratacion_temporal.informe_nuevo_admisible_ct123(jsonb),
    vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(text,text)
FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
    vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb),
    vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(
        jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
    vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(text,text)
TO vec_contratacion_temporal_ejecutor;
COMMIT;
