import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPRPTPublica, ErrorClienteRPTPublica } from "./cliente-http-rpt-publica.js";
function categoria() { return { clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 57, dotacion: 158, puestos_vinculados: 57, dotacion_vinculada: 158, recuento_coincide: true }; }
function puesto() { return { codigo: "430-101-001", denominacion: "SECRETARIA DE GRUPO", centro_codigo: "101", centro: "GABINETE DE PRESIDENCIA", delegacion: "PRESIDENCIA", grupos: [], escala: "", categoria_clave: "", nivel_destino: 0, complemento_especifico_anual_centimos: 0, dotacion: 3, tipo: "E", provision: "I" }; }
function sobre(vista = "categorias") { return { data: { rpt: { items: [vista === "puestos" ? puesto() : categoria()], total: 1, limit: 25, offset: 0, vista, enlaces: true, esquema: "vec.catalogo.rpt.v1", fuente: { documento: "RPT publicada", importacion: "rpt-publica-v1", generado_en: "2026-09-17", aviso: "Datos públicos sin ocupantes.", huella_sha256: "a".repeat(64) }, resumen: { puestos: 842, dotacion: 1714, categorias: 145, centros: 41 } } } }; }
function respuesta(cuerpo, estado = 200) { return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } }); }
test("cliente RPT usa GET same-origin y conserva la URL de categorías", async () => { const llamadas = []; const cliente = crearClienteHTTPRPTPublica({ fetchImpl: async (...args) => { llamadas.push(args); return respuesta(sobre()); } }); const pagina = await cliente.listar({ vista: "categorias", q: "admin", limit: 25, offset: 0 }); assert.equal(pagina.items[0].denominacion, "ADMINISTRATIVO"); assert.deepEqual(pagina.resumen, { puestos: 842, dotacion: 1714, categorias: 145, centros: 41 }); assert.equal(llamadas[0][0], "/api/vec/personal/rpt-publica?q=admin&limit=25&offset=0&enlaces=1"); assert.equal(llamadas[0][1].credentials, "same-origin"); assert.equal(llamadas[0][1].cache, "no-store"); });
test("cliente RPT admite solo la proyección pública completa de puestos", async () => { const cliente = crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(sobre("puestos")) }); const pagina = await cliente.listar({ vista: "puestos", q: "presidencia", limit: 25, offset: 0 }); assert.equal(pagina.items[0].codigo, "430-101-001"); assert.equal(pagina.items[0].centro, "GABINETE DE PRESIDENCIA"); assert.equal(pagina.items[0].nivel_destino, 0); const sensible = sobre("puestos"); sensible.data.rpt.items[0].ocupante = "Antonio López Fernández"; const estricto = crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(sensible) }); await assert.rejects(() => estricto.listar({ vista: "puestos", q: "", limit: 25, offset: 0 }), ErrorClienteRPTPublica); });
test("cliente RPT rechaza selector y esquema incompatibles", async () => { const cliente = crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(sobre()) }); await assert.rejects(() => cliente.listar({ vista: "jefaturas", q: "", limit: 25, offset: 0 }), TypeError); const falsa = sobre(); falsa.data.rpt.resumen.centros = -1; const estricto = crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(falsa) }); await assert.rejects(() => estricto.listar({ vista: "categorias", q: "", limit: 25, offset: 0 }), ErrorClienteRPTPublica); });
test("cliente RPT propaga cancelación durante una petición activa", async () => { let señal; const cliente = crearClienteHTTPRPTPublica({ fetchImpl: async (_ruta, opciones) => new Promise((_resolver, rechazar) => { señal = opciones.signal; señal.addEventListener("abort", () => rechazar(new DOMException("abort", "AbortError")), { once: true }); }) }); const abortador = new AbortController(); const pendiente = cliente.listar({ vista: "categorias", q: "", limit: 25, offset: 0 }, { signal: abortador.signal }); abortador.abort(); await assert.rejects(() => pendiente, ErrorClienteRPTPublica); assert.equal(señal.aborted, true); });

