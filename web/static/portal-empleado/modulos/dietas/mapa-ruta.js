/**
 * Adaptador cartográfico del navegador.
 *
 * Solo admite OpenStreetMap servido por el mismo despliegue. No descarga
 * Leaflet ni consulta teselas, geocodificadores o rutas de terceros. Si la
 * biblioteca o el servicio interno no están disponibles muestra un estado
 * textual; nunca dibuja una ruta simulada con apariencia cartográfica.
 */
import {
  ATRIBUCION_OSM_INTERNA,
  PLANTILLA_TESELAS_OSM_INTERNA,
  validarGeometriaRutaDietas,
} from "./contrato.js";
import { MENSAJES_DIETAS_ES, crearTraductorDietas } from "./i18n.js";

function validarDescriptorMapa(descriptor) {
  if (!descriptor || typeof descriptor !== "object" || Array.isArray(descriptor)
    || descriptor.proveedor !== "openstreetmap" || descriptor.despliegue !== "red_interna"
    || descriptor.plantilla_teselas !== PLANTILLA_TESELAS_OSM_INTERNA
    || descriptor.atribucion !== ATRIBUCION_OSM_INTERNA) {
    throw new TypeError("descriptor de mapa de Dietas no válido");
  }
  return Object.freeze({
    ...descriptor,
    geometria: validarGeometriaRutaDietas(descriptor.geometria),
  });
}

function leafletLocalDisponible(entorno) {
  const leaflet = entorno?.L;
  return Boolean(leaflet && typeof leaflet.map === "function"
    && typeof leaflet.tileLayer === "function" && typeof leaflet.polyline === "function"
    && (typeof leaflet.circleMarker === "function" || typeof leaflet.marker === "function"));
}

function resultadoNoDisponible() {
  return Object.freeze({ modo: "mapa_no_disponible", desmontar() {} });
}

const MAXIMO_ERRORES_TESELA = 3;
const TIEMPO_ESPERA_TESELAS_MS = 7_000;

function mostrarMapaNoDisponible(lienzo, estado, atribucion, t) {
  if (lienzo) {
    lienzo.replaceChildren?.();
    if (lienzo.dataset) delete lienzo.dataset.modoMapa;
    const documento = lienzo.ownerDocument;
    if (documento && typeof documento.createElement === "function" && typeof lienzo.append === "function") {
      const aviso = documento.createElement("p");
      aviso.className = "dietas-mapa-espera";
      aviso.setAttribute("role", "status");
      aviso.textContent = t("mapa_nota_no_disponible");
      lienzo.append(aviso);
    } else {
      lienzo.textContent = t("mapa_nota_no_disponible");
    }
  }
  if (atribucion) atribucion.hidden = true;
  if (estado) estado.textContent = t("mapa_nota_no_disponible");
  return resultadoNoDisponible();
}

function traducirControlesZoom(lienzo, t) {
  const controles = [
    [".leaflet-control-zoom-in", "mapa_zoom_acercar"],
    [".leaflet-control-zoom-out", "mapa_zoom_alejar"],
  ];
  controles.forEach(([selector, clave]) => {
    const control = lienzo?.querySelector?.(selector);
    if (!control || typeof control.setAttribute !== "function") return;
    const etiqueta = t(clave);
    control.setAttribute("title", etiqueta);
    control.setAttribute("aria-label", etiqueta);
  });
}

