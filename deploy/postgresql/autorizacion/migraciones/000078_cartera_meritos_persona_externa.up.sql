\set ON_ERROR_STOP on
-- AUT78: cotejo vivo de cartera de Méritos para persona externa, también antes
-- de una candidatura. El certificado no crea una concesión ni una asignación.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000078',0));
DO $pre$ BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_meritos_cartera_externo') IS NOT NULL
 OR to_regrole('vec_autorizacion_fuente_meritos_externa') IS NOT NULL
 OR to_regrole('vec_autorizacion_registro_meritos_externo') IS NOT NULL
 OR to_regrole('vec_autorizacion_motivos_meritos_externo') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.validar_motivo_retirada_meritos_actual_v1(jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.listar_motivos_meritos_cartera_actual_v1(text)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.resolver_motivo_autorizacion_v2_actual(text,integer,text,text)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.listar_motivos_meritos_cartera_v1(text)') IS NULL
 OR to_regprocedure('vec_autorizacion.revalidar_sesion_cartera_meritos_v1(jsonb,timestamptz,timestamptz,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_identidad_externa_v1.acreditar_sesion_externa_v1(text,text,text,text,text,text,boolean,text,text,text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,timestamptz)') IS NULL
 OR to_regprocedure('vec_autorizacion.login_candidato_externo_v1(text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion.vinculo_contexto_actor_v2_valido(jsonb)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(text,text,text)') IS NULL
 OR to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)') IS NULL
 OR to_regprocedure('vec_autorizacion.registrar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.resolver_motivo_meritos_externo_v1(text,integer,text,text,timestamptz)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(bytea,bytea,numeric,numeric)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(bytea,bytea,numeric,numeric)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.decision_contexto_actor_v3_canonica(jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion.motivo_contexto_actor_v3_canonico(jsonb)') IS NULL
 OR to_regprocedure('vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(text,text)') IS NULL
 OR to_regclass('vec_autorizacion.asignacion_perfil_actual_externa') IS NULL
 THEN RAISE EXCEPTION 'AUT78: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_meritos_cartera_externo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_fuente_meritos_externa NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_registro_meritos_externo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_autorizacion_motivos_meritos_externo NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
DO $db$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_meritos_cartera_externo',current_database()); END $db$;
DO $db_fuente$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_fuente_meritos_externa',current_database()); END $db_fuente$;
DO $db_registro$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_registro_meritos_externo',current_database()); END $db_registro$;
DO $db_motivos$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_autorizacion_motivos_meritos_externo',current_database()); END $db_motivos$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.obtener_instantanea_meritos_externo_v1(
 p_principal_id text,p_perfil_activo_ref text)
RETURNS TABLE(documento_asignacion jsonb,documento_rol jsonb,
 documento_control_rol jsonb,revision_catalogo text,huella_catalogo text,
 documentos_politicas jsonb)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE rol record;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion.login_candidato_externo_v1(
  'vec_externo_meritos_fuente_autorizacion','vec_autorizacion_fuente_meritos_externa') IS NOT TRUE
 OR vec_contexto_actor_v1.perfil_usuarios_externo_provisionado_v1(
  p_principal_id,p_perfil_activo_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT78: fuente externa denegada' USING ERRCODE='42501'; END IF;
 SELECT r.rol_id,r.documento INTO rol
 FROM vec_autorizacion.asignacion_perfil_actual_externa ap
 JOIN vec_autorizacion.asignacion_perfil_externa a
  ON a.perfil_activo_ref=ap.perfil_activo_ref AND a.asignacion_ref=ap.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 WHERE ap.perfil_activo_ref=p_perfil_activo_ref AND a.principal_id=p_principal_id
 FOR SHARE OF ap,a,r;
 IF NOT FOUND OR rol.rol_id IS DISTINCT FROM 'persona_meritos_cartera'
 OR jsonb_typeof(rol.documento->'concesiones') IS DISTINCT FROM 'array'
 OR ((SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
    FROM jsonb_array_elements(rol.documento->'concesiones') x) IS DISTINCT FROM
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[]
  AND (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
    FROM jsonb_array_elements(rol.documento->'concesiones') x) IS DISTINCT FROM
   ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[])
 THEN RAISE EXCEPTION 'AUT78: rol externo de Méritos ausente' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.documento,r.documento,c.documento,cat.revision::text,cat.huella_sha256,
  coalesce((SELECT jsonb_agg(p.documento ORDER BY p.politica_ref)
   FROM vec_autorizacion.politica_restrictiva_actual pa
   JOIN vec_autorizacion.politica_restrictiva p
    ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref),'[]'::jsonb)
 FROM vec_autorizacion.asignacion_perfil_actual_externa ap
 JOIN vec_autorizacion.asignacion_perfil_externa a
  ON a.perfil_activo_ref=ap.perfil_activo_ref AND a.asignacion_ref=ap.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol_actual cp
  ON cp.version_rol_ref=r.version_rol_ref
 JOIN vec_autorizacion.control_vigencia_version_rol c
  ON c.version_rol_ref=cp.version_rol_ref AND c.revision=cp.revision
 JOIN vec_autorizacion.control_catalogo_politicas cat ON cat.control_id IS TRUE
 WHERE ap.perfil_activo_ref=p_perfil_activo_ref AND a.principal_id=p_principal_id
  AND r.rol_id='persona_meritos_cartera' AND r.documento->>'estado'='publicada'
  AND c.estado='habilitada';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)
 TO vec_autorizacion_fuente_meritos_externa;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_autorizacion_fuente_meritos_externa;

CREATE FUNCTION vec_autorizacion.validar_motivo_retirada_meritos_actual_v1(p_motivo jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE cc jsonb;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR jsonb_typeof(p_motivo) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_motivo))<>4
 OR (p_motivo ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']) IS NOT TRUE
 OR p_motivo->>'catalogo_id' IS DISTINCT FROM 'motivos_meritos_cartera'
 OR jsonb_typeof(p_motivo->'catalogo_version') IS DISTINCT FROM 'number'
 OR p_motivo->>'catalogo_version' !~ '^[1-9][0-9]{0,8}$'
 OR p_motivo->>'catalogo_huella_sha256' !~ '^[0-9a-f]{64}$'
 OR p_motivo->>'entrada_clave' !~ '^motivo_[0-9a-f]{32}$'
 THEN RETURN false; END IF;
 cc:=vec_catalogos_configurables.listar_motivos_meritos_cartera_v1('es');
 IF cc#>>'{catalogo_aut,catalogo_id}' IS DISTINCT FROM p_motivo->>'catalogo_id'
 OR cc#>>'{catalogo_aut,catalogo_version}' IS DISTINCT FROM p_motivo->>'catalogo_version'
 OR cc#>>'{catalogo_aut,catalogo_huella_sha256}' IS DISTINCT FROM p_motivo->>'catalogo_huella_sha256'
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(cc->'motivos') e
  WHERE e->>'entrada_clave'=p_motivo->>'entrada_clave')
 THEN RETURN false;END IF;
 RETURN vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
  p_motivo->>'catalogo_id',(p_motivo->>'catalogo_version')::integer,
  p_motivo->>'catalogo_huella_sha256',p_motivo->>'entrada_clave') IS TRUE;
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.validar_motivo_retirada_meritos_actual_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.validar_motivo_retirada_meritos_actual_v1(jsonb)
 TO vec_meritos_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion TO vec_meritos_propietario;

