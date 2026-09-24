import assert from "node:assert/strict";
import test from "node:test";
import { crearGestorTramitacion } from "./vista-expedientes-tramitacion.js";
import { renderizarModuloContratacionTemporal } from "./vista-expedientes-render.js";

const A = "expediente:subsanacion:A01";
const B = "expediente:subsanacion:B02";
const ACCION = "contratacion_temporal.subsanacion_reparos.registrar";
const turno = () => new Promise((resolver) => setImmediate(resolver));

function detalle(ref, version) {
  return {
    expediente_ref: ref, demostracion: false, version, numero_visible: ref,
    flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella: "a".repeat(64),
    cabecera: [], fases: [], tareas: [], historial: version >= 7 ? [
      { secuencia: 7, version_expediente: 7, accion_clave: ACCION,
        fecha: "24 sept 2026", fase: "Subsanación", accion: "Subsanación registrada",
        estado: "Incidencia", estado_clave: "incidencia" },
      ...(version > 7 ? [{ secuencia: version, version_expediente: version,
        accion_clave: "registrar_fiscalizacion", fecha: "24 sept 2026",
        fase: "Fiscalización", accion: "Nuevo reparo",
        estado: "Incidencia", estado_clave: "incidencia" }] : []),
    ] : [],
  };
}

function estado(ref, version = 6) {
  return {
    vista: "expediente", carga: "listo", expediente_ref: ref,
    cuadro: { demostracion: false, expedientes: [A, B].map((referencia) => ({
      expediente_ref: referencia, version: referencia === ref ? version : 6,
      fase_clave: "subsanacion_unidad", estado_clave: "incidencia",
    })) },
    expediente: detalle(ref, version), tarea_ref: "", navegacion: {},
    mensaje_clave: "", ocupado: false,
  };
}

function recibo(solicitud) {
  return {
    esquema: "vec.contratacion-temporal.recibo-subsanacion-reparos.v1",
    operacion: "registrar_subsanacion", expediente_ref: solicitud.expediente_ref,
    version_resultante: solicitud.version_esperada + 1,
    fase_resultante: "subsanacion_unidad", estado_resultante: "incidencia",
    recibo_ref: `recibo:${solicitud.expediente_ref}`,
    auditoria_ref: `auditoria:${solicitud.expediente_ref}`,
    evento_ref: `evento:${solicitud.expediente_ref}`,
    actor_ref: "actor:rrhh:001", registrada_en: "2026-09-24T08:00:00Z",
  };
}

function crearRaiz() {
  const documento = {
    body: { append() {} },
    createElement: () => ({ click() {}, remove() {} }),
  };
  documento.activeElement = documento.body;
  let html = "", panel = null;
  const raiz = {
    ownerDocument: documento,
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor;
      panel = valor.includes("data-ct-exp-subsanacion")
        || valor.includes("data-ct-exp-recuperar-archivo") ? crearPanel(documento) : null;
    },
    querySelector(selector) {
      return ["[data-ct-exp-subsanacion]", "[data-ct-exp-recuperar-archivo]"].includes(selector)
        ? panel : null;
    },
    querySelectorAll() { return []; },
    contains(nodo) { return nodo === panel; },
  };
  return { raiz, panel: () => panel };
}

