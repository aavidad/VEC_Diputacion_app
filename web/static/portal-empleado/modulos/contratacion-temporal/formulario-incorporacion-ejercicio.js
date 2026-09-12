import { validarSolicitudIncorporacionEjercicio, validarReciboIncorporacionEjercicio,
  validarPreparacionIncorporacionEjercicio } from "./contrato-incorporacion-ejercicio.js";
import { escaparHTML as e } from "./componentes-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

// Catálogo local de esta pieza: admite las mismas sobrescrituras que el módulo.
const textos = Object.freeze({
  titulo: "Incorporación de ejercicio", contexto: "Contexto de la incorporación",
  limites: "Ejercicio sintético: no acredita firma oficial ni eficacia administrativa.",
  expediente_ref: "Expediente", solicitud_personal_ref: "Solicitud de Personal",
  version_actual: "Versión actual del expediente", version_original: "Versión original del expediente",
  version_solicitud_personal: "Versión de la solicitud de Personal",
  version_seguimiento_esperada: "Versión esperada del seguimiento",
  desde: "Inicio del período", hasta: "Fin del período", documentos: "Documentos revisados",
  sin_documentos: "Sin referencias documentales", motivo: "Motivo del catálogo",
  seleccionar: "Seleccione un motivo", revision: "He revisado la solicitud de Personal, el período y los documentos indicados.",
  ejercicio: "Confirmo que esta actuación corresponde exclusivamente al ejercicio sintético.",
  confirmar: "Confirmar incorporación de ejercicio", reintentar: "Reintentar el mismo contenido",
  recibo: "Recibo original", recibo_ref: "Referencia del recibo", relacion_ref: "Relación de Personal",
  seguimiento_ref: "Seguimiento", actuacion_ref: "Actuación", auditoria_ref: "Auditoría", outbox_ref: "Evento de salida",
  version_seguimiento_anterior: "Versión anterior del seguimiento",
  version_seguimiento_resultante: "Versión resultante del seguimiento", registrada_en: "Fecha original de registro",
  esquema: "Esquema del recibo", ejercicio_sintetico: "Naturaleza sintética",
  firma_oficial: "Firma oficial", eficacia_administrativa: "Eficacia administrativa",
  detalles_trazabilidad: "Detalles de trazabilidad",
  pendiente: "Revise los datos y marque las dos confirmaciones antes de continuar.",
  no_disponible: "La preparación no está disponible para confirmar.",
  validacion: "Revise el motivo y las dos confirmaciones obligatorias.",
  cancelada: "No se ha iniciado un nuevo envío.", enviando: "Confirmando; espere la respuesta.",
  consultando: "Resultado incierto: consultando el recibo original.",
  incierta: "No se pudo verificar un recibo. El contenido queda bloqueado; solo puede reintentar la misma intención.",
  rechazada: "La solicitud ha sido rechazada. Revise los datos antes de continuar.",
  confirmada: "Recibo verificado de la operación.",
  historico: "Se muestra el recibo original conservado; no confirma que coincida con un intento divergente.",
  volver: "Volver al cuadro actualizado",
});

