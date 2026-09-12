\set ON_ERROR_STOP on
-- Comprobación focal de catálogo tras CT91, por canal de lectura autorizado.
-- No ejecuta funciones de negocio, no captura V3 ni prueba como propietario.
-- No acredita todavía autorización, concurrencia o descarga por navegador.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
DO $catalogo$
DECLARE
 original text; nueva text; llamada text; sustituta text; huella text;
 p pg_proc%ROWTYPE; q pg_proc%ROWTYPE;
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 lector oid:='vec_contratacion_temporal_consultor_rrhh'::regrole;
BEGIN
 FOR original,nueva,llamada,sustituta,huella IN VALUES
  ('motor_consultar_detalle_rrhh_v1','motor_consultar_original_propuesta_rrhh_v1',
   'materializar_detalle_rrhh_v1','materializar_original_propuesta_rrhh_v1',
   'f85bdf81cdc3e30b0b1acf0af60252d61fcdeed20aef4031a5cbe6312c1af4d0'),
  ('consultar_detalle_rrhh_atestado_v1','consultar_original_propuesta_rrhh_atestado_v1',
   'motor_consultar_detalle_rrhh_v1','motor_consultar_original_propuesta_rrhh_v1',
   '7b7d6c4a419262d54ddb7f2a096e5a4d6e1c962bc58e3027546e221717ed1814')
 LOOP
  SELECT * INTO STRICT p FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace AND proname=original;
  SELECT * INTO STRICT q FROM pg_proc WHERE pronamespace=p.pronamespace AND proname=nueva;
  IF encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM huella
   OR q.prosrc IS DISTINCT FROM replace(p.prosrc,llamada,sustituta)
   OR (to_jsonb(q)-ARRAY['oid','proname','prosrc','proacl'])
      IS DISTINCT FROM (to_jsonb(p)-ARRAY['oid','proname','prosrc','proacl']) THEN
   RAISE EXCEPTION 'CT91: cuerpo o atributos divergentes';
  END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace
  AND proname='materializar_detalle_rrhh_v1'
  AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='d7ba4ebd03ef8ddd10ca2b464547011a7508ffae401e1611c35810409323dfad') THEN
  RAISE EXCEPTION 'CT91: consulta ordinaria alterada';
 END IF;
 IF (SELECT count(*) FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace
  AND proname IN ('materializar_original_propuesta_rrhh_v1','motor_consultar_original_propuesta_rrhh_v1',
                 'consultar_original_propuesta_rrhh_atestado_v1'))<>3 THEN
  RAISE EXCEPTION 'CT91: inventario divergente';
 END IF;
 FOR p IN SELECT * FROM pg_proc WHERE pronamespace='vec_contratacion_temporal'::regnamespace
  AND proname IN ('materializar_original_propuesta_rrhh_v1','motor_consultar_original_propuesta_rrhh_v1',
                 'consultar_original_propuesta_rrhh_atestado_v1')
 LOOP
  IF p.proowner<>propietario OR NOT p.prosecdef OR p.proparallel<>'u'
   OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC',
      'lock_timeout=1s','statement_timeout=4s','idle_in_transaction_session_timeout=6s']::text[]
   OR EXISTS(SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>propietario
     OR (a.grantee<>propietario AND NOT(a.grantee=lector AND p.proname='consultar_original_propuesta_rrhh_atestado_v1')))
   OR has_function_privilege(lector,p.oid,'EXECUTE')
      IS DISTINCT FROM (p.proname='consultar_original_propuesta_rrhh_atestado_v1') THEN
   RAISE EXCEPTION 'CT91: atributos o ACL incompatibles';
  END IF;
 END LOOP;
END
$catalogo$;
ROLLBACK;
