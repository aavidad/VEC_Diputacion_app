import assert from 'node:assert/strict';
import test from 'node:test';
import { crearClienteLiquidacionDietasHTTP } from './cliente-liquidacion-http.js';

const ref = 'dco_1234567890123456789012';
const hash = 'a'.repeat(64);
const propuesta = { esquema: 'vec_dietas_preparacion_liquidacion_publicada_v1', comision_ref: ref, comision_version: 3,
  snapshot_sha256: hash, catalogo_ref: 'cat', catalogo_version: 'v1', catalogo_sha256: hash,
  catalogo: { referencia: 'cat', version: 'v1', huella_sha256: hash, publicacion_ref: 'pub', fecha_publicacion: '2026-01-01', vigente_desde: '2026-01-01', vigente_hasta: '2026-12-31', version_tarifa_ref: 'tarifa-v1', normas: [{ referencia: 'norma', articulo: 'art. 1', url: 'https://example.invalid' }],
    reglas: [{ referencia: 'regla', tipo: 'dieta', concepto: 'manutencion' }] },
  lineas: [{ indice: 0, regla_ref: 'regla', original_centimos: 1000, calculado_publicado_centimos: 1000, diferencia_centimos: 0, reconocido_propuesto_centimos: 1000, rechazado_centimos: 0, motivo_codigo: '' }],
  totales: { original_centimos: 1000, calculado_publicado_centimos: 1000, diferencia_centimos: 0, reconocido_propuesto_centimos: 1000, rechazado_centimos: 0 }, decision_pendiente: false };
const liquidacion = { esquema: 'vec_dietas_liquidacion_economica_v1', snapshot_sha256: hash, economia: propuesta,
  constancia: { version_entrada: 3, version_resultante: 4, recibo_ref: 'recibo', fecha: '2026-10-14T10:00:00Z' } };
const json = (datos, status = 200) => new Response(JSON.stringify(datos), { status, headers: { 'content-type': 'application/json' } });
const entrada = { version_esperada: 3, clave_idempotencia: 'solicitud-liquidacion-0001', revisiones: [{ indice: 0, regla_ref: 'regla', reconocido_propuesto_centimos: 1000, motivo_codigo: '' }] };

test('GET y POST usan la ruta nominal, cuerpo limitado y certificado del mismo origen', async () => {
  const llamadas = [];
  const cliente = crearClienteLiquidacionDietasHTTP({ fetchImpl: async (ruta, opciones) => { llamadas.push([ruta, opciones]);
    return json(ruta.endsWith('/propuesta?version_esperada=3') ? { propuesta } : { liquidacion }); } });
  assert.deepEqual(await cliente.propuesta(ref, 3), propuesta);
  assert.equal((await cliente.registrar(ref, entrada)).constancia.recibo_ref, 'recibo');
  assert.equal((await cliente.leer(ref, 4)).constancia.version_resultante, 4);
  assert.match(llamadas[0][0], /\/circuito\/dco_.*\/liquidaciones\/propuesta\?version_esperada=3$/u);
  assert.equal(llamadas[1][0], `/api/vec/dietas/comisiones/circuito/${ref}/liquidaciones`);
  assert.deepEqual(JSON.parse(llamadas[1][1].body), entrada);
  for (const [, opciones] of llamadas) { assert.equal(opciones.credentials, 'same-origin'); assert.equal(opciones.mode, 'same-origin'); assert.equal(opciones.cache, 'no-store'); assert.equal(opciones.redirect, 'error'); assert.equal(opciones.referrerPolicy, 'no-referrer'); }
});

test('rechaza identidad añadida, 403 y respuesta que no corresponde a la versión', async () => {
  const cliente = crearClienteLiquidacionDietasHTTP({ fetchImpl: async () => json({ error: 'dietas.error.acceso_denegado' }, 403) });
  await assert.rejects(cliente.propuesta(ref, 3), { codigo: 'acceso_denegado', estado: 403 });
  assert.throws(() => cliente.registrar(ref, { ...entrada, unidad_ref: 'inventada' }), TypeError);
  const incompatible = crearClienteLiquidacionDietasHTTP({ fetchImpl: async () => json({ propuesta: { ...propuesta, comision_version: 7 } }) });
  await assert.rejects(incompatible.propuesta(ref, 3), { codigo: 'respuesta_incompatible' });
});

