import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { obtenerDatosPresentacion } from "../../datos-presentacion.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";
import { crearAdaptadorContratacionTemporalPresentacion } from "./adaptador-presentacion.js";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { renderizarCuadro, renderizarDocumentos, renderizarExpediente } from "./componentes-expedientes.js";
import {
  CAPACIDADES_CONTRATACION_TEMPORAL as CAP,
  validarAuditoriaContratacionTemporal,
  validarComandoActuacion,
  validarCuadroContratacionTemporal,
  validarDocumentosContratacionTemporal,
  validarExpedienteContratacionTemporal,
  validarReciboActuacion,
} from "./contrato-expedientes.js";
import { crearBorradorAlta, crearComandoAlta } from "./contrato.js";
import {
  crearAuditoriaContratacionTemporalPresentacion,
  crearCuadroContratacionTemporalPresentacion,
  crearDocumentosContratacionTemporalPresentacion,
  crearExpedienteContratacionTemporalPresentacion,
} from "./datos-presentacion.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import {
  crearEjecutorAltaConRefresco,
  montarModuloContratacionTemporal,
  renderizarModuloContratacionTemporal,
} from "./vista-expedientes.js";

function contexto(perfil = "administrador") {
  return crearContextoActorPresentacionDesdeSesion(
    obtenerDatosPresentacion(perfil).sesion,
  );
}

function adaptador(perfil = "administrador") {
  return crearAdaptadorContratacionTemporalPresentacion({
    contextoActor: contexto(perfil),
  });
}

function presentadorDe(fuente, capacidades = fuente.capacidades) {
  return crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades });
}

test("la continuidad real abre el índice documental sin fingir envío GINPIX ni consulta B11", () => {
  const expediente = { ...crearExpedienteContratacionTemporalPresentacion(),
    demostracion: false, version: 8 };
  const estado = { vista: "expediente", carga: "listo", expediente,
    expediente_ref: expediente.expediente_ref, tarea_ref: expediente.tareas[0].tarea_ref,
    navegacion: { documentos: true }, ocupado: false,
    actualizacion_pendiente: false, resultado_indeterminado: false };
  const t = crearTraductorExpedientesContratacion();
  const html = renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");

  assert.match(html, /Documentos y continuidad de la incorporación/u);
  assert.match(html, /data-ct-exp-vista="documentos">Consultar documentos del expediente/u);
  assert.match(html, /recibo de incorporación confirmado/u);
  assert.doesNotMatch(html, /data-ct-ficha-ginpix-descargar|data-ct-seguimiento-consultar|ginpix\.enviar/u);
  assert.doesNotMatch(renderizarExpediente({ ...estado, navegacion: { documentos: false } }, t, "es-ES", "Europe/Madrid"), /ct-exp-continuidad/u);
  assert.doesNotMatch(renderizarExpediente({ ...estado, expediente: { ...expediente, version: 7 } }, t, "es-ES", "Europe/Madrid"), /ct-exp-continuidad/u);
});

test("documentos explica ficha manual y seguimiento con textos inyectados escapados", () => {
  const expediente = { ...crearExpedienteContratacionTemporalPresentacion(),
    demostracion: false, version: 8 };
  const t = crearTraductorExpedientesContratacion({
    continuidad_documentos_ficha_limite: "Sin transmisión <externa>",
  });
  const html = renderizarDocumentos({ expediente, documentos: { documentos: [] } }, t);

  assert.match(html, /Ficha manual para GINPIX/u);
  assert.match(html, /Sin transmisión &lt;externa&gt;/u);
  assert.match(html, /Seguimiento de la incorporación/u);
  assert.match(html, /El seguimiento original se consulta desde el mismo recibo/u);
  assert.match(html, /data-ct-exp-vista="expediente">Expediente/u);
  assert.doesNotMatch(html, /<externa>|data-ct-ficha-ginpix-descargar|data-ct-seguimiento-consultar/u);
});

test("el índice documental mantiene el regreso al expediente y explica el estado vacío", () => {
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const t = crearTraductorExpedientesContratacion();
  const html = renderizarDocumentos({ expediente, documentos: { documentos: [] } }, t);

  assert.match(html, /class="panel ct-exp-documentos" aria-labelledby="ct-exp-documentos-titulo"/u);
  assert.match(html, /class="cabecera-panel ct-exp-subcabecera ct-exp-documentos-cabecera"/u);
  assert.match(html, /id="ct-exp-documentos-titulo" tabindex="-1"/u);
  assert.match(html, /data-ct-exp-vista="expediente">Expediente<\/button>/u);
  assert.match(html, /role="status">No hay datos disponibles en este panel\.<\/p>/u);
  assert.doesNotMatch(html, /<table/u);
  assert.doesNotMatch(html, /data-ct-ficha-ginpix-descargar|data-ct-exp-efecto="[^"]*ginpix\.enviar/u);
});

test("el índice documental conserva cada estado y escapa referencias en la tabla", () => {
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const t = crearTraductorExpedientesContratacion();
  const html = renderizarDocumentos({ expediente, documentos: { documentos: [{
    titulo: "Ficha <GINPIX>", documento_ref: "documento:<interno>", tipo: "JSON",
    version: 1, estado: "Preparado", firma: "Sin firma", fecha: "24/09/2026",
    descarga_disponible: true,
  }] } }, t);

  assert.match(html, /Ficha &lt;GINPIX&gt;<small><code>documento:&lt;interno&gt;<\/code><\/small>/u);
  assert.match(html, /Preparado<\/td><td>Sin firma<\/td>/u);
  assert.match(html, /Descarga pendiente de conectar/u);
  assert.doesNotMatch(html, /<GINPIX>|data-ct-ficha-ginpix-descargar/u);
});

