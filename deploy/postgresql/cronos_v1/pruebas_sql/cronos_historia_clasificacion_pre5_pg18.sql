\set ON_ERROR_STOP on
-- Ejecutar solo tras 000001 + 000004 y antes de 000005, en base desechable.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SELECT set_config('vec.cronos.empleado_ref','emp_aaaaaaaaaaaaaaaaaaaaaa',true);
INSERT INTO vec_cronos_v1.marcaje_original
(marcaje_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,material,material_sha256,movimiento,
 instante_utc,recibo_ref,recibo_json,auditoria_ref,decision_ref,consumo_huella_sha256,registrada_en)
SELECT 'marcaje:cronos:historico','emp_aaaaaaaaaaaaaaaaaaaaaa','historico001',
 'per_aaaaaaaaaaaaaaaaaaaaaa','prf_aaaaaaaaaaaaaaaaaaaaaa',m.material,
 encode(sha256(convert_to(m.material,'UTF8')),'hex'),'entrada','2026-09-24T06:00:00Z',
 'recibo:cronos:historico','{}'::jsonb,'auditoria:cronos:historico',
 'decision:cronos:historico',encode(sha256(convert_to('historico','UTF8')),'hex'),clock_timestamp()
FROM (VALUES ('{"canal":{"politica_version_ref":"politica:canal:historica","canal_ref":"canal:historico","origen_ref":"opaco_historico","calidad_ref":"calidad:historica"}}')) m(material);
INSERT INTO vec_cronos_v1.clasificacion_canal
(politica_version_ref,canal_ref,origen_ref,calidad_ref,tipo_origen,fuente_ref,publicada_en)
VALUES ('politica:canal:historica','canal:historico','opaco_historico','calidad:historica',
 'terminal','fuente:sintetica:posterior',clock_timestamp());
DO $assert$
BEGIN
 IF (SELECT tipo_origen FROM vec_cronos_v1.marcaje_original WHERE marcaje_ref='marcaje:cronos:historico') IS NOT NULL THEN
  RAISE EXCEPTION 'clasificación posterior reetiquetó hecho histórico';
 END IF;
END $assert$;
ROLLBACK;
