\set ON_ERROR_STOP on
-- Vectores compartidos con los tests Go. Solo contiene datos sintéticos.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL standard_conforming_strings=off;

DO $prueba$
DECLARE
  v_ambitos jsonb:=pg_catalog.jsonb_build_object(
    'orden_z','segundo','orden_a','primero');
  v_atributos jsonb:=pg_catalog.jsonb_build_object(
    'utf8_nfc','áéíóú ñ 😀',
    'separadores','línea'||pg_catalog.chr(8232)||
      'párrafo'||pg_catalog.chr(8233)||'fin',
    'html','&<>',
    'comillas_barra',E'dijo "sí" \\ fin');
  v_preimagen text;
  v_huella text;
  v_mutante jsonb;
  v_base jsonb:=$json$
{
  "referencia":"propuesta-cobertura:sha256:d17048b801f5bbacf56589d314b22a168a9427f058c79e27e610d1abf69e19af",
  "huella_sha256":"d17048b801f5bbacf56589d314b22a168a9427f058c79e27e610d1abf69e19af",
  "canon":{
    "dominio":"vec.dipgra.contratacion-temporal.propuesta-decision-cobertura",
    "version_esquema":1,
    "algoritmo":"sha-256"
  },
  "organizacion_ref":"organizacion_diputacion_granada",
  "expediente_ref":"expediente_temporal_configurable_01",
  "version_expediente":3,
  "analisis_ref":"analisis_rrhh_configurable_01",
  "analisis_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "preparacion_evidencias_ref":"preparacion_evidencias_configurable_01",
  "preparacion_evidencias_huella_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
  "catalogo":{
    "referencia":"catalogo_vias_configurables_01",
    "version":17,
    "huella_sha256":"949d6d0589b34b14142b8d1876b1a5dab6fdc3cdd744c0664ece7d1fa7cd82b6"
  },
  "politica":{
    "referencia":"politica_decision_configurable_01",
    "version":4,
    "huella_sha256":"6cca178dd48d3b430d37ef3860d4b823b569f44a1b826448fbb25f4e93fa9b2b"
  },
  "finalidad_clave":"gestionar_cobertura_temporal",
  "finalidad_ref":"finalidad:contratacion-temporal:cobertura",
  "categoria_ref":"categoria_configurable_01",
  "periodo":{
    "inicio":"2026-08-01T00:00:00Z",
    "fin":"2027-03-31T00:00:00Z"
  },
  "generada_en":"2026-07-25T08:01:00Z",
  "valida_hasta":"2026-07-25T08:10:00Z",
  "estado":"viable",
  "via_propuesta":"via_futura_configurable",
  "resultados":[
    {
      "clave":"hecho_alternativo",
      "evidencias":[{
        "resultado":"afirmativa",
        "fuente_ref":"fuente_comprobacion_configurable_11",
        "recibo_ref":"recibo_comprobacion_configurable_11",
        "evaluada_en":"2026-07-25T08:00:30Z"
      }]
    },
    {
      "clave":"hecho_compartido",
      "evidencias":[{
        "resultado":"afirmativa",
        "fuente_ref":"fuente_comprobacion_configurable_10",
        "recibo_ref":"recibo_comprobacion_configurable_10",
        "evaluada_en":"2026-07-25T08:00:30Z"
      }]
    },
    {
      "clave":"hecho_futuro",
      "evidencias":[{
        "resultado":"no_consta",
        "fuente_ref":"fuente_comprobacion_configurable_12",
        "recibo_ref":"recibo_comprobacion_configurable_12",
        "evaluada_en":"2026-07-25T08:00:30Z"
      }]
    }
  ],
  "evaluaciones":[
    {
      "via_clave":"via_futura_configurable",
      "prioridad":10,
      "estado":"viable",
      "ausencias_admitidas":["hecho_futuro"]
    },
    {
      "via_clave":"via_alternativa_configurable",
      "prioridad":20,
      "estado":"viable"
    }
  ]
}
$json$::jsonb;
  v_borde jsonb;
  v_reordenada jsonb;
  v_semantica text;
  v_semantica_mutante text;
  v_material bytea;
  v_contexto bytea;
  v_exacta text;
  v_lista jsonb;
  v_recurso jsonb;
