import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteInscripcionesRRHH } from "./inscripcion-rrhh-cliente.js";
import { leerRutaInscripcionesRRHH, rutaInscripcionesRRHH,
  montarInscripcionesRRHH } from "./inscripcion-rrhh-vista.js";
import { cargarTextos } from "../../../comun/textos.js";

const solicitud = { solicitud_ref: "solicitud:1", recibo_ref: "recibo:solicitud:1",
  convocatoria_ref: "convocatoria:1", categoria_ref: "categoria:1", categoria: "Auxiliar administrativo",
  declaracion_ref: "declaracion:1", bolsa_ref: null, persona_resumen: "Lucía Martín",
  estado: "pendiente", version: 1, registrada_en: "2026-10-08T10:00:00Z", motivo_codigo: null };
const detalle = { ...solicitud, bases_ref: "bases:1", catalogo_version: 1,
  plazo_inicio: "2026-10-01T00:00:00Z", plazo_fin: "2026-10-31T23:59:59Z",
  requisitos: [{ codigo: "titulo", descripcion: "Titulación requerida", obligatorio: true,
    estado: "cumple", fuente_ref: "fuente:1", evidencia_ref: "evidencia:1" }] };
const convocatoriaHistorica = { convocatoria_ref: "convocatoria:historica", titulo: "Bolsa de personal de apoyo",
  categorias_resumen: "Auxiliar administrativo", plazo_fin: "2025-10-08T23:59:59Z",
  estado_publicacion: "sustituida" };
const respuesta = (status, data) => ({ ok: status >= 200 && status < 300, status, redirected: false,
  headers: { get: (nombre) => nombre.toLowerCase() === "content-type" ? "application/json" : null },
  text: async () => JSON.stringify({ data }) });

test("cliente RRHH conserva filtros, identidad fuera del cuerpo y valida recibo", async () => {
  const llamadas = [];
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    if (ruta.includes("/decisiones")) return respuesta(201, { esquema: "vec.bolsa.inscripcion.decision.recibo.v1",
      solicitud_ref: "solicitud:1", recibo_ref: "recibo:1", estado: "admitida_a_convocatoria", version: 2,
      decidida_en: "2026-10-08T11:00:00Z", participacion_ref: null, repetida: false });
    return respuesta(200, { esquema: "vec.bolsa.inscripciones.rrhh.v1", convocatoria_titulo: "Bolsa de personal de apoyo", solicitudes: [solicitud],
      total: 1, cursor_siguiente: null });
  } });
  assert.equal((await cliente.listar({ convocatoria: "convocatoria:1", idioma: "en" })).total, 1);
  const recibo = await cliente.decidir({ solicitudRef: "solicitud:1", decision: "admitir", versionEsperada: 1,
    claveIdempotencia: "inscripcion-12345678" });
  assert.equal(recibo.estado, "admitida_a_convocatoria");
  assert.equal(recibo.participacion_ref, null);
  assert.match(llamadas[0].ruta, /estado=pendiente.*convocatoria_ref=convocatoria%3A1.*idioma=en/u);
  assert.equal(llamadas[1].opciones.credentials, "same-origin");
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), { decision: "admitir", version_esperada: 1,
    clave_idempotencia: "inscripcion-12345678" });
});

test("selector consulta convocatoria histórica paginada y la bandeja exige convocatoria", async () => {
  const llamadas = [];
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (ruta) => {
    llamadas.push(ruta);
    return respuesta(200, { esquema: "vec.bolsa.inscripciones.rrhh.convocatorias.v1",
      convocatorias: [convocatoriaHistorica], total: 2, cursor_siguiente: "cv1_siguiente_v1" });
  } });
  await assert.rejects(cliente.listar({ idioma: "es" }), /referencia incompatible/u);
  assert.equal(llamadas.length, 0);
  const pagina = await cliente.convocatorias({ limite: 20, cursor: "cv1_inicio_v1", idioma: "en" });
  assert.equal(pagina.convocatorias[0].estado_publicacion, "sustituida");
  assert.equal(llamadas[0], "/api/vec/bolsa/rrhh/inscripciones/convocatorias?limite=20&cursor=cv1_inicio_v1&idioma=en");
});

