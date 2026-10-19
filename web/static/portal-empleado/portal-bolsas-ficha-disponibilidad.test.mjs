import test from "node:test";
import assert from "node:assert/strict";
import { renderizarReincorporacionesTitular, manejarClickReincorporacionesTitular } from "./portal-bolsas-reincorporaciones.js";
import { consultarOperacionesSituacion, crearControladorOperacionesSituacion, renderizarOperacionesSituacion } from "./portal-bolsas-operaciones.js";
import { crearControladorBolsas } from "./portal-bolsas-api.js";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
import { prepararTextosPortal } from "./portal-i18n.js";
await prepararTextosPortal("bolsa");
await prepararMensajesContratos();

const bolsa = "bolsa:uno", participacion = "participacion:uno", desde = "2026-10-08T08:00:00Z";
const reglas = { ok: true, datos: { causas_baja: [], transiciones: { en_revision: ["disponible", "excluido"] } } };
const registro = (estado) => ({ estado, bolsa_ref: bolsa, participacion_ref: participacion });
const historial = (documental, reincorporaciones = "sin_montaje") => ({ esquema: "vec.bolsa.rrhh.operaciones_situacion.v1",
  items: [], cambios: [], situacion_vigente: { situacion: "en_revision", desde }, capacidades_ficha: {
    solicitudes_documentales: registro(documental), reincorporaciones_titular: registro(reincorporaciones) } });
const leer = async (data) => consultarOperacionesSituacion(bolsa, participacion,
  { fetchImpl: async () => new Response(JSON.stringify({ data }), { status: 200 }) });
const preparar = (opciones = {}) => {
  const modal = { candidato: { participacion_ref: participacion, nombre_visible: "Elena García", estado_clave: "en_revision", estado_desde: desde } };
  const estado = { bolsaSeleccionada: bolsa, modalFicha: modal };
  const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar: async () => {},
    consultarReglas: async () => reglas, ...opciones });
  return { modal, estado, controlador };
};

test("la ficha real omite solo las secciones declaradas ausentes o denegadas por el servidor", async () => {
  for (const estadoCap of ["sin_montaje", "no_autorizado"]) {
    let documentales = 0, reincorporaciones = 0;
    const res = await leer(historial(estadoCap, estadoCap));
    const { modal, controlador } = preparar({ consultarOperaciones: async () => res,
      consultarDocumentales: async () => { documentales++; return { ok: true, datos: [] }; },
      consultarReincorporaciones: async () => { reincorporaciones++; return { ok: true, datos: [] }; } });
    await controlador.cargar(modal, { incluirSecciones: false });
    assert.equal(documentales, 0); assert.equal(reincorporaciones, 0);
    assert.equal(modal.operacionesB8.carga, "listo");
    assert.match(modal.operacionesB8.solicitudesError,
      estadoCap === "sin_montaje" ? /Informática debe habilitarla/u : /no está autorizada para esta bolsa/u);
    assert.equal(modal.reincorporacionesTitular.carga, "omitida");
    assert.doesNotMatch(renderizarOperacionesSituacion({ candidato: modal.candidato, estado: modal.operacionesB8 }), /data-operacion="regularizar"/);
    assert.match(renderizarReincorporacionesTitular({ estado: modal.reincorporacionesTitular,
      escaparHTML: String }), estadoCap === "sin_montaje" ? /Informática debe habilitarlo/u : /no está autorizado para esta bolsa/u);
  }
});

test("las secciones disponibles esperan el historial nominal y se consultan una sola vez", async () => {
  let entregar, documentales = 0, reincorporaciones = 0;
  const promesa = new Promise((resolve) => { entregar = resolve; });
  const { modal, controlador } = preparar({ consultarOperaciones: () => promesa,
    consultarDocumentales: async () => { documentales++; return { ok: true, datos: [] }; },
    consultarReincorporaciones: async () => { reincorporaciones++; return { ok: true, datos: [] }; } });
  const carga = controlador.cargar(modal, { incluirSecciones: false });
  assert.equal(documentales, 0); assert.equal(reincorporaciones, 0);
  entregar(await leer(historial("disponible", "disponible"))); await carga;
  assert.equal(documentales, 1); assert.equal(reincorporaciones, 1);
  assert.equal(modal.operacionesB8.carga, "listo");
});

