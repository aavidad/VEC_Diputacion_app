\set ON_ERROR_STOP on
-- Ejecutar después de CT249 en la copia local desechable de la principal.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba_ct249$
DECLARE p pg_proc%ROWTYPE; t pg_class%ROWTYPE;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999 THEN
  RAISE EXCEPTION 'CT249 prueba: PostgreSQL 18 requerido'; END IF;
 SELECT * INTO t FROM pg_class WHERE oid=to_regclass('vec_contratacion_temporal.resolucion_manual_respuesta_rrhh');
 IF NOT FOUND OR t.relowner<>'vec_contratacion_temporal_propietario'::regrole
    OR t.relrowsecurity IS NOT TRUE OR t.relforcerowsecurity IS NOT TRUE
    OR NOT EXISTS (SELECT 1 FROM pg_attribute a
      WHERE a.attrelid=t.oid AND a.attname='comunicacion_ref' AND NOT a.attnotnull)
    OR NOT EXISTS (SELECT 1 FROM pg_attribute a
      WHERE a.attrelid=t.oid AND a.attname='fuente_respuesta' AND a.attnotnull) THEN
  RAISE EXCEPTION 'CT249 prueba: propiedad, RLS o forma de tabla divergente'; END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh r
   WHERE r.fuente_respuesta='correo_rrhh' AND
    (r.comunicacion_ref IS NULL OR r.respuesta_portal_ref IS NOT NULL
     OR r.recibo_respuesta_portal_ref IS NOT NULL OR r.participacion_ref IS NOT NULL)) THEN
  RAISE EXCEPTION 'CT249 prueba: historia de correo alterada'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid=t.oid
    AND c.conname='resolucion_manual_fuente_disjunta_check'
    AND pg_get_constraintdef(c.oid) LIKE '%recibo:respuesta-portal:%')
    OR NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid=t.oid
    AND c.conname='resolucion_manual_llamamiento_unico'
    AND pg_get_constraintdef(c.oid)='UNIQUE (organizacion_ref, expediente_ref, llamamiento_ref)')
    OR NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid=t.oid
    AND c.conname='resolucion_manual_plazo_coherente_check'
    AND pg_get_constraintdef(c.oid) LIKE '%revision_manual_portal%') THEN
  RAISE EXCEPTION 'CT249 prueba: guardas de fuente o plazo ausentes'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF NOT FOUND OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT TRUE
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
    OR position('registrar_y_consumir_resolucion_manual_ct_v3_atestada' in p.prosrc)=0
    OR position('verificar_aceptacion_nativa_ct_para_ct_v1' in p.prosrc)=0
    OR position('origen_seleccion_o6_ct249_v1' in p.prosrc)=0
    OR position('vec_bolsa_llamamientos.respuesta_portal_llamamiento' in p.prosrc)<>0 THEN
  RAISE EXCEPTION 'CT249 prueba: autorización o frontera Bolsa divergente'; END IF;
 IF has_function_privilege('vec_contratacion_temporal_consultor_rrhh',p.oid,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_propietario',p.oid,'EXECUTE') THEN
  RAISE EXCEPTION 'CT249 prueba: ejecución concedida fuera de RRHH'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
    OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
       <> 'a27e9e849cae6debafb3f1ea060a33d9f670aecca031bbb6d1da6bfebc69d799'
    OR position('origen_ct_confirmado' in p.prosrc)=0
    OR position('resolucion_portal_disponible' in p.prosrc)=0
    OR position('origen_seleccion_o6_ct249_v1' in p.prosrc)=0
    OR position('r.resolucion_ref IS NULL' in p.prosrc)=0
    OR position('resolucion_ct' in p.prosrc)<>0
    OR position('registrar_y_consumir_resolucion_manual_ct_v3_atestada' in p.prosrc)<>0 THEN
  RAISE EXCEPTION 'CT249 prueba: proyección nominal divergente'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(jsonb,jsonb,text)');
 IF NOT FOUND OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR p.prosecdef IS NOT FALSE OR p.provolatile<>'i'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}'
    OR position('confirmacion_canonica_seleccion_llamamiento_o6_v1' in p.prosrc)=0
    OR position('retencion_seleccion' in p.prosrc)=0
    OR position('clock_timestamp' in p.prosrc)<>0 THEN
  RAISE EXCEPTION 'CT249 prueba: ayuda O6 financiera divergente'; END IF;
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 e
   WHERE e.tipo_seleccion='llamamiento' AND e.situacion='confirmada'
     AND vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(
      e.solicitud_json,e.recibo_json,e.artefacto_canonico) IS NOT TRUE) THEN
  RAISE EXCEPTION 'CT249 prueba: selección O6 confirmada sin prueba financiera original'; END IF;
