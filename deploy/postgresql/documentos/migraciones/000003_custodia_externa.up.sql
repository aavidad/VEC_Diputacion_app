\set ON_ERROR_STOP on
-- Documentos-3: documentos con custodia externa. VEC registra la referencia
-- opaca que da el custodio, la huella SHA-256 y los metadatos gobernados
-- (tipo, versión, política de conservación). No recibe ni guarda contenido:
-- la tabla no tiene columnas de objeto y la descarga sigue limitada a
-- originales custodiados por VEC. Conserva 000001/000002 e historia.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000003',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regprocedure('vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_documentos.referencia_externa') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='auditoria_operacion_accion_check' AND conrelid='vec_documentos.auditoria_operacion'::regclass)
    OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='outbox_tipo_check' AND conrelid='vec_documentos.outbox'::regclass)
    -- El registro de identificadores parte vacío: se instala junto a
    -- 000001/000002, antes de la primera alta (la secuencia no se ha usado).
    OR (SELECT is_called FROM vec_documentos.numero_interno_seq)
 THEN RAISE EXCEPTION 'Documentos-3: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Referencia dentro del custodio: imprimible, sin espacios, rutas ni comodines.
CREATE FUNCTION vec_documentos.referencia_custodio_v1(p text) RETURNS boolean
LANGUAGE sql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT p IS NOT NULL AND length(p) BETWEEN 3 AND 512 AND p ~ '^[!-~]+$'
  AND p !~ '[/\\*?%]' AND strpos(p,'..')=0
 $f$;

