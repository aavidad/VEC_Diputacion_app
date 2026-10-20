\set ON_ERROR_STOP on
-- CT247: canon del legado idéntico al instalado y dos historias auténticas del hijo.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $ct247$
DECLARE
 legado vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
 hijo vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
 hijo_original vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1;
 contenido bytea;
 base_presupuesto jsonb;
 duplicado_presupuesto jsonb;
BEGIN
 legado:=pg_catalog.jsonb_populate_record(NULL::vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1,'{"resumen":{"expediente_ref":"expediente:rrhh:padre:01","organizacion_ref":"organizacion:rrhh:01","numero_visible":"2026/CT-001","version":6,"flujo_ref":"flujo:ct:01","flujo_version":1,"flujo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fase_clave":"fiscalizacion","estado_clave":"en_curso","centro_ref":"centro:rrhh:01","categoria_ref":"categoria:rrhh:01","modalidad_clave":"interinidad","unidad_ref":"unidad:rrhh:01","creado_en":"2026-10-18T07:00:00Z","actualizado_en":"2026-10-18T07:05:00Z"},"solicitud":{"grupo_subgrupo":"C2","motivo_clave":"sustitucion","periodo_inicio":"2026-11-01T00:00:00Z","periodo_fin":"2026-12-01T00:00:00Z","periodo_causa_fin":null},"analisis_presente":true,"analisis":{"modalidad_clave":"interinidad","categoria_ref":"categoria:rrhh:01","causa_clave":"sustitucion","periodo_inicio":"2026-11-01T00:00:00Z","periodo_fin":"2026-12-01T00:00:00Z","porcentaje_jornada":10000,"resultado_rc":"no_requerida","coste_presente":false,"coste_centimos":0,"coste_moneda":"","fuente_coste_ref":"","observaciones":"","periodo_causa_fin":null},"referencia_analisis":2,"cobertura_presente":true,"cobertura":{"via_clave":"bolsa_vigente","decision_gobernada":false,"procedimiento_ref":"procedimiento:rrhh:01","bolsa_ref":"bolsa:rrhh:01","comprobaciones":[{"clave":"bolsa_vigente","resultado":"afirmativa"}]},"referencia_cobertura":3,"asignacion_presente":true,"asignacion":{"unidad_ref":"unidad:rrhh:01","asignada_en":"2026-10-18T07:03:00Z","motivo_clave":""},"referencia_asignacion":4,"hitos":[{"secuencia":1,"version_expediente":1,"accion_clave":"solicitud.registrar","realizada_en":"2026-10-18T07:00:00Z","fase_origen":"","fase_destino":"solicitud","estado_origen":"pendiente","estado_destino":"en_curso"},{"secuencia":2,"version_expediente":2,"accion_clave":"analisis.registrar","realizada_en":"2026-10-18T07:01:00Z","fase_origen":"solicitud","fase_destino":"solicitud","estado_origen":"en_curso","estado_destino":"en_curso"},{"secuencia":3,"version_expediente":3,"accion_clave":"cobertura.registrar","realizada_en":"2026-10-18T07:02:00Z","fase_origen":"solicitud","fase_destino":"asignacion_unidad","estado_origen":"en_curso","estado_destino":"en_curso"},{"secuencia":4,"version_expediente":4,"accion_clave":"asignacion.registrar","realizada_en":"2026-10-18T07:03:00Z","fase_origen":"asignacion_unidad","fase_destino":"asignacion_unidad","estado_origen":"en_curso","estado_destino":"en_curso"},{"secuencia":5,"version_expediente":5,"accion_clave":"informe.registrar","realizada_en":"2026-10-18T07:04:00Z","fase_origen":"asignacion_unidad","fase_destino":"informe_juridico","estado_origen":"en_curso","estado_destino":"en_curso"},{"secuencia":6,"version_expediente":6,"accion_clave":"contratacion_temporal.fiscalizacion.registrar","realizada_en":"2026-10-18T07:05:00Z","fase_origen":"informe_juridico","fase_destino":"fiscalizacion","estado_origen":"en_curso","estado_destino":"en_curso"}],"fiscalizacion_presente":true,"fiscalizacion":{"resultado_clave":"favorable","reparo_clave":"","reparo_texto":"","registrada_en":"2026-10-18T07:05:00Z","subsanacion_registrada_en":null,"subsanacion_texto":null},"referencia_fiscalizacion":6,"referencia_subsanacion":0}'::jsonb);
 contenido:=vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1('2026-10-18T07:10:00Z'::timestamptz,legado);
 IF encode(sha256(contenido),'hex')<>'e74c17a6e028df13182921b3bdcccbaf1f129b82936d2966b6b9cafaeabfd846'
 THEN RAISE EXCEPTION 'CT247: canon legado cambiado'; END IF;
 hijo:=pg_catalog.jsonb_populate_record(
   NULL::vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1,
   pg_catalog.to_jsonb(legado)||pg_catalog.jsonb_build_object(
     'resumen',pg_catalog.to_jsonb(legado.resumen)||pg_catalog.jsonb_build_object(
       'expediente_ref','expediente:rrhh:hijo:01',
       'numero_visible','2026/CT-001-P1',
       'creado_en','2026-10-18T07:06:00Z',
       'actualizado_en','2026-10-18T07:06:00Z'),
     'hitos',pg_catalog.jsonb_build_array(
       (pg_catalog.to_jsonb(legado)->'hitos'->5)||pg_catalog.jsonb_build_object(
         'accion_clave','contratacion_temporal.nombramiento.derivar',
         'realizada_en','2026-10-18T07:06:00Z',
         'fase_origen','fiscalizacion')),
     'instruccion_compartida',pg_catalog.jsonb_build_object(
       'derivacion_ref','derivacion:ct:01',
       'padre_ref','expediente:rrhh:padre:01',
       'numero_visible_padre','2026/CT-001',
       'oferta_ref','oferta:ct:01',
       'numero_plaza',1,
       'recibo_derivacion_ref','recibo:ct:01',
       'huella_padre_sha256',pg_catalog.repeat('b',64),
       'creado_en_padre','2026-10-18T07:00:00Z',
       'actualizado_en_padre','2026-10-18T07:05:00Z',
       'hitos_padre',pg_catalog.to_jsonb(legado)->'hitos')));
 hijo_original:=hijo;
 contenido:=vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1('2026-10-18T07:10:00Z'::timestamptz,hijo);
 IF left(convert_from(contenido,'UTF8'),33)<>'VEC-CT-CONTENIDO-DETALLE-RRHH-V5'||chr(10)
 THEN RAISE EXCEPTION 'CT247: falta cabecera del hijo'; END IF;
 IF (hijo.hitos[1]).secuencia<>6 OR
    jsonb_array_length(hijo.instruccion_compartida->'hitos_padre')<>6
 THEN RAISE EXCEPTION 'CT247: cronologías mezcladas'; END IF;
 -- El agregado duplicado supera 256 KiB; cada fragmento acreditado conserva
 -- las cotas de profundidad y nodos del legado, y el total se acota a 2 MiB.
 base_presupuesto:=jsonb_build_object('relleno',repeat('x',130000));
 duplicado_presupuesto:=base_presupuesto||jsonb_build_object(
   'instruccion_compartida',jsonb_build_object(
     'expediente_padre',base_presupuesto,
     'preimagen_padre_json',base_presupuesto::text));
 IF octet_length(duplicado_presupuesto::text)<=262144
    OR octet_length(duplicado_presupuesto::text)>2097152
    OR vec_contratacion_temporal.json_rrhh_seguro_v1(duplicado_presupuesto) IS NOT FALSE
    OR vec_contratacion_temporal.json_rrhh_seguro_v1(
       duplicado_presupuesto-'instruccion_compartida') IS NOT TRUE
    OR vec_contratacion_temporal.json_rrhh_seguro_v1(base_presupuesto) IS NOT TRUE
    OR octet_length(jsonb_build_object('relleno',repeat('x',2097152))::text)<=2097152
 THEN RAISE EXCEPTION 'CT247: presupuesto de agregado derivado incorrecto'; END IF;
 BEGIN
  hijo.instruccion_compartida:=jsonb_set(hijo.instruccion_compartida,'{hitos_padre,5,accion_clave}','"accion:falsa"');
  PERFORM vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1('2026-10-18T07:10:00Z'::timestamptz,hijo);
  RAISE EXCEPTION 'CT247: adulteración del padre admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 hijo:=hijo_original;
 BEGIN
  hijo.instruccion_compartida:=jsonb_set(hijo.instruccion_compartida,'{recibo_derivacion_ref}','""');
  PERFORM vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1('2026-10-18T07:10:00Z'::timestamptz,hijo);
  RAISE EXCEPTION 'CT247: recibo vacío admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
 hijo:=hijo_original;
 BEGIN
  hijo:=pg_catalog.jsonb_populate_record(NULL::vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1,
    pg_catalog.jsonb_set(pg_catalog.to_jsonb(hijo),'{hitos,0,secuencia}','5'::jsonb));
  PERFORM vec_contratacion_temporal.canon_contenido_detalle_rrhh_v1('2026-10-18T07:10:00Z'::timestamptz,hijo);
  RAISE EXCEPTION 'CT247: acto padre como propio admitido';
 EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END $ct247$;

