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
} from "./portal-catalogo-modulos.js?v=20261007-pantallas-textos-final-v1";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL, traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import {
  componerCronosInterno,
  componerDietasInternas,
  componerPersonalVisible,
  componerRegistroPersonal,
} from "./portal-composicion-empleado.js?v=20261008-alta-rpt-circular-v4";
import { VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js?v=20261007-pantallas-textos-final-v1";
import { cargarTextos } from "../comun/textos.js";
import { INDICE_IDIOMAS } from "../comun/idioma.js";
import {
  CLAVES_CARGA_MODULAR,
  LIMITE_CARGA_MODULAR_MS,
  cargarModuloConLimite,
  consultarConLimite,
} from "./portal-modulos-carga.js?v=20260926-integracion-bolsa-ct-v1";

const CLAVE_CONTRATACION_TEMPORAL = "contratacion_temporal";
export function localizacionAdmitida(locale, indice = INDICE_IDIOMAS) {
  if (typeof locale !== "string" || !indice?.idiomas?.some(({ localizacion }) => localizacion === locale)) return false;
  try { return Intl.DateTimeFormat.supportedLocalesOf([locale]).length === 1; }
  catch { return false; }
}
const PERFILES_CT_MENU = new Set(["tecnico_rrhh", "intervencion"]);
// El circuito se consulta al cargar CT. Un fallo transitorio de un catálogo no
// debe quedar congelado en el módulo ni retrasar el arranque del portal.
const cargarFasesCircuitoPredeterminado = async (idioma) =>
  (await cargarTextos("contratacion-temporal-circuito-rrhh", { idioma })).seccion("fases");
const SIN_CATALOGOS_PUBLICOS = Object.freeze({ recursos: Object.freeze({}), disponibles: Object.freeze([]), estadoRPT: "ausente" });
function estadoSondaRPT(resultado) {
  if (resultado?.ok === true) return "disponible";
  if (resultado?.codigo === "montaje_ausente") return "ausente";
  if (resultado?.codigo === "estado_no_valido" && resultado.estado === 404) return "ausente";
  if (resultado?.codigo === "estado_no_valido" && [401, 403].includes(resultado.estado)) return "denegado";
  return "incidencia";
}
const CLAVE_PERSONAL = "personal";
const CLAVE_DOCUMENTOS = "documentos";
// «documentos» es también una sección de Bolsa; la vista del servicio común
// de Documentos usa un nombre propio para no confundirse con ella.
export const VISTA_DOCUMENTOS_EXPEDIENTE = "documentos-expediente";
export const VISTA_PLANTILLAS_RRHH = "plantillas-rrhh";
export const VISTA_CATEGORIAS_RPT = "categorias-rpt";
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
      import("./modulos/cronos/vista-saldo-conectado.js?v=20261001-cronos-saldo-explicado-v1"),
      import("./modulos/cronos/vista-remoto.js?v=20261001-cronos-grafo-bandeja-v5"),
      import("./modulos/cronos/vista-movimientos-conectado.js?v=20261001-cronos-movimientos-consulta-v1"),
      import("./modulos/cronos/vista-movimientos-propios.js?v=20261007-u-dietas-catalogo-v1"),
      import("./modulos/cronos/vista-permisos-propios.js?v=20261007-pantallas-textos-final-v1"),
      import("./modulos/cronos/cliente-saldo-http.js"),
      import("./modulos/cronos/cliente-remoto-http.js"),
      import("./modulos/cronos/cliente-solicitudes-http.js"),
      import("./modulos/cronos/i18n.js?v=20260929-i18n-textos-v1"),
      import("./modulos/cronos/vista-bandeja-permisos.js?v=20261001-f-reconciliacion-321-v1"),
      import("./modulos/cronos/vista-avisos-propios.js?v=20261001-cronos-avisos-confirmados-v1"),
      import("./modulos/cronos/cliente-resolucion-http.js"),
      import("./modulos/cronos/i18n-resolucion.js?v=20261001-cronos-grafo-bandeja-v5"),
      import("./modulos/cronos/vista-notificaciones-propias.js?v=20261001-cronos-c9-historial-v2"),
      import("./modulos/cronos/vista-bandeja-notificaciones.js?v=20261001-cronos-c9-recuperacion-v3"),
      import("./modulos/cronos/cliente-notificaciones-http.js"),
      import("./modulos/cronos/i18n-notificaciones.js"),
    ]);
    return Object.freeze({ saldo, remoto, movimientos, movimientosPropios, permisosPropios,
      clienteSaldo, clienteRemoto, clienteSolicitudes, i18n, bandejaPermisos, avisosPropios, clienteResolucion, i18nResolucion,
      notificacionesPropias, bandejaNotificaciones, clienteNotificaciones, i18nNotificaciones });
  },
  contratacion_temporal: async () => {
    const [contrato, cliente] = await Promise.all([
      import("./modulos/contratacion-temporal/contrato.js?v=20261008-alta-corte-v1"),
      import("./modulos/contratacion-temporal/cliente-http.js?v=20261008-analisis-confirmado-v2"),

    ]);
    let completos;
    const cargarCompleto = () => {
      completos ??= Promise.all([
        import("./modulos/contratacion-temporal/presentador-expedientes.js?v=20261008-analisis-confirmado-v2"),
        import("./modulos/contratacion-temporal/adaptador-http-expedientes.js?v=20261008-alta-corte-v1"),
        import("./modulos/contratacion-temporal/cliente-http-incorporacion-personal-b2.js?v=20261008-alta-corte-v1"),
      ]).then(([presentador, adaptador, incorporacionB2]) => ({ presentador, adaptador, incorporacionB2 }))
        .catch((error) => { completos = null; throw error; });
      return completos;
    };
    const cargarCuadroLigero = () => import("./modulos/contratacion-temporal/vista-cuadro-ligera.js?v=20261008-ct-centros-v1");
    // La vista (unos 130 ficheros) solo se carga al abrir CT. Importarla tras
    // los consumidores previos evita leer el catálogo de fases sin iniciar.
    // Auditoría comparte el cargador de textos con CT.
    const cargarVista = async () => {
      const vista = await import("./modulos/contratacion-temporal/vista-expedientes.js?v=20261008-analisis-confirmado-v2");

      const [auditoriaVista, auditoriaCliente] = await Promise.all([
        import("./modulos/auditoria/vista.js?v=20261007-pantallas-textos-final-v1"),
        import("./modulos/auditoria/cliente-http.js?v=20261007-auditoria-disponibilidad-v1"),
      ]);
      return Object.freeze({ vista, auditoriaVista, auditoriaCliente });
    };
    return Object.freeze({ contrato, cliente, cargarCuadroLigero, cargarCompleto, cargarVista });
  },
  personal: async () => {
    const [contrato, cliente, vista, ficha, registro, clienteRegistro, clienteCatalogosRegistro, i18n, clienteFichaPropia, contacto] = await Promise.all([
      import("./modulos/personal/contrato.js?v=20260920-personal-catalogo-v1"),
      import("./modulos/personal/cliente-http-categorias.js?v=20260925-portal-integrado-v1"),
      import("./modulos/personal/vista.js?v=20261008-alta-rpt-circular-v4"),
      import("./modulos/personal/vista-ficha-integral.js?v=20261008-alta-rpt-circular-v4"),
      import("./modulos/personal/registro-b2.js?v=20261008-alta-rpt-circular-v4"),
      import("./modulos/personal/registro-b2-cliente.js?v=20261002-b-base-401-acumulada-v3"),
      import("./modulos/personal/registro-b2-catalogos-cliente.js?v=20260925-b2-mtls-v1"),
      import("./modulos/personal/i18n.js?v=20261008-alta-rpt-circular-v4"),
      import("./modulos/personal/cliente-http-ficha-propia.js?v=20261004-personal-relaciones-v1"),
      import("./modulos/personal/vista-contacto-propio.js?v=20261007-pantallas-textos-final-v1"),
    ]);
    await i18n.prepararTextosPersonal();
    return Object.freeze({ contrato, cliente, vista, clienteCategorias: cliente, vistaCategorias: vista,
      ficha, registro, clienteRegistro, clienteCatalogosRegistro, i18n, clienteFichaPropia, contacto });
  },
  // Catálogos públicos de Personal (RPT publicada y estructura de referencia).
  // Van en todos los paquetes web (el import nunca da 404); solo se ofrecen si
  // el servidor responde a su consulta con una página válida.
  personal_catalogos_publicos: async () => {
    const [clienteRPT, vistaRPT, clienteEstructura, vistaEstructura, i18n] = await Promise.all([
      import("./modulos/personal/cliente-http-rpt-publica.js?v=20261008-alta-rpt-circular-v4"),
      import("./modulos/personal/vista-rpt-publica.js?v=20261008-alta-rpt-circular-v4"),
      import("./modulos/personal/cliente-http-estructura-organizativa-publica.js?v=20260925-portal-integrado-v1"),
      import("./modulos/personal/vista-estructura-organizativa-publica.js?v=20261008-alta-rpt-circular-v4"),
      import("./modulos/personal/i18n.js?v=20261008-alta-rpt-circular-v4"),
    ]);
    await i18n.prepararTextosPersonal();
    return Object.freeze({ clienteRPT, vistaRPT, clienteEstructura, vistaEstructura });
  },
  dietas: async () => {
    const [contrato, recorridos, clienteBorradores, clienteAsignacion, calculador, mapa, clienteCircuito, clienteRectificacion] = await Promise.all([
      import("./modulos/dietas/contrato.js"),
      import("./modulos/dietas/vista-recorridos.js?v=20261007-u-dietas-catalogo-v1"),
      import("./modulos/dietas/cliente-borradores-http.js?v=20260925-d5d6-v2"),
      import("./modulos/dietas/cliente-asignacion-http.js?v=20260925-d5d6-v1"),
      import("./modulos/dietas/calculador-rutas-http.js?v=20260925-d5d6-v1"),
      import("./modulos/dietas/mapa-ruta.js?v=20261007-u-dietas-catalogo-v1"),
      import("./modulos/dietas/cliente-circuito-http.js?v=20261001-dietas-decision-v1"),
      import("./modulos/dietas/cliente-rectificacion-http.js?v=20261002-codexe-d7c-web-v1"),
    ]);
    return Object.freeze({ contrato, recorridos, clienteBorradores, clienteAsignacion, calculador, mapa, clienteCircuito, clienteRectificacion });
  },
  // Consulta del expediente documental (RRHH). El servidor sólo publica el
  // módulo cuando su montaje está compuesto; cada consulta la autoriza V3.
  documentos: async () => {
    const [vista, cliente] = await Promise.all([
      import("./modulos/documentos/vista.js?v=20261007-pantallas-textos-final-v1"),
      import("./modulos/documentos/cliente-http.js?v=20261007-pantallas-textos-final-v1"),
    ]);
    return Object.freeze({ vista, cliente });
  },
});

