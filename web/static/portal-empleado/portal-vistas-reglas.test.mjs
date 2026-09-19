import assert from "node:assert/strict";
import test from "node:test";
import { crearVistaReglas } from "./portal-vistas-reglas.js";

function utilidades() {
  const escaparHTML = (valor) => String(valor ?? "").replace(/[&<>"]/g, (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[caracter]);
  return {
    escaparHTML, numero: (valor) => String(valor), fecha: (valor) => String(valor ?? ""),
    chip: (estado) => `<span>${escaparHTML(estado)}</span>`,
    tabla: ({ filas, vacio }) => filas.length ? `<table>${filas.flat().join("|")}</table>` : `<p>${escaparHTML(vacio)}</p>`,
    kpi: (_sigla, valor, etiqueta) => `${etiqueta}:${valor}`,
    encabezadoVista: (_miga, titulo, descripcion) => `<h1>${titulo}</h1><p>${descripcion}</p>`,
    avisoPresentacion: (texto) => `<aside>${escaparHTML(texto)}</aside>`,
    fuentePresentacion: () => "<span>Datos sintéticos · Memoria volátil</span>",
  };
}
test("reglas conserva versión y procedencia sin habilitar edición", () => {
  const html = crearVistaReglas(utilidades()).renderizarReglas({
    reglas: [{ nombre: "Experiencia", ambito: "Bolsa demo", version: "v3", vigencia: "Desde hoy", estado: "Borrador", procedencia: "Escenario sintético" }], criterios_baremo: [],
  });
  assert.match(html, /v3/); assert.match(html, /Escenario sintético/);
  assert.match(html, /Datos sintéticos/); assert.match(html, /Edición de versiones no conectada/);
  assert.match(html, /data-comando="guardar-reglas-baremo"/);
  assert.match(html, /disabled aria-disabled="true"/);
});
test("reglas representa carga, error, denegación y vacío sin revelar datos", () => {
  const vista = crearVistaReglas(utilidades());
  assert.match(vista.renderizarReglas({}, { fase: "cargando" }), /Cargando versiones/);
  assert.match(vista.renderizarReglas({}, { fase: "error", detalle: "Error de red" }), /Error de red/);
  assert.match(vista.renderizarReglas({}, { fase: "denegado" }), /Acceso a reglas denegado/);
  assert.match(vista.renderizarReglas({ reglas: [], criterios_baremo: [] }), /No hay versiones autorizadas que mostrar/);
});
