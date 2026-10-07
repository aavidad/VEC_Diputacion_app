\set ON_ERROR_STOP on
-- AUT-18. Compatibilidad acotada con json.Marshal de los tipos canónicos Go.
-- AUT-17 ya instalada conserva historia: no se modifica ni se reaplica.
-- Admite exclusivamente time.Time cero; no elimina campos ni cambia huellas.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000018',0));
DO $privilegio$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AUT-18: migrador incompatible' USING ERRCODE='55000'; END IF;
END $privilegio$;
SET LOCAL ROLE vec_autorizacion_propietario;

DO $parche$
DECLARE p pg_catalog.pg_proc%ROWTYPE; original text; nuevo text;
        marca text := $marca$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7$marca$;
        sustituto text := $sustituto$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(7+CASE WHEN d ? 'retirada_en' THEN 1 ELSE 0 END)
    OR (d ? 'retirada_en' AND d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)$sustituto$;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc
 WHERE oid=pg_catalog.to_regprocedure('vec_autorizacion.rol_usuarios_externo_acotado_v1(jsonb)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
    OR p.provolatile IS DISTINCT FROM 'i'::"char" OR p.prosecdef IS DISTINCT FROM false
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.octet_length(p.prosrc)<>3381
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM '3ffc0cf7e28104bd5a82e79183929b0a5b92b6674b655f8c887135366f15bd98'
    OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl))<>1
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
       WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole)
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 THEN RAISE EXCEPTION 'AUT-18: preimagen rol_usuarios_externo_acotado_v1 incompatible' USING ERRCODE='55000'; END IF;
 original:=pg_catalog.pg_get_functiondef(p.oid);
 IF (length(p.prosrc)-length(replace(p.prosrc,marca,'')))/length(marca)<>1
    OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT-18: marca rol_usuarios_externo_acotado_v1 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustituto);
 EXECUTE nuevo;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc q WHERE q.oid=p.oid
    AND q.proowner=p.proowner AND q.prolang=p.prolang AND q.provolatile=p.provolatile
    AND q.prosecdef=p.prosecdef AND q.proconfig=p.proconfig AND q.proacl=p.proacl
    AND q.prorettype=p.prorettype AND q.proretset=p.proretset
    AND q.proargtypes=p.proargtypes AND q.proallargtypes IS NOT DISTINCT FROM p.proallargtypes
    AND q.proargmodes IS NOT DISTINCT FROM p.proargmodes AND q.proargnames IS NOT DISTINCT FROM p.proargnames
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(q.prosrc,'UTF8')),'hex')='6a32d06608d62cb3caaaba1c15e3600b2ab998602154939821c29279a89d2fc8')
 THEN RAISE EXCEPTION 'AUT-18: reconstrucción rol_usuarios_externo_acotado_v1 incompatible' USING ERRCODE='55000'; END IF;
END $parche$;