-- Estas llamadas ejecutan los planes aun con cero expedientes; detectan
-- columnas/alias ausentes en los siete filtros sin inventar autorizaciones.
INSERT INTO vec_contratacion_temporal.control_publicacion_rrhh
 (control,corte_base,ultimo_corte,creada_en,actualizada_en)
 SELECT true,0,0,date_trunc('microseconds',clock_timestamp()),date_trunc('microseconds',clock_timestamp())
 WHERE NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.control_publicacion_rrhh WHERE control);
DO $planes$
DECLARE
 a vec_contratacion_temporal.alcance_consulta_rrhh_v1 :=
   ROW('organizacion:ct247:sin-datos','unidad_gestion','unidad:ct247:sin-datos');
 q vec_contratacion_temporal.consulta_cuadro_rrhh_v1 := ROW('','','',100::smallint,'');
 e vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
 total numeric;
BEGIN
 e:=jsonb_populate_record(NULL::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
      '{"es_continuacion":false,"corte_global":0,"pagina_presentada":0}'::jsonb);
 PERFORM vec_contratacion_temporal.materializar_cuadro_rrhh_v1(a,q,e);
 SELECT total_filtrado INTO STRICT total
 FROM vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(a,q,'');
 IF total<>0 THEN RAISE EXCEPTION 'CT247: aparecen expedientes de otra organización'; END IF;
 PERFORM * FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(a,q,'');
 IF EXISTS(SELECT 1 FROM vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(
      a,'mensual',date '2026-01-01',date '2026-01-31')
    WHERE altas<>0 OR llamamientos<>0 OR formalizaciones<>0 OR cierres<>0 OR incidencias<>0)
 THEN RAISE EXCEPTION 'CT247: estadísticas de ámbito ajeno'; END IF;
