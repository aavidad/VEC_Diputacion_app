/** Proyección de resultados conservados por Bolsa; no ejecuta ni deduce una respuesta. */
import { justificanteTraducido } from "../../portal-justificante.js";

const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");

export function aceptacionBolsaCompleta(expediente) {
  const resultado = expediente?.resultado_bolsa;
  return Number.isSafeInteger(resultado?.personas_solicitadas) && resultado.personas_solicitadas > 0
    && Number.isSafeInteger(resultado.aceptaciones_firmes)
    && resultado.aceptaciones_firmes >= resultado.personas_solicitadas;
}

function fecha(valor, locale, zonaHoraria) {
  const instante = new Date(valor);
  return valor && Number.isFinite(instante.getTime())
    ? `<time datetime="${escapar(valor)}">${escapar(new Intl.DateTimeFormat(locale, {
      dateStyle: "medium", timeStyle: "short", timeZone: zonaHoraria,
    }).format(instante))}</time>` : "—";
}

function enlaceSeguimiento(vinculo, t) {
  return `<button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="${escapar(vinculo.bolsa_ref)}" data-seguimiento="${escapar(vinculo.llamamiento_ref)}">${escapar(t("resultado_bolsa_seguimiento"))}</button>`;
}

export function continuidadBolsaDisponible(expediente) {
  const vinculos = expediente?.resultado_bolsa?.vinculos;
  if (!Array.isArray(vinculos) || !vinculos.length || aceptacionBolsaCompleta(expediente)) return false;
  const normalizar = (instante) => instante.replace(/(?:\.(\d+))?Z$/u,
    (_final, fraccion = "") => `.${fraccion.padEnd(9, "0")}Z`);
  const ultimo = [...vinculos].sort((a, b) => normalizar(a.emitido_en).localeCompare(normalizar(b.emitido_en))).at(-1);
  return ultimo.participaciones?.length > 0 && ultimo.participaciones.every((persona) =>
    (persona.modo === "firme" && ["renuncia", "renuncia_justificada"].includes(persona.respuesta))
      || (persona.respuesta === null && persona.situacion_actual === "renuncia"));
}

function resultadoPersona(persona, t, locale, zonaHoraria) {
  const respuesta = persona.respuesta && persona.modo === "firme";
  const pendiente = persona.respuesta && persona.modo === "propuesta_rrhh";
  const renunciaSituacion = persona.situacion_actual === "renuncia";
  const clave = respuesta ? `resultado_bolsa_${persona.respuesta}` : pendiente
    ? "resultado_bolsa_propuesta" : "resultado_bolsa_sin_respuesta";
  return `<dl class="ct-exp-datos">
    <div><dt>${escapar(t("resultado_bolsa_respuesta"))}</dt><dd>${escapar(t(clave))}</dd></div>
    ${persona.contacto_resultado ? `<div><dt>${escapar(t("resultado_bolsa_contacto"))}</dt><dd>${escapar(t(`resultado_bolsa_contacto_${persona.contacto_resultado}`))}</dd></div>
      <div><dt>${escapar(t("resultado_bolsa_fecha_contacto"))}</dt><dd>${fecha(persona.contacto_en, locale, zonaHoraria)}</dd></div>
      <div><dt>${escapar(t("resultado_bolsa_justificante_contacto"))}</dt><dd>${justificanteTraducido(persona.recibo_contacto_ref, escapar, t)}</dd></div>` : ""}
    ${persona.respuesta ? `<div><dt>${escapar(t("resultado_bolsa_fecha"))}</dt><dd>${fecha(persona.respondida_en, locale, zonaHoraria)}</dd></div>
      <div><dt>${escapar(t("resultado_bolsa_justificante"))}</dt><dd>${justificanteTraducido(persona.recibo_respuesta_ref, escapar, t)}</dd></div>` : ""}
    ${!respuesta && renunciaSituacion ? `<div><dt>${escapar(t("resultado_bolsa_situacion"))}</dt><dd>${escapar(t("resultado_bolsa_situacion_renuncia"))}</dd></div>
      <div><dt>${escapar(t("resultado_bolsa_fecha_situacion"))}</dt><dd>${fecha(persona.situacion_desde, locale, zonaHoraria)}</dd></div>
      <div><dt>${escapar(t("resultado_bolsa_justificante_situacion"))}</dt><dd>${justificanteTraducido(persona.recibo_situacion_ref, escapar, t)}</dd></div>` : ""}
  </dl>`;
}

export function renderizarResultadoBolsa(expediente, t, locale, zonaHoraria, estado = {}) {
  const resultado = expediente?.resultado_bolsa;
  if (!resultado) return "";
  const vinculos = estado.vinculos ?? resultado.vinculos ?? [];
  const candidatas = resultado.emisiones_vinculables ?? [];
  const ocupado = estado.ocupado === true;
  const contenido = vinculos.map((vinculo) => `<article class="panel-separado">
    <div class="cabecera-panel"><h4>${escapar(t("resultado_bolsa_llamamiento"))} · ${fecha(vinculo.emitido_en, locale, zonaHoraria)}</h4>${enlaceSeguimiento(vinculo, t)}</div>
    <div class="cuerpo-panel">${(vinculo.participaciones ?? []).map(
      (persona) => resultadoPersona(persona, t, locale, zonaHoraria),
    ).join("")}</div></article>`).join("");
  const opciones = candidatas.map((emision, indice) => `<option value="${indice}">${escapar(emision.referencia_visible)} · ${escapar(new Intl.DateTimeFormat(locale, {
    dateStyle: "medium", timeStyle: "short", timeZone: zonaHoraria,
  }).format(new Date(emision.emitido_en)))}</option>`).join("");
  const seleccion = candidatas.length ? `<form data-ct-bolsa-vincular>
    <label class="ct-exp-campo" for="ct-bolsa-emision"><span>${escapar(t("resultado_bolsa_elegir"))}</span>
      <select id="ct-bolsa-emision" name="emision" required${ocupado ? " disabled" : ""}>
        <option value="">${escapar(t("resultado_bolsa_seleccionar"))}</option>${opciones}</select></label>
    <button type="submit" class="boton-secundario"${ocupado ? " disabled" : ""}>${escapar(t("resultado_bolsa_vincular"))}</button>
  </form>` : "";
  return `<section class="panel" aria-labelledby="ct-resultado-bolsa-titulo" data-ct-resultado-bolsa>
    <div class="cabecera-panel"><h3 id="ct-resultado-bolsa-titulo">${escapar(t("resultado_bolsa_titulo"))}</h3></div>
    <div class="cuerpo-panel">${contenido}${!vinculos.length && !candidatas.length ? `<p>${escapar(t("resultado_bolsa_sin_llamamientos"))}</p>` : ""}
      ${seleccion}<div class="acciones-vista">${estado.anterior ? `<button type="button" class="boton-secundario" data-ct-bolsa-anterior${ocupado ? " disabled" : ""}>${escapar(t("resultado_bolsa_anterior"))}</button>` : ""}
      ${(Object.hasOwn(estado, "siguiente_cursor") ? estado.siguiente_cursor : resultado.siguiente_cursor) ? `<button type="button" class="boton-secundario" data-ct-bolsa-mas${ocupado ? " disabled" : ""}>${escapar(t("resultado_bolsa_mas"))}</button>` : ""}</div>
      <p role="status" aria-live="polite" data-ct-bolsa-estado>${estado.mensaje ? escapar(t(estado.mensaje)) : ""}</p>
    </div></section>`;
}
