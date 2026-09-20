import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";
import { montarVistaRecorridosDietas } from "./vista-recorridos.js";

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

test("la superficie visible conserva las tres etapas y deja las acciones no conectadas deshabilitadas", async () => {
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
  assert.match(fuente, /obtenerAtlasSinteticoRRHH/u);
  assert.match(fuente, /recorridos_limite_operativo/u);
  assert.doesNotMatch(fuente, /renderizarEstadoEntrega/u);
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie/u);
});

test("los indicadores reutilizan las tarjetas KPI corporativas", async () => {
  const fuente = await readFile(new URL("vista-recorridos.js", import.meta.url), "utf8");
  assert.match(fuente, /rejilla-kpi dietas-kpi/u);
  assert.match(fuente, /tarjeta-kpi/u);
  assert.match(fuente, /icono-kpi/u);
  assert.match(fuente, /valor-kpi/u);
  assert.match(fuente, /etiqueta-kpi/u);
  assert.doesNotMatch(fuente, /dietas-presentacion-kpis/u);
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

test("selecciona localmente otra comisión sin habilitar efectos", () => {
  const contenedor = crearRaiz();
  const vista = montarVistaRecorridosDietas(contenedor);
  const raiz = contenedor.querySelector("[data-dietas-recorridos]");
  raiz.listeners.click({ target: raiz.querySelector('[data-dietas-seleccionar-comision="DIE-2026-0091"]') });
  const detalle = raiz.querySelector('[data-dietas-detalle-presentacion="DIE-2026-0091"]');
  assert.equal(detalle.hidden, false);
  assert.equal(raiz.querySelectorAll("button").some((boton) => boton.disabled), true);
  vista.desmontar();
});

test("sustituye el indicador inicial y conserva el área del mapa al cambiar de etapa", () => {
  const contenedor = crearRaiz();
  const indicador = contenedor.ownerDocument.createElement("p");
  indicador.dataset.cargando = "";
  contenedor.append(indicador);
  let areaMapa;
  const vista = montarVistaRecorridosDietas(contenedor, {
    montarItinerario: (area) => {
      areaMapa = area;
      return { desmontar() {} };
    },
  });
  const raiz = contenedor.querySelector("[data-dietas-recorridos]");
  const panelSolicitante = raiz.querySelector(
    '[data-dietas-panel-etapa="solicitante"]',
  );
  assert.ok(
    panelSolicitante.children.indexOf(areaMapa) <
      panelSolicitante.children.indexOf(
        panelSolicitante.querySelector("[data-dietas-area-borradores]"),
      ),
    "el itinerario debe aparecer antes que los borradores",
  );
  assert.equal(contenedor.querySelector("[data-cargando]"), null);
  raiz.listeners.click({
    target: raiz.querySelector('[data-dietas-cambiar-etapa="jefatura"]'),
  });
  raiz.listeners.click({
    target: raiz.querySelector('[data-dietas-cambiar-etapa="solicitante"]'),
  });
  assert.equal(raiz.querySelector("[data-dietas-area-itinerario]"), areaMapa);
  vista.desmontar();
});

test("desmonta un itinerario asíncrono que resuelve después de abandonar la vista", async () => {
  const contenedor = crearRaiz();
  let resolver;
  let limpiezas = 0;
  const pendiente = new Promise((resolve) => {
    resolver = resolve;
  });
  const vista = montarVistaRecorridosDietas(contenedor, {
    montarItinerario: () => pendiente,
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
