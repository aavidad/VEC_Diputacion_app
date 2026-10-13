\set ON_ERROR_STOP on
-- CA43: seleccion corporativa por perfil, conservando las versiones y recibos.
-- Dependencia: AD258 y CA42 instaladas; fuente literal pg_get_functiondef de la base post-1090.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $pre$
DECLARE malo integer;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_contexto_actor_v1.vinculo_corporativo_actual') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(jsonb)') IS NULL
 OR to_regclass('vec_contexto_actor_v1.prueba_publicacion_vinculo_f1') IS NULL
 OR EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND attname='perfil_ref' AND NOT attisdropped)
 THEN RAISE EXCEPTION 'CA43: PARO clave=preimagen actual=incompatible esperado=CA42_post1090' USING ERRCODE='55000'; END IF;
 SELECT count(*) INTO malo FROM (VALUES
  ('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)','05c5fa5a16e72114bafea2fef1b3123056bef657245b7e699650b0a8a2c38bff'),
  ('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)','1034877e46f7770145a29d9d0322cd58f304a3cf5a6fed0eac33ce6f401e1d5f'),
  ('vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(text,text,text,numeric,text)','a868adf6ae1cb4c5d9eea7d74a11af6fffe39674195f73e8fd2effb2c5e9f30d'),
  ('vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(text,text,text,text,text,text,text)','06d525c0edb6c453c5591c4efbebd1ec3c4389dbde2cbea06fc2238a4e2a94cf'),
  ('vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)','6c7b9dc59df713d3b33ecd6cd8a6da117e86515301976d59cd3e0bfc2cd3213f')
 ) e(firma,huella) LEFT JOIN pg_proc p ON p.oid=to_regprocedure(e.firma)
 WHERE p.oid IS NULL OR p.proowner<>'vec_contexto_actor_v1_propietario'::regrole
  OR NOT p.prosecdef OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM e.huella;
 IF malo<>0 OR NOT EXISTS(SELECT 1 FROM pg_proc p
   WHERE p.oid='vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(text,text,text)'::regprocedure
   AND p.proowner='vec_contexto_actor_v1_propietario'::regrole AND p.prosecdef
   AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='fd4a536d15de5358884e0625bbe41b360b02344410f766dcccb02b809cad02f3')
 THEN RAISE EXCEPTION 'CA43: PARO clave=funciones actual=deriva esperado=literal_instalado' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM pg_proc WHERE prosrc LIKE '%vinculo_corporativo_actual%')<>5
 THEN RAISE EXCEPTION 'CA43: PARO clave=consumidores actual=deriva esperado=cinco' USING ERRCODE='55000'; END IF;
END $pre$;
-- A/B/C compartidas, D exclusiva; tablas antes de E para evitar ciclos.
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1',0));
SELECT pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:proyeccion-corporativa-automatica:v1',0));
LOCK TABLE vec_contexto_actor_v1.procedencias,
 vec_contexto_actor_v1.proyeccion_cuenta_versiones,
 vec_contexto_actor_v1.persona_versiones,
 vec_contexto_actor_v1.perfil_versiones,
 vec_contexto_actor_v1.vinculo_contexto_versiones IN SHARE MODE;
LOCK TABLE vec_contexto_actor_v1.organizacion_versiones,
 vec_contexto_actor_v1.organizacion_actual IN SHARE MODE;
LOCK TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones,
 vec_contexto_actor_v1.vinculo_corporativo_actual IN ACCESS EXCLUSIVE MODE;
SELECT pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
DO $pre_bajo_lock$
DECLARE n integer;
BEGIN
 SELECT count(*) INTO n FROM (VALUES
  ('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)','05c5fa5a16e72114bafea2fef1b3123056bef657245b7e699650b0a8a2c38bff'),
  ('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)','1034877e46f7770145a29d9d0322cd58f304a3cf5a6fed0eac33ce6f401e1d5f'),
  ('vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(text,text,text,numeric,text)','a868adf6ae1cb4c5d9eea7d74a11af6fffe39674195f73e8fd2effb2c5e9f30d'),
  ('vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(text,text,text,text,text,text,text)','06d525c0edb6c453c5591c4efbebd1ec3c4389dbde2cbea06fc2238a4e2a94cf'),
  ('vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)','6c7b9dc59df713d3b33ecd6cd8a6da117e86515301976d59cd3e0bfc2cd3213f')
 ) e(firma,huella) LEFT JOIN pg_proc p ON p.oid=to_regprocedure(e.firma)
 WHERE p.oid IS NULL OR p.proowner<>'vec_contexto_actor_v1_propietario'::regrole
  OR NOT p.prosecdef OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM e.huella;
 IF n<>0 THEN RAISE EXCEPTION 'CA43: PARO clave=funciones_bajo_lock actual=deriva esperado=literal_instalado' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND relowner='vec_contexto_actor_v1_propietario'::regrole AND relkind='r'
  AND relrowsecurity AND relforcerowsecurity AND relacl IS NULL)
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass
  AND relowner='vec_contexto_actor_v1_propietario'::regrole AND relkind='r'
  AND relrowsecurity AND relforcerowsecurity AND relacl IS NULL)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND conname='vinculo_corporativo_actual_pk' AND pg_get_constraintdef(oid,false)='PRIMARY KEY (cuenta_ref, superficie, uso)')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND conname='vinculo_corporativo_actual_version_fk' AND
  pg_get_constraintdef(oid,false)='FOREIGN KEY (cuenta_ref, superficie, uso, vinculo_corporativo_ref, version) REFERENCES vec_contexto_actor_v1.vinculo_corporativo_versiones(cuenta_ref, superficie, uso, vinculo_corporativo_ref, version) MATCH FULL')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass
  AND conname='vinculo_corporativo_versiones_actual_uq' AND
  pg_get_constraintdef(oid,false)='UNIQUE (cuenta_ref, superficie, uso, vinculo_corporativo_ref, version)')
 OR (SELECT count(*) FROM pg_trigger WHERE tgrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND NOT tgisinternal AND tgenabled='O'
  AND tgname IN ('serializar_mutacion_punteros_actuales_v2','avanzar_generacion_punteros_actuales_v2','puntero_actual_no_truncable_v2'))<>3
 THEN RAISE EXCEPTION 'CA43: PARO clave=puntero actual=deriva esperado=CA4a_protegido' USING ERRCODE='55000'; END IF;
 SELECT count(*) INTO n FROM pg_attribute WHERE attrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND attnum>0 AND NOT attisdropped;
 IF n<>5 THEN RAISE EXCEPTION 'CA43: PARO clave=columnas actual=% esperado=5',n USING ERRCODE='55000'; END IF;