test("las etiquetas históricas de hasta 2048 bytes caben en selector y página de 50", async () => {
  for (const etiqueta of ["á".repeat(205), "á".repeat(1024)]) {
    const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (ruta) => ruta.includes("/convocatorias?")
      ? respuesta(200, { esquema: "vec.bolsa.inscripciones.rrhh.convocatorias.v1",
        convocatorias: [{ ...convocatoriaHistorica, categorias_resumen: etiqueta }],
        total: 1, cursor_siguiente: null })
      : respuesta(200, { esquema: "vec.bolsa.inscripciones.rrhh.v1",
        convocatoria_titulo: "Bolsa de personal de apoyo",
        solicitudes: Array.from({ length: 50 }, (_, i) => ({ ...solicitud,
          solicitud_ref: `solicitud:${i + 1}`, categoria: etiqueta })), total: 50, cursor_siguiente: null }) });
    assert.equal((await cliente.convocatorias()).convocatorias[0].categorias_resumen, etiqueta);
    assert.equal((await cliente.listar({ convocatoria: "convocatoria:1" })).solicitudes.length, 50);
  }
  const excesiva = "á".repeat(1025);
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => respuesta(200,
    { esquema: "vec.bolsa.inscripciones.rrhh.convocatorias.v1",
      convocatorias: [{ ...convocatoriaHistorica, categorias_resumen: excesiva }],
      total: 1, cursor_siguiente: null }) });
  await assert.rejects(cliente.convocatorias(), /convocatorias incompatibles/u);
});

test("una respuesta de otra solicitud o sin recibo no confirma la decisión", async () => {
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => respuesta(200,
    { esquema: "vec.bolsa.inscripcion.rrhh.v1", solicitud: { ...detalle, solicitud_ref: "solicitud:ajena" } }) });
  await assert.rejects(cliente.detalle("solicitud:1"), /detalle incompatible/u);
  await assert.rejects(cliente.decidir({ solicitudRef: "solicitud:1", decision: "rechazar",
    motivoCodigo: "base", versionEsperada: 1, claveIdempotencia: "inscripcion-12345678" }), /recibo incompatible/u);
});

test("incorporación pide evidencia y acepta solo una participación verificada por el servidor", async () => {
  let cuerpo;
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (ruta, opciones) => {
    assert.match(ruta, /\/incorporaciones$/u);
    cuerpo = JSON.parse(opciones.body);
    return respuesta(201, { esquema: "vec.bolsa.inscripcion.incorporacion.recibo.v1",
      solicitud_ref: "solicitud:1", recibo_ref: "recibo:incorporacion:1", estado: "incorporada",
      version: 3, participacion_ref: "participacion:1", incorporada_en: "2026-10-09T00:00:00Z",
      repetida: false });
  } });
  const recibo = await cliente.incorporar({ solicitudRef: "solicitud:1", evidenciaRef: "acta:1",
    versionEsperada: 2, claveIdempotencia: "inscripcion-12345678" });
  assert.equal(recibo.participacion_ref, "participacion:1");
  assert.deepEqual(cuerpo, { evidencia_ref: "acta:1", version_esperada: 2,
    clave_idempotencia: "inscripcion-12345678" });
});

test("admisión que dice incorporada exige referencia de participación", async () => {
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => respuesta(201,
    { esquema: "vec.bolsa.inscripcion.decision.recibo.v1", solicitud_ref: "solicitud:1",
      recibo_ref: "recibo:1", estado: "incorporada", version: 2,
      decidida_en: "2026-10-09T00:00:00Z", repetida: false }) });
  await assert.rejects(cliente.decidir({ solicitudRef: "solicitud:1", decision: "admitir",
    versionEsperada: 1, claveIdempotencia: "inscripcion-12345678" }), /recibo incompatible/u);
});

test("los motivos se consultan sólo en el idioma activo", async () => {
  let ruta;
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async (valor) => {
    ruta = valor;
    return respuesta(200, { esquema: "vec.bolsa.inscripcion.motivos.v1", catalogo_version: 3,
      motivos: [{ codigo: "falta_requisito", etiqueta: "Missing requirement", obligatorio: true }] });
  } });
  assert.equal((await cliente.motivos("rechazar", { idioma: "en" })).motivos[0].etiqueta, "Missing requirement");
  assert.equal(ruta, "/api/vec/bolsa/rrhh/inscripciones/motivos?decision=rechazar&idioma=en");
});

