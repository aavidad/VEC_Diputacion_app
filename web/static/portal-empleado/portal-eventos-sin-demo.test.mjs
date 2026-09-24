import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearControladorPortal } from "./portal-eventos.js";

function montarControlador({ estado, propuesta, datosPanel = { necesidades_llamamiento: [] } } = {}) {
  const escuchas = new Map();
  const anuncios = [];
  const dialogo = { open: false, addEventListener() {}, showModal() { this.open = true; } };
  const controles = {
    "dialogo-detalle": dialogo,
    "titulo-dialogo": { textContent: "" },
    "contenido-dialogo": { innerHTML: "" },
    "contenido-principal": { focus() {} },
    "boton-menu": { addEventListener() {} },
    "velo-menu": { addEventListener() {} },
    "boton-texto": { addEventListener() {} },
    "boton-contraste": { addEventListener() {} },
  };
  const documentoAnterior = globalThis.document;
  const ventanaAnterior = globalThis.window;
  globalThis.document = { addEventListener: (tipo, escuchar) => escuchas.set(tipo, escuchar) };
  globalThis.window = { addEventListener() {} };
  let repintados = 0;
  let solicitudes = 0;
  const controlador = crearControladorPortal({
    anunciar: (mensaje) => anuncios.push(mensaje),
    cerrarMenuMovil() {},
    escaparHTML: (valor) => String(valor).replaceAll("<", "&lt;").replaceAll(">", "&gt;"),
    estado,
    navegar() {},
    notaOperacionNoCompuesta: () => "Capacidad pendiente",
    obtenerDatosPanel: () => datosPanel,
    porId: (id) => controles[id],
    renderizar: () => { repintados += 1; },
    solicitarPropuestaLlamamiento: async () => { solicitudes += 1; return propuesta; },
    vistaDesdeHash: () => "portal",
  });
  controlador.instalar();
  return {
    anuncios,
    controles,
    clic(accion, extra = {}) {
      const boton = { dataset: { accion, ...extra }, textContent: accion };
      escuchas.get("click")({
        target: { closest: (selector) => selector === "[data-accion]" ? boton : null },
      });
    },
    limpiar() {
      globalThis.document = documentoAnterior;
      globalThis.window = ventanaAnterior;
    },
    get repintados() { return repintados; },
    get solicitudes() { return solicitudes; },
  };
}

test("las acciones heredadas de presentación no producen recibos ni alteran el estado", () => {
  const estado = { modoPresentacion: true, pasoLlamamiento: 4, reciboLlamamiento: null };
  const prueba = montarControlador({ estado });
  try {
    for (const accion of ["validar-recorrido", "preparar-llamamiento-demo", "operacion-presentacion"]) {
      prueba.clic(accion, { operacion: "emitir-llamamiento", objetivo: "EXP-1" });
    }
    assert.equal(estado.pasoLlamamiento, 4);
    assert.equal(estado.reciboLlamamiento, null);
    assert.equal(prueba.repintados, 0);
    assert.equal(prueba.solicitudes, 0);
    assert.equal(prueba.controles["dialogo-detalle"].open, false);
    assert.deepEqual(prueba.anuncios, []);
  } finally {
    prueba.limpiar();
  }
});

test("el asistente real limita la navegación y conserva la confirmación de servidor", async () => {
  const estado = { pasoLlamamiento: 1, confirmacionPropuestaLlamamiento: null };
  const prueba = montarControlador({
    estado,
    propuesta: { ok: true, confirmacion: { referencia: "recibo-servidor" } },
  });
  try {
    prueba.clic("solicitar-propuesta");
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(prueba.solicitudes, 1);
    assert.equal(estado.pasoLlamamiento, 2);
    assert.match(prueba.anuncios.at(-1), /Confirmación de propuesta recibida/u);

    prueba.clic("ir-paso", { paso: "3" });
    assert.equal(estado.pasoLlamamiento, 1);
    assert.match(prueba.anuncios.at(-1), /pasos posteriores no están conectados/u);
    prueba.clic("siguiente-paso");
    assert.equal(estado.pasoLlamamiento, 1);
    assert.equal(prueba.solicitudes, 1);
  } finally {
    prueba.limpiar();
  }
});

test("una respuesta sin confirmación de servidor no abre el detalle", async () => {
  const estado = { pasoLlamamiento: 1 };
  const prueba = montarControlador({
    estado,
    propuesta: { ok: true, avanzar: true, sintetica: true },
  });
  try {
    prueba.clic("solicitar-propuesta");
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(estado.pasoLlamamiento, 1);
    assert.match(prueba.anuncios.at(-1), /configuración del llamamiento permanece bloqueada/u);
  } finally {
    prueba.limpiar();
  }
});

test("seleccionar una necesidad reinicia solo el estado de navegación del llamamiento", () => {
  const estado = {
    pasoLlamamiento: 2,
    necesidadSeleccionada: "anterior",
    confirmacionPropuestaLlamamiento: { referencia: "recibo-anterior" },
    errorPropuesta: "error anterior",
  };
  const prueba = montarControlador({
    estado,
    datosPanel: { necesidades_llamamiento: [{ id: "necesidad-real" }] },
  });
  try {
    prueba.clic("seleccionar-necesidad", { id: "necesidad-real" });
    assert.equal(estado.pasoLlamamiento, 1);
    assert.equal(estado.necesidadSeleccionada, "necesidad-real");
    assert.equal(estado.confirmacionPropuestaLlamamiento, null);
    assert.equal(estado.errorPropuesta, "");
    assert.equal(prueba.solicitudes, 0);
  } finally {
    prueba.limpiar();
  }
});

test("la acción pendiente explica su denegación y escapa el motivo recibido", () => {
  const prueba = montarControlador({ estado: {} });
  try {
    prueba.clic("bloqueo-presentacion", { motivo: "<sin capacidad>" });
    assert.equal(prueba.controles["dialogo-detalle"].open, true);
    assert.match(prueba.controles["contenido-dialogo"].innerHTML, /&lt;sin capacidad&gt;/u);
    assert.doesNotMatch(prueba.controles["contenido-dialogo"].innerHTML, /<sin capacidad>/u);
    assert.equal(prueba.repintados, 0);
    assert.equal(prueba.solicitudes, 0);
  } finally {
    prueba.limpiar();
  }
});

test("el controlador distribuido carece de rutas y texto de efectos simulados", async () => {
  const codigo = await readFile(new URL("./portal-eventos.js", import.meta.url), "utf8");
  assert.doesNotMatch(codigo, /validar-recorrido|preparar-llamamiento-demo|operacion-presentacion|recibo-presentacion/u);
  assert.doesNotMatch(codigo, /modoPresentacion|ejecutarOperacionPresentacion|DEMO/u);
});
