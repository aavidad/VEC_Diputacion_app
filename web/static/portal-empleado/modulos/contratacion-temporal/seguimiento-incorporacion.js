import { validarPreparacionIncorporacionEjercicio } from "./contrato-incorporacion-ejercicio.js";
import { validarConsultaSeguimientoIncorporacion, validarSeguimientoIncorporacion } from "./contrato-seguimiento-incorporacion.js";
import { escaparHTML as escapar } from "./componentes-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

export const CLAVES_I18N_SEGUIMIENTO_INCORPORACION = Object.freeze([
  "seguimiento_incorporacion_titulo",
  "seguimiento_incorporacion_consultar", "seguimiento_incorporacion_cargando",
  "seguimiento_incorporacion_error", "seguimiento_incorporacion_recibo",
  "seguimiento_incorporacion_estado", "seguimiento_incorporacion_periodo",
  "seguimiento_incorporacion_registrado", "seguimiento_incorporacion_hitos",
  "seguimiento_incorporacion_sin_hitos", "seguimiento_incorporacion_documentos",
  "seguimiento_incorporacion_sin_documentos", "seguimiento_incorporacion_sin_recibo",
  "seguimiento_incorporacion_referencia", "seguimiento_incorporacion_transicion",
  "seguimiento_incorporacion_efectiva", "seguimiento_incorporacion_expediente",
  "seguimiento_incorporacion_version_expediente", "seguimiento_incorporacion_seguimiento",
  "seguimiento_incorporacion_version_seguimiento",
  "seguimiento_incorporacion_referencia_tecnica", "seguimiento_incorporacion_clave_tecnica",
  "seguimiento_incorporacion_estado_pendiente", "seguimiento_incorporacion_estado_pendiente_incorporacion",
  "seguimiento_incorporacion_estado_incorporada", "seguimiento_incorporacion_estado_vigente",
  "seguimiento_incorporacion_estado_cerrado_administrativamente",
  "seguimiento_incorporacion_transicion_confirmar_incorporacion",
  "seguimiento_incorporacion_transicion_cerrar_administrativamente_sin_cese",
  "seguimiento_incorporacion_documento_justificante",
  "seguimiento_incorporacion_documento_resolucion_ejercicio",
  "seguimiento_incorporacion_documento_anexo_ejercicio",
]);

function fila(etiqueta, valor) {
  return `<div><dt>${escapar(etiqueta)}</dt><dd><code>${escapar(String(valor))}</code></dd></div>`;
}

const CLAVES_ETIQUETADAS = Object.freeze({
  estado_pendiente: true, estado_pendiente_incorporacion: true, estado_incorporada: true,
  estado_vigente: true, estado_cerrado_administrativamente: true,
  transicion_confirmar_incorporacion: true,
  transicion_cerrar_administrativamente_sin_cese: true,
  documento_justificante: true, documento_resolucion_ejercicio: true, documento_anexo_ejercicio: true,
});

function etiquetaClave(valor, categoria, t) {
  const clave = `${categoria}_${valor}`;
  return Object.hasOwn(CLAVES_ETIQUETADAS, clave)
    ? t(`seguimiento_incorporacion_${clave}`)
    : t("seguimiento_incorporacion_clave_tecnica", { clave: valor });
}

function filaReferencia(etiqueta, valor, t) {
  return `<div><dt>${escapar(etiqueta)}</dt><dd><small>${escapar(t("seguimiento_incorporacion_referencia_tecnica"))}: </small><code>${escapar(String(valor))}</code></dd></div>`;
}

function filaClave(etiqueta, valor, categoria, t, ocultarClaveConocida = false) {
  const tecnica = !ocultarClaveConocida || !Object.hasOwn(CLAVES_ETIQUETADAS, `${categoria}_${valor}`);
  return `<div><dt>${escapar(etiqueta)}</dt><dd>${escapar(etiquetaClave(valor, categoria, t))}${tecnica ? ` <small><code>${escapar(String(valor))}</code></small>` : ""}</dd></div>`;
}

