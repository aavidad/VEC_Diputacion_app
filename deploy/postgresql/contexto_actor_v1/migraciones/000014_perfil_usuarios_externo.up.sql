\set ON_ERROR_STOP on
-- ContextoActor 000014: identidad inmutable y provisión propia de Usuarios
-- externo. Solo referencias opacas; ninguna tabla compartida recibe PII.
-- DOWN prohibido después de publicar versiones o decisiones AUT17.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
 'vec_contexto_actor_v1:migracion:perfil_usuarios_externo:000014',0));
DO $preimagen$
DECLARE dueno oid:=pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
        aut oid:=pg_catalog.to_regrole('vec_autorizacion_propietario');
        candidato oid:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_candidato_externo_v1(text,text,text)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR dueno IS NULL OR aut IS NULL OR candidato IS NULL
    OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=candidato) IS DISTINCT FROM dueno
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex')
          FROM pg_catalog.pg_proc WHERE oid=candidato)
       IS DISTINCT FROM '59666a279c930f338dede0024be19abd688e1f7727054f8a1127a8bbc07b6a57'
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.perfil_usuarios_externo_identidad') IS NOT NULL
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.perfil_usuarios_externo_versiones') IS NOT NULL
    OR pg_catalog.to_regclass('vec_contexto_actor_v1.perfil_usuarios_externo_actual') IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)') IS NOT NULL THEN
  RAISE EXCEPTION 'ContextoActor 000014: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $preimagen$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path=pg_catalog;
