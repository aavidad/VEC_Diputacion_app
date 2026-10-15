\set ON_ERROR_STOP on
-- Ensayo focal en una copia con AUT97 instalado y un contexto nominal real de
-- la primera firma del informe técnico. La preparación privada de Dirección
-- deja una fila en pg_temp.aut97_firma_fixture(contexto bytea,
-- relacion_ct jsonb, consumo_v3 jsonb) en la misma sesión Y transacción
-- serializable ya abierta. El consumo y su auditoría deben ser de esa
-- transacción; no sirven los de una firma confirmada anteriormente.
-- No se guarda el contexto, el certificado ni el recibo en Git o en la salida.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
DO $test$
DECLARE f oid; respuesta jsonb;
BEGIN
 f:=to_regprocedure('vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(bytea,jsonb,jsonb)');
 IF f IS NULL OR to_regclass('pg_temp.aut97_firma_fixture') IS NULL THEN
  RAISE EXCEPTION 'AUT97: función o fixture privado ausente';
 END IF;
 IF (SELECT count(*) FROM pg_temp.aut97_firma_fixture)<>1 THEN
  RAISE EXCEPTION 'AUT97: se espera un único contexto nominal';
 END IF;
 SELECT vec_autorizacion.validar_competencia_nominal_firmante_ct_v1(
   contexto,relacion_ct,consumo_v3)
 INTO STRICT respuesta FROM pg_temp.aut97_firma_fixture;
 IF jsonb_typeof(respuesta) IS DISTINCT FROM 'object'
   OR respuesta->'competencia' IS NULL THEN
  RAISE EXCEPTION 'AUT97: validación nominal no devolvió competencia';
 END IF;
END $test$;
ROLLBACK;
