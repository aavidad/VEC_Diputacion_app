-- Retirada protegida de la prueba autoritativa de consumo de consultas RRHH.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000006', 0
    )
);

LOCK TABLE
    vec_autorizacion_atestada_v3.revocacion_clave_capacidad,
    vec_autorizacion_atestada_v3.revocacion_configuracion,
    vec_autorizacion_atestada_v3.revocacion_raiz
IN ACCESS EXCLUSIVE MODE;

ALTER FUNCTION
    vec_autorizacion_atestada_v3
    .serializar_revocacion_consultas_rrhh_v3()
OWNER TO vec_autorizacion_atestada_v3_propietario;
ALTER FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
        text, bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
OWNER TO vec_autorizacion_atestada_v3_propietario;
ALTER FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
OWNER TO vec_autorizacion_atestada_v3_propietario;
ALTER FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
OWNER TO vec_autorizacion_atestada_v3_propietario;

DO $prevalidacion$
DECLARE
    v_interna oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_cuadro oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_detalle oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_serializar oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()'
    );
    v_triggers oid[];
    v_propietario oid;
    v_propietario_ct oid;
    v_resultado text :=
        'TABLE(decision_ref text, efecto_ref text, '
        || 'huella_efecto_sha256 text, consumo_huella_sha256 text, '
        || 'auditoria_ref text, auditoria_huella_sha256 text, '
        || 'consumida_en timestamp with time zone, '
        || 'revalidada_en timestamp with time zone)';
