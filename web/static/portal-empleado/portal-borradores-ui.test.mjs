import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  ErrorAPIBorradores,
  RUTAS_API_BORRADORES,
  crearClienteBorradores,
} from "./portal-borradores-api.js";
import {
  ESQUEMAS_BORRADORES,
  validarSolicitudActualizarBorrador,
  validarSolicitudCrearBorrador,
} from "./portal-borradores-contrato.js";
import { crearSuperficieBorradoresPortal } from "./portal-borradores-ui.js";
import {
  CLAVE_IDEMPOTENCIA_A,
  CLAVE_IDEMPOTENCIA_B,
  detalle,
  etagEstado,
  lista,
  opciones,
  recibo,
  respuestaJSON,
} from "./portal-borradores-fixtures.test-helper.mjs";

const directorio = new URL("./", import.meta.url);
const escaparHTML = (valor) => String(valor ?? "")
  .replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;")
  .replaceAll("'", "&#039;");

function crearDobleCliente(sobrescrituras = {}) {
  return {
    obtenerOpciones: async () => structuredClone(opciones()),
    listar: async () => structuredClone(lista()),
    obtenerDetalle: async () => structuredClone(detalle()),
    crear: async () => structuredClone(recibo("crear")),
    actualizar: async () => structuredClone(recibo("actualizar")),
    ...sobrescrituras,
  };
}

function crearSuperficie({ cliente, proveedor = () => "bearer-en-memoria", claves } = {}) {
  const anuncios = [];
  const cambios = [];
  let indiceClave = 0;
  const superficie = crearSuperficieBorradoresPortal({
    escaparHTML,
    anunciar: (mensaje) => anuncios.push(mensaje),
    alCambiar: () => cambios.push("cambio"),
    resolverProveedorBearer: () => proveedor,
    confirmar: () => true,
    crearClienteImpl: () => cliente || crearDobleCliente(),
    generarClaveImpl: () => (claves?.[indiceClave++] || CLAVE_IDEMPOTENCIA_A),
  });
  return { anuncios, cambios, superficie };
}

function cambiar(superficie, ruta, valor, tipo = "text", checked = false) {
  assert.equal(superficie.actualizarCampo({ ruta, valor, tipo, checked }), true);
}

function detalleRemoto() {
  const remoto = structuredClone(detalle());
  remoto.referencia_estado.revision = 4;
  remoto.etag = etagEstado(4);
  remoto.contenido_editable.titulo = "Título vigente del servidor";
  remoto.contenido_editable.resumen = "Resumen modificado por otra sesión.";
  return remoto;
}

function diferida() {
  let resolver;
  const promesa = new Promise((resolve) => { resolver = resolve; });
  return { promesa, resolver };
}

test("la vista real carga bandeja, editor gobernado y evidencias desde el cliente autenticado", async () => {
  const { anuncios, cambios, superficie } = crearSuperficie({ cliente: crearDobleCliente() });
  assert.equal(await superficie.activar(), true);
  const html = superficie.renderizar();
  assert.match(html, /Bandeja de borradores/);
  assert.match(html, /data-borrador-form="editor"/);
  assert.match(html, /Identidad gobernada de solo lectura/);
  assert.match(html, /Control de concurrencia/);
  assert.match(html, /ETag fuerte/);
  assert.match(html, /Configuración acreditada/);
  assert.match(html, /Guardar con CAS/);
  assert.match(html, /aria-label="Contexto y evidencias del borrador"/);
  assert.ok(cambios.length >= 3);
  assert.ok(anuncios.includes("Borrador abierto para edición"));
});

test("la capacidad de consulta sin actualización deja el detalle realmente en solo lectura", async () => {
  const lectura = structuredClone(detalle());
  lectura.capacidades.actualizar = false;
  const { superficie } = crearSuperficie({
    cliente: crearDobleCliente({ obtenerDetalle: async () => structuredClone(lectura) }),
  });
  await superficie.activar();
  const html = superficie.renderizar();
  assert.match(html, /Solo lectura: sin capacidad/);
  assert.match(html, /data-borrador-form="editor" inert[^>]+aria-disabled="true"/);
  assert.equal(superficie.actualizarCampo({
    ruta: "contenido_editable.titulo", valor: "Cambio no autorizado",
  }), false);
  assert.doesNotMatch(superficie.renderizar(), /Cambio no autorizado/);
});