function filaEstados(etiqueta, origen, destino, t, ocultarClaveConocida = false) {
  const tecnica = !ocultarClaveConocida || !Object.hasOwn(CLAVES_ETIQUETADAS, `estado_${origen}`)
    || !Object.hasOwn(CLAVES_ETIQUETADAS, `estado_${destino}`);
  return `<div><dt>${escapar(etiqueta)}</dt><dd>${escapar(etiquetaClave(origen, "estado", t))} → ${escapar(etiquetaClave(destino, "estado", t))}${tecnica ? ` <small><code>${escapar(origen)} → ${escapar(destino)}</code></small>` : ""}</dd></div>`;
}

function filaFecha(etiqueta, valor, formatear) {
  return `<div><dt>${escapar(etiqueta)}</dt><dd><time datetime="${escapar(valor)}">${escapar(formatear(valor))}</time></dd></div>`;
}

function filaPeriodo(etiqueta, periodo, formatear) {
  return `<div><dt>${escapar(etiqueta)}</dt><dd><time datetime="${escapar(periodo.desde)}">${escapar(formatear(periodo.desde))}</time> — <time datetime="${escapar(periodo.hasta)}">${escapar(formatear(periodo.hasta))}</time></dd></div>`;
}

function crearFormateadoresFechas(locale, zonaHoraria) {
  const fechaCivil = new Intl.DateTimeFormat(locale, {
    day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC",
  });
  const instante = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria,
  });
  return Object.freeze({
    fechaCivil: (valor) => fechaCivil.format(new Date(valor)),
    instante: (valor) => instante.format(new Date(valor)),
  });
}

function renderizarSeguimiento(datos, t, fechas, ocultarClaveConocida = false) {
  const hitos = datos.actuaciones.length === 0
    ? `<p>${escapar(t("seguimiento_incorporacion_sin_hitos"))}</p>`
    : `<ol>${datos.actuaciones.map((actuacion) => {
      const documentos = actuacion.documentos.length === 0
        ? escapar(t("seguimiento_incorporacion_sin_documentos"))
        : actuacion.documentos.map(({ tipo_clave: tipo, referencia }) =>
          `<li>${escapar(etiquetaClave(tipo, "documento", t))} <small><code>${escapar(tipo)}</code> · ${escapar(t("seguimiento_incorporacion_referencia_tecnica"))}: <code>${escapar(referencia)}</code></small></li>`).join("");
      return `<li><dl>${filaClave(t("seguimiento_incorporacion_transicion"), actuacion.transicion_clave, "transicion", t, ocultarClaveConocida)}
        ${filaEstados(t("seguimiento_incorporacion_estado"), actuacion.estado_origen, actuacion.estado_destino, t, ocultarClaveConocida)}
        ${filaFecha(t("seguimiento_incorporacion_efectiva"), actuacion.efectivo_en, fechas.instante)}${filaFecha(t("seguimiento_incorporacion_registrado"), actuacion.registrada_en, fechas.instante)}
        ${filaReferencia(t("seguimiento_incorporacion_referencia"), actuacion.actuacion_ref, t)}
        </dl><h5>${escapar(t("seguimiento_incorporacion_documentos"))}</h5><ul>${documentos}</ul></li>`;
    }).join("")}</ol>`;
  return `<dl class="ct-resumen">${filaClave(t("seguimiento_incorporacion_estado"), datos.estado_clave, "estado", t, ocultarClaveConocida)}
    ${filaPeriodo(t("seguimiento_incorporacion_periodo"), datos.periodo, fechas.fechaCivil)}
    ${filaFecha(t("seguimiento_incorporacion_registrado"), datos.registrado_en, fechas.instante)}</dl>
    <details class="ct-seguimiento-trazabilidad"><summary>${escapar(t("consulta_seguimiento_trazabilidad"))}</summary>
      <dl class="ct-resumen">${filaReferencia(t("seguimiento_incorporacion_expediente"), datos.expediente_ref, t)}
      ${fila(t("seguimiento_incorporacion_version_expediente"), datos.version_expediente)}
      ${filaReferencia(t("seguimiento_incorporacion_seguimiento"), datos.seguimiento_ref, t)}
      ${fila(t("seguimiento_incorporacion_version_seguimiento"), datos.version_seguimiento)}</dl></details>
    <h4>${escapar(t("seguimiento_incorporacion_hitos"))}</h4>${hitos}`;
}

