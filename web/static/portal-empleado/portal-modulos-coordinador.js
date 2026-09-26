/**
 * Composición de módulos del Portal del Empleado.
 *
 * El shell conserva la navegación, el tema y el router comunes. Este archivo
 * solo registra adaptadores disponibles y monta su vista; no contiene reglas
 * de negocio.
 */
import {
  cargarCatalogoModulosInterno,
  renderizarNavegacionModulos,
} from "./portal-catalogo-modulos.js?v=20260926-reparos-informe-v1";
import { traducirPortal } from "./portal-i18n.js?v=20260926-reparos-informe-v1";
import { calcularMetricasCuadro, tramitesParaInicio } from "./portal-inicio.js?v=20260926-reparos-informe-v1";
import {
  componerCronosInterno,
  componerDietasInternas,
  componerPersonalVisible,
  componerRegistroPersonal,
} from "./portal-composicion-empleado.js?v=20260925-cronos-notif-e10-v1";
import { VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js?v=20260926-reparos-informe-v1";
import {
  CLAVES_CARGA_MODULAR,
  LIMITE_CARGA_MODULAR_MS,
  cargarModuloConLimite,
  consultarConLimite,
} from "./portal-modulos-carga.js?v=20260926-integracion-bolsa-ct-v1";

const CLAVE_CONTRATACION_TEMPORAL = "contratacion_temporal";
const SIN_CATALOGOS_PUBLICOS = Object.freeze({ recursos: Object.freeze({}), disponibles: Object.freeze([]) });
const CLAVE_PERSONAL = "personal";
const CLAVE_DOCUMENTOS = "documentos";
// «documentos» es también una sección de Bolsa; la vista del servicio común
// de Documentos usa un nombre propio para no confundirse con ella.
export const VISTA_DOCUMENTOS_EXPEDIENTE = "documentos-expediente";
export const CLAVES_MODULOS_VEC_REGISTRADOS = Object.freeze([
  CLAVE_PERSONAL, "cronos", "dietas", CLAVE_DOCUMENTOS, "bolsa",
  CLAVE_CONTRATACION_TEMPORAL, "administracion", "usuarios",
]);
// Módulos del registro con pantalla en este portal. Administración y Usuarios
// siguen publicados en `/api/vec/modules` para la carcasa general (que sí los
// consume), pero aquí no tienen vista: no se ofrecen en menú ni en Inicio.
export const CLAVES_MODULOS_CON_VISTA_PORTAL = Object.freeze([
  CLAVE_PERSONAL, "cronos", "dietas", CLAVE_DOCUMENTOS, "bolsa", CLAVE_CONTRATACION_TEMPORAL,
]);
// Documentos sólo se carga cuando el servidor lo tiene montado, pero no tiene
// entrada propia en menú ni en Inicio: su vista se abre únicamente desde un
// expediente, que le entrega la referencia a consultar. Personal, Cronos y
// Dietas se cargan y conservan su vista, pero tampoco tienen entrada en el
// menú, en Inicio ni en el ayudante de trámites: el portal solo ofrece Bolsa
// y la contratación temporal. Su URL directa sigue funcionando.
export const CLAVES_SIN_ENTRADA_PORTAL = Object.freeze([CLAVE_DOCUMENTOS, CLAVE_PERSONAL, "cronos", "dietas"]);
const CLAVES_CARGA_PORTAL = Object.freeze([...CLAVES_CARGA_MODULAR, CLAVE_DOCUMENTOS]);
// Módulos sin entrada que no se cargan al arrancar, sino al pedir una de sus
// vistas. Documentos no se difiere: se abre desde el expediente y su carga no
// hace consultas.
const CLAVES_DIFERIDAS_PORTAL = Object.freeze([CLAVE_PERSONAL, "cronos", "dietas"]);
// Rol con el que la frontera de identidad atesta a Intervención. Solo decide
// qué pantalla se ofrece; cada operación la sigue autorizando el servidor.
const ROL_INTERVENCION = "intervencion";
const CARGADORES_INTERNOS_PREDETERMINADOS = Object.freeze({
  // El portal interno monta las vistas conectadas de la persona empleada.
  // Los clientes se piden por la
  // misma URL que usan las vistas (sin ?v=, servida no-cache): así sus clases
  // de error son la misma y los instanceof de cada vista siguen valiendo.
  // i18n.js y vista-movimientos-propios.js van versionados con la misma URL
  // que piden las vistas, para que cada módulo se evalúe una sola vez.
  cronos: async () => {
    const [saldo, remoto, movimientos, movimientosPropios, permisosPropios,
      clienteSaldo, clienteRemoto, clienteSolicitudes, i18n,
      bandejaPermisos, avisosPropios, clienteResolucion, i18nResolucion,
      notificacionesPropias, bandejaNotificaciones, clienteNotificaciones, i18nNotificaciones] = await Promise.all([
      import("./modulos/cronos/vista-saldo-conectado.js?v=20260925-tanda2-v1"),
      import("./modulos/cronos/vista-remoto.js?v=20260925-tanda2-v1"),
      import("./modulos/cronos/vista-movimientos-conectado.js?v=20260925-tanda2-v1"),
      import("./modulos/cronos/vista-movimientos-propios.js?v=20260925-tanda2-v1"),
      import("./modulos/cronos/vista-permisos-propios.js?v=20260925-cronos-notif-e10-v1"),
      import("./modulos/cronos/cliente-saldo-http.js"),
      import("./modulos/cronos/cliente-remoto-http.js"),
      import("./modulos/cronos/cliente-solicitudes-http.js"),
      import("./modulos/cronos/i18n.js?v=20260925-tanda2-v1"),
      import("./modulos/cronos/vista-bandeja-permisos.js?v=20260925-cronos-notif-e10-v1"),
      import("./modulos/cronos/vista-avisos-propios.js?v=20260925-cronos-notif-e10-v1"),
      import("./modulos/cronos/cliente-resolucion-http.js"),
      import("./modulos/cronos/i18n-resolucion.js"),
      import("./modulos/cronos/vista-notificaciones-propias.js?v=20260925-cronos-notif-e10-v1"),
      import("./modulos/cronos/vista-bandeja-notificaciones.js?v=20260925-cronos-notif-e10-v1"),
      import("./modulos/cronos/cliente-notificaciones-http.js"),
      import("./modulos/cronos/i18n-notificaciones.js"),
    ]);
    return Object.freeze({ saldo, remoto, movimientos, movimientosPropios, permisosPropios,
      clienteSaldo, clienteRemoto, clienteSolicitudes, i18n, bandejaPermisos, avisosPropios, clienteResolucion, i18nResolucion,
      notificacionesPropias, bandejaNotificaciones, clienteNotificaciones, i18nNotificaciones });
  },
  contratacion_temporal: async () => {
    const [contrato, cliente, presentador, vista, adaptador] = await Promise.all([
      import("./modulos/contratacion-temporal/contrato.js"),
      import("./modulos/contratacion-temporal/cliente-http.js"),
      import("./modulos/contratacion-temporal/presentador-expedientes.js"),
      import("./modulos/contratacion-temporal/vista-expedientes.js?v=20260926-reparos-informe-v1"),
      import("./modulos/contratacion-temporal/adaptador-http-expedientes.js"),
    ]);
    return Object.freeze({ contrato, cliente, presentador, vista, adaptador });
  },
  personal: async () => {
    const [contrato, cliente, vista, ficha, registro, clienteRegistro, clienteCatalogosRegistro, i18n, clienteFichaPropia] = await Promise.all([
      import("./modulos/personal/contrato.js?v=20260920-personal-catalogo-v1"),
      import("./modulos/personal/cliente-http-categorias.js?v=20260925-portal-integrado-v1"),
      import("./modulos/personal/vista.js?v=20260925-personal-e10-v1"),
      import("./modulos/personal/vista-ficha-integral.js?v=20260925-personal-e10-v1"),
      import("./modulos/personal/registro-b2.js?v=20260925-personal-e10-v1"),
      import("./modulos/personal/registro-b2-cliente.js?v=20260925-b2-selector-v1"),
      import("./modulos/personal/registro-b2-catalogos-cliente.js?v=20260925-b2-mtls-v1"),
      import("./modulos/personal/i18n.js?v=20260925-personal-e10-v1"),
      import("./modulos/personal/cliente-http-ficha-propia.js?v=20260925-personal-e10-v1"),
    ]);
    return Object.freeze({ contrato, cliente, vista, clienteCategorias: cliente, vistaCategorias: vista,
      ficha, registro, clienteRegistro, clienteCatalogosRegistro, i18n, clienteFichaPropia });
  },
  // Catálogos públicos de Personal (RPT publicada y estructura de referencia).
  // Van en todos los paquetes web (el import nunca da 404); solo se ofrecen si
  // el servidor responde a su consulta con una página válida.
  personal_catalogos_publicos: async () => {
    const [clienteRPT, vistaRPT, clienteEstructura, vistaEstructura] = await Promise.all([
      import("./modulos/personal/cliente-http-rpt-publica.js?v=20260925-portal-integrado-v1"),
      import("./modulos/personal/vista-rpt-publica.js?v=20260925-portal-integrado-v1"),
      import("./modulos/personal/cliente-http-estructura-organizativa-publica.js?v=20260925-portal-integrado-v1"),
      import("./modulos/personal/vista-estructura-organizativa-publica.js?v=20260925-personal-e10-v1"),
    ]);
    return Object.freeze({ clienteRPT, vistaRPT, clienteEstructura, vistaEstructura });
  },
  dietas: async () => {
    const [contrato, recorridos, clienteBorradores, clienteAsignacion, calculador, mapa, clienteCircuito] = await Promise.all([
      import("./modulos/dietas/contrato.js"),
      import("./modulos/dietas/vista-recorridos.js?v=20260926-reparos-informe-v1"),
      import("./modulos/dietas/cliente-borradores-http.js?v=20260925-d5d6-v2"),
      import("./modulos/dietas/cliente-asignacion-http.js?v=20260925-d5d6-v1"),
      import("./modulos/dietas/calculador-rutas-http.js?v=20260925-d5d6-v1"),
      import("./modulos/dietas/mapa-ruta.js?v=20260926-reparos-informe-v1"),
      import("./modulos/dietas/cliente-circuito-http.js?v=20260925-d5d6-v1"),
    ]);
    return Object.freeze({ contrato, recorridos, clienteBorradores, clienteAsignacion, calculador, mapa, clienteCircuito });
  },
  // Consulta del expediente documental (RRHH). El servidor sólo publica el
  // módulo cuando su montaje está compuesto; cada consulta la autoriza V3.
  documentos: async () => {
    const [vista, cliente] = await Promise.all([
      import("./modulos/documentos/vista.js?v=20260925-documentos-web-v3"),
      import("./modulos/documentos/cliente-http.js?v=20260926-integracion-bolsa-ct-v1"),
    ]);
    return Object.freeze({ vista, cliente });
  },
});

export const VISTAS_MODULOS_PERSONALES = Object.freeze(new Set(["cronos", "cronos-permisos", "cronos-avisos", "cronos-bandeja",
  "cronos-notificaciones", "cronos-bandeja-notificaciones", "dietas", "personal", "personal-registro"]));
const SUBVISTAS_CRONOS = Object.freeze(new Set(["cronos-permisos", "cronos-avisos", "cronos-bandeja", "cronos-notificaciones", "cronos-bandeja-notificaciones"]));
const VISTAS_MODULO_BOLSA = Object.freeze(new Set(VISTAS_INTERNAS_BOLSA));
export const VISTAS_MODULOS_CONECTADOS = Object.freeze(new Set([
  "contratacion-temporal", VISTA_DOCUMENTOS_EXPEDIENTE, ...VISTAS_MODULOS_PERSONALES,
]));

// Estado de un módulo autorizado sin entrada en el portal que aún no se ha
// pedido: no se carga hasta que se abre una de sus vistas.
const ESTADO_DIFERIDO = "diferido";

/** Código del error con el que se rechaza una carga sustituida por otra. */
export const CODIGO_CARGA_SUSTITUIDA = "carga_sustituida";
function errorCargaSustituida() {
  return Object.assign(new Error("carga interna sustituida"), { codigo: CODIGO_CARGA_SUSTITUIDA });
}

export function moduloDeVistaPortal(vista) {
  if (vista === "portal") return "portal";
  if (vista === "contratacion-temporal") return "contratacion_temporal";
  if (vista === "personal" || vista === "personal-registro") return CLAVE_PERSONAL;
  if (SUBVISTAS_CRONOS.has(vista)) return "cronos";
  if (vista === VISTA_DOCUMENTOS_EXPEDIENTE) return CLAVE_DOCUMENTOS;
  if (VISTAS_MODULOS_PERSONALES.has(vista)) return vista;
  if (VISTAS_MODULO_BOLSA.has(vista)) return "bolsa";
  return "";
}

/** Indica si una vista pertenece a un módulo que se ofrece en menú e Inicio. */
export function vistaConEntradaPortal(vista) {
  return !CLAVES_SIN_ENTRADA_PORTAL.includes(moduloDeVistaPortal(vista));
}

export function rutaDeVistaPortal(vista) {
  if (vista === "portal") return "#portal";
  if (vista === "contratacion-temporal") return "#contratacion-temporal";
  if (vista === VISTA_DOCUMENTOS_EXPEDIENTE) return `#${VISTA_DOCUMENTOS_EXPEDIENTE}`;
  if (VISTAS_MODULOS_PERSONALES.has(vista)) return `#${vista}`;
  if (VISTAS_MODULO_BOLSA.has(vista)) return `#bolsa/${vista}`;
  return "#portal";
}

export function crearCoordinadorModulosPortal({
  escaparHTML,
  anunciar = () => {},
  confirmarOperacion = () => false,
  montajeBolsa = null,
  entorno = globalThis,
  traducir = traducirPortal,
  cargarCatalogoInterno = null,
  cargadoresInternos = CARGADORES_INTERNOS_PREDETERMINADOS,
  consultarSesion = null,
  limiteCargaModularMs = LIMITE_CARGA_MODULAR_MS,
  temporizadores = globalThis,
  // Módulos que no se cargan al arrancar sino al pedir una de sus vistas.
  modulosDiferidos = CLAVES_DIFERIDAS_PORTAL,
} = {}) {
  if (typeof escaparHTML !== "function" || typeof anunciar !== "function"
    || typeof confirmarOperacion !== "function" || typeof traducir !== "function"
    || (cargarCatalogoInterno !== null && typeof cargarCatalogoInterno !== "function")
    || (consultarSesion !== null && typeof consultarSesion !== "function")
    || (montajeBolsa !== null && (typeof montajeBolsa?.montar !== "function"
      || typeof montajeBolsa?.disponible !== "function"))
    || typeof cargadoresInternos?.contratacion_temporal !== "function"
    || !Number.isSafeInteger(limiteCargaModularMs)
    || limiteCargaModularMs < 1 || limiteCargaModularMs > 10_000
    || !Array.isArray(modulosDiferidos) || !modulosDiferidos.every((clave) => CLAVES_CARGA_PORTAL.includes(clave))) {
    throw new TypeError("dependencias del coordinador de módulos no válidas");
  }

  let catalogo = Object.freeze([]);
  let catalogoOfrecido = Object.freeze([]);
  let composicion = null;
  let desmontarVista = null;
  let vistaMontada = "";
  let raizMontada = null;
  let referenciaElaboracionMontada = "";
  let secuenciaMontaje = 0;
  let secuenciaCarga = 0;
  // Hay una carga en curso (catálogo o módulos) que aún no ha terminado. Nace
  // en verdadero: hasta la primera carga el shell tampoco sabe nada del perfil.
  let cargaEnCurso = true;
  // Consultas de red de la carga en curso: los módulos se cargan en paralelo,
  // así que puede haber varias a la vez. Sustituir la carga las aborta todas.
  const controladoresCarga = new Set();
  // Sonda del registro RRHH de Personal: se hace al entrar por primera vez en
  // la vista y su resultado vale para la sesión (null: aún sin comprobar). Así
  // abrir Personal no genera una lectura auditada del registro.
  let registroPersonalServido = null;
  // Arranca un módulo diferido dentro de la carga vigente (null sin carga).
  let cargaDiferida = null;
  let sondaRegistro = null;

  // Invalida siempre la carga en curso, también entre dos consultas (cuando no
  // hay ninguna pendiente): sus módulos tardíos ya no publican ni avisan.
  function cancelarCargaInterna() {
    secuenciaCarga += 1;
    cargaEnCurso = false;
    for (const controlador of controladoresCarga) controlador.abort();
    controladoresCarga.clear();
  }

  // Retira la vista montada sin tocar la composición de módulos: cambiar de
  // vista (o repintar Inicio) mientras aún cargan otros módulos no los cancela.
  function retirarVistaMontada() {
    secuenciaMontaje += 1;
    // Una sonda interrumpida no decide nada: se repetirá al volver a entrar.
    sondaRegistro?.abort();
    sondaRegistro = null;
    if (typeof desmontarVista === "function") desmontarVista();
    desmontarVista = null;
    vistaMontada = "";
    raizMontada = null;
    referenciaElaboracionMontada = "";
  }

  function desmontarVistaActual() {
    retirarVistaMontada();
    cancelarCargaInterna();
  }

  function reutilizarElaboracion(vista, raiz, opciones) {
    if (vista !== "elaboracion" || vistaMontada !== vista || raizMontada !== raiz) return false;
    if (opciones === null || typeof opciones !== "object" || !Object.hasOwn(opciones, "referencia")) return true;
    return opciones.referencia === referenciaElaboracionMontada;
  }

  function fetchDelEntorno() {
    return typeof entorno.fetch === "function" ? entorno.fetch.bind(entorno) : undefined;
  }

  // En contratación temporal, el cuadro, los catálogos de alta y la configuración
  // del análisis son consultas independientes; se piden a la vez.
  async function cargarContratacionTemporal({ consultar, exigirVigente, notificar = () => {} }) {
    const recursos = await cargarModuloConLimite(
      cargadoresInternos.contratacion_temporal,
      CLAVE_CONTRATACION_TEMPORAL,
      limiteCargaModularMs,
      temporizadores,
    );
    exigirVigente();
    const cliente = recursos.cliente.crearClienteHTTPContratacionTemporal({
      fetchImpl: fetchDelEntorno(),
      HeadersImpl: entorno.Headers,
    });
    let alta = null;
    // La jornada completa de referencia llega con la configuración del análisis.
    let jornadaCompleta = null;
    const fuente = recursos.adaptador
      .crearAdaptadorHTTPExpedientesContratacionTemporal({
        cliente, obtenerCatalogos: () => alta?.catalogos ?? null,
        obtenerJornadaCompleta: () => jornadaCompleta,
      });
    // Los catálogos del alta (centros y categorías) no retrasan el cuadro:
    // Inicio se pinta con el cuadro y la configuración, y los nombres de centro
    // y categoría aparecen en cuanto llegan (se avisa con `notificar`). Abrir
    // Contratación espera a que terminen, para que el alta y la bandeja se
    // monten ya con ellos.
    const promesaAlta = consultar((opciones) => cliente.obtenerCatalogosAlta(opciones)).then((valor) => {
      try {
        alta = Object.freeze({
          catalogos: recursos.contrato.validarCatalogosAlta(valor),
          capacidad: recursos.contrato.CAPACIDAD_CREAR_SOLICITUD,
          ejecutor: cliente.registrarSolicitud,
        });
      } catch {
        alta = null;
      }
    }, () => { alta = null; });
    const [cuadro, configuracion] = await Promise.allSettled([
      consultar((opciones) => fuente.listar(opciones)),
      consultar((opciones) => cliente.obtenerConfiguracionAnalisis(opciones)),
    ]);
    exigirVigente();
    const cuadroDisponible = cuadro.status === "fulfilled";
    const listadoCuadro = cuadroDisponible ? cuadro.value : null;
    let analisis = null;
    let subsanacion = null;
    if (configuracion.status === "fulfilled") {
      const configuracionAnalisis = configuracion.value;
      try {
        if (configuracionAnalisis.subsanacion_disponible === true
          && typeof cliente.registrarSubsanacionReparos === "function") {
          subsanacion = Object.freeze({ disponible: true, cliente });
        }
        jornadaCompleta = configuracionAnalisis.jornada_completa_minutos_semanales;
        analisis = Object.freeze({
          cliente,
          catalogos: Object.freeze({
            modalidades: configuracionAnalisis.modalidades,
            categorias: configuracionAnalisis.categorias,
            causas: configuracionAnalisis.causas,
            entradas_rc: configuracionAnalisis.entradas_rc,
            motivos_rectificacion: configuracionAnalisis.motivos_rectificacion,
            jornada_completa_minutos_semanales: configuracionAnalisis.jornada_completa_minutos_semanales,
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
        analisis = null;
      }
    }
    // Sin cuadro o sin análisis, el perfil (RRHH o Intervención) depende
    // también del alta: entonces sí se esperan sus catálogos.
    const altaPendiente = cuadroDisponible && analisis !== null;
    if (!altaPendiente) {
      await promesaAlta;
      exigirVigente();
    } else {
      void promesaAlta.then(() => notificar());
    }
    // Sin alta ni análisis, la única pantalla posible es la de fiscalización,
    // y solo se ofrece si la sesión atesta el perfil de Intervención: a otra
    // persona (p. ej. una empleada sin concesión) no se le monta un formulario
    // cuyas operaciones el servidor le denegaría.
    const fiscalizacion = alta === null && analisis === null
      && typeof cliente.registrarResultadoFiscalizacion === "function"
      && typeof recursos.vista.montarModuloFiscalizacionContratacionTemporal === "function"
      && await sesionDeIntervencion(consultar)
      ? Object.freeze({ cliente }) : null;
    exigirVigente();
    if (!cuadroDisponible && alta === null && fiscalizacion === null) {
      throw new Error("contratación temporal no disponible");
    }
    return {
      contratacionTemporal: Object.freeze({
        crearPresentador: () => recursos.presentador
          .crearPresentadorExpedientesContratacionTemporal({
            fuente, capacidades: fuente.capacidades,
            altaDisponible: alta !== null,
          }),
        get alta() { return alta; },
        esperarAlta: () => promesaAlta,
        analisis,
        fiscalizacion,
        subsanacion,
        continuidad: fiscalizacion === null ? Object.freeze({ cliente }) : null,
        obtenerMetricas: () => (listadoCuadro ? calcularMetricasCuadro(listadoCuadro) : null),
        // Número, centro y categoría se presentan al pedirlo, con los catálogos de alta que hayan llegado.
        obtenerTramitesInicio: () => {
          if (!listadoCuadro) return null;
          const { etiquetaCatalogo: etiqueta } = recursos.adaptador;
          return tramitesParaInicio(listadoCuadro).map((e) => ({
            ...e,
            numero_visible: (recursos.vista.numeroExpedienteVisible ?? String)(e.numero_visible),
            centro: etiqueta(alta?.catalogos?.centros, e.centro),
            categoria: etiqueta(alta?.catalogos?.categorias, e.categoria),
          }));
        },
        montar: recursos.vista.montarModuloContratacionTemporal,
        montarFiscalizacion: recursos.vista.montarModuloFiscalizacionContratacionTemporal,
      }),
    };
  }

  async function sesionDeIntervencion(consultar) {
    if (consultarSesion === null) return false;
    try {
      const sesion = await consultar((opciones) => consultarSesion(opciones), "consultar sesión");
      return Array.isArray(sesion?.roles) && sesion.roles.includes(ROL_INTERVENCION);
    } catch {
      return false;
    }
  }

  async function cargarCronos({ exigirVigente }) {
    const recursos = await cargarModuloConLimite(
      cargadoresInternos.cronos || CARGADORES_INTERNOS_PREDETERMINADOS.cronos,
      "cronos", limiteCargaModularMs, temporizadores,
    );
    exigirVigente();
    // Falta cualquier montar* o cliente → undefined: falla cerrado.
    const cronos = componerCronosInterno(recursos, entorno);
    if (!cronos) throw new TypeError("vistas de Cronos no disponibles");
    return { cronos };
  }

  // Personal y sus catálogos públicos opcionales se cargan a la vez.
  async function cargarPersonal({ consultar, exigirVigente }) {
    const [principal, publicos] = await Promise.allSettled([
      cargarModuloConLimite(
        cargadoresInternos.personal || CARGADORES_INTERNOS_PREDETERMINADOS.personal,
        CLAVE_PERSONAL, limiteCargaModularMs, temporizadores,
      ),
      cargarCatalogosPublicosPersonal({ consultar }),
    ]);
    exigirVigente();
    if (principal.status !== "fulfilled") throw new TypeError("vista de Personal no disponible");
    const recursos = principal.value;
    if (typeof recursos?.cliente?.crearClienteHTTPCategoriasPersonal !== "function"
      || typeof recursos?.vista?.montarModuloPersonal !== "function"
      || recursos?.contrato?.CAPACIDAD_CONSULTAR_PUESTO !== "personal.puesto.read") {
      throw new TypeError("vista de Personal no disponible");
    }
    const catalogos = publicos.status === "fulfilled" ? publicos.value : SIN_CATALOGOS_PUBLICOS;
    const personal = typeof recursos.ficha?.montarVistaFichaIntegralPersonal === "function"
      ? componerPersonalVisible({ ...recursos, ...catalogos.recursos }, entorno, {
        catalogosPublicos: catalogos.disponibles, ocultarSinFuente: true,
        destinosDisponibles: () => ({ dietas: vistaDisponible("dietas"), cronos: vistaDisponible("cronos") }),
      })
      : Object.freeze({
        cliente: recursos.cliente.crearClienteHTTPCategoriasPersonal({ fetchImpl: fetchDelEntorno() }),
        montar: recursos.vista.montarModuloPersonal,
      });
    if (!personal) throw new TypeError("ficha de Personal no disponible");
    // El registro RRHH es una vista aparte: si falta, Personal sigue. Se compone
    // siempre; lo ofrece vistaDisponible («personal-registro») solo al perfil
    // RRHH y su sonda se hace al entrar en la vista, no al cargar Personal.
    return { personal, personalRegistro: componerRegistroPersonal(recursos, entorno) };
  }

  // Carga los catálogos públicos de Personal y sondea las dos consultas a la
  // vez. Solo se ofrecen las que responden con una página válida; el resto se
  // omite sin error. Las consultas pertenecen a la carga en curso: se cancelan
  // con ella.
  async function cargarCatalogosPublicosPersonal({ consultar }) {
    let recursos;
    try {
      recursos = await cargarModuloConLimite(
        cargadoresInternos.personal_catalogos_publicos
          || CARGADORES_INTERNOS_PREDETERMINADOS.personal_catalogos_publicos,
        "personal_catalogos_publicos", limiteCargaModularMs, temporizadores,
      );
    } catch {
      return SIN_CATALOGOS_PUBLICOS;
    }
    const fetchImpl = fetchDelEntorno();
    if (!fetchImpl) return SIN_CATALOGOS_PUBLICOS;
    const sondeos = [
      ["rpt", () => recursos?.clienteRPT?.crearClienteHTTPRPTPublica({ fetchImpl }),
        (cliente, opciones) => cliente.listar({ vista: "categorias", q: "", limit: 1, offset: 0 }, opciones)],
      ["estructura", () => recursos?.clienteEstructura?.crearClienteHTTPEstructuraOrganizativaPublica({ fetchImpl }),
        (cliente, opciones) => cliente.obtener(opciones)],
    ];
    const resultados = await Promise.allSettled(sondeos.map(([, crear, sondear]) => {
      const cliente = crear();
      return consultar((opciones) => sondear(cliente, opciones), "consultar catálogos de Personal");
    }));
    const disponibles = sondeos.filter((_, indice) => resultados[indice].status === "fulfilled")
      .map(([clave]) => clave);
    return Object.freeze({ recursos, disponibles: Object.freeze(disponibles) });
  }

  async function cargarDietas({ exigirVigente }) {
    const recursos = await cargarModuloConLimite(
      cargadoresInternos.dietas || CARGADORES_INTERNOS_PREDETERMINADOS.dietas,
      "dietas", limiteCargaModularMs, temporizadores,
    );
    exigirVigente();
    // El recorrido interno consume clientes HTTP del mismo origen.
    const dietas = componerDietasInternas(recursos, entorno);
    if (dietas === undefined) throw new TypeError("vista de Dietas no disponible");
    return { dietas };
  }

  async function cargarDocumentos({ exigirVigente }) {
    const recursos = await cargarModuloConLimite(
      cargadoresInternos.documentos || CARGADORES_INTERNOS_PREDETERMINADOS.documentos,
      CLAVE_DOCUMENTOS, limiteCargaModularMs, temporizadores,
    );
    exigirVigente();
    const fetchImpl = fetchDelEntorno();
    if (typeof recursos?.vista?.montarVistaDocumentos !== "function"
      || typeof recursos?.cliente?.crearFuenteDocumentosHTTP !== "function" || !fetchImpl) {
      throw new TypeError("vista de Documentos no disponible");
    }
    return {
      documentos: Object.freeze({
        montar: ({ raiz, anunciar: avisar, registrarDesmontar, expedienteRef }) => recursos.vista.montarVistaDocumentos({
          raiz, anunciar: avisar, registrarDesmontar, expedienteRef,
          fuente: recursos.cliente.crearFuenteDocumentosHTTP({ fetchImpl }),
        }),
      }),
    };
  }

  const CARGAS_MODULOS = Object.freeze({
    [CLAVE_CONTRATACION_TEMPORAL]: cargarContratacionTemporal,
    cronos: cargarCronos,
    [CLAVE_PERSONAL]: cargarPersonal,
    dietas: cargarDietas,
    [CLAVE_DOCUMENTOS]: cargarDocumentos,
  });

  /**
   * Carga el catálogo y, después, todos los módulos autorizados en paralelo.
   * La composición se publica en cuanto llega el catálogo (módulos en estado
   * «cargando») y se actualiza al terminar cada módulo; `alCambiar(clave)`
   * avisa al shell para repintar. Un módulo lento o fallido no retrasa ni
   * oculta a los demás: queda «no_disponible» por sí solo.
   */
  async function cargarInterno({ alCambiar = null } = {}) {
    // Las vistas de Bolsa no dependen de la composición de módulos: una vista
    // de Bolsa ya montada (p. ej. al recargar con F5 en #bolsa/resumen) se
    // conserva y sus consultas en curso no se cancelan.
    if (!VISTAS_MODULO_BOLSA.has(vistaMontada)) retirarVistaMontada();
    cancelarCargaInterna();
    const carga = ++secuenciaCarga;
    composicion = null;
    catalogo = Object.freeze([]);
    catalogoOfrecido = catalogo;
    cargaEnCurso = true;
    const vigente = () => carga === secuenciaCarga;
    const exigirVigente = () => {
      if (!vigente()) throw errorCargaSustituida();
    };
    const consultar = (consulta, operacion) => {
      const controlador = new AbortController();
      controladoresCarga.add(controlador);
      return consultarConLimite(consulta, controlador, limiteCargaModularMs, temporizadores, operacion)
        .finally(() => controladoresCarga.delete(controlador));
    };
    const notificar = (clave) => {
      if (!vigente() || typeof alCambiar !== "function") return;
      try { alCambiar(clave); } catch { /* un fallo al pintar no detiene la carga */ }
    };

    let catalogoInterno;
    try {
      catalogoInterno = await consultar(({ signal }) => {
        if (cargarCatalogoInterno !== null) return cargarCatalogoInterno(signal);
        const fetchImpl = fetchDelEntorno() ?? globalThis.fetch;
        return cargarCatalogoModulosInterno((ruta, opciones) => fetchImpl(ruta, {
          ...opciones, signal,
        }));
      }, "cargar catálogo de módulos");
    } catch (error) {
      exigirVigente();
      cargaEnCurso = false;
      throw error;
    }
    exigirVigente();
    const conVista = catalogoInterno
      .filter((modulo) => CLAVES_MODULOS_CON_VISTA_PORTAL.includes(modulo.clave));
    catalogo = conVista.length === catalogoInterno.length ? catalogoInterno : Object.freeze(conVista);
    const ofrecidos = catalogo.filter((modulo) => !CLAVES_SIN_ENTRADA_PORTAL.includes(modulo.clave));
    catalogoOfrecido = ofrecidos.length === catalogo.length ? catalogo : Object.freeze(ofrecidos);

    const partes = {
      contratacionTemporal: undefined,
      cronos: undefined,
      dietas: undefined,
      documentos: undefined,
      personal: undefined,
      personalRegistro: undefined,
    };
    const autorizados = CLAVES_CARGA_PORTAL
      .filter((clave) => catalogo.some((modulo) => modulo.clave === clave));
    // Los módulos sin entrada en el portal no se cargan al arrancar: ni su
    // código ni sus consultas. Quedan «diferidos» hasta que se pide una de sus
    // vistas por su URL directa (`prepararVista`).
    const cargables = autorizados.filter((clave) => !modulosDiferidos.includes(clave));
    const estados = Object.fromEntries(CLAVES_CARGA_PORTAL
      .map((clave) => [clave, cargables.includes(clave) ? "cargando"
        : (autorizados.includes(clave) ? ESTADO_DIFERIDO : "no_disponible")]));
    const publicar = () => {
      composicion = Object.freeze({ ...partes, estadosModulos: Object.freeze({ ...estados }) });
    };
    const cargarModulo = async (clave) => {
      let resultado;
      try {
        resultado = await CARGAS_MODULOS[clave]({ consultar, exigirVigente, notificar: () => notificar(clave) });
      } catch {
        resultado = undefined;
      }
      if (!vigente()) return;
      if (resultado) Object.assign(partes, resultado);
      estados[clave] = resultado ? "disponible" : "no_disponible";
      publicar();
      notificar(clave);
    };
    cargaDiferida = (clave) => {
      if (!vigente() || estados[clave] !== ESTADO_DIFERIDO) return null;
      estados[clave] = "cargando";
      publicar();
      return cargarModulo(clave);
    };
    publicar();
    notificar("catalogo");

    await Promise.allSettled(cargables.map(cargarModulo));
    exigirVigente();
    cargaEnCurso = false;
  }

  /**
   * Una vista de un módulo diferido (sin entrada en el portal) se ha pedido:
   * empieza a cargar su módulo. Devuelve la promesa de esa carga, o null si no
   * había nada que cargar (módulo ya cargado, en curso, no autorizado o sin
   * catálogo todavía).
   */
  function prepararVista(vista) {
    const modulo = moduloDeVistaPortal(vista);
    if (!modulosDiferidos.includes(modulo) || typeof cargaDiferida !== "function") return null;
    return cargaDiferida(modulo);
  }

  // Estado de carga de un módulo del catálogo: «cargando», «disponible»,
  // «no_disponible» o "" si no hay composición.
  function estadoCargaModulo(clave) {
    return composicion?.estadosModulos?.[clave] || "";
  }

  // El catálogo aún no ha llegado a esta carga.
  function catalogoPendiente() {
    return cargaEnCurso && composicion === null;
  }

  /**
   * Inicio no puede decidir todavía entre el perfil RRHH y el de empleado:
   * contratación temporal está autorizada y aún carga (o no ha llegado el
   * catálogo). Mientras tanto se pinta un Inicio neutro.
   */
  function inicioPendiente() {
    if (catalogoPendiente()) return true;
    return catalogo.some((modulo) => modulo.clave === CLAVE_CONTRATACION_TEMPORAL)
      && estadoCargaModulo(CLAVE_CONTRATACION_TEMPORAL) === "cargando";
  }

  /**
   * Una vista gestionada aún no disponible cuyo módulo sigue cargando: el shell
   * pinta «Comprobando» en lugar de «no disponible».
   */
  function vistaPendiente(vista) {
    if (!vistaGestionada(vista) || vistaDisponible(vista)) return false;
    if (catalogoPendiente()) return true;
    const modulo = moduloDeVistaPortal(vista);
    if (["cargando", ESTADO_DIFERIDO].includes(estadoCargaModulo(modulo))) return true;
    // El registro RRHH depende también de conocer el perfil.
    return vista === "personal-registro" && estadoCargaModulo(CLAVE_CONTRATACION_TEMPORAL) === "cargando";
  }

  function obtenerCatalogo() {
    return catalogoOfrecido;
  }

  function vistaDisponible(vista) {
    if (VISTAS_MODULO_BOLSA.has(vista)) {
      return montajeBolsa !== null && montajeBolsa.disponible(vista) === true;
    }
    if (vista === "contratacion-temporal") {
      return composicion?.contratacionTemporal !== undefined;
    }
    if (vista === "cronos") return composicion?.cronos !== undefined;
    if (vista === "cronos-permisos") return typeof composicion?.cronos?.montarPermisos === "function";
    if (vista === "cronos-avisos") return typeof composicion?.cronos?.montarAvisos === "function";
    if (vista === "cronos-bandeja") return typeof composicion?.cronos?.montarBandeja === "function";
    if (vista === "cronos-notificaciones") return typeof composicion?.cronos?.montarNotificaciones === "function";
    if (vista === "cronos-bandeja-notificaciones") return typeof composicion?.cronos?.montarBandejaNotificaciones === "function";
    if (vista === "dietas") return composicion?.dietas !== undefined;
    if (vista === VISTA_DOCUMENTOS_EXPEDIENTE) return composicion?.documentos !== undefined;
    if (vista === "personal") return composicion?.personal !== undefined;
    // Oferta de interfaz para el perfil RRHH; cada lectura la autoriza V3.
    if (vista === "personal-registro") return composicion?.personal !== undefined
      && composicion?.personalRegistro !== undefined && registroPersonalServido !== false && esPerfilRRHH();
    return false;
  }

  // El shell consulta el montaje real antes de intentar entrar en una vista.
  function vistaGestionada(vista) {
    return (montajeBolsa !== null && VISTAS_MODULO_BOLSA.has(vista))
      || VISTAS_MODULOS_CONECTADOS.has(vista);
  }

  function resolverAcceso(clave, bolsaDisponible = true) {
    if (clave === "bolsa") {
      const autorizada = catalogo.some((modulo) => modulo.clave === "bolsa");
      const acceso = bolsaDisponible !== null && typeof bolsaDisponible === "object"
        ? bolsaDisponible
        : { disponible: bolsaDisponible === true, vista: "resumen" };
      return Object.freeze({
        ...acceso,
        disponible: acceso.disponible === true && autorizada,
        vista: acceso.disponible === true && autorizada ? acceso.vista : "",
        ...(!autorizada ? { estado: "no_disponible" } : {}),
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
    if (clave === CLAVE_DOCUMENTOS && vistaDisponible(VISTA_DOCUMENTOS_EXPEDIENTE)) {
      return Object.freeze({ disponible: true, vista: VISTA_DOCUMENTOS_EXPEDIENTE });
    }
    if (clave === CLAVE_PERSONAL && vistaDisponible("personal")) {
      return Object.freeze({ disponible: true, vista: "personal",
        etiqueta: traducir("personal_catalogo_profesional") });
    }
    // Mientras su carga sigue en curso, la tarjeta y el menú dicen «Comprobando».
    if (composicion?.estadosModulos?.[clave] === "cargando") {
      return Object.freeze({ disponible: false, vista: "", estado: "cargando" });
    }
    if (clave === CLAVE_CONTRATACION_TEMPORAL
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
    if (CLAVES_MODULOS_VEC_REGISTRADOS.includes(clave)) {
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
      ? catalogoOfrecido
      : catalogoOfrecido.filter((modulo) => modulo.clave === moduloActivo);
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
    if (reutilizarElaboracion(vista, raiz, opciones)) return true;
    retirarVistaMontada();
    const montaje = ++secuenciaMontaje;
    if (VISTAS_MODULO_BOLSA.has(vista)) {
      // Una vista de Bolsa consta montada desde ya: una carga del catálogo que
      // empiece mientras se monta (F5 en #bolsa/resumen) no la retira.
      vistaMontada = vista;
      raizMontada = raiz;
      referenciaElaboracionMontada = vista === "elaboracion" ? opciones?.referencia || "" : "";
    }
    raiz.innerHTML = `<section class="panel"><div class="cuerpo-panel" role="status">${escaparHTML(traducir("estado_modulo_comprobando"))}</div></section>`;

    if (VISTAS_MODULO_BOLSA.has(vista)) {
      let resultado;
      try {
        resultado = await montajeBolsa.montar({ vista, raiz, opciones, anunciar });
      } catch (error) {
        if (montaje === secuenciaMontaje) {
          vistaMontada = "";
          raizMontada = null;
          referenciaElaboracionMontada = "";
        }
        throw error;
      }
      const limpiar = typeof resultado === "function" ? resultado : resultado?.desmontar;
      if (montaje !== secuenciaMontaje) {
        if (typeof limpiar === "function") limpiar();
        return false;
      }
      desmontarVista = typeof limpiar === "function" ? limpiar : null;
      return true;
    }

    if (vista === "contratacion-temporal") {
      // Los catálogos del alta ya están en camino desde la carga del módulo.
      await composicion.contratacionTemporal.esperarAlta?.();
      if (montaje !== secuenciaMontaje) return false;
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

    if (vista === "cronos" || SUBVISTAS_CRONOS.has(vista)) {
      if (typeof composicion.cronos.montar === "function") {
        raiz.replaceChildren();
        const t = composicion.cronos.traducir;
        const navegacion = raiz.ownerDocument.createElement("nav");
        navegacion.className = "panel";
        navegacion.setAttribute("aria-label", t("navegacion_etiqueta"));
        navegacion.dataset.cronosSubvistas = "";
        navegacion.innerHTML = `<div class="cabecera-panel"><h3>${escaparHTML(t("navegacion_etiqueta"))}</h3></div>
          <div class="cuerpo-panel">
            <button type="button" class="boton-secundario" data-vista="cronos"${vista === "cronos" ? ' aria-current="page"' : ""}>${escaparHTML(t("jornada_titulo"))}</button>
            <button type="button" class="boton-secundario" data-vista="cronos-permisos"${vista === "cronos-permisos" ? ' aria-current="page"' : ""}>${escaparHTML(t("navegacion_permisos"))}</button>
            ${typeof composicion.cronos.montarAvisos === "function" ? `<button type="button" class="boton-secundario" data-vista="cronos-avisos"${vista === "cronos-avisos" ? ' aria-current="page"' : ""}>${escaparHTML(composicion.cronos.etiquetas.avisos)}</button>
            <button type="button" class="boton-secundario" data-vista="cronos-bandeja"${vista === "cronos-bandeja" ? ' aria-current="page"' : ""}>${escaparHTML(composicion.cronos.etiquetas.bandeja)}</button>` : ""}
            ${typeof composicion.cronos.montarNotificaciones === "function" ? `<button type="button" class="boton-secundario" data-vista="cronos-notificaciones"${vista === "cronos-notificaciones" ? ' aria-current="page"' : ""}>${escaparHTML(composicion.cronos.etiquetas.notificaciones)}</button>
            <button type="button" class="boton-secundario" data-vista="cronos-bandeja-notificaciones"${vista === "cronos-bandeja-notificaciones" ? ' aria-current="page"' : ""}>${escaparHTML(composicion.cronos.etiquetas.bandejaNotificaciones)}</button>` : ""}
          </div>`;
        raiz.append(navegacion);
        desmontarVista = () => navegacion?.remove();
        const montarCronos = {
          "cronos-permisos": composicion.cronos.montarPermisos,
          "cronos-avisos": composicion.cronos.montarAvisos,
          "cronos-bandeja": composicion.cronos.montarBandeja,
          "cronos-notificaciones": composicion.cronos.montarNotificaciones,
          "cronos-bandeja-notificaciones": composicion.cronos.montarBandejaNotificaciones,
        }[vista] ?? composicion.cronos.montar;
        const modulo = await montarCronos({ raiz, anunciar,
          registrarDesmontar: (limpiar) => {
            const retirar = () => { limpiar(); navegacion?.remove(); };
            if (montaje !== secuenciaMontaje) { retirar(); return; }
            desmontarVista = retirar;
          },
        });
        if (montaje !== secuenciaMontaje) { modulo.desmontar(); navegacion?.remove(); return false; }
        desmontarVista = () => { modulo.desmontar(); navegacion?.remove(); };
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

    if (vista === VISTA_DOCUMENTOS_EXPEDIENTE) {
      raiz.replaceChildren();
      const moduloDocumentos = composicion.documentos.montar({
        raiz, anunciar,
        expedienteRef: typeof opciones?.expedienteRef === "string" ? opciones.expedienteRef : "",
        registrarDesmontar: (limpiar) => {
          if (typeof limpiar !== "function") throw new TypeError("limpieza de Documentos no válida");
          if (montaje !== secuenciaMontaje) { limpiar(); return; }
          desmontarVista = limpiar;
        },
      });
      if (montaje !== secuenciaMontaje) { moduloDocumentos.desmontar(); return false; }
      desmontarVista = moduloDocumentos.desmontar;
      return true;
    }

    if (vista === "personal-registro") {
      if (registroPersonalServido === null) {
        const servido = await sondearRegistroPersonal();
        if (montaje !== secuenciaMontaje) return false;
        if (servido !== null) registroPersonalServido = servido;
      }
      if (registroPersonalServido !== true) {
        // Esta superficie no sirve el registro o no autoriza su lectura: no se
        // vuelve a ofrecer en la sesión.
        raiz.innerHTML = `<section class="panel"><div class="cuerpo-panel vacio-controlado" role="status"><p><strong>${escaparHTML(traducir("estado_modulo_no_disponible_titulo"))}</strong></p></div></section>`;
        return false;
      }
      raiz.replaceChildren();
      const navegacion = navegacionPersonal(raiz, vista);
      const registro = composicion.personalRegistro.montar({ raiz, anunciar,
        registrarDesmontar: (limpiar) => {
          const retirar = () => { limpiar(); navegacion?.remove(); };
          if (montaje !== secuenciaMontaje) { retirar(); return; }
          desmontarVista = retirar;
        },
      });
      if (montaje !== secuenciaMontaje) { registro.desmontar(); navegacion?.remove(); return false; }
      desmontarVista = () => { registro.desmontar(); navegacion?.remove(); };
      return true;
    }

    if (vista === "personal") {
      raiz.replaceChildren();
      const navegacionFicha = navegacionPersonal(raiz, vista);
      // La ficha consulta sus fuentes antes de pintarse: mientras tanto se ve
      // el mismo estado de carga común del shell y salir de la vista cancela
      // la consulta en curso.
      const controlador = new AbortController();
      const cargando = panelComprobando(raiz);
      raiz.append(cargando);
      desmontarVista = () => { controlador.abort(); cargando.remove(); navegacionFicha?.remove(); };
      let moduloPersonal;
      try {
        moduloPersonal = await composicion.personal.montar({
          raiz,
          cliente: composicion.personal.cliente,
          anunciar,
          signal: controlador.signal,
          registrarDesmontar: (limpiar) => {
            if (typeof limpiar !== "function") throw new TypeError("limpieza de Personal no válida");
            const retirar = () => { controlador.abort(); limpiar(); navegacionFicha?.remove(); };
            if (montaje !== secuenciaMontaje) { retirar(); return; }
            desmontarVista = retirar;
          },
        });
      } catch (error) {
        cargando.remove();
        if (montaje !== secuenciaMontaje || controlador.signal.aborted) return false;
        throw error;
      }
      cargando.remove();
      if (montaje !== secuenciaMontaje) {
        moduloPersonal.desmontar();
        navegacionFicha?.remove();
        return false;
      }
      desmontarVista = () => { moduloPersonal.desmontar(); navegacionFicha?.remove(); };
      return true;
    }

    raiz.replaceChildren();
    const moduloDietas = await composicion.dietas.montar({
      raiz,
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

  // Estado de carga común de las vistas del shell, como nodo: el mismo panel
  // «Comprobando» con el que montarVista abre cualquier vista.
  function panelComprobando(raiz) {
    const documento = raiz.ownerDocument;
    const seccion = documento.createElement("section");
    seccion.className = "panel";
    seccion.dataset.portalCargaVista = "";
    const cuerpo = documento.createElement("div");
    cuerpo.className = "cuerpo-panel";
    cuerpo.setAttribute("role", "status");
    cuerpo.textContent = traducir("estado_modulo_comprobando");
    seccion.append(cuerpo);
    return seccion;
  }

  // Consulta mínima autorizada del registro. Devuelve true si responde, false
  // si falla y null si se interrumpe al cambiar de vista.
  async function sondearRegistroPersonal() {
    const sondear = composicion?.personalRegistro?.sondear;
    if (typeof sondear !== "function") return false;
    const controlador = new AbortController();
    sondaRegistro = controlador;
    try {
      await consultarConLimite((opciones) => sondear(opciones), controlador,
        limiteCargaModularMs, temporizadores, "consultar registro de Personal");
      return true;
    } catch {
      return sondaRegistro === controlador ? false : null;
    } finally {
      if (sondaRegistro === controlador) sondaRegistro = null;
    }
  }

  // Subnavegación de Personal: solo si el registro RRHH se ofrece.
  function navegacionPersonal(raiz, vista) {
    if (!vistaDisponible("personal-registro")) return null;
    const t = composicion.personalRegistro.traducir;
    const navegacion = raiz.ownerDocument.createElement("nav");
    navegacion.className = "panel";
    navegacion.setAttribute("aria-label", t("registro_b2_navegacion"));
    navegacion.dataset.personalSubvistas = "";
    navegacion.innerHTML = `<div class="cuerpo-panel">
        <button type="button" class="boton-secundario" data-vista="personal"${vista === "personal" ? ' aria-current="page"' : ""}>${escaparHTML(t("registro_b2_mi_ficha"))}</button>
        <button type="button" class="boton-secundario" data-vista="personal-registro"${vista === "personal-registro" ? ' aria-current="page"' : ""}>${escaparHTML(t("registro_b2_titulo"))}</button>
      </div>`;
    raiz.append(navegacion);
    return navegacion;
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
    desmontarVistaActual,
    prepararVista,
    esPerfilRRHH,
    inicioPendiente,
    montarVista,
    obtenerTramitesInicio,
    obtenerCatalogo,
    obtenerMetricasCuadro,
    renderizarNavegacion,
    resolverAcceso,
    retirarVistaMontada,
    vistaGestionada,
    vistaDisponible,
    vistaPendiente,
  });
}