test("una respuesta excesiva se corta antes de convertirla en JSON", async () => {
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => ({ ok: true, status: 200,
    redirected: false, headers: { get: (nombre) => nombre === "content-type" ? "application/json" : null },
    body: new ReadableStream({ start(controlador) { controlador.enqueue(new Uint8Array(256 * 1024 + 1)); } }),
    text: async () => { throw new Error("no debe leerse completa"); } }) });
  await assert.rejects(cliente.listar({ convocatoria: "convocatoria:1" }), /respuesta excesiva/u);
});

test("el cliente conserva el código de error gobernado sin mostrarlo directamente", async () => {
  const cliente = crearClienteInscripcionesRRHH({ fetchImpl: async () => ({ ...respuesta(422, null),
    text: async () => JSON.stringify({ error: { codigo: "catalogo_cambiado" } }) }) });
  await assert.rejects(cliente.decidir({ solicitudRef: "solicitud:1", decision: "rechazar",
    motivoCodigo: "falta_requisito", versionEsperada: 1, claveIdempotencia: "inscripcion-12345678" }),
  (error) => error.estado === 422 && error.codigo === "catalogo_cambiado");
});

test("filtro compartible conserva la URL y normaliza estado desconocido", () => {
  assert.deepEqual(leerRutaInscripcionesRRHH("?inscripcion_estado=otro&inscripcion_convocatoria=convocatoria%3A1"),
    { estado: "pendiente", convocatoria: "convocatoria:1", cursor: "", selectorCursor: "" });
  assert.equal(leerRutaInscripcionesRRHH("?inscripcion_convocatorias_cursor=cv1_inicio_v1").selectorCursor,
    "cv1_inicio_v1");
  assert.equal(rutaInscripcionesRRHH("https://vec.example/portal-empleado/?lang=en#solicitudes",
    { estado: "rechazada", convocatoria: "convocatoria:1", cursor: "c:2" }),
  "/portal-empleado/?lang=en&inscripcion_estado=rechazada&inscripcion_convocatoria=convocatoria%3A1&inscripcion_cursor=c%3A2#solicitudes");
});

test("catálogos propios de la pantalla tienen las mismas claves y no necesitan el otro idioma al cargar", async () => {
  const es = await cargarTextos("bolsa-inscripcion-rrhh", { idioma: "es", porDefecto: "es" });
  const en = await cargarTextos("bolsa-inscripcion-rrhh", { idioma: "en", porDefecto: "es" });
  assert.deepEqual(Object.keys(es.seccion("rrhh")), Object.keys(en.seccion("rrhh")));
  assert.equal(en.traducir("rrhh.estado_pendiente"), "Pending");
});

test("selector histórico se abre antes de la bandeja y enlaza filtro exacto", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true };
  let lecturasLista = 0;
  let ruta;
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/#solicitudes"),
    historial: { pushState: (_estado, _titulo, destino) => { ruta = destino; } },
    cliente: { convocatorias: async () => ({ convocatorias: [convocatoriaHistorica], total: 1,
      cursor_siguiente: null }),
      listar: async ({ convocatoria }) => { lecturasLista++; assert.equal(convocatoria, "convocatoria:historica");
        return { convocatoria_titulo: "Bolsa de personal de apoyo", solicitudes: [], total: 0, cursor_siguiente: null }; },
      detalle: async () => detalle, motivos: async () => ({ motivos: [] }),
      decidir: async () => ({}), incorporar: async () => ({}) } });
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(lecturasLista, 0);
  assert.match(raiz.innerHTML, /Primera categoría: Auxiliar administrativo/u);
  assert.match(raiz.innerHTML, /Sustituida/u);
  assert.match(raiz.innerHTML, /Plazo finalizado/u);
  let impedido = false;
  eventos.get("click")({ target: { closest: () => ({ dataset: { inscripcionElegir: "convocatoria:historica" },
    matches: () => true, hasAttribute: () => false }) }, ctrlKey: true,
  preventDefault() { impedido = true; } });
  assert.equal(impedido, false);
  assert.equal(lecturasLista, 0);
  eventos.get("click")({ target: { closest: () => ({ dataset: { inscripcionElegir: "convocatoria:historica" },
    matches: () => true, hasAttribute: () => false }) }, preventDefault() {} });
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(lecturasLista, 1);
  assert.match(ruta, /inscripcion_convocatoria=convocatoria%3Ahistorica/u);
  vista.desmontar();
});

