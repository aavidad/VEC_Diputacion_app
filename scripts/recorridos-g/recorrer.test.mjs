import test from 'node:test';
import assert from 'node:assert/strict';
import { catalogo, main, plan, solicitudPermitida, interceptar, validarCaso, validarConfiguracion, validarPasos, validarPuertaPersonal,
  validarPuertaCompetencias, validarRecibo, validarRuta, validarDesglose } from './recorrer.mjs';

const origen = 'https://127.0.0.1:8443';
const cuerpoAlta = JSON.stringify({ clave_idempotencia: 'clave_0123456789abcdef', fecha_inicio: '2026-10-01' });
const req = (ruta, metodo = 'GET', cuerpo = cuerpoAlta) => ({ url: () => origen + ruta, method: () => metodo, postData: () => cuerpo });
const recibo = (repeticion = false) => ({ comision: { referencia: 'dco_1234567890123456789012', estado: 'borrador', version: 1 },
  recibo: { referencia: 'rcd_1234567890123456789012', version: 1, registrado_en: '2026-10-01T10:00:00.000000Z', repeticion } });

test('plan puro enumera D1–D9 sin pedir configuración ni activar Chrome', () => {
  assert.equal(plan().length, 10);
  assert.equal(new Set(catalogo.escenarios.map(c => c.id)).size, 10);
  assert.equal(plan().filter(c => c.estado === 'CONTRATO_PENDIENTE').length, 2);
  assert.throws(() => validarCaso('informe_pdf'));
});

test('ruta, desglose y tramos se comprueban antes de dar verde', () => {
  assert.throws(() => validarRuta({ data: { code: 'Ok', engine: 'osrm_on_premise', routes: [] } }));
  assert.throws(() => validarDesglose({ comision: { vehiculo_propio: true, rutas: [{}], documento: { lineas: [{ tipo: 'kilometraje' }] } } }));
  assert.throws(() => validarRecibo({ comision: { referencia: 'dco_1', estado: 'enviado_pendiente_revision', version: 1 }, recibo: { referencia: 'rcd_1', version: 1, registrado_en: '2026-10-01T10:00:00Z' } }, 'enviado_pendiente_revision'));
});

test('redirección y Set-Cookie se abortan sin entregar respuesta', async () => {
  for (const [status, headers] of [[302, []], [200, [{ name: 'Set-Cookie', value: 'x=y' }]]]) {
    let abortos = 0;
    const datos = { bloqueadas: 0, red_fallida: 0, http_fallidos: 0 };
    await interceptar({ request: () => req('/portal-empleado/'), fetch: async options => {
      assert.equal(options.maxRedirects, 0);
      return { status: () => status, url: () => origen + '/portal-empleado/', headersArray: async () => headers };
    }, abort: async () => { abortos++; }, fulfill: () => assert.fail('no entregar') }, origen, 'empleado_solicitud', datos);
    assert.equal(abortos, 1); assert.equal(datos.bloqueadas, 1);
  }
});

test('sin AUT27, Personal y montaje completos no se admiten pasos ni ejecución', async () => {
  const c = { version: 1, sintetico: true, entorno_controlado: true, commit_servido: 'a'.repeat(40),
    idioma: 'es', origen, dependencias: { AUT27: false, Personal: true, Dietas_SQL: true, Montaje: true } };
  assert.throws(() => validarConfiguracion(c, 'empleado_solicitud'));
  assert.equal(await main(['--caso', 'empleado_solicitud', '--config', '/ruta-inexistente', '--salida', '/tmp/salida']), 2);
});

test('pasos vacíos, efecto ajeno o desglose sin ruta no pueden terminar verdes', () => {
  assert.throws(() => validarPasos('empleado_solicitud', [{ tipo: 'visible', selector: '[data-dietas-recorridos]' }]));
  assert.throws(() => validarPasos('empleado_solicitud', [
    { tipo: 'visible', selector: '[data-dietas-recorridos]' },
    { tipo: 'efecto', selector: '[data-dietas-borrador-guardar]', metodo: 'POST', ruta: '/api/vec/dietas/comisiones/circuito/dco_1/decisiones', estado_esperado: 'borrador' },
  ]));
  assert.throws(() => validarPasos('empleado_gastos_km', [
    { tipo: 'visible', selector: '[data-dietas-recorridos]' },
    { tipo: 'efecto', selector: '[data-dietas-borrador-guardar]', metodo: 'PUT', ruta: '/api/vec/dietas/comisiones/dco_1', estado_esperado: 'borrador' },
  ]));
  assert.throws(() => validarPasos('revision', [
    { tipo: 'visible', selector: '[data-dietas-recorridos]' },
    { tipo: 'efecto', selector: '[data-dietas-circuito-decision="devolver"]', metodo: 'POST', ruta: '/api/vec/dietas/comisiones/circuito/dco_123/decisiones', etapa: 'revision', decision: 'devolver', estado_esperado: 'devuelta' },
  ]));
});

