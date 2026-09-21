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

export async function consultarBolsas({ fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(RUTA_BOLSAS, {
      method: "GET",
      credentials: "omit",
      signal,
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

export async function consultarCandidatosBolsa(bolsaRef, opciones = {}, { fetchImpl = fetch, signal } = {}) {
  if (typeof bolsaRef !== "string" || bolsaRef.trim() === "") {
    return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de bolsa no válida." };
  }

  const url = rutaCandidatosBolsa(bolsaRef, opciones);

  try {
    const respuesta = await fetchImpl(url, {
      method: "GET",
      credentials: "omit",
      signal,
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

export async function cambiarSituacionCandidato(bolsaRef, participacionRef, payload, { fetchImpl = fetch } = {}) {
  if (!bolsaRef || !participacionRef || !payload?.situacion || !payload?.motivo || !payload?.clave_idempotencia) {
    return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: "Faltan los datos obligatorios del cambio de situación." };
  }
  try {
    const respuesta = await fetchImpl(`${rutaCandidatosBolsa(bolsaRef)}/${segmentoRuta(participacionRef)}/situacion`, {
      method: "POST", credentials: "omit",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": payload.clave_idempotencia },
      body: JSON.stringify({ situacion: payload.situacion, motivo: payload.motivo, fecha_disponible: payload.fecha_disponible || null }),
    });
    const cuerpo = await respuesta.json().catch(() => ({}));
    if (respuesta.ok && cuerpo?.data?.recibo_ref) return { ok: true, datos: cuerpo.data };
    return { ok: false, status: respuesta.status, codigo: cuerpo?.error?.codigo || "error_servidor", mensaje: respuesta.status === 403 ? "La sesión no dispone de permiso para cambiar esta situación." : "No se pudo registrar el cambio de situación." };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: error instanceof Error ? error.message : "Error de comunicación." };
  }
}

export async function consultarContactosCandidato(participacionRef, { fetchImpl = fetch, signal } = {}) {
  if (typeof participacionRef !== "string" || participacionRef.trim() === "") {
    return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de candidato no válida." };
  }

  const url = `/api/vec/bolsa/candidatos/${segmentoRuta(participacionRef)}/contactos`;

  try {
    const respuesta = await fetchImpl(url, {
      method: "GET",
      credentials: "omit",
      signal,
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

export function crearControladorBolsas({ estado, renderizar, navegar, obtenerFuenteLectura = () => null, documento = globalThis.document }) {
  const controladoresLectura = new Map();
  function fuenteLectura() {
    const fuente = obtenerFuenteLectura();
    return fuente && typeof fuente === "object" ? fuente : null;
  }

  function iniciarLectura(clave) {
    controladoresLectura.get(clave)?.abort();
    const controlador = new AbortController();
    controladoresLectura.set(clave, controlador);
    return controlador;
  }

  function lecturaVigente(clave, controlador) {
    return !controlador.signal.aborted && controladoresLectura.get(clave) === controlador;
  }

  function terminarLectura(clave, controlador) {
    if (controladoresLectura.get(clave) === controlador) controladoresLectura.delete(clave);
  }

  function limpiarEstadoCarga(clave) {
    if (clave === "bolsas" && estado.datosBolsas?.carga === "cargando") estado.datosBolsas = null;
    if (clave === "candidatos" && estado.datosCandidatos?.carga === "cargando") estado.datosCandidatos = null;
    if (clave === "contactos" && estado.modalContactos?.carga === "cargando") estado.modalContactos = null;
  }

  async function resolverLectura(clave, controlador, operacion) {
    try {
      return await operacion();
    } catch (error) {
      if (!lecturaVigente(clave, controlador)) return null;
      if (error?.name === "AbortError") {
        terminarLectura(clave, controlador);
        limpiarEstadoCarga(clave);
        return null;
      }
      return {
        ok: false,
        status: 0,
        codigo: "error_red_o_contrato",
        mensaje: error instanceof Error ? error.message : "Error de comunicación con Bolsa.",
      };
    }
  }

  function cancelarPeticiones() {
    for (const controlador of controladoresLectura.values()) controlador.abort();
    controladoresLectura.clear();
    for (const clave of ["bolsas", "candidatos", "contactos"]) limpiarEstadoCarga(clave);
  }

  async function cargarBolsas() {
    const controlador = iniciarLectura("bolsas");
    estado.datosBolsas = { carga: "cargando", datos: null, error: "" };
    renderizar();
    const fuente = fuenteLectura();
    const res = await resolverLectura("bolsas", controlador, () => fuente?.consultarBolsas
      ? fuente.consultarBolsas({ signal: controlador.signal })
      : consultarBolsas({ signal: controlador.signal }));
    if (res === null || !lecturaVigente("bolsas", controlador)) return;
    terminarLectura("bolsas", controlador);
    if (res.ok) {
      estado.datosBolsas = { carga: "listo", datos: res.datos, error: "" };
    } else if (res.status === 403) {
      estado.datosBolsas = { carga: "denegado", datos: null, error: res.mensaje };
    } else {
      estado.datosBolsas = { carga: "error", datos: null, error: res.mensaje };
    }
    renderizar();
  }

  async function cargarCandidatosBolsa(bolsaRef, { cursor = "", enfocarDestino = false } = {}) {
    if (!bolsaRef) return;
    const controlador = iniciarLectura("candidatos");
    estado.bolsaSeleccionada = bolsaRef;
    estado.datosCandidatos = { carga: "cargando", datos: null, error: "" };
    renderizar();
    const fuente = fuenteLectura();
    const opciones = {
      estado: estado.filtrosBolsa?.estado || "",
      texto: estado.filtrosBolsa?.texto || "",
      cursor,
    };
    const res = await resolverLectura("candidatos", controlador, () => fuente?.consultarCandidatosBolsa
      ? fuente.consultarCandidatosBolsa(bolsaRef, opciones, { signal: controlador.signal })
      : consultarCandidatosBolsa(bolsaRef, opciones, { signal: controlador.signal }));
    if (res === null || !lecturaVigente("candidatos", controlador)) return;
    terminarLectura("candidatos", controlador);
    if (res.ok) {
      estado.datosCandidatos = { carga: "listo", datos: res.datos, error: "" };
    } else if (res.status === 403) {
      estado.datosCandidatos = { carga: "denegado", datos: null, error: res.mensaje };
    } else {
      estado.datosCandidatos = { carga: "error", datos: null, error: res.mensaje };
    }
    renderizar();
    if (enfocarDestino) {
      documento.querySelector("[data-bolsa-b5-destino='true']")?.focus?.();
    }
  }

  async function abrirContactos(participacionRef, nombreVisible = "") {
    if (!participacionRef) return;
    const controlador = iniciarLectura("contactos");
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
    const res = await resolverLectura("contactos", controlador, () => fuente?.consultarContactosCandidato
      ? fuente.consultarContactosCandidato(participacionRef, { signal: controlador.signal })
      : consultarContactosCandidato(participacionRef, { signal: controlador.signal }));
    if (res === null || !lecturaVigente("contactos", controlador)) return;
    terminarLectura("contactos", controlador);
    if (res.ok) {
      estado.modalContactos.carga = "listo";
      estado.modalContactos.contactos = res.datos.contactos;
    } else {
      estado.modalContactos.carga = "error";
      estado.modalContactos.error = res.mensaje;
    }
    renderizar();
  }

  function abrirFicha(participacionRef) {
    const datos = estado.datosCandidatos?.datos;
    const candidato = datos?.candidatos?.find((item) => item.participacion_ref === participacionRef);
    if (!candidato || !datos?.bolsa) return;
    estado.modalFicha = { abierto: true, candidato, bolsa: datos.bolsa };
    renderizar();
    documento.querySelector("[data-bolsa-ficha-inline='true']")?.focus?.();
  }

  function abrirCambioSituacion() {
    if (!estado.modalFicha) return;
    estado.modalFicha.cambioSituacion = true;
    estado.modalFicha.errorCambioSituacion = "";
    renderizar();
  }

  function cerrarFicha() {
    const participacionRef = estado.modalFicha?.candidato?.participacion_ref;
    estado.modalFicha = null;
    renderizar();
    if (!participacionRef) return;
    const controles = documento.querySelectorAll('[data-bolsa-accion="abrir-ficha"][data-bolsa-control-principal="true"]');
    for (const control of controles) {
      if (control.dataset.participacionRef === participacionRef) {
        control.focus?.();
        return;
      }
    }
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
    documento.addEventListener("click", (evento) => {
      const botonVer = evento.target?.closest?.('[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]');
      if (botonVer) {
        evento.preventDefault();
        const ref = botonVer.dataset.bolsaRef;
        if (ref) {
          estado.bolsaSeleccionada = ref;
          estado.filtrosBolsa = { estado: "", texto: "" };
          navegar("bolsa-candidatos");
          void cargarCandidatosBolsa(ref, { enfocarDestino: true });
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
      } else if (accion === "filtrar-estado") {
        evento.preventDefault();
        estado.filtrosBolsa = { ...estado.filtrosBolsa, estado: botonAccion.dataset.estado || "" };
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
      } else if (accion === "abrir-ficha") {
        evento.preventDefault();
        abrirFicha(botonAccion.dataset.participacionRef);
      } else if (accion === "abrir-cambio-situacion") {
        evento.preventDefault();
        abrirCambioSituacion();
      } else if (accion === "cerrar-ficha") {
        evento.preventDefault();
        cerrarFicha();
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

    documento.addEventListener("submit", (evento) => {
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

      const formCambioSituacion = evento.target?.closest?.('[data-bolsa-form="cambio-situacion"]');
      if (formCambioSituacion) {
        evento.preventDefault();
        const datos = new FormData(formCambioSituacion);
        const situacion = String(datos.get("situacion") || "");
        const motivo = String(datos.get("motivo") || "").trim();
        const fecha = String(datos.get("fecha_disponible") || "");
        if (!situacion || !motivo || (situacion === "disponible_desde" && !fecha)) {
          if (estado.modalFicha) { estado.modalFicha.errorCambioSituacion = "Indique destino, motivo y la fecha futura cuando corresponda."; renderizar(); }
          return;
        }
        const clave = globalThis.crypto?.randomUUID?.() || `situacion-${Date.now()}-${Math.random().toString(16).slice(2)}`;
        void cambiarSituacionCandidato(estado.bolsaSeleccionada, formCambioSituacion.dataset.participacionRef, {
          situacion, motivo, fecha_disponible: fecha ? new Date(fecha).toISOString() : null, clave_idempotencia: clave,
        }).then((res) => {
          if (res.ok) { estado.modalFicha = null; void cargarCandidatosBolsa(estado.bolsaSeleccionada); }
          else if (estado.modalFicha) { estado.modalFicha.errorCambioSituacion = res.mensaje; renderizar(); }
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

    documento.addEventListener("keydown", (evento) => {
      if (evento.key === "Escape" && estado.modalFicha?.abierto) {
        evento.preventDefault();
        cerrarFicha();
      }
    });
  }

  return Object.freeze({
    cancelarPeticiones,
    cargarBolsas,
    cargarCandidatosBolsa,
    abrirFicha,
    cerrarFicha,
    abrirContactos,
    cerrarContactos,
    abrirLlamar,
    cerrarLlamar,
    abrirResultado,
    cerrarResultado,
    instalar,
  });
}
