/**
 * Composición de módulos del Portal del Empleado.
 *
 * El shell conserva la navegación, el tema y el router comunes. Este archivo
 * solo registra adaptadores disponibles y monta su vista; no contiene reglas
 * de negocio. Los adaptadores sintéticos se importan únicamente cuando el
 * arranque declara de forma explícita el modo presentación.
 */
import {
  compartirContextoActor,
  crearProveedorContextoActorFijo,
} from "./identidad/contexto-actor.js";
import {
  cargarCatalogoModulosInterno,
  renderizarNavegacionModulos,
} from "./portal-catalogo-modulos.js?v=20260906-acceso-certificado-v1";
import { traducirPortal } from "./portal-i18n.js?v=20260920-personal-catalogo-v1";
import { calcularMetricasCuadro, tramitesParaInicio } from "./portal-inicio.js";
import {
  componerCronosVisible,
  componerDietasInternas,
  componerDietasVisible,
  componerPersonalVisible,
} from "./portal-composicion-empleado.js";

const CLAVE_CONTRATACION_TEMPORAL = "contratacion_temporal";
const CLAVE_PERSONAL = "personal";
const CLAVES_CARGA_MODULAR = Object.freeze([
  CLAVE_CONTRATACION_TEMPORAL,
  CLAVE_PERSONAL,
  "cronos",
  "dietas",
]);
// Estos recorridos son exclusivamente visuales. Se cargan sólo desde el
// arranque de presentación y no son una concesión, un manifiesto productivo ni
// una sustitución de sus casos de uso.
const CLAVES_PRESENTACION_VISUAL = Object.freeze([
  "nominas",
  "solicitudes",
  "meritos",
  "comunicaciones",
  "documentos",
  "aprobaciones",
  "auditoria",
  "administracion",
]);
const VISTA_POR_CLAVE_PRESENTACION = Object.freeze(Object.fromEntries(
  CLAVES_PRESENTACION_VISUAL.map((clave) => [clave, `${clave}-empleado`]),
));
const CLAVE_POR_VISTA_PRESENTACION = Object.freeze(Object.fromEntries(
  Object.entries(VISTA_POR_CLAVE_PRESENTACION).map(([clave, vista]) => [vista, clave]),
));
export const VISTAS_PRESENTACION_VISUALES = Object.freeze(new Set(Object.keys(CLAVE_POR_VISTA_PRESENTACION)));
const CLAVES_CARGA_PRESENTACION = Object.freeze([
  ...CLAVES_CARGA_MODULAR,
  ...CLAVES_PRESENTACION_VISUAL,
]);
const LIMITE_CARGA_MODULAR_MS = 2_000;

const CARGADORES_PRESENTACION_PREDETERMINADOS = Object.freeze({
  base: async () => {
    const [identidad, catalogo] = await Promise.all([
      import("./identidad/presentacion.js"),
      import("./portal-catalogo-presentacion.js"),
    ]);
    return Object.freeze({ identidad, catalogo });
  },
  contratacion_temporal: async () => {
    const [contrato, presentador, vista, adaptador] = await Promise.all([
      import("./modulos/contratacion-temporal/contrato.js"),
      import("./modulos/contratacion-temporal/presentador-expedientes.js"),
      import("./modulos/contratacion-temporal/vista-expedientes.js"),
      import("./modulos/contratacion-temporal/adaptador-presentacion.js"),
    ]);
    return Object.freeze({ contrato, presentador, vista, adaptador });
  },
  cronos: async () => {
    const [contrato, recorridos] = await Promise.all([
      import("./modulos/cronos/contrato.js"),
      import("./modulos/cronos/vista-recorridos.js"),
    ]);
    return Object.freeze({ contrato, recorridos });
  },
  dietas: async () => {
    const [contrato, vista, mapa, calculador, recorridos] = await Promise.all([
      import("./modulos/dietas/contrato.js"),
      import("./modulos/dietas/vista-itinerario.js?v=20260920-dietas-ayuda-contextual-v3"),
      import("./modulos/dietas/mapa-ruta.js"),
      import("./modulos/dietas/calculador-rutas-presentacion-osrm.js"),
      import("./modulos/dietas/vista-recorridos.js?v=20260920-dietas-ayuda-contextual-v3"),
    ]);
    return Object.freeze({ contrato, vista, mapa, calculador, recorridos });
  },
  personal: async () => {
    const [contrato, clienteCategorias, vistaCategorias, clienteRPT, vistaRPT, clienteEstructura, vistaEstructura, ficha] = await Promise.all([
      import("./modulos/personal/contrato.js?v=20260920-personal-catalogo-v1"),
      import("./modulos/personal/cliente-http-categorias.js?v=20260920-personal-catalogo-v1"),
      import("./modulos/personal/vista.js?v=20260920-personal-catalogo-v1"),
      import("./modulos/personal/cliente-http-rpt-publica.js?v=20260920-personal-rpt-publica-v3"),
      import("./modulos/personal/vista-rpt-publica.js?v=20260920-personal-rpt-publica-v3"),
      import("./modulos/personal/cliente-http-estructura-organizativa-publica.js?v=20260920-personal-estructura-v1"),
      import("./modulos/personal/vista-estructura-organizativa-publica.js?v=20260920-personal-estructura-v1"),
      import("./modulos/personal/vista-ficha-integral.js"),
    ]);
    return Object.freeze({ contrato, clienteCategorias, vistaCategorias, clienteRPT, vistaRPT, clienteEstructura, vistaEstructura, ficha });
  },
  nominas: async () => Object.freeze({
    vista: await import("./modulos/nominas/vista.js"),
  }),
  solicitudes: async () => Object.freeze({
    vista: await import("./modulos/solicitudes/vista.js"),
  }),
  meritos: async () => Object.freeze({
    vista: await import("./modulos/meritos/vista.js"),
  }),
  comunicaciones: async () => Object.freeze({
    vista: await import("./modulos/comunicaciones/vista.js"),
  }),
  documentos: async () => Object.freeze({
    vista: await import("./modulos/documentos/vista.js"),
  }),
  aprobaciones: async () => Object.freeze({
    vista: await import("./modulos/aprobaciones/vista.js"),
  }),
  auditoria: async () => Object.freeze({
    vista: await import("./modulos/auditoria/vista.js"),
  }),
  administracion: async () => Object.freeze({
    vista: await import("./modulos/administracion/vista.js"),
  }),
});

