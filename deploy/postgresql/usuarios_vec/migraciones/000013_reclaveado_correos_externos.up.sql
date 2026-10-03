\set ON_ERROR_STOP on
-- Usuarios013: transición offline nominal, sin claves secretas en PostgreSQL.
-- Depende de Usuarios010/012. Instalar y ejecutar con el proceso externo parado.
-- Solo DBA; no concede facultades nuevas a ningún LOGIN de los portales.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='120s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000013',0));
DO $pre$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
 OR to_regprocedure('vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()') IS NULL
 OR to_regprocedure('vec_usuarios_correos_externo.correos_transicion_valida()') IS NULL
 OR to_regnamespace('vec_usuarios_correos_reclaveado') IS NOT NULL
 THEN RAISE EXCEPTION 'Usuarios013: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE SCHEMA vec_usuarios_correos_reclaveado;
REVOKE ALL ON SCHEMA vec_usuarios_correos_reclaveado FROM PUBLIC;
CREATE TABLE vec_usuarios_correos_reclaveado.recibo (
 persona_ref text NOT NULL,
 lote_ref text NOT NULL CHECK(lote_ref ~ '^[A-Za-z0-9:._-]{16,128}$'),
 aprobacion_ref text NOT NULL CHECK(aprobacion_ref ~ '^[A-Za-z0-9:._-]{16,128}$'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_ref text NOT NULL UNIQUE,
 ejecutor text NOT NULL,
 xid xid8 NOT NULL,
 registrado_en timestamptz NOT NULL,
 PRIMARY KEY(persona_ref,lote_ref)
);
CREATE TABLE vec_usuarios_correos_reclaveado.fila (
 persona_ref text NOT NULL,
 lote_ref text NOT NULL,
 tabla text NOT NULL CHECK(tabla IN ('correos_conjunto','correos_direccion','correos_desafio',
  'correos_intento_fallido','correos_historia','correos_recibo','correos_envio')),
 pk jsonb NOT NULL CHECK(jsonb_typeof(pk)='object'),
 preimagen_sha256 text NOT NULL CHECK(preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 postimagen_sha256 text NOT NULL CHECK(postimagen_sha256 ~ '^[0-9a-f]{64}$'),
 xid xid8 NOT NULL,
 PRIMARY KEY(persona_ref,lote_ref,tabla,pk),
 FOREIGN KEY(persona_ref,lote_ref) REFERENCES vec_usuarios_correos_reclaveado.recibo(persona_ref,lote_ref)
);
REVOKE ALL ON ALL TABLES IN SCHEMA vec_usuarios_correos_reclaveado FROM PUBLIC;
REVOKE ALL ON TYPE vec_usuarios_correos_reclaveado.recibo,vec_usuarios_correos_reclaveado.fila FROM PUBLIC;
ALTER TABLE vec_usuarios_correos_reclaveado.recibo ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_usuarios_correos_reclaveado.recibo FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_usuarios_correos_reclaveado.fila ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_usuarios_correos_reclaveado.fila FORCE ROW LEVEL SECURITY;
CREATE TRIGGER recibo_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios_correos_reclaveado.recibo
 FOR EACH ROW EXECUTE FUNCTION vec_usuarios_correos_externo.rechazar_cambio_inmutable();
CREATE TRIGGER fila_inmutable BEFORE UPDATE OR DELETE ON vec_usuarios_correos_reclaveado.fila
 FOR EACH ROW EXECUTE FUNCTION vec_usuarios_correos_externo.rechazar_cambio_inmutable();
CREATE TRIGGER recibo_no_truncar BEFORE TRUNCATE ON vec_usuarios_correos_reclaveado.recibo
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios_correos_externo.rechazar_cambio_inmutable();
CREATE TRIGGER fila_no_truncar BEFORE TRUNCATE ON vec_usuarios_correos_reclaveado.fila
 FOR EACH STATEMENT EXECUTE FUNCTION vec_usuarios_correos_externo.rechazar_cambio_inmutable();
GRANT USAGE ON SCHEMA vec_usuarios_correos_reclaveado TO vec_usuarios_correos_preflight_externo;
GRANT SELECT ON ALL TABLES IN SCHEMA vec_usuarios_correos_reclaveado TO vec_usuarios_correos_preflight_externo;
CREATE POLICY lectura_preflight ON vec_usuarios_correos_reclaveado.recibo
 FOR SELECT TO vec_usuarios_correos_preflight_externo USING(true);
CREATE POLICY lectura_preflight ON vec_usuarios_correos_reclaveado.fila
 FOR SELECT TO vec_usuarios_correos_preflight_externo USING(true);

-- PK nominal por tabla, nunca una identidad deducida solo por el recuento.
CREATE FUNCTION vec_usuarios_correos_reclaveado.pk_fila_v1(p_tabla text,p_fila jsonb)
RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT CASE p_tabla
 WHEN 'correos_conjunto' THEN jsonb_build_object('persona_ref',p_fila->'persona_ref')
 WHEN 'correos_direccion' THEN jsonb_build_object('persona_ref',p_fila->'persona_ref','correo_ref',p_fila->'correo_ref')
 WHEN 'correos_desafio' THEN jsonb_build_object('persona_ref',p_fila->'persona_ref','correo_ref',p_fila->'correo_ref','desafio_ref',p_fila->'desafio_ref')
 WHEN 'correos_intento_fallido' THEN jsonb_build_object('persona_ref',p_fila->'persona_ref','correo_ref',p_fila->'correo_ref','desafio_ref',p_fila->'desafio_ref','numero',p_fila->'numero')
 WHEN 'correos_historia' THEN jsonb_build_object('persona_ref',p_fila->'persona_ref','version',p_fila->'version')
 WHEN 'correos_recibo' THEN jsonb_build_object('persona_ref',p_fila->'persona_ref','clave_operacion',p_fila->'clave_operacion')
 WHEN 'correos_envio' THEN jsonb_build_object('envio_ref',p_fila->'envio_ref') END
$f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_reclaveado.pk_fila_v1(text,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_usuarios_correos_reclaveado.pk_fila_v1(text,jsonb)
 TO vec_usuarios_correos_preflight_externo;
CREATE FUNCTION vec_usuarios_correos_reclaveado.fila_sellada_v1(p_tabla text,p_fila jsonb)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on
 SET timezone='UTC' SET datestyle='ISO, YMD' SET bytea_output='hex' AS $f$
DECLARE k text;
BEGIN
 FOREACH k IN ARRAY ARRAY['actualizado_en','vence_en','creado_en','verificado_en','retirado_en','ocurrido_en','registrada_en','resuelto_en'] LOOP
  IF p_fila->>k IS NOT NULL THEN p_fila:=p_fila||jsonb_build_object(k,(p_fila->>k)::timestamptz); END IF;
 END LOOP;
 RETURN EXISTS(SELECT 1 FROM vec_usuarios_correos_reclaveado.fila s
 WHERE s.persona_ref=p_fila->>'persona_ref' AND s.tabla=p_tabla
 AND s.pk=vec_usuarios_correos_reclaveado.pk_fila_v1(p_tabla,p_fila)
 AND s.postimagen_sha256=encode(sha256(convert_to(p_fila::text,'UTF8')),'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_reclaveado.fila_sellada_v1(text,jsonb) FROM PUBLIC;
ALTER FUNCTION vec_usuarios_correos_reclaveado.fila_sellada_v1(text,jsonb)
 OWNER TO vec_usuarios_correos_preflight_externo;
-- El CHECK de desafíos solo necesita este booleano. No expone el ledger.
GRANT USAGE ON SCHEMA vec_usuarios_correos_reclaveado TO vec_usuarios_correos_externo_propietario;
GRANT EXECUTE ON FUNCTION vec_usuarios_correos_reclaveado.fila_sellada_v1(text,jsonb)
 TO vec_usuarios_correos_externo_propietario;

-- La única excepción del trigger exige DBA y una fila exacta ya registrada
-- en esta misma transacción. El resto conserva las transiciones originales.
CREATE OR REPLACE FUNCTION vec_usuarios_correos_externo.correos_transicion_valida()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog SET timezone='UTC'
 SET datestyle='ISO, YMD' SET bytea_output='hex' AS $f$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'historia de correos inmutable' USING ERRCODE='55000'; END IF;
 IF TG_TABLE_NAME='correos_direccion' THEN
  IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
     AND EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
     THEN
   IF (to_jsonb(NEW)-'clave_sobre_ref'-'clave_igualdad_ref'-'nonce'-'cifrado'-'huella_igualdad')
       = (to_jsonb(OLD)-'clave_sobre_ref'-'clave_igualdad_ref'-'nonce'-'cifrado'-'huella_igualdad')
     AND EXISTS(SELECT 1 FROM vec_usuarios_correos_reclaveado.fila s
       WHERE s.xid=pg_current_xact_id_if_assigned() AND s.tabla='correos_direccion'
        AND s.persona_ref=OLD.persona_ref
        AND s.pk=jsonb_build_object('persona_ref',OLD.persona_ref,'correo_ref',OLD.correo_ref)
        AND s.preimagen_sha256=encode(sha256(convert_to(to_jsonb(OLD)::text,'UTF8')),'hex')
        AND s.postimagen_sha256=encode(sha256(convert_to(to_jsonb(NEW)::text,'UTF8')),'hex'))
  THEN RETURN NEW; END IF;
  END IF;
  IF (to_jsonb(NEW)-'estado'-'activo'-'verificado_en'-'retirado_en') IS DISTINCT FROM (to_jsonb(OLD)-'estado'-'activo'-'verificado_en'-'retirado_en')
     OR OLD.estado='retirado' OR (OLD.estado='verificado' AND NEW.estado='pendiente')
     OR (OLD.verificado_en IS NOT NULL AND NEW.verificado_en IS DISTINCT FROM OLD.verificado_en)
  THEN RAISE EXCEPTION 'transición de correo inválida' USING ERRCODE='55000'; END IF;
 ELSIF TG_TABLE_NAME='correos_desafio' THEN
  IF (to_jsonb(NEW)-'estado'-'intentos') IS DISTINCT FROM (to_jsonb(OLD)-'estado'-'intentos')
     OR OLD.estado<>'pendiente' OR NEW.intentos<OLD.intentos
  THEN RAISE EXCEPTION 'transición de desafío inválida' USING ERRCODE='55000'; END IF;
 ELSIF TG_TABLE_NAME='correos_envio' THEN
  IF (to_jsonb(NEW)-'estado'-'resuelto_en') IS DISTINCT FROM (to_jsonb(OLD)-'estado'-'resuelto_en')
     OR OLD.estado<>'reservado' OR NEW.estado='reservado'
  THEN RAISE EXCEPTION 'transición de envío inválida' USING ERRCODE='55000'; END IF;
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.correos_transicion_valida() FROM PUBLIC;
ALTER TABLE vec_usuarios_correos_externo.correos_desafio DROP CONSTRAINT clave_codigo_externa_check;
-- Las filas anteriores solo se admiten terminales y selladas exactamente.
ALTER TABLE vec_usuarios_correos_externo.correos_desafio ADD CONSTRAINT clave_codigo_externa_check CHECK
 (clave_ref='clave:kms:desarrollo:usuarios-correos-externo-codigo:v1'
 OR (estado<>'pendiente' AND vec_usuarios_correos_reclaveado.fila_sellada_v1('correos_desafio',
    jsonb_build_object('persona_ref',persona_ref,'correo_ref',correo_ref,'desafio_ref',desafio_ref,
     'huella_codigo',huella_codigo,'clave_ref',clave_ref,'vence_en',vence_en,'estado',estado,
     'intentos',intentos,'creado_en',creado_en)))) NOT VALID;

CREATE FUNCTION vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1(p_persona text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET timezone='UTC'
 SET datestyle='ISO, YMD' SET bytea_output='hex' SET lock_timeout='5s' AS $f$
DECLARE t text; filas jsonb:='{}'; a jsonb;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
 THEN RAISE EXCEPTION 'reclaveado: DBA requerido' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'reclaveado: transacción serializable requerida' USING ERRCODE='25001'; END IF;
 IF EXISTS(SELECT 1 FROM pg_stat_activity a WHERE a.datname=current_database()
  AND a.pid<>pg_backend_pid() AND a.usesysid IS NOT NULL
  AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=a.usesysid
   AND m.roleid='vec_usuarios_ejecutor_externo'::regrole))
 THEN RAISE EXCEPTION 'reclaveado: proceso externo conectado' USING ERRCODE='55000'; END IF;
 IF p_persona IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
 THEN RAISE EXCEPTION 'reclaveado: persona inválida' USING ERRCODE='22023'; END IF;
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio','correos_contexto'] LOOP
  EXECUTE format('LOCK TABLE vec_usuarios_correos_externo.%I IN ACCESS EXCLUSIVE MODE',t);
 END LOOP;
 LOCK TABLE vec_usuarios_correos_reclaveado.recibo,vec_usuarios_correos_reclaveado.fila IN ACCESS EXCLUSIVE MODE;
 IF EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_contexto)
 OR EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_envio WHERE estado='reservado')
 THEN RAISE EXCEPTION 'reclaveado: contexto o envío reservado pendiente' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_conjunto WHERE persona_ref=p_persona)
 THEN RAISE EXCEPTION 'reclaveado: persona inexistente' USING ERRCODE='22023'; END IF;
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio'] LOOP
  EXECUTE format('SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY to_jsonb(r)::text),''[]''::jsonb) FROM vec_usuarios_correos_externo.%I r WHERE persona_ref=$1',t)
   INTO a USING p_persona;
  filas:=filas||jsonb_build_object(t,a);
 END LOOP;
 RETURN jsonb_build_object('persona_ref',p_persona,'filas',filas,
  'preimagen_sha256',encode(sha256(convert_to(filas::text,'UTF8')),'hex'));
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1(text) FROM PUBLIC;

CREATE FUNCTION vec_usuarios_correos_reclaveado.aplicar_reclaveado_persona_v1(
 p_persona text,p_preimagen text,p_material jsonb,p_lote text,p_aprobacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET timezone='UTC'
 SET datestyle='ISO, YMD' SET bytea_output='hex' SET lock_timeout='5s' AS $f$
DECLARE snap jsonb; t text; b jsonb; a jsonb; nuevo jsonb; pk jsonb; matsha text;
 recibo vec_usuarios_correos_reclaveado.recibo; cambios integer; total integer;
BEGIN
 snap:=vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1(p_persona);
 IF p_preimagen IS NULL OR p_preimagen !~ '^[0-9a-f]{64}$'
 OR p_lote IS NULL OR p_lote !~ '^[A-Za-z0-9:._-]{16,128}$'
 OR p_aprobacion IS NULL OR p_aprobacion !~ '^[A-Za-z0-9:._-]{16,128}$'
 OR jsonb_typeof(p_material) IS DISTINCT FROM 'object' OR octet_length(p_material::text)>16777216
 OR p_material->>'clave_sobre_ref' IS DISTINCT FROM 'clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1'
 OR p_material->>'clave_igualdad_ref' IS DISTINCT FROM 'clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1'
 OR p_material->>'clave_codigo_ref' IS DISTINCT FROM 'clave:kms:desarrollo:usuarios-correos-externo-codigo:v1'
 OR p_material->>'clave_semantica_ref' IS DISTINCT FROM 'clave:kms:desarrollo:usuarios-correos-externo-semantica:v1'
 OR jsonb_typeof(p_material->'direcciones') IS DISTINCT FROM 'array'
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p_material) k)
   IS DISTINCT FROM ARRAY['clave_codigo_ref','clave_igualdad_ref','clave_semantica_ref','clave_sobre_ref','direcciones']
 THEN RAISE EXCEPTION 'reclaveado: material inválido' USING ERRCODE='22023'; END IF;
 matsha:=encode(sha256(convert_to(p_material::text,'UTF8')),'hex');
 SELECT * INTO recibo FROM vec_usuarios_correos_reclaveado.recibo WHERE persona_ref=p_persona AND lote_ref=p_lote;
 IF FOUND THEN
  IF recibo.preimagen_sha256 IS DISTINCT FROM p_preimagen OR recibo.material_sha256 IS DISTINCT FROM matsha
   OR recibo.aprobacion_ref IS DISTINCT FROM p_aprobacion
  THEN RAISE EXCEPTION 'reclaveado: lote reutilizado' USING ERRCODE='23505'; END IF;
  RETURN vec_usuarios_correos_reclaveado.recuperar_reclaveado_persona_v1(p_persona,p_preimagen,p_lote,p_aprobacion);
 END IF;
 IF snap->>'preimagen_sha256' IS DISTINCT FROM p_preimagen
 THEN RAISE EXCEPTION 'reclaveado: preimagen cambió' USING ERRCODE='40001'; END IF;
 IF EXISTS(SELECT 1 FROM vec_usuarios_correos_reclaveado.recibo WHERE persona_ref=p_persona)
 THEN RAISE EXCEPTION 'reclaveado: persona ya convertida' USING ERRCODE='55000'; END IF;
 total:=jsonb_array_length(snap#>'{filas,correos_direccion}');
 IF jsonb_array_length(p_material->'direcciones')<>total OR total=0
 OR (SELECT count(DISTINCT d->>'correo_ref') FROM jsonb_array_elements(p_material->'direcciones') d)<>total
 THEN RAISE EXCEPTION 'reclaveado: direcciones incompletas' USING ERRCODE='22023'; END IF;
 FOR nuevo IN SELECT * FROM jsonb_array_elements(p_material->'direcciones') LOOP
  IF jsonb_typeof(nuevo) IS DISTINCT FROM 'object' OR nuevo->>'correo_ref' IS NULL
   OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(nuevo) k)
      IS DISTINCT FROM ARRAY['cifrado_hex','correo_ref','huella_igualdad_hex','nonce_hex','version_sobre']
   OR nuevo->>'nonce_hex' IS NULL OR nuevo->>'nonce_hex' !~ '^[0-9a-f]{24}$'
   OR nuevo->>'cifrado_hex' IS NULL OR nuevo->>'cifrado_hex' !~ '^[0-9a-f]+$'
   OR length(nuevo->>'cifrado_hex') NOT BETWEEN 38 AND 540
   OR length(nuevo->>'cifrado_hex')%2<>0
   OR nuevo->>'huella_igualdad_hex' IS NULL OR nuevo->>'huella_igualdad_hex' !~ '^[0-9a-f]{64}$'
   OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(snap#>'{filas,correos_direccion}') d
      WHERE d->>'correo_ref'=nuevo->>'correo_ref' AND d->'version_sobre'=nuevo->'version_sobre')
  THEN RAISE EXCEPTION 'reclaveado: sobre inválido' USING ERRCODE='22023'; END IF;
 END LOOP;
 INSERT INTO vec_usuarios_correos_reclaveado.recibo VALUES(p_persona,p_lote,p_aprobacion,p_preimagen,matsha,
  'reclaveado:'||replace(gen_random_uuid()::text,'-',''),session_user,pg_current_xact_id(),clock_timestamp()) RETURNING * INTO recibo;
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio'] LOOP
  FOR b IN SELECT * FROM jsonb_array_elements(snap->'filas'->t) LOOP
   a:=b;
   IF t='correos_conjunto' THEN a:=b||jsonb_build_object('clave_igualdad_ref',p_material->>'clave_igualdad_ref');
   ELSIF t='correos_direccion' THEN
    SELECT d INTO STRICT nuevo FROM jsonb_array_elements(p_material->'direcciones') d WHERE d->>'correo_ref'=b->>'correo_ref';
    a:=b||jsonb_build_object('clave_sobre_ref',p_material->>'clave_sobre_ref','clave_igualdad_ref',p_material->>'clave_igualdad_ref',
     'nonce','\x'||(nuevo->>'nonce_hex'),'cifrado','\x'||(nuevo->>'cifrado_hex'),'huella_igualdad','\x'||(nuevo->>'huella_igualdad_hex'));
   ELSIF t='correos_desafio' AND b->>'clave_ref'<>p_material->>'clave_codigo_ref' AND b->>'estado'='pendiente'
   THEN a:=b||jsonb_build_object('estado','sustituido'); END IF;
   pk:=vec_usuarios_correos_reclaveado.pk_fila_v1(t,b);
   INSERT INTO vec_usuarios_correos_reclaveado.fila VALUES(p_persona,p_lote,t,pk,
    encode(sha256(convert_to(b::text,'UTF8')),'hex'),encode(sha256(convert_to(a::text,'UTF8')),'hex'),pg_current_xact_id());
  END LOOP;
 END LOOP;
 UPDATE vec_usuarios_correos_externo.correos_conjunto SET clave_igualdad_ref=p_material->>'clave_igualdad_ref'
 WHERE persona_ref=p_persona;
 UPDATE vec_usuarios_correos_externo.correos_direccion d SET clave_sobre_ref=p_material->>'clave_sobre_ref',
  clave_igualdad_ref=p_material->>'clave_igualdad_ref',nonce=decode(n->>'nonce_hex','hex'),
  cifrado=decode(n->>'cifrado_hex','hex'),huella_igualdad=decode(n->>'huella_igualdad_hex','hex')
 FROM jsonb_array_elements(p_material->'direcciones') n WHERE d.persona_ref=p_persona AND d.correo_ref=n->>'correo_ref';
 GET DIAGNOSTICS cambios=ROW_COUNT;
 IF cambios<>total THEN RAISE EXCEPTION 'reclaveado: número de direcciones divergente' USING ERRCODE='40001'; END IF;
 UPDATE vec_usuarios_correos_externo.correos_desafio SET estado='sustituido'
 WHERE persona_ref=p_persona AND clave_ref<>p_material->>'clave_codigo_ref' AND estado='pendiente';
 -- Verificación exacta postimagen de TODAS las filas, incluida historia.
 snap:=vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1(p_persona);
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio'] LOOP
  FOR a IN SELECT * FROM jsonb_array_elements(snap->'filas'->t) LOOP
   IF NOT vec_usuarios_correos_reclaveado.fila_sellada_v1(t,a)
   THEN RAISE EXCEPTION 'reclaveado: postimagen divergente' USING ERRCODE='40001'; END IF;
  END LOOP;
 END LOOP;
 RETURN to_jsonb(recibo)||jsonb_build_object('replay',false);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_reclaveado.aplicar_reclaveado_persona_v1(text,text,jsonb,text,text) FROM PUBLIC;

CREATE FUNCTION vec_usuarios_correos_reclaveado.recuperar_reclaveado_persona_v1(
 p_persona text,p_preimagen text,p_lote text,p_aprobacion text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog SET timezone='UTC'
 SET datestyle='ISO, YMD' SET bytea_output='hex' SET lock_timeout='5s' AS $f$
DECLARE snap jsonb; r vec_usuarios_correos_reclaveado.recibo; t text; a jsonb; n integer:=0;
BEGIN
 snap:=vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1(p_persona);
 SELECT * INTO r FROM vec_usuarios_correos_reclaveado.recibo WHERE persona_ref=p_persona AND lote_ref=p_lote;
 IF NOT FOUND THEN RETURN NULL; END IF;
 IF r.preimagen_sha256 IS DISTINCT FROM p_preimagen OR r.aprobacion_ref IS DISTINCT FROM p_aprobacion
 THEN RAISE EXCEPTION 'reclaveado: lote reutilizado' USING ERRCODE='P1409'; END IF;
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio'] LOOP
  FOR a IN SELECT * FROM jsonb_array_elements(snap->'filas'->t) LOOP
   n:=n+1;
   IF NOT vec_usuarios_correos_reclaveado.fila_sellada_v1(t,a)
   THEN RAISE EXCEPTION 'reclaveado: población posterior cambió' USING ERRCODE='40001'; END IF;
  END LOOP;
 END LOOP;
 IF n<>(SELECT count(*) FROM vec_usuarios_correos_reclaveado.fila WHERE persona_ref=p_persona AND lote_ref=p_lote)
 THEN RAISE EXCEPTION 'reclaveado: población posterior incompleta' USING ERRCODE='40001'; END IF;
 RETURN to_jsonb(r)||jsonb_build_object('replay',true);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_reclaveado.recuperar_reclaveado_persona_v1(text,text,text,text) FROM PUBLIC;

-- Mismo contrato booleano de Usuarios012. Una referencia histórica antigua
-- se admite solo con su PK y hash EXACTOS; nunca sirve para repetir efecto.
-- Contextos y desafíos antiguos pendientes continúan bloqueando el arranque.
CREATE OR REPLACE FUNCTION vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on
 SET timezone='UTC' SET datestyle='ISO, YMD' SET bytea_output='hex' AS $f$
DECLARE t text; s record; a jsonb;
BEGIN
 IF session_user::text<>'vec_externo_preflight_v3_desarrollo' THEN RETURN false; END IF;
 IF EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_conjunto
      WHERE clave_igualdad_ref<>'clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1')
 OR EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_direccion
      WHERE clave_sobre_ref<>'clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1'
       OR clave_igualdad_ref<>'clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1')
 OR EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_contexto)
 OR EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_desafio d
      WHERE clave_ref<>'clave:kms:desarrollo:usuarios-correos-externo-codigo:v1'
       AND (estado='pendiente' OR NOT vec_usuarios_correos_reclaveado.fila_sellada_v1('correos_desafio',to_jsonb(d))))
 OR EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_intento_fallido i
      WHERE huella_clave_ref<>'clave:kms:desarrollo:usuarios-correos-externo-semantica:v1'
       AND NOT vec_usuarios_correos_reclaveado.fila_sellada_v1('correos_intento_fallido',to_jsonb(i)))
 OR EXISTS(SELECT 1 FROM vec_usuarios_correos_externo.correos_recibo r
      WHERE huella_clave_ref<>'clave:kms:desarrollo:usuarios-correos-externo-semantica:v1'
       AND NOT vec_usuarios_correos_reclaveado.fila_sellada_v1('correos_recibo',to_jsonb(r)))
 THEN RETURN false; END IF;
 -- También se detecta pérdida de una fila sellada, no solo claves huérfanas.
 FOR s IN SELECT * FROM vec_usuarios_correos_reclaveado.fila
  WHERE tabla IN ('correos_desafio','correos_intento_fallido','correos_historia','correos_recibo','correos_envio') LOOP
  EXECUTE format('SELECT to_jsonb(r) FROM vec_usuarios_correos_externo.%I r WHERE persona_ref=$1 AND vec_usuarios_correos_reclaveado.pk_fila_v1($2,to_jsonb(r))=$3',s.tabla)
   INTO a USING s.persona_ref,s.tabla,s.pk;
  IF a IS NULL THEN RETURN false; END IF;
  IF s.tabla='correos_desafio' AND a->>'clave_ref'='clave:kms:desarrollo:usuarios-correos-externo-codigo:v1' THEN CONTINUE; END IF;
  IF encode(sha256(convert_to(a::text,'UTF8')),'hex')<>s.postimagen_sha256
  THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1() FROM PUBLIC;

