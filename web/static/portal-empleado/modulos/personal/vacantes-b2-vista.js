import { cargarTextos } from "../../../comun/textos.js";
import { proyectarPaginaVacantesB2 } from "./vacantes-b2-proyeccion.js?v=20261002-b-base-401-acumulada-v3";

const catalogo = await cargarTextos("personal-vacantes");
const traducirPorDefecto = (codigoMensaje, variables) => catalogo.traducir(`general.${codigoMensaje}`, variables);
const ESTADOS = new Set(["disponible", "cargando", "error", "denegado", "cobertura_no_acreditada"]);
function nodo(d, tipo, texto) { const elemento = d.createElement(tipo); if (texto !== undefined) elemento.textContent = texto; return elemento; }
function mensaje(d, texto, alerta = false) {
  const elemento = nodo(d, "p", texto); elemento.className = "personal-registro-b2-estado";
  elemento.setAttribute("role", alerta ? "alert" : "status"); return elemento;
}
function listaDatos(d, entradas, formatos, agregar, t) {
  const lista = nodo(d, "dl");
  for (const entrada of entradas) {
    const valor = entrada.valor === "" ? t("sin_fin") : entrada.tipo === "texto" ? entrada.valor : formatos[entrada.tipo](entrada.valor);
    agregar(lista, t(entrada.codigoMensaje), valor);
  }
  return lista;
}
function origen(d, fila, numeroFila, formatos, agregar, t) {
  const detalles = nodo(d, "details"); detalles.className = "personal-registro-b2-traza";
  const abrir = nodo(d, "summary", t("ver_origen"));
  abrir.setAttribute("aria-label", t("origen_fila", { codigo: fila.codigoPlaza || t("sin_codigo"), fila: formatos.numero(numeroFila) }));
  detalles.append(abrir, listaDatos(d, fila.origenVisible, formatos, agregar, t));
  const contexto = nodo(d, "dl");
  agregar(contexto, t("fuente"), t("sin_denominacion_origen")); agregar(contexto, t("acto"), t("sin_denominacion_origen"));
  detalles.append(contexto, nodo(d, "p", t("garantia_origen")));
  const tecnico = nodo(d, "details"); tecnico.className = "personal-registro-b2-traza-tecnica";
  tecnico.append(nodo(d, "summary", t("detalle_tecnico")), listaDatos(d, fila.origenTecnico, formatos, agregar, t));
  detalles.append(tecnico); return detalles;
}
function celdaEstado(d, codigoMensaje, t, tono = "info", ayuda) {
  const celda = nodo(d, "td"); const estado = nodo(d, "span", t(codigoMensaje)); estado.className = `estado-chip ${tono}`;
  celda.append(estado);
  if (ayuda) { const detalle = nodo(d, "small", t(ayuda)); detalle.className = "personal-registro-b2-secundario"; celda.append(detalle); }
  return celda;
}
function tabla(d, modelo, formatos, agregar, t, filas = modelo.filas) {
  const region = nodo(d, "div"); region.className = "tabla-contenedor";
  region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("titulo"));
  const tabla = nodo(d, "table"); tabla.className = "tabla-datos"; tabla.append(nodo(d, "caption", t("titulo")));
  const cabecera = nodo(d, "thead"); const filaCabecera = nodo(d, "tr");
  for (const codigoMensaje of ["plaza", "unidad", "dotacion", "puesto", "necesidad", "origen"]) {
    const celda = nodo(d, "th", t(codigoMensaje)); celda.setAttribute("scope", "col"); filaCabecera.append(celda);
  }
  cabecera.append(filaCabecera); const cuerpo = nodo(d, "tbody");
  for (const fila of filas) {
    const indice = modelo.filas.indexOf(fila);
    const registro = nodo(d, "tr"); registro.dataset.personalVacante = "";
    const plaza = nodo(d, "th", fila.codigoPlaza || t("sin_codigo")); plaza.setAttribute("scope", "row");
    const puesto = celdaEstado(d, fila.puestoMensaje, t, "neutro", "ocupacion_no_determinada");
    if (fila.puesto) { const nombre = nodo(d, "strong", fila.puesto); nombre.className = "personal-registro-b2-secundario"; puesto.append(nombre); }
    const celdaOrigen = nodo(d, "td"); celdaOrigen.append(origen(d, fila, indice + 1, formatos, agregar, t));
    registro.append(plaza, nodo(d, "td", fila.unidad || t("sin_denominacion")), celdaEstado(d, fila.dotacionMensaje, t), puesto,
      celdaEstado(d, fila.necesidadMensaje, t, "", "criterios_pendientes"), celdaOrigen);
    cuerpo.append(registro);
  }
  tabla.append(cabecera, cuerpo); region.append(tabla); return region;
}

