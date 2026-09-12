import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { renderizarCuadro } from "./componentes-expedientes.js";
import { validarCuadroContratacionTemporal } from "./contrato-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import {
  montarModuloContratacionTemporal,
  renderizarModuloContratacionTemporal,
} from "./vista-expedientes.js";

const CURSOR_B = "A".repeat(43);
const CURSOR_C = "B".repeat(42) + "E";
const CURSOR_D = "C".repeat(42) + "I";

function expediente(sufijo) {
  return {
    expediente_ref: `expediente:ct:pag-${sufijo}`,
    numero_visible: `2026/CT-00${sufijo}`,
    centro: "centro:rrhh",
    categoria: "categoria:auxiliar",
    modalidad: "Bolsa",
    estado_clave: "pendiente",
    estado: "Pendiente",
    fase_clave: "solicitud",
    fase_actual: "Solicitud",
    fecha_solicitud: "2026-09-07T08:00:00Z",
    responsable: "—",
    plazo: "—",
    version: 7,
  };
}

function cuadro({ pagina, cursor, siguiente = "", sufijo }) {
  return validarCuadroContratacionTemporal({
    esquema: "vec.contratacion_temporal.cuadro.v1",
    demostracion: false,
    generado_en: "2026-09-07T08:00:00Z",
    indicadores: [],
    expedientes: [expediente(sufijo)],
    paginacion: {
      pagina,
      cursor_actual: cursor,
      cursor_siguiente: siguiente,
    },
  });
}

