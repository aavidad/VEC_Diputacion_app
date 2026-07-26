\set ON_ERROR_STOP on

CREATE FUNCTION pg_temp.recurso_canonico_t13(p_filtro jsonb, p_actor text)
RETURNS text
LANGUAGE sql
SET search_path = pg_catalog
AS $funcion$
    SELECT encode(convert_to(jsonb_build_object(
        'ambitos', jsonb_build_object('version_esquema', '1'),
        'atributos', jsonb_build_object(
            'actor_operador_seudonimizado', p_actor,
            'finalidad_consulta', p_filtro ->> 'finalidad_consulta',
            'filtro_version_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'version'
                ),
            'filtro_actor_seudonimizado_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'actor_seudonimizado'
                ),
            'filtro_modulo_id_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'module_id'
                ),
            'filtro_accion_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'accion'
                ),
            'filtro_finalidad_acceso_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'finalidad_acceso'
                ),
            'filtro_recurso_ref_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'recurso_ref'
                ),
            'filtro_expediente_ref_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'expediente_ref'
                ),
            'filtro_resultado_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'resultado'
                ),
            'filtro_desde_inclusive_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'desde_inclusive'
                ),
            'filtro_hasta_exclusive_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'hasta_exclusive'
                ),
            'filtro_version_objeto_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'version_objeto'
                ),
            'filtro_limite_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'limite'
                ),
            'filtro_cursor_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'cursor'
                ),
            'filtro_finalidad_consulta_sha256',
                vec_bolsa_registro_accesos.huella_valor_filtro_v1(
                    p_filtro ->> 'finalidad_consulta'
                )
        )
    )::text, 'UTF8'), 'base64')
$funcion$;

CREATE FUNCTION pg_temp.consulta_t13_debe_fallar(p_solicitud jsonb)
RETURNS boolean
LANGUAGE plpgsql
SET search_path = pg_catalog
AS $funcion$
BEGIN
    PERFORM vec_bolsa_registro_accesos.
        consultar_accesos_administrativos_v1(p_solicitud);
    RETURN false;
EXCEPTION
    WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN
        RETURN true;
END
$funcion$;

-- Segundo evento previo para ejercer límite/cursor. La primera lectura debe
-- ver este evento, no su propio asiento de auditoría.
SELECT jsonb_build_object(
    'actor_id', 'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
    'actor_profile', 'auditoria-interna', 'actor_roles', '[]'::jsonb,
    'represented_subject_id', '', 'auth_method', 'sso',
    'auth_assurance', 'alto', 'authorization_ref', '',
    'purpose', 'tramitacion', 'action', 'expediente.leer',
    'module_id', 'vec.module.bolsa',
    'subject_ref', 'expediente:sha256:' || repeat('d', 64),
    'object_version', 6, 'expediente_ref', '', 'document_ref', '',
    'rule_ref', '', 'reason', '', 'result', 'permitido',
    'before_hash', '', 'after_hash', '',
    'correlation_ref', 'evento-paginacion-' || repeat('8', 32),
    'metadata', '{}'::jsonb,
    'occurred_at', to_char(
        clock_timestamp() AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    )
)::text AS evento_paginacion \gset
SET SESSION AUTHORIZATION vec_bolsa_accesos_propietario;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.registrar_interno_v1(
    :'evento_paginacion'::jsonb
);
COMMIT;
RESET SESSION AUTHORIZATION;

