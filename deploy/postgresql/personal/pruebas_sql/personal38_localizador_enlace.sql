\set ON_ERROR_STOP on
-- Vector de Personal38. Como superusuario, sobre un clon desechable con
-- Personal28/29/37/38 y un nodo orgánico vigente; todo en ROLLBACK. Siembra un
-- cargo sintético y dos enlaces (titular con el TIPO de recurso del paso y
-- otro de otra persona), y comprueba el localizador y la revalidación por
-- tipo con el tipo real de la firma VEC (firma_vec_documento_contratacion_temporal).
-- Cada caso imprime «OK <caso>»; un fallo aborta (pg_temp.exigir).
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SELECT n.nodo_ref::text AS nodo,n.revision AS rev,n.organismo_ref AS org,n.unidad_ref AS uni
 FROM vec_personal.org_nodo_historia n WHERE NOT n.retirado
  AND n.revision=(SELECT max(x.revision) FROM vec_personal.org_nodo_historia x WHERE x.nodo_ref=n.nodo_ref)
  AND n.vigente_desde<=current_date AND (n.vigente_hasta IS NULL OR n.vigente_hasta>current_date)
 ORDER BY n.nodo_ref LIMIT 1 \gset
SET LOCAL ROLE vec_personal_propietario;
INSERT INTO vec_personal.cargo_competencial_historia(cargo_ref,version,huella_sha256,organizacion_ref,unidad_ref,puesto_ref,puesto_revision,
 organo_ref,organo_revision,denominacion_catalogo_ref,estado,vigente_desde,vigente_hasta,acto_ref,acto_version,acto_huella_sha256,
 fuente_ref,fuente_version,fuente_huella_sha256,publicada_en,decision_ref,auditoria_ref,recibo_ref)
VALUES('car_P38AAAAAAAAAAAAAAAAAAAAAA',1,repeat('1',64),:'org',:'uni',NULL,NULL,:'nodo'::uuid,:rev,'cargo:p38:jefatura','vigente',
 now()-interval '1 day',now()+interval '100 days','acto:p38:nombramiento',1,repeat('2',64),'fuente:p38',1,repeat('3',64),now(),'decision:p38','aud_p38','percar_p38_cargo');
INSERT INTO vec_personal.cargo_competencial_actual VALUES('car_P38AAAAAAAAAAAAAAAAAAAAAA',1,repeat('1',64));
INSERT INTO vec_personal.enlace_cargo_competencial_historia(enlace_ref,version,huella_sha256,cargo_ref,cargo_version,persona_ref,clase,
 titular_enlace_ref,titular_enlace_version,titular_enlace_sha256,delegante_persona_ref,accion_ref,recurso_ref,finalidad_ref,estado,
 vigente_desde,vigente_hasta,acto_ref,acto_version,acto_huella_sha256,fuente_ref,fuente_version,fuente_huella_sha256,
 requiere_enlace_laboral,empleado_ref,ocupacion_ref,ocupacion_revision,publicada_en,decision_ref,auditoria_ref,recibo_ref)
VALUES
 ('enc_P38TITULARAAAAAAAAAAAAAAA',1,repeat('4',64),'car_P38AAAAAAAAAAAAAAAAAAAAAA',1,'per_P38FIRMANTEAAAAAAAAAAAAA','titular',NULL,NULL,NULL,NULL,
  'contratacion_temporal.documento.firma_vec.registrar','firma_vec_documento_contratacion_temporal','gestionar_contratacion_temporal','vigente',
  now()-interval '1 hour',now()+interval '50 days','acto:p38:nombramiento',1,repeat('2',64),'fuente:p38',1,repeat('3',64),false,NULL,NULL,NULL,now(),'decision:p38','aud_p38','percar_p38_t'),
 ('enc_P38CADUCADOAAAAAAAAAAAAAA',1,repeat('5',64),'car_P38AAAAAAAAAAAAAAAAAAAAAA',1,'per_P38OTRAAAAAAAAAAAAAAAAAA','titular',NULL,NULL,NULL,NULL,
  'contratacion_temporal.documento.firma_vec.registrar','firma_vec_documento_contratacion_temporal','gestionar_contratacion_temporal','vigente',
  now()-interval '10 days',now()-interval '2 days','acto:p38:antiguo',1,repeat('2',64),'fuente:p38',1,repeat('3',64),false,NULL,NULL,NULL,now(),'decision:p38','aud_p38','percar_p38_c');