END $prueba_ct249$;
RESET ROLE;
DO $fuentes_ct249$
DECLARE restricciones text;
BEGIN
 SELECT string_agg(pg_get_constraintdef(c.oid),',' ORDER BY c.conname) INTO restricciones
 FROM pg_constraint c
 WHERE c.conrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
   AND c.conname IN ('resolucion_manual_plazo_coherente_check',
    'resolucion_manual_fuente_disjunta_check',
    'resolucion_manual_respuesta_rrhh_version_resultante_check');
 IF restricciones IS NULL THEN RAISE EXCEPTION 'CT249 prueba: restricciones ausentes'; END IF;
 EXECUTE 'CREATE TEMP TABLE ct249_fuentes (fuente_respuesta text,estado_plazo text,'||
  'justificante_ref text,contacto_ref text,causa_justificada_ref text,'||
  'respuesta_fuera_de_plazo boolean,comunicacion_ref text,respuesta_portal_ref text,'||
  'recibo_respuesta_portal_ref text,participacion_ref text,solicitud_json jsonb,'||
  'version_resultante numeric,'||restricciones||') ON COMMIT DROP';
 INSERT INTO ct249_fuentes VALUES
  ('correo_rrhh','vigente','justificante:correo',NULL,NULL,false,'comunicacion:correo',
   NULL,NULL,NULL,'{"Respuesta":"aceptacion"}',3),
  ('correo_rrhh','expirado',NULL,'evento:contacto',NULL,false,'comunicacion:expiracion',
   NULL,NULL,NULL,'{"Respuesta":"expiracion_gobernada"}',3),
  ('portal','revision_manual_portal',NULL,NULL,NULL,false,NULL,
   'respuesta-portal:'||repeat('a',64),'recibo:respuesta-portal:'||repeat('a',64),'participacion:uno',
   '{"Respuesta":"aceptacion"}',1);
 BEGIN
  INSERT INTO ct249_fuentes VALUES ('correo_rrhh','revision_manual_portal','j',NULL,NULL,false,
   'comunicacion:mezcla',NULL,NULL,NULL,'{"Respuesta":"aceptacion"}',3);
  RAISE EXCEPTION 'CT249 prueba: correo admitió estado portal';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO ct249_fuentes VALUES ('portal','vigente',NULL,NULL,NULL,false,NULL,
   'respuesta-portal:'||repeat('b',64),'recibo:respuesta-portal:'||repeat('b',64),'participacion:dos',
   '{"Respuesta":"aceptacion"}',1);
  RAISE EXCEPTION 'CT249 prueba: portal admitió plazo vigente';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO ct249_fuentes VALUES ('portal','revision_manual_portal',NULL,NULL,NULL,false,
   'comunicacion:inventada','respuesta-portal:'||repeat('c',64),'recibo:respuesta-portal:'||repeat('c',64),
   'participacion:tres','{"Respuesta":"aceptacion"}',1);
  RAISE EXCEPTION 'CT249 prueba: portal admitió comunicación CT54';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO ct249_fuentes VALUES ('portal','revision_manual_portal',NULL,NULL,NULL,false,
   NULL,'respuesta-portal:'||repeat('d',64),'recibo:portal:incorrecto',
   'participacion:cuatro','{"Respuesta":"aceptacion"}',1);
  RAISE EXCEPTION 'CT249 prueba: recibo ajeno al formato B30';
 EXCEPTION WHEN check_violation THEN NULL; END;
END $fuentes_ct249$;
DO $unico_ct249$
DECLARE definicion text;
BEGIN
 SELECT pg_get_constraintdef(c.oid) INTO definicion FROM pg_constraint c
 WHERE c.conrelid='vec_contratacion_temporal.resolucion_manual_respuesta_rrhh'::regclass
  AND c.conname='resolucion_manual_llamamiento_unico';
 IF definicion IS NULL THEN RAISE EXCEPTION 'CT249 prueba: unicidad ausente'; END IF;
 EXECUTE 'CREATE TEMP TABLE ct249_unico (organizacion_ref text,expediente_ref text,'||
  'llamamiento_ref text,fuente_respuesta text,'||definicion||') ON COMMIT DROP';
 INSERT INTO ct249_unico VALUES ('org:uno','exp:uno','llamamiento:uno','correo_rrhh');
 BEGIN
  INSERT INTO ct249_unico VALUES ('org:uno','exp:uno','llamamiento:uno','portal');
  RAISE EXCEPTION 'CT249 prueba: correo y portal duplicaron llamamiento';
 EXCEPTION WHEN unique_violation THEN NULL; END;
