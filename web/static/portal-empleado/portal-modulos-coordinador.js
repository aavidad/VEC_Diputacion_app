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
} from "./portal-catalogo-modulos.js";
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

const MODULOS_COMPARTIDOS = Object.freeze(["bolsa", "cronos", "dietas"]);
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

export function moduloDeVistaPortal(vista) {
  if (vista === "portal") return "portal";
  if (VISTAS_MODULOS_PERSONALES.has(vista)) return vista;
  return "bolsa";
}

export function rutaDeVistaPortal(vista) {
  if (vista === "portal") return "#portal";
  if (VISTAS_MODULOS_PERSONALES.has(vista)) return `#${vista}`;
  return `#bolsa/${vista}`;
}

export function crearCoordinadorModulosPortal({
  escaparHTML,
  anunciar = () => {},
  confirmarOperacion = () => false,
  entorno = globalThis,
} = {}) {
  if (typeof escaparHTML !== "function" || typeof anunciar !== "function"
    || typeof confirmarOperacion !== "function") {
    throw new TypeError("dependencias del coordinador de módulos no válidas");
  }

  let catalogo = Object.freeze([]);
  let composicion = null;
  let desmontarVista = null;
  let secuenciaMontaje = 0;
  // El binario de presentación no compone conectores de red. Producción debe
  // crear este puerto con `permitirTeselas: true` únicamente tras publicar el
  // proxy same-origin `/tiles/osm/` dentro de la red corporativa.
  const visorRutaDietas = crearVisorRutaDietas({ entorno, permitirTeselas: false });

  function desmontarVistaActual() {
    secuenciaMontaje += 1;
    if (typeof desmontarVista === "function") desmontarVista();
    desmontarVista = null;
  }

  async function cargarPresentacion(sesionBolsa) {
    desmontarVistaActual();
    const [identidad, datosCronos, adaptadorCronos, adaptadorDietas, calculadorRutasDietas, documentos, catalogoDemo] = await Promise.all([
      import("./identidad/presentacion.js"),
      import("./modulos/cronos/datos-presentacion.js"),
      import("./modulos/cronos/adaptador-presentacion.js"),
      import("./modulos/dietas/adaptador-presentacion.js"),
      import("./modulos/dietas/calculador-rutas-presentacion.js"),
      import("./documentos/descarga-recibos-presentacion.js"),
      import("./portal-catalogo-presentacion.js"),
    ]);
    const contexto = identidad.crearContextoActorPresentacionDesdeSesion(sesionBolsa);
    const contextos = compartirContextoActor(
      crearProveedorContextoActorFijo(contexto), MODULOS_COMPARTIDOS,
    );
    const descargarRecibo = documentos.crearDescargadorRecibosPresentacion(entorno);
    const origenComprobacion = entorno.location?.origin || "";
    const cronos = crearPresentadorCronos({
      contextoActor: contextos.cronos,
      capacidades: CAPACIDADES_AUTOSERVICIO_CRONOS,
      datos: datosCronos.crearDatosCronosPresentacion(contextos.cronos),
      ejecutor: adaptadorCronos.crearEjecutorCronosPresentacion(),
      descargarRecibo,
      origenComprobacion,
    });
    const dietas = Object.freeze({
      contextoActor: contextos.dietas,
      capacidades: CAPACIDADES_AUTOSERVICIO_DIETAS,
      adaptador: adaptadorDietas.crearAdaptadorDietasPresentacion({
        contextoActor: contextos.dietas,
        capacidades: CAPACIDADES_AUTOSERVICIO_DIETAS,
      }),
      calculadorRuta: calculadorRutasDietas.crearCalculadorRutasDietasPresentacion({
        contextoActor: contextos.dietas,
        capacidades: CAPACIDADES_AUTOSERVICIO_DIETAS,
      }),
      descargarRecibo,
      visorRuta: visorRutaDietas,
      origenComprobacion,
    });
    catalogo = catalogoDemo.obtenerCatalogoModulosPresentacion();
    composicion = Object.freeze({ contextos, cronos, dietas });
    return contextos.bolsa;
  }

  async function cargarInterno() {
    desmontarVistaActual();
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
    if (vista === "cronos") return composicion?.cronos !== undefined;
    if (vista === "dietas") return composicion?.dietas !== undefined;
    return false;
  }

  function resolverAcceso(clave, bolsaDisponible = true) {
    if (clave === "bolsa") return Object.freeze({ disponible: bolsaDisponible, vista: "resumen" });
    if (clave === "cronos" && vistaDisponible("cronos")) {
      return Object.freeze({ disponible: true, vista: "cronos" });
    }
    if (clave === "dietas" && vistaDisponible("dietas")) {
      return Object.freeze({ disponible: true, vista: "dietas" });
    }
    return Object.freeze({ disponible: false, vista: "" });
  }

  function renderizarNavegacion(bolsaDisponible = true, moduloActivo = "portal") {
    const catalogoVisible = moduloActivo === "portal"
      ? catalogo
      : catalogo.filter((modulo) => modulo.clave === moduloActivo);
    return renderizarNavegacionModulos({
      catalogo: catalogoVisible,
      resolverAcceso: (clave) => resolverAcceso(clave, bolsaDisponible),
      escaparHTML,
    });
  }

  async function montarVista(vista, raiz) {
    if (!VISTAS_MODULOS_PERSONALES.has(vista) || !vistaDisponible(vista)) return false;
    if (!raiz || typeof raiz.replaceChildren !== "function") {
      throw new TypeError("raíz del módulo no válida");
    }
    desmontarVistaActual();
    const montaje = ++secuenciaMontaje;
    raiz.innerHTML = '<section class="panel"><div class="cuerpo-panel" role="status">Cargando módulo…</div></section>';

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