test('POST con respuesta perdida queda incierto y una cancelación explícita no lo convierte en éxito', async () => {
  const cliente = crearClienteLiquidacionDietasHTTP({ fetchImpl: async () => { throw new Error('red'); } });
  await assert.rejects(cliente.registrar(ref, entrada), { resultadoIndeterminado: true });
  const controlador = new AbortController();
  const cancelado = crearClienteLiquidacionDietasHTTP({ fetchImpl: async (_ruta, opciones) => { opciones.signal.addEventListener('abort', () => {}); controlador.abort(); throw new Error('abort'); } });
  await assert.rejects(cancelado.registrar(ref, entrada, { signal: controlador.signal }), { codigo: 'operacion_abortada', resultadoIndeterminado: false });
});

test('admite el máximo de 102 revisiones del contrato y rechaza una más antes de enviar', async () => {
  let peticiones = 0;
  const cliente = crearClienteLiquidacionDietasHTTP({ fetchImpl: async () => { peticiones++; return json({ error: 'dietas.error.acceso_denegado' }, 403); } });
  const revisiones = Array.from({ length: 102 }, (_, indice) => ({ indice, regla_ref: 'regla', reconocido_propuesto_centimos: 1000, motivo_codigo: '' }));
  await assert.rejects(cliente.registrar(ref, { ...entrada, revisiones }), { codigo: 'acceso_denegado' });
  assert.equal(peticiones, 1);
  assert.throws(() => cliente.registrar(ref, { ...entrada, revisiones: [...revisiones, { indice: 102, regla_ref: 'regla', reconocido_propuesto_centimos: 1000, motivo_codigo: '' }] }), TypeError);
  assert.equal(peticiones, 1);
});

test('PDF y Word durables usan versión exacta, idioma activo y bytes originales con huella comprobada', async () => {
  const { createHash, webcrypto } = await import('node:crypto');
  const nombre = 'VEC-D-2026-000012'; const llamadas = [];
  const contenidos = { pdf: new TextEncoder().encode('%PDF-1.7\ncontenido'), docx: Uint8Array.from([0x50, 0x4b, 0x03, 0x04, 1, 2, 3]) };
  const mime = { pdf: 'application/pdf', docx: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' };
  const responder = (formato, overrides = {}) => {
    const bytes = contenidos[formato];
    const headers = { 'content-type': mime[formato], 'content-disposition': `attachment; filename="${nombre}.${formato}"`,
      'content-length': String(bytes.length), 'cache-control': 'no-store', 'x-content-type-options': 'nosniff',
      'x-content-sha256': createHash('sha256').update(bytes).digest('hex'), ...overrides };
    return new Response(bytes, { status: 200, headers });
  };
  const cliente = crearClienteLiquidacionDietasHTTP({ cryptoImpl: webcrypto, fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]); return responder(ruta.endsWith('pdf') ? 'pdf' : 'docx'); } });
  const pdf = await cliente.descargar(ref, 4, nombre, 'pdf', 'es');
  const word = await cliente.descargar(ref, 4, nombre, 'docx', 'en');
  assert.deepEqual(pdf.bytes, contenidos.pdf); assert.deepEqual(word.bytes, contenidos.docx);
  assert.equal(pdf.nombre, `${nombre}.pdf`); assert.equal(word.nombre, `${nombre}.docx`);
  assert.equal(llamadas[0][0], `/api/vec/dietas/comisiones/${ref}/liquidaciones/4/documento?formato=pdf`);
  assert.equal(llamadas[1][1].headers.Accept, mime.docx);
  assert.equal(llamadas[1][1].headers['Accept-Language'], 'en');
  assert.equal(llamadas[0][1].credentials, 'same-origin'); assert.equal(llamadas[0][1].mode, 'same-origin'); assert.equal(llamadas[0][1].cache, 'no-store');
  assert.equal(llamadas[0][1].redirect, 'error'); assert.equal(llamadas[0][1].referrerPolicy, 'no-referrer');
  await assert.rejects(cliente.descargar(ref, 'actual', nombre, 'pdf', 'es'), TypeError);
});

