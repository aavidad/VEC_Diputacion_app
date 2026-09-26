\set ON_ERROR_STOP on
-- CT124 sobre la estructura real restaurada con AD3-82/83/88, CT113/115/116
-- y CT124 instaladas. Requiere antes, en la misma base, ct115_ct116_fixture_pg18.sql
-- y ct124_fixture_pg18.sql. Las fachadas AD3 son dobles explícitos: se prueba
-- la transacción CT (versión, actuación, outbox, recibo, idempotencia, RLS,
-- cierre con el GINPIX confirmado y confirmación del centro), no la
-- criptografía V3. Base desechable: los datos se confirman.
\set exp_a 'expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'

-- Requiere cargar antes ct124_utilidades.sql en la misma sesión.

SELECT recibo_ref AS inc_a FROM vec_contratacion_temporal.incorporacion_registro_v2 WHERE expediente_ref=:'exp_a' \gset
SELECT pg_temp.entrada_cese(:'exp_a',7,'cese','ninguna')::text AS cese_sin_inc \gset
SELECT pg_temp.entrada_cese(:'exp_a',7,'cese',:'inc_a')::text AS cese \gset

-- ---------------------------------------------------------------- cese y cierre sin GINPIX confirmado
SET SESSION AUTHORIZATION vec_ct115_runtime;
SELECT pg_temp.exigir(NOT has_table_privilege('vec_contratacion_temporal.confirmacion_ginpix_v1','SELECT')
  AND NOT has_table_privilege('vec_contratacion_temporal.incorporacion_centro_v1','SELECT')
  AND NOT has_function_privilege('vec_contratacion_temporal.ginpix_confirmado_ct124(text,text)','EXECUTE'),'tablas y auxiliares cerrados al ejecutor');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_cese_nombramiento_v1',:'cese'::jsonb)::text AS r_cese \gset
