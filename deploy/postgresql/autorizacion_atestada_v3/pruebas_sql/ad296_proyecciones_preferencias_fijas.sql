\set ON_ERROR_STOP on
-- Focal estructural tras AD296, en copia desechable. No simula autoridad V3.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $focal$
DECLARE x record;p pg_proc%ROWTYPE;f regprocedure;
 b8 jsonb:='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","recibo_ref","registrado_en"]';
 m5 jsonb:='["version","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]';
 u10 jsonb:='["version","orden_guardado","bolsas_actuales","bolsas_nuevas","bolsas_retiradas","revision_requerida","modalidades_guardadas","modalidades_disponibles","recibo_ref","registrado_en"]';
 m5_otro_orden jsonb:='["modalidades_guardadas","version","modalidades_disponibles","recibo_ref","registrado_en"]';
 m6_extra jsonb:='["version","modalidades_guardadas","modalidades_disponibles","contacto","recibo_ref","registrado_en"]';
BEGIN
 FOR x IN SELECT * FROM (VALUES
  ('registrar_lectura_preferencias_bolsas_v1','014ea7fbf8229619c736d0ebcea7964634dbd7c545b9b6a76d659d5a2a334bf8'),
  ('registrar_lectura_preferencias_bolsas_interna_v1','0a6ec140c5b7b4c52a2242269418397c527f446fc21c131af91a7cd0d728c5dc'),
  ('registrar_intento_lectura_preferencias_bolsas_v1','819b55746d49c058c718bd4e69bfa41477391d72ca7ce16d6854b57532de7d27'),
  ('registrar_intento_lectura_preferencias_bolsas_interna_v1','473150abcd18ec6b34c7ac46aad63260867093f14cdd6528c61b1e7a51ddf23c')
 ) v(nombre,huella) LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||x.nombre||'(bytea,bytea,jsonb)');
  IF f IS NULL THEN RAISE EXCEPTION 'AD296: falta %',x.nombre; END IF;
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
   OR p.prosecdef IS NOT TRUE OR p.provolatile<>'v' OR p.proparallel<>'u'
   OR p.proacl::text IS DISTINCT FROM
    '{vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario,vec_bolsa_llamamientos_propietario=X/vec_autorizacion_atestada_v3_propietario}'
   OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=10s']
   OR has_function_privilege('public',f,'EXECUTE')
   OR obj_description(f,'pg_proc') IS DISTINCT FROM 'vec.f1-13.preferencias.proyecciones-fijas.8-5.v1'
   OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM x.huella
  THEN RAISE EXCEPTION 'AD296: definición, comentario o ACL divergentes %',x.nombre; END IF;
 END LOOP;
 -- PostgreSQL conserva el orden de elementos JSONB: solo las dos capturas
 -- pactadas son iguales a los literales que guardan ambas fachadas positivas.
 IF b8 IS NOT DISTINCT FROM m5 OR u10 IN (b8,m5) OR m5_otro_orden IN (b8,m5)
 OR m6_extra IN (b8,m5) OR b8 IS NULL OR m5 IS NULL
 THEN RAISE EXCEPTION 'AD296: proyección ajena admitida'; END IF;
END $focal$;
ROLLBACK;
