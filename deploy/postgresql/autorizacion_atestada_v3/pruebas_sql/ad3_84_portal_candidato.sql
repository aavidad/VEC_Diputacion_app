\set ON_ERROR_STOP on
-- AD3-84: la fachada del portal del candidato admite exactamente las cuatro
-- acciones propias, cada una con su audiencia y su tipo de recurso, y rechaza
-- cualquier cruce antes de llegar al núcleo. Una forma correcta sí llega al
-- núcleo, que la rechaza aquí por falta de material firmado (otro mensaje).
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
DO $prueba$
DECLARE
 caso record; mensaje text;
 cand text := 'mi-bolsa:can_portal_sintetico_0000000001';
 oferta text := 'oferta:'||repeat('a',64);
BEGIN
 FOR caso IN
  SELECT * FROM (VALUES
   ('solicitar_pausa', 'solicitar_pausa', 'participaciones_candidato', cand, true),
   ('solicitar_reactivacion', 'solicitar_reactivacion', 'participaciones_candidato', cand, true),
   ('responder_llamamiento', 'responder_llamamiento', 'participaciones_candidato', cand, true),
   ('manifestar_disposicion', 'manifestar_disposicion', 'oferta_bolsa', oferta, true),
   -- Cruces que la fachada debe rechazar.
   ('solicitar_pausa', 'responder_llamamiento', 'participaciones_candidato', cand, false),
   ('manifestar_disposicion', 'manifestar_disposicion', 'participaciones_candidato', cand, false),
   ('manifestar_disposicion', 'manifestar_disposicion', 'oferta_bolsa', cand, false),
   ('solicitar_pausa', 'solicitar_pausa', 'oferta_bolsa', oferta, false),
   ('solicitar_pausa', 'solicitar_pausa', 'participaciones_candidato', 'mi-bolsa:otro', false),
   ('consultar', 'consultar', 'participaciones_candidato', cand, false)
  ) AS t(operacion, audiencia, tipo, recurso, admitido)
 LOOP
  BEGIN
   PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(
    convert_to(jsonb_build_object('operacion','bolsa.participaciones_propias.'||caso.operacion,
      'audiencia_consumo','vec_bolsa_llamamientos.participaciones_propias.'||caso.audiencia||'.v1',
      'efecto_ref',caso.recurso,'huella_efecto_sha256',repeat('0',64))::text,'UTF8'),
    convert_to(jsonb_build_object('accion','bolsa.participaciones_propias.'||caso.operacion,'modulo_id','bolsa',
      'tipo_recurso',caso.tipo,'finalidad','gestion_participaciones_propias','recurso_ref',caso.recurso,
      'contexto_recurso_huella_sha256',repeat('0',64),'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb)::text,'UTF8'),
    NULL, NULL, 1, 1, NULL, NULL, NULL, NULL);
   RAISE EXCEPTION 'AD3-84: material sin firma aceptado (%)', caso.operacion;
  EXCEPTION WHEN others THEN
   mensaje := SQLERRM;
   IF mensaje LIKE 'AD3-84: material sin firma aceptado%' THEN RAISE; END IF;
   IF caso.admitido = (mensaje = 'AD3-84: material del portal del candidato rechazado') THEN
    RAISE EXCEPTION 'AD3-84: guarda inexacta en % → % (%)', caso.operacion, caso.tipo, mensaje;
   END IF;
  END;
 END LOOP;
END $prueba$;
ROLLBACK;

-- Solo el propietario de Bolsa ejecuta la fachada.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl$ BEGIN
 PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_portal_candidato_bolsa_v3_atestada(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
 RAISE EXCEPTION 'AD3-84: fachada alcanzable por el ejecutor';
EXCEPTION WHEN insufficient_privilege THEN NULL;
END $acl$;
ROLLBACK;
