\set ON_ERROR_STOP on
-- CT203: auditoría nominal de la ruta nativa del catálogo Personal B2.
-- Fuente literal: función instalada en clon HZ21 tras CT168, SHA256 b3b26c06066b9027fd18bb7e29bd3918fd8a2dd945f29f267618341050636b69.
-- Solo añade una superficie/ruta exacta; conserva ACL, historia y límites.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000203',0));
LOCK TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta IN ACCESS EXCLUSIVE MODE;
DO $preimagen$
DECLARE
  actual_funcion text;
  actual_restriccion text;
BEGIN
  SELECT encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') INTO actual_funcion
  FROM pg_proc p
  WHERE p.oid='vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure
    AND p.proowner='vec_contratacion_temporal_propietario'::regrole
    AND p.prosecdef AND p.provolatile='v' AND p.prokind='f'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s'];
  SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex') INTO actual_restriccion
  FROM pg_constraint c
  WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
    AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'
    AND c.contype='c' AND c.convalidated;
  IF actual_funcion IS DISTINCT FROM 'b3b26c06066b9027fd18bb7e29bd3918fd8a2dd945f29f267618341050636b69' THEN
    RAISE EXCEPTION 'CT203: preimagen funcion: esperada %, actual %', 'b3b26c06066b9027fd18bb7e29bd3918fd8a2dd945f29f267618341050636b69', coalesce(actual_funcion,'ausente') USING ERRCODE='55000';
  END IF;
  IF actual_restriccion IS DISTINCT FROM '2038d8eb51ce7ed5a8898c9cb1f285f5531b1db764ea3ed1a1831a1fbabbcc6f' THEN
    RAISE EXCEPTION 'CT203: preimagen restriccion: esperada %, actual %', '2038d8eb51ce7ed5a8898c9cb1f285f5531b1db764ea3ed1a1831a1fbabbcc6f', coalesce(actual_restriccion,'ausente') USING ERRCODE='55000';
  END IF;
END $preimagen$;
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
       OR p_motivo IS NULL OR p_motivo NOT IN ('autenticacion_requerida', 'acceso_denegado')
       OR p_superficie IS NULL OR p_ruta IS NULL
       OR pg_catalog.length(p_ruta) > 512
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
    ELSIF p_superficie = 'api.personal.registro_empleado.ruta_exacta'
          AND p_ruta = '/api/vec/personal/catalogos-registro-empleado' THEN
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
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  DROP CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check;
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  ADD CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check
  CHECK (
    (superficie='api.contratacion_temporal.ruta_exacta' AND ruta ~ '^/api/vec/contratacion-temporal/[a-z0-9_-]+(/([a-z0-9_-]+))*$')
    OR (superficie='organizacion_historica_personal' AND ruta='/api/vec/personal/organizacion-historica')
    OR (superficie='api.seleccion.preparacion_bases.ruta_exacta'
        AND ruta IN ('/api/vec/seleccion/preparacion-bases/guardar','/api/vec/seleccion/preparacion-bases/consultar')
        AND motivo='acceso_denegado' AND actor_ref IS NULL)
    OR (superficie='api.bolsa.reglas_baremo.ruta_exacta'
        AND ruta IN ('/api/vec/bolsa/reglas-baremo/borradores/alta','/api/vec/bolsa/reglas-baremo/versiones/consultar','/api/vec/bolsa/reglas-baremo/recibos/recuperar')
        AND actor_ref IS NULL)
    OR (superficie='api.personal.registro_empleado.ruta_exacta'
        AND ruta='/api/vec/personal/catalogos-registro-empleado')
  );
COMMIT;
