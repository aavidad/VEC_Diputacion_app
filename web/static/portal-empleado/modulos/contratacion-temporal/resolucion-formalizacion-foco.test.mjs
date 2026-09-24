import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioResolucionFormalizacion } from "./formulario-resolucion-formalizacion.js";

const preparacion = {
  esquema: "vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1",
  expediente_ref: "expediente:ct:foco", propuesta_ref: "propuesta:ct:foco",
  version_esperada: 7, version_actual: 7, recibo: null,
};
const recibo = {
  esquema: "vec.contratacion-temporal.resolucion-formalizacion.v1", estado: "registrada",
  expediente_ref: preparacion.expediente_ref, version_resultante: 8,
  propuesta_ref: preparacion.propuesta_ref, resolucion_formalizacion_ref: "resolucion:ct:foco",
  documento_resolucion_ref: "documento:ct:foco", documento_resolucion_version: 1,
  documento_resolucion_sha256: "a".repeat(64), actuacion_ref: "actuacion:ct:foco",
  auditoria_ref: "auditoria:ct:foco", outbox_ref: "outbox:ct:foco",
  recibo_ref: "recibo:ct:foco", registrada_en: "2026-09-06T12:00:00Z",
  tipo_validacion: "manual_de_ejercicio", firma_oficial: false, eficacia_administrativa: false,
};