function crearPanel(documento) {
  const eventos = new Map(), nodos = new Map();
  let html = "";
  const panel = {
    ownerDocument: documento,
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor;
      nodos.clear();
      for (const selector of ["[data-ct-subsanacion-form]", "[data-ct-subsanacion-guardar]",
        "[data-ct-subsanacion-enviar]", "[data-ct-subsanacion-recuperar]",
        "[data-ct-subsanacion-recibo]", "[data-ct-subsanacion-archivo]"]) {
        if (!valor.includes(selector.slice(1, -1))) continue;
        const nodo = {
          focus() { documento.activeElement = this; },
          closest(patron) { return patron === selector ? this : null; },
        };
        if (selector === "[data-ct-subsanacion-form]") {
          nodo.elements = { namedItem: () => ({ value: "Corrección conservada para A." }) };
        }
        nodos.set(selector, nodo);
      }
    },
    addEventListener(nombre, fn) { eventos.set(nombre, fn); },
    removeEventListener(nombre) { eventos.delete(nombre); },
    querySelector(selector) { return nodos.get(selector) ?? null; },
    contains(nodo) { return [...nodos.values()].includes(nodo); },
    replaceChildren() { this.innerHTML = ""; },
    async activar(tipo, selector) {
      const target = nodos.get(selector);
      assert.ok(target, `control visible ${selector}`);
      await eventos.get(tipo)({ target, preventDefault() {} });
    },
    async importar(archivo) {
      const target = nodos.get("[data-ct-subsanacion-archivo]");
      assert.ok(target, "selector de archivo visible");
      target.files = [archivo];
      await eventos.get("change")({ target });
    },
  };
  return panel;
}

function escenario(cliente) {
  const dom = crearRaiz();
  let actual = estado(A), montado = true;
  const presentador = {
    obtenerEstado: () => actual,
    cargar: async () => {},
    seleccionarExpediente: async () => {},
  };
  const gestor = crearGestorTramitacion({
    raiz: dom.raiz, presentador, clienteSubsanacion: cliente,
    subsanacionDisponible: true, confirmarOperacion: () => true,
    repintar: () => pintar(), esMontada: () => montado,
  });
  function pintar() {
    gestor.retirarComponentes();
    if (actual.carga === "denegado") gestor.invalidarSubsanacionPorDenegacion();
    dom.raiz.innerHTML = renderizarModuloContratacionTemporal(actual, {
      subsanacionDisponible: true,
      reciboSubsanacionConfirmado: gestor.obtenerReciboSubsanacionConfirmado(actual),
      recuperacionSubsanacionPendiente: gestor.tieneIntencionSubsanacionParaEstado(actual),
    });
    gestor.montarSubsanacionDesdeExpedienteActual();
  }
  function navegar(ref, version = 6) { actual = estado(ref, version); pintar(); }
  function denegar() {
    actual = { vista: "expediente", carga: "denegado", cuadro: null,
      expediente: null, navegacion: {}, mensaje_clave: "estado_denegado" };
    pintar();
  }
  pintar();
  return { dom, gestor, navegar, denegar, desmontar() { montado = false; gestor.retirarComponentes(); } };
}

test("A→B→A tras 503 conserva DTO y clave, también con hito v+1", async () => {
  const peticiones = [], efectos = new Set();
  const x = escenario({ registrarSubsanacionReparos: async (solicitud) => {
    peticiones.push(solicitud);
    efectos.add(solicitud.clave_idempotencia);
    if (peticiones.length === 1) throw Object.assign(new Error("sin respuesta"), { estado: 503 });
    return recibo(solicitud);
  } });
  try {
    await x.dom.panel().activar("submit", "[data-ct-subsanacion-form]");
    assert.equal(peticiones.length, 0, "preparar no envía");
    await x.dom.panel().activar("click", "[data-ct-subsanacion-guardar]");
    await x.dom.panel().activar("click", "[data-ct-subsanacion-enviar]");
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    const original = peticiones[0];
    x.navegar(B);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /Corrección conservada para A/u);
    x.navegar(A, 7);
    assert.match(x.dom.raiz.innerHTML, /data-ct-exp-subsanacion/u);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    await x.dom.panel().activar("click", "[data-ct-subsanacion-recuperar]");
    await turno();
    assert.deepEqual(peticiones, [original, original]);
    assert.equal(efectos.size, 1);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recibo/u);
    x.navegar(A, 7);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recibo/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
  } finally { x.desmontar(); }
});