test("URL directa identifica la convocatoria vacía y conserva página histórica al filtrar", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; } };
  let destino;
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/?inscripcion_convocatoria=convocatoria%3Ahistorica&inscripcion_convocatorias_cursor=cv1_hist_v1#solicitudes"),
    historial: { pushState: (_estado, _titulo, ruta) => { destino = ruta; } },
    cliente: { convocatorias: async () => { throw new Error("no debe cargar selector"); },
      listar: async () => ({ convocatoria_titulo: "Bolsa histórica de apoyo", solicitudes: [],
        total: 0, cursor_siguiente: null }),
      detalle: async () => detalle, motivos: async () => ({ motivos: [] }),
      decidir: async () => ({}), incorporar: async () => ({}) } });
  await new Promise((r) => setTimeout(r, 0));
  assert.match(raiz.innerHTML, /Filtro activo: Bolsa histórica de apoyo/u);
  assert.doesNotMatch(raiz.innerHTML, /convocatoria:historica/u);
  const FormDataAnterior = globalThis.FormData;
  try {
    globalThis.FormData = class { get() { return "rechazada"; } };
    eventos.get("submit")({ target: { matches: () => true }, preventDefault() {} });
  } finally { globalThis.FormData = FormDataAnterior; }
  assert.match(destino, /inscripcion_estado=rechazada/u);
  assert.match(destino, /inscripcion_convocatorias_cursor=cv1_hist_v1/u);
  vista.desmontar();
});

test("selector vacío, caído o denegado ofrece una salida sin pedir la bandeja", async () => {
  for (const estado of [0, 503, 403]) {
    const raiz = { innerHTML: "", addEventListener() {}, removeEventListener() {},
      replaceChildren() { this.innerHTML = ""; } };
    const vista = await montarInscripcionesRRHH({ raiz,
      localizacion: new URL("https://vec.example/portal-empleado/#solicitudes"),
      historial: { pushState() {} },
      cliente: { convocatorias: async () => { if (estado) throw Object.assign(new Error("no disponible"), { estado });
        return { convocatorias: [], total: 0, cursor_siguiente: null }; },
        listar: async () => { throw new Error("no debe consultar solicitudes"); },
        detalle: async () => detalle, motivos: async () => ({ motivos: [] }),
        decidir: async () => ({}), incorporar: async () => ({}) } });
    await new Promise((r) => setTimeout(r, 0));
    assert.match(raiz.innerHTML, estado === 503 ? /No se pudieron consultar las convocatorias/u
      : estado === 403 ? /No puede consultar estas solicitudes/u : /No hay convocatorias publicadas/u);
    assert.doesNotMatch(raiz.innerHTML, /tabla-datos/u);
    vista.desmontar();
  }
});

test("una página histórica tardía no sustituye el selector elegido después", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true };
  let resolverTardia;
  let consultas = 0;
  const pagina = (titulo, cursor_siguiente = null) => ({ convocatorias: [{ ...convocatoriaHistorica, titulo }],
    total: 2, cursor_siguiente });
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/#solicitudes"),
    historial: { pushState() {} },
    cliente: { convocatorias: async () => { consultas++;
      if (consultas === 2) return new Promise((resolver) => { resolverTardia = resolver; });
      return consultas === 1 ? pagina("Primera lista", "cv1_siguiente_v1") : pagina("Lista actual"); },
      listar: async () => { throw new Error("no debe consultar solicitudes"); },
      detalle: async () => detalle, motivos: async () => ({ motivos: [] }),
      decidir: async () => ({}), incorporar: async () => ({}) } });
  await new Promise((r) => setTimeout(r, 0));
  const pulsar = (atributo) => eventos.get("click")({ target: { closest: () => ({ dataset: {},
    matches: () => true, hasAttribute: (nombre) => nombre === atributo }) }, preventDefault() {} });
  pulsar("data-inscripcion-selector-siguiente");
  pulsar("data-inscripcion-selector-total");
  await new Promise((r) => setTimeout(r, 0));
  resolverTardia(pagina("Lista antigua"));
  await new Promise((r) => setTimeout(r, 0));
  assert.match(raiz.innerHTML, /Lista actual/u);
  assert.doesNotMatch(raiz.innerHTML, /Lista antigua/u);
  vista.desmontar();
});

