-- Inventario negativo de ACL. Se ejecuta como superusuario de integracion.
DO $roles$
DECLARE
    rol text;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = 'vec_bolsa_convocatorias_propietario'
           AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
    ) THEN
        RAISE EXCEPTION 'propietario no conserva NOLOGIN/NOBYPASSRLS';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion',
        'vec_bolsa_convocatorias_verificador_recibo',
        'vec_convocatorias_ejecutor_prueba',
        'vec_convocatorias_proyector_prueba',
        'vec_convocatorias_registrador_prueba',
        'vec_convocatorias_verificador_prueba'
    ] LOOP
        IF pg_has_role(
               rol, 'vec_bolsa_convocatorias_propietario', 'MEMBER'
           ) THEN
            RAISE EXCEPTION 'rol runtime miembro del propietario: %', rol;
        END IF;
    END LOOP;
END
$roles$;

DO $cierre$
DECLARE
    rol text;
    funcion record;
    tabla record;
    secuencia record;
    tipo record;
    atributo record;
    privilegio text;
    permitidas text[];
    debe_ejecutar boolean;
    es_ejecutor boolean;
    es_proyector boolean;
    es_verificador boolean;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'public',
        'vec_bolsa_convocatorias_ejecutor_consulta',
        'vec_bolsa_convocatorias_proyector_gobierno',
        'vec_bolsa_convocatorias_registrador_atestacion',
        'vec_bolsa_convocatorias_verificador_recibo',
        'vec_convocatorias_ejecutor_prueba',
        'vec_convocatorias_proyector_prueba',
        'vec_convocatorias_registrador_prueba',
        'vec_convocatorias_verificador_prueba'
    ] LOOP
        es_ejecutor := rol IN (
            'vec_bolsa_convocatorias_ejecutor_consulta',
            'vec_convocatorias_ejecutor_prueba'
        );
        es_proyector := rol IN (
            'vec_bolsa_convocatorias_proyector_gobierno',
            'vec_convocatorias_proyector_prueba'
        );
        es_verificador := rol IN (
            'vec_bolsa_convocatorias_verificador_recibo',
            'vec_convocatorias_verificador_prueba'
        );
        IF has_schema_privilege(
               rol, 'vec_bolsa_convocatorias', 'USAGE'
           ) IS DISTINCT FROM (
               es_ejecutor OR es_proyector OR es_verificador
           ) THEN
            RAISE EXCEPTION 'USAGE de schema inesperado para %', rol;
        END IF;
        IF has_schema_privilege(
               rol, 'vec_bolsa_convocatorias', 'CREATE'
           ) THEN
            RAISE EXCEPTION 'CREATE de schema abierto para %', rol;
        END IF;
        permitidas := CASE rol
            WHEN 'vec_bolsa_convocatorias_ejecutor_consulta' THEN ARRAY[
                'vec_bolsa_convocatorias.listar_borradores_v1(jsonb,jsonb,jsonb,bytea,bytea)',
                'vec_bolsa_convocatorias.obtener_borrador_v1(text,jsonb,jsonb,bytea,bytea)'
            ]
            WHEN 'vec_convocatorias_ejecutor_prueba' THEN ARRAY[
                'vec_bolsa_convocatorias.listar_borradores_v1(jsonb,jsonb,jsonb,bytea,bytea)',
                'vec_bolsa_convocatorias.obtener_borrador_v1(text,jsonb,jsonb,bytea,bytea)'
            ]
            WHEN 'vec_bolsa_convocatorias_proyector_gobierno' THEN ARRAY[
                'vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb)',
                'vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(jsonb,text,bigint,bigint,timestamp with time zone)',
                'vec_bolsa_convocatorias.reservar_decision_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea)',
                'vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(bigint,bigint,jsonb,jsonb,bytea,bytea,bytea,bytea)',
                'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)',
                'vec_bolsa_convocatorias.confirmar_borrador_v1(text,jsonb,bytea)'
            ]
            WHEN 'vec_convocatorias_proyector_prueba' THEN ARRAY[
                'vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb)',
                'vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(jsonb,text,bigint,bigint,timestamp with time zone)',
                'vec_bolsa_convocatorias.reservar_decision_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea)',
                'vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(bigint,bigint,jsonb,jsonb,bytea,bytea,bytea,bytea)',
                'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)',
                'vec_bolsa_convocatorias.confirmar_borrador_v1(text,jsonb,bytea)'
            ]
            WHEN 'vec_bolsa_convocatorias_verificador_recibo' THEN ARRAY[
                'vec_bolsa_convocatorias.verificar_recibo_borrador_v1(text,text,text)'
            ]
            WHEN 'vec_convocatorias_verificador_prueba' THEN ARRAY[
                'vec_bolsa_convocatorias.verificar_recibo_borrador_v1(text,text,text)'
            ]
            ELSE ARRAY[]::text[]
        END;
        FOR funcion IN
            SELECT p.oid, p.oid::regprocedure AS firma
              FROM pg_catalog.pg_proc AS p
              JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
        LOOP
            debe_ejecutar := funcion.firma::text = ANY(permitidas);
            IF has_function_privilege(rol, funcion.oid, 'EXECUTE')
               IS DISTINCT FROM debe_ejecutar THEN
                RAISE EXCEPTION 'ACL de funcion inesperada para %: % (esperada=%)',
                    rol, funcion.firma, debe_ejecutar;
            END IF;
        END LOOP;
        FOR tabla IN
            SELECT c.oid, c.relname
              FROM pg_catalog.pg_class AS c
              JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
               AND c.relkind IN ('r','p')
        LOOP
            FOREACH privilegio IN ARRAY ARRAY[
                'SELECT','INSERT','UPDATE','DELETE','TRUNCATE',
                'REFERENCES','TRIGGER','MAINTAIN'
            ] LOOP
                IF has_table_privilege(rol, tabla.oid, privilegio) THEN
                    RAISE EXCEPTION 'tabla abierta para %: % (%)',
                        rol, tabla.relname, privilegio;
                END IF;
            END LOOP;
            FOR atributo IN
                SELECT a.attnum, a.attname
                  FROM pg_catalog.pg_attribute AS a
                 WHERE a.attrelid = tabla.oid AND a.attnum > 0
                   AND NOT a.attisdropped
            LOOP
                FOREACH privilegio IN ARRAY ARRAY[
                    'SELECT','INSERT','UPDATE','REFERENCES'
                ] LOOP
                    IF has_column_privilege(
                           rol, tabla.oid, atributo.attnum, privilegio
                       ) THEN
                        RAISE EXCEPTION 'columna abierta para %: %.% (%)',
                            rol, tabla.relname, atributo.attname, privilegio;
                    END IF;
                END LOOP;
            END LOOP;
        END LOOP;
        FOR secuencia IN
            SELECT c.oid, c.relname
              FROM pg_catalog.pg_class AS c
              JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
               AND c.relkind = 'S'
        LOOP
            FOREACH privilegio IN ARRAY ARRAY['SELECT','USAGE','UPDATE'] LOOP
                IF has_sequence_privilege(
                       rol, secuencia.oid, privilegio
                   ) THEN
                    RAISE EXCEPTION 'secuencia abierta para %: % (%)',
                        rol, secuencia.relname, privilegio;
                END IF;
            END LOOP;
        END LOOP;
        FOR tipo IN
            SELECT t.oid, t.typname
              FROM pg_catalog.pg_type AS t
              LEFT JOIN pg_catalog.pg_class AS c ON c.oid = t.typrelid
              JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
             WHERE n.nspname = 'vec_bolsa_convocatorias'
               AND t.typisdefined AND t.typelem = 0
               AND (t.typrelid = 0 OR c.relkind = 'c')
        LOOP
            IF has_type_privilege(rol, tipo.oid, 'USAGE') THEN
                RAISE EXCEPTION 'tipo abierto para %: %', rol, tipo.typname;
            END IF;
        END LOOP;
    END LOOP;