CREATE FUNCTION vec_autorizacion.listar_motivos_meritos_cartera_actual_v1(p_idioma text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE cc jsonb;aut jsonb;entrada jsonb;salida jsonb:='[]'::jsonb;
 id text;version integer;huella text;clave text;etiqueta text;total integer:=0;
 total_aut integer;ahora timestamptz(6);
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR p_idioma NOT IN ('es','en') OR p_idioma IS NULL
 OR session_user<>'vec_externo_meritos_cartera'
 OR NOT pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
 THEN RAISE EXCEPTION 'AUT78: lectura de motivos denegada' USING ERRCODE='42501';END IF;
 cc:=vec_catalogos_configurables.listar_motivos_meritos_cartera_v1(p_idioma);
 aut:=cc->'catalogo_aut';
 IF jsonb_typeof(cc) IS DISTINCT FROM 'object'
 OR cc->>'idioma' IS DISTINCT FROM p_idioma
 OR cc#>>'{catalogo_cc,referencia}' IS DISTINCT FROM 'meritos.cartera.motivos'
 OR jsonb_typeof(aut) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(aut))<>3
 OR aut->>'catalogo_id' IS DISTINCT FROM 'motivos_meritos_cartera'
 OR jsonb_typeof(aut->'catalogo_version') IS DISTINCT FROM 'number'
 OR coalesce(aut->>'catalogo_version' ~ '^[1-9][0-9]{0,8}$',false) IS NOT TRUE
 OR coalesce(aut->>'catalogo_huella_sha256' ~ '^[0-9a-f]{64}$',false) IS NOT TRUE
 OR jsonb_typeof(cc->'motivos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'AUT78: catálogo CC incompatible' USING ERRCODE='55000';END IF;
 id:=aut->>'catalogo_id';version:=(aut->>'catalogo_version')::integer;
 huella:=aut->>'catalogo_huella_sha256';
 FOR entrada IN SELECT value FROM jsonb_array_elements(cc->'motivos') LOOP
  clave:=entrada->>'entrada_clave';etiqueta:=entrada->>'etiqueta';
  IF jsonb_typeof(entrada) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(entrada))<>2
  OR clave !~ '^motivo_[0-9a-f]{32}$'
  OR etiqueta IS NULL OR octet_length(etiqueta) NOT BETWEEN 1 AND 200
  OR vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
    id,version,huella,clave) IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT78: motivo CC/AUT divergente' USING ERRCODE='55000';END IF;
  salida:=salida||jsonb_build_array(jsonb_build_object(
   'motivo',jsonb_build_object('catalogo_id',id,'catalogo_version',version,
    'catalogo_huella_sha256',huella,'entrada_clave',clave),
   'etiqueta',etiqueta));
  total:=total+1;
 END LOOP;
 IF total NOT BETWEEN 1 AND 128
 THEN RAISE EXCEPTION 'AUT78: motivos vigentes ausentes' USING ERRCODE='55000';END IF;
 ahora:=clock_timestamp();
 SELECT count(*)::integer INTO total_aut
 FROM vec_autorizacion.motivo_v2_catalogo_publicado c
 JOIN vec_autorizacion.motivo_v2_entrada e
  ON e.catalogo_id=c.catalogo_id AND e.catalogo_version=c.catalogo_version
 WHERE c.catalogo_id=id AND c.catalogo_version=version
  AND c.catalogo_huella_publicada_sha256=huella
  AND c.publicado_en<=ahora AND e.vigente_desde<=ahora
  AND (e.vigente_hasta IS NULL OR ahora<e.vigente_hasta)
  AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r
   WHERE r.catalogo_id=c.catalogo_id AND r.catalogo_version=c.catalogo_version);
 IF total_aut<>total
 THEN RAISE EXCEPTION 'AUT78: motivo vigente sin etiqueta' USING ERRCODE='55000';END IF;
 RETURN jsonb_build_object('motivos',salida,'idioma',p_idioma);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.listar_motivos_meritos_cartera_actual_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.listar_motivos_meritos_cartera_actual_v1(text)
 TO vec_meritos_propietario;
CREATE FUNCTION vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
 p_vinculo jsonb,p_emitida_en timestamptz,p_valida_hasta timestamptz,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_vinculo) IS DISTINCT FROM 'object'
 OR vec_autorizacion.vinculo_contexto_actor_v2_valido(p_vinculo) IS NOT TRUE
 OR p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR p_vinculo->>'metodo_observado' NOT IN ('certificado','dnie')
 OR p_vinculo->>'garantia_observada' IS DISTINCT FROM 'alto'
 OR p_emitida_en IS NULL OR p_valida_hasta IS NULL OR p_instante IS NULL
 OR p_valida_hasta<=p_emitida_en OR p_instante<p_emitida_en OR p_instante>=p_valida_hasta
 OR NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   JOIN pg_roles g ON g.oid=m.roleid
   WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL
   AND ((g.rolname='vec_meritos_cartera_externo'
     AND l.rolname='vec_externo_meritos_cartera')
    OR (g.rolname='vec_autorizacion_registro_meritos_externo'
     AND l.rolname='vec_externo_meritos_registro_autorizacion')
    OR (g.rolname='vec_meritos_ejecutor'
     AND l.rolname='vec_externo_meritos_operacion'))
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 OR (CASE WHEN session_user='vec_externo_meritos_operacion' THEN
      (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>2
      OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
       AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
       AND NOT m.set_option AND NOT m.admin_option)
      ELSE (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1 END)
 THEN RETURN false; END IF;
 IF vec_identidad_externa_v1.acreditar_sesion_externa_v1(
   p_vinculo->>'autenticacion_ref',p_vinculo->>'autenticacion_huella_sha256',
   p_vinculo->>'asercion_ref',p_vinculo->>'sesion_ref',p_vinculo->>'cuenta_ref',
   p_vinculo->>'cuenta_ordinaria_ref',(p_vinculo->>'cuenta_privilegiada')::boolean,
   p_vinculo->>'superficie',p_vinculo->>'metodo_observado',p_vinculo->>'garantia_observada',
   p_vinculo->>'politica_garantia_ref',p_vinculo->>'politica_garantia_huella_sha256',
   (p_vinculo->>'autenticacion_verificada_en')::timestamptz,
   (p_vinculo->>'sesion_emitida_en')::timestamptz,
   p_vinculo->>'control_sesion_ref',p_vinculo->>'control_sesion_revision','activa',
   p_vinculo->>'control_sesion_huella_sha256',
   (p_vinculo->>'sesion_revalidada_en')::timestamptz,
   (p_vinculo->>'sesion_valida_hasta')::timestamptz) IS NOT TRUE
 THEN RETURN false; END IF;
 RETURN p_emitida_en >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_valida_hasta <= (p_vinculo->>'sesion_valida_hasta')::timestamptz
  AND p_instante >= (p_vinculo->>'sesion_revalidada_en')::timestamptz
  AND p_instante < (p_vinculo->>'sesion_valida_hasta')::timestamptz;
EXCEPTION WHEN data_exception OR invalid_text_representation OR datetime_field_overflow THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
 jsonb,timestamptz,timestamptz,timestamptz) FROM PUBLIC;

