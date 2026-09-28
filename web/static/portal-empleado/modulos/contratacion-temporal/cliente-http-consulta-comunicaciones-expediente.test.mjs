import assert from "node:assert/strict";
import test from "node:test";
import {
  crearConsultaComunicacionesExpedienteClienteHTTP, RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE,
  validarPaginaComunicacionesExpediente, instanteOrdenComunicacion,
} from "./cliente-http-consulta-comunicaciones-expediente.js";

const expediente_ref = "expediente:ct140:001";
const fila = (n, registrada_en = `2026-09-24T10:${String(n).padStart(2, "0")}:00Z`) => ({
  organizacion_ref: "organizacion:ct140", expediente_ref,
  llamamiento_ref: `llamamiento:ct140:${n}`, comunicacion_ref: `comunicacion:ct140:${n}`,
  version: 2, estado: "registrada_localmente", registrada_en,
  recibo_comunicacion_ref: `recibo:comunicacion:${n}`, antecedente_tipo: "seleccion_confirmada",
  recibo_antecedente_ref: `recibo:seleccion:${n}`,
});
const cursor = `comunicacion:ct140:10#${"a".repeat(64)}`;

test("GET CT140 fija límite 10 y cursor, sin organización ni clave en URL", async () => {
  const llamadas = [];
  const cliente = crearConsultaComunicacionesExpedienteClienteHTTP({
    validarOpciones: ({ signal }) => ({ signal }),
    ejecutar: async (opciones) => {
      llamadas.push(opciones);
      return opciones.validarRespuesta({ expediente_ref, comunicaciones: [fila(1)] });
    },
  });
  const controlador = new AbortController();
  await cliente.consultarComunicacionesExpediente({ expediente_ref }, { signal: controlador.signal });
  await cliente.consultarComunicacionesExpediente({ expediente_ref, cursor }, { signal: controlador.signal });
  assert.equal(llamadas[0].ruta, `${RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE}?expediente_ref=expediente%3Act140%3A001&limite=10`);
  assert.equal(llamadas[1].ruta, `${llamadas[0].ruta}&cursor=${encodeURIComponent(cursor)}`);
  assert.ok(llamadas.every((opciones) => opciones.metodo === "GET" && opciones.efecto === false
    && opciones.estadoEsperado === 200 && opciones.signal === controlador.signal
    && !Object.hasOwn(opciones, "entrada") && !/clave_idempotencia|organizacion_ref/u.test(opciones.ruta)));
});

test("CT140 valida diez campos, orden microsegundo, cursor y referencias del expediente", () => {
  const pagina = { expediente_ref, comunicaciones: [fila(1, "2026-09-24T10:00:00.12345Z"),
    fila(2, "2026-09-24T10:00:00.123450Z")] };
  assert.equal(instanteOrdenComunicacion(pagina.comunicaciones[0].registrada_en),
    instanteOrdenComunicacion(pagina.comunicaciones[1].registrada_en));
  assert.equal(validarPaginaComunicacionesExpediente(pagina, { expediente_ref }).comunicaciones.length, 2);
  for (const cambio of [{ expediente_ref: "expediente:ajeno" }, { version: 3 },
    { estado: "otro" }, { organizacion_ref: ["organizacion:ct140"] },
    { antecedente_tipo: "inventado" }, { recibo_antecedente_ref: "persona@example.invalid" },
    { actor_ref: "actor:ajeno" }]) {
    assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
      comunicaciones: [{ ...fila(1), ...cambio }] }, { expediente_ref }), TypeError);
  }
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: Array.from({ length: 11 }, (_, i) => fila(i + 1)) }, { expediente_ref }), TypeError);
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: [fila(2), fila(1)] }, { expediente_ref }), TypeError);
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: [fila(1)], siguiente_cursor: cursor }, { expediente_ref }), TypeError);
  assert.throws(() => validarPaginaComunicacionesExpediente({ expediente_ref,
    comunicaciones: Array(2) }, { expediente_ref }), TypeError);
});
