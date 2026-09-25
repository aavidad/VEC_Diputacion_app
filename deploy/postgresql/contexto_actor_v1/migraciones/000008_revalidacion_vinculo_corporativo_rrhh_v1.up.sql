-- ContextoActor 000008: revalidación por petición del vínculo corporativo RRHH.
--
-- El vínculo corporativo (000004/000004a) solo se cotejaba al aprovisionar
-- (alias HMAC de Identidad y publicación del permiso interno). Revocarlo no
-- cortaba el acceso de vec-interno: la resolución F1 de cada petición no lo
-- consulta. Esta migración expone al runtime de ContextoActor una única
-- lectura cerrada que responde sí/no, sin devolver datos, a la pregunta:
--
--   ¿la cuenta, el perfil, la persona y el vínculo de contexto F1 exactos de
--   esta petición tienen ahora un vínculo corporativo actual, activo y vigente
--   para interna_corporativa/consulta_rrhh, ligado a las versiones actuales de
--   cuenta, perfil, persona y vínculo de contexto, y con una organización
--   actual, activa y vigente de procedencia autoritativa?
--
-- * Mismo cotejo que vec-publicar-permiso-interno y aprovisionar.py, sin
--   parámetro de organización: la organización es la que fija el vínculo.
-- * Instante autoritativo del servidor (clock_timestamp); el llamante no lo
--   aporta. Revocación, caducidad, ausencia o una versión nueva de cualquier
--   eslabón que el vínculo no referencie devuelven false.
-- * Sin caché ni registro: cada llamada lee los punteros actuales. Solo
--   lectura; no toma bloqueos de publicación.
-- * El runtime pasa de cinco a seis funciones: exigir_runtime_contexto_actor_v1
--   se sustituye por la misma comprobación con la función nueva en su
--   manifiesto cerrado (misma estructura, contadores 6/8/6).
--
-- Orden: tras 000007. DOWN permitido: la función no firma ni registra nada;
-- restaura exactamente la comprobación de runtime de 000007.
BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:migracion:revalidacion_vinculo_corporativo:v1',0));
DO $preimagen$
DECLARE
  e oid := pg_catalog.to_regprocedure('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()');
  propietario oid := pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
  runtime oid := pg_catalog.to_regrole('vec_contexto_actor_v1_runtime');
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
     OR current_setting('transaction_isolation') <> 'read committed' THEN
    RAISE EXCEPTION 'migracion ContextoActor 000008 requiere superusuario y transaccion ordinaria' USING ERRCODE='42501';
  END IF;
  IF e IS NULL OR propietario IS NULL OR runtime IS NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') IS NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') IS NULL
     OR pg_catalog.to_regclass('vec_contexto_actor_v1.vinculo_corporativo_versiones') IS NULL
     OR pg_catalog.to_regclass('vec_contexto_actor_v1.vinculo_corporativo_actual') IS NULL
     OR pg_catalog.to_regclass('vec_contexto_actor_v1.organizacion_versiones') IS NULL
     OR pg_catalog.to_regclass('vec_contexto_actor_v1.organizacion_actual') IS NULL
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                 WHERE p.pronamespace='vec_contexto_actor_v1'::regnamespace
                   AND p.proname='revalidar_vinculo_corporativo_rrhh_v1') THEN
    RAISE EXCEPTION 'ContextoActor 000008 exige 000007 instalada y no aplicada antes' USING ERRCODE='55000';
  END IF;
  -- Postimagen exacta de 000007 para la comprobación de runtime.
  IF (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
        FROM pg_catalog.pg_proc WHERE oid=e) <> 'f29b3374ea974e13e49454456e7fe80d6e1e7030cab0f566eb6c41eea0f3ab88'
     OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=e) <> propietario
     OR (SELECT count(*) FROM pg_catalog.pg_proc p
          WHERE p.pronamespace='vec_contexto_actor_v1'::regnamespace
            AND pg_catalog.has_function_privilege(runtime,p.oid,'EXECUTE')) <> 5 THEN
    RAISE EXCEPTION 'preimagen de runtime ContextoActor 000007 divergente' USING ERRCODE='55000';
  END IF;
END
$preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