test("el listado precede al trabajo auxiliar y cada expediente ofrece un resumen inicial cerrado", async () => {
  const fuente = adaptador();
  const cuadro = await fuente.listar();
  const t = crearTraductorExpedientesContratacion();
  const html = renderizarCuadro({
    vista: "cuadro",
    carga: "listo",
    cuadro,
    filtros: { texto: "", estado: "", fase: "" },
  }, t);
  const primerExpediente = cuadro.expedientes[0];

  assert.ok(html.indexOf("ct-exp-listado") < html.indexOf("ct-exp-operativo"));
  const resumenId = `ct-exp-resumen-${primerExpediente.expediente_ref}`;
  assert.match(html, new RegExp(
    `data-ct-exp-resumen aria-controls="${resumenId}" aria-expanded="false"`,
    "u",
  ));
  assert.match(html, new RegExp(
    `<tr class="ct-exp-fila-resumen" id="${resumenId}"[\\s\\S]*?data-ct-exp-resumen-fila[\\s\\S]*?hidden>`,
    "u",
  ));
  assert.match(html, new RegExp(
    `<tr class="ct-exp-fila"[\\s\\S]*?${primerExpediente.numero_visible}[\\s\\S]*?</tr>\\s*<tr class="ct-exp-fila-resumen"`,
    "u",
  ));
  assert.match(html, new RegExp(`aria-label="Resumen del expediente ${primerExpediente.numero_visible}"`, "u"));
  assert.match(html, new RegExp(
    `data-ct-exp-abrir="${primerExpediente.expediente_ref}"[\\s\\S]*?Abrir expediente completo`,
    "u",
  ));
});

test("el resumen inicial escapa datos, marca la fase y omite las columnas de la fila", () => {
  const t = crearTraductorExpedientesContratacion();
  const html = renderizarCuadro({
    vista: "cuadro",
    carga: "listo",
    cuadro: {
      demostracion: false,
      indicadores: [],
      expedientes: [{
        expediente_ref: 'expediente:ct:resumen:&lt;script&gt;',
        numero_visible: "CT-<1>",
        centro: "Centro <seguro>",
        categoria: "Categoría <segura>",
        modalidad: "Modalidad <segura>",
        estado_clave: "en_curso",
        estado: "En curso <seguro>",
        fase_clave: "analisis_rrhh",
        fase_actual: "Análisis <seguro>",
        plazo: "Hoy <seguro>",
        fecha_solicitud: "2026-09-21T10:00:00Z",
        version: 3,
      }],
    },
    filtros: { texto: "", estado: "", fase: "" },
  }, t);

  assert.match(html, /aria-label="Resumen del expediente CT-&lt;1&gt;"/u);
  assert.match(html, /Centro &lt;seguro&gt;/u);
  assert.match(html, /class="ct-exp-chip ct-fase-en_curso">En curso &lt;seguro&gt;<\/span>/u);
  assert.match(html, /data-ct-exp-abrir="expediente:ct:resumen:&amp;lt;script&amp;gt;"/u);
  assert.match(html, /data-ct-fase="analisis_rrhh"/u);
  const ficha = html.match(/<tr class="ct-exp-fila-resumen"[\s\S]*?<\/tr>/u)?.[0];
  assert.ok(ficha);
  assert.match(ficha, /aria-current="step">Análisis de RRHH/u);
  assert.match(ficha, /Solicitud registrada<\/dt><dd>/u);
  assert.match(ficha, /Versión<\/dt><dd>3<\/dd>/u);
  for (const columna of ["Centro", "Categoría", "Modalidad", "Estado", "Fase actual", "Plazo"]) {
    assert.doesNotMatch(ficha, new RegExp(`<dt>${columna}</dt>`, "u"));
  }
  assert.doesNotMatch(html, /<script>/u);
});

test("la bandeja presenta una referencia de centro legible y deja la técnica en title", () => {
  const t = crearTraductorExpedientesContratacion();
  const html = renderizarCuadro({ vista: "cuadro", carga: "listo", cuadro: {
    demostracion: false, indicadores: [], expedientes: [{ expediente_ref: "expediente:ct:centro", numero_visible: "2026/CT-000042", centro: "centro:desarrollo:001", categoria: "Auxiliar", modalidad: "Sustitución", estado_clave: "en_curso", estado: "En curso", fase_actual: "Solicitud", plazo: "Sin plazo" }],
  }, filtros: { texto: "", estado: "", fase: "" } }, t);
  assert.match(html, /title="centro:desarrollo:001">Centro desarrollo · 001/u);
  assert.match(html, />Sustitución<\/td>/u);
});

test("la modalidad ausente se rotula con precisión y los metadatos ausentes se omiten", () => {
  const t = crearTraductorExpedientesContratacion();
  const html = renderizarCuadro({ vista: "cuadro", carga: "listo", cuadro: {
    demostracion: false, indicadores: [], expedientes: [{
      expediente_ref: "expediente:ct:sin-modalidad", numero_visible: "2026/CT-000043",
      centro: "Centro", categoria: "Auxiliar", modalidad: "—", estado_clave: "en_curso",
      estado: "En curso", fase_clave: "solicitud", fase_actual: "Solicitud", plazo: "—",
    }],
  }, filtros: { texto: "", estado: "", fase: "" } }, t);
  assert.match(html, /title="La consulta de la bandeja no devuelve la modalidad de este expediente\.">—<\/td>/u);
  assert.doesNotMatch(html, /ct-exp-nota-tabla|en Modalidad: la consulta/u);
  const ficha = html.match(/<tr class="ct-exp-fila-resumen"[\s\S]*?<\/tr>/u)?.[0];
  assert.ok(ficha);
  assert.doesNotMatch(ficha, /ct-exp-resumen-datos[\s\S]*?<div>/u);
});