test("metadata de otra participación no provoca consultas opcionales ni borra el historial", async () => {
  const data = historial("disponible", "disponible");
  data.capacidades_ficha.solicitudes_documentales.participacion_ref = "participacion:otra";
  data.capacidades_ficha.reincorporaciones_titular.bolsa_ref = "bolsa:otra";
  let consultas = 0;
  const { modal, controlador } = preparar({ consultarOperaciones: async () => leer(data),
    consultarDocumentales: async () => { consultas++; }, consultarReincorporaciones: async () => { consultas++; } });
  await controlador.cargar(modal, { incluirSecciones: false });
  assert.equal(consultas, 0); assert.equal(modal.operacionesB8.carga, "listo");
  assert.equal(modal.operacionesB8.solicitudesMetadatos, true);
  assert.equal(modal.reincorporacionesTitular.carga, "metadatos");
});

test("una respuesta tardía al salir de la ficha no consulta las secciones de otro candidato", async () => {
  let entregar, consultas = 0;
  const { modal, estado, controlador } = preparar({ consultarOperaciones: () => new Promise((resolve) => { entregar = resolve; }),
    consultarDocumentales: async () => { consultas++; }, consultarReincorporaciones: async () => { consultas++; } });
  const carga = controlador.cargar(modal, { incluirSecciones: false });
  estado.modalFicha = { candidato: { participacion_ref: "participacion:otra" } };
  entregar(await leer(historial("disponible", "disponible"))); await carga;
  assert.equal(consultas, 0); assert.equal(estado.modalFicha.candidato.participacion_ref, "participacion:otra");
});

test("abrir otra ficha desde lista o histórico conserva la operación incierta y su clave", async () => {
  for (const bloqueo of ["resultadoInciertoCambioSituacion", "enviandoCambioSituacion"]) {
  const oyentes = new Map(), focos = [];
  const documento = { addEventListener(tipo, fn) { oyentes.set(tipo, [...oyentes.get(tipo) || [], fn]); },
    querySelector(selector) { return selector === "[data-directa-paso]" ? { focus: () => focos.push(selector) } : null; },
    querySelectorAll() { return []; } };
  const pendiente = Object.freeze({ clave: "clave-original", cuerpo: Object.freeze({ motivo: "Aceptación", situacion_desde_esperada: desde }),
    llamamiento: "llamamiento:original" });
  const modal = { candidato: { participacion_ref: participacion }, [bloqueo]: true,
    intentosContacto: { recibo: "recibo:contacto" }, operacionesB8: { directa: { paso: "incierto", pendiente } } };
  const estado = { bolsaSeleccionada: bolsa, modalFicha: modal, filtrosBolsa: { seguimiento: { llamamiento_ref: "llamamiento:original" }, pestana: "historico" },
    datosCandidatos: { datos: { bolsa: { bolsa_ref: bolsa }, candidatos: [{ participacion_ref: "participacion:otra" }] } } };
  const lecturas = [];
  const controlador = crearControladorBolsas({ estado, renderizar() {}, navegar() {}, documento,
    obtenerFuenteLectura: () => ({ consultarCandidatosBolsa: async (ref) => {
      lecturas.push(ref); return { ok: true, datos: { bolsa: { bolsa_ref: ref }, candidatos: [] } };
    } }) });
  controlador.abrirFicha("participacion:otra");
  assert.equal(estado.modalFicha, modal);
  controlador.instalar();
  const pulsar = (accion) => {
    const boton = { dataset: { bolsaAccion: accion, participacionRef: "participacion:otra" } };
    const evento = { target: { closest(selector) { return selector === "[data-bolsa-accion]" ? boton : null; } }, preventDefault() {} };
    for (const oyente of oyentes.get("click") || []) oyente(evento);
  };
  pulsar("abrir-ficha"); pulsar("abrir-ficha-historico");
  assert.equal(estado.modalFicha, modal);
  assert.equal(estado.modalFicha.operacionesB8.directa.pendiente, pendiente);
  assert.equal(estado.filtrosBolsa.pestana, "historico");
  assert.ok(focos.length >= 2);
  const datosAntes = estado.datosCandidatos;
  void controlador.cargarCandidatosBolsa("bolsa:otra", { cursor: "siguiente" });
  void controlador.cargarGlobalBolsa("disponibles");
  assert.equal(estado.bolsaSeleccionada, bolsa, "la bolsa activa no cambia mientras se recupera el acto");
  assert.equal(estado.datosCandidatos, datosAntes);
  assert.deepEqual(lecturas, []);
  modal[bloqueo] = false; modal.operacionesB8.directa = null; estado.filtrosBolsa = {};
  await controlador.cargarCandidatosBolsa("bolsa:otra");
  assert.equal(estado.bolsaSeleccionada, "bolsa:otra", "sin operación pendiente la navegación sigue funcionando");
  assert.deepEqual(lecturas, ["bolsa:otra"]);
  }
});

