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

test("B7 recupera con la misma clave un 503 posterior al commit aunque se cancele y reinicie", async () => {
  const restaurar = simularFormData();
  const fetchOriginal = globalThis.fetch;
  const envios = [];
  const huella = "b".repeat(64);
  const confirmacion = { llamamiento_ref: `llamamiento:${huella}`, recibo_ref: `recibo:llamamiento:${huella}`,
    bolsa_ref: "bolsa:01", estado: "emitido_pendiente_respuesta",
    participaciones: [candidata.participacion_ref], configuracion, emitido_en: "2026-09-23T10:00:00Z",
    reutilizada: true };
  let commitSimulado = null;
  globalThis.fetch = async (_url, opciones) => {
    envios.push({ cuerpo: opciones.body, clave: opciones.headers["Idempotency-Key"] });
    if (!commitSimulado) {
      commitSimulado = { ...confirmacion, reutilizada: false };
      return { status: 503, json: async () => ({ error: { codigo: "fallo_tras_commit" } }) };
    }
    assert.deepEqual(JSON.parse(opciones.body), JSON.parse(envios[0].cuerpo));
    return { status: 200, json: async () => ({ data: confirmacion }) };
  };
  try {
    const app = montar({ paso: 4 });
    app.submit(4); await esperar();
    assert.equal(envios.length, 1);
    assert.equal(app.flujo.recibo, "");
    assert.match(app.flujo.error, /resultado del registro es incierto/i);
    app.click("cancelar-b7");
    assert.equal(app.estado.filtrosBolsa.nuevo_llamamiento, undefined);
    app.click("iniciar-b7");
    assert.equal(app.estado.filtrosBolsa.nuevo_llamamiento, app.flujo);
    assert.equal(app.flujo.paso, 4);
    app.flujo.configuracion = { ...configuracion, asunto: "cambiado" };
    app.flujo.participaciones = ["participacion:otra"];
    app.submit(4); await esperar();
    assert.equal(envios.length, 2);
    assert.deepEqual(envios[1], envios[0]);
    assert.deepEqual(app.flujo.participaciones, [candidata.participacion_ref]);
    assert.equal(app.flujo.configuracion.asunto, configuracion.asunto);
    assert.equal(app.flujo.recibo, confirmacion.recibo_ref);
    assert.match(app.html(), new RegExp(confirmacion.recibo_ref));
  } finally { globalThis.fetch = fetchOriginal; restaurar(); }
});

for (const status of [400, 409, 422]) test(`B7 libera la clave tras rechazo definitivo ${status} y exige revisión antes de otra emisión`, async () => {
  const restaurar = simularFormData();
  const fetchOriginal = globalThis.fetch;
  const envios = [];
  const huella = (status === 400 ? "d" : status === 409 ? "e" : "f").repeat(64);
  globalThis.fetch = async (_url, opciones) => {
    envios.push({ cuerpo: opciones.body, clave: opciones.headers["Idempotency-Key"] });
    if (envios.length === 1) return { status, json: async () => ({
      error: { codigo: status === 409 ? "clave_divergente" : status === 422 ? "emision_invalida" : "solicitud_invalida" },
    }) };
    const peticion = JSON.parse(opciones.body);
    return { status: 201, json: async () => ({ data: {
      llamamiento_ref: `llamamiento:${huella}`, recibo_ref: `recibo:llamamiento:${huella}`,
      bolsa_ref: peticion.bolsa_ref, estado: "emitido_pendiente_respuesta",
      participaciones: peticion.participaciones, configuracion: peticion.configuracion,
      emitido_en: "2026-09-23T10:00:00Z", reutilizada: false,
    } }) };
  };
  try {
    const app = montar({ paso: 4 });
    app.submit(4); await esperar();
    assert.equal(envios.length, 1);
    assert.equal(app.flujo.clave_idempotencia, "");
    assert.equal(app.flujo.revision_obligatoria, true);
    assert.match(app.html(), status === 400 ? /rechazó los datos del llamamiento \(400\)/ :
      status === 409 ? /clave ya corresponde a otro llamamiento/ : /rechazó la emisión \(422\)/);
    assert.match(app.html(), /class="boton-primario" disabled aria-describedby="b7-motivo-revision"/);
    assert.match(app.html(), /id="b7-motivo-revision"[^>]*>Revise la configuración o actualice la selección/);
    assert.match(app.html(), /data-bolsa-accion="b7-revisar-configuracion"/);
    assert.match(app.html(), /data-bolsa-accion="b7-volver-seleccion"/);
    app.submit(4);
    assert.equal(envios.length, 1, "otro clic sin revisar no repite el POST");
    app.click("b7-revisar-configuracion");
    assert.equal(app.flujo.paso, 3);
    const campos = { ...configuracion, asunto: "Llamamiento revisado", cuerpo: "Texto revisado",
      confirmacion: "on" };
    globalThis.FormData = class { get(clave) { return campos[clave] ?? null; } };
    app.submit(3);
    assert.equal(app.flujo.paso, 4);
    assert.equal(app.flujo.revision_obligatoria, false);
    assert.doesNotMatch(app.html(), /aria-describedby="b7-motivo-revision"/);
    app.submit(4); await esperar();
    assert.equal(envios.length, 2);
    assert.notEqual(envios[1].clave, envios[0].clave);
    assert.notEqual(envios[1].cuerpo, envios[0].cuerpo);
    assert.equal(app.flujo.recibo, `recibo:llamamiento:${huella}`);
  } finally { globalThis.fetch = fetchOriginal; restaurar(); }
});

