import { crearTraductorMeritos } from "./i18n.js";

const PESTANAS = Object.freeze(["inventario", "requisitos", "valoraciones"]);
const ESTADOS = new Set(["no_configurado", "cargando", "disponible", "vacio", "denegado", "error"]);
const ESTADOS_HECHO = new Set(["declarado", "pendiente", "acreditado", "rechazado"]);
const RESULTADOS = new Set(["cumple", "no_cumple", "pendiente"]);
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const texto = (valor, reserva) => escapar(typeof valor === "string" && valor.trim() ? valor : reserva);
const lista = (valor) => Array.isArray(valor) ? valor.filter((item) => item && typeof item === "object" && !Array.isArray(item)) : [];

/** Proyección de solo lectura. El coordinador debe obtenerla de una fuente autorizada. */
function normalizar(datos = {}) {
  if (!datos || typeof datos !== "object" || Array.isArray(datos) || !ESTADOS.has(datos.estado ?? "no_configurado")) throw new TypeError("estado de Méritos no disponible");
  const estado = datos.estado ?? "no_configurado";
  if (estado !== "disponible") return { estado, meritos: [], requisitos: [], valoraciones: [] };
  for (const clave of ["meritos", "requisitos", "valoraciones"]) {
    if (datos[clave] !== undefined && !Array.isArray(datos[clave])) throw new TypeError(`lista de Méritos inválida: ${clave}`);
  }
  return { estado, meritos: lista(datos.meritos), requisitos: lista(datos.requisitos), valoraciones: lista(datos.valoraciones) };
}

function pastilla(codigo, permitidos, t) {
  const seguro = permitidos.has(codigo) ? codigo : "sin_dato";
  return `<span class="meritos-estado meritos-estado--${seguro}">${escapar(t(seguro))}</span>`;
}

function panel(titulo, subtitulo, cuerpo) {
  return `<section class="panel meritos-panel"><header class="cabecera-panel"><div><h3>${escapar(titulo)}</h3><p>${escapar(subtitulo)}</p></div></header><div class="cuerpo-panel">${cuerpo}</div></section>`;
}

function tabla(titulo, columnas, filas) {
  return `<div class="meritos-tabla-wrap" role="region" tabindex="0" aria-label="${escapar(titulo)}"><table class="meritos-tabla"><caption class="meritos-sr">${escapar(titulo)}</caption><thead><tr>${columnas.map((nombre) => `<th scope="col">${escapar(nombre)}</th>`).join("")}</tr></thead><tbody>${filas.join("")}</tbody></table></div>`;
}

function inventario(datos, filtro, seleccionado, t) {
  const visibles = datos.meritos.map((item, indice) => ({ item, indice })).filter(({ item }) => filtro === "todos" || item.estado === filtro);
  const controles = `<div class="meritos-controles"><label>${t("filtrar")} <select data-meritos-filtro>${["todos", ...ESTADOS_HECHO].map((clave) => `<option value="${clave}"${filtro === clave ? " selected" : ""}>${escapar(t(clave))}</option>`).join("")}</select></label><span class="meritos-cuenta" role="status">${escapar(t("registros", { numero: new Intl.NumberFormat("es-ES").format(visibles.length) }))}</span></div>`;
  const filas = visibles.map(({ item, indice }, posicion) => {
    const nombre = typeof item.nombre === "string" && item.nombre.trim() ? item.nombre : t("sin_dato");
    const abierto = seleccionado === indice;
    const codigoEstado = ESTADOS_HECHO.has(item.estado) ? item.estado : "sin_dato";
    const fila = `<tr class="meritos-fila${posicion % 2 ? " meritos-fila--alterna" : ""}" data-estado="${codigoEstado}"><th scope="row"><button type="button" class="meritos-detalle-boton" data-meritos-detalle="${indice}" aria-expanded="${abierto}" aria-label="${escapar(t(abierto ? "cerrar_detalle" : "abrir_detalle", { nombre }))}">${escapar(nombre)}</button></th><td>${texto(item.tipo, t("sin_dato"))}</td><td>${pastilla(item.estado, ESTADOS_HECHO, t)}</td></tr>`;
    if (!abierto) return fila;
    return `${fila}<tr class="meritos-fila-detalle" data-estado="${codigoEstado}"><td colspan="3"><section aria-label="${escapar(t("detalle"))}"><dl class="meritos-detalle-datos"><div><dt>${escapar(t("fuente"))}</dt><dd>${texto(item.fuente, t("aviso_sin_fuente"))}</dd></div><div><dt>${escapar(t("evidencia"))}</dt><dd>${texto(item.evidencia, t("aviso_sin_evidencia"))}</dd></div><div><dt>${escapar(t("vigencia"))}</dt><dd>${texto(item.vigencia, t("aviso_sin_vigencia"))}</dd></div></dl></section></td></tr>`;
  });
  return panel(t("inventario"), t("inventario_ayuda"), `${controles}${visibles.length ? tabla(t("inventario"), ["nombre", "tipo", "estado"].map(t), filas) : `<p class="meritos-vacio">${escapar(t(datos.meritos.length ? "sin_resultados" : "vacio_texto"))}</p>`}${accion(t)}`);
}