BEGIN
    SELECT oid INTO v_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v3_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    SELECT oid INTO v_propietario_ct
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_contratacion_temporal_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    SELECT pg_catalog.array_agg(g.oid ORDER BY c.relname)
      INTO v_triggers
      FROM pg_catalog.pg_trigger g
      JOIN pg_catalog.pg_class c ON c.oid = g.tgrelid
      JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
     WHERE n.nspname = 'vec_autorizacion_atestada_v3'
       AND c.relname = ANY(ARRAY[
           'revocacion_clave_capacidad',
           'revocacion_configuracion',
           'revocacion_raiz'
       ])
       AND g.tgname = 'serializar_revalidacion_rrhh_antes'
       AND NOT g.tgisinternal
       AND g.tgfoid = v_serializar
       AND g.tgtype = 7
       AND g.tgenabled = 'O'
       AND g.tgnargs = 0
       AND g.tgparentid = 0
       AND g.tgconstraint = 0
       AND g.tgconstrrelid = 0
       AND g.tgconstrindid = 0
       AND NOT g.tgdeferrable
       AND NOT g.tginitdeferred
       AND g.tgattr = ''::pg_catalog.int2vector
       AND g.tgargs = '\x'::bytea
       AND g.tgqual IS NULL
       AND g.tgoldtable IS NULL
       AND g.tgnewtable IS NULL
       AND pg_catalog.pg_get_triggerdef(g.oid, false) =
           pg_catalog.format(
               'CREATE TRIGGER serializar_revalidacion_rrhh_antes '
               || 'BEFORE INSERT ON vec_autorizacion_atestada_v3.%I '
               || 'FOR EACH ROW EXECUTE FUNCTION '
               || 'vec_autorizacion_atestada_v3.'
               || 'serializar_revocacion_consultas_rrhh_v3()',
               c.relname
           )
       AND pg_catalog.obj_description(
               g.oid, 'pg_catalog.pg_trigger'
           ) IS NULL
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_seclabels s
            WHERE s.classoid = 'pg_catalog.pg_trigger'::regclass
              AND s.objoid = g.oid
       );

    IF v_interna IS NULL OR v_cuadro IS NULL OR v_detalle IS NULL
       OR v_serializar IS NULL
       OR COALESCE(pg_catalog.cardinality(v_triggers), 0) <> 3
       OR v_propietario IS NULL OR v_propietario_ct IS NULL
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc f
            WHERE f.oid = ANY(ARRAY[v_interna, v_cuadro, v_detalle])
       ) <> 3
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
             JOIN pg_catalog.pg_language l ON l.oid = f.prolang
            WHERE f.oid = ANY(ARRAY[v_interna, v_cuadro, v_detalle])
              AND (
                  f.proowner <> v_propietario
                  OR l.lanname <> 'plpgsql'
                  OR NOT f.prosecdef
                  OR f.provolatile <> 'v'
                  OR f.proisstrict
                  OR f.proleakproof
                  OR f.proparallel <> 'u'
                  OR f.prokind <> 'f'
                  OR NOT f.proretset
                  OR f.prorettype <> 'pg_catalog.record'::regtype
                  OR f.proconfig IS DISTINCT FROM
                     ARRAY[
                         'search_path=pg_catalog',
                         'lock_timeout=1s'
                     ]::text[]
                  OR pg_catalog.pg_get_function_result(f.oid) <>
                     v_resultado
                  OR pg_catalog.encode(
                         pg_catalog.sha256(
                             pg_catalog.convert_to(f.prosrc, 'UTF8')
                         ),
                         'hex'
                     ) IS DISTINCT FROM CASE f.oid
                     WHEN v_interna THEN
                         '4cc79858a566d9b31f31729ee48bbde7a11e962b73062f8db3beacf3e96d632f'
                     WHEN v_cuadro THEN
                         '0a225931a597f0d84a65ee5e8f0920560dce7330d647091b6e246c485f5caf8a'
                     WHEN v_detalle THEN
                         'fa5a9b2732a97a1afa579687108dedfafd64f86d95d0dfecd1b79a01fcff5e65'
                     END
                  OR pg_catalog.obj_description(
                         f.oid, 'pg_catalog.pg_proc'
                     ) IS NOT NULL
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_seclabels s
            WHERE s.classoid = 'pg_catalog.pg_proc'::regclass
              AND s.objoid = ANY(ARRAY[
                  v_interna, v_cuadro, v_detalle, v_serializar
              ])
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'catálogo de prueba de consumo RRHH divergente';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
          JOIN pg_catalog.pg_language l ON l.oid = f.prolang
         WHERE f.oid = v_serializar
           AND (
               f.proowner <> v_propietario
               OR l.lanname <> 'plpgsql'
               OR NOT f.prosecdef
               OR f.provolatile <> 'v'
               OR f.proisstrict
               OR f.proleakproof
               OR f.proparallel <> 'u'
               OR f.prokind <> 'f'
               OR f.proretset
               OR f.prorettype <> 'pg_catalog.trigger'::regtype
               OR f.proconfig IS DISTINCT FROM
                  ARRAY['search_path=pg_catalog']::text[]
               OR pg_catalog.encode(
                      pg_catalog.sha256(
                          pg_catalog.convert_to(f.prosrc, 'UTF8')
                      ),
                      'hex'
                  ) <>
                  '296a4a80d06c0d1f4668601eaf8690131215f547986430ff1be01286ed0a95eb'
               OR pg_catalog.obj_description(
                      f.oid, 'pg_catalog.pg_proc'
                  ) IS NOT NULL
           )
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'catálogo de serialización RRHH divergente';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
         WHERE f.oid = ANY(ARRAY[v_interna, v_serializar])
           AND (
               SELECT pg_catalog.count(*)
                 FROM pg_catalog.aclexplode(
                     COALESCE(
                         f.proacl,
                         pg_catalog.acldefault('f', f.proowner)
                     )
                 ) a
                WHERE a.privilege_type = 'EXECUTE'
           ) <> 1
    )
       OR (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc f
         CROSS JOIN LATERAL pg_catalog.aclexplode(
             COALESCE(
                 f.proacl,
                 pg_catalog.acldefault('f', f.proowner)
             )
         ) a
         WHERE f.oid = ANY(ARRAY[v_interna, v_serializar])
           AND a.privilege_type = 'EXECUTE'
    ) <> 2
       OR NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    f.proacl,
                    pg_catalog.acldefault('f', f.proowner)
                )
            ) a
            WHERE f.oid = ANY(ARRAY[v_interna, v_serializar])
              AND a.grantee = v_propietario
              AND a.grantor = v_propietario
              AND a.privilege_type = 'EXECUTE'
              AND NOT a.is_grantable
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    f.proacl,
                    pg_catalog.acldefault('f', f.proowner)
                )
            ) a
            WHERE f.oid = ANY(ARRAY[v_interna, v_serializar])
              AND (
                  a.grantee <> v_propietario
                  OR a.grantor <> v_propietario
                  OR a.privilege_type <> 'EXECUTE'
                  OR a.is_grantable
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'ACL interna de prueba de consumo RRHH divergente';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
         WHERE f.oid = ANY(ARRAY[v_cuadro, v_detalle])
           AND (
               SELECT pg_catalog.count(*)
                 FROM pg_catalog.aclexplode(
                     COALESCE(
                         f.proacl,
                         pg_catalog.acldefault('f', f.proowner)
                     )
                 ) a
                WHERE a.privilege_type = 'EXECUTE'
           ) <> 2
    )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    f.proacl,
                    pg_catalog.acldefault('f', f.proowner)
                )
            ) a
            WHERE f.oid = ANY(ARRAY[v_cuadro, v_detalle])
              AND (
                  a.grantee NOT IN (v_propietario, v_propietario_ct)
                  OR a.grantor <> v_propietario
                  OR a.privilege_type <> 'EXECUTE'
                  OR a.is_grantable
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            WHERE f.oid = ANY(ARRAY[v_cuadro, v_detalle])
              AND NOT pg_catalog.has_function_privilege(
                  v_propietario_ct, f.oid, 'EXECUTE'
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'ACL nominal de prueba de consumo RRHH divergente';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc f
          JOIN pg_catalog.pg_namespace n ON n.oid = f.pronamespace
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND f.proname ~
               '^revalidar_evidencia_consumo_consulta_.*rrhh_v3'
           AND f.oid <> ALL(ARRAY[v_interna, v_cuadro, v_detalle])
    )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
             JOIN pg_catalog.pg_namespace n ON n.oid = f.pronamespace
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND f.proname =
                  'serializar_revocacion_consultas_rrhh_v3'
              AND f.oid <> v_serializar
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_depend d
            WHERE d.refclassid = 'pg_catalog.pg_proc'::regclass
              AND d.refobjid = ANY(ARRAY[
                  v_interna, v_cuadro, v_detalle, v_serializar
              ])
              AND NOT (
                  d.classid = 'pg_catalog.pg_proc'::regclass
                  AND d.objid = ANY(ARRAY[
                      v_interna, v_cuadro, v_detalle, v_serializar
                  ])
              )
              AND NOT (
                  d.classid = 'pg_catalog.pg_trigger'::regclass
                  AND d.objid = ANY(v_triggers)
              )
              AND d.deptype NOT IN ('i', 'e')
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_depend d
            WHERE d.classid = 'pg_catalog.pg_trigger'::regclass
              AND d.objid = ANY(v_triggers)
       ) <> 6
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_trigger g
             JOIN pg_catalog.pg_depend d
               ON d.classid = 'pg_catalog.pg_trigger'::regclass
              AND d.objid = g.oid
            WHERE g.oid = ANY(v_triggers)
              AND NOT (
                  d.refclassid = 'pg_catalog.pg_proc'::regclass
                  AND d.refobjid = v_serializar
                  AND d.deptype = 'n'
              )
              AND NOT (
                  d.refclassid = 'pg_catalog.pg_class'::regclass
                  AND d.refobjid = g.tgrelid
                  AND d.deptype = 'a'
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            WHERE f.oid <> ALL(ARRAY[
                      v_interna, v_cuadro, v_detalle, v_serializar
                  ])
              AND f.prokind IN ('f', 'p')
              AND (
                  pg_catalog.pg_get_functiondef(f.oid) LIKE
                    '%revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada%'
                  OR pg_catalog.pg_get_functiondef(f.oid) LIKE
                    '%revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada%'
                  OR pg_catalog.pg_get_functiondef(f.oid) LIKE
                    '%revalidar_evidencia_consumo_consulta_rrhh_v3_interna%'
                  OR pg_catalog.pg_get_functiondef(f.oid) LIKE
                    '%serializar_revocacion_consultas_rrhh_v3%'
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '2BP01',
            MESSAGE = 'dependencias impiden retirar prueba de consumo RRHH';
    END IF;
END
$prevalidacion$;

DROP TRIGGER serializar_revalidacion_rrhh_antes ON
    vec_autorizacion_atestada_v3.revocacion_clave_capacidad;
DROP TRIGGER serializar_revalidacion_rrhh_antes ON
    vec_autorizacion_atestada_v3.revocacion_configuracion;
DROP TRIGGER serializar_revalidacion_rrhh_antes ON
    vec_autorizacion_atestada_v3.revocacion_raiz;

DROP FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
RESTRICT;
DROP FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
RESTRICT;
DROP FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
        text, bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
RESTRICT;
DROP FUNCTION
    vec_autorizacion_atestada_v3
    .serializar_revocacion_consultas_rrhh_v3()
RESTRICT;

COMMIT;