END $unico_ct249$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
-- JSON sintético en memoria: no se inserta un acto ni se emite un recibo.
DO $ct249_finanzas$
DECLARE ref jsonb:=jsonb_build_object('referencia','ref:prueba','version',1,
 'huella_sha256',repeat('a',64)); neutra jsonb:='{"referencia":"","version":0,"huella_sha256":""}'::jsonb;
 datos jsonb; comando jsonb; recibo jsonb; evidencia jsonb; solicitud jsonb;
 artefacto jsonb; canon text; huellas text[]; valido_neutro jsonb; recibo_neutro jsonb;
 valido_positivo jsonb; recibo_positivo jsonb; falso jsonb;
BEGIN
 datos:=jsonb_build_object('operacion_ref','operacion:prueba','organizacion_ref','organizacion:prueba',
  'expediente_ref','expediente:prueba','version_expediente',6,'correlacion_ref','correlacion:prueba',
  'contrato_version',1,'autoridad_solicitante','autoridad:prueba','autorizacion',ref,
  'accion',ref,'recurso',ref,'finalidad',ref,'solicitada_en','2026-10-20T08:00:00Z',
  'valida_hasta','2026-10-20T08:10:00Z');
 comando:=jsonb_build_object('contexto',jsonb_build_object('datos',datos,
  'clave_verificacion_ref','clave:prueba','sello_hmac','hmac:prueba'),
  'necesidad',ref,'bolsa',ref,'orden',ref,'politica',ref,'total_posiciones_orden',1,
  'maxima_posicion_evaluable',1,'huella_recibo_orden',repeat('b',64));
 recibo:=jsonb_build_object('operacion_ref','operacion:prueba','organizacion_ref','organizacion:prueba',
  'expediente_ref','expediente:prueba','version_expediente',6,'correlacion_ref','correlacion:prueba',
  'necesidad',ref,'bolsa',ref,'orden',ref,'politica',ref,'resultado',ref,
  'propuesta_generada',true,'propuesta',ref,
  'accion_evento',jsonb_build_object('referencia','llamamiento.emitir.v1','version',1,
    'huella_sha256','2472cb427ec518ae7f2fe8671284d890f184918cdd3d6d88d9eeb12c61e3a811'),
  'llamamiento_ref','llamamiento:prueba','seleccion_ref','seleccion:prueba',
  'retencion_seleccion',neutra,'orden_seleccionado',1,'recibo_ref','recibo:prueba',
  'auditoria_ref','auditoria:prueba','evento_ref','evento:prueba','confirmada_en','2026-10-20T08:01:00Z',
  'procedencia',jsonb_build_object('autoridad_ref','autoridad:bolsa:llamamiento:desarrollo',
   'respuesta_ref','respuesta:prueba','contrato_version',1,'fuente',ref,
   'evidencia',jsonb_build_object('evidencia_ref','evidencia:prueba',
    'clave_verificacion_ref','clave:respuesta','sello_hmac','hmac:prueba',
    'emitida_en','2026-10-20T08:02:00Z','valida_hasta','2026-10-20T08:08:00Z',
    'retener_hasta','0001-01-01T00:00:00Z'),'sin_retencion_declarada',true));
 evidencia:=jsonb_build_object('esquema','vec.contratacion-temporal.evidencia-bolsa.v1',
  'tipo_material','recibo_llamamiento','autoridad_ref','autoridad:bolsa:llamamiento:desarrollo',
  'clave_verificacion_ref','clave:respuesta','evidencia_ref','evidencia:prueba',
  'peticion_ref','operacion:prueba','huella_peticion_sha256',repeat('0',64),
  'respuesta_ref','respuesta:prueba','huella_respuesta_sha256',repeat('0',64),
  'sello_hmac','hmac:prueba','emitida_en','2026-10-20T08:02:00Z',
  'valida_hasta','2026-10-20T08:08:00Z','retener_hasta','0001-01-01T00:00:00Z',
  'sin_retencion_declarada',true);
 solicitud:=jsonb_build_object('organizacion_ref','organizacion:prueba',
  'expediente_ref','expediente:prueba','version_expediente',6,
  'necesidad',ref,'bolsa',ref,'politica',ref);
 artefacto:=jsonb_build_object('esquema','vec.contratacion-temporal.artefacto-bolsa',
  'version',1,'tipo','recibo_llamamiento','comando',comando,'recibo',recibo,
  'evidencia',evidencia,'clave_verificacion_ref','clave:respuesta',
  'sello_hmac','hmac:prueba','huella_artefacto_sha256','');
 huellas:=vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(artefacto);
 artefacto:=jsonb_set(jsonb_set(artefacto,'{evidencia,huella_peticion_sha256}',to_jsonb(huellas[1])),
  '{evidencia,huella_respuesta_sha256}',to_jsonb(huellas[2]));
 artefacto:=jsonb_set(artefacto,'{huella_artefacto_sha256}',to_jsonb(encode(sha256(convert_to(
  vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(artefacto,true),'UTF8')),'hex')));
 canon:=vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(artefacto,false);
 IF canon IS NULL OR vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
   artefacto,canon,recibo,vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(recibo)) IS NOT TRUE
   OR vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(solicitud,recibo,canon) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT249 prueba: rama neutra canónica rechazada'; END IF;
 valido_neutro:=artefacto; recibo_neutro:=recibo;
 -- Rama ordinaria: retención positiva original, sin marca de no retención.
 recibo:=jsonb_set(recibo,'{retencion_seleccion}',ref);
 recibo:=jsonb_set(recibo,'{procedencia}',(recibo->'procedencia')-'sin_retencion_declarada');
 recibo:=jsonb_set(recibo,'{procedencia,evidencia,retener_hasta}',
  '"2026-10-21T08:08:00Z"'::jsonb);
 artefacto:=jsonb_set(artefacto,'{recibo}',recibo);
 artefacto:=jsonb_set(artefacto,'{evidencia}',(artefacto->'evidencia')-'sin_retencion_declarada');
 artefacto:=jsonb_set(artefacto,'{evidencia,retener_hasta}',
  '"2026-10-21T08:08:00Z"'::jsonb);
 huellas:=vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(artefacto);
 artefacto:=jsonb_set(jsonb_set(artefacto,'{evidencia,huella_peticion_sha256}',to_jsonb(huellas[1])),
  '{evidencia,huella_respuesta_sha256}',to_jsonb(huellas[2]));
 artefacto:=jsonb_set(artefacto,'{huella_artefacto_sha256}',to_jsonb(encode(sha256(convert_to(
  vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(artefacto,true),'UTF8')),'hex')));
 canon:=vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(artefacto,false);
 IF canon IS NULL OR vec_contratacion_temporal.confirmacion_canonica_seleccion_llamamiento_o6_v1(
   artefacto,canon,recibo,vec_contratacion_temporal.recibo_json_seleccion_llamamiento_o6_v1(recibo)) IS NOT TRUE
   OR vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(solicitud,recibo,canon) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT249 prueba: rama con retención canónica rechazada'; END IF;
 valido_positivo:=artefacto; recibo_positivo:=recibo;
 IF vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(solicitud,
    jsonb_set(recibo_neutro,'{retencion_seleccion}',ref),
    vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(valido_neutro,false)) IS TRUE
 OR vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(solicitud,
    jsonb_set(recibo_positivo,'{retencion_seleccion}',neutra),canon) IS TRUE
 OR vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(
    jsonb_set(solicitud,'{necesidad,referencia}','"necesidad:ajena"'::jsonb),recibo_positivo,canon) IS TRUE
 THEN RAISE EXCEPTION 'CT249 prueba: retención o solicitud divergente admitida'; END IF;
 falso:=jsonb_set(valido_positivo,'{huella_artefacto_sha256}',to_jsonb(repeat('f',64)));
 IF vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(solicitud,recibo_positivo,
    vec_contratacion_temporal.artefacto_json_seleccion_llamamiento_o6_v1(falso,false)) IS TRUE
 OR vec_contratacion_temporal.origen_seleccion_o6_ct249_v1(solicitud,
    jsonb_set(recibo_positivo,'{llamamiento_ref}','"llamamiento:ajeno"'::jsonb),canon) IS TRUE
 THEN RAISE EXCEPTION 'CT249 prueba: huella o vínculo de recibo divergente admitido'; END IF;
END $ct249_finanzas$;
ROLLBACK;
