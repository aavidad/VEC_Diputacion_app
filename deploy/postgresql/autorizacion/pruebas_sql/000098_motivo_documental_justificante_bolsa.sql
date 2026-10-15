\set ON_ERROR_STOP on
-- AUT98: postimagen exacta. Ejecutar tras la migración en el clon de la principal.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='20s';
DO $prueba$
DECLARE f record; actual pg_catalog.pg_proc%ROWTYPE; cuerpo text;
BEGIN
 FOR f IN SELECT * FROM (VALUES
  ('vec_autorizacion.resolver_motivo_candidato_externo_v1(text,integer,text,text,timestamptz)',
   '46fcbb5aa9bb611f84ec4a45ba27fca6e21cc2632ba9904b7a8899695ce5bd32',
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_motivos_externos=X/vec_autorizacion_propietario}',false),
  ('vec_autorizacion.registrar_decision_candidato_externo_v3(bytea,bytea,numeric,numeric)',
   'aaf798aa5d9e532fc5d4ea644f0e6d888ae97fd5699d38b377e0ce5d1f0b41ca',
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_registro_externo=X/vec_autorizacion_propietario}',true),
  ('vec_autorizacion.registrar_y_revalidar_decision_contexto_actor_externa_v3(bytea,bytea,numeric,numeric)',
   'e273354738a2382c534c0dba6e5554c2bb6f421a215152bdf767099341f134e4',
   '{vec_autorizacion_propietario=X/vec_autorizacion_propietario,vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_propietario}',true)
 ) AS esperado(firma,huella,acl,registro)
 LOOP
  SELECT * INTO STRICT actual FROM pg_proc WHERE oid=to_regprocedure(f.firma);
  cuerpo:=pg_get_functiondef(actual.oid);
  IF actual.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
     OR actual.prosecdef IS NOT TRUE
     OR actual.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::text[]
     OR actual.proacl::text IS DISTINCT FROM f.acl
     OR encode(sha256(convert_to(cuerpo,'UTF8')),'hex') IS DISTINCT FROM f.huella
     OR strpos(cuerpo,'motivos_autorizacion_justificante_renuncia_bolsa')=0
     OR strpos(cuerpo,'motivos_mi_bolsa_desarrollo')=0
     OR strpos(cuerpo,'motivos_historial_mi_bolsa_desarrollo')=0
     OR strpos(cuerpo,'motivos_portal_mi_bolsa_desarrollo')=0
     OR strpos(cuerpo,'motivos_inscripcion_bolsa_desarrollo')=0
     OR strpos(cuerpo,'motivos_alegaciones_bolsa')=0
     OR (f.registro AND (strpos(cuerpo,'bolsa.justificante_renuncia.subir_propio')=0
       OR strpos(cuerpo,'bolsa.justificante_renuncia.descargar_propio')=0
       OR strpos(cuerpo,'justificante_renuncia')=0
       OR strpos(cuerpo,'custodiar_justificante_renuncia_bolsa')=0))
     OR strpos(cuerpo,'bolsa.justificante_renuncia.descargar_rrhh')>0
  THEN RAISE EXCEPTION 'AUT98: postimagen o límite documental divergente %',f.firma USING ERRCODE='55000'; END IF;
 END LOOP;
END $prueba$;
ROLLBACK;
