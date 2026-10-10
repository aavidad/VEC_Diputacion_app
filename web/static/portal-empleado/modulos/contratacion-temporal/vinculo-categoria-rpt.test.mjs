import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../../comun/textos.js";
import { montarVinculoCategoriaRPT } from "./vinculo-categoria-rpt.js";
import { crearClienteIncorporacionPersonalB2HTTP } from "./cliente-http-incorporacion-personal-b2.js";
import { RUTA_CATEGORIAS_RPT, RUTA_VINCULO_RPT, validarEntradaVinculoRPT, validarLecturaVinculoRPT,
  validarPaginaCategoriasRPT, validarReciboVinculoRPT } from "./contrato-vinculo-categoria-rpt.js";

const expediente = "expediente:ct:1", idempotencia = "99000000-0000-4000-8000-000000000002";
const sha = "a".repeat(64), huella = "b".repeat(64);
const analisis = { version_expediente: 7, analisis_version: 3, analisis_recibo_ref: "recibo:analisis:1",
  analisis_huella_sha256: sha, categoria_ref: "categoria:rpt:auxiliar-administrativo" };
const lectura = (vinculo = null) => ({ organizacion_ref: "organizacion:dipgra", expediente_ref: expediente, analisis, vinculo });
const categoria = (id, etiqueta, en) => ({ categoria_id: id, etiqueta, atributos: { grupos: "C2", etiqueta_en: en },
  catalogo_version: 1, catalogo_huella_sha256: huella, fuente_ref: "ejemplo:rpt-publica", aprobacion_ref: "ejemplo:sin-aprobacion-juridica" });
const pagina = () => ({ categorias: [categoria("categoria:rpt:administrativo", "Administrativo", "Administrative officer"),
  categoria("categoria:rpt:auxiliar-administrativo", "Auxiliar administrativo", "Administrative assistant")], hay_mas: false, siguiente_cursor: "" });
const estado = (revision = 1, id = analisis.categoria_ref) => ({ revision, recibo_ref: `recibo:vinculo:${revision}`, catalogo_id: "categorias_rpt",
  modulo_id: "personal", catalogo_version: 1, catalogo_huella_sha256: huella, categoria_id: id, fuente_ref: "ejemplo:rpt-publica",
  motivo_ref: analisis.analisis_recibo_ref, aprobacion_ref: "ejemplo:sin-aprobacion-juridica", prospectivo: true, acredita_procedencia_historica: false });
const recibo = (revision = 1) => ({ expediente_ref: expediente, recibo_ref: `recibo:vinculo:${revision}`, registrado_en: "2026-10-10T09:00:00.123456Z",
  revision, prospectivo: true, acredita_procedencia_historica: false, vinculo: estado(revision) });
const textos = await cargarTextos("contratacion-temporal-vinculo-categoria-rpt");
const error = (estadoHTTP, codigo) => Object.assign(new Error(codigo), { estado: estadoHTTP, codigo, envelopeValido: true });

function raiz() {
  const eventos = new Map();
  return { innerHTML: "", isConnected: true, addEventListener(k, f) { eventos.set(k, f); }, removeEventListener(k) { eventos.delete(k); },
    replaceChildren() { this.innerHTML = ""; },
    async click(accion) { await eventos.get("click")({ target: { closest: () => ({ getAttribute: () => accion }) } }); } };
}
const pausa = () => new Promise((r) => setImmediate(r));
async function montar(cliente, opciones = {}) {
  const r = raiz();
  const desmontar = montarVinculoCategoriaRPT({ raiz: r, cliente, expedienteRef: expediente, textos, generarClave: () => idempotencia, ...opciones });
  await pausa(); await pausa();
  return { r, desmontar };
}
function cliente(extra = {}) {
  const registros = [];
  return { registros, consultarVinculoRPT: async () => lectura(), listarCategoriasRPT: async () => pagina(),
    registrarVinculoRPT: async (e) => { registros.push(e); return recibo(e.revision_esperada + 1); }, ...extra };
}

