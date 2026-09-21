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
  CLAVES_MODULOS_VEC_REGISTRADOS,
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

test("el catálogo interno conserva el certificado solo en mismo origen, sin redirecciones ni caché", async () => {
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
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.mode, "same-origin");
    assert.equal(opciones.redirect, "error");
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

test("el catálogo admite el manifiesto real de usuarios con menú nil sin atribuir acciones", () => {
  const usuarios = {
    id: "vec.module.usuarios",
    name_key: "ui.vec.module.usuarios.name",
    description_key: "ui.vec.module.usuarios.description",
    version: "v0.1.0",
    group: "usuarios_vec",
    base_path: "/modules/usuarios",
    permissions: [
      { key: "vec.contacto_usuario.alta", label_key: "ui.permission.usuarios.contacto_alta" },
      { key: "vec.contacto_usuario.actualizar", label_key: "ui.permission.usuarios.contacto_actualizar" },
      { key: "vec.contacto_usuario.consultar", label_key: "ui.permission.usuarios.contacto_consultar" },
    ],
    menu: null,
  };
  const traducciones = {
    ...TRADUCCIONES_CONTRATACION_TEMPORAL,
    "ui.vec.module.usuarios.name": "Usuarios",
    "ui.vec.module.usuarios.description": "Contacto VEC",
  };
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [usuarios, manifiestoContratacionTemporal()],
    traducciones,
  );
  assert.deepEqual(catalogo.map(({ clave }) => clave), ["usuarios", "contratacion_temporal"]);
  assert.deepEqual(Object.keys(catalogo[0]).sort(), [
    "clave", "grupo", "rutaBase", "sigla", "texto", "titulo", "version",
  ]);
  for (const coleccionesInvalidas of [
    { permissions: null, menu: null },
    { permissions: [], menu: null },
    { permissions: usuarios.permissions, menu: {} },
  ]) assert.throws(() => crearCatalogoModulosDesdeManifiestos(
    [{ ...usuarios, ...coleccionesInvalidas }],
    traducciones,
  ), /colecciones del manifiesto no válidas/);
});

test("los siete módulos registrados conservan estado fiel sin inventar vistas", async () => {
  assert.deepEqual(CLAVES_MODULOS_VEC_REGISTRADOS, [
    "personal", "cronos", "dietas", "bolsa", "contratacion_temporal",
    "administracion", "usuarios",
  ]);
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => Object.freeze([
      { clave: "personal" }, { clave: "cronos" }, { clave: "dietas" },
      { clave: "bolsa" }, { clave: "contratacion_temporal" },
      { clave: "administracion" }, { clave: "usuarios" },
    ]),
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("sin dependencia CT"); },
    },
  });
  await coordinador.cargarInterno();
  for (const clave of ["administracion", "usuarios"]) {
    assert.deepEqual(coordinador.resolverAcceso(clave), {
      disponible: false, vista: "", estado: "no_disponible",
    });
    assert.equal(coordinador.vistaGestionada(clave), false);
  }
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


test("Personal falla cerrado en presentación sin el cliente HTTP de la concesión", async () => {
  const internoRecibido = [];
  const cargadoresPresentacion = {
    base: async () => {
      const [identidad, catalogo] = await Promise.all([
        import("./identidad/presentacion.js"),
        import("./portal-catalogo-presentacion.js"),
      ]);
      return Object.freeze({ identidad, catalogo });
    },
    personal: async () => Object.freeze({
      vista: Object.freeze({
        montarModuloPersonal: async () => Object.freeze({ desmontar() {} }),
      }),
    }),
  };
  const coordinadorPresentacion = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargadoresPresentacion,
  });
  await coordinadorPresentacion.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  assert.equal(coordinadorPresentacion.vistaDisponible("personal"), false);
  assert.equal(await coordinadorPresentacion.montarVista("personal", raizFalsa()), false);

  const coordinadorInterno = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "personal" }]),
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("no debe cargarse"); },
      personal: async () => Object.freeze({
        contrato: Object.freeze({ CAPACIDAD_CONSULTAR_PUESTO: "personal.puesto.read" }),
        cliente: Object.freeze({ crearClienteHTTPCategoriasPersonal: () => Object.freeze({ listarCategorias() {} }) }),
        vista: Object.freeze({
          montarModuloPersonal: async (dependencias) => {
            internoRecibido.push(dependencias);
            return Object.freeze({ desmontar() {} });
          },
        }),
      }),
    },
  });
  await coordinadorInterno.cargarInterno();
  assert.equal(await coordinadorInterno.montarVista("personal", raizFalsa()), true);
  assert.equal(internoRecibido.length, 1);
  assert.equal(Object.hasOwn(internoRecibido[0], "cliente"), true);
  assert.equal(Object.hasOwn(internoRecibido[0], "anunciar"), true);
  assert.equal(Object.hasOwn(internoRecibido[0], "presentacion"), false);
  assert.equal(Object.hasOwn(internoRecibido[0], "token"), false);
});

