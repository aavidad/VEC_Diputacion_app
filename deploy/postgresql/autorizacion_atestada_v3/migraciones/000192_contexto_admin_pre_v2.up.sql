\set ON_ERROR_STOP on
-- AD192: familia propia antes de V2; conserva cadena y CHECK medidos post191.
BEGIN;
SET LOCAL search_path=pg_catalog;SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000192',0));
DO $pre$ DECLARE actual text;BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN RAISE EXCEPTION 'AD192: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501';END IF;
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO actual FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4' AND c.convalidated;
 IF actual IS DISTINCT FROM '6b92079faedcd2482b720b1d0714ba6a1b09dc359badae8f6d7c0dd3f3275caf' THEN RAISE EXCEPTION 'AD192: PARO clave=CHECK_sha actual=% esperado=6b92079faedcd2482b720b1d0714ba6a1b09dc359badae8f6d7c0dd3f3275caf',COALESCE(actual,'ausente') USING ERRCODE='55000';END IF;
 SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_proc WHERE oid=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef;
 IF actual IS DISTINCT FROM 'b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45' THEN RAISE EXCEPTION 'AD192: PARO clave=core_src_sha actual=% esperado=b7eb48be035e854928c9685c916f9139166a197e3cae731510b3597f0fe40a45',COALESCE(actual,'ausente') USING ERRCODE='55000';END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(jsonb)') IS NOT NULL OR to_regclass('vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1') IS NOT NULL THEN RAISE EXCEPTION 'AD192: PARO clave=instalacion actual=existente esperado=sin_AD192' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$ DECLARE anterior text;nueva text;BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND c.conname='auditoria_tipo_disjunto_v4';
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR ('||$tipo$
 tipo_registro IS NOT DISTINCT FROM 'contexto_admin_pre_v2'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL AND version_consumo IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND aprobacion_ref IS NULL AND plan_sha256 IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL AND unidad_alcance IS NULL
 AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL AND bootstrap_solicitud_sha256 IS NULL
 AND mantenimiento_detalle IS NULL AND mantenimiento_solicitud_sha256 IS NULL AND periodica_detalle IS NULL AND preservacion_detalle IS NULL
 AND gobierno_usuarios_detalle IS NULL AND gobierno_usuarios_solicitud_sha256 IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT NULL AND accion IN('resolver_cuenta_admin','vincular_sesion_admin','registrar_contexto_admin','reconciliar_contexto_admin')
 AND modulo_id IS NOT DISTINCT FROM 'administracion' AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL
 AND resultado IS NOT NULL AND resultado IN('permitido','denegado','error') AND motivo_ref IS NOT NULL AND proceso IS NOT NULL
 AND canal IS NOT DISTINCT FROM 'administracion_privilegiada' AND finalidad_ref IS NOT DISTINCT FROM 'establecer_contexto_admin'
 AND ((fuente_ref IS NULL AND fuente_sha256 IS NULL) OR(fuente_ref IS NOT NULL AND fuente_sha256 IS NOT NULL))
 AND (actor_ref IS NULL OR fuente_ref IS NOT NULL) AND(perfil_activo_ref IS NULL OR actor_ref IS NOT NULL)
 AND(resultado<>'permitido' OR(actor_ref IS NOT NULL AND perfil_activo_ref IS NOT NULL AND fuente_ref IS NOT NULL))
 $tipo$||'))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
CREATE TABLE vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1(
 login_nombre name PRIMARY KEY,proceso text NOT NULL CHECK(proceso~'^[a-z][a-z0-9._-]{1,79}$'),
 canal text NOT NULL CHECK(canal='administracion_privilegiada'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
ALTER TABLE vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario ON vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 TO vec_autorizacion_atestada_v3_propietario USING(current_user='vec_autorizacion_atestada_v3_propietario') WITH CHECK(current_user='vec_autorizacion_atestada_v3_propietario');
CREATE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_contexto_admin_pre_v2_v1() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$ BEGIN RAISE EXCEPTION 'AD192: configuracion_inmutable' USING ERRCODE='55000';END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_contexto_admin_pre_v2_v1() FROM PUBLIC;
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_contexto_admin_pre_v2_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion_config_contexto_admin_pre_v2_v1();
REVOKE ALL ON TABLE vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 FROM PUBLIC;
REVOKE ALL ON TYPE vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.motivos_contexto_admin_pre_v2_v1() RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT $datos$[
  {
    "accion": "resolver_cuenta_admin",
    "resultado": "permitido",
    "motivo_ref": "contexto_admin_pre_v2_permitido"
  },
  {
    "accion": "resolver_cuenta_admin",
    "resultado": "denegado",
    "motivo_ref": "contexto_admin_pre_v2_denegado"
  },
  {
    "accion": "resolver_cuenta_admin",
    "resultado": "error",
    "motivo_ref": "contexto_admin_pre_v2_error"
  },
  {
    "accion": "vincular_sesion_admin",
    "resultado": "permitido",
    "motivo_ref": "contexto_admin_pre_v2_permitido"
  },
  {
    "accion": "vincular_sesion_admin",
    "resultado": "denegado",
    "motivo_ref": "contexto_admin_pre_v2_denegado"
  },
  {
    "accion": "vincular_sesion_admin",
    "resultado": "error",
    "motivo_ref": "contexto_admin_pre_v2_error"
  },
  {
    "accion": "registrar_contexto_admin",
    "resultado": "permitido",
    "motivo_ref": "contexto_admin_pre_v2_permitido"
  },
  {
    "accion": "registrar_contexto_admin",
    "resultado": "denegado",
    "motivo_ref": "contexto_admin_pre_v2_denegado"
  },
  {
    "accion": "registrar_contexto_admin",
    "resultado": "error",
    "motivo_ref": "contexto_admin_pre_v2_error"
  },
  {
    "accion": "reconciliar_contexto_admin",
    "resultado": "permitido",
    "motivo_ref": "contexto_admin_pre_v2_permitido"
  },
  {
    "accion": "reconciliar_contexto_admin",
    "resultado": "denegado",
    "motivo_ref": "contexto_admin_pre_v2_denegado"
  },
  {
    "accion": "reconciliar_contexto_admin",
    "resultado": "error",
    "motivo_ref": "contexto_admin_pre_v2_error"
  }
]$datos$::jsonb
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivos_contexto_admin_pre_v2_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.exigir_config_contexto_admin_pre_v2_v1()
RETURNS vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE c vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1;l record;BEGIN
 IF current_setting('role')<>'none' OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' THEN RAISE EXCEPTION 'AD192: transaccion_no_admitida' USING ERRCODE='25000';END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;
 IF NOT FOUND OR NOT l.rolcanlogin OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls THEN RAISE EXCEPTION 'AD192: LOGIN_no_acreditado' USING ERRCODE='42501';END IF;
 SELECT * INTO c FROM vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<c.vigente_desde OR clock_timestamp()>=c.vigente_hasta THEN RAISE EXCEPTION 'AD192: configuracion_no_vigente' USING ERRCODE='42501';END IF;
 RETURN c;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.exigir_config_contexto_admin_pre_v2_v1() FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_interna_v1(e jsonb,p_autoridad text)
RETURNS TABLE(auditoria_ref text,secuencia bigint,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE orden text[]:=ARRAY['tipo_registro','evento_ref','operador_login','actor_ref','perfil_activo_ref','accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256'];campo text;
 c vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1;material bytea;material_sha text;existente record;s bigint;anterior text;h text;ref text;instante timestamptz;
BEGIN
 c:=vec_autorizacion_atestada_v3.exigir_config_contexto_admin_pre_v2_v1();
 IF jsonb_typeof(e) IS DISTINCT FROM 'object' OR NOT e ?& orden OR(SELECT count(*) FROM jsonb_object_keys(e))<>15 THEN RAISE EXCEPTION 'AD192: objeto_cerrado_invalido' USING ERRCODE='22023';END IF;
 FOREACH campo IN ARRAY orden LOOP
  IF jsonb_typeof(e->campo) IS DISTINCT FROM 'string' AND NOT(campo IN('actor_ref','perfil_activo_ref','fuente_ref','fuente_sha256') AND jsonb_typeof(e->campo)='null') THEN RAISE EXCEPTION 'AD192: tipo_campo_invalido' USING ERRCODE='22023';END IF;
 END LOOP;
 IF e->>'tipo_registro' IS DISTINCT FROM 'contexto_admin_pre_v2' OR e->>'operador_login' IS DISTINCT FROM session_user::text OR e->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR e->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$' OR e->>'recurso_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$'
 OR e->>'proceso' IS DISTINCT FROM c.proceso OR e->>'canal' IS DISTINCT FROM c.canal OR e->>'finalidad_ref' IS DISTINCT FROM 'establecer_contexto_admin'
 OR(p_autoridad='is' AND e->>'accion' NOT IN('resolver_cuenta_admin','vincular_sesion_admin')) OR(p_autoridad='ca' AND e->>'accion' NOT IN('registrar_contexto_admin','reconciliar_contexto_admin')) OR p_autoridad IS NULL OR p_autoridad NOT IN('is','ca')
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(vec_autorizacion_atestada_v3.motivos_contexto_admin_pre_v2_v1()) x WHERE x->>'accion'=e->>'accion' AND x->>'resultado'=e->>'resultado' AND x->>'motivo_ref'=e->>'motivo_ref')
 OR ((e->>'fuente_ref') IS NULL) IS DISTINCT FROM ((e->>'fuente_sha256') IS NULL)
 OR(e->>'fuente_ref' IS NOT NULL AND(e->>'fuente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$' OR e->>'fuente_sha256' !~ '^[0-9a-f]{64}$'))
 OR(e->>'actor_ref' IS NOT NULL AND(e->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,124}$' OR e->>'fuente_ref' IS NULL))
 OR(e->>'perfil_activo_ref' IS NOT NULL AND(e->>'perfil_activo_ref' !~ '^prf_[A-Za-z0-9_-]{22,124}$' OR e->>'actor_ref' IS NULL))
 OR(e->>'resultado'='permitido' AND(e->>'actor_ref' IS NULL OR e->>'perfil_activo_ref' IS NULL OR e->>'fuente_ref' IS NULL))
 THEN RAISE EXCEPTION 'AD192: coordenadas_no_acreditadas' USING ERRCODE='22023';END IF;
 material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.contexto-admin-pre-v2.v1');
 FOREACH campo IN ARRAY orden LOOP material:=material||vec_autorizacion_atestada_v3.encuadrar_mac(COALESCE(e->>campo,''));END LOOP;
 material_sha:=encode(sha256(material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:evento-admin:'||(e->>'evento_ref'),0));
 SELECT a.* INTO existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=e->>'evento_ref';
 IF FOUND THEN
  IF existente.tipo_registro IS DISTINCT FROM 'contexto_admin_pre_v2' OR existente.evento_material_sha256 IS DISTINCT FROM material_sha THEN RAISE EXCEPTION 'AD192: replay_distinto' USING ERRCODE='23505';END IF;
  PERFORM vec_autorizacion_atestada_v3.exigir_config_contexto_admin_pre_v2_v1();
  RETURN QUERY SELECT existente.auditoria_ref,existente.secuencia,existente.huella_sha256,existente.correlacion_ref,existente.registrada_en;RETURN;
 END IF;
 SELECT v.secuencia,v.cabeza_sha256 INTO STRICT s,anterior FROM vec_autorizacion_atestada_v3.control_cadena_auditoria v WHERE v.control_id FOR UPDATE;
 IF s>=9007199254740991 THEN RAISE EXCEPTION 'AD192: secuencia_fuera_limite' USING ERRCODE='22003';END IF;
 PERFORM vec_autorizacion_atestada_v3.exigir_config_contexto_admin_pre_v2_v1();
 s:=s+1;instante:=clock_timestamp();ref:='aud_v3_ap2_'||substr(e->>'evento_ref',8,32);
 h:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.contexto-admin-pre-v2.v1')||vec_autorizacion_atestada_v3.encuadrar_mac(s::text)||vec_autorizacion_atestada_v3.encuadrar_mac(anterior)||vec_autorizacion_atestada_v3.encuadrar_mac(ref)||vec_autorizacion_atestada_v3.encuadrar_mac(material_sha)||vec_autorizacion_atestada_v3.encuadrar_mac(to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,evento_ref,evento_material_sha256,operador_login,actor_ref,perfil_activo_ref,accion,modulo_id,recurso_ref,resultado,motivo_ref,proceso,canal,finalidad_ref,correlacion_ref,fuente_ref,fuente_sha256)
 VALUES(ref,s,anterior,h,instante,'contexto_admin_pre_v2',e->>'evento_ref',material_sha,session_user,e->>'actor_ref',e->>'perfil_activo_ref',e->>'accion','administracion',e->>'recurso_ref',e->>'resultado',e->>'motivo_ref',c.proceso,c.canal,'establecer_contexto_admin',e->>'correlacion_ref',e->>'fuente_ref',e->>'fuente_sha256');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria SET secuencia=s,cabeza_sha256=h,actualizada_en=instante WHERE control_id;
 RETURN QUERY SELECT ref,s,h,e->>'correlacion_ref',instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_interna_v1(jsonb,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(e jsonb)
RETURNS TABLE(auditoria_ref text,secuencia bigint,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$ SELECT * FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_interna_v1(e,'is') $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(e jsonb)
RETURNS TABLE(auditoria_ref text,secuencia bigint,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$ SELECT * FROM vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_interna_v1(e,'ca') $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(jsonb),vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_identidad_sesiones_v1_propietario,vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_is_v1(jsonb) TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_contexto_admin_pre_v2_ca_v1(jsonb) TO vec_contexto_actor_v1_propietario;

-- Recuperación de COMMIT incierto: lectura propietario, sin append ni locks
-- de cadena/evento. La misma operación debe aportar su frame15 original.
CREATE FUNCTION vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_interna_v1(e jsonb,p_autoridad text)
RETURNS TABLE(auditoria_ref text,secuencia bigint,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' AS $f$
DECLARE orden text[]:=ARRAY['tipo_registro','evento_ref','operador_login','actor_ref','perfil_activo_ref','accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256'];campo text;material bytea;material_sha text;existente record;
BEGIN
 IF jsonb_typeof(e) IS DISTINCT FROM 'object' OR NOT e ?& orden OR(SELECT count(*) FROM jsonb_object_keys(e))<>15 THEN RAISE EXCEPTION 'AD192: objeto_cotejo_cerrado_invalido' USING ERRCODE='22023';END IF;
 FOREACH campo IN ARRAY orden LOOP
  IF jsonb_typeof(e->campo) IS DISTINCT FROM 'string' AND NOT(campo IN('actor_ref','perfil_activo_ref','fuente_ref','fuente_sha256') AND jsonb_typeof(e->campo)='null') THEN RAISE EXCEPTION 'AD192: tipo_cotejo_invalido' USING ERRCODE='22023';END IF;
  IF campo IN('actor_ref','perfil_activo_ref','fuente_ref','fuente_sha256') AND e->>campo='' THEN RAISE EXCEPTION 'AD192: cadena_nullable_vacia' USING ERRCODE='22023';END IF;
 END LOOP;
 IF e->>'tipo_registro' IS DISTINCT FROM 'contexto_admin_pre_v2' OR e->>'operador_login' IS DISTINCT FROM session_user::text OR e->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_autoridad IS NULL OR p_autoridad NOT IN('is','ca') OR(p_autoridad='is' AND e->>'accion' NOT IN('resolver_cuenta_admin','vincular_sesion_admin')) OR(p_autoridad='ca' AND e->>'accion' NOT IN('registrar_contexto_admin','reconciliar_contexto_admin'))
 THEN RAISE EXCEPTION 'AD192: autoridad_cotejo_invalida' USING ERRCODE='42501';END IF;
 material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.contexto-admin-pre-v2.v1');
 FOREACH campo IN ARRAY orden LOOP material:=material||vec_autorizacion_atestada_v3.encuadrar_mac(COALESCE(e->>campo,''));END LOOP;material_sha:=encode(sha256(material),'hex');
 SELECT a.* INTO existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=e->>'evento_ref';
 IF NOT FOUND THEN RETURN;END IF;
 IF existente.tipo_registro IS DISTINCT FROM 'contexto_admin_pre_v2' OR existente.evento_material_sha256 IS DISTINCT FROM material_sha OR existente.operador_login IS DISTINCT FROM session_user THEN RAISE EXCEPTION 'AD192: cotejo_material_distinto' USING ERRCODE='23505';END IF;
 RETURN QUERY SELECT existente.auditoria_ref,existente.secuencia,existente.huella_sha256,existente.correlacion_ref,existente.registrada_en;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_interna_v1(jsonb,text) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(e jsonb)
RETURNS TABLE(auditoria_ref text,secuencia bigint,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$ SELECT * FROM vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_interna_v1(e,'is') $f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(e jsonb)
RETURNS TABLE(auditoria_ref text,secuencia bigint,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$ SELECT * FROM vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_interna_v1(e,'ca') $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(jsonb),vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_is_v1(jsonb) TO vec_identidad_sesiones_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.cotejar_contexto_admin_pre_v2_ca_v1(jsonb) TO vec_contexto_actor_v1_propietario;
COMMIT;
