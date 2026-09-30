\set ON_ERROR_STOP on
-- Inbox de avisos externos. ACK significa aceptación durable, nunca SMTP.
-- Depende de Usuarios 000010, CTX16 y la autoridad de auditoría técnica externa.
-- La autoridad auditada y el inbox confirman juntos, incluidos los replays.
-- pg_temp se fija al final para que no suplante catálogos del actor técnico.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000014',0));
DO $pre$ DECLARE esq oid:=to_regnamespace('vec_usuarios_correos_externo'); prop oid:=to_regrole('vec_usuarios_correos_externo_propietario'); f regprocedure:=to_regprocedure('vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(text)'); a regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_auditoria_inbox_externo_v1(text,text,text,text,text,text,text,bigint,text)'); BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR esq IS NULL OR prop IS NULL OR f IS NULL OR a IS NULL
 OR NOT has_function_privilege(prop,a,'EXECUTE')
 OR has_function_privilege('vec_usuarios_ejecutor_externo',a,'EXECUTE')
 OR has_function_privilege('vec_usuarios_ejecutor_interno',a,'EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=a AND prosecdef AND prorettype='text'::regtype AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE oid=esq AND nspowner=prop)
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_usuarios_correos_externo.correos_direccion') AND relowner=prop AND relrowsecurity AND relforcerowsecurity)
 OR to_regclass('vec_usuarios_correos_externo.avisos_inbox') IS NOT NULL
 OR to_regprocedure('vec_usuarios_correos_externo.sobre_correo_json(vec_usuarios_correos_externo.correos_direccion)') IS NULL
 OR to_regprocedure('vec_usuarios_correos_externo.rechazar_cambio_inmutable()') IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AND proowner=prop AND prosecdef)
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE oid=to_regnamespace('vec_usuarios_correos_avisos') AND nspowner=prop)
 OR NOT has_function_privilege(prop,f,'EXECUTE')
 OR has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
 OR NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND prosecdef AND prorettype='text'::regtype AND proconfig @> ARRAY['search_path=pg_catalog, pg_temp'])
 OR EXISTS(SELECT 1 FROM pg_roles WHERE oid=prop AND (rolcanlogin OR rolsuper OR rolbypassrls))
 THEN RAISE EXCEPTION 'Usuarios U14 preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_usuarios_correos_externo_propietario;
-- El inbox sustituye la lectura de direcciones externas desde dentro. Se
-- conserva la función histórica y sus datos; sus concesiones se retiran en
-- esta misma transacción. También se revocan concesiones directas residuales.
REVOKE ALL ON FUNCTION vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_usuarios_ejecutor_interno;
REVOKE ALL ON SCHEMA vec_usuarios_correos_avisos FROM PUBLIC,vec_usuarios_ejecutor_interno;
DO $retirar_fachada$ DECLARE r record; f regprocedure:='vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; BEGIN
 FOR r IN SELECT DISTINCT rol.rolname FROM pg_proc p
  CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  JOIN pg_roles rol ON rol.oid=a.grantee WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,r.rolname);
 END LOOP;
 FOR r IN SELECT DISTINCT rol.rolname FROM pg_namespace n
  CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
  JOIN pg_roles rol ON rol.oid=a.grantee WHERE n.oid='vec_usuarios_correos_avisos'::regnamespace AND a.grantee<>n.nspowner LOOP
  EXECUTE format('REVOKE ALL ON SCHEMA vec_usuarios_correos_avisos FROM %I',r.rolname);
 END LOOP;