BEGIN
  v_preimagen:='{"ambitos":'||
    vec_contratacion_temporal.o404e_mapa_json_go_v1(v_ambitos)||
    ',"atributos":'||
    vec_contratacion_temporal.o404e_mapa_json_go_v1(v_atributos)||'}';
  v_huella:=pg_catalog.encode(
    pg_catalog.sha256(pg_catalog.convert_to(v_preimagen,'UTF8')),'hex');
  IF current_setting('standard_conforming_strings')<>'off'
     OR v_preimagen<>$esperada${"ambitos":{"orden_a":"primero","orden_z":"segundo"},"atributos":{"comillas_barra":"dijo \"sí\" \\ fin","html":"\u0026\u003c\u003e","separadores":"línea\u2028párrafo\u2029fin","utf8_nfc":"áéíóú ñ 😀"}}$esperada$
     OR v_huella<>
       '5dd8dc79912e15e6540f4fdf03b88b1783182188f64ecd74eb0f13141cb2f603'
  THEN
    RAISE EXCEPTION 'vector RecursoAutorizable SQL/Go divergente: %',v_huella;
  END IF;

  -- JSONB entrega otro orden físico, pero el canon debe permanecer estable.
  v_preimagen:='{"ambitos":'||
    vec_contratacion_temporal.o404e_mapa_json_go_v1(
      pg_catalog.jsonb_build_object(
        'orden_a','primero','orden_z','segundo'))||
    ',"atributos":'||
    vec_contratacion_temporal.o404e_mapa_json_go_v1(
      pg_catalog.jsonb_build_object(
        'comillas_barra',E'dijo "sí" \\ fin',
        'html','&<>',
        'separadores','línea'||pg_catalog.chr(8232)||
          'párrafo'||pg_catalog.chr(8233)||'fin',
        'utf8_nfc','áéíóú ñ 😀'))||'}';
  IF pg_catalog.encode(
       pg_catalog.sha256(pg_catalog.convert_to(v_preimagen,'UTF8')),'hex'
     )<>v_huella THEN
    RAISE EXCEPTION 'el orden de mapas alteró el vector RecursoAutorizable';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_atributos,'{utf8_nfc}',
    pg_catalog.to_jsonb('a'||pg_catalog.chr(769)||'éíóú ñ 😀'));
  IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
       '{"ambitos":'||
       vec_contratacion_temporal.o404e_mapa_json_go_v1(v_ambitos)||
       ',"atributos":'||
       vec_contratacion_temporal.o404e_mapa_json_go_v1(v_mutante)||'}',
       'UTF8')),'hex')=v_huella THEN
    RAISE EXCEPTION 'el canon normalizó Unicode de forma implícita';
  END IF;
  IF vec_contratacion_temporal.o404e_mapa_json_go_v1(
       pg_catalog.jsonb_set(v_atributos,'{html}','1'::jsonb)
     ) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon aceptó un valor de mapa no textual';
  END IF;

  -- El helper CT exige forma cerrada del recurso. El fixture usa los bordes
  -- válidos de referencia (3/160) y de versión (1/2^53-1).
  v_recurso:=pg_catalog.jsonb_build_object(
    'cabecera',pg_catalog.jsonb_build_object(
      'organizacion_ref','organizacion_diputacion_granada',
      'expediente_ref','expediente_temporal_configurable_01',
      'version_expediente',9007199254740991::numeric,
      'reserva_ref','reserva_configurable_01',
      'revision_cercado',7),
    'denegacion',pg_catalog.jsonb_build_object(
      'accion_vec','contratacion_temporal.cobertura.decidir',
      'ambitos',pg_catalog.jsonb_build_object(
        'organizacion_ref','organizacion_diputacion_granada',
        'unidad_ejecutora_ref','unidad_ejecutora_configurable_01'),
      'atributos',pg_catalog.jsonb_build_object(
        'accion','contratacion_temporal.cobertura.decidir',
        'analisis_huella_sha256',pg_catalog.repeat('a',64),
        'analisis_ref','a01',
        'catalogo_huella_sha256',pg_catalog.repeat('b',64),
        'catalogo_ref',pg_catalog.repeat('c',160),
        'catalogo_version','1',
        'expediente_ref','expediente_temporal_configurable_01',
        'politica_actuacion_huella_sha256',pg_catalog.repeat('d',64),
        'politica_actuacion_ref','politica_actuacion_configurable_01',
        'politica_actuacion_version','32768',
        'politica_huella_sha256',pg_catalog.repeat('e',64),
        'politica_ref','politica_configurable_01',
        'politica_version','9007199254740991',
        'preparacion_evidencias_huella_sha256',pg_catalog.repeat('f',64),
        'preparacion_evidencias_ref','preparacion_configurable_01',
        'propuesta_huella_sha256',pg_catalog.repeat('1',64),
        'propuesta_ref',
          'propuesta-cobertura:sha256:'||pg_catalog.repeat('1',64),
        'propuesta_semantica_huella_sha256',pg_catalog.repeat('2',64),
        'propuesta_semantica_ref',
          'propuesta-cobertura-semantica:sha256:'||
            pg_catalog.repeat('2',64),
        'reserva_ref','reserva_configurable_01',
        'revision_cercado','7',
        'tipo_operacion','inicial',
        'version_expediente_esperada','9007199254740991',
        'via_elegida','via_futura_configurable')));
  v_contexto:=vec_contratacion_temporal
    .o404e_contexto_recurso_denegacion_v1(v_recurso);
  IF v_contexto IS NULL THEN
    RAISE EXCEPTION 'el helper CT rechazó bordes válidos de refs/versiones';
  END IF;
  v_mutante:=pg_catalog.jsonb_set(
    v_recurso,'{denegacion,atributos}',
    (v_recurso#>'{denegacion,atributos}')-'analisis_ref');
  IF vec_contratacion_temporal
       .o404e_contexto_recurso_denegacion_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el helper CT aceptó un atributo obligatorio ausente';
  END IF;
  v_mutante:=pg_catalog.jsonb_set(
    v_recurso,'{denegacion,atributos}',
    (v_recurso#>'{denegacion,atributos}')||
      pg_catalog.jsonb_build_object('campo_extra','rechazar'));
  IF vec_contratacion_temporal
       .o404e_contexto_recurso_denegacion_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el helper CT aceptó un atributo extra';
  END IF;
  FOREACH v_mutante IN ARRAY ARRAY[
    pg_catalog.jsonb_set(
      v_recurso,'{denegacion,atributos,catalogo_version}',
      pg_catalog.to_jsonb('0'::text)),
    pg_catalog.jsonb_set(
      v_recurso,'{denegacion,atributos,politica_version}',
      pg_catalog.to_jsonb('9007199254740992'::text)),
    pg_catalog.jsonb_set(
      v_recurso,'{denegacion,atributos,analisis_ref}',
      pg_catalog.to_jsonb('ab'::text)),
    pg_catalog.jsonb_set(
      v_recurso,'{denegacion,atributos,catalogo_ref}',
      pg_catalog.to_jsonb(pg_catalog.repeat('c',161)))
  ] LOOP
    IF vec_contratacion_temporal
         .o404e_contexto_recurso_denegacion_v1(v_mutante) IS NOT NULL THEN
      RAISE EXCEPTION 'el helper CT aceptó un borde inválido de ref/versión';
    END IF;
  END LOOP;

  IF vec_contratacion_temporal.o404e_propuesta_cobertura_exacta_v1(v_base)
       IS NOT TRUE
     OR v_base#>'{evaluaciones,1}'?'ausencias_admitidas'
     OR v_base#>'{evaluaciones,1}'?'conflictos' THEN
    RAISE EXCEPTION 'fixture real omitempty inválido';
  END IF;
  v_semantica:=
    vec_contratacion_temporal.o404e_semantica_propuesta_v1(v_base);
  IF v_semantica<>
       'a9f78ff73feb40d6a0ef0de9506ea23668c67b50fe2f1a685631bc63586a1806'
  THEN
    RAISE EXCEPTION 'vector semántico SQL/Go divergente: %',v_semantica;
  END IF;

  v_borde:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(
      v_base,'{evaluaciones,0,prioridad}','32768'::jsonb),
    '{evaluaciones,1,prioridad}','65535'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_borde);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_borde:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_borde,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF v_exacta<>
       '8d50612104114047b20cfdf76e674dc4dc4996537e114ff8330c6b8836250171'
     OR vec_contratacion_temporal
          .o404e_propuesta_cobertura_exacta_v1(v_borde) IS NOT TRUE
     OR vec_contratacion_temporal.o404e_semantica_propuesta_v1(v_borde)<>
       '8f0cc8725238cbc355f4a1bb651e163938fe4fe36d7554554092602de0fd223b'
  THEN
    RAISE EXCEPTION 'vector uint16 SQL/Go divergente';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,prioridad}','32767'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_propuesta_cobertura_exacta_v1(v_mutante) IS NOT TRUE
     OR vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NULL THEN
    RAISE EXCEPTION 'el canon rechazó la prioridad válida 32767';
  END IF;

  SELECT pg_catalog.jsonb_agg(value ORDER BY ordinalidad DESC)
    INTO v_lista
    FROM pg_catalog.jsonb_array_elements(v_base->'resultados')
      WITH ORDINALITY e(value,ordinalidad);
  v_reordenada:=pg_catalog.jsonb_set(v_base,'{resultados}',v_lista);
  SELECT pg_catalog.jsonb_agg(value ORDER BY ordinalidad DESC)
    INTO v_lista
    FROM pg_catalog.jsonb_array_elements(v_base->'evaluaciones')
      WITH ORDINALITY e(value,ordinalidad);
  v_reordenada:=pg_catalog.jsonb_set(
    v_reordenada,'{evaluaciones}',v_lista);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_reordenada);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_reordenada:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_reordenada,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_propuesta_cobertura_exacta_v1(v_reordenada) IS NOT TRUE
     OR vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_reordenada)<>v_semantica THEN
    RAISE EXCEPTION 'la reordenación alteró la semántica';
  END IF;

  -- Las listas internas también son conjuntos canónicos, no secuencias.
  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,resultados_omitidos}',
    '["clave_lista_a","clave_lista_b"]'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  v_semantica_mutante:=vec_contratacion_temporal
    .o404e_semantica_propuesta_v1(v_mutante);
  v_reordenada:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,resultados_omitidos}',
    '["clave_lista_b","clave_lista_a"]'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_reordenada);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_reordenada:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_reordenada,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF v_semantica_mutante IS NULL
     OR vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_reordenada)<>v_semantica_mutante THEN
    RAISE EXCEPTION 'reordenar una lista interna alteró su huella semántica';
  END IF;

  -- Dos evidencias funcionalmente iguales se deduplican por resultado.
  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{resultados,0,evidencias}',
    (v_base#>'{resultados,0,evidencias}')||
      pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object(
        'resultado','afirmativa',
        'fuente_ref','fuente_comprobacion_configurable_duplicada',
        'recibo_ref','recibo_comprobacion_configurable_duplicado',
        'evaluada_en','2026-07-25T08:00:31Z')));
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_propuesta_cobertura_exacta_v1(v_mutante) IS NOT TRUE
     OR vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante)<>v_semantica THEN
    RAISE EXCEPTION 'no se deduplicó una evidencia funcional repetida';
  END IF;

  -- El material exacto conserva microsegundos negativos respecto de Unix.
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(
      v_base,'{generada_en}',
      pg_catalog.to_jsonb('1969-12-31T23:59:59.999999Z'::text)),
    '{valida_hasta}',
    pg_catalog.to_jsonb('1970-01-01T00:00:00.000001Z'::text));
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_propuesta_cobertura_exacta_v1(v_mutante) IS NOT TRUE
     OR vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante)<>v_semantica THEN
    RAISE EXCEPTION 'instante pre-epoch/microsegundo divergente';
  END IF;

  -- Una renovación probatoria cambia el canon exacto, no el significado.
  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{preparacion_evidencias_ref}',
    pg_catalog.to_jsonb('preparacion_evidencias_configurable_02'::text));
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante)<>v_semantica THEN
    RAISE EXCEPTION 'una renovación probatoria alteró la semántica';
  END IF;

  -- Un dato decidible sí debe modificar la identidad semántica.
  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{categoria_ref}',
    pg_catalog.to_jsonb('categoria_configurable_02'::text));
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante)=v_semantica THEN
    RAISE EXCEPTION 'un dato decidible no alteró la semántica';
  END IF;

  -- El canon exacto histórico admite estas formas; el semántico debe
  -- rechazarlas para igualar los límites y unicidad del dominio Go.
  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,prioridad}','0'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon semántico aceptó prioridad cero';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,1,prioridad}',
    v_base#>'{evaluaciones,0,prioridad}');
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon semántico aceptó una prioridad duplicada';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,1,via_clave}',
    v_base#>'{evaluaciones,0,via_clave}');
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon semántico aceptó una vía duplicada';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones}','[]'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon semántico aceptó cero evaluaciones';
  END IF;

  SELECT pg_catalog.jsonb_agg(
      pg_catalog.to_jsonb('clave_'||i) ORDER BY i)
    INTO v_lista FROM pg_catalog.generate_series(1,33) i;
  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,resultados_omitidos}',v_lista);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon semántico aceptó 33 claves por vía';
  END IF;

  SELECT pg_catalog.jsonb_agg(
      pg_catalog.to_jsonb('omitida_'||i) ORDER BY i)
    INTO v_lista FROM pg_catalog.generate_series(1,16) i;
  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,resultados_omitidos}',v_lista);
  SELECT pg_catalog.jsonb_agg(
      pg_catalog.to_jsonb('bloqueante_'||i) ORDER BY i)
    INTO v_lista FROM pg_catalog.generate_series(1,16) i;
  v_mutante:=pg_catalog.jsonb_set(
    v_mutante,'{evaluaciones,0,ausencias_bloqueantes}',v_lista);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon aceptó 33 claves repartidas entre listas';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,resultados_omitidos}',
    '["hecho_futuro"]'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon aceptó una clave duplicada entre listas';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{evaluaciones,0,resultados_omitidos}',
    '["hecho_repetido","hecho_repetido"]'::jsonb);
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon aceptó una clave duplicada en una lista';
  END IF;

  v_mutante:=pg_catalog.jsonb_set(
    v_base,'{resultados}',(v_base->'resultados')||
      pg_catalog.jsonb_build_array(v_base#>'{resultados,0}'));
  v_material:=vec_contratacion_temporal
    .o404e_material_propuesta_cobertura_v1(v_mutante);
  v_exacta:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
  v_mutante:=pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(v_mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(v_exacta)),
    '{referencia}',pg_catalog.to_jsonb(
      'propuesta-cobertura:sha256:'||v_exacta));
  IF vec_contratacion_temporal
       .o404e_semantica_propuesta_v1(v_mutante) IS NOT NULL THEN
    RAISE EXCEPTION 'el canon semántico aceptó un resultado repetido';
  END IF;
END
$prueba$;
ROLLBACK;
