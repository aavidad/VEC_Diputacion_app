import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { crearTraductorAprobaciones } from "./i18n.js";

function crear(documento, etiqueta, texto = "", clase = "") {
  const nodo = documento.createElement(etiqueta);
  if (texto) nodo.textContent = texto;
  if (clase) nodo.className = clase;
  return nodo;
}

function cabeceraPanel(documento, titulo, ayuda) {
  const cabecera = crear(documento, "header", "", "cabecera-panel aprobaciones-panel-cabecera");
  cabecera.append(crear(documento, "h3", titulo));
  if (ayuda) {
    const desplegable = crear(documento, "details", "", "aprobaciones-ayuda");
    const resumen = crear(documento, "summary", "?");
    resumen.setAttribute("aria-label", `${titulo}: ${ayuda}`);
    desplegable.append(resumen, crear(documento, "p", ayuda));
    cabecera.append(desplegable);
  }
  return cabecera;
}

function botonInactivo(documento, texto, motivo) {
  const bloque = crear(documento, "div", "", "aprobaciones-accion-bloqueada");
  const boton = crear(documento, "button", texto, "aprobaciones-accion");
  boton.type = "button";
  boton.disabled = true;
  boton.setAttribute("aria-disabled", "true");
  bloque.append(boton, crear(documento, "small", motivo));
  return bloque;
}

