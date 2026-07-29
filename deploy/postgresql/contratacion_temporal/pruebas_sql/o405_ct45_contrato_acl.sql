\set ON_ERROR_STOP on

-- CT-000045: contrato exterior exacto de las dos fachadas. Esta prueba no
-- conoce auxiliares privados de la implementación y solo inspecciona catálogo.
DO $prueba$
DECLARE
    v_cuadro regprocedure :=
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1('
        'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
        'vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'
        'bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea'
        ')'::regprocedure;
    v_detalle regprocedure :=
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1('
        'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
        'vec_contratacion_temporal.consulta_detalle_rrhh_v1,'
        'bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea'
        ')'::regprocedure;
    v_funcion record;
    v_modos "char"[];
    v_tipos regtype[];
    v_entradas text[];
    v_salidas text[];
BEGIN
    v_entradas := ARRAY[
        'p_alcance', 'p_consulta', 'p_capacidad_canonica',
        'p_decision_canonica', 'p_motivo_canonico',
        'p_contexto_actor_canonico', 'p_persona_version',
        'p_perfil_version', 'p_payload_vec_ad_3',
        'p_sobre_cose_sign_1', 'p_evidencia_verificacion',
        'p_raiz_publica_spki'
    ];
    v_modos := ARRAY[
        'i','i','i','i','i','i','i','i','i','i','i','i',
        't','t','t','t','t','t','t','t','t','t','t','t','t',
        't','t','t','t','t','t','t','t'
    ]::"char"[];
    v_salidas := ARRAY[
        'contenido_canonico', 'cursor_siguiente',
        'esquema', 'acceso_ref', 'secuencia', 'anterior_sha256',
        'huella_sha256', 'vinculo_identidad_huella_sha256',
        'alcance_huella_sha256', 'registrada_en', 'auditoria_vec_ref',
        'auditoria_vec_huella_sha256', 'consumo_vec_huella_sha256',
        'contenido_huella_sha256', 'resultado_huella_sha256',
        'cursor_huella_sha256', 'generada_en', 'expediente_ref',
        'version_expediente', 'total', 'recibo_sello_sha256'
    ];
    v_tipos := ARRAY[
        'vec_contratacion_temporal.alcance_consulta_rrhh_v1'::regtype,
        'vec_contratacion_temporal.consulta_cuadro_rrhh_v1'::regtype,
        'bytea'::regtype, 'bytea'::regtype, 'bytea'::regtype,
        'bytea'::regtype, 'numeric'::regtype, 'numeric'::regtype,
        'bytea'::regtype, 'bytea'::regtype, 'bytea'::regtype,
        'bytea'::regtype,
        'bytea'::regtype, 'text'::regtype, 'text'::regtype,
        'text'::regtype, 'numeric'::regtype, 'text'::regtype,
        'text'::regtype, 'text'::regtype, 'text'::regtype,
        'timestamptz'::regtype, 'text'::regtype, 'text'::regtype,
        'text'::regtype, 'text'::regtype, 'text'::regtype,
        'text'::regtype, 'timestamptz'::regtype, 'text'::regtype,
        'numeric'::regtype, 'smallint'::regtype, 'text'::regtype
    ];

    SELECT funcion.*, propietario.rolname AS propietario
      INTO STRICT v_funcion
      FROM pg_catalog.pg_proc funcion
      JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = funcion.proowner
     WHERE funcion.oid = v_cuadro;
    IF v_funcion.prokind <> 'f'
       OR v_funcion.prosecdef IS NOT TRUE
       OR v_funcion.provolatile <> 'v'
       OR v_funcion.proparallel <> 'u'
       OR v_funcion.proisstrict IS TRUE
       OR v_funcion.proleakproof IS TRUE
       OR v_funcion.propietario <>
          'vec_contratacion_temporal_propietario'
       OR v_funcion.proargmodes IS DISTINCT FROM v_modos
       OR pg_catalog.to_jsonb(v_funcion.proargtypes::oid[])
          IS DISTINCT FROM
          pg_catalog.to_jsonb(v_tipos[1:12]::oid[])
       OR v_funcion.proallargtypes::oid[] IS DISTINCT FROM v_tipos::oid[]
       OR v_funcion.proargnames IS DISTINCT FROM
          (v_entradas || v_salidas)
       OR pg_catalog.obj_description(v_funcion.oid, 'pg_proc') <>
          'Fachada nominal de cuadro RRHH: consume una capacidad y devuelve contenido canónico con Recibo V2.'
       OR v_funcion.proconfig IS DISTINCT FROM ARRAY[
          'search_path=pg_catalog', 'row_security=on', 'TimeZone=UTC',
          'lock_timeout=1s', 'statement_timeout=4s',
          'idle_in_transaction_session_timeout=6s'
       ] THEN
        RAISE EXCEPTION 'contrato exterior de cuadro CT45 divergente';
    END IF;

    v_modos := v_modos[1:13] || v_modos[15:33];
    v_salidas := v_salidas[1:1] || v_salidas[3:21];
    v_tipos[2] :=
        'vec_contratacion_temporal.consulta_detalle_rrhh_v1'::regtype;
    v_tipos := v_tipos[1:13] || v_tipos[15:33];
    SELECT funcion.*, propietario.rolname AS propietario
      INTO STRICT v_funcion
      FROM pg_catalog.pg_proc funcion
      JOIN pg_catalog.pg_roles propietario
        ON propietario.oid = funcion.proowner
     WHERE funcion.oid = v_detalle;
    IF v_funcion.prokind <> 'f'
       OR v_funcion.prosecdef IS NOT TRUE
       OR v_funcion.provolatile <> 'v'
       OR v_funcion.proparallel <> 'u'
       OR v_funcion.proisstrict IS TRUE
       OR v_funcion.proleakproof IS TRUE
       OR v_funcion.propietario <>
          'vec_contratacion_temporal_propietario'
       OR v_funcion.proargmodes IS DISTINCT FROM v_modos
       OR pg_catalog.to_jsonb(v_funcion.proargtypes::oid[])
          IS DISTINCT FROM
          pg_catalog.to_jsonb(v_tipos[1:12]::oid[])
       OR v_funcion.proallargtypes::oid[] IS DISTINCT FROM v_tipos::oid[]
       OR v_funcion.proargnames IS DISTINCT FROM
          (v_entradas || v_salidas)
       OR pg_catalog.obj_description(v_funcion.oid, 'pg_proc') <>
          'Fachada nominal de detalle RRHH: consume una capacidad y devuelve contenido canónico con Recibo V2.'
       OR v_funcion.proconfig IS DISTINCT FROM ARRAY[
          'search_path=pg_catalog', 'row_security=on', 'TimeZone=UTC',
          'lock_timeout=1s', 'statement_timeout=4s',
          'idle_in_transaction_session_timeout=6s'
       ] THEN
        RAISE EXCEPTION 'contrato exterior de detalle CT45 divergente';
    END IF;
END
$prueba$;

-- La inspección cerrada del cuerpo evita el falso verde de limitar después de
-- canonizar o decodificar. La huella de instalación incluye estos cuerpos.
DO $prueba$
DECLARE
    v_nombre name;
    v_definicion text;
    v_limite integer;
    v_posicion integer;
    v_patron text;
    v_patrones_material text[] := ARRAY[
        'octet_length\s*\(\s*p_capacidad_canonica\s*\)\s*NOT BETWEEN 512 AND 32768',
        'octet_length\s*\(\s*p_decision_canonica\s*\)\s*NOT BETWEEN 1 AND 524288',
        'octet_length\s*\(\s*p_motivo_canonico\s*\)\s*NOT BETWEEN 1 AND 65536',
        'octet_length\s*\(\s*p_contexto_actor_canonico\s*\)\s*NOT BETWEEN 1 AND 262144',
        'p_persona_version\s+NOT BETWEEN\s+1 AND 9007199254740991',
        'p_persona_version\s+<>\s+pg_catalog\.trunc\(p_persona_version\)',
        'p_perfil_version\s+NOT BETWEEN\s+1 AND 9007199254740991',
        'p_perfil_version\s+<>\s+pg_catalog\.trunc\(p_perfil_version\)',
        'octet_length\s*\(\s*p_payload_vec_ad_3\s*\)\s*NOT BETWEEN 1 AND 1048576',
        'octet_length\s*\(\s*p_sobre_cose_sign_1\s*\)\s*NOT BETWEEN 1 AND 1048576',
        'octet_length\s*\(\s*p_evidencia_verificacion\s*\)\s*NOT BETWEEN 1 AND 262144',
        'octet_length\s*\(\s*p_raiz_publica_spki\s*\)\s*<>\s*44'
    ];
    v_patrones_alcance text[] := ARRAY[
        'octet_length\s*\(\s*COALESCE\(p_alcance\.organizacion_ref',
        'octet_length\s*\(\s*COALESCE\(p_alcance\.clase_ambito',
        'octet_length\s*\(\s*COALESCE\(p_alcance\.ambito_ref'
    ];
    v_patrones_consulta text[];
    v_constantes text[];
    v_terminales text[];
    v_motor integer;
BEGIN
    FOREACH v_nombre IN ARRAY ARRAY[
        'consultar_cuadro_rrhh_atestado_v1',
        'consultar_detalle_rrhh_atestado_v1'
    ]::name[] LOOP
        SELECT pg_catalog.pg_get_functiondef(funcion.oid)
          INTO STRICT v_definicion
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::regnamespace
           AND funcion.proname = v_nombre;
        v_limite := least(
            pg_catalog.strpos(
                v_definicion,
                'PERFORM vec_contratacion_temporal.canon_alcance_rrhh_v1'
            ),
            pg_catalog.strpos(
                v_definicion,
                'v_capacidad := pg_catalog.convert_from'
            )
        );
        IF v_limite <= 0 THEN
            RAISE EXCEPTION 'orden pre-canon CT45 no localizable: %',
                v_nombre;
        END IF;
        v_patrones_consulta := CASE v_nombre
            WHEN 'consultar_cuadro_rrhh_atestado_v1' THEN ARRAY[
                'octet_length\s*\(\s*COALESCE\(p_consulta\.texto',
                'octet_length\s*\(\s*COALESCE\(p_consulta\.estado_clave',
                'octet_length\s*\(\s*COALESCE\(p_consulta\.fase_clave',
                'octet_length\s*\(\s*COALESCE\(p_consulta\.cursor'
            ]
            ELSE ARRAY[
                'octet_length\s*\(\s*COALESCE\(p_consulta\.expediente_ref'
            ]
        END;
        IF v_nombre = 'consultar_cuadro_rrhh_atestado_v1' THEN
            v_constantes := ARRAY[
                'v_capacidad\s*->>\s*''operacion''\s+IS DISTINCT FROM\s*''contratacion_temporal\.cuadro\.consultar''',
                'v_capacidad\s*->>\s*''audiencia_consumo''\s+IS DISTINCT FROM\s*''vec_contratacion_temporal\.consultar_cuadro_rrhh_atestado\.v1''',
                'v_decision\s*->>\s*''accion''\s+IS DISTINCT FROM\s*''contratacion_temporal\.cuadro\.consultar''',
                'v_decision\s*->>\s*''modulo_id''\s+IS DISTINCT FROM\s*''contratacion_temporal''',
                'v_decision\s*->>\s*''tipo_recurso''\s+IS DISTINCT FROM\s*''cuadro_rrhh_contratacion_temporal''',
                'v_decision\s*->>\s*''finalidad''\s+IS DISTINCT FROM\s*''gestion_operativa_contratacion_temporal''',
                'vec\.contratacion_temporal\.consulta_rrhh\.cuadro\.v1',
                'v_capacidad\s*->>\s*''efecto_ref''\s+IS DISTINCT FROM\s*p_alcance\.ambito_ref',
                'v_decision\s*->>\s*''recurso_ref''\s+IS DISTINCT FROM\s*p_alcance\.ambito_ref'
            ];
            v_terminales := ARRAY[
                'v_resultado\.cursor_siguiente\s+IS DISTINCT FROM\s*''''',
                'v_cierre\.cursor_huella_sha256\s+IS DISTINCT FROM\s*''''',
                'v_cierre\.expediente_ref\s+IS DISTINCT FROM\s*''''',
                'v_cierre\.version_expediente\s+IS DISTINCT FROM\s*0'
            ];
        ELSE
            v_constantes := ARRAY[
                'v_capacidad\s*->>\s*''operacion''\s+IS DISTINCT FROM\s*''contratacion_temporal\.expediente\.consultar''',
                'v_capacidad\s*->>\s*''audiencia_consumo''\s+IS DISTINCT FROM\s*''vec_contratacion_temporal\.consultar_detalle_rrhh_atestado\.v1''',
                'v_decision\s*->>\s*''accion''\s+IS DISTINCT FROM\s*''contratacion_temporal\.expediente\.consultar''',
                'v_decision\s*->>\s*''modulo_id''\s+IS DISTINCT FROM\s*''contratacion_temporal''',
                'v_decision\s*->>\s*''tipo_recurso''\s+IS DISTINCT FROM\s*''expediente_contratacion_temporal''',
                'v_decision\s*->>\s*''finalidad''\s+IS DISTINCT FROM\s*''tramitacion_expediente_contratacion_temporal''',
                'vec\.contratacion_temporal\.consulta_rrhh\.detalle\.v1',
                'v_capacidad\s*->>\s*''efecto_ref''\s+IS DISTINCT FROM\s*p_consulta\.expediente_ref',
                'v_decision\s*->>\s*''recurso_ref''\s+IS DISTINCT FROM\s*p_consulta\.expediente_ref'
            ];
            v_terminales := ARRAY[
                'v_cierre\.cursor_huella_sha256\s+IS DISTINCT FROM\s*''''',
                'v_cierre\.alcance_huella_sha256\s+IS DISTINCT FROM\s*'''''
            ];
        END IF;
        v_motor := pg_catalog.regexp_instr(
            v_definicion,
            CASE v_nombre
                WHEN 'consultar_cuadro_rrhh_atestado_v1'
                THEN '\mmotor_consultar_cuadro_rrhh_v1\s*\('
                ELSE '\mmotor_consultar_detalle_rrhh_v1\s*\('
            END,
            1, 1, 0, 'n'
        );
        FOREACH v_patron IN ARRAY (
            v_patrones_alcance || v_patrones_consulta ||
            v_patrones_material
        ) LOOP
            v_posicion := pg_catalog.regexp_instr(
                v_definicion, v_patron, 1, 1, 0, 'n'
            );
            IF v_posicion = 0 OR v_posicion >= v_limite THEN
                RAISE EXCEPTION
                    'guarda CT45 ausente o tardía en %: %',
                    v_nombre, v_patron;
            END IF;
        END LOOP;
        FOREACH v_patron IN ARRAY v_constantes LOOP
            v_posicion := pg_catalog.regexp_instr(
                v_definicion, v_patron, 1, 1, 0, 'n'
            );
            IF v_posicion = 0 OR v_posicion >= v_motor THEN
                RAISE EXCEPTION 'constante CT45 ausente o tardía: %/%',
                    v_nombre, v_patron;
            END IF;
        END LOOP;
        FOREACH v_patron IN ARRAY v_terminales LOOP
            v_posicion := pg_catalog.regexp_instr(
                v_definicion, v_patron, 1, 1, 0, 'n'
            );
            IF v_posicion <= v_motor THEN
                RAISE EXCEPTION 'guarda NULL CT45 ausente: %/%',
                    v_nombre, v_patron;
            END IF;
        END LOOP;
        IF v_motor = 0 OR pg_catalog.regexp_count(
               v_definicion,
               CASE v_nombre
                   WHEN 'consultar_cuadro_rrhh_atestado_v1'
                   THEN '\mmotor_consultar_cuadro_rrhh_v1\s*\('
                   ELSE '\mmotor_consultar_detalle_rrhh_v1\s*\('
               END,
               1, 'n'
           ) <> 1 THEN
            RAISE EXCEPTION 'motor CT44 no se invoca una sola vez: %',
                v_nombre;
        END IF;
    END LOOP;
END
$prueba$;

-- El consultor recibe exclusivamente las dos fachadas. PUBLIC y los roles
-- internos no adquieren privilegios por defecto o por herencia accidental.
DO $prueba$
DECLARE
    v_firma regprocedure;
    v_rol name;
BEGIN
    FOREACH v_firma IN ARRAY ARRAY[
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1('
        'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
        'vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'
        'bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea'
        ')'::regprocedure,
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1('
        'vec_contratacion_temporal.alcance_consulta_rrhh_v1,'
        'vec_contratacion_temporal.consulta_detalle_rrhh_v1,'
        'bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea'
        ')'::regprocedure
    ] LOOP
        IF pg_catalog.has_function_privilege('public', v_firma, 'EXECUTE')
           OR NOT pg_catalog.has_function_privilege(
              'vec_contratacion_temporal_consultor_rrhh',
              v_firma, 'EXECUTE'
           ) THEN
            RAISE EXCEPTION 'ACL exterior CT45 divergente: %', v_firma;
        END IF;
        FOREACH v_rol IN ARRAY ARRAY[
            'vec_contratacion_temporal_migrador',
            'vec_contratacion_temporal_ejecutor',
            'vec_contratacion_temporal_confirmador_cobertura',
            'vec_contratacion_temporal_gobernador',
            'vec_contratacion_temporal_lector_resultado_cobertura'
        ]::name[] LOOP
            IF pg_catalog.has_function_privilege(
                v_rol, v_firma, 'EXECUTE'
            ) THEN
                RAISE EXCEPTION
                    'rol % recibió ejecución CT45 no autorizada', v_rol;
            END IF;
        END LOOP;
    END LOOP;
END
$prueba$;

-- La cuenta runtime es nominativa y su única pertenencia directa es el grupo
-- consultor, sin ADMIN, SET ROLE ni puente adicional.
DO $prueba$
DECLARE
    v_login pg_catalog.pg_authid%ROWTYPE;
BEGIN
    SELECT * INTO STRICT v_login
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_c2d2_registro_runtime';
    IF NOT v_login.rolcanlogin OR NOT v_login.rolinherit
       OR v_login.rolsuper OR v_login.rolcreatedb OR v_login.rolcreaterole
       OR v_login.rolreplication OR v_login.rolbypassrls
       OR (
          SELECT pg_catalog.count(*)
            FROM pg_catalog.pg_auth_members membresia
            JOIN pg_catalog.pg_roles grupo ON grupo.oid = membresia.roleid
           WHERE membresia.member = v_login.oid
             AND grupo.rolname = 'vec_contratacion_temporal_consultor_rrhh'
             AND NOT membresia.admin_option
             AND membresia.inherit_option
             AND NOT membresia.set_option
       ) <> 1
       OR (
          SELECT pg_catalog.count(*)
            FROM pg_catalog.pg_auth_members
           WHERE member = v_login.oid
       ) <> 1 THEN
        RAISE EXCEPTION 'topología runtime CT45 divergente';
    END IF;
END
$prueba$;

-- Sin USAGE CT40--CT44 se puede invocar una firma mediante ROW escalar, pero
-- el mismo login no puede emplear esos tipos para crear almacenamiento.
DO $prueba$
DECLARE
    v_tipo regtype;
BEGIN
    FOREACH v_tipo IN ARRAY ARRAY[
        'vec_contratacion_temporal.alcance_consulta_rrhh_v1'::regtype,
        'vec_contratacion_temporal.consulta_cuadro_rrhh_v1'::regtype,
        'vec_contratacion_temporal.consulta_detalle_rrhh_v1'::regtype,
        'vec_contratacion_temporal.evidencia_resultado_rrhh_v1'::regtype,
        'vec_contratacion_temporal.resumen_publicacion_rrhh_v1'::regtype,
        'vec_contratacion_temporal.solicitud_operativa_rrhh_v1'::regtype,
        'vec_contratacion_temporal.analisis_operativo_rrhh_v1'::regtype,
        'vec_contratacion_temporal.comprobacion_operativa_rrhh_v1'::regtype,
        'vec_contratacion_temporal.cobertura_operativa_rrhh_v1'::regtype,
        'vec_contratacion_temporal.asignacion_operativa_rrhh_v1'::regtype,
        'vec_contratacion_temporal.hito_expediente_rrhh_v1'::regtype,
        'vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1'::regtype,
        'vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2'::regtype,
        'vec_contratacion_temporal.contexto_cierre_prueba_rrhh_v2'::regtype,
        'vec_contratacion_temporal.contenido_cierre_prueba_rrhh_v2'::regtype,
        'vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3'::regtype,
        'vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2'::regtype,
        'vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3'::regtype,
        'vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1'::regtype,
        'vec_contratacion_temporal.materializacion_cuadro_rrhh_v1'::regtype,
        'vec_contratacion_temporal.materializacion_detalle_rrhh_v1'::regtype,
        'vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1'::regtype,
        'vec_contratacion_temporal.resultado_motor_cuadro_rrhh_v1'::regtype,
        'vec_contratacion_temporal.resultado_motor_detalle_rrhh_v1'::regtype
    ] LOOP
        IF pg_catalog.has_type_privilege(
            'vec_c2d2_registro_runtime', v_tipo, 'USAGE'
        ) THEN
            RAISE EXCEPTION 'runtime recibió USAGE privado: %', v_tipo;
        END IF;
    END LOOP;
END
$prueba$;

-- La ejecución no abre una vía lateral de almacenamiento o DDL.
DO $prueba$
BEGIN
    IF pg_catalog.has_schema_privilege(
           'vec_c2d2_registro_runtime',
           'vec_contratacion_temporal', 'CREATE'
       )
       OR pg_catalog.has_database_privilege(
           'vec_c2d2_registro_runtime',
           pg_catalog.current_database(), 'TEMP'
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_class relacion
             CROSS JOIN pg_catalog.unnest(ARRAY[
                 'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'TRUNCATE',
                 'REFERENCES', 'TRIGGER'
             ]) privilegio
            WHERE relacion.relnamespace =
                  'vec_contratacion_temporal'::regnamespace
              AND relacion.relkind = ANY(ARRAY[
                  'r', 'p', 'v', 'm', 'f'
              ]::"char"[])
              AND pg_catalog.has_table_privilege(
                  'vec_c2d2_registro_runtime',
                  relacion.oid, privilegio
              )
       ) THEN
        RAISE EXCEPTION 'runtime conserva almacenamiento o DDL CT45';
    END IF;
END
$prueba$;
