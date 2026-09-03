-- Publicacion minima del consumidor F0-C2 a la autoridad de ContextoActor.
-- C3 no crea roles R0, no publica tablas y no corrige deriva catalogal.

DO $precondiciones_c3$
DECLARE
    v_contexto pg_catalog.oid;
    v_contratacion pg_catalog.oid;
    v_esquema pg_catalog.oid;
    v_funcion pg_catalog.oid;
    v_propietario pg_catalog.oid;
    v_total pg_catalog.int8;
    v_validos pg_catalog.int8;
BEGIN
    SELECT r.oid INTO v_propietario
      FROM pg_catalog.pg_roles AS r
     WHERE r.rolname = 'vec_autorizacion_atestada_v3_propietario';
    SELECT r.oid INTO v_contexto
      FROM pg_catalog.pg_roles AS r
     WHERE r.rolname = 'vec_contexto_actor_v1_propietario';
    SELECT r.oid INTO v_contratacion
      FROM pg_catalog.pg_roles AS r
     WHERE r.rolname = 'vec_contratacion_temporal_propietario';
    SELECT n.oid INTO v_esquema
      FROM pg_catalog.pg_namespace AS n
     WHERE n.nspname = 'vec_autorizacion_atestada_v3'
       AND n.nspowner = v_propietario;
    v_funcion := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea)'
    )::pg_catalog.oid;

    IF v_propietario IS NULL OR v_contexto IS NULL
       OR v_contratacion IS NULL OR v_esquema IS NULL
       OR v_funcion IS NULL
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc AS p
             JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
            WHERE n.oid = v_esquema
              AND p.proname =
                  'consumir_fuente_corporativa_contexto_actor_v1_atestada'
          ) <> 1
       OR NOT EXISTS (
          SELECT 1
            FROM pg_catalog.pg_proc AS p
           WHERE p.oid = v_funcion
             AND p.proowner = v_propietario
             AND p.prokind = 'f'
             AND p.prorettype = 'record'::pg_catalog.regtype
             AND p.proretset
             AND p.pronargs = 11
             AND p.pronargdefaults = 0
             AND p.proargdefaults IS NULL
             AND p.provariadic = 0
             AND pg_catalog.oidvectortypes(p.proargtypes) =
                 'text, text, text, text, text, text, bytea, bytea, bytea, bytea, bytea'
             AND p.provolatile = 'v'
             AND NOT p.proisstrict
             AND p.prosecdef
             AND NOT p.proleakproof
             AND p.proparallel = 'u'
             AND p.procost = 100
             AND p.prorows = 1000
             AND p.prosupport = 0
             AND p.protrftypes IS NULL
             AND p.probin IS NULL
             AND p.prosqlbody IS NULL
             AND p.prolang = (
                 SELECT l.oid FROM pg_catalog.pg_language AS l
                  WHERE l.lanname = 'plpgsql'
             )
             AND p.proconfig = ARRAY[
                 'search_path=pg_catalog', 'lock_timeout=2s'
             ]::pg_catalog.text[]
             AND p.proargnames = ARRAY[
                 'p_audiencia_consumo_esperada',
                 'p_accion_esperada',
                 'p_tipo_efecto_esperado',
                 'p_operacion_ref_esperada',
                 'p_efecto_ref_esperada',
                 'p_huella_efecto_sha256_esperada',
                 'p_capacidad_canonica',
                 'p_manifiesto_fuente_canonico',
                 'p_sobre_cose_sign1',
                 'p_evidencia_verificacion',
                 'p_raiz_publica_spki',
                 'capacidad_ref',
                 'fuente_ref',
                 'fuente_version',
                 'evento_fuente_ref',
                 'huella_evento_fuente_sha256',
                 'huella_manifiesto_fuente_sha256',
                 'operacion_ref',
                 'efecto_ref',
                 'huella_efecto_sha256',
                 'consumo_huella_sha256',
                 'consumida_en',
                 'consumo_nuevo'
             ]::pg_catalog.text[]
             AND p.proargmodes = ARRAY[
                 'i','i','i','i','i','i','i','i','i','i','i',
                 't','t','t','t','t','t','t','t','t','t','t','t'
             ]::pg_catalog."char"[]
             AND p.proallargtypes = ARRAY[
                 'text'::pg_catalog.regtype, 'text'::pg_catalog.regtype,
                 'text'::pg_catalog.regtype, 'text'::pg_catalog.regtype,
                 'text'::pg_catalog.regtype, 'text'::pg_catalog.regtype,
                 'bytea'::pg_catalog.regtype, 'bytea'::pg_catalog.regtype,
                 'bytea'::pg_catalog.regtype, 'bytea'::pg_catalog.regtype,
                 'bytea'::pg_catalog.regtype, 'text'::pg_catalog.regtype,
                 'text'::pg_catalog.regtype, 'numeric'::pg_catalog.regtype,
                 'text'::pg_catalog.regtype, 'text'::pg_catalog.regtype,
                 'text'::pg_catalog.regtype, 'text'::pg_catalog.regtype,
                 'text'::pg_catalog.regtype, 'text'::pg_catalog.regtype,
                 'text'::pg_catalog.regtype,
                 'timestamptz'::pg_catalog.regtype,
                 'boolean'::pg_catalog.regtype
             ]::pg_catalog.oid[]
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'C3: contrato C2 ausente o derivado';
    END IF;

    SELECT pg_catalog.count(*),
           pg_catalog.count(*) FILTER (
               WHERE a.grantor = v_propietario
                 AND a.grantee = v_propietario
                 AND a.privilege_type = 'EXECUTE'
                 AND NOT a.is_grantable
           )
      INTO v_total, v_validos
      FROM pg_catalog.pg_proc AS p
      CROSS JOIN LATERAL pg_catalog.aclexplode(
          COALESCE(
              p.proacl,
              pg_catalog.acldefault('f', p.proowner)
          )
      ) AS a
     WHERE p.oid = v_funcion;
    IF (v_total, v_validos) <>
       (1::pg_catalog.int8, 1::pg_catalog.int8) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'C3: ACL previa de C2 derivada';
    END IF;

    SELECT pg_catalog.count(*),
           pg_catalog.count(*) FILTER (
               WHERE a.grantor = v_propietario
                 AND NOT a.is_grantable
                 AND (
                    (a.grantee = v_propietario AND
                     a.privilege_type IN ('CREATE', 'USAGE'))
                    OR
                    (a.grantee = v_contratacion AND
                     a.privilege_type = 'USAGE')
                 )
           )
      INTO v_total, v_validos
      FROM pg_catalog.pg_namespace AS n
      CROSS JOIN LATERAL pg_catalog.aclexplode(
          COALESCE(
              n.nspacl,
              pg_catalog.acldefault('n', n.nspowner)
          )
      ) AS a
     WHERE n.oid = v_esquema;
    IF (v_total, v_validos) <>
       (3::pg_catalog.int8, 3::pg_catalog.int8) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'C3: ACL previa del esquema derivada';
    END IF;

    IF EXISTS (
           SELECT 1
             FROM pg_catalog.pg_class AS c
             JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
             CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a
            WHERE n.oid = v_esquema AND a.grantee = v_contexto
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_type AS t
             JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
             CROSS JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a
            WHERE n.oid = v_esquema AND a.grantee = v_contexto
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS p
             CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
            WHERE p.pronamespace = v_esquema
              AND a.grantee = v_contexto
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_trigger AS t
            WHERE t.tgrelid =
                  'vec_autorizacion_atestada_v3.revocacion_raiz'
                      ::pg_catalog.regclass
              AND t.tgname = 'dependencia_f0_fuente_corporativa_v1'
              AND NOT t.tgisinternal
       )
       OR (
           SELECT pg_catalog.pg_get_constraintdef(c.oid, false)
             FROM pg_catalog.pg_constraint AS c
            WHERE c.conrelid =
                  'vec_autorizacion_atestada_v3.clave_capacidad_version'
                      ::pg_catalog.regclass
              AND c.conname =
                  'clave_capacidad_version_audiencia_consumo_check'
              AND c.contype = 'c'
       ) IS DISTINCT FROM
          'CHECK (audiencia_consumo = ANY (ARRAY[''vec_contratacion_temporal.confirmar_alta_atestada.v1''::text, ''vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1''::text, ''vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1''::text, ''vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1''::text, ''vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1''::text, ''vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1''::text, ''vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1''::text]))'
    THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'C3: clausura catalogal previa derivada';
    END IF;
END
$precondiciones_c3$;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
    TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3
    .consumir_fuente_corporativa_contexto_actor_v1_atestada(
        pg_catalog.text, pg_catalog.text, pg_catalog.text, pg_catalog.text,
        pg_catalog.text, pg_catalog.text, pg_catalog.bytea, pg_catalog.bytea,
        pg_catalog.bytea, pg_catalog.bytea, pg_catalog.bytea
    )
    TO vec_contexto_actor_v1_propietario;

CREATE TRIGGER dependencia_f0_fuente_corporativa_v1
BEFORE INSERT ON vec_autorizacion_atestada_v3.revocacion_raiz
FOR EACH ROW EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3();
ALTER TABLE vec_autorizacion_atestada_v3.revocacion_raiz
    DISABLE TRIGGER dependencia_f0_fuente_corporativa_v1;

DO $postcondiciones_c3$
DECLARE
    v_contexto pg_catalog.oid :=
        'vec_contexto_actor_v1_propietario'::pg_catalog.regrole;
    v_contratacion pg_catalog.oid :=
        'vec_contratacion_temporal_propietario'::pg_catalog.regrole;
    v_esquema pg_catalog.oid :=
        'vec_autorizacion_atestada_v3'::pg_catalog.regnamespace;
    v_funcion pg_catalog.oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.consumir_fuente_corporativa_contexto_actor_v1_atestada(text,text,text,text,text,text,bytea,bytea,bytea,bytea,bytea)'
    )::pg_catalog.oid;
    v_propietario pg_catalog.oid :=
        'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole;
    v_serializar pg_catalog.oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()'
    )::pg_catalog.oid;
    v_trigger pg_catalog.oid;
    v_total pg_catalog.int8;
    v_validos pg_catalog.int8;
