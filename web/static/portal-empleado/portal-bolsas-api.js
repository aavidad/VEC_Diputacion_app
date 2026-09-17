/**
 * Cliente HTTP seguro para la consulta interna de Bolsas y Candidatos.
 *
 * Sigue la política de seguridad DEC-053:
 * - credentials: "omit" en toda llamada.
 * - Accept: "application/json".
 * - Validación exhaustiva con los contratos de portal-bolsas-contrato.js.
 */

import {
  validarRespuestaBolsas,
  validarRespuestaCandidatosBolsa,
} from "./portal-bolsas-contrato.js";

export const RUTA_BOLSAS = "/api/vec/bolsa/bolsas";

export function rutaCandidatosBolsa(bolsaRef, { estado = "", texto = "", cursor = "", limite = 50 } = {}) {
  const parametros = new URLSearchParams();
  if (estado && estado.trim() !== "") parametros.set("estado", estado.trim());
  if (texto && texto.trim() !== "") parametros.set("texto", texto.trim());
  if (cursor && cursor.trim() !== "") parametros.set("cursor", cursor.trim());
  if (limite && Number.isSafeInteger(Number(limite))) parametros.set("limite", String(limite));

  const query = parametros.toString();
  const rutaBase = `${RUTA_BOLSAS}/${encodeURIComponent(bolsaRef)}/candidatos`;
  return query ? `${rutaBase}?${query}` : rutaBase;
}

export async function consultarBolsas({ fetchImpl = fetch } = {}) {
  try {
    const respuesta = await fetchImpl(RUTA_BOLSAS, {
      method: "GET",
      credentials: "omit",
      headers: { Accept: "application/json" },
    });

    if (!respuesta.ok) {
      if (respuesta.status === 401) {
        return { ok: false, status: 401, codigo: "no_autenticado", mensaje: "Se requiere una sesión interna autenticada." };
      }
      if (respuesta.status === 403) {
        return { ok: false, status: 403, codigo: "acceso_denegado", mensaje: "La sesión no dispone de permisos para consultar bolsas." };
      }
      if (respuesta.status === 404) {
        return { ok: false, status: 404, codigo: "no_encontrado", mensaje: "El servicio de bolsas de trabajo no está disponible." };
      }
      return {
        ok: false,
        status: respuesta.status,
        codigo: "error_servidor",
        mensaje: `No se pudieron consultar las bolsas de trabajo (HTTP ${respuesta.status}).`,
      };
    }

    const envelope = await respuesta.json();
    const datos = validarRespuestaBolsas(envelope);
    return { ok: true, datos };
  } catch (error) {
    return {
      ok: false,
      status: 0,
      codigo: "error_red_o_contrato",
      mensaje: error instanceof Error ? error.message : "Error de comunicación con el servicio de bolsas.",
    };
  }
}

export async function consultarCandidatosBolsa(bolsaRef, opciones = {}, { fetchImpl = fetch } = {}) {
  if (typeof bolsaRef !== "string" || bolsaRef.trim() === "") {
    return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de bolsa no válida." };
  }

  const url = rutaCandidatosBolsa(bolsaRef, opciones);

  try {
    const respuesta = await fetchImpl(url, {
      method: "GET",
      credentials: "omit",
      headers: { Accept: "application/json" },
    });

    if (!respuesta.ok) {
      if (respuesta.status === 400) {
        return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: "Parámetros de consulta no válidos." };
      }
      if (respuesta.status === 401) {
        return { ok: false, status: 401, codigo: "no_autenticado", mensaje: "Se requiere una sesión interna autenticada." };
      }
      if (respuesta.status === 403) {
        return { ok: false, status: 403, codigo: "acceso_denegado", mensaje: "La sesión no dispone de permisos para consultar candidatos." };
      }
      if (respuesta.status === 404) {
        return { ok: false, status: 404, codigo: "no_encontrado", mensaje: "Bolsa de trabajo no encontrada." };
      }
      if (respuesta.status === 422) {
        return { ok: false, status: 422, codigo: "no_procesable", mensaje: "Consulta de candidatos no procesable." };
      }
      return {
        ok: false,
        status: respuesta.status,
        codigo: "error_servidor",
        mensaje: `No se pudieron consultar los candidatos (HTTP ${respuesta.status}).`,
      };
    }

    const envelope = await respuesta.json();
    const datos = validarRespuestaCandidatosBolsa(envelope);
    return { ok: true, datos };
  } catch (error) {
    return {
      ok: false,
      status: 0,
      codigo: "error_red_o_contrato",
      mensaje: error instanceof Error ? error.message : "Error de comunicación con el servicio de candidatos.",
    };
  }
}

