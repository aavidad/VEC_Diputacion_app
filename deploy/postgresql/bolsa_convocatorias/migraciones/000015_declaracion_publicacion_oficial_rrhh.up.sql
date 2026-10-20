\set ON_ERROR_STOP on
-- BC15 / F1-95: RRHH declara desde la web la publicación oficial de las bases.
-- * declarar_publicacion_oficial_bases_rrhh_v1: fachada del propietario de
--   convocatorias para el pool de Bolsa. Consume la autorización V3
--   (bolsa.convocatoria.publicacion_oficial.declarar sobre la versión) y llama
--   a declarar_publicacion_oficial_bases_v1 (BC14) en la misma transacción.
--   Quién declara y el motivo salen de la decisión consumida, nunca del JSON.
--   Mientras el núcleo no tenga el consumidor
--   consumir_publicacion_oficial_bases_v1, responde 55000 sin escribir nada.
-- * consultar_publicaciones_oficiales_bases_rrhh_v1: versiones publicadas
--   vigentes con su ámbito de gestión y la declaración vigente, para que la
--   pantalla elija la versión y la aplicación ligue el recurso de la decisión.
--   Solo datos que el anuncio oficial hará públicos; la aplicación la sirve a
--   quien tiene la concesión de declarar.
-- * publicacion_oficial_bases_autorizacion: decisión y auditoría de cada
--   declaración hecha por esta vía; solo adición.
-- Sin DOWN ni reaplicación. No toca BC14 ni el núcleo.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000015',0));
DO $pre$
DECLARE r record; f regprocedure; p record;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(text,integer)') IS NOT NULL
 OR to_regrole('vec_bolsa_llamamientos_ejecutor') IS NULL
 OR (SELECT rolcanlogin FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_ejecutor')
 OR has_function_privilege('vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(text,integer,text,text)','EXECUTE') IS NOT TRUE
 -- La única concesión nueva del esquema es USAGE para el ejecutor de Bolsa.
 OR (SELECT nspacl::text FROM pg_namespace WHERE nspname='vec_bolsa_convocatorias') IS DISTINCT FROM
    '{vec_bolsa_convocatorias_propietario=UC/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_propietario=U/vec_bolsa_convocatorias_propietario}'
 THEN RAISE EXCEPTION 'BC15: preimagen causal incompatible' USING ERRCODE='55000'; END IF;
 FOR r IN SELECT * FROM (VALUES
   ('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb)','b9aa572b6ee96890b36e3d6baf58e3c99119ad6e8ec9bfd72e732b03d1e6a6ac'),
   ('vec_bolsa_convocatorias.publicacion_oficial_vigente_v1(text,bigint,text)','10bfcf49333b5c08aa0b6d4a1f9d7df78306f28a0d8a75f31f38462d34c3840d'),
   ('vec_bolsa_convocatorias.huella_id_inscripcion_v1(text)','39f91b5b5440488ad0ca46c0749a4dc0114e45051d72d87a7b80cbcde8755e8b')
  ) AS esperado(firma,huella) LOOP
  f:=to_regprocedure(r.firma);
  IF f IS NULL THEN RAISE EXCEPTION 'BC15: función causal ausente' USING ERRCODE='55000'; END IF;
  SELECT proowner,prosecdef,proacl INTO STRICT p FROM pg_proc WHERE oid=f;
  IF pg_get_userbyid(p.proowner)<>'vec_bolsa_convocatorias_propietario' OR p.prosecdef
  OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario}'
  OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>r.huella
  THEN RAISE EXCEPTION 'BC15: definición BC14 divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- Vínculo de cada declaración web con la decisión consumida. Solo adición.
