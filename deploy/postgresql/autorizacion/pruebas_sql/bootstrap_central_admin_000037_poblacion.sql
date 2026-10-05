\set ON_ERROR_STOP on
-- Requiere el arranque real sintético 2+1 ya confirmado por la fachada.
-- No crea perfiles, certificados, asignaciones ni control para aparentarlo.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='10s';
DO $poblacion$
DECLARE total jsonb;app jsonb;e jsonb;categoria text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.bootstrap_central_admin_v3 WHERE singleton)
 THEN RAISE EXCEPTION 'AUT37 población: fuente bootstrap real ausente'; END IF;
 total:=vec_autorizacion.administradores_efectivos_internos_v1();
 app:=vec_autorizacion.administradores_aplicacion_efectivos_internos_v3();
 IF jsonb_array_length(total)<>3 OR jsonb_array_length(app)<>2
 OR (SELECT count(DISTINCT value->>'persona_ref') FROM jsonb_array_elements(app))<>2
 THEN RAISE EXCEPTION 'AUT37 población: Sistemas alteró continuidad/doble control de Aplicación'; END IF;
 FOR e IN SELECT value FROM jsonb_array_elements(app) LOOP
  SELECT m.categoria_administrativa INTO STRICT categoria
  FROM vec_autorizacion.asignacion_perfil a JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 m USING(version_rol_ref)
  WHERE a.asignacion_ref=e->>'asignacion_ref';
  IF categoria IS DISTINCT FROM 'aplicacion'
  THEN RAISE EXCEPTION 'AUT37 población: contó Sistemas como Aplicación'; END IF;
 END LOOP;
END $poblacion$;
ROLLBACK;
