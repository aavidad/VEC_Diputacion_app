import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";

import { cargarHistorialMiBolsa, montarHistorialMiBolsa, renderizarHistorialMiBolsa, validarHistorialMiBolsa } from "./mi-bolsa-historial.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

const ahora = "2026-09-27T20:00:00.000000Z";
const comunes = { bolsa: "bolsa:prueba:1", categoria: "Auxiliar <sanitario>", ocurrido_en: "2026-09-26T10:00:00.000000Z" };
const contrato = () => ({ ...comunes, clase: "contrato_bolsa", tipo: "incorporacion", inicio: "2026-09-26T08:00:00.000000Z", fin_previsto: null, modalidad_clave: null, procedencia: "evento_ct_recibido" });
const llamamiento = () => ({ ...comunes, clase: "llamamiento", canal: "correo", resultado: "enviado" });
const renuncia = () => ({ ...comunes, clase: "renuncia", respuesta: "renuncia_justificada", modo: "propuesta_rrhh", estado: "propuesta_pendiente_rrhh" });
const pagina = (items = [], extra = {}) => ({ data: { esquema: "vec.bolsa.mi-bolsa.historial.v1", consultada_en: ahora,
  campos_visibles: ["contratos_propios", "llamamientos_propios", "renuncias_propias"], historial: { pagina: 1, tamano: 20, hay_mas: false, items }, ...extra } });

test("el contrato admite solo los tres tipos y campos expresamente visibles", () => {
  const original = pagina([contrato(), llamamiento(), renuncia()]);
  const datos = validarHistorialMiBolsa(original, 1);
  assert.equal(datos.historial.items.length, 3);
  assert.notEqual(datos, original.data);
  assert.throws(() => validarHistorialMiBolsa(pagina([{ ...contrato(), candidato_ref: "ajeno" }]), 1), /no autorizados/u);
  assert.throws(() => validarHistorialMiBolsa(pagina([renuncia()], { campos_visibles: ["contratos_propios"] }), 1), /no autorizada/u);
  assert.throws(() => validarHistorialMiBolsa(pagina([contrato()], { campos_visibles: ["renuncias_propias", "contratos_propios"] }), 1), /Campos visibles/u);
  assert.throws(() => validarHistorialMiBolsa(pagina([contrato()], { historial: { pagina: 2, tamano: 20, hay_mas: false, items: [contrato()] } }), 1), /Paginación/u);
  assert.throws(() => validarHistorialMiBolsa(pagina([contrato()], { historial: { pagina: 1, tamano: 20, hay_mas: true, items: [contrato()] } }), 1), /Paginación/u);
});

test("solo pide número de página; denegación y respuesta incompatible no presentan datos", async () => {
  const peticiones = [];
  const fetchImpl = async (url, opciones) => {
    peticiones.push({ url, opciones });
    return { status: 200, text: async () => JSON.stringify(pagina([contrato()])) };
  };
  assert.equal((await cargarHistorialMiBolsa({ fetchImpl })).historial.items.length, 1);
  assert.equal(peticiones[0].url, "/api/vec/bolsa/mi-bolsa/historial?pagina=1");
  assert.equal(peticiones[0].opciones.credentials, "same-origin");
  assert.equal(peticiones[0].opciones.cache, "no-store");
  assert.equal(peticiones[0].opciones.method, "GET");
  await assert.rejects(cargarHistorialMiBolsa({ pagina: 10001, fetchImpl }), /Página no válida/u);
  await assert.rejects(cargarHistorialMiBolsa({ fetchImpl: async () => ({ status: 403 }) }), (error) => error.codigo === "denegado");
  await assert.rejects(cargarHistorialMiBolsa({ fetchImpl: async () => ({ status: 200, text: async () => JSON.stringify(pagina([{ ...contrato(), nombre: "dato privado" }])) }) }), /no autorizados/u);
});

test("la vista distingue carga, vacío, campos ocultos, error y página; escapa contenido", () => {
  assert.match(renderizarHistorialMiBolsa(), /Cargando histórico autorizado/u);
  assert.match(renderizarHistorialMiBolsa(pagina().data, { estado: "correcto" }), /No constan actuaciones en el histórico de VEC/u);
  assert.match(renderizarHistorialMiBolsa(pagina([], { campos_visibles: [] }).data, { estado: "correcto" }), /no ha habilitado datos/u);
  assert.match(renderizarHistorialMiBolsa(null, { estado: "denegado" }), /No tiene permiso/u);
  const datos = validarHistorialMiBolsa(pagina([contrato(), llamamiento(), renuncia()]), 1);
  const html = renderizarHistorialMiBolsa(datos, { estado: "correcto" });
  assert.match(html, /Incorporación comunicada a Bolsa/u);
  assert.match(html, /Correo de llamamiento/u);
  assert.match(html, /Propuesta pendiente de RRHH/u);
  assert.match(html, /Auxiliar &lt;sanitario&gt;/u);
  assert.doesNotMatch(html, /Auxiliar <sanitario>|bolsa:prueba:1|candidato_ref|Firmado/u);
  assert.match(html, /pendiente de que RRHH la resuelva/u);
});

