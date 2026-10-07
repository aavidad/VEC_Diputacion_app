\set ON_ERROR_STOP on
-- CT168: auditoría previa al contexto para Preparación de bases y Baremo.
-- E05/E06/E10: sólo cinco rutas nominales, sin actor, datos ni permisos nuevos.
-- Fuente real post-CT162 PG18.4; conserva CT108, OH, ACL e historia.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000168',0));
LOCK TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta IN ACCESS EXCLUSIVE MODE;
DO $cambio$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)');
 t oid:='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass;
 original text; fuente text; nuevo text; actual text; pareja text;
 meta jsonb; tabla jsonb; deps jsonb; compartidas jsonb; controles jsonb;
 disparadores jsonb; politicas jsonb; filas text;
 marca text:=$marca$    ELSIF p_superficie = 'organizacion_historica_personal'$marca$;
 extension text:=$extension$    ELSIF p_superficie = 'api.seleccion.preparacion_bases.ruta_exacta'
          AND p_ruta IN ('/api/vec/seleccion/preparacion-bases/guardar',
                         '/api/vec/seleccion/preparacion-bases/consultar')
          AND p_motivo = 'acceso_denegado' AND p_actor_ref IS NULL THEN
        NULL;
    ELSIF p_superficie = 'api.bolsa.reglas_baremo.ruta_exacta'
          AND p_ruta IN ('/api/vec/bolsa/reglas-baremo/borradores/alta',
                         '/api/vec/bolsa/reglas-baremo/versiones/consultar',
                         '/api/vec/bolsa/reglas-baremo/recibos/recuperar')
          AND p_actor_ref IS NULL THEN
        NULL;
$extension$;
BEGIN
 IF f IS NULL OR current_user<>'vec_contratacion_temporal_propietario' THEN
  RAISE EXCEPTION 'CT168: autoridad ausente' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc'
 INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '5ead94fccb71fcb5da5075c37a1d6aa694fa550ff6ce9bdedfdd97d9cc7be488'
 OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM '8cc0cce2a2b9a39957b94894543a081100e8d65c4225cf0d9d4a8724520b592f'
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND p.prosecdef AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_registrador_frontera'::regrole)
      OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR length(original)-length(replace(original,marca,''))<>length(marca)
 OR strpos(original,'api.seleccion.preparacion_bases.ruta_exacta')<>0
 OR strpos(original,'api.bolsa.reglas_baremo.ruta_exacta')<>0 THEN
  RAISE EXCEPTION 'CT168: preimagen de función incompatible' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT pareja FROM pg_constraint c
 WHERE c.conrelid=t AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'
 AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(pareja,'UTF8')),'hex') IS DISTINCT FROM '9adb283a8c6d4407ac68ee66a3530a7388fa59d26476bf5429b40490360f9af1'
 OR left(pareja,7)<>'CHECK (' OR right(pareja,1)<>')'
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=t
   AND c.relowner='vec_contratacion_temporal_propietario'::regrole
   AND c.relrowsecurity AND c.relforcerowsecurity) THEN
  RAISE EXCEPTION 'CT168: preimagen de tabla incompatible' USING ERRCODE='55000';
 END IF;
 SELECT to_jsonb(c)-'relchecks' INTO STRICT tabla FROM pg_class c WHERE c.oid=t;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
 AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 SELECT coalesce(jsonb_agg(to_jsonb(c) ORDER BY c.conname),'[]'::jsonb) INTO controles
 FROM pg_constraint c WHERE c.conrelid=t AND c.conname<>'auditoria_frontera_ruta_exacta_superficie_ruta_check';
 SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.oid),'[]'::jsonb) INTO disparadores FROM pg_trigger x WHERE x.tgrelid=t;
 SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.oid),'[]'::jsonb) INTO politicas FROM pg_policy x WHERE x.polrelid=t;
 SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.evento_id)::text,'[]'),'UTF8')),'hex')
 INTO filas FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta x;

 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,extension||marca,marca) IS DISTINCT FROM original THEN
  RAISE EXCEPTION 'CT168: función alterada fuera del delta nominal' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  DROP CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check;
 EXECUTE 'ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta '
  ||'ADD CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check '
  ||left(pareja,length(pareja)-1)
  ||' OR (superficie=''api.seleccion.preparacion_bases.ruta_exacta'' '
  ||'AND ruta IN (''/api/vec/seleccion/preparacion-bases/guardar'',''/api/vec/seleccion/preparacion-bases/consultar'') '
  ||'AND motivo=''acceso_denegado'' AND actor_ref IS NULL) '
  ||'OR (superficie=''api.bolsa.reglas_baremo.ruta_exacta'' '
  ||'AND ruta IN (''/api/vec/bolsa/reglas-baremo/borradores/alta'',''/api/vec/bolsa/reglas-baremo/versiones/consultar'',''/api/vec/bolsa/reglas-baremo/recibos/recuperar'') '
  ||'AND actor_ref IS NULL))';

 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT to_jsonb(c)-'relchecks' FROM pg_class c WHERE c.oid=t) IS DISTINCT FROM tabla
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
     FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
     FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f
     AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 OR (SELECT coalesce(jsonb_agg(to_jsonb(c) ORDER BY c.conname),'[]'::jsonb)
     FROM pg_constraint c WHERE c.conrelid=t AND c.conname<>'auditoria_frontera_ruta_exacta_superficie_ruta_check') IS DISTINCT FROM controles
 OR (SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.oid),'[]'::jsonb) FROM pg_trigger x WHERE x.tgrelid=t) IS DISTINCT FROM disparadores
 OR (SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.oid),'[]'::jsonb) FROM pg_policy x WHERE x.polrelid=t) IS DISTINCT FROM politicas
 OR (SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.evento_id)::text,'[]'),'UTF8')),'hex')
     FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta x) IS DISTINCT FROM filas THEN
  RAISE EXCEPTION 'CT168: historia o metadatos alterados' USING ERRCODE='55000';
 END IF;
END $cambio$;
COMMIT;
