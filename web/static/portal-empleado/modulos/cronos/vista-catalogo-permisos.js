import { MENSAJES_CRONOS_C6_ES } from "./i18n-c6.js?v=20260925-aspecto-v1";

const TIPOS = Object.freeze([
  "asistencia_examenes", "asuntos_propios", "dias_trienios", "dias_servicio",
  "conciliacion", "compensacion_festivos", "compensacion_horas_extra",
  "compensacion_sabados", "compensacion_tiempo_ocio", "contrato_relevo",
  "formacion", "embarazo_maternidad", "enfermedad_familiar",
  "enfermedad_sin_baja", "fallecimiento_familiar", "gestion_servicio",
  "horas_medico", "horas_sindicales", "miercoles_semana_santa", "corpus",
  "nacimiento", "progenitor_distinto", "trabajo_no_presencial",
  "traslado_domicilio", "vacaciones",
]);

const CAMPOS = Object.freeze(["unidad", "computo", "maximo", "minimo", "autorizador", "justificante"]);

function texto(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function traductor(mensajes) {
  return (clave, variables = {}) => String(mensajes?.[clave] ?? MENSAJES_CRONOS_C6_ES[clave] ?? "")
    .replace(/\{([a-z_]+)\}/gu, (_coincidencia, variable) => String(variables[variable] ?? ""));
}

function normalizar(valor) {
  return String(valor ?? "").slice(0, 120).normalize("NFD")
    .replace(/\p{Diacritic}/gu, "").toLocaleLowerCase("es").trim();
}

/** Filtro local de nombres; no consulta saldos ni servicios. */
export function filtrarCatalogoPermisosCronos(consulta = "", mensajes = MENSAJES_CRONOS_C6_ES) {
  const t = traductor(mensajes);
  const termino = normalizar(consulta);
  return TIPOS.filter((id) => normalizar(t(`tipo_${id}`)).includes(termino));
}

/** Consulta local provisional, independiente del formulario y de toda autorización. */
export function renderizarCatalogoPermisosCronos({ consulta = "", mensajes = MENSAJES_CRONOS_C6_ES } = {}) {
  const t = traductor(mensajes);
  const visibles = new Set(filtrarCatalogoPermisosCronos(consulta, mensajes));
  const cantidad = new Intl.NumberFormat("es-ES").format(visibles.size);
  const campo = (clave) => `<div><dt>${texto(t(clave))}</dt><dd>${texto(t("pendiente"))}</dd></div>`;
  const filas = TIPOS.map((id) => `<li class="cronos-c6-fila" data-cronos-c6-tipo="${id}"${visibles.has(id) ? "" : " hidden"}>
    <details><summary><span class="cronos-c6-nombre">${texto(t(`tipo_${id}`))}</span><span class="cronos-c6-estado">${texto(t("provisional"))}</span></summary>
      <div class="cronos-c6-detalle"><p>${texto(t("ver_detalle"))}</p><dl>${CAMPOS.map(campo).join("")}<div><dt>${texto(t("procedencia"))}</dt><dd>${texto(t("procedencia_valor"))}</dd></div></dl></div>
    </details></li>`).join("");
  return `<section class="panel cronos-c6" aria-labelledby="cronos-c6-titulo" data-cronos-c6-catalogo="provisional">
    <header class="cabecera-panel"><div><h3 id="cronos-c6-titulo">${texto(t("titulo"))}</h3></div><span class="cronos-c6-estado">${texto(t("provisional"))}</span></header>
    <div class="cuerpo-panel"><label class="cronos-c6-busqueda" for="cronos-c6-buscar"><span>${texto(t("buscar"))}</span><input id="cronos-c6-buscar" type="search" maxlength="120" autocomplete="off" value="${texto(String(consulta).slice(0, 120))}" data-cronos-c6-buscar aria-describedby="cronos-c6-ayuda"></label>
      <p id="cronos-c6-ayuda" class="cronos-c6-ayuda">${texto(t("buscar_ayuda"))}</p>
      <p class="cronos-c6-resultados" data-cronos-c6-resultados role="status">${texto(t("resultados", { cantidad }))}</p>
      <p class="cronos-c6-vacio" data-cronos-c6-vacio role="status"${visibles.size ? " hidden" : ""}>${texto(t("sin_resultados"))}</p>
      <ul class="cronos-c6-lista">${filas}</ul><p class="cronos-c6-aviso">${texto(t("aviso"))}</p></div>
  </section>`;
}

export function montarCatalogoPermisosCronos({ raiz, registrarDesmontar, mensajes = MENSAJES_CRONOS_C6_ES } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("contenedor de catálogo de Cronos no disponible");
  }
  const contenedor = raiz.ownerDocument.createElement("div");
  contenedor.innerHTML = renderizarCatalogoPermisosCronos({ mensajes });
  raiz.append(contenedor);
  const t = traductor(mensajes);
  const actualizar = (evento) => {
    if (!evento.target?.matches?.("[data-cronos-c6-buscar]")) return;
    const visibles = new Set(filtrarCatalogoPermisosCronos(evento.target.value, mensajes));
    contenedor.querySelectorAll?.("[data-cronos-c6-tipo]").forEach((fila) => { fila.hidden = !visibles.has(fila.dataset.cronosC6Tipo); });
    const estado = contenedor.querySelector?.("[data-cronos-c6-resultados]");
    if (estado) estado.textContent = t("resultados", { cantidad: new Intl.NumberFormat("es-ES").format(visibles.size) });
    const vacio = contenedor.querySelector?.("[data-cronos-c6-vacio]");
    if (vacio) vacio.hidden = visibles.size !== 0;
  };
  contenedor.addEventListener?.("input", actualizar);
  let activo = true;
  const desmontar = () => {
    if (!activo) return;
    activo = false;
    contenedor.removeEventListener?.("input", actualizar);
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