test("el control de resumen abre una fila, cierra las demás y no selecciona expediente", async () => {
  const fuente = adaptador();
  const presentador = presentadorDe(fuente);
  await presentador.cargar();
  const eventos = new Map();
  const crearControl = (id) => {
    const atributos = new Map([["aria-controls", id], ["aria-expanded", "false"]]);
    const control = {
      getAttribute: (nombre) => atributos.get(nombre) || null,
      setAttribute: (nombre, valor) => atributos.set(nombre, valor),
      closest: (selector) => selector === "[data-ct-exp-resumen]" ? control : null,
    };
    return { control, atributos };
  };
  const primero = crearControl("ct-exp-resumen-uno");
  const segundo = crearControl("ct-exp-resumen-dos");
  const filas = new Map([
    ["ct-exp-resumen-uno", { hidden: true }],
    ["ct-exp-resumen-dos", { hidden: true }],
  ]);
  const raiz = {
    innerHTML: "",
    ownerDocument: { getElementById: (id) => filas.get(id) || null },
    addEventListener: (tipo, manejador) => eventos.set(tipo, manejador),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: () => null,
    querySelectorAll: (selector) => selector === "[data-ct-exp-resumen]"
      ? [primero.control, segundo.control] : [],
    contains: () => true,
  };
  const montaje = await montarModuloContratacionTemporal({ raiz, presentador });
  try {
    const click = eventos.get("click");
    await click({ target: primero.control, preventDefault() {} });
    assert.equal(primero.atributos.get("aria-expanded"), "true");
    assert.equal(filas.get("ct-exp-resumen-uno").hidden, false);
    assert.equal(segundo.atributos.get("aria-expanded"), "false");
    assert.equal(filas.get("ct-exp-resumen-dos").hidden, true);

    await click({ target: segundo.control, preventDefault() {} });
    assert.equal(primero.atributos.get("aria-expanded"), "false");
    assert.equal(filas.get("ct-exp-resumen-uno").hidden, true);
    assert.equal(segundo.atributos.get("aria-expanded"), "true");
    assert.equal(filas.get("ct-exp-resumen-dos").hidden, false);

    await click({ target: segundo.control, preventDefault() {} });
    assert.equal(segundo.atributos.get("aria-expanded"), "false");
    assert.equal(filas.get("ct-exp-resumen-dos").hidden, true);
    assert.equal(presentador.obtenerEstado().vista, "cuadro");
    assert.equal(presentador.obtenerEstado().expediente_ref, "");
  } finally {
    montaje.desmontar();
  }
});

test("el alta disponible no concede capacidades de consulta", async () => {
  let consultas = 0;
  const fuente = {
    capacidades: [],
    async listar() {
      consultas += 1;
      throw new Error("cuadro todavía no compuesto");
    },
    async obtener() { throw new Error("detalle no compuesto"); },
    async ejecutar() { throw new Error("actuación no compuesta"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente,
    capacidades: [],
    altaDisponible: true,
  });

  presentador.cambiarVista("alta");
  await presentador.cargar();
  assert.equal(consultas, 1);
  assert.equal(presentador.obtenerEstado().vista, "alta");
  assert.equal(presentador.obtenerEstado().carga, "error");
  assert.deepEqual(fuente.capacidades, []);
  assert.throws(
    () => crearPresentadorExpedientesContratacionTemporal({
      fuente, capacidades: [], altaDisponible: "si",
    }),
    /disponibilidad de alta no válida/u,
  );
});

test("un 502 de cuadro conserva el error al navegar y no muestra llamamiento", async () => {
  let consultas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      consultas += 1;
      assert.equal(ruta, "/api/vec/contratacion-temporal/cuadro/consultas");
      assert.equal(opciones.method, "POST");
      return new Response(JSON.stringify({ error: {
        codigo: "resultado_no_confiable",
        clave_i18n: "api.contratacion_temporal.consulta_rrhh.error.resultado_no_confiable",
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
      } }), { status: 502, headers: { "Content-Type": "application/json" } });
    },
  });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente });
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente, capacidades: [], altaDisponible: true,
  });
  const eventos = new Map();
  const raiz = {
    innerHTML: "",
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: () => null,
    contains: () => true,
  };
  const montaje = await montarModuloContratacionTemporal({
    raiz, presentador, llamamiento: { cliente },
  });
  try {
    for (const vista of ["cuadro", "alta", "cuadro"]) {
      await eventos.get("click")({
        target: { closest: (selector) => selector === "[data-ct-exp-vista]"
          ? { dataset: { ctExpVista: vista } } : null },
        preventDefault() {},
      });
      const estado = presentador.obtenerEstado();
      assert.equal(estado.carga, "error");
      assert.equal(estado.cuadro, null);
      assert.equal(estado.expediente, null);
      assert.equal(estado.mensaje_clave, "estado_error_carga");
      assert.equal(estado.tipo_mensaje, "error");
      assert.match(raiz.innerHTML, /role="alert"/u);
      assert.doesNotMatch(raiz.innerHTML, /Cuadro de contratación temporal actualizado/u);
      assert.doesNotMatch(raiz.innerHTML, /data-ct-exp-llamamiento/u);
      if (vista === "cuadro") {
        assert.match(raiz.innerHTML, /data-ct-exp-accion="reintentar"/u);
      }
    }
    assert.equal(consultas, 1, "navegar no reintenta ni crea un efecto");
    assert.deepEqual(fuente.capacidades, []);
  } finally {
    montaje.desmontar();
  }
});

