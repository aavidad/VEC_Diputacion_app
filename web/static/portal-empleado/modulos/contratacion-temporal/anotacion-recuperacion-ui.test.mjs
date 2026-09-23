import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import test from "node:test";

import { montarFormularioAnotacionAdministrativa } from "./formulario-anotacion-administrativa.js";

const expediente = { expediente_ref: "expediente:anotacion:1", version: 7 };
// El contrato exige un UUID; el marcador deja claro que solo es un fixture sintético.
const clave = "SINTETICO_NO_SECRETO_UUID_V4".replace("SINTETICO_NO_SECRETO_UUID_V4", () => randomUUID());
const recibo = {
  operacion: "registrar_anotacion_administrativa", organizacion_ref: "organizacion:1",
  expediente_ref: expediente.expediente_ref, version_anterior: 7, version_resultante: 8,
  fase_resultante: "nombramiento", estado_resultante: "en_curso",
  seguimiento_original: { seguimiento_ref: "seguimiento:1", version_seguimiento: 1, huella_raiz_seguimiento_sha256: "a".repeat(64) },
  recibo_ref: "recibo:anotacion:1", auditoria_ref: "auditoria:1", evento_ref: "evento:1",
  actor_ref: "actor:1", registrada_en: "2026-09-10T12:00:00Z",
};

function raizFormulario() {
  const eventos = new Map();
  return {
    innerHTML: "",
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    replaceChildren() { this.innerHTML = ""; },
    recuperar(valor) {
      return eventos.get("submit")({
        preventDefault() {},
        target: { dataset: { ctAnotacionRecuperar: "" }, elements: { clave_idempotencia: { value: valor } } },
      });
    },
  };
}

test("un 503 conserva la clave en memoria visible y reintenta la misma recuperación sin duplicar efecto", async () => {
  let resolverPrimera;
  const consultas = [];
  const confirmaciones = [];
  const raiz = raizFormulario();
  const desmontar = montarFormularioAnotacionAdministrativa({
    raiz, expediente,
    cliente: {
      registrar() { assert.fail("la recuperación no debe crear otra anotación"); },
      recuperar(consulta) {
        consultas.push(consulta);
        if (consultas.length === 1) return new Promise((_, reject) => { resolverPrimera = reject; });
        return Promise.resolve(recibo);
      },
    },
    alConfirmar: (r) => confirmaciones.push(r.recibo_ref),
  });

  const primera = raiz.recuperar(clave);
  assert.match(raiz.innerHTML, new RegExp(`name="clave_idempotencia" value="${clave}"[^>]* disabled`, "u"));
  assert.match(raiz.innerHTML, /data-ct-anotacion-recuperar aria-busy="true"/u);
  assert.match(raiz.innerHTML, /class="boton-secundario" disabled/u);
  await raiz.recuperar(clave);
  assert.equal(consultas.length, 1);

  resolverPrimera(Object.assign(new Error("servicio temporalmente no disponible"), { estado: 503 }));
  await primera;
  assert.match(raiz.innerHTML, new RegExp(`name="clave_idempotencia" value="${clave}"`, "u"));
  assert.doesNotMatch(raiz.innerHTML, /class="boton-secundario" disabled/u);
  assert.match(raiz.innerHTML, /No se pudo verificar el recibo/u);

  const claveRestaurada = raiz.innerHTML.match(/name="clave_idempotencia" value="([^"]*)"/u)?.[1];
  await raiz.recuperar(claveRestaurada);
  assert.deepEqual(consultas, [
    { expediente_ref: expediente.expediente_ref, clave_idempotencia: clave },
    { expediente_ref: expediente.expediente_ref, clave_idempotencia: clave },
  ]);
  assert.deepEqual(confirmaciones, [recibo.recibo_ref]);
  assert.match(raiz.innerHTML, /recibo:anotacion:1/u);
  assert.match(raiz.innerHTML, /class="boton-secundario" disabled/u);
  assert.match(raiz.innerHTML, /name="clave_idempotencia" value=""/u);
  await raiz.recuperar(clave);
  assert.equal(consultas.length, 2);
  desmontar();
  assert.equal(raiz.innerHTML, "");
});