-- Un perfil conserva para siempre la misma cuenta, persona y vínculo. AUT17
-- puede revalidar por persona/perfil sin resucitar una cuenta anterior.
CREATE TABLE vec_contexto_actor_v1.perfil_usuarios_externo_identidad(
 perfil_ref text PRIMARY KEY CHECK(vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_')),
 provision_ref text NOT NULL UNIQUE CHECK(vec_contexto_actor_v1.referencia_valida(provision_ref,'pue_')),
 cuenta_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_')),
 persona_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(persona_ref,'per_')),
 contexto_ref text NOT NULL CHECK(vec_contexto_actor_v1.referencia_valida(contexto_ref,'vca_')),
 UNIQUE(provision_ref,perfil_ref,cuenta_ref,persona_ref,contexto_ref)
);
CREATE TABLE vec_contexto_actor_v1.perfil_usuarios_externo_versiones(
 provision_ref text NOT NULL,
 version numeric(20,0) NOT NULL CHECK(version BETWEEN 1 AND 18446744073709551615::numeric),
 cuenta_ref text NOT NULL,
 cuenta_version numeric(20,0) NOT NULL CHECK(cuenta_version BETWEEN 1 AND 18446744073709551615::numeric),
 perfil_ref text NOT NULL,
 perfil_version numeric(20,0) NOT NULL CHECK(perfil_version BETWEEN 1 AND 18446744073709551615::numeric),
 persona_ref text NOT NULL,
 persona_version numeric(20,0) NOT NULL CHECK(persona_version BETWEEN 1 AND 18446744073709551615::numeric),
 contexto_ref text NOT NULL,
 contexto_version numeric(20,0) NOT NULL CHECK(contexto_version BETWEEN 1 AND 18446744073709551615::numeric),
 estado text NOT NULL CHECK(estado IN('activo','revocado')),
 vigente_desde timestamptz NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_desde)),
 vigente_hasta timestamptz NOT NULL CHECK(vec_contexto_actor_v1.instante_valido(vigente_hasta)),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_huella_sha256 text NOT NULL CHECK(fuente_huella_sha256 ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(provision_ref,version), CHECK(vigente_hasta>vigente_desde),
 FOREIGN KEY(provision_ref,perfil_ref,cuenta_ref,persona_ref,contexto_ref)
  REFERENCES vec_contexto_actor_v1.perfil_usuarios_externo_identidad
  (provision_ref,perfil_ref,cuenta_ref,persona_ref,contexto_ref)
);
CREATE TABLE vec_contexto_actor_v1.perfil_usuarios_externo_actual(
 provision_ref text PRIMARY KEY,version numeric(20,0) NOT NULL,
 FOREIGN KEY(provision_ref,version) REFERENCES vec_contexto_actor_v1.perfil_usuarios_externo_versiones(provision_ref,version)
);
CREATE INDEX perfil_usuarios_externo_persona_idx ON vec_contexto_actor_v1.perfil_usuarios_externo_versiones(persona_ref,perfil_ref,provision_ref,version);
DO $rls$ DECLARE tabla text; BEGIN
 FOREACH tabla IN ARRAY ARRAY['perfil_usuarios_externo_identidad','perfil_usuarios_externo_versiones','perfil_usuarios_externo_actual'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('CREATE POLICY acceso_propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING (current_user=''vec_contexto_actor_v1_propietario'') WITH CHECK (current_user=''vec_contexto_actor_v1_propietario'')',tabla);
  EXECUTE pg_catalog.format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_contexto_actor_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado()',tabla);
 END LOOP;
END $rls$;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.perfil_usuarios_externo_identidad FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.perfil_usuarios_externo_versiones FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER serializar_mutacion_punteros_actuales_v2 BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.perfil_usuarios_externo_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
CREATE TRIGGER avanzar_generacion_punteros_actuales_v2 AFTER INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.perfil_usuarios_externo_actual FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();
REVOKE ALL ON TABLE vec_contexto_actor_v1.perfil_usuarios_externo_identidad,vec_contexto_actor_v1.perfil_usuarios_externo_versiones,vec_contexto_actor_v1.perfil_usuarios_externo_actual FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo,vec_autorizacion_propietario;
REVOKE ALL ON TYPE vec_contexto_actor_v1.perfil_usuarios_externo_identidad,vec_contexto_actor_v1.perfil_usuarios_externo_versiones,vec_contexto_actor_v1.perfil_usuarios_externo_actual FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo,vec_autorizacion_propietario;
-- Única lectura privada. El puntero de generación fuerza 40001 ante snapshot
-- anterior a una publicación/revocación que ganó el advisory exclusivo.
CREATE FUNCTION vec_contexto_actor_v1.perfil_usuarios_externo_vigente_v1(
 p_persona_ref text,p_perfil_ref text
) RETURNS TABLE(provision_ref text,version numeric,huella_sha256 text,cuenta_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE p record; n integer; ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE THEN RETURN; END IF;
 PERFORM 1 FROM vec_contexto_actor_v1.control_generacion_punteros_actuales_v2
  WHERE control_id=true FOR SHARE;
 IF NOT FOUND THEN RETURN; END IF;
 ahora:=pg_catalog.clock_timestamp();
 SELECT count(*) INTO n FROM vec_contexto_actor_v1.perfil_usuarios_externo_actual a
 JOIN vec_contexto_actor_v1.perfil_usuarios_externo_versiones v USING(provision_ref,version)
 WHERE v.persona_ref=p_persona_ref AND v.perfil_ref=p_perfil_ref
   AND v.estado='activo' AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta;
 IF n<>1 THEN RETURN; END IF;
 SELECT v.* INTO STRICT p FROM vec_contexto_actor_v1.perfil_usuarios_externo_actual a
 JOIN vec_contexto_actor_v1.perfil_usuarios_externo_versiones v USING(provision_ref,version)
 WHERE v.persona_ref=p_persona_ref AND v.perfil_ref=p_perfil_ref
   AND v.estado='activo' AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta;
 -- Un perfil Usuarios no puede tener ni haber tenido la cuenta/perfil de
 -- Bolsa candidata o una cuenta de uso corporativo.
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.candidato_externo_versiones c
           WHERE c.cuenta_ref=p.cuenta_ref OR c.perfil_ref=p.perfil_ref)
    OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones c
              WHERE c.cuenta_ref=p.cuenta_ref OR c.perfil_ref=p.perfil_ref)
    OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual a
        JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones v USING(cuenta_ref,version)
        WHERE a.cuenta_ref=p.cuenta_ref AND a.version=p.cuenta_version
          AND v.estado='activo' AND v.procedencia_autoridad='autoridad_maestra_acreditada'
          AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta)
    OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.persona_actual a
        JOIN vec_contexto_actor_v1.persona_versiones v USING(persona_ref,version)
        WHERE a.persona_ref=p.persona_ref AND a.version=p.persona_version
          AND v.estado='activo' AND v.procedencia_autoridad='autoridad_maestra_acreditada'
          AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta)
    OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_actual a
        JOIN vec_contexto_actor_v1.perfil_versiones v USING(perfil_ref,version)
        WHERE a.perfil_ref=p.perfil_ref AND a.version=p.perfil_version
          AND v.persona_ref=p.persona_ref AND v.estado='activo'
          AND v.procedencia_autoridad='autoridad_maestra_acreditada'
          AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta)
    OR NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
        JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
        WHERE a.vinculo_ref=p.contexto_ref AND a.version=p.contexto_version
          AND v.cuenta_ref=p.cuenta_ref AND v.perfil_ref=p.perfil_ref
          AND v.persona_ref=p.persona_ref AND v.estado='activo'
          AND v.procedencia_autoridad='autoridad_maestra_acreditada'
          AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta)
    OR (SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_actual a
        JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
        WHERE v.cuenta_ref=p.cuenta_ref AND v.perfil_ref=p.perfil_ref
          AND v.estado='activo' AND ahora>=v.vigente_desde AND ahora<v.vigente_hasta)<>1 THEN RETURN; END IF;
 RETURN QUERY SELECT p.provision_ref,p.version,p.huella_sha256,p.cuenta_ref;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.perfil_usuarios_externo_vigente_v1(text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo,vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
 p_cuenta_ref text,p_persona_ref text,p_perfil_ref text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE vigente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT * INTO vigente FROM vec_contexto_actor_v1.perfil_usuarios_externo_vigente_v1(p_persona_ref,p_perfil_ref);
 IF NOT FOUND THEN RETURN false; END IF;
 RETURN vigente.cuenta_ref IS NOT DISTINCT FROM p_cuenta_ref;
END $funcion$;
CREATE FUNCTION vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(
 p_persona_ref text,p_perfil_ref text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE vigente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off' THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT * INTO vigente FROM vec_contexto_actor_v1.perfil_usuarios_externo_vigente_v1(p_persona_ref,p_perfil_ref);
 RETURN FOUND;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text),
 vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text),
 vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)
 TO vec_autorizacion_propietario;
