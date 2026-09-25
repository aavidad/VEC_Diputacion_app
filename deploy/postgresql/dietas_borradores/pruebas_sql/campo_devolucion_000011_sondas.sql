\set ON_ERROR_STOP on
-- Superficie EXCLUSIVA del contenedor desechable del ensayo de AD3-75 y
-- Dietas 000011; se carga después de dietas_documento_ad3_000059_sondas.sql.
-- Igual que prueba59.preparar, pero con la lista de campos de la decisión
-- elegida por el ensayo: así se comprueba qué listas admite cada fachada AD3
-- REAL (la guarda de campos está antes del núcleo; sin clave para la
-- audiencia, una lista admitida se detiene en «capacidad VEC-AD-3 rechazada»).
BEGIN;
SET LOCAL search_path=pg_catalog;
CREATE FUNCTION prueba59.preparar_campos(p_caso text,p_operacion text,p_audiencia text,
 p_tipo text,p_finalidad text,p_campos jsonb) RETURNS void
LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
DECLARE b prueba59.material; c jsonb; d jsonb; decision bytea; efecto text; huella text;
BEGIN
 SELECT * INTO STRICT b FROM prueba59.material WHERE caso='ct_alta';
 efecto:='prueba75:'||p_caso;
 huella:=encode(sha256(convert_to(efecto,'UTF8')),'hex');
 d:=convert_from(b.decision,'UTF8')::jsonb
   || jsonb_build_object('decision_ref','decision:prueba75:'||p_caso,'accion',p_operacion,
      'modulo_id','dietas','tipo_recurso',p_tipo,'finalidad',p_finalidad,'recurso_ref',efecto,
      'contexto_recurso_huella_sha256',huella,'campos_permitidos',p_campos,'obligaciones','[]'::jsonb);
 decision:=convert_to(d::text,'UTF8');
 c:=convert_from(b.capacidad,'UTF8')::jsonb
   || jsonb_build_object('decision_ref',d->>'decision_ref','operacion',p_operacion,
      'audiencia_consumo',p_audiencia,'efecto_ref',efecto,'huella_efecto_sha256',huella,
      'huella_decision_sha256',encode(sha256(decision),'hex'),
      'nonce',encode(sha256(convert_to('nonce:prueba75:'||p_caso,'UTF8')),'hex'));
 DELETE FROM prueba59.material WHERE caso=p_caso;
 INSERT INTO prueba59.material VALUES (p_caso,
  vec_autorizacion_atestada_v3.capacidad_canonica(c),decision,b.motivo,b.contexto,
  b.persona_version,b.perfil_version,b.payload,b.sobre,b.evidencia,b.raiz);
END $f$;
REVOKE ALL ON FUNCTION prueba59.preparar_campos(text,text,text,text,text,jsonb) FROM PUBLIC;
COMMIT;
