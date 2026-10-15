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

DO $pre$
DECLARE f oid := 'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure;
 tabla oid := 'vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass;
 superficie text; motivo text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT superficie FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'
  AND c.contype='c' AND c.convalidated;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT motivo FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname='auditoria_frontera_ruta_exacta_motivo_check'
  AND c.contype='c' AND c.convalidated;
 IF current_user<>'vec_contratacion_temporal_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '442a0e0c97d3638c1ff4331cc758295368096042013ea2161d955ef5e23374d9'
 OR encode(sha256(convert_to(superficie,'UTF8')),'hex') IS DISTINCT FROM
  '86b4867344ac7d62d1ad239587fc70091029349abe6515f06729b3708e856803'
 OR encode(sha256(convert_to(motivo,'UTF8')),'hex') IS DISTINCT FROM
  'cae40bfa0378f3c4baf62c55d367fc6a51c1d53c16b12ebe66e138600021da2e'
 OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
 OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM
  ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s']
 OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM
  '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_registrador_frontera=X/vec_contratacion_temporal_propietario}'
 OR (SELECT relowner FROM pg_class WHERE oid=tabla) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
 OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=tabla AND relrowsecurity AND relforcerowsecurity)
 THEN RAISE EXCEPTION 'CT222: preimagen HZ37 o ACL divergente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(p_correlacion_ref text, p_motivo text, p_superficie text, p_ruta text, p_actor_ref text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '1s'
 SET statement_timeout TO '2s'
AS $function$
DECLARE
    v_segmentos text[];