const CARGADORES_INTERNOS_PREDETERMINADOS = Object.freeze({
  contratacion_temporal: async () => {
    const [contrato, cliente, presentador, vista, adaptador] = await Promise.all([
      import("./modulos/contratacion-temporal/contrato.js"),
      import("./modulos/contratacion-temporal/cliente-http.js"),
      import("./modulos/contratacion-temporal/presentador-expedientes.js"),
      import("./modulos/contratacion-temporal/vista-expedientes.js"),
      import("./modulos/contratacion-temporal/adaptador-http-expedientes.js"),
    ]);
    return Object.freeze({ contrato, cliente, presentador, vista, adaptador });
  },
  personal: async () => {
    const [contrato, cliente, vista] = await Promise.all([
      import("./modulos/personal/contrato.js?v=20260920-personal-catalogo-v1"),
      import("./modulos/personal/cliente-http-categorias.js?v=20260920-personal-catalogo-v1"),
      import("./modulos/personal/vista.js?v=20260920-personal-catalogo-v1"),
    ]);
    return Object.freeze({ contrato, cliente, vista });
  },
  dietas: async () => {
    const [contrato, vista, mapa, calculador, recorridos, clienteBorradores] = await Promise.all([
      import("./modulos/dietas/contrato.js"),
      import("./modulos/dietas/vista-itinerario.js?v=20260920-dietas-ayuda-contextual-v3"),
      import("./modulos/dietas/mapa-ruta.js"),
      import("./modulos/dietas/calculador-rutas-http.js"),
      import("./modulos/dietas/vista-recorridos.js?v=20260920-dietas-ayuda-contextual-v3"),
      import("./modulos/dietas/cliente-borradores-http.js"),
    ]);
    return Object.freeze({ contrato, vista, mapa, calculador, recorridos, clienteBorradores });
  },
});

function cargarModuloConLimite(cargar, clave, limiteMs, temporizadores) {
  if (typeof temporizadores?.setTimeout !== "function"
    || typeof temporizadores?.clearTimeout !== "function") {
    return Promise.reject(new TypeError("temporizadores modulares no disponibles"));
  }
  return new Promise((resolver, rechazar) => {
    let terminada = false;
    const finalizar = (continuacion, valor) => {
      if (terminada) return;
      terminada = true;
      temporizadores.clearTimeout(temporizador);
      continuacion(valor);
    };
    const temporizador = temporizadores.setTimeout(
      () => finalizar(rechazar, new Error(`tiempo agotado al cargar ${clave}`)),
      limiteMs,
    );
    Promise.resolve()
      .then(cargar)
      .then(
        (recursos) => finalizar(resolver, recursos),
        () => finalizar(rechazar, new Error(`no se pudo cargar ${clave}`)),
      );
  });
}
function consultarConLimite(consultar, controlador, limiteMs, temporizadores) {
  return new Promise((resolver, rechazar) => {
    let terminada = false;
    const finalizar = (continuacion, valor) => {
      if (terminada) return;
      terminada = true;
      temporizadores.clearTimeout(temporizador);
      continuacion(valor);
    };
    const temporizador = temporizadores.setTimeout(() => {
      controlador.abort();
      finalizar(
        rechazar,
        new Error("tiempo agotado al consultar contratación temporal"),
      );
    }, limiteMs);
    Promise.resolve()
      .then(() => consultar({ signal: controlador.signal }))
      .then(
        (resultado) => finalizar(resolver, resultado),
        () => finalizar(
          rechazar,
          new Error("no se pudo consultar contratación temporal"),
        ),
      );
  });
}