-- Se clona solo el transporte externo V3 existente. El consumidor AD250
-- conserva las mismas firmas criptográficas y comprueba acción/audiencia.
DO $clonar_v3$
DECLARE v record;p pg_proc%ROWTYPE;original text;nuevo text;firma_nueva text;
BEGIN
 FOR v IN SELECT * FROM (VALUES
  ('vec_autorizacion.registrar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)',
   'registrar_decision_contexto_actor_v3_externa_interna',
   'registrar_decision_contexto_actor_v3_meritos_interna',
   '4dd4719d58c64526ec9f6001a2227d479a5182001cf9c125548898cb52fc2fa6'),
  ('vec_autorizacion.revalidar_decision_contexto_actor_v3_externa_interna(bytea,bytea,numeric,numeric)',
   'revalidar_decision_contexto_actor_v3_externa_interna',
   'revalidar_decision_contexto_actor_v3_meritos_interna',
   'eeed4d2d437d463a5fbed163e23168126184dd2003d095fc4c90ceb0ee54052c')
 ) AS x(firma,nombre,nuevo_nombre,huella) LOOP
  SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(v.firma);
  IF NOT FOUND OR p.proowner<>to_regrole('vec_autorizacion_propietario')
   OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v'
   OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM v.huella
  THEN RAISE EXCEPTION 'AUT78: preimagen V3 externa divergente %',v.firma USING ERRCODE='55000';END IF;
  original:=pg_get_functiondef(p.oid);
  IF strpos(original,'CREATE OR REPLACE FUNCTION vec_autorizacion.'||v.nombre||'(')<>1
   OR (length(original)-length(replace(original,
    'vec_autorizacion.revalidar_sesion_vinculo_externo_v2','')))
     /length('vec_autorizacion.revalidar_sesion_vinculo_externo_v2')<1
  THEN RAISE EXCEPTION 'AUT78: fuente V3 sin marca única %',v.firma USING ERRCODE='55000';END IF;
 END LOOP;
END $clonar_v3$;

