\set ON_ERROR_STOP on
-- AD204: conjunto 3 de capacidades ADMIN de AD198. Son las cuatro audiencias
-- del conjunto 2 (usuarios listar y consultar, lote ordinario de perfiles y
-- gobierno del plan nominal de firma), en el mismo orden y con los mismos
-- tramos, más la publicación de cargos competenciales de Personal (AD166 y
-- Personal28): así vec-admin renueva cada día, en una sola operación, también
-- la clave con la que emite esas decisiones. Sólo añade una fila al catálogo
-- cerrado de conjuntos: no toca el núcleo, el CHECK de audiencias (AD166 ya
-- admite la audiencia) ni las funciones de AD198. Requiere AD198, AD202 y
-- AD166. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000204',0));
DO $pre$
DECLARE dos record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD204: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF to_regclass('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)') IS NULL
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=3)
 THEN RAISE EXCEPTION 'AD204: PARO clave=preimagen actual=incompatible esperado=AD198_sin_conjunto_3' USING ERRCODE='55000'; END IF;
 SELECT * INTO dos FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=2;
 IF NOT FOUND
 OR dos.audiencias IS DISTINCT FROM ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1']
 OR dos.segmentos IS DISTINCT FROM ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma']
 THEN RAISE EXCEPTION 'AD204: PARO clave=conjunto_2 actual=distinto esperado=AD202' USING ERRCODE='55000'; END IF;
 -- La tabla de claves debe admitir ya la audiencia de los cargos (AD166).
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid,false),'vec_personal.cargo_competencial.publicar.v1')>0)
 THEN RAISE EXCEPTION 'AD204: PARO clave=AD166 actual=audiencia_no_admitida esperado=CHECK_con_cargo_competencial' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos) VALUES
 (3,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1'],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial']);
RESET ROLE;
DO $post$
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=3
   AND audiencias=ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
    'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1']
   AND segmentos=ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial'])<>1
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=2
   AND audiencias=ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'])
 THEN RAISE EXCEPTION 'AD204: PARO clave=postimagen esperado=conjunto_3_exacto actual=divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
