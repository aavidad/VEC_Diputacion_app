import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
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


test("el cargador interno predeterminado de Personal monta la ficha sin cargar catálogos al entrar", async () => {
  const categorias = { data: { categories: { items: null, total: 0, limit: 25, offset: 0, catalogo: { catalogo_id: "categorias-profesionales", catalogo_version: 1, catalogo_huella_sha256: "a".repeat(64) }, fuente: { revision: "demo-v1", actualizada_en: "2026-09-20T08:00:00Z", demostracion: true, aviso: "DEMOSTRACIÓN pendiente de validación RRHH." } } } };
  const llamadas = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: { fetch: async (ruta) => { llamadas.push(ruta); return respuestaJSON(categorias); } },
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "personal" }]),
    cargadoresInternos: { contratacion_temporal: async () => { throw new Error("no debe cargar CT"); } },
  });
  await coordinador.cargarInterno(); assert.equal(coordinador.resolverAcceso("personal").etiqueta, "Catálogo profesional de Personal");
  const raiz = raizDietasFalsa(); assert.equal(await coordinador.montarVista("personal", raiz), true);
  assert.ok(raiz.querySelector("[data-personal-ficha-integral]"));
  assert.equal(raiz.querySelector("[data-personal-categorias]"), null);
  assert.equal(llamadas.length, 0);
  assert.equal(raiz.querySelectorAll('[data-personal-ficha-estado="no_configurado"]').length, 6);
  raiz.querySelector('[data-personal-ficha-tab="relaciones"]').listeners.click();
  assert.equal(llamadas.length, 0);
  raiz.querySelector('[data-personal-ficha-tab="catalogos"]').listeners.click();
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(llamadas, ["/api/vec/personal/categories?q=&area=&limit=25&offset=0"]);
  assert.ok(raiz.querySelector("[data-personal-categorias]"));
  coordinador.desmontarVistaActual();
  assert.equal(raiz.querySelector("[data-personal-ficha-integral]"), null);
});

test("RRHH abre el Registro de Personal desde Personal con subnavegación; sin perfil RRHH no se ofrece", async () => {
  const catalogo = [...crearCatalogoModulosDesdeManifiestos([manifiestoContratacionTemporal()], TRADUCCIONES_CONTRATACION_TEMPORAL), Object.freeze({ clave: "personal" })];
  const rutas = [];
  const fuenteCT = Object.freeze({
    capacidades: Object.freeze(["contratacion_temporal.cuadro.consultar", "contratacion_temporal.expediente.consultar"]),
    async listar() { return { expedientes: [] }; }, async obtener() { throw new Error("sin expedientes"); }, async ejecutar() { throw new Error("solo lectura"); },
  });
  const cargadorCT = async () => ({
    cliente: { crearClienteHTTPContratacionTemporal: () => ({}) },
    adaptador: { crearAdaptadorHTTPExpedientesContratacionTemporal: () => fuenteCT },
    presentador: { crearPresentadorExpedientesContratacionTemporal: () => ({ obtenerEstado: () => ({}), cargar: async () => ({}) }) },
    vista: { montarModuloContratacionTemporal: async () => ({ desmontar() {} }) },
  });
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: { fetch: async (ruta) => { rutas.push(ruta); return new Response(null, { status: 403 }); } },
    cargarCatalogoInterno: async () => Object.freeze(catalogo),
    cargadoresInternos: { contratacion_temporal: cargadorCT },
  });
  await coordinador.cargarInterno();
  assert.equal(coordinador.esPerfilRRHH(), true);
  assert.equal(coordinador.vistaDisponible("personal-registro"), true);
  assert.equal(moduloDeVistaPortal("personal-registro"), "personal");
  assert.equal(rutaDeVistaPortal("personal-registro"), "#personal-registro");
  const raiz = raizDietasFalsa();
  assert.equal(await coordinador.montarVista("personal-registro", raiz), true);
  assert.ok(raiz.querySelector("[data-personal-subvistas]"));
  assert.ok(raiz.querySelector("[data-personal-registro-b2]"));
  await new Promise((resolve) => setImmediate(resolve));
  assert.ok(rutas.some((ruta) => ruta.startsWith("/api/vec/personal/empleados-organismo?")));
  coordinador.desmontarVistaActual();
  assert.equal(raiz.querySelector("[data-personal-registro-b2]"), null);
  assert.equal(raiz.querySelector("[data-personal-subvistas]"), null);
  assert.equal(await coordinador.montarVista("personal", raiz), true);
  assert.ok(raiz.querySelector("[data-personal-subvistas]"));
  assert.ok(raiz.querySelector("[data-personal-ficha-integral]"));
  coordinador.desmontarVistaActual();

  const sinRRHH = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: { fetch: async () => new Response(null, { status: 403 }) },
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "personal" }]),
    cargadoresInternos: { contratacion_temporal: async () => { throw new Error("no debe cargar CT"); } },
  });
  await sinRRHH.cargarInterno();
  assert.equal(sinRRHH.vistaDisponible("personal"), true);
  assert.equal(sinRRHH.vistaDisponible("personal-registro"), false);
  const raizEmpleado = raizDietasFalsa();
  assert.equal(await sinRRHH.montarVista("personal", raizEmpleado), true);
  assert.equal(raizEmpleado.querySelector("[data-personal-subvistas]"), null);
});