CREATE OR REPLACE FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3_meritos_interna(p_decision_canonica bytea, p_motivo_canonico bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS TABLE(concedida boolean, codigo text, decision_huella_sha256 text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE
    d jsonb;
    v jsonb;
    motivo jsonb;
    rm jsonb;
    asignacion record;
    rol record;
    catalogo record;
    manifiesto jsonb;
    referencias jsonb;
    instante timestamptz;
    primera_acreditacion timestamptz;
    huella text;
    coincidencias integer;
    replay record;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION USING ERRCODE = '25000',
            MESSAGE = 'registro V3 requiere SERIALIZABLE de escritura';
    END IF;
    -- Toda invocacion participa en la barrera de retirada. El down toma este
    -- mismo advisory en modo exclusivo antes de comprobar o alterar objetos.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
        )
    );
    IF p_decision_canonica IS NULL OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR pg_catalog.octet_length(p_motivo_canonico) NOT BETWEEN 1 AND 65536
       OR p_persona_version IS NULL OR pg_catalog.scale(p_persona_version) <> 0
       OR p_persona_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_perfil_version IS NULL OR pg_catalog.scale(p_perfil_version) <> 0
       OR p_perfil_version NOT BETWEEN 1 AND 18446744073709551615::numeric THEN
        RETURN;
    END IF;
    BEGIN
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
        motivo := pg_catalog.convert_from(p_motivo_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN character_not_in_repertoire OR untranslatable_character
            OR invalid_text_representation OR data_exception THEN
            RETURN;
    END;
    IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
       OR vec_autorizacion.decision_contexto_actor_v3_canonica(d)
          IS DISTINCT FROM p_decision_canonica
       OR pg_catalog.jsonb_typeof(motivo) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(motivo)) <> 2
       OR NOT (motivo ?& ARRAY['esquema', 'referencia'])
       OR motivo ->> 'esquema' IS DISTINCT FROM
          'vec.autorizacion.motivo.v2.referencia-opaca-catalogada'
       OR pg_catalog.jsonb_typeof(motivo -> 'referencia') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(motivo -> 'referencia')) <> 4
       OR pg_catalog.encode(pg_catalog.sha256(p_motivo_canonico), 'hex')
          IS DISTINCT FROM d ->> 'motivo_huella_sha256' THEN
        RETURN;
    END IF;
    rm := motivo -> 'referencia';
    IF NOT (rm ?& ARRAY[
           'catalogo_id', 'catalogo_version',
           'catalogo_huella_sha256', 'entrada_clave'
       ])
       OR pg_catalog.jsonb_typeof(rm -> 'catalogo_id') <> 'string'
       OR pg_catalog.jsonb_typeof(rm -> 'catalogo_version') <> 'number'
       OR pg_catalog.jsonb_typeof(rm -> 'catalogo_huella_sha256') <> 'string'
       OR pg_catalog.jsonb_typeof(rm -> 'entrada_clave') <> 'string'
       OR rm ->> 'catalogo_id' !~ '^[a-z][a-z0-9._-]{0,127}$'
       OR rm ->> 'catalogo_version' !~ '^[1-9][0-9]{0,9}$'
       OR (rm ->> 'catalogo_version')::numeric NOT BETWEEN 1 AND 2147483647
       OR rm ->> 'catalogo_huella_sha256' !~ '^[0-9a-f]{64}$'
       OR rm ->> 'catalogo_huella_sha256' = pg_catalog.repeat('0', 64)
       OR rm ->> 'entrada_clave' !~ '^motivo_[0-9a-f]{32}$'
       OR vec_autorizacion.motivo_contexto_actor_v3_canonico(motivo)
          IS DISTINCT FROM p_motivo_canonico THEN
        RETURN;
    END IF;
    v := d -> 'vinculo_autenticacion_actor';
    huella := pg_catalog.encode(pg_catalog.sha256(p_decision_canonica), 'hex');

    -- Reconciliacion de un COMMIT incierto: una evidencia ya durable se
    -- devuelve incluso si despues caduco o fue revocada. No crea una capacidad
    -- nueva; reproduce el mismo resultado sellado y exige tambien las dos
    -- versiones que no viajan en el vinculo minimizado.
    SELECT pg_catalog.count(*) INTO coincidencias
      FROM (
        SELECT decision_ref, huella_decision_sha256
          FROM vec_autorizacion.decision_concedida_contexto_actor_v3_externa
        UNION ALL
        SELECT decision_ref, huella_decision_sha256
          FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa
      ) AS existente
     WHERE existente.decision_ref = d ->> 'decision_ref'
        OR existente.huella_decision_sha256 = huella;
    IF coincidencias > 0 THEN
        SELECT existente.concedida, existente.codigo, existente.registrada_en
          INTO replay
          FROM (
            SELECT true AS concedida, 'concedida'::text AS codigo,
                   c.decision_ref, c.huella_decision_sha256,
                   c.decision_canonica, c.motivo_canonico,
                   c.persona_version, c.perfil_version, c.registrada_en
              FROM vec_autorizacion.decision_concedida_contexto_actor_v3_externa AS c
            UNION ALL
            SELECT false, n.codigo, n.decision_ref,
                   n.huella_decision_sha256, n.decision_canonica,
                   n.motivo_canonico, n.persona_version,
                   n.perfil_version, n.registrada_en
              FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa AS n
          ) AS existente
         WHERE existente.decision_ref = d ->> 'decision_ref'
           AND existente.huella_decision_sha256 = huella
           AND existente.decision_canonica = p_decision_canonica
           AND existente.motivo_canonico = p_motivo_canonico
           AND existente.persona_version = p_persona_version
           AND existente.perfil_version = p_perfil_version;
        IF NOT FOUND OR coincidencias <> 1 THEN
            RAISE EXCEPTION USING ERRCODE = '23505',
                MESSAGE = 'colision de identidad de decision V3';
        END IF;
        RETURN QUERY SELECT replay.concedida, replay.codigo, huella,
                            replay.registrada_en;
        RETURN;
    END IF;

    -- Primera acreditacion: antes de cualquier lock propio de autorizacion.
    primera_acreditacion :=
      vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        v ->> 'registro_contexto_ref', v ->> 'contexto_actor_esquema',
        v ->> 'contexto_actor_huella_sha256',
        v ->> 'manifiesto_procedencia_huella_sha256',
        v ->> 'autoridad_efectiva', v ->> 'cuenta_ref',
        (v ->> 'contexto_actor_cuenta_version')::numeric,
        v ->> 'principal_id', p_persona_version,
        v ->> 'perfil_activo_ref', p_perfil_version,
        v ->> 'contexto_actor_ref',
        (v ->> 'contexto_actor_version')::numeric,
        v ->> 'metodo_observado', v ->> 'garantia_observada',
        (d ->> 'emitida_en')::timestamptz,
        (d ->> 'valida_hasta')::timestamptz
      );
    IF primera_acreditacion IS NULL THEN RETURN; END IF;

    -- Serializa replay/colision y retirada de la migracion tras acreditar.
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended(
            'vec_autorizacion:decision-v3:' || (d ->> 'decision_ref'), 0
        )
    );
    LOCK TABLE vec_autorizacion.decision_concedida_contexto_actor_v3_externa,
               vec_autorizacion.decision_denegada_contexto_actor_v3_externa
        IN ROW EXCLUSIVE MODE;

    -- Orden propio unico: motivo, asignacion, rol, catalogo/politicas, sesion.
    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR SHARE;
    IF NOT FOUND THEN RETURN; END IF;

    SELECT a.asignacion_ref, a.principal_id, a.version_rol_ref,
           a.huella_sha256, a.documento
      INTO asignacion
      FROM vec_autorizacion.asignacion_perfil_actual_externa AS actual
      JOIN vec_autorizacion.asignacion_perfil_externa AS a
        ON a.perfil_activo_ref = actual.perfil_activo_ref
       AND a.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = d ->> 'perfil_activo_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion.asignacion_ref IS DISTINCT FROM d ->> 'asignacion_ref'
       OR asignacion.principal_id IS DISTINCT FROM d ->> 'principal_id'
       OR asignacion.version_rol_ref IS DISTINCT FROM d ->> 'version_rol_ref'
       OR asignacion.huella_sha256 IS DISTINCT FROM
          d ->> 'asignacion_huella_sha256' THEN
        RETURN;
    END IF;

    SELECT r.huella_sha256, r.publicada_en, r.documento,
           c.version_rol_ref, c.revision, c.estado,
           c.huella_sha256 AS control_huella
      INTO rol
      FROM vec_autorizacion.version_rol AS r
      JOIN vec_autorizacion.control_vigencia_version_rol_actual AS actual
        ON actual.version_rol_ref = r.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol AS c
        ON c.version_rol_ref = actual.version_rol_ref
       AND c.revision = actual.revision
     WHERE r.version_rol_ref = asignacion.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND OR rol.huella_sha256 IS DISTINCT FROM
          d ->> 'version_rol_huella_sha256'
       OR rol.version_rol_ref IS DISTINCT FROM
          d ->> 'control_vigencia_version_rol_ref'
       OR rol.revision IS DISTINCT FROM
          (d ->> 'control_vigencia_version_rol_revision')::numeric
       OR rol.control_huella IS DISTINCT FROM
          d ->> 'control_vigencia_version_rol_huella_sha256' THEN
        RETURN;
    END IF;

    SELECT revision, huella_sha256
      INTO catalogo
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND OR catalogo.revision IS DISTINCT FROM
          (d ->> 'revision_catalogo_politicas')::numeric
       OR catalogo.huella_sha256 IS DISTINCT FROM
          d ->> 'catalogo_politicas_huella_sha256' THEN
        RETURN;
    END IF;
    SELECT coalesce(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_object(
                   'referencia', p.politica_ref,
                   'huella_sha256', p.huella_sha256
               ) ORDER BY p.politica_ref COLLATE "C"
           ), '[]'::jsonb),
           coalesce(pg_catalog.jsonb_agg(
               p.politica_ref ORDER BY p.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO manifiesto, referencias
      FROM vec_autorizacion.politica_restrictiva_actual AS actual
      JOIN vec_autorizacion.politica_restrictiva AS p
        ON p.politica_id = actual.politica_id
       AND p.politica_ref = actual.politica_ref;
    IF manifiesto IS DISTINCT FROM d -> 'politicas_evaluadas' THEN
        RETURN;
    END IF;

    IF vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
           v, (d ->> 'emitida_en')::timestamptz,
           (d ->> 'valida_hasta')::timestamptz,
           (d ->> 'emitida_en')::timestamptz
       ) IS NOT TRUE THEN
        RETURN;
    END IF;

    -- Segunda acreditacion: mismos 17 valores, despues de todos los locks.
    instante :=
      vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        v ->> 'registro_contexto_ref', v ->> 'contexto_actor_esquema',
        v ->> 'contexto_actor_huella_sha256',
        v ->> 'manifiesto_procedencia_huella_sha256',
        v ->> 'autoridad_efectiva', v ->> 'cuenta_ref',
        (v ->> 'contexto_actor_cuenta_version')::numeric,
        v ->> 'principal_id', p_persona_version,
        v ->> 'perfil_activo_ref', p_perfil_version,
        v ->> 'contexto_actor_ref',
        (v ->> 'contexto_actor_version')::numeric,
        v ->> 'metodo_observado', v ->> 'garantia_observada',
        (d ->> 'emitida_en')::timestamptz,
        (d ->> 'valida_hasta')::timestamptz
      );
    IF instante IS NULL OR instante < primera_acreditacion
       OR vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
              v, (d ->> 'emitida_en')::timestamptz,
              (d ->> 'valida_hasta')::timestamptz, instante
          ) IS NOT TRUE THEN
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM vec_autorizacion.motivo_v2_catalogo_publicado AS mc
          JOIN vec_autorizacion.motivo_v2_entrada AS me
            ON me.catalogo_id = mc.catalogo_id
           AND me.catalogo_version = mc.catalogo_version
         WHERE mc.catalogo_id = rm ->> 'catalogo_id'
           AND mc.catalogo_version = (rm ->> 'catalogo_version')::integer
           AND mc.catalogo_huella_publicada_sha256 =
               rm ->> 'catalogo_huella_sha256'
           AND me.entrada_clave = rm ->> 'entrada_clave'
           AND mc.publicado_en <= instante
           AND me.vigente_desde <= instante
           AND (me.vigente_hasta IS NULL OR instante < me.vigente_hasta)
           AND NOT EXISTS (
               SELECT 1 FROM vec_autorizacion.motivo_v2_retirada AS mr
                WHERE mr.catalogo_id = mc.catalogo_id
                  AND mr.catalogo_version = mc.catalogo_version
           )
    ) THEN RETURN; END IF;

    IF instante < (d ->> 'emitida_en')::timestamptz
       OR instante >= (d ->> 'valida_hasta')::timestamptz THEN
        RETURN;
    END IF;

    -- Para concesion, SQL vuelve a exigir estado/vigencia ejecutables. Una
    -- denegacion conserva la instantanea exacta, pero nunca recibe cda_.
    IF (d ->> 'concedida')::boolean AND (
        asignacion.documento ->> 'estado' IS DISTINCT FROM 'activa'
        OR instante < (asignacion.documento ->> 'vigente_desde')::timestamptz
        OR instante >= (asignacion.documento ->> 'vigente_hasta')::timestamptz
        OR rol.documento ->> 'estado' IS DISTINCT FROM 'publicada'
        OR rol.publicada_en > instante
        OR rol.estado IS DISTINCT FROM 'habilitada'
    ) THEN RETURN; END IF;

    -- Replay exacto. Colision de referencia o huella no se degrada a exito.
    IF EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_concedida_contexto_actor_v3_externa x
         WHERE x.decision_ref = d ->> 'decision_ref'
            OR x.huella_decision_sha256 = huella
    ) THEN
        SELECT x.registrada_en
          INTO STRICT instante
          FROM vec_autorizacion.decision_concedida_contexto_actor_v3_externa x
         WHERE x.decision_ref = d ->> 'decision_ref'
           AND x.huella_decision_sha256 = huella
           AND x.decision_canonica = p_decision_canonica
           AND x.motivo_canonico = p_motivo_canonico
           AND x.persona_version = p_persona_version
           AND x.perfil_version = p_perfil_version;
        RETURN QUERY SELECT true, 'concedida'::text, huella, instante;
        RETURN;
    ELSIF EXISTS (
        SELECT 1 FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa x
         WHERE x.decision_ref = d ->> 'decision_ref'
            OR x.huella_decision_sha256 = huella
    ) THEN
        SELECT x.codigo, x.registrada_en
          INTO STRICT codigo, instante
          FROM vec_autorizacion.decision_denegada_contexto_actor_v3_externa x
         WHERE x.decision_ref = d ->> 'decision_ref'
           AND x.huella_decision_sha256 = huella
           AND x.decision_canonica = p_decision_canonica
           AND x.motivo_canonico = p_motivo_canonico
           AND x.persona_version = p_persona_version
           AND x.perfil_version = p_perfil_version;
        RETURN QUERY SELECT false, codigo, huella, instante;
        RETURN;
    END IF;

    IF (d ->> 'concedida')::boolean THEN
        INSERT INTO vec_autorizacion.decision_concedida_contexto_actor_v3_externa(
            decision_ref, huella_decision_sha256, decision_canonica,
            documento, motivo_canonico, motivo_catalogo_id,
            motivo_catalogo_version, motivo_entrada_clave,
            registro_contexto_ref, contexto_actor_huella_sha256,
            manifiesto_procedencia_huella_sha256, persona_version,
            perfil_version, asignacion_ref,
            version_rol_ref, control_vigencia_version_rol_revision,
            revision_catalogo_politicas, emitida_en,
            valida_hasta, registrada_en
        ) VALUES (
            d ->> 'decision_ref', huella, p_decision_canonica, d,
            p_motivo_canonico, rm ->> 'catalogo_id',
            (rm ->> 'catalogo_version')::integer,
            rm ->> 'entrada_clave', v ->> 'registro_contexto_ref',
            v ->> 'contexto_actor_huella_sha256',
            v ->> 'manifiesto_procedencia_huella_sha256',
            p_persona_version, p_perfil_version,
            d ->> 'asignacion_ref', d ->> 'version_rol_ref',
            (d ->> 'control_vigencia_version_rol_revision')::numeric,
            (d ->> 'revision_catalogo_politicas')::numeric,
            (d ->> 'emitida_en')::timestamptz,
            (d ->> 'valida_hasta')::timestamptz, instante
        );
        RETURN QUERY SELECT true, 'concedida'::text, huella, instante;
    ELSE
        INSERT INTO vec_autorizacion.decision_denegada_contexto_actor_v3_externa(
            decision_ref, huella_decision_sha256, decision_canonica,
            documento, motivo_canonico, motivo_catalogo_id,
            motivo_catalogo_version, motivo_entrada_clave,
            registro_contexto_ref, contexto_actor_huella_sha256,
            manifiesto_procedencia_huella_sha256, persona_version,
            perfil_version, asignacion_ref,
            version_rol_ref, control_vigencia_version_rol_revision,
            revision_catalogo_politicas, codigo, emitida_en, valida_hasta,
            registrada_en
        ) VALUES (
            d ->> 'decision_ref', huella, p_decision_canonica, d,
            p_motivo_canonico, rm ->> 'catalogo_id',
            (rm ->> 'catalogo_version')::integer,
            rm ->> 'entrada_clave', v ->> 'registro_contexto_ref',
            v ->> 'contexto_actor_huella_sha256',
            v ->> 'manifiesto_procedencia_huella_sha256',
            p_persona_version, p_perfil_version,
            d ->> 'asignacion_ref', d ->> 'version_rol_ref',
            (d ->> 'control_vigencia_version_rol_revision')::numeric,
            (d ->> 'revision_catalogo_politicas')::numeric,
            d ->> 'codigo', (d ->> 'emitida_en')::timestamptz,
            (d ->> 'valida_hasta')::timestamptz, instante
        );
        RETURN QUERY SELECT false, d ->> 'codigo', huella, instante;
    END IF;
