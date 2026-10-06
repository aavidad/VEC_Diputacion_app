\set ON_ERROR_STOP on
-- CT179: o404e_construir_lote_c1_v1 vuelve a construir el lote C1.
-- Antes de CT179 cualquier llamada falla con «column reference "x" is
-- ambiguous» (fallo SQL-2 del recorrido del 05/10/2026). Datos sintéticos;
-- sólo llama a funciones inmutables y termina en ROLLBACK.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='20s';
DO $prueba$
DECLARE
 p_legacy jsonb:='{"inicio":"2026-01-01T00:00:00Z","fin":"2026-12-31T00:00:00Z"}';
 p_fecha jsonb:='{"inicio":"2026-10-02T00:00:00Z","fin":"2026-10-31T00:00:00Z","politica_fin":{"regla_ref":"regla:modalidad:acumulacion","catalogo_version":2,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fecha_fin":"opcional","causa_fin":"fin_tareas"}}';
 p_causa jsonb:='{"inicio":"2026-10-02T00:00:00Z","causa_fin":"reincorporacion_titular","politica_fin":{"regla_ref":"regla:modalidad:sustitucion","catalogo_version":2,"catalogo_huella_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","fecha_fin":"no_aplica","causa_fin":"reincorporacion_titular"}}';
 periodos jsonb[];
 periodo jsonb;
 consumo jsonb;
 consumos jsonb;
 carga jsonb;
 lote jsonb;
 ev jsonb;
 casos integer:=0;
 v_raiz text:='hmac-sha256:vec.contratacion-temporal.cobertura-decision.ambito/v1:'||pg_catalog.repeat('9',64);
