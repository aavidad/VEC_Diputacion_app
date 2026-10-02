\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='20s';
DO $prueba$
DECLARE
  resumen vec_contratacion_temporal.resumen_publicacion_rrhh_v1;
  solicitud vec_contratacion_temporal.solicitud_operativa_rrhh_v1;
  analisis vec_contratacion_temporal.analisis_operativo_rrhh_v1;
  entrada vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
  hitos vec_contratacion_temporal.hito_expediente_rrhh_v1[];
  legado bytea;
  abierto bytea;
  instante timestamptz := '2026-09-06T02:00:00Z';
BEGIN
  resumen := ROW(
    'expediente:ct:sintetico:001','organizacion:sintetica:001',
    '2026/CT-000001',2::numeric,'flujo:ct:sintetico',1::numeric,
    repeat('a',64),'analisis','en_curso','centro:sintetico:001',
    'categoria:sintetica:c2','sustitucion','',
    '2026-09-06T01:00:00Z'::timestamptz,
    '2026-09-06T01:30:00Z'::timestamptz
  );
  solicitud := ROW('C2','sustitucion',
    '2026-10-01T00:00:00Z'::timestamptz,
    '2026-12-31T00:00:00Z'::timestamptz,NULL::text);
  analisis := ROW(
    'sustitucion','categoria:sintetica:c2','necesidad_temporal',
    '2026-10-01T00:00:00Z'::timestamptz,
    '2026-12-31T00:00:00Z'::timestamptz,
    10000::smallint,'validada',false,0::bigint,'','','',NULL::text
  );
  hitos := ARRAY[
    ROW(1::numeric,1::numeric,'alta',
      '2026-09-06T01:00:00Z'::timestamptz,
      '','solicitud','pendiente','en_curso'
    )::vec_contratacion_temporal.hito_expediente_rrhh_v1,
    ROW(2::numeric,2::numeric,'analisis',
      '2026-09-06T01:30:00Z'::timestamptz,
      'solicitud','analisis','en_curso','en_curso'
    )::vec_contratacion_temporal.hito_expediente_rrhh_v1
  ];
  entrada := ROW(resumen,solicitud,true,analisis,2::numeric,
    false,NULL,0::numeric,false,NULL,0::numeric,hitos,
    false,NULL,0::numeric,0::numeric);
  legado := vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(instante,entrada);
  IF position(convert_to('VEC-CT-CONTENIDO-DETALLE-RRHH-V3'||chr(10),'UTF8') IN legado)<>1 THEN
    RAISE EXCEPTION 'CT165 alteró cabecera histórica';
  END IF;
  solicitud.periodo_fin := NULL;
  solicitud.periodo_causa_fin := 'reincorporacion_titular';
  analisis.periodo_fin := NULL;
  analisis.periodo_causa_fin := 'reincorporacion_titular';
  entrada.solicitud := solicitud;
  entrada.analisis := analisis;
  abierto := vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1(instante,entrada);
  IF position(convert_to('VEC-CT-CONTENIDO-DETALLE-RRHH-V4'||chr(10),'UTF8') IN abierto)<>1
     OR position(convert_to('causa:reincorporacion_titular','UTF8') IN abierto)=0 THEN
    RAISE EXCEPTION 'CT165 canon V4 no conserva causa';
  END IF;
END
$prueba$;
ROLLBACK;
