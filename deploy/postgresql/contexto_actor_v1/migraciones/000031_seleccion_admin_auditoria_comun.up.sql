\set ON_ERROR_STOP on
-- CA31: selector propio de Aplicación. No concede perfiles ni habilita Sistemas.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regprocedure('vec_autorizacion.consultar_asignacion_admin_perfiles_v1(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.bloquear_contexto_admin_v1(text,text,text,text,numeric,numeric,numeric,numeric)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL
 OR pg_catalog.to_regclass('vec_contexto_actor_v1.seleccion_admin_auditada_v1') IS NOT NULL
 THEN RAISE EXCEPTION 'CA31: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;
-- Proyección del catálogo fijo AUT33: versiones antiguas sin categoría
-- positiva no se reclasifican por nombre de rol.
CREATE FUNCTION vec_autorizacion.consultar_metadata_preperfil_aplicacion_v1(p_cuenta text,p_perfil text)
RETURNS TABLE(rol_version_ref text,clave_i18n text,categoria_admin text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE a record;r record;meta record;e jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA31: metadata requiere SERIALIZABLE READ WRITE' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT x.* INTO STRICT a FROM vec_autorizacion.asignacion_perfil_actual ac JOIN vec_autorizacion.asignacion_perfil x USING(asignacion_ref) WHERE x.perfil_activo_ref=p_perfil AND x.documento->>'estado'='activa' FOR SHARE OF ac;
 e:=vec_contexto_actor_v1.enlace_perfil_admin_interno_v1(a.principal_id,p_perfil);
 IF e->>'cuenta_ref' IS DISTINCT FROM p_cuenta THEN RETURN; END IF;
 SELECT x.* INTO STRICT r FROM vec_autorizacion.version_rol x WHERE x.version_rol_ref=a.version_rol_ref;
 SELECT x.* INTO STRICT meta FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 x WHERE x.version_rol_ref=r.version_rol_ref;
 IF meta.categoria_administrativa IS DISTINCT FROM 'aplicacion' OR meta.tipo_perfil IS DISTINCT FROM 'fijo_sistema'
 OR meta.version_rol_huella_sha256 IS DISTINCT FROM r.huella_sha256
 OR meta.fuente_ref IS DISTINCT FROM r.version_rol_ref OR meta.fuente_version IS DISTINCT FROM r.version
 OR meta.fuente_huella_sha256 IS DISTINCT FROM r.huella_sha256
 OR a.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_asignacion_perfil_admin_v1(a.documento),'UTF8')),'hex')
 OR r.huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(r.documento),'UTF8')),'hex')
 OR r.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.control_vigencia_version_rol_actual c JOIN vec_autorizacion.control_vigencia_version_rol v USING(version_rol_ref,revision) WHERE c.version_rol_ref=r.version_rol_ref AND v.estado='habilitada')
 OR clock_timestamp()<(a.documento->>'vigente_desde')::timestamptz OR clock_timestamp()>=(a.documento->>'vigente_hasta')::timestamptz
 THEN RETURN; END IF;
 RETURN QUERY SELECT r.version_rol_ref,'administracion.perfiles.rol'::text,meta.categoria_administrativa;
