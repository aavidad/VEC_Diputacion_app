\set ON_ERROR_STOP on
-- Usuarios 000007 (5.08c): la auditoría de denegaciones de frontera admite
-- también las dos rutas de «Mi imagen». Mismos grupos registradores por
-- superficie; no cambia firma, ACL ni propietario. Aplicar tras 000006.
BEGIN;
SET LOCAL ROLE vec_usuarios_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000007',0));
DO $pre$ BEGIN
 IF current_user<>'vec_usuarios_propietario'
    OR to_regclass('vec_usuarios.imagen_actual') IS NULL
    OR to_regclass('vec_usuarios.denegacion_frontera_preferencias') IS NULL
    OR to_regprocedure('vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='vec_usuarios.denegacion_frontera_preferencias'::regclass
      AND conname='denegacion_frontera_preferencias_ruta_check' AND contype='c'
      AND position('/api/vec/usuarios/area-personal/mis-correos' IN pg_get_constraintdef(oid))>0
      AND position('mi-imagen' IN pg_get_constraintdef(oid))=0)
    OR (SELECT count(*) FROM pg_policies WHERE schemaname='vec_usuarios' AND tablename='denegacion_frontera_preferencias')<>2
 THEN RAISE EXCEPTION 'Usuarios 000007: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_usuarios.denegacion_frontera_preferencias IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_usuarios.denegacion_frontera_preferencias DROP CONSTRAINT denegacion_frontera_preferencias_ruta_check;
ALTER TABLE vec_usuarios.denegacion_frontera_preferencias ADD CONSTRAINT denegacion_frontera_preferencias_ruta_check
 CHECK(ruta IN ('/api/vec/usuarios/mis-preferencias','/api/vec/usuarios/area-personal/mis-preferencias',
   '/api/vec/usuarios/mis-correos','/api/vec/usuarios/area-personal/mis-correos',
   '/api/vec/usuarios/mi-imagen','/api/vec/usuarios/area-personal/mi-imagen'));
DROP POLICY propietario_lectura ON vec_usuarios.denegacion_frontera_preferencias;
DROP POLICY propietario_insercion ON vec_usuarios.denegacion_frontera_preferencias;
CREATE POLICY propietario_lectura ON vec_usuarios.denegacion_frontera_preferencias
 FOR SELECT TO vec_usuarios_propietario USING (
  superficie='api.usuarios.preferencias.ruta_exacta'
  AND ruta IN ('/api/vec/usuarios/mis-preferencias','/api/vec/usuarios/area-personal/mis-preferencias',
               '/api/vec/usuarios/mis-correos','/api/vec/usuarios/area-personal/mis-correos',
               '/api/vec/usuarios/mi-imagen','/api/vec/usuarios/area-personal/mi-imagen')
  AND pg_catalog.pg_has_role(session_user,'vec_usuarios_migrador','MEMBER')
 );
CREATE POLICY propietario_insercion ON vec_usuarios.denegacion_frontera_preferencias
 FOR INSERT TO vec_usuarios_propietario WITH CHECK (
  superficie='api.usuarios.preferencias.ruta_exacta'
  AND ((ruta IN ('/api/vec/usuarios/mis-preferencias','/api/vec/usuarios/mis-correos','/api/vec/usuarios/mi-imagen')
        AND pg_catalog.pg_has_role(session_user,'vec_usuarios_registrador_frontera_interno','MEMBER'))
    OR (ruta IN ('/api/vec/usuarios/area-personal/mis-preferencias','/api/vec/usuarios/area-personal/mis-correos','/api/vec/usuarios/area-personal/mi-imagen')
        AND pg_catalog.pg_has_role(session_user,'vec_usuarios_registrador_frontera_externo','MEMBER')))
  AND ((motivo='autenticacion_requerida' AND actor_ref IS NULL)
    OR (motivo='acceso_denegado' AND actor_ref IS NOT NULL))
 );
DO $funcion$
DECLARE
 f oid:='vec_usuarios.registrar_denegacion_preferencias_v1(text,text,text,text,text)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; acl aclitem[];
 ruta_vieja text:=E' IF p_ruta IS NULL OR p_ruta NOT IN (''/api/vec/usuarios/mis-preferencias'',''/api/vec/usuarios/area-personal/mis-preferencias'',\n    ''/api/vec/usuarios/mis-correos'',''/api/vec/usuarios/area-personal/mis-correos'')\n';
 ruta_nueva text:=E' IF p_ruta IS NULL OR p_ruta NOT IN (''/api/vec/usuarios/mis-preferencias'',''/api/vec/usuarios/area-personal/mis-preferencias'',\n    ''/api/vec/usuarios/mis-correos'',''/api/vec/usuarios/area-personal/mis-correos'',\n    ''/api/vec/usuarios/mi-imagen'',''/api/vec/usuarios/area-personal/mi-imagen'')\n';
 grupo_viejo text:=E'  WHEN p_ruta IN (''/api/vec/usuarios/mis-preferencias'',''/api/vec/usuarios/mis-correos'')\n    THEN';
 grupo_nuevo text:=E'  WHEN p_ruta IN (''/api/vec/usuarios/mis-preferencias'',''/api/vec/usuarios/mis-correos'',''/api/vec/usuarios/mi-imagen'')\n    THEN';
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl INTO STRICT original,meta,acl FROM pg_proc p WHERE p.oid=f;
 IF length(original)-length(replace(original,ruta_vieja,''))<>length(ruta_vieja)
    OR length(original)-length(replace(original,grupo_viejo,''))<>length(grupo_viejo)
    OR strpos(original,'mi-imagen')<>0
 THEN RAISE EXCEPTION 'Usuarios 000007: función de frontera incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(original,ruta_vieja,ruta_nueva),grupo_viejo,grupo_nuevo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(actual,ruta_nueva,ruta_vieja),grupo_nuevo,grupo_viejo) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
 THEN RAISE EXCEPTION 'Usuarios 000007: función de frontera alterada fuera de contrato' USING ERRCODE='55000'; END IF;
END $funcion$;
COMMIT;
