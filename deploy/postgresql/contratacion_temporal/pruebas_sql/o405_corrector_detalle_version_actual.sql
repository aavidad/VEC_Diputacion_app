\set ON_ERROR_STOP on

-- Ajusta únicamente el selector de versión de un vector sintético de detalle.
-- Cuando p_recalcular_vec es cierto se vuelven a sellar la decisión, el
-- contexto de recurso y la capacidad. El canon conserva literalmente la
-- versión solicitada; nunca se sustituye cero por la versión materializada.
CREATE FUNCTION public.ajustar_version_observada_ct43a(
    p_caso text,
    p_version_observada numeric,
    p_recalcular_vec boolean
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_cierre public.vectores_cierre_ct43%ROWTYPE;
    v_vector public.vectores_consulta_rrhh_v3%ROWTYPE;
    v_contexto
        vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2;
    v_consulta
        vec_contratacion_temporal.consulta_detalle_rrhh_v1;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_decision jsonb;
    v_decision_canonica bytea;
    v_capacidad jsonb;
    v_clave record;
BEGIN
    IF p_caso IS NULL
       OR p_version_observada IS NULL
       OR p_version_observada NOT BETWEEN
          0 AND 9007199254740991::numeric
       OR p_version_observada <> pg_catalog.trunc(p_version_observada)
       OR p_recalcular_vec IS NULL THEN
        RAISE EXCEPTION 'ajuste CT43A inválido';
    END IF;
    SELECT *
      INTO STRICT v_cierre
      FROM public.vectores_cierre_ct43
     WHERE caso = p_caso AND perfil = 'detalle';
    SELECT *
      INTO STRICT v_vector
      FROM public.vectores_consulta_rrhh_v3
     WHERE caso = p_caso;

    v_consulta := ROW(
        ((v_cierre.contexto).consulta_detalle).expediente_ref,
        p_version_observada
    )::vec_contratacion_temporal.consulta_detalle_rrhh_v1;
    v_contexto := ROW(
        (v_cierre.contexto).organizacion_ref,
        (v_cierre.contexto).clase_ambito,
        (v_cierre.contexto).ambito_ref,
        NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
        v_consulta,
        (v_cierre.contexto).familia_ref
    )::vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2;

    IF p_recalcular_vec THEN
        v_consulta_canonica :=
            vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
                v_consulta
            );
        v_contexto_recurso := pg_catalog.convert_to(
            '{"ambitos":{"ambito_ref":"' || v_contexto.ambito_ref
            || '","clase_ambito":"' || v_contexto.clase_ambito
            || '","organizacion_ref":"' || v_contexto.organizacion_ref
            || '"},"atributos":{"consulta_dominio":"'
            || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
            || '","consulta_huella_sha256":"'
            || pg_catalog.encode(
                pg_catalog.sha256(v_consulta_canonica), 'hex'
            ) || '"}}', 'UTF8'
        );
        v_contexto_huella := pg_catalog.encode(
            pg_catalog.sha256(v_contexto_recurso), 'hex'
        );

        v_decision := pg_catalog.convert_from(
            v_vector.decision, 'UTF8'
        )::jsonb;
        v_decision := pg_catalog.jsonb_set(
            v_decision, '{contexto_recurso_huella_sha256}',
            pg_catalog.to_jsonb(v_contexto_huella), false
        );
        v_decision_canonica :=
            vec_autorizacion.decision_contexto_actor_v3_canonica(
                v_decision
            );

        v_capacidad := pg_catalog.convert_from(
            v_vector.capacidad, 'UTF8'
        )::jsonb;
        v_capacidad := pg_catalog.jsonb_set(
            v_capacidad, '{huella_decision_sha256}',
            pg_catalog.to_jsonb(pg_catalog.encode(
                pg_catalog.sha256(v_decision_canonica), 'hex'
            )), false
        );
        v_capacidad := pg_catalog.jsonb_set(
            v_capacidad, '{huella_efecto_sha256}',
            pg_catalog.to_jsonb(v_contexto_huella), false
        );
        v_capacidad := pg_catalog.jsonb_set(
            v_capacidad, '{mac_sha256}',
            pg_catalog.to_jsonb(pg_catalog.repeat('f', 64)), false
        );
        SELECT *
          INTO STRICT v_clave
          FROM vec_autorizacion_atestada_v3.clave_capacidad_version
         WHERE clave_id = v_capacidad ->> 'clave_id'
           AND version =
               (v_capacidad ->> 'clave_version')::numeric;
        v_capacidad := pg_catalog.jsonb_set(
            v_capacidad, '{mac_sha256}',
            pg_catalog.to_jsonb(pg_catalog.encode(public.hmac(
                vec_autorizacion_atestada_v3.preimagen_mac(v_capacidad),
                v_clave.secreto_hmac, 'sha256'
            ), 'hex')), false
        );
        UPDATE public.vectores_consulta_rrhh_v3
           SET capacidad =
                   vec_autorizacion_atestada_v3.capacidad_canonica(
                       v_capacidad
                   ),
               decision = v_decision_canonica
         WHERE caso = p_caso;
    END IF;

    UPDATE public.vectores_cierre_ct43
       SET contexto = v_contexto
     WHERE caso = p_caso;