EXCEPTION WHEN no_data_found OR too_many_rows THEN RETURN;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.consultar_metadata_preperfil_aplicacion_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.consultar_metadata_preperfil_aplicacion_v1(text,text) TO vec_contexto_actor_v1_propietario;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE TABLE vec_contexto_actor_v1.seleccion_admin_auditada_v1 (
 cuenta_ref text NOT NULL,revision numeric(20,0) NOT NULL CHECK(revision BETWEEN 1 AND 18446744073709551615),
 persona_ref text NOT NULL,perfil_ref text NOT NULL,vinculo_ref text NOT NULL,
 revision_esperada numeric(20,0) NOT NULL CHECK(revision_esperada=revision-1),
 seleccionada_en timestamptz NOT NULL CHECK(pg_catalog.isfinite(seleccionada_en)),
 auditoria_comun_ref text NOT NULL,
 PRIMARY KEY(cuenta_ref,revision),
 CHECK(vec_contexto_actor_v1.referencia_valida(cuenta_ref,'cta_') IS TRUE),
 CHECK(vec_contexto_actor_v1.referencia_valida(persona_ref,'per_') IS TRUE),
 CHECK(vec_contexto_actor_v1.referencia_valida(perfil_ref,'prf_') IS TRUE),
 CHECK(vec_contexto_actor_v1.referencia_valida(vinculo_ref,'vca_') IS TRUE)
);
CREATE TABLE vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 (
 cuenta_ref text PRIMARY KEY,revision numeric(20,0) NOT NULL,
 FOREIGN KEY(cuenta_ref,revision) REFERENCES vec_contexto_actor_v1.seleccion_admin_auditada_v1(cuenta_ref,revision)
);
DO $acl$
DECLARE nombre text;
BEGIN
 FOREACH nombre IN ARRAY ARRAY['seleccion_admin_auditada_v1','seleccion_admin_actual_auditada_v1'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I ENABLE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('ALTER TABLE vec_contexto_actor_v1.%I FORCE ROW LEVEL SECURITY',nombre);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_contexto_actor_v1.%I FOR ALL TO vec_contexto_actor_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',nombre,'vec_contexto_actor_v1_propietario','vec_contexto_actor_v1_propietario');
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_contexto_actor_v1.%I FROM PUBLIC',nombre);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_contexto_actor_v1.%I FROM PUBLIC',nombre);
 END LOOP;
END $acl$;
CREATE TRIGGER historia BEFORE UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_admin_auditada_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_mutacion_historia();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.seleccion_admin_auditada_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();
CREATE FUNCTION vec_contexto_actor_v1.validar_avance_admin_auditado_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 IF TG_OP='DELETE' OR (TG_OP='INSERT' AND NEW.revision<>1)
 OR (TG_OP='UPDATE' AND (NEW.cuenta_ref IS DISTINCT FROM OLD.cuenta_ref OR NEW.revision IS DISTINCT FROM OLD.revision+1))
 THEN RAISE EXCEPTION 'CA31: avance incompatible' USING ERRCODE='23514'; END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.validar_avance_admin_auditado_v1() FROM PUBLIC;
CREATE TRIGGER avance BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.validar_avance_admin_auditado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_contexto_actor_v1.rechazar_truncado();