test("cliente RPT pide enlaces y filtra por claves exactas sin transformar q", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPRPTPublica({ fetchImpl: async (url) => { llamadas.push(url); const v = sobre("puestos"); v.data.rpt.items[0].categoria_clave = "administrativo"; v.data.rpt.items[0].centro_codigo = "101"; return respuesta(v); } });
  await cliente.listar({ vista: "puestos", q: "secretaria", categoria_clave: "administrativo", centro_codigo: "101", limit: 25, offset: 0 });
  assert.equal(llamadas[0], "/api/vec/personal/rpt-publica?q=secretaria&limit=25&offset=0&enlaces=1&vista=puestos&categoria_clave=administrativo&centro_codigo=101");
  await assert.rejects(() => cliente.listar({ vista: "categorias", q: "", categoria_clave: "administrativo", limit: 25, offset: 0 }), TypeError);
});

test("cliente RPT coteja recuentos enlazados y lista de centros", async () => {
  const pagina = sobre(); pagina.data.rpt.items[0] = { ...categoria(), puestos_vinculados: 62, dotacion_vinculada: 163, recuento_coincide: false };
  const cliente = crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(pagina) });
  assert.equal((await cliente.listar({ vista: "categorias", q: "", limit: 25, offset: 0 })).items[0].puestos_vinculados, 62);
  const incorrecta = structuredClone(pagina); incorrecta.data.rpt.items[0].recuento_coincide = true;
  await assert.rejects(() => crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(incorrecta) }).listar({ vista: "categorias", q: "", limit: 25, offset: 0 }), ErrorClienteRPTPublica);
  const centros = sobre("puestos"); centros.data.rpt.vista = "centros";
  centros.data.rpt.items = [{ codigo: "101", denominacion: "CENTRO 101", puestos: 2, dotacion: 5 }];
  const lista = await crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(centros) }).listar({ vista: "centros", q: "", limit: 25, offset: 0 });
  assert.equal(lista.items[0].codigo, "101");
});

test("cliente RPT sólo admite el puesto del código exacto y combina filtros", async () => {
  const llamadas = [];
  const correcto = sobre("puestos"); correcto.data.rpt.items[0].codigo = "217";
  correcto.data.rpt.items[0].categoria_clave = "administrativo";
  const cliente = crearClienteHTTPRPTPublica({ fetchImpl: async (url) => { llamadas.push(url); return respuesta(correcto); } });
  const consulta = { vista: "puestos", q: "secretaria", categoria_clave: "administrativo", centro_codigo: "101", codigo_puesto: "217", limit: 25, offset: 0 };
  assert.equal((await cliente.listar(consulta)).items[0].codigo, "217");
  assert.equal(llamadas[0], "/api/vec/personal/rpt-publica?q=secretaria&limit=25&offset=0&enlaces=1&vista=puestos&categoria_clave=administrativo&centro_codigo=101&codigo_puesto=217");
  const parcial = structuredClone(correcto); parcial.data.rpt.items[0].codigo = "1217";
  await assert.rejects(() => crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(parcial) }).listar(consulta), ErrorClienteRPTPublica);
  const multiples = structuredClone(correcto); multiples.data.rpt.total = 2; multiples.data.rpt.offset = 25; multiples.data.rpt.items = [];
  await assert.rejects(() => crearClienteHTTPRPTPublica({ fetchImpl: async () => respuesta(multiples) }).listar({ ...consulta, offset: 25 }), ErrorClienteRPTPublica);
  for (const codigo_puesto of ["217a", " 217", "217/1", 217]) {
    await assert.rejects(() => cliente.listar({ ...consulta, codigo_puesto }), TypeError);
  }
  await assert.rejects(() => cliente.listar({ ...consulta, vista: "categorias" }), TypeError);
});