test('solo admite la petición exacta mientras espera el efecto declarado', () => {
  const alta = '/api/vec/dietas/comisiones';
  assert.equal(solicitudPermitida(req(alta, 'POST'), origen, 'empleado_solicitud'), false);
  assert.equal(solicitudPermitida(req(alta, 'POST'), origen, 'empleado_solicitud', { metodo: 'POST', ruta: alta }), true);
  assert.equal(solicitudPermitida(req(alta, 'POST'), origen, 'revision', { metodo: 'POST', ruta: alta }), false);
  assert.equal(solicitudPermitida(req(alta + '?actor=otro', 'POST'), origen, 'empleado_solicitud', { metodo: 'POST', ruta: alta }), false);
  assert.equal(solicitudPermitida(req('/api/vec/dietas/comisiones/dco_123/enviar', 'POST'), origen, 'empleado_solicitud', { metodo: 'POST', ruta: alta }), false);
  assert.equal(solicitudPermitida(req('/api/vec/dietas/road-route', 'POST'), origen, 'liquidacion'), false);
  assert.equal(solicitudPermitida(req('/api/vec/dietas/comisiones', 'DELETE'), origen, 'empleado_solicitud'), false);
  assert.equal(solicitudPermitida({ ...req(alta), url: () => 'https://127.0.0.1:8444' + alta }, origen, 'empleado_solicitud'), false);
});

test('D6 comprueba etapa y decisión del cuerpo, incluida la etapa ajena', () => {
  const ruta = '/api/vec/dietas/comisiones/circuito/dco_123/decisiones';
  const esperado = { metodo: 'POST', ruta, etapa: 'revision', decision: 'aprobar', consumida: false };
  const cuerpo = (etapa, decision) => JSON.stringify({ etapa, decision, motivo: '', clave_idempotencia: 'clave_0123456789abcdef', version_esperada: 2 });
  assert.equal(solicitudPermitida(req(ruta, 'POST', cuerpo('revision', 'aprobar')), origen, 'revision', esperado), true);
  assert.equal(solicitudPermitida(req(ruta, 'POST', cuerpo('autorizacion', 'aprobar')), origen, 'revision', esperado), false);
  assert.equal(solicitudPermitida(req(ruta, 'POST', cuerpo('revision', 'devolver')), origen, 'revision', esperado), false);
  esperado.consumida = true;
  assert.equal(solicitudPermitida(req(ruta, 'POST', cuerpo('revision', 'aprobar')), origen, 'revision', esperado), false);
});

test('cuota consumida antes de fetch bloquea un segundo POST real', async () => {
  const ruta = '/api/vec/dietas/comisiones';
  const datos = { bloqueadas: 0, red_fallida: 0, http_fallidos: 0, mutaciones_enviadas: 0,
    efectoEsperado: { metodo: 'POST', ruta, consumida: false } };
  let fetches = 0, abortos = 0, entregas = 0;
  const route = () => ({ request: () => req(ruta, 'POST'), fetch: async () => {
    fetches++;
    return { status: () => 201, url: () => origen + ruta, headersArray: async () => [] };
  }, abort: async () => { abortos++; }, fulfill: async () => { entregas++; } });
  await Promise.all([interceptar(route(), origen, 'empleado_solicitud', datos), interceptar(route(), origen, 'empleado_solicitud', datos)]);
  assert.deepEqual([fetches, abortos, entregas, datos.mutaciones_enviadas, datos.bloqueadas], [1, 1, 1, 1, 1]);
});

test('recibo nuevo y recuperación se distinguen; referencias y fecha permanecen privadas en el resultado', () => {
  const nuevo = validarRecibo(recibo(false), 'borrador', 'nuevo', 201);
  assert.deepEqual([nuevo.comision_ref, nuevo.recibo_ref, nuevo.registrado_en, nuevo.version, nuevo.repeticion],
    ['dco_1234567890123456789012', 'rcd_1234567890123456789012', '2026-10-01T10:00:00.000000Z', 1, false]);
  assert.equal(validarRecibo(recibo(true), 'borrador', 'recuperacion', 200).repeticion, true);
  assert.throws(() => validarRecibo(recibo(true), 'borrador', 'nuevo', 200));
  assert.throws(() => validarRecibo(recibo(false), 'borrador', 'recuperacion', 201));
});

test('un 200 sin autoridad Personal, competencia o recibo compatible no acredita escenario', () => {
  assert.throws(() => validarPuertaPersonal({ relaciones_autorizadas: [], fecha_referencia: '2026-10-01' }));
  assert.throws(() => validarPuertaCompetencias({ fuente: 'sin_fuente', etapas: [] }, 'revision'));
  assert.throws(() => validarRecibo({ comision: { referencia: 'dco_1', estado: 'borrador', version: 2 }, recibo: { referencia: 'rcd_1', version: 1, registrado_en: '2026-10-01T10:00:00Z' } }, 'borrador'));
  assert.throws(() => validarRecibo({ comision: { referencia: 'dco_1', estado: 'borrador', version: 1 }, recibo: { referencia: 'rcd_1', version: 1, registrado_en: '2026-10-01T10:00:00Z' } }, 'enviado_pendiente_revision'));
});
