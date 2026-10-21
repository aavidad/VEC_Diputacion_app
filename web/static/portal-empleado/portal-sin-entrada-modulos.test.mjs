import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { prepararTextosPortal } from "./portal-i18n.js?v=VEC_WEB_VERSION";
await prepararTextosPortal("ayuda");
const { TRAMITES_AYUDANTE_PORTAL } = await import("./ayuda-contenido.js?v=VEC_WEB_VERSION");
import {
  CLAVES_ENTRADA_SEGUN_ACCESO,
  CLAVES_SIN_ENTRADA_PORTAL,
  crearCoordinadorModulosPortal,
  vistaConEntradaPortal,
} from "./portal-modulos-coordinador.js?v=VEC_WEB_VERSION";
import { crearVistaInicioPortal } from "./portal-inicio.js?v=VEC_WEB_VERSION";

// Personal y Dietas se siguen cargando (su URL directa funciona), pero no
// tienen entrada en el menú lateral, en Inicio ni en el ayudante de trámites.
// Cronos entra en el menú y en Inicio solo si el servidor confirma que la
// sesión puede usarlo (lectura del saldo propio del día con 200).

const OCULTOS = ["personal", "cronos", "dietas"];
const SIEMPRE_OCULTOS = ["personal", "dietas"];
const SALDO_HOY = Object.freeze({
  periodo: { tipo: "hoy", desde: "2026-10-21", hasta: "2026-10-21" },
  resumen: { previstos_minutos: null, trabajados_minutos: 0, saldo_minutos: null, estado: "sin_datos" },
  detalle: [],
});
const modulo = (clave, sigla) => Object.freeze({ clave, sigla, titulo: `Módulo ${sigla}`, texto: `Texto ${sigla}` });
const CATALOGO = Object.freeze([
  modulo("personal", "PER"), modulo("cronos", "CRO"), modulo("dietas", "DIE"),
  modulo("bolsa", "BOL"), modulo("contratacion_temporal", "CT"),
]);
const esperarTurnos = async (turnos = 5) => {
  for (let i = 0; i < turnos; i += 1) await new Promise((seguir) => setImmediate(seguir));
};
const sinTemporizador = Object.freeze({ setTimeout: () => 0, clearTimeout() {} });

function coordinadorConModulosCargando() {
  // Personal, Cronos y Dietas quedan «cargando»: sin el filtro se ofrecerían.
  const nunca = () => new Promise(() => {});
  return crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => CATALOGO,
    entorno: { fetch: async () => { throw new Error("sin red"); } },
    temporizadores: sinTemporizador,
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("sin CT"); },
      personal: nunca, cronos: nunca, dietas: nunca,
    },
  });
}

test("Personal y Dietas no tienen entrada propia; Cronos depende del acceso y el ayudante no lo ofrece", () => {
  for (const clave of SIEMPRE_OCULTOS) assert.ok(CLAVES_SIN_ENTRADA_PORTAL.includes(clave), clave);
  assert.deepEqual(CLAVES_ENTRADA_SEGUN_ACCESO, ["cronos"]);
  assert.equal(CLAVES_SIN_ENTRADA_PORTAL.includes("cronos"), false);
  for (const vista of ["personal", "personal-registro", "cronos", "cronos-permisos", "dietas"])
    assert.equal(vistaConEntradaPortal(vista), false, vista);
  for (const vista of ["contratacion-temporal", "resumen", "llamamientos"])
    assert.equal(vistaConEntradaPortal(vista), true, vista);
});

test("mientras no se confirma el acceso a Cronos, el menú lateral e Inicio solo ofrecen Bolsa y la contratación temporal", async () => {
  const coordinador = coordinadorConModulosCargando();
  void coordinador.cargarInterno().catch(() => {});
  await esperarTurnos();
  // Cronos se carga al arrancar para comprobar el acceso; mientras tanto no se ofrece.
  assert.equal(coordinador.resolverAcceso("cronos").estado, "cargando");
  assert.equal(coordinador.vistaPendiente("cronos"), true, "su URL directa espera a su carga");
  assert.deepEqual(coordinador.obtenerCatalogo().map(({ clave }) => clave), ["bolsa", "contratacion_temporal"]);

  const menu = coordinador.renderizarNavegacion(true);
  for (const clave of OCULTOS) assert.doesNotMatch(menu, new RegExp(`data-modulo-portal="${clave}"`, "u"));

  for (const esPerfilRRHH of [() => false, () => true]) {
    const inicio = crearVistaInicioPortal({
      encabezadoVista: (_s, titulo) => `<h2>${titulo}</h2>`,
      escaparHTML: String,
      obtenerCatalogo: coordinador.obtenerCatalogo,
      resolverAcceso: (clave) => coordinador.resolverAcceso(clave),
      esPerfilRRHH,
    })();
    for (const clave of OCULTOS) {
      assert.doesNotMatch(inicio, new RegExp(`data-modulo-catalogo="${clave}"`, "u"));
      assert.doesNotMatch(inicio, new RegExp(`data-vista="${clave}`, "u"));
    }
    assert.doesNotMatch(inicio, /Módulo (PER|CRO|DIE)/u);
  }
});

