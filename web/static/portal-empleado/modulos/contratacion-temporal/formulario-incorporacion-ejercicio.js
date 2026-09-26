import { validarSolicitudIncorporacionEjercicio, validarReciboIncorporacionEjercicio,
  validarPreparacionIncorporacionEjercicio } from "./contrato-incorporacion-ejercicio.js";
import { escaparHTML as e } from "./componentes-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { justificanteTraducido } from "../../portal-justificante.js";

// Catálogo local de esta pieza: admite las mismas sobrescrituras que el módulo.
const textos = Object.freeze({
  titulo: "Incorporación", contexto: "Contexto de la incorporación",
  limites: "La incorporación no acredita firma oficial ni eficacia administrativa.",
  expediente_ref: "Expediente", solicitud_personal_ref: "Solicitud de Personal",
  version_actual: "Versión actual del expediente", version_original: "Versión original del expediente",
  version_valor: "Versión {version}", documentos_numero: "{numero} documento(s) indicado(s)",
  motivo_ejercicio_incorporacion: "Incorporación efectiva",
  motivo_cierre_administrativo_ejercicio: "Cierre administrativo",
  version_solicitud_personal: "Versión de la solicitud de Personal",
  version_seguimiento_esperada: "Versión esperada del seguimiento",
  desde: "Inicio del período", hasta: "Fin del período", documentos: "Documentos revisados",
  sin_documentos: "Sin referencias documentales", motivo: "Motivo del catálogo",
  seleccionar: "Seleccione un motivo", revision: "He revisado la solicitud de Personal, el período y los documentos indicados.",
  ejercicio: "Confirmo que esta actuación no acredita firma oficial ni eficacia administrativa.",
  confirmar: "Confirmar incorporación", reintentar: "Reintentar el mismo contenido",
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
  // Versión en palabras; las referencias y versiones técnicas viajan solo en el contrato.
  const filaVersion = (k, v) => fila(k, t("incorporacion_ejercicio_version_valor", { version: v }));
  // Motivo del catálogo por su nombre; una clave sin traducir se muestra sin guiones bajos.
  const motivoLegible = (m) => {
    try { return t(`incorporacion_ejercicio_motivo_${m}`); } catch { return m.replaceAll("_", " "); }
  };
  // Los límites del período se presentan por su día civil UTC; el instante de
  // registro sí muestra la hora en la zona elegida por quien consulta el recibo.
  const fechaCivil = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeZone: "UTC" });
  const fechaRegistro = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria });
  const periodo = (p) => fila("desde", fechaCivil.format(new Date(p.desde)))
    + fila("hasta", fechaCivil.format(new Date(p.hasta)));
  let montado = true, ocupado = false, inmutable = false, controlador = null;
  let recibo = contexto.recibo, versionActual = contexto.version_actual_expediente, solicitud = null;
  let valores = { motivo_clave: "", confirma_revision_personal: false, confirma_ejercicio_sintetico: false };

  function pintar(mensaje = "") {
    if (!montado) return;
    const p = contexto.preparacion;
    raiz.innerHTML = `<section class="ct-alta" data-ct-incorporacion-ejercicio>
      <h3>${texto("titulo")}</h3>
      <dl class="ct-resumen" aria-label="${texto("contexto")}">
        ${filaVersion("version_actual", versionActual)}
      </dl>
      ${recibo ? `<section class="ct-recibo" role="status"><h4>${texto("recibo")}</h4><dl>
        <div><dt>${texto("recibo_ref")}</dt><dd>${justificanteTraducido(recibo.recibo_ref, e, t)}</dd></div>
        <div><dt>${texto("registrada_en")}</dt><dd><time datetime="${e(recibo.registrada_en)}">${e(fechaRegistro.format(new Date(recibo.registrada_en)))}</time></dd></div>
        ${fila("firma_oficial", siNo(recibo.firma_oficial))}
        ${fila("eficacia_administrativa", siNo(recibo.eficacia_administrativa))}
        ${periodo(recibo.periodo_incorporacion)}
        ${filaVersion("version_original", recibo.version_actual_expediente)}</dl>
        <button type="button" class="boton-secundario" data-ct-exp-accion="volver-cuadro-actualizado">${texto("volver")}</button>
      </section>` : `<dl class="ct-resumen">
        ${periodo(p.periodo_incorporacion)}
        ${fila("documentos", p.documentos_refs.length ? t("incorporacion_ejercicio_documentos_numero", { numero: p.documentos_refs.length }) : t("incorporacion_ejercicio_sin_documentos"))}</dl>
        ${p.disponible ? `<form data-ct-incorporacion-ejercicio-form aria-busy="${ocupado}">
          <fieldset${ocupado || inmutable ? " disabled" : ""}>
            <legend>${texto("contexto")}</legend>
            <div class="ct-campo"><label for="ct-ie-motivo">${texto("motivo")} *</label>
              <select id="ct-ie-motivo" name="motivo_clave" required><option value="">${texto("seleccionar")}</option>
              ${p.motivos.map((m) => `<option value="${e(m)}"${valores.motivo_clave === m ? " selected" : ""}>${e(motivoLegible(m))}</option>`).join("")}</select></div>
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
