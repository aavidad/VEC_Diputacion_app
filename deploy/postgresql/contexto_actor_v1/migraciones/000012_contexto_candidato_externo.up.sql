\set ON_ERROR_STOP on
-- ContextoActor 000012: proyección y recibo nominales del candidato externo.
-- Solo referencias opacas. La misma persona puede tener vínculos de empleado,
-- que nunca aparecen en el canon firmado del proceso externo.
-- DOWN prohibido una vez exista provisión o recibo.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:candidato_externo:000012',0));
DO $preimagen$
DECLARE r oid; c oid; a oid; e oid; p oid:='vec_contexto_actor_v1_propietario'::regrole;
BEGIN
 r:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])');
 c:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])');
 a:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)');
 e:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()');
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_contexto_actor_v1_candidato_externo') IS NULL
    OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)') IS NULL
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.candidato_externo_versiones') IS NOT NULL THEN
  RAISE EXCEPTION 'ContextoActor 000012: preimagen incompatible' USING ERRCODE='55000';
 END IF;
 IF (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=r)
       IS DISTINCT FROM '4546a0dbd5b67dfa56b6508293cf6a5e5035782a25e84b5665d2220c51ddbb6e'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=c)
       IS DISTINCT FROM 'd02d25fe02c6807901198251dc8b93a802a453a981ee671cbc3d89137f214f7a'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=a)
       IS DISTINCT FROM '538f7eb0a0ed6bcee9b3f9428e6cd776015719349c02bd361650cf415cce360e'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') FROM pg_catalog.pg_proc WHERE oid=e)
       IS DISTINCT FROM '005fff9328377a75bcde2a23e997959f2d1603f658ae39a2e6f2eb8d8986d3c8'
    OR (SELECT count(*) FROM pg_catalog.pg_proc WHERE oid IN(r,c,a,e) AND proowner=p AND prosecdef
          AND proconfig=ARRAY['search_path=pg_catalog']::text[])<>4 THEN
  RAISE EXCEPTION 'ContextoActor 000012: definición viva divergente' USING ERRCODE='55000';
 END IF;
END $preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
CREATE TABLE vec_contexto_actor_v1.candidato_externo_versiones(
 provision_ref text NOT NULL CHECK (vec_contexto_actor_v1.referencia_valida(provision_ref,'pce_')),
 version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 18446744073709551615::numeric),
 cuenta_ref text NOT NULL CHECK (vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_')),
 cuenta_version numeric(20,0) NOT NULL CHECK(cuenta_version>=1),
 perfil_ref text NOT NULL CHECK (vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_')),
 perfil_version numeric(20,0) NOT NULL CHECK(perfil_version>=1),
 persona_ref text NOT NULL CHECK (vec_contexto_actor_v1.referencia_valida(persona_ref,'per_')),
 persona_version numeric(20,0) NOT NULL CHECK(persona_version>=1),
 contexto_ref text NOT NULL CHECK (vec_contexto_actor_v1.referencia_valida(contexto_ref,'vca_')),
 contexto_version numeric(20,0) NOT NULL CHECK(contexto_version>=1),
 vinculo_candidato_ref text NOT NULL CHECK (vec_contexto_actor_v1.referencia_valida(vinculo_candidato_ref,'vin_')),
 vinculo_candidato_version numeric(20,0) NOT NULL CHECK(vinculo_candidato_version>=1),
 candidato_ref text NOT NULL CHECK (vec_contexto_actor_v1.referencia_valida(candidato_ref,'can_')),
 estado text NOT NULL CHECK(estado IN ('activo','revocado')),
 vigente_desde timestamptz NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_desde)),
 vigente_hasta timestamptz NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_hasta)),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(provision_ref,version), CHECK(vigente_hasta>vigente_desde)
);
CREATE TABLE vec_contexto_actor_v1.candidato_externo_actual(
 provision_ref text PRIMARY KEY, version numeric(20,0) NOT NULL,
 FOREIGN KEY(provision_ref,version) REFERENCES vec_contexto_actor_v1.candidato_externo_versiones(provision_ref,version)
);
CREATE TABLE vec_contexto_actor_v1.registro_candidato_externo_v1(
 registro_contexto_ref text PRIMARY KEY REFERENCES vec_contexto_actor_v1.registros_contexto(registro_contexto_ref),
 operacion_ref text NOT NULL UNIQUE,
 provision_ref text NOT NULL,
 provision_version numeric(20,0) NOT NULL,
 provision_huella_sha256 text NOT NULL,
 candidato_ref text NOT NULL,
 FOREIGN KEY(provision_ref,provision_version) REFERENCES vec_contexto_actor_v1.candidato_externo_versiones(provision_ref,version)
);
CREATE INDEX candidato_externo_cuenta_perfil_idx ON vec_contexto_actor_v1.candidato_externo_versiones(cuenta_ref,perfil_ref,provision_ref,version);
CREATE INDEX candidato_externo_persona_perfil_idx ON vec_contexto_actor_v1.candidato_externo_versiones(persona_ref,perfil_ref,provision_ref,version);
DO $rls$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['candidato_externo_versiones','candidato_externo_actual','registro_candidato_externo_v1'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('CREATE POLICY acceso_propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING (current_user=''vec_contexto_actor_v1_propietario'') WITH CHECK (current_user=''vec_contexto_actor_v1_propietario'')',tabla);
  EXECUTE pg_catalog.format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',tabla);
 END LOOP;
END $rls$;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.candidato_externo_versiones FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.registro_candidato_externo_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER serializar_mutacion_punteros_actuales_v2 BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.candidato_externo_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
CREATE TRIGGER avanzar_generacion_punteros_actuales_v2 AFTER INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.candidato_externo_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();
REVOKE ALL ON TABLE vec_contexto_actor_v1.candidato_externo_versiones,vec_contexto_actor_v1.candidato_externo_actual,vec_contexto_actor_v1.registro_candidato_externo_v1 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo;
REVOKE ALL ON TYPE vec_contexto_actor_v1.candidato_externo_versiones,vec_contexto_actor_v1.candidato_externo_actual,vec_contexto_actor_v1.registro_candidato_externo_v1 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo;
-- Una sola provisión activa por cuenta/perfil; las versiones anteriores no se
-- reactivan. Este lector no devuelve la cuenta ni permite elegir candidato.
CREATE FUNCTION vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(
 p_cuenta_ref text,p_perfil_ref text,p_ahora timestamptz
) RETURNS TABLE(provision_ref text,version numeric,huella_sha256 text,candidato_ref text)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE n integer; p record; candidatos integer;
BEGIN
 IF vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
    OR vec_contexto_actor_v1.instante_valido(p_ahora) IS NOT TRUE THEN RETURN; END IF;
 -- El advisory ordena frente al mutador; esta fila MVCC obliga a SERIALIZABLE
 -- a detectar un snapshot anterior a una revocación ya confirmada.
 PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  WHERE control_id=true FOR SHARE;
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.candidato_externo_actual a
 JOIN vec_contexto_actor_v1.candidato_externo_versiones v USING(provision_ref,version)
 WHERE v.cuenta_ref=p_cuenta_ref AND v.perfil_ref=p_perfil_ref AND v.estado='activo'
   AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta;
 IF n<>1 THEN RETURN; END IF;
 SELECT v.* INTO p FROM vec_contexto_actor_v1.candidato_externo_actual a
 JOIN vec_contexto_actor_v1.candidato_externo_versiones v USING(provision_ref,version)
 WHERE v.cuenta_ref=p_cuenta_ref AND v.perfil_ref=p_perfil_ref AND v.estado='activo'
   AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta;
 IF NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version)
    WHERE a.cuenta_ref=p.cuenta_ref AND a.version=p.cuenta_version AND v.estado='activo' AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta)
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
    WHERE a.perfil_ref=p.perfil_ref AND a.version=p.perfil_version AND v.persona_ref=p.persona_ref AND v.estado='activo' AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta)
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_actual a JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version)
    WHERE a.persona_ref=p.persona_ref AND a.version=p.persona_version AND v.estado='activo' AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta)
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
    WHERE a.vinculo_ref=p.contexto_ref AND a.version=p.contexto_version AND v.cuenta_ref=p.cuenta_ref AND v.perfil_ref=p.perfil_ref AND v.persona_ref=p.persona_ref AND v.estado='activo' AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta)
 OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
    WHERE a.vinculo_ref=p.vinculo_candidato_ref AND a.version=p.vinculo_candidato_version AND v.persona_ref=p.persona_ref AND v.tipo='candidato' AND v.referencia=p.candidato_ref AND v.estado='activo' AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta) THEN RETURN; END IF;
 SELECT count(*) INTO candidatos FROM vec_contexto_actor_v1.vinculo_referencia_actual a
 JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
 WHERE v.persona_ref=p.persona_ref AND v.tipo='candidato' AND v.estado='activo'
   AND p_ahora>=v.vigente_desde AND p_ahora<v.vigente_hasta;
 IF candidatos<>1 THEN RETURN; END IF;
 RETURN QUERY SELECT p.provision_ref,p.version,p.huella_sha256,p.candidato_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(text,text,timestamptz) FROM PUBLIC;

