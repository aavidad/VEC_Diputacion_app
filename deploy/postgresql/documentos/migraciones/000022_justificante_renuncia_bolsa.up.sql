\set ON_ERROR_STOP on
-- D22: custodia local del justificante de renuncia de Bolsa sobre D20 final.
-- No instala D21 ni acepta Dietas/Cronos. Requiere el consumidor AD277;
-- B119 aporta la validación del dueño antes de activar las fachadas.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000022',0));
DO $pre$
DECLARE f record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_documentos_bolsa_subida_datos') IS NOT NULL
 OR to_regrole('vec_documentos_bolsa_rrhh') IS NOT NULL
 OR to_regprocedure('vec_documentos.selector_subida_local_v1()') IS NOT NULL
 OR to_regprocedure('vec_documentos.consultar_justificante_renuncia_rrhh_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_documentos.validar_justificante_renuncia_bolsa_v1(text,text,text,text,text)') IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE
   p.oid=to_regprocedure('vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v'
   AND has_function_privilege('vec_documentos_propietario',p.oid,'EXECUTE')
   AND NOT has_function_privilege('public',p.oid,'EXECUTE'))
 THEN RAISE EXCEPTION 'D22: preimagen o consumidor incompatibles' USING ERRCODE='55000'; END IF;
 FOR f IN SELECT * FROM (VALUES
  ('subida_local_login_valido_v1(boolean)','b367587c7452e1789c5d3d27867aed8fa73d8e454c32dca046506f064b24fede'),
  ('consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','42402a4e446879f340123e3ce522cf7f70ecc76f681f784055d2bb33bf3b6619'),
  ('operar_subida_local_interna_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','84e7267231b2ba023d9ff7806fc8d79537af1ecf179dadc31a258ee48349c901'),
  ('instantanea_subida_local_v1(text)','7e5d217407b96d323da25d2b14fcc18ca5c5471b7761472f7ce6c8130688d1e2'),
  ('verificar_integridad_subida_local_v1(text)','28ab537a6480cec3a535f03e8732e1e8b9cf4fa0633c781691383480d900cc37')
 ) AS x(firma,huella) LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=to_regprocedure('vec_documentos.'||f.firma)
   AND p.proowner='vec_documentos_propietario'::regrole AND p.prosecdef
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')=f.huella)
  THEN RAISE EXCEPTION 'D22: cuerpo D20 divergente: %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.subida_local'::regclass
  AND conname='subida_local_modulo_id_check' AND convalidated
  AND pg_get_constraintdef(oid,false)='CHECK ((modulo_id = ''meritos''::text))')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.subida_local'::regclass
  AND conname='subida_local_tipo_recurso_check' AND convalidated
  AND pg_get_constraintdef(oid,false)='CHECK ((tipo_recurso = ''hecho''::text))')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.subida_local'::regclass
  AND conname='subida_local_finalidad_check' AND convalidated
  AND pg_get_constraintdef(oid,false)='CHECK ((finalidad = ''custodiar_evidencia_meritos''::text))')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_documentos.subida_local_lectura'::regclass
  AND conname='subida_local_lectura_fase_check' AND convalidated
  AND pg_get_constraintdef(oid,false)='CHECK ((fase = ANY (ARRAY[''consulta_carga''::text, ''consulta_documento''::text, ''abrir''::text])))')
 THEN RAISE EXCEPTION 'D22: restricciones D20 divergentes' USING ERRCODE='55000'; END IF;
END $pre$;

-- Los LOGIN nominales se crean y se acreditan en el guion de activación.
-- Cada uno hereda un solo grupo, sin SET ni ADMIN y sin secreto en Git.
CREATE ROLE vec_documentos_bolsa_subida_datos NOLOGIN INHERIT NOSUPERUSER
 NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_documentos_bolsa_rrhh NOLOGIN INHERIT NOSUPERUSER
 NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_documentos_propietario;
GRANT USAGE ON SCHEMA vec_documentos TO vec_documentos_bolsa_subida_datos,vec_documentos_bolsa_rrhh;
GRANT USAGE ON SCHEMA vec_documentos TO vec_bolsa_llamamientos_propietario;

ALTER TABLE vec_documentos.subida_local DROP CONSTRAINT subida_local_modulo_id_check;
ALTER TABLE vec_documentos.subida_local DROP CONSTRAINT subida_local_tipo_recurso_check;
ALTER TABLE vec_documentos.subida_local DROP CONSTRAINT subida_local_finalidad_check;
ALTER TABLE vec_documentos.subida_local ADD CONSTRAINT subida_local_modulo_id_check
 CHECK(modulo_id IN ('meritos','bolsa'));
ALTER TABLE vec_documentos.subida_local ADD CONSTRAINT subida_local_tipo_recurso_check
 CHECK((modulo_id='meritos' AND tipo_recurso='hecho') OR
  (modulo_id='bolsa' AND tipo_recurso='justificante_renuncia'));
ALTER TABLE vec_documentos.subida_local ADD CONSTRAINT subida_local_finalidad_check
 CHECK((modulo_id='meritos' AND finalidad='custodiar_evidencia_meritos') OR
  (modulo_id='bolsa' AND finalidad='custodiar_justificante_renuncia_bolsa'));
ALTER TABLE vec_documentos.subida_local_lectura DROP CONSTRAINT subida_local_lectura_fase_check;
ALTER TABLE vec_documentos.subida_local_lectura ADD CONSTRAINT subida_local_lectura_fase_check
 CHECK(fase IN ('consulta_carga','consulta_documento','abrir','abrir_rrhh'));

CREATE FUNCTION vec_documentos.selector_subida_local_v1() RETURNS jsonb
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT CASE session_user
  WHEN 'vec_externo_documentos_datos' THEN jsonb_build_object(
   'modulo','meritos','tipo','hecho','finalidad','custodiar_evidencia_meritos',
   'subir','meritos.cartera.comprobante.subir_propio',
   'descargar','meritos.cartera.comprobante.descargar_propio',
   'audiencia_subir','vec_documentos.subida_local.acto_modulo.v1',
   'audiencia_descargar','vec_documentos.subida_local.descarga_propia.v1',
   'superficie','externa_personal')
  WHEN 'vec_externo_documentos_bolsa_datos' THEN jsonb_build_object(
   'modulo','bolsa','tipo','justificante_renuncia',
   'finalidad','custodiar_justificante_renuncia_bolsa',
   'subir','bolsa.justificante_renuncia.subir_propio',
   'descargar','bolsa.justificante_renuncia.descargar_propio',
   'audiencia_subir','vec_documentos.bolsa.justificante_renuncia.subida.v1',
   'audiencia_descargar','vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1',
   'superficie','externa_personal')
 END WHERE current_user='vec_documentos_propietario'
$f$;
REVOKE ALL ON FUNCTION vec_documentos.selector_subida_local_v1() FROM PUBLIC;

CREATE OR REPLACE FUNCTION vec_documentos.subida_local_login_valido_v1(p_frontera boolean) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_user='vec_documentos_propietario' AND (
  (session_user=CASE WHEN p_frontera THEN 'vec_externo_documentos_frontera'
   ELSE 'vec_externo_documentos_datos' END AND
   EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit AND
    NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)) AND
   (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1 AND
   EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND
    m.roleid=(CASE WHEN p_frontera THEN 'vec_documentos_frontera_externa'
     ELSE 'vec_documentos_subida_datos' END)::regrole
    AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option))
  OR
  (p_frontera IS FALSE AND session_user='vec_externo_documentos_bolsa_datos' AND
   EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND
    r.rolinherit AND r.rolconfig IS NULL AND
    NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)) AND
   (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1 AND
   EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND
    m.roleid='vec_documentos_bolsa_subida_datos'::regrole AND
    m.inherit_option AND NOT m.set_option AND NOT m.admin_option AND
    NOT EXISTS(SELECT 1 FROM pg_auth_members superior WHERE superior.member=m.roleid))))