test("sin proveedor Bearer la superficie delega la identidad al canal interno", async () => {
  let clientesCreados = 0;
  let configuracionCliente;
  const cerrada = crearSuperficieBorradoresPortal({
    escaparHTML,
    anunciar: () => {},
    alCambiar: () => {},
    resolverProveedorBearer: () => null,
    confirmar: () => true,
    crearClienteImpl: (configuracion) => {
      clientesCreados += 1;
      configuracionCliente = configuracion;
      return crearDobleCliente();
    },
    generarClaveImpl: () => CLAVE_IDEMPOTENCIA_A,
  });
  assert.equal(await cerrada.activar(), true);
  assert.equal(clientesCreados, 1);
  assert.equal(configuracionCliente.obtenerBearer, null);
  const html = cerrada.renderizar();
  assert.match(html, /Bandeja de borradores/);
  assert.doesNotMatch(html, /proveedor.*no está configurado/i);
});

test("un proveedor Bearer de tipo inválido bloquea la superficie sin crear cliente", async () => {
  let clientesCreados = 0;
  const superficie = crearSuperficieBorradoresPortal({
    escaparHTML,
    anunciar: () => {},
    alCambiar: () => {},
    resolverProveedorBearer: () => ({ token: "configuracion-invalida" }),
    confirmar: () => true,
    crearClienteImpl: () => { clientesCreados += 1; return crearDobleCliente(); },
    generarClaveImpl: () => CLAVE_IDEMPOTENCIA_A,
  });
  assert.equal(await superficie.activar(), false);
  assert.equal(clientesCreados, 0);
  const html = superficie.renderizar();
  assert.match(html, /proveedor opcional de credencial.*no es válido/i);
  assert.match(html, /no puede operar sin el backend autenticado/i);
});

test("el modo Bearer re-resuelve el proveedor y una retirada bloquea antes de Fetch", async () => {
  let proveedor = () => "token-inicial";
  const autorizaciones = [];
  let peticiones = 0;
  const fetchImpl = async (ruta, configuracion) => {
    peticiones += 1;
    autorizaciones.push(configuracion.headers.get("authorization"));
    if (ruta === RUTAS_API_BORRADORES.opciones) return respuestaJSON(opciones());
    if (ruta.startsWith(`${RUTAS_API_BORRADORES.lista}?`)) return respuestaJSON(lista());
    return respuestaJSON(detalle(), { etag: detalle().etag });
  };
  const superficie = crearSuperficieBorradoresPortal({
    escaparHTML,
    anunciar: () => {},
    alCambiar: () => {},
    resolverProveedorBearer: () => proveedor,
    confirmar: () => true,
    crearClienteImpl: (configuracion) => crearClienteBorradores({ ...configuracion, fetchImpl }),
    generarClaveImpl: () => CLAVE_IDEMPOTENCIA_A,
  });
  assert.equal(await superficie.activar(), true);
  assert.deepEqual(autorizaciones, Array(3).fill("Bearer token-inicial"));

  proveedor = () => "token-sustituido";
  assert.equal(await superficie.aplicarFiltro({}), true);
  assert.equal(autorizaciones.at(-1), "Bearer token-sustituido");
  const peticionesAntesRetirada = peticiones;

  proveedor = null;
  assert.equal(await superficie.aplicarFiltro({ texto: "no-debe-consultarse" }), false);
  assert.equal(peticiones, peticionesAntesRetirada, "la retirada debe bloquear antes de Fetch");
  assert.match(superficie.renderizar(), /proveedor opcional de credencial.*dejó de estar disponible/i);

  proveedor = { token: "configuracion-invalida" };
  assert.equal(await superficie.aplicarFiltro({ texto: "tampoco-debe-consultarse" }), false);
  assert.equal(peticiones, peticionesAntesRetirada, "un reemplazo inválido debe bloquear antes de Fetch");
  assert.match(superficie.renderizar(), /proveedor opcional de credencial.*no es válido/i);
});

test("una respuesta de filtro obsoleta no reemplaza la bandeja más reciente", async () => {
  const antigua = diferida();
  const reciente = diferida();
  let consultas = 0;
  const cliente = crearDobleCliente({
    listar: async () => {
      consultas += 1;
      if (consultas === 1) return structuredClone(lista());
      return consultas === 2 ? antigua.promesa : reciente.promesa;
    },
  });
  const { superficie } = crearSuperficie({ cliente });
  await superficie.activar();
  const cargaAntigua = superficie.aplicarFiltro({ texto: "antigua" });
  const cargaReciente = superficie.aplicarFiltro({ texto: "reciente" });
  const resultadoReciente = structuredClone(lista());
  resultadoReciente.elementos[0].titulo = "Resultado reciente";
  reciente.resolver(resultadoReciente);
  assert.equal(await cargaReciente, true);
  const resultadoAntiguo = structuredClone(lista());
  resultadoAntiguo.elementos[0].titulo = "Resultado obsoleto";
  antigua.resolver(resultadoAntiguo);
  assert.equal(await cargaAntigua, false);
  const html = superficie.renderizar();
  assert.match(html, /Resultado reciente/);
  assert.doesNotMatch(html, /Resultado obsoleto/);
});

