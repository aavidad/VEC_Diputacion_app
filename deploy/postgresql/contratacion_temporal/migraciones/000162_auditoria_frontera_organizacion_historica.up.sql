-- CT162: amplía solo la pareja nominal de auditoría para Organización histórica.
-- GET lo controla la frontera HTTP: el contrato instalado tiene cinco argumentos.
-- Conserva CT108, sus filas, ACL, disparadores y las rutas CT ya admitidas.
-- Corrige el cuantificador ARE heredado del actor sin ampliar longitud ni alfabeto.
-- No instala permisos, lectores, personas ni una audiencia nueva V3.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000162',0));
LOCK TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta IN ACCESS EXCLUSIVE MODE;
DO $cambio$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)');
 meta jsonb; tabla jsonb; deps jsonb; controles jsonb; filas text; fuente text;
BEGIN
 SELECT to_jsonb(p)-'prosrc'-'proconfig',p.prosrc INTO STRICT meta,fuente FROM pg_proc p WHERE p.oid=f;
 IF current_user<>'vec_contratacion_temporal_propietario'
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM 'a0d433448159b6d145a8931294f37fcaa9f63811df5469308077beb1203c72ec'
    OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
        AND p.proowner='vec_contratacion_temporal_propietario'::regrole
        AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u'
        AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s'])
    OR (SELECT count(*) FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_contratacion_temporal_registrador_frontera'::regrole)
         OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'CT162: preimagen de funcion o ACL incompatible' USING ERRCODE='55000';
 END IF;
 SELECT to_jsonb(c)-'relchecks' INTO STRICT tabla FROM pg_class c
 WHERE c.oid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass;
 IF NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
     AND c.relowner='vec_contratacion_temporal_propietario'::regrole AND c.relrowsecurity AND c.relforcerowsecurity) THEN
  RAISE EXCEPTION 'CT162: tabla de auditoria incompatible' USING ERRCODE='55000';
 END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) INTO deps
 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(c) ORDER BY c.conname),'[]'::jsonb) INTO controles
 FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
 AND c.conname NOT IN('auditoria_frontera_ruta_exacta_superficie_check','auditoria_frontera_ruta_exacta_ruta_check','auditoria_frontera_ruta_exacta_superficie_ruta_check','auditoria_frontera_ruta_exacta_actor_ref_check');
 SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.evento_id)::text,'[]'),'UTF8')),'hex') INTO filas
 FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta t;

 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
     AND c.conname='auditoria_frontera_ruta_exacta_superficie_check' AND c.contype='c' AND c.convalidated
     AND pg_get_constraintdef(c.oid)='CHECK ((superficie = ''api.contratacion_temporal.ruta_exacta''::text))')
    OR NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
     AND c.conname='auditoria_frontera_ruta_exacta_ruta_check' AND c.contype='c' AND c.convalidated
     AND pg_get_constraintdef(c.oid)='CHECK ((ruta ~ ''^/api/vec/contratacion-temporal/[a-z0-9_-]+(/([a-z0-9_-]+))*$''::text))')
    OR EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
     AND c.conname='auditoria_frontera_ruta_exacta_superficie_ruta_check') THEN
  RAISE EXCEPTION 'CT162: preimagen de superficies incompatible' USING ERRCODE='55000';
 END IF;
 -- PostgreSQL ARE no admite {1,512}; se conserva su intención con longitud.
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
     AND c.conname='auditoria_frontera_ruta_exacta_actor_ref_check' AND c.contype='c' AND c.convalidated
     AND pg_get_constraintdef(c.oid)='CHECK (((actor_ref IS NULL) OR (actor_ref ~ ''^[A-Za-z0-9:_-]{1,512}$''::text)))') THEN
  RAISE EXCEPTION 'CT162: preimagen de actor incompatible' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  DROP CONSTRAINT auditoria_frontera_ruta_exacta_actor_ref_check;
 ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  ADD CONSTRAINT auditoria_frontera_ruta_exacta_actor_ref_check CHECK(
   actor_ref IS NULL OR (length(actor_ref) BETWEEN 1 AND 512 AND actor_ref ~ '^[A-Za-z0-9:_-]+$'));
 ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  DROP CONSTRAINT auditoria_frontera_ruta_exacta_superficie_check,
  DROP CONSTRAINT auditoria_frontera_ruta_exacta_ruta_check;
 ALTER TABLE vec_contratacion_temporal.auditoria_frontera_ruta_exacta
  ADD CONSTRAINT auditoria_frontera_ruta_exacta_superficie_ruta_check CHECK(
   (superficie='api.contratacion_temporal.ruta_exacta'
    AND ruta ~ '^/api/vec/contratacion-temporal/[a-z0-9_-]+(/([a-z0-9_-]+))*$')
   OR (superficie='organizacion_historica_personal'
    AND ruta='/api/vec/personal/organizacion-historica'));
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(
    p_correlacion_ref text,
    p_motivo text,
    p_superficie text,
    p_ruta text,
    p_actor_ref text
)
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog,pg_temp
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '2s'
AS $funcion$
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
$funcion$;
 REVOKE ALL ON FUNCTION vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text) FROM PUBLIC;
 IF (SELECT to_jsonb(p)-'prosrc'-'proconfig' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=1s','statement_timeout=2s']
    OR (SELECT to_jsonb(c)-'relchecks' FROM pg_class c WHERE c.oid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass) IS DISTINCT FROM tabla
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(c) ORDER BY c.conname),'[]'::jsonb) FROM pg_constraint c
       WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass
       AND c.conname NOT IN('auditoria_frontera_ruta_exacta_superficie_check','auditoria_frontera_ruta_exacta_ruta_check','auditoria_frontera_ruta_exacta_superficie_ruta_check','auditoria_frontera_ruta_exacta_actor_ref_check')) IS DISTINCT FROM controles
    OR (SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.evento_id)::text,'[]'),'UTF8')),'hex') FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta t) IS DISTINCT FROM filas THEN
  RAISE EXCEPTION 'CT162: cambio fuera del contrato' USING ERRCODE='55000';
 END IF;
END $cambio$;
COMMIT;
