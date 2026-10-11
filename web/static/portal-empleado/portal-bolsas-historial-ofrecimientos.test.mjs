import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteHistorialOfrecimientos, crearSuperficieHistorialOfrecimientos, validarHistorialOfrecimientos, validarRegistroOfrecimiento,
  instanteDesdeFechaLocal, ESQUEMA_HISTORIAL_OFRECIMIENTOS, traducirHistorialOfrecimientos } from "./portal-bolsas-historial-ofrecimientos.js?v=20261010-ct-bolsa-cohorte-v10";
import { crearTextos } from "../comun/textos.js";
import { readFile } from "node:fs/promises";
const bolsa = "bolsa:prueba", oferta = `oferta:${"a".repeat(64)}`, otraOferta = `oferta:${"b".repeat(64)}`;
const participante = { participacion_ref: "participacion:prueba", nombre_visible: "Persona de prueba <b>", orden: 2 };
const comando = (extra = {}) => ({ canal: "correo", resultado: "enviado", oferta_ref: oferta, instante: "2026-10-02T10:00:00.000Z", anotacion: "Registro de aviso", evidencia_ref: "", evidencia_huella_sha256: "", ...extra });
const contacto = (extra = {}) => ({ ...comando(), contacto_ref: "contacto:prueba", participacion_ref: participante.participacion_ref, recibo_ref: "recibo:prueba", reutilizado: false, ...extra });
const historial = (contactos = [], extra = {}) => ({ data: { esquema: ESQUEMA_HISTORIAL_OFRECIMIENTOS, bolsa_ref: bolsa, oferta_ref: oferta, contactos, cursor_siguiente: null, ...extra } });
const tick = () => new Promise((r) => setImmediate(r));
const respuesta = (status, body) => ({ status, ok: status < 300, json: async () => body });
const clienteBase = (extra = {}) => ({ consultar: async () => ({ ok: true, datos: historial().data }), buscar: async () => ({ ok: true, datos: { bolsa: { bolsa_ref: bolsa }, candidatos: [participante], hay_mas: false } }), registrar: async () => ({ ok: true, contacto: contacto() }), ...extra });
const borrador = (extra = {}) => ({ resultado: "enviado", instanteLocal: "2026-10-02T12:00", anotacion: "Registro de aviso", evidencia_ref: "", evidencia_huella_sha256: "", ...extra });

test("contrato de lectura rechaza oferta ajena, resultados desconocidos, duplicados y declaración sin evidencia", () => {
  assert.equal(validarHistorialOfrecimientos(historial([contacto()]), bolsa, oferta).contactos.length, 1);
  for (const sobre of [historial([], { oferta_ref: otraOferta }), historial([contacto({ resultado: "entregado" })]), historial([contacto(), contacto()]), historial([contacto({ resultado: "entrega_declarada" })])]) {
    assert.throws(() => validarHistorialOfrecimientos(sobre, bolsa, oferta));
  }
});

test("entrega declarada exige constancia opaca y huella; datos personales evidentes se rechazan", () => {
  assert.equal(validarRegistroOfrecimiento(comando({ resultado: "entrega_declarada", evidencia_ref: "constancia:prueba", evidencia_huella_sha256: "a".repeat(64) })), "");
  assert.equal(validarRegistroOfrecimiento(comando({ resultado: "entrega_declarada" })), "error_evidencia");
  for (const anotacion of ["aviso@invalid.example", "12345678Z", "600123456", "a".repeat(1001)]) assert.equal(validarRegistroOfrecimiento(comando({ anotacion })), "error_datos");
  assert.equal(validarRegistroOfrecimiento(comando({ evidencia_ref: "constancia:prueba" })), "error_evidencia");
});

test("cliente usa consultas canónicas, política de transporte, candidatos existentes e Idempotency-Key", async () => {
  const llamadas = [];
  const cliente = crearClienteHistorialOfrecimientos({ fetchImpl: async (url, opts) => { llamadas.push({ url, opts }); return respuesta(opts.method === "POST" ? 201 : 200, opts.method === "POST" ? { data: contacto() } : historial()); },
    consultarPersonas: async (b, filtros, opts) => { assert.equal(b, bolsa); assert.equal(filtros.texto, "Persona"); assert.equal(filtros.limite, 50); assert.ok(opts.signal); return { ok: true }; } });
  const { signal } = new AbortController();
  await cliente.consultar(bolsa, oferta, { cursor: "pagina:2", signal });
  await cliente.buscar(bolsa, { texto: "Persona" }, { signal });
  const r = await cliente.registrar(bolsa, participante.participacion_ref, comando(), "clave-prueba", { signal });
  assert.equal(r.contacto.recibo_ref, "recibo:prueba");
  assert.equal(new URL(llamadas[0].url, "https://invalid.example").searchParams.get("oferta_ref"), oferta);
  assert.equal(llamadas[1].url, `/api/vec/bolsa/bolsas/${bolsa}/candidatos/${participante.participacion_ref}/contactos`);
  assert.equal(llamadas[1].opts.headers["Idempotency-Key"], "clave-prueba");
  assert.deepEqual(JSON.parse(llamadas[1].opts.body), comando());
  for (const { opts } of llamadas) { assert.equal(opts.credentials, "same-origin"); assert.equal(opts.mode, "same-origin"); assert.equal(opts.redirect, "error"); assert.equal(opts.cache, "no-store"); assert.equal(opts.signal, signal); assert.equal(opts.referrerPolicy, "no-referrer"); }
});