export function crearControladorBolsas({ estado, renderizar, navegar }) {
  async function cargarBolsas() {
    estado.datosBolsas = { carga: "cargando", datos: null, error: "" };
    renderizar();
    const res = await consultarBolsas();
    if (res.ok) {
      estado.datosBolsas = { carga: "listo", datos: res.datos, error: "" };
    } else if (res.status === 403) {
      estado.datosBolsas = { carga: "denegado", datos: null, error: res.mensaje };
    } else {
      estado.datosBolsas = { carga: "error", datos: null, error: res.mensaje };
    }
    renderizar();
  }

  async function cargarCandidatosBolsa(bolsaRef, { cursor = "" } = {}) {
    if (!bolsaRef) return;
    estado.bolsaSeleccionada = bolsaRef;
    estado.datosCandidatos = { carga: "cargando", datos: null, error: "" };
    renderizar();
    const res = await consultarCandidatosBolsa(bolsaRef, {
      estado: estado.filtrosBolsa?.estado || "",
      texto: estado.filtrosBolsa?.texto || "",
      cursor,
    });
    if (res.ok) {
      estado.datosCandidatos = { carga: "listo", datos: res.datos, error: "" };
    } else if (res.status === 403) {
      estado.datosCandidatos = { carga: "denegado", datos: null, error: res.mensaje };
    } else {
      estado.datosCandidatos = { carga: "error", datos: null, error: res.mensaje };
    }
    renderizar();
  }

  function instalar() {
    document.addEventListener("click", (evento) => {
      const botonVer = evento.target?.closest?.('[data-accion="ver-bolsa"]');
      if (botonVer) {
        evento.preventDefault();
        const ref = botonVer.dataset.bolsaRef;
        if (ref) {
          estado.bolsaSeleccionada = ref;
          estado.filtrosBolsa = { estado: "", texto: "" };
          navegar("bolsa-candidatos");
          void cargarCandidatosBolsa(ref);
        }
        return;
      }
      const botonAccion = evento.target?.closest?.("[data-bolsa-accion]");
      if (!botonAccion) return;
      const accion = botonAccion.dataset.bolsaAccion;
      if (accion === "reintentar-bolsas") {
        evento.preventDefault();
        void cargarBolsas();
      } else if (accion === "reintentar-candidatos") {
        evento.preventDefault();
        void cargarCandidatosBolsa(estado.bolsaSeleccionada);
      } else if (accion === "limpiar-filtros") {
        evento.preventDefault();
        estado.filtrosBolsa = { estado: "", texto: "" };
        void cargarCandidatosBolsa(estado.bolsaSeleccionada);
      } else if (accion === "pagina-siguiente") {
        evento.preventDefault();
        const cursor = botonAccion.dataset.cursor || "";
        void cargarCandidatosBolsa(estado.bolsaSeleccionada, { cursor });
      }
    });

    document.addEventListener("submit", (evento) => {
      const form = evento.target?.closest?.('[data-bolsa-form="filtros"]');
      if (!form) return;
      evento.preventDefault();
      const datos = new FormData(form);
      estado.filtrosBolsa = {
        estado: datos.get("estado") || "",
        texto: datos.get("texto") || "",
      };
      void cargarCandidatosBolsa(estado.bolsaSeleccionada);
    });
  }

  return Object.freeze({
    cargarBolsas,
    cargarCandidatosBolsa,
    instalar,
  });
}
