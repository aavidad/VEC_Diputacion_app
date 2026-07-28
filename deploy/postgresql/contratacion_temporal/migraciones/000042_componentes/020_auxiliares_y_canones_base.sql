-- CT-000042: auxiliares puros y cánones de cuadro, material y recibo.
CREATE FUNCTION vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
    p_valor bytea
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_longitud text := pg_catalog.octet_length(p_valor)::text;
BEGIN
    IF pg_catalog.octet_length(p_valor)
       + pg_catalog.octet_length(v_longitud) + 2 > 262144 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'valor canónico RRHH demasiado grande';
    END IF;
    RETURN vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
        v_longitud || ':'
    )
        || p_valor || '\x0a'::bytea;
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
    p_resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_canon bytea := ''::bytea;
    v_campo text;
    v_campos text[];
BEGIN
    IF p_resumen.expediente_ref IS NULL
       OR p_resumen.expediente_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.organizacion_ref IS NULL
       OR p_resumen.organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.numero_visible IS NULL
       OR p_resumen.numero_visible !~
          '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR p_resumen.version IS NULL
       OR p_resumen.version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_resumen.flujo_ref IS NULL
       OR p_resumen.flujo_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.flujo_version IS NULL
       OR p_resumen.flujo_version
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_resumen.flujo_huella_sha256 IS NULL
       OR p_resumen.flujo_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_resumen.flujo_huella_sha256 = pg_catalog.repeat('0', 64)
       OR p_resumen.fase_clave IS NULL
       OR p_resumen.fase_clave !~ '^[a-z][a-z0-9._-]{1,79}$'
       OR p_resumen.estado_clave IS NULL
       OR p_resumen.estado_clave NOT IN (
           'pendiente', 'en_curso', 'espera_externa',
           'completado', 'incidencia', 'cancelado'
       )
       OR p_resumen.centro_ref IS NULL
       OR p_resumen.centro_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.categoria_ref IS NULL
       OR p_resumen.categoria_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.modalidad_clave IS NULL
       OR p_resumen.modalidad_clave !~
          '^$|^[a-z][a-z0-9._-]{1,79}$'
       OR p_resumen.unidad_ref IS NULL
       OR p_resumen.unidad_ref !~
          '^$|^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_resumen.creado_en IS NULL
       OR p_resumen.actualizado_en IS NULL
       OR p_resumen.actualizado_en < p_resumen.creado_en THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'resumen canónico RRHH inválido';
    END IF;
    v_campos := ARRAY[
        p_resumen.expediente_ref, p_resumen.organizacion_ref,
        p_resumen.numero_visible, p_resumen.version::text,
        p_resumen.flujo_ref, p_resumen.flujo_version::text,
        p_resumen.flujo_huella_sha256, p_resumen.fase_clave,
        p_resumen.estado_clave, p_resumen.centro_ref,
        p_resumen.categoria_ref, p_resumen.modalidad_clave,
        p_resumen.unidad_ref,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_resumen.creado_en
        ),
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_resumen.actualizado_en
        )
    ];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'resumen canónico RRHH inválido';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.canon_resultado_consulta_rrhh_puro_v1(
    p_evidencia vec_contratacion_temporal.evidencia_resultado_rrhh_v1
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_canon bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'VEC-CT-RESULTADO-CONSULTA-RRHH-V1'
            || pg_catalog.chr(10)
    );
    v_campo text;
BEGIN
    IF p_evidencia.tipo_consulta IS NULL
       OR p_evidencia.generada_en IS NULL
       OR p_evidencia.total IS NULL
       OR p_evidencia.contenido_huella_sha256 IS NULL
       OR p_evidencia.contenido_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_evidencia.contenido_huella_sha256 = pg_catalog.repeat('0', 64)
       OR p_evidencia.cursor_huella_sha256 IS NULL
       OR NOT (
           p_evidencia.tipo_consulta = 'detalle'
           AND p_evidencia.total = 1
           AND p_evidencia.cursor_huella_sha256 = ''
           OR p_evidencia.tipo_consulta = 'cuadro'
              AND p_evidencia.total BETWEEN 0 AND 100
              AND (
                  p_evidencia.cursor_huella_sha256 = ''
                  OR p_evidencia.total > 0
                     AND p_evidencia.cursor_huella_sha256 ~
                         '^[0-9a-f]{64}$'
                     AND p_evidencia.cursor_huella_sha256 <>
                         pg_catalog.repeat('0', 64)
              )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'resultado canónico RRHH inválido';
    END IF;
    FOREACH v_campo IN ARRAY ARRAY[
        p_evidencia.tipo_consulta,
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_evidencia.generada_en
        ),
        p_evidencia.total::text,
        p_evidencia.contenido_huella_sha256,
        p_evidencia.cursor_huella_sha256
    ]::text[] LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'resultado canónico RRHH inválido';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.canon_recibo_lectura_rrhh_v2(
    p_evidencia
        vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_canon bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'VEC-CT-RECIBO-LECTURA-RRHH-V2' || pg_catalog.chr(10)
    );
    v_campos text[];
    v_campo text;
    v_registrada_texto text;
    v_generada_texto text;
    v_acceso_esperado text;
    v_es_cuadro boolean;
    v_es_detalle boolean;
    v_huellas_no_nulas text[];
