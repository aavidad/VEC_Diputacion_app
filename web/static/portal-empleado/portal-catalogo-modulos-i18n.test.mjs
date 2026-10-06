import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// El selector común fija el idioma al cargar el módulo, como en el navegador.
globalThis.location = { href: "https://vec.example/portal-empleado/?lang=en" };
const { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } = await import("../comun/idioma.js");
const { cambiarIdioma, montarSelectorIdioma } = await import("../comun/idioma.js");
const { cargarMensajesPortal, crearTraductorPortal, traducirPortal, formatearNumeroPortal } = await import("./portal-i18n.js");
const MENSAJES_PORTAL_CASTELLANO = await cargarMensajesPortal("es");
const { aplicarIdiomaDocumento, aplicarTextosPortal } = await import("./portal-idioma.js");
const { cargarCatalogoModulosInterno, presentarSesionPortal } = await import("./portal-catalogo-modulos.js");
const { crearVistaInicioPortal } = await import("./portal-inicio.js");

test("?lang=en elige traducción común, catálogo inglés y cifras en en-GB", async () => {
  const traducciones = JSON.parse(await readFile(new URL("../../../locales/en.json", import.meta.url), "utf8"));
  const manifiesto = { id: "vec.module.bolsa", name_key: "ui.vec.module.bolsa.name",
    description_key: "ui.vec.module.bolsa.description", version: "v1.0.0",
    group: "recursos_humanos", base_path: "/modules/bolsa",
    permissions: [{ key: "bolsa.consultar", label_key: "ui.permission.bolsa.consultar" }], menu: null };
  const llamadas = [];
  const catalogo = await cargarCatalogoModulosInterno(async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return { ok: true, json: async () => ruta === "/api/vec/modules"
      ? { data: { modules: [manifiesto] } } : traducciones };
  });
  assert.deepEqual([IDIOMA_ACTUAL, LOCALIZACION_ACTUAL, llamadas[1].ruta, catalogo[0].titulo,
    presentarSesionPortal({ nombre: "Ana", roles: ["tecnico_rrhh"] }).perfil,
    traducirPortal("contratacion_temporal_encabezado"), formatearNumeroPortal(1234)],
  ["en", "en-GB", "/locales/en.json", "Employment pools", "Human Resources", "Temporary staff requests", "1,234"]);
  assert.equal(llamadas[1].opciones.credentials, "same-origin");
  assert.equal(llamadas[1].opciones.redirect, "error");
  assert.equal(llamadas[1].opciones.cache, "no-store");
});