-- Publicación gobernada por huella de fuente y CAS. Solo el propietario puede
-- invocarla; ninguna petición HTTP obtiene permiso de provisión.
CREATE FUNCTION vec_contexto_actor_v1.publicar_provision_candidato_externo_v1(
 p_provision_ref text,p_version_esperada numeric,p_huella_esperada text,p_huella_aprobada text,
 p_cuenta_ref text,p_cuenta_version numeric,p_perfil_ref text,p_perfil_version numeric,
 p_persona_ref text,p_persona_version numeric,p_contexto_ref text,p_contexto_version numeric,
 p_vinculo_candidato_ref text,p_vinculo_candidato_version numeric,p_candidato_ref text,
 p_estado text,p_desde timestamptz,p_hasta timestamptz,p_fuente_huella_sha256 text
) RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE anterior record; nueva_version numeric; nueva_huella text; vigente record;
BEGIN
 IF session_user <> 'vec_contexto_actor_v1_propietario' AND current_user <> 'vec_contexto_actor_v1_propietario' THEN
  RAISE EXCEPTION 'provisión candidata denegada' USING ERRCODE='42501';
 END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR vec_contexto_actor_v1.referencia_valida(p_provision_ref,'pce_') IS NOT TRUE
    OR p_version_esperada IS NULL OR p_version_esperada<0 OR p_version_esperada<>trunc(p_version_esperada)
    OR p_estado NOT IN ('activo','revocado') OR p_desde IS NULL OR p_hasta<=p_desde
    OR p_fuente_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR p_huella_aprobada !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'provisión candidata inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT a.version,v.huella_sha256 INTO anterior FROM vec_contexto_actor_v1.candidato_externo_actual a
 JOIN vec_contexto_actor_v1.candidato_externo_versiones v USING(provision_ref,version)
 WHERE a.provision_ref=p_provision_ref FOR UPDATE OF a;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version<>p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada)) THEN
  RAISE EXCEPTION 'provisión candidata: CAS divergente' USING ERRCODE='40001';
 END IF;
 nueva_version:=p_version_esperada+1;
 nueva_huella:=encode(pg_catalog.sha256(convert_to(pg_catalog.jsonb_build_array(
  p_provision_ref,nueva_version,p_cuenta_ref,p_cuenta_version,p_perfil_ref,p_perfil_version,
  p_persona_ref,p_persona_version,p_contexto_ref,p_contexto_version,p_vinculo_candidato_ref,
  p_vinculo_candidato_version,p_candidato_ref,p_estado,p_desde,p_hasta,p_fuente_huella_sha256)::text,'UTF8')),'hex');
 IF nueva_huella IS DISTINCT FROM p_huella_aprobada THEN
  RAISE EXCEPTION 'provisión candidata: huella aprobada divergente' USING ERRCODE='42501';
 END IF;
 INSERT INTO vec_contexto_actor_v1.candidato_externo_versiones VALUES(
  p_provision_ref,nueva_version,p_cuenta_ref,p_cuenta_version,p_perfil_ref,p_perfil_version,
  p_persona_ref,p_persona_version,p_contexto_ref,p_contexto_version,p_vinculo_candidato_ref,
  p_vinculo_candidato_version,p_candidato_ref,p_estado,p_desde,p_hasta,nueva_huella,p_fuente_huella_sha256);
 INSERT INTO vec_contexto_actor_v1.candidato_externo_actual VALUES(p_provision_ref,nueva_version)
 ON CONFLICT(provision_ref) DO UPDATE SET version=excluded.version;
 IF p_estado='activo' THEN
  SELECT * INTO vigente FROM vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(p_cuenta_ref,p_perfil_ref,pg_catalog.clock_timestamp());
  IF vigente.provision_ref IS DISTINCT FROM p_provision_ref OR vigente.version IS DISTINCT FROM nueva_version THEN
   RAISE EXCEPTION 'provisión candidata sin vínculo único vigente' USING ERRCODE='55000';
  END IF;
 END IF;
 RETURN nueva_huella;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.publicar_provision_candidato_externo_v1(text,numeric,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz,text) FROM PUBLIC;

