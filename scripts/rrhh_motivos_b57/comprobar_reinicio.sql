\set ON_ERROR_STOP on
SET SESSION AUTHORIZATION vec_b57_revisor;
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $r$
DECLARE h text; p text; n integer;
BEGIN
 h:=encode(sha256(convert_to(array_to_string(ARRAY['bolsa.causa_participacion.v1','actualizacion_contacto','2','Datos de contacto verificados','false','true','true','true'],E'\n'),'UTF8')),'hex');
 p:='propuesta:causa:'||encode(sha256(convert_to('bolsa.causa_participacion.propuesta.v1'||E'\n'||h||E'\nactor:rrhh:proponente','UTF8')),'hex');
 IF public.b57_get('actor:rrhh:revisor',p)<>'publicada' THEN RAISE EXCEPTION 'GET propuesta no conservado'; END IF;
 IF public.b57_publicar('actor:rrhh:revisor',p) IS NULL THEN RAISE EXCEPTION 'replay publicación postreinicio'; END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.propuesta_causa_participacion;
 IF n<>1 THEN RAISE EXCEPTION 'propuesta duplicada tras reinicio'; END IF;
END $r$;
COMMIT;
RESET SESSION AUTHORIZATION;
DO $h$ BEGIN
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE recibo_ref='recibo:b2')<>1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.datos_contacto_participacion WHERE recibo_ref='recibo:b4')<>1
 THEN RAISE EXCEPTION 'historia B2/B4 no conservada'; END IF;
END $h$;
