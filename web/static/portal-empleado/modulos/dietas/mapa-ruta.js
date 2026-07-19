/**
 * Adaptador cartográfico del navegador.
 *
 * Solo admite OpenStreetMap servido por el mismo despliegue. No descarga
 * Leaflet ni consulta teselas, geocodificadores o rutas de terceros. Si la
 * librería versionada localmente no está disponible conserva el croquis SVG
 * seguro que la vista entrega como respaldo.
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

function resultadoRespaldo() {
  return Object.freeze({ modo: "croquis_svg", desmontar() {} });
}

export function crearVisorRutaDietas({
  entorno = globalThis,
  permitirTeselas = false,
  mensajes = MENSAJES_DIETAS_ES,
} = {}) {
  if (typeof permitirTeselas !== "boolean") throw new TypeError("configuración de teselas de Dietas no válida");
  const t = crearTraductorDietas(mensajes);
  return Object.freeze({
    montar({ raiz, descriptor } = {}) {
      const datos = validarDescriptorMapa(descriptor);
      if (!raiz || typeof raiz.querySelector !== "function") throw new TypeError("raíz de mapa de Dietas no válida");
      const lienzo = raiz.querySelector("[data-dietas-mapa-canvas]");
      const estado = raiz.querySelector("[data-dietas-mapa-estado]");
      const atribucion = raiz.querySelector("[data-dietas-mapa-atribucion]");
      // La presentación aislada no compone clientes de red. Mantiene el SVG
      // hasta que el despliegue interno declare expresamente que el proxy de
      // teselas same-origin está operativo; así no genera 404 ni aparenta un
      // mapa cartográfico que el entorno todavía no sirve.
      if (!permitirTeselas || !leafletLocalDisponible(entorno)) {
        if (atribucion) atribucion.hidden = true;
        return resultadoRespaldo();
      }
      if (!lienzo) throw new TypeError("lienzo de mapa de Dietas no disponible");

      const respaldo = lienzo.innerHTML;
      let mapa = null;
      try {
        lienzo.replaceChildren();
        mapa = entorno.L.map(lienzo, {
          scrollWheelZoom: false,
          attributionControl: true,
        });
        mapa.attributionControl?.setPrefix?.(false);
        entorno.L.tileLayer(PLANTILLA_TESELAS_OSM_INTERNA, {
          maxNativeZoom: 14,
          maxZoom: 14,
          attribution: ATRIBUCION_OSM_INTERNA,
        }).addTo(mapa);
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
        lienzo.dataset.modoMapa = "openstreetmap_interno";
        if (atribucion) {
          // La atribución interactiva enlazada la aporta Leaflet. Este texto
          // adicional sigue siendo legible si el control se oculta al imprimir.
          atribucion.textContent = "© OpenStreetMap contributors · © OpenMapTiles · servido en red interna";
          atribucion.hidden = false;
        }
        if (estado) estado.textContent = t("mapa_nota_osm_interno");
        return Object.freeze({
          modo: "openstreetmap_interno",
          desmontar() {
            mapa?.remove?.();
            mapa = null;
          },
        });
      } catch {
        mapa?.remove?.();
        mapa = null;
        lienzo.innerHTML = respaldo;
        delete lienzo.dataset.modoMapa;
        if (atribucion) atribucion.hidden = true;
        if (estado) estado.textContent = t(datos.geometria.origen === "osrm_interno"
          ? "mapa_nota_osrm_local" : "mapa_nota_fallback");
        return resultadoRespaldo();
      }
    },
  });
}