/** Hoja sin transporte ni caché: el montaje B2 conserva autorización, cancelación
 * y paginación. Inyecta sus formatoFecha/formatoInstante/datoTraza existentes.
 * Un estado de fallo ignora la página recibida y nunca presenta filas anteriores.
 */
export function crearVistaVacantesB2({ documento: d, pagina, estado = "disponible", formatos, anadirDatoTraza, traducir = traducirPorDefecto } = {}) {
  if (!d?.createElement || typeof traducir !== "function" || !ESTADOS.has(estado)) throw new TypeError("vista de vacantes no disponible");
  const panel = nodo(d, "section"); panel.className = "panel"; panel.dataset.personalVacantesB2 = "";
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-panel"; cabecera.append(nodo(d, "h3", traducir("titulo")));
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel"; panel.append(cabecera, cuerpo);
  if (estado !== "disponible") { cuerpo.append(mensaje(d, traducir(estado), ["error", "denegado", "cobertura_no_acreditada"].includes(estado))); return panel; }
  if (!formatos || !["fecha", "instante", "numero"].every((tipo) => typeof formatos[tipo] === "function") || typeof anadirDatoTraza !== "function") throw new TypeError("formato de vacantes no disponible");
  const modelo = proyectarPaginaVacantesB2(pagina);
  const recuento = nodo(d, "span"); recuento.setAttribute("role", "status");
  recuento.setAttribute("aria-live", "polite"); recuento.setAttribute("aria-atomic", "true"); cabecera.append(recuento);
  const ayuda = nodo(d, "details"); ayuda.className = "personal-registro-b2-ayuda";
  const abrirAyuda = nodo(d, "summary", "?"); abrirAyuda.setAttribute("aria-label", traducir("ayuda_abrir"));
  ayuda.append(abrirAyuda, nodo(d, "p", traducir("ayuda"))); cabecera.append(ayuda);
  cuerpo.append(nodo(d, "p", traducir("alcance")), nodo(d, "p", traducir("limite")));
  const alcance = nodo(d, "details"); alcance.className = "personal-registro-b2-traza";
  alcance.open = true; alcance.append(nodo(d, "summary", traducir("origen")));
  alcance.append(listaDatos(d, [
    { codigoMensaje: "fecha_efectos_corte", valor: modelo.corte.vigenteEn, tipo: "fecha" },
    { codigoMensaje: "conocido_hasta", valor: modelo.corte.conocidoEn, tipo: "instante" },
  ], formatos, anadirDatoTraza, traducir));
  const tecnico = nodo(d, "details"); tecnico.className = "personal-registro-b2-traza-tecnica";
  tecnico.append(nodo(d, "summary", traducir("detalle_tecnico")), listaDatos(d, [{ codigoMensaje: "organismo_ref", valor: modelo.ambitoRef, tipo: "texto" }], formatos, anadirDatoTraza, traducir));
  alcance.append(tecnico);
  const busqueda = nodo(d, "div"); busqueda.className = "personal-registro-b2-toolbar";
  const etiqueta = nodo(d, "label", traducir("buscar_pagina"));
  const campo = nodo(d, "input"); campo.type = "search"; campo.value = ""; campo.autocomplete = "off"; campo.spellcheck = false;
  campo.disabled = modelo.filas.length === 0; etiqueta.append(campo);
  const limpiar = nodo(d, "button", traducir("limpiar_busqueda")); limpiar.type = "button"; limpiar.disabled = true;
  busqueda.append(etiqueta, limpiar);
  const resultado = nodo(d, "div");
  function filtrar() {
    const consulta = campo.value.trim().normalize("NFC").toLowerCase();
    const filas = modelo.filas.filter((fila) => [fila.codigoPlaza, fila.unidad, fila.puesto]
      .some((valor) => valor.normalize("NFC").toLowerCase().includes(consulta)));
    limpiar.disabled = campo.value === "";
    recuento.textContent = traducir(consulta ? "coincidencias_pagina" : "recuento", { cuenta: filas.length, total: modelo.filas.length });
    resultado.replaceChildren(filas.length ? tabla(d, modelo, formatos, anadirDatoTraza, traducir, filas)
      : mensaje(d, traducir(modelo.filas.length ? "sin_coincidencias" : "vacio")));
  }
  campo.addEventListener("input", filtrar);
  limpiar.addEventListener("click", () => { campo.value = ""; filtrar(); campo.focus(); });
  filtrar(); cuerpo.append(busqueda, alcance, resultado);
  if (modelo.hayPaginaSiguiente) cuerpo.append(nodo(d, "p", traducir("paginas")));
  return panel;
}
