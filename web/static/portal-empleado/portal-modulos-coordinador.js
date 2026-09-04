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
} from "./portal-catalogo-modulos.js?v=20260831-ct-catalogo-v1";
import { traducirPortal } from "./portal-i18n.js?v=20260831-ct-catalogo-i18n-v1";

const CLAVE_CONTRATACION_TEMPORAL = "contratacion_temporal";
const CLAVES_CARGA_MODULAR = Object.freeze([
  CLAVE_CONTRATACION_TEMPORAL,
  "cronos",
  "dietas",
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
    const [contrato, presentador, datos, adaptador, documentos] = await Promise.all([
      import("./modulos/cronos/contrato.js"),
      import("./modulos/cronos/presentador.js"),
      import("./modulos/cronos/datos-presentacion.js"),
      import("./modulos/cronos/adaptador-presentacion.js"),
      import("./documentos/descarga-recibos-presentacion.js"),
    ]);
    return Object.freeze({ contrato, presentador, datos, adaptador, documentos });
  },
  dietas: async () => {
    const [contrato, vista, mapa, adaptador, calculador, documentos] = await Promise.all([
      import("./modulos/dietas/contrato.js"),
      import("./modulos/dietas/vista.js"),
      import("./modulos/dietas/mapa-ruta.js"),
      import("./modulos/dietas/adaptador-presentacion.js"),
      import("./modulos/dietas/calculador-rutas-presentacion-osrm.js"),
      import("./documentos/descarga-recibos-presentacion.js"),
    ]);
    return Object.freeze({ contrato, vista, mapa, adaptador, calculador, documentos });
  },
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
  claves = CLAVES_CARGA_MODULAR,
  limiteMs = LIMITE_CARGA_MODULAR_MS,
  temporizadores = globalThis,
} = {}) {
  if (!Array.isArray(claves)
    || claves.some((clave) => !CLAVES_CARGA_MODULAR.includes(clave))
    || new Set(claves).size !== claves.length
    || !Number.isSafeInteger(limiteMs) || limiteMs < 1 || limiteMs > 10_000) {
    throw new TypeError("configuración de carga modular no válida");
  }
  const clavesSolicitadas = new Set(claves);
  const resultados = await Promise.allSettled(CLAVES_CARGA_MODULAR.map((clave) => {
    if (!clavesSolicitadas.has(clave)) return Promise.resolve(undefined);
    const cargar = cargadores?.[clave];
    if (typeof cargar !== "function") {
      return Promise.reject(new TypeError(`cargador modular ausente: ${clave}`));
    }
    return cargarModuloConLimite(cargar, clave, limiteMs, temporizadores);
  }));
  return Object.freeze(Object.fromEntries(CLAVES_CARGA_MODULAR.map((clave, indice) => {
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

function capacidadesCronos(contrato) {
  return Object.freeze([
    contrato.CAPACIDAD_CONSULTAR_FICHAJES,
    contrato.CAPACIDAD_REGISTRAR_FICHAJE,
    contrato.CAPACIDAD_CONSULTAR_HORARIO,
    contrato.CAPACIDAD_CONSULTAR_PERMISOS,
    contrato.CAPACIDAD_SOLICITAR_PERMISO,
  ]);
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

export const VISTAS_MODULOS_PERSONALES = Object.freeze(new Set(["cronos", "dietas"]));
export const VISTAS_MODULOS_CONECTADOS = Object.freeze(new Set([
  "contratacion-temporal", ...VISTAS_MODULOS_PERSONALES,
]));

export function moduloDeVistaPortal(vista) {
  if (vista === "portal") return "portal";
  if (vista === "contratacion-temporal") return "contratacion_temporal";
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
    const cargas = await resolverCargasModularesPresentacion(
      cargadoresPresentacion,
      {
        claves: CLAVES_CARGA_MODULAR.filter((clave) => contextos[clave] !== undefined),
        limiteMs: limiteCargaModularMs,
        temporizadores,
      },
    );
    if (carga !== secuenciaCarga) throw new Error("carga de presentación sustituida");
    const origenComprobacion = entorno.location?.origin || "";
    const cronos = componerModuloAislado(contextos.cronos, cargas.cronos, (recursos) => {
      const capacidades = capacidadesCronos(recursos.contrato);
      return recursos.presentador.crearPresentadorCronos({
        contextoActor: contextos.cronos,
        capacidades,
        datos: recursos.datos.crearDatosCronosPresentacion(contextos.cronos),
        ejecutor: recursos.adaptador.crearEjecutorCronosPresentacion(),
        descargarRecibo: recursos.documentos.crearDescargadorRecibosPresentacion(entorno),
        origenComprobacion,
      });
    });
    const dietas = componerModuloAislado(contextos.dietas, cargas.dietas, (recursos) => {
      const capacidades = capacidadesDietas(recursos.contrato);
      return Object.freeze({
        contextoActor: contextos.dietas,
        capacidades,
        adaptador: recursos.adaptador.crearAdaptadorDietasPresentacion({
          contextoActor: contextos.dietas,
          capacidades,
        }),
        calculadorRuta: recursos.calculador.crearCalculadorRutasDietasPresentacionOSRM({
          contextoActor: contextos.dietas,
          capacidades,
          fetchImpl: typeof entorno.fetch === "function" ? entorno.fetch.bind(entorno) : undefined,
        }),
        descargarRecibo: recursos.documentos.crearDescargadorRecibosPresentacion(entorno),
        visorRuta: recursos.mapa.crearVisorRutaDietas({ entorno, permitirTeselas: true }),
        montar: recursos.vista.montarModuloDietas,
        origenComprobacion,
      });
    });
    const contratacionTemporal = componerModuloAislado(
      contextos.contratacion_temporal,
      cargas.contratacion_temporal,
      (recursos) => {
        const fuente = recursos.adaptador.crearAdaptadorContratacionTemporalPresentacion({
          contextoActor: contextos.contratacion_temporal,
        });
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
          montar: recursos.vista.montarModuloContratacionTemporal,
        });
      },
    );
    if (carga !== secuenciaCarga) throw new Error("carga de presentación sustituida");
    catalogo = base.catalogo.obtenerCatalogoModulosPresentacion();
    composicion = Object.freeze({
      contextos,
      contratacionTemporal,
      cronos,
      dietas,
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
        const fuente = recursos.adaptador
          .crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente });
        const controladorConsulta = new AbortController();
        controladorCargaInterna = controladorConsulta;
        let cuadroDisponible = false;
        try {
          await consultarConLimite(
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
        let alta = null;
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
          montar: recursos.vista.montarModuloContratacionTemporal,
          montarFiscalizacion: recursos.vista.montarModuloFiscalizacionContratacionTemporal,
        });
      } catch {
        contratacionTemporal = undefined;
      }
    }
    if (carga !== secuenciaCarga) throw new Error("carga interna sustituida");
    composicion = Object.freeze({
      contextos: Object.freeze({}),
      contratacionTemporal,
      estadosModulos: Object.freeze({
        contratacion_temporal: contratacionTemporal === undefined
          ? "no_disponible" : "disponible",
        cronos: "no_disponible",
        dietas: "no_disponible",
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
    return false;
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
      return Object.freeze({ disponible: true, vista: "cronos" });
    }
    if (clave === "dietas" && vistaDisponible("dietas")) {
      return Object.freeze({ disponible: true, vista: "dietas" });
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

  async function montarVista(vista, raiz) {
    if (!VISTAS_MODULOS_CONECTADOS.has(vista) || !vistaDisponible(vista)) return false;
    if (!raiz || typeof raiz.replaceChildren !== "function") {
      throw new TypeError("raíz del módulo no válida");
    }
    desmontarVistaActual();
    const montaje = ++secuenciaMontaje;
    raiz.innerHTML = '<section class="panel"><div class="cuerpo-panel" role="status">Cargando módulo…</div></section>';

    if (vista === "contratacion-temporal") {
      const esFiscalizacion = composicion.contratacionTemporal.fiscalizacion !== null;
      const moduloContratacion = esFiscalizacion
        ? await composicion.contratacionTemporal.montarFiscalizacion({
          raiz,
          cliente: composicion.contratacionTemporal.fiscalizacion.cliente,
          confirmarOperacion,
          anunciar,
        })
        : await composicion.contratacionTemporal.montar({
          raiz,
          presentador: composicion.contratacionTemporal.crearPresentador(),
          alta: composicion.contratacionTemporal.alta,
          analisis: composicion.contratacionTemporal.analisis,
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
      raiz.innerHTML = composicion.cronos.renderizar();
      const retirarEventos = composicion.cronos.instalarEventos({ raiz, anunciar });
      if (montaje !== secuenciaMontaje) {
        retirarEventos();
        return false;
      }
      desmontarVista = retirarEventos;
      return true;
    }

    const moduloDietas = await composicion.dietas.montar({
      raiz,
      contextoActor: composicion.dietas.contextoActor,
      capacidades: composicion.dietas.capacidades,
      adaptador: composicion.dietas.adaptador,
      calculadorRuta: composicion.dietas.calculadorRuta,
      descargarRecibo: composicion.dietas.descargarRecibo,
      visorRuta: composicion.dietas.visorRuta,
      confirmarOperacion,
      origenComprobacion: composicion.dietas.origenComprobacion,
      anunciar,
    });
    if (montaje !== secuenciaMontaje) {
      moduloDietas.desmontar();
      return false;
    }
    desmontarVista = moduloDietas.desmontar;
    return true;
  }

  return Object.freeze({
    cargarInterno,
    cargarPresentacion,
    desmontarVistaActual,
    montarVista,
    obtenerCatalogo,
    obtenerContextoBolsa,
    renderizarNavegacion,
    resolverAcceso,
    vistaDisponible,
  });
}