DO $parche$
DECLARE p pg_catalog.pg_proc%ROWTYPE; original text; nuevo text;
        marca text := $marca$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>7
    OR d ?| ARRAY['retirada_por','retirada_en','retirada_ref','motivo_retirada_codigo']$marca$;
        sustituto text := $sustituto$(SELECT count(*) FROM pg_catalog.jsonb_object_keys(d))<>(7+CASE WHEN d ? 'retirada_en' THEN 1 ELSE 0 END)
    OR (d ? 'retirada_en' AND d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)
    OR d ?| ARRAY['retirada_por','retirada_ref','motivo_retirada_codigo']$sustituto$;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc
 WHERE oid=pg_catalog.to_regprocedure('vec_autorizacion.publicar_rol_usuarios_externo_v1(bytea,text,bytea,text,numeric,text,text,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
    OR p.provolatile IS DISTINCT FROM 'v'::"char" OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.octet_length(p.prosrc)<>5994
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM '931ff10febb4c4fd3a819bcb7f766180943f98e74b109b594bc7e70422317461'
    OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl))<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
       WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole, 'vec_autorizacion_publicador_usuarios_externo'::regrole)
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 THEN RAISE EXCEPTION 'AUT-18: preimagen publicar_rol_usuarios_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 original:=pg_catalog.pg_get_functiondef(p.oid);
 IF (length(p.prosrc)-length(replace(p.prosrc,marca,'')))/length(marca)<>1
    OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT-18: marca publicar_rol_usuarios_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustituto);
 EXECUTE nuevo;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc q WHERE q.oid=p.oid
    AND q.proowner=p.proowner AND q.prolang=p.prolang AND q.provolatile=p.provolatile
    AND q.prosecdef=p.prosecdef AND q.proconfig=p.proconfig AND q.proacl=p.proacl
    AND q.prorettype=p.prorettype AND q.proretset=p.proretset
    AND q.proargtypes=p.proargtypes AND q.proallargtypes IS NOT DISTINCT FROM p.proallargtypes
    AND q.proargmodes IS NOT DISTINCT FROM p.proargmodes AND q.proargnames IS NOT DISTINCT FROM p.proargnames
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(q.prosrc,'UTF8')),'hex')='bc5308a55c5c77fc1ce66bcadddf7e0db5729df715b2781b52a8d506688201c0')
 THEN RAISE EXCEPTION 'AUT-18: reconstrucción publicar_rol_usuarios_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
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
    OR (d ? 'revocada_en' AND d->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb)$sustituto$;
BEGIN
 SELECT * INTO p FROM pg_catalog.pg_proc
 WHERE oid=pg_catalog.to_regprocedure('vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bytea,text,bigint,text,text,text,text)');
 IF p.oid IS NULL OR p.proowner IS DISTINCT FROM 'vec_autorizacion_propietario'::regrole
    OR p.prolang IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
    OR p.provolatile IS DISTINCT FROM 'v'::"char" OR p.prosecdef IS DISTINCT FROM true
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
    OR pg_catalog.octet_length(p.prosrc)<>5907
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
       IS DISTINCT FROM 'c5ccd886dfff4b811456bcdb44162db9ff4e6e86c07e3f52683520fea4653962'
    OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl))<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
       WHERE a.grantee NOT IN ('vec_autorizacion_propietario'::regrole, 'vec_autorizacion_publicador_usuarios_externo'::regrole)
          OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
 THEN RAISE EXCEPTION 'AUT-18: preimagen publicar_asignacion_usuarios_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 original:=pg_catalog.pg_get_functiondef(p.oid);
 IF (length(p.prosrc)-length(replace(p.prosrc,marca,'')))/length(marca)<>1
    OR (length(original)-length(replace(original,marca,'')))/length(marca)<>1
 THEN RAISE EXCEPTION 'AUT-18: marca publicar_asignacion_usuarios_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustituto);
 EXECUTE nuevo;
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc q WHERE q.oid=p.oid
    AND q.proowner=p.proowner AND q.prolang=p.prolang AND q.provolatile=p.provolatile
    AND q.prosecdef=p.prosecdef AND q.proconfig=p.proconfig AND q.proacl=p.proacl
    AND q.prorettype=p.prorettype AND q.proretset=p.proretset
    AND q.proargtypes=p.proargtypes AND q.proallargtypes IS NOT DISTINCT FROM p.proallargtypes
    AND q.proargmodes IS NOT DISTINCT FROM p.proargmodes AND q.proargnames IS NOT DISTINCT FROM p.proargnames
    AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(q.prosrc,'UTF8')),'hex')='c830fc51eeb8e1a841ba33a176c6ded203bc9e8c2627c7d01c6a24fba48e17e1')
 THEN RAISE EXCEPTION 'AUT-18: reconstrucción publicar_asignacion_usuarios_externo_v1 incompatible' USING ERRCODE='55000'; END IF;
END $parche$;
COMMIT;
