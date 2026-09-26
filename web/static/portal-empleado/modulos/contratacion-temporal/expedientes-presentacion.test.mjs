import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

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

  // El documento se nombra; su referencia interna no se muestra.
  assert.match(html, /<th scope="row">Ficha &lt;GINPIX&gt;<\/th>/u);
  assert.doesNotMatch(html, /documento:&lt;interno&gt;<\/code>/u);
  assert.match(html, /Preparado<\/td><td>Sin firma<\/td>/u);
  assert.match(html, /Descarga pendiente de conectar/u);
  assert.doesNotMatch(html, /<GINPIX>|data-ct-ficha-ginpix-descargar/u);
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
  // Sin desplegable de referencias internas en la cabecera.
  assert.doesNotMatch(htmlComponente, /ct-exp-detalle-tecnico/);
  assert.doesNotMatch(htmlComponente, /<summary>Referencias<\/summary>/);
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
  // El identificador técnico anterior a la numeración no se muestra: figura sin numerar.
  assert.ok(html.includes("<h3>Sin numerar</h3>"));
  assert.ok(!html.includes(expediente.numero_visible));
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
