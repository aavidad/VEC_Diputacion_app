import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirRenovado, exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";
import {
  cargarCatalogoModulosInterno,
  crearCatalogoModulosDesdeManifiestos,
  extraerModulosEnvelopeCanonico,
} from "./portal-catalogo-modulos.js";
import {
  CLAVES_MODULOS_VEC_REGISTRADOS,
  crearCoordinadorModulosPortal,
  moduloDeVistaPortal,
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

function raizDietasFalsa() {
  const clave = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
  class Nodo {
    constructor(documento, etiqueta = "div") {
      this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {};
      this.listeners = {}; this.parent = null; this.textContent = "";
    }
    append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
    replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
    removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
    remove() { this.parent?.removeChild(this); }
    addEventListener(tipo, manejador) { this.listeners[tipo] = manejador; }
    removeEventListener(tipo) { delete this.listeners[tipo]; }
    setAttribute() {}
    matches(selector) {
      const coincidencia = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
      if (!coincidencia) return this.tagName === selector;
      const actual = this.dataset[clave(coincidencia[1])];
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
  const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) };
  return new Nodo(documento, "root");
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
        { distance: 70_400, duration: 3_300, geometry: { type: "LineString", coordinates: [
          [-3.59869101, 37.17428891], [-3.56, 36.95], [-3.52045559, 36.74535308],
        ] } },
        { distance: 70_400, duration: 3_300, geometry: { type: "LineString", coordinates: [
          [-3.52045559, 36.74535308], [-3.56, 36.95], [-3.59869101, 37.17428891],
        ] } },
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


test("la consulta inicial tiene timeout y se aborta al desmontar o sustituir", async () => {
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL,
  );
  const señales = [];
  let resolver = false;
  const fuente = {
    capacidades: ["contratacion_temporal.cuadro.consultar"],
    listar({ signal } = {}) {
      señales.push(signal);
      if (resolver) return Promise.resolve({ expedientes: [] });
      return new Promise((_resolver, rechazar) => {
        signal.addEventListener("abort", () => {
          const error = new Error("consulta cancelada");
          error.name = "AbortError";
          rechazar(error);
        }, { once: true });
      });
    },
    async obtener() { throw new Error("sin expedientes"); },
    async ejecutar() { throw new Error("solo lectura"); },
  };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => catalogo,
    limiteCargaModularMs: 20,
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => ({}) },
        adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => fuente },
        presentador: {
          crearPresentadorExpedientesContratacionTemporal: () => ({}),
        },
        vista: { montarModuloContratacionTemporal: async () => ({ desmontar() {} }) },
      }),
    },
  });
  const esperarConsulta = async (total) => {
    for (let intento = 0; intento < 50 && señales.length < total; intento += 1) {
      await new Promise((continuar) => setImmediate(continuar));
    }
    assert.equal(señales.length, total);
  };

  await coordinador.cargarInterno();
  assert.equal(señales[0].aborted, true);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, false);

  const cargaDesmontada = coordinador.cargarInterno();
  await esperarConsulta(2);
  coordinador.desmontarVistaActual();
  await assert.rejects(cargaDesmontada, /carga interna sustituida/);
  assert.equal(señales[1].aborted, true);

  const cargaSustituida = coordinador.cargarInterno();
  await esperarConsulta(3);
  resolver = true;
  await coordinador.cargarInterno();
  await assert.rejects(cargaSustituida, /carga interna sustituida/);
  assert.equal(señales[2].aborted, true);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
});

test("un catálogo interno pendiente deja de bloquear el arranque y admite reintento", async () => {
  const catalogo = Object.freeze([{ clave: "bolsa" }]);
  const senales = [];
  let intentos = 0;
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    limiteCargaModularMs: 20,
    cargarCatalogoInterno: (signal) => {
      senales.push(signal);
      intentos += 1;
      return intentos === 1 ? new Promise(() => {}) : Promise.resolve(catalogo);
    },
  });
  await assert.rejects(coordinador.cargarInterno(), /tiempo agotado.*catálogo/u);
  assert.equal(senales[0].aborted, true);
  assert.deepEqual(coordinador.obtenerCatalogo(), []);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, false);
  await coordinador.cargarInterno();
  assert.equal(coordinador.obtenerCatalogo(), catalogo);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, true);
});

