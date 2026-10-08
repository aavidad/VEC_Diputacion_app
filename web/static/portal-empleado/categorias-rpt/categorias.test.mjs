import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal } from "../modulos/contratacion-temporal/cliente-http.js?v=20261008-alta-circular-v3";
import { crearClienteCategorias } from "./cliente.js?v=20261008-alta-circular-v3";
import { crearPuertoCategorias } from "./puerto.js";
import { filtrarCategorias, TAMANO_PAGINA } from "./vista.js";

const catalogo = () => ({
  esquema: "vec.contratacion_temporal.catalogos_alta.v1",
  centros: [],
  categorias: [{
    referencia: "categoria:rpt:auxiliar",
    etiqueta: "Auxiliar administrativo",
    grupos_subgrupos: [{ clave: "C2", etiqueta: "Grupo C2" }],
  }],
  motivos: [],
  documentos: [],
});

test("consulta solo el catálogo de alta por el transporte protegido y no infiere gobierno", async () => {
  const llamadas = [];
  const clienteAlta = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return new Response(JSON.stringify({ data: catalogo() }), {
        status: 200,
        headers: { "content-type": "application/json; charset=utf-8" },
      });
    },
  });
  const puerto = crearClienteCategorias({ clienteAlta });
  const opciones = await puerto.listarOpciones();
  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].ruta, "/api/vec/contratacion-temporal/catalogos-alta");
  assert.equal(llamadas[0].opciones.method, "GET");
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.mode, "same-origin");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  assert.deepEqual(opciones, catalogo().categorias);
  assert.deepEqual(Object.keys(opciones[0]).sort(), ["etiqueta", "grupos_subgrupos", "referencia"]);
  assert.equal(Object.isFrozen(opciones[0]), true);
  assert.equal(puerto.solicitarAlta, null);
  assert.equal(puerto.solicitarDeshabilitacion, null);
});

test("rechaza un DTO que añada un estado supuesto y conserva la denegación del servidor", async () => {
  const conEstado = catalogo();
  conEstado.categorias[0].habilitada = true;
  const clienteMal = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => new Response(JSON.stringify({ data: conEstado }), {
      status: 200, headers: { "content-type": "application/json; charset=utf-8" },
    }),
  });
  await assert.rejects(crearClienteCategorias({ clienteAlta: clienteMal }).listarOpciones(),
    (error) => error.codigo === "respuesta_incompatible");

  const clienteDenegado = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => new Response("", { status: 403 }),
  });
  await assert.rejects(crearClienteCategorias({ clienteAlta: clienteDenegado }).listarOpciones(),
    (error) => error.estado === 403);
});

test("el puerto admite escritores futuros inyectados y la pantalla actual mantiene los botones cerrados", async () => {
  const solicitarAlta = async () => ({ registro: "externo" });
  const puerto = crearPuertoCategorias({ listarOpciones: async () => [], solicitarAlta });
  assert.equal(puerto.solicitarAlta, solicitarAlta);
  assert.equal(puerto.solicitarDeshabilitacion, null);
  const html = await readFile(new URL("./index.html", import.meta.url), "utf8");
  assert.match(html, /data-i18n="anadir"[^>]*disabled|disabled[^>]*data-i18n="anadir"/u);
  assert.match(html, /data-i18n="deshabilitar"[^>]*disabled|disabled[^>]*data-i18n="deshabilitar"/u);
  assert.doesNotMatch(html, /<form[^>]*action=/u);
});

test("filtra por categoría y grupo sin alterar la fuente y pagina a veinte filas", () => {
  const opciones = catalogo().categorias;
  assert.equal(filtrarCategorias(opciones, "AUXILIAR", "", "es-ES").length, 1);
  assert.equal(filtrarCategorias(opciones, "", "C2", "es-ES").length, 1);
  assert.equal(filtrarCategorias(opciones, "", "A1", "es-ES").length, 0);
  assert.equal(TAMANO_PAGINA, 20);
});
