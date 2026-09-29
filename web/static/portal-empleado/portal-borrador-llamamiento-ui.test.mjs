import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearSuperficieBorradorLlamamiento } from "./portal-borrador-llamamiento-ui.js";
import { MENSAJES_BORRADORES_EN, MENSAJES_BORRADORES_ES } from "./portal-borradores-i18n.js";

const ref = `borrador-llamamiento:alta:${"b".repeat(64)}`;
const recibido = { borrador_ref: ref, estado: "borrador_interno", version: "1", resumen: "Preparar cobertura interna", recibo_ref: `recibo:${"b".repeat(64)}`, registrado_en: "2026-09-21T10:30:00Z", reintento_idempotente: false };
const portal = await readFile(new URL("./portal.js", import.meta.url), "utf8");

test("el montaje de llamamientos incorpora el corte y lo desmonta al salir", () => {
  assert.match(portal, /crearSuperficieBorradorLlamamiento/);
  assert.match(portal, /vista === "llamamientos"[\s\S]{0,200}superficieBorradorLlamamiento\.desmontar/);
  assert.match(portal, /if \(vista === "llamamientos"\) \{ superficieBorradorLlamamiento\.activar\(\);[\s\S]{0,160}superficieBorradorLlamamiento\.renderizar/);
});

test("la superficie muestra un formulario acotado y no afirma selección ni contacto", () => {
  const superficie = crearSuperficieBorradorLlamamiento({ crearClienteImpl: () => ({ generarClave: () => "blam-12345678", crear: async () => recibido }) });
  const html = superficie.renderizar();
  assert.match(html, /Preparar borrador interno/);
  assert.match(html, /data-borrador-llamamiento-form/);
  assert.match(html, /No incluya datos personales/);
  assert.match(html, /No selecciona personas ni realiza contactos/);
});

test("conserva la misma clave ante fallo incierto y confirma solo tras recibo", async () => {
  const claves = []; let intentos = 0;
  const superficie = crearSuperficieBorradorLlamamiento({ crearClienteImpl: () => ({ generarClave: () => "blam-12345678", crear: async ({ claveIdempotencia }) => { claves.push(claveIdempotencia); intentos += 1; if (intentos === 1) throw new Error("red caída"); return recibido; } }) });
  assert.equal(await superficie.manejarEnvio({ resumen: "Preparar cobertura interna" }), false);
  assert.match(superficie.renderizar(), /Puede reintentar el mismo contenido/);
  assert.equal(await superficie.manejarEnvio({ resumen: "Preparar cobertura interna" }), true);
  assert.deepEqual(claves, ["blam-12345678", "blam-12345678"]);
  assert.match(superficie.renderizar(), new RegExp(ref));
});

test("recupera un borrador por referencia opaca sin listado ni estado persistente", async () => {
  const consultas = [];
  const superficie = crearSuperficieBorradorLlamamiento({ crearClienteImpl: () => ({ generarClave: () => "blam-12345678", crear: async () => recibido, consultar: async ({ referencia }) => { consultas.push(referencia); return recibido; } }) });
  assert.match(superficie.renderizar(), /Recuperar borrador por referencia/);
  assert.equal(await superficie.manejarConsulta({ referencia: ref }), true);
  assert.deepEqual(consultas, [ref]);
  assert.match(superficie.renderizar(), new RegExp(ref));
});

test("un fallo al recuperar no ofrece reintentar el contenido de otra creación", async () => {
  const superficie = crearSuperficieBorradorLlamamiento({ crearClienteImpl: () => ({ generarClave: () => "blam-12345678", crear: async () => { throw new Error("red"); }, consultar: async () => { throw new Error("lectura"); } }) });
  await superficie.manejarEnvio({ resumen: "Preparar cobertura interna" });
  await superficie.manejarConsulta({ referencia: ref });
  assert.doesNotMatch(superficie.renderizar(), /Puede reintentar el mismo contenido/);
});

test("el catálogo bilingüe conserva las mismas claves y el escape seguro", async () => {
  assert.deepEqual(Object.keys(MENSAJES_BORRADORES_EN), Object.keys(MENSAJES_BORRADORES_ES));
  const traducir = (clave, variables = {}) => MENSAJES_BORRADORES_EN[`bl_${clave}`].replace(/\{([a-z_]+)\}/g, (_, nombre) => String(variables[nombre] ?? ""));
  const superficie = crearSuperficieBorradorLlamamiento({ traducir, crearClienteImpl: () => ({ generarClave: () => "blam-12345678", crear: async () => { throw new Error("red"); }, consultar: async () => recibido }) });
  assert.match(superficie.renderizar(), /Prepare an internal draft/);
  assert.match(superficie.renderizar(), /does not select people or contact anyone/);
  await superficie.manejarEnvio({ resumen: "<script>" });
  assert.equal(superficie.estado().enviando, false);
  assert.match(superficie.renderizar(), /The draft could not be prepared/);
  assert.doesNotMatch(superficie.renderizar(), /<script>/);
});

test("desmontar aborta, ignora respuesta tardía y el remonte permite una nueva operación", async () => {
  let resolver; let abortada = false; let intentos = 0;
  const superficie = crearSuperficieBorradorLlamamiento({ crearClienteImpl: () => ({ generarClave: () => "blam-12345678", crear: ({ signal }) => { intentos += 1; if (intentos === 2) return Promise.resolve(recibido); return new Promise((resolve) => { resolver = resolve; signal.addEventListener("abort", () => { abortada = true; }, { once: true }); }); }, consultar: async () => recibido }) });
  const promesa = superficie.manejarEnvio({ resumen: "Preparar cobertura interna" });
  superficie.desmontar(); resolver(recibido);
  assert.equal(await promesa, false);
  assert.equal(abortada, true);
  assert.equal(superficie.estado().recibo, null);
  assert.equal(superficie.estado().enviando, false);
  superficie.activar();
  assert.equal(await superficie.manejarEnvio({ resumen: "Preparar cobertura interna" }), true);
  assert.equal(intentos, 2);
});
