\set ON_ERROR_STOP on
-- D17. Metadatos e historia operativa de carga directa; nunca bytes del fichero.
-- Depende causalmente de D16 y AD240. No instalar sobre historia ajena.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000017',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
  OR to_regprocedure('vec_documentos.reclamar_reserva_recurso_carga_v1(text,text,text,text,bigint,text,text)') IS NULL
  OR to_regprocedure('vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
    'vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex')
    IS DISTINCT FROM '8ea64568627e3f8181e41b83ad51bd4e24eb3c8454700313d8aba1f6e9b861b6'
  OR to_regclass('vec_documentos.carga_directa_operacion') IS NOT NULL
  OR to_regclass('vec_documentos.carga_directa_historia') IS NOT NULL
  OR to_regclass('vec_documentos.recibo_carga_directa') IS NOT NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
    'vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex')
    IS DISTINCT FROM '739e454676a6b992d2ce48b9cc44fd9a24d8514559447fe0141868cf80e29d3b'
  OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
    'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex')
    IS DISTINCT FROM '71432cb8ece498d006aeb2ca8b96424a07981b22ab14b59cf18d0e1e62a6ae80'
 THEN RAISE EXCEPTION 'D17: dependencia D16/AD240 ausente' USING ERRCODE='55000'; END IF;
END $pre$;

