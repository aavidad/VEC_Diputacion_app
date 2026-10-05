\set ON_ERROR_STOP on
-- Personal38 (4c-4, corte 1). Decisión de dirección del 05/10: el enlace de
-- ejercicio de un cargo guarda como recurso el TIPO de recurso del paso del
-- plan (por ejemplo documento_contratacion_temporal), no un documento
-- concreto. La competencia del firmante se comprueba por tipo; la decisión y
-- el consumo de la firma siguen ligados al documento exacto (OriginalRef y su
-- huella) en CT172/AD170, que este corte no toca.
--  1. localizar_enlace_cargo_ct_v1: localizador privado del único enlace
--     vigente de una persona para un cargo, acción, tipo y finalidad. Lo usa
--     la selección central de AUT antes del PDP; no concede nada.
--  2. leer_revalidar_cargo_ocupante_ct_v1 (Personal28): si el contexto trae
--     recurso.tipo_recurso, coteja con él el recurso del enlace; si no, con
--     recurso_autorizable_ref como hasta ahora (Personal29). Sustitución en
--     sitio con preimagen exacta; propietario y ACL no cambian.
-- Requiere Personal28, Personal29 y Personal37. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_personal:migracion:000038',0));
DO $pre$
DECLARE f regprocedure:=pg_catalog.to_regprocedure('vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'Personal38: PARO clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF pg_catalog.current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'Personal38: PARO clave=PG actual=% esperado=180000..189999',pg_catalog.current_setting('server_version_num') USING ERRCODE='55000'; END IF;
 IF f IS NULL
 OR pg_catalog.to_regprocedure('vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_personal.localizar_enlace_cargo_ct_v1(text,text,text,text,text,text,text)') IS NOT NULL
 OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
 OR pg_catalog.to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'Personal38: PARO clave=preimagen actual=incompatible esperado=Personal28_29_sin_38' USING ERRCODE='55000'; END IF;
 -- Personal37: sin ella ningún enlace se puede insertar.
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_constraint WHERE conrelid='vec_personal.enlace_cargo_competencial_historia'::regclass
   AND contype='c' AND pg_catalog.pg_get_constraintdef(oid) ~ '\{[0-9]+,511\}')
 THEN RAISE EXCEPTION 'Personal38: PARO clave=Personal37 actual=ausente esperado=CHECK_corregidas' USING ERRCODE='55000'; END IF;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex')
   IS DISTINCT FROM '627ff1a7542005cf1f39fd359be6800b0345bdef72f08a3bcf2aa2b0942c7e2d'
 THEN RAISE EXCEPTION 'Personal38: PARO clave=leer_revalidar actual=distinta esperado=Personal28' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_personal_propietario;