test("Personal de presentación conserva categorías E24, RPT y estructura pública", async () => {
  const llamadas = [];
  const rpt = {
    data: {
      rpt: {
        items: [{
          clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 57, dotacion: 158,
        }],
        total: 1, limit: 25, offset: 0, esquema: "vec.catalogo.rpt.v1",
        fuente: { documento: "RPT publicada", importacion: "rpt-publica-v1", generado_en: "2026-09-17", aviso: "Datos públicos sin ocupantes.", huella_sha256: "a".repeat(64) },
      },
    },
  };
  const categorias = { data: { categories: { items: [{ catalog: "categoria_profesional", clave: "administrativo", slug: "administrativo", etiqueta: "Administrativo", name: "Administrativo", orden: 1, area: "administracion_general", area_etiqueta: "Administración general", source: "catalogo_gobernado_vec", module_key: "vec.module.personal", state: "Demostración pendiente de validación RRHH", usage: "Bolsa, RPT, certificados y demás módulos autorizados." }], total: 1, limit: 25, offset: 0, catalogo: { catalogo_id: "categorias-profesionales", catalogo_version: 1, catalogo_huella_sha256: "a".repeat(64) }, fuente: { revision: "demo-v1", actualizada_en: "2026-09-20T08:00:00Z", demostracion: true, aviso: "DEMOSTRACIÓN pendiente de validación RRHH." } } } };
  const unidades = []; for (let i = 0; i < 14; i += 1) unidades.push({ clave: `delegacion-${i}`, etiqueta: "Delegación", tipo: "delegacion" }); for (let i = 0; i < 41; i += 1) unidades.push({ clave: `centro-${i}`, etiqueta: "Centro", tipo: "centro", adscripcion_clave: "delegacion-0" }); for (let i = 0; i < 11; i += 1) unidades.push({ clave: `puesto-${i}`, etiqueta: "Jefatura", tipo: "puesto_responsabilidad", adscripcion_clave: "centro-0" });
  const estructura = { data: { estructura_organizativa: { esquema: "vec.personal.estructura-organizativa-publica.v1", catalogo_id: "estructura-organizativa-dipgra", catalogo_version: 1, catalogo_revision: 1, fuente_ref: "https://example.test/rpt", fuente: { revision: "demo-v1", actualizada_en: "2026-09-06T00:00:00Z", demostracion: true, aviso: "Demo sin vigencia", huella_sha256: "0e52d878526d6a5e7ee4ab6f525ef92a70144aef665f0b031fca6051564e054c" }, unidades } } };
  const [identidad, catalogo, contrato, clienteCategorias, vistaCategorias, clienteRPT, vistaRPT, clienteEstructura, vistaEstructura] = await Promise.all([
    import("./identidad/presentacion.js"), import("./portal-catalogo-presentacion.js"),
    import("./modulos/personal/contrato.js"), import("./modulos/personal/cliente-http-categorias.js"),
    import("./modulos/personal/vista.js"), import("./modulos/personal/cliente-http-rpt-publica.js"),
    import("./modulos/personal/vista-rpt-publica.js"), import("./modulos/personal/cliente-http-estructura-organizativa-publica.js"), import("./modulos/personal/vista-estructura-organizativa-publica.js"),
  ]);
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: {
      fetch: async (ruta, opciones) => {
        llamadas.push({ ruta, opciones });
        return respuestaJSON(ruta.startsWith("/api/vec/personal/categories") ? categorias : ruta.startsWith("/api/vec/personal/estructura") ? estructura : rpt);
      },
    },
    cargadoresPresentacion: {
      base: async () => Object.freeze({ identidad, catalogo }),
      personal: async () => Object.freeze({ contrato, clienteCategorias, vistaCategorias, clienteRPT, vistaRPT, clienteEstructura, vistaEstructura }),
    },
  });
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  assert.equal(coordinador.vistaDisponible("personal"), true);
  const raiz = raizDietasFalsa();
  assert.equal(await coordinador.montarVista("personal", raiz), true);
  assert.ok(raiz.querySelector("[data-personal-categorias]"));
  assert.ok(raiz.querySelector("[data-personal-rpt-publica]"));
  assert.ok(raiz.querySelector("[data-personal-estructura-organizativa-publica]"));
  assert.equal(llamadas.length, 3);
  assert.deepEqual(new Set(llamadas.map(({ ruta }) => ruta)), new Set(["/api/vec/personal/categories?q=&area=&limit=25&offset=0", "/api/vec/personal/rpt-publica?q=&limit=25&offset=0", "/api/vec/personal/estructura-organizativa-publica"]));
  llamadas.forEach(({ opciones }) => { assert.equal(opciones.method, "GET"); assert.equal(opciones.credentials, "same-origin"); assert.equal(opciones.redirect, "error"); assert.equal(Object.hasOwn(opciones, "headers"), false); });
});

