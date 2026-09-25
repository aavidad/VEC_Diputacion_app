\set ON_ERROR_STOP on
-- Prueba de 000033 sobre una base con B8 (000019) y 000033 instaladas y al
-- menos una participación con situación. Todo termina en ROLLBACK.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE p text; base timestamptz; v record; n bigint; lista text;
BEGIN
 SELECT participacion_ref, clock_timestamp()+interval '1 hour' INTO STRICT p, base
   FROM vec_bolsa_llamamientos.situacion_participacion ORDER BY desde LIMIT 1;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.consultar_politica_segregacion_v1();
 IF v.operaciones <> ARRAY['excluir'] THEN RAISE EXCEPTION 'B33: la política inicial no es la de hoy'; END IF;
 SELECT max(version) INTO STRICT n FROM vec_bolsa_llamamientos.politica_segregacion;

 -- Con la política inicial la pausa admite a la misma persona.
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'no_disponible',base,'Pausa B33','persona:operador-b33',base,'b33:pausa-1','recibo:b33:pausa-1');
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES(p,base,'pausar','solicitud_candidato','justificante:b33-1',repeat('a',64),'persona:operador-b33','persona:operador-b33',base,base,'b33:pausa-1');
 IF (SELECT politica_segregacion_version FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p AND desde=base) IS DISTINCT FROM n THEN
  RAISE EXCEPTION 'B33: la fila no guarda la versión aplicada';
 END IF;

 -- Publicar: canoniza, versiona y es idempotente.
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.publicar_politica_segregacion_v1('vec.bolsa.roles_segregacion:1:s01.segunda_persona', repeat('b',64), ARRAY['excluir','pausar']);
 IF v.reutilizada OR v.version <> n+1 OR v.operaciones <> ARRAY['pausar','excluir'] THEN RAISE EXCEPTION 'B33: publicación inesperada %', v; END IF;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.publicar_politica_segregacion_v1('vec.bolsa.roles_segregacion:1:s01.segunda_persona', repeat('b',64), ARRAY['pausar','excluir']);
 IF NOT v.reutilizada OR v.version <> n+1 THEN RAISE EXCEPTION 'B33: republicar creó historia'; END IF;

 -- Reactivar sigue sin exigir otra persona.
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'disponible',base+interval '1 second','Reactivación B33','persona:operador-b33',base+interval '1 second','b33:reactiva','recibo:b33:reactiva');
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES(p,base+interval '1 second','reactivar','solicitud_candidato','justificante:b33-2',repeat('c',64),'persona:operador-b33','persona:operador-b33',base+interval '1 second',base+interval '1 second','b33:reactiva');

 -- La pausa ya exige segunda persona.
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
  VALUES(p,'no_disponible',base+interval '2 second','Pausa B33','persona:operador-b33',base+interval '2 second','b33:pausa-2','recibo:b33:pausa-2');
  INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
  VALUES(p,base+interval '2 second','pausar','solicitud_candidato','justificante:b33-3',repeat('d',64),'persona:operador-b33','persona:operador-b33',base+interval '2 second',base+interval '2 second','b33:pausa-2');
  RAISE EXCEPTION 'B33: pausa autovalidada aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
 VALUES(p,'no_disponible',base+interval '2 second','Pausa B33','persona:operador-b33',base+interval '2 second','b33:pausa-2','recibo:b33:pausa-2');
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
 VALUES(p,base+interval '2 second','pausar','solicitud_candidato','justificante:b33-3',repeat('d',64),'persona:operador-b33','persona:validador-b33',base+interval '2 second',base+interval '2 second','b33:pausa-2');
 IF (SELECT politica_segregacion_version FROM vec_bolsa_llamamientos.operacion_situacion_participacion WHERE participacion_ref=p AND desde=base+interval '2 second') IS DISTINCT FROM n+1 THEN
  RAISE EXCEPTION 'B33: la fila no guarda la versión nueva';
 END IF;

 -- La exclusión autovalidada sigue rechazada por la política y por el CHECK fijo.
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref,situacion,desde,motivo,actor,registrada_en,clave_idempotencia,recibo_ref)
  VALUES(p,'excluido',base+interval '3 second','Exclusión B33','persona:operador-b33',base+interval '3 second','b33:excluye','recibo:b33:excluye');
  INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref,desde,operacion,justificante_tipo,justificante_ref,justificante_sha256,actor,validador,validada_en,registrada_en,clave_idempotencia)
  VALUES(p,base+interval '3 second','excluir','resolucion','justificante:b33-4',repeat('e',64),'persona:operador-b33','persona:operador-b33',base+interval '3 second',base+interval '3 second','b33:excluye');
  RAISE EXCEPTION 'B33: exclusión autovalidada aceptada';
 EXCEPTION WHEN invalid_parameter_value OR check_violation THEN NULL; END;

 -- Ninguna publicación puede quitar la exclusión ni meter operaciones ajenas.
 FOREACH lista IN ARRAY ARRAY['pausar,pausar','excluir,excluir','excluir,cancelar','excluir,','pausar'] LOOP
  BEGIN
   PERFORM vec_bolsa_llamamientos.publicar_politica_segregacion_v1('ref:b33', repeat('f',64),
     ARRAY(SELECT NULLIF(e,'') FROM unnest(string_to_array(lista, ',')) e));
   RAISE EXCEPTION 'B33: publicación inválida aceptada %', lista;
  EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 END LOOP;
 BEGIN
  PERFORM vec_bolsa_llamamientos.publicar_politica_segregacion_v1('ref:b33', repeat('f',64), NULL);
  RAISE EXCEPTION 'B33: política nula aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.politica_segregacion(version,catalogo_ref,catalogo_sha256,operaciones,publicada_en)
  VALUES(n+10,'ref:b33',repeat('f',64),ARRAY['pausar'],clock_timestamp());
  RAISE EXCEPTION 'B33: fila sin exclusión aceptada';
 EXCEPTION WHEN check_violation THEN NULL; END;

 -- Historia de solo adición.
 BEGIN
  UPDATE vec_bolsa_llamamientos.politica_segregacion SET operaciones=ARRAY['excluir'] WHERE version=n+1;
  RAISE EXCEPTION 'B33: política modificada';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'B33:%' THEN RAISE; END IF; END;
 BEGIN
  DELETE FROM vec_bolsa_llamamientos.politica_segregacion WHERE version=n+1;
  RAISE EXCEPTION 'B33: política borrada';
 EXCEPTION WHEN others THEN IF SQLERRM LIKE 'B33:%' THEN RAISE; END IF; END;
