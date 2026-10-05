\set ON_ERROR_STOP on
-- Personal33: una unidad inicial declarada sintética, sin perfiles ni OH11.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:migracion:000033',0));
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal33: PARO clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'Personal33: PARO clave=PG actual=% esperado=180000..189999',current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_personal.org_nodo_historia') IS NULL
 OR to_regclass('vec_personal.control_unidad_bootstrap_admin_v1') IS NULL
 OR to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(jsonb)') IS NULL
 THEN RAISE EXCEPTION 'Personal33: PARO clave=dependencias actual=ausente esperado=Personal10_31_AD176' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid='vec_personal.org_nodo_historia'::regclass
  AND tgname='barrera_unidad_bootstrap_admin_v1' AND tgenabled='O' AND NOT tgisinternal
  AND tgfoid=to_regprocedure('vec_personal.avanzar_barrera_unidad_bootstrap_admin_v1()'))
 THEN RAISE EXCEPTION 'Personal33: PARO clave=trigger_Personal31 actual=ausente_o_inactivo esperado=INSERT_normal_avanza_generacion' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_personal.unidad_inicial_admin_v1') IS NOT NULL OR to_regrole('vec_personal_unidad_inicial_ejecutor') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal33: PARO clave=instalacion actual=presente esperado=ausente' USING ERRCODE='55000'; END IF;
 CREATE ROLE vec_personal_unidad_inicial_ejecutor NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_personal_unidad_inicial_ejecutor',current_database());
END $pre$;
SET LOCAL ROLE vec_personal_propietario;

