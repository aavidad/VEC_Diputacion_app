\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';

DO $prueba$
DECLARE b text; p1 text; p2 text; base timestamptz; filas bigint; orden1 bigint; orden2 bigint; posicion_registrada bigint; v_razon text;
BEGIN
 SELECT c.bolsa_ref,(array_agg(e.participacion_ref ORDER BY e.orden))[1],(array_agg(e.participacion_ref ORDER BY e.orden))[2],clock_timestamp()+interval '1 second'
   INTO STRICT b,p1,p2,base
   FROM vec_bolsa_llamamientos.constitucion c
   JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
  GROUP BY c.bolsa_ref HAVING count(*)>=2 ORDER BY max(c.confirmada_en) DESC LIMIT 1;

 -- Una pausa no ocupa turno y desplaza de forma reproducible al disponible.
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p1,'no_disponible',base,'Pausa sintética B6','persona:prueba-b6',base,'b6:pausa','recibo:b6:pausa');
 SELECT orden_vigente INTO STRICT orden2 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b,base+interval '1 millisecond') WHERE participacion_ref=p2;
 IF orden2<>1 THEN RAISE EXCEPTION 'B6: la pausa no liberó el turno'; END IF;

 -- Se reactiva, pasa a pendiente/trabajando y vuelve disponible. La política
 -- v2 de prueba coloca al final y el trigger deja una sola reposición explicable.
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref) VALUES
 (p1,'disponible',base+interval '1 second','Reactivación sintética B6','persona:prueba-b6',base+interval '1 second','b6:reactiva','recibo:b6:reactiva'),
 (p1,'pendiente_incorporacion',base+interval '2 second','Propuesta sintética B6','persona:prueba-b6',base+interval '2 second','b6:pendiente','recibo:b6:pendiente'),
 (p1,'trabajando',base+interval '3 second','Contrato sintético B6','persona:prueba-b6',base+interval '3 second','b6:trabajando','recibo:b6:trabajando');
 INSERT INTO vec_bolsa_llamamientos.politica_orden_bolsa(politica_ref,bolsa_ref,version,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,registrada_en)
 SELECT politica_ref,bolsa_ref,2,criterio,'rotatoria','fin_lista',true,rotulo,'persona:prueba-b6',base+interval '3500 milliseconds',base+interval '3500 milliseconds'
   FROM vec_bolsa_llamamientos.politica_orden_bolsa WHERE bolsa_ref=b AND version=1;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p1,'disponible',base+interval '4 second','Fin de contrato sintético B6','persona:prueba-b6',base+interval '4 second','b6:repuesta','recibo:b6:repuesta');
 SELECT q.orden_vigente,q.razon INTO STRICT orden1,v_razon FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b,base+interval '5 second') q WHERE q.participacion_ref=p1;
 SELECT orden_vigente INTO STRICT orden2 FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(b,base+interval '5 second') WHERE participacion_ref=p2;
 SELECT count(*),max(posicion_resultante) INTO filas,posicion_registrada FROM vec_bolsa_llamamientos.reposicion_orden_bolsa WHERE situacion_recibo_ref='recibo:b6:repuesta';
 IF orden1<>2 OR orden2<>1 OR posicion_registrada<>orden1 OR v_razon<>'reposicion_tras_contrato' OR filas<>1 THEN RAISE EXCEPTION 'B6: reposición no explicable (%,%,%,%,%)',orden1,orden2,posicion_registrada,v_razon,filas; END IF;
 IF strpos(pg_get_functiondef('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure),'leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido)')=0 THEN
  RAISE EXCEPTION 'B6: B7 no reserva contra el orden vigente';
 END IF;
END $prueba$;

-- El ejecutor solo consume la función estrecha; las tablas permanecen cerradas.
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl$
BEGIN
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.politica_orden_bolsa; RAISE EXCEPTION 'B6: tabla visible al ejecutor';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl$;
ROLLBACK;
