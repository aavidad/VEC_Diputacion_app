import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { MENSAJES_PERSONAL, crearTraductorPersonal, formatearFechaEstructuraOrganizativa, formatearRecuentoCategorias, formatearRecuentoRPT, prepararTextosPersonal } from "./i18n.js";

test.before(async () => {
  await prepararTextosPersonal();
});

test("el módulo se importa sin cargar textos y exige preparación antes de traducir", async () => {
  const aislado = await import("./i18n.js?sin-preparar");
  assert.equal(aislado.MENSAJES_PERSONAL, undefined);
  assert.throws(() => aislado.crearTraductorPersonal(), /pendientes de preparación/);
});

test("el catálogo de Personal advierte de la naturaleza DEMO y de sus límites", () => {
  const t = crearTraductorPersonal();
  assert.match(t("presentacion_demo"), /DEMO/);
  assert.match(t("aviso_demo"), /no consulta datos reales/i);
  assert.match(t("servicios_ayuda"), /no se calcula antigüedad, trienios/i);
  assert.match(t("nominas_ayuda"), /No se muestran importes/i);
  assert.match(t("dietas_ayuda"), /no acreditan liquidación/i);
  assert.match(t("ficha_accesos_ayuda"), /no envía identificadores ni acredita una relación de servicio/i);
  assert.match(t("ficha_estado_no_configurado"), /Fuente no conectada/i);
  assert.match(t("ficha_catalogos_completos"), /No acreditan ocupación actual/i);
  assert.match(t("ficha_abrir_ayuda"), /^\? /);
  assert.match(t("ficha_tiempo_ayuda"), /fichaje no acredita servicios reconocidos/i);
  assert.match(t("ficha_no_configurado"), /ausencia de datos no equivale a cero/i);
  assert.match(t("ficha_desplazar_tabla"), /horizontalmente/i);
  assert.equal(t("actualizado", { fecha: "20/09/2026" }), "Actualizado: 20/09/2026");
  assert.ok(Object.isFrozen(MENSAJES_PERSONAL));
});

test("el traductor rechaza catálogos y claves incompletos", () => {
  assert.throws(() => crearTraductorPersonal({ titulo: "incompleto" }), /incompleto/);
  assert.throws(() => crearTraductorPersonal()("desconocida"), /desconocida/);
});

test("el catálogo conectado conserva textos y plural localizados", () => {
  const t = crearTraductorPersonal();
  assert.match(t("catalogo_error"), /No se muestran datos anteriores/);
  assert.match(t("catalogo_demo"), /pendiente de validación por RRHH/);
  assert.equal(formatearRecuentoCategorias(1), "1 categoría");
  assert.equal(formatearRecuentoCategorias(1_000), "1000 categorías");
  assert.throws(() => formatearRecuentoCategorias(-1), /no válido/);
});

test("la RPT pública tiene catálogo completo y recuento localizado", () => {
  const t = crearTraductorPersonal();
  ["rpt_titulo", "rpt_ayuda", "rpt_error", "rpt_fuente", "rpt_huella", "rpt_vacio", "rpt_tabla", "rpt_paginacion", "rpt_cabecera_dotacion"].forEach((clave) => assert.notEqual(t(clave), ""));
  assert.match(t("rpt_fuente", { documento: "RPT", aviso: "sin ocupantes" }), /RPT/);
  assert.equal(formatearRecuentoRPT(1), "1 categoría RPT");
  assert.equal(formatearRecuentoRPT(1_000), "1000 categorías RPT");
  assert.throws(() => formatearRecuentoRPT(-1), /no válido/);
});

test("la fecha de estructura se presenta en castellano y Europe/Madrid", () => {
  const fecha = formatearFechaEstructuraOrganizativa("2026-09-06T00:00:00Z");
  assert.match(fecha, /6 sept 2026|6\/9\/2026|06\/09\/2026/);
  assert.match(fecha, /Europe\/Madrid/);
  assert.doesNotMatch(fecha, /2026-09-06T00:00:00Z/);
  assert.throws(() => formatearFechaEstructuraOrganizativa("2026-09-06"), /no válida/);
});

test("Personal adopta el idioma efectivo y comunica la incidencia de respaldo", async () => {
  const ingles = await prepararTextosPersonal({ idioma: "en", porDefecto: "es" });
  assert.equal(ingles.idioma, "en");
  assert.equal(ingles.localizacion, "en-GB");
  assert.equal(ingles.incidenciaCatalogo, null);
  assert.equal(formatearRecuentoCategorias(1_000), "1,000 categories");

  const respaldo = JSON.parse(await readFile(new URL("../../../textos/es/personal.json", import.meta.url), "utf8"));
  const lecturas = [];
  const resultado = await prepararTextosPersonal({
    idioma: "en", porDefecto: "es", avisar: () => {},
    leer: async (url) => {
      lecturas.push(url.pathname);
      if (url.pathname.endsWith("/en/personal.json")) throw new Error("catálogo temporalmente inaccesible");
      return respaldo;
    },
  });
  assert.equal(resultado.idioma, "es");
  assert.equal(resultado.localizacion, "es-ES");
  assert.equal(resultado.incidenciaCatalogo.codigo, "catalogo_no_disponible");
  assert.deepEqual(lecturas.map((ruta) => ruta.match(/\/([^/]+)\/personal\.json$/u)[1]), ["en", "es"]);
  assert.equal(formatearRecuentoCategorias(1_000), "1000 categorías");
});