test("consume cada cursor una vez y reinicia antes de actualizar o cambiar filtros", async () => {
  const llamadas = [];
  const consumidos = new Set();
  let primeras = 0;
  const fuente = {
    async listar({ filtros, cursor, numeroPagina }) {
      llamadas.push({ filtros, cursor, numeroPagina });
      if (cursor !== "") {
        if (consumidos.has(cursor)) throw new Error("cursor reutilizado");
        consumidos.add(cursor);
      }
      if (cursor === "") {
        primeras += 1;
        return cuadro({ pagina: 1, cursor, siguiente: primeras === 1 ? CURSOR_B : CURSOR_D, sufijo: "1" });
      }
      if (cursor === CURSOR_B) return cuadro({ pagina: 2, cursor, siguiente: CURSOR_C, sufijo: "2" });
      if (cursor === CURSOR_D) return cuadro({ pagina: 2, cursor, siguiente: CURSOR_C, sufijo: "2" });
      return cuadro({ pagina: 3, cursor, sufijo: "3" });
    },
    async obtener() { throw new Error("no se debe abrir detalle"); },
    async ejecutar() { throw new Error("no se debe ejecutar efecto"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente,
    capacidades: ["contratacion_temporal.cuadro.consultar"],
  });

  await presentador.cargar({ texto: "auxiliar", estado: "", fase: "" });
  await presentador.navegarPagina("siguiente");
  assert.equal(presentador.obtenerEstado().cuadro.expedientes[0].expediente_ref, "expediente:ct:pag-2");
  assert.deepEqual(llamadas.at(-1), {
    filtros: { texto: "auxiliar", estado: "", fase: "" }, cursor: CURSOR_B, numeroPagina: 2,
  });
  await presentador.cargar();
  assert.deepEqual(llamadas.at(-1), {
    filtros: { texto: "auxiliar", estado: "", fase: "" }, cursor: "", numeroPagina: 1,
  });
  await presentador.navegarPagina("siguiente");
  await presentador.navegarPagina("siguiente");
  assert.equal(presentador.obtenerEstado().cuadro.paginacion.pagina, 3);
  await presentador.cargar({ texto: "", estado: "pendiente", fase: "" });
  assert.deepEqual(llamadas.at(-1), {
    filtros: { texto: "", estado: "pendiente", fase: "" }, cursor: "", numeroPagina: 1,
  });
});

test("descarta la página retrasada tras cambiar filtros y no conserva selección oculta", async () => {
  let resolverRetrasada;
  let resolverReciente;
  let llamadas = 0;
  const fuente = {
    listar({ cursor, filtros }) {
      llamadas += 1;
      if (llamadas === 1) return Promise.resolve(cuadro({
        pagina: 1, cursor: "", siguiente: CURSOR_B, sufijo: "1",
      }));
      return new Promise((resolver) => {
        if (cursor === CURSOR_B) resolverRetrasada = resolver;
        else if (filtros.texto === "nueva") resolverReciente = resolver;
      });
    },
    async obtener() { throw new Error("no se debe abrir detalle"); },
    async ejecutar() { throw new Error("no se debe ejecutar efecto"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente,
    capacidades: ["contratacion_temporal.cuadro.consultar"],
  });
  await presentador.cargar();
  const segunda = presentador.navegarPagina("siguiente");
  const filtros = presentador.cargar({ texto: "nueva", estado: "", fase: "" });
  resolverReciente(cuadro({ pagina: 1, cursor: "", sufijo: "fil" }));
  await filtros;
  resolverRetrasada(cuadro({ pagina: 2, cursor: CURSOR_B, sufijo: "2" }));
  await segunda;
  assert.equal(llamadas, 3);
  assert.equal(presentador.obtenerEstado().expediente_ref, "");
  assert.equal(presentador.obtenerEstado().cuadro.expedientes[0].expediente_ref, "expediente:ct:pag-fil");
  assert.equal(presentador.obtenerEstado().cuadro.paginacion.pagina, 1);
});

test("proyecta controles de página sin exponer el cursor", async () => {
  const html = renderizarCuadro({
    carga: "listo",
    filtros: { texto: "", estado: "", fase: "" },
    cuadro: cuadro({ pagina: 2, cursor: CURSOR_B, siguiente: CURSOR_C, sufijo: "2" }),
  }, crearTraductorExpedientesContratacion());
  assert.match(html, /Página 2/u);
  assert.match(html, /data-ct-exp-pagina="primera"/u);
  assert.match(html, /data-ct-exp-pagina="siguiente"/u);
  assert.doesNotMatch(html, /data-ct-exp-pagina="anterior"/u);
  assert.doesNotMatch(html, new RegExp(CURSOR_B, "u"));
  assert.doesNotMatch(html, new RegExp(CURSOR_C, "u"));
});

test("un cursor fallido conserva la página anterior y reinicia con cursor vacío", async () => {
  let llamadas = 0;
  const cursores = [];
  const fuente = {
    async listar({ cursor }) {
      llamadas += 1;
      cursores.push(cursor);
      if (cursor === "") return cuadro({ pagina: 1, cursor, siguiente: CURSOR_B, sufijo: "1" });
      throw new Error("cursor caducado");
    },
    async obtener() { throw new Error("no se debe abrir detalle"); },
    async ejecutar() { throw new Error("no se debe ejecutar efecto"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente,
    capacidades: ["contratacion_temporal.cuadro.consultar"],
  });
  await presentador.cargar();
  await presentador.navegarPagina("siguiente");
  assert.equal(presentador.obtenerEstado().carga, "error");
  assert.equal(presentador.obtenerEstado().cuadro.paginacion.pagina, 1);
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_error_paginacion");
  const recuperable = presentador.obtenerEstado();
  const html = renderizarModuloContratacionTemporal(recuperable);
  assert.match(html, /2026\/CT-001/u);
  assert.match(html, /Reiniciar consulta/u);
  assert.match(html, /data-ct-exp-pagina="siguiente"\s*\n\s*disabled/u);
  assert.match(html, /no es el final de la lista/u);
  const htmlSinCuadro = renderizarModuloContratacionTemporal({
    ...recuperable, cuadro: null, paginacion: null, paginacion_requiere_reinicio: false,
  });
  assert.match(htmlSinCuadro, /ct-exp-estado-global/u);
  assert.doesNotMatch(htmlSinCuadro, /2026\/CT-001/u);
  await presentador.cargar();
  assert.equal(presentador.obtenerEstado().carga, "listo");
  assert.deepEqual(llamadas, 3);
  assert.deepEqual(cursores, ["", CURSOR_B, ""]);
});

test("adaptador conserva cursor_siguiente validado y lo reenvía sin inventar total", async () => {
  const solicitudes = [];
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: {
      async consultarCuadroRRHH(solicitud) {
        solicitudes.push(solicitud);
        return {
          esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
          generada_en: "2026-09-07T08:00:00Z",
          expedientes: [{
            expediente_ref: "expediente:ct:real-1", numero_visible: "2026/CT-0001", version: 7,
            flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
            fase_clave: "solicitud", estado_clave: "pendiente", centro_ref: "centro:rrhh",
            categoria_ref: "categoria:auxiliar", creado_en: "2026-09-07T08:00:00Z", actualizado_en: "2026-09-07T08:00:00Z",
          }], hay_mas: true, cursor_siguiente: CURSOR_B,
        };
      },
      async consultarDetalleRRHH() { throw new Error("no se debe pedir detalle"); },
    },
  });
  const pagina = await adaptador.listar({ cursor: CURSOR_B, numeroPagina: 2 });
  assert.equal(pagina.paginacion.cursor_siguiente, CURSOR_B);
  assert.deepEqual(solicitudes[0].paginacion, { limite: 100, cursor: CURSOR_B });
  assert.equal(Object.hasOwn(pagina, "total"), false);
});

test("rechaza cursor_siguiente mal formado antes de alterar la caché", async () => {
  let consultas = 0;
  let versionDetalle;
  const respuesta = (version, cursor_siguiente) => ({
    esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-07T08:00:00Z",
    expedientes: [{
      expediente_ref: "expediente:ct:cursor-1", numero_visible: "2026/CT-0001", version,
      flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
      fase_clave: "solicitud", estado_clave: "pendiente", centro_ref: "centro:rrhh",
      categoria_ref: "categoria:auxiliar", creado_en: "2026-09-07T08:00:00Z", actualizado_en: "2026-09-07T08:00:00Z",
    }], hay_mas: true, cursor_siguiente,
  });
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: {
      async consultarCuadroRRHH() {
        consultas += 1;
        return consultas === 1 ? respuesta(7, CURSOR_B) : respuesta(8, "cursor-invalido");
      },
      async consultarDetalleRRHH(solicitud) {
        versionDetalle = solicitud.version_observada;
        throw new Error("detalle de prueba");
      },
    },
  });
  await adaptador.listar();
  await assert.rejects(
    adaptador.listar({ cursor: CURSOR_B, numeroPagina: 2 }),
    /paginación/u,
  );
  await assert.rejects(adaptador.obtener("expediente:ct:cursor-1"), /detalle de prueba/u);
  assert.equal(versionDetalle, 7);
});

