import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaRecorridosDietas } from "./vista-recorridos.js?v=20261001-ct-a-i18n-v1";

const claveDato = (atributo) => atributo.slice(5).replace(/-([a-z])/gu, (_todo, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {};
    this.attrs = {}; this.listeners = {}; this.parent = null; this.textContent = ""; this.disabled = false;
  }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  remove() { this.parent?.removeChild(this); }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); }
  removeAttribute(nombre) { delete this.attrs[nombre]; }
  addEventListener(tipo, manejador) { this.listeners[tipo] = manejador; }
  removeEventListener(tipo) { delete this.listeners[tipo]; }
  focus() { this.ownerDocument.activeElement = this; }
  matches(selector) {
    if (!selector.startsWith("[")) return this.tagName === selector;
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    if (!coincidencia) return false;
    const atributo = coincidencia[1];
    const valor = atributo.startsWith("data-") ? this.dataset[claveDato(atributo)] : this.attrs[atributo];
    return valor !== undefined && (coincidencia[2] === undefined || valor === coincidencia[2]);
  }
  closest(selector) { for (let actual = this; actual; actual = actual.parent) if (actual.matches(selector)) return actual; return null; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) { const salida = []; const visitar = (actual) => { if (actual.matches(selector)) salida.push(actual); actual.children.forEach(visitar); }; visitar(this); return salida; }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
function texto(nodo) { return [nodo.textContent, ...nodo.children.map(texto)].join(" "); }
const item = Object.freeze({
  comision: { referencia: "dco_1234567890123456789012", estado: "borrador", fecha_inicio: "2026-09-24",
    fecha_fin: "2026-09-25", motivo: "Visita al centro", codigos_ruta: [], relacion_ref: "rel_1234567890123456789012" },
  recibo: { referencia: "rcd_1234567890123456789012", version: 1, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: false },
});
const clienteBorradores = () => ({ listar: async () => ({ items: [item] }), obtener: async () => item, crear: async () => item });

test("Dietas monta el recorrido interno y abre el formulario real sin itinerario de presentación", async () => {
  const contenedor = raiz();
  const vista = montarVistaRecorridosDietas(contenedor, { clienteBorradores: clienteBorradores() });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  const formulario = contenedor.querySelector("[data-dietas-borrador-form]");
  assert.ok(panel); assert.ok(formulario);
  assert.equal(formulario.hidden, true);
  assert.equal(contenedor.querySelector("[data-dietas-area-itinerario]"), null);
  assert.equal(contenedor.querySelector("[data-dietas-mapa-pendiente]"), null);
  // Sin clientes de circuito solo existe el recorrido propio: no hay pasos que no se puedan abrir.
  assert.equal(panel.querySelectorAll("[data-dietas-cambiar-etapa]").length, 0);
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-abrir-nueva-comision]") });
  assert.equal(formulario.hidden, false);
  assert.equal(panel.querySelector("[data-dietas-abrir-nueva-comision]").attrs["aria-expanded"], "true");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-abrir-nueva-comision]") });
  assert.equal(formulario.hidden, true);
  vista.desmontar();
  assert.equal(contenedor.querySelector("[data-dietas-recorridos]"), null);
});

test("la bandeja propia consulta GET y conserva el recibo real en la ficha", async () => {
  const contenedor = raiz(); let consultas = 0;
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: { ...clienteBorradores(), obtener: async () => { consultas += 1; return item; } },
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  const boton = contenedor.querySelector("[data-dietas-borrador-detalle]");
  await contenedor.querySelector("[data-dietas-borradores-propios]").listeners.click({ target: boton });
  assert.equal(consultas, 1);
  const recibo = contenedor.querySelector("[data-dietas-borrador-recibo]");
  assert.ok(recibo);
  assert.match(texto(recibo), /rcd_1234567890123456789012/u);
  assert.equal(panel.querySelector("[data-dietas-panel-etapa=\"circuito\"]").hidden, true);
  vista.desmontar();
});