BEGIN
    v_acceso_esperado := 'acceso:rrhh:' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                'acceso:rrhh:' || p_evidencia.consumo_vec_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    v_es_cuadro :=
        p_evidencia.accion =
            'contratacion_temporal.cuadro.consultar'
        AND p_evidencia.finalidad =
            'gestion_operativa_contratacion_temporal'
        AND p_evidencia.expediente_ref = ''
        AND p_evidencia.version_expediente = 0
        AND p_evidencia.total BETWEEN 0 AND 100
        AND p_evidencia.alcance_huella_sha256 ~ '^[0-9a-f]{64}$'
        AND p_evidencia.alcance_huella_sha256 <> pg_catalog.repeat('0', 64)
        AND (
            p_evidencia.cursor_huella_sha256 = ''
            OR (
                p_evidencia.total > 0
                AND p_evidencia.cursor_huella_sha256 ~ '^[0-9a-f]{64}$'
                AND p_evidencia.cursor_huella_sha256 <>
                    pg_catalog.repeat('0', 64)
            )
        );
    v_es_detalle :=
        p_evidencia.accion =
            'contratacion_temporal.expediente.consultar'
        AND p_evidencia.finalidad =
            'tramitacion_expediente_contratacion_temporal'
        AND p_evidencia.expediente_ref ~
            '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
        AND p_evidencia.version_expediente
            BETWEEN 1 AND 9007199254740991::numeric
        AND p_evidencia.total = 1
        AND p_evidencia.alcance_huella_sha256 = ''
        AND p_evidencia.cursor_huella_sha256 = '';
    v_huellas_no_nulas := ARRAY[
        p_evidencia.huella_sha256,
        p_evidencia.vinculo_identidad_huella_sha256,
        p_evidencia.auditoria_vec_huella_sha256,
        p_evidencia.consumo_vec_huella_sha256,
        p_evidencia.decision_huella_sha256,
        p_evidencia.capacidad_huella_sha256,
        p_evidencia.material_huella_sha256,
        p_evidencia.consulta_huella_sha256,
        p_evidencia.autenticacion_huella_sha256,
        p_evidencia.control_sesion_huella_sha256,
        p_evidencia.contenido_huella_sha256,
        p_evidencia.resultado_huella_sha256
    ];
    FOREACH v_campo IN ARRAY v_huellas_no_nulas LOOP
        IF v_campo IS NULL OR v_campo !~ '^[0-9a-f]{64}$'
           OR v_campo = pg_catalog.repeat('0', 64) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'recibo de lectura RRHH inválido';
        END IF;
    END LOOP;
    IF p_evidencia.esquema IS DISTINCT FROM
       'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2'
       OR p_evidencia.acceso_ref IS NULL
       OR p_evidencia.acceso_ref !~ '^acceso:rrhh:[0-9a-f]{32}$'
       OR p_evidencia.acceso_ref <> v_acceso_esperado
       OR p_evidencia.secuencia IS NULL
       OR p_evidencia.secuencia <> pg_catalog.trunc(p_evidencia.secuencia)
       OR p_evidencia.secuencia
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_evidencia.anterior_sha256 IS NULL
       OR p_evidencia.anterior_sha256 !~ '^[0-9a-f]{64}$'
       OR (
           p_evidencia.secuencia = 1
           AND p_evidencia.anterior_sha256 <> pg_catalog.repeat('0', 64)
       ) OR (
           p_evidencia.secuencia > 1
           AND p_evidencia.anterior_sha256 = pg_catalog.repeat('0', 64)
       )
       OR p_evidencia.registrada_en IS NULL
       OR p_evidencia.generada_en IS NULL
       OR p_evidencia.registrada_en < p_evidencia.generada_en
       OR p_evidencia.auditoria_vec_ref IS NULL
       OR p_evidencia.auditoria_vec_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.decision_ref IS NULL
       OR p_evidencia.decision_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.decision_huella_sha256 IS NULL
       OR p_evidencia.decision_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_evidencia.capacidad_huella_sha256 IS NULL
       OR p_evidencia.capacidad_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_evidencia.material_huella_sha256 IS NULL
       OR p_evidencia.material_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_evidencia.consulta_huella_sha256 IS NULL
       OR p_evidencia.consulta_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_evidencia.correlacion_ref IS NULL
       OR p_evidencia.correlacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.autenticacion_ref IS NULL
       OR p_evidencia.autenticacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.sesion_ref IS NULL
       OR p_evidencia.sesion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.control_sesion_ref IS NULL
       OR p_evidencia.control_sesion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.control_sesion_revision IS NULL
       OR p_evidencia.control_sesion_revision <>
          pg_catalog.trunc(p_evidencia.control_sesion_revision)
       OR p_evidencia.control_sesion_revision
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_evidencia.actor_ref IS NULL
       OR p_evidencia.actor_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.perfil_ref IS NULL
       OR p_evidencia.perfil_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.perfil_version IS NULL
       OR p_evidencia.perfil_version <>
          pg_catalog.trunc(p_evidencia.perfil_version)
       OR p_evidencia.perfil_version
          NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_evidencia.organizacion_ref IS NULL
       OR p_evidencia.organizacion_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_evidencia.clase_ambito IS NULL
       OR p_evidencia.clase_ambito NOT IN (
           'organizacion', 'centro', 'unidad_gestion'
       )
       OR p_evidencia.ambito_ref IS NULL
       OR p_evidencia.ambito_ref !~
          '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR (
           p_evidencia.clase_ambito = 'organizacion'
           AND p_evidencia.ambito_ref <> p_evidencia.organizacion_ref
       )
       OR p_evidencia.accion IS NULL
       OR p_evidencia.finalidad IS NULL
       OR p_evidencia.expediente_ref IS NULL
       OR p_evidencia.version_expediente IS NULL
       OR p_evidencia.version_expediente <>
          pg_catalog.trunc(p_evidencia.version_expediente)
       OR p_evidencia.total IS NULL
       OR p_evidencia.alcance_huella_sha256 IS NULL
       OR p_evidencia.cursor_huella_sha256 IS NULL
       OR (NOT v_es_cuadro AND NOT v_es_detalle) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'recibo de lectura RRHH inválido';
    END IF;
    v_registrada_texto :=
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_evidencia.registrada_en
        );
    v_generada_texto :=
        vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
            p_evidencia.generada_en
        );
    v_campos := ARRAY[
        p_evidencia.esquema, p_evidencia.acceso_ref,
        p_evidencia.secuencia::text, p_evidencia.anterior_sha256,
        p_evidencia.huella_sha256,
        p_evidencia.vinculo_identidad_huella_sha256,
        p_evidencia.alcance_huella_sha256, v_registrada_texto,
        p_evidencia.auditoria_vec_ref,
        p_evidencia.auditoria_vec_huella_sha256,
        p_evidencia.consumo_vec_huella_sha256,
        p_evidencia.decision_ref, p_evidencia.decision_huella_sha256,
        p_evidencia.capacidad_huella_sha256,
        p_evidencia.material_huella_sha256,
        p_evidencia.consulta_huella_sha256,
        p_evidencia.correlacion_ref, p_evidencia.autenticacion_ref,
        p_evidencia.autenticacion_huella_sha256,
        p_evidencia.sesion_ref, p_evidencia.control_sesion_ref,
        p_evidencia.control_sesion_revision::text,
        p_evidencia.control_sesion_huella_sha256,
        p_evidencia.actor_ref, p_evidencia.perfil_ref,
        p_evidencia.perfil_version::text, p_evidencia.organizacion_ref,
        p_evidencia.clase_ambito, p_evidencia.ambito_ref,
        p_evidencia.accion, p_evidencia.finalidad,
        p_evidencia.expediente_ref,
        p_evidencia.version_expediente::text,
        p_evidencia.total::text, p_evidencia.contenido_huella_sha256,
        p_evidencia.resultado_huella_sha256,
        p_evidencia.cursor_huella_sha256, v_generada_texto
    ];
    IF pg_catalog.cardinality(v_campos) <> 38 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'recibo de lectura RRHH inválido';
    END IF;
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_canon := v_canon
            || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
                vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                    v_campo
                )
            );
    END LOOP;
    IF pg_catalog.octet_length(v_canon) > 262144 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'recibo de lectura RRHH inválido';
    END IF;
    RETURN v_canon;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'recibo de lectura RRHH inválido';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.huella_material_consumo_rrhh_v3(
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_capacidad jsonb;
    v_campos text[];
    v_claves text[] := ARRAY[
        'decision_ref', 'huella_decision_sha256',
        'huella_motivo_sha256', 'contexto_ref',
        'huella_contexto_sha256', 'operacion', 'efecto_ref',
        'huella_efecto_sha256', 'audiencia_consumo',
        'emitida_en', 'expira_en'
    ]::text[];
    v_campo text;
    v_bloques bytea[];
    v_bloque bytea;
    v_preimagen bytea := ''::bytea;
    v_indice integer;
    v_codigo integer;
    v_emitida_texto text;
    v_expira_texto text;
    v_emitida timestamp;
    v_expira timestamp;
    v_fraccion text;
    v_hora integer;
    v_minuto integer;
    v_segundo integer;
BEGIN
    IF pg_catalog.octet_length(p_capacidad_canonica)
       NOT BETWEEN 512 AND 32768
       OR pg_catalog.octet_length(p_decision_canonica)
          NOT BETWEEN 1 AND 524288
       OR pg_catalog.octet_length(p_motivo_canonico)
          NOT BETWEEN 1 AND 65536
       OR pg_catalog.octet_length(p_contexto_actor_canonico)
          NOT BETWEEN 1 AND 262144
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR pg_catalog.octet_length(p_payload_vec_ad_3)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_sobre_cose_sign_1)
          NOT BETWEEN 1 AND 1048576
       OR pg_catalog.octet_length(p_evidencia_verificacion)
          NOT BETWEEN 1 AND 262144
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44
       OR substring(p_raiz_publica_spki FROM 1 FOR 12)
          <> pg_catalog.decode('302a300506032b6570032100', 'hex') THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;

    v_capacidad :=
        vec_contratacion_temporal.decodificar_texto_utf8_rrhh_v1(
            p_capacidad_canonica
        )::jsonb;
    IF pg_catalog.jsonb_typeof(v_capacidad) <> 'object' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;
    v_campos := ARRAY[
        v_capacidad ->> 'decision_ref',
        v_capacidad ->> 'huella_decision_sha256',
        v_capacidad ->> 'huella_motivo_sha256',
        v_capacidad ->> 'contexto_ref',
        v_capacidad ->> 'huella_contexto_sha256',
        v_capacidad ->> 'operacion',
        v_capacidad ->> 'efecto_ref',
        v_capacidad ->> 'huella_efecto_sha256',
        v_capacidad ->> 'audiencia_consumo',
        v_capacidad ->> 'emitida_en',
        v_capacidad ->> 'expira_en'
    ];
    IF pg_catalog.cardinality(v_claves) <> 11
       OR pg_catalog.cardinality(v_campos) <> 11
       OR pg_catalog.array_position(v_campos, NULL) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;
    FOR v_indice IN 1..11 LOOP
        IF pg_catalog.jsonb_typeof(
            v_capacidad -> v_claves[v_indice]
        ) IS DISTINCT FROM 'string' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'material de consumo RRHH inválido';
        END IF;
    END LOOP;
    FOR v_indice IN 1..9 LOOP
        v_campo := v_campos[v_indice];
        IF v_campo = '' OR pg_catalog.octet_length(v_campo) > 512 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'material de consumo RRHH inválido';
        END IF;
        FOR v_codigo IN 1..pg_catalog.char_length(v_campo) LOOP
            IF pg_catalog.ascii(substring(
                v_campo FROM v_codigo FOR 1
            )) BETWEEN 0 AND 31
               OR pg_catalog.ascii(substring(
                   v_campo FROM v_codigo FOR 1
               )) BETWEEN 127 AND 159
               OR pg_catalog.ascii(substring(
                   v_campo FROM v_codigo FOR 1
               )) = ANY(ARRAY[
                   32, 133, 160, 5760, 8192, 8193, 8194, 8195, 8196,
                   8197, 8198, 8199, 8200, 8201, 8202, 8232, 8233,
                   8239, 8287, 12288
               ])
               OR pg_catalog.ascii(substring(
                   v_campo FROM v_codigo FOR 1
               )) = 42 THEN
                RAISE EXCEPTION USING ERRCODE = '22023',
                    MESSAGE = 'material de consumo RRHH inválido';
            END IF;
        END LOOP;
    END LOOP;
    FOREACH v_indice IN ARRAY ARRAY[2, 3, 5, 8]::integer[] LOOP
        IF v_campos[v_indice] !~ '^[0-9a-f]{64}$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'material de consumo RRHH inválido';
        END IF;
    END LOOP;

    v_emitida_texto := v_campos[10];
    v_expira_texto := v_campos[11];
    IF v_emitida_texto !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{0,5}[1-9])?Z$'
       OR v_expira_texto !~
       '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{0,5}[1-9])?Z$' THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;
    v_hora := substring(v_emitida_texto FROM 12 FOR 2)::integer;
    v_minuto := substring(v_emitida_texto FROM 15 FOR 2)::integer;
    v_segundo := substring(v_emitida_texto FROM 18 FOR 2)::integer;
    IF substring(v_emitida_texto FROM 1 FOR 4)::integer NOT BETWEEN 1 AND 9999
       OR v_hora NOT BETWEEN 0 AND 23
       OR v_minuto NOT BETWEEN 0 AND 59
       OR v_segundo NOT BETWEEN 0 AND 59 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;
    v_fraccion := CASE WHEN pg_catalog.strpos(v_emitida_texto, '.') = 0
        THEN '' ELSE pg_catalog.split_part(
            pg_catalog.split_part(v_emitida_texto, '.', 2), 'Z', 1
        ) END;
    v_emitida := pg_catalog.make_timestamp(
        substring(v_emitida_texto FROM 1 FOR 4)::integer,
        substring(v_emitida_texto FROM 6 FOR 2)::integer,
        substring(v_emitida_texto FROM 9 FOR 2)::integer,
        substring(v_emitida_texto FROM 12 FOR 2)::integer,
        substring(v_emitida_texto FROM 15 FOR 2)::integer,
        substring(v_emitida_texto FROM 18 FOR 2)::double precision
        + CASE WHEN v_fraccion = '' THEN 0 ELSE
          ('0.' || v_fraccion)::double precision END
    );
    v_hora := substring(v_expira_texto FROM 12 FOR 2)::integer;
    v_minuto := substring(v_expira_texto FROM 15 FOR 2)::integer;
    v_segundo := substring(v_expira_texto FROM 18 FOR 2)::integer;
    IF substring(v_expira_texto FROM 1 FOR 4)::integer NOT BETWEEN 1 AND 9999
       OR v_hora NOT BETWEEN 0 AND 23
       OR v_minuto NOT BETWEEN 0 AND 59
       OR v_segundo NOT BETWEEN 0 AND 59 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;
    v_fraccion := CASE WHEN pg_catalog.strpos(v_expira_texto, '.') = 0
        THEN '' ELSE pg_catalog.split_part(
            pg_catalog.split_part(v_expira_texto, '.', 2), 'Z', 1
        ) END;
    v_expira := pg_catalog.make_timestamp(
        substring(v_expira_texto FROM 1 FOR 4)::integer,
        substring(v_expira_texto FROM 6 FOR 2)::integer,
        substring(v_expira_texto FROM 9 FOR 2)::integer,
        substring(v_expira_texto FROM 12 FOR 2)::integer,
        substring(v_expira_texto FROM 15 FOR 2)::integer,
        substring(v_expira_texto FROM 18 FOR 2)::double precision
        + CASE WHEN v_fraccion = '' THEN 0 ELSE
          ('0.' || v_fraccion)::double precision END
    );
    IF v_expira <= v_emitida
       OR v_expira - v_emitida > pg_catalog.make_interval(secs => 5) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;

    v_bloques := ARRAY[p_capacidad_canonica];
    FOREACH v_campo IN ARRAY v_campos LOOP
        v_bloques := pg_catalog.array_append(
            v_bloques,
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                v_campo
            )
        );
    END LOOP;
    v_bloques := v_bloques || ARRAY[
        p_decision_canonica, p_motivo_canonico,
        p_contexto_actor_canonico,
        pg_catalog.int8send(p_persona_version::bigint),
        pg_catalog.int8send(p_perfil_version::bigint),
        p_payload_vec_ad_3, p_sobre_cose_sign_1,
        p_evidencia_verificacion, p_raiz_publica_spki
    ]::bytea[];
    FOREACH v_bloque IN ARRAY v_bloques LOOP
        v_preimagen := v_preimagen
            || pg_catalog.int8send(
                pg_catalog.octet_length(v_bloque)::bigint
            )
            || v_bloque;
    END LOOP;
    IF pg_catalog.cardinality(v_bloques) <> 21 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'material de consumo RRHH inválido';
    END IF;
    RETURN pg_catalog.encode(pg_catalog.sha256(v_preimagen), 'hex');
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION USING ERRCODE = '22023',
        MESSAGE = 'material de consumo RRHH inválido';
