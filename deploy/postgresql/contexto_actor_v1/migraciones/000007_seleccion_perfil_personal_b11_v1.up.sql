-- B11: seleccion gobernada cuenta -> perfil; nunca recibe perfil del cliente.
-- 000001-000005 permanecen intactas. La guarda conserva identidad/OID y ACL.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:perfil-personal-b11:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR current_setting('server_version_num')::integer < 180000
    OR to_regclass('vec_contexto_actor_v1.control_generacion_punteros_actuales_v2') IS NULL
    OR to_regrole('vec_contexto_actor_perfil_personal_b11_runtime') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_contexto_actor_perfil_personal_b11_runtime'
        AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
        AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL)
    OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_contexto_actor_perfil_personal_b11_runtime'::regrole)
    OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid='vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'::regprocedure
        AND proowner='vec_contexto_actor_v1_propietario'::regrole AND prosecdef
        AND proconfig=ARRAY['search_path=pg_catalog']::text[]
        AND proacl=ARRAY['vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario']::aclitem[])
    OR (SELECT md5(prosrc) FROM pg_proc WHERE oid='vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'::regprocedure)
       IS DISTINCT FROM '937d9415ba81d66b3b3692da0529d19e'
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='precondicion B11 rechazada'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_exacto_b11(p_b11 boolean)
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
      INTO grupo FROM pg_catalog.pg_roles WHERE rolname = CASE WHEN p_b11 THEN 'vec_contexto_actor_perfil_personal_b11_runtime' ELSE 'vec_contexto_actor_v1_runtime' END;
    runtime_oid := grupo.oid;
    login_oid := login.oid;
    SELECT oid INTO esquema_oid FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1';
    SELECT oid INTO base_oid FROM pg_catalog.pg_database WHERE datname=current_database();
    IF p_b11 IS NULL THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='modo runtime invalido'; END IF;
    IF p_b11 THEN
        funciones := ARRAY[
          to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_actor_perfil_personal_b11_v1()'),
          to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz)'),
          to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz)')
        ];
    ELSE
    funciones := ARRAY[
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)'),
      pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz)')
    ];
    END IF;
    SELECT count(*) INTO membresias FROM pg_catalog.pg_auth_members
     WHERE member = login_oid;
    IF login_oid IS NULL OR runtime_oid IS NULL OR esquema_oid IS NULL OR base_oid IS NULL
       OR cardinality(funciones) <> 3 OR array_position(funciones,NULL) IS NOT NULL
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
           SELECT count(*)=5 AND bool_and(
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
           SELECT count(*)=3 AND count(DISTINCT p.oid)=3
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

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN vec_contexto_actor_v1.exigir_runtime_exacto_b11(
   pg_has_role(session_user,'vec_contexto_actor_perfil_personal_b11_runtime','MEMBER'));
END $f$;

CREATE TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones (
 seleccion_ref text NOT NULL, version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 18446744073709551615::numeric), cuenta_ref text NOT NULL, perfil_ref text NOT NULL,
 procedencia_ref text NOT NULL, procedencia_version numeric(20,0) NOT NULL, procedencia_huella_sha256 text NOT NULL, procedencia_autoridad text NOT NULL,
 estado text NOT NULL, vigente_desde timestamptz NOT NULL, vigente_hasta timestamptz NOT NULL,
 PRIMARY KEY(seleccion_ref,version), UNIQUE(cuenta_ref,seleccion_ref,version), UNIQUE(cuenta_ref,version),
 CHECK(vec_contexto_actor_v1.referencia_valida(seleccion_ref,'spp_')), CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_')), CHECK(vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_')),
 CHECK(vec_contexto_actor_v1.procedencia_valida(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)),
 CHECK(procedencia_autoridad='autoridad_maestra_acreditada'), CHECK(estado IN ('activo','revocado')), CHECK(vec_contexto_actor_v1.instante_valido(vigente_desde)), CHECK(vec_contexto_actor_v1.instante_valido(vigente_hasta)), CHECK(vigente_hasta>vigente_desde),
 FOREIGN KEY(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad) REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad));
CREATE TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual(
 cuenta_ref text PRIMARY KEY, seleccion_ref text NOT NULL, version numeric(20,0) NOT NULL,
 FOREIGN KEY(cuenta_ref,seleccion_ref,version) REFERENCES vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones(cuenta_ref,seleccion_ref,version));