test("denegación borra lista y la respuesta tardía no vuelve a mostrarla", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true };
  const localizacion = new URL("https://vec.example/portal-empleado/?inscripcion_convocatoria=convocatoria%3A1#solicitudes");
  let denegaciones = 0;
  const vista = await montarInscripcionesRRHH({ raiz, localizacion, historial: { pushState() {} },
    alDenegacion: () => { denegaciones++; },
    cliente: { convocatorias: async () => ({ convocatorias: [], total: 0, cursor_siguiente: null }),
      listar: async () => { throw Object.assign(new Error("denegado"), { estado: 403 }); },
      detalle: async () => solicitud, motivos: async () => ({ motivos: [] }),
      decidir: async () => ({}), incorporar: async () => ({}) } });
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(denegaciones, 1);
  assert.match(raiz.innerHTML, /No puede consultar estas solicitudes/u);
  assert.doesNotMatch(raiz.innerHTML, /Lucía Martín/u);
  vista.desmontar();
  assert.equal(raiz.innerHTML, "");
});

test("RRHH no puede confirmar admisión con un requisito obligatorio pendiente", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true };
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/?inscripcion_convocatoria=convocatoria%3A1#solicitudes"),
    historial: { pushState() {} },
    cliente: { convocatorias: async () => ({ convocatorias: [], total: 0, cursor_siguiente: null }),
      listar: async () => ({ convocatoria_titulo: "Bolsa de personal de apoyo", solicitudes: [solicitud], total: 1, cursor_siguiente: null }),
      detalle: async () => ({ ...detalle, requisitos: [{ ...detalle.requisitos[0], estado: "pendiente" }] }),
      motivos: async () => ({ motivos: [] }), decidir: async () => { throw new Error("no debe decidir"); },
      incorporar: async () => { throw new Error("no debe incorporar"); } } });
  await new Promise((r) => setTimeout(r, 0));
  const accion = { dataset: { inscripcionAbrir: "solicitud:1" }, matches: () => false };
  eventos.get("click")({ target: { closest: () => accion } });
  await new Promise((r) => setTimeout(r, 0));
  assert.match(raiz.innerHTML, /data-inscripcion-decidir="admitir" disabled/u);
  assert.match(raiz.innerHTML, /Pendiente de comprobar/u);
  assert.match(raiz.innerHTML, /Auxiliar administrativo/u);
  vista.desmontar();
});

test("cambiar de pantalla durante la carga de textos no lanza una lectura tardía", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "anterior", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; } };
  const controlador = new AbortController();
  let continuar;
  let lecturas = 0;
  const textos = await cargarTextos("bolsa-inscripcion-rrhh", { idioma: "es", porDefecto: "es" });
  const montaje = montarInscripcionesRRHH({ raiz, signal: controlador.signal,
    localizacion: new URL("https://vec.example/portal-empleado/#solicitudes"),
    cargarCatalogo: () => new Promise((resolver) => { continuar = resolver; }),
    cliente: { convocatorias: async () => ({ convocatorias: [], total: 0, cursor_siguiente: null }),
      listar: async () => { lecturas++; return { convocatoria_titulo: "Bolsa de personal de apoyo", solicitudes: [], total: 0, cursor_siguiente: null }; },
      detalle: async () => solicitud, motivos: async () => ({ motivos: [] }),
      decidir: async () => ({}), incorporar: async () => ({}) } });
  controlador.abort(); continuar(textos);
  const vista = await montaje;
  assert.equal(lecturas, 0);
  assert.equal(raiz.innerHTML, "");
  assert.equal(eventos.size, 0);
  vista.desmontar();
});

