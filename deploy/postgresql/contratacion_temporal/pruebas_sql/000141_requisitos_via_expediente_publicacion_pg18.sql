\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
CREATE FUNCTION vec_ct141_prueba.publicar_v2()
RETURNS text
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog
SET timezone='UTC'
AS $funcion$
DECLARE
 v_p jsonb;
 v_c jsonb;
 v_d jsonb;
 v_a jsonb;
 v_resultado text;
BEGIN
 IF session_user <> 'vec_ct141_gob' THEN
   RAISE EXCEPTION 'fixture fuera de rol de gobierno';
 END IF;
 SELECT contenido_evento INTO STRICT v_p
 FROM vec_contratacion_temporal.gobi_o404b_evento
 WHERE secuencia=1;
 v_c:=v_p->'catalogo';
 v_c:=jsonb_set(v_c,'{canon,version_esquema}','2'::jsonb);
 v_c:=jsonb_set(v_c,'{version}','2'::jsonb);
 v_c:=jsonb_set(v_c,'{es_ejemplo}','true'::jsonb);
 v_c:=jsonb_set(v_c,'{vias,0,documentos}',
   '[{"clave":"informe_necesidad","orden":1,"clave_i18n":"ct.cobertura.doc.informe_necesidad"}]'::jsonb);
 v_c:=jsonb_set(v_c,'{vias,0,datos}',
   '[{"clave":"categoria","orden":1,"clave_i18n":"ct.cobertura.dato.categoria"}]'::jsonb);
 v_c:=jsonb_set(v_c,'{huella_sha256}',to_jsonb(pg_catalog.encode(
   pg_catalog.sha256(
     vec_contratacion_temporal.gobi_o404b_material_catalogo(v_c)
   ),'hex')));
 v_d:=v_p->'politica';
 v_d:=jsonb_set(v_d,'{version}','2'::jsonb);
 v_d:=jsonb_set(v_d,'{catalogo}',jsonb_build_object(
   'referencia',v_c->>'referencia','version',v_c->'version',
   'huella_sha256',v_c->>'huella_sha256'));
 v_d:=jsonb_set(v_d,'{huella_sha256}',to_jsonb(pg_catalog.encode(
   pg_catalog.sha256(
     vec_contratacion_temporal.gobi_o404b_material_politica(v_d)
   ),'hex')));
 v_a:=v_p->'actuacion';
 v_a:=jsonb_set(v_a,'{version}','2'::jsonb);
 v_a:=jsonb_set(v_a,'{catalogo}',v_d->'catalogo');
 v_a:=jsonb_set(v_a,'{politica}',jsonb_build_object(
   'referencia',v_d->>'referencia','version',v_d->'version',
   'huella_sha256',v_d->>'huella_sha256'));
 v_a:=jsonb_set(v_a,'{huella_sha256}',to_jsonb(pg_catalog.encode(
   pg_catalog.sha256(
     vec_contratacion_temporal.gobi_o404b_material_actuacion(v_a)
   ),'hex')));
 v_p:=jsonb_build_object(
   'esquema','vec.contratacion-temporal.gobierno-cobertura.o4-04b.v1',
   'secuencia',2,
   'evento_ref','evento_gobi_o404b_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeee2',
   'catalogo',v_c,'politica',v_d,'actuacion',v_a);
 SELECT resultado INTO STRICT v_resultado
 FROM vec_contratacion_temporal.gobi_o404b_publicar(v_p);
 RETURN v_resultado;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_ct141_prueba.publicar_v2() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_ct141_prueba.publicar_v2() TO vec_ct141_gob;
RESET ROLE;