BEGIN
 periodos:=ARRAY[p_legacy,p_fecha,p_causa];
 FOREACH periodo IN ARRAY periodos LOOP
  consumo:=pg_catalog.jsonb_build_object(
   'posicion',1,'total',1,
   'peticion_ref','peticion:ct179:sintetica:1',
   'organizacion_ref','organizacion:ct179:sintetica',
   'expediente_ref','expediente:ct179:sintetico',
   'version_expediente',2,
   'catalogo_ref','catalogo:ct179:sintetico','catalogo_version',1,
   'catalogo_huella_sha256',pg_catalog.repeat('c',64),
   'via_clave','via_ct179','comprobacion_clave','comprobacion_ct179',
   'comprobacion_resultado','afirmativa',
   'comprobacion_fuente_ref','fuente:ct179:sintetica',
   'comprobacion_recibo_ref','respuesta:ct179:sintetica:1',
   'comprobacion_evaluada_en','2026-10-01T10:00:00Z',
   'orden_comprobacion',1,'obligatoria',true,
   'procedencia_clave','fuente_ct179',
   'definicion_fuente_ref','fuente:ct179:sintetica',
   'categoria_ref','categoria:ct179:sintetica',
   'periodo',periodo,
   'solicitada_en','2026-10-01T10:00:00Z',
   'emitida_en','2026-10-01T10:00:00Z',
   'valida_hasta','2026-10-01T10:00:04Z',
   'huella_peticion_sha256',pg_catalog.repeat('1',64),
   'huella_resultado_sha256',pg_catalog.repeat('2',64),
   'huella_respuesta_sha256',pg_catalog.repeat('3',64),
   'autoridad_ref','autoridad:ct179:sintetica','generacion',1,
   'recibo_respuesta_ref','respuesta:ct179:sintetica:1',
   'verificador_ref','verificador:ct179:sintetico',
   'publicador_catalogo_ref','publicador:ct179:sintetico',
   'pruebas_canonicas',pg_catalog.jsonb_build_object(
     'peticion_hex','01','resultado_hex','02','atestacion_hex','03',
     'confirmacion_tcb_hex','04','catalogo_hex','05',
     'verificador_hex','06','resumen_hex','07'));
  consumos:=pg_catalog.jsonb_build_array(consumo);
  carga:=pg_catalog.jsonb_build_object(
   'cabecera',pg_catalog.jsonb_build_object(
     'organizacion_ref','organizacion:ct179:sintetica',
     'expediente_ref','expediente:ct179:sintetico',
     'version_expediente',2,
     'reserva_ref','reserva:ct179:sintetica',
     'preparacion_c1_ref','preparacion:ct179:sintetica',
     'preparacion_c1_huella_sha256',pg_catalog.repeat('4',64),
     'huella_orden_sha256',pg_catalog.repeat('5',64),
     'huella_ordenes_consumo_c1_sha256',
       vec_contratacion_temporal.o404e_huella_ordenes_c1_v1(consumos),
     'decision_vec_ref','decision-vec:ct179:sintetica',
     'correlacion_vec_ref','correlacion:ct179:sintetica'),
   'gobierno',pg_catalog.jsonb_build_object('catalogo',pg_catalog.jsonb_build_object(
     'referencia','catalogo:ct179:sintetico','version',1,
     'huella_sha256',pg_catalog.repeat('c',64))),
   'consumos_c1',consumos,
   'concesion',pg_catalog.jsonb_build_object(
     'efecto_en','2026-10-01T10:00:01Z',
     'propuesta',pg_catalog.jsonb_build_object('periodo',periodo)));
  IF carga#>>'{cabecera,huella_ordenes_consumo_c1_sha256}' IS NULL THEN
   RAISE EXCEPTION 'CT179 huella de órdenes sintética inválida';
  END IF;

  -- Positivo: el periodo del consumo y el de la propuesta coinciden.
  lote:=vec_contratacion_temporal.o404e_construir_lote_c1_v1(carga,v_raiz);
  IF lote IS NULL OR pg_catalog.jsonb_array_length(lote->'evidencias')<>1 THEN
   RAISE EXCEPTION 'CT179 lote no construido con periodo %',periodo;
  END IF;
  ev:=lote#>'{evidencias,0}';
  IF ev->'periodo_inicio' IS DISTINCT FROM periodo->'inicio'
     OR ev->'periodo_fin' IS DISTINCT FROM COALESCE(periodo->'fin','null'::jsonb)
     OR ev->'politica_fin' IS DISTINCT FROM periodo->'politica_fin'
     OR ev->'causa_fin' IS DISTINCT FROM periodo->'causa_fin' THEN
   RAISE EXCEPTION 'CT179 evidencia no conserva el periodo %',periodo;
  END IF;

  -- Negativo: con política de fin, el periodo del consumo debe ser el de la
  -- propuesta. El cambio de alias no debe aflojar esta condición.
  IF periodo ? 'politica_fin' AND vec_contratacion_temporal.o404e_construir_lote_c1_v1(
       pg_catalog.jsonb_set(carga,'{concesion,propuesta,periodo,inicio}',
         '"2026-10-03T00:00:00Z"'::jsonb),v_raiz) IS NOT NULL THEN
   RAISE EXCEPTION 'CT179 aceptó un periodo distinto del de la propuesta';
  END IF;
  -- Negativo: la huella de las órdenes debe corresponder a los consumos.
  IF vec_contratacion_temporal.o404e_construir_lote_c1_v1(
       pg_catalog.jsonb_set(carga,'{cabecera,huella_ordenes_consumo_c1_sha256}',
         pg_catalog.to_jsonb(pg_catalog.repeat('0',64))),v_raiz) IS NOT NULL THEN
   RAISE EXCEPTION 'CT179 aceptó una huella de órdenes ajena';
  END IF;
  casos:=casos+1;
 END LOOP;
 IF casos<>3 THEN RAISE EXCEPTION 'CT179 casos incompletos'; END IF;
 RAISE NOTICE 'CT179-PRUEBA-OK';
END
$prueba$;
ROLLBACK;
