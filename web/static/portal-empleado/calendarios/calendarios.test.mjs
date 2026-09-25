import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { versionDe } from "../versiones-cache.test-helper.mjs";
import {
  API_CALENDARIO, ErrorCalendarios, crearCliente, etiquetaMunicipio, instanteDesdeCampo, mensajeError,
  renderizarCalendario, renderizarFestivos, renderizarMes, renderizarPlazo, tipoDia, validarCalendario, validarPlazo,
} from "./calendarios.js";
import { MENSAJES_CALENDARIOS_ES, crearTraductorCalendarios, formatearFechaCivil } from "./i18n.js";

const leer = (nombre) => readFileSync(new URL(nombre, import.meta.url), "utf8");

function version(id, tipo, ref, sintetica = false) {
  return { id, ambito: { tipo, ref }, anio: 2026, numero: 1, denominacion: `Calendario ${id}`,
    procedencia: { norma: "Norma", referencia: "ref", publicada_en: "2025-10-28", sintetica }, conocido_desde: "2025-11-01T08:00:00Z" };
}

function calendario() {
  const dias = [];
  for (let t = Date.UTC(2026, 0, 1); t < Date.UTC(2027, 0, 1); t += 86400000) {
    const f = new Date(t);
    const fecha = f.toISOString().slice(0, 10);
    const finde = f.getUTCDay() === 0 || f.getUTCDay() === 6;
    const motivos = [];
    if (fecha === "2026-02-28") motivos.push({ ambito: { tipo: "autonomico", ref: "es-an" }, efecto: "festivo", denominacion: "Día de Andalucía", version_id: "and", sintetico: false });
    if (fecha === "2026-12-24") motivos.push({ ambito: { tipo: "centro", ref: "centro-530" }, efecto: "no_laborable", denominacion: "Cierre interno sintético", version_id: "cen", sintetico: true });
    if (fecha === "2026-01-01") motivos.push({ ambito: { tipo: "nacional", ref: "es" }, efecto: "festivo", denominacion: "Año Nuevo", version_id: "nac", sintetico: false });
    dias.push({ fecha, fin_de_semana: finde, festivo_oficial: motivos.some((m) => m.efecto === "festivo"),
      inhabil_administrativo: finde || motivos.some((m) => m.efecto === "festivo"), laborable: !finde && motivos.length === 0, motivos: motivos.length ? motivos : null });
  }
  return { data: { centro_ref: "centro-530", anio: 2026, conocido_en: "2026-09-25T10:00:00Z", zona: "Europe/Madrid", comunidad_ref: "es-an",
    municipio_ref: "municipio:ine:18087", versiones: [version("nac", "nacional", "es"), version("and", "autonomico", "es-an"), version("cen", "centro", "centro-530", true)],
    dias, resumen: { dias_naturales: 365, laborables: 246, habiles_administrativos: 249, festivos_oficiales: 14, no_laborables_centro: 2 } } };
}

test("el catálogo i18n está completo y toda clave de la página existe", () => {
  const t = crearTraductorCalendarios();
  assert.throws(() => crearTraductorCalendarios({ titulo: "x" }), /incompleto/);
  assert.throws(() => t("desconocida"), /desconocida/);
  const html = leer("./index.html");
  for (const [, clave] of html.matchAll(/data-i18n(?:-label)?="([^"]+)"/gu)) assert.ok(Object.hasOwn(MENSAJES_CALENDARIOS_ES, clave), clave);
  const js = leer("./calendarios.js");
  for (const [, id] of js.matchAll(/\$\("([a-z-]+)"\)/gu)) assert.match(html, new RegExp(`id="${id}"`, "u"), id);
  assert.ok(!/style=|<script>/u.test(html), "sin estilos ni guiones en línea");
  assert.equal(formatearFechaCivil("2026-10-25"), "25 de octubre de 2026");
});

test("las versiones en caché se renuevan juntas", () => {
  const html = leer("./index.html");
  const js = leer("./calendarios.js");
  assert.equal(versionDe(html, "calendarios.js"), versionDe(html, "calendarios.css"));
  assert.equal(versionDe(js, "i18n.js"), versionDe(html, "calendarios.js"));
  const manifiesto = readFileSync(new URL("../../../interno.manifest", import.meta.url), "utf8");
  for (const f of ["index.html", "calendarios.js", "calendarios.css", "i18n.js"]) assert.match(manifiesto, new RegExp(`static/portal-empleado/calendarios/${f.replace(".", "\\.")}`, "u"));
});

