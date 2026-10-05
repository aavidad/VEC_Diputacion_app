\set ON_ERROR_STOP on
-- Sólo clon desechable, DBA. Fuente y plan canónicos se cargan por parámetros
-- enlazados en GUC de sesión; este archivo no configura permisos ni aprobaciones.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='20s';
DO $pruebas$
DECLARE texto text:=current_setting('vec.ensayo.plan_unidad_inicial');p jsonb:=texto::jsonb;
 fuente text:=current_setting('vec.ensayo.fuente_unidad_inicial');f jsonb:=fuente::jsonb;
 pre jsonb;repetido jsonb;antes bigint;gen bigint;t text;funcion regprocedure;
BEGIN
 IF texto IS DISTINCT FROM vec_personal.canon_unidad_inicial_admin_v1(p,'plan')
 OR fuente IS DISTINCT FROM vec_personal.canon_unidad_inicial_admin_v1(f,'fuente')
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM p#>>'{fuente,huella_sha256}'
 OR p->'unidad' IS DISTINCT FROM f->'unidad'
 THEN RAISE EXCEPTION 'Personal33 pruebas: canon/fuente discrepante'; END IF;
 SELECT count(*) INTO antes FROM vec_personal.org_nodo_historia;
 SELECT generacion INTO STRICT gen FROM vec_personal.control_unidad_bootstrap_admin_v1 WHERE singleton;
 pre:=vec_personal.preimagen_unidad_inicial_admin_v1(p);
 repetido:=vec_personal.preimagen_unidad_inicial_admin_v1(p);
 IF pre IS DISTINCT FROM repetido OR (pre->>'generacion')::bigint<>gen
 OR (SELECT count(*) FROM vec_personal.org_nodo_historia)<>antes
 THEN RAISE EXCEPTION 'Personal33 pruebas: preparar preimagen creó efectos'; END IF;
 FOREACH t IN ARRAY ARRAY['config_unidad_inicial_admin_v1','unidad_inicial_admin_v1'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_personal.'||t) AND relowner='vec_personal_propietario'::regrole AND relrowsecurity AND relforcerowsecurity)
  OR EXISTS(SELECT 1 FROM pg_class c,LATERAL aclexplode(COALESCE(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=to_regclass('vec_personal.'||t) AND a.grantee<>c.relowner)
  THEN RAISE EXCEPTION 'Personal33 pruebas: tabla privada o RLS divergente'; END IF;
 END LOOP;
 funcion:='vec_personal.inicializar_unidad_sintetica_admin_v1(text,text,text)'::regprocedure;
 IF NOT has_function_privilege('vec_personal_unidad_inicial_ejecutor',funcion,'EXECUTE')
 OR has_table_privilege('vec_personal_unidad_inicial_ejecutor','vec_personal.org_nodo_historia','INSERT,UPDATE,DELETE')
 OR has_function_privilege('vec_personal_unidad_inicial_ejecutor','vec_personal.aplicar_efecto_unidad_inicial_admin_v1(text,text,text)','EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc pr,LATERAL aclexplode(COALESCE(pr.proacl,acldefault('f',pr.proowner))) a WHERE pr.oid=funcion AND a.grantee=0)
 THEN RAISE EXCEPTION 'Personal33 pruebas: frontera de ejecución ampliada'; END IF;
END $pruebas$;
ROLLBACK;