test("navegar sin resultado o cancelar una carga no anuncia un cuadro actualizado", async () => {
  let rechazar;
  const fuente = {
    listar: () => new Promise((_, reject) => { rechazar = reject; }),
    obtener() {}, ejecutar() {},
  };
  const presentador = presentadorDe(fuente, [CAP.consultarCuadro]);
  presentador.cambiarVista("cuadro");
  assert.equal(presentador.obtenerEstado().carga, "inicial");
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_inicial");
  const carga = presentador.cargar();
  presentador.cambiarVista("cuadro");
  assert.equal(presentador.obtenerEstado().carga, "inicial");
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_lectura_cancelada");
  rechazar(new Error("consulta sintética cancelada"));
  await carga;
  assert.equal(presentador.obtenerEstado().carga, "inicial");
  const denegado = presentadorDe(fuente, []);
  denegado.cambiarVista("cuadro");
  assert.equal(denegado.obtenerEstado().carga, "denegado");
  assert.equal(denegado.obtenerEstado().mensaje_clave, "estado_denegado");
});

function estadoVista(expediente, tareaRef = expediente.tareas[0].tarea_ref) {
  return {
    vista: "expediente",
    carga: "listo",
    cuadro: null,
    expediente,
    documentos: null,
    auditoria: null,
    expediente_ref: expediente.expediente_ref,
    tarea_ref: tareaRef,
    filtros: { texto: "", estado: "", fase: "" },
    ocupado: false,
    actualizacion_pendiente: false,
    recibo: null,
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  };
}

function comandoDe(expediente, tarea, accion) {
  return validarComandoActuacion({
    esquema: "vec.contratacion_temporal.actuacion.v1",
    expediente_ref: expediente.expediente_ref,
    version_esperada: expediente.version,
    tarea_ref: tarea.tarea_ref,
    accion_ref: accion.accion_ref,
    datos: {},
  });
}

function expedienteConAccionSinteticaDisponible() {
  const entrada = crearExpedienteContratacionTemporalPresentacion();
  const tarea = entrada.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-solicitud");
  tarea.acciones[0] = {
    ...tarea.acciones[0],
    disponible: true,
    motivo_no_disponible: "",
  };
  return validarExpedienteContratacionTemporal(entrada);
}