test("al cambiar de bandeja o salir aborta lecturas pendientes y no conserva una etapa ajena", async () => {
  const contenedor = raiz(); const consultas = [];
  const clienteCircuito = {
    listar: (_consulta, { signal }) => new Promise((_resolver, rechazar) => {
      consultas.push(signal);
      signal.addEventListener("abort", () => rechazar(Object.assign(new Error("cancelada"), { codigo: "operacion_abortada" })), { once: true });
    }),
    decidir: async () => { throw new Error("no debe decidir"); },
    competencias: async () => ({ fuente: "acreditada", etapas: ["autorizacion", "revision"] }),
  };
  const vista = montarVistaRecorridosDietas(contenedor, { clienteBorradores: clienteBorradores(), clienteCircuito });
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  await Promise.resolve(); await Promise.resolve();
  assert.deepEqual(panel.querySelectorAll("[data-dietas-cambiar-etapa]").map((boton) => boton.dataset.dietasCambiarEtapa),
    ["solicitante", "revision", "autorizacion", "control"]);
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-cambiar-etapa="revision"]') });
  assert.equal(consultas.length, 1);
  assert.ok(panel.querySelector("[data-dietas-bandeja-circuito]"));
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-cambiar-etapa="autorizacion"]') });
  assert.equal(consultas[0].aborted, true);
  assert.equal(consultas.length, 2);
  vista.desmontar();
  assert.equal(consultas[1].aborted, true);
  assert.equal(contenedor.querySelector("[data-dietas-recorridos]"), null);
});

test("la lista muestra el estado real y el motivo de Personal cierra el alta", async () => {
  const contenedor = raiz();
  const enviado = { ...item, comision: { ...item.comision, estado: "enviado_pendiente_revision" } };
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: { ...clienteBorradores(), listar: async () => ({ items: [enviado] }) },
    estadoRelaciones: "no_disponible", motivoRelaciones: "empleado_no_disponible",
  });
  await Promise.resolve(); await Promise.resolve();
  const chip = contenedor.querySelector("[data-dietas-estado-comision]");
  assert.equal(chip.dataset.dietasEstadoComision, "enviado_pendiente_revision");
  assert.equal(chip.textContent, "Enviada a revisión");
  const abrir = contenedor.querySelector("[data-dietas-abrir-nueva-comision]");
  assert.equal(abrir.disabled, true);
  assert.match(abrir.title, /No consta un empleado vigente/u);
  assert.match(texto(contenedor), /No consta un empleado vigente asociado a su usuario/u);
  assert.throws(() => montarVistaRecorridosDietas(raiz(), {
    clienteBorradores: clienteBorradores(), estadoRelaciones: "disponible", motivoRelaciones: "empleado_no_disponible",
  }), TypeError);
  vista.desmontar();
});

test("sin fuente de competencia no ofrece pestañas del circuito ni consulta bandejas", async () => {
  const contenedor = raiz(); let listados = 0;
  const clienteCircuito = {
    listar: async () => { listados += 1; return { items: [], competencia: "sin_fuente" }; },
    decidir: async () => { throw new Error("no debe decidir"); },
    competencias: async () => ({ fuente: "sin_fuente", etapas: [] }),
  };
  const vista = montarVistaRecorridosDietas(contenedor, { clienteBorradores: clienteBorradores(), clienteCircuito });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  assert.deepEqual(panel.querySelectorAll("[data-dietas-cambiar-etapa]").map((boton) => boton.dataset.dietasCambiarEtapa), []);
  assert.equal(panel.querySelector("[data-dietas-circuito-sin-fuente]"), null);
  assert.equal(panel.querySelector("[data-dietas-circuito-decision]"), null);
  assert.equal(listados, 0);
  vista.desmontar();
});