test("el cargador interno predeterminado de Personal compone contrato, cliente y vista reales", async () => {
  const categorias = { data: { categories: { items: null, total: 0, limit: 25, offset: 0, catalogo: { catalogo_id: "categorias-profesionales", catalogo_version: 1, catalogo_huella_sha256: "a".repeat(64) }, fuente: { revision: "demo-v1", actualizada_en: "2026-09-20T08:00:00Z", demostracion: true, aviso: "DEMOSTRACIÓN pendiente de validación RRHH." } } } };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: { fetch: async () => respuestaJSON(categorias) },
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "personal" }]),
    cargadoresInternos: { contratacion_temporal: async () => { throw new Error("no debe cargar CT"); } },
  });
  await coordinador.cargarInterno(); assert.equal(coordinador.resolverAcceso("personal").etiqueta, "Catálogo profesional de Personal");
  const raiz = raizDietasFalsa(); assert.equal(await coordinador.montarVista("personal", raiz), true); assert.ok(raiz.querySelector("[data-personal-categorias]"));
});

test("CT interno se activa solo después de una consulta autorizada", async () => {
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL,
  );
  let consultas = 0;
  let signalConsulta;
  let montajes = 0;
  const fuente = Object.freeze({
    capacidades: Object.freeze([
      "contratacion_temporal.cuadro.consultar",
      "contratacion_temporal.expediente.consultar",
    ]),
    async listar({ signal } = {}) {
      consultas += 1;
      signalConsulta = signal;
      return { expedientes: [] };
    },
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
  assert.ok(signalConsulta instanceof AbortSignal);
  assert.equal(signalConsulta.aborted, false);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
  assert.equal(await coordinador.montarVista("contratacion-temporal", raizFalsa()), true);
  assert.equal(montajes, 1);
});

test("Intervención abre CT con acceso directo a fiscalización, sin funciones de RRHH", async () => {
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL,
  );
  const cliente = Object.freeze({
    async obtenerCatalogosAlta() { throw new Error("alta reservada a RRHH"); },
    async obtenerConfiguracionAnalisis() { throw new Error("análisis reservado a RRHH"); },
    async registrarResultadoFiscalizacion() {},
  });
  const fuente = Object.freeze({
    capacidades: Object.freeze([
      "contratacion_temporal.cuadro.consultar",
      "contratacion_temporal.expediente.consultar",
    ]),
    async listar() { return { expedientes: [] }; },
    async obtener() { return {}; },
    async ejecutar() { throw new Error("solo lectura"); },
  });
  let montaje;
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => catalogo,
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => cliente },
        adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => fuente },
        contrato: { validarCatalogosAlta: (valor) => valor },
        presentador: {
          crearPresentadorExpedientesContratacionTemporal: () => ({}),
        },
        vista: {
          montarModuloContratacionTemporal: async (dependencias) => {
            montaje = dependencias;
            return { desmontar() {} };
          },
          montarModuloFiscalizacionContratacionTemporal: async (dependencias) => {
            montaje = dependencias;
            return { desmontar() {} };
          },
        },
      }),
    },
  });

  await coordinador.cargarInterno();
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
  assert.equal(await coordinador.montarVista("contratacion-temporal", raizFalsa()), true);
  assert.strictEqual(montaje.cliente, cliente);
});

