import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import * as rutasBolsa from "./portal-bolsas-ruta-filtros.js?v=20261008-bolsa-global-v2";
import { rutaPortalConFiltroCT } from "./portal-ct-ruta-filtro.js";
import { vistaBolsaNavegable, vistaBolsaOfrecida } from "./portal-menu-bolsa.js";
import { moduloDeVistaPortal } from "./portal-modulos-coordinador.js";

const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");
const inicio = fuente.indexOf("resolverBolsa: (");
const fin = fuente.indexOf("    montar:", inicio);
assert.ok(inicio > 0 && fin > inicio);

function resolver({ permiso = true, carga = "listo", bolsas = [], ct = false, ctCarga = "listo", ctBolsas = [] } = {}) {
  return runInNewContext(`({${fuente.slice(inicio, fin)}}).resolverBolsa`, {
    vistaPermitida: () => permiso,
    VISTA_CANDIDATOS_BOLSA: "bolsa-candidatos",
    estado: { vista: ct ? "contratacion-temporal" : "portal", datosBolsas: { carga, datos: { bolsas } } },
    contextoBolsaCT: ct ? { carga: ctCarga, datos: { bolsas: ctBolsas } } : null,
  });
}

test("la categoría sin bolsa se confirma únicamente desde el conjunto autorizado listo", () => {
  for (const bolsas of [[], [{ bolsa_ref: "bolsa:antigua", categoria_clave: "auxiliar", vigente_hasta: "2026-01-01" }]]) {
    assert.equal(resolver({ bolsas })("", { categoriaRef: "categoria:rpt:auxiliar" }).estado, "sin_bolsa");
  }
  for (const carga of ["cargando", "error", "vacio", "denegado"]) {
    assert.equal(resolver({ carga })("", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  }
  assert.equal(resolver({ permiso: false })("", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  assert.equal(resolver({ bolsas: null })("", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  assert.equal(resolver()("bolsa:oculta", { categoriaRef: "categoria:rpt:auxiliar" }), null);
  assert.equal(resolver()("", { categoriaRef: "" }), null);
});

test("la bolsa vigente conserva el acceso desde CT sin crear otro llamamiento", () => {
  const bolsa = { bolsa_ref: "bolsa:auxiliar", categoria_clave: "auxiliar", categoria: "Auxiliar administrativo", vigente_hasta: null };
  const buscar = resolver({ bolsas: [bolsa] });
  const porCategoria = buscar("", { categoriaRef: "categoria:rpt:auxiliar" });
  assert.equal(porCategoria.bolsa_ref, bolsa.bolsa_ref);
  assert.equal(porCategoria.categoria, bolsa.categoria);
  assert.equal(buscar(bolsa.bolsa_ref).categoria, bolsa.categoria);
  assert.equal(buscar("bolsa:otra"), null);
});

test("la ficha CT usa solo su contexto autorizado y expone error o denegación como estados distintos", () => {
  const bolsa = { bolsa_ref: "bolsa:auxiliar", categoria_clave: "auxiliar", categoria: "Auxiliar", vigente_hasta: null };
  assert.equal(resolver({ carga: "error", ct: true, ctBolsas: [bolsa] })("", { categoriaRef: "categoria:rpt:auxiliar" }).bolsa_ref, bolsa.bolsa_ref);
  assert.equal(resolver({ carga: "error", ct: true, ctBolsas: [] })("", { categoriaRef: "categoria:rpt:auxiliar" }).estado, "sin_bolsa");
  for (const [ctCarga, esperado] of [["error", "error"], ["denegado", "denegado"]]) {
    const buscar = resolver({ carga: "error", ct: true, ctCarga });
    assert.equal(buscar("", { categoriaRef: "categoria:rpt:auxiliar" }).estado, esperado);
  }
  assert.equal(resolver({ carga: "error", ct: true, ctCarga: "cargando" })("", { categoriaRef: "categoria:rpt:auxiliar" }), null);
});

const inicioNavegar = fuente.indexOf("function navegar(vista, opciones = {})");
const finNavegar = fuente.indexOf("function montarVistaBolsa(", inicioNavegar);
const inicioAplicarRuta = fuente.indexOf("function aplicarRutaCandidatosBolsa()");
const finAplicarRuta = fuente.indexOf("function actualizarVistaBolsa(", inicioAplicarRuta);
const inicioCapacidades = fuente.indexOf("function capacidadesBolsa()");
const finCapacidades = fuente.indexOf("function vistaPermitida(vista)", inicioCapacidades);
const inicioVistaPermitida = finCapacidades;
const finVistaPermitida = fuente.indexOf("function etiquetaFuentePanel()", inicioVistaPermitida);
const inicioEnlaces = fuente.indexOf("function instalarEnlacesBolsa()");
const finEnlaces = fuente.indexOf("function instalarEventosAuditoriaBolsa()", inicioEnlaces);
assert.ok(inicioNavegar > 0 && finNavegar > inicioNavegar && inicioEnlaces > 0 && finEnlaces > inicioEnlaces);
assert.ok(inicioAplicarRuta > 0 && finAplicarRuta > inicioAplicarRuta);
assert.ok(inicioCapacidades > 0 && finCapacidades > inicioCapacidades && finVistaPermitida > inicioVistaPermitida);

function navegadorFichaCT({ bolsaRef = "bolsa:auxiliar", origenRef = "expediente:ct:1", cargaCT = "listo", bolsasCT = null,
  esperaRutas = null, fallanRutas = false, historico = false } = {}) {
  const bolsa = { bolsa_ref: "bolsa:auxiliar", categoria: "Auxiliar", categoria_clave: "auxiliar", vigente_hasta: null };
  const contexto = { expedienteRef: "expediente:ct:1", carga: cargaCT,
    datos: { bolsas: bolsasCT ?? [bolsa] } };
  const ubicacion = new URL("https://vec.example/portal-empleado/#contratacion-temporal");
  const llamadas = { rutas: 0, getBolsas: 0, aplicar: 0, navegadas: [], avisos: [], clicks: 0 };
  const control = { tagName: "BUTTON", dataset: historico ? {
    bolsaRef, pestana: "historico",
  } : {
    bolsaRef, origenExpediente: origenRef, origenReferencia: "2026/CT-00001",
    origenCentro: "Residencia", origenInicio: "2026-10-20",
  }, classList: { contains: (clase) => historico && clase === "enlace-tabla" }, closest(selector) {
    if (selector === '[data-accion="ver-bolsa"][data-bolsa-ref]') return this;
    if (selector === "[data-ct-bolsa-ficha]") return historico ? null : {};
    if (selector === ".ct-expedientes") return historico ? {} : null;
    return null;
  }, hasAttribute: () => false };
  const listeners = new Map();
  const contextoVM = {
    URL, window: { location: ubicacion, addEventListener(tipo, fn) { listeners.set(tipo, fn); } },
    document: { addEventListener(tipo, fn) { listeners.set(tipo, fn); } },
    history: { pushState(_estado, _titulo, ruta) { contextoVM.window.location = new URL(ruta, contextoVM.window.location); },
      replaceState(_estado, _titulo, ruta) { contextoVM.window.location = new URL(ruta, contextoVM.window.location); } },
    estado: { vista: "contratacion-temporal", datosBolsas: null, filtrosBolsa: {}, datosCandidatos: null },
    contextoBolsaCT: contexto, rutasBolsa: null, controladorBolsas: null, rutaCandidatosAplicada: null,
    generacionCuadroBolsas: 0, cicloLecturaBolsas: 0,
    TITULOS: { "bolsa-candidatos": ["", "Candidatos"], llamamientos: ["", "Llamamientos"] },
    VISTA_PLANTILLAS_RRHH: "plantillas-rrhh", vistaBolsaPendienteNoCompuesta: () => false,
    sondearCapacidadAlAbrir: () => false, vistaBolsaNavegable, vistaBolsaOfrecida,
    moduloDeVistaPortal, superficieBorradores: { obtenerAcceso: () => ({ disponible: false }) },
    coordinadorModulos: { vistaDisponible: () => false, obtenerCatalogo: () => [] },
    VISTA_CANDIDATOS_BOLSA: "bolsa-candidatos", VISTAS_MODULOS_PERSONALES: new Set(),
    VISTAS_AUTOSERVICIO_EMPLEADO: new Set(),
    vistaNecesitaBolsa: (vista) => vista === "bolsa-candidatos" || vista === "llamamientos",
    rutaDeVista: (vista) => vista === "llamamientos" ? "#bolsa/llamamientos" : "#bolsa/bolsa-candidatos",
    selectorLlamamientos: null,
    rutaPortalConFiltroCT, filtroServidorCTValido: () => false,
    renderizar: () => {}, cerrarMenuMovil: () => {}, tituloDeVista: () => ["", "Candidatos"],
    anunciar: (mensaje) => llamadas.avisos.push(mensaje), traducirPortal: (clave) => clave,
    porId: () => ({ contains: (nodo) => nodo === control }),
    prepararRutasBolsa: async () => {
      llamadas.rutas++; await esperaRutas;
      if (fallanRutas) throw new Error("helper no disponible");
      contextoVM.rutasBolsa = rutasBolsa; return rutasBolsa;
    },
    pedirCuadroBolsas: () => { llamadas.getBolsas++; contextoVM.estado.datosBolsas = { carga: "cargando", datos: null }; },
    aplicarRutaCandidatosBolsa: () => { llamadas.aplicar++; return false; },
    esPerfilRRHH: () => false,
  };
  runInNewContext(`${fuente.slice(inicioCapacidades, finCapacidades)}\n${fuente.slice(inicioVistaPermitida, finVistaPermitida)}\n${fuente.slice(inicioNavegar, finNavegar)}\n${fuente.slice(inicioEnlaces, finEnlaces)}\ninstalarEnlacesBolsa();`, contextoVM);
  const click = async () => listeners.get("click")({ target: control, button: 0,
    stopImmediatePropagation() { llamadas.clicks++; }, preventDefault() {} });
  return { click, control, contextoVM, llamadas, bolsa };
}

test("clic CT abre el asistente de Bolsa y espera su lectura autorizada", async () => {
  const { click, contextoVM, llamadas, bolsa } = navegadorFichaCT();
  assert.equal(contextoVM.capacidadesBolsa().bolsasConsultables, false, "CT no presta su contexto al menú");
  assert.equal(contextoVM.vistaPermitida("llamamientos"), true, "la ruta puede mostrar carga antes del GET");
  assert.equal(vistaBolsaOfrecida("bolsa-candidatos", { bolsasConsultables: false }), false);
  await click();
  assert.equal(llamadas.rutas, 1, "solo se importa el helper de rutas al pulsar");
  assert.equal(llamadas.getBolsas, 1, "la navegación inicia la lectura ordinaria de Bolsa");
  assert.equal(contextoVM.estado.datosBolsas.carga, "cargando", "el contexto CT no se presta como cuadro general");
  assert.equal(contextoVM.contextoBolsaCT, null, "la ficha se invalida al salir");
  assert.equal(contextoVM.window.location.hash, "#bolsa/llamamientos");
  assert.equal(contextoVM.window.location.searchParams.get("bolsa_ref"), bolsa.bolsa_ref);
  assert.equal(contextoVM.window.location.searchParams.get("origen_expediente"), "expediente:ct:1");
  assert.equal(contextoVM.window.location.searchParams.get("origen_inicio"), "2026-10-20");
  assert.equal(llamadas.avisos.some((mensaje) => mensaje.includes("no_esta_autorizada")), false);
  assert.equal(llamadas.aplicar, 0, "antes del GET no hay candidaturas ni acto");
  // El GET normal debe devolver una lista autorizada antes de aceptar la ruta.
  assert.equal(rutasBolsa.leerLlamamientoBolsaCompartible(contextoVM.window.location.search, [bolsa]).bolsaRef, bolsa.bolsa_ref);
  assert.throws(() => rutasBolsa.leerLlamamientoBolsaCompartible(contextoVM.window.location.search, []), RangeError);
});

test("bolsa u origen ajenos y denegación CT no navegan ni cargan rutas", async () => {
  for (const opciones of [{ bolsaRef: "bolsa:ajena" }, { origenRef: "expediente:ct:ajeno" }, { cargaCT: "denegado" }]) {
    const { click, contextoVM, llamadas } = navegadorFichaCT(opciones);
    await click();
    assert.equal(llamadas.rutas, 0);
    assert.equal(llamadas.getBolsas, 0);
    assert.equal(contextoVM.estado.vista, "contratacion-temporal");
    assert.equal(contextoVM.window.location.hash, "#contratacion-temporal");
  }
});

test("un clic pendiente de cargar rutas no revive la ficha CT tras salir", async () => {
  let continuar;
  const esperaRutas = new Promise((resolver) => { continuar = resolver; });
  const { click, contextoVM, llamadas } = navegadorFichaCT({ esperaRutas });
  const pendiente = click();
  contextoVM.contextoBolsaCT = null;
  continuar();
  await pendiente;
  assert.equal(llamadas.rutas, 1);
  assert.equal(llamadas.getBolsas, 0);
  assert.equal(contextoVM.estado.vista, "contratacion-temporal");
  assert.equal(contextoVM.window.location.hash, "#contratacion-temporal");
  assert.equal(llamadas.avisos.length, 0, "la cancelación no se presenta como denegación");
});

test("un fallo al cargar rutas informa indisponibilidad y permite intentar de nuevo", async () => {
  const { click, contextoVM, llamadas } = navegadorFichaCT({ fallanRutas: true });
  await click();
  assert.equal(llamadas.getBolsas, 0);
  assert.equal(contextoVM.estado.vista, "contratacion-temporal");
  assert.deepEqual(llamadas.avisos, ["txt_no_se_pudieron_cargar_las_bolsas_de_trabajo"]);
});

test("el enlace histórico de cobertura usa la bolsa visible en la ficha y espera el GET general", async () => {
  const { click, contextoVM, llamadas, bolsa } = navegadorFichaCT({ historico: true });
  await click();
  assert.equal(llamadas.getBolsas, 1);
  assert.equal(contextoVM.estado.datosBolsas.carga, "cargando");
  assert.equal(contextoVM.estado.filtrosBolsa.pestana, "historico");
  assert.equal(contextoVM.window.location.searchParams.get("bolsa_ref"), bolsa.bolsa_ref);
  assert.equal(contextoVM.window.location.searchParams.has("origen_expediente"), false);
  assert.equal(llamadas.avisos.some((mensaje) => mensaje.includes("no_esta_autorizada")), false);
  // La lectura general revalida pertenencia antes de solicitar candidaturas.
  let candidatos = 0;
  contextoVM.controladorBolsas = { cargarCandidatosBolsa: () => { candidatos++; return Promise.resolve(); } };
  contextoVM.estado.datosBolsas = { carga: "listo", datos: { bolsas: [bolsa] } };
  runInNewContext(`${fuente.slice(inicioAplicarRuta, finAplicarRuta)}\naplicarRutaCandidatosBolsa();`, contextoVM);
  assert.equal(candidatos, 1);
  assert.equal(contextoVM.estado.filtrosBolsa.pestana, "historico");
});

test("histórico ajeno y revalidación 403 no abren candidaturas ni actos", async () => {
  for (const opciones of [{ historico: true, bolsaRef: "bolsa:ajena" }, { historico: true, cargaCT: "denegado" }]) {
    const { click, llamadas } = navegadorFichaCT(opciones);
    await click();
    assert.equal(llamadas.getBolsas, 0);
    assert.equal(llamadas.aplicar, 0);
  }
  const { click, contextoVM, llamadas } = navegadorFichaCT({ historico: true });
  await click();
  contextoVM.estado.datosBolsas = { carga: "denegado", datos: null };
  let candidatos = 0;
  contextoVM.controladorBolsas = { cargarCandidatosBolsa: () => { candidatos++; return Promise.resolve(); } };
  assert.equal(runInNewContext(`${fuente.slice(inicioAplicarRuta, finAplicarRuta)}\naplicarRutaCandidatosBolsa();`, contextoVM), false);
  assert.equal(candidatos, 0);
  assert.equal(llamadas.getBolsas, 1, "la nueva lectura sí se intentó antes de recibir 403");
});