END
$function$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric) FROM PUBLIC;
DO $verificar_literal$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.registrar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '7bab35a1d21c0ab31125d3de275ec9bdda0c158219188775eb416c85d77f71c6'
 THEN RAISE EXCEPTION 'AUT78: definición final divergente' USING ERRCODE='55000'; END IF;
END $verificar_literal$;

CREATE OR REPLACE FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(p_decision_canonica bytea, p_motivo_canonico bytea, p_persona_version numeric, p_perfil_version numeric)
 RETURNS timestamp with time zone
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE
    d jsonb;
    v jsonb;
    motivo jsonb;
    rm jsonb;
    asignacion record;
    rol record;
    catalogo record;
    manifiesto jsonb;
    ahora timestamptz(6);
    acreditada_en timestamptz(6);
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR p_decision_canonica IS NULL OR p_motivo_canonico IS NULL
       OR p_persona_version IS NULL OR p_perfil_version IS NULL
       OR pg_catalog.scale(p_persona_version) <> 0
       OR pg_catalog.scale(p_perfil_version) <> 0
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric THEN
        RETURN NULL;
    END IF;
    BEGIN
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
        motivo := pg_catalog.convert_from(p_motivo_canonico, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RETURN NULL;
    END;
    IF vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
       OR vec_autorizacion.decision_contexto_actor_v3_canonica(d)
          IS DISTINCT FROM p_decision_canonica
       OR vec_autorizacion.motivo_contexto_actor_v3_canonico(motivo)
          IS DISTINCT FROM p_motivo_canonico
       OR pg_catalog.encode(
           pg_catalog.sha256(p_motivo_canonico), 'hex'
       ) IS DISTINCT FROM d ->> 'motivo_huella_sha256'
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.decision_concedida_contexto_actor_v3_externa e
            WHERE e.decision_ref = d ->> 'decision_ref'
              AND e.decision_canonica = p_decision_canonica
              AND e.motivo_canonico = p_motivo_canonico
              AND e.persona_version = p_persona_version
              AND e.perfil_version = p_perfil_version
       ) THEN
        RETURN NULL;
    END IF;
    v := d -> 'vinculo_autenticacion_actor';
    rm := motivo -> 'referencia';

    -- Misma barrera y mismo orden de autoridades que el registrador nominal.
    PERFORM pg_catalog.pg_advisory_xact_lock_shared(
        pg_catalog.hashtextextended(
            'vec_autorizacion:migracion:registro_contexto_actor_v3:000005', 0
        )
    );
    acreditada_en :=
      vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        v ->> 'registro_contexto_ref', v ->> 'contexto_actor_esquema',
        v ->> 'contexto_actor_huella_sha256',
        v ->> 'manifiesto_procedencia_huella_sha256',
        v ->> 'autoridad_efectiva', v ->> 'cuenta_ref',
        (v ->> 'contexto_actor_cuenta_version')::numeric,
        v ->> 'principal_id', p_persona_version,
        v ->> 'perfil_activo_ref', p_perfil_version,
        v ->> 'contexto_actor_ref',
        (v ->> 'contexto_actor_version')::numeric,
        v ->> 'metodo_observado', v ->> 'garantia_observada',
        (d ->> 'emitida_en')::timestamptz,
        (d ->> 'valida_hasta')::timestamptz
      );
    IF acreditada_en IS NULL THEN
        RETURN NULL;
    END IF;

    PERFORM ultima_secuencia
      FROM vec_autorizacion.motivo_v2_checkpoint_origen
     WHERE control_id = true
     FOR SHARE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;
    SELECT a.asignacion_ref, a.principal_id, a.version_rol_ref,
           a.huella_sha256, a.documento
      INTO asignacion
      FROM vec_autorizacion.asignacion_perfil_actual_externa actual
      JOIN vec_autorizacion.asignacion_perfil_externa a
        ON a.perfil_activo_ref = actual.perfil_activo_ref
       AND a.asignacion_ref = actual.asignacion_ref
     WHERE actual.perfil_activo_ref = d ->> 'perfil_activo_ref'
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR asignacion.asignacion_ref <> d ->> 'asignacion_ref'
       OR asignacion.principal_id <> d ->> 'principal_id'
       OR asignacion.version_rol_ref <> d ->> 'version_rol_ref'
       OR asignacion.huella_sha256 <> d ->> 'asignacion_huella_sha256'
       OR asignacion.documento ->> 'estado' <> 'activa' THEN
        RETURN NULL;
    END IF;

    SELECT r.huella_sha256, r.publicada_en, r.documento,
           c.version_rol_ref, c.revision, c.estado,
           c.huella_sha256 AS control_huella
      INTO rol
      FROM vec_autorizacion.version_rol r
      JOIN vec_autorizacion.control_vigencia_version_rol_actual actual
        ON actual.version_rol_ref = r.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol c
        ON c.version_rol_ref = actual.version_rol_ref
       AND c.revision = actual.revision
     WHERE r.version_rol_ref = asignacion.version_rol_ref
     FOR UPDATE OF actual;
    IF NOT FOUND
       OR rol.huella_sha256 <> d ->> 'version_rol_huella_sha256'
       OR rol.version_rol_ref <>
          d ->> 'control_vigencia_version_rol_ref'
       OR rol.revision <>
          (d ->> 'control_vigencia_version_rol_revision')::numeric
       OR rol.control_huella <>
          d ->> 'control_vigencia_version_rol_huella_sha256'
       OR rol.documento ->> 'estado' <> 'publicada'
       OR rol.estado <> 'habilitada' THEN
        RETURN NULL;
    END IF;

    SELECT revision, huella_sha256
      INTO catalogo
      FROM vec_autorizacion.control_catalogo_politicas
     WHERE control_id = true
     FOR UPDATE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;
    SELECT coalesce(pg_catalog.jsonb_agg(
               pg_catalog.jsonb_build_object(
                   'referencia', p.politica_ref,
                   'huella_sha256', p.huella_sha256
               ) ORDER BY p.politica_ref COLLATE "C"
           ), '[]'::jsonb)
      INTO manifiesto
      FROM vec_autorizacion.politica_restrictiva_actual actual
      JOIN vec_autorizacion.politica_restrictiva p
        ON p.politica_id = actual.politica_id
       AND p.politica_ref = actual.politica_ref;
    IF catalogo.revision <>
          (d ->> 'revision_catalogo_politicas')::numeric
       OR catalogo.huella_sha256 <>
          d ->> 'catalogo_politicas_huella_sha256'
       OR manifiesto IS DISTINCT FROM d -> 'politicas_evaluadas' THEN
        RETURN NULL;
    END IF;

    ahora := pg_catalog.clock_timestamp();
    IF ahora < (d ->> 'emitida_en')::timestamptz
       OR ahora >= (d ->> 'valida_hasta')::timestamptz
       OR ahora < (asignacion.documento ->> 'vigente_desde')::timestamptz
       OR ahora >= (asignacion.documento ->> 'vigente_hasta')::timestamptz
       OR rol.publicada_en > ahora
       OR vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
              v, (d ->> 'emitida_en')::timestamptz,
              (d ->> 'valida_hasta')::timestamptz, ahora
          ) IS NOT TRUE
       OR NOT EXISTS (
           SELECT 1
             FROM vec_autorizacion.motivo_v2_catalogo_publicado mc
             JOIN vec_autorizacion.motivo_v2_entrada me
               ON me.catalogo_id = mc.catalogo_id
              AND me.catalogo_version = mc.catalogo_version
            WHERE mc.catalogo_id = rm ->> 'catalogo_id'
              AND mc.catalogo_version =
                  (rm ->> 'catalogo_version')::integer
              AND mc.catalogo_huella_publicada_sha256 =
                  rm ->> 'catalogo_huella_sha256'
              AND me.entrada_clave = rm ->> 'entrada_clave'
              AND mc.publicado_en <= ahora
              AND me.vigente_desde <= ahora
              AND (me.vigente_hasta IS NULL OR ahora < me.vigente_hasta)
              AND NOT EXISTS (
                  SELECT 1
                    FROM vec_autorizacion.motivo_v2_retirada retirada
                   WHERE retirada.catalogo_id = mc.catalogo_id
                     AND retirada.catalogo_version = mc.catalogo_version
              )
       ) THEN
        RETURN NULL;
    END IF;
    RETURN ahora;
