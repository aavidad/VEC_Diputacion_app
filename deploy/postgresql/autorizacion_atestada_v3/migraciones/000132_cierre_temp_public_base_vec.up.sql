\set ON_ERROR_STOP on
-- AD3-132. Reparación de ACL de DATABASE posterior al lote funcional H6.
-- Requiere aprobación DBA explícita entregada por aplicar_000132_temp_public.py.
-- No modifica funciones, objetos de negocio, roles, membresías ni CONNECT/CREATE.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.set_config('vec.h6_temp_132.approval', :'h6_approval', true);
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:database:postgres:temp-public:000132', 0));

DO $repair$
DECLARE
    approved jsonb := pg_catalog.current_setting('vec.h6_temp_132.approval')::jsonb;
    db_before record;
    db_after record;
    roles_hash text;
    roles_hash_after text;
    connect_roles jsonb;
    login_roles jsonb;
    acl_after jsonb;
    acl_expected jsonb;
    anchor_count integer;
    schemas text[];
    public_relations text[];
    public_functions text[];
    extensions text[];
BEGIN
    SELECT oid, datname, datdba, datallowconn, datacl,
           pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
               pg_catalog.jsonb_build_object('oid', oid, 'name', datname,
                 'owner', datdba, 'allowconn', datallowconn,
                 'acl', datacl::text)::text, 'UTF8')), 'hex') AS acl_hash
      INTO STRICT db_before
      FROM pg_catalog.pg_database WHERE datname = pg_catalog.current_database();

    -- Anclas de identidad VEC: objetos y propietarios de módulos independientes.
    SELECT count(*) INTO anchor_count
      FROM pg_catalog.pg_namespace n
      JOIN pg_catalog.pg_roles r ON r.oid = n.nspowner
     WHERE (n.nspname = 'vec_autorizacion_atestada_v3'
            AND r.rolname = 'vec_autorizacion_atestada_v3_propietario')
        OR (n.nspname = 'vec_contratacion_temporal'
            AND r.rolname = 'vec_contratacion_temporal_propietario')
        OR (n.nspname = 'vec_identidad_sesiones_v1'
            AND r.rolname = 'vec_identidad_sesiones_v1_propietario');
    SELECT pg_catalog.array_agg(n.nspname || '|' || r.rolname ORDER BY n.nspname)
      INTO schemas FROM pg_catalog.pg_namespace n
      JOIN pg_catalog.pg_roles r ON r.oid=n.nspowner
     WHERE n.nspname <> 'public' AND n.nspname <> 'information_schema'
       AND n.nspname !~ '^pg_';
    SELECT pg_catalog.array_agg(c.relkind::text || ':' || c.relname || ':' || r.rolname
                                ORDER BY c.relkind::text, c.relname)
      INTO public_relations FROM pg_catalog.pg_class c
      JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      JOIN pg_catalog.pg_roles r ON r.oid=c.relowner
     WHERE n.nspname='public';
    SELECT pg_catalog.array_agg(p.proname ORDER BY p.proname)
      INTO public_functions FROM pg_catalog.pg_proc p
      JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='public' AND NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_depend d JOIN pg_catalog.pg_extension e ON e.oid=d.refobjid
        WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=p.oid
          AND d.refclassid='pg_catalog.pg_extension'::regclass
          AND d.deptype='e' AND e.extname='pgcrypto');
    SELECT pg_catalog.array_agg(e.extname || ':' || n.nspname ORDER BY e.extname)
      INTO extensions FROM pg_catalog.pg_extension e
      JOIN pg_catalog.pg_namespace n ON n.oid=e.extnamespace;

    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.jsonb_build_object(
        'roles', (SELECT coalesce(pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(oid, rolname, rolsuper, rolinherit,
              rolcreaterole, rolcreatedb, rolcanlogin, rolreplication,
              rolbypassrls, rolconnlimit, rolvaliduntil, rolconfig)
            ORDER BY oid), '[]'::jsonb) FROM pg_catalog.pg_roles),
        'memberships', (SELECT coalesce(pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(roleid, member, grantor, admin_option,
              inherit_option, set_option) ORDER BY roleid, member, grantor),
            '[]'::jsonb) FROM pg_catalog.pg_auth_members),
        'role_settings', (SELECT coalesce(pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(setdatabase, setrole, setconfig)
            ORDER BY setdatabase, setrole), '[]'::jsonb)
            FROM pg_catalog.pg_db_role_setting)
      )::text, 'UTF8')), 'hex') INTO roles_hash;

    SELECT coalesce(pg_catalog.jsonb_agg(rolname ORDER BY rolname), '[]'::jsonb)
      INTO connect_roles FROM pg_catalog.pg_roles
     WHERE rolcanlogin AND pg_catalog.has_database_privilege(oid, db_before.oid, 'CONNECT');
    SELECT coalesce(pg_catalog.jsonb_agg(rolname ORDER BY rolname), '[]'::jsonb)
      INTO login_roles FROM pg_catalog.pg_roles WHERE rolcanlogin;

    IF pg_catalog.current_database() <> 'postgres'
       OR db_before.datallowconn IS NOT TRUE
       OR db_before.datdba <> (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = session_user)
       OR current_user <> session_user
       OR current_setting('role') <> 'none'
       OR (SELECT rolsuper AND rolcanlogin FROM pg_catalog.pg_roles
            WHERE rolname = session_user) IS NOT TRUE
       OR anchor_count <> 3
       OR schemas IS DISTINCT FROM ARRAY[
          'vec_aspirantes|vec_aspirantes_propietario',
          'vec_autorizacion|vec_autorizacion_propietario',
          'vec_autorizacion_atestada_v3|vec_autorizacion_atestada_v3_propietario',
          'vec_bolsa_importacion_convoca|vec_bolsa_importacion_convoca_propietario',
          'vec_bolsa_llamamientos|vec_bolsa_llamamientos_propietario',
          'vec_calendarios|vec_calendarios_propietario',
          'vec_contexto_actor_v1|vec_contexto_actor_v1_propietario',
          'vec_contratacion_temporal|vec_contratacion_temporal_propietario',
          'vec_cronos_v1|vec_cronos_v1_propietario',
          'vec_dietas|vec_dietas_propietario',
          'vec_documentos|vec_documentos_propietario',
          'vec_identidad_externa_v1|vec_identidad_sesiones_v1_propietario',
          'vec_identidad_sesiones_v1|vec_identidad_sesiones_v1_propietario',
          'vec_personal|vec_personal_propietario',
          'vec_usuarios|vec_usuarios_propietario',
          'vec_usuarios_correos_avisos|vec_usuarios_correos_externo_propietario',
          'vec_usuarios_correos_externo|vec_usuarios_correos_externo_propietario',
          'vec_usuarios_correos_interno|vec_usuarios_correos_interno_propietario']
       OR public_relations IS DISTINCT FROM ARRAY[
          'i:vectores_o2_05_pkey:postgres', 'r:vectores_o2_05:postgres']
       OR public_functions IS DISTINCT FROM ARRAY[
          'aplicar_bundle_go_o2_05', 'durabilizar_decision_o2_05',
          'exportar_entrada_go_o2_05', 'invocar_vector_o2_05',
          'mutar_efecto_o2_05', 'mutar_tipo_capacidad_o2_05',
          'preparar_vector_o2_05']
       OR extensions IS DISTINCT FROM ARRAY['pgcrypto:public', 'plpgsql:pg_catalog']
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)') IS NULL
       OR approved->>'operation' IS DISTINCT FROM 'vec.h6.ad3_132.revoke_public_temp.v1'
       OR approved->>'database_name' IS DISTINCT FROM db_before.datname
       OR approved->>'database_oid' IS DISTINCT FROM db_before.oid::text
       OR approved->>'owner_name' IS DISTINCT FROM session_user
       OR approved->>'owner_oid' IS DISTINCT FROM db_before.datdba::text
       OR approved->>'acl_sha256' IS DISTINCT FROM db_before.acl_hash
       OR approved->>'role_graph_sha256' IS DISTINCT FROM roles_hash
       OR approved->'connect_roles' IS DISTINCT FROM connect_roles
       OR approved->'login_roles' IS DISTINCT FROM login_roles
       OR (approved->>'plan_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR (approved->>'sql_list_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
       OR (approved->>'source_commit' ~ '^[0-9a-f]{40}$') IS NOT TRUE
       OR approved->>'approved_by' IS DISTINCT FROM session_user
       OR (approved->>'approval_ref' ~ '^[A-Za-z0-9._:/-]{8,120}$') IS NOT TRUE
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_stat_activity a
                   WHERE a.usesysid <> db_before.datdba
                     AND a.pid <> pg_catalog.pg_backend_pid())
    THEN
       RAISE EXCEPTION 'AD3-132: identidad, autoridad o preimagen incompatible' USING ERRCODE = '55000';
    END IF;

    IF (SELECT count(*) FROM pg_catalog.aclexplode(coalesce(
          db_before.datacl, pg_catalog.acldefault('d', db_before.datdba))) a
        WHERE a.grantee = 0 AND a.privilege_type = 'TEMPORARY'
          AND NOT a.is_grantable) <> 1
    THEN
        RAISE EXCEPTION 'AD3-132: PUBLIC TEMP ausente o ACL no canónica' USING ERRCODE = '55000';
    END IF;

    SELECT coalesce(pg_catalog.jsonb_agg(
        pg_catalog.jsonb_build_array(a.grantor, a.grantee, a.privilege_type, a.is_grantable)
        ORDER BY a.grantor, a.grantee, a.privilege_type, a.is_grantable), '[]'::jsonb)
      INTO acl_expected
      FROM pg_catalog.aclexplode(coalesce(db_before.datacl,
          pg_catalog.acldefault('d', db_before.datdba))) a
     WHERE NOT (a.grantee = 0 AND a.privilege_type = 'TEMPORARY');

    EXECUTE 'REVOKE TEMPORARY ON DATABASE postgres FROM PUBLIC';

    SELECT oid, datname, datdba, datallowconn, datacl INTO STRICT db_after
      FROM pg_catalog.pg_database WHERE oid = db_before.oid;
    SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.jsonb_build_object(
        'roles', (SELECT coalesce(pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(oid, rolname, rolsuper, rolinherit,
              rolcreaterole, rolcreatedb, rolcanlogin, rolreplication,
              rolbypassrls, rolconnlimit, rolvaliduntil, rolconfig)
            ORDER BY oid), '[]'::jsonb) FROM pg_catalog.pg_roles),
        'memberships', (SELECT coalesce(pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(roleid, member, grantor, admin_option,
              inherit_option, set_option) ORDER BY roleid, member, grantor),
            '[]'::jsonb) FROM pg_catalog.pg_auth_members),
        'role_settings', (SELECT coalesce(pg_catalog.jsonb_agg(
            pg_catalog.jsonb_build_array(setdatabase, setrole, setconfig)
            ORDER BY setdatabase, setrole), '[]'::jsonb)
            FROM pg_catalog.pg_db_role_setting)
      )::text, 'UTF8')), 'hex') INTO roles_hash_after;
    SELECT coalesce(pg_catalog.jsonb_agg(
        pg_catalog.jsonb_build_array(a.grantor, a.grantee, a.privilege_type, a.is_grantable)
        ORDER BY a.grantor, a.grantee, a.privilege_type, a.is_grantable), '[]'::jsonb)
      INTO acl_after
      FROM pg_catalog.aclexplode(coalesce(db_after.datacl,
          pg_catalog.acldefault('d', db_after.datdba))) a;

    IF db_after.datname IS DISTINCT FROM db_before.datname
       OR db_after.datdba IS DISTINCT FROM db_before.datdba
       OR db_after.datallowconn IS DISTINCT FROM db_before.datallowconn
       OR roles_hash_after IS DISTINCT FROM roles_hash
       OR acl_after IS DISTINCT FROM acl_expected
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles r
                   WHERE r.rolcanlogin AND r.oid <> db_after.datdba
                     AND pg_catalog.has_database_privilege(r.oid, db_after.oid, 'TEMPORARY'))
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_stat_activity a
                   WHERE a.usesysid <> db_after.datdba
                     AND a.pid <> pg_catalog.pg_backend_pid())
       OR (SELECT coalesce(pg_catalog.jsonb_agg(rolname ORDER BY rolname), '[]'::jsonb)
             FROM pg_catalog.pg_roles WHERE rolcanlogin AND
               pg_catalog.has_database_privilege(oid, db_after.oid, 'CONNECT'))
          IS DISTINCT FROM connect_roles
    THEN
       RAISE EXCEPTION 'AD3-132: postimagen o TEMP efectivo incompatible' USING ERRCODE = '55000';
    END IF;
END $repair$;
COMMIT;
