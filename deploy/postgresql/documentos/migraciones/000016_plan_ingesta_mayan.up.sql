\set ON_ERROR_STOP on
-- D16. Requisito Alberto 11/10: original de curso/título en gestor externo.
-- Solo plan e intento; ningún byte, URL, token ni recibo administrativo.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000016',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
  OR to_regclass('vec_documentos.reserva_recurso_documental') IS NOT NULL
  OR to_regclass('vec_documentos.plan_ingesta_mayan') IS NOT NULL
  OR to_regprocedure('vec_documentos.operar_reserva_recurso_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR to_regprocedure('vec_documentos.reclamar_reserva_recurso_carga_v1(text,text,text,text,bigint,text,text)') IS NOT NULL
  OR to_regprocedure('vec_documentos.operar_ingesta_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  OR to_regprocedure('vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regclass('vec_documentos.auditoria_operacion') IS NULL
  OR to_regclass('vec_documentos.outbox') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR strpos(pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'documentos.ingesta_mayan.preparar')=0
 THEN RAISE EXCEPTION 'D16: AD238 o preimagen Documentos ausente' USING ERRCODE='55000'; END IF;
END $pre$;

-- El recurso es una referencia documental reservada antes de la carga S3.
-- No declara un hecho RUM. Una referencia abandonada/caducada no se borra ni
-- se asigna a otra persona. Varios planes de evidencia pueden apuntar a ella.
CREATE TABLE vec_documentos.reserva_recurso_documental (
 recurso_ref text PRIMARY KEY CHECK(recurso_ref ~ '^ref:[0-9a-f]{64}$' AND recurso_ref<>('ref:'||repeat('0',64))),
 persona_ref text NOT NULL CHECK(vec_documentos.principal_ref_v1(persona_ref) AND persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 clave_idempotencia text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(clave_idempotencia)),
 huella_solicitud_sha256 text NOT NULL CHECK(huella_solicitud_sha256 ~ '^[0-9a-f]{64}$'),
 registro_contexto_ref text NOT NULL CHECK(registro_contexto_ref ~ '^rca_[A-Za-z0-9_-]{22,128}$'),
 huella_contexto_sha256 text NOT NULL CHECK(huella_contexto_sha256 ~ '^[0-9a-f]{64}$'),
 estado text NOT NULL CHECK(estado IN ('reservada','vinculada','abandonada','caducada')),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 3),
 decision_reserva_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 expira_en timestamptz(6) NOT NULL,
 actualizada_en timestamptz(6) NOT NULL,
 UNIQUE(persona_ref,clave_idempotencia),
 CHECK(expira_en>creada_en AND expira_en<=creada_en+interval '10 minutes'),
 CHECK((estado='reservada' AND version=1) OR (estado IN ('vinculada','abandonada','caducada') AND version>=2))
);
REVOKE ALL ON TABLE vec_documentos.reserva_recurso_documental FROM PUBLIC,vec_documentos_ejecutor;
REVOKE ALL ON TYPE vec_documentos.reserva_recurso_documental FROM PUBLIC;
ALTER TABLE vec_documentos.reserva_recurso_documental ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.reserva_recurso_documental FORCE ROW LEVEL SECURITY;
CREATE POLICY recurso_lectura ON vec_documentos.reserva_recurso_documental FOR SELECT TO vec_documentos_propietario
 USING (persona_ref=nullif(current_setting('vec.documentos.persona_ref',true),''));
CREATE POLICY recurso_alta ON vec_documentos.reserva_recurso_documental FOR INSERT TO vec_documentos_propietario
 WITH CHECK (persona_ref=nullif(current_setting('vec.documentos.persona_ref',true),''));
CREATE POLICY recurso_cas ON vec_documentos.reserva_recurso_documental FOR UPDATE TO vec_documentos_propietario
 USING (persona_ref=nullif(current_setting('vec.documentos.persona_ref',true),''))
 WITH CHECK (persona_ref=nullif(current_setting('vec.documentos.persona_ref',true),''));
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.reserva_recurso_documental
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();

CREATE TABLE vec_documentos.plan_ingesta_mayan (
 plan_ref text PRIMARY KEY CHECK(plan_ref ~ '^plan:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 carga_id text NOT NULL UNIQUE CHECK(vec_documentos.referencia_opaca_v1(carga_id)),
 persona_ref text NOT NULL CHECK(vec_documentos.principal_ref_v1(persona_ref) AND persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 expediente_ref text NOT NULL REFERENCES vec_documentos.reserva_recurso_documental(recurso_ref)
  CHECK(expediente_ref ~ '^ref:[0-9a-f]{64}$' AND expediente_ref<>('ref:'||repeat('0',64))),
 evidencia_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(evidencia_ref)),
 tipo_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(tipo_ref)),
 principal_ref text NOT NULL CHECK(vec_documentos.principal_ref_v1(principal_ref)),
 clave_reserva text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(clave_reserva)),
 clave_idempotencia text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(clave_idempotencia)),
 registro_contexto_ref text NOT NULL CHECK(registro_contexto_ref ~ '^rca_[A-Za-z0-9_-]{22,128}$'),
 huella_contexto_sha256 text NOT NULL CHECK(huella_contexto_sha256 ~ '^[0-9a-f]{64}$'),
 huella_solicitud_sha256 text NOT NULL CHECK(huella_solicitud_sha256 ~ '^[0-9a-f]{64}$'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 mime text NOT NULL CHECK(mime ~ '^[a-z0-9.+-]+/[a-z0-9.+-]+$'),
 tamano bigint NOT NULL CHECK(tamano BETWEEN 1 AND 67108864),
 carga_version bigint NOT NULL CHECK(carga_version=5),
 politica_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(politica_ref)),
 version_politica bigint NOT NULL CHECK(version_politica>0),
 huella_politica_sha256 text NOT NULL CHECK(huella_politica_sha256 ~ '^[0-9a-f]{64}$'),
 conservacion_hasta timestamptz(6) NOT NULL,
 estado_politica text NOT NULL CHECK(estado_politica IN ('aprobada','provisional')),
 proteccion text NOT NULL CHECK(proteccion IN ('conservacion','bloqueo')),
 version bigint NOT NULL CHECK(version BETWEEN 1 AND 4),
 estado text NOT NULL CHECK(estado IN ('preparado','despachado','confirmado','indeterminado','rechazado')),
 documento_id text CHECK(documento_id IS NULL OR documento_id ~ '^[1-9][0-9]{0,19}$'),
 archivo_id text CHECK(archivo_id IS NULL OR archivo_id ~ '^[1-9][0-9]{0,19}$'),
 version_id text CHECK(version_id IS NULL OR version_id ~ '^[1-9][0-9]{0,19}$'),
 decision_preparacion_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 actualizada_en timestamptz(6) NOT NULL,
 UNIQUE(principal_ref,clave_idempotencia),
 CHECK(estado_politica<>'provisional' OR proteccion='conservacion'),
 CHECK((estado IN ('preparado','despachado') AND documento_id IS NULL AND archivo_id IS NULL AND version_id IS NULL)
   OR (estado='confirmado' AND documento_id IS NOT NULL AND archivo_id IS NOT NULL AND version_id IS NOT NULL)
   OR (estado='indeterminado' AND (archivo_id IS NULL OR documento_id IS NOT NULL) AND (version_id IS NULL OR archivo_id IS NOT NULL))
   OR (estado='rechazado' AND documento_id IS NULL AND archivo_id IS NULL AND version_id IS NULL)),
 CHECK((estado='preparado' AND version=1) OR (estado='despachado' AND version=2)
   OR (estado='indeterminado' AND version=3) OR (estado IN ('confirmado','rechazado') AND version IN (3,4)))
);
REVOKE ALL ON TABLE vec_documentos.plan_ingesta_mayan FROM PUBLIC,vec_documentos_ejecutor;
REVOKE ALL ON TYPE vec_documentos.plan_ingesta_mayan FROM PUBLIC;
ALTER TABLE vec_documentos.plan_ingesta_mayan ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_documentos.plan_ingesta_mayan FORCE ROW LEVEL SECURITY;
CREATE POLICY plan_lectura ON vec_documentos.plan_ingesta_mayan FOR SELECT TO vec_documentos_propietario
 USING (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE POLICY plan_alta ON vec_documentos.plan_ingesta_mayan FOR INSERT TO vec_documentos_propietario
 WITH CHECK (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE POLICY plan_cas ON vec_documentos.plan_ingesta_mayan FOR UPDATE TO vec_documentos_propietario
 USING (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''))
 WITH CHECK (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.plan_ingesta_mayan
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();

-- Se amplían únicamente listas positivas existentes. Las acciones quedan
-- ligadas a V3 en AD238; la cuenta ejecutora no recibe DML de tablas.
DO $listas$
DECLARE tabla text; columna text; nombre text; previo text; cuerpo text; anadido text;
BEGIN
 FOR tabla,columna,nombre,anadido IN SELECT * FROM (VALUES
  ('auditoria_operacion','accion','auditoria_operacion_accion_check',
   'accion IN (''documentos.ingesta_mayan.reservar_recurso'',''documentos.ingesta_mayan.consultar_reserva'',''documentos.ingesta_mayan.abandonar_recurso'',''documentos.ingesta_mayan.preparar'',''documentos.ingesta_mayan.despachar'',''documentos.ingesta_mayan.resolver'',''documentos.ingesta_mayan.conciliar'',''documentos.ingesta_mayan.consultar'')'),
  ('auditoria_operacion','resultado','auditoria_operacion_resultado_check',
   'resultado IN (''reservada'',''vinculada'',''abandonada'',''caducada'',''despachado'',''confirmado'',''indeterminado'',''rechazado'')'),
  ('outbox','tipo','outbox_tipo_check','tipo=''ingesta_mayan_verificada''')
 ) AS x(t,c,n,a) LOOP
  SELECT pg_get_constraintdef(oid) INTO previo FROM pg_constraint
   WHERE conrelid=('vec_documentos.'||tabla)::regclass AND conname=nombre;
  IF previo IS NULL OR left(previo,7)<>'CHECK (' OR right(previo,1)<>')'
    OR strpos(previo,'ingesta_mayan')>0 THEN
   RAISE EXCEPTION 'D16: CHECK % incompatible',nombre USING ERRCODE='55000'; END IF;
  cuerpo:=substring(previo FROM 8 FOR length(previo)-8);
  EXECUTE format('ALTER TABLE vec_documentos.%I DROP CONSTRAINT %I',tabla,nombre);
  EXECUTE format('ALTER TABLE vec_documentos.%I ADD CONSTRAINT %I CHECK ((%s) OR (%s))',tabla,nombre,cuerpo,anadido);
 END LOOP;
END $listas$;

CREATE FUNCTION vec_documentos.operar_reserva_recurso_mayan_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; d jsonb; v record; r vec_documentos.reserva_recurso_documental%ROWTYPE;
 accion text; clave text; persona text; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR p_preimagen IS NULL OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D16: reserva denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D16: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['accion','dato']
  OR jsonb_typeof(m->'dato')<>'object' THEN RAISE EXCEPTION 'D16: reserva no canónica' USING ERRCODE='22023'; END IF;
 accion:=m->>'accion'; d:=m->'dato'; clave:=d->>'clave_idempotencia'; persona:=p_auth->>'principal_id';
 IF accion NOT IN ('documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva','documentos.ingesta_mayan.abandonar_recurso')
  OR NOT vec_documentos.referencia_opaca_v1(clave) OR persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR p_auth->>'accion' IS DISTINCT FROM accion OR p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM clave OR p_auth->>'ambito_ref' IS DISTINCT FROM clave
 THEN RAISE EXCEPTION 'D16: reserva no ligada' USING ERRCODE='42501'; END IF;
 IF accion='documentos.ingesta_mayan.reservar_recurso' THEN
  IF ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM
     ARRAY['clave_idempotencia','creada_en','estado','expira_en','huella_contexto_sha256',
       'huella_solicitud_sha256','persona_ref','recurso_ref','registro_contexto_ref','version']
   OR d->>'persona_ref' IS DISTINCT FROM persona
   OR d->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
   OR d->>'recurso_ref' IS NOT DISTINCT FROM ('ref:'||repeat('0',64))
   OR d->>'huella_solicitud_sha256' !~ '^[0-9a-f]{64}$'
   OR d->>'huella_contexto_sha256' !~ '^[0-9a-f]{64}$'
   OR d->>'registro_contexto_ref' !~ '^rca_[A-Za-z0-9_-]{22,128}$'
   OR d->>'estado' IS DISTINCT FROM ''
   OR d->>'version' IS DISTINCT FROM '0'
   OR d->>'creada_en' IS DISTINCT FROM '0001-01-01T00:00:00Z'
   OR d->>'expira_en' IS DISTINCT FROM '0001-01-01T00:00:00Z'
  THEN RAISE EXCEPTION 'D16: candidata inválida' USING ERRCODE='22023'; END IF;
 ELSE
  IF ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM ARRAY['clave_idempotencia']
  THEN RAISE EXCEPTION 'D16: consulta de reserva inválida' USING ERRCODE='22023'; END IF;
 END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v2(
  p_preimagen,accion,clave,persona,p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.persona_ref',persona,true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:reserva_recurso:'||persona||':'||clave,0));
 SELECT * INTO r FROM vec_documentos.reserva_recurso_documental
  WHERE persona_ref=persona AND clave_idempotencia=clave FOR UPDATE;
 IF NOT FOUND THEN
  IF accion<>'documentos.ingesta_mayan.reservar_recurso' OR v.consumo_nuevo IS FALSE
  THEN RAISE EXCEPTION 'D16: reserva ausente' USING ERRCODE='02000'; END IF;
  INSERT INTO vec_documentos.reserva_recurso_documental(
   recurso_ref,persona_ref,clave_idempotencia,huella_solicitud_sha256,registro_contexto_ref,
   huella_contexto_sha256,estado,version,decision_reserva_ref,creada_en,expira_en,actualizada_en)
  VALUES(d->>'recurso_ref',persona,clave,d->>'huella_solicitud_sha256',d->>'registro_contexto_ref',
   d->>'huella_contexto_sha256','reservada',1,v.decision_ref,ahora,ahora+interval '10 minutes',ahora)
  RETURNING * INTO r;
 ELSIF accion='documentos.ingesta_mayan.reservar_recurso' AND
    r.huella_solicitud_sha256 IS DISTINCT FROM d->>'huella_solicitud_sha256' THEN
  RAISE EXCEPTION 'D16: clave reutilizada con otra solicitud' USING ERRCODE='23505';
 END IF;
 IF r.estado='reservada' AND ahora>=r.expira_en THEN
  UPDATE vec_documentos.reserva_recurso_documental SET estado='caducada',version=version+1,actualizada_en=ahora
   WHERE recurso_ref=r.recurso_ref AND version=r.version RETURNING * INTO r;
 END IF;
 IF accion='documentos.ingesta_mayan.abandonar_recurso' AND r.estado='reservada' THEN
  PERFORM set_config('vec.documentos.expediente_ref',r.recurso_ref,true);
  IF EXISTS(SELECT 1 FROM vec_documentos.plan_ingesta_mayan p WHERE p.expediente_ref=r.recurso_ref)
  THEN RAISE EXCEPTION 'D16: recurso utilizado' USING ERRCODE='23505'; END IF;
  UPDATE vec_documentos.reserva_recurso_documental SET estado='abandonada',version=version+1,actualizada_en=ahora
   WHERE recurso_ref=r.recurso_ref AND version=r.version RETURNING * INTO r;
 END IF;
 PERFORM set_config('vec.documentos.expediente_ref',r.recurso_ref,true);
 INSERT INTO vec_documentos.auditoria_operacion(
  accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,
  decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(accion,r.recurso_ref,r.recurso_ref,persona,p_auth->>'perfil_activo_ref',p_auth->>'finalidad',
  p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,r.estado,ahora);
 RETURN jsonb_build_object('reserva',to_jsonb(r)-'decision_reserva_ref');
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.operar_reserva_recurso_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.operar_reserva_recurso_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

-- Solo una función owner del repositorio de cargas puede invocar esta guarda
-- dentro de su TX V3. El ejecutor no recibe EXECUTE directo. La primera carga
-- convierte la reserva en vinculada; las siguientes evidencias de la misma
-- persona pueden reclamarla con su versión actual y otra clave de carga.
CREATE FUNCTION vec_documentos.reclamar_reserva_recurso_carga_v1(
 p_recurso_ref text,p_persona_ref text,p_clave_reserva text,p_huella_reserva_sha256 text,
 p_version_reserva bigint,p_indice_carga_hmac text,p_huella_carga_hmac text) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE r vec_documentos.reserva_recurso_documental%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
  OR p_recurso_ref !~ '^ref:[0-9a-f]{64}$' OR p_recurso_ref=('ref:'||repeat('0',64))
  OR p_persona_ref !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR NOT vec_documentos.referencia_opaca_v1(p_clave_reserva)
  OR p_huella_reserva_sha256 !~ '^[0-9a-f]{64}$' OR p_version_reserva<1
  OR p_indice_carga_hmac !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR p_huella_carga_hmac !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'D16: reclamación denegada' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.documentos.persona_ref',p_persona_ref,true);
 SELECT * INTO r FROM vec_documentos.reserva_recurso_documental
  WHERE recurso_ref=p_recurso_ref AND persona_ref=p_persona_ref
   AND clave_idempotencia=p_clave_reserva FOR UPDATE;
 IF NOT FOUND OR r.huella_solicitud_sha256 IS DISTINCT FROM p_huella_reserva_sha256
  OR r.version IS DISTINCT FROM p_version_reserva OR r.estado NOT IN ('reservada','vinculada')
  OR (r.estado='reservada' AND ahora>=r.expira_en)
 THEN RAISE EXCEPTION 'D16: reserva no vigente o divergente' USING ERRCODE='42501'; END IF;
 IF r.estado='reservada' THEN
  UPDATE vec_documentos.reserva_recurso_documental SET estado='vinculada',version=version+1,actualizada_en=ahora
   WHERE recurso_ref=r.recurso_ref AND version=r.version RETURNING * INTO r;
 END IF;
 RETURN jsonb_build_object('recurso_ref',r.recurso_ref,'persona_ref',r.persona_ref,
  'clave_reserva',r.clave_idempotencia,'huella_reserva_sha256',r.huella_solicitud_sha256,
  'version_reserva',r.version,'estado',r.estado,
  'indice_carga_hmac',p_indice_carga_hmac,'huella_carga_hmac',p_huella_carga_hmac);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.reclamar_reserva_recurso_carga_v1(text,text,text,text,bigint,text,text)
 FROM PUBLIC,vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.operar_ingesta_mayan_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; d jsonb; v record; r vec_documentos.plan_ingesta_mayan%ROWTYPE;
 recurso vec_documentos.reserva_recurso_documental%ROWTYPE;
 accion text; ref text; ambito text; nuevo boolean:=false; plan_existe boolean:=false;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR p_preimagen IS NULL OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D16: operación denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D16: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['accion','dato']
  OR jsonb_typeof(m->'dato')<>'object' THEN RAISE EXCEPTION 'D16: preimagen no canónica' USING ERRCODE='22023'; END IF;
 d:=m->'dato'; accion:=m->>'accion'; ambito:=d->>'expediente_ref';
 IF accion='documentos.ingesta_mayan.preparar' THEN ref:=d->>'carga_id';
 ELSE ref:=d->>'plan_ref'; END IF;
 IF accion NOT IN ('documentos.ingesta_mayan.preparar','documentos.ingesta_mayan.despachar',
   'documentos.ingesta_mayan.resolver','documentos.ingesta_mayan.conciliar','documentos.ingesta_mayan.consultar')
  OR NOT vec_documentos.referencia_opaca_v1(ref)
  OR ambito !~ '^ref:[0-9a-f]{64}$' OR ambito=('ref:'||repeat('0',64))
  OR p_auth->>'accion' IS DISTINCT FROM accion OR p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM ref OR p_auth->>'ambito_ref' IS DISTINCT FROM ambito
  OR NOT vec_documentos.principal_ref_v1(p_auth->>'principal_id')
 THEN RAISE EXCEPTION 'D16: acción no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v2(
  p_preimagen,accion,ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',ambito,true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:ingesta_mayan:'||ref,0));
 IF accion='documentos.ingesta_mayan.preparar' THEN
  SELECT * INTO r FROM vec_documentos.plan_ingesta_mayan WHERE carga_id=ref FOR UPDATE;
 ELSE
  SELECT * INTO r FROM vec_documentos.plan_ingesta_mayan WHERE plan_ref=ref FOR UPDATE;
 END IF;
 plan_existe:=FOUND;
 IF accion='documentos.ingesta_mayan.preparar' THEN
  IF ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM
   ARRAY['actualizado_en','archivo_id','carga_id','carga_version','clave_idempotencia','clave_reserva','conservacion_hasta',
    'creado_en','documento_id','estado','estado_politica','evidencia_ref','expediente_ref','huella_contexto_sha256',
    'huella_politica_sha256','huella_sha256','huella_solicitud_sha256','mime','persona_ref','plan_ref','politica_ref','proteccion',
    'registro_contexto_ref','tamano','tipo_ref','version','version_id','version_politica']
   OR d->>'estado' IS DISTINCT FROM '' OR d->>'version' IS DISTINCT FROM '0'
   OR d->>'plan_ref' IS DISTINCT FROM ''
   OR d->>'documento_id' IS DISTINCT FROM '' OR d->>'archivo_id' IS DISTINCT FROM '' OR d->>'version_id' IS DISTINCT FROM ''
   OR d->>'creado_en' IS DISTINCT FROM '0001-01-01T00:00:00Z'
   OR d->>'actualizado_en' IS DISTINCT FROM '0001-01-01T00:00:00Z'
  THEN RAISE EXCEPTION 'D16: plan inicial inválido' USING ERRCODE='22023'; END IF;
  IF plan_existe THEN
   IF r.principal_ref IS DISTINCT FROM p_auth->>'principal_id' OR r.huella_solicitud_sha256 IS DISTINCT FROM d->>'huella_solicitud_sha256'
    OR r.carga_id IS DISTINCT FROM d->>'carga_id' OR r.expediente_ref IS DISTINCT FROM ambito
    OR r.clave_reserva IS DISTINCT FROM d->>'clave_reserva' OR r.clave_idempotencia IS DISTINCT FROM d->>'clave_idempotencia'
    OR r.persona_ref IS DISTINCT FROM d->>'persona_ref' OR r.evidencia_ref IS DISTINCT FROM d->>'evidencia_ref'
    OR r.tipo_ref IS DISTINCT FROM d->>'tipo_ref' OR r.huella_sha256 IS DISTINCT FROM d->>'huella_sha256'
    OR r.mime IS DISTINCT FROM d->>'mime' OR r.tamano::text IS DISTINCT FROM d->>'tamano'
    OR r.politica_ref IS DISTINCT FROM d->>'politica_ref' OR r.version_politica::text IS DISTINCT FROM d->>'version_politica'
    OR r.huella_politica_sha256 IS DISTINCT FROM d->>'huella_politica_sha256'
    OR r.conservacion_hasta IS DISTINCT FROM (d->>'conservacion_hasta')::timestamptz
    OR r.estado_politica IS DISTINCT FROM d->>'estado_politica' OR r.proteccion IS DISTINCT FROM d->>'proteccion'
   THEN RAISE EXCEPTION 'D16: plan divergente' USING ERRCODE='23505'; END IF;
  ELSE
   IF v.consumo_nuevo IS FALSE THEN RAISE EXCEPTION 'D16: replay sin plan' USING ERRCODE='42501'; END IF;
   PERFORM set_config('vec.documentos.persona_ref',p_auth->>'principal_id',true);
   SELECT * INTO recurso FROM vec_documentos.reserva_recurso_documental
    WHERE recurso_ref=ambito AND persona_ref=p_auth->>'principal_id' AND clave_idempotencia=d->>'clave_reserva' FOR UPDATE;
   IF NOT FOUND OR recurso.estado<>'vinculada'
      OR recurso.persona_ref IS DISTINCT FROM d->>'persona_ref'
   THEN RAISE EXCEPTION 'D16: recurso previo no vigente' USING ERRCODE='42501'; END IF;
   INSERT INTO vec_documentos.plan_ingesta_mayan(
    plan_ref,carga_id,persona_ref,expediente_ref,evidencia_ref,tipo_ref,principal_ref,clave_reserva,clave_idempotencia,
    registro_contexto_ref,huella_contexto_sha256,
    huella_solicitud_sha256,huella_sha256,mime,tamano,carga_version,politica_ref,version_politica,
    huella_politica_sha256,conservacion_hasta,estado_politica,proteccion,version,estado,decision_preparacion_ref,creada_en,actualizada_en)
   VALUES('plan:'||gen_random_uuid()::text,ref,d->>'persona_ref',ambito,d->>'evidencia_ref',d->>'tipo_ref',p_auth->>'principal_id',
    d->>'clave_reserva',d->>'clave_idempotencia',d->>'registro_contexto_ref',d->>'huella_contexto_sha256',
    d->>'huella_solicitud_sha256',d->>'huella_sha256',d->>'mime',
    (d->>'tamano')::bigint,(d->>'carga_version')::bigint,d->>'politica_ref',(d->>'version_politica')::bigint,
    d->>'huella_politica_sha256',(d->>'conservacion_hasta')::timestamptz,d->>'estado_politica',d->>'proteccion',
    1,'preparado',v.decision_ref,ahora,ahora) RETURNING * INTO r;
  END IF;
 ELSIF NOT plan_existe THEN
  RAISE EXCEPTION 'D16: plan inexistente' USING ERRCODE='02000';
 ELSIF r.expediente_ref IS DISTINCT FROM ambito OR r.principal_ref IS DISTINCT FROM p_auth->>'principal_id' THEN
  RAISE EXCEPTION 'D16: ámbito ajeno' USING ERRCODE='42501';
 ELSIF accion='documentos.ingesta_mayan.consultar' THEN
  IF ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM ARRAY['expediente_ref','plan_ref']
  THEN RAISE EXCEPTION 'D16: consulta inválida' USING ERRCODE='22023'; END IF;
 ELSE
  IF ARRAY(SELECT jsonb_object_keys(d) ORDER BY 1) IS DISTINCT FROM
   ARRAY['documento','estado','expediente_ref','plan_ref','version_esperada']
   OR jsonb_typeof(d->'documento')<>'object' OR (d->>'version_esperada')::bigint<1
  THEN RAISE EXCEPTION 'D16: transición inválida' USING ERRCODE='22023'; END IF;
  IF accion='documentos.ingesta_mayan.despachar' THEN
   IF d->>'estado'<>'despachado' OR d->'documento' IS DISTINCT FROM
     '{"documento_id":"","archivo_id":"","version_id":"","huella_sha256":"","mime":"","tamano":0}'::jsonb
   THEN RAISE EXCEPTION 'D16: despacho inválido' USING ERRCODE='22023'; END IF;
   IF r.estado='preparado' AND r.version=(d->>'version_esperada')::bigint AND v.consumo_nuevo IS TRUE THEN
    UPDATE vec_documentos.plan_ingesta_mayan SET estado='despachado',version=2,actualizada_en=ahora
     WHERE plan_ref=ref AND version=1 RETURNING * INTO r;
    nuevo:=true;
   ELSIF r.estado='preparado' THEN
    RAISE EXCEPTION 'D16: despacho en conflicto' USING ERRCODE='40001';
   END IF;
  ELSIF accion='documentos.ingesta_mayan.resolver' OR accion='documentos.ingesta_mayan.conciliar' THEN
   IF r.version IS DISTINCT FROM (d->>'version_esperada')::bigint
   THEN RAISE EXCEPTION 'D16: CAS divergente' USING ERRCODE='40001'; END IF;
   IF (accion='documentos.ingesta_mayan.resolver' AND r.estado<>'despachado')
      OR (accion='documentos.ingesta_mayan.conciliar' AND r.estado<>'indeterminado')
      OR d->>'estado' NOT IN ('confirmado','indeterminado','rechazado')
      OR (accion='documentos.ingesta_mayan.resolver' AND d->>'estado'='rechazado')
      OR (accion='documentos.ingesta_mayan.conciliar' AND d->>'estado'='indeterminado')
      OR (d->>'estado'='rechazado' AND r.documento_id IS NOT NULL)
      OR v.consumo_nuevo IS FALSE
   THEN RAISE EXCEPTION 'D16: resolución no admitida' USING ERRCODE='42501'; END IF;
   IF d->>'estado'='confirmado' AND (
      d#>>'{documento,huella_sha256}' IS DISTINCT FROM r.huella_sha256
      OR (d#>>'{documento,tamano}')::bigint IS DISTINCT FROM r.tamano
      OR d#>>'{documento,mime}' IS DISTINCT FROM r.mime
      OR d#>>'{documento,documento_id}' !~ '^[1-9][0-9]{0,19}$'
      OR d#>>'{documento,archivo_id}' !~ '^[1-9][0-9]{0,19}$'
      OR d#>>'{documento,version_id}' !~ '^[1-9][0-9]{0,19}$'
      OR (r.documento_id IS NOT NULL AND r.documento_id IS DISTINCT FROM d#>>'{documento,documento_id}')
      OR (r.archivo_id IS NOT NULL AND r.archivo_id IS DISTINCT FROM d#>>'{documento,archivo_id}')
      OR (r.version_id IS NOT NULL AND r.version_id IS DISTINCT FROM d#>>'{documento,version_id}'))
   THEN RAISE EXCEPTION 'D16: evidencia de custodia divergente' USING ERRCODE='42501'; END IF;
   UPDATE vec_documentos.plan_ingesta_mayan SET estado=d->>'estado',version=r.version+1,actualizada_en=ahora,
    documento_id=nullif(d#>>'{documento,documento_id}',''),archivo_id=nullif(d#>>'{documento,archivo_id}',''),
    version_id=nullif(d#>>'{documento,version_id}','')
    WHERE plan_ref=ref AND version=r.version RETURNING * INTO r;
   IF r.estado='confirmado' THEN
    INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
      VALUES('ingesta_mayan_verificada',ref,ambito,r.huella_sha256,ahora);
   END IF;
  END IF;
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(
  accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,
  decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(accion,ref,ambito,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',
  p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,
  CASE WHEN accion='documentos.ingesta_mayan.consultar' THEN 'autorizado'
       WHEN accion='documentos.ingesta_mayan.preparar' THEN 'preparado'
       WHEN nuevo THEN 'despachado' ELSE r.estado END,ahora);
 RETURN jsonb_build_object('plan',
  (to_jsonb(r)-'principal_ref'-'decision_preparacion_ref'-'creada_en'-'actualizada_en')
    ||jsonb_build_object('creado_en',r.creada_en,'actualizado_en',r.actualizada_en),
  'nuevo_despacho',nuevo);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.operar_ingesta_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.operar_ingesta_mayan_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;
COMMIT;