test("un fallo de competencias permite reintentar una sola lectura y acredita las pestañas una vez", async () => {
  const contenedor = raiz(); let competencias = 0; let listados = 0; let resolver;
  const clienteCircuito = {
    competencias: () => {
      competencias += 1;
      if (competencias === 1) return Promise.reject(Object.assign(new Error("red"), { codigo: "red_no_disponible" }));
      return new Promise((completar) => { resolver = completar; });
    },
    listar: async () => { listados += 1; return { items: [], competencia: "acreditada" }; },
  };
  const vista = montarVistaRecorridosDietas(contenedor, { clienteBorradores: clienteBorradores(), clienteCircuito });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-recorridos]");
  const estado = panel.querySelector("[data-dietas-estado-competencias]");
  const boton = panel.querySelector("[data-dietas-reintentar-competencias]");
  assert.equal(estado.hidden, false);
  assert.match(texto(estado), /No se han podido comprobar sus bandejas/u);
  assert.equal(panel.querySelector("[data-dietas-area-borradores]") !== null, true);
  assert.equal(listados, 0);
  boton.focus();
  panel.listeners.click({ target: boton }); panel.listeners.click({ target: boton });
  assert.equal(competencias, 2);
  assert.equal(boton.disabled, true);
  resolver({ fuente: "acreditada", etapas: ["revision", "autorizacion"] });
  await Promise.resolve(); await Promise.resolve();
  assert.equal(estado.hidden, true);
  assert.deepEqual(panel.querySelectorAll("[data-dietas-cambiar-etapa]").map((paso) => paso.dataset.dietasCambiarEtapa),
    ["solicitante", "revision", "autorizacion", "control"]);
  assert.equal(contenedor.ownerDocument.activeElement.dataset.dietasCambiarEtapa, "revision");
  panel.listeners.click({ target: boton });
  assert.equal(competencias, 2);
  assert.equal(listados, 0);
  vista.desmontar();
});

test("la denegación de competencias no ofrece reintento ni consulta bandejas", async () => {
  const contenedor = raiz(); let listados = 0;
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: clienteBorradores(),
    clienteCircuito: {
      competencias: async () => { throw Object.assign(new Error("denegado"), { codigo: "acceso_denegado", estado: 403 }); },
      listar: async () => { listados += 1; return { items: [] }; },
    },
  });
  await Promise.resolve(); await Promise.resolve();
  const estado = contenedor.querySelector("[data-dietas-estado-competencias]");
  assert.match(texto(estado), /No tiene permiso para consultar las bandejas/u);
  assert.equal(estado.querySelector("[data-dietas-reintentar-competencias]").hidden, true);
  assert.equal(contenedor.querySelectorAll("[data-dietas-cambiar-etapa]").length, 0);
  assert.equal(listados, 0);
  vista.desmontar();
});

test("una respuesta tardía tras desmontar no monta pasos y las etapas vacías son válidas", async () => {
  const contenedor = raiz(); let resolver; let signal;
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: clienteBorradores(),
    clienteCircuito: { competencias: ({ signal: recibido }) => { signal = recibido; return new Promise((completar) => { resolver = completar; }); } },
  });
  vista.desmontar();
  assert.equal(signal.aborted, true);
  resolver({ fuente: "acreditada", etapas: ["revision"] });
  await Promise.resolve();
  assert.equal(contenedor.querySelector("[data-dietas-recorridos]"), null);

  const otra = raiz();
  const vistaVacia = montarVistaRecorridosDietas(otra, {
    clienteBorradores: clienteBorradores(),
    clienteCircuito: { competencias: async () => ({ fuente: "acreditada", etapas: [] }) },
  });
  await Promise.resolve(); await Promise.resolve();
  assert.equal(otra.querySelector("[data-dietas-estado-competencias]").hidden, true);
  assert.equal(otra.querySelectorAll("[data-dietas-cambiar-etapa]").length, 0);
  vistaVacia.desmontar();
});