test("el ayudante de trámites y el resumen de accesos no ofrecen los módulos ocultos", async () => {
  const ofrecidos = TRAMITES_AYUDANTE_PORTAL.filter((tramite) => vistaConEntradaPortal(tramite.vista));
  assert.ok(ofrecidos.length > 0, "Bolsa conserva sus trámites guiados");
  for (const tramite of ofrecidos) {
    assert.doesNotMatch(tramite.vista, /^(personal|cronos|dietas)/u, tramite.id);
    for (const paso of tramite.pasos) if (paso.vista) assert.equal(vistaConEntradaPortal(paso.vista), true, tramite.id);
  }
  const portal = await readFile(new URL("./portal.js", import.meta.url), "utf8");
  assert.match(portal, /TRAMITES_AYUDANTE_PORTAL\.filter\(\(tramite\) => vistaConEntradaPortal\(tramite\.vista\)\)/u);
  // El resumen cuenta las entradas que el menú muestra, que ya excluyen los ocultos.
  assert.match(portal, /contenedor\.querySelectorAll\("\[data-modulo-portal\]"\)/u);
});

// Cronos según la respuesta del servidor a la lectura del saldo propio del día.
const recursosCronosReales = async () => {
  const [saldo, remoto, movimientos, movimientosPropios, permisosPropios, clienteSaldo, clienteRemoto,
    clienteSolicitudes, i18n, bandejaPermisos, avisosPropios, clienteResolucion, i18nResolucion,
    notificacionesPropias, bandejaNotificaciones, clienteNotificaciones, i18nNotificaciones] = await Promise.all([
    import("./modulos/cronos/vista-saldo-conectado.js"), import("./modulos/cronos/vista-remoto.js"),
    import("./modulos/cronos/vista-movimientos-conectado.js"), import("./modulos/cronos/vista-movimientos-propios.js"),
    import("./modulos/cronos/vista-permisos-propios.js"), import("./modulos/cronos/cliente-saldo-http.js"),
    import("./modulos/cronos/cliente-remoto-http.js"), import("./modulos/cronos/cliente-solicitudes-http.js"),
    import("./modulos/cronos/i18n.js"), import("./modulos/cronos/vista-bandeja-permisos.js"),
    import("./modulos/cronos/vista-avisos-propios.js"), import("./modulos/cronos/cliente-resolucion-http.js"),
    import("./modulos/cronos/i18n-resolucion.js"), import("./modulos/cronos/vista-notificaciones-propias.js"),
    import("./modulos/cronos/vista-bandeja-notificaciones.js"), import("./modulos/cronos/cliente-notificaciones-http.js"),
    import("./modulos/cronos/i18n-notificaciones.js"),
  ]);
  return { saldo, remoto, movimientos, movimientosPropios, permisosPropios, clienteSaldo, clienteRemoto,
    clienteSolicitudes, i18n, bandejaPermisos, avisosPropios, clienteResolucion, i18nResolucion,
    notificacionesPropias, bandejaNotificaciones, clienteNotificaciones, i18nNotificaciones };
};
const json = (estado, cuerpo) => new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": "application/json" } });

function coordinadorCronos(respuestas) {
  const pedidas = [];
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => CATALOGO,
    entorno: { fetch: async (ruta) => {
      const camino = String(ruta).split("?")[0]; pedidas.push(camino);
      return respuestas[camino] ? respuestas[camino]() : json(404, { error: "no_encontrado" });
    } },
    temporizadores: sinTemporizador,
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("sin CT"); },
      cronos: recursosCronosReales,
    },
  });
  return { coordinador, pedidas };
}
const SALDO = "/api/interna/cronos/saldos/propio";
const AVISOS = "/api/interna/cronos/avisos/propio";
const NOTIFICACIONES = "/api/interna/cronos/notificaciones/propio";

