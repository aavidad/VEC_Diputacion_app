import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPRPTPublicaV2, ErrorClienteRPTPublicaV2, RUTA_RPT_PUBLICA_V2 } from "./cliente-http-rpt-publica-v2.js";

const huella = "a".repeat(64);
const evidencia = { recibo_ref: "rptpublica:00000000-0000-0000-0000-000000000001", decision_ref: "decision:prueba", efecto_ref: "rpt-publicada:2026-05-07", consumo_huella_sha256: huella, auditoria_ref: "auditoria:prueba", consultada_en: "2026-10-07T10:00:00.000000Z" };
const consulta = (cambios = {}) => ({ vista: "categorias", q: "", limit: 25, offset: 0, categoria_clave: "", centro_codigo: "", ...cambios });
function categoria() { return { clave: "administrativo", denominacion: "ADMINISTRATIVO", origen: "categoria", grupos: ["C1"], escalas: ["AG"], puestos: 2, dotacion: 3, nivel_destino_mediana: 17, complemento_especifico_anual_centimos_mediana: 1461544 }; }
function puesto() { return { codigo: "430-101-001", denominacion: "SECRETARÍA", centro_codigo: "101", centro: "GABINETE", delegacion: "PRESIDENCIA", grupos: [], escala: "", categoria_clave: "", categorias_claves: [], categorias_pendientes: [{ denominacion: "AUXILIAR", origen: "categoria" }], nivel_destino: 0, complemento_especifico_anual_centimos: 0, dotacion: 3, tipo: "E", provision: "I" }; }
function sobre(vista = "categorias") { return { data: { rpt: { items: [vista === "puestos" ? puesto() : categoria()], total: 1, limit: 25, offset: 0, vista, esquema: "vec.catalogo.rpt.candidato.v1", estado: "preparacion_no_autoritativa", publicacion_ref: "rpt-publicada:2026-05-07", corte: "2026-05-07", huella_sha256: huella, fuente: { documento: "RPT publicada", importacion: "rpt-2026", generado_en: "2026-09-17", aviso: "Sin ocupantes." }, resumen: { puestos: 842, dotacion: 1714, categorias: 131, centros: 41 }, categorias_pendientes_grupo: ["AUXILIAR"], evidencia } } }; }
function respuesta(cuerpo, estado = 200) { return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } }); }

test("cliente v2 consulta la ruta interna sin cookies y conserva procedencia e incertidumbres", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPRPTPublicaV2({ fetchImpl: async (...args) => { llamadas.push(args); return respuesta(sobre("puestos")); } });
  const pagina = await cliente.listar(consulta({ vista: "puestos", q: "secretaría" }));
  assert.equal(llamadas[0][0], `${RUTA_RPT_PUBLICA_V2}?vista=puestos&q=secretar%C3%ADa&limit=25&offset=0`);
  assert.equal(llamadas[0][1].credentials, "omit");
  assert.equal(llamadas[0][1].cache, "no-store");
  assert.equal(pagina.estado, "preparacion_no_autoritativa");
  assert.equal(pagina.categorias_pendientes_grupo[0], "AUXILIAR");
  assert.equal(pagina.items[0].categorias_pendientes[0].denominacion, "AUXILIAR");
  assert.equal(pagina.evidencia.efecto_ref, pagina.publicacion_ref);
});

test("cliente v2 envía categoría y centro como filtros exactos de la página", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPRPTPublicaV2({ fetchImpl: async (...args) => { llamadas.push(args); return respuesta(sobre("puestos")); } });
  await cliente.listar(consulta({ vista: "puestos", categoria_clave: "administrativo", centro_codigo: "101" }));
  assert.equal(llamadas[0][0], `${RUTA_RPT_PUBLICA_V2}?vista=puestos&q=&limit=25&offset=0&categoria_clave=administrativo&centro_codigo=101`);
  await assert.rejects(() => cliente.listar(consulta({ categoria_clave: "administrativo" })), TypeError);
});

test("cliente v2 conserva 100 caracteres Unicode legales y acota la clave", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPRPTPublicaV2({ fetchImpl: async (...args) => { llamadas.push(args); return respuesta(sobre("puestos")); } });
  for (const q of ["á".repeat(100), "😀".repeat(100)]) {
    await cliente.listar(consulta({ vista: "puestos", q, categoria_clave: "administrativo", centro_codigo: "101" }));
    const url = llamadas.at(-1)[0];
    assert.ok(url.slice(RUTA_RPT_PUBLICA_V2.length + 1).length > 512);
    assert.ok(url.slice(RUTA_RPT_PUBLICA_V2.length + 1).length <= 2048);
    assert.equal(new URL(url, "https://vec.example").searchParams.get("q"), q);
  }
  await assert.rejects(() => cliente.listar(consulta({ vista: "puestos", categoria_clave: "a".repeat(61) })), TypeError);
  await assert.rejects(() => cliente.listar(consulta({ vista: "puestos", q: "😀".repeat(101) })), TypeError);
  assert.equal(llamadas.length, 2);
});

test("cliente v2 rechaza atributos personales, autoridad falsa y evidencia ajena", async () => {
  for (const alterar of [
    (r) => { r.data.rpt.items[0].ocupante = "persona"; },
    (r) => { r.data.rpt.estado = "habilitada"; },
    (r) => { r.data.rpt.evidencia.efecto_ref = "rpt-publicada:ajena"; },
  ]) {
    const r = sobre("puestos"); alterar(r);
    const cliente = crearClienteHTTPRPTPublicaV2({ fetchImpl: async () => respuesta(r) });
    await assert.rejects(() => cliente.listar(consulta({ vista: "puestos" })), ErrorClienteRPTPublicaV2);
  }
});

test("cliente v2 distingue falta de autenticación y cancela una petición", async () => {
  const sinSesion = crearClienteHTTPRPTPublicaV2({ fetchImpl: async () => respuesta({ error: "autenticacion_requerida" }, 401) });
  await assert.rejects(() => sinSesion.listar(consulta()), (e) => e instanceof ErrorClienteRPTPublicaV2 && e.codigo === "autenticacion_requerida" && e.estado === 401);
  let señal;
  const cliente = crearClienteHTTPRPTPublicaV2({ fetchImpl: (_ruta, opciones) => new Promise((_res, reject) => { señal = opciones.signal; señal.addEventListener("abort", () => reject(new DOMException("abort", "AbortError")), { once: true }); }) });
  const c = new AbortController();
  const pendiente = cliente.listar(consulta(), { signal: c.signal });
  c.abort();
  await assert.rejects(() => pendiente, ErrorClienteRPTPublicaV2);
  assert.equal(señal.aborted, true);
});