test("un catálogo interno sustituido se aborta y no publica resultados tardíos", async () => {
  let resolverPrimero;
  const catalogoViejo = Object.freeze([{ clave: "bolsa" }]);
  const catalogoNuevo = Object.freeze([{ clave: "contratacion_temporal" }]);
  const senales = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    limiteCargaModularMs: 100,
    cargarCatalogoInterno: (signal) => {
      senales.push(signal);
      return senales.length === 1
        ? new Promise((resolver) => { resolverPrimero = resolver; })
        : Promise.resolve(catalogoNuevo);
    },
    cargadoresInternos: { contratacion_temporal: async () => ({}) },
  });
  const primera = coordinador.cargarInterno();
  const rechazoPrimera = assert.rejects(primera, /carga interna sustituida/u);
  await new Promise((resolver) => setImmediate(resolver));
  await coordinador.cargarInterno();
  assert.equal(senales[0].aborted, true);
  await rechazoPrimera;
  resolverPrimero(catalogoViejo);
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(coordinador.obtenerCatalogo(), catalogoNuevo);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, false);
});

test("el catálogo real aborta ambas consultas al agotarse el límite y falla cerrado", async () => {
  const solicitudes = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    limiteCargaModularMs: 20,
    entorno: { fetch: (ruta, opciones) => {
      solicitudes.push({ ruta, opciones });
      return new Promise(() => {});
    } },
  });
  await assert.rejects(coordinador.cargarInterno(), /tiempo agotado.*catálogo/u);
  assert.deepEqual(solicitudes.map(({ ruta }) => ruta), ["/api/vec/modules", "/locales/es.json"]);
  assert.ok(solicitudes.every(({ opciones }) => opciones.signal.aborted === true));
  assert.deepEqual(coordinador.obtenerCatalogo(), []);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, false);
});

test("las rutas estables no mezclan el submenú de Bolsa con los módulos personales", () => {
  assert.equal(rutaDeVistaPortal("portal"), "#portal");
  assert.equal(rutaDeVistaPortal("resumen"), "#bolsa/resumen");
  assert.equal(rutaDeVistaPortal("cronos"), "#cronos");
  assert.equal(rutaDeVistaPortal("cronos-permisos"), "#cronos-permisos");
  assert.equal(rutaDeVistaPortal("dietas"), "#dietas");
  assert.equal(moduloDeVistaPortal("convocatorias"), "bolsa");
  assert.equal(moduloDeVistaPortal("cronos"), "cronos");
  assert.equal(moduloDeVistaPortal("cronos-permisos"), "cronos");
  assert.equal(moduloDeVistaPortal("vista-no-registrada"), "");
  assert.equal(rutaDeVistaPortal("vista-no-registrada"), "#portal");
});

test("Bolsa usa el montaje común para B12 y B5 sin sondear vistas desconocidas", async () => {
  const montajes = [];
  const desmontajes = [];
  const consultasDisponibilidad = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    montajeBolsa: {
      disponible: (vista) => {
        consultasDisponibilidad.push(vista);
        return ["resumen", "bolsa-candidatos"].includes(vista);
      },
      montar: ({ vista, raiz }) => {
        montajes.push(vista);
        raiz.innerHTML = `<p>${vista}</p>`;
        return { desmontar: () => desmontajes.push(vista) };
      },
    },
  });
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("resumen", raiz), true);
  assert.equal(await coordinador.montarVista("bolsa-candidatos", raiz), true);
  assert.deepEqual(desmontajes, ["resumen"]);
  coordinador.desmontarVistaActual();
  assert.deepEqual(desmontajes, ["resumen", "bolsa-candidatos"]);
  assert.equal(coordinador.vistaGestionada("vista-no-registrada"), false);
  assert.equal(coordinador.vistaDisponible("vista-no-registrada"), false);
  assert.equal(await coordinador.montarVista("vista-no-registrada", raiz), false);
  assert.equal(consultasDisponibilidad.includes("vista-no-registrada"), false);
  assert.deepEqual(montajes, ["resumen", "bolsa-candidatos"]);
});

test("Elaboración se reutiliza al repintar y sustituye solo la referencia o la vista", async () => {
  const montajes = [];
  const abortadas = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    montajeBolsa: {
      disponible: (vista) => ["elaboracion", "resumen"].includes(vista),
      montar: ({ vista, opciones }) => {
        const controlador = new AbortController();
        montajes.push({ vista, referencia: opciones.referencia || "", signal: controlador.signal });
        return { desmontar: () => { controlador.abort(); abortadas.push(vista); } };
      },
    },
  });
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("elaboracion", raiz, { referencia: "DEMO-BORRADOR-001" }), true);
  const inicial = montajes[0];
  assert.equal(await coordinador.montarVista("elaboracion", raiz), true);
  assert.equal(montajes.length, 1);
  assert.equal(inicial.signal.aborted, false);
  assert.equal(await coordinador.montarVista("elaboracion", raiz, { referencia: "DEMO-BORRADOR-002" }), true);
  assert.equal(inicial.signal.aborted, true);
  assert.deepEqual(montajes.map(({ referencia }) => referencia), ["DEMO-BORRADOR-001", "DEMO-BORRADOR-002"]);
  const nueva = montajes[1];
  assert.equal(await coordinador.montarVista("resumen", raiz), true);
  assert.equal(nueva.signal.aborted, true);
  assert.deepEqual(abortadas, ["elaboracion", "elaboracion"]);
});

