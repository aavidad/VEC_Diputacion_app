\set ON_ERROR_STOP on
-- Instalar roles_up y AD3-000060 antes de esta migración. Solo añade objetos.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000001',0));
DO $pre$
BEGIN
 IF current_user <> 'vec_documentos_propietario'
    OR to_regclass('vec_documentos.documento') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_documentos_propietario',
      'vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'documentos 000001: AD3-60 o preimagen incompatibles' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE SEQUENCE vec_documentos.numero_interno_seq AS bigint START WITH 1 NO CYCLE;
REVOKE ALL ON SEQUENCE vec_documentos.numero_interno_seq FROM PUBLIC;

CREATE FUNCTION vec_documentos.referencia_opaca_v1(p text) RETURNS boolean
LANGUAGE sql IMMUTABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT p IS NOT NULL AND (
  (p ~ '^ref:[0-9a-f]{64}$' AND p <> 'ref:'||repeat('0',64))
  OR p ~ '^[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$')
 $f$;

CREATE TABLE vec_documentos.documento (
 id text PRIMARY KEY CHECK(vec_documentos.referencia_opaca_v1(id)),
 numero_vec text NOT NULL UNIQUE CHECK(numero_vec ~ '^VEC-[0-9]{4}-[0-9]{1,12}$'),
 clave_idempotencia text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(clave_idempotencia)),
 principal_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(principal_ref)),
 modulo_id text NOT NULL,
 expediente_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(expediente_ref)),
 tipo_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(tipo_ref)),
 version bigint NOT NULL CHECK(version>0),
 mime text NOT NULL CHECK(length(mime) BETWEEN 1 AND 255 AND mime !~ '[[:cntrl:]]'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 tamano bigint NOT NULL CHECK(tamano>0),
 objeto_ref text NOT NULL,
 objeto_version text NOT NULL,
 conector_ref text NOT NULL,
 recibo_objeto_ref text NOT NULL,
 recibo_objeto_huella_sha256 text NOT NULL CHECK(recibo_objeto_huella_sha256 ~ '^[0-9a-f]{64}$'),
 -- Nulo solo con política provisional: el conector no fija retención hasta
 -- aplicar el catálogo definitivo (tarea pendiente en el README).
 objeto_retenido_hasta timestamptz(6),
 objeto_inmovilizado boolean NOT NULL,
 politica_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(politica_ref)),
 version_politica bigint NOT NULL CHECK(version_politica>0),
 huella_politica_sha256 text NOT NULL CHECK(huella_politica_sha256 ~ '^[0-9a-f]{64}$'),
 conservacion_hasta timestamptz(6) NOT NULL,
 proteccion text NOT NULL CHECK(proteccion IN ('conservacion','bloqueo')),
 estado_politica text NOT NULL CHECK(estado_politica IN ('aprobada','provisional')),
 CHECK((estado_politica='aprobada' AND objeto_retenido_hasta IS NOT NULL AND objeto_retenido_hasta>=conservacion_hasta)
    OR (estado_politica='provisional' AND objeto_retenido_hasta IS NULL AND NOT objeto_inmovilizado AND proteccion='conservacion')),
 CHECK(proteccion<>'bloqueo' OR objeto_inmovilizado),
 estado_firma text NOT NULL DEFAULT 'pendiente_proveedor' CHECK(estado_firma='pendiente_proveedor'),
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ad3_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 UNIQUE(principal_ref,clave_idempotencia),
 UNIQUE(modulo_id,expediente_ref,tipo_ref,version)
);
CREATE INDEX documento_expediente_idx ON vec_documentos.documento(expediente_ref,creada_en,id);

