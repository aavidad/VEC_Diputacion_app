\set ON_ERROR_STOP on
-- Prueba transaccional sintética. Requiere CT190 instalada en un clon PG18.
-- No conserva ninguna fila y no ejecuta DOWN.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
-- Alta inicial y cambio de fase durante la ventana sin base activa.
-- Las tres tablas de origen se clonan sólo en esta transacción sintética.
DO $sin_base$
DECLARE
 origen vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 nueva vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 reserva vec_contratacion_temporal.identidad_reserva_alta%ROWTYPE;
 alta vec_contratacion_temporal.expediente_alta%ROWTYPE;
 captura vec_contratacion_temporal.fase_regla_instantanea_v1%ROWTYPE;
 ref text:='expediente:ct190:sin-base:'||gen_random_uuid()::text;
 numero text:='2026/CT190-SIN-BASE'; instante timestamptz(6); paso integer;
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_base_activacion_v1) THEN
  RAISE EXCEPTION 'CT190: clave=activaciones esperado=0 observado=%',
   (SELECT count(*) FROM vec_contratacion_temporal.regla_base_activacion_v1);
 END IF;
 SELECT * INTO STRICT origen FROM vec_contratacion_temporal.expediente_version_integral
  ORDER BY expediente_ref,version DESC LIMIT 1;
 SELECT * INTO STRICT alta FROM vec_contratacion_temporal.expediente_alta
  WHERE expediente_ref=origen.expediente_ref;
 SELECT * INTO STRICT reserva FROM vec_contratacion_temporal.identidad_reserva_alta
  WHERE reserva_ref=alta.reserva_ref;
 reserva.ambito_hmac:='hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||
  encode(sha256(convert_to(ref,'UTF8')),'hex');
 reserva.reserva_ref:='reserva:'||gen_random_uuid()::text;
 reserva.expediente_ref:=ref; reserva.numero_visible:=numero;
 reserva.recibo_ref:='recibo:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.identidad_reserva_alta SELECT reserva.*;
 alta.expediente_ref:=ref; alta.reserva_ref:=reserva.reserva_ref;
 alta.numero_visible:=numero; alta.decision_ref:='decision:'||gen_random_uuid()::text;
 alta.efecto_ref:='efecto:'||gen_random_uuid()::text;
 INSERT INTO vec_contratacion_temporal.expediente_alta SELECT alta.*;
 FOR paso IN 1..3 LOOP
  instante:=date_trunc('microseconds',clock_timestamp());
  nueva:=origen; nueva.expediente_ref:=ref; nueva.version:=paso;
  nueva.fase_clave:=CASE WHEN paso=3 THEN 'asignacion_unidad' ELSE 'fiscalizacion' END;
  nueva.estado:='en_curso'; nueva.registrada_en:=instante;
  nueva.prueba_canonica:=origen.prueba_canonica||convert_to(':ct190:sin-base:'||paso::text,'UTF8');
  nueva.prueba_huella_sha256:=encode(sha256(nueva.prueba_canonica),'hex');
  nueva.agregado_json:=replace(replace(origen.agregado_json::text,
   origen.expediente_ref,ref),origen.agregado_json->>'numero_visible',numero)::jsonb;
  nueva.agregado_json:=jsonb_set(jsonb_set(jsonb_set(jsonb_set(
   nueva.agregado_json,'{version}',to_jsonb(nueva.version)),
   '{fase_actual}',to_jsonb(nueva.fase_clave)),'{estado_actual}',to_jsonb(nueva.estado)),
   '{actualizado_en}',to_jsonb(to_char(instante AT TIME ZONE 'UTC',
    'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
  nueva.agregado_json_huella_sha256:=encode(sha256(convert_to(nueva.agregado_json::text,'UTF8')),'hex');
  INSERT INTO vec_contratacion_temporal.expediente_version_integral SELECT nueva.*;
  SELECT * INTO STRICT captura FROM vec_contratacion_temporal.fase_regla_instantanea_v1
   WHERE expediente_ref=ref AND version=paso;
  IF captura.estado<>'legado_sin_instantanea' OR captura.base_version IS NOT NULL THEN
   RAISE EXCEPTION 'CT190: clave=sin_base.estado esperado=legado_sin_instantanea observado=%',captura.estado;
  END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
  ROW('organizacion:desarrollo:dipgra','organizacion','organizacion:desarrollo:dipgra')::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
  ROW('','','',100,'')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'') r
  WHERE r.clase='plazo' AND r.captura->>'estado' IS DISTINCT FROM 'legado_sin_instantanea') THEN
  RAISE EXCEPTION 'CT190: clave=sin_base.resumen esperado=legado_sin_instantanea observado=distinto';
 END IF;
END $sin_base$;

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
 BEGIN
  INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
   (secuencia,secuencia_esperada,activa,aprobacion_ref)
  VALUES (secuencia_previa+1,secuencia_previa,false,'aprobacion:sintetica:ct190');
  RAISE EXCEPTION 'CT190: primera activación inactiva aceptada';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
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
 IF (SELECT t.base_version FROM vec_contratacion_temporal.regla_legado_transicion_v1 t)
    IS DISTINCT FROM base_version
    OR (SELECT t.ajustes_version FROM vec_contratacion_temporal.regla_legado_transicion_v1 t)<>0
 THEN RAISE EXCEPTION 'CT190: baseline legado cambió tras editar ajustes'; END IF;
 canonico:=jsonb_set(canonico::jsonb,'{version}',to_jsonb(base_version+1))::text;
 huella:=encode(sha256(convert_to(canonico,'UTF8')),'hex');
 INSERT INTO vec_contratacion_temporal.regla_base_publicada_v1
  (catalogo_id,version,huella_sha256,canonico,fuente_sha256,aprobacion_ref)
 VALUES ('vec.contratacion_temporal.reglas',base_version+1,huella,canonico,
  encode(sha256(convert_to('fuente sintética CT190 posterior','UTF8')),'hex'),
  'aprobacion:sintetica:ct190');
 INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
  (secuencia,secuencia_esperada,activa,catalogo_id,version,huella_sha256,aprobacion_ref)
 VALUES (secuencia_previa+2,secuencia_previa+1,true,
  'vec.contratacion_temporal.reglas',base_version+1,huella,'aprobacion:sintetica:ct190');
 IF (SELECT t.base_version FROM vec_contratacion_temporal.regla_legado_transicion_v1 t)
    IS DISTINCT FROM base_version
 THEN RAISE EXCEPTION 'CT190: activación posterior movió baseline legado'; END IF;
 -- Desactivar la base tampoco debe bloquear una fase nueva.
 INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
  (secuencia,secuencia_esperada,activa,aprobacion_ref)
 VALUES (secuencia_previa+3,secuencia_previa+2,false,'aprobacion:sintetica:ct190');
 SELECT e.* INTO STRICT anterior FROM vec_contratacion_temporal.expediente_version_integral e
  WHERE e.expediente_ref<>nueva.expediente_ref ORDER BY e.expediente_ref,e.version DESC LIMIT 1;
 instante:=date_trunc('microseconds',clock_timestamp());
 nueva:=anterior; nueva.version:=anterior.version+1;
 nueva.fase_clave:=CASE WHEN anterior.fase_clave='fiscalizacion'
  THEN 'asignacion_unidad' ELSE 'fiscalizacion' END;
 nueva.registrada_en:=instante;
 nueva.prueba_canonica:=anterior.prueba_canonica||convert_to(':ct190:inactiva','UTF8');
 nueva.prueba_huella_sha256:=encode(sha256(nueva.prueba_canonica),'hex');
 nueva.agregado_json:=jsonb_set(jsonb_set(jsonb_set(anterior.agregado_json,
  '{version}',to_jsonb(nueva.version)),'{fase_actual}',to_jsonb(nueva.fase_clave)),
  '{actualizado_en}',to_jsonb(to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
 nueva.agregado_json_huella_sha256:=encode(sha256(convert_to(nueva.agregado_json::text,'UTF8')),'hex');
 INSERT INTO vec_contratacion_temporal.expediente_version_integral SELECT nueva.*;
 SELECT * INTO STRICT instantanea FROM vec_contratacion_temporal.fase_regla_instantanea_v1
  WHERE expediente_ref=nueva.expediente_ref AND version=nueva.version;
 IF instantanea.estado<>'legado_sin_instantanea' OR instantanea.base_version IS NOT NULL THEN
  RAISE EXCEPTION 'CT190: clave=inactiva.estado esperado=legado_sin_instantanea observado=%',instantanea.estado;
 END IF;
END $prueba$;

SELECT 'CT190-PRUEBAS-OK' AS resultado;
DO $lectores$
DECLARE v_contenido bytea; v_contextos jsonb; v_grupos integer;
BEGIN
 SELECT contenido_canonico INTO v_contenido
 FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
 WHERE substring(contenido_canonico FROM 1 FOR 32)=
  convert_to('VEC-CT-CONTENIDO-CUADRO-RRHH-V1'||chr(10),'UTF8')
  AND EXISTS (SELECT 1 FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(contenido_canonico))
 LIMIT 1;
 IF v_contenido IS NULL THEN RAISE EXCEPTION 'CT190: falta canon sintético de página'; END IF;
 v_contextos:=vec_contratacion_temporal.leer_capturas_pagina_rrhh_v1(v_contenido);
 IF jsonb_typeof(v_contextos->'filas')<>'array'
  OR jsonb_typeof(v_contextos->'bases')<>'object'
  OR jsonb_typeof(v_contextos->'ajustes')<>'object' THEN
  RAISE EXCEPTION 'CT190: diccionario de página incompleto'; END IF;
 SELECT count(*) INTO v_grupos
 FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
  ROW('organizacion:desarrollo:dipgra','organizacion','organizacion:desarrollo:dipgra')::
   vec_contratacion_temporal.alcance_consulta_rrhh_v1,
  ROW('','','',100,'')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'') r
 WHERE r.clase='plazo' AND r.captura->>'estado' IN ('capturada','legado_base_transicion');
 IF v_grupos<1 THEN RAISE EXCEPTION 'CT190: grupos sin captura'; END IF;
END $lectores$;
-- Doble privado transaccional de v3: ejercita v4/v5 sin atribuir autorización V3 nominal.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 p_capacidad_canonica bytea,p_decision_canonica bytea,p_motivo_canonico bytea,
 p_contexto_actor_canonico bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload_vec_ad_3 bytea,p_sobre_cose_sign_1 bytea,p_evidencia_verificacion bytea,p_raiz_publica_spki bytea)
RETURNS TABLE(contenido_canonico bytea,cursor_siguiente text,esquema text,acceso_ref text,
 secuencia numeric,anterior_sha256 text,huella_sha256 text,vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text,registrada_en timestamptz,auditoria_vec_ref text,
 auditoria_vec_huella_sha256 text,consumo_vec_huella_sha256 text,contenido_huella_sha256 text,
 resultado_huella_sha256 text,cursor_huella_sha256 text,generada_en timestamptz,
 expediente_ref text,version_expediente numeric,total smallint,recibo_sello_sha256 text,
 total_filtrado numeric,en_tramitacion numeric,con_incidencia numeric,en_llamamiento numeric,
 fase_desde_expedientes text[],fase_desde_instantes timestamptz[])
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security='on' AS $f$
DECLARE v_contenido bytea; v_refs text[]; v_desde timestamptz[]; v_total smallint; v_filtrado numeric; v_tramite numeric;
BEGIN
 SELECT p.contenido_canonico INTO v_contenido FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 p
 WHERE substring(p.contenido_canonico FROM 1 FOR 32)=convert_to('VEC-CT-CONTENIDO-CUADRO-RRHH-V1'||chr(10),'UTF8')
 AND EXISTS (SELECT 1 FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(p.contenido_canonico)) LIMIT 1;
 SELECT coalesce(array_agg(e.expediente_ref ORDER BY e.orden),'{}'),coalesce(array_agg(s.fase_desde ORDER BY e.orden),'{}'),count(*)::smallint
 INTO v_refs,v_desde,v_total FROM vec_contratacion_temporal.expedientes_contenido_cuadro_rrhh_v1(v_contenido) e
 JOIN vec_contratacion_temporal.fase_regla_instantanea_v1 s ON s.expediente_ref=e.expediente_ref AND s.version=e.version;
 SELECT coalesce(sum(r.numero),0) INTO v_filtrado FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(p_alcance,p_consulta,p_consulta.cursor) r WHERE r.clase='estado_fase';
 SELECT coalesce(sum(r.numero),0) INTO v_tramite FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(p_alcance,p_consulta,p_consulta.cursor) r WHERE r.clase='plazo';
 RETURN QUERY SELECT v_contenido,''::text,'doble'::text,'acceso:doble'::text,1::numeric,
 repeat('0',64),repeat('1',64),repeat('2',64),repeat('3',64),clock_timestamp(),
 'auditoria:doble'::text,repeat('4',64),repeat('5',64),repeat('6',64),repeat('7',64),repeat('8',64),
 clock_timestamp(), 'expediente:doble'::text,1::numeric,v_total,repeat('9',64),
 v_filtrado,v_tramite,0::numeric,0::numeric,v_refs,v_desde;
END $f$;
DO $test$
DECLARE v_resultado record;
BEGIN
 SELECT * INTO STRICT v_resultado FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v5(
  ROW('organizacion:desarrollo:dipgra','organizacion','organizacion:desarrollo:dipgra')::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
  ROW('','','',100,'')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
  '\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,1,1,
  '\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
 IF jsonb_array_length(v_resultado.capturas_plazo->'filas')<>v_resultado.total
 OR jsonb_array_length(v_resultado.capturas_grupos->'grupos')<>cardinality(v_resultado.plazo_fases)
 OR v_resultado.capturas_grupos->'bases'='{}'::jsonb THEN
  RAISE EXCEPTION 'CT190: v5 desalineada'; END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.contar_resumen_cuadro_rrhh_v1(
  ROW('organizacion:desarrollo:dipgra','organizacion','organizacion:desarrollo:dipgra')::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
  ROW('','','',100,'')::vec_contratacion_temporal.consulta_cuadro_rrhh_v1,'') r
  WHERE r.captura ? 'base_canonico' OR r.captura ? 'ajustes_canonico') THEN
  RAISE EXCEPTION 'CT190: clave=resumen.canonicos esperado=ausentes observado=presentes';
 END IF;
 IF EXISTS (SELECT 1 FROM jsonb_each_text(v_resultado.capturas_grupos->'bases') b
  WHERE b.key<>encode(sha256(convert_to(b.value,'UTF8')),'hex'))
 OR EXISTS (SELECT 1 FROM jsonb_each_text(v_resultado.capturas_grupos->'ajustes') a
  WHERE a.key<>encode(sha256(convert_to(a.value,'UTF8')),'hex'))
 OR v_resultado.capturas_grupos->'ajustes'->>encode(sha256(convert_to('{}','UTF8')),'hex')
    IS DISTINCT FROM '{}'
 OR v_resultado.capturas_grupos->'ajustes'->>encode(sha256(convert_to(
    '{"c03.plazo_fiscalizacion":{"cantidad":"7"}}','UTF8')),'hex')
    IS DISTINCT FROM '{"c03.plazo_fiscalizacion":{"cantidad":"7"}}' THEN
  RAISE EXCEPTION 'CT190: clave=v5.definiciones esperado=huellas_coherentes_y_ajuste_cero observado=distinto';
 END IF;
END $test$;

ROLLBACK;
