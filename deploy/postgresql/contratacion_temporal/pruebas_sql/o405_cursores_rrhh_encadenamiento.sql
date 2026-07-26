\set ON_ERROR_STOP on

CREATE FUNCTION pg_temp.prueba_alcance_rrhh(
    p_acceso text, p_familia text, p_organizacion text, p_clase text,
    p_ambito text, p_actor text, p_perfil text, p_perfil_version numeric,
    p_sesion text, p_sesion_huella text, p_registrado timestamptz
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-ALCANCE-ACCESO-RRHH-V1' || pg_catalog.chr(10),
               'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso)
        || vec_contratacion_temporal.encuadrar_texto_v1('cuadro')
        || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_organizacion)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_clase)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_ambito)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_actor)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_perfil)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_perfil_version::text
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_sesion)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_sesion_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_registrado)
           )
$funcion$;

CREATE FUNCTION pg_temp.prueba_cursor_rrhh(
    p_token text, p_familia text, p_padre text, p_pagina numeric,
    p_padre_emitida timestamptz, p_ultimo_actualizado timestamptz,
    p_ultimo_expediente text, p_familia_creada timestamptz,
    p_familia_valida timestamptz, p_emitida timestamptz, p_acceso text
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-CURSOR-CUADRO-RRHH-V1' || pg_catalog.chr(10),
               'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_token)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               COALESCE(p_padre, '')
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_pagina::text)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               CASE WHEN p_padre_emitida IS NULL THEN ''
                    ELSE vec_contratacion_temporal.instante_utc_v1(
                        p_padre_emitida
                    ) END
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_ultimo_actualizado
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_ultimo_expediente
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_familia_creada
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_familia_valida
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_emitida)
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso)
$funcion$;

CREATE FUNCTION pg_temp.prueba_consumo_rrhh(
    p_token text, p_familia text, p_decision text,
    p_decision_huella text, p_consumo_huella text,
    p_acceso_emision text, p_acceso_consumo text,
    p_cursor_emitida timestamptz, p_familia_valida timestamptz,
    p_consumido timestamptz
)
RETURNS bytea
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog
AS $funcion$
    SELECT pg_catalog.convert_to(
               'VEC-CT-CONSUMO-CURSOR-CUADRO-RRHH-V1'
                   || pg_catalog.chr(10), 'UTF8'
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_token)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_decision)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_decision_huella
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               p_consumo_huella
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso_emision)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_acceso_consumo)
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_cursor_emitida
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(
                   p_familia_valida
               )
           )
        || vec_contratacion_temporal.encuadrar_texto_v1(
               vec_contratacion_temporal.instante_utc_v1(p_consumido)
           )
$funcion$;

CREATE TEMP TABLE respaldo_p2 AS
SELECT
    token_huella_sha256, familia_ref, padre_token_huella_sha256,
    pagina, padre_emitida_en, ultimo_actualizado_en,
    ultimo_expediente_ref, familia_creada_en, familia_valida_hasta,
    emitida_en, acceso_emision_ref, prueba_canonica, prueba_huella_sha256
  FROM vec_contratacion_temporal.cursor_cuadro_rrhh
 WHERE pagina = 2;
CREATE TEMP TABLE respaldo_consumo AS
  TABLE vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;

DO $alcance_hostil$
DECLARE
    v_acceso vec_contratacion_temporal.registro_acceso_rrhh%ROWTYPE;
    v_familia vec_contratacion_temporal.familia_cursor_cuadro_rrhh%ROWTYPE;
    v_prueba bytea;