function requisitos(datos, t) {
  const filas = datos.requisitos.map((item) => `<tr><th scope="row">${texto(item.requisito, t("sin_dato"))}<small>${escapar(t("convocatoria"))}: ${texto(item.convocatoria, t("sin_dato"))}</small></th><td>${texto(item.versionBases, t("aviso_sin_version"))}</td><td>${pastilla(item.resultado, RESULTADOS, t)}</td><td>${texto(item.motivo, t("sin_dato"))}</td><td>${texto(item.procedencia, t("sin_dato"))}<small>${escapar(t("hito"))}: ${texto(item.hito, t("sin_dato"))}</small></td></tr>`);
  return panel(t("requisitos"), t("requisitos_ayuda"), filas.length ? tabla(t("requisitos"), ["requisito", "bases", "resultado", "motivo", "procedencia"].map(t), filas) : `<p class="meritos-vacio">${escapar(t("sin_requisitos"))}</p>`);
}

function valoraciones(datos, t) {
  const filas = datos.valoraciones.map((item) => `<tr><th scope="row">${texto(item.merito, t("sin_dato"))}<small>${escapar(t("convocatoria"))}: ${texto(item.convocatoria, t("sin_dato"))}</small></th><td>${texto(item.versionBases, t("aviso_sin_version"))}</td><td>${texto(item.criterio, t("sin_dato"))}</td><td class="meritos-numero">${Number.isFinite(item.puntos) ? escapar(new Intl.NumberFormat("es-ES", { maximumFractionDigits: 2 }).format(item.puntos)) : escapar(t("sin_dato"))}</td></tr>`);
  return panel(t("valoraciones"), t("valoraciones_ayuda"), filas.length ? tabla(t("valoraciones"), ["merito", "bases", "criterio", "puntos"].map(t), filas) : `<p class="meritos-vacio">${escapar(t("sin_valoraciones"))}</p>`);
}

function accion(t) {
  return `<div class="meritos-accion"><button type="button" disabled aria-disabled="true" title="${escapar(t("aportar_pendiente"))}">${escapar(t("aportar"))}</button><small>${escapar(t("aportar_pendiente"))}</small></div>`;
}

function resumen(datos, t) {
  const metricas = [
    [t("total"), datos.meritos.length, "total", "≡"],
    [t("acreditados"), datos.meritos.filter((x) => x.estado === "acreditado").length, "acreditado", "✓"],
    [t("pendientes"), datos.meritos.filter((x) => x.estado === "pendiente" || x.estado === "declarado").length, "pendiente", "…"],
    [t("rechazados"), datos.meritos.filter((x) => x.estado === "rechazado").length, "rechazado", "!"],
  ];
  return `<section class="rejilla-kpi meritos-kpis" aria-label="${escapar(t("inventario"))}">${metricas.map(([etiqueta, valor, clase, icono]) => `<article class="tarjeta-kpi meritos-kpi meritos-kpi--${clase}"><span class="icono-kpi" aria-hidden="true">${icono}</span><div><strong class="valor-kpi">${new Intl.NumberFormat("es-ES").format(valor)}</strong><span class="etiqueta-kpi">${escapar(etiqueta)}</span></div></article>`).join("")}</section>`;
}

/** Render puro para estado y datos ya autorizados; no consulta ni valora méritos. */
export function renderizarMeritos(entrada = {}, { pestana = "inventario", filtro = "todos", seleccionado = -1 } = {}) {
  const datos = normalizar(entrada);
  const t = crearTraductorMeritos();
  const actual = PESTANAS.includes(pestana) ? pestana : "inventario";
  const filtroSeguro = filtro === "todos" || ESTADOS_HECHO.has(filtro) ? filtro : "todos";
  const aviso = datos.estado === "disponible" ? "" : `<section class="panel meritos-aviso meritos-aviso--${datos.estado}" role="status"><div class="cuerpo-panel"><strong>${escapar(t(datos.estado))}</strong><p>${escapar(t(`${datos.estado}_texto`))}</p></div></section>`;
  const contenido = datos.estado === "disponible" ? { inventario: () => inventario(datos, filtroSeguro, seleccionado, t), requisitos: () => requisitos(datos, t), valoraciones: () => valoraciones(datos, t) }[actual]() : aviso;
  return `<header class="meritos-cabecera"><div><p class="meritos-sobrelinea">${escapar(t("sobrelinea"))}</p><h2>${escapar(t("titulo"))}</h2><p>${escapar(t("descripcion"))}</p></div><details class="meritos-ayuda"><summary aria-label="${escapar(t("ayuda"))}" title="${escapar(t("ayuda"))}">?</summary><p>${escapar(t("ayuda_texto"))}</p></details></header><div class="meritos-situacion"><span class="meritos-estado meritos-estado--${datos.estado === "disponible" ? "acreditado" : "pendiente"}">${escapar(t(datos.estado))}</span><span>${escapar(t(datos.estado === "disponible" ? "solo_lectura" : "consulta_pendiente"))}</span></div>${datos.estado === "disponible" ? resumen(datos, t) : ""}<nav class="meritos-tabs" role="tablist" aria-label="${escapar(t("navegacion"))}">${PESTANAS.map((clave) => `<button type="button" role="tab" data-meritos-vista="${clave}" aria-selected="${clave === actual}" tabindex="${clave === actual ? "0" : "-1"}">${escapar(t(clave))}</button>`).join("")}</nav><div class="meritos-contenido" role="tabpanel" aria-label="${escapar(t(actual))}">${contenido}</div>`;
}

