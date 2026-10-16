import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";
import test from "node:test";
import { renderizarVistaPoliticaCeseRRHH } from "./modulos/bolsa/rrhh-politica-cese-vista.js";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";

// Reglas y versiones: un 404 de /api/vec/bolsa/politica-cese abre la vista para
// decir «no disponible» (antes devolvía a Inicio); un 403 sigue en Inicio.
const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");
const cuerpo = (nombre) => { const i = fuente.indexOf(`function ${nombre}(`); return fuente.slice(i, fuente.indexOf("\n}\n", i)); };

test("el 404 de la política de cese abre Reglas como no disponible y el 403 vuelve a Inicio", () => {
  const comprobar = cuerpo("comprobarAccesoPoliticaCese");
  assert.match(comprobar, /estado\.politicaCeseAusente = error\?\.estado === 404;/u);
  assert.match(comprobar, /if \(estado\.politicaCeseAusente && estado\.vista === "portal"\) navegar\("reglas"\);\s*else history\.replaceState\(null, "", rutaDeVista\("portal"\)\);/u);
  assert.match(cuerpo("vistaPermitida"), /vista === "reglas" && \(estado\.politicaCese !== null \|\| estado\.politicaCeseAusente\)\)\s*return coordinadorModulos\.obtenerCatalogo\(\)\.some\(\(m\) => m\.clave === "bolsa"\)/u);
  assert.match(cuerpo("montarVistaBolsa"), /noDisponible: estado\.politicaCeseAusente/u);
  assert.match(cuerpo("cargarFuenteDatos"), /estado\.politicaCeseAusente = false;/u);
});

