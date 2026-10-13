\set ON_ERROR_STOP on
-- AD251: un asiento positivo común y una lectura ligada a la misma transacción.
-- Orden causal AUT82, CT208, B109, P47, AD249; no concede perfiles al instalar.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion.cotejar_lectura_auditoria_rrhh_actual_v1(jsonb,text,text)') IS NULL
 OR to_regprocedure('vec_contratacion_temporal.buscar_objetivos_auditoria_rrhh_v1(text,text,text,integer)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.buscar_objetivos_auditoria_rrhh_v1(text,text,text,text,integer)') IS NULL
 OR to_regprocedure('vec_personal.buscar_objetivos_persona_auditoria_rrhh_v1(text,text,text,integer)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_bolsa_interna_v1(bytea,bytea,jsonb)') IS NULL
 THEN RAISE EXCEPTION 'AD251: dependencias o preimagen incompatibles' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO h
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF h IS DISTINCT FROM '1a299eefbc2936a06b414d47defbb17b213fec7a29f2fa6eec448a008727d8ba'
 THEN RAISE EXCEPTION 'AD251: familia auditora divergente: %',h USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
DO $familia$
DECLARE anterior text;nueva text;marca constant text:='''lectura_preferencias_bolsas''::text';
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
 '1a299eefbc2936a06b414d47defbb17b213fec7a29f2fa6eec448a008727d8ba'
 OR (length(anterior)-length(replace(anterior,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AD251: preimagen de familia incompatible' USING ERRCODE='55000'; END IF;
 nueva:=replace(anterior,marca,marca||',''lectura_auditoria_rrhh''::text');
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_lectura_rrhh_formato_v1 CHECK (
 tipo_registro<>'lectura_auditoria_rrhh' OR (
  intento_ref ~ '^lectura_[0-9a-f]{32}$' AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
  AND contexto_sha256 ~ '^[0-9a-f]{64}$' AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
  AND autenticacion_sha256 ~ '^[0-9a-f]{64}$' AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
  AND lectura_revision_permisos BETWEEN 1 AND 18446744073709551615::numeric
  AND lectura_instantanea_sha256 ~ '^[0-9a-f]{64}$'
  AND modulo_id='auditoria' AND proceso='vec-server' AND canal='interna_corporativa'
  AND accion='vec.auditoria.consultar' AND finalidad_ref='consulta_auditoria_rrhh'
  AND resultado='obtenida'
  AND motivo_ref ~ '^motivos_autorizacion_auditoria:2:[a-z][a-z0-9_.:-]{0,127}$'
  AND correlacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'));

CREATE TABLE vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1 (
 auditoria_ref text PRIMARY KEY REFERENCES vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref),
 transaccion_origen xid8 NOT NULL,
 operacion text NOT NULL CHECK(operacion IN('opciones','buscar','consultar')),
 fuente text NOT NULL CHECK(fuente IN('ct','bolsa','persona')),
 organizacion_ref text NOT NULL CHECK(organizacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 unidad_ref text NOT NULL CHECK(unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 ambito_ref text NOT NULL CHECK(ambito_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 recurso_ref text NOT NULL,
 filtro_sha256 text NOT NULL CHECK(filtro_sha256 ~ '^[0-9a-f]{64}$'),
 filtro jsonb NOT NULL CHECK(jsonb_typeof(filtro)='object'),
 motivo_catalogo_ref text NOT NULL,
 motivo_catalogo_sha256 text NOT NULL CHECK(motivo_catalogo_sha256 ~ '^[0-9a-f]{64}$'),
 motivo_consulta text NOT NULL CHECK(octet_length(motivo_consulta) BETWEEN 3 AND 500 AND motivo_consulta !~ '[[:cntrl:]]'),
 detalle_sha256 text NOT NULL CHECK(detalle_sha256 ~ '^[0-9a-f]{64}$')
);
CREATE INDEX detalle_lectura_rrhh_org_objetivo_v1
 ON vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1(
  organizacion_ref,fuente,recurso_ref,auditoria_ref);
CREATE INDEX auditoria_consumo_v3_rrhh_recurso_fecha_v1
 ON vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  recurso_ref,registrada_en DESC,auditoria_ref DESC);
REVOKE ALL ON vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1 FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1 FORCE ROW LEVEL SECURITY;
CREATE POLICY detalle_lectura_rrhh_propietario ON vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1
 TO vec_autorizacion_atestada_v3_propietario USING(true) WITH CHECK(true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1
 FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1
 FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();

CREATE FUNCTION vec_autorizacion_atestada_v3.login_lectura_auditoria_rrhh_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
  SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid
  JOIN pg_roles g ON g.oid=m.roleid
  WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit
   AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
   AND l.rolconfig IS NULL
   AND g.rolname='vec_contratacion_temporal_consultor_rrhh'
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND g.rolconfig IS NULL AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
   AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)));
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_lectura_auditoria_rrhh_valido_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(
 p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
DECLARE
 v_contexto jsonb;v_vinculo jsonb;v_sesion record;v_filtro jsonb;
 v_claves constant text[]:=ARRAY[
  'intento_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256',
  'autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id',
  'recurso_ref','organizacion_ref','unidad_ref','ambito_ref','fuente','finalidad_ref','campos_permitidos',
  'resultado','motivo_ref','proceso','canal','correlacion_ref',
  'lectura_revision_permisos','lectura_instantanea_sha256',
  'asignacion_ref','asignacion_version','asignacion_huella_sha256',
  'version_rol_ref','version_rol_version','version_rol_huella_sha256',
  'control_version_rol_ref','revision_control_version_rol',
  'control_version_rol_huella_sha256','revision_catalogo_politicas',
  'catalogo_politicas_huella_sha256','politicas_evaluadas','emitida_en','valida_hasta',
  'operacion','filtro_sha256','motivo_catalogo_ref','motivo_catalogo_sha256','motivo_consulta'];
 v_campos_filtro constant text[]:=ARRAY[
  'fuente','recurso_ref','actor_ref','desde','hasta','limite','antes_en',
  'antes_fuente','antes_id','finalidad_ref','motivo_ref','motivo_consulta',
  'autor_lectura_ref','busqueda','despues_ref'];
 v_clave text;v_material bytea;v_material_sha text;v_anterior text;v_secuencia numeric;
 v_instante timestamptz(6);v_huella text;v_ref text;v_canon text;v_filtro_sha text;
 v_detalle_sha text;v_entrada text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR vec_autorizacion_atestada_v3.login_lectura_auditoria_rrhh_valido_v1() IS NOT TRUE
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 OR p_vinculo_canonico IS NULL OR octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(v_claves,1)+1
 OR (p_orden ?& v_claves) IS NOT TRUE OR jsonb_typeof(p_orden->'filtro') IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'AD251: emisor de lectura inválido' USING ERRCODE='42501'; END IF;
 FOREACH v_clave IN ARRAY v_claves LOOP
  IF v_clave IN ('campos_permitidos','politicas_evaluadas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'array'
   THEN RAISE EXCEPTION 'AD251: orden inválida' USING ERRCODE='22023'; END IF;
  ELSIF v_clave IN ('asignacion_version','version_rol_version',
                    'revision_control_version_rol','revision_catalogo_politicas') THEN
   IF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'number'
   OR (p_orden->>v_clave) !~ '^[1-9][0-9]{0,19}$'
   THEN RAISE EXCEPTION 'AD251: versión inválida' USING ERRCODE='22023'; END IF;
  ELSIF jsonb_typeof(p_orden->v_clave) IS DISTINCT FROM 'string'
   OR octet_length(p_orden->>v_clave) NOT BETWEEN 1 AND 512
  THEN RAISE EXCEPTION 'AD251: orden inválida' USING ERRCODE='22023'; END IF;
 END LOOP;
 v_filtro:=p_orden->'filtro';
 IF (SELECT count(*) FROM jsonb_object_keys(v_filtro))<>array_length(v_campos_filtro,1)
 OR (v_filtro ?& v_campos_filtro) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: filtro inválido' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_campos_filtro LOOP
  IF jsonb_typeof(v_filtro->v_clave) IS DISTINCT FROM 'string'
  OR octet_length(v_filtro->>v_clave)>1024
  THEN RAISE EXCEPTION 'AD251: filtro inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_orden->>'intento_ref' !~ '^lectura_[0-9a-f]{32}$'
 OR p_orden->>'contexto_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'autenticacion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'lectura_revision_permisos' !~ '^[1-9][0-9]{0,19}$'
 OR p_orden->>'lectura_instantanea_sha256' !~ '^[0-9a-f]{64}$'
 OR (p_orden->>'lectura_revision_permisos')::numeric>18446744073709551615::numeric
 OR p_orden->>'accion' IS DISTINCT FROM 'vec.auditoria.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'auditoria'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'consulta_auditoria_rrhh'
 OR p_orden->>'resultado' IS DISTINCT FROM 'obtenida'
 OR p_orden->>'motivo_ref' IS DISTINCT FROM p_orden->>'motivo_catalogo_ref'
 OR p_orden->>'motivo_catalogo_ref' !~ '^motivos_autorizacion_auditoria:2:[a-z][a-z0-9_.:-]{0,127}$'
 OR p_orden->>'motivo_catalogo_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'correlacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_orden->>'filtro_sha256' !~ '^[0-9a-f]{64}$'
 OR p_orden->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_orden->>'unidad_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_orden->>'ambito_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 OR p_orden->>'fuente' NOT IN ('ct','bolsa','persona')
 OR p_orden->>'operacion' NOT IN ('opciones','buscar','consultar')
 OR p_orden->'campos_permitidos' IS DISTINCT FROM
 '["accion","actor_ref","antes","antes_sha256","canal","datos_disponibles","despues","despues_sha256","expediente_ref","finalidad_ref","fuente","id","modulo_id","motivo","ocurrido_en","perfil_ref","recibo_ref","resultado"]'::jsonb
 OR octet_length(p_orden->>'motivo_consulta') NOT BETWEEN 3 AND 500
 OR p_orden->>'motivo_consulta' ~ '[[:cntrl:]]'
 OR EXISTS(SELECT 1 FROM unnest(v_campos_filtro) AS k WHERE v_filtro->>k ~ '[[:cntrl:]]')
 OR v_filtro->>'fuente' IS DISTINCT FROM p_orden->>'fuente'
 OR v_filtro->>'recurso_ref' IS DISTINCT FROM p_orden->>'recurso_ref'
 OR v_filtro->>'finalidad_ref' IS DISTINCT FROM p_orden->>'finalidad_ref'
 OR v_filtro->>'motivo_ref' IS DISTINCT FROM p_orden->>'motivo_catalogo_ref'
 OR v_filtro->>'motivo_consulta' IS DISTINCT FROM p_orden->>'motivo_consulta'
 OR v_filtro->>'autor_lectura_ref' IS NULL
 OR v_filtro->>'limite' !~ '^[1-9][0-9]{0,2}$'
 OR (v_filtro->>'limite')::integer NOT BETWEEN 1 AND 100
 OR v_filtro->>'desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 OR v_filtro->>'hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 OR (v_filtro->>'antes_en'<>'' AND v_filtro->>'antes_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$')
 OR (v_filtro->>'antes_en'='' AND (v_filtro->>'antes_fuente'<>'' OR v_filtro->>'antes_id'<>''))
 OR (v_filtro->>'antes_en'<>'' AND (v_filtro->>'antes_fuente'<>p_orden->>'fuente' OR v_filtro->>'antes_id'=''))
 OR (p_orden->>'operacion'='buscar' AND (octet_length(v_filtro->>'busqueda')>80 OR octet_length(v_filtro->>'despues_ref')>512))
 OR (p_orden->>'operacion'<>'buscar' AND (v_filtro->>'busqueda'<>'' OR v_filtro->>'despues_ref'<>''))
 OR (p_orden->>'operacion'='opciones' AND (p_orden->>'recurso_ref'<>'auditoria:opciones' OR p_orden->>'fuente'<>'ct'))
 OR (p_orden->>'operacion'='buscar' AND p_orden->>'recurso_ref' IS DISTINCT FROM
  ('auditoria:objetivos:'||(p_orden->>'fuente')))
 THEN RAISE EXCEPTION 'AD251: contrato de lectura inválido' USING ERRCODE='22023'; END IF;
 v_canon:=array_to_string(ARRAY[
  'vec.auditoria.filtro.v1',v_filtro->>'fuente',v_filtro->>'recurso_ref',
  v_filtro->>'actor_ref',v_filtro->>'desde',v_filtro->>'hasta',v_filtro->>'limite',
  v_filtro->>'antes_en',v_filtro->>'antes_fuente',v_filtro->>'antes_id',
  v_filtro->>'finalidad_ref',v_filtro->>'motivo_ref'],E'\n')||E'\nvec.auditoria.lectura-sesion.v1\n'||
  v_filtro->>'motivo_consulta'||E'\n'||v_filtro->>'autor_lectura_ref';
 IF p_orden->>'operacion'='buscar' THEN
  v_canon:=v_canon||E'\nvec.auditoria.busqueda.v1\n'||v_filtro->>'busqueda'||E'\n'||v_filtro->>'despues_ref';
 END IF;
 v_filtro_sha:=encode(sha256(convert_to(v_canon,'UTF8')),'hex');
 IF v_filtro_sha IS DISTINCT FROM p_orden->>'filtro_sha256'
 THEN RAISE EXCEPTION 'AD251: huella de filtro divergente' USING ERRCODE='42501'; END IF;
 v_entrada:=regexp_replace(p_orden->>'motivo_catalogo_ref','^motivos_autorizacion_auditoria:2:','');
 IF vec_autorizacion.resolver_motivo_autorizacion_v2_actual(
  'motivos_autorizacion_auditoria',2,p_orden->>'motivo_catalogo_sha256',v_entrada) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: motivo no vigente' USING ERRCODE='42501'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
  v_vinculo:=convert_from(p_vinculo_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'AD251: vínculo de lectura inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
 OR jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
 OR v_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR v_vinculo->>'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
 OR v_vinculo->>'bloque_version' IS DISTINCT FROM '2'
 OR v_vinculo->>'principal_id' IS DISTINCT FROM v_contexto->>'principal_ref'
 OR v_vinculo->>'perfil_activo_ref' IS DISTINCT FROM v_contexto->>'perfil_activo_ref'
 OR v_vinculo->>'cuenta_ref' IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_vinculo->>'metodo_observado' IS DISTINCT FROM v_contexto->>'metodo'
 OR v_vinculo->>'garantia_observada' IS DISTINCT FROM v_contexto->>'garantia'
 OR v_vinculo->>'contexto_actor_ref' IS DISTINCT FROM v_contexto->>'contexto_actor_ref'
 OR v_vinculo->>'contexto_actor_version' IS DISTINCT FROM v_contexto->>'contexto_version'
 OR v_vinculo->>'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto->>'cuenta_version'
 OR v_vinculo->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref'
 OR v_vinculo->>'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_vinculo->>'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden->>'procedencia_sha256'
 OR v_vinculo->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref'
 OR v_vinculo->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref'
 OR v_vinculo->>'autenticacion_huella_sha256' IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_vinculo->>'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
 OR v_vinculo->>'superficie' IS DISTINCT FROM 'interna_corporativa'
 OR v_vinculo->>'cuenta_privilegiada' IS DISTINCT FROM 'false'
 OR encode(sha256(p_contexto_canonico),'hex') IS DISTINCT FROM p_orden->>'contexto_sha256'
 OR v_filtro->>'autor_lectura_ref' IS DISTINCT FROM
   (v_contexto->>'principal_ref')||':'||(v_contexto->>'perfil_activo_ref')
 OR vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
  p_orden->>'registro_contexto_ref',p_orden->>'contexto_sha256',
  p_orden->>'procedencia_sha256',p_contexto_canonico) IS NOT TRUE
 OR vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(
  p_contexto_canonico) IS NOT TRUE
 OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
  p_vinculo_canonico) IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: vínculo histórico inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO v_sesion FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref');
 IF NOT FOUND THEN RAISE EXCEPTION 'AD251: sesión actual ausente' USING ERRCODE='42501'; END IF;
 IF v_sesion.autenticacion_huella_sha256 IS DISTINCT FROM p_orden->>'autenticacion_sha256'
 OR v_sesion.cuenta_ref IS DISTINCT FROM v_contexto->>'cuenta_ref'
 OR v_sesion.cuenta_ordinaria_ref IS DISTINCT FROM v_vinculo->>'cuenta_ordinaria_ref'
 OR v_sesion.control_sesion_ref IS DISTINCT FROM v_vinculo->>'control_sesion_ref'
 OR v_sesion.control_sesion_revision IS DISTINCT FROM v_vinculo->>'control_sesion_revision'
 OR v_sesion.control_sesion_huella_sha256 IS DISTINCT FROM v_vinculo->>'control_sesion_huella_sha256'
 OR v_sesion.superficie IS DISTINCT FROM 'interna_corporativa'
 OR v_sesion.cuenta_privilegiada IS NOT FALSE
 OR v_sesion.garantia_observada IS DISTINCT FROM 'alto'
 OR v_sesion.garantia_observada IS DISTINCT FROM v_vinculo->>'garantia_observada'
 OR v_sesion.sesion_valida_hasta IS DISTINCT FROM (v_vinculo->>'sesion_valida_hasta')::timestamptz
 OR v_sesion.sesion_revalidada_en IS DISTINCT FROM (v_vinculo->>'sesion_revalidada_en')::timestamptz
 OR vec_autorizacion.cotejar_lectura_auditoria_rrhh_actual_v1(
  p_orden,v_contexto->>'principal_ref',v_contexto->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD251: autoridad actual denegada' USING ERRCODE='42501'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.lectura_auditoria_rrhh.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_contexto_canonico),'hex'))||
  vec_autorizacion_atestada_v3.encuadrar_mac(encode(sha256(p_vinculo_canonico),'hex'));
 FOREACH v_clave IN ARRAY v_claves LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_orden->>v_clave);
 END LOOP;
 v_material_sha:=encode(sha256(v_material),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended(
  'vec_autorizacion_atestada_v3:lectura:'||(p_orden->>'intento_ref'),0));
 PERFORM 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE intento_ref=p_orden->>'intento_ref';
 IF FOUND THEN RAISE EXCEPTION 'AD251: lectura ya asentada' USING ERRCODE='23505'; END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 v_secuencia:=v_secuencia+1;v_instante:=clock_timestamp();
 v_ref:='aud_v3_la_'||substr(p_orden->>'intento_ref',9,32);
 v_huella:=encode(sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.lectura_auditoria_rrhh.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(v_instante AT TIME ZONE 'UTC',
   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  intento_ref,intento_material_sha256,actor_ref,perfil_activo_ref,registro_contexto_ref,
  contexto_sha256,procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,
  correlacion_ref,vinculo_sha256,transaccion_origen,
  lectura_revision_permisos,lectura_instantanea_sha256)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'lectura_auditoria_rrhh',
  p_orden->>'intento_ref',v_material_sha,v_contexto->>'principal_ref',
  v_contexto->>'perfil_activo_ref',p_orden->>'registro_contexto_ref',
  p_orden->>'contexto_sha256',p_orden->>'procedencia_sha256',
  p_orden->>'autenticacion_ref',p_orden->>'sesion_ref',p_orden->>'autenticacion_sha256',
  p_orden->>'accion',p_orden->>'modulo_id',p_orden->>'recurso_ref',
  p_orden->>'finalidad_ref',p_orden->>'resultado',p_orden->>'motivo_ref',
  p_orden->>'proceso',p_orden->>'canal',p_orden->>'correlacion_ref',
  encode(sha256(p_vinculo_canonico),'hex'),pg_current_xact_id(),
  (p_orden->>'lectura_revision_permisos')::numeric,
  p_orden->>'lectura_instantanea_sha256');
 v_detalle_sha:=encode(sha256(convert_to(array_to_string(ARRAY[
  'vec.auditoria.detalle-lectura-rrhh.v1',v_ref,
  p_orden->>'operacion',p_orden->>'fuente',p_orden->>'organizacion_ref',
  p_orden->>'unidad_ref',p_orden->>'ambito_ref',
  p_orden->>'recurso_ref',v_filtro_sha,p_orden->>'motivo_catalogo_ref',
  p_orden->>'motivo_catalogo_sha256',p_orden->>'motivo_consulta'],E'\n'),'UTF8')),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1(
  auditoria_ref,transaccion_origen,operacion,fuente,organizacion_ref,unidad_ref,ambito_ref,recurso_ref,
  filtro_sha256,filtro,motivo_catalogo_ref,motivo_catalogo_sha256,motivo_consulta,detalle_sha256)
 VALUES(v_ref,pg_current_xact_id(),p_orden->>'operacion',p_orden->>'fuente',
  p_orden->>'organizacion_ref',p_orden->>'unidad_ref',p_orden->>'ambito_ref',
  p_orden->>'recurso_ref',v_filtro_sha,v_filtro,
  p_orden->>'motivo_catalogo_ref',p_orden->>'motivo_catalogo_sha256',
  p_orden->>'motivo_consulta',v_detalle_sha);
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden->>'correlacion_ref',v_instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_consultor_rrhh;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(bytea,bytea,jsonb)
 TO vec_contratacion_temporal_consultor_rrhh;

CREATE FUNCTION vec_autorizacion_atestada_v3.cotejar_asiento_lectura_auditoria_rrhh_v1(
 p_auditoria_ref text,p_operacion text,p_fuente text,p_organizacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE d record;a record;h text;
BEGIN
 IF vec_autorizacion_atestada_v3.login_lectura_auditoria_rrhh_valido_v1() IS NOT TRUE
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD251: consulta sin asiento' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT d FROM vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1
 WHERE auditoria_ref=p_auditoria_ref AND transaccion_origen=pg_current_xact_id() FOR SHARE;
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE auditoria_ref=d.auditoria_ref AND transaccion_origen=pg_current_xact_id() FOR SHARE;
 h:=encode(sha256(convert_to(array_to_string(ARRAY[
  'vec.auditoria.detalle-lectura-rrhh.v1',d.auditoria_ref,
  d.operacion,d.fuente,d.organizacion_ref,d.unidad_ref,d.ambito_ref,
  d.recurso_ref,d.filtro_sha256,
  d.motivo_catalogo_ref,d.motivo_catalogo_sha256,d.motivo_consulta],E'\n'),'UTF8')),'hex');
 IF d.operacion IS DISTINCT FROM p_operacion OR d.fuente IS DISTINCT FROM p_fuente
 OR d.organizacion_ref IS DISTINCT FROM p_organizacion_ref
 OR d.detalle_sha256 IS DISTINCT FROM h
 OR a.tipo_registro IS DISTINCT FROM 'lectura_auditoria_rrhh'
 OR a.recurso_ref IS DISTINCT FROM d.recurso_ref
 OR a.motivo_ref IS DISTINCT FROM d.motivo_catalogo_ref
 OR a.resultado IS DISTINCT FROM 'obtenida'
 THEN RAISE EXCEPTION 'AD251: consulta sin asiento' USING ERRCODE='42501'; END IF;
 RETURN d.filtro||jsonb_build_object('unidad_ref',d.unidad_ref,'ambito_ref',d.ambito_ref);
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'AD251: consulta sin asiento' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.cotejar_asiento_lectura_auditoria_rrhh_v1(text,text,text,text) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.buscar_objetivos_auditoria_rrhh_v1(
 p_fuente text,p_organizacion_ref text,p_busqueda text,p_despues_ref text,
 p_limite integer,p_auditoria_ref text)
RETURNS TABLE(referencia text,etiqueta text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
DECLARE filtro jsonb;
BEGIN
 filtro:=vec_autorizacion_atestada_v3.cotejar_asiento_lectura_auditoria_rrhh_v1(
  p_auditoria_ref,'buscar',p_fuente,p_organizacion_ref);
 IF p_busqueda IS DISTINCT FROM filtro->>'busqueda'
 OR p_despues_ref IS DISTINCT FROM filtro->>'despues_ref'
 OR p_limite IS DISTINCT FROM (filtro->>'limite')::integer+1
 OR p_limite NOT BETWEEN 2 AND 51
 OR filtro->>'recurso_ref' IS DISTINCT FROM ('auditoria:objetivos:'||p_fuente)
 THEN RAISE EXCEPTION 'AD251: búsqueda no ligada' USING ERRCODE='42501'; END IF;
 CASE p_fuente
 WHEN 'ct' THEN
  RETURN QUERY SELECT x.referencia,x.etiqueta FROM vec_contratacion_temporal.buscar_objetivos_auditoria_rrhh_v1(
   p_organizacion_ref,p_busqueda,p_despues_ref,p_limite) x;
 WHEN 'bolsa' THEN
  RETURN QUERY SELECT x.referencia,x.etiqueta FROM vec_bolsa_llamamientos.buscar_objetivos_auditoria_rrhh_v1(
   filtro->>'unidad_ref',filtro->>'ambito_ref',p_busqueda,p_despues_ref,p_limite) x;
 WHEN 'persona' THEN
  RETURN QUERY SELECT x.referencia,x.etiqueta FROM vec_personal.buscar_objetivos_persona_auditoria_rrhh_v1(
   p_organizacion_ref,p_busqueda,p_despues_ref,p_limite) x;
 ELSE RAISE EXCEPTION 'AD251: fuente no admitida' USING ERRCODE='42501'; END CASE;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.buscar_objetivos_auditoria_rrhh_v1(text,text,text,text,integer,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.buscar_objetivos_auditoria_rrhh_v1(text,text,text,text,integer,text)
 TO vec_contratacion_temporal_consultor_rrhh;

-- Solo las lecturas nuevas conservan organización demostrada en el detalle
-- inmutable; un recurso homónimo antiguo no cruza ámbitos por igualdad textual.
CREATE FUNCTION vec_autorizacion_atestada_v3.registros_comunes_auditoria_rrhh_v1(
 p_fuente text,p_organizacion_ref text,p_recurso_ref text,p_actor_filtro text,
 p_desde timestamptz,p_hasta timestamptz,p_antes_en timestamptz,p_antes_id text,p_limite integer)
RETURNS TABLE(id text,fuente text,modulo_id text,accion text,actor_ref text,
 ocurrido_en timestamptz,resultado text,expediente_ref text,recibo_ref text,
 antes_sha256 text,despues_sha256 text,motivo text,antes jsonb,despues jsonb,
 datos_disponibles boolean,perfil_ref text,finalidad_ref text,canal text)
LANGUAGE sql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT a.auditoria_ref,p_fuente,a.modulo_id,a.accion,COALESCE(a.actor_ref,''),
  a.registrada_en,COALESCE(a.resultado,''),p_recurso_ref,COALESCE(a.decision_ref,''),
  ''::text,a.huella_sha256,d.motivo_consulta,'{}'::jsonb,'{}'::jsonb,false,
  COALESCE(a.perfil_activo_ref,''),COALESCE(a.finalidad_ref,''),COALESCE(a.canal,'')
 FROM vec_autorizacion_atestada_v3.detalle_lectura_auditoria_rrhh_v1 d
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a ON a.auditoria_ref=d.auditoria_ref
 WHERE d.fuente=p_fuente AND d.organizacion_ref=p_organizacion_ref
 AND d.recurso_ref=p_recurso_ref AND d.operacion='consultar'
 AND a.tipo_registro='lectura_auditoria_rrhh' AND a.recurso_ref=p_recurso_ref
 AND a.registrada_en>=p_desde AND a.registrada_en<p_hasta
 AND (p_actor_filtro='' OR a.actor_ref=p_actor_filtro)
 AND (p_antes_en IS NULL OR (a.registrada_en,a.auditoria_ref COLLATE "C")
  <(p_antes_en,p_antes_id COLLATE "C"))
 ORDER BY a.registrada_en DESC,a.auditoria_ref COLLATE "C" DESC LIMIT p_limite;
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registros_comunes_auditoria_rrhh_v1(
 text,text,text,text,timestamptz,timestamptz,timestamptz,text,integer) FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.consultar_registros_auditoria_rrhh_v1(
 p_fuente text,p_organizacion_ref text,p_recurso_ref text,
 p_desde timestamptz,p_hasta timestamptz,p_antes_en timestamptz,p_antes_id text,
 p_limite integer,p_actor_filtro text,p_auditoria_ref text)
RETURNS TABLE(id text,fuente text,modulo_id text,accion text,actor_ref text,
 ocurrido_en timestamptz,resultado text,expediente_ref text,recibo_ref text,
 antes_sha256 text,despues_sha256 text,motivo text,antes jsonb,despues jsonb,
 datos_disponibles boolean,perfil_ref text,finalidad_ref text,canal text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s'
SET statement_timeout='10s' AS $f$
DECLARE filtro jsonb;cursor_en text;
BEGIN
 filtro:=vec_autorizacion_atestada_v3.cotejar_asiento_lectura_auditoria_rrhh_v1(
  p_auditoria_ref,'consultar',p_fuente,p_organizacion_ref);
 cursor_en:=CASE WHEN p_antes_en IS NULL THEN '' ELSE
  to_char(p_antes_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END;
 IF p_recurso_ref IS DISTINCT FROM filtro->>'recurso_ref'
 OR p_actor_filtro IS DISTINCT FROM filtro->>'actor_ref'
 OR to_char(p_desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM filtro->>'desde'
 OR to_char(p_hasta AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM filtro->>'hasta'
 OR cursor_en IS DISTINCT FROM filtro->>'antes_en'
 OR p_antes_id IS DISTINCT FROM filtro->>'antes_id'
 OR p_limite IS DISTINCT FROM (filtro->>'limite')::integer+1
 OR p_limite NOT BETWEEN 2 AND 101
 OR p_desde IS NULL OR p_hasta IS NULL OR p_desde>=p_hasta OR p_hasta-p_desde>interval '31 days'
 OR (p_antes_en IS NULL) <> (p_antes_id='')
 THEN RAISE EXCEPTION 'AD251: consulta no ligada' USING ERRCODE='42501'; END IF;
 CASE p_fuente
 WHEN 'ct' THEN
  IF NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.recursos_auditoria_rrhh_v1(
   p_organizacion_ref,p_recurso_ref) x) THEN
   RAISE EXCEPTION 'AD251: objetivo ajeno' USING ERRCODE='42501'; END IF;
  RETURN QUERY SELECT r.* FROM (
   SELECT * FROM vec_contratacion_temporal.registros_auditoria_rrhh_v1(
    p_organizacion_ref,p_recurso_ref,p_actor_filtro,p_desde,p_hasta,p_antes_en,p_antes_id,p_limite)
   UNION ALL
   SELECT * FROM vec_autorizacion_atestada_v3.registros_comunes_auditoria_rrhh_v1(
    p_fuente,p_organizacion_ref,p_recurso_ref,p_actor_filtro,p_desde,p_hasta,p_antes_en,p_antes_id,p_limite)
  ) r ORDER BY r.ocurrido_en DESC,r.id COLLATE "C" DESC LIMIT p_limite;
 WHEN 'bolsa' THEN
  IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.recursos_auditoria_rrhh_v1(
   filtro->>'unidad_ref',filtro->>'ambito_ref',p_recurso_ref) x) THEN
   RAISE EXCEPTION 'AD251: objetivo ajeno' USING ERRCODE='42501'; END IF;
  RETURN QUERY SELECT r.* FROM (
   SELECT * FROM vec_bolsa_llamamientos.registros_auditoria_rrhh_v1(
    filtro->>'unidad_ref',filtro->>'ambito_ref',p_recurso_ref,p_actor_filtro,
    p_desde,p_hasta,p_antes_en,p_antes_id,p_limite)
   UNION ALL
   SELECT * FROM vec_autorizacion_atestada_v3.registros_comunes_auditoria_rrhh_v1(
    p_fuente,p_organizacion_ref,p_recurso_ref,p_actor_filtro,p_desde,p_hasta,p_antes_en,p_antes_id,p_limite)
  ) r ORDER BY r.ocurrido_en DESC,r.id COLLATE "C" DESC LIMIT p_limite;
 WHEN 'persona' THEN
  IF NOT EXISTS(SELECT 1 FROM vec_personal.recursos_persona_auditoria_rrhh_v1(
   p_organizacion_ref,p_recurso_ref) x) THEN
   RAISE EXCEPTION 'AD251: objetivo ajeno' USING ERRCODE='42501'; END IF;
  RETURN QUERY SELECT r.* FROM (
   SELECT * FROM vec_personal.registros_persona_auditoria_rrhh_v1(
    p_organizacion_ref,p_recurso_ref,p_actor_filtro,p_desde,p_hasta,p_antes_en,p_antes_id,p_limite)
   UNION ALL
   SELECT * FROM vec_autorizacion_atestada_v3.registros_comunes_auditoria_rrhh_v1(
    p_fuente,p_organizacion_ref,p_recurso_ref,p_actor_filtro,p_desde,p_hasta,p_antes_en,p_antes_id,p_limite)
  ) r ORDER BY r.ocurrido_en DESC,r.id COLLATE "C" DESC LIMIT p_limite;
 ELSE RAISE EXCEPTION 'AD251: fuente no admitida' USING ERRCODE='42501'; END CASE;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consultar_registros_auditoria_rrhh_v1(
 text,text,text,timestamptz,timestamptz,timestamptz,text,integer,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consultar_registros_auditoria_rrhh_v1(
 text,text,text,timestamptz,timestamptz,timestamptz,text,integer,text,text)
 TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