EXCEPTION
    WHEN data_exception OR invalid_text_representation
      OR datetime_field_overflow OR no_data_found OR too_many_rows THEN
        RETURN NULL;
END
$function$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric) FROM PUBLIC;
DO $verificar_literal$ BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef('vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(bytea,bytea,numeric,numeric)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM '2ab399528c3acc6fda69c6dc50036c977769f419b8b8f8926698e2f3806983cb'
 THEN RAISE EXCEPTION 'AUT78: definición final divergente' USING ERRCODE='55000'; END IF;
END $verificar_literal$;


CREATE FUNCTION vec_autorizacion.resolver_motivo_meritos_externo_v1(
 p_catalogo_id text,p_catalogo_version integer,p_huella text,p_entrada text,p_instante timestamptz)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 IF vec_autorizacion.login_candidato_externo_v1(
  'vec_externo_meritos_motivos_autorizacion','vec_autorizacion_motivos_meritos_externo') IS NOT TRUE
 OR p_catalogo_id NOT IN ('motivos_autorizacion_meritos_cartera','motivos_meritos_declaracion')
 THEN RAISE EXCEPTION 'AUT78: motivo V3 externo denegado' USING ERRCODE='42501';END IF;
 RETURN vec_autorizacion.resolver_motivo_autorizacion_v2_historico(
  p_catalogo_id,p_catalogo_version,p_huella,p_entrada,p_instante);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.resolver_motivo_meritos_externo_v1(
 text,integer,text,text,timestamptz) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.resolver_motivo_meritos_externo_v1(
 text,integer,text,text,timestamptz) TO vec_autorizacion_motivos_meritos_externo;

CREATE FUNCTION vec_autorizacion.decision_meritos_externa_exacta_v1(d jsonb,m jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE v jsonb;
BEGIN
 v:=d->'vinculo_autenticacion_actor';
 IF current_user<>'vec_autorizacion_propietario'
 OR vec_autorizacion.decision_contexto_actor_v3_valida(d) IS NOT TRUE
 OR d->>'accion' NOT IN ('meritos.cartera.retirar_propio','meritos.hecho.declarar')
 OR d->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho'
 OR d->>'finalidad' IS DISTINCT FROM (CASE d->>'accion'
   WHEN 'meritos.hecho.declarar' THEN 'declaracion_hecho_propio'
   ELSE 'gestion_cartera_propia' END)
 OR v->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR v->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR d->>'principal_id' IS DISTINCT FROM v->>'principal_id'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM v->>'perfil_activo_ref'
 OR vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(
  v->>'cuenta_ref',d->>'principal_id',d->>'perfil_activo_ref') IS NOT TRUE
 OR m#>>'{referencia,catalogo_id}' IS DISTINCT FROM (CASE d->>'accion'
   WHEN 'meritos.hecho.declarar' THEN 'motivos_meritos_declaracion'
   ELSE 'motivos_autorizacion_meritos_cartera' END)
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r
  WHERE r.version_rol_ref=d->>'version_rol_ref'
   AND r.rol_id='persona_meritos_cartera'
   AND ((d->>'accion'='meritos.hecho.declarar' AND
     (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
      FROM jsonb_array_elements(r.documento->'concesiones') x)
      =ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[])
    OR (d->>'accion'='meritos.cartera.retirar_propio' AND
     (SELECT array_agg(x.value->>'accion' ORDER BY x.value->>'accion')
      FROM jsonb_array_elements(r.documento->'concesiones') x) IN
      (ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[],
       ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[]))))
 THEN RETURN false;END IF;
 -- La denegación conserva listas canónicas de la decisión V3, limitadas
 -- al contrato nominal. No exige campos que nunca llegaron a concederse.
 IF d->>'concedida'='true' THEN
  RETURN d->'campos_permitidos' IS NOT DISTINCT FROM (CASE d->>'accion'
    WHEN 'meritos.hecho.declarar' THEN '["declarante_ref","hecho","recibo","version"]'::jsonb
    ELSE '["disponibilidad","recibo"]'::jsonb END)
   AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb;
 END IF;
 RETURN d->>'concedida'='false'
  AND (d->'campos_permitidos' <@ (CASE d->>'accion'
    WHEN 'meritos.hecho.declarar' THEN '["declarante_ref","hecho","recibo","version"]'::jsonb
    ELSE '["disponibilidad","recibo"]'::jsonb END))
  AND (d->'obligaciones' <@ '["auditar"]'::jsonb);