END $retirar_fachada$;
CREATE TABLE vec_usuarios_correos_externo.avisos_inbox (
 productor_ref text NOT NULL,
 evento_ref text NOT NULL,
 material text NOT NULL CHECK(octet_length(material) BETWEEN 2 AND 4096),
 huella text NOT NULL CHECK(huella ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE CHECK(recibo_ref ~ '^aviso_recibo:[0-9a-f]{32}$'),
 aceptado_en timestamptz(6) NOT NULL,
 PRIMARY KEY(productor_ref,evento_ref)
);
CREATE TABLE vec_usuarios_correos_externo.avisos_reserva (
 recibo_ref text PRIMARY KEY REFERENCES vec_usuarios_correos_externo.avisos_inbox(recibo_ref),
 reserva_sha256 text NOT NULL UNIQUE CHECK(reserva_sha256 ~ '^[0-9a-f]{64}$'),
 persona_ref text CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 xid xid8 NOT NULL,
 backend_pid integer NOT NULL,
 sesion text NOT NULL,
 reservado_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_usuarios_correos_externo.avisos_historia (
 recibo_ref text NOT NULL REFERENCES vec_usuarios_correos_externo.avisos_inbox(recibo_ref),
 accion text NOT NULL CHECK(accion IN ('aceptar','reservar','aceptado','no_aceptado','reservado_incierto','sin_destino','replay_aceptar','replay_reservar','replay_confirmar','denegado_aceptar','denegado_reservar','denegado_confirmar')),
 registrado_en timestamptz(6) NOT NULL,
 correlacion_ref text NOT NULL,
 sesion text NOT NULL,
 auditoria_ref text PRIMARY KEY CHECK(auditoria_ref ~ '^auditoria_tecnica_externa:[0-9a-f]{32}$')
);
CREATE UNIQUE INDEX avisos_un_cambio ON vec_usuarios_correos_externo.avisos_historia(recibo_ref,accion)
 WHERE accion IN ('aceptar','reservar','aceptado','no_aceptado','reservado_incierto','sin_destino');
-- reservado_incierto es un resultado durable. Su reserva permanece inmutable
-- y excluye tanto otro despacho como sustituirlo por aceptación o rechazo.
CREATE UNIQUE INDEX avisos_un_resultado ON vec_usuarios_correos_externo.avisos_historia(recibo_ref)
 WHERE accion IN ('aceptado','no_aceptado','reservado_incierto','sin_destino');
CREATE FUNCTION vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1()
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_user='vec_usuarios_correos_externo_propietario' AND session_user='vec_externo_avisos_usuarios'
 AND EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND rolinherit
   AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
 AND (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)=1
 AND EXISTS(SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole
  AND roleid='vec_usuarios_ejecutor_externo'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_usuarios_ejecutor_externo'::regrole)
$f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1() FROM PUBLIC,vec_usuarios_ejecutor_interno;
GRANT EXECUTE ON FUNCTION vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1() TO vec_usuarios_ejecutor_externo;
DO $tablas$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['avisos_inbox','avisos_reserva','avisos_historia'] LOOP
  EXECUTE format('ALTER TABLE vec_usuarios_correos_externo.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_usuarios_correos_externo.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_usuarios_correos_externo.%I FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo',t);
  EXECUTE format('CREATE POLICY avisos_lectura ON vec_usuarios_correos_externo.%I FOR SELECT TO vec_usuarios_correos_externo_propietario USING (vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1())',t);
  EXECUTE format('CREATE POLICY avisos_alta ON vec_usuarios_correos_externo.%I FOR INSERT TO vec_usuarios_correos_externo_propietario WITH CHECK (vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1())',t);
  EXECUTE format('CREATE TRIGGER avisos_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_usuarios_correos_externo.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios_correos_externo.rechazar_cambio_inmutable()',t);
 END LOOP;
END $tablas$;
-- El sobre sólo puede leerse en la transacción que acaba de reservar el aviso.
-- La reserva no contiene dirección ni cifrado y nunca se modifica.
CREATE FUNCTION vec_usuarios_correos_externo.contexto_inbox_direccion_v1(p_persona text)
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1()
 AND EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.avisos_reserva r
  WHERE r.persona_ref=p_persona AND r.xid=pg_current_xact_id_if_assigned()
   AND r.backend_pid=pg_backend_pid() AND r.sesion=session_user)
$f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.contexto_inbox_direccion_v1(text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE POLICY correos_direccion_inbox ON vec_usuarios_correos_externo.correos_direccion
 FOR SELECT TO vec_usuarios_correos_externo_propietario
 USING (activo AND estado='verificado' AND vec_usuarios_correos_externo.contexto_inbox_direccion_v1(persona_ref));
CREATE FUNCTION vec_usuarios_correos_externo.recibo_inbox_json_v1(i vec_usuarios_correos_externo.avisos_inbox,p_replay boolean)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT jsonb_build_object('recibo_ref',i.recibo_ref,'huella',i.huella,
 'aceptado_en',to_char(i.aceptado_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),'replay',p_replay)
$f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.recibo_inbox_json_v1(vec_usuarios_correos_externo.avisos_inbox,boolean) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
-- Estados auditables del inbox, sin dirección, token ni sobre. La preimagen
-- incluye la huella del evento cerrado y el estado durable de la operación.
CREATE FUNCTION vec_usuarios_correos_externo.huella_estado_inbox_v1(p_huella text,p_estado text)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT encode(sha256(convert_to('vec.usuarios.inbox-externo.v1:'||p_huella||':'||p_estado,'UTF8')),'hex')
$f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.huella_estado_inbox_v1(text,text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE FUNCTION vec_usuarios_correos_externo.estado_actual_inbox_v1(p_recibo text)
RETURNS text LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT coalesce((SELECT accion FROM vec_usuarios_correos_externo.avisos_historia
  WHERE recibo_ref=p_recibo AND accion IN ('aceptado','no_aceptado','reservado_incierto','sin_destino')),
  (CASE WHEN EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.avisos_reserva WHERE recibo_ref=p_recibo) THEN 'reservado' ELSE 'inbox' END))
$f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.estado_actual_inbox_v1(text) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE FUNCTION vec_usuarios_correos_externo.auditar_historia_inbox_v1(
 i vec_usuarios_correos_externo.avisos_inbox,p_historia text,p_accion text,p_resultado text,p_antes text,p_despues text,p_version bigint)
RETURNS text LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE a text; BEGIN
 a:=vec_autorizacion_atestada_v3.registrar_auditoria_inbox_externo_v1(p_accion,i.productor_ref,i.evento_ref,i.recibo_ref,i.material::jsonb->>'correlacion_ref',p_antes,p_despues,p_version,p_resultado);
 IF a IS NULL OR a !~ '^auditoria_tecnica_externa:[0-9a-f]{32}$' THEN RAISE EXCEPTION 'auditoria no disponible' USING ERRCODE='XX000'; END IF;
 INSERT INTO vec_usuarios_correos_externo.avisos_historia VALUES(i.recibo_ref,p_historia,clock_timestamp(),i.material::jsonb->>'correlacion_ref',session_user,a);
 RETURN a;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.auditar_historia_inbox_v1(vec_usuarios_correos_externo.avisos_inbox,text,text,text,text,text,bigint) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
-- Sólo errores de negocio controlados llegan aquí. La denegación se confirma
-- sin efecto; el fallo de su propia auditoría aborta la transacción completa.
CREATE FUNCTION vec_usuarios_correos_externo.denegar_inbox_v1(
 i vec_usuarios_correos_externo.avisos_inbox,p_accion text,p_error text,p_estado text,p_version bigint)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE a text; h text:=repeat('0',64); estado text; v bigint; BEGIN
 IF i.recibo_ref IS NULL THEN
  a:=vec_autorizacion_atestada_v3.registrar_auditoria_inbox_externo_v1(p_accion,'productor:no_disponible','evento:no_disponible',NULL,'correlacion:no_disponible',h,h,0,'denegado');
  IF a IS NULL OR a !~ '^auditoria_tecnica_externa:[0-9a-f]{32}$' THEN RAISE EXCEPTION 'auditoria no disponible' USING ERRCODE='XX000'; END IF;
 ELSE
  estado:=vec_usuarios_correos_externo.estado_actual_inbox_v1(i.recibo_ref);
  v:=(CASE estado WHEN 'inbox' THEN 1 WHEN 'reservado' THEN 2 ELSE 3 END);
  h:=vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,estado);
  a:=vec_usuarios_correos_externo.auditar_historia_inbox_v1(i,'denegado_'||p_accion,p_accion,'denegado',h,h,v);
 END IF;
 RETURN jsonb_build_object('error',p_error,'auditoria_ref',a);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.denegar_inbox_v1(vec_usuarios_correos_externo.avisos_inbox,text,text,text,bigint) FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
CREATE FUNCTION vec_usuarios_correos_externo.aceptar_aviso_externo_v1(p_material text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE m jsonb; canon text; k text; i vec_usuarios_correos_externo.avisos_inbox; h text; t timestamptz; a text; estado_h text; estado text; v bigint; BEGIN
 IF NOT vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1() THEN RAISE EXCEPTION 'ejecutor no admitido' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material)>4096 THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 BEGIN m:=p_material::jsonb; EXCEPTION WHEN invalid_text_representation THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END;
 IF jsonb_typeof(m)<>'object' THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 IF (SELECT count(*) FROM jsonb_object_keys(m))<>10 THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 FOREACH k IN ARRAY ARRAY['evento_ref','productor_ref','tipo_versionado','ocurrido_en','correlacion_ref','destinatario_externo_ref','comunicacion_ref','plantilla_ref','plantilla_version','recurso_publico_ref'] LOOP
  IF jsonb_typeof(m->k) IS DISTINCT FROM 'string' THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 END LOOP;
 FOREACH k IN ARRAY ARRAY['evento_ref','productor_ref','correlacion_ref','plantilla_ref','plantilla_version'] LOOP
  IF m->>k !~ '^[A-Za-z0-9:._-]{1,255}[A-Za-z0-9:._-]?$' THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 END LOOP;
 IF m->>'tipo_versionado'<>'vec.bolsa.aviso-llamamiento.v1'
 OR m->>'destinatario_externo_ref' !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR m->>'comunicacion_ref' !~ '^llamamiento:[0-9a-f]{64}$'
 OR ((m->>'recurso_publico_ref')<>'' AND m->>'recurso_publico_ref' !~ '^[A-Za-z0-9:._-]{1,255}[A-Za-z0-9:._-]?$')
 OR m->>'ocurrido_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 BEGIN t:=(m->>'ocurrido_en')::timestamptz; EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END;
 IF to_char(t AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')<>m->>'ocurrido_en' THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 canon:='{"evento_ref":'||to_json(m->>'evento_ref')::text||',"productor_ref":'||to_json(m->>'productor_ref')::text
 ||',"tipo_versionado":'||to_json(m->>'tipo_versionado')::text||',"ocurrido_en":'||to_json(m->>'ocurrido_en')::text
 ||',"correlacion_ref":'||to_json(m->>'correlacion_ref')::text||',"destinatario_externo_ref":'||to_json(m->>'destinatario_externo_ref')::text
 ||',"comunicacion_ref":'||to_json(m->>'comunicacion_ref')::text||',"plantilla_ref":'||to_json(m->>'plantilla_ref')::text
 ||',"plantilla_version":'||to_json(m->>'plantilla_version')::text||',"recurso_publico_ref":'||to_json(m->>'recurso_publico_ref')::text||'}';
 IF p_material<>canon THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','invalido','inbox',0); END IF;
 h:=encode(sha256(convert_to(canon,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('u14:'||(m->>'productor_ref')||':'||(m->>'evento_ref'),0));
 SELECT * INTO i FROM vec_usuarios_correos_externo.avisos_inbox WHERE productor_ref=m->>'productor_ref' AND evento_ref=m->>'evento_ref';
 IF FOUND THEN
  IF i.huella<>h OR i.material<>canon THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'aceptar','conflicto','inbox',1); END IF;
  estado:=vec_usuarios_correos_externo.estado_actual_inbox_v1(i.recibo_ref);
  v:=(CASE estado WHEN 'inbox' THEN 1 WHEN 'reservado' THEN 2 ELSE 3 END);
  estado_h:=vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,estado);
  a:=vec_usuarios_correos_externo.auditar_historia_inbox_v1(i,'replay_aceptar','aceptar','replay',estado_h,estado_h,v);
  RETURN jsonb_build_object('recibo',vec_usuarios_correos_externo.recibo_inbox_json_v1(i,true),'auditoria_ref',a);
 END IF;
 INSERT INTO vec_usuarios_correos_externo.avisos_inbox VALUES(m->>'productor_ref',m->>'evento_ref',canon,h,
 'aviso_recibo:'||replace(gen_random_uuid()::text,'-',''),clock_timestamp()) RETURNING * INTO i;
 a:=vec_usuarios_correos_externo.auditar_historia_inbox_v1(i,'aceptar','aceptar','aceptado',repeat('0',64),vec_usuarios_correos_externo.huella_estado_inbox_v1(h,'inbox'),1);
 RETURN jsonb_build_object('recibo',vec_usuarios_correos_externo.recibo_inbox_json_v1(i,false),'auditoria_ref',a);
END $f$;
CREATE FUNCTION vec_usuarios_correos_externo.reservar_aviso_externo_v1(p_recibo text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE i vec_usuarios_correos_externo.avisos_inbox; m jsonb; persona text; token text; d vec_usuarios_correos_externo.correos_direccion; resultado jsonb; estado text; a text; estado_h text; version_actual bigint; BEGIN
 IF NOT vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1() THEN RAISE EXCEPTION 'ejecutor no admitido' USING ERRCODE='42501'; END IF;
 IF p_recibo IS NULL OR p_recibo !~ '^aviso_recibo:[0-9a-f]{32}$' THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'reservar','invalido','inbox',0); END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('u14:despacho:'||p_recibo,0));
 SELECT * INTO i FROM vec_usuarios_correos_externo.avisos_inbox WHERE recibo_ref=p_recibo;
 IF NOT FOUND THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'reservar','no_disponible','inbox',0); END IF;
 IF EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.avisos_reserva WHERE recibo_ref=p_recibo) THEN
  SELECT accion INTO estado FROM vec_usuarios_correos_externo.avisos_historia WHERE recibo_ref=p_recibo AND accion IN ('aceptado','no_aceptado','reservado_incierto','sin_destino');
  version_actual:=(CASE WHEN estado IS NULL THEN 2 ELSE 3 END);
  estado_h:=vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,coalesce(estado,'reservado'));
  a:=vec_usuarios_correos_externo.auditar_historia_inbox_v1(i,'replay_reservar','reservar','replay',estado_h,estado_h,version_actual);
  RETURN jsonb_build_object('recibo',vec_usuarios_correos_externo.recibo_inbox_json_v1(i,true),'estado',coalesce(estado,'reservado'),'replay',true,'auditoria_ref',a);
 END IF;
 m:=i.material::jsonb;
 persona:=vec_contexto_actor_v1.persona_candidato_externo_avisos_v1(m->>'destinatario_externo_ref');
 token:='reserva:'||replace(gen_random_uuid()::text,'-','');
 INSERT INTO vec_usuarios_correos_externo.avisos_reserva VALUES(p_recibo,encode(sha256(convert_to(token,'UTF8')),'hex'),persona,pg_current_xact_id(),pg_backend_pid(),session_user,clock_timestamp());
 a:=vec_usuarios_correos_externo.auditar_historia_inbox_v1(i,'reservar','reservar','reservado',vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,'inbox'),vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,'reservado'),2);
 resultado:=jsonb_build_object('recibo',vec_usuarios_correos_externo.recibo_inbox_json_v1(i,false),'evento',m,'reserva_ref',token,'estado','reservado','replay',false,'auditoria_ref',a);
 IF persona IS NOT NULL THEN
  -- Mismo bloqueo nominal que las mutaciones de Mis correos (U10). No usar
  -- FOR SHARE: añadiría las condiciones RLS UPDATE ajenas a esta lectura.
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_usuarios_correos_externo:correos:'||persona,0));
  SELECT * INTO d FROM vec_usuarios_correos_externo.correos_direccion WHERE persona_ref=persona AND activo AND estado='verificado';
  IF FOUND THEN resultado:=resultado||jsonb_build_object('persona_ref',persona,'correo_ref',d.correo_ref,'sobre',vec_usuarios_correos_externo.sobre_correo_json(d)); END IF;
 END IF;
 RETURN resultado;
