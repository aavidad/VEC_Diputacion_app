\set ON_ERROR_STOP on
BEGIN READ ONLY;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $prueba$
DECLARE h oid:=to_regprocedure('vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(text)');
 r oid:=to_regprocedure('vec_autorizacion.documento_rol_cargo_firma_v1(jsonb,jsonb)');
 anterior jsonb;nuevo jsonb;
BEGIN
 IF h IS NULL OR r IS NULL
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=h AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='e53fa04fe991b61fa15d2655a7a408c87936ca84396050166f9dc2f62d4726cd')
  OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=r AND p.proowner=to_regrole('vec_autorizacion_propietario')
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='1ba7ba69a449973664a37a2875174a809976e4684ca2790477cd62d4d979f1c1')
 THEN RAISE EXCEPTION 'AUT103: constructor o ACL divergentes' USING ERRCODE='55000'; END IF;
 FOR anterior,nuevo IN SELECT vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2(op),
  vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3(op)
  FROM unnest(ARRAY['consultar_detalle','consultar_cuadro','descargar_borrador',
   'consultar_r5','firmar_vec','consultar_circuito']) op LOOP
  IF anterior IS DISTINCT FROM nuevo THEN RAISE EXCEPTION 'AUT103: concesión anterior modificada' USING ERRCODE='55000'; END IF;
 END LOOP;
 anterior:=vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v2('consultar_detalle');
 nuevo:=vec_autorizacion.concesion_v2_cargo_firma_desarrollo_v3('consultar_detalle_selectores');
 IF nuevo-'campos_permitidos' IS DISTINCT FROM anterior-'campos_permitidos'
  OR nuevo->'campos_permitidos' IS DISTINCT FROM '["selectores_recuperacion_firmas"]'::jsonb
 THEN RAISE EXCEPTION 'AUT103: campo adicional divergente' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
