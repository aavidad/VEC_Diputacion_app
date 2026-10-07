import assert from "node:assert/strict";
import test from "node:test";
import { actualizarEntrada } from "./entrada.js";

const textos = { traducir: () => "Preparar solicitud" };
function enlace(datos) {
  return { dataset: datos, hidden: true, removeAttribute(nombre) { delete this[nombre]; } };
}

test("la ficha gobernada abre preparación y conserva idioma y convocatoria", () => {
  const nodo = enlace({ convocatoria: "auxiliares-2026", demostracion: "false" });
  actualizarEntrada(nodo, textos, "en");
  assert.equal(nodo.href, "/bolsa/preparacion/?convocatoria=auxiliares-2026&lang=en");
  assert.equal(nodo.hidden, false);
});

test("una ficha de demostración o referencia inválida no abre preparación", () => {
  const nodo = enlace({ convocatoria: "auxiliares-2026", demostracion: "false" });
  actualizarEntrada(nodo, textos, "es");
  nodo.dataset.demostracion = "true";
  actualizarEntrada(nodo, textos, "es");
  assert.equal(nodo.hidden, true);
  assert.equal(nodo.href, undefined);
  nodo.dataset = { convocatoria: "../externo", demostracion: "false" };
  actualizarEntrada(nodo, textos, "es");
  assert.equal(nodo.hidden, true);
});