export async function resolverCargasModularesPresentacion(cargadores, {
  claves = CLAVES_CARGA_PRESENTACION,
  limiteMs = LIMITE_CARGA_MODULAR_MS,
  temporizadores = globalThis,
} = {}) {
  if (!Array.isArray(claves)
    || claves.some((clave) => !CLAVES_CARGA_PRESENTACION.includes(clave))
    || new Set(claves).size !== claves.length
    || !Number.isSafeInteger(limiteMs) || limiteMs < 1 || limiteMs > 10_000) {
    throw new TypeError("configuración de carga modular no válida");
  }
  const clavesSolicitadas = new Set(claves);
  const resultados = await Promise.allSettled(CLAVES_CARGA_PRESENTACION.map((clave) => {
    if (!clavesSolicitadas.has(clave)) return Promise.resolve(undefined);
    const cargar = cargadores?.[clave];
    if (typeof cargar !== "function") {
      return Promise.reject(new TypeError(`cargador modular ausente: ${clave}`));
    }
    return cargarModuloConLimite(cargar, clave, limiteMs, temporizadores);
  }));
  return Object.freeze(Object.fromEntries(CLAVES_CARGA_PRESENTACION.map((clave, indice) => {
    if (!clavesSolicitadas.has(clave)) {
      return [clave, Object.freeze({ disponible: false, estado: "denegado" })];
    }
    const resultado = resultados[indice];
    return [clave, resultado.status === "fulfilled"
      ? Object.freeze({
        disponible: true,
        estado: "disponible",
        recursos: resultado.value,
      })
      : Object.freeze({ disponible: false, estado: "no_disponible" })];
  })));
}

function capacidadesDietas(contrato) {
  return Object.freeze([
    contrato.CAPACIDAD_CONSULTAR_GASTO,
    contrato.CAPACIDAD_GESTIONAR_GASTO,
    contrato.CAPACIDAD_CONSULTAR_RUTA,
    contrato.CAPACIDAD_GESTIONAR_RUTA,
  ]);
}

function componerModuloAislado(contexto, carga, componer) {
  if (!contexto || carga?.disponible !== true) return undefined;
  try {
    return componer(carga.recursos);
  } catch {
    return undefined;
  }
}

function componerVistaPresentacionAislada(carga, exportacion) {
  if (carga?.disponible !== true || typeof carga.recursos?.vista?.[exportacion] !== "function") {
    return undefined;
  }
  return Object.freeze({ montar: carga.recursos.vista[exportacion] });
}

export const VISTAS_MODULOS_PERSONALES = Object.freeze(new Set(["cronos", "dietas", "personal"]));
export const VISTAS_MODULOS_CONECTADOS = Object.freeze(new Set([
  "contratacion-temporal", ...VISTAS_MODULOS_PERSONALES,
]));

export function moduloDeVistaPortal(vista) {
  if (vista === "portal") return "portal";
  if (vista === "contratacion-temporal") return "contratacion_temporal";
  if (vista === "personal") return CLAVE_PERSONAL;
  if (VISTAS_MODULOS_PERSONALES.has(vista)) return vista;
  return "bolsa";
}

export function rutaDeVistaPortal(vista) {
  if (vista === "portal") return "#portal";
  if (vista === "contratacion-temporal") return "#contratacion-temporal";
  if (VISTAS_MODULOS_PERSONALES.has(vista)) return `#${vista}`;
  return `#bolsa/${vista}`;
}

