import assert from "node:assert/strict";
import test from "node:test";

import {
  ATRIBUCION_OSM_INTERNA,
  ESQUEMA_CALCULO_RUTA_DIETAS,
  ESQUEMA_GEOMETRIA_RUTA_DIETAS,
  validarCalculoRutaDietas,
} from "./contrato.js";
import { crearVisorRutaDietas } from "./mapa-ruta.js";

function calculoValido({ demostracion, motor, origenGeometria }) {
  return {
    esquema: ESQUEMA_CALCULO_RUTA_DIETAS,
    referencia: demostracion ? "RUTA-DEMO-01" : "RUTA-OSRM-01",
    demostracion,
    liquidable: false,
    motor,
    version_grafo: demostracion ? "demo-1" : "osm-1",
    alternativas: [{
      referencia: demostracion ? "RUTA-DEMO-01-A1" : "RUTA-OSRM-01-A1",
      recomendada: true,
      etiqueta: "Ruta recomendada",
      kilometros: 70.2,
      duracion_minutos: 58,
      tramos: [{
        indice: 0,
        origen_codigo: "18087",
        origen_nombre: "Granada",
        destino_codigo: "18140",
        destino_nombre: "Motril",
        kilometros: 70.2,
        duracion_minutos: 58,
      }],
      geometria: {
        esquema: ESQUEMA_GEOMETRIA_RUTA_DIETAS,
        origen: origenGeometria,
        liquidable: false,
        paradas: [
          { etiqueta: "Granada", latitud: 37.1773, longitud: -3.5986 },
          { etiqueta: "Motril", latitud: 36.7447, longitud: -3.518 },
        ],
        trazado: [[37.1773, -3.5986], [36.7447, -3.518]],
      },
    }],
  };
}

test("el contrato impide mezclar motores y geometrías DEMO con producto", () => {
  const demo = calculoValido({
    demostracion: true,
    motor: "simulacion_osrm_demo",
    origenGeometria: "sintetica_demo",
  });
  const producto = calculoValido({
    demostracion: false,
    motor: "osrm_interno",
    origenGeometria: "osrm_interno",
  });
  assert.equal(validarCalculoRutaDietas(demo).demostracion, true);
  assert.equal(validarCalculoRutaDietas(producto).demostracion, false);

  const demoConMotorProducto = structuredClone(demo);
  demoConMotorProducto.motor = "osrm_interno";
  assert.throws(
    () => validarCalculoRutaDietas(demoConMotorProducto),
    /motor de ruta no corresponde al entorno/u,
  );

  const productoConMotorDemo = structuredClone(producto);
  productoConMotorDemo.motor = "simulacion_osrm_demo";
  assert.throws(
    () => validarCalculoRutaDietas(productoConMotorDemo),
    /motor de ruta no corresponde al entorno/u,
  );

  const demoConGeometriaProducto = structuredClone(demo);
  demoConGeometriaProducto.alternativas[0].geometria.origen = "osrm_interno";
  assert.throws(
    () => validarCalculoRutaDietas(demoConGeometriaProducto),
    /geometria de ruta no corresponde al entorno/u,
  );

  const productoConGeometriaDemo = structuredClone(producto);
  productoConGeometriaDemo.alternativas[0].geometria.origen = "sintetica_demo";
  assert.throws(
    () => validarCalculoRutaDietas(productoConGeometriaDemo),
    /geometria de ruta no corresponde al entorno/u,
  );
});

test("el visor entrega los tooltips como nodos de texto y solo solicita teselas internas", () => {
  const descriptor = {
    proveedor: "openstreetmap",
    despliegue: "red_interna",
    plantilla_teselas: "/tiles/osm/{z}/{x}/{y}.png",
    atribucion: ATRIBUCION_OSM_INTERNA,
    geometria: calculoValido({
      demostracion: true,
      motor: "simulacion_osrm_demo",
      origenGeometria: "sintetica_demo",
    }).alternativas[0].geometria,
  };
  descriptor.geometria.paradas[0].etiqueta = "<img src=x onerror=alert(1)>";

  const nodos = [];
  const documento = {
    createElement(nombre) {
      const nodo = { nodeType: 1, nodeName: nombre.toUpperCase(), textContent: "" };
      nodos.push(nodo);
      return nodo;
    },
  };
  const lienzo = {
    ownerDocument: documento,
    innerHTML: "<svg>respaldo</svg>",
    dataset: {},
    replaceChildren() { this.innerHTML = ""; },
  };
  const estado = { textContent: "" };
  const atribucion = { hidden: true, textContent: "" };
  const raiz = { querySelector(selector) {
    if (selector === "[data-dietas-mapa-canvas]") return lienzo;
    if (selector === "[data-dietas-mapa-estado]") return estado;
    if (selector === "[data-dietas-mapa-atribucion]") return atribucion;
    return null;
  } };

  let plantillaTeselas = "";
  let opcionesTeselas;
  let peticionesExternas = 0;
  const tooltips = [];
  const mapa = {
    attributionControl: { setPrefix() {} },
    fitBounds() {},
    remove() {},
  };
  const capa = () => ({ addTo() { return this; } });
  const entorno = {
    fetch() { peticionesExternas += 1; },
    L: {
      map() { return mapa; },
      tileLayer(plantilla, opciones) {
        plantillaTeselas = plantilla;
        opcionesTeselas = opciones;
        return capa();
      },
      polyline() {
        return { ...capa(), getBounds() { return { isValid: () => true }; } };
      },
      circleMarker() {
        return {
          ...capa(),
          bindTooltip(contenido) { tooltips.push(contenido); },
        };
      },
    },
  };

  const montaje = crearVisorRutaDietas({ entorno, permitirTeselas: true })
    .montar({ raiz, descriptor });
  assert.equal(montaje.modo, "openstreetmap_interno");
  assert.equal(plantillaTeselas, "/tiles/osm/{z}/{x}/{y}.png");
  assert.doesNotMatch(plantillaTeselas, /^https?:/iu);
  assert.equal(peticionesExternas, 0);
  assert.equal(typeof tooltips[0], "object");
  assert.strictEqual(tooltips[0], nodos[0]);
  assert.equal(tooltips[0].textContent, "1. <img src=x onerror=alert(1)>");
  assert.equal(Object.hasOwn(tooltips[0], "innerHTML"), false);
  assert.match(opcionesTeselas.attribution, /OpenStreetMap/u);
  assert.match(opcionesTeselas.attribution, /OpenMapTiles/u);
  assert.match(opcionesTeselas.attribution, /href="https:\/\/www\.openstreetmap\.org\/copyright"/u);
  assert.match(opcionesTeselas.attribution, /href="https:\/\/openmaptiles\.org\/"/u);
  assert.match(opcionesTeselas.attribution, /rel="noopener noreferrer"/u);
  assert.equal(atribucion.textContent, "© OpenStreetMap contributors · © OpenMapTiles · servido en red interna");
});
