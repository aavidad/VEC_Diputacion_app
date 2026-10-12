\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $test$
DECLARE d jsonb;una jsonb;ajena jsonb;canon text;
BEGIN
 d:=jsonb_build_object(
  'rol_id','persona_meritos_cartera','version',1,'nombre','areaPersonal.meritos.rolCartera',
  'concesiones',jsonb_build_array(
   jsonb_build_object('accion','meritos.cartera.consultar_propia','modulo_id','meritos',
    'tipo_recurso','cartera_personal','finalidades',jsonb_build_array('gestion_cartera_propia'),
    'garantia_minima','alto','campos_permitidos',jsonb_build_array(
     'denominacion','disponibilidad','hecho_ref','retirada','tipo','total','version_hecho'),
    'obligaciones',jsonb_build_array('auditar')),
   jsonb_build_object('accion','meritos.cartera.retirar_propio','modulo_id','meritos',
    'tipo_recurso','hecho','finalidades',jsonb_build_array('gestion_cartera_propia'),
    'garantia_minima','alto','campos_permitidos',jsonb_build_array('disponibilidad','recibo'),
    'obligaciones',jsonb_build_array('auditar'))));
 canon:=vec_autorizacion.canon_gobierno_rol_nuevo_v1(d,'definicion');
 IF canon IS NULL OR convert_from(convert_to(canon,'UTF8'),'UTF8')::jsonb IS DISTINCT FROM d
 THEN RAISE EXCEPTION 'AUT80: canon de dos concesiones divergente' USING ERRCODE='55000';END IF;
 una:=jsonb_set(d,'{concesiones}',jsonb_build_array(d#>'{concesiones,0}'));
 BEGIN
  PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(una,'definicion');
  RAISE EXCEPTION 'AUT80: rol M4 incompleto aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 ajena:=jsonb_set(d,'{rol_id}','"otro_rol_ordinario"'::jsonb);
 BEGIN
  PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(ajena,'definicion');
  RAISE EXCEPTION 'AUT80: dos concesiones ajenas aceptadas' USING ERRCODE='55000';
 EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
 una:=jsonb_set(ajena,'{concesiones}',jsonb_build_array(d#>'{concesiones,0}'));
 PERFORM vec_autorizacion.canon_gobierno_rol_nuevo_v1(una,'definicion');
 IF to_regprocedure('vec_autorizacion.validar_plan_rol_nuevo_v1(jsonb,boolean)') IS NULL
 THEN RAISE EXCEPTION 'AUT80: validador ausente' USING ERRCODE='55000';END IF;
END $test$;
ROLLBACK;