COMMIT;
SELECT pg_temp.exigir((:'r_cese'::jsonb)->>'resultado'='confirmada' AND (:'r_cese'::jsonb)#>>'{recibo,version_resultante}'='8','cese en la versión 8 '||:'r_cese');
SELECT (:'r_cese'::jsonb)#>>'{recibo,recibo_ref}' AS cese_ref \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.entrada_cierre(:'exp_a',8,'GX-2027-0042','2027-02-16','cierre1',:'cese_ref')::text AS cierre_sin_ginpix \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar('preparar_cierre_expediente_v1','vec.contratacion-temporal.preparar-cierre-expediente.v1',:'cierre_sin_ginpix'::jsonb)->>'resultado' AS prep_cierre \gset
COMMIT;
SELECT pg_temp.exigir(:'prep_cierre'='ginpix_no_confirmado','preparación del cierre sin GINPIX confirmado');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_cierre_expediente_v1',:'cierre_sin_ginpix'::jsonb)->>'resultado'='ginpix_no_confirmado','cierre sin GINPIX confirmado');
COMMIT;
RESET SESSION AUTHORIZATION;

-- ---------------------------------------------------------------- confirmación de GINPIX
SELECT pg_temp.entrada_ginpix(:'exp_a',8,'GX-2027-0042','2027-02-10','g1',:'inc_a')::text AS ginpix \gset
SELECT pg_temp.entrada_ginpix(:'exp_a',8,'GX-2027-0042','2026-12-10','antes',:'inc_a')::text AS ginpix_antes \gset
SELECT pg_temp.entrada_ginpix(:'exp_b',7,'GX-2027-0043','2027-02-10','b',:'inc_a')::text AS ginpix_b \gset
SELECT pg_temp.entrada_ginpix(:'exp_a',8,'GX-2027-0042','2027-02-10','accion',:'inc_a')::text AS ginpix_accion \gset
SELECT jsonb_set(:'ginpix_accion'::jsonb,'{_decision,accion}','"contratacion_temporal.expediente.cerrar"')::text AS ginpix_accion \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar('preparar_confirmacion_ginpix_v1','vec.contratacion-temporal.preparar-confirmacion-ginpix.v1',:'ginpix'::jsonb)::text AS prep_g \gset
SELECT pg_temp.preparar('preparar_confirmacion_ginpix_v1','vec.contratacion-temporal.preparar-confirmacion-ginpix.v1',:'ginpix_b'::jsonb)->>'resultado' AS prep_gb \gset
COMMIT;
SELECT pg_temp.exigir((:'prep_g'::jsonb)->>'resultado'='preparada' AND (:'prep_g'::jsonb)#>>'{incorporacion,recibo_ref}'=:'inc_a'
  AND (:'prep_g'::jsonb)#>>'{expediente,version}'='8','preparación de la confirmación de GINPIX');
SELECT pg_temp.exigir(:'prep_gb'='sin_incorporacion','GINPIX sin incorporación');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir((pg_temp.preparar('preparar_confirmacion_ginpix_v1','vec.contratacion-temporal.preparar-confirmacion-ginpix.v1',:'ginpix'::jsonb))->>'error'='42501','preparación en escritura');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix_antes'::jsonb)->>'resultado'='fecha_anterior_incorporacion','GINPIX anterior a la incorporación');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix_b'::jsonb)->>'resultado'='sin_incorporacion','GINPIX sin incorporación en la confirmación');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix_accion'::jsonb)->>'error'='42501','acción ajena');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',jsonb_set(:'ginpix'::jsonb,'{contexto,atributos,ginpix_numero}','"GX-OTRO"'))->>'error'='42501','contexto manipulado');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',jsonb_set(:'ginpix'::jsonb,'{expediente_siguiente,estado_actual}','"completado"'))->>'error'='22023','proyección manipulada');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',jsonb_set(:'ginpix'::jsonb,'{material,ginpix_numero}','"-malo"'))->>'error'='22023','número con formato inválido');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',jsonb_set(:'ginpix'::jsonb,'{material,ginpix_confirmada_en}','"2027-02-30"'))->>'error'='22023','fecha imposible');
ROLLBACK;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix'::jsonb)::text AS r_g \gset
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix'::jsonb)::text AS r_g2 \gset
SELECT pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',jsonb_set(:'ginpix'::jsonb,'{material,ginpix_numero}','"GX-2027-9999"'))->>'resultado' AS r_g3 \gset
COMMIT;
SELECT pg_temp.exigir((:'r_g'::jsonb)->>'resultado'='confirmada' AND (:'r_g'::jsonb)#>>'{recibo,version_resultante}'='9'
  AND (:'r_g'::jsonb)#>>'{recibo,ginpix_numero}'='GX-2027-0042' AND (:'r_g'::jsonb)#>>'{recibo,operacion}'='confirmar_ginpix','confirmación de GINPIX '||:'r_g');
SELECT pg_temp.exigir((:'r_g2'::jsonb)->'recibo'=(:'r_g'::jsonb)->'recibo','repetición con el mismo recibo');
SELECT pg_temp.exigir(:'r_g3'='idempotencia_reutilizada','clave reutilizada con otro número');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar('preparar_confirmacion_ginpix_v1','vec.contratacion-temporal.preparar-confirmacion-ginpix.v1',:'ginpix'::jsonb)::text AS recup_g \gset
COMMIT;
SELECT pg_temp.exigir((:'recup_g'::jsonb)->>'resultado'='confirmada' AND (:'recup_g'::jsonb)->'recibo'=(:'r_g'::jsonb)->'recibo','recuperación por la preparación');
RESET SESSION AUTHORIZATION;
SELECT pg_temp.entrada_ginpix(:'exp_a',9,'GX-2027-0050','2027-02-11','g2',:'inc_a')::text AS ginpix_otra \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_confirmacion_ginpix_v1',:'ginpix_otra'::jsonb)->>'resultado'='ginpix_existente','segunda confirmación de la misma incorporación');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.exigir(pg_temp.preparar('preparar_confirmacion_ginpix_v1','vec.contratacion-temporal.preparar-confirmacion-ginpix.v1',:'ginpix_otra'::jsonb)->>'resultado'='ginpix_existente','preparación de una segunda confirmación');
COMMIT;
RESET SESSION AUTHORIZATION;

-- ---------------------------------------------------------------- cierre con el número confirmado
SELECT pg_temp.entrada_cierre(:'exp_a',9,'GX-2027-0099','2027-02-10','cierre2',:'cese_ref')::text AS cierre_distinto \gset
SELECT pg_temp.entrada_cierre(:'exp_a',9,'GX-2027-0042','2027-02-11','cierre3',:'cese_ref')::text AS cierre_otra_fecha \gset
SELECT pg_temp.entrada_cierre(:'exp_a',9,'GX-2027-0042','2027-02-10','cierre4',:'cese_ref')::text AS cierre_bueno \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_cierre_expediente_v1',:'cierre_distinto'::jsonb)->>'resultado'='ginpix_no_confirmado','cierre con otro número');
SELECT pg_temp.exigir(pg_temp.confirmar('confirmar_cierre_expediente_v1',:'cierre_otra_fecha'::jsonb)->>'resultado'='ginpix_no_confirmado','cierre con otra fecha');
COMMIT;
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT pg_temp.preparar('preparar_cierre_expediente_v1','vec.contratacion-temporal.preparar-cierre-expediente.v1',:'cierre_bueno'::jsonb)->>'resultado' AS prep_bueno \gset
COMMIT;
SELECT pg_temp.exigir(:'prep_bueno'='preparada','preparación del cierre con el número confirmado');
BEGIN ISOLATION LEVEL SERIALIZABLE;
SELECT pg_temp.confirmar('confirmar_cierre_expediente_v1',:'cierre_bueno'::jsonb)::text AS r_cierre \gset
COMMIT;
SELECT pg_temp.exigir((:'r_cierre'::jsonb)->>'resultado'='confirmada' AND (:'r_cierre'::jsonb)#>>'{recibo,estado_resultante}'='completado'
  AND (:'r_cierre'::jsonb)#>>'{recibo,version_resultante}'='10','cierre con el número confirmado '||:'r_cierre');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.consultar_incorporacion_acreditada_v1((:'cierre_bueno'::jsonb)#>>'{material,organizacion_ref}',:'exp_a')::text AS consulta \gset
COMMIT;
SELECT pg_temp.exigir((:'consulta'::jsonb)#>>'{ginpix,ginpix_numero}'='GX-2027-0042' AND (:'consulta'::jsonb)#>>'{ginpix,ginpix_confirmada_en}'='2027-02-10'
  AND (:'consulta'::jsonb)->'centro'='null'::jsonb
  AND (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys((:'consulta'::jsonb)->'ginpix') k)=ARRAY['ginpix_confirmada_en','ginpix_numero','recibo'],'consulta del detalle '||:'consulta');
SELECT pg_temp.exigir(pg_temp.codigo_consulta('organizacion:otra:ct124', :'exp_a')='42501','consulta con otra organización denegada');
RESET SESSION AUTHORIZATION;

-- ---------------------------------------------------------------- confirmación del centro
SELECT r.peticion->'configuracion'->'ratificador' AS actor_rat, r.peticion->'configuracion'->'solicitante' AS actor_sol, r.peticion_ref AS pet
  FROM vec_contratacion_temporal.peticion_centro_revision r WHERE r.version=2 AND r.centro_ref='centro-520' \gset
SELECT agregado_json->>'organizacion_ref' AS org FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_b' AND version=7 \gset
SELECT jsonb_build_object('modo','bandeja','organizacion_ref',:'org','actor',:'actor_rat'::jsonb)::text AS consulta_c \gset
SELECT pg_temp.material_centro(:'actor_rat'::jsonb,:'org',:'pet',:'exp_b','7c9e6679-7425-40de-944b-e07fc1f90ae7','2026-09-01','sustitucion') AS mat \gset
SELECT pg_temp.material_centro(:'actor_rat'::jsonb,:'org',:'pet',:'exp_b','7c9e6679-7425-40de-944b-e07fc1f90ae7','2026-09-02','sustitucion') AS mat_otro \gset
SELECT pg_temp.material_centro(:'actor_rat'::jsonb,:'org',:'pet',:'exp_b','9b2f4d1e-2a3c-4b5d-8e6f-7a8b9c0d1e2f','2026-09-01','vacante') AS mat_modalidad \gset
SELECT pg_temp.material_centro(:'actor_rat'::jsonb,:'org',:'pet',:'exp_b','9b2f4d1e-2a3c-4b5d-8e6f-7a8b9c0d1e20','2999-01-08','sustitucion') AS mat_futuro \gset
SELECT pg_temp.material_centro(:'actor_rat'::jsonb,:'org',:'pet',:'exp_a','9b2f4d1e-2a3c-4b5d-8e6f-7a8b9c0d1e21','2026-09-01','sustitucion') AS mat_ajeno \gset
SELECT pg_temp.material_centro(jsonb_set(:'actor_rat'::jsonb,'{centro_ref}','"centro-999"'),:'org',:'pet',:'exp_b','9b2f4d1e-2a3c-4b5d-8e6f-7a8b9c0d1e22','2026-09-01','sustitucion') AS mat_otro_centro \gset
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN;
SELECT pg_temp.centro('consultar_incorporaciones_centro_v1',:'consulta_c',pg_temp.decision_centro('contratacion_temporal.incorporacion.consultar_centro',
  'incorporaciones:centro:centro-520',:'actor_rat'::jsonb,:'org',:'consulta_c','b1'))::text AS bandeja \gset
COMMIT;
SELECT pg_temp.exigir((:'bandeja'::jsonb)->>'esquema'='vec.contratacion-temporal.incorporaciones-centro.v1'
  AND jsonb_array_length((:'bandeja'::jsonb)->'expedientes')=1 AND (:'bandeja'::jsonb)#>>'{expedientes,0,expediente_ref}'=:'exp_b'
  AND (:'bandeja'::jsonb)#>>'{expedientes,0,fase}'='nombramiento' AND (:'bandeja'::jsonb)#>>'{expedientes,0,modalidad_clave}'='sustitucion'
  AND (:'bandeja'::jsonb)#>'{expedientes,0,confirmacion}'='null'::jsonb,'bandeja del centro '||:'bandeja');
-- Pertenencia dedicada (cancelación desde el centro): el expediente exacto,
-- sin paginar y sin consumir autorización ni dejar acceso.
RESET SESSION AUTHORIZATION;
SELECT count(*) AS accesos_antes FROM vec_contratacion_temporal.incorporacion_centro_acceso_v1 \gset
CREATE FUNCTION pg_temp.pertenece(a text, o text, e text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN RETURN vec_contratacion_temporal.expediente_del_centro_v1(a, o, e)::text;
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE; END $f$;
SET SESSION AUTHORIZATION vec_ct115_runtime;
SELECT pg_temp.pertenece((:'actor_rat'::jsonb)::text,:'org',:'exp_b') AS pert_rat, pg_temp.pertenece((:'actor_sol'::jsonb)::text,:'org',:'exp_b') AS pert_sol,
       pg_temp.pertenece((:'actor_rat'::jsonb)::text,:'org',:'exp_a') AS pert_ajeno,
       pg_temp.pertenece(jsonb_set(:'actor_rat'::jsonb,'{centro_ref}','"centro-999"')::text,:'org',:'exp_b') AS pert_otro_centro,
       pg_temp.pertenece(jsonb_set(:'actor_rat'::jsonb,'{actor_ref}','"per_otro"')::text,:'org',:'exp_b') AS pert_otro_actor,
       pg_temp.pertenece((:'actor_rat'::jsonb)::text,'organizacion:otra:ct124',:'exp_b') AS pert_otra_org,
       pg_temp.pertenece('{"actor_ref":"x"}',:'org',:'exp_b') AS pert_invalido \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.exigir(:'pert_rat'='true' AND :'pert_sol'='true' AND :'pert_ajeno'='false' AND :'pert_otro_centro'='false'
  AND :'pert_otro_actor'='false' AND :'pert_otra_org'='false' AND :'pert_invalido'='22023'
  AND (SELECT count(*) FROM vec_contratacion_temporal.incorporacion_centro_acceso_v1)=:'accesos_antes'::bigint
  AND NOT has_function_privilege('public','vec_contratacion_temporal.expediente_del_centro_v1(text,text,text)','EXECUTE'),
  'pertenencia del expediente exacto al centro, sin consumir autorización');
SET SESSION AUTHORIZATION vec_ct115_runtime;
BEGIN;
SELECT pg_temp.exigir(pg_temp.centro('consultar_incorporaciones_centro_v1',:'consulta_c',pg_temp.decision_centro('contratacion_temporal.incorporacion.consultar_centro',
  'incorporaciones:centro:centro-999',:'actor_rat'::jsonb,:'org',:'consulta_c','b2'))->>'error'='42501','bandeja con otro recurso');
ROLLBACK;
BEGIN;
SELECT pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat_modalidad',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat_modalidad','m1'))::text AS r_m1 \gset
SELECT pg_temp.exigir((:'r_m1'::jsonb)->>'error'='P0682','modalidad distinta de la del expediente '||:'r_m1');
ROLLBACK;
BEGIN;
SELECT pg_temp.exigir(pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat_futuro',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat_futuro','m2'))->>'error'='22023','incorporación futura');
ROLLBACK;
BEGIN;
SELECT pg_temp.exigir(pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat_ajeno',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_a',:'actor_rat'::jsonb,:'org',:'mat_ajeno','m3'))->>'error'='P0682','expediente de otra petición');
ROLLBACK;
BEGIN;
SELECT pg_temp.exigir(pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat_otro_centro',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',jsonb_set(:'actor_rat'::jsonb,'{centro_ref}','"centro-999"'),:'org',:'mat_otro_centro','m4'))->>'error'='P0682','actor de otro centro');
ROLLBACK;
BEGIN;
SELECT pg_temp.exigir(pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat_otro','m5'))->>'error'='42501','decisión de otro material');
ROLLBACK;
BEGIN READ ONLY;
SELECT pg_temp.exigir(pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat','m6'))->>'error'='42501','confirmación en solo lectura');
ROLLBACK;
BEGIN;
SELECT pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat','c1'))::text AS r_c \gset
COMMIT;
BEGIN;
SELECT pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat','c2'))::text AS r_c2 \gset
SELECT pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat_otro',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat_otro','c3'))->>'error' AS r_c3 \gset
COMMIT;
SELECT pg_temp.exigir((:'r_c'::jsonb)->>'estado_local'='registrado' AND (:'r_c'::jsonb)->>'expediente_ref'=:'exp_b'
  AND (:'r_c'::jsonb)->>'documento_tipo'='toma_posesion' AND (:'r_c'::jsonb)->>'numero_visible'='2026/B-124','confirmación del centro '||:'r_c');
