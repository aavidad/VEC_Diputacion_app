import test from 'node:test';
import assert from 'node:assert/strict';
import { seleccionarOpciones, resumirRecuperacionFirmas, interceptarHTTP, aplicarReconciliacionFirma, esRespuesta, consultarFirmas } from './recorrer.mjs';
import { validarReciboFirma, origen } from './config.mjs';

test('B2 selecciona referencias del expediente consultado y corta ambigüedades', () => {
  const consulta = { opciones: { vacantes: [
    { plaza_ref: 'plaza:1', puesto_ref: 'puesto:1', version_plantilla_ref: 'plantilla:1', version_rpt_ref: 'rpt:1' },
    { plaza_ref: 'plaza:1', puesto_ref: 'puesto:2', version_plantilla_ref: 'plantilla:2', version_rpt_ref: 'rpt:2' }],
    regimenes: [{ ref: 'regimen:1', version: 1 }, { ref: 'regimen:1', version: 2 }], modalidades: [{ ref: 'modalidad:1', version: 1 }],
    clases_ocupacion: [{ valor: 'titular' }], motivos: ['incorporacion'],
    documentos: [{ documento_ref: 'documento:1', documento_sha256: 'a'.repeat(64) }], periodo: { fuente_ref: '', desde: '', hasta: '' } } };
  const seleccion = { vacante: { plaza_ref: 'plaza:1', puesto_ref: 'puesto:2', version_plantilla_ref: 'plantilla:2', version_rpt_ref: 'rpt:2' },
    regimen: { ref: 'regimen:1', version: 2 }, modalidad: { ref: 'modalidad:1', version: 1 },
    clase_ocupacion: 'titular', motivo: 'incorporacion', documento: { documento_ref: 'documento:1', documento_sha256: 'a'.repeat(64) },
    desde: '2026-10-02', hasta: '' };
  assert.equal(seleccionarOpciones(consulta, seleccion).vacante, '1');
  assert.equal(seleccionarOpciones(consulta, seleccion).regimen, '1');
  assert.throws(() => seleccionarOpciones(consulta, { ...seleccion, vacante: { ...seleccion.vacante, puesto_ref: 'puesto:otra' } }), /b2_opcion/u);
  consulta.opciones.vacantes.push({ ...seleccion.vacante });
  assert.throws(() => seleccionarOpciones(consulta, seleccion), /b2_opcion/u);
});

test('recuperación E3 exige pasos firmados, recibo y fecha idénticos', () => {
  const firma = { documento: 'resolucion', orden: 1, recibo_ref: 'recibo:firma:1',
    registrada_en: '2026-10-01T10:00:00Z', custodiado: null };
  const estado = { documentos: [{ documento: 'resolucion', completo: true, pasos: [{ orden: 1,
    estado: 'firmado', recibo_ref: firma.recibo_ref, registrada_en: firma.registrada_en }] }] };
  assert.equal(resumirRecuperacionFirmas(estado, [firma]).recibos_cotejados, 1);
  const inventario = resumirRecuperacionFirmas(estado, []).inventario;
  assert.equal(inventario.length, 1, 'todos los recibos se inventarían incluso si ya estaban firmados');
  assert.throws(() => resumirRecuperacionFirmas(estado, [], [{ ...inventario[0], recibo_ref: 'recibo:otro' }]), /recuperacion/u);
  assert.throws(() => resumirRecuperacionFirmas(estado, [{ ...firma, recibo_ref: 'recibo:otro' }]), /recuperacion/u);
  assert.throws(() => resumirRecuperacionFirmas({ documentos: [{ ...estado.documentos[0], completo: false }] }, [firma]), /firma_incompleta/u);
  assert.throws(() => resumirRecuperacionFirmas({ documentos: [estado.documentos[0], { documento: 'diligencia', completo: true, pasos: [pasoConRecibo(1, firma.recibo_ref)] }] }, []), /recuperacion/u);
});

function pasoConRecibo(orden, recibo_ref) { return { orden, estado: 'firmado', recibo_ref, registrada_en: '2026-10-01T10:00:00Z' }; }

