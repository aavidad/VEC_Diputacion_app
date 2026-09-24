-- Casos SINTÉTICOS de Personal 000016. Ningún dato real; referencias opacas
-- inventadas. Se ejecuta como superusuario de ensayo y publica mediante el
-- propietario de Personal, igual que la carga gobernada.
\set ON_ERROR_STOP on
SET search_path = pg_catalog;
SET timezone = 'UTC';
CREATE TEMP TABLE base_ensayo AS SELECT date_trunc('second', clock_timestamp()) AS t;

CREATE FUNCTION pg_temp.clase(p text) RETURNS text LANGUAGE sql AS $$
  SELECT resultado FROM vec_personal.resolver_empleado_canonico_persona_v1(p, clock_timestamp())
$$;
CREATE FUNCTION pg_temp.empleado(p text) RETURNS text LANGUAGE sql AS $$
  SELECT empleado_ref FROM vec_personal.resolver_empleado_canonico_persona_v1(p, clock_timestamp())
$$;
CREATE FUNCTION pg_temp.exigir(condicion boolean, caso text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  IF condicion IS NOT TRUE THEN RAISE EXCEPTION 'caso fallido: %', caso; END IF;
END $$;
-- Ejecuta una sentencia que debe fallar con el SQLSTATE indicado.
CREATE FUNCTION pg_temp.rechaza(sentencia text, estado text, caso text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
  BEGIN
    EXECUTE sentencia;
  EXCEPTION WHEN OTHERS THEN
    IF SQLSTATE <> estado THEN
      RAISE EXCEPTION 'caso %: SQLSTATE % en lugar de %', caso, SQLSTATE, estado;
    END IF;
    RETURN;
  END;
  RAISE EXCEPTION 'caso %: se aceptó y debía rechazarse', caso;
END $$;
CREATE FUNCTION pg_temp.publicar(
  pep text, v bigint, per text, emp text, estado text, motivo text,
  desde interval DEFAULT interval '-1 day', hasta interval DEFAULT interval '365 days'
) RETURNS timestamptz LANGUAGE plpgsql AS $$
DECLARE r timestamptz; b timestamptz := (SELECT t FROM pg_temp.base_ensayo);
BEGIN
  SET LOCAL ROLE vec_personal_propietario;
  SELECT p.registrada_en INTO r FROM vec_personal.publicar_proyeccion_empleado_persona_v1(
    pep, v, per, emp, estado, b + desde, b + hasta,
    motivo, 'prc_personal_sintetica_00000000000001', 1, repeat('b',64)) p;
  RESET ROLE;
  RETURN r;
END $$;

-- 1. Persona sin empleado: contexto válido sin empleado, no un error.
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_sin_empleado_0000000001') = 'sin_empleado', 'sin empleado');
SELECT pg_temp.exigir(pg_temp.empleado('per_sintetica_sin_empleado_0000000001') IS NULL, 'sin empleado sin emp');

-- 2. Empleado activo.
SELECT pg_temp.publicar('pep_sintetica_activa_000000000000001',1,'per_sintetica_activa_00000000000000001','emp_sintetico_activo_0000000000000001','activa',NULL);
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_activa_00000000000000001') = 'empleado', 'activo');
SELECT pg_temp.exigir(pg_temp.empleado('per_sintetica_activa_00000000000000001') = 'emp_sintetico_activo_0000000000000001', 'activo emp');
SELECT pg_temp.exigir((SELECT version = 1 AND procedencia_ref = 'prc_personal_sintetica_00000000000001' AND efectivas = 1
  FROM vec_personal.resolver_empleado_canonico_persona_v1('per_sintetica_activa_00000000000000001', clock_timestamp())), 'activo procedencia');

-- 3. Vínculo ausente: la proyección de otra persona no se comparte.
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_ausente_0000000000000001') = 'sin_empleado', 'ausente');

-- 4. Ambiguo: dos empleados canónicos efectivos -> ambiguo, nunca elige.
SELECT pg_temp.publicar('pep_sintetica_ambigua_a_000000000001',1,'per_sintetica_ambigua_0000000000000001','emp_sintetico_ambiguo_a_000000000001','activa',NULL);
SELECT pg_temp.publicar('pep_sintetica_ambigua_b_000000000001',1,'per_sintetica_ambigua_0000000000000001','emp_sintetico_ambiguo_b_000000000001','activa',NULL);
SELECT pg_temp.exigir((SELECT resultado = 'ambiguo' AND empleado_ref IS NULL AND proyeccion_ref IS NULL AND efectivas = 2
  FROM vec_personal.resolver_empleado_canonico_persona_v1('per_sintetica_ambigua_0000000000000001', clock_timestamp())), 'ambiguo');
-- Revocar una resuelve la ambigüedad sin reescribir historia.
SELECT pg_temp.publicar('pep_sintetica_ambigua_b_000000000001',2,'per_sintetica_ambigua_0000000000000001','emp_sintetico_ambiguo_b_000000000001','revocada','error_material');
SELECT pg_temp.exigir(pg_temp.empleado('per_sintetica_ambigua_0000000000000001') = 'emp_sintetico_ambiguo_a_000000000001', 'ambiguo resuelto');

-- 5. No activa: deja de ser efectiva; el pasado conocido se conserva.
SELECT pg_temp.publicar('pep_sintetica_noactiva_00000000000001',1,'per_sintetica_noactiva_000000000000001','emp_sintetico_noactivo_000000000000001','activa',NULL);
CREATE TEMP TABLE antes_no_activa AS SELECT clock_timestamp() AS t;
SELECT pg_temp.publicar('pep_sintetica_noactiva_00000000000001',2,'per_sintetica_noactiva_000000000000001','emp_sintetico_noactivo_000000000000001','no_activa','suspension');
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_noactiva_000000000000001') = 'sin_empleado', 'no activa');
SELECT pg_temp.exigir((SELECT resultado = 'empleado' AND version = 1
  FROM vec_personal.resolver_empleado_canonico_persona_v1('per_sintetica_noactiva_000000000000001', (SELECT t FROM antes_no_activa))), 'no activa pasado conocido');
-- Reactivación admisible como versión nueva.
SELECT pg_temp.publicar('pep_sintetica_noactiva_00000000000001',3,'per_sintetica_noactiva_000000000000001','emp_sintetico_noactivo_000000000000001','activa',NULL);
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_noactiva_000000000000001') = 'empleado', 'reactivada');

-- 6. Revocada: terminal.
SELECT pg_temp.publicar('pep_sintetica_revocada_00000000000001',1,'per_sintetica_revocada_000000000000001','emp_sintetico_revocado_000000000000001','activa',NULL);
SELECT pg_temp.publicar('pep_sintetica_revocada_00000000000001',2,'per_sintetica_revocada_000000000000001','emp_sintetico_revocado_000000000000001','revocada','error_material');
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_revocada_000000000000001') = 'sin_empleado', 'revocada');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_revocada_00000000000001',3,'per_sintetica_revocada_000000000000001','emp_sintetico_revocado_000000000000001','activa',NULL)$q$, '23505', 'revocada terminal');

