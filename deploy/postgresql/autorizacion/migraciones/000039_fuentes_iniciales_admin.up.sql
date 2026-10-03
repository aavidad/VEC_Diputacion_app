\set ON_ERROR_STOP on
-- AUT39: provisión inicial sintética, sin perfiles ni continuidad administrativa.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:fuentes-iniciales:v1',0));
DO $pre$
DECLARE firma text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.to_regclass('vec_autorizacion.fuentes_iniciales_admin_v1') IS NOT NULL
 OR pg_catalog.to_regrole('vec_admin_fuentes_iniciales_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'AUT39: PARO clave=instalacion actual=divergente esperado=primera_instalacion_superusuario' USING ERRCODE='55000'; END IF;
 FOREACH firma IN ARRAY ARRAY[
  'vec_autorizacion.json_cadena_canonica_go_admin_v1(text)',
  'vec_autorizacion.fecha_canonica_go_admin_v1(text,boolean)',
  'vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(jsonb)',
  'vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(jsonb,jsonb,text,text,text,text)',
  'vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(jsonb,text)',
  'vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(jsonb,text,text,text,text,text)',
  'vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text)',
  'vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb)',
  'vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(jsonb)'] LOOP
  IF pg_catalog.to_regprocedure(firma) IS NULL THEN
   RAISE EXCEPTION 'AUT39: PARO clave=dependencia actual=ausente esperado=%',firma USING ERRCODE='55000';
  END IF;
 END LOOP;
 CREATE ROLE vec_admin_fuentes_iniciales_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 EXECUTE pg_catalog.format('GRANT CONNECT ON DATABASE %I TO vec_admin_fuentes_iniciales_ejecutor',pg_catalog.current_database());
END $pre$;
SET LOCAL ROLE vec_autorizacion_propietario;

-- Orden de encoding/json del PlanFuentesInicialesAdminV1; no aprueba JSONB::text.
CREATE FUNCTION vec_autorizacion.canon_fuentes_iniciales_admin_v1(v jsonb,tipo text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE campos text[];tipos text[];i integer;valor text;salida text:='';lista text;item jsonb;
BEGIN
 CASE tipo
 WHEN 'plan' THEN
  campos:=ARRAY['version','operacion_ref','preparado_en','caduca_en','entorno','alcance_fuente','procedencia','organizacion','personas','fuente_hmac','politica_admin'];
  tipos:=ARRAY['n','s','t','t','s','s','e','o','personas','e','p'];
 WHEN 'e' THEN campos:=ARRAY['referencia','version','huella_sha256'];tipos:=ARRAY['s','n','s'];
 WHEN 'o' THEN campos:=ARRAY['organizacion_ref','version_esperada','vigente_hasta'];tipos:=ARRAY['s','n','t'];
 WHEN 'persona' THEN
  campos:=ARRAY['persona_ref','version_esperada','vigente_hasta','operacion_cuenta_ordinaria_ref','operacion_cuenta_privilegiada_ref','fuente_titularidad'];
  tipos:=ARRAY['s','n','t','s','s','e'];
 WHEN 'p' THEN
  campos:=ARRAY['politica_ref','host_admin','ca_sha256','huella_aprobacion_sha256','maxima_edad_revocacion_segundos','vigente_hasta'];
  tipos:=ARRAY['s','s','s','s','n','t'];
 WHEN 's' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'string' OR pg_catalog.octet_length(v#>>'{}') NOT BETWEEN 1 AND 256
  OR (v#>>'{}') ~ '[^!-~]' OR pg_catalog.strpos(v#>>'{}','*')<>0
  THEN RAISE EXCEPTION 'AUT39: PARO clave=cadena actual=invalida esperado=ASCII_acotado' USING ERRCODE='22023'; END IF;
  RETURN vec_autorizacion.json_cadena_canonica_go_admin_v1(v#>>'{}');
 WHEN 'n' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'number' OR (v#>>'{}') !~ '^(0|[1-9][0-9]{0,19})$'
  OR (v#>>'{}')::numeric>18446744073709551615::numeric
  THEN RAISE EXCEPTION 'AUT39: PARO clave=numero actual=invalido esperado=uint64' USING ERRCODE='22023'; END IF;
  RETURN v#>>'{}';
 WHEN 't' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'string' OR (v#>>'{}') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
  THEN RAISE EXCEPTION 'AUT39: PARO clave=fecha actual=invalida esperado=UTC_segundos' USING ERRCODE='22023'; END IF;
  valor:=vec_autorizacion.fecha_canonica_go_admin_v1(v#>>'{}',false);
  RETURN vec_autorizacion.json_cadena_canonica_go_admin_v1(valor);
 WHEN 'personas' THEN
  IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'array' OR pg_catalog.jsonb_array_length(v)<>2
  THEN RAISE EXCEPTION 'AUT39: PARO clave=personas actual=invalido esperado=dos_personas' USING ERRCODE='22023'; END IF;
  lista:='';
  FOR item IN SELECT e.value FROM pg_catalog.jsonb_array_elements(v) WITH ORDINALITY e(value,n) ORDER BY n LOOP
   IF lista<>'' THEN lista:=lista||','; END IF;
   lista:=lista||vec_autorizacion.canon_fuentes_iniciales_admin_v1(item,'persona');
  END LOOP;
  RETURN '['||lista||']';
 ELSE RAISE EXCEPTION 'AUT39: PARO clave=tipo actual=desconocido esperado=contrato_cerrado' USING ERRCODE='22023';
 END CASE;
 IF pg_catalog.jsonb_typeof(v) IS DISTINCT FROM 'object'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v))<>pg_catalog.array_length(campos,1)
 OR NOT v ?& campos
 THEN RAISE EXCEPTION 'AUT39: PARO clave=objeto actual=divergente esperado=campos_exactos' USING ERRCODE='22023'; END IF;
 FOR i IN 1..pg_catalog.array_length(campos,1) LOOP
  valor:=vec_autorizacion.canon_fuentes_iniciales_admin_v1(v->campos[i],tipos[i]);
  IF salida<>'' THEN salida:=salida||','; END IF;
  salida:=salida||vec_autorizacion.json_cadena_canonica_go_admin_v1(campos[i])||':'||valor;
 END LOOP;
 RETURN '{'||salida||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.canon_fuentes_iniciales_admin_v1(jsonb,text) FROM PUBLIC;

-- Sólo el DBA configura. La aplicación no puede publicar una aprobación propia.
CREATE TABLE vec_autorizacion.config_fuentes_iniciales_admin_v1(
 login_nombre name PRIMARY KEY,
 proceso text NOT NULL CHECK(proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
 motivo_ref text NOT NULL CHECK(motivo_ref ~ '^[a-z][a-z0-9._:-]{0,159}$'),
 entorno text NOT NULL CHECK(entorno='desarrollo'),
 alcance_fuente text NOT NULL CHECK(alcance_fuente='sintetico_declarado'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 material_hmac_canonico text NOT NULL CHECK(pg_catalog.octet_length(material_hmac_canonico) BETWEEN 1 AND 65536),
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) NOT NULL,
 CHECK(pg_catalog.isfinite(vigente_desde) AND pg_catalog.isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
CREATE TABLE vec_autorizacion.fuentes_iniciales_admin_v1(
 operacion_ref text PRIMARY KEY CHECK(operacion_ref ~ '^pfi_[A-Za-z0-9_-]{22,124}$'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 plan_canonico bytea NOT NULL CHECK(plan_sha256=pg_catalog.encode(pg_catalog.sha256(plan_canonico),'hex')),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 configuracion_sha256 text NOT NULL CHECK(configuracion_sha256 ~ '^[0-9a-f]{64}$'),
 operador_login name NOT NULL,
 aprobacion_ref text NOT NULL,
 auditoria_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 recibo jsonb NOT NULL,
 registrada_en timestamptz(6) NOT NULL CHECK(pg_catalog.isfinite(registrada_en))
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['config_fuentes_iniciales_admin_v1','fuentes_iniciales_admin_v1'] LOOP
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE pg_catalog.format('CREATE POLICY propietario_exacto ON vec_autorizacion.%I FOR ALL TO vec_autorizacion_propietario USING(current_user=''vec_autorizacion_propietario'') WITH CHECK(current_user=''vec_autorizacion_propietario'')',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion.%I FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE pg_catalog.format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion.rechazar_mutacion_inmutable()',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TABLE vec_autorizacion.%I FROM PUBLIC,vec_admin_fuentes_iniciales_ejecutor',t);
  EXECUTE pg_catalog.format('REVOKE ALL ON TYPE vec_autorizacion.%I FROM PUBLIC,vec_admin_fuentes_iniciales_ejecutor',t);
 END LOOP;
END $tablas$;

CREATE FUNCTION vec_autorizacion.exigir_operador_fuentes_iniciales_admin_v1()
RETURNS vec_autorizacion.config_fuentes_iniciales_admin_v1
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE l record;g record;cfg vec_autorizacion.config_fuentes_iniciales_admin_v1;
 ns oid;db oid;funcion oid;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC' OR pg_catalog.current_setting('role')<>'none'
 THEN RAISE EXCEPTION 'AUT39: PARO clave=transaccion actual=incompatible esperado=serializable_RW_UTC_sin_SETROLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_catalog.pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_catalog.pg_roles WHERE rolname='vec_admin_fuentes_iniciales_ejecutor';
 ns:=pg_catalog.to_regnamespace('vec_autorizacion');
 SELECT oid INTO db FROM pg_catalog.pg_database WHERE datname=pg_catalog.current_database();
 funcion:=pg_catalog.to_regprocedure('vec_autorizacion.provisionar_fuentes_iniciales_admin_v1(text,text)');
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit
 OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid AND NOT (
  (dbid=db AND classid='pg_catalog.pg_namespace'::regclass AND objid=ns AND deptype='a')
  OR (dbid=db AND classid='pg_catalog.pg_proc'::regclass AND objid=funcion AND deptype='a')
  OR (dbid=0 AND classid='pg_catalog.pg_database'::regclass AND objid=db AND deptype='a')))
 OR pg_catalog.has_schema_privilege(l.oid,ns,'CREATE') OR pg_catalog.has_database_privilege(l.oid,db,'CREATE,TEMP')
 THEN RAISE EXCEPTION 'AUT39: PARO clave=operador actual=no_acreditado esperado=LOGIN_exclusivo_minimo' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_autorizacion.config_fuentes_iniciales_admin_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR pg_catalog.clock_timestamp()<cfg.vigente_desde OR pg_catalog.clock_timestamp()>=cfg.vigente_hasta
 THEN RAISE EXCEPTION 'AUT39: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobacion_externa_vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.exigir_operador_fuentes_iniciales_admin_v1() FROM PUBLIC;

-- Fachadas propietarias: no consulta ninguna tabla CA/IS desde AUT.
CREATE FUNCTION vec_autorizacion.preimagen_orquestada_fuentes_admin_v1(p jsonb,material text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE ca jsonb;identidad jsonb;
BEGIN
 ca:=vec_contexto_actor_v1.preimagen_fuentes_iniciales_admin_v1(p);
 identidad:=vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(p,material);
 RETURN pg_catalog.jsonb_build_object('esquema','vec.admin.preimagen-fuentes.v1','ca',ca,'is',identidad);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.preimagen_orquestada_fuentes_admin_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.aplicar_efecto_fuentes_iniciales_admin_v1(p_plan_canonico text,p_huella_aprobada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE cfg vec_autorizacion.config_fuentes_iniciales_admin_v1;p jsonb;sha text;pre jsonb;pre_sha text;cfg_sha text;
 previo vec_autorizacion.fuentes_iniciales_admin_v1;is_recibo jsonb;ca_recibo jsonb;fuente jsonb;fuente_sha text;
 ahora timestamptz;evento text;correlacion text;recibo_ref text;aud record;resultado jsonb;persona jsonb;operaciones text[]:=ARRAY[]::text[];op text;
BEGIN
 cfg:=vec_autorizacion.exigir_operador_fuentes_iniciales_admin_v1();
 IF p_plan_canonico IS NULL OR pg_catalog.octet_length(p_plan_canonico) NOT BETWEEN 1 AND 65536
 OR p_huella_aprobada IS DISTINCT FROM cfg.plan_sha256
 THEN RAISE EXCEPTION 'AUT39: PARO clave=aprobacion actual=divergente esperado=plan_externo_aprobado' USING ERRCODE='42501'; END IF;
 p:=p_plan_canonico::jsonb;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_plan_canonico,'UTF8')),'hex');
 IF sha IS DISTINCT FROM cfg.plan_sha256 OR p_plan_canonico IS DISTINCT FROM vec_autorizacion.canon_fuentes_iniciales_admin_v1(p,'plan')
 OR p->>'version'<>'1' OR p->>'entorno' IS DISTINCT FROM cfg.entorno OR p->>'alcance_fuente' IS DISTINCT FROM cfg.alcance_fuente
 OR (p->>'operacion_ref') !~ '^pfi_[A-Za-z0-9_-]{22,124}$'
 OR (p#>>'{personas,0,persona_ref}') COLLATE "C">=(p#>>'{personas,1,persona_ref}') COLLATE "C"
 OR p#>>'{organizacion,version_esperada}'<>'0'
 OR p#>>'{procedencia,version}'<>'1' OR p#>>'{fuente_hmac,version}'<>'1'
 OR (p#>>'{procedencia,huella_sha256}') !~ '^[0-9a-f]{64}$' OR (p#>>'{fuente_hmac,huella_sha256}') !~ '^[0-9a-f]{64}$'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(cfg.material_hmac_canonico,'UTF8')),'hex') IS DISTINCT FROM p#>>'{fuente_hmac,huella_sha256}'
 OR (p->>'preparado_en')::timestamptz>pg_catalog.clock_timestamp()
 OR (p->>'caduca_en')::timestamptz<=pg_catalog.clock_timestamp()
 OR (p#>>'{politica_admin,maxima_edad_revocacion_segundos}')::numeric NOT BETWEEN 1 AND 2147483647
 THEN RAISE EXCEPTION 'AUT39: PARO clave=plan actual=invalido_o_divergente esperado=canon_fuentes_sinteticas_aprobadas' USING ERRCODE='22023'; END IF;
 FOR persona IN SELECT e.value FROM pg_catalog.jsonb_array_elements(p->'personas') e LOOP
  IF persona->>'version_esperada'<>'0' OR persona#>>'{fuente_titularidad,version}'<>'1'
  OR (persona#>>'{fuente_titularidad,huella_sha256}') !~ '^[0-9a-f]{64}$'
  OR (persona->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
  THEN RAISE EXCEPTION 'AUT39: PARO clave=persona actual=divergente esperado=version_inicial_fuente_vigente' USING ERRCODE='22023'; END IF;
  FOREACH op IN ARRAY ARRAY[persona->>'operacion_cuenta_ordinaria_ref',persona->>'operacion_cuenta_privilegiada_ref'] LOOP
   IF op=ANY(operaciones) THEN RAISE EXCEPTION 'AUT39: PARO clave=operacion_cuenta actual=repetida esperado=cuatro_distintas' USING ERRCODE='22023'; END IF;
   operaciones:=pg_catalog.array_append(operaciones,op);
  END LOOP;
 END LOOP;
 IF (p#>>'{organizacion,vigente_hasta}')::timestamptz<(p->>'caduca_en')::timestamptz
 OR (p#>>'{politica_admin,vigente_hasta}')::timestamptz<(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'AUT39: PARO clave=vigencias actual=cortas esperado=cubren_plan' USING ERRCODE='22023'; END IF;
 cfg_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((pg_catalog.to_jsonb(cfg)-'material_hmac_canonico')::text,'UTF8')),'hex');
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:fuentes-iniciales:v1',0));
 SELECT * INTO previo FROM vec_autorizacion.fuentes_iniciales_admin_v1 WHERE operacion_ref=p->>'operacion_ref';
 IF FOUND THEN
  IF previo.plan_sha256<>sha OR previo.operador_login<>session_user OR previo.configuracion_sha256<>cfg_sha
  OR previo.aprobacion_ref<>cfg.aprobacion_ref
  THEN RAISE EXCEPTION 'AUT39: PARO clave=replay actual=divergente esperado=misma_operacion_y_aprobacion' USING ERRCODE='40001'; END IF;
  is_recibo:=vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(p->>'operacion_ref',sha,cfg.aprobacion_ref);
  IF is_recibo IS DISTINCT FROM previo.recibo->'is'
  THEN RAISE EXCEPTION 'AUT39: PARO clave=recibo_IS actual=divergente esperado=original_propietario' USING ERRCODE='55000'; END IF;
  -- Una espera puede agotar la ventana aunque el recibo histórico exista.
  PERFORM vec_autorizacion.exigir_operador_fuentes_iniciales_admin_v1();
  IF pg_catalog.clock_timestamp()>=(p->>'caduca_en')::timestamptz
  THEN RAISE EXCEPTION 'AUT39: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
  RETURN pg_catalog.jsonb_build_object('recibo',previo.recibo,'replay',true);
 END IF;
 pre:=vec_autorizacion.preimagen_orquestada_fuentes_admin_v1(p,cfg.material_hmac_canonico);
 pre_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pre::text,'UTF8')),'hex');
 IF pre_sha IS DISTINCT FROM cfg.preimagen_sha256
 THEN RAISE EXCEPTION 'AUT39: PARO clave=preimagen actual=% esperado=%',pre_sha,cfg.preimagen_sha256 USING ERRCODE='40001'; END IF;
 is_recibo:=vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p,cfg.material_hmac_canonico,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((pre->'is')::text,'UTF8')),'hex'),p->>'operacion_ref',sha,cfg.aprobacion_ref);
 ca_recibo:=vec_contexto_actor_v1.confirmar_fuentes_iniciales_admin_v1(p,is_recibo,
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to((pre->'ca')::text,'UTF8')),'hex'),p->>'operacion_ref',sha,cfg.aprobacion_ref);
 fuente:=pg_catalog.jsonb_build_object('esquema','vec.admin.fuentes-confirmadas.v1','ca',ca_recibo,'is',is_recibo);
 fuente_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente::text,'UTF8')),'hex');
 evento:='evento_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 correlacion:='correlacion_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 recibo_ref:='recibo_fuentes:'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 SELECT * INTO aud FROM vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(pg_catalog.jsonb_build_object(
  'tipo_registro','provision_fuentes_iniciales_admin','evento_ref',evento,'operador_login',session_user::text,
  'plan_ref',p->>'operacion_ref','plan_sha256',sha,'preimagen_sha256',pre_sha,'configuracion_sha256',cfg_sha,
  'aprobacion_ref',cfg.aprobacion_ref,'alcance_fuente',cfg.alcance_fuente,'accion','provisionar_fuentes_iniciales_admin_v1',
  'recurso_ref',p->>'operacion_ref','resultado','permitido','motivo_ref',cfg.motivo_ref,'proceso',cfg.proceso,
  'canal','operacion_tecnica_privada','finalidad_ref','provision_fuentes_iniciales_admin','correlacion_ref',correlacion,
  'fuente_ref',recibo_ref,'fuente_sha256',fuente_sha));
 PERFORM vec_autorizacion.exigir_operador_fuentes_iniciales_admin_v1();
 IF pg_catalog.clock_timestamp()>=(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'AUT39: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
 ahora:=pg_catalog.clock_timestamp();
 resultado:=fuente||pg_catalog.jsonb_build_object('recibo_ref',recibo_ref,'operacion_ref',p->>'operacion_ref',
  'plan_sha256',sha,'preimagen_sha256',pre_sha,'configuracion_sha256',cfg_sha,'operador_login',session_user::text,
  'aprobacion_ref',cfg.aprobacion_ref,'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,
  'auditoria_huella_sha256',aud.huella_sha256,'registrada_en',ahora);
 INSERT INTO vec_autorizacion.fuentes_iniciales_admin_v1 VALUES(p->>'operacion_ref',sha,pg_catalog.convert_to(p_plan_canonico,'UTF8'),
  pre_sha,cfg_sha,session_user::name,cfg.aprobacion_ref,aud.auditoria_ref,recibo_ref,resultado,ahora);
 RETURN pg_catalog.jsonb_build_object('recibo',resultado,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.aplicar_efecto_fuentes_iniciales_admin_v1(text,text) FROM PUBLIC;

-- El motor observa la solicitud; no atribuye al CLI una procedencia no probada.
-- Error controlado: rollback del subbloque, append común y envelope que el
-- consumidor debe confirmar con COMMIT antes de informar su resultado.
CREATE FUNCTION vec_autorizacion.provisionar_fuentes_iniciales_admin_v1(p_plan_canonico text,p_huella_aprobada text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE respuesta jsonb;estado text:='permitido';motivo text;codigo text;aud record;
 solicitud text:='solicitud_fuentes:'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 evento text:='evento_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 correlacion text:='correlacion_'||pg_catalog.replace(pg_catalog.gen_random_uuid()::text,'-','');
 solicitud_sha text;
BEGIN
 solicitud_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.jsonb_build_object('plan',p_plan_canonico,'huella_aprobada',p_huella_aprobada)::text,'UTF8')),'hex');
 BEGIN
  respuesta:=vec_autorizacion.aplicar_efecto_fuentes_iniciales_admin_v1(p_plan_canonico,p_huella_aprobada);
  motivo:=CASE WHEN (respuesta->>'replay')::boolean THEN 'fuentes_replay' ELSE 'fuentes_registradas' END;
 EXCEPTION WHEN OTHERS THEN
  -- Las variables sobreviven a EXCEPTION; ninguna referencia revertida sale.
  respuesta:=NULL;
  codigo:=SQLSTATE;
  estado:=CASE WHEN codigo IN('42501','22023','40001','23505','25000') THEN 'denegado' ELSE 'error' END;
  motivo:=CASE WHEN estado='denegado' THEN 'fuentes_denegadas' ELSE 'fuentes_error' END;
  codigo:=CASE WHEN estado='denegado' THEN 'fuentes_rechazadas' ELSE 'fuentes_no_disponibles' END;
 END;
 SELECT * INTO aud FROM vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(pg_catalog.jsonb_build_object(
  'tipo_registro','intento_fuentes_iniciales_admin','evento_ref',evento,'operador_login',session_user::text,
  'solicitud_sha256',solicitud_sha,'accion','provisionar_fuentes_iniciales_admin_v1','recurso_ref',solicitud,
  'resultado',estado,'motivo_ref',motivo,'proceso','postgresql','canal','operacion_tecnica_privada',
  'finalidad_ref','provision_fuentes_iniciales_admin','correlacion_ref',correlacion));
 RETURN pg_catalog.jsonb_build_object('estado',estado,'codigo',codigo,'recibo',respuesta->'recibo',
  'replay',COALESCE((respuesta->>'replay')::boolean,false),'auditoria_intento',pg_catalog.jsonb_build_object(
   'auditoria_ref',aud.auditoria_ref,'secuencia',aud.secuencia,'huella_sha256',aud.huella_sha256,
   'correlacion_ref',aud.correlacion_ref,'registrada_en',aud.registrada_en));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.provisionar_fuentes_iniciales_admin_v1(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_admin_fuentes_iniciales_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion.provisionar_fuentes_iniciales_admin_v1(text,text) TO vec_admin_fuentes_iniciales_ejecutor;
COMMIT;
