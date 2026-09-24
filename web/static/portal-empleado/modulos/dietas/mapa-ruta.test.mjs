import assert from "node:assert/strict";
import test from "node:test";

import {
  ATRIBUCION_OSM_INTERNA,
  ESQUEMA_GEOMETRIA_RUTA_DIETAS,
  PLANTILLA_TESELAS_OSM_INTERNA,
} from "./contrato.js";
import { crearVisorRutaDietas, montarMapaInicialGranadaDietas } from "./mapa-ruta.js";

function escenarioMapa() {
  const eventos = new Map();
  const temporizadores = new Map();
  const atribucionLeaflet = { hidden: false };
  const avisos = [];
  let ultimoTemporizador = 0;
  let retiradas = 0;
  let urlTeselas;
  let opcionesTeselas;
  let opcionesMapa;
  const lienzo = {
    dataset: {},
    ownerDocument: {
      createElement() {
        return { setAttribute(nombre, valor) { this[nombre] = valor; }, textContent: "" };
      },
    },
    replaceChildren() { avisos.length = 0; },
    append(aviso) { avisos.push(aviso); },
    querySelector() { return null; },
  };
  const estado = { textContent: "" };
  const atribucionAlternativa = { hidden: true };
  const raiz = { querySelector(selector) {
    if (selector === "[data-dietas-mapa-canvas]") return lienzo;
    if (selector === "[data-dietas-mapa-estado]") return estado;
    if (selector === "[data-dietas-mapa-atribucion]") return atribucionAlternativa;
    return null;
  } };
  const capaBase = { addTo() { return this; } };
  const mapa = {
    attributionControl: {
      getContainer() { return atribucionLeaflet; },
      setPrefix() {},
    },
    setView() {},
    fitBounds() {},
    remove() { retiradas += 1; },
  };
  const capaTeselas = {
    ...capaBase,
    on(tipo, manejador) { eventos.set(tipo, manejador); return this; },
    off(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
      return this;
    },
  };
  const entorno = {
    setTimeout(tarea) { ultimoTemporizador += 1; temporizadores.set(ultimoTemporizador, tarea); return ultimoTemporizador; },
    clearTimeout(id) { temporizadores.delete(id); },
    L: {
      map(destino, opciones) { assert.strictEqual(destino, lienzo); opcionesMapa = opciones; return mapa; },
      tileLayer(url, opciones) { urlTeselas = url; opcionesTeselas = opciones; return capaTeselas; },
      polyline() { return { ...capaBase, getBounds() { return { isValid: () => true }; } }; },
      circleMarker() { return { ...capaBase, bindTooltip() {} }; },
    },
  };
  return {
    entorno, raiz, eventos, estado, avisos, atribucionLeaflet, atribucionAlternativa,
    opciones: () => ({ urlTeselas, opcionesTeselas, opcionesMapa }),
    retiradas: () => retiradas,
    expirar() { const tarea = temporizadores.get(ultimoTemporizador); assert.ok(tarea); tarea(); },
  };
}

function descriptorOSRM() {
  return {
    proveedor: "openstreetmap",
    despliegue: "red_interna",
    plantilla_teselas: PLANTILLA_TESELAS_OSM_INTERNA,
    atribucion: ATRIBUCION_OSM_INTERNA,
    geometria: {
      esquema: ESQUEMA_GEOMETRIA_RUTA_DIETAS,
      origen: "osrm_interno",
      liquidable: false,
      paradas: [
        { etiqueta: "Granada", latitud: 37.1773, longitud: -3.5986 },
        { etiqueta: "Motril", latitud: 36.7447, longitud: -3.518 },
      ],
      trazado: [[37.1773, -3.5986], [36.7447, -3.518]],
    },
  };
}

test("el mapa base espera teselas internas y no declara completa la cobertura histórica", () => {
  const caso = escenarioMapa();
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  assert.equal(montaje.modo, "mapa_cargando");
  assert.equal(caso.atribucionLeaflet.hidden, true);
  assert.deepEqual(caso.opciones(), {
    urlTeselas: "/tiles/osm/{z}/{x}/{y}.png",
    opcionesTeselas: {
      minZoom: 8, maxNativeZoom: 12, maxZoom: 12, attribution: ATRIBUCION_OSM_INTERNA,
    },
    opcionesMapa: { scrollWheelZoom: false, attributionControl: true, minZoom: 8, maxZoom: 12 },
  });
  caso.eventos.get("load")();
  assert.equal(montaje.modo, "openstreetmap_interno");
  assert.equal(caso.atribucionLeaflet.hidden, false);
  assert.match(caso.estado.textContent, /cobertura completa.*aún no|aún no.*cobertura completa/u);
  montaje.desmontar();
  assert.equal(caso.retiradas(), 1);
});

