import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearTextos } from "../../../../comun/textos.js";
import {
  bytesABase64, comprobarFichero, crearClienteCargaConvoca, ErrorCargaConvoca, MAXIMO_FICHERO,
  RUTA_CONFIRMAR, RUTA_VISTA_PREVIA, validarRecibo, validarVistaPrevia,
} from "./cliente.js";
import {
  claveCampo, claveCategoria, escribirEstadoRuta, filtroControl, filtroServidor, leerEstadoRuta,
  nombrePersona, paginaServidor, textoBloqueo, textoError, textoIncidencia,
} from "./modelo.js";

const leerCatalogo = async (idioma) => JSON.parse(await readFile(new URL(`../../../../textos/${idioma}/bolsa-carga-convoca.json`, import.meta.url), "utf8"));

async function textosDe(idioma) {
  const respaldo = await leerCatalogo("es");
  const propio = idioma === "es" ? null : await leerCatalogo(idioma);
  return crearTextos({ modulo: "bolsa-carga-convoca", idioma, localizacion: idioma === "es" ? "es-ES" : "en-GB", respaldo, propio,
    avisar: (mensaje) => { throw new Error(mensaje); } });
}

const huella = "a".repeat(64);
const vista = () => ({ data: {
  esquema: "vec.bolsa.rrhh.carga_convoca.vista_previa.v1", huella_sha256: huella, nombre_fichero: "bolsa.xlsx",
  esquema_fichero: "convoca_resumen_persona_v1", filas_leidas: 3, aceptadas: 2, rechazadas: 1, con_avisos: 1, bloqueo: "",
  filtro: "todas", limite: 50, desplazamiento: 0, total_filtrado: 3,
  filas: [
    { numero: 2, estado: "aceptada", posicion: 1, documento: "***4521**", primer_apellido: "Reyes", segundo_apellido: "Álvarez", nombre: "Antonio", total: "15.75", errores: [], avisos: [] },
    { numero: 3, estado: "aceptada", posicion: 2, documento: "***2218**", primer_apellido: "García", nombre: "María", total: "10", errores: [], avisos: ["identidad_ambigua"] },
    { numero: 4, estado: "rechazada", errores: [{ campo: "Total", codigo: "total_incoherente" }], avisos: [] },
  ],
} });
const recibo = () => ({ data: {
  esquema: "vec.bolsa.rrhh.carga_convoca.recibo.v1", bolsa_ref: "bolsa:aux:2026-10-05", version_bolsa: 1, acta_ref: "acta:importacion-convoca:" + huella,
  huella_sha256: huella, reutilizada: false, acta_reutilizada: false, filas_cargadas: 2, filas_excluidas: 1,
  pendientes_revision: [{ fila: 3, motivo: "identidad_ambigua" }], sustituye_a: [], auditoria_ref: "aud_v3_" + "0".repeat(32),
  confirmada_en: "2026-10-05T10:00:00.000000Z",
} });
const respuestaJSON = (cuerpo, status = 200) => new Response(JSON.stringify(cuerpo), { status, headers: { "content-type": "application/json; charset=utf-8" } });

test("los catálogos de español e inglés tienen las mismas claves y no les falta ninguna", async () => {
  for (const idioma of ["es", "en"]) await textosDe(idioma);
  const claves = (objeto, ruta = "") => Object.entries(objeto).flatMap(([k, v]) =>
    typeof v === "object" && !("other" in v) ? claves(v, `${ruta}${k}.`) : [`${ruta}${k}`]);
  assert.deepEqual(claves(await leerCatalogo("en")).sort(), claves(await leerCatalogo("es")).sort());
});

test("el HTML solo usa claves que existen en el catálogo", async () => {
  const html = await readFile(new URL("./index.html", import.meta.url), "utf8");
  const general = (await leerCatalogo("es")).general;
  for (const [, clave] of html.matchAll(/data-i18n(?:-label)?="([^"]+)"/gu)) assert.ok(Object.hasOwn(general, clave), clave);
  assert.doesNotMatch(html, /localStorage|sessionStorage|document\.cookie/u);
});

test("comprueba el fichero antes de leerlo", () => {
  assert.equal(comprobarFichero(undefined), "faltaFichero");
  assert.equal(comprobarFichero({ name: "lista.csv", size: 10 }), "ficheroTipo");
  assert.equal(comprobarFichero({ name: "lista.xlsx", size: MAXIMO_FICHERO + 1 }), "ficheroGrande");
  assert.equal(comprobarFichero({ name: "lista.xlsx", size: 0 }), "ficheroGrande");
  assert.equal(comprobarFichero({ name: "Lista.XLS", size: 10 }), "");
  assert.equal(bytesABase64(new Uint8Array([0, 255, 16])), "AP8Q");
  const grande = new Uint8Array(100000).fill(65);
  assert.equal(Buffer.from(bytesABase64(grande), "base64").length, 100000);
});