test("CT recupera incorporación con continuidad si falla análisis y el alta sigue disponible", async () => {
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL,
  );
  const cliente = Object.freeze({
    async obtenerCatalogosAlta() { return { centros: [], categorias: [], documentos: [] }; },
    async registrarSolicitud() {},
    async registrarResultadoFiscalizacion() {},
    async obtenerConfiguracionAnalisis() { throw new Error("análisis no disponible"); },
    async prepararIncorporacionEjercicio() {},
    async confirmarIncorporacionEjercicio() {},
  });
  const fuente = Object.freeze({
    capacidades: Object.freeze(["contratacion_temporal.cuadro.consultar"]),
    async listar() { return { expedientes: [] }; },
    async obtener() { return {}; },
    async ejecutar() { throw new Error("solo lectura"); },
  });
  let montaje;
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => catalogo,
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => cliente },
        adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => fuente },
        contrato: { validarCatalogosAlta: (valor) => valor },
        presentador: { crearPresentadorExpedientesContratacionTemporal: () => ({}) },
        vista: { montarModuloFiscalizacionContratacionTemporal: async () => {
          assert.fail("RRHH conserva su vista cuando el alta está disponible");
        }, montarModuloContratacionTemporal: async (dependencias) => {
          montaje = dependencias;
          return { desmontar() {} };
        } },
      }),
    },
  });

  await coordinador.cargarInterno();
  assert.equal(await coordinador.montarVista("contratacion-temporal", raizFalsa()), true);
  assert.ok(montaje.alta);
  assert.equal(montaje.analisis, null);
  assert.strictEqual(montaje.continuidad.cliente, cliente);
  assert.equal(typeof montaje.continuidad.cliente.prepararIncorporacionEjercicio, "function");
  assert.equal(typeof montaje.continuidad.cliente.confirmarIncorporacionEjercicio, "function");
});

test("CT interno mantiene el alta real cuando el cuadro sigue en 503", async () => {
  const catalogo = crearCatalogoModulosDesdeManifiestos(
    [manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL,
  );
  const catalogosAlta = Object.freeze({
    esquema: "vec.contratacion_temporal.catalogos_alta.v1",
    centros: Object.freeze([]),
    categorias: Object.freeze([]),
    motivos: Object.freeze([]),
    documentos: Object.freeze([]),
  });
  const registrarSolicitud = async () => ({ recibo_ref: "recibo:opaco:001" });
  const configuracionAnalisis = Object.freeze({
    esquema: "vec.contratacion_temporal.configuracion_analisis.v1",
    artefacto_ref: "artefacto:analisis:vigente:001",
    modalidades: Object.freeze([
      { clave: "sustitucion", etiqueta: "Sustitución" },
      { clave: "vacante", etiqueta: "Vacante" },
      { clave: "acumulacion_tareas", etiqueta: "Acumulación de tareas" },
      { clave: "programa", etiqueta: "Programa temporal" },
      { clave: "relevo", etiqueta: "Contrato de relevo" },
    ]),
    categorias: Object.freeze([]),
    causas: Object.freeze([]),
    entradas_rc: Object.freeze([]),
    motivos_rectificacion: Object.freeze([]),
  });
  let consultasCuadro = 0;
  let consultasCatalogo = 0;
  let consultasAnalisis = 0;
  const orden = [];
  let argumentosPresentador;
  let altaMontada;
  let analisisMontado;
  const fuente = Object.freeze({
    capacidades: Object.freeze([]),
    async listar() {
      consultasCuadro += 1;
      orden.push("cuadro");
      const error = new Error("cuadro pendiente");
      error.estado = 503;
      throw error;
    },
    async obtener() { throw new Error("detalle no compuesto"); },
    async ejecutar() { throw new Error("actuación no compuesta"); },
  });
  const cliente = Object.freeze({
    async obtenerCatalogosAlta() {
      consultasCatalogo += 1;
      orden.push("alta");
      return catalogosAlta;
    },
    async obtenerConfiguracionAnalisis() {
      consultasAnalisis += 1;
      orden.push("analisis");
      return configuracionAnalisis;
    },
    registrarSolicitud,
    async registrarAnalisis() {},
  });
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => catalogo,
    cargadoresInternos: {
      contratacion_temporal: async () => ({
        cliente: { crearClienteHTTPContratacionTemporal: () => cliente },
        adaptador: {
          crearAdaptadorHTTPExpedientesContratacionTemporal: () => fuente,
        },
        contrato: {
          validarCatalogosAlta: (valor) => valor,
          CAPACIDAD_CREAR_SOLICITUD: "contratacion_temporal.solicitud.crear",
        },
        presentador: {
          crearPresentadorExpedientesContratacionTemporal: (argumentos) => {
            argumentosPresentador = argumentos;
            return {};
          },
        },
        vista: {
          montarModuloContratacionTemporal: async ({ alta, analisis }) => {
            altaMontada = alta;
            analisisMontado = analisis;
            return { desmontar() {} };
          },
        },
      }),
    },
  });

  await coordinador.cargarInterno();
  await coordinador.cargarInterno();
  assert.equal(consultasCuadro, 2);
  assert.equal(consultasCatalogo, 2);
  assert.equal(consultasAnalisis, 2);
  assert.deepEqual(orden, [
    "cuadro", "alta", "analisis", "cuadro", "alta", "analisis",
  ]);
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);
  assert.equal(await coordinador.montarVista("contratacion-temporal", raizFalsa()), true);
  assert.equal(argumentosPresentador.altaDisponible, true);
  assert.deepEqual(argumentosPresentador.capacidades, []);
  assert.strictEqual(altaMontada.catalogos, catalogosAlta);
  assert.strictEqual(altaMontada.ejecutor, registrarSolicitud);
  assert.equal(Object.hasOwn(altaMontada, "desarrolloNoAutoritativo"), false);
  assert.strictEqual(analisisMontado.cliente, cliente);
  assert.strictEqual(
    analisisMontado.catalogos.modalidades,
    configuracionAnalisis.modalidades,
  );
  assert.equal(analisisMontado.catalogos.modalidades.length, 5);
  assert.deepEqual(analisisMontado.contexto, {
    operacion: "registrar",
    artefacto_ref: "artefacto:analisis:vigente:001",
  });
  assert.equal(analisisMontado.analisisInicial, null);
  assert.equal(Object.hasOwn(analisisMontado, "desarrolloNoAutoritativo"), false);
});

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

