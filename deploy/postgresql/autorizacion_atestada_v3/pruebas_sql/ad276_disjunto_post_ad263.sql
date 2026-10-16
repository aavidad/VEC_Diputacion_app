\set ON_ERROR_STOP on
-- Lectura local tras la cadena de Documentos hasta AD287, B118 y AD276.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog,pg_temp;
DO $prueba$ DECLARE disjunto text;validas int;
BEGIN
 SELECT pg_get_constraintdef(c.oid) INTO STRICT disjunto
 FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname='auditoria_tipo_disjunto_v4';
 SELECT count(*) INTO validas FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname IN ('auditoria_tipo_disjunto_v4',
     'auditoria_lectura_exclusiones_llamamiento_formato_v1')
   AND c.convalidated;
 IF validas<>2 OR position('confianza_externa' IN disjunto)=0
 OR position('lectura_exclusiones_llamamiento' IN disjunto)=0
 OR position('lectura_respuestas_llamamiento' IN disjunto)=0
 OR position('preperfil_bolsa_rrhh' IN disjunto)=0
 OR position('bolsa.justificante_renuncia.perfil_propio.consultar' IN disjunto)=0
 THEN RAISE EXCEPTION 'AD276: se perdió una rama o falta validar la guarda'; END IF;
END $prueba$;
ROLLBACK;
