\set ON_ERROR_STOP on
-- Documentos 13: reserva durable del original firmable antes de escribir bytes.
-- AD3-158 habilita las dos operaciones en el consumidor V3 documental.
-- La reserva, los intentos y la confirmación son historia de solo adición.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000013',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regprocedure('vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_documentos.confirmar_alta_v2(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_documentos.principal_ref_v1(text)') IS NULL
    OR to_regclass('vec_documentos.reserva_original_firmable') IS NOT NULL
    OR to_regclass('vec_documentos.intento_original_firmable') IS NOT NULL
    OR to_regclass('vec_documentos.confirmacion_original_firmable') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR strpos(pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'documentos.original_firmable.reservar')=0
    OR strpos(pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'documentos.original_firmable.confirmar')=0
 THEN RAISE EXCEPTION 'Documentos-13: preimagen o dependencia ausente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_documentos.reserva_original_firmable (
 reserva_ref text PRIMARY KEY CHECK(vec_documentos.referencia_opaca_v1(reserva_ref)),
 documento_id text NOT NULL UNIQUE CHECK(documento_id ~ '^ref:[0-9a-f]{64}$' AND documento_id<>'ref:'||repeat('0',64)),
 clave_idempotencia text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(clave_idempotencia)),
 modulo_id text NOT NULL CHECK(modulo_id ~ '^[a-z][a-z0-9_]{2,63}$'),
 expediente_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(expediente_ref)),
 tipo_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(tipo_ref)),
 version bigint NOT NULL CHECK(version>0),
 mime text NOT NULL CHECK(mime='application/pdf'),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 tamano bigint NOT NULL CHECK(tamano>0),
 politica_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(politica_ref)),
 version_politica bigint NOT NULL CHECK(version_politica>0),
 huella_politica_sha256 text NOT NULL CHECK(huella_politica_sha256 ~ '^[0-9a-f]{64}$'),
 conservacion_hasta timestamptz(6) NOT NULL,
 proteccion text NOT NULL CHECK(proteccion IN ('conservacion','bloqueo')),
 estado_politica text NOT NULL CHECK(estado_politica IN ('aprobada','provisional')),
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 principal_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(principal_ref)),
 reservada_en timestamptz(6) NOT NULL,
 UNIQUE(modulo_id,expediente_ref,tipo_ref,version),
 UNIQUE(principal_ref,clave_idempotencia)
);
CREATE TABLE vec_documentos.intento_original_firmable (
 reserva_ref text NOT NULL REFERENCES vec_documentos.reserva_original_firmable(reserva_ref),
 intento_num bigint NOT NULL CHECK(intento_num>0),
 clave_almacen_ref text NOT NULL UNIQUE CHECK(vec_documentos.referencia_opaca_v1(clave_almacen_ref)),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ad3_ref text NOT NULL,
 principal_ref text NOT NULL,
 expediente_ref text NOT NULL,
 creada_en timestamptz(6) NOT NULL,
 PRIMARY KEY(reserva_ref,intento_num)
);
CREATE TABLE vec_documentos.confirmacion_original_firmable (
 reserva_ref text PRIMARY KEY REFERENCES vec_documentos.reserva_original_firmable(reserva_ref),
 documento_id text NOT NULL UNIQUE REFERENCES vec_documentos.documento(id),
 intento_num bigint NOT NULL,
 clave_almacen_ref text NOT NULL,
 huella_preimagen_sha256 text NOT NULL CHECK(huella_preimagen_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ad3_ref text NOT NULL,
 expediente_ref text NOT NULL,
 confirmada_en timestamptz(6) NOT NULL,
 FOREIGN KEY(reserva_ref,intento_num) REFERENCES vec_documentos.intento_original_firmable(reserva_ref,intento_num)
);
DO $seguridad$
DECLARE t text; a record;
BEGIN
 FOREACH t IN ARRAY ARRAY['reserva_original_firmable','intento_original_firmable','confirmacion_original_firmable'] LOOP
  EXECUTE format('ALTER TABLE vec_documentos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_documentos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_documentos.%I FROM PUBLIC, vec_documentos_ejecutor',t);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c
    CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
    WHERE c.oid=to_regclass('vec_documentos.'||t) AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_documentos.%I FROM %I',t,pg_get_userbyid(a.grantee));
  END LOOP;
  EXECUTE format('REVOKE ALL ON TYPE vec_documentos.%I FROM PUBLIC',t);
  EXECUTE format('CREATE POLICY alcance_lectura ON vec_documentos.%I FOR SELECT TO vec_documentos_propietario USING (expediente_ref = nullif(current_setting(''vec.documentos.expediente_ref'',true),''''))',t);
  EXECUTE format('CREATE POLICY alcance_alta ON vec_documentos.%I FOR INSERT TO vec_documentos_propietario WITH CHECK (expediente_ref = nullif(current_setting(''vec.documentos.expediente_ref'',true),''''))',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_documentos.%I FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1()',t);
 END LOOP;
END $seguridad$;

-- Preserva literalmente las acciones ya autorizadas por migraciones anteriores;
-- falla si la restricción deja de ser una lista positiva de acciones.
DO $auditoria$
DECLARE anterior text; condicion text;
BEGIN
 SELECT pg_get_constraintdef(oid) INTO STRICT anterior FROM pg_constraint
  WHERE conrelid='vec_documentos.auditoria_operacion'::regclass
    AND conname='auditoria_operacion_accion_check';
 IF anterior !~ '^CHECK \(\(accion = ANY \(ARRAY\[' OR anterior !~ '\]\)\)\)$'
 THEN RAISE EXCEPTION 'Documentos-13: auditoría incompatible' USING ERRCODE='55000'; END IF;
 condicion:=substring(anterior FROM 8 FOR length(anterior)-8);
 EXECUTE 'ALTER TABLE vec_documentos.auditoria_operacion DROP CONSTRAINT auditoria_operacion_accion_check';
 EXECUTE 'ALTER TABLE vec_documentos.auditoria_operacion ADD CONSTRAINT auditoria_operacion_accion_check CHECK (('
     ||condicion||') OR accion IN (''documentos.original_firmable.reservar'',''documentos.original_firmable.confirmar''))';
END $auditoria$;

CREATE FUNCTION vec_documentos.reservar_original_firmable_v1(
 p_preimagen bytea,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; r vec_documentos.reserva_original_firmable%ROWTYPE;
 i vec_documentos.intento_original_firmable%ROWTYPE; c vec_documentos.confirmacion_original_firmable%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text; n bigint; nuevo boolean:=false;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR p_auth IS NULL OR p_preimagen IS NULL
 THEN RAISE EXCEPTION 'documentos: reserva denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
    ARRAY['accion','clave_idempotencia','conservacion_hasta','estado_politica','expediente_ref','huella_politica_sha256','huella_sha256','id','mime','modulo_id','politica_ref','proteccion','tamano','tipo_ref','version','version_politica']
    OR m->>'accion' IS DISTINCT FROM 'documentos.original_firmable.reservar'
    OR m->>'mime' IS DISTINCT FROM 'application/pdf'
    OR m->>'id' !~ '^ref:[0-9a-f]{64}$' OR m->>'id'='ref:'||repeat('0',64)
    OR NOT vec_documentos.referencia_opaca_v1(m->>'clave_idempotencia')
    OR NOT vec_documentos.referencia_opaca_v1(m->>'expediente_ref')
    OR NOT vec_documentos.referencia_opaca_v1(m->>'tipo_ref')
    OR NOT vec_documentos.referencia_opaca_v1(m->>'politica_ref')
    OR m->>'modulo_id' !~ '^[a-z][a-z0-9_]{2,63}$'
    OR jsonb_typeof(m->'version') IS DISTINCT FROM 'number' OR (m->>'version')::numeric NOT BETWEEN 1 AND 9223372036854775807
    OR (m->>'version')::numeric<>trunc((m->>'version')::numeric)
    OR jsonb_typeof(m->'tamano') IS DISTINCT FROM 'number' OR (m->>'tamano')::numeric NOT BETWEEN 1 AND 9223372036854775807
    OR (m->>'tamano')::numeric<>trunc((m->>'tamano')::numeric)
    OR jsonb_typeof(m->'version_politica') IS DISTINCT FROM 'number' OR (m->>'version_politica')::numeric NOT BETWEEN 1 AND 9223372036854775807
    OR (m->>'version_politica')::numeric<>trunc((m->>'version_politica')::numeric)
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR m->>'huella_politica_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'estado_politica' NOT IN ('aprobada','provisional')
    OR m->>'proteccion' NOT IN ('conservacion','bloqueo')
    OR jsonb_typeof(m->'conservacion_hasta') IS DISTINCT FROM 'string'
    OR ARRAY(SELECT jsonb_object_keys(p_auth) ORDER BY 1) IS DISTINCT FROM
      ARRAY['accion','ambito_ref','correlacion_ref','finalidad','perfil_activo_ref','principal_id','recurso_ref']
    OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_original_firmable'
    OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'id'
    OR p_auth->>'ambito_ref' IS DISTINCT FROM m->>'expediente_ref'
    OR NOT vec_documentos.principal_ref_v1(p_auth->>'principal_id')
 THEN RAISE EXCEPTION 'documentos: reserva no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v2(
  p_preimagen,m->>'accion',m->>'id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'expediente_ref',true);
 h:=encode(sha256(p_preimagen),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:original_firmable:'||m->>'modulo_id'||':'||m->>'expediente_ref'||':'||m->>'tipo_ref'||':'||m->>'version',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:original_firmable:documento:'||m->>'id',0));
 SELECT * INTO r FROM vec_documentos.reserva_original_firmable
  WHERE modulo_id=m->>'modulo_id' AND expediente_ref=m->>'expediente_ref'
    AND tipo_ref=m->>'tipo_ref' AND version=(m->>'version')::bigint;
 IF FOUND THEN
  -- La fecha de retención calculada al reintentar puede ser posterior.
  -- Cotejar todo el material estable; nunca acortar la conservación fijada.
  IF r.documento_id IS DISTINCT FROM m->>'id'
     OR r.clave_idempotencia IS DISTINCT FROM m->>'clave_idempotencia'
     OR r.principal_ref IS DISTINCT FROM p_auth->>'principal_id'
     OR r.mime IS DISTINCT FROM m->>'mime'
     OR r.huella_sha256 IS DISTINCT FROM m->>'huella_sha256'
     OR r.tamano::text IS DISTINCT FROM m->>'tamano'
     OR r.politica_ref IS DISTINCT FROM m->>'politica_ref'
     OR r.version_politica::text IS DISTINCT FROM m->>'version_politica'
     OR r.huella_politica_sha256 IS DISTINCT FROM m->>'huella_politica_sha256'
     OR r.proteccion IS DISTINCT FROM m->>'proteccion'
     OR r.estado_politica IS DISTINCT FROM m->>'estado_politica'
     OR (m->>'conservacion_hasta')::timestamptz<r.conservacion_hasta
     OR (v.consumo_nuevo IS FALSE AND r.huella_preimagen_sha256 IS DISTINCT FROM h)
  THEN RAISE EXCEPTION 'documentos: original divergente' USING ERRCODE='23505'; END IF;
 ELSE
  IF v.consumo_nuevo IS FALSE THEN RAISE EXCEPTION 'documentos: replay sin reserva' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_documentos.reserva_original_firmable VALUES (
   'ref:'||encode(sha256(convert_to('vec-documentos-original-firmable-v1:'||m->>'id','UTF8')),'hex'),
   m->>'id',m->>'clave_idempotencia',m->>'modulo_id',m->>'expediente_ref',m->>'tipo_ref',
   (m->>'version')::bigint,m->>'mime',m->>'huella_sha256',(m->>'tamano')::bigint,
   m->>'politica_ref',(m->>'version_politica')::bigint,m->>'huella_politica_sha256',
   (m->>'conservacion_hasta')::timestamptz,m->>'proteccion',m->>'estado_politica',h,
   p_auth->>'principal_id',ahora) RETURNING * INTO r;
 END IF;
 SELECT * INTO c FROM vec_documentos.confirmacion_original_firmable WHERE reserva_ref=r.reserva_ref;
 IF FOUND THEN
  INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
  VALUES(m->>'accion',r.documento_id,r.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
   p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,'repetido',ahora);
  RETURN jsonb_build_object('reserva_ref',r.reserva_ref,'estado','confirmado','id',r.documento_id,
    'huella_sha256',r.huella_sha256,'intento_num',c.intento_num,'clave_almacen_ref',c.clave_almacen_ref);
 END IF;
 SELECT * INTO i FROM vec_documentos.intento_original_firmable WHERE decision_ref=v.decision_ref;
 IF FOUND THEN
  IF i.reserva_ref IS DISTINCT FROM r.reserva_ref OR v.consumo_nuevo IS NOT FALSE
  THEN RAISE EXCEPTION 'documentos: decisión de intento reutilizada' USING ERRCODE='23505'; END IF;
 ELSE
  IF v.consumo_nuevo IS FALSE THEN RAISE EXCEPTION 'documentos: replay sin intento' USING ERRCODE='42501'; END IF;
  SELECT coalesce(max(intento_num),0)+1 INTO n FROM vec_documentos.intento_original_firmable WHERE reserva_ref=r.reserva_ref;
  INSERT INTO vec_documentos.intento_original_firmable VALUES (
    r.reserva_ref,n,
    'ref:'||encode(sha256(convert_to(r.reserva_ref||':'||n::text||':'||v.decision_ref,'UTF8')),'hex'),
    v.decision_ref,v.auditoria_ref,p_auth->>'principal_id',r.expediente_ref,ahora) RETURNING * INTO i;
  nuevo:=true;
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',r.documento_id,r.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,
  CASE WHEN nuevo THEN 'creado' ELSE 'repetido' END,ahora);
 RETURN jsonb_build_object('reserva_ref',r.reserva_ref,'estado','pendiente','id',r.documento_id,
  'huella_sha256',r.huella_sha256,'intento_num',i.intento_num,'clave_almacen_ref',i.clave_almacen_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.reservar_original_firmable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.reservar_original_firmable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

CREATE FUNCTION vec_documentos.confirmar_original_firmable_v1(
 p_preimagen bytea,p_objeto jsonb,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; r vec_documentos.reserva_original_firmable%ROWTYPE;
 i vec_documentos.intento_original_firmable%ROWTYPE; c vec_documentos.confirmacion_original_firmable%ROWTYPE;
 d vec_documentos.documento%ROWTYPE; ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
 h text; ultimo bigint; nuevo boolean:=false;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user OR p_auth IS NULL OR p_objeto IS NULL OR p_preimagen IS NULL
 THEN RAISE EXCEPTION 'documentos: confirmación denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
    ARRAY['accion','clave_almacen_ref','huella_sha256','id','intento_num','objeto','reserva_ref']
    OR m->>'accion' IS DISTINCT FROM 'documentos.original_firmable.confirmar'
    OR m->>'id' !~ '^ref:[0-9a-f]{64}$' OR m->>'id'='ref:'||repeat('0',64)
    OR NOT vec_documentos.referencia_opaca_v1(m->>'reserva_ref')
    OR NOT vec_documentos.referencia_opaca_v1(m->>'clave_almacen_ref')
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(m->'intento_num') IS DISTINCT FROM 'number'
    OR (m->>'intento_num')::numeric NOT BETWEEN 1 AND 9223372036854775807
    OR (m->>'intento_num')::numeric<>trunc((m->>'intento_num')::numeric)
    OR m->'objeto' IS DISTINCT FROM p_objeto
    OR jsonb_typeof(p_objeto)<>'object' OR ARRAY(SELECT jsonb_object_keys(p_objeto) ORDER BY 1) IS DISTINCT FROM
      ARRAY['clave_almacen_ref','conector_ref','huella_sha256','inmovilizado','mime','objeto_ref','objeto_version','recibo_objeto_huella_sha256','recibo_objeto_ref','retenido_hasta','tamano']
    OR p_objeto->>'clave_almacen_ref' IS DISTINCT FROM m->>'clave_almacen_ref'
    OR p_objeto->>'huella_sha256' IS DISTINCT FROM m->>'huella_sha256'
    OR p_objeto->>'recibo_objeto_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR p_objeto->>'objeto_ref' IS NULL OR p_objeto->>'objeto_version' IS NULL
    OR p_objeto->>'conector_ref' IS NULL OR p_objeto->>'recibo_objeto_ref' IS NULL
    OR jsonb_typeof(p_objeto->'inmovilizado') IS DISTINCT FROM 'boolean'
    OR ARRAY(SELECT jsonb_object_keys(p_auth) ORDER BY 1) IS DISTINCT FROM
      ARRAY['accion','ambito_ref','correlacion_ref','finalidad','perfil_activo_ref','principal_id','recurso_ref']
    OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_original_firmable'
    OR p_auth->>'recurso_ref' IS DISTINCT FROM m->>'id'
    OR NOT vec_documentos.principal_ref_v1(p_auth->>'principal_id')
 THEN RAISE EXCEPTION 'documentos: confirmación no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v2(
  p_preimagen,m->>'accion',m->>'id',p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',p_auth->>'ambito_ref',true);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:original_firmable:documento:'||m->>'id',0));
 SELECT * INTO r FROM vec_documentos.reserva_original_firmable WHERE reserva_ref=m->>'reserva_ref';
 IF NOT FOUND OR r.documento_id IS DISTINCT FROM m->>'id'
    OR r.expediente_ref IS DISTINCT FROM p_auth->>'ambito_ref'
    OR r.principal_ref IS DISTINCT FROM p_auth->>'principal_id'
    OR r.huella_sha256 IS DISTINCT FROM m->>'huella_sha256'
    OR r.mime IS DISTINCT FROM p_objeto->>'mime'
    OR r.tamano::text IS DISTINCT FROM p_objeto->>'tamano'
 THEN RAISE EXCEPTION 'documentos: reserva no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO i FROM vec_documentos.intento_original_firmable
  WHERE reserva_ref=r.reserva_ref AND intento_num=(m->>'intento_num')::bigint;
 IF NOT FOUND OR i.clave_almacen_ref IS DISTINCT FROM m->>'clave_almacen_ref'
 THEN RAISE EXCEPTION 'documentos: intento no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO c FROM vec_documentos.confirmacion_original_firmable WHERE reserva_ref=r.reserva_ref;
 h:=encode(sha256(p_preimagen),'hex');
 IF FOUND THEN
  SELECT * INTO d FROM vec_documentos.documento WHERE id=c.documento_id;
  IF NOT FOUND OR c.intento_num IS DISTINCT FROM i.intento_num
     OR c.clave_almacen_ref IS DISTINCT FROM i.clave_almacen_ref
     OR c.huella_preimagen_sha256 IS DISTINCT FROM h
     OR d.huella_preimagen_sha256 IS DISTINCT FROM h
     OR d.huella_sha256 IS DISTINCT FROM r.huella_sha256
     OR d.recibo_objeto_ref IS DISTINCT FROM p_objeto->>'recibo_objeto_ref'
     OR d.recibo_objeto_huella_sha256 IS DISTINCT FROM p_objeto->>'recibo_objeto_huella_sha256'
     OR d.objeto_ref IS DISTINCT FROM p_objeto->>'objeto_ref'
     OR d.objeto_version IS DISTINCT FROM p_objeto->>'objeto_version'
     OR d.conector_ref IS DISTINCT FROM p_objeto->>'conector_ref'
     OR d.objeto_retenido_hasta IS DISTINCT FROM (p_objeto->>'retenido_hasta')::timestamptz
     OR d.objeto_inmovilizado IS DISTINCT FROM (p_objeto->>'inmovilizado')::boolean
     OR (v.consumo_nuevo IS FALSE AND (c.decision_ref IS DISTINCT FROM v.decision_ref OR c.auditoria_ad3_ref IS DISTINCT FROM v.auditoria_ref))
  THEN RAISE EXCEPTION 'documentos: confirmación divergente' USING ERRCODE='23505'; END IF;
 ELSE
  IF v.consumo_nuevo IS FALSE THEN RAISE EXCEPTION 'documentos: replay sin confirmación' USING ERRCODE='42501'; END IF;
  SELECT max(intento_num) INTO ultimo FROM vec_documentos.intento_original_firmable WHERE reserva_ref=r.reserva_ref;
  IF ultimo IS DISTINCT FROM i.intento_num THEN RAISE EXCEPTION 'documentos: intento sustituido' USING ERRCODE='23505'; END IF;
  IF (r.estado_politica='aprobada' AND (jsonb_typeof(p_objeto->'retenido_hasta') IS DISTINCT FROM 'string'
       OR (p_objeto->>'retenido_hasta')::timestamptz<r.conservacion_hasta))
     OR (r.estado_politica='provisional' AND (jsonb_typeof(p_objeto->'retenido_hasta') IS DISTINCT FROM 'null'
       OR p_objeto->>'inmovilizado' IS DISTINCT FROM 'false' OR r.proteccion IS DISTINCT FROM 'conservacion'))
     OR (r.proteccion='bloqueo' AND p_objeto->>'inmovilizado'<>'true')
  THEN RAISE EXCEPTION 'documentos: conservación no acreditada' USING ERRCODE='42501'; END IF;
  IF EXISTS(SELECT 1 FROM vec_documentos.documento WHERE id=r.documento_id
      OR (modulo_id=r.modulo_id AND expediente_ref=r.expediente_ref AND tipo_ref=r.tipo_ref AND version=r.version))
  THEN RAISE EXCEPTION 'documentos: alta previa sin reserva confirmada' USING ERRCODE='23505'; END IF;
  INSERT INTO vec_documentos.documento(
    id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,mime,
    huella_sha256,tamano,objeto_ref,objeto_version,conector_ref,recibo_objeto_ref,recibo_objeto_huella_sha256,
    objeto_retenido_hasta,objeto_inmovilizado,politica_ref,version_politica,huella_politica_sha256,
    conservacion_hasta,proteccion,estado_politica,huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
  VALUES(r.documento_id,'VEC-'||to_char(ahora,'YYYY')||'-'||nextval('vec_documentos.numero_interno_seq')::text,
    r.clave_idempotencia,r.principal_ref,r.modulo_id,r.expediente_ref,r.tipo_ref,r.version,r.mime,
    r.huella_sha256,r.tamano,p_objeto->>'objeto_ref',p_objeto->>'objeto_version',p_objeto->>'conector_ref',
    p_objeto->>'recibo_objeto_ref',p_objeto->>'recibo_objeto_huella_sha256',
    (p_objeto->>'retenido_hasta')::timestamptz,(p_objeto->>'inmovilizado')::boolean,
    r.politica_ref,r.version_politica,r.huella_politica_sha256,r.conservacion_hasta,r.proteccion,r.estado_politica,
    h,v.decision_ref,v.auditoria_ref,ahora) RETURNING * INTO d;
  INSERT INTO vec_documentos.confirmacion_original_firmable VALUES(
    r.reserva_ref,d.id,i.intento_num,i.clave_almacen_ref,h,v.decision_ref,v.auditoria_ref,r.expediente_ref,ahora);
  INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
   VALUES('documento_generado',d.id,d.expediente_ref,d.huella_sha256,ahora);
  nuevo:=true;
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES(m->>'accion',d.id,d.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,
  CASE WHEN nuevo THEN 'creado' ELSE 'repetido' END,ahora);
 RETURN vec_documentos.proyectar_documento_v1(d) || jsonb_build_object(
  'reserva_ref',r.reserva_ref,'estado','confirmado','intento_num',i.intento_num,
  'clave_almacen_ref',i.clave_almacen_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.confirmar_original_firmable_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.confirmar_original_firmable_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;

-- Una vez reservada la identidad, ningún alta genérica puede saltarse su
-- confirmación. Es diferida para admitir documento y confirmación en una TX.
CREATE FUNCTION vec_documentos.exigir_confirmacion_original_firmable_v1() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET row_security=on AS $f$
DECLARE anterior text:=current_setting('vec.documentos.expediente_ref',true); r text;
BEGIN
 PERFORM set_config('vec.documentos.expediente_ref',NEW.expediente_ref,true);
 SELECT reserva_ref INTO r FROM vec_documentos.reserva_original_firmable
  WHERE documento_id=NEW.id OR (modulo_id=NEW.modulo_id AND expediente_ref=NEW.expediente_ref
    AND tipo_ref=NEW.tipo_ref AND version=NEW.version);
 IF r IS NOT NULL AND NOT EXISTS(
    SELECT 1 FROM vec_documentos.confirmacion_original_firmable c
    WHERE c.reserva_ref=r AND c.documento_id=NEW.id)
 THEN RAISE EXCEPTION 'documentos: original reservado sin confirmación' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.documentos.expediente_ref',coalesce(anterior,''),true);
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.exigir_confirmacion_original_firmable_v1() FROM PUBLIC;
CREATE CONSTRAINT TRIGGER exigir_confirmacion_original_firmable AFTER INSERT ON vec_documentos.documento
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION vec_documentos.exigir_confirmacion_original_firmable_v1();
COMMIT;
