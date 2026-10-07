\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_identidad_sesiones_v1:migracion:000009',0));

-- Preimagen causal: propietario, superficie ACL y protección de las tablas
-- que consume esta migración. No depende del LOGIN o entorno del ensayo.
DO $frontera_preimagen$
DECLARE nombre text;tabla regclass;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_identidad_sesiones_v1_propietario' AND NOT(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_identidad_sesiones_v1' AND nspowner=to_regrole('vec_identidad_sesiones_v1_propietario'))
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='vec_identidad_sesiones_v1' AND (a.grantee=0 OR (a.privilege_type='CREATE' AND a.grantee<>n.nspowner)))
 THEN RAISE EXCEPTION 'ADMIN: namespace o propietario divergente' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['cuenta','estado_cuenta','estado_cuenta_actual'] LOOP
  tabla:=to_regclass('vec_identidad_sesiones_v1.'||nombre);
  IF tabla IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=tabla AND relowner=to_regrole('vec_identidad_sesiones_v1_propietario') AND relrowsecurity=true AND relforcerowsecurity=true)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=tabla AND a.grantee<>c.relowner)
  THEN RAISE EXCEPTION 'ADMIN: tabla o ACL divergente %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $frontera_preimagen$;

DO $preimagen$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_propietario') IS NULL
    OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.cuenta') IS NULL
    OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.estado_cuenta_actual') IS NULL
    OR pg_catalog.to_regclass('vec_autorizacion.control_continuidad_admin') IS NULL
    OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NULL
    OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.politica_certificado_admin_v1') IS NOT NULL
    OR pg_catalog.to_regclass('vec_identidad_sesiones_v1.vinculo_certificado_admin_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'IS9: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

-- Solo el operador puede registrar esta política por el canal privado de
-- bootstrap. No se siembra CA, huella ni cuenta. El entorno y el host exactos
-- son datos de la política aprobada, no permiso derivado de un certificado.
CREATE TABLE vec_identidad_sesiones_v1.politica_certificado_admin_v1 (
 singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
 politica_ref text NOT NULL UNIQUE CHECK (vec_identidad_sesiones_v1.referencia_valida(politica_ref,'pga_') IS TRUE),
 entorno text NOT NULL CHECK (entorno IN ('desarrollo','cidonia','produccion')),
 host_admin text NOT NULL CHECK (host_admin ~ '^[a-z0-9.-]{4,253}$' AND host_admin !~ '\.\.'),
 ca_sha256 text NOT NULL CHECK (ca_sha256 ~ '^[0-9a-f]{64}$' AND ca_sha256<>pg_catalog.repeat('0',64)),
 huella_aprobacion_sha256 text NOT NULL CHECK (huella_aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 maxima_edad_revocacion interval NOT NULL CHECK (maxima_edad_revocacion > interval '0 seconds'),
 vigente_hasta timestamptz(6) NOT NULL CHECK (pg_catalog.isfinite(vigente_hasta)),
 registrada_por text NOT NULL DEFAULT session_user,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 activa boolean NOT NULL DEFAULT true,
 CHECK (pg_catalog.isfinite(registrada_en) AND vigente_hasta>registrada_en)
);
CREATE FUNCTION vec_identidad_sesiones_v1.inmutabilidad_politica_admin_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $funcion$
BEGIN
 IF TG_OP='UPDATE' AND OLD.activa AND NOT NEW.activa
    AND (NEW.singleton,NEW.politica_ref,NEW.entorno,NEW.host_admin,NEW.ca_sha256,
         NEW.huella_aprobacion_sha256,NEW.maxima_edad_revocacion,NEW.vigente_hasta,
         NEW.registrada_por,NEW.registrada_en)
        IS NOT DISTINCT FROM
        (OLD.singleton,OLD.politica_ref,OLD.entorno,OLD.host_admin,OLD.ca_sha256,
         OLD.huella_aprobacion_sha256,OLD.maxima_edad_revocacion,OLD.vigente_hasta,
         OLD.registrada_por,OLD.registrada_en)
 THEN
  IF pg_catalog.current_setting('transaction_isolation')='repeatable read' THEN
   RAISE EXCEPTION 'IS9: aislamiento sin revalidación actual' USING ERRCODE='25000'; END IF;
  -- UPDATE ya retiene el cerrojo de esta fila. Alta y revalidación la leen
  -- FOR SHARE hasta terminar su transacción, sin invertir el orden de locks.
  IF EXISTS (
   SELECT 1 FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a
   JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 v USING(vinculo_ref,version)
   WHERE v.estado='activo' AND v.politica_ref=OLD.politica_ref
  ) THEN RAISE EXCEPTION 'IS9: política con administradores activos' USING ERRCODE='55000'; END IF;
  RETURN NEW;
 END IF;
 RAISE EXCEPTION 'IS9: política inmutable' USING ERRCODE='55000';
END $funcion$;
-- Solo se permite retirar la política, sin sustituir CA o alargar vigencia.
CREATE TRIGGER politica_admin_inmutable BEFORE UPDATE OR DELETE
 ON vec_identidad_sesiones_v1.politica_certificado_admin_v1
 FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.inmutabilidad_politica_admin_v1();
CREATE TRIGGER politica_admin_no_truncar BEFORE TRUNCATE
 ON vec_identidad_sesiones_v1.politica_certificado_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

CREATE TABLE vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 (
 vinculo_ref text NOT NULL CHECK (vec_identidad_sesiones_v1.referencia_valida(vinculo_ref,'vca_') IS TRUE),
 version numeric(20,0) NOT NULL CHECK (version BETWEEN 1 AND 18446744073709551615),
 persona_ref text NOT NULL CHECK (vec_identidad_sesiones_v1.referencia_valida(persona_ref,'per_') IS TRUE),
 cuenta_privilegiada_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
 certificado_sha256 text NOT NULL CHECK (certificado_sha256 ~ '^[0-9a-f]{64}$' AND certificado_sha256<>pg_catalog.repeat('0',64)),
 ca_sha256 text NOT NULL CHECK (ca_sha256 ~ '^[0-9a-f]{64}$' AND ca_sha256<>pg_catalog.repeat('0',64)),
 politica_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.politica_certificado_admin_v1(politica_ref),
 estado text NOT NULL CHECK (estado IN ('activo','revocado')),
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 acto_ref text NOT NULL UNIQUE CHECK (acto_ref ~ '^acto_admin:[0-9a-f]{32}$'),
 registrado_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 PRIMARY KEY(vinculo_ref,version),
 CHECK (pg_catalog.isfinite(vigente_desde) AND pg_catalog.isfinite(vigente_hasta)
        AND vigente_hasta>vigente_desde AND pg_catalog.isfinite(registrado_en))
);
CREATE UNIQUE INDEX certificado_admin_una_persona_v1
 ON vec_identidad_sesiones_v1.vinculo_certificado_admin_v1(certificado_sha256)
 WHERE version=1;
CREATE TABLE vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 (
 vinculo_ref text PRIMARY KEY,
 version numeric(20,0) NOT NULL,
 actualizada_en timestamptz(6) NOT NULL,
 acto_ref text NOT NULL,
 FOREIGN KEY (vinculo_ref,version)
   REFERENCES vec_identidad_sesiones_v1.vinculo_certificado_admin_v1(vinculo_ref,version)
);
CREATE FUNCTION vec_identidad_sesiones_v1.avance_vinculo_certificado_admin_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE nuevo record; previo record;
BEGIN
 SELECT * INTO STRICT nuevo FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_v1
  WHERE vinculo_ref=NEW.vinculo_ref AND version=NEW.version;
 IF TG_OP='INSERT' THEN
  IF NEW.version<>1 OR nuevo.estado<>'activo' THEN
   RAISE EXCEPTION 'IS9: alta inválida' USING ERRCODE='23514'; END IF;
 ELSE
  SELECT * INTO STRICT previo FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_v1
   WHERE vinculo_ref=OLD.vinculo_ref AND version=OLD.version;
  IF NEW.vinculo_ref IS DISTINCT FROM OLD.vinculo_ref OR NEW.version<>OLD.version+1
     OR previo.estado<>'activo' OR nuevo.estado<>'revocado'
     OR (nuevo.persona_ref,nuevo.cuenta_privilegiada_ref,nuevo.certificado_sha256,
         nuevo.ca_sha256,nuevo.politica_ref,nuevo.vigente_desde,nuevo.vigente_hasta)
        IS DISTINCT FROM
        (previo.persona_ref,previo.cuenta_privilegiada_ref,previo.certificado_sha256,
         previo.ca_sha256,previo.politica_ref,previo.vigente_desde,previo.vigente_hasta)
  THEN RAISE EXCEPTION 'IS9: vínculo revocado no revivible' USING ERRCODE='23514'; END IF;
 END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER avance_vinculo_certificado_admin_v1
 BEFORE INSERT OR UPDATE ON vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1
 FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.avance_vinculo_certificado_admin_v1();
CREATE TRIGGER vinculo_certificado_admin_historia_inmutable
 BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.vinculo_certificado_admin_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();
CREATE TRIGGER vinculo_certificado_admin_actual_no_eliminar
 BEFORE DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion();

-- AUT24 consumirá estas funciones dentro de su acto atestado y su CAS. IS9
-- nunca deduce la persona de un certificado ni concede un perfil por sí sola.
CREATE FUNCTION vec_identidad_sesiones_v1.crear_vinculo_certificado_admin_v1(
 p_vinculo_ref text,p_persona_ref text,p_cuenta_ref text,p_certificado_sha256 text,
 p_ca_sha256 text,p_politica_ref text,p_vigente_hasta timestamptz,p_acto_ref text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $funcion$
DECLARE pol record; cuenta record; ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'IS9: requiere SERIALIZABLE de escritura' USING ERRCODE='25000'; END IF;
 IF vec_identidad_sesiones_v1.referencia_valida(p_vinculo_ref,'vca_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
    OR p_certificado_sha256 IS NULL OR p_certificado_sha256 !~ '^[0-9a-f]{64}$'
    OR p_ca_sha256 IS NULL OR p_ca_sha256 !~ '^[0-9a-f]{64}$'
    OR p_certificado_sha256=pg_catalog.repeat('0',64)
    OR p_ca_sha256=pg_catalog.repeat('0',64)
    OR vec_identidad_sesiones_v1.referencia_valida(p_politica_ref,'pga_') IS NOT TRUE
    OR p_vigente_hasta IS NULL OR NOT pg_catalog.isfinite(p_vigente_hasta)
    OR p_acto_ref IS NULL OR p_acto_ref !~ '^acto_admin:[0-9a-f]{32}$' THEN
  RAISE EXCEPTION 'IS9: vínculo inválido' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO pol FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1
  WHERE singleton FOR SHARE;
 ahora:=pg_catalog.clock_timestamp();
 IF NOT FOUND OR NOT pol.activa OR pol.politica_ref<>p_politica_ref
    OR pol.ca_sha256<>p_ca_sha256 OR ahora>=pol.vigente_hasta
    OR p_vigente_hasta>pol.vigente_hasta OR p_vigente_hasta<=ahora THEN
  RAISE EXCEPTION 'IS9: política no vigente' USING ERRCODE='55000'; END IF;
 SELECT c.cuenta_privilegiada,c.cuenta_ordinaria_ref,s.estado,
        so.estado AS estado_ordinaria INTO cuenta
  FROM vec_identidad_sesiones_v1.cuenta c
  JOIN vec_identidad_sesiones_v1.estado_cuenta_actual a USING(cuenta_ref)
  JOIN vec_identidad_sesiones_v1.estado_cuenta s USING(cuenta_ref,revision)
  JOIN vec_identidad_sesiones_v1.estado_cuenta_actual ao
    ON ao.cuenta_ref=c.cuenta_ordinaria_ref
  JOIN vec_identidad_sesiones_v1.estado_cuenta so
    ON so.cuenta_ref=ao.cuenta_ref AND so.revision=ao.revision
  WHERE c.cuenta_ref=p_cuenta_ref FOR SHARE OF a,ao;
 IF NOT FOUND OR NOT cuenta.cuenta_privilegiada OR cuenta.cuenta_ordinaria_ref IS NULL
    OR cuenta.estado<>'activa' OR cuenta.estado_ordinaria<>'activa' THEN
  RAISE EXCEPTION 'IS9: cuenta privilegiada no vigente' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_identidad_sesiones_v1.vinculo_certificado_admin_v1
  (vinculo_ref,version,persona_ref,cuenta_privilegiada_ref,certificado_sha256,
   ca_sha256,politica_ref,estado,vigente_desde,vigente_hasta,acto_ref)
 VALUES(p_vinculo_ref,1,p_persona_ref,p_cuenta_ref,p_certificado_sha256,
        p_ca_sha256,p_politica_ref,'activo',ahora,p_vigente_hasta,p_acto_ref);
 INSERT INTO vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1
  (vinculo_ref,version,actualizada_en,acto_ref)
 VALUES(p_vinculo_ref,1,ahora,p_acto_ref);
 RETURN true;
END $funcion$;

CREATE FUNCTION vec_identidad_sesiones_v1.revocar_vinculo_certificado_admin_v1(
 p_vinculo_ref text,p_version_esperada numeric,p_acto_ref text
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on AS $funcion$
DECLARE vinculo record; ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR pg_catalog.current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'IS9: requiere SERIALIZABLE de escritura' USING ERRCODE='25000'; END IF;
 IF vec_identidad_sesiones_v1.referencia_valida(p_vinculo_ref,'vca_') IS NOT TRUE
    OR p_version_esperada IS NULL OR p_version_esperada<1
    OR p_version_esperada>=18446744073709551615::numeric
    OR p_acto_ref IS NULL OR p_acto_ref !~ '^acto_admin:[0-9a-f]{32}$' THEN
  RAISE EXCEPTION 'IS9: revocación inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT v.* INTO vinculo
  FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a
  JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 v USING(vinculo_ref,version)
  WHERE a.vinculo_ref=p_vinculo_ref FOR UPDATE OF a;
 IF NOT FOUND OR vinculo.version<>p_version_esperada OR vinculo.estado<>'activo' THEN
  RAISE EXCEPTION 'IS9: CAS divergente o vínculo revocado' USING ERRCODE='40001'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 INSERT INTO vec_identidad_sesiones_v1.vinculo_certificado_admin_v1
  (vinculo_ref,version,persona_ref,cuenta_privilegiada_ref,certificado_sha256,
   ca_sha256,politica_ref,estado,vigente_desde,vigente_hasta,acto_ref)
 VALUES(vinculo.vinculo_ref,vinculo.version+1,vinculo.persona_ref,vinculo.cuenta_privilegiada_ref,
        vinculo.certificado_sha256,vinculo.ca_sha256,vinculo.politica_ref,'revocado',
        vinculo.vigente_desde,vinculo.vigente_hasta,p_acto_ref);
 UPDATE vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1
  SET version=vinculo.version+1,actualizada_en=ahora,acto_ref=p_acto_ref
  WHERE vinculo_ref=p_vinculo_ref AND version=p_version_esperada;
 IF NOT FOUND THEN RAISE EXCEPTION 'IS9: CAS perdido' USING ERRCODE='40001'; END IF;
 RETURN true;
END $funcion$;

-- Fachada interna. Su llamador futuro debe ser exclusivamente el adaptador
-- ADMIN que ha verificado VerifiedChains, revocación y el canal TLS directo.
-- Los booleanos no son atestaciones autónomas; por eso no hay GRANT runtime.
CREATE FUNCTION vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(
 p_entorno text,p_host text,p_persona_ref text,p_cuenta_ref text,
 p_certificado_sha256 text,p_ca_sha256 text,p_tls_verificado boolean,
 p_revocacion_verificada_en timestamptz,p_kerberos_verificado boolean,
 p_red_corporativa_verificada boolean
) RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on AS $funcion$
DECLARE pol record; vinculo record; cuenta record; ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_read_only')<>'off'
    OR pg_catalog.current_setting('transaction_isolation')='repeatable read'
    OR p_tls_verificado IS DISTINCT FROM true
    OR p_entorno IS NULL OR p_entorno NOT IN ('desarrollo','cidonia','produccion')
    OR p_host IS NULL
    OR vec_identidad_sesiones_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
    OR vec_identidad_sesiones_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
    OR p_certificado_sha256 IS NULL OR p_certificado_sha256 !~ '^[0-9a-f]{64}$'
    OR p_ca_sha256 IS NULL OR p_ca_sha256 !~ '^[0-9a-f]{64}$'
    OR p_revocacion_verificada_en IS NULL
    OR NOT pg_catalog.isfinite(p_revocacion_verificada_en) THEN RETURN false; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO pol FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1
  WHERE singleton FOR SHARE;
 ahora:=pg_catalog.clock_timestamp();
 IF NOT FOUND OR NOT pol.activa OR pol.entorno IS DISTINCT FROM p_entorno
    OR pol.host_admin IS DISTINCT FROM p_host
    OR pol.ca_sha256 IS DISTINCT FROM p_ca_sha256 OR ahora>=pol.vigente_hasta
    OR p_revocacion_verificada_en<pol.registrada_en
    OR p_revocacion_verificada_en>ahora
    OR ahora-p_revocacion_verificada_en>pol.maxima_edad_revocacion
    OR (pol.entorno='produccion' AND
        (p_kerberos_verificado IS DISTINCT FROM true OR
         p_red_corporativa_verificada IS DISTINCT FROM true)) THEN RETURN false; END IF;
 SELECT v.* INTO vinculo
  FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a
  JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 v USING(vinculo_ref,version)
  WHERE v.persona_ref=p_persona_ref AND v.cuenta_privilegiada_ref=p_cuenta_ref
    AND v.certificado_sha256=p_certificado_sha256 AND v.ca_sha256=p_ca_sha256
    AND v.politica_ref=pol.politica_ref FOR SHARE OF a;
 IF NOT FOUND OR vinculo.estado<>'activo' OR ahora<vinculo.vigente_desde
    OR ahora>=vinculo.vigente_hasta THEN RETURN false; END IF;
 SELECT c.cuenta_privilegiada,c.cuenta_ordinaria_ref,s.estado,
        so.estado AS estado_ordinaria INTO cuenta
  FROM vec_identidad_sesiones_v1.cuenta c
  JOIN vec_identidad_sesiones_v1.estado_cuenta_actual a USING(cuenta_ref)
  JOIN vec_identidad_sesiones_v1.estado_cuenta s USING(cuenta_ref,revision)
  JOIN vec_identidad_sesiones_v1.estado_cuenta_actual ao
    ON ao.cuenta_ref=c.cuenta_ordinaria_ref
  JOIN vec_identidad_sesiones_v1.estado_cuenta so
    ON so.cuenta_ref=ao.cuenta_ref AND so.revision=ao.revision
  WHERE c.cuenta_ref=p_cuenta_ref FOR SHARE OF a,ao;
 RETURN FOUND AND cuenta.cuenta_privilegiada AND cuenta.cuenta_ordinaria_ref IS NOT NULL
    AND cuenta.estado='activa' AND cuenta.estado_ordinaria='activa';
EXCEPTION WHEN data_exception OR read_only_sql_transaction THEN RETURN false;
END $funcion$;

-- La autoridad anterior de identidad puede inactivar cuentas sin conocer AUT.
-- Hasta AUT24 se cierra ese camino para toda cuenta ligada a un certificado
-- ADMIN: el acto futuro habrá de consumir la misma barrera y contar personas.
CREATE FUNCTION vec_identidad_sesiones_v1.bloquear_inactivacion_admin_hasta_aut24_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $funcion$
BEGIN
 IF NEW.estado='inactiva' THEN
  IF pg_catalog.current_setting('transaction_isolation')='repeatable read' THEN
   RAISE EXCEPTION 'IS9: aislamiento sin revalidación actual' USING ERRCODE='25000'; END IF;
  -- El cambio histórico ya retiene FOR UPDATE del puntero de cuenta;
  -- crear/revalidar retienen FOR SHARE del mismo puntero. Otro advisory
  -- aquí invertiría el orden y causaría un interbloqueo.
  IF EXISTS (
   SELECT 1 FROM vec_identidad_sesiones_v1.vinculo_certificado_admin_actual_v1 a
   JOIN vec_identidad_sesiones_v1.vinculo_certificado_admin_v1 v USING(vinculo_ref,version)
   JOIN vec_identidad_sesiones_v1.cuenta c ON c.cuenta_ref=v.cuenta_privilegiada_ref
   WHERE (v.cuenta_privilegiada_ref=NEW.cuenta_ref
          OR c.cuenta_ordinaria_ref=NEW.cuenta_ref) AND v.estado='activo'
  ) THEN RAISE EXCEPTION 'IS9: cuenta ADMIN requiere AUT24' USING ERRCODE='55000'; END IF;
 END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER estado_cuenta_admin_continuidad_v1
 BEFORE INSERT ON vec_identidad_sesiones_v1.estado_cuenta
 FOR EACH ROW EXECUTE FUNCTION vec_identidad_sesiones_v1.bloquear_inactivacion_admin_hasta_aut24_v1();

DO $acl$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['politica_certificado_admin_v1','vinculo_certificado_admin_v1',
                              'vinculo_certificado_admin_actual_v1'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_sesiones_v1.%I ENABLE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('ALTER TABLE vec_identidad_sesiones_v1.%I FORCE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.%I FOR ALL TO vec_identidad_sesiones_v1_propietario USING (current_user=%L) WITH CHECK (current_user=%L)',nombre,'vec_identidad_sesiones_v1_propietario','vec_identidad_sesiones_v1_propietario');
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_identidad_sesiones_v1.%I FROM PUBLIC',nombre);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_identidad_sesiones_v1.%I FROM PUBLIC',nombre);
 END LOOP;
END $acl$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.inmutabilidad_politica_admin_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.avance_vinculo_certificado_admin_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.crear_vinculo_certificado_admin_v1(text,text,text,text,text,text,timestamptz,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.revocar_vinculo_certificado_admin_v1(text,numeric,text) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.revalidar_certificado_admin_v1(text,text,text,text,text,text,boolean,timestamptz,boolean,boolean) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.bloquear_inactivacion_admin_hasta_aut24_v1() FROM PUBLIC;
COMMIT;
