\set ON_ERROR_STOP on
-- Ejecutar únicamente después de 000006 en PostgreSQL 18 desechable.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE sin_vehiculo jsonb; con_vehiculo jsonb; resultado jsonb;
        calculo jsonb; comando jsonb; comando_vacio jsonb; gasto jsonb;
        tramo_dieta jsonb; linea_dieta jsonb; material text; regla jsonb;
BEGIN
 material:='{"esquema":"vec.dietas.borrador-operacion.v2","operacion":"editar","recurso_ref":"dco_DDDDDDDDDDDDDDDDDDDDDD","identidad":{"persona_ref":"per_AAAAAAAAAAAAAAAAAAAAAA","empleado_ref":"emp_BBBBBBBBBBBBBBBBBBBBBB","relacion_ref":"rel_CCCCCCCCCCCCCCCCCCCCCC","unidad_ref":"U1","relacion_version":1},"huella_semantica":"prueba","comando":{"clave_idempotencia":"clave_0000000000000001","version_esperada":1,"relacion_ref":"rel_CCCCCCCCCCCCCCCCCCCCCC"},"referencia":"dco_DDDDDDDDDDDDDDDDDDDDDD"}';
 IF vec_dietas.huella_semantica_mutacion_v2(material)
    IS DISTINCT FROM '764aa2ed69d3fe85bee049313a28a076a7b8c9b3fd6384e9bdc8d66cacf3a286' THEN
  RAISE EXCEPTION 'D2: canon Go de huella semántica v2 incompatible'; END IF;
 sin_vehiculo:='{"vehiculo_propio":false,"rutas":[],"calculo":{"tramos_ruta":[],"rutas":[],"kilometros":"0.0000","importe_kilometraje_centimos":0}}'::jsonb;
 resultado:=vec_dietas.validar_rutas_d4_v2(sin_vehiculo,0.2600,'provisional:rd462:20260923','PROVISIONAL · pendiente de confirmación por RRHH');
 IF resultado IS DISTINCT FROM '{"lineas":[],"importe_centimos":0}'::jsonb THEN
  RAISE EXCEPTION 'D4: sin vehículo no conserva cero exacto'; END IF;
 IF vec_dietas.validar_rutas_d4_v2(jsonb_set(sin_vehiculo,'{calculo,vehiculo_propio}','true'::jsonb),
    0.2600,'provisional:rd462:20260923','PROVISIONAL · pendiente de confirmación por RRHH') IS NOT NULL THEN
  RAISE EXCEPTION 'D4: vehículo contradictorio aceptado'; END IF;
 con_vehiculo:='{"vehiculo_propio":true,"rutas":[{"codigos_ruta":["GR1","GR2"],"ajuste_kilometros":"-0.5000","motivo_ajuste":"Desvío acreditado"}],"calculo":{"vehiculo_propio":true,"version_grafo":"grafo_sintetico_1","tramos_ruta":[],"rutas":[{"codigos_ruta":["GR1","GR2"],"version_grafo":"grafo_sintetico_1","tramos_ruta":[{"origen_codigo":"GR1","destino_codigo":"GR2","kilometros":"1.0000"}],"kilometros_base":"1.0000","ajuste_kilometros":"-0.5000","motivo_ajuste":"Desvío acreditado","kilometros_finales":"0.5000","importe_centimos":13}],"kilometros":"0.5000","importe_kilometraje_centimos":13}}'::jsonb;
 resultado:=vec_dietas.validar_rutas_d4_v2(con_vehiculo,0.2600,'provisional:rd462:20260923','PROVISIONAL · pendiente de confirmación por RRHH');
 IF resultado IS NULL OR resultado->>'importe_centimos'<>'13'
    OR jsonb_array_length(resultado->'lineas')<>1
    OR resultado->'lineas'->0->>'ajuste_kilometros'<>'-0.5000' THEN
  RAISE EXCEPTION 'D4: ajuste justificado y redondeo rechazados'; END IF;
 IF vec_dietas.validar_rutas_d4_v2(jsonb_set(con_vehiculo,'{calculo,rutas,0,importe_centimos}','12'::jsonb),
    0.2600,'provisional:rd462:20260923','PROVISIONAL · pendiente de confirmación por RRHH') IS NOT NULL THEN
  RAISE EXCEPTION 'D4: importe alterado aceptado'; END IF;
 IF vec_dietas.validar_rutas_d4_v2(jsonb_set(con_vehiculo,'{rutas,0,motivo_ajuste}','""'::jsonb),
    0.2600,'provisional:rd462:20260923','PROVISIONAL · pendiente de confirmación por RRHH') IS NOT NULL THEN
  RAISE EXCEPTION 'D4: ajuste sin motivo aceptado'; END IF;

 calculo:='{"procedencia":"sin_vehiculo_propio","version_grafo":"no_aplica","motor":"no_aplica","version_tarifa":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH","hora_inicio":"08:00","hora_fin":"12:00","kilometros":"0.0000","eur_por_km":"0.2600","importe_kilometraje_centimos":0,"tramos_ruta":[],"opciones_dieta":[{"grupo":1,"calculo":{"tramos":[],"manutencion_centimos":0,"alojamiento_tope_centimos":0,"total_maximo_orientativo_centimos":0,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}},{"grupo":2,"calculo":{"tramos":[],"manutencion_centimos":0,"alojamiento_tope_centimos":0,"total_maximo_orientativo_centimos":0,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}},{"grupo":3,"calculo":{"tramos":[],"manutencion_centimos":0,"alojamiento_tope_centimos":0,"total_maximo_orientativo_centimos":0,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}}]}'::jsonb;
 -- El cálculo lleva la regla de devengo publicada en el catálogo de 000006.
 regla:=vec_dietas.consultar_regla_devengo_dietas_v1('provisional:rd462:20260923',DATE '2026-09-23','ES','nacional_ordinaria');
 calculo:=calculo||jsonb_build_object('regla_ref',regla->>'regla_ref','regla_huella_sha256',regla->>'huella_sha256');
 comando:=jsonb_build_object('fecha_inicio','2026-09-23','fecha_fin','2026-09-23',
  'hora_inicio','08:00','hora_fin','12:00','codigos_ruta',jsonb_build_array('GR1','GR2'),
  'vehiculo_propio',false,'rutas','[]'::jsonb,'asignacion',jsonb_build_object('grupo_dieta',2),
  'tramos_aceptados','[]'::jsonb,'version_tarifa_aceptada','provisional:rd462:20260923',
  'calculo',calculo,'documento',jsonb_build_object('vehiculo_propio',false,
   'grupo_dieta',2,'version_tarifa_aceptada','provisional:rd462:20260923',
   'tramos_aceptados','[]'::jsonb,'lineas','[]'::jsonb,
   'manutencion_centimos',0,'alojamiento_tope_centimos',0,
   'kilometraje_centimos',0,'otros_centimos',0,'total_orientativo_centimos',0));
 IF vec_dietas.validar_documento_v2(comando) IS NOT TRUE THEN
  RAISE EXCEPTION 'D3: documento vacío de grupo acreditado rechazado'; END IF;
 IF vec_dietas.validar_documento_v2(jsonb_set(comando,'{documento,grupo_dieta}','3'::jsonb)) IS NOT FALSE THEN
  RAISE EXCEPTION 'D3: grupo distinto de Personal aceptado'; END IF;
 comando_vacio:=comando;
 tramo_dieta:='{"fecha":"2026-09-23","tipo":"manutencion","porcentaje":50,"importe_centimos":1870,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}'::jsonb;
 linea_dieta:='{"tipo":"dieta","grupo":2,"indice_tramo":0,"fecha":"2026-09-23","concepto":"manutencion","importe_centimos":1870,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}'::jsonb;
 comando:=jsonb_set(comando,'{hora_fin}','"17:00"'::jsonb);
 comando:=jsonb_set(comando,'{calculo,hora_fin}','"17:00"'::jsonb);
 comando:=jsonb_set(comando,'{calculo,opciones_dieta,1,calculo,tramos}',jsonb_build_array(tramo_dieta));
 comando:=jsonb_set(comando,'{calculo,opciones_dieta,1,calculo,manutencion_centimos}','1870'::jsonb);
 comando:=jsonb_set(comando,'{calculo,opciones_dieta,1,calculo,total_maximo_orientativo_centimos}','1870'::jsonb);
 comando:=jsonb_set(comando,'{tramos_aceptados}','[0]'::jsonb);
 comando:=jsonb_set(comando,'{documento,tramos_aceptados}','[0]'::jsonb);
 comando:=jsonb_set(comando,'{documento,lineas}',jsonb_build_array(linea_dieta));
 comando:=jsonb_set(comando,'{documento,manutencion_centimos}','1870'::jsonb);
 comando:=jsonb_set(comando,'{documento,total_orientativo_centimos}','1870'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT TRUE THEN
  RAISE EXCEPTION 'D3: tramo único del grupo Personal rechazado'; END IF;
 comando:=jsonb_set(comando,'{tramos_aceptados}','[0,0]'::jsonb);
 comando:=jsonb_set(comando,'{documento,tramos_aceptados}','[0,0]'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT FALSE THEN
  RAISE EXCEPTION 'D3: índice duplicado aceptado'; END IF;
 comando:=comando_vacio;
 gasto:='{"tipo":"otro_gasto","concepto":"Peaje sintético","importe_centimos":1000,"justificante_ref":"","justificante_sha256":""}'::jsonb;
 comando:=jsonb_set(comando,'{otros}',jsonb_build_array(gasto));
 comando:=jsonb_set(comando,'{documento,lineas}',jsonb_build_array(gasto));
 comando:=jsonb_set(comando,'{documento,otros_centimos}','1000'::jsonb);
 comando:=jsonb_set(comando,'{documento,total_orientativo_centimos}','1000'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT TRUE THEN
  RAISE EXCEPTION 'D5: otro gasto sin justificante rechazado'; END IF;
 comando:=jsonb_set(comando,'{otros,0,tipo}','"otro_medio"'::jsonb);
 comando:=jsonb_set(comando,'{documento,lineas,0,tipo}','"otro_medio"'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT TRUE THEN
  RAISE EXCEPTION 'D5: otro medio con texto justificativo sin fichero rechazado'; END IF;
 comando:=jsonb_set(comando,'{otros,0,justificante_ref}','"justificante:real"'::jsonb);
 comando:=jsonb_set(comando,'{documento,lineas,0,justificante_ref}','"justificante:real"'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT FALSE THEN
  RAISE EXCEPTION 'D5: referencia sin huella aceptada'; END IF;
END $prueba$;
ROLLBACK;
