import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteInscripcionBolsa } from "./inscripcion-bolsa-api.js";
import { prepararIdiomas } from "../comun/idioma.js";

await prepararIdiomas();

const respuesta = (json, status = 200, headers = {}) => new Response(JSON.stringify(json), {
  status, headers: { "Content-Type": "application/json; charset=utf-8", ...headers },
});

const instante = "2026-10-09T09:00:00Z";
const bolsa = { convocatoria_ref: "convocatoria:plaza-42", titulo: "Bolsa de auxiliares 2026",
  numero_categorias: 1, categorias: [{ categoria_ref: "categoria:auxiliar", categoria: "Auxiliar administrativo" }],
  plazo_inicio: instante, plazo_fin: "2026-10-22T23:59:59Z", requisitos_resumen: "Titulación requerida",
  catalogo_version: 2, puede_iniciar: true, estado_solicitud_propia: null, solicitud_ref: null };
const { categorias: _categoriasDetalle, ...resumen } = bolsa;

test("la persona consulta páginas y presenta sin datos de identidad del navegador", async () => {
  const llamadas = [];
  const respuestas = [
    { data: { esquema: "vec.bolsa.inscripciones.convocatorias_abiertas.v1", convocatorias: [resumen], total: 1, cursor_siguiente: null } },
    { data: { esquema: "vec.bolsa.inscripcion.convocatoria.v1", convocatoria: { ...bolsa, puede_iniciar: true,
      requisitos: [{ codigo: "titulo", descripcion: "Título exigido", obligatorio: true,
        estado: "pendiente", motivo_codigo: "sin_evaluacion", motivo_etiqueta: "Pendiente de comprobar" }] } } },
    { data: { esquema: "vec.bolsa.inscripcion.recibo.v1", solicitud_ref: "solicitud:42", recibo_ref: "recibo:42",
      convocatoria_ref: bolsa.convocatoria_ref, categoria: bolsa.categorias[0].categoria, estado: "pendiente", version: 1, registrada_en: instante, repetida: false } },
  ];
  const cliente = crearClienteInscripcionBolsa({ idioma: "es", fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuesta(respuestas.shift());
  } });
  assert.equal((await cliente.abiertas()).total, 1);
  assert.equal((await cliente.convocatoria(bolsa.convocatoria_ref)).convocatoria.catalogo_version, 2);
  const recibo = await cliente.inscribir({ convocatoriaRef: bolsa.convocatoria_ref,
    categoriaRef: bolsa.categorias[0].categoria_ref, catalogoVersion: 2,
    claveIdempotencia: "77777777-7777-4777-8777-777777777777" });
  assert.equal(recibo.estado, "pendiente");
  assert.equal(llamadas[0].ruta, "/api/vec/bolsa/inscripciones/convocatorias-abiertas?limite=20&idioma=es");
  assert.equal(llamadas[1].ruta, "/api/vec/bolsa/inscripciones/convocatorias-abiertas/convocatoria:plaza-42?idioma=es");
  assert.equal(llamadas[2].ruta, "/api/vec/bolsa/inscripciones/propias");
  assert.deepEqual(JSON.parse(llamadas[2].opciones.body), { convocatoria_ref: bolsa.convocatoria_ref,
    categoria_ref: bolsa.categorias[0].categoria_ref,
    catalogo_version: 2, clave_idempotencia: "77777777-7777-4777-8777-777777777777", declaraciones: [] });
  assert.ok(llamadas.every(({ opciones }) => opciones.credentials === "same-origin"
    && opciones.mode === "same-origin" && opciones.cache === "no-store"
    && opciones.redirect === "error" && opciones.referrerPolicy === "no-referrer"));
});

test("rechaza respuesta de otra persona o bolsa y conserva el código de error", async () => {
  const solicitud = { solicitud_ref: "solicitud:ajena", recibo_ref: "recibo:1", convocatoria_ref: bolsa.convocatoria_ref,
    convocatoria_titulo: bolsa.titulo, categoria: bolsa.categorias[0].categoria,
    estado: "pendiente", version: 1, registrada_en: instante };
  const cliente = crearClienteInscripcionBolsa({ fetchImpl: async () => respuesta({
    data: { esquema: "vec.bolsa.inscripcion.propias.detalle.v1", solicitud },
  }) });
  await assert.rejects(cliente.detallePropio("solicitud:propia"), /distinta/u);
  const denegado = crearClienteInscripcionBolsa({ fetchImpl: async () => new Response(null, { status: 403 }) });
  await assert.rejects(denegado.propias(), (error) => error.status === 403);
});