test("envía la vista previa y la confirmación por POST JSON sin credenciales propias", async () => {
  const llamadas = [];
  const cliente = crearClienteCargaConvoca({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    return respuestaJSON(ruta === RUTA_VISTA_PREVIA ? vista() : recibo(), ruta === RUTA_CONFIRMAR ? 201 : 200);
  } });
  const v = await cliente.previsualizar({ nombre: "bolsa.xlsx", base64: "AAAA" });
  assert.equal(v.filas.length, 3);
  const r = await cliente.confirmar({ nombre: "bolsa.xlsx", base64: "AAAA", categoria: "auxiliar", excluir: true,
    filtro: "rechazadas", limite: 1, desplazamiento: 50 });
  assert.equal(r.filas_cargadas, 2);
  for (const [, opciones] of llamadas) {
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.cache, "no-store");
    assert.deepEqual(Object.keys(opciones.headers).sort(), ["Accept", "Content-Type"]);
  }
  assert.deepEqual(JSON.parse(llamadas[0][1].body), { nombre_fichero: "bolsa.xlsx", contenido_base64: "AAAA",
    filtro: "todas", limite: 50, desplazamiento: 0 });
  assert.deepEqual(JSON.parse(llamadas[1][1].body), { nombre_fichero: "bolsa.xlsx", contenido_base64: "AAAA", categoria: "auxiliar", excluir_filas_con_errores: true });
});

test("pide una página exacta y rechaza una respuesta que suplante filtro o posición", async () => {
  const peticiones = [];
  const cliente = crearClienteCargaConvoca({ fetchImpl: async (_ruta, opciones) => {
    const solicitud = JSON.parse(opciones.body);
    peticiones.push(solicitud);
    return respuestaJSON({ data: { ...vista().data, filtro: "rechazadas", limite: 1,
      desplazamiento: 0, total_filtrado: 1, filas: [vista().data.filas[2]] } });
  } });
  const pagina = await cliente.previsualizar({ nombre: "bolsa.xlsx", base64: "AAAA", filtro: "rechazadas", limite: 1, desplazamiento: 0 });
  assert.equal(pagina.filas.length, 1);
  assert.equal(peticiones[0].filtro, "rechazadas");
  assert.equal(peticiones[0].limite, 1);
  assert.equal(peticiones[0].desplazamiento, 0);
  await assert.rejects(cliente.previsualizar({ nombre: "bolsa.xlsx", base64: "AAAA", filtro: "todas" }), /página de vista previa incompatible/u);
  await assert.rejects(cliente.previsualizar({ nombre: "bolsa.xlsx", base64: "AAAA", filtro: "otro" }), TypeError);
});

test("traduce los fallos a mensajes en llano sin códigos", async () => {
  const textos = await textosDe("es");
  const cliente = crearClienteCargaConvoca({ fetchImpl: async () => respuestaJSON({ error: { codigo: "acceso_denegado" } }, 403) });
  await assert.rejects(cliente.previsualizar({ nombre: "a.xlsx", base64: "AAAA" }), (e) => e instanceof ErrorCargaConvoca && e.estado === 403 && e.codigo === "acceso_denegado");
  assert.match(textoError(textos, new ErrorCargaConvoca(403, "acceso_denegado")), /perfil no permite/u);
  assert.match(textoError(textos, new ErrorCargaConvoca(502, "codigo_que_no_existe")), /avise a Informática/u);
  assert.match(textoError(textos, new TypeError("red")), /avise a Informática/u);
  for (const mensaje of [textoError(textos, new ErrorCargaConvoca(500, "x"))]) assert.doesNotMatch(mensaje, /\b[45]\d\d\b|error_|_/u);
});

test("rechaza respuestas con otra forma", () => {
  assert.throws(() => validarVistaPrevia({ data: { ...vista().data, esquema: "otro" } }));
  assert.throws(() => validarVistaPrevia({ data: { ...vista().data, filas: [{ numero: -1, estado: "aceptada", errores: [], avisos: [] }] } }));
  assert.throws(() => validarVistaPrevia({ data: { ...vista().data, total_filtrado: 2 } }));
  assert.throws(() => validarVistaPrevia({ data: { ...vista().data, limite: 0 } }));
  assert.throws(() => validarVistaPrevia({ data: { ...vista().data, filas: [] } }));
  assert.throws(() => validarVistaPrevia({ data: { ...vista().data, filtro: "rechazadas" } }));
  assert.equal(validarVistaPrevia({ data: { ...vista().data, filtro: "rechazadas", total_filtrado: 1,
    filas: [vista().data.filas[2]] } }).filas.length, 1);
  assert.throws(() => validarRecibo({ data: { ...recibo().data, auditoria_ref: 7 } }));
  for (const cambio of [
    { bolsa_ref: "" }, { version_bolsa: 0 }, { acta_ref: "" }, { auditoria_ref: "" },
    { filas_cargadas: 0 }, { confirmada_en: "ayer" },
    { sustituye_a: ["bolsa:anterior"] },
    { sustituye_a: [{ bolsa_ref: "bolsa:anterior", version_bolsa: 0 }] },
    { pendientes_revision: [{ fila: 0, motivo: "identidad_ambigua" }] },
  ]) assert.throws(() => validarRecibo({ data: { ...recibo().data, ...cambio } }));
});