END $f$;
CREATE FUNCTION vec_usuarios_correos_externo.confirmar_aviso_externo_v1(p_recibo text,p_token text,p_estado text)
RETURNS jsonb LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE r vec_usuarios_correos_externo.avisos_reserva; i vec_usuarios_correos_externo.avisos_inbox; anterior text; a text; estado_h text; BEGIN
 IF NOT vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1() THEN RAISE EXCEPTION 'ejecutor no admitido' USING ERRCODE='42501'; END IF;
 IF p_recibo IS NULL OR p_recibo !~ '^aviso_recibo:[0-9a-f]{32}$' OR p_token IS NULL OR p_token !~ '^reserva:[0-9a-f]{32}$' OR p_estado IS NULL OR p_estado NOT IN ('aceptado','no_aceptado','reservado_incierto','sin_destino') THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'confirmar','invalido','inbox',0); END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('u14:despacho:'||p_recibo,0));
 SELECT * INTO i FROM vec_usuarios_correos_externo.avisos_inbox WHERE recibo_ref=p_recibo;
 SELECT * INTO r FROM vec_usuarios_correos_externo.avisos_reserva WHERE recibo_ref=p_recibo;
 IF NOT FOUND OR r.reserva_sha256<>encode(sha256(convert_to(p_token,'UTF8')),'hex') OR r.sesion<>session_user THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'confirmar','no_disponible','reservado',(CASE WHEN i.recibo_ref IS NULL THEN 0 ELSE 2 END)); END IF;
 SELECT accion INTO anterior FROM vec_usuarios_correos_externo.avisos_historia WHERE recibo_ref=p_recibo AND accion IN ('aceptado','no_aceptado','reservado_incierto','sin_destino');
 IF FOUND THEN
  IF anterior<>p_estado THEN RETURN vec_usuarios_correos_externo.denegar_inbox_v1(i,'confirmar','conflicto',anterior,3); END IF;
  estado_h:=vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,anterior);
  a:=vec_usuarios_correos_externo.auditar_historia_inbox_v1(i,'replay_confirmar','confirmar','replay',estado_h,estado_h,3);
  RETURN jsonb_build_object('confirmado',true,'auditoria_ref',a);
 END IF;
 SELECT * INTO STRICT i FROM vec_usuarios_correos_externo.avisos_inbox WHERE recibo_ref=p_recibo;
 a:=vec_usuarios_correos_externo.auditar_historia_inbox_v1(i,p_estado,'confirmar',p_estado,vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,'reservado'),vec_usuarios_correos_externo.huella_estado_inbox_v1(i.huella,p_estado),3);
 RETURN jsonb_build_object('confirmado',true,'auditoria_ref',a);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.aceptar_aviso_externo_v1(text),vec_usuarios_correos_externo.reservar_aviso_externo_v1(text),vec_usuarios_correos_externo.confirmar_aviso_externo_v1(text,text,text) FROM PUBLIC,vec_usuarios_ejecutor_interno;