test("el menú abre la política real con el perfil propio y una sola consulta autorizada", async () => {
  const inicio = fuente.indexOf("let consultaAccesoPlantillas = null;");
  const fin = fuente.indexOf("async function comprobarAccesoPlantillas()", inicio);
  assert.ok(inicio >= 0 && fin > inicio);
  const politica = { version: 2, catalogo_ref: "politica:bolsa:cese:2", catalogo_sha256: "a".repeat(64),
    mapeo: { interinidad: "general" }, meses_general: 5, meses_acumulacion: 9,
    publicada_en: "2026-09-28T10:00:00Z" };
  let consultas = 0, montajes = 0;
  const estado = { vista: "reglas", perfilPropioBolsa: true, politicaCese: null,
    politicaCeseComprobada: false, politicaCeseAusente: false };
  const entorno = { estado, AbortController, VISTA_PLANTILLAS_RRHH: "plantillas-rrhh",
    coordinadorModulos: { vistaDisponible: () => false }, clientePoliticaCese: {
      consultar: async () => { consultas += 1; return politica; },
    }, actualizarVistaBolsa: () => { montajes += 1; }, actualizarNavegacionModulos: () => {},
    navegar: () => assert.fail("no debe volver a Inicio"), history: {}, rutaDeVista: () => "#portal" };
  const { sondearCapacidadAlAbrir, comprobarAccesoPoliticaCese } = runInNewContext(
    `${fuente.slice(inicio, fin)}; ({ sondearCapacidadAlAbrir, comprobarAccesoPoliticaCese })`, entorno);
  assert.equal(sondearCapacidadAlAbrir("reglas"), false);
  assert.equal(consultas, 0, "la comprobación empieza al montar la vista, no al cargar el menú");
  await comprobarAccesoPoliticaCese();
  assert.equal(consultas, 1);
  assert.equal(montajes, 1);
  await prepararTextosPortal("bolsa");
  const html = renderizarVistaPoliticaCeseRRHH({ politica: estado.politicaCese });
  assert.match(html, /5 meses naturales/u);
  assert.match(html, /9 meses naturales/u);
  assert.match(cuerpo("montarVistaBolsa"), /vista === "reglas"[\s\S]*?montarVistaPoliticaCeseRRHH\(\{ raiz: contenedor,[\s\S]*?politica: estado\.politicaCese/u);
  assert.match(cuerpo("renderizar"), /"alegaciones", "reglas"\]\.includes\(estado\.vista\)/u);
});

test("una respuesta tardía no remonta Reglas después de navegar a otro módulo", async () => {
  const inicio = fuente.indexOf("let consultaAccesoPlantillas = null;");
  const fin = fuente.indexOf("async function comprobarAccesoPlantillas()", inicio);
  let resolver;
  const estado = { vista: "reglas", perfilPropioBolsa: true, politicaCese: null,
    politicaCeseComprobada: false, politicaCeseAusente: false };
  let montajes = 0, cambiosMenu = 0;
  const entorno = { estado, AbortController, clientePoliticaCese: { consultar: () => new Promise((r) => { resolver = r; }) },
    actualizarVistaBolsa: () => { montajes += 1; }, actualizarNavegacionModulos: () => { cambiosMenu += 1; },
    navegar: () => assert.fail("la respuesta no debe redirigir"), history: {}, rutaDeVista: () => "#portal" };
  const comprobar = runInNewContext(`${fuente.slice(inicio, fin)}; comprobarAccesoPoliticaCese`, entorno);
  const pendiente = comprobar();
  estado.vista = "contratacion-temporal";
  resolver({ meses_general: 5, meses_acumulacion: 9 });
  await pendiente;
  assert.equal(montajes, 0);
  assert.equal(cambiosMenu, 1);
  assert.match(cuerpo("navegar"), /consultaAccesoPoliticaCese\?\.abort\(\); consultaAccesoPoliticaCese = null;/u);
});

test("el 403 al abrir Reglas borra la política anterior y devuelve a Inicio", async () => {
  const inicio = fuente.indexOf("let consultaAccesoPlantillas = null;");
  const fin = fuente.indexOf("async function comprobarAccesoPlantillas()", inicio);
  const estado = { vista: "reglas", perfilPropioBolsa: true, politicaCese: { meses_general: 5 },
    politicaCeseComprobada: false, politicaCeseAusente: false };
  const destinos = [];
  const entorno = { estado, AbortController, clientePoliticaCese: { consultar: async () => {
    throw Object.assign(new Error("denegada"), { estado: 403 });
  } }, actualizarVistaBolsa: () => assert.fail("no debe montar política"),
  actualizarNavegacionModulos: () => {}, navegar: (vista) => { destinos.push(vista); estado.vista = vista; },
  history: {}, rutaDeVista: () => "#portal" };
  const comprobar = runInNewContext(`${fuente.slice(inicio, fin)}; comprobarAccesoPoliticaCese`, entorno);
  await comprobar();
  assert.equal(estado.politicaCese, null);
  assert.deepEqual(destinos, ["portal"]);
});

test("B45 positivo abre el enlace directo aunque el perfil propio aún no haya respondido", async () => {
  const estado = { vista: "portal", perfilPropioBolsa: false, politicaCese: null,
    politicaCeseComprobada: false, politicaCeseAusente: false };
  const catalogo = [{ clave: "bolsa" }];
  const entorno = { estado, AbortController, VISTA_PLANTILLAS_RRHH: "plantillas-rrhh",
    vistaBolsaPendienteNoCompuesta: () => false, VISTAS_MODULOS_PERSONALES: new Set(),
    VISTAS_AUTOSERVICIO_EMPLEADO: new Set(), moduloDeVistaPortal: () => "bolsa",
    vistaBolsaNavegable: () => false, capacidadesBolsa: () => ({ perfilPropioBolsa: false }),
    coordinadorModulos: { obtenerCatalogo: () => catalogo },
    clientePoliticaCese: { consultar: async () => ({ meses_general: 5, meses_acumulacion: 9 }) },
    actualizarVistaBolsa: () => {}, actualizarNavegacionModulos: () => {},
    navegar: (vista) => { if (vista === "reglas") assert.equal(entorno.vistaPermitida(vista), true); estado.vista = vista; },
    history: {}, rutaDeVista: () => "#portal" };
  entorno.vistaPermitida = runInNewContext(`${cuerpo("vistaPermitida")}\n}; vistaPermitida`, entorno);
  assert.equal(entorno.vistaPermitida("reglas"), false, "el perfil pendiente no concede B45 antes de leerlo");
  const inicio = fuente.indexOf("let consultaAccesoPlantillas = null;");
  const fin = fuente.indexOf("async function comprobarAccesoPlantillas()", inicio);
  const comprobar = runInNewContext(`${fuente.slice(inicio, fin)}; destinoPoliticaCeseInicial = true; comprobarAccesoPoliticaCese`, entorno);
  await comprobar();
  assert.equal(estado.vista, "reglas");
  assert.equal(estado.perfilPropioBolsa, false);
  assert.equal(estado.politicaCese.meses_general, 5);
  estado.politicaCese = null;
  assert.equal(entorno.vistaPermitida("reglas"), false, "sin B45 ni perfil la ruta vuelve a cerrarse");
  estado.politicaCeseAusente = true;
  assert.equal(entorno.vistaPermitida("reglas"), true, "el 404 mantiene su aviso controlado");
  catalogo.length = 0;
  assert.equal(entorno.vistaPermitida("reglas"), false, "no muestra Bolsa ausente del catálogo");
});
