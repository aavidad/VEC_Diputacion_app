import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import {
  ErrorAPIBorradores,
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

function crearSuperficie({ cliente, claves } = {}) {
  const anuncios = [];
  const cambios = [];
  let indiceClave = 0;
  const superficie = crearSuperficieBorradoresPortal({
    escaparHTML,
    anunciar: (mensaje) => anuncios.push(mensaje),
    alCambiar: () => cambios.push("cambio"),
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

test("la navegación comprueba capacidad sin leer la bandeja y reutiliza las opciones al entrar", async () => {
  const llamadas = [];
  const cliente = crearDobleCliente({
    obtenerOpciones: async () => { llamadas.push("opciones"); return structuredClone(opciones()); },
    listar: async () => { llamadas.push("lista"); return structuredClone(lista()); },
    obtenerDetalle: async () => { llamadas.push("detalle"); return structuredClone(detalle()); },
  });
  const { superficie } = crearSuperficie({ cliente });
  assert.equal(await superficie.comprobarDisponibilidad(), true);
  assert.deepEqual(llamadas, ["opciones"], "el acceso no debe anticipar una lectura de expedientes");
  assert.deepEqual(superficie.obtenerAcceso(), {
    disponible: true,
    vista: "elaboracion",
    estado: "disponible",
    etiqueta: "Borradores disponibles",
  });
  assert.equal(await superficie.activar(), true);
  assert.deepEqual(llamadas, ["opciones", "lista", "detalle"]);
});

test("una denegación sobrevenida retira de inmediato lista, detalle y edición", async () => {
  let denegada = false;
  const cliente = crearDobleCliente({
    obtenerOpciones: async () => {
      if (denegada) throw new ErrorAPIBorradores("Acceso retirado.", 403, undefined, {
        codigo: "autorizacion_denegada",
      });
      return structuredClone(opciones());
    },
  });
  const { superficie } = crearSuperficie({ cliente });
  await superficie.activar();
  assert.match(superficie.renderizar(), /Auxiliar administrativo/);
  denegada = true;
  assert.equal(await superficie.comprobarDisponibilidad({ forzar: true }), false);
  const html = superficie.renderizar();
  assert.doesNotMatch(html, /Auxiliar administrativo/);
  assert.doesNotMatch(html, /data-borrador-form="editor"/);
  assert.equal(superficie.obtenerAcceso().estado, "denegado");
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

test("la superficie delega exclusivamente la identidad al canal interno", async () => {
  let clientesCreados = 0;
  let configuracionCliente;
  const cerrada = crearSuperficieBorradoresPortal({
    escaparHTML,
    anunciar: () => {},
    alCambiar: () => {},
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
  assert.equal(configuracionCliente, undefined);
  const html = cerrada.renderizar();
  assert.match(html, /Bandeja de borradores/);
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
  const { signal, ...controlSinSignal } = control;
  assert.equal(signal instanceof AbortSignal, true);
  assert.deepEqual(controlSinSignal, { etag: detalle().etag, claveIdempotencia: CLAVE_IDEMPOTENCIA_A });
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
  const { signal, ...controlSinSignal } = alta[2];
  assert.equal(signal instanceof AbortSignal, true);
  assert.deepEqual(controlSinSignal, { claveIdempotencia: CLAVE_IDEMPOTENCIA_A });
  const html = superficie.renderizar();
  assert.match(html, /Guardado confirmado/);
  assert.match(html, /detalle actualizado no se pudo recargar/i);
  assert.match(html, /transaccion:borrador:2026:17/);
  assert.doesNotMatch(html, /data-borrador-form="editor"/);
});

function errorAcceso(estadoHTTP, camino) {
  return new ErrorAPIBorradores(`Acceso retirado en ${camino}.`, estadoHTTP, undefined, {
    codigo: `acceso_retirado_${camino}`,
  });
}

function afirmarRevocada(superficie) {
  assert.equal(superficie.obtenerAcceso().estado, "denegado");
  const html = superficie.renderizar();
  assert.doesNotMatch(html, /Auxiliar administrativo/);
  assert.doesNotMatch(html, /data-borrador-form="editor"/);
  assert.doesNotMatch(html, /Recibo administrativo/);
  assert.doesNotMatch(html, /Título local privado/);
}

for (const estadoHTTP of [401, 403]) {
  test(`un ${estadoHTTP} de listado revoca la superficie completa`, async () => {
    let lecturas = 0;
    let signal;
    const cliente = crearDobleCliente({
      listar: async (selector) => {
        lecturas += 1;
        if (lecturas === 1) return structuredClone(lista());
        signal = selector.signal;
        throw errorAcceso(estadoHTTP, "lista");
      },
    });
    const { superficie } = crearSuperficie({ cliente });
    await superficie.activar();
    assert.equal(await superficie.aplicarFiltro({ texto: "privado" }), false);
    assert.equal(signal.aborted, true);
    afirmarRevocada(superficie);
  });

  test(`un ${estadoHTTP} de detalle revoca la superficie completa`, async () => {
    let lecturas = 0;
    let signal;
    const cliente = crearDobleCliente({
      obtenerDetalle: async (_referencia, _limites, control) => {
        lecturas += 1;
        if (lecturas === 1) return structuredClone(detalle());
        signal = control.signal;
        throw errorAcceso(estadoHTTP, "detalle");
      },
    });
    const { superficie } = crearSuperficie({ cliente });
    await superficie.activar();
    assert.equal(await superficie.manejarAccion({
      accion: "borradores-abrir", id: detalle().referencia_estado.referencia,
    }), false);
    assert.equal(signal.aborted, true);
    afirmarRevocada(superficie);
  });

  test(`un ${estadoHTTP} de guardado revoca y elimina los cambios locales`, async () => {
    let signal;
    const cliente = crearDobleCliente({
      actualizar: async (_referencia, _solicitud, _limites, control) => {
        signal = control.signal;
        throw errorAcceso(estadoHTTP, "guardado");
      },
    });
    const { superficie } = crearSuperficie({ cliente });
    await superficie.activar();
    cambiar(superficie, "contenido_editable.titulo", "Título local privado");
    assert.equal(await superficie.guardar(), false);
    assert.equal(signal.aborted, true);
    afirmarRevocada(superficie);
  });

  test(`un ${estadoHTTP} del listado posguardado revoca incluso con recibo previo`, async () => {
    let lecturas = 0;
    let signal;
    const cliente = crearDobleCliente({
      listar: async (selector) => {
        lecturas += 1;
        if (lecturas === 1) return structuredClone(lista());
        signal = selector.signal;
        throw errorAcceso(estadoHTTP, "lista_posguardado");
      },
    });
    const { superficie } = crearSuperficie({ cliente });
    await superficie.activar();
    cambiar(superficie, "contenido_editable.titulo", "Título local privado");
    assert.equal(await superficie.guardar(), false);
    assert.equal(signal.aborted, true);
    afirmarRevocada(superficie);
  });

  test(`un ${estadoHTTP} del detalle posguardado revoca incluso con recibo previo`, async () => {
    let lecturas = 0;
    let signal;
    const cliente = crearDobleCliente({
      obtenerDetalle: async (_referencia, _limites, control) => {
        lecturas += 1;
        if (lecturas === 1) return structuredClone(detalle());
        signal = control.signal;
        throw errorAcceso(estadoHTTP, "detalle_posguardado");
      },
    });
    const { superficie } = crearSuperficie({ cliente });
    await superficie.activar();
    cambiar(superficie, "contenido_editable.titulo", "Título local privado");
    assert.equal(await superficie.guardar(), false);
    assert.equal(signal.aborted, true);
    afirmarRevocada(superficie);
  });

  test(`un ${estadoHTTP} de lectura CAS revoca la superficie completa`, async () => {
    let lecturas = 0;
    let signal;
    const cliente = crearDobleCliente({
      actualizar: async () => {
        throw new ErrorAPIBorradores("Conflicto CAS.", 412, undefined, { codigo: "conflicto_cas" });
      },
      obtenerDetalle: async (_referencia, _limites, control) => {
        lecturas += 1;
        if (lecturas === 1) return structuredClone(detalle());
        signal = control.signal;
        throw errorAcceso(estadoHTTP, "cas");
      },
    });
    const { superficie } = crearSuperficie({ cliente });
    await superficie.activar();
    cambiar(superficie, "contenido_editable.titulo", "Título local privado");
    assert.equal(await superficie.guardar(), false);
    assert.equal(await superficie.manejarAccion({ accion: "borradores-cargar-vigente" }), false);
    assert.equal(signal.aborted, true);
    afirmarRevocada(superficie);
  });
}

test("una respuesta de guardado tardía no repuebla tras revocar el acceso", async () => {
  const tardia = diferida();
  let denegar = false;
  let signalGuardado;
  const cliente = crearDobleCliente({
    obtenerOpciones: async () => {
      if (denegar) throw errorAcceso(403, "comprobacion");
      return structuredClone(opciones());
    },
    actualizar: async (_referencia, _solicitud, _limites, control) => {
      signalGuardado = control.signal;
      return tardia.promesa;
    },
  });
  const { superficie } = crearSuperficie({ cliente });
  await superficie.activar();
  cambiar(superficie, "contenido_editable.titulo", "Título local privado");
  const guardado = superficie.guardar();
  await Promise.resolve();
  denegar = true;
  assert.equal(await superficie.comprobarDisponibilidad({ forzar: true }), false);
  assert.equal(signalGuardado.aborted, true);
  tardia.resolver(structuredClone(recibo("actualizar")));
  assert.equal(await guardado, false);
  afirmarRevocada(superficie);
});

test("un detalle posguardado tardío no repuebla tras revocar el acceso", async () => {
  const lecturaIniciada = diferida();
  const tardia = diferida();
  let denegar = false;
  let lecturas = 0;
  let signalLectura;
  const cliente = crearDobleCliente({
    obtenerOpciones: async () => {
      if (denegar) throw errorAcceso(401, "comprobacion");
      return structuredClone(opciones());
    },
    obtenerDetalle: async (_referencia, _limites, control) => {
      lecturas += 1;
      if (lecturas === 1) return structuredClone(detalle());
      signalLectura = control.signal;
      lecturaIniciada.resolver();
      return tardia.promesa;
    },
  });
  const { superficie } = crearSuperficie({ cliente });
  await superficie.activar();
  cambiar(superficie, "contenido_editable.titulo", "Título local privado");
  const guardado = superficie.guardar();
  await lecturaIniciada.promesa;
  denegar = true;
  assert.equal(await superficie.comprobarDisponibilidad({ forzar: true }), false);
  assert.equal(signalLectura.aborted, true);
  tardia.resolver(detalleRemoto());
  assert.equal(await guardado, false);
  afirmarRevocada(superficie);
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
