import test from "node:test";
import assert from "node:assert/strict";
import {
  causasBaja,
  consultarReglasSituacion,
  destinosSituacion,
  fechaDisponiblePropuesta,
  motivoConCausa,
  renderizarCamposReposicion,
  textoProcedenciaReposicion,
} from "./portal-bolsas-reglas-situacion.js";
import { crearControladorOperacionesSituacion, renderizarOperacionesSituacion } from "./portal-bolsas-operaciones.js";

const procedencia = (articulo) => ({ clave: "b27.causa_baja.x", referencia: "vec.bolsa.reglas:1:x", articulo, norma: "Reglamento", ejemplo: false });

const REGLAS_CON_CATALOGO = Object.freeze({
  esquema: "vec.bolsa.rrhh.reglas_situacion.v1",
  configuradas: true,
  transiciones: { renuncia: ["excluido"], trabajando: ["disponible", "excluido", "disponible_desde"] },
  causas_baja: [
    { codigo: "no_acepta", etiqueta: "No acepta el llamamiento", procedencia: procedencia("art. 11.1.a") },
    { codigo: "renuncia_nombramiento", etiqueta: "Renuncia a un nombramiento o contrato vigente", procedencia: procedencia("art. 11.2") },
  ],
  reposicion: {
    modalidades: [{ codigo: "acumulacion_tareas", meses: 9 }],
    propuesta: {
      fecha_disponible: "2099-06-30T22:00:00Z", ultimo_dia_no_disponible: "2099-06-30", meses: 5,
      procedencia: { clave: "b14.reposicion_general", referencia: "vec.bolsa.reglas:1:b14.reposicion_general", articulo: "art. 9.1", norma: "Reglamento", ejemplo: false },
    },
  },
});

const REGLAS_SIN_CATALOGO = Object.freeze({
  esquema: "vec.bolsa.rrhh.reglas_situacion.v1", configuradas: false,
  transiciones: { renuncia: ["disponible", "excluido"] }, causas_baja: [], reposicion: null,
});

const escapar = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;");

test("consulta las reglas con la fecha de fin y valida el contrato", async () => {
  let pedida = "";
  const res = await consultarReglasSituacion({ finRelacion: "2026-01-31", modalidad: "acumulacion_tareas", fetchImpl: async (url, opciones) => {
    pedida = url;
    assert.equal(opciones.method, "GET");
    assert.equal(opciones.credentials, "same-origin");
    return { ok: true, json: async () => ({ data: REGLAS_CON_CATALOGO }) };
  } });
  assert.equal(pedida, "/api/vec/bolsa/reglas-situacion?fin_relacion=2026-01-31&modalidad=acumulacion_tareas");
  assert.equal(res.ok, true);
  const invalida = await consultarReglasSituacion({ fetchImpl: async () => ({ ok: true, json: async () => ({ data: { ...REGLAS_CON_CATALOGO, causas_baja: [{ codigo: "X" }] } }) }) });
  assert.equal(invalida.ok, false);
  const caida = await consultarReglasSituacion({ fetchImpl: async () => ({ ok: false, status: 503 }) });
  assert.equal(caida.ok, false);
  assert.equal((await consultarReglasSituacion({ finRelacion: "31/01/2026", fetchImpl: () => assert.fail("no debe pedirse") })).ok, false);
});

test("sin catálogo los destinos, la fecha y el motivo siguen como antes", () => {
  assert.deepEqual(destinosSituacion(null, "renuncia"), ["disponible", "excluido"]);
  assert.deepEqual(destinosSituacion(REGLAS_SIN_CATALOGO, "renuncia"), ["disponible", "excluido"]);
  assert.deepEqual(causasBaja(REGLAS_SIN_CATALOGO), []);
  assert.equal(renderizarCamposReposicion({ reglas: REGLAS_SIN_CATALOGO, candidato: { estado_clave: "trabajando" }, escaparHTML: escapar }), "");
  assert.equal(fechaDisponiblePropuesta(REGLAS_SIN_CATALOGO), "");
});

test("con catálogo la renuncia solo va a baja y se propone la fecha de reposición", () => {
  assert.deepEqual(destinosSituacion(REGLAS_CON_CATALOGO, "renuncia"), ["excluido"]);
  const campos = renderizarCamposReposicion({ reglas: REGLAS_CON_CATALOGO, candidato: { estado_clave: "trabajando" }, escaparHTML: escapar });
  assert.match(campos, /<fieldset data-bolsa-reposicion><legend>Fin de la relación<\/legend>/);
  assert.match(campos, /<input type="date" name="fin_relacion" value="\d{4}-\d{2}-\d{2}">/);
  assert.match(campos, /<option value="acumulacion_tareas">Acumulación de tareas \(9 meses\)<\/option>/);
  assert.match(campos, /role="status" aria-live="polite" data-bolsa-procedencia-reposicion>Fecha propuesta: 5 meses de fecha a fecha \(Reglamento, art\. 9\.1\)\./);
  assert.equal(renderizarCamposReposicion({ reglas: REGLAS_CON_CATALOGO, candidato: { estado_clave: "disponible" }, escaparHTML: escapar }), "");
  assert.match(fechaDisponiblePropuesta(REGLAS_CON_CATALOGO), /^2099-0[67]-\d{2}T\d{2}:\d{2}$/);
  const pasada = { ...REGLAS_CON_CATALOGO.reposicion.propuesta, fecha_disponible: "2020-01-01T00:00:00Z" };
  assert.match(textoProcedenciaReposicion(pasada), /La fecha ya ha pasado: elija «Disponible»\./);
});

