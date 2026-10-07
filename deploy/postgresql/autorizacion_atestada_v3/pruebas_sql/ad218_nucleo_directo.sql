\set ON_ERROR_STOP on
-- POST218: llama al núcleo con el perfil B1 sin pasar por su fachada.
-- Capacidad canónica de forma, con huellas sintéticas y sin firma: el núcleo
-- llega a su ligadura cerrada, rechaza y no crea historia. No prueba un positivo.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF f IS NULL OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
   'c74551eab17bea78bb564d14d96b77f33bd24a5874b7bdb6e100a59d8f2fe714'
 OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f)
   IS DISTINCT FROM '79d2f29752235a01716d49095777fe8e890a6deed5269a8647d1f866d4b671b5'
 OR to_regprocedure('vec_autorizacion_atestada_v3.prueba_ad218_nucleo_directo_privada(text,jsonb,jsonb,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD218 núcleo directo: POST218 o fixture incompatible' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec.ad218.directo_consumos',
  (SELECT count(*)::text FROM vec_autorizacion_atestada_v3.consumo_decision_v3),true);
 PERFORM set_config('vec.ad218.directo_auditorias',
  (SELECT count(*)::text FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),true);
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_nucleo_directo_privada(
 perfil text,c jsonb,d jsonb,x jsonb) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE codigo text; mensaje text; capacidad bytea;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' THEN RETURN 'owner_ajeno'; END IF;
 IF vec_autorizacion_atestada_v3.capacidad_tipos_validos(c) IS NOT TRUE THEN RETURN 'tipos_invalidos'; END IF;
 capacidad:=vec_autorizacion_atestada_v3.capacidad_canonica(c);
 IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(capacidad) IS NOT TRUE THEN RETURN 'canon_invalido'; END IF;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
   perfil,capacidad,convert_to(d::text,'UTF8'),convert_to('{}','UTF8'),
   convert_to(x::text,'UTF8'),1,1,convert_to('x','UTF8'),
   convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to(repeat('a',44),'UTF8'));
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE, mensaje=MESSAGE_TEXT;
  RETURN codigo||'|'||mensaje;
 END;
 RETURN 'aceptado';
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_nucleo_directo_privada(text,jsonb,jsonb,jsonb)
 OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_nucleo_directo_privada(text,jsonb,jsonb,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_nucleo_directo_privada(text,jsonb,jsonb,jsonb)
 TO vec_bolsa_llamamientos_ejecutor;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $negativos$
DECLARE
 claves text[]:=ARRAY['esquema','version','clave_id','clave_version','revision_gobierno',
  'huella_gobierno_sha256','emisor_id','audiencia_consumo','nonce','emitida_en','expira_en',
  'decision_ref','huella_decision_sha256','huella_motivo_sha256','huella_payload_vec_ad_3_sha256',
  'huella_sobre_cose_sign1_sha256','huella_prueba_confianza_sha256','contexto_ref',
  'huella_contexto_sha256','audiencia_despliegue','operacion','efecto_ref',
  'huella_efecto_sha256','decision_valida_hasta','verificada_en','revision_confianza',
  'configuracion_secuencia','huella_configuracion_sha256','configuracion_publicada_en',
  'configuracion_expira_en','raiz_clave_id','raiz_version','huella_raiz_spki_sha256',
  'raiz_valida_desde','raiz_valida_hasta','suite','mac_sha256'];
 c jsonb; d jsonb; x jsonb; cc jsonb; dd jsonb; caso text; codigo text;
 ref text:='acta:importacion-convoca:'||repeat('a',64);
BEGIN
 IF session_user<>'vec_bolsa_llamamientos_desarrollo' OR current_user<>session_user
 OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
 THEN RAISE EXCEPTION 'AD218 núcleo directo: identidad de sesión inválida' USING ERRCODE='55000'; END IF;
 c:=jsonb_object(claves,array_fill('x'::text,ARRAY[cardinality(claves)]))||jsonb_build_object(
  'esquema','vec.autorizacion.capacidad-registro-consumo-atestado.v3',
  'version',3,'clave_version',1,'revision_gobierno',1,'configuracion_secuencia',1,'raiz_version',1,
  'suite','VEC-AD-3-COSE-EDDSA-1','nonce',repeat('a',64),
  'operacion','bolsa.carga_convoca.confirmar',
  'audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1',
  'efecto_ref',ref,'huella_efecto_sha256',repeat('b',64));
 d:=jsonb_build_object('concedida',true,'decision_ref','dec_prueba',
  'accion','bolsa.carga_convoca.confirmar','modulo_id','bolsa','tipo_recurso','carga_convoca',
  'finalidad','carga_bolsa_convoca','recurso_ref',ref,
  'contexto_recurso_huella_sha256',repeat('b',64),
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'));
 x:=jsonb_build_object('esquema','vec.contexto-actor.vinculado.v2','persona_version',1,
  'perfil_version',1,'principal_ref','per_prueba','perfil_activo_ref','perfil_prueba');
 FOREACH caso IN ARRAY ARRAY['tupla_b1','perfil_ajeno','accion','audiencia','modulo','tipo','finalidad','recurso','campos','superficie'] LOOP
  cc:=c; dd:=d;
  IF caso='accion' THEN dd:=jsonb_set(d,'{accion}','"bolsa.carga_convoca.editar"'::jsonb);
  ELSIF caso='audiencia' THEN cc:=jsonb_set(c,'{audiencia_consumo}','"vec_bolsa_llamamientos.otra.v1"'::jsonb);
  ELSIF caso='modulo' THEN dd:=jsonb_set(d,'{modulo_id}','"personal"'::jsonb);
  ELSIF caso='tipo' THEN dd:=jsonb_set(d,'{tipo_recurso}','"otro"'::jsonb);
  ELSIF caso='finalidad' THEN dd:=jsonb_set(d,'{finalidad}','"consultar"'::jsonb);
  ELSIF caso='recurso' THEN dd:=jsonb_set(d,'{recurso_ref}',to_jsonb('acta:importacion-convoca:'||repeat('c',64)));
  ELSIF caso='campos' THEN dd:=jsonb_set(d,'{campos_permitidos}','["datos"]'::jsonb);
  ELSIF caso='superficie' THEN dd:=jsonb_set(d,'{vinculo_autenticacion_actor,superficie}','"externa_personal"'::jsonb);
  END IF;
  codigo:=vec_autorizacion_atestada_v3.prueba_ad218_nucleo_directo_privada(
   CASE WHEN caso='perfil_ajeno' THEN 'consulta_rrhh_bolsa' ELSE 'carga_convoca_bolsa' END,cc,dd,x);
  IF codigo IS DISTINCT FROM '22023|ligadura VEC-AD-3 inválida' THEN
   RAISE EXCEPTION 'AD218 núcleo directo: % no alcanzó la ligadura cerrada (%)',caso,codigo USING ERRCODE='55000'; END IF;
 END LOOP;
END $negativos$;
RESET SESSION AUTHORIZATION;
DO $sin_efecto$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)
   IS DISTINCT FROM current_setting('vec.ad218.directo_consumos',true)::bigint
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)
   IS DISTINCT FROM current_setting('vec.ad218.directo_auditorias',true)::bigint
 THEN RAISE EXCEPTION 'AD218 núcleo directo: se creó consumo o auditoría sin firma' USING ERRCODE='55000'; END IF;
END $sin_efecto$;
ROLLBACK;
