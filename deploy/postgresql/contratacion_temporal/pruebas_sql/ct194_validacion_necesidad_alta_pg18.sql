\set ON_ERROR_STOP on
BEGIN;
DO $prueba$
DECLARE
 raw bytea := pg_catalog.convert_to(
   '{"esquema":"vec.ct.necesidades_alta.v1","referencia":"catalogo:ct:ct194","version":1,"jornada_referencia_minutos":2250,"causas":[{"clave":"sustitucion","regla_ref":"regla:ct:sustitucion:v1","fecha_fin":"opcional","causa_fin":"reincorporacion_titular","maximo_meses":36,"campos_permitidos":["numero_personas","puesto_codigo","titular_ref","rpt_catalogo_huella_sha256","porcentaje_financiacion","justificacion_temporal","programa_fin"],"campos_obligatorios":["porcentaje_financiacion"]}]}',
   'UTF8');
 n jsonb;
 s jsonb;
 mutada jsonb;
 caso record;
 lector text;
BEGIN
 n:=pg_catalog.jsonb_build_object(
   'esquema','vec.ct.necesidad_alta.v1',
   'catalogo_ref','catalogo:ct:ct194',
   'catalogo_version',1,
   'catalogo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(raw),'hex'),
   'causa_clave','sustitucion',
   'periodo',pg_catalog.jsonb_build_object('inicio','2026-10-01','fin','2026-10-31'),
   'jornada_minutos',2250,
   'campos',pg_catalog.jsonb_build_object('porcentaje_financiacion','50'),
   'catalogo_instantanea',pg_catalog.replace(pg_catalog.encode(raw,'base64'),E'\n',''));
 s:=pg_catalog.jsonb_build_object('motivo_clave','sustitucion',
      'periodo',n->'periodo','necesidad',n);
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(s) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT194: base y catálogo anterior con número opcional rechazados';
 END IF;
 FOR caso IN
   SELECT * FROM (VALUES
      ('porcentaje 1','porcentaje_financiacion','1',true),
      ('porcentaje 100','porcentaje_financiacion','100',true),
      ('porcentaje 0','porcentaje_financiacion','0',false),
      ('porcentaje 01','porcentaje_financiacion','01',false),
      ('porcentaje 101','porcentaje_financiacion','101',false),
      ('porcentaje +1','porcentaje_financiacion','+1',false),
      ('porcentaje decimal','porcentaje_financiacion','50.5',false),
      ('codigo valido','puesto_codigo','A.1/2',true),
      ('codigo con dos puntos','puesto_codigo','A:1',false),
      ('codigo largo','puesto_codigo',pg_catalog.repeat('A',81),false),
      ('referencia valida','titular_ref','persona:abc',true),
      ('referencia corta','titular_ref','a:',false),
      ('referencia larga','titular_ref',pg_catalog.repeat('a',161),false),
      ('huella valida','rpt_catalogo_huella_sha256',pg_catalog.repeat('a',64),true),
      ('huella vacia','rpt_catalogo_huella_sha256',pg_catalog.repeat('0',64),false),
      ('huella mayuscula','rpt_catalogo_huella_sha256',pg_catalog.repeat('A',64),false),
      ('texto 4000','justificacion_temporal',pg_catalog.repeat('ñ',4000),true),
      ('texto 4001','justificacion_temporal',pg_catalog.repeat('ñ',4001),false),
      ('texto salto interior','justificacion_temporal',E'valor\nparte',true),
      ('texto salto inicial','justificacion_temporal',E'\nvalor',false),
      ('texto tab final','justificacion_temporal',E'valor\t',false),
      ('texto control','justificacion_temporal','valor'||chr(7),false),
      ('texto NFD','justificacion_temporal','e'||U&'\0301',false),
      ('programa fecha valida','programa_fin','2026-11-01',true),
      ('programa formato','programa_fin','01/11/2026',false)
   ) AS v(nombre,clave,valor,esperado)
 LOOP
   mutada:=pg_catalog.jsonb_set(s,'{necesidad,campos}',
     (s#>'{necesidad,campos}') ||
     pg_catalog.jsonb_build_object(caso.clave,caso.valor));
   IF vec_contratacion_temporal.necesidad_alta_valida_v3(mutada)
      IS DISTINCT FROM caso.esperado THEN
     RAISE EXCEPTION 'CT194: caso % esperado=% actual=%',caso.nombre,
       caso.esperado,vec_contratacion_temporal.necesidad_alta_valida_v3(mutada);
   END IF;
 END LOOP;
 SELECT pg_catalog.pg_get_functiondef(pg_catalog.to_regprocedure(
   'vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)'))
   INTO lector;
 IF pg_catalog.regexp_count(lector, 'WHEN SQLSTATE ''40001'' THEN RAISE') <> 2
    OR pg_catalog.regexp_count(lector, 'WHEN SQLSTATE ''40P01'' THEN RAISE') <> 2 THEN
   RAISE EXCEPTION 'CT194: lector oculta errores reintentables';
 END IF;
END $prueba$;
ROLLBACK;