test("la actualización envía CAS e idempotencia exactos y muestra el recibo", async () => {
  const llamadas = [];
  const cliente = crearDobleCliente({
    actualizar: async (...argumentos) => {
      llamadas.push(argumentos);
      return structuredClone(recibo("actualizar"));
    },
  });
  const { superficie } = crearSuperficie({ cliente });
  await superficie.activar();
  cambiar(superficie, "contenido_editable.titulo", "Título local actualizado");
  assert.equal(await superficie.guardar(), true);
  assert.equal(llamadas.length, 1);
  const [referencia, solicitud, limitesAPI, control] = llamadas[0];
  assert.equal(referencia, detalle().referencia_estado.referencia);
  assert.equal(solicitud.esquema, ESQUEMAS_BORRADORES.actualizar);
  assert.equal(solicitud.contenido_editable.titulo, "Título local actualizado");
  assert.equal(Object.hasOwn(solicitud, "identificador_publico"), false);
  assert.deepEqual(validarSolicitudActualizarBorrador(solicitud, limitesAPI), solicitud);
  assert.deepEqual(limitesAPI, opciones().limites);
  assert.deepEqual(control, { etag: detalle().etag, claveIdempotencia: CLAVE_IDEMPOTENCIA_A });
  assert.match(superficie.renderizar(), /Recibo administrativo del borrador/);
});

test("un 409 conserva el formulario local y permite rotar conscientemente la clave", async () => {
  const claves = [];
  const cliente = crearDobleCliente({
    actualizar: async (_referencia, _solicitud, _limites, control) => {
      claves.push(control.claveIdempotencia);
      throw new ErrorAPIBorradores("La clave ya acredita otra operación.", 409, undefined, {
        codigo: "conflicto_idempotencia",
        correlacion: "correlacion:borradores:0123456789abcdef",
      });
    },
  });
  const { superficie } = crearSuperficie({
    cliente,
    claves: [CLAVE_IDEMPOTENCIA_A, CLAVE_IDEMPOTENCIA_B],
  });
  await superficie.activar();
  cambiar(superficie, "contenido_editable.titulo", "Título local que debe sobrevivir");
  assert.equal(await superficie.guardar(), false);
  let html = superficie.renderizar();
  assert.match(html, /Título local que debe sobrevivir/);
  assert.match(html, /Conflicto de idempotencia \(HTTP 409\)/);
  assert.match(html, /Los cambios introducidos continúan en este editor/);
  assert.equal(await superficie.manejarAccion({ accion: "borradores-rotar-idempotencia" }), false);
  assert.deepEqual(claves, [CLAVE_IDEMPOTENCIA_A, CLAVE_IDEMPOTENCIA_B]);
  html = superficie.renderizar();
  assert.match(html, /Título local que debe sobrevivir/);
});

test("un 412 compara la revisión vigente sin pisar cambios y reaplica con su ETag", async () => {
  let lecturasDetalle = 0;
  const actualizaciones = [];
  const remoto = detalleRemoto();
  const cliente = crearDobleCliente({
    obtenerDetalle: async () => {
      lecturasDetalle += 1;
      return structuredClone(lecturasDetalle === 1 ? detalle() : remoto);
    },
    actualizar: async (...argumentos) => {
      actualizaciones.push(argumentos);
      if (actualizaciones.length === 1) {
        throw new ErrorAPIBorradores("La revisión base ya no está vigente.", 412, undefined, {
          codigo: "conflicto_revision",
          correlacion: "correlacion:borradores:fedcba9876543210",
        });
      }
      return structuredClone(recibo("actualizar"));
    },
  });
  const { superficie } = crearSuperficie({ cliente });
  await superficie.activar();
  cambiar(superficie, "contenido_editable.titulo", "Título local sin guardar");
  assert.equal(await superficie.guardar(), false);
  let html = superficie.renderizar();
  assert.match(html, /Conflicto de revisión CAS \(HTTP 412\)/);
  assert.match(html, /Título local sin guardar/);
  assert.match(html, /fedcba9876543210/);
  assert.equal(await superficie.manejarAccion({ accion: "borradores-cargar-vigente" }), true);
  html = superficie.renderizar();
  assert.match(html, /Comparación antes de resolver/);
  assert.match(html, /Título local sin guardar/);
  assert.match(html, /Título vigente del servidor/);
  assert.match(html, /He revisado la comparación/);
  cambiar(superficie, "confirmar_reaplicacion", "", "checkbox", true);
  assert.equal(await superficie.manejarAccion({ accion: "borradores-reaplicar-vigente" }), true);
  assert.equal(actualizaciones.length, 2);
  assert.equal(actualizaciones[1][1].contenido_editable.titulo, "Título local sin guardar");
  assert.equal(actualizaciones[1][3].etag, remoto.etag);
});

