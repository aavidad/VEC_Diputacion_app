import assert from "node:assert/strict";
import test from "node:test";

import { montarVistaItinerarioDietas } from "./vista-itinerario.js";
import { crearCalculadorRutasDietasHTTP } from "./calculador-rutas-http.js";
import { CAPACIDAD_CONSULTAR_RUTA } from "./contrato.js";
import { ESQUEMA_CONTEXTO_ACTOR_FRONTEND, validarYCongelarContextoActor } from "../../identidad/contexto-actor.js";

const clave = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_coincidencia, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {};
    this.attrs = {}; this.value = ""; this.textContent = ""; this.hidden = false; this.listeners = {};
  }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children.forEach((nodo) => { nodo.parent = null; }); this.children = []; this.append(...nodos); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  addEventListener(tipo, listener) { this.listeners[tipo] = listener; }
  removeEventListener(tipo) { delete this.listeners[tipo]; }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); }
  getAttribute(nombre) { return this.attrs[nombre] ?? null; }
  matches(selector) {
    if (!selector.startsWith("[")) return this.tagName === selector;
    const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/);
    if (!coincidencia) return false;
    const [, atributo, valor] = coincidencia;
    const actual = this.dataset[clave(atributo)];
    return actual !== undefined && (valor === undefined || actual === valor);
  }
  closest(selector) { for (let nodo = this; nodo; nodo = nodo.parent) if (nodo.matches(selector)) return nodo; return null; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) { const salida = []; const visitar = (nodo) => { if (nodo.matches(selector)) salida.push(nodo); nodo.children.forEach(visitar); }; visitar(this); return salida; }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
function contexto() { return validarYCongelarContextoActor({ esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND, revision: 1, demostracion: false, persona_ref: "per_persona_productiva_dietas_000001", cuenta_ref: "cta_cuenta_productiva_dietas_000001", perfil_ref: "prf_perfil_productivo_dietas_000001", actor: { actor_ref: "act_actor_productivo_dietas_000001", nombre_visible: "Test", iniciales: "TE" }, rol: { clave: "personal", etiqueta: "Personal" }, ambito: { clase: "personal_interno", organizacion_ref: "org_diputacion_granada_productiva_000001", unidad_ref: "uni_unidad_productiva_dietas_000001", modulos: ["dietas"] }, autenticacion: { sesion_ref: "ses_sesion_productiva_dietas_000001", metodo: "kerberos_ad", garantia: "alto" }, resuelto_en: "2026-07-19T10:00:00.000Z" }); }
const punto = (code, name, lat, lon) => ({ code, name, kind: "municipio", municipality_code: code, municipality_name: name, lat, lon, source: "Catálogo", state: "Vigente" });
const catalogo = { generated_at: "2026-07-19T10:00:00Z", province_route_points: [punto("18087", "Granada", 37.17, -3.59), punto("18003", "Albolote", 37.23, -3.65), punto("18140", "Motril", 36.74, -3.51)], province_route_matrix: { matrix_version: "test-v1", route_points_loaded: 3, import_required_before_liquidation: true } };
const ruta = (distance, duration) => ({ distance, duration, legs: [{ distance: 40000, duration: 2000 }, { distance: distance - 40000, duration: duration - 2000 }], geometry: { type: "LineString", coordinates: [[-3.59, 37.17], [-3.65, 37.23], [-3.51, 36.74]] } });
function calculador() { return crearCalculadorRutasDietasHTTP({ contextoActor: contexto(), capacidades: [CAPACIDAD_CONSULTAR_RUTA], fetchImpl: async (url) => new Response(JSON.stringify(url.endsWith("catalog") ? catalogo : { data: { code: "Ok", engine: "osrm_on_premise", route_scope: "provincia", data_version: "test-v1", routes: [ruta(80000, 4000), ruta(84000, 4200), ruta(90000, 4600)] } }), { status: 200, headers: { "content-type": "application/json" } }) }); }