-- 7. Vigencia: futura y vencida no son efectivas.
SELECT pg_temp.publicar('pep_sintetica_futura_000000000000001',1,'per_sintetica_futura_00000000000000001','emp_sintetico_futuro_0000000000000001','activa',NULL, interval '3650 days', interval '3660 days');
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_futura_00000000000000001') = 'sin_empleado', 'futura');
SELECT pg_temp.publicar('pep_sintetica_vencida_00000000000001',1,'per_sintetica_vencida_0000000000000001','emp_sintetico_vencido_000000000000001','activa',NULL, interval '-30 days', interval '-10 days');
SELECT pg_temp.exigir(pg_temp.clase('per_sintetica_vencida_0000000000000001') = 'sin_empleado', 'vencida');

-- Invariantes de la historia.
SELECT pg_temp.exigir(
  pg_temp.publicar('pep_sintetica_activa_000000000000001',1,'per_sintetica_activa_00000000000000001','emp_sintetico_activo_0000000000000001','activa',NULL)
  = (SELECT registrada_en FROM vec_personal.proyeccion_empleado_persona_historia
      WHERE proyeccion_ref='pep_sintetica_activa_000000000000001' AND version=1), 'reenvío idempotente');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_activa_000000000000001',1,'per_sintetica_activa_00000000000000001','emp_sintetico_activo_0000000000000001','no_activa','baja')$q$, '23505', 'colisión de versión');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_activa_000000000000001',3,'per_sintetica_activa_00000000000000001','emp_sintetico_activo_0000000000000001','no_activa','baja')$q$, '23505', 'hueco de versión');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_activa_000000000000001',2,'per_sintetica_otra_000000000000000000001','emp_sintetico_activo_0000000000000001','activa',NULL)$q$, '23505', 'cambio de persona');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_robo_00000000000000000001',1,'per_sintetica_otra_000000000000000000001','emp_sintetico_activo_0000000000000001','activa',NULL)$q$, '23505', 'emp_ en dos personas');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_nueva_0000000000000000001',2,'per_sintetica_otra_000000000000000000001','emp_sintetico_nuevo_00000000000000000001','activa',NULL)$q$, '23505', 'primera versión distinta de 1');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_nueva_0000000000000000001',1,'per_sintetica_otra_000000000000000000001','emp_sintetico_nuevo_00000000000000000001','revocada',NULL)$q$, '23514', 'motivo obligatorio');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_sintetica_nueva_0000000000000000001',1,'per_sintetica_otra_000000000000000000001','emp_sintetico_nuevo_00000000000000000001','activa','baja')$q$, '23514', 'activa sin motivo');
