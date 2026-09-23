import test from "node:test";
import assert from "node:assert/strict";

import { consultarEstadisticas } from "./cliente-http-estadisticas.js";

const detallePrivado = "DSN privado: postgres://usuario:secreto@interno";

test("el error privado de transporte queda fuera de la respuesta pública", async () => {
  const resultado = await consultarEstadisticas({}, {
    fetchImpl: async () => { throw new Error(detallePrivado); },
  });

  assert.deepEqual(resultado, {
    ok: false,
    status: 0,
    codigo: "error_red",
    mensaje: "No se pudo conectar con el servicio de estadísticas.",
  });
  assert.equal(JSON.stringify(resultado).includes(detallePrivado), false);
});

test("JSON inválido y contrato rechazado muestran un error público estable", async () => {
  for (const json of [
    async () => { throw new SyntaxError(detallePrivado); },
    async () => ({ data: { nombre: detallePrivado } }),
  ]) {
    const resultado = await consultarEstadisticas({}, {
      fetchImpl: async () => ({ ok: true, status: 200, json }),
    });

    assert.deepEqual(resultado, {
      ok: false,
      status: 200,
      codigo: "respuesta_invalida",
      mensaje: "La respuesta del servicio de estadísticas no es válida.",
    });
    assert.equal(JSON.stringify(resultado).includes(detallePrivado), false);
  }
});

test("la señal opcional llega intacta a fetch y conserva las opciones HTTP", async () => {
  const controlador = new AbortController();
  let opcionesRecibidas;
  const resultado = await consultarEstadisticas({ periodo: "anual" }, {
    signal: controlador.signal,
    fetchImpl: async (_url, opciones) => {
      opcionesRecibidas = opciones;
      return { ok: false, status: 403 };
    },
  });

  assert.equal(opcionesRecibidas.signal, controlador.signal);
  assert.equal(opcionesRecibidas.method, "GET");
  assert.equal(opcionesRecibidas.credentials, "same-origin");
  assert.deepEqual(opcionesRecibidas.headers, { Accept: "application/json" });
  assert.equal(resultado.codigo, "acceso_denegado");
});