-- La autoridad interna aprueba el hash del material completo y la preimagen
-- CAS antes de publicar. Ningún LOGIN web hereda esta función.
CREATE FUNCTION vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1(
 p_provision_ref text,p_version_esperada numeric,p_huella_esperada text,p_huella_aprobada text,
 p_cuenta_ref text,p_cuenta_version numeric,p_persona_ref text,p_persona_version numeric,
 p_perfil_ref text,p_perfil_version numeric,p_contexto_ref text,p_contexto_version numeric,
 p_estado text,p_desde timestamptz,p_hasta timestamptz,p_fuente_huella_sha256 text
) RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $funcion$
DECLARE anterior record; nueva_version numeric; nueva_huella text; vigente record; ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off'
    OR vec_contexto_actor_v1.referencia_valida(p_provision_ref,'pue_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
    OR vec_contexto_actor_v1.referencia_valida(p_contexto_ref,'vca_') IS NOT TRUE
    OR p_version_esperada IS NULL OR p_version_esperada<0
    OR p_version_esperada<>pg_catalog.trunc(p_version_esperada)
    OR p_version_esperada>=18446744073709551615::numeric
    OR p_cuenta_version IS NULL OR p_cuenta_version<1 OR p_cuenta_version<>pg_catalog.trunc(p_cuenta_version)
    OR p_persona_version IS NULL OR p_persona_version<1 OR p_persona_version<>pg_catalog.trunc(p_persona_version)
    OR p_perfil_version IS NULL OR p_perfil_version<1 OR p_perfil_version<>pg_catalog.trunc(p_perfil_version)
    OR p_contexto_version IS NULL OR p_contexto_version<1 OR p_contexto_version<>pg_catalog.trunc(p_contexto_version)
    OR p_estado NOT IN('activo','revocado')
    OR vec_contexto_actor_v1.instante_valido(p_desde) IS NOT TRUE
    OR vec_contexto_actor_v1.instante_valido(p_hasta) IS NOT TRUE OR p_hasta<=p_desde
    OR p_huella_aprobada !~ '^[0-9a-f]{64}$'
    OR p_fuente_huella_sha256 !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'provisión Usuarios externo inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT a.version,v.huella_sha256,v.estado,v.vigente_hasta
 INTO anterior FROM vec_contexto_actor_v1.perfil_usuarios_externo_actual a
 JOIN vec_contexto_actor_v1.perfil_usuarios_externo_versiones v USING(provision_ref,version)
 WHERE a.provision_ref=p_provision_ref FOR UPDATE OF a;
 IF (NOT FOUND AND (p_version_esperada<>0 OR p_huella_esperada IS NOT NULL))
    OR (FOUND AND (anterior.version<>p_version_esperada OR anterior.huella_sha256 IS DISTINCT FROM p_huella_esperada)) THEN
  RAISE EXCEPTION 'provisión Usuarios externo: CAS divergente' USING ERRCODE='40001';
 END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF p_estado='activo' AND (ahora<p_desde OR ahora>=p_hasta
       OR (p_version_esperada>0 AND (anterior.estado<>'activo' OR ahora>=anterior.vigente_hasta))) THEN
  RAISE EXCEPTION 'perfil Usuarios externo no reactivable' USING ERRCODE='55000';
 END IF;
 nueva_version:=p_version_esperada+1;
 nueva_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_array(
  p_provision_ref,nueva_version,p_cuenta_ref,p_cuenta_version,p_persona_ref,p_persona_version,
  p_perfil_ref,p_perfil_version,p_contexto_ref,p_contexto_version,p_estado,p_desde,p_hasta,
  p_fuente_huella_sha256)::text,'UTF8')),'hex');
 IF nueva_huella IS DISTINCT FROM p_huella_aprobada THEN
  RAISE EXCEPTION 'provisión Usuarios externo: huella aprobada divergente' USING ERRCODE='42501';
 END IF;
 IF p_version_esperada=0 THEN
  INSERT INTO vec_contexto_actor_v1.perfil_usuarios_externo_identidad
   (perfil_ref,provision_ref,cuenta_ref,persona_ref,contexto_ref)
  VALUES(p_perfil_ref,p_provision_ref,p_cuenta_ref,p_persona_ref,p_contexto_ref);
 END IF;
 INSERT INTO vec_contexto_actor_v1.perfil_usuarios_externo_versiones VALUES(
  p_provision_ref,nueva_version,p_cuenta_ref,p_cuenta_version,p_perfil_ref,p_perfil_version,
  p_persona_ref,p_persona_version,p_contexto_ref,p_contexto_version,p_estado,p_desde,p_hasta,
  nueva_huella,p_fuente_huella_sha256);
 INSERT INTO vec_contexto_actor_v1.perfil_usuarios_externo_actual VALUES(p_provision_ref,nueva_version)
 ON CONFLICT(provision_ref) DO UPDATE SET version=excluded.version;
 IF p_estado='activo' THEN
  SELECT * INTO vigente FROM vec_contexto_actor_v1.perfil_usuarios_externo_vigente_v1(p_persona_ref,p_perfil_ref);
  IF NOT FOUND OR vigente.provision_ref IS DISTINCT FROM p_provision_ref
       OR vigente.version IS DISTINCT FROM nueva_version
       OR vigente.huella_sha256 IS DISTINCT FROM nueva_huella
       OR vigente.cuenta_ref IS DISTINCT FROM p_cuenta_ref THEN
   RAISE EXCEPTION 'provisión Usuarios externo sin perfil único vigente' USING ERRCODE='55000';
  END IF;
 END IF;
 RETURN nueva_huella;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.publicar_perfil_usuarios_externo_v1(
 text,numeric,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,timestamptz,timestamptz,text)
 FROM PUBLIC,vec_contexto_actor_v1_runtime,vec_contexto_actor_v1_candidato_externo,vec_autorizacion_propietario;