CREATE TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos(
 operacion_ref text PRIMARY KEY, seleccion_ref text NOT NULL, version numeric(20,0) NOT NULL, registro_contexto_ref text NOT NULL UNIQUE,
 FOREIGN KEY(seleccion_ref,version) REFERENCES vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones(seleccion_ref,version),
 FOREIGN KEY(operacion_ref) REFERENCES vec_contexto_actor_v1.registros_contexto(operacion_ref),
 FOREIGN KEY(registro_contexto_ref) REFERENCES vec_contexto_actor_v1.registros_contexto(registro_contexto_ref));
ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones ENABLE ROW LEVEL SECURITY; ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual ENABLE ROW LEVEL SECURITY; ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos ENABLE ROW LEVEL SECURITY; ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_b11 ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user='vec_contexto_actor_v1_propietario') WITH CHECK(current_user='vec_contexto_actor_v1_propietario');
CREATE POLICY acceso_propietario_b11 ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user='vec_contexto_actor_v1_propietario') WITH CHECK(current_user='vec_contexto_actor_v1_propietario');
CREATE POLICY acceso_propietario_b11 ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user='vec_contexto_actor_v1_propietario') WITH CHECK(current_user='vec_contexto_actor_v1_propietario');
CREATE TRIGGER historia_b11_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_b11_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER usos_b11_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER puntero_b11_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER serializar_b11 BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
CREATE TRIGGER generacion_b11 AFTER INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();

CREATE TRIGGER usos_b11_inmutables BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();

-- Invariantes relacionales adicionales: un uso nunca enlaza un recibo ajeno,
-- y el puntero no puede retroceder ni eliminarse para reciclar una seleccion.
CREATE FUNCTION vec_contexto_actor_v1.validar_enlace_seleccion_b11()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_TABLE_NAME='seleccion_perfil_personal_b11_actual' THEN
  IF TG_OP='DELETE' OR (TG_OP='UPDATE' AND (NEW.cuenta_ref<>OLD.cuenta_ref OR NEW.seleccion_ref<>OLD.seleccion_ref OR NEW.version<=OLD.version))
  THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='puntero B11 no monotono'; END IF;
 ELSE
  IF NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r
    JOIN vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones s ON s.cuenta_ref=r.cuenta_ref AND s.perfil_ref=r.perfil_ref
    WHERE r.operacion_ref=NEW.operacion_ref AND r.registro_contexto_ref=NEW.registro_contexto_ref
      AND s.seleccion_ref=NEW.seleccion_ref AND s.version=NEW.version)
  THEN RAISE EXCEPTION USING ERRCODE='23514',MESSAGE='uso B11 no corresponde al contexto'; END IF;
 END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER puntero_b11_monotono BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.validar_enlace_seleccion_b11();
CREATE TRIGGER uso_b11_coherente BEFORE INSERT ON vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.validar_enlace_seleccion_b11();
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.validar_enlace_seleccion_b11() FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;

