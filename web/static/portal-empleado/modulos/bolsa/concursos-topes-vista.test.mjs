import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { renderizarConcursos } from './concursos-vista.js';
import { crearEditorConcursos } from './concursos-editor.js';
import { cargarTextos } from '../../../comun/textos.js';

const ejemplo = JSON.parse(await readFile(new URL('../../../../../internal/modules/provision/adapters/simulacion/ejemplos/concursos-v1.sintetico.json', import.meta.url), 'utf8'));
const resultado = (bruto, maximo_total, total, completo = true) => ({ resultado: {
  completo, bruto, maximo_total, total, incidencias: completo ? [] : ['dato_no_disponible:grado'],
  desglose: [], version_reglas: 'version-respuesta', huella_reglas: 'a'.repeat(64),
  huella_entrada: 'b'.repeat(64), huella_resultado: 'c'.repeat(64),
} });
async function estadoComparado(antes, despues) {
  const respuestas = [antes, despues];
  const editor = crearEditorConcursos({ cliente: { simular: async () => respuestas.shift() } });
  editor.cargar(ejemplo); await editor.comparar();
  return editor.estado();
}

for (const idioma of ['es', 'en']) {
  test(`Concursos explica el tope de cada respuesta y conserva seis decimales (${idioma})`, async () => {
    const textos = await cargarTextos('baremo-concursos', { idioma });
    // El borrador y el desglose no aportan estas cifras: la vista debe usar la respuesta.
    const estado = await estadoComparado(resultado('7123456', '5000000', '5000000'), resultado('1234567890123456789', '2234567890123456789', '1234567890123456789'));
    const html = renderizarConcursos(estado, { textos, ejemplos: [ejemplo] });
    const separador = new Intl.NumberFormat(textos.localizacion).formatToParts(1.5).find((p) => p.type === 'decimal').value;
    assert.ok(html.includes(textos.traducir('concursos.suma_apartados')));
    assert.ok(html.includes(textos.traducir('concursos.tope_total')));
    assert.ok(html.includes(textos.traducir('concursos.total_tras_tope')));
    assert.ok(html.includes(`data-concurso-suma="antes">7${separador}123456</dd>`));
    assert.match(html, /data-concurso-tope="antes">5<\/dd>/u);
    assert.match(html, /data-concurso-total="antes">5<\/dd>/u);
    const exacto = `${textos.numero(1234567890123n)}${separador}456789`;
    const maximo = `${textos.numero(2234567890123n)}${separador}456789`;
    assert.ok(html.includes(`data-concurso-suma="despues">${exacto}</dd>`));
    assert.ok(html.includes(`data-concurso-tope="despues">${maximo}</dd>`));
    assert.ok(html.includes(`data-concurso-total="despues">${exacto}</dd>`));
    assert.deepEqual(textos.faltantes, []);
  });

  test(`Concursos distingue un cero completo de una suma parcial pendiente (${idioma})`, async () => {
    const textos = await cargarTextos('baremo-concursos', { idioma });
    const estado = await estadoComparado(resultado('0', '0', '0'), resultado('7777777', '0', null, false));
    const html = renderizarConcursos(estado, { textos, ejemplos: [ejemplo] });
    for (const dato of ['suma', 'tope', 'total']) {
      assert.ok(html.includes(`data-concurso-${dato}="antes">0</dd>`));
      assert.ok(!html.includes(`data-concurso-${dato}="despues"`));
    }
    assert.ok(html.includes(textos.traducir('concursos.pendiente_datos')));
    assert.ok(html.includes(textos.traducir('concursos.incidencia_dato_no_disponible', { familia: textos.traducir('concursos.familia_grado') })));
    assert.deepEqual(textos.faltantes, []);
  });
}