-- AUT solo consulta si el par persona/perfil tiene una provisión externa
-- inequívoca; nunca devuelve una cuenta de otro perfil.
CREATE FUNCTION vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(p_persona_ref text,p_perfil_ref text)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE p record; n integer;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.candidato_externo_actual a
 JOIN vec_contexto_actor_v1.candidato_externo_versiones v USING(provision_ref,version)
 WHERE v.persona_ref=p_persona_ref AND v.perfil_ref=p_perfil_ref AND v.estado='activo'
   AND pg_catalog.clock_timestamp()>=v.vigente_desde AND pg_catalog.clock_timestamp()<v.vigente_hasta;
 IF n<>1 THEN RETURN false; END IF;
 SELECT v.* INTO p FROM vec_contexto_actor_v1.candidato_externo_actual a
 JOIN vec_contexto_actor_v1.candidato_externo_versiones v USING(provision_ref,version)
 WHERE v.persona_ref=p_persona_ref AND v.perfil_ref=p_perfil_ref AND v.estado='activo'
   AND pg_catalog.clock_timestamp()>=v.vigente_desde AND pg_catalog.clock_timestamp()<v.vigente_hasta;
 RETURN EXISTS(SELECT 1 FROM vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(p.cuenta_ref,p_perfil_ref,pg_catalog.clock_timestamp()));
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.perfil_candidato_externo_provisionado_v1(text,text) TO vec_autorizacion_propietario;
-- El LOGIN externo hereda un solo grupo. La función falla si hay membresías,
-- ACL o parámetros de rol adicionales. No concede la resolución general.
CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE l record; g record; n integer; esquema oid; base oid; funciones oid[];
BEGIN
 SELECT oid,rolcanlogin,rolsuper,rolinherit,rolcreaterole,rolcreatedb,rolreplication,rolbypassrls,rolconfig
 INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT oid,rolcanlogin,rolsuper,rolinherit,rolcreaterole,rolcreatedb,rolreplication,rolbypassrls,rolconfig
 INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_contexto_actor_v1_candidato_externo';
 SELECT oid INTO esquema FROM pg_catalog.pg_namespace WHERE nspname='vec_contexto_actor_v1';
 SELECT oid INTO base FROM pg_catalog.pg_database WHERE datname=current_database();
 funciones:=ARRAY[
  pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1()'),
  pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(text,text,text,text,timestamptz)'),
  pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(text,text,text,text,timestamptz)')];
 SELECT count(*) INTO n FROM pg_catalog.pg_auth_members WHERE member=l.oid;
 IF l.oid IS NULL OR g.oid IS NULL OR esquema IS NULL OR base IS NULL OR array_position(funciones,NULL) IS NOT NULL
   OR l.rolcanlogin IS NOT TRUE OR l.rolsuper OR NOT l.rolinherit OR l.rolcreaterole OR l.rolcreatedb
   OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
   OR g.rolcanlogin OR g.rolsuper OR g.rolinherit OR g.rolcreaterole OR g.rolcreatedb
   OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
   OR current_setting('role')<>'none' OR n<>1
   OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid
       AND NOT admin_option AND inherit_option AND NOT set_option)
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>l.oid AND r.oid<>g.oid AND pg_catalog.pg_has_role(l.oid,r.oid,'MEMBER'))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>g.oid AND pg_catalog.pg_has_role(g.oid,r.oid,'MEMBER'))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_default_acl d LEFT JOIN LATERAL pg_catalog.aclexplode(coalesce(d.defaclacl,'{}'::aclitem[])) a ON true
      WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
   OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend d WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=l.oid)
   OR NOT coalesce((SELECT count(*)=5 AND bool_and(d.deptype='a' AND d.objsubid=0 AND (
       (d.classid='pg_catalog.pg_database'::regclass AND d.objid=base) OR
       (d.classid='pg_catalog.pg_namespace'::regclass AND d.objid=esquema) OR
       (d.classid='pg_catalog.pg_proc'::regclass AND d.objid=ANY(funciones))))
      FROM pg_catalog.pg_shdepend d WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=g.oid),false)
   OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable)
      FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(d.datacl,pg_catalog.acldefault('d',d.datdba))) a
      WHERE d.oid=base AND a.grantee=g.oid),false)
   OR NOT coalesce((SELECT count(*)=1 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable)
      FROM pg_catalog.pg_namespace d CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(d.nspacl,pg_catalog.acldefault('n',d.nspowner))) a
      WHERE d.oid=esquema AND a.grantee=g.oid),false)
   OR NOT coalesce((SELECT count(*)=3 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable)
      FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
      WHERE p.oid=ANY(funciones) AND a.grantee=g.oid),false)
   OR vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos(l.oid,base,esquema,funciones) IS NOT TRUE THEN
  RAISE EXCEPTION 'LOGIN candidato externo ContextoActor no acreditado' USING ERRCODE='42501';
 END IF;
 RETURN session_user;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1() FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1()
RETURNS TABLE(identidad_login text,acreditada boolean)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 identidad_login:=vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1();
 acreditada:=true; RETURN NEXT;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1() FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    login_oid oid; runtime_oid oid; esquema_oid oid; base_oid oid;
    membresias integer; funciones oid[]; login record; grupo record;
