\set ON_ERROR_STOP on
SET ROLE vec_contratacion_temporal_propietario;
DO $prueba$
DECLARE
  v_j jsonb;
  v_m bytea;
BEGIN
  SELECT j,m INTO STRICT v_j,v_m FROM vec_ct141_prueba.v1;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(v_j)
       IS DISTINCT FROM v_m THEN
    RAISE EXCEPTION 'CT141 cambió preimagen V1';
  END IF;
  v_j:=jsonb_set(v_j,'{canon,version_esquema}','2'::jsonb);
  v_j:=jsonb_set(v_j,'{vias,0,documentos}',
    '[{"clave":"doc_ejemplo","orden":3,"clave_i18n":"ct.via.doc_ejemplo"}]'::jsonb);
  v_j:=jsonb_set(v_j,'{vias,0,datos}',
    '[{"clave":"dato_ejemplo","orden":7,"clave_i18n":"ct.via.dato_ejemplo"}]'::jsonb);
  v_j:=jsonb_set(v_j,'{es_ejemplo}','true'::jsonb);
  v_m:=vec_contratacion_temporal.gobi_o404b_material_catalogo(v_j);
  IF v_m IS NULL OR pg_catalog.get_byte(v_m,0) <> 0 THEN
    RAISE EXCEPTION 'CT141 V2 no genera material';
  END IF;
  INSERT INTO vec_ct141_prueba.v1(j,m) VALUES(v_j,v_m);
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{es_ejemplo}','false'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta falso explícito no canónico';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,documentos}','[]'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta array vacío';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,documentos,0,orden}','0'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta orden cero';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,documentos,0,clave}','"A"'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta clave no canónica';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,documentos,0,clave}','true'::jsonb)
     ) IS NOT NULL OR
     vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,datos,0,clave}','false'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta clave booleana';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{referencia}','123'::jsonb)
     ) IS NOT NULL OR
     vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{procedencia_ref}','123'::jsonb)
     ) IS NOT NULL OR
     vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,comprobaciones,0,procedencia,clave}',
                 'true'::jsonb)
     ) IS NOT NULL OR
     vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vigencia,hasta}','null'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta tipo JSON ajeno';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j - 'es_ejemplo','{ES_EJEMPLO}','true'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta ES_EJEMPLO';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j #- '{vias,0,documentos}',
                 '{vias,0,DOCUMENTOS}',
                 v_j #> '{vias,0,documentos}')
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta DOCUMENTOS';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j #- '{vias,0,documentos,0,clave_i18n}',
                 '{vias,0,documentos,0,CLAVE_I18N}',
                 v_j #> '{vias,0,documentos,0,clave_i18n}')
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta CLAVE_I18N';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,documentos}',
         jsonb_build_array(
           v_j #> '{vias,0,documentos,0}',
           jsonb_set(v_j #> '{vias,0,documentos,0}',
                     '{orden}','4'::jsonb)))
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta clave documental duplicada';
  END IF;
  IF vec_contratacion_temporal.gobi_o404b_material_catalogo(
       jsonb_set(v_j,'{vias,0,datos}',
         jsonb_build_array(
           v_j #> '{vias,0,datos,0}',
           jsonb_set(v_j #> '{vias,0,datos,0}',
                     '{orden}','8'::jsonb)))
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'CT141 acepta clave de dato duplicada';
  END IF;
END
$prueba$;
DO $vector$
DECLARE
 v_j jsonb := $json${"referencia":"catalogo_vector_v2","version":1,"huella_sha256":"ed4cb260730e1f95bf235e564f070d3bc99203d228c2ae8c01922a561ade675c","canon":{"dominio":"vec.dipgra.contratacion-temporal.catalogo-vias-cobertura","version_esquema":2,"algoritmo":"sha-256"},"publicado_en":"2026-09-29T00:00:00Z","vigencia":{"desde":"2026-09-29T00:00:00Z","hasta":"0001-01-01T00:00:00Z"},"procedencia_ref":"procedencia_vector_v2","es_ejemplo":true,"vias":[{"clave":"bolsa_vigente","orden":1,"comprobaciones":[{"clave":"existe_bolsa_vigente","orden":1,"obligatoria":true,"procedencia":{"clave":"bolsa","definicion_fuente_ref":"fuente_vector_v2"}}],"documentos":[{"clave":"informe_necesidad","orden":1,"clave_i18n":"ct.cobertura.doc.informe_necesidad"}],"datos":[{"clave":"categoria","orden":1,"clave_i18n":"ct.cobertura.dato.categoria"}]}]}$json$::jsonb;
 v_hash text;
BEGIN
 IF pg_catalog.encode(vec_contratacion_temporal.gobi_o404b_material_catalogo(v_j),'hex') IS DISTINCT FROM
    '000000387665632e6469706772612e636f6e747261746163696f6e2d74656d706f72616c2e636174616c6f676f2d766961732d636f626572747572610002000000077368612d32353600000012636174616c6f676f5f766563746f725f7632000000000000000100065c93dd1ee00000065c93dd1ee000000000001570726f636564656e6369615f766563746f725f763201000000010000000d626f6c73615f766967656e7465000100000001000000146578697374655f626f6c73615f766967656e746500010100000005626f6c7361000000106675656e74655f766563746f725f76320000000100000011696e666f726d655f6e656365736964616400010000002263742e636f626572747572612e646f632e696e666f726d655f6e6563657369646164000000010000000963617465676f72696100010000001b63742e636f626572747572612e6461746f2e63617465676f726961' THEN
   RAISE EXCEPTION 'CT141 material Go/SQL divergente';
 END IF;
 v_hash:=pg_catalog.encode(pg_catalog.sha256(
   vec_contratacion_temporal.gobi_o404b_material_catalogo(v_j)),'hex');
 IF v_hash IS DISTINCT FROM v_j->>'huella_sha256' THEN
   RAISE EXCEPTION 'CT141 Go/SQL divergentes: %',v_hash;
 END IF;
END
$vector$;
GRANT USAGE ON SCHEMA vec_ct141_prueba TO vec_ct141_gob;

CREATE FUNCTION vec_ct141_prueba.publicar_v1()
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path=pg_catalog
SET timezone='UTC'
AS $$
DECLARE
  v_ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp());
  v_desde text:=vec_contratacion_temporal.texto_instante_utc_go_v2(
    (v_ahora-interval '1 minute')::text);
  v_hasta text:=vec_contratacion_temporal.texto_instante_utc_go_v2(
    (v_ahora+interval '2 hours')::text);
  v_publicada text:=vec_contratacion_temporal.texto_instante_utc_go_v2(
    (v_ahora-interval '2 minutes')::text);
  v_catalogo jsonb;
  v_politica jsonb;
  v_actuacion jsonb;
  v_publicacion jsonb;
  v_secuencia bigint;
