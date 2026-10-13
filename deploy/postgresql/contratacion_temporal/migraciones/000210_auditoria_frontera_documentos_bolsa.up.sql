BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_contratacion_temporal:migracion:000210',0));
LOCK TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta IN ACCESS EXCLUSIVE MODE;

-- El registrador es compartido por vec/ports; el espacio CT es histórico.
-- Se preservan cuerpo, firma, dueño, ACL y todas las superficies instaladas.
DO $pre$
DECLARE f oid:=pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)');
BEGIN
 IF f IS NULL OR
    (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
       pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')
       FROM pg_catalog.pg_proc p WHERE p.oid=f
         AND p.proowner='vec_contratacion_temporal_propietario'::regrole
         AND p.prosecdef AND p.provolatile='v' AND p.prokind='f'
         AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on',
           'TimeZone=UTC','lock_timeout=1s','statement_timeout=2s'])
       IS DISTINCT FROM '3f64035ddcbd26b8f73bdc9503bd16a7c809169d866109dc3bb61468a716ac0d'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.pg_get_constraintdef(c.oid,true),'UTF8')),'hex')
        FROM pg_catalog.pg_constraint c
        WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
          AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check'
          AND c.contype='c' AND c.convalidated)
       IS DISTINCT FROM '053c7aa00e0f1af30a19ce390e9369d441a77186c592bbf4b2785fbf7f8bb58a'
    OR (SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.pg_get_constraintdef(c.oid,true),'UTF8')),'hex')
        FROM pg_catalog.pg_constraint c
        WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
          AND c.conname='auditoria_frontera_ruta_exacta_motivo_check'
          AND c.contype='c' AND c.convalidated)
       IS DISTINCT FROM '375a1424c7539590e97e285b9edc969fee5c2b3944f9b57f7a41be1e1fa895b0'
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='preimagen CT210 distinta'; END IF;
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
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
 DROP CONSTRAINT auditoria_frontera_ruta_exacta_motivo_check;
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
 ADD CONSTRAINT auditoria_frontera_ruta_exacta_motivo_check
 CHECK (motivo IN ('autenticacion_requerida','acceso_denegado')
        OR (superficie='api.bolsa.rrhh.documentos.ruta_exacta'
            AND motivo='servicio_no_disponible'));
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
 DROP CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check;
ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
 ADD CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check
 CHECK (((((superficie = 'api.contratacion_temporal.ruta_exacta'::text) AND (ruta ~ '^/api/vec/contratacion-temporal/[a-z0-9_-]+(/([a-z0-9_-]+))*$'::text)) OR ((superficie = 'organizacion_historica_personal'::text) AND (ruta = '/api/vec/personal/organizacion-historica'::text)) OR ((superficie = 'api.seleccion.preparacion_bases.ruta_exacta'::text) AND (ruta = ANY (ARRAY['/api/vec/seleccion/preparacion-bases/guardar'::text, '/api/vec/seleccion/preparacion-bases/consultar'::text])) AND (motivo = 'acceso_denegado'::text) AND (actor_ref IS NULL)) OR ((superficie = 'api.bolsa.reglas_baremo.ruta_exacta'::text) AND (ruta = ANY (ARRAY['/api/vec/bolsa/reglas-baremo/borradores/alta'::text, '/api/vec/bolsa/reglas-baremo/versiones/consultar'::text, '/api/vec/bolsa/reglas-baremo/recibos/recuperar'::text])) AND (actor_ref IS NULL)) OR ((superficie = 'api.personal.registro_empleado.ruta_exacta'::text) AND (ruta = ANY (ARRAY['/api/vec/personal/catalogos-registro-empleado'::text, '/api/vec/personal/empleados/{emp_ref}'::text, '/api/vec/personal/empleados-organismo'::text, '/api/vec/personal/vacantes'::text]))))) OR (superficie='api.bolsa.rrhh.documentos.ruta_exacta'
     AND pg_catalog.length(ruta) <= 768
     AND pg_catalog.strpos(ruta,'..')=0
     AND (ruta !~ '/recibo$' OR pg_catalog.length(pg_catalog.split_part(ruta,'/',8))<=512)
     AND (motivo<>'autenticacion_requerida' OR actor_ref IS NULL)
     AND (ruta ~ '^/api/vec/bolsa/bolsas/bolsa:[a-z0-9:._-]{1,194}/documentos$'
          OR ruta ~ '^/api/vec/bolsa/bolsas/bolsa:[a-z0-9:._-]{1,194}/documentos/[A-Za-z0-9][A-Za-z0-9:._~-]*/recibo$')));

DO $post$
DECLARE f oid:=pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)');
BEGIN
 IF pg_catalog.pg_get_userbyid((SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f))
       <> 'vec_contratacion_temporal_propietario'
    OR NOT pg_catalog.has_function_privilege(
      'vec_contratacion_temporal_registrador_frontera',f,'EXECUTE')
    OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p,
          LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=f)<>2
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
          LATERAL pg_catalog.aclexplode(p.proacl) a
          WHERE p.oid=f AND a.grantee NOT IN
            ('vec_contratacion_temporal_propietario'::regrole,
             'vec_contratacion_temporal_registrador_frontera'::regrole))
 THEN RAISE EXCEPTION USING ERRCODE='55000',MESSAGE='ACL CT210 inesperada'; END IF;
END $post$;
COMMIT;
