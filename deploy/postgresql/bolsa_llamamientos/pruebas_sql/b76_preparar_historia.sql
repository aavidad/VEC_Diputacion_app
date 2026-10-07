\set ON_ERROR_STOP on
-- Solo clon desechable: prepara tres actos históricos antes de instalar B76.
-- Son fixtures sintéticos insertados por el propietario, no actos reales.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $datos$
DECLARE p text; t timestamptz; v bigint;
BEGIN
 SELECT e.participacion_ref INTO STRICT p FROM vec_bolsa_llamamientos.constitucion_entrada e
 JOIN LATERAL (SELECT s.situacion FROM vec_bolsa_llamamientos.situacion_participacion s
  WHERE s.participacion_ref=e.participacion_ref ORDER BY s.desde DESC LIMIT 1) a ON true
 WHERE a.situacion='disponible' LIMIT 1;
 SELECT greatest(clock_timestamp(),max(desde)) + interval '1 hour' INTO t FROM vec_bolsa_llamamientos.situacion_participacion;
 SELECT max(version) INTO v FROM vec_bolsa_llamamientos.politica_transiciones_situacion;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion
 (participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version)
 VALUES (p,'renuncia',t,'Fixture histórico B76','persona:rrhh-b76',t,'b76:legacy:renuncia','recibo:b76:legacy:renuncia',v),
 (p,'no_disponible',t+interval '1 second','Fixture histórico B76','persona:rrhh-b76',t+interval '1 second','b76:legacy:pausar','recibo:b76:legacy:pausar',v),
 (p,'disponible',t+interval '2 second','Fixture histórico B76','persona:rrhh-b76',t+interval '2 second','b76:legacy:reactivar','recibo:b76:legacy:reactivar',v);
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion
 (participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,
 actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES (p,t+interval '1 second','pausar','resolucion','justificante:b76:legacy',repeat('b',64),'persona:rrhh-b76','persona:validador-b76',t,t+interval '1 second','b76:legacy:pausar'),
 (p,t+interval '2 second','reactivar','resolucion','justificante:b76:legacy',repeat('b',64),'persona:rrhh-b76','persona:validador-b76',t,t+interval '2 second','b76:legacy:reactivar');
END $datos$;
COMMIT;
