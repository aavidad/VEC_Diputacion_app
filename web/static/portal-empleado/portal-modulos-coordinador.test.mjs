import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearAdaptadorPresentacion } from "./portal-presentacion-adaptador.js";
import {
  crearCoordinadorModulosPortal,
  moduloDeVistaPortal,
  resolverCargasModularesPresentacion,
  rutaDeVistaPortal,
} from "./portal-modulos-coordinador.js";

function raizFalsa() {
  const eventos = new Map();
  return {
    eventos,
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    replaceChildren() { this.innerHTML = ""; },
    contains() { return true; },
    querySelector() { return null; },
    querySelectorAll() { return []; },
    setAttribute() {},
    removeAttribute() {},
  };
}

const VERSION_GRAFO = "granada-buffer-osrm-v1-53aba0ad43c4";

function respuestaOSRM() {
  return {
    code: "Ok",
    engine: "osrm_on_premise",
    route_scope: "Granada provincia + 15 km",
    data_version: VERSION_GRAFO,
    routes: [{
      distance: 140_800,
      duration: 6_600,
      legs: [
        { distance: 70_400, duration: 3_300 },
        { distance: 70_400, duration: 3_300 },
      ],
      geometry: {
        type: "LineString",
        coordinates: [
          [-3.59869101, 37.17428891],
          [-3.56, 36.95],
          [-3.52045559, 36.74535308],
          [-3.56, 36.95],
          [-3.59869101, 37.17428891],
        ],
      },
    }],
    waypoints: [],
  };
}

function respuestaJSON(datos) {
  return new Response(JSON.stringify(datos), {
    status: 200,
    headers: { "Content-Type": "application/json; charset=UTF-8" },
  });
}

function crearCoordinador({ fetchImpl = async () => respuestaJSON(respuestaOSRM()), anunciar = () => {} } = {}) {
  return crearCoordinadorModulosPortal({
    escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;"),
    anunciar,
    confirmarOperacion: () => true,
    entorno: { location: { origin: "http://127.0.0.2:8081" }, fetch: fetchImpl },
  });
}

test("el administrador solo compone Bolsa con su ContextoActor", async () => {
  const datosBolsa = obtenerDatosPresentacion("administrador");
  const coordinador = crearCoordinador();
  const contextoBolsa = await coordinador.cargarPresentacion(datosBolsa.sesion);
  const adaptadorBolsa = crearAdaptadorPresentacion({ datosIniciales: datosBolsa, contextoActor: contextoBolsa });
  assert.equal(adaptadorBolsa.identidad, contextoBolsa);
  assert.equal(adaptadorBolsa.actor, contextoBolsa.actor.actor_ref);
  assert.equal(coordinador.obtenerContextoBolsa(), contextoBolsa);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, true);
  assert.equal(coordinador.resolverAcceso("cronos", true).disponible, false);
  assert.equal(coordinador.resolverAcceso("cronos", true).estado, "denegado");
  assert.equal(coordinador.resolverAcceso("dietas", true).disponible, false);
  assert.equal(coordinador.resolverAcceso("dietas", true).estado, "denegado");
});

test("el portal muestra el catálogo completo y dentro de cada módulo solo el acceso activo", async () => {
  const coordinador = crearCoordinador();
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("tecnico").sesion);
  const portal = coordinador.renderizarNavegacion(true, "portal");
  assert.equal((portal.match(/data-modulo-portal=/g) || []).length, 13);
  assert.equal((portal.match(/modulo-habilitado/g) || []).length, 2);

  const bolsa = coordinador.renderizarNavegacion(true, "bolsa");
  assert.equal((bolsa.match(/data-modulo-portal=/g) || []).length, 1);
  assert.match(bolsa, /data-modulo-portal="bolsa"/);
  assert.doesNotMatch(bolsa, /data-modulo-portal="cronos"/);

  const cronos = coordinador.renderizarNavegacion(true, "cronos");
  assert.equal((cronos.match(/data-modulo-portal=/g) || []).length, 1);
  assert.match(cronos, /data-modulo-portal="cronos"/);
  assert.doesNotMatch(cronos, /data-modulo-portal="bolsa"/);
});

