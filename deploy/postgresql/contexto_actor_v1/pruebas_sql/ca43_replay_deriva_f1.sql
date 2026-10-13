\set ON_ERROR_STOP on
-- Sólo clon desechable. Simula vcr v2/v3 y revierte historia/puntero/asientos.
-- La secuencia auditora puede avanzar aunque los asientos se reviertan.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
SELECT s.operacion_ref AS operacion,s.login_nombre AS login,
 s.cuenta_ref AS cuenta,s.persona_ref AS persona,s.perfil_ref AS perfil,
 s.vinculo_contexto_ref AS contexto,s.vinculo_corporativo_ref AS vinculo,
 s.organizacion_ref AS organizacion,cv.vinculo_contexto_version AS contexto_version,
 cv.procedencia_ref AS procedencia,cv.procedencia_version AS procedencia_version,
 (SELECT count(*) FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1
  WHERE vinculo_corporativo_ref=s.vinculo_corporativo_ref)::text AS pruebas_antes,
 'vca_'||substr(encode(sha256(convert_to('CA43-replay-contexto:'||s.vinculo_contexto_ref,'UTF8')),'hex'),1,32) AS nuevo_contexto
FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1 prf
JOIN vec_contexto_actor_v1.fuente_aprobada_vinculo_f1 s ON s.operacion_ref=prf.operacion_ref
JOIN vec_contexto_actor_v1.vinculo_corporativo_actual a
 ON a.vinculo_corporativo_ref=prf.vinculo_corporativo_ref AND a.version=prf.version
 AND a.cuenta_ref=s.cuenta_ref AND a.perfil_ref=s.perfil_ref
JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones cv
 ON cv.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND cv.version=a.version
WHERE cv.version=1 AND cv.estado='activo' AND clock_timestamp()>=cv.vigente_desde
 AND clock_timestamp()<cv.vigente_hasta
ORDER BY prf.registrada_en,prf.operacion_ref LIMIT 1
\gset ca43_
SELECT p.procedencia_ref AS ref,p.procedencia_version AS version,
 p.procedencia_huella_sha256 AS huella,p.procedencia_autoridad AS autoridad
FROM vec_contexto_actor_v1.procedencias p
WHERE p.procedencia_autoridad='autoridad_maestra_acreditada'
 AND p.procedencia_ref<>:'ca43_procedencia'
ORDER BY p.procedencia_ref,p.procedencia_version LIMIT 1
\gset ca43_alterna_
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1',0));
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:proyeccion-corporativa-automatica:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));

-- El mismo LOGIN y fuente deben admitir el replay original antes de la deriva.
SET SESSION AUTHORIZATION :"ca43_login";
SELECT (vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(
 :'ca43_operacion',:'ca43_cuenta',:'ca43_persona',:'ca43_perfil',
 :'ca43_contexto',:'ca43_vinculo',:'ca43_organizacion')->>'estado')='replay' AS original_ok
\gset ca43_
SELECT 1 / (:'ca43_original_ok'::boolean::integer);
RESET SESSION AUTHORIZATION;

