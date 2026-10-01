import test from 'node:test';
import assert from 'node:assert/strict';
import { comprobarResultadoConcursos, crearClienteConcursos } from './concursos-cliente.js';

const solicitud = { configuracion: { convocatoria_ref: 'convocatoria:prueba', version: 'version:prueba' } };
const respuesta = () => ({ schema_version: 'provision.simulacion.v1', alcance: 'simulacion', resultado: {
  estado: 'simulacion_local_sin_efectos', version_motor: 'provision.v1', version_reglas: solicitud.configuracion.version,
  convocatoria_ref: solicitud.configuracion.convocatoria_ref, completo: true, bruto: '7123456', maximo_total: '5000000', total: '5000000',
  incidencias: [], desglose: [], huella_reglas: 'a'.repeat(64), huella_entrada: 'b'.repeat(64), huella_resultado: 'c'.repeat(64),
} });

test('el resultado completo exige suma y máximo global canónicos, incluido cero', () => {
  const datos = respuesta();
  assert.equal(comprobarResultadoConcursos(datos, solicitud), datos);
  for (const campo of ['bruto', 'maximo_total', 'total']) {
    for (const valor of [undefined, null, 0, '', '00', '01', '-1', '1.2', '1/2', '1e6', '10000000000000000000', '<script>']) {
      const invalida = respuesta(); invalida.resultado[campo] = valor;
      assert.throws(() => comprobarResultadoConcursos(invalida, solicitud), { codigo: 'respuesta_invalida' }, `${campo}: ${String(valor)}`);
    }
    const cero = respuesta(); cero.resultado[campo] = '0';
    assert.equal(comprobarResultadoConcursos(cero, solicitud), cero);
  }
});

test('una respuesta pendiente mantiene total ausente y conserva la suma parcial sin completarla', () => {
  const datos = respuesta(); datos.resultado.completo = false; datos.resultado.total = null;
  assert.equal(comprobarResultadoConcursos(datos, solicitud), datos);
  assert.equal(datos.resultado.bruto, '7123456');
  for (const total of [undefined, '0', '5000000']) {
    const invalida = structuredClone(datos); invalida.resultado.total = total;
    assert.throws(() => comprobarResultadoConcursos(invalida, solicitud), { codigo: 'respuesta_invalida' });
  }
});

test('el cliente rechaza un resultado completo sin tope antes de entregarlo al editor', async () => {
  const datos = respuesta(); delete datos.resultado.maximo_total;
  const cliente = crearClienteConcursos({ fetchImpl: async () => ({ ok: true, text: async () => JSON.stringify(datos) }) });
  await assert.rejects(cliente.simular(solicitud), { codigo: 'respuesta_invalida' });
});
