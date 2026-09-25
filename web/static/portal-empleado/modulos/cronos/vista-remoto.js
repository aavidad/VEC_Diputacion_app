import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260925-cronos-p2-v1";
import { validarDisponibilidadRemota, validarReciboMarcajeRemoto } from "./cliente-remoto-http.js";

const MOVIMIENTOS = Object.freeze(["entrada", "inicio_pausa", "fin_pausa", "salida"]);
const MENSAJE_MOTIVO = Object.freeze({
  teletrabajo_no_autorizado: "remoto_motivo_sin_autorizacion",
});

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function instanteVisible(instante, locale, zonaHoraria) {
  return new Intl.DateTimeFormat(locale, { day: "numeric", month: "short", year: "numeric",
    hour: "2-digit", minute: "2-digit", second: "2-digit", timeZone: zonaHoraria,
    timeZoneName: "short" }).format(new Date(instante));
}
function traducir(t, clave, variables) { return escapar(t(clave, variables)); }

/** Modelo de presentación. No concede permisos: sólo representa el GET validado. */
export function renderizarVistaRemotoCronos({ disponibilidad = null, estado = "consultando", recibo = null,
  pendiente = null, avisoSecuencia = false, t = crearTraductorCronos(), locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  let disponible = null; let continuidadAusente = false; let secuenciaAusente = false;
  if (disponibilidad) {
    try { disponible = validarDisponibilidadRemota(disponibilidad); }
    catch (error) {
      if (error?.codigo === "continuidad_no_confirmada") continuidadAusente = true;
      else if (error?.codigo === "secuencia_no_permitida") secuenciaAusente = true;
      else throw error;
    }
  }
  if (typeof t !== "function") throw new TypeError("traductor de Cronos no disponible");
  if (!["consultando", "listo", "enviando", "recuperando", "incierto", "servicio_incierto", "recuperacion_no_disponible", "error", "no_disponible", "error_registro", "conflicto", "registrado", "autenticacion", "acceso_denegado", "servicio", "continuidad", "secuencia"].includes(estado)) {
    throw new TypeError("estado remoto no válido");
  }
  const autorizado = disponible?.autorizado === true && disponible?.continuidad_confirmada === true;
  const bloqueado = !autorizado || estado !== "listo";
  const motivo = continuidadAusente || (disponible?.autorizado && !disponible?.continuidad_confirmada)
    ? "remoto_continuidad_pendiente"
    : secuenciaAusente ? "remoto_secuencia_no_permitida"
    : !disponible ? (estado === "consultando" ? "remoto_consultando" : "remoto_error_consulta")
      : autorizado ? (disponible.movimientos_permitidos.length ? null : "remoto_sin_movimientos")
        : MENSAJE_MOTIVO[disponible.motivo];
  const periodo = disponible?.periodo
    ? t("remoto_periodo", { desde: instanteVisible(disponible.periodo.desde, locale, zonaHoraria),
      hasta: instanteVisible(disponible.periodo.hasta, locale, zonaHoraria) }) : t("remoto_sin_periodo");
  const etiquetaEstado = estado === "consultando" ? "remoto_consultando"
    : estado === "enviando" ? "remoto_enviando"
      : estado === "recuperando" ? "remoto_recuperando"
      : estado === "incierto" ? "remoto_incierto"
      : estado === "servicio_incierto" ? "remoto_servicio_no_disponible"
        : estado === "recuperacion_no_disponible" ? "remoto_recuperacion_no_disponible"
      : estado === "error" ? "remoto_error_consulta"
      : estado === "no_disponible" ? "remoto_no_disponible"
        : estado === "autenticacion" ? "remoto_autenticacion_requerida"
          : estado === "acceso_denegado" ? "remoto_acceso_denegado"
            : estado === "servicio" ? "remoto_servicio_no_disponible"
              : estado === "continuidad" ? "remoto_continuidad_pendiente"
                : estado === "secuencia" ? "remoto_secuencia_no_permitida"
        : estado === "error_registro" ? "remoto_error_registro"
          : estado === "conflicto" ? "remoto_conflicto"
          : estado === "registrado" ? (recibo?.replay ? "remoto_replay" : "remoto_registrado")
            : autorizado ? (disponible.movimientos_permitidos.length ? "remoto_autorizado" : "remoto_sin_movimientos") : motivo;
  const tituloBloqueo = bloqueado ? traducir(t, estado === "listo" ? (motivo ?? etiquetaEstado) : etiquetaEstado) : "";
  const acciones = MOVIMIENTOS.map((movimiento, indice) => {
    const etiqueta = traducir(t, `accion_${movimiento}`);
    const permitido = autorizado && disponible.movimientos_permitidos.includes(movimiento);
    const causa = bloqueado ? tituloBloqueo : traducir(t, disponible.movimientos_permitidos.length ? "remoto_secuencia_no_permitida" : "remoto_sin_movimientos");
    return `<button type="button" data-cronos-remoto-movimiento="${movimiento}" class="${indice === 0 ? "boton-primario" : "boton-secundario"}" ${!permitido || estado !== "listo" ? `disabled aria-disabled="true" title="${causa}"` : ""}>${etiqueta}</button>`;
  }).join("");
  const reintento = ["incierto", "servicio_incierto", "recuperacion_no_disponible", "autenticacion", "acceso_denegado"].includes(estado) && pendiente
    ? `<button type="button" data-cronos-remoto-reintentar class="boton-primario">${traducir(t, "remoto_reintentar")}</button>` : "";
  const resultado = estado === "registrado" && recibo
    ? `<div class="cronos-estado cronos-estado-exito" role="status">${traducir(t, recibo.replay ? "remoto_replay" : "remoto_registrado")}</div>
      <dl class="cronos-resumen-datos"><div><dt>${traducir(t, "remoto_hora_servidor")}</dt><dd><time datetime="${escapar(recibo.instante_utc)}">${escapar(instanteVisible(recibo.instante_utc, locale, zonaHoraria))}</time></dd></div>
      <div><dt>${traducir(t, "remoto_recibo")}</dt><dd><code>${escapar(recibo.referencia)}</code></dd></div></dl>` : "";
  return `<section class="panel cronos-panel" data-cronos-remoto aria-labelledby="cronos-remoto-titulo">
    <div class="cabecera-panel cronos-cabecera-compacta"><div><h3 id="cronos-remoto-titulo">${traducir(t, "remoto_titulo")}</h3>
      <p>${traducir(t, "remoto_origen")}</p></div>
      <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${traducir(t, "abrir_ayuda", { asunto: t("remoto_titulo") })}" title="${traducir(t, "abrir_ayuda", { asunto: t("remoto_titulo") })}">?</button></div>
    <div class="cuerpo-panel"><p>${escapar(periodo)}</p>
      <p class="${estado === "no_disponible" ? "cronos-vacio" : `cronos-estado cronos-estado-${autorizado && estado === "listo" && disponible.movimientos_permitidos.length ? "exito" : "aviso"}`}" role="status" aria-live="polite">${traducir(t, etiquetaEstado)}${estado === "servicio_incierto" ? ` ${traducir(t, "remoto_incierto")}` : ""}</p>
      ${avisoSecuencia ? `<p role="status">${traducir(t, "remoto_secuencia_no_permitida")}</p>` : ""}
      <div class="cronos-acciones">${estado === "no_disponible" ? "" : acciones}${reintento}
        <button type="button" data-cronos-remoto-actualizar class="boton-secundario" ${pendiente || ["consultando", "enviando", "recuperando", "incierto", "servicio_incierto", "recuperacion_no_disponible"].includes(estado) ? 'disabled aria-disabled="true"' : ""}>${traducir(t, "remoto_actualizar")}</button></div>
      ${resultado}</div>
  </section>`;
}

function nuevaClave(cryptoImpl) {
  const clave = cryptoImpl?.randomUUID?.();
  if (typeof clave !== "string" || !/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu.test(clave)) {
    throw new Error("generador de claves no disponible");
  }
  return clave;
}

/** Montaje explícito del bloque remoto; el shell decide cuándo mostrarlo. */
export function montarVistaRemotoCronos({ raiz, cliente, mensajes = MENSAJES_CRONOS_ES,
  cryptoImpl = globalThis.crypto, locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.ownerDocument?.createElement || typeof raiz.append !== "function"
    || typeof cliente?.disponibilidad !== "function" || typeof cliente?.registrar !== "function") {
    throw new TypeError("montaje remoto de Cronos incompleto");
  }
  const t = crearTraductorCronos(mensajes);
  const nodo = raiz.ownerDocument.createElement("div");
  let activo = true; let generacion = 0; let controlador = null;
  let disponibilidad = null; let estado = "consultando"; let pendiente = null; let recibo = null; let avisoSecuencia = false;
  function pintar() {
    if (!activo) return;
    nodo.innerHTML = renderizarVistaRemotoCronos({ disponibilidad, estado, recibo, pendiente, avisoSecuencia, t, locale, zonaHoraria });
  }
  function cancelar() { controlador?.abort(); controlador = null; generacion += 1; }
  async function actualizar() {
    if (!activo || pendiente || ["enviando", "recuperando", "incierto", "servicio_incierto", "recuperacion_no_disponible"].includes(estado)) return;
    cancelar(); const version = generacion; controlador = new AbortController();
    disponibilidad = null; recibo = null; estado = "consultando"; pintar();
    try {
      const respuesta = await cliente.disponibilidad({ signal: controlador.signal });
      if (!activo || version !== generacion) return;
      disponibilidad = validarDisponibilidadRemota(respuesta);
      estado = "listo";
    } catch (error) {
      if (!activo || version !== generacion) return;
      if (error?.codigo === "teletrabajo_no_autorizado") {
        disponibilidad = { autorizado: false, continuidad_confirmada: false, movimientos_permitidos: [], motivo: "teletrabajo_no_autorizado" };
        estado = "listo";
      } else {
        estado = error?.codigo === "autenticacion_requerida" ? "autenticacion"
          : error?.codigo === "acceso_denegado" ? "acceso_denegado"
            : error?.codigo === "continuidad_no_confirmada" ? "continuidad"
              : error?.codigo === "secuencia_no_permitida" ? "secuencia"
              : error?.codigo === "servicio_no_disponible" ? "servicio"
                : error?.estado === 404 ? "no_disponible" : "error";
      }
    } finally { if (activo && version === generacion) { controlador = null; pintar(); } }
  }
  async function enviar(movimiento, reintentar = false) {
    if (!activo || ["enviando", "recuperando"].includes(estado) || disponibilidad?.autorizado !== true || disponibilidad?.continuidad_confirmada !== true) return;
    if (reintentar ? !["incierto", "servicio_incierto", "recuperacion_no_disponible", "autenticacion", "acceso_denegado"].includes(estado) || !pendiente : estado !== "listo" || !MOVIMIENTOS.includes(movimiento)) return;
    if (!reintentar) {
      if (!disponibilidad.movimientos_permitidos.includes(movimiento)) return;
      pendiente = { movimiento, clave_operacion: nuevaClave(cryptoImpl) };
      avisoSecuencia = false;
    }
    cancelar(); const version = generacion; controlador = new AbortController();
    if (reintentar) {
      estado = "recuperando"; pintar();
      if (typeof cliente.recuperar !== "function") {
        estado = "recuperacion_no_disponible"; controlador = null; pintar(); return;
      }
      try {
        const recuperado = await cliente.recuperar(pendiente, { signal: controlador.signal });
        if (!activo || version !== generacion) return;
        recibo = validarReciboMarcajeRemoto({ recibo: recuperado });
        pendiente = null; estado = "registrado"; controlador = null; pintar(); return;
      } catch (error) {
        if (!activo || version !== generacion) return;
        if (error?.codigo !== "ausencia_confirmada" || error?.estado !== 404) {
          if (error?.estado === 409) { pendiente = null; estado = "conflicto"; }
          else estado = error?.codigo === "autenticacion_requerida" ? "autenticacion"
            : error?.codigo === "acceso_denegado" ? "acceso_denegado"
              : error?.estado === 503 ? "servicio_incierto" : "recuperacion_no_disponible";
          controlador = null; pintar(); return;
        }
      }
      try {
        const actual = validarDisponibilidadRemota(await cliente.disponibilidad({ signal: controlador.signal }));
        if (!activo || version !== generacion) return;
        disponibilidad = actual;
        if (!actual.autorizado || !actual.continuidad_confirmada
          || !actual.movimientos_permitidos.includes(pendiente.movimiento)) {
          pendiente = null; estado = "listo"; controlador = null; pintar(); return;
        }
      } catch (error) {
        if (!activo || version !== generacion) return;
        estado = error?.codigo === "autenticacion_requerida" ? "autenticacion"
          : error?.codigo === "acceso_denegado" ? "acceso_denegado" : "servicio_incierto";
        controlador = null; pintar(); return;
      }
    }
    estado = "enviando"; pintar();
    let refrescar = false;
    try {
      const respuesta = await cliente.registrar(pendiente, { signal: controlador.signal });
      if (!activo || version !== generacion) return;
      // También valida adaptadores inyectados: la vista no confirma por una mera promesa resuelta.
      recibo = validarReciboMarcajeRemoto({ recibo: respuesta });
      pendiente = null; estado = "registrado";
    } catch (error) {
      if (!activo || version !== generacion) return;
      if (error?.codigo === "teletrabajo_no_autorizado") {
        disponibilidad = { autorizado: false, continuidad_confirmada: false, movimientos_permitidos: [], motivo: "teletrabajo_no_autorizado" };
        pendiente = null; estado = "listo";
      } else if (error?.codigo === "autenticacion_requerida") {
        disponibilidad = null; pendiente = null; estado = "autenticacion";
      } else if (error?.codigo === "acceso_denegado" || error?.estado === 403) {
        disponibilidad = null; pendiente = null; estado = "acceso_denegado";
      } else if (error?.codigo === "continuidad_no_confirmada") {
        disponibilidad = { ...disponibilidad, continuidad_confirmada: false, movimientos_permitidos: [], motivo: "continuidad_no_confirmada" };
        pendiente = null; estado = "continuidad";
      } else if (error?.codigo === "secuencia_no_permitida") {
        pendiente = null; estado = "secuencia"; avisoSecuencia = true; refrescar = true;
      } else if (error?.estado === 409 || error?.codigo === "conflicto") {
        pendiente = null; estado = "conflicto";
      } else if (error?.incierto !== false) {
        estado = error?.estado === 503 || error?.codigo === "servicio_no_disponible" ? "servicio_incierto" : "incierto";
      } else {
        pendiente = null; estado = "error_registro";
      }
    } finally { if (activo && version === generacion) { controlador = null; pintar(); if (refrescar) void actualizar(); } }
  }
  function alPulsar(evento) {
    const boton = evento.target?.closest?.("[data-cronos-remoto-movimiento], [data-cronos-remoto-reintentar], [data-cronos-remoto-actualizar]");
    if (!boton || boton.disabled) return;
    if (boton.hasAttribute("data-cronos-remoto-actualizar")) void actualizar();
    else if (boton.hasAttribute("data-cronos-remoto-reintentar")) void enviar(null, true);
    else void enviar(boton.getAttribute("data-cronos-remoto-movimiento"));
  }
  nodo.addEventListener("click", alPulsar);
  raiz.append(nodo); pintar(); void actualizar();
  return Object.freeze({ actualizar, enviar, desmontar() { if (!activo) return; activo = false; cancelar(); nodo.removeEventListener("click", alPulsar); nodo.remove(); },
    estado: () => Object.freeze({ estado, disponibilidad, pendiente: pendiente && { ...pendiente }, recibo }) });
}
