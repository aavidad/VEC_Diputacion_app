\set ON_ERROR_STOP on
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $pure$
BEGIN
 IF vec_autorizacion_atestada_v3.detalle_gobierno_usuarios_valido_v1(jsonb_build_object('preimagen_sha256',repeat('a',64),'configuracion_origen_ref','confianza:atestacion:ct:desarrollo:2026-10-02','configuracion_destino_ref','confianza:atestacion:ct:desarrollo:2026-10-04','claves_sha256',repeat('b',64))) IS DISTINCT FROM true
 THEN RAISE EXCEPTION 'AD188 test: detalle propio rechazado';END IF;
 IF vec_autorizacion_atestada_v3.detalle_gobierno_usuarios_valido_v1(jsonb_build_object('preimagen_sha256',repeat('a',64),'configuracion_origen_ref','x','configuracion_destino_ref','y','claves_sha256',repeat('b',64),'actor_ref','no_permitido')) IS DISTINCT FROM false
 THEN RAISE EXCEPTION 'AD188 test: actor extra aceptado';END IF;
 IF vec_autorizacion_atestada_v3.motivo_intento_gobierno_usuarios_valido_v1('permitido','gobierno_usuarios_replay') IS DISTINCT FROM true
 OR vec_autorizacion_atestada_v3.motivo_intento_gobierno_usuarios_valido_v1('permitido','gobierno_usuarios_denegado') IS DISTINCT FROM false
 THEN RAISE EXCEPTION 'AD188 test: pareja resultado motivo incorrecta';END IF;
 IF EXISTS(SELECT 1 FROM pg_proc f CROSS JOIN LATERAL aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) a WHERE f.oid=to_regprocedure('vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1(text,text,text)') AND a.grantee=0 AND a.privilege_type='EXECUTE')
 OR has_function_privilege('vec_gobierno_usuarios_admin_operador','vec_autorizacion_atestada_v3.efecto_gobierno_usuarios_admin_v1(text,text,text)','EXECUTE')
 OR has_table_privilege('vec_gobierno_usuarios_admin_operador','vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT')
 THEN RAISE EXCEPTION 'AD188 test: ACL excesiva';END IF;
END $pure$;
ROLLBACK;
-- Positivo/replay/reinicio/rollback de auditoría y tres GRANT OPTION se ejecutan
-- por el proveedor real con LOGIN/config privados y plan aprobado. No se
-- inventan claves, actas de renovación o decisiones favorables en este vector.