BEGIN
    SELECT * INTO STRICT v_acceso
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE decision_ref = 'decision:cursor:rrhh:5';
    SELECT * INTO STRICT v_familia
      FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh;
    v_prueba := pg_temp.prueba_alcance_rrhh(
        v_acceso.acceso_ref, v_familia.familia_ref,
        v_acceso.organizacion_ref, v_familia.clase_ambito,
        v_acceso.ambito_ref, v_acceso.actor_ref, v_acceso.perfil_id,
        v_acceso.perfil_version, v_acceso.sesion_id,
        v_acceso.sesion_huella_sha256, v_acceso.registrada_en
    );
    INSERT INTO vec_contratacion_temporal.alcance_acceso_rrhh VALUES (
        v_acceso.acceso_ref, 'cuadro', v_familia.familia_ref,
        v_acceso.organizacion_ref, v_familia.clase_ambito,
        v_acceso.ambito_ref, v_acceso.actor_ref, v_acceso.perfil_id,
        v_acceso.perfil_version, v_acceso.sesion_id,
        v_acceso.sesion_huella_sha256, v_acceso.registrada_en,
        v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
END
$alcance_hostil$;

SET session_replication_role = replica;
DELETE FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
DELETE FROM vec_contratacion_temporal.cursor_cuadro_rrhh;
RESET session_replication_role;

DO $p2_hostil$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_acceso_rrhh%ROWTYPE;
    v_familia vec_contratacion_temporal.familia_cursor_cuadro_rrhh%ROWTYPE;
    v_p2 respaldo_p2%ROWTYPE;
    v_prueba bytea;
    v_token constant text := pg_catalog.repeat('2', 64);
BEGIN
    SELECT * INTO STRICT v_p2 FROM respaldo_p2;
    SELECT * INTO STRICT v_alcance
      FROM vec_contratacion_temporal.alcance_acceso_rrhh
     WHERE acceso_ref = (
         SELECT acceso_ref
           FROM vec_contratacion_temporal.registro_acceso_rrhh
          WHERE decision_ref = 'decision:cursor:rrhh:5'
     );
    SELECT * INTO STRICT v_familia
      FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh
     WHERE familia_ref = v_p2.familia_ref;
    v_prueba := pg_temp.prueba_cursor_rrhh(
        v_token, v_p2.familia_ref, NULL, 2, NULL,
        v_p2.ultimo_actualizado_en, 'expediente:hostil:p2',
        v_p2.familia_creada_en, v_p2.familia_valida_hasta,
        v_alcance.acceso_registrado_en, v_alcance.acceso_ref
    );
    BEGIN
        INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
            token_huella_sha256, familia_ref,
            padre_token_huella_sha256, pagina, padre_emitida_en,
            ultimo_actualizado_en, ultimo_expediente_ref,
            familia_creada_en, familia_valida_hasta, emitida_en,
            acceso_emision_ref, prueba_canonica, prueba_huella_sha256
        ) VALUES (
            v_token, v_p2.familia_ref, NULL, 2, NULL,
            v_p2.ultimo_actualizado_en, 'expediente:hostil:p2',
            v_p2.familia_creada_en, v_p2.familia_valida_hasta,
            v_alcance.acceso_registrado_en, v_alcance.acceso_ref,
            v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
        );
        RAISE EXCEPTION 'p2 aceptó acceso distinto del origen';
    EXCEPTION
        WHEN check_violation OR foreign_key_violation THEN NULL;
    END;
END
$p2_hostil$;

SET session_replication_role = replica;
INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
    token_huella_sha256, familia_ref, padre_token_huella_sha256,
    pagina, padre_emitida_en, ultimo_actualizado_en,
    ultimo_expediente_ref, familia_creada_en, familia_valida_hasta,
    emitida_en, acceso_emision_ref, prueba_canonica, prueba_huella_sha256
) SELECT * FROM respaldo_p2;
INSERT INTO vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
  SELECT * FROM respaldo_consumo;
RESET session_replication_role;

SET session_replication_role = replica;
DELETE FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
RESET session_replication_role;

DO $mismo_acceso$
DECLARE
    v_acceso vec_contratacion_temporal.registro_acceso_rrhh%ROWTYPE;
    v_p2 vec_contratacion_temporal.cursor_cuadro_rrhh%ROWTYPE;
    v_prueba bytea;
BEGIN
    SELECT * INTO STRICT v_p2
      FROM vec_contratacion_temporal.cursor_cuadro_rrhh WHERE pagina = 2;
    SELECT * INTO STRICT v_acceso
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE acceso_ref = v_p2.acceso_emision_ref;
    v_prueba := pg_temp.prueba_consumo_rrhh(
        v_p2.token_huella_sha256, v_p2.familia_ref,
        v_acceso.decision_ref, v_acceso.decision_huella_sha256,
        v_acceso.consumo_vec_huella_sha256, v_acceso.acceso_ref,
        v_acceso.acceso_ref, v_p2.emitida_en, v_p2.familia_valida_hasta,
        v_acceso.registrada_en
    );
    BEGIN
        INSERT INTO vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
        VALUES (
            v_p2.token_huella_sha256, v_p2.familia_ref,
            v_acceso.decision_ref, v_acceso.decision_huella_sha256,
            v_acceso.consumo_vec_huella_sha256, v_acceso.acceso_ref,
            v_acceso.acceso_ref, v_p2.emitida_en,
            v_p2.familia_valida_hasta, v_acceso.registrada_en,
            v_prueba, pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
        );
        RAISE EXCEPTION 'consumo aceptó el acceso de emisión';
    EXCEPTION WHEN check_violation THEN NULL;
    END;