test("Bolsa conserva su estado de API y puede abrir Elaboración sin depender del panel", async () => {
  const coordinador = crearCoordinador();
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("tecnico").sesion);
  const acceso = coordinador.resolverAcceso("bolsa", {
    disponible: true,
    vista: "elaboracion",
    estado: "disponible",
    etiqueta: "Borradores disponibles",
  });
  assert.equal(acceso.disponible, true);
  assert.equal(acceso.vista, "elaboracion");

  const cargando = coordinador.renderizarNavegacion({
    disponible: false,
    vista: "",
    estado: "cargando",
    etiqueta: "Comprobando acceso a borradores",
  }, "bolsa");
  assert.match(cargando, /data-modulo-portal="bolsa" disabled aria-busy="true"/);
  assert.match(cargando, />Comprobando<\/span>/);

  const denegado = coordinador.renderizarNavegacion({
    disponible: false,
    vista: "",
    estado: "denegado",
    etiqueta: "Sin permiso para gestionar borradores",
  }, "bolsa");
  assert.match(denegado, />Sin permiso<\/span>/);
  assert.doesNotMatch(denegado, /data-vista=/);
});

test("el funcionario comparte una sola identidad y solo compone Cronos y Dietas", async () => {
  const coordinador = crearCoordinador();
  const contextoBolsa = await coordinador.cargarPresentacion(
    obtenerDatosPresentacion("funcionario").sesion,
  );
  assert.equal(contextoBolsa, null);
  assert.equal(coordinador.obtenerContextoBolsa(), null);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, false);
  assert.equal(coordinador.resolverAcceso("cronos", true).disponible, true);
  assert.equal(coordinador.resolverAcceso("dietas", true).disponible, true);
  const navegacion = coordinador.renderizarNavegacion(true, "portal", (vista) => ["cronos", "dietas"].includes(vista));
  assert.equal((navegacion.match(/modulo-habilitado/g) || []).length, 2);
  assert.match(navegacion, /data-modulo-portal="bolsa"[^>]*disabled/u);
});

test("las rutas estables no mezclan el submenú de Bolsa con los módulos personales", () => {
  assert.equal(rutaDeVistaPortal("portal"), "#portal");
  assert.equal(rutaDeVistaPortal("resumen"), "#bolsa/resumen");
  assert.equal(rutaDeVistaPortal("cronos"), "#cronos");
  assert.equal(rutaDeVistaPortal("dietas"), "#dietas");
  assert.equal(moduloDeVistaPortal("convocatorias"), "bolsa");
  assert.equal(moduloDeVistaPortal("cronos"), "cronos");
});

test("Cronos y Dietas montan contenido administrativo y nunca dejan el área en blanco", async () => {
  const coordinador = crearCoordinador();
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("cronos", raiz), true);
  assert.match(raiz.innerHTML, /class="cronos-area"/);
  assert.match(raiz.innerHTML, /Descargar recibo/);
  assert.equal(await coordinador.montarVista("dietas", raiz), true);
  assert.match(raiz.innerHTML, /data-modulo="dietas"/);
  assert.match(raiz.innerHTML, /comisiones de servicio/i);
  coordinador.desmontarVistaActual();
});

