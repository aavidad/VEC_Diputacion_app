import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearGestorRecuperacionFirmasHistoricas, selectoresHistoricosDesdeEstado } from "./gestor-recuperacion-firmas-historicas.js";

const selector = Object.freeze({ expediente_ref: "expediente:historico", version_expediente: 7,
  documento: "informe_definitivo", paso_orden: 1, clave_idempotencia: "clave-firma-prueba-001",
  catalogo_huella: "a".repeat(64), via: "certificado_vec" });
const entrada = Object.freeze({ firma_ref: "firma:historica:uno", selector });
const textos = JSON.parse(await readFile(new URL("../../../textos/es/contratacion-temporal-recuperacion-firmas-v2.json", import.meta.url), "utf8")).general;

function estado(entradas = [entrada], expedienteRef = selector.expediente_ref) {
  return { vista: "expediente", carga: "listo", expediente_ref: expedienteRef,
    expediente: { demostracion: false, expediente_ref: expedienteRef, version: 9,
      ...(entradas === null ? {} : { selectores_recuperacion_firmas: entradas }) } };
}

class Nodo {
  constructor(doc, tag) { this.ownerDocument = doc; this.tagName = tag; this.children = []; this.dataset = {};
    this.parent = null; this.attributes = {}; this.listeners = {}; this._texto = ""; this.hidden = false; this.disabled = false;
    this.classList = { add: (...nombres) => { this.className = [this.className, ...nombres].filter(Boolean).join(" "); } }; }
  get textContent() { return this._texto + this.children.map((hijo) => hijo.textContent).join(""); }
  set textContent(valor) { for (const hijo of this.children) hijo.parent = null; this.children = []; this._texto = String(valor); }
  get isConnected() { return Boolean(this.parent?.isConnected); }
  append(...nodos) { for (const hijo of nodos) { hijo.parent = this; this.children.push(hijo); } }
  replaceChildren(...nodos) { for (const hijo of this.children) hijo.parent = null; this.children = []; this._texto = ""; this.append(...nodos); }
  remove() { if (!this.parent) return; this.parent.children = this.parent.children.filter((hijo) => hijo !== this); this.parent = null; }
  setAttribute(clave, valor) { this.attributes[clave] = String(valor); }
  addEventListener(tipo, accion) { this.listeners[tipo] = accion; }
  removeEventListener(tipo) { delete this.listeners[tipo]; }
  pulsar() { this.listeners.click?.(); }
  buscar(predicado) { if (predicado(this)) return this; for (const hijo of this.children) { const hallado = hijo.buscar(predicado); if (hallado) return hallado; } return null; }
}
function dom() {
  const doc = { createElement(tag) { return new Nodo(doc, tag); } };
  const raiz = new Nodo(doc, "main"); Object.defineProperty(raiz, "isConnected", { value: true });
  const zona = doc.createElement("div"); raiz.append(zona);
  raiz.querySelector = (query) => query === ".ct-exp-contenido" ? zona : null;
  return { raiz, zona };
}
function gestor(cliente, opciones = {}) {
  const { raiz, zona } = dom();
  return { ...crearGestorRecuperacionFirmasHistoricas({ raiz, cliente,
    mensajes: Object.hasOwn(opciones, "mensajes") ? opciones.mensajes : textos,
    errorRecuperacionFirmas: opciones.errorRecuperacionFirmas ?? null,
    mensajeError: () => "No se pueden mostrar las firmas anteriores. Vuelva a abrir el expediente.",
    locale: "es-ES", zonaHoraria: "Europe/Madrid" }), zona };
}
const turno = () => new Promise((resolver) => setTimeout(resolver, 0));

test("la ficha antigua sin selectores o sin autorización conserva su estado sin panel", () => {
  assert.equal(selectoresHistoricosDesdeEstado(estado(null)), null);
  assert.deepEqual(selectoresHistoricosDesdeEstado(estado([])), []);
  assert.equal(selectoresHistoricosDesdeEstado(estado([entrada], "expediente:ajeno")), null);
  assert.equal(selectoresHistoricosDesdeEstado({ ...estado(), vista: "cuadro" }), null);
  const montaje = gestor({ recuperar() { throw new Error("no debe consultar"); } });
  assert.equal(montaje.montarSiProcede(estado(null)), false);
  assert.equal(montaje.montarSiProcede(estado([])), false);
  assert.equal(montaje.zona.children.length, 0);
});

test("rechaza selectores incompletos, ajenos o duplicados sin pedir firmas", () => {
  for (const entradas of [
    [{ ...entrada, selector: { ...selector, via: "otra" } }],
    [{ ...entrada, selector: { ...selector, expediente_ref: "expediente:ajeno" } }],
    [{ ...entrada, selector: { ...selector, version_expediente: 10 } }],
    [entrada, entrada],
    [{ ...entrada, extra: true }],
  ]) assert.equal(selectoresHistoricosDesdeEstado(estado(entradas)), null);
});

