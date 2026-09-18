import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  renderizarCuadro,
  renderizarEstadoCarga,
  renderizarExpediente,
} from "./componentes-expedientes.js";
import {
  crearCuadroContratacionTemporalPresentacion,
  crearExpedienteContratacionTemporalPresentacion,
} from "./datos-presentacion.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import {
  montarModuloContratacionTemporal,
  renderizarModuloContratacionTemporal,
} from "./vista-expedientes.js";

const t = crearTraductorExpedientesContratacion();

test("orden de lectura en el detalle del expediente: cabecera, borradores, raíl, tramitación e historial", () => {
  const expediente = {
    ...crearExpedienteContratacionTemporalPresentacion(),
    version: 7,
    demostracion: false,
    historial: [
      { secuencia: 1, fecha: "2026-07-01", accion: "Alta", fase: "Solicitud", estado: "registrado" },
      { secuencia: 2, fecha: "2026-07-02", accion: "Análisis", fase: "Análisis", estado: "aprobado" },
    ],
  };
  const cuadro = {
    ...crearCuadroContratacionTemporalPresentacion(),
    demostracion: false,
    expedientes: [
      {
        expediente_ref: expediente.expediente_ref,
        version: 7,
        fase_clave: "nombramiento",
        estado_clave: "en_curso",
      },
    ],
  };
  const estado = {
    vista: "expediente",
    carga: "listo",
    ocupado: false,
    actualizacion_pendiente: false,
    resultado_indeterminado: false,
    expediente_ref: expediente.expediente_ref,
    tarea_ref: expediente.tareas[0]?.tarea_ref || null,
    expediente,
    cuadro,
    recibo: null,
  };

  const html = renderizarExpediente(estado, t, "es-ES", "Europe/Madrid", false);

  const posCabecera = html.indexOf('class="ct-exp-cabecera-expediente"');
  const posBorradores = html.indexOf('class="ct-exp-borradores"');
  const posProgreso = html.indexOf('class="ct-exp-progreso"');
  const posTramitacion = html.indexOf('class="ct-exp-tramitacion"');
  const posHistorial = html.indexOf('class="ct-exp-detalle-tecnico ct-exp-historial"');

  assert.ok(posCabecera !== -1, "falta cabecera");
  assert.ok(posBorradores !== -1, "faltan borradores");
  assert.ok(posProgreso !== -1, "falta progreso de fases");
  assert.ok(posTramitacion !== -1, "falta tramitación");
  assert.ok(posHistorial !== -1, "falta historial");

  assert.ok(posCabecera < posBorradores, "la cabecera debe preceder a los borradores");
  assert.ok(posBorradores < posProgreso, "los borradores deben preceder al raíl de fases");
  assert.ok(posProgreso < posTramitacion, "el raíl debe preceder a la tramitación");
  assert.ok(posTramitacion < posHistorial, "la tramitación debe preceder al historial");
});

test("estilos responsivos a 390px garantizan lectura de arriba abajo sin solapamientos ni cortes", async () => {
  const css = await readFile(new URL("./expedientes-responsive.css", import.meta.url), "utf8");

  assert.match(css, /@media\s*\(max-width:\s*620px\)/u, "falta media query móvil");
  assert.match(css, /\.ct-exp-cabecera-expediente\s*\{[^}]*padding:\s*10px\s+12px;/u);
  assert.match(css, /\.ct-exp-cabecera-expediente\s+dt,\s*\.ct-exp-cabecera-expediente\s+dd\s*\{[^}]*overflow-wrap:\s*anywhere;/u);
  assert.match(css, /\.ct-exp-flujo\s+code\s*\{[^}]*overflow-wrap:\s*anywhere;/u);

  assert.match(css, /\.ct-exp-borradores\s+ul\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\);/u);
  assert.match(css, /\.ct-exp-borradores-acciones\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\);/u);
  assert.match(css, /\.ct-exp-borradores-acciones\s+\.boton-secundario\s*\{[^}]*white-space:\s*normal;/u);

  assert.match(css, /\.ct-exp-progreso\s+ol\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\);/u);
  assert.match(css, /\.ct-exp-progreso\s+ol\s*\{[^}]*min-width:\s*0;/u);
  assert.match(css, /\.ct-exp-progreso\s+li\s*\{[^}]*text-align:\s*left;/u);

  assert.match(css, /\.ct-exp-historial\s*\{[^}]*box-sizing:\s*border-box;/u);
  assert.match(css, /\.ct-exp-historial\s+\.tabla-contenedor\s*\{[^}]*overflow-x:\s*auto;/u);
  assert.match(css, /\.ct-exp-acciones-estado\s*\{[^}]*flex-direction:\s*column;/u);
});