RESET ROLE;
DO $postimagen$
DECLARE dueno oid:='vec_contexto_actor_v1_propietario'::regrole;
        aut oid:='vec_autorizacion_propietario'::regrole;
        f oid; tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['perfil_usuarios_externo_identidad','perfil_usuarios_externo_versiones','perfil_usuarios_externo_actual'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=pg_catalog.to_regclass('vec_contexto_actor_v1.'||tabla)
       AND c.relowner=dueno AND c.relrowsecurity AND c.relforcerowsecurity)
     OR pg_catalog.has_table_privilege(aut,pg_catalog.to_regclass('vec_contexto_actor_v1.'||tabla),'SELECT') THEN
   RAISE EXCEPTION 'ContextoActor 000014: tabla o RLS incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
 FOR f IN SELECT pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)')
          UNION ALL SELECT pg_catalog.to_regprocedure('vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)') LOOP
  IF f IS NULL OR NOT pg_catalog.has_function_privilege(aut,f,'EXECUTE')
     OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime',f,'EXECUTE')
     OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_candidato_externo',f,'EXECUTE')
     OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
          CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
          WHERE p.oid=f AND (a.grantee=0 OR a.grantee NOT IN(dueno,aut)
              OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
   RAISE EXCEPTION 'ContextoActor 000014: ACL incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $postimagen$;
COMMIT;