-- La seleccion se provisiona exclusivamente por la autoridad propietaria.
-- No amplia las ACL del selector corporativo RRHH ni del runtime historico.
CREATE FUNCTION vec_contexto_actor_v1.publicar_seleccion_perfil_personal_b11_v1(p_seleccion_ref text,p_version numeric,p_cuenta_ref text,p_perfil_ref text,p_procedencia_ref text,p_procedencia_version numeric,p_procedencia_huella text,p_estado text,p_desde timestamptz,p_hasta timestamptz)
RETURNS void LANGUAGE plpgsql SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE anterior record; perfil record; cuenta record; ahora timestamptz;
BEGIN
 IF current_user <> 'vec_contexto_actor_v1_propietario' THEN RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='publicador B11 no acreditado'; END IF;
 IF p_version IS NULL OR scale(p_version)<>0 OR p_version NOT BETWEEN 1 AND 18446744073709551615::numeric
 OR p_estado IS NULL OR p_estado NOT IN ('activo','revocado')
 OR vec_contexto_actor_v1.instante_valido(p_desde) IS NOT TRUE OR vec_contexto_actor_v1.instante_valido(p_hasta) IS NOT TRUE OR p_hasta<=p_desde
 THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='seleccion B11 invalida'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT * INTO anterior FROM vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual WHERE cuenta_ref=p_cuenta_ref FOR UPDATE;
 IF (FOUND AND (anterior.seleccion_ref<>p_seleccion_ref OR anterior.version+1<>p_version))
 OR (NOT FOUND AND p_version<>1)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones WHERE seleccion_ref=p_seleccion_ref AND cuenta_ref<>p_cuenta_ref)
 THEN RAISE EXCEPTION USING ERRCODE='23505',MESSAGE='version o titular de seleccion B11 incompatible'; END IF;
 SELECT cv.* INTO STRICT cuenta FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING(cuenta_ref,version) WHERE ca.cuenta_ref=p_cuenta_ref FOR SHARE OF ca;
 SELECT pv.* INTO STRICT perfil FROM vec_contexto_actor_v1.perfil_actual pa JOIN vec_contexto_actor_v1.perfil_versiones pv USING(perfil_ref,version) WHERE pa.perfil_ref=p_perfil_ref FOR SHARE OF pa;
 ahora:=clock_timestamp();
 IF p_estado='activo' AND (perfil.estado<>'activo' OR cuenta.estado<>'activo'
 OR perfil.procedencia_autoridad<>'autoridad_maestra_acreditada' OR cuenta.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR ahora<perfil.vigente_desde OR ahora>=perfil.vigente_hasta OR ahora<cuenta.vigente_desde OR ahora>=cuenta.vigente_hasta)
 THEN RAISE EXCEPTION USING ERRCODE='P0002',MESSAGE='cuenta o perfil B11 no acreditados'; END IF;
 INSERT INTO vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones VALUES(p_seleccion_ref,p_version,p_cuenta_ref,p_perfil_ref,p_procedencia_ref,p_procedencia_version,p_procedencia_huella,'autoridad_maestra_acreditada',p_estado,p_desde,p_hasta);
 INSERT INTO vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual VALUES(p_cuenta_ref,p_seleccion_ref,p_version) ON CONFLICT(cuenta_ref) DO UPDATE SET seleccion_ref=excluded.seleccion_ref,version=excluded.version;
END $f$;

CREATE FUNCTION vec_contexto_actor_v1.validar_entrada_b11(p_operacion_ref text,p_registro_ref text,p_cuenta_ref text,p_metodo text,p_garantia text,p_solicitado_en timestamptz)
RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref,'oca_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_ref,'rca_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
 OR p_metodo IS NULL OR p_metodo NOT IN ('certificado','dnie','sso','clave','kerberos_ad','demo')
 OR p_garantia IS NULL OR p_garantia NOT IN ('bajo','sustancial','alto')
 OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE
 THEN RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='entrada B11 invalida'; END IF;
END $f$;

-- Recuperacion: el advisory global ya esta adquirido por la fachada. Todas
-- las mutaciones de punteros usan su modalidad exclusiva. La fila generacion
-- aporta el conflicto MVCC en SERIALIZABLE; READ COMMITTED relee tras esperar.
-- La comparacion exige las mismas versiones del recibo, no solo mismo perfil.
CREATE FUNCTION vec_contexto_actor_v1.revalidar_registro_b11(p_registro_ref text)
RETURNS timestamptz LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE r record; d jsonb; cuenta record; perfil record; persona record; contexto record;
 ahora timestamptz; n integer; enlaces jsonb;
