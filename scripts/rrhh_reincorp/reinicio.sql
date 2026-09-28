\set ON_ERROR_STOP on
DO $test$
DECLARE c record; b record; o record;
BEGIN
 SELECT * INTO c FROM vec_contratacion_temporal.reincorporacion_titular_v1
  WHERE evento_ref='evento:ct130:ensayo';
 SELECT * INTO b FROM vec_bolsa_llamamientos.reincorporacion_titular_ct
  WHERE evento_ref='evento:ct130:ensayo';
 IF c.recibo_ref IS DISTINCT FROM 'recibo:ct130:ensayo'
    OR c.recibo_json->>'recibo_ref' IS DISTINCT FROM c.recibo_ref
    OR c.recibo_json->>'cese_evento_ref' IS DISTINCT FROM c.cese_evento_ref
    OR b.recibo_ct_ref IS DISTINCT FROM c.recibo_ref
    OR b.cese_evento_ref IS DISTINCT FROM c.cese_evento_ref
    OR b.cese_recibo_ref IS DISTINCT FROM c.cese_recibo_ref
    OR b.fecha_efectiva IS DISTINCT FROM c.fecha_efectiva THEN
  RAISE EXCEPTION 'recuperación CT130/B46 divergente'; END IF;
 SELECT * INTO o FROM vec_contratacion_temporal.outbox_expediente_integral
  WHERE evento_ref=c.evento_ref;
 IF o.tipo_evento IS DISTINCT FROM 'ct.reincorporacion_titular.v1'
    OR o.version_expediente IS DISTINCT FROM c.version_esperada+1
    OR o.payload_huella_sha256 IS DISTINCT FROM encode(sha256(o.payload_canonico),'hex')
    OR (SELECT count(*) FROM vec_contratacion_temporal.reincorporacion_titular_v1 WHERE expediente_ref=c.expediente_ref)<>1
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.reincorporacion_titular_ct WHERE evento_ref=c.evento_ref)<>1 THEN
  RAISE EXCEPTION 'historia duplicada o outbox divergente'; END IF;
END $test$;
SELECT 'RRHH_REINCORP_REINICIO_OK';