END
$cierre$;

DO $acl_por_defecto$
DECLARE
    propietario oid;
    esquema oid;
    clase "char";
    rol oid;
BEGIN
    SELECT oid INTO STRICT propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_bolsa_convocatorias_propietario';
    SELECT oid INTO STRICT esquema
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_bolsa_convocatorias';
    FOREACH clase IN ARRAY ARRAY['r'::"char", 'S'::"char", 'f'::"char", 'T'::"char"] LOOP
        FOREACH rol IN ARRAY ARRAY[
            0::oid,
            'vec_bolsa_convocatorias_ejecutor_consulta'::regrole::oid,
            'vec_bolsa_convocatorias_proyector_gobierno'::regrole::oid,
            'vec_bolsa_convocatorias_registrador_atestacion'::regrole::oid,
            'vec_bolsa_convocatorias_verificador_recibo'::regrole::oid
        ] LOOP
            IF EXISTS (
                SELECT 1
                  FROM pg_catalog.aclexplode(COALESCE(
                           (SELECT d.defaclacl
                              FROM pg_catalog.pg_default_acl AS d
                             WHERE d.defaclrole = propietario
                               AND d.defaclnamespace = 0
                               AND d.defaclobjtype = clase),
                           pg_catalog.acldefault(clase, propietario)
                       )) AS a
                 WHERE a.grantee = rol
            ) OR EXISTS (
                SELECT 1
                  FROM pg_catalog.pg_default_acl AS d
                  CROSS JOIN LATERAL pg_catalog.aclexplode(d.defaclacl) AS a
                 WHERE d.defaclrole = propietario
                   AND d.defaclnamespace = esquema
                   AND d.defaclobjtype = clase
                   AND a.grantee = rol
            ) THEN
                RAISE EXCEPTION
                    'ACL por defecto abierta para clase % y grantee %',
                    clase, rol;
            END IF;
        END LOOP;
    END LOOP;