BEGIN
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto WHERE registro_contexto_ref=p_registro_ref FOR SHARE;
 d:=convert_from(r.representacion_canonica,'UTF8')::jsonb;
 SELECT v.* INTO STRICT cuenta FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version) WHERE a.cuenta_ref=r.cuenta_ref FOR SHARE OF a;
 SELECT v.* INTO STRICT perfil FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version) WHERE a.perfil_ref=r.perfil_ref FOR SHARE OF a;
 PERFORM 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version) WHERE v.cuenta_ref=r.cuenta_ref AND v.perfil_ref=r.perfil_ref ORDER BY a.vinculo_ref FOR SHARE OF a;
 GET DIAGNOSTICS n=ROW_COUNT;
 IF n<>1 THEN RAISE EXCEPTION USING ERRCODE='P0002',MESSAGE='contexto B11 ambiguo o ausente'; END IF;
 SELECT v.* INTO STRICT contexto FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version) WHERE v.cuenta_ref=r.cuenta_ref AND v.perfil_ref=r.perfil_ref;
 SELECT v.* INTO STRICT persona FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version) WHERE a.persona_ref=perfil.persona_ref FOR SHARE OF a;
 PERFORM 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version) WHERE v.persona_ref=perfil.persona_ref ORDER BY a.vinculo_ref FOR SHARE OF a;
 PERFORM generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 WHERE control_id FOR SHARE;
 ahora:=clock_timestamp();
 IF cuenta.version IS DISTINCT FROM (d->>'cuenta_version')::numeric
 OR perfil.version IS DISTINCT FROM (d->>'perfil_version')::numeric
 OR persona.version IS DISTINCT FROM (d->>'persona_version')::numeric
 OR contexto.version IS DISTINCT FROM (d->>'contexto_version')::numeric
 OR contexto.vinculo_ref IS DISTINCT FROM d->>'contexto_actor_ref'
 OR perfil.persona_ref IS DISTINCT FROM d->>'persona_ref' OR contexto.persona_ref<>perfil.persona_ref
 OR EXISTS(SELECT 1 FROM (VALUES(cuenta.estado,cuenta.procedencia_autoridad,cuenta.vigente_desde,cuenta.vigente_hasta),
 (perfil.estado,perfil.procedencia_autoridad,perfil.vigente_desde,perfil.vigente_hasta),
 (persona.estado,persona.procedencia_autoridad,persona.vigente_desde,persona.vigente_hasta),
 (contexto.estado,contexto.procedencia_autoridad,contexto.vigente_desde,contexto.vigente_hasta)) AS v(estado,autoridad,desde,hasta)
 WHERE estado<>'activo' OR autoridad<>'autoridad_maestra_acreditada' OR ahora<desde OR ahora>=hasta)
 THEN RAISE EXCEPTION USING ERRCODE='P0002',MESSAGE='contexto B11 revocado o cambiado'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('vinculo_ref',v.vinculo_ref,'version',v.version,'tipo',v.tipo,'referencia',v.referencia)
 ORDER BY v.tipo,v.referencia,v.version,v.vinculo_ref),'[]'::jsonb) INTO enlaces
 FROM vec_contexto_actor_v1.vinculo_referencia_actual a JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version) WHERE v.persona_ref=perfil.persona_ref;
 IF enlaces IS DISTINCT FROM (SELECT coalesce(jsonb_agg(jsonb_build_object('vinculo_ref',x->>'vinculo_ref','version',(x->>'version')::numeric,'tipo',x->>'tipo','referencia',x->>'referencia') ORDER BY x->>'tipo',x->>'referencia',(x->>'version')::numeric,x->>'vinculo_ref'),'[]'::jsonb) FROM jsonb_array_elements(d->'vinculos') x)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version) WHERE v.persona_ref=perfil.persona_ref AND (v.estado<>'activo' OR v.procedencia_autoridad<>'autoridad_maestra_acreditada' OR ahora<v.vigente_desde OR ahora>=v.vigente_hasta))
 THEN RAISE EXCEPTION USING ERRCODE='P0002',MESSAGE='referencias B11 revocadas o cambiadas'; END IF;
 RETURN ahora;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_perfil_personal_b11_v1(p_operacion_ref text,p_registro_ref text,p_cuenta_ref text,p_metodo text,p_garantia text,p_solicitado_en timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s record; r record; ahora timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_exacto_b11(true);
 PERFORM vec_contexto_actor_v1.validar_entrada_b11(p_operacion_ref,p_registro_ref,p_cuenta_ref,p_metodo,p_garantia,p_solicitado_en);
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION USING ERRCODE='25000',MESSAGE='aislamiento B11 incorrecto'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:operacion:v2:'||p_operacion_ref,0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 PERFORM generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 WHERE control_id FOR SHARE;
 SELECT v.* INTO STRICT s FROM vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual a JOIN vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones v USING(cuenta_ref,seleccion_ref,version) WHERE a.cuenta_ref=p_cuenta_ref FOR SHARE OF a;

 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto x WHERE x.operacion_ref=p_operacion_ref) THEN
  SELECT x.* INTO r FROM vec_contexto_actor_v1.registros_contexto x
  JOIN vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos u ON u.operacion_ref=x.operacion_ref AND u.registro_contexto_ref=x.registro_contexto_ref
  WHERE x.operacion_ref=p_operacion_ref AND x.cuenta_ref=p_cuenta_ref AND x.perfil_ref=s.perfil_ref
    AND x.metodo=p_metodo AND x.garantia=p_garantia AND x.solicitado_en=p_solicitado_en
    AND u.seleccion_ref=s.seleccion_ref AND u.version=s.version;
  IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE='23505',MESSAGE='colision de operacion B11'; END IF;
  ahora:=vec_contexto_actor_v1.revalidar_registro_b11(r.registro_contexto_ref);
 ELSE
  SELECT * INTO STRICT r FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(p_operacion_ref,p_registro_ref,p_cuenta_ref,s.perfil_ref,p_metodo,p_garantia,p_solicitado_en);
  ahora:=r.resuelto_en; -- unico reloj de negocio, tomado por V2 tras todos sus locks
  INSERT INTO vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos VALUES(p_operacion_ref,s.seleccion_ref,s.version,r.registro_contexto_ref);
 END IF;
 IF s.estado<>'activo' OR ahora<s.vigente_desde OR ahora>=s.vigente_hasta THEN RAISE EXCEPTION USING ERRCODE='P0002',MESSAGE='seleccion B11 no vigente'; END IF;
 PERFORM vec_contexto_actor_v1.exigir_runtime_exacto_b11(true);
 RETURN QUERY SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,r.huella_sha256,r.manifiesto_procedencia_canonico,r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_perfil_personal_b11_v1(p_operacion_ref text,p_registro_ref text,p_cuenta_ref text,p_metodo text,p_garantia text,p_solicitado_en timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE s record; r record; ahora timestamptz;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_exacto_b11(true);
 PERFORM vec_contexto_actor_v1.validar_entrada_b11(p_operacion_ref,p_registro_ref,p_cuenta_ref,p_metodo,p_garantia,p_solicitado_en);
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION USING ERRCODE='25000',MESSAGE='aislamiento B11 incorrecto'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:operacion:v2:'||p_operacion_ref,0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 PERFORM generacion FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2 WHERE control_id FOR SHARE;
 SELECT v.* INTO STRICT s FROM vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual a JOIN vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones v USING(cuenta_ref,seleccion_ref,version) WHERE a.cuenta_ref=p_cuenta_ref FOR SHARE OF a;

 SELECT x.* INTO r FROM vec_contexto_actor_v1.registros_contexto x
 JOIN vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos u ON u.operacion_ref=x.operacion_ref AND u.registro_contexto_ref=x.registro_contexto_ref
 WHERE x.operacion_ref=p_operacion_ref AND x.registro_contexto_ref=p_registro_ref AND x.cuenta_ref=p_cuenta_ref AND x.perfil_ref=s.perfil_ref
 AND x.metodo=p_metodo AND x.garantia=p_garantia AND x.solicitado_en=p_solicitado_en
 AND u.seleccion_ref=s.seleccion_ref AND u.version=s.version;
 IF NOT FOUND THEN RETURN; END IF;
 ahora:=vec_contexto_actor_v1.revalidar_registro_b11(r.registro_contexto_ref);
 IF s.estado<>'activo' OR ahora<s.vigente_desde OR ahora>=s.vigente_hasta THEN RAISE EXCEPTION USING ERRCODE='P0002',MESSAGE='seleccion B11 no vigente'; END IF;
 PERFORM vec_contexto_actor_v1.exigir_runtime_exacto_b11(true);
 RETURN QUERY SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,r.huella_sha256,r.manifiesto_procedencia_canonico,r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en;
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_perfil_personal_b11_v1(OUT identidad_login text,OUT acreditada boolean) RETURNS record LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN identidad_login:=vec_contexto_actor_v1.exigir_runtime_exacto_b11(true); acreditada:=true; END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_exacto_b11(boolean) FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.publicar_seleccion_perfil_personal_b11_v1(text,numeric,text,text,text,numeric,text,text,timestamptz,timestamptz) FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.validar_entrada_b11(text,text,text,text,text,timestamptz) FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.revalidar_registro_b11(text) FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz) FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz) FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_perfil_personal_b11_v1() FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON TYPE vec_contexto_actor_v1.seleccion_perfil_personal_b11_versiones FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON TYPE vec_contexto_actor_v1.seleccion_perfil_personal_b11_actual FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON TABLE vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
REVOKE ALL ON TYPE vec_contexto_actor_v1.seleccion_perfil_personal_b11_usos FROM PUBLIC,vec_contexto_actor_perfil_personal_b11_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_perfil_personal_b11_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz) TO vec_contexto_actor_perfil_personal_b11_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_perfil_personal_b11_v1(text,text,text,text,text,timestamptz) TO vec_contexto_actor_perfil_personal_b11_runtime;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_perfil_personal_b11_v1() TO vec_contexto_actor_perfil_personal_b11_runtime;
COMMIT;
