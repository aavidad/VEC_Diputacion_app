\set ON_ERROR_STOP on
-- Postcondición DBA tras operación/COMMIT reales. No crea identidades ni fuentes.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
DO $pruebas$
DECLARE a record;p jsonb:=current_setting('vec.ensayo.plan_mantenimiento')::jsonb;old_a record;
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE x.version_rol_ref='rol:administracion_perfiles:v5')<>2 THEN RAISE EXCEPTION 'AUT42 prueba: población APP5 divergente';END IF;
 FOR a IN SELECT x.* FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) WHERE x.version_rol_ref='rol:administracion_perfiles:v5' LOOP
  SELECT * INTO STRICT old_a FROM vec_autorizacion.asignacion_perfil WHERE asignacion_id=a.asignacion_id AND version=1;
  IF a.version<>2 OR (a.principal_id,a.perfil_activo_ref,a.documento->'ambitos',a.documento->>'vigente_desde',a.documento->>'vigente_hasta') IS DISTINCT FROM (old_a.principal_id,old_a.perfil_activo_ref,old_a.documento->'ambitos',old_a.documento->>'vigente_desde',old_a.documento->>'vigente_hasta') THEN RAISE EXCEPTION 'AUT42 prueba: identidad ámbito o vigencia cambiados';END IF;
  IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.sello_efecto_admin_tx_v1 z WHERE z.asignacion_ref=a.asignacion_ref AND z.operacion_ref=p->>'operacion_ref') THEN RAISE EXCEPTION 'AUT42 prueba: sello común ausente';END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual q JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref) JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 meta USING(version_rol_ref) WHERE meta.categoria_administrativa='sistemas' AND x.version=1 AND x.documento->>'estado'='activa') THEN RAISE EXCEPTION 'AUT42 prueba: Sistemas alterado';END IF;
 IF vec_autorizacion.validar_administrador_denominacion_persona_v1('{}','{}') THEN RAISE EXCEPTION 'AUT42 prueba: gate permite sin decisión/material';END IF;
END $pruebas$;
ROLLBACK;
