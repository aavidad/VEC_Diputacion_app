import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { crearCoordinadorModulosPortal, moduloDeVistaPortal, rutaDeVistaPortal } from "./portal-modulos-coordinador.js";
import * as saldo from "./modulos/cronos/vista-saldo-conectado.js";
import * as remoto from "./modulos/cronos/vista-remoto.js";
import * as movimientos from "./modulos/cronos/vista-movimientos-conectado.js";
import * as movimientosPropios from "./modulos/cronos/vista-movimientos-propios.js";
import * as permisosPropios from "./modulos/cronos/vista-permisos-propios.js";
import * as clienteSaldo from "./modulos/cronos/cliente-saldo-http.js";
import * as clienteRemoto from "./modulos/cronos/cliente-remoto-http.js";
import * as clienteSolicitudes from "./modulos/cronos/cliente-solicitudes-http.js";
import * as i18n from "./modulos/cronos/i18n.js";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { exigirRenovado } from "./versiones-cache.test-helper.mjs";

function raizFalsa() {
  const documento = {
    createElement() {
      return {
        ownerDocument: documento, hijos: [], dataset: {}, innerHTML: "", listeners: new Map(), atributos: {},
        setAttribute(nombre, valor) { this.atributos[nombre] = valor; },
        addEventListener(tipo, manejar) { this.listeners.set(tipo, manejar); },
        removeEventListener(tipo) { this.listeners.delete(tipo); },
        append(nodo) { nodo.padre = this; this.hijos.push(nodo); },
        quitar(nodo) { this.hijos = this.hijos.filter((hijo) => hijo !== nodo); },
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
    entorno: { fetch: async (ruta) => { consultas.push(String(ruta).split("?")[0]);
      return new Response("{}", { status: 404, headers: { "content-type": "application/json" } }); } },
    cargarCatalogoInterno: async () => catalogo,
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("CT no debe cargarse"); },
      cronos: async () => ({ saldo, remoto, movimientos, movimientosPropios, permisosPropios,
        clienteSaldo, clienteRemoto, clienteSolicitudes, i18n }),
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

const esperar = async () => { for (let i = 0; i < 10; i++) await new Promise((resolve) => setImmediate(resolve)); };

test("Permisos monta los permisos propios; con la API en 404 muestra su estado y limpia listeners al navegar", async () => {
  const consultas = [];
  const coordinador = crearCoordinador([{ clave: "cronos" }], consultas);
  await coordinador.cargarInterno();
  assert.equal(coordinador.resolverAcceso("cronos").disponible, true);
  assert.equal(coordinador.vistaDisponible("cronos-permisos"), true);
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("cronos-permisos", raiz), true);
  await esperar();
  assert.equal(raiz.hijos.length, 2);
  const [navegacion, permisos] = raiz.hijos;
  assert.equal(navegacion.atributos["aria-label"], "Contenido de Cronos");
  assert.match(navegacion.innerHTML, /data-vista="cronos-permisos" aria-current="page"/u);
  assert.match(navegacion.innerHTML, /data-vista="cronos"/u);
  assert.equal(permisos.dataset.cronosPermisosPropios, "");
  assert.match(permisos.innerHTML, /cronos-permisos-propios" [^>]*data-estado="error"/u);
  assert.doesNotMatch(permisos.innerHTML, /<form/u);
  assert.equal(permisos.listeners.has("click"), true);
  assert.deepEqual(consultas, ["/api/interna/cronos/permisos/propio"]);
  assert.equal(await coordinador.montarVista("cronos", raiz), true);
  await esperar();
  assert.equal(permisos.listeners.size, 0);
  assert.equal(raiz.hijos[1].className, "cronos-encabezado", "encabezado único de «Jornada»");
  assert.deepEqual(raiz.hijos.slice(2).map((nodo) => nodo.dataset.cronosParte), ["saldo", "remoto", "movimientos", "calendario"]);
  assert.match(raiz.hijos[0].innerHTML, /data-vista="cronos" aria-current="page"/u);
  coordinador.desmontarVistaActual();
  assert.equal(raiz.hijos.length, 0);
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
  // Cada eslabón cambiado después de su versión publicada pide una URL nueva
  // y la misma en todos sus importadores. El interno ya no carga jornada ni
  // recorridos; los clientes van sin ?v= para compartir módulo con las vistas.
  const cronosInterno = cargadorInterno.split("cronos: async () => {")[1].split("contratacion_temporal: async")[0];
  assert.doesNotMatch(cronosInterno, /cronos\/vista\.js|vista-recorridos\.js/u);
  for (const vista of ["vista-saldo-conectado.js", "vista-remoto.js", "vista-movimientos-conectado.js",
    "vista-movimientos-propios.js", "vista-permisos-propios.js"]) {
    exigirRenovado(cronosInterno, `./modulos/cronos/${vista}`, ["20260925-tanda-v1", "20260925-cronos-pantallas-v1"]);
    assert.match(manifiesto, new RegExp(`static/portal-empleado/modulos/cronos/${vista.replaceAll(".", "\\.")}`, "u"), vista);
  }
  for (const cliente of ["cliente-saldo-http.js", "cliente-remoto-http.js", "cliente-solicitudes-http.js"]) {
    assert.match(cronosInterno, new RegExp(`import\\("\\./modulos/cronos/${cliente.replaceAll(".", "\\.")}"\\)`, "u"), cliente);
    assert.match(manifiesto, new RegExp(`static/portal-empleado/modulos/cronos/${cliente.replaceAll(".", "\\.")}`, "u"), cliente);
  }
  for (const hoja of ["cronos-conectado.css", "vista-solicitudes.css"]) {
    exigirRenovado(html, `/portal-empleado/modulos/cronos/${hoja}`, "20260925-tanda-v1");
    assert.match(manifiesto, new RegExp(`static/portal-empleado/modulos/cronos/${hoja.replaceAll(".", "\\.")}`, "u"), hoja);
  }
  assert.match(manifiesto, /static\/portal-empleado\/modulos\/cronos\/i18n-solicitudes\.js/u);
  assert.match(manifiesto, /static\/portal-empleado\/modulos\/cronos\/vista-recorridos\.js/u);
  assert.match(manifiesto, /static\/portal-empleado\/modulos\/cronos\/permisos\.css/u);
  assert.match(manifiesto, /static\/portal-empleado\/modulos\/cronos\/i18n-permisos\.js/u);
  exigirRenovado(portal, "./portal-modulos-coordinador.js", ["20260924-web-paradas-periodos-v1", "20260925-tanda-v1", "20260925-cronos-pantallas-v1"]);
  exigirRenovado(html, "/portal-empleado/portal.js", ["20260924-rescate-web-v4", "20260925-tanda-v1", "20260925-cronos-pantallas-v1"]);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-web-c-ayuda-v5/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v5/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-web-c-ayuda-v4/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v4/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-web-c-v3/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-v3/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-f2-dietas-consulta-v2/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-dietas-consulta-v2/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-f2-cronos-permisos-v2/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cronos-permisos-v2/u);
  exigirRenovado([portal, coordinador], "./portal-i18n.js", ["20260924-rescate-web-v4", "20260924-f2-cronos-permisos-v2"]);
  assert.doesNotMatch(html, /data-vista="cronos-permisos"/u);
});

test("coordinador y vistas piden i18n.js y vista-movimientos-propios.js por la misma URL: un solo módulo", async () => {
  const leer = (ruta) => readFile(new URL(ruta, import.meta.url), "utf8");
  const coordinador = await leer("portal-modulos-coordinador.js");
  const cronosInterno = coordinador.split("const CARGADORES_INTERNOS_PREDETERMINADOS =")[1]
    .split("function capacidadesDietas")[0].split("cronos: async () => {")[1].split("contratacion_temporal: async")[0];
  const importadoresI18n = await Promise.all(["vista-saldo-conectado.js", "vista-remoto.js", "vista-movimientos-conectado.js",
    "vista-recorridos.js", "vista.js", "i18n-c4.js"].map((f) => leer(`modulos/cronos/${f}`)));
  const permisos = await leer("modulos/cronos/vista-permisos-propios.js");
  for (const fuente of [cronosInterno, ...importadoresI18n]) assert.doesNotMatch(fuente, /\/i18n\.js["']/u, "sin importación sin versión");
  assert.doesNotMatch(permisos, /vista-movimientos-propios\.js["']/u);
  exigirRenovado([cronosInterno, ...importadoresI18n], "i18n.js", ["20260925-tanda-v1"]);
  exigirRenovado([cronosInterno, permisos], "vista-movimientos-propios.js", ["20260925-cronos-pantallas-v1"]);
  // Los clientes siguen sin ?v= en todos los importadores (misma clase de error para instanceof).
  for (const fuente of [cronosInterno, permisos, ...importadoresI18n]) assert.doesNotMatch(fuente, /cliente-[a-z-]+-http\.js\?v=/u);
});
