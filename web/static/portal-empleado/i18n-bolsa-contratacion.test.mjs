import assert from "node:assert/strict";
import { readdir, readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_PORTAL_ES } from "./portal-i18n.js";
import { MENSAJES_CONTRATACION_TEMPORAL_ES } from "./modulos/contratacion-temporal/i18n.js";
import { aplicarIdiomaDocumento, aplicarTextosPortal, instalarValidacionI18n, mensajeValidacionPortal } from "./portal-idioma.js";
import { cadenasHumanas, hallazgosHTML, hallazgosTextosLiterales } from "./textos-literales.test-helper.mjs";

// Todo texto visible de Bolsa y Contratación temporal sale de un catálogo i18n:
// etiquetas, bocadillos (`title`), `aria-label`, `placeholder`, `alt`, estados,
// errores y confirmaciones. Esta prueba recorre las vistas y falla si aparece un
// literal nuevo fuera de un catálogo.

const raiz = new URL("./", import.meta.url);
const BOLSA = /^(?:portal-(?:bolsas|borrador|borradores|llamamientos|panel-interno|menu-bolsa|vistas|inicio|justificante)[\w-]*|portal|ayuda-contenido|ayudante-tramites|estado-entrega|portal-idioma)\.js$/u;
const CARPETAS = ["peticiones-centro", "reglas", "modulos/contratacion-temporal"];
// Catálogos (su contenido es el texto traducible) y ficheros que no son interfaz.
const EXCLUIDOS = /(?:^|\/)(?:[\w-]*i18n[\w-]*\.js|datos-presentacion[\w-]*\.js|formulario-llamamiento-pruebas\.js)$/u;

async function ficherosAuditados() {
  const lista = (await readdir(raiz)).filter((nombre) => BOLSA.test(nombre));
  for (const carpeta of CARPETAS) {
    for (const nombre of await readdir(new URL(`${carpeta}/`, raiz))) {
      if (nombre.endsWith(".js") && !nombre.includes(".test.")) lista.push(`${carpeta}/${nombre}`);
    }
  }
  return lista.filter((ruta) => !EXCLUIDOS.test(ruta)).sort();
}

const PAGINAS = ["index.html", "peticiones-centro/index.html", "reglas/index.html", "modulos/contratacion-temporal/consulta-seguimiento.html"];

// Cadenas con aspecto de texto que no se muestran como tal. Cada entrada se
// justifica; una cadena nueva visible debe ir al catálogo, no a esta lista.
const PERMITIDAS = new Map([
  // Valores centinela de filtros que se comparan con datos, no rótulos.
  ["portal.js", ["Todos", "Todas"]],
  // Modalidad del llamamiento: es el valor que se envía al servidor; su rótulo sale de `panel_b7_modalidad_*`.
  ["portal-panel-interno.js", ["Sustitución", "Vacante", "Programa temporal", "Acumulación de tareas"]],
  // Estado textual devuelto por el servidor con el que se compara.
  ["portal-inicio.js", ["en curso"]],
  // Cabecera HTTP y error de programación por opciones internas mal formadas.
  ["portal-borradores-api.js", ["Accept", "  no respeta el contrato del cliente."]],
  // Mensaje interno: la vista muestra los errores por campo de `errores`, ya traducidos.
  ["modulos/contratacion-temporal/contrato.js", ["La solicitud no respeta el contrato de alta"]],
  ["modulos/contratacion-temporal/cliente-http-transporte.js", ["Accept", "respuesta descartada", "operación cancelada"]],
  // Motivo interno de error de programación (el usuario ve el mensaje del catálogo).
  ["modulos/contratacion-temporal/cliente-http.js", ["cliente HTTP de contratación temporal:  "]],
  // Nombres de campos para errores internos de validación de contratos.
  ["modulos/contratacion-temporal/vista-expedientes-analisis.js", ["composición del análisis", "contexto de composición del análisis", "composición de rectificación del análisis"]],
  ["modulos/contratacion-temporal/formulario-analisis.js", ["catálogos del análisis", "motivos de rectificación", "grupos o subgrupos"]],
  // Segundo argumento de `t(clave, respaldo)`: la clave existe en el catálogo.
  ["modulos/contratacion-temporal/adaptador-http-expedientes.js", ["Observaciones"]],
]);
// En los contratos, las cadenas en minúscula nombran campos en errores internos.
const CONTRATO = /(?:^|\/)(?:contrato[\w-]*|[\w-]*-contrato)\.js$/u;
const DATO_CONTRATO = "Provisional, pendiente de RRHH (dudas 13–14)";