test("estado vacío del cuadro: texto en castellano claro y acción posible Reintentar", () => {
  const estadoVacio = {
    vista: "cuadro",
    carga: "vacio",
    cuadro: {
      demostracion: false,
      expedientes: [],
      indicadores: [],
    },
    filtros: { texto: "", estado: "", fase: "" },
  };

  const htmlEstado = renderizarEstadoCarga(estadoVacio, t);
  assert.match(htmlEstado, /Sin resultados/u);
  assert.match(htmlEstado, /Cambie o quite algún filtro para ampliar la búsqueda/u);
  assert.match(htmlEstado, /data-ct-exp-accion="reintentar"/u);
  assert.match(htmlEstado, /Reintentar/u);

  const htmlCuadro = renderizarCuadro(estadoVacio, t);
  assert.match(htmlCuadro, /Sin resultados/u);
  assert.match(htmlCuadro, /data-ct-exp-accion="reintentar"/u);
});

test("estado de error en expediente: texto claro en castellano y acciones Reintentar y Volver al cuadro", () => {
  const estadoError = {
    vista: "expediente",
    carga: "error",
    expediente_ref: "expediente:ct:001",
    expediente: null,
    cuadro: null,
  };

  const htmlEstado = renderizarEstadoCarga(estadoError, t);
  assert.match(htmlEstado, /La superficie no está disponible/u);
  assert.match(htmlEstado, /No se pudo cargar el cuadro/u);
  assert.match(htmlEstado, /data-ct-exp-accion="reintentar"/u);
  assert.match(htmlEstado, /data-ct-exp-vista="cuadro"/u);
  assert.match(htmlEstado, /Volver al cuadro/u);

  const htmlExpediente = renderizarExpediente(estadoError, t, "es-ES", "Europe/Madrid");
  assert.match(htmlExpediente, /La superficie no está disponible/u);
  assert.match(htmlExpediente, /No se pudo cargar el expediente/u);
  assert.match(htmlExpediente, /data-ct-exp-accion="reintentar"/u);
  assert.match(htmlExpediente, /data-ct-exp-vista="cuadro"/u);
  assert.match(htmlExpediente, /Volver al cuadro/u);
});

test("formulario sin catálogo: renderizado estático y montaje interactivo con Reintentar y Volver al cuadro", async () => {
  const estadoAlta = {
    vista: "alta",
    carga: "listo",
    expediente: null,
    cuadro: null,
  };

  const htmlSinCatalogo = renderizarModuloContratacionTemporal(estadoAlta, {
    mensajes: {},
    altaDisponible: false,
    catalogoDisponible: false,
  });
  assert.match(htmlSinCatalogo, /Catálogo no disponible/u);
  assert.match(htmlSinCatalogo, /No se ha podido cargar el catálogo/u);
  assert.match(htmlSinCatalogo, /data-ct-exp-accion="reintentar"/u);
  assert.match(htmlSinCatalogo, /data-ct-exp-vista="cuadro"/u);

  const eventos = new Map();
  const contenedorAlta = {
    innerHTML: "",
    querySelector: () => null,
    addEventListener: () => {},
    removeEventListener: () => {},
    contains: () => true,
  };
  const raiz = {
    innerHTML: '<div class="ct-exp-contenido"><div data-ct-exp-alta></div></div>',
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: (sel) => (sel === "[data-ct-exp-alta]" ? contenedorAlta : null),
    contains: () => true,
    ownerDocument: { createElement: () => ({ setAttribute() {}, append() {} }) },
  };

  let vistaSeleccionada = null;
  const presentador = {
    obtenerEstado: () => ({
      vista: "alta",
      carga: "listo",
      ocupado: false,
      actualizacion_pendiente: false,
      resultado_indeterminado: false,
      cuadro: null,
      expediente: null,
      expediente_ref: null,
      tarea_ref: null,
      mensaje: "",
      mensaje_clave: "",
      tipo_mensaje: "info",
    }),
    cargar: async () => {},
    cambiarVista: async (v) => { vistaSeleccionada = v; },
    cancelar: () => {},
  };

  const montaje = await montarModuloContratacionTemporal({
    raiz,
    presentador,
    alta: { catalogos: null, ejecutor: () => {} },
  });

  assert.match(contenedorAlta.innerHTML, /Catálogo no disponible/u);
  assert.match(contenedorAlta.innerHTML, /data-ct-exp-accion="reintentar"/u);
  assert.match(contenedorAlta.innerHTML, /data-ct-exp-vista="cuadro"/u);

  const clickHandler = eventos.get("click");
  assert.equal(typeof clickHandler, "function");
  await clickHandler({
    target: {
      closest: (sel) => (sel === "[data-ct-exp-vista]" ? { dataset: { ctExpVista: "cuadro" } } : null),
    },
    preventDefault: () => {},
  });
  assert.equal(vistaSeleccionada, "cuadro", "el botón Volver al cuadro navega al cuadro");

  montaje.desmontar();
});

