\set ON_ERROR_STOP on
-- CT222: una ruta exacta de denegación RRHH usa la bitácora CT210 existente.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
 'vec_contratacion_temporal:migracion:000222',0));
LOCK TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta IN ACCESS EXCLUSIVE MODE;

DO $evolucion$
DECLARE f pg_catalog.regprocedure :=
 'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::pg_catalog.regprocedure;
 tabla oid := 'vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::pg_catalog.regclass;
 cuerpo text; nuevo_cuerpo text; superficie text; nueva_superficie text; motivo text;
 marca_funcion text := $m$    ELSIF p_superficie = 'api.bolsa.rrhh.documentos.ruta_exacta' THEN$m$;
 marca_check text := $m$OR superficie = 'api.bolsa.rrhh.documentos.ruta_exacta'::text AND length(ruta) <= 768$m$;
 rama_check text := $m$OR superficie = 'api.bolsa.rrhh.documentos.ruta_exacta'::text AND ruta = '/api/vec/bolsa/justificantes-renuncia/rrhh'::text AND actor_ref IS NULL $m$;
 hash_funcion text; hash_superficie text; hash_motivo text;
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT cuerpo;
 SELECT pg_catalog.pg_get_constraintdef(c.oid,true) INTO STRICT superficie
 FROM pg_catalog.pg_constraint c WHERE c.conrelid=tabla
  AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'
  AND c.contype='c' AND c.convalidated;
 SELECT pg_catalog.pg_get_constraintdef(c.oid,true) INTO STRICT motivo
 FROM pg_catalog.pg_constraint c WHERE c.conrelid=tabla
  AND c.conname='auditoria_frontera_ruta_exacta_motivo_check'
  AND c.contype='c' AND c.convalidated;
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     'vec_contratacion_temporal_propietario'::pg_catalog.regrole
    OR (SELECT p.proconfig FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s']
    OR (SELECT p.proacl::text FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_registrador_frontera=X/vec_contratacion_temporal_propietario}'
    OR (SELECT c.relowner FROM pg_catalog.pg_class c WHERE c.oid=tabla) IS DISTINCT FROM
     'vec_contratacion_temporal_propietario'::pg_catalog.regrole
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c WHERE c.oid=tabla AND c.relrowsecurity AND c.relforcerowsecurity)
 THEN RAISE EXCEPTION 'CT222: dueño, ACL o RLS incompatibles' USING ERRCODE='55000'; END IF;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(cuerpo,'UTF8')),'hex') IS DISTINCT FROM
     '442a0e0c97d3638c1ff4331cc758295368096042013ea2161d955ef5e23374d9'
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(superficie,'UTF8')),'hex') IS DISTINCT FROM
     '86b4867344ac7d62d1ad239587fc70091029349abe6515f06729b3708e856803'
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(motivo,'UTF8')),'hex') IS DISTINCT FROM
     'cae40bfa0378f3c4baf62c55d367fc6a51c1d53c16b12ebe66e138600021da2e'
    OR pg_catalog.length(cuerpo)-pg_catalog.length(pg_catalog.replace(cuerpo,marca_funcion,''))<>pg_catalog.length(marca_funcion)
    OR pg_catalog.length(superficie)-pg_catalog.length(pg_catalog.replace(superficie,marca_check,''))<>pg_catalog.length(marca_check)
 THEN RAISE EXCEPTION 'CT222: preimagen HZ37 distinta' USING ERRCODE='55000'; END IF;

 nuevo_cuerpo:=pg_catalog.replace(cuerpo,marca_funcion,
  $m$    ELSIF p_superficie = 'api.bolsa.rrhh.documentos.ruta_exacta'
          AND p_ruta = '/api/vec/bolsa/justificantes-renuncia/rrhh'
          AND p_actor_ref IS NULL THEN
        NULL;
$m$||marca_funcion);
 nueva_superficie:=pg_catalog.replace(superficie,marca_check,rama_check||marca_check);
 EXECUTE nuevo_cuerpo;
 ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  DROP CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check;
 EXECUTE 'ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta '
  ||'ADD CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check '||nueva_superficie;

 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.pg_get_functiondef(f),'UTF8')),'hex') INTO hash_funcion;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.pg_get_constraintdef(c.oid,true),'UTF8')),'hex') INTO hash_superficie
 FROM pg_catalog.pg_constraint c WHERE c.conrelid=tabla
  AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check' AND c.contype='c' AND c.convalidated;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  pg_catalog.pg_get_constraintdef(c.oid,true),'UTF8')),'hex') INTO hash_motivo
 FROM pg_catalog.pg_constraint c WHERE c.conrelid=tabla
  AND c.conname='auditoria_frontera_ruta_exacta_motivo_check' AND c.contype='c' AND c.convalidated;
 IF hash_funcion IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(nuevo_cuerpo,'UTF8')),'hex')
    OR hash_superficie IS NULL OR hash_superficie=
      '86b4867344ac7d62d1ad239587fc70091029349abe6515f06729b3708e856803'
    OR hash_motivo IS DISTINCT FROM 'cae40bfa0378f3c4baf62c55d367fc6a51c1d53c16b12ebe66e138600021da2e'
    OR (SELECT p.proacl::text FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM
     '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_registrador_frontera=X/vec_contratacion_temporal_propietario}'
    OR pg_catalog.strpos(pg_catalog.pg_get_constraintdef((SELECT c.oid FROM pg_catalog.pg_constraint c
       WHERE c.conrelid=tabla AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'),true),
      '/api/vec/bolsa/justificantes-renuncia/rrhh')=0
 THEN RAISE EXCEPTION 'CT222: postimagen o ACL divergente' USING ERRCODE='55000'; END IF;
 RAISE NOTICE 'CT222: post SHA funcion %, ruta %, motivo %',hash_funcion,hash_superficie,hash_motivo;
END $evolucion$;
COMMIT;
