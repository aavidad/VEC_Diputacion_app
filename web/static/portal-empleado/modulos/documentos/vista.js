import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { obtenerDatosDocumentosPresentacion } from "./datos-presentacion.js";
import { crearTraductorDocumentos } from "./i18n.js";

function nodo(documento, etiqueta, texto = "", clase = "") {
  const resultado = documento.createElement(etiqueta);
  if (texto) resultado.textContent = texto;
  if (clase) resultado.className = clase;
  return resultado;
}

function botonBloqueado(documento, texto, motivo) {
  const boton = nodo(documento, "button", texto, "documentos-accion");
  boton.type = "button";
  boton.disabled = true;
  boton.setAttribute("aria-disabled", "true");
  boton.title = motivo;
  return boton;
}

function tabla(documento, titulo, cabeceras, filas) {
  const region = nodo(documento, "div", "", "documentos-tabla");
  region.tabIndex = 0;
  region.setAttribute("role", "region");
  region.setAttribute("aria-label", titulo);
  const elemento = nodo(documento, "table");
  elemento.append(nodo(documento, "caption", titulo));
  const thead = nodo(documento, "thead");
  const cabecera = nodo(documento, "tr");
  cabeceras.forEach((texto) => { const th = nodo(documento, "th", texto); th.scope = "col"; cabecera.append(th); });
  thead.append(cabecera);
  const tbody = nodo(documento, "tbody");
  filas.forEach((fila) => tbody.append(fila));
  elemento.append(thead, tbody);
  region.append(elemento);
  return region;
}

function estadoClase(estado) { return `documentos-estado--${estado}`; }

/**
 * Superficie exclusivamente visual para RRHH. No realiza red, almacenamiento,
 * descarga, generación, firma, envío ni verificación externa.
 */