test("el funcionario comparte una sola identidad y compone Cronos, Dietas y Personal", async () => {
  const coordinador = crearCoordinador();
  const contextoBolsa = await coordinador.cargarPresentacion(
    obtenerDatosPresentacion("funcionario").sesion,
  );
  assert.equal(contextoBolsa, null);
  assert.equal(coordinador.obtenerContextoBolsa(), null);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, false);
  assert.equal(coordinador.resolverAcceso("cronos", true).disponible, true);
  assert.equal(coordinador.resolverAcceso("dietas", true).disponible, true);
  assert.equal(coordinador.resolverAcceso("personal", true).disponible, true);
  const navegacion = coordinador.renderizarNavegacion(true, "portal", (vista) => ["cronos", "dietas", "personal"].includes(vista));
  assert.equal((navegacion.match(/modulo-habilitado/g) || []).length, 3);
  assert.match(navegacion, /data-modulo-portal="bolsa"[^>]*disabled/u);
});

test("técnico y administrador mantienen Personal denegado por falta de contexto", async () => {
  for (const perfil of ["tecnico", "administrador"]) {
    const coordinador = crearCoordinador();
    await coordinador.cargarPresentacion(obtenerDatosPresentacion(perfil).sesion);
    assert.equal(coordinador.resolverAcceso("personal", true).disponible, false, perfil);
  }
});

