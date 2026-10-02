-- BORRADOR: CA22 real y fachada nominal AUT24 son dependencias previas.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000023',0));
DO $preimagen$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[])') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_actor_v2_propietaria_admin_v1(text,text,text,text,text,text,timestamptz,text[])') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.listar_asignaciones_admin_perfiles_propias_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.listar_asignaciones_admin_perfiles_propias_reconciliacion_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.consultar_metadata_perfil_admin_v1(text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.revalidar_audiencia_selector_admin_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(text,text)') IS NULL
 OR pg_catalog.to_regclass('vec_contexto_actor_v1.procedencia_acto_admin_v1') IS NOT NULL
 OR pg_catalog.to_regrole('vec_identidad_sesiones_v1_admin_perfiles') IS NOT NULL
 THEN RAISE EXCEPTION 'CA23: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
CREATE ROLE vec_identidad_sesiones_v1_admin_perfiles NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_identidad_sesiones_v1_admin_perfiles',current_database());
END $conexion$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;

-- Solo consume la asignación central nominal de AUT24. Nunca crea perfiles,
-- asignaciones ni vínculos, ni toma un perfil solicitado como autoridad.
-- Elección de un único perfil, propia de la cuenta autenticada. No concede roles.
CREATE TABLE vec_contexto_actor_v1.seleccion_perfil_admin_v1(
 cuenta_ref text NOT NULL, revision numeric(20,0) NOT NULL CHECK(revision BETWEEN 1 AND 18446744073709551615),
 persona_ref text NOT NULL,perfil_ref text NOT NULL,vinculo_ref text NOT NULL,
 fuente text NOT NULL CHECK(fuente='vec_autorizacion.admin'),
 causa text NOT NULL CHECK(causa IN('eleccion_explicita','autoseleccion_unica')),
 autenticacion_verificada_en timestamptz(6) NOT NULL,
 seleccionada_en timestamptz(6) NOT NULL,
 auditoria_ref text NOT NULL UNIQUE,
 documento jsonb NOT NULL,huella_sha256 text NOT NULL,
 PRIMARY KEY(cuenta_ref,revision),
 CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_') IS TRUE AND vec_contexto_actor_v1.referencia_valida(persona_ref,'per_') IS TRUE AND vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_') IS TRUE AND vec_contexto_actor_v1.referencia_valida(vinculo_ref,'vca_') IS TRUE),
 CHECK(pg_catalog.isfinite(autenticacion_verificada_en) AND pg_catalog.isfinite(seleccionada_en) AND seleccionada_en>=autenticacion_verificada_en),
 CHECK(huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento::text,'UTF8')),'hex')),
 CHECK((documento->>'cuenta_ref',documento->>'actor_persona_ref',documento->>'perfil_ref',documento->>'vinculo_ref') IS NOT DISTINCT FROM (cuenta_ref,persona_ref,perfil_ref,vinculo_ref) AND (documento->>'revision')::numeric IS NOT DISTINCT FROM revision)
);
REVOKE ALL ON TABLE vec_contexto_actor_v1.seleccion_perfil_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contexto_actor_v1.seleccion_perfil_admin_v1 FROM PUBLIC;
CREATE TABLE vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1(
 cuenta_ref text PRIMARY KEY,revision numeric(20,0) NOT NULL,
 FOREIGN KEY(cuenta_ref,revision) REFERENCES vec_contexto_actor_v1.seleccion_perfil_admin_v1(cuenta_ref,revision)
);
REVOKE ALL ON TABLE vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 FROM PUBLIC;
ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.seleccion_perfil_admin_v1 FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user='vec_contexto_actor_v1_propietario') WITH CHECK(current_user='vec_contexto_actor_v1_propietario');
ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user='vec_contexto_actor_v1_propietario') WITH CHECK(current_user='vec_contexto_actor_v1_propietario');
CREATE TRIGGER seleccion_perfil_admin_historia BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_perfil_admin_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER seleccion_perfil_admin_no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.seleccion_perfil_admin_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE TRIGGER seleccion_perfil_admin_actual_no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE FUNCTION vec_contexto_actor_v1.validar_avance_seleccion_perfil_admin_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'CA23: selección no eliminable' USING ERRCODE='23514'; END IF;
 IF (TG_OP='INSERT' AND NEW.revision<>1) OR (TG_OP='UPDATE' AND (NEW.cuenta_ref IS DISTINCT FROM OLD.cuenta_ref OR NEW.revision IS DISTINCT FROM OLD.revision+1)) THEN RAISE EXCEPTION 'CA23: avance de selección inválido' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.validar_avance_seleccion_perfil_admin_v1() FROM PUBLIC;
