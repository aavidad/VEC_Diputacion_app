import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  crearTraductorPortal,
  MENSAJES_PORTAL_ES,
  MENSAJES_PORTAL_EN,
  MENSAJES_BOLSA_INTERNA_ES,
  MENSAJES_BOLSA_INTERNA_EN,
} from "./portal-i18n.js";

test("los catálogos británicos tienen las mismas claves y marcadores que los castellanos", () => {
  const marcadores = (texto) => [...texto.matchAll(/\{([a-z_]+)\}/gu)].map((m) => m[1]).sort();
  for (const [es, en] of [
    [MENSAJES_PORTAL_ES, MENSAJES_PORTAL_EN],
    [MENSAJES_BOLSA_INTERNA_ES, MENSAJES_BOLSA_INTERNA_EN],
  ]) {
    assert.deepEqual(Object.keys(en).sort(), Object.keys(es).sort());
    for (const clave of Object.keys(es)) {
      assert.ok(en[clave].trim(), `traducción inglesa vacía: ${clave}`);
      assert.deepEqual(marcadores(en[clave]), marcadores(es[clave]), clave);
    }
  }
  assert.equal(crearTraductorPortal(MENSAJES_PORTAL_EN)("selector_idioma_etiqueta"), "Interface language");
  assert.equal(crearTraductorPortal(MENSAJES_PORTAL_EN)("paginacion_marco_recuento", { inicio: 1, fin: 10, total: 20 }), "Showing 1–10 of 20");
});

test("la selección inglesa toma el catálogo y los formatos en-GB de la autoridad común", () => {
  const script = `globalThis.navigator = { languages: ["en-GB"] };
    const modulo = await import(${JSON.stringify(new URL("portal-i18n.js", import.meta.url).href)});
    console.log(JSON.stringify({ idioma: modulo.IDIOMA_PORTAL,
      localizacion: modulo.LOCALIZACION_PORTAL,
      titulo: modulo.traducirPortal("selector_idioma_etiqueta"),
      estado: modulo.traducirBolsaInterna("fecha_sin_valor"),
      cifra: modulo.formatearNumeroPortal(12345),
      fecha: modulo.formatearFechaPortal("2026-08-01T09:00") }));`;
  const resultado = JSON.parse(execFileSync(process.execPath, ["--input-type=module", "-e", script], { encoding: "utf8" }));
  assert.deepEqual(resultado, {
    idioma: "en", localizacion: "en-GB", titulo: "Interface language", estado: "No date",
    cifra: "12,345", fecha: "01/08/2026, 09:00",
  });
});

test("el catálogo i18n cubre los estados nuevos de acceso, navegación y reintento", () => {
  const traducir = crearTraductorPortal();
  for (const clave of Object.keys(MENSAJES_PORTAL_ES)) {
    assert.equal(typeof traducir(clave), "string");
    assert.notEqual(traducir(clave), "");
  }
  assert.match(traducir("acceso_borradores_denegado"), /permiso/);
  assert.match(traducir("accion_reintentar"), /Reintentar/);
  assert.match(traducir("error_catalogo_modulos"), /catálogo interno/u);
  assert.match(traducir("titulo_error_catalogo_modulos"), /módulos/u);
  assert.equal(traducir("personal_catalogo_profesional"), "Catálogo profesional de Personal");
});

test("un catálogo incompleto o una clave no gobernada fallan cerrados", () => {
  assert.throws(() => crearTraductorPortal({}), /incompleto/);
  const traducir = crearTraductorPortal();
  assert.throws(() => traducir("texto_improvisado"), /desconocida/);
});

test("Cronos toma migas y títulos de Jornada y Permisos del catálogo común", async () => {
  const traducir = crearTraductorPortal();
  assert.equal(traducir("cronos_miga"), "Portal del Empleado → Cronos");
  assert.equal(traducir("cronos_jornada_titulo"), "Cronos · jornada y fichajes");
  assert.equal(traducir("cronos_permisos_miga"), "Portal del Empleado → Cronos → Permisos");
  assert.equal(traducir("cronos_permisos_titulo"), "Cronos · permisos y ausencias");
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  for (const clave of ["cronos_miga", "cronos_jornada_titulo", "cronos_permisos_miga", "cronos_permisos_titulo"]) {
    assert.match(portal, new RegExp(`traducirPortal\\("${clave}"\\)`, "u"));
  }
  assert.doesNotMatch(portal, /"Portal del Empleado → Cronos(?: → Permisos)?"|"Cronos · (?:jornada y fichajes|permisos y ausencias)"/u);
});

test("Bolsa interna usa catálogo común y formatos es-ES para textos y valores", async () => {
  const {
    crearTraductorBolsaInterna,
    MENSAJES_BOLSA_INTERNA_ES,
    formatearFechaPortal,
    formatearNumeroPortal,
  } = await import("./portal-i18n.js");
  const traducir = crearTraductorBolsaInterna();
  for (const clave of Object.keys(MENSAJES_BOLSA_INTERNA_ES)) {
    assert.notEqual(traducir(clave), "");
  }
  assert.equal(traducir("numero_convocatorias", { numero: "3" }), "3 convocatorias encontradas.");
  assert.equal(formatearNumeroPortal(12345), "12.345");
  assert.equal(formatearFechaPortal("2026-08-01T09:00"), "1/8/26, 9:00");
  assert.throws(() => traducir("bolsa_texto_improvisado"), /desconocida/);
});

test("B24 traduce desde el catálogo común los eventos del recurso", async () => {
  const { traducirBolsaInterna, MENSAJES_BOLSA_INTERNA_ES } = await import("./portal-i18n.js");
  assert.equal(traducirBolsaInterna("b24_recurso_historial", { total: 2 }), "Historial del recurso (2)");
  assert.equal(traducirBolsaInterna("b24_recurso_evento", { estado: "Interpuesto", fecha: "24 sept 2026", actor: "persona:registro" }),
    "Interpuesto · 24 sept 2026 · Anotado por persona:registro");
  assert.equal(traducirBolsaInterna("b24_recurso_documento", { referencia: "registro:1" }), "Escrito: registro:1");
  for (const clave of ["b24_recurso_historial", "b24_recurso_evento", "b24_recurso_documento"]) {
    assert.equal(typeof MENSAJES_BOLSA_INTERNA_ES[clave], "string");
  }
});
