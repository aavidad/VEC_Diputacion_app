\set ON_ERROR_STOP on
-- Documentos-6: la repetición exacta de un registro externo devuelve el
-- recibo original aunque llegue con otra concesión V3.
--
-- La ruta HTTP pide una concesión nueva en cada petición y la política de
-- conservación calcula conservacion_hasta como «ahora + plazo». La preimagen,
-- y con ella su huella, cambia por tanto en cada repetición, y 000003 exigía
-- huella_preimagen_sha256 idéntica: el segundo POST respondía 409 conflicto.
--
-- Ahora, con consumo nuevo, se coteja el material semántico campo a campo
-- (identificador, clave, módulo, expediente, tipo, versión, MIME, tamaño,
-- huella, custodio, referencia, política, protección y estado) y se admite una
-- fecha de conservación igual o posterior a la registrada, que es lo que da
-- resolver la misma política más tarde. Cualquier otro cambio sigue siendo
-- 23505. El replay AD3 de la misma concesión (consumo_nuevo false) conserva
-- la comparación exacta de decisión, auditoría y preimagen. No reescribe
-- filas ni cambia firmas, ACL ni huellas de 000004; conserva 000001–000005.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000006',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regprocedure('vec_documentos.principal_ref_v1(text)') IS NULL
    OR to_regprocedure('vec_documentos.registro_externo_equivalente_v1(vec_documentos.referencia_externa,jsonb)') IS NOT NULL
    -- Cuerpo exacto instalado por 000003 (SHA-256 de prosrc en UTF-8).
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=
        'vec_documentos.registrar_referencia_externa_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       IS DISTINCT FROM 'c151333d0bdf94570846c94b225393af037afb9ca9e3cd8cb2e734cbc0431111'
 THEN RAISE EXCEPTION 'Documentos-6: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Un registro existente e y una preimagen m nombran el mismo efecto: mismo
-- material y una conservación que no retrocede.
CREATE FUNCTION vec_documentos.registro_externo_equivalente_v1(e vec_documentos.referencia_externa,m jsonb) RETURNS boolean
LANGUAGE sql STABLE SET search_path=pg_catalog SET timezone='UTC' AS $f$
 SELECT coalesce(
  e.id=m->>'id' AND e.clave_idempotencia=m->>'clave_idempotencia'
  AND e.modulo_id=m->>'modulo_id' AND e.expediente_ref=m->>'expediente_ref'
  AND e.tipo_ref=m->>'tipo_ref' AND e.version::text=m->>'version'
  AND coalesce(e.mime,'')=m->>'mime' AND coalesce(e.tamano,0)::text=m->>'tamano'
  AND e.huella_sha256=m->>'huella_sha256' AND e.custodio_id=m->>'custodio_id'
  AND e.custodia_ref=m->>'custodia_ref' AND e.politica_ref=m->>'politica_ref'
  AND e.version_politica::text=m->>'version_politica'
  AND e.huella_politica_sha256=m->>'huella_politica_sha256'
  AND e.proteccion=m->>'proteccion' AND e.estado_politica=m->>'estado_politica'
  AND (m->>'conservacion_hasta')::timestamptz>=e.conservacion_hasta,false)
 $f$;
REVOKE ALL ON FUNCTION vec_documentos.registro_externo_equivalente_v1(vec_documentos.referencia_externa,jsonb) FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.registrar_referencia_externa_v1(
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
    IS DISTINCT FROM ARRAY['accion','clave_idempotencia','conservacion_hasta','custodia_ref','custodio_id','estado_politica','expediente_ref','huella_politica_sha256','huella_sha256','id','mime','modulo_id','politica_ref','proteccion','tamano','tipo_ref','version','version_politica']
    OR m->>'accion'<>'documentos.externo.registrar'
    OR m->>'id' IS DISTINCT FROM p_auth->>'recurso_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_auth->>'ambito_ref'
    OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'registrar_documento_externo'
    OR jsonb_typeof(m->'tamano')<>'number' OR jsonb_typeof(m->'version')<>'number' OR jsonb_typeof(m->'version_politica')<>'number'
    OR jsonb_typeof(m->'mime')<>'string'
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR m->>'huella_politica_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'proteccion' NOT IN ('conservacion','bloqueo')
    OR jsonb_typeof(m->'estado_politica') IS DISTINCT FROM 'string'
    OR m->>'estado_politica' NOT IN ('aprobada','provisional')
    OR (m->>'estado_politica'='provisional' AND m->>'proteccion'<>'conservacion')
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
  -- Replay AD3 de la misma concesión: todo idéntico. Concesión nueva: mismo
  -- material semántico; conservacion_hasta puede ser posterior.
  IF (v.consumo_nuevo IS FALSE AND (e.decision_ref IS DISTINCT FROM v.decision_ref OR e.auditoria_ad3_ref IS DISTINCT FROM v.auditoria_ref
       OR e.huella_preimagen_sha256 IS DISTINCT FROM h))
     OR NOT vec_documentos.registro_externo_equivalente_v1(e,m)
  THEN RAISE EXCEPTION 'documentos: clave idempotente reutilizada' USING ERRCODE='23505'; END IF;
 ELSE
  INSERT INTO vec_documentos.referencia_externa(
    id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,
    huella_sha256,mime,tamano,custodio_id,custodia_ref,
    politica_ref,version_politica,huella_politica_sha256,conservacion_hasta,proteccion,estado_politica,
    huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
  VALUES(m->>'id','VEC-'||to_char(ahora,'YYYY')||'-'||nextval('vec_documentos.numero_interno_seq')::text,
    m->>'clave_idempotencia',p_auth->>'principal_id',m->>'modulo_id',m->>'expediente_ref',m->>'tipo_ref',(m->>'version')::bigint,
    m->>'huella_sha256',nullif(m->>'mime',''),nullif((m->>'tamano')::bigint,0),m->>'custodio_id',m->>'custodia_ref',
    m->>'politica_ref',(m->>'version_politica')::bigint,m->>'huella_politica_sha256',(m->>'conservacion_hasta')::timestamptz,
    m->>'proteccion',m->>'estado_politica',h,v.decision_ref,v.auditoria_ref,ahora) RETURNING * INTO e;
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
COMMIT;