export function crearCoordinadorModulosPortal({
  escaparHTML,
  anunciar = () => {},
  confirmarOperacion = () => false,
  entorno = globalThis,
  traducir = traducirPortal,
  cargarCatalogoInterno = cargarCatalogoModulosInterno,
  cargadoresPresentacion = CARGADORES_PRESENTACION_PREDETERMINADOS,
  cargadoresInternos = CARGADORES_INTERNOS_PREDETERMINADOS,
  limiteCargaModularMs = LIMITE_CARGA_MODULAR_MS,
  temporizadores = globalThis,
} = {}) {
  if (typeof escaparHTML !== "function" || typeof anunciar !== "function"
    || typeof confirmarOperacion !== "function" || typeof traducir !== "function"
    || typeof cargarCatalogoInterno !== "function"
    || typeof cargadoresPresentacion?.base !== "function"
    || typeof cargadoresInternos?.contratacion_temporal !== "function"
    || !Number.isSafeInteger(limiteCargaModularMs)
    || limiteCargaModularMs < 1 || limiteCargaModularMs > 10_000) {
    throw new TypeError("dependencias del coordinador de módulos no válidas");
  }

  let catalogo = Object.freeze([]);
  let composicion = null;
  let presentacionActiva = false;
  let desmontarVista = null;
  let secuenciaMontaje = 0;
  let secuenciaCarga = 0;
  let controladorCargaInterna = null;

  function cancelarCargaInterna() {
    if (controladorCargaInterna === null) return;
    secuenciaCarga += 1;
    controladorCargaInterna.abort();
    controladorCargaInterna = null;
  }

  function desmontarVistaActual() {
    secuenciaMontaje += 1;
    if (typeof desmontarVista === "function") desmontarVista();
    desmontarVista = null;
    cancelarCargaInterna();
  }

  async function cargarPresentacion(sesionBolsa) {
    desmontarVistaActual();
    const carga = ++secuenciaCarga;
    presentacionActiva = true;
    composicion = null;
    catalogo = Object.freeze([]);
    const base = await cargadoresPresentacion.base();
    if (carga !== secuenciaCarga) throw new Error("carga de presentación sustituida");
    const contexto = base.identidad.crearContextoActorPresentacionDesdeSesion(sesionBolsa);
    const contextos = compartirContextoActor(
      crearProveedorContextoActorFijo(contexto), contexto.ambito.modulos,
    );
    // Personal usa la misma concesión positiva de ámbito que los demás
    // módulos: nunca se deduce de la etiqueta de rol ni de un menú.
    const personalPresentacionPermitido = contextos.personal !== undefined;
    const cargas = await resolverCargasModularesPresentacion(
      cargadoresPresentacion,
      {
        // Las cuatro composiciones existentes siguen sometidas al contexto
        // sintético. Los recorridos nuevos son una lámina de presentación:
        // no reciben ni derivan ContextoActor ni permisos del menú.
        claves: [
          ...CLAVES_CARGA_MODULAR.filter((clave) => contextos[clave] !== undefined),
          ...CLAVES_PRESENTACION_VISUAL,
        ],
        limiteMs: limiteCargaModularMs,
        temporizadores,
      },
    );
    if (carga !== secuenciaCarga) throw new Error("carga de presentación sustituida");
    const cronos = componerModuloAislado(contextos.cronos, cargas.cronos,
      (recursos) => componerCronosVisible(recursos, contextos.cronos, entorno));
    const dietas = componerModuloAislado(contextos.dietas, cargas.dietas,
      (recursos) => componerDietasVisible(recursos, contextos.dietas, capacidadesDietas(recursos.contrato), entorno));
    const personal = componerModuloAislado(contextos.personal, cargas.personal,
      (recursos) => componerPersonalVisible(recursos, entorno));
    const contratacionTemporal = componerModuloAislado(
      contextos.contratacion_temporal,
      cargas.contratacion_temporal,
      (recursos) => {
        const fuente = recursos.adaptador.crearAdaptadorContratacionTemporalPresentacion({
          contextoActor: contextos.contratacion_temporal,
        });
        let metricasCache = null;
        return Object.freeze({
          crearPresentador: () => recursos.presentador
            .crearPresentadorExpedientesContratacionTemporal({
              fuente, capacidades: fuente.capacidades,
            }),
          alta: Object.freeze({
            catalogos: fuente.obtenerCatalogosAlta(),
            capacidad: recursos.contrato.CAPACIDAD_CREAR_SOLICITUD,
            ejecutor: fuente.registrarSolicitud,
          }),
          // La presentación no compone análisis, fiscalización ni subsanación: el montaje
          // los distingue de la composición interna por ser nulos, no ausentes.
          analisis: null,
          fiscalizacion: null,
          subsanacion: null,
          cargarMetricas: async () => {
            try {
              const listado = await fuente.listar({
                filtros: { texto: "", estado: "", fase: "" },
                cursor: "",
                numeroPagina: 1,
              });
              metricasCache = calcularMetricasCuadro(listado);
            } catch {}
          },
          obtenerMetricas: () => metricasCache,
          montar: recursos.vista.montarModuloContratacionTemporal,
        });
      },
    );
    if (contratacionTemporal && typeof contratacionTemporal.cargarMetricas === "function") {
      await contratacionTemporal.cargarMetricas();
    }
    const vistasPresentacion = Object.freeze({
      nominas: componerVistaPresentacionAislada(cargas.nominas, "montarVistaNominas"),
      solicitudes: componerVistaPresentacionAislada(cargas.solicitudes, "montarVistaSolicitudes"),
      meritos: componerVistaPresentacionAislada(cargas.meritos, "montarVistaMeritos"),
      comunicaciones: componerVistaPresentacionAislada(cargas.comunicaciones, "montarVistaComunicaciones"),
      documentos: componerVistaPresentacionAislada(cargas.documentos, "montarVistaDocumentos"),
      aprobaciones: componerVistaPresentacionAislada(cargas.aprobaciones, "montarVistaAprobaciones"),
      auditoria: componerVistaPresentacionAislada(cargas.auditoria, "montarVistaAuditoria"),
      administracion: componerVistaPresentacionAislada(cargas.administracion, "montarVistaAdministracion"),
    });
    if (carga !== secuenciaCarga) throw new Error("carga de presentación sustituida");
    catalogo = base.catalogo.obtenerCatalogoModulosPresentacion();
    composicion = Object.freeze({
      contextos,
      contratacionTemporal,
      cronos,
      dietas,
      personal,
      vistasPresentacion,
      estadosModulos: Object.freeze({
        contratacion_temporal: contextos.contratacion_temporal === undefined
          ? "denegado"
          : (contratacionTemporal === undefined ? "no_disponible" : "disponible"),
        cronos: contextos.cronos === undefined
          ? "denegado"
          : (cronos === undefined ? "no_disponible" : "disponible"),
        dietas: contextos.dietas === undefined
          ? "denegado"
          : (dietas === undefined ? "no_disponible" : "disponible"),
        personal: contextos.personal === undefined
          ? "denegado"
          : (personal === undefined ? "no_disponible" : "disponible"),
        ...Object.fromEntries(CLAVES_PRESENTACION_VISUAL.map((clave) => [
          clave,
          vistasPresentacion[clave] === undefined ? "no_disponible" : "presentacion",
        ])),
      }),
    });
    return contextos.bolsa || null;
  }

  async function cargarInterno() {
    desmontarVistaActual();
    const carga = ++secuenciaCarga;
    presentacionActiva = false;
    composicion = null;
    catalogo = Object.freeze([]);
    const catalogoInterno = await cargarCatalogoInterno();
    if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
    catalogo = catalogoInterno;
    let contratacionTemporal;
    let personal;
    let dietas;
    if (catalogo.some(({ clave }) => clave === CLAVE_CONTRATACION_TEMPORAL)) {
      try {
        const recursos = await cargarModuloConLimite(
          cargadoresInternos.contratacion_temporal,
          CLAVE_CONTRATACION_TEMPORAL,
          limiteCargaModularMs,
          temporizadores,
        );
        if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
        const cliente = recursos.cliente.crearClienteHTTPContratacionTemporal({
          fetchImpl: typeof entorno.fetch === "function"
            ? entorno.fetch.bind(entorno) : undefined,
          HeadersImpl: entorno.Headers,
        });
        let alta = null;
        const fuente = recursos.adaptador
          .crearAdaptadorHTTPExpedientesContratacionTemporal({
            cliente, obtenerCatalogos: () => alta?.catalogos ?? null,
          });
        const controladorConsulta = new AbortController();
        controladorCargaInterna = controladorConsulta;
        let cuadroDisponible = false;
        let listadoCuadro = null;
        try {
          listadoCuadro = await consultarConLimite(
            (opciones) => fuente.listar(opciones),
            controladorConsulta,
            limiteCargaModularMs,
            temporizadores,
          );
          cuadroDisponible = true;
        } catch {
          if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
        } finally {
          if (controladorCargaInterna === controladorConsulta) controladorCargaInterna = null;
        }
        if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
        const controladorCatalogos = new AbortController();
        controladorCargaInterna = controladorCatalogos;
        try {
          const catalogos = recursos.contrato.validarCatalogosAlta(
            await consultarConLimite(
              (opciones) => cliente.obtenerCatalogosAlta(opciones),
              controladorCatalogos,
              limiteCargaModularMs,
              temporizadores,
            ),
          );
          if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
          alta = Object.freeze({
            catalogos,
            capacidad: recursos.contrato.CAPACIDAD_CREAR_SOLICITUD,
            ejecutor: cliente.registrarSolicitud,
          });
        } catch {
          if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
        } finally {
          if (controladorCargaInterna === controladorCatalogos) {
            controladorCargaInterna = null;
          }
        }
        let analisis = null;
        let subsanacion = null;
        const controladorAnalisis = new AbortController();
        controladorCargaInterna = controladorAnalisis;
        try {
          const configuracionAnalisis = await consultarConLimite(
            (opciones) => cliente.obtenerConfiguracionAnalisis(opciones),
            controladorAnalisis,
            limiteCargaModularMs,
            temporizadores,
          );
          if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
          if (configuracionAnalisis.subsanacion_disponible === true
            && typeof cliente.registrarSubsanacionReparos === "function") {
            subsanacion = Object.freeze({ disponible: true, cliente });
          }
          analisis = Object.freeze({
            cliente,
            catalogos: Object.freeze({
              modalidades: configuracionAnalisis.modalidades,
              categorias: configuracionAnalisis.categorias,
              causas: configuracionAnalisis.causas,
              entradas_rc: configuracionAnalisis.entradas_rc,
              motivos_rectificacion: configuracionAnalisis.motivos_rectificacion,
            }),
            contexto: Object.freeze({
              operacion: "registrar",
              artefacto_ref: configuracionAnalisis.artefacto_ref,
            }),
            analisisInicial: null,
            ...(configuracionAnalisis.motivos_rectificacion.length > 0 ? {
              rectificacion: Object.freeze({
                operacion: "rectificar",
                artefacto_ref: configuracionAnalisis.artefacto_ref,
                analisisInicial: null,
              }),
            } : {}),
          });
        } catch {
          if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
        } finally {
          if (controladorCargaInterna === controladorAnalisis) {
            controladorCargaInterna = null;
          }
        }
        const fiscalizacion = alta === null && analisis === null
          && typeof cliente.registrarResultadoFiscalizacion === "function"
          && typeof recursos.vista.montarModuloFiscalizacionContratacionTemporal === "function"
          ? Object.freeze({ cliente }) : null;
        if (!cuadroDisponible && alta === null && fiscalizacion === null) {
          throw new Error("contratación temporal no disponible");
        }
        contratacionTemporal = Object.freeze({
          crearPresentador: () => recursos.presentador
            .crearPresentadorExpedientesContratacionTemporal({
              fuente, capacidades: fuente.capacidades,
              altaDisponible: alta !== null,
            }),
          alta,
          analisis,
          fiscalizacion,
          subsanacion,
          continuidad: fiscalizacion === null ? Object.freeze({ cliente }) : null,
          obtenerMetricas: () => (listadoCuadro ? calcularMetricasCuadro(listadoCuadro) : null),
          // El listado inicial se carga antes que los catálogos: los nombres de
          // centro y categoría se resuelven al pedirlo, con lo que haya llegado.
          obtenerTramitesInicio: () => {
            if (!listadoCuadro) return null;
            const etiqueta = (lista, referencia) => (Array.isArray(lista)
              ? lista.find((opcion) => opcion.referencia === referencia)?.etiqueta : undefined) ?? referencia;
            return tramitesParaInicio(listadoCuadro).map((e) => ({
              ...e,
              centro: etiqueta(alta?.catalogos?.centros, e.centro),
              categoria: etiqueta(alta?.catalogos?.categorias, e.categoria),
            }));
          },
          montar: recursos.vista.montarModuloContratacionTemporal,
          montarFiscalizacion: recursos.vista.montarModuloFiscalizacionContratacionTemporal,
        });
      } catch {
        contratacionTemporal = undefined;
      }
    }
    if (catalogo.some(({ clave }) => clave === CLAVE_PERSONAL)) {
      try {
        const cargarPersonal = cargadoresInternos.personal
          || CARGADORES_INTERNOS_PREDETERMINADOS.personal;
        const recursos = await cargarModuloConLimite(
          cargarPersonal, CLAVE_PERSONAL, limiteCargaModularMs, temporizadores,
        );
        if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
        if (typeof recursos?.cliente?.crearClienteHTTPCategoriasPersonal !== "function"
          || typeof recursos?.vista?.montarModuloPersonal !== "function"
          || recursos?.contrato?.CAPACIDAD_CONSULTAR_PUESTO !== "personal.puesto.read") {
          throw new TypeError("vista de Personal no disponible");
        }
        personal = Object.freeze({
          cliente: recursos.cliente.crearClienteHTTPCategoriasPersonal({
            fetchImpl: typeof entorno.fetch === "function" ? entorno.fetch.bind(entorno) : undefined,
          }),
          montar: recursos.vista.montarModuloPersonal,
        });
      } catch {
        personal = undefined;
      }
    }
    if (catalogo.some(({ clave }) => clave === "dietas")) {
      try {
        const cargarDietas = cargadoresInternos.dietas
          || CARGADORES_INTERNOS_PREDETERMINADOS.dietas;
        const recursos = await cargarModuloConLimite(
          cargarDietas, "dietas", limiteCargaModularMs, temporizadores,
        );
        if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
        // La composición interna sólo consume el cliente de borradores. Los
        // demás recursos se cargan como contrato del módulo, pero el cálculo
        // necesita una identidad explícita que este coordinador no posee.
        dietas = componerDietasInternas(recursos, entorno);
        if (dietas === undefined) throw new TypeError("vista de Dietas no disponible");
      } catch {
        dietas = undefined;
      }
    }
    if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
    composicion = Object.freeze({
      contextos: Object.freeze({}),
      contratacionTemporal,
      dietas,
      personal,
      vistasPresentacion: Object.freeze({}),
      estadosModulos: Object.freeze({
        contratacion_temporal: contratacionTemporal === undefined
          ? "no_disponible" : "disponible",
        cronos: "no_disponible",
        dietas: dietas === undefined ? "no_disponible" : "disponible",
        personal: personal === undefined ? "no_disponible" : "disponible",
      }),
    });
  }

  function obtenerCatalogo() {
    return catalogo;
  }

  function obtenerContextoBolsa() {
    return composicion?.contextos.bolsa || null;
  }

  function vistaDisponible(vista) {
    if (vista === "contratacion-temporal") {
      return composicion?.contratacionTemporal !== undefined;
    }
    if (vista === "cronos") return composicion?.cronos !== undefined;
    if (vista === "dietas") return composicion?.dietas !== undefined;
    if (vista === "personal") return composicion?.personal !== undefined;
    const clavePresentacion = CLAVE_POR_VISTA_PRESENTACION[vista];
    if (clavePresentacion !== undefined) {
      return presentacionActiva && composicion?.vistasPresentacion?.[clavePresentacion] !== undefined;
    }
    return false;
  }

  // Portal.js usa este predicado en vez del inventario estático: las claves
  // visuales que coinciden con sub-vistas de Bolsa sólo se interceptan durante
  // la presentación, nunca en el portal interno.
  function vistaGestionada(vista) {
    return VISTAS_MODULOS_CONECTADOS.has(vista)
      || (presentacionActiva && VISTAS_PRESENTACION_VISUALES.has(vista));
  }

  function resolverAcceso(clave, bolsaDisponible = true) {
    if (clave === "bolsa") {
      const autorizada = !presentacionActiva || composicion?.contextos.bolsa !== undefined;
      const acceso = bolsaDisponible !== null && typeof bolsaDisponible === "object"
        ? bolsaDisponible
        : { disponible: bolsaDisponible === true, vista: "resumen" };
      return Object.freeze({
        ...acceso,
        disponible: acceso.disponible === true && autorizada,
        vista: acceso.disponible === true && autorizada ? acceso.vista : "",
      });
    }
    if (clave === "contratacion_temporal" && vistaDisponible("contratacion-temporal")) {
      return Object.freeze({ disponible: true, vista: "contratacion-temporal" });
    }
    if (clave === "cronos" && vistaDisponible("cronos")) {
      return Object.freeze({ disponible: true, vista: "cronos", ...(presentacionActiva ? {
        estado: "presentacion", etiqueta: "Recorrido visual · pendiente de backend",
        presentacion: true, accion_etiqueta: "Ver recorrido",
      } : {}) });
    }
    if (clave === "dietas" && vistaDisponible("dietas")) {
      return Object.freeze({ disponible: true, vista: "dietas", ...(presentacionActiva ? {
        estado: "presentacion", etiqueta: "Mapa conectado · expediente pendiente de backend",
        presentacion: true, accion_etiqueta: "Ver recorrido",
      } : {}) });
    }
    if (clave === CLAVE_PERSONAL && vistaDisponible("personal")) {
      return Object.freeze({ disponible: true, vista: "personal", ...(presentacionActiva ? {
        estado: "presentacion", etiqueta: "Catálogos conectados · ficha pendiente de backend",
        presentacion: true, accion_etiqueta: "Ver recorrido",
      } : { etiqueta: traducir("personal_catalogo_profesional") }) });
    }
    if (CLAVES_PRESENTACION_VISUAL.includes(clave)
      && vistaDisponible(VISTA_POR_CLAVE_PRESENTACION[clave])) {
      return Object.freeze({
        disponible: true,
        vista: VISTA_POR_CLAVE_PRESENTACION[clave],
        estado: "presentacion",
        etiqueta: "Recorrido visual · pendiente de backend",
        presentacion: true,
        accion_etiqueta: "Ver recorrido",
      });
    }
    if (clave === CLAVE_CONTRATACION_TEMPORAL && !presentacionActiva
      && catalogo.some((modulo) => modulo.clave === CLAVE_CONTRATACION_TEMPORAL)) {
      return Object.freeze({
        disponible: false,
        vista: "",
        estado: "no_disponible",
        textoEstado: traducir("estado_modulo_no_disponible_titulo"),
      });
    }
    if (CLAVES_CARGA_MODULAR.includes(clave)) {
      return Object.freeze({
        disponible: false,
        vista: "",
        estado: composicion?.estadosModulos?.[clave] || "denegado",
      });
    }
    if (CLAVES_PRESENTACION_VISUAL.includes(clave)) {
      return Object.freeze({
        disponible: false,
        vista: "",
        estado: composicion?.estadosModulos?.[clave] || "no_disponible",
      });
    }
    return Object.freeze({ disponible: false, vista: "" });
  }

  function renderizarNavegacion(bolsaDisponible = true, moduloActivo = "portal", vistaPermitida = () => true) {
    if (typeof vistaPermitida !== "function") throw new TypeError("filtro de vistas no válido");
    const catalogoVisible = moduloActivo === "portal"
      ? catalogo
      : catalogo.filter((modulo) => modulo.clave === moduloActivo);
    return renderizarNavegacionModulos({
      catalogo: catalogoVisible,
      resolverAcceso: (clave) => {
        const acceso = resolverAcceso(clave, bolsaDisponible);
        if (acceso.disponible !== true) return acceso;
        return vistaPermitida(acceso.vista) ? acceso : Object.freeze({
          ...acceso,
          disponible: false,
          vista: "",
          estado: "denegado",
          etiqueta: traducir("permiso_perfil_denegado"),
        });
      },
      escaparHTML,
      traducir,
    });
  }

  async function montarVista(vista, raiz, opciones = {}) {
    if (!vistaGestionada(vista) || !vistaDisponible(vista)) return false;
    if (!raiz || typeof raiz.replaceChildren !== "function") {
      throw new TypeError("raíz del módulo no válida");
    }
    desmontarVistaActual();
    const montaje = ++secuenciaMontaje;
    raiz.innerHTML = '<section class="panel"><div class="cuerpo-panel" role="status">Cargando módulo…</div></section>';

    if (vista === "contratacion-temporal") {
      const esFiscalizacion = composicion.contratacionTemporal.fiscalizacion !== null;
      const presentadorCT = composicion.contratacionTemporal.crearPresentador();
      if (opciones?.subvista && typeof presentadorCT?.cambiarVista === "function"
        && ["alta", "cuadro"].includes(opciones.subvista)) {
        try { presentadorCT.cambiarVista(opciones.subvista); } catch {}
      }
      if (typeof opciones?.expedienteRef === "string" && opciones.expedienteRef !== ""
        && typeof presentadorCT?.seleccionarExpediente === "function") {
        try { void presentadorCT.seleccionarExpediente(opciones.expedienteRef); } catch {}
      }
      if (!esFiscalizacion && opciones?.filtros && typeof presentadorCT?.cargar === "function") {
        // Filtros pedidos desde el inicio (cifras del resumen): se cargan antes de
        // montar la vista, que así pinta directamente el cuadro filtrado.
        try { await presentadorCT.cargar({ texto: "", estado: "", fase: "", ...opciones.filtros }); } catch {}
        if (montaje !== secuenciaMontaje) return false;
      }
      const moduloContratacion = esFiscalizacion
        ? await composicion.contratacionTemporal.montarFiscalizacion({
          raiz,
          cliente: composicion.contratacionTemporal.fiscalizacion.cliente,
          confirmarOperacion,
          anunciar,
        })
        : await composicion.contratacionTemporal.montar({
          raiz,
          presentador: presentadorCT,
          alta: composicion.contratacionTemporal.alta,
          analisis: composicion.contratacionTemporal.analisis,
          fiscalizacion: typeof composicion.contratacionTemporal.analisis?.cliente
            ?.registrarResultadoFiscalizacion === "function"
            ? { cliente: composicion.contratacionTemporal.analisis.cliente }
            : null,
          continuidad: composicion.contratacionTemporal.continuidad,
          subsanacion: composicion.contratacionTemporal.subsanacion,
          confirmarOperacion,
          anunciar,
        });
      if (montaje !== secuenciaMontaje) {
        moduloContratacion.desmontar();
        return false;
      }
      desmontarVista = moduloContratacion.desmontar;
      return true;
    }

    if (vista === "cronos") {
      if (typeof composicion.cronos.montar === "function") {
        raiz.replaceChildren();
        const modulo = await composicion.cronos.montar({ raiz, anunciar,
          registrarDesmontar: (limpiar) => {
            if (montaje !== secuenciaMontaje) { limpiar(); return; }
            desmontarVista = limpiar;
          },
        });
        if (montaje !== secuenciaMontaje) { modulo.desmontar(); return false; }
        desmontarVista = modulo.desmontar;
        return true;
      }
      raiz.innerHTML = composicion.cronos.renderizar();
      const retirarEventos = composicion.cronos.instalarEventos({ raiz, anunciar });
      if (montaje !== secuenciaMontaje) {
        retirarEventos();
        return false;
      }
      desmontarVista = retirarEventos;
      return true;
    }

    if (vista === "personal") {
      raiz.replaceChildren();
      const moduloPersonal = await composicion.personal.montar({
        raiz,
        cliente: composicion.personal.cliente,
        anunciar,
        registrarDesmontar: (limpiar) => {
          if (typeof limpiar !== "function") throw new TypeError("limpieza de Personal no válida");
          if (montaje !== secuenciaMontaje) { limpiar(); return; }
          desmontarVista = limpiar;
        },
      });
      if (montaje !== secuenciaMontaje) {
        moduloPersonal.desmontar();
        return false;
      }
      desmontarVista = moduloPersonal.desmontar;
      return true;
    }

    const clavePresentacion = CLAVE_POR_VISTA_PRESENTACION[vista];
    if (clavePresentacion !== undefined) {
      raiz.replaceChildren();
      const moduloPresentacion = await composicion.vistasPresentacion[clavePresentacion].montar({
        raiz,
        anunciar,
        registrarDesmontar: (limpiar) => {
          if (typeof limpiar !== "function") throw new TypeError("limpieza de presentación no válida");
          if (montaje !== secuenciaMontaje) { limpiar(); return; }
          desmontarVista = limpiar;
        },
      });
      if (!moduloPresentacion || typeof moduloPresentacion.desmontar !== "function") {
        throw new TypeError("montaje de presentación no válido");
      }
      if (montaje !== secuenciaMontaje) {
        moduloPresentacion.desmontar();
        return false;
      }
      desmontarVista = moduloPresentacion.desmontar;
      return true;
    }

    raiz.replaceChildren();
    const moduloDietas = await composicion.dietas.montar({
      raiz,
      calculador: composicion.dietas.calculador,
      visorRuta: composicion.dietas.visorRuta,
      anunciar,
      registrarDesmontar: (limpiar) => {
        if (typeof limpiar !== "function") throw new TypeError("limpieza de Dietas no válida");
        if (montaje !== secuenciaMontaje) {
          limpiar();
          return;
        }
        desmontarVista = limpiar;
      },
    });
    if (montaje !== secuenciaMontaje) {
      moduloDietas.desmontar();
      return false;
    }
    desmontarVista = moduloDietas.desmontar;
    return true;
  }

  function esPerfilRRHH() {
    if (!vistaDisponible("contratacion-temporal")) return false;
    return composicion?.contratacionTemporal?.fiscalizacion === null;
  }

  function obtenerMetricasCuadro() {
    if (!esPerfilRRHH()) return null;
    return composicion?.contratacionTemporal?.obtenerMetricas?.() || null;
  }

  function obtenerTramitesInicio() {
    if (!esPerfilRRHH()) return null;
    return composicion?.contratacionTemporal?.obtenerTramitesInicio?.() || null;
  }

  return Object.freeze({
    cargarInterno,
    cargarPresentacion,
    desmontarVistaActual,
    esPerfilRRHH,
    montarVista,
    obtenerTramitesInicio,
    obtenerCatalogo,
    obtenerContextoBolsa,
    obtenerMetricasCuadro,
    renderizarNavegacion,
    resolverAcceso,
    vistaGestionada,
    vistaDisponible,
  });
}
