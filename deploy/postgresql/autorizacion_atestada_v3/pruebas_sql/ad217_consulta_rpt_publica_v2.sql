\set ON_ERROR_STOP on
-- Sólo sobre clon PG18 con AD216→AD217. No fabrica decisión firmada ni publica
-- un perfil humano. El wrapper y sus GRANT terminan en ROLLBACK.
-- AD172 tiene PK por LOGIN/audiencia/operación: una fila del mismo LOGIN con
-- otra tupla pasa su PK. La preimagen de AD217 debe detectar CUALQUIER fila.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 (login_nombre,audiencia_consumo,operacion,proceso,canal_permitido)
VALUES ('vec_ad217_origen_ajeno_privado','vec_personal.otra.v1',
 'personal.otra.consultar','vec-server','interna_corporativa');
DO $origen_ajeno$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE login_nombre='vec_ad217_origen_ajeno_privado')<>1
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE login_nombre='vec_ad217_origen_ajeno_privado'
   AND audiencia_consumo='vec_personal.rpt_publica.consultar.v2'
   AND operacion='personal.rpt_publica.consultar')
 THEN RAISE EXCEPTION 'AD217 prueba: preexistencia ajena no distingue PK compuesta' USING ERRCODE='55000'; END IF;
END $origen_ajeno$;
ROLLBACK;

BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $contrato$
DECLARE
 nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fachada oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fuente text; check_aud text;
 campos constant text:='["categorias_pendientes_grupo","corte","esquema","estado","evidencia","fuente","huella_sha256","items","limit","offset","publicacion_ref","resumen","total","vista"]';
BEGIN
 IF nucleo IS NULL OR fachada IS NULL THEN RAISE EXCEPTION 'AD217 prueba: funciones ausentes' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
   WHERE login_nombre='vec_ad217_origen_ajeno_privado')
 THEN RAISE EXCEPTION 'AD217 prueba: fixture privada sobrevivió ROLLBACK' USING ERRCODE='55000'; END IF;
 SELECT p.prosrc INTO STRICT fuente FROM pg_proc p WHERE p.oid=nucleo;
 IF encode(sha256(convert_to(pg_get_functiondef(nucleo),'UTF8')),'hex') IS DISTINCT FROM
   'e314eb6242dd0681dbd8cf5618e4287140831f56ab9faa8fe2222f89f0196773'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM
   '1a8271745e5c0cb028ad91d105e69f31a13dd7e046aa31385405d54e5518c9f7'
 OR strpos(fuente,'consulta_rpt_publica_v2_personal')=0
 OR strpos(fuente,'personal.rpt_publica.consultar')=0
 OR strpos(fuente,'vec_personal_rpt_v2_lector')=0
 OR strpos(fuente,campos)=0
 OR strpos(fuente,'resolver_origen_consumo_v1')=0
 OR strpos(fuente,'consumo_confirmado_v4')=0
 OR strpos(fuente,'(CASE WHEN octet_length(d->>''recurso_ref'') BETWEEN 79 AND 200')=0
 THEN RAISE EXCEPTION 'AD217 prueba: núcleo, campos o historia AD216 divergentes' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT check_aud FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(check_aud,'vec_personal.rpt_publica.consultar.v2')=0
 OR strpos(check_aud,'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1')=0
 THEN RAISE EXCEPTION 'AD217 prueba: audiencias nuevas o históricas ausentes' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=nucleo
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=fachada
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(p.proacl) a WHERE p.oid=fachada)<>2
 OR NOT has_function_privilege('vec_personal_propietario',fachada,'EXECUTE')
 OR has_function_privilege('vec_personal_rpt_v2_lector',fachada,'EXECUTE')
 OR has_function_privilege('vec_personal_ejecutor',fachada,'EXECUTE')
 THEN RAISE EXCEPTION 'AD217 prueba: ABI o ACL abrió la fachada' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
    WHERE login_nombre='vec_personal_rpt_v2_app')<>1
 OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
    WHERE login_nombre='vec_personal_rpt_v2_app'
    AND audiencia_consumo='vec_personal.rpt_publica.consultar.v2'
    AND operacion='personal.rpt_publica.consultar'
    AND proceso='vec-server' AND canal_permitido='interna_corporativa')
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
    WHERE login_nombre='vec_personal_rpt_v2_app' AND operacion<>'personal.rpt_publica.consultar')
 THEN RAISE EXCEPTION 'AD217 prueba: origen técnico fuera de tupla' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec.ad217.test_consumos',
  (SELECT count(*)::text FROM vec_autorizacion_atestada_v3.consumo_decision_v3),true);
 PERFORM set_config('vec.ad217.test_auditorias',
  (SELECT count(*)::text FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),true);
END $contrato$;

