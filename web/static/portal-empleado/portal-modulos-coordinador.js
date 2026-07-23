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
} from "./portal-catalogo-modulos.js?v=20260721-acceso-real-v2";
import {
  CAPACIDAD_CREAR_SOLICITUD,
} from "./modulos/contratacion-temporal/contrato.js";
import {
  crearPresentadorExpedientesContratacionTemporal,
} from "./modulos/contratacion-temporal/presentador-expedientes.js";
import {
  montarModuloContratacionTemporal,
} from "./modulos/contratacion-temporal/vista-expedientes.js";
import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_SOLICITAR_PERMISO,
} from "./modulos/cronos/contrato.js";
import { crearPresentadorCronos } from "./modulos/cronos/presentador.js";
import {
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_GESTIONAR_RUTA,
} from "./modulos/dietas/contrato.js";
import { montarModuloDietas } from "./modulos/dietas/vista.js";
import { crearVisorRutaDietas } from "./modulos/dietas/mapa-ruta.js";
import { traducirPortal } from "./portal-i18n.js?v=20260721-acceso-real-v2";

const CAPACIDADES_AUTOSERVICIO_CRONOS = Object.freeze([
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_SOLICITAR_PERMISO,
]);
const CAPACIDADES_AUTOSERVICIO_DIETAS = Object.freeze([
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_RUTA,
]);

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
} = {}) {
  if (typeof escaparHTML !== "function" || typeof anunciar !== "function"
    || typeof confirmarOperacion !== "function" || typeof traducir !== "function") {
    throw new TypeError("dependencias del coordinador de módulos no válidas");
  }

  let catalogo = Object.freeze([]);
  let composicion = null;
  let presentacionActiva = false;
  let desmontarVista = null;
  let secuenciaMontaje = 0;
  // La pila Docker de presentacion publica teselas bajo el mismo origen. Si
  // Leaflet no estuviera disponible, el visor muestra un aviso textual y
  // nunca intenta una fuente cartografica externa ni una ruta simulada.
  const visorRutaDietas = crearVisorRutaDietas({ entorno, permitirTeselas: true });

  function desmontarVistaActual() {
    secuenciaMontaje += 1;
    if (typeof desmontarVista === "function") desmontarVista();
    desmontarVista = null;
  }

  async function cargarPresentacion(sesionBolsa) {
    desmontarVistaActual();
    presentacionActiva = true;
    const [
      identidad,
      datosCronos,
      adaptadorCronos,
      adaptadorDietas,
      calculadorRutasDietas,
      documentos,
      catalogoDemo,
      adaptadorContratacion,
    ] = await Promise.all([
      import("./identidad/presentacion.js"),
      import("./modulos/cronos/datos-presentacion.js"),
      import("./modulos/cronos/adaptador-presentacion.js"),
      import("./modulos/dietas/adaptador-presentacion.js"),
      import("./modulos/dietas/calculador-rutas-presentacion-osrm.js"),
      import("./documentos/descarga-recibos-presentacion.js"),
      import("./portal-catalogo-presentacion.js"),
      import("./modulos/contratacion-temporal/adaptador-presentacion.js"),
    ]);
    const contexto = identidad.crearContextoActorPresentacionDesdeSesion(sesionBolsa);
    const contextos = compartirContextoActor(
      crearProveedorContextoActorFijo(contexto), contexto.ambito.modulos,
    );
    const descargarRecibo = documentos.crearDescargadorRecibosPresentacion(entorno);
    const origenComprobacion = entorno.location?.origin || "";
    const cronos = contextos.cronos ? crearPresentadorCronos({
      contextoActor: contextos.cronos,
      capacidades: CAPACIDADES_AUTOSERVICIO_CRONOS,
      datos: datosCronos.crearDatosCronosPresentacion(contextos.cronos),
      ejecutor: adaptadorCronos.crearEjecutorCronosPresentacion(),
      descargarRecibo,
      origenComprobacion,
    }) : undefined;
    const dietas = contextos.dietas ? Object.freeze({
      contextoActor: contextos.dietas,
      capacidades: CAPACIDADES_AUTOSERVICIO_DIETAS,
      adaptador: adaptadorDietas.crearAdaptadorDietasPresentacion({
        contextoActor: contextos.dietas,
        capacidades: CAPACIDADES_AUTOSERVICIO_DIETAS,
      }),
      calculadorRuta: calculadorRutasDietas.crearCalculadorRutasDietasPresentacionOSRM({
        contextoActor: contextos.dietas,
        capacidades: CAPACIDADES_AUTOSERVICIO_DIETAS,
        fetchImpl: typeof entorno.fetch === "function" ? entorno.fetch.bind(entorno) : undefined,
      }),
      descargarRecibo,
      visorRuta: visorRutaDietas,
      origenComprobacion,
    }) : undefined;
    const contratacionTemporal = contextos.contratacion_temporal ? (() => {
      const fuente = adaptadorContratacion.crearAdaptadorContratacionTemporalPresentacion({
        contextoActor: contextos.contratacion_temporal,
      });
      return Object.freeze({
        crearPresentador: () => crearPresentadorExpedientesContratacionTemporal({
          fuente, capacidades: fuente.capacidades,
        }),
        alta: Object.freeze({
          catalogos: fuente.obtenerCatalogosAlta(),
          capacidad: CAPACIDAD_CREAR_SOLICITUD,
          ejecutor: fuente.registrarSolicitud,
        }),
      });
    })() : undefined;
    catalogo = catalogoDemo.obtenerCatalogoModulosPresentacion();
    composicion = Object.freeze({
      contextos,
      contratacionTemporal,
      cronos,
      dietas,
    });
    return contextos.bolsa || null;
  }

  async function cargarInterno() {
    desmontarVistaActual();
    presentacionActiva = false;
    composicion = null;
    catalogo = await cargarCatalogoModulosInterno();
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
      const moduloContratacion = await montarModuloContratacionTemporal({
        raiz,
        presentador: composicion.contratacionTemporal.crearPresentador(),
        alta: composicion.contratacionTemporal.alta,
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

    const moduloDietas = await montarModuloDietas({
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
