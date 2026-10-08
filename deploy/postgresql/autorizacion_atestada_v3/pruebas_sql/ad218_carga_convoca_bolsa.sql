\set ON_ERROR_STOP on
-- Sólo en copia PG18 POST216→AD218. Sin capacidad ni decisión firmada;
-- wrapper y GRANTs privados terminan en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $contrato$
DECLARE
 nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fachada oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 fuente text; check_aud text;
BEGIN
 IF nucleo IS NULL OR fachada IS NULL THEN RAISE EXCEPTION 'AD218 prueba: funciones ausentes' USING ERRCODE='55000'; END IF;
 SELECT p.prosrc INTO STRICT fuente FROM pg_proc p WHERE p.oid=nucleo;
 IF encode(sha256(convert_to(pg_get_functiondef(nucleo),'UTF8')),'hex') IS DISTINCT FROM
   'c74551eab17bea78bb564d14d96b77f33bd24a5874b7bdb6e100a59d8f2fe714'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM
   '79d2f29752235a01716d49095777fe8e890a6deed5269a8647d1f866d4b671b5'
 OR strpos(fuente,'carga_convoca_bolsa')=0
 OR strpos(fuente,'bolsa.carga_convoca.confirmar')=0
 OR strpos(fuente,'vec_bolsa_llamamientos.carga_convoca.confirmar.v1')=0
 OR strpos(fuente,'vec_bolsa_llamamientos_ejecutor')=0
 OR strpos(fuente,'^acta:importacion-convoca:[0-9a-f]{64}$')=0
 OR strpos(fuente,'resolver_origen_consumo_v1')=0
 OR strpos(fuente,'consumo_confirmado_v4')=0
 OR strpos(fuente,'(CASE WHEN octet_length(d->>''recurso_ref'') BETWEEN 79 AND 200')=0
 OR strpos(fuente,'personal.rpt_publica.consultar')<>0
 THEN RAISE EXCEPTION 'AD218 prueba: núcleo, historia o tupla B1 divergentes' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT check_aud FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(check_aud,'vec_bolsa_llamamientos.carga_convoca.confirmar.v1')=0
 OR strpos(check_aud,'vec_bolsa_llamamientos.rrhh.candidatos.consultar.v1')=0
 OR strpos(check_aud,'vec_personal.rpt_publica.consultar.v2')<>0
 THEN RAISE EXCEPTION 'AD218 prueba: CHECK de audiencias divergente' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=nucleo
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=fachada
    AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(p.proacl) a WHERE p.oid=fachada)<>2
 OR NOT has_function_privilege('vec_bolsa_llamamientos_propietario',fachada,'EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',fachada,'EXECUTE')
 THEN RAISE EXCEPTION 'AD218 prueba: ABI o ACL abrió la fachada' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
    WHERE login_nombre='vec_bolsa_llamamientos_desarrollo'
      AND audiencia_consumo='vec_bolsa_llamamientos.carga_convoca.confirmar.v1'
      AND operacion='bolsa.carga_convoca.confirmar')<>1
 OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
    WHERE login_nombre='vec_bolsa_llamamientos_desarrollo'
    AND audiencia_consumo='vec_bolsa_llamamientos.carga_convoca.confirmar.v1'
    AND operacion='bolsa.carga_convoca.confirmar'
    AND proceso='vec-server' AND canal_permitido='interna_corporativa')
 THEN RAISE EXCEPTION 'AD218 prueba: origen técnico B1 incorrecto' USING ERRCODE='55000'; END IF;
 PERFORM set_config('vec.ad218.test_consumos',
  (SELECT count(*)::text FROM vec_autorizacion_atestada_v3.consumo_decision_v3),true);
 PERFORM set_config('vec.ad218.test_auditorias',
  (SELECT count(*)::text FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),true);
END $contrato$;