test("Dietas calcula con el mediador OSRM real de presentación y nunca con simulación", async () => {
  const llamadas = [];
  const anuncios = [];
  const coordinador = crearCoordinador({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return respuestaJSON(respuestaOSRM());
    },
    anunciar: (mensaje, tipo) => anuncios.push({ mensaje, tipo }),
  });
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("dietas", raiz), true);

  const botonCalcular = {
    closest(selector) { return selector === "[data-dietas-ruta-calcular]" ? this : null; },
  };
  await raiz.eventos.get("click")({ target: botonCalcular, preventDefault() {} });

  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].ruta, "/api/presentacion/cartografia/rutas");
  assert.equal(llamadas[0].opciones.method, "POST");
  assert.equal(llamadas[0].opciones.credentials, "omit");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), {
    coordinates: [
      { lat: 37.17428891, lon: -3.59869101, name: "Granada" },
      { lat: 36.74535308, lon: -3.52045559, name: "Motril" },
      { lat: 37.17428891, lon: -3.59869101, name: "Granada" },
    ],
    alternatives: 3,
  });
  assert.match(raiz.innerHTML, /osrm_interno/);
  assert.match(raiz.innerHTML, new RegExp(VERSION_GRAFO));
  assert.match(raiz.innerHTML, /Ruta OSRM interna · primera alternativa/);
  assert.match(raiz.innerHTML, /data-dietas-mapa-ref="borrador-ruta-calculada"/);
  assert.doesNotMatch(raiz.innerHTML, /simulacion_osrm_demo|Croquis SVG sintético DEMO/);
  assert.ok(anuncios.some(({ mensaje }) => /calculada por el puerto interno/i.test(mensaje)));
  coordinador.desmontarVistaActual();
});

test("Dietas falla cerrada sin cliente HTTP y Cronos permanece disponible", async () => {
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: { location: { origin: "http://127.0.0.2:8081" } },
  });
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  assert.equal(coordinador.vistaDisponible("cronos"), true);
  assert.equal(coordinador.vistaDisponible("dietas"), false);
  assert.equal(coordinador.resolverAcceso("dietas").estado, "no_disponible");
  assert.match(coordinador.renderizarNavegacion(true, "dietas"), />No disponible<\/span>/);
});

test("un cargador modular rechazado no contamina módulos independientes", async () => {
  const cargas = await resolverCargasModularesPresentacion({
    contratacion_temporal: async () => Object.freeze({ modulo: "contratacion" }),
    cronos: async () => Object.freeze({ modulo: "cronos" }),
    dietas: async () => { throw new Error("cartografía no disponible"); },
  });
  assert.equal(cargas.contratacion_temporal.disponible, true);
  assert.equal(cargas.cronos.disponible, true);
  assert.equal(cargas.dietas.disponible, false);
  assert.equal(cargas.dietas.estado, "no_disponible");
  assert.equal(cargas.contratacion_temporal.recursos.modulo, "contratacion");
  assert.equal(cargas.cronos.recursos.modulo, "cronos");
});

test("un cargador pendiente queda acotado y no paraliza los resultados independientes", async () => {
  const inicio = Date.now();
  const cargas = await resolverCargasModularesPresentacion({
    contratacion_temporal: async () => Object.freeze({ modulo: "contratacion" }),
    cronos: async () => Object.freeze({ modulo: "cronos" }),
    dietas: async () => new Promise(() => {}),
  }, { limiteMs: 20 });
  assert.ok(Date.now() - inicio < 1_000, "la carga pendiente debe quedar acotada");
  assert.equal(cargas.contratacion_temporal.disponible, true);
  assert.equal(cargas.cronos.disponible, true);
  assert.deepEqual(cargas.dietas, {
    disponible: false,
    estado: "no_disponible",
  });
});

test("un módulo ajeno al ámbito del actor ni siquiera ejecuta su cargador", async () => {
  let cargasDietas = 0;
  const cargas = await resolverCargasModularesPresentacion({
    contratacion_temporal: async () => Object.freeze({ modulo: "contratacion" }),
    dietas: async () => {
      cargasDietas += 1;
      return new Promise(() => {});
    },
  }, { claves: ["contratacion_temporal"], limiteMs: 20 });
  assert.equal(cargasDietas, 0);
  assert.equal(cargas.contratacion_temporal.disponible, true);
  assert.equal(cargas.cronos.estado, "denegado");
  assert.equal(cargas.dietas.estado, "denegado");
});

test("una recarga inválida borra la composición anterior antes de fallar", async () => {
  const coordinador = crearCoordinador();
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("administrador").sesion);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, true);
  await assert.rejects(coordinador.cargarPresentacion(null));
  assert.equal(coordinador.obtenerContextoBolsa(), null);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, false);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, false);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").estado, "denegado");
});

