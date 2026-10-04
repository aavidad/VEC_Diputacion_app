\set ON_ERROR_STOP on
-- Sólo clon sintético: requiere organizacion, ambito y hasta del fixture
-- privado aprobado. No crea una unidad inicial ni una fuente favorable ficticia.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='10s';
SELECT set_config('vec_test_personal31.org',:'organizacion',true) AS org_test,
 set_config('vec_test_personal31.ambito',:'ambito',true) AS ambito_test,
 set_config('vec_test_personal31.hasta',:'hasta',true) AS hasta_test \gset
SET LOCAL ROLE vec_autorizacion_propietario;
DO $fuente$
DECLARE org text:=current_setting('vec_test_personal31.org');a jsonb:=current_setting('vec_test_personal31.ambito')::jsonb;
 hasta timestamptz:=current_setting('vec_test_personal31.hasta')::timestamptz;r jsonb;otra jsonb;version numeric;
BEGIN
 r:=vec_personal.cotejar_unidad_bootstrap_admin_v1(org,a,hasta);
 IF r->>'organizacion_ref' IS DISTINCT FROM org OR r->>'unidad_ref' IS DISTINCT FROM a#>>'{valores,0}'
 OR r->'fuente' IS DISTINCT FROM a->'fuente'
 THEN RAISE EXCEPTION 'fuente gobernada no coincide con tuple'; END IF;
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1('org_inexistente_personal31',a,hasta);
  RAISE EXCEPTION 'organismo ajeno admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 otra:=jsonb_set(a,'{fuente,huella_sha256}',to_jsonb(CASE WHEN a#>>'{fuente,huella_sha256}'=repeat('1',64) THEN repeat('2',64) ELSE repeat('1',64) END));
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1(org,otra,hasta);
  RAISE EXCEPTION 'SHA divergente admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 version:=(a#>>'{fuente,version}')::numeric;
 otra:=jsonb_set(a,'{fuente,version}',to_jsonb(CASE WHEN version=1 THEN 2::numeric ELSE version-1 END));
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1(org,otra,hasta);
  RAISE EXCEPTION 'versión divergente admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $fuente$;
RESET ROLE;
SET LOCAL ROLE vec_personal_propietario;
-- Sólo se añade retirada negativa del nodo gobernado ya cotejado. La historia
-- original y su fuente permanecen intactas; la nueva revisión se revierte al final.
INSERT INTO vec_personal.org_nodo_historia
SELECT (jsonb_populate_record(NULL::vec_personal.org_nodo_historia,to_jsonb(h)||jsonb_build_object(
 'revision',h.revision+1,'catalogo_revision',h.catalogo_revision+1,
 'conocido_desde',clock_timestamp(),'retirado',true))).*
FROM vec_personal.org_nodo_historia h
WHERE h.organismo_ref=current_setting('vec_test_personal31.org')
 AND h.unidad_ref=(current_setting('vec_test_personal31.ambito')::jsonb)#>>'{valores,0}'
 AND h.revision=((current_setting('vec_test_personal31.ambito')::jsonb)#>>'{fuente,version}')::integer
 AND h.fuente_ref=(current_setting('vec_test_personal31.ambito')::jsonb)#>>'{fuente,referencia}'
 AND h.huella_fuente_sha256=(current_setting('vec_test_personal31.ambito')::jsonb)#>>'{fuente,huella_sha256}';
RESET ROLE;
SET LOCAL ROLE vec_autorizacion_propietario;
DO $retirada$
BEGIN
 BEGIN
  PERFORM vec_personal.cotejar_unidad_bootstrap_admin_v1(current_setting('vec_test_personal31.org'),current_setting('vec_test_personal31.ambito')::jsonb,current_setting('vec_test_personal31.hasta')::timestamptz);
  RAISE EXCEPTION 'retirada recuperó revisión antigua';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $retirada$;
RESET ROLE;
ROLLBACK;