-- vcr v2 conserva todo salvo la referencia VCA, que sigue teniendo FK completa.
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones(
 vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
 procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
SELECT :'ca43_nuevo_contexto',1,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,procedencia_version,
 procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta
FROM vec_contexto_actor_v1.vinculo_contexto_versiones
WHERE vinculo_ref=:'ca43_contexto' AND version=:'ca43_contexto_version'::numeric;
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones(
 vinculo_corporativo_ref,version,cuenta_ref,cuenta_version,persona_ref,persona_version,
 perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,organizacion_version,
 organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
 organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,procedencia_version,
 procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
SELECT vinculo_corporativo_ref,2,cuenta_ref,cuenta_version,persona_ref,persona_version,
 perfil_ref,perfil_version,:'ca43_nuevo_contexto',1,organizacion_ref,organizacion_version,
 organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
 organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,procedencia_version,
 procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta
FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
WHERE vinculo_corporativo_ref=:'ca43_vinculo' AND version=1;
UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual
 SET version=2 WHERE cuenta_ref=:'ca43_cuenta' AND perfil_ref=:'ca43_perfil'
 AND superficie='interna_corporativa' AND uso='consulta_rrhh'
 AND vinculo_corporativo_ref=:'ca43_vinculo' AND version=1;
SELECT 1 / ((SELECT (to_jsonb(v2)-ARRAY['version','vinculo_contexto_ref'])=
                    (to_jsonb(v1)-ARRAY['version','vinculo_contexto_ref'])
 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v1
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones v2
  ON v2.vinculo_corporativo_ref=v1.vinculo_corporativo_ref AND v2.version=2
 WHERE v1.vinculo_corporativo_ref=:'ca43_vinculo' AND v1.version=1)::integer);
SET SESSION AUTHORIZATION :"ca43_login";
SELECT (vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(
 :'ca43_operacion',:'ca43_cuenta',:'ca43_persona',:'ca43_perfil',
 :'ca43_contexto',:'ca43_vinculo',:'ca43_organizacion')->>'estado')='denegado' AS contexto_denegado
\gset ca43_
SELECT 1 / (:'ca43_contexto_denegado'::boolean::integer);
RESET SESSION AUTHORIZATION;

-- vcr v3 restaura VCA y cambia sólo la procedencia por otra ya acreditada.
INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones(
 vinculo_corporativo_ref,version,cuenta_ref,cuenta_version,persona_ref,persona_version,
 perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,organizacion_version,
 organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
 organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,procedencia_version,
 procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
SELECT vinculo_corporativo_ref,3,cuenta_ref,cuenta_version,persona_ref,persona_version,
 perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,organizacion_version,
 organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
 organizacion_procedencia_autoridad,superficie,uso,
 :'ca43_alterna_ref',:'ca43_alterna_version'::numeric,:'ca43_alterna_huella',:'ca43_alterna_autoridad',
 estado,vigente_desde,vigente_hasta
FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
WHERE vinculo_corporativo_ref=:'ca43_vinculo' AND version=1;
UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual
 SET version=3 WHERE cuenta_ref=:'ca43_cuenta' AND perfil_ref=:'ca43_perfil'
 AND superficie='interna_corporativa' AND uso='consulta_rrhh'
 AND vinculo_corporativo_ref=:'ca43_vinculo' AND version=2;
SELECT 1 / ((SELECT (to_jsonb(v3)-ARRAY['version','procedencia_ref','procedencia_version',
                              'procedencia_huella_sha256','procedencia_autoridad'])=
                    (to_jsonb(v1)-ARRAY['version','procedencia_ref','procedencia_version',
                              'procedencia_huella_sha256','procedencia_autoridad'])
 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v1
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones v3
  ON v3.vinculo_corporativo_ref=v1.vinculo_corporativo_ref AND v3.version=3
 WHERE v1.vinculo_corporativo_ref=:'ca43_vinculo' AND v1.version=1)::integer);
SET SESSION AUTHORIZATION :"ca43_login";
SELECT (vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(
 :'ca43_operacion',:'ca43_cuenta',:'ca43_persona',:'ca43_perfil',
 :'ca43_contexto',:'ca43_vinculo',:'ca43_organizacion')->>'estado')='denegado' AS procedencia_denegada
\gset ca43_
SELECT 1 / (:'ca43_procedencia_denegada'::boolean::integer);
RESET SESSION AUTHORIZATION;
SELECT 1 / (((SELECT count(*) FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1
 WHERE vinculo_corporativo_ref=:'ca43_vinculo')::text=:'ca43_pruebas_antes')::integer);
ROLLBACK;

-- Sin v2/v3 ni nuevo contexto, puntero y prueba originales restaurados.
SELECT 1 / ((NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
 WHERE vinculo_corporativo_ref=:'ca43_vinculo' AND version IN (2,3))
 AND NOT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones
 WHERE vinculo_ref=:'ca43_nuevo_contexto')
 AND EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_actual
 WHERE cuenta_ref=:'ca43_cuenta' AND perfil_ref=:'ca43_perfil'
  AND vinculo_corporativo_ref=:'ca43_vinculo' AND version=1)
 AND (SELECT count(*) FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1
  WHERE vinculo_corporativo_ref=:'ca43_vinculo')::text=:'ca43_pruebas_antes')::integer);