test("?lang=en renderiza la portada y sus estados con las claves inglesas", () => {
  const escaparHTML = (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
  const html = crearVistaInicioPortal({
    encabezadoVista: (_sobrelinea, titulo, _descripcion, acciones) =>
      `<header><h1>${escaparHTML(titulo)}</h1>${acciones}</header>`,
    escaparHTML, obtenerCatalogo: () => [],
    resolverAcceso: (clave) => clave === "bolsa"
      ? { disponible: false, estado: "denegado" } : { disponible: true, vista: "contratacion-temporal" },
    esPerfilRRHH: () => true,
    obtenerCuadroInicio: () => ({ generadoEn: "2026-09-29T07:00:00Z", resumen: { en_tramite: 0, con_incidencia: 0,
      vencidos: 0, vencen_hoy: 0, vencen_semana: 0, sin_calcular: 0, por_fase: {} } }),
    ahora: () => new Date("2026-09-29T08:00:00Z"),
    locale: "en-GB",
  })();
  assert.match(html, /<h3 id="inicio-rrhh-pendientes-titulo">Pending<\/h3>/u);
  assert.match(html, /No deadline is due today and there are no open issues\./u);
  assert.match(html, /No permission for this profile|Your session does not have permission/u);
  assert.match(html, /SAE job offers[\s\S]*?To be agreed with HR|To be agreed with HR[\s\S]*?SAE job offers/u);
  assert.match(html, /New staff request/u);
  assert.doesNotMatch(html, /Lo pendiente|Peticiones por fase|Ofertas al SAE|Nueva petición/u);
});

test("?lang=en traduce marca y selector; volver a es conserva la ruta y el catálogo castellano", async () => {
  const html = await readFile(new URL("index.html", import.meta.url), "utf8");
  const claves = ["txt_gestion_de_recursos_humanos", "selector_idioma_etiqueta", "selector_idioma_es", "selector_idioma_en"];
  const nodos = new Map(claves.map((clave) => [clave, {
    textContent: "",
    getAttribute: () => clave,
  }]));
  for (const clave of claves) assert.match(html, new RegExp(`data-i18n-portal="${clave}"`, "u"));
  const documento = {
    documentElement: { lang: "es" },
    querySelectorAll: (selector) => selector === "[data-i18n-portal]" ? [...nodos.values()] : [],
  };
  aplicarIdiomaDocumento(documento);
  aplicarTextosPortal(documento);
  assert.equal(documento.documentElement.lang, "en");
  assert.deepEqual(claves.map((clave) => nodos.get(clave).textContent),
    ["Human Resources management", "Interface language", "Español", "English"]);
  assert.equal(traducirPortal("txt_portal_del_empleado"), "Employee Portal");
  assert.equal(traducirPortal("contratacion_temporal_encabezado"), "Temporary staff requests");
  assert.equal(traducirPortal("auditoria_expediente_panel"), "Case audit trail");
  assert.equal(traducirPortal("auditoria_expediente_accion"), "View this case’s audit trail");
  assert.equal(traducirPortal("contratacion_temporal_miga"), "Employee Portal → Temporary staff requests");
  assert.equal(traducirPortal("contratacion_temporal_titulo"), "Manage temporary staff requests");
  assert.equal(traducirPortal("plantillas_rrhh_nav"), "Document templates");
  assert.equal(traducirPortal("txt_modulos"), "Modules");
  assert.equal(traducirPortal("txt_modulos_del_portal"), "Portal modules");
  assert.equal(traducirPortal("txt_portal_de_recursos_humanos"), "Human Resources Portal");
  assert.equal(traducirPortal("txt_2026_diputacion_de_granada_portal_del_empleado"),
    "© 2026 Diputación de Granada · Employee Portal");
  assert.equal(traducirPortal("txt_proteccion_de_datos_accesibilidad_ayuda"),
    "Data protection · Accessibility · Help");

  let alCambiar;
  let destino;
  const selector = { value: "", addEventListener: (_tipo, escucha) => { alCambiar = escucha; } };
  const ubicacion = {
    href: "https://vec.example/portal-empleado/?lang=en&vista=ct#expedientes",
    assign: (url) => { destino = url; },
  };
  assert.equal(montarSelectorIdioma(selector, ubicacion), true);
  assert.equal(selector.value, "en");
  selector.value = "es";
  alCambiar();
  assert.equal(destino, "https://vec.example/portal-empleado/?lang=es&vista=ct#expedientes");
  assert.equal(cambiarIdioma("invalido", ubicacion), false);
  const es = crearTraductorPortal(MENSAJES_PORTAL_CASTELLANO);
  assert.equal(es("auditoria_expediente_panel"), "Auditoría del expediente");
  assert.equal(es("auditoria_expediente_accion"), "Consultar auditoría de este expediente");
  assert.equal(es("contratacion_temporal_titulo"), "Gestión de peticiones de personal temporal");
  assert.equal(es("plantillas_rrhh_nav"), "Plantillas de documentos");
  assert.equal(es("txt_modulos"), "Módulos");
  assert.equal(es("txt_proteccion_de_datos_accesibilidad_ayuda"), "Protección de datos · Accesibilidad · Ayuda");
  aplicarTextosPortal(documento, crearTraductorPortal(MENSAJES_PORTAL_CASTELLANO));
  assert.deepEqual(claves.map((clave) => nodos.get(clave).textContent),
    ["Gestión de Recursos Humanos", "Idioma de la interfaz", "Español", "Inglés"]);
});
