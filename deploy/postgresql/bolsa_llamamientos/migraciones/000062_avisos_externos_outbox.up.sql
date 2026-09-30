\set ON_ERROR_STOP on
-- B62. CTX16 debe preceder a esta migración. El LOGIN consumidor se
-- provisiona fuera de Git, sin contraseña aquí y con su único grupo nominal.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000062',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regrole('vec_bolsa_avisos_externos_consumidor') IS NOT NULL
    OR to_regclass('vec_bolsa_llamamientos.aviso_externo_outbox') IS NOT NULL
    OR to_regprocedure('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.registrar_contactos_llamamiento_v1(text,text,text,bytea,jsonb)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text)') IS NULL
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(text,text,text,text,text,text,text,bigint,text)','EXECUTE')
    OR to_regprocedure('vec_contexto_actor_v1.es_candidato_externo_avisos_v1(text)') IS NULL
    OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario','vec_contexto_actor_v1.es_candidato_externo_avisos_v1(text)','EXECUTE')
 THEN RAISE EXCEPTION 'B62: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_bolsa_avisos_externos_consumidor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_avisos_externos_consumidor',current_database());
END $conexion$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
CREATE TABLE vec_bolsa_llamamientos.aviso_externo_outbox(
 productor_ref text NOT NULL CHECK(productor_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$'),
 recibo_outbox_ref text NOT NULL UNIQUE CHECK(recibo_outbox_ref ~ '^recibo_outbox:[0-9a-f]{64}$'),
 evento_ref text NOT NULL CHECK(evento_ref ~ '^evento_aviso:[0-9a-f]{64}$'),
 llamamiento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref),
 participacion_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.vinculo_candidato(participacion_ref),
 evento jsonb NOT NULL CHECK(jsonb_typeof(evento)='object'),
 canon text NOT NULL CHECK(octet_length(canon) BETWEEN 1 AND 4096),
 huella_sha256 text NOT NULL CHECK(huella_sha256=encode(sha256(convert_to(canon,'UTF8')),'hex')),
 registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(productor_ref,evento_ref), UNIQUE(llamamiento_ref,participacion_ref)
);
CREATE TABLE vec_bolsa_llamamientos.aviso_externo_aceptacion(
 productor_ref text NOT NULL,
 evento_ref text NOT NULL,
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_externo_ref text NOT NULL UNIQUE CHECK(recibo_externo_ref ~ '^aviso_recibo:[0-9a-f]{32}$'),
 aceptada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(productor_ref,evento_ref),
 FOREIGN KEY(productor_ref,evento_ref) REFERENCES vec_bolsa_llamamientos.aviso_externo_outbox(productor_ref,evento_ref)
);
CREATE TABLE vec_bolsa_llamamientos.aviso_externo_resultado(
 productor_ref text NOT NULL,
 evento_ref text NOT NULL,
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 9007199254740991),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_externo_ref text NOT NULL CHECK(recibo_externo_ref ~ '^aviso_recibo:[0-9a-f]{32}$'),
 estado text NOT NULL CHECK(estado IN('aceptado','no_aceptado','sin_destino','reservado_incierto')),
 registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(productor_ref,evento_ref,version),
 UNIQUE(productor_ref,evento_ref,recibo_externo_ref,estado),
 FOREIGN KEY(productor_ref,evento_ref) REFERENCES vec_bolsa_llamamientos.aviso_externo_outbox(productor_ref,evento_ref)
);
CREATE INDEX aviso_externo_outbox_pendientes ON vec_bolsa_llamamientos.aviso_externo_outbox(registrada_en,productor_ref,evento_ref);
DO $tablas$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['aviso_externo_outbox','aviso_externo_aceptacion','aviso_externo_resultado'] LOOP
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY solo_propietario ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario USING(current_user=''vec_bolsa_llamamientos_propietario'') WITH CHECK(current_user=''vec_bolsa_llamamientos_propietario'')',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_bolsa_llamamientos.%I FROM PUBLIC,vec_bolsa_avisos_externos_consumidor,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_bolsa_llamamientos.%I FROM PUBLIC,vec_bolsa_avisos_externos_consumidor,vec_bolsa_llamamientos_ejecutor,vec_bolsa_llamamientos_portal_externo',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.%I FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_bolsa_llamamientos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion()',t);
 END LOOP;
