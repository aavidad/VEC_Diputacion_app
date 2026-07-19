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

export function crearVisorRutaDietas({ entorno = globalThis } = {}) {
  return Object.freeze({
    montar({ raiz, descriptor } = {}) {
      const datos = validarDescriptorMapa(descriptor);
      if (!raiz || typeof raiz.querySelector !== "function") throw new TypeError("raíz de mapa de Dietas no válida");
      const lienzo = raiz.querySelector("[data-dietas-mapa-canvas]");
      const estado = raiz.querySelector("[data-dietas-mapa-estado]");
      const atribucion = raiz.querySelector("[data-dietas-mapa-atribucion]");
      if (!leafletLocalDisponible(entorno)) {
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
        entorno.L.tileLayer(PLANTILLA_TESELAS_OSM_INTERNA, {
          maxZoom: 19,
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
          marcador.bindTooltip?.(`${indice + 1}. ${parada.etiqueta}`);
        });
        const limites = linea.getBounds?.();
        if (limites?.isValid?.()) mapa.fitBounds(limites, { padding: [24, 24] });
        lienzo.dataset.modoMapa = "openstreetmap_interno";
        if (atribucion) atribucion.hidden = false;
        if (estado) estado.textContent = "OpenStreetMap cargado desde teselas servidas en la red interna.";
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
        if (estado) estado.textContent = "Croquis SVG sintético DEMO. El visor OpenStreetMap se activa al desplegar Leaflet y las teselas en la red interna.";
        return resultadoRespaldo();
      }
    },
  });
}
