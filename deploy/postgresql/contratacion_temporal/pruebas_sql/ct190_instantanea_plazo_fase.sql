\set ON_ERROR_STOP on
-- Prueba transaccional sintética. Requiere CT190 instalada en un clon PG18.
-- No conserva ninguna fila y no ejecuta DOWN.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $prueba$
DECLARE
 anterior vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 nueva vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 instantanea vec_contratacion_temporal.fase_regla_instantanea_v1%ROWTYPE;
 primera vec_contratacion_temporal.fase_regla_instantanea_v1%ROWTYPE;
 secuencia_previa bigint; base_version bigint; version_ajustes_previa bigint;
 fase_nueva text; fase_siguiente text; canonico text; huella text;
 ajustes text; ajustes_huella text; ajustes_futuros text; instante timestamptz(6); paso integer;
BEGIN
 IF (SELECT count(*) FROM vec_contratacion_temporal.fase_regla_instantanea_v1)
    <> (SELECT count(*) FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.fase_entrada_publicacion_rrhh f
       LEFT JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s
        USING (expediente_ref,version)
       WHERE s.version IS NULL OR s.fase_clave<>f.fase_clave OR s.fase_desde<>f.fase_desde)
 THEN RAISE EXCEPTION 'CT190: faltan filas alineadas'; END IF;
 IF has_table_privilege('vec_contratacion_temporal_ejecutor',
       'vec_contratacion_temporal.fase_regla_instantanea_v1','SELECT')
    OR has_table_privilege('vec_contratacion_temporal_consultor_rrhh',
       'vec_contratacion_temporal.regla_base_publicada_v1','SELECT')
 THEN RAISE EXCEPTION 'CT190: ACL incorrecta'; END IF;

 SELECT * INTO anterior FROM vec_contratacion_temporal.expediente_version_integral
  ORDER BY expediente_ref,version DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT190: el clon no tiene expediente sintético'; END IF;
 -- La base de prueba solo ejercita el contrato SQL y se revierte; el artefacto
 -- de despliegue real lo genera el dominio Go, nunca esta construcción.
 SELECT coalesce(max(version),0)+1 INTO base_version
  FROM vec_contratacion_temporal.regla_base_publicada_v1;
 canonico:=jsonb_build_object('id','vec.contratacion_temporal.reglas',
   'version',base_version,'modulo_id','contratacion_temporal',
   'estado','publicado','entradas','[]'::jsonb,
   'aprobacion_ref','aprobacion:sintetica:ct190')::text;
 huella:=encode(sha256(convert_to(canonico,'UTF8')),'hex');
 INSERT INTO vec_contratacion_temporal.regla_base_publicada_v1
  (catalogo_id,version,huella_sha256,canonico,fuente_sha256,aprobacion_ref)
 VALUES ('vec.contratacion_temporal.reglas',base_version,huella,canonico,
   encode(sha256(convert_to('fuente sintética CT190','UTF8')),'hex'),
   'aprobacion:sintetica:ct190');
 SELECT coalesce(max(secuencia),0) INTO secuencia_previa
  FROM vec_contratacion_temporal.regla_base_activacion_v1;
 INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
  (secuencia,secuencia_esperada,activa,catalogo_id,version,huella_sha256,aprobacion_ref)
 VALUES (secuencia_previa+1,secuencia_previa,true,
  'vec.contratacion_temporal.reglas',base_version,huella,'aprobacion:sintetica:ct190');
 BEGIN
  INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
   (secuencia,secuencia_esperada,activa,catalogo_id,version,huella_sha256,aprobacion_ref)
  VALUES (secuencia_previa+2,secuencia_previa,true,
   'vec.contratacion_temporal.reglas',base_version,huella,'aprobacion:sintetica:ct190');
  RAISE EXCEPTION 'CT190: CAS aceptó cabeza antigua';
 EXCEPTION WHEN SQLSTATE '40001' THEN NULL; END;
 BEGIN
  UPDATE vec_contratacion_temporal.regla_base_publicada_v1 SET canonico='{}'
  WHERE catalogo_id='vec.contratacion_temporal.reglas' AND version=base_version;
  RAISE EXCEPTION 'CT190: historia base mutable';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;

 SELECT coalesce(max(version),0) INTO version_ajustes_previa
  FROM vec_contratacion_temporal.regla_ajuste_version_v1;
 fase_nueva:=CASE WHEN anterior.fase_clave='fiscalizacion'
  THEN 'asignacion_unidad' ELSE 'fiscalizacion' END;
 fase_siguiente:=CASE WHEN fase_nueva='fiscalizacion'
  THEN 'subsanacion_unidad' ELSE 'fiscalizacion' END;

 FOR paso IN 1..3 LOOP
  instante:=date_trunc('microseconds',clock_timestamp());
  nueva:=anterior;
  nueva.version:=anterior.version+1;
  nueva.fase_clave:=CASE WHEN paso=3 THEN fase_siguiente ELSE fase_nueva END;
  nueva.estado:='en_curso';
  nueva.registrada_en:=instante;
  nueva.prueba_canonica:=set_byte(anterior.prueba_canonica,0,
    (get_byte(anterior.prueba_canonica,0)+paso)%256);
  nueva.prueba_huella_sha256:=encode(sha256(nueva.prueba_canonica),'hex');
  nueva.agregado_json:=jsonb_set(jsonb_set(jsonb_set(jsonb_set(
    anterior.agregado_json,'{version}',to_jsonb(nueva.version)),
    '{fase_actual}',to_jsonb(nueva.fase_clave)),
    '{estado_actual}',to_jsonb(nueva.estado)),
    '{actualizado_en}',to_jsonb(to_char(instante AT TIME ZONE 'UTC',
       'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
  nueva.agregado_json_huella_sha256:=encode(
    sha256(convert_to(nueva.agregado_json::text,'UTF8')),'hex');
  INSERT INTO vec_contratacion_temporal.expediente_version_integral
   (expediente_ref,version,agregado_json,agregado_json_huella_sha256,
    prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,
    flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
  VALUES (nueva.expediente_ref,nueva.version,nueva.agregado_json,
    nueva.agregado_json_huella_sha256,nueva.prueba_canonica,nueva.prueba_huella_sha256,
    nueva.flujo_ref,nueva.flujo_version,nueva.flujo_huella_sha256,
    nueva.fase_clave,nueva.estado,nueva.origen_version,nueva.operacion_ref,nueva.registrada_en);
  SELECT * INTO STRICT instantanea FROM vec_contratacion_temporal.fase_regla_instantanea_v1
   WHERE expediente_ref=nueva.expediente_ref AND version=nueva.version;
  IF paso=1 THEN
   IF instantanea.estado<>'capturada' OR instantanea.base_version<>base_version
      OR instantanea.ajustes_version<>version_ajustes_previa THEN
    RAISE EXCEPTION 'CT190: tramo nuevo no capturó cabeza visible'; END IF;
   primera:=instantanea;
   ajustes:='{"c03.plazo_fiscalizacion":{"cantidad":"7"}}';
   ajustes_huella:=encode(sha256(convert_to(ajustes,'UTF8')),'hex');
   INSERT INTO vec_contratacion_temporal.regla_ajuste_version_v1
    (catalogo_id,version,version_esperada,organizacion_ref,ajustes,ajustes_canonico,
     huella_sha256,base_version,base_huella_sha256,actor_ref,motivo_clave,
     vigente_desde,clave_idempotencia,solicitud_huella_sha256,recibo_ref,
     decision_ref,consumo_huella_sha256,auditoria_ref)
   VALUES ('vec.contratacion_temporal.reglas.ajustes',version_ajustes_previa+1,
    version_ajustes_previa,'organizacion:desarrollo:dipgra',ajustes::jsonb,ajustes,
    ajustes_huella,base_version,huella,'principal:sintetico:ct190','prueba_sintetica',
    date_trunc('microseconds',clock_timestamp()),gen_random_uuid(),
    repeat('a',64),'recibo:'||gen_random_uuid()::text,
    'decision:sintetica:ct190',repeat('b',64),'auditoria:sintetica:ct190');
  ELSIF paso=2 THEN
   IF instantanea.fase_desde<>primera.fase_desde OR
      instantanea.base_huella_sha256<>primera.base_huella_sha256 OR
      instantanea.ajustes_version<>primera.ajustes_version OR
      instantanea.capturada_en<>primera.capturada_en THEN
    RAISE EXCEPTION 'CT190: misma fase no heredó captura'; END IF;
   ajustes_futuros:='{"c03.plazo_fiscalizacion":{"cantidad":"6"}}';
   INSERT INTO vec_contratacion_temporal.regla_ajuste_version_v1
    (catalogo_id,version,version_esperada,organizacion_ref,ajustes,ajustes_canonico,
     huella_sha256,base_version,base_huella_sha256,actor_ref,motivo_clave,
     vigente_desde,clave_idempotencia,solicitud_huella_sha256,recibo_ref,
     decision_ref,consumo_huella_sha256,auditoria_ref)
   VALUES ('vec.contratacion_temporal.reglas.ajustes',version_ajustes_previa+2,
    version_ajustes_previa+1,'organizacion:desarrollo:dipgra',
    ajustes_futuros::jsonb,ajustes_futuros,
    encode(sha256(convert_to(ajustes_futuros,'UTF8')),'hex'),base_version,huella,
    'principal:sintetico:ct190','prueba_sintetica',
    date_trunc('microseconds',clock_timestamp()+interval '1 day'),
    gen_random_uuid(),repeat('c',64),'recibo:'||gen_random_uuid()::text,
    'decision:sintetica:ct190',repeat('d',64),'auditoria:sintetica:ct190');
  ELSE
   IF instantanea.fase_desde=primera.fase_desde OR
      instantanea.ajustes_version<>version_ajustes_previa+1 OR
      instantanea.ajustes_huella_sha256<>ajustes_huella THEN
    RAISE EXCEPTION 'CT190: reentrada no eligió el ajuste efectivo anterior al programado'; END IF;
  END IF;
  anterior:=nueva;
 END LOOP;
END $prueba$;

SELECT 'CT190-PRUEBAS-OK' AS resultado;
ROLLBACK;