CREATE TABLE vec_documentos.preparacion_notificacion (
 id text PRIMARY KEY CHECK(vec_documentos.referencia_opaca_v1(id)),
 clave_idempotencia text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(clave_idempotencia)),
 documento_id text NOT NULL REFERENCES vec_documentos.documento(id),
 expediente_ref text NOT NULL,
 version bigint NOT NULL CHECK(version>0),
 destinatario_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(destinatario_ref)),
 canal text NOT NULL,
 estado text NOT NULL DEFAULT 'preparada' CHECK(estado='preparada'),
 principal_ref text NOT NULL,
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ad3_ref text NOT NULL,
 preparada_en timestamptz(6) NOT NULL,
 UNIQUE(principal_ref,clave_idempotencia)
);
CREATE TABLE vec_documentos.auditoria_operacion (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 accion text NOT NULL CHECK(accion IN ('documentos.generado.alta','documentos.expediente.listar','documentos.original.descargar','documentos.notificacion.preparar')),
 recurso_ref text NOT NULL,
 expediente_ref text NOT NULL,
 principal_ref text NOT NULL,
 perfil_ref text NOT NULL,
 finalidad text NOT NULL,
 correlacion_ref text NOT NULL,
 decision_ref text NOT NULL,
 auditoria_ad3_ref text NOT NULL,
 resultado text NOT NULL CHECK(resultado IN ('creado','repetido','autorizado','preparado')),
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_documentos.outbox (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 tipo text NOT NULL CHECK(tipo IN ('documento_generado','notificacion_preparada')),
 recurso_ref text NOT NULL,
 expediente_ref text NOT NULL,
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL
);

DO $rls$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['documento','preparacion_notificacion','auditoria_operacion','outbox'] LOOP
  EXECUTE format('ALTER TABLE vec_documentos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_documentos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY alcance_lectura ON vec_documentos.%I FOR SELECT TO vec_documentos_propietario USING (expediente_ref = nullif(current_setting(''vec.documentos.expediente_ref'',true),''''))',t);
  EXECUTE format('CREATE POLICY alcance_alta ON vec_documentos.%I FOR INSERT TO vec_documentos_propietario WITH CHECK (expediente_ref = nullif(current_setting(''vec.documentos.expediente_ref'',true),''''))',t);
 END LOOP;
END $rls$;

CREATE FUNCTION vec_documentos.rechazar_mutacion_v1() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $f$
BEGIN RAISE EXCEPTION 'documentos: historia inmutable' USING ERRCODE='42501'; END $f$;
DO $inmutable$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['documento','preparacion_notificacion','auditoria_operacion','outbox'] LOOP
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_documentos.%I FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1()',t);
 END LOOP;
END $inmutable$;

-- La función interna no tiene concesión al ejecutor. El propio AD3 coteja
-- sesión, concesión vigente, firma COSE, gobierno de clave y consumo único.
CREATE FUNCTION vec_documentos.consumir_v3_v1(
 p_preimagen bytea,p_accion text,p_recurso text,p_principal text,p_perfil text,p_finalidad text,p_correlacion text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; v record; h text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_documentos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_documentos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR octet_length(p_preimagen) NOT BETWEEN 1 AND 16384
 THEN RAISE EXCEPTION 'documentos: contexto de ejecución denegado' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: autorización inválida' USING ERRCODE='42501'; END;
 -- Huella del contexto de recurso V3 documental: ámbitos vacíos y el único
 -- atributo preimagen_sha256 (la que emite el PDP real y recalcula Go en
 -- ports.HuellaEfectoV3). La decisión queda ligada a esta preimagen exacta.
 h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"organizacion:desarrollo:dipgra"},"atributos":{"preimagen_sha256":"'||encode(sha256(p_preimagen),'hex')||'"}}','UTF8')),'hex');
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.operacion.v1'
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
 THEN RAISE EXCEPTION 'documentos: decisión no ligada al efecto' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.efecto_ref IS DISTINCT FROM p_recurso OR v.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'documentos: consumo AD3 no ligado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT v.decision_ref,v.efecto_ref,v.huella_efecto_sha256,v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en,true;
END $f$;