test("Elaboración reserva el montaje pendiente antes de una reentrada y descarta el resultado sustituido", async () => {
  const pendientes = [];
  const abortadas = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    montajeBolsa: {
      disponible: (vista) => vista === "elaboracion",
      montar: ({ opciones }) => new Promise((resolver) => {
        const controlador = new AbortController();
        pendientes.push({ referencia: opciones.referencia || "", controlador, resolver });
      }),
    },
  });
  const raiz = raizFalsa();
  const primera = coordinador.montarVista("elaboracion", raiz, { referencia: "DEMO-BORRADOR-001" });
  const htmlPendiente = raiz.innerHTML;
  assert.equal(await coordinador.montarVista("elaboracion", raiz), true);
  assert.equal(pendientes.length, 1);
  assert.equal(raiz.innerHTML, htmlPendiente);
  const segunda = coordinador.montarVista("elaboracion", raiz, { referencia: "DEMO-BORRADOR-002" });
  assert.equal(pendientes.length, 2);
  pendientes[0].resolver({ desmontar: () => { pendientes[0].controlador.abort(); abortadas.push("primera"); } });
  assert.equal(await primera, false);
  assert.equal(pendientes[0].controlador.signal.aborted, true);
  pendientes[1].resolver({ desmontar: () => { pendientes[1].controlador.abort(); abortadas.push("segunda"); } });
  assert.equal(await segunda, true);
  coordinador.desmontarVistaActual();
  assert.equal(pendientes[1].controlador.signal.aborted, true);
  assert.deepEqual(abortadas, ["primera", "segunda"]);
});