export function montarFormularioIncorporacionEjercicio({
  raiz, cliente, preparacion, confirmarOperacion = () => false,
  mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || typeof cliente?.prepararIncorporacionEjercicio !== "function"
    || typeof cliente?.confirmarIncorporacionEjercicio !== "function") {
    throw new TypeError("dependencias de incorporación no válidas");
  }
  const contexto = validarPreparacionIncorporacionEjercicio(preparacion, preparacion?.expediente_ref);
  const t = crearTraductorContratacionTemporal({
    ...Object.fromEntries(Object.entries(textos).map(([k, v]) => ["incorporacion_ejercicio_" + k, v])),
    ...mensajes,
  });
  const texto = (k) => e(t("incorporacion_ejercicio_" + k));
  const fila = (k, v) => `<div><dt>${texto(k)}</dt><dd>${e(String(v))}</dd></div>`;
  const siNo = (valor) => valor ? "Sí" : "No";
  const fecha = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria });
  const periodo = (p) => fila("desde", fecha.format(new Date(p.desde))) + fila("hasta", fecha.format(new Date(p.hasta)));
  let montado = true, ocupado = false, inmutable = false, controlador = null;
  let recibo = contexto.recibo, versionActual = contexto.version_actual_expediente, solicitud = null;
  let valores = { motivo_clave: "", confirma_revision_personal: false, confirma_ejercicio_sintetico: false };

  function pintar(mensaje = "") {
    if (!montado) return;
    const p = contexto.preparacion;
    raiz.innerHTML = `<section class="ct-alta" data-ct-incorporacion-ejercicio>
      <h3>${texto("titulo")}</h3><p class="ct-ayuda" id="ct-ie-ayuda">${texto("limites")}</p>
      <dl class="ct-resumen" aria-label="${texto("contexto")}">
        ${fila("expediente_ref", contexto.expediente_ref)}${fila("version_actual", versionActual)}
      </dl>
      ${recibo ? `<section class="ct-recibo" role="status"><h4>${texto("recibo")}</h4><dl>
        ${fila("recibo_ref", recibo.recibo_ref)}
        ${fila("registrada_en", `${fecha.format(new Date(recibo.registrada_en))} · ${recibo.registrada_en}`)}
        ${fila("ejercicio_sintetico", siNo(recibo.ejercicio_sintetico))}
        ${fila("firma_oficial", siNo(recibo.firma_oficial))}
        ${fila("eficacia_administrativa", siNo(recibo.eficacia_administrativa))}
        ${periodo(recibo.periodo_incorporacion)}</dl><details><summary>${texto("detalles_trazabilidad")}</summary><dl>
        ${["esquema", "expediente_ref", "solicitud_personal_ref", "relacion_ref", "seguimiento_ref", "actuacion_ref",
          "auditoria_ref", "outbox_ref", "version_solicitud_personal", "version_seguimiento_anterior",
          "version_seguimiento_resultante"].map((k) => fila(k, recibo[k])).join("")}
        ${fila("version_original", recibo.version_actual_expediente)}
        </dl></details><button type="button" class="boton-secundario" data-ct-exp-accion="volver-cuadro-actualizado">${texto("volver")}</button>
      </section>` : `<dl class="ct-resumen">
        ${fila("solicitud_personal_ref", p.solicitud_personal_ref)}${fila("version_solicitud_personal", p.version_solicitud_personal)}
        ${fila("version_seguimiento_esperada", p.version_seguimiento_esperada)}${periodo(p.periodo_incorporacion)}
        ${fila("documentos", p.documentos_refs.length ? p.documentos_refs.join(" · ") : t("incorporacion_ejercicio_sin_documentos"))}</dl>
        ${p.disponible ? `<form data-ct-incorporacion-ejercicio-form aria-busy="${ocupado}">
          <fieldset${ocupado || inmutable ? " disabled" : ""} aria-describedby="ct-ie-ayuda">
            <legend>${texto("contexto")}</legend>
            <div class="ct-campo"><label for="ct-ie-motivo">${texto("motivo")} *</label>
              <select id="ct-ie-motivo" name="motivo_clave" required><option value="">${texto("seleccionar")}</option>
              ${p.motivos.map((m) => `<option value="${e(m)}"${valores.motivo_clave === m ? " selected" : ""}>${e(m)}</option>`).join("")}</select></div>
            <div class="ct-campo"><label><input type="checkbox" name="confirma_revision_personal" required${valores.confirma_revision_personal ? " checked" : ""}> ${texto("revision")}</label></div>
            <div class="ct-campo"><label><input type="checkbox" name="confirma_ejercicio_sintetico" required${valores.confirma_ejercicio_sintetico ? " checked" : ""}> ${texto("ejercicio")}</label></div>
          </fieldset><div class="ct-acciones"><button type="submit" class="boton-primario"${ocupado ? " disabled" : ""}>${texto(inmutable ? "reintentar" : "confirmar")}</button></div>
        </form>` : ""}`}
      <p role="status" aria-live="polite">${texto(mensaje || (recibo ? "historico" : p.disponible ? "pendiente" : "no_disponible"))}</p>
    </section>`;
  }

  async function recuperar(signal) {
    pintar("consultando");
    try {
      const respuesta = await cliente.prepararIncorporacionEjercicio(contexto.expediente_ref, { signal });
      if (!montado || signal.aborted) return "incierta";
      const historico = validarPreparacionIncorporacionEjercicio(respuesta, contexto.expediente_ref);
      // Una historia válida no necesita tener la versión actual del primer GET.
      // Tampoco prueba que el material de un intento divergente fuese aceptado.
      if (historico.recibo && historico.recibo.solicitud_personal_ref === solicitud.solicitud_personal_ref) {
        recibo = historico.recibo;
        versionActual = historico.version_actual_expediente;
        return "historico";
      }
    } catch { /* No liberar la intención por un GET fallido o sin recibo. */ }
    return "incierta";
  }

  async function enviar(evento) {
    evento.preventDefault();
    if (!montado || ocupado || recibo || !contexto.preparacion?.disponible) return;
    const recuperando = inmutable;
    if (!inmutable) {
      try {
        const c = evento.target.elements;
        valores = { motivo_clave: c.motivo_clave.value,
          confirma_revision_personal: c.confirma_revision_personal.checked,
          confirma_ejercicio_sintetico: c.confirma_ejercicio_sintetico.checked };
        if (!contexto.preparacion.motivos.includes(valores.motivo_clave)) throw new TypeError();
        solicitud = validarSolicitudIncorporacionEjercicio({
          expediente_ref: contexto.expediente_ref,
          solicitud_personal_ref: contexto.preparacion.solicitud_personal_ref,
          version_actual_expediente_observada: contexto.version_actual_expediente,
          documentos_refs: [...contexto.preparacion.documentos_refs], ...valores,
        });
      } catch { solicitud = null; pintar("validacion"); return; }
    }
    let confirmado = false;
    try {
      confirmado = confirmarOperacion({ titulo: t("incorporacion_ejercicio_titulo"),
        advertencia: t("incorporacion_ejercicio_limites"), referencia: contexto.expediente_ref,
        datos: solicitud }) === true;
    } catch { /* No iniciar efectos sin confirmación expresa. */ }
    if (!confirmado) { pintar("cancelada"); return; }
    if (!montado) return;
    ocupado = true;
    controlador = new AbortController();
    const { signal } = controlador;
    let mensaje = "", respuestaRecibida = false;
    pintar("enviando");
    try {
      const respuesta = await cliente.confirmarIncorporacionEjercicio(solicitud, { signal });
      if (!montado || signal.aborted) return;
      respuestaRecibida = true;
      recibo = validarReciboIncorporacionEjercicio(respuesta, solicitud);
      mensaje = "confirmada";
    } catch (error) {
      if (!montado || signal.aborted) return;
      const determinado = !recuperando && !respuestaRecibida && error?.estado !== 409
        && error?.resultadoIndeterminado === false && error?.envelopeValido === true;
      if (determinado) { solicitud = null; mensaje = "rechazada"; }
      else { inmutable = true; mensaje = await recuperar(signal); }
    } finally {
      ocupado = false; controlador = null; pintar(mensaje);
    }
  }
  raiz.addEventListener("submit", enviar);
  pintar();
  return () => {
    montado = false; controlador?.abort();
    raiz.removeEventListener("submit", enviar); raiz.replaceChildren();
  };
}