GRANT EXECUTE ON FUNCTION vec_usuarios_correos_externo.aceptar_aviso_externo_v1(text),vec_usuarios_correos_externo.reservar_aviso_externo_v1(text),vec_usuarios_correos_externo.confirmar_aviso_externo_v1(text,text,text) TO vec_usuarios_ejecutor_externo;
REVOKE ALL ON TYPE vec_usuarios_correos_externo.avisos_inbox,vec_usuarios_correos_externo.avisos_reserva,vec_usuarios_correos_externo.avisos_historia FROM PUBLIC,vec_usuarios_ejecutor_interno,vec_usuarios_ejecutor_externo;
RESET ROLE;
DO $post_fachada$ DECLARE f regprocedure:='vec_usuarios_correos_avisos.correo_activo_avisos_llamamiento_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; BEGIN
 IF has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
 OR has_schema_privilege('vec_usuarios_ejecutor_interno','vec_usuarios_correos_avisos','USAGE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner)
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.oid='vec_usuarios_correos_avisos'::regnamespace AND a.grantee<>n.nspowner)
 OR EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolcanlogin AND NOT r.rolsuper AND pg_has_role(r.oid,'vec_usuarios_ejecutor_interno','MEMBER')
  AND (has_function_privilege(r.oid,f,'EXECUTE') OR has_schema_privilege(r.oid,'vec_usuarios_correos_avisos','USAGE')))
 THEN RAISE EXCEPTION 'Usuarios U14 fachada interna todavía accesible' USING ERRCODE='55000'; END IF;
