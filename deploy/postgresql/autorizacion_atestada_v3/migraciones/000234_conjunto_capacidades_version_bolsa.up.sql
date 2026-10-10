\set ON_ERROR_STOP on
-- AD234: conjunto 5 de capacidades ADMIN de AD198. Son las seis audiencias
-- del conjunto 4 (usuarios listar y consultar, lote ordinario de perfiles,
-- gobierno del plan nominal de firma, cargos competenciales y certificados
-- nominales), en el mismo orden y con los mismos tramos, más las dos de la
-- versión de rol de Bolsa (B1, AD227): propuesta y cierre. Así
-- vec-gobierno-usuarios-admin publica cada día, en la misma operación de
-- AD198, también las dos claves con las que vec-admin emite esas decisiones.
-- Sólo añade una fila al catálogo cerrado de conjuntos: no toca el núcleo
-- V3, el CHECK de audiencias (AD227 ya admite las dos) ni las funciones de
-- AD198. Requiere AD198, AD205 y AD227. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000234',0));
DO $pre$
DECLARE cuatro record;
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD234: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regclass('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_version_rol_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version>=5)
 THEN RAISE EXCEPTION 'AD234: PARO clave=preimagen actual=incompatible esperado=AD198_AD227_sin_conjunto_5' USING ERRCODE='55000'; END IF;
 SELECT * INTO cuatro FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=4;
 IF NOT FOUND
 OR cuatro.audiencias IS DISTINCT FROM ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1']
 OR cuatro.segmentos IS DISTINCT FROM ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal']
 THEN RAISE EXCEPTION 'AD234: PARO clave=conjunto_4 actual=distinto esperado=AD205' USING ERRCODE='55000'; END IF;
 -- La tabla de claves debe admitir ya las dos audiencias de B1 (AD227).
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated
   AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,false),'''vec_autorizacion.versionar_rol_bolsa.propuesta.v1''::text')>0
   AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,false),'''vec_autorizacion.versionar_rol_bolsa.cierre.v1''::text')>0)
 THEN RAISE EXCEPTION 'AD234: PARO clave=AD227 actual=audiencias_no_admitidas esperado=CHECK_con_version_rol_bolsa' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos) VALUES
 (5,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1',
   'vec_autorizacion.versionar_rol_bolsa.propuesta.v1','vec_autorizacion.versionar_rol_bolsa.cierre.v1'],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal',
   'perfiles:version-bolsa:propuesta','perfiles:version-bolsa:cierre']);
RESET ROLE;
DO $post$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=5
   AND audiencias=ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
    'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1',
    'vec_autorizacion.versionar_rol_bolsa.propuesta.v1','vec_autorizacion.versionar_rol_bolsa.cierre.v1']
   AND segmentos=ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal',
    'perfiles:version-bolsa:propuesta','perfiles:version-bolsa:cierre'])<>1
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)<>5
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=4
   AND audiencias=ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
    'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1'])
 -- El catálogo de conjuntos sigue sin ningún permiso fuera de su propietario.
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(c.relacl,pg_catalog.acldefault('r',c.relowner))) a
   WHERE c.oid='vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1'::regclass AND a.grantee<>c.relowner)
 THEN RAISE EXCEPTION 'AD234: PARO clave=postimagen esperado=conjunto_5_exacto actual=divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