test("una respuesta antigua no sustituye la caché de versión para el detalle", async () => {
  let resolverAntigua;
  let resolverReciente;
  let detalleSolicitado;
  const paginaServidor = (version) => ({
    esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-07T08:00:00Z",
    expedientes: [{
      expediente_ref: "expediente:ct:cache-1", numero_visible: "2026/CT-0001", version,
      flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
      fase_clave: "solicitud", estado_clave: "pendiente", centro_ref: "centro:rrhh",
      categoria_ref: "categoria:auxiliar", creado_en: "2026-09-07T08:00:00Z", actualizado_en: "2026-09-07T08:00:00Z",
    }], hay_mas: false,
  });
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: {
      consultarCuadroRRHH({ paginacion }) {
        return new Promise((resolver) => {
          if (paginacion.cursor === "") resolverAntigua = resolver;
          else resolverReciente = resolver;
        });
      },
      async consultarDetalleRRHH(solicitud) {
        detalleSolicitado = solicitud;
        throw new Error("detalle de prueba");
      },
    },
  });
  const antigua = adaptador.listar();
  const reciente = adaptador.listar({ cursor: CURSOR_B, numeroPagina: 2 });
  resolverReciente(paginaServidor(8));
  await reciente;
  resolverAntigua(paginaServidor(7));
  await antigua;
  await assert.rejects(adaptador.obtener("expediente:ct:cache-1"), /detalle de prueba/u);
  assert.equal(detalleSolicitado.version_observada, 8);
});

