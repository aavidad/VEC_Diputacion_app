\set ON_ERROR_STOP on
-- M4. Disponibilidad independiente del estado factual y cartera propia externa.
-- Requiere AUT78 y AD250. No modifica las versiones históricas RUM03.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:migracion:000004',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regclass('vec_meritos.hecho_identidad') IS NULL
 OR to_regclass('vec_meritos.disponibilidad_evento_v1') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion.validar_motivo_retirada_meritos_actual_v1(jsonb)') IS NULL
 OR to_regrole('vec_meritos_cartera_externo') IS NULL
 THEN RAISE EXCEPTION 'M4: preimagen o autoridad incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_meritos TO vec_meritos_cartera_externo;
SET LOCAL ROLE vec_meritos_propietario;

CREATE TABLE vec_meritos.disponibilidad_evento_v1 (
 hecho_ref text PRIMARY KEY REFERENCES vec_meritos.hecho_identidad(hecho_ref),
 persona_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(persona_ref)),
 version integer NOT NULL CHECK(version=1),
 estado text NOT NULL CHECK(estado='retirado'),
 hecho_version_esperada integer NOT NULL CHECK(hecho_version_esperada>0),
 motivo jsonb NOT NULL CHECK(jsonb_typeof(motivo)='object'),
 retirada_en timestamptz(6) NOT NULL,
 recibo_ref text NOT NULL UNIQUE CHECK(vec_meritos.referencia_valida_v1(recibo_ref)),
 UNIQUE(hecho_ref,persona_ref),
 FOREIGN KEY(hecho_ref,persona_ref) REFERENCES vec_meritos.hecho_identidad(hecho_ref,persona_ref)
);
CREATE TABLE vec_meritos.retirada_operacion_v1 (
 actor_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(actor_ref)),
 clave text NOT NULL CHECK(vec_meritos.referencia_valida_v1(clave)),
 comando bytea NOT NULL CHECK(octet_length(comando) BETWEEN 1 AND 8192),
 huella_comando text NOT NULL CHECK(huella_comando ~ '^[0-9a-f]{64}$'),
 hecho_ref text NOT NULL UNIQUE REFERENCES vec_meritos.disponibilidad_evento_v1(hecho_ref),
 persona_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(persona_ref)),
 decision_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 recibo jsonb NOT NULL CHECK(jsonb_typeof(recibo)='object'),
 registrada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(actor_ref,clave),
 FOREIGN KEY(hecho_ref,persona_ref) REFERENCES vec_meritos.disponibilidad_evento_v1(hecho_ref,persona_ref)
);
CREATE TABLE vec_meritos.retirada_acceso_v1 (
 acceso_ref text PRIMARY KEY CHECK(vec_meritos.referencia_valida_v1(acceso_ref)),
 actor_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(actor_ref)),
 clave text NOT NULL CHECK(vec_meritos.referencia_valida_v1(clave)),
 persona_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(persona_ref)),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(actor_ref,clave) REFERENCES vec_meritos.retirada_operacion_v1(actor_ref,clave)
);
CREATE TABLE vec_meritos.retirada_outbox_v1 (
 evento_ref text PRIMARY KEY CHECK(vec_meritos.referencia_valida_v1(evento_ref)),
 actor_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(actor_ref)),
 clave text NOT NULL CHECK(vec_meritos.referencia_valida_v1(clave)),
 persona_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(persona_ref)),
 hecho_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(hecho_ref)),
 tipo text NOT NULL CHECK(tipo='meritos.hecho.disponibilidad_retirada.v1'),
 carga jsonb NOT NULL CHECK(jsonb_typeof(carga)='object'),
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(actor_ref,clave),
 FOREIGN KEY(actor_ref,clave) REFERENCES vec_meritos.retirada_operacion_v1(actor_ref,clave)
);
CREATE TABLE vec_meritos.retirada_resultado_v1 (
 auditoria_ref text PRIMARY KEY CHECK(vec_meritos.referencia_valida_v1(auditoria_ref)),
 actor_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(actor_ref)),
 persona_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(persona_ref)),
 hecho_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(hecho_ref)),
 clave text NOT NULL CHECK(vec_meritos.referencia_valida_v1(clave)),
 codigo text NOT NULL CHECK(codigo IN ('confirmada','repetida','conflicto')),
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_meritos.cartera_acceso_v1 (
 acceso_ref text PRIMARY KEY CHECK(vec_meritos.referencia_valida_v1(acceso_ref)),
 persona_ref text NOT NULL CHECK(vec_meritos.referencia_valida_v1(persona_ref)),
 auditoria_ref text NOT NULL UNIQUE,
 correlacion_ref text NOT NULL,
 filtro text NOT NULL CHECK(filtro IN ('todos','disponibles','retirados')),
 corte_en timestamptz(6) NOT NULL,
 total bigint NOT NULL CHECK(total>=0),
 consultada_en timestamptz(6) NOT NULL
);
CREATE INDEX hecho_version_persona_corte_m4_idx
 ON vec_meritos.hecho_version(persona_ref,registrada_en,hecho_ref,version DESC);