test("un recibo ajeno o sin referencia no confirma una escritura", async () => {
  for (const data of [contacto({ participacion_ref: "participacion:otra" }), contacto({ recibo_ref: null }), contacto({ anotacion: "Otra operación" })]) {
    const c = crearClienteHistorialOfrecimientos({ fetchImpl: async () => respuesta(201, { data }) });
    await assert.rejects(c.registrar(bolsa, participante.participacion_ref, comando(), "clave"));
  }
});

test("Madrid se interpreta sin depender de zona del navegador y rechaza horas ambiguas o inexistentes", () => {
  assert.equal(instanteDesdeFechaLocal("2026-10-02T12:00"), "2026-10-02T10:00:00.000Z");
  assert.equal(instanteDesdeFechaLocal("2026-01-02T12:00"), "2026-01-02T11:00:00.000Z");
  assert.equal(instanteDesdeFechaLocal("2026-03-29T02:30"), null);
  assert.equal(instanteDesdeFechaLocal("2026-10-25T02:30"), null);
  assert.equal(instanteDesdeFechaLocal("2026-02-30T12:00"), null);
});

test("tras fallo, reintento conserva clave, instante y contenido; éxito conserva recibo", async () => {
  const escritos = []; let intentos = 0;
  const ui = crearSuperficieHistorialOfrecimientos({ cliente: clienteBase({ registrar: async (...args) => { escritos.push(args); if (intentos++ === 0) throw new Error("fallo_controlado"); return { ok: true, contacto: contacto({ reutilizado: true }) }; } }), generarClave: () => "clave-prueba" });
  ui.activar(bolsa, oferta); await tick(); ui.seleccionar(participante.participacion_ref);
  assert.equal(ui.preparar(borrador()), true);
  assert.equal(escritos.length, 0);
  await ui.confirmar(); assert.ok(ui.estado().pendiente); await ui.confirmar();
  assert.deepEqual(escritos[0].slice(0, 4), escritos[1].slice(0, 4));
  assert.equal(ui.estado().recibo.recibo_ref, "recibo:prueba");
  assert.ok(ui.renderizar().includes(traducirHistorialOfrecimientos("recuperado")));
});

test("cambio de oferta y desmontaje cancelan lectura, búsqueda y escritura; respuestas tardías no cambian la nueva", async () => {
  const solicitudes = [], escrituras = [];
  const ui = crearSuperficieHistorialOfrecimientos({ cliente: clienteBase({ consultar: (b, o, opts) => new Promise((resolve) => solicitudes.push({ b, o, opts, resolve })), registrar: (...args) => new Promise((resolve) => escrituras.push({ args, resolve })) }) });
  ui.activar(bolsa, oferta); await tick(); ui.seleccionar(participante.participacion_ref); ui.preparar(borrador()); void ui.confirmar();
  ui.activar(bolsa, otraOferta); await tick();
  assert.equal(solicitudes[0].opts.signal.aborted, true); assert.equal(escrituras[0].args[4].signal.aborted, true);
  solicitudes[0].resolve({ ok: true, datos: historial([contacto()]).data }); escrituras[0].resolve({ ok: true, contacto: contacto() }); await tick();
  assert.equal(ui.estado().oferta, otraOferta); assert.equal(ui.estado().recibo, null); assert.deepEqual(ui.estado().contactos, []);
  ui.desmontar(); assert.equal(solicitudes[1].opts.signal.aborted, true); assert.equal(ui.renderizar(), "");
});

test("búsqueda paginada conserva nombre seguro y renderiza envío sin constancia de entrega", async () => {
  const consultas = [];
  const ui = crearSuperficieHistorialOfrecimientos({ cliente: clienteBase({ consultar: async () => ({ ok: true, datos: historial([contacto()]).data }), buscar: async (b, f) => { consultas.push(f); return { ok: true, datos: { bolsa: { bolsa_ref: bolsa }, candidatos: f.cursor ? [{ ...participante, participacion_ref: "participacion:dos", nombre_visible: "Otra persona" }] : [participante], hay_mas: !f.cursor, cursor_siguiente: f.cursor ? null : "pagina:2" } }; } }) });
  ui.activar(bolsa, oferta); await tick(); await ui.buscar({ texto: "Persona", estado: "disponible" }); await ui.buscar({ mas: true });
  assert.equal(consultas.at(-1).cursor, "pagina:2"); assert.equal(consultas.at(-1).texto, "Persona"); assert.equal(consultas.at(-1).estado, "disponible");
  const html = ui.renderizar(); assert.match(html, /Persona de prueba &lt;b&gt;/); assert.doesNotMatch(html, /Persona de prueba <b>/); assert.ok(html.includes(traducirHistorialOfrecimientos("sin_entrega")));
  assert.doesNotMatch(html, /Entrega declarada por RRHH<\/td>/);
});