test("un catálogo inaccesible no rompe la importación y permite otra preparación", async () => {
  const aislado = await import("./i18n.js?recuperacion");
  const datos = JSON.parse(await readFile(new URL("../../../textos/es/personal.json", import.meta.url), "utf8"));
  await assert.rejects(aislado.prepararTextosPersonal({
    idioma: "es", porDefecto: "es", avisar: () => {}, leer: async () => { throw new Error("sin catálogo"); },
  }), /sin catálogo/);
  assert.throws(() => aislado.crearTraductorPersonal(), /pendientes de preparación/);
  const recuperado = await aislado.prepararTextosPersonal({
    idioma: "es", porDefecto: "es", avisar: () => {}, leer: async () => datos,
  });
  assert.equal(recuperado.idioma, "es");
  assert.equal(aislado.crearTraductorPersonal()("catalogo_recuento_uno", { total: "1" }), "1 categoría");
});

test("un JSON válido con general vacío no publica textos y se recupera al reintentar", async () => {
  const aislado = await import("./i18n.js?general-vacio");
  const datos = JSON.parse(await readFile(new URL("../../../textos/es/personal.json", import.meta.url), "utf8"));
  let lecturas = 0;
  await assert.rejects(aislado.prepararTextosPersonal({
    idioma: "es", porDefecto: "es", avisar: () => {},
    leer: async () => { lecturas++; return { general: {} }; },
  }), /no válido|incompleto/);
  assert.equal(lecturas, 2);
  assert.equal(aislado.MENSAJES_PERSONAL, undefined);
  assert.throws(() => aislado.crearTraductorPersonal(), /pendientes de preparación/);
  const recuperado = await aislado.prepararTextosPersonal({
    idioma: "es", porDefecto: "es", avisar: () => {}, leer: async () => datos,
  });
  assert.equal(recuperado.idioma, "es");
  assert.ok(Object.isFrozen(aislado.MENSAJES_PERSONAL));
  assert.equal(aislado.crearTraductorPersonal()("catalogo_recuento_uno", { total: "1" }), "1 categoría");
});

function diferida() {
  let resolver;
  let rechazar;
  const promesa = new Promise((resolve, reject) => { resolver = resolve; rechazar = reject; });
  return { promesa, resolver, rechazar };
}

test("dos preparaciones simultáneas resuelven con el último idioma ya disponible", async () => {
  const aislado = await import("./i18n.js?preparaciones-simultaneas");
  const datosES = JSON.parse(await readFile(new URL("../../../textos/es/personal.json", import.meta.url), "utf8"));
  const datosEN = JSON.parse(await readFile(new URL("../../../textos/en/personal.json", import.meta.url), "utf8"));
  const primeraLectura = diferida();
  const ultimaLectura = diferida();
  const primera = aislado.prepararTextosPersonal({ idioma: "es", porDefecto: "es", leer: () => primeraLectura.promesa });
  const ultima = aislado.prepararTextosPersonal({ idioma: "en", porDefecto: "es", leer: () => ultimaLectura.promesa });
  let primeraTerminada = false;
  primera.then(() => { primeraTerminada = true; });
  primeraLectura.resolver(datosES);
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(primeraTerminada, false);
  assert.throws(() => aislado.crearTraductorPersonal(), /pendientes de preparación/);
  ultimaLectura.resolver(datosEN);
  const [resultadoPrimero, resultadoUltimo] = await Promise.all([primera, ultima]);
  assert.equal(resultadoPrimero.idioma, "en");
  assert.equal(resultadoUltimo.idioma, "en");
  assert.equal(aislado.formatearRecuentoCategorias(1_000), "1,000 categories");
});

test("si la última preparación falla, ninguna llamada anterior confirma textos obsoletos", async () => {
  const aislado = await import("./i18n.js?ultima-preparacion-fallida");
  const datos = JSON.parse(await readFile(new URL("../../../textos/es/personal.json", import.meta.url), "utf8"));
  const primeraLectura = diferida();
  const ultimaLectura = diferida();
  const primera = aislado.prepararTextosPersonal({ idioma: "es", porDefecto: "es", leer: () => primeraLectura.promesa });
  const ultima = aislado.prepararTextosPersonal({ idioma: "es", porDefecto: "es", leer: () => ultimaLectura.promesa });
  primeraLectura.resolver(datos);
  await new Promise((resolver) => setImmediate(resolver));
  ultimaLectura.rechazar(new Error("último catálogo inaccesible"));
  const resultados = await Promise.allSettled([primera, ultima]);
  assert.deepEqual(resultados.map(({ status }) => status), ["rejected", "rejected"]);
  assert.match(resultados[0].reason.message, /último catálogo inaccesible/);
  assert.match(resultados[1].reason.message, /último catálogo inaccesible/);
  assert.equal(aislado.MENSAJES_PERSONAL, undefined);
  assert.throws(() => aislado.crearTraductorPersonal(), /pendientes de preparación/);
});