END $pre_bajo_lock$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_versiones
 ADD CONSTRAINT vinculo_corporativo_versiones_actual_perfil_uq
 UNIQUE (cuenta_ref,perfil_ref,superficie,uso,vinculo_corporativo_ref,version);
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual ADD COLUMN perfil_ref text;
DO $relleno$
DECLARE esperadas bigint;modificadas bigint;
BEGIN
 SELECT count(*) INTO esperadas FROM vec_contexto_actor_v1.vinculo_corporativo_actual;
 IF esperadas>0 THEN
  UPDATE vec_contexto_actor_v1.vinculo_corporativo_actual a SET perfil_ref=v.perfil_ref
  FROM vec_contexto_actor_v1.vinculo_corporativo_versiones v
  WHERE (v.cuenta_ref,v.superficie,v.uso,v.vinculo_corporativo_ref,v.version)=
   (a.cuenta_ref,a.superficie,a.uso,a.vinculo_corporativo_ref,a.version);
  GET DIAGNOSTICS modificadas=ROW_COUNT;
  IF modificadas<>esperadas THEN RAISE EXCEPTION 'CA43: PARO clave=relleno actual=% esperado=%',modificadas,esperadas USING ERRCODE='55000'; END IF;
 END IF;
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_actual WHERE perfil_ref IS NULL)
 THEN RAISE EXCEPTION 'CA43: PARO clave=perfil actual=ausente esperado=historia_exacta' USING ERRCODE='55000'; END IF;
END $relleno$;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual ALTER COLUMN perfil_ref SET NOT NULL;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual DROP CONSTRAINT vinculo_corporativo_actual_version_fk;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual DROP CONSTRAINT vinculo_corporativo_actual_pk;
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
 ADD CONSTRAINT vinculo_corporativo_actual_pk PRIMARY KEY (cuenta_ref,perfil_ref,superficie,uso);
ALTER TABLE vec_contexto_actor_v1.vinculo_corporativo_actual
 ADD CONSTRAINT vinculo_corporativo_actual_version_fk FOREIGN KEY
 (cuenta_ref,perfil_ref,superficie,uso,vinculo_corporativo_ref,version)
 REFERENCES vec_contexto_actor_v1.vinculo_corporativo_versiones
 (cuenta_ref,perfil_ref,superficie,uso,vinculo_corporativo_ref,version) MATCH FULL;
-- Definiciones instaladas reconstruidas literalmente; sólo cambian los selectores
-- de perfil y el search_path seguro en CA9/CA25.
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(p_cuenta_ref text, p_perfil_ref text, p_persona_ref text, p_vinculo_contexto_ref text, p_vinculo_contexto_version numeric)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE
    ahora timestamptz; coincidencias integer;
BEGIN
    PERFORM vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1();
    IF vec_contexto_actor_v1.referencia_valida(p_cuenta_ref, 'cta_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref, 'prf_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_persona_ref, 'per_') IS NOT TRUE
       OR vec_contexto_actor_v1.referencia_valida(p_vinculo_contexto_ref, 'vca_') IS NOT TRUE
       OR p_vinculo_contexto_version IS NULL
       OR p_vinculo_contexto_version NOT BETWEEN 1 AND 18446744073709551615::numeric
       OR p_vinculo_contexto_version <> pg_catalog.trunc(p_vinculo_contexto_version) THEN
        RETURN false;
    END IF;
    ahora := pg_catalog.clock_timestamp();
    SELECT count(*) INTO coincidencias
      FROM vec_contexto_actor_v1.vinculo_contexto_actual va
      JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
        ON (v.vinculo_ref, v.version) = (va.vinculo_ref, va.version)
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref = v.cuenta_ref
      JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c
        ON (c.cuenta_ref, c.version) = (ca.cuenta_ref, ca.version)
      JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref = v.perfil_ref
      JOIN vec_contexto_actor_v1.perfil_versiones p
        ON (p.perfil_ref, p.version) = (pa.perfil_ref, pa.version)
      JOIN vec_contexto_actor_v1.persona_actual xa ON xa.persona_ref = v.persona_ref
      JOIN vec_contexto_actor_v1.persona_versiones x
        ON (x.persona_ref, x.version) = (xa.persona_ref, xa.version)
      JOIN vec_contexto_actor_v1.vinculo_corporativo_actual vc
        ON vc.cuenta_ref = v.cuenta_ref AND vc.perfil_ref = v.perfil_ref
       AND vc.superficie = 'interna_corporativa'
       AND vc.uso = 'consulta_rrhh'
      JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones cv
        ON (cv.vinculo_corporativo_ref, cv.version) = (vc.vinculo_corporativo_ref, vc.version)
      JOIN vec_contexto_actor_v1.organizacion_actual oa ON oa.organizacion_ref = cv.organizacion_ref
      JOIN vec_contexto_actor_v1.organizacion_versiones ov
        ON (ov.organizacion_ref, ov.version) = (oa.organizacion_ref, oa.version)
     WHERE va.vinculo_ref = p_vinculo_contexto_ref AND va.version = p_vinculo_contexto_version
       AND v.cuenta_ref = p_cuenta_ref AND v.perfil_ref = p_perfil_ref
       AND v.persona_ref = p_persona_ref AND p.persona_ref = p_persona_ref
       AND cv.cuenta_ref = v.cuenta_ref AND cv.perfil_ref = v.perfil_ref
       AND cv.persona_ref = v.persona_ref
       AND cv.vinculo_contexto_ref = v.vinculo_ref AND cv.vinculo_contexto_version = v.version
       AND cv.cuenta_version = ca.version AND cv.perfil_version = pa.version
       AND cv.persona_version = xa.version
       AND cv.organizacion_version = oa.version
       AND cv.organizacion_procedencia_ref = ov.procedencia_ref
       AND cv.organizacion_procedencia_version = ov.procedencia_version
       AND cv.organizacion_procedencia_huella_sha256 = ov.procedencia_huella_sha256
       AND cv.organizacion_procedencia_autoridad = ov.procedencia_autoridad
       AND cv.procedencia_autoridad = 'autoridad_maestra_acreditada'
       AND ov.procedencia_autoridad = 'autoridad_maestra_acreditada'
       AND cv.superficie = 'interna_corporativa' AND cv.uso = 'consulta_rrhh'
       AND cv.estado = 'activo' AND ov.estado = 'activo'
       AND v.estado = 'activo' AND c.estado = 'activo'
       AND p.estado = 'activo' AND x.estado = 'activo'
       AND ahora >= v.vigente_desde AND ahora < v.vigente_hasta
       AND ahora >= c.vigente_desde AND ahora < c.vigente_hasta
       AND ahora >= p.vigente_desde AND ahora < p.vigente_hasta
       AND ahora >= x.vigente_desde AND ahora < x.vigente_hasta
       AND ahora >= cv.vigente_desde AND ahora < cv.vigente_hasta
       AND ahora >= ov.vigente_desde AND ahora < ov.vigente_hasta;
    RETURN coincidencias = 1;
