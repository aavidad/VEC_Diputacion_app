\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone='UTC';

DO $prueba$
DECLARE saltos bigint; tres bigint; total bigint;
BEGIN
 IF current_user='vec_bolsa_llamamientos_propietario' THEN
  RAISE EXCEPTION 'la focal debe ejecutarse con el rol runtime de lectura' USING ERRCODE='55000';
 END IF;
 SELECT count(*) FILTER(WHERE tipo='salto_orden') INTO saltos
   FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(clock_timestamp());
 SELECT count(*),count(*) FILTER(WHERE tipo='tres_anos') INTO total,tres
   FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(clock_timestamp()+interval '3 years');
 IF saltos<1 OR total<2 OR tres<1 THEN
  RAISE EXCEPTION 'D6 no conserva el salto B7 o la proyección reproducible del periodo B2' USING ERRCODE='55000';
 END IF;
 IF EXISTS (
   SELECT 1 FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v1(clock_timestamp()+interval '3 years')
    WHERE tipo NOT IN ('salto_orden','tres_anos') OR bolsa_ref IS NULL OR referencia IS NULL OR fecha IS NULL
       OR jsonb_typeof(detalle)<>'object' OR detalle ?| ARRAY['nombre','dni','documento','correo','telefono']
 ) THEN
  RAISE EXCEPTION 'contrato D3-AV inválido o con datos personales' USING ERRCODE='55000';
 END IF;
END $prueba$;

ROLLBACK;