INSERT INTO vec_personal.enlace_cargo_competencial_actual VALUES('enc_P38TITULARAAAAAAAAAAAAAAA',1,repeat('4',64)),('enc_P38CADUCADOAAAAAAAAAAAAAA',1,repeat('5',64));
RESET ROLE;
CREATE ROLE prueba_p38_ct LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO prueba_p38_ct;
GRANT USAGE ON SCHEMA vec_personal TO prueba_p38_ct;
-- Sólo para el vector: el LOGIN CT llama directamente (en producción la llama
-- la selección central de AUT, propietaria del EXECUTE).
GRANT EXECUTE ON FUNCTION vec_personal.localizar_enlace_cargo_ct_v1(text,text,text,text,text,text,text) TO prueba_p38_ct;
GRANT EXECUTE ON FUNCTION vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea) TO prueba_p38_ct;
GRANT EXECUTE ON FUNCTION vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text) TO prueba_p38_ct;
CREATE FUNCTION pg_temp.exigir(r text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF r IS NULL OR r NOT LIKE 'OK %' THEN RAISE EXCEPTION '%',coalesce(r,'FALLO sin resultado'); END IF; RETURN r; END $f$;
CREATE FUNCTION pg_temp.localizar(persona text,tipo text,cargo text DEFAULT 'car_P38AAAAAAAAAAAAAAAAAAAAAA') RETURNS text LANGUAGE plpgsql AS $f$
DECLARE r jsonb;
BEGIN
 r:=vec_personal.localizar_enlace_cargo_ct_v1(cargo,persona,'contratacion_temporal.documento.firma_vec.registrar',tipo,
  'gestionar_contratacion_temporal',current_setting('p38.org'),current_setting('p38.uni'));
 RETURN r#>>'{enlace,referencia}';
EXCEPTION WHEN insufficient_privilege THEN RAISE NOTICE 'localizar: %',SQLERRM; RETURN 'denegado';
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.localizar(text,text,text) TO prueba_p38_ct;
CREATE FUNCTION pg_temp.revalidar(con_tipo boolean) RETURNS text LANGUAGE plpgsql AS $f$
DECLARE per jsonb:=current_setting('p38.per')::jsonb;rec jsonb;
BEGIN
 rec:=jsonb_build_object('organizacion_ref',per->>'organizacion_ref','unidad_ref',per->>'unidad_ref',
  'recurso_autorizable_ref','documento:p38:original-a');
 IF con_tipo THEN rec:=rec||jsonb_build_object('tipo_recurso','firma_vec_documento_contratacion_temporal'); END IF;
 PERFORM vec_personal.leer_revalidar_cargo_ocupante_ct_v1(convert_to(jsonb_build_object(
  'esquema','vec.competencia-firmante.historica.v1','fecha_historica',clock_timestamp(),
  'personal',jsonb_build_object('cargo',per->'cargo','enlace_ocupante',per->'enlace_ocupante',
   'ocupante_persona_ref',per->>'ocupante_persona_ref','cargo_ref_enlace',per->>'cargo_ref_enlace',
   'cargo_vigente_desde',per->'cargo_vigente_desde','cargo_vigente_hasta',per->'cargo_vigente_hasta',
   'enlace_vigente_desde',per->'enlace_vigente_desde','enlace_vigente_hasta',per->'enlace_vigente_hasta','delegacion',per->'delegacion'),
  'identidad',jsonb_build_object('persona_ref',per->>'persona_ejerciente_ref'),'recurso',rec,
  'accion',per->>'accion_ref','finalidad',per->>'finalidad_ref')::text,'UTF8'));
 RETURN 'aceptado';
EXCEPTION WHEN insufficient_privilege THEN RETURN 'denegado';
END $f$;
GRANT EXECUTE ON FUNCTION pg_temp.revalidar(boolean) TO prueba_p38_ct;
DO $g$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO prueba_p38_ct',pg_my_temp_schema()::regnamespace::text); END $g$;
SELECT set_config('p38.org',:'org',true),set_config('p38.uni',:'uni',true);
SET SESSION AUTHORIZATION prueba_p38_ct;
SET LOCAL timezone='UTC';
SELECT CASE WHEN pg_temp.localizar('per_P38FIRMANTEAAAAAAAAAAAAA','firma_vec_documento_contratacion_temporal')='enc_P38TITULARAAAAAAAAAAAAAAA'
 THEN 'OK localiza_por_tipo' ELSE 'FALLO localiza_por_tipo' END AS r \gset
SELECT pg_temp.exigir(:'r');
SELECT CASE WHEN pg_temp.localizar('per_P38FIRMANTEAAAAAAAAAAAAA','documento:p38:original-a')='denegado'
 THEN 'OK no_localiza_por_documento' ELSE 'FALLO no_localiza_por_documento' END AS r \gset
SELECT pg_temp.exigir(:'r');
SELECT CASE WHEN pg_temp.localizar('per_P38OTRAAAAAAAAAAAAAAAAAA','firma_vec_documento_contratacion_temporal')='denegado'
 THEN 'OK enlace_caducado_no_localizado' ELSE 'FALLO enlace_caducado_no_localizado' END AS r \gset
SELECT pg_temp.exigir(:'r');
SELECT CASE WHEN pg_temp.localizar('per_P38NADIEAAAAAAAAAAAAAAAA','firma_vec_documento_contratacion_temporal')='denegado'
 THEN 'OK otra_persona_denegada' ELSE 'FALLO otra_persona_denegada' END AS r \gset
SELECT pg_temp.exigir(:'r');
-- Personal29 con el enlace por tipo: su contexto lleva recurso_autorizable_ref
-- = recurso del enlace (el tipo) y la revalidación lo acepta.
SELECT vec_personal.resolver_fuente_cargo_ocupante_ct_v1('car_P38AAAAAAAAAAAAAAAAAAAAAA','enc_P38TITULARAAAAAAAAAAAAAAA',
 'per_P38FIRMANTEAAAAAAAAAAAAA',:'org',:'uni')::text AS per \gset
SELECT CASE WHEN (:'per'::jsonb)->>'recurso_autorizable_ref'='firma_vec_documento_contratacion_temporal'
 AND (:'per'::jsonb)#>>'{enlace_ejerciente,referencia}'='enc_P38TITULARAAAAAAAAAAAAAAA'
 THEN 'OK personal29_por_tipo' ELSE 'FALLO personal29_por_tipo' END AS r \gset
SELECT pg_temp.exigir(:'r');
-- Canon como el de AUT32/AUT35: el recurso es el documento exacto y trae el
-- tipo del paso. Con el tipo, la revalidación acepta; sin él, deniega.
SELECT set_config('p38.per',:'per',true);
SELECT CASE WHEN pg_temp.revalidar(true)='aceptado' THEN 'OK revalida_documento_por_tipo' ELSE 'FALLO revalida_documento_por_tipo' END AS r \gset
SELECT pg_temp.exigir(:'r');
SELECT CASE WHEN pg_temp.revalidar(false)='denegado' THEN 'OK sin_tipo_documento_denegado' ELSE 'FALLO sin_tipo_documento_denegado' END AS r \gset
SELECT pg_temp.exigir(:'r');
RESET SESSION AUTHORIZATION;
ROLLBACK;