test("B7 permite actualizar la selección tras 422 sin repetir el POST rechazado", async () => {
  const restaurar = simularFormData();
  const fetchOriginal = globalThis.fetch;
  let posts = 0;
  globalThis.fetch = async () => {
    posts += 1;
    return { status: 422, json: async () => ({ error: { codigo: "emision_invalida" } }) };
  };
  try {
    const app = montar({ paso: 4, fuente: async () => ({ ok: true, datos }) });
    app.submit(4); await esperar();
    assert.equal(app.flujo.revision_obligatoria, true);
    app.click("b7-volver-seleccion"); await esperar();
    assert.equal(app.flujo.paso, 2);
    assert.deepEqual(app.flujo.participaciones, []);
    assert.equal(posts, 1);
    app.submit(2);
    assert.equal(app.flujo.paso, 3);
    assert.equal(app.flujo.revision_obligatoria, false);
  } finally { globalThis.fetch = fetchOriginal; restaurar(); }
});

test("B7 conserva un 201 tardío tras abrir una ficha desde avisos B5", async () => {
  const restaurar = simularFormData();
  const fetchOriginal = globalThis.fetch;
  let responder;
  const huella = "c".repeat(64);
  globalThis.fetch = async () => new Promise((resolver) => { responder = resolver; });
  try {
    const app = montar({ paso: 4, fuente: async () => ({ ok: true, datos }) });
    app.submit(4);
    app.controlador.suspenderLlamamientoB7();
    app.estado.bolsaSeleccionada = "bolsa:otra";
    app.estado.filtrosBolsa = { estado: "", texto: "" };
    app.controlador.cancelarPeticiones();
    assert.equal(app.estado.filtrosBolsa.nuevo_llamamiento, undefined);
    responder({ status: 201, json: async () => ({ data: {
      llamamiento_ref: `llamamiento:${huella}`, recibo_ref: `recibo:llamamiento:${huella}`,
      bolsa_ref: "bolsa:01", estado: "emitido_pendiente_respuesta",
      participaciones: [candidata.participacion_ref], configuracion,
      emitido_en: "2026-09-23T10:00:00Z", reutilizada: false,
    } }) });
    await esperar();
    assert.equal(app.flujo.recibo, `recibo:llamamiento:${huella}`);
    app.click("iniciar-b7"); await esperar();
    assert.equal(app.estado.bolsaSeleccionada, "bolsa:01");
    assert.equal(app.estado.filtrosBolsa.nuevo_llamamiento, app.flujo);
    assert.match(app.html(), new RegExp(`recibo:llamamiento:${huella}`));
    app.submit(4);
    assert.equal(app.flujo.enviando, false);
  } finally { globalThis.fetch = fetchOriginal; restaurar(); }
});

test("B7 deniega la recuperación del comando si la nueva lectura devuelve 403", async () => {
  const restaurar = simularFormData();
  const fetchOriginal = globalThis.fetch;
  let posts = 0;
  globalThis.fetch = async () => {
    posts += 1;
    return { status: 503, json: async () => ({ error: { codigo: "no_disponible" } }) };
  };
  try {
    const app = montar({ paso: 4, fuente: async () => ({ ok: false, status: 403, mensaje: "Denegado" }) });
    app.submit(4); await esperar();
    app.click("cancelar-b7");
    app.estado.bolsaSeleccionada = "bolsa:otra";
    app.estado.filtrosBolsa = { estado: "", texto: "" };
    app.click("iniciar-b7"); await esperar();
    assert.equal(app.estado.datosCandidatos.carga, "denegado");
    assert.equal(app.estado.datosCandidatos.datos, null);
    assert.equal(app.flujo.acceso_denegado, true);
    assert.doesNotMatch(app.html(), /Nombre privado sintético|name="participacion"/);
    app.submit(4);
    assert.equal(posts, 1, "la lectura denegada impide el replay");
  } finally { globalThis.fetch = fetchOriginal; restaurar(); }
});

test("B7 purga candidatos ante 403 del POST y conserva la clave incierta sin replay", async () => {
  const restaurar = simularFormData();
  const fetchOriginal = globalThis.fetch;
  let posts = 0;
  globalThis.fetch = async () => {
    posts += 1;
    return { status: 403, json: async () => ({ error: { codigo: "acceso_denegado" } }) };
  };
  try {
    const app = montar({ paso: 4 });
    app.submit(4); await esperar();
    assert.equal(app.estado.datosCandidatos.carga, "denegado");
    assert.equal(app.estado.datosCandidatos.datos, null);
    assert.equal(app.flujo.acceso_denegado, true);
    assert.ok(app.flujo.clave_idempotencia);
    assert.doesNotMatch(app.html(), /Nombre privado sintético|name="participacion"/);
    app.submit(4);
    assert.equal(posts, 1);
  } finally { globalThis.fetch = fetchOriginal; restaurar(); }
});
