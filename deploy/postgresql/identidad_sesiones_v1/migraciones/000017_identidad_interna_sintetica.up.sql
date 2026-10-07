\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'IS17: clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'IS17: clave=server_version_num actual=% esperado=180000..189999',current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea)') IS NULL THEN RAISE EXCEPTION 'IS17: clave=dependencia.vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea) actual=ausente esperado=presente' USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_identidad_sesiones_v1.claves_fuentes_admin_v1(jsonb,text[])') IS NULL THEN RAISE EXCEPTION 'IS17: dependencia IS15 ausente' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_identidad_sesiones_v1.identidad_interna_sintetica_v1') IS NOT NULL THEN RAISE EXCEPTION 'IS17: clave=vec_identidad_sesiones_v1.identidad_interna_sintetica_v1 actual=presente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

-- Únicamente AUT57, tras cotejar operador/configuración/aprobación externa,
-- invoca estas fachadas. No hay concesiones a runtime ni permiso en el JSON.
CREATE FUNCTION vec_identidad_sesiones_v1.validar_plan_identidad_interna_sintetica_v1(p jsonb)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE pe jsonb; f jsonb; ahora timestamptz:=clock_timestamp();
BEGIN
 IF p IS NULL OR octet_length(p::text)>65536 OR jsonb_path_exists(p,'$.** ? (@ == null)') THEN RAISE EXCEPTION 'IS17: null no admitido' USING ERRCODE='22023'; END IF;
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(p,ARRAY['version','operacion_ref','preparado_en','caduca_en','entorno','alcance_fuente','procedencia','organizacion','persona','fuente_hmac']) IS NOT TRUE
 OR p->>'version' IS DISTINCT FROM '1' OR p->>'entorno' IS DISTINCT FROM 'desarrollo'
 OR p->>'alcance_fuente' IS DISTINCT FROM 'sintetico_declarado'
 OR (p->>'operacion_ref' ~ '^piis_[A-Za-z0-9_-]{22,123}$') IS NOT TRUE
 THEN RAISE EXCEPTION 'IS17: plan cerrado inválido' USING ERRCODE='22023'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_each(p) WHERE jsonb_typeof(value)<>(CASE WHEN key='version' THEN 'number' WHEN key IN ('organizacion','procedencia','fuente_hmac','persona') THEN 'object' ELSE 'string' END))
 THEN RAISE EXCEPTION 'IS17: tipos inválidos' USING ERRCODE='22023'; END IF;
 IF (p->>'preparado_en') !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (p->>'caduca_en') !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (p->>'preparado_en')::timestamptz>ahora OR (p->>'caduca_en')::timestamptz<=ahora
 OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 OR (p->>'caduca_en')::timestamptz>(p->>'preparado_en')::timestamptz+interval '24 hours'
 THEN RAISE EXCEPTION 'IS17: ventana del plan inválida' USING ERRCODE='22023'; END IF;
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(p->'organizacion',ARRAY['organizacion_ref','version_esperada','procedencia_huella_sha256','vigente_hasta']) IS NOT TRUE
 OR p#>>'{organizacion,version_esperada}' !~ '^[1-9][0-9]{0,15}$'
 OR (p#>>'{organizacion,version_esperada}')::numeric>9007199254740991
 OR EXISTS(SELECT 1 FROM jsonb_each(p->'organizacion') WHERE jsonb_typeof(value)<>(CASE WHEN key='version_esperada' THEN 'number' ELSE 'string' END))
 OR p#>>'{organizacion,procedencia_huella_sha256}' !~ '^[0-9a-f]{64}$'
 OR p#>>'{organizacion,procedencia_huella_sha256}'=repeat('0',64)
 OR p#>>'{organizacion,organizacion_ref}' !~ '^org_[a-z0-9]{16,80}$'
 OR p#>>'{organizacion,vigente_hasta}' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (p#>>'{organizacion,vigente_hasta}')::timestamptz<(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'IS17: organización inválida' USING ERRCODE='22023'; END IF;
 FOR f IN SELECT p->'procedencia' UNION ALL SELECT p->'fuente_hmac' UNION ALL SELECT p#>'{persona,fuente_titularidad}' LOOP
  IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(f,ARRAY['referencia','version','huella_sha256']) IS NOT TRUE
  OR vec_identidad_sesiones_v1.referencia_valida(f->>'referencia','prc_') IS NOT TRUE
  OR octet_length(f->>'referencia')>128
  OR f->>'version' IS DISTINCT FROM '1' OR f->>'huella_sha256' !~ '^[0-9a-f]{64}$'
  OR f->>'huella_sha256'=repeat('0',64)
  OR EXISTS(SELECT 1 FROM jsonb_each(f) WHERE jsonb_typeof(value)<>(CASE WHEN key='version' THEN 'number' ELSE 'string' END))
  THEN RAISE EXCEPTION 'IS17: fuente inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 pe:=p->'persona';
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(pe,ARRAY['persona_ref','version_esperada','vigente_hasta','operacion_cuenta_ordinaria_ref','fuente_titularidad']) IS NOT TRUE
 OR vec_identidad_sesiones_v1.referencia_valida(pe->>'persona_ref','per_') IS NOT TRUE OR octet_length(pe->>'persona_ref')>128
 OR pe->>'version_esperada' IS DISTINCT FROM '0'
 OR EXISTS(SELECT 1 FROM jsonb_each(pe) WHERE jsonb_typeof(value)<>(CASE WHEN key='version_esperada' THEN 'number' WHEN key='fuente_titularidad' THEN 'object' ELSE 'string' END))
 OR pe->>'vigente_hasta' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (pe->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
 OR (pe->>'vigente_hasta')::timestamptz>(p#>>'{organizacion,vigente_hasta}')::timestamptz
 OR vec_identidad_sesiones_v1.referencia_valida(pe->>'operacion_cuenta_ordinaria_ref','opr_') IS NOT TRUE
 OR octet_length(pe->>'operacion_cuenta_ordinaria_ref')>128
 THEN RAISE EXCEPTION 'IS17: persona inválida' USING ERRCODE='22023'; END IF;
EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'IS17: plan inválido' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.validar_plan_identidad_interna_sintetica_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.validar_plan_identidad_interna_sintetica_v1(jsonb) TO vec_contexto_actor_v1_propietario,vec_autorizacion_propietario;

CREATE TABLE vec_identidad_sesiones_v1.identidad_interna_sintetica_v1(
 operacion_ref text PRIMARY KEY,plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL,plan jsonb NOT NULL,material_sha256 text NOT NULL,
 recibo jsonb NOT NULL,registrada_en timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE vec_identidad_sesiones_v1.titularidad_identidad_interna_sintetica_v1(
 cuenta_ref text PRIMARY KEY REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
 persona_ref text NOT NULL CHECK(vec_identidad_sesiones_v1.referencia_valida(persona_ref,'per_') IS TRUE),
 version numeric(20,0) NOT NULL CHECK(version=1),
 fuente_ref text NOT NULL,fuente_version numeric(20,0) NOT NULL CHECK(fuente_version=1),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 alcance_fuente text NOT NULL CHECK(alcance_fuente='sintetico_declarado'),
 operacion_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.identidad_interna_sintetica_v1(operacion_ref) DEFERRABLE INITIALLY DEFERRED,
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
DO $tables$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['identidad_interna_sintetica_v1','titularidad_identidad_interna_sintetica_v1'] LOOP
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()',t);
  EXECUTE format('ALTER TABLE vec_identidad_sesiones_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_identidad_sesiones_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.%I TO vec_identidad_sesiones_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',t,'vec_identidad_sesiones_v1_propietario','vec_identidad_sesiones_v1_propietario');
  EXECUTE format('REVOKE ALL ON TABLE vec_identidad_sesiones_v1.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_identidad_sesiones_v1.%I FROM PUBLIC',t);
 END LOOP;
END $tables$;

CREATE FUNCTION vec_identidad_sesiones_v1.validar_material_identidad_interna_sintetica_v1(p jsonb, material text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; pe jsonb; mp jsonb; clave text;
BEGIN
 PERFORM vec_identidad_sesiones_v1.validar_plan_identidad_interna_sintetica_v1(p);
 IF material IS NULL OR octet_length(material) NOT BETWEEN 1 AND 32768
 OR encode(pg_catalog.sha256(convert_to(material,'UTF8')),'hex') IS DISTINCT FROM p#>>'{fuente_hmac,huella_sha256}'
 THEN RAISE EXCEPTION 'IS17: material privado divergente' USING ERRCODE='22023'; END IF;
 m:=material::jsonb;
 IF jsonb_path_exists(m,'$.** ? (@ == null)') THEN RAISE EXCEPTION 'IS17: null privado no admitido' USING ERRCODE='22023'; END IF;
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(m,ARRAY['version','fuente_ref','fuente_version','esquema_hmac','dominio_hmac_ref','clave_hmac_id','clave_hmac_version','personas']) IS NOT TRUE
 OR m->>'version' IS DISTINCT FROM '1'
 OR EXISTS(SELECT 1 FROM jsonb_each(m) WHERE jsonb_typeof(value)<>(CASE WHEN key IN ('version','fuente_version','clave_hmac_version') THEN 'number' WHEN key='personas' THEN 'array' ELSE 'string' END)) OR m->>'fuente_ref' IS DISTINCT FROM p#>>'{fuente_hmac,referencia}'
 OR m->>'fuente_version' IS DISTINCT FROM p#>>'{fuente_hmac,version}'
 OR m->>'clave_hmac_version' !~ '^[1-9][0-9]{0,18}$'
 OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(m->>'esquema_hmac',m->>'dominio_hmac_ref',m->>'clave_hmac_id',(m->>'clave_hmac_version')::bigint) IS NOT TRUE
 OR jsonb_typeof(m->'personas') IS DISTINCT FROM 'array' OR jsonb_array_length(m->'personas')<>1
 THEN RAISE EXCEPTION 'IS17: material cerrado inválido' USING ERRCODE='22023'; END IF;
 FOR pe IN SELECT p->'persona' LOOP
  SELECT value INTO mp FROM jsonb_array_elements(m->'personas') WHERE value->>'persona_ref'=pe->>'persona_ref';
  IF mp IS NULL OR (SELECT count(*) FROM jsonb_array_elements(m->'personas') WHERE value->>'persona_ref'=pe->>'persona_ref')<>1
  OR vec_identidad_sesiones_v1.claves_fuentes_admin_v1(mp,ARRAY['persona_ref','cuenta_ordinaria_id_hmac_hex','sujeto_id_hmac_hex']) IS NOT TRUE
  OR EXISTS(SELECT 1 FROM jsonb_each(mp) WHERE jsonb_typeof(value)<>'string')
  THEN RAISE EXCEPTION 'IS17: titularidad material inválida' USING ERRCODE='22023'; END IF;
  FOREACH clave IN ARRAY ARRAY['cuenta_ordinaria_id_hmac_hex','sujeto_id_hmac_hex'] LOOP
   IF mp->>clave IS NULL OR mp->>clave !~ '^[0-9a-f]{64}$'
   OR vec_identidad_sesiones_v1.huella_hmac_valida(decode(mp->>clave,'hex')) IS NOT TRUE
   THEN RAISE EXCEPTION 'IS17: coordenada privada inválida' USING ERRCODE='22023'; END IF;
  END LOOP;
  IF mp->>'cuenta_ordinaria_id_hmac_hex'=mp->>'sujeto_id_hmac_hex'
  THEN RAISE EXCEPTION 'IS17: coordenadas confundidas' USING ERRCODE='22023'; END IF;
 END LOOP;
 RETURN m;
EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'IS17: material privado inválido' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.validar_material_identidad_interna_sintetica_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(p jsonb,material text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; r record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'IS17: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 m:=vec_identidad_sesiones_v1.validar_material_identidad_interna_sintetica_v1(p,material);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:identidad-interna-sintetica:v1',0));
 SELECT * INTO r FROM vec_identidad_sesiones_v1.identidad_interna_sintetica_v1 WHERE operacion_ref=p->>'operacion_ref';
 IF FOUND THEN
  IF r.plan IS DISTINCT FROM p OR r.material_sha256 IS DISTINCT FROM p#>>'{fuente_hmac,huella_sha256}'
  THEN RAISE EXCEPTION 'IS17: operación divergente' USING ERRCODE='40001'; END IF;
  RETURN jsonb_build_object('esquema','vec.is.preimagen-identidad-interna-sintetica.v1','version',1,'operacion_ref',p->>'operacion_ref','recibo',r.recibo);
 END IF;
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta WHERE acto_ref=p#>>'{persona,operacion_cuenta_ordinaria_ref}')
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1 WHERE persona_ref=p#>>'{persona,persona_ref}')
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.titularidad_identidad_interna_sintetica_v1 WHERE persona_ref=p#>>'{persona,persona_ref}')
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.alias_hmac_cuenta a
  WHERE a.esquema_hmac=m->>'esquema_hmac' AND a.dominio_hmac_ref=m->>'dominio_hmac_ref'
   AND a.clave_hmac_id=m->>'clave_hmac_id' AND a.clave_hmac_version=(m->>'clave_hmac_version')::bigint
   AND (a.cuenta_id_hmac IN(decode(m#>>'{personas,0,cuenta_ordinaria_id_hmac_hex}','hex'),decode(m#>>'{personas,0,sujeto_id_hmac_hex}','hex'))
    OR a.sujeto_id_hmac IN(decode(m#>>'{personas,0,cuenta_ordinaria_id_hmac_hex}','hex'),decode(m#>>'{personas,0,sujeto_id_hmac_hex}','hex'))))
 THEN RAISE EXCEPTION 'IS17: identidad ya existente' USING ERRCODE='40001'; END IF;
 RETURN jsonb_build_object('esquema','vec.is.preimagen-identidad-interna-sintetica.v1','version',1,'operacion_ref',p->>'operacion_ref','cuentas',jsonb_build_array());
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(jsonb,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(jsonb,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(p jsonb,material text,pre_sha text,operacion text,plan_sha text,aprobacion text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; pre jsonb; pe jsonb; mp jsonb; ordinaria text; recibo jsonb; existente record; ahora timestamptz;
BEGIN
 pre:=vec_identidad_sesiones_v1.preimagen_identidad_interna_sintetica_v1(p,material);
 IF operacion IS DISTINCT FROM p->>'operacion_ref' OR plan_sha IS NULL OR plan_sha !~ '^[0-9a-f]{64}$'
 OR vec_identidad_sesiones_v1.texto_tecnico_valido(aprobacion,128) IS NOT TRUE
 THEN RAISE EXCEPTION 'IS17: compromiso inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO existente FROM vec_identidad_sesiones_v1.identidad_interna_sintetica_v1 WHERE operacion_ref=operacion;
 IF FOUND THEN
  IF existente.plan_sha256 IS DISTINCT FROM plan_sha OR existente.aprobacion_ref IS DISTINCT FROM aprobacion
  THEN RAISE EXCEPTION 'IS17: replay divergente' USING ERRCODE='40001'; END IF;
  RETURN existente.recibo;
 END IF;
 IF pre_sha IS DISTINCT FROM encode(pg_catalog.sha256(convert_to(pre::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'IS17: CAS preimagen divergente' USING ERRCODE='40001'; END IF;
 m:=material::jsonb; ahora:=clock_timestamp(); pe:=p->'persona'; mp:=m#>'{personas,0}';
 SELECT cuenta_ref INTO STRICT ordinaria FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(pe->>'operacion_cuenta_ordinaria_ref',m->>'esquema_hmac',m->>'dominio_hmac_ref',m->>'clave_hmac_id',(m->>'clave_hmac_version')::bigint,decode(mp->>'cuenta_ordinaria_id_hmac_hex','hex'),decode(mp->>'sujeto_id_hmac_hex','hex'),false,NULL);
 INSERT INTO vec_identidad_sesiones_v1.titularidad_identidad_interna_sintetica_v1(cuenta_ref,persona_ref,version,fuente_ref,fuente_version,fuente_sha256,alcance_fuente,operacion_ref,vigente_desde,vigente_hasta)
 VALUES(ordinaria,pe->>'persona_ref',1,pe#>>'{fuente_titularidad,referencia}',1,pe#>>'{fuente_titularidad,huella_sha256}','sintetico_declarado',operacion,ahora,(pe->>'vigente_hasta')::timestamptz);
 recibo:=jsonb_build_object('esquema','vec.is.identidad-interna-sintetica.v1','version',1,'recibo_ref','recibo_is_identidad:'||replace(pg_catalog.gen_random_uuid()::text,'-',''),'operacion_ref',operacion,'plan_sha256',plan_sha,'aprobacion_ref',aprobacion,'alcance_fuente','sintetico_declarado','registrada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'datos',jsonb_build_object('persona_ref',pe->>'persona_ref','cuenta_ordinaria_ref',ordinaria,'version_titularidad',1));
 recibo:=recibo||jsonb_build_object('huella_sha256',encode(pg_catalog.sha256(convert_to(recibo::text,'UTF8')),'hex'));
 INSERT INTO vec_identidad_sesiones_v1.identidad_interna_sintetica_v1(operacion_ref,plan_sha256,aprobacion_ref,plan,material_sha256,recibo) VALUES(operacion,plan_sha,aprobacion,p,p#>>'{fuente_hmac,huella_sha256}',recibo);
 RETURN recibo;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(jsonb,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.aplicar_identidad_interna_sintetica_v1(jsonb,text,text,text,text,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_identidad_sesiones_v1.cotejar_recibo_identidad_interna_sintetica_v1(operacion text,plan_sha text,aprobacion text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r jsonb;
BEGIN
 SELECT recibo INTO r FROM vec_identidad_sesiones_v1.identidad_interna_sintetica_v1 WHERE operacion_ref=operacion AND plan_sha256=plan_sha AND aprobacion_ref=aprobacion;
 IF NOT FOUND THEN RAISE EXCEPTION 'IS17: recibo no acreditado' USING ERRCODE='55000'; END IF;
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.cotejar_recibo_identidad_interna_sintetica_v1(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_contexto_actor_v1_propietario,vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.cotejar_recibo_identidad_interna_sintetica_v1(text,text,text) TO vec_contexto_actor_v1_propietario,vec_autorizacion_propietario;
COMMIT;
