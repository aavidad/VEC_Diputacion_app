import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

// Dobles de transporte y DOM; adapter, contratos, presentador y vista son los reales.
const A = "expediente:ct:001", B = "expediente:ct:002";

test("resolución: volver desde el recibo recarga el cuadro sin repetir el registro", async () => {
  const x = await escenario(8);
  try {
    await x.dom.click("abrir", A);
    x.pendientes[0].resolver(preparacion(A, 8)); await flush();
    assert.match(x.dom.actual().innerHTML, /Volver al cuadro actualizado/u);
    // El presentador es inmutable: la consulta se observa en su transporte.
    const antes = x.lecturas.filter((ruta) => ruta.endsWith("/cuadro/consultas")).length;
    await x.dom.click("accion", "volver-cuadro-actualizado");
    assert.equal(x.presentador.obtenerEstado().vista, "cuadro");
    assert.equal(x.presentador.obtenerEstado().carga, "listo");
    assert.equal(x.lecturas.filter((ruta) => ruta.endsWith("/cuadro/consultas")).length, antes + 1);
    assert.deepEqual(x.posts, []);
    assert.equal(x.dom.actual(), null);
  } finally { x.modulo.desmontar(); }
});

test("resolución: reapertura del detalle v8 presenta recibo con cero POST", async () => {
  const x = await escenario(8);
  try {
    await x.dom.click("abrir", A);
    assert.equal(x.presentador.obtenerEstado().expediente.version, 8);
    x.pendientes[0].resolver(preparacion(A, 8)); await flush();
    assert.match(x.dom.actual().innerHTML, /recibo:ct:historico/u);
    assert.doesNotMatch(x.dom.actual().innerHTML, /data-ct-resolucion-formalizacion-form/u);
    assert.deepEqual(x.posts, []);
  } finally { x.modulo.desmontar(); }
});

test("resolución: preparación de otro expediente y degradación v8→v7 fallan cerradas", async () => {
  for (const version of [7, 8]) {
    const x = await escenario(version);
    try {
      await x.dom.click("abrir", A);
      x.pendientes[0].resolver(preparacion(version === 7 ? B : A)); await flush();
      assert.match(x.dom.actual().innerHTML, /no está disponible/u);
      assert.doesNotMatch(x.dom.actual().innerHTML, /data-ct-resolucion-formalizacion-form|propuesta:ct:B/u);
      assert.deepEqual(x.posts, []);
    } finally { x.modulo.desmontar(); }
  }
});

function resumen(ref) {
  return { expediente_ref: ref, numero_visible: ref === A ? "2026/CT-0001" : "2026/CT-0002",
    version: 7, flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
    fase_clave: "nombramiento", estado_clave: "en_curso", centro_ref: "centro:001",
    categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z" };
}
function detalle(ref) {
  return { esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen: resumen(ref),
    solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion", periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
    hitos: Array.from({ length: 7 }, (_, i) => ({
      secuencia: i + 1, version_expediente: i + 1, accion_clave: i === 6 ? "registrar_propuesta" : "registrar_solicitud",
      realizada_en: "2026-09-03T09:00:00Z", fase_destino: i === 6 ? "nombramiento" : "solicitud",
      estado_origen: "en_curso", estado_destino: "en_curso",
    })) };
}
function preparacion(ref, version = 7) {
  const propuesta_ref = ref === A ? "propuesta:ct:A" : "propuesta:ct:B";
  return { esquema: "vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1",
    expediente_ref: ref, propuesta_ref, version_esperada: 7, version_actual: version,
    recibo: version === 7 ? null : { esquema: "vec.contratacion-temporal.resolucion-formalizacion.v1",
      estado: "replay_registrada", expediente_ref: ref, propuesta_ref, version_resultante: 8,
      resolucion_formalizacion_ref: "resolucion:ct:001", documento_resolucion_ref: "documento:ct:001",
      documento_resolucion_version: 1, documento_resolucion_sha256: "b".repeat(64),
      actuacion_ref: "actuacion:ct:001", auditoria_ref: "auditoria:ct:001", outbox_ref: "outbox:ct:001",
      recibo_ref: "recibo:ct:historico", registrada_en: "2026-09-06T12:00:00Z",
      tipo_validacion: "manual_de_ejercicio", firma_oficial: false, eficacia_administrativa: false } };
}
function diferida() {
  let resolver, rechazar;
  const promesa = new Promise((resolve, reject) => { resolver = resolve; rechazar = reject; });
  return { promesa, resolver, rechazar };
}
const flush = () => new Promise((resolve) => setImmediate(resolve));

function crearRaiz() {
  const eventos = new Map(), efectivos = new Set();
  let html = "", contenedor = null;
  const nodos = [];
  const raiz = {
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor; efectivos.clear(); contenedor = null;
      if (valor.includes('<div data-ct-exp-resolucion-formalizacion>')) {
        const locales = new Map();
        contenedor = { innerHTML: "", addEventListener: (t, h) => locales.set(t, h),
          removeEventListener: (t) => locales.delete(t), replaceChildren() { this.innerHTML = ""; } };
        efectivos.add(contenedor); nodos.push(contenedor);
      }
    },
    contains(nodo) { return efectivos.has(nodo); },
    querySelector(selector) {
      return selector === "[data-ct-exp-resolucion-formalizacion]" ? contenedor : null;
    },
    querySelectorAll() { return []; },
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
  };
  async function click(tipo, valor) {
    const selector = `[data-ct-exp-${tipo}]`;
    const clave = tipo === "abrir" ? "ctExpAbrir" : tipo === "vista" ? "ctExpVista" : "ctExpAccion";
    const superficie = html + (contenedor?.innerHTML ?? "");
    assert.ok(superficie.includes(`data-ct-exp-${tipo}="${valor}"`), "control realmente renderizado");
    const control = { dataset: { [clave]: valor }, closest: (s) => s === selector ? control : null };
    efectivos.add(control);
    await eventos.get("click")({ target: control, preventDefault() {} });
  }
  return { raiz, nodos, click, actual: () => contenedor };
}