-- El único enlace vigente (titular o delegado) de la persona para el cargo,
-- la acción, el tipo de recurso y la finalidad del paso, en la organización y
-- la unidad del cargo. Ausente o ambiguo: denegado. Bloquea punteros y
-- versiones hasta COMMIT, como Personal29; la selección central vuelve a
-- revalidar con Personal29 en la misma transacción.
CREATE FUNCTION vec_personal.localizar_enlace_cargo_ct_v1(
 p_cargo_ref text,p_persona_ref text,p_accion text,p_tipo_recurso text,p_finalidad text,
 p_organizacion_ref text,p_unidad_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c record;e record;n integer;ahora timestamptz(6);
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR current_setting('TimeZone')<>'UTC'
  OR current_setting('role')<>'none'
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
  OR (p_cargo_ref ~ '^car_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (p_persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (p_accion ~ '^[a-z][a-z0-9_.:-]{2,255}$') IS NOT TRUE
  OR (p_tipo_recurso ~ '^[a-z][a-z0-9_.:/#-]+$') IS NOT TRUE OR char_length(p_tipo_recurso) NOT BETWEEN 3 AND 512
  OR (p_finalidad ~ '^[a-z][a-z0-9_.:-]+$') IS NOT TRUE OR char_length(p_finalidad) NOT BETWEEN 3 AND 512
  OR (p_organizacion_ref ~ '^[a-z][a-z0-9_:-]{2,127}$') IS NOT TRUE
  OR (p_unidad_ref ~ '^[a-z][a-z0-9_:-]{2,127}$') IS NOT TRUE
 THEN RAISE EXCEPTION 'cargo_localizacion_denegada' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:cargos_competenciales:v1',0));
 ahora:=clock_timestamp();
 SELECT a.huella_sha256 AS puntero_sha256,h.* INTO STRICT c
 FROM vec_personal.cargo_competencial_actual a
 JOIN vec_personal.cargo_competencial_historia h USING(cargo_ref,version)
 WHERE a.cargo_ref=p_cargo_ref FOR SHARE OF a,h;
 IF c.puntero_sha256 IS DISTINCT FROM c.huella_sha256 OR c.estado<>'vigente'
  OR c.organizacion_ref IS DISTINCT FROM p_organizacion_ref OR c.unidad_ref IS DISTINCT FROM p_unidad_ref
  OR ahora<c.vigente_desde OR ahora>=c.vigente_hasta
 THEN RAISE EXCEPTION 'cargo_localizacion_cargo_no_vigente' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO n
 FROM vec_personal.enlace_cargo_competencial_actual a
 JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
 WHERE h.cargo_ref=p_cargo_ref AND h.cargo_version=c.version AND h.persona_ref=p_persona_ref
  AND h.accion_ref=p_accion AND h.recurso_ref=p_tipo_recurso AND h.finalidad_ref=p_finalidad
  AND h.estado='vigente' AND ahora>=h.vigente_desde AND ahora<h.vigente_hasta
  AND a.huella_sha256=h.huella_sha256;
 IF n<>1 THEN RAISE EXCEPTION 'cargo_localizacion_ausente_o_ambigua' USING ERRCODE='42501'; END IF;
 SELECT h.* INTO STRICT e
 FROM vec_personal.enlace_cargo_competencial_actual a
 JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
 WHERE h.cargo_ref=p_cargo_ref AND h.cargo_version=c.version AND h.persona_ref=p_persona_ref
  AND h.accion_ref=p_accion AND h.recurso_ref=p_tipo_recurso AND h.finalidad_ref=p_finalidad
  AND h.estado='vigente' AND ahora>=h.vigente_desde AND ahora<h.vigente_hasta
  AND a.huella_sha256=h.huella_sha256
 FOR SHARE OF a,h;
 RETURN jsonb_build_object('esquema','vec.personal.enlace-cargo-localizado.ct.v1',
  'cargo',jsonb_build_object('referencia',c.cargo_ref,'version',c.version,'huella_sha256',c.huella_sha256),
  'enlace',jsonb_build_object('referencia',e.enlace_ref,'version',e.version,'huella_sha256',e.huella_sha256),
  'clase',e.clase,'persona_ref',e.persona_ref);
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'cargo_localizacion_ausente_o_ambigua' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_personal.localizar_enlace_cargo_ct_v1(text,text,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.localizar_enlace_cargo_ct_v1(text,text,text,text,text,text,text) TO vec_autorizacion_propietario;

-- Revalidación de Personal28: el recurso del enlace se coteja con el tipo del
-- recurso cuando el contexto lo trae (canon de AUT32/AUT35). Una sola
-- sustitución textual medida; deshacerla devuelve la preimagen exacta.
DO $revalidar$
DECLARE f regprocedure:='vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)'::regprocedure;
 original text;nuevo text;
 antes text:=E'recurso:=j#>>''{recurso,recurso_autorizable_ref}'';';
 despues text:=E'recurso:=COALESCE(j#>>''{recurso,tipo_recurso}'',j#>>''{recurso,recurso_autorizable_ref}'');';
BEGIN
 original:=pg_catalog.pg_get_functiondef(f);
 IF (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,antes,'')))/pg_catalog.length(antes)<>1
 THEN RAISE EXCEPTION 'Personal38: PARO clave=revalidar_ancla actual=no_unica esperado=1' USING ERRCODE='55000'; END IF;
 nuevo:=pg_catalog.replace(original,antes,despues);
 IF pg_catalog.replace(nuevo,despues,antes) IS DISTINCT FROM original
 THEN RAISE EXCEPTION 'Personal38: PARO clave=revalidar_reversible actual=no esperado=si' USING ERRCODE='55000'; END IF;
 EXECUTE nuevo;
END $revalidar$;
RESET ROLE;

DO $post$
DECLARE loc regprocedure:='vec_personal.localizar_enlace_cargo_ct_v1(text,text,text,text,text,text,text)'::regprocedure;
 rev regprocedure:='vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=loc AND proowner='vec_personal_propietario'::regrole AND prosecdef)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid IN(loc,rev) AND (x.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole) OR x.is_grantable))
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',loc,'EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario',rev,'EXECUTE')
 OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=rev)<>'vec_personal_propietario'::regrole
 OR NOT (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=rev)
 OR pg_catalog.strpos(pg_catalog.pg_get_functiondef(rev),'COALESCE(j#>>''{recurso,tipo_recurso}''')=0
 THEN RAISE EXCEPTION 'Personal38: PARO clave=postimagen actual=divergente esperado=localizador_y_revalidar' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