test("el mes se pinta en rejilla de lunes a domingo con colores por ámbito", () => {
  const cal = validarCalendario(calendario());
  const porFecha = new Map(cal.dias.map((d) => [d.fecha, d]));
  const febrero = renderizarMes(2026, 2, porFecha);
  assert.equal((febrero.match(/class="cal-vacio"/gu) || []).length, 6 + 1); // empieza en domingo y acaba en sábado
  assert.match(febrero, /cal-dia--autonomico[^>]*title="28 de febrero de 2026: Sábado o domingo, Día de Andalucía \(Andalucía\), inhábil"/u);
  assert.match(febrero, /<caption>Febrero<\/caption>/u);
  assert.equal(tipoDia(porFecha.get("2026-12-24")), "centro");
  assert.equal(tipoDia(porFecha.get("2026-12-26")), "finde");
  assert.equal(tipoDia(porFecha.get("2026-12-23")), "habil");
  assert.equal((renderizarCalendario(cal).match(/<table class="cal-mes"/gu) || []).length, 12);
  const festivos = renderizarFestivos(cal);
  assert.match(festivos, /Cierre interno sintético<\/td><td>Centro<\/td><td><span class="cal-estado cal-estado--sintetico">Sintético/u);
  assert.match(festivos, /Año Nuevo<\/td><td>Nacional<\/td><td><span class="cal-estado cal-estado--oficial">Oficial/u);
});

test("las respuestas inválidas se rechazan y el texto se escapa", () => {
  const malo = calendario();
  malo.data.dias[0].fecha = "2025-12-31";
  assert.throws(() => validarCalendario(malo), ErrorCalendarios);
  const inyectado = calendario();
  inyectado.data.dias[0].motivos[0].denominacion = "<img src=x onerror=alert(1)>";
  const cal = validarCalendario(inyectado);
  assert.doesNotMatch(renderizarFestivos(cal), /<img/u);
  assert.throws(() => validarPlazo({ data: { inicio: "2026-01-01" } }), ErrorCalendarios);
});

test("el plazo explica vencimiento, prórroga y días excluidos", () => {
  const p = validarPlazo({ data: { inicio: "2026-10-23", primer_dia: "2026-10-24", fin_nominal: "2026-11-02", vencimiento: "2026-11-03", prorrogado: true,
    vence_antes_de: "2026-11-03T23:00:00Z", dias_excluidos: [{ fecha: "2026-11-02", fin_de_semana: false, motivos: [{ ambito: { tipo: "autonomico", ref: "es-an" }, efecto: "festivo", denominacion: "Lunes siguiente a Todos los Santos", version_id: "and", sintetico: false }] }],
    versiones_utilizadas: [version("nac", "nacional", "es")] } });
  const html = renderizarPlazo(p);
  assert.match(html, /Vence el 3 de noviembre de 2026/u);
  assert.match(html, /Prorrogado desde el 2 de noviembre de 2026/u);
  assert.match(html, /Lunes siguiente a Todos los Santos/u);
});

test("el cliente no envía identidad y traduce los errores de la API", async () => {
  const llamadas = [];
  const respuesta = (estado, cuerpo) => ({ ok: estado < 400, status: estado, text: async () => JSON.stringify(cuerpo) });
  const cliente = crearCliente(async (url, opciones) => {
    llamadas.push({ url, opciones });
    if (url.split("?")[0] === API_CALENDARIO) return respuesta(422, { error: { codigo: "calendario_no_publicado", detalle: { anio: 2027, faltan: [{ tipo: "local", ref: "municipio:ine:18087" }] } } });
    return respuesta(401, {});
  });
  await assert.rejects(cliente.calendario("centro-530", "2027", ""), (e) => {
    assert.equal(mensajeError(e), "No hay calendario publicado para 2027: falta Municipio INE 18087.");
    return true;
  });
  await assert.rejects(cliente.centros("2026", ""), (e) => e.codigo === "error_autenticacion_requerida");
  assert.equal(llamadas[0].url, `${API_CALENDARIO}?centro=centro-530&anio=2027`);
  for (const { opciones } of llamadas) {
    assert.deepEqual(Object.keys(opciones.headers), ["Accept"]);
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.method, "GET");
  }
  const caido = crearCliente(async () => { throw new TypeError("red"); });
  await assert.rejects(caido.centros("2026", ""), (e) => mensajeError(e).includes("no está disponible"));
});

test("municipios e instantes se presentan sin inventar nombres", () => {
  assert.equal(etiquetaMunicipio("municipio:sintetico:a"), "Municipio sintético A");
  assert.equal(etiquetaMunicipio("municipio:ine:18087"), "Municipio INE 18087");
  assert.equal(instanteDesdeCampo(""), "");
  // «Conocido en» se interpreta en hora de Madrid, no en la del navegador.
  assert.equal(instanteDesdeCampo("2026-04-20T10:00"), "2026-04-20T08:00:00.000Z");
  assert.equal(instanteDesdeCampo("2026-01-20T10:00"), "2026-01-20T09:00:00.000Z");
  assert.equal(instanteDesdeCampo("2026-03-29T02:30"), "", "hora inexistente del cambio de marzo");
  assert.equal(instanteDesdeCampo("2026-10-25T02:30"), "2026-10-25T00:30:00.000Z", "primera aparición de la hora repetida");
  assert.equal(instanteDesdeCampo("20/04/2026"), "");
});