CREATE FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_b1_privada(p_capacidad bytea,p_decision bytea) RETURNS text
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE codigo text;
BEGIN
 IF vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(
   'vec_bolsa_llamamientos.carga_convoca.confirmar.v1','bolsa.carga_convoca.confirmar','interna_corporativa') IS DISTINCT FROM 'vec-server'
 THEN RETURN 'origen_ausente'; END IF;
 IF vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(
   'vec_bolsa_llamamientos.carga_convoca.confirmar.v1','bolsa.carga_convoca.confirmar','externa_personal') IS NOT NULL
 THEN RETURN 'canal_abierto'; END IF;
 BEGIN
  IF p_capacidad IS NULL THEN
   PERFORM * FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
    'carga_convoca_bolsa',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  ELSE
   IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad) IS NOT TRUE
   THEN RETURN 'capacidad_mal_formada'; END IF;
   PERFORM * FROM vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(
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
ALTER FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_b1_privada(bytea,bytea)
 OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_b1_privada(bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.prueba_ad218_b1_privada(bytea,bytea)
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
 cap jsonb; dec jsonb; variacion jsonb; caso text; codigo text;
 ref text:='acta:importacion-convoca:'||repeat('a',64);
BEGIN
 IF session_user<>'vec_bolsa_llamamientos_desarrollo' OR current_user<>session_user
 OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
 OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
   AND m.roleid='vec_bolsa_llamamientos_ejecutor'::regrole
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 OR vec_autorizacion_atestada_v3.prueba_ad218_b1_privada(NULL,NULL)<>'22023'
 THEN RAISE EXCEPTION 'AD218 prueba: LOGIN, origen o rechazo inválido' USING ERRCODE='55000'; END IF;
 cap:=jsonb_object(claves,array_fill('x'::text,ARRAY[cardinality(claves)]))||jsonb_build_object(
  'operacion','bolsa.carga_convoca.confirmar','audiencia_consumo','vec_bolsa_llamamientos.carga_convoca.confirmar.v1',
  'efecto_ref',ref,'huella_efecto_sha256',repeat('b',64));
 dec:=jsonb_build_object('concedida',true,'decision_ref','dec_prueba',
  'accion','bolsa.carga_convoca.confirmar','modulo_id','bolsa',
  'tipo_recurso','carga_convoca','finalidad','carga_bolsa_convoca',
  'recurso_ref',ref,'contexto_recurso_huella_sha256',repeat('b',64),
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,
  'vinculo_autenticacion_actor',jsonb_build_object('superficie','interna_corporativa'));
 FOREACH caso IN ARRAY ARRAY['accion','audiencia','modulo','tipo','finalidad','recurso','hex','huella','campos','obligaciones','superficie'] LOOP
  variacion:=dec;
  IF caso='accion' THEN variacion:=jsonb_set(dec,'{accion}','"bolsa.carga_convoca.editar"'::jsonb);
  ELSIF caso='modulo' THEN variacion:=jsonb_set(dec,'{modulo_id}','"contratacion_temporal"'::jsonb);
  ELSIF caso='tipo' THEN variacion:=jsonb_set(dec,'{tipo_recurso}','"consulta_candidatos_bolsa"'::jsonb);
  ELSIF caso='finalidad' THEN variacion:=jsonb_set(dec,'{finalidad}','"consulta_rrhh_bolsas"'::jsonb);
  ELSIF caso='recurso' THEN variacion:=jsonb_set(dec,'{recurso_ref}',to_jsonb('acta:importacion-convoca:'||repeat('c',64)));
  ELSIF caso='hex' THEN variacion:=jsonb_set(dec,'{recurso_ref}',to_jsonb('acta:importacion-convoca:'||repeat('A',64)));
  ELSIF caso='huella' THEN variacion:=jsonb_set(dec,'{contexto_recurso_huella_sha256}',to_jsonb(repeat('c',64)));
  ELSIF caso='campos' THEN variacion:=jsonb_set(dec,'{campos_permitidos}','["datos"]'::jsonb);
  ELSIF caso='obligaciones' THEN variacion:=jsonb_set(dec,'{obligaciones}','["imprimir"]'::jsonb);
  ELSIF caso='superficie' THEN variacion:=jsonb_set(dec,'{vinculo_autenticacion_actor,superficie}','"externa_personal"'::jsonb);
  END IF;
  IF caso='audiencia' THEN
   codigo:=vec_autorizacion_atestada_v3.prueba_ad218_b1_privada(
    convert_to((cap||jsonb_build_object('audiencia_consumo','vec_bolsa_llamamientos.otra.v1'))::text,'UTF8'),
    convert_to(variacion::text,'UTF8'));
  ELSE
   codigo:=vec_autorizacion_atestada_v3.prueba_ad218_b1_privada(
    convert_to(cap::text,'UTF8'),convert_to(variacion::text,'UTF8'));
  END IF;
  IF codigo<>'42501' THEN RAISE EXCEPTION 'AD218 prueba: % obtuvo % en vez de 42501',caso,codigo USING ERRCODE='55000'; END IF;
 END LOOP;
END $negativos$;
RESET SESSION AUTHORIZATION;
DO $sin_efecto$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3)
   IS DISTINCT FROM current_setting('vec.ad218.test_consumos',true)::bigint
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3)
   IS DISTINCT FROM current_setting('vec.ad218.test_auditorias',true)::bigint
 THEN RAISE EXCEPTION 'AD218 prueba: denegación creó consumo o auditoría' USING ERRCODE='55000'; END IF;
END $sin_efecto$;
ROLLBACK;