CREATE TRIGGER seleccion_perfil_admin_actual_avance BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.validar_avance_seleccion_perfil_admin_v1();

-- Ambos perfiles proceden del mismo catálogo AUT24 central; no hay lector paralelo de Sistemas.
CREATE FUNCTION vec_contexto_actor_v1.consultar_seleccion_perfil_admin_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,fuente text,seleccion_revision numeric,seleccionada_en timestamptz,auditoria_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE rev numeric;r record;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation') NOT IN('serializable','read committed') THEN RETURN; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT a.revision INTO rev FROM vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 a WHERE a.cuenta_ref=p_cuenta FOR SHARE;
 IF NOT FOUND THEN RETURN; END IF;
 SELECT x.* INTO STRICT r FROM vec_contexto_actor_v1.seleccion_perfil_admin_v1 x WHERE x.cuenta_ref=p_cuenta AND x.revision=rev;
 RETURN QUERY SELECT r.persona_ref,r.perfil_ref,r.fuente,r.revision,r.seleccionada_en,r.auditoria_ref;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.consultar_seleccion_perfil_admin_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.listar_perfiles_admin_propios_v1(p_cuenta text,p_persona text,p_audiencia text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz,fuente text,rol_version_ref text,clave_i18n text,categoria_admin text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE b record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA23: perfiles propios requieren SERIALIZABLE de escritura' USING ERRCODE='25000'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 IF vec_autorizacion.revalidar_audiencia_selector_admin_v1(p_audiencia) IS NOT TRUE THEN RAISE EXCEPTION 'CA23: audiencia del selector no acreditada' USING ERRCODE='42501'; END IF;
 FOR b IN SELECT x.* FROM vec_autorizacion.listar_asignaciones_admin_perfiles_propias_v1(p_cuenta) x ORDER BY x.perfil_ref LOOP
  IF b.persona_ref IS DISTINCT FROM p_persona THEN CONTINUE; END IF;
  PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
  ahora:=pg_catalog.clock_timestamp();
  IF b.vigente_hasta IS NULL OR NOT pg_catalog.isfinite(b.vigente_hasta) OR ahora>=b.vigente_hasta THEN CONTINUE; END IF;
  RETURN QUERY SELECT b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version,b.audiencia,b.vigente_hasta,'vec_autorizacion.admin'::text,b.rol_version_ref,b.clave_i18n,b.categoria_admin;
 END LOOP;
 -- Las familias y categorías proceden del mismo catálogo central AUT24.
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.listar_perfiles_admin_propios_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.listar_perfiles_admin_propios_reconciliacion_v1(p_cuenta text,p_persona text,p_audiencia text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz,fuente text,rol_version_ref text,clave_i18n text,categoria_admin text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE b record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA23: listado propio requiere READ COMMITTED de escritura' USING ERRCODE='25000'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 IF vec_autorizacion.revalidar_audiencia_selector_admin_v1(p_audiencia) IS NOT TRUE THEN RAISE EXCEPTION 'CA23: audiencia del selector no acreditada' USING ERRCODE='42501'; END IF;
 FOR b IN SELECT x.* FROM vec_autorizacion.listar_asignaciones_admin_perfiles_propias_reconciliacion_v1(p_cuenta) x ORDER BY x.perfil_ref LOOP
  IF b.persona_ref IS DISTINCT FROM p_persona THEN CONTINUE; END IF;
  PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
  ahora:=pg_catalog.clock_timestamp();
  IF b.vigente_hasta IS NULL OR NOT pg_catalog.isfinite(b.vigente_hasta) OR ahora>=b.vigente_hasta THEN CONTINUE; END IF;
  RETURN QUERY SELECT b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version,b.audiencia,b.vigente_hasta,'vec_autorizacion.admin'::text,b.rol_version_ref,b.clave_i18n,b.categoria_admin;
 END LOOP;
 -- Las familias y categorías proceden del mismo catálogo central AUT24.
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.listar_perfiles_admin_propios_reconciliacion_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.seleccionar_perfil_admin_propietaria_v1(p_cuenta text,p_persona text,p_perfil text,p_revision_esperada numeric,p_audiencia text,p_autenticada timestamptz,p_observacion_hasta timestamptz,p_causa text)
RETURNS TABLE(perfil_ref text,seleccion_revision numeric,seleccionada_en timestamptz,auditoria_ref text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE b record;r record;rev numeric:=0;ahora timestamptz;doc jsonb;huella text;audit text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA23: selección requiere SERIALIZABLE de escritura' USING ERRCODE='25000'; END IF;
 IF vec_contexto_actor_v1.referencia_valida(p_cuenta,'cta_') IS NOT TRUE OR vec_contexto_actor_v1.referencia_valida(p_persona,'per_') IS NOT TRUE OR vec_contexto_actor_v1.referencia_valida(p_perfil,'prf_') IS NOT TRUE
 OR p_revision_esperada IS NULL OR p_revision_esperada NOT BETWEEN 0 AND 18446744073709551614::numeric OR p_revision_esperada<>pg_catalog.trunc(p_revision_esperada)
 OR p_autenticada IS NULL OR p_observacion_hasta IS NULL OR NOT pg_catalog.isfinite(p_autenticada) OR NOT pg_catalog.isfinite(p_observacion_hasta)
 OR p_causa IS NULL OR p_causa NOT IN('eleccion_explicita','autoseleccion_unica') THEN RAISE EXCEPTION 'CA23: selección inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 IF vec_autorizacion.revalidar_audiencia_selector_admin_v1(p_audiencia) IS NOT TRUE THEN RAISE EXCEPTION 'CA23: audiencia del selector no acreditada' USING ERRCODE='42501'; END IF;
 SELECT a.revision INTO rev FROM vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 a WHERE a.cuenta_ref=p_cuenta FOR UPDATE;
 IF NOT FOUND THEN rev:=0; END IF;
 IF rev IS DISTINCT FROM p_revision_esperada THEN RAISE EXCEPTION 'CA23: revisión de selección obsoleta' USING ERRCODE='40001'; END IF;
 -- Perfil del body es solo una referencia técnica: AUT24 debe acreditar cuenta,
 -- persona, rol, ámbito, huella y vigencia antes de cambiar la elección.
 SELECT * INTO STRICT b FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta,p_perfil);
 IF b.persona_ref IS DISTINCT FROM p_persona OR b.perfil_ref IS DISTINCT FROM p_perfil THEN RAISE EXCEPTION 'CA23: perfil ajeno a la cuenta' USING ERRCODE='42501'; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
 IF p_causa='autoseleccion_unica' THEN RAISE EXCEPTION 'CA23: unicidad ADMIN no acreditada entre fuentes' USING ERRCODE='42501'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora<p_autenticada OR ahora>=p_observacion_hasta OR b.vigente_hasta IS NULL OR ahora>=b.vigente_hasta THEN RAISE EXCEPTION 'CA23: selección sin observación vigente' USING ERRCODE='42501'; END IF;
 IF rev>0 THEN
  SELECT x.* INTO STRICT r FROM vec_contexto_actor_v1.seleccion_perfil_admin_v1 x WHERE x.cuenta_ref=p_cuenta AND x.revision=rev;
  IF r.persona_ref IS DISTINCT FROM p_persona THEN RAISE EXCEPTION 'CA23: titular de selección divergente' USING ERRCODE='42501'; END IF;
  IF r.perfil_ref=p_perfil AND r.fuente='vec_autorizacion.admin' THEN RETURN QUERY SELECT r.perfil_ref,r.revision,r.seleccionada_en,r.auditoria_ref; RETURN; END IF;
 END IF;
 rev:=rev+1;
 doc:=pg_catalog.jsonb_build_object('esquema','vec.admin.seleccion-perfil.v1','actor_persona_ref',p_persona,'cuenta_ref',p_cuenta,'perfil_ref',p_perfil,'vinculo_ref',b.vinculo_ref,'fuente','vec_autorizacion.admin','revision',rev,'causa',p_causa,'autenticacion_verificada_en',p_autenticada,'seleccionada_en',ahora);
 huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc::text,'UTF8')),'hex');
 audit:='auditoria_seleccion_admin:'||huella;
 INSERT INTO vec_contexto_actor_v1.seleccion_perfil_admin_v1 VALUES(p_cuenta,rev,p_persona,p_perfil,b.vinculo_ref,'vec_autorizacion.admin',p_causa,p_autenticada,ahora,audit,doc,huella);
 IF rev=1 THEN INSERT INTO vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 VALUES(p_cuenta,rev);
 ELSE UPDATE vec_contexto_actor_v1.seleccion_perfil_admin_actual_v1 a SET revision=rev WHERE a.cuenta_ref=p_cuenta AND a.revision=p_revision_esperada;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA23: CAS de selección perdido' USING ERRCODE='40001'; END IF;
 END IF;
 RETURN QUERY SELECT p_perfil,rev,ahora,audit;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RAISE EXCEPTION 'CA23: perfil propio no acreditado' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.seleccionar_perfil_admin_propietaria_v1(text,text,text,numeric,text,timestamptz,timestamptz,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.consultar_seleccion_perfil_admin_v1(text),vec_contexto_actor_v1.listar_perfiles_admin_propios_v1(text,text,text),vec_contexto_actor_v1.listar_perfiles_admin_propios_reconciliacion_v1(text,text,text),vec_contexto_actor_v1.seleccionar_perfil_admin_propietaria_v1(text,text,text,numeric,text,timestamptz,timestamptz,text) TO vec_identidad_sesiones_v1_propietario;

CREATE FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz,seleccion_revision numeric,rol_id text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE b record;sel record;meta record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation')<>'serializable' THEN RETURN; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT sel FROM vec_contexto_actor_v1.consultar_seleccion_perfil_admin_v1(p_cuenta);
 IF sel.fuente IS DISTINCT FROM 'vec_autorizacion.admin' THEN RETURN; END IF;
 SELECT * INTO STRICT b FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta,sel.perfil_ref);
 IF b.persona_ref IS DISTINCT FROM sel.persona_ref OR b.perfil_ref IS DISTINCT FROM sel.perfil_ref OR b.vinculo_ref IS NULL
 OR b.cuenta_version IS NULL OR b.persona_version IS NULL OR b.perfil_version IS NULL OR b.vinculo_version IS NULL
 OR b.audiencia IS NULL OR b.audiencia !~ '^[a-z0-9][a-z0-9._:-]{3,255}$' OR b.vigente_hasta IS NULL OR NOT pg_catalog.isfinite(b.vigente_hasta) THEN RETURN; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
 SELECT * INTO STRICT meta FROM vec_autorizacion.consultar_metadata_perfil_admin_v1(p_cuenta,b.perfil_ref);
 IF meta.rol_id IS NULL OR meta.rol_version_ref IS NULL OR meta.rol_huella_sha256 IS NULL OR meta.rol_huella_sha256 !~ '^[0-9a-f]{64}$' THEN RETURN; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora>=b.vigente_hasta THEN RETURN; END IF;
 RETURN QUERY SELECT b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version,b.audiencia,b.vigente_hasta,sel.seleccion_revision,meta.rol_id;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_reconciliacion_v1(p_cuenta text)