test("un alta confirmada conserva recibo aunque falle la lectura posterior", async () => {
  let lecturasDetalle = 0;
  let alta = null;
  const cliente = crearDobleCliente({
    obtenerDetalle: async () => {
      lecturasDetalle += 1;
      if (lecturasDetalle > 1) throw new Error("backend de lectura temporalmente caído");
      return structuredClone(detalle());
    },
    crear: async (...argumentos) => {
      alta = argumentos;
      return structuredClone(recibo("crear"));
    },
  });
  const { superficie } = crearSuperficie({ cliente });
  await superficie.activar();
  await superficie.manejarAccion({ accion: "borradores-nuevo" });
  cambiar(superficie, "identificador_publico", "bolsa-nueva-2026");
  cambiar(superficie, "codigo_version_publica", "version_inicial");
  cambiar(superficie, "expediente_ref", "expediente:seleccion:2026:99");
  cambiar(superficie, "contenido_editable.titulo", "Nueva bolsa temporal");
  cambiar(superficie, "contenido_editable.resumen", "Resumen del nuevo borrador.");
  cambiar(superficie, "contenido_editable.descripcion", "Descripción completa del nuevo borrador.");
  cambiar(superficie, "contenido_editable.plazos.0.referencia", "plazo:solicitudes:nueva");
  cambiar(superficie, "contenido_editable.plazos.0.tipo", "presentacion_solicitudes");
  cambiar(superficie, "contenido_editable.plazos.0.titulo", "Presentación de solicitudes");
  cambiar(superficie, "contenido_editable.plazos.0.descripcion", "Periodo habilitado.");
  cambiar(superficie, "contenido_editable.plazos.0.abre_en", "2026-09-01T08:00:00Z");
  cambiar(superficie, "contenido_editable.plazos.0.cierra_en", "2026-09-15T12:00:00Z");
  assert.equal(await superficie.guardar(), true);
  assert.equal(alta[0].esquema, ESQUEMAS_BORRADORES.crear);
  assert.equal(alta[0].plantilla_ref, opciones().plantillas[0].plantilla_ref);
  assert.equal(alta[0].identificador_publico, "bolsa-nueva-2026");
  assert.deepEqual(validarSolicitudCrearBorrador(alta[0], alta[1]), alta[0]);
  assert.deepEqual(alta[2], { claveIdempotencia: CLAVE_IDEMPOTENCIA_A });
  const html = superficie.renderizar();
  assert.match(html, /Guardado confirmado/);
  assert.match(html, /detalle actualizado no se pudo recargar/i);
  assert.match(html, /transaccion:borrador:2026:17/);
  assert.doesNotMatch(html, /data-borrador-form="editor"/);
});

test("el contrato DOM es accesible, adaptable y no conecta storage ni presentación", async () => {
  const [controlador, vista, portal, estilos] = await Promise.all([
    readFile(new URL("portal-borradores-ui.js", directorio), "utf8"),
    readFile(new URL("portal-borradores-vista.js", directorio), "utf8"),
    readFile(new URL("portal.js", directorio), "utf8"),
    readFile(new URL("portal-flujos.css", directorio), "utf8"),
  ]);
  const codigoBorradores = `${controlador}\n${vista}`;
  assert.doesNotMatch(codigoBorradores, /localStorage|sessionStorage|document\.cookie/);
  assert.doesNotMatch(codigoBorradores, /datos-presentacion|presentacion=rrhh/);
  assert.match(vista, /<label class="campo"/);
  assert.match(vista, /<fieldset class="grupo-editor/);
  assert.match(vista, /role="alert"/);
  assert.match(vista, /aria-live="polite"/);
  assert.match(vista, /<caption>/);
  assert.match(estilos, /\.espacio-borradores/);
  assert.match(estilos, /@media \(max-width: 1040px\)[\s\S]*\.espacio-borradores/);
  assert.match(estilos, /@media \(max-width: 780px\)[\s\S]*\.campos-editor-dos/);
  assert.match(portal, /const superficie = estado\.modoPresentacion \? superficieBorradoresPresentacion : superficieBorradores/);
  assert.match(portal, /crearClienteImpl: \(\) => moduloBorradores\.crearClienteBorradoresPresentacion\(\)/);
  assert.match(portal, /import\("\.\/portal-borradores-demo-cliente\.js/);
  assert.doesNotMatch(portal, /^import .*portal-borradores-demo-cliente/m,
    "el adaptador DEMO debe poder excluirse físicamente de producción");
});
