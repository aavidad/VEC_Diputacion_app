import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";
import { montarVistaRecorridosDietas } from "./vista-recorridos.js";
import { MENSAJES_REVISION_DIETAS_ES } from "./i18n-revision.js";

const claveDatos = (atributo) =>
  atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    this.ownerDocument = documento;
    this.tagName = etiqueta;
    this.children = [];
    this.dataset = {};
    this.listeners = {};
    this.attrs = {};
    this.parent = null;
    this.disabled = false;
    this.textContent = "";
  }
  append(...nodos) {
    this.children.push(...nodos);
    nodos.forEach((hijo) => {
      hijo.parent = this;
    });
  }
  replaceChildren(...nodos) {
    this.children = [];
    this.append(...nodos);
  }
  removeChild(nodo) {
    this.children = this.children.filter((hijo) => hijo !== nodo);
    nodo.parent = null;
  }
  remove() {
    this.parent?.removeChild(this);
  }
  addEventListener(tipo, listener) {
    this.listeners[tipo] = listener;
  }
  removeEventListener(tipo) {
    delete this.listeners[tipo];
  }
  setAttribute(nombre, valor) {
    this.attrs[nombre] = String(valor);
  }
  matches(selector) {
    if (!selector.startsWith("[")) return this.tagName === selector;
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    const valor = this.dataset[claveDatos(coincidencia[1])];
    return (
      valor !== undefined &&
      (coincidencia[2] === undefined || valor === coincidencia[2])
    );
  }
  closest(selector) {
    for (let actual = this; actual; actual = actual.parent)
      if (actual.matches(selector)) return actual;
    return null;
  }
  querySelector(selector) {
    return this.querySelectorAll(selector)[0] || null;
  }
  querySelectorAll(selector) {
    const salida = [];
    const visitar = (actual) => {
      if (actual.matches(selector)) salida.push(actual);
      actual.children.forEach(visitar);
    };
    visitar(this);
    return salida;
  }
}
function crearRaiz() {
  const documento = {
    createElement: (etiqueta) => new Nodo(documento, etiqueta),
  };
  return new Nodo(documento, "root");
}
function textoVisible(nodo) {
  return [nodo.textContent, ...nodo.children.map(textoVisible)].join(" ");
}

const item = Object.freeze({
  comision: {
    referencia: "dco_1234567890123456789012", estado: "borrador",
    fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", motivo: "Reunión",
    codigos_ruta: [], relacion_ref: "rel_1234567890123456789012",
  },
  recibo: { referencia: "rcd_1234567890123456789012", version: 1, registrado_en: "2026-09-20T10:00:00Z", repeticion: false },
});
const cliente = Object.freeze({
  listar: async () => ({ items: [item] }),
  obtener: async () => item,
  crear: async () => item,
});

test("la superficie visible conserva las tres etapas sin expedientes sintéticos ni acciones no conectadas", async () => {
  const fuente = await readFile(
    new URL("vista-recorridos.js", import.meta.url),
    "utf8",
  );
  assert.match(fuente, /montarVistaRecorridosDietas/u);
  assert.match(fuente, /recorridos_solicitante/u);
  assert.match(fuente, /recorridos_jefatura/u);
  assert.match(fuente, /recorridos_gestion/u);
  assert.match(fuente, /boton\.disabled = true/u);
  assert.match(fuente, /montarItinerario/u);
  assert.doesNotMatch(fuente, /COMISIONES_PRESENTACION|obtenerAtlasSinteticoRRHH/u);
  assert.match(fuente, /formularioInicialmenteVisible: false/u);
  assert.doesNotMatch(fuente, /recorridos_titulo_presentacion/u);
  assert.doesNotMatch(fuente, /dietasResumenEtapa|dietas-recorridos-lateral/u);
  assert.doesNotMatch(fuente, /recorridos_limite_operativo/u);
  assert.doesNotMatch(fuente, /dietas-recorridos-conexion-pendiente/u);
  assert.doesNotMatch(fuente, /renderizarEstadoEntrega/u);
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie/u);
});

