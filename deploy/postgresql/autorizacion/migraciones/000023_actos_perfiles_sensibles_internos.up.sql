\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000023',0));

-- Preimagen causal: propietario, superficie ACL y protección de las tablas
-- que consume esta migración. No depende del LOGIN o entorno del ensayo.
DO $frontera_preimagen$
DECLARE nombre text;tabla regclass;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_propietario' AND NOT(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_autorizacion' AND nspowner=to_regrole('vec_autorizacion_propietario'))
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='vec_autorizacion' AND (a.grantee=0 OR (a.privilege_type='CREATE' AND a.grantee<>n.nspowner)))
 THEN RAISE EXCEPTION 'ADMIN: namespace o propietario divergente' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['version_rol','asignacion_perfil','asignacion_perfil_actual','control_vigencia_version_rol','control_vigencia_version_rol_actual'] LOOP
  tabla:=to_regclass('vec_autorizacion.'||nombre);
  IF tabla IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=tabla AND relowner=to_regrole('vec_autorizacion_propietario') AND relrowsecurity=true AND relforcerowsecurity=true)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=tabla AND a.grantee<>c.relowner)
  THEN RAISE EXCEPTION 'ADMIN: tabla o ACL divergente %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $frontera_preimagen$;
LOCK TABLE vec_autorizacion.asignacion_perfil_actual,
           vec_autorizacion.control_vigencia_version_rol_actual IN SHARE ROW EXCLUSIVE MODE;
DO $preimagen$
DECLARE v record;
BEGIN
  SELECT * INTO v FROM vec_autorizacion.version_rol
  WHERE version_rol_ref='rol:administracion_perfiles:v1';
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
     OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
     OR pg_catalog.to_regrole('vec_contexto_actor_v1_propietario') IS NULL
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
       WHERE c.oid='vec_autorizacion.asignacion_perfil_actual'::regclass
         AND c.relowner='vec_autorizacion_propietario'::regrole
         AND c.relrowsecurity AND c.relforcerowsecurity)
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
       WHERE p.oid=pg_catalog.to_regprocedure('vec_autorizacion.impedir_revivir_asignacion_v1()')
         AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef
         AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[])
     OR v.rol_id IS DISTINCT FROM 'administracion_perfiles'
     OR v.version IS DISTINCT FROM 1
     OR v.documento->>'estado' IS DISTINCT FROM 'publicada'
     OR pg_catalog.jsonb_array_length(v.documento->'concesiones') IS DISTINCT FROM 1
     OR v.documento->'concesiones'->0->>'accion' IS DISTINCT FROM 'administracion.perfiles.consultar'
     OR EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil WHERE version_rol_ref=v.version_rol_ref)
     OR pg_catalog.to_regclass('vec_autorizacion.control_continuidad_admin') IS NOT NULL
     OR pg_catalog.to_regclass('vec_autorizacion.propuesta_perfil_sensible') IS NOT NULL
     OR pg_catalog.to_regprocedure('vec_autorizacion.avanzar_continuidad_admin_interna_v1(bigint)') IS NOT NULL
     OR EXISTS (SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id='administracion_perfiles' AND version<>1)
  THEN RAISE EXCEPTION 'AUT23: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE TABLE vec_autorizacion.clase_perfil_sensible (
  clase text PRIMARY KEY CHECK (clase IN ('administrador','intervencion')),
  doble_control boolean NOT NULL CHECK (doble_control)
);
CREATE TABLE vec_autorizacion.operacion_perfil_sensible (
  clase text NOT NULL REFERENCES vec_autorizacion.clase_perfil_sensible(clase),
  operacion text NOT NULL CHECK (operacion IN ('otorgar','revocar')),
  PRIMARY KEY(clase,operacion)
);
CREATE TABLE vec_autorizacion.rol_sensible_exacto (
  version_rol_ref text PRIMARY KEY REFERENCES vec_autorizacion.version_rol(version_rol_ref),
  clase text NOT NULL REFERENCES vec_autorizacion.clase_perfil_sensible(clase),
  huella_sha256 text NOT NULL CHECK (huella_sha256 ~ '^[0-9a-f]{64}$'),
  publicado_en timestamptz NOT NULL DEFAULT pg_catalog.clock_timestamp(),
  UNIQUE(version_rol_ref,clase)
);
-- Bootstrap: dos personas nominativas en un único acto. Tras él se admite
-- 2→1 con doble control; 1→0 se deniega. AUT24 comprobará la población
-- efectiva con CA20/identidad y bloqueará otros actos sensibles si queda una.
CREATE TABLE vec_autorizacion.control_continuidad_admin (
  control_id boolean PRIMARY KEY DEFAULT true CHECK (control_id),
  revision bigint NOT NULL CHECK (revision>0),
  bootstrap_minimo_personas integer NOT NULL CHECK (bootstrap_minimo_personas=2),
  minimo_personas integer NOT NULL CHECK (minimo_personas=1),
  bootstrap_estado text NOT NULL CHECK (bootstrap_estado IN ('pendiente','consumido')),
  bootstrap_acto_ref text CHECK (bootstrap_acto_ref IS NULL OR bootstrap_acto_ref ~ '^acto_admin:[0-9a-f]{32}$'),
  actualizado_en timestamptz NOT NULL,
  CHECK ((bootstrap_estado='pendiente' AND bootstrap_acto_ref IS NULL) OR
         (bootstrap_estado='consumido' AND bootstrap_acto_ref IS NOT NULL))
);
INSERT INTO vec_autorizacion.clase_perfil_sensible VALUES
 ('administrador',true),('intervencion',true);
