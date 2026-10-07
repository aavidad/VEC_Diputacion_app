import assert from "node:assert/strict";
import test from "node:test";
import { ErrorClienteResolucionCronos } from "./cliente-resolucion-http.js";
import { montarBandejaPermisosCronos } from "./vista-bandeja-permisos.js";

const solicitudRef = "permiso:cronos:solicitud:persona-sintetica-1";
const datos = (paso = "responsable") => ({ paso, pendientes: [{ solicitud_ref: solicitudRef,
  empleado_ref: "emp_AAAAAAAAAAAAAAAAAAAAAA", empleado_etiqueta: "Persona sintética",
  permiso_ref: "permiso:cronos:vacaciones", nombre: "Vacaciones", circuito: "J-A",
  pendiente_asignacion: false, justificante_exigido: false, desde: "2026-10-05", hasta: "2026-10-07",
  cantidad: 3, unidad: "dia", estado: "solicitado", version: 1, solicitada_en: "2026-09-24T08:00:00Z" }] });
function diferida() {
  let resolver; let rechazar;
  const promesa = new Promise((si, no) => { resolver = si; rechazar = no; });
  return { promesa, resolver, rechazar };
}
const esperar = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };

// El repintado retira los controles anteriores, como innerHTML en Chrome.
// La comprobación de navegador complementa este DOM mínimo de foco.
function superficie() {
  const documento = { body: {}, activeElement: null, hasFocus: () => true };
  documento.activeElement = documento.body;
  let html = ""; let controles = [];
  const coincide = (control, selector) => {
    const partes = selector.split(" ").at(-1);
    return [...partes.matchAll(/\[([a-z-]+)(?:="([^"]*)"|=([a-z_-]+))?\]/gu)].every(([, atributo, citado, simple]) => {
      const valor = citado ?? simple;
      return control.hasAttribute(atributo) && (valor === undefined || control.getAttribute(atributo) === valor);
    });
  };
  const nodo = {
    dataset: {}, eventos: {},
    get innerHTML() { return html; },
    set innerHTML(valor) {
      if (controles.includes(documento.activeElement)) documento.activeElement = documento.body;
      html = valor;
      controles = [...html.matchAll(/<(button|input|textarea|p|span|form)\b([^>]*)>/gu)].map(([, tag, texto]) => {
        const atributos = new Map([...texto.matchAll(/([a-z-]+)(?:="([^"]*)")?/gu)].map(([, clave, valor]) => [clave, valor ?? ""]));
        const dataset = Object.fromEntries([...atributos].filter(([clave]) => clave.startsWith("data-")).map(([clave, valor]) =>
          [clave.slice(5).replace(/-([a-z])/gu, (_, letra) => letra.toUpperCase()), valor]));
        const control = { tag, dataset, value: atributos.get("value") ?? "", disabled: atributos.has("disabled"),
          getAttribute: (clave) => atributos.get(clave) ?? null,
          hasAttribute: (clave) => atributos.has(clave),
          focus() { if (!this.disabled) documento.activeElement = this; },
          closest: (selector) => coincide(control, selector) ? control : null,
        };
        if (tag === "textarea") control.value = html.match(/<textarea\b[^>]*>([^<]*)<\/textarea>/u)?.[1] ?? "";
        return control;
      });
    },
    contains: (control) => controles.includes(control),
    querySelector: (selector) => controles.find((control) => coincide(control, selector)) ?? null,
    addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() {},
  };
  documento.createElement = () => nodo;
  const raiz = { ownerDocument: documento, append() {} };
  const elegir = (selector) => { const control = nodo.querySelector(selector); assert.ok(control, selector); control.focus(); return control; };
  const pulsar = (selector) => { const control = elegir(selector); nodo.eventos.click({ target: control }); };
  const enviar = (decision = "aprobar", motivo = "") => nodo.eventos.submit({ preventDefault() {}, target: {
    matches: (selector) => selector === "[data-cronos-resolucion-formulario]",
    elements: { namedItem: (nombre) => ({ value: nombre === "decision" ? decision : motivo }) },
  } });
  return { raiz, nodo, documento, elegir, pulsar, enviar };
}

test("el cambio de paso conserva el botón en carga, éxito y error de lectura", async () => {
  const s = superficie(); let lectura = diferida();
  const vista = montarBandejaPermisosCronos({ raiz: s.raiz, cliente: {
    consultarBandeja: ({ paso }) => paso === "responsable" ? Promise.resolve(datos(paso)) : lectura.promesa,
    resolver: async () => ({}),
  } });
  await esperar();
  s.pulsar('[data-cronos-paso="administracion"]');
  assert.equal(s.documento.activeElement, s.nodo.querySelector('[data-cronos-paso="administracion"]'));
  lectura.resolver(datos("administracion")); await esperar();
  assert.equal(s.documento.activeElement, s.nodo.querySelector('[data-cronos-paso="administracion"]'));
  lectura = diferida(); const recarga = vista.recargar();
  lectura.rechazar(new ErrorClienteResolucionCronos("servicio_no_disponible", 503)); await recarga;
  assert.equal(s.documento.activeElement, s.nodo.querySelector('[data-cronos-paso="administracion"]'));
});

test("el envío usa un destino provisional y devuelve el foco al reintento tras el error", async () => {
  const s = superficie(); const respuesta = diferida();
  montarBandejaPermisosCronos({ raiz: s.raiz, cliente: { consultarBandeja: async () => datos(), resolver: () => respuesta.promesa } });
  await esperar(); s.pulsar(`[data-cronos-resolver="${solicitudRef}"]`);
  s.elegir("[data-cronos-resolucion-enviar]"); const envio = s.enviar("denegar", "Motivo conservado");
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-resolucion-formulario]"));
  respuesta.rechazar(new ErrorClienteResolucionCronos("servicio_no_disponible", 503)); await envio;
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-resolucion-enviar]"));
  assert.equal(s.nodo.querySelector('[name="motivo"]').value, "Motivo conservado");
});