test("un error parcial inicial impide declarar cargado un mapa incompleto", async () => {
  const caso = escenarioMapa();
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  caso.eventos.get("tileerror")();
  caso.eventos.get("load")();
  assert.equal(montaje.modo, "mapa_no_disponible");
  await Promise.resolve();
  assert.equal(caso.retiradas(), 1);
  assert.equal(caso.atribucionLeaflet.hidden, true);
  assert.equal(caso.avisos[0].role, "status");
  assert.match(caso.estado.textContent, /no está disponible/u);
});

test("el mapa base sigue vigilando teselas después de cargar y se retira si faltan al moverlo", async () => {
  const caso = escenarioMapa();
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  caso.eventos.get("load")();
  caso.eventos.get("loading")();
  caso.eventos.get("tileerror")();
  caso.eventos.get("load")();
  assert.equal(montaje.modo, "mapa_no_disponible");
  await Promise.resolve();
  assert.equal(caso.retiradas(), 1);
  assert.equal(caso.eventos.has("tileerror"), false);
});

test("una nueva carga posterior que no termina vence y retira el mapa", () => {
  const caso = escenarioMapa();
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  caso.eventos.get("load")();
  caso.eventos.get("loading")();
  caso.expirar();
  assert.equal(montaje.modo, "mapa_no_disponible");
  assert.equal(caso.retiradas(), 1);
});

test("al retirar el mapa devuelve el foco del control de zoom al aviso accesible", () => {
  const caso = escenarioMapa();
  const focoZoom = {};
  const lienzo = caso.raiz.querySelector("[data-dietas-mapa-canvas]");
  lienzo.ownerDocument.activeElement = focoZoom;
  lienzo.contains = (elemento) => elemento === focoZoom;
  let recibioFoco = false;
  caso.estado.setAttribute = (nombre, valor) => { caso.estado[nombre] = valor; };
  caso.estado.focus = () => { recibioFoco = true; };
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  caso.eventos.get("load")();
  caso.eventos.get("loading")();
  caso.expirar();
  assert.equal(montaje.modo, "mapa_no_disponible");
  assert.equal(caso.estado.tabindex, "-1");
  assert.equal(recibioFoco, true);
});

test("el visor de ruta acreditada también observa los fallos posteriores sin sustituir la geometría", async () => {
  const caso = escenarioMapa();
  const montaje = crearVisorRutaDietas({ entorno: caso.entorno, permitirTeselas: true })
    .montar({ raiz: caso.raiz, descriptor: descriptorOSRM() });
  assert.equal(montaje.modo, "mapa_cargando");
  assert.equal(caso.atribucionLeaflet.hidden, true);
  caso.eventos.get("load")();
  assert.equal(caso.atribucionLeaflet.hidden, false);
  caso.eventos.get("loading")();
  caso.eventos.get("tileerror")();
  caso.eventos.get("load")();
  assert.equal(montaje.modo, "mapa_no_disponible");
  assert.equal(caso.atribucionAlternativa.hidden, true);
  await Promise.resolve();
  assert.equal(caso.retiradas(), 1);
});

test("un 404 durante la carga y dos imágenes tardías no desmontan Leaflet dentro de su callback", async () => {
  const caso = escenarioMapa();
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  const tileReady404 = () => {
    caso.eventos.get("tileerror")();
    caso.eventos.get("load")();
    // Leaflet 1.9.4 continúa leyendo _map._fadeAnimated tras emitir load.
    if (caso.retiradas() !== 0) throw new TypeError("Cannot read properties of null (reading '_fadeAnimated')");
  };
  assert.doesNotThrow(tileReady404);
  assert.equal(montaje.modo, "mapa_no_disponible");
  assert.match(caso.estado.textContent, /no está disponible/u);
  await Promise.resolve();
  assert.equal(caso.retiradas(), 1);
  const onloadTardio = () => {
    // TileLayer._tileReady comprueba _map y sale sin invocar GridLayer.
    if (caso.retiradas() === 0) caso.eventos.get("load")?.();
  };
  assert.doesNotThrow(onloadTardio);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.doesNotThrow(onloadTardio);
  assert.equal(caso.retiradas(), 1);
});

test("salir de la vista con teselas pendientes desactiva escuchas y onload tardíos", async () => {
  const caso = escenarioMapa();
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  const onloadPendiente = caso.eventos.get("load");
  montaje.desmontar();
  assert.equal(caso.retiradas(), 1);
  assert.equal(caso.eventos.size, 0);
  assert.doesNotThrow(onloadPendiente);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.doesNotThrow(onloadPendiente);
  assert.equal(caso.retiradas(), 1);
});

test("salir tras un error antes de la microtarea retira una sola vez y no reabre el aviso", async () => {
  const caso = escenarioMapa();
  const montaje = montarMapaInicialGranadaDietas({ raiz: caso.raiz, entorno: caso.entorno, permitirTeselas: true });
  caso.eventos.get("tileerror")();
  caso.eventos.get("load")();
  assert.equal(montaje.modo, "mapa_no_disponible");
  assert.equal(caso.retiradas(), 0);
  montaje.desmontar();
  await Promise.resolve();
  assert.equal(caso.retiradas(), 1);
  assert.equal(caso.avisos.length, 0);
});