CREATE FUNCTION vec_autorizacion_atestada_v3.prueba_ad217_origen_privada(p_capacidad bytea,p_decision bytea) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE codigo text;
BEGIN
 IF vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(
   'vec_personal.rpt_publica.consultar.v2','personal.rpt_publica.consultar','interna_corporativa') IS DISTINCT FROM 'vec-server'
 THEN RETURN 'origen_ausente'; END IF;
 BEGIN
  IF p_capacidad IS NULL THEN
   PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
    'consulta_rpt_publica_v2_personal',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  ELSE
   IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
   THEN RETURN 'capacidad_mal_formada'; END IF;
   PERFORM * FROM vec_autorizacion_atestada_v3.consumir_rpt_publica_v2_v3_atestada(
    p_capacidad,p_decision,convert_to('{}','UTF8'),convert_to('{}','UTF8'),
    1,1,convert_to('x','UTF8'),convert_to('x','UTF8'),convert_to('x','UTF8'),
    convert_to(repeat('a',44),'UTF8'));
  END IF;
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS codigo=RETURNED_SQLSTATE;
  RETURN codigo;
 END;
 RETURN 'aceptado';
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.prueba_ad217_origen_privada(bytea,bytea)
 OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad217_origen_privada(bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_rpt_v2_lector;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad217_origen_privada(bytea,bytea)
 TO vec_personal_rpt_v2_lector;
SET SESSION AUTHORIZATION vec_personal_rpt_v2_app;
DO $login$
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
 cap jsonb; dec jsonb; caso text; variacion jsonb; codigo text;
BEGIN
 IF session_user<>'vec_personal_rpt_v2_app' OR current_user<>session_user
 OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
 OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
   AND m.roleid='vec_personal_rpt_v2_lector'::regrole
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 OR has_database_privilege(session_user,current_database(),'TEMP')
 OR vec_autorizacion_atestada_v3.prueba_ad217_origen_privada(NULL,NULL)<>'22023'
 THEN RAISE EXCEPTION 'AD217 prueba: LOGIN, origen o rechazo inválido' USING ERRCODE='55000'; END IF;
 cap:=jsonb_object(claves,array_fill('x'::text,ARRAY[cardinality(claves)]))||jsonb_build_object(
  'operacion','personal.rpt_publica.consultar','audiencia_consumo','vec_personal.rpt_publica.consultar.v2',
  'efecto_ref','rpt-publicada:abc','huella_efecto_sha256',repeat('a',64));
 dec:=jsonb_build_object('concedida',true,'decision_ref','dec_prueba',
  'accion','personal.rpt_publica.consultar','modulo_id','personal',
  'tipo_recurso','rpt_publica_publicacion','finalidad','consultar_organizacion_publicada',
  'recurso_ref','rpt-publicada:abc','contexto_recurso_huella_sha256',repeat('a',64),
  'campos_permitidos','["categorias_pendientes_grupo","corte","esquema","estado","evidencia","fuente","huella_sha256","items","limit","offset","publicacion_ref","resumen","total","vista"]'::jsonb,
  'obligaciones','[]'::jsonb,'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'));
 FOREACH caso IN ARRAY ARRAY['campos_solo_metadatos','recurso_ajeno','recurso_formato',
  'accion_ajena','modulo_ajeno','finalidad_ajena','audiencia_ajena','huella_ajena',
  'superficie_ajena','obligacion_extra'] LOOP
  variacion:=dec;
  IF caso='campos_solo_metadatos' THEN variacion:=jsonb_set(dec,'{campos_permitidos}','["corte","huella_sha256","publicacion_ref"]'::jsonb);
  ELSIF caso='recurso_ajeno' THEN variacion:=jsonb_set(dec,'{recurso_ref}','"rpt-publicada:otra"'::jsonb);
  ELSIF caso='recurso_formato' THEN variacion:=jsonb_set(dec,'{recurso_ref}','"rpt-publicada:ABC"'::jsonb);
  ELSIF caso='accion_ajena' THEN variacion:=jsonb_set(dec,'{accion}','"personal.rpt_publica.editar"'::jsonb);
  ELSIF caso='modulo_ajeno' THEN variacion:=jsonb_set(dec,'{modulo_id}','"bolsa"'::jsonb);
  ELSIF caso='finalidad_ajena' THEN variacion:=jsonb_set(dec,'{finalidad}','"consultar_persona"'::jsonb);
  ELSIF caso='huella_ajena' THEN variacion:=jsonb_set(dec,'{contexto_recurso_huella_sha256}',to_jsonb(repeat('b',64)));
  ELSIF caso='superficie_ajena' THEN variacion:=jsonb_set(dec,'{vinculo_autenticacion_actor,superficie}','"externa_personal"'::jsonb);
  ELSIF caso='obligacion_extra' THEN variacion:=jsonb_set(dec,'{obligaciones}','["imprimir"]'::jsonb);
  END IF;
  IF caso='audiencia_ajena' THEN
   codigo:=vec_autorizacion_atestada_v3.prueba_ad217_origen_privada(
    convert_to((cap||jsonb_build_object('audiencia_consumo','vec_personal.otra.v1'))::text,'UTF8'),
    convert_to(variacion::text,'UTF8'));
  ELSE
   codigo:=vec_autorizacion_atestada_v3.prueba_ad217_origen_privada(
    convert_to(cap::text,'UTF8'),convert_to(variacion::text,'UTF8'));
  END IF;
  IF codigo<>'42501' THEN RAISE EXCEPTION 'AD217 prueba: % obtuvo % en vez de 42501',caso,codigo USING ERRCODE='55000'; END IF;
 END LOOP;
END $login$;
RESET SESSION AUTHORIZATION;
DO $sin_efecto$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)
    IS DISTINCT FROM current_setting('vec.ad217.test_consumos',true)::bigint
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)
    IS DISTINCT FROM current_setting('vec.ad217.test_auditorias',true)::bigint
 THEN RAISE EXCEPTION 'AD217 prueba: negativo produjo consumo o auditoría' USING ERRCODE='55000'; END IF;
END $sin_efecto$;
ROLLBACK;