test("las vistas de Bolsa y Contratación temporal no escriben texto visible ni atributos fuera del catálogo", async () => {
  const ficheros = await ficherosAuditados();
  assert.ok(ficheros.length > 100, "la auditoría debe cubrir las vistas de ambos módulos");
  const hallazgos = [];
  for (const ruta of ficheros) {
    const fuente = await readFile(new URL(ruta, raiz), "utf8");
    for (const h of hallazgosTextosLiterales(fuente)) hallazgos.push(`${ruta}:${h.linea} [${h.clase}] ${h.texto}`);
  }
  assert.deepEqual(hallazgos, []);
});

test("las vistas no dejan mensajes, estados ni rótulos en castellano fuera del catálogo", async () => {
  const hallazgos = [];
  for (const ruta of await ficherosAuditados()) {
    const fuente = await readFile(new URL(ruta, raiz), "utf8");
    const permitidas = PERMITIDAS.get(ruta) || [];
    for (const h of cadenasHumanas(fuente)) {
      const texto = h.literal.texto.replaceAll("\u0001", " ");
      if (permitidas.includes(texto) || permitidas.includes(h.texto)) continue;
      if (CONTRATO.test(ruta) && (/^[a-záéíóúñ]/u.test(texto) || texto === DATO_CONTRATO)) continue;
      hallazgos.push(`${ruta}:${h.linea} ${h.texto}`);
    }
  }
  assert.deepEqual(hallazgos, []);
});

test("las páginas estáticas declaran la clave de cada texto y atributo visible", async () => {
  const hallazgos = [];
  for (const ruta of PAGINAS) {
    for (const h of hallazgosHTML(await readFile(new URL(ruta, raiz), "utf8"))) hallazgos.push(`${ruta}:${h.linea} [${h.clase}] ${h.texto}`);
  }
  assert.deepEqual(hallazgos, []);
});

test("toda clave usada en las vistas existe en su catálogo", async () => {
  const faltan = [];
  const rutas = [...await ficherosAuditados(), ...PAGINAS];
  for (const ruta of rutas) {
    const fuente = await readFile(new URL(ruta, raiz), "utf8");
    for (const [, clave] of fuente.matchAll(/"(txt_[a-z0-9_]+)"/gu)) if (!Object.hasOwn(MENSAJES_PORTAL_ES, clave)) faltan.push(`${ruta}: ${clave}`);
    for (const [, clave] of fuente.matchAll(/"(ct_txt_[a-z0-9_]+)"/gu)) if (!Object.hasOwn(MENSAJES_CONTRATACION_TEMPORAL_ES, clave)) faltan.push(`${ruta}: ${clave}`);
  }
  assert.deepEqual(faltan, []);
});