function raizDOM() {
  const eventos = new Map();
  const documento = { body: {} };
  documento.activeElement = documento.body;
  const raiz = {
    ownerDocument: documento,
    _html: "", _formulario: null, _estado: null,
    get innerHTML() { return this._html; },
    set innerHTML(html) {
      if (this._formulario && documento.activeElement?.raiz === this) documento.activeElement = documento.body;
      this._html = html;
      const nodo = () => ({ raiz: this, focus() { documento.activeElement = this; },
        setAttribute(nombre, valor) { this[nombre] = valor; } });
      this._estado = html.includes("data-ct-rf-estado") ? nodo() : null;
      this._formulario = html.includes("data-ct-resolucion-formalizacion-form") ? {
        ...nodo(), elements: Object.fromEntries([
          "numero_resolucion", "fecha_resolucion", "motivo", "confirma_revision_propuesta",
          "confirma_ejercicio_manual", "clave_idempotencia",
        ].map((nombre) => [nombre, { ...nodo(), value: "", checked: false }])),
        querySelector(selector) {
          if (selector === 'button[type="submit"]') return this.boton;
          if (selector === "fieldset") return this.grupo;
          return null;
        },
        boton: nodo(), grupo: nodo(),
      } : null;
    },
    addEventListener(tipo, accion) { eventos.set(tipo, accion); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
    contains(nodo) { return nodo?.raiz === this; },
    querySelector(selector) {
      if (selector === "[data-ct-rf-estado]") return this._estado;
      if (selector === "[data-ct-resolucion-formalizacion-form]") return this._formulario;
      return null;
    },
    enviar() { return eventos.get("submit")({ preventDefault() {}, target: this._formulario }); },
  };
  return { raiz, documento };
}

function rellenar(formulario, cambios = {}) {
  const datos = { numero_resolucion: "R-24/2026", fecha_resolucion: "2026-09-06",
    motivo: "Revisión manual", confirma_revision_propuesta: true,
    confirma_ejercicio_manual: true, ...cambios };
  for (const [nombre, valor] of Object.entries(datos)) {
    formulario.elements[nombre][typeof valor === "boolean" ? "checked" : "value"] = valor;
  }
}

function montar(raiz, cliente) {
  return montarFormularioResolucionFormalizacion({ raiz, cliente, preparacion,
    generarClaveIdempotencia: () => "123e4567-e89b-42d3-a456-426614174000",
    confirmarOperacion: () => true });
}

test("validación enfoca el primer campo inválido sin reemplazar controles ni enviar", async () => {
  const { raiz, documento } = raizDOM(); let posts = 0;
  montar(raiz, { registrarResolucionFormalizacion() { posts += 1; return recibo; } });
  const formulario = raiz._formulario;
  rellenar(formulario, { numero_resolucion: "   ", confirma_ejercicio_manual: false });
  formulario.boton.focus();
  await raiz.enviar();
  assert.equal(posts, 0);
  assert.equal(raiz._formulario, formulario);
  assert.equal(formulario.elements.motivo.value, "Revisión manual");
  assert.equal(documento.activeElement, formulario.elements.numero_resolucion);
  assert.match(raiz._estado.textContent, /Revise número/u);
  rellenar(formulario, { confirma_ejercicio_manual: false });
  await raiz.enviar();
  assert.equal(documento.activeElement, formulario.elements.confirma_ejercicio_manual);
  assert.equal(posts, 0);
});

test("respuesta tardía no recupera el foco externo; el recibo válido sigue visible", async () => {
  const { raiz, documento } = raizDOM();
  let terminar, posts = 0;
  montar(raiz, { registrarResolucionFormalizacion() {
    posts += 1;
    return new Promise((resolver) => { terminar = resolver; });
  } });
  rellenar(raiz._formulario);
  raiz._formulario.boton.focus();
  const vuelo = raiz.enviar();
  assert.equal(raiz._formulario["aria-busy"], "true");
  assert.equal(raiz._formulario.elements.motivo.readOnly, true);
  assert.equal(raiz._formulario.elements.confirma_ejercicio_manual.disabled, true);
  assert.equal(raiz._formulario.boton.disabled, true);
  assert.equal(documento.activeElement, raiz._estado);
  const externo = { focus() { documento.activeElement = this; } };
  externo.focus();
  terminar(recibo);
  await vuelo;
  assert.equal(documento.activeElement, externo);
  assert.equal(posts, 1);
  assert.match(raiz.innerHTML, /recibo:ct:foco/u);
  assert.match(raiz.innerHTML, /Firma oficial/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-resolucion-formalizacion-form/u);
});

test("rechazo determinado anuncia el estado si el foco sigue dentro; desmontar impide recuperarlo", async () => {
  const { raiz, documento } = raizDOM();
  let rechazar; const enviados = [];
  const desmontar = montar(raiz, { registrarResolucionFormalizacion(solicitud) {
    enviados.push(solicitud);
    return new Promise((_resolver, rechazo) => { rechazar = rechazo; });
  } });
  rellenar(raiz._formulario);
  const formulario = raiz._formulario;
  raiz._formulario.boton.focus();
  const vuelo = raiz.enviar();
  // El navegador no permite modificar un textarea readOnly mientras la petición está en vuelo.
  if (!formulario.elements.motivo.readOnly) formulario.elements.motivo.value = "Corrección prematura";
  assert.equal(formulario.elements.motivo.value, "Revisión manual");
  assert.equal(formulario.elements.clave_idempotencia.value, enviados[0].clave_idempotencia);
  assert.equal(formulario.elements.confirma_revision_propuesta.disabled, true);
  assert.equal(formulario.boton.disabled, true);
  rechazar(Object.assign(new Error("rechazada"), { envelopeValido: true, resultadoIndeterminado: false }));
  await vuelo;
  assert.equal(documento.activeElement, raiz._estado);
  assert.equal(raiz._formulario, formulario);
  assert.equal(formulario.elements.numero_resolucion.value, "R-24/2026");
  assert.equal(formulario.elements.motivo.value, "Revisión manual");
  assert.equal(formulario.elements.clave_idempotencia.value, enviados[0].clave_idempotencia);
  assert.equal(formulario.elements.motivo.readOnly, false);
  assert.equal(formulario.elements.confirma_revision_propuesta.disabled, false);
  assert.equal(formulario.boton.disabled, false);

  formulario.elements.motivo.value = "Corrección tras 422";
  raiz._formulario.boton.focus();
  const otroVuelo = raiz.enviar();
  assert.equal(enviados[1].motivo, "Corrección tras 422");
  assert.equal(enviados[1].clave_idempotencia, enviados[0].clave_idempotencia);
  desmontar();
  const focoTrasDesmontar = documento.activeElement;
  rechazar(Object.assign(new Error("tardía"), { envelopeValido: true, resultadoIndeterminado: false }));
  await otroVuelo;
  assert.equal(documento.activeElement, focoTrasDesmontar);
  assert.equal(raiz.innerHTML, "");
});

test("409 congela campos nativos y el reintento conserva el cuerpo y la clave visible", async () => {
  const { raiz, documento } = raizDOM(); const enviados = [];
  montar(raiz, {
    registrarResolucionFormalizacion(solicitud) {
      enviados.push(solicitud);
      if (enviados.length === 1) return Promise.reject(Object.assign(new Error("409"), { estado: 409 }));
      return Promise.resolve(recibo);
    },
    prepararResolucionFormalizacion: async () => preparacion,
  });
  const formulario = raiz._formulario;
  rellenar(formulario);
  formulario.boton.focus();
  await raiz.enviar();
  assert.equal(raiz._formulario, formulario);
  assert.equal(documento.activeElement, raiz._estado);
  assert.equal(formulario.grupo.disabled, true);
  assert.equal(formulario.boton.disabled, false);
  assert.match(formulario.boton.textContent, /Reintentar/u);
  assert.equal(formulario.elements.motivo.value, "Revisión manual");
  assert.equal(formulario.elements.clave_idempotencia.value, enviados[0].clave_idempotencia);
  await raiz.enviar();
  assert.deepEqual(enviados[1], enviados[0]);
  assert.match(raiz.innerHTML, /recibo:ct:foco/u);
});

test("503 indeterminado mantiene la clave visible y el reintento exacto sin reemplazar controles", async () => {
  const { raiz, documento } = raizDOM(); const enviados = [];
  montar(raiz, { registrarResolucionFormalizacion(solicitud) {
    enviados.push(solicitud);
    if (enviados.length === 1) return Promise.reject(Object.assign(new Error("503"), {
      estado: 503, envelopeValido: true, resultadoIndeterminado: true,
    }));
    return Promise.resolve(recibo);
  } });
  const formulario = raiz._formulario;
  rellenar(formulario);
  formulario.boton.focus();
  await raiz.enviar();
  assert.equal(raiz._formulario, formulario);
  assert.equal(documento.activeElement, raiz._estado);
  assert.equal(formulario.grupo.disabled, true);
  assert.equal(formulario.boton.disabled, false);
  assert.equal(formulario.elements.clave_idempotencia.value, enviados[0].clave_idempotencia);
  await raiz.enviar();
  assert.deepEqual(enviados[1], enviados[0]);
  assert.match(raiz.innerHTML, /recibo:ct:foco/u);
});
