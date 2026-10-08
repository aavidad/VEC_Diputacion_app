import assert from "node:assert/strict";
import test from "node:test";
import { renderizarListaPeticiones } from "./vista-expedientes-lista.js";

const filtro = Object.freeze({ texto: "", fase: "", centro: "", categoria: "", mostrar: "todas" });
const estado = Object.freeze({ cuadro: { expedientes: [], generado_en: "2026-10-08T10:00:00Z",
  paginacion: { cursor_siguiente: "cursor_de_prueba" } }, filtros: {} });
const ayudas = Object.freeze({ numeroVisible: (numero) => numero,
  centroVisible: (referencia) => ({ etiqueta: referencia, referencia }) });
const traducir = (clave) => clave;

test("una página con total exacto V2 no anuncia un recuento parcial", () => {
  const opciones = { tituloConjunto: true, opcionesFaseServidor: [],
    filtroResultados: filtro, totalConjunto: 101, totalConjuntoExacto: true };
  const exacto = renderizarListaPeticiones(estado, traducir, filtro, ayudas, "", opciones);
  assert.doesNotMatch(exacto, /lista_recuento_parcial/u);
  const v1 = renderizarListaPeticiones(estado, traducir, filtro, ayudas, "", {
    ...opciones, totalConjuntoExacto: false });
  assert.match(v1, /lista_recuento_parcial/u);
  const sinTotal = renderizarListaPeticiones(estado, traducir, filtro, ayudas, "", {
    ...opciones, totalConjunto: null });
  assert.match(sinTotal, /lista_recuento_parcial/u);
});