test("consulta una sola vez con el selector histórico exacto y muestra fecha, estado y recibo", async () => {
  const llamadas = [];
  const montaje = gestor({ recuperar: async (solicitud, { signal }) => {
    llamadas.push({ solicitud, signal });
    return { firmas: [{ firma_ref: entrada.firma_ref, recibo_ref: "recibo:firma:uno",
      registrada_en: "2026-10-14T10:00:00Z", resultado: "firmado", paso_orden: 1 }] };
  } });
  assert.equal(montaje.montarSiProcede(estado()), true);
  const boton = montaje.zona.buscar((nodo) => Object.hasOwn(nodo.dataset, "ctRecuperacionHistoricaConsultar"));
  assert.ok(boton);
  boton.pulsar(); boton.pulsar();
  await turno();
  assert.equal(llamadas.length, 1);
  assert.deepEqual(llamadas[0].solicitud, selector);
  assert.match(montaje.zona.textContent, /Informe definitivo/u);
  assert.match(montaje.zona.textContent, /Recibo: recibo:firma:uno/u);
  assert.doesNotMatch(montaje.zona.textContent, /canon|sha256|firma:historica:uno/u);
  montaje.retirar();
});

test("cambiar ficha aborta la petición pendiente y limpia el resultado", async () => {
  let terminar;
  let signal;
  const montaje = gestor({ recuperar: (_solicitud, opciones) => {
    signal = opciones.signal;
    return new Promise((resolver) => { terminar = resolver; });
  } });
  montaje.montarSiProcede(estado());
  montaje.zona.buscar((nodo) => Object.hasOwn(nodo.dataset, "ctRecuperacionHistoricaConsultar")).pulsar();
  await turno();
  assert.equal(signal.aborted, false);
  montaje.montarSiProcede(estado(null, "expediente:otro"));
  assert.equal(signal.aborted, true);
  terminar({ firmas: [{ firma_ref: entrada.firma_ref, recibo_ref: "recibo:antiguo",
    registrada_en: "2026-10-14T10:00:00Z", resultado: "firmado" }] });
  await turno();
  assert.equal(montaje.zona.children.length, 0);
});

test("una respuesta sin la firma seleccionada no se presenta como recuperación válida", async () => {
  const montaje = gestor({ recuperar: async () => ({ firmas: [{ firma_ref: "firma:ajena:dos",
    recibo_ref: "recibo:ajeno", registrada_en: "2026-10-14T10:00:00Z",
    resultado: "firmado", paso_orden: 1 }] }) });
  montaje.montarSiProcede(estado());
  montaje.zona.buscar((nodo) => Object.hasOwn(nodo.dataset, "ctRecuperacionHistoricaConsultar")).pulsar();
  await turno();
  assert.match(montaje.zona.textContent, /No se muestra ningún dato/u);
  assert.doesNotMatch(montaje.zona.textContent, /recibo:ajeno/u);
  montaje.retirar();
});

test("la denegación se explica y no conserva un recibo anterior", async () => {
  const montaje = gestor({ recuperar: async () => { throw Object.assign(new Error("denegado"), { estado: 403 }); } });
  montaje.montarSiProcede(estado());
  montaje.zona.buscar((nodo) => Object.hasOwn(nodo.dataset, "ctRecuperacionHistoricaConsultar")).pulsar();
  await turno();
  assert.match(montaje.zona.textContent, /No puede consultar estas firmas/u);
  assert.doesNotMatch(montaje.zona.textContent, /Recibo:/u);
  montaje.retirar();
});

test("un fallo del catálogo deja abierta la ficha y explica el problema si hay firmas", () => {
  let llamadas = 0;
  const montaje = gestor({ recuperar() { llamadas += 1; } }, {
    mensajes: null, errorRecuperacionFirmas: new Error("catalogo_no_disponible"),
  });
  assert.equal(montaje.montarSiProcede(estado()), false);
  assert.match(montaje.zona.textContent, /No se pueden mostrar las firmas anteriores/u);
  assert.equal(montaje.zona.buscar((nodo) => Object.hasOwn(nodo.dataset, "ctRecuperacionHistoricaConsultar")), null);
  assert.equal(llamadas, 0);
  montaje.retirar();
});

test("un selector inválido deja un error visible sin enviar peticiones", () => {
  let llamadas = 0;
  const montaje = gestor({ recuperar() { llamadas += 1; } });
  const invalido = [{ ...entrada, selector: { ...selector, via: "otra" } }];
  assert.equal(montaje.montarSiProcede(estado(invalido)), false);
  assert.match(montaje.zona.textContent, /No se pueden mostrar las firmas anteriores/u);
  assert.equal(llamadas, 0);
  montaje.retirar();
});

test("si falta el cliente, la ficha explica que la consulta no está disponible", () => {
  const montaje = gestor(null);
  assert.equal(montaje.montarSiProcede(estado()), false);
  assert.match(montaje.zona.textContent, /No se pueden mostrar las firmas anteriores/u);
  montaje.retirar();
});


test("las firmas del mismo paso en cortes distintos se distinguen por su versión", () => {
  const { montarSiProcede, zona } = gestor({ recuperar() { throw new Error("no debe consultar al pintar"); } });
  assert.equal(montarSiProcede(estado([entrada, { firma_ref: "firma:historica:dos",
    selector: { ...selector, version_expediente: 8 } } ])), true);
  assert.ok(zona.textContent.includes("Versión del expediente 7"));
  assert.ok(zona.textContent.includes("Versión del expediente 8"));
  const etiquetas = [];
  const recorrer = (nodo) => { if (nodo.dataset.ctRecuperacionHistoricaConsultar !== undefined) etiquetas.push(nodo.attributes["aria-label"]); nodo.children.forEach(recorrer); };
  recorrer(zona);
  assert.equal(new Set(etiquetas).size, 2);
});