test("la recuperación del formulario conserva el motivo editado y el foco", async () => {
  const s = superficie(); let lectura = diferida(); let consultas = 0;
  const vista = montarBandejaPermisosCronos({ raiz: s.raiz, cliente: {
    consultarBandeja: () => ++consultas === 1 ? Promise.resolve(datos()) : lectura.promesa, resolver: async () => ({}),
  } });
  await esperar(); s.pulsar(`[data-cronos-resolver="${solicitudRef}"]`);
  s.elegir('[name="motivo"]').value = "Borrador editado sin enviar";
  const recarga = vista.recargar();
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-bandeja-estado]"));
  lectura.resolver(datos()); await recarga;
  assert.equal(s.documento.activeElement, s.nodo.querySelector('[name="motivo"]'));
  assert.equal(s.documento.activeElement.value, "Borrador editado sin enviar");
});

test("las respuestas tardías respetan el foco externo durante lectura y envío", async () => {
  for (const operacion of ["lectura", "envio"]) {
    const s = superficie(); const respuesta = diferida(); let consultas = 0;
    const vista = montarBandejaPermisosCronos({ raiz: s.raiz, cliente: {
      consultarBandeja: () => ++consultas === 1 ? Promise.resolve(datos()) : respuesta.promesa,
      resolver: () => respuesta.promesa,
    } });
    await esperar(); let pendiente;
    if (operacion === "lectura") { s.elegir('[data-cronos-paso="responsable"]'); pendiente = vista.recargar(); }
    else { s.pulsar(`[data-cronos-resolver="${solicitudRef}"]`); s.elegir("[data-cronos-resolucion-enviar]"); pendiente = s.enviar(); }
    const externo = {}; s.documento.activeElement = externo;
    if (operacion === "lectura") respuesta.resolver(datos());
    else respuesta.rechazar(new ErrorClienteResolucionCronos("servicio_no_disponible", 503));
    await pendiente; assert.equal(s.documento.activeElement, externo);
  }
});

test("la resolución confirmada lleva al resultado sin añadir tabulaciones positivas", async () => {
  const s = superficie(); let consultas = 0;
  montarBandejaPermisosCronos({ raiz: s.raiz, cliente: {
    consultarBandeja: async () => ++consultas === 1 ? datos() : { paso: "responsable", pendientes: [] },
    resolver: async () => ({ estado: "pendiente_administracion", replay: false }),
  } });
  await esperar(); s.pulsar(`[data-cronos-resolver="${solicitudRef}"]`);
  s.elegir("[data-cronos-resolucion-enviar]"); await s.enviar();
  assert.equal(s.documento.activeElement, s.nodo.querySelector("[data-cronos-bandeja-resultado]"));
  assert.doesNotMatch(s.nodo.innerHTML, /tabindex="[1-9]/u);
});

test("la lectura no recupera foco cuando el documento ha perdido la activación", async () => {
  const s = superficie(); const respuesta = diferida(); let consultas = 0;
  const vista = montarBandejaPermisosCronos({ raiz: s.raiz, cliente: {
    consultarBandeja: () => ++consultas === 1 ? Promise.resolve(datos()) : respuesta.promesa,
    resolver: async () => ({}),
  } });
  await esperar(); s.elegir('[data-cronos-paso="responsable"]'); const lectura = vista.recargar();
  s.documento.hasFocus = () => false;
  respuesta.resolver(datos()); await lectura;
  assert.equal(s.documento.activeElement, s.documento.body);
});
