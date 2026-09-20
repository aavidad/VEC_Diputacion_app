import assert from "node:assert/strict";
import test from "node:test";

import { crearCalculadorRutasDietasPresentacionOSRM } from "./calculador-rutas-presentacion-osrm.js";
import { CAPACIDAD_CONSULTAR_RUTA } from "./contrato.js";
import { montarVistaItinerarioDietas } from "./vista-itinerario.js";
import { obtenerDatosPresentacion } from "../../datos-presentacion.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";

const claveDatos = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_coincidencia, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {};
    this.attrs = {}; this.listeners = {}; this.parent = null; this.textContent = "";
  }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children.forEach((nodo) => { nodo.parent = null; }); this.children = []; this.append(...nodos); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  remove() { this.parent?.removeChild(this); }
  addEventListener(tipo, listener) { this.listeners[tipo] = listener; }
  removeEventListener(tipo) { delete this.listeners[tipo]; }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); }
  matches(selector) {
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    if (!coincidencia) return this.tagName === selector;
    const actual = this.dataset[claveDatos(coincidencia[1])];
    return actual !== undefined && (coincidencia[2] === undefined || actual === coincidencia[2]);
  }
  closest(selector) { for (let nodo = this; nodo; nodo = nodo.parent) if (nodo.matches(selector)) return nodo; return null; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) {
    const salida = [];
    const visitar = (nodo) => { if (nodo.matches(selector)) salida.push(nodo); nodo.children.forEach(visitar); };
    visitar(this); return salida;
  }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
function respuestaJSON(datos) {
  return new Response(JSON.stringify(datos), { status: 200, headers: { "Content-Type": "application/json; charset=UTF-8" } });
}
function respuestaOSRM() {
  return { code: "Ok", engine: "osrm_on_premise", route_scope: "Granada provincia + 15 km", graph_version: "granada-buffer-osrm-v1-53aba0ad43c4", routes: [{
    distance: 140_800, duration: 6_600,
    legs: [{ distance: 70_400, duration: 3_300 }, { distance: 70_400, duration: 3_300 }],
    geometry: { type: "LineString", coordinates: [[-3.59869101, 37.17428891], [-3.52045559, 36.74535308], [-3.59869101, 37.17428891]] },
  }], waypoints: [] };
}
function crearCalculador(llamadas) {
  return crearCalculadorRutasDietasPresentacionOSRM({
    contextoActor: crearContextoActorPresentacionDesdeSesion(obtenerDatosPresentacion("funcionario").sesion),
    capacidades: [CAPACIDAD_CONSULTAR_RUTA],
    fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuestaJSON(respuestaOSRM()); },
  });
}
async function clicar(contenedor, selector) {
  await contenedor.listeners.click({ target: contenedor.querySelector(selector) });
}

test("consulta el puerto OSRM inyectado, muestra catálogo y desmonta el mapa", async () => {
  const llamadas = []; const mapas = []; const avisos = []; const r = raiz(); let mapaDesmontado = false;
  const vista = await montarVistaItinerarioDietas({
    raiz: r, calculador: crearCalculador(llamadas), anunciar: (...valor) => avisos.push(valor),
    visorRuta: { montar({ descriptor }) { mapas.push(descriptor); return { desmontar() { mapaDesmontado = true; } }; } },
  });
  const contenedor = r.querySelector("[data-dietas-itinerario]");
  assert.match(contenedor.querySelector("[data-itinerario-catalogo]").textContent, /175 puntos disponibles/u);
  await clicar(contenedor, "[data-itinerario-calcular]");
  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].ruta, "/api/presentacion/cartografia/rutas");
  assert.equal(llamadas[0].opciones.method, "POST");
  assert.equal(llamadas[0].opciones.credentials, "omit");
  assert.ok(mapas.length === 1, JSON.stringify(avisos));
  assert.equal(mapas[0].geometria.liquidable, false);
  assert.equal(mapas[0].geometria.origen, "osrm_interno");
  assert.ok(avisos.some(([mensaje]) => /calculada por el puerto interno/u.test(mensaje)));
  vista.desmontar();
  assert.equal(r.querySelector("[data-dietas-itinerario]"), null);
  assert.equal(mapaDesmontado, true);
});

test("cancela el cálculo al desmontar y no sustituye una vista posterior", async () => {
  const r = raiz(); let resolver; let senal;
  const calculador = {
    obtenerCatalogo: () => crearCalculador([]).obtenerCatalogo(),
    calcular(_solicitud, { signal }) { senal = signal; return new Promise((resolve) => { resolver = resolve; }); },
  };
  const vista = await montarVistaItinerarioDietas({ raiz: r, calculador, visorRuta: { montar() { throw new Error("no debe montar mapa"); } } });
  const contenedor = r.querySelector("[data-dietas-itinerario]");
  const calculando = clicar(contenedor, "[data-itinerario-calcular]");
  vista.desmontar();
  const ajeno = r.ownerDocument.createElement("section"); ajeno.dataset.ajeno = ""; r.append(ajeno);
  resolver(null); await calculando;
  assert.equal(senal.aborted, true);
  assert.equal(r.querySelector("[data-ajeno]"), ajeno);
  assert.equal(r.querySelector("[data-dietas-itinerario]"), null);
});

test("registra la limpieza antes de que llegue un catálogo diferido", async () => {
  const r = raiz(); let resolver; let senal; let limpiar;
  const catalogoPendiente = new Promise((resolve) => { resolver = resolve; });
  const montaje = montarVistaItinerarioDietas({
    raiz: r,
    calculador: {
      obtenerCatalogo({ signal }) { senal = signal; return catalogoPendiente; },
      calcular: async () => null,
    },
    visorRuta: { montar() { throw new Error("no debe montar mapa"); } },
    registrarDesmontar: (desmontar) => { limpiar = desmontar; },
  });
  assert.equal(typeof limpiar, "function");
  limpiar();
  const ajeno = r.ownerDocument.createElement("section"); ajeno.dataset.ajeno = ""; r.append(ajeno);
  resolver(crearCalculador([]).obtenerCatalogo());
  await montaje;
  assert.equal(senal.aborted, true);
  assert.equal(r.querySelector("[data-ajeno]"), ajeno);
  assert.equal(r.querySelector("[data-dietas-itinerario]"), null);
});

test("muestra un error cerrado si el puerto de rutas falla", async () => {
  const r = raiz(); const avisos = [];
  const vista = await montarVistaItinerarioDietas({
    raiz: r,
    calculador: {
      obtenerCatalogo: () => crearCalculador([]).obtenerCatalogo(),
      calcular: async () => { throw new Error("detalle interno no publicable"); },
    },
    visorRuta: { montar() { throw new Error("no debe montar mapa"); } },
    anunciar: (...mensaje) => avisos.push(mensaje),
  });
  const contenedor = r.querySelector("[data-dietas-itinerario]");
  await clicar(contenedor, "[data-itinerario-calcular]");
  assert.match(contenedor.querySelector("[data-itinerario-error]").textContent, /servicio cartográfico interno/u);
  assert.doesNotMatch(contenedor.querySelector("[data-itinerario-error]").textContent, /detalle interno/u);
  assert.ok(avisos.some(([, nivel]) => nivel === "error"));
  vista.desmontar();
});

test("la vista no importa ni expone comandos económicos", async () => {
  const { readFile } = await import("node:fs/promises");
  const fuente = await readFile(new URL("vista-itinerario.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /adaptador-presentacion|crear_borrador|enviar_validacion|descargarRecibo/u);
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|document\.cookie/u);
  assert.doesNotMatch(fuente, /\} km`|\} min`/u);
});