// La consulta interna de solo lectura no dispone de un recibo de preparación.
// El contrato del GET verifica la proyección y su cruce con el expediente pedido.
export function renderizarConsultaSeguimientoIncorporacion(datos, expedienteRef, {
  mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid",
} = {}) {
  const vista = validarConsultaSeguimientoIncorporacion(datos, expedienteRef);
  return renderizarSeguimiento(
    vista,
    crearTraductorContratacionTemporal(mensajes),
    crearFormateadoresFechas(locale, zonaHoraria),
    true,
  );
}

export function montarSeguimientoIncorporacion({ raiz, cliente, recibo, mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || typeof cliente?.consultar !== "function") {
    throw new TypeError("dependencias de seguimiento de incorporación no válidas");
  }
  let reciboConfirmado = null;
  try {
    const preparado = validarPreparacionIncorporacionEjercicio({
      esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2",
      expediente_ref: recibo?.expediente_ref,
      version_actual_expediente: recibo?.version_actual_expediente,
      preparacion: null, recibo,
    }, recibo?.expediente_ref);
    reciboConfirmado = preparado.recibo;
  } catch { /* Sin recibo V2 confirmado, la consulta queda deshabilitada. */ }
  const fechas = crearFormateadoresFechas(locale, zonaHoraria);
  const t = crearTraductorContratacionTemporal(mensajes);
  let activo = true, controlador = null, estado = "inicial", datos = null;

  function pintar() {
    if (!activo) return;
    const botonAnterior = raiz.querySelector?.("[data-ct-seguimiento-consultar]");
    const estadoAnterior = raiz.querySelector?.("[data-ct-seguimiento-estado-consulta]");
    const focoActual = raiz.ownerDocument?.activeElement ?? globalThis.document?.activeElement;
    const cargando = estado === "cargando";
    const destinoFoco = raiz.isConnected === false ? null
      : cargando && botonAnterior && focoActual === botonAnterior ? "estado"
        : !cargando && ((estadoAnterior && focoActual === estadoAnterior)
          || (botonAnterior && focoActual === botonAnterior)) ? "boton" : null;
    const contenido = estado === "listo" ? renderizarSeguimiento(datos, t, fechas)
      : estado === "error" ? `<p role="status">${escapar(t("seguimiento_incorporacion_error"))}</p>` : "";
    raiz.innerHTML = `<section data-ct-seguimiento-incorporacion>
      <h3>${escapar(t("seguimiento_incorporacion_titulo"))}</h3>
      <p><strong>${escapar(t("seguimiento_incorporacion_recibo"))}:</strong> <code>${escapar(reciboConfirmado?.recibo_ref ?? "—")}</code></p>
      <button type="button" class="boton-secundario" data-ct-seguimiento-consultar${cargando || !reciboConfirmado ? " disabled" : ""}>${escapar(t("seguimiento_incorporacion_consultar"))}</button>
      <p data-ct-seguimiento-estado-consulta role="status" aria-live="polite" tabindex="-1">${cargando ? escapar(t("seguimiento_incorporacion_cargando")) : ""}</p>
      ${!reciboConfirmado ? `<p role="status">${escapar(t("seguimiento_incorporacion_sin_recibo"))}</p>` : contenido}</section>`;
    if (destinoFoco && activo && raiz.isConnected !== false) {
      raiz.querySelector?.(destinoFoco === "estado"
        ? "[data-ct-seguimiento-estado-consulta]" : "[data-ct-seguimiento-consultar]")?.focus?.();
    }
  }

  async function consultar(evento) {
    if (!evento.target?.closest?.("[data-ct-seguimiento-consultar]") || controlador || !activo || !reciboConfirmado) return;
    controlador = new AbortController();
    const actual = controlador;
    estado = "cargando"; datos = null; pintar();
    try {
      const respuesta = await cliente.consultar(reciboConfirmado.expediente_ref, { signal: actual.signal });
      if (activo && !actual.signal.aborted) {
        datos = validarSeguimientoIncorporacion(respuesta, reciboConfirmado);
        estado = "listo";
      }
    } catch {
      if (activo && !actual.signal.aborted) estado = "error";
    } finally {
      if (controlador === actual) controlador = null;
      if (activo) pintar();
    }
  }

  raiz.addEventListener("click", consultar);
  pintar();
  return function destruir() {
    activo = false;
    controlador?.abort();
    raiz.removeEventListener("click", consultar);
    raiz.replaceChildren();
  };
}