test("contrato: la entrada exige categoría igual a la del análisis, UUID y recibo anterior coherente", () => {
  const base = { expediente_ref: expediente, version_expediente_esperada: 7, analisis_version: 3, analisis_recibo_ref: "recibo:analisis:1",
    analisis_huella_sha256: sha, categoria_ref: analisis.categoria_ref, catalogo_version: 1, catalogo_huella_sha256: huella,
    categoria_id: analisis.categoria_ref, fuente_ref: "ejemplo:rpt-publica", motivo_ref: "recibo:analisis:1",
    aprobacion_ref: "ejemplo:sin-aprobacion-juridica", revision_esperada: 0, anterior_recibo_ref: "", clave_idempotencia: idempotencia };
  assert.deepEqual(validarEntradaVinculoRPT(base), base);
  for (const cambio of [{ categoria_id: "categoria:rpt:administrativo" }, { clave_idempotencia: "no-uuid" },
    { revision_esperada: 1 }, { anterior_recibo_ref: "recibo:x" }, { persona_ref: "persona:intrusa" }]) {
    assert.throws(() => validarEntradaVinculoRPT({ ...base, ...cambio }));
  }
  assert.throws(() => validarLecturaVinculoRPT(lectura(), "expediente:otro"));
  assert.throws(() => validarPaginaCategoriasRPT({ ...pagina(), categorias: [...pagina().categorias].reverse() }));
  assert.throws(() => validarPaginaCategoriasRPT({ ...pagina(), hay_mas: true, siguiente_cursor: "categoria:rpt:z" }));
  assert.throws(() => validarReciboVinculoRPT({ ...recibo(), revision: 2 }, base));
});

test("cliente: rutas fijas, cursor en la consulta y POST con la intención exacta", async () => {
  const llamadas = [];
  const respuesta = (data) => new Response(JSON.stringify({ data }), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });
  const c = crearClienteIncorporacionPersonalB2HTTP({ fetchImpl: async (ruta, o) => {
    llamadas.push({ ruta, ...o });
    if (ruta.startsWith(RUTA_CATEGORIAS_RPT)) return respuesta({ ...pagina(), categorias: pagina().categorias.slice(1) });
    return respuesta(o.method === "POST" ? recibo() : lectura());
  } });
  await c.consultarVinculoRPT(expediente);
  await c.listarCategoriasRPT("categoria:rpt:administrativo");
  const entrada = { expediente_ref: expediente, version_expediente_esperada: 7, analisis_version: 3, analisis_recibo_ref: "recibo:analisis:1",
    analisis_huella_sha256: sha, categoria_ref: analisis.categoria_ref, catalogo_version: 1, catalogo_huella_sha256: huella,
    categoria_id: analisis.categoria_ref, fuente_ref: "ejemplo:rpt-publica", motivo_ref: "recibo:analisis:1",
    aprobacion_ref: "ejemplo:sin-aprobacion-juridica", revision_esperada: 0, anterior_recibo_ref: "", clave_idempotencia: idempotencia };
  await c.registrarVinculoRPT(entrada);
  assert.equal(llamadas[0].ruta, `${RUTA_VINCULO_RPT}?expediente_ref=${encodeURIComponent(expediente)}`);
  assert.equal(llamadas[1].ruta, `${RUTA_CATEGORIAS_RPT}?cursor=${encodeURIComponent("categoria:rpt:administrativo")}`);
  assert.equal(llamadas[2].method, "POST");
  assert.deepEqual(JSON.parse(llamadas[2].body), entrada);
  assert.ok(llamadas.every((l) => l.credentials === "same-origin" && l.cache === "no-store"));
});

