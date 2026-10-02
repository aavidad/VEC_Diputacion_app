\set ON_ERROR_STOP on
-- CT166: sintéticos V1/V2; no escribe historia de expedientes.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $prueba$
DECLARE
 p_legacy jsonb:='{"inicio":"2026-10-02T13:15:00Z","fin":"2026-10-03T13:15:00Z"}';
 p_fecha jsonb:='{"inicio":"2026-10-02T00:00:00Z","fin":"2026-10-31T00:00:00Z","politica_fin":{"regla_ref":"regla:modalidad:acumulacion","catalogo_version":2,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fecha_fin":"opcional","causa_fin":"fin_tareas"}}';
 p_causa jsonb:='{"inicio":"2026-10-02T00:00:00Z","causa_fin":"reincorporacion_titular","politica_fin":{"regla_ref":"regla:modalidad:sustitucion","catalogo_version":2,"catalogo_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","fecha_fin":"no_aplica","causa_fin":"reincorporacion_titular"}}';
 e jsonb;
 e_legacy jsonb;
 m bytea;
 propuesta jsonb;
 mutante jsonb;
 h text;
BEGIN
 IF vec_contratacion_temporal.ct166_periodo_valido_v1(p_legacy) IS NOT TRUE
    OR vec_contratacion_temporal.ct166_periodo_valido_v1(p_fecha) IS NOT TRUE
    OR vec_contratacion_temporal.ct166_periodo_valido_v1(p_causa) IS NOT TRUE
    OR vec_contratacion_temporal.ct166_periodo_valido_v1(
         pg_catalog.jsonb_set(p_causa,'{politica_fin,catalogo_version}','3'::jsonb)) IS NOT TRUE
    OR vec_contratacion_temporal.ct166_periodo_valido_v1(
         pg_catalog.jsonb_set(p_causa,'{politica_fin,causa_fin}','"otra"'::jsonb)) IS NOT FALSE
    OR vec_contratacion_temporal.ct166_periodo_valido_v1(
         p_causa || '{"fin":null}'::jsonb) IS NOT FALSE THEN
   RAISE EXCEPTION 'CT166 forma del periodo incorrecta';
 END IF;
 m := vec_contratacion_temporal.ct166_periodo_material_v1(p_legacy);
 IF m IS DISTINCT FROM
    pg_catalog.int8send(vec_contratacion_temporal.gobi_o404b_microsegundos(
      (p_legacy->>'inicio')::timestamptz)) ||
    pg_catalog.int8send(vec_contratacion_temporal.gobi_o404b_microsegundos(
      (p_legacy->>'fin')::timestamptz)) THEN
   RAISE EXCEPTION 'CT166 cambió los bytes del periodo legacy';
 END IF;
 IF vec_contratacion_temporal.ct166_periodo_material_v1(p_causa) IS NULL
    OR vec_contratacion_temporal.ct166_periodo_material_v1(p_causa)=
       vec_contratacion_temporal.ct166_periodo_material_v1(
          pg_catalog.jsonb_set(p_causa,'{politica_fin,catalogo_version}','3'::jsonb))
    OR vec_contratacion_temporal.ct166_periodo_orden_v1(p_causa) IS NULL THEN
   RAISE EXCEPTION 'CT166 material V2 no liga la política';
 END IF;

 e := pg_catalog.jsonb_build_object(
   'posicion',1,'total',1,'organizacion_ref','org:sintetica',
   'expediente_ref','exp:sintetico','version_expediente',2,
   'peticion_ref','peticion:sintetica','autoridad_ref','bolsa:sintetica',
   'generacion',1,'recibo_respuesta_ref','recibo:sintetico',
   'catalogo_ref','catalogo:sintetico','catalogo_version',1,
   'via_clave','bolsa','comprobacion_clave','bolsa_vigente',
   'comprobacion_resultado','afirmativa','orden_comprobacion',1,
   'comprobacion_obligatoria',true,'procedencia_clave','bolsa',
   'definicion_fuente_ref','fuente:sintetica','categoria_ref','cat:sintetica',
   'periodo_inicio',p_causa->'inicio','periodo_fin','null'::jsonb,
   'causa_fin',p_causa->'causa_fin','politica_fin',p_causa->'politica_fin',
   'solicitada_en','2026-10-01T10:00:00Z',
   'emitida_en','2026-10-01T10:00:01Z',
   'valida_hasta','2026-10-01T10:00:04Z',
   'verificador_ref','verificador:sintetico',
   'publicador_catalogo_ref','publicador:sintetico');
 e := e || pg_catalog.jsonb_build_object(
   'evidencia_huella_sha256',pg_catalog.repeat('a',64),
   'huella_peticion_sha256',pg_catalog.repeat('b',64),
   'huella_resultado_sha256',pg_catalog.repeat('c',64),
   'huella_respuesta_sha256',pg_catalog.repeat('d',64),
   'catalogo_huella_sha256',pg_catalog.repeat('e',64),
   'peticion_canon_hex','aa','resultado_canon_hex','aa',
   'atestacion_canon_hex','aa','confirmacion_tcb_canon_hex','aa',
   'catalogo_canon_hex','aa','verificador_canon_hex','aa',
   'resumen_canon_hex','aa');
 m := vec_contratacion_temporal.o404d_material_evidencia_v1(e);
 IF m IS NULL OR m=vec_contratacion_temporal.o404d_material_evidencia_v1(
     pg_catalog.jsonb_set(e,'{politica_fin,catalogo_version}','3'::jsonb))
    OR vec_contratacion_temporal.o404d_material_evidencia_v1(
     e-'causa_fin') IS NOT NULL THEN
   RAISE EXCEPTION 'CT166 evidencia C1 V2 inválida';
 END IF;
 e_legacy := pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(e-'causa_fin'-'politica_fin',
      '{periodo_inicio}',p_legacy->'inicio'),
    '{periodo_fin}',p_legacy->'fin');
 IF vec_contratacion_temporal.o404d_material_evidencia_v1(e_legacy) IS NULL THEN
   RAISE EXCEPTION 'CT166 rechazó evidencia legacy no nocturna';
 END IF;

 propuesta := $json${
   "referencia":"propuesta-cobertura:sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
   "huella_sha256":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
   "canon":{"dominio":"vec.dipgra.contratacion-temporal.propuesta-decision-cobertura","version_esquema":2,"algoritmo":"sha-256"},
   "organizacion_ref":"org:sintetica","expediente_ref":"exp:sintetico",
   "version_expediente":2,"analisis_ref":"analisis:sintetico",
   "analisis_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
   "preparacion_evidencias_ref":"preparacion:sintetica",
   "preparacion_evidencias_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
   "catalogo":{"referencia":"catalogo:sintetico","version":1,"huella_sha256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
   "politica":{"referencia":"politica:sintetica","version":1,"huella_sha256":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
   "finalidad_clave":"gestionar_cobertura_temporal","finalidad_ref":"finalidad:sintetica",
   "categoria_ref":"cat:sintetica","periodo":{},
   "generada_en":"2026-10-01T10:00:00Z","valida_hasta":"2026-10-01T10:10:00Z",
   "estado":"viable","via_propuesta":"bolsa","resultados":[],
   "evaluaciones":[{"via_clave":"bolsa","prioridad":1,"estado":"viable"}]
 }$json$::jsonb;
 propuesta := pg_catalog.jsonb_set(propuesta,'{periodo}',p_causa);
 m := vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(propuesta);
 IF m IS NULL THEN RAISE EXCEPTION 'CT166 propuesta abierta V2 rechazada'; END IF;
 h := pg_catalog.encode(pg_catalog.sha256(m),'hex');
 propuesta := pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(propuesta,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('propuesta-cobertura:sha256:'||h));
 h := vec_contratacion_temporal.o404e_semantica_propuesta_v1(propuesta);
 mutante := pg_catalog.jsonb_set(
    propuesta,'{periodo,politica_fin,catalogo_version}','3'::jsonb);
 m := vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(mutante);
 mutante := pg_catalog.jsonb_set(
    pg_catalog.jsonb_set(mutante,'{huella_sha256}',
      pg_catalog.to_jsonb(pg_catalog.encode(pg_catalog.sha256(m),'hex'))),
    '{referencia}',pg_catalog.to_jsonb('propuesta-cobertura:sha256:'||
      pg_catalog.encode(pg_catalog.sha256(m),'hex')));
 IF h IS NULL OR h=vec_contratacion_temporal.o404e_semantica_propuesta_v1(mutante) THEN
   RAISE EXCEPTION 'CT166 semántica V2 no liga política';
 END IF;
 propuesta := pg_catalog.jsonb_set(propuesta,'{periodo}',p_fecha);
 IF vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(propuesta) IS NULL THEN
   RAISE EXCEPTION 'CT166 propuesta fechada V2 rechazada';
 END IF;
END
$prueba$;
ROLLBACK;