// Organización nombra sus centros «centro-<código>»; Contratación los guarda
// también como «centro:rpt:<CÓDIGO>». Se ofrecen ambas formas.
export function centrosDeOrganizacion(unidades) {
  const centros = new Map();
  for (const unidad of Array.isArray(unidades) ? unidades : []) {
    if (unidad?.tipo !== "centro" || typeof unidad.clave !== "string" || typeof unidad.etiqueta !== "string") continue;
    centros.set(unidad.clave, unidad.etiqueta);
    const codigo = /^centro-([a-z0-9]{1,12})$/iu.exec(unidad.clave)?.[1];
    if (codigo) centros.set(`centro:rpt:${codigo.toUpperCase()}`, unidad.etiqueta);
  }
  return centros;
}

function nombreCentroEn(mapa, referencia) {
  if (!(mapa instanceof Map)) return undefined;
  const exacto = mapa.get(referencia);
  if (exacto !== undefined) return exacto;
  const desdeOrganizacion = /^centro-([a-z0-9]{1,12})$/iu.exec(referencia ?? "");
  if (desdeOrganizacion) return mapa.get(`centro:rpt:${desdeOrganizacion[1].toUpperCase()}`);
  const desdeRPT = /^centro:rpt:([a-z0-9]{1,12})$/iu.exec(referencia ?? "");
  return desdeRPT ? mapa.get(`centro-${desdeRPT[1].toLowerCase()}`)
    ?? mapa.get(`centro-${desdeRPT[1].toUpperCase()}`) : undefined;
}

export const VISTAS_MODULOS_PERSONALES = Object.freeze(new Set(["cronos", "cronos-permisos", "cronos-avisos", "cronos-bandeja",
  "cronos-notificaciones", "cronos-bandeja-notificaciones", "dietas", "personal", "personal-registro"]));