test("muestra la categoría del análisis y la vincula con versión, huella y motivo del análisis", async () => {
  const c = cliente();
  let avisado = null;
  const { r } = await montar(c, { alRegistrar: (v) => { avisado = v; } });
  assert.match(r.innerHTML, /Auxiliar administrativo/u);
  assert.match(r.innerHTML, /Consultar las 2 categorías publicadas \(solo consulta\)/u);
  assert.match(r.innerHTML, /Auxiliar administrativo \(la del análisis\)/u);
  assert.match(r.innerHTML, /no se puede cambiar aquí/u);
  assert.match(r.innerHTML, /Vincular esta categoría/u);
  assert.doesNotMatch(r.innerHTML, /categoria:rpt|ejemplo:|recibo:analisis/u);
  assert.equal(c.registros.length, 0);
  await r.click("vincular"); await pausa();
  assert.equal(c.registros.length, 1);
  const e = c.registros[0];
  assert.equal(e.categoria_id, analisis.categoria_ref);
  assert.equal(e.catalogo_huella_sha256, huella);
  assert.equal(e.motivo_ref, analisis.analisis_recibo_ref);
  assert.equal(e.version_expediente_esperada, 7);
  assert.equal(e.revision_esperada, 0);
  assert.equal(e.anterior_recibo_ref, "");
  assert.match(r.innerHTML, /Categoría vinculada al expediente/u);
  assert.match(r.innerHTML, /\(hecho\)/u);
  assert.equal(avisado.recibo_ref, "recibo:vinculo:1");
});

test("el botón de ayuda abre y cierra la ayuda sin registrar nada", async () => {
  const c = cliente();
  const { r } = await montar(c);
  assert.match(r.innerHTML, /id="ct-vinculo-rpt-ayuda" class="ct-ayuda" hidden/u);
  await r.click("ayuda");
  assert.match(r.innerHTML, /aria-expanded="true"/u);
  assert.doesNotMatch(r.innerHTML, /id="ct-vinculo-rpt-ayuda" class="ct-ayuda" hidden/u);
  assert.equal(c.registros.length, 0);
});

test("vínculo vigente: se muestra como vinculado y no ofrece registrar otra vez", async () => {
  const { r } = await montar(cliente({ consultarVinculoRPT: async () => lectura(estado(1)) }));
  assert.match(r.innerHTML, /Categoría vinculada al expediente/u);
  assert.doesNotMatch(r.innerHTML, /data-vinculo-accion="vincular"/u);
});

test("si el análisis cambió, avisa y registra una revisión nueva enlazada a la anterior", async () => {
  const c = cliente({ consultarVinculoRPT: async () => lectura(estado(1, "categoria:rpt:administrativo")) });
  const { r } = await montar(c);
  assert.match(r.innerHTML, /La categoría vinculada \(Administrativo\) ya no coincide/u);
  await r.click("vincular"); await pausa();
  assert.equal(c.registros[0].revision_esperada, 1);
  assert.equal(c.registros[0].anterior_recibo_ref, "recibo:vinculo:1");
});

test("categoría del análisis sin publicar: lo explica y no ofrece registrar", async () => {
  const { r } = await montar(cliente({ listarCategoriasRPT: async () => ({ ...pagina(), categorias: pagina().categorias.slice(0, 1) }) }));
  assert.match(r.innerHTML, /no está entre las categorías publicadas/u);
  assert.doesNotMatch(r.innerHTML, /data-vinculo-accion="vincular"/u);
});

test("recorre todas las páginas del listado", async () => {
  const cursores = [];
  const paginas = [{ categorias: pagina().categorias.slice(0, 1), hay_mas: true, siguiente_cursor: "categoria:rpt:administrativo" },
    { categorias: pagina().categorias.slice(1), hay_mas: false, siguiente_cursor: "" }];
  const { r } = await montar(cliente({ listarCategoriasRPT: async (cursor) => { cursores.push(cursor); return paginas[cursores.length - 1]; } }));
  assert.deepEqual(cursores, ["", "categoria:rpt:administrativo"]);
  assert.match(r.innerHTML, /Auxiliar administrativo/u);
});

