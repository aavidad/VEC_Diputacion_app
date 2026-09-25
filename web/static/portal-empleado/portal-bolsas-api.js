/**
 * Cliente HTTP seguro para la consulta interna de Bolsas y Candidatos.
 *
 * Sigue la política de seguridad DEC-053:
 * - credentials: "same-origin" en toda llamada.
 * - Accept: "application/json".
 * - Validación exhaustiva con los contratos de portal-bolsas-contrato.js.
 */

import {
  validarRespuestaBolsas,
  validarRespuestaCandidatosBolsa,
  validarRespuestaContactos,
  validarRespuestaEstadisticas,
} from "./portal-bolsas-contrato.js";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL, traducirBolsaInterna, traducirPortal } from "./portal-i18n.js?v=20260926-integracion-bolsa-ct-v1";
import { crearControladorOperacionesSituacion } from "./portal-bolsas-operaciones.js?v=20260926-integracion-bolsa-ct-v1";
import { crearControladorIntentosContacto } from "./portal-bolsas-intentos.js?v=20260926-integracion-bolsa-ct-v1";
import { crearControladorSanciones } from "./portal-bolsas-sanciones.js?v=20260926-sanciones-efectos-v1";
import { crearControladorCorreoLlamamiento } from "./portal-bolsas-correo.js?v=20260926-integracion-bolsa-ct-v1";
import { emitirLlamamiento, crearLlamamientoCandidato, registrarResultadoLlamamiento } from "./portal-llamamientos-operaciones-api.js?v=20260926-integracion-bolsa-ct-v1";
export { emitirLlamamiento, crearLlamamientoCandidato, registrarResultadoLlamamiento } from "./portal-llamamientos-operaciones-api.js?v=20260926-integracion-bolsa-ct-v1";
import { crearControladorOrigenContacto } from "./portal-bolsas-contacto-origen.js?v=20260926-integracion-bolsa-ct-v1";

export const RUTA_BOLSAS = "/api/vec/bolsa/bolsas";
export const RUTA_ESTADISTICAS_BOLSA = "/api/vec/bolsa/estadisticas";
export const RUTA_PLAZO_RESPUESTA_LLAMAMIENTO = "/api/vec/bolsa/llamamientos/plazo-respuesta";
const ESQUEMA_PLAZO_RESPUESTA = "vec.bolsa.llamamiento.plazo_respuesta.v1";
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

export function seleccionarParticipacionesPorEstado(candidatos, estados, limite = 100) {
  const estadosIncluidos = new Set(estados || []);
  return (candidatos || [])
    .filter((candidato) => estadosIncluidos.has(candidato.estado_clave) && Number.isSafeInteger(candidato.orden))
    .sort((izquierda, derecha) => izquierda.orden - derecha.orden
      || String(izquierda.participacion_ref).localeCompare(String(derecha.participacion_ref), "es"))
    .slice(0, limite)
    .map((candidato) => candidato.participacion_ref);
}

// La página visible de B5 no representa el conjunto del filtro. La selección
// sólo conserva referencias y se rehace mediante el cursor de lectura existente.
export async function consultarSeleccionMasivaBolsa(bolsaRef, estados, { consultar = consultarCandidatosBolsa, signal } = {}) {
  const primeras = [];
  const estadosIncluidos = new Set(estados);
  let total = 0;
  const cursores = new Set();
  const referencias = new Set();
  let cursor = "";
  let bolsaInicial = null;
  let totalBolsa = 0;
  for (;;) {
    if (signal?.aborted) return { ok: false, status: 0, mensaje: "La consulta se ha cancelado." };
    const respuesta = await consultar(bolsaRef, { cursor, limite: 100 }, { signal });
    if (!respuesta.ok) return respuesta;
    if (signal?.aborted) return { ok: false, status: 0, mensaje: "La consulta se ha cancelado." };
    const datos = respuesta.datos;
    const bolsa = datos?.bolsa;
    if (!bolsa || bolsa.bolsa_ref !== bolsaRef || !Array.isArray(datos.candidatos)) {
      return { ok: false, status: 409, mensaje: "La consulta devolvió otra bolsa o una página incompleta. Vuelva a seleccionar." };
    }
    const version = JSON.stringify([datos.generado_en, bolsa.total, bolsa.por_estado, bolsa.politica_orden?.politica_ref, bolsa.politica_orden?.version]);
    if (bolsaInicial !== null && version !== bolsaInicial) {
      return { ok: false, status: 409, mensaje: "La bolsa o su orden cambiaron durante la consulta. Vuelva a seleccionar." };
    }
    bolsaInicial = version;
    totalBolsa = bolsa.total;
    for (const candidata of datos.candidatos) {
      if (referencias.has(candidata.participacion_ref)) {
        return { ok: false, status: 409, mensaje: "La lista cambió durante la consulta. Vuelva a seleccionar." };
      }
      referencias.add(candidata.participacion_ref);
      if (estadosIncluidos.has(candidata.estado_clave) && Number.isSafeInteger(candidata.orden)) {
        total += 1;
        primeras.push({ participacion_ref: candidata.participacion_ref, estado_clave: candidata.estado_clave, orden: candidata.orden });
        primeras.sort((a, b) => a.orden - b.orden || a.participacion_ref.localeCompare(b.participacion_ref, "es"));
        if (primeras.length > 100) primeras.pop();
      }
    }
    if (!datos.hay_mas) break;
    if (!datos.candidatos.length || !datos.cursor_siguiente || cursores.has(datos.cursor_siguiente)) {
      return { ok: false, status: 409, mensaje: "No se pudo completar la paginación. Vuelva a seleccionar." };
    }
    cursor = datos.cursor_siguiente;
    cursores.add(cursor);
  }
  if (referencias.size !== totalBolsa) {
    return { ok: false, status: 409, mensaje: "La cantidad de candidatos cambió durante la consulta. Vuelva a seleccionar." };
  }
  const participaciones = primeras.map((candidata) => candidata.participacion_ref);
  const orden = Object.fromEntries(primeras.map((candidata) => [candidata.participacion_ref, candidata.orden]));
  return { ok: true, participaciones, orden, total };
}

