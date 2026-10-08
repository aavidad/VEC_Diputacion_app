\set ON_ERROR_STOP on
BEGIN;
CREATE FUNCTION pg_temp.ct193_solicitud(
 p_catalogo bytea, p_causa text, p_periodo jsonb,
 p_campos jsonb, p_jornada integer
) RETURNS jsonb LANGUAGE sql IMMUTABLE SET search_path=pg_catalog
AS $funcion$
 SELECT pg_catalog.jsonb_build_object(
   'motivo_clave',p_causa,'periodo',p_periodo,
   'necesidad',pg_catalog.jsonb_build_object(
     'esquema','vec.ct.necesidad_alta.v1',
     'catalogo_ref','catalogo:ct:paridad',
     'catalogo_version',1,
     'catalogo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(p_catalogo),'hex'),
     'causa_clave',p_causa,'periodo',p_periodo,
     'jornada_minutos',p_jornada,'campos',p_campos,
     'catalogo_instantanea',pg_catalog.replace(
       pg_catalog.encode(p_catalogo,'base64'),E'\n','')))
$funcion$;

DO $paridad$
DECLARE
 c jsonb;
 raw bytea;
 raw_limite_mutado bytea;
 presupuestos jsonb;
 rpt jsonb;
 campos_vacante jsonb;
 campos_sustitucion jsonb;
 campos_programa jsonb;
 campos_acumulacion jsonb;
 p_finito jsonb := '{"inicio":"2026-10-01","fin":"2027-09-30"}'::jsonb;
 p_abierto jsonb;
 p_acumulacion jsonb := '{"inicio":"2026-05-31","fin":"2027-02-28"}'::jsonb;
 s jsonb;