END $post_fachada$;
DO $post$ DECLARE t text; f regprocedure; prop oid:='vec_usuarios_correos_externo_propietario'::regrole; BEGIN
 FOREACH t IN ARRAY ARRAY['avisos_inbox','avisos_reserva','avisos_historia'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_usuarios_correos_externo.'||t)
   AND relowner=prop AND relrowsecurity AND relforcerowsecurity)
   OR has_table_privilege('vec_usuarios_ejecutor_externo','vec_usuarios_correos_externo.'||t,'SELECT')
   OR has_table_privilege('vec_usuarios_ejecutor_interno','vec_usuarios_correos_externo.'||t,'SELECT')
   OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
     WHERE c.oid=to_regclass('vec_usuarios_correos_externo.'||t) AND a.grantee<>prop)
  THEN RAISE EXCEPTION 'Usuarios U14 tabla incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY['aceptar_aviso_externo_v1(text)','reservar_aviso_externo_v1(text)','confirmar_aviso_externo_v1(text,text,text)'] LOOP
  f:=to_regprocedure('vec_usuarios_correos_externo.'||t);
  IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE oid=f AND proowner=prop AND prosecdef AND proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
   OR NOT has_function_privilege('vec_usuarios_ejecutor_externo',f,'EXECUTE')
   OR has_function_privilege('vec_usuarios_ejecutor_interno',f,'EXECUTE')
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
     WHERE p.oid=f AND a.grantee NOT IN (prop,'vec_usuarios_ejecutor_externo'::regrole))
  THEN RAISE EXCEPTION 'Usuarios U14 fachada incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $post$;
COMMIT;