SELECT pg_temp.exigir((:'r_c2'::jsonb)->>'estado_local'='replay_confirmado' AND (:'r_c2'::jsonb)->>'recibo_ref'=(:'r_c'::jsonb)->>'recibo_ref','repetición del centro');
SELECT pg_temp.exigir(:'r_c3'='P0681','clave del centro con otro contenido');
SELECT pg_temp.material_centro(:'actor_rat'::jsonb,:'org',:'pet',:'exp_b','1c9e6679-7425-40de-944b-e07fc1f90ae7','2026-09-01','sustitucion') AS mat_segunda \gset
BEGIN;
SELECT pg_temp.exigir(pg_temp.centro('confirmar_incorporacion_centro_v1',:'mat_segunda',pg_temp.decision_centro('contratacion_temporal.incorporacion.confirmar_centro',
  :'exp_b',:'actor_rat'::jsonb,:'org',:'mat_segunda','c4'))->>'error'='P0682','segunda confirmación del mismo expediente');
ROLLBACK;
BEGIN;
SELECT pg_temp.centro('consultar_incorporaciones_centro_v1',:'consulta_c',pg_temp.decision_centro('contratacion_temporal.incorporacion.consultar_centro',
  'incorporaciones:centro:centro-520',:'actor_rat'::jsonb,:'org',:'consulta_c','b3'))::text AS bandeja2 \gset