test("el espacio operativo separa tareas y distribución en paneles legibles", async () => {
  const [estilos, portal, tema] = await Promise.all([
    readFile(new URL("./expedientes-operativo.css", import.meta.url), "utf8"),
    readFile(new URL("../../portal.css", import.meta.url), "utf8"),
    readFile(new URL("../../../comun/tema-vec.css", import.meta.url), "utf8"),
  ]);
  assert.match(
    estilos,
    /\.ct-exp-mis-tareas,\s*\n\.ct-exp-distribucion\s*\{[\s\S]*border:[^;]+;[\s\S]*background:/u,
  );
  assert.match(estilos, /\.ct-exp-operativo\s*\{[\s\S]*grid-template-columns:/u);
  assert.match(portal, /^\s*@import\s+url\(\s*["']\.\.\/comun\/tema-vec\.css["']\s*\)\s*;/mu);
  const base = tema.match(/:root\s*\{([^}]*)\}/u)?.[1];
  assert.ok(base, "el tema común debe declarar sus tokens base en :root");
  for (const token of [
    "--portal-espacio-1", "--portal-espacio-2", "--portal-espacio-3",
    "--portal-espacio-4", "--portal-radio-md", "--portal-radio-lg",
    "--portal-sombra-sm", "--portal-tinta-suave",
  ]) {
    const declaracion = new RegExp(`^[ \\t]*${token}[ \\t]*:`, "gmu");
    assert.equal([...base.matchAll(declaracion)].length, 1,
      `${token} debe tener una sola definición base en el tema común`);
    assert.doesNotMatch(portal, declaracion,
      `${token} no debe redefinirse en portal.css`);
  }
});

test("los cuatro contratos rechazan extras, duplicados, cruces y valores no canónicos", () => {
  const cuadro = crearCuadroContratacionTemporalPresentacion();
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const documentos = crearDocumentosContratacionTemporalPresentacion();
  const auditoria = crearAuditoriaContratacionTemporalPresentacion();
  assert.equal(validarCuadroContratacionTemporal(cuadro).expedientes.length, 5);
  assert.equal(validarExpedienteContratacionTemporal(expediente).tareas.length, 18);
  assert.equal(validarDocumentosContratacionTemporal(documentos).documentos.length, 7);
  assert.equal(validarAuditoriaContratacionTemporal(auditoria).actuaciones.length, 8);
  assert.throws(() => validarCuadroContratacionTemporal({ ...cuadro, secreto: "x" }), /cerrado/);
  assert.throws(() => validarExpedienteContratacionTemporal({
    ...expediente,
    tareas: [...expediente.tareas, expediente.tareas[0]],
  }), /duplicados/);
  assert.throws(() => validarExpedienteContratacionTemporal({
    ...expediente,
    tareas: expediente.tareas.map((tarea, indice) => (
      indice === 0 ? { ...tarea, fase_ref: "fase-inexistente" } : tarea
    )),
  }), /fase inexistente/);
  assert.throws(() => validarDocumentosContratacionTemporal({
    ...documentos,
    documentos: documentos.documentos.map((documento, indice) => (
      indice === 0 ? { ...documento, extra: true } : documento
    )),
  }), /cerrado/);
  assert.throws(() => validarAuditoriaContratacionTemporal({
    ...auditoria,
    actuaciones: [...auditoria.actuaciones, auditoria.actuaciones[0]],
  }), /duplicados/);
  assert.throws(() => validarComandoActuacion({
    esquema: "vec.contratacion_temporal.actuacion.v1",
    expediente_ref: expediente.expediente_ref,
    version_esperada: expediente.version,
    tarea_ref: expediente.tareas[13].tarea_ref,
    accion_ref: expediente.tareas[13].acciones[0].accion_ref,
    datos: { campo: { anidado: true } },
  }), /no válido/);
  assert.throws(() => validarReciboActuacion({
    esquema: "vec.contratacion_temporal.recibo-actuacion.v1",
    recibo_ref: "rec-prueba-001",
    expediente_ref: expediente.expediente_ref,
    numero_visible: expediente.numero_visible,
    version: 13,
    actuacion: "Prueba",
    estado_resultante: "Registrado",
    registrada_en: "2026-07-23T10:00:00Z",
    token: "prohibido",
  }), /cerrado/);
});

test("filtro, selección y proyecciones segregadas conservan referencia y versión", async () => {
  const fuente = adaptador();
  const presentador = presentadorDe(fuente);
  assert.deepEqual(presentador.obtenerEstado().navegacion, {
    documentos: true,
    auditoria: true,
  });
  const navegacion = renderizarModuloContratacionTemporal(presentador.obtenerEstado());
  assert.match(navegacion, /data-ct-exp-vista="documentos"/u);
  assert.match(navegacion, /data-ct-exp-vista="auditoria"/u);
  await presentador.cargar({ texto: "Secretaría", estado: "", fase: "" });
  let estado = presentador.obtenerEstado();
  assert.equal(estado.cuadro.expedientes.length, 1);
  const referencia = estado.cuadro.expedientes[0].expediente_ref;
  await presentador.seleccionarExpediente(referencia, "documentos");
  estado = presentador.obtenerEstado();
  assert.equal(estado.expediente.expediente_ref, referencia);
  assert.equal(estado.documentos.expediente_ref, referencia);
  assert.equal(estado.documentos.version, estado.expediente.version);
  assert.equal(estado.auditoria, null);
  await presentador.seleccionarExpediente(referencia, "auditoria");
  estado = presentador.obtenerEstado();
  assert.equal(estado.auditoria.expediente_ref, referencia);
  assert.equal(estado.auditoria.version, estado.expediente.version);
  assert.equal(estado.documentos, null);
  await presentador.cargar({ texto: "sin coincidencias", estado: "", fase: "" });
  estado = presentador.obtenerEstado();
  assert.equal(estado.vista, "cuadro");
  assert.equal(estado.expediente_ref, "");
  assert.equal(estado.expediente, null);
});

test("cuadro y detalle son coherentes para las cinco referencias sintéticas", async () => {
  const fuente = adaptador();
  const cuadro = await fuente.listar();
  for (const resumen of cuadro.expedientes) {
    const detalle = await fuente.obtener(resumen.expediente_ref);
    assert.equal(detalle.numero_visible, resumen.numero_visible);
    assert.equal(detalle.version, resumen.version);
    const activas = detalle.tareas.filter((tarea) => (
      ["en_curso", "espera", "incidencia"].includes(tarea.estado_clave)
    ));
    if (resumen.estado_clave === "completado") {
      assert.equal(activas.length, 0);
      assert.ok(detalle.tareas.every(({ estado_clave: clave }) => clave === "completado"));
      assert.ok(detalle.fases.every(({ estado_clave: clave }) => clave === "completado"));
    } else {
      assert.equal(activas.length, 1, resumen.numero_visible);
      assert.equal(activas[0].estado_clave, resumen.estado_clave);
      const fase = detalle.fases.find(({ fase_ref }) => fase_ref === activas[0].fase_ref);
      assert.equal(fase.estado_clave, resumen.estado_clave);
    }
  }
});

test("RBAC se proyecta en HTML y vuelve a imponerse dentro del adaptador", async () => {
  const fuenteAdmin = adaptador("administrador");
  const fuenteTecnica = adaptador("tecnico");
  const referencia = (await fuenteAdmin.listar()).expedientes[0].expediente_ref;
  const presentadorAdmin = presentadorDe(fuenteAdmin);
  const presentadorTecnica = presentadorDe(fuenteTecnica);
  await Promise.all([presentadorAdmin.cargar(), presentadorTecnica.cargar()]);
  await Promise.all([
    presentadorAdmin.seleccionarExpediente(referencia),
    presentadorTecnica.seleccionarExpediente(referencia),
  ]);
  for (const presentador of [presentadorAdmin, presentadorTecnica]) {
    presentador.seleccionarTarea("tarea-formalizacion");
  }
  const t = crearTraductorExpedientesContratacion();
  const htmlAdmin = renderizarExpediente(
    presentadorAdmin.obtenerEstado(), t, "es-ES", "Europe/Madrid",
  );
  const htmlTecnica = renderizarExpediente(
    presentadorTecnica.obtenerEstado(), t, "es-ES", "Europe/Madrid",
  );
  assert.match(htmlAdmin, /data-ct-exp-efecto="preparar_borrador_firma_demo"[\s\S]*?>Preparar borrador para firma \(DEMO\)/);
  assert.match(
    htmlAdmin,
    /data-ct-exp-efecto="preparar_borrador_firma_demo"[\s\S]{0,500}?disabled/,
  );
  assert.match(htmlAdmin, /circuito Portafirmas P4, sus documentos, firmantes y orden siguen pendientes de definición por RRHH/u);
  assert.match(htmlTecnica, /data-ct-exp-efecto="preparar_borrador_firma_demo"[\s\S]{0,500}?disabled/);
  const detalle = await fuenteAdmin.obtener(referencia);
  const tarea = detalle.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-formalizacion");
  const accion = tarea.acciones.find(({ accion_ref }) => accion_ref === "preparar_borrador_firma_demo");
  const version = detalle.version;
  const auditoriaAntes = await fuenteAdmin.obtenerAuditoria(referencia);
  await assert.rejects(fuenteAdmin.ejecutar(comandoDe(detalle, tarea, accion)), /Portafirmas P4/u);
  assert.equal((await fuenteAdmin.obtener(referencia)).version, version);
  assert.deepEqual(await fuenteAdmin.obtenerAuditoria(referencia), auditoriaAntes);
});

test("una transición emite recibo, añade auditoría y no puede repetirse", async () => {
  const fuente = adaptador();
  const resumen = (await fuente.listar()).expedientes.find(
    ({ expediente_ref }) => expediente_ref === "exp-demo-contratacion-005484",
  );
  const antes = await fuente.obtener(resumen.expediente_ref);
  const auditoriaAntes = await fuente.obtenerAuditoria(resumen.expediente_ref);
  const tarea = antes.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-seleccion-candidato");
  const accion = tarea.acciones.find(
    ({ accion_ref }) => accion_ref === "seleccionar_candidato",
  );
  const comando = comandoDe(antes, tarea, accion);
  const recibo = await fuente.ejecutar(comando);
  const despues = await fuente.obtener(resumen.expediente_ref);
  const auditoriaDespues = await fuente.obtenerAuditoria(resumen.expediente_ref);
  assert.equal(recibo.version, antes.version + 1);
  assert.equal(despues.version, recibo.version);
  const tareaDespues = despues.tareas.find(({ tarea_ref }) => tarea_ref === tarea.tarea_ref);
  assert.equal(tareaDespues.recibo_ref, recibo.recibo_ref);
  assert.ok(tareaDespues.decision_ref);
  assert.equal(
    tareaDespues.acciones.find(({ accion_ref }) => accion_ref === accion.accion_ref).disponible,
    false,
  );
  assert.equal(auditoriaDespues.actuaciones.length, auditoriaAntes.actuaciones.length + 1);
  await assert.rejects(fuente.ejecutar({
    ...comando,
    version_esperada: recibo.version,
  }), /no está disponible/);
});

test("el presentador rechaza recibos cruzados y conserva el éxito si falla la recarga", async () => {
  const expediente = expedienteConAccionSinteticaDisponible();
  const cuadro = validarCuadroContratacionTemporal(crearCuadroContratacionTemporalPresentacion());
  const tarea = expediente.tareas[0];
  const accion = tarea.acciones[0];
  let lecturas = 0;
  const fuenteCruzada = {
    listar: async () => cuadro,
    obtener: async () => expediente,
    ejecutar: async () => validarReciboActuacion({
      esquema: "vec.contratacion_temporal.recibo-actuacion.v1",
      recibo_ref: "rec-prueba-cruzado",
      expediente_ref: expediente.expediente_ref,
      numero_visible: "2026/CT-99999",
      version: expediente.version + 1,
      actuacion: accion.etiqueta,
      estado_resultante: "Registrado",
      registrada_en: "2026-07-23T10:00:00Z",
    }),
  };
  const cruzado = presentadorDe(fuenteCruzada, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.enviarAnalisis,
  ]);
  await cruzado.cargar();
  await cruzado.seleccionarExpediente(expediente.expediente_ref);
  cruzado.seleccionarTarea(tarea.tarea_ref);
  await cruzado.ejecutarActuacion({ accionRef: accion.accion_ref });
  assert.equal(cruzado.obtenerEstado().mensaje_clave, "estado_error_actuacion");
  assert.equal(cruzado.obtenerEstado().recibo, null);

  const fuenteSinRefresco = {
    ...fuenteCruzada,
    obtener: async () => {
      lecturas += 1;
      if (lecturas > 1) throw new Error("detalle privado");
      return expediente;
    },
    ejecutar: async () => validarReciboActuacion({
      esquema: "vec.contratacion_temporal.recibo-actuacion.v1",
      recibo_ref: "rec-prueba-valido",
      expediente_ref: expediente.expediente_ref,
      numero_visible: expediente.numero_visible,
      version: expediente.version + 1,
      actuacion: accion.etiqueta,
      estado_resultante: "Registrado",
      registrada_en: "2026-07-23T10:00:00Z",
    }),
  };
  const sinRefresco = presentadorDe(fuenteSinRefresco, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.enviarAnalisis,
  ]);
  await sinRefresco.cargar();
  await sinRefresco.seleccionarExpediente(expediente.expediente_ref);
  sinRefresco.seleccionarTarea(tarea.tarea_ref);
  await sinRefresco.ejecutarActuacion({ accionRef: accion.accion_ref });
  assert.equal(sinRefresco.obtenerEstado().recibo.recibo_ref, "rec-prueba-valido");
  assert.equal(sinRefresco.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(
    sinRefresco.obtenerEstado().mensaje_clave,
    "estado_confirmada_actualizacion_pendiente",
  );
});

test("mutex y cancelación impiden doble efecto y dejan resultado indeterminado visible", async () => {
  const expediente = expedienteConAccionSinteticaDisponible();
  const cuadro = validarCuadroContratacionTemporal(crearCuadroContratacionTemporalPresentacion());
  let ejecuciones = 0;
  const fuente = {
    listar: async () => cuadro,
    obtener: async () => expediente,
    ejecutar: async (_comando, { signal }) => {
      ejecuciones += 1;
      return new Promise((resolve, reject) => {
        signal.addEventListener("abort", () => reject(
          new DOMException("cancelada", "AbortError"),
        ), { once: true });
      });
    },
  };
  const presentador = presentadorDe(fuente, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.enviarAnalisis,
  ]);
  await presentador.cargar();
  await presentador.seleccionarExpediente(expediente.expediente_ref);
  presentador.seleccionarTarea("tarea-solicitud");
  const primera = presentador.ejecutarActuacion({
    accionRef: "reenviar_analisis",
  });
  const segunda = presentador.ejecutarActuacion({
    accionRef: "reenviar_analisis",
  });
  presentador.cancelar();
  await Promise.all([primera, segunda]);
  assert.equal(ejecuciones, 1);
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_cancelado");
});

test("cancelar una lectura no fabrica un efecto indeterminado", async () => {
  const expediente = expedienteConAccionSinteticaDisponible();
  const fuente = {
    listar: async ({ signal }) => new Promise((_resolve, reject) => {
      signal.addEventListener("abort", () => reject(
        new DOMException("cancelada", "AbortError"),
      ), { once: true });
    }),
    obtener: async () => expediente,
    ejecutar: async () => {
      throw new Error("no debe ejecutarse");
    },
  };
  const presentador = presentadorDe(fuente, [CAP.consultarCuadro]);
  const carga = presentador.cargar();
  presentador.cancelar();
  await carga;
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, false);
  assert.equal(presentador.obtenerEstado().resultado_indeterminado, false);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_lectura_cancelada",
  );
});

