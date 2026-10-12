/** Vista pura de cartera. Las versiones y recibos proceden exclusivamente de la API. */
const ESTADOS = new Set(["cargando", "vacio", "listo", "error", "denegado", "conflicto", "resultado_indeterminado"]);
const FILTROS = new Set(["todos", "disponibles", "retirados"]);
const TIPOS = new Set(["titulacion", "curso_asistencia", "curso_superacion", "experiencia", "idioma", "otro"]);

export function escaparMeritos(valor) {
  return String(valor ?? "").replace(/[&<>"']/gu, (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[caracter]);
}

const t = (textos, clave, variables) => escaparMeritos(textos.traducir(`propio.${clave}`, variables));

function ficha(hecho, textos, puedeRetirar) {
  if (!hecho || !TIPOS.has(hecho.tipo) || !["disponible", "retirado"].includes(hecho.disponibilidad?.estado)
    || !Number.isInteger(hecho.version_hecho) || hecho.version_hecho < 1) {
    throw new TypeError("meritos.cartera.ficha_invalida");
  }
  const disponible = hecho.disponibilidad.estado === "disponible";
  const accion = disponible && puedeRetirar
    ? `<button type="button" class="boton-secundario" data-meritos-retirar="${escaparMeritos(hecho.hecho_ref)}">${t(textos, "retirar_este")}</button>`
    : "";
  const recibo = hecho.retirada?.recibo_ref
    ? `<details><summary>${t(textos, "detalle_recibo")}</summary><p>${escaparMeritos(hecho.retirada.recibo_ref)}</p></details>`
    : "";
  return `<article class="meritos-reales-ficha"><div class="meritos-reales-ficha-contenido">
    <div class="meritos-reales-ficha-cabecera"><div><h3>${escaparMeritos(hecho.denominacion)}</h3>
      <p>${t(textos, `tipo_${hecho.tipo}`)}</p></div>
      <span class="meritos-reales-estado meritos-reales-estado--${hecho.disponibilidad.estado}">${t(textos, `disponibilidad_${hecho.disponibilidad.estado}`)}</span></div>
    <div class="meritos-reales-ficha-pie">${accion}${recibo}</div></div></article>`;
}

function filtros(modelo, textos) {
  return `<nav class="meritos-reales-filtros" aria-label="${t(textos, "filtros_titulo")}">
    ${["todos", "disponibles", "retirados"].map((filtro) => `<button type="button" data-meritos-filtro="${filtro}"
      aria-pressed="${modelo.filtro === filtro}">${t(textos, `filtro_${filtro}`)}</button>`).join("")}
  </nav>`;
}

function revisarRetirada(modelo, textos) {
  if (!modelo.seleccion) return "";
  const hecho = modelo.hechos.find((item) => item.hecho_ref === modelo.seleccion.hecho_ref
    && item.disponibilidad?.estado === "disponible");
  if (!hecho) return "";
  const motivos = Array.isArray(modelo.motivos) ? modelo.motivos : [];
  const opciones = motivos.map((m, indice) => `<option value="${indice}">${escaparMeritos(m.etiqueta)}</option>`).join("");
  return `<section class="meritos-reales-revision" aria-labelledby="meritos-retirada-titulo">
    <header><h3 id="meritos-retirada-titulo">${t(textos, "revision_titulo")}</h3></header>
    <div><p class="meritos-reales-seleccion">${escaparMeritos(hecho.denominacion)}</p>
      <p>${t(textos, "revision_efecto")}</p>
      ${motivos.length ? `<form id="meritos-reales-retirada-formulario"><label for="meritos-retirada-motivo">${t(textos, "motivo")}</label>
        <select id="meritos-retirada-motivo" name="motivo" required><option value="">${t(textos, "motivo_elegir")}</option>${opciones}</select>
        <div class="meritos-reales-acciones"><button type="button" class="boton-secundario" data-meritos-cancelar>${t(textos, "cancelar")}</button>
          <button type="submit" class="boton-peligro"${modelo.guardando ? " disabled" : ""}>${t(textos, modelo.guardando ? "guardando" : "confirmar_retirada")}</button></div>
      </form>` : `<p role="status">${t(textos, "motivos_no_disponibles")}</p><button type="button" data-meritos-cancelar class="boton-secundario">${t(textos, "cancelar")}</button>`}
    </div></section>`;
}

function confirmacion(modelo, textos) {
  if (!modelo.confirmacion?.recibo?.referencia) return "";
  const recibo = modelo.confirmacion.recibo;
  return `<section class="meritos-reales-confirmacion" role="status" tabindex="-1"><div>
    <strong>${t(textos, modelo.confirmacion.repetida ? "reintento_confirmado" : "retirada_confirmada")}</strong>
    <p>${t(textos, "recibo_numero", { numero: recibo.referencia })}</p>
    <button type="button" class="boton-secundario" data-meritos-consultar>${t(textos, "consultar_de_nuevo")}</button>
  </div></section>`;
}

function retiradaPendiente(modelo, textos) {
  if (!modelo.pendiente || modelo.guardando) return "";
  return `<section class="meritos-reales-pendiente" role="alert" tabindex="-1">
    <p>${t(textos, "retirada_pendiente")}</p>
    <button type="button" class="boton-secundario" data-meritos-filtro="todos">${t(textos, "consultar_todos")}</button>
  </section>`;
}

/** Un dato agrupado solo se muestra con el filtro de esa misma página. */
export function renderizarMeritosPropios(modelo, textos) {
  if (!modelo || !ESTADOS.has(modelo.estado) || !FILTROS.has(modelo.filtro) ||
    typeof textos?.traducir !== "function") throw new TypeError("meritos.cartera.modelo_invalido");
  const hechos = Array.isArray(modelo.hechos) ? modelo.hechos : [];
  const lista = modelo.estado === "listo" && hechos.length
    ? hechos.map((h) => ficha(h, textos, Array.isArray(modelo.motivos) && modelo.motivos.length > 0)).join("")
    : `<div class="meritos-reales-aviso" role="status" tabindex="-1">${t(textos, `estado_vista_${modelo.estado}`)}
      ${["error", "conflicto", "resultado_indeterminado"].includes(modelo.estado)
        ? `<button type="button" data-meritos-consultar class="boton-secundario">${t(textos, "consultar_de_nuevo")}</button>` : ""}</div>`;
  const total = modelo.estado === "listo" && Number.isSafeInteger(modelo.total) && modelo.total >= 0
    ? `<button type="button" data-meritos-filtro="${modelo.filtro}" class="meritos-reales-total">${t(textos, "total_filtro", { cuenta: modelo.total })}</button>`
    : "";
  const mas = modelo.estado === "listo" && modelo.cursor_siguiente
    ? `<button type="button" data-meritos-mas class="boton-secundario">${t(textos, "ver_mas")}</button>` : "";
  return `<section class="panel meritos-reales" aria-labelledby="meritos-cartera-titulo">
    <header><h2 id="meritos-cartera-titulo" tabindex="-1">${t(textos, "lista_titulo")}</h2></header>
    <div class="panel-contenido">${filtros(modelo, textos)}${retiradaPendiente(modelo, textos)}<div class="meritos-reales-lista">${lista}</div>
      <div class="meritos-reales-paginacion">${total}${mas}</div>${confirmacion(modelo, textos)}${revisarRetirada(modelo, textos)}</div>
  </section>`;
}
