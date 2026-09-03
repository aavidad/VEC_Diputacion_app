import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearAdaptadorPresentacion } from "./portal-presentacion-adaptador.js";
import {
  cargarCatalogoModulosInterno,
  crearCatalogoModulosDesdeManifiestos,
  extraerModulosEnvelopeCanonico,
} from "./portal-catalogo-modulos.js";
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

function manifiestoContratacionTemporal() {
  return {
    id: "vec.module.contratacion_temporal",
    name_key: "ui.vec.module.contratacion_temporal.name",
    description_key: "ui.vec.module.contratacion_temporal.description",
    version: "v0.2.0",
    group: "recursos_humanos",
    base_path: "/modules/contratacion-temporal",
    permissions: [{
      key: "contratacion_temporal.cuadro.consultar",
      label_key: "ui.permission.contratacion_temporal.cuadro",
    }],
    menu: [{
      id: "contratacion_temporal.cuadro",
      module_id: "vec.module.contratacion_temporal",
      label_key: "ui.vec.menu.contratacion_temporal.cuadro",
      path: "/modules/contratacion-temporal/cuadro",
      icon: "layout-dashboard",
      group: "modulo_contratacion_temporal",
      order: 100,
      required_permissions: ["contratacion_temporal.cuadro.consultar"],
    }],
  };
}

const TRADUCCIONES_CONTRATACION_TEMPORAL = Object.freeze({
  "ui.vec.module.contratacion_temporal.name": "Contratación temporal",
  "ui.vec.module.contratacion_temporal.description": "Expedientes de contratación temporal",
});

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
  assert.match(cargando, /data-modulo-portal="bolsa" disabled aria-disabled="true" aria-busy="true"/);
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

test("el catálogo interno consume solo el envelope canónico y omite credenciales y caché", async () => {
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    if (ruta === "/api/vec/modules") {
      return respuestaJSON({ data: { modules: [manifiestoContratacionTemporal()] } });
    }
    return respuestaJSON(TRADUCCIONES_CONTRATACION_TEMPORAL);
  };
  const catalogo = await cargarCatalogoModulosInterno(fetchImpl);
  assert.equal(catalogo.length, 1);
  assert.deepEqual(catalogo[0], {
    clave: "contratacion_temporal",
    sigla: "CON",
    titulo: "Contratación temporal",
    texto: "Expedientes de contratación temporal",
    version: "v0.2.0",
    grupo: "recursos_humanos",
    rutaBase: "/modules/contratacion-temporal",
  });
  assert.deepEqual(llamadas.map(({ ruta }) => ruta), ["/api/vec/modules", "/locales/es.json"]);
  const cabeceraAutoridad = ["Author", "ization"].join("");
  for (const { opciones } of llamadas) {
    assert.equal(opciones.method, "GET");
    assert.equal(opciones.credentials, "omit");
    assert.equal(opciones.cache, "no-store");
    assert.deepEqual(opciones.headers, { Accept: "application/json" });
    assert.equal(Object.hasOwn(opciones.headers, cabeceraAutoridad), false);
  }
});

test("el catálogo rechaza raíces raw y envolturas ausentes, extra o ambiguas", () => {
  const modules = [manifiestoContratacionTemporal()];
  assert.deepEqual(extraerModulosEnvelopeCanonico({ data: { modules } }), modules);
  for (const invalida of [
    { modules },
    { data: { modules }, extra: true },
    { data: { modules, extra: true } },
    { data: { modules }, modules },
    { data: {} },
    { data: null },
    { data: { modules: null } },
  ]) assert.throws(() => extraerModulosEnvelopeCanonico(invalida), /respuesta de manifiestos no válida/);
});

test("el catálogo rechaza manifiestos y colecciones internas no canónicos", () => {
  const conCampoExtra = { ...manifiestoContratacionTemporal(), estado: "activo" };
  assert.throws(() => crearCatalogoModulosDesdeManifiestos(
    [conCampoExtra], TRADUCCIONES_CONTRATACION_TEMPORAL,
  ), /manifiesto de módulo no válido/);

  const permisoIncompleto = manifiestoContratacionTemporal();
  permisoIncompleto.permissions = [{ key: "contratacion_temporal.cuadro.consultar" }];
  assert.throws(() => crearCatalogoModulosDesdeManifiestos(
    [permisoIncompleto], TRADUCCIONES_CONTRATACION_TEMPORAL,
  ), /permiso de módulo no válido/);

  const menuAmbiguo = manifiestoContratacionTemporal();
  menuAmbiguo.menu[0].extra = true;
  assert.throws(() => crearCatalogoModulosDesdeManifiestos(
    [menuAmbiguo], TRADUCCIONES_CONTRATACION_TEMPORAL,
  ), /entrada de menú no válida/);
});

test("CT inventariado queda visible no_disponible si falla su carga real", async () => {
  let cargasContratacion = 0;
  const clavesTraducidas = [];
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL,
  );
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => catalogo,
    traducir: (clave) => {
      clavesTraducidas.push(clave);
      return `i18n:${clave}`;
    },
    cargadoresPresentacion: {
      base: async () => { throw new Error("la presentación no debe cargarse"); },
    },
    cargadoresInternos: {
      contratacion_temporal: async () => {
        cargasContratacion += 1;
        throw new Error("CT no disponible");
      },
    },
  });
  await coordinador.cargarInterno();
  assert.deepEqual(coordinador.resolverAcceso("contratacion_temporal"), {
    disponible: false,
    vista: "",
    estado: "no_disponible",
    textoEstado: "i18n:estado_modulo_no_disponible_titulo",
  });
  const navegacion = coordinador.renderizarNavegacion(true, "portal");
  assert.match(navegacion, /data-modulo-portal="contratacion_temporal" disabled aria-disabled="true"/);
  assert.match(navegacion, />i18n:estado_modulo_no_disponible_titulo<\/span>/);
  assert.doesNotMatch(navegacion, /data-vista=/);
  assert.deepEqual(clavesTraducidas, [
    "estado_modulo_no_disponible_titulo",
    "estado_modulo_no_disponible_titulo",
  ]);
  assert.equal(await coordinador.montarVista("contratacion-temporal", raizFalsa()), false);
  assert.equal(cargasContratacion, 1);
});

test("CT interno se activa solo después de una consulta autorizada", async () => {
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL,
  );
  let consultas = 0;
  let montajes = 0;
  const fuente = Object.freeze({
    capacidades: Object.freeze([
      "contratacion_temporal.cuadro.consultar",
      "contratacion_temporal.expediente.consultar",
    ]),
    async listar() { consultas += 1; return { expedientes: [] }; },
    async obtener() { throw new Error("sin expedientes"); },
    async ejecutar() { throw new Error("solo lectura"); },
  });
  const presentador = {
    obtenerEstado: () => ({}),
    cargar: async () => ({}),
  };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => catalogo,
    entorno: { fetch: async () => { throw new Error("no debe usarse"); } },
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => ({}) },
        adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => fuente },
        presentador: {
          crearPresentadorExpedientesContratacionTemporal: () => presentador,
        },
        vista: {
          montarModuloContratacionTemporal: async () => {
            montajes += 1;
            return { desmontar() {} };
          },
        },
      }),
    },
  });
  await coordinador.cargarInterno();
  assert.equal(consultas, 1);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
  assert.equal(await coordinador.montarVista("contratacion-temporal", raizFalsa()), true);
  assert.equal(montajes, 1);
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
  const versionCoordinador = "20260831-ct-catalogo-v1";
  const versionPortal = "20260831-ct-catalogo-v1";
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