test("la causa de baja se registra con su referencia y «otro motivo» exige texto", () => {
  const causas = REGLAS_CON_CATALOGO.causas_baja;
  assert.equal(motivoConCausa(causas, "no_acepta", ""), "No acepta el llamamiento (Reglamento, art. 11.1.a)");
  assert.equal(motivoConCausa(causas, "no_acepta", "Llamada del 3 de marzo"), "No acepta el llamamiento (Reglamento, art. 11.1.a). Llamada del 3 de marzo");
  assert.equal(motivoConCausa(causas, "otro", "Motivo libre"), "Motivo libre");
  assert.equal(motivoConCausa(causas, "otro", " "), "");
  assert.equal(motivoConCausa(causas, "", "texto"), "");
  assert.equal(motivoConCausa(causas, "no_acepta", "x".repeat(1000)), "");
});

test("la exclusión B8 ofrece las causas del catálogo y compone el motivo", () => {
  const candidato = { estado_clave: "renuncia" };
  const conCausas = renderizarOperacionesSituacion({ candidato, estado: { carga: "listo", items: [], paso: 1, operacion: "excluir", causasBaja: REGLAS_CON_CATALOGO.causas_baja, formulario: {} }, escaparHTML: escapar });
  assert.match(conCausas, /<select name="causa" required><option value="">Seleccione una causa<\/option><option value="no_acepta">No acepta el llamamiento · Reglamento, art\. 11\.1\.a<\/option>/);
  assert.match(conCausas, /<option value="otro">Otro motivo<\/option>/);
  const sinCausas = renderizarOperacionesSituacion({ candidato, estado: { carga: "listo", items: [], paso: 1, operacion: "excluir", formulario: {} }, escaparHTML: escapar });
  assert.doesNotMatch(sinCausas, /name="causa"/);
  assert.match(sinCausas, /<textarea name="motivo" required minlength="2"/);
  const pausa = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], paso: 1, operacion: "pausar", causasBaja: REGLAS_CON_CATALOGO.causas_baja, formulario: {} }, escaparHTML: escapar });
  assert.doesNotMatch(pausa, /name="causa"/);

  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.valores = f.valores; } get(c) { return this.valores[c] ?? null; } };
  try {
    const flujo = { carga: "listo", items: [], paso: 1, operacion: "excluir", causasBaja: REGLAS_CON_CATALOGO.causas_baja, formulario: {} };
    const estado = { modalFicha: { candidato, operacionesB8: flujo } };
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {} });
    const enviar = (valores) => controlador.manejarSubmit({ target: { closest: () => ({ valores }) }, preventDefault() {} });
    enviar({ causa: "otro", motivo: "" });
    assert.equal(flujo.paso, 1);
    assert.match(flujo.errorFormulario, /Elija una causa/);
    enviar({ causa: "renuncia_nombramiento", motivo: "" });
    assert.equal(flujo.paso, 2);
    assert.equal(flujo.formulario.motivo, "Renuncia a un nombramiento o contrato vigente (Reglamento, art. 11.2)");
  } finally { globalThis.FormData = anterior; }
});

test("al abrir la ficha pide las reglas con el historial y guarda causas y propuesta", async () => {
  const pedidas = [];
  const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: null };
  const modal = { candidato: { participacion_ref: "participacion:uno", estado_clave: "trabajando" } };
  estado.modalFicha = modal;
  const anteriorFetch = globalThis.fetch;
  globalThis.fetch = async () => ({ ok: true, json: async () => ({ data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [] } }) });
  try {
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {}, consultarReglas: async (opciones) => {
      pedidas.push(opciones.finRelacion);
      return { ok: true, datos: REGLAS_CON_CATALOGO };
    } });
    await controlador.cargar(modal);
  } finally { globalThis.fetch = anteriorFetch; }
  assert.match(pedidas[0], /^\d{4}-\d{2}-\d{2}$/);
  assert.equal(modal.reglasSituacion, REGLAS_CON_CATALOGO);
  assert.equal(modal.operacionesB8.causasBaja.length, 2);
});