END
$mismo_acceso$;

SET session_replication_role = replica;
INSERT INTO vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
  SELECT * FROM respaldo_consumo;
RESET session_replication_role;

DO $p3$
DECLARE
    v_alcance vec_contratacion_temporal.alcance_acceso_rrhh%ROWTYPE;
    v_consumo vec_contratacion_temporal.consumo_cursor_cuadro_rrhh%ROWTYPE;
    v_p2 vec_contratacion_temporal.cursor_cuadro_rrhh%ROWTYPE;
    v_prueba bytea;
    v_token_hostil constant text := pg_catalog.repeat('3', 64);
    v_token_valido constant text := pg_catalog.repeat('4', 64);
BEGIN
    SELECT * INTO STRICT v_p2
      FROM vec_contratacion_temporal.cursor_cuadro_rrhh WHERE pagina = 2;
    SELECT * INTO STRICT v_consumo
      FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
     WHERE token_huella_sha256 = v_p2.token_huella_sha256;
    SELECT * INTO STRICT v_alcance
      FROM vec_contratacion_temporal.alcance_acceso_rrhh
     WHERE acceso_ref = (
         SELECT acceso_ref
           FROM vec_contratacion_temporal.registro_acceso_rrhh
          WHERE decision_ref = 'decision:cursor:rrhh:5'
     );
    v_prueba := pg_temp.prueba_cursor_rrhh(
        v_token_hostil, v_p2.familia_ref, v_p2.token_huella_sha256,
        3, v_p2.emitida_en, v_p2.ultimo_actualizado_en,
        'expediente:hostil:p3', v_p2.familia_creada_en,
        v_p2.familia_valida_hasta, v_alcance.acceso_registrado_en,
        v_alcance.acceso_ref
    );
    BEGIN
        INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
            token_huella_sha256, familia_ref,
            padre_token_huella_sha256, pagina, padre_emitida_en,
            ultimo_actualizado_en, ultimo_expediente_ref,
            familia_creada_en, familia_valida_hasta, emitida_en,
            acceso_emision_ref, prueba_canonica, prueba_huella_sha256
        ) VALUES (
            v_token_hostil, v_p2.familia_ref, v_p2.token_huella_sha256,
            3, v_p2.emitida_en, v_p2.ultimo_actualizado_en,
            'expediente:hostil:p3', v_p2.familia_creada_en,
            v_p2.familia_valida_hasta, v_alcance.acceso_registrado_en,
            v_alcance.acceso_ref, v_prueba,
            pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
        );
        RAISE EXCEPTION 'p3 aceptó acceso distinto del consumo padre';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;

    v_prueba := pg_temp.prueba_cursor_rrhh(
        v_token_valido, v_p2.familia_ref, v_p2.token_huella_sha256,
        3, v_p2.emitida_en, v_p2.ultimo_actualizado_en,
        'expediente:valido:p3', v_p2.familia_creada_en,
        v_p2.familia_valida_hasta, v_consumo.consumido_en,
        v_consumo.acceso_consumo_ref
    );
    INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
        token_huella_sha256, familia_ref, padre_token_huella_sha256,
        pagina, padre_emitida_en, ultimo_actualizado_en,
        ultimo_expediente_ref, familia_creada_en, familia_valida_hasta,
        emitida_en, acceso_emision_ref, prueba_canonica,
        prueba_huella_sha256
    ) VALUES (
        v_token_valido, v_p2.familia_ref, v_p2.token_huella_sha256,
        3, v_p2.emitida_en, v_p2.ultimo_actualizado_en,
        'expediente:valido:p3', v_p2.familia_creada_en,
        v_p2.familia_valida_hasta, v_consumo.consumido_en,
        v_consumo.acceso_consumo_ref, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
END
$p3$;