CREATE FUNCTION vec_documentos.proyectar_documento_v1(p vec_documentos.documento)
RETURNS jsonb LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
 SELECT jsonb_build_object(
  'id',p.id,'numero_vec',p.numero_vec,'modulo_id',p.modulo_id,
  'expediente_ref',p.expediente_ref,'tipo_ref',p.tipo_ref,'version',p.version,
  'mime',p.mime,'huella_sha256',p.huella_sha256,'tamano',p.tamano,
  'objeto_ref',p.objeto_ref,'objeto_version',p.objeto_version,
  'politica_ref',p.politica_ref,'version_politica',p.version_politica,
  'huella_politica_sha256',p.huella_politica_sha256,
  'conservacion_hasta',p.conservacion_hasta,'proteccion',p.proteccion,
  'estado_politica',p.estado_politica,
  'estado_firma',p.estado_firma,'creado_en',p.creada_en)
 $f$;

CREATE FUNCTION vec_documentos.confirmar_alta_v1(
 p_preimagen bytea,p_objeto jsonb,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; d vec_documentos.documento%ROWTYPE; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text; nuevo boolean:=false;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
    OR p_auth IS NULL OR p_objeto IS NULL OR p_preimagen IS NULL
 THEN RAISE EXCEPTION 'documentos: alta denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1)
    IS DISTINCT FROM ARRAY['accion','clave_idempotencia','conservacion_hasta','estado_politica','expediente_ref','huella_politica_sha256','huella_sha256','id','mime','modulo_id','politica_ref','proteccion','tamano','tipo_ref','version','version_politica']
    OR m->>'accion'<>'documentos.generado.alta'
    OR m->>'id' IS DISTINCT FROM p_auth->>'recurso_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_auth->>'ambito_ref'
    OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'alta_documento_generado'
    OR p_objeto->>'mime' IS DISTINCT FROM m->>'mime'
    OR p_objeto->>'tamano' IS DISTINCT FROM m->>'tamano'
    OR p_objeto->>'huella_sha256' IS DISTINCT FROM m->>'huella_sha256'
    OR p_objeto->>'objeto_ref' IS NULL OR p_objeto->>'objeto_version' IS NULL
    OR p_objeto->>'conector_ref' IS NULL OR p_objeto->>'recibo_objeto_ref' IS NULL
    OR p_objeto->>'recibo_objeto_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_objeto->'inmovilizado') IS DISTINCT FROM 'boolean'
    OR jsonb_typeof(m->'estado_politica') IS DISTINCT FROM 'string'
    OR m->>'estado_politica' NOT IN ('aprobada','provisional')
    -- Aprobada: retención del proveedor hasta el plazo. Provisional: el
    -- conector no fijó retención ni inmovilizó el objeto (retenido_hasta null).
    OR (m->>'estado_politica'='aprobada' AND (jsonb_typeof(p_objeto->'retenido_hasta') IS DISTINCT FROM 'string'
        OR (p_objeto->>'retenido_hasta')::timestamptz < (m->>'conservacion_hasta')::timestamptz))
    OR (m->>'estado_politica'='provisional' AND (jsonb_typeof(p_objeto->'retenido_hasta') IS DISTINCT FROM 'null'
        OR p_objeto->>'inmovilizado' IS DISTINCT FROM 'false' OR m->>'proteccion' IS DISTINCT FROM 'conservacion'))
    OR (m->>'proteccion'='bloqueo' AND p_objeto->>'inmovilizado'<>'true')
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'huella_politica_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'proteccion' NOT IN ('conservacion','bloqueo')
 THEN RAISE EXCEPTION 'documentos: alta no ligada al objeto' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v1(
  p_preimagen,'documentos.generado.alta',p_auth->>'recurso_ref',p_auth->>'principal_id',
  p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'expediente_ref',true);
 h:=encode(sha256(p_preimagen),'hex');
 SELECT * INTO d FROM vec_documentos.documento
 WHERE principal_ref=p_auth->>'principal_id' AND clave_idempotencia=m->>'clave_idempotencia' FOR SHARE;
 IF FOUND THEN
  IF d.huella_preimagen_sha256 IS DISTINCT FROM h OR d.id IS DISTINCT FROM m->>'id'
     OR d.objeto_ref IS DISTINCT FROM p_objeto->>'objeto_ref'
     OR d.objeto_version IS DISTINCT FROM p_objeto->>'objeto_version'
     OR d.objeto_retenido_hasta IS DISTINCT FROM (p_objeto->>'retenido_hasta')::timestamptz
     OR d.objeto_inmovilizado IS DISTINCT FROM (p_objeto->>'inmovilizado')::boolean
     OR d.estado_politica IS DISTINCT FROM m->>'estado_politica'
  THEN RAISE EXCEPTION 'documentos: clave idempotente reutilizada' USING ERRCODE='23505'; END IF;
 ELSE
  INSERT INTO vec_documentos.documento(
    id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,mime,
    huella_sha256,tamano,objeto_ref,objeto_version,conector_ref,recibo_objeto_ref,recibo_objeto_huella_sha256,
    objeto_retenido_hasta,objeto_inmovilizado,
    politica_ref,version_politica,huella_politica_sha256,conservacion_hasta,proteccion,estado_politica,
    huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
  VALUES(m->>'id','VEC-'||to_char(ahora,'YYYY')||'-'||nextval('vec_documentos.numero_interno_seq')::text,
    m->>'clave_idempotencia',p_auth->>'principal_id',m->>'modulo_id',m->>'expediente_ref',m->>'tipo_ref',(m->>'version')::bigint,m->>'mime',
    m->>'huella_sha256',(m->>'tamano')::bigint,p_objeto->>'objeto_ref',p_objeto->>'objeto_version',
    p_objeto->>'conector_ref',p_objeto->>'recibo_objeto_ref',p_objeto->>'recibo_objeto_huella_sha256',
    (p_objeto->>'retenido_hasta')::timestamptz,(p_objeto->>'inmovilizado')::boolean,
    m->>'politica_ref',(m->>'version_politica')::bigint,m->>'huella_politica_sha256',(m->>'conservacion_hasta')::timestamptz,
    m->>'proteccion',m->>'estado_politica',h,v.decision_ref,v.auditoria_ref,ahora) RETURNING * INTO d;
  INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
   VALUES('documento_generado',d.id,d.expediente_ref,d.huella_sha256,ahora);
  nuevo:=true;
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.generado.alta',d.id,d.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,
  CASE WHEN nuevo THEN 'creado' ELSE 'repetido' END,ahora);
 RETURN vec_documentos.proyectar_documento_v1(d);
END $f$;

CREATE FUNCTION vec_documentos.listar_expediente_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; d vec_documentos.documento%ROWTYPE; resultado jsonb:='[]'::jsonb;
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
  SELECT creada_en,id INTO v_desde,v_cursor FROM vec_documentos.documento
   WHERE id=m->>'cursor' AND expediente_ref=m->>'expediente_ref';
  IF NOT FOUND THEN RAISE EXCEPTION 'documentos: cursor inválido' USING ERRCODE='22023'; END IF;
 END IF;
 FOR d IN SELECT * FROM vec_documentos.documento
  WHERE expediente_ref=m->>'expediente_ref' AND (creada_en,id)>(v_desde,v_cursor)
  ORDER BY creada_en,id LIMIT v_limite+1 LOOP
  v_total:=v_total+1;
  EXIT WHEN v_total>v_limite;
  resultado:=resultado||jsonb_build_array(vec_documentos.proyectar_documento_v1(d));
  v_cursor:=d.id;
 END LOOP;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.expediente.listar',m->>'expediente_ref',m->>'expediente_ref',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'autorizado',ahora);
 RETURN jsonb_build_object('items',resultado,'siguiente_cursor',CASE WHEN v_total>v_limite THEN v_cursor ELSE '' END);
END $f$;

CREATE FUNCTION vec_documentos.obtener_original_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; d vec_documentos.documento%ROWTYPE; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: consulta inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['accion','documento_id','version']
    OR m->>'accion'<>'documentos.original.descargar' OR m->>'documento_id' IS DISTINCT FROM p_auth->>'recurso_ref'
    OR NOT vec_documentos.referencia_opaca_v1(p_auth->>'ambito_ref') OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'descargar_documento_original'
 THEN RAISE EXCEPTION 'documentos: consulta denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v1(p_preimagen,'documentos.original.descargar',
  p_auth->>'recurso_ref',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',p_auth->>'ambito_ref',true);
 SELECT * INTO d FROM vec_documentos.documento WHERE id=m->>'documento_id' AND version=(m->>'version')::bigint AND expediente_ref=p_auth->>'ambito_ref';
 IF NOT FOUND THEN RAISE EXCEPTION 'documentos: original no disponible' USING ERRCODE='02000'; END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.original.descargar',d.id,d.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'autorizado',ahora);
 RETURN vec_documentos.proyectar_documento_v1(d);
END $f$;

CREATE FUNCTION vec_documentos.preparar_notificacion_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; n vec_documentos.preparacion_notificacion%ROWTYPE; d vec_documentos.documento%ROWTYPE; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text; nuevo boolean:=false;
BEGIN
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: preparación inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['accion','canal','clave_idempotencia','destinatario_ref','documento_id','id','version']
    OR m->>'accion'<>'documentos.notificacion.preparar' OR m->>'id' IS DISTINCT FROM p_auth->>'recurso_ref'
    OR NOT vec_documentos.referencia_opaca_v1(p_auth->>'ambito_ref') OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'preparar_notificacion'
 THEN RAISE EXCEPTION 'documentos: preparación denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v1(p_preimagen,'documentos.notificacion.preparar',
  p_auth->>'recurso_ref',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',p_auth->>'ambito_ref',true);
 SELECT * INTO d FROM vec_documentos.documento WHERE id=m->>'documento_id' AND version=(m->>'version')::bigint AND expediente_ref=p_auth->>'ambito_ref';
 IF NOT FOUND THEN RAISE EXCEPTION 'documentos: preparación sin documento' USING ERRCODE='02000'; END IF;
 h:=encode(sha256(p_preimagen),'hex');
 SELECT * INTO n FROM vec_documentos.preparacion_notificacion WHERE principal_ref=p_auth->>'principal_id' AND clave_idempotencia=m->>'clave_idempotencia' FOR SHARE;
 IF FOUND THEN
  IF n.huella_preimagen_sha256 IS DISTINCT FROM h OR n.id IS DISTINCT FROM m->>'id'
  THEN RAISE EXCEPTION 'documentos: clave idempotente reutilizada' USING ERRCODE='23505'; END IF;
 ELSE
  INSERT INTO vec_documentos.preparacion_notificacion(id,clave_idempotencia,documento_id,expediente_ref,version,destinatario_ref,canal,principal_ref,huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,preparada_en)
  VALUES(m->>'id',m->>'clave_idempotencia',d.id,d.expediente_ref,d.version,m->>'destinatario_ref',m->>'canal',p_auth->>'principal_id',h,v.decision_ref,v.auditoria_ref,ahora) RETURNING * INTO n;
  INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
  VALUES('notificacion_preparada',n.id,n.expediente_ref,h,ahora);
  nuevo:=true;
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.notificacion.preparar',n.id,n.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,
 CASE WHEN nuevo THEN 'preparado' ELSE 'repetido' END,ahora);
 RETURN jsonb_build_object('id',n.id,'documento_id',n.documento_id,'version',n.version,
  'destinatario_ref',n.destinatario_ref,'canal',n.canal,'preparada_en',n.preparada_en);
END $f$;

REVOKE ALL ON ALL TABLES IN SCHEMA vec_documentos FROM PUBLIC,vec_documentos_ejecutor;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA vec_documentos FROM PUBLIC,vec_documentos_ejecutor;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA vec_documentos FROM PUBLIC,vec_documentos_ejecutor;
-- confirmar_alta_v1 y preparar_notificacion_v1 no se conceden: el alta y la
-- preparación solo entran por sus versiones v2 (replay autorizado exacto de
-- 000002). Se conservan porque 000002 comprueba su presencia como preimagen.
GRANT EXECUTE ON FUNCTION vec_documentos.listar_expediente_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.obtener_original_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_ejecutor;
COMMIT;
