\set ON_ERROR_STOP on
-- Sólo tras AD178 y AD177 instaladas sobre la postimagen de AD193.
-- Prueba estructural y de rechazos: no crea consumos, no fabrica filas
-- favorables y no acredita el recorrido causal con productores reales.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
DO $estructura$
DECLARE f oid;s text;nombre text;
BEGIN
 -- Núcleo y CHECK: postimágenes medidas de AD178.
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF (SELECT encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex'))
    IS DISTINCT FROM '2ccd704afe6140d604faa626631e9743edda8f785517d1136054c746cb9b1381'
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
    IS DISTINCT FROM '4729b6666065a8a3582443d803bf8535940f0eae650b27a315aca260f3183b8b'
 THEN RAISE EXCEPTION 'AD177/178 prueba: núcleo divergente'; END IF;
 IF (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated)
    IS DISTINCT FROM 'e76428d2ecd1c79da88a827cf33138f5c75026ed2138858c67f93420e932eae7'
 THEN RAISE EXCEPTION 'AD177/178 prueba: CHECK de audiencias divergente'; END IF;
 -- Los dos comprobadores nuevos usan el sello AD193 y la familia v4, nunca xmin.
 FOREACH nombre IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)',
  'vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)'] LOOP
  SELECT prosrc INTO STRICT s FROM pg_proc WHERE oid=nombre::regprocedure;
  IF strpos(s,'xmin')>0
  OR strpos(s,'consumo_origen IS DISTINCT FROM pg_current_xact_id()')=0
  OR strpos(s,'auditoria_origen IS DISTINCT FROM pg_current_xact_id()')=0
  OR strpos(s,'IS DISTINCT FROM ''consumo_confirmado_v4''')=0
  OR strpos(s,'version_consumo IS DISTINCT FROM 4')=0
  OR strpos(s,'consumo_confirmado_v3')>0
  THEN RAISE EXCEPTION 'AD177/178 prueba: comprobador sin sello v4 %',nombre; END IF;
 END LOOP;
 -- Ningún LOGIN ni el runtime CT ejecuta directamente los comprobadores o el núcleo,
 -- salvo por herencia de un grupo propietario ya autorizado (p. ej. el LOGIN de gobierno H9).
 FOREACH nombre IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(bytea,jsonb)',
  'vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)',
  'vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(jsonb)',
  'vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(bytea)',
  'vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.consumir_recuperacion_firmas_r5_ct_v2_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
  IF EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolcanlogin AND NOT r.rolsuper
     AND has_function_privilege(r.oid,nombre::regprocedure,'EXECUTE')
     AND NOT pg_has_role(r.oid,'vec_autorizacion_atestada_v3_propietario','USAGE')
     AND NOT pg_has_role(r.oid,'vec_autorizacion_propietario','USAGE')
     AND NOT pg_has_role(r.oid,'vec_catalogos_configurables_propietario','USAGE')
     AND NOT pg_has_role(r.oid,'vec_contratacion_temporal_propietario','USAGE'))
  OR has_function_privilege('vec_contratacion_temporal_ejecutor',nombre::regprocedure,'EXECUTE')
  THEN RAISE EXCEPTION 'AD177/178 prueba: EXECUTE inesperado en %',nombre; END IF;
 END LOOP;
 -- La fachada exterior de gobierno es la única abierta al runtime CT (grupo, no LOGIN directo).
 IF NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'EXECUTE')
 OR (SELECT array_agg(a.grantee::regrole::text ORDER BY a.grantee::regrole::text) FROM pg_proc p
     CROSS JOIN LATERAL aclexplode(p.proacl) a
     WHERE p.oid='vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
    IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario','vec_contratacion_temporal_ejecutor']
 THEN RAISE EXCEPTION 'AD177/178 prueba: ACL de la fachada de gobierno divergente'; END IF;
 -- Ninguna fila histórica recibió sello: sin backfill.
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
   WHERE tipo_registro<>'consumo_confirmado_v4' AND transaccion_origen IS NOT NULL)
 THEN RAISE EXCEPTION 'AD177/178 prueba: sello en familia histórica'; END IF;
END $estructura$;

-- Rechazos con un consumo real comprometido en otra transacción (sello NULL).
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $rechazos$
DECLARE r record;consumo jsonb;material bytea;contexto text;
BEGIN
 SELECT a.decision_ref,a.efecto_ref,a.huella_efecto_sha256,a.consumo_huella_sha256,u.auditoria_ref,a.consumida_en
 INTO r FROM vec_autorizacion_atestada_v3.consumo_decision_v3 a
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u
  ON u.decision_ref=a.decision_ref AND u.efecto_ref=a.efecto_ref AND u.huella_efecto_sha256=a.huella_efecto_sha256
 ORDER BY a.consumida_en DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'AD177/178 prueba: el clon no tiene consumos previos'; END IF;
 consumo:=jsonb_build_object('decision_ref',r.decision_ref,'efecto_ref',r.efecto_ref,
  'huella_efecto_sha256',r.huella_efecto_sha256,'consumo_huella_sha256',r.consumo_huella_sha256,
  'auditoria_ref',r.auditoria_ref,'consumida_en',r.consumida_en,'consumo_nuevo',true);
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(consumo);
  RAISE EXCEPTION 'AD177/178 prueba: gobierno aceptó un consumo de otra transacción';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 -- Variante con consumo_nuevo ausente, propiedad extra y tipo erróneo.
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1((consumo-'consumo_nuevo')||'{"transaccion_origen":"1"}');
  RAISE EXCEPTION 'AD177/178 prueba: gobierno aceptó un recibo con sello del cliente';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 -- Recuperación R5: el material liga expediente y contexto; la fila es de otra transacción.
 material:=convert_to('{"CatalogoHuella":"'||repeat('a',64)||'","ClaveIdempotencia":"prueba-ad178-000001","Documento":"contrato","ExpedienteRef":"'||r.efecto_ref||
  '","FirmantePrincipalCandidatoRef":"per_prueba","OrganizacionRef":"org-prueba","PasoOrden":1,"UnidadRef":null,"VersionExpediente":1,"Via":"certificado_vec"}','UTF8');
 contexto:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"org-prueba"},"atributos":{"material_sha256":"'||encode(sha256(material),'hex')||'"}}','UTF8')),'hex');
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_consumo_recuperacion_firmas_ct_v2(material,
   consumo||jsonb_build_object('huella_efecto_sha256',contexto,'consumida_en',r.consumida_en::text));
  RAISE EXCEPTION 'AD177/178 prueba: recuperación aceptó un consumo ajeno';
 EXCEPTION WHEN insufficient_privilege THEN
  -- Debe llegar a la relectura de filas, no caer antes por un error del validador.
  IF SQLERRM NOT LIKE '%clave=filas%' THEN
   RAISE EXCEPTION 'AD177/178 prueba: rechazo de recuperación por causa inesperada: %',SQLERRM;
  END IF;
 END;
 -- Fachada de gobierno: material malformado se rechaza antes de consumir.
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(
   convert_to('{}','UTF8'),convert_to('{}','UTF8'),convert_to('{}','UTF8'),NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD177/178 prueba: fachada aceptó material vacío';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
END $rechazos$;
ROLLBACK;
SELECT 'AD177-AD178-POST193-ESTRUCTURA-Y-RECHAZOS-OK; sin consumo positivo ni recorrido causal';
