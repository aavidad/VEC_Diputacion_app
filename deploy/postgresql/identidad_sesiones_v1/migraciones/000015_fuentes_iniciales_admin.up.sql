\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $pre$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'IS15: clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'IS15: clave=server_version_num actual=% esperado=180000..189999',current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF to_regprocedure('vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea)') IS NULL THEN RAISE EXCEPTION 'IS15: clave=dependencia.vec_identidad_sesiones_v1.provisionar_cuenta_v1(text,text,text,text,bigint,bytea,bytea,boolean,bytea) actual=ausente esperado=presente' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_identidad_sesiones_v1.politica_certificado_admin_v1') IS NULL THEN RAISE EXCEPTION 'IS15: clave=dependencia.vec_identidad_sesiones_v1.politica_certificado_admin_v1 actual=ausente esperado=presente' USING ERRCODE='55000'; END IF;
 IF to_regclass('vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1') IS NOT NULL THEN RAISE EXCEPTION 'IS15: clave=vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1 actual=presente esperado=ausente' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;

-- Únicamente AUT39, tras cotejar operador/configuración/aprobación externa,
-- invoca estas fachadas. No hay concesiones a runtime ni permiso en el JSON.
CREATE FUNCTION vec_identidad_sesiones_v1.claves_fuentes_admin_v1(p jsonb, claves text[])
RETURNS boolean LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT jsonb_typeof(p)='object' AND p ?& claves
 AND (SELECT count(*) FROM jsonb_object_keys(p))=cardinality(claves)