export async function consultarBolsas({ fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(RUTA_BOLSAS, {
      method: "GET",
      credentials: "same-origin",
      mode: "same-origin", cache: "no-store", redirect: "error",
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

export async function consultarEstadisticasBolsa({ fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(RUTA_ESTADISTICAS_BOLSA, { method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", signal, headers: { Accept: "application/json" } });
    if (!respuesta.ok) {
      const mensajes = { 401: "Se requiere una sesión interna autenticada.", 403: "La sesión no dispone de permisos para consultar estadísticas de Bolsa.", 404: "El servicio de estadísticas de Bolsa no está disponible." };
      return { ok: false, status: respuesta.status, codigo: respuesta.status === 403 ? "acceso_denegado" : "error_servidor", mensaje: mensajes[respuesta.status] || `No se pudieron consultar las estadísticas de Bolsa (HTTP ${respuesta.status}).` };
    }
    return { ok: true, datos: validarRespuestaEstadisticas(await respuesta.json()) };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red_o_contrato", mensaje: error instanceof Error ? error.message : "Error de comunicación con el servicio de estadísticas de Bolsa." };
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
      credentials: "same-origin",
      mode: "same-origin", cache: "no-store", redirect: "error",
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
    const respuesta = await fetchImpl(`${RUTA_BOLSAS}/${segmentoRuta(bolsaRef)}/candidatos/${segmentoRuta(participacionRef)}/situacion`, {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": payload.clave_idempotencia },
      body: JSON.stringify({ situacion: payload.situacion, motivo: payload.motivo, fecha_disponible: payload.fecha_disponible || null }),
    });
    const cuerpo = await respuesta.json().catch(() => ({}));
    if (respuesta.ok && cuerpo?.data?.recibo_ref) return { ok: true, datos: cuerpo.data };
    const mensaje = respuesta.status === 403
      ? "La sesión no dispone de permiso para cambiar esta situación."
      : respuesta.status === 409
        ? "El cambio entra en conflicto con la situación vigente o con un reintento anterior."
        : "No se pudo registrar el cambio de situación.";
    return { ok: false, status: respuesta.status, codigo: cuerpo?.error?.codigo || "error_servidor", mensaje };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: error instanceof Error ? error.message : "Error de comunicación." };
  }
}

export async function consultarContactosCandidato(bolsaRef, participacionRef, { fetchImpl = fetch, signal } = {}) {
  if (typeof participacionRef !== "string" || participacionRef.trim() === "") {
    return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de candidato no válida." };
  }

  const url = `${RUTA_BOLSAS}/${segmentoRuta(bolsaRef)}/candidatos/${segmentoRuta(participacionRef)}/contactos?limite=20`;

  try {
    const respuesta = await fetchImpl(url, {
      method: "GET",
      credentials: "same-origin",
      mode: "same-origin", cache: "no-store", redirect: "error",
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

export async function registrarContactoCandidato(bolsaRef, participacionRef, payload, { fetchImpl = fetch } = {}) {
  if (!bolsaRef || !participacionRef || !payload?.canal || !payload?.resultado || !payload?.anotacion || !payload?.clave_idempotencia) return { ok:false,status:400,codigo:"solicitud_invalida",mensaje:traducirBolsaInterna("contacto_solicitud_invalida") };
  try {
    const respuesta=await fetchImpl(`${RUTA_BOLSAS}/${segmentoRuta(bolsaRef)}/candidatos/${segmentoRuta(participacionRef)}/contactos`,{method:"POST",credentials: "same-origin",mode:"same-origin",cache:"no-store",redirect:"error",headers:{Accept:"application/json","Content-Type":"application/json","Idempotency-Key":payload.clave_idempotencia},body:JSON.stringify({canal:payload.canal,resultado:payload.resultado,anotacion:payload.anotacion,instante:payload.instante,llamamiento_ref:payload.llamamiento_ref||""})});
    const cuerpo=await respuesta.json().catch(()=>({})); if(respuesta.ok&&cuerpo?.data?.recibo_ref)return{ok:true,datos:cuerpo.data}; return{ok:false,status:respuesta.status,codigo:cuerpo?.error?.codigo||"error_servidor",mensaje:traducirBolsaInterna(respuesta.status===403?"contacto_permiso_denegado":respuesta.status===409?"contacto_clave_conflicto":"contacto_registro_error")};
  } catch(_error){return{ok:false,status:0,codigo:"error_red",mensaje:traducirBolsaInterna("contacto_comunicacion_error")}}
}

/**
 * Consulta de solo lectura del plazo de respuesta que propone el catálogo de
 * reglas de Bolsa. Sin catálogo responde configurada=false; cualquier fallo o
 * contrato inesperado deja el asistente B7 con el texto libre de siempre.
 */
export async function consultarPlazoRespuestaLlamamiento({ fetchImpl = globalThis.fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(RUTA_PLAZO_RESPUESTA_LLAMAMIENTO, { method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", signal, headers: { Accept: "application/json" } });
    if (!respuesta?.ok) return { ok: false, status: respuesta?.status || 0 };
    const datos = (await respuesta.json())?.data;
    if (datos?.esquema !== ESQUEMA_PLAZO_RESPUESTA || typeof datos.configurada !== "boolean") return { ok: false, status: 0 };
    return { ok: true, datos };
  } catch (_error) {
    return { ok: false, status: 0 };
  }
}

/**
 * Convierte la regla en la propuesta editable del paso 3 y su procedencia,
 * visible solo para RRHH. Devuelve null si no hay regla válida.
 */
export function propuestaPlazoRespuesta(datos) {
  if (!datos?.configurada) return null;
  const regla = datos.regla || {};
  const venceEn = new Date(datos.vencimiento?.vence_en || "");
  const cadena = (valor) => typeof valor === "string" && valor.trim().length > 0;
  if (!cadena(regla.texto) || !/^[^:\s]+:\d+:[^:\s]+$/.test(regla.referencia || "") || !["ejemplo", "reglamento"].includes(regla.origen) ||
    (regla.origen === "reglamento" && !cadena(regla.articulo)) || !Number.isFinite(venceEn.getTime())) return null;
  const fecha = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: ZONA_HORARIA_PORTAL, weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(venceEn);
  const hora = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: ZONA_HORARIA_PORTAL, hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23" }).format(venceEn);
  const procedencia = regla.origen === "ejemplo"
    ? traducirPortal("panel_b7_plazo_procedencia_ejemplo")
    : traducirPortal(regla.ejemplo ? "panel_b7_plazo_procedencia_reglamento_parcial" : "panel_b7_plazo_procedencia_reglamento", { articulo: regla.articulo });
  return { texto: traducirPortal("panel_b7_plazo_regla_texto", { fecha, hora }), procedencia, ejemplo: regla.origen === "ejemplo" || regla.ejemplo === true, descripcion: regla.texto, referencia: regla.referencia };
}

export function crearControladorBolsas({ estado, renderizar, navegar, obtenerFuenteLectura = () => null, documento = globalThis.document }) {
  const controladoresLectura = new Map();
  let controladorSeleccionMasiva = null;
  let revisionSeleccionMasiva = 0;
  // Custodia volátil del comando mientras el resultado de un POST puede ser incierto.
  // La navegación cambia filtros y bolsa; nunca cambia esta clave ni este cuerpo.
  let emisionB7 = null;
  function restaurarComandoB7(registro) {
    registro.flujo.participaciones = [...registro.comando.participaciones];
    registro.flujo.configuracion = { ...registro.comando.configuracion };
    registro.flujo.clave_idempotencia = registro.comando.clave_idempotencia;
  }
  // La propuesta del catálogo se pide al abrir el asistente, para que esté
  // lista en el paso 3; sin ella el plazo sigue siendo texto libre.
  async function cargarPlazoRespuestaB7(flujo) {
    const resultado = await consultarPlazoRespuestaLlamamiento();
    const propuesta = resultado.ok ? propuestaPlazoRespuesta(resultado.datos) : null;
    if (!propuesta || estado.filtrosBolsa?.nuevo_llamamiento !== flujo) return;
    flujo.reglaPlazo = propuesta;
    // Si RRHH ya está en el paso 3 no se vuelve a pintar (perdería lo escrito):
    // solo se rellena el plazo si sigue vacío.
    const campo = flujo.paso === 3 ? documento.querySelector?.('[data-bolsa-form="b7-paso3"] input[name="plazo"]') : null;
    if (campo && !String(campo.value || "").trim()) campo.value = propuesta.texto;
  }
  const plazoIndicado = (valor) => {
    const plazo = String(valor || "").trim();
    return plazo.length > 0 && !/\bpendiente\b/i.test(plazo);
  };
  function invalidarSeleccionMasiva() {
    revisionSeleccionMasiva += 1;
    controladorSeleccionMasiva?.abort();
    controladorSeleccionMasiva = null;
    const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
    if (flujo && flujo !== emisionB7?.flujo) {
      flujo.consultando = false;
      flujo.seleccion_total = false;
      flujo.participaciones = [];
      flujo.ordenSeleccion = {};
      flujo.totalElegibles = null;
    }
  }
  async function seleccionarTodoElFiltro(estados) {
    invalidarSeleccionMasiva();
    const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
    if (!flujo || flujo.paso !== 2 || flujo.acceso_denegado) return;
    if (!estados.length) {
      flujo.error = traducirBolsaInterna("b7_estados_obligatorios");
      renderizar();
      documento.querySelector("[data-b7-seleccion-error]")?.focus?.();
      return;
    }
    const revision = revisionSeleccionMasiva;
    const bolsaRef = estado.bolsaSeleccionada;
    const filtros = { estado: estado.filtrosBolsa.estado || "", texto: estado.filtrosBolsa.texto || "" };
    const controlador = new AbortController();
    controladorSeleccionMasiva = controlador;
    flujo.estados = estados;
    flujo.consultando = true;
    flujo.error = "";
    renderizar();
    documento.querySelector("[data-b7-seleccion-status]")?.focus?.();
    const fuente = fuenteLectura();
    let resultado;
    try {
      resultado = await consultarSeleccionMasivaBolsa(bolsaRef, estados, {
        consultar: (ref, opciones, contexto) => fuente?.consultarCandidatosBolsa
          ? fuente.consultarCandidatosBolsa(ref, opciones, contexto)
          : consultarCandidatosBolsa(ref, opciones, contexto),
        signal: controlador.signal,
      });
    } catch (error) {
      resultado = { ok: false, status: 0, mensaje: error instanceof Error ? error.message : "Error de comunicación con Bolsa." };
    }
    if (controlador.signal.aborted || revision !== revisionSeleccionMasiva || estado.filtrosBolsa?.nuevo_llamamiento !== flujo || estado.bolsaSeleccionada !== bolsaRef || estado.filtrosBolsa.estado !== filtros.estado || estado.filtrosBolsa.texto !== filtros.texto || flujo.estados.join("|") !== estados.join("|")) return;
    controladorSeleccionMasiva = null;
    flujo.consultando = false;
    if (resultado.ok) {
      flujo.participaciones = resultado.participaciones;
      flujo.ordenSeleccion = resultado.orden;
      flujo.totalElegibles = resultado.total;
      flujo.seleccion_total = true;
    } else {
      flujo.participaciones = [];
      flujo.ordenSeleccion = {};
      flujo.totalElegibles = null;
      flujo.seleccion_total = false;
      flujo.error = [401, 403].includes(resultado.status) ? traducirBolsaInterna("b7_seleccion_denegada") : resultado.status === 409 ? resultado.mensaje : traducirBolsaInterna("b7_consulta_fallida", { motivo: resultado.mensaje || traducirBolsaInterna("b7_error_lectura") });
      if ([401, 403].includes(resultado.status)) retirarCandidatosDenegados(flujo.error);
    }
    renderizar();
    documento.querySelector(resultado.ok ? '[data-bolsa-accion="b7-seleccionar-todas"]' : '[data-b7-seleccion-error]')?.focus?.();
  }
  function retirarCandidatosDenegados(mensaje) {
    invalidarSeleccionMasiva();
    // Una lectura anterior no puede restaurar datos tras perder autorización.
    for (const clave of ["candidatos", "contactos"]) {
      controladoresLectura.get(clave)?.abort();
      controladoresLectura.delete(clave);
    }
    estado.modalFicha?.controladorOperaciones?.abort();
    estado.modalFicha?.controladorSanciones?.abort();
    estado.modalFicha = null;
    estado.modalContactos = null;
    estado.modalResultado = null;
    const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
    if (flujo) flujo.acceso_denegado = true;
    estado.datosCandidatos = { carga: "denegado", datos: null, error: mensaje };
  }
  const controladorIntentosContacto = crearControladorIntentosContacto({ estado, renderizar });
  const controladorCorreoB7 = crearControladorCorreoLlamamiento({ estado, renderizar });
  const controladorOrigenContacto = crearControladorOrigenContacto({ estado, renderizar });
  const controladorOperacionesB8 = crearControladorOperacionesSituacion({
    estado,
    renderizar,
    recargar: async (participacionRef) => {
      const bolsaRef = estado.bolsaSeleccionada;
      const modal = estado.modalFicha;
      await Promise.all([cargarCandidatosBolsa(bolsaRef), cargarBolsas(), cargarEstadisticas()]);
      if (estado.modalFicha !== modal) return;
      const candidato = estado.datosCandidatos?.datos?.candidatos?.find((item) => item.participacion_ref === participacionRef);
      if (candidato) modal.candidato = candidato;
    },
  });
  // El bloque de sanciones refresca la ficha igual que B8 tras un efecto.
  const controladorSancionesB24 = crearControladorSanciones({
    estado,
    renderizar,
    recargar: async (participacionRef) => {
      const bolsaRef = estado.bolsaSeleccionada;
      const modal = estado.modalFicha;
      await Promise.all([cargarCandidatosBolsa(bolsaRef), cargarBolsas(), cargarEstadisticas()]);
      if (estado.modalFicha !== modal) return;
      const candidato = estado.datosCandidatos?.datos?.candidatos?.find((item) => item.participacion_ref === participacionRef);
      if (candidato) modal.candidato = candidato;
    },
  });
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
    if (clave === "estadisticas" && estado.datosEstadisticas?.carga === "cargando") estado.datosEstadisticas = null;
  }

  async function cargarEstadisticas() {
    const controlador = iniciarLectura("estadisticas");
    estado.datosEstadisticas = { carga: "cargando", datos: null, error: "" };
    renderizar();
    const res = await resolverLectura("estadisticas", controlador, () => consultarEstadisticasBolsa({ signal: controlador.signal }));
    if (res === null || !lecturaVigente("estadisticas", controlador)) return;
    terminarLectura("estadisticas", controlador);
    estado.datosEstadisticas = res.ok ? { carga: "listo", datos: res.datos, error: "" } : { carga: res.status === 403 ? "denegado" : "error", datos: null, error: res.mensaje };
    renderizar();
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
    // El POST de emisión no es cancelable: conservar su flujo permite recuperar
    // el recibo al volver a Bolsa aunque se abandone la vista durante la espera.
    const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
    if (!flujo?.enviando && !flujo?.clave_idempotencia) invalidarSeleccionMasiva();
    estado.modalFicha?.controladorOperaciones?.abort();
    estado.modalFicha?.controladorSanciones?.abort();
    for (const controlador of controladoresLectura.values()) controlador.abort();
    controladoresLectura.clear();
    for (const clave of ["bolsas", "candidatos", "contactos"]) limpiarEstadoCarga(clave);
  }

  function suspenderLlamamientoB7() {
    const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
    if (flujo && flujo !== emisionB7?.flujo) invalidarSeleccionMasiva();
    if (flujo) {
      const { nuevo_llamamiento: _omitido, ...resto } = estado.filtrosBolsa;
      estado.filtrosBolsa = resto;
    }
  }

  function sincronizarPaginaB7(formulario) {
    const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
    if (!flujo || !formulario || flujo.consultando || flujo.acceso_denegado) return;
    const visibles = (estado.datosCandidatos?.datos?.candidatos || []).filter((candidata) => flujo.estados.includes(candidata.estado_clave) && Number.isSafeInteger(candidata.orden));
    const pagina = Math.max(0, Math.min(Number(flujo.pagina) || 0, Math.max(0, Math.ceil(visibles.length / 6) - 1)));
    const presentes = visibles.slice(pagina * 6, pagina * 6 + 6);
    const marcadas = new Set(new FormData(formulario).getAll("participacion").map(String));
    const seleccionadas = new Set(flujo.participaciones || []);
    flujo.ordenSeleccion ||= {};
    for (const candidata of presentes) {
      flujo.ordenSeleccion[candidata.participacion_ref] = candidata.orden;
      if (marcadas.has(candidata.participacion_ref)) seleccionadas.add(candidata.participacion_ref);
      else seleccionadas.delete(candidata.participacion_ref);
    }
    flujo.participaciones = [...seleccionadas].sort((a, b) => (flujo.ordenSeleccion[a] || 0) - (flujo.ordenSeleccion[b] || 0) || a.localeCompare(b, "es"));
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
    if (estado.bolsaSeleccionada !== bolsaRef) invalidarSeleccionMasiva();
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
      if (estado.filtrosBolsa?.nuevo_llamamiento) estado.filtrosBolsa.nuevo_llamamiento.acceso_denegado = false;
    } else if ([401, 403].includes(res.status)) {
      retirarCandidatosDenegados(res.mensaje);
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
      ? fuente.consultarContactosCandidato(estado.bolsaSeleccionada, participacionRef, { signal: controlador.signal })
      : consultarContactosCandidato(estado.bolsaSeleccionada, participacionRef, { signal: controlador.signal }));
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
    estado.modalFicha?.controladorOperaciones?.abort();
    estado.modalFicha?.controladorSanciones?.abort();
    estado.modalFicha = { abierto: true, candidato, bolsa: datos.bolsa };
    void controladorSancionesB24.cargar(estado.modalFicha);
    void controladorOperacionesB8.cargar(estado.modalFicha);
    void controladorIntentosContacto.cargar(estado.modalFicha);
    void controladorOrigenContacto.cargar(estado.modalFicha);
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
    estado.modalFicha?.controladorOperaciones?.abort();
    estado.modalFicha?.controladorSanciones?.abort();
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
    controladorOperacionesB8.instalar(documento);
    controladorIntentosContacto.instalar(documento);
    controladorSancionesB24.instalar(documento);
    controladorCorreoB7.instalar(documento);
    documento.addEventListener("change", (evento) => {
      const control = evento.target;
      if (!control?.closest?.('[data-bolsa-form="b7-paso2"]')) return;
      const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
      if (!flujo) return;
      if (control.name === "estado") {
        const formulario = documento.querySelector('[data-bolsa-form="b7-paso2"]');
        const estados = formulario ? new FormData(formulario).getAll("estado").map(String) : [];
        invalidarSeleccionMasiva();
        flujo.estados = estados;
        flujo.pagina = 0;
        flujo.error = "";
        renderizar();
      } else if (control.name === "participacion" && flujo.seleccion_total) {
        sincronizarPaginaB7(documento.querySelector('[data-bolsa-form="b7-paso2"]'));
        flujo.seleccion_total = false;
        flujo.totalElegibles = null;
        const estadoSeleccion = documento.querySelector("[data-b7-seleccion-status]");
        if (estadoSeleccion) estadoSeleccion.textContent = `${flujo.participaciones.length} seleccionadas.`;
      } else if (control.name === "participacion") {
        sincronizarPaginaB7(documento.querySelector('[data-bolsa-form="b7-paso2"]'));
        const estadoSeleccion = documento.querySelector("[data-b7-seleccion-status]");
        if (estadoSeleccion) estadoSeleccion.textContent = `${flujo.participaciones.length} seleccionadas.`;
      }
    });
    documento.addEventListener("click", (evento) => {
      const botonVer = evento.target?.closest?.('[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]');
      if (botonVer) {
        evento.preventDefault();
        if (estado.filtrosBolsa?.nuevo_llamamiento?.enviando) return;
        const ref = botonVer.dataset.bolsaRef;
        if (ref) {
          invalidarSeleccionMasiva();
          estado.bolsaSeleccionada = ref;
          estado.filtrosBolsa = { estado: botonVer.dataset.estado || "", texto: "" };
          navegar("bolsa-candidatos");
          void cargarCandidatosBolsa(ref, { enfocarDestino: true });
        }
        return;
      }
      const botonAccion = evento.target?.closest?.("[data-bolsa-accion]");
      if (!botonAccion) return;
      const accion = botonAccion.dataset.bolsaAccion;
      if (estado.filtrosBolsa?.nuevo_llamamiento?.enviando && accion !== "cancelar-b7") {
        evento.preventDefault();
        return;
      }
      if (accion === "reintentar-bolsas") {
        evento.preventDefault();
        void cargarBolsas();
      } else if (accion === "reintentar-candidatos") {
        evento.preventDefault();
        void cargarCandidatosBolsa(estado.bolsaSeleccionada, { cursor: estado.filtrosBolsa?.nuevo_llamamiento?.cursoresPagina?.at(-1) || "" });
      } else if (accion === "reintentar-estadisticas") {
        evento.preventDefault();
        void cargarEstadisticas();
      } else if (accion === "limpiar-filtros") {
        evento.preventDefault();
        invalidarSeleccionMasiva();
        estado.filtrosBolsa = { estado: "", texto: "" };
        void cargarCandidatosBolsa(estado.bolsaSeleccionada);
      } else if (accion === "filtrar-estado") {
        evento.preventDefault();
        invalidarSeleccionMasiva();
        estado.filtrosBolsa = { ...estado.filtrosBolsa, estado: botonAccion.dataset.estado || "" };
        void cargarCandidatosBolsa(estado.bolsaSeleccionada);
      } else if (accion === "cambiar-pestana") {
        evento.preventDefault();
        estado.filtrosBolsa = { ...estado.filtrosBolsa, pestana: botonAccion.dataset.pestana || "candidatos", pagina_historico: 0 };
        renderizar();
      } else if (accion === "pagina-historico") {
        evento.preventDefault();
        estado.filtrosBolsa = { ...estado.filtrosBolsa, pagina_historico: Math.max(0, Number(botonAccion.dataset.pagina) || 0) };
        renderizar();
      } else if (accion === "pagina-siguiente") {
        evento.preventDefault();
        invalidarSeleccionMasiva();
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
      } else if (accion === "iniciar-b7") {
        evento.preventDefault();
        if (emisionB7) {
          const bolsaOriginal = emisionB7.comando.bolsa_ref;
          restaurarComandoB7(emisionB7);
          const recargar = estado.bolsaSeleccionada !== bolsaOriginal ||
            estado.datosCandidatos?.carga !== "listo" ||
            estado.datosCandidatos?.datos?.bolsa?.bolsa_ref !== bolsaOriginal;
          estado.filtrosBolsa = { ...estado.filtrosBolsa, estado: "", texto: "", nuevo_llamamiento: emisionB7.flujo };
          emisionB7.flujo.paso = 4;
          if (recargar) {
            void cargarCandidatosBolsa(bolsaOriginal, { enfocarDestino: true });
          } else {
            renderizar();
            documento.querySelector('[aria-current="step"]')?.focus?.();
          }
          return;
        }
        if (estado.filtrosBolsa?.nuevo_llamamiento?.enviando) return;
        invalidarSeleccionMasiva();
        estado.filtrosBolsa = { ...estado.filtrosBolsa, estado: "", texto: "", nuevo_llamamiento: { paso: 1, estados: ["disponible"], participaciones: [], configuracion: null, error: "", recibo: "", seleccion_total: false, cursoresPagina: [""] } };
        void cargarPlazoRespuestaB7(estado.filtrosBolsa.nuevo_llamamiento);
        controladorCorreoB7.prepararFlujo(estado.filtrosBolsa.nuevo_llamamiento);
        renderizar();
        documento.querySelector('[aria-current="step"]')?.focus?.();
      } else if (accion === "cancelar-b7") {
        evento.preventDefault();
        if (emisionB7?.flujo.recibo) emisionB7 = null;
        suspenderLlamamientoB7();
        renderizar();
      } else if (accion === "ver-historico-b7") {
        evento.preventDefault();
        if (emisionB7?.flujo.recibo) emisionB7 = null;
        invalidarSeleccionMasiva();
        estado.filtrosBolsa = { estado: "", texto: "", pestana: "historico", pagina_historico: 0 };
        void cargarCandidatosBolsa(estado.bolsaSeleccionada, { enfocarDestino: true });
      } else if (accion === "b7-pagina") {
        evento.preventDefault();
        const formulario = documento.querySelector('[data-bolsa-form="b7-paso2"]');
        if (!estado.filtrosBolsa.nuevo_llamamiento.seleccion_total) sincronizarPaginaB7(formulario);
        estado.filtrosBolsa.nuevo_llamamiento.pagina = Math.max(0, Number(botonAccion.dataset.pagina) || 0);
        renderizar();
      } else if (accion === "b7-fuente-siguiente" || accion === "b7-fuente-anterior") {
        evento.preventDefault();
        const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
        if (!flujo || flujo.consultando || flujo.enviando) return;
        if (!flujo.seleccion_total) sincronizarPaginaB7(documento.querySelector('[data-bolsa-form="b7-paso2"]'));
        flujo.cursoresPagina ||= [""];
        if (accion === "b7-fuente-siguiente") {
          const cursor = estado.datosCandidatos?.datos?.cursor_siguiente;
          if (!estado.datosCandidatos?.datos?.hay_mas || !cursor || flujo.cursoresPagina.includes(cursor)) return;
          flujo.cursoresPagina.push(cursor);
        } else if (flujo.cursoresPagina.length > 1) {
          flujo.cursoresPagina.pop();
        } else return;
        flujo.pagina = 0;
        void cargarCandidatosBolsa(estado.bolsaSeleccionada, { cursor: flujo.cursoresPagina.at(-1) })
          .then(() => documento.querySelector('[data-bolsa-form="b7-paso2"] input[name="participacion"]')?.focus?.());
      } else if (accion === "b7-limpiar-seleccion") {
        evento.preventDefault();
        invalidarSeleccionMasiva();
        renderizar();
      } else if (accion === "b7-revisar-configuracion") {
        evento.preventDefault();
        const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
        if (!flujo || flujo.enviando || flujo.acceso_denegado || emisionB7?.flujo === flujo) return;
        flujo.paso = 3;
        renderizar();
      } else if (accion === "b7-volver-seleccion") {
        evento.preventDefault();
        const flujo = estado.filtrosBolsa?.nuevo_llamamiento;
        if (!flujo || flujo.enviando || emisionB7?.flujo === flujo) return;
        invalidarSeleccionMasiva();
        flujo.paso = 2;
        flujo.error = "";
        flujo.cursoresPagina = [""];
        flujo.pagina = 0;
        void cargarCandidatosBolsa(estado.bolsaSeleccionada)
          .then(() => documento.querySelector('[data-bolsa-form="b7-paso2"] input[name="participacion"]')?.focus?.());
      } else if (accion === "b7-seleccionar-todas") {
        evento.preventDefault();
        const flujo = estado.filtrosBolsa.nuevo_llamamiento;
        const formulario = documento.querySelector('[data-bolsa-form="b7-paso2"]');
        const estados = formulario ? new FormData(formulario).getAll("estado").map(String) : flujo.estados;
        void seleccionarTodoElFiltro(estados);
      }
    });

    documento.addEventListener("submit", (evento) => {
      const paso1 = evento.target?.closest?.('[data-bolsa-form="b7-paso1"]');
      if (paso1) { evento.preventDefault(); estado.filtrosBolsa.nuevo_llamamiento.paso = 2; void cargarCandidatosBolsa(estado.bolsaSeleccionada); return; }
      const paso2 = evento.target?.closest?.('[data-bolsa-form="b7-paso2"]');
      if (paso2) {
        evento.preventDefault(); const datos = new FormData(paso2);
        const flujo = estado.filtrosBolsa.nuevo_llamamiento;
        if (flujo.consultando || flujo.acceso_denegado) return;
        const estados = datos.getAll("estado").map(String);
        if (flujo.seleccion_total && estados.join("|") !== flujo.estados.join("|")) {
          invalidarSeleccionMasiva();
          flujo.error = traducirBolsaInterna("b7_estados_cambiados");
          renderizar(); return;
        }
        if (!flujo.seleccion_total) sincronizarPaginaB7(paso2);
        const seleccion = [...flujo.participaciones];
        if (!seleccion.length) { estado.filtrosBolsa.nuevo_llamamiento.error = "Seleccione al menos un candidato."; renderizar(); return; }
        if (seleccion.length > 100) { flujo.error = traducirBolsaInterna("b7_limite_envio"); renderizar(); return; }
        Object.assign(flujo, { paso: 3, estados, participaciones: seleccion, error: "", seleccion_total: false, revision_obligatoria: false }); renderizar(); return;
      }
      const paso3 = evento.target?.closest?.('[data-bolsa-form="b7-paso3"]');
      if (paso3) {
        evento.preventDefault(); const datos = new FormData(paso3); const get = n => String(datos.get(n)||"").trim();
        const flujo = estado.filtrosBolsa.nuevo_llamamiento;
        if (flujo.acceso_denegado) return;
        const configuracion = { referencia:get("referencia"), descripcion:get("descripcion"), categoria:get("categoria"), centro:get("centro"), modalidad:get("modalidad"), fecha_inicio:get("fecha_inicio"), plazo:get("plazo"), plantilla_version:get("plantilla_version"), asunto:get("asunto"), cuerpo:"" };
        flujo.cuerpoBorrador = get("cuerpo");
        flujo.configuracion = configuracion;
        if (!plazoIndicado(configuracion.plazo)) {
          flujo.error = traducirPortal("panel_b7_plazo_error");
          renderizar(); return;
        }
        configuracion.cuerpo = get("cuerpo") + traducirPortal("panel_b7_cuerpo_metadatos", configuracion);
        if (configuracion.cuerpo.length > 4000) {
          flujo.error = traducirBolsaInterna("b7_cuerpo_excesivo");
          renderizar(); return;
        }
        flujo.error = "";
        flujo.error_422 = false;
        flujo.revision_obligatoria = false;
        flujo.paso = 4;
        renderizar(); return;
      }
      const paso4 = evento.target?.closest?.('[data-bolsa-form="b7-paso4"]');
      if (paso4) {
        evento.preventDefault(); const datos = new FormData(paso4); const flujo=estado.filtrosBolsa?.nuevo_llamamiento;
        if (!flujo || !datos.get("confirmacion") || flujo.enviando || flujo.recibo || flujo.acceso_denegado || flujo.revision_obligatoria) return;
        if (emisionB7 && emisionB7.flujo !== flujo) return;
        if (emisionB7) restaurarComandoB7(emisionB7);
        if (!emisionB7 && (!flujo.participaciones?.length || flujo.participaciones.length > 100 || Number(paso4.dataset.cantidad) !== flujo.participaciones.length)) {
          flujo.error = traducirBolsaInterna("b7_seleccion_cambiada");
          renderizar(); return;
        }
        if (!emisionB7 && !plazoIndicado(flujo.configuracion?.plazo)) {
          flujo.paso = 3;
          flujo.error = traducirPortal("panel_b7_plazo_error");
          renderizar(); return;
        }
        if (!emisionB7) {
          flujo.clave_idempotencia ||= globalThis.crypto?.randomUUID?.() || `llamamiento-${Date.now()}-${Math.random().toString(16).slice(2)}`;
          emisionB7 = {
            flujo,
            comando: Object.freeze({
              bolsa_ref: estado.bolsaSeleccionada,
              participaciones: Object.freeze([...flujo.participaciones]),
              configuracion: Object.freeze({ ...flujo.configuracion }),
              clave_idempotencia: flujo.clave_idempotencia,
            }),
            estado: "incierto",
          };
        }
        const registro = emisionB7;
        flujo.enviando=true; flujo.error=""; flujo.error_422=false; registro.estado="enviando"; renderizar();
        void emitirLlamamiento(registro.comando).then(res=>{
          if (emisionB7 !== registro) return;
          flujo.enviando = false;
          if (res.ok) {
            registro.estado = "confirmado";
            flujo.recibo = res.datos.recibo_ref;
            flujo.llamamiento_ref = res.datos.llamamiento_ref;
            flujo.avisos_contacto = res.datos.avisos_contacto || [];
          } else if ([400, 409, 422].includes(res.status)) {
            // Rechazo definitivo: el servidor no aplicó este comando. Una
            // revisión podrá iniciar otra intención con una clave nueva.
            emisionB7 = null;
            flujo.recibo = "";
            flujo.llamamiento_ref = "";
            flujo.clave_idempotencia = "";
            flujo.error_422 = true;
            flujo.revision_obligatoria = true;
            flujo.error = res.status === 422 ? traducirBolsaInterna("b7_emision_rechazada")
              : res.status === 400 ? traducirPortal("panel_b7_solicitud_rechazada")
              : res.mensaje;
          } else if ([401, 403].includes(res.status)) {
            registro.estado = "incierto";
            flujo.acceso_denegado = true;
            flujo.error = res.mensaje;
            if (estado.filtrosBolsa?.nuevo_llamamiento === flujo) {
              estado.datosCandidatos = { carga: "denegado", datos: null, error: res.mensaje };
            }
          } else {
            registro.estado = "incierto";
            flujo.error = traducirPortal("panel_b7_resultado_incierto");
          }
          if (estado.filtrosBolsa?.nuevo_llamamiento === flujo) {
            renderizar();
            documento.querySelector(res.ok ? "[data-b7-recibo]" : "[data-b7-emision-error]")?.focus?.();
          }
        }); return;
      }
      const formFiltros = evento.target?.closest?.('[data-bolsa-form="filtros"]');
      if (formFiltros) {
        evento.preventDefault();
        invalidarSeleccionMasiva();
        const datos = new FormData(formFiltros);
        estado.filtrosBolsa = {
          estado: datos.get("estado") || "",
          texto: datos.get("texto") || "",
        };
        void cargarCandidatosBolsa(estado.bolsaSeleccionada);
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
        const fechaDisponible = fecha && situacion === "disponible_desde" ? new Date(fecha).toISOString() : null;
        const huellaComando = JSON.stringify([situacion, motivo, fechaDisponible]);
        let clave = estado.modalFicha?.claveCambioSituacion;
        if (!clave || estado.modalFicha?.huellaCambioSituacion !== huellaComando) {
          clave = globalThis.crypto?.randomUUID?.() || `situacion-${Date.now()}-${Math.random().toString(16).slice(2)}`;
          if (estado.modalFicha) {
            estado.modalFicha.claveCambioSituacion = clave;
            estado.modalFicha.huellaCambioSituacion = huellaComando;
          }
        }
        void cambiarSituacionCandidato(estado.bolsaSeleccionada, formCambioSituacion.dataset.participacionRef, {
          situacion, motivo, fecha_disponible: fechaDisponible, clave_idempotencia: clave,
        }).then(async (res) => {
          if (res.ok) {
            const participacionRef = formCambioSituacion.dataset.participacionRef;
            estado.modalFicha = null;
            await cargarCandidatosBolsa(estado.bolsaSeleccionada);
            abrirFicha(participacionRef);
            if (estado.modalFicha) { estado.modalFicha.reciboSituacion = res.datos.recibo_ref; renderizar(); }
          }
          else if (estado.modalFicha) { estado.modalFicha.errorCambioSituacion = res.mensaje; renderizar(); }
        });
        return;
      }

      const formContacto = evento.target?.closest?.('[data-bolsa-form="contacto"]');
      if (formContacto) {
        evento.preventDefault(); const datos=new FormData(formContacto); const canal=String(datos.get("canal")||""); const resultado=String(datos.get("resultado")||""); const anotacion=String(datos.get("anotacion")||"").trim();
        if(!canal||!resultado||!anotacion){if(estado.modalFicha){estado.modalFicha.errorContacto=traducirBolsaInterna("contacto_formulario_incompleto");renderizar()}return}
		const huella=JSON.stringify([canal,resultado,anotacion,String(datos.get("llamamiento_ref")||"")]); let clave=estado.modalFicha?.claveContacto; let instante=estado.modalFicha?.instanteContacto;
		if(!clave||estado.modalFicha?.huellaContacto!==huella){clave=globalThis.crypto?.randomUUID?.()||`contacto-${Date.now()}-${Math.random().toString(16).slice(2)}`;instante=new Date().toISOString();if(estado.modalFicha){estado.modalFicha.claveContacto=clave;estado.modalFicha.huellaContacto=huella;estado.modalFicha.instanteContacto=instante}}
        void registrarContactoCandidato(estado.bolsaSeleccionada,formContacto.dataset.participacionRef,{canal,resultado,anotacion,instante,llamamiento_ref:String(datos.get("llamamiento_ref")||""),clave_idempotencia:clave}).then(async(res)=>{if(res.ok){const ref=formContacto.dataset.participacionRef;estado.modalFicha=null;await cargarCandidatosBolsa(estado.bolsaSeleccionada);abrirFicha(ref);if(estado.modalFicha){estado.modalFicha.reciboContacto=res.datos.recibo_ref;renderizar()}}else if(estado.modalFicha){estado.modalFicha.errorContacto=res.mensaje;renderizar()}}); return;
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
	        evento.preventDefault(); cerrarFicha();
	      }
	    });
  }

  return Object.freeze({
    cancelarPeticiones,
    suspenderLlamamientoB7,
    cargarBolsas,
    cargarCandidatosBolsa,
    cargarEstadisticas,
    abrirFicha,
    cerrarFicha,
	    abrirContactos, cerrarContactos,
	    abrirResultado, cerrarResultado, instalar,
  });
}