test("el coordinador no autentica ni conserva estado en el navegador", async () => {
  const [fuenteCoordinador, fuenteCarga, estilos, empleado] = await Promise.all([
    readFile(new URL("portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("portal-modulos-carga.js", import.meta.url), "utf8"),
    readFile(new URL("portal-modulos.css", import.meta.url), "utf8"),
    readFile(new URL("portal-composicion-empleado.js", import.meta.url), "utf8"),
  ]);
  const fuente = `${fuenteCoordinador}\n${fuenteCarga}`;
  assert.doesNotMatch(fuente, /document\.cookie|localStorage|sessionStorage/);
  assert.doesNotMatch(empleado, /document\.cookie|localStorage|sessionStorage/);
  assert.match(fuente, /Promise\.all/);
  assert.match(fuente, /LIMITE_CARGA_MODULAR_MS/);
  assert.match(fuente, /composicion = null/);
  assert.match(fuente, /secuenciaCarga/);
  assert.doesNotMatch(empleado, /function componerCronosVisible|function componerDietasVisible/);
  assert.match(fuenteCoordinador, /componerCronosInterno/);
  assert.match(empleado, /montarVistaSaldoCronos/);
  assert.match(empleado, /montarPermisosPropiosCronos/);
  assert.match(fuenteCoordinador, /componerDietasInternas/);
  assert.doesNotMatch(fuenteCoordinador, /cargarPresentacion|cargadoresPresentacion|resolverCargasModularesPresentacion/);
  assert.doesNotMatch(fuente, /^import .*\/modulos\//mu);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/cronos\/datos-presentacion\.js/);
  assert.match(fuente, /import\("\.\/modulos\/cronos\/vista-saldo-conectado\.js\?v=/);
  assert.match(fuente, /import\("\.\/modulos\/cronos\/vista-permisos-propios\.js\?v=/);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/dietas\/vista-itinerario\.js/);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/dietas\/adaptador-presentacion\.js/);
  assert.doesNotMatch(fuenteCoordinador, /calculador-rutas-presentacion-osrm\.js/);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/dietas\/calculador-rutas-presentacion\.js"\)/);
  assert.doesNotMatch(fuente, /versionGrafo|granada-buffer-osrm-v/u);
  assert.doesNotMatch(empleado, /datos-sinteticos-rrhh/u);
  // El visor de Dietas solo pide teselas del mismo origen: ningún proveedor externo.
  assert.match(empleado, /crearVisorRutaDietas\(\{ entorno, permitirTeselas: true \}\)/u);
  assert.doesNotMatch(empleado, /https?:\/\//u);
  assert.match(estilos, /data-modulo-catalogo="bolsa"/);
  assert.match(estilos, /data-modulo-catalogo="cronos"/);
  assert.match(estilos, /data-modulo-catalogo="dietas"/);
  assert.match(estilos, /data-modulo-portal="cronos"/);
  assert.match(estilos, /forced-colors: active/);
  assert.match(estilos, /\.modulo-personal\s+\.rpt-huella\s*\{[^}]*overflow-wrap:\s*anywhere;/);
  assert.doesNotMatch(estilos, /\.tarjeta-modulo-bloqueada/);
});

test("el cache busting de módulos avanza en cascada hasta el HTML", async () => {
  const versionShellF2 = "20260924-f2-shell-v1";
  const versionTemaBase = "20260924-f2-tema-base-v2";
  const versionSaltoMovil = "20260924-f2-salto-movil-v3";
  const versionCacheF2 = "20260924-f2-cache-v2";
  const versionCachePersonal = "20260924-f2-cache-v3";
  const versionPersonalInterno = "20260924-p1-personal-interno-v2";
  const versionPersonalEstados = "20260924-f2-personal-estados-v4";
  const versionCronosPermisos = "20260924-f2-cronos-permisos-v2";
  const versionDietasShell = "20260924-web-paradas-periodos-v1";
  const versionEntradaAyuda = "20260924-rescate-web-v4";
  const versionVistasC = "20260924-web-c-v1";
  const versionDietasRecuperacion = "20260924-dietas-recuperacion-v3";
  const versionDietasVista = "20260924-web-paradas-periodos-v1";
  const versionPublicada = "20260925-aspecto-v1";
  const versionCarga = "20260923-p4-estado-modulos-v1";
  const versionModuloBolsa = "20260924-rescate-web-v4";
  const versionSubsanacion = "20260924-web-subsanacion-v1";
  const versionClientePersonal = versionPersonalInterno;
  const versionCatalogo = versionCronosPermisos;
  const versionCronos = "20260924-web-paradas-periodos-v1";
  const versionDietasCSS = "20260924-dietas-ayuda-icono-v1";
  const versionRPT = "20260920-personal-rpt-publica-v3";
  const versionEstilos = "20260920-personal-rpt-publica-v3";
  const versionFlujos = "20260923-pweb16-v1";
  const [portal, html] = await Promise.all([
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  const coordinador = await readFile(
    new URL("portal-modulos-coordinador.js", import.meta.url),
    "utf8",
  );
  // Los recursos cambiados después de su versión publicada exigen una URL
  // posterior y única entre sus importadores; los demás conservan la exacta.
  exigirRenovado(portal, "./portal-modulos-coordinador.js", versionDietasShell);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-web-c-ayuda-v5/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-web-c-ayuda-v4/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-web-c-v3/u);
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-f2-dietas-consulta-v2/u);
  assert.doesNotMatch(portal, new RegExp(`portal-modulos-coordinador\\.js\\?v=${versionCronosPermisos}`));
  assert.doesNotMatch(portal, /portal-modulos-coordinador\.js\?v=20260924-f2-cronos-permisos-v1/u);
  assert.doesNotMatch(portal, new RegExp(`portal-modulos-coordinador\\.js\\?v=${versionPersonalEstados}`));
  assert.doesNotMatch(portal, new RegExp(`portal-modulos-coordinador\\.js\\?v=${versionPersonalInterno}`));
  // El límite de carga cambió (recorrido del 25/09): su URL avanza.
  exigirRenovado(coordinador, "./portal-modulos-carga.js", versionCarga);
  exigirRenovado(portal, "./portal-bolsas-api.js", versionModuloBolsa);
  exigirRenovado([portal, coordinador], "./portal-i18n.js", [versionEntradaAyuda, versionCronosPermisos]);
  exigirRenovado(portal, "./portal-eventos.js", versionEntradaAyuda);
  exigirRenovado([portal, coordinador], "./portal-inicio.js", versionCronosPermisos);
  exigirRenovado(portal, "./portal-borradores-ui.js", versionCronosPermisos);
  exigirRenovado(coordinador, "./portal-catalogo-modulos.js", versionCatalogo);
  // El cliente del catálogo pasó a presentar el certificado mTLS: URL renovada.
  exigirRenovado(coordinador, "./modulos/personal/cliente-http-categorias.js", versionClientePersonal);
  // RPT y estructura públicas solo las pide el cargador opcional de catálogos
  // públicos, nunca el cargador principal de Personal.
  const cargadorPersonal = coordinador.split("personal: async () => {")[1].split("personal_catalogos_publicos: async () => {")[0];
  assert.doesNotMatch(cargadorPersonal, /modulos\/personal\/(?:cliente-http|vista)-rpt-publica\.js/);
  assert.doesNotMatch(cargadorPersonal, /estructura-organizativa-publica\.js/);
  // La vista interna conserva su versión renovada.
  for (const [vista, montajes, versionVista] of [["vista.js", 1, versionPersonalEstados]]) {
    exigirVersiones(coordinador, `./modulos/personal/${vista}`, posterior(versionVista), montajes);
  }
  assert.doesNotMatch(coordinador, new RegExp(`modulos/personal/vista\\.js\\?v=${versionCachePersonal}`));
  exigirRenovado(html, "/portal-empleado/portal.js", versionEntradaAyuda);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v5/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-ayuda-v4/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-web-c-v3/u);
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-dietas-consulta-v2/u);
  assert.doesNotMatch(html, new RegExp(`portal\\.js\\?v=${versionCronosPermisos}`));
  assert.doesNotMatch(html, /portal\.js\?v=20260924-f2-cronos-permisos-v1/u);
  assert.doesNotMatch(html, new RegExp(`portal\\.js\\?v=${versionPersonalEstados}`));
  assert.doesNotMatch(html, new RegExp(`portal\\.js\\?v=${versionPersonalInterno}`));
  exigirRenovado(html, "/portal-empleado/portal-modulos.css", versionEstilos);
  exigirRenovado(html, "/portal-empleado/portal-flujos.css", versionFlujos);
  exigirRenovado(html, "/portal-empleado/portal.css", versionSaltoMovil);
  assert.doesNotMatch(html, new RegExp(`portal\\.css\\?v=${versionTemaBase}`));
  assert.doesNotMatch(html, /portal\.css\?v=20260924-f2-salto-movil-v1/u);
  assert.doesNotMatch(html, /portal\.css\?v=20260924-f2-salto-movil-v2/u);
  assert.doesNotMatch(html, new RegExp(`portal\\.css\\?v=${versionShellF2}`));
  exigirRenovado(html, "/portal-empleado/modulos/contratacion-temporal/expedientes-operativo.css", versionShellF2);
  exigirRenovado(coordinador, "./modulos/cronos/vista-saldo-conectado.js", "20260925-tanda-v1");
  exigirRenovado(coordinador, "./modulos/cronos/vista-permisos-propios.js", "20260925-tanda-v1");
  // Dietas solo tiene montaje interno: un cargador, renovado respecto a lo publicado.
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-itinerario\.js/u);
  const versionDietasMontaje = exigirVersiones(coordinador, "./modulos/dietas/vista-recorridos.js", posterior(versionDietasVista), 1);
  assert.notEqual(versionDietasMontaje, versionPublicada);
  // El mapa comparte los textos de la vista y se renueva con ella; los clientes
  // HTTP no cambian y conservan su URL de montaje, también posterior a lo publicado.
  exigirVersiones(coordinador, "./modulos/dietas/mapa-ruta.js", versionDietasMontaje, 1);
  for (const cliente of ["cliente-borradores-http", "cliente-asignacion-http", "calculador-rutas-http"])
    assert.notEqual(exigirVersiones(coordinador, `./modulos/dietas/${cliente}.js`, posterior(versionDietasVista), 1), versionPublicada);
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-recorridos\.js\?v=20260924-dietas-ayuda-sin-guia-v1/u);
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-recorridos\.js\?v=20260924-dietas-ayuda-icono-v1/u);
  assert.doesNotMatch(coordinador, new RegExp(`modulos/dietas/vista-recorridos\\.js\\?v=${versionDietasRecuperacion}`));
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-recorridos\.js\?v=20260924-f2-dietas-consulta-v2/u);
  assert.doesNotMatch(coordinador, new RegExp(`modulos/dietas/vista-recorridos\\.js\\?v=${versionShellF2}`));
  exigirRenovado(html, "/portal-empleado/modulos/cronos/cronos.css", versionCronos);
  exigirRenovado(html, "/portal-empleado/modulos/dietas/dietas.css", [versionDietasCSS, versionPublicada]);
  exigirRenovado(html, "/portal-empleado/modulos/dietas/borradores-propios.css", versionPublicada);
});