$f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.claves_fuentes_admin_v1(jsonb,text[]) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.validar_plan_fuentes_iniciales_admin_v1(p jsonb)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE pe jsonb; f jsonb; pol jsonb; ahora timestamptz:=clock_timestamp();
BEGIN
 IF p IS NULL OR jsonb_path_exists(p,'$.** ? (@ == null)') THEN RAISE EXCEPTION 'IS15: null no admitido' USING ERRCODE='22023'; END IF;
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(p,ARRAY['version','operacion_ref','preparado_en','caduca_en','entorno','alcance_fuente','procedencia','organizacion','personas','fuente_hmac','politica_admin']) IS NOT TRUE
 OR p->>'version' IS DISTINCT FROM '1' OR p->>'entorno' IS DISTINCT FROM 'desarrollo'
 OR p->>'alcance_fuente' IS DISTINCT FROM 'sintetico_declarado'
 OR vec_identidad_sesiones_v1.referencia_valida(p->>'operacion_ref','pfi_') IS NOT TRUE OR octet_length(p->>'operacion_ref')>128
 OR jsonb_typeof(p->'personas') IS DISTINCT FROM 'array' OR jsonb_array_length(p->'personas')<>2
 THEN RAISE EXCEPTION 'IS15: plan cerrado inválido' USING ERRCODE='22023'; END IF;
 IF (p->>'preparado_en') !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (p->>'caduca_en') !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (p->>'preparado_en')::timestamptz>ahora OR (p->>'caduca_en')::timestamptz<=ahora
 OR (p->>'caduca_en')::timestamptz<=(p->>'preparado_en')::timestamptz
 THEN RAISE EXCEPTION 'IS15: ventana del plan inválida' USING ERRCODE='22023'; END IF;
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(p->'organizacion',ARRAY['organizacion_ref','version_esperada','vigente_hasta']) IS NOT TRUE
 OR p#>>'{organizacion,version_esperada}' IS DISTINCT FROM '0'
 OR EXISTS(SELECT 1 FROM jsonb_each(p->'organizacion') WHERE jsonb_typeof(value)<>(CASE WHEN key='version_esperada' THEN 'number' ELSE 'string' END))
 OR p#>>'{organizacion,organizacion_ref}' !~ '^org_[a-z0-9]{16,80}$'
 OR p#>>'{organizacion,vigente_hasta}' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (p#>>'{organizacion,vigente_hasta}')::timestamptz<(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'IS15: organización inválida' USING ERRCODE='22023'; END IF;
 FOR f IN SELECT p->'procedencia' UNION ALL SELECT p->'fuente_hmac'
 UNION ALL SELECT value->'fuente_titularidad' FROM jsonb_array_elements(p->'personas') LOOP
  IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(f,ARRAY['referencia','version','huella_sha256']) IS NOT TRUE
  OR vec_identidad_sesiones_v1.referencia_valida(f->>'referencia','prc_') IS NOT TRUE
  OR f->>'version' IS DISTINCT FROM '1' OR f->>'huella_sha256' !~ '^[0-9a-f]{64}$'
  OR f->>'huella_sha256'=repeat('0',64)
  OR EXISTS(SELECT 1 FROM jsonb_each(f) WHERE jsonb_typeof(value)<>(CASE WHEN key='version' THEN 'number' ELSE 'string' END))
  THEN RAISE EXCEPTION 'IS15: fuente inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 FOR pe IN SELECT value FROM jsonb_array_elements(p->'personas') LOOP
  IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(pe,ARRAY['persona_ref','version_esperada','vigente_hasta','operacion_cuenta_ordinaria_ref','operacion_cuenta_privilegiada_ref','fuente_titularidad']) IS NOT TRUE
  OR vec_identidad_sesiones_v1.referencia_valida(pe->>'persona_ref','per_') IS NOT TRUE
  OR pe->>'version_esperada' IS DISTINCT FROM '0'
  OR EXISTS(SELECT 1 FROM jsonb_each(pe) WHERE jsonb_typeof(value)<>(CASE WHEN key='version_esperada' THEN 'number' WHEN key='fuente_titularidad' THEN 'object' ELSE 'string' END))
  OR pe->>'vigente_hasta' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
  OR (pe->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
  OR (pe->>'vigente_hasta')::timestamptz>(p#>>'{organizacion,vigente_hasta}')::timestamptz
  OR vec_identidad_sesiones_v1.referencia_valida(pe->>'operacion_cuenta_ordinaria_ref','opr_') IS NOT TRUE
  OR vec_identidad_sesiones_v1.referencia_valida(pe->>'operacion_cuenta_privilegiada_ref','opr_') IS NOT TRUE
  THEN RAISE EXCEPTION 'IS15: persona inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF (SELECT count(DISTINCT value->>'persona_ref') FROM jsonb_array_elements(p->'personas'))<>2
 OR (SELECT count(DISTINCT r) FROM (SELECT value->>'operacion_cuenta_ordinaria_ref' r FROM jsonb_array_elements(p->'personas') UNION ALL SELECT value->>'operacion_cuenta_privilegiada_ref' FROM jsonb_array_elements(p->'personas')) x)<>4
 THEN RAISE EXCEPTION 'IS15: personas u operaciones repetidas' USING ERRCODE='22023'; END IF;
 pol:=p->'politica_admin';
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(pol,ARRAY['politica_ref','host_admin','ca_sha256','huella_aprobacion_sha256','maxima_edad_revocacion_segundos','vigente_hasta']) IS NOT TRUE
 OR vec_identidad_sesiones_v1.referencia_valida(pol->>'politica_ref','pga_') IS NOT TRUE
 OR octet_length(pol->>'host_admin') NOT BETWEEN 4 AND 253
 OR pol->>'host_admin' !~ '^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$'
 OR pol->>'ca_sha256' !~ '^[0-9a-f]{64}$' OR pol->>'ca_sha256'=repeat('0',64)
 OR pol->>'huella_aprobacion_sha256' !~ '^[0-9a-f]{64}$'
 OR pol->>'maxima_edad_revocacion_segundos' !~ '^[1-9][0-9]{0,9}$'
 OR (pol->>'maxima_edad_revocacion_segundos')::numeric NOT BETWEEN 1 AND 2147483647
 OR pol->>'vigente_hasta' !~ '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$'
 OR (pol->>'vigente_hasta')::timestamptz<(p->>'caduca_en')::timestamptz
 THEN RAISE EXCEPTION 'IS15: política inválida' USING ERRCODE='22023'; END IF;
 -- Null y tipos ajenos no pueden pasar como cadenas JSON.
 IF EXISTS(SELECT 1 FROM jsonb_each(p) WHERE jsonb_typeof(value)<>(CASE WHEN key='version' THEN 'number' WHEN key='personas' THEN 'array' WHEN key IN ('organizacion','procedencia','fuente_hmac','politica_admin') THEN 'object' ELSE 'string' END))
 OR EXISTS(SELECT 1 FROM jsonb_each(pol) WHERE key NOT IN ('maxima_edad_revocacion_segundos') AND jsonb_typeof(value)<>'string')
 OR EXISTS(SELECT 1 FROM jsonb_each(pol) WHERE key='maxima_edad_revocacion_segundos' AND jsonb_typeof(value)<>'number')
 THEN RAISE EXCEPTION 'IS15: tipos inválidos' USING ERRCODE='22023'; END IF;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.validar_plan_fuentes_iniciales_admin_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.validar_plan_fuentes_iniciales_admin_v1(jsonb) TO vec_contexto_actor_v1_propietario,vec_autorizacion_propietario;

CREATE TABLE vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1(
 operacion_ref text PRIMARY KEY,plan_sha256 text NOT NULL CHECK(plan_sha256 ~ '^[0-9a-f]{64}$'),
 aprobacion_ref text NOT NULL,plan jsonb NOT NULL,material_sha256 text NOT NULL,
 recibo jsonb NOT NULL,registrada_en timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1(
 cuenta_ref text PRIMARY KEY REFERENCES vec_identidad_sesiones_v1.cuenta(cuenta_ref),
 persona_ref text NOT NULL CHECK(vec_identidad_sesiones_v1.referencia_valida(persona_ref,'per_') IS TRUE),
 version numeric(20,0) NOT NULL CHECK(version=1),
 fuente_ref text NOT NULL,fuente_version numeric(20,0) NOT NULL CHECK(fuente_version=1),
 fuente_sha256 text NOT NULL CHECK(fuente_sha256 ~ '^[0-9a-f]{64}$'),
 alcance_fuente text NOT NULL CHECK(alcance_fuente='sintetico_declarado'),
 operacion_ref text NOT NULL REFERENCES vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1(operacion_ref) DEFERRABLE INITIALLY DEFERRED,
 vigente_desde timestamptz NOT NULL,vigente_hasta timestamptz NOT NULL,
 CHECK(isfinite(vigente_desde) AND isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)
);
DO $tables$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['fuentes_iniciales_admin_v1','titularidad_cuenta_persona_v1'] LOOP
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_identidad_sesiones_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_identidad_sesiones_v1.rechazar_mutacion()',t);
  EXECUTE format('ALTER TABLE vec_identidad_sesiones_v1.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_identidad_sesiones_v1.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY propietario_exacto ON vec_identidad_sesiones_v1.%I TO vec_identidad_sesiones_v1_propietario USING(current_user=%L) WITH CHECK(current_user=%L)',t,'vec_identidad_sesiones_v1_propietario','vec_identidad_sesiones_v1_propietario');
  EXECUTE format('REVOKE ALL ON TABLE vec_identidad_sesiones_v1.%I FROM PUBLIC',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_identidad_sesiones_v1.%I FROM PUBLIC',t);
 END LOOP;
END $tables$;

CREATE FUNCTION vec_identidad_sesiones_v1.validar_material_fuentes_admin_v1(p jsonb, material text)
RETURNS jsonb LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; pe jsonb; mp jsonb; clave text;
BEGIN
 PERFORM vec_identidad_sesiones_v1.validar_plan_fuentes_iniciales_admin_v1(p);
 IF material IS NULL OR octet_length(material) NOT BETWEEN 1 AND 32768
 OR encode(pg_catalog.sha256(convert_to(material,'UTF8')),'hex') IS DISTINCT FROM p#>>'{fuente_hmac,huella_sha256}'
 THEN RAISE EXCEPTION 'IS15: material privado divergente' USING ERRCODE='22023'; END IF;
 m:=material::jsonb;
 IF jsonb_path_exists(m,'$.** ? (@ == null)') THEN RAISE EXCEPTION 'IS15: null privado no admitido' USING ERRCODE='22023'; END IF;
 IF vec_identidad_sesiones_v1.claves_fuentes_admin_v1(m,ARRAY['version','fuente_ref','fuente_version','esquema_hmac','dominio_hmac_ref','clave_hmac_id','clave_hmac_version','personas']) IS NOT TRUE
 OR m->>'version' IS DISTINCT FROM '1'
 OR EXISTS(SELECT 1 FROM jsonb_each(m) WHERE jsonb_typeof(value)<>(CASE WHEN key IN ('version','fuente_version','clave_hmac_version') THEN 'number' WHEN key='personas' THEN 'array' ELSE 'string' END)) OR m->>'fuente_ref' IS DISTINCT FROM p#>>'{fuente_hmac,referencia}'
 OR m->>'fuente_version' IS DISTINCT FROM p#>>'{fuente_hmac,version}'
 OR m->>'clave_hmac_version' !~ '^[1-9][0-9]{0,18}$'
 OR vec_identidad_sesiones_v1.coordenadas_hmac_validas(m->>'esquema_hmac',m->>'dominio_hmac_ref',m->>'clave_hmac_id',(m->>'clave_hmac_version')::bigint) IS NOT TRUE
 OR jsonb_typeof(m->'personas') IS DISTINCT FROM 'array' OR jsonb_array_length(m->'personas')<>2
 THEN RAISE EXCEPTION 'IS15: material cerrado inválido' USING ERRCODE='22023'; END IF;
 FOR pe IN SELECT value FROM jsonb_array_elements(p->'personas') LOOP
  SELECT value INTO mp FROM jsonb_array_elements(m->'personas') WHERE value->>'persona_ref'=pe->>'persona_ref';
  IF mp IS NULL OR (SELECT count(*) FROM jsonb_array_elements(m->'personas') WHERE value->>'persona_ref'=pe->>'persona_ref')<>1
  OR vec_identidad_sesiones_v1.claves_fuentes_admin_v1(mp,ARRAY['persona_ref','cuenta_ordinaria_id_hmac_hex','cuenta_privilegiada_id_hmac_hex','sujeto_id_hmac_hex']) IS NOT TRUE
  OR EXISTS(SELECT 1 FROM jsonb_each(mp) WHERE jsonb_typeof(value)<>'string')
  THEN RAISE EXCEPTION 'IS15: titularidad material inválida' USING ERRCODE='22023'; END IF;
  FOREACH clave IN ARRAY ARRAY['cuenta_ordinaria_id_hmac_hex','cuenta_privilegiada_id_hmac_hex','sujeto_id_hmac_hex'] LOOP
   IF mp->>clave IS NULL OR mp->>clave !~ '^[0-9a-f]{64}$'
   OR vec_identidad_sesiones_v1.huella_hmac_valida(decode(mp->>clave,'hex')) IS NOT TRUE
   THEN RAISE EXCEPTION 'IS15: coordenada privada inválida' USING ERRCODE='22023'; END IF;
  END LOOP;
  IF mp->>'cuenta_ordinaria_id_hmac_hex'=mp->>'sujeto_id_hmac_hex'
  OR mp->>'cuenta_privilegiada_id_hmac_hex'=mp->>'sujeto_id_hmac_hex'
  OR mp->>'cuenta_ordinaria_id_hmac_hex'=mp->>'cuenta_privilegiada_id_hmac_hex'
  THEN RAISE EXCEPTION 'IS15: coordenadas confundidas' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF (SELECT count(DISTINCT value->>'sujeto_id_hmac_hex') FROM jsonb_array_elements(m->'personas'))<>2
 OR (SELECT count(DISTINCT h) FROM (SELECT value->>'cuenta_ordinaria_id_hmac_hex' h FROM jsonb_array_elements(m->'personas') UNION ALL SELECT value->>'cuenta_privilegiada_id_hmac_hex' FROM jsonb_array_elements(m->'personas')) x)<>4
 THEN RAISE EXCEPTION 'IS15: sujetos o cuentas confundidos' USING ERRCODE='22023'; END IF;
 IF (SELECT count(DISTINCT h) FROM (SELECT value->>'cuenta_ordinaria_id_hmac_hex' h FROM jsonb_array_elements(m->'personas') UNION ALL SELECT value->>'cuenta_privilegiada_id_hmac_hex' FROM jsonb_array_elements(m->'personas') UNION ALL SELECT value->>'sujeto_id_hmac_hex' FROM jsonb_array_elements(m->'personas')) x)<>6 THEN RAISE EXCEPTION 'IS15: dominios privados confundidos' USING ERRCODE='22023'; END IF;
 RETURN m;
EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'IS15: material privado inválido' USING ERRCODE='22023';
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.validar_material_fuentes_admin_v1(jsonb,text) FROM PUBLIC;

CREATE FUNCTION vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(p jsonb,material text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; r record;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'IS15: requiere SERIALIZABLE escritura' USING ERRCODE='25000'; END IF;
 m:=vec_identidad_sesiones_v1.validar_material_fuentes_admin_v1(p,material);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:continuidad:v1',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec:admin:fuentes-iniciales:v1',0));
 SELECT * INTO r FROM vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1 WHERE operacion_ref=p->>'operacion_ref';
 IF FOUND THEN
  IF r.plan IS DISTINCT FROM p OR r.material_sha256 IS DISTINCT FROM p#>>'{fuente_hmac,huella_sha256}'
  THEN RAISE EXCEPTION 'IS15: operación divergente' USING ERRCODE='40001'; END IF;
  RETURN jsonb_build_object('esquema','vec.is.preimagen-fuentes-admin.v1','version',1,'operacion_ref',p->>'operacion_ref','recibo',r.recibo);
 END IF;
 IF EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.politica_certificado_admin_v1)
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.cuenta c JOIN LATERAL jsonb_array_elements(p->'personas') pe ON c.acto_ref IN(pe->>'operacion_cuenta_ordinaria_ref',pe->>'operacion_cuenta_privilegiada_ref'))
 OR EXISTS(SELECT 1 FROM vec_identidad_sesiones_v1.alias_hmac_cuenta a JOIN LATERAL jsonb_array_elements(m->'personas') mp ON a.cuenta_id_hmac IN(decode(mp->>'cuenta_ordinaria_id_hmac_hex','hex'),decode(mp->>'cuenta_privilegiada_id_hmac_hex','hex')) WHERE a.esquema_hmac=m->>'esquema_hmac' AND a.dominio_hmac_ref=m->>'dominio_hmac_ref' AND a.clave_hmac_id=m->>'clave_hmac_id' AND a.clave_hmac_version=(m->>'clave_hmac_version')::bigint)
 THEN RAISE EXCEPTION 'IS15: fuentes iniciales ya existentes' USING ERRCODE='40001'; END IF;
 RETURN jsonb_build_object('esquema','vec.is.preimagen-fuentes-admin.v1','version',1,'operacion_ref',p->>'operacion_ref','politica',NULL,'cuentas',jsonb_build_array());
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(jsonb,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(jsonb,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(p jsonb,material text,pre_sha text,operacion text,plan_sha text,aprobacion text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb; pre jsonb; pe jsonb; mp jsonb; pol jsonb; ordinaria text; privilegiada text; cuentas jsonb:='[]'::jsonb; recibo jsonb; existente record; ahora timestamptz;
BEGIN
 pre:=vec_identidad_sesiones_v1.preimagen_fuentes_iniciales_admin_v1(p,material);
 IF operacion IS DISTINCT FROM p->>'operacion_ref' OR plan_sha IS NULL OR plan_sha !~ '^[0-9a-f]{64}$'
 OR vec_identidad_sesiones_v1.texto_tecnico_valido(aprobacion,128) IS NOT TRUE
 THEN RAISE EXCEPTION 'IS15: compromiso inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO existente FROM vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1 WHERE operacion_ref=operacion;
 IF FOUND THEN
  IF existente.plan_sha256 IS DISTINCT FROM plan_sha OR existente.aprobacion_ref IS DISTINCT FROM aprobacion
  THEN RAISE EXCEPTION 'IS15: replay divergente' USING ERRCODE='40001'; END IF;
  RETURN existente.recibo;
 END IF;
 IF pre_sha IS DISTINCT FROM encode(pg_catalog.sha256(convert_to(pre::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'IS15: CAS preimagen divergente' USING ERRCODE='40001'; END IF;
 m:=material::jsonb; ahora:=clock_timestamp(); pol:=p->'politica_admin';
 FOR pe IN SELECT value FROM jsonb_array_elements(p->'personas') ORDER BY value->>'persona_ref' LOOP
  SELECT value INTO STRICT mp FROM jsonb_array_elements(m->'personas') WHERE value->>'persona_ref'=pe->>'persona_ref';
  SELECT cuenta_ref INTO STRICT ordinaria FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(pe->>'operacion_cuenta_ordinaria_ref',m->>'esquema_hmac',m->>'dominio_hmac_ref',m->>'clave_hmac_id',(m->>'clave_hmac_version')::bigint,decode(mp->>'cuenta_ordinaria_id_hmac_hex','hex'),decode(mp->>'sujeto_id_hmac_hex','hex'),false,NULL);
  SELECT cuenta_ref INTO STRICT privilegiada FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(pe->>'operacion_cuenta_privilegiada_ref',m->>'esquema_hmac',m->>'dominio_hmac_ref',m->>'clave_hmac_id',(m->>'clave_hmac_version')::bigint,decode(mp->>'cuenta_privilegiada_id_hmac_hex','hex'),decode(mp->>'sujeto_id_hmac_hex','hex'),true,decode(mp->>'cuenta_ordinaria_id_hmac_hex','hex'));
  INSERT INTO vec_identidad_sesiones_v1.titularidad_cuenta_persona_v1(cuenta_ref,persona_ref,version,fuente_ref,fuente_version,fuente_sha256,alcance_fuente,operacion_ref,vigente_desde,vigente_hasta)
  SELECT c,pe->>'persona_ref',1,pe#>>'{fuente_titularidad,referencia}',1,pe#>>'{fuente_titularidad,huella_sha256}','sintetico_declarado',operacion,ahora,(pe->>'vigente_hasta')::timestamptz FROM unnest(ARRAY[ordinaria,privilegiada]) c;
  cuentas:=cuentas||jsonb_build_array(jsonb_build_object('persona_ref',pe->>'persona_ref','cuenta_ordinaria_ref',ordinaria,'cuenta_privilegiada_ref',privilegiada,'version_titularidad',1));
 END LOOP;
 INSERT INTO vec_identidad_sesiones_v1.politica_certificado_admin_v1(politica_ref,entorno,host_admin,ca_sha256,huella_aprobacion_sha256,maxima_edad_revocacion,vigente_hasta)
 VALUES(pol->>'politica_ref','desarrollo',pol->>'host_admin',pol->>'ca_sha256',pol->>'huella_aprobacion_sha256',(pol->>'maxima_edad_revocacion_segundos')::integer*interval '1 second',(pol->>'vigente_hasta')::timestamptz);
 recibo:=jsonb_build_object('esquema','vec.is.fuentes-iniciales-admin.v1','version',1,'recibo_ref','recibo_is_fuentes:'||replace(pg_catalog.gen_random_uuid()::text,'-',''),'operacion_ref',operacion,'plan_sha256',plan_sha,'aprobacion_ref',aprobacion,'alcance_fuente','sintetico_declarado','registrada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'datos',jsonb_build_object('personas',cuentas,'politica_ref',pol->>'politica_ref'));
 recibo:=recibo||jsonb_build_object('huella_sha256',encode(pg_catalog.sha256(convert_to(recibo::text,'UTF8')),'hex'));
 INSERT INTO vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1(operacion_ref,plan_sha256,aprobacion_ref,plan,material_sha256,recibo) VALUES(operacion,plan_sha,aprobacion,p,p#>>'{fuente_hmac,huella_sha256}',recibo);
 RETURN recibo;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(jsonb,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.aplicar_fuentes_iniciales_admin_v1(jsonb,text,text,text,text,text) TO vec_autorizacion_propietario;

CREATE FUNCTION vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(operacion text,plan_sha text,aprobacion text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE r jsonb;
BEGIN
 SELECT recibo INTO r FROM vec_identidad_sesiones_v1.fuentes_iniciales_admin_v1 WHERE operacion_ref=operacion AND plan_sha256=plan_sha AND aprobacion_ref=aprobacion;
 IF NOT FOUND THEN RAISE EXCEPTION 'IS15: recibo no acreditado' USING ERRCODE='55000'; END IF;
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_identidad_sesiones_v1 TO vec_contexto_actor_v1_propietario,vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text) TO vec_contexto_actor_v1_propietario,vec_autorizacion_propietario;
COMMIT;
