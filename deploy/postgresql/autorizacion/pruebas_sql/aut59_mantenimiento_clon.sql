\set ON_ERROR_STOP on
-- AUT59 POSITIVO, sólo clon desechable PostgreSQL 18 como superusuario.
-- Precondiciones: AUT59 instalada, Rol8 publicado por la CLI con plan v4 y
-- aprobación externa vigente, dos APP v5 vivas y una decisión V3 real de
-- vec-admin tomada después de publicar Rol8. Proporcionar
--   psql -X -v ON_ERROR_STOP=1 -v aut59_operacion=pmf_... -f este_archivo
-- No crea aprobaciones, identidades ni decisiones. El replay y su intento
-- quedan deshechos por ROLLBACK; los registros originales se conservan.
\if :{?aut59_operacion}
\else
\echo 'AUT59: falta -v aut59_operacion=pmf_... de la publicación original'
\quit 3
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';

CREATE TEMP TABLE aut59_fuente AS
SELECT r.*,convert_from(r.plan,'UTF8') AS plan_texto
FROM vec_autorizacion.registro_mantenimiento_gobierno_definiciones_admin_v1 r
WHERE r.operacion_ref=:'aut59_operacion';
DO $pre$
DECLARE r record;
BEGIN
 SELECT * INTO STRICT r FROM aut59_fuente;
 IF r.plan_texto::jsonb::text IS DISTINCT FROM r.plan_texto
 OR encode(sha256(r.plan),'hex') IS DISTINCT FROM r.plan_sha256
 OR r.plan_texto::jsonb->>'version' IS DISTINCT FROM '4'
 OR r.recibo->>'rol_origen_ref' IS DISTINCT FROM 'rol:administracion_perfiles:v7'
 OR r.recibo->>'rol_destino_ref' IS DISTINCT FROM 'rol:administracion_perfiles:v8'
 OR r.recibo->>'plan_sha256' IS DISTINCT FROM r.plan_sha256
 OR jsonb_array_length(r.recibo->'asignaciones') IS DISTINCT FROM 2
 OR (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual q
     JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
     WHERE a.version_rol_ref='rol:administracion_perfiles:v8' AND a.version=5)<>2
 THEN RAISE EXCEPTION 'AUT59: falta publicación íntegra del plan v4'; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.config_mantenimiento_gobierno_definiciones_admin_v1 c
   WHERE c.login_nombre=r.login_nombre AND c.plan_sha256=r.plan_sha256
   AND clock_timestamp()>=c.vigente_desde AND clock_timestamp()<c.vigente_hasta)
 THEN RAISE EXCEPTION 'AUT59: aprobación original caducada; usar otro clon/ventana'; END IF;
END $pre$;
CREATE TEMP TABLE aut59_aud_base AS
SELECT max(secuencia) AS ultima FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;

-- Sólo el superusuario preparador lee el plan privado. El LOGIN técnico llama
-- exclusivamente a la fachada con los bytes del mismo registro inmutable.
SELECT plan_texto AS aut59_plan,plan_sha256 AS aut59_sha,login_nombre::text AS aut59_login
FROM aut59_fuente \gset
SET SESSION AUTHORIZATION :"aut59_login";
SELECT vec_autorizacion.mantener_version_perfil_fijo_gobierno_definiciones_admin_v1(
  :'aut59_plan',:'aut59_sha') AS aut59_replay \gset
RESET SESSION AUTHORIZATION;
CREATE TEMP TABLE aut59_resultado AS SELECT :'aut59_replay'::jsonb AS valor;

DO $replay$
DECLARE r record;x jsonb;
BEGIN
 SELECT * INTO STRICT r FROM aut59_fuente;
 SELECT valor INTO STRICT x FROM aut59_resultado;
 IF x->>'estado' IS DISTINCT FROM 'permitido'
 OR (x->>'replay')::boolean IS NOT TRUE
 OR x->'recibo' IS DISTINCT FROM r.recibo
 OR (SELECT count(*) FROM vec_autorizacion.registro_mantenimiento_gobierno_definiciones_admin_v1
     WHERE operacion_ref=r.operacion_ref)<>1
 OR (SELECT count(*) FROM vec_autorizacion.version_rol
     WHERE version_rol_ref='rol:administracion_perfiles:v8')<>1
 OR (SELECT count(*) FROM vec_autorizacion.asignacion_perfil a
     WHERE a.version_rol_ref='rol:administracion_perfiles:v8' AND a.version=5)<>2
 OR (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a,
      aut59_aud_base b WHERE a.secuencia>coalesce(b.ultima,0)
      AND a.tipo_registro='intento_mantenimiento_perfil_fijo_admin'
      AND a.resultado='permitido' AND a.motivo_ref='mantenimiento_replay')<>1
 THEN RAISE EXCEPTION 'AUT59: replay no devolvió recibo original o duplicó efecto: %',x; END IF;
END $replay$;

-- La decisión real posterior a Rol8 prueba las dos concesiones nominales y
-- un recurso ajeno. Se requiere recorrido previo de vec-admin en este clon.
CREATE TEMP TABLE aut59_decision AS
SELECT convert_from(a.decision_canonica,'UTF8')::jsonb AS valor
FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
WHERE convert_from(a.decision_canonica,'UTF8')::jsonb->>'version_rol_ref'='rol:administracion_perfiles:v8'
AND convert_from(a.decision_canonica,'UTF8')::jsonb#>>'{vinculo_autenticacion_actor,superficie}'='administracion_privilegiada'
ORDER BY a.registrada_en DESC LIMIT 1;
DO $nominal$
DECLARE d jsonb;accion text;tipo text;
BEGIN
 SELECT valor INTO STRICT d FROM aut59_decision;
 FOREACH accion IN ARRAY ARRAY['administracion.perfiles.definicion.proponer','administracion.perfiles.definicion.aprobar'] LOOP
  tipo:=CASE WHEN accion LIKE '%.proponer' THEN 'definicion_rol' ELSE 'propuesta_definicion_rol' END;
  IF vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
      accion,'administracion',tipo,'gobierno_definiciones_perfiles','[]',d->'vinculo_autenticacion_actor') IS NOT TRUE
  THEN RAISE EXCEPTION 'AUT59: Rol8 no acredita %',accion; END IF;
 END LOOP;
 IF vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'administracion.perfiles.definicion.aprobar','administracion','definicion_rol','gobierno_definiciones_perfiles','[]',d->'vinculo_autenticacion_actor') IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'administracion.perfiles.definicion.proponer','otro','definicion_rol','gobierno_definiciones_perfiles','[]',d->'vinculo_autenticacion_actor') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT59: Rol8 acredita fuera del par nominal'; END IF;
END $nominal$;
ROLLBACK;
SELECT 'AUT59-MANTENIMIENTO-CLON-OK';