END
$funcion$;

CREATE FUNCTION
vec_contratacion_temporal.canon_contenido_cuadro_rrhh_v1(
    p_generada_en timestamptz,
    p_resumenes vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    p_hay_mas boolean,
    p_cursor_huella bytea
)
RETURNS bytea
LANGUAGE plpgsql
IMMUTABLE
STRICT
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_canon bytea :=
        vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
            'VEC-CT-CONTENIDO-CUADRO-RRHH-V1' || pg_catalog.chr(10)
    );
    v_indice integer;
    v_anterior vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_actual vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
    v_vistos text[] := ARRAY[]::text[];
BEGIN
    PERFORM vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
        p_generada_en
    );
    IF pg_catalog.cardinality(p_resumenes) > 100
       OR (
           pg_catalog.cardinality(p_resumenes) = 0
           AND pg_catalog.array_ndims(p_resumenes) IS NOT NULL
       )
       OR (
           pg_catalog.cardinality(p_resumenes) > 0
           AND (
               pg_catalog.array_ndims(p_resumenes) <> 1
               OR pg_catalog.array_lower(p_resumenes, 1) <> 1
           )
       )
       OR (p_hay_mas AND pg_catalog.cardinality(p_resumenes) = 0)
       OR (p_hay_mas AND pg_catalog.octet_length(p_cursor_huella) <> 32)
       OR (
           p_hay_mas
           AND p_cursor_huella = pg_catalog.decode(
               pg_catalog.repeat('00', 32), 'hex'
           )
       )
       OR (
           NOT p_hay_mas
           AND pg_catalog.octet_length(p_cursor_huella) <> 0
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de cuadro RRHH inválido';
    END IF;
    IF pg_catalog.cardinality(p_resumenes) > 0
       AND pg_catalog.array_position(p_resumenes, NULL) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de cuadro RRHH inválido';
    END IF;
    v_canon := v_canon
        || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                vec_contratacion_temporal.texto_instante_canonico_rrhh_v1(
                    p_generada_en
                )
            )
        )
        || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                pg_catalog.cardinality(p_resumenes)::text
            )
        );

    FOR v_indice IN 1..pg_catalog.cardinality(p_resumenes) LOOP
        v_actual := p_resumenes[v_indice];
        v_canon := v_canon
            || vec_contratacion_temporal.canon_resumen_publicacion_rrhh_v1(
                v_actual
            );
        IF v_actual.actualizado_en > p_generada_en
           OR v_actual.expediente_ref = ANY(v_vistos)
           OR (
               v_indice > 1
               AND (
                   v_anterior.actualizado_en < v_actual.actualizado_en
                   OR (
                       v_anterior.actualizado_en = v_actual.actualizado_en
                       AND v_anterior.expediente_ref COLLATE "C"
                           <= v_actual.expediente_ref COLLATE "C"
                   )
               )
           ) THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de cuadro RRHH inválido';
        END IF;
        IF pg_catalog.octet_length(v_canon) > 262144 THEN
            RAISE EXCEPTION USING ERRCODE = '22023',
                MESSAGE = 'contenido de cuadro RRHH inválido';
        END IF;
        v_vistos := pg_catalog.array_append(
            v_vistos, v_actual.expediente_ref
        );
        v_anterior := v_actual;
    END LOOP;
    v_canon := v_canon
        || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            vec_contratacion_temporal.codificar_texto_utf8_rrhh_v1(
                CASE WHEN p_hay_mas THEN '1' ELSE '0' END
            )
        )
        || vec_contratacion_temporal.encuadrar_valor_rrhh_v1(
            p_cursor_huella
        );
    IF pg_catalog.octet_length(v_canon) > 262144 THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de cuadro RRHH inválido';
    END IF;
    RETURN v_canon;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR numeric_value_out_of_range OR array_subscript_error THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'contenido de cuadro RRHH inválido';
END
$funcion$;
