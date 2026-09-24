import test from "node:test";
import assert from "node:assert/strict";
import { crearControladorBolsas } from "./portal-bolsas-api.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

const candidata = { participacion_ref: "participacion:001", nombre_visible: "Nombre privado sintético", estado_clave: "disponible", orden: 1 };
const datos = { generado_en: "2026-09-23T10:00:00Z", bolsa: { bolsa_ref: "bolsa:01", categoria: "Auxiliar", total: 1, por_estado: { disponible: 1 } }, candidatos: [candidata], contactos: [], hay_mas: false, cursor_siguiente: null };
const configuracion = { referencia: "NEC-01", descripcion: "Cobertura", categoria: "Auxiliar", centro: "Centro", modalidad: "Sustitución", fecha_inicio: "2026-10-01", plazo: "48 horas", plantilla_version: "bolsa-llamamiento-v1", asunto: "Llamamiento", cuerpo: "Texto del correo" };
const esperar = () => new Promise(setImmediate);

function montar({ fuente, paso = 2 } = {}) {
  const escuchas = {};
  const form = { dataset: { cantidad: "1" } };
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector(selector) { return selector === '[data-bolsa-form="b7-paso2"]' ? form : null; } };
  const flujo = { paso, estados: ["disponible"], participaciones: paso === 4 ? [candidata.participacion_ref] : [], configuracion, error: "", recibo: "" };
  const estado = { bolsaSeleccionada: "bolsa:01", filtrosBolsa: { estado: "", texto: "", nuevo_llamamiento: flujo }, datosCandidatos: { carga: "listo", datos, error: "" } };
  const controlador = crearControladorBolsas({ estado, renderizar() {}, navegar() {}, documento, obtenerFuenteLectura: () => ({ consultarCandidatosBolsa: fuente }) });
  controlador.instalar();
  const click = (accion) => escuchas.click({ preventDefault() {}, target: { closest(selector) { return selector === "[data-bolsa-accion]" ? { dataset: { bolsaAccion: accion } } : null; } } });
  const submit = (numero) => escuchas.submit({ preventDefault() {}, target: { closest(selector) { return selector === `[data-bolsa-form="b7-paso${numero}"]` ? form : null; } } });
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "info", etiquetaClave: (v) => v, encabezadoVista: () => "<header>Bolsa</header>", escaparHTML: (v) => String(v ?? ""), numero: (v) => String(v ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => estado.datosCandidatos, obtenerEstadoCandidatos: () => estado.filtrosBolsa,
  });
  return { estado, flujo, controlador, click, submit, html: () => presentador.renderizarVista("bolsa-candidatos") };
}

function simularFormData() {
  const original = globalThis.FormData;
  globalThis.FormData = class { getAll(clave) { return clave === "estado" ? ["disponible"] : clave === "participacion" ? [candidata.participacion_ref] : []; } get(clave) { return clave === "confirmacion" ? "on" : null; } };
  return () => { globalThis.FormData = original; };
}

for (const status of [401, 403]) test(`B7 retira los datos y bloquea avance tras selección denegada ${status}`, async () => {
  const restaurar = simularFormData();
  try {
    let permitida = false;
    const app = montar({ fuente: async () => permitida ? { ok: true, datos } : { ok: false, status, mensaje: "Denegado" } });
    assert.match(app.html(), /Nombre privado sintético/);
    app.click("b7-seleccionar-todas"); await esperar();
    assert.equal(app.estado.datosCandidatos.carga, "denegado");
    assert.equal(app.estado.datosCandidatos.datos, null);
    assert.doesNotMatch(app.html(), /Nombre privado sintético|name="participacion"/);
    app.submit(2);
    assert.equal(app.flujo.paso, 2);
    assert.deepEqual(app.flujo.participaciones, []);
    permitida = true;
    await app.controlador.cargarCandidatosBolsa("bolsa:01");
    app.submit(2);
    assert.equal(app.flujo.paso, 3, "solo una lectura autorizada nueva permite seleccionar de nuevo");
  } finally { restaurar(); }
});

test("B7 invalida una lectura concurrente que llega después de la denegación", async () => {
  const restaurar = simularFormData();
  try {
    let resolverAnterior;
    let llamadas = 0;
    const app = montar({ fuente: async () => ++llamadas === 1 ? new Promise((resolver) => { resolverAnterior = resolver; }) : { ok: false, status: 403, mensaje: "Denegado" } });
    const pendiente = app.controlador.cargarCandidatosBolsa("bolsa:01");
    app.click("b7-seleccionar-todas"); await esperar();
    resolverAnterior({ ok: true, datos }); await pendiente;
    assert.equal(app.estado.datosCandidatos.carga, "denegado");
    assert.doesNotMatch(app.html(), /Nombre privado sintético|name="participacion"/);
  } finally { restaurar(); }
});

test("B7 conserva el comando y la clave tras navegar durante POST503 y reintenta exactamente una vez", async () => {
  const restaurar = simularFormData();
  const fetchOriginal = globalThis.fetch;
  const envios = [];
  let responder;
  globalThis.fetch = async (_url, opciones) => { envios.push({ cuerpo: opciones.body, clave: opciones.headers["Idempotency-Key"] }); return new Promise((resolver) => { responder = resolver; }); };
  try {
    const app = montar({ paso: 4 });
    app.submit(4); app.controlador.cancelarPeticiones();
    responder({ status: 503, json: async () => ({ error: { codigo: "no_disponible" } }) }); await esperar();
    app.controlador.cancelarPeticiones();
    assert.deepEqual(app.flujo.participaciones, [candidata.participacion_ref]);
    assert.equal(envios.length, 1, "navegar no repite el POST");
    app.submit(4);
    assert.equal(envios.length, 2);
    assert.deepEqual(envios[1], envios[0], "el reintento conserva clave y cuerpo exactos");
    const huella = "a".repeat(64);
    responder({ status: 201, json: async () => ({ data: { llamamiento_ref: `llamamiento:${huella}`, recibo_ref: `recibo:llamamiento:${huella}`, bolsa_ref: "bolsa:01", estado: "emitido_pendiente_respuesta", participaciones: [candidata.participacion_ref], configuracion, emitido_en: "2026-09-23T10:00:00Z", reutilizada: true } }) }); await esperar();
    assert.equal(app.flujo.recibo, `recibo:llamamiento:${huella}`);
  } finally { globalThis.fetch = fetchOriginal; restaurar(); }
});
