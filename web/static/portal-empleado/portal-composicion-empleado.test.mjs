import assert from "node:assert/strict";
import test from "node:test";

import {
  componerCronosVisible,
  componerDietasInternas,
  componerDietasVisible,
  componerPersonalVisible,
} from "./portal-composicion-empleado.js";

test("Cronos solo compone el recorrido visible y falla cerrado sin él", () => {
  const montar = () => {};
  assert.deepEqual(componerCronosVisible({ recorridos: { montarVistaRecorridosCronos: montar } }), { montar });
  assert.equal(componerCronosVisible({
    presentador: { crearPresentadorCronos() { throw new Error("no debe usar presentador de demostración"); } },
  }), undefined);
});

test("Dietas visible compone exclusivamente cálculo, mapa y recorrido sin cliente de borradores", () => {
  const llamadas = [];
  const calculador = { tipo: "calculador" };
  const visorRuta = { tipo: "visor" };
  const montaje = componerDietasVisible({
    calculador: { crearCalculadorRutasDietasPresentacionOSRM(entrada) { llamadas.push(["calculador", entrada]); return calculador; } },
    mapa: { crearVisorRutaDietas(entrada) { llamadas.push(["mapa", entrada]); return visorRuta; } },
    vista: { montarVistaItinerarioDietas(entrada) { llamadas.push(["itinerario", entrada]); } },
    recorridos: { montarVistaRecorridosDietas(raiz, entrada) { llamadas.push(["recorridos", raiz, entrada]); } },
  }, { actor: "contexto-real" }, { ruta: "consulta" }, { fetch() {} });

  assert.equal(montaje.calculador, calculador);
  assert.equal(montaje.visorRuta, visorRuta);
  assert.equal(llamadas[0][1].contextoActor.actor, "contexto-real");
  assert.equal(llamadas[0][1].capacidades.ruta, "consulta");
  assert.equal("clienteBorradores" in llamadas[0][1], false);
  montaje.montar({ raiz: "raiz", anunciar: "anunciar", registrarDesmontar: "registro" });
  assert.equal(llamadas[2][0], "recorridos");
  assert.equal("clienteBorradores" in llamadas[2][2], false);
  llamadas[2][2].montarItinerario("hueco");
  assert.deepEqual(llamadas[3][1], {
    raiz: "hueco", calculador, visorRuta, anunciar: "anunciar",
    centroSalidaAsociado: { etiqueta: "Sede provincial · Granada", localidad: "Granada" },
  });
});

test("Dietas interna muestra la zona cartográfica cerrada sin derivar identidad ni rutas del catálogo", () => {
  const llamadas = [];
  const cliente = Object.freeze({ listar() {}, crear() {} });
  const dietas = componerDietasInternas({
    contrato: Object.freeze({}),
    clienteBorradores: {
      crearClienteBorradoresDietasHTTP(entrada) {
        llamadas.push(["cliente", entrada]);
        return cliente;
      },
    },
    recorridos: {
      montarVistaRecorridosDietas(raiz, entrada) {
        llamadas.push(["recorridos", raiz, entrada]);
        return Object.freeze({ desmontar() {} });
      },
    },
    vista: {
      montarVistaItinerarioPendienteDietas(entrada) {
        llamadas.push(["itinerario-pendiente", entrada]);
        return Object.freeze({ desmontar() {} });
      },
    },
    calculador: { crearCalculadorRutasDietasHTTP() { assert.fail("no debe derivar ContextoActor"); } },
  }, { fetch() {} });

  assert.strictEqual(dietas.clienteBorradores, cliente);
  assert.equal(typeof llamadas[0][1].fetchImpl, "function");
  const resultado = dietas.montar({ raiz: "raiz", anunciar: () => {}, registrarDesmontar: () => {} });
  assert.equal(typeof resultado.desmontar, "function");
  assert.strictEqual(llamadas[1][2].clienteBorradores, cliente);
  assert.equal(typeof llamadas[1][2].montarItinerario, "function");
  llamadas[1][2].montarItinerario("hueco");
  assert.deepEqual(llamadas[2][1], { raiz: "hueco" });
});

test("Dietas interna usa el calculador ya autorizado cuando la raíz lo inyecta", () => {
  const llamadas = [];
  const calculador = Object.freeze({ obtenerCatalogo() {}, calcular() {} });
  const visorRuta = Object.freeze({ montar() {} });
  const dietas = componerDietasInternas({
    contrato: Object.freeze({}),
    clienteBorradores: { crearClienteBorradoresDietasHTTP() { return Object.freeze({}); } },
    recorridos: { montarVistaRecorridosDietas(_raiz, entrada) { llamadas.push(["recorridos", entrada]); return Object.freeze({ desmontar() {} }); } },
    vista: {
      montarVistaItinerarioDietas(entrada) { llamadas.push(["itinerario", entrada]); return Object.freeze({ desmontar() {} }); },
      montarVistaItinerarioPendienteDietas() { assert.fail("no debe mostrar el estado pendiente"); },
    },
  }, { fetch() {}, dietasItinerarioAutorizado: { calculador, visorRuta } });
  dietas.montar({ raiz: "raiz", anunciar: "anunciar" });
  llamadas[0][1].montarItinerario("hueco");
  assert.deepEqual(llamadas[1][1], { raiz: "hueco", calculador, visorRuta, anunciar: "anunciar" });
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
