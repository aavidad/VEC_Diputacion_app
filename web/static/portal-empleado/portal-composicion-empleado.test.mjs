import assert from "node:assert/strict";
import test from "node:test";

import {
  componerCronosVisible,
  componerDietasInternas,
  componerPersonalVisible,
} from "./portal-composicion-empleado.js";

test("Cronos solo compone el recorrido visible y falla cerrado sin él", () => {
  const montar = () => {};
  assert.deepEqual(componerCronosVisible({ recorridos: { montarVistaRecorridosCronos: montar } }), { montar });
  assert.equal(componerCronosVisible({
    presentador: { crearPresentadorCronos() { throw new Error("no debe usar presentador de demostración"); } },
  }), undefined);
});

function recursosDietas(llamadas, { cliente, asignacion, calculador, visor, montar }) {
  return {
    contrato: Object.freeze({}),
    clienteBorradores: { crearClienteBorradoresDietasHTTP(entrada) { llamadas.push(["cliente", entrada]); return cliente; } },
    clienteAsignacion: { crearClienteAsignacionDietasHTTP(entrada) { llamadas.push(["asignacion", entrada]); return asignacion; } },
    // El calculador solo recibe el transporte: la autorización de rutas vive en el servidor.
    calculador: { crearCalculadorRutasDietasHTTP(entrada) {
      assert.deepEqual(Object.keys(entrada), ["fetchImpl"]);
      llamadas.push(["calculador", entrada]); return calculador;
    } },
    mapa: { crearVisorRutaDietas(entrada) {
      assert.equal(entrada.permitirTeselas, true);
      llamadas.push(["visor", entrada]); return visor;
    } },
    recorridos: { montarVistaRecorridosDietas(raiz, entrada) {
      llamadas.push(["recorridos", raiz, entrada]);
      return montar?.() ?? Object.freeze({ desmontar() {} });
    } },
  };
}

test("Dietas interna inyecta clientes HTTP, ruta, mapa y relaciones acreditadas por Personal", async () => {
  const llamadas = [];
  const cliente = Object.freeze({ listar() {}, crear() {} });
  const relaciones = Object.freeze([{ relacion_ref: `rel_${"a".repeat(22)}`, unidad_ref: "U1", version: 1 }]);
  const asignacion = Object.freeze({ obtener() {}, async obtenerRelaciones({ signal }) {
    assert.equal(signal.aborted, false);
    return { relaciones_autorizadas: relaciones, fecha_referencia: "2026-09-25" };
  } });
  const calculador = Object.freeze({ obtenerCatalogo() {}, calcular() {} });
  const visor = Object.freeze({ montar() {} });
  const dietas = componerDietasInternas(recursosDietas(llamadas, { cliente, asignacion, calculador, visor }), { fetch() {} });
  assert.strictEqual(dietas.clienteBorradores, cliente);
  assert.strictEqual(dietas.clienteAsignacion, asignacion);
  assert.deepEqual(llamadas.map(([tipo]) => tipo), ["cliente", "asignacion", "calculador", "visor"]);
  llamadas.slice(0, 3).forEach(([, entrada]) => assert.equal(typeof entrada.fetchImpl, "function"));
  const resultado = await dietas.montar({ raiz: "raiz", anunciar: () => {}, registrarDesmontar: () => {} });
  assert.equal(typeof resultado.desmontar, "function");
  const [, raiz, entrada] = llamadas.at(-1);
  assert.equal(raiz, "raiz");
  assert.strictEqual(entrada.clienteBorradores, cliente);
  assert.strictEqual(entrada.clienteAsignacion, asignacion);
  assert.strictEqual(entrada.calculadorRuta, calculador);
  assert.strictEqual(entrada.visorRuta, visor);
  assert.strictEqual(entrada.relacionesAutorizadas, relaciones);
  assert.equal(entrada.fechaReferenciaPersonal, "2026-09-25");
  assert.equal(entrada.estadoRelaciones, "disponible");
  assert.equal(entrada.motivoRelaciones, undefined);
  assert.equal(Object.hasOwn(entrada, "montarItinerario"), false);
});

test("Dietas interna conserva el motivo de Personal y no monta si el portal ya salió", async () => {
  for (const [codigo, motivo] of [["empleado_no_disponible", "empleado_no_disponible"], ["empleado_ambiguo", "empleado_ambiguo"], ["acceso_denegado", undefined]]) {
    const llamadas = [];
    const asignacion = { obtener() {}, async obtenerRelaciones() { throw Object.assign(new Error("denegada"), { codigo }); } };
    const dietas = componerDietasInternas(recursosDietas(llamadas, { cliente: {}, asignacion, calculador: {}, visor: {} }), { fetch() {} });
    await dietas.montar({ raiz: "raiz", anunciar: () => {}, registrarDesmontar: () => {} });
    const entrada = llamadas.at(-1)[2];
    assert.equal(entrada.estadoRelaciones, "no_disponible");
    assert.deepEqual(entrada.relacionesAutorizadas, []);
    assert.equal(entrada.motivoRelaciones, motivo);
  }
  const llamadas = [];
  let salir;
  let senal;
  const asignacion = { obtener() {}, obtenerRelaciones({ signal }) {
    senal = signal;
    return new Promise((resolver) => { signal.addEventListener("abort", () => resolver({ relaciones_autorizadas: [], fecha_referencia: "2026-09-25" })); });
  } };
  const dietas = componerDietasInternas(recursosDietas(llamadas, { cliente: {}, asignacion, calculador: {}, visor: {} }), { fetch() {} });
  const montaje = dietas.montar({ raiz: "raiz", anunciar: () => {}, registrarDesmontar: (limpiar) => { salir ??= limpiar; } });
  salir();
  const resultado = await montaje;
  assert.equal(senal.aborted, true);
  assert.equal(llamadas.some(([tipo]) => tipo === "recorridos"), false);
  assert.equal(typeof resultado.desmontar, "function");
});