BEGIN
    IF p_correlacion_ref IS NULL
       OR (p_correlacion_ref <> 'corr_no_disponible' AND p_correlacion_ref !~ '^corr_[0-9a-f]{32}$')
       OR p_motivo IS NULL OR (p_motivo NOT IN ('autenticacion_requerida', 'acceso_denegado') AND NOT (p_superficie = 'api.bolsa.rrhh.documentos.ruta_exacta' AND p_motivo = 'servicio_no_disponible'))
       OR p_superficie IS NULL OR p_ruta IS NULL
       OR pg_catalog.length(p_ruta) > (CASE WHEN p_superficie = 'api.bolsa.rrhh.documentos.ruta_exacta' THEN 768 ELSE 512 END)
       OR p_actor_ref IS NOT NULL AND (
            pg_catalog.length(p_actor_ref) > 512
            OR p_actor_ref <> pg_catalog.btrim(p_actor_ref)
            OR p_actor_ref !~ '^[A-Za-z0-9:_-]+$'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'auditoria de frontera invalida';
    END IF;
    IF p_superficie = 'api.contratacion_temporal.ruta_exacta' THEN
        IF pg_catalog.length(p_ruta) <= pg_catalog.length('/api/vec/contratacion-temporal/')
           OR p_ruta !~ '^/api/vec/contratacion-temporal/[a-z0-9_-]+(/([a-z0-9_-]+))*$' THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'auditoria de frontera invalida';
        END IF;
        v_segmentos := pg_catalog.string_to_array(
            pg_catalog.substr(p_ruta, pg_catalog.length('/api/vec/') + 1), '/'
        );
        IF v_segmentos IS NULL OR v_segmentos[1] <> 'contratacion-temporal'
           OR EXISTS (SELECT 1 FROM pg_catalog.unnest(v_segmentos) AS segmento
              WHERE pg_catalog.length(segmento) NOT BETWEEN 1 AND 64
                 OR segmento !~ '^[a-z0-9_-]+$') THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'auditoria de frontera invalida';
        END IF;
    ELSIF p_superficie = 'api.seleccion.preparacion_bases.ruta_exacta'
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
    ELSIF p_superficie = 'api.bolsa.rrhh.documentos.ruta_exacta'
          AND p_ruta = '/api/vec/bolsa/justificantes-renuncia/rrhh'
          AND p_actor_ref IS NULL THEN
        NULL;
    ELSIF p_superficie = 'api.bolsa.rrhh.documentos.ruta_exacta' THEN
        IF pg_catalog.strpos(p_ruta,'..') <> 0
           OR (p_ruta ~ '/recibo$' AND pg_catalog.length(pg_catalog.split_part(p_ruta,'/',8)) > 512)
           OR (p_motivo = 'autenticacion_requerida' AND p_actor_ref IS NOT NULL)
           OR (p_ruta !~ '^/api/vec/bolsa/bolsas/bolsa:[a-z0-9:._-]{1,194}/documentos$'
               AND p_ruta !~ '^/api/vec/bolsa/bolsas/bolsa:[a-z0-9:._-]{1,194}/documentos/[A-Za-z0-9][A-Za-z0-9:._~-]*/recibo$') THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'auditoria de frontera invalida';
        END IF;
    ELSIF p_superficie = 'api.personal.registro_empleado.ruta_exacta'
          AND p_ruta IN ('/api/vec/personal/catalogos-registro-empleado',
                         '/api/vec/personal/empleados/{emp_ref}',
                         '/api/vec/personal/empleados-organismo',
                         '/api/vec/personal/vacantes') THEN
        NULL;
    ELSIF p_superficie = 'organizacion_historica_personal'
          AND p_ruta = '/api/vec/personal/organizacion-historica' THEN
        NULL;
    ELSE
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'auditoria de frontera invalida';
    END IF;
    INSERT INTO vec_contratacion_temporal.auditoria_frontera_ruta_exacta (
        correlacion_ref, motivo, superficie, ruta, actor_ref, registrada_en
    ) VALUES (
        p_correlacion_ref, p_motivo, p_superficie, p_ruta, p_actor_ref,
        pg_catalog.date_trunc('microseconds', pg_catalog.clock_timestamp())
    );
    RETURN true;
END
$function$;

DO $ruta$
DECLARE tabla oid := 'vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass;
 vieja text; nueva text; post text;
 marca text := $m$OR superficie = 'api.bolsa.rrhh.documentos.ruta_exacta'::text AND length(ruta) <= 768$m$;
 rama text := $m$OR superficie = 'api.bolsa.rrhh.documentos.ruta_exacta'::text AND ruta = '/api/vec/bolsa/justificantes-renuncia/rrhh'::text AND actor_ref IS NULL $m$;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT vieja FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'
  AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(vieja,'UTF8')),'hex') IS DISTINCT FROM
  '86b4867344ac7d62d1ad239587fc70091029349abe6515f06729b3708e856803'
 OR length(vieja)-length(replace(vieja,marca,''))<>length(marca)
 THEN RAISE EXCEPTION 'CT222: CHECK de rutas anterior divergente' USING ERRCODE='55000'; END IF;
 nueva:=replace(vieja,marca,rama||marca);
 ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  DROP CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check;
 EXECUTE 'ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta '
  ||'ADD CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check '||nueva;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT post FROM pg_constraint c
 WHERE c.conrelid=tabla AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'
  AND c.contype='c' AND c.convalidated;
 IF post IS DISTINCT FROM nueva OR
  encode(sha256(convert_to(post,'UTF8')),'hex') IS DISTINCT FROM
   '9583301a475458eb6f2346a9da03cb8486797fe90c3dc6660efae1a9325f658f'
 THEN RAISE EXCEPTION 'CT222: CHECK de rutas posterior divergente' USING ERRCODE='55000'; END IF;
END $ruta$;

DO $post$
DECLARE f oid := 'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure;
 tabla oid := 'vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass;
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM
  '6457d96e414374a9936bd0f0e399022e511db1b2b8ae10e78214e5c6cb18c7ef'
 OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
 OR (SELECT proacl::text FROM pg_proc WHERE oid=f) IS DISTINCT FROM
  '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_registrador_frontera=X/vec_contratacion_temporal_propietario}'
 OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex') FROM pg_constraint c
   WHERE c.conrelid=tabla AND c.conname='auditoria_frontera_ruta_exacta_motivo_check') IS DISTINCT FROM
  'cae40bfa0378f3c4baf62c55d367fc6a51c1d53c16b12ebe66e138600021da2e'
 THEN RAISE EXCEPTION 'CT222: función, motivos o ACL posteriores divergentes' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