$f$;
REVOKE ALL ON FUNCTION vec_documentos.subida_local_login_valido_v1(boolean) FROM PUBLIC;
CREATE OR REPLACE FUNCTION vec_documentos.consumir_preimagen_subida_local_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE j jsonb; c jsonb; d jsonb; x jsonb; v record; claves text[]; fase text; accion text;
 efecto text; ambito text; pre_sha text; esperado text; s jsonb;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  octet_length(p_preimagen) NOT BETWEEN 1 AND 65536 OR
  jsonb_typeof(p_auth) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(p_auth))<>7 OR
  p_auth ?& ARRAY['accion','recurso_ref','ambito_ref','principal_id','perfil_activo_ref','finalidad','correlacion_ref'] IS NOT TRUE
 THEN RAISE EXCEPTION 'D20: invocación denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  j:=convert_from(p_preimagen,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D20: material ilegible' USING ERRCODE='22023'; END;
 IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR
  jsonb_typeof(d) IS DISTINCT FROM 'object' OR jsonb_typeof(x) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'D20: objeto inválido' USING ERRCODE='22023'; END IF;
 fase:=j->>'fase'; accion:=j->>'accion';
 s:=vec_documentos.selector_subida_local_v1();
 IF s IS NULL THEN RAISE EXCEPTION 'D22: selector ausente' USING ERRCODE='42501'; END IF;
 claves:=CASE fase
  WHEN 'reservar' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','modulo_id','tipo_recurso','finalidad','clave_idempotencia_hmac','huella_solicitud_hmac','huella_pdf_sha256','tamano','mime','propietario_version','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'confirmar_recepcion' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','huella_pdf_sha256','objeto','evidencia_almacen','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'iniciar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','recibo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'registrar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','analisis','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'cerrar_analisis' THEN ARRAY['accion','fase','carga_id','recurso_ref','persona_ref','perfil_activo_ref','finalidad','version_esperada','estado_final','codigo_rechazo','analisis','promocion','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'consulta_carga' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'consulta_documento' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  WHEN 'abrir' THEN ARRAY['accion','fase','carga_id','documento_id','persona_ref','perfil_activo_ref','objeto_ref','objeto_version','huella_pdf_sha256','tamano','mime','registro_contexto_ref','huella_contexto_sha256','correlacion_ref']
  ELSE NULL END;
 IF claves IS NULL OR (SELECT count(*) FROM jsonb_object_keys(j))<>array_length(claves,1) OR
  j ?& claves IS NOT TRUE OR
  ((fase IN ('reservar','confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis','consulta_carga'))
    <> (accion=s->>'subir')) OR
  accion NOT IN (s->>'subir',s->>'descargar') OR
  j->>'persona_ref' !~ '^[A-Za-z0-9:_-]{1,128}$' OR
  j->>'persona_ref' IS DISTINCT FROM x->>'persona_ref' OR
  j->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref' OR
  j->>'registro_contexto_ref' IS DISTINCT FROM c->>'contexto_ref' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM c->>'huella_contexto_sha256' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  j->>'finalidad' IS DISTINCT FROM (CASE WHEN fase IN ('reservar','confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis')
   THEN s->>'finalidad' ELSE j->>'finalidad' END) OR
  p_auth->>'accion' IS DISTINCT FROM accion OR
  p_auth->>'principal_id' IS DISTINCT FROM j->>'persona_ref' OR
  p_auth->>'perfil_activo_ref' IS DISTINCT FROM j->>'perfil_activo_ref' OR
  p_auth->>'finalidad' IS DISTINCT FROM s->>'finalidad' OR
  p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref'
 THEN RAISE EXCEPTION 'D20: preimagen divergente' USING ERRCODE='42501'; END IF;
 efecto:=CASE WHEN fase='consulta_documento' THEN j->>'documento_id' ELSE j->>'carga_id' END;
 ambito:=CASE WHEN fase IN ('consulta_carga','consulta_documento') THEN efecto
  WHEN fase='abrir' THEN p_auth->>'ambito_ref' ELSE j->>'recurso_ref' END;
 IF efecto IS NULL OR efecto !~ '^ref:[0-9a-f]{64}$' OR
  p_auth->>'recurso_ref' IS DISTINCT FROM efecto OR
  p_auth->>'ambito_ref' IS DISTINCT FROM ambito OR
  (fase IN ('consulta_carga','consulta_documento') AND p_auth->>'ambito_ref' IS DISTINCT FROM efecto) OR
  c->>'efecto_ref' IS DISTINCT FROM efecto OR d->>'recurso_ref' IS DISTINCT FROM efecto OR
  c->>'operacion' IS DISTINCT FROM accion OR d->>'accion' IS DISTINCT FROM accion OR
  c->>'audiencia_consumo' IS DISTINCT FROM (CASE WHEN accion=s->>'subir'
   THEN s->>'audiencia_subir' ELSE s->>'audiencia_descargar' END) OR
  d->>'modulo_id' IS DISTINCT FROM s->>'modulo' OR d->>'tipo_recurso' IS DISTINCT FROM s->>'tipo' OR
  d->>'finalidad' IS DISTINCT FROM s->>'finalidad' OR
  d->'campos_permitidos' IS DISTINCT FROM (CASE WHEN accion=s->>'subir'
   THEN '["contenido","estado","recibo"]'::jsonb ELSE '["contenido"]'::jsonb END) OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM j->>'registro_contexto_ref' OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM j->>'huella_contexto_sha256' OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM s->>'superficie'
 THEN RAISE EXCEPTION 'D20: decisión divergente' USING ERRCODE='42501'; END IF;
 pre_sha:=encode(sha256(p_preimagen),'hex');
 esperado:=encode(sha256(convert_to(format(
  '{"ambitos":{"persona_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}',
  j->>'persona_ref',pre_sha),'UTF8')),'hex');
 IF d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM esperado OR
  c->>'huella_efecto_sha256' IS DISTINCT FROM esperado
 THEN RAISE EXCEPTION 'D20: efecto no ligado' USING ERRCODE='42501'; END IF;
 IF s->>'modulo'='bolsa' THEN
  SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_modulo_v1(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF v.consumo_nuevo IS DISTINCT FROM TRUE OR v.efecto_ref IS DISTINCT FROM efecto OR
  v.huella_efecto_sha256 IS DISTINCT FROM esperado
 THEN RAISE EXCEPTION 'D20: consumo inválido' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('preimagen',j,'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
  'preimagen_sha256',pre_sha,'persona_version',p_persona_version);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.consumir_preimagen_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;


CREATE OR REPLACE FUNCTION vec_documentos.operar_subida_local_interna_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE material jsonb; j jsonb; r vec_documentos.subida_local%ROWTYPE; fase text;
 objeto jsonb; evidencia jsonb; v_analisis jsonb; v_promocion jsonb; huella text;
 documento text; recibo text; codigo text; estado_final text; s jsonb;
BEGIN
 material:=vec_documentos.consumir_preimagen_subida_local_v1(p_preimagen,p_auth,p_capacidad,p_decision,
  p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 j:=material->'preimagen'; fase:=j->>'fase'; s:=vec_documentos.selector_subida_local_v1();
 IF s IS NULL THEN RAISE EXCEPTION 'D22: selector ausente' USING ERRCODE='42501'; END IF;
 IF fase='reservar' THEN
  IF j->>'modulo_id' IS DISTINCT FROM s->>'modulo' OR j->>'tipo_recurso' IS DISTINCT FROM s->>'tipo' OR
   j->>'finalidad' IS DISTINCT FROM s->>'finalidad' OR
   (s->>'modulo'='meritos' AND j->>'recurso_ref' !~ '^ref:[0-9a-f]{64}$') OR
   (s->>'modulo'='bolsa' AND j->>'recurso_ref' !~ '^justificante-renuncia:[0-9a-f]{64}$') OR
   j->>'clave_idempotencia_hmac' !~ '^hmac-sha256:local_v1:[0-9a-f]{64}$' OR
   j->>'huella_solicitud_hmac' !~ '^hmac-sha256:[a-z0-9_]{1,64}:[0-9a-f]{64}$' OR
   j->>'huella_pdf_sha256' !~ '^[0-9a-f]{64}$' OR
   j->>'mime' IS DISTINCT FROM 'application/pdf' OR
   j->>'tamano' !~ '^[1-9][0-9]{0,7}$' OR
   (j->>'tamano')::bigint>16777216 OR
   j->>'propietario_version' !~ '^[1-9][0-9]{0,18}$' OR
   (s->>'modulo'='meritos' AND (j->>'propietario_version')::numeric IS DISTINCT FROM p_persona_version) OR
   (s->>'modulo'='bolsa' AND (j->>'propietario_version')::numeric IS DISTINCT FROM 1) OR
   j->>'persona_ref' IS DISTINCT FROM (convert_from(p_contexto,'UTF8')::jsonb)->>'persona_ref'
  THEN RAISE EXCEPTION 'D20: reserva inválida' USING ERRCODE='42501'; END IF;
  IF s->>'modulo'='bolsa' THEN
   IF vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(
    j->>'recurso_ref',(j->>'propietario_version')::bigint,j->>'persona_ref',
    j->>'registro_contexto_ref') IS NOT TRUE
   THEN RAISE EXCEPTION 'D22: titular o llamamiento no vigentes' USING ERRCODE='42501'; END IF;
  END IF;
  PERFORM pg_advisory_xact_lock(hashtextextended('vec_documentos:subida_local:'||(j->>'persona_ref')||':'||(j->>'clave_idempotencia_hmac'),0));
  SELECT * INTO r FROM vec_documentos.subida_local WHERE
   persona_ref=j->>'persona_ref' AND clave_idempotencia_hmac=j->>'clave_idempotencia_hmac' FOR UPDATE;
  IF FOUND THEN
   IF r.carga_id IS DISTINCT FROM j->>'carga_id' OR r.recurso_ref IS DISTINCT FROM j->>'recurso_ref' OR
    r.modulo_id IS DISTINCT FROM s->>'modulo' OR r.tipo_recurso IS DISTINCT FROM s->>'tipo' OR
    r.finalidad IS DISTINCT FROM s->>'finalidad' OR
    r.perfil_activo_ref IS DISTINCT FROM j->>'perfil_activo_ref' OR
    r.huella_solicitud_hmac IS DISTINCT FROM j->>'huella_solicitud_hmac' OR
    r.huella_pdf_sha256 IS DISTINCT FROM j->>'huella_pdf_sha256' OR
    r.tamano IS DISTINCT FROM (j->>'tamano')::bigint
   THEN RAISE EXCEPTION 'D20: clave usada para otra carga' USING ERRCODE='23505'; END IF;
   RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
  END IF;
  INSERT INTO vec_documentos.subida_local(carga_id,recurso_ref,persona_ref,perfil_activo_ref,
   modulo_id,tipo_recurso,finalidad,clasificacion,propietario_version,clave_idempotencia_hmac,
   huella_solicitud_hmac,huella_pdf_sha256,tamano,mime,correlacion_ref,registro_contexto_ref,
   huella_contexto_sha256,version,estado)
  VALUES(j->>'carga_id',j->>'recurso_ref',j->>'persona_ref',j->>'perfil_activo_ref',
   s->>'modulo',s->>'tipo',s->>'finalidad','confidencial',(j->>'propietario_version')::numeric,
   j->>'clave_idempotencia_hmac',j->>'huella_solicitud_hmac',j->>'huella_pdf_sha256',
   (j->>'tamano')::bigint,'application/pdf',j->>'correlacion_ref',j->>'registro_contexto_ref',
   j->>'huella_contexto_sha256',1,'reservada');
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(j->>'carga_id',1,'reservada',
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
  RETURN vec_documentos.instantanea_subida_local_v1(j->>'carga_id');
 END IF;
 IF fase='consulta_documento' THEN
  IF j->>'carga_id' IS DISTINCT FROM '' OR j->>'documento_id' !~ '^ref:[0-9a-f]{64}$'
  THEN RAISE EXCEPTION 'D20: consulta documental inválida' USING ERRCODE='42501'; END IF;
  SELECT * INTO r FROM vec_documentos.subida_local WHERE documento_id=j->>'documento_id' FOR SHARE;
 ELSE
  IF j->>'carga_id' !~ '^ref:[0-9a-f]{64}$' THEN RAISE EXCEPTION 'D20: carga inválida' USING ERRCODE='42501'; END IF;
  SELECT * INTO r FROM vec_documentos.subida_local WHERE carga_id=j->>'carga_id'
   FOR UPDATE;
 END IF;
 IF NOT FOUND OR r.modulo_id IS DISTINCT FROM s->>'modulo' OR
  r.tipo_recurso IS DISTINCT FROM s->>'tipo' OR r.finalidad IS DISTINCT FROM s->>'finalidad' OR
  r.persona_ref IS DISTINCT FROM j->>'persona_ref' OR
  (fase IN ('confirmar_recepcion','iniciar_analisis','registrar_analisis','cerrar_analisis') AND
   (r.recurso_ref IS DISTINCT FROM j->>'recurso_ref' OR r.finalidad IS DISTINCT FROM j->>'finalidad' OR
    r.correlacion_ref IS DISTINCT FROM j->>'correlacion_ref')) OR
  (fase='abrir' AND (r.documento_id IS DISTINCT FROM j->>'documento_id' OR
   p_auth->>'ambito_ref' IS DISTINCT FROM r.recurso_ref OR
   j->>'objeto_ref' IS DISTINCT FROM r.admitido#>>'{Objeto,Objeto,Referencia}' OR
   j->>'objeto_version' IS DISTINCT FROM r.admitido#>>'{Objeto,Objeto,Version}' OR
   j->>'huella_pdf_sha256' IS DISTINCT FROM r.admitido#>>'{Objeto,HuellaSHA256}' OR
   j->>'tamano' IS DISTINCT FROM r.admitido#>>'{Objeto,Tamano}' OR
   j->>'mime' IS DISTINCT FROM r.admitido#>>'{Objeto,MIME}')) OR
  (fase='consulta_carga' AND j->>'documento_id' IS DISTINCT FROM '') OR
  (fase='consulta_documento' AND r.estado IS DISTINCT FROM 'admitido')
 THEN RAISE EXCEPTION 'D20: carga propia ausente' USING ERRCODE='42501'; END IF;
 PERFORM vec_documentos.verificar_integridad_subida_local_v1(r.carga_id);
 IF fase IN ('consulta_carga','consulta_documento','abrir') THEN
  IF fase='abrir' AND (r.estado IS DISTINCT FROM 'admitido' OR r.admitido IS NULL OR
   r.analisis->>'Estado' IS DISTINCT FROM 'limpio')
  THEN RAISE EXCEPTION 'D20: original no admitido' USING ERRCODE='42501'; END IF;
  INSERT INTO vec_documentos.subida_local_lectura(decision_ref,carga_id,fase,persona_ref,auditoria_v3_ref,huella_preimagen_sha256)
  VALUES(material->>'decision_ref',r.carga_id,fase,r.persona_ref,material->>'auditoria_ref',material->>'preimagen_sha256');
  RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
 END IF;
 IF j->>'version_esperada' !~ '^[1-4]$' OR r.version<>(j->>'version_esperada')::bigint OR
  (s->>'modulo'='meritos' AND r.propietario_version>p_persona_version) OR
  r.estado IS DISTINCT FROM (CASE fase WHEN 'confirmar_recepcion' THEN 'reservada'
   WHEN 'iniciar_analisis' THEN 'recibido' ELSE 'analizando' END)
 THEN RAISE EXCEPTION 'D20: transición obsoleta' USING ERRCODE='40001'; END IF;
 IF fase='confirmar_recepcion' THEN
  objeto:=j->'objeto'; evidencia:=j->'evidencia_almacen';
  IF jsonb_typeof(objeto)<>'object' OR jsonb_typeof(evidencia)<>'object' OR
   j->>'huella_pdf_sha256' IS DISTINCT FROM r.huella_pdf_sha256 OR
   objeto->>'Zona' IS DISTINCT FROM 'cuarentena' OR objeto->>'MIME' IS DISTINCT FROM r.mime OR
   objeto->>'HuellaSHA256' IS DISTINCT FROM r.huella_pdf_sha256 OR
   objeto->>'Tamano' IS DISTINCT FROM r.tamano::text OR
   objeto#>>'{Objeto,Referencia}' !~ '^obj_[0-9a-f]{32}$' OR
   objeto#>>'{Objeto,Version}' IS DISTINCT FROM '1' OR
   evidencia->>'Accion' IS DISTINCT FROM 'escribir' OR
   evidencia->>'AccionNegocio' IS DISTINCT FROM j->>'accion' OR
   evidencia->>'CargaRef' IS DISTINCT FROM r.carga_id OR
   evidencia->>'RecursoRef' IS DISTINCT FROM r.carga_id OR
   evidencia->>'ModuloID' IS DISTINCT FROM r.modulo_id OR
   evidencia->>'Finalidad' IS DISTINCT FROM r.finalidad OR
   evidencia->>'CorrelacionRef' IS DISTINCT FROM r.correlacion_ref OR
   evidencia->>'Referencia' IS DISTINCT FROM objeto->>'EvidenciaCreacionRef' OR
   evidencia->>'ConectorID' IS DISTINCT FROM objeto->>'ConectorID' OR
   evidencia#>>'{Objeto,Referencia}' IS DISTINCT FROM objeto#>>'{Objeto,Referencia}' OR
   evidencia#>>'{Objeto,Version}' IS DISTINCT FROM objeto#>>'{Objeto,Version}'
  THEN RAISE EXCEPTION 'D20: recibo de cuarentena divergente' USING ERRCODE='42501'; END IF;
  recibo:='ref:'||encode(sha256(convert_to('vec.documentos.subida_local.recibo.v1:'||r.carga_id||':'||
   (objeto#>>'{Objeto,Referencia}')||':'||(objeto#>>'{Objeto,Version}')||':'||r.huella_pdf_sha256,'UTF8')),'hex');
  UPDATE vec_documentos.subida_local SET estado='recibido',version=2,cuarentena=jsonb_build_object(
   'Objeto',objeto,'Evidencia',evidencia),recibo_ref=recibo,actualizada_en=clock_timestamp() WHERE carga_id=r.carga_id;
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,2,'recibido',
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
 ELSIF fase='iniciar_analisis' THEN
  IF j->>'recibo_ref' IS DISTINCT FROM r.recibo_ref
  THEN RAISE EXCEPTION 'D20: recibo divergente' USING ERRCODE='42501'; END IF;
  UPDATE vec_documentos.subida_local SET estado='analizando',version=3,actualizada_en=clock_timestamp()
   WHERE carga_id=r.carga_id;
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,3,'analizando',
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
 ELSIF fase IN ('registrar_analisis','cerrar_analisis') THEN
  v_analisis:=j->'analisis'; v_promocion:=j->'promocion'; codigo:=j->>'codigo_rechazo';
  estado_final:=j->>'estado_final';
  IF jsonb_typeof(v_analisis)<>'object' OR
   v_analisis#>>'{Objeto,Referencia}' IS DISTINCT FROM r.cuarentena#>>'{Objeto,Objeto,Referencia}' OR
   v_analisis#>>'{Objeto,Version}' IS DISTINCT FROM r.cuarentena#>>'{Objeto,Objeto,Version}' OR
   v_analisis->>'HuellaObjetoSHA256' IS DISTINCT FROM r.huella_pdf_sha256 OR
   v_analisis->>'TamanoObjeto' IS DISTINCT FROM r.tamano::text OR
   v_analisis->>'MIMEDeclarado' IS DISTINCT FROM r.mime OR
   v_analisis->>'CorrelacionRef' IS DISTINCT FROM r.correlacion_ref OR
   v_analisis->>'Finalidad' IS DISTINCT FROM r.finalidad OR
   nullif(v_analisis->>'EvidenciaRef','') IS NULL
  THEN RAISE EXCEPTION 'D20: análisis sin vínculo' USING ERRCODE='42501'; END IF;
  IF fase='registrar_analisis' THEN
   IF r.version<>3 OR r.analisis IS NOT NULL OR
    (v_analisis->>'Estado'=ANY(ARRAY['limpio','malicioso','sospechoso','no_concluyente','error'])) IS NOT TRUE
   THEN RAISE EXCEPTION 'D20: registro de análisis inválido' USING ERRCODE='42501'; END IF;
   UPDATE vec_documentos.subida_local SET analisis=v_analisis,version=4,actualizada_en=clock_timestamp()
    WHERE carga_id=r.carga_id;
   PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,4,'analizando',
    material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
   RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
  END IF;
  IF r.version<>4 OR v_analisis IS DISTINCT FROM r.analisis OR
   ((estado_final='admitido' AND (codigo IS DISTINCT FROM '' OR
    v_analisis->>'Estado' IS DISTINCT FROM 'limpio' OR
    v_analisis->>'MIMEDetectado' IS DISTINCT FROM r.mime OR
    jsonb_typeof(v_promocion) IS DISTINCT FROM 'object')) OR
    (estado_final='rechazado' AND (codigo NOT IN ('pdf_no_valido','contenido_no_permitido','analisis_no_concluyente') OR
     jsonb_typeof(v_promocion) IS DISTINCT FROM 'null' OR
     (v_analisis->>'Estado'=ANY(ARRAY['malicioso','sospechoso','no_concluyente','error'])) IS NOT TRUE OR
     (codigo='pdf_no_valido' AND v_analisis->>'CodigoResultado' IS DISTINCT FROM 'PDF_INVALIDO') OR
     (v_analisis->>'Estado' IN ('malicioso','sospechoso') AND
       codigo NOT IN ('contenido_no_permitido','pdf_no_valido')) OR
     (v_analisis->>'Estado' IN ('no_concluyente','error') AND
       codigo NOT IN ('analisis_no_concluyente','pdf_no_valido')))) OR
    estado_final NOT IN ('admitido','rechazado'))
  THEN RAISE EXCEPTION 'D20: análisis divergente' USING ERRCODE='42501'; END IF;
  IF estado_final='admitido' AND s->>'modulo'='bolsa' THEN
   IF vec_bolsa_llamamientos.validar_propietario_subida_documental_v1(
    r.recurso_ref,r.propietario_version::bigint,r.persona_ref,
    j->>'registro_contexto_ref') IS NOT TRUE
   THEN RAISE EXCEPTION 'D22: titular o llamamiento cambiados' USING ERRCODE='42501'; END IF;
  END IF;
  IF estado_final='admitido' THEN
   objeto:=v_promocion->'Objeto'; evidencia:=v_promocion->'Evidencia';
   IF objeto->>'Zona' IS DISTINCT FROM 'admitida' OR objeto->>'MIME' IS DISTINCT FROM r.mime OR
    objeto->>'HuellaSHA256' IS DISTINCT FROM r.huella_pdf_sha256 OR
    objeto->>'Tamano' IS DISTINCT FROM r.tamano::text OR
    objeto#>>'{Objeto,Referencia}' IS NOT DISTINCT FROM r.cuarentena#>>'{Objeto,Objeto,Referencia}' OR
    objeto#>>'{Objeto,Referencia}' !~ '^obj_[0-9a-f]{32}$' OR
    objeto#>>'{Objeto,Version}' IS DISTINCT FROM '1' OR
    evidencia->>'Accion' IS DISTINCT FROM 'promover' OR
    evidencia->>'AccionNegocio' IS DISTINCT FROM j->>'accion' OR
    evidencia->>'CargaRef' IS DISTINCT FROM r.carga_id OR
    evidencia->>'RecursoRef' IS DISTINCT FROM r.carga_id OR
    evidencia->>'ModuloID' IS DISTINCT FROM r.modulo_id OR
    evidencia->>'Finalidad' IS DISTINCT FROM r.finalidad OR
    evidencia->>'FundamentoRef' IS DISTINCT FROM v_analisis->>'EvidenciaRef' OR
    evidencia->>'CorrelacionRef' IS DISTINCT FROM r.correlacion_ref OR
    evidencia->>'Referencia' IS DISTINCT FROM objeto->>'EvidenciaCreacionRef' OR
    evidencia->>'ConectorID' IS DISTINCT FROM objeto->>'ConectorID' OR
    evidencia#>>'{Objeto,Referencia}' IS DISTINCT FROM objeto#>>'{Objeto,Referencia}' OR
    evidencia#>>'{Objeto,Version}' IS DISTINCT FROM objeto#>>'{Objeto,Version}'
   THEN RAISE EXCEPTION 'D20: promoción divergente' USING ERRCODE='42501'; END IF;
   documento:='ref:'||encode(sha256(convert_to('vec.documentos.subida_local.comprobante.v1:'||r.carga_id,'UTF8')),'hex');
   UPDATE vec_documentos.subida_local SET estado='admitido',version=5,analisis=v_analisis,
    admitido=v_promocion,documento_id=documento,documento_version=1,codigo_rechazo='',
    registrada_en=clock_timestamp(),actualizada_en=clock_timestamp() WHERE carga_id=r.carga_id;
  ELSE
   UPDATE vec_documentos.subida_local SET estado='rechazado',version=5,analisis=v_analisis,
    codigo_rechazo=codigo,actualizada_en=clock_timestamp() WHERE carga_id=r.carga_id;
  END IF;
  PERFORM vec_documentos.registrar_cambio_subida_local_v1(r.carga_id,5,estado_final,
   material->>'decision_ref',material->>'auditoria_ref',material->>'preimagen_sha256');
 ELSE RAISE EXCEPTION 'D20: fase desconocida' USING ERRCODE='22023'; END IF;
 RETURN vec_documentos.instantanea_subida_local_v1(r.carga_id);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.operar_subida_local_interna_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;

-- Las seis fachadas D20 conservan su firma. El selector y el consumidor V3
-- distinguen el LOGIN externo de Bolsa del de Méritos dentro de cada llamada.
GRANT EXECUTE ON FUNCTION
 vec_documentos.reservar_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.confirmar_recepcion_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.iniciar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.registrar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.cerrar_analisis_subida_local_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_documentos.consultar_subida_local_propia_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_bolsa_subida_datos;

-- Solo Bolsa consulta esta prueba dentro de su propia transacción de renuncia.
-- El contexto recibido es el de la respuesta B30, acreditado por Bolsa; la
-- reserva documental conserva otro contexto porque fue otra petición.
-- La respuesta es booleana y nunca entrega metadatos de otra persona.
CREATE FUNCTION vec_documentos.validar_justificante_renuncia_bolsa_v1(
 p_documento_id text,p_huella_sha256 text,p_recurso_ref text,p_persona_ref text,
 p_registro_contexto_ref text) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE r vec_documentos.subida_local%ROWTYPE;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR
  p_documento_id IS NULL OR p_documento_id !~ '^ref:[0-9a-f]{64}$' OR
  p_huella_sha256 IS NULL OR p_huella_sha256 !~ '^[0-9a-f]{64}$' OR
  p_recurso_ref IS NULL OR p_recurso_ref !~ '^justificante-renuncia:[0-9a-f]{64}$' OR
  p_persona_ref IS NULL OR p_registro_contexto_ref IS NULL
 THEN RETURN false; END IF;
 SELECT * INTO r FROM vec_documentos.subida_local
 WHERE documento_id=p_documento_id FOR SHARE;
 IF NOT FOUND OR r.modulo_id IS DISTINCT FROM 'bolsa' OR
  r.tipo_recurso IS DISTINCT FROM 'justificante_renuncia' OR
  r.finalidad IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  r.estado IS DISTINCT FROM 'admitido' OR r.version IS DISTINCT FROM 5 OR
  r.recurso_ref IS DISTINCT FROM p_recurso_ref OR
  r.persona_ref IS DISTINCT FROM p_persona_ref OR
  r.huella_pdf_sha256 IS DISTINCT FROM p_huella_sha256 OR
  r.admitido#>>'{Objeto,HuellaSHA256}' IS DISTINCT FROM p_huella_sha256 OR
  r.analisis->>'Estado' IS DISTINCT FROM 'limpio' OR
  r.recibo_ref IS NULL OR r.registrada_en IS NULL
 THEN RETURN false; END IF;
 PERFORM vec_documentos.verificar_integridad_subida_local_v1(r.carga_id);
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.validar_justificante_renuncia_bolsa_v1(text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.validar_justificante_renuncia_bolsa_v1(text,text,text,text,text)
 TO vec_bolsa_llamamientos_propietario;

CREATE FUNCTION vec_documentos.login_justificante_renuncia_rrhh_valido_v1() RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT current_user='vec_documentos_propietario' AND
  session_user='vec_interno_documentos_bolsa_rrhh' AND
  EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
   AND r.rolinherit AND r.rolconfig IS NULL AND
   NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)) AND
  (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1 AND
  EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
   AND m.roleid='vec_documentos_bolsa_rrhh'::regrole
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND NOT EXISTS(SELECT 1 FROM pg_auth_members superior WHERE superior.member=m.roleid))
$f$;
REVOKE ALL ON FUNCTION vec_documentos.login_justificante_renuncia_rrhh_valido_v1() FROM PUBLIC;

-- El titular y el actor de RRHH son distintos. AD277 verifica la decisión,
-- sesión y perfil RRHH; Bolsa coteja participación y llamamiento reales.
-- Solo después se proyecta la instantánea admitida para abrir su objeto.
CREATE FUNCTION vec_documentos.consultar_justificante_renuncia_rrhh_v1(
 p_preimagen bytea,p_auth jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE j jsonb; c jsonb; d jsonb; x jsonb; r vec_documentos.subida_local%ROWTYPE;
 v record; v_pre_sha text; v_efecto_sha text; v_claves constant text[]:=ARRAY[
  'fase','accion','carga_id','documento_id','recurso_ref','persona_ref','actor_ref',
  'perfil_activo_ref','bolsa_ref','participacion_ref','registro_contexto_ref',
  'huella_contexto_sha256','correlacion_ref'];
BEGIN
 IF vec_documentos.login_justificante_renuncia_rrhh_valido_v1() IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  p_preimagen IS NULL OR octet_length(p_preimagen) NOT BETWEEN 1 AND 65536 OR
  jsonb_typeof(p_auth) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(p_auth))<>7 OR
  (p_auth ?& ARRAY['accion','recurso_ref','ambito_ref','principal_id',
    'perfil_activo_ref','finalidad','correlacion_ref']) IS NOT TRUE
 THEN RAISE EXCEPTION 'D22: consulta RRHH denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  j:=convert_from(p_preimagen,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D22: material RRHH ilegible' USING ERRCODE='22023'; END;
 IF jsonb_typeof(j) IS DISTINCT FROM 'object' OR
  (SELECT count(*) FROM jsonb_object_keys(j))<>array_length(v_claves,1) OR
  (j ?& v_claves) IS NOT TRUE OR
  EXISTS(SELECT 1 FROM unnest(v_claves) AS k(clave) WHERE
   jsonb_typeof(j->k.clave) IS DISTINCT FROM 'string' OR
   octet_length(j->>k.clave)>512 OR
   (k.clave<>'carga_id' AND octet_length(j->>k.clave)=0)) OR
  jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR
  jsonb_typeof(x) IS DISTINCT FROM 'object' OR
  j->>'fase' IS DISTINCT FROM 'abrir_rrhh' OR
  j->>'accion' IS DISTINCT FROM 'bolsa.justificante_renuncia.descargar_rrhh' OR
  (j->>'carga_id'<>'' AND j->>'carga_id' !~ '^ref:[0-9a-f]{64}$') OR
  j->>'documento_id' !~ '^ref:[0-9a-f]{64}$' OR
  j->>'recurso_ref' !~ '^justificante-renuncia:[0-9a-f]{64}$' OR
  j->>'persona_ref' !~ '^[A-Za-z0-9:_-]{1,128}$' OR
  nullif(j->>'actor_ref','') IS NULL OR nullif(j->>'perfil_activo_ref','') IS NULL OR
  j->>'bolsa_ref' !~ '^[A-Za-z0-9:_-]{1,256}$' OR
  j->>'participacion_ref' !~ '^[A-Za-z0-9:_-]{1,256}$' OR
  nullif(j->>'correlacion_ref','') IS NULL OR
  nullif(j->>'registro_contexto_ref','') IS NULL OR
  j->>'huella_contexto_sha256' !~ '^[0-9a-f]{64}$' OR
  j->>'registro_contexto_ref' IS DISTINCT FROM c->>'contexto_ref' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM c->>'huella_contexto_sha256' OR
  j->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  j->>'actor_ref' IS DISTINCT FROM x->>'principal_ref' OR
  j->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref' OR
  p_auth->>'accion' IS DISTINCT FROM j->>'accion' OR
  p_auth->>'recurso_ref' IS DISTINCT FROM j->>'documento_id' OR
  p_auth->>'ambito_ref' IS DISTINCT FROM j->>'recurso_ref' OR
  p_auth->>'principal_id' IS DISTINCT FROM j->>'actor_ref' OR
  p_auth->>'perfil_activo_ref' IS DISTINCT FROM j->>'perfil_activo_ref' OR
  p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  p_auth->>'correlacion_ref' IS DISTINCT FROM j->>'correlacion_ref' OR
  c->>'efecto_ref' IS DISTINCT FROM j->>'documento_id' OR
  c->>'operacion' IS DISTINCT FROM j->>'accion' OR
  c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1' OR
  d->>'accion' IS DISTINCT FROM j->>'accion' OR
  d->>'recurso_ref' IS DISTINCT FROM j->>'documento_id' OR
  d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'justificante_renuncia' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  d->>'principal_id' IS DISTINCT FROM j->>'actor_ref' OR
  d->>'perfil_activo_ref' IS DISTINCT FROM j->>'perfil_activo_ref' OR
  d->'campos_permitidos' IS DISTINCT FROM '["contenido"]'::jsonb OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM j->>'registro_contexto_ref' OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM j->>'huella_contexto_sha256' OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 THEN RAISE EXCEPTION 'D22: decisión RRHH divergente' USING ERRCODE='42501'; END IF;
 v_pre_sha:=encode(sha256(p_preimagen),'hex');
 v_efecto_sha:=encode(sha256(convert_to(format(
  '{"ambitos":{"bolsa_ref":"%s","participacion_ref":"%s"},"atributos":{"preimagen_sha256":"%s"}}',
  j->>'bolsa_ref',j->>'participacion_ref',v_pre_sha),'UTF8')),'hex');
 IF c->>'huella_efecto_sha256' IS DISTINCT FROM v_efecto_sha OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_efecto_sha
 THEN RAISE EXCEPTION 'D22: recurso RRHH no ligado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM true OR
  v.efecto_ref IS DISTINCT FROM j->>'documento_id' OR
  v.huella_efecto_sha256 IS DISTINCT FROM v_efecto_sha
 THEN RAISE EXCEPTION 'D22: consumo RRHH inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO r FROM vec_documentos.subida_local
 WHERE documento_id=j->>'documento_id' FOR SHARE;
 IF NOT FOUND OR r.modulo_id IS DISTINCT FROM 'bolsa' OR
  (j->>'carga_id'<>'' AND r.carga_id IS DISTINCT FROM j->>'carga_id') OR
  r.tipo_recurso IS DISTINCT FROM 'justificante_renuncia' OR
  r.finalidad IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  r.recurso_ref IS DISTINCT FROM j->>'recurso_ref' OR
  r.persona_ref IS DISTINCT FROM j->>'persona_ref' OR
  r.estado IS DISTINCT FROM 'admitido' OR r.version IS DISTINCT FROM 5 OR
  r.analisis->>'Estado' IS DISTINCT FROM 'limpio' OR
  r.admitido IS NULL OR r.recibo_ref IS NULL OR r.registrada_en IS NULL
 THEN RAISE EXCEPTION 'D22: justificante RRHH no disponible' USING ERRCODE='42501'; END IF;
 PERFORM vec_documentos.verificar_integridad_subida_local_v1(r.carga_id);
 IF vec_bolsa_llamamientos.validar_consulta_justificante_renuncia_rrhh_v1(
  r.recurso_ref,r.persona_ref,j->>'bolsa_ref',j->>'participacion_ref',
  j->>'actor_ref',j->>'perfil_activo_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'D22: relación RRHH denegada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_documentos.subida_local_lectura(decision_ref,carga_id,fase,persona_ref,
  auditoria_v3_ref,huella_preimagen_sha256)
 VALUES(v.decision_ref,r.carga_id,'abrir_rrhh',r.persona_ref,v.auditoria_ref,v_pre_sha);
 RETURN jsonb_build_object('resultado',jsonb_build_object('carga_id',r.carga_id,
  'comprobante_ref',r.documento_id,'documento_version',r.documento_version,
  'recibo_ref',r.recibo_ref,'estado',r.estado,'codigo_rechazo',r.codigo_rechazo,
  'registrado_en',r.registrada_en),
  'recurso_ref',r.recurso_ref,'persona_ref',r.persona_ref,'correlacion_ref',r.correlacion_ref,
  'modulo_id',r.modulo_id,'tipo_recurso',r.tipo_recurso,'finalidad',r.finalidad,
  'clasificacion',r.clasificacion,'huella_solicitud_hmac',r.huella_solicitud_hmac,
  'huella_pdf_sha256',r.huella_pdf_sha256,'mime',r.mime,'tamano',r.tamano,
  'version',r.version,'cuarentena',r.cuarentena,'analisis',r.analisis,'admitido',r.admitido);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.consultar_justificante_renuncia_rrhh_v1(
 bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.consultar_justificante_renuncia_rrhh_v1(
 bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_bolsa_rrhh;

-- La representación canónica V2 ya está ligada a la capacidad V3. Esta
-- comprobación pura cierra la selección del candidato antes de proyectar
-- hechos de Bolsa, sin invocar los helpers CA38 reservados al LOGIN de Bolsa.
CREATE FUNCTION vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
 p_contexto jsonb,p_persona_ref text,p_candidato_ref text,p_en timestamptz)
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' AS $f$
DECLARE v_numero bigint;v_ref text;v_desde text;v_hasta text;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR
  jsonb_typeof(p_contexto) IS DISTINCT FROM 'object' OR
  p_contexto->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2' OR
  p_contexto->>'persona_ref' IS DISTINCT FROM p_persona_ref OR
  p_contexto->>'estado' IS DISTINCT FROM 'activo' OR
  coalesce(p_candidato_ref ~ '^can_[A-Za-z0-9_-]{22,128}$',false) IS NOT TRUE OR
  p_en IS NULL OR NOT isfinite(p_en) OR
  jsonb_typeof(p_contexto->'vinculos') IS DISTINCT FROM 'array'
 THEN RETURN false; END IF;
 IF p_contexto->>'vigente_desde' IS NULL OR
  p_contexto->>'vigente_hasta' IS NULL OR
  (p_contexto->>'vigente_desde')::timestamptz>p_en OR
  (p_contexto->>'vigente_hasta')::timestamptz<=p_en
 THEN RETURN false; END IF;
 SELECT count(*),max(e->>'referencia'),max(e->>'vigente_desde'),max(e->>'vigente_hasta')
 INTO v_numero,v_ref,v_desde,v_hasta
 FROM jsonb_array_elements(p_contexto->'vinculos') e
 WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_numero<>1 OR v_ref IS DISTINCT FROM p_candidato_ref OR
  v_desde IS NULL OR v_hasta IS NULL OR
  v_desde::timestamptz>p_en OR v_hasta::timestamptz<=p_en
 THEN RETURN false; END IF;
 RETURN true;
EXCEPTION WHEN data_exception OR invalid_text_representation THEN
 RETURN false;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
 jsonb,text,text,timestamptz) FROM PUBLIC;

-- Preparación propia: AD277 consume la decisión antes de que Bolsa proyecte
-- el llamamiento vigente. El navegador nunca aporta la participación eficaz.
CREATE FUNCTION vec_documentos.resolver_justificante_renuncia_bolsa_v1(
 p_persona_ref text,p_candidato_ref text,p_bolsa_ref text,p_registro_contexto_ref text,
 p_corte timestamptz,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;d jsonb;x jsonb;v record;p record;h text;
BEGIN
 IF vec_documentos.subida_local_login_valido_v1(false) IS NOT TRUE OR
  session_user<>'vec_externo_documentos_bolsa_datos' OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  coalesce(p_persona_ref ~ '^[A-Za-z0-9:_-]{1,128}$',false) IS NOT TRUE OR
  coalesce(p_candidato_ref ~ '^[A-Za-z0-9:_-]{1,256}$',false) IS NOT TRUE OR
  coalesce(p_bolsa_ref ~ '^[A-Za-z0-9:_-]{1,256}$',false) IS NOT TRUE OR
  nullif(p_registro_contexto_ref,'') IS NULL OR
  p_corte IS NULL OR NOT isfinite(p_corte) OR
  p_corte<clock_timestamp()-interval '30 seconds' OR
  p_corte>clock_timestamp()+interval '1 second'
 THEN RAISE EXCEPTION 'D22: preparación propia denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D22: material propio ilegible' USING ERRCODE='22023'; END;
 h:=encode(sha256(convert_to(format(
  '{"ambitos":{"persona_ref":"%s"},"atributos":{"bolsa_ref":"%s","candidato_ref":"%s"}}',
  p_persona_ref,p_bolsa_ref,p_candidato_ref),'UTF8')),'hex');
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR
  jsonb_typeof(x) IS DISTINCT FROM 'object' OR
  x->>'persona_ref' IS DISTINCT FROM p_persona_ref OR
  c->>'contexto_ref' IS DISTINCT FROM p_registro_contexto_ref OR
  c->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  c->>'efecto_ref' IS DISTINCT FROM p_bolsa_ref OR
  c->>'operacion' IS DISTINCT FROM 'bolsa.justificante_renuncia.subir_propio' OR
  c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1' OR
  c->>'huella_efecto_sha256' IS DISTINCT FROM h OR
  d->>'accion' IS DISTINCT FROM c->>'operacion' OR
  d->>'recurso_ref' IS DISTINCT FROM p_bolsa_ref OR
  d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'justificante_renuncia' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  d->>'principal_id' IS DISTINCT FROM p_persona_ref OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR
  d->'campos_permitidos' IS DISTINCT FROM '["contenido","estado","recibo"]'::jsonb OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM p_registro_contexto_ref OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM
   encode(sha256(p_contexto),'hex') OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal' OR
  vec_documentos.candidato_propio_contexto_justificante_bolsa_v1(
   x,p_persona_ref,p_candidato_ref,clock_timestamp()) IS NOT TRUE
 THEN RAISE EXCEPTION 'D22: preparación propia divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM true OR v.efecto_ref IS DISTINCT FROM p_bolsa_ref OR
  v.huella_efecto_sha256 IS DISTINCT FROM h OR v.decision_ref IS NULL OR v.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'D22: consumo propio inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_bolsa_llamamientos.proyectar_justificante_renuncia_v1(
  p_persona_ref,p_candidato_ref,p_bolsa_ref,p_registro_contexto_ref,p_corte);
 IF coalesce(p.recurso_ref ~ '^justificante-renuncia:[0-9a-f]{64}$',false) IS NOT TRUE OR
  p.version_dueno IS DISTINCT FROM 1 OR
  p.candidato_ref IS DISTINCT FROM p_candidato_ref OR
  p.persona_ref IS DISTINCT FROM p_persona_ref OR
  nullif(p.participacion_ref,'') IS NULL OR nullif(p.llamamiento_ref,'') IS NULL
 THEN RAISE EXCEPTION 'D22: proyección propia divergente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('recurso_ref',p.recurso_ref,'version_dueno',p.version_dueno,
  'participacion_ref',p.participacion_ref,'llamamiento_ref',p.llamamiento_ref,
  'candidato_ref',p.candidato_ref,'persona_ref',p.persona_ref,
  'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.resolver_justificante_renuncia_bolsa_v1(
 text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.resolver_justificante_renuncia_bolsa_v1(
 text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_bolsa_subida_datos;

-- Preparación RRHH: el recurso de autorización es la participación, y la
-- proyección posterior decide qué documento enlazó Bolsa a esa respuesta.
CREATE FUNCTION vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(
 p_bolsa_ref text,p_participacion_ref text,p_actor_ref text,p_perfil_ref text,
 p_registro_contexto_ref text,p_capacidad bytea,p_decision bytea,p_motivo bytea,
 p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE c jsonb;d jsonb;x jsonb;v record;p record;h text;
BEGIN
 IF vec_documentos.login_justificante_renuncia_rrhh_valido_v1() IS NOT TRUE OR
  current_setting('transaction_isolation')<>'serializable' OR
  current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC' OR
  coalesce(p_bolsa_ref ~ '^[A-Za-z0-9:_-]{1,256}$',false) IS NOT TRUE OR
  coalesce(p_participacion_ref ~ '^[A-Za-z0-9:_-]{1,256}$',false) IS NOT TRUE OR
  nullif(p_actor_ref,'') IS NULL OR nullif(p_perfil_ref,'') IS NULL OR
  nullif(p_registro_contexto_ref,'') IS NULL
 THEN RAISE EXCEPTION 'D22: preparación RRHH denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire THEN
  RAISE EXCEPTION 'D22: material RRHH ilegible' USING ERRCODE='22023'; END;
 h:=encode(sha256(convert_to(format(
  '{"ambitos":{"bolsa_ref":"%s","participacion_ref":"%s"},"atributos":{}}',
  p_bolsa_ref,p_participacion_ref),'UTF8')),'hex');
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' OR
  jsonb_typeof(x) IS DISTINCT FROM 'object' OR
  x->>'principal_ref' IS DISTINCT FROM p_actor_ref OR
  x->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref OR
  c->>'contexto_ref' IS DISTINCT FROM p_registro_contexto_ref OR
  c->>'huella_contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
  c->>'efecto_ref' IS DISTINCT FROM p_participacion_ref OR
  c->>'operacion' IS DISTINCT FROM 'bolsa.justificante_renuncia.descargar_rrhh' OR
  c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1' OR
  c->>'huella_efecto_sha256' IS DISTINCT FROM h OR
  d->>'accion' IS DISTINCT FROM c->>'operacion' OR
  d->>'recurso_ref' IS DISTINCT FROM p_participacion_ref OR
  d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR
  d->>'tipo_recurso' IS DISTINCT FROM 'justificante_renuncia' OR
  d->>'finalidad' IS DISTINCT FROM 'custodiar_justificante_renuncia_bolsa' OR
  d->>'principal_id' IS DISTINCT FROM p_actor_ref OR
  d->>'perfil_activo_ref' IS DISTINCT FROM p_perfil_ref OR
  d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h OR
  d->'campos_permitidos' IS DISTINCT FROM '["contenido"]'::jsonb OR
  d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR
  d#>>'{vinculo_autenticacion_actor,registro_contexto_ref}' IS DISTINCT FROM p_registro_contexto_ref OR
  d#>>'{vinculo_autenticacion_actor,contexto_actor_huella_sha256}' IS DISTINCT FROM
   encode(sha256(p_contexto),'hex') OR
  d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 THEN RAISE EXCEPTION 'D22: preparación RRHH divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_subida_local_bolsa_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS DISTINCT FROM true OR v.efecto_ref IS DISTINCT FROM p_participacion_ref OR
  v.huella_efecto_sha256 IS DISTINCT FROM h OR v.decision_ref IS NULL OR v.auditoria_ref IS NULL
 THEN RAISE EXCEPTION 'D22: consumo RRHH de preparación inválido' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT p FROM vec_bolsa_llamamientos.proyectar_consulta_justificante_renuncia_rrhh_v1(
  p_bolsa_ref,p_participacion_ref,p_actor_ref,p_perfil_ref,p_registro_contexto_ref);
 IF coalesce(p.documento_ref ~ '^ref:[0-9a-f]{64}$',false) IS NOT TRUE OR
  coalesce(p.documento_sha256 ~ '^[0-9a-f]{64}$',false) IS NOT TRUE OR
  coalesce(p.recurso_ref ~ '^justificante-renuncia:[0-9a-f]{64}$',false) IS NOT TRUE OR
  nullif(p.persona_titular_ref,'') IS NULL
 THEN RAISE EXCEPTION 'D22: proyección RRHH divergente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('documento_ref',p.documento_ref,
  'documento_sha256',p.documento_sha256,'recurso_ref',p.recurso_ref,
  'persona_titular_ref',p.persona_titular_ref,
  'decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref);
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(
 text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1(
 text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_documentos_bolsa_rrhh;

RESET ROLE;
DO $post$
DECLARE f oid; nombre text;
BEGIN
 IF (SELECT count(*) FROM pg_constraint WHERE conrelid='vec_documentos.subida_local'::regclass
  AND conname IN ('subida_local_modulo_id_check','subida_local_tipo_recurso_check',
   'subida_local_finalidad_check') AND convalidated)<>3
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE
   conrelid='vec_documentos.subida_local_lectura'::regclass AND
   conname='subida_local_lectura_fase_check' AND convalidated)
 OR has_table_privilege('vec_documentos_bolsa_subida_datos','vec_documentos.subida_local','SELECT,INSERT,UPDATE,DELETE')
 OR has_table_privilege('vec_documentos_bolsa_rrhh','vec_documentos.subida_local','SELECT,INSERT,UPDATE,DELETE')
 OR NOT has_schema_privilege('vec_bolsa_llamamientos_propietario','vec_documentos','USAGE')
 THEN RAISE EXCEPTION 'D22: tabla o ACL abierta' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['reservar_subida_local_v1','confirmar_recepcion_subida_local_v1',
  'iniciar_analisis_subida_local_v1','registrar_analisis_subida_local_v1',
  'cerrar_analisis_subida_local_v1','consultar_subida_local_propia_v1'] LOOP
  f:=to_regprocedure('vec_documentos.'||nombre||'('
   ||'bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL OR NOT has_function_privilege('vec_documentos_subida_datos',f,'EXECUTE')
   OR NOT has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
   OR has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
   OR has_function_privilege('public',f,'EXECUTE')
  THEN RAISE EXCEPTION 'D22: ACL de fachada divergente: %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 f:=to_regprocedure('vec_documentos.consultar_justificante_renuncia_rrhh_v1('
  ||'bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_subida_datos',f,'EXECUTE')
 THEN RAISE EXCEPTION 'D22: ACL RRHH divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_documentos.validar_justificante_renuncia_bolsa_v1(text,text,text,text,text)');
 IF f IS NULL OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
 THEN RAISE EXCEPTION 'D22: ACL validación divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_documentos.resolver_justificante_renuncia_bolsa_v1('
  ||'text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
 THEN RAISE EXCEPTION 'D22: ACL preparación propia divergente' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1('
  ||'text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL OR NOT has_function_privilege('vec_documentos_bolsa_rrhh',f,'EXECUTE')
  OR has_function_privilege('vec_documentos_bolsa_subida_datos',f,'EXECUTE')
  OR has_function_privilege('public',f,'EXECUTE')
 THEN RAISE EXCEPTION 'D22: ACL preparación RRHH divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
