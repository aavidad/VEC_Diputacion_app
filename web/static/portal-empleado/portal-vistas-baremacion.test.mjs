import assert from "node:assert/strict";
import test from "node:test";
import { crearVistasBaremacion } from "./portal-vistas-baremacion.js";

const vistas = crearVistasBaremacion({
  escaparHTML: (valor) => String(valor),
  numero: (valor) => String(valor),
  chip: (valor) => `<span>${valor}</span>`,
  tabla: ({ filas }) => `<table>${filas.join("")}</table>`,
  kpi: (_sigla, valor) => `<span>${valor}</span>`,
  encabezadoVista: (_seccion, titulo, descripcion, accion = "") => `<h1>${titulo}</h1><p>${descripcion}</p>${accion}`,
  avisoPresentacion: (texto = "") => `<aside>${texto}</aside>`,
  botonOperacion: () => "<button>operación</button>",
  campo: (_etiqueta, control) => control,
  fuentePresentacion: () => "",
});

const datos = {
  meritos_revision: [{ id: "DEMO-MER-1", persona_ref: "DEMO-PER-1", tipo: "Formación", evidencia: "DEMO-DOC-1", declarado: "Dato", puntos: "1", estado: "Pendiente" }],
  criterios_baremo: [{ bloque: "Formación", version: "v-demo" }],
  ranking: [{ posicion: 1, persona_ref: "DEMO-PER-1", experiencia: "0", formacion: "1", otros: "0", total: "1", desempate: "No aplicado", estado: "Provisional" }],
  alegaciones: [{ id: "DEMO-ALE-1", persona_ref: "DEMO-PER-1", objeto: "Dato", registrada: "—", plazo: "—", evidencia: "DEMO-DOC-1", estado: "Pendiente" }],
};

test("las vistas de baremación cierran carga, error y denegación sin exponer datos", () => {
  for (const [vista, renderizar] of [
    ["meritos", vistas.renderizarMeritos],
    ["baremacion", vistas.renderizarBaremacion],
    ["alegaciones", vistas.renderizarAlegaciones],
  ]) {
    for (const [carga, texto] of [
      ["cargando", "Comprobando el acceso y cargando datos"],
      ["error", "No se pudo consultar esta información"],
      ["denegado", "La sesión no dispone de acceso"],
    ]) {
      const html = renderizar(datos, { vistasBaremacion: { [vista]: { carga, error: "Fallo controlado" } } });
      assert.match(html, new RegExp(texto));
      assert.doesNotMatch(html, /DEMO-PER-1/);
    }
  }
});

test("la presentación no habilita efectos y comunica que el ranking es sintético", () => {
  const html = vistas.renderizarBaremacion(datos);
  assert.match(html, /sintéticos/i);
  assert.match(html, /disabled aria-disabled="true"/);
  assert.doesNotMatch(html, /data-accion=/);
  assert.match(vistas.renderizarMeritos({}), /Sin registros/);
});