/** Montaje local sin red. actualizar() acepta una nueva proyección autorizada. */
export function montarVistaMeritos({ raiz, datos, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de Méritos no disponible");
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Méritos no disponible");
  let proyeccion = normalizar(datos);
  const t = crearTraductorMeritos();
  const contenedor = documento.createElement("section"); contenedor.className = "modulo-meritos"; contenedor.dataset.moduloMeritos = ""; raiz.append(contenedor);
  let activa = true; let pestana = "inventario"; let filtro = "todos"; let seleccionado = -1;
  const pintar = () => { if (activa && contenedor.parentElement === raiz) contenedor.innerHTML = renderizarMeritos(proyeccion, { pestana, filtro, seleccionado }); };
  pintar(); anunciar(t("anuncio_inicial"), "informacion");
  const alClick = (evento) => {
    const boton = evento.target?.closest?.("[data-meritos-vista]");
    if (boton && contenedor.contains?.(boton) && PESTANAS.includes(boton.dataset.meritosVista)) {
      pestana = boton.dataset.meritosVista; filtro = "todos"; seleccionado = -1; pintar();
      contenedor.querySelector?.(`[data-meritos-vista="${pestana}"]`)?.focus?.();
      anunciar(t("anuncio_seccion", { nombre: t(pestana) }), "informacion");
      return;
    }
    const detalle = evento.target?.closest?.("[data-meritos-detalle]");
    if (!detalle || !contenedor.contains?.(detalle)) return;
    const indice = Number(detalle.dataset.meritosDetalle);
    if (!Number.isInteger(indice) || indice < 0 || indice >= proyeccion.meritos.length || (filtro !== "todos" && proyeccion.meritos[indice].estado !== filtro)) return;
    seleccionado = seleccionado === indice ? -1 : indice; pintar();
    contenedor.querySelector?.(`[data-meritos-detalle="${indice}"]`)?.focus?.();
  };
  const alCambio = (evento) => { if (!evento.target?.matches?.("[data-meritos-filtro]")) return; filtro = evento.target.value; seleccionado = -1; pintar(); contenedor.querySelector?.("[data-meritos-filtro]")?.focus?.(); };
  const alTecla = (evento) => { const boton = evento.target?.closest?.("[data-meritos-vista]"); if (!boton || !contenedor.contains?.(boton) || !["ArrowRight", "ArrowLeft", "Home", "End"].includes(evento.key)) return; const indice = PESTANAS.indexOf(boton.dataset.meritosVista); if (indice < 0) return; evento.preventDefault(); const salto = evento.key === "Home" ? -indice : evento.key === "End" ? PESTANAS.length - indice - 1 : evento.key === "ArrowRight" ? 1 : -1; pestana = PESTANAS[(indice + salto + PESTANAS.length) % PESTANAS.length]; filtro = "todos"; pintar(); contenedor.querySelector?.(`[data-meritos-vista="${pestana}"]`)?.focus?.(); };
  contenedor.addEventListener("click", alClick); contenedor.addEventListener("change", alCambio); contenedor.addEventListener("keydown", alTecla);
  const actualizar = (siguiente) => { if (!activa) return; proyeccion = normalizar(siguiente); filtro = "todos"; seleccionado = -1; pintar(); anunciar(t("anuncio_actualizado"), "informacion"); };
  const desmontar = () => { if (!activa) return; activa = false; contenedor.removeEventListener("click", alClick); contenedor.removeEventListener("change", alCambio); contenedor.removeEventListener("keydown", alTecla); contenedor.remove(); };
  registrarDesmontar?.(desmontar); return Object.freeze({ actualizar, desmontar });
}