test('el recibo E3 exige verificación positiva y conserva límite de eficacia', () => {
  const recibido = { esquema: 'vec.contratacion-temporal.recibo-firma-documento.v1', expediente_ref: 'expediente:b2:1',
    documento: 'resolucion', paso_orden: 1, perfil_ref: 'perfil:ct:jefatura', resultado: 'firmado', firma_verificada: true, firma_eficaz: false,
    verificacion: { estado: 'valida', motivo: 'verificada', certificado_sha256: 'b'.repeat(64), firmado_sha256: 'a'.repeat(64) },
    recibo_ref: 'recibo:firma:1', registrada_en: '2026-10-01T10:00:00Z' };
  const identidad = { perfil_ref: 'perfil:ct:jefatura', certificado_firma_sha256: 'b'.repeat(64) };
  assert.equal(validarReciboFirma(recibido, 'expediente:b2:1', 'resolucion', 1, identidad).firmado_sha256, 'a'.repeat(64));
  assert.throws(() => validarReciboFirma({ ...recibido, firma_eficaz: true }, 'expediente:b2:1', 'resolucion', 1, identidad), /entrada/u);
  assert.throws(() => validarReciboFirma(recibido, 'expediente:b2:1', 'resolucion', 1, { ...identidad, perfil_ref: 'perfil:ct:tecnico' }), /entrada/u);
  assert.throws(() => validarReciboFirma(recibido, 'expediente:b2:1', 'resolucion', 1, { ...identidad, certificado_firma_sha256: 'c'.repeat(64) }), /entrada/u);
});

test('la red auxiliar solo admite un origen local exacto', () => {
  assert.equal(origen('https://127.0.0.1:8443'), 'https://127.0.0.1:8443');
  assert.equal(origen('wss://127.0.0.1:63117', { auxiliar: true }), 'wss://127.0.0.1:63117');
  assert.throws(() => origen('https://example.org:443'), /entrada/u);
  assert.throws(() => origen('https://127.0.0.1:8443/api'), /entrada/u);
});

test('la clave se persiste antes de enviar el POST B2; si falla la escritura se aborta', async () => {
  const eventos = [], origenApp = 'https://127.0.0.1:8443';
  const body = { expediente_ref: 'expediente:b2:1', plan_ref: 'plan:b2:1', version_plan: 1,
    clave_idempotencia: '00000000-0000-4000-8000-000000000001' };
  const request = { url: () => `${origenApp}/api/interno/contratacion-temporal/incorporacion-personal-b2/confirmar/v1`,
    method: () => 'POST', postDataJSON: () => body };
  const route = { request: () => request, fetch: async () => { eventos.push('red'); return {
    status: () => 200, url: () => request.url(), headersArray: async () => [] }; },
  fulfill: async () => eventos.push('respuesta'), abort: async () => eventos.push('abortada') };
  const estado = { b2: {}, red_bloqueada: 0 };
  await interceptarHTTP(route, { origen: origenApp, auxiliares: [], expediente_ref: body.expediente_ref }, estado,
    () => eventos.push('persistida'));
  assert.deepEqual(eventos, ['persistida', 'red', 'respuesta']);
  assert.equal(estado.pendiente.clave_idempotencia, body.clave_idempotencia);
  eventos.length = 0;
  await interceptarHTTP(route, { origen: origenApp, auxiliares: [], expediente_ref: body.expediente_ref }, estado,
    () => { throw new Error('disco'); });
  assert.deepEqual(eventos, ['abortada']);
});

test('firma incierta observada por GET conserva bloqueo al faltar la clave E3', () => {
  const pendiente = { ruta: '/api/vec/contratacion-temporal/firmas-documento', documento: 'resolucion', paso_orden: 1 };
  const paso = { orden: 1, estado: 'firmado', recibo_ref: 'recibo:firma:1', registrada_en: '2026-10-01T10:00:00Z' };
  const estado = { pendiente, firmas: [], b2: {} };
  aplicarReconciliacionFirma(estado, { documentos: [{ documento: 'resolucion', completo: true, pasos: [paso] }] }, paso);
  assert.equal(estado.estado, 'FIRMA_CONSULTADA_SIN_CLAVE');
  assert.deepEqual(estado.pendiente, pendiente);
  assert.equal(estado.firma, undefined);
  assert.equal(estado.reconciliacion.inventario.length, 1);
});

test('una respuesta auxiliar con la misma ruta nunca acredita un POST de VEC', () => {
  const c = { origen: 'https://127.0.0.1:8443' };
  const r = { url: () => 'https://127.0.0.1:63117/api/vec/contratacion-temporal/firmas-documento',
    request: () => ({ method: () => 'POST' }) };
  assert.equal(esRespuesta(r, c, '/api/vec/contratacion-temporal/firmas-documento'), false);
});

test('el catálogo E3 de ejemplo admite una consulta real verificada', async () => {
  const pagina = { evaluate: async () => ({ status: 200, body: { data: {
    esquema: 'vec.contratacion-temporal.estado-firmas-documento.v1', catalogo_ref: 'catalogo:ejemplo',
    huella_sha256: 'a'.repeat(64), ejemplo: true, firma_eficaz: false,
    verificacion_disponible: true, documentos: [],
  } } }) };
  const estado = await consultarFirmas(pagina, { expediente_ref: 'expediente:b2:1' });
  assert.equal(estado.ejemplo, true);
});
