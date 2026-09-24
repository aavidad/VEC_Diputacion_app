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

test("Dietas interna inyecta ambos clientes HTTP sin derivados de presentación", () => {
  const llamadas = [];
  const cliente = Object.freeze({ listar() {}, crear() {} });
  const asignacion = Object.freeze({ obtener() {} });
  const dietas = componerDietasInternas({
    contrato: Object.freeze({}),
    clienteBorradores: {
      crearClienteBorradoresDietasHTTP(entrada) {
        llamadas.push(["cliente", entrada]);
        return cliente;
      },
    },
    clienteAsignacion: {
      crearClienteAsignacionDietasHTTP(entrada) {
        llamadas.push(["asignacion", entrada]);
        return asignacion;
      },
    },
    recorridos: {
      montarVistaRecorridosDietas(raiz, entrada) {
        llamadas.push(["recorridos", raiz, entrada]);
        return Object.freeze({ desmontar() {} });
      },
    },
    calculador: { crearCalculadorRutasDietasHTTP() { assert.fail("no debe derivar ContextoActor"); } },
  }, { fetch() {} });

  assert.strictEqual(dietas.clienteBorradores, cliente);
  assert.strictEqual(dietas.clienteAsignacion, asignacion);
  assert.equal(typeof llamadas[0][1].fetchImpl, "function");
  assert.equal(typeof llamadas[1][1].fetchImpl, "function");
  const resultado = dietas.montar({ raiz: "raiz", anunciar: () => {}, registrarDesmontar: () => {} });
  assert.equal(typeof resultado.desmontar, "function");
  assert.strictEqual(llamadas[2][2].clienteBorradores, cliente);
  assert.strictEqual(llamadas[2][2].clienteAsignacion, asignacion);
  assert.equal(Object.hasOwn(llamadas[2][2], "montarItinerario"), false);
});

test("Dietas interna falla cerrada sin asignación o transporte inyectado", () => {
  const recursos = {
    contrato: Object.freeze({}),
    clienteBorradores: { crearClienteBorradoresDietasHTTP() { return {}; } },
    recorridos: { montarVistaRecorridosDietas() { return { desmontar() {} }; } },
  };
  assert.equal(componerDietasInternas(recursos, { fetch() {} }), undefined);
  assert.equal(componerDietasInternas({ ...recursos, clienteAsignacion: {
    crearClienteAsignacionDietasHTTP() { return {}; },
  } }, {}), undefined);
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