test("errores: pendiente, sin permiso y conflicto con textos llanos y sin códigos", async () => {
  let r = (await montar(cliente({ consultarVinculoRPT: async () => { throw error(409, "preparacion_pendiente"); } }))).r;
  assert.match(r.innerHTML, /análisis de RRHH de este expediente no está confirmado/u);
  let retirado = false;
  r = (await montar(cliente({ consultarVinculoRPT: async () => { throw error(403, "acceso_denegado"); } }), { alDenegar: () => { retirado = true; } })).r;
  assert.equal(retirado, true);
  const c = cliente({ registrarVinculoRPT: async () => { throw error(409, "conflicto"); } });
  r = (await montar(c)).r;
  await r.click("vincular"); await pausa();
  assert.match(r.innerHTML, /han cambiado\. Pulse Actualizar/u);
  assert.doesNotMatch(r.innerHTML, /data-vinculo-accion="vincular"/u);
  await r.click("consultar"); await pausa(); await pausa();
  assert.match(r.innerHTML, /data-vinculo-accion="vincular"/u);
  assert.doesNotMatch(r.innerHTML, /409|conflicto/u);
});

test("resultado incierto: no repite a ciegas; «Comprobar» reconoce el vínculo ya registrado", async () => {
  let lecturas = 0;
  const c = cliente({ registrarVinculoRPT: async () => { throw new Error("red"); },
    consultarVinculoRPT: async () => (++lecturas > 1 ? lectura(estado(1)) : lectura()) });
  const { r } = await montar(c);
  await r.click("vincular"); await pausa();
  assert.match(r.innerHTML, /No se ha podido confirmar si el vínculo quedó registrado/u);
  assert.doesNotMatch(r.innerHTML, /data-vinculo-accion="vincular"/u);
  await r.click("consultar"); await pausa(); await pausa();
  assert.match(r.innerHTML, /Categoría vinculada al expediente/u);
});

test("en inglés usa la etiqueta inglesa del catálogo y los textos en inglés", async () => {
  const en = await cargarTextos("contratacion-temporal-vinculo-categoria-rpt", { idioma: "en" });
  const r = raiz();
  montarVinculoCategoriaRPT({ raiz: r, cliente: cliente(), expedienteRef: expediente, textos: en, generarClave: () => idempotencia });
  await pausa(); await pausa();
  assert.match(r.innerHTML, /Administrative assistant/u);
  assert.match(r.innerHTML, /Link this category/u);
});

test("resultado incierto sin registro: «Comprobar» y después repite con la misma clave", async () => {
  let intentos = 0;
  const claves = [];
  let n = 0;
  const c = cliente({ registrarVinculoRPT: async (e) => { claves.push(e.clave_idempotencia); if (++intentos === 1) throw new Error("red"); return recibo(1); } });
  const r = raiz();
  montarVinculoCategoriaRPT({ raiz: r, cliente: c, expedienteRef: expediente, textos, generarClave: () => `99000000-0000-4000-8000-00000000000${++n}` });
  await pausa(); await pausa();
  await r.click("vincular"); await pausa();
  await r.click("consultar"); await pausa(); await pausa();
  assert.match(r.innerHTML, /data-vinculo-accion="vincular"/u);
  await r.click("vincular"); await pausa();
  assert.equal(claves.length, 2);
  assert.equal(claves[0], claves[1]);
  assert.match(r.innerHTML, /Categoría vinculada al expediente/u);
});

test("desmontar cancela y deja el contenedor vacío", async () => {
  let senal = null;
  const r = raiz();
  const desmontar = montarVinculoCategoriaRPT({ raiz: r, cliente: cliente({ consultarVinculoRPT: (_, o) => { senal = o.signal; return new Promise(() => {}); } }),
    expedienteRef: expediente, textos });
  desmontar();
  assert.equal(senal.aborted, true);
  assert.equal(r.innerHTML, "");
});
