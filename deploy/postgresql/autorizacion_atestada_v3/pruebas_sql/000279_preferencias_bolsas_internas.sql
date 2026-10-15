\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$
DECLARE nucleo oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 interno oid := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 definicion text; ayuda text; audiencia text; lectura text; fallo boolean;
BEGIN
 IF interno IS NULL THEN RAISE EXCEPTION 'AD279: consumidor nominal ausente' USING ERRCODE='55000'; END IF;
 definicion:=pg_get_functiondef(nucleo);
 ayuda:=pg_get_functiondef(interno);
 IF encode(sha256(convert_to(definicion,'UTF8')),'hex') IS DISTINCT FROM
    '6f4c010dbd90ca8570032fd8882fa263baf6d0fe01eb1308074bdfd4e7ed4000'
    OR strpos(definicion,'documentos_bolsa_externa')=0
    OR strpos(definicion,'documentos_bolsa_rrhh')=0
    OR strpos(definicion,'meritos_cartera_retirada')=0
    OR strpos(definicion,'vinculo_oferta_bolsa_ct')=0
    OR strpos(definicion,'cierre_intencion_oferta_ct_bolsa')=0
    OR strpos(definicion,'descarte_seleccion_oferta_ct')=0
    OR strpos(definicion,'p_perfil_mutacion IS NOT DISTINCT FROM ''preferencias_bolsas_interna''')=0
    OR strpos(definicion,'session_user=''vec_interno_bolsa_preferencias_desarrollo''')=0
    OR strpos(definicion,'d#>>''{vinculo_autenticacion_actor,superficie}'' IS NOT DISTINCT FROM ''interna_corporativa''')=0
    OR strpos(definicion,'AND p_perfil_mutacion IS DISTINCT FROM ''preferencias_bolsas_interna''')=0
 THEN RAISE EXCEPTION 'AD279: núcleo nominal o segregación divergentes' USING ERRCODE='55000'; END IF;
 IF (SELECT p.proowner FROM pg_proc p WHERE p.oid=interno) IS DISTINCT FROM
    'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT p.proacl::text FROM pg_proc p WHERE p.oid=interno) IS DISTINCT FROM
    '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
    OR strpos(ayuda,'session_user<>''vec_interno_bolsa_preferencias_desarrollo''')=0
    OR strpos(ayuda,'''vec.bolsa.preferencias-bolsas.ordenar.interna.v1''')=0
    OR strpos(ayuda,'d#>>''{vinculo_autenticacion_actor,superficie}'' IS DISTINCT FROM ''interna_corporativa''')=0
    OR strpos(ayuda,'''preferencias_bolsas_interna'',p_capacidad')=0
    OR EXISTS (SELECT 1 FROM pg_class r WHERE r.oid IN (
      'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass,
      'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass)
      AND (r.relowner<>'vec_autorizacion_atestada_v3_propietario'::regrole
       OR NOT r.relrowsecurity OR NOT r.relforcerowsecurity OR r.relacl IS NOT NULL))
 THEN RAISE EXCEPTION 'AD279: consumidor con otra audiencia, perfil o ACL' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT audiencia FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT lectura FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(audiencia,'UTF8')),'hex') IS DISTINCT FROM
    '46d00817c816d7b0350a24bebe23f2aee697c9f9b93a07fc5af03c9d933de920'
    OR encode(sha256(convert_to(lectura,'UTF8')),'hex') IS DISTINCT FROM
    '87b134b21a26da7c26b89d2125305dd24c96e742a3aced1dbb9e68e94cf7aa9e'
    OR strpos(lectura,'canal = ANY (ARRAY[''externa_personal''::text, ''interna_corporativa''::text])')=0
    OR EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
      AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v1')
 THEN RAISE EXCEPTION 'AD279: restricción de audiencias o lectura divergente' USING ERRCODE='55000'; END IF;

 -- Otra sesión se deniega antes de interpretar material no atestado. El SHA
 -- exacto anterior prueba además que otro perfil y el canal exterior no
 -- tienen una rama positiva en este consumidor.
 fallo:=false;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 EXCEPTION WHEN insufficient_privilege THEN fallo:=true; END;
 IF NOT fallo THEN RAISE EXCEPTION 'AD279: sesión no nominal admitida' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
