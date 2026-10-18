import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  cargarMensajesPortal,
  crearTraductorPortal,
  MENSAJES_PORTAL,
} from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";

import { comprobarGrafoVersionado, exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const MENSAJES_PORTAL_INGLES = await cargarMensajesPortal("en");

test("la auditoría del expediente tiene textos simétricos ES y EN", async () => {
  const es = crearTraductorPortal();
  const en = crearTraductorPortal(MENSAJES_PORTAL_INGLES);
  assert.equal(es("auditoria_expediente_panel"), "Auditoría del expediente");
  assert.equal(es("auditoria_expediente_accion"), "Consultar auditoría de este expediente");
  assert.equal(en("auditoria_expediente_panel"), "Case audit trail");
  assert.equal(en("auditoria_expediente_accion"), "View this case’s audit trail");
  const vista = await readFile(new URL("modulos/contratacion-temporal/vista-expedientes.js", import.meta.url), "utf8");
  for (const clave of ["auditoria_expediente_panel", "auditoria_expediente_accion"]) {
    assert.match(vista, new RegExp(`traducirPortal\\(\"${clave}\"\\)`, "u"));
  }
});

test("miga, título, navegación y pie de CT usan el catálogo común en ambos idiomas", async () => {
  const es = crearTraductorPortal();
  const en = crearTraductorPortal(MENSAJES_PORTAL_INGLES);
  const textos = [
    ["contratacion_temporal_miga", "Portal del Empleado → Peticiones de personal temporal", "Employee Portal → Temporary staff requests"],
    ["contratacion_temporal_titulo", "Gestión de peticiones de personal temporal", "Manage temporary staff requests"],
    ["plantillas_rrhh_nav", "Plantillas de documentos", "Document templates"],
    ["txt_modulos", "Áreas", "Areas"],
    ["txt_modulos_del_portal", "Áreas del portal", "Portal areas"],
    ["txt_portal_de_recursos_humanos", "Portal de Recursos Humanos", "Human Resources Portal"],
    ["txt_2026_diputacion_de_granada_portal_del_empleado", "© 2026 Diputación de Granada · Portal del Empleado", "© 2026 Diputación de Granada · Employee Portal"],
    ["txt_proteccion_de_datos_accesibilidad_ayuda", "Protección de datos · Accesibilidad · Ayuda", "Data protection · Accessibility · Help"],
  ];
  const [portal, html] = await Promise.all([
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  for (const [clave, textoES, textoEN] of textos) {
    assert.equal(es(clave), textoES, clave);
    assert.equal(en(clave), textoEN, clave);
    if (clave.startsWith("contratacion_temporal_")) assert.ok(portal.includes(`traducirPortal("${clave}")`));
    else assert.ok(html.includes(`data-i18n-portal${clave === "txt_modulos_del_portal" ? "-aria-label" : ""}="${clave}"`), clave);
  }
});

test("HTML, caché y módulos usan la misma versión calculada por recurso", async () => {
  const destinos = await comprobarGrafoVersionado(new URL("./", import.meta.url));
  for (const recurso of ["portal.js", "portal-modulos-coordinador.js", "portal-i18n.js"]) {
    assert.ok(destinos.get(recurso).length >= 2, `${recurso}: varios importadores reales`);
  }
});

test("el catálogo i18n cubre los estados nuevos de acceso, navegación y reintento", () => {
  const traducir = crearTraductorPortal();
  for (const clave of Object.keys(MENSAJES_PORTAL)) {
    assert.equal(typeof traducir(clave), "string");
    assert.notEqual(traducir(clave), "");
  }
  assert.match(traducir("acceso_borradores_denegado"), /permiso/);
  assert.match(traducir("accion_reintentar"), /Reintentar/);
  assert.match(traducir("error_catalogo_modulos"), /áreas del portal/u);
  assert.match(traducir("titulo_error_catalogo_modulos"), /áreas/u);
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
    MENSAJES_BOLSA_INTERNA,
    formatearFechaPortal,
    formatearNumeroPortal,
  } = await import("./portal-i18n.js");
  const traducir = crearTraductorBolsaInterna();
  for (const clave of Object.keys(MENSAJES_BOLSA_INTERNA)) {
    assert.notEqual(traducir(clave), "");
  }
  assert.equal(traducir("numero_convocatorias", { numero: "3" }), "3 convocatorias encontradas.");
  assert.equal(formatearNumeroPortal(12345), "12.345");
  assert.equal(formatearFechaPortal("2026-08-01T09:00"), "1/8/26, 9:00");
  assert.throws(() => traducir("bolsa_texto_improvisado"), /desconocida/);
});

test("B24 traduce desde el catálogo común los eventos del recurso", async () => {
  const { traducirBolsaInterna, MENSAJES_BOLSA_INTERNA } = await import("./portal-i18n.js");
  assert.equal(traducirBolsaInterna("b24_recurso_historial", { total: 2 }), "Historial del recurso (2)");
  assert.equal(traducirBolsaInterna("b24_recurso_evento", { estado: "Interpuesto", fecha: "24 sept 2026", actor: "persona:registro" }),
    "Interpuesto · 24 sept 2026 · Anotado por persona:registro");
  assert.equal(traducirBolsaInterna("b24_recurso_documento", { referencia: "registro:1" }), "Escrito: registro:1");
  for (const clave of ["b24_recurso_historial", "b24_recurso_evento", "b24_recurso_documento"]) {
    assert.equal(typeof MENSAJES_BOLSA_INTERNA[clave], "string");
  }
});


test("Cronos comparte los traductores y la entrada entre sus importadores", async () => {
  const [coordinador, bandeja, avisos, permisos, portal, html] = await Promise.all([
    "portal-modulos-coordinador.js", "modulos/cronos/vista-bandeja-permisos.js",
    "modulos/cronos/vista-avisos-propios.js", "modulos/cronos/vista-permisos-propios.js", "portal.js", "index.html",
  ].map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  exigirMismaVersion([coordinador, bandeja, avisos], "i18n-resolucion.js");
  versionDe(permisos, "./i18n-permisos.js");
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
});

test("Dietas y borradores comparten sus recursos entre entradas", async () => {
  const rutas = ["index.html", "portal.js", "portal-modulos-coordinador.js",
    "modulos/dietas/vista-recorridos.js", "modulos/dietas/vista-borradores-propios.js",
    "portal-borradores-ui.js", "portal-borradores-acceso.js", "portal-borradores-ui-soporte.js",
    "portal-borradores-vista.js", "cache-publica-v1.json"];
  const [html, portal, coordinador, recorridos, borradores, ui, acceso, soporte, vista, cache] = await Promise.all(
    rutas.map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  exigirMismaVersion([portal, coordinador], "portal-i18n.js");
  exigirMismaVersion([html, cache], "portal.js");
  exigirMismaVersion([ui, acceso, soporte], "portal-borradores-api.js");
  exigirMismaVersion([html, ui], "portal-borradores-vista.js");
  exigirMismaVersion([html, ui, vista], "portal-borradores-ui-soporte.js");
  versionDe(coordinador, "./modulos/dietas/vista-recorridos.js");
  versionDe(recorridos, "./vista-borradores-propios.js");
  versionDe(borradores, "./vista-mapa-comision.js");
});
