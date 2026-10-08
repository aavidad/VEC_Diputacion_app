import assert from "node:assert/strict";
import test from "node:test";
import { crearIntegracionPreferenciasPortal } from "./portal-preferencias-integracion.js";

const valores = Object.freeze({ idioma: "es", tamano_texto: "normal", alto_contraste: false,
  tema: "sistema", inicio: "cuadro", filas: 20, aviso_correo_tareas: false, aviso_correo_plazos: false });
const opciones = (nombre, codigos) => codigos.map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.${nombre}.${codigo}` }));
const catalogo = Object.freeze({ version_ref: "usuarios-preferencias-v1",
  idiomas: opciones("idioma", ["navegador", "es", "en"]),
  tamanos_texto: opciones("tamano_texto", ["normal", "grande", "muy_grande"]),
  temas: opciones("tema", ["sistema", "claro", "oscuro"]),
  inicios: opciones("inicio", ["cuadro", "peticiones", "bolsas"]),
  filas: [20, 50, 100], predeterminados: valores });

function entorno(url, eventos) {
  const documento = { documentElement: { lang: "en", dataset: {} }, body: { dataset: {} } };
  const ventana = { location: { href: url, replace(destino) { eventos.push(["redirigir", destino]); } },
    history: { replaceState(_estado, _titulo, destino) { eventos.push(["limpiar_lang", destino]); } },
    navigator: { languages: ["en-US"] }, matchMedia: () => ({ matches: false, addEventListener() {}, removeEventListener() {} }) };
  const integracion = crearIntegracionPreferenciasPortal({ documento, ventana,
    porId: () => null, estado: { vista: "contratacion-temporal" }, renderizar() {}, aplicarFilas() {},
    navegar() {}, vistaBolsaDisponible: () => false, altaCTDisponible: () => false,
    anunciar() {}, traducir: (clave) => clave });
  return { integracion, eventos };
}

function respuesta(estado, idioma = "es") {
  const cuerpo = estado === 200 ? { data: { catalogo, estado: { version: 0,
    catalogo_version_ref: catalogo.version_ref, valores: { ...valores, idioma } } } } : {};
  return new Response(JSON.stringify(cuerpo), { status: estado,
    headers: { "Content-Type": "application/json" } });
}

// Misma arista que consume la raíz: primer paint, arranque de preferencias,
// espera condicional y carga de módulos solo en el documento que permanece.
async function arrancar(integracion, eventos) {
  eventos.push(["pintar"]);
  const cargaPreferencias = integracion.superficie.cargar();
  const esperarIdioma = integracion.prepararInicio();
  if (esperarIdioma) {
    await cargaPreferencias;
    if (integracion.cambioIdiomaPendiente()) return;
  }
  eventos.push(["cargar_modulos"]);
  await cargaPreferencias;
}

for (const [nombre, url, estado, idioma, espera, redirige] of [
  ["idioma distinto", "http://localhost/portal-empleado/#contratacion-temporal", 200, "es", true, true],
  ["idioma igual", "http://localhost/portal-empleado/#contratacion-temporal", 200, "en", true, false],
  ["preferencias 404", "http://localhost/portal-empleado/#contratacion-temporal", 404, "es", true, false],
  ["preferencias 503", "http://localhost/portal-empleado/#contratacion-temporal", 503, "es", true, false],
  ["idioma explícito", "http://localhost/portal-empleado/?lang=en#contratacion-temporal", 200, "es", false, false],
]) {
  test(`arranque con ${nombre}`, async () => {
    const anterior = globalThis.fetch;
    const eventos = [];
    let resolver;
    let llamadas = 0;
    globalThis.fetch = () => {
      llamadas += 1;
      eventos.push(["consultar_preferencias"]);
      if (estado === 503 && llamadas > 1) return Promise.resolve(respuesta(503));
      return new Promise((resolve) => { resolver = resolve; });
    };
    try {
      const { integracion } = entorno(url, eventos);
      assert.equal(integracion.prepararInicio(), espera);
      const arranque = arrancar(integracion, eventos);
      await Promise.resolve();
      assert.equal(eventos[0][0], "pintar");
      assert.equal(eventos.filter(([evento]) => evento === "cargar_modulos").length, espera ? 0 : 1);
      await new Promise((resolve) => setTimeout(resolve, 200));
      eventos.push(["respuesta_preferencias"]);
      resolver(respuesta(estado, idioma));
      await arranque;
      assert.equal(integracion.cambioIdiomaPendiente(), redirige);
      assert.equal(eventos.filter(([evento]) => evento === "redirigir").length, redirige ? 1 : 0);
      assert.equal(eventos.filter(([evento]) => evento === "cargar_modulos").length, redirige ? 0 : 1);
      if (espera && !redirige) assert.ok(eventos.findIndex(([evento]) => evento === "cargar_modulos")
        > eventos.findIndex(([evento]) => evento === "respuesta_preferencias"));
    } finally { globalThis.fetch = anterior; }
  });
}
