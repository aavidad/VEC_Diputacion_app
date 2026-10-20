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
    AND pg_get_constraintdef(c.oid) LIKE '%respuesta-portal:%')
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
    OR position('vec_bolsa_llamamientos.respuesta_portal_llamamiento' in p.prosrc)<>0 THEN
  RAISE EXCEPTION 'CT249 prueba: autorización o frontera Bolsa divergente'; END IF;
 IF has_function_privilege('vec_contratacion_temporal_consultor_rrhh',p.oid,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_propietario',p.oid,'EXECUTE') THEN
  RAISE EXCEPTION 'CT249 prueba: ejecución concedida fuera de RRHH'; END IF;
 SELECT * INTO p FROM pg_proc WHERE oid=to_regprocedure(
  'vec_contratacion_temporal.consultar_detalle_rrhh_con_bolsa_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,text)');
 IF NOT FOUND OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
    OR p.proacl::text IS DISTINCT FROM '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_consultor_rrhh=X/vec_contratacion_temporal_propietario}'
    OR position('resolucion_portal_disponible' in p.prosrc)=0
    OR position('resolucion_ct' in p.prosrc)=0
    OR position('registrar_y_consumir_resolucion_manual_ct_v3_atestada' in p.prosrc)<>0 THEN
  RAISE EXCEPTION 'CT249 prueba: proyección nominal divergente'; END IF;
END $prueba_ct249$;
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
   'respuesta-portal:'||repeat('a',64),'recibo:portal:uno','participacion:uno',
   '{"Respuesta":"aceptacion"}',1);
 BEGIN
  INSERT INTO ct249_fuentes VALUES ('correo_rrhh','revision_manual_portal','j',NULL,NULL,false,
   'comunicacion:mezcla',NULL,NULL,NULL,'{"Respuesta":"aceptacion"}',3);
  RAISE EXCEPTION 'CT249 prueba: correo admitió estado portal';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO ct249_fuentes VALUES ('portal','vigente',NULL,NULL,NULL,false,NULL,
   'respuesta-portal:'||repeat('b',64),'recibo:portal:dos','participacion:dos',
   '{"Respuesta":"aceptacion"}',1);
  RAISE EXCEPTION 'CT249 prueba: portal admitió plazo vigente';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO ct249_fuentes VALUES ('portal','revision_manual_portal',NULL,NULL,NULL,false,
   'comunicacion:inventada','respuesta-portal:'||repeat('c',64),'recibo:portal:tres',
   'participacion:tres','{"Respuesta":"aceptacion"}',1);
  RAISE EXCEPTION 'CT249 prueba: portal admitió comunicación CT54';
 EXCEPTION WHEN check_violation THEN NULL; END;
END $fuentes_ct249$;
ROLLBACK;