CREATE TABLE vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion (
 publicacion_ref text PRIMARY KEY
  REFERENCES vec_bolsa_convocatorias.publicacion_oficial_bases(publicacion_ref),
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL,
 auditoria_ref text NOT NULL,
 perfil_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 CHECK (decision_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{2,255}$'
  AND consumo_huella_sha256 ~ '^[0-9a-f]{64}$'
  AND auditoria_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{2,255}$'
  AND perfil_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{2,255}$')
);
CREATE TRIGGER publicacion_oficial_bases_autorizacion_inmutable BEFORE UPDATE OR DELETE
 ON vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable();
CREATE TRIGGER publicacion_oficial_bases_autorizacion_no_truncar BEFORE TRUNCATE
 ON vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_convocatorias.rechazar_mutacion_inmutable();
ALTER TABLE vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion FORCE ROW LEVEL SECURITY;
CREATE POLICY acceso_propietario_exacto ON vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion
 FOR ALL TO vec_bolsa_convocatorias_propietario
 USING (current_user = 'vec_bolsa_convocatorias_propietario')
 WITH CHECK (current_user = 'vec_bolsa_convocatorias_propietario');
REVOKE ALL ON vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion FROM PUBLIC;
REVOKE ALL ON TYPE vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion FROM PUBLIC;

-- Sesión del pool de Bolsa: miembro del ejecutor y de nada propietario.
CREATE FUNCTION vec_bolsa_convocatorias.sesion_bolsa_rrhh_valida_v1()
RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT session_user<>current_user
  AND coalesce(pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER'),false)
  AND NOT coalesce(pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER'),true)
  AND NOT coalesce(pg_has_role(session_user,'vec_bolsa_convocatorias_propietario','MEMBER'),true)
  AND NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
   AND (r.rolsuper OR r.rolbypassrls OR r.rolcreaterole))
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.sesion_bolsa_rrhh_valida_v1() FROM PUBLIC;

-- Versión publicada vigente (la última de su convocatoria) por referencia
-- canónica cv1_<sha256 del id>_v<secuencia>, con su ámbito de gestión.
CREATE FUNCTION vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(p_convocatoria_ref text)
RETURNS jsonb LANGUAGE plpgsql STABLE
SET search_path=pg_catalog,pg_temp SET "TimeZone"='UTC' AS $f$
DECLARE partes text[]; v vec_bolsa_convocatorias.version_convocatoria%ROWTYPE; c jsonb;
 politica jsonb; ambito jsonb; abre timestamptz; vigente record;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario' THEN
  RAISE EXCEPTION 'BC15: lectura no autorizada' USING ERRCODE='42501'; END IF;
 partes:=regexp_match(coalesce(p_convocatoria_ref,''),'^cv1_([0-9a-f]{64})_v([1-9][0-9]{0,15})$');
 IF partes IS NULL OR partes[2]::numeric>9007199254740991 THEN
  RAISE EXCEPTION 'BC15: referencia no canónica' USING ERRCODE='22023'; END IF;
 BEGIN
  SELECT * INTO STRICT v FROM vec_bolsa_convocatorias.version_convocatoria x
   WHERE vec_bolsa_convocatorias.huella_id_inscripcion_v1(x.convocatoria_id) COLLATE "C"=partes[1] COLLATE "C"
    AND x.secuencia=partes[2]::bigint;
 EXCEPTION WHEN no_data_found OR too_many_rows THEN
  RAISE EXCEPTION 'BC15: versión no vigente' USING ERRCODE='B9601'; END;
 IF v.estado<>'publicada' OR encode(sha256(v.version_canonica),'hex')<>v.huella_version_sha256
 OR EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
   WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v.secuencia
    AND posterior.estado IN('publicada','sustituida','retirada'))
 THEN RAISE EXCEPTION 'BC15: versión no vigente' USING ERRCODE='B9601'; END IF;
 BEGIN c:=convert_from(v.version_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'BC15: versión ilegible' USING ERRCODE='55000'; END;
 politica:=c#>'{configuracion,catalogos}';
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR c->>'id' IS DISTINCT FROM v.convocatoria_id
 OR jsonb_typeof(politica) IS DISTINCT FROM 'object'
 OR coalesce(politica->>'version','') !~ '^[1-9][0-9]{0,8}$'
 THEN RAISE EXCEPTION 'BC15: versión sin política de gestión' USING ERRCODE='55000'; END IF;
 ambito:=vec_catalogos_configurables.comprobar_ambito_gestion_inscripcion_v1(
  politica->>'id',(politica->>'version')::integer,politica->>'huella_contenido_sha256',p_convocatoria_ref);
 SELECT min((x->>'abre_en')::timestamptz) INTO abre
 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(c#>'{contenido,plazos}')='array'
  THEN c#>'{contenido,plazos}' ELSE '[]'::jsonb END) x
 WHERE x->>'tipo'='inscripcion';
 SELECT p.* INTO vigente FROM vec_bolsa_convocatorias.publicacion_oficial_bases p
 WHERE p.convocatoria_id=v.convocatoria_id AND p.secuencia=v.secuencia
  AND p.huella_version_sha256=v.huella_version_sha256
  AND NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.publicacion_oficial_bases s
   WHERE s.sustituye_ref=p.publicacion_ref);
 RETURN jsonb_build_object('convocatoria_ref',p_convocatoria_ref,
  'convocatoria_id',v.convocatoria_id,'secuencia',v.secuencia,
  'identificador_publico',c#>>'{contenido,identificador_publico}',
  'titulo',c#>>'{contenido,titulo}',
  'huella_version_sha256',v.huella_version_sha256,
  'unidad_ref',ambito->>'unidad_ref','ambito_ref',ambito->>'ambito_ref',
  'plazo_inscripcion_abre_en',abre,
  'publicacion',CASE WHEN vigente.publicacion_ref IS NULL THEN NULL ELSE jsonb_build_object(
   'publicacion_ref',vigente.publicacion_ref,'medio',vigente.medio,
   'boletin_numero',vigente.boletin_numero,'anuncio_ref',vigente.anuncio_ref,
   'fecha_publicacion',to_char(vigente.fecha_publicacion,'YYYY-MM-DD'),
   'direccion',vigente.direccion,'registrada_en',vigente.registrada_en) END);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(text) FROM PUBLIC;

-- Lista para la pantalla de RRHH, o una sola versión si se indica su
-- referencia. Las versiones sin ámbito de gestión publicado no se listan:
-- nadie podría autorizar su declaración.
CREATE FUNCTION vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(
 p_convocatoria_ref text,p_limite integer)
RETURNS jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE x record; fila jsonb; items jsonb:='[]'::jsonb; total integer:=0;
BEGIN
 IF NOT vec_bolsa_convocatorias.sesion_bolsa_rrhh_valida_v1() THEN
  RAISE EXCEPTION 'BC15: consulta denegada' USING ERRCODE='42501'; END IF;
 IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN
  RAISE EXCEPTION 'BC15: límite inválido' USING ERRCODE='22023'; END IF;
 IF p_convocatoria_ref IS NOT NULL THEN
  RETURN jsonb_build_object('items',jsonb_build_array(
   vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(p_convocatoria_ref)-'convocatoria_id'),'completa',true);
 END IF;
 FOR x IN SELECT 'cv1_'||vec_bolsa_convocatorias.huella_id_inscripcion_v1(v.convocatoria_id)
   ||'_v'||v.secuencia::text AS referencia
  FROM vec_bolsa_convocatorias.version_convocatoria v
  WHERE v.estado='publicada'
   AND NOT EXISTS(SELECT 1 FROM vec_bolsa_convocatorias.version_convocatoria posterior
    WHERE posterior.convocatoria_id=v.convocatoria_id AND posterior.secuencia>v.secuencia
     AND posterior.estado IN('publicada','sustituida','retirada'))
  ORDER BY v.registrada_en DESC,v.convocatoria_id,v.secuencia
 LOOP
  BEGIN
   fila:=vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(x.referencia);
  EXCEPTION WHEN SQLSTATE 'B9601' OR SQLSTATE '22023' OR SQLSTATE '55000' THEN
   CONTINUE;
  END;
  total:=total+1;
  IF total>p_limite THEN
   RETURN jsonb_build_object('items',items,'completa',false); END IF;
  items:=items||jsonb_build_array(fila-'convocatoria_id');
 END LOOP;
 RETURN jsonb_build_object('items',items,'completa',true);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(text,integer) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(text,integer)
 TO vec_bolsa_llamamientos_ejecutor;

-- Declaración desde la web. p: anuncio_ref, boletin_numero,
-- clave_idempotencia, convocatoria_ref, direccion, fecha_publicacion,
-- huella_version_sha256, medio y sustituye_ref. El atributo material_sha256 del
-- recurso V3 es sha256(p::text), como en B133.
CREATE FUNCTION vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(
 p jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET "TimeZone"='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE ctx jsonb; huella text; contexto_sha text; consumo record; d jsonb; m jsonb;
 r jsonb; previa record; vinculo record; es_replay boolean;
 consumidor regprocedure;
BEGIN
 IF NOT vec_bolsa_convocatorias.sesion_bolsa_rrhh_valida_v1() THEN
  RAISE EXCEPTION 'BC15: declaración denegada' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'BC15: la declaración exige una transacción serializable' USING ERRCODE='55000'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>4096
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p) k) IS DISTINCT FROM
    ARRAY['anuncio_ref','boletin_numero','clave_idempotencia','convocatoria_ref','direccion',
     'fecha_publicacion','huella_version_sha256','medio','sustituye_ref']
 OR jsonb_typeof(p->'convocatoria_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p->'huella_version_sha256') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p->'clave_idempotencia') IS DISTINCT FROM 'string'
 OR p->>'clave_idempotencia' !~ '^[A-Za-z0-9:_-]{8,128}$'
 THEN RAISE EXCEPTION 'BC15: solicitud inválida' USING ERRCODE='22023'; END IF;
 -- Sin consumidor en el núcleo no hay declaración: falla cerrada, sin efectos.
 IF has_schema_privilege('vec_autorizacion_atestada_v3','USAGE') IS NOT TRUE THEN
  RAISE EXCEPTION 'BC15: autorización de la declaración no disponible' USING ERRCODE='55000'; END IF;
 consumidor:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF consumidor IS NULL OR has_function_privilege(consumidor,'EXECUTE') IS NOT TRUE THEN
  RAISE EXCEPTION 'BC15: autorización de la declaración no disponible' USING ERRCODE='55000'; END IF;
 ctx:=vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(p->>'convocatoria_ref');
 IF ctx->>'huella_version_sha256' IS DISTINCT FROM p->>'huella_version_sha256' THEN
  RAISE EXCEPTION 'BC15: versión no vigente' USING ERRCODE='B9601'; END IF;
 huella:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 -- Canon del recurso V3, igual que RecursoAutorizable en Go y B133.
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":'||to_json(ctx->>'ambito_ref')::text||
  ',"unidad_ref":'||to_json(ctx->>'unidad_ref')::text||
  '},"atributos":{"material_sha256":"'||huella||'"}}','UTF8')),'hex');
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 m:=convert_from(p_motivo,'UTF8')::jsonb->'referencia';
 IF consumo.efecto_ref IS DISTINCT FROM p->>'convocatoria_ref'
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 OR d->>'accion' IS DISTINCT FROM 'bolsa.convocatoria.publicacion_oficial.declarar'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'version_convocatoria'
 OR d#>>'{ambitos,unidad_ref}' IS DISTINCT FROM ctx->>'unidad_ref'
 OR d#>>'{ambitos,ambito_ref}' IS DISTINCT FROM ctx->>'ambito_ref'
 OR jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR coalesce(m->>'catalogo_id','') !~ '^[a-z][a-z0-9._-]{0,127}$'
 OR coalesce(m->>'catalogo_version','') !~ '^[1-9][0-9]{0,9}$'
 OR coalesce(m->>'entrada_clave','') !~ '^[a-z][a-z0-9._-]{0,127}$'
 THEN RAISE EXCEPTION 'BC15: decisión ajena a la declaración' USING ERRCODE='42501'; END IF;
 SELECT x.* INTO previa FROM vec_bolsa_convocatorias.publicacion_oficial_bases x
  WHERE x.clave_idempotencia=p->>'clave_idempotencia';
 es_replay:=FOUND;
 IF NOT es_replay AND consumo.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'BC15: falta efecto del consumo' USING ERRCODE='42501'; END IF;
 r:=vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb_build_object(
  'convocatoria_id',ctx->>'convocatoria_id','secuencia',(ctx->>'secuencia')::bigint,
  'huella_version_sha256',p->'huella_version_sha256','medio',p->'medio',
  'boletin_numero',p->'boletin_numero','anuncio_ref',p->'anuncio_ref',
  'fecha_publicacion',p->'fecha_publicacion','direccion',p->'direccion',
  'sustituye_ref',p->'sustituye_ref','declarada_por',d->'principal_id',
  'motivo_ref',(m->>'catalogo_id')||':'||(m->>'catalogo_version')||':'||(m->>'entrada_clave'),
  'clave_idempotencia',p->'clave_idempotencia'));
 IF (r->>'reutilizada')::boolean IS DISTINCT FROM es_replay THEN
  RAISE EXCEPTION 'BC15: estado de la declaración incoherente' USING ERRCODE='55000'; END IF;
 IF NOT es_replay THEN
  INSERT INTO vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion(
   publicacion_ref,decision_ref,consumo_huella_sha256,auditoria_ref,perfil_ref,registrada_en)
  VALUES(r->>'publicacion_ref',consumo.decision_ref,consumo.consumo_huella_sha256,
   consumo.auditoria_ref,d->>'perfil_activo_ref',(r->>'registrada_en')::timestamptz);
 END IF;
 SELECT a.* INTO vinculo FROM vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion a
  WHERE a.publicacion_ref=r->>'publicacion_ref';
 -- Una clave declarada fuera de esta vía no se recupera desde la web.
 IF NOT FOUND THEN
  RAISE EXCEPTION 'BC15: clave reutilizada con otra declaración' USING ERRCODE='23505'; END IF;
 -- Recibo mínimo: sin identificador interno de la convocatoria ni declarante.
 RETURN jsonb_build_object('publicacion_ref',r->>'publicacion_ref','recibo_ref',r->>'recibo_ref',
  'evento_ref',r->>'evento_ref','convocatoria_ref',p->>'convocatoria_ref',
  'secuencia',(ctx->>'secuencia')::bigint,'medio',r->>'medio','boletin_numero',r->'boletin_numero',
  'anuncio_ref',r->>'anuncio_ref','fecha_publicacion',r->>'fecha_publicacion',
  'direccion',r->'direccion','sustituye_ref',r->'sustituye_ref',
  'auditoria_ref',vinculo.auditoria_ref,'registrada_en',r->'registrada_en',
  'reutilizada',es_replay);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(
 jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
GRANT USAGE ON SCHEMA vec_bolsa_convocatorias TO vec_bolsa_llamamientos_ejecutor;

DO $post$
DECLARE x record; ejecutor oid:='vec_bolsa_llamamientos_ejecutor'::regrole;
BEGIN
 -- El ejecutor solo alcanza las dos funciones nuevas del esquema.
 IF EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE n.nspname='vec_bolsa_convocatorias' AND a.grantee IN(0,ejecutor)
    AND p.oid NOT IN('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
     'vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(text,integer)'::regprocedure))
 OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
   CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE n.nspname='vec_bolsa_convocatorias' AND a.grantee IN(0,ejecutor))
 THEN RAISE EXCEPTION 'BC15: el ejecutor alcanza otros objetos' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT * FROM (VALUES
  ('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',true,true),
  ('vec_bolsa_convocatorias.consultar_publicaciones_oficiales_bases_rrhh_v1(text,integer)',true,true),
  ('vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(text)',false,false),
  ('vec_bolsa_convocatorias.sesion_bolsa_rrhh_valida_v1()',false,false)
 ) v(firma,definidora,concedida) LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure(x.firma)
    AND pg_get_userbyid(p.proowner)='vec_bolsa_convocatorias_propietario'
    AND p.prosecdef=x.definidora AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']
    AND p.proacl::text=CASE WHEN x.concedida
     THEN '{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_convocatorias_propietario}'
     ELSE '{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario}' END)
  THEN RAISE EXCEPTION 'BC15: función sin cierre de permisos %',x.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF encode(sha256(convert_to(pg_get_functiondef('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb)'::regprocedure),'UTF8')),'hex')
  <>'b9aa572b6ee96890b36e3d6baf58e3c99119ad6e8ec9bfd72e732b03d1e6a6ac'
 THEN RAISE EXCEPTION 'BC15: BC14 alterada' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
