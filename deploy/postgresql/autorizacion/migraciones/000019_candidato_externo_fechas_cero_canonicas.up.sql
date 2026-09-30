\set ON_ERROR_STOP on
-- AUT-19. Compatibilidad acotada con json.Marshal de los tipos canónicos Go.
-- AUT-16 ya instalada conserva historia: no se modifica ni se reaplica.
-- Admite ausencia, null y time.Time cero en las fechas opcionales.
-- Conserva el documento original y su huella; una fecha efectiva sigue denegada.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000019',0));
DO $privilegio$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT-19: migrador incompatible' USING ERRCODE='55000'; END IF;
END $privilegio$;
SET LOCAL ROLE vec_autorizacion_propietario;

DO $parche$
DECLARE p pg_catalog.pg_proc%ROWTYPE; original text; nuevo text;
        marca text := $marca$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7$marca$;
        sustituto text := $sustituto$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(7+CASE WHEN d ? 'retirada_en' THEN 1 ELSE 0 END)
    OR (d ? 'retirada_en' AND d->'retirada_en' IS DISTINCT FROM 'null'::jsonb AND d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)$sustituto$;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc
 WHERE oid=pg_catalog.to_regprocedure('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
    OR p.provolatile IS DISTINCT FROM 'i'::"char" OR p.prosecdef IS DISTINCT FROM false
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.octet_length(p.prosrc)<>3399
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM '03ad572190700238595e8dab70de7613cdb2b220572ded9864ece3d24653cc48'
    OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl))<>1
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
       WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole)
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 THEN RAISE EXCEPTION 'AUT-19: preimagen rol_candidato_externo_acotado_v1 incompatible' USING ERRCODE='55000'; END IF;
 original:=pg_catalog.pg_get_functiondef(p.oid);
 IF (length(p.prosrc)-length(replace(p.prosrc,marca,'')))/length(marca)<>1
    OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT-19: marca rol_candidato_externo_acotado_v1 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustituto);
 EXECUTE nuevo;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc q WHERE q.oid=p.oid
    AND q.proowner=p.proowner AND q.prolang=p.prolang AND q.provolatile=p.provolatile
    AND q.prosecdef=p.prosecdef AND q.proconfig=p.proconfig AND q.proacl=p.proacl
    AND q.prorettype=p.prorettype AND q.proretset=p.proretset
    AND q.proargtypes=p.proargtypes AND q.proallargtypes IS NOT DISTINCT FROM p.proallargtypes
    AND q.proargmodes IS NOT DISTINCT FROM p.proargmodes AND q.proargnames IS NOT DISTINCT FROM p.proargnames
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(q.prosrc,'UTF8')),'hex')='4212e0388ee883840011ea7874079e24eac5c947b2e8cc3c2f71921e613965da')
 THEN RAISE EXCEPTION 'AUT-19: reconstrucción rol_candidato_externo_acotado_v1 incompatible' USING ERRCODE='55000'; END IF;
END $parche$;

