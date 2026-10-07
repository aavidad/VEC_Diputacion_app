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
 ajustes text; ajustes_huella text; instante timestamptz(6); paso integer;
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
    OR NOT has_function_privilege('vec_contratacion_temporal_consultor_rrhh',
       'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,boolean)',
       'EXECUTE')
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
  ELSE
   IF instantanea.fase_desde=primera.fase_desde OR
      instantanea.ajustes_version<>version_ajustes_previa+1 OR
      instantanea.ajustes_huella_sha256<>ajustes_huella THEN
    RAISE EXCEPTION 'CT190: reentrada no capturó ajustes nuevos'; END IF;
  END IF;
  anterior:=nueva;
 END LOOP;
 PERFORM set_config('ct190.ref',anterior.expediente_ref,true);
 PERFORM set_config('ct190.version',anterior.version::text,true);
 PERFORM set_config('ct190.fase',instantanea.fase_clave,true);
 PERFORM set_config('ct190.desde',instantanea.fase_desde::text,true);
END $prueba$;

-- Una página atestada doble de v5 comprueba solo el añadido de v6. La
-- autorización de v5 está probada por su propio contrato y no se suplanta en
-- la aplicación: esta sustitución existe solo hasta el ROLLBACK final.
SELECT set_config('ct190.contenido',
 'VEC-CT-CONTENIDO-CUADRO-RRHH-V1'||chr(10)||
 (SELECT string_agg(octet_length(convert_to(valor,'UTF8'))::text||':'||valor||chr(10),'' ORDER BY orden)
 FROM unnest(ARRAY['2026-09-30T19:00:00Z','1',current_setting('ct190.ref'),'x','x',
   current_setting('ct190.version')]||array_fill('x'::text,ARRAY[11]))
 WITH ORDINALITY AS campo(valor,orden)),true);
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,
 p_contexto_actor_canonico bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload_vec_ad_3 bytea,p_sobre_cose_sign_1 bytea,p_evidencia_verificacion bytea,
 p_raiz_publica_spki bytea)
RETURNS TABLE (
 contenido_canonico bytea,cursor_siguiente text,esquema text,acceso_ref text,
 secuencia numeric,anterior_sha256 text,huella_sha256 text,
 vinculo_identidad_huella_sha256 text,alcance_huella_sha256 text,
 registrada_en timestamptz,auditoria_vec_ref text,auditoria_vec_huella_sha256 text,
 consumo_vec_huella_sha256 text,contenido_huella_sha256 text,
 resultado_huella_sha256 text,cursor_huella_sha256 text,generada_en timestamptz,
 expediente_ref text,version_expediente numeric,total smallint,recibo_sello_sha256 text,
 total_filtrado numeric,en_tramitacion numeric,con_incidencia numeric,
 en_llamamiento numeric,fase_desde_expedientes text[],fase_desde_instantes timestamptz[],
 urgente_expedientes boolean[],
 recuento_estados text[],recuento_fases text[],recuento_numeros numeric[],
 plazo_fases text[],plazo_desde timestamptz[],plazo_urgentes boolean[],plazo_numeros numeric[])
LANGUAGE sql AS $doble$
 SELECT convert_to(current_setting('ct190.contenido'),'UTF8'),NULL::text,'doble'::text,
 'acceso'::text,1::numeric,NULL::text,'h'::text,'v'::text,'a'::text,now(),
 'aud'::text,'ah'::text,'ch'::text,'coh'::text,'rh'::text,'cuh'::text,now(),
 NULL::text,NULL::numeric,1::smallint,'s'::text,
 1::numeric,1::numeric,0::numeric,0::numeric,
 ARRAY[current_setting('ct190.ref')],ARRAY[current_setting('ct190.desde')::timestamptz],
 ARRAY[false],
 ARRAY['en_tramite'],ARRAY['fiscalizacion'],ARRAY[1::numeric],
 ARRAY['fiscalizacion'],ARRAY[current_setting('ct190.desde')::timestamptz],ARRAY[false],ARRAY[1::numeric]
$doble$;
-- Dobla sólo el grupo autorizado por el mismo corte sintético. El helper real
-- conserva sus propios predicados y se ensaya aparte sobre el clon completo.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.contar_plazos_con_instantanea_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,p_cursor text)
RETURNS TABLE(fase_clave text,fase_desde timestamptz,urgente boolean,numero numeric,
 estado text,base_catalogo_id text,base_version bigint,base_huella_sha256 text,
 ajustes_catalogo_id text,ajustes_encontrados boolean,ajustes_version bigint,
 ajustes_huella_sha256 text,ajustes_canonico text,ajustes_vigente_desde timestamptz,
 capturada_en timestamptz)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $grupo$
 SELECT s.fase_clave,s.fase_desde,false,1::numeric,s.estado,s.base_catalogo_id,
  s.base_version,s.base_huella_sha256,s.ajustes_catalogo_id,s.ajustes_encontrados,
  s.ajustes_version,s.ajustes_huella_sha256,s.ajustes_canonico,
  s.ajustes_vigente_desde,s.capturada_en
 FROM vec_contratacion_temporal.fase_regla_instantanea_v1 s
 WHERE s.expediente_ref=current_setting('ct190.ref')
  AND s.version=current_setting('ct190.version')::numeric
$grupo$;
SET LOCAL ROLE vec_contratacion_temporal_consultor_rrhh;
DO $fachada$
DECLARE salida record; fila jsonb;
BEGIN
 SELECT * INTO STRICT salida FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v6(
  NULL::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
  NULL::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
  NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,NULL::numeric,NULL::numeric,
  NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,true);
 fila:=salida.instantaneas_regla[1];
 IF cardinality(salida.instantaneas_regla) IS DISTINCT FROM 1
    OR fila->>'estado' IS DISTINCT FROM 'capturada'
    OR fila->>'expediente_ref' IS DISTINCT FROM current_setting('ct190.ref')
    OR (fila->>'version_expediente')::numeric IS DISTINCT FROM current_setting('ct190.version')::numeric
    OR fila->>'fase' IS DISTINCT FROM current_setting('ct190.fase')
    OR cardinality(salida.recuento_numeros) IS DISTINCT FROM 1
    OR cardinality(salida.plazo_numeros) IS DISTINCT FROM 1
    OR jsonb_array_length(salida.bases_regla) IS DISTINCT FROM 1
    OR jsonb_array_length(salida.ajustes_regla) IS DISTINCT FROM 1
    OR cardinality(salida.plazo_contextos) IS DISTINCT FROM 1
    OR jsonb_array_length(salida.plazo_bases) IS DISTINCT FROM 1
    OR jsonb_array_length(salida.plazo_ajustes) IS DISTINCT FROM 1 THEN
  RAISE EXCEPTION 'CT190: vínculo nominal o diccionario divergente';
 END IF;
END $fachada$;
ROLLBACK;