test("una carga válida obsoleta no puede republicar permisos tras otra inválida", async () => {
  let invocacionesBase = 0;
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargadoresPresentacion: {
      base: async () => {
        invocacionesBase += 1;
        if (invocacionesBase === 1) {
          await new Promise((resolver) => setTimeout(resolver, 50));
        }
        const [identidad, catalogo] = await Promise.all([
          import("./identidad/presentacion.js"),
          import("./portal-catalogo-presentacion.js"),
        ]);
        return Object.freeze({ identidad, catalogo });
      },
    },
  });
  const cargaAnterior = coordinador.cargarPresentacion(
    obtenerDatosPresentacion("administrador").sesion,
  );
  await new Promise((resolver) => setTimeout(resolver, 5));
  await assert.rejects(coordinador.cargarPresentacion(null));
  await assert.rejects(cargaAnterior, /carga de presentación sustituida/u);
  assert.equal(coordinador.obtenerContextoBolsa(), null);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, false);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, false);
});

test("el coordinador no autentica ni conserva estado en el navegador", async () => {
  const [fuente, estilos] = await Promise.all([
    readFile(new URL("portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("portal-modulos.css", import.meta.url), "utf8"),
  ]);
  assert.doesNotMatch(fuente, /document\.cookie|localStorage|sessionStorage/);
  assert.match(fuente, /Promise\.allSettled/);
  assert.match(fuente, /LIMITE_CARGA_MODULAR_MS/);
  assert.match(fuente, /composicion = null/);
  assert.match(fuente, /secuenciaCarga/);
  assert.match(fuente, /function capacidadesCronos/);
  assert.match(fuente, /function capacidadesDietas/);
  assert.doesNotMatch(fuente, /^import .*\/modulos\//mu);
  assert.match(fuente, /import\("\.\/modulos\/cronos\/datos-presentacion\.js/);
  assert.match(fuente, /import\("\.\/modulos\/dietas\/adaptador-presentacion\.js/);
  assert.match(fuente, /calculador-rutas-presentacion-osrm\.js/);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/dietas\/calculador-rutas-presentacion\.js"\)/);
  assert.doesNotMatch(fuente, /versionGrafo|granada-buffer-osrm-v/u);
  assert.match(fuente, /recursos\.mapa\.crearVisorRutaDietas\(\{ entorno, permitirTeselas: true \}\)/);
  assert.match(estilos, /data-modulo-catalogo="bolsa"/);
  assert.match(estilos, /data-modulo-catalogo="cronos"/);
  assert.match(estilos, /data-modulo-catalogo="dietas"/);
  assert.match(estilos, /data-modulo-portal="cronos"/);
  assert.match(estilos, /forced-colors: active/);
  assert.doesNotMatch(estilos, /\.tarjeta-modulo-bloqueada/);
});

test("el cache busting de módulos avanza en cascada hasta el HTML", async () => {
  const versionCoordinador = "20260725-aislamiento-modular-v2";
  const versionPortal = "20260725-aislamiento-modular-v2";
  const versionTema = "20260725-aislamiento-modular-v1";
  const versionPulido = "20260720-pulido-escritorio-v2";
  const [portal, html] = await Promise.all([
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  const coordinador = await readFile(
    new URL("portal-modulos-coordinador.js", import.meta.url),
    "utf8",
  );
  assert.match(portal, new RegExp(`portal-modulos-coordinador\\.js\\?v=${versionCoordinador}`));
  assert.match(coordinador, new RegExp(`portal-catalogo-modulos\\.js\\?v=${versionCoordinador}`));
  assert.match(html, new RegExp(`portal\\.js\\?v=${versionPortal}`));
  assert.match(html, new RegExp(`portal\\.css\\?v=${versionTema}`));
  assert.match(html, new RegExp(`expedientes-operativo\\.css\\?v=${versionTema}`));
  assert.match(html, new RegExp(`modulos/dietas/dietas\\.css\\?v=${versionPulido}`));
});
