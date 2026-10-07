import assert from "node:assert/strict";
import test from "node:test";
import { bolsasVigentesParaLlamamiento, instalarSelectorLlamamientos, renderizarPantallaLlamamientos, renderizarSelectorLlamamientos } from "./portal-llamamientos-selector.js";

const opciones = (estadoBolsas) => ({
  estadoBolsas,
  encabezadoVista: (_miga, titulo) => `<h2>${titulo}</h2>`,
  escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;"),
});

test("la selección muestra solo bolsas vigentes de la lectura real y conserva la referencia", () => {
  const estado = { carga: "listo", datos: { bolsas: [
    { bolsa_ref: "bolsa:vigente", categoria: "Administración", vigente_hasta: null },
    { bolsa_ref: "bolsa:antigua", categoria: "Antigua", vigente_hasta: "2026-09-01" },
  ] } };
  assert.deepEqual(bolsasVigentesParaLlamamiento(estado).map((bolsa) => bolsa.bolsa_ref), ["bolsa:vigente"]);
  const html = renderizarSelectorLlamamientos(opciones(estado));
  assert.match(html, /data-elegir-bolsa-llamamiento="bolsa:vigente"/);
  assert.doesNotMatch(html, /bolsa:antigua/);
  assert.doesNotMatch(html, /#bolsa\/resumen/);
});

test("lectura vacía, denegada y fallida tienen estados distintos sin acción de llamamiento", () => {
  const vacio = renderizarSelectorLlamamientos(opciones({ carga: "listo", datos: { bolsas: [] } }));
  const denegado = renderizarSelectorLlamamientos(opciones({ carga: "denegado" }));
  const error = renderizarSelectorLlamamientos(opciones({ carga: "error" }));
  for (const html of [vacio, denegado, error]) assert.doesNotMatch(html, /data-elegir-bolsa-llamamiento/);
  assert.match(vacio, /No hay bolsas de trabajo activas/);
  assert.match(denegado, /Acceso denegado/);
  assert.match(error, /data-bolsa-accion="reintentar-bolsas"/);
});

test("una categoría recibida se escapa en texto y atributo", () => {
  const html = renderizarSelectorLlamamientos(opciones({ carga: "listo", datos: { bolsas: [
    { bolsa_ref: 'bolsa:"ref', categoria: '<Prueba> "A"', vigente_hasta: null },
  ] } }));
  assert.doesNotMatch(html, /<Prueba>/);
  assert.match(html, /&lt;Prueba>/);
  assert.match(html, /data-elegir-bolsa-llamamiento="bolsa:&quot;ref"/);
});

test("un error al leer la bolsa permite reintentar o elegir otra sin salir del llamamiento", () => {
  const html = renderizarPantallaLlamamientos({
    estado: { bolsaSeleccionada: "bolsa:vigente", llamamientoDesdeMenu: true, datosCandidatos: { carga: "error" } },
    encabezadoVista: opciones(null).encabezadoVista,
    escaparHTML: opciones(null).escaparHTML,
    presentador: { renderizarSoloBolsas: () => assert.fail("no se debe mostrar la ficha B5") },
  });
  assert.match(html, /data-reintentar-bolsa-llamamiento/);
  assert.match(html, /data-cambiar-bolsa-llamamiento/);
  assert.doesNotMatch(html, /#bolsa\/resumen|data-vista="resumen"/);
});

test("elegir una bolsa inicia el B7 existente en la misma vista y cancelar devuelve la selección", async () => {
  let escuchar;
  let inicios = 0;
  let repintados = 0;
  const estado = { vista: "llamamientos", bolsaSeleccionada: "", llamamientoDesdeMenu: false,
    datosBolsas: { carga: "listo", datos: { bolsas: [{ bolsa_ref: "bolsa:vigente", vigente_hasta: null }] } } };
  instalarSelectorLlamamientos({
    documento: { addEventListener: (_tipo, accion) => { escuchar = accion; } },
    estado,
    controladorBolsas: { async cargarCandidatosBolsa(ref) {
      estado.bolsaSeleccionada = ref;
      estado.datosCandidatos = { carga: "listo" };
    } },
    actualizarVistaBolsa: () => { repintados++; },
    porId: () => ({ querySelector: () => ({ click: () => { inicios++; } }) }),
  });
  const boton = { dataset: { elegirBolsaLlamamiento: "bolsa:vigente" },
    closest: (selector) => selector === "#espacio-trabajo" ? {} : selector === "[data-elegir-bolsa-llamamiento]" ? boton : null };
  escuchar({ target: boton, preventDefault() {} });
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(inicios, 1);
  assert.equal(estado.vista, "llamamientos");
  assert.equal(estado.bolsaSeleccionada, "bolsa:vigente");
  escuchar({ target: { closest: (selector) => selector.includes('cancelar-b7') ? {} : null } });
  assert.equal(estado.bolsaSeleccionada, "");
  assert.equal(estado.llamamientoDesdeMenu, false);
  assert.equal(repintados, 1);
});
