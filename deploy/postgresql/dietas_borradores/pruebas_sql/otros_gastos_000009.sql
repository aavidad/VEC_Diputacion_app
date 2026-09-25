\set ON_ERROR_STOP on
-- Ejecutar únicamente después de 000009 en PostgreSQL 18 desechable.
-- Casos de validar_documento_v2 para líneas D5 de otros medios y gastos.
BEGIN;
SET LOCAL ROLE vec_dietas_propietario;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE calculo jsonb; base jsonb; comando jsonb; medio jsonb; gasto jsonb; regla jsonb;
        caso record;
BEGIN
 calculo:='{"procedencia":"sin_vehiculo_propio","version_grafo":"no_aplica","motor":"no_aplica","version_tarifa":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH","hora_inicio":"08:00","hora_fin":"12:00","kilometros":"0.0000","eur_por_km":"0.2600","importe_kilometraje_centimos":0,"tramos_ruta":[],"opciones_dieta":[{"grupo":1,"calculo":{"tramos":[],"manutencion_centimos":0,"alojamiento_tope_centimos":0,"total_maximo_orientativo_centimos":0,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}},{"grupo":2,"calculo":{"tramos":[],"manutencion_centimos":0,"alojamiento_tope_centimos":0,"total_maximo_orientativo_centimos":0,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}},{"grupo":3,"calculo":{"tramos":[],"manutencion_centimos":0,"alojamiento_tope_centimos":0,"total_maximo_orientativo_centimos":0,"version_tarifa_ref":"provisional:rd462:20260923","rotulo":"PROVISIONAL · pendiente de confirmación por RRHH"}}]}'::jsonb;
 regla:=vec_dietas.consultar_regla_devengo_dietas_v1('provisional:rd462:20260923',DATE '2026-09-23','ES','nacional_ordinaria');
 calculo:=calculo||jsonb_build_object('regla_ref',regla->>'regla_ref','regla_huella_sha256',regla->>'huella_sha256');
 base:=jsonb_build_object('fecha_inicio','2026-09-23','fecha_fin','2026-09-24',
  'hora_inicio','08:00','hora_fin','12:00','codigos_ruta',jsonb_build_array('GR1','GR2'),
  'vehiculo_propio',false,'rutas','[]'::jsonb,'asignacion',jsonb_build_object('grupo_dieta',2),
  'tramos_aceptados','[]'::jsonb,'version_tarifa_aceptada','provisional:rd462:20260923',
  'calculo',calculo,'documento',jsonb_build_object('vehiculo_propio',false,
   'grupo_dieta',2,'version_tarifa_aceptada','provisional:rd462:20260923',
   'tramos_aceptados','[]'::jsonb,'lineas','[]'::jsonb,
   'manutencion_centimos',0,'alojamiento_tope_centimos',0,
   'kilometraje_centimos',0,'otros_centimos',0,'total_orientativo_centimos',0));
 IF vec_dietas.validar_documento_v2(base) IS NOT TRUE THEN
  RAISE EXCEPTION 'D5: documento sin otros gastos rechazado'; END IF;
 medio:='{"tipo":"otro_medio","tipo_gasto":"taxi","catalogo_version":"provisional:otros-gastos:20260925","fecha":"2026-09-23","concepto":"Taxi estación a sede","importe_centimos":1250,"justificante_ref":"ticket:taxi-0923","justificante_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}'::jsonb;
 gasto:='{"tipo":"otro_gasto","tipo_gasto":"aparcamiento","catalogo_version":"provisional:otros-gastos:20260925","fecha":"2026-09-24","concepto":"Aparcamiento sede","importe_centimos":600,"justificante_ref":"ticket:parking-0924","justificante_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}'::jsonb;
 comando:=jsonb_set(base,'{otros}',jsonb_build_array(medio,gasto));
 comando:=jsonb_set(comando,'{documento,lineas}',jsonb_build_array(medio,gasto));
 comando:=jsonb_set(comando,'{documento,otros_centimos}','1850'::jsonb);
 comando:=jsonb_set(comando,'{documento,total_orientativo_centimos}','1850'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT TRUE THEN
  RAISE EXCEPTION 'D5: medio y gasto justificados rechazados'; END IF;
 -- Cada alteración se aplica igual en la declaración y en la línea, salvo
 -- los casos que prueban precisamente su discrepancia o el total.
 FOR caso IN SELECT * FROM (VALUES
  ('forma anterior sin tipo ni fecha','{"tipo":"otro_medio","concepto":"Taxi estación a sede","importe_centimos":1250,"justificante_ref":"","justificante_sha256":""}'::jsonb),
  ('sin justificante',medio||'{"justificante_ref":"","justificante_sha256":""}'),
  ('referencia sin huella',medio||'{"justificante_sha256":""}'),
  ('huella en mayúsculas',medio||jsonb_build_object('justificante_sha256',repeat('B',64))),
  ('referencia con espacio',medio||'{"justificante_ref":"ticket taxi"}'),
  ('tipo fuera del catálogo',medio||'{"tipo_gasto":"restaurante"}'),
  ('apartado incoherente',medio||'{"tipo":"otro_gasto"}'),
  ('versión no publicada',medio||'{"catalogo_version":"provisional:otros-gastos:20990101"}'),
  ('fecha antes de la salida',medio||'{"fecha":"2026-09-22"}'),
  ('fecha tras el regreso',medio||'{"fecha":"2026-09-25"}'),
  ('fecha inexistente',medio||'{"fecha":"2026-02-30"}'),
  ('sin fecha',medio-'fecha'),
  ('clave desconocida',medio||'{"importe_eur":"12.50"}'),
  ('descripción corta',medio||'{"concepto":"tx"}'),
  ('descripción con tabulador',medio||jsonb_build_object('concepto','Taxi'||chr(9)||'sede')),
  ('importe cero',medio||'{"importe_centimos":0}'),
  ('importe decimal',medio||'{"importe_centimos":12.5}'),
  ('importe desmesurado',medio||'{"importe_centimos":100000001}')) AS t(nombre,linea) LOOP
  comando:=jsonb_set(base,'{otros}',jsonb_build_array(caso.linea));
  comando:=jsonb_set(comando,'{documento,lineas}',jsonb_build_array(caso.linea));
  comando:=jsonb_set(comando,'{documento,otros_centimos}',coalesce(caso.linea->'importe_centimos','0'::jsonb));
  comando:=jsonb_set(comando,'{documento,total_orientativo_centimos}',coalesce(caso.linea->'importe_centimos','0'::jsonb));
  IF vec_dietas.validar_documento_v2(comando) IS NOT FALSE THEN
   RAISE EXCEPTION 'D5: aceptado caso «%»', caso.nombre; END IF;
 END LOOP;
 comando:=jsonb_set(base,'{otros}',jsonb_build_array(medio));
 comando:=jsonb_set(comando,'{documento,lineas}',jsonb_build_array(medio||'{"importe_centimos":1300}'));
 comando:=jsonb_set(comando,'{documento,otros_centimos}','1300'::jsonb);
 comando:=jsonb_set(comando,'{documento,total_orientativo_centimos}','1300'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT FALSE THEN
  RAISE EXCEPTION 'D5: línea distinta de la declaración aceptada'; END IF;
 comando:=jsonb_set(base,'{otros}',jsonb_build_array(medio));
 comando:=jsonb_set(comando,'{documento,lineas}',jsonb_build_array(medio));
 comando:=jsonb_set(comando,'{documento,otros_centimos}','1250'::jsonb);
 comando:=jsonb_set(comando,'{documento,total_orientativo_centimos}','1200'::jsonb);
 IF vec_dietas.validar_documento_v2(comando) IS NOT FALSE THEN
  RAISE EXCEPTION 'D5: total que no suma el gasto aceptado'; END IF;
END $prueba$;
ROLLBACK;