test("un efecto indeterminado bloquea cualquier repetición hasta recuperar estado", async () => {
  const expediente = expedienteConAccionSinteticaDisponible();
  const cuadro = validarCuadroContratacionTemporal(
    crearCuadroContratacionTemporalPresentacion(),
  );
  let ejecuciones = 0;
  const fuente = {
    listar: async () => cuadro,
    obtener: async () => expediente,
    ejecutar: async () => {
      ejecuciones += 1;
      const error = new Error("detalle privado");
      error.resultadoIndeterminado = true;
      error.reintentoPermitido = false;
      throw error;
    },
  };
  const presentador = presentadorDe(fuente, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.enviarAnalisis,
  ]);
  await presentador.cargar();
  await presentador.seleccionarExpediente(expediente.expediente_ref);
  presentador.seleccionarTarea("tarea-solicitud");
  await presentador.ejecutarActuacion({
    accionRef: "reenviar_analisis",
  });
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
  await presentador.ejecutarActuacion({
    accionRef: "reenviar_analisis",
  });
  assert.equal(ejecuciones, 1);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
  await presentador.seleccionarExpediente(expediente.expediente_ref);
  assert.equal(presentador.obtenerEstado().resultado_indeterminado, true);
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
  presentador.cambiarVista("cuadro");
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
});