test("un POST aceptado con recibo incompatible exige comprobar la bolsa antes de repetir", async () => {
  const textos = await textosDe("es");
  const cliente = crearClienteCargaConvoca({ fetchImpl: async () => respuestaJSON({ data: { ...recibo().data, bolsa_ref: "" } }, 201) });
  await assert.rejects(cliente.confirmar({ nombre: "bolsa.xlsx", base64: "AAAA", categoria: "auxiliar" }),
    (error) => error instanceof ErrorCargaConvoca && error.codigo === "recibo_incoherente");
  assert.match(textoError(textos, new ErrorCargaConvoca(0, "recibo_incoherente")), /Consulte las bolsas/u);
});

test("el plazo de confirmación puede vencer sin respuesta de un POST ya enviado", async () => {
  const cliente = crearClienteCargaConvoca({ plazoMs: 1, fetchImpl: async (_ruta, opciones) =>
    new Promise((_resolver, rechazar) => opciones.signal.addEventListener("abort", () =>
      rechazar(new DOMException("", "AbortError")), { once: true })) });
  await assert.rejects(cliente.confirmar({ nombre: "bolsa.xlsx", base64: "AAAA", categoria: "auxiliar" }),
    (error) => error.name === "AbortError");
});

test("la página del servidor y la URL conservan solo filtro, página y otros parámetros", async () => {
  const textos = await textosDe("es");
  const filas = validarVistaPrevia(vista()).filas;
  assert.deepEqual(paginaServidor({ limite: 50, desplazamiento: 50, total_filtrado: 117 }), { pagina: 2, total: 3 });
  assert.equal(filtroServidor("errores"), "rechazadas");
  assert.equal(filtroServidor("avisos"), "con_avisos");
  assert.equal(filtroControl("con_avisos"), "avisos");
  assert.deepEqual(leerEstadoRuta("?lang=en&x=1&convoca_estado=con_avisos&convoca_pagina=3"), { filtro: "con_avisos", pagina: 3 });
  assert.deepEqual(leerEstadoRuta("?convoca_estado=otra&convoca_pagina=999999"), { filtro: "todas", pagina: 1 });
  assert.equal(escribirEstadoRuta("?lang=en&x=1&convoca_estado=aceptadas", "con_avisos", 3),
    "?lang=en&x=1&convoca_estado=con_avisos&convoca_pagina=3");
  assert.equal(escribirEstadoRuta("?lang=en&x=1", "todas", 1), "?lang=en&x=1");
  assert.equal(nombrePersona(filas[0]), "Reyes Álvarez, Antonio");
  assert.equal(nombrePersona(filas[1]), "García, María");
  assert.equal(claveCampo("Primer Apellido"), "primer_apellido");
  assert.equal(claveCampo("DNI/NIE enmascarado"), "dni_nie_enmascarado");
  assert.equal(claveCampo("Formación"), "formacion");
  assert.equal(textoIncidencia(textos, { campo: "Total", codigo: "total_incoherente" }), "Total: no es la suma de experiencia y formación.");
  assert.equal(textoIncidencia(textos, { campo: "Rareza", codigo: "nuevo_codigo" }), "Columna: el dato no es válido.");
  assert.match(textoBloqueo(textos, "esquema_detalle_meritos"), /resumen por persona/u);
  assert.equal(claveCategoria("categoria:rpt:auxiliar_administrativo"), "auxiliar_administrativo");
  assert.equal(claveCategoria("otra:cosa"), "");
});

test("las cuentas usan plural y números localizados", async () => {
  const es = await textosDe("es");
  const en = await textosDe("en");
  assert.equal(es.traducir("general.cuentaFilas", { cuenta: 1 }), "1 fila");
  assert.equal(es.traducir("general.cuentaFilas", { cuenta: 1200 }), "1200 filas");
  assert.equal(en.traducir("general.excluir", { cuenta: 3 }), "Load the pool without the 3 rows with errors");
});