-- Lectura cerrada sí/no. Una entrada inválida deniega igual que la ausencia.
CREATE FUNCTION vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(
    p_cuenta_ref text, p_perfil_ref text, p_persona_ref text,
    p_vinculo_contexto_ref text, p_vinculo_contexto_version numeric
) RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    ahora timestamptz; coincidencias integer;
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    IF vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_persona_ref, 'per_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_vinculo_contexto_ref, 'vca_') IS NOT TRUE
       OR p_vinculo_contexto_version IS NULL
       OR p_vinculo_contexto_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_vinculo_contexto_version <> pg_catalog.trunc(p_vinculo_contexto_version) THEN
        RETURN false;
    END IF;
    ahora := pg_catalog.clock_timestamp();
    SELECT count(*) INTO coincidencias
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
        ON (v.vinculo_ref, v.version) = (va.vinculo_ref, va.version)
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref = v.cuenta_ref
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c
        ON (c.cuenta_ref, c.version) = (ca.cuenta_ref, ca.version)
      JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref = v.perfil_ref
      JOIN vec_contexto_actor_v1.perfil_versiones p
        ON (p.perfil_ref, p.version) = (pa.perfil_ref, pa.version)
      JOIN vec_contexto_actor_v1.persona_actual xa ON xa.persona_ref = v.persona_ref
      JOIN vec_contexto_actor_v1.persona_versiones x
        ON (x.persona_ref, x.version) = (xa.persona_ref, xa.version)
      JOIN vec_contexto_actor_v1.vinculo_corporativo_actual vc
        ON vc.cuenta_ref = v.cuenta_ref AND vc.superficie = 'interna_corporativa'
       AND vc.uso = 'consulta_rrhh'
      JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones cv
        ON (cv.vinculo_corporativo_ref, cv.version) = (vc.vinculo_corporativo_ref, vc.version)
      JOIN vec_contexto_actor_v1.organizacion_actual oa ON oa.organizacion_ref = cv.organizacion_ref
      JOIN vec_contexto_actor_v1.organizacion_versiones ov
        ON (ov.organizacion_ref, ov.version) = (oa.organizacion_ref, oa.version)
     WHERE va.vinculo_ref = p_vinculo_contexto_ref AND va.version = p_vinculo_contexto_version
       AND v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref
       AND v.persona_ref = p_persona_ref AND p.persona_ref = p_persona_ref
       AND cv.cuenta_ref = v.cuenta_ref AND cv.perfil_ref = v.perfil_ref
       AND cv.persona_ref = v.persona_ref
       AND cv.vinculo_contexto_ref = v.vinculo_ref AND cv.vinculo_contexto_version = v.version
       AND cv.cuenta_version = ca.version AND cv.perfil_version = pa.version
       AND cv.persona_version = xa.version
       AND cv.organizacion_version = oa.version
       AND cv.organizacion_procedencia_ref = ov.procedencia_ref
       AND cv.organizacion_procedencia_version = ov.procedencia_version
       AND cv.organizacion_procedencia_huella_sha256 = ov.procedencia_huella_sha256
       AND cv.organizacion_procedencia_autoridad = ov.procedencia_autoridad
       AND cv.procedencia_autoridad = 'autoridad_maestra_acreditada'
       AND ov.procedencia_autoridad = 'autoridad_maestra_acreditada'
       AND cv.superficie = 'interna_corporativa' AND cv.uso = 'consulta_rrhh'
       AND cv.estado = 'activo' AND ov.estado = 'activo'
       AND v.estado = 'activo' AND c.estado = 'activo'
       AND p.estado = 'activo' AND x.estado = 'activo'
       AND ahora >= v.vigente_desde AND ahora < v.vigente_hasta
       AND ahora >= c.vigente_desde AND ahora < c.vigente_hasta
       AND ahora >= p.vigente_desde AND ahora < p.vigente_hasta
       AND ahora >= x.vigente_desde AND ahora < x.vigente_hasta
       AND ahora >= cv.vigente_desde AND ahora < cv.vigente_hasta
       AND ahora >= ov.vigente_desde AND ahora < ov.vigente_hasta;
    RETURN coincidencias = 1;
END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)
    TO vec_contexto_actor_v1_runtime;

-- Estructura del runtime: seis funciones (la revalidación corporativa nueva).
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;
BEGIN
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO login FROM pg_catalog.pg_roles WHERE rolname = session_user;
    SELECT oid, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin,
           rolreplication, rolbypassrls, rolconfig
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = 'vec_contexto_actor_v1_runtime';
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    funciones := ARRAY[
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)')
    ];
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 6 OR array_position(funciones,NULL) IS NOT NULL
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
           SELECT count(*)=8 AND bool_and(
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
           SELECT count(*)=6 AND count(DISTINCT p.oid)=6
                  AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))
            ) a
            WHERE p.oid=ANY(funciones) AND a.grantee=runtime_oid
       ),false)
       OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(
            login_oid,base_oid,esquema_oid,funciones) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'LOGIN runtime de contexto actor V1 no acreditado';
    END IF;
    RETURN session_user;
END
$f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1() FROM PUBLIC;

-- Postimagen: runtime con exactamente seis funciones; la nueva, propia del
-- propietario, SECURITY DEFINER y sin EXECUTE de PUBLIC.
DO $postimagen$
DECLARE runtime oid := 'vec_contexto_actor_v1_runtime'::regrole;
        propietario oid := 'vec_contexto_actor_v1_propietario'::regrole;
        f oid := 'vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)'::regprocedure;
BEGIN
    IF (SELECT count(*) FROM pg_catalog.pg_proc p
          WHERE p.pronamespace = 'vec_contexto_actor_v1'::regnamespace
            AND pg_catalog.has_function_privilege(runtime, p.oid, 'EXECUTE')) <> 6
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid = f
                        AND p.proowner = propietario AND p.prosecdef
                        AND p.proconfig = ARRAY['search_path=pg_catalog']::text[])
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p, pg_catalog.aclexplode(p.proacl) acl
                   WHERE p.oid = f AND (acl.grantee = 0 OR acl.is_grantable
                     OR acl.grantee NOT IN (runtime, propietario)))
       OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
             FROM pg_catalog.pg_proc
            WHERE oid = 'vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'::regprocedure)
          <> '005fff9328377a75bcde2a23e997959f2d1603f658ae39a2e6f2eb8d8986d3c8' THEN
        RAISE EXCEPTION 'postimagen ContextoActor 000008 divergente' USING ERRCODE='55000';
    END IF;
END
$postimagen$;
COMMIT;