// Navegación propia: no incluye vistas de gestión ni acredita permisos.
export const VISTAS_AUTOSERVICIO_EMPLEADO = Object.freeze(new Set([
  "personal", "cronos", "cronos-permisos", "cronos-avisos", "cronos-notificaciones", "dietas", "mis-tramites",
]));
const SUBVISTAS_CRONOS = Object.freeze(new Set(["cronos-permisos", "cronos-avisos", "cronos-bandeja", "cronos-notificaciones", "cronos-bandeja-notificaciones"]));
const VISTAS_MODULO_BOLSA = Object.freeze(new Set(VISTAS_INTERNAS_BOLSA));
export const VISTAS_MODULOS_CONECTADOS = Object.freeze(new Set([
  "contratacion-temporal", VISTA_PLANTILLAS_RRHH, VISTA_CATEGORIAS_RPT, VISTA_DOCUMENTOS_EXPEDIENTE, ...VISTAS_MODULOS_PERSONALES, "mis-tramites",
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
  if (vista === "portal" || vista === "mis-tramites") return "portal";
  if (vista === "contratacion-temporal" || vista === VISTA_PLANTILLAS_RRHH || vista === VISTA_CATEGORIAS_RPT) return "contratacion_temporal";
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
  if (vista === "mis-tramites") return "#mis-tramites";
  if (vista === "contratacion-temporal") return "#contratacion-temporal";
  if (vista === "ofertas-sae") return "#ofertas-sae";
  if (vista === VISTA_PLANTILLAS_RRHH) return "#contratacion-temporal/plantillas-rrhh";
  if (vista === VISTA_CATEGORIAS_RPT) return "#contratacion-temporal/categorias-rpt";
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
  locale = LOCALIZACION_PORTAL,
  cargarCatalogoInterno = null,
  cargadoresInternos = CARGADORES_INTERNOS_PREDETERMINADOS,
  cargarTramitesPropios = async () => {
    const [fuente, vista] = await Promise.all([
      import("./modulos/solicitudes/fuente-tramites-propios.js?v=20261004-b-tramites-devoluciones-v2"),
      import("./modulos/solicitudes/vista-tramites-propios.js?v=20261007-pantallas-textos-final-v1"),
    ]);
    return { fuente, vista };
  },
  consultarSesion = null,
  cargarFasesCircuito = cargarFasesCircuitoPredeterminado,
  limiteCargaModularMs = LIMITE_CARGA_MODULAR_MS,
  temporizadores = globalThis,
  // Módulos que no se cargan al arrancar sino al pedir una de sus vistas.
  modulosDiferidos = CLAVES_DIFERIDAS_PORTAL,
} = {}) {
  if (typeof escaparHTML !== "function" || typeof anunciar !== "function"
    || typeof confirmarOperacion !== "function" || typeof traducir !== "function"
    || (cargarCatalogoInterno !== null && typeof cargarCatalogoInterno !== "function")
    || (consultarSesion !== null && typeof consultarSesion !== "function")
    || typeof cargarFasesCircuito !== "function"
    || (montajeBolsa !== null && (typeof montajeBolsa?.montar !== "function"
      || typeof montajeBolsa?.disponible !== "function"))
    || typeof cargadoresInternos?.contratacion_temporal !== "function" || typeof cargarTramitesPropios !== "function"
    || !localizacionAdmitida(locale)
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
  let ctRecuperableEnMenu = false;
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
  let recursosTramites = null;
  let cargaTramites = null;
  let estadoTramites = ESTADO_DIFERIDO;
  let notificarTramites = () => {};
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
  async function cargarContratacionTemporal({ consultar, exigirVigente, notificar = () => {}, recursosRecibidos = null }) {
    const recursos = recursosRecibidos ?? await cargarModuloConLimite(
      cargadoresInternos.contratacion_temporal,
      CLAVE_CONTRATACION_TEMPORAL,
      limiteCargaModularMs,
      temporizadores,
    ).catch(() => { throw Object.assign(new Error("recursos CT no disponibles"), { codigo: "carga_recursos_ct" }); });
    exigirVigente();
    if (typeof recursos.cargarCuadroLigero === "function") {
      const cliente = recursos.cliente.crearClienteHTTPContratacionTemporal({
        fetchImpl: fetchDelEntorno(), HeadersImpl: entorno.Headers,
      });
      const idiomaLista = INDICE_IDIOMAS.idiomas.find(({ localizacion }) => localizacion === locale).codigo;
      let promesaCatalogoNombres = null;
      let promesaOrganizacionNombres = null;
      let etiquetasLista = { centrosCatalogo: new Map(), centrosOrganizacion: new Map(),
        categorias: new Map(), textos: null };
      const denegacionEstable = (error) => [401, 403, 404].includes(error?.estado ?? error?.status);
      const nombresCatalogo = () => {
        if (typeof cliente.obtenerCatalogosAlta !== "function") return Promise.resolve(null);
        promesaCatalogoNombres ??= consultar(async (opciones) => {
          try { return { respuesta: await cliente.obtenerCatalogosAlta(opciones) }; }
          catch (error) { return { error }; }
        }, "consultar catálogo CT").then(({ respuesta, error }) => {
          if (error) throw error;
          const catalogo = recursos.contrato.validarCatalogosAlta(respuesta);
          return {
            centros: new Map(catalogo.centros.map(({ referencia, etiqueta }) => [referencia, etiqueta])),
            categorias: new Map(catalogo.categorias.map(({ referencia, etiqueta }) => [referencia, etiqueta])),
          };
        }).catch((error) => {
          if (!denegacionEstable(error)) promesaCatalogoNombres = null;
          return null;
        });
        return promesaCatalogoNombres;
      };
      const nombresOrganizacion = () => {
        promesaOrganizacionNombres ??= import("./modulos/personal/cliente-http-estructura-organizativa-publica.js?v=20260925-portal-integrado-v1")
          .then((modulo) => modulo.crearClienteHTTPEstructuraOrganizativaPublica({
            fetchImpl: fetchDelEntorno() ?? globalThis.fetch,
          }).obtener())
          .then((estructura) => {
            if (!Array.isArray(estructura?.unidades)) throw new TypeError("estructura de centros no válida");
            return centrosDeOrganizacion(estructura.unidades);
          }).catch((error) => {
            if (!denegacionEstable(error)) promesaOrganizacionNombres = null;
            return null;
          });
        return promesaOrganizacionNombres;
      };
      const prepararNombresLista = async (paginaPromesa) => {
        // Las etiquetas sólo se consultan después de una página autorizada.
        // Una denegación del cuadro no inicia consultas auxiliares.
        const pagina = await paginaPromesa;
        const [catalogo, textos] = await Promise.all([
          nombresCatalogo(), cargarTextos("contratacion-temporal-ficha-lista", { idioma: idiomaLista }),
        ]);
        const faltaCentro = Array.isArray(pagina?.expedientes) && pagina.expedientes.some(
          ({ centro_ref: referencia }) => !nombreCentroEn(catalogo?.centros, referencia));
        const organizacion = faltaCentro ? await nombresOrganizacion() : null;
        etiquetasLista = {
          centrosCatalogo: catalogo?.centros ?? new Map(),
          centrosOrganizacion: organizacion ?? new Map(),
          categorias: catalogo?.categorias ?? new Map(),
          textos,
        };
        return pagina;
      };
      const nombreCentro = (referencia) => nombreCentroEn(etiquetasLista.centrosCatalogo, referencia)
        ?? nombreCentroEn(etiquetasLista.centrosOrganizacion, referencia)
        ?? etiquetasLista.textos.traducir("general.lista_centro_nombre_no_disponible");
      const nombreCategoria = (referencia) => etiquetasLista.categorias.get(referencia)
        ?? etiquetasLista.textos.traducir("general.lista_categoria_nombre_no_disponible");
      let perfilIntervencion = false;
      if (consultarSesion !== null) {
        const sesion = await consultar((opciones) => consultarSesion(opciones), "consultar sesión CT");
        if (!Array.isArray(sesion?.roles) || sesion.roles.length !== 1
          || !PERFILES_CT_MENU.has(sesion.roles[0])) {
          throw new TypeError("perfil CT no acreditado");
        }
        perfilIntervencion = sesion.roles[0] === ROL_INTERVENCION;
      }
      let promesaCuadro = null;
      let promesaCompleto = null;
      let promesaResumen = null;
      let cuadroInicio = null;
      const esperarCuadroLigero = () => {
        promesaCuadro ??= cargarModuloConLimite(recursos.cargarCuadroLigero,
          CLAVE_CONTRATACION_TEMPORAL, limiteCargaModularMs, temporizadores)
          .catch((error) => { promesaCuadro = null; throw error; });
        return promesaCuadro;
      };
      const activarCompleto = () => {
        promesaCompleto ??= recursos.cargarCompleto().then((partes) => cargarContratacionTemporal({
          consultar, exigirVigente, notificar,
          recursosRecibidos: { ...recursos, ...partes, cargarCuadroLigero: null },
        })).then(({ contratacionTemporal }) => contratacionTemporal)
          .catch((error) => { promesaCompleto = null; throw error; });
        return promesaCompleto;
      };
      const prepararResumenInicio = () => {
        if (perfilIntervencion) return Promise.resolve(null);
        promesaResumen ??= consultar((opciones) => cliente.consultarCuadroRRHH({
          filtros: { texto: "", estado_clave: "", fase_clave: "" },
          paginacion: { limite: 1, cursor: "" }, resumen: true,
        }, opciones)).then((pagina) => {
          if (!pagina?.resumen) throw new TypeError("resumen CT no disponible");
          cuadroInicio = Object.freeze({ resumen: pagina.resumen });
          return cuadroInicio;
        }).catch((error) => { promesaResumen = null; throw error; });
        return promesaResumen;
      };
      return { contratacionTemporal: Object.freeze({
        modoLigero: true, cliente, esperarCuadroLigero, activarCompleto,
        prepararNombresLista, nombreCentro, nombreCategoria,
        alta: null, fiscalizacion: perfilIntervencion ? Object.freeze({ cliente }) : null,
        prepararResumenInicio, obtenerCuadroInicio: () => cuadroInicio,
      }) };
    }
    // Vista y auditoría: ya incluidas (cargadores de prueba) o, en el portal,
    // pedidas con `cargarVista` al abrir CT.
    let partesVista = recursos.vista ? recursos : null;
    let promesaVista = null;
    const esperarVista = () => {
      promesaVista ??= partesVista ? Promise.resolve(partesVista)
        : cargarModuloConLimite(recursos.cargarVista, CLAVE_CONTRATACION_TEMPORAL, limiteCargaModularMs, temporizadores)
          .then((partes) => { partesVista = partes; return partes; })
          .catch((error) => { promesaVista = null; throw error; });
      return promesaVista;
    };
    const idiomaCircuito = INDICE_IDIOMAS.idiomas.find(({ localizacion }) => localizacion === locale).codigo;
    let fasesCircuito;
    try {
      fasesCircuito = await cargarModuloConLimite(() => cargarFasesCircuito(idiomaCircuito),
        "contratacion_temporal.circuito", limiteCargaModularMs, temporizadores);
    } catch {
      throw Object.assign(new Error("catálogo del circuito no disponible"), {
        codigo: "catalogo_circuito_no_disponible",
      });
    }
    if (!fasesCircuito || typeof fasesCircuito !== "object") {
      throw Object.assign(new Error("catálogo del circuito no válido"), {
        codigo: "catalogo_circuito_no_valido",
      });
    }
    const rotulosCircuito = (prefijo) => Object.fromEntries(Object.entries(fasesCircuito)
      .map(([clave, rotulo]) => [`${prefijo}circuito_${clave}`, rotulo]));
    const mensajesExpedientesIdioma = idiomaCircuito === "en"
      ? await (await import("./modulos/contratacion-temporal/i18n-expedientes.js?v=20261007-pantallas-textos-final-v1"))
        .cargarMensajesExpedientesContratacionEnIdioma(idiomaCircuito)
      : {};
    const mensajesExpedientes = {
      ...mensajesExpedientesIdioma,
      ...rotulosCircuito("contratacion_temporal.fase."),
      ...rotulosCircuito("etiqueta_fase_"),
    };
    exigirVigente();
    const cliente = recursos.cliente.crearClienteHTTPContratacionTemporal({
      fetchImpl: fetchDelEntorno(),
      HeadersImpl: entorno.Headers,
    });
    const clienteIncorporacionB2 = typeof recursos.incorporacionB2?.crearClienteIncorporacionPersonalB2HTTP === "function"
      ? recursos.incorporacionB2.crearClienteIncorporacionPersonalB2HTTP({
        fetchImpl: fetchDelEntorno(), HeadersImpl: entorno.Headers,
      }) : null;
    let alta = null;
    let altaResuelta = false;
    let promesaCentrosOrganizacion = null;
    let centrosOrganizacionResueltos = null;
    // Sin catálogo del alta (perfil que no da de alta peticiones), los nombres
    // de centro salen de la estructura pública de Organización. Una sola
    // consulta por carga; si falla, la lista sigue con la referencia.
    const centrosOrganizacion = () => {
      if (!altaResuelta || alta !== null) return null;
      promesaCentrosOrganizacion ??= import("./modulos/personal/cliente-http-estructura-organizativa-publica.js?v=20260925-portal-integrado-v1")
        .then((modulo) => modulo.crearClienteHTTPEstructuraOrganizativaPublica({ fetchImpl: fetchDelEntorno() ?? globalThis.fetch }).obtener())
        .then((estructura) => centrosDeOrganizacion(estructura.unidades))
        .catch(() => null)
        .then((centros) => { centrosOrganizacionResueltos = centros; return centros; });
      return promesaCentrosOrganizacion;
    };
    // La jornada completa de referencia y las etiquetas de las modalidades
    // llegan con la configuración del análisis.
    let jornadaCompleta = null;
    let entregarModalidades = () => {};
    const promesaModalidades = new Promise((resolver) => { entregarModalidades = resolver; });
    const fuente = recursos.adaptador
      .crearAdaptadorHTTPExpedientesContratacionTemporal({
        cliente, locale, mensajes: mensajesExpedientes,
        obtenerCatalogos: () => alta?.catalogos ?? null,
        obtenerJornadaCompleta: () => jornadaCompleta,
        obtenerModalidades: () => promesaModalidades,
        obtenerCentrosOrganizacion: centrosOrganizacion,
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
          obtenerCatalogosNecesidadesAlta: (opciones) => cliente.obtenerCatalogosNecesidadesAlta(opciones),
        });
      } catch {
        alta = null;
      }
    }, () => { alta = null; }).finally(() => {
      altaResuelta = true;
      // Sin alta, se piden ya los nombres de Organización y se avisa para
      // repintar Inicio y la lista con ellos.
      if (alta === null) void centrosOrganizacion()?.then((centros) => { if (centros) notificar(); });
    });
    // La portada solo necesita los recuentos de todo el cuadro, no sus filas.
    const consultaCuadro = consultar((opciones) => fuente.resumenInicio(opciones));
    const promesaConfiguracion = consultar((opciones) => cliente.obtenerConfiguracionAnalisis(opciones));
    promesaConfiguracion.then((valor) => entregarModalidades(valor?.modalidades ?? null), () => entregarModalidades(null));
    const [cuadro, configuracion] = await Promise.allSettled([consultaCuadro, promesaConfiguracion]);
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
            // Opcionales: solo existen si el catálogo de reglas las publica.
            ...(configuracionAnalisis.duraciones_maximas
              ? { duraciones_maximas: configuracionAnalisis.duraciones_maximas } : {}),
            ...(configuracionAnalisis.urgencia_disponible === true ? { urgencia_disponible: true } : {}),
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
      && await esperarVista().then((partes) => typeof partes.vista.montarModuloFiscalizacionContratacionTemporal === "function", () => false)
      && await sesionDeIntervencion(consultar)
      ? Object.freeze({ cliente }) : null;
    exigirVigente();
    if (!cuadroDisponible && alta === null && fiscalizacion === null) {
      throw new Error("contratación temporal no disponible");
    }
    return {
      contratacionTemporal: Object.freeze({
        mensajesExpedientes,
        clienteIncorporacionB2,
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
        // Con la vista ya cargada (el montaje la espera antes de leer esto).
        get auditoriaComun() {
          return typeof partesVista?.auditoriaVista?.montarVistaAuditoria === "function"
            && typeof partesVista.auditoriaCliente?.crearFuenteAuditoriaHTTP === "function"
            ? Object.freeze({ montar: partesVista.auditoriaVista.montarVistaAuditoria,
              fuente: partesVista.auditoriaCliente.crearFuenteAuditoriaHTTP({ fetchImpl: fetchDelEntorno() ?? globalThis.fetch }) }) : null;
        },
        esperarVista,
        // Portada: recuentos de todo el cuadro calculados por el servidor con
        // la misma autorización que la lista.
        obtenerCuadroInicio: () => listadoCuadro,
        montar: async (opciones) => (await esperarVista()).vista.montarModuloContratacionTemporal(opciones),
        montarFiscalizacion: async (opciones) => (await esperarVista()).vista
          .montarModuloFiscalizacionContratacionTemporal(opciones),
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
    const estadoRPT = { estado: catalogos.estadoRPT };
    let reintentoRPTEnCurso = null;
    const reintentarRPT = ({ signal } = {}) => {
      if (estadoRPT.estado !== "incidencia" || signal?.aborted) return Promise.resolve(estadoRPT.estado);
      try { exigirVigente(); } catch { return Promise.resolve(estadoRPT.estado); }
      if (reintentoRPTEnCurso && !reintentoRPTEnCurso.signal?.aborted)
        return reintentoRPTEnCurso.promesa;
      const vuelo = (async () => {
        const crearCliente = catalogos.recursos?.clienteRPT?.crearClienteHTTPRPTPublica;
        const fetchImpl = fetchDelEntorno();
        if (typeof crearCliente !== "function" || !fetchImpl) return "ausente";
        let resultado;
        try {
          const cliente = crearCliente({ fetchImpl });
          await cliente.listar({ vista: "categorias", q: "", limit: 1, offset: 0 }, { signal });
          resultado = { ok: true };
        } catch (error) {
          resultado = { codigo: error?.codigo, estado: error?.estado };
        }
        try { exigirVigente(); } catch { return estadoRPT.estado; }
        if (signal?.aborted) return estadoRPT.estado;
        estadoRPT.estado = estadoSondaRPT(resultado);
        return estadoRPT.estado;
      })();
      const intento = { signal, promesa: null };
      intento.promesa = vuelo.finally(() => {
        if (reintentoRPTEnCurso === intento) reintentoRPTEnCurso = null;
      });
      reintentoRPTEnCurso = intento;
      return intento.promesa;
    };
    const personal = typeof recursos.ficha?.montarVistaFichaIntegralPersonal === "function"
      ? componerPersonalVisible({ ...recursos, ...catalogos.recursos }, entorno, {
        catalogosPublicos: catalogos.disponibles, ocultarSinFuente: true,
        estadoRPT, reintentarRPT,
        // Abrir un destino diferido no exige haberlo visitado antes. Esto
        // sólo ofrece navegación propia; su lectura se autoriza al entrar.
        // Sólo se ofrecen destinos del catálogo; un módulo oculto no aparece.
        destinosDisponibles: () => Object.fromEntries(["dietas", "cronos"]
          .filter((clave) => catalogo.some((modulo) => modulo.clave === clave))
          .map((clave) => [clave, [ESTADO_DIFERIDO, "cargando", "disponible"].includes(estadoCargaModulo(clave))])),
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

  // Una incidencia temporal de RPT permite reintento expreso desde la ficha;
  // ausencia y denegación se mantienen diferenciadas.
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
    const resultados = await Promise.allSettled(sondeos.map(async ([, crear, sondear]) => {
      let cliente;
      try { cliente = crear(); } catch { return { ok: false, codigo: "montaje_ausente" }; }
      if (!cliente || typeof cliente !== "object") return { ok: false, codigo: "montaje_ausente" };
      return consultar(async (opciones) => {
        try { await sondear(cliente, opciones); return { ok: true }; }
        catch (error) { return { ok: false, codigo: error?.codigo, estado: error?.estado }; }
      }, "consultar catálogos de Personal");
    }));
    const disponibles = sondeos.filter((_, indice) => resultados[indice].status === "fulfilled" && resultados[indice].value.ok)
      .map(([clave]) => clave);
    const estadoRPT = resultados[0].status === "fulfilled" ? estadoSondaRPT(resultados[0].value) : "incidencia";
    return Object.freeze({ recursos, disponibles: Object.freeze(disponibles), estadoRPT });
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
    recursosTramites = null; cargaTramites = null; estadoTramites = ESTADO_DIFERIDO;
    catalogo = Object.freeze([]);
    catalogoOfrecido = catalogo;
    ctRecuperableEnMenu = false;
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
    notificarTramites = () => notificar("mis-tramites");

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
      const intentos = clave === CLAVE_CONTRATACION_TEMPORAL ? 2 : 1;
      for (let intento = 0; intento < intentos && vigente(); intento += 1) {
        try {
          resultado = await CARGAS_MODULOS[clave]({ consultar, exigirVigente, notificar: () => notificar(clave) });
          break;
        } catch (error) {
          if (clave !== CLAVE_CONTRATACION_TEMPORAL || !vigente()) break;
          // Solo códigos controlados: la excepción puede contener URL, datos de
          // respuesta o detalles de identidad que no deben ir a la consola.
          const codigo = ["catalogo_circuito_no_disponible", "catalogo_circuito_no_valido", "carga_recursos_ct"]
            .includes(error?.codigo) ? error.codigo : "carga_ct_no_disponible";
          const causa = codigo === "carga_recursos_ct" ? "importacion"
            : codigo.startsWith("catalogo_") ? "catalogo" : "consulta_o_permiso";
          entorno.console?.error?.("portal.modulo.carga_fallida", {
            modulo: CLAVE_CONTRATACION_TEMPORAL, codigo, causa, intento: intento + 1,
          });
          // La importación y los rótulos no han iniciado consultas de CT.
          // Una respuesta fallida tras consultar puede haber auditado la lectura;
          // se recupera solo mediante la acción expresa de la persona.
          if (codigo === "carga_ct_no_disponible") break;
        }
      }
      if (!vigente()) return;
      if (clave === CLAVE_CONTRATACION_TEMPORAL && !resultado && consultarSesion !== null) {
        try {
          const sesion = await consultarSesion();
          if (!vigente()) return;
          ctRecuperableEnMenu = Array.isArray(sesion?.roles) && sesion.roles.length === 1
            && PERFILES_CT_MENU.has(sesion.roles[0]);
        } catch { ctRecuperableEnMenu = false; }
      }
      if (!vigente()) return;
      if (resultado) Object.assign(partes, resultado);
      estados[clave] = resultado ? "disponible" : "no_disponible";
      publicar();
      notificar(clave);
    };
    const promesasModulos = new Map();
    const iniciarModulo = (clave) => {
      const promesa = cargarModulo(clave); promesasModulos.set(clave, promesa); return promesa;
    };
    cargaDiferida = (clave, esperarEnCurso = false) => {
      if (!vigente()) return null;
      if (esperarEnCurso && estados[clave] === "cargando") return promesasModulos.get(clave) ?? null;
      if (estados[clave] !== ESTADO_DIFERIDO) return null;
      estados[clave] = "cargando";
      publicar();
      return iniciarModulo(clave);
    };
    publicar();
    notificar("catalogo");

    await Promise.allSettled(cargables.map(iniciarModulo));
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
    if (vista === "mis-tramites") {
      const claves = ["cronos", "dietas"].filter((clave) => catalogo.some((modulo) => modulo.clave === clave));
      if (!claves.length || cargaTramites || typeof cargaDiferida !== "function") return null;
      const carga = secuenciaCarga;
      estadoTramites = "cargando";
      cargaTramites = Promise.all([cargarModuloConLimite(cargarTramitesPropios, "mis-tramites", limiteCargaModularMs, temporizadores),
        ...claves.map((clave) => cargaDiferida(clave, true))])
        .then(([recursos]) => {
          if (carga !== secuenciaCarga) return;
          if (typeof recursos?.fuente?.crearFuenteTramitesPropios !== "function"
            || typeof recursos?.vista?.montarVistaTramitesPropios !== "function") throw new TypeError("trámites propios no disponibles");
          recursosTramites = recursos; estadoTramites = "disponible";
        }).catch(() => { if (carga === secuenciaCarga) estadoTramites = "no_disponible"; })
        .finally(() => { if (carga === secuenciaCarga) notificarTramites(); });
      return cargaTramites;
    }
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
    if (vista === "mis-tramites") return catalogo.some((modulo) => ["cronos", "dietas"].includes(modulo.clave))
      && [ESTADO_DIFERIDO, "cargando"].includes(estadoTramites);
    const modulo = moduloDeVistaPortal(vista);
    if (["cargando", ESTADO_DIFERIDO].includes(estadoCargaModulo(modulo))) return true;
    // El registro RRHH depende también de conocer el perfil.
    return vista === "personal-registro" && estadoCargaModulo(CLAVE_CONTRATACION_TEMPORAL) === "cargando";
  }

  function obtenerCatalogo() {
    return catalogoOfrecido;
  }

  function obtenerAccesosEmpleado() {
    const accesos = Object.fromEntries(["personal", "cronos", "dietas"]
      .filter((clave) => catalogo.some((modulo) => modulo.clave === clave))
      .map((clave) => [clave, Object.freeze({ estado: estadoCargaModulo(clave) })]));
    if (catalogo.some((modulo) => ["cronos", "dietas"].includes(modulo.clave)))
      accesos["mis-tramites"] = Object.freeze({ estado: estadoTramites });
    return Object.freeze(accesos);
  }

  function vistaDisponible(vista) {
    if (vista === "mis-tramites") return estadoTramites === "disponible" && recursosTramites !== null;
    if (VISTAS_MODULO_BOLSA.has(vista)) {
      return montajeBolsa !== null && montajeBolsa.disponible(vista) === true;
    }
    if (vista === "contratacion-temporal" || vista === VISTA_PLANTILLAS_RRHH || vista === VISTA_CATEGORIAS_RPT) {
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
        recuperable: ctRecuperableEnMenu,
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
    // Menú fijo: los mismos módulos en cualquier pantalla; solo cambia el activo.
    return renderizarNavegacionModulos({
      catalogo: catalogoOfrecido,
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

    if (vista === "mis-tramites") {
      const fuente = recursosTramites.fuente.crearFuenteTramitesPropios({
        consultarPermisos: composicion?.cronos?.consultarPermisos,
        listarComisiones: composicion?.dietas?.clienteBorradores?.listar?.bind(composicion.dietas.clienteBorradores),
      });
      const vistaTramites = recursosTramites.vista.montarVistaTramitesPropios({ raiz, fuente, anunciar });
      desmontarVista = vistaTramites.desmontar;
      return true;
    }

    if (vista === VISTA_CATEGORIAS_RPT) {
      const { montarCategoriasRPT } = await import("./categorias-rpt/montaje.js?v=20261008-analisis-confirmado-v2");
      if (montaje !== secuenciaMontaje) return false;
      const modulo = montarCategoriasRPT({ raiz });
      if (montaje !== secuenciaMontaje) { modulo.desmontar(); return false; }
      desmontarVista = modulo.desmontar;
      return true;
    }

    if (vista === VISTA_PLANTILLAS_RRHH) {
      const { montarRRHHPlantillas } = await import("./modulos/contratacion-temporal/rrhh-plantillas-vista.js?v=20261008-alta-rpt-circular-v6");
      if (montaje !== secuenciaMontaje) return false;
      const modulo = montarRRHHPlantillas({ raiz, anunciar });
      if (montaje !== secuenciaMontaje) { modulo.desmontar(); return false; }
      desmontarVista = modulo.desmontar;
      return true;
    }

    if (vista === "contratacion-temporal") {
      let temporal = composicion.contratacionTemporal;
      if (temporal.modoLigero === true) {
        const expedienteDirecto = typeof opciones?.expedienteRef === "string" && opciones.expedienteRef !== "";
        const requiereCompleto = expedienteDirecto || opciones?.subvista === "alta"
          || temporal.fiscalizacion !== null || opciones?.filtros != null;
        if (!requiereCompleto) {
          const controladorMontaje = new AbortController();
          desmontarVista = () => controladorMontaje.abort();
          const moduloLigero = await temporal.esperarCuadroLigero();
          if (montaje !== secuenciaMontaje) { controladorMontaje.abort(); return false; }
          const mostrarError = (destino, { error, reintentar, mensaje }) => {
            if (montaje !== secuenciaMontaje) return;
            const denegado = [401, 403].includes(error?.estado) || error?.codigo === "acceso_denegado";
            destino.innerHTML = `<section class="panel"><div class="cuerpo-panel vacio-controlado" role="alert">
              <p>${escaparHTML(mensaje ?? traducir(denegado ? "estado_modulo_sin_permiso" : "estado_modulo_no_disponible"))}</p>
              ${denegado ? "" : `<button type="button" data-ct-reintentar>${escaparHTML(traducir("accion_reintentar"))}</button>`}
            </div></section>`;
            destino.querySelector?.("[data-ct-reintentar]")?.addEventListener("click", () => { void reintentar(); }, { once: true });
          };
          const modulo = await moduloLigero.montarCuadroContratacionLigero({
            raiz, cliente: { consultarCuadroRRHH: async (solicitud, opciones) => {
              return temporal.prepararNombresLista(temporal.cliente.consultarCuadroRRHH(solicitud, opciones));
            } }, idioma: INDICE_IDIOMAS.idiomas.find(({ localizacion }) => localizacion === locale).codigo,
            filtroLista: opciones?.filtroLista ?? null, signal: controladorMontaje.signal,
            filtroServidorRuta: opciones?.filtroServidorRuta ?? null,
            alCambiarFiltroLista: opciones?.alCambiarFiltroLista ?? null,
            nombreCentro: temporal.nombreCentro, nombreCategoria: temporal.nombreCategoria,
            abrirDetalle: ({ expedienteRef }) => montarVista("contratacion-temporal", raiz, { ...opciones, expedienteRef }),
            abrirAlta: esPerfilRRHH() ? () => montarVista("contratacion-temporal", raiz, { ...opciones, subvista: "alta" }) : null,
            mostrarError,
          });
          if (montaje !== secuenciaMontaje) { controladorMontaje.abort(); modulo.desmontar(); return false; }
          desmontarVista = () => { controladorMontaje.abort(); modulo.desmontar(); };
          return true;
        }
        temporal = await temporal.activarCompleto();
        if (montaje !== secuenciaMontaje) return false;
      }
      // Los catálogos del alta ya están en camino; la vista se pide ahora.
      await Promise.all([temporal.esperarAlta?.(), temporal.esperarVista?.()]);
      if (montaje !== secuenciaMontaje) return false;
      const esFiscalizacion = temporal.fiscalizacion !== null;
      const presentadorCT = temporal.crearPresentador();
      let controladorBolsaFicha = null;
      let claveBolsaFicha = "";
      let actualizarBolsaFicha = () => {};
      const cancelarBolsaFicha = () => {
        controladorBolsaFicha?.abort();
        controladorBolsaFicha = null;
        claveBolsaFicha = "";
        montajeBolsa?.limpiarFicha?.();
      };
      const prepararFichaBolsa = (estadoFicha, alActualizar = () => {}, reintentar = false) => {
        const expediente = estadoFicha?.expediente;
        const resumen = estadoFicha?.cuadro?.expedientes?.find(
          ({ expediente_ref: ref }) => ref === expediente?.expediente_ref,
        );
        const valida = estadoFicha?.vista === "expediente" && estadoFicha.carga === "listo"
          && expediente?.expediente_ref === estadoFicha.expediente_ref
          && Number.isSafeInteger(expediente?.version) && expediente.version > 0
          && resumen?.version === expediente.version && expediente.demostracion === false
          && estadoFicha.cuadro?.demostracion === false;
        if (!valida || typeof montajeBolsa?.prepararFicha !== "function") {
          if (controladorBolsaFicha) cancelarBolsaFicha();
          return;
        }
        actualizarBolsaFicha = alActualizar;
        const clave = `${expediente.expediente_ref}:${expediente.version}`;
        if (!reintentar && claveBolsaFicha === clave) return;
        cancelarBolsaFicha();
        claveBolsaFicha = clave;
        controladorBolsaFicha = new AbortController();
        const signal = controladorBolsaFicha.signal;
        void Promise.resolve().then(() => montajeBolsa.prepararFicha({ expedienteRef: expediente.expediente_ref, signal }))
          .catch(() => { if (!signal.aborted) montajeBolsa.fallarFicha?.(expediente.expediente_ref); })
          .then(() => { if (!signal.aborted && claveBolsaFicha === clave && montaje === secuenciaMontaje) actualizarBolsaFicha(); });
      };
      // El montaje puede cambiar mientras se consulta el cuadro o el detalle.
      // Registrar la limpieza antes de esperar evita publicar una respuesta tardía.
      desmontarVista = () => { cancelarBolsaFicha(); presentadorCT.desmontar?.(); };
      if (opciones?.subvista && typeof presentadorCT?.cambiarVista === "function"
        && ["alta", "cuadro"].includes(opciones.subvista)) {
        try { presentadorCT.cambiarVista(opciones.subvista); } catch {}
      }
      const expedienteRef = typeof opciones?.expedienteRef === "string" ? opciones.expedienteRef : "";
      if (!esFiscalizacion && (opciones?.filtros || expedienteRef)
        && typeof presentadorCT?.cargar === "function") {
        // La selección exige un cuadro consultado con capacidad positiva.
        // La vista no repetirá la carga porque el presentador ya tiene estado.
        await presentadorCT.cargar({ texto: "", estado: "", fase: "", ...(opciones.filtros || {}) });
        if (montaje !== secuenciaMontaje) return false;
      }
      if (!esFiscalizacion && expedienteRef
        && presentadorCT?.obtenerEstado?.().cuadro?.expedientes?.some(
          (expediente) => expediente.expediente_ref === expedienteRef,
        ) && typeof presentadorCT?.seleccionarExpediente === "function") {
        await presentadorCT.seleccionarExpediente(expedienteRef);
        if (montaje !== secuenciaMontaje) return false;
      }
      if (!esFiscalizacion) prepararFichaBolsa(presentadorCT.obtenerEstado?.());
      const moduloContratacion = esFiscalizacion
        ? await temporal.montarFiscalizacion({
          raiz,
          cliente: temporal.fiscalizacion.cliente,
          locale,
          zonaHoraria: ZONA_HORARIA_PORTAL,
          confirmarOperacion,
          anunciar,
        })
        : await temporal.montar({
          raiz,
          presentador: presentadorCT,
          filtroLista: opciones?.filtroLista ?? null,
          locale,
          zonaHoraria: ZONA_HORARIA_PORTAL,
          mensajes: temporal.mensajesExpedientes,
          alta: temporal.alta,
          analisis: temporal.analisis,
          fiscalizacion: typeof temporal.analisis?.cliente
            ?.registrarResultadoFiscalizacion === "function"
            ? { cliente: temporal.analisis.cliente }
            : null,
          continuidad: temporal.continuidad,
          incorporacionPersonalB2: temporal.clienteIncorporacionB2,
          subsanacion: temporal.subsanacion,
          auditoriaComun: temporal.auditoriaComun,
          // Lista común de documentos en la ficha, solo si el portal publica
          // el módulo de Documentos para esta sesión (se consulta al pintar).
          documentosComun: Object.freeze({
            montar: (opciones) => (typeof composicion?.documentos?.montar === "function"
              ? composicion.documentos.montar(opciones) : null),
          }),
          confirmarOperacion,
          anunciar,
          resolverBolsa: typeof montajeBolsa?.resolverBolsa === "function" ? montajeBolsa.resolverBolsa : null,
          prepararFichaBolsa,
        });
      if (montaje !== secuenciaMontaje) {
        moduloContratacion.desmontar();
        return false;
      }
      desmontarVista = () => { cancelarBolsaFicha(); moduloContratacion.desmontar(); };
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
            ${vista === "cronos-bandeja" ? `<button type="button" class="boton-secundario" data-vista="cronos-bandeja" aria-current="page">${escaparHTML(composicion.cronos.etiquetas.bandeja)}</button>` : ""}` : ""}
            ${typeof composicion.cronos.montarNotificaciones === "function" ? `<button type="button" class="boton-secundario" data-vista="cronos-notificaciones"${vista === "cronos-notificaciones" ? ' aria-current="page"' : ""}>${escaparHTML(composicion.cronos.etiquetas.notificaciones)}</button>
            ${vista === "cronos-bandeja-notificaciones" ? `<button type="button" class="boton-secundario" data-vista="cronos-bandeja-notificaciones" aria-current="page">${escaparHTML(composicion.cronos.etiquetas.bandejaNotificaciones)}</button>` : ""}` : ""}
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

  function altaCTDisponible() {
    return esPerfilRRHH() && composicion?.contratacionTemporal?.alta != null;
  }

  function obtenerCuadroInicio() {
    if (!esPerfilRRHH()) return null;
    return composicion?.contratacionTemporal?.obtenerCuadroInicio?.() || null;
  }

  function prepararResumenInicio() {
    if (!esPerfilRRHH()) return Promise.resolve(null);
    return composicion?.contratacionTemporal?.prepararResumenInicio?.()
      ?? Promise.resolve(obtenerCuadroInicio());
  }

  return Object.freeze({
    cargarInterno,
    desmontarVistaActual,
    prepararVista,
    esPerfilRRHH,
    altaCTDisponible,
    inicioPendiente,
    montarVista,
    obtenerCatalogo,
    obtenerAccesosEmpleado,
    obtenerCuadroInicio,
    prepararResumenInicio,
    renderizarNavegacion,
    resolverAcceso,
    retirarVistaMontada,
    vistaGestionada,
    vistaDisponible,
    vistaPendiente,
  });
}