END
$acl_por_defecto$;

DO $funciones$
DECLARE
    firma regprocedure;
BEGIN
    FOREACH firma IN ARRAY ARRAY[
        'vec_bolsa_convocatorias.consultar_identidades_borrador_v1(jsonb)'::regprocedure,
        'vec_bolsa_convocatorias.reconciliar_operacion_borrador_v1(jsonb,text,bigint,bigint,timestamp with time zone)'::regprocedure,
        'vec_bolsa_convocatorias.reservar_decision_borrador_v1(jsonb,jsonb,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.preparar_confirmacion_borrador_v1(jsonb,jsonb,jsonb,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.confirmar_borrador_v1(text,jsonb,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.verificar_recibo_borrador_v1(text,text,text)'::regprocedure,
        'vec_bolsa_convocatorias.reclamar_reserva_borrador_v1(bigint,bigint,jsonb,jsonb,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.listar_borradores_v1(jsonb,jsonb,jsonb,bytea,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.obtener_borrador_v1(text,jsonb,jsonb,bytea,bytea)'::regprocedure
    ] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_proc
             WHERE oid = firma AND prosecdef
               AND proconfig @> ARRAY[
                   'search_path=pg_catalog, pg_temp', 'TimeZone=UTC'
               ]::text[]
        ) THEN
            RAISE EXCEPTION 'wrapper sin SECURITY DEFINER cerrado: %', firma;
        END IF;
    END LOOP;
    FOREACH firma IN ARRAY ARRAY[
        'vec_bolsa_convocatorias.reservar_decision_borrador_interna_v1(jsonb,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.confirmar_borrador_interna_v1(jsonb,bytea,bytea,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.reclamar_reserva_borrador_interna_v1(bigint,bigint,jsonb,bytea,bytea,bytea,bytea)'::regprocedure,
        'vec_bolsa_convocatorias.reconciliar_operacion_borrador_interna_v1(jsonb,text,bigint,bigint,timestamp with time zone)'::regprocedure,
        'vec_bolsa_convocatorias.consultar_diario_borrador_interna_v1(jsonb)'::regprocedure,
        'vec_bolsa_convocatorias.consultar_identidades_borrador_interna_v1(jsonb)'::regprocedure,
        'vec_bolsa_convocatorias.listar_borradores_interna_v1(jsonb,jsonb)'::regprocedure,
        'vec_bolsa_convocatorias.obtener_borrador_interna_v1(text,jsonb)'::regprocedure
    ] LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_proc
             WHERE oid = firma AND NOT prosecdef
               AND proconfig @> ARRAY[
                   'search_path=pg_catalog, pg_temp', 'TimeZone=UTC'
               ]::text[]
        ) THEN
            RAISE EXCEPTION 'nucleo interno no conserva INVOKER: %', firma;
        END IF;
    END LOOP;
END
$funciones$;

DO $rls$
BEGIN
    IF (SELECT count(*)
          FROM pg_catalog.pg_class AS c
          JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_bolsa_convocatorias'
           AND c.relname IN (
               'atestacion_pdp_borrador','material_borrador',
               'diario_borrador_version','diario_borrador_actual',
               'identidad_alias_borrador',
               'prueba_desenlace_borrador','uso_decision_borrador',
               'sellado_motivo_borrador',
               'borrador_convocatoria_version',
               'borrador_convocatoria_actual','auditoria_borrador',
               'auditoria_borrador_actual','outbox_borrador',
               'uso_decision_lectura_borrador',
               'auditoria_lectura_borrador','cursor_listado_borrador',
               'preparacion_confirmacion_kms_borrador',
               'cifrado_kms_borrador','acreditacion_kms_borrador'
           )
           AND c.relrowsecurity AND c.relforcerowsecurity) <> 19 THEN
        RAISE EXCEPTION 'inventario RLS 000003/000004 incompleto';
    END IF;
END
$rls$;