test("la revisión reutiliza la consulta propia y textos del catálogo", async () => {
  const fuente = await readFile(new URL("vista-recorridos.js", import.meta.url), "utf8");
  assert.match(fuente, /montarVistaBorradoresPropios/u);
  assert.match(fuente, /crearTraductorRevisionDietas/u);
  assert.equal(MENSAJES_REVISION_DIETAS_ES.revision_titulo, "Revisión de comisión");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie/u);
});

test("el formulario propio no recurre a memoria web ni presenta éxito sin respuesta", async () => {
  const fuente = await readFile(
    new URL("vista-borradores-propios.js", import.meta.url),
    "utf8",
  );
  assert.doesNotMatch(
    fuente,
    /localStorage|sessionStorage|indexedDB|document\.cookie/u,
  );
  assert.match(fuente, /cliente\.crear\(solicitud/u);
  assert.match(fuente, /borradores_propios_pendiente_conexion/u);
  assert.match(fuente, /AbortController/u);
});

test("muestra la estructura sin afirmar lista vacía cuando el servicio no está conectado", () => {
  const contenedor = crearRaiz();
  const vista = montarVistaBorradoresPropios(contenedor);
  assert.equal(
    contenedor.querySelector("[data-dietas-borradores-propios]") !== null,
    true,
  );
  assert.equal(
    contenedor.querySelector("[data-dietas-borradores-vacio]"),
    null,
  );
  assert.equal(
    contenedor.querySelectorAll("input").every((control) => control.disabled),
    true,
  );
  assert.match(
    contenedor.querySelector("[data-dietas-borradores-estado]").textContent,
    /pendiente de conexión/u,
  );
  vista.desmontar();
  assert.equal(
    contenedor.querySelector("[data-dietas-borradores-propios]"),
    null,
  );
});

test("navega por las tres etapas sin convertir el selector en autorización", () => {
  const contenedor = crearRaiz();
  const vista = montarVistaRecorridosDietas(contenedor);
  const raiz = contenedor.querySelector("[data-dietas-recorridos]");
  const solicitante = raiz.querySelector(
    '[data-dietas-panel-etapa="solicitante"]',
  );
  assert.equal(solicitante.hidden, false);
  assert.equal(raiz.querySelector("[data-dietas-abrir-nueva-comision]").disabled, true);
  raiz.listeners.click({
    target: raiz.querySelector('[data-dietas-cambiar-etapa="jefatura"]'),
  });
  const jefatura = raiz.querySelector('[data-dietas-panel-etapa="jefatura"]');
  assert.equal(jefatura.hidden, false);
  assert.equal(solicitante.hidden, true);
  raiz.listeners.click({
    target: raiz.querySelector('[data-dietas-cambiar-etapa="gestion"]'),
  });
  assert.equal(
    raiz.querySelector('[data-dietas-panel-etapa="gestion"]') !== null,
    true,
  );
  assert.equal(
    raiz.querySelectorAll("button").some((boton) => boton.disabled),
    true,
  );
  vista.desmontar();
});

test("consulta un borrador mediante GET propio y presenta su recibo real", async () => {
  const contenedor = crearRaiz();
  const llamadas = [];
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: {
      listar: async () => { llamadas.push("listar"); return { items: [item] }; },
      obtener: async () => { llamadas.push("obtener"); return item; },
      crear: async () => item,
    },
  });
  await Promise.resolve();
  await Promise.resolve();
  const raiz = contenedor.querySelector("[data-dietas-recorridos]");
  const boton = raiz.querySelector('[data-dietas-borrador-detalle="dco_1234567890123456789012"]');
  assert.ok(boton);
  await raiz.querySelector("[data-dietas-borradores-propios]").listeners.click({ target: boton });
  assert.deepEqual(llamadas, ["listar", "obtener"]);
  assert.match(textoVisible(raiz), /rcd_1234567890123456789012/u);
  assert.equal(raiz.querySelector("[data-dietas-nueva-comision]").hidden, true);
  vista.desmontar();
});