async function escenario(version = 7) {
  const dom = crearRaiz(), pendientes = [], posts = [], lecturas = [];
  // Pasar por el cliente público valida los fixtures de lectura antes del adapter.
  const clienteLectura = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    lecturas.push(ruta);
    const entrada = JSON.parse(opciones.body);
    const data = ruta.endsWith("/cuadro/consultas")
      ? { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-03T09:05:00Z",
        expedientes: [resumen(A), resumen(B)], hay_mas: false }
      : detalle(entrada.expediente_ref);
    if (data.expedientes) data.expedientes.forEach((r) => { r.version = version; });
    else {
      data.resumen.version = version;
      if (version === 8) data.hitos.push({ ...data.hitos[6], secuencia: 8,
        version_expediente: 8, accion_clave: "registrar_resolucion" });
    }
    return new Response(JSON.stringify({ data }), { status: 200, headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteLectura });
  await fuente.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  const modulo = await montarModuloContratacionTemporal({
    raiz: dom.raiz, presentador, llamamiento: { cliente: {
      prepararResolucionFormalizacion(ref, opciones) {
        const d = diferida(); pendientes.push({ ref, opciones, ...d }); return d.promesa;
      },
      registrarResolucionFormalizacion(s) { posts.push(s); throw new Error("no debe haber POST"); },
    } },
  });
  return { dom, presentador, modulo, pendientes, posts, lecturas };
}

test("resolución: GET fuera de orden A→B solo monta B con presentador real y cero POST", async () => {
  const x = await escenario();
  try {
    await x.dom.click("abrir", A);
    assert.equal(x.pendientes.length, 1);
    const nodoA = x.dom.actual();
    await x.dom.click("vista", "cuadro");
    await x.dom.click("abrir", B);
    assert.equal(x.pendientes.length, 2);
    const nodoB = x.dom.actual();
    assert.notEqual(nodoA, nodoB); assert.equal(x.dom.raiz.contains(nodoA), false);
    x.pendientes[1].resolver(preparacion(B)); await flush();
    assert.match(nodoB.innerHTML, /propuesta:ct:B/u);
    assert.match(nodoB.innerHTML, /data-ct-resolucion-formalizacion-form/u);
    const antes = nodoB.innerHTML;
    x.pendientes[0].resolver(preparacion(A)); await flush();
    assert.equal(nodoB.innerHTML, antes);
    assert.doesNotMatch(nodoA.innerHTML, /data-ct-resolucion-formalizacion-form/u);
    assert.doesNotMatch(nodoB.innerHTML, /propuesta:ct:A/u);
    assert.equal(x.presentador.obtenerEstado().expediente.expediente_ref, B);
    assert.deepEqual(x.posts, []);
  } finally { x.modulo.desmontar(); }
});

test("resolución: detalle v7 superado por GET v8 muestra recibo histórico sin POST", async () => {
  const x = await escenario();
  try {
    await x.dom.click("abrir", A);
    x.pendientes[0].resolver(preparacion(A, 8)); await flush();
    assert.match(x.dom.actual().innerHTML, /recibo:ct:historico/u);
    assert.match(x.dom.actual().innerHTML, /documento:ct:001/u);
    assert.doesNotMatch(x.dom.actual().innerHTML, /data-ct-resolucion-formalizacion-form/u);
    assert.deepEqual(x.posts, []);
  } finally { x.modulo.desmontar(); }
});

for (const salir of ["cuadro", "desmontar"]) {
  test(`resolución: respuesta tras ${salir} no monta en nodo retirado`, async () => {
    const x = await escenario();
    await x.dom.click("abrir", A); const anterior = x.dom.actual();
    if (salir === "cuadro") await x.dom.click("vista", "cuadro");
    else x.modulo.desmontar();
    const html = anterior.innerHTML;
    x.pendientes[0].resolver(preparacion(A)); await flush();
    assert.equal(anterior.innerHTML, html);
    assert.doesNotMatch(anterior.innerHTML, /data-ct-resolucion-formalizacion-form/u);
    assert.deepEqual(x.posts, []); x.modulo.desmontar();
  });
}
for (const estado of [403, 503]) {
  test(`resolución: GET ${estado} visible sin habilitar actuación`, async () => {
    const x = await escenario();
    try {
      await x.dom.click("abrir", A);
      x.pendientes[0].rechazar(Object.assign(new Error("detalle privado"), { estado, envelopeValido: true }));
      await flush();
      const html = x.dom.actual().innerHTML;
      assert.match(html, estado === 403 ? /No dispone de permiso/u : /no está disponible/u);
      assert.doesNotMatch(html, /data-ct-resolucion-formalizacion-form|detalle privado/u);
      assert.deepEqual(x.posts, []);
      if (estado === 503) {
        const reintento = x.dom.click("accion", "reintentar-resolucion");
        assert.equal(x.pendientes.length, 2);
        x.pendientes[1].resolver(preparacion(A)); await reintento;
        assert.match(x.dom.actual().innerHTML, /propuesta:ct:A/u);
      }
    } finally { x.modulo.desmontar(); }
  });
}