test('descarga rechaza 403, tipo, tamaño, nombre y huella incompatibles sin entregar bytes', async () => {
  const { createHash, webcrypto } = await import('node:crypto'); const nombre = 'VEC-D-2026-000012';
  const bytes = new TextEncoder().encode('%PDF-1.7\ncontenido'); const hash = createHash('sha256').update(bytes).digest('hex');
  const headers = { 'content-type': 'application/pdf', 'content-disposition': `attachment; filename="${nombre}.pdf"`,
    'content-length': String(bytes.length), 'cache-control': 'no-store', 'x-content-type-options': 'nosniff', 'x-content-sha256': hash };
  for (const mal of [{ 'content-type': 'text/html' }, { 'content-length': String(9 * 1024 * 1024) },
    { 'content-disposition': 'attachment; filename="otro.pdf"' }, { 'x-content-sha256': 'b'.repeat(64) }]) {
    const cliente = crearClienteLiquidacionDietasHTTP({ cryptoImpl: webcrypto, fetchImpl: async () => new Response(bytes, { status: 200, headers: { ...headers, ...mal } }) });
    await assert.rejects(cliente.descargar(ref, 4, nombre, 'pdf', 'es'));
  }
  const denegado = crearClienteLiquidacionDietasHTTP({ cryptoImpl: webcrypto, fetchImpl: async () => json({ error: 'dietas.error.acceso_denegado' }, 403) });
  await assert.rejects(denegado.descargar(ref, 4, nombre, 'pdf', 'es'), { codigo: 'acceso_denegado', estado: 403 });
});

test('propuesta pendiente conserva diferencias firmadas y acepta reconocido superior al declarado si lo calcula la tarifa publicada', async () => {
  const pendiente = { ...propuesta, decision_pendiente: true,
    lineas: [
      { indice: 0, regla_ref: 'regla', original_centimos: 1000, calculado_publicado_centimos: 1200, diferencia_centimos: 200, reconocido_propuesto_centimos: 1200, rechazado_centimos: 0, motivo_codigo: '' },
      { indice: 1, regla_ref: 'regla', original_centimos: 1200, calculado_publicado_centimos: 1000, diferencia_centimos: -200, reconocido_propuesto_centimos: 1000, rechazado_centimos: 0, motivo_codigo: '' },
    ], totales: { original_centimos: 2200, calculado_publicado_centimos: 2200, diferencia_centimos: 0, reconocido_propuesto_centimos: 2200, rechazado_centimos: 0 } };
  const lectura = crearClienteLiquidacionDietasHTTP({ fetchImpl: async () => json({ propuesta: pendiente }) });
  assert.deepEqual((await lectura.propuesta(ref, 3)).lineas.map((l) => l.diferencia_centimos), [200, -200]);
  const revisiones = pendiente.lineas.map((l) => ({ indice: l.indice, regla_ref: l.regla_ref,
    reconocido_propuesto_centimos: l.calculado_publicado_centimos, motivo_codigo: 'diferencia_tarifa_publicada' }));
  const economia = { ...pendiente, decision_pendiente: false, lineas: pendiente.lineas.map((l) => ({ ...l, motivo_codigo: 'diferencia_tarifa_publicada' })) };
  const registro = crearClienteLiquidacionDietasHTTP({ fetchImpl: async () => json({ liquidacion: { ...liquidacion, economia } }) });
  assert.equal((await registro.registrar(ref, { ...entrada, revisiones })).economia.lineas[0].reconocido_propuesto_centimos, 1200);
  const alterada = { ...pendiente, lineas: [{ ...pendiente.lineas[0], diferencia_centimos: -200 }, pendiente.lineas[1]] };
  const rechazado = crearClienteLiquidacionDietasHTTP({ fetchImpl: async () => json({ propuesta: alterada }) });
  await assert.rejects(rechazado.propuesta(ref, 3), { codigo: 'respuesta_incompatible' });
});
