\set ON_ERROR_STOP on
-- AD202: conjunto 2 de capacidades ADMIN de AD198. Son las tres audiencias del
-- conjunto 1 (usuarios listar y consultar, lote ordinario de perfiles), en el
-- mismo orden y con los mismos tramos, más la audiencia del gobierno del plan
-- nominal de firma (AD177/AD178): así vec-admin renueva cada día, en una sola
-- operación, también la clave con la que emite esas decisiones. Sólo añade una
-- fila al catálogo cerrado de conjuntos: no toca el núcleo, el CHECK de
-- audiencias (AD178 ya admite la audiencia) ni las funciones de AD198. Requiere
-- AD198 y AD178. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000202',0));
DO $pre$
DECLARE uno record;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD202: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF to_regclass('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)') IS NULL
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=2)
 THEN RAISE EXCEPTION 'AD202: PARO clave=preimagen actual=incompatible esperado=AD198_sin_conjunto_2' USING ERRCODE='55000'; END IF;
 SELECT * INTO uno FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=1;
 IF NOT FOUND
 OR uno.audiencias IS DISTINCT FROM ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1']
 OR uno.segmentos IS DISTINCT FROM ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote']
 THEN RAISE EXCEPTION 'AD202: PARO clave=conjunto_1 actual=distinto esperado=AD198' USING ERRCODE='55000'; END IF;
 -- La tabla de claves debe admitir ya la audiencia del gobierno (AD178).
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated
   AND strpos(pg_get_constraintdef(c.oid,false),'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1')>0)
 THEN RAISE EXCEPTION 'AD202: PARO clave=AD178 actual=audiencia_no_admitida esperado=CHECK_con_gobierno_plan' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos) VALUES
 (2,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma']);
RESET ROLE;
DO $post$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=2
   AND audiencias=ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
    'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1']
   AND segmentos=ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma'])
 THEN RAISE EXCEPTION 'AD202: PARO clave=postimagen esperado=conjunto_2_exacto actual=divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
