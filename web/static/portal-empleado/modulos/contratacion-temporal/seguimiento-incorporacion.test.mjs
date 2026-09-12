import assert from "node:assert/strict";
import test from "node:test";
import { montarSeguimientoIncorporacion } from "./seguimiento-incorporacion.js";

const recibo = Object.freeze({
  esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2",
  expediente_ref: "expediente:ct:001", solicitud_personal_ref: "solicitud:personal:001",
  relacion_ref: "relacion:personal:001", recibo_ref: "recibo:ct:original",
  actuacion_ref: "actuacion:ct:001", registrada_en: "2026-09-10T10:00:00Z",
  periodo_incorporacion: { desde: "2026-09-11T00:00:00Z", hasta: "2026-09-12T00:00:00Z" },
  version_solicitud_personal: 1, version_actual_expediente: 2,
  seguimiento_ref: "seguimiento:ct:001", version_seguimiento_anterior: 0,
  version_seguimiento_resultante: 1, auditoria_ref: "auditoria:ct:001",
  outbox_ref: "outbox:ct:001", ejercicio_sintetico: true,
  firma_oficial: false, eficacia_administrativa: false,
});

const seguimiento = () => ({
  esquema: "vec.contratacion-temporal.seguimiento-incorporacion.v2",
  alcance: "original_incorporacion", recibo_incorporacion_ref: recibo.recibo_ref,
  expediente_ref: recibo.expediente_ref, version_expediente: 2,
  seguimiento_ref: recibo.seguimiento_ref, version_seguimiento: 1,
  estado_clave: "incorporada", periodo: recibo.periodo_incorporacion,
  registrado_en: recibo.registrada_en, actuaciones: [{
    actuacion_ref: recibo.actuacion_ref, transicion_clave: "confirmar_incorporacion",
    estado_origen: "pendiente", estado_destino: "incorporada",
    efectivo_en: recibo.periodo_incorporacion.desde, registrada_en: recibo.registrada_en,
    documentos: [{ tipo_clave: "justificante", referencia: "documento:ct:001" }],
  }], ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false,
});

function crearRaiz() {
  const eventos = new Map();
  return {
    innerHTML: "", addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    pulsarConsulta() {
      const control = { closest: (selector) => selector === "[data-ct-seguimiento-consultar]" ? control : null };
      return eventos.get("click")?.({ target: control });
    },
  };
}

test("seguimiento: consulta y muestra solo el vínculo original del recibo confirmado", async () => {
  const raiz = crearRaiz(), llamadas = [];
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo, mensajes: {}, cliente: {
    async consultar(expediente_ref, opciones) { llamadas.push([expediente_ref, opciones]); return seguimiento(); },
  } });
  await raiz.pulsarConsulta();
  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0][0], recibo.expediente_ref);
  assert.ok(llamadas[0][1].signal instanceof AbortSignal);
  assert.match(raiz.innerHTML, /documento:ct:001/u);
  assert.match(raiz.innerHTML, /pendiente → incorporada/u);
  assert.match(raiz.innerHTML, /<time datetime="2026-09-10T10:00:00Z">10 sept 2026, 12:00:00<\/time>/u);
  assert.doesNotMatch(raiz.innerHTML, /No se ha podido consultar el seguimiento original/u);
  destruir();
});

test("seguimiento: el período conserva sus fechas civiles UTC y los instantes usan Europe/Madrid", async () => {
  const reciboCambioDia = Object.freeze({
    ...recibo,
    registrada_en: "2026-01-10T23:00:00Z",
    periodo_incorporacion: { desde: "2026-01-10T23:00:00Z", hasta: "2026-01-11T23:00:00Z" },
  });
  const datos = seguimiento();
  datos.periodo = reciboCambioDia.periodo_incorporacion;
  datos.registrado_en = reciboCambioDia.registrada_en;
  datos.actuaciones[0].efectivo_en = reciboCambioDia.periodo_incorporacion.desde;
  datos.actuaciones[0].registrada_en = reciboCambioDia.registrada_en;
  const raiz = crearRaiz();
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo: reciboCambioDia, mensajes: {}, cliente: { async consultar() { return datos; } } });
  await raiz.pulsarConsulta();
  assert.match(raiz.innerHTML, /10\/01\/2026<\/time> — <time datetime="2026-01-11T23:00:00Z">11\/01\/2026/u);
  assert.match(raiz.innerHTML, /<time datetime="2026-01-10T23:00:00Z">10\/01\/2026<\/time>/u);
  assert.match(raiz.innerHTML, /<time datetime="2026-01-10T23:00:00Z">11 ene 2026, 0:00:00<\/time>/u);
  destruir();
});

test("seguimiento: sin recibo V2 confirmado mantiene el control inactivo y no consulta", async () => {
  const raiz = crearRaiz();
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo: null, mensajes: {}, cliente: {
    consultar() { assert.fail("consulta inesperada"); },
  } });
  assert.match(raiz.innerHTML, /data-ct-seguimiento-consultar disabled/u);
  assert.match(raiz.innerHTML, /solo se consulta desde un recibo V2 confirmado/u);
  await raiz.pulsarConsulta();
  destruir();
});

test("seguimiento: desmontar cancela la consulta y descarta la respuesta tardía", async () => {
  const raiz = crearRaiz(); let resolver, signal;
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo, mensajes: {}, cliente: {
    consultar(_expediente_ref, opciones) {
      signal = opciones.signal;
      return new Promise((resolve) => { resolver = resolve; });
    },
  } });
  const consulta = raiz.pulsarConsulta();
  destruir();
  assert.equal(signal.aborted, true);
  resolver(seguimiento()); await consulta;
  assert.equal(raiz.innerHTML, "");
});

test("seguimiento: error de consulta no muestra datos ni reactiva una actuación", async () => {
  const raiz = crearRaiz();
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo, mensajes: {}, cliente: {
    async consultar() { throw new Error("detalle interno"); },
  } });
  await raiz.pulsarConsulta();
  assert.match(raiz.innerHTML, /No se ha podido consultar el seguimiento original/u);
  assert.doesNotMatch(raiz.innerHTML, /detalle interno|seguimiento:ct:001|documento:ct:001/u);
  assert.match(raiz.innerHTML, /data-ct-seguimiento-consultar/u);
  destruir();
});
