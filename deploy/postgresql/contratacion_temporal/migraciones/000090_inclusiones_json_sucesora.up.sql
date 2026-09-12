\set ON_ERROR_STOP on
-- CT90: corrige únicamente la asociación de tres inclusiones JSON de CT87.
-- Avance guardado sobre la función instalada; CT87 e historia permanecen intactos.
-- No tiene DOWN: restaurar deliberadamente el defecto requiere otro corte revisado.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.cierre_administrativo_sin_cese.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000090:inclusiones-sucesora',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $ct90$
DECLARE
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 f oid; p pg_proc%ROWTYPE; antes jsonb; otras jsonb; nuevo text; ddl text; campo text; origen text; destino text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR getdatabaseencoding()<>'UTF8'
  OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=propietario)
  OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CT90: propietario incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_contratacion_temporal.cierre87_validar_sucesora(jsonb,jsonb)');
 IF f IS NULL THEN RAISE EXCEPTION 'CT90: CT87 requerido' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF p.proowner<>propietario OR p.pronamespace<>'vec_contratacion_temporal'::regnamespace
  OR p.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
  OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '90b9112b5e26ceecb667d79d339af7d4d261520cb8e43a9bf3e8ee1144662760'
  OR (to_jsonb(p)-ARRAY['oid','proowner','pronamespace','prolang','prosrc']) IS DISTINCT FROM $meta${"proacl":["vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario"],"proallargtypes":null,"proargdefaults":null,"proargmodes":null,"proargnames":["o","p"],"proargtypes":["3802","3802"],"probin":null,"proconfig":["search_path=pg_catalog"],"procost":100,"proisstrict":false,"prokind":"f","proleakproof":false,"proname":"cierre87_validar_sucesora","pronargdefaults":0,"pronargs":2,"proparallel":"s","proretset":false,"prorettype":"3802","prorows":0,"prosecdef":false,"prosqlbody":null,"prosupport":"-","protrftypes":null,"provariadic":"0","provolatile":"i"}$meta$::jsonb
 THEN RAISE EXCEPTION 'CT90: cuerpo, firma o atributos previos incompatibles' USING ERRCODE='55000'; END IF;
 antes:=to_jsonb(p)-'prosrc';
 SELECT jsonb_agg(to_jsonb(z) ORDER BY z.oid) INTO otras FROM pg_proc z
  WHERE z.pronamespace=p.pronamespace AND z.oid<>f;
 nuevo:=p.prosrc;
 FOREACH campo IN ARRAY ARRAY['estados','transiciones','motivos'] LOOP
  origen:=format('b->%L @> a->%L',campo,campo);
  destino:=format('(b->%L) @> (a->%L)',campo,campo);
  IF (length(nuevo)-length(replace(nuevo,origen,'')))/length(origen)<>1
  THEN RAISE EXCEPTION 'CT90: inclusión previa no unívoca' USING ERRCODE='55000'; END IF;
  nuevo:=replace(nuevo,origen,destino);
 END LOOP;
 IF encode(sha256(convert_to(nuevo,'UTF8')),'hex')<>'d465e183252d25f42850a298873dde245a90541a9816cd7b4e24d67bb4730a8f'
 THEN RAISE EXCEPTION 'CT90: postimagen inesperada' USING ERRCODE='55000'; END IF;
 ddl:=pg_get_functiondef(f);
 IF (length(ddl)-length(replace(ddl,p.prosrc,'')))/length(p.prosrc)<>1
 THEN RAISE EXCEPTION 'CT90: definición no unívoca' USING ERRCODE='55000'; END IF;
 EXECUTE replace(ddl,p.prosrc,nuevo);
 IF (SELECT to_jsonb(z)-'prosrc' FROM pg_proc z WHERE z.oid=f) IS DISTINCT FROM antes
  OR (SELECT prosrc FROM pg_proc WHERE oid=f) IS DISTINCT FROM nuevo
  OR (SELECT jsonb_agg(to_jsonb(z) ORDER BY z.oid) FROM pg_proc z
      WHERE z.pronamespace=p.pronamespace AND z.oid<>f) IS DISTINCT FROM otras
 THEN RAISE EXCEPTION 'CT90: postimagen, OID, atributos u otras funciones alterados' USING ERRCODE='55000'; END IF;
END $ct90$;
COMMIT;