test("salir durante POST abortado conserva recuperación y oculta datos al denegar", async () => {
  const peticiones = [];
  let rechazarPrimera;
  const pendiente = new Promise((_, rechazar) => { rechazarPrimera = rechazar; });
  const x = escenario({ registrarSubsanacionReparos: (solicitud, opciones) => {
    peticiones.push({ solicitud, signal: opciones.signal });
    return peticiones.length === 1 ? pendiente : Promise.resolve(recibo(solicitud));
  } });
  try {
    const panelOriginal = x.dom.panel();
    await panelOriginal.activar("submit", "[data-ct-subsanacion-form]");
    await panelOriginal.activar("click", "[data-ct-subsanacion-guardar]");
    const envio = panelOriginal.activar("click", "[data-ct-subsanacion-enviar]");
    assert.equal(peticiones.length, 1);
    x.navegar(B);
    assert.equal(peticiones[0].signal.aborted, true);
    rechazarPrimera(new Error("respuesta perdida tras salir"));
    await envio;
    assert.equal(panelOriginal.innerHTML, "");
    x.navegar(A);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    await x.dom.panel().activar("click", "[data-ct-subsanacion-recuperar]");
    await turno();
    assert.deepEqual(peticiones[1].solicitud, peticiones[0].solicitud);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recibo/u);
    x.denegar();
    assert.equal(x.dom.panel(), null);
    assert.doesNotMatch(x.dom.raiz.innerHTML, /Corrección conservada para A|data-ct-subsanacion-recuperar/u);
    x.navegar(A);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar|data-ct-subsanacion-recibo/u);
    assert.equal(peticiones.length, 2);
  } finally { x.desmontar(); }
});

test("recarga con hito v+1 permite importar solo el archivo del expediente y versión originales", async () => {
  const peticiones = [];
  const x = escenario({ registrarSubsanacionReparos: async (solicitud) => {
    peticiones.push(solicitud);
    return recibo(solicitud);
  } });
  const solicitud = {
    expediente_ref: A, version_esperada: 6,
    clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000",
    observaciones: "Corrección recuperada del archivo.",
  };
  const archivo = (dto) => {
    const texto = JSON.stringify({
      esquema: "vec.contratacion-temporal.subsanacion-recuperacion.v1", solicitud: dto,
    });
    return { size: new TextEncoder().encode(texto).byteLength, text: async () => texto };
  };
  try {
    x.navegar(A, 7);
    assert.match(x.dom.raiz.innerHTML, /Subsanación registrada/u);
    assert.doesNotMatch(x.dom.raiz.innerHTML, /data-ct-exp-subsanacion/u);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-archivo/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    await x.dom.panel().importar(archivo({ ...solicitud, expediente_ref: B }));
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    assert.equal(peticiones.length, 0);
    await x.dom.panel().importar(archivo(solicitud));
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    await x.dom.panel().activar("click", "[data-ct-subsanacion-recuperar]");
    await turno();
    assert.deepEqual(peticiones, [solicitud]);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recibo/u);
  } finally { x.desmontar(); }
});

test("403 del POST purga intención y datos visibles antes de cambiar de identidad", async () => {
  const peticiones = [];
  const x = escenario({ registrarSubsanacionReparos: async (solicitud) => {
    peticiones.push(solicitud);
    throw Object.assign(new Error("detalle privado"), { estado: 403, envelopeValido: true });
  } });
  try {
    await x.dom.panel().activar("submit", "[data-ct-subsanacion-form]");
    await x.dom.panel().activar("click", "[data-ct-subsanacion-guardar]");
    await x.dom.panel().activar("click", "[data-ct-subsanacion-enviar]");
    assert.equal(peticiones.length, 1);
    assert.doesNotMatch(x.dom.panel().innerHTML,
      /Corrección conservada para A|detalle privado|data-ct-subsanacion-recuperar|expediente:subsanacion:A01/u);
    x.navegar(B);
    x.navegar(A);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    assert.equal(peticiones.length, 1);
  } finally { x.desmontar(); }
});