SELECT pg_temp.rechaza($q$SELECT pg_temp.publicar('pep_corta',1,'per_sintetica_otra_000000000000000000001','emp_sintetico_nuevo_00000000000000000001','activa',NULL)$q$, '23514', 'referencia no canónica');
SELECT pg_temp.rechaza($q$SELECT * FROM vec_personal.resolver_empleado_canonico_persona_v1('dni_12345678Z', clock_timestamp())$q$, '22023', 'selector civil');
SELECT pg_temp.rechaza($q$SELECT * FROM vec_personal.resolver_empleado_canonico_persona_v1('per_sintetica_activa_00000000000000001', 'infinity')$q$, '22023', 'instante infinito');
SELECT pg_temp.rechaza($q$UPDATE vec_personal.proyeccion_empleado_persona_historia SET estado='revocada', motivo='baja'$q$, '55000', 'update');
SELECT pg_temp.rechaza($q$DELETE FROM vec_personal.proyeccion_empleado_persona_historia$q$, '55000', 'delete');
SELECT pg_temp.rechaza($q$TRUNCATE vec_personal.proyeccion_empleado_persona_historia$q$, '55000', 'truncate');

-- ACL: nada para PUBLIC ni para el ejecutor runtime de Personal.
SELECT pg_temp.exigir(NOT EXISTS (
  SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
   WHERE p.pronamespace = 'vec_personal'::regnamespace
     AND p.proname IN ('publicar_proyeccion_empleado_persona_v1','resolver_empleado_canonico_persona_v1',
                       'validar_version_proyeccion_empleado_v1','rechazar_mutacion_proyeccion_empleado_v1')
     AND a.grantee <> p.proowner), 'EXECUTE solo propietario');
SELECT pg_temp.exigir(NOT has_table_privilege('vec_personal_ejecutor','vec_personal.proyeccion_empleado_persona_historia','SELECT'), 'ejecutor sin SELECT');
SELECT pg_temp.exigir((SELECT relrowsecurity AND relforcerowsecurity FROM pg_class
  WHERE oid = 'vec_personal.proyeccion_empleado_persona_historia'::regclass), 'RLS forzada');
SELECT pg_temp.exigir((SELECT bool_and(prosecdef AND proconfig @> ARRAY['search_path=pg_catalog']) FROM pg_proc
  WHERE oid IN ('vec_personal.publicar_proyeccion_empleado_persona_v1(text,bigint,text,text,text,timestamptz,timestamptz,text,text,bigint,text)'::regprocedure,
                'vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)'::regprocedure)), 'SECURITY DEFINER con search_path fijo');
SELECT pg_temp.exigir((SELECT count(*) FROM vec_personal.proyeccion_empleado_persona_historia) = 11, 'historia completa');
\echo 'Personal 000016: casos de proyección persona-empleado OK'