END
$funcion$;
REVOKE ALL ON FUNCTION
    public.ajustar_version_observada_ct43a(text, numeric, boolean)
FROM PUBLIC;

-- Acredita que solicitud, prueba durable, registro de acceso y Recibo no
-- confunden «versión actual» con la versión cero. El hash del acceso procede
-- del canon con cero; las filas durables conservan la versión positiva leída.
CREATE FUNCTION public.comprobar_version_actual_ct43a(p_caso text)
RETURNS void
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_cierre public.vectores_cierre_ct43%ROWTYPE;
    v_canon bytea;
    v_huella text;
    v_prueba record;
BEGIN
    SELECT *
      INTO STRICT v_cierre
      FROM public.vectores_cierre_ct43
     WHERE caso = p_caso AND perfil = 'detalle';
    v_canon :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(
            (v_cierre.contexto).consulta_detalle
        );
    v_huella := pg_catalog.encode(
        pg_catalog.sha256(v_canon), 'hex'
    );
    SELECT prueba.version_expediente,
           prueba.version_expediente_prueba_v2,
           prueba.consulta_huella_sha256,
           acceso.version_expediente AS version_acceso,
           acceso.version_expediente_prueba_v2 AS version_prueba_acceso,
           pg_catalog.octet_length(prueba.recibo_canonico) AS recibo_octetos
      INTO STRICT v_prueba
      FROM vec_contratacion_temporal
           .prueba_resultado_recibo_rrhh_v2 prueba
      JOIN vec_contratacion_temporal.registro_acceso_rrhh acceso
        USING (acceso_ref)
     WHERE prueba.decision_ref = 'decision:consulta-rrhh:' || p_caso;

    IF pg_catalog.convert_from(v_canon, 'UTF8') IS DISTINCT FROM
       '{"dominio":"vec.contratacion_temporal.consulta_rrhh.detalle.v1",'
       || '"version":1,"expediente_ref":"expediente:rrhh:minimizado",'
       || '"version_observada":0}'
       OR v_prueba.consulta_huella_sha256 IS DISTINCT FROM v_huella
       OR v_prueba.version_expediente IS DISTINCT FROM 1::numeric
       OR v_prueba.version_expediente_prueba_v2
          IS DISTINCT FROM 1::numeric
       OR v_prueba.version_acceso IS DISTINCT FROM 1::numeric
       OR v_prueba.version_prueba_acceso IS DISTINCT FROM 1::numeric
       OR v_prueba.recibo_octetos <= 0 THEN
        RAISE EXCEPTION 'prueba de versión actual CT43A incoherente';
    END IF;
END
$funcion$;
REVOKE ALL ON FUNCTION
    public.comprobar_version_actual_ct43a(text)
FROM PUBLIC;