export function crearVisorRutaDietas({
  entorno = globalThis,
  permitirTeselas = false,
  mensajes = MENSAJES_DIETAS_ES,
  tiempoEsperaMs = TIEMPO_ESPERA_TESELAS_MS,
} = {}) {
  if (typeof permitirTeselas !== "boolean") throw new TypeError("configuración de teselas de Dietas no válida");
  if (!Number.isSafeInteger(tiempoEsperaMs) || tiempoEsperaMs < 10 || tiempoEsperaMs > 15_000) {
    throw new TypeError("tiempo de espera de teselas no válido");
  }
  const t = crearTraductorDietas(mensajes);
  return Object.freeze({
    montar({ raiz, descriptor } = {}) {
      const datos = validarDescriptorMapa(descriptor);
      if (!raiz || typeof raiz.querySelector !== "function") throw new TypeError("raíz de mapa de Dietas no válida");
      const lienzo = raiz.querySelector("[data-dietas-mapa-canvas]");
      const estado = raiz.querySelector("[data-dietas-mapa-estado]");
      const atribucion = raiz.querySelector("[data-dietas-mapa-atribucion]");
      // Solo una geometría acreditada por el mediador OSRM puede adquirir
      // apariencia de mapa real. La geometría sintética sigue siendo útil en
      // pruebas de dominio, pero nunca se monta sobre teselas cartográficas.
      if (datos.geometria.origen !== "osrm_interno") {
        return mostrarMapaNoDisponible(lienzo, estado, atribucion, t);
      }
      // La falta de Leaflet o de la concesión de teselas queda visible y
      // falla cerrada: no se construye un mapa sintético que pueda confundirse
      // con la geometría calculada por carretera.
      if (!permitirTeselas || !leafletLocalDisponible(entorno)) {
        return mostrarMapaNoDisponible(lienzo, estado, atribucion, t);
      }
      if (!lienzo) throw new TypeError("lienzo de mapa de Dietas no disponible");

      let mapa = null;
      let capaTeselas = null;
      let temporizador = null;
      let modo = "mapa_cargando";
      let terminado = false;
      let desmontado = false;
      let erroresTesela = 0;
      const programar = typeof entorno?.setTimeout === "function"
        ? entorno.setTimeout.bind(entorno) : globalThis.setTimeout.bind(globalThis);
      const cancelarTemporizador = typeof entorno?.clearTimeout === "function"
        ? entorno.clearTimeout.bind(entorno) : globalThis.clearTimeout.bind(globalThis);
      const retirarEscuchas = () => {
        capaTeselas?.off?.("load", alCargarTeselas);
        capaTeselas?.off?.("tileerror", alFallarTesela);
      };
      const limpiarTemporizador = () => {
        if (temporizador !== null) cancelarTemporizador(temporizador);
        temporizador = null;
      };
      const declararNoDisponible = () => {
        if (terminado || desmontado) return;
        terminado = true;
        modo = "mapa_no_disponible";
        limpiarTemporizador();
        retirarEscuchas();
        mapa?.remove?.();
        mapa = null;
        mostrarMapaNoDisponible(lienzo, estado, atribucion, t);
      };
      function alCargarTeselas() {
        if (terminado || desmontado) return;
        terminado = true;
        modo = "openstreetmap_interno";
        limpiarTemporizador();
        retirarEscuchas();
        lienzo.dataset.modoMapa = "openstreetmap_interno";
        if (estado) estado.textContent = t("mapa_nota_osm_interno");
      }
      function alFallarTesela() {
        if (terminado || desmontado) return;
        erroresTesela += 1;
        if (erroresTesela >= MAXIMO_ERRORES_TESELA) declararNoDisponible();
      }
      try {
        lienzo.replaceChildren();
        lienzo.dataset.modoMapa = "mapa_cargando";
        if (estado) estado.textContent = t("mapa_cargando_interno");
        if (atribucion) atribucion.hidden = true;
        mapa = entorno.L.map(lienzo, {
          scrollWheelZoom: false,
          attributionControl: true,
        });
        traducirControlesZoom(lienzo, t);
        mapa.attributionControl?.setPrefix?.(false);
        capaTeselas = entorno.L.tileLayer(PLANTILLA_TESELAS_OSM_INTERNA, {
          maxNativeZoom: 14,
          maxZoom: 14,
          attribution: ATRIBUCION_OSM_INTERNA,
        });
        if (!capaTeselas || typeof capaTeselas.on !== "function"
          || typeof capaTeselas.addTo !== "function") {
          throw new TypeError("capa de teselas interna no observable");
        }
        capaTeselas.on("load", alCargarTeselas);
        capaTeselas.on("tileerror", alFallarTesela);
        capaTeselas.addTo(mapa);
        const linea = entorno.L.polyline(datos.geometria.trazado, {
          color: "#155e75",
          weight: 5,
          opacity: 0.9,
        }).addTo(mapa);
        const marcadores = new Set();
        datos.geometria.paradas.forEach((parada, indice) => {
          const clave = `${parada.latitud}:${parada.longitud}:${parada.etiqueta}`;
          if (marcadores.has(clave)) return;
          marcadores.add(clave);
          const crearMarcador = typeof entorno.L.circleMarker === "function"
            ? entorno.L.circleMarker : entorno.L.marker;
          const marcador = crearMarcador.call(entorno.L, [parada.latitud, parada.longitud], {
            radius: 6,
            color: "#0f172a",
            fillColor: "#ffffff",
            fillOpacity: 1,
          }).addTo(mapa);
          // Leaflet interpreta las cadenas de tooltip como HTML. Se entrega un
          // nodo cuyo contenido se fija con textContent para que una etiqueta
          // gobernada nunca pueda convertirse en marcado ejecutable.
          const documento = lienzo.ownerDocument;
          if (documento && typeof documento.createElement === "function") {
            const textoTooltip = documento.createElement("span");
            textoTooltip.textContent = `${indice + 1}. ${parada.etiqueta}`;
            marcador.bindTooltip?.(textoTooltip);
          }
        });
        const limites = linea.getBounds?.();
        if (limites?.isValid?.()) mapa.fitBounds(limites, { padding: [24, 24] });
        if (atribucion) {
          // La atribución interactiva y enlazada ya la aporta Leaflet dentro
          // del mapa. Se conserva el texto alternativo en el DOM para salida
          // documental, pero oculto en pantalla para no duplicarlo.
          atribucion.textContent = "© OpenStreetMap contributors · © OpenMapTiles · servido en red interna";
          atribucion.hidden = true;
        }
        if (!terminado) {
          temporizador = programar(declararNoDisponible, tiempoEsperaMs);
        }
        return Object.freeze({
          get modo() { return modo; },
          desmontar() {
            if (desmontado) return;
            desmontado = true;
            limpiarTemporizador();
            retirarEscuchas();
            mapa?.remove?.();
            mapa = null;
          },
        });
      } catch {
        limpiarTemporizador();
        retirarEscuchas();
        mapa?.remove?.();
        mapa = null;
        return mostrarMapaNoDisponible(lienzo, estado, atribucion, t);
      }
    },
  });
}
