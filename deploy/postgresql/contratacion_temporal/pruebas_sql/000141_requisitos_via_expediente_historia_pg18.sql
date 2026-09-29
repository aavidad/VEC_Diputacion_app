\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
DO $prueba$
DECLARE
 v_p jsonb;
 v_hash text;
BEGIN
 IF (SELECT count(*) FROM vec_contratacion_temporal.gobi_o404b_evento) <> 2
    OR (SELECT count(*) FROM vec_contratacion_temporal.gobi_o404b_catalogo) <> 2
    OR (SELECT ultima_secuencia FROM vec_contratacion_temporal.gobi_o404b_checkpoint WHERE control) <> 2 THEN
   RAISE EXCEPTION 'CT141 publicar/replay duplicó o perdió historia';
 END IF;
 SELECT publicacion_json INTO STRICT v_p
 FROM vec_contratacion_temporal.gobi_o404b_catalogo
 WHERE version=2 AND referencia='catalogo:o404e:golden';
 v_hash:=pg_catalog.encode(pg_catalog.sha256(
   vec_contratacion_temporal.gobi_o404b_material_catalogo(v_p)),'hex');
 IF v_hash IS DISTINCT FROM v_p->>'huella_sha256' THEN
   RAISE EXCEPTION 'CT141 CHECK material publicado no coincide';
 END IF;
 BEGIN
   INSERT INTO vec_contratacion_temporal.gobi_o404b_catalogo (
     referencia,version,huella_sha256,publicacion_json,publicado_en,
     vigente_desde,vigente_hasta,evento_ref,secuencia
   ) VALUES(
     v_p->>'referencia',3,v_p->>'huella_sha256',
     jsonb_set(v_p,'{version}','3'::jsonb),
     (v_p->>'publicado_en')::timestamptz,
     (v_p#>>'{vigencia,desde}')::timestamptz,
     (v_p#>>'{vigencia,hasta}')::timestamptz,
     'evento_gobi_o404b_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeee2',2
   );
   RAISE EXCEPTION 'CT141 CHECK aceptó huella V2 adulterada';
 EXCEPTION WHEN check_violation THEN
   NULL;
 END;
 BEGIN
   UPDATE vec_contratacion_temporal.gobi_o404b_catalogo
   SET version=3 WHERE version=2;
   RAISE EXCEPTION 'CT141 historia permite UPDATE';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN
   NULL;
 END;
END
$prueba$;
RESET ROLE;
