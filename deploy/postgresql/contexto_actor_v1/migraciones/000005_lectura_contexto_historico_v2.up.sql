-- Historia original Contexto V2. Sin tablas, punteros vivos ni nuevas sesiones.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:base:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:migracion:lectura_historica:v2',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL timezone='UTC';

DO $pre$
DECLARE r oid:='vec_contexto_actor_v1_lector_historico'::regrole;
BEGIN
 IF to_regprocedure('vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_contexto_actor_v1'::regnamespace AND proname='leer_contexto_original_v2')
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE oid='vec_contexto_actor_v1'::regnamespace AND nspowner=current_user::regrole)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=r AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
   AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
   AND rolconfig IS NULL AND rolconnlimit=-1 AND rolvaliduntil IS NULL)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) a
   WHERE n.oid='vec_contexto_actor_v1'::regnamespace AND a.grantee=r)
 OR (SELECT count(*) FROM pg_class WHERE relnamespace='vec_contexto_actor_v1'::regnamespace
   AND relkind='r' AND relowner=current_user::regrole AND relname=ANY(ARRAY[
     'registros_contexto','proyeccion_cuenta_versiones','persona_versiones','perfil_versiones',
     'vinculo_contexto_versiones','vinculo_referencia_versiones']))<>6
 OR to_regprocedure('vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(oid,oid,oid,oid[])') IS NULL
 THEN RAISE EXCEPTION 'precondiciones lector historico no satisfechas' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contexto_actor_v1.leer_contexto_original_v2(
 p_registro_ref text, p_contexto_sha256 text, p_procedencia_sha256 text
) RETURNS TABLE(
 registro_contexto_ref text, representacion_canonica bytea, huella_sha256 text,
 manifiesto_procedencia_canonico bytea, manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text, resuelto_en timestamptz
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog
SET statement_timeout='5s'
SET lock_timeout='2s'
AS $historia$
DECLARE

    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;

    registro record; cuenta record; perfil record; persona record; enlace record;
    doc jsonb; item jsonb; clave text;
    numero_enlaces integer; tipos integer; referencias integer;
    enlaces_texto text; enlaces_procedencia_texto text;
    manifiesto_procedencia text; manifiesto_procedencia_bytes bytea;
    manifiesto_procedencia_huella text; documento text; canonica bytea;
    inicio timestamptz := pg_catalog.clock_timestamp();
BEGIN
    IF current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'on'
       OR current_setting('TimeZone') <> 'UTC' THEN
        RAISE EXCEPTION 'lector historico requiere SERIALIZABLE READ ONLY UTC' USING ERRCODE='25000';
    END IF;

    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_contexto_actor_v1_lector_historico';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[pg_catalog.to_regprocedure('vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)')];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 1 OR array_position(funciones,NULL) IS NOT NULL
       OR login.rolcanlogin IS NOT TRUE
       OR login.rolsuper OR NOT login.rolinherit OR login.rolcreaterole OR login.rolcreatedb
       OR login.rolreplication OR login.rolbypassrls OR login.rolconfig IS NOT NULL
       OR grupo.rolcanlogin OR grupo.rolsuper OR grupo.rolinherit
       OR grupo.rolcreaterole OR grupo.rolcreatedb OR grupo.rolreplication
       OR grupo.rolbypassrls OR grupo.rolconfig IS NOT NULL
       OR current_setting('role') <> 'none' OR membresias <> 1
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>login_oid
              AND pg_catalog.pg_has_role(login_oid,r.oid,'MEMBER')
              AND r.oid<>runtime_oid
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.oid<>runtime_oid
              AND pg_catalog.pg_has_role(runtime_oid,r.oid,'MEMBER')
       )
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = runtime_oid
              AND admin_option IS FALSE AND inherit_option IS TRUE AND set_option IS FALSE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_db_role_setting s
            WHERE s.setrole IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_default_acl d
           LEFT JOIN LATERAL pg_catalog.aclexplode(
             coalesce(d.defaclacl,'{}'::aclitem[])
           ) a ON true
            WHERE d.defaclrole IN (login_oid,runtime_oid)
               OR a.grantee IN (login_oid,runtime_oid)
               OR a.grantor IN (login_oid,runtime_oid)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_policy p
            WHERE login_oid=ANY(p.polroles) OR runtime_oid=ANY(p.polroles)
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=login_oid
       )
       OR NOT COALESCE((
           SELECT count(*)=3 AND bool_and(
             d.deptype='a' AND d.objsubid=0 AND (
               (d.classid='pg_catalog.pg_database'::regclass AND d.objid=base_oid) OR
               (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=esquema_oid) OR
               (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(funciones))
             ))
             FROM pg_catalog.pg_shdepend d
            WHERE d.refclassid='pg_catalog.pg_authid'::regclass
              AND d.refobjid=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
             FROM pg_catalog.pg_database b
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(b.datacl,pg_catalog.acldefault('d',b.datdba))
             ) a
            WHERE b.oid=base_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_namespace n
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(n.nspacl,pg_catalog.acldefault('n',n.nspowner))
             ) a
            WHERE n.oid=esquema_oid AND a.grantee=runtime_oid
       ),false)
       OR NOT COALESCE((
           SELECT count(*)=1 AND count(DISTINCT p.oid)=1
                  AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))
            ) a
            WHERE p.oid=ANY(funciones) AND a.grantee=runtime_oid
       ),false)
       OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(
            login_oid,base_oid,esquema_oid,funciones) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN lector historico no acreditado';
    END IF;

    IF vec_contexto_actor_v1.referencia_operacion_valida(p_registro_ref,'rca_') IS NOT TRUE
       OR p_contexto_sha256 IS NULL OR p_contexto_sha256 !~ '^[0-9a-f]{64}$'
       OR p_procedencia_sha256 IS NULL OR p_procedencia_sha256 !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'selector historico invalido' USING ERRCODE='22023';
    END IF;
    PERFORM pg_advisory_xact_lock_shared(hashtextextended(
        'vec_contexto_actor_v1:migracion:lectura_historica:v2',0));
    -- Sin row locks ni punteros actuales: todas las fuentes son append-only.
    SELECT r.* INTO STRICT registro FROM vec_contexto_actor_v1.registros_contexto r
    WHERE r.registro_contexto_ref=p_registro_ref
      AND r.huella_sha256=p_contexto_sha256
      AND r.manifiesto_procedencia_huella_sha256=p_procedencia_sha256;
    IF octet_length(registro.representacion_canonica) NOT BETWEEN 1 AND 65536
       OR octet_length(registro.manifiesto_procedencia_canonico) NOT BETWEEN 1 AND 65536
       OR encode(sha256(registro.representacion_canonica),'hex') IS DISTINCT FROM p_contexto_sha256
       OR encode(sha256(registro.manifiesto_procedencia_canonico),'hex') IS DISTINCT FROM p_procedencia_sha256
       OR registro.autoridad_efectiva IS DISTINCT FROM 'autoridad_maestra_acreditada'
       OR vec_contexto_actor_v1.instante_valido(registro.resuelto_en) IS NOT TRUE
       OR vec_contexto_actor_v1.instante_valido(registro.solicitado_en) IS NOT TRUE
       OR registro.resuelto_en < registro.solicitado_en
       OR registro.resuelto_en > registro.solicitado_en + interval '5 seconds' THEN
        RAISE EXCEPTION 'original historico incoherente' USING ERRCODE='22023';
    END IF;
    doc := convert_from(registro.representacion_canonica,'UTF8')::jsonb;
    IF jsonb_typeof(doc) IS DISTINCT FROM 'object'
       OR jsonb_typeof(doc->'vinculos') IS DISTINCT FROM 'array' THEN
        RAISE EXCEPTION 'canon historico invalido' USING ERRCODE='22023';
    END IF;
    IF jsonb_array_length(doc->'vinculos') > 128 THEN
        RAISE EXCEPTION 'canon historico excede cota' USING ERRCODE='22023';
    END IF;
    FOREACH clave IN ARRAY ARRAY['cuenta_version','persona_version','perfil_version','contexto_version'] LOOP
        IF jsonb_typeof(doc->clave) IS DISTINCT FROM 'number'
           OR (doc->>clave) !~ '^[1-9][0-9]{0,19}$'
           OR (doc->>clave)::numeric > 18446744073709551615::numeric THEN
            RAISE EXCEPTION 'version historica invalida' USING ERRCODE='22023';
        END IF;
    END LOOP;
    FOR item IN SELECT value FROM jsonb_array_elements(doc->'vinculos') LOOP
        IF jsonb_typeof(item) IS DISTINCT FROM 'object'
           OR jsonb_typeof(item->'version') IS DISTINCT FROM 'number'
           OR (item->>'version') !~ '^[1-9][0-9]{0,19}$'
           OR (item->>'version')::numeric > 18446744073709551615::numeric
           OR vec_contexto_actor_v1.referencia_valida(item->>'vinculo_ref','vin_') IS NOT TRUE THEN
            RAISE EXCEPTION 'vinculo historico invalido' USING ERRCODE='22023';
        END IF;
    END LOOP;
    SELECT v.* INTO STRICT cuenta FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones v
      WHERE v.cuenta_ref=registro.cuenta_ref AND v.version=(doc->>'cuenta_version')::numeric;
    SELECT v.* INTO STRICT perfil FROM vec_contexto_actor_v1.perfil_versiones v
      WHERE v.perfil_ref=registro.perfil_ref AND v.version=(doc->>'perfil_version')::numeric;
    SELECT v.* INTO STRICT persona FROM vec_contexto_actor_v1.persona_versiones v
      WHERE v.persona_ref=perfil.persona_ref AND v.version=(doc->>'persona_version')::numeric;
    SELECT v.* INTO STRICT enlace FROM vec_contexto_actor_v1.vinculo_contexto_versiones v
      WHERE v.vinculo_ref=doc->>'contexto_actor_ref' AND v.version=(doc->>'contexto_version')::numeric;
    IF enlace.cuenta_ref IS DISTINCT FROM registro.cuenta_ref
       OR enlace.perfil_ref IS DISTINCT FROM registro.perfil_ref
       OR enlace.persona_ref IS DISTINCT FROM perfil.persona_ref THEN
        RAISE EXCEPTION 'enlaces historicos divergentes' USING ERRCODE='22023';
    END IF;
    -- Vigencia en resuelto_en ORIGINAL, nunca reloj actual: no renueva permiso.
    FOR item IN SELECT x FROM (VALUES(to_jsonb(cuenta)),(to_jsonb(perfil)),
                                (to_jsonb(persona)),(to_jsonb(enlace))) t(x) LOOP
        IF item->>'estado' IS DISTINCT FROM 'activo'
           OR item->>'procedencia_autoridad' IS DISTINCT FROM 'autoridad_maestra_acreditada'
           OR registro.resuelto_en < (item->>'vigente_desde')::timestamptz
           OR registro.resuelto_en >= (item->>'vigente_hasta')::timestamptz THEN
            RAISE EXCEPTION 'fuente historica incoherente' USING ERRCODE='22023';
        END IF;
    END LOOP;
    SELECT count(*), count(DISTINCT vr.tipo), count(DISTINCT (vr.tipo,vr.referencia)),
           string_agg(format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s}',
             to_json(vr.vinculo_ref)::text, vr.version::text, to_json(vr.tipo)::text,
             to_json(vr.referencia)::text, to_json(vr.estado)::text,
             to_json(to_char(vr.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
             to_json(to_char(vr.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text
           ), ',' ORDER BY vr.tipo,vr.referencia,vr.version,vr.vinculo_ref),
           string_agg(format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',
             to_json(vr.vinculo_ref)::text,vr.version::text,to_json(vr.tipo)::text,
             to_json(vr.referencia)::text,to_json(vr.procedencia_ref)::text,
             vr.procedencia_version::text,to_json(vr.procedencia_huella_sha256)::text,
             to_json(vr.procedencia_autoridad)::text
           ), ',' ORDER BY vr.tipo,vr.referencia,vr.version,vr.vinculo_ref)
      INTO numero_enlaces, tipos, referencias, enlaces_texto,enlaces_procedencia_texto
      FROM pg_catalog.jsonb_array_elements(doc->'vinculos') AS j(e)
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr
        ON vr.vinculo_ref=j.e->>'vinculo_ref' AND vr.version=(j.e->>'version')::numeric
     WHERE vr.persona_ref=perfil.persona_ref;

    IF numero_enlaces <> jsonb_array_length(doc->'vinculos')
       OR tipos <> numero_enlaces OR referencias <> numero_enlaces
       OR EXISTS (
         SELECT 1 FROM jsonb_array_elements(doc->'vinculos') j(e)
         JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v
           ON v.vinculo_ref=j.e->>'vinculo_ref' AND v.version=(j.e->>'version')::numeric
         WHERE v.persona_ref IS DISTINCT FROM perfil.persona_ref
            OR v.estado <> 'activo' OR v.procedencia_autoridad <> 'autoridad_maestra_acreditada'
            OR registro.resuelto_en < v.vigente_desde OR registro.resuelto_en >= v.vigente_hasta
       ) THEN
        RAISE EXCEPTION 'vinculos historicos divergentes' USING ERRCODE='22023';
    END IF;
    -- Mismos formatos, orden y escapes del serializador propietario base V2.
    -- Comparar bytes cierra claves extra/ausentes, tipos, null, duplicados y orden.
    manifiesto_procedencia := format(
      '{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada","cuenta":{"cuenta_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"persona":{"persona_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"perfil":{"perfil_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"contexto":{"vinculo_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"vinculos":[%s]}',
      to_json(registro.cuenta_ref)::text,cuenta.version::text,to_json(cuenta.procedencia_ref)::text,
      cuenta.procedencia_version::text,to_json(cuenta.procedencia_huella_sha256)::text,to_json(cuenta.procedencia_autoridad)::text,
      to_json(perfil.persona_ref)::text,persona.version::text,to_json(persona.procedencia_ref)::text,
      persona.procedencia_version::text,to_json(persona.procedencia_huella_sha256)::text,to_json(persona.procedencia_autoridad)::text,
      to_json(registro.perfil_ref)::text,perfil.version::text,to_json(perfil.procedencia_ref)::text,
      perfil.procedencia_version::text,to_json(perfil.procedencia_huella_sha256)::text,to_json(perfil.procedencia_autoridad)::text,
      to_json(enlace.vinculo_ref)::text,enlace.version::text,to_json(enlace.procedencia_ref)::text,
      enlace.procedencia_version::text,to_json(enlace.procedencia_huella_sha256)::text,to_json(enlace.procedencia_autoridad)::text,
      coalesce(enlaces_procedencia_texto,''));
    manifiesto_procedencia_bytes := convert_to(manifiesto_procedencia,'UTF8');
    manifiesto_procedencia_huella := encode(pg_catalog.sha256(manifiesto_procedencia_bytes),'hex');

    documento := format(
      '{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":%s,"garantia":%s,"perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
      to_json(perfil.persona_ref)::text, to_json(registro.metodo)::text, to_json(registro.garantia)::text,
      to_json(registro.perfil_ref)::text, to_json(perfil.persona_ref)::text,
      to_json(enlace.vinculo_ref)::text, enlace.version::text, to_json(registro.cuenta_ref)::text, cuenta.version::text,
      persona.version::text, perfil.version::text, to_json(enlace.estado)::text,
      to_json(to_char(enlace.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(enlace.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(registro.resuelto_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      coalesce(enlaces_texto,''));
    canonica := convert_to(documento,'UTF8');

    IF canonica IS DISTINCT FROM registro.representacion_canonica
       OR manifiesto_procedencia_bytes IS DISTINCT FROM registro.manifiesto_procedencia_canonico
       OR encode(sha256(canonica),'hex') IS DISTINCT FROM p_contexto_sha256
       OR manifiesto_procedencia_huella IS DISTINCT FROM p_procedencia_sha256 THEN
        RAISE EXCEPTION 'canon historico divergente' USING ERRCODE='22023';
    END IF;
    IF clock_timestamp() > inicio + interval '5 seconds' THEN
        RAISE EXCEPTION 'lectura historica fuera de ventana' USING ERRCODE='57014';
    END IF;
    RETURN QUERY SELECT registro.registro_contexto_ref,registro.representacion_canonica,
        registro.huella_sha256,registro.manifiesto_procedencia_canonico,
        registro.manifiesto_procedencia_huella_sha256,registro.autoridad_efectiva,registro.resuelto_en;
EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION 'original historico no disponible' USING ERRCODE='P0002';
    WHEN data_exception THEN
        RAISE EXCEPTION 'original historico invalido' USING ERRCODE='22023';
END
$historia$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_v1_lector_historico;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)
 TO vec_contexto_actor_v1_lector_historico;
DO $acl$
DECLARE f oid:='vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)'::regprocedure;
 o oid:='vec_contexto_actor_v1_propietario'::regrole;
 r oid:='vec_contexto_actor_v1_lector_historico'::regrole;
BEGIN
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(p.proacl) a WHERE p.oid=f
   AND (a.grantee NOT IN(o,r) OR a.grantor<>o OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'ACL historica de creacion inesperada' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