CREATE TABLE vec_documentos.referencia_externa (
 id text PRIMARY KEY CHECK(vec_documentos.referencia_opaca_v1(id)),
 numero_vec text NOT NULL UNIQUE CHECK(numero_vec ~ '^VEC-[0-9]{4}-[0-9]{1,12}$'),
 clave_idempotencia text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(clave_idempotencia)),
 principal_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(principal_ref)),
 modulo_id text NOT NULL CHECK(modulo_id ~ '^[a-z][a-z0-9_.-]{1,127}$'),
 expediente_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(expediente_ref)),
 tipo_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(tipo_ref)),
 version bigint NOT NULL CHECK(version>0),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$' AND huella_sha256<>repeat('0',64)),
 mime text CHECK(mime IS NULL OR (length(mime) BETWEEN 3 AND 255 AND mime ~ '^[a-z0-9.+-]+/[a-z0-9.+-]+$')),
 tamano bigint CHECK(tamano IS NULL OR tamano>0),
 custodio_id text NOT NULL CHECK(custodio_id ~ '^[a-z][a-z0-9_.-]{1,127}$'),
 custodia_ref text NOT NULL CHECK(vec_documentos.referencia_custodio_v1(custodia_ref)),
 politica_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(politica_ref)),
 version_politica bigint NOT NULL CHECK(version_politica>0),
 huella_politica_sha256 text NOT NULL CHECK(huella_politica_sha256 ~ '^[0-9a-f]{64}$'),
 conservacion_hasta timestamptz(6) NOT NULL,
 proteccion text NOT NULL CHECK(proteccion IN ('conservacion','bloqueo')),
 estado_firma text NOT NULL DEFAULT 'pendiente_proveedor' CHECK(estado_firma='pendiente_proveedor'),
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ad3_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(principal_ref,clave_idempotencia),
 -- Un expediente anota una sola vez cada original externo. Varios
 -- justificantes del mismo tipo y versión conviven en un expediente.
 UNIQUE(expediente_ref,custodio_id,custodia_ref,huella_sha256)
);
CREATE INDEX referencia_externa_expediente_idx ON vec_documentos.referencia_externa(expediente_ref,creada_en,id);
ALTER TABLE vec_documentos.referencia_externa ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.referencia_externa FORCE ROW LEVEL SECURITY;
CREATE POLICY alcance_lectura ON vec_documentos.referencia_externa FOR SELECT TO vec_documentos_propietario
 USING (expediente_ref = nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE POLICY alcance_alta ON vec_documentos.referencia_externa FOR INSERT TO vec_documentos_propietario
 WITH CHECK (expediente_ref = nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_documentos.referencia_externa
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.referencia_externa
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();

-- Auditoría y outbox admiten la nueva acción sin tocar su historia.
ALTER TABLE vec_documentos.auditoria_operacion DROP CONSTRAINT auditoria_operacion_accion_check;
ALTER TABLE vec_documentos.auditoria_operacion ADD CONSTRAINT auditoria_operacion_accion_check
 CHECK(accion IN ('documentos.generado.alta','documentos.expediente.listar','documentos.original.descargar','documentos.notificacion.preparar','documentos.externo.registrar'));
ALTER TABLE vec_documentos.outbox DROP CONSTRAINT outbox_tipo_check;
ALTER TABLE vec_documentos.outbox ADD CONSTRAINT outbox_tipo_check
 CHECK(tipo IN ('documento_generado','notificacion_preparada','documento_externo_registrado'));

-- Un mismo identificador no puede nombrar a la vez un original custodiado por
-- VEC y una referencia externa: ambas altas reservan el id en un registro con
-- clave primaria, así la carrera entre las dos termina en 23505.
CREATE TABLE vec_documentos.identificador_documental (
 id text PRIMARY KEY CHECK(vec_documentos.referencia_opaca_v1(id))
);
ALTER TABLE vec_documentos.identificador_documental ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.identificador_documental FORCE ROW LEVEL SECURITY;
CREATE POLICY reserva ON vec_documentos.identificador_documental FOR INSERT TO vec_documentos_propietario WITH CHECK (true);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_documentos.identificador_documental
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.identificador_documental
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE FUNCTION vec_documentos.reservar_identificador_v1() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN
 INSERT INTO vec_documentos.identificador_documental(id) VALUES (NEW.id);
 RETURN NEW;
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'documentos: identificador ya usado' USING ERRCODE='23505';
END $f$;
CREATE TRIGGER identificador_unico BEFORE INSERT ON vec_documentos.documento
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.reservar_identificador_v1();
CREATE TRIGGER identificador_unico BEFORE INSERT ON vec_documentos.referencia_externa
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.reservar_identificador_v1();

CREATE FUNCTION vec_documentos.proyectar_referencia_externa_v1(p vec_documentos.referencia_externa)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT jsonb_build_object(
  'id',p.id,'numero_vec',p.numero_vec,'modulo_id',p.modulo_id,
  'expediente_ref',p.expediente_ref,'tipo_ref',p.tipo_ref,'version',p.version,
  'mime',coalesce(p.mime,''),'huella_sha256',p.huella_sha256,'tamano',coalesce(p.tamano,0),
  'custodia','externa','custodio_id',p.custodio_id,'custodia_ref',p.custodia_ref,
  'politica_ref',p.politica_ref,'version_politica',p.version_politica,
  'huella_politica_sha256',p.huella_politica_sha256,
  'conservacion_hasta',p.conservacion_hasta,'proteccion',p.proteccion,
  'estado_firma',p.estado_firma,'creado_en',p.creada_en)
 $f$;

CREATE FUNCTION vec_documentos.registrar_referencia_externa_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; e vec_documentos.referencia_externa%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text; nuevo boolean:=false;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR p_auth IS NULL OR p_preimagen IS NULL
 THEN RAISE EXCEPTION 'documentos: registro externo denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1)
    IS DISTINCT FROM ARRAY['accion','clave_idempotencia','conservacion_hasta','custodia_ref','custodio_id','expediente_ref','huella_politica_sha256','huella_sha256','id','mime','modulo_id','politica_ref','proteccion','tamano','tipo_ref','version','version_politica']
    OR m->>'accion'<>'documentos.externo.registrar'
    OR m->>'id' IS DISTINCT FROM p_auth->>'recurso_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_auth->>'ambito_ref'
    OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'registrar_documento_externo'
    OR jsonb_typeof(m->'tamano')<>'number' OR jsonb_typeof(m->'version')<>'number' OR jsonb_typeof(m->'version_politica')<>'number'
    OR jsonb_typeof(m->'mime')<>'string'
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR m->>'huella_politica_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'proteccion' NOT IN ('conservacion','bloqueo')
    OR NOT vec_documentos.referencia_custodio_v1(m->>'custodia_ref')
 THEN RAISE EXCEPTION 'documentos: registro externo no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v2(
  p_preimagen,'documentos.externo.registrar',p_auth->>'recurso_ref',p_auth->>'principal_id',
  p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'expediente_ref',true);
 h:=encode(sha256(p_preimagen),'hex');
 SELECT * INTO e FROM vec_documentos.referencia_externa
  WHERE principal_ref=p_auth->>'principal_id' AND clave_idempotencia=m->>'clave_idempotencia';
 IF v.consumo_nuevo IS FALSE AND NOT FOUND THEN
  RAISE EXCEPTION 'documentos: replay sin registro confirmado' USING ERRCODE='42501';
 END IF;
 IF FOUND THEN
  IF (v.consumo_nuevo IS FALSE AND (e.decision_ref IS DISTINCT FROM v.decision_ref OR e.auditoria_ad3_ref IS DISTINCT FROM v.auditoria_ref))
     OR e.huella_preimagen_sha256 IS DISTINCT FROM h OR e.id IS DISTINCT FROM m->>'id'
  THEN RAISE EXCEPTION 'documentos: clave idempotente reutilizada' USING ERRCODE='23505'; END IF;
 ELSE
  INSERT INTO vec_documentos.referencia_externa(
    id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,
    huella_sha256,mime,tamano,custodio_id,custodia_ref,
    politica_ref,version_politica,huella_politica_sha256,conservacion_hasta,proteccion,
    huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
  VALUES(m->>'id','VEC-'||to_char(ahora,'YYYY')||'-'||nextval('vec_documentos.numero_interno_seq')::text,
    m->>'clave_idempotencia',p_auth->>'principal_id',m->>'modulo_id',m->>'expediente_ref',m->>'tipo_ref',(m->>'version')::bigint,
    m->>'huella_sha256',nullif(m->>'mime',''),nullif((m->>'tamano')::bigint,0),m->>'custodio_id',m->>'custodia_ref',
    m->>'politica_ref',(m->>'version_politica')::bigint,m->>'huella_politica_sha256',(m->>'conservacion_hasta')::timestamptz,
    m->>'proteccion',h,v.decision_ref,v.auditoria_ref,ahora) RETURNING * INTO e;
  INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
   VALUES('documento_externo_registrado',e.id,e.expediente_ref,e.huella_sha256,ahora);
  nuevo:=true;
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.externo.registrar',e.id,e.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,
  CASE WHEN nuevo THEN 'creado' ELSE 'repetido' END,ahora);
 RETURN vec_documentos.proyectar_referencia_externa_v1(e);
END $f$;

-- Lista v2: originales VEC y referencias externas del mismo expediente, con
-- el mismo cursor (creada_en,id). Cada elemento declara su custodia.
CREATE FUNCTION vec_documentos.listar_expediente_v2(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; fila record; resultado jsonb:='[]'::jsonb;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
 v_limite integer; v_total integer:=0; v_cursor text:=''; v_desde timestamptz(6):='-infinity';
BEGIN
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: lista inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['accion','cursor','expediente_ref','limite']
    OR m->>'accion'<>'documentos.expediente.listar' OR m->>'expediente_ref' IS DISTINCT FROM p_auth->>'recurso_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_auth->>'ambito_ref' OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'listar_documentos_expediente'
    OR jsonb_typeof(m->'limite')<>'number' OR m->>'limite' !~ '^[0-9]{1,3}$'
    OR (m->>'limite')::integer NOT BETWEEN 1 AND 100
    OR jsonb_typeof(m->'cursor')<>'string'
 THEN RAISE EXCEPTION 'documentos: lista denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v1(p_preimagen,'documentos.expediente.listar',
  p_auth->>'recurso_ref',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'expediente_ref',true);
 v_limite:=(m->>'limite')::integer;
 IF m->>'cursor'<>'' THEN
  SELECT u.creada_en,u.id INTO v_desde,v_cursor FROM (
    SELECT creada_en,id FROM vec_documentos.documento WHERE id=m->>'cursor' AND expediente_ref=m->>'expediente_ref'
    UNION ALL
    SELECT creada_en,id FROM vec_documentos.referencia_externa WHERE id=m->>'cursor' AND expediente_ref=m->>'expediente_ref') u;
  IF NOT FOUND THEN RAISE EXCEPTION 'documentos: cursor inválido' USING ERRCODE='22023'; END IF;
 END IF;
 FOR fila IN
  SELECT u.creada_en,u.id,u.proyeccion FROM (
    SELECT d.creada_en,d.id,vec_documentos.proyectar_documento_v1(d)||jsonb_build_object('custodia','vec') AS proyeccion
      FROM vec_documentos.documento d
     WHERE d.expediente_ref=m->>'expediente_ref' AND (d.creada_en,d.id)>(v_desde,v_cursor)
    UNION ALL
    SELECT e.creada_en,e.id,vec_documentos.proyectar_referencia_externa_v1(e)
      FROM vec_documentos.referencia_externa e
     WHERE e.expediente_ref=m->>'expediente_ref' AND (e.creada_en,e.id)>(v_desde,v_cursor)) u
  ORDER BY u.creada_en,u.id LIMIT v_limite+1 LOOP
  v_total:=v_total+1;
  EXIT WHEN v_total>v_limite;
  resultado:=resultado||jsonb_build_array(fila.proyeccion);
  v_cursor:=fila.id;
 END LOOP;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.expediente.listar',m->>'expediente_ref',m->>'expediente_ref',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'autorizado',ahora);
 RETURN jsonb_build_object('items',resultado,'siguiente_cursor',CASE WHEN v_total>v_limite THEN v_cursor ELSE '' END);
END $f$;

REVOKE ALL ON TABLE vec_documentos.referencia_externa, vec_documentos.identificador_documental FROM PUBLIC,vec_documentos_ejecutor;
REVOKE ALL ON FUNCTION vec_documentos.referencia_custodio_v1(text),
 vec_documentos.reservar_identificador_v1(),
 vec_documentos.proyectar_referencia_externa_v1(vec_documentos.referencia_externa),
 vec_documentos.registrar_referencia_externa_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.listar_expediente_v2(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
 vec_documentos.registrar_referencia_externa_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.listar_expediente_v2(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_ejecutor;
COMMIT;