SELECT jsonb_build_object(
    'version', 1,
    'filtro', jsonb_build_object(
        'version', 1,
        'actor_seudonimizado',
            'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
        'module_id', '', 'accion', '', 'finalidad_acceso', '',
        'recurso_ref', '', 'expediente_ref', '', 'resultado', '',
        'desde_inclusive', to_char(
            (clock_timestamp() - interval '1 hour') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'hasta_exclusive', to_char(
            (clock_timestamp() + interval '1 hour') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'version_objeto', 0, 'limite', 1, 'cursor', '',
        'finalidad_consulta', 'control-interno'
    ),
    'auditoria', jsonb_build_object(
        'actor_id',
            'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
        'actor_profile', 'auditoria-interna',
        'actor_roles', '["auditor","supervisor"]'::jsonb,
        'represented_subject_id', '', 'auth_method', 'sso',
        'auth_assurance', 'alto',
        'authorization_ref', 'decision_mecanica_t13',
        'purpose', 'control-interno',
        'action', 'bolsa.registro_accesos.consultar',
        'module_id', 'vec.module.bolsa',
        'subject_ref', 'consulta-accesos:sha256:' || repeat('2', 64),
        'object_version', 1, 'expediente_ref', '', 'document_ref', '',
        'rule_ref', '', 'reason', '', 'result', 'permitido',
        'before_hash', '', 'after_hash', '',
        'correlation_ref', 'correlacion_' || repeat('1', 32),
        'metadata', '{}'::jsonb,
        'occurred_at', to_char(
            clock_timestamp() AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
    ),
    'autorizacion', jsonb_build_object(
        'prueba', jsonb_build_object(
            'esquema_huella',
                'vec.autorizacion.decision.reforzada.v2.solicitud-ligada',
            'decision_ref', 'decision_mecanica_t13',
            'huella_decision_sha256',
                encode(sha256(convert_to('{}', 'UTF8')), 'hex'),
            'verificada_en', to_char(
                clock_timestamp() AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
            ),
            'principal_ref', 'principal_mecanica_t13'
        ),
        'decision_canonica', encode(convert_to('{}', 'UTF8'), 'base64'),
        'recurso_canonico', encode(convert_to(jsonb_build_object(
            'ambitos', jsonb_build_object('version_esquema', '1'),
            'atributos', jsonb_build_object(
                'actor_operador_seudonimizado',
                    'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
                'finalidad_consulta', 'control-interno'
            )
        )::text, 'UTF8'), 'base64')
    )
)::text AS solicitud_1 \gset
SELECT jsonb_set(
    :'solicitud_1'::jsonb, '{autorizacion,recurso_canonico}',
    to_jsonb(pg_temp.recurso_canonico_t13(
        :'solicitud_1'::jsonb -> 'filtro',
        :'solicitud_1'::jsonb -> 'auditoria' ->> 'actor_id'
    ))
)::text AS solicitud_1 \gset

-- Una decisión y un recurso canónico válidos no autorizan un filtro mutado.
SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
WITH base AS (
    SELECT :'solicitud_1'::jsonb AS solicitud
), mutaciones(campo, valor) AS (
    VALUES
        ('version', '2'::jsonb),
        ('actor_seudonimizado', to_jsonb(
            'hmac-sha256:bolsa_accesos_v1:' || repeat('b', 64)
        )),
        ('module_id', '"otro.modulo"'::jsonb),
        ('accion', '"otra.accion"'::jsonb),
        ('finalidad_acceso', '"otra-finalidad"'::jsonb),
        ('recurso_ref', '"expediente:otro"'::jsonb),
        ('expediente_ref', '"expediente:distinto"'::jsonb),
        ('resultado', '"denegado"'::jsonb),
        ('desde_inclusive', to_jsonb(to_char(
            (
                (SELECT solicitud -> 'filtro' ->> 'desde_inclusive' FROM base)
            )::timestamptz + interval '1 microsecond',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ))),
        ('hasta_exclusive', to_jsonb(to_char(
            (
                (SELECT solicitud -> 'filtro' ->> 'hasta_exclusive' FROM base)
            )::timestamptz - interval '1 microsecond',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ))),
        ('version_objeto', '8'::jsonb),
        ('limite', '2'::jsonb),
        ('cursor', to_jsonb('cursor:v1:' || repeat('e', 64))),
        ('finalidad_consulta', '"otra-finalidad"'::jsonb)
)
SELECT bool_and(pg_temp.consulta_t13_debe_fallar(
           jsonb_set(base.solicitud, ARRAY['filtro', campo], valor)
       )) AS filtro_ligado
  FROM base CROSS JOIN mutaciones \gset
ROLLBACK;
RESET SESSION AUTHORIZATION;
\if :filtro_ligado
\else
    \echo 'una mutación del filtro escapó del recurso VEC durable'
    SELECT 1 / 0;
\endif

-- Cada clave rechaza JSON null y un tipo booleano antes de cualquier cast.
SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
WITH base AS (
    SELECT :'solicitud_1'::jsonb AS solicitud
), campos(campo) AS (
    SELECT unnest(ARRAY[
        'version', 'actor_seudonimizado', 'module_id', 'accion',
        'finalidad_acceso', 'recurso_ref', 'expediente_ref', 'resultado',
        'desde_inclusive', 'hasta_exclusive', 'version_objeto', 'limite',
        'cursor', 'finalidad_consulta'
    ])
)
SELECT bool_and(
           pg_temp.consulta_t13_debe_fallar(jsonb_set(
               base.solicitud, ARRAY['filtro', campo], 'null'::jsonb
           ))
           AND pg_temp.consulta_t13_debe_fallar(jsonb_set(
               base.solicitud, ARRAY['filtro', campo], 'true'::jsonb
           ))
       ) AS tipos_cerrados
  FROM base CROSS JOIN campos \gset
ROLLBACK;
RESET SESSION AUTHORIZATION;
\if :tipos_cerrados
\else
    \echo 'un null o tipo JSON incorrecto llegó a los casts T13'
    SELECT 1 / 0;
\endif

-- La traza T13 también es exacta; no basta una decisión válida.
SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
WITH base AS (
    SELECT :'solicitud_1'::jsonb AS solicitud
), mutaciones(campo, valor) AS (
    VALUES
        ('actor_id', to_jsonb(
            'hmac-sha256:bolsa_accesos_v1:' || repeat('b', 64)
        )),
        ('actor_profile', '""'::jsonb),
        ('actor_roles', '[]'::jsonb),
        ('represented_subject_id', '"sujeto"'::jsonb),
        ('auth_method', '"dnie"'::jsonb),
        ('auth_assurance', '"medio"'::jsonb),
        ('authorization_ref', '"otra-decision"'::jsonb),
        ('purpose', '"otra-finalidad"'::jsonb),
        ('action', '"otra.accion"'::jsonb),
        ('module_id', '"otro.modulo"'::jsonb),
        ('subject_ref', to_jsonb(
            'consulta-accesos:sha256:' || repeat('e', 64)
        )),
        ('object_version', '2'::jsonb),
        ('expediente_ref', '"expediente"'::jsonb),
        ('document_ref', '"documento"'::jsonb),
        ('rule_ref', '"regla"'::jsonb),
        ('reason', '"motivo"'::jsonb),
        ('result', '"denegado"'::jsonb),
        ('before_hash', to_jsonb(repeat('a', 64))),
        ('after_hash', to_jsonb(repeat('b', 64))),
        ('correlation_ref', to_jsonb(
            'correlacion_' || repeat('e', 32)
        )),
        ('metadata', '{"inesperado":"valor"}'::jsonb)
)
SELECT bool_and(pg_temp.consulta_t13_debe_fallar(
           jsonb_set(base.solicitud, ARRAY['auditoria', campo], valor)
       )) AS auditoria_exacta
  FROM base CROSS JOIN mutaciones \gset
ROLLBACK;
RESET SESSION AUTHORIZATION;
\if :auditoria_exacta
\else
    \echo 'una mutación de auditoría T13 fue aceptada'
    SELECT 1 / 0;
\endif

-- Hexadecimal mayúsculo nunca es equivalente al canónico SQL.
SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT pg_temp.consulta_t13_debe_fallar(jsonb_set(
           :'solicitud_1'::jsonb, '{filtro,actor_seudonimizado}',
           to_jsonb(
               'hmac-sha256:bolsa_accesos_v1:' || repeat('A', 64)
           )
       ))
       AND pg_temp.consulta_t13_debe_fallar(jsonb_set(
           :'solicitud_1'::jsonb, '{filtro,cursor}',
           to_jsonb('cursor:v1:' || repeat('A', 64))
       )) AS hex_minusculo \gset
ROLLBACK;
RESET SESSION AUTHORIZATION;
\if :hex_minusculo
\else
    \echo 'hexadecimal mayúsculo aceptado en T13'
    SELECT 1 / 0;
\endif

SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
PREPARE consultar_t13(jsonb) AS
    SELECT vec_bolsa_registro_accesos.
        consultar_accesos_administrativos_v1($1);
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
EXECUTE consultar_t13(:'solicitud_1'::jsonb) \gset pagina_
COMMIT;
DEALLOCATE consultar_t13;
RESET SESSION AUTHORIZATION;

SELECT (:'pagina_consultar_accesos_administrativos_v1'::jsonb
            ->> 'siguiente') ~ '^cursor:v1:[0-9a-f]{64}$'
       AND jsonb_array_length(
           :'pagina_consultar_accesos_administrativos_v1'::jsonb
               -> 'registros'
       ) = 1
       AND :'pagina_consultar_accesos_administrativos_v1'::jsonb
           -> 'registros' -> 0 ->> 'recurso_ref' =
           'expediente:sha256:' || repeat('d', 64)
       AND NOT EXISTS (
           SELECT 1
             FROM jsonb_array_elements(
                 :'pagina_consultar_accesos_administrativos_v1'::jsonb
                     -> 'registros'
             ) AS fila
            WHERE fila ->> 'recurso_ref' =
                  'consulta-accesos:sha256:' || repeat('2', 64)
       )
       AND EXISTS (
           SELECT 1
             FROM vec_bolsa_registro_accesos.registro_acceso
            WHERE subject_ref =
                  'consulta-accesos:sha256:' || repeat('2', 64)
       ) AS pagina_limitada \gset
\if :pagina_limitada
\else
    \echo 'límite, cursor o frontera anterior al asiento T13 incorrectos'
    SELECT 1 / 0;
\endif

-- Un evento nuevo aparece después de la primera lectura.
SELECT jsonb_build_object(
    'actor_id', 'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
    'actor_profile', 'auditoria-interna', 'actor_roles', '[]'::jsonb,
    'represented_subject_id', '', 'auth_method', 'sso',
    'auth_assurance', 'alto', 'authorization_ref', '',
    'purpose', 'tramitacion', 'action', 'expediente.leer',
    'module_id', 'vec.module.bolsa',
    'subject_ref', 'expediente:sha256:' || repeat('f', 64),
    'object_version', 8, 'expediente_ref', '', 'document_ref', '',
    'rule_ref', '', 'reason', '', 'result', 'permitido',
    'before_hash', '', 'after_hash', '',
    'correlation_ref', 'evento-intermedio-' || repeat('5', 32),
    'metadata', '{}'::jsonb,
    'occurred_at', to_char(
        clock_timestamp() AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
    )
)::text AS evento_intermedio \gset
SET SESSION AUTHORIZATION vec_bolsa_accesos_propietario;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.registrar_interno_v1(
    :'evento_intermedio'::jsonb
);
COMMIT;
RESET SESSION AUTHORIZATION;

-- El mismo efecto no puede ejecutar otra SELECT y observar el evento nuevo.
SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
\set ON_ERROR_STOP off
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(
    :'solicitud_1'::jsonb
);
\set estado_replay :SQLSTATE
ROLLBACK;
\set ON_ERROR_STOP on
RESET SESSION AUTHORIZATION;
SELECT :'estado_replay' = '42501' AS replay_denegado \gset
\if :replay_denegado
\else
    \echo 'una decisión consumida volvió a ejecutar la lectura'
    SELECT 1 / 0;
\endif

-- Una decisión nueva recorre la página siguiente con el cursor anterior.
-- El evento intermedio es más nuevo que el cursor y no puede colarse.
SELECT jsonb_build_object(
    'version', 1,
    'filtro', jsonb_set(
        :'solicitud_1'::jsonb -> 'filtro',
        '{cursor}',
        to_jsonb(
            :'pagina_consultar_accesos_administrativos_v1'::jsonb
                ->> 'siguiente'
        )
    ),
    'auditoria', jsonb_set(
        jsonb_set(
            jsonb_set(
                :'solicitud_1'::jsonb -> 'auditoria',
                '{authorization_ref}', '"decision_mecanica_t13_2"'
            ),
            '{subject_ref}',
            to_jsonb('consulta-accesos:sha256:' || repeat('4', 64))
        ),
        '{correlation_ref}',
        to_jsonb('correlacion_' || repeat('3', 32))
    ),
    'autorizacion', jsonb_set(
        :'solicitud_1'::jsonb -> 'autorizacion',
        '{prueba,decision_ref}', '"decision_mecanica_t13_2"'
    )
)::text AS solicitud_pagina_2 \gset
SELECT jsonb_set(
    :'solicitud_pagina_2'::jsonb, '{autorizacion,recurso_canonico}',
    to_jsonb(pg_temp.recurso_canonico_t13(
        :'solicitud_pagina_2'::jsonb -> 'filtro',
        :'solicitud_pagina_2'::jsonb -> 'auditoria' ->> 'actor_id'
    ))
)::text AS solicitud_pagina_2 \gset
SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(
    :'solicitud_pagina_2'::jsonb
) \gset pagina_2_
COMMIT;
RESET SESSION AUTHORIZATION;

SELECT jsonb_array_length(
           :'pagina_2_consultar_accesos_administrativos_v1'::jsonb
               -> 'registros'
       ) = 1
       AND :'pagina_2_consultar_accesos_administrativos_v1'::jsonb
           -> 'registros' -> 0 ->> 'recurso_ref' =
           'expediente:sha256:' || repeat('b', 64)
       AND :'pagina_2_consultar_accesos_administrativos_v1'::jsonb
           ->> 'siguiente' = ''
       AS pagina_estable \gset
\if :pagina_estable
\else
    \echo 'el cursor incorporó eventos nuevos o perdió la página estable'
    SELECT 1 / 0;
\endif

-- Tercera decisión: combina finalidad, módulo, acción, recurso, resultado y
-- versión exactos. Solo puede devolver el registro inicial.
SELECT jsonb_build_object(
    'version', 1,
    'filtro', jsonb_build_object(
        'version', 1,
        'actor_seudonimizado',
            'hmac-sha256:bolsa_accesos_v1:' || repeat('a', 64),
        'module_id', 'vec.module.bolsa',
        'accion', 'expediente.leer',
        'finalidad_acceso', 'tramitacion',
        'recurso_ref', 'expediente:sha256:' || repeat('b', 64),
        'expediente_ref', 'expediente:sha256:' || repeat('c', 64),
        'resultado', 'permitido',
        'desde_inclusive', to_char(
            (clock_timestamp() - interval '1 hour') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'hasta_exclusive', to_char(
            (clock_timestamp() + interval '1 hour') AT TIME ZONE 'UTC',
            'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        ),
        'version_objeto', 7, 'limite', 10, 'cursor', '',
        'finalidad_consulta', 'control-interno'
    ),
    'auditoria', jsonb_set(
        jsonb_set(
            jsonb_set(
                :'solicitud_1'::jsonb -> 'auditoria',
                '{authorization_ref}', '"decision_mecanica_t13_3"'
            ),
            '{subject_ref}',
            to_jsonb('consulta-accesos:sha256:' || repeat('7', 64))
        ),
        '{correlation_ref}',
        to_jsonb('correlacion_' || repeat('6', 32))
    ),
    'autorizacion', jsonb_set(
        :'solicitud_1'::jsonb -> 'autorizacion',
        '{prueba,decision_ref}', '"decision_mecanica_t13_3"'
    )
)::text AS solicitud_2 \gset
SELECT jsonb_set(
    :'solicitud_2'::jsonb, '{autorizacion,recurso_canonico}',
    to_jsonb(pg_temp.recurso_canonico_t13(
        :'solicitud_2'::jsonb -> 'filtro',
        :'solicitud_2'::jsonb -> 'auditoria' ->> 'actor_id'
    ))
)::text AS solicitud_2 \gset
SET SESSION AUTHORIZATION vec_bolsa_accesos_consultor_prueba;
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT vec_bolsa_registro_accesos.consultar_accesos_administrativos_v1(
    :'solicitud_2'::jsonb
) \gset filtrada_
COMMIT;
RESET SESSION AUTHORIZATION;

SELECT jsonb_array_length(
           :'filtrada_consultar_accesos_administrativos_v1'::jsonb
               -> 'registros'
       ) = 1
       AND :'filtrada_consultar_accesos_administrativos_v1'::jsonb
           -> 'registros' -> 0 ->> 'version_objeto' = '7'
       AND :'filtrada_consultar_accesos_administrativos_v1'::jsonb
           -> 'registros' -> 0 ->> 'finalidad' = 'tramitacion'
       AND (SELECT count(*)
              FROM vec_bolsa_registro_accesos.consumo_efecto_consulta) = 3
       AS filtros_y_consumo \gset
\if :filtros_y_consumo
\else
    \echo 'filtros/finalidad o consumo único T13 incorrectos'
    SELECT 1 / 0;
\endif