test("el alta crea un expediente nuevo mínimo sin heredar candidato ni documentos", async () => {
  const fuente = adaptador();
  const catalogos = fuente.obtenerCatalogosAlta();
  const base = crearBorradorAlta();
  const borrador = {
    ...base,
    centro_ref: catalogos.centros[0].referencia,
    contacto_ref: catalogos.centros[0].contactos[0].referencia,
    categoria_ref: catalogos.categorias[0].referencia,
    grupo_subgrupo: catalogos.categorias[0].grupos_subgrupos[0].clave,
    motivo_clave: catalogos.motivos[0].clave,
    detalle: "Necesidad sintética para validar el alta coherente.",
    inicio: "2026-08-15",
    fin: "2027-04-14",
    documentos_adjuntos: [catalogos.documentos[0].referencia],
  };
  const comando = crearComandoAlta(
    borrador,
    catalogos,
    "12345678-1234-4abc-8def-1234567890ab",
  );
  const recibo = await fuente.registrarSolicitud(comando);
  const cuadro = await fuente.listar();
  assert.equal(cuadro.expedientes[0].expediente_ref, recibo.expediente_ref);
  const detalle = await fuente.obtener(recibo.expediente_ref);
  const documentos = await fuente.obtenerDocumentos(recibo.expediente_ref);
  const auditoria = await fuente.obtenerAuditoria(recibo.expediente_ref);
  assert.equal(detalle.tareas[0].estado_clave, "en_curso");
  assert.ok(detalle.tareas.slice(1).every(({ estado_clave }) => estado_clave === "pendiente"));
  assert.doesNotMatch(JSON.stringify(detalle), /CAND-DEMO|fiscalización favorable/i);
  assert.equal(documentos.documentos.length, 0);
  assert.equal(auditoria.actuaciones.length, 1);
});