test("422 diferencia plazo cerrado de requisito cambiado mediante código público", async () => {
  const cliente = crearClienteInscripcionBolsa({ fetchImpl: async () => respuesta(
    { error: { codigo: "requisito_invalido", mensaje: "no mostrar" } }, 422) });
  await assert.rejects(cliente.inscribir({ convocatoriaRef: bolsa.convocatoria_ref,
    categoriaRef: bolsa.categorias[0].categoria_ref, catalogoVersion: 2,
    claveIdempotencia: "clave-42", declaraciones: [{ requisito_codigo: "titulo" }] }),
  (error) => error.status === 422 && error.codigo === "requisito_invalido" && !error.message.includes("no mostrar"));
});

test("una denegación propia requiere motivo visible y no muestra el código", async () => {
  const solicitud = { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", convocatoria_ref: bolsa.convocatoria_ref,
    convocatoria_titulo: bolsa.titulo, categoria: bolsa.categorias[0].categoria, estado: "rechazada", version: 2, registrada_en: instante,
    motivo_codigo: "titulo_no_acreditado", decision_ref: "decision:42" };
  const cliente = crearClienteInscripcionBolsa({ fetchImpl: async () => respuesta({
    data: { esquema: "vec.bolsa.inscripcion.propias.detalle.v1", solicitud },
  }) });
  await assert.rejects(cliente.detallePropio("solicitud:42"), /inválida/u);
  solicitud.motivo_etiqueta = "No consta la titulación exigida";
  assert.equal((await cliente.detallePropio("solicitud:42")).solicitud.motivo_etiqueta,
    "No consta la titulación exigida");
});

test("la lectura propia pide sólo el idioma activo", async () => {
  const rutas = [];
  const solicitud = { solicitud_ref: "solicitud:42", recibo_ref: "recibo:42", convocatoria_ref: bolsa.convocatoria_ref,
    convocatoria_titulo: bolsa.titulo, categoria: bolsa.categorias[0].categoria, estado: "pendiente", version: 1, registrada_en: instante };
  const cliente = crearClienteInscripcionBolsa({ idioma: "en", fetchImpl: async (ruta) => {
    rutas.push(ruta);
    return respuesta({ data: { esquema: "vec.bolsa.inscripcion.propias.detalle.v1", solicitud } });
  } });
  await cliente.detallePropio("solicitud:42");
  assert.deepEqual(rutas, ["/api/vec/bolsa/inscripciones/propias/solicitud:42?idioma=en"]);
});

test("el contrato acepta 128 categorías sin truncar y rechaza una página de 129", async () => {
  const categorias = Array.from({ length: 128 }, (_, indice) => ({
    categoria_ref: `categoria:${indice + 1}`, categoria: `Categoría ${indice + 1}`,
  }));
  const publicada = { ...bolsa, categorias, numero_categorias: 128 };
  const { categorias: _completas, ...resumenPublicada } = publicada;
  const respuestas = [
    { data: { esquema: "vec.bolsa.inscripciones.convocatorias_abiertas.v1",
      convocatorias: [resumenPublicada], total: 1, cursor_siguiente: null } },
    { data: { esquema: "vec.bolsa.inscripcion.convocatoria.v1",
      convocatoria: { ...publicada, requisitos: [] } } },
  ];
  const cliente = crearClienteInscripcionBolsa({ idioma: "es", fetchImpl: async () => respuesta(respuestas.shift()) });
  assert.equal((await cliente.abiertas()).convocatorias[0].numero_categorias, 128);
  assert.equal((await cliente.convocatoria(publicada.convocatoria_ref)).convocatoria.categorias[127].categoria_ref,
    "categoria:128");
  const invalido = crearClienteInscripcionBolsa({ idioma: "es", fetchImpl: async () => respuesta({
    data: { esquema: "vec.bolsa.inscripciones.convocatorias_abiertas.v1",
      convocatorias: [{ ...resumenPublicada, numero_categorias: 129 }],
      total: 1, cursor_siguiente: null },
  }) });
  await assert.rejects(invalido.abiertas(), /inválida/u);
});

test("20 tarjetas con 128 categorías quedan bajo 256 KiB y la lista no arrastra los 128 nombres", async () => {
  const convocatorias = Array.from({ length: 20 }, (_, indice) => ({
    ...resumen, convocatoria_ref: `convocatoria:${indice + 1}`, numero_categorias: 128,
    titulo: `Convocatoria de prueba ${indice + 1}`,
  }));
  const respuesta = { data: { esquema: "vec.bolsa.inscripciones.convocatorias_abiertas.v1",
    convocatorias, total: 20, cursor_siguiente: null } };
  assert.ok(Buffer.byteLength(JSON.stringify(respuesta)) < 256 * 1024);
  const cliente = crearClienteInscripcionBolsa({ idioma: "es", fetchImpl: async () =>
    new Response(JSON.stringify(respuesta), { headers: { "Content-Type": "application/json" } }) });
  const pagina = await cliente.abiertas();
  assert.equal(pagina.convocatorias.length, 20);
  assert.ok(pagina.convocatorias.every((convocatoria) => convocatoria.categorias === undefined));
});
