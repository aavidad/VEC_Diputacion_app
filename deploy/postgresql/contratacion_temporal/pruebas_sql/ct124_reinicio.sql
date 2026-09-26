\set ON_ERROR_STOP on
-- CT124 tras reiniciar PostgreSQL: los mismos materiales devuelven los mismos
-- recibos y no escriben filas nuevas. Requiere cargar antes ct124_utilidades.sql.
\set exp_a 'expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
SELECT recibo_ref AS inc_a FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE expediente_ref=:'exp_a' \gset
SELECT recibo_json::text AS recibo_g FROM vec_contratacion_temporal.confirmacion_ginpix_v1 WHERE expediente_ref=:'exp_a' \gset
SELECT k.recibo_json::text AS recibo_k, cese.recibo_ref AS cese_ref FROM vec_contratacion_temporal.cierre_expediente_v1 k
  JOIN vec_contratacion_temporal.cese_nombramiento_v1 cese USING (expediente_ref) WHERE k.expediente_ref=:'exp_a' \gset
SELECT recibo_ref AS recibo_c FROM vec_contratacion_temporal.incorporacion_centro_v1 WHERE expediente_ref=:'exp_b' \gset
SELECT r.peticion->'configuracion'->'ratificador' AS actor_rat, r.peticion_ref AS pet
  FROM vec_contratacion_temporal.peticion_centro_revision r WHERE r.version=2 AND r.centro_ref='centro-520' \gset
SELECT agregado_json->>'organizacion_ref' AS org FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=7 \gset
SELECT pg_temp.entrada_ginpix(:'exp_a',8,'GX-2027-0042','2027-02-10','g1',:'inc_a')::text AS ginpix \gset
SELECT pg_temp.entrada_cierre(:'exp_a',9,'GX-2027-0042','2027-02-10','cierre4',:'cese_ref')::text AS cierre_bueno \gset
SELECT pg_temp.material_centro(:'actor_rat'::jsonb,:'org',:'pet',:'exp_b','7c9e6679-7425-40de-944b-e07fc1f90ae7','2026-09-01','sustitucion') AS mat \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix'::jsonb)::text AS r_g \gset
SELECT pg_temp.confirmar('confirmar_cierre_expediente_v1',:'cierre_bueno'::jsonb)::text AS r_k \gset
COMMIT;
BEGIN;
SELECT pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat','r1'))::text AS r_c \gset
COMMIT;
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir((:'r_g'::jsonb)->>'resultado'='confirmada' AND (:'r_g'::jsonb)->'recibo'=:'recibo_g'::jsonb,'GINPIX repetido tras el reinicio');
SELECT pg_temp.exigir((:'r_k'::jsonb)->>'resultado'='confirmada' AND (:'r_k'::jsonb)->'recibo'=:'recibo_k'::jsonb,'cierre repetido tras el reinicio');
SELECT pg_temp.exigir((:'r_c'::jsonb)->>'estado_local'='replay_confirmado' AND (:'r_c'::jsonb)->>'recibo_ref'=:'recibo_c','centro repetido tras el reinicio');
SELECT 'CT124 reinicio OK' AS resultado;