DO $acl$ DECLARE f record; t record;
BEGIN
 FOR f IN SELECT oid,proowner FROM pg_proc WHERE pronamespace='vec_usuarios_correos_reclaveado'::regnamespace LOOP
  IF has_function_privilege('public',f.oid,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_externo',f.oid,'EXECUTE')
    OR has_function_privilege('vec_usuarios_ejecutor_interno',f.oid,'EXECUTE')
  THEN RAISE EXCEPTION 'Usuarios013: función de mantenimiento expuesta' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOR t IN SELECT oid FROM pg_class WHERE relnamespace='vec_usuarios_correos_reclaveado'::regnamespace AND relkind='r' LOOP
  IF has_table_privilege('public',t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
   OR has_table_privilege('vec_usuarios_correos_externo_propietario',t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
   OR has_table_privilege('vec_usuarios_ejecutor_externo',t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
   OR has_table_privilege('vec_usuarios_ejecutor_interno',t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  THEN RAISE EXCEPTION 'Usuarios013: ledger expuesto' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF has_type_privilege('public','vec_usuarios_correos_reclaveado.recibo','USAGE')
 OR has_type_privilege('public','vec_usuarios_correos_reclaveado.fila','USAGE')
 OR has_schema_privilege('vec_usuarios_ejecutor_externo','vec_usuarios_correos_reclaveado','USAGE,CREATE')
 OR has_schema_privilege('vec_usuarios_ejecutor_interno','vec_usuarios_correos_reclaveado','USAGE,CREATE')
 THEN RAISE EXCEPTION 'Usuarios013: esquema de mantenimiento expuesto' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