test("cancelar una continuación impide reutilizar su cursor y no altera versiones al recibirla tarde", async () => {
  let devolver;
  let llamadas = 0;
  let versionDetalle;
  const paginaServidor = (version, hayMas = false) => ({
    esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-07T08:00:00Z",
    expedientes: [{
      expediente_ref: "expediente:ct:cancel-1", numero_visible: "2026/CT-0001", version,
      flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
      fase_clave: "solicitud", estado_clave: "pendiente", centro_ref: "centro:rrhh",
      categoria_ref: "categoria:auxiliar", creado_en: "2026-09-07T08:00:00Z", actualizado_en: "2026-09-07T08:00:00Z",
    }], hay_mas: hayMas, ...(hayMas ? { cursor_siguiente: CURSOR_B } : {}),
  });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: {
      consultarCuadroRRHH() {
        llamadas += 1;
        if (llamadas === 1) return Promise.resolve(paginaServidor(7, true));
        if (llamadas === 2) return new Promise((resolver) => { devolver = resolver; });
        return Promise.reject(new Error("cursor consumido"));
      },
      async consultarDetalleRRHH(solicitud) {
        versionDetalle = solicitud.version_observada;
        throw new Error("captura del detalle");
      },
    },
  });
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente, capacidades: ["contratacion_temporal.cuadro.consultar"],
  });
  await presentador.cargar();
  const pendiente = presentador.navegarPagina("siguiente");
  presentador.cancelar();
  devolver(paginaServidor(8));
  await pendiente;
  await assert.rejects(fuente.obtener("expediente:ct:cancel-1"), /captura del detalle/u);
  await presentador.navegarPagina("siguiente");
  assert.deepEqual({ versionDetalle, llamadas }, { versionDetalle: 7, llamadas: 2 });
  const html = renderizarCuadro(presentador.obtenerEstado(), crearTraductorExpedientesContratacion());
  assert.match(html, /data-ct-exp-pagina="siguiente"\s+disabled/u);
  assert.match(html, /Reiniciar consulta/u);
});

test("el envío de filtro inválido conserva el cuadro y se presenta sin rechazar", async () => {
  const eventos = new Map();
  const mensaje = { textContent: "", setAttribute() {} };
  const raiz = {
    innerHTML: "",
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: () => mensaje,
    contains: () => true,
  };
  let consultas = 0;
  const fuente = {
    async listar() { consultas += 1; return cuadro({ pagina: 1, cursor: "", sufijo: "1" }); },
    async obtener() { throw new Error("no se debe abrir detalle"); },
    async ejecutar() { throw new Error("no se debe ejecutar efecto"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente, capacidades: ["contratacion_temporal.cuadro.consultar"],
  });
  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(formulario) { this.formulario = formulario; } get(campo) { return this.formulario.valores[campo]; } };
  const montaje = await montarModuloContratacionTemporal({ raiz, presentador });
  try {
    await eventos.get("submit")({
      preventDefault() {},
      target: { closest: () => ({ valores: { texto: "A".repeat(81), estado: "", fase: "" } }) },
    });
    assert.equal(consultas, 1);
    assert.equal(presentador.obtenerEstado().cuadro.expedientes[0].expediente_ref, "expediente:ct:pag-1");
    assert.match(mensaje.textContent, /hasta 80 caracteres/u);
  } finally {
    montaje.desmontar();
    globalThis.FormData = anterior;
  }
});
