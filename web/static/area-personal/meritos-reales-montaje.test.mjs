import test from "node:test";
import assert from "node:assert/strict";
import { montarMeritosReales } from "./meritos-reales-montaje.js";

function contenedorFalso() {
  const cabeza = [];
  const documento = { head: { append: (nodo) => cabeza.push(nodo) },
    createElement: (tipo) => ({ tipo, dataset: {}, remove() {}, setAttribute() {} }) };
  return { cabeza, contenedor: { ownerDocument: documento, isConnected: true,
    innerHTML: "", replaceChildren() { this.innerHTML = ""; },
    addEventListener() {}, removeEventListener() {}, contains() { return true; } } };
}

test("la cartera abierta consulta una vez y pide motivos en el idioma elegido", async () => {
  const { cabeza, contenedor } = contenedorFalso();
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return { status: 200, ok: true, async json() { return { data: { hechos: [], total: 0,
      filtro: "todos", motivos_retirada: [{ etiqueta: "Incorrect document",
        motivo: { entrada_clave: "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } }],
      corte_en: "2026-10-12T16:00:00Z", recibo_acceso: { auditoria_ref: "aud:1" } } }; } };
  };
  const vista = montarMeritosReales({ contenedor, fetchImpl, idioma: "en",
    ubicacion: new URL("https://ejemplo.local/area-personal/?lang=en"), historial: { replaceState() {} } });
  try {
    await vista.preparada;
    assert.equal(llamadas.length, 1);
    assert.equal(llamadas[0].ruta, "/api/vec/meritos/cartera?filtro=todos&limite=25&lang=en");
    assert.equal(llamadas[0].opciones.credentials, "omit");
    assert.match(contenedor.innerHTML, /class="panel meritos-reales"/u);
    assert.equal(cabeza.length, 1);
  } finally { vista.desmontar(); }
});
