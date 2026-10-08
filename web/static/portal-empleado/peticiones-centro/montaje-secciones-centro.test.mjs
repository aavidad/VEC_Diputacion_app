import assert from "node:assert/strict";
import test from "node:test";
import { crearMontajeSeccionesCentro } from "./montaje-secciones-centro.js";
import * as incorporaciones from "./incorporaciones-centro.js";
import * as cancelaciones from "./cancelaciones-centro.js";

function contenedor() {
  const eventos = new Map();
  return { innerHTML: "", hidden: true, eventos, ownerDocument: { addEventListener() {} },
    addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}
const esperar = () => new Promise((r) => setImmediate(r));

test("recupera secciones una vez y descarta respuestas del contexto retirado", async () => {
  const i = contenedor(), c = contenedor();
  const ayudaI = [{ hidden: false }], ayudaC = [{ hidden: false }];
  let consultas = 0;
  const pendientes = [];
  const documento = { querySelector: (s) => s === "#incorporaciones-centro" ? i : c,
    querySelectorAll: (s) => s.includes("incorporacion") ? ayudaI : ayudaC };
  const montaje = crearMontajeSeccionesCentro({ documento,
    incorporaciones: { ...incorporaciones, instalarAyudaIncorporacionesCentro() {},
      crearClienteIncorporacionesCentro: () => incorporaciones.crearClienteIncorporacionesCentro(() => {
        consultas++;
        return new Promise((r) => pendientes.push(r));
      }) },
    cancelaciones: { ...cancelaciones, instalarAyudaCancelacionesCentro() {} },
  });
  montaje.actualizar({ incorporaciones: false, cancelaciones: true });
  assert.equal(consultas, 0);
  assert.ok(ayudaI[0].hidden && ayudaC[0].hidden);
  montaje.actualizar({ incorporaciones: true, cancelaciones: true });
  montaje.actualizar({ incorporaciones: true, cancelaciones: true });
  assert.equal(consultas, 1, "dos secciones y dos renders comparten una sola petición");
  assert.ok(!ayudaI[0].hidden && !ayudaC[0].hidden);
  montaje.actualizar();
  pendientes.shift()({ ok: true, text: async () => JSON.stringify({ data: { esquema: "vec.contratacion-temporal.incorporaciones-centro.v1", expedientes: [], puede_confirmar: true } }) });
  await esperar(); await esperar();
  assert.ok(i.hidden && c.hidden);
  assert.equal(i.innerHTML + c.innerHTML, "");
  assert.equal(i.eventos.size + c.eventos.size, 0);
  montaje.actualizar({ incorporaciones: true, cancelaciones: true });
  assert.equal(consultas, 2, "una recuperación no reutiliza el cliente del contexto retirado");
  pendientes.shift()({ ok: true, text: async () => JSON.stringify({ data: { esquema: "vec.contratacion-temporal.incorporaciones-centro.v1", expedientes: [], puede_confirmar: true } }) });
  await esperar(); await esperar();
  assert.ok(!i.hidden && !c.hidden);
  montaje.desmontar();
  assert.ok(i.hidden && c.hidden);
});