BEGIN
  IF session_user<>'vec_ct141_gob' THEN
    RAISE EXCEPTION 'publicación de gobierno fuera del fixture';
  END IF;
  v_catalogo:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.catalogo-vias-cobertura',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','catalogo:o404e:golden','version',1,
    'huella_sha256',repeat('f',64),'publicado_en',v_publicada,
    'vigencia',jsonb_build_object('desde',v_desde,'hasta',v_hasta),
    'procedencia_ref','procedencia:o404e:golden',
    'vias',jsonb_build_array(jsonb_build_object(
      'clave','via_o404e','orden',1,
      'comprobaciones',jsonb_build_array(jsonb_build_object(
        'clave','comprobacion_o404e','orden',1,'obligatoria',true,
        'procedencia',jsonb_build_object(
          'clave','fuente_o404e',
          'definicion_fuente_ref','fuente:o404e:golden'))))));
  v_catalogo:=jsonb_set(v_catalogo,'{huella_sha256}',to_jsonb(encode(
    sha256(vec_contratacion_temporal.gobi_o404b_material_catalogo(v_catalogo)),
    'hex')));
  v_politica:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.politica-decision-cobertura',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','politica:o404e:golden','version',1,
    'huella_sha256',repeat('f',64),
    'catalogo',jsonb_build_object(
      'referencia',v_catalogo->>'referencia','version',v_catalogo->'version',
      'huella_sha256',v_catalogo->>'huella_sha256'),
    'organizacion_ref','organizacion:o404e:golden',
    'finalidad_clave','gestionar_cobertura_temporal',
    'finalidad_ref','finalidad:o404e:golden',
    'publicada_en',v_publicada,
    'vigencia',jsonb_build_object('desde',v_desde,'hasta',v_hasta),
    'procedencia_ref','procedencia:o404e:golden',
    'vias',jsonb_build_array(jsonb_build_object(
      'via_clave','via_o404e','prioridad',1,
      'comprobaciones',jsonb_build_array(jsonb_build_object(
        'clave','comprobacion_o404e',
        'resultados_habilitantes',jsonb_build_array('afirmativa'),
        'tratamiento_ausencia','bloquea')))));
  v_politica:=jsonb_set(v_politica,'{huella_sha256}',to_jsonb(encode(
    sha256(vec_contratacion_temporal.gobi_o404b_material_politica(v_politica)),
    'hex')));
  v_actuacion:=jsonb_build_object(
    'canon',jsonb_build_object(
      'dominio','vec.dipgra.contratacion-temporal.politica-actuacion-cobertura',
      'version_esquema',1,'algoritmo','sha-256'),
    'referencia','actuacion:o404e:gobierno:golden','version',1,
    'huella_sha256',repeat('f',64),
    'organizacion_ref','organizacion:o404e:golden',
    'accion','contratacion_temporal.cobertura.decidir',
    'catalogo',v_politica->'catalogo',
    'politica',jsonb_build_object(
      'referencia',v_politica->>'referencia','version',v_politica->'version',
      'huella_sha256',v_politica->>'huella_sha256'),
    'finalidad_contratacion_clave','gestionar_cobertura_temporal',
    'finalidad_contratacion_ref','finalidad:o404e:golden',
    'finalidad_autorizacion_vec','gestion',
    'unidad_ejecutora_ref','unidad:o404e:golden',
    'fase_destino','fase_cobertura','estado_destino','en_curso',
    'motivo_autorizacion_decidir',jsonb_build_object(
      'catalogo_id','motivos_v3','catalogo_version',1,
      'catalogo_huella_sha256',repeat('9',64),
      'entrada_clave','motivo_33333333333333333333333333333333'),
    'motivo_autorizacion_rectificar',jsonb_build_object(
      'catalogo_id','motivos_v3','catalogo_version',1,
      'catalogo_huella_sha256',repeat('9',64),
      'entrada_clave','motivo_44444444444444444444444444444444'),
    'publicada_en',v_publicada,
    'vigencia',jsonb_build_object('desde',v_desde,'hasta',v_hasta));
  v_actuacion:=jsonb_set(v_actuacion,'{huella_sha256}',to_jsonb(encode(
    sha256(vec_contratacion_temporal.gobi_o404b_material_actuacion(v_actuacion)),
    'hex')));
  SELECT ultima_secuencia+1 INTO STRICT v_secuencia
    FROM vec_contratacion_temporal.gobi_o404b_checkpoint WHERE control;
  v_publicacion:=jsonb_build_object(
    'esquema','vec.contratacion-temporal.gobierno-cobertura.o4-04b.v1',
    'secuencia',v_secuencia,'evento_ref','evento_gobi_o404b_'||
      lpad(to_hex(v_secuencia),32,'e'),
    'catalogo',v_catalogo,'politica',v_politica,'actuacion',v_actuacion);
  PERFORM * FROM vec_contratacion_temporal.gobi_o404b_publicar(v_publicacion);
  RETURN true;
END
$$;
REVOKE ALL ON FUNCTION vec_ct141_prueba.publicar_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_ct141_prueba.publicar_v1()
  TO vec_ct141_gob;


RESET ROLE;