test("el analizador detecta bocadillos, etiquetas y textos literales", () => {
  const clases = (fuente) => hallazgosTextosLiterales(fuente).map((h) => h.clase);
  assert.deepEqual(clases('const a = `<div title="Please fill out">${x}</div>`;'), ["atributo"]);
  assert.deepEqual(clases('const a = `<button aria-label="Ver ${n} filas">${n}</button>`;'), ["atributo"]);
  assert.deepEqual(clases('const a = `<p>Cargando…</p>`;'), ["texto"]);
  assert.deepEqual(clases('boton.title = "Guardar";'), ["propiedad"]);
  assert.deepEqual(clases('window.confirm("¿Seguro?");'), ["propiedad"]);
  assert.deepEqual(clases('const a = `<p title="${t("k")}">${t("k")}</p>`;'), []);
  assert.deepEqual(clases('const MENSAJES_X_ES = Object.freeze({ a: "<p>Texto</p>" });'), []);
  assert.deepEqual(hallazgosHTML('<button title="Avisos">x</button>').map((h) => h.clase), ["atributo"]);
  assert.deepEqual(hallazgosHTML('<button title="Avisos" data-i18n-portal-title="k">x</button>'), []);
});

function control(validez, propiedades = {}) {
  let mensajePropio = "";
  const elemento = {
    type: "text", tagName: "TEXTAREA", minLength: 3, maxLength: 20, min: "", max: "", ...propiedades,
    get validity() { return { valid: false, customError: mensajePropio !== "", ...validez }; },
    get validationMessage() { return mensajePropio || "Please fill out this field."; },
    setCustomValidity(mensaje) { mensajePropio = mensaje; },
  };
  return elemento;
}

test("la validación nativa se muestra con los textos del portal", () => {
  assert.equal(mensajeValidacionPortal(control({ valueMissing: true })), "Rellene este campo.");
  assert.equal(mensajeValidacionPortal(control({ valueMissing: true }, { type: "checkbox", tagName: "INPUT" })), "Marque esta casilla para continuar.");
  assert.equal(mensajeValidacionPortal(control({ valueMissing: true }, { tagName: "SELECT", type: "select-one" })), "Seleccione una opción.");
  assert.equal(mensajeValidacionPortal(control({ tooShort: true })), "Escriba al menos 3 caracteres.");
  assert.equal(mensajeValidacionPortal(control({ typeMismatch: true }, { type: "email", tagName: "INPUT" })), "Introduzca una dirección de correo válida.");
  assert.equal(mensajeValidacionPortal({ validity: { valid: true } }), "");
});

test("el evento invalid recibe el mensaje traducido y se retira al corregir", () => {
  const oyentes = new Map();
  const documento = { addEventListener: (tipo, fn) => oyentes.set(tipo, fn), removeEventListener: (tipo) => oyentes.delete(tipo) };
  const retirar = instalarValidacionI18n(documento);
  assert.equal(instalarValidacionI18n(documento), retirar, "la instalación es idempotente");
  const campo = control({ valueMissing: true });
  oyentes.get("invalid")({ target: campo });
  assert.equal(campo.validationMessage, "Rellene este campo.");
  oyentes.get("input")({ target: campo });
  assert.equal(campo.validationMessage, "Please fill out this field.", "al escribir se retira el mensaje propio");
  // Un mensaje fijado por el propio módulo se respeta.
  const propio = control({});
  propio.setCustomValidity("Referencia no válida");
  oyentes.get("invalid")({ target: propio });
  assert.equal(propio.validationMessage, "Referencia no válida");
  retirar();
  assert.equal(oyentes.size, 0);
});

test("el documento declara el idioma del portal y traduce sus textos estáticos", () => {
  const raizDocumento = { lang: "en" };
  aplicarIdiomaDocumento({ documentElement: raizDocumento });
  assert.equal(raizDocumento.lang, "es");
  const nodos = {
    "[data-i18n-portal]": [{ getAttribute: () => "txt_saltar_al_contenido_principal", textContent: "" }],
    "[data-i18n-portal-title]": [{ getAttribute: () => "txt_avisos", setAttribute(nombre, valor) { this[nombre] = valor; } }],
  };
  aplicarTextosPortal({ querySelectorAll: (selector) => nodos[selector] || [] });
  assert.equal(nodos["[data-i18n-portal]"][0].textContent, "Saltar al contenido principal");
  assert.equal(nodos["[data-i18n-portal-title]"][0].title, "Avisos");
});
