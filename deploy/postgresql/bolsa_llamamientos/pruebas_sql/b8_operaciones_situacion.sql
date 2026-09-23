\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE b text; p text; base timestamptz; posicion_antes bigint; posicion_despues bigint; politica text; filas bigint;
BEGIN
 SELECT c.bolsa_ref, e.participacion_ref, clock_timestamp()+interval '1 hour'
   INTO STRICT b,p,base
   FROM vec_bolsa_llamamientos.constitucion c
   JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
   JOIN LATERAL (SELECT situacion FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=e.participacion_ref ORDER BY desde DESC LIMIT 1) actual ON true
  WHERE actual.situacion='disponible' LIMIT 1;
 SELECT orden_vigente,reposicion INTO STRICT posicion_antes,politica FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b,base-interval '1 second') WHERE participacion_ref=p;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'no_disponible',base,'Pausa B8 sintética','persona:operador-b8',base,'b8:pausa','recibo:b8:pausa');
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES(p,base,'pausar','solicitud_candidato','justificante:pausa',repeat('a',64),'persona:operador-b8','persona:operador-b8',base,base,'b8:pausa');
 IF EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b,base+interval '1 millisecond') WHERE participacion_ref=p AND orden_vigente IS NOT NULL) THEN RAISE EXCEPTION 'B8: pausa mantiene turno'; END IF;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'disponible',base+interval '1 second','Reactivación B8 sintética','persona:operador-b8',base+interval '1 second','b8:reactiva','recibo:b8:reactiva');
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES(p,base+interval '1 second','reactivar','solicitud_candidato','justificante:reactiva',repeat('b',64),'persona:operador-b8','persona:operador-b8',base+interval '1 second',base+interval '1 second','b8:reactiva');
 SELECT orden_vigente INTO STRICT posicion_despues FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b,base+interval '2 second') WHERE participacion_ref=p;
 IF posicion_despues<>posicion_antes OR politica<>'misma_posicion' OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.reposicion_orden_bolsa WHERE situacion_recibo_ref='recibo:b8:reactiva') THEN RAISE EXCEPTION 'B8: pausa alteró posición o política B6'; END IF;
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
  VALUES(p,base+interval '3 second','excluir','resolucion','justificante:excluye',repeat('c',64),'persona:operador-b8','persona:operador-b8',base+interval '3 second',base+interval '3 second','b8:excluye');
  RAISE EXCEPTION 'B8: exclusión sin segunda identidad aceptada';
 EXCEPTION WHEN check_violation THEN NULL; END;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'excluido',base+interval '3 second','Exclusión B8 sintética','persona:operador-b8',base+interval '3 second','b8:excluye','recibo:b8:excluye');
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES(p,base+interval '3 second','excluir','resolucion','justificante:excluye',repeat('c',64),'persona:operador-b8','persona:validador-b8',base+interval '3 second',base+interval '3 second','b8:excluye');
 SELECT count(*) INTO filas FROM vec_bolsa_llamamientos.listar_operaciones_situacion_participacion_v1(p);
 IF filas<>3 THEN RAISE EXCEPTION 'B8: historial incompleto'; END IF;
END $prueba$;
ROLLBACK;