BEGIN
 presupuestos:=pg_catalog.jsonb_build_object(
   'organica_codigo','100','funcional_codigo','200',
   'proyecto_gasto_codigo','300','porcentaje_financiacion','100');
 rpt:=pg_catalog.jsonb_build_object(
   'puesto_codigo','3388','rpt_catalogo_ref','catalogo:rpt:prueba',
   'rpt_catalogo_huella_sha256',pg_catalog.repeat('a',64));
 campos_vacante:=presupuestos||rpt;
 campos_sustitucion:=presupuestos||rpt;
 campos_programa:=presupuestos||pg_catalog.jsonb_build_object(
   'programa_denominacion','Refuerzo temporal',
   'programa_fin','2027-10-01','proyecto_codigo','P01',
   'financiacion_ref','financiacion:opaca:1','rc_ref','rc:opaca:1');
 campos_acumulacion:=presupuestos||pg_catalog.jsonb_build_object(
   'justificacion_temporal','Refuerzo limitado.');
 c:=pg_catalog.jsonb_build_object(
   'esquema','vec.ct.necesidades_alta.v1',
   'referencia','catalogo:ct:paridad','version',1,'es_ejemplo',true,
   'fuente_ref','circular:ct:prueba','fuente_url','https://www.dipgra.es/ct',
   'jornada_referencia_minutos',2250,
   'jornada_fuente_ref','operador:ct:prueba',
   'causas',pg_catalog.jsonb_build_array(
     pg_catalog.jsonb_build_object(
       'clave','vacante','etiqueta_clave','ct.necesidad.vacante',
       'fuente_ref','circular:ct:prueba','fuente_url','https://www.dipgra.es/ct',
       'regla_ref','regla:ct:vacante:v1','fecha_fin','obligatoria',
       'maximo_meses',36,
       'campos_permitidos','["plaza_codigo","puesto_codigo","rpt_catalogo_ref","rpt_catalogo_huella_sha256","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb,
       'campos_obligatorios','["puesto_codigo","rpt_catalogo_ref","rpt_catalogo_huella_sha256","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb),
     pg_catalog.jsonb_build_object(
       'clave','sustitucion','etiqueta_clave','ct.necesidad.sustitucion',
       'fuente_ref','circular:ct:prueba','fuente_url','https://www.dipgra.es/ct',
       'regla_ref','regla:ct:sustitucion:v1','fecha_fin','opcional',
       'causa_fin','reincorporacion_titular','maximo_meses',36,
       'campos_permitidos','["puesto_codigo","plaza_codigo","rpt_catalogo_ref","rpt_catalogo_huella_sha256","titular_ref","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb,
       'campos_obligatorios','["puesto_codigo","rpt_catalogo_ref","rpt_catalogo_huella_sha256","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb),
     pg_catalog.jsonb_build_object(
       'clave','programa_temporal','etiqueta_clave','ct.necesidad.programa',
       'fuente_ref','circular:ct:prueba','fuente_url','https://www.dipgra.es/ct',
       'regla_ref','regla:ct:programa:v1','fecha_fin','obligatoria',
       'maximo_meses',36,
       'campos_permitidos','["programa_denominacion","programa_fin","proyecto_codigo","financiacion_ref","rc_ref","intervencion_ref","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb,
       'campos_obligatorios','["programa_denominacion","programa_fin","proyecto_codigo","financiacion_ref","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb,
       'uno_de','[["rc_ref","intervencion_ref"]]'::jsonb),
     pg_catalog.jsonb_build_object(
       'clave','acumulacion_tareas','etiqueta_clave','ct.necesidad.acumulacion',
       'fuente_ref','circular:ct:prueba','fuente_url','https://www.dipgra.es/ct',
       'regla_ref','regla:ct:acumulacion:v1','fecha_fin','obligatoria',
       'maximo_meses',9,'ventana_meses',18,
       'campos_permitidos','["justificacion_temporal","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb,
       'campos_obligatorios','["justificacion_temporal","organica_codigo","funcional_codigo","proyecto_gasto_codigo","porcentaje_financiacion"]'::jsonb)));
 raw:=pg_catalog.convert_to(c::text,'UTF8');
 raw_limite_mutado:=pg_catalog.convert_to(
   pg_catalog.jsonb_set(c,'{causas,3,maximo_meses}','8'::jsonb)::text,'UTF8');
 p_abierto:=pg_catalog.jsonb_build_object(
   'inicio','2026-10-01','causa_fin','reincorporacion_titular',
   'politica_fin',pg_catalog.jsonb_build_object(
      'regla_ref','regla:ct:sustitucion:v1','catalogo_version',1,
      'catalogo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(raw),'hex'),
      'fecha_fin','opcional','causa_fin','reincorporacion_titular'));

 s:=pg_temp.ct193_solicitud(raw,'vacante',p_finito,campos_vacante,2250);
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(s) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
       pg_temp.ct193_solicitud(raw,'vacante',p_finito,
         campos_vacante-'puesto_codigo',2250)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
       pg_temp.ct193_solicitud(raw,'vacante',p_finito,
         campos_vacante||pg_catalog.jsonb_build_object('titular_ref','persona:opaca:1'),2250)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
       pg_temp.ct193_solicitud(raw,'vacante',p_finito,
         campos_vacante||pg_catalog.jsonb_build_object('rpt_catalogo_version','1'),2250)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
       pg_temp.ct193_solicitud(raw,'vacante',p_finito,campos_vacante,10081)) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT193: vacante, RPT, campo inesperado o jornada divergente';
 END IF;
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'sustitucion',p_abierto,campos_sustitucion,2250)) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'sustitucion',p_abierto,campos_sustitucion,2400)) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'sustitucion',p_abierto,campos_sustitucion,10081)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'sustitucion',p_abierto,
        campos_sustitucion-'puesto_codigo',2250)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'sustitucion',p_abierto-'politica_fin',
        campos_sustitucion,2250)) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT193: sustitución o política de fin divergente';
 END IF;
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'programa_temporal',p_finito,campos_programa,1800)) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'programa_temporal',p_finito,
        campos_programa-'rc_ref',1800)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'programa_temporal',p_finito,
        campos_programa||pg_catalog.jsonb_build_object('intervencion_ref','intervencion:opaca:1'),1800)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'programa_temporal',p_finito,
        pg_catalog.jsonb_set(campos_programa,'{programa_fin}','"2027-01-01"'::jsonb),1800)) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT193: programa, vía única o fin divergente';
 END IF;
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'acumulacion_tareas',p_acumulacion,
        campos_acumulacion,1800)) IS NOT TRUE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
      pg_temp.ct193_solicitud(raw,'acumulacion_tareas',
        pg_catalog.jsonb_set(p_acumulacion,'{fin}','"2027-03-01"'::jsonb),
        campos_acumulacion,1800)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
       pg_temp.ct193_solicitud(raw_limite_mutado,'acumulacion_tareas',
         p_acumulacion,campos_acumulacion,1800)) IS NOT FALSE
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(
       pg_temp.ct193_solicitud(raw,'vacante',p_acumulacion,
         campos_acumulacion,1800)) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT193: límite de mes de acumulación divergente';
 END IF;
END $paridad$;
ROLLBACK;
