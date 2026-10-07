import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

const REF = "expediente:ct:reintento";
const resumen = Object.freeze({
  expediente_ref: REF, numero_visible: "2026/CT-0001", version: 8,
  flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
  fase_clave: "nombramiento", estado_clave: "en_curso", centro_ref: "centro:001",
  categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z",
});
const seguimiento = Object.freeze({
  opciones: {
    causas_cese: [{ clave: "fin_sustitucion", etiqueta: "Fin de la sustitución", justificante_tipo: "comunicacion_reincorporacion" }],
    condiciones_cierre: ["cese_registrado"], fase_retorno_modificacion: "fiscalizacion", motivos_modificacion: [],
  },
  estado: { expediente_ref: REF, incorporacion: { inicio: "2026-10-01" },
    cese: { causa_clave: "fin_sustitucion", fecha_efecto: "2027-02-15" }, cierre: null },
});
const esperar = () => new Promise((resolver) => setImmediate(resolver));

function crearDOM() {
  const eventos = new Map();
  let zona = null;
  const documento = { activeElement: null, createElement: () => nodo() };
  const nodo = () => {
    const escuchas = new Map();
    return { innerHTML: "", dataset: {}, hijos: [], isConnected: true, hidden: false,
      ownerDocument: documento, setAttribute() {},
      focus() { documento.activeElement = this; },
      remove() { this.isConnected = false; if (documento.activeElement === this) documento.activeElement = null; },
      addEventListener: (tipo, fn) => escuchas.set(tipo, fn),
      removeEventListener: (tipo) => escuchas.delete(tipo),
      replaceChildren() { this.innerHTML = ""; this.hijos = []; },
      append(hijo) { this.hijos.push(hijo); },
      querySelector: () => null, escuchas,
    };
  };
  const raiz = {
    ownerDocument: documento,
    set innerHTML(valor) {
      if (zona) zona.isConnected = false;
      zona = valor.includes("ct-exp-contenido") ? nodo() : null;
    },
    get innerHTML() { return ""; },
    querySelector(selector) { return selector === ".ct-exp-contenido" ? zona : null; },
    querySelectorAll: () => [], contains: () => true,
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
  };
  return { raiz, zona: () => zona, documento };
}

async function escenario(capacidad, disponibilidad = true) {
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: {
    async consultarCuadroRRHH() { return { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
      generada_en: "2026-09-03T09:05:00Z", expedientes: [resumen], hay_mas: false }; },
    async consultarDetalleRRHH() { return { esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen,
      solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion",
        periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
      hitos: [{ secuencia: 1, version_expediente: 8, accion_clave: "registrar_solicitud",
        realizada_en: "2026-09-03T09:00:00Z", fase_destino: "nombramiento",
        estado_origen: "en_curso", estado_destino: "en_curso" }] }; },
  } });
  await fuente.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  await presentador.cargar();
  await presentador.seleccionarExpediente(REF, "expediente");
  const dom = crearDOM();
  let lecturas = 0, capacidades = 0;
  const modulo = await montarModuloContratacionTemporal({ raiz: dom.raiz, presentador,
    resolverDisponibilidadOpcional: (clave, contexto) => clave === "reincorporacion_titular" && disponibilidad
      ? (typeof disponibilidad === "object" ? disponibilidad : { disponible: true, ...contexto }) : null,
    llamamiento: { cliente: {
      seguimientoCese: { async consultarSeguimientoCese() {
        lecturas++;
        if (lecturas === 1) throw Object.assign(new Error("temporal"), { estado: 503 });
        return seguimiento;
      } },
      reincorporacionTitular: {
        async consultarCapacidadReincorporacion(expediente, { signal }) {
          capacidades++;
          assert.deepEqual(expediente, { expediente_ref: REF, version_esperada: 8 });
          assert.ok(signal instanceof AbortSignal);
          return capacidad(signal);
        },
        registrarReincorporacion() { assert.fail("la prueba no registra efectos"); },
      },
    } },
  });
  await esperar();
  return { modulo, dom, lecturas: () => lecturas, capacidades: () => capacidades };
}

