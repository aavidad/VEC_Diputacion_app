import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPCategoriasPersonal, ErrorClienteCategoriasPersonal } from "./cliente-http-categorias.js";

function sobre({ items = [] } = {}) { return { data: { categories: { items, total: items.length, limit: 25, offset: 0, catalogo: { catalogo_id: "categorias-profesionales", catalogo_version: 1, catalogo_huella_sha256: "a".repeat(64) }, fuente: { revision: "demo-v1", actualizada_en: "2026-09-20T08:00:00Z", demostracion: true, aviso: "DEMOSTRACIÓN pendiente de validación RRHH." } } } }; }
function categoria() { return { catalog: "categoria_profesional", clave: "administrativo", slug: "administrativo", etiqueta: "Administrativo", name: "Administrativo", descripcion: "", orden: 1, area: "administracion_general", area_etiqueta: "Administración general", source: "catalogo_gobernado_vec", module_key: "vec.module.personal", state: "Demostración pendiente de validación RRHH", usage: "Bolsa, RPT, certificados y demás módulos autorizados." }; }
function respuesta(cuerpo, estado = 200, tipo = "application/json; charset=utf-8") { return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": tipo } }); }

test("el cliente usa la ruta, filtros y protección same-origin exactos", async () => {
  const llamadas = []; const sinDescripcion = categoria(); delete sinDescripcion.descripcion;
  const cliente = crearClienteHTTPCategoriasPersonal({ fetchImpl: async (...argumentos) => { llamadas.push(argumentos); return respuesta(sobre({ items: [sinDescripcion] })); } });
  const salida = await cliente.listarCategorias({ q: "técnico", area: "administracion_general", limit: 25, offset: 0 });
  assert.equal(salida.items[0].clave, "administrativo"); assert.equal(salida.items[0].descripcion, ""); assert.equal(llamadas[0][0], "/api/vec/personal/categories?q=t%C3%A9cnico&area=administracion_general&limit=25&offset=0");
  assert.deepEqual({ ...llamadas[0][1], signal: undefined }, { method: "GET", credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal: undefined }); assert.equal(llamadas[0][1].signal.aborted, false);
});

test("401, 403, 5xx y sobres incompatibles no exponen filas", async () => {
  for (const respuestaFallida of [respuesta({ error: "autenticacion_requerida" }, 401), respuesta({ error: "acceso_denegado" }, 403), respuesta({ error: "servicio_no_disponible" }, 503), respuesta({ data: { categories: { items: [categoria()] } } })]) {
    const cliente = crearClienteHTTPCategoriasPersonal({ fetchImpl: async () => respuestaFallida });
    await assert.rejects(() => cliente.listarCategorias({ q: "", area: "", limit: 25, offset: 0 }), ErrorClienteCategoriasPersonal);
  }
});

test("el AbortSignal se transmite y cierra la consulta", async () => {
  let señal; const cliente = crearClienteHTTPCategoriasPersonal({ fetchImpl: async (_ruta, opciones) => { señal = opciones.signal; throw new DOMException("abort", "AbortError"); } });
  const controlador = new AbortController(); controlador.abort();
  await assert.rejects(() => cliente.listarCategorias({ q: "", area: "", limit: 25, offset: 0 }, { signal: controlador.signal }), /operacion_abortada/);
  assert.equal(señal, undefined);
});

test("admite exactamente el vacío Go y el offset posterior al total", async () => {
  const paginaVaciaGo = sobre(); paginaVaciaGo.data.categories.items = null; paginaVaciaGo.data.categories.total = 3; paginaVaciaGo.data.categories.offset = 9;
  const cliente = crearClienteHTTPCategoriasPersonal({ fetchImpl: async () => respuesta(paginaVaciaGo) });
  const salida = await cliente.listarCategorias({ q: "", area: "", limit: 25, offset: 9 });
  assert.deepEqual(salida.items, []); assert.equal(salida.offset, 9);
});

test("acota tamaño, plazo y abortos sin dejar la petición activa", async () => {
  const excesiva = crearClienteHTTPCategoriasPersonal({ fetchImpl: async () => new Response("{}", { status: 200, headers: { "content-type": "application/json; charset=utf-8", "content-length": "999999" } }) });
  await assert.rejects(() => excesiva.listarCategorias({ q: "", area: "", limit: 25, offset: 0 }), /respuesta_excesiva/);
  let señal; const lenta = crearClienteHTTPCategoriasPersonal({ plazoMs: 10, fetchImpl: async (_ruta, opciones) => new Promise((_resolver, rechazar) => { señal = opciones.signal; señal.addEventListener("abort", () => rechazar(new DOMException("abort", "AbortError")), { once: true }); }) });
  await assert.rejects(() => lenta.listarCategorias({ q: "", area: "", limit: 25, offset: 0 }), /plazo_agotado/); assert.equal(señal.aborted, true);
  const externo = new AbortController(); const abortable = crearClienteHTTPCategoriasPersonal({ plazoMs: 1000, fetchImpl: async (_ruta, opciones) => new Promise((_resolver, rechazar) => opciones.signal.addEventListener("abort", () => rechazar(new DOMException("abort", "AbortError")), { once: true })) });
  const promesa = abortable.listarCategorias({ q: "", area: "", limit: 25, offset: 0 }, { signal: externo.signal }); externo.abort(); await assert.rejects(() => promesa, /operacion_abortada/);
});

test("cancela cuerpos descartados y no confunde Content-Length comprimido con bytes JSON", async () => {
  for (const respuestaFallida of [
    { ok: false, status: 503, redirected: false, headers: { get: () => "application/json; charset=utf-8" } },
    { ok: true, status: 200, redirected: false, headers: { get: () => "text/plain" } },
  ]) {
    let cancelada = false; respuestaFallida.body = { cancel() { cancelada = true; return Promise.resolve(); } };
    const cliente = crearClienteHTTPCategoriasPersonal({ fetchImpl: async () => respuestaFallida });
    await assert.rejects(() => cliente.listarCategorias({ q: "", area: "", limit: 25, offset: 0 }), ErrorClienteCategoriasPersonal); assert.equal(cancelada, true);
  }
  let cancelada = false; const enorme = { ok: true, status: 200, redirected: false, headers: { get: (clave) => clave === "content-type" ? "application/json; charset=utf-8" : null }, body: { getReader() { return { async read() { return { done: false, value: new Uint8Array(262145) }; }, cancel() { cancelada = true; return Promise.resolve(); }, releaseLock() {} }; } } };
  const clienteEnorme = crearClienteHTTPCategoriasPersonal({ fetchImpl: async () => enorme }); await assert.rejects(() => clienteEnorme.listarCategorias({ q: "", area: "", limit: 25, offset: 0 }), /respuesta_excesiva/); assert.equal(cancelada, true);
  const comprimida = respuesta(sobre(), 200); const original = comprimida.headers.get.bind(comprimida.headers); Object.defineProperty(comprimida, "headers", { value: { get: (clave) => clave === "content-length" ? "1" : original(clave) } });
  const clienteComprimido = crearClienteHTTPCategoriasPersonal({ fetchImpl: async () => comprimida }); assert.equal((await clienteComprimido.listarCategorias({ q: "", area: "", limit: 25, offset: 0 })).total, 0);
});