END $tablas$;

-- Las cadenas canónicas admiten sólo ASCII opaco: ninguna necesita escape JSON.
-- El orden coincide exactamente con json.Marshal del DTO neutral (recurso siempre presente).
CREATE FUNCTION vec_bolsa_llamamientos.canon_aviso_externo_v1(p_evento jsonb)
RETURNS text LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE k text; v text; BEGIN
 IF p_evento IS NULL OR jsonb_typeof(p_evento)<>'object' THEN
  RAISE EXCEPTION 'B62: evento invalido' USING ERRCODE='22023'; END IF;
 IF (SELECT array_agg(key ORDER BY key) FROM jsonb_each(p_evento)) IS DISTINCT FROM
   ARRAY['comunicacion_ref','correlacion_ref','destinatario_externo_ref','evento_ref','ocurrido_en','plantilla_ref','plantilla_version','productor_ref','recurso_publico_ref','tipo_versionado']
 THEN RAISE EXCEPTION 'B62: propiedades de evento invalidas' USING ERRCODE='22023'; END IF;
 FOR k,v IN SELECT key,value#>>'{}' FROM jsonb_each(p_evento) LOOP
  IF jsonb_typeof(p_evento->k)<>'string' OR v IS NULL OR octet_length(v)>192
   OR (k='recurso_publico_ref' AND v<>'' AND v !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$')
   OR (k NOT IN('recurso_publico_ref','ocurrido_en') AND v !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$')
  THEN RAISE EXCEPTION 'B62: valor de evento invalido' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'evento_ref' !~ '^evento_aviso:[0-9a-f]{64}$'
    OR p_evento->>'destinatario_externo_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_evento->>'comunicacion_ref' !~ '^llamamiento:[0-9a-f]{64}$'
    OR p_evento->>'tipo_versionado'<>'vec.bolsa.aviso-llamamiento.v1'
    OR p_evento->>'ocurrido_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 THEN RAISE EXCEPTION 'B62: contrato de evento invalido' USING ERRCODE='22023'; END IF;
 RETURN '{"evento_ref":"'||(p_evento->>'evento_ref')||'","productor_ref":"'||(p_evento->>'productor_ref')||'","tipo_versionado":"'||(p_evento->>'tipo_versionado')||'","ocurrido_en":"'||(p_evento->>'ocurrido_en')||'","correlacion_ref":"'||(p_evento->>'correlacion_ref')||'","destinatario_externo_ref":"'||(p_evento->>'destinatario_externo_ref')||'","comunicacion_ref":"'||(p_evento->>'comunicacion_ref')||'","plantilla_ref":"'||(p_evento->>'plantilla_ref')||'","plantilla_version":"'||(p_evento->>'plantilla_version')||'","recurso_publico_ref":"'||(p_evento->>'recurso_publico_ref')||'"}';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.canon_aviso_externo_v1(jsonb) FROM PUBLIC;

-- El puerto devuelve sólo el candidato externo de una participación propia.
CREATE FUNCTION vec_bolsa_llamamientos.destinatario_externo_participacion_v1(p_bolsa text,p_participacion text)
RETURNS text LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
 SELECT v.candidato_ref FROM vec_bolsa_llamamientos.vinculo_candidato v
 JOIN vec_bolsa_llamamientos.constitucion c ON c.instantanea_ref=v.instantanea_ref AND c.version_instantanea=v.version_instantanea AND c.acta_ref=v.acta_ref
 JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=v.participacion_ref
 WHERE c.bolsa_ref=p_bolsa AND v.participacion_ref=p_participacion
 AND vec_contexto_actor_v1.es_candidato_externo_avisos_v1(v.candidato_ref) IS TRUE
$f$;
CREATE FUNCTION vec_bolsa_llamamientos.avisos_externos_llamamiento_v1(p_bolsa text,p_clave text)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
 SELECT coalesce(jsonb_agg(jsonb_build_object('evento',o.evento,'huella',o.huella_sha256,'recibo_outbox_ref',o.recibo_outbox_ref,'estado_despacho',coalesce(despacho.estado,'')) ORDER BY o.participacion_ref),'[]'::jsonb)
 FROM vec_bolsa_llamamientos.llamamiento_emitido l JOIN vec_bolsa_llamamientos.aviso_externo_outbox o ON o.llamamiento_ref=l.llamamiento_ref
 LEFT JOIN LATERAL (SELECT r.estado FROM vec_bolsa_llamamientos.aviso_externo_resultado r WHERE r.productor_ref=o.productor_ref AND r.evento_ref=o.evento_ref ORDER BY r.version DESC LIMIT 1) despacho ON true
 WHERE l.bolsa_ref=p_bolsa AND l.clave_idempotencia=p_clave
$f$;

CREATE FUNCTION vec_bolsa_llamamientos.reservar_llamamiento_avisos_externos_v1(
 p_llamamiento text,p_recibo text,p_bolsa text,p_actor text,p_clave text,p_participaciones jsonb,p_configuracion jsonb,p_emitido timestamptz,p_token_finalizacion bytea,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,p_eventos jsonb)
RETURNS TABLE(emision jsonb,reutilizada boolean) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE reservado record; previo record; consumo record; d jsonb; evento jsonb; v_canon text; huella text;
        participacion text; candidato text; total integer; n integer:=0; vistos text[]:=ARRAY[]::text[]; contactos jsonb; resultado jsonb;
BEGIN
 IF octet_length(p_token_finalizacion) IS DISTINCT FROM 32 OR p_eventos IS NULL OR jsonb_typeof(p_eventos)<>'array' THEN RAISE EXCEPTION 'B62: lista de eventos invalida' USING ERRCODE='22023'; END IF;
 IF jsonb_array_length(p_eventos)>100 OR octet_length(p_eventos::text)>409600 THEN RAISE EXCEPTION 'B62: lista de eventos excesiva' USING ERRCODE='22023'; END IF;
 -- La función original consume V3 y crea la reserva; cualquier validación
 -- posterior fallida revierte también esa reserva y el consumo.
 SELECT * INTO STRICT reservado FROM vec_bolsa_llamamientos.reservar_llamamiento_v1(p_llamamiento,p_recibo,p_bolsa,p_actor,p_clave,p_participaciones,p_configuracion,p_emitido,sha256(p_token_finalizacion),p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO STRICT previo FROM vec_bolsa_llamamientos.llamamiento_emitido WHERE bolsa_ref=p_bolsa AND clave_idempotencia=p_clave;
 IF reservado.reutilizada THEN
  -- La v1 histórica retorna pronto en replay. Esta frontera revalida V3 antes
  -- de recuperar eventos y no confía en aquella autorización histórica.
  SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
  BEGIN d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'B62: decision invalida' USING ERRCODE='42501'; END;
  IF consumo.efecto_ref IS DISTINCT FROM p_bolsa OR d->>'principal_id' IS DISTINCT FROM p_actor OR d->>'accion' IS DISTINCT FROM 'llamamiento.emitir.v1' OR d->>'tipo_recurso' IS DISTINCT FROM 'bolsa_constituida'
  THEN RAISE EXCEPTION 'B62: replay no autorizado' USING ERRCODE='42501'; END IF;
 END IF;
 IF (SELECT count(DISTINCT value->>'productor_ref')>1 OR count(DISTINCT value->>'correlacion_ref')>1 FROM jsonb_array_elements(p_eventos))
 THEN RAISE EXCEPTION 'B62: productor o correlacion divergentes' USING ERRCODE='VBE01'; END IF;
 IF p_llamamiento IS DISTINCT FROM previo.llamamiento_ref THEN RAISE EXCEPTION 'B62: llamamiento divergente' USING ERRCODE='VBE01'; END IF;
 FOR evento IN SELECT value FROM jsonb_array_elements(p_eventos) LOOP
  v_canon:=vec_bolsa_llamamientos.canon_aviso_externo_v1(evento);
  huella:=encode(sha256(convert_to(v_canon,'UTF8')),'hex');
  IF reservado.reutilizada AND NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_outbox o WHERE o.llamamiento_ref=previo.llamamiento_ref AND o.productor_ref=evento->>'productor_ref' AND o.evento_ref=evento->>'evento_ref' AND o.huella_sha256=huella AND o.canon=v_canon)
  THEN RAISE EXCEPTION 'B62: replay de evento divergente' USING ERRCODE='VBE01'; END IF;
  SELECT v.participacion_ref,v.candidato_ref INTO participacion,candidato
   FROM vec_bolsa_llamamientos.vinculo_candidato v
   WHERE v.candidato_ref=evento->>'destinatario_externo_ref' AND previo.participaciones ? v.participacion_ref
   AND evento->>'evento_ref'='evento_aviso:'||encode(sha256(convert_to(previo.llamamiento_ref||chr(31)||v.participacion_ref,'UTF8')),'hex');
  IF NOT FOUND OR participacion=ANY(vistos)
     OR evento->>'comunicacion_ref' IS DISTINCT FROM previo.llamamiento_ref
     OR evento->>'ocurrido_en' IS DISTINCT FROM to_char(previo.emitido_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
     OR evento->>'plantilla_ref' IS DISTINCT FROM previo.configuracion->>'plantilla_version'
     OR evento->>'plantilla_version' IS DISTINCT FROM previo.configuracion->>'plantilla_version'
  THEN RAISE EXCEPTION 'B62: destinatario o plantilla ajenos' USING ERRCODE='22023'; END IF;
  IF vec_contexto_actor_v1.es_candidato_externo_avisos_v1(candidato) IS NOT TRUE
  THEN RAISE EXCEPTION 'B62: candidato externo no vigente' USING ERRCODE='42501'; END IF;
  vistos:=array_append(vistos,participacion); n:=n+1;
  IF reservado.reutilizada THEN
   IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_outbox o WHERE o.llamamiento_ref=previo.llamamiento_ref AND o.participacion_ref=participacion AND o.productor_ref=evento->>'productor_ref' AND o.evento_ref=evento->>'evento_ref' AND o.huella_sha256=huella AND o.canon=v_canon)
   THEN RAISE EXCEPTION 'B62: replay de evento divergente' USING ERRCODE='VBE01'; END IF;
  ELSE
   INSERT INTO vec_bolsa_llamamientos.aviso_externo_outbox(productor_ref,evento_ref,llamamiento_ref,participacion_ref,evento,canon,huella_sha256,recibo_outbox_ref)
    VALUES(evento->>'productor_ref',evento->>'evento_ref',previo.llamamiento_ref,participacion,evento,v_canon,huella,'recibo_outbox:'||encode(sha256(convert_to((evento->>'productor_ref')||chr(31)||(evento->>'evento_ref')||chr(31)||huella,'UTF8')),'hex'));
  END IF;
 END LOOP;
 IF reservado.reutilizada THEN
  SELECT count(*) INTO total FROM vec_bolsa_llamamientos.aviso_externo_outbox WHERE llamamiento_ref=previo.llamamiento_ref;
 ELSE
  SELECT count(*) INTO total FROM vec_bolsa_llamamientos.vinculo_candidato v WHERE previo.participaciones ? v.participacion_ref AND vec_contexto_actor_v1.es_candidato_externo_avisos_v1(v.candidato_ref) IS TRUE;
 END IF;
 IF n<>total THEN RAISE EXCEPTION 'B62: conjunto de eventos divergente' USING ERRCODE='VBE01'; END IF;
 -- Este perfil admite únicamente una emisión totalmente externa. Ausencia,
 -- revocación o ambigüedad no activan el circuito interno ni SMTP local.
 IF n=0 OR n<>jsonb_array_length(previo.participaciones)
 THEN RAISE EXCEPTION 'B62: emision no totalmente externa' USING ERRCODE='42501'; END IF;
 SELECT jsonb_agg(jsonb_build_object('participacion_ref',x.ref,'resultado',CASE despacho.estado WHEN 'aceptado' THEN 'enviado' WHEN 'no_aceptado' THEN 'no_enviado' WHEN 'sin_destino' THEN 'no_enviado' ELSE 'aviso_pendiente' END,'recibo_ref',o.recibo_outbox_ref) ORDER BY x.n)
  INTO contactos FROM jsonb_array_elements_text(previo.participaciones) WITH ORDINALITY x(ref,n)
  JOIN vec_bolsa_llamamientos.aviso_externo_outbox o ON o.llamamiento_ref=previo.llamamiento_ref AND o.participacion_ref=x.ref
  LEFT JOIN LATERAL (SELECT r.estado FROM vec_bolsa_llamamientos.aviso_externo_resultado r WHERE r.productor_ref=o.productor_ref AND r.evento_ref=o.evento_ref ORDER BY r.version DESC LIMIT 1) despacho ON true;
 -- Proyección del hecho durable de cola; no crea contacto ni declara SMTP.
 resultado:=reservado.emision||jsonb_build_object('contactos',contactos,'estado','emitido_pendiente_respuesta');
 RETURN QUERY SELECT resultado||jsonb_build_object('avisos_externos',vec_bolsa_llamamientos.avisos_externos_llamamiento_v1(p_bolsa,p_clave)),reservado.reutilizada;
END $f$;

-- La cuenta de extracción carece de acceso a cualquier tabla, tipo de fila,
-- otra función VEC o esquema VEC, incluso por una concesión directa añadida.
CREATE FUNCTION vec_bolsa_llamamientos.exigir_consumidor_avisos_externos_v1()
RETURNS void LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET statement_timeout='5s' AS $f$
DECLARE login pg_roles%ROWTYPE; grupo pg_roles%ROWTYPE; permitidas oid[];
BEGIN
 SELECT * INTO login FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO grupo FROM pg_roles WHERE rolname='vec_bolsa_avisos_externos_consumidor';
 permitidas:=ARRAY['vec_bolsa_llamamientos.tirar_avisos_externos_v1(integer)'::regprocedure::oid,'vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(text,text,text,text)'::regprocedure::oid,'vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(text,text,text,text,text)'::regprocedure::oid];
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR login.rolname<>'vec_externo_avisos_bolsa'
 OR login.oid IS NULL OR (login.rolvaliduntil IS NOT NULL AND login.rolvaliduntil<=clock_timestamp()) OR NOT login.rolcanlogin OR NOT login.rolinherit OR login.rolsuper OR login.rolcreatedb OR login.rolcreaterole OR login.rolreplication OR login.rolbypassrls
 OR grupo.oid IS NULL OR grupo.rolcanlogin OR NOT grupo.rolinherit OR grupo.rolsuper OR grupo.rolcreatedb OR grupo.rolcreaterole OR grupo.rolreplication OR grupo.rolbypassrls
 OR (SELECT count(*) FROM pg_auth_members WHERE member=login.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=login.oid AND roleid=grupo.oid AND inherit_option AND NOT set_option AND NOT admin_option)
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member=grupo.oid OR roleid=login.oid)
 OR NOT has_schema_privilege(login.oid,'vec_bolsa_llamamientos','USAGE') OR has_schema_privilege(login.oid,'vec_bolsa_llamamientos','CREATE')
 OR EXISTS(SELECT 1 FROM unnest(permitidas) f WHERE NOT has_function_privilege(login.oid,f,'EXECUTE'))
 OR EXISTS(SELECT 1 FROM pg_namespace n WHERE n.nspname LIKE 'vec_%' AND n.nspname<>'vec_bolsa_llamamientos' AND has_schema_privilege(login.oid,n.oid,'USAGE,CREATE'))
 OR EXISTS(SELECT 1 FROM pg_namespace n WHERE n.nspname LIKE 'vec_%' AND (n.nspowner IN(login.oid,grupo.oid) OR EXISTS(SELECT 1 FROM aclexplode(n.nspacl) a WHERE a.grantee=login.oid OR (a.grantee=grupo.oid AND (n.nspname<>'vec_bolsa_llamamientos' OR a.privilege_type<>'USAGE' OR a.is_grantable)))))
 OR EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname LIKE 'vec_%' AND (p.proowner IN(login.oid,grupo.oid) OR (p.oid<>ALL(permitidas) AND has_function_privilege(login.oid,p.oid,'EXECUTE')) OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE (p.oid=ANY(permitidas) AND a.grantee=0) OR a.grantee=login.oid OR (a.grantee=grupo.oid AND (p.oid<>ALL(permitidas) OR a.privilege_type<>'EXECUTE' OR a.is_grantable)))))
 OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%' AND (c.relowner IN(login.oid,grupo.oid) OR EXISTS(SELECT 1 FROM aclexplode(c.relacl) a WHERE a.grantee IN(login.oid,grupo.oid)) OR (CASE WHEN c.relkind IN('r','p','v','m','f') THEN has_table_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') OR has_any_column_privilege(login.oid,c.oid,'SELECT,INSERT,UPDATE,REFERENCES') WHEN c.relkind='S' THEN has_sequence_privilege(login.oid,c.oid,'USAGE,SELECT,UPDATE') ELSE false END)))
 OR EXISTS(SELECT 1 FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%' AND EXISTS(SELECT 1 FROM aclexplode(a.attacl) x WHERE x.grantee IN(login.oid,grupo.oid)))
 OR EXISTS(SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname LIKE 'vec_%' AND (t.typowner IN(login.oid,grupo.oid) OR EXISTS(SELECT 1 FROM aclexplode(t.typacl) a WHERE a.grantee IN(login.oid,grupo.oid))))
 THEN RAISE EXCEPTION 'B62: consumidor nominal invalido' USING ERRCODE='42501'; END IF;
END $f$;
-- La auditoría común guarda su actor técnico y su cadena aparte. El helper
-- devuelve su referencia real; un fallo cancela toda la operación funcional.
CREATE FUNCTION vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1(
 p_accion text,p_productor text,p_evento text,p_recibo text,p_huella text,p_version bigint,p_resultado text,p_correlacion text)
RETURNS text LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE correlacion text; antes text; despues text;
BEGIN
 correlacion:=coalesce(p_correlacion,'corr_tecnica:'||replace(gen_random_uuid()::text,'-',''));
 antes:=coalesce(p_huella,encode(sha256(convert_to('vec.bolsa.outbox.sin_registro.v1','UTF8')),'hex'));
 despues:=encode(sha256(convert_to(jsonb_build_object('tipo','vec.bolsa.outbox.operacion.v1','accion',p_accion,'productor_ref',p_productor,'evento_ref',p_evento,'recibo_ref',p_recibo,'version',p_version,'resultado',p_resultado)::text,'UTF8')),'hex');
 RETURN vec_autorizacion_atestada_v3.registrar_auditoria_outbox_interno_v1(p_accion,p_productor,p_evento,p_recibo,correlacion,antes,despues,p_version,p_resultado);
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.tirar_avisos_externos_v1(p_limite integer)
RETURNS TABLE(evento jsonb,huella text,auditoria_ref text,error_codigo text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE fila record; ref text; n integer:=0;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumidor_avisos_externos_v1();
 IF p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100 THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('extraer',NULL,NULL,NULL,NULL,0,'denegado',NULL);
  RETURN QUERY SELECT NULL::jsonb,NULL::text,ref,'22023'::text; RETURN;
 END IF;
 FOR fila IN SELECT o.evento,o.huella_sha256,o.productor_ref,o.evento_ref FROM vec_bolsa_llamamientos.aviso_externo_outbox o
  WHERE NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_aceptacion a WHERE a.productor_ref=o.productor_ref AND a.evento_ref=o.evento_ref)
  ORDER BY o.registrada_en,o.productor_ref,o.evento_ref LIMIT p_limite LOOP
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('extraer',fila.productor_ref,fila.evento_ref,NULL,fila.huella_sha256,1,'extraido',fila.evento->>'correlacion_ref');
  n:=n+1; RETURN QUERY SELECT fila.evento,fila.huella_sha256,ref,NULL::text;
 END LOOP;
 IF n=0 THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('extraer',NULL,NULL,NULL,NULL,0,'sin_registro',NULL);
  RETURN QUERY SELECT NULL::jsonb,NULL::text,ref,NULL::text;
 END IF;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(p_productor text,p_evento text,p_huella text,p_recibo text)
RETURNS TABLE(aceptada boolean,auditoria_ref text,error_codigo text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE previo record; original record; ref text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumidor_avisos_externos_v1();
 IF p_productor IS NULL OR p_productor !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR p_evento IS NULL OR p_evento !~ '^evento_aviso:[0-9a-f]{64}$' OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$' OR p_recibo IS NULL OR p_recibo !~ '^aviso_recibo:[0-9a-f]{32}$' THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('aceptar',NULL,NULL,NULL,NULL,0,'denegado',NULL);
  RETURN QUERY SELECT false,ref,'22023'::text; RETURN;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:aceptacion-aviso:'||p_productor||':'||p_evento,0));
 SELECT * INTO original FROM vec_bolsa_llamamientos.aviso_externo_outbox o WHERE o.productor_ref=p_productor AND o.evento_ref=p_evento;
 IF NOT FOUND THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('aceptar',NULL,NULL,NULL,NULL,0,'denegado',NULL);
  RETURN QUERY SELECT false,ref,'VBE01'::text; RETURN;
 END IF;
 IF original.huella_sha256 IS DISTINCT FROM p_huella THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('aceptar',p_productor,p_evento,NULL,original.huella_sha256,1,'denegado',original.evento->>'correlacion_ref');
  RETURN QUERY SELECT false,ref,'VBE01'::text; RETURN;
 END IF;
 SELECT * INTO previo FROM vec_bolsa_llamamientos.aviso_externo_aceptacion a WHERE a.productor_ref=p_productor AND a.evento_ref=p_evento;
 IF FOUND THEN
  IF previo.huella_sha256 IS DISTINCT FROM p_huella OR previo.recibo_externo_ref IS DISTINCT FROM p_recibo THEN
   ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('aceptar',p_productor,p_evento,NULL,p_huella,1,'denegado',original.evento->>'correlacion_ref');
   RETURN QUERY SELECT false,ref,'VBE01'::text; RETURN;
  END IF;
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('aceptar',p_productor,p_evento,p_recibo,p_huella,1,'replay',original.evento->>'correlacion_ref');
  RETURN QUERY SELECT true,ref,NULL::text; RETURN;
 END IF;
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.aviso_externo_aceptacion a WHERE a.recibo_externo_ref=p_recibo) THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('aceptar',p_productor,p_evento,NULL,p_huella,1,'denegado',original.evento->>'correlacion_ref');
  RETURN QUERY SELECT false,ref,'VBE01'::text; RETURN;
 END IF;
 INSERT INTO vec_bolsa_llamamientos.aviso_externo_aceptacion(productor_ref,evento_ref,huella_sha256,recibo_externo_ref) VALUES(p_productor,p_evento,p_huella,p_recibo);
 ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('aceptar',p_productor,p_evento,p_recibo,p_huella,1,'aceptado',original.evento->>'correlacion_ref');
 RETURN QUERY SELECT true,ref,NULL::text;
END $f$;
CREATE FUNCTION vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(p_productor text,p_evento text,p_huella text,p_recibo text,p_estado text)
RETURNS TABLE(registrada boolean,auditoria_ref text,error_codigo text) LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE original record; ultimo record; existente record; v_version bigint; ref text;
BEGIN
 PERFORM vec_bolsa_llamamientos.exigir_consumidor_avisos_externos_v1();
 IF p_productor IS NULL OR p_productor !~ '^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' OR p_evento IS NULL OR p_evento !~ '^evento_aviso:[0-9a-f]{64}$' OR p_huella IS NULL OR p_huella !~ '^[0-9a-f]{64}$' OR p_recibo IS NULL OR p_recibo !~ '^aviso_recibo:[0-9a-f]{32}$' OR p_estado IS NULL OR p_estado NOT IN('aceptado','no_aceptado','sin_destino','reservado_incierto') THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('resultado',NULL,NULL,NULL,NULL,0,'denegado',NULL);
  RETURN QUERY SELECT false,ref,'22023'::text; RETURN;
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:resultado-aviso:'||p_productor||':'||p_evento,0));
 SELECT * INTO original FROM vec_bolsa_llamamientos.aviso_externo_outbox o WHERE o.productor_ref=p_productor AND o.evento_ref=p_evento;
 IF NOT FOUND THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('resultado',NULL,NULL,NULL,NULL,0,'denegado',NULL);
  RETURN QUERY SELECT false,ref,'VBE01'::text; RETURN;
 END IF;
 IF original.huella_sha256 IS DISTINCT FROM p_huella THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('resultado',p_productor,p_evento,NULL,original.huella_sha256,1,'denegado',original.evento->>'correlacion_ref');
  RETURN QUERY SELECT false,ref,'VBE01'::text; RETURN;
 END IF;
 SELECT * INTO ultimo FROM vec_bolsa_llamamientos.aviso_externo_resultado r WHERE r.productor_ref=p_productor AND r.evento_ref=p_evento ORDER BY r.version DESC LIMIT 1;
 SELECT * INTO existente FROM vec_bolsa_llamamientos.aviso_externo_resultado r WHERE r.productor_ref=p_productor AND r.evento_ref=p_evento AND r.recibo_externo_ref=p_recibo AND r.estado=p_estado;
 IF FOUND THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('resultado',p_productor,p_evento,p_recibo,p_huella,existente.version,'replay',original.evento->>'correlacion_ref');
  RETURN QUERY SELECT true,ref,NULL::text; RETURN;
 END IF;
 IF ultimo.version IS NOT NULL AND (ultimo.recibo_externo_ref IS DISTINCT FROM p_recibo OR (ultimo.estado<>'reservado_incierto' AND ultimo.estado IS DISTINCT FROM p_estado)) THEN
  ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('resultado',p_productor,p_evento,NULL,p_huella,ultimo.version,'denegado',original.evento->>'correlacion_ref');
  RETURN QUERY SELECT false,ref,'VBE01'::text; RETURN;
 END IF;
 v_version:=coalesce(ultimo.version,0)+1;
 INSERT INTO vec_bolsa_llamamientos.aviso_externo_resultado(productor_ref,evento_ref,version,huella_sha256,recibo_externo_ref,estado) VALUES(p_productor,p_evento,v_version,p_huella,p_recibo,p_estado);
 ref:=vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1('resultado',p_productor,p_evento,p_recibo,p_huella,v_version,p_estado,original.evento->>'correlacion_ref');
 RETURN QUERY SELECT true,ref,NULL::text;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.exigir_consumidor_avisos_externos_v1(),vec_bolsa_llamamientos.auditar_operacion_aviso_externo_v1(text,text,text,text,text,bigint,text,text),vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(text,text,text,text,text),vec_bolsa_llamamientos.tirar_avisos_externos_v1(integer),vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(text,text,text,text),vec_bolsa_llamamientos.destinatario_externo_participacion_v1(text,text),vec_bolsa_llamamientos.avisos_externos_llamamiento_v1(text,text),vec_bolsa_llamamientos.reservar_llamamiento_avisos_externos_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_avisos_externos_consumidor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.tirar_avisos_externos_v1(integer),vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(text,text,text,text),vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(text,text,text,text,text) TO vec_bolsa_avisos_externos_consumidor;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.destinatario_externo_participacion_v1(text,text),vec_bolsa_llamamientos.avisos_externos_llamamiento_v1(text,text),vec_bolsa_llamamientos.reservar_llamamiento_avisos_externos_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,jsonb) TO vec_bolsa_llamamientos_ejecutor;

COMMIT;
