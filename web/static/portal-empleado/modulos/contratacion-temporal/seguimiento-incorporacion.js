import { validarPreparacionIncorporacionEjercicio } from "./contrato-incorporacion-ejercicio.js";
import { validarSeguimientoIncorporacion } from "./contrato-seguimiento-incorporacion.js";
import { escaparHTML as escapar } from "./componentes-expedientes.js";

export const CLAVES_I18N_SEGUIMIENTO_INCORPORACION = Object.freeze([
  "seguimiento_incorporacion_titulo", "seguimiento_incorporacion_alcance",
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
]);

export const MENSAJES_SEGUIMIENTO_INCORPORACION_ES = Object.freeze({
  seguimiento_incorporacion_titulo: "Seguimiento de la incorporación",
  seguimiento_incorporacion_alcance:
    "Consulta del seguimiento original asociado al recibo confirmado. No permite anotar, cerrar ni alterar el expediente.",
  seguimiento_incorporacion_consultar: "Consultar seguimiento original",
  seguimiento_incorporacion_cargando: "Consultando el seguimiento original…",
  seguimiento_incorporacion_error: "No se ha podido consultar el seguimiento original.",
  seguimiento_incorporacion_recibo: "Recibo de incorporación",
  seguimiento_incorporacion_estado: "Estado posterior histórico",
  seguimiento_incorporacion_periodo: "Período",
  seguimiento_incorporacion_registrado: "Registrado",
  seguimiento_incorporacion_hitos: "Actuaciones históricas",
  seguimiento_incorporacion_sin_hitos: "No constan actuaciones históricas.",
  seguimiento_incorporacion_documentos: "Documentos",
  seguimiento_incorporacion_sin_documentos: "Sin documentos referenciados.",
  seguimiento_incorporacion_sin_recibo: "El seguimiento solo se consulta desde un recibo V2 confirmado.",
  seguimiento_incorporacion_referencia: "Referencia",
  seguimiento_incorporacion_transicion: "Transición",
  seguimiento_incorporacion_efectiva: "Efectiva",
  seguimiento_incorporacion_expediente: "Expediente",
  seguimiento_incorporacion_version_expediente: "Versión de expediente",
  seguimiento_incorporacion_seguimiento: "Seguimiento",
  seguimiento_incorporacion_version_seguimiento: "Versión de seguimiento",
});

function texto(mensajes, clave) {
  return typeof mensajes[clave] === "string"
    ? mensajes[clave] : MENSAJES_SEGUIMIENTO_INCORPORACION_ES[clave];
}

function fila(etiqueta, valor) {
  return `<div><dt>${escapar(etiqueta)}</dt><dd><code>${escapar(String(valor))}</code></dd></div>`;
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

function renderizarSeguimiento(datos, mensajes, fechas) {
  const hitos = datos.actuaciones.length === 0
    ? `<p>${escapar(texto(mensajes, "seguimiento_incorporacion_sin_hitos"))}</p>`
    : `<ol>${datos.actuaciones.map((actuacion) => {
      const documentos = actuacion.documentos.length === 0
        ? escapar(texto(mensajes, "seguimiento_incorporacion_sin_documentos"))
        : actuacion.documentos.map(({ tipo_clave: tipo, referencia }) =>
          `<li><code>${escapar(tipo)}</code>: <code>${escapar(referencia)}</code></li>`).join("");
      return `<li><dl>${fila(texto(mensajes, "seguimiento_incorporacion_referencia"), actuacion.actuacion_ref)}
        ${fila(texto(mensajes, "seguimiento_incorporacion_transicion"), actuacion.transicion_clave)}
        ${fila(texto(mensajes, "seguimiento_incorporacion_estado"), `${actuacion.estado_origen} → ${actuacion.estado_destino}`)}
        ${filaFecha(texto(mensajes, "seguimiento_incorporacion_efectiva"), actuacion.efectivo_en, fechas.instante)}${filaFecha(texto(mensajes, "seguimiento_incorporacion_registrado"), actuacion.registrada_en, fechas.instante)}
        </dl><h5>${escapar(texto(mensajes, "seguimiento_incorporacion_documentos"))}</h5><ul>${documentos}</ul></li>`;
    }).join("")}</ol>`;
  return `<dl class="ct-resumen">${fila(texto(mensajes, "seguimiento_incorporacion_expediente"), datos.expediente_ref)}
    ${fila(texto(mensajes, "seguimiento_incorporacion_version_expediente"), datos.version_expediente)}
    ${fila(texto(mensajes, "seguimiento_incorporacion_seguimiento"), datos.seguimiento_ref)}
    ${fila(texto(mensajes, "seguimiento_incorporacion_version_seguimiento"), datos.version_seguimiento)}
    ${fila(texto(mensajes, "seguimiento_incorporacion_estado"), datos.estado_clave)}
    ${filaPeriodo(texto(mensajes, "seguimiento_incorporacion_periodo"), datos.periodo, fechas.fechaCivil)}
    ${filaFecha(texto(mensajes, "seguimiento_incorporacion_registrado"), datos.registrado_en, fechas.instante)}</dl>
    <h4>${escapar(texto(mensajes, "seguimiento_incorporacion_hitos"))}</h4>${hitos}`;
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
  let activo = true, controlador = null, estado = "inicial", datos = null;

  function pintar() {
    if (!activo) return;
    const cargando = estado === "cargando";
    const contenido = estado === "listo" ? renderizarSeguimiento(datos, mensajes, fechas)
      : estado === "error" ? `<p role="status">${escapar(texto(mensajes, "seguimiento_incorporacion_error"))}</p>` : "";
    raiz.innerHTML = `<section data-ct-seguimiento-incorporacion>
      <h3>${escapar(texto(mensajes, "seguimiento_incorporacion_titulo"))}</h3>
      <p class="ct-ayuda">${escapar(texto(mensajes, "seguimiento_incorporacion_alcance"))}</p>
      <p><strong>${escapar(texto(mensajes, "seguimiento_incorporacion_recibo"))}:</strong> <code>${escapar(reciboConfirmado?.recibo_ref ?? "—")}</code></p>
      <button type="button" class="boton-secundario" data-ct-seguimiento-consultar${cargando || !reciboConfirmado ? " disabled" : ""}>${escapar(texto(mensajes, "seguimiento_incorporacion_consultar"))}</button>
      <p role="status" aria-live="polite">${cargando ? escapar(texto(mensajes, "seguimiento_incorporacion_cargando")) : ""}</p>
      ${!reciboConfirmado ? `<p role="status">${escapar(texto(mensajes, "seguimiento_incorporacion_sin_recibo"))}</p>` : contenido}</section>`;
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
