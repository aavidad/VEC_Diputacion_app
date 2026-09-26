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
 * Se ofrece sin incorporación, sin cese y sin no incorporación de la persona
 * de la propuesta vigente (la de una propuesta ya sustituida es historia).
 */
export function ofrecerNoIncorporacion(estado, opciones) {
  return Boolean(opciones?.no_incorporacion) && !estado.incorporacion && !estado.cese && !noIncorporacionVigente(estado);
}

/** Fila del resumen cuando consta la no incorporación. */
export function filasNoIncorporacion(estado, opciones, t, fecha) {
  const n = noIncorporacionVigente(estado);
  if (!n) return [];
  const motivo = opciones?.no_incorporacion?.motivos?.find((m) => m.clave === n.motivo_clave);
  return [[t("no_incorporacion"), t("no_incorporacion_registrada", { motivo: motivo ? motivo.etiqueta : n.motivo_clave, fecha: fecha(n.fecha_notificacion) })]];
}

export function formularioNoIncorporacion({ opciones, t, escapar, deshabilitado }) {
  const regla = opciones.no_incorporacion;
  const motivos = regla.motivos.map((m) => `<option value="${escapar(m.clave)}">${escapar(m.etiqueta)}</option>`).join("");
  const desactivado = deshabilitado === "disabled";
  return `<form class="ct-seg-cese-form" data-ct-seg-form="no_incorporacion" aria-labelledby="ct-seg-no-inc-titulo" novalidate>
      <h4 id="ct-seg-no-inc-titulo">${escapar(t("no_incorporacion_titulo"))}</h4>
      <div class="ct-seg-cese-rejilla">
      <label class="ct-campo"><span>${escapar(t("no_incorporacion_motivo"))}</span><select name="motivo_clave" required ${deshabilitado}>${motivos}</select></label>
      <label class="ct-campo"><span>${escapar(t("no_incorporacion_fecha"))}</span><input type="date" name="fecha_notificacion" required ${deshabilitado}></label>
      <label class="ct-campo"><span>${escapar(t("no_incorporacion_resolucion"))}</span><input type="text" name="resolucion_ref" required maxlength="160" autocomplete="off" ${deshabilitado}></label>
      ${renderizarCampoHuellaArchivo({ id: "ct-seg-no-inc-resolucion", nombre: "resolucion_sha256", etiqueta: t("no_incorporacion_archivo"),
        clase: "ct-campo", deshabilitado: desactivado, escapar })}
      <label class="ct-campo"><span>${escapar(t(regla.segunda_persona ? "no_incorporacion_resuelta_segunda" : "no_incorporacion_resuelta"))}</span><input type="text" name="resuelta_por" required maxlength="160" autocomplete="off" ${deshabilitado}></label>
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("no_incorporacion_observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" ${deshabilitado}></textarea></label>
      </div>
      <div class="ct-acciones"><button type="submit" class="boton-primario" ${deshabilitado}>${escapar(t("no_incorporacion_enviar"))}</button></div></form>`;
}

/** Solicitud para el cliente HTTP a partir del formulario. */
export function solicitudNoIncorporacion(base, campos) {
  return ["registrarNoIncorporacion", { ...base, motivo_clave: campos.motivo_clave ?? "", resolucion_ref: campos.resolucion_ref ?? "",
    resolucion_sha256: (campos.resolucion_sha256 ?? "").toLowerCase(), resuelta_por: campos.resuelta_por ?? "",
    fecha_notificacion: campos.fecha_notificacion ?? "", observaciones: campos.observaciones ?? "" }];
}
