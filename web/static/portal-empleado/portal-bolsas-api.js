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
  validarRespuestaContactos,
  validarPayloadCrearLlamamiento,
  validarPayloadResultadoLlamamiento,
  construirEnvelopeAccionBolsa,
} from "./portal-bolsas-contrato.js";

export const RUTA_BOLSAS = "/api/vec/bolsa/bolsas";

// El enrutador del servidor solo acepta rutas canónicas (sin secuencias
// porcentuales): las referencias llevan ":" y "-", legales en un segmento de
// ruta, así que se envían sin escapar y solo se escapa lo que no es legal.
function segmentoRuta(referencia) {
  return encodeURIComponent(String(referencia ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaCandidatosBolsa(bolsaRef, { estado = "", texto = "", cursor = "", limite = 50 } = {}) {
  const parametros = new URLSearchParams();
  if (estado && estado.trim() !== "") parametros.set("estado", estado.trim());
  if (texto && texto.trim() !== "") parametros.set("texto", texto.trim());
  if (cursor && cursor.trim() !== "") parametros.set("cursor", cursor.trim());
  if (limite && Number.isSafeInteger(Number(limite))) parametros.set("limite", String(limite));

  const query = parametros.toString();
  const rutaBase = `${RUTA_BOLSAS}/${segmentoRuta(bolsaRef)}/candidatos`;
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

export async function consultarContactosCandidato(participacionRef, { fetchImpl = fetch } = {}) {
  if (typeof participacionRef !== "string" || participacionRef.trim() === "") {
    return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de candidato no válida." };
  }

  const url = `/api/vec/bolsa/candidatos/${segmentoRuta(participacionRef)}/contactos`;

  try {
    const respuesta = await fetchImpl(url, {
      method: "GET",
      credentials: "omit",
      headers: { Accept: "application/json" },
    });

    if (!respuesta.ok) {
      if (respuesta.status === 401) {
        return { ok: false, status: 401, codigo: "no_autenticado", mensaje: "Se requiere una sesión interna autenticada." };
      }
      if (respuesta.status === 403) {
        return { ok: false, status: 403, codigo: "acceso_denegado", mensaje: "La sesión no dispone de permisos para consultar contactos." };
      }
      if (respuesta.status === 404) {
        return { ok: false, status: 404, codigo: "no_encontrado", mensaje: "Candidato o contactos no encontrados." };
      }
      return {
        ok: false,
        status: respuesta.status,
        codigo: "error_servidor",
        mensaje: `No se pudieron consultar los contactos (HTTP ${respuesta.status}).`,
      };
    }

    const envelope = await respuesta.json();
    const datos = validarRespuestaContactos(envelope);
    return { ok: true, datos };
  } catch (error) {
    return {
      ok: false,
      status: 0,
      codigo: "error_red_o_contrato",
      mensaje: error instanceof Error ? error.message : "Error de comunicación al consultar contactos.",
    };
  }
}

export async function crearLlamamientoCandidato(participacionRef, payload, { fetchImpl = fetch } = {}) {
  if (typeof participacionRef !== "string" || participacionRef.trim() === "") {
    return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de candidato no válida." };
  }

  let payloadValidado;
  let envelopeAccion;
  try {
    payloadValidado = validarPayloadCrearLlamamiento(payload);
    envelopeAccion = construirEnvelopeAccionBolsa("crear_llamamiento", payloadValidado, { confirmacion: true });
  } catch (err) {
    return {
      ok: false,
      status: 400,
      codigo: "solicitud_invalida",
      mensaje: err instanceof Error ? err.message : "Datos de llamamiento no válidos.",
    };
  }

  const url = `/api/vec/bolsa/candidatos/${segmentoRuta(participacionRef)}/llamamientos`;

  try {
    const respuesta = await fetchImpl(url, {
      method: "POST",
      credentials: "omit",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify(envelopeAccion),
    });

    if (!respuesta.ok) {
      if (respuesta.status === 400) {
        return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: "Solicitud de llamamiento rechazada por el servidor." };
      }
      if (respuesta.status === 401) {
        return { ok: false, status: 401, codigo: "no_autenticado", mensaje: "Se requiere una sesión interna autenticada." };
      }
      if (respuesta.status === 403) {
        return { ok: false, status: 403, codigo: "acceso_denegado", mensaje: "Sin permiso para registrar llamamientos." };
      }
      if (respuesta.status === 404) {
        return { ok: false, status: 404, codigo: "no_encontrado", mensaje: "Candidato no encontrado para el llamamiento." };
      }
      if (respuesta.status === 409) {
        return { ok: false, status: 409, codigo: "conflicto", mensaje: "El aspirante no está disponible o ya tiene un llamamiento en curso." };
      }
      if (respuesta.status === 422) {
        return { ok: false, status: 422, codigo: "no_procesable", mensaje: "Los datos del llamamiento no cumplen las reglas de negocio." };
      }
      return {
        ok: false,
        status: respuesta.status,
        codigo: "error_servidor",
        mensaje: `No se pudo registrar el llamamiento (HTTP ${respuesta.status}).`,
      };
    }

    const envelope = await respuesta.json();
    return { ok: true, datos: envelope.data || envelope };
  } catch (error) {
    return {
      ok: false,
      status: 0,
      codigo: "error_red_o_contrato",
      mensaje: error instanceof Error ? error.message : "Error de comunicación al registrar llamamiento.",
    };
  }
}

export async function registrarResultadoLlamamiento(llamamientoRef, payload, { fetchImpl = fetch } = {}) {
  if (typeof llamamientoRef !== "string" || llamamientoRef.trim() === "") {
    return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de llamamiento no válida." };
  }

  let payloadValidado;
  let envelopeAccion;
  try {
    payloadValidado = validarPayloadResultadoLlamamiento(payload);
    envelopeAccion = construirEnvelopeAccionBolsa("registrar_resultado", payloadValidado, { confirmacion: true });
  } catch (err) {
    return {
      ok: false,
      status: 400,
      codigo: "solicitud_invalida",
      mensaje: err instanceof Error ? err.message : "Datos de resultado no válidos.",
    };
  }

  const url = `/api/vec/bolsa/llamamientos/${segmentoRuta(llamamientoRef)}/resultado`;

  try {
    const respuesta = await fetchImpl(url, {
      method: "POST",
      credentials: "omit",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify(envelopeAccion),
    });

    if (!respuesta.ok) {
      if (respuesta.status === 400) {
        return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: "Solicitud de resultado rechazada por el servidor." };
      }
      if (respuesta.status === 401) {
        return { ok: false, status: 401, codigo: "no_autenticado", mensaje: "Se requiere una sesión interna autenticada." };
      }
      if (respuesta.status === 403) {
        return { ok: false, status: 403, codigo: "acceso_denegado", mensaje: "Sin permiso para registrar resultado de llamamiento." };
      }
      if (respuesta.status === 404) {
        return { ok: false, status: 404, codigo: "no_encontrado", mensaje: "Llamamiento no encontrado." };
      }
      if (respuesta.status === 409) {
        return { ok: false, status: 409, codigo: "conflicto", mensaje: "El llamamiento ya tiene resultado o su estado no permite registrarlo." };
      }
      if (respuesta.status === 422) {
        return { ok: false, status: 422, codigo: "no_procesable", mensaje: "El resultado no es procesable según las reglas de bolsa." };
      }
      return {
        ok: false,
        status: respuesta.status,
        codigo: "error_servidor",
        mensaje: `No se pudo registrar el resultado (HTTP ${respuesta.status}).`,
      };
    }

    const envelope = await respuesta.json();
    return { ok: true, datos: envelope.data || envelope };
  } catch (error) {
    return {
      ok: false,
      status: 0,
      codigo: "error_red_o_contrato",
      mensaje: error instanceof Error ? error.message : "Error de comunicación al registrar resultado.",
    };
  }
}

export function crearControladorBolsas({ estado, renderizar, navegar, obtenerFuenteLectura = () => null }) {
  function fuenteLectura() {
    const fuente = obtenerFuenteLectura();
    return fuente && typeof fuente === "object" ? fuente : null;
  }

  async function cargarBolsas() {
    estado.datosBolsas = { carga: "cargando", datos: null, error: "" };
    renderizar();
    const fuente = fuenteLectura();
    const res = fuente?.consultarBolsas
      ? await fuente.consultarBolsas()
      : await consultarBolsas();
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
    const fuente = fuenteLectura();
    const opciones = {
      estado: estado.filtrosBolsa?.estado || "",
      texto: estado.filtrosBolsa?.texto || "",
      cursor,
    };
    const res = fuente?.consultarCandidatosBolsa
      ? await fuente.consultarCandidatosBolsa(bolsaRef, opciones)
      : await consultarCandidatosBolsa(bolsaRef, opciones);
    if (res.ok) {
      estado.datosCandidatos = { carga: "listo", datos: res.datos, error: "" };
    } else if (res.status === 403) {
      estado.datosCandidatos = { carga: "denegado", datos: null, error: res.mensaje };
    } else {
      estado.datosCandidatos = { carga: "error", datos: null, error: res.mensaje };
    }
    renderizar();
  }

  async function abrirContactos(participacionRef, nombreVisible = "") {
    if (!participacionRef) return;
    estado.modalContactos = {
      abierto: true,
      participacionRef,
      nombreVisible,
      carga: "cargando",
      contactos: [],
      error: "",
    };
    renderizar();
    const fuente = fuenteLectura();
    const res = fuente?.consultarContactosCandidato
      ? await fuente.consultarContactosCandidato(participacionRef)
      : await consultarContactosCandidato(participacionRef);
    if (res.ok) {
      estado.modalContactos.carga = "listo";
      estado.modalContactos.contactos = res.datos.contactos;
    } else {
      estado.modalContactos.carga = "error";
      estado.modalContactos.error = res.mensaje;
    }
    renderizar();
  }

  function cerrarContactos() {
    estado.modalContactos = null;
    renderizar();
  }

  function abrirLlamar(participacionRef, nombreVisible = "", orden = 0) {
    if (!participacionRef) return;
    estado.modalLlamar = {
      abierto: true,
      participacionRef,
      nombreVisible,
      orden,
      carga: "ocioso",
      error: "",
    };
    renderizar();
  }

  function cerrarLlamar() {
    estado.modalLlamar = null;
    renderizar();
  }

  function abrirResultado(llamamientoRef, participacionRef = "", nombreVisible = "", orden = 0) {
    if (!llamamientoRef) return;
    estado.modalResultado = {
      abierto: true,
      llamamientoRef,
      participacionRef,
      nombreVisible,
      orden,
      carga: "ocioso",
      error: "",
    };
    renderizar();
  }

  function cerrarResultado() {
    estado.modalResultado = null;
    renderizar();
  }

  function instalar() {
    document.addEventListener("click", (evento) => {
      const botonVer = evento.target?.closest?.('[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]');
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
      } else if (accion === "abrir-contactos") {
        evento.preventDefault();
        const ref = botonAccion.dataset.participacionRef;
        const nom = botonAccion.dataset.nombreVisible || "";
        void abrirContactos(ref, nom);
      } else if (accion === "cerrar-contactos") {
        evento.preventDefault();
        cerrarContactos();
      } else if (accion === "abrir-llamar") {
        evento.preventDefault();
        const ref = botonAccion.dataset.participacionRef;
        const nom = botonAccion.dataset.nombreVisible || "";
        const ord = Number(botonAccion.dataset.orden || 0);
        abrirLlamar(ref, nom, ord);
      } else if (accion === "cerrar-llamar") {
        evento.preventDefault();
        cerrarLlamar();
      } else if (accion === "abrir-resultado") {
        evento.preventDefault();
        const ref = botonAccion.dataset.llamamientoRef;
        const pref = botonAccion.dataset.participacionRef || "";
        const nom = botonAccion.dataset.nombreVisible || "";
        const ord = Number(botonAccion.dataset.orden || 0);
        abrirResultado(ref, pref, nom, ord);
      } else if (accion === "cerrar-resultado") {
        evento.preventDefault();
        cerrarResultado();
      }
    });

    document.addEventListener("submit", (evento) => {
      const formFiltros = evento.target?.closest?.('[data-bolsa-form="filtros"]');
      if (formFiltros) {
        evento.preventDefault();
        const datos = new FormData(formFiltros);
        estado.filtrosBolsa = {
          estado: datos.get("estado") || "",
          texto: datos.get("texto") || "",
        };
        void cargarCandidatosBolsa(estado.bolsaSeleccionada);
        return;
      }

      const formLlamar = evento.target?.closest?.('[data-bolsa-form="llamar"]');
      if (formLlamar) {
        evento.preventDefault();
        const datos = new FormData(formLlamar);
        const ref = formLlamar.dataset.participacionRef;
        if (!datos.get("confirmacion")) {
          if (estado.modalLlamar) {
            estado.modalLlamar.error = "Debe confirmar explícitamente el llamamiento.";
            renderizar();
          }
          return;
        }
        const canal = datos.get("canal") || "correo";
        const valorComunicado = datos.get("comunicado_en");
        const comunicadoEn = valorComunicado
          ? (valorComunicado.includes("Z") ? valorComunicado : new Date(valorComunicado).toISOString())
          : new Date().toISOString();
        const valorPlazo = datos.get("plazo_respuesta_hasta");
        const plazoRespuestaHasta = valorPlazo
          ? (valorPlazo.includes("Z") ? valorPlazo : (valorPlazo.includes("T") ? new Date(valorPlazo).toISOString() : `${valorPlazo}T23:59:59Z`))
          : new Date(Date.now() + 48 * 3600 * 1000).toISOString();
        const anotacion = datos.get("anotacion") || "";

        if (estado.modalLlamar) {
          estado.modalLlamar.carga = "enviando";
          estado.modalLlamar.error = "";
          renderizar();
        }

        void crearLlamamientoCandidato(ref, {
          canal,
          comunicado_en: comunicadoEn,
          plazo_respuesta_hasta: plazoRespuestaHasta,
          anotacion,
        }).then((res) => {
          if (res.ok) {
            estado.modalLlamar = null;
            void cargarCandidatosBolsa(estado.bolsaSeleccionada);
          } else {
            if (estado.modalLlamar) {
              estado.modalLlamar.carga = "error";
              estado.modalLlamar.error = res.mensaje;
              renderizar();
            }
          }
        });
        return;
      }

      const formResultado = evento.target?.closest?.('[data-bolsa-form="resultado"]');
      if (formResultado) {
        evento.preventDefault();
        const datos = new FormData(formResultado);
        const ref = formResultado.dataset.llamamientoRef;
        if (!datos.get("confirmacion")) {
          if (estado.modalResultado) {
            estado.modalResultado.error = "Debe confirmar explícitamente el registro de resultado.";
            renderizar();
          }
          return;
        }
        const resultadoClave = datos.get("resultado_clave");
        const anotacion = datos.get("anotacion") || "";

        if (estado.modalResultado) {
          estado.modalResultado.carga = "enviando";
          estado.modalResultado.error = "";
          renderizar();
        }

        void registrarResultadoLlamamiento(ref, {
          resultado_clave: resultadoClave,
          anotacion,
        }).then((res) => {
          if (res.ok) {
            estado.modalResultado = null;
            void cargarCandidatosBolsa(estado.bolsaSeleccionada);
          } else {
            if (estado.modalResultado) {
              estado.modalResultado.carga = "error";
              estado.modalResultado.error = res.mensaje;
              renderizar();
            }
          }
        });
        return;
      }
    });
  }

  return Object.freeze({
    cargarBolsas,
    cargarCandidatosBolsa,
    abrirContactos,
    cerrarContactos,
    abrirLlamar,
    cerrarLlamar,
    abrirResultado,
    cerrarResultado,
    instalar,
  });
}