BEGIN
    SELECT pg_catalog.count(*),
           pg_catalog.count(*) FILTER (
               WHERE a.grantor = v_propietario
                 AND NOT a.is_grantable
                 AND a.privilege_type = 'EXECUTE'
                 AND a.grantee IN (v_propietario, v_contexto)
           )
      INTO v_total, v_validos
      FROM pg_catalog.pg_proc AS p
      CROSS JOIN LATERAL pg_catalog.aclexplode(
          COALESCE(
              p.proacl,
              pg_catalog.acldefault('f', p.proowner)
          )
      ) AS a
     WHERE p.oid = v_funcion;
    IF (v_total, v_validos) <>
       (2::pg_catalog.int8, 2::pg_catalog.int8) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'C3: publicacion C2 no minima';
    END IF;

    SELECT pg_catalog.count(*),
           pg_catalog.count(*) FILTER (
               WHERE a.grantor = v_propietario
                 AND NOT a.is_grantable
                 AND (
                    (a.grantee = v_propietario AND
                     a.privilege_type IN ('CREATE', 'USAGE'))
                    OR
                    (a.grantee IN (v_contratacion, v_contexto) AND
                     a.privilege_type = 'USAGE')
                 )
           )
      INTO v_total, v_validos
      FROM pg_catalog.pg_namespace AS n
      CROSS JOIN LATERAL pg_catalog.aclexplode(
          COALESCE(
              n.nspacl,
              pg_catalog.acldefault('n', n.nspowner)
          )
      ) AS a
     WHERE n.oid = v_esquema;
    IF (v_total, v_validos) <>
       (4::pg_catalog.int8, 4::pg_catalog.int8) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'C3: publicacion de esquema no minima';
    END IF;

    SELECT t.oid INTO v_trigger
      FROM pg_catalog.pg_trigger AS t
     WHERE t.tgrelid =
           'vec_autorizacion_atestada_v3.revocacion_raiz'
               ::pg_catalog.regclass
       AND t.tgname = 'dependencia_f0_fuente_corporativa_v1'
       AND NOT t.tgisinternal
       AND t.tgfoid = v_serializar
       AND t.tgtype = 7
       AND t.tgenabled = 'D'
       AND t.tgnargs = 0
       AND t.tgparentid = 0
       AND t.tgconstraint = 0
       AND t.tgconstrrelid = 0
       AND NOT t.tgdeferrable
       AND NOT t.tginitdeferred
       AND t.tgqual IS NULL;
    IF v_trigger IS NULL
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_trigger AS t
            WHERE t.tgfoid = v_serializar AND NOT t.tgisinternal) <> 4
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_depend AS d
            WHERE d.classid = 'pg_catalog.pg_trigger'::pg_catalog.regclass
              AND d.objid = v_trigger
              AND d.objsubid = 0
              AND d.refclassid = 'pg_catalog.pg_proc'::pg_catalog.regclass
              AND d.refobjid = v_serializar
              AND d.refobjsubid = 0
              AND d.deptype = 'n') <> 1
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_class AS c
             JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
             CROSS JOIN LATERAL pg_catalog.aclexplode(c.relacl) AS a
            WHERE n.oid = v_esquema AND a.grantee = v_contexto
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_type AS t
             JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
             CROSS JOIN LATERAL pg_catalog.aclexplode(t.typacl) AS a
            WHERE n.oid = v_esquema AND a.grantee = v_contexto
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc AS p
             CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
            WHERE p.pronamespace = v_esquema
              AND p.oid <> v_funcion
              AND a.grantee = v_contexto
       )
    THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'C3: clausura catalogal posterior derivada';
    END IF;
END
$postcondiciones_c3$;
