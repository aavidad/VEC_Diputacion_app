import assert from "node:assert/strict";
import test from "node:test";

import * as i18nCronos from "./modulos/cronos/i18n.js";
import * as composicionEmpleado from "./portal-composicion-empleado.js";
import {
  componerCronosInterno,
  componerDietasInternas,
  componerPersonalVisible,
} from "./portal-composicion-empleado.js";

test("la composición no exporta montajes huérfanos de presentación", () => {
  assert.equal(Object.hasOwn(composicionEmpleado, "componerCronosVisible"), false);
  assert.equal(Object.hasOwn(composicionEmpleado, "componerDietasVisible"), false);
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

test("el acceso de Personal a correos usa sólo la ruta interna existente de preferencias", () => {
  let entradaFicha; let peticiones = 0; const location = { hash: "#personal" }; const focos = [];
  const personal = componerPersonalVisible({
    ficha: { montarVistaFichaIntegralPersonal(entrada) { entradaFicha = entrada; return { desmontar() {} }; } },
    clienteCategorias: { crearClienteHTTPCategoriasPersonal() { throw new Error("no debe consultar"); } },
    vistaCategorias: { montarModuloPersonal() {} },
  }, { location, document: { getElementById(id) {
    assert.equal(id, "contenido-principal"); return { focus(opciones) { focos.push(opciones); } };
  } }, fetch() { peticiones += 1; } }, { catalogosPublicos: false });
  personal.montar({ raiz: {}, anunciar() {} });
  entradaFicha.abrirCorreos();
  assert.equal(location.hash, "#mis-preferencias");
  assert.equal(peticiones, 0);
  assert.deepEqual(focos, [{ preventScroll: true }]);
  assert.deepEqual(entradaFicha.fuentes, {});
});

test("los accesos propios de Mi ficha llevan el foco al contenido estable al cambiar de vista", () => {
  let entrada; const location = { hash: "#personal" }; const focos = [];
  const personal = componerPersonalVisible({
    ficha: { montarVistaFichaIntegralPersonal(opciones) { entrada = opciones; return { desmontar() {} }; } },
    clienteCategorias: { crearClienteHTTPCategoriasPersonal() { throw new Error("sin consulta de catálogo"); } },
    vistaCategorias: { montarModuloPersonal() {} },
  }, { location, document: { getElementById(id) {
    assert.equal(id, "contenido-principal"); return { focus(opciones) { focos.push(opciones); } };
  } }, fetch() { assert.fail("navegar no consulta datos"); } }, { catalogosPublicos: false });
  personal.montar({ raiz: {} });
  for (const destino of ["cronos", "dietas"]) {
    entrada.navegarModulo(destino);
    assert.equal(location.hash, `#${destino}`);
  }
  entrada.navegarModulo("personal-registro");
  assert.equal(location.hash, "#dietas", "no abre gestión desde estos accesos");
  assert.deepEqual(focos, [{ preventScroll: true }, { preventScroll: true }]);
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

test("Personal compone solo los catálogos públicos servidos y pasa accesos y ocultación a la ficha", async () => {
  const clientes = [];
  const recursos = (entradaFicha) => ({
    ficha: { montarVistaFichaIntegralPersonal(entrada) { entradaFicha.push(entrada); return { desmontar() {} }; } },
    clienteCategorias: { crearClienteHTTPCategoriasPersonal() { clientes.push("categorias"); return {}; } },
    clienteEstructura: { crearClienteHTTPEstructuraOrganizativaPublica() { clientes.push("estructura"); return {}; } },
    vistaCategorias: { montarModuloPersonal: async () => ({ desmontar() {} }) },
    vistaEstructura: { montarModuloEstructuraOrganizativaPublica: async () => ({ desmontar() {} }) },
  });
  const entradas = [];
  let disponibles = { dietas: false, cronos: false };
  // Sin recursos RPT: basta con que el servidor no lo sirva para no exigirlos.
  const personal = componerPersonalVisible(recursos(entradas), { fetch() {} }, {
    catalogosPublicos: ["estructura"], ocultarSinFuente: true, destinosDisponibles: () => disponibles,
  });
  assert.notEqual(personal, undefined);
  disponibles = { dietas: true, cronos: false };
  personal.montar({ raiz: {}, anunciar() {} });
  assert.equal(entradas[0].ocultarSinFuente, true);
  assert.equal(entradas[0].rptDisponible, false, "una sonda RPT fallida no abre Catálogos desde la ficha");
  assert.deepEqual(entradas[0].destinosDisponibles, { dietas: true, cronos: false }, "la disponibilidad se evalúa al montar");
  await entradas[0].montarCatalogos({ raiz: {}, anunciar() {} });
  assert.deepEqual(clientes, ["categorias", "estructura"]);
  const conRPT = componerPersonalVisible({ ...recursos(entradas),
    clienteRPT: { crearClienteHTTPRPTPublica() { return {}; } },
    vistaRPT: { montarModuloRPTPublica: async () => ({ desmontar() {} }) },
  }, { fetch() {} }, { catalogosPublicos: ["rpt"], ocultarSinFuente: true });
  assert.notEqual(conRPT, undefined);
  conRPT.montar({ raiz: {}, anunciar() {} });
  assert.equal(entradas[1].rptDisponible, true, "solo la sonda RPT servida permite abrir Catálogos desde el enlace");
  // Pedir un catálogo servido sin sus recursos, o uno desconocido, no compone.
  assert.equal(componerPersonalVisible(recursos([]), {}, { catalogosPublicos: ["rpt"] }), undefined);
  assert.equal(componerPersonalVisible(recursos([]), {}, { catalogosPublicos: ["otro"] }), undefined);
  assert.equal(componerPersonalVisible(recursos([]), {}, { destinosDisponibles: {} }), undefined);
});

test("Dietas interna compone el circuito de revisión solo con su cliente HTTP same-origin", async () => {
  const llamadas = [];
  const asignacion = Object.freeze({ async obtenerRelaciones() { return { relaciones_autorizadas: [], fecha_referencia: "2026-09-25" }; } });
  const circuito = Object.freeze({ competencias() {}, listar() {}, decidir() {}, documento() {} });
  const recursos = { ...recursosDietas(llamadas, { cliente: {}, asignacion, calculador: {}, visor: {} }),
    clienteCircuito: { crearClienteCircuitoDietasHTTP(entrada) { assert.deepEqual(Object.keys(entrada), ["fetchImpl"]); return circuito; } } };
  const dietas = componerDietasInternas(recursos, { fetch() {} });
  await dietas.montar({ raiz: "raiz", anunciar: () => {}, registrarDesmontar: () => {} });
  assert.strictEqual(llamadas.at(-1)[2].clienteCircuito, circuito);
  const sinCircuito = componerDietasInternas(recursosDietas([], { cliente: {}, asignacion, calculador: {}, visor: {} }), { fetch() {} });
  assert.notEqual(sinCircuito, undefined);
});

test("Dietas entrega el cliente de rectificación propia solo tras consultar relaciones a Personal", async () => {
  const llamadas = [];
  const relaciones = [{ relacion_ref: `rel_${"a".repeat(22)}`, unidad_ref: "U1", version: 1 }];
  const asignacion = { async obtenerRelaciones() { llamadas.push(["relaciones"]); return {
    relaciones_autorizadas: relaciones, fecha_referencia: "2026-10-02",
  }; } };
  const clienteRectificacion = Object.freeze({ consultar() {}, solicitar() {} });
  const recursos = { ...recursosDietas(llamadas, { cliente: {}, asignacion, calculador: {}, visor: {} }),
    clienteRectificacion: { crearClienteRectificacionDietasHTTP({ fetchImpl }) {
      assert.equal(typeof fetchImpl, "function");
      llamadas.push(["rectificacion"]);
      return clienteRectificacion;
    } } };
  const dietas = componerDietasInternas(recursos, { fetch() {} });
  await dietas.montar({ raiz: "raiz", anunciar() {}, registrarDesmontar() {} });
  const [, , opciones] = llamadas.at(-1);
  assert.deepEqual(opciones.relacionesAutorizadas, relaciones);
  assert.strictEqual(opciones.clienteRectificacion, clienteRectificacion);
  assert.equal(Object.hasOwn(opciones, "clienteRectificacionAdmin"), false);
});

function domFalso() {
  class Nodo {
    constructor(etiqueta) { this.tagName = etiqueta; this.children = []; this.dataset = {}; this.atributos = {}; this.textContent = ""; this.parent = null; }
    get ownerDocument() { return documento; }
    append(...nodos) { for (const n of nodos) { n.parent = this; this.children.push(n); } }
    replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
    remove() { if (this.parent) this.parent.children = this.parent.children.filter((n) => n !== this); this.parent = null; }
    setAttribute(nombre, valor) { this.atributos[nombre] = String(valor); }
  }
  const documento = { createElement: (etiqueta) => new Nodo(etiqueta) };
  return new Nodo("root");
}

function recursosCronos(partes) {
  const cliente = (nombre) => ({ [nombre]: () => Object.freeze({}) });
  return {
    saldo: { montarVistaSaldoCronos: partes.saldo }, remoto: { montarVistaRemotoCronos: partes.remoto },
    movimientos: { montarVistaMovimientosCronos: partes.movimientos },
    movimientosPropios: { montarMovimientosPropiosCronos: partes.calendario },
    permisosPropios: { montarPermisosPropiosCronos: () => Object.freeze({ desmontar() {} }) },
    clienteSaldo: cliente("crearClienteSaldoCronosHTTP"), clienteRemoto: cliente("crearClienteRemotoCronosHTTP"),
    clienteSolicitudes: cliente("crearClienteSolicitudesCronosHTTP"), i18n: i18nCronos,
  };
}

test("Jornada: una parte que falla deja su aviso accesible y, sin calendario, no se ofrece el olvido", () => {
  const recibidas = {};
  const bien = (nombre, extra = {}) => (opciones) => { recibidas[nombre] = opciones; return Object.freeze({ desmontar() {}, ...extra }); };
  const falla = () => { throw new Error("detalle interno: /api/interna/x"); };
  const cronos = componerCronosInterno(recursosCronos({ saldo: falla, remoto: bien("remoto"),
    movimientos: bien("movimientos"), calendario: falla }), {});
  const raiz = domFalso();
  const montaje = cronos.montar({ raiz });
  const parte = (nombre) => raiz.children.find((n) => n.dataset.cronosParte === nombre);
  assert.deepEqual(raiz.children.map((n) => n.dataset.cronosParte ?? n.tagName), ["header", "saldo", "remoto", "movimientos", "calendario"]);
  for (const nombre of ["saldo", "calendario"]) {
    assert.equal(parte(nombre).dataset.cronosParteEstado, "error", nombre);
    const [aviso] = parte(nombre).children;
    assert.equal(aviso.atributos.role, "alert");
    assert.equal(aviso.textContent, i18nCronos.MENSAJES_CRONOS.jornada_parte_error);
    assert.doesNotMatch(aviso.textContent, /interno|api/u);
  }
  assert.equal(Object.hasOwn(recibidas.movimientos, "abrirCorreccion"), false, "sin calendario no hay olvido que abrir");
  assert.equal(recibidas.movimientos.incrustada, true);
  montaje.desmontar();
  assert.equal(raiz.children.length, 0);

  let olvidos = 0;
  const conCalendario = componerCronosInterno(recursosCronos({ saldo: bien("saldo"), remoto: bien("remoto"),
    movimientos: bien("movimientos"), calendario: bien("calendario", { abrirOlvido: () => { olvidos += 1; } }) }), {});
  conCalendario.montar({ raiz: domFalso() });
  assert.equal(recibidas.calendario.incrustada, true);
  recibidas.movimientos.abrirCorreccion();
  assert.equal(olvidos, 1);
});

test("fichaje confirmado actualiza las lecturas montadas sin sustituir el periodo ni actuar tras salir", async () => {
  const lecturas = [];
  let confirmar;
  const parte = (nombre) => () => ({ desmontar() {}, actualizar() { lecturas.push(nombre); } });
  const cronos = componerCronosInterno(recursosCronos({
    saldo: parte("saldo"), movimientos: parte("movimientos"), calendario: parte("calendario"),
    remoto: (opciones) => { confirmar = opciones.onRegistrado; return { desmontar() {} }; },
  }), {});
  const montaje = cronos.montar({ raiz: domFalso() });
  assert.deepEqual(lecturas, []);
  await confirmar();
  assert.deepEqual(lecturas, ["saldo", "movimientos", "calendario"]);
  montaje.desmontar();
  await confirmar();
  assert.equal(lecturas.length, 3);
});

test("una parte no montada no impide actualizar el resto después del fichaje", async () => {
  let confirmar;
  let lecturas = 0;
  const falla = () => { throw new Error("no disponible"); };
  const cronos = componerCronosInterno(recursosCronos({
    saldo: falla, calendario: falla,
    movimientos: () => ({ desmontar() {}, actualizar() { lecturas += 1; } }),
    remoto: (opciones) => { confirmar = opciones.onRegistrado; return { desmontar() {} }; },
  }), {});
  const montaje = cronos.montar({ raiz: domFalso() });
  await confirmar();
  assert.equal(lecturas, 1);
  montaje.desmontar();
});

test("Cronos ofrece bandeja y avisos solo con sus tres piezas y un único cliente de resolución", () => {
  const creados = []; const montados = [];
  const partes = { saldo: () => ({}), remoto: () => ({}), movimientos: () => ({}), calendario: () => ({}) };
  const conResolucion = {
    ...recursosCronos(partes),
    bandejaPermisos: { montarBandejaPermisosCronos: (o) => { montados.push(["bandeja", o]); return Object.freeze({ desmontar() {} }); } },
    avisosPropios: { montarAvisosPropiosCronos: (o) => { montados.push(["avisos", o]); return Object.freeze({ desmontar() {} }); } },
    clienteResolucion: { crearClienteResolucionCronosHTTP: (t) => { const c = Object.freeze({ t }); creados.push(c); return c; } },
    i18nResolucion: { crearTraductorResolucionCronos: () => (clave) => ({ bandeja_titulo: "Solicitudes por resolver", avisos_titulo: "Avisos de resolución" })[clave] },
  };
  const cronos = componerCronosInterno(conResolucion, { fetch() {} });
  assert.equal(creados.length, 1);
  assert.deepEqual({ ...cronos.etiquetas }, { bandeja: "Solicitudes por resolver", avisos: "Avisos de resolución" });
  cronos.montarBandeja({ raiz: "r1" }); cronos.montarAvisos({ raiz: "r2" });
  assert.deepEqual(montados.map(([n, o]) => [n, o.raiz]), [["bandeja", "r1"], ["avisos", "r2"]]);
  assert.strictEqual(montados[0][1].cliente, creados[0]);
  assert.strictEqual(montados[1][1].cliente, creados[0]);
  const { clienteResolucion: _omitido, ...incompleto } = conResolucion;
  const sin = componerCronosInterno(incompleto, {});
  assert.notEqual(sin, undefined, "sin resolución Cronos sigue disponible");
  assert.equal(sin.montarBandeja, undefined);
  assert.equal(sin.montarAvisos, undefined);
});

test("Cronos ofrece las notificaciones a RRHH solo con sus cuatro piezas, con o sin resolución", () => {
  const creados = []; const montados = [];
  const partes = { saldo: () => ({}), remoto: () => ({}), movimientos: () => ({}), calendario: () => ({}) };
  const conNotificaciones = {
    ...recursosCronos(partes),
    notificacionesPropias: { montarNotificacionesPropiasCronos: (o) => { montados.push(["propias", o]); return Object.freeze({ desmontar() {} }); } },
    bandejaNotificaciones: { montarBandejaNotificacionesCronos: (o) => { montados.push(["bandeja", o]); return Object.freeze({ desmontar() {} }); } },
    clienteNotificaciones: { crearClienteNotificacionesCronosHTTP: (t) => { const c = Object.freeze({ t }); creados.push(c); return c; } },
    i18nNotificaciones: { crearTraductorNotificacionesCronos: () => (clave) => ({ notificaciones_titulo: "Notificaciones a RRHH", bandeja_notificaciones_titulo: "Notificaciones recibidas" })[clave] },
  };
  const cronos = componerCronosInterno(conNotificaciones, { fetch() {} });
  assert.equal(creados.length, 1);
  assert.deepEqual({ ...cronos.etiquetas }, { notificaciones: "Notificaciones a RRHH", bandejaNotificaciones: "Notificaciones recibidas" });
  assert.equal(cronos.montarBandeja, undefined, "sin resolución no hay bandeja de permisos");
  cronos.montarNotificaciones({ raiz: "r1" }); cronos.montarBandejaNotificaciones({ raiz: "r2" });
  assert.deepEqual(montados.map(([n, o]) => [n, o.raiz]), [["propias", "r1"], ["bandeja", "r2"]]);
  assert.strictEqual(montados[0][1].cliente, creados[0]);
  assert.strictEqual(montados[1][1].cliente, creados[0]);
  const { i18nNotificaciones: _omitido, ...incompleto } = conNotificaciones;
  const sin = componerCronosInterno(incompleto, {});
  assert.equal(sin.montarNotificaciones, undefined);
  assert.equal(sin.etiquetas, undefined);
});