RETURNS TABLE(persona_ref text,perfil_ref text,vinculo_ref text,cuenta_version numeric,persona_version numeric,perfil_version numeric,vinculo_version numeric,audiencia text,vigente_hasta timestamptz,seleccion_revision numeric,rol_id text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE b record;sel record;meta record;ahora timestamptz;
BEGIN
 IF current_setting('transaction_read_only')<>'off' OR current_setting('transaction_isolation')<>'read committed' THEN RETURN; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT * INTO STRICT sel FROM vec_contexto_actor_v1.consultar_seleccion_perfil_admin_v1(p_cuenta);
 IF sel.fuente IS DISTINCT FROM 'vec_autorizacion.admin' THEN RETURN; END IF;
 SELECT * INTO STRICT b FROM vec_autorizacion.consultar_asignacion_admin_perfiles_reconciliacion_v1(p_cuenta,sel.perfil_ref);
 IF b.persona_ref IS DISTINCT FROM sel.persona_ref OR b.perfil_ref IS DISTINCT FROM sel.perfil_ref OR b.vinculo_ref IS NULL
 OR b.cuenta_version IS NULL OR b.persona_version IS NULL OR b.perfil_version IS NULL OR b.vinculo_version IS NULL
 OR b.audiencia IS NULL OR b.audiencia !~ '^[a-z0-9][a-z0-9._:-]{3,255}$' OR b.vigente_hasta IS NULL OR NOT pg_catalog.isfinite(b.vigente_hasta) THEN RETURN; END IF;
 PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
 SELECT * INTO STRICT meta FROM vec_autorizacion.consultar_metadata_perfil_admin_v1(p_cuenta,b.perfil_ref);
 IF meta.rol_id IS NULL OR meta.rol_version_ref IS NULL OR meta.rol_huella_sha256 IS NULL OR meta.rol_huella_sha256 !~ '^[0-9a-f]{64}$' THEN RETURN; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF ahora>=b.vigente_hasta THEN RETURN; END IF;
 RETURN QUERY SELECT b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version,b.audiencia,b.vigente_hasta,sel.seleccion_revision,meta.rol_id;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.consultar_perfil_admin_perfiles_reconciliacion_v1(text) FROM PUBLIC;

-- Procedencia de un acto V3 consumido: solo ciclo de perfil/vínculo existente.
-- AUT24 la registra antes de CA20 y del avance de continuidad, en el mismo COMMIT.
CREATE TABLE vec_contexto_actor_v1.procedencia_acto_admin_v1(
 acto_ref text PRIMARY KEY CHECK(acto_ref ~ '^acto_admin:[0-9a-f]{32}$'),
 operacion_ref text NOT NULL UNIQUE CHECK(pg_catalog.octet_length(operacion_ref) BETWEEN 1 AND 512 AND operacion_ref COLLATE "C" !~ '[^!-~]' AND pg_catalog.strpos(operacion_ref,'*')=0),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL CHECK(pg_catalog.octet_length(decision_ref) BETWEEN 1 AND 512 AND decision_ref COLLATE "C" !~ '[^!-~]' AND pg_catalog.strpos(decision_ref,'*')=0),
 auditoria_ref text NOT NULL CHECK(pg_catalog.octet_length(auditoria_ref) BETWEEN 1 AND 512 AND auditoria_ref COLLATE "C" !~ '[^!-~]' AND pg_catalog.strpos(auditoria_ref,'*')=0),
 perfil_ref text NOT NULL,vinculo_ref text NOT NULL,
 perfil_version_previa numeric(20,0) NOT NULL,vinculo_version_previa numeric(20,0) NOT NULL,
 estado_nuevo text NOT NULL CHECK(estado_nuevo='revocado'),
 procedencia_ref text NOT NULL UNIQUE,procedencia_version numeric(20,0) NOT NULL CHECK(procedencia_version=1),
 procedencia_huella_sha256 text NOT NULL,
 documento jsonb NOT NULL,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp(),
 CHECK(procedencia_huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(documento::text,'UTF8')),'hex')),
 FOREIGN KEY(perfil_ref,perfil_version_previa) REFERENCES vec_contexto_actor_v1.perfil_versiones(perfil_ref,version),
 FOREIGN KEY(vinculo_ref,vinculo_version_previa) REFERENCES vec_contexto_actor_v1.vinculo_contexto_versiones(vinculo_ref,version),
 FOREIGN KEY(procedencia_ref,procedencia_version) REFERENCES vec_contexto_actor_v1.procedencias(procedencia_ref,procedencia_version)
);
REVOKE ALL ON TABLE vec_contexto_actor_v1.procedencia_acto_admin_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_contexto_actor_v1.procedencia_acto_admin_v1 FROM PUBLIC;
ALTER TABLE vec_contexto_actor_v1.procedencia_acto_admin_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_contexto_actor_v1.procedencia_acto_admin_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.procedencia_acto_admin_v1 FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user='vec_contexto_actor_v1_propietario') WITH CHECK(current_user='vec_contexto_actor_v1_propietario');
CREATE TRIGGER procedencia_acto_admin_historia BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.procedencia_acto_admin_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER procedencia_acto_admin_no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.procedencia_acto_admin_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();