function contenedor(r) { return r.querySelector("[data-dietas-itinerario]"); }
async function clicar(r, selector) { const c = contenedor(r); await c.listeners.click({ target: c.querySelector(selector) }); }

test("itinerario interno monta mapa OSRM, motivo alternativo y ajuste trazable", async () => {
  const r = raiz(); const montajes = []; const avisos = [];
  const visor = { montar({ raiz: mapa, descriptor }) { montajes.push({ mapa, descriptor }); return { desmontar() { montajes.at(-1).desmontado = true; } }; } };
  const vista = await montarVistaItinerarioDietas({ raiz: r, calculador: calculador(), visorRuta: visor, anunciar: (...mensaje) => avisos.push(mensaje) });
  assert.equal(contenedor(r).className, "modulo-dietas");
  await clicar(r, "[data-itinerario-calcular]");
  assert.equal(montajes.length, 1); assert.equal(montajes[0].descriptor.geometria.origen, "osrm_interno");
  const alternativa = contenedor(r).querySelectorAll("[data-itinerario-alternativa]").at(1);
  await contenedor(r).listeners.click({ target: alternativa });
  assert.ok(contenedor(r).querySelector("[data-itinerario-error]"));
  const motivoAlternativa = contenedor(r).querySelectorAll("[data-itinerario-motivo-alternativa]").at(0);
  motivoAlternativa.value = "Obra acreditada en la vía principal";
  await contenedor(r).listeners.click({ target: alternativa });
  const km = contenedor(r).querySelector("[data-itinerario-ajuste-km]");
  const motivoAjuste = contenedor(r).querySelector("[data-itinerario-motivo-ajuste]");
  km.value = "1.5"; motivoAjuste.value = "Desvío acreditado por obras";
  await clicar(r, "[data-itinerario-aplicar-ajuste]");
  assert.equal(contenedor(r).querySelector("[data-itinerario-ajuste-km]").value, "1.5");
  await clicar(r, "[data-itinerario-anadir]");
  const selectorNuevo = contenedor(r).querySelector("[data-itinerario-parada=\"2\"]");
  const vacia = selectorNuevo.children.find((opcion) => opcion.value === "");
  assert.ok(vacia); assert.equal(vacia.selected, true);
  await clicar(r, "[data-itinerario-quitar]");
  assert.equal(contenedor(r).querySelectorAll("[data-itinerario-parada]").length, 3);
  vista.desmontar(); assert.equal(r.children.length, 0); assert.ok(montajes.at(-1).desmontado); assert.ok(avisos.length > 0);
});

test("no borra una vista ajena si el catálogo termina tarde o se sustituye", async () => {
  const r = raiz(); let resolver;
  const pendiente = new Promise((resolve) => { resolver = resolve; });
  const montaje = montarVistaItinerarioDietas({ raiz: r, calculador: { obtenerCatalogo: () => pendiente, calcular: async () => null } });
  const ajeno = r.ownerDocument.createElement("section"); ajeno.dataset.ajeno = ""; r.replaceChildren(ajeno);
  resolver(catalogo); const vista = await montaje;
  assert.equal(r.querySelector("[data-ajeno]"), ajeno); assert.equal(r.querySelector("[data-dietas-itinerario]"), null);
  vista.desmontar(); assert.equal(r.querySelector("[data-ajeno]"), ajeno);
});

test("no filtra errores internos en la alerta visible", async () => {
  const r = raiz(); const secreto = "detalle-interno-no-publicable";
  await montarVistaItinerarioDietas({ raiz: r, calculador: { obtenerCatalogo: async () => { throw new Error(secreto); }, calcular: async () => null } });
  assert.match(contenedor(r).querySelector("[data-itinerario-error]").textContent, /servicio cartográfico interno/u);
  assert.doesNotMatch(contenedor(r).textContent, new RegExp(secreto));
});