test("403 retira datos personales y conserva solo la intención hasta una lectura nominal autorizada", async () => {
  let completar, lecturas = 0;
  const datosPrevios = { carga: "listo", datos: { bolsa: { bolsa_ref: bolsa }, candidatos: [] } };
  const pendiente = { clave: "original", cuerpo: { motivo: "Aceptación", situacion_desde_esperada: desde },
    bolsa, participacion, llamamiento: "llamamiento:original" };
  const modal = { candidato: { participacion_ref: participacion }, operacionesB8: { directa: { paso: "incierto", pendiente } } };
  const estado = { bolsaSeleccionada: bolsa, modalFicha: modal, datosCandidatos: datosPrevios, filtrosBolsa: {} };
  const controlador = crearControladorBolsas({ estado, renderizar() {}, navegar() {},
    documento: { querySelector() { return null; } },
    obtenerFuenteLectura: () => ({ consultarCandidatosBolsa: () => ++lecturas === 1
      ? new Promise((resolve) => { completar = resolve; })
      : Promise.resolve({ ok: true, datos: { bolsa: { bolsa_ref: bolsa },
        candidatos: [{ participacion_ref: participacion, nombre_visible: "Elena", estado_clave: "disponible", estado_desde: desde }] } }) }) });
  const carga = controlador.cargarCandidatosBolsa("bolsa:otra");
  modal.resultadoInciertoCambioSituacion = true;
  completar({ ok: false, status: 403, mensaje: "Permiso retirado" });
  await carga;
  assert.equal(estado.modalFicha, null);
  assert.equal(estado.datosCandidatos.carga, "denegado");
  assert.equal(estado.datosCandidatos.datos, null);
  assert.equal(estado.bolsaSeleccionada, bolsa);
  await controlador.cargarCandidatosBolsa(bolsa);
  controlador.abrirFicha(participacion);
  assert.equal(estado.modalFicha.operacionesB8.directa.pendiente, pendiente);
  assert.equal(estado.modalFicha.operacionesB8.directa.paso, "incierto");
  assert.equal(estado.modalFicha.resultadoInciertoCambioSituacion, true);
});

