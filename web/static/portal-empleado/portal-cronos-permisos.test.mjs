import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { crearCoordinadorModulosPortal, moduloDeVistaPortal, rutaDeVistaPortal } from "./portal-modulos-coordinador.js";
import { montarJornadaCronos } from "./modulos/cronos/vista.js";
import { montarVistaRecorridosCronos } from "./modulos/cronos/vista-recorridos.js";
import { crearTraductorCronos } from "./modulos/cronos/i18n.js";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";

function raizFalsa() {
  const documento = {
    createElement() {
      return {
        dataset: {}, innerHTML: "", listeners: new Map(), atributos: {},
        setAttribute(nombre, valor) { this.atributos[nombre] = valor; },
        addEventListener(tipo, manejar) { this.listeners.set(tipo, manejar); },
        removeEventListener(tipo) { this.listeners.delete(tipo); },
        querySelectorAll() { return []; },
        querySelector() { return null; },
        remove() { this.padre?.quitar(this); },
      };
    },
  };
  return {
    ownerDocument: documento, hijos: [], innerHTML: "",
    append(nodo) { nodo.padre = this; this.hijos.push(nodo); },
    quitar(nodo) { this.hijos = this.hijos.filter((hijo) => hijo !== nodo); },
    replaceChildren() { this.hijos = []; this.innerHTML = ""; },
  };
}

function crearCoordinador(catalogo, consultas) {
  return crearCoordinadorModulosPortal({
    escaparHTML: (texto) => String(texto).replaceAll("&", "&amp;").replaceAll("<", "&lt;"),
    entorno: { fetch: async (...args) => { consultas.push(args); throw new Error("sin consultas de Cronos"); } },
    cargarCatalogoInterno: async () => catalogo,
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("CT no debe cargarse"); },
      cronos: async () => ({ vista: { montarJornadaCronos },
        recorridos: { montarVistaRecorridosCronos }, i18n: { crearTraductorCronos } }),
    },
  });
}

test("la ruta de Permisos comparte Cronos y resuelve el hash directo sin inventar un módulo", async () => {
  assert.equal(moduloDeVistaPortal("cronos-permisos"), "cronos");
  assert.equal(rutaDeVistaPortal("cronos-permisos"), "#cronos-permisos");
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  assert.match(portal, /"cronos-permisos": \[traducirPortal\("cronos_permisos_miga"\), traducirPortal\("cronos_permisos_titulo"\)\]/u);
  const inicio = portal.indexOf("function vistaDesdeHash()");
  const fin = portal.indexOf("function rutaDeVista(vista)", inicio);
  assert.ok(inicio > 0 && fin > inicio);
  const ventana = { location: { hash: "#cronos-permisos" } };
  const reemplazos = [];
  const resolver = runInNewContext(`${portal.slice(inicio, fin)}; vistaDesdeHash`, {
    window: ventana, history: { replaceState: (...args) => reemplazos.push(args) },
    TITULOS: { "cronos-permisos": ["Cronos", "Permisos"] },
    estado: { modoPresentacion: false }, VISTAS_PRESENTACION_VISUALES: new Set(),
  });
  assert.equal(resolver(), "cronos-permisos");
  assert.deepEqual(reemplazos, []);
  ventana.location.hash = "#cronos-permisos/desconocido";
  assert.equal(resolver(), "portal");
  assert.equal(reemplazos.at(-1)[2], "#portal");
});

