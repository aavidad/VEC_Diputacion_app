-- B75: inserción nueva con política R1, política completa y modo histórico.
-- Datos sintéticos; una transacción revierte todo. La prueba B71 separada
-- recorre publicar_oferta_v4 y su replay con el trigger completo activo.
\set ON_ERROR_STOP on
BEGIN;
SET LOCAL timezone='UTC';
SET LOCAL session_replication_role=replica;
INSERT INTO vec_bolsa_llamamientos.bolsa_constituida VALUES
 ('bolsa:b75:sintetica',1,encode(sha256('{}'::bytea),'hex'),'{}'::bytea,
  'categoria:rpt:aux',now()-interval '10 days',NULL,'vigente',now()-interval '10 days');
SET LOCAL session_replication_role=origin;

INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version
 (bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,
  version_esperada,recibo_ref,publicada_en,decision_ref,auditoria_ref)
SELECT 'bolsa:b75:sintetica',v,p,encode(sha256(convert_to(p::text,'UTF8')),'hex'),true,
 'per_actoractoractoractoractor','clave-politica-b75-'||v,v-1,
 'recibo:politica-ofertas:'||encode(sha256(convert_to('b75-politica-'||v,'UTF8')),'hex'),
 now()-interval '2 days','decision:b75:politica:'||v,'auditoria:b75:politica:'||v
FROM generate_series(1,3) v
CROSS JOIN LATERAL (
 SELECT jsonb_build_object(
  'plazo',jsonb_build_object('inicio','notificacion','unidad','dias_naturales',
    'cantidad',2,'computo','administrativo','municipio_sede','18087'),
  'adjudicacion',jsonb_build_object('criterio','orden_vigente','elegibilidad','disposicion_en_plazo')
    || CASE WHEN v=2 THEN '{"confirmacion":"aceptacion_previa"}'::jsonb ELSE '{}'::jsonb END,
  'no_cubierta',jsonb_build_object('accion','llamamiento_directo','condicion','sin_disposiciones_elegibles'))
  || CASE WHEN v<>3 THEN jsonb_build_object('plazas',jsonb_build_object(
      'llamada','simultanea','respuesta_horas',24,'tras_renuncia','siguiente_en_orden'))
     ELSE '{}'::jsonb END AS p
) politica;

-- Se aísla la nueva guarda. B71 se prueba sin desactivar su trigger en su
-- suite de publicación/replay. B75 continúa habilitada en esta transacción.
ALTER TABLE vec_bolsa_llamamientos.oferta_publicada DISABLE TRIGGER oferta_politica_b47;

DO $prueba$
DECLARE v integer; rechazos integer:=0;
BEGIN
 FOR v IN 1..3 LOOP
  BEGIN
   INSERT INTO vec_bolsa_llamamientos.oferta_publicada
    (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
     publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
   VALUES (
    'oferta:'||encode(sha256(convert_to('b75-oferta-'||v,'UTF8')),'hex'),
    'recibo:oferta:'||encode(sha256(convert_to('b75-oferta-'||v,'UTF8')),'hex'),
    'bolsa:b75:sintetica','per_actoractoractoractoractor','clave-oferta-b75-'||v,
    '{"categoria":"Auxiliar administrativo","centro":"Residencia Sierra","fecha_inicio":"2026-10-10","descripcion":"Sustitución"}'::jsonb,
    jsonb_build_object('politica_version',v),now(),now()+interval '2 days',repeat('a',64),
    'decision:b75:oferta:'||v);
   IF v=1 THEN RAISE EXCEPTION 'B75: se insertó una oferta nueva con segundo plazo'; END IF;
  EXCEPTION WHEN sqlstate '22023' THEN
   IF v<>1 OR position('B75: clave=confirmacion_oferta_nueva' IN SQLERRM)=0 THEN RAISE; END IF;
   rechazos:=rechazos+1;
  END;
 END LOOP;
 IF rechazos<>1 OR
    (SELECT count(*) FROM vec_bolsa_llamamientos.oferta_publicada WHERE bolsa_ref='bolsa:b75:sintetica')<>2 THEN
  RAISE EXCEPTION 'B75: clave=resultado esperado=un_rechazo_dos_altas actual=distinto';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.oferta_publicada
                WHERE bolsa_ref='bolsa:b75:sintetica' AND plazo->>'politica_version'='3') THEN
  RAISE EXCEPTION 'B75: la política histórica sin plazas debe conservar su adjudicación definitiva';
 END IF;
END $prueba$;

ALTER TABLE vec_bolsa_llamamientos.oferta_publicada ENABLE TRIGGER oferta_politica_b47;
ROLLBACK;