END
$function$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(p_contexto_canonico bytea)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
AS $function$
DECLARE j jsonb;r record;ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_contexto_canonico IS NULL OR octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
 THEN RETURN false; END IF;
 j:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
 IF jsonb_typeof(j) IS DISTINCT FROM 'object'
 OR j->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR j->>'estado' IS DISTINCT FROM 'activo'
 OR vec_contexto_actor_v1.referencia_valida(j->>'cuenta_ref','cta_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(j->>'persona_ref','per_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(j->>'perfil_activo_ref','prf_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(j->>'contexto_actor_ref','vca_') IS NOT TRUE
 OR (j->>'cuenta_version') !~ '^[1-9][0-9]{0,19}$'
 OR (j->>'persona_version') !~ '^[1-9][0-9]{0,19}$'
 OR (j->>'perfil_version') !~ '^[1-9][0-9]{0,19}$'
 OR (j->>'contexto_version') !~ '^[1-9][0-9]{0,19}$'
 THEN RETURN false; END IF;
 -- CA9 ya publica la relación corporativa. Aquí se fijan sus punteros actuales
 -- hasta el COMMIT que también registra la lectura en AD237 y proyecta B103.
 SELECT v.*,c.version AS cuenta_version,c.estado AS cuenta_estado,
  c.vigente_desde AS cuenta_desde,c.vigente_hasta AS cuenta_hasta,
  c.procedencia_autoridad AS cuenta_autoridad,
  x.version AS persona_version,x.estado AS persona_estado,
  x.vigente_desde AS persona_desde,x.vigente_hasta AS persona_hasta,
  x.procedencia_autoridad AS persona_autoridad,
  p.version AS perfil_version,p.estado AS perfil_estado,p.persona_ref AS perfil_persona_ref,
  p.vigente_desde AS perfil_desde,p.vigente_hasta AS perfil_hasta,
  p.procedencia_autoridad AS perfil_autoridad,
  cv.estado AS corporativo_estado,cv.vigente_desde AS corporativo_desde,
  cv.vigente_hasta AS corporativo_hasta,cv.procedencia_autoridad AS corporativo_autoridad,
  cv.cuenta_ref AS corporativo_cuenta_ref,cv.perfil_ref AS corporativo_perfil_ref,
  cv.persona_ref AS corporativo_persona_ref,
  cv.vinculo_contexto_ref AS corporativo_vinculo_ref,
  cv.vinculo_contexto_version AS corporativo_vinculo_version,
  cv.cuenta_version AS corporativo_cuenta_version,
  cv.perfil_version AS corporativo_perfil_version,
  cv.persona_version AS corporativo_persona_version,
  cv.organizacion_version AS corporativo_organizacion_version,
  cv.organizacion_procedencia_ref AS corporativo_organizacion_procedencia_ref,
  cv.organizacion_procedencia_version AS corporativo_organizacion_procedencia_version,
  cv.organizacion_procedencia_huella_sha256 AS corporativo_organizacion_procedencia_huella_sha256,
  cv.organizacion_procedencia_autoridad AS corporativo_organizacion_procedencia_autoridad,
  ov.version AS organizacion_version,ov.procedencia_ref AS organizacion_procedencia_ref,
  ov.procedencia_version AS organizacion_procedencia_version,
  ov.procedencia_huella_sha256 AS organizacion_procedencia_huella_sha256,
  ov.estado AS organizacion_estado,ov.vigente_desde AS organizacion_desde,
  ov.vigente_hasta AS organizacion_hasta,ov.procedencia_autoridad AS organizacion_autoridad
 INTO STRICT r
 FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v
  ON v.vinculo_ref=va.vinculo_ref AND v.version=va.version
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref=v.cuenta_ref
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c
  ON c.cuenta_ref=ca.cuenta_ref AND c.version=ca.version
 JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref=v.perfil_ref
 JOIN vec_contexto_actor_v1.perfil_versiones p
  ON p.perfil_ref=pa.perfil_ref AND p.version=pa.version
 JOIN vec_contexto_actor_v1.persona_actual xa ON xa.persona_ref=v.persona_ref
 JOIN vec_contexto_actor_v1.persona_versiones x
  ON x.persona_ref=xa.persona_ref AND x.version=xa.version
 JOIN vec_contexto_actor_v1.vinculo_corporativo_actual vc
  ON vc.cuenta_ref=v.cuenta_ref AND vc.perfil_ref=v.perfil_ref
   AND vc.superficie='interna_corporativa'
   AND vc.uso='consulta_rrhh'
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones cv
  ON cv.vinculo_corporativo_ref=vc.vinculo_corporativo_ref AND cv.version=vc.version
 JOIN vec_contexto_actor_v1.organizacion_actual oa
  ON oa.organizacion_ref=cv.organizacion_ref
 JOIN vec_contexto_actor_v1.organizacion_versiones ov
  ON ov.organizacion_ref=oa.organizacion_ref AND ov.version=oa.version
 WHERE va.vinculo_ref=j->>'contexto_actor_ref'
  AND v.cuenta_ref=j->>'cuenta_ref' AND v.persona_ref=j->>'persona_ref'
  AND v.perfil_ref=j->>'perfil_activo_ref'
  AND cv.superficie='interna_corporativa' AND cv.uso='consulta_rrhh'
 FOR SHARE OF va,ca,pa,xa,vc,oa;
 ahora:=clock_timestamp();
 RETURN r.version IS NOT DISTINCT FROM (j->>'contexto_version')::numeric
  AND r.cuenta_version IS NOT DISTINCT FROM (j->>'cuenta_version')::numeric
  AND r.persona_version IS NOT DISTINCT FROM (j->>'persona_version')::numeric
  AND r.perfil_version IS NOT DISTINCT FROM (j->>'perfil_version')::numeric
  AND r.estado='activo' AND r.cuenta_estado='activo'
  AND r.persona_estado='activo' AND r.perfil_estado='activo'
  AND r.corporativo_estado='activo' AND r.organizacion_estado='activo'
  AND r.procedencia_autoridad='autoridad_maestra_acreditada'
  AND r.cuenta_autoridad='autoridad_maestra_acreditada'
  AND r.persona_autoridad='autoridad_maestra_acreditada'
  AND r.perfil_autoridad='autoridad_maestra_acreditada'
  AND r.corporativo_autoridad='autoridad_maestra_acreditada'
  AND r.organizacion_autoridad='autoridad_maestra_acreditada'
  AND r.persona_ref IS NOT DISTINCT FROM r.perfil_persona_ref
  AND r.cuenta_ref IS NOT DISTINCT FROM r.corporativo_cuenta_ref
  AND r.perfil_ref IS NOT DISTINCT FROM r.corporativo_perfil_ref
  AND r.persona_ref IS NOT DISTINCT FROM r.corporativo_persona_ref
  AND r.vinculo_ref IS NOT DISTINCT FROM r.corporativo_vinculo_ref
  AND r.version IS NOT DISTINCT FROM r.corporativo_vinculo_version
  AND r.cuenta_version IS NOT DISTINCT FROM r.corporativo_cuenta_version
  AND r.perfil_version IS NOT DISTINCT FROM r.corporativo_perfil_version
  AND r.persona_version IS NOT DISTINCT FROM r.corporativo_persona_version
  AND r.organizacion_version IS NOT DISTINCT FROM r.corporativo_organizacion_version
  AND r.organizacion_procedencia_ref IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_ref
  AND r.organizacion_procedencia_version IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_version
  AND r.organizacion_procedencia_huella_sha256 IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_huella_sha256
  AND r.organizacion_autoridad IS NOT DISTINCT FROM r.corporativo_organizacion_procedencia_autoridad
  AND ahora>=GREATEST(r.vigente_desde,r.cuenta_desde,r.persona_desde,r.perfil_desde,
   r.corporativo_desde,r.organizacion_desde)
  AND ahora<LEAST(r.vigente_hasta,r.cuenta_hasta,r.persona_hasta,r.perfil_hasta,
   r.corporativo_hasta,r.organizacion_hasta,(j->>'vigente_hasta')::timestamptz);
EXCEPTION WHEN no_data_found OR too_many_rows OR data_exception OR invalid_text_representation THEN
 RETURN false;
END $function$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(c text, p text, v text, v_version numeric, org_esperada text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE s jsonb;pr record;cr record;o record;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR vec_contexto_actor_v1.organizacion_ref_valida(org_esperada) IS NOT TRUE
  OR v_version IS NULL OR v_version<1 OR pg_catalog.scale(v_version)<>0 THEN RETURN NULL; END IF;
 s:=vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(c,p,v);
 IF s IS NULL OR (s#>>'{vinculo_cuenta_persona,version}')::numeric<>v_version THEN RETURN NULL; END IF;
 SELECT x.* INTO pr FROM vec_contexto_actor_v1.perfil_actual a
 JOIN vec_contexto_actor_v1.perfil_versiones x USING(perfil_ref,version)
 WHERE a.perfil_ref=s#>>'{vinculo_cuenta_persona,perfil_ref}' FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT x.* INTO cr FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones x
  ON x.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND x.version=a.version
 WHERE a.cuenta_ref=c AND a.perfil_ref=s#>>'{vinculo_cuenta_persona,perfil_ref}'
  AND a.superficie='interna_corporativa' AND a.uso='consulta_rrhh' FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT x.* INTO o FROM vec_contexto_actor_v1.organizacion_actual a
 JOIN vec_contexto_actor_v1.organizacion_versiones x USING(organizacion_ref,version)
 WHERE a.organizacion_ref=org_esperada FOR SHARE OF a,x;
 IF NOT FOUND THEN RETURN NULL; END IF;
 ahora:=pg_catalog.clock_timestamp();
 IF pr.persona_ref<>p OR pr.estado<>'activo' OR pr.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR cr.cuenta_ref<>c OR cr.persona_ref<>p OR cr.perfil_ref<>pr.perfil_ref
  OR cr.vinculo_contexto_ref<>v OR cr.vinculo_contexto_version<>v_version
  OR cr.cuenta_version<>(s#>>'{cuenta,version}')::numeric
  OR cr.persona_version<>(s#>>'{persona,version}')::numeric OR cr.perfil_version<>pr.version
  OR cr.organizacion_ref<>org_esperada OR cr.organizacion_version<>o.version
  OR cr.organizacion_procedencia_ref<>o.procedencia_ref
  OR cr.organizacion_procedencia_version<>o.procedencia_version
  OR cr.organizacion_procedencia_huella_sha256<>o.procedencia_huella_sha256
  OR cr.organizacion_procedencia_autoridad<>o.procedencia_autoridad
  OR cr.estado<>'activo' OR o.estado<>'activo'
  OR cr.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR o.procedencia_autoridad<>'autoridad_maestra_acreditada'
  OR ahora<GREATEST(pr.vigente_desde,cr.vigente_desde,o.vigente_desde)
  OR ahora>=LEAST(pr.vigente_hasta,cr.vigente_hasta,o.vigente_hasta)
 THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object('organizacion_ref',o.organizacion_ref,'organizacion_version',o.version,
  'organizacion_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.to_jsonb(o)::text,'UTF8')),'hex'),
  'organizacion_procedencia_ref',o.procedencia_ref,'organizacion_procedencia_version',o.procedencia_version,
  'organizacion_procedencia_sha256',o.procedencia_huella_sha256,
  'vinculo_corporativo_ref',cr.vinculo_corporativo_ref,'vinculo_corporativo_version',cr.version,
  'vinculo_corporativo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.to_jsonb(cr)::text,'UTF8')),'hex'));
END $function$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(p_operacion_ref text, p_cuenta_ref text, p_persona_ref text, p_perfil_ref text, p_vinculo_contexto_ref text, p_vinculo_corporativo_ref text, p_organizacion_ref text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE s vec_contexto_actor_v1.fuente_aprobada_vinculo_f1; l record; g record;
 c record; x record; p record; v record; o record; vc record; pr record;
 ahora timestamptz(6); plan jsonb; sha text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('role')<>'none' OR current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'CA42: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 SELECT * INTO l FROM pg_roles WHERE rolname=session_user;
 SELECT * INTO g FROM pg_roles WHERE rolname='vec_contexto_actor_corporativo_rrhh_publicador';
 IF l.rolname IS NULL OR NOT l.rolcanlogin OR NOT l.rolinherit OR l.rolsuper OR l.rolcreatedb OR l.rolcreaterole
 OR l.rolreplication OR l.rolbypassrls OR l.rolvaliduntil IS NOT NULL
 OR g.rolname IS NULL OR g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
 OR g.rolreplication OR g.rolbypassrls OR g.rolvaliduntil IS NOT NULL
 OR NOT pg_has_role(session_user,g.rolname,'member')
 OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
  AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
  AND m.grantor=(SELECT datdba FROM pg_database WHERE datname=current_database()))
 OR has_database_privilege(l.oid,current_database(),'CREATE,TEMPORARY')
 OR has_schema_privilege(l.oid,'vec_contexto_actor_v1','CREATE')
 OR EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole IN (l.oid,g.oid))
 OR pg_has_role(session_user,'vec_contexto_actor_v1_runtime','member')
 OR has_schema_privilege(l.oid,'vec_autorizacion_atestada_v3','USAGE')
 OR EXISTS(SELECT 1 FROM pg_class t WHERE t.relnamespace='vec_contexto_actor_v1'::regnamespace
  AND t.relkind IN ('r','p') AND has_table_privilege(l.oid,t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'))
 OR EXISTS(SELECT 1 FROM pg_proc f WHERE f.pronamespace='vec_contexto_actor_v1'::regnamespace
  AND f.oid NOT IN (
   'vec_contexto_actor_v1.comprobar_publicacion_vinculo_corporativo_f1(text,text,text,text,text,text,text)'::regprocedure,
   'vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)'::regprocedure)
  AND has_function_privilege(l.oid,f.oid,'EXECUTE'))
 THEN RAISE EXCEPTION 'CA42: PARO clave=operador actual=incompatible esperado=LOGIN_publicador_exclusivo' USING ERRCODE='42501'; END IF;
 IF (p_operacion_ref ~ '^pubv_[A-Za-z0-9_-]{22,123}$') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_cuenta_ref,'cta_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_persona_ref,'per_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_perfil_ref,'prf_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_vinculo_contexto_ref,'vca_') IS NOT TRUE
 OR vec_contexto_actor_v1.referencia_valida(p_vinculo_corporativo_ref,'vcr_') IS NOT TRUE
 OR vec_contexto_actor_v1.organizacion_ref_valida(p_organizacion_ref) IS NOT TRUE
 THEN RAISE EXCEPTION 'CA42: PARO clave=referencias actual=incompatible esperado=opacas_validas' USING ERRCODE='22023'; END IF;
 -- A→B→C→D y E antes de punteros, igual que C2.3 y los triggers V2.
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:migracion:acreditacion_uso:v2',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:organizacion-corporativa-rrhh:v1',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:vinculo-corporativo-rrhh:v1',0));
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_contexto_actor_v1:proyeccion-corporativa-automatica:v1',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:publicacion-f1:operacion:'||p_operacion_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:publicacion-f1:organizacion:'||p_organizacion_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:publicacion-f1:vinculo:'||p_vinculo_corporativo_ref,0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 SELECT * INTO s FROM vec_contexto_actor_v1.fuente_aprobada_vinculo_f1
 WHERE operacion_ref=p_operacion_ref FOR SHARE;
 IF NOT FOUND OR s.login_nombre IS DISTINCT FROM session_user::name
 OR s.cuenta_ref IS DISTINCT FROM p_cuenta_ref OR s.persona_ref IS DISTINCT FROM p_persona_ref
 OR s.perfil_ref IS DISTINCT FROM p_perfil_ref OR s.vinculo_contexto_ref IS DISTINCT FROM p_vinculo_contexto_ref
 OR s.vinculo_corporativo_ref IS DISTINCT FROM p_vinculo_corporativo_ref
 OR s.organizacion_ref IS DISTINCT FROM p_organizacion_ref
 THEN RAISE EXCEPTION 'CA42: PARO clave=fuente actual=no_acreditada esperado=exacta' USING ERRCODE='42501'; END IF;
 SELECT ov.*,oa.version AS actual_version INTO o
 FROM vec_contexto_actor_v1.organizacion_actual oa
 JOIN vec_contexto_actor_v1.organizacion_versiones ov ON ov.organizacion_ref=oa.organizacion_ref AND ov.version=oa.version
 WHERE oa.organizacion_ref=s.organizacion_ref FOR SHARE OF oa,ov;
 SELECT cv.*,ca.version AS actual_version INTO c FROM vec_contexto_actor_v1.proyeccion_cuenta_actual ca
 JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones cv ON cv.cuenta_ref=ca.cuenta_ref AND cv.version=ca.version
 WHERE ca.cuenta_ref=s.cuenta_ref FOR SHARE OF ca,cv;
 SELECT xv.*,xa.version AS actual_version INTO x FROM vec_contexto_actor_v1.persona_actual xa
 JOIN vec_contexto_actor_v1.persona_versiones xv ON xv.persona_ref=xa.persona_ref AND xv.version=xa.version
 WHERE xa.persona_ref=s.persona_ref FOR SHARE OF xa,xv;
 SELECT pv.*,pa.version AS actual_version INTO p FROM vec_contexto_actor_v1.perfil_actual pa
 JOIN vec_contexto_actor_v1.perfil_versiones pv ON pv.perfil_ref=pa.perfil_ref AND pv.version=pa.version
 WHERE pa.perfil_ref=s.perfil_ref FOR SHARE OF pa,pv;
 SELECT vv.*,va.version AS actual_version INTO v FROM vec_contexto_actor_v1.vinculo_contexto_actual va
 JOIN vec_contexto_actor_v1.vinculo_contexto_versiones vv ON vv.vinculo_ref=va.vinculo_ref AND vv.version=va.version
 WHERE va.vinculo_ref=s.vinculo_contexto_ref FOR SHARE OF va,vv;
 SELECT vv.*,va.version AS actual_version INTO vc FROM vec_contexto_actor_v1.vinculo_corporativo_actual va
 JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones vv ON vv.vinculo_corporativo_ref=va.vinculo_corporativo_ref AND vv.version=va.version
 WHERE va.cuenta_ref=s.cuenta_ref AND va.perfil_ref=s.perfil_ref
 AND va.superficie='interna_corporativa' AND va.uso='consulta_rrhh' FOR SHARE OF va,vv;
 SELECT * INTO pr FROM vec_contexto_actor_v1.procedencias
 WHERE procedencia_ref=s.procedencia_ref AND procedencia_version=s.procedencia_version FOR SHARE;
 ahora:=clock_timestamp();
 IF o.organizacion_ref IS NULL OR c.cuenta_ref IS NULL OR x.persona_ref IS NULL
 OR p.perfil_ref IS NULL OR v.vinculo_ref IS NULL OR pr.procedencia_ref IS NULL
 OR c.estado<>'activo' OR x.estado<>'activo' OR p.estado<>'activo' OR v.estado<>'activo' OR o.estado<>'activo'
 OR c.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR x.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR p.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR v.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR o.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR pr.procedencia_autoridad<>'autoridad_maestra_acreditada'
 OR p.persona_ref IS DISTINCT FROM s.persona_ref
 OR v.cuenta_ref IS DISTINCT FROM s.cuenta_ref OR v.persona_ref IS DISTINCT FROM s.persona_ref
 OR v.perfil_ref IS DISTINCT FROM s.perfil_ref
 OR (pr.procedencia_ref,pr.procedencia_version,pr.procedencia_huella_sha256,pr.procedencia_autoridad)
 IS DISTINCT FROM (s.procedencia_ref,s.procedencia_version,s.procedencia_huella_sha256,s.procedencia_autoridad)
 OR ahora<greatest(s.vigente_desde,c.vigente_desde,x.vigente_desde,p.vigente_desde,v.vigente_desde,o.vigente_desde)
 OR ahora>=least(s.vigente_hasta,c.vigente_hasta,x.vigente_hasta,p.vigente_hasta,v.vigente_hasta,o.vigente_hasta)
 OR s.vigente_desde<greatest(c.vigente_desde,x.vigente_desde,p.vigente_desde,v.vigente_desde,o.vigente_desde)
 OR s.vigente_hasta>least(c.vigente_hasta,x.vigente_hasta,p.vigente_hasta,v.vigente_hasta,o.vigente_hasta)
 THEN RAISE EXCEPTION 'CA42: PARO clave=fuentes actual=incompatible esperado=activas_vigentes_exactas' USING ERRCODE='42501'; END IF;
 IF vc.vinculo_corporativo_ref IS NOT NULL AND
  (vc.vinculo_corporativo_ref IS DISTINCT FROM s.vinculo_corporativo_ref
   OR vc.perfil_ref IS DISTINCT FROM s.perfil_ref OR vc.persona_ref IS DISTINCT FROM s.persona_ref
   OR vc.organizacion_ref IS DISTINCT FROM s.organizacion_ref OR vc.estado<>'activo'
   OR vc.vinculo_contexto_ref IS DISTINCT FROM s.vinculo_contexto_ref
   OR vc.procedencia_ref IS DISTINCT FROM s.procedencia_ref
   OR vc.procedencia_version IS DISTINCT FROM s.procedencia_version
   OR vc.procedencia_huella_sha256 IS DISTINCT FROM s.procedencia_huella_sha256
   OR vc.procedencia_autoridad IS DISTINCT FROM s.procedencia_autoridad
   OR ahora<vc.vigente_desde OR ahora>=vc.vigente_hasta
   OR vc.cuenta_version IS DISTINCT FROM c.version OR vc.persona_version IS DISTINCT FROM x.version
   OR vc.perfil_version IS DISTINCT FROM p.version OR vc.vinculo_contexto_version IS DISTINCT FROM v.version
   OR vc.organizacion_version IS DISTINCT FROM o.version)
 THEN RAISE EXCEPTION 'CA42: PARO clave=puntero actual=otro_perfil_o_vinculo esperado=ausente_o_replay_exacto' USING ERRCODE='40001'; END IF;
 IF vc.vinculo_corporativo_ref IS NULL AND EXISTS(
  SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_versiones
  WHERE vinculo_corporativo_ref=s.vinculo_corporativo_ref)
 THEN RAISE EXCEPTION 'CA42: PARO clave=historia actual=existente esperado=alta_inicial' USING ERRCODE='40001'; END IF;
 plan:=jsonb_build_object('esquema','vec.ca.publicacion-vinculo-f1.v1','operacion_ref',s.operacion_ref,
  'login',s.login_nombre,'cuenta_ref',s.cuenta_ref,'cuenta_version',c.version,
  'persona_ref',s.persona_ref,'persona_version',x.version,'perfil_ref',s.perfil_ref,'perfil_version',p.version,
  'vinculo_contexto_ref',s.vinculo_contexto_ref,'vinculo_contexto_version',v.version,
  'vinculo_corporativo_ref',s.vinculo_corporativo_ref,'organizacion_ref',s.organizacion_ref,'organizacion_version',o.version,
  'organizacion_procedencia_ref',o.procedencia_ref,'organizacion_procedencia_version',o.procedencia_version,
  'organizacion_procedencia_huella_sha256',o.procedencia_huella_sha256,
  'procedencia_ref',s.procedencia_ref,'procedencia_version',s.procedencia_version,
  'procedencia_huella_sha256',s.procedencia_huella_sha256,'fuente_huella_sha256',s.fuente_huella_sha256,
  'aprobacion_ref',s.aprobacion_ref,'aprobacion_huella_sha256',s.aprobacion_huella_sha256,
  'vigente_desde',s.vigente_desde,'vigente_hasta',s.vigente_hasta,'version',1);
 sha:=encode(sha256(convert_to(plan::text,'UTF8')),'hex');
 IF vc.vinculo_corporativo_ref IS NOT NULL AND NOT EXISTS(
  SELECT 1 FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1 prf
  WHERE prf.operacion_ref=s.operacion_ref AND prf.plan_sha256=sha
   AND prf.vinculo_corporativo_ref=s.vinculo_corporativo_ref AND prf.version=1)
 THEN RAISE EXCEPTION 'CA42: PARO clave=replay actual=divergente esperado=prueba_original' USING ERRCODE='40001'; END IF;
 RETURN plan||jsonb_build_object('plan_sha256',sha,'estado',CASE WHEN vc.vinculo_corporativo_ref IS NULL THEN 'publicable' ELSE 'replay' END);
END $function$;

CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(p_operacion_ref text, p_cuenta_ref text, p_persona_ref text, p_perfil_ref text, p_vinculo_contexto_ref text, p_vinculo_corporativo_ref text, p_organizacion_ref text, p_plan_sha256 text)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '10s'
AS $function$
DECLARE p jsonb; a record; previo vec_contexto_actor_v1.prueba_publicacion_vinculo_f1;
 evento text; corr text;
BEGIN
 BEGIN
 p:=vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref);
 IF p_plan_sha256 !~ '^[0-9a-f]{64}$' OR p_plan_sha256 IS DISTINCT FROM p->>'plan_sha256'
 THEN RAISE EXCEPTION 'CA42: PARO clave=plan actual=deriva esperado=huella_comprobada' USING ERRCODE='40001'; END IF;
 SELECT * INTO previo FROM vec_contexto_actor_v1.prueba_publicacion_vinculo_f1 WHERE operacion_ref=p_operacion_ref FOR SHARE;
 IF FOUND THEN
  IF p->>'estado'<>'replay' OR previo.plan_sha256 IS DISTINCT FROM p_plan_sha256
   OR previo.vinculo_corporativo_ref IS DISTINCT FROM p_vinculo_corporativo_ref OR previo.version<>1
  THEN RAISE EXCEPTION 'CA42: PARO clave=replay actual=divergente esperado=original' USING ERRCODE='40001'; END IF;
  RETURN jsonb_build_object('codigo','CA42','estado','replay','huella',previo.plan_sha256,
   'auditoria_ref',previo.auditoria_ref,'auditoria_huella_sha256',previo.auditoria_huella_sha256,'evento_ref',previo.evento_ref);
 END IF;
 IF p->>'estado'<>'publicable' THEN RAISE EXCEPTION 'CA42: PARO clave=puntero actual=ocupado esperado=alta_inicial' USING ERRCODE='40001'; END IF;
 evento:='evento_'||substr(encode(sha256(convert_to('vec.ca.publicacion-vinculo-f1:'||p_operacion_ref,'UTF8')),'hex'),1,32);
 corr:='correlacion_'||substr(p_plan_sha256,33,32);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_versiones(
  vinculo_corporativo_ref,version,cuenta_ref,cuenta_version,persona_ref,persona_version,
  perfil_ref,perfil_version,vinculo_contexto_ref,vinculo_contexto_version,organizacion_ref,organizacion_version,
  organizacion_procedencia_ref,organizacion_procedencia_version,organizacion_procedencia_huella_sha256,
  organizacion_procedencia_autoridad,superficie,uso,procedencia_ref,procedencia_version,
  procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES(p->>'vinculo_corporativo_ref',1,p->>'cuenta_ref',(p->>'cuenta_version')::numeric,
  p->>'persona_ref',(p->>'persona_version')::numeric,p->>'perfil_ref',(p->>'perfil_version')::numeric,
  p->>'vinculo_contexto_ref',(p->>'vinculo_contexto_version')::numeric,p->>'organizacion_ref',(p->>'organizacion_version')::numeric,
  p->>'organizacion_procedencia_ref',(p->>'organizacion_procedencia_version')::numeric,
  p->>'organizacion_procedencia_huella_sha256','autoridad_maestra_acreditada',
  'interna_corporativa','consulta_rrhh',p->>'procedencia_ref',(p->>'procedencia_version')::numeric,
  p->>'procedencia_huella_sha256','autoridad_maestra_acreditada','activo',(p->>'vigente_desde')::timestamptz,(p->>'vigente_hasta')::timestamptz);
 INSERT INTO vec_contexto_actor_v1.vinculo_corporativo_actual(cuenta_ref,perfil_ref,superficie,uso,vinculo_corporativo_ref,version)
 VALUES(p->>'cuenta_ref',p->>'perfil_ref','interna_corporativa','consulta_rrhh',p->>'vinculo_corporativo_ref',1);
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(
  jsonb_build_object('evento_ref',evento,'operador_login',session_user::text,'plan_sha256',p_plan_sha256,
   'aprobacion_ref',p->>'aprobacion_ref','aprobacion_sha256',p->>'aprobacion_huella_sha256',
   'procedencia_sha256',p->>'procedencia_huella_sha256','version','1','correlacion_ref',corr,
   'tipo','publicacion','estado','publicado'));
 INSERT INTO vec_contexto_actor_v1.prueba_publicacion_vinculo_f1(
  operacion_ref,plan_sha256,vinculo_corporativo_ref,version,auditoria_ref,auditoria_huella_sha256,evento_ref,registrada_en)
 VALUES(p_operacion_ref,p_plan_sha256,p_vinculo_corporativo_ref,1,a.auditoria_ref,a.huella_sha256,evento,a.registrada_en);
 INSERT INTO vec_contexto_actor_v1.buzon_publicacion_vinculo_f1(evento_ref,operacion_ref,auditoria_ref,plan_sha256,creada_en)
 VALUES(evento,p_operacion_ref,a.auditoria_ref,p_plan_sha256,a.registrada_en);
 RETURN jsonb_build_object('codigo','CA42','estado','publicado','huella',p_plan_sha256,
  'auditoria_ref',a.auditoria_ref,'auditoria_huella_sha256',a.huella_sha256,'evento_ref',evento);
 EXCEPTION WHEN SQLSTATE '42501' OR SQLSTATE '40001' THEN
  RETURN vec_contexto_actor_v1.auditar_denegacion_publicacion_vinculo_f1('publicar',
   p_operacion_ref,p_cuenta_ref,p_persona_ref,p_perfil_ref,p_vinculo_contexto_ref,p_vinculo_corporativo_ref,p_organizacion_ref);
 END;
END $function$;

RESET ROLE;
DO $post$
DECLARE malo integer;
BEGIN
 SELECT count(*) INTO malo FROM (VALUES
  ('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)','fcb368c2a6d5787b865af9ad8f9cc8c3479a4d6705ad042f4620152658d232f1'),
  ('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)','52ceeca135a7a8c4e6c967e7cbd39bef79534c89c17bcef976eb37e5c77b8900'),
  ('vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(text,text,text,numeric,text)','15c7c93e574bd08c7d98d7877dfc04b41ce67afd2ada6e1c2d91d5472211e573'),
  ('vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(text,text,text,text,text,text,text)','6a70b647c03b13f4be92d705c905e7c5156f2d0b972599ce39f06fa65831e4c7'),
  ('vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)','7b958b1614b60e29662e0fe5f15e0e689b4e2bb6f6d4e81d9101382eec29c820')
 ) e(firma,huella) LEFT JOIN pg_proc p ON p.oid=to_regprocedure(e.firma)
 WHERE p.oid IS NULL OR p.proowner<>'vec_contexto_actor_v1_propietario'::regrole
  OR NOT p.prosecdef OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM e.huella
  OR p.proconfig IS DISTINCT FROM CASE
   WHEN e.firma LIKE '%revalidar_vinculo_corporativo_rrhh_v1%' THEN ARRAY['search_path=pg_catalog, pg_temp']::text[]
   WHEN e.firma LIKE '%acreditar_organizacion_destino_certificado_v1%' THEN
    ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s']::text[]
   WHEN e.firma LIKE '%cotejar_contexto_lectura_bolsa_actual_v1%' THEN
    ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s']::text[]
   ELSE ARRAY['search_path=pg_catalog, pg_temp','row_security=on','lock_timeout=2s','statement_timeout=10s']::text[] END;
 IF malo<>0 THEN RAISE EXCEPTION 'CA43: PARO clave=funciones_actuales actual=deriva esperado=literal_perfil' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND conname='vinculo_corporativo_actual_pk' AND
  pg_get_constraintdef(oid,false)='PRIMARY KEY (cuenta_ref, perfil_ref, superficie, uso)')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND conname='vinculo_corporativo_actual_version_fk' AND
  pg_get_constraintdef(oid,false)='FOREIGN KEY (cuenta_ref, perfil_ref, superficie, uso, vinculo_corporativo_ref, version) REFERENCES vec_contexto_actor_v1.vinculo_corporativo_versiones(cuenta_ref, perfil_ref, superficie, uso, vinculo_corporativo_ref, version) MATCH FULL')
 OR NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='vec_contexto_actor_v1.vinculo_corporativo_versiones'::regclass
  AND conname='vinculo_corporativo_versiones_actual_perfil_uq' AND
  pg_get_constraintdef(oid,false)='UNIQUE (cuenta_ref, perfil_ref, superficie, uso, vinculo_corporativo_ref, version)')
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_corporativo_actual a
  JOIN vec_contexto_actor_v1.vinculo_corporativo_versiones v
   ON v.vinculo_corporativo_ref=a.vinculo_corporativo_ref AND v.version=a.version
  WHERE a.perfil_ref IS DISTINCT FROM v.perfil_ref)
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND relowner='vec_contexto_actor_v1_propietario'::regrole AND relrowsecurity AND relforcerowsecurity AND relacl IS NULL)
 OR (SELECT count(*) FROM pg_trigger WHERE tgrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND NOT tgisinternal AND tgenabled='O'
  AND tgname IN ('serializar_mutacion_punteros_actuales_v2','avanzar_generacion_punteros_actuales_v2','puntero_actual_no_truncable_v2'))<>3
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid='vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(text,text,text)'::regprocedure
  AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='fd4a536d15de5358884e0625bbe41b360b02344410f766dcccb02b809cad02f3')
 THEN RAISE EXCEPTION 'CA43: PARO clave=estructura actual=deriva esperado=perfil_exacto' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM pg_policy WHERE polrelid='vec_contexto_actor_v1.vinculo_corporativo_actual'::regclass
  AND polname='acceso_propietario_exacto' AND polpermissive
  AND polroles=ARRAY['vec_contexto_actor_v1_propietario'::regrole::oid])<>1
 OR EXISTS(SELECT 1 FROM (VALUES
  ('vec_contexto_actor_v1.revalidar_vinculo_corporativo_rrhh_v1(text,text,text,text,numeric)',
   ARRAY['vec_contexto_actor_v1_propietario'::regrole::oid,'vec_contexto_actor_v1_runtime'::regrole::oid]),
  ('vec_contexto_actor_v1.cotejar_contexto_lectura_bolsa_actual_v1(bytea)',
   ARRAY['vec_contexto_actor_v1_propietario'::regrole::oid,'vec_autorizacion_atestada_v3_propietario'::regrole::oid]),
  ('vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(text,text,text,numeric,text)',
   ARRAY['vec_contexto_actor_v1_propietario'::regrole::oid,'vec_autorizacion_propietario'::regrole::oid]),
  ('vec_contexto_actor_v1.preimagen_publicacion_vinculo_f1(text,text,text,text,text,text,text)',
   ARRAY['vec_contexto_actor_v1_propietario'::regrole::oid]),
  ('vec_contexto_actor_v1.publicar_vinculo_corporativo_desde_fuente_v1(text,text,text,text,text,text,text,text)',
   ARRAY['vec_contexto_actor_v1_propietario'::regrole::oid,'vec_contexto_actor_corporativo_rrhh_publicador'::regrole::oid])
 ) e(firma,autorizados) JOIN pg_proc p ON p.oid=to_regprocedure(e.firma)
 LEFT JOIN LATERAL aclexplode(p.proacl) a ON true
 GROUP BY e.firma,e.autorizados
 HAVING count(a.grantee)<>cardinality(e.autorizados)
  OR bool_and(a.grantee=ANY(e.autorizados) AND a.grantor='vec_contexto_actor_v1_propietario'::regrole::oid
   AND a.privilege_type='EXECUTE' AND NOT a.is_grantable) IS NOT TRUE)
 THEN RAISE EXCEPTION 'CA43: PARO clave=ACL actual=deriva esperado=misma_autoridad' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