test("401 y 403 del POST retiran la ficha y exigen nueva lectura nominal antes de repetir la misma aceptación", async () => {
  const fetchAnterior = globalThis.fetch, formAnterior = globalThis.FormData, documentoAnterior = globalThis.document;
  const llamamiento = `llamamiento:${"a".repeat(64)}`;
  const candidato = { participacion_ref: participacion, nombre_visible: "Elena García", estado_clave: "en_revision",
    estado_desde: desde, marcas: { en_revision: "aceptacion_pendiente" }, ultimo_llamamiento: { llamamiento_ref: llamamiento } };
  const contestar = (status, body) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  globalThis.FormData = class { constructor(form) { this.valores = form.valores; } get(clave) { return this.valores[clave]; } };
  try {
    for (const denegacion of [401, 403]) {
      const oyentes = new Map(), peticiones = []; let lecturas = 0;
      const documento = { addEventListener(tipo, fn) { oyentes.set(tipo, [...oyentes.get(tipo) || [], fn]); },
        querySelector() { return null; }, querySelectorAll() { return []; } };
      globalThis.document = documento;
      globalThis.fetch = async (ruta, opciones) => {
        if (ruta.endsWith("/confirmacion")) {
          peticiones.push({ ruta, opciones });
          return peticiones.length === 1
            ? contestar(denegacion, { error: { codigo: "acceso_denegado" } })
            : contestar(200, { data: { participacion_ref: participacion, situacion: "pendiente_incorporacion",
              desde: "2026-10-18T10:00:00Z", respuesta_ref: "respuesta:1", resolucion_ref: "resolucion:1",
              recibo_ref: "recibo:1", reutilizada: true, resuelta_en: "2026-10-18T10:00:00Z", motivo: "Aceptación comprobada" } });
        }
        return contestar(503, { error: { codigo: "servicio_no_disponible" } });
      };
      const estado = { bolsaSeleccionada: bolsa, filtrosBolsa: {},
        datosCandidatos: { carga: "listo", datos: { bolsa: { bolsa_ref: bolsa }, candidatos: [candidato] } },
        modalFicha: { abierto: true, bolsa: { bolsa_ref: bolsa }, candidato, operacionesB8: { carga: "listo", items: [] } } };
      const controlador = crearControladorBolsas({ estado, renderizar() {}, navegar() {}, documento,
        obtenerFuenteLectura: () => ({ consultarCandidatosBolsa: async () => {
          lecturas++;
          return lecturas === 1 ? { ok: false, status: denegacion, mensaje: "Acceso denegado" }
            : { ok: true, datos: { bolsa: { bolsa_ref: bolsa }, candidatos: [candidato] } };
        } }) });
      controlador.instalar();
      const click = (accion) => {
        const boton = { dataset: { b8Accion: accion } };
        const evento = { target: { closest: (selector) => selector === "[data-b8-accion]" ? boton : null }, preventDefault() {} };
        for (const oyente of oyentes.get("click") || []) oyente(evento);
      };
      click("directa-abrir");
      const formulario = { dataset: { b8Form: "aceptacion-directa" }, valores: { motivo: "Aceptación comprobada" } };
      for (const oyente of oyentes.get("submit") || []) oyente({ target: formulario, preventDefault() {} });
      const pendiente = estado.modalFicha.operacionesB8.directa.pendiente;
      click("directa-confirmar");
      await new Promise((resolve) => setImmediate(resolve));
      assert.equal(peticiones.length, 1);
      assert.equal(estado.modalFicha, null);
      assert.equal(estado.datosCandidatos.carga, "denegado");
      assert.equal(estado.datosCandidatos.datos, null);
      controlador.abrirFicha(participacion);
      click("directa-repetir");
      assert.equal(estado.modalFicha, null);
      assert.equal(peticiones.length, 1, "sin lectura autorizada no se repite el POST");
      await controlador.cargarCandidatosBolsa(bolsa);
      assert.equal(estado.modalFicha, null);
      assert.equal(estado.datosCandidatos.datos, null, "otro GET denegado mantiene la lista vacía");
      await controlador.cargarCandidatosBolsa(bolsa);
      controlador.abrirFicha(participacion);
      assert.equal(estado.modalFicha.operacionesB8.directa.pendiente, pendiente);
      assert.equal(estado.modalFicha.operacionesB8.directa.paso, "incierto");
      click("directa-repetir");
      await new Promise((resolve) => setImmediate(resolve));
      assert.equal(peticiones.length, 2);
      assert.equal(peticiones[1].opciones.headers["Idempotency-Key"], peticiones[0].opciones.headers["Idempotency-Key"]);
      assert.equal(peticiones[1].opciones.body, peticiones[0].opciones.body);
      assert.equal(peticiones[1].ruta, peticiones[0].ruta);
    }
  } finally { globalThis.fetch = fetchAnterior; globalThis.FormData = formAnterior; globalThis.document = documentoAnterior; }
});

