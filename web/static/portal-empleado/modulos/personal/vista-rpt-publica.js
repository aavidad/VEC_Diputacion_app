import { crearTraductorRPTPuestos, formatearCentimosRPT, formatearRecuentoRPTPuestos } from "./i18n-rpt-puestos.js?v=20260929-i18n-personal-v1";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
function nodo(documento, etiqueta, texto = "") { const salida = documento.createElement(etiqueta); if (texto !== "") salida.textContent = texto; return salida; }
function sigueMontada(raiz, contenedor) { return raiz.querySelector?.("[data-personal-rpt-publica]") === contenedor; }
function retirar(raiz, contenedor) { if (!sigueMontada(raiz, contenedor)) return; if (typeof contenedor.remove === "function") contenedor.remove(); else raiz.removeChild?.(contenedor); }
function formulario(documento, consulta, recargar, t) {
  const salida = nodo(documento, "form");
  salida.className = "panel filtros-quitables";
  salida.dataset.personalRptPublicaFiltros = "";
  salida.dataset.personalRptPublicaConsulta = `${consulta.q}|${consulta.categoria_clave}|${consulta.centro_codigo}`;
  const etiqueta = nodo(documento, "label"), entrada = nodo(documento, "input");
  entrada.type = "search"; entrada.name = "q"; entrada.value = consulta.q; entrada.maxLength = 100;
  entrada.dataset.personalRptPublicaBusqueda = "";
  etiqueta.append(nodo(documento, "span", t("buscar")), entrada);
  const boton = nodo(documento, "button", t("accion_buscar"));
  boton.type = "submit"; boton.className = "boton-primario";
  salida.append(etiqueta, boton);
  const activos = nodo(documento, "div"); activos.className = "filtros-activos";
  const filtro = (clave, valor, mensaje, borrar) => {
    if (!valor) return;
    const aplicada = nodo(documento, "p", t(mensaje, { valor, busqueda: valor }));
    aplicada.className = "rpt-huella";
    if (clave === "q") aplicada.dataset.personalRptPublicaBusquedaAplicada = "";
    const quitar = nodo(documento, "button", t(clave === "q" ? "quitar_busqueda" : "quitar_filtro"));
    quitar.type = "button"; quitar.className = "boton-secundario";
    quitar.dataset[clave === "q" ? "personalRptPublicaQuitarBusqueda" : "personalRptPublicaQuitarFiltro"] = clave;
    quitar.addEventListener("click", () => recargar(borrar, { enfocarResultado: clave !== "q" }));
    activos.append(aplicada, quitar);
  };
  filtro("q", consulta.q, "busqueda_aplicada", { q: "", offset: 0 });
  filtro("categoria_clave", consulta.categoria_clave, "categoria_aplicada", { categoria_clave: "", offset: 0 });
  filtro("centro_codigo", consulta.centro_codigo, "centro_aplicado", { centro_codigo: "", offset: 0 });
  if (activos.children.length > 0) salida.append(activos);
  salida.addEventListener("submit", (evento) => { evento.preventDefault(); recargar({ q: entrada.value.trim(), offset: 0 }); });
  return salida;
}
function conservarFormulario(documento, contenedor, consulta, recargar, t) {
  const anterior = contenedor.querySelector("[data-personal-rpt-publica-filtros]");
  const recuperar = Boolean(anterior?.contains?.(documento.activeElement));
  const clave = `${consulta.q}|${consulta.categoria_clave}|${consulta.centro_codigo}`;
  const elemento = anterior?.dataset.personalRptPublicaConsulta === clave ? anterior : formulario(documento, consulta, recargar, t);
  return { elemento, restaurarFoco() { if (recuperar) elemento.querySelector("[data-personal-rpt-publica-busqueda]")?.focus?.(); } };
}
function pestañas(documento, consulta, recargar, t) {
  const salida = nodo(documento, "nav"); salida.setAttribute("aria-label", t("titulo"));
  [["categorias", "pestana_categorias"], ["puestos", "pestana_puestos"], ["centros", "pestana_centros"]].forEach(([vista, clave]) => {
    const boton = nodo(documento, "button", t(clave)); boton.type = "button";
    boton.dataset.personalRptPublicaVista = vista;
    boton.setAttribute("aria-pressed", String(consulta.vista === vista)); boton.disabled = consulta.vista === vista;
    boton.addEventListener("click", () => recargar({ vista, categoria_clave: "", centro_codigo: "", offset: 0 }, { enfocarResultado: true })); salida.append(boton);
  });
  return salida;
}
function columnas(vista) {
  if (vista === "puestos") return [["codigo", "codigo"], ["denominacion", "denominacion"], ["centro", "centro"], ["grupos", "grupos"], ["nivel_destino", "nivel"], ["dotacion", "dotacion"]];
  if (vista === "centros") return [["codigo", "centro_codigo"], ["denominacion", "centro"], ["puestos", "puestos"], ["dotacion", "dotacion"]];
  return [["clave", "clave"], ["denominacion", "denominacion"], ["grupos", "grupos"], ["escalas", "escalas"], ["puestos_vinculados", "puestos"], ["dotacion_vinculada", "dotacion"]];
}
const CAMPOS_DETALLE_PUESTO = Object.freeze([["centro_codigo", "centro_codigo"], ["delegacion", "delegacion"], ["escala", "escala"], ["categoria_clave", "categoria"], ["complemento_especifico_anual_centimos", "complemento"], ["tipo", "tipo"], ["provision", "provision"]]);
function valor(item, campo, t) {
  if (campo === "grupos" || campo === "escalas") return item[campo].join(", ") || t(campo === "escalas" ? "sin_escalas" : "sin_categoria");
  if (campo === "categoria_clave") return item[campo] || t("sin_categoria");
  if (campo === "escala") return item[campo] || t("sin_escalas");
  if (campo === "complemento_especifico_anual_centimos") return formatearCentimosRPT(item[campo]);
  return String(item[campo]);
}
function fechaGeneracionRPT(valor) {
  if (typeof valor !== "string") return "";
  if (/^\d{4}-\d{2}-\d{2}$/u.test(valor)) {
    const fecha = new Date(`${valor}T12:00:00Z`);
    if (Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor)
      return new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "long", timeZone: "Europe/Madrid" }).format(fecha);
  }
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(valor)) {
    const fecha = new Date(valor);
    if (Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 19) === valor.slice(0, 19))
      return new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "long", timeStyle: "short", timeZone: "Europe/Madrid" }).format(fecha);
  }
  return valor;
}
function detallePuesto(documento, item, identificador, t, cerrar) { const detalle = nodo(documento, "section"); detalle.id = identificador; detalle.dataset.personalRptPublicaDetalle = ""; detalle.setAttribute("aria-label", t("detalle_puesto", { codigo: valor(item, "codigo", t) })); const titulo = nodo(documento, "h3", t("detalle_puesto", { codigo: valor(item, "codigo", t) })); const boton = nodo(documento, "button", t("cerrar_detalle")); boton.type = "button"; boton.dataset.personalRptPublicaCerrarDetalle = ""; boton.addEventListener("click", cerrar); const lista = nodo(documento, "dl"); CAMPOS_DETALLE_PUESTO.forEach(([campo, clave]) => lista.append(nodo(documento, "dt", t(clave)), nodo(documento, "dd", valor(item, campo, t)))); detalle.append(titulo, boton, lista); return detalle; }
function tablaRPT(documento, pagina, t, recargar) {
  const tabla = nodo(documento, "table"); tabla.className = "tabla-datos";
  tabla.append(nodo(documento, "caption", t(pagina.vista === "puestos" ? "tabla_puestos" : pagina.vista === "centros" ? "tabla_centros" : "tabla_categorias")));
  const cabeza = nodo(documento, "thead"), filaCabeza = nodo(documento, "tr"), columnasVista = columnas(pagina.vista);
  columnasVista.forEach(([, clave]) => { const celda = nodo(documento, "th", t(clave)); celda.setAttribute("scope", "col"); filaCabeza.append(celda); });
  cabeza.append(filaCabeza); tabla.append(cabeza);
  const cuerpo = nodo(documento, "tbody"); let abierto = null, focoPendiente = null;
  const cerrar = (devolverFoco = false) => { if (abierto === null) return; if (devolverFoco) focoPendiente = abierto; abierto = null; pintarFilas(); };
  cuerpo.addEventListener("keydown", (evento) => { if (evento.key === "Escape") { evento.preventDefault(); cerrar(true); } });
  const pintarFilas = () => {
    cuerpo.replaceChildren();
    pagina.items.forEach((item, indice) => {
      const fila = nodo(documento, "tr"), identificador = `personal-rpt-puesto-${pagina.offset}-${indice}`;
      let botonDetalle = null;
      const alternar = () => { if (abierto === indice) { focoPendiente = indice; abierto = null; } else abierto = indice; pintarFilas(); };
      columnasVista.forEach(([campo], indiceCampo) => {
        const celda = nodo(documento, indiceCampo === 0 ? "th" : "td");
        if (indiceCampo === 0) celda.setAttribute("scope", "row");
        if (pagina.vista === "puestos" && indiceCampo === 0) {
          const boton = nodo(documento, "button", valor(item, campo, t)); botonDetalle = boton; boton.type = "button";
          boton.dataset.personalRptPublicaDetalleBoton = ""; boton.setAttribute("aria-controls", identificador);
          boton.setAttribute("aria-expanded", String(abierto === indice));
          boton.setAttribute("aria-label", t(abierto === indice ? "ocultar_detalle" : "ver_detalle", { codigo: valor(item, campo, t) }));
          boton.addEventListener("click", alternar);
          boton.addEventListener("keydown", (evento) => { if (evento.key === " ") { evento.preventDefault(); alternar(); } });
          celda.append(boton);
        } else if (pagina.vista !== "puestos") {
          const boton = nodo(documento, "button", valor(item, campo, t)); boton.type = "button";
          boton.className = "boton-enlace"; boton.dataset.personalRptPublicaEnlace = campo;
          boton.setAttribute("aria-label", t(pagina.vista === "centros" ? "ver_puestos_centro" : "ver_puestos_categoria", { valor: item.denominacion }));
          boton.addEventListener("click", () => recargar({ vista: "puestos", q: "", categoria_clave: pagina.vista === "categorias" ? item.clave : "", centro_codigo: pagina.vista === "centros" ? item.codigo : "", offset: 0 }, { enfocarResultado: true }));
          celda.append(boton);
          if (pagina.vista === "categorias" && campo === "denominacion" && !item.recuento_coincide) {
            const aviso = nodo(documento, "small", t("aviso_recuento_categoria")); aviso.className = "rpt-huella"; celda.append(aviso);
          }
        } else celda.textContent = valor(item, campo, t);
        fila.append(celda);
      });
      cuerpo.append(fila);
      if (focoPendiente === indice) { botonDetalle?.focus?.(); focoPendiente = null; }
      if (pagina.vista === "puestos" && abierto === indice) {
        const filaDetalle = nodo(documento, "tr"); filaDetalle.dataset.personalRptPublicaDetalleFila = "";
        const celdaDetalle = nodo(documento, "td"); celdaDetalle.colSpan = columnasVista.length;
        celdaDetalle.append(detallePuesto(documento, item, identificador, t, () => cerrar(true)));
        filaDetalle.append(celdaDetalle); cuerpo.append(filaDetalle);
      }
    });
  };
  if (pagina.items.length === 0) {
    const fila = nodo(documento, "tr"), celda = nodo(documento, "td", t("vacio"));
    celda.colSpan = columnasVista.length; fila.append(celda); cuerpo.append(fila);
  } else pintarFilas();
  tabla.append(cuerpo);
  const contenedor = nodo(documento, "div"); contenedor.className = "tabla-contenedor";
  contenedor.setAttribute("tabindex", "0"); contenedor.setAttribute("role", "region"); contenedor.dataset.personalRptPublicaTabla = "";
  contenedor.setAttribute("aria-label", t(pagina.vista === "puestos" ? "tabla_puestos" : pagina.vista === "centros" ? "tabla_centros" : "tabla_categorias"));
  contenedor.append(tabla); return contenedor;
}
function resumenEnlazado(documento, pagina, recargar, t) {
  const resumen = nodo(documento, "div"); resumen.className = "panel"; resumen.dataset.personalRptPublicaResumen = "";
  const cuerpo = nodo(documento, "div"); cuerpo.className = "cuerpo-panel";
  const numero = new Intl.NumberFormat(LOCALIZACION_ACTUAL, { useGrouping: "always" });
  [["puestos", "puestos", "puestos"], ["dotacion", "dotacion", "puestos"], ["categorias", "categorias", "categorias"], ["centros", "centros", "centros"]].forEach(([campo, clave, vista]) => {
    const boton = nodo(documento, "button", t(`resumen_${clave}`, { total: numero.format(pagina.resumen[campo]) }));
    boton.type = "button"; boton.className = "boton-enlace"; boton.dataset.personalRptPublicaResumenEnlace = campo;
    boton.addEventListener("click", () => recargar({ vista, q: "", categoria_clave: "", centro_codigo: "", offset: 0 }, { enfocarResultado: true }));
    cuerpo.append(boton);
  });
  resumen.append(cuerpo); return resumen;
}
function pintar(raiz, contenedor, estado, recargar, t) {
  if (!sigueMontada(raiz, contenedor)) return;
  const documento = contenedor.ownerDocument, filtros = conservarFormulario(documento, contenedor, estado.consulta, recargar, t);
  contenedor.replaceChildren();
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-vista";
  const ayuda = nodo(documento, "div"); ayuda.id = "personal-rpt-publica-ayuda";
  ayuda.className = "panel ayuda-contextual"; ayuda.dataset.personalRptPublicaAyuda = ""; ayuda.hidden = true;
  const cuerpoAyuda = nodo(documento, "div"); cuerpoAyuda.className = "cuerpo-panel"; cuerpoAyuda.append(nodo(documento, "p", t("ayuda"))); ayuda.append(cuerpoAyuda);
  const abrirAyuda = nodo(documento, "button", "?"); abrirAyuda.type = "button"; abrirAyuda.dataset.personalRptPublicaAbrirAyuda = "";
  abrirAyuda.setAttribute("aria-label", t("ayuda")); abrirAyuda.setAttribute("aria-controls", ayuda.id); abrirAyuda.setAttribute("aria-expanded", "false");
  abrirAyuda.addEventListener("click", () => { ayuda.hidden = !ayuda.hidden; abrirAyuda.setAttribute("aria-expanded", String(!ayuda.hidden)); });
  abrirAyuda.addEventListener("keydown", (evento) => { if (evento.key === "Escape") { evento.preventDefault(); ayuda.hidden = true; abrirAyuda.setAttribute("aria-expanded", "false"); abrirAyuda.focus?.(); } });
  cabecera.append(nodo(documento, "h2", t("titulo")), abrirAyuda); contenedor.append(cabecera, ayuda);
  if (estado.tipo === "cargando" || estado.tipo === "error") {
    const aviso = nodo(documento, "p", estado.tipo === "cargando" ? t("cargando") : estado.mensaje);
    aviso.setAttribute("role", estado.tipo === "cargando" ? "status" : "alert");
    if (estado.tipo === "cargando") aviso.setAttribute("aria-live", "polite");
    aviso.dataset.personalRptPublicaEstado = ""; aviso.setAttribute("tabindex", "-1");
    contenedor.append(aviso, pestañas(documento, estado.consulta, recargar, t), filtros.elemento); filtros.restaurarFoco(); return;
  }
  const { pagina } = estado;
  cuerpoAyuda.append(nodo(documento, "p", t("fuente", pagina.fuente)));
  const generada = fechaGeneracionRPT(pagina.fuente.generado_en);
  if (generada) cuerpoAyuda.append(nodo(documento, "p", t("generacion", { valor: generada })));
  const huella = nodo(documento, "p", t("huella", { importacion: pagina.fuente.importacion, huella: pagina.fuente.huella_sha256 }));
  huella.className = "rpt-huella"; cuerpoAyuda.append(huella);
  contenedor.append(resumenEnlazado(documento, pagina, recargar, t), pestañas(documento, estado.consulta, recargar, t), filtros.elemento, tablaRPT(documento, pagina, t, recargar));
  const navegacion = nodo(documento, "nav"); navegacion.setAttribute("aria-label", t("paginacion"));
  const anterior = nodo(documento, "button", t("anterior")); anterior.type = "button"; anterior.dataset.personalRptPublicaAnterior = ""; anterior.disabled = pagina.offset === 0;
  const siguiente = nodo(documento, "button", t("siguiente")); siguiente.type = "button"; siguiente.dataset.personalRptPublicaSiguiente = ""; siguiente.disabled = pagina.offset + pagina.items.length >= pagina.total;
  navegacion.append(nodo(documento, "span", formatearRecuentoRPTPuestos(pagina.total, pagina.vista)), anterior, siguiente);
  anterior.addEventListener("click", () => recargar({ offset: Math.max(0, pagina.offset - pagina.limit) }, { enfocarResultado: true }));
  siguiente.addEventListener("click", () => recargar({ offset: pagina.offset + pagina.limit }, { enfocarResultado: true }));
  contenedor.append(navegacion); filtros.restaurarFoco();
}
function consultaDesdeURL() {
  const parametros = new URLSearchParams(globalThis.window?.location?.search || "");
  const vista = parametros.get("rpt_vista") || "categorias";
  const q = parametros.get("rpt_q") || "";
  const categoria_clave = parametros.get("rpt_categoria") || "", centro_codigo = parametros.get("rpt_centro") || "";
  const offset = Number(parametros.get("rpt_offset") || "0");
  if (!["categorias", "puestos", "centros"].includes(vista) || q !== q.trim() || q.length > 100
    || !Number.isSafeInteger(offset) || offset < 0 || categoria_clave && (categoria_clave.length > 64 || !/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/u.test(categoria_clave))
    || centro_codigo.length > 64 || vista !== "puestos" && (categoria_clave || centro_codigo))
    return Object.freeze({ vista: "categorias", q: "", categoria_clave: "", centro_codigo: "", limit: 25, offset: 0 });
  return Object.freeze({ vista, q, categoria_clave, centro_codigo, limit: 25, offset });
}
function conservarConsultaURL(consulta) {
  const ventana = globalThis.window;
  if (!ventana?.location?.pathname || !ventana.history?.replaceState) return;
  const parametros = new URLSearchParams(ventana.location.search || "");
  for (const clave of ["rpt_vista", "rpt_q", "rpt_categoria", "rpt_centro", "rpt_offset"]) parametros.delete(clave);
  if (consulta.vista !== "categorias") parametros.set("rpt_vista", consulta.vista);
  if (consulta.q) parametros.set("rpt_q", consulta.q);
  if (consulta.categoria_clave) parametros.set("rpt_categoria", consulta.categoria_clave);
  if (consulta.centro_codigo) parametros.set("rpt_centro", consulta.centro_codigo);
  if (consulta.offset) parametros.set("rpt_offset", String(consulta.offset));
  const query = parametros.toString();
  ventana.history.replaceState(null, "", `${ventana.location.pathname}${query ? `?${query}` : ""}${ventana.location.hash || ""}`);
}
export async function montarModuloRPTPublica({ raiz, cliente, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !cliente?.listar || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function"))
    throw new TypeError("módulo RPT pública no disponible");
  const documento = raiz.ownerDocument; if (!documento?.createElement) throw new TypeError("documento RPT pública no disponible");
  const t = crearTraductorRPTPuestos(), contenedor = nodo(documento, "section");
  contenedor.className = "modulo-personal"; contenedor.dataset.personalRptPublica = ""; raiz.append(contenedor);
  let activa = true, controlador = null, consulta = consultaDesdeURL();
  const desmontar = () => { if (!activa) return; activa = false; controlador?.abort(); retirar(raiz, contenedor); };
  registrarDesmontar?.(desmontar);
  const recargar = async (cambios = {}, { enfocarResultado = false } = {}) => {
    if (!activa || !sigueMontada(raiz, contenedor)) return;
    const siguienteConsulta = { ...consulta, ...cambios };
    if (!["categorias", "puestos", "centros"].includes(siguienteConsulta.vista) || typeof siguienteConsulta.q !== "string"
      || siguienteConsulta.q !== siguienteConsulta.q.trim() || siguienteConsulta.q.length > 100
      || !Number.isSafeInteger(siguienteConsulta.limit) || siguienteConsulta.limit < 1 || siguienteConsulta.limit > 100
      || !Number.isSafeInteger(siguienteConsulta.offset) || siguienteConsulta.offset < 0
      || siguienteConsulta.vista !== "puestos" && (siguienteConsulta.categoria_clave || siguienteConsulta.centro_codigo)
      || siguienteConsulta.categoria_clave && (siguienteConsulta.categoria_clave.length > 64 || !/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/u.test(siguienteConsulta.categoria_clave))
      || siguienteConsulta.centro_codigo && (typeof siguienteConsulta.centro_codigo !== "string" || siguienteConsulta.centro_codigo.length > 64)) {
      pintar(raiz, contenedor, { tipo: "error", mensaje: t("error"), consulta }, recargar, t); return;
    }
    controlador?.abort(); const vuelo = new AbortController(); controlador = vuelo;
    consulta = Object.freeze(siguienteConsulta); conservarConsultaURL(consulta);
    pintar(raiz, contenedor, { tipo: "cargando", consulta }, recargar, t);
    if (enfocarResultado) contenedor.querySelector("[data-personal-rpt-publica-estado]")?.focus?.();
    try {
      const pagina = await cliente.listar(consulta, { signal: vuelo.signal });
      if (activa && controlador === vuelo && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) {
        pintar(raiz, contenedor, { tipo: "disponible", pagina, consulta }, recargar, t);
        if (enfocarResultado) contenedor.querySelector("[data-personal-rpt-publica-tabla]")?.focus?.();
      }
    } catch {
      if (activa && controlador === vuelo && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) {
        const mensaje = t("error"); anunciar(mensaje, "error");
        pintar(raiz, contenedor, { tipo: "error", mensaje, consulta }, recargar, t);
        if (enfocarResultado) contenedor.querySelector("[data-personal-rpt-publica-estado]")?.focus?.();
      }
    } finally { if (controlador === vuelo) controlador = null; }
  };
  await recargar(); return Object.freeze({ desmontar });
}