CREATE FUNCTION vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1(p_acto_ref text,p_operacion_ref text,p_material_sha256 text,p_decision_ref text,p_auditoria_ref text,p_perfil_ref text,p_vinculo_ref text,p_perfil_ver_prev numeric,p_vinculo_ver_prev numeric,p_estado_nuevo text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE pf record;v record;prev record;doc jsonb;ref text;huella text;ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA23: procedencia requiere SERIALIZABLE de escritura' USING ERRCODE='25000'; END IF;
 IF p_acto_ref IS NULL OR p_acto_ref !~ '^acto_admin:[0-9a-f]{32}$'
 OR p_material_sha256 IS NULL OR p_material_sha256 !~ '^[0-9a-f]{64}$'
 OR p_estado_nuevo IS DISTINCT FROM 'revocado'
 OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE OR vec_contexto_actor_v1.referencia_valida(p_vinculo_ref,'vca_') IS NOT TRUE
 OR p_perfil_ver_prev IS NULL OR p_vinculo_ver_prev IS NULL
 OR p_perfil_ver_prev NOT BETWEEN 1 AND 18446744073709551614::numeric OR p_vinculo_ver_prev NOT BETWEEN 1 AND 18446744073709551614::numeric
 OR p_perfil_ver_prev<>pg_catalog.trunc(p_perfil_ver_prev) OR p_vinculo_ver_prev<>pg_catalog.trunc(p_vinculo_ver_prev)
 OR EXISTS(SELECT 1 FROM pg_catalog.unnest(ARRAY[p_operacion_ref,p_decision_ref,p_auditoria_ref]) t(ref) WHERE t.ref IS NULL OR pg_catalog.octet_length(t.ref) NOT BETWEEN 1 AND 512 OR t.ref COLLATE "C" ~ '[^!-~]' OR pg_catalog.strpos(t.ref,'*')>0)
 THEN RAISE EXCEPTION 'CA23: descriptor de acto inválido' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 SELECT x.* INTO STRICT pf FROM vec_contexto_actor_v1.perfil_actual a JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version) WHERE a.perfil_ref=p_perfil_ref FOR UPDATE OF a;
 SELECT x.* INTO STRICT v FROM vec_contexto_actor_v1.vinculo_contexto_actual a JOIN vec_contexto_actor_v1.vinculo_contexto_versiones x USING(vinculo_ref,version) WHERE a.vinculo_ref=p_vinculo_ref FOR UPDATE OF a;
 IF v.perfil_ref IS DISTINCT FROM p_perfil_ref OR v.persona_ref IS DISTINCT FROM pf.persona_ref THEN RAISE EXCEPTION 'CA23: perfil y vínculo divergentes' USING ERRCODE='42501'; END IF;
 doc:=pg_catalog.jsonb_build_object('esquema','vec.contexto-actor.acto-perfil.v1','acto_ref',p_acto_ref,'operacion_ref',p_operacion_ref,'material_sha256',p_material_sha256,'decision_ref',p_decision_ref,'auditoria_ref',p_auditoria_ref,'perfil_ref',p_perfil_ref,'vinculo_ref',p_vinculo_ref,'perfil_version_previa',p_perfil_ver_prev,'vinculo_version_previa',p_vinculo_ver_prev,'estado_nuevo',p_estado_nuevo);
 ref:='prc_admin_acto_'||pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_acto_ref,'UTF8')),'hex');
 huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(doc::text,'UTF8')),'hex');
 SELECT x.* INTO prev FROM vec_contexto_actor_v1.procedencia_acto_admin_v1 x WHERE x.acto_ref=p_acto_ref;
 IF FOUND THEN
  IF prev.documento IS DISTINCT FROM doc THEN RAISE EXCEPTION 'CA23: acto con material divergente' USING ERRCODE='23505'; END IF;
  IF NOT ((pf.version,v.version,pf.estado,v.estado) IS NOT DISTINCT FROM (p_perfil_ver_prev,p_vinculo_ver_prev,'activo'::text,'activo'::text)
   OR ((pf.version,v.version,pf.estado,v.estado) IS NOT DISTINCT FROM (p_perfil_ver_prev+1,p_vinculo_ver_prev+1,'revocado'::text,'revocado'::text)
    AND (pf.procedencia_ref,pf.procedencia_version,pf.procedencia_huella_sha256,v.procedencia_ref,v.procedencia_version,v.procedencia_huella_sha256) IS NOT DISTINCT FROM (ref,1::numeric,huella,ref,1::numeric,huella)))
  THEN RAISE EXCEPTION 'CA23: replay de acto con versiones divergentes' USING ERRCODE='40001'; END IF;
  RETURN pg_catalog.jsonb_build_object('procedencia_ref',prev.procedencia_ref,'procedencia_version',prev.procedencia_version,'procedencia_huella_sha256',prev.procedencia_huella_sha256);
 END IF;
 IF (pf.version,v.version) IS DISTINCT FROM (p_perfil_ver_prev,p_vinculo_ver_prev) THEN RAISE EXCEPTION 'CA23: CAS de acto divergente' USING ERRCODE='40001'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF pf.estado<>'activo' OR v.estado<>'activo' OR pf.procedencia_autoridad<>'autoridad_maestra_acreditada' OR v.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR ahora<GREATEST(pf.vigente_desde,v.vigente_desde) OR ahora>=LEAST(pf.vigente_hasta,v.vigente_hasta) THEN RAISE EXCEPTION 'CA23: acto sobre perfil no vigente' USING ERRCODE='42501'; END IF;
 -- La clasificación satisface el contrato CA20 de ciclo de perfil. No crea ni
 -- modifica procedencia de personas o cuentas; AUT24 ya consumió la decisión V3.
 INSERT INTO vec_contexto_actor_v1.procedencias VALUES(ref,1,huella,'autoridad_maestra_acreditada');
 INSERT INTO vec_contexto_actor_v1.procedencia_acto_admin_v1 VALUES(p_acto_ref,p_operacion_ref,p_material_sha256,p_decision_ref,p_auditoria_ref,p_perfil_ref,p_vinculo_ref,p_perfil_ver_prev,p_vinculo_ver_prev,p_estado_nuevo,ref,1,huella,doc,ahora);
 RETURN pg_catalog.jsonb_build_object('procedencia_ref',ref,'procedencia_version',1,'procedencia_huella_sha256',huella);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1(text,text,text,text,text,text,text,numeric,numeric,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.registrar_procedencia_acto_admin_v1(text,text,text,text,text,text,text,numeric,numeric,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1()
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE l record;g record;fs oid[];ns oid[];base oid;
BEGIN
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_identidad_sesiones_v1_admin_perfiles';
 SELECT oid INTO base FROM pg_catalog.pg_database WHERE datname=current_database();
 fs:=ARRAY[pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1()'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz)'),pg_catalog.to_regprocedure('vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.resolver_cuenta_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.vincular_sesion_admin_perfiles_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,text)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.listar_perfiles_admin_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.seleccionar_perfil_admin_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz,text,numeric)'),pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.autoseleccionar_perfil_admin_unico_v1(text,text,text,text,text,timestamptz,timestamptz,timestamptz,timestamptz)')];
 ns:=ARRAY[pg_catalog.to_regnamespace('vec_contexto_actor_v1'),pg_catalog.to_regnamespace('vec_identidad_sesiones_v1')];
 IF l.oid IS NULL OR g.oid IS NULL OR array_position(fs,NULL) IS NOT NULL OR array_position(ns,NULL) IS NOT NULL
 OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR current_setting('role')<>'none'
 OR (SELECT count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.oid<>g.oid AND r.oid<>l.oid AND pg_catalog.pg_has_role(l.oid,r.oid,'MEMBER'))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_default_acl d LEFT JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.defaclacl,'{}'::aclitem[])) a ON true WHERE d.defaclrole IN(l.oid,g.oid) OR a.grantee IN(l.oid,g.oid) OR a.grantor IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_policy p WHERE l.oid=ANY(p.polroles) OR g.oid=ANY(p.polroles))
 OR NOT COALESCE((SELECT count(*)=1 AND bool_and(a.privilege_type='CONNECT' AND NOT a.is_grantable) FROM pg_catalog.pg_database d CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(d.datacl,pg_catalog.acldefault('d',d.datdba))) a WHERE d.oid=base AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=2 AND bool_and(a.privilege_type='USAGE' AND NOT a.is_grantable) FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a WHERE n.oid=ANY(ns) AND a.grantee=g.oid),false)
 OR NOT COALESCE((SELECT count(*)=8 AND bool_and(a.privilege_type='EXECUTE' AND NOT a.is_grantable) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=ANY(fs) AND a.grantee=g.oid),false)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a WHERE n.oid=ANY(ns) AND a.grantee=0)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.pronamespace=ANY(ns) AND a.grantee=0 AND p.prosecdef)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.relnamespace=ANY(ns) AND c.relkind IN('r','p','v','m','S') AND (pg_catalog.has_table_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR pg_catalog.has_any_column_privilege(l.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES')))

 OR NOT COALESCE((SELECT count(*)=11 AND bool_and(deptype='a' AND objsubid=0 AND ((classid='pg_catalog.pg_database'::regclass AND objid=base) OR (classid='pg_catalog.pg_namespace'::regclass AND objid=ANY(ns)) OR (classid='pg_catalog.pg_proc'::regclass AND objid=ANY(fs)))) FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid),false)
 THEN RAISE EXCEPTION 'CA23: LOGIN ADMIN no acreditado' USING ERRCODE='42501'; END IF;
 RETURN session_user;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1() FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1()
RETURNS TABLE(identidad_login text,acreditada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 RETURN QUERY SELECT vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1(),true;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1() FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(p_operacion text,p_registro text,p_cuenta text,p_solicitado timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE b record;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA23: aislamiento de contexto incompatible' USING ERRCODE='25000'; END IF;
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion,'oca_') IS NOT TRUE THEN RAISE EXCEPTION 'CA23: operación de contexto inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:operacion:v2:'||p_operacion,0));
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(p_cuenta);
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2_propietaria_admin_v1(p_operacion,p_registro,p_cuenta,b.perfil_ref,'certificado','alto',p_solicitado,'{}'::text[]);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz) FROM PUBLIC;
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(p_operacion text,p_registro text,p_cuenta text,p_solicitado timestamptz)
RETURNS TABLE(operacion_ref text,registro_contexto_ref text,representacion_canonica bytea,huella_sha256 text,manifiesto_procedencia_canonico bytea,manifiesto_procedencia_huella_sha256 text,autoridad_efectiva text,resuelto_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE b record;
BEGIN
 PERFORM vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1();
 IF current_setting('transaction_isolation')<>'read committed' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA23: aislamiento de contexto incompatible' USING ERRCODE='25000'; END IF;
 IF vec_contexto_actor_v1.referencia_operacion_valida(p_operacion,'oca_') IS NOT TRUE THEN RAISE EXCEPTION 'CA23: operación de contexto inválida' USING ERRCODE='22023'; END IF;
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:operacion:v2:'||p_operacion,0));
 SELECT * INTO STRICT b FROM vec_contexto_actor_v1.consultar_perfil_admin_perfiles_reconciliacion_v1(p_cuenta);
 RETURN QUERY SELECT * FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2_propietaria_admin_v1(p_operacion,p_registro,p_cuenta,b.perfil_ref,'certificado','alto',p_solicitado,'{}'::text[]);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz) FROM PUBLIC;


REVOKE ALL ON FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz),vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_identidad_sesiones_v1_admin_perfiles,vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.exigir_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.consultar_perfil_admin_perfiles_v1(text) TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.acreditar_runtime_admin_perfiles_v1(),vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1(text,text,text,timestamptz),vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1(text,text,text,timestamptz) TO vec_identidad_sesiones_v1_admin_perfiles;
COMMIT;