test("un servidor sin metadata conserva el lector anterior y sus secciones operativas", async () => {
  const data = historial("disponible"); delete data.capacidades_ficha;
  let consultas = 0;
  const { modal, controlador } = preparar({ consultarOperaciones: async () => leer(data),
    consultarDocumentales: async () => { consultas++; return { ok: true, datos: [] }; } });
  await controlador.cargar(modal, { incluirSecciones: false });
  assert.equal(consultas, 1); assert.equal(modal.operacionesB8.carga, "listo");
});

test("una disponibilidad que falla se reconsulta desde el historial y recupera solo su sección", async () => {
  let bases = 0, documentales = 0;
  const { modal, controlador } = preparar({ consultarOperaciones: async () => leer(historial(++bases === 1 ? "indisponible" : "disponible")),
    consultarDocumentales: async () => { documentales++; return { ok: true, datos: [] }; } });
  await controlador.cargar(modal, { incluirSecciones: false });
  assert.equal(documentales, 0); assert.equal(modal.operacionesB8.solicitudesMetadatos, true);
  const control = { dataset: { b8Accion: "reintentar-solicitudes" } };
  controlador.manejarClick({ target: { closest: () => control }, preventDefault() {} });
  await new Promise(setImmediate); await new Promise(setImmediate);
  assert.equal(bases, 2); assert.equal(documentales, 1);
  assert.equal(modal.operacionesB8.solicitudesMetadatos, false);
});

test("un fallo real de una sección disponible se muestra y conserva el candidato", async () => {
  const { modal, controlador } = preparar({ consultarOperaciones: async () => leer(historial("disponible")),
    consultarDocumentales: async () => ({ ok: false, status: 503, mensaje: "No se han podido consultar los documentos. Reintente la consulta." }) });
  await controlador.cargar(modal, { incluirSecciones: false });
  assert.equal(modal.candidato.nombre_visible, "Elena García");
  assert.equal(modal.operacionesB8.carga, "listo");
  assert.equal(modal.operacionesB8.solicitudesStatus, 503);
  assert.match(renderizarOperacionesSituacion({ candidato: modal.candidato, estado: modal.operacionesB8 }), /data-b8-accion="reintentar-solicitudes"/);
});


test("reincorporaciones indisponibles muestran el error y reintentan metadata, sin una lectura ciega", () => {
  const modal = { candidato: { participacion_ref: participacion }, reincorporacionesTitular: { carga: "metadatos", items: [] } };
  const vista = renderizarReincorporacionesTitular({ estado: modal.reincorporacionesTitular, escaparHTML: String });
  assert.match(vista, /role="alert"/);
  assert.match(vista, /data-reincorporacion-accion="reintentar"/);
  let reintentos = 0;
  const control = { dataset: { reincorporacionAccion: "reintentar" } };
  assert.equal(manejarClickReincorporacionesTitular({ target: { closest: () => control }, preventDefault() {} },
    { estado: { modalFicha: modal }, renderizar() {}, reintentarMetadatos: () => { reintentos++; } }), true);
  assert.equal(reintentos, 1);
});

test("una reincorporación lenta no retrasa el historial principal ya leído", async () => {
  let entregar;
  const pendiente = new Promise((resolve) => { entregar = resolve; });
  const { modal, controlador } = preparar({ consultarOperaciones: async () => leer(historial("sin_montaje", "disponible")),
    consultarReincorporaciones: () => pendiente });
  await controlador.cargar(modal, { incluirSecciones: false });
  assert.equal(modal.operacionesB8.carga, "listo");
  assert.equal(modal.reincorporacionesTitular.carga, "cargando");
  entregar({ ok: true, datos: [] }); await modal.promesaReincorporaciones;
  assert.equal(modal.reincorporacionesTitular.carga, "listo");
});
