import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import test from "node:test";
import { montarFormularioSubsanacionReparos } from "./formulario-subsanacion-reparos.js";
import { crearClienteSubsanacionReparosHTTP, RUTA_SUBSANACION_REPAROS } from "./cliente-http-subsanacion-reparos.js";
import { MENSAJES_SUBSANACION_REPAROS_ES as textos } from "./i18n-subsanacion-reparos.js";

const contexto = { expediente_ref: "expediente:subsanacion:001", version_esperada: 6 };
const clave = randomUUID();
const recibo = {
  esquema: "vec.contratacion-temporal.recibo-subsanacion-reparos.v1",
  operacion: "registrar_subsanacion", expediente_ref: contexto.expediente_ref,
  version_resultante: 7, fase_resultante: "subsanacion_unidad", estado_resultante: "incidencia",
  recibo_ref: "recibo:subsanacion:001", auditoria_ref: "auditoria:subsanacion:001",
  evento_ref: "evento:subsanacion:001", actor_ref: "actor:subsanacion:001",
  registrada_en: "2026-09-13T08:00:00Z",
};

function escenario(respuestas, confirmar = () => true) {
  const eventos = new Map(), peticiones = [], confirmaciones = [], anuncios = [], intenciones = [], denegaciones = [];
  const documento = { body: { append() {} }, activeElement: null,
    createElement: () => ({ click() {}, remove() {} }) };
  documento.activeElement = documento.body;
  const nodos = new Map();
  let html = "", ultimoFormulario = null, ultimoBoton = null;
  const ultimo = new Map();
  const raiz = {
    ownerDocument: documento,
    get innerHTML() { return html; },
    set innerHTML(valor) {
      if (this.contains(documento.activeElement)) documento.activeElement = documento.body;
      html = valor;
      nodos.clear();
      for (const selector of ["[data-ct-subsanacion-form]", "[data-ct-subsanacion-guardar]", "[data-ct-subsanacion-enviar]", "[data-ct-subsanacion-recuperar]", "[data-ct-subsanacion-recibo]", "[data-ct-subsanacion-form] textarea"]) {
        if (selector === "[data-ct-subsanacion-form] textarea" ? !valor.includes("data-ct-subsanacion-form") : !valor.includes(selector.slice(1, -1))) continue;
        const nodo = {
          focus() { if (nodos.get(selector) === this) documento.activeElement = this; },
          closest(patron) { return patron === selector ? this : null; },
        };
        if (selector === "[data-ct-subsanacion-form]") {
          nodo.elements = { namedItem: () => ({ value: "Corrección del reparo." }) };
          ultimoFormulario = nodo;
        }
        if (selector === "[data-ct-subsanacion-recuperar]") ultimoBoton = nodo;
        nodos.set(selector, nodo);
        ultimo.set(selector, nodo);
      }
    },
    addEventListener(nombre, funcion) { eventos.set(nombre, funcion); },
    removeEventListener(nombre) { eventos.delete(nombre); },
    contains(nodo) { return [...nodos.values()].includes(nodo); },
    querySelector(selector) { return nodos.get(selector) ?? null; },
    replaceChildren() { this.innerHTML = ""; },
  };
  const cliente = crearClienteSubsanacionReparosHTTP({
    validarOpciones: (opciones) => opciones,
    serializarAcotado: JSON.stringify,
    ejecutar: async (peticion) => {
      peticiones.push(peticion);
      const respuesta = await respuestas.shift();
      if (respuesta instanceof Error) throw respuesta;
      return peticion.validarRespuesta(respuesta);
    },
  });
  const desmontar = montarFormularioSubsanacionReparos({
    raiz, cliente, contexto, traducir: (claveTexto) => textos[claveTexto] ?? claveTexto,
    generarClaveIdempotencia: () => clave, confirmarOperacion: confirmar,
    anunciar: (mensaje) => anuncios.push(mensaje),
    alConfirmar: (confirmado) => confirmaciones.push(confirmado),
    alCambiarIntencion: (intencion) => { intenciones.push(intencion); return true; },
    alDenegacion: () => denegaciones.push("purgar"),
  });
  return {
    raiz, documento, peticiones, confirmaciones, anuncios, intenciones, denegaciones, desmontar,
    enviar: () => {
      const target = raiz.querySelector("[data-ct-subsanacion-form]") ?? ultimoFormulario;
      target?.focus();
      return eventos.get("submit")({ target, preventDefault() {} });
    },
    recuperar: () => {
      const target = raiz.querySelector("[data-ct-subsanacion-recuperar]") ?? ultimoBoton;
      target?.focus();
      return eventos.get("click")({ target, preventDefault() {} });
    },
    guardar: () => eventos.get("click")({ target: raiz.querySelector("[data-ct-subsanacion-guardar]") ?? ultimo.get("[data-ct-subsanacion-guardar]"), preventDefault() {} }),
    enviarPreparada: () => eventos.get("click")({ target: raiz.querySelector("[data-ct-subsanacion-enviar]") ?? ultimo.get("[data-ct-subsanacion-enviar]"), preventDefault() {} }),
    iniciar: () => { eventos.get("submit")({ target: raiz.querySelector("[data-ct-subsanacion-form]"), preventDefault() {} }); eventos.get("click")({ target: raiz.querySelector("[data-ct-subsanacion-guardar]"), preventDefault() {} }); return eventos.get("click")({ target: raiz.querySelector("[data-ct-subsanacion-enviar]"), preventDefault() {} }); },
  };
}

