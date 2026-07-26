\set ON_ERROR_STOP on

SELECT pg_catalog.set_config('vec_o405.caso', :'caso', false);
SELECT pg_catalog.set_config('vec_o405.accion', :'accion', false);

DO $deriva$
DECLARE
    v_accion text := pg_catalog.current_setting('vec_o405.accion');
    v_caso text := pg_catalog.current_setting('vec_o405.caso');
    v_correcto boolean;
    v_disparador name;
    v_tabla regclass;
BEGIN
    IF v_accion = 'preparar' THEN
        CASE v_caso
        WHEN 'constraint' THEN
            ALTER TABLE vec_contratacion_temporal.alcance_acceso_rrhh
                DROP CONSTRAINT
                    alcance_acceso_rrhh_acceso_ref_tipo_consulta_fkey;
            ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
                DROP CONSTRAINT registro_acceso_rrhh_cursor_tipo_unico;
            ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
                ADD CONSTRAINT registro_acceso_rrhh_cursor_tipo_unico
                CHECK (true);
        WHEN 'trigger_definicion' THEN
            DROP TRIGGER cursor_cuadro_rrhh_inmutable
                ON vec_contratacion_temporal.cursor_cuadro_rrhh;
            CREATE TRIGGER cursor_cuadro_rrhh_inmutable
                BEFORE UPDATE
                ON vec_contratacion_temporal.cursor_cuadro_rrhh
                FOR EACH ROW EXECUTE FUNCTION
                vec_contratacion_temporal.rechazar_mutacion_historia_v1();
        WHEN 'trigger_deshabilitado' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                DISABLE TRIGGER cursor_cuadro_rrhh_no_truncar;
        WHEN 'trigger_ri' THEN
            SELECT disparador.tgrelid::regclass, disparador.tgname
              INTO STRICT v_tabla, v_disparador
              FROM pg_catalog.pg_trigger disparador
              JOIN pg_catalog.pg_constraint restriccion
                ON restriccion.oid = disparador.tgconstraint
             WHERE disparador.tgisinternal
               AND restriccion.conrelid =
                   'vec_contratacion_temporal.alcance_acceso_rrhh'::regclass
               AND restriccion.conname =
                   'alcance_acceso_rrhh_acceso_ref_tipo_consulta_fkey'
             ORDER BY disparador.tgrelid, disparador.tgname COLLATE "C"
             LIMIT 1;
            EXECUTE pg_catalog.format(
                'ALTER TABLE %s DISABLE TRIGGER %I',
                v_tabla, v_disparador
            );
        WHEN 'regla' THEN
            CREATE RULE prueba_deriva_cursor_rrhh
                AS ON INSERT
                TO vec_contratacion_temporal.cursor_cuadro_rrhh
                DO ALSO NOTHING;
        WHEN 'columna' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                ADD COLUMN prueba_deriva boolean;
        WHEN 'indice' THEN
            CREATE INDEX prueba_deriva_cursor_rrhh_idx
                ON vec_contratacion_temporal.cursor_cuadro_rrhh(
                    familia_ref
                );
        WHEN 'rls' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                NO FORCE ROW LEVEL SECURITY;
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                DISABLE ROW LEVEL SECURITY;
        WHEN 'politica' THEN
            DROP POLICY propietario_total
                ON vec_contratacion_temporal.cursor_cuadro_rrhh;
            CREATE POLICY propietario_total
                ON vec_contratacion_temporal.cursor_cuadro_rrhh
                TO PUBLIC USING (true) WITH CHECK (true);
        WHEN 'acl' THEN
            GRANT SELECT
                ON vec_contratacion_temporal.cursor_cuadro_rrhh
                TO PUBLIC;
        WHEN 'propietario' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                OWNER TO postgres;
        ELSE
            RAISE EXCEPTION 'caso de deriva desconocido: %', v_caso;
        END CASE;
        RETURN;
    END IF;

    IF v_accion = 'verificar' THEN
        CASE v_caso
        WHEN 'constraint' THEN
            SELECT EXISTS (
                SELECT 1
                  FROM pg_catalog.pg_constraint
                 WHERE conrelid =
                       'vec_contratacion_temporal.registro_acceso_rrhh'
                           ::regclass
                   AND conname =
                       'registro_acceso_rrhh_cursor_tipo_unico'
                   AND contype = 'c'
            ) AND NOT EXISTS (
                SELECT 1
                  FROM pg_catalog.pg_constraint
                 WHERE conrelid =
                       'vec_contratacion_temporal.alcance_acceso_rrhh'
                           ::regclass
                   AND conname =
                       'alcance_acceso_rrhh_acceso_ref_tipo_consulta_fkey'
            ) INTO v_correcto;
        WHEN 'trigger_definicion' THEN
            SELECT pg_catalog.pg_get_triggerdef(oid, false)
                   NOT LIKE '% OR DELETE %'
              INTO v_correcto
              FROM pg_catalog.pg_trigger
             WHERE tgrelid =
                   'vec_contratacion_temporal.cursor_cuadro_rrhh'::regclass
               AND tgname = 'cursor_cuadro_rrhh_inmutable';
        WHEN 'trigger_deshabilitado' THEN
            SELECT tgenabled = 'D'
              INTO v_correcto
              FROM pg_catalog.pg_trigger
             WHERE tgrelid =
                   'vec_contratacion_temporal.cursor_cuadro_rrhh'::regclass
               AND tgname = 'cursor_cuadro_rrhh_no_truncar';
        WHEN 'trigger_ri' THEN
            SELECT EXISTS (
                SELECT 1
                  FROM pg_catalog.pg_trigger disparador
                  JOIN pg_catalog.pg_constraint restriccion
                    ON restriccion.oid = disparador.tgconstraint
                 WHERE disparador.tgisinternal
                   AND disparador.tgenabled = 'D'
                   AND restriccion.conrelid =
                       'vec_contratacion_temporal.alcance_acceso_rrhh'
                           ::regclass
                   AND restriccion.conname =
                       'alcance_acceso_rrhh_acceso_ref_tipo_consulta_fkey'
            ) INTO v_correcto;
        WHEN 'regla' THEN
            SELECT EXISTS (
                SELECT 1
                  FROM pg_catalog.pg_rewrite regla
                 WHERE regla.ev_class =
                       'vec_contratacion_temporal.cursor_cuadro_rrhh'
                           ::regclass
                   AND regla.rulename = 'prueba_deriva_cursor_rrhh'
            ) INTO v_correcto;
        WHEN 'columna' THEN
            SELECT EXISTS (
                SELECT 1
                  FROM pg_catalog.pg_attribute
                 WHERE attrelid =
                       'vec_contratacion_temporal.cursor_cuadro_rrhh'
                           ::regclass
                   AND attname = 'prueba_deriva'
                   AND attnum > 0
                   AND NOT attisdropped
            ) INTO v_correcto;
        WHEN 'indice' THEN
            SELECT pg_catalog.to_regclass(
                'vec_contratacion_temporal.prueba_deriva_cursor_rrhh_idx'
            ) IS NOT NULL INTO v_correcto;
        WHEN 'rls' THEN
            SELECT NOT relrowsecurity AND NOT relforcerowsecurity
              INTO v_correcto
              FROM pg_catalog.pg_class
             WHERE oid =
                   'vec_contratacion_temporal.cursor_cuadro_rrhh'::regclass;
        WHEN 'politica' THEN
            SELECT roles = ARRAY['public']::name[]
              INTO v_correcto
              FROM pg_catalog.pg_policies
             WHERE schemaname = 'vec_contratacion_temporal'
               AND tablename = 'cursor_cuadro_rrhh'
               AND policyname = 'propietario_total';
        WHEN 'acl' THEN
            SELECT EXISTS (
                SELECT 1
                  FROM pg_catalog.pg_class tabla
                 CROSS JOIN LATERAL pg_catalog.aclexplode(
                     tabla.relacl
                 ) privilegio
                 WHERE tabla.oid =
                       'vec_contratacion_temporal.cursor_cuadro_rrhh'
                           ::regclass
                   AND privilegio.grantee = 0
                   AND privilegio.privilege_type = 'SELECT'
            ) INTO v_correcto;
        WHEN 'propietario' THEN
            SELECT propietario.rolname = 'postgres'
              INTO v_correcto
              FROM pg_catalog.pg_class tabla
              JOIN pg_catalog.pg_roles propietario
                ON propietario.oid = tabla.relowner
             WHERE tabla.oid =
                   'vec_contratacion_temporal.cursor_cuadro_rrhh'::regclass;
        ELSE
            RAISE EXCEPTION 'caso de deriva desconocido: %', v_caso;
        END CASE;
        IF v_correcto IS DISTINCT FROM true THEN
            RAISE EXCEPTION 'down mutó la deriva probada: %', v_caso;
        END IF;
        RETURN;
    END IF;

    IF v_accion = 'restaurar' THEN
        CASE v_caso
        WHEN 'constraint' THEN
            ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
                DROP CONSTRAINT registro_acceso_rrhh_cursor_tipo_unico;
            ALTER TABLE vec_contratacion_temporal.registro_acceso_rrhh
                ADD CONSTRAINT registro_acceso_rrhh_cursor_tipo_unico
                UNIQUE (acceso_ref, tipo_consulta);
            ALTER TABLE vec_contratacion_temporal.alcance_acceso_rrhh
                ADD CONSTRAINT
                    alcance_acceso_rrhh_acceso_ref_tipo_consulta_fkey
                FOREIGN KEY (acceso_ref, tipo_consulta)
                REFERENCES vec_contratacion_temporal.registro_acceso_rrhh(
                    acceso_ref, tipo_consulta
                ) ON UPDATE RESTRICT ON DELETE RESTRICT;
        WHEN 'trigger_definicion' THEN
            DROP TRIGGER cursor_cuadro_rrhh_inmutable
                ON vec_contratacion_temporal.cursor_cuadro_rrhh;
            CREATE TRIGGER cursor_cuadro_rrhh_inmutable
                BEFORE UPDATE OR DELETE
                ON vec_contratacion_temporal.cursor_cuadro_rrhh
                FOR EACH ROW EXECUTE FUNCTION
                vec_contratacion_temporal.rechazar_mutacion_historia_v1();
        WHEN 'trigger_deshabilitado' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                ENABLE TRIGGER cursor_cuadro_rrhh_no_truncar;
        WHEN 'columna' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                DROP COLUMN prueba_deriva;
        WHEN 'indice' THEN
            DROP INDEX
                vec_contratacion_temporal.prueba_deriva_cursor_rrhh_idx;
        WHEN 'rls' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                ENABLE ROW LEVEL SECURITY;
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                FORCE ROW LEVEL SECURITY;
        WHEN 'politica' THEN
            DROP POLICY propietario_total
                ON vec_contratacion_temporal.cursor_cuadro_rrhh;
            CREATE POLICY propietario_total
                ON vec_contratacion_temporal.cursor_cuadro_rrhh
                TO vec_contratacion_temporal_propietario
                USING (true) WITH CHECK (true);
        WHEN 'acl' THEN
            REVOKE SELECT
                ON vec_contratacion_temporal.cursor_cuadro_rrhh
                FROM PUBLIC;
        WHEN 'propietario' THEN
            ALTER TABLE vec_contratacion_temporal.cursor_cuadro_rrhh
                OWNER TO vec_contratacion_temporal_propietario;
        ELSE
            RAISE EXCEPTION 'caso de deriva desconocido: %', v_caso;
        END CASE;
        RETURN;
    END IF;

    RAISE EXCEPTION 'acción de deriva desconocida: %', v_accion;
END
$deriva$;