test("el alta ejecuta un solo efecto y conserva su recibo sin refresco automático", async () => {
  const recibo = Object.freeze({
    expediente_ref: "expediente:ct:real:001",
    numero_visible: "2026/CT-0001",
    version: 1,
    recibo_ref: "recibo:ct:real:001",
    confirmada_en: "2026-09-04T07:55:00Z",
  });
  let altas = 0;
  let refrescos = 0;
  const ejecutar = async () => {
    altas += 1;
    return recibo;
  };
  const presentador = {
    async cargar() {
      refrescos += 1;
      throw new Error("cuadro todavía no compuesto");
    },
  };
  const ejecutarConRefresco = crearEjecutorAltaConRefresco(ejecutar, presentador);

  assert.deepEqual(await ejecutarConRefresco({}, {}), recibo);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(altas, 1);
  assert.equal(refrescos, 0);
});

test("el alta no inicia un refresco pendiente que pueda retirar el análisis", async () => {
  const recibo = Object.freeze({
    expediente_ref: "expediente:ct:real:pendiente",
    numero_visible: "2026/CT-0002",
    version: 1,
    recibo_ref: "recibo:ct:real:pendiente",
    confirmada_en: "2026-09-04T08:05:00Z",
  });
  const refrescoPendiente = new Promise(() => {});
  const eventos = [];
  let montajes = 0;
  let refrescos = 0;
  const ejecutarConRefresco = crearEjecutorAltaConRefresco(
    async () => {
      eventos.push("alta");
      return recibo;
    },
    {
      cargar() {
        refrescos += 1;
        eventos.push("refresco");
        return refrescoPendiente;
      },
    },
    (confirmado) => {
      montajes += 1;
      eventos.push("analisis");
      assert.deepEqual(confirmado, recibo);
    },
  );

  const resultado = await Promise.race([
    ejecutarConRefresco({}, {}),
    new Promise((_, reject) => setImmediate(() => {
      reject(new Error("el alta quedó bloqueada por el refresco"));
    })),
  ]);

  assert.deepEqual(resultado, recibo);
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(eventos, ["alta", "analisis"]);
  assert.equal(montajes, 1);
  assert.equal(refrescos, 0);
});

test("un refresco satisfactorio conserva la vista de alta", async () => {
  const presentador = presentadorDe(adaptador());
  await presentador.cargar();
  presentador.cambiarVista("alta");
  await presentador.cargar();
  assert.equal(presentador.obtenerEstado().vista, "alta");
});

test("HTML escapa contenido, bloquea históricos y expone semántica accesible", () => {
  const entrada = crearExpedienteContratacionTemporalPresentacion();
  entrada.cabecera[0].valor = '<img src=x onerror="alert(1)">';
  const expediente = validarExpedienteContratacionTemporal(entrada);
  const t = crearTraductorExpedientesContratacion();
  const estado = estadoVista(expediente, "tarea-analisis");
  const html = renderizarModuloContratacionTemporal(estado);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/);
  assert.doesNotMatch(html, /<img src=x|style="/);
  assert.match(html, /aria-labelledby="ct-exp-titulo"/);
  assert.match(html, /aria-current="step"/);
  assert.match(html, /Vista histórica o de consulta/);
  assert.match(html, /<select[^>]+disabled/);
  const htmlComponente = renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");
  assert.match(htmlComponente, /<nav class="ct-exp-tareas" aria-label=/);
  assert.match(htmlComponente, /<details class="ct-exp-detalle-tecnico">/);
  assert.doesNotMatch(htmlComponente, /<details class="ct-exp-detalle-tecnico" open/);
  assert.match(htmlComponente, /<summary>Referencias<\/summary>/);
  assert.doesNotMatch(htmlComponente, /Metadatos técnicos/);
});

test("el identificador completo puede envolver y los paneles vacíos no ocultan auditoría", async () => {
  const css = await readFile(new URL("./expedientes.css", import.meta.url), "utf8");
  assert.match(css, /\.ct-exp-cabecera-expediente > div\s*\{\s*min-width: 0;\s*\}/u);
  assert.match(css, /\.ct-exp-cabecera-expediente h3\s*\{\s*overflow-wrap: anywhere;\s*\}/u);
  const expediente = validarExpedienteContratacionTemporal({
    ...crearExpedienteContratacionTemporalPresentacion(),
    numero_visible: "2026/CT-" + "b".repeat(32),
    fases: [], tareas: [],
  });
  const estado = estadoVista(expediente, "");
  const html = renderizarModuloContratacionTemporal(estado);
  // El identificador técnico se abrevia a prefijo y seis caracteres; el completo queda en el título.
  assert.ok(html.includes(`<h3><span title="${expediente.numero_visible}">2026/CT-bbbbbb…</span></h3>`));
  assert.match(html, /ct-exp-cabecera-expediente/u);
  assert.doesNotMatch(html, /class="ct-exp-(?:progreso|tareas|tramitacion)"/u);
  const auditoria = validarAuditoriaContratacionTemporal(
    crearAuditoriaContratacionTemporalPresentacion(),
  );
  const htmlAuditoria = renderizarModuloContratacionTemporal({
    ...estado, vista: "auditoria", auditoria,
  });
  assert.ok(auditoria.actuaciones.length > 0);
  assert.match(htmlAuditoria, /ct-exp-tabla-auditoria/u);
  for (const actuacion of auditoria.actuaciones) {
    assert.ok(htmlAuditoria.includes(actuacion.fecha));
  }
});
