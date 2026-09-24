\set ON_ERROR_STOP on
-- Ejecutar como LOGIN nominal, sin SET ROLE. Cada estado conserva su motivo.
DO $prueba$
DECLARE
  superficie constant text := 'api.personal.asignaciones_dietas';
  coleccion constant text := '/api/vec/personal/asignaciones-dietas';
  detalle constant text := '/api/vec/personal/asignaciones-dietas/detalle';
  grupo constant text := '/api/vec/personal/asignaciones-dietas/grupo';
  relaciones constant text := '/api/vec/personal/relaciones-dietas';
  relacion constant text := 'rel_AbCdef0123456789_-QRST';
  empleado constant text := 'emp_AbCdef0123456789_-QRST';
BEGIN
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','peticion_invalida',superficie,coleccion,
      'registrar_inicial',NULL,NULL,400::smallint) THEN RAISE EXCEPTION '400'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','autenticacion_requerida',superficie,detalle,
      'consultar',NULL,NULL,401::smallint) THEN RAISE EXCEPTION '401'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_0123456789abcdef0123456789abcdef','acceso_denegado',superficie,detalle,
      'consultar','per_sintetico',relacion,403::smallint) THEN RAISE EXCEPTION '403'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','no_encontrada',superficie,detalle,
      'consultar',NULL,NULL,404::smallint) THEN RAISE EXCEPTION '404'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','metodo_no_permitido',superficie,coleccion,
      'metodo_no_admitido',NULL,NULL,405::smallint) THEN RAISE EXCEPTION '405'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','representacion_no_admitida',superficie,grupo,
      'grupo_corregir',NULL,relacion,406::smallint) THEN RAISE EXCEPTION '406'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','conflicto',superficie,grupo,
      'grupo_corregir','per_sintetico',relacion,409::smallint) THEN RAISE EXCEPTION '409'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','dependencia_no_disponible',superficie,relaciones,
      'consultar_relaciones','per_sintetico',empleado,503::smallint) THEN RAISE EXCEPTION '503'; END IF;
  -- Actor conocido sin empleado/selector: la clase de ruta sigue auditada.
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','dependencia_no_disponible',superficie,relaciones,
      'consultar_relaciones','per_sintetico',NULL,503::smallint) THEN RAISE EXCEPTION '503 sin empleado'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','metodo_no_permitido',superficie,coleccion,
      'metodo_no_admitido','per_sintetico',NULL,405::smallint) THEN RAISE EXCEPTION '405 sin selector'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','no_encontrada',superficie,coleccion,
      'registrar_inicial','per_sintetico',NULL,404::smallint) THEN RAISE EXCEPTION '404 sin selector'; END IF;
  IF NOT vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
      'corr_no_disponible','acceso_denegado',superficie,relaciones,
      'metodo_no_admitido','per_sintetico',NULL,403::smallint) THEN RAISE EXCEPTION '403 metodo sin instancia'; END IF;

  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','acceso_denegado',superficie,detalle,
        'consultar','per_sintetico',relacion,400::smallint);
    RAISE EXCEPTION 'motivo/estado cruzado';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','acceso_denegado',superficie,detalle,
        'consultar','per_sintetico',NULL,403::smallint);
    RAISE EXCEPTION '403 detalle sin relacion';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','conflicto',superficie,grupo,
        'grupo_corregir','per_sintetico',NULL,409::smallint);
    RAISE EXCEPTION '409 grupo sin relacion';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','acceso_denegado',superficie,relaciones,
        'consultar_relaciones','per_sintetico',NULL,403::smallint);
    RAISE EXCEPTION '403 relaciones sin empleado';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','dependencia_no_disponible',superficie,relaciones,
        'consultar_relaciones','per_sintetico',relacion,503::smallint);
    RAISE EXCEPTION 'relacion en ruta empleado';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','autenticacion_requerida',superficie,coleccion,
        'registrar_inicial','per_sintetico',relacion,401::smallint);
    RAISE EXCEPTION 'actor en 401';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','acceso_denegado',superficie,grupo,
        'consultar','per_sintetico',relacion,403::smallint);
    RAISE EXCEPTION 'ruta/accion cruzada';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','acceso_denegado',superficie,detalle,
        'consultar','actor con espacios',relacion,403::smallint);
    RAISE EXCEPTION 'actor fuera de contrato';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
  BEGIN
    PERFORM vec_personal.registrar_auditoria_frontera_asignacion_dietas_v1(
        'corr_no_disponible','acceso_denegado',superficie,detalle,
        'consultar','per_sintetico',relacion,418::smallint);
    RAISE EXCEPTION 'estado fuera de contrato';
  EXCEPTION WHEN SQLSTATE '22023' THEN NULL; END;
END
$prueba$;
