import test from "node:test";
import assert from "node:assert/strict";
import { consultarOperacionesSituacion, crearControladorOperacionesSituacion, renderizarOperacionesSituacion } from "./portal-bolsas-operaciones.js";

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
    assert.equal(modal.operacionesB8.solicitudesError, "");
    assert.equal(modal.reincorporacionesTitular.carga, "omitida");
    assert.match(renderizarOperacionesSituacion({ candidato: modal.candidato, estado: modal.operacionesB8 }), /data-operacion="regularizar"/);
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

test("un servidor sin metadata conserva el lector anterior y sus secciones operativas", async () => {
  const data = historial("disponible"); delete data.capacidades_ficha;
  let consultas = 0;
  const { modal, controlador } = preparar({ consultarOperaciones: async () => leer(data),
    consultarDocumentales: async () => { consultas++; return { ok: true, datos: [] }; } });
  await controlador.cargar(modal, { incluirSecciones: false });
  assert.equal(consultas, 1); assert.equal(modal.operacionesB8.carga, "listo");
});
