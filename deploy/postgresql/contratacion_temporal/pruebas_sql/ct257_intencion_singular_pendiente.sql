\set ON_ERROR_STOP on
-- CT257: la clave O6 singular solo se consulta después del consumo V3.
-- El positivo se recorre con sesión y decisión reales; esta prueba comprueba
-- que una llamada sin ellas se deniega antes de leer la intención.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE f oid; cuerpo text; estado text; acl text;
BEGIN
 f:=to_regprocedure('vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 SELECT p.proacl::text INTO acl FROM pg_proc p WHERE p.oid=f;
 IF f IS NULL OR acl IS DISTINCT FROM
    '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
 THEN RAISE EXCEPTION 'CT257: permiso de lectura abierto'; END IF;
 cuerpo:=pg_get_functiondef(f);
 IF position('registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(' IN cuerpo)=0
    OR position('registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(' IN cuerpo)>
       position('numero_personas_detalle_rrhh_ct225(' IN cuerpo)
 THEN RAISE EXCEPTION 'CT257: lectura O6 anterior al consumo V3'; END IF;
 BEGIN
  PERFORM vec_contratacion_temporal.consultar_estado_oferta_expediente_ct225(
   '{}',NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'CT257: lectura sin decisión aceptada';
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS estado=RETURNED_SQLSTATE;
  IF estado<>'42501' THEN RAISE; END IF;
 END;
END $prueba$;
ROLLBACK;