INSERT INTO vec_autorizacion.operacion_perfil_sensible VALUES
 ('administrador','otorgar'),('administrador','revocar'),
 ('intervencion','otorgar'),('intervencion','revocar');
INSERT INTO vec_autorizacion.control_continuidad_admin
 (control_id,revision,bootstrap_minimo_personas,minimo_personas,bootstrap_estado,actualizado_en)
VALUES(true,1,2,1,'pendiente',pg_catalog.clock_timestamp());

-- Ningún rol sensible tiene permiso por esta migración: solo se publican las
-- definiciones. Intervención aún carece de mapeo exacto y permanece cerrada.
DO $rol_v2$
DECLARE instante text; documento text; control_documento text; rol jsonb; control jsonb;
        rol_ref constant text := 'rol:administracion_perfiles:v2';
BEGIN
 instante:=pg_catalog.to_char(pg_catalog.date_trunc('second',pg_catalog.clock_timestamp()) AT TIME ZONE 'UTC',
                               'YYYY-MM-DD"T"HH24:MI:SS"Z"');
 documento:=pg_catalog.format('{"rol_id":"administracion_perfiles","version":2,"nombre":"administracion.perfiles.rol","estado":"publicada","concesiones":[{"accion":"administracion.perfiles.consultar","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto","campos_permitidos":["perfil_ref","vinculo_ref","version"]},{"accion":"administracion.perfiles.proponer","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto"},{"accion":"administracion.perfiles.aprobar","modulo_id":"administracion","tipo_recurso":"propuesta_perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto"},{"accion":"administracion.perfiles.rechazar","modulo_id":"administracion","tipo_recurso":"propuesta_perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto"},{"accion":"administracion.perfiles.otorgar","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto"},{"accion":"administracion.perfiles.revocar","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto"},{"accion":"administracion.perfiles.historial.consultar","modulo_id":"administracion","tipo_recurso":"historial_perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto"},{"accion":"administracion.perfiles.recibo.consultar","modulo_id":"administracion","tipo_recurso":"recibo_perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto"}],"publicada_por":"migracion:autorizacion:000023","publicada_en":"%s","retirada_en":"0001-01-01T00:00:00Z"}',instante);
 rol:=documento::jsonb;
 IF vec_autorizacion.concesiones_positivas_validas(rol) IS NOT TRUE THEN
   RAISE EXCEPTION 'AUT23: rol v2 inválido' USING ERRCODE='23514'; END IF;
 INSERT INTO vec_autorizacion.version_rol
 (version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
 VALUES(rol_ref,'administracion_perfiles',2,
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento,'UTF8')),'hex'),
   instante::timestamptz,rol);
 control_documento:=pg_catalog.format('{"version_rol_ref":"%s","revision":1,"estado":"habilitada","actualizado_por":"migracion:autorizacion:000023","actualizado_en":"%s"}',rol_ref,instante);
 control:=control_documento::jsonb;
 INSERT INTO vec_autorizacion.control_vigencia_version_rol
 (version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
 VALUES(rol_ref,1,'habilitada',
   pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(control_documento,'UTF8')),'hex'),
   instante::timestamptz,control);
 INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
 (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
 VALUES(rol_ref,1,instante::timestamptz,'migracion:autorizacion:000023','migracion:autorizacion:000023');
 INSERT INTO vec_autorizacion.rol_sensible_exacto(version_rol_ref,clase,huella_sha256)
 SELECT rol_ref,'administrador',huella_sha256 FROM vec_autorizacion.version_rol
 WHERE version_rol_ref=rol_ref;
END $rol_v2$;

CREATE TABLE vec_autorizacion.propuesta_perfil_sensible (
  propuesta_ref text PRIMARY KEY CHECK (propuesta_ref ~ '^propuesta_admin:[0-9a-f]{32}$'),
  clase text NOT NULL,
  operacion text NOT NULL,
  version_rol_ref text NOT NULL,
  objetivo_persona_ref text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(objetivo_persona_ref,512) IS TRUE),
  objetivo_cuenta_ref text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(objetivo_cuenta_ref,512) IS TRUE),
  objetivo_perfil_ref text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(objetivo_perfil_ref,512) IS TRUE),
  proponente_persona_ref text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(proponente_persona_ref,512) IS TRUE),
  proponente_cuenta_ref text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(proponente_cuenta_ref,512) IS TRUE),
  proponente_perfil_ref text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(proponente_perfil_ref,512) IS TRUE),
  motivo_codigo text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(motivo_codigo,128) IS TRUE),
  revision_continuidad_esperada bigint NOT NULL CHECK (revision_continuidad_esperada>0),
  preimagen_huella_sha256 text NOT NULL CHECK (preimagen_huella_sha256 ~ '^[0-9a-f]{64}$'),
  documento_canonico bytea NOT NULL CHECK (pg_catalog.octet_length(documento_canonico) BETWEEN 1 AND 65536),
  huella_sha256 text NOT NULL CHECK (huella_sha256=pg_catalog.encode(pg_catalog.sha256(documento_canonico),'hex')),
  creada_en timestamptz NOT NULL CHECK (pg_catalog.isfinite(creada_en)),
  caduca_en timestamptz NOT NULL CHECK (pg_catalog.isfinite(caduca_en) AND caduca_en>creada_en),
  FOREIGN KEY(clase,operacion) REFERENCES vec_autorizacion.operacion_perfil_sensible(clase,operacion),
  FOREIGN KEY(version_rol_ref,clase) REFERENCES vec_autorizacion.rol_sensible_exacto(version_rol_ref,clase)
);
CREATE TABLE vec_autorizacion.cierre_propuesta_perfil_sensible (
  propuesta_ref text PRIMARY KEY REFERENCES vec_autorizacion.propuesta_perfil_sensible(propuesta_ref),
  cierre_ref text NOT NULL UNIQUE CHECK (cierre_ref ~ '^cierre_admin:[0-9a-f]{32}$'),
  resultado text NOT NULL CHECK (resultado IN ('aprobada','rechazada','caducada')),
  aprobador_persona_ref text,
  aprobador_cuenta_ref text,
  aprobador_perfil_ref text,
  motivo_codigo text NOT NULL CHECK (vec_autorizacion.texto_positivo_valido(motivo_codigo,128) IS TRUE),
  revision_continuidad_observada bigint NOT NULL CHECK (revision_continuidad_observada>0),
  documento_canonico bytea NOT NULL CHECK (pg_catalog.octet_length(documento_canonico) BETWEEN 1 AND 65536),
  huella_sha256 text NOT NULL CHECK (huella_sha256=pg_catalog.encode(pg_catalog.sha256(documento_canonico),'hex')),
  cerrada_en timestamptz NOT NULL CHECK (pg_catalog.isfinite(cerrada_en)),
  CHECK ((resultado='caducada' AND aprobador_persona_ref IS NULL
          AND aprobador_cuenta_ref IS NULL AND aprobador_perfil_ref IS NULL) OR
         (resultado<>'caducada' AND
          vec_autorizacion.texto_positivo_valido(aprobador_persona_ref,512) IS TRUE AND
          vec_autorizacion.texto_positivo_valido(aprobador_cuenta_ref,512) IS TRUE AND
          vec_autorizacion.texto_positivo_valido(aprobador_perfil_ref,512) IS TRUE))
);
CREATE TABLE vec_autorizacion.acto_perfil_sensible (
  acto_ref text PRIMARY KEY CHECK (acto_ref ~ '^acto_admin:[0-9a-f]{32}$'),
  propuesta_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.propuesta_perfil_sensible(propuesta_ref),
  cierre_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.cierre_propuesta_perfil_sensible(cierre_ref),
  revision_continuidad_previa bigint NOT NULL CHECK (revision_continuidad_previa>0),
  revision_continuidad_posterior bigint NOT NULL CHECK (revision_continuidad_posterior>revision_continuidad_previa),
  preimagen_huella_sha256 text NOT NULL CHECK (preimagen_huella_sha256 ~ '^[0-9a-f]{64}$'),
  postimagen_huella_sha256 text NOT NULL CHECK (postimagen_huella_sha256 ~ '^[0-9a-f]{64}$'),
  documento_canonico bytea NOT NULL CHECK (pg_catalog.octet_length(documento_canonico) BETWEEN 1 AND 65536),
  huella_sha256 text NOT NULL CHECK (huella_sha256=pg_catalog.encode(pg_catalog.sha256(documento_canonico),'hex')),
  ejecutado_en timestamptz NOT NULL CHECK (pg_catalog.isfinite(ejecutado_en))
);
CREATE TABLE vec_autorizacion.recibo_perfil_sensible (
  recibo_ref text PRIMARY KEY CHECK (recibo_ref ~ '^recibo_admin:[0-9a-f]{32}$'),
  acto_ref text NOT NULL UNIQUE REFERENCES vec_autorizacion.acto_perfil_sensible(acto_ref),
  documento_canonico bytea NOT NULL CHECK (pg_catalog.octet_length(documento_canonico) BETWEEN 1 AND 65536),
  huella_sha256 text NOT NULL CHECK (huella_sha256=pg_catalog.encode(pg_catalog.sha256(documento_canonico),'hex')),
  emitido_en timestamptz NOT NULL CHECK (pg_catalog.isfinite(emitido_en))
);