test("Dietas interna falla cerrada sin asignación, ruta, mapa o transporte inyectado", () => {
  const completos = recursosDietas([], { cliente: {}, asignacion: {}, calculador: {}, visor: {} });
  assert.notEqual(componerDietasInternas(completos, { fetch() {} }), undefined);
  for (const falta of ["clienteAsignacion", "calculador", "mapa", "recorridos"]) {
    const { [falta]: _omitido, ...recursos } = completos;
    assert.equal(componerDietasInternas(recursos, { fetch() {} }), undefined, falta);
  }
  assert.equal(componerDietasInternas(completos, {}), undefined);
});

test("Personal monta ficha antes de crear catálogos y limpia registros tempranos y tardíos una sola vez", async () => {
  const clientes = [];
  const limpiezas = [0, 0, 0];
  const pendientes = [];
  let montarCatalogos;
  let desmontajeFicha = 0;
  const personal = componerPersonalVisible({
    ficha: { montarVistaFichaIntegralPersonal(entrada) { assert.deepEqual(entrada.fuentes, {}); montarCatalogos = entrada.montarCatalogos; return { desmontar() { desmontajeFicha += 1; } }; } },
    clienteCategorias: { crearClienteHTTPCategoriasPersonal() { clientes.push("categorias"); return {}; } },
    clienteRPT: { crearClienteHTTPRPTPublica() { clientes.push("rpt"); return {}; } },
    clienteEstructura: { crearClienteHTTPEstructuraOrganizativaPublica() { clientes.push("estructura"); return {}; } },
    vistaCategorias: { montarModuloPersonal(entrada) { const limpiar = () => { limpiezas[0] += 1; }; entrada.registrarDesmontar(limpiar); return new Promise((resolve) => { pendientes[0] = () => resolve({ desmontar: limpiar }); }); } },
    vistaRPT: { montarModuloRPTPublica(entrada) { const limpiar = () => { limpiezas[1] += 1; }; entrada.registrarDesmontar(limpiar); return new Promise((resolve) => { pendientes[1] = () => resolve({ desmontar: limpiar }); }); } },
    vistaEstructura: { montarModuloEstructuraOrganizativaPublica(entrada) { const limpiar = () => { limpiezas[2] += 1; }; entrada.registrarDesmontar(limpiar); return new Promise((resolve) => { pendientes[2] = () => resolve({ desmontar: limpiar }); }); } },
  }, { fetch() {} });

  const globales = [];
  const ficha = personal.montar({ raiz: {}, anunciar() {}, registrarDesmontar: (limpiar) => globales.push(limpiar) });
  assert.equal(typeof montarCatalogos, "function");
  assert.deepEqual(clientes, []);

  const catalogos = montarCatalogos({ raiz: {}, anunciar() {}, registrarDesmontar: (limpiar) => globales.push(limpiar) });
  await Promise.resolve();
  assert.deepEqual(clientes, ["categorias", "rpt", "estructura"]);
  globales[0]();
  pendientes.forEach((resolver) => resolver());
  const resultado = await catalogos;
  resultado.desmontar();
  ficha.desmontar();
  assert.deepEqual(limpiezas, [1, 1, 1]);
  assert.equal(desmontajeFicha, 1);
});

test("Personal limpia una vez también si un catálogo falla después de registrar temprano", async () => {
  let limpiarTemprano = 0;
  let resolverTardio;
  const directo = componerPersonalVisible({
    clienteCategorias: { crearClienteHTTPCategoriasPersonal() { return {}; } }, clienteRPT: { crearClienteHTTPRPTPublica() { return {}; } }, clienteEstructura: { crearClienteHTTPEstructuraOrganizativaPublica() { return {}; } },
    vistaCategorias: { montarModuloPersonal(entrada) { const limpiar = () => { limpiarTemprano += 1; }; entrada.registrarDesmontar(limpiar); return new Promise((resolve) => { resolverTardio = () => resolve({ desmontar: limpiar }); }); } },
    vistaRPT: { montarModuloRPTPublica() { return Promise.reject(new Error("fallo controlado")); } }, vistaEstructura: { montarModuloEstructuraOrganizativaPublica() { return Promise.resolve({ desmontar() {} }); } },
  }, {});
  await assert.rejects(directo.montar({ raiz: {}, anunciar() {} }), /fallo controlado/);
  resolverTardio();
  await Promise.resolve();
  assert.equal(limpiarTemprano, 1);
});