function menuEInicio(coordinador) {
  const menu = coordinador.renderizarNavegacion(true, "portal");
  const inicio = crearVistaInicioPortal({
    encabezadoVista: (_s, titulo) => `<h2>${titulo}</h2>`, escaparHTML: String,
    obtenerCatalogo: coordinador.obtenerCatalogo, resolverAcceso: (clave) => coordinador.resolverAcceso(clave),
    esPerfilRRHH: () => false,
  })();
  return { menu, inicio };
}

test("con el saldo propio en 200, Cronos entra en el menú y en Inicio", async () => {
  const { coordinador, pedidas } = coordinadorCronos({ [SALDO]: () => json(200, SALDO_HOY) });
  await coordinador.cargarInterno();
  assert.equal(pedidas[0], SALDO, "la comprobación es la lectura del saldo propio");
  assert.deepEqual(coordinador.obtenerCatalogo().map(({ clave }) => clave), ["cronos", "bolsa", "contratacion_temporal"]);
  const { menu, inicio } = menuEInicio(coordinador);
  assert.match(menu, /data-modulo-portal="cronos"[^>]*data-vista="cronos"/u);
  assert.match(inicio, /data-modulo-catalogo="cronos"/u);
  for (const clave of SIEMPRE_OCULTOS) assert.doesNotMatch(menu, new RegExp(`data-modulo-portal="${clave}"`, "u"));
  assert.equal(vistaConEntradaPortal("cronos"), false, "el ayudante de trámites sigue sin ofrecerlo");
});

for (const [estado, motivo] of [[403, "sin permiso de Cronos"], [401, "sin sesión válida"],
  [404, "Cronos apagado en el servidor"], [500, "fallo del servidor"]]) {
  test(`con el saldo propio en ${estado} (${motivo}), Cronos no aparece y su URL directa no lo monta`, async () => {
    const { coordinador, pedidas } = coordinadorCronos({ [SALDO]: () => json(estado, { error: "x" }) });
    await coordinador.cargarInterno();
    assert.deepEqual(pedidas, [SALDO], "sin acceso no se sondean sus servicios");
    assert.equal(coordinador.obtenerCatalogo().some(({ clave }) => clave === "cronos"), false);
    const { menu, inicio } = menuEInicio(coordinador);
    assert.doesNotMatch(menu, /data-modulo-portal="cronos"/u);
    assert.doesNotMatch(inicio, /data-modulo-catalogo="cronos"|data-vista="cronos/u);
    assert.equal(coordinador.vistaDisponible("cronos"), false);
    assert.equal(coordinador.vistaPendiente("cronos"), false);
    assert.equal(coordinador.resolverAcceso("cronos").disponible, false);
  });
}

test("una pestaña de Cronos cuyo servicio está apagado (404) no se ofrece; la del servicio activo sí", async () => {
  const { coordinador, pedidas } = coordinadorCronos({
    [SALDO]: () => json(200, SALDO_HOY),
    [NOTIFICACIONES]: () => json(200, { tipos: [], notificaciones: [] }),
  });
  await coordinador.cargarInterno();
  assert.deepEqual([...pedidas].sort(), [AVISOS, NOTIFICACIONES, SALDO].sort());
  assert.equal(coordinador.vistaDisponible("cronos-avisos"), false, "avisos apagado");
  assert.equal(coordinador.vistaDisponible("cronos-bandeja"), false, "su bandeja va con avisos");
  assert.equal(coordinador.vistaDisponible("cronos-notificaciones"), true);
  assert.equal(coordinador.vistaDisponible("cronos-bandeja-notificaciones"), true);
});

test("si la comprobación de un servicio falla por otra causa, su pestaña se conserva", async () => {
  const { coordinador } = coordinadorCronos({
    [SALDO]: () => json(200, SALDO_HOY),
    [AVISOS]: () => json(503, { error: "x" }),
    [NOTIFICACIONES]: () => json(403, { error: "x" }),
  });
  await coordinador.cargarInterno();
  assert.equal(coordinador.vistaDisponible("cronos-avisos"), true);
  assert.equal(coordinador.vistaDisponible("cronos-notificaciones"), true);
});