DO $parche$
DECLARE p pg_catalog.pg_proc%ROWTYPE; original text; nuevo text;
        marca text := $marca$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7
    OR d ?| ARRAY['retirada_por','retirada_en','retirada_ref','motivo_retirada_codigo']$marca$;
        sustituto text := $sustituto$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(7+CASE WHEN d ? 'retirada_en' THEN 1 ELSE 0 END)
    OR (d ? 'retirada_en' AND d->'retirada_en' IS DISTINCT FROM 'null'::jsonb AND d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR d ?| ARRAY['retirada_por','retirada_ref','motivo_retirada_codigo']$sustituto$;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc
 WHERE oid=pg_catalog.to_regprocedure('vec_autorizacion.publicar_rol_candidato_externo_v1(bytea,text,bytea,text,numeric,text,text,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
    OR p.provolatile IS DISTINCT FROM 'v'::"char" OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.octet_length(p.prosrc)<>7980
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM 'ff4e7892d445b9953aa2fd49e3aa83f5b164ced4644c177fe0c5c691deebfd31'
    OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl))<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
       WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole, 'vec_autorizacion_publicador_candidato_externo'::regrole)
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 THEN RAISE EXCEPTION 'AUT-19: preimagen publicar_rol_candidato_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 original:=pg_catalog.pg_get_functiondef(p.oid);
 IF (length(p.prosrc)-length(replace(p.prosrc,marca,'')))/length(marca)<>1
    OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT-19: marca publicar_rol_candidato_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustituto);
 EXECUTE nuevo;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc q WHERE q.oid=p.oid
    AND q.proowner=p.proowner AND q.prolang=p.prolang AND q.provolatile=p.provolatile
    AND q.prosecdef=p.prosecdef AND q.proconfig=p.proconfig AND q.proacl=p.proacl
    AND q.prorettype=p.prorettype AND q.proretset=p.proretset
    AND q.proargtypes=p.proargtypes AND q.proallargtypes IS NOT DISTINCT FROM p.proallargtypes
    AND q.proargmodes IS NOT DISTINCT FROM p.proargmodes AND q.proargnames IS NOT DISTINCT FROM p.proargnames
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(q.prosrc,'UTF8')),'hex')='b58b39f2c35341a545640bfde366f07a89ee65831306950c7f3ea342ee846ff0')
 THEN RAISE EXCEPTION 'AUT-19: reconstrucción publicar_rol_candidato_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
END $parche$;

DO $parche$
DECLARE p pg_catalog.pg_proc%ROWTYPE; original text; nuevo text;
        marca text := $marca$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>11
    OR NOT d ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref',
      'estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en']
    OR d->>'estado' IS DISTINCT FROM 'activa'
    OR d->>'revocada_por' IS NOT NULL OR d->>'revocada_en' IS NOT NULL OR d->>'revocacion_ref' IS NOT NULL$marca$;
        sustituto text := $sustituto$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(11+CASE WHEN d ? 'revocada_en' THEN 1 ELSE 0 END)
    OR NOT d ?& ARRAY['asignacion_id','version','perfil_activo_ref','principal_id','version_rol_ref',
      'estado','ambitos','vigente_desde','vigente_hasta','emitida_por','emitida_en']
    OR d->>'estado' IS DISTINCT FROM 'activa'
    OR d ?| ARRAY['revocada_por','revocacion_ref']
    OR (d ? 'revocada_en' AND d->'revocada_en' IS DISTINCT FROM 'null'::jsonb AND d->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)$sustituto$;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc
 WHERE oid=pg_catalog.to_regprocedure('vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
    OR p.provolatile IS DISTINCT FROM 'v'::"char" OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.octet_length(p.prosrc)<>5812
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM '3d3df454d36b94c02847ffab2590db00b66eb42df8a2b33f5f767782b9bbc53e'
    OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl))<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
       WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole, 'vec_autorizacion_publicador_candidato_externo'::regrole)
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 THEN RAISE EXCEPTION 'AUT-19: preimagen publicar_asignacion_candidato_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 original:=pg_catalog.pg_get_functiondef(p.oid);
 IF (length(p.prosrc)-length(replace(p.prosrc,marca,'')))/length(marca)<>1
    OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT-19: marca publicar_asignacion_candidato_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustituto);
 EXECUTE nuevo;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc q WHERE q.oid=p.oid
    AND q.proowner=p.proowner AND q.prolang=p.prolang AND q.provolatile=p.provolatile
    AND q.prosecdef=p.prosecdef AND q.proconfig=p.proconfig AND q.proacl=p.proacl
    AND q.prorettype=p.prorettype AND q.proretset=p.proretset
    AND q.proargtypes=p.proargtypes AND q.proallargtypes IS NOT DISTINCT FROM p.proallargtypes
    AND q.proargmodes IS NOT DISTINCT FROM p.proargmodes AND q.proargnames IS NOT DISTINCT FROM p.proargnames
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(q.prosrc,'UTF8')),'hex')='4c1801f62da2beb36fea79199fbf3006572ca1f1444c3030729e14458e873811')
 THEN RAISE EXCEPTION 'AUT-19: reconstrucción publicar_asignacion_candidato_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
END $parche$;
COMMIT;
