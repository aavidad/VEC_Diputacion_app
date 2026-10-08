import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
import * as rutasBolsa from "./portal-bolsas-ruta-filtros.js";
import { rutaPortalConFiltroCT } from "./portal-ct-ruta-filtro.js";

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
const inicioEnlaces = fuente.indexOf("function instalarEnlacesBolsa()");
const finEnlaces = fuente.indexOf("function instalarEventosAuditoriaBolsa()", inicioEnlaces);
assert.ok(inicioNavegar > 0 && finNavegar > inicioNavegar && inicioEnlaces > 0 && finEnlaces > inicioEnlaces);

function navegadorFichaCT({ bolsaRef = "bolsa:auxiliar", origenRef = "expediente:ct:1", cargaCT = "listo", bolsasCT = null,
  esperaRutas = null, fallanRutas = false } = {}) {
  const bolsa = { bolsa_ref: "bolsa:auxiliar", categoria: "Auxiliar", categoria_clave: "auxiliar", vigente_hasta: null };
  const contexto = { expedienteRef: "expediente:ct:1", carga: cargaCT,
    datos: { bolsas: bolsasCT ?? [bolsa] } };
  const ubicacion = new URL("https://vec.example/portal-empleado/#contratacion-temporal");
  const llamadas = { rutas: 0, getBolsas: 0, aplicar: 0, navegadas: [], avisos: [], clicks: 0 };
  const control = { tagName: "BUTTON", dataset: {
    bolsaRef, origenExpediente: origenRef, origenReferencia: "2026/CT-00001",
    origenCentro: "Residencia", origenInicio: "2026-10-20",
  }, closest(selector) {
    if (selector === '[data-accion="ver-bolsa"][data-bolsa-ref]') return this;
    if (selector === "[data-ct-bolsa-ficha]") return {};
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
    TITULOS: { "bolsa-candidatos": ["", "Candidatos"] },
    VISTA_PLANTILLAS_RRHH: "plantillas-rrhh", vistaBolsaPendienteNoCompuesta: () => false,
    sondearCapacidadAlAbrir: () => false, vistaPermitida: () => true,
    vistaNecesitaBolsa: (vista) => vista === "bolsa-candidatos", rutaDeVista: () => "#bolsa/bolsa-candidatos",
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
  runInNewContext(`${fuente.slice(inicioNavegar, finNavegar)}\n${fuente.slice(inicioEnlaces, finEnlaces)}\ninstalarEnlacesBolsa();`, contextoVM);
  const click = async () => listeners.get("click")({ target: control, button: 0,
    stopImmediatePropagation() { llamadas.clicks++; }, preventDefault() {} });
  return { click, control, contextoVM, llamadas, bolsa };
}

test("clic real CT con cuadro global y rutas ausentes llega a Bolsa y exige su GET posterior", async () => {
  const { click, contextoVM, llamadas, bolsa } = navegadorFichaCT();
  await click();
  assert.equal(llamadas.rutas, 1, "solo se importa el helper de rutas al pulsar");
  assert.equal(llamadas.getBolsas, 1, "la navegación inicia la lectura ordinaria de Bolsa");
  assert.equal(contextoVM.estado.datosBolsas.carga, "cargando", "el contexto CT no se presta como cuadro general");
  assert.equal(contextoVM.contextoBolsaCT, null, "la ficha se invalida al salir");
  assert.equal(contextoVM.window.location.hash, "#bolsa/bolsa-candidatos");
  assert.equal(contextoVM.window.location.searchParams.get("bolsa_ref"), bolsa.bolsa_ref);
  assert.equal(contextoVM.window.location.searchParams.get("origen_expediente"), "expediente:ct:1");
  assert.equal(contextoVM.window.location.searchParams.get("origen_inicio"), "2026-10-20");
  assert.equal(llamadas.avisos.some((mensaje) => mensaje.includes("no_esta_autorizada")), false);
  assert.equal(llamadas.aplicar, 1, "antes del GET no hay candidaturas ni acto");
  // El GET normal debe devolver una lista autorizada antes de aceptar la ruta.
  assert.equal(rutasBolsa.leerCandidatosBolsaCompartible(contextoVM.window.location.search, [bolsa]).bolsaRef, bolsa.bolsa_ref);
  assert.throws(() => rutasBolsa.leerCandidatosBolsaCompartible(contextoVM.window.location.search, []), RangeError);
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
