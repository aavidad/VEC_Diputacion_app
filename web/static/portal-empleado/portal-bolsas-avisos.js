import { icono } from "../comun/iconos-vec.js?v=20260925-aspecto-v1";

export const RUTA_AVISOS_BOLSA = "/api/vec/bolsa/avisos";
export const ESQUEMA_AVISOS_BOLSA = "vec.bolsa.rrhh.avisos.v1";

const TIPOS = new Set(["salto_orden", "tres_anos"]);

function texto(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function referenciaOpaca(valor) {
  return typeof valor === "string" && /^[A-Za-z0-9][A-Za-z0-9:._/-]{2,255}$/.test(valor);
}

function numeroNatural(valor) {
  return Number.isSafeInteger(valor) && valor >= 0;
}

export function validarAvisosBolsa(sobre) {
  const datos = sobre?.data;
  if (!datos || datos.esquema !== ESQUEMA_AVISOS_BOLSA || !Array.isArray(datos.items) ||
      !datos.conteos || !datos.paginacion || typeof datos.provisionalidad !== "string" ||
      !numeroNatural(datos.conteos.salto_orden) || !numeroNatural(datos.conteos.tres_anos) ||
      !numeroNatural(datos.paginacion.desde) || !numeroNatural(datos.paginacion.hasta) || !numeroNatural(datos.paginacion.total)) {
    throw new TypeError("Contrato de avisos de Bolsa no válido.");
  }
  for (const aviso of datos.items) {
    if (!TIPOS.has(aviso?.tipo) || !referenciaOpaca(aviso.bolsa) || !referenciaOpaca(aviso.referencia) ||
        !aviso.detalle || typeof aviso.detalle !== "object" || Number.isNaN(Date.parse(aviso.fecha))) {
      throw new TypeError("Aviso de Bolsa no válido.");
    }
    const participacion = aviso.detalle.participacion_ref;
    if (participacion !== undefined && !referenciaOpaca(participacion)) throw new TypeError("Participación de aviso no válida.");
  }
  return datos;
}

export async function consultarAvisosBolsa({ cursor = "", limite = 6, fetchImpl = fetch, signal } = {}) {
  const parametros = new URLSearchParams({ limite: String(limite) });
  if (cursor) parametros.set("cursor", cursor);
  try {
    const respuesta = await fetchImpl(`${RUTA_AVISOS_BOLSA}?${parametros}`, { method: "GET", credentials: "same-origin", signal, headers: { Accept: "application/json" } });
    if (!respuesta.ok) {
      const mensajes = { 401: "Se requiere una sesión interna autenticada.", 403: "La sesión no dispone de ámbito para consultar avisos.", 404: "El servicio de avisos no está disponible." };
      return { ok: false, status: respuesta.status, mensaje: mensajes[respuesta.status] || `No se pudieron consultar los avisos (HTTP ${respuesta.status}).` };
    }
    return { ok: true, datos: validarAvisosBolsa(await respuesta.json()) };
  } catch (error) {
    return { ok: false, status: 0, mensaje: error instanceof Error ? error.message : "Error de comunicación con los avisos." };
  }
}

function fechaVisible(valor) {
  const fecha = new Date(valor);
  return Number.isNaN(fecha.valueOf()) ? "Fecha no disponible" : new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short" }).format(fecha);
}

function detalleAviso(aviso) {
  if (aviso.tipo === "salto_orden") {
    return `Orden ${texto(aviso.detalle.orden)}; primera persona llamada: ${texto(aviso.detalle.orden_primero_llamado)}.`;
  }
  return `Alcanza tres años: ${texto(fechaVisible(aviso.detalle.alcanza_tres_anos_en))}.`;
}

function filaAviso(aviso) {
  const titulo = aviso.tipo === "salto_orden" ? "Posible salto de orden" : "Tres años de trabajo continuado";
  const participacion = aviso.detalle.participacion_ref;
  const enlace = referenciaOpaca(participacion)
    ? `<button type="button" class="boton-enlace" data-accion="abrir-ficha-b5" data-bolsa-ref="${texto(aviso.bolsa)}" data-participacion-ref="${texto(participacion)}">Abrir ficha</button>`
    : "";
  return `<li class="lista-actividad__item" data-tipo-aviso="${texto(aviso.tipo)}"><div><strong>${titulo}</strong><p>${detalleAviso(aviso)}</p><small>${texto(fechaVisible(aviso.fecha))}</small></div>${enlace}</li>`;
}

export function renderizarBloqueAvisos({ estado = "cargando", datos = null, error = "" } = {}) {
  const conteos = datos?.conteos && datos.items.length > 0
    ? `<span class="estado-chip advertencia">${datos.conteos.salto_orden} saltos de orden</span><span class="estado-chip info">${datos.conteos.tres_anos} tres años</span>`
    : "";
  // El cómputo legal aún pendiente de RRHH se señala con una pastilla; la explicación
  // completa vive en la ayuda («?»), no en la pantalla.
  const pendiente = datos?.provisionalidad ? '<span class="estado-chip advertencia" data-avisos-provisional>Pendiente de RRHH</span>' : "";
  const cabecera = `<div class="cabecera-panel"><h2>Avisos</h2>${conteos || pendiente ? `<div class="avisos-bolsa-conteos" aria-label="Avisos por tipo">${conteos}${pendiente}</div>` : ""}</div>`;
  if (estado === "cargando") return `<section class="panel avisos-bolsa" aria-busy="true" aria-live="polite">${cabecera}<div class="cuerpo-panel avisos-bolsa-vacio" role="status">Cargando avisos…</div></section>`;
  if (estado === "error") return `<section class="panel avisos-bolsa" aria-live="assertive">${cabecera}<div class="cuerpo-panel aviso aviso--error"><span>${texto(error || "No se pudieron cargar los avisos.")}</span><button type="button" data-accion="reintentar-avisos">Reintentar</button></div></section>`;
  if (!datos || datos.items.length === 0) return `<section class="panel avisos-bolsa" aria-live="polite">${cabecera}<div class="cuerpo-panel avisos-bolsa-vacio"><span class="avisos-bolsa-icono" aria-hidden="true">${icono("correcto")}</span><span><strong>Sin avisos.</strong> No hay saltos de orden ni periodos de tres años.</span></div></section>`;
  const paginacion = `<footer class="paginacion"><span>Mostrando ${datos.paginacion.desde} a ${datos.paginacion.hasta} de ${datos.paginacion.total}</span><button type="button" data-accion="siguiente-avisos"${datos.paginacion.cursor_siguiente ? "" : " disabled"}>Siguiente</button></footer>`;
  return `<section class="panel avisos-bolsa" aria-live="polite">${cabecera}<div class="tabla-contenedor avisos-bolsa-lista" tabindex="0"><ul class="lista-actividad">${datos.items.map(filaAviso).join("")}</ul></div>${paginacion}</section>`;
}

// El montaje P-WEB-10 puede delegar aquí sin conocer el contrato: la ficha de
// participación sigue siendo la única dueña de la apertura y el foco del detalle inline.
export function manejarAccionAvisos(evento, { abrirFichaB5, siguiente, reintentar } = {}) {
  const control = evento?.target?.closest?.("[data-accion]");
  if (!control) return false;
  const accion = control.dataset.accion;
  if (accion === "abrir-ficha-b5" && typeof abrirFichaB5 === "function") abrirFichaB5(control.dataset.bolsaRef, control.dataset.participacionRef, control);
  else if (accion === "siguiente-avisos" && typeof siguiente === "function" && !control.disabled) siguiente();
  else if (accion === "reintentar-avisos" && typeof reintentar === "function") reintentar();
  else return false;
  return true;
}