const ESTADOS = Object.freeze(["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]);

/**
 * Superficie de Aprobaciones sin fuente de pendientes conectada. El integrador
 * aporta la raíz y, en el futuro, el estado verificado por su conector.
 * No se crean decisiones, recibos, firmas, envíos ni datos de muestra.
 */
export function montarVistaAprobaciones({ raiz, anunciar = () => {}, registrarDesmontar, estadoVista = "no_configurado" } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de Aprobaciones no disponible");
  }
  if (!ESTADOS.includes(estadoVista)) throw new TypeError("estado de Aprobaciones no disponible");
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Aprobaciones no disponible");
  const t = crearTraductorAprobaciones();
  let activa = true;

  const contenedor = crear(documento, "section", "", "modulo-aprobaciones");
  contenedor.dataset.aprobaciones = "";
  contenedor.dataset.estadoEntrega = "visual_pendiente_backend";
  contenedor.dataset.estadoVista = estadoVista;
  const cabecera = crear(documento, "header", "", "aprobaciones-cabecera");
  const introduccion = crear(documento, "div", "", "aprobaciones-intro");
  introduccion.append(crear(documento, "p", t("sobrelinea"), "sobrelinea"), crear(documento, "h2", t("titulo")), crear(documento, "p", t("descripcion")));
  cabecera.append(introduccion, crear(documento, "span", t(`estado_${estadoVista}`), "estado-chip neutro"));

  const filtros = crear(documento, "form", "", "panel aprobaciones-filtros");
  const filtrosCuerpo = crear(documento, "div", "", "cuerpo-panel aprobaciones-filtros-cuerpo");
  const circuito = crear(documento, "label", t("circuito"));
  const selector = crear(documento, "select");
  const todos = crear(documento, "option", t("todos_circuitos"));
  todos.value = "todos";
  selector.append(todos);
  selector.disabled = true;
  circuito.append(selector);
  const busqueda = crear(documento, "label", t("buscar"));
  const campo = crear(documento, "input");
  campo.type = "search";
  campo.placeholder = t("buscar_placeholder");
  campo.disabled = true;
  busqueda.append(campo);
  const aplicar = crear(documento, "button", t("aplicar"), "boton-primario aprobaciones-filtrar");
  aplicar.type = "submit";
  aplicar.disabled = true;
  filtrosCuerpo.append(circuito, busqueda, aplicar);
  filtros.append(cabeceraPanel(documento, t("filtros"), t("ayuda_filtros")), filtrosCuerpo);
  filtros.addEventListener("submit", (evento) => evento.preventDefault());

  const indicadores = crear(documento, "div", "", "rejilla-kpi aprobaciones-indicadores");
  const resumen = [
    ["▤", t("pendientes_visibles"), "0", t("sin_fuente")],
    ["!", t("alta_prioridad"), "0", t("sin_plazos")],
    ["◫", t("circuitos"), "0", t("sin_fuente")],
    ["◇", t("firma_multiple"), t("firma_no_verificada"), t("portafirmas_pendiente")],
  ];
  indicadores.append(...resumen.map(([icono, nombre, valor, nota], indice) => {
    const tarjeta = crear(documento, "article", "", `tarjeta-kpi aprobaciones-kpi aprobaciones-kpi--${indice}`);
    const simbolo = crear(documento, "span", icono, "icono-kpi");
    simbolo.setAttribute("aria-hidden", "true");
    tarjeta.append(simbolo, crear(documento, "span", nombre), crear(documento, "strong", valor, "valor-kpi"), crear(documento, "small", nota));
    return tarjeta;
  }));

  const principal = crear(documento, "div", "", "aprobaciones-principal");
  const bandeja = crear(documento, "section", "", "panel aprobaciones-bandeja");
  const bandejaCuerpo = crear(documento, "div", "", "cuerpo-panel");
  const mensaje = estadoVista === "disponible" ? "mensaje_vacio" : `mensaje_${estadoVista}`;
  bandejaCuerpo.append(crear(documento, "p", t(mensaje), "aprobaciones-vacio"));
  bandeja.replaceChildren(cabeceraPanel(documento, t("bandeja"), t("ayuda_bandeja")), bandejaCuerpo);

  const detalle = crear(documento, "section", "", "panel aprobaciones-detalle");
  const detalleCuerpo = crear(documento, "div", "", "cuerpo-panel aprobaciones-detalle-cuerpo");
  detalleCuerpo.append(crear(documento, "p", t("detalle_vacio"), "aprobaciones-vacio"), crear(documento, "p", t("separacion_funciones"), "aprobaciones-separacion"), crear(documento, "h4", t("acciones_pendientes")));
  const acciones = crear(documento, "div", "", "aprobaciones-acciones");
  [["aprobar", "motivo_aprobar"], ["devolver", "motivo_devolver"], ["rechazar", "motivo_rechazar"], ["firmar", "motivo_firmar"], ["remitir", "motivo_remitir"], ["descargar", "motivo_descargar"], ["delegar", "motivo_delegar"]].forEach(([accion, motivo]) => acciones.append(botonInactivo(documento, t(accion), t(motivo))));
  detalleCuerpo.append(acciones);
  detalle.replaceChildren(cabeceraPanel(documento, t("detalle"), t("ayuda_detalle")), detalleCuerpo);
  principal.append(bandeja, detalle);

  const suplencias = crear(documento, "section", "", "panel aprobaciones-suplencias");
  const suplenciasCuerpo = crear(documento, "div", "", "cuerpo-panel");
  suplenciasCuerpo.append(crear(documento, "p", t("suplencias_sin_fuente"), "aprobaciones-vacio"));
  suplencias.replaceChildren(cabeceraPanel(documento, t("suplencias"), t("suplencias_ayuda")), suplenciasCuerpo);

  const pie = crear(documento, "div", "", "aprobaciones-limite");
  pie.innerHTML = renderizarEstadoEntrega({
    estado: "visual_pendiente_backend",
    resumen: t("resumen_entrega"),
    pendientes: ["pendiente_identidad", "pendiente_matriz", "pendiente_firma", "pendiente_operacion", "pendiente_origen"].map(t),
    fuente: { etiqueta: t("sin_fuente") },
    conexion: t("conexion"),
  });
  contenedor.append(cabecera, crear(documento, "p", t("aviso_sin_fuente"), "aprobaciones-aviso"), filtros, indicadores, principal, suplencias, pie);
  raiz.append(contenedor);
  const desmontar = () => { if (!activa) return; activa = false; contenedor.remove(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