END $prueba$;

SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl$
DECLARE v record;
BEGIN
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.politica_segregacion;
  RAISE EXCEPTION 'B33: lectura directa de la política permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  INSERT INTO vec_bolsa_llamamientos.politica_segregacion(version,catalogo_ref,catalogo_sha256,operaciones,publicada_en)
  VALUES(999,'ref:b33',repeat('f',64),ARRAY['excluir'],clock_timestamp());
  RAISE EXCEPTION 'B33: escritura directa de la política permitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.consultar_politica_segregacion_v1();
 IF v.operaciones <> ARRAY['pausar','excluir'] THEN RAISE EXCEPTION 'B33: el ejecutor no ve la vigente'; END IF;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.publicar_politica_segregacion_v1('vec.bolsa.roles_segregacion:1:s01.segunda_persona', repeat('b',64), ARRAY['pausar','excluir']);
 IF NOT v.reutilizada THEN RAISE EXCEPTION 'B33: el ejecutor no publica por la función'; END IF;
END $acl$;
RESET ROLE;
DO $publico$
BEGIN
 IF has_function_privilege('public', 'vec_bolsa_llamamientos.publicar_politica_segregacion_v1(text,text,text[])', 'EXECUTE')
    OR has_function_privilege('public', 'vec_bolsa_llamamientos.consultar_politica_segregacion_v1()', 'EXECUTE')
    OR has_function_privilege('public', 'vec_bolsa_llamamientos.aplicar_politica_segregacion()', 'EXECUTE') THEN
  RAISE EXCEPTION 'B33: PUBLIC conserva EXECUTE';
 END IF;
END $publico$;
-- Constancia de quién publicó: cada versión guarda la cuenta de conexión.
DO $publicador$ BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_segregacion WHERE publicada_por IS DISTINCT FROM session_user) THEN
  RAISE EXCEPTION 'la política no deja constancia de quién la publicó';
 END IF;
END $publicador$;
ROLLBACK;
