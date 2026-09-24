import assert from "node:assert/strict";
import test from "node:test";

import {
  ESQUEMA_CALCULO_RUTA_DIETAS,
  ESQUEMA_CATALOGO_RUTAS_DIETAS,
} from "./contrato.js";
import { montarVistaMapaComisionDietas } from "./vista-mapa-comision.js";

const claveDatos = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_todo, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {};
    this.attrs = {}; this.parent = null; this.textContent = ""; this.hidden = false;
  }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children.forEach((nodo) => { nodo.parent = null; }); this.children = []; this.append(...nodos); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  remove() { this.parent?.removeChild(this); }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); }
  matches(selector) {
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    if (!coincidencia) return this.tagName === selector;
    const actual = this.dataset[claveDatos(coincidencia[1])];
    return actual !== undefined && (coincidencia[2] === undefined || actual === coincidencia[2]);
  }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) {
    const salida = [];
    const visitar = (nodo) => { if (nodo.matches(selector)) salida.push(nodo); nodo.children.forEach(visitar); };
    visitar(this); return salida;
  }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }

function catalogo() {
  return {
    esquema: ESQUEMA_CATALOGO_RUTAS_DIETAS, demostracion: false, completo: false, version: "granada-v1",
    puntos: [
      { codigo: "18087", nombre: "Granada", tipo: "municipio", municipio_codigo: "18087", municipio_nombre: "Granada" },
      { codigo: "18140", nombre: "Motril", tipo: "municipio", municipio_codigo: "18140", municipio_nombre: "Motril" },
    ],
  };
}
function calculo() {
  const trazado = [[37.1773, -3.5986], [36.7447, -3.518]];
  return {
    esquema: ESQUEMA_CALCULO_RUTA_DIETAS, referencia: "RUTA-OSRM-1234567890ABCDEF", demostracion: false,
    liquidable: false, motor: "osrm_interno", version_grafo: "granada-v1",
    alternativas: [{
      referencia: "RUTA-OSRM-1234567890ABCDEF-A1", recomendada: true, etiqueta: "OSRM A1",
      kilometros: 70.4, duracion_minutos: 55,
      tramos: [{ indice: 0, origen_codigo: "18087", origen_nombre: "Granada", destino_codigo: "18140", destino_nombre: "Motril", kilometros: 70.4, duracion_minutos: 55, trazado }],
      geometria: { esquema: "vec.dietas.geometria-ruta.v1", origen: "osrm_interno", liquidable: false,
        paradas: [{ etiqueta: "Granada", latitud: 37.1773, longitud: -3.5986 }, { etiqueta: "Motril", latitud: 36.7447, longitud: -3.518 }], trazado,
        tramos: [{ indice: 0, trazado }] },
    }],
  };
}

test("solo monta OSM tras un cálculo OSRM válido y bloquea guardar antes", async () => {
  const r = raiz(); const montajes = []; const avisos = []; let desmontajes = 0;
  const vista = await montarVistaMapaComisionDietas({
    raiz: r, codigos: ["18087", "18140"], anunciar: (...valor) => avisos.push(valor),
    calculador: { async obtenerCatalogo() { return catalogo(); }, async calcular() { return calculo(); } },
    visorRuta: { montar(entrada) { montajes.push(entrada); return { desmontar() { desmontajes += 1; } }; } },
  });
  assert.throws(() => vista.obtenerCalculoParaGuardar(["18087", "18140"]), /Calcule la ruta/u);
  await vista.calcular();
  assert.equal(montajes.length, 1);
  assert.equal(montajes[0].descriptor.geometria.origen, "osrm_interno");
  assert.equal(montajes[0].descriptor.despliegue, "red_interna");
  assert.equal(vista.obtenerCalculoParaGuardar(["18087", "18140"]).motor, "osrm_interno");
  assert.ok(avisos.some(([mensaje]) => /Ruta calculada por el puerto interno/u.test(mensaje)));
  vista.establecerCodigos(["18140", "18087"]);
  assert.equal(desmontajes, 1);
  assert.throws(() => vista.obtenerCalculoParaGuardar(["18140", "18087"]), /Calcule la ruta/u);
  vista.desmontar();
  assert.equal(r.querySelector("[data-dietas-mapa-comision]"), null);
});

test("cancela el cálculo pendiente cuando cambia la ruta o se desmonta", async () => {
  const r = raiz(); let señal;
  const vista = await montarVistaMapaComisionDietas({
    raiz: r, codigos: ["18087", "18140"],
    calculador: { async obtenerCatalogo() { return catalogo(); }, calcular(_solicitud, opciones) {
      señal = opciones.signal;
      return new Promise(() => {});
    } },
    visorRuta: { montar() { throw new Error("no debe montar sin resultado"); } },
  });
  void vista.calcular();
  await new Promise((resolver) => setImmediate(resolver));
  vista.establecerCodigos(["18140", "18087"]);
  assert.equal(señal.aborted, true);
  vista.desmontar();
});
