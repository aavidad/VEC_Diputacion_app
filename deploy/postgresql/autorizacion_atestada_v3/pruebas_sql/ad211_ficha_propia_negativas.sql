\set ON_ERROR_STOP on
-- El rechazo de material ausente ocurre antes de cualquier consumo o auditoría.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE antes_consumo bigint; antes_auditoria bigint; despues_consumo bigint; despues_auditoria bigint;
BEGIN
 SELECT count(*) INTO antes_consumo FROM vec_autorizacion_atestada_v3.consumo_decision_v3;
 SELECT count(*) INTO antes_auditoria FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 BEGIN
  PERFORM * FROM vec_autorizacion_atestada_v3.consumir_ficha_propia_empleado_v3_atestada(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'AD211 prueba: clave=material_nulo esperado=42501 actual=aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 SELECT count(*) INTO despues_consumo FROM vec_autorizacion_atestada_v3.consumo_decision_v3;
 SELECT count(*) INTO despues_auditoria FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3;
 IF antes_consumo IS DISTINCT FROM despues_consumo OR antes_auditoria IS DISTINCT FROM despues_auditoria THEN
  RAISE EXCEPTION 'AD211 prueba: clave=sin_efecto esperado=consumo_%,auditoria_% actual=consumo_%,auditoria_%',
   antes_consumo,antes_auditoria,despues_consumo,despues_auditoria;
 END IF;
END $prueba$;
ROLLBACK;