test("jefatura y gestión indican la falta de conector sin mostrar aprobaciones ni pagos", () => {
  const contenedor = crearRaiz();
  const vista = montarVistaRecorridosDietas(contenedor);
  for (const etapa of ["jefatura", "gestion"]) {
    const panel = contenedor.querySelector(`[data-dietas-panel-etapa="${etapa}"]`);
    assert.match(textoVisible(panel), /no están conectadas/u);
    assert.ok(panel.querySelectorAll("button").every((boton) => boton.disabled));
  }
  assert.doesNotMatch(textoVisible(contenedor), /DIE-2026-|Liquidada|61,88 €/u);
  vista.desmontar();
});

test("mantiene rutas e itinerario ocultos hasta abrir una nueva comisión", async () => {
  const contenedor = crearRaiz();
  const indicador = contenedor.ownerDocument.createElement("p");
  indicador.dataset.cargando = "";
  contenedor.append(indicador);
  let areaMapa;
  let montajes = 0;
  let limpiezas = 0;
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: cliente,
    montarItinerario: (area) => {
      montajes += 1;
      areaMapa = area;
      return { desmontar() { limpiezas += 1; } };
    },
  });
  const raiz = contenedor.querySelector("[data-dietas-recorridos]");
  const nueva = raiz.querySelector("[data-dietas-nueva-comision]");
  assert.equal(nueva.hidden, true);
  assert.equal(montajes, 0);
  raiz.listeners.click({
    target: raiz.querySelector("[data-dietas-abrir-nueva-comision]"),
  });
  await Promise.resolve();
  assert.equal(nueva.hidden, false);
  assert.equal(montajes, 1);
  assert.equal(nueva.querySelector("[data-dietas-area-itinerario]"), areaMapa);
  assert.equal(raiz.querySelector("[data-dietas-borrador-form]").hidden, false);
  assert.equal(contenedor.querySelector("[data-cargando]"), null);
  raiz.listeners.click({ target: raiz.querySelector("[data-dietas-cerrar-nueva-comision]") });
  assert.equal(nueva.hidden, true);
  assert.equal(raiz.querySelector("[data-dietas-borrador-form]").hidden, true);
  assert.equal(limpiezas, 1, "cerrar libera el itinerario ya montado");
  raiz.listeners.click({
    target: raiz.querySelector("[data-dietas-abrir-nueva-comision]"),
  });
  await Promise.resolve();
  assert.equal(montajes, 2, "una reapertura inicia un itinerario nuevo");
  vista.desmontar();
});

test("no recupera un itinerario de una apertura cerrada al reabrir", async () => {
  const contenedor = crearRaiz();
  const pendientes = [];
  let limpiezas = 0;
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: cliente,
    montarItinerario: () => new Promise((resolve) => pendientes.push(resolve)),
  });
  const raiz = contenedor.querySelector("[data-dietas-recorridos]");
  const pulsar = (selector) => raiz.listeners.click({ target: raiz.querySelector(selector) });
  pulsar("[data-dietas-abrir-nueva-comision]");
  pulsar("[data-dietas-cerrar-nueva-comision]");
  pulsar("[data-dietas-abrir-nueva-comision]");
  assert.equal(pendientes.length, 2);
  pendientes[0]({ desmontar() { limpiezas += 1; } });
  await Promise.resolve();
  assert.equal(limpiezas, 1, "la respuesta antigua se libera al resolver");
  pendientes[1]({ desmontar() { limpiezas += 1; } });
  await Promise.resolve();
  vista.desmontar();
  assert.equal(limpiezas, 2, "la apertura vigente se libera al desmontar");
});

test("desmonta un itinerario asíncrono que resuelve después de abandonar la vista", async () => {
  const contenedor = crearRaiz();
  let resolver;
  let limpiezas = 0;
  const pendiente = new Promise((resolve) => {
    resolver = resolve;
  });
  const vista = montarVistaRecorridosDietas(contenedor, {
    clienteBorradores: cliente,
    montarItinerario: () => pendiente,
  });
  const raiz = contenedor.querySelector("[data-dietas-recorridos]");
  raiz.listeners.click({
    target: raiz.querySelector("[data-dietas-abrir-nueva-comision]"),
  });
  vista.desmontar();
  resolver({
    desmontar() {
      limpiezas += 1;
    },
  });
  await Promise.resolve();
  assert.equal(limpiezas, 1);
  assert.equal(contenedor.querySelector("[data-dietas-recorridos]"), null);
});