DO $historia$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['clase_perfil_sensible','operacion_perfil_sensible','rol_sensible_exacto',
   'propuesta_perfil_sensible','cierre_propuesta_perfil_sensible','acto_perfil_sensible','recibo_perfil_sensible'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exactamente ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING (current_user=''vec_autorizacion_propietario'') WITH CHECK (current_user=''vec_autorizacion_propietario'')',tabla);
  EXECUTE pg_catalog.format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',tabla);
  EXECUTE pg_catalog.format('CREATE TRIGGER historia_no_truncable BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',tabla);
 END LOOP;
END $historia$;
ALTER TABLE vec_autorizacion.control_continuidad_admin ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion.control_continuidad_admin FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exactamente ON vec_autorizacion.control_continuidad_admin FOR ALL
 TO vec_autorizacion_propietario
 USING (current_user='vec_autorizacion_propietario')
 WITH CHECK (current_user='vec_autorizacion_propietario');
CREATE TRIGGER control_continuidad_no_eliminar BEFORE DELETE OR TRUNCATE
 ON vec_autorizacion.control_continuidad_admin FOR EACH STATEMENT
 EXECUTE FUNCTION vec_autorizacion.rechazar_eliminacion_versionada();

CREATE FUNCTION vec_autorizacion.validar_avance_continuidad_admin_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $funcion$
BEGIN
 IF NEW.control_id IS DISTINCT FROM OLD.control_id
    OR NEW.revision IS DISTINCT FROM OLD.revision+1
    OR NEW.bootstrap_minimo_personas IS DISTINCT FROM OLD.bootstrap_minimo_personas
    OR NEW.minimo_personas IS DISTINCT FROM OLD.minimo_personas
    OR (OLD.bootstrap_estado='consumido' AND
       (NEW.bootstrap_estado IS DISTINCT FROM OLD.bootstrap_estado OR
        NEW.bootstrap_acto_ref IS DISTINCT FROM OLD.bootstrap_acto_ref))
    OR (OLD.bootstrap_estado='pendiente' AND NEW.bootstrap_estado='pendiente'
       AND NEW.bootstrap_acto_ref IS NOT NULL)
 THEN RAISE EXCEPTION 'continuidad admin: avance inválido' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER control_continuidad_avance BEFORE UPDATE ON vec_autorizacion.control_continuidad_admin
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_avance_continuidad_admin_v1();
REVOKE ALL ON FUNCTION vec_autorizacion.validar_avance_continuidad_admin_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.avanzar_continuidad_admin_interna_v1(p_revision_esperada bigint)
RETURNS bigint LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE nueva_revision bigint;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 IF p_revision_esperada IS NULL OR p_revision_esperada<1 THEN
  RAISE EXCEPTION 'continuidad admin: revisión inválida' USING ERRCODE='22023'; END IF;
 UPDATE vec_autorizacion.control_continuidad_admin
 SET revision=revision+1,actualizado_en=pg_catalog.clock_timestamp()
 WHERE control_id=true AND revision=p_revision_esperada
 RETURNING revision INTO nueva_revision;
 IF NOT FOUND THEN RAISE EXCEPTION 'continuidad admin: CAS divergente' USING ERRCODE='40001'; END IF;
 RETURN nueva_revision;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion.avanzar_continuidad_admin_interna_v1(bigint) FROM PUBLIC;
-- AUT24 concederá EXECUTE exacto al propietario CA cuando el orquestador
-- consuma también identidad y la decisión V3; aquí nadie más lo ejecuta.

-- AUT24 sustituirá esta guarda por su efecto atestado. Ni el publicador genérico
-- ni un perfil del portal pueden anticipar el bootstrap o saltarse el doble control.
CREATE FUNCTION vec_autorizacion.bloquear_asignacion_sensible_hasta_aut24_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE rol text; referencia text;
BEGIN
 SELECT r.rol_id,r.version_rol_ref INTO STRICT rol,referencia
 FROM vec_autorizacion.asignacion_perfil a
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 WHERE a.asignacion_ref=NEW.asignacion_ref;
 IF rol='administracion_perfiles' OR EXISTS (
   SELECT 1 FROM vec_autorizacion.rol_sensible_exacto s
   WHERE s.version_rol_ref=referencia) THEN
  RAISE EXCEPTION 'asignación sensible cerrada hasta AUT24' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER asignacion_sensible_cerrada_hasta_aut24 BEFORE INSERT OR UPDATE
ON vec_autorizacion.asignacion_perfil_actual FOR EACH ROW
EXECUTE FUNCTION vec_autorizacion.bloquear_asignacion_sensible_hasta_aut24_v1();
REVOKE ALL ON FUNCTION vec_autorizacion.bloquear_asignacion_sensible_hasta_aut24_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.validar_propuesta_perfil_sensible_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE vigente bigint; rol record;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT revision INTO STRICT vigente FROM vec_autorizacion.control_continuidad_admin
 WHERE control_id=true FOR UPDATE;
 SELECT r.rol_id,r.huella_sha256,r.documento->>'estado' AS estado,c.estado AS control_estado
 INTO STRICT rol
 FROM vec_autorizacion.rol_sensible_exacto s
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=s.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE s.version_rol_ref=NEW.version_rol_ref AND s.clase=NEW.clase;
 IF NEW.revision_continuidad_esperada IS DISTINCT FROM vigente
    OR rol.estado IS DISTINCT FROM 'publicada' OR rol.control_estado IS DISTINCT FROM 'habilitada'
    OR NEW.caduca_en<=pg_catalog.clock_timestamp()
 THEN RAISE EXCEPTION 'propuesta sensible: preimagen no vigente' USING ERRCODE='40001'; END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER propuesta_preimagen_v1 BEFORE INSERT ON vec_autorizacion.propuesta_perfil_sensible
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_propuesta_perfil_sensible_v1();
REVOKE ALL ON FUNCTION vec_autorizacion.validar_propuesta_perfil_sensible_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.validar_cierre_propuesta_sensible_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE propuesta record; vigente bigint;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT revision INTO STRICT vigente FROM vec_autorizacion.control_continuidad_admin
 WHERE control_id=true FOR UPDATE;
 SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible
 WHERE propuesta_ref=NEW.propuesta_ref FOR SHARE;
 IF NEW.cerrada_en<propuesta.creada_en
    OR NEW.cerrada_en>pg_catalog.clock_timestamp()
    OR NEW.revision_continuidad_observada IS DISTINCT FROM vigente
 THEN RAISE EXCEPTION 'cierre sensible: revisión o instante inválidos' USING ERRCODE='40001'; END IF;
 IF NEW.resultado='aprobada' AND
   (vigente IS DISTINCT FROM propuesta.revision_continuidad_esperada
    OR NEW.cerrada_en>=propuesta.caduca_en
    OR NEW.aprobador_persona_ref IS NOT DISTINCT FROM propuesta.proponente_persona_ref
    OR NEW.aprobador_persona_ref IS NOT DISTINCT FROM propuesta.objetivo_persona_ref)
 THEN RAISE EXCEPTION 'aprobación sensible no independiente o caducada' USING ERRCODE='42501'; END IF;
 IF NEW.resultado='rechazada' AND
   NEW.aprobador_persona_ref IS NOT DISTINCT FROM propuesta.proponente_persona_ref
 THEN RAISE EXCEPTION 'rechazo sensible no independiente' USING ERRCODE='42501'; END IF;
 IF NEW.resultado='caducada' AND NEW.cerrada_en<propuesta.caduca_en THEN
  RAISE EXCEPTION 'propuesta aún vigente' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER cierre_independiente_v1 BEFORE INSERT ON vec_autorizacion.cierre_propuesta_perfil_sensible
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_cierre_propuesta_sensible_v1();
REVOKE ALL ON FUNCTION vec_autorizacion.validar_cierre_propuesta_sensible_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.validar_acto_perfil_sensible_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE propuesta record; cierre record; vigente bigint;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT revision INTO STRICT vigente FROM vec_autorizacion.control_continuidad_admin
 WHERE control_id=true FOR UPDATE;
 SELECT * INTO STRICT propuesta FROM vec_autorizacion.propuesta_perfil_sensible
 WHERE propuesta_ref=NEW.propuesta_ref FOR SHARE;
 SELECT * INTO STRICT cierre FROM vec_autorizacion.cierre_propuesta_perfil_sensible
 WHERE cierre_ref=NEW.cierre_ref FOR SHARE;
 IF cierre.propuesta_ref IS DISTINCT FROM propuesta.propuesta_ref
    OR cierre.resultado IS DISTINCT FROM 'aprobada'
    OR NEW.preimagen_huella_sha256 IS DISTINCT FROM propuesta.preimagen_huella_sha256
    OR NEW.revision_continuidad_previa IS DISTINCT FROM propuesta.revision_continuidad_esperada
    OR NEW.revision_continuidad_posterior IS DISTINCT FROM vigente
    OR NEW.ejecutado_en<cierre.cerrada_en
    OR NEW.ejecutado_en>pg_catalog.clock_timestamp()
 THEN RAISE EXCEPTION 'acto sensible sin cierre o efecto coherente' USING ERRCODE='40001'; END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER acto_coherente_v1 BEFORE INSERT ON vec_autorizacion.acto_perfil_sensible
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_acto_perfil_sensible_v1();
REVOKE ALL ON FUNCTION vec_autorizacion.validar_acto_perfil_sensible_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.validar_recibo_perfil_sensible_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $funcion$
DECLARE instante timestamptz;
BEGIN
 SELECT ejecutado_en INTO STRICT instante FROM vec_autorizacion.acto_perfil_sensible
 WHERE acto_ref=NEW.acto_ref FOR SHARE;
 IF NEW.emitido_en<instante OR NEW.emitido_en>pg_catalog.clock_timestamp() THEN
  RAISE EXCEPTION 'recibo sensible sin efecto previo' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $funcion$;
CREATE TRIGGER recibo_con_acto_v1 BEFORE INSERT ON vec_autorizacion.recibo_perfil_sensible
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.validar_recibo_perfil_sensible_v1();
REVOKE ALL ON FUNCTION vec_autorizacion.validar_recibo_perfil_sensible_v1() FROM PUBLIC;

DO $acl$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['clase_perfil_sensible','operacion_perfil_sensible','rol_sensible_exacto',
   'control_continuidad_admin','propuesta_perfil_sensible','cierre_propuesta_perfil_sensible',
   'acto_perfil_sensible','recibo_perfil_sensible'] LOOP
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC,vec_autorizacion_fuente',tabla);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC,vec_autorizacion_fuente',tabla);
 END LOOP;
END $acl$;
COMMIT;