test("una ficha tardía no sustituye la selección RRHH más reciente", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true };
  let resolverPrimera;
  const segunda = { ...detalle, solicitud_ref: "solicitud:2", persona_resumen: "Marina López" };
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/?inscripcion_convocatoria=convocatoria%3A1#solicitudes"),
    historial: { pushState() {} },
    cliente: { convocatorias: async () => ({ convocatorias: [], total: 0, cursor_siguiente: null }),
      listar: async () => ({ convocatoria_titulo: "Bolsa de personal de apoyo", solicitudes: [solicitud, segunda], total: 2, cursor_siguiente: null }),
      detalle: (ref) => ref === "solicitud:1" ? new Promise((r) => { resolverPrimera = r; }) : Promise.resolve(segunda),
      motivos: async () => ({ motivos: [] }), decidir: async () => ({}), incorporar: async () => ({}) } });
  await new Promise((r) => setTimeout(r, 0));
  const pulsar = (ref) => eventos.get("click")({ target: { closest: () => ({
    dataset: { inscripcionAbrir: ref }, matches: () => false }) } });
  pulsar("solicitud:1"); pulsar("solicitud:2");
  await new Promise((r) => setTimeout(r, 0));
  resolverPrimera(detalle);
  await new Promise((r) => setTimeout(r, 0));
  assert.match(raiz.innerHTML, /<dd>Marina López<\/dd>/u);
  assert.doesNotMatch(raiz.innerHTML, /<dd>Lucía Martín<\/dd>/u);
  vista.desmontar();
});

test("identidad pendiente conserva solicitud y clave al reintentar incorporación", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    contains: () => true,
    querySelector: (selector) => selector === "[data-inscripcion-evidencia]" ? { value: "acta:1" } : null };
  const admitida = { ...detalle, estado: "admitida_a_convocatoria", version: 2 };
  const intentos = [];
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/?inscripcion_convocatoria=convocatoria%3A1#solicitudes"),
    historial: { pushState() {} },
    cliente: { convocatorias: async () => ({ convocatorias: [], total: 0, cursor_siguiente: null }),
      listar: async () => ({ convocatoria_titulo: "Bolsa de personal de apoyo", solicitudes: [admitida], total: 1, cursor_siguiente: null }),
      detalle: async () => admitida, motivos: async () => ({ motivos: [] }), decidir: async () => ({}),
      incorporar: async (argumento) => { intentos.push(argumento); throw Object.assign(new Error("pendiente"),
        { estado: 409, codigo: "vinculo_identidad_pendiente" }); } } });
  const pulsar = (atributo, valor) => eventos.get("click")({ target: { closest: () => ({
    dataset: { [atributo]: valor }, matches: () => false,
    hasAttribute: (nombre) => nombre === atributo }) } });
  await new Promise((r) => setTimeout(r, 0));
  pulsar("inscripcionAbrir", "solicitud:1");
  await new Promise((r) => setTimeout(r, 0));
  pulsar("inscripcionDecidir", "incorporar");
  await new Promise((r) => setTimeout(r, 0));
  const confirmar = () => eventos.get("click")({ target: { closest: () => ({ dataset: {}, matches: () => false,
    hasAttribute: (nombre) => nombre === "data-inscripcion-confirmar" }) } });
  confirmar(); await new Promise((r) => setTimeout(r, 0));
  assert.match(raiz.innerHTML, /Pida a Administración que revise su identidad/u);
  assert.match(raiz.innerHTML, /data-inscripcion-confirmar >/u);
  confirmar(); await new Promise((r) => setTimeout(r, 0));
  assert.equal(intentos.length, 2);
  assert.equal(intentos[0].solicitudRef, "solicitud:1");
  assert.equal(intentos[0].claveIdempotencia, intentos[1].claveIdempotencia);
  vista.desmontar();
});

test("sin rutas publicadas (404) la bandeja dice que no está activada, sin reintento, y avisa al menú", async () => {
  const raiz = { innerHTML: "", addEventListener() {}, removeEventListener() {},
    replaceChildren() { this.innerHTML = ""; } };
  const avisos = [];
  const vista = await montarInscripcionesRRHH({ raiz,
    localizacion: new URL("https://vec.example/portal-empleado/#solicitudes"),
    historial: { pushState() {} }, alDisponibilidad: (valor) => avisos.push(valor),
    cliente: { convocatorias: async () => { throw Object.assign(new Error("ausente"), { estado: 404 }); },
      listar: async () => { throw new Error("no debe consultar solicitudes"); },
      detalle: async () => detalle, motivos: async () => ({ motivos: [] }),
      decidir: async () => ({}), incorporar: async () => ({}) } });
  await new Promise((r) => setTimeout(r, 0));
  assert.match(raiz.innerHTML, /no está activada todavía/u);
  assert.doesNotMatch(raiz.innerHTML, /data-inscripcion-reintentar|role="alert"/u);
  assert.deepEqual(avisos, [false]);
  vista.desmontar();
});