DO $seguridad$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['disponibilidad_evento_v1','retirada_operacion_v1','retirada_acceso_v1','retirada_outbox_v1','retirada_resultado_v1','cartera_acceso_v1'] LOOP
  EXECUTE format('ALTER TABLE vec_meritos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_meritos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY lectura_nominal ON vec_meritos.%I FOR SELECT TO vec_meritos_propietario USING (persona_ref=current_setting(''vec_meritos.persona_ref'',true))',t);
  EXECUTE format('CREATE POLICY alta_nominal ON vec_meritos.%I FOR INSERT TO vec_meritos_propietario WITH CHECK (persona_ref=current_setting(''vec_meritos.persona_ref'',true))',t);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_meritos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_meritos.historia_inmutable_v1()',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_meritos.%I FROM PUBLIC,vec_meritos_ejecutor,vec_meritos_externo,vec_meritos_interno,vec_meritos_migrador,vec_meritos_cartera_externo',t);
  EXECUTE format('REVOKE ALL ON TYPE vec_meritos.%I FROM PUBLIC,vec_meritos_ejecutor,vec_meritos_externo,vec_meritos_interno,vec_meritos_migrador,vec_meritos_cartera_externo',t);
 END LOOP;
END $seguridad$;

CREATE FUNCTION vec_meritos.listar_cartera_propia_v1(p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE a record; persona text;limite integer;filtro text;cursor text;corte timestamptz(6);
 total bigint;datos jsonb;siguiente text;recibo text;ahora timestamptz(6);
BEGIN
 IF current_user<>'vec_meritos_propietario' OR session_user=current_user
 OR NOT pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (p_orden ?& ARRAY['esquema','persona_ref','limite','filtro','cursor_hecho_ref',
  'corte_en','correlacion_ref']) IS NOT TRUE
 OR p_orden->>'esquema' IS DISTINCT FROM 'vec.meritos.cartera.propia.v1'
 OR p_orden->>'filtro' IS NULL
 OR p_orden->>'filtro' NOT IN ('todos','disponibles','retirados')
 OR jsonb_typeof(p_orden->'limite') IS DISTINCT FROM 'number'
 OR coalesce(p_orden->>'limite' ~ '^([1-9]|[1-4][0-9]|50)$',false) IS NOT TRUE
 OR vec_meritos.referencia_valida_v1(p_orden->>'persona_ref') IS NOT TRUE
 OR jsonb_typeof(p_orden->'cursor_hecho_ref') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'M4: consulta inválida o sin ejecutor' USING ERRCODE='42501'; END IF;
 persona:=p_orden->>'persona_ref';limite:=(p_orden->>'limite')::integer;
 filtro:=p_orden->>'filtro';cursor:=coalesce(p_orden->>'cursor_hecho_ref','');
 BEGIN corte:=(p_orden->>'corte_en')::timestamptz(6);
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'M4: corte inválido' USING ERRCODE='22023'; END;
 IF corte IS NULL OR corte>clock_timestamp() OR corte<clock_timestamp()-interval '30 days'
 OR (cursor<>'' AND vec_meritos.referencia_valida_v1(cursor) IS NOT TRUE)
 THEN RAISE EXCEPTION 'M4: corte o cursor inválido' USING ERRCODE='22023'; END IF;
 -- El registrador común verifica contexto, sesión, permiso vivo y titularidad,
 -- y asienta la lectura positiva antes de proyectar datos personales.
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF a.auditoria_ref IS NULL OR a.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
 OR a.secuencia IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'M4: auditoría divergente' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec_meritos.persona_ref',persona,true);
 WITH ultimas AS MATERIALIZED (
  SELECT DISTINCT ON (hv.hecho_ref) hv.hecho_ref,hv.version,hv.registro->'hecho' AS hecho
  FROM vec_meritos.hecho_version hv
  WHERE hv.persona_ref=persona AND hv.registrada_en<=corte
  ORDER BY hv.hecho_ref,hv.version DESC
 ),base AS MATERIALIZED (
  SELECT u.hecho_ref,u.version,u.hecho,d.estado AS retirada_estado,
   d.version AS disponibilidad_version,d.recibo_ref,d.retirada_en
  FROM ultimas u LEFT JOIN vec_meritos.disponibilidad_evento_v1 d
   ON d.hecho_ref=u.hecho_ref AND d.persona_ref=persona AND d.retirada_en<=corte
  WHERE filtro='todos' OR (filtro='retirados')=(d.hecho_ref IS NOT NULL)
 ),pagina AS (
  SELECT * FROM base WHERE hecho_ref COLLATE "C">cursor COLLATE "C"
  ORDER BY hecho_ref COLLATE "C" LIMIT limite+1
 )
 SELECT (SELECT count(*) FROM base),
  (SELECT coalesce(jsonb_agg(jsonb_build_object(
   'hecho_ref',p.hecho_ref,'version_hecho',p.version,
   'denominacion',p.hecho->>'denominacion','tipo',p.hecho->>'tipo',
   'disponibilidad',jsonb_build_object('estado',coalesce(p.retirada_estado,'disponible'),
     'version',coalesce(p.disponibilidad_version,0)),
   'retirada',CASE WHEN p.retirada_estado IS NULL THEN NULL ELSE
    jsonb_build_object('recibo_ref',p.recibo_ref,'fecha',to_char(p.retirada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   ORDER BY p.hecho_ref COLLATE "C"),'[]'::jsonb) FROM (SELECT * FROM pagina ORDER BY hecho_ref COLLATE "C" LIMIT limite) p),
  (SELECT CASE WHEN count(*)>limite THEN (SELECT hecho_ref FROM pagina ORDER BY hecho_ref COLLATE "C" OFFSET limite-1 LIMIT 1) ELSE NULL END FROM pagina)
 INTO total,datos,siguiente;
 ahora:=clock_timestamp();recibo:='cartera:'||gen_random_uuid()::text;
 INSERT INTO vec_meritos.cartera_acceso_v1 VALUES(recibo,persona,a.auditoria_ref,a.correlacion_ref,filtro,corte,total,ahora);
 RETURN jsonb_build_object('hechos',datos,'total',total,'cursor_siguiente_hecho_ref',coalesce(siguiente,''),
  'corte_en',to_char(corte AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'recibo_acceso',jsonb_build_object('referencia',recibo,'auditoria_ref',a.auditoria_ref,
   'secuencia',a.secuencia::text,'huella_sha256',a.huella_sha256,
   'correlacion_ref',a.correlacion_ref,'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb) TO vec_meritos_cartera_externo;

CREATE FUNCTION vec_meritos.registrar_resultado_retirada_v1(p_auditoria text,p_actor text,
 p_persona text,p_hecho text,p_clave text,p_codigo text,p_fecha timestamptz)
RETURNS void LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 INSERT INTO vec_meritos.retirada_resultado_v1
  (auditoria_ref,actor_ref,persona_ref,hecho_ref,clave,codigo,registrada_en)
 VALUES(p_auditoria,p_actor,p_persona,p_hecho,p_clave,p_codigo,p_fecha);
END $f$;
REVOKE ALL ON FUNCTION vec_meritos.registrar_resultado_retirada_v1(text,text,text,text,text,text,timestamptz) FROM PUBLIC;

CREATE FUNCTION vec_meritos.retirar_hecho_propio_v1(
 p_comando bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;d jsonb;u record;op vec_meritos.retirada_operacion_v1%ROWTYPE;
 persona text;actor text;hecho text;clave text;huella text;contexto text;
 v_hecho integer;v_actual integer;identidad record;
 fecha timestamptz(6);instante text;recibo text;evento text;r jsonb;
BEGIN
 IF current_user<>'vec_meritos_propietario' OR session_user=current_user
 OR NOT pg_has_role(session_user,'vec_meritos_cartera_externo','MEMBER')
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR p_comando IS NULL OR octet_length(p_comando) NOT BETWEEN 1 AND 8192
 THEN RAISE EXCEPTION 'M4: retirada sin ejecutor' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_comando,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'M4: material inválido' USING ERRCODE='22023'; END;
 IF vec_meritos.claves_exactas_v1(c,
  ARRAY['esquema','actor_ref','persona_ref','hecho_ref','hecho_version_esperada',
   'disponibilidad_version_esperada','motivo','clave_idempotencia']) IS NOT TRUE
 OR c->>'esquema' IS DISTINCT FROM 'vec.meritos.hecho.retirada.v1'
 OR c->>'actor_ref' IS DISTINCT FROM c->>'persona_ref'
 OR vec_meritos.referencia_valida_v1(c->>'actor_ref') IS NOT TRUE
 OR vec_meritos.referencia_valida_v1(c->>'hecho_ref') IS NOT TRUE
 OR vec_meritos.claves_exactas_v1(c->'motivo',
  ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave']) IS NOT TRUE
 OR vec_meritos.referencia_valida_v1(c#>>'{motivo,catalogo_id}') IS NOT TRUE
 OR vec_meritos.referencia_valida_v1(c#>>'{motivo,entrada_clave}') IS NOT TRUE
 OR jsonb_typeof(c#>'{motivo,catalogo_version}') IS DISTINCT FROM 'number'
 OR c#>>'{motivo,catalogo_version}' !~ '^[1-9][0-9]{0,9}$'
 OR c#>>'{motivo,catalogo_huella_sha256}' !~ '^[0-9a-f]{64}$'
 OR vec_meritos.referencia_valida_v1(c->>'clave_idempotencia') IS NOT TRUE
 OR jsonb_typeof(c->'hecho_version_esperada') IS DISTINCT FROM 'number'
 OR c->>'hecho_version_esperada' !~ '^[1-9][0-9]{0,9}$'
 OR (c->>'hecho_version_esperada')::numeric>1073741824
 OR jsonb_typeof(c->'disponibilidad_version_esperada') IS DISTINCT FROM 'number'
 OR c->>'disponibilidad_version_esperada' IS DISTINCT FROM '0'
 THEN RAISE EXCEPTION 'M4: comando inválido' USING ERRCODE='22023'; END IF;
 persona:=c->>'persona_ref';actor:=c->>'actor_ref';hecho:=c->>'hecho_ref';
 clave:=c->>'clave_idempotencia';
 v_hecho:=(c->>'hecho_version_esperada')::integer;
 huella:=encode(sha256(p_comando),'hex');
 contexto:='{"ambitos":{"disponibilidad_version_esperada":"0",'
  ||'"hecho_version_esperada":"'||v_hecho::text||'",'
  ||'"huella_comando_sha256":"'||huella||'",'
  ||'"persona_ref":"'||persona||'"},"atributos":{"propiedad":"persona"}}';
 IF d->>'principal_id' IS DISTINCT FROM actor
 OR d->>'accion' IS DISTINCT FROM 'meritos.cartera.retirar_propio'
 OR d->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'hecho'
 OR d->>'recurso_ref' IS DISTINCT FROM hecho
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_cartera_propia'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM
  encode(sha256(convert_to(contexto,'UTF8')),'hex')
 OR d->'campos_permitidos' IS DISTINCT FROM '["disponibilidad","recibo"]'::jsonb
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal'
 THEN RAISE EXCEPTION 'M4: decisión divergente' USING ERRCODE='42501'; END IF;
 -- La fachada común consume V3 estricta y registra auditoría en esta TX,
 -- antes de que el módulo lea el hecho o compruebe una clave de replay.
 SELECT * INTO STRICT u FROM vec_autorizacion_atestada_v3.consumir_retirada_merito_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF u.consumo_nuevo IS NOT TRUE OR u.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR u.efecto_ref IS DISTINCT FROM hecho OR u.huella_efecto_sha256 IS DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 OR vec_meritos.referencia_valida_v1(u.auditoria_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'M4: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec_meritos.persona_ref',persona,true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_meritos:retirada:clave:'||actor||':'||clave,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_meritos:retirada:hecho:'||hecho,0));
 SELECT * INTO op FROM vec_meritos.retirada_operacion_v1 x
  WHERE x.actor_ref=actor AND x.clave=clave;
 fecha:=date_trunc('microseconds',clock_timestamp());
 IF FOUND THEN
  IF op.persona_ref IS DISTINCT FROM persona OR op.hecho_ref IS DISTINCT FROM hecho
  OR op.comando IS DISTINCT FROM p_comando OR op.huella_comando IS DISTINCT FROM huella
  THEN
   PERFORM vec_meritos.registrar_resultado_retirada_v1(u.auditoria_ref,actor,persona,hecho,clave,'conflicto',fecha);
   RETURN jsonb_build_object('codigo','conflicto','auditoria_ref',u.auditoria_ref,'recibo',NULL);
  END IF;
  INSERT INTO vec_meritos.retirada_acceso_v1 VALUES(
   'acceso:'||gen_random_uuid()::text,actor,clave,persona,
   u.decision_ref,u.auditoria_ref,u.consumo_huella_sha256,fecha);
  PERFORM vec_meritos.registrar_resultado_retirada_v1(u.auditoria_ref,actor,persona,hecho,clave,'repetida',fecha);
  RETURN jsonb_build_object('codigo','repetida',
   'auditoria_ref',u.auditoria_ref,'recibo',op.recibo);
 END IF;
 SELECT i.hecho_ref,i.persona_ref INTO identidad FROM vec_meritos.hecho_identidad i
  WHERE i.hecho_ref=hecho AND i.persona_ref=persona;
 IF NOT FOUND THEN
  PERFORM vec_meritos.registrar_resultado_retirada_v1(u.auditoria_ref,actor,persona,hecho,clave,'conflicto',fecha);
  RETURN jsonb_build_object('codigo','conflicto',
   'auditoria_ref',u.auditoria_ref,'recibo',NULL);
 END IF;
 SELECT max(hv.version) INTO v_actual FROM vec_meritos.hecho_version hv
  WHERE hv.hecho_ref=hecho AND hv.persona_ref=persona;
 IF v_actual IS DISTINCT FROM v_hecho
 OR EXISTS(SELECT 1 FROM vec_meritos.disponibilidad_evento_v1 x WHERE x.hecho_ref=hecho)
 THEN
  PERFORM vec_meritos.registrar_resultado_retirada_v1(u.auditoria_ref,actor,persona,hecho,clave,'conflicto',fecha);
  RETURN jsonb_build_object('codigo','conflicto',
   'auditoria_ref',u.auditoria_ref,'recibo',NULL);
 END IF;
 IF vec_autorizacion.validar_motivo_retirada_meritos_actual_v1(c->'motivo') IS NOT TRUE
 THEN RAISE EXCEPTION 'M4: motivo de retirada no vigente' USING ERRCODE='42501';END IF;
 recibo:='recibo:'||gen_random_uuid()::text;
 evento:='evento:'||gen_random_uuid()::text;
 instante:=to_char(fecha AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 r:=jsonb_build_object('referencia',recibo,'hecho_ref',hecho,'version_hecho',v_hecho,
  'disponibilidad',jsonb_build_object('estado','retirado','version',1),
  'retirado_en',instante,'motivo',c->'motivo','clave_idempotencia',clave,
  'huella_comando_sha256',huella,'auditoria_ref',u.auditoria_ref,
  'evento_ref',evento);
 INSERT INTO vec_meritos.disponibilidad_evento_v1
  (hecho_ref,persona_ref,version,estado,hecho_version_esperada,motivo,retirada_en,recibo_ref)
 VALUES(hecho,persona,1,'retirado',v_hecho,c->'motivo',fecha,recibo);
 INSERT INTO vec_meritos.retirada_operacion_v1
  (actor_ref,clave,comando,huella_comando,hecho_ref,persona_ref,decision_ref,
   consumo_huella_sha256,auditoria_ref,recibo_ref,recibo,registrada_en)
 VALUES(actor,clave,p_comando,huella,hecho,persona,u.decision_ref,
  u.consumo_huella_sha256,u.auditoria_ref,recibo,r,fecha);
 INSERT INTO vec_meritos.retirada_outbox_v1
  (evento_ref,actor_ref,clave,persona_ref,hecho_ref,tipo,carga,creada_en)
 VALUES(evento,actor,clave,persona,hecho,'meritos.hecho.disponibilidad_retirada.v1',
  jsonb_build_object('hecho_ref',hecho,'disponibilidad_version',1,'recibo_ref',recibo),fecha);
 PERFORM vec_meritos.registrar_resultado_retirada_v1(u.auditoria_ref,actor,persona,hecho,clave,'confirmada',fecha);
 RETURN jsonb_build_object('codigo','confirmada',
  'auditoria_ref',u.auditoria_ref,'recibo',r);
END $f$;
REVOKE ALL ON FUNCTION vec_meritos.retirar_hecho_propio_v1(
 bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_meritos.retirar_hecho_propio_v1(
 bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_meritos_cartera_externo;
COMMIT;