BEGIN
    IF pg_catalog.pg_has_role(session_user,'vec_contexto_actor_v1_candidato_externo','MEMBER') THEN
        RETURN vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1();
    END IF;
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

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz,
    p_proyecciones text[]
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE
    ahora timestamptz; coincidencias integer; cuenta record; perfil record;
    persona record; enlace record; enlaces_texto text; documento text;
    enlaces_procedencia_texto text; manifiesto_procedencia text;
    manifiesto_procedencia_bytes bytea; manifiesto_procedencia_huella text;
    canonica bytea; huella text; numero_enlaces integer; tipos integer; referencias integer;
    pide_empleado boolean; pide_candidato boolean; provision record; persona_perfil text; empleado record;
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    pide_candidato := pg_catalog.pg_has_role(session_user,'vec_contexto_actor_v1_candidato_externo','MEMBER');
    SELECT NULL::text AS provision_ref,NULL::numeric AS version,NULL::text AS huella_sha256,
           NULL::text AS candidato_ref INTO provision;
    IF current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'registro de contexto actor V2 requiere SERIALIZABLE de escritura';
    END IF;
    IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref, 'oca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref, 'rca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR p_metodo NOT IN ('certificado','dnie','sso','clave','kerberos_ad','demo')
       OR p_garantia NOT IN ('bajo','sustancial','alto')
       OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE
       OR p_proyecciones IS NULL
       OR (p_proyecciones IS DISTINCT FROM '{}'::text[]
           AND p_proyecciones IS DISTINCT FROM ARRAY['empleado']::text[]) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'solicitud de contexto actor V2 invalida';
    END IF;
    IF pide_candidato AND p_proyecciones <> '{}'::text[] THEN
        RAISE EXCEPTION 'proyeccion corporativa denegada al candidato externo' USING ERRCODE='42501';
    END IF;
    pide_empleado := p_proyecciones = ARRAY['empleado']::text[];

    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:operacion:v2:' || p_operacion_ref,0));
    IF pide_candidato THEN
        PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
        SELECT * INTO provision FROM vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(p_cuenta_ref,p_perfil_ref,pg_catalog.clock_timestamp());
        IF provision.provision_ref IS NULL THEN
            RAISE EXCEPTION 'provision candidata no vigente' USING ERRCODE='P0002';
        END IF;
    END IF;

    -- Idempotencia por operación, solicitud y alcance. El alcance del registro
    -- se deriva de su canon firmado; otro alcance es colisión.
    IF EXISTS (SELECT 1 FROM vec_contexto_actor_v1.registros_contexto r WHERE r.operacion_ref = p_operacion_ref) THEN
        RETURN QUERY
        SELECT r.operacion_ref,r.registro_contexto_ref,r.representacion_canonica,
               r.huella_sha256,r.manifiesto_procedencia_canonico,
               r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en
          FROM vec_contexto_actor_v1.registros_contexto r
         WHERE r.operacion_ref=p_operacion_ref AND r.cuenta_ref=p_cuenta_ref
           AND r.perfil_ref=p_perfil_ref AND r.metodo=p_metodo
           AND r.garantia=p_garantia AND r.solicitado_en=p_solicitado_en
           AND vec_contexto_actor_v1.alcance_registro_contexto_v2(r.representacion_canonica) = p_proyecciones
           AND pide_candidato = EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registro_candidato_externo_v1 e WHERE e.registro_contexto_ref=r.registro_contexto_ref)
           AND (NOT pide_candidato OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registro_candidato_externo_v1 e
                WHERE e.registro_contexto_ref=r.registro_contexto_ref AND e.provision_ref=provision.provision_ref
                  AND e.provision_version=provision.version AND e.provision_huella_sha256=provision.huella_sha256));
        IF NOT FOUND THEN
            RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'colision de operacion de contexto actor V2';
        END IF;
        RETURN;
    END IF;

    PERFORM 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
     WHERE ca.cuenta_ref=p_cuenta_ref ORDER BY ca.cuenta_ref FOR UPDATE OF ca;
    PERFORM 1 FROM vec_contexto_actor_v1.perfil_actual pa
     WHERE pa.perfil_ref=p_perfil_ref ORDER BY pa.perfil_ref FOR UPDATE OF pa;
    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
     WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ORDER BY va.vinculo_ref FOR UPDATE OF va;
    PERFORM 1 FROM vec_contexto_actor_v1.persona_actual pe
     WHERE pe.persona_ref IN (
       SELECT pv.persona_ref FROM vec_contexto_actor_v1.perfil_actual pa
       JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
       WHERE pa.perfil_ref=p_perfil_ref
       UNION
       SELECT vv.persona_ref FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ) ORDER BY pe.persona_ref FOR UPDATE OF pe;
    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones rv USING (vinculo_ref,version)
     WHERE rv.persona_ref IN (
       SELECT pv.persona_ref FROM vec_contexto_actor_v1.perfil_actual pa
       JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
       WHERE pa.perfil_ref=p_perfil_ref
       UNION
       SELECT vv.persona_ref FROM vec_contexto_actor_v1.vinculo_contexto_actual va
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
       WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref
     ) ORDER BY ra.vinculo_ref FOR UPDATE OF ra;
    -- Barrera de Personal antes del reloj y antes de leer su historia:
    -- consultivo compartido por persona y generación FOR SHARE. Una
    -- publicación confirmada tras la instantánea SERIALIZABLE provoca 40001.
    IF pide_empleado THEN
        SELECT pv.persona_ref INTO persona_perfil
          FROM vec_contexto_actor_v1.perfil_actual pa
          JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
         WHERE pa.perfil_ref=p_perfil_ref;
        IF persona_perfil IS NOT NULL THEN
            PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona_perfil);
        END IF;
    END IF;

    ahora := pg_catalog.clock_timestamp();
    IF ahora < p_solicitado_en OR ahora > p_solicitado_en + interval '5 seconds' THEN
        RAISE EXCEPTION USING ERRCODE = '57014', MESSAGE = 'ventana fresca de contexto actor V2 agotada';
    END IF;

    SELECT count(*) INTO coincidencias
      FROM vec_contexto_actor_v1.vinculo_contexto_actual a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
        ON v.vinculo_ref=a.vinculo_ref AND v.version=a.version
     WHERE v.cuenta_ref=p_cuenta_ref AND v.perfil_ref=p_perfil_ref;
    IF coincidencias <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contexto actor V2 no resuelto';
    END IF;

    SELECT cv.version, cv.procedencia_ref,cv.procedencia_version,
           cv.procedencia_huella_sha256,cv.procedencia_autoridad,
           cv.estado, cv.vigente_desde, cv.vigente_hasta
      INTO STRICT cuenta
      FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv USING (cuenta_ref,version)
     WHERE ca.cuenta_ref=p_cuenta_ref;
    SELECT pv.version, pv.persona_ref,pv.procedencia_ref,pv.procedencia_version,
           pv.procedencia_huella_sha256,pv.procedencia_autoridad,
           pv.estado, pv.vigente_desde, pv.vigente_hasta
      INTO STRICT perfil
      FROM vec_contexto_actor_v1.perfil_actual pa
      JOIN vec_contexto_actor_v1.perfil_versiones pv USING (perfil_ref,version)
     WHERE pa.perfil_ref=p_perfil_ref;
    SELECT vv.vinculo_ref, vv.version, vv.persona_ref,
           vv.procedencia_ref,vv.procedencia_version,vv.procedencia_huella_sha256,
           vv.procedencia_autoridad,vv.estado, vv.vigente_desde, vv.vigente_hasta
      INTO STRICT enlace
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv USING (vinculo_ref,version)
     WHERE vv.cuenta_ref=p_cuenta_ref AND vv.perfil_ref=p_perfil_ref;
    SELECT pv.version,pv.procedencia_ref,pv.procedencia_version,
           pv.procedencia_huella_sha256,pv.procedencia_autoridad,
           pv.estado, pv.vigente_desde, pv.vigente_hasta
      INTO STRICT persona
      FROM vec_contexto_actor_v1.persona_actual pa
      JOIN vec_contexto_actor_v1.persona_versiones pv USING (persona_ref,version)
     WHERE pa.persona_ref=perfil.persona_ref;

    IF perfil.persona_ref <> enlace.persona_ref
       OR cuenta.estado <> 'activo' OR perfil.estado <> 'activo'
       OR persona.estado <> 'activo' OR enlace.estado <> 'activo'
       OR cuenta.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR perfil.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR persona.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR enlace.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       OR ahora < cuenta.vigente_desde OR ahora >= cuenta.vigente_hasta
       OR ahora < perfil.vigente_desde OR ahora >= perfil.vigente_hasta
       OR ahora < persona.vigente_desde OR ahora >= persona.vigente_hasta
       OR ahora < enlace.vigente_desde OR ahora >= enlace.vigente_hasta THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contexto actor V2 no vigente';
    END IF;

    -- Filtros temporales de 000006 intactos. Con alcance {empleado} los
    -- punteros de empleado del núcleo quedan fuera: Personal es la autoridad.
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
      FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING (vinculo_ref,version)
     WHERE vr.persona_ref=perfil.persona_ref
       AND vr.estado='activo' AND ahora >= vr.vigente_desde
       AND ahora < vr.vigente_hasta
       AND (NOT pide_empleado OR vr.tipo <> 'empleado')
       AND (NOT pide_candidato OR vr.tipo = 'candidato');
    IF (pide_candidato AND (numero_enlaces <> 1 OR NOT EXISTS (
         SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
         JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING(vinculo_ref,version)
         WHERE vr.persona_ref=perfil.persona_ref AND vr.tipo='candidato' AND vr.referencia=provision.candidato_ref
           AND vr.estado='activo' AND ahora>=vr.vigente_desde AND ahora<vr.vigente_hasta)))
       OR numero_enlaces > 128 OR tipos <> numero_enlaces OR referencias <> numero_enlaces
       OR EXISTS (
           SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual ra
           JOIN vec_contexto_actor_v1.vinculo_referencia_versiones vr USING (vinculo_ref,version)
           WHERE vr.persona_ref=perfil.persona_ref
             AND vr.estado='activo' AND ahora >= vr.vigente_desde
             AND ahora < vr.vigente_hasta
             AND (NOT pide_empleado OR vr.tipo <> 'empleado')
             AND (NOT pide_candidato OR vr.tipo = 'candidato')
             AND vr.procedencia_autoridad <> 'autoridad_maestra_acreditada'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'referencias de contexto actor V2 no vigentes';
    END IF;

    IF pide_empleado THEN
        SELECT * INTO STRICT empleado
          FROM vec_contexto_actor_v1.proyeccion_empleado_personal_v2(perfil.persona_ref,ahora);
        IF empleado.resultado = 'sin_empleado' THEN
            RAISE EXCEPTION USING ERRCODE = 'PCA01', MESSAGE = 'proyeccion empleado pedida sin empleado canonico';
        ELSIF empleado.resultado <> 'empleado' THEN
            RAISE EXCEPTION USING ERRCODE = 'PCA02', MESSAGE = 'proyeccion empleado pedida ambigua';
        END IF;
        IF numero_enlaces >= 128 THEN
            RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'referencias de contexto actor V2 no vigentes';
        END IF;
        -- 'empleado' es el mayor tipo admitido y el núcleo ya no aporta
        -- ninguno: añadirlo al final conserva el orden canónico.
        enlaces_texto := concat_ws(',', enlaces_texto, format(
             '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"estado":"activo","vigente_desde":%s,"vigente_hasta":%s}',
             to_json(empleado.proyeccion_ref)::text, empleado.version::text,
             to_json(empleado.empleado_ref)::text,
             to_json(to_char(empleado.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
             to_json(to_char(empleado.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text));
        enlaces_procedencia_texto := concat_ws(',', enlaces_procedencia_texto, format(
             '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":"autoridad_maestra_acreditada"}',
             to_json(empleado.proyeccion_ref)::text, empleado.version::text,
             to_json(empleado.empleado_ref)::text, to_json(empleado.procedencia_ref)::text,
             empleado.procedencia_version::text, to_json(empleado.procedencia_huella_sha256)::text));
    END IF;

    manifiesto_procedencia := format(
      '{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada","cuenta":{"cuenta_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"persona":{"persona_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"perfil":{"perfil_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"contexto":{"vinculo_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"vinculos":[%s]}',
      to_json(p_cuenta_ref)::text,cuenta.version::text,to_json(cuenta.procedencia_ref)::text,
      cuenta.procedencia_version::text,to_json(cuenta.procedencia_huella_sha256)::text,to_json(cuenta.procedencia_autoridad)::text,
      to_json(perfil.persona_ref)::text,persona.version::text,to_json(persona.procedencia_ref)::text,
      persona.procedencia_version::text,to_json(persona.procedencia_huella_sha256)::text,to_json(persona.procedencia_autoridad)::text,
      to_json(p_perfil_ref)::text,perfil.version::text,to_json(perfil.procedencia_ref)::text,
      perfil.procedencia_version::text,to_json(perfil.procedencia_huella_sha256)::text,to_json(perfil.procedencia_autoridad)::text,
      to_json(enlace.vinculo_ref)::text,enlace.version::text,to_json(enlace.procedencia_ref)::text,
      enlace.procedencia_version::text,to_json(enlace.procedencia_huella_sha256)::text,to_json(enlace.procedencia_autoridad)::text,
      coalesce(enlaces_procedencia_texto,''));
    manifiesto_procedencia_bytes := convert_to(manifiesto_procedencia,'UTF8');
    manifiesto_procedencia_huella := encode(pg_catalog.sha256(manifiesto_procedencia_bytes),'hex');

    documento := format(
      '{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":%s,"garantia":%s,"perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
      to_json(perfil.persona_ref)::text, to_json(p_metodo)::text, to_json(p_garantia)::text,
      to_json(p_perfil_ref)::text, to_json(perfil.persona_ref)::text,
      to_json(enlace.vinculo_ref)::text, enlace.version::text, to_json(p_cuenta_ref)::text, cuenta.version::text,
      persona.version::text, perfil.version::text, to_json(enlace.estado)::text,
      to_json(to_char(enlace.vigente_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(enlace.vigente_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      to_json(to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
      coalesce(enlaces_texto,''));
    canonica := convert_to(documento,'UTF8');
    IF octet_length(canonica) > 65536 THEN
        RAISE EXCEPTION USING ERRCODE = '22001', MESSAGE = 'snapshot de contexto actor V2 excede cota';
    END IF;
    IF vec_contexto_actor_v1.alcance_registro_contexto_v2(canonica) IS DISTINCT FROM p_proyecciones THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'alcance de contexto actor V2 no reconstruible';
    END IF;
    huella := encode(pg_catalog.sha256(canonica),'hex');
    INSERT INTO vec_contexto_actor_v1.registros_contexto(
        operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,
        solicitado_en,resuelto_en,representacion_canonica,huella_sha256,
        manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva
    ) VALUES (
        p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,p_metodo,p_garantia,
        p_solicitado_en,ahora,canonica,huella,
        manifiesto_procedencia_bytes,manifiesto_procedencia_huella,'autoridad_maestra_acreditada'
    );
    IF pide_candidato THEN
        INSERT INTO vec_contexto_actor_v1.registro_candidato_externo_v1
          (registro_contexto_ref,operacion_ref,provision_ref,provision_version,provision_huella_sha256,candidato_ref)
        VALUES(p_registro_contexto_ref,p_operacion_ref,provision.provision_ref,provision.version,provision.huella_sha256,provision.candidato_ref);
    END IF;
    RETURN QUERY SELECT p_operacion_ref,p_registro_contexto_ref,canonica,huella,
      manifiesto_procedencia_bytes,manifiesto_procedencia_huella,
      'autoridad_maestra_acreditada'::text,ahora;
END
$f$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
    p_operacion_ref text, p_registro_contexto_ref text, p_cuenta_ref text,
    p_perfil_ref text, p_metodo text, p_garantia text, p_solicitado_en timestamptz,
    p_proyecciones text[]
) RETURNS TABLE (
    operacion_ref text, registro_contexto_ref text, representacion_canonica bytea,
    huella_sha256 text, manifiesto_procedencia_canonico bytea,
    manifiesto_procedencia_huella_sha256 text, autoridad_efectiva text,
    resuelto_en timestamptz
) LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog AS $f$
DECLARE pide_candidato boolean; provision record;
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    pide_candidato := pg_catalog.pg_has_role(session_user,'vec_contexto_actor_v1_candidato_externo','MEMBER');
    SELECT NULL::text AS provision_ref,NULL::numeric AS version,NULL::text AS huella_sha256,
           NULL::text AS candidato_ref INTO provision;
    IF current_setting('transaction_isolation') <> 'read committed'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'reconciliacion de contexto actor V2 requiere READ COMMITTED de escritura';
    END IF;
    IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion_ref, 'oca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_operacion_valida(p_registro_contexto_ref, 'rca_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR p_metodo NOT IN ('certificado','dnie','sso','clave','kerberos_ad','demo')
       OR p_garantia NOT IN ('bajo','sustancial','alto')
       OR vec_contexto_actor_v1.instante_valido(p_solicitado_en) IS NOT TRUE
       OR p_proyecciones IS NULL
       OR (p_proyecciones IS DISTINCT FROM '{}'::text[]
           AND p_proyecciones IS DISTINCT FROM ARRAY['empleado']::text[]) THEN
        RAISE EXCEPTION USING ERRCODE = '22023',
            MESSAGE = 'reconciliacion de contexto actor V2 invalida';
    END IF;
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
      'vec_contexto_actor_v1:operacion:v2:' || p_operacion_ref,0));
    IF pide_candidato THEN
        IF p_proyecciones <> '{}'::text[] THEN RETURN; END IF;
        PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
        SELECT * INTO provision FROM vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(p_cuenta_ref,p_perfil_ref,pg_catalog.clock_timestamp());
        IF provision.provision_ref IS NULL THEN RETURN; END IF;
    END IF;
    RETURN QUERY
    SELECT r.operacion_ref, r.registro_contexto_ref, r.representacion_canonica,
           r.huella_sha256,r.manifiesto_procedencia_canonico,
           r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva,r.resuelto_en
      FROM vec_contexto_actor_v1.registros_contexto AS r
     WHERE r.operacion_ref = p_operacion_ref
       AND r.registro_contexto_ref = p_registro_contexto_ref
       AND r.cuenta_ref = p_cuenta_ref AND r.perfil_ref = p_perfil_ref
       AND r.metodo = p_metodo AND r.garantia = p_garantia
       AND r.solicitado_en = p_solicitado_en
       AND vec_contexto_actor_v1.alcance_registro_contexto_v2(r.representacion_canonica) = p_proyecciones
       AND pide_candidato = EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registro_candidato_externo_v1 e WHERE e.registro_contexto_ref=r.registro_contexto_ref)
       AND (NOT pide_candidato OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registro_candidato_externo_v1 e
          WHERE e.registro_contexto_ref=r.registro_contexto_ref AND e.provision_ref=provision.provision_ref
            AND e.provision_version=provision.version AND e.provision_huella_sha256=provision.huella_sha256));
END
$f$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
    p_registro_contexto_ref text,
    p_contexto_actor_esquema text,
    p_contexto_actor_huella_sha256 text,
    p_manifiesto_procedencia_huella_sha256 text,
    p_autoridad_efectiva text,
    p_cuenta_ref text,
    p_cuenta_version numeric,
    p_persona_ref text,
    p_persona_version numeric,
    p_perfil_ref text,
    p_perfil_version numeric,
    p_contexto_actor_ref text,
    p_contexto_actor_version numeric,
    p_metodo text,
    p_garantia text,
    p_emitida_en timestamptz,
    p_valida_hasta timestamptz
) RETURNS timestamptz
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    registro record;
    cuenta record;
    perfil record;
    persona record;
    contexto record;
    generacion_observada numeric;
    ahora timestamptz;
    coincidencias integer;
    numero_vinculos integer;
    tipos integer;
    referencias integer;
    vinculos_texto text;
    vinculos_procedencia_texto text;
    representacion_texto text;
    representacion_reconstruida bytea;
    manifiesto_texto text;
    manifiesto_reconstruido bytea;
    pide_empleado boolean; pide_candidato boolean; provision record; marca record;
    empleado record;
BEGIN
    SELECT NULL::text AS provision_ref,NULL::numeric AS version,NULL::text AS huella_sha256,
           NULL::text AS candidato_ref INTO provision;
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'acreditacion de uso ContextoActor V2 requiere SERIALIZABLE de escritura';
    END IF;

    IF vec_contexto_actor_v1.referencia_operacion_valida(
           p_registro_contexto_ref, 'rca_'
       ) IS NOT TRUE
       OR p_contexto_actor_esquema IS DISTINCT FROM
          'vec.contexto-actor.vinculado.v2'
       OR p_contexto_actor_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_manifiesto_procedencia_huella_sha256 !~ '^[0-9a-f]{64}$'
       OR p_autoridad_efectiva IS DISTINCT FROM
          'autoridad_maestra_acreditada'
       OR vec_contexto_actor_v1.referencia_valida(
           p_cuenta_ref, 'cta_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_persona_ref, 'per_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_perfil_ref, 'prf_'
       ) IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(
           p_contexto_actor_ref, 'vca_'
       ) IS NOT TRUE
       OR p_cuenta_version IS NULL OR pg_catalog.scale(p_cuenta_version) <> 0
       OR p_cuenta_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_persona_version IS NULL OR pg_catalog.scale(p_persona_version) <> 0
       OR p_persona_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_perfil_version IS NULL OR pg_catalog.scale(p_perfil_version) <> 0
       OR p_perfil_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_contexto_actor_version IS NULL
       OR pg_catalog.scale(p_contexto_actor_version) <> 0
       OR p_contexto_actor_version NOT BETWEEN
          1 AND 18446744073709551615::numeric
       OR p_metodo NOT IN (
           'certificado', 'dnie', 'sso', 'clave', 'kerberos_ad', 'demo'
       )
       OR p_garantia NOT IN ('bajo', 'sustancial', 'alto')
       OR vec_contexto_actor_v1.instante_valido(p_emitida_en) IS NOT TRUE
       OR vec_contexto_actor_v1.instante_valido(p_valida_hasta) IS NOT TRUE
       OR p_valida_hasta <= p_emitida_en THEN
        RETURN NULL;
    END IF;

    SELECT r.operacion_ref, r.registro_contexto_ref, r.cuenta_ref,
           r.perfil_ref, r.metodo, r.garantia, r.solicitado_en,
           r.resuelto_en, r.representacion_canonica, r.huella_sha256,
           r.manifiesto_procedencia_canonico,
           r.manifiesto_procedencia_huella_sha256,
           r.autoridad_efectiva
      INTO registro
      FROM vec_contexto_actor_v1.registros_contexto AS r
     WHERE r.registro_contexto_ref = p_registro_contexto_ref
     FOR SHARE OF r;
    IF NOT FOUND
       OR registro.cuenta_ref IS DISTINCT FROM p_cuenta_ref
       OR registro.perfil_ref IS DISTINCT FROM p_perfil_ref
       OR registro.metodo IS DISTINCT FROM p_metodo
       OR registro.garantia IS DISTINCT FROM p_garantia
       OR registro.huella_sha256 IS DISTINCT FROM
          p_contexto_actor_huella_sha256
       OR registro.manifiesto_procedencia_huella_sha256 IS DISTINCT FROM
          p_manifiesto_procedencia_huella_sha256
       OR registro.autoridad_efectiva IS DISTINCT FROM p_autoridad_efectiva
       OR pg_catalog.encode(
           pg_catalog.sha256(registro.representacion_canonica), 'hex'
       ) IS DISTINCT FROM registro.huella_sha256
       OR pg_catalog.encode(
           pg_catalog.sha256(registro.manifiesto_procedencia_canonico), 'hex'
       ) IS DISTINCT FROM registro.manifiesto_procedencia_huella_sha256
       OR registro.resuelto_en > p_emitida_en THEN
        RETURN NULL;
    END IF;
    -- El alcance procede del canon firmado; no lo aporta el consumidor.
    pide_empleado := vec_contexto_actor_v1.alcance_registro_contexto_v2(
        registro.representacion_canonica) = ARRAY['empleado']::text[];
    SELECT * INTO marca FROM vec_contexto_actor_v1.registro_candidato_externo_v1
      WHERE registro_contexto_ref=p_registro_contexto_ref;
    pide_candidato := FOUND;
    IF pide_candidato AND pide_empleado THEN RETURN NULL; END IF;

    -- El advisory global hace que todo mutador entre por su BEFORE STATEMENT
    -- antes de tocar filas. La seguridad frente a snapshots obsoletos no
    -- depende de este lock: la acredita la fila MVCC leida mas abajo.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_contexto_actor_v1:mutacion_punteros_actuales:v2', 0
        )
    );

    -- Orden identico al resolutor: cuenta, perfil, candidatos de contexto,
    -- personas y referencias de modulo. Se toma el reloj solo al final.
    SELECT v.version, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO cuenta
      FROM vec_contexto_actor_v1.proyeccion_cuenta_actual AS a
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones AS v
        USING (cuenta_ref, version)
     WHERE a.cuenta_ref = p_cuenta_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    SELECT v.version, v.persona_ref, v.procedencia_ref,
           v.procedencia_version, v.procedencia_huella_sha256,
           v.procedencia_autoridad, v.estado, v.vigente_desde,
           v.vigente_hasta
      INTO perfil
      FROM vec_contexto_actor_v1.perfil_actual AS a
      JOIN vec_contexto_actor_v1.perfil_versiones AS v
        USING (perfil_ref, version)
     WHERE a.perfil_ref = p_perfil_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_contexto_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref
     ORDER BY a.vinculo_ref
     FOR UPDATE OF a;
    GET DIAGNOSTICS coincidencias = ROW_COUNT;
    IF coincidencias <> 1 THEN RETURN NULL; END IF;

    SELECT v.vinculo_ref, v.version, v.cuenta_ref, v.perfil_ref,
           v.persona_ref, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO contexto
      FROM vec_contexto_actor_v1.vinculo_contexto_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref;

    SELECT v.version, v.procedencia_ref, v.procedencia_version,
           v.procedencia_huella_sha256, v.procedencia_autoridad,
           v.estado, v.vigente_desde, v.vigente_hasta
      INTO persona
      FROM vec_contexto_actor_v1.persona_actual AS a
      JOIN vec_contexto_actor_v1.persona_versiones AS v
        USING (persona_ref, version)
     WHERE a.persona_ref = p_persona_ref
     FOR UPDATE OF a;
    IF NOT FOUND THEN RETURN NULL; END IF;

    PERFORM 1
      FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.persona_ref = p_persona_ref
     ORDER BY a.vinculo_ref
     FOR UPDATE OF a;

    -- Debe ocurrir despues de bloquear y releer todos los punteros. Si una
    -- mutacion comprometio despues del snapshot SERIALIZABLE, FOR SHARE no
    -- puede bloquear la version nueva invisible y PostgreSQL fuerza 40001.
    -- Si la acreditacion obtuvo primero el advisory, el mutador espera y queda
    -- serializado despues de su COMMIT.
    SELECT generacion
      INTO STRICT generacion_observada
      FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
     WHERE control_id = true
     FOR SHARE;

    -- Barrera de Personal antes del reloj: el consultivo solo ordena frente
    -- al publicador; la generación FOR SHARE impide acreditar con la
    -- instantánea anterior a una publicación ya confirmada (40001).
    IF pide_empleado THEN
        PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(p_persona_ref);
    END IF;

    ahora := pg_catalog.clock_timestamp();
    IF pide_candidato THEN
        SELECT * INTO provision FROM vec_contexto_actor_v1.provision_candidato_externo_vigente_v1(p_cuenta_ref,p_perfil_ref,ahora);
        IF provision.provision_ref IS DISTINCT FROM marca.provision_ref
           OR provision.version IS DISTINCT FROM marca.provision_version
           OR provision.huella_sha256 IS DISTINCT FROM marca.provision_huella_sha256
           OR provision.candidato_ref IS DISTINCT FROM marca.candidato_ref
           OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.candidato_externo_versiones v
              WHERE v.provision_ref=marca.provision_ref AND v.version=marca.provision_version
                AND p_emitida_en>=v.vigente_desde AND p_valida_hasta<=v.vigente_hasta) THEN RETURN NULL;
        END IF;
    END IF;

    IF cuenta.version IS DISTINCT FROM p_cuenta_version
       OR perfil.version IS DISTINCT FROM p_perfil_version
       OR perfil.persona_ref IS DISTINCT FROM p_persona_ref
       OR persona.version IS DISTINCT FROM p_persona_version
       OR contexto.vinculo_ref IS DISTINCT FROM p_contexto_actor_ref
       OR contexto.version IS DISTINCT FROM p_contexto_actor_version
       OR contexto.persona_ref IS DISTINCT FROM p_persona_ref
       OR cuenta.estado <> 'activo' OR perfil.estado <> 'activo'
       OR persona.estado <> 'activo' OR contexto.estado <> 'activo'
       OR cuenta.procedencia_autoridad <> p_autoridad_efectiva
       OR perfil.procedencia_autoridad <> p_autoridad_efectiva
       OR persona.procedencia_autoridad <> p_autoridad_efectiva
       OR contexto.procedencia_autoridad <> p_autoridad_efectiva
       OR p_emitida_en < cuenta.vigente_desde
       OR p_valida_hasta > cuenta.vigente_hasta
       OR ahora < cuenta.vigente_desde OR ahora >= cuenta.vigente_hasta
       OR p_emitida_en < perfil.vigente_desde
       OR p_valida_hasta > perfil.vigente_hasta
       OR ahora < perfil.vigente_desde OR ahora >= perfil.vigente_hasta
       OR p_emitida_en < persona.vigente_desde
       OR p_valida_hasta > persona.vigente_hasta
       OR ahora < persona.vigente_desde OR ahora >= persona.vigente_hasta
       OR p_emitida_en < contexto.vigente_desde
       OR p_valida_hasta > contexto.vigente_hasta
       OR ahora < contexto.vigente_desde OR ahora >= contexto.vigente_hasta
       OR ahora < p_emitida_en OR ahora >= p_valida_hasta THEN
        RETURN NULL;
    END IF;

    SELECT pg_catalog.count(*), pg_catalog.count(DISTINCT v.tipo),
           pg_catalog.count(DISTINCT (v.tipo, v.referencia)),
           pg_catalog.string_agg(pg_catalog.format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s}',
             pg_catalog.to_json(v.vinculo_ref)::text, v.version::text,
             pg_catalog.to_json(v.tipo)::text,
             pg_catalog.to_json(v.referencia)::text,
             pg_catalog.to_json(v.estado)::text,
             pg_catalog.to_json(pg_catalog.to_char(
                 v.vigente_desde AT TIME ZONE 'UTC',
                 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
             ))::text,
             pg_catalog.to_json(pg_catalog.to_char(
                 v.vigente_hasta AT TIME ZONE 'UTC',
                 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
             ))::text
           ), ',' ORDER BY v.tipo, v.referencia, v.version, v.vinculo_ref),
           pg_catalog.string_agg(pg_catalog.format(
             '{"vinculo_ref":%s,"version":%s,"tipo":%s,"referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s}',
             pg_catalog.to_json(v.vinculo_ref)::text, v.version::text,
             pg_catalog.to_json(v.tipo)::text,
             pg_catalog.to_json(v.referencia)::text,
             pg_catalog.to_json(v.procedencia_ref)::text,
             v.procedencia_version::text,
             pg_catalog.to_json(v.procedencia_huella_sha256)::text,
             pg_catalog.to_json(v.procedencia_autoridad)::text
           ), ',' ORDER BY v.tipo, v.referencia, v.version, v.vinculo_ref)
      INTO numero_vinculos, tipos, referencias, vinculos_texto,
           vinculos_procedencia_texto
      FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
        USING (vinculo_ref, version)
     WHERE v.persona_ref = p_persona_ref
       AND v.estado = 'activo' AND ahora >= v.vigente_desde
       AND ahora < v.vigente_hasta
       AND (NOT pide_empleado OR v.tipo <> 'empleado')
       AND (NOT pide_candidato OR v.tipo = 'candidato');

    IF (pide_candidato AND (numero_vinculos<>1 OR NOT EXISTS(
         SELECT 1 FROM vec_contexto_actor_v1.vinculo_referencia_actual a
         JOIN vec_contexto_actor_v1.vinculo_referencia_versiones v USING(vinculo_ref,version)
         WHERE v.persona_ref=p_persona_ref AND v.tipo='candidato' AND v.referencia=provision.candidato_ref
           AND v.estado='activo' AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta)))
       OR numero_vinculos > 128 OR tipos <> numero_vinculos
       OR referencias <> numero_vinculos
       OR EXISTS (
           SELECT 1
             FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
             JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
               USING (vinculo_ref, version)
            WHERE v.persona_ref = p_persona_ref
              AND v.estado = 'activo' AND ahora >= v.vigente_desde
              AND ahora < v.vigente_hasta
              AND (NOT pide_empleado OR v.tipo <> 'empleado')
              AND (NOT pide_candidato OR v.tipo = 'candidato')
              AND (v.procedencia_autoridad <> p_autoridad_efectiva
                   OR p_emitida_en < v.vigente_desde
                   OR p_valida_hasta > v.vigente_hasta)
       ) THEN
        RETURN NULL;
    END IF;

    -- Revalidación en el instante autoritativo: la proyección de Personal debe
    -- seguir siendo la única efectiva y cubrir toda la vigencia del uso. Una
    -- revocación, baja, caducidad o nueva versión cambia los bytes y anula.
    IF pide_empleado THEN
        SELECT * INTO STRICT empleado
          FROM vec_contexto_actor_v1.proyeccion_empleado_personal_v2(p_persona_ref, ahora);
        IF empleado.resultado IS DISTINCT FROM 'empleado' OR numero_vinculos >= 128
           OR p_emitida_en < empleado.vigente_desde
           OR p_valida_hasta > empleado.vigente_hasta THEN
            RETURN NULL;
        END IF;
        vinculos_texto := concat_ws(',', vinculos_texto, pg_catalog.format(
          '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"estado":"activo","vigente_desde":%s,"vigente_hasta":%s}',
          pg_catalog.to_json(empleado.proyeccion_ref)::text, empleado.version::text,
          pg_catalog.to_json(empleado.empleado_ref)::text,
          pg_catalog.to_json(pg_catalog.to_char(
              empleado.vigente_desde AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text,
          pg_catalog.to_json(pg_catalog.to_char(
              empleado.vigente_hasta AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))::text));
        vinculos_procedencia_texto := concat_ws(',', vinculos_procedencia_texto, pg_catalog.format(
          '{"vinculo_ref":%s,"version":%s,"tipo":"empleado","referencia":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":"autoridad_maestra_acreditada"}',
          pg_catalog.to_json(empleado.proyeccion_ref)::text, empleado.version::text,
          pg_catalog.to_json(empleado.empleado_ref)::text,
          pg_catalog.to_json(empleado.procedencia_ref)::text,
          empleado.procedencia_version::text,
          pg_catalog.to_json(empleado.procedencia_huella_sha256)::text));
    END IF;

    manifiesto_texto := pg_catalog.format(
      '{"esquema":"vec.contexto-actor.procedencia-manifiesto.v1","autoridad_efectiva":"autoridad_maestra_acreditada","cuenta":{"cuenta_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"persona":{"persona_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"perfil":{"perfil_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"contexto":{"vinculo_ref":%s,"version":%s,"procedencia_ref":%s,"procedencia_version":%s,"procedencia_huella_sha256":%s,"procedencia_autoridad":%s},"vinculos":[%s]}',
      pg_catalog.to_json(p_cuenta_ref)::text, cuenta.version::text,
      pg_catalog.to_json(cuenta.procedencia_ref)::text,
      cuenta.procedencia_version::text,
      pg_catalog.to_json(cuenta.procedencia_huella_sha256)::text,
      pg_catalog.to_json(cuenta.procedencia_autoridad)::text,
      pg_catalog.to_json(p_persona_ref)::text, persona.version::text,
      pg_catalog.to_json(persona.procedencia_ref)::text,
      persona.procedencia_version::text,
      pg_catalog.to_json(persona.procedencia_huella_sha256)::text,
      pg_catalog.to_json(persona.procedencia_autoridad)::text,
      pg_catalog.to_json(p_perfil_ref)::text, perfil.version::text,
      pg_catalog.to_json(perfil.procedencia_ref)::text,
      perfil.procedencia_version::text,
      pg_catalog.to_json(perfil.procedencia_huella_sha256)::text,
      pg_catalog.to_json(perfil.procedencia_autoridad)::text,
      pg_catalog.to_json(contexto.vinculo_ref)::text,
      contexto.version::text,
      pg_catalog.to_json(contexto.procedencia_ref)::text,
      contexto.procedencia_version::text,
      pg_catalog.to_json(contexto.procedencia_huella_sha256)::text,
      pg_catalog.to_json(contexto.procedencia_autoridad)::text,
      coalesce(vinculos_procedencia_texto, '')
    );
    manifiesto_reconstruido := pg_catalog.convert_to(
        manifiesto_texto, 'UTF8'
    );

    representacion_texto := pg_catalog.format(
      '{"esquema":"vec.contexto-actor.vinculado.v2","principal_ref":%s,"metodo":%s,"garantia":%s,"perfil_activo_ref":%s,"persona_ref":%s,"contexto_actor_ref":%s,"contexto_version":%s,"cuenta_ref":%s,"cuenta_version":%s,"persona_version":%s,"perfil_version":%s,"estado":%s,"vigente_desde":%s,"vigente_hasta":%s,"resuelto_en":%s,"vinculos":[%s]}',
      pg_catalog.to_json(p_persona_ref)::text,
      pg_catalog.to_json(p_metodo)::text,
      pg_catalog.to_json(p_garantia)::text,
      pg_catalog.to_json(p_perfil_ref)::text,
      pg_catalog.to_json(p_persona_ref)::text,
      pg_catalog.to_json(contexto.vinculo_ref)::text,
      contexto.version::text,
      pg_catalog.to_json(p_cuenta_ref)::text,
      cuenta.version::text, persona.version::text, perfil.version::text,
      pg_catalog.to_json(contexto.estado)::text,
      pg_catalog.to_json(pg_catalog.to_char(
          contexto.vigente_desde AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      pg_catalog.to_json(pg_catalog.to_char(
          contexto.vigente_hasta AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      pg_catalog.to_json(pg_catalog.to_char(
          registro.resuelto_en AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      ))::text,
      coalesce(vinculos_texto, '')
    );
    representacion_reconstruida := pg_catalog.convert_to(
        representacion_texto, 'UTF8'
    );

    IF representacion_reconstruida IS DISTINCT FROM
          registro.representacion_canonica
       OR manifiesto_reconstruido IS DISTINCT FROM
          registro.manifiesto_procedencia_canonico
       OR pg_catalog.encode(
           pg_catalog.sha256(representacion_reconstruida), 'hex'
       ) IS DISTINCT FROM p_contexto_actor_huella_sha256
       OR pg_catalog.encode(
           pg_catalog.sha256(manifiesto_reconstruido), 'hex'
       ) IS DISTINCT FROM p_manifiesto_procedencia_huella_sha256 THEN
        RETURN NULL;
    END IF;

    RETURN ahora;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
        OR datetime_field_overflow OR no_data_found OR too_many_rows
        OR object_not_in_prerequisite_state THEN
        RETURN NULL;
END
$funcion$;

-- Cinco argumentos, sin método, garantía ni selector de proyecciones elegibles
-- por el cliente. El núcleo comparte recibos V2 pero marca la población.
CREATE FUNCTION vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(
 p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz
) RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
  p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,'certificado','alto',p_solicitado_en,'{}'::text[]);
END $f$;
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(
 p_operacion_ref text,p_registro_contexto_ref text,p_cuenta_ref text,p_perfil_ref text,p_solicitado_en timestamptz
) RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,
 huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,
 autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_candidato_externo_v1();
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
  p_operacion_ref,p_registro_contexto_ref,p_cuenta_ref,p_perfil_ref,'certificado','alto',p_solicitado_en,'{}'::text[]);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(text,text,text,text,timestamptz),
 vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(text,text,text,text,timestamptz) FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_contexto_actor_v1_candidato_externo;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1(),
 vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1(text,text,text,text,timestamptz),
 vec_contexto_actor_v1.reconciliar_contexto_candidato_externo_v1(text,text,text,text,timestamptz)
 TO vec_contexto_actor_v1_candidato_externo;
DO $postimagen$
DECLARE g oid:='vec_contexto_actor_v1_candidato_externo'::regrole;
BEGIN
 IF (SELECT count(*) FROM pg_catalog.pg_proc p JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a ON true
      WHERE a.grantee=g AND p.pronamespace='vec_contexto_actor_v1'::regnamespace)<>3
    OR pg_catalog.has_function_privilege(g,'vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(text,text,text,text,text,text,timestamptz,text[])','EXECUTE')
    OR pg_catalog.has_table_privilege(g,'vec_contexto_actor_v1.registro_candidato_externo_v1','SELECT')
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
       WHERE p.pronamespace='vec_contexto_actor_v1'::regnamespace AND a.grantee=0 AND p.proname IN
       ('resolver_contexto_candidato_externo_v1','reconciliar_contexto_candidato_externo_v1','perfil_candidato_externo_provisionado_v1')) THEN
  RAISE EXCEPTION 'ContextoActor 000012: postimagen ACL incompatible' USING ERRCODE='55000';
 END IF;
END $postimagen$;
COMMIT;