test("el paquete interno de Personal no solicita recursos RPT ni estructura pública", async () => {
  const [codigo, manifiesto] = await Promise.all([
    readFile(new URL("portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("../../interno.manifest", import.meta.url), "utf8"),
  ]);
  const interno = codigo.split("const CARGADORES_INTERNOS_PREDETERMINADOS =")[1]
    .split("export const VISTAS_MODULOS_PERSONALES")[0];
  const personalInterno = interno.split("personal: async () => {")[1].split("dietas: async () => {")[0];
  const recursosInternos = [...personalInterno.matchAll(/import\("\.\/modulos\/personal\/([^?"']+)/gu)]
    .map((match) => match[1]);
  assert.deepEqual(recursosInternos, ["contrato.js", "cliente-http-categorias.js",
    "vista.js", "vista-ficha-integral.js", "registro-b2.js", "registro-b2-cliente.js",
    "registro-b2-catalogos-cliente.js", "i18n.js"]);
  for (const recurso of recursosInternos) {
    assert.match(manifiesto, new RegExp(`static/portal-empleado/modulos/personal/${recurso.replaceAll(".", "\\.")}`, "u"));
  }
  for (const recurso of ["cliente-http-rpt-publica.js", "vista-rpt-publica.js",
    "cliente-http-estructura-organizativa-publica.js", "vista-estructura-organizativa-publica.js"]) {
    assert.doesNotMatch(interno, new RegExp(recurso.replaceAll(".", "\\."), "u"), recurso);
    assert.doesNotMatch(manifiesto, new RegExp(recurso.replaceAll(".", "\\."), "u"), recurso);
  }
});

async function recursosCronosInternos() {
  const [saldo, remoto, movimientos, movimientosPropios, permisosPropios, clienteSaldo, clienteRemoto, clienteSolicitudes, i18n] = await Promise.all([
    import("./modulos/cronos/vista-saldo-conectado.js"), import("./modulos/cronos/vista-remoto.js"),
    import("./modulos/cronos/vista-movimientos-conectado.js"), import("./modulos/cronos/vista-movimientos-propios.js"),
    import("./modulos/cronos/vista-permisos-propios.js"), import("./modulos/cronos/cliente-saldo-http.js"),
    import("./modulos/cronos/cliente-remoto-http.js"), import("./modulos/cronos/cliente-solicitudes-http.js"),
    import("./modulos/cronos/i18n.js"),
  ]);
  return { saldo, remoto, movimientos, movimientosPropios, permisosPropios, clienteSaldo, clienteRemoto, clienteSolicitudes, i18n };
}
const esperarVueltas = async () => { for (let i = 0; i < 10; i++) await new Promise((resolve) => setImmediate(resolve)); };

test("Cronos interno monta saldo, fichaje remoto, movimientos y calendario; con la API en 404 cada parte muestra su estado", async () => {
  const recursos = await recursosCronosInternos();
  const pedidas = [];
  const entorno = { fetch: async (ruta) => { pedidas.push(String(ruta).split("?")[0]);
    return new Response(JSON.stringify({ error: "no_encontrado" }), { status: 404, headers: { "content-type": "application/json" } }); } };
  const cargadoresInternos = {
    contratacion_temporal: async () => { throw new Error("no debe cargar CT"); },
    cronos: async () => recursos,
  };
  const coordinador = crearCoordinadorModulosPortal({ escaparHTML: String, entorno,
    cargarCatalogoInterno: async () => [{ clave: "cronos" }], cargadoresInternos });
  await coordinador.cargarInterno();
  assert.equal(coordinador.resolverAcceso("cronos").disponible, true);
  const raiz = raizDietasFalsa();
  assert.equal(await coordinador.montarVista("cronos", raiz), true);
  await esperarVueltas();
  const partes = raiz.querySelectorAll("[data-cronos-parte]");
  assert.deepEqual(partes.map((p) => p.dataset.cronosParte), ["saldo", "remoto", "movimientos", "calendario"]);
  const html = (nombre) => { const texto = []; const visitar = (n) => { texto.push(n.innerHTML ?? ""); n.children.forEach(visitar); };
    visitar(raiz.querySelector(`[data-cronos-parte="${nombre}"]`)); return texto.join(""); };
  assert.match(html("saldo"), /data-cronos-saldo-estado="no_disponible"/u);
  assert.match(html("movimientos"), /data-cronos-movimientos-estado="no_disponible"/u);
  assert.match(html("movimientos"), /data-cronos-accion="solicitar-correccion">/u, "olvido de marcaje habilitado");
  assert.match(html("calendario"), /cronos-movimientos-propios" [^>]*data-estado="no_disponible"/u);
  assert.match(html("remoto"), /El fichaje remoto no está disponible/u);
  for (const nombre of ["saldo", "remoto", "movimientos", "calendario"]) {
    // 404 = capacidad no disponible: estado neutro, sin alerta ni sobrelínea propia.
    assert.doesNotMatch(html(nombre), /role="alert"[^>]*>[^<]|No se pudo|No se pudieron|class="sobrelinea"/u, nombre);
  }
  const cabecera = raiz.children.find((nodo) => nodo.tagName === "header");
  assert.deepEqual(cabecera.children[0].children.map((n) => [n.tagName, n.className, n.textContent]),
    [["p", "sobrelinea", "Cronos"], ["h2", undefined, "Mi jornada"]], "un único encabezado de página");
  assert.doesNotMatch(html("remoto"), /data-cronos-remoto-movimiento="entrada"(?![^>]*disabled)/u, "sin fichaje sin disponibilidad");
  assert.ok(pedidas.includes("/api/interna/cronos/saldos/propio"));
  assert.ok(pedidas.includes("/api/interna/cronos/marcajes/remoto/disponibilidad"));
  assert.equal(new Set(pedidas).size >= 3, true);
  coordinador.desmontarVistaActual();
  assert.equal(raiz.querySelectorAll("[data-cronos-parte]").length, 0);
  assert.equal(raiz.children.length, 0);
  const sinCatalogo = crearCoordinadorModulosPortal({ escaparHTML: String, entorno,
    cargarCatalogoInterno: async () => [], cargadoresInternos });
  await sinCatalogo.cargarInterno();
  assert.equal(sinCatalogo.resolverAcceso("cronos").disponible, false);
});

test("Cronos interno: olvido de marcaje abre el formulario del calendario y faltar una vista cierra el módulo", async () => {
  const reales = await recursosCronosInternos();
  const llamadas = [];
  const parte = (nombre, extra = {}) => (opciones) => {
    llamadas.push([nombre, opciones]);
    const nodo = opciones.raiz.ownerDocument.createElement("section"); opciones.raiz.append(nodo);
    return Object.freeze({ desmontar: () => { llamadas.push([`${nombre}:desmontar`]); nodo.remove(); }, ...extra });
  };
  let olvidos = 0;
  const recursos = { ...reales,
    saldo: { montarVistaSaldoCronos: parte("saldo") }, remoto: { montarVistaRemotoCronos: parte("remoto") },
    movimientos: { montarVistaMovimientosCronos: parte("movimientos") },
    movimientosPropios: { montarMovimientosPropiosCronos: parte("calendario", { abrirOlvido: () => { olvidos += 1; } }) },
    permisosPropios: { montarPermisosPropiosCronos: (opciones) => { llamadas.push(["permisos", opciones]);
      const r = parte("permisos-montaje")(opciones); opciones.registrarDesmontar?.(r.desmontar); return r; } },
  };
  const crear = (cronos) => crearCoordinadorModulosPortal({ escaparHTML: String, entorno: { fetch: async () => { throw new Error("sin red"); } },
    cargarCatalogoInterno: async () => [{ clave: "cronos" }], cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("no debe cargar CT"); }, cronos: async () => cronos } });
  const coordinador = crear(recursos);
  await coordinador.cargarInterno();
  const raiz = raizDietasFalsa();
  assert.equal(await coordinador.montarVista("cronos", raiz), true);
  const movimientos = llamadas.find(([nombre]) => nombre === "movimientos")[1];
  assert.equal(typeof movimientos.abrirCorreccion, "function");
  assert.equal(typeof movimientos.cliente.consultar, "function");
  movimientos.abrirCorreccion();
  assert.equal(olvidos, 1);
  assert.equal(await coordinador.montarVista("cronos-permisos", raiz), true);
  assert.deepEqual(llamadas.filter(([n]) => n.endsWith(":desmontar")).map(([n]) => n),
    ["calendario:desmontar", "movimientos:desmontar", "remoto:desmontar", "saldo:desmontar"]);
  const permisos = llamadas.find(([nombre]) => nombre === "permisos")[1];
  assert.equal(typeof permisos.cliente.consultarPermisos, "function");
  coordinador.desmontarVistaActual();
  assert.equal(raiz.children.length, 0);
  for (const falta of ["saldo", "remoto", "movimientos", "movimientosPropios", "permisosPropios", "clienteSaldo", "clienteRemoto", "clienteSolicitudes"]) {
    const incompleto = crear({ ...recursos, [falta]: {} });
    await incompleto.cargarInterno();
    assert.equal(incompleto.resolverAcceso("cronos").disponible, false, falta);
  }
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
