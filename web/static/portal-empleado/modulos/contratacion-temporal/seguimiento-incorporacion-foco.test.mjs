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
    documentos: [],
  }], ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false,
});

function diferido() {
  let resolve, reject;
  const promise = new Promise((ok, error) => { resolve = ok; reject = error; });
  return { promise, resolve, reject };
}

function crearRaiz() {
  const eventos = new Map();
  const documento = { activeElement: { nombre: "body" } };
  const raiz = {
    ownerDocument: documento, isConnected: true, boton: null, estado: null, html: "", enfoques: 0,
    get innerHTML() { return this.html; },
    set innerHTML(valor) {
      if (documento.activeElement === this.boton || documento.activeElement === this.estado) {
        documento.activeElement = documento.body;
      }
      this.html = valor;
      const etiqueta = valor.match(/<button\b[^>]*data-ct-seguimiento-consultar[^>]*>/u)?.[0];
      this.boton = etiqueta ? {
        disabled: /\sdisabled(?=[\s>])/u.test(etiqueta),
        closest: (selector) => selector === "[data-ct-seguimiento-consultar]" ? this.boton : null,
        focus: () => {
          if (this.isConnected && !this.boton.disabled) {
            documento.activeElement = this.boton;
            this.enfoques++;
          }
        },
      } : null;
      this.estado = /<p\b[^>]*data-ct-seguimiento-estado-consulta[^>]*tabindex="-1"/u.test(valor) ? {
        focus: () => {
          if (this.isConnected) {
            documento.activeElement = this.estado;
            this.enfoques++;
          }
        },
      } : null;
    },
    querySelector(selector) {
      if (selector === "[data-ct-seguimiento-consultar]") return this.boton;
      if (selector === "[data-ct-seguimiento-estado-consulta]") return this.estado;
      return null;
    },
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
    pulsarConsulta() { return this.boton?.disabled ? undefined : eventos.get("click")?.({ target: this.boton }); },
  };
  documento.body = documento.activeElement;
  return { raiz, documento };
}

test("teclado: carga deshabilita el botón y pasa el foco al estado; el resultado lo devuelve", async () => {
  const { raiz, documento } = crearRaiz();
  const respuesta = diferido();
  let llamadas = 0;
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo, cliente: {
    consultar() { llamadas++; return respuesta.promise; },
  } });
  raiz.boton.focus();
  const consulta = raiz.pulsarConsulta();
  assert.equal(documento.activeElement, raiz.estado);
  assert.equal(raiz.boton.disabled, true);
  assert.match(raiz.innerHTML, /Consultando/u);
  await raiz.pulsarConsulta();
  assert.equal(llamadas, 1);
  respuesta.resolve(seguimiento());
  await consulta;
  assert.equal(documento.activeElement, raiz.boton);
  assert.equal(raiz.boton.disabled, false);
  assert.match(raiz.innerHTML, /Confirmar incorporación/u);
  destruir();
});

test("teclado: error y reintento dejan el control recuperable", async () => {
  const { raiz, documento } = crearRaiz();
  const primera = diferido(), segunda = diferido();
  let llamadas = 0;
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo, cliente: {
    consultar() { return ++llamadas === 1 ? primera.promise : segunda.promise; },
  } });
  raiz.boton.focus();
  const consulta = raiz.pulsarConsulta();
  assert.equal(documento.activeElement, raiz.estado);
  assert.equal(raiz.boton.disabled, true);
  primera.reject(new Error("fallo de red"));
  await consulta;
  assert.equal(documento.activeElement, raiz.boton);
  assert.match(raiz.innerHTML, /No se ha podido consultar el seguimiento original/u);
  const reintento = raiz.pulsarConsulta();
  assert.equal(documento.activeElement, raiz.estado);
  assert.equal(raiz.boton.disabled, true);
  segunda.resolve(seguimiento());
  await reintento;
  assert.equal(llamadas, 2);
  assert.equal(documento.activeElement, raiz.boton);
  destruir();
});

test("teclado: si el usuario mueve el foco durante la espera no se lo roba el resultado", async () => {
  const { raiz, documento } = crearRaiz();
  const respuesta = diferido();
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo, cliente: {
    consultar() { return respuesta.promise; },
  } });
  raiz.boton.focus();
  const consulta = raiz.pulsarConsulta();
  assert.equal(documento.activeElement, raiz.estado);
  const fuera = { nombre: "otro control" };
  documento.activeElement = fuera;
  respuesta.resolve(seguimiento());
  await consulta;
  assert.equal(documento.activeElement, fuera);
  destruir();
});

test("teclado: desmontar cancela y no enfoca controles retirados", async () => {
  const { raiz, documento } = crearRaiz();
  const respuesta = diferido();
  let signal;
  const destruir = montarSeguimientoIncorporacion({ raiz, recibo, cliente: {
    consultar(_expediente, opciones) { signal = opciones.signal; return respuesta.promise; },
  } });
  raiz.boton.focus();
  const consulta = raiz.pulsarConsulta();
  const enfoquesAntes = raiz.enfoques;
  raiz.isConnected = false;
  destruir();
  respuesta.resolve(seguimiento());
  await consulta;
  assert.equal(signal.aborted, true);
  assert.equal(raiz.enfoques, enfoquesAntes);
  assert.equal(raiz.innerHTML, "");
  assert.equal(documento.activeElement, documento.body);
});
