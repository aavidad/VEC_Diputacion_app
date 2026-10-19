import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";

const portal = await readFile(new URL("./portal.js", import.meta.url), "utf8");
const funcion = portal.slice(portal.indexOf("function montarVistaBolsa("),
  portal.indexOf("function aplicarRutaCandidatosBolsa("));

// Contenedor mínimo: guarda el HTML pintado y los nodos añadidos por la vista.
function contenedorFalso() {
  return {
    innerHTML: "", hijos: [],
    replaceChildren() { this.innerHTML = ""; this.hijos = []; },
    append(nodo) { this.hijos.push(nodo); },
    querySelector(selector) {
      return selector === "[data-bolsa-documentos]" ? this.hijos.find((h) => h.documentos) ?? null : null;
    },
  };
}

test("Documentos de Bolsas retira el aviso de carga al montar la vista", async () => {
  const estado = { vista: "documentos", datosBolsas: { carga: "cargando" } };
  const contexto = {
    estado, vistaDocumentosBolsa: null, AbortController,
    vistaBolsaPendienteNoCompuesta: () => false,
    textoPortal: (clave) => clave,
    recursosVistas: new Map([["documentosBolsa", { montarVistaDocumentosBolsa: async ({ raiz }) => {
      raiz.append({ documentos: true });
      return { desmontar() {} };
    } }]]),
    anunciar() {}, navegar() {}, controladorBolsas: null,
  };
  runInNewContext(`${funcion}; this.montarVistaBolsa = montarVistaBolsa;`, contexto);
  const contenedor = contenedorFalso();
  contexto.montarVistaBolsa("documentos", contenedor);
  assert.match(contenedor.innerHTML, /estado_modulo_comprobando/);
  estado.datosBolsas = { carga: "listo", datos: { bolsas: [] } };
  contexto.montarVistaBolsa("documentos", contenedor);
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(contenedor.innerHTML, "");
  assert.equal(contenedor.hijos.length, 1);
  assert.equal(contenedor.hijos[0].documentos, true);
});