test("la lectura de cese no consulta capacidad de otro expediente ni de origen desconocido", async () => {
  for (const disponibilidad of [false, { disponible: true, expediente_ref: "otro", version_observada: 8 }]) {
    const caso = await escenario(() => assert.fail("capacidad no consultable"), disponibilidad);
    try {
      const panel = caso.dom.zona()?.hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
      panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
      await esperar(); await esperar();
      assert.equal(caso.lecturas(), 2);
      assert.equal(caso.capacidades(), 0);
    } finally { caso.modulo.desmontar(); }
  }
});

test("tras 503, Reintentar recupera seguimiento y monta reincorporación con una capacidad", async () => {
  const caso = await escenario(() => true);
  try {
    const panel = caso.dom.zona()?.hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
    assert.ok(panel);
    assert.equal(caso.lecturas(), 1);
    assert.equal(caso.capacidades(), 0);
    panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
    await esperar(); await esperar();
    assert.equal(caso.lecturas(), 2);
    assert.equal(caso.capacidades(), 1);
    assert.equal(caso.dom.zona().dataset.ctCapacidadReincorporacion, "permitida");
    assert.ok(caso.dom.zona().hijos.some((h) => h.innerHTML.includes("data-ct-rrhh-reincorporacion")));
    assert.equal(caso.dom.documento.activeElement, null, "la consulta de fondo no mueve el foco");
  } finally { caso.modulo.desmontar(); }
});

test("si el reintento responde sin capacidad, el estado visible conserva el foco", async () => {
  let intento = 0;
  const caso = await escenario(() => {
    intento++;
    if (intento === 1) throw Object.assign(new Error("temporal"), { estado: 503 });
    return false;
  });
  try {
    const panel = caso.dom.zona().hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
    panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
    await esperar(); await esperar();
    const aviso = caso.dom.zona().hijos.find((h) => h.className === "ct-exp-mensaje ct-tono-peligro");
    caso.dom.documento.activeElement = aviso.hijos[1];
    aviso.hijos[1].escuchas.get("click")();
    await esperar(); await esperar();
    const resultado = caso.dom.zona().hijos.find((h) => h.isConnected
      && h.textContent === "No puede registrar la reincorporación en este expediente.");
    assert.equal(caso.dom.documento.activeElement, resultado);
    assert.equal(caso.lecturas(), 2);
  } finally { caso.modulo.desmontar(); }
});

test("una respuesta tardía de capacidad tras desmontar no monta reincorporación", async () => {
  let resolver;
  const caso = await escenario(() => new Promise((resolve) => { resolver = resolve; }));
  const panel = caso.dom.zona()?.hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
  panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
  await esperar();
  assert.equal(caso.capacidades(), 1);
  const zona = caso.dom.zona();
  caso.modulo.desmontar();
  resolver(true);
  await esperar();
  assert.equal(zona.hijos.filter((h) => h.innerHTML.includes("data-ct-rrhh-reincorporacion")).length, 0);
});

test("un fallo de capacidad conserva un estado controlado sin ofrecer escritura", async () => {
  const caso = await escenario(() => Promise.reject(Object.assign(new Error("fallo"), { estado: 503 })));
  try {
    const panel = caso.dom.zona()?.hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
    panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
    await esperar(); await esperar();
    assert.equal(caso.lecturas(), 2);
    assert.equal(caso.capacidades(), 1);
    assert.equal(caso.dom.zona().dataset.ctCapacidadReincorporacion, "error");
    assert.ok(!caso.dom.zona().hijos.some((h) => h.innerHTML.includes("data-ct-rrhh-reincorporacion")));
    const aviso = caso.dom.zona().hijos.find((h) => h.className === "ct-exp-mensaje ct-tono-peligro");
    assert.match(aviso.hijos[0].textContent, /No se pudo comprobar si puede registrar la reincorporación/u);
    assert.equal(aviso.hijos[1].textContent, "Reintentar comprobación de reincorporación");
    caso.dom.documento.activeElement = aviso.hijos[1];
    aviso.hijos[1].escuchas.get("click")();
    await esperar(); await esperar();
    const repetido = caso.dom.zona().hijos.find((h) => h.isConnected && h.className === "ct-exp-mensaje ct-tono-peligro");
    assert.equal(caso.dom.documento.activeElement, repetido.hijos[1]);
    assert.equal(caso.lecturas(), 2);
  } finally { caso.modulo.desmontar(); }
});