test("Permisos exige catálogo Cronos; muestra recorrido inerte y limpia listeners al navegar", async () => {
  const consultas = [];
  const coordinador = crearCoordinador([{ clave: "cronos" }], consultas);
  await coordinador.cargarInterno();
  assert.equal(coordinador.resolverAcceso("cronos").disponible, true);
  assert.equal(coordinador.vistaDisponible("cronos-permisos"), true);
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("cronos-permisos", raiz), true);
  assert.equal(raiz.hijos.length, 2);
  const [navegacion, recorrido] = raiz.hijos;
  assert.equal(navegacion.atributos["aria-label"], "Contenido de Cronos");
  assert.match(navegacion.innerHTML, /data-vista="cronos-permisos" aria-current="page"/u);
  assert.match(navegacion.innerHTML, /data-vista="cronos"/u);
  assert.match(recorrido.innerHTML, /data-estado-entrega="no_configurado"/u);
  assert.match(recorrido.innerHTML, /id="cronos-persona"/u);
  assert.match(recorrido.innerHTML, /id="cronos-responsable"[^>]+hidden/u);
  assert.match(recorrido.innerHTML, /id="cronos-rrhh"[^>]+hidden/u);
  for (const campo of ["tipo", "desde", "hasta", "observacion", "documento_ref"]) {
    assert.match(recorrido.innerHTML, new RegExp(`name="${campo}"[^>]*disabled`, "u"));
  }
  assert.match(recorrido.innerHTML, /Registrar solicitud<\/button>/u);
  assert.equal(recorrido.listeners.has("keydown"), true);
  let impedido = false;
  recorrido.listeners.get("submit")({ target: { matches: () => true },
    preventDefault() { impedido = true; } });
  assert.equal(impedido, true);
  assert.deepEqual(consultas, [], "el montaje no consulta datos ni envía solicitudes");
  assert.equal(await coordinador.montarVista("cronos", raiz), true);
  assert.equal(recorrido.listeners.size, 0);
  assert.equal(raiz.hijos.length, 2);
  assert.match(raiz.hijos[0].innerHTML, /data-vista="cronos" aria-current="page"/u);
  assert.match(raiz.hijos[1].innerHTML, /data-estado="no_configurado"/u);
  coordinador.desmontarVistaActual();
  assert.equal(raiz.hijos.length, 0);
  assert.deepEqual(consultas, []);
});

test("sin concesión positiva de Cronos no carga ni monta Permisos", async () => {
  const consultas = [];
  const coordinador = crearCoordinador([], consultas);
  await coordinador.cargarInterno();
  assert.equal(coordinador.resolverAcceso("cronos").disponible, false);
  assert.equal(coordinador.vistaDisponible("cronos-permisos"), false);
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("cronos-permisos", raiz), false);
  assert.deepEqual(raiz.hijos, []);
  assert.deepEqual(consultas, []);
});

test("la presentación no expone la subvista interna aunque muestre Cronos", async () => {
  const coordinador = crearCoordinadorModulosPortal({ escaparHTML: String });
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  assert.equal(coordinador.vistaDisponible("cronos"), true);
  assert.equal(coordinador.vistaDisponible("cronos-permisos"), false);
  assert.equal(await coordinador.montarVista("cronos-permisos", raizFalsa()), false);
});

test("el cargador interno nominal y el manifiesto contienen el recorrido sin nueva entrada de catálogo", async () => {
  const [coordinador, manifiesto, portal, html] = await Promise.all([
    readFile(new URL("portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("../../interno.manifest", import.meta.url), "utf8"),
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  const cargadorInterno = coordinador.split("const CARGADORES_INTERNOS_PREDETERMINADOS =")[1]
    .split("function capacidadesDietas")[0];
  assert.match(cargadorInterno, /import\("\.\/modulos\/cronos\/vista-recorridos\.js\?v=20260924-cronos-integrado-v1"\)/u);
  assert.match(cargadorInterno, /import\("\.\/modulos\/cronos\/i18n\.js\?v=20260924-cronos-integrado-v1"\)/u);
  assert.match(manifiesto, /static\/portal-empleado\/modulos\/cronos\/vista-recorridos\.js/u);
  assert.match(manifiesto, /static\/portal-empleado\/modulos\/cronos\/permisos\.css/u);
  assert.match(manifiesto, /static\/portal-empleado\/modulos\/cronos\/i18n-permisos\.js/u);
  assert.match(portal, /portal-modulos-coordinador\.js\?v=20260924-web-c-v2/u);
  assert.match(html, /portal\.js\?v=20260924-web-c-v2/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-f2-dietas-consulta-v2/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-dietas-consulta-v2/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-f2-cronos-permisos-v2/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cronos-permisos-v2/u);
  assert.match(portal, /portal-i18n\.js\?v=20260924-f2-cronos-permisos-v2/u);
  assert.match(coordinador, /portal-i18n\.js\?v=20260924-f2-cronos-permisos-v2/u);
  assert.doesNotMatch(html, /data-vista="cronos-permisos"/u);
});