-- Canon compatible encoding/json; sólo tipos y campos de los DTO cerrados.
CREATE FUNCTION vec_personal.canon_unidad_inicial_admin_v1(v jsonb,tipo text)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE campos text[];tipos text[];i int;salida text:='';valor text;
BEGIN
 CASE tipo
 WHEN 'plan' THEN
  campos:=ARRAY['version','operacion_ref','preparado_en','caduca_en','entorno','alcance_fuente','unidad','acto_tecnico_ref','fuente'];
  tipos:=ARRAY['n','s','t','t','s','s','unidad','s','evidencia'];
 WHEN 'fuente' THEN
  campos:=ARRAY['version','referencia','entorno','alcance_fuente','unidad','acto_tecnico_ref'];
  tipos:=ARRAY['n','s','s','s','unidad','s'];
 WHEN 'unidad' THEN
  campos:=ARRAY['nodo_ref','organizacion_ref','unidad_ref','clase','denominacion','catalogo_ref','catalogo_version','catalogo_revision','catalogo_entrada_clave','revision_esperada','vigente_desde','vigente_hasta'];
  tipos:=ARRAY['s','s','s','s','s','s','n','n','s','n','d','d'];
 WHEN 'evidencia' THEN campos:=ARRAY['referencia','version','huella_sha256'];tipos:=ARRAY['s','n','s'];
 WHEN 's' THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'string' OR length(v#>>'{}') NOT BETWEEN 1 AND 300 OR v#>>'{}' ~ '[[:cntrl:]]'
  THEN RAISE EXCEPTION 'Personal33: PARO clave=cadena actual=invalida esperado=string_acotado' USING ERRCODE='22023'; END IF;
  valor:=to_jsonb(v#>>'{}')::text;
  RETURN replace(replace(replace(replace(replace(valor,'&',E'\\u0026'),'<',E'\\u003c'),'>',E'\\u003e'),chr(8232),E'\\u2028'),chr(8233),E'\\u2029');
 WHEN 'n' THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'number' OR v#>>'{}' !~ '^(0|[1-9][0-9]{0,9})$' OR (v#>>'{}')::numeric>2147483647
  THEN RAISE EXCEPTION 'Personal33: PARO clave=numero actual=invalido esperado=int_no_negativo' USING ERRCODE='22023'; END IF;
  RETURN v#>>'{}';
 WHEN 't' THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'string' OR v#>>'{}' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
  OR to_char((v#>>'{}')::timestamptz AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') IS DISTINCT FROM v#>>'{}'
  THEN RAISE EXCEPTION 'Personal33: PARO clave=instante actual=invalido esperado=UTC_segundos' USING ERRCODE='22023'; END IF;
  RETURN to_jsonb(v#>>'{}')::text;
 WHEN 'd' THEN
  IF jsonb_typeof(v) IS DISTINCT FROM 'string' OR v#>>'{}' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
  OR to_char((v#>>'{}')::date,'YYYY-MM-DD') IS DISTINCT FROM v#>>'{}'
  THEN RAISE EXCEPTION 'Personal33: PARO clave=fecha actual=invalida esperado=dia_calendario' USING ERRCODE='22023'; END IF;
  RETURN to_jsonb(v#>>'{}')::text;
 ELSE RAISE EXCEPTION 'Personal33: PARO clave=tipo actual=desconocido esperado=ABI_cerrado' USING ERRCODE='22023';
 END CASE;
 IF jsonb_typeof(v) IS DISTINCT FROM 'object' OR NOT v ?& campos
 OR (SELECT count(*) FROM jsonb_object_keys(v))<>cardinality(campos)
 THEN RAISE EXCEPTION 'Personal33: PARO clave=objeto actual=divergente esperado=campos_exactos' USING ERRCODE='22023'; END IF;
 FOR i IN 1..cardinality(campos) LOOP
  valor:=vec_personal.canon_unidad_inicial_admin_v1(v->campos[i],tipos[i]);
  IF salida<>'' THEN salida:=salida||','; END IF;
  salida:=salida||to_jsonb(campos[i])::text||':'||valor;
 END LOOP;
 RETURN '{'||salida||'}';
END $f$;
REVOKE ALL ON FUNCTION vec_personal.canon_unidad_inicial_admin_v1(jsonb,text) FROM PUBLIC,vec_personal_ejecutor;

CREATE TABLE vec_personal.config_unidad_inicial_admin_v1(
 login_nombre name PRIMARY KEY,
 entorno text NOT NULL CHECK(entorno='desarrollo'),alcance_fuente text NOT NULL CHECK(alcance_fuente='sintetico_declarado'),
 plan_sha256 text NOT NULL UNIQUE CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
 aprobacion_sha256 text NOT NULL CHECK(aprobacion_sha256 ~ '^[0-9a-f]{64}$'),
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
CREATE TABLE vec_personal.unidad_inicial_admin_v1(
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 operacion_ref text NOT NULL UNIQUE CHECK(operacion_ref ~ '^pui_[A-Za-z0-9_-]{22,124}$'),
 plan_sha256 text NOT NULL UNIQUE,plan_canonico bytea NOT NULL CHECK(plan_sha256=encode(pg_catalog.sha256(plan_canonico),'hex')),
 fuente_sha256 text NOT NULL,fuente_canonica bytea NOT NULL CHECK(fuente_sha256=encode(pg_catalog.sha256(fuente_canonica),'hex')),
 preimagen_sha256 text NOT NULL,configuracion_sha256 text NOT NULL,
 alcance_fuente text NOT NULL CHECK(alcance_fuente='sintetico_declarado'),
 operador_login name NOT NULL,aprobacion_ref text NOT NULL,
 nodo_ref uuid NOT NULL,revision integer NOT NULL CHECK(revision=1),
 recibo_ref text NOT NULL UNIQUE,recibo_sha256 text NOT NULL,recibo jsonb NOT NULL,
 auditoria_ref text NOT NULL UNIQUE,registrada_en timestamptz NOT NULL CHECK(isfinite(registrada_en)),
 FOREIGN KEY(nodo_ref,revision) REFERENCES vec_personal.org_nodo_historia(nodo_ref,revision)
);
DO $tablas$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['config_unidad_inicial_admin_v1','unidad_inicial_admin_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_personal.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_personal.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_personal.%I TO vec_personal_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',t,'vec_personal_propietario','vec_personal_propietario');
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_personal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_organizacion_v1()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor,vec_personal_unidad_inicial_ejecutor',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_personal.%I FROM PUBLIC,vec_personal_ejecutor,vec_personal_unidad_inicial_ejecutor',t);
 END LOOP;
END $tablas$;

CREATE FUNCTION vec_personal.exigir_operador_unidad_inicial_admin_v1()
RETURNS vec_personal.config_unidad_inicial_admin_v1 LANGUAGE plpgsql SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE l record;g record;cfg vec_personal.config_unidad_inicial_admin_v1;ns oid;db oid;funcion oid;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC' OR current_setting('role')<>'none'
 THEN RAISE EXCEPTION 'Personal33: PARO clave=transaccion actual=incompatible esperado=SERIALIZABLE_RW_UTC_sin_SETROLE' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_roles WHERE rolname='vec_personal_unidad_inicial_ejecutor';
 ns:=to_regnamespace('vec_personal');SELECT oid INTO db FROM pg_database WHERE datname=current_database();
 funcion:=to_regprocedure('vec_personal.inicializar_unidad_sintetica_admin_v1(text,text,text)');
 IF l.oid IS NULL OR g.oid IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit
 OR l.rolsuper OR l.rolcreaterole OR l.rolcreatedb OR l.rolreplication OR l.rolbypassrls OR l.rolconfig IS NOT NULL
 OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreaterole OR g.rolcreatedb OR g.rolreplication OR g.rolbypassrls OR g.rolconfig IS NOT NULL
 OR (SELECT count(*) FROM pg_auth_members WHERE member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=l.oid AND roleid=g.oid AND NOT admin_option AND inherit_option AND NOT set_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=g.oid)
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN(l.oid,g.oid))
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=l.oid)
 OR EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_catalog.pg_authid'::regclass AND refobjid=g.oid AND NOT(
  (dbid=db AND classid='pg_catalog.pg_namespace'::regclass AND objid=ns AND deptype='a')
  OR (dbid=db AND classid='pg_catalog.pg_proc'::regclass AND objid=funcion AND deptype='a')
  OR (dbid=0 AND classid='pg_catalog.pg_database'::regclass AND objid=db AND deptype='a')))
 OR NOT EXISTS(SELECT 1 FROM pg_namespace n,LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid=ns AND a.grantee=g.oid AND a.privilege_type='USAGE' AND NOT a.is_grantable)
 OR EXISTS(SELECT 1 FROM pg_namespace n,LATERAL aclexplode(COALESCE(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid=ns AND a.grantee=g.oid AND (a.privilege_type<>'USAGE' OR a.is_grantable))
 OR NOT EXISTS(SELECT 1 FROM pg_database d,LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) a WHERE d.oid=db AND a.grantee=g.oid AND a.privilege_type='CONNECT' AND NOT a.is_grantable)
 OR EXISTS(SELECT 1 FROM pg_database d,LATERAL aclexplode(COALESCE(d.datacl,acldefault('d',d.datdba))) a WHERE d.oid=db AND a.grantee=g.oid AND (a.privilege_type<>'CONNECT' OR a.is_grantable))
 OR NOT EXISTS(SELECT 1 FROM pg_proc f,LATERAL aclexplode(COALESCE(f.proacl,acldefault('f',f.proowner))) a WHERE f.oid=funcion AND a.grantee=g.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 OR EXISTS(SELECT 1 FROM pg_proc f,LATERAL aclexplode(COALESCE(f.proacl,acldefault('f',f.proowner))) a WHERE f.oid=funcion AND a.grantee=g.oid AND (a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR has_schema_privilege(l.oid,ns,'CREATE') OR has_database_privilege(l.oid,db,'CREATE,TEMP')
 THEN RAISE EXCEPTION 'Personal33: PARO clave=operador actual=no_acreditado esperado=LOGIN_exclusivo_minimo' USING ERRCODE='42501'; END IF;
 SELECT * INTO cfg FROM vec_personal.config_unidad_inicial_admin_v1 WHERE login_nombre=session_user FOR SHARE;
 IF NOT FOUND OR clock_timestamp()<cfg.vigente_desde OR clock_timestamp()>=cfg.vigente_hasta
 THEN RAISE EXCEPTION 'Personal33: PARO clave=configuracion actual=ausente_o_caducada esperado=aprobacion_externa_vigente' USING ERRCODE='42501'; END IF;
 RETURN cfg;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.exigir_operador_unidad_inicial_admin_v1() FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.validar_unidad_inicial_admin_v1(p jsonb)
RETURNS void LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE u jsonb:=p->'unidad';
BEGIN
 PERFORM vec_personal.canon_unidad_inicial_admin_v1(p,'plan');
 IF p->>'version' IS DISTINCT FROM '1' OR p->>'entorno' IS DISTINCT FROM 'desarrollo' OR p->>'alcance_fuente' IS DISTINCT FROM 'sintetico_declarado'
 OR p->>'operacion_ref' !~ '^pui_[A-Za-z0-9_-]{22,124}$'
 OR p->>'acto_tecnico_ref' !~ '^acto_tecnico:[a-z0-9_:-]{1,147}$'
 OR p#>>'{fuente,referencia}' !~ '^[a-z][a-z0-9_:-]{2,127}$' OR p#>>'{fuente,version}' IS DISTINCT FROM '1'
 OR p#>>'{fuente,huella_sha256}' !~ '^[0-9a-f]{64}$' OR p#>>'{fuente,huella_sha256}'=repeat('0',64)
 OR (p->>'preparado_en')::timestamptz>clock_timestamp() OR (p->>'caduca_en')::timestamptz<=clock_timestamp()
 OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 OR u->>'nodo_ref' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
 OR u->>'organizacion_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$' OR u->>'unidad_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR u->>'clase' NOT IN('delegacion','centro','puesto_responsabilidad')
 OR u->>'denominacion' IS DISTINCT FROM btrim(u->>'denominacion',chr(9)||chr(10)||chr(11)||chr(12)||chr(13)||chr(32)||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288))
 OR u->>'catalogo_ref' IS DISTINCT FROM 'estructura-organizativa-dipgra'
 OR u->>'catalogo_version' IS DISTINCT FROM '1' OR u->>'catalogo_revision' IS DISTINCT FROM '1'
 OR u->>'catalogo_entrada_clave' !~ '^[a-z][a-z0-9_:-]{2,159}$' OR u->>'revision_esperada' IS DISTINCT FROM '0'
 OR (u->>'vigente_desde')::date>((p->>'preparado_en')::timestamptz AT TIME ZONE 'UTC')::date
 OR (u->>'vigente_hasta')::date<=(u->>'vigente_desde')::date
 OR ((u->>'vigente_hasta')::date::timestamp AT TIME ZONE 'UTC')<(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'Personal33: PARO clave=plan actual=divergente esperado=unidad_sintetica_inicial_aprobada' USING ERRCODE='22023'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.validar_unidad_inicial_admin_v1(jsonb) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.preimagen_unidad_inicial_admin_v1(p jsonb)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE u jsonb:=p->'unidad';gen bigint;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'Personal33: PARO clave=transaccion actual=incompatible esperado=SERIALIZABLE_RW' USING ERRCODE='25000'; END IF;
 PERFORM vec_personal.validar_unidad_inicial_admin_v1(p);
 SELECT generacion INTO STRICT gen FROM vec_personal.control_unidad_bootstrap_admin_v1 WHERE singleton FOR UPDATE;
 IF EXISTS(SELECT 1 FROM vec_personal.unidad_inicial_admin_v1)
 OR EXISTS(SELECT 1 FROM vec_personal.org_nodo_historia h WHERE h.nodo_ref=(u->>'nodo_ref')::uuid
  OR h.unidad_ref=u->>'unidad_ref' OR (h.catalogo_ref=u->>'catalogo_ref' AND h.catalogo_entrada_clave=u->>'catalogo_entrada_clave'))
 THEN RAISE EXCEPTION 'Personal33: PARO clave=CAS_inicial actual=historia_o_recibo_existente esperado=ausencia_total' USING ERRCODE='40001'; END IF;
 RETURN jsonb_build_object('esquema','vec.personal.unidad-inicial.preimagen.v1','operacion_ref',p->>'operacion_ref',
  'nodo_ref',u->>'nodo_ref','organizacion_ref',u->>'organizacion_ref','unidad_ref',u->>'unidad_ref',
  'catalogo_entrada_clave',u->>'catalogo_entrada_clave','generacion',gen,'ausente',true);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.preimagen_unidad_inicial_admin_v1(jsonb) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.aplicar_efecto_unidad_inicial_admin_v1(plan_canonico text,sha_aprobado text,fuente_canonica text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE cfg vec_personal.config_unidad_inicial_admin_v1;p jsonb;f jsonb;u jsonb;plan_sha text;fuente_sha text;cfg_sha text;pre jsonb;pre_sha text;
 previo vec_personal.unidad_inicial_admin_v1;r vec_personal.org_nodo_historia;recibo jsonb;recibo_base jsonb;recibo_ref_nuevo text;recibo_sha text;aud record;evento text;correlacion text;ahora timestamptz;
BEGIN
 cfg:=vec_personal.exigir_operador_unidad_inicial_admin_v1();
 IF plan_canonico IS NULL OR octet_length(plan_canonico) NOT BETWEEN 1 AND 65536 OR fuente_canonica IS NULL OR octet_length(fuente_canonica) NOT BETWEEN 1 AND 65536
 OR sha_aprobado IS DISTINCT FROM cfg.plan_sha256
 THEN RAISE EXCEPTION 'Personal33: PARO clave=aprobacion actual=divergente esperado=plan_y_fuente_externos_aprobados' USING ERRCODE='42501'; END IF;
 p:=plan_canonico::jsonb;f:=fuente_canonica::jsonb;u:=p->'unidad';
 plan_sha:=encode(pg_catalog.sha256(pg_catalog.convert_to(plan_canonico,'UTF8')),'hex');fuente_sha:=encode(pg_catalog.sha256(pg_catalog.convert_to(fuente_canonica,'UTF8')),'hex');
 PERFORM vec_personal.validar_unidad_inicial_admin_v1(p);
 IF plan_canonico IS DISTINCT FROM vec_personal.canon_unidad_inicial_admin_v1(p,'plan')
 OR fuente_canonica IS DISTINCT FROM vec_personal.canon_unidad_inicial_admin_v1(f,'fuente')
 OR plan_sha IS DISTINCT FROM cfg.plan_sha256 OR fuente_sha IS DISTINCT FROM cfg.fuente_sha256 OR fuente_sha IS DISTINCT FROM p#>>'{fuente,huella_sha256}'
 OR f->>'version' IS DISTINCT FROM '1' OR f->>'referencia' IS DISTINCT FROM p#>>'{fuente,referencia}'
 OR f->>'entorno' IS DISTINCT FROM cfg.entorno OR f->>'alcance_fuente' IS DISTINCT FROM cfg.alcance_fuente
 OR f->'unidad' IS DISTINCT FROM u OR f->>'acto_tecnico_ref' IS DISTINCT FROM p->>'acto_tecnico_ref'
 THEN RAISE EXCEPTION 'Personal33: PARO clave=fuente actual=divergente esperado=canon_y_datos_reales_del_plan_aprobado' USING ERRCODE='22023'; END IF;
 cfg_sha:=encode(pg_catalog.sha256(pg_catalog.convert_to(to_jsonb(cfg)::text,'UTF8')),'hex');
 PERFORM generacion FROM vec_personal.control_unidad_bootstrap_admin_v1 WHERE singleton FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'Personal33: PARO clave=barrera actual=ausente esperado=Personal31' USING ERRCODE='55000'; END IF;
 SELECT * INTO previo FROM vec_personal.unidad_inicial_admin_v1 WHERE singleton;
 IF FOUND THEN
  IF previo.operacion_ref IS DISTINCT FROM p->>'operacion_ref' OR previo.plan_sha256 IS DISTINCT FROM plan_sha
  OR previo.plan_canonico IS DISTINCT FROM pg_catalog.convert_to(plan_canonico,'UTF8') OR previo.fuente_canonica IS DISTINCT FROM pg_catalog.convert_to(fuente_canonica,'UTF8')
  OR previo.operador_login IS DISTINCT FROM session_user::name OR previo.configuracion_sha256 IS DISTINCT FROM cfg_sha
  OR previo.aprobacion_ref IS DISTINCT FROM cfg.aprobacion_ref
  THEN RAISE EXCEPTION 'Personal33: PARO clave=replay actual=divergente esperado=misma_operacion_y_aprobacion_originales' USING ERRCODE='40001'; END IF;
  -- Revalidar tras esperar la barrera: el replay conserva el recibo histórico,
  -- pero sólo se entrega con configuración y plan todavía vigentes.
  PERFORM vec_personal.exigir_operador_unidad_inicial_admin_v1();
  IF clock_timestamp()>=(p->>'caduca_en')::timestamptz
  THEN RAISE EXCEPTION 'Personal33: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
  RETURN jsonb_build_object('recibo',previo.recibo,'replay',true);
 END IF;
 pre:=vec_personal.preimagen_unidad_inicial_admin_v1(p);pre_sha:=encode(pg_catalog.sha256(pg_catalog.convert_to(pre::text,'UTF8')),'hex');
 IF pre_sha IS DISTINCT FROM cfg.preimagen_sha256
 THEN RAISE EXCEPTION 'Personal33: PARO clave=preimagen actual=% esperado=%',pre_sha,cfg.preimagen_sha256 USING ERRCODE='40001'; END IF;
 ahora:=clock_timestamp();
 INSERT INTO vec_personal.org_nodo_historia(nodo_ref,revision,organismo_ref,unidad_ref,clase,catalogo_ref,catalogo_version,catalogo_revision,catalogo_entrada_clave,denominacion,centro_padre_ref,retirado,vigente_desde,vigente_hasta,conocido_desde,fuente_ref,acto_ref,huella_fuente_sha256)
 VALUES((u->>'nodo_ref')::uuid,1,u->>'organizacion_ref',u->>'unidad_ref',u->>'clase',u->>'catalogo_ref',1,1,u->>'catalogo_entrada_clave',u->>'denominacion',NULL,false,(u->>'vigente_desde')::date,(u->>'vigente_hasta')::date,ahora,p#>>'{fuente,referencia}',p->>'acto_tecnico_ref',fuente_sha)
 RETURNING * INTO r;
 recibo_ref_nuevo:='recibo_unidad:'||replace(pg_catalog.gen_random_uuid()::text,'-','');
 recibo_base:=jsonb_build_object('esquema','vec.personal.unidad-inicial.v1','version',1,'recibo_ref',recibo_ref_nuevo,'operacion_ref',p->>'operacion_ref',
  'plan_sha256',plan_sha,'fuente_ref',p#>>'{fuente,referencia}','fuente_version',1,'fuente_sha256',fuente_sha,
  'preimagen_sha256',pre_sha,'configuracion_sha256',cfg_sha,'aprobacion_ref',cfg.aprobacion_ref,'alcance_fuente',cfg.alcance_fuente,
  'registrada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'unidad',to_jsonb(r));
 recibo_sha:=encode(pg_catalog.sha256(pg_catalog.convert_to(recibo_base::text,'UTF8')),'hex');
 evento:='evento_'||replace(pg_catalog.gen_random_uuid()::text,'-','');correlacion:='correlacion_'||replace(pg_catalog.gen_random_uuid()::text,'-','');
 SELECT * INTO aud FROM vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(jsonb_build_object(
  'tipo_registro','unidad_inicial_personal','evento_ref',evento,'operador_login',session_user::text,
  'plan_ref',p->>'operacion_ref','plan_sha256',plan_sha,'preimagen_sha256',pre_sha,'configuracion_sha256',cfg_sha,
  'aprobacion_ref',cfg.aprobacion_ref,'alcance_fuente',cfg.alcance_fuente,'accion','inicializar_unidad_sintetica_admin_v1',
  'recurso_ref','unidad:'||r.nodo_ref::text,'resultado','permitido','motivo_ref','unidad_registrada','proceso','postgresql',
  'canal','operacion_tecnica_privada','finalidad_ref','inicializar_unidad_sintetica_admin','correlacion_ref',correlacion,
  'fuente_ref',p#>>'{fuente,referencia}','fuente_sha256',fuente_sha,'recibo_ref',recibo_ref_nuevo,'recibo_sha256',recibo_sha));
 PERFORM vec_personal.exigir_operador_unidad_inicial_admin_v1();
 IF clock_timestamp()>=(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'Personal33: PARO clave=vigencia_final actual=caducada esperado=plan_vigente' USING ERRCODE='42501'; END IF;
 recibo:=recibo_base||jsonb_build_object('recibo_sha256',recibo_sha,'auditoria_ref',aud.auditoria_ref,'auditoria_secuencia',aud.secuencia,'auditoria_huella_sha256',aud.huella_sha256,'auditoria_registrada_en',to_char(aud.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 INSERT INTO vec_personal.unidad_inicial_admin_v1(singleton,operacion_ref,plan_sha256,plan_canonico,fuente_sha256,fuente_canonica,preimagen_sha256,configuracion_sha256,alcance_fuente,operador_login,aprobacion_ref,nodo_ref,revision,recibo_ref,recibo_sha256,recibo,auditoria_ref,registrada_en)
 VALUES(true,p->>'operacion_ref',plan_sha,pg_catalog.convert_to(plan_canonico,'UTF8'),fuente_sha,pg_catalog.convert_to(fuente_canonica,'UTF8'),pre_sha,cfg_sha,cfg.alcance_fuente,session_user::name,cfg.aprobacion_ref,r.nodo_ref,1,recibo_ref_nuevo,recibo_sha,recibo,aud.auditoria_ref,ahora);
 RETURN jsonb_build_object('recibo',recibo,'replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.aplicar_efecto_unidad_inicial_admin_v1(text,text,text) FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.inicializar_unidad_sintetica_admin_v1(plan_canonico text,sha_aprobado text,fuente_canonica text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE respuesta jsonb;estado text:='permitido';codigo text;motivo text;aud record;solicitud_sha text;
 solicitud text:='solicitud_unidad:'||replace(pg_catalog.gen_random_uuid()::text,'-','');evento text:='evento_'||replace(pg_catalog.gen_random_uuid()::text,'-','');correlacion text:='correlacion_'||replace(pg_catalog.gen_random_uuid()::text,'-','');
BEGIN
 solicitud_sha:=encode(pg_catalog.sha256(pg_catalog.convert_to(jsonb_build_object('plan',plan_canonico,'sha_aprobado',sha_aprobado,'fuente',fuente_canonica)::text,'UTF8')),'hex');
 BEGIN
  respuesta:=vec_personal.aplicar_efecto_unidad_inicial_admin_v1(plan_canonico,sha_aprobado,fuente_canonica);
  motivo:=CASE WHEN (respuesta->>'replay')::boolean THEN 'unidad_replay' ELSE 'unidad_registrada' END;
 EXCEPTION WHEN OTHERS THEN
  respuesta:=NULL;codigo:=SQLSTATE;
  estado:=CASE WHEN codigo IN('42501','22023','40001','23505','25000') THEN 'denegado' ELSE 'error' END;
  motivo:=CASE WHEN estado='denegado' THEN 'unidad_denegada' ELSE 'unidad_error' END;
  codigo:=CASE WHEN estado='denegado' THEN 'unidad_rechazada' ELSE 'unidad_no_disponible' END;
 END;
 SELECT * INTO aud FROM vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(jsonb_build_object(
  'tipo_registro','intento_unidad_inicial_personal','evento_ref',evento,'operador_login',session_user::text,'solicitud_sha256',solicitud_sha,
  'accion','inicializar_unidad_sintetica_admin_v1','recurso_ref',solicitud,'resultado',estado,'motivo_ref',motivo,'proceso','postgresql',
  'canal','operacion_tecnica_privada','finalidad_ref','inicializar_unidad_sintetica_admin','correlacion_ref',correlacion));
 RETURN jsonb_build_object('estado',estado,'codigo',codigo,'recibo',respuesta->'recibo','replay',COALESCE((respuesta->>'replay')::boolean,false),
  'auditoria_intento',jsonb_build_object('auditoria_ref',aud.auditoria_ref,'secuencia',aud.secuencia,'huella_sha256',aud.huella_sha256,'correlacion_ref',aud.correlacion_ref,'registrada_en',to_char(aud.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.inicializar_unidad_sintetica_admin_v1(text,text,text) FROM PUBLIC,vec_personal_ejecutor;
GRANT USAGE ON SCHEMA vec_personal TO vec_personal_unidad_inicial_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.inicializar_unidad_sintetica_admin_v1(text,text,text) TO vec_personal_unidad_inicial_ejecutor;
DO $acl_final$
DECLARE f record;permitidos oid[];owner oid:='vec_personal_propietario'::regrole;ejecutor oid:='vec_personal_unidad_inicial_ejecutor'::regrole;
BEGIN
 FOR f IN SELECT p.oid,p.proname FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_personal' AND p.proname IN('canon_unidad_inicial_admin_v1','exigir_operador_unidad_inicial_admin_v1','validar_unidad_inicial_admin_v1','preimagen_unidad_inicial_admin_v1','aplicar_efecto_unidad_inicial_admin_v1','inicializar_unidad_sintetica_admin_v1') LOOP
  permitidos:=ARRAY[owner];
  IF f.proname='inicializar_unidad_sintetica_admin_v1' THEN permitidos:=array_append(permitidos,ejecutor); END IF;
  IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f.oid AND (a.grantee<>ALL(permitidos) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'Personal33: PARO clave=ACL_funcion actual=ampliada esperado=owner_y_fachada_unica' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl_final$;
COMMIT;