test("todos los estados de error y vacío carecen de tokens técnicos y claves sin traducir", () => {
  const estados = [
    renderizarEstadoCarga({ vista: "cuadro", carga: "vacio" }, t),
    renderizarEstadoCarga({ vista: "expediente", carga: "error", expediente_ref: "ct:1" }, t),
    renderizarExpediente({ vista: "expediente", carga: "error", expediente: null }, t, "es-ES", "Europe/Madrid"),
    renderizarModuloContratacionTemporal({ vista: "alta", carga: "listo", expediente: null, cuadro: null }, {
      altaDisponible: false, catalogoDisponible: false,
    }),
  ];

  for (const html of estados) {
    assert.doesNotMatch(html, /undefined|null|\[object Object\]/u);
    assert.doesNotMatch(html, /\{[a-zA-Z0-9_]+\}/u);
    assert.match(html, /class="boton-secundario"/u);
  }
});

test("un expediente con incidencia explica su origen y ofrece atajos", () => {
  const expediente = {
    esquema: "vec.contratacion_temporal.expediente.v1", demostracion: false,
    expediente_ref: "expediente:ct:incidencia", numero_visible: "2026/CT-000010", version: 7,
    flujo_ref: "flujo:ct", flujo_version: 1, flujo_huella: "a".repeat(64),
    cabecera: [], tareas: [],
    fases: [
      { orden: 1, clave: "solicitud", etiqueta: "Solicitud", estado_clave: "completado" },
      { orden: 4, clave: "fiscalizacion", etiqueta: "Fiscalización", estado_clave: "incidencia" },
      { orden: 5, clave: "obtencion_candidato", etiqueta: "Obtención del candidato", estado_clave: "pendiente" },
    ],
    historial: [
      { secuencia: 5, fecha: "16 sept 2026", fase: "Informe jurídico", accion: "Informe jurídico generado", estado_clave: "en_curso", estado: "En tramitación", accion_clave: "contratacion_temporal.informe_juridico.generar", version_expediente: 5 },
      { secuencia: 6, fecha: "17 sept 2026", fase: "Subsanación por la unidad", accion: "Fiscalización registrada", estado_clave: "incidencia", estado: "Con incidencia", accion_clave: "contratacion_temporal.fiscalizacion.registrar", version_expediente: 6 },
      { secuencia: 7, fecha: "17 sept 2026", fase: "Subsanación por la unidad", accion: "Subsanación de reparos registrada", estado_clave: "incidencia", estado: "Con incidencia", accion_clave: "contratacion_temporal.subsanacion_reparos.registrar", version_expediente: 7 },
    ],
  };
  expediente.fiscalizacion = {
    resultado_clave: "desfavorable", resultado: "Desfavorable", registrada_en: "17 sept 2026",
    reparos: [{ clave: "observaciones_fiscalizacion", texto: "Falta justificar el coste." }],
    subsanacion: { registrada_en: "17 sept 2026", texto: "Justificación aportada." },
  };
  const html = renderizarExpediente({ vista: "expediente", carga: "listo", expediente }, t, "es-ES", "Europe/Madrid");
  assert.match(html, /class="ct-exp-incidencia" role="alert"/u);
  assert.match(html, /Reparos de Intervención:/u);
  assert.match(html, /<blockquote class="ct-exp-incidencia-reparo">Falta justificar el coste\.<\/blockquote>/u);
  assert.match(html, /Subsanación de la unidad:<\/strong> Justificación aportada\./u);
  assert.match(html, /Incidencia en «Fiscalización»/u);
  assert.match(html, /Origen: Fiscalización registrada, 17 sept 2026 \(actuación 6\)/u);
  assert.match(html, /registró la subsanación \(17 sept 2026\)/u);
  assert.match(html, /data-ct-exp-accion="abrir-historial"/u);
  assert.match(html, /data-ct-exp-vista="documentos"/u);
  assert.match(html, /data-ct-exp-vista="auditoria"/u);
  assert.match(html, /<li class="ct-fase-completado"/u);
  assert.match(html, /<li class="ct-fase-incidencia"/u);
  assert.match(html, /<li class="ct-fase-pendiente"/u);
});
