/**
 * No incorporación en el panel de seguimiento (duda 12 de RRHH): la persona
 * aceptada y nombrada no llega a incorporarse. RRHH la registra con un
 * motivo del catálogo, la resolución (referencia y archivo, del que solo se
 * registra la huella), quién la resolvió y la fecha de notificación. El
 * servidor decide la consecuencia en la bolsa y el siguiente llamamiento;
 * aquí solo se pinta.
 */

import { renderizarCampoHuellaArchivo } from "../../portal-huella-archivo.js";
import { noIncorporacionVigente } from "./seguimiento-propuestas.js?v=20260926-propuesta-sucesor-v1";

/**
 * Se ofrece sin incorporación, sin cese, sin no incorporación de la persona
 * de la propuesta vigente (la de una propuesta ya sustituida es historia) y
 * sin propuesta pendiente. Con segunda persona (c22) el formulario propone;
 * otra persona confirma o rechaza después.
 */
export function ofrecerNoIncorporacion(estado, opciones) {
  return Boolean(opciones?.no_incorporacion) && !estado.incorporacion && !estado.cese && !noIncorporacionVigente(estado)
    && !estado.no_incorporacion_propuesta;
}

/** La propuesta pendiente se ofrece para confirmar o rechazar. */
export function ofrecerPropuestaNoIncorporacion(estado, opciones) {
  return Boolean(opciones?.no_incorporacion?.segunda_persona) && Boolean(estado.no_incorporacion_propuesta)
    && !estado.incorporacion && !estado.cese && !noIncorporacionVigente(estado);
}

function etiquetaMotivo(opciones, clave) {
  const motivo = opciones?.no_incorporacion?.motivos?.find((m) => m.clave === clave);
  return motivo ? motivo.etiqueta : clave;
}

/** Fila del resumen cuando consta la no incorporación o su propuesta. */
export function filasNoIncorporacion(estado, opciones, t, fecha) {
  const n = noIncorporacionVigente(estado);
  if (n) return [[t("no_incorporacion"), t("no_incorporacion_registrada", { motivo: etiquetaMotivo(opciones, n.motivo_clave), fecha: fecha(n.fecha_notificacion) })]];
  const p = estado.no_incorporacion_propuesta;
  if (p) return [[t("no_incorporacion"), t("no_incorporacion_propuesta_pendiente", { motivo: etiquetaMotivo(opciones, p.motivo_clave), fecha: fecha(p.fecha_notificacion) })]];
  return [];
}

export function formularioNoIncorporacion({ opciones, t, escapar, deshabilitado }) {
  const regla = opciones.no_incorporacion;
  const motivos = regla.motivos.map((m) => `<option value="${escapar(m.clave)}">${escapar(m.etiqueta)}</option>`).join("");
  const desactivado = deshabilitado === "disabled";
  // Con segunda persona nadie declara quién resuelve: lo resuelve quien confirma.
  const resuelta = regla.segunda_persona ? ""
    : `<label class="ct-campo"><span>${escapar(t("no_incorporacion_resuelta"))}</span><input type="text" name="resuelta_por" required maxlength="160" autocomplete="off" ${deshabilitado}></label>`;
  const titulo = regla.segunda_persona ? "no_incorporacion_titulo_proponer" : "no_incorporacion_titulo";
  return `<form class="ct-seg-cese-form" data-ct-seg-form="no_incorporacion" aria-labelledby="ct-seg-no-inc-titulo" novalidate>
      <h4 id="ct-seg-no-inc-titulo">${escapar(t(titulo))}</h4>
      <div class="ct-seg-cese-rejilla">
      <label class="ct-campo"><span>${escapar(t("no_incorporacion_motivo"))}</span><select name="motivo_clave" required ${deshabilitado}>${motivos}</select></label>
      <label class="ct-campo"><span>${escapar(t("no_incorporacion_fecha"))}</span><input type="date" name="fecha_notificacion" required ${deshabilitado}></label>
      <label class="ct-campo"><span>${escapar(t("no_incorporacion_resolucion"))}</span><input type="text" name="resolucion_ref" required maxlength="160" autocomplete="off" ${deshabilitado}></label>
      ${renderizarCampoHuellaArchivo({ id: "ct-seg-no-inc-resolucion", nombre: "resolucion_sha256", etiqueta: t("no_incorporacion_archivo"),
        clase: "ct-campo", deshabilitado: desactivado, escapar })}
      ${resuelta}
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("no_incorporacion_observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" ${deshabilitado}></textarea></label>
      </div>
      <div class="ct-acciones"><button type="submit" class="boton-primario" ${deshabilitado}>${escapar(t(regla.segunda_persona ? "no_incorporacion_proponer" : "no_incorporacion_enviar"))}</button></div></form>`;
}