test("las rutas estables no mezclan el submenú de Bolsa con los módulos personales", () => {
  assert.equal(rutaDeVistaPortal("portal"), "#portal");
  assert.equal(rutaDeVistaPortal("resumen"), "#bolsa/resumen");
  assert.equal(rutaDeVistaPortal("cronos"), "#cronos");
  assert.equal(rutaDeVistaPortal("dietas"), "#dietas");
  assert.equal(moduloDeVistaPortal("convocatorias"), "bolsa");
  assert.equal(moduloDeVistaPortal("cronos"), "cronos");
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

test("Cronos y Dietas montan contenido administrativo y nunca dejan el área en blanco", async () => {
  const coordinador = crearCoordinador();
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  const raiz = raizDietasFalsa();
  assert.equal(await coordinador.montarVista("cronos", raiz), true);
  const cronos = raiz.querySelector("[data-cronos-recorridos]");
  assert.ok(cronos);
  assert.match(cronos.innerHTML, /class="cronos-area/);
  assert.match(cronos.innerHTML, /Movimientos/);
  assert.doesNotMatch(cronos.innerHTML, /Descargar recibo/);
  const raizDietas = raizDietasFalsa();
  assert.equal(await coordinador.montarVista("dietas", raizDietas), true);
  assert.ok(raizDietas.querySelector("[data-dietas-itinerario]"));
  assert.ok(raizDietas.querySelector("[data-itinerario-catalogo]"));
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
  const raiz = raizDietasFalsa();
  assert.equal(await coordinador.montarVista("dietas", raiz), true);

  const contenedorDietas = raiz.querySelector("[data-dietas-itinerario]");
  await contenedorDietas.listeners.click({
    target: contenedorDietas.querySelector("[data-itinerario-calcular]"),
  });

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
  assert.ok(raiz.querySelector("[data-dietas-mapa-ref]"));
  assert.ok(anuncios.some(({ mensaje }) => /calculada por el puerto interno/i.test(mensaje)));
  coordinador.desmontarVistaActual();
});

test("una navegación aborta el catálogo Dietas pendiente sin publicar su montaje obsoleto", async () => {
  let resolverCatalogo; let senalCatalogo;
  const catalogoPendiente = new Promise((resolver) => { resolverCatalogo = resolver; });
  const [identidad, catalogo, cronosContrato, cronosPresentador, cronosDatos, cronosAdaptador, documentos,
    dietasContrato, dietaVista, personalVista, catalogoDietas] = await Promise.all([
    import("./identidad/presentacion.js"), import("./portal-catalogo-presentacion.js"),
    import("./modulos/cronos/contrato.js"), import("./modulos/cronos/presentador.js"),
    import("./modulos/cronos/datos-presentacion.js"), import("./modulos/cronos/adaptador-presentacion.js"),
    import("./documentos/descarga-recibos-presentacion.js"), import("./modulos/dietas/contrato.js"),
    import("./modulos/dietas/vista-itinerario.js"), import("./modulos/personal/vista.js"),
    import("./modulos/dietas/catalogo-rutas-provincial.js"),
  ]);
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: { location: { origin: "http://127.0.0.2:8081" }, fetch: async () => { throw new Error("no procede"); } },
    cargadoresPresentacion: {
      base: async () => Object.freeze({ identidad, catalogo }),
      cronos: async () => Object.freeze({
        contrato: cronosContrato, presentador: cronosPresentador, datos: cronosDatos,
        adaptador: cronosAdaptador, documentos,
      }),
      dietas: async () => Object.freeze({
        contrato: dietasContrato,
        vista: dietaVista,
        mapa: { crearVisorRutaDietas: () => ({ montar() { throw new Error("no debe montar mapa"); } }) },
        calculador: { crearCalculadorRutasDietasPresentacionOSRM: () => ({
          obtenerCatalogo({ signal }) { senalCatalogo = signal; return catalogoPendiente; },
          calcular: async () => null,
        }) },
      }),
      personal: async () => Object.freeze({ vista: personalVista }),
    },
  });
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("funcionario").sesion);
  const raiz = raizDietasFalsa();
  const montajeAnterior = coordinador.montarVista("dietas", raiz);
  assert.ok(raiz.querySelector("[data-dietas-itinerario]"));
  coordinador.desmontarVistaActual();
  const ajeno = raiz.ownerDocument.createElement("section"); ajeno.dataset.ajeno = ""; raiz.append(ajeno);
  resolverCatalogo(catalogoDietas.obtenerCatalogoRutasProvincial());
  assert.equal(await montajeAnterior, false);
  assert.equal(senalCatalogo.aborted, true);
  assert.equal(raiz.querySelector("[data-ajeno]"), ajeno);
  assert.equal(raiz.querySelector("[data-dietas-itinerario]"), null);
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
  const [fuente, estilos, empleado] = await Promise.all([
    readFile(new URL("portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("portal-modulos.css", import.meta.url), "utf8"),
    readFile(new URL("portal-composicion-empleado.js", import.meta.url), "utf8"),
  ]);
  assert.doesNotMatch(fuente, /document\.cookie|localStorage|sessionStorage/);
  assert.doesNotMatch(empleado, /document\.cookie|localStorage|sessionStorage/);
  assert.match(fuente, /Promise\.allSettled/);
  assert.match(fuente, /LIMITE_CARGA_MODULAR_MS/);
  assert.match(fuente, /composicion = null/);
  assert.match(fuente, /secuenciaCarga/);
  assert.match(empleado, /function componerCronosVisible/);
  assert.match(fuente, /function capacidadesDietas/);
  assert.doesNotMatch(fuente, /^import .*\/modulos\//mu);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/cronos\/datos-presentacion\.js/);
  assert.match(fuente, /import\("\.\/modulos\/cronos\/vista-recorridos\.js/);
  assert.match(fuente, /import\("\.\/modulos\/dietas\/vista-itinerario\.js/);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/dietas\/adaptador-presentacion\.js/);
  assert.match(fuente, /calculador-rutas-presentacion-osrm\.js/);
  assert.doesNotMatch(fuente, /import\("\.\/modulos\/dietas\/calculador-rutas-presentacion\.js"\)/);
  assert.doesNotMatch(fuente, /versionGrafo|granada-buffer-osrm-v/u);
  assert.match(empleado, /recursos\.mapa\.crearVisorRutaDietas\(\{ entorno, permitirTeselas: true \}\)/);
  assert.match(estilos, /data-modulo-catalogo="bolsa"/);
  assert.match(estilos, /data-modulo-catalogo="cronos"/);
  assert.match(estilos, /data-modulo-catalogo="dietas"/);
  assert.match(estilos, /data-modulo-portal="cronos"/);
  assert.match(estilos, /forced-colors: active/);
  assert.match(estilos, /\.modulo-personal\s+\.rpt-huella\s*\{[^}]*overflow-wrap:\s*anywhere;/);
  assert.doesNotMatch(estilos, /\.tarjeta-modulo-bloqueada/);
});

test("el cache busting de módulos avanza en cascada hasta el HTML", async () => {
  const versionCoordinador = "20260921-avisos-r5-v1";
  const versionPortal = "20260921-avisos-r5-v1";
  const versionBolsaAPI = "20260921-montaje-modulos-b1";
  const versionI18n = "20260920-personal-catalogo-v1";
  const versionCatalogo = "20260906-acceso-certificado-v1";
  const versionTema = "20260920-referencia-rrhh-v1";
  const versionTemaCT = "20260918-botones-v1";
  const versionPulido = "20260920-recorridos-visibles-v1";
  const versionRPT = "20260920-personal-rpt-publica-v3";
  const versionEstilos = "20260920-personal-rpt-publica-v3";
  const versionFlujos = "20260921-avisos-movil-v1";
  const [portal, html] = await Promise.all([
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  const coordinador = await readFile(
    new URL("portal-modulos-coordinador.js", import.meta.url),
    "utf8",
  );
  assert.match(portal, new RegExp(`portal-modulos-coordinador\\.js\\?v=${versionCoordinador}`));
  assert.match(portal, new RegExp(`portal-bolsas-api\\.js\\?v=${versionBolsaAPI}`));
  assert.match(portal, new RegExp(`portal-i18n\\.js\\?v=${versionI18n}`));
  assert.match(coordinador, new RegExp(`portal-catalogo-modulos\\.js\\?v=${versionCatalogo}`));
  assert.match(coordinador, new RegExp(`portal-i18n\\.js\\?v=${versionI18n}`));
  assert.match(coordinador, new RegExp(`modulos/personal/cliente-http-categorias\\.js\\?v=${versionI18n}`));
  assert.match(coordinador, new RegExp(`modulos/personal/cliente-http-rpt-publica\\.js\\?v=${versionRPT}`));
  assert.match(coordinador, new RegExp(`modulos/personal/vista-rpt-publica\\.js\\?v=${versionRPT}`));
  assert.match(html, new RegExp(`portal\\.js\\?v=${versionPortal}`));
  assert.match(html, new RegExp(`portal-modulos\\.css\\?v=${versionEstilos}`));
  assert.match(html, new RegExp(`portal-flujos\\.css\\?v=${versionFlujos}`));
  assert.match(html, new RegExp(`portal\\.css\\?v=${versionTema}`));
  assert.match(html, new RegExp(`expedientes-operativo\\.css\\?v=${versionTemaCT}`));
  assert.match(html, new RegExp(`modulos/cronos/cronos\\.css\\?v=${versionPulido}`));
  assert.match(html, new RegExp(`modulos/dietas/dietas\\.css\\?v=${versionPulido}`));
});
