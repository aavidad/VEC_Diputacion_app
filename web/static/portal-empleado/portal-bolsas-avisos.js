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
    const respuesta = await fetchImpl(`${RUTA_AVISOS_BOLSA}?${parametros}`, { method: "GET", credentials: "omit", signal, headers: { Accept: "application/json" } });
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
    ? `<button type="button" class="boton-enlace" data-accion="abrir-ficha-b5" data-bolsa-ref="${texto(aviso.bolsa)}" data-participacion-ref="${texto(participacion)}">Abrir ficha B5</button>`
    : "";
  return `<li class="lista-actividad__item" data-tipo-aviso="${texto(aviso.tipo)}"><div><strong>${titulo}</strong><p>${detalleAviso(aviso)}</p><small>${texto(fechaVisible(aviso.fecha))} · Referencias opacas</small></div>${enlace}</li>`;
}

export function renderizarBloqueAvisos({ estado = "cargando", datos = null, error = "" } = {}) {
  const cabecera = `<header class="seccion-cabecera"><div><p class="eyebrow">Control interno</p><h2>Avisos</h2></div></header>`;
  if (estado === "cargando") return `<section class="panel panel--contenido" aria-busy="true" aria-live="polite">${cabecera}<p>Cargando avisos…</p></section>`;
  if (estado === "error") return `<section class="panel panel--contenido" aria-live="assertive">${cabecera}<div class="aviso aviso--error"><p>${texto(error || "No se pudieron cargar los avisos.")}</p><button type="button" data-accion="reintentar-avisos">Reintentar</button></div></section>`;
  if (!datos || datos.items.length === 0) return `<section class="panel panel--contenido" aria-live="polite">${cabecera}<div class="estado-vacio"><h3>Sin avisos</h3><p>No hay saltos de orden ni periodos de tres años detectados en el corte consultado.</p></div><p class="texto-ayuda">${texto(datos?.provisionalidad || "")}</p></section>`;
  const conteos = `<div class="resumen-indicadores" aria-label="Avisos por tipo"><span><strong>${datos.conteos.salto_orden}</strong> saltos de orden</span><span><strong>${datos.conteos.tres_anos}</strong> tres años</span></div>`;
  const paginacion = `<footer class="paginacion"><span>Mostrando ${datos.paginacion.desde} a ${datos.paginacion.hasta} de ${datos.paginacion.total}</span><button type="button" data-accion="siguiente-avisos"${datos.paginacion.cursor_siguiente ? "" : " disabled"}>Siguiente</button></footer>`;
  return `<section class="panel panel--contenido" aria-live="polite">${cabecera}${conteos}<div class="tabla-contenedor" tabindex="0"><ul class="lista-actividad">${datos.items.map(filaAviso).join("")}</ul></div>${paginacion}<p class="texto-ayuda"><strong>Provisional:</strong> ${texto(datos.provisionalidad)}</p></section>`;
}

// El montaje P-WEB-10 puede delegar aquí sin conocer el contrato: la ficha B5
// sigue siendo la única dueña de la apertura y el foco del detalle inline.
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

