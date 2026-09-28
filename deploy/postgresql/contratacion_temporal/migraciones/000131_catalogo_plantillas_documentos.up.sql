\set ON_ERROR_STOP on
-- CT-131: historia administrativa del catálogo completo de plantillas.
-- AD3-94 debe precederla. La publicación inicial se provisiona por el
-- migrador antes de exponer HTTP; ninguna edición puede importarla.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000131',0));
DO $pre$
DECLARE f oid;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR to_regclass('vec_contratacion_temporal.catalogo_plantillas_historia_v1') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)') IS NOT NULL
    OR to_regprocedure('vec_contratacion_temporal.rechazar_mutacion_historia_v1()') IS NULL
    OR has_schema_privilege('vec_contratacion_temporal_migrador'::regrole,
          'vec_contratacion_temporal'::regnamespace,'USAGE')
 THEN RAISE EXCEPTION 'CT-131: preimagen incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege(current_user,f,'EXECUTE')
 THEN RAISE EXCEPTION 'CT-131: AD3-94 requerido' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_contratacion_temporal.catalogo_plantillas_historia_v1 (
 secuencia bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 9007199254740991),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 9007199254740991),
 estado text NOT NULL CHECK(estado IN ('borrador','publicado')),
 origen text NOT NULL CHECK(origen IN ('bootstrap','edicion','publicacion')),
 catalogo jsonb NOT NULL CHECK(jsonb_typeof(catalogo)='object' AND octet_length(catalogo::text)<=16777216),
 catalogo_huella_sha256 text NOT NULL CHECK(catalogo_huella_sha256~'^[0-9a-f]{64}$'),
 contenido_json_sha256 text NOT NULL CHECK(contenido_json_sha256~'^[0-9a-f]{64}$'),
 clave_idempotencia uuid,
 solicitud_huella_sha256 text,
 recibo_ref text,
 actor_ref text NOT NULL,
 decision_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL,
 auditoria_ref text NOT NULL,
 provision_fuente_ref text,
 provision_aprobacion_ref text,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(version,revision,estado),
 UNIQUE(clave_idempotencia),
 UNIQUE(recibo_ref),
 CHECK ((origen='bootstrap')=(clave_idempotencia IS NULL)),
 CHECK ((origen='bootstrap')=(solicitud_huella_sha256 IS NULL)),
 CHECK ((origen='bootstrap')=(recibo_ref IS NULL)),
 CHECK (solicitud_huella_sha256 IS NULL OR solicitud_huella_sha256~'^[0-9a-f]{64}$'),
 CHECK (recibo_ref IS NULL OR recibo_ref~'^recibo:[0-9a-f-]{36}$'),
 CHECK (actor_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 CHECK (decision_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 CHECK (consumo_huella_sha256~'^[0-9a-f]{64}$'),
 CHECK (auditoria_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 CHECK ((origen='bootstrap')=(provision_fuente_ref IS NOT NULL)),
 CHECK ((origen='bootstrap')=(provision_aprobacion_ref IS NOT NULL)),
 CHECK (provision_fuente_ref IS NULL OR provision_fuente_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 CHECK (provision_aprobacion_ref IS NULL OR provision_aprobacion_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 CHECK (origen<>'bootstrap' OR (catalogo->>'fuente_ref'=provision_fuente_ref AND catalogo->>'aprobacion_ref'=provision_aprobacion_ref)),
 CHECK (registrada_en=date_trunc('microseconds',registrada_en)),
 CHECK (catalogo->>'id'='vec.contratacion_temporal.plantillas_documentos'),
 CHECK (catalogo->>'modulo_id'='contratacion_temporal'),
 CHECK (catalogo->>'estado'=estado),
 CHECK ((catalogo->>'version')::bigint=version AND (catalogo->>'revision')::bigint=revision)
);
CREATE INDEX catalogo_plantillas_historia_estado_secuencia_v1 ON vec_contratacion_temporal.catalogo_plantillas_historia_v1(estado,secuencia DESC);

CREATE TABLE vec_contratacion_temporal.catalogo_plantillas_outbox_v1 (
 evento_ref text PRIMARY KEY CHECK(evento_ref~'^evento:[0-9a-f-]{36}$'),
 historia_secuencia bigint NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.catalogo_plantillas_historia_v1(secuencia),
 recibo_ref text NOT NULL UNIQUE,
 tipo text NOT NULL CHECK(tipo IN ('contratacion_temporal.plantillas_documentos.borrador_actualizado','contratacion_temporal.plantillas_documentos.publicado')),
 estado text NOT NULL DEFAULT 'pendiente' CHECK(estado='pendiente'),
 creada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(recibo_ref) REFERENCES vec_contratacion_temporal.catalogo_plantillas_historia_v1(recibo_ref)
);
CREATE TABLE vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1 (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref~'^recibo:[0-9a-f-]{36}$'),
 historia_secuencia bigint NOT NULL UNIQUE REFERENCES vec_contratacion_temporal.catalogo_plantillas_historia_v1(secuencia),
 auditoria_ref text NOT NULL UNIQUE CHECK(auditoria_ref~'^provision:[0-9a-f-]{36}$'),
 solicitud_huella_sha256 text NOT NULL CHECK(solicitud_huella_sha256~'^[0-9a-f]{64}$'),
 instalador_ref text NOT NULL CHECK(instalador_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 fuente_ref text NOT NULL CHECK(fuente_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'),
 registrada_en timestamptz(6) NOT NULL CHECK(registrada_en=date_trunc('microseconds',registrada_en))
);
DO $proteccion$
DECLARE tabla text; r record;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['catalogo_plantillas_historia_v1','catalogo_plantillas_outbox_v1','catalogo_plantillas_provision_auditoria_v1'] LOOP
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_contratacion_temporal.%I FOR EACH ROW EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',tabla);
  EXECUTE format('CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_contratacion_temporal.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_contratacion_temporal.rechazar_mutacion_historia_v1()',tabla);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_contratacion_temporal.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY propietario_total ON vec_contratacion_temporal.%I TO vec_contratacion_temporal_propietario USING (true) WITH CHECK (true)',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM PUBLIC',tabla);
  FOR r IN SELECT DISTINCT a.grantee FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
   WHERE c.oid=to_regclass('vec_contratacion_temporal.'||tabla) AND a.grantee<>0 AND a.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_contratacion_temporal.%I FROM %I',tabla,pg_get_userbyid(r.grantee));
  END LOOP;
  EXECUTE format('REVOKE ALL ON TYPE vec_contratacion_temporal.%I FROM PUBLIC',tabla);
 END LOOP;
END $proteccion$;

-- Instalación gobernada fuera del proceso HTTP. La misma solicitud devuelve
-- el mismo recibo; otra publicación o una solicitud divergente se rechaza.
CREATE FUNCTION vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(
 p_catalogo jsonb,p_huella_sha256 text,p_fuente_ref text,p_aprobacion_ref text)
RETURNS TABLE(resultado text,recibo_ref text,version bigint,revision bigint,catalogo_huella_sha256 text,
 contenido_json_sha256 text,procedencia_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $provision$
DECLARE h record; a record; v bigint; r bigint; json_h text; solicitud_h text;
 n integer; distintas integer; sec bigint; recibo text; auditoria text; ahora timestamptz(6);
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_gobernador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT-131: provisión denegada' USING ERRCODE='42501'; END IF;
 IF p_catalogo IS NULL OR jsonb_typeof(p_catalogo)<>'object'
    OR octet_length(p_catalogo::text)>16777216
    OR p_catalogo->>'id' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR p_catalogo->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR p_catalogo->>'estado' IS DISTINCT FROM 'publicado'
    OR jsonb_typeof(p_catalogo->'entradas') IS DISTINCT FROM 'array'
    OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR p_fuente_ref IS NULL OR p_fuente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_aprobacion_ref IS NULL OR p_aprobacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_catalogo->>'fuente_ref' IS DISTINCT FROM p_fuente_ref
    OR p_catalogo->>'aprobacion_ref' IS DISTINCT FROM p_aprobacion_ref
    OR nullif(p_catalogo->>'creado_por','') IS NULL
    OR nullif(p_catalogo->>'creado_en','') IS NULL
    OR nullif(p_catalogo->>'publicado_por','') IS NULL
    OR nullif(p_catalogo->>'publicado_en','') IS NULL
    OR nullif(p_catalogo->>'motivo_publicacion','') IS NULL
 THEN RAISE EXCEPTION 'CT-131: catálogo de provisión inválido' USING ERRCODE='22023'; END IF;
 BEGIN v:=(p_catalogo->>'version')::bigint; r:=(p_catalogo->>'revision')::bigint;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-131: versión de provisión inválida' USING ERRCODE='22023'; END;
 IF v<1 OR r<1 OR v>9007199254740991 OR r>9007199254740991
 THEN RAISE EXCEPTION 'CT-131: versión de provisión inválida' USING ERRCODE='22023'; END IF;
 SELECT count(*)::integer,count(DISTINCT e->>'clave')::integer INTO n,distintas
 FROM jsonb_array_elements(p_catalogo->'entradas') e;
 IF n<1 OR n>10000 OR n<>distintas OR EXISTS (
  SELECT 1 FROM jsonb_array_elements(p_catalogo->'entradas') e
  WHERE jsonb_typeof(e)<>'object' OR nullif(e->>'clave','') IS NULL)
 THEN RAISE EXCEPTION 'CT-131: entradas de provisión inválidas' USING ERRCODE='22023'; END IF;
 json_h:=encode(sha256(convert_to(p_catalogo::text,'UTF8')),'hex');
 solicitud_h:=encode(sha256(convert_to(p_huella_sha256||'|'||json_h||'|'||p_fuente_ref||'|'||p_aprobacion_ref,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:catalogo_plantillas_documentos',0));
 SELECT * INTO h FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
 IF FOUND THEN
  IF h.origen<>'bootstrap' OR h.version<>v OR h.revision<>r
     OR h.catalogo_huella_sha256<>p_huella_sha256 OR h.contenido_json_sha256<>json_h
     OR h.provision_fuente_ref<>p_fuente_ref OR h.provision_aprobacion_ref<>p_aprobacion_ref
     OR h.consumo_huella_sha256<>solicitud_h
  THEN RAISE EXCEPTION 'CT-131: provisión existente divergente' USING ERRCODE='23505'; END IF;
  SELECT * INTO a FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
   WHERE historia_secuencia=h.secuencia;
  IF NOT FOUND OR a.auditoria_ref<>h.auditoria_ref OR a.solicitud_huella_sha256<>solicitud_h
     OR a.instalador_ref<>h.actor_ref OR a.fuente_ref<>p_fuente_ref
     OR a.aprobacion_ref<>p_aprobacion_ref OR a.registrada_en<>h.registrada_en
  THEN RAISE EXCEPTION 'CT-131: auditoría de provisión incompatible' USING ERRCODE='55000'; END IF;
  RETURN QUERY SELECT 'replay'::text,a.recibo_ref,h.version,h.revision,h.catalogo_huella_sha256,
    h.contenido_json_sha256,h.provision_fuente_ref,h.registrada_en;
  RETURN;
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 recibo:='recibo:'||gen_random_uuid()::text;
 auditoria:='provision:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_historia_v1(
  version,revision,estado,origen,catalogo,catalogo_huella_sha256,contenido_json_sha256,
  actor_ref,decision_ref,consumo_huella_sha256,auditoria_ref,
  provision_fuente_ref,provision_aprobacion_ref,registrada_en)
 VALUES(v,r,'publicado','bootstrap',p_catalogo,p_huella_sha256,json_h,
  session_user::text,p_aprobacion_ref,solicitud_h,auditoria,
  p_fuente_ref,p_aprobacion_ref,ahora) RETURNING secuencia INTO sec;
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1(
  recibo_ref,historia_secuencia,auditoria_ref,solicitud_huella_sha256,
  instalador_ref,fuente_ref,aprobacion_ref,registrada_en)
 VALUES(recibo,sec,auditoria,solicitud_h,session_user::text,p_fuente_ref,p_aprobacion_ref,ahora);
 RETURN QUERY SELECT 'registrado'::text,recibo,v,r,p_huella_sha256,json_h,p_fuente_ref,ahora;
END $provision$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text) FROM PUBLIC;
-- El migrador no hereda USAGE del propietario (roles_up usa INHERIT FALSE).
-- Esta concesión permite invocar solo la fachada; no concede tablas ni tipos.
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_migrador;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text) TO vec_contratacion_temporal_migrador;

-- Sonda mínima del arranque. No devuelve el texto del catálogo y nunca escribe.
CREATE FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(
 p_huella_sha256 text,p_version bigint,p_fuente_ref text)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $comprobar$
DECLARE b record; a record;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$'
    OR p_version IS NULL OR p_version<1 OR p_fuente_ref IS NULL
 THEN RAISE EXCEPTION 'CT-131: comprobación denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO b FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
 IF NOT FOUND OR b.origen<>'bootstrap' OR b.estado<>'publicado'
    OR b.version<>p_version OR b.catalogo_huella_sha256<>p_huella_sha256
    OR b.provision_fuente_ref<>p_fuente_ref
    OR b.contenido_json_sha256<>encode(sha256(convert_to(b.catalogo::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT-131: publicación inicial no provisionada o divergente' USING ERRCODE='55000'; END IF;
 SELECT * INTO a FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
  WHERE historia_secuencia=b.secuencia;
 IF NOT FOUND OR a.auditoria_ref<>b.auditoria_ref OR a.solicitud_huella_sha256<>b.consumo_huella_sha256
    OR a.fuente_ref<>b.provision_fuente_ref OR a.aprobacion_ref<>b.provision_aprobacion_ref
    OR a.instalador_ref<>b.actor_ref OR a.registrada_en<>b.registrada_en
 THEN RAISE EXCEPTION 'CT-131: auditoría de provisión ausente o divergente' USING ERRCODE='55000'; END IF;
 RETURN true;
END $comprobar$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text) TO vec_contratacion_temporal_ejecutor;

CREATE FUNCTION vec_contratacion_temporal.operar_catalogo_plantillas_v1(
 p_material jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC' SET lock_timeout='2s'
AS $funcion$
DECLARE
 operacion text; d jsonb; c jsonb; nuevo jsonb; solicitud jsonb;
 consumo record; anterior record; previa record; publicado jsonb; borrador jsonb; provision record; provision_auditoria record;
 material_h text; contexto_h text; solicitud_h text; catalogo_h text;
 entrada jsonb; vieja_entradas jsonb; nueva_entradas jsonb; clave_entrada text;
 vieja_cantidad integer; nueva_cantidad integer; vieja_claves integer; nueva_claves integer;
 clave uuid; esperado_v bigint; esperado_r bigint; nuevo_v bigint; nuevo_r bigint;
 actor text; ahora timestamptz(6); recibo text; sec bigint; resp jsonb;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'CT-131: ejecución denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR jsonb_typeof(p_material)<>'object'
    OR octet_length(p_material::text)>17000000
 THEN RAISE EXCEPTION 'CT-131: material inválido' USING ERRCODE='22023'; END IF;
 operacion:=p_material->>'operacion';
 IF operacion IS NULL OR operacion<>ALL(ARRAY['consultar','editar','publicar'])
    OR (p_material-ARRAY['operacion','clave_idempotencia','version_esperada','revision_esperada','catalogo','catalogo_huella_sha256','catalogo_base_huella_sha256','solicitud'])<>'{}'::jsonb
 THEN RAISE EXCEPTION 'CT-131: operación inválida' USING ERRCODE='22023'; END IF;
 BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-131: decisión inválida' USING ERRCODE='22023'; END;
 material_h:=encode(sha256(convert_to(p_material::text,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'contratacion_temporal.plantillas_documentos.'||operacion
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_plantillas_contratacion_temporal'
    OR d->>'finalidad' IS DISTINCT FROM 'gestionar_catalogo_plantillas_contratacion_temporal'
    OR d->>'recurso_ref' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR d->'campos_permitidos' IS DISTINCT FROM (CASE WHEN operacion='consultar'
         THEN '["borrador","editor_de_esta_version","publicado"]'::jsonb ELSE '["catalogo","recibo"]'::jsonb END)
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
 THEN RAISE EXCEPTION 'CT-131: decisión divergente' USING ERRCODE='42501'; END IF;
 actor:=d->>'principal_id';
 IF actor IS NULL OR actor !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
 THEN RAISE EXCEPTION 'CT-131: actor inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_catalogo_plantillas_ct_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE
    OR consumo.efecto_ref IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_h
 THEN RAISE EXCEPTION 'CT-131: consumo divergente' USING ERRCODE='42501'; END IF;
 -- Incluso la consulta queda registrada por V3, antes de leer contenido.
 IF operacion='consultar' THEN
  IF (p_material-ARRAY['operacion'])<>'{}'::jsonb
  THEN RAISE EXCEPTION 'CT-131: consulta inválida' USING ERRCODE='22023'; END IF;
  SELECT CASE WHEN h.estado='borrador' THEN h.catalogo ELSE NULL END INTO borrador
   FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h ORDER BY h.secuencia DESC LIMIT 1;
  SELECT h.catalogo INTO publicado FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h WHERE h.estado='publicado' ORDER BY h.secuencia DESC LIMIT 1;
  RETURN jsonb_build_object('borrador',borrador,'publicado',publicado,
   'editor_de_esta_version',CASE WHEN borrador IS NULL THEN false ELSE EXISTS (
    SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 e
     WHERE e.estado='borrador' AND e.version=(borrador->>'version')::bigint
       AND e.actor_ref=actor) END);
 END IF;
 IF p_material->>'clave_idempotencia' IS NULL
    OR p_material->>'clave_idempotencia' !~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
    OR jsonb_typeof(p_material->'solicitud')<>'object'
    OR jsonb_typeof(p_material->'version_esperada')<>'number'
    OR jsonb_typeof(p_material->'revision_esperada')<>'number'
 THEN RAISE EXCEPTION 'CT-131: cambio inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  clave:=(p_material->>'clave_idempotencia')::uuid;
  esperado_v:=(p_material->>'version_esperada')::bigint;
  esperado_r:=(p_material->>'revision_esperada')::bigint;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'CT-131: versión inválida' USING ERRCODE='22023'; END;
 IF esperado_v<0 OR esperado_r<0 OR esperado_v>9007199254740991 OR esperado_r>9007199254740991
 THEN RAISE EXCEPTION 'CT-131: versión inválida' USING ERRCODE='22023'; END IF;
 nuevo:=p_material->'catalogo'; solicitud:=p_material->'solicitud';
 catalogo_h:=p_material->>'catalogo_huella_sha256';
 solicitud_h:=encode(sha256(convert_to(solicitud::text,'UTF8')),'hex');
 IF solicitud->>'version_esperada' IS DISTINCT FROM esperado_v::text
    OR solicitud->>'revision_esperada' IS DISTINCT FROM esperado_r::text
 THEN RAISE EXCEPTION 'CT-131: solicitud y versión divergentes' USING ERRCODE='22023'; END IF;
 -- Serializa clave, OCC y lectura de cabeza. El advisory lock pertenece solo
 -- a este catálogo y se libera al terminar la transacción.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:catalogo_plantillas_documentos',0));
 SELECT * INTO previa FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 h WHERE h.clave_idempotencia=clave;
 IF FOUND THEN
  IF previa.solicitud_huella_sha256 IS DISTINCT FROM solicitud_h
     OR previa.actor_ref IS DISTINCT FROM actor
     OR previa.origen IS DISTINCT FROM (CASE WHEN operacion='editar' THEN 'edicion' ELSE 'publicacion' END)
  THEN RAISE EXCEPTION 'CT-131: clave reutilizada con otra solicitud' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('catalogo',previa.catalogo,'recibo',jsonb_build_object(
   'recibo_ref',previa.recibo_ref,'clave_idempotencia',clave,'operacion',operacion,
   'version',previa.version,'revision',previa.revision,'catalogo_huella_sha256',previa.catalogo_huella_sha256,
   'decision_ref',previa.decision_ref,'auditoria_ref',previa.auditoria_ref,
   'consumo_huella_sha256',previa.consumo_huella_sha256,'registrado_en',previa.registrada_en,
   'estado_replay','replay'),'replay',true);
 END IF;
 IF jsonb_typeof(nuevo)<>'object' OR catalogo_h IS NULL
    OR catalogo_h !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'CT-131: catálogo de cambio requerido' USING ERRCODE='22023'; END IF;
 IF nuevo->>'id' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos'
    OR nuevo->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR jsonb_typeof(nuevo->'entradas') IS DISTINCT FROM 'array'
    OR jsonb_array_length(nuevo->'entradas')>10000
    OR (operacion='editar' AND nuevo->>'estado' IS DISTINCT FROM 'borrador')
    OR (operacion='publicar' AND nuevo->>'estado' IS DISTINCT FROM 'publicado')
    OR octet_length(nuevo::text)>16777216
 THEN RAISE EXCEPTION 'CT-131: catálogo inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO anterior FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT-131: catálogo bootstrap no provisionado' USING ERRCODE='55000'; END IF;
 SELECT * INTO provision FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
 IF provision.origen<>'bootstrap' OR provision.estado<>'publicado'
    OR provision.contenido_json_sha256<>encode(sha256(convert_to(provision.catalogo::text,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'CT-131: control bootstrap incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO provision_auditoria FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1
  WHERE historia_secuencia=provision.secuencia;
 IF NOT FOUND OR provision_auditoria.auditoria_ref<>provision.auditoria_ref
    OR provision_auditoria.solicitud_huella_sha256<>provision.consumo_huella_sha256
    OR provision_auditoria.fuente_ref<>provision.provision_fuente_ref
    OR provision_auditoria.aprobacion_ref<>provision.provision_aprobacion_ref
    OR provision_auditoria.instalador_ref<>provision.actor_ref
    OR provision_auditoria.registrada_en<>provision.registrada_en
 THEN RAISE EXCEPTION 'CT-131: auditoría bootstrap incompatible' USING ERRCODE='55000'; END IF;
 IF anterior.origen='bootstrap' AND (
    operacion<>'editar' OR anterior.version<>esperado_v OR esperado_r<>0
    OR p_material->>'catalogo_base_huella_sha256' IS DISTINCT FROM anterior.catalogo_huella_sha256)
 THEN RAISE EXCEPTION 'CT-131: huella o versión bootstrap divergente' USING ERRCODE='40001'; END IF;
 IF anterior.origen<>'bootstrap' AND p_material ? 'catalogo_base_huella_sha256'
 THEN RAISE EXCEPTION 'CT-131: huella bootstrap tardía' USING ERRCODE='40001'; END IF;
 IF operacion='editar' THEN
  IF anterior.secuencia IS NULL THEN
   RAISE EXCEPTION 'CT-131: catálogo bootstrap requerido' USING ERRCODE='55000';
  ELSIF anterior.estado='publicado' THEN
   IF esperado_v<>anterior.version OR esperado_r<>0 THEN RAISE EXCEPTION 'CT-131: conflicto de versión' USING ERRCODE='40001'; END IF;
   nuevo_v:=anterior.version+1; nuevo_r:=1;
  ELSE
   IF esperado_v<>anterior.version OR esperado_r<>anterior.revision THEN RAISE EXCEPTION 'CT-131: conflicto de revisión' USING ERRCODE='40001'; END IF;
   nuevo_v:=anterior.version; nuevo_r:=anterior.revision+1;
  END IF;
  IF nuevo->>'creado_por' IS DISTINCT FROM (CASE WHEN nuevo_r=1 THEN actor ELSE anterior.catalogo->>'creado_por' END)
     OR nuevo->>'creado_en' IS NULL
     OR (nuevo_r>1 AND nuevo->>'ultima_modificacion_por' IS DISTINCT FROM actor)
     OR (nuevo_r>1 AND nuevo->>'version_anterior_ref' IS DISTINCT FROM anterior.catalogo->>'version_anterior_ref')
     OR (nuevo_r>1 AND nuevo->>'creado_en' IS DISTINCT FROM anterior.catalogo->>'creado_en')
     OR (nuevo_r>1 AND nuevo->>'motivo_creacion' IS DISTINCT FROM anterior.catalogo->>'motivo_creacion')
  THEN RAISE EXCEPTION 'CT-131: autoría o filiación divergente' USING ERRCODE='42501'; END IF;
  entrada:=solicitud->'entrada'; clave_entrada:=entrada->>'clave';
  IF jsonb_typeof(entrada) IS DISTINCT FROM 'object' OR clave_entrada IS NULL OR clave_entrada=''
     OR nuevo->>'fuente_ref' IS DISTINCT FROM solicitud->>'fuente_ref'
     OR nuevo->>'nombre' IS DISTINCT FROM anterior.catalogo->>'nombre'
     OR nuevo->>'descripcion' IS DISTINCT FROM anterior.catalogo->>'descripcion'
     OR (nuevo_r=1 AND nuevo->>'motivo_creacion' IS DISTINCT FROM solicitud->>'motivo')
     OR (nuevo_r>1 AND nuevo->>'motivo_modificacion' IS DISTINCT FROM solicitud->>'motivo')
  THEN RAISE EXCEPTION 'CT-131: edición divergente de solicitud' USING ERRCODE='42501'; END IF;
  SELECT coalesce(jsonb_object_agg(e->>'clave',e),'{}'::jsonb),count(*)::integer,count(DISTINCT e->>'clave')::integer
   INTO vieja_entradas,vieja_cantidad,vieja_claves
   FROM jsonb_array_elements(anterior.catalogo->'entradas') e;
  SELECT coalesce(jsonb_object_agg(e->>'clave',e),'{}'::jsonb),count(*)::integer,count(DISTINCT e->>'clave')::integer
   INTO nueva_entradas,nueva_cantidad,nueva_claves FROM jsonb_array_elements(nuevo->'entradas') e;
  IF vieja_cantidad<>vieja_claves OR nueva_cantidad<>nueva_claves
     OR nueva_entradas IS DISTINCT FROM (vieja_entradas||jsonb_build_object(clave_entrada,entrada))
  THEN RAISE EXCEPTION 'CT-131: entradas distintas de la edición autorizada' USING ERRCODE='42501'; END IF;
 ELSE
  IF anterior.secuencia IS NULL OR anterior.estado<>'borrador'
     OR esperado_v<>anterior.version OR esperado_r<>anterior.revision
  THEN RAISE EXCEPTION 'CT-131: borrador no vigente' USING ERRCODE='40001'; END IF;
  nuevo_v:=anterior.version; nuevo_r:=anterior.revision;
  IF nuevo->>'publicado_por' IS DISTINCT FROM actor
     OR actor=anterior.catalogo->>'creado_por'
     OR actor=anterior.catalogo->>'ultima_modificacion_por'
     OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 e
                 WHERE e.estado='borrador' AND e.version=anterior.version AND e.actor_ref=actor)
     OR nuevo->>'aprobacion_ref' IS NULL OR nuevo->>'aprobacion_ref'=''
     OR nuevo->>'motivo_publicacion' IS NULL OR nuevo->>'motivo_publicacion'=''
     OR nuevo->>'publicado_en' IS NULL OR nuevo->>'publicado_en'=''
     OR nuevo->>'aprobacion_ref' IS DISTINCT FROM solicitud->>'aprobacion_ref'
     OR nuevo->>'motivo_publicacion' IS DISTINCT FROM solicitud->>'motivo'
     OR (nuevo-ARRAY['estado','publicado_por','publicado_en','aprobacion_ref','motivo_publicacion'])
        IS DISTINCT FROM (anterior.catalogo-ARRAY['estado','publicado_por','publicado_en','aprobacion_ref','motivo_publicacion'])
  THEN RAISE EXCEPTION 'CT-131: separación de funciones o contenido divergente' USING ERRCODE='42501'; END IF;
 END IF;
 IF nuevo->>'version' IS DISTINCT FROM nuevo_v::text OR nuevo->>'revision' IS DISTINCT FROM nuevo_r::text
    OR (nuevo_v>1 AND nuevo->>'version_anterior_ref' IS DISTINCT FROM 'vec.contratacion_temporal.plantillas_documentos:'||(nuevo_v-1)::text)
 THEN RAISE EXCEPTION 'CT-131: versión del catálogo divergente' USING ERRCODE='40001'; END IF;
 ahora:=date_trunc('microseconds',clock_timestamp()); recibo:='recibo:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_historia_v1(
  version,revision,estado,origen,catalogo,catalogo_huella_sha256,contenido_json_sha256,
  clave_idempotencia,solicitud_huella_sha256,recibo_ref,actor_ref,decision_ref,
  consumo_huella_sha256,auditoria_ref,registrada_en)
 VALUES(nuevo_v,nuevo_r,CASE WHEN operacion='editar' THEN 'borrador' ELSE 'publicado' END,
  CASE WHEN operacion='editar' THEN 'edicion' ELSE 'publicacion' END,
  nuevo,catalogo_h,encode(sha256(convert_to(nuevo::text,'UTF8')),'hex'),clave,solicitud_h,recibo,
  actor,consumo.decision_ref,consumo.consumo_huella_sha256,consumo.auditoria_ref,ahora)
 RETURNING secuencia INTO sec;
 INSERT INTO vec_contratacion_temporal.catalogo_plantillas_outbox_v1(evento_ref,historia_secuencia,recibo_ref,tipo,estado,creada_en)
 VALUES('evento:'||gen_random_uuid()::text,sec,recibo,
  CASE WHEN operacion='editar' THEN 'contratacion_temporal.plantillas_documentos.borrador_actualizado'
       ELSE 'contratacion_temporal.plantillas_documentos.publicado' END,'pendiente',ahora);
 resp:=jsonb_build_object('catalogo',nuevo,'recibo',jsonb_build_object(
  'recibo_ref',recibo,'clave_idempotencia',clave,'operacion',operacion,'version',nuevo_v,'revision',nuevo_r,
  'catalogo_huella_sha256',catalogo_h,'decision_ref',consumo.decision_ref,
  'auditoria_ref',consumo.auditoria_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
  'registrado_en',ahora,'estado_replay','registrado'),'replay',false);
 RETURN resp;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;
DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
  LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
  WHERE p.oid=f AND x.grantee<>p.proowner AND x.grantee<>'vec_contratacion_temporal_ejecutor'::regrole LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
 END LOOP;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
               WHERE p.oid=f AND (x.grantee NOT IN (p.proowner,'vec_contratacion_temporal_ejecutor'::regrole)
                                  OR x.privilege_type<>'EXECUTE'))
 THEN RAISE EXCEPTION 'CT-131: ACL de fachada incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
