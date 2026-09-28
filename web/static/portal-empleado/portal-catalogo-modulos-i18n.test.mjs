import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// El selector común fija el idioma al cargar el módulo, como en el navegador.
globalThis.location = { href: "https://vec.example/portal-empleado/?lang=en" };
const { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } = await import("../comun/idioma.js");
const { traducirPortal, formatearNumeroPortal } = await import("./portal-i18n.js");
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
    obtenerMetricasCuadro: () => ({ en_tramitacion: 0, con_incidencia: 0, en_llamamiento: 0 }),
    obtenerTramitesInicio: () => [],
  })();
  assert.match(html, /<h1>Temporary staff requests<\/h1>/u);
  assert.match(html, /data-accion="ayuda" aria-label="Help">\?<\/button>/u);
  assert.match(html, /No recent cases\. Open the dashboard/u);
  assert.match(html, /No permission for this profile|Your session does not have permission/u);
  assert.match(html, /SAE offers cannot be viewed yet/u);
  assert.doesNotMatch(html, /No hay expedientes recientes|Peticiones de personal temporal|Las ofertas al SAE/u);
});