test("reintentar capacidad tras 503 reutiliza el seguimiento leído y llega al panel", async () => {
  let intento = 0;
  const caso = await escenario(() => {
    intento++;
    if (intento === 1) throw Object.assign(new Error("temporal"), { estado: 503 });
    return true;
  });
  try {
    const panel = caso.dom.zona().hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
    panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
    await esperar(); await esperar();
    const aviso = caso.dom.zona().hijos.find((h) => h.className === "ct-exp-mensaje ct-tono-peligro");
    assert.ok(aviso);
    caso.dom.documento.activeElement = aviso.hijos[1];
    aviso.hijos[1].escuchas.get("click")();
    const comprobando = caso.dom.zona().hijos.find((h) => h.isConnected
      && h.textContent === "Comprobando si puede registrar la reincorporación.");
    assert.equal(caso.dom.documento.activeElement, comprobando);
    await esperar(); await esperar();
    assert.equal(caso.lecturas(), 2, "el botón de capacidad no repite el POST de seguimiento");
    assert.equal(caso.capacidades(), 2);
    assert.equal(caso.dom.zona().dataset.ctCapacidadReincorporacion, "permitida");
    const formulario = caso.dom.zona().hijos.find((h) => h.innerHTML.includes("data-ct-rrhh-reincorporacion"));
    assert.ok(formulario);
    assert.equal(caso.dom.documento.activeElement, formulario);
  } finally { caso.modulo.desmontar(); }
});

test("si el foco se mueve durante el reintento, la respuesta no lo recupera", async () => {
  let intento = 0, resolver;
  const caso = await escenario(() => {
    intento++;
    if (intento === 1) throw Object.assign(new Error("temporal"), { estado: 503 });
    return new Promise((resolve) => { resolver = resolve; });
  });
  try {
    const panel = caso.dom.zona().hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
    panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
    await esperar(); await esperar();
    const aviso = caso.dom.zona().hijos.find((h) => h.className === "ct-exp-mensaje ct-tono-peligro");
    caso.dom.documento.activeElement = aviso.hijos[1];
    aviso.hijos[1].escuchas.get("click")();
    await esperar();
    const otro = caso.dom.raiz.ownerDocument.createElement("button");
    otro.focus();
    resolver(true);
    await esperar();
    assert.equal(caso.dom.documento.activeElement, otro);
    assert.equal(caso.lecturas(), 2);
  } finally { caso.modulo.desmontar(); }
});

test("un 403 de capacidad explica la denegación y no ofrece reintentar", async () => {
  let intento = 0;
  const caso = await escenario(() => {
    intento++;
    return Promise.reject(Object.assign(new Error("denegado"), { estado: intento === 1 ? 503 : 403 }));
  });
  try {
    const panel = caso.dom.zona().hijos.find((h) => h.innerHTML.includes("data-ct-seg-reintentar"));
    panel.escuchas.get("click")({ target: { closest: () => ({ matches: (selector) => selector === "[data-ct-seg-reintentar]" }) } });
    await esperar(); await esperar();
    const primero = caso.dom.zona().hijos.find((h) => h.className === "ct-exp-mensaje ct-tono-peligro");
    caso.dom.documento.activeElement = primero.hijos[1];
    primero.hijos[1].escuchas.get("click")();
    await esperar(); await esperar();
    const aviso = caso.dom.zona().hijos.find((h) => h.isConnected && h.className === "ct-exp-mensaje ct-tono-peligro");
    assert.match(aviso.hijos[0].textContent, /No tiene permiso para consultar la reincorporación/u);
    assert.equal(aviso.hijos.length, 1);
    assert.equal(caso.dom.documento.activeElement, aviso);
    assert.equal(caso.capacidades(), 2);
    assert.equal(caso.lecturas(), 2);
  } finally { caso.modulo.desmontar(); }
});