test("503: solo el CTA explícito recupera con el mismo DTO y la misma clave", async () => {
  const fallo = new Error("servicio no disponible"); fallo.estado = 503;
  const x = escenario([fallo, recibo]);
  await x.iniciar();
  assert.equal(x.peticiones.length, 1);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  assert.equal(x.documento.activeElement, x.raiz.querySelector("[data-ct-subsanacion-recuperar]"));
  assert.equal(x.anuncios.length, 1);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-form/u);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-recibo/u);
  await x.enviar();
  assert.equal(x.peticiones.length, 1, "el envío original sigue bloqueado");
  await x.recuperar();
  assert.equal(x.peticiones.length, 2);
  assert.equal(x.peticiones[0].ruta, RUTA_SUBSANACION_REPAROS);
  assert.equal(x.peticiones[1].ruta, RUTA_SUBSANACION_REPAROS);
  assert.equal(x.peticiones[0].estadoEsperado, 201);
  assert.deepEqual(x.peticiones[1].entrada, x.peticiones[0].entrada);
  assert.equal(x.peticiones[1].entrada.clave_idempotencia, clave);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recibo/u);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  assert.equal(x.anuncios.length, 2);
  await Promise.resolve();
  assert.deepEqual(x.confirmaciones, [recibo]);
  await x.recuperar();
  assert.equal(x.peticiones.length, 2);
  x.desmontar();
});

test("respuesta inválida y cancelación: la incertidumbre permanece sin otro POST", async () => {
  let confirmaciones = 0;
  const x = escenario([{ ...recibo, expediente_ref: "expediente:ajeno:001" }, recibo], () => ++confirmaciones !== 2);
  await x.iniciar();
  assert.equal(x.peticiones.length, 1);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  assert.equal(x.documento.activeElement, x.raiz.querySelector("[data-ct-subsanacion-recuperar]"));
  assert.equal(x.anuncios.length, 1);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-recibo/u);
  await x.recuperar();
  assert.equal(x.peticiones.length, 1);
  assert.equal(x.anuncios.length, 1);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  await x.recuperar();
  assert.equal(x.peticiones.length, 2);
  assert.deepEqual(x.peticiones[1].entrada, x.peticiones[0].entrada);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recibo/u);
  x.desmontar();
});

test("si la persona mueve el foco durante la espera, el aviso no lo recupera", async () => {
  let rechazar;
  const pendiente = new Promise((_, reject) => { rechazar = reject; });
  const x = escenario([pendiente]);
  const envio = x.iniciar();
  const enlaceExterno = { focus() { x.documento.activeElement = this; } };
  enlaceExterno.focus();
  rechazar(new Error("sin respuesta"));
  await envio;
  assert.equal(x.documento.activeElement, enlaceExterno);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  assert.equal(x.anuncios.length, 1);
  x.desmontar();
});

test("un resultado tardío tras desmontar no enfoca ni anuncia", async () => {
  let rechazar;
  const pendiente = new Promise((_, reject) => { rechazar = reject; });
  const x = escenario([pendiente]);
  const envio = x.iniciar();
  x.desmontar();
  rechazar(new Error("sin respuesta"));
  await envio;
  assert.equal(x.documento.activeElement, x.documento.body);
  assert.equal(x.anuncios.length, 0);
});

test("rechazo del reintento no borra la incertidumbre original", async () => {
  const fallo = new Error("sin respuesta fiable"); fallo.estado = 503;
  const rechazo = new Error("conflicto");
  Object.assign(rechazo, { resultadoIndeterminado: false, envelopeValido: true, estado: 409 });
  const x = escenario([fallo, rechazo, recibo]);
  await x.iniciar();
  await x.recuperar();
  assert.equal(x.peticiones.length, 2);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  assert.equal(x.documento.activeElement, x.raiz.querySelector("[data-ct-subsanacion-recuperar]"));
  assert.match(x.raiz.innerHTML, /resultado de la operación original sigue sin verificarse/u);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-form/u);
  await x.recuperar();
  assert.equal(x.peticiones.length, 3);
  assert.deepEqual(x.peticiones[2].entrada, x.peticiones[0].entrada);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-recibo/u);
  x.desmontar();
});

test("401/403 purga intención y observaciones antes de repintar", async () => {
  const fallo = new Error("sin respuesta fiable"); fallo.estado = 503;
  const denegacion = new Error("sesión sin permiso");
  Object.assign(denegacion, { resultadoIndeterminado: false, envelopeValido: true, estado: 403 });
  const x = escenario([fallo, denegacion]);
  await x.iniciar();
  await x.recuperar();
  assert.deepEqual(x.denegaciones, ["purgar"]);
  assert.match(x.raiz.innerHTML, /data-ct-subsanacion-denegada/u);
  assert.doesNotMatch(x.raiz.innerHTML, /Justificación|Corrección del reparo|expediente:subsanacion:001|data-ct-subsanacion-recuperar|data-ct-subsanacion-form/u);
  await x.recuperar();
  assert.equal(x.peticiones.length, 2);
  assert.equal(x.intenciones.at(-1)?.incierta, true, "la purga pertenece al gestor, no a alCambiarIntencion(null)");
  x.desmontar();
});