COMMIT;
SELECT pg_temp.exigir((:'bandeja2'::jsonb)#>>'{expedientes,0,confirmacion,fecha_incorporacion}'='2026-09-01'
  AND (:'bandeja2'::jsonb)#>>'{expedientes,0,confirmacion,documento_tipo}'='toma_posesion','la bandeja muestra la confirmación');
BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(:'org',:'exp_b')::text AS consulta_b \gset
COMMIT;
SELECT pg_temp.exigir((:'consulta_b'::jsonb)#>>'{centro,fecha_incorporacion}'='2026-09-01' AND (:'consulta_b'::jsonb)#>>'{centro,documento_sha256}'=repeat('f',64)
  AND (:'consulta_b'::jsonb)->'ginpix'='null'::jsonb,'RRHH ve la confirmación del centro');
RESET SESSION AUTHORIZATION;

-- ---------------------------------------------------------------- efectos durables e historia
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_a'
  AND tipo_evento IN ('ct.cese.v1','ct.ginpix-confirmada.v1','ct.cierre_expediente.v1'))=3,'tres eventos del expediente A');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE expediente_ref=:'exp_b'
  AND tipo_evento='ct.incorporacion-confirmada-centro.v1')=1,'evento del centro');
SELECT pg_temp.exigir((SELECT version FROM vec_contratacion_temporal.expediente_integral_actual WHERE expediente_ref=:'exp_b')=7,'la confirmación del centro no cambia la versión');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_contratacion_temporal.incorporacion_centro_acceso_v1)=2,'dos consultas auditadas del centro');
SELECT pg_temp.exigir((SELECT origen_version FROM vec_contratacion_temporal.expediente_version_integral WHERE expediente_ref=:'exp_a' AND version=9)='confirmacion_ginpix_ct124','origen de la versión de GINPIX');
-- Aun sin RLS (superusuario), la historia no se reescribe ni se borra.
DO $$ BEGIN
 BEGIN UPDATE vec_contratacion_temporal.confirmacion_ginpix_v1 SET ginpix_numero='X-1';
  RAISE EXCEPTION 'FALLO reescritura de GINPIX';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF; END;
 BEGIN DELETE FROM vec_contratacion_temporal.incorporacion_centro_v1;
  RAISE EXCEPTION 'FALLO borrado de la confirmación del centro';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'FALLO%' THEN RAISE; END IF; END;
END $$;
SELECT 'CT124 OK' AS resultado;