/** Propuesta pendiente: sus datos y los dos formularios de la segunda persona. */
export function formularioPropuestaNoIncorporacion({ estado, opciones, t, escapar, deshabilitado, fecha }) {
  const p = estado.no_incorporacion_propuesta;
  const paso = (tipo, titulo, boton, clase) => `<form class="ct-seg-cese-form" data-ct-seg-form="${tipo}" aria-labelledby="ct-seg-${tipo}-titulo" novalidate>
      <h5 id="ct-seg-${tipo}-titulo">${escapar(t(titulo))}</h5>
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("no_incorporacion_observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" ${deshabilitado}></textarea></label>
      <div class="ct-acciones"><button type="submit" class="${clase}" ${deshabilitado}>${escapar(t(boton))}</button></div></form>`;
  return `<section class="ct-seg-cese-form" aria-labelledby="ct-seg-no-inc-prop-titulo" data-ct-seg-propuesta>
      <h4 id="ct-seg-no-inc-prop-titulo">${escapar(t("no_incorporacion_propuesta_titulo"))}</h4>
      <p>${escapar(t("no_incorporacion_registrada", { motivo: etiquetaMotivo(opciones, p.motivo_clave), fecha: fecha(p.fecha_notificacion) }))}</p>
      ${paso("no_incorporacion_confirmar", "no_incorporacion_confirmar_titulo", "no_incorporacion_confirmar", "boton-primario")}
      ${paso("no_incorporacion_rechazar", "no_incorporacion_rechazar_titulo", "no_incorporacion_rechazar", "boton-secundario")}</section>`;
}

/**
 * Solicitud para el cliente HTTP a partir del formulario: registro de un
 * paso sin segunda persona; propuesta con ella (sin quién resuelve).
 */
export function solicitudNoIncorporacion(base, campos, regla) {
  const segunda = Boolean(regla?.segunda_persona);
  return ["registrarNoIncorporacion", { ...base, paso: segunda ? "proponer" : "registrar", propuesta_ref: "",
    motivo_clave: campos.motivo_clave ?? "", resolucion_ref: campos.resolucion_ref ?? "",
    resolucion_sha256: (campos.resolucion_sha256 ?? "").toLowerCase(), resuelta_por: segunda ? "" : (campos.resuelta_por ?? ""),
    fecha_notificacion: campos.fecha_notificacion ?? "", observaciones: campos.observaciones ?? "" }];
}

/**
 * Confirmación o rechazo de la propuesta pendiente: sus mismos datos (que el
 * servidor coteja) y las observaciones de quien la resuelve, que es siempre
 * la persona autenticada.
 */
export function solicitudResolucionPropuestaNoIncorporacion(base, propuesta, paso, campos) {
  return ["registrarNoIncorporacion", { ...base, paso, propuesta_ref: propuesta?.propuesta_ref ?? "", motivo_clave: propuesta?.motivo_clave ?? "",
    resolucion_ref: propuesta?.resolucion_ref ?? "", resolucion_sha256: propuesta?.resolucion_sha256 ?? "", resuelta_por: "",
    fecha_notificacion: propuesta?.fecha_notificacion ?? "", observaciones: campos.observaciones ?? "" }];
}