test("una consulta tardía no borra los datos que RRHH ya escribió en el formulario", async () => {
  const ui = crearSuperficieHistorialOfrecimientos({ cliente: clienteBase() });
  ui.activar(bolsa, oferta); await tick(); ui.seleccionar(participante.participacion_ref);
  const campo = { name: "anotacion", value: "Acuse revisado por RRHH", closest: () => ({ dataset: { historialForm: "registro" } }) };
  assert.equal(ui.manejarInput({ target: campo }), true);
  await ui.cargar();
  assert.match(ui.renderizar(), /Acuse revisado por RRHH/);
});

test("una respuesta de lectura no borra el texto de búsqueda sin enviar", async () => {
  const ui = crearSuperficieHistorialOfrecimientos({ cliente: clienteBase() });
  ui.activar(bolsa, oferta); await tick();
  const buscador = { name: "texto", value: "Persona buscada", closest: () => ({ dataset: { historialForm: "buscar" } }) };
  assert.equal(ui.manejarInput({ target: buscador }), true);
  await ui.cargar();
  assert.match(ui.renderizar(), /Persona buscada/);
});

test("RRHH puede identificar una fila histórica fuera de la primera página", async () => {
  const leidas = [];
  const ui = crearSuperficieHistorialOfrecimientos({ cliente: clienteBase({
    consultar: async () => ({ ok: true, datos: historial([contacto({ participacion_ref: "participacion:segunda" })]).data }),
    buscar: async (_bolsa, filtros) => {
      leidas.push(filtros.cursor || "");
      return { ok: true, datos: { bolsa: { bolsa_ref: bolsa }, candidatos: filtros.cursor ? [{ ...participante, participacion_ref: "participacion:segunda", nombre_visible: "Marina de prueba" }] : [participante], hay_mas: !filtros.cursor, cursor_siguiente: filtros.cursor ? null : "participacion:primera" } };
    },
  }) });
  ui.activar(bolsa, oferta); await tick();
  assert.match(ui.renderizar(), /Buscar persona/);
  await ui.identificar("participacion:segunda");
  assert.deepEqual(leidas.slice(-2), ["", "participacion:primera"]);
  assert.match(ui.renderizar(), /Marina de prueba/);
});

test("catálogos completos y traducción inglesa por lector común", async () => {
  const es = JSON.parse(await readFile(new URL("../textos/es/bolsa-historial-ofrecimientos.json", import.meta.url)));
  const en = JSON.parse(await readFile(new URL("../textos/en/bolsa-historial-ofrecimientos.json", import.meta.url)));
  assert.deepEqual(Object.keys(es.historial).sort(), Object.keys(en.historial).sort());
  const t = crearTextos({ modulo: "bolsa-historial-ofrecimientos", idioma: "en", localizacion: "en-GB", respaldo: es, propio: en });
  assert.equal(t.traducir("historial.sin_entrega"), "No delivery evidence"); assert.equal(t.faltantes.length, 0);
  assert.ok([es.historial.entrega_declarada, en.historial.entrega_declarada].includes(traducirHistorialOfrecimientos("entrega_declarada")));
});

test("declaración se presenta como declaración de RRHH y el historial pagina sin repetir entradas", async () => {
  const cursores = [];
  const declarado = contacto({ contacto_ref: "contacto:constancia", resultado: "entrega_declarada", evidencia_ref: "constancia:prueba", evidencia_huella_sha256: "a".repeat(64) });
  const ui = crearSuperficieHistorialOfrecimientos({ cliente: clienteBase({ consultar: async (b, o, { cursor }) => { cursores.push(cursor); return { ok: true, datos: historial(cursor ? [declarado] : [contacto()], { cursor_siguiente: cursor ? null : "pagina:2" }).data }; } }) });
  ui.activar(bolsa, oferta); await tick(); await ui.cargar({ mas: true });
  assert.deepEqual(cursores, ["", "pagina:2"]);
  assert.equal(ui.estado().contactos.length, 2);
  const html = ui.renderizar();
  assert.ok(html.includes(traducirHistorialOfrecimientos("entrega_declarada")));
  assert.ok(html.includes(traducirHistorialOfrecimientos("sin_entrega")));
  assert.match(html, /constancia:prueba/);
  assert.doesNotMatch(html, /SMTP/);
});
