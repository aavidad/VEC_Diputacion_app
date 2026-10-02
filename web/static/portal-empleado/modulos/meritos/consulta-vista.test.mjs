import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../../comun/textos.js";
import { crearTextosConsultaMerito } from "./consulta-i18n.js";
import { montarConsultaMeritoPropio, renderizarConsultaMerito } from "./consulta-vista.js";
import { resultadoPrueba, noEncontradaPrueba, pendiente, raizPrueba } from "./consulta-prueba.test-helper.mjs";

const textos = crearTextosConsultaMerito(await cargarTextos("meritos-consulta", { idioma: "es", porDefecto: "es" }));

test("ficha actual con estado, versión, fechas Madrid y evidencias opacas sin acciones de modificación", () => {
  const resultado = resultadoPrueba("hecho:propio-a", '<Curso & "gestión">');
  const html = renderizarConsultaMerito({ estado: "resultado", resultado, hechoRef: "hecho:propio-a" }, textos);
  assert.match(html, /&lt;Curso &amp; &quot;gestión&quot;&gt;/u);
  assert.match(html, /Acreditado/u); assert.match(html, /Versión consultada/u);
  assert.match(html, /2 oct 2026, 10:40/u);
  assert.match(html, /Sin fecha final registrada/u);
  assert.match(html, /1 referencia de evidencia/u);
  assert.doesNotMatch(html, /<input|<select|download|persona_ref|actor_ref/u);
  const detalle = html.slice(html.indexOf('<details class="consulta-merito-tecnico"'));
  assert.match(detalle, /<summary>Ver detalle técnico de la consulta<\/summary>/u);
  assert.match(detalle, /Recibo de consulta/u); assert.match(detalle, /motivo:verificacion/u);
  assert.doesNotMatch(html.slice(0, html.indexOf('<details class="consulta-merito-tecnico"')), /motivo:verificacion|consulta:1/u);
});

test("los estados sin datos retiran cualquier resultado anterior; 404 HTTP no se confunde con no encontrada autorizada", () => {
  for (const estado of ["inicial", "cargando", "denegada", "error"]) {
    const html = renderizarConsultaMerito({ estado, resultado: resultadoPrueba(), hechoRef: "hecho:propio-a" }, textos);
    assert.doesNotMatch(html, /Curso de gestión|documento:curso|consulta:1/u);
    assert.doesNotMatch(html, /\b(?:401|403|404|503)\b/u);
    assert.match(html, estado === "cargando" ? /aria-busy="true"/u : /aria-busy="false"/u);
  }
  const html = renderizarConsultaMerito({ estado: "no_encontrada", resultado: noEncontradaPrueba(), hechoRef: "hecho:propio-a" }, textos);
  assert.match(html, /Mérito no encontrado/u); assert.match(html, /consulta:1/u); assert.doesNotMatch(html, /Curso de gestión/u);
});

test("montaje: la respuesta de la selección anterior no reemplaza la ficha actual", async () => {
  const raiz = raizPrueba(); const a = pendiente(); const b = pendiente(); const recibidas = [];
  const lector = { consultar(entrada) { recibidas.push(entrada); return entrada.hechoRef === "hecho:a" ? a.promesa : b.promesa; } };
  const vista = montarConsultaMeritoPropio({ raiz, lector, hechoRef: "hecho:a", textos });
  const segunda = vista.seleccionar("hecho:b");
  assert.equal(recibidas[0].signal.aborted, true);
  b.resolver(resultadoPrueba("hecho:b", "Curso actual")); await segunda;
  a.resolver(resultadoPrueba("hecho:a", "Curso anterior")); await vista.preparada;
  assert.match(raiz.children[0].innerHTML, /Curso actual/u); assert.doesNotMatch(raiz.children[0].innerHTML, /Curso anterior/u);
  vista.desmontar();
});

test("desmontar cancela la consulta y descarta la respuesta pendiente", async () => {
  const raiz = raizPrueba(); const diferida = pendiente(); let signal;
  const vista = montarConsultaMeritoPropio({ raiz, hechoRef: "hecho:a", textos, lector: { consultar(entrada) { signal = entrada.signal; return diferida.promesa; } } });
  const nodo = raiz.children[0]; const antes = nodo.innerHTML;
  vista.desmontar(); assert.equal(signal.aborted, true); assert.equal(raiz.children.length, 0);
  diferida.resolver(resultadoPrueba("hecho:a")); await vista.preparada;
  assert.equal(nodo.innerHTML, antes); assert.equal(nodo.listeners.size, 0);
});

test("cada actualización hace otra lectura y la denegación posterior elimina la ficha y el recibo", async () => {
  const raiz = raizPrueba(); let llamadas = 0; const avisos = [];
  const lector = { async consultar() { llamadas++; if (llamadas === 3) throw Object.assign(new Error(), { codigo: "denegada", data: resultadoPrueba() }); const valor = resultadoPrueba(); valor.recibo_consulta.referencia = `consulta:${llamadas}`; return valor; } };
  const vista = montarConsultaMeritoPropio({ raiz, lector, hechoRef: "hecho:propio-a", textos, anunciar: (...v) => avisos.push(v) });
  await vista.preparada; assert.match(raiz.children[0].innerHTML, /consulta:1/u);
  await vista.actualizar(); assert.match(raiz.children[0].innerHTML, /consulta:2/u);
  await vista.actualizar(); assert.match(raiz.children[0].innerHTML, /Consulta no autorizada/u);
  assert.doesNotMatch(raiz.children[0].innerHTML, /consulta:2|Curso de gestión/u);
  assert.deepEqual(avisos.at(-1), ["Consulta no autorizada", "error"]);
  assert.equal(llamadas, 3); vista.desmontar();
});

test("sin selección no consulta; los fallos temporales permiten reintentar con lectura nueva", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  const vista = montarConsultaMeritoPropio({ raiz, textos, lector: { async consultar() { llamadas++; if (llamadas === 1) throw new Error("private transport content"); return resultadoPrueba(); } } });
  await vista.preparada; assert.equal(llamadas, 0); assert.match(raiz.children[0].innerHTML, /Seleccione un mérito/u);
  await vista.seleccionar("hecho:propio-a"); assert.match(raiz.children[0].innerHTML, /Volver a consultar/u); assert.doesNotMatch(raiz.children[0].innerHTML, /private transport/u);
  await vista.actualizar(); assert.match(raiz.children[0].innerHTML, /Curso de gestión/u); assert.equal(llamadas, 2);
  await vista.seleccionar(); assert.match(raiz.children[0].innerHTML, /Seleccione un mérito/u); assert.doesNotMatch(raiz.children[0].innerHTML, /Curso de gestión/u);
  vista.desmontar();
});