test("al salir cancela la lectura y una respuesta tardía no cambia otra vista", async () => {
  let completar;
  let signal;
  const eventos = new Map();
  const contenedor = { innerHTML: "", addEventListener: (nombre, fn) => eventos.set(nombre, fn), removeEventListener: (nombre) => eventos.delete(nombre) };
  const montaje = montarHistorialMiBolsa({ contenedor, fetchImpl: async (_url, opciones) => {
    signal = opciones.signal;
    return new Promise((resolver) => { completar = resolver; });
  } });
  assert.match(contenedor.innerHTML, /Cargando histórico autorizado/u);
  montaje.destruir();
  assert.equal(signal.aborted, true);
  assert.equal(eventos.size, 0);
  contenedor.innerHTML = "Otra vista";
  completar({ status: 200, text: async () => JSON.stringify(pagina([contrato()])) });
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(contenedor.innerHTML, "Otra vista");
});

test("la paginación consulta la página siguiente sin enviar referencias de participación", async () => {
  const rutas = [];
  const eventos = new Map();
  const contenedor = { innerHTML: "", addEventListener: (nombre, fn) => eventos.set(nombre, fn), removeEventListener: (nombre) => eventos.delete(nombre) };
  const montaje = montarHistorialMiBolsa({ contenedor, fetchImpl: async (ruta) => {
    rutas.push(ruta);
    const numero = Number(new URL(ruta, "https://vec.example").searchParams.get("pagina"));
    const items = numero === 1 ? Array.from({ length: 20 }, contrato) : [renuncia()];
    const respuesta = pagina(items, { historial: { pagina: numero, tamano: 20, hay_mas: numero === 1, items } });
    return { status: 200, text: async () => JSON.stringify(respuesta) };
  } });
  await new Promise((resolver) => setImmediate(resolver));
  assert.match(contenedor.innerHTML, /data-historial-accion="siguiente"/u);
  eventos.get("click")({ target: { closest: () => ({ dataset: { historialAccion: "siguiente" } }) } });
  await new Promise((resolver) => setImmediate(resolver));
  assert.deepEqual(rutas, ["/api/vec/bolsa/mi-bolsa/historial?pagina=1", "/api/vec/bolsa/mi-bolsa/historial?pagina=2"]);
  assert.match(contenedor.innerHTML, /Página 2[\s\S]*data-historial-accion="siguiente" disabled/u);
  montaje.destruir();
});

test("los textos nuevos están en el catálogo común", async () => {
  const catalogo = await catalogoPlano("es");
  for (const clave of ["titulo", "cargando", "vacio", "sinCampos", "denegado", "limite", "contrato", "llamamiento", "renuncia"]) {
    assert.equal(typeof catalogo[`areaPersonal.miBolsa.historial.${clave}`], "string");
  }
});

test("acepta la respuesta vacía que devuelve el servidor con sus nombres de campo", () => {
  // Cuerpo literal de GET /api/vec/bolsa/mi-bolsa/historial en cidonia (09/10/2026).
  const real = JSON.parse('{"data":{"campos_visibles":["contratos_propios","llamamientos_propios","renuncias_propias"],"consultada_en":"2026-10-09T03:52:16.452772Z","esquema":"vec.bolsa.mi-bolsa.historial.v1","historial":{"hay_mas":false,"items":[],"pagina":1,"tamano":20}}}');
  const datos = validarHistorialMiBolsa(real, 1);
  assert.deepEqual(datos.campos_visibles, ["contratos_propios", "llamamientos_propios", "renuncias_propias"]);
  assert.doesNotMatch(renderizarHistorialMiBolsa(datos, { estado: "correcto" }), /todavía no está disponible/u);
  assert.throws(() => validarHistorialMiBolsa({ data: { ...real.data, campos_visibles: ["contratos", "llamamientos", "renuncias"] } }, 1), /Campos visibles/u);
});