export function montarVistaDocumentos({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError(crearTraductorDocumentos()("error_vista"));
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError(crearTraductorDocumentos()("error_documento"));
  const t = crearTraductorDocumentos(); const datos = obtenerDatosDocumentosPresentacion(); const { atlas, documentos: DOCUMENTOS, plantillas: PLANTILLAS } = datos;
  let activa = true;
  let filtro = "";
  let seleccion = DOCUMENTOS[0];
  const contenedor = nodo(documento, "section", "", "modulo-documentos");
  contenedor.dataset.documentos = "";
  raiz.append(contenedor);
  const desmontar = () => { if (!activa) return; activa = false; contenedor.remove(); };
  registrarDesmontar?.(desmontar);

  const cabecera = nodo(documento, "header", "", "documentos-cabecera");
  cabecera.append(
    nodo(documento, "p", t("sobrelinea"), "sobrelinea"), nodo(documento, "h2", t("titulo")), nodo(documento, "p", t("descripcion")), nodo(documento, "p", datos.aviso, "documentos-aviso"),
  );
  const identidad = nodo(documento, "section", "", "documentos-identidad");
  identidad.setAttribute("aria-label", t("contexto"));
  identidad.append(nodo(documento, "strong", `${atlas.tecnica_rrhh.nombre_visible} · ${t("rol_rrhh")}`), nodo(documento, "span", `${atlas.unidad.nombre_visible} · ${atlas.centro.nombre_visible}`), nodo(documento, "small", t("identidad_aviso")));
  const filtros = nodo(documento, "form", "", "documentos-filtros");
  const etiquetaFiltro = nodo(documento, "label", t("filtro"));
  const entradaFiltro = nodo(documento, "input"); entradaFiltro.type = "search"; entradaFiltro.name = "filtro"; entradaFiltro.placeholder = t("filtro_placeholder");
  etiquetaFiltro.append(entradaFiltro);
  const aplicar = nodo(documento, "button", t("aplicar_filtro"), "boton-primario"); aplicar.type = "submit";
  filtros.append(etiquetaFiltro, aplicar);
  const indicadores = nodo(documento, "div", "", "documentos-indicadores");
  const principal = nodo(documento, "div", "", "documentos-principal");
  const listado = nodo(documento, "section", "", "panel documentos-listado");
  const ficha = nodo(documento, "aside", "", "panel documentos-ficha");
  const inferiores = nodo(documento, "div", "", "documentos-inferiores");
  const entrega = nodo(documento, "div", "", "documentos-entrega");
  // El componente común escapa todos los textos de este contrato cerrado.
  entrega.innerHTML = renderizarEstadoEntrega({ estado: "visual_pendiente_backend", resumen: t("entrega_resumen"), pendientes: [t("entrega_repositorio"), t("entrega_autorizacion"), t("entrega_huella"), t("entrega_firma"), t("entrega_auditoria")], fuente: { etiqueta: datos.aviso }, conexion: t("entrega_conexion") });

  function pintarIndicadores() {
    const resumen = [[t("resumen_documentos"), String(DOCUMENTOS.length), t("resumen_documentos_nota")], [t("resumen_firma"), t("resumen_firma_valor"), t("resumen_firma_nota")], [t("resumen_plantillas"), String(PLANTILLAS.length), t("resumen_plantillas_nota")], [t("resumen_trazabilidad"), t("resumen_trazabilidad_valor"), t("resumen_trazabilidad_nota")]];
    indicadores.replaceChildren(...resumen.map(([etiqueta, valor, nota]) => { const tarjeta = nodo(documento, "article", "", "documentos-indicador"); tarjeta.append(nodo(documento, "span", etiqueta), nodo(documento, "strong", valor), nodo(documento, "small", nota)); return tarjeta; }));
  }
  function pintarListado() {
    const termino = filtro.toLocaleLowerCase("es");
    const visibles = DOCUMENTOS.filter((item) => !termino || `${item.titulo} ${item.tipo} ${item.expediente}`.toLocaleLowerCase("es").includes(termino));
    const filas = visibles.map((item) => { const tr = nodo(documento, "tr"); if (item.id === seleccion.id) tr.dataset.seleccionada = "true"; [item.titulo, item.tipo, item.expediente, item.version, item.fecha].forEach((valor) => tr.append(nodo(documento, "td", valor))); const estado = nodo(documento, "td"); estado.append(nodo(documento, "span", item.estado, `documentos-estado ${estadoClase(item.estado_clave)}`)); tr.append(estado); const acciones = nodo(documento, "td"); const ver = nodo(documento, "button", t("ver_ficha"), "documentos-ver"); ver.type = "button"; ver.addEventListener("click", () => { seleccion = item; pintar(); anunciar(t("ficha_seleccionada", { titulo: item.titulo }), "informacion"); }); acciones.append(ver); tr.append(acciones); return tr; });
    if (!filas.length) { const tr = nodo(documento, "tr"); const td = nodo(documento, "td", t("sin_resultados")); td.colSpan = 7; tr.append(td); filas.push(tr); }
    listado.replaceChildren(nodo(documento, "h3", t("biblioteca")), nodo(documento, "p", t("biblioteca_descripcion")), tabla(documento, t("tabla_documentos"), [t("col_documento"), t("col_tipo"), t("col_expediente"), t("col_version"), t("col_fecha"), t("col_estado"), t("col_accion")], filas));
  }
  function linea(documento, etiqueta, valor) { const fila = nodo(documento, "div"); fila.append(nodo(documento, "dt", etiqueta), nodo(documento, "dd", valor)); return fila; }
  function pintarFicha() {
    const metadatos = nodo(documento, "dl", "", "documentos-metadatos");
    [[t("col_expediente"), seleccion.expediente], [t("col_version"), seleccion.version], [t("responsable"), seleccion.responsable], [t("huella"), seleccion.huella], [t("conservacion"), seleccion.conservacion], [t("circuito"), seleccion.circuito]].forEach(([a, b]) => metadatos.append(linea(documento, a, b)));
    const acciones = nodo(documento, "div", "", "documentos-acciones");
    [["generar_version", "generar_motivo"], ["subir_original", "subir_motivo"], ["firmar", "firmar_motivo"], ["verificar", "verificar_motivo"], ["enviar", "enviar_motivo"], ["descargar", "descargar_motivo"]].forEach(([texto, motivo]) => acciones.append(botonBloqueado(documento, t(texto), t(motivo))));
    ficha.replaceChildren(nodo(documento, "h3", t("ficha")), nodo(documento, "p", seleccion.titulo, "documentos-titulo-ficha"), metadatos, nodo(documento, "h4", t("acciones_pendientes")), acciones, nodo(documento, "p", t("aclaracion_firma"), "documentos-aclaracion"));
  }
  function pintarInferiores() {
    const plantillas = nodo(documento, "section", "", "panel");
    const filasPlantillas = PLANTILLAS.map(([nombre, estado]) => { const tr = nodo(documento, "tr"); tr.append(nodo(documento, "td", nombre), nodo(documento, "td", estado)); const td = nodo(documento, "td"); td.append(botonBloqueado(documento, t("generar"), t("generar_plantilla_motivo"))); tr.append(td); return tr; });
    plantillas.append(nodo(documento, "h3", t("plantillas")), tabla(documento, t("tabla_plantillas"), [t("col_plantilla"), t("col_estado"), t("col_accion")], filasPlantillas));
    const circuito = nodo(documento, "section", "", "panel documentos-circuito");
    circuito.append(nodo(documento, "h3", t("circuito_titulo")), nodo(documento, "ol", "", "documentos-pasos"));
    const lista = circuito.querySelector("ol"); ["paso_1", "paso_2", "paso_3", "paso_4", "paso_5"].forEach((clave, indice) => { const item = nodo(documento, "li"); item.append(nodo(documento, "strong", `${indice + 1}. `), nodo(documento, "span", t(clave)), nodo(documento, "small", t("paso_pendiente"))); lista.append(item); });
    inferiores.replaceChildren(plantillas, circuito);
  }
  function pintar() { if (!activa) return; pintarIndicadores(); pintarListado(); pintarFicha(); pintarInferiores(); }
  filtros.addEventListener("submit", (evento) => { evento.preventDefault(); filtro = entradaFiltro.value.trim(); pintar(); anunciar(filtro ? t("filtro_aplicado", { filtro }) : t("filtro_eliminado"), "informacion"); });
  contenedor.append(cabecera, identidad, filtros, indicadores, principal, inferiores, entrega); principal.append(listado, ficha); pintar();
  return Object.freeze({ desmontar });
}