CREATE FUNCTION vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(p_cuenta text,p_persona text,p_audiencia text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE b record;sel record;meta record;lista jsonb:='[]'::jsonb;hasta timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'CA31: requiere SERIALIZABLE READ WRITE' USING ERRCODE='25000'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 SELECT s.* INTO sel FROM vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 a JOIN vec_contexto_actor_v1.seleccion_admin_auditada_v1 s USING(cuenta_ref,revision) WHERE a.cuenta_ref=p_cuenta FOR SHARE OF a;
 IF sel.persona_ref IS NOT NULL AND sel.persona_ref IS DISTINCT FROM p_persona THEN RAISE EXCEPTION 'CA31: titular divergente' USING ERRCODE='42501'; END IF;
 FOR b IN SELECT x.* FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta) x ORDER BY x.perfil_ref LOOP
  IF b.persona_ref IS DISTINCT FROM p_persona OR b.audiencia IS DISTINCT FROM p_audiencia THEN CONTINUE; END IF;
  PERFORM vec_contexto_actor_v1.bloquear_contexto_admin_v1(p_cuenta,b.persona_ref,b.perfil_ref,b.vinculo_ref,b.cuenta_version,b.persona_version,b.perfil_version,b.vinculo_version);
  SELECT * INTO meta FROM vec_autorizacion.consultar_metadata_preperfil_aplicacion_v1(p_cuenta,b.perfil_ref);
  IF NOT FOUND OR b.vigente_hasta IS NULL OR clock_timestamp()>=b.vigente_hasta THEN CONTINUE; END IF;
  hasta:=CASE WHEN hasta IS NULL THEN b.vigente_hasta ELSE LEAST(hasta,b.vigente_hasta) END;
  lista:=lista||jsonb_build_array(jsonb_build_object('perfil_ref',b.perfil_ref,'rol_version_ref',meta.rol_version_ref,'clave_i18n',meta.clave_i18n,'categoria_admin',meta.categoria_admin));
 END LOOP;
 RETURN jsonb_build_object('revision',COALESCE(sel.revision,0::numeric),'perfil_activo_ref',CASE WHEN EXISTS(SELECT 1 FROM jsonb_array_elements(lista) x WHERE x->>'perfil_ref'=sel.perfil_ref) THEN sel.perfil_ref ELSE '' END,'perfiles',lista,'perfiles_vigentes_hasta',hasta);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_contexto_actor_v1.seleccionar_admin_preperfil_propietaria_v1(p_cuenta text,p_persona text,p_audiencia text,p_perfil text,p_revision_esperada numeric,p_auditoria_comun_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE lista jsonb;b record;sel record;rev numeric:=0;ahora timestamptz;
BEGIN
 IF p_revision_esperada IS NULL OR p_revision_esperada NOT BETWEEN 0 AND 18446744073709551614::numeric OR p_revision_esperada<>trunc(p_revision_esperada)
 OR vec_contexto_actor_v1.referencia_valida(p_perfil,'prf_') IS NOT TRUE
 OR p_auditoria_comun_ref IS NULL OR p_auditoria_comun_ref !~ '^aud_v3_p_[0-9a-f]{32}$' THEN RAISE EXCEPTION 'CA31: selección inválida' USING ERRCODE='22023'; END IF;
 lista:=vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(p_cuenta,p_persona,p_audiencia);
 IF NOT EXISTS(SELECT 1 FROM jsonb_array_elements(lista->'perfiles') x WHERE x->>'perfil_ref'=p_perfil) THEN RAISE EXCEPTION 'CA31: perfil propio no vigente' USING ERRCODE='42501'; END IF;
 SELECT s.* INTO sel FROM vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 a JOIN vec_contexto_actor_v1.seleccion_admin_auditada_v1 s USING(cuenta_ref,revision) WHERE a.cuenta_ref=p_cuenta FOR UPDATE OF a;
 IF FOUND THEN
  rev:=sel.revision;
  -- Repetición del mismo CAS, sin ABA ni elección nueva: autoridad revalidada arriba.
  IF sel.perfil_ref=p_perfil AND sel.persona_ref=p_persona AND p_revision_esperada IN(sel.revision,sel.revision_esperada) THEN
   RETURN jsonb_build_object('perfil_ref',sel.perfil_ref,'seleccion_revision',sel.revision,'seleccionada_en',sel.seleccionada_en);
  END IF;
 END IF;
 IF rev IS DISTINCT FROM p_revision_esperada THEN RAISE EXCEPTION 'CA31: CAS obsoleto' USING ERRCODE='VCA31'; END IF;
 SELECT x.* INTO STRICT b FROM vec_autorizacion.consultar_asignacion_admin_perfiles_v1(p_cuenta) x WHERE x.perfil_ref=p_perfil AND x.persona_ref=p_persona AND x.audiencia=p_audiencia;
 ahora:=clock_timestamp();
 IF ahora>=b.vigente_hasta THEN RAISE EXCEPTION 'CA31: perfil caducado' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_contexto_actor_v1.seleccion_admin_auditada_v1 VALUES(p_cuenta,rev+1,p_persona,p_perfil,b.vinculo_ref,p_revision_esperada,ahora,p_auditoria_comun_ref);
 IF rev=0 THEN INSERT INTO vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 VALUES(p_cuenta,1);
 ELSE UPDATE vec_contexto_actor_v1.seleccion_admin_actual_auditada_v1 a SET revision=rev+1 WHERE a.cuenta_ref=p_cuenta AND a.revision=rev;
  IF NOT FOUND THEN RAISE EXCEPTION 'CA31: CAS perdido' USING ERRCODE='VCA31'; END IF;
 END IF;
 RETURN jsonb_build_object('perfil_ref',p_perfil,'seleccion_revision',rev+1,'seleccionada_en',ahora);
END $f$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.seleccionar_admin_preperfil_propietaria_v1(text,text,text,text,numeric,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.listar_admin_preperfil_propietaria_v1(text,text,text),vec_contexto_actor_v1.seleccionar_admin_preperfil_propietaria_v1(text,text,text,text,numeric,text) TO vec_identidad_sesiones_v1_propietario;
COMMIT;