test("un reparo nuevo v9 mantiene visible el replay de la intención incierta v6", async () => {
  const peticiones = [];
  const x = escenario({ registrarSubsanacionReparos: async (solicitud) => {
    peticiones.push(solicitud);
    if (peticiones.length === 1) throw Object.assign(new Error("sin respuesta"), { estado: 503 });
    return recibo(solicitud);
  } });
  try {
    await x.dom.panel().activar("submit", "[data-ct-subsanacion-form]");
    await x.dom.panel().activar("click", "[data-ct-subsanacion-guardar]");
    await x.dom.panel().activar("click", "[data-ct-subsanacion-enviar]");
    const original = peticiones[0];
    x.navegar(B);
    x.navegar(A, 9);
    assert.match(x.dom.raiz.innerHTML, /data-ct-exp-subsanacion/u);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    await x.dom.panel().activar("click", "[data-ct-subsanacion-recuperar]");
    await turno();
    assert.deepEqual(peticiones, [original, original]);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recibo/u);
  } finally { x.desmontar(); }
});

test("403 tardío de A desmontada purga intención antes de volver desde B", async () => {
  const peticiones = [];
  let rechazarPrimera;
  const pendiente = new Promise((_, rechazar) => { rechazarPrimera = rechazar; });
  const x = escenario({ registrarSubsanacionReparos: (solicitud) => {
    peticiones.push(solicitud);
    return pendiente;
  } });
  try {
    const panelA = x.dom.panel();
    await panelA.activar("submit", "[data-ct-subsanacion-form]");
    await panelA.activar("click", "[data-ct-subsanacion-guardar]");
    const envio = panelA.activar("click", "[data-ct-subsanacion-enviar]");
    assert.equal(peticiones.length, 1);
    x.navegar(B);
    await x.dom.panel().activar("submit", "[data-ct-subsanacion-form]");
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-guardar/u);
    rechazarPrimera(Object.assign(new Error("privado"), { estado: 403, envelopeValido: true }));
    await envio;
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    assert.doesNotMatch(x.dom.panel().innerHTML,
      /data-ct-subsanacion-guardar|Corrección conservada para A/u);
    x.navegar(A);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-form/u);
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar|Corrección conservada para A/u);
    assert.equal(peticiones.length, 1);
  } finally { x.desmontar(); }
});

test("archivo v6 tras recarga y reparo v9 solo habilita replay manual del DTO original", async () => {
  const peticiones = [];
  const x = escenario({ registrarSubsanacionReparos: async (solicitud) => {
    peticiones.push(solicitud);
    return recibo(solicitud);
  } });
  const solicitud = {
    expediente_ref: A, version_esperada: 6,
    clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000",
    observaciones: "Corrección v6 conservada en archivo.",
  };
  const archivo = (dto) => {
    const texto = JSON.stringify({
      esquema: "vec.contratacion-temporal.subsanacion-recuperacion.v1", solicitud: dto,
    });
    return { size: new TextEncoder().encode(texto).byteLength, text: async () => texto };
  };
  try {
    x.navegar(A, 9);
    await x.dom.panel().importar(archivo({ ...solicitud, expediente_ref: B }));
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    await x.dom.panel().importar(archivo({ ...solicitud, version_esperada: 5 }));
    assert.doesNotMatch(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    await x.dom.panel().importar(archivo(solicitud));
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recuperar/u);
    assert.doesNotMatch(x.dom.panel().innerHTML,
      /data-ct-subsanacion-form|data-ct-subsanacion-enviar/u);
    assert.equal(peticiones.length, 0);
    await x.dom.panel().activar("click", "[data-ct-subsanacion-recuperar]");
    await turno();
    assert.deepEqual(peticiones, [solicitud]);
    assert.match(x.dom.panel().innerHTML, /data-ct-subsanacion-recibo/u);
  } finally { x.desmontar(); }
});
