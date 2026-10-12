\set ON_ERROR_STOP on
-- Fuente única: data/catalogos/bolsa/politica_cese_ejemplo_v2.json
-- SHA256 del catálogo fuente: d78d6eb60e60edba68b42ca36421c3248e10d9ce73973cbcf09f936dbe43492d
-- Publicación controlada B45: solo dirección, en sesión DBA autorizada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog, pg_temp;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '20s';
SELECT pg_advisory_xact_lock(hashtextextended('bolsa:politica-cese',0));
DO $pre$
DECLARE v vec_bolsa_llamamientos.politica_cese_bolsa;
BEGIN
 SELECT * INTO v FROM vec_bolsa_llamamientos.politica_cese_bolsa ORDER BY version DESC LIMIT 1;
 IF v.version=1 AND v.catalogo_ref='catalogo:bolsa:cese:ejemplo-sintetico:v1'
    AND v.catalogo_sha256=encode(sha256(convert_to(jsonb_build_object('mapeo','{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general"}'::jsonb,'meses_general',5,'meses_acumulacion',9,'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado','ejemplo_sintetico')::text,'UTF8')),'hex') AND v.mapeo='{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general"}'::jsonb
    AND v.meses_general=5 AND v.meses_acumulacion=9
    AND v.computo='fecha_cese_meses_calendario_ajuste_fin_mes' AND v.estado='ejemplo_sintetico' THEN
  RETURN;
 END IF;
 IF v.version=2 AND v.catalogo_ref='catalogo:bolsa:cese:ejemplo-sintetico:v2'
    AND v.catalogo_sha256=encode(sha256(convert_to(jsonb_build_object('mapeo','{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general","vacante":"general"}'::jsonb,'meses_general',5,'meses_acumulacion',9,'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado','ejemplo_sintetico')::text,'UTF8')),'hex') AND v.mapeo='{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general","vacante":"general"}'::jsonb
    AND v.meses_general=5 AND v.meses_acumulacion=9
    AND v.computo='fecha_cese_meses_calendario_ajuste_fin_mes' AND v.estado='ejemplo_sintetico' THEN
  RETURN; -- replay de una publicación ya confirmada.
 END IF;
 RAISE EXCEPTION 'politica_cese_version/catalogo_ref esperado %/% o %/%, observado %/%',
   1,'catalogo:bolsa:cese:ejemplo-sintetico:v1',2,'catalogo:bolsa:cese:ejemplo-sintetico:v2',
   coalesce(v.version::text,'ausente'),coalesce(v.catalogo_ref,'ausente') USING ERRCODE='55000';
END $pre$;
SELECT version,reutilizada,catalogo_sha256
FROM vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(
 'catalogo:bolsa:cese:ejemplo-sintetico:v2','{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general","vacante":"general"}'::jsonb,5,9,'ejemplo_sintetico');
DO $post$
DECLARE v1 vec_bolsa_llamamientos.politica_cese_bolsa;
        v2 vec_bolsa_llamamientos.politica_cese_bolsa;
BEGIN
 SELECT * INTO v1 FROM vec_bolsa_llamamientos.politica_cese_bolsa WHERE version=1;
 SELECT * INTO v2 FROM vec_bolsa_llamamientos.politica_cese_bolsa WHERE version=2;
 IF v1.catalogo_ref IS DISTINCT FROM 'catalogo:bolsa:cese:ejemplo-sintetico:v1' OR v1.mapeo IS DISTINCT FROM '{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general"}'::jsonb
    OR v1.catalogo_sha256 IS DISTINCT FROM encode(sha256(convert_to(jsonb_build_object('mapeo','{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general"}'::jsonb,'meses_general',5,'meses_acumulacion',9,'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado','ejemplo_sintetico')::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'politica_cese_v1 esperado %/%, observado %/%',
   'catalogo:bolsa:cese:ejemplo-sintetico:v1',encode(sha256(convert_to(jsonb_build_object('mapeo','{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general"}'::jsonb,'meses_general',5,'meses_acumulacion',9,'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado','ejemplo_sintetico')::text,'UTF8')),'hex'),coalesce(v1.catalogo_ref,'ausente'),coalesce(v1.catalogo_sha256,'ausente') USING ERRCODE='55000';
 END IF;
 IF v2.catalogo_ref IS DISTINCT FROM 'catalogo:bolsa:cese:ejemplo-sintetico:v2' OR v2.mapeo IS DISTINCT FROM '{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general","vacante":"general"}'::jsonb
    OR v2.meses_general IS DISTINCT FROM 5
    OR v2.meses_acumulacion IS DISTINCT FROM 9
    OR v2.computo IS DISTINCT FROM 'fecha_cese_meses_calendario_ajuste_fin_mes' OR v2.estado IS DISTINCT FROM 'ejemplo_sintetico'
    OR v2.catalogo_sha256 IS DISTINCT FROM encode(sha256(convert_to(jsonb_build_object('mapeo','{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general","vacante":"general"}'::jsonb,'meses_general',5,'meses_acumulacion',9,'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado','ejemplo_sintetico')::text,'UTF8')),'hex') THEN
  RAISE EXCEPTION 'politica_cese_v2 esperado %/%, observado %/%',
   'catalogo:bolsa:cese:ejemplo-sintetico:v2',encode(sha256(convert_to(jsonb_build_object('mapeo','{"contratacion_temporal.interinidad":"general","contratacion_temporal.interinidad|acumulacion_tareas":"acumulacion_tareas","interinidad":"general","interinidad|acumulacion_tareas":"acumulacion_tareas","relevo":"general","vacante":"general"}'::jsonb,'meses_general',5,'meses_acumulacion',9,'computo','fecha_cese_meses_calendario_ajuste_fin_mes','estado','ejemplo_sintetico')::text,'UTF8')),'hex'),coalesce(v2.catalogo_ref,'ausente'),coalesce(v2.catalogo_sha256,'ausente') USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
