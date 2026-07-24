import assert from "node:assert/strict";
import test from "node:test";
import {
  crearEstadoBorradores,
  limpiarEstadoBorradoresRevocado,
} from "./portal-borradores-estado.js";

test("la revocación elimina todas las proyecciones y controles sensibles", () => {
  const estado = crearEstadoBorradores();
  Object.assign(estado, {
    opciones: { capacidades: { consultar: true } },
    lista: { elementos: [{ referencia: "privada" }] },
    detalle: { referencia: "privada" },
    editor: { contenido: "privado" },
    recibo: { referencia: "privada" },
    conflictoRemoto: { referencia: "privada" },
    referenciaSeleccionada: "privada",
    claveIdempotencia: "secreta",
    sucio: true,
    guardando: true,
  });
  const errorLista = Object.freeze({ codigo: "sesion_denegada" });
  limpiarEstadoBorradoresRevocado(estado, errorLista);
  assert.equal(estado.faseLista, "error");
  assert.equal(estado.errorLista, errorLista);
  for (const campo of ["opciones", "lista", "detalle", "editor", "recibo", "conflictoRemoto"]) {
    assert.equal(estado[campo], null, campo);
  }
  assert.equal(estado.referenciaSeleccionada, "");
  assert.equal(estado.claveIdempotencia, "");
  assert.equal(estado.sucio, false);
  assert.equal(estado.guardando, false);
});