EXCEPTION WHEN data_exception OR invalid_text_representation THEN RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.decision_meritos_externa_exacta_v1(jsonb,jsonb)
 FROM PUBLIC;

CREATE FUNCTION vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS TABLE(concedida boolean,codigo text,decision_huella_sha256 text,
 registrada_en timestamptz,revalidada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE d jsonb;m jsonb;registro record;viva timestamptz;
BEGIN
 IF NOT (vec_autorizacion.login_candidato_externo_v1(
  'vec_externo_meritos_registro_autorizacion','vec_autorizacion_registro_meritos_externo') IS TRUE
  OR (session_user='vec_externo_meritos_cartera'
   AND pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
  OR (session_user='vec_externo_meritos_operacion'
   AND pg_has_role(session_user,'vec_meritos_ejecutor','MEMBER')
   AND pg_has_role(session_user,'vec_meritos_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)))
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
 OR p_motivo IS NULL OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'AUT78: registro V3 externo denegado' USING ERRCODE='42501';END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;m:=convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AUT78: material V3 inválido' USING ERRCODE='22023';END;
 IF vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
 OR vec_autorizacion.decision_meritos_externa_exacta_v1(d,m) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT78: decisión M4 divergente' USING ERRCODE='42501';END IF;
 SELECT * INTO registro FROM vec_autorizacion.registrar_decision_contexto_actor_v3_meritos_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF NOT FOUND THEN RETURN;END IF;
 -- La fila negativa permite confirmar el asiento durable. No revalida
 -- permiso ejecutable ni produce material de capacidad para una denegación.
 IF registro.concedida IS FALSE THEN
  RETURN QUERY SELECT false,registro.codigo,registro.decision_huella_sha256,
   registro.registrada_en,NULL::timestamptz;
  RETURN;
 END IF;
 IF registro.concedida IS NOT TRUE THEN RETURN;END IF;
 viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
 IF viva IS NULL THEN RETURN;END IF;
 RETURN QUERY SELECT registro.concedida,registro.codigo,registro.decision_huella_sha256,
  registro.registrada_en,viva;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(
 bytea,bytea,numeric,numeric) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(
 bytea,bytea,numeric,numeric) TO vec_autorizacion_registro_meritos_externo,
 vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(
 p_decision bytea,p_motivo bytea,p_persona_version numeric,p_perfil_version numeric)
RETURNS timestamptz LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE d jsonb;m jsonb;
BEGIN
 IF NOT ((session_user='vec_externo_meritos_cartera'
   AND pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1)
  OR (session_user='vec_externo_meritos_operacion'
   AND pg_has_role(session_user,'vec_meritos_ejecutor','MEMBER')
   AND pg_has_role(session_user,'vec_meritos_externo','MEMBER')
   AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_ejecutor'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)
   AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
    AND m.roleid='vec_meritos_externo'::regrole AND m.inherit_option
    AND NOT m.set_option AND NOT m.admin_option)))
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_decision IS NULL OR p_motivo IS NULL
 THEN RAISE EXCEPTION 'AUT78: revalidación V3 externa denegada' USING ERRCODE='42501';END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb;m:=convert_from(p_motivo,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AUT78: material V3 inválido' USING ERRCODE='22023';END;
 IF vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision
 OR vec_autorizacion.motivo_contexto_actor_v3_canonico(m) IS DISTINCT FROM p_motivo
 OR vec_autorizacion.decision_meritos_externa_exacta_v1(d,m) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT78: revalidación M4 divergente' USING ERRCODE='42501';END IF;
 RETURN vec_autorizacion.revalidar_decision_contexto_actor_v3_meritos_interna(
  p_decision,p_motivo,p_persona_version,p_perfil_version);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(
 bytea,bytea,numeric,numeric) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_decision_meritos_externo_v3_viva(
 bytea,bytea,numeric,numeric) TO vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion
 TO vec_autorizacion_registro_meritos_externo,vec_autorizacion_motivos_meritos_externo,
 vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion.cotejar_cartera_meritos_actual_v1(p_orden jsonb,p_vinculo jsonb)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE asignacion record;rol record;control record;catalogo record;
 politicas jsonb;concesion jsonb;ahora timestamptz(6);accion text;persona text;
 campos jsonb;
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR jsonb_typeof(p_vinculo) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
   JOIN pg_roles g ON g.oid=m.roleid
   WHERE l.rolname=session_user AND l.rolname='vec_externo_meritos_cartera'
    AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL AND g.rolname='vec_meritos_cartera_externo'
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 THEN RETURN false; END IF;
 accion:=p_orden->>'accion';persona:=p_orden->>'persona_ref';
 campos:=CASE accion
  WHEN 'meritos.cartera.consultar_propia' THEN
   '["denominacion","disponibilidad","hecho_ref","motivos_retirada","retirada","tipo","total","version_hecho"]'::jsonb
  WHEN 'meritos.cartera.retirar_propio' THEN '["disponibilidad","recibo"]'::jsonb
  ELSE NULL END;
 IF campos IS NULL OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'tipo_recurso' IS DISTINCT FROM
  (CASE WHEN accion='meritos.cartera.consultar_propia' THEN 'cartera_personal' ELSE 'hecho' END)
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'gestion_cartera_propia'
 OR p_orden->>'canal' IS DISTINCT FROM 'externa_personal'
 OR (accion='meritos.cartera.consultar_propia' AND
  (p_orden->>'idioma' IS NULL OR p_orden->>'idioma' NOT IN ('es','en')))
 OR p_orden->'campos_permitidos' IS DISTINCT FROM campos
 OR p_orden->>'recurso_ref' IS DISTINCT FROM
  (CASE WHEN accion='meritos.cartera.consultar_propia' THEN 'cartera:'||persona
  ELSE p_orden->>'hecho_ref' END)
 OR persona IS DISTINCT FROM p_vinculo->>'principal_id'
 OR p_orden->>'perfil_activo_ref' IS DISTINCT FROM p_vinculo->>'perfil_activo_ref'
 OR p_vinculo->>'superficie' IS DISTINCT FROM 'externa_personal'
 OR p_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR p_orden->>'emitida_en' IS NULL OR p_orden->>'valida_hasta' IS NULL
 OR jsonb_typeof(p_orden->'politicas_evaluadas') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 ahora:=clock_timestamp();
 IF (p_orden->>'emitida_en')::timestamptz>ahora
 OR (p_orden->>'valida_hasta')::timestamptz<=ahora
 OR (p_orden->>'valida_hasta')::timestamptz>ahora+interval '30 seconds'
 OR vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
   p_vinculo,(p_orden->>'emitida_en')::timestamptz,
   (p_orden->>'valida_hasta')::timestamptz,ahora) IS NOT TRUE
 THEN RETURN false; END IF;
 SELECT x.* INTO asignacion FROM vec_autorizacion.asignacion_perfil_actual_externa ap
 JOIN vec_autorizacion.asignacion_perfil_externa x
 ON x.perfil_activo_ref=ap.perfil_activo_ref AND x.asignacion_ref=ap.asignacion_ref
 WHERE ap.perfil_activo_ref=p_orden->>'perfil_activo_ref' FOR SHARE OF ap,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO rol FROM vec_autorizacion.version_rol x
 WHERE x.version_rol_ref=asignacion.version_rol_ref FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT x.* INTO control FROM vec_autorizacion.control_vigencia_version_rol_actual cp
 JOIN vec_autorizacion.control_vigencia_version_rol x
 ON x.version_rol_ref=cp.version_rol_ref AND x.revision=cp.revision
 WHERE cp.version_rol_ref=rol.version_rol_ref FOR SHARE OF cp,x;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT * INTO catalogo FROM vec_autorizacion.control_catalogo_politicas
 WHERE control_id IS TRUE FOR SHARE;
 IF NOT FOUND THEN RETURN false; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('referencia',p.politica_ref,
  'huella_sha256',p.huella_sha256) ORDER BY p.politica_ref),'[]'::jsonb)
 INTO politicas FROM vec_autorizacion.politica_restrictiva_actual pa
 JOIN vec_autorizacion.politica_restrictiva p
 ON p.politica_id=pa.politica_id AND p.politica_ref=pa.politica_ref;
 IF asignacion.principal_id IS DISTINCT FROM persona
 OR asignacion.asignacion_ref IS DISTINCT FROM p_orden->>'asignacion_ref'
 OR asignacion.version IS DISTINCT FROM (p_orden->>'asignacion_version')::numeric
 OR asignacion.huella_sha256 IS DISTINCT FROM p_orden->>'asignacion_huella_sha256'
 OR asignacion.documento->>'estado' IS DISTINCT FROM 'activa'
 OR ahora<(asignacion.documento->>'vigente_desde')::timestamptz
 OR ahora>=(asignacion.documento->>'vigente_hasta')::timestamptz
 OR rol.rol_id IS DISTINCT FROM 'persona_meritos_cartera'
 OR rol.version_rol_ref IS DISTINCT FROM p_orden->>'version_rol_ref'
 OR rol.version IS DISTINCT FROM (p_orden->>'version_rol_version')::numeric
 OR rol.huella_sha256 IS DISTINCT FROM p_orden->>'version_rol_huella_sha256'
 OR rol.documento->>'estado' IS DISTINCT FROM 'publicada'
 OR jsonb_typeof(rol.documento->'concesiones') IS DISTINCT FROM 'array'
 OR ((SELECT array_agg(e.value->>'accion' ORDER BY e.value->>'accion')
     FROM jsonb_array_elements(rol.documento->'concesiones') e) IS DISTINCT FROM
    ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio']::text[]
  AND (SELECT array_agg(e.value->>'accion' ORDER BY e.value->>'accion')
     FROM jsonb_array_elements(rol.documento->'concesiones') e) IS DISTINCT FROM
    ARRAY['meritos.cartera.consultar_propia','meritos.cartera.retirar_propio','meritos.hecho.declarar']::text[])
 OR control.version_rol_ref IS DISTINCT FROM p_orden->>'control_version_rol_ref'
 OR control.revision IS DISTINCT FROM (p_orden->>'revision_control_version_rol')::numeric
 OR control.huella_sha256 IS DISTINCT FROM p_orden->>'control_version_rol_huella_sha256'
 OR control.estado IS DISTINCT FROM 'habilitada'
 OR catalogo.revision IS DISTINCT FROM (p_orden->>'revision_catalogo_politicas')::numeric
 OR catalogo.huella_sha256 IS DISTINCT FROM p_orden->>'catalogo_politicas_huella_sha256'
 OR politicas IS DISTINCT FROM p_orden->'politicas_evaluadas'
 OR jsonb_typeof(asignacion.documento->'ambitos') IS DISTINCT FROM 'array'
 OR jsonb_array_length(asignacion.documento->'ambitos')<>1
 OR vec_autorizacion.ambitos_positivos_validos(asignacion.documento) IS NOT TRUE
 OR asignacion.documento->'ambitos' @> jsonb_build_array(
    jsonb_build_object('clave','persona_ref','valores',jsonb_build_array(persona))) IS NOT TRUE
 THEN RETURN false; END IF;
 SELECT e.value INTO concesion FROM jsonb_array_elements(rol.documento->'concesiones') e
 WHERE e.value->>'accion'=accion;
 IF NOT FOUND OR (SELECT count(*) FROM jsonb_array_elements(rol.documento->'concesiones') e
   WHERE e.value->>'accion'=accion)<>1
 OR concesion->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR concesion->>'tipo_recurso' IS DISTINCT FROM p_orden->>'tipo_recurso'
 OR concesion->'finalidades' IS DISTINCT FROM '["gestion_cartera_propia"]'::jsonb
 OR concesion->'campos_permitidos' IS DISTINCT FROM campos
 OR concesion->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR coalesce(concesion->'obligaciones','[]'::jsonb)
  IS DISTINCT FROM '["auditar"]'::jsonb
 THEN RETURN false; END IF;
 RETURN true;
EXCEPTION WHEN data_exception OR invalid_text_representation OR numeric_value_out_of_range THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.cotejar_cartera_meritos_actual_v1(jsonb,jsonb)
 TO vec_autorizacion_atestada_v3_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion.revalidar_sesion_cartera_meritos_v1(
 jsonb,timestamptz,timestamptz,timestamptz)
 TO vec_autorizacion_atestada_v3_propietario;
RESET ROLE;
COMMIT;