END $planes$;

-- Tras crear el hijo mediante el POST real, repetir esta focal con
-- -v ct247_hijo_ref=<referencia del hijo>. Sin ese dato, la parte anterior
-- comprueba el canon y los planes; no atribuye autoridad a una fila ficticia.
\if :{?ct247_hijo_ref}
SELECT pg_catalog.set_config('vec.ct247_hijo_ref', :'ct247_hijo_ref', true);
DO $ambitos_reales$
DECLARE
 caso record;
 variante record;
 alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1;
 consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1;
 filtro vec_contratacion_temporal.consulta_cuadro_rrhh_v1;
 estado vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1;
 cuadro vec_contratacion_temporal.materializacion_cuadro_rrhh_v1;
 detalle vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
 encontrados numeric;
 total numeric;
BEGIN
 SELECT d.hijo_ref,d.organizacion_ref,d.material::jsonb->>'unidad_ref' AS unidad_ref,
        p.centro_ref,p.numero_visible,p.corte_global
 INTO STRICT caso
 FROM vec_contratacion_temporal.nombramiento_derivado_v1 d
 JOIN vec_contratacion_temporal.expediente_alta a
   ON a.expediente_ref=d.hijo_ref AND a.derivacion_ref=d.derivacion_ref
 JOIN vec_contratacion_temporal.publicacion_version_rrhh p
   ON p.expediente_ref=d.hijo_ref AND p.version=6
 WHERE d.hijo_ref=pg_catalog.current_setting('vec.ct247_hijo_ref')
   AND d.material_sha256=pg_catalog.encode(pg_catalog.sha256(
       pg_catalog.convert_to(d.material,'UTF8')),'hex')
   AND d.material::jsonb->>'hijo_ref'=d.hijo_ref
   AND d.material::jsonb->>'organizacion_ref'=d.organizacion_ref
   AND p.organizacion_ref=d.organizacion_ref
   AND p.unidad_ref=d.material::jsonb->>'unidad_ref'
   AND a.decision_ref=d.decision_ref
   AND a.actor_ref=d.actor_ref AND a.perfil_ref=d.perfil_ref;
 consulta:=ROW(caso.hijo_ref,0);
 filtro:=ROW(caso.numero_visible,'','',100::smallint,'');
 estado:=pg_catalog.jsonb_populate_record(
   NULL::vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,
   pg_catalog.jsonb_build_object('es_continuacion',false,
     'corte_global',caso.corte_global,'pagina_presentada',0));

 FOR variante IN SELECT * FROM (VALUES
   ('organizacion',caso.organizacion_ref,true),
   ('unidad_gestion',caso.unidad_ref,true),
   ('centro',caso.centro_ref,false),
   ('unidad_gestion','unidad:ct247:ajena',false)
 ) AS x(clase_ambito,ambito_ref,visible) LOOP
  alcance:=ROW(caso.organizacion_ref,variante.clase_ambito,variante.ambito_ref);
  cuadro:=vec_contratacion_temporal.materializar_cuadro_rrhh_v1(alcance,filtro,estado);
  SELECT pg_catalog.count(*) INTO STRICT encontrados
  FROM pg_catalog.unnest(cuadro.resumenes) AS resumen
  WHERE resumen.expediente_ref=caso.hijo_ref;
  IF (encontrados=1) IS DISTINCT FROM variante.visible THEN
   RAISE EXCEPTION 'CT247: bandeja divergente para %',variante.clase_ambito;
  END IF;

  SELECT total_filtrado INTO STRICT total
  FROM vec_contratacion_temporal.contar_totales_cuadro_rrhh_v1(alcance,filtro,'');
  IF (total=1) IS DISTINCT FROM variante.visible THEN
   RAISE EXCEPTION 'CT247: contador divergente para %',variante.clase_ambito;
  END IF;
  SELECT COALESCE(pg_catalog.sum(numero) FILTER (
    WHERE clase='estado_fase'),0) INTO STRICT total
  FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(alcance,filtro,'');
  IF (total=1) IS DISTINCT FROM variante.visible THEN
   RAISE EXCEPTION 'CT247: resumen divergente para %',variante.clase_ambito;
  END IF;

  IF variante.visible THEN
   detalle:=vec_contratacion_temporal.materializar_detalle_rrhh_v1(
     alcance,consulta,caso.corte_global);
   IF ((detalle.detalle).resumen).expediente_ref IS DISTINCT FROM caso.hijo_ref
      OR (detalle.detalle).instruccion_compartida IS NULL THEN
    RAISE EXCEPTION 'CT247: detalle propio incompleto para %',variante.clase_ambito;
   END IF;
  ELSE
   BEGIN
    PERFORM vec_contratacion_temporal.materializar_detalle_rrhh_v1(
      alcance,consulta,caso.corte_global);
    RAISE EXCEPTION 'CT247: detalle ajeno visible para %',variante.clase_ambito;
   EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
  END IF;
 END LOOP;
END $ambitos_reales$;
\else
\echo CT247: canon y planes comprobados; ámbitos positivos pendientes de un hijo creado por el POST real.
\endif
ROLLBACK;