-- Fachada nueva y cerrada: D2 conserva exclusivamente la audiencia histórica
-- vec_documentos.operacion.v1. Ninguna acción D17 pasa por esa ABI instalada.
CREATE FUNCTION vec_documentos.consumir_carga_directa_v3_v1(
 p_preimagen bytea,p_accion text,p_recurso text,p_principal text,p_perfil text,p_finalidad text,p_correlacion text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,
 auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record; h text; audiencia text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
  OR NOT pg_has_role(session_user,'vec_documentos_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_documentos_propietario','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable'
  OR octet_length(p_preimagen) NOT BETWEEN 1 AND 16384
  OR octet_length(p_capacidad) NOT BETWEEN 1 AND 65536
  OR octet_length(p_decision) NOT BETWEEN 1 AND 65536
 THEN RAISE EXCEPTION 'D17: contexto V3 denegado' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: V3 inválida' USING ERRCODE='42501'; END;
 audiencia:=CASE p_accion
  WHEN 'documentos.carga.preparar_reserva' THEN 'vec_documentos.carga.preparar_reserva.v1'
  WHEN 'documentos.carga.preparar_confirmacion' THEN 'vec_documentos.carga.preparar_confirmacion.v1'
  WHEN 'documentos.carga.preparar_respuesta' THEN 'vec_documentos.carga.preparar_respuesta.v1'
  WHEN 'documentos.carga.consultar_propia' THEN 'vec_documentos.carga.consultar_propia.v1'
  WHEN 'documentos.carga.conciliar' THEN 'vec_documentos.carga.conciliar.v1'
  WHEN 'documentos.carga.confirmar_intencion' THEN 'vec_documentos.carga.confirmar_intencion.v1'
  WHEN 'documentos.carga.confirmar_resultado' THEN 'vec_documentos.carga.confirmar_resultado.v1'
  WHEN 'vec.documentos.carga.analizar' THEN 'vec_documentos.carga.analizar.v1'
  WHEN 'documentos.carga.promover_intencion' THEN 'vec_documentos.carga.promover_intencion.v1'
  WHEN 'documentos.carga.promover_resultado' THEN 'vec_documentos.carga.promover_resultado.v1' END;
 h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"preimagen_sha256":"'||
  encode(sha256(p_preimagen),'hex')||'"}}','UTF8')),'hex');
 IF audiencia IS NULL OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
  OR c->>'operacion' IS DISTINCT FROM p_accion
  OR c->>'efecto_ref' IS DISTINCT FROM p_recurso
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
  OR d->>'accion' IS DISTINCT FROM p_accion
  OR d->>'modulo_id' IS DISTINCT FROM 'documentos'
  OR d->>'recurso_ref' IS DISTINCT FROM p_recurso
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
  OR d->>'principal_id' IS DISTINCT FROM p_principal
  OR d->>'perfil_activo_ref' IS DISTINCT FROM p_perfil
  OR d->>'finalidad' IS DISTINCT FROM p_finalidad
  OR d->>'correlacion_ref' IS DISTINCT FROM p_correlacion
  OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'D17: decisión V3 no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.efecto_ref IS DISTINCT FROM p_recurso OR v.huella_efecto_sha256 IS DISTINCT FROM h
  OR v.decision_ref IS DISTINCT FROM c->>'decision_ref' OR v.consumo_nuevo IS NULL
 THEN RAISE EXCEPTION 'D17: consumo V3 no ligado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,
  v.auditoria_ref,v.consumida_en,v.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.consumir_carga_directa_v3_v1(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_documentos_ejecutor;

CREATE TABLE vec_documentos.carga_directa_operacion (
 carga_id text PRIMARY KEY CHECK(vec_documentos.referencia_opaca_v1(carga_id)),
 recurso_ref text NOT NULL CHECK(recurso_ref ~ '^ref:[0-9a-f]{64}$' AND recurso_ref<>('ref:'||repeat('0',64))),
 expediente_ref text NOT NULL CHECK(expediente_ref=recurso_ref),
 persona_ref text NOT NULL CHECK(persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
 indice_carga_hmac text NOT NULL UNIQUE CHECK(indice_carga_hmac ~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'),
 huella_carga_hmac text NOT NULL CHECK(huella_carga_hmac ~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'),
 huella_token_sha256 text NOT NULL CHECK(huella_token_sha256 ~ '^[0-9a-f]{64}$'),
 huella_preimagen_reserva_sha256 text NOT NULL CHECK(huella_preimagen_reserva_sha256 ~ '^[0-9a-f]{64}$'),
 huella_carga_json_sha256 text NOT NULL CHECK(huella_carga_json_sha256 ~ '^[0-9a-f]{64}$'),
 version_contenido integer NOT NULL CHECK(version_contenido>0),
 version_operativa bigint NOT NULL CHECK(version_operativa>0),
 estado_operativo text NOT NULL CHECK(estado_operativo IN
  ('preparacion_iniciada','emision_recibo_iniciada','recibo_registrado',
   'confirmacion_iniciada','confirmacion_indeterminada','cuarentena',
   'analizada_limpia','retenida_seguridad','promocion_iniciada',
   'promocion_indeterminada','admitida',
   'preparacion_indeterminada','recibo_indeterminado','no_recuperable')),
 carga jsonb NOT NULL CHECK(jsonb_typeof(carga)='object'),
 manifiesto jsonb,
 huella_manifiesto_sha256 text CHECK(huella_manifiesto_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_indice_hmac text UNIQUE CHECK(recibo_indice_hmac ~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'),
 indice_promocion_hmac text CHECK(indice_promocion_hmac ~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'),
 entrega_acreditada boolean NOT NULL DEFAULT false CHECK(entrega_acreditada=false),
 decision_reserva_ref text NOT NULL,
 decision_confirmacion_ref text,
 decision_respuesta_ref text,
 decision_promocion_intencion_ref text,
 creada_en timestamptz(6) NOT NULL,
 actualizada_en timestamptz(6) NOT NULL,
 CHECK ((manifiesto IS NULL AND huella_manifiesto_sha256 IS NULL) OR
        (jsonb_typeof(manifiesto)='object' AND huella_manifiesto_sha256 IS NOT NULL)),
 CHECK (estado_operativo IN ('preparacion_iniciada','preparacion_indeterminada','no_recuperable')
        OR manifiesto IS NOT NULL)
);
CREATE INDEX carga_directa_recurso_idx ON vec_documentos.carga_directa_operacion(recurso_ref,creada_en,carga_id);
CREATE TABLE vec_documentos.carga_directa_historia (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 carga_id text NOT NULL REFERENCES vec_documentos.carga_directa_operacion(carga_id),
 expediente_ref text NOT NULL,
 persona_ref text NOT NULL,
 version_operativa bigint NOT NULL CHECK(version_operativa>0),
 estado_anterior text,
 estado_siguiente text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ad3_ref text NOT NULL,
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 evidencia_ref text,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(carga_id,version_operativa)
);
CREATE TABLE vec_documentos.recibo_carga_directa (
 indice_hmac text PRIMARY KEY CHECK(indice_hmac ~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'),
 carga_id text NOT NULL UNIQUE REFERENCES vec_documentos.carga_directa_operacion(carga_id),
 expediente_ref text NOT NULL,
 persona_ref text NOT NULL,
 grupo_hmac text NOT NULL CHECK(grupo_hmac ~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'),
 vinculo_hmac text NOT NULL CHECK(vinculo_hmac ~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'),
 evidencia_alta_ref text NOT NULL,
 autorizacion_emision_ref text NOT NULL,
 expira_en timestamptz(6) NOT NULL,
 registrado_en timestamptz(6) NOT NULL,
 estado text NOT NULL CHECK(estado IN ('activo','consumido')),
 version bigint NOT NULL CHECK(version>0),
 evidencia_consumo_ref text,
 intencion_confirmacion_ref text UNIQUE,
 huella_intencion_hmac text,
 valida_hasta timestamptz(6),
 consumido_en timestamptz(6),
 CHECK ((estado='activo' AND consumido_en IS NULL AND intencion_confirmacion_ref IS NULL)
        OR (estado='consumido' AND consumido_en IS NOT NULL AND intencion_confirmacion_ref IS NOT NULL
            AND huella_intencion_hmac IS NOT NULL AND valida_hasta IS NOT NULL))
);
CREATE UNIQUE INDEX recibo_carga_grupo_activo_uq ON vec_documentos.recibo_carga_directa(grupo_hmac)
 WHERE estado='activo';

DO $rls$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['carga_directa_operacion','carga_directa_historia','recibo_carga_directa'] LOOP
  EXECUTE format('ALTER TABLE vec_documentos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_documentos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY alcance_lectura ON vec_documentos.%I FOR SELECT TO vec_documentos_propietario USING (expediente_ref=nullif(current_setting(''vec.documentos.expediente_ref'',true),''''))',t);
  EXECUTE format('CREATE POLICY alcance_alta ON vec_documentos.%I FOR INSERT TO vec_documentos_propietario WITH CHECK (expediente_ref=nullif(current_setting(''vec.documentos.expediente_ref'',true),''''))',t);
 END LOOP;
END $rls$;
CREATE POLICY carga_cas ON vec_documentos.carga_directa_operacion FOR UPDATE TO vec_documentos_propietario
 USING (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''))
 WITH CHECK (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE POLICY recibo_cas ON vec_documentos.recibo_carga_directa FOR UPDATE TO vec_documentos_propietario
 USING (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''))
 WITH CHECK (expediente_ref=nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_documentos.carga_directa_historia
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE TRIGGER historia_no_truncar BEFORE TRUNCATE ON vec_documentos.carga_directa_historia
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE TRIGGER carga_no_truncar BEFORE TRUNCATE ON vec_documentos.carga_directa_operacion
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();
CREATE TRIGGER recibo_no_truncar BEFORE TRUNCATE ON vec_documentos.recibo_carga_directa
 FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1();

REVOKE ALL ON vec_documentos.carga_directa_operacion,vec_documentos.carga_directa_historia,
 vec_documentos.recibo_carga_directa FROM PUBLIC,vec_documentos_ejecutor;
REVOKE ALL ON SEQUENCE vec_documentos.carga_directa_historia_id_seq FROM PUBLIC,vec_documentos_ejecutor;

DO $listas$
DECLARE tabla text; nombre text; previo text; cuerpo text; anadido text;
BEGIN
 FOR tabla,nombre,anadido IN SELECT * FROM (VALUES
  ('auditoria_operacion','auditoria_operacion_accion_check',
   'accion IN (''documentos.carga.preparar_reserva'',''documentos.carga.preparar_confirmacion'',''documentos.carga.preparar_respuesta'',''documentos.carga.consultar_propia'',''documentos.carga.conciliar'',''documentos.carga.confirmar_intencion'',''documentos.carga.confirmar_resultado'',''vec.documentos.carga.analizar'',''documentos.carga.promover_intencion'',''documentos.carga.promover_resultado'')'),
  ('auditoria_operacion','auditoria_operacion_resultado_check',
   'resultado IN (''preparacion_iniciada'',''emision_recibo_iniciada'',''recibo_registrado'',''confirmacion_iniciada'',''confirmacion_indeterminada'',''no_recuperable'',''denegado'',''cuarentena'',''analizada_limpia'',''retenida_seguridad'',''promocion_iniciada'',''admitida'')'),
  ('outbox','outbox_tipo_check','tipo=''carga_directa_estado''')
 ) AS x(t,n,a) LOOP
  SELECT pg_get_constraintdef(oid) INTO previo FROM pg_constraint
   WHERE conrelid=('vec_documentos.'||tabla)::regclass AND conname=nombre;
  IF previo IS NULL OR left(previo,7)<>'CHECK (' OR right(previo,1)<>')'
    OR strpos(previo,'carga_directa')>0 THEN
   RAISE EXCEPTION 'D17: CHECK incompatible' USING ERRCODE='55000'; END IF;
  cuerpo:=substring(previo FROM 8 FOR length(previo)-8);
  EXECUTE format('ALTER TABLE vec_documentos.%I DROP CONSTRAINT %I',tabla,nombre);
  EXECUTE format('ALTER TABLE vec_documentos.%I ADD CONSTRAINT %I CHECK ((%s) OR (%s))',tabla,nombre,cuerpo,anadido);
 END LOOP;
END $listas$;

CREATE FUNCTION vec_documentos.reservar_carga_directa_v1(
 p_preimagen bytea,p_carga bytea,p_reserva jsonb,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; j jsonb; v record; d jsonb; r vec_documentos.carga_directa_operacion%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
  OR octet_length(p_preimagen) NOT BETWEEN 1 AND 16384
  OR octet_length(p_carga) NOT BETWEEN 1 AND 65536
  OR pg_column_size(p_reserva)>4096 OR pg_column_size(p_auth)>4096
  OR p_reserva IS NULL OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: reserva denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; j:=convert_from(p_carga,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(j)<>'object' OR jsonb_typeof(p_reserva)<>'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>14
  OR (SELECT count(*) FROM jsonb_object_keys(p_reserva))<>5
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','persona_ref','clave_reserva','huella_reserva_sha256',
    'version_reserva','indice_carga_hmac','huella_carga_hmac','mime','tamano',
    'huella_contenido_sha256','huella_carga_json_sha256','huella_token_sha256'))
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(j) k WHERE k NOT IN
   ('id','version','principal_id','recurso_ref','modulo_id','tipo_recurso','operacion_ref',
    'correlacion_ref','finalidad','clasificacion','mime_declarado','tamano_declarado',
    'huella_declarada_sha256','indice_idempotencia_hmac','huella_solicitud_hmac',
    'vinculo_sesion_hmac','estado','autorizacion_preparacion_ref',
    'autorizacion_recepcion_ref','autorizacion_analisis_ref','autorizacion_promocion_ref',
    'contenido_cuarentena','analisis','contenido_admitido','creada_en','preparada_en',
    'expira_en','actualizada_en'))
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(p_reserva) k WHERE k NOT IN
   ('recurso_ref','persona_ref','clave_reserva','huella_reserva_sha256','version_reserva'))
  OR m->>'accion' IS DISTINCT FROM 'documentos.carga.preparar_reserva'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'carga_id'= ('ref:'||repeat('0',64))
  OR m->>'carga_id' IS DISTINCT FROM 'ref:'||encode(sha256(
    convert_to('vec:carga-documental:id:v3','UTF8')||decode('00','hex')||
    convert_to(m->>'indice_carga_hmac','UTF8')),'hex')
  OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$' OR m->>'recurso_ref'= ('ref:'||repeat('0',64))
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'huella_carga_json_sha256' IS DISTINCT FROM encode(sha256(p_carga),'hex')
  OR m->>'huella_token_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'huella_contenido_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'huella_reserva_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'indice_carga_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR m->>'huella_carga_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR j->>'id' IS DISTINCT FROM m->>'carga_id'
  OR j->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR j->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR j->>'indice_idempotencia_hmac' IS DISTINCT FROM m->>'indice_carga_hmac'
  OR j->>'huella_solicitud_hmac' IS DISTINCT FROM m->>'huella_carga_hmac'
  OR j->>'huella_declarada_sha256' IS DISTINCT FROM m->>'huella_contenido_sha256'
  OR j->>'mime_declarado' IS DISTINCT FROM m->>'mime'
  OR j->>'tamano_declarado' IS DISTINCT FROM m->>'tamano'
  OR j->>'version' IS DISTINCT FROM '1' OR j->>'estado' IS DISTINCT FROM 'reservada'
  OR j->>'modulo_id' IS DISTINCT FROM 'meritos' OR j->>'tipo_recurso' IS DISTINCT FROM 'hecho'
  OR j->>'finalidad' IS DISTINCT FROM 'custodiar_evidencia_meritos'
  OR p_reserva->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_reserva->>'persona_ref' IS DISTINCT FROM m->>'persona_ref'
  OR p_reserva->>'clave_reserva' IS DISTINCT FROM m->>'clave_reserva'
  OR p_reserva->>'huella_reserva_sha256' IS DISTINCT FROM m->>'huella_reserva_sha256'
  OR p_reserva->>'version_reserva' IS DISTINCT FROM m->>'version_reserva'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR p_auth->>'finalidad' IS DISTINCT FROM j->>'finalidad'
  OR p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D17: reserva no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:carga:'||(m->>'indice_carga_hmac'),0));
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion
  WHERE indice_carga_hmac=m->>'indice_carga_hmac' FOR UPDATE;
 IF FOUND THEN
  IF r.carga_id IS DISTINCT FROM m->>'carga_id' OR
    r.persona_ref IS DISTINCT FROM m->>'persona_ref' OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref'
    OR r.huella_carga_hmac IS DISTINCT FROM m->>'huella_carga_hmac'
    OR r.carga->>'huella_declarada_sha256' IS DISTINCT FROM m->>'huella_contenido_sha256'
    OR r.carga->>'mime_declarado' IS DISTINCT FROM m->>'mime'
    OR r.carga->>'tamano_declarado' IS DISTINCT FROM m->>'tamano'
    OR r.carga->>'operacion_ref' IS DISTINCT FROM j->>'operacion_ref'
    OR r.carga->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
    OR r.carga->>'finalidad' IS DISTINCT FROM j->>'finalidad'
    OR r.carga->>'clasificacion' IS DISTINCT FROM j->>'clasificacion'
  THEN RAISE EXCEPTION 'D17: clave divergente' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
   'version_operativa',r.version_operativa,'pendiente',true,'carga',r.carga);
 END IF;
 IF v.consumo_nuevo IS FALSE THEN RAISE EXCEPTION 'D17: replay sin carga' USING ERRCODE='42501'; END IF;
 SELECT vec_documentos.reclamar_reserva_recurso_carga_v1(
  m->>'recurso_ref',m->>'persona_ref',m->>'clave_reserva',m->>'huella_reserva_sha256',
  (m->>'version_reserva')::bigint,m->>'indice_carga_hmac',m->>'huella_carga_hmac') INTO d;
 IF d->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref' OR d->>'persona_ref' IS DISTINCT FROM m->>'persona_ref'
  OR d->>'clave_reserva' IS DISTINCT FROM m->>'clave_reserva'
  OR d->>'huella_reserva_sha256' IS DISTINCT FROM m->>'huella_reserva_sha256'
  OR d->>'indice_carga_hmac' IS DISTINCT FROM m->>'indice_carga_hmac'
  OR d->>'huella_carga_hmac' IS DISTINCT FROM m->>'huella_carga_hmac'
 THEN RAISE EXCEPTION 'D17: reserva D16 divergente' USING ERRCODE='42501'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 INSERT INTO vec_documentos.carga_directa_operacion(
  carga_id,recurso_ref,expediente_ref,persona_ref,indice_carga_hmac,huella_carga_hmac,
  huella_token_sha256,huella_preimagen_reserva_sha256,huella_carga_json_sha256,version_contenido,version_operativa,
  estado_operativo,carga,decision_reserva_ref,creada_en,actualizada_en)
 VALUES(m->>'carga_id',m->>'recurso_ref',m->>'recurso_ref',m->>'persona_ref',
  m->>'indice_carga_hmac',m->>'huella_carga_hmac',m->>'huella_token_sha256',h,m->>'huella_carga_json_sha256',1,1,
  'preparacion_iniciada',j,v.decision_ref,ahora,ahora) RETURNING * INTO r;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,1,'reservada','preparacion_iniciada',v.decision_ref,v.auditoria_ref,h,ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'preparacion_iniciada',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'pendiente',false,'carga',r.carga);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.reservar_carga_directa_v1(bytea,bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.reservar_carga_directa_v1(bytea,bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.confirmar_preparacion_carga_directa_v1(
 p_preimagen bytea,p_carga bytea,p_manifiesto bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; j jsonb; d jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 16384 OR
  octet_length(p_carga) NOT BETWEEN 1 AND 65536 OR
  octet_length(p_manifiesto) NOT BETWEEN 1 AND 65536 OR
  pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: confirmación denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=convert_from(p_preimagen,'UTF8')::jsonb;
  j:=convert_from(p_carga,'UTF8')::jsonb;
  d:=convert_from(p_manifiesto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(j)<>'object' OR jsonb_typeof(d)<>'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>11
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','persona_ref','huella_token_sha256',
    'huella_manifiesto_sha256','huella_manifiesto_json_sha256','version_esperada',
    'huella_anterior_sha256','version_siguiente','huella_siguiente_sha256'))
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(j) k WHERE k NOT IN
   ('id','version','principal_id','recurso_ref','modulo_id','tipo_recurso','operacion_ref',
    'correlacion_ref','finalidad','clasificacion','mime_declarado','tamano_declarado',
    'huella_declarada_sha256','indice_idempotencia_hmac','huella_solicitud_hmac',
    'vinculo_sesion_hmac','estado','autorizacion_preparacion_ref',
    'autorizacion_recepcion_ref','autorizacion_analisis_ref','autorizacion_promocion_ref',
    'contenido_cuarentena','analisis','contenido_admitido','creada_en','preparada_en',
    'expira_en','actualizada_en'))
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(d) k WHERE k NOT IN
   ('esquema','carga_id','version_carga','huella_carga_sha256','indice_idempotencia_hmac',
    'operacion_ref','correlacion_ref','carga_ref','finalidad','clasificacion',
    'sujeto_seudonimo_hmac','huella_solicitud_hmac','recurso_ref','modulo_id',
    'tipo_recurso','huella_recurso_base_sha256','huella_recurso_sha256','mime',
    'tamano','huella_contenido_sha256','vinculo_sesion_hmac','conector_almacen_id',
    'esquema_contexto','accion_negocio','accion_tecnica','paso_ref','efecto_ref',
    'huella_plan_efecto_sha256','esquema_huella_decision','decision_ref',
    'huella_decision_sha256','contexto_verificado_en','decision_valida_hasta',
    'preparada_en','expira_en','huella_manifiesto_sha256'))
  OR m->>'accion' IS DISTINCT FROM 'documentos.carga.preparar_confirmacion'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'huella_token_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'huella_manifiesto_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'huella_anterior_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'huella_siguiente_sha256' IS DISTINCT FROM encode(sha256(p_carga),'hex')
  OR m->>'huella_manifiesto_json_sha256' IS DISTINCT FROM encode(sha256(p_manifiesto),'hex')
  OR m->>'version_esperada' IS DISTINCT FROM '1' OR m->>'version_siguiente' IS DISTINCT FROM '2'
  OR j->>'id' IS DISTINCT FROM m->>'carga_id' OR j->>'version' IS DISTINCT FROM '2'
  OR j->>'estado' IS DISTINCT FROM 'preparada'
  OR j->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR j->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR d->>'carga_id' IS DISTINCT FROM m->>'carga_id'
  OR d->>'version_carga' IS DISTINCT FROM '2'
  OR d->>'huella_carga_sha256' IS DISTINCT FROM m->>'huella_siguiente_sha256'
  OR d->>'huella_manifiesto_sha256' IS DISTINCT FROM m->>'huella_manifiesto_sha256'
  OR d->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR d->>'vinculo_sesion_hmac' IS DISTINCT FROM j->>'vinculo_sesion_hmac'
  OR d->>'indice_idempotencia_hmac' IS DISTINCT FROM j->>'indice_idempotencia_hmac'
  OR d->>'huella_solicitud_hmac' IS DISTINCT FROM j->>'huella_solicitud_hmac'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR p_auth->>'finalidad' IS DISTINCT FROM j->>'finalidad'
  OR p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D17: confirmación no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.persona_ref IS DISTINCT FROM m->>'persona_ref' OR
  r.recurso_ref IS DISTINCT FROM m->>'recurso_ref' OR
  r.huella_token_sha256 IS DISTINCT FROM m->>'huella_token_sha256' OR
  r.huella_carga_json_sha256 IS DISTINCT FROM m->>'huella_anterior_sha256' OR
  r.version_contenido<>1 OR r.version_operativa<>1 OR r.estado_operativo<>'preparacion_iniciada' OR
  v.consumo_nuevo IS FALSE
 THEN RAISE EXCEPTION 'D17: preparación no recuperable automáticamente' USING ERRCODE='23505'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 UPDATE vec_documentos.carga_directa_operacion SET
  carga=j,version_contenido=2,version_operativa=3,estado_operativo='emision_recibo_iniciada',
  manifiesto=d,huella_manifiesto_sha256=m->>'huella_manifiesto_sha256',
  huella_carga_json_sha256=m->>'huella_siguiente_sha256',decision_confirmacion_ref=v.decision_ref,
  actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=1 RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS preparación' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,2,'preparacion_iniciada','preparada',v.decision_ref,v.auditoria_ref,h,ahora),
       (r.carga_id,r.recurso_ref,r.persona_ref,3,'preparada','emision_recibo_iniciada',v.decision_ref,v.auditoria_ref,h,ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'emision_recibo_iniciada',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'carga',r.carga,'manifiesto',r.manifiesto);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.confirmar_preparacion_carga_directa_v1(bytea,bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.confirmar_preparacion_carga_directa_v1(bytea,bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.registrar_recibo_carga_directa_v1(
 p_preimagen bytea,p_registro bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; g jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 o vec_documentos.recibo_carga_directa%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 16384 OR
  octet_length(p_registro) NOT BETWEEN 1 AND 4096 OR
  pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: alta recibo denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; g:=convert_from(p_registro,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: preimagen recibo inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(g)<>'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>13
  OR (SELECT count(*) FROM jsonb_object_keys(g))<>6
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','version_carga','persona_ref','registro_contexto_ref',
    'huella_contexto_sha256','huella_manifiesto_sha256','indice_hmac','grupo_hmac',
    'vinculo_hmac','evidencia_alta_ref','expira_en'))
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(g) k WHERE k NOT IN
   ('indice_hmac','grupo_hmac','vinculo_hmac','evidencia_alta_ref',
    'autorizacion_emision_ref','expira_en'))
  OR m->>'accion' IS DISTINCT FROM 'documentos.carga.preparar_respuesta'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'version_carga' IS DISTINCT FROM '2'
  OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'huella_contexto_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'huella_manifiesto_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'indice_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR m->>'grupo_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR m->>'vinculo_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR g->>'indice_hmac' IS DISTINCT FROM m->>'indice_hmac'
  OR g->>'grupo_hmac' IS DISTINCT FROM m->>'grupo_hmac'
  OR g->>'vinculo_hmac' IS DISTINCT FROM m->>'vinculo_hmac'
  OR g->>'evidencia_alta_ref' IS DISTINCT FROM m->>'evidencia_alta_ref'
  OR g->>'expira_en' IS DISTINCT FROM m->>'expira_en'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
 THEN RAISE EXCEPTION 'D17: recibo no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- La decisión V3 se consume antes de consultar. RLS se acota al recurso
 -- alegado; la fila leída debe confirmar después titular y recurso exactos.
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.persona_ref IS DISTINCT FROM m->>'persona_ref'
  OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref'
  OR r.version_contenido<>2 OR r.version_operativa<>3 OR r.estado_operativo<>'emision_recibo_iniciada'
  OR r.huella_manifiesto_sha256 IS DISTINCT FROM m->>'huella_manifiesto_sha256'
  OR r.manifiesto->>'decision_ref' IS DISTINCT FROM g->>'autorizacion_emision_ref'
  OR r.carga->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR r.carga->>'version' IS DISTINCT FROM m->>'version_carga'
  OR v.consumo_nuevo IS FALSE
  OR (m->>'expira_en')::timestamptz<=ahora
 THEN RAISE EXCEPTION 'D17: recibo no registrable' USING ERRCODE='23505'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 INSERT INTO vec_documentos.recibo_carga_directa(
  indice_hmac,carga_id,expediente_ref,persona_ref,grupo_hmac,vinculo_hmac,evidencia_alta_ref,
  autorizacion_emision_ref,expira_en,registrado_en,estado,version)
 VALUES(m->>'indice_hmac',r.carga_id,r.recurso_ref,r.persona_ref,m->>'grupo_hmac',m->>'vinculo_hmac',
  m->>'evidencia_alta_ref',g->>'autorizacion_emision_ref',(m->>'expira_en')::timestamptz,
  ahora,'activo',1) RETURNING * INTO o;
 UPDATE vec_documentos.carga_directa_operacion SET version_operativa=4,estado_operativo='recibo_registrado',
  recibo_indice_hmac=o.indice_hmac,decision_respuesta_ref=v.decision_ref,actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=3 RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS recibo' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,evidencia_ref,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,4,'emision_recibo_iniciada','recibo_registrado',
  v.decision_ref,v.auditoria_ref,h,o.evidencia_alta_ref,ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'recibo_registrado',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('indice_hmac',o.indice_hmac,'grupo_hmac',o.grupo_hmac,
  'autorizacion_emision_ref',o.autorizacion_emision_ref,'registrado_en',o.registrado_en,
  'estado_operativo',r.estado_operativo,'entrega_acreditada',false);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.registrar_recibo_carga_directa_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.registrar_recibo_carga_directa_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.consumir_recibo_carga_directa_v3_v1(
 p_preimagen bytea,p_orden bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; g jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 o vec_documentos.recibo_carga_directa%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 16384 OR
  octet_length(p_orden) NOT BETWEEN 1 AND 4096 OR pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: consumo denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; g:=convert_from(p_orden,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: consumo inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(g)<>'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>15
  OR (SELECT count(*) FROM jsonb_object_keys(g))<>8
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','persona_ref','version_carga','registro_contexto_ref',
    'huella_contexto_sha256','indice_hmac','grupo_hmac','vinculo_hmac',
    'evidencia_consumo_ref','intencion_ref','huella_intencion_hmac','registrado_en','valida_hasta'))
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(g) k WHERE k NOT IN
   ('indice_hmac','grupo_hmac','vinculo_hmac','evidencia_consumo_ref',
    'intencion_confirmacion_ref','huella_intencion_hmac','registrado_en','valida_hasta'))
  OR m->>'accion' IS DISTINCT FROM 'documentos.carga.confirmar_intencion'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'version_carga' IS DISTINCT FROM '2'
  OR m->>'indice_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR m->>'grupo_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR m->>'vinculo_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR m->>'huella_intencion_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR g->>'indice_hmac' IS DISTINCT FROM m->>'indice_hmac'
  OR g->>'grupo_hmac' IS DISTINCT FROM m->>'grupo_hmac'
  OR g->>'vinculo_hmac' IS DISTINCT FROM m->>'vinculo_hmac'
  OR g->>'evidencia_consumo_ref' IS DISTINCT FROM m->>'evidencia_consumo_ref'
  OR g->>'intencion_confirmacion_ref' IS DISTINCT FROM m->>'intencion_ref'
  OR g->>'huella_intencion_hmac' IS DISTINCT FROM m->>'huella_intencion_hmac'
  OR g->>'registrado_en' IS DISTINCT FROM m->>'registrado_en'
  OR g->>'valida_hasta' IS DISTINCT FROM m->>'valida_hasta'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
 THEN RAISE EXCEPTION 'D17: intención no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref' OR
  r.persona_ref IS DISTINCT FROM m->>'persona_ref' OR r.version_contenido<>2 OR
  r.version_operativa<>4 OR r.estado_operativo<>'recibo_registrado' OR
  r.recibo_indice_hmac IS DISTINCT FROM m->>'indice_hmac' OR v.consumo_nuevo IS FALSE
 THEN RAISE EXCEPTION 'D17: confirmación ya iniciada' USING ERRCODE='23505'; END IF;
 SELECT * INTO o FROM vec_documentos.recibo_carga_directa WHERE indice_hmac=m->>'indice_hmac' FOR UPDATE;
 IF NOT FOUND OR o.carga_id IS DISTINCT FROM r.carga_id OR o.estado<>'activo' OR
  o.grupo_hmac IS DISTINCT FROM m->>'grupo_hmac' OR o.vinculo_hmac IS DISTINCT FROM m->>'vinculo_hmac' OR
  o.registrado_en::text IS NULL OR o.registrado_en IS DISTINCT FROM (m->>'registrado_en')::timestamptz OR
  ahora<o.registrado_en OR ahora>=o.expira_en OR ahora>=(m->>'valida_hasta')::timestamptz
 THEN RAISE EXCEPTION 'D17: recibo no consumible' USING ERRCODE='23505'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 UPDATE vec_documentos.recibo_carga_directa SET estado='consumido',version=2,
  evidencia_consumo_ref=m->>'evidencia_consumo_ref',
  intencion_confirmacion_ref=m->>'intencion_ref',huella_intencion_hmac=m->>'huella_intencion_hmac',
  valida_hasta=(m->>'valida_hasta')::timestamptz,consumido_en=ahora
 WHERE indice_hmac=o.indice_hmac AND version=1 RETURNING * INTO o;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS consumo' USING ERRCODE='40001'; END IF;
 UPDATE vec_documentos.carga_directa_operacion SET estado_operativo='confirmacion_iniciada',
  version_operativa=5,actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=4 RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS intención' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,evidencia_ref,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,5,'recibo_registrado','confirmacion_iniciada',
  v.decision_ref,v.auditoria_ref,h,o.intencion_confirmacion_ref,ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'confirmacion_iniciada',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('indice_hmac',o.indice_hmac,'grupo_hmac',o.grupo_hmac,'vinculo_hmac',o.vinculo_hmac,
  'evidencia_consumo_ref',o.evidencia_consumo_ref,'intencion_confirmacion_ref',o.intencion_confirmacion_ref,
  'huella_intencion_hmac',o.huella_intencion_hmac,'registrado_en',o.registrado_en,
  'consumido_en',o.consumido_en,'expira_en',o.expira_en,'valida_hasta',o.valida_hasta,
  'estado_operativo',r.estado_operativo);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.consumir_recibo_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.consumir_recibo_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.confirmar_resultado_carga_directa_v3_v1(
 p_preimagen bytea,p_carga bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; j jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 o vec_documentos.recibo_carga_directa%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 16384 OR
  octet_length(p_carga) NOT BETWEEN 1 AND 65536 OR pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: resultado denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; j:=convert_from(p_carga,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: resultado inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(j)<>'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>10
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','persona_ref','version_esperada',
    'huella_anterior_sha256','version_siguiente','huella_siguiente_sha256',
    'autorizacion_v1_ref','evento'))
  OR m->>'accion' IS DISTINCT FROM 'documentos.carga.confirmar_resultado'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'version_esperada' IS DISTINCT FROM '2' OR m->>'version_siguiente' IS DISTINCT FROM '3'
  OR m->>'evento' IS DISTINCT FROM 'vec.documentos.carga.recibida'
  OR m->>'huella_siguiente_sha256' IS DISTINCT FROM encode(sha256(p_carga),'hex')
  OR j->>'id' IS DISTINCT FROM m->>'carga_id' OR j->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR j->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR j->>'version' IS DISTINCT FROM '3' OR j->>'estado' IS DISTINCT FROM 'cuarentena'
  OR j->>'autorizacion_recepcion_ref' IS DISTINCT FROM m->>'autorizacion_v1_ref'
  OR jsonb_typeof(j->'contenido_cuarentena') IS DISTINCT FROM 'object'
  OR j#>>'{contenido_cuarentena,zona}' IS DISTINCT FROM 'cuarentena'
  OR j#>>'{contenido_cuarentena,huella_sha256}' IS DISTINCT FROM j->>'huella_declarada_sha256'
  OR j#>>'{contenido_cuarentena,mime}' IS DISTINCT FROM j->>'mime_declarado'
  OR j#>>'{contenido_cuarentena,tamano}' IS DISTINCT FROM j->>'tamano_declarado'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR p_auth->>'finalidad' IS DISTINCT FROM j->>'finalidad'
  OR p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D17: resultado no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref' OR
  r.persona_ref IS DISTINCT FROM m->>'persona_ref' OR r.version_contenido<>2 OR
  r.version_operativa<>5 OR r.estado_operativo<>'confirmacion_iniciada' OR
  r.huella_carga_json_sha256 IS DISTINCT FROM m->>'huella_anterior_sha256' OR
  r.carga->>'indice_idempotencia_hmac' IS DISTINCT FROM j->>'indice_idempotencia_hmac' OR
  r.carga->>'vinculo_sesion_hmac' IS DISTINCT FROM j->>'vinculo_sesion_hmac' OR
  v.consumo_nuevo IS FALSE
 THEN RAISE EXCEPTION 'D17: resultado fuera de intención' USING ERRCODE='23505'; END IF;
 SELECT * INTO o FROM vec_documentos.recibo_carga_directa WHERE indice_hmac=r.recibo_indice_hmac FOR SHARE;
 IF NOT FOUND OR o.estado<>'consumido' OR o.carga_id IS DISTINCT FROM r.carga_id OR
  o.intencion_confirmacion_ref IS NULL OR o.huella_intencion_hmac IS NULL
 THEN RAISE EXCEPTION 'D17: recibo sin intención' USING ERRCODE='23505'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 UPDATE vec_documentos.carga_directa_operacion SET carga=j,version_contenido=3,
  version_operativa=6,estado_operativo='cuarentena',
  huella_carga_json_sha256=m->>'huella_siguiente_sha256',actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=5 RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS resultado' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,evidencia_ref,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,6,'confirmacion_iniciada','cuarentena',
  v.decision_ref,v.auditoria_ref,h,j#>>'{contenido_cuarentena,evidencia_ref}',ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'cuarentena',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'carga',r.carga);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.confirmar_resultado_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.confirmar_resultado_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.registrar_analisis_carga_directa_v3_v1(
 p_preimagen bytea,p_carga bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; j jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text; estado text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 16384 OR
  octet_length(p_carga) NOT BETWEEN 1 AND 65536 OR pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: análisis denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; j:=convert_from(p_carga,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: análisis inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(j)<>'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>10
  OR m->>'accion' IS DISTINCT FROM 'vec.documentos.carga.analizar'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'version_esperada' IS DISTINCT FROM '3' OR m->>'version_siguiente' IS DISTINCT FROM '4'
  OR m->>'evento' IS DISTINCT FROM 'vec.documentos.carga.analizada'
  OR m->>'huella_siguiente_sha256' IS DISTINCT FROM encode(sha256(p_carga),'hex')
  OR j->>'id' IS DISTINCT FROM m->>'carga_id' OR j->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR j->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR j->>'version' IS DISTINCT FROM '4'
  OR j->>'estado' NOT IN ('analizada_limpia','retenida_seguridad')
  OR (j->>'estado'='analizada_limpia') IS DISTINCT FROM (j#>>'{analisis,estado}'='limpio')
  OR j->>'autorizacion_analisis_ref' IS DISTINCT FROM m->>'autorizacion_v1_ref'
  OR jsonb_typeof(j->'analisis') IS DISTINCT FROM 'object'
  OR j#>>'{analisis,objeto_referencia}' IS DISTINCT FROM j#>>'{contenido_cuarentena,referencia}'
  OR j#>>'{analisis,objeto_version}' IS DISTINCT FROM j#>>'{contenido_cuarentena,version}'
  OR j#>>'{analisis,huella_objeto_sha256}' IS DISTINCT FROM j#>>'{contenido_cuarentena,huella_sha256}'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR p_auth->>'finalidad' IS DISTINCT FROM j->>'finalidad'
  OR p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D17: análisis no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref' OR
  r.persona_ref IS DISTINCT FROM m->>'persona_ref' OR r.version_contenido<>3 OR
  r.version_operativa<>6 OR r.estado_operativo<>'cuarentena' OR
  r.huella_carga_json_sha256 IS DISTINCT FROM m->>'huella_anterior_sha256' OR
  r.carga->'contenido_cuarentena' IS DISTINCT FROM j->'contenido_cuarentena' OR
  v.consumo_nuevo IS FALSE
 THEN RAISE EXCEPTION 'D17: análisis fuera de versión' USING ERRCODE='23505'; END IF;
 estado:=j->>'estado';h:=encode(sha256(p_preimagen),'hex');
 UPDATE vec_documentos.carga_directa_operacion SET carga=j,version_contenido=4,
  version_operativa=7,estado_operativo=estado,
  huella_carga_json_sha256=m->>'huella_siguiente_sha256',actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=6 RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS análisis' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,evidencia_ref,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,7,'cuarentena',estado,
  v.decision_ref,v.auditoria_ref,h,j#>>'{analisis,evidencia_ref}',ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,estado,ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'carga',r.carga);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.registrar_analisis_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.registrar_analisis_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.iniciar_promocion_carga_directa_v3_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 16384 OR pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: promoción denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: promoción inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>13
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','persona_ref','version_carga','huella_carga_json_sha256',
    'indice_promocion_hmac','origen_ref','origen_version','huella_origen_sha256',
    'evidencia_analisis_ref','registro_contexto_ref','huella_contexto_sha256'))
  OR m->>'accion' IS DISTINCT FROM 'documentos.carga.promover_intencion'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'version_carga' IS DISTINCT FROM '4'
  OR m->>'huella_carga_json_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'huella_origen_sha256' !~ '^[0-9a-f]{64}$'
  OR m->>'indice_promocion_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
 THEN RAISE EXCEPTION 'D17: promoción no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref' OR
  r.persona_ref IS DISTINCT FROM m->>'persona_ref' OR r.version_contenido<>4 OR
  r.version_operativa<>7 OR r.estado_operativo<>'analizada_limpia' OR
  r.huella_carga_json_sha256 IS DISTINCT FROM m->>'huella_carga_json_sha256' OR
  r.carga->>'estado' IS DISTINCT FROM 'analizada_limpia' OR
  r.carga#>>'{analisis,estado}' IS DISTINCT FROM 'limpio' OR
  r.carga#>>'{analisis,evidencia_ref}' IS DISTINCT FROM m->>'evidencia_analisis_ref' OR
  r.carga#>>'{contenido_cuarentena,referencia}' IS DISTINCT FROM m->>'origen_ref' OR
  r.carga#>>'{contenido_cuarentena,version}' IS DISTINCT FROM m->>'origen_version' OR
  r.carga#>>'{contenido_cuarentena,huella_sha256}' IS DISTINCT FROM m->>'huella_origen_sha256' OR
  v.consumo_nuevo IS FALSE
 THEN RAISE EXCEPTION 'D17: promoción fuera de versión' USING ERRCODE='23505'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 UPDATE vec_documentos.carga_directa_operacion SET
  estado_operativo='promocion_iniciada',version_operativa=8,
  indice_promocion_hmac=m->>'indice_promocion_hmac',
  decision_promocion_intencion_ref=v.decision_ref,actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=7 RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS promoción' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,evidencia_ref,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,8,'analizada_limpia','promocion_iniciada',
  v.decision_ref,v.auditoria_ref,h,m->>'evidencia_analisis_ref',ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'promocion_iniciada',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'indice_promocion_hmac',r.indice_promocion_hmac);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.iniciar_promocion_carga_directa_v3_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.iniciar_promocion_carga_directa_v3_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.confirmar_promocion_carga_directa_v3_v1(
 p_preimagen bytea,p_carga bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; j jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 16384 OR
  octet_length(p_carga) NOT BETWEEN 1 AND 65536 OR pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: resultado promoción denegado' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; j:=convert_from(p_carga,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: resultado promoción inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR jsonb_typeof(j)<>'object'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>11
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','persona_ref','version_esperada','huella_anterior_sha256',
    'version_siguiente','huella_siguiente_sha256','autorizacion_v1_ref','evento','indice_promocion_hmac'))
  OR m->>'accion' IS DISTINCT FROM 'documentos.carga.promover_resultado'
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'version_esperada' IS DISTINCT FROM '4' OR m->>'version_siguiente' IS DISTINCT FROM '5'
  OR m->>'evento' IS DISTINCT FROM 'vec.documentos.carga.promovida'
  OR m->>'indice_promocion_hmac' !~ '^hmac-sha256:[^:]{3,64}:[0-9a-f]{64}$'
  OR m->>'huella_siguiente_sha256' IS DISTINCT FROM encode(sha256(p_carga),'hex')
  OR j->>'id' IS DISTINCT FROM m->>'carga_id' OR j->>'recurso_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR j->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR j->>'version' IS DISTINCT FROM '5' OR j->>'estado' IS DISTINCT FROM 'admitida'
  OR j->>'autorizacion_promocion_ref' IS DISTINCT FROM m->>'autorizacion_v1_ref'
  OR j#>>'{analisis,estado}' IS DISTINCT FROM 'limpio'
  OR jsonb_typeof(j->'contenido_admitido') IS DISTINCT FROM 'object'
  OR j#>>'{contenido_admitido,zona}' IS DISTINCT FROM 'admitida'
  OR j#>>'{contenido_admitido,huella_sha256}' IS DISTINCT FROM j->>'huella_declarada_sha256'
  OR j#>>'{contenido_admitido,mime}' IS DISTINCT FROM j->>'mime_declarado'
  OR j#>>'{contenido_admitido,tamano}' IS DISTINCT FROM j->>'tamano_declarado'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR p_auth->>'finalidad' IS DISTINCT FROM j->>'finalidad'
  OR p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D17: resultado promoción no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref' OR
  r.persona_ref IS DISTINCT FROM m->>'persona_ref' OR r.version_contenido<>4 OR
  r.version_operativa<>8 OR r.estado_operativo<>'promocion_iniciada' OR
  r.indice_promocion_hmac IS DISTINCT FROM m->>'indice_promocion_hmac' OR
  r.huella_carga_json_sha256 IS DISTINCT FROM m->>'huella_anterior_sha256' OR
  r.carga->'contenido_cuarentena' IS DISTINCT FROM j->'contenido_cuarentena' OR
  r.carga->'analisis' IS DISTINCT FROM j->'analisis' OR v.consumo_nuevo IS FALSE
 THEN RAISE EXCEPTION 'D17: resultado promoción fuera de intención' USING ERRCODE='23505'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 UPDATE vec_documentos.carga_directa_operacion SET carga=j,version_contenido=5,
  version_operativa=9,estado_operativo='admitida',
  huella_carga_json_sha256=m->>'huella_siguiente_sha256',actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=8 RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS resultado promoción' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,evidencia_ref,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,9,'promocion_iniciada','admitida',
  v.decision_ref,v.auditoria_ref,h,j#>>'{contenido_admitido,evidencia_ref}',ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'admitida',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'carga',r.carga);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.confirmar_promocion_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.confirmar_promocion_carga_directa_v3_v1(bytea,bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.consultar_carga_directa_propia_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 4096 OR
  pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: consulta denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: consulta inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR m->>'accion' IS DISTINCT FROM 'documentos.carga.consultar_propia'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>10
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','persona_ref','perfil_activo_ref',
    'registro_contexto_ref','huella_contexto_sha256','finalidad','correlacion_ref',
    'incluye_manifiesto'))
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
  OR m->>'huella_contexto_sha256' !~ '^[0-9a-f]{64}$'
  OR jsonb_typeof(m->'incluye_manifiesto') IS DISTINCT FROM 'boolean'
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
  OR p_auth->>'principal_id' IS DISTINCT FROM m->>'persona_ref'
  OR p_auth->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_activo_ref'
  OR p_auth->>'finalidad' IS DISTINCT FROM m->>'finalidad'
  OR p_auth->>'correlacion_ref' IS DISTINCT FROM m->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D17: consulta no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS FALSE THEN RAISE EXCEPTION 'D17: consulta requiere decisión nueva' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion
  WHERE carga_id=m->>'carga_id' AND recurso_ref=m->>'recurso_ref'
    AND persona_ref=m->>'persona_ref';
 IF NOT FOUND THEN
  INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
   finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
  VALUES(m->>'accion',m->>'carga_id',m->>'recurso_ref',m->>'persona_ref',p_auth->>'perfil_activo_ref',
   p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'denegado',ahora);
  RETURN jsonb_build_object('denegado',true);
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,r.persona_ref,p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'autorizado',ahora);
 RETURN jsonb_build_object('carga',r.carga,'manifiesto',CASE WHEN (m->>'incluye_manifiesto')::boolean
  THEN r.manifiesto ELSE NULL END,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'entrega_acreditada',false);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.consultar_carga_directa_propia_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.consultar_carga_directa_propia_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.cerrar_carga_directa_no_recuperable_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on
 SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; r vec_documentos.carga_directa_operacion%ROWTYPE;
 o vec_documentos.recibo_carga_directa%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text; anterior text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 4096 OR
  pg_column_size(p_auth)>4096 OR p_auth IS NULL
 THEN RAISE EXCEPTION 'D17: conciliación denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'D17: conciliación inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR m->>'accion' IS DISTINCT FROM 'documentos.carga.conciliar'
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>9
  OR EXISTS(SELECT 1 FROM jsonb_object_keys(m) k WHERE k NOT IN
   ('accion','carga_id','recurso_ref','version_operativa_esperada',
    'estado_operativo_esperado','estado_resultado','evidencia_ref',
    'huella_evidencia_sha256','motivo_ref'))
  OR m->>'carga_id' !~ '^ref:[0-9a-f]{64}$' OR m->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$'
  OR m->>'estado_resultado' IS DISTINCT FROM 'no_recuperable'
  OR m->>'estado_operativo_esperado' NOT IN
   ('preparacion_iniciada','preparacion_indeterminada','emision_recibo_iniciada',
    'recibo_indeterminado','recibo_registrado')
  OR jsonb_typeof(m->'version_operativa_esperada') IS DISTINCT FROM 'number'
  OR (m->>'version_operativa_esperada')::numeric<1
  OR m->>'huella_evidencia_sha256' !~ '^[0-9a-f]{64}$'
  OR NOT vec_documentos.referencia_opaca_v1(m->>'evidencia_ref')
  OR NOT vec_documentos.referencia_opaca_v1(m->>'motivo_ref')
  OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
  OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'carga_id'
  OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'recurso_ref'
 THEN RAISE EXCEPTION 'D17: conciliación no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_carga_directa_v3_v1(
  p_preimagen,m->>'accion',m->>'carga_id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS FALSE THEN RAISE EXCEPTION 'D17: conciliación ya consumida' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.documentos.expediente_ref',m->>'recurso_ref',true);
 SELECT * INTO r FROM vec_documentos.carga_directa_operacion WHERE carga_id=m->>'carga_id' FOR UPDATE;
 IF NOT FOUND OR r.recurso_ref IS DISTINCT FROM m->>'recurso_ref' OR
  r.version_operativa::text IS DISTINCT FROM m->>'version_operativa_esperada' OR
  r.estado_operativo IS DISTINCT FROM m->>'estado_operativo_esperado' OR
  r.estado_operativo='no_recuperable'
 THEN RAISE EXCEPTION 'D17: conciliación en conflicto' USING ERRCODE='23505'; END IF;
 IF r.recibo_indice_hmac IS NOT NULL THEN
  SELECT * INTO o FROM vec_documentos.recibo_carga_directa
   WHERE indice_hmac=r.recibo_indice_hmac FOR UPDATE;
  IF NOT FOUND OR o.estado<>'activo' OR ahora<o.expira_en
  THEN RAISE EXCEPTION 'D17: recibo con efecto posible' USING ERRCODE='23505'; END IF;
 ELSIF ahora<(r.carga->>'expira_en')::timestamptz THEN
  RAISE EXCEPTION 'D17: sesión todavía vigente' USING ERRCODE='23505';
 END IF;
 anterior:=r.estado_operativo;
 h:=encode(sha256(p_preimagen),'hex');
 UPDATE vec_documentos.carga_directa_operacion SET
  estado_operativo='no_recuperable',version_operativa=version_operativa+1,actualizada_en=ahora
 WHERE carga_id=r.carga_id AND version_operativa=r.version_operativa RETURNING * INTO r;
 IF NOT FOUND THEN RAISE EXCEPTION 'D17: CAS conciliación' USING ERRCODE='40001'; END IF;
 INSERT INTO vec_documentos.carga_directa_historia(carga_id,expediente_ref,persona_ref,version_operativa,
  estado_anterior,estado_siguiente,decision_ref,auditoria_ad3_ref,huella_preimagen_sha256,evidencia_ref,registrada_en)
 VALUES(r.carga_id,r.recurso_ref,r.persona_ref,r.version_operativa,anterior,'no_recuperable',
  v.decision_ref,v.auditoria_ref,h,m->>'evidencia_ref',ahora);
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,
  finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.carga_id,r.recurso_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'no_recuperable',ahora);
 INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
 VALUES('carga_directa_estado',r.carga_id,r.recurso_ref,h,ahora);
 RETURN jsonb_build_object('carga_id',r.carga_id,'estado_operativo',r.estado_operativo,
  'version_operativa',r.version_operativa,'entrega_acreditada',false);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.cerrar_carga_directa_no_recuperable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.cerrar_carga_directa_no_recuperable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;
COMMIT;
