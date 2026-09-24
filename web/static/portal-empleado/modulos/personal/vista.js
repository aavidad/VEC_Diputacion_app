import { CAPACIDAD_CONSULTAR_PUESTO, validarConsultaCategorias } from "./contrato.js?v=20260920-personal-catalogo-v1";
import { crearTraductorPersonal, formatearRecuentoCategorias } from "./i18n.js?v=20260925-organizacion-historica-v1";

function nodo(documento, etiqueta, texto = "") { const salida = documento.createElement(etiqueta); if (texto !== "") salida.textContent = texto; return salida; }
function sigueMontada(raiz, contenedor) { return raiz.querySelector?.("[data-personal-categorias]") === contenedor; }
function retirar(raiz, contenedor) { if (!sigueMontada(raiz, contenedor)) return; if (typeof contenedor.remove === "function") contenedor.remove(); else raiz.removeChild?.(contenedor); }
function control(contenedor, clave) { return contenedor.querySelector?.(`[data-personal-categorias-foco="${clave}"]`); }
function focoActual(documento, contenedor) {
  const activo = documento.activeElement;
  return activo && contenedor.contains?.(activo) ? activo.dataset?.personalCategoriasFoco || "" : "";
}
function borradorActual(contenedor) {
  const q = control(contenedor, "q")?.value; const area = control(contenedor, "area")?.value;
  return typeof q === "string" && typeof area === "string" ? { q, area } : null;
}
function restaurarFoco(contenedor, clave, cargando = false) {
  if (!clave) return;
  let destino = control(contenedor, clave);
  if (!destino || destino.disabled) {
    if (cargando && (clave === "anterior" || clave === "siguiente")) destino = control(contenedor, "espera");
    else if (clave === "anterior" || clave === "siguiente") destino = control(contenedor, clave === "anterior" ? "siguiente" : "anterior");
  }
  if ((!destino || destino.disabled) && (clave === "anterior" || clave === "siguiente")) destino = control(contenedor, "buscar");
  if (destino && !destino.disabled) destino.focus?.();
}
function clasificarError(causa, t) {
  const estado = causa?.name === "ErrorClienteCategoriasPersonal" && causa.codigo === "estado_no_valido" ? causa.estado : 0;
  if (estado === 401 || estado === 403) return { tipo: "denegado", mensaje: t("catalogo_denegado") };
  if (estado === 404) return { tipo: "sin_ruta", mensaje: t("catalogo_sin_ruta") };
  if (estado === 503) return { tipo: "servicio_no_disponible", mensaje: t("catalogo_servicio_no_disponible") };
  return { tipo: "error", mensaje: t("catalogo_error") };
}

function formularioFiltros(documento, consulta, recargar, t, borrador = null) {
  const formulario = nodo(documento, "form"); formulario.className = "barra-filtros panel"; formulario.dataset.personalCategoriasFiltros = "";
  const etiquetaQ = nodo(documento, "label"); etiquetaQ.className = "campo"; const entradaQ = nodo(documento, "input"); entradaQ.name = "q"; entradaQ.dataset.personalCategoriasFoco = "q"; entradaQ.value = borrador?.q ?? consulta.q; entradaQ.maxLength = 100; etiquetaQ.append(nodo(documento, "span", t("catalogo_buscar")), entradaQ);
  const etiquetaArea = nodo(documento, "label"); etiquetaArea.className = "campo"; const entradaArea = nodo(documento, "input"); entradaArea.name = "area"; entradaArea.dataset.personalCategoriasFoco = "area"; entradaArea.value = borrador?.area ?? consulta.area; entradaArea.maxLength = 80; etiquetaArea.append(nodo(documento, "span", t("catalogo_area")), entradaArea);
  const buscar = nodo(documento, "button", t("catalogo_accion_buscar")); buscar.type = "submit"; buscar.className = "boton-primario"; buscar.dataset.personalCategoriasFoco = "buscar"; formulario.append(etiquetaQ, etiquetaArea, buscar);
  formulario.addEventListener("submit", (evento) => { evento.preventDefault(); recargar({ q: entradaQ.value.trim(), area: entradaArea.value.trim(), offset: 0 }); });
  return formulario;
}

function pintar(raiz, contenedor, estado, recargar, t, borrador = null) {
  if (!sigueMontada(raiz, contenedor)) return;
  const documento = contenedor.ownerDocument; contenedor.replaceChildren(); contenedor.dataset.personalCategoriasEstado = estado.tipo;
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-vista";
  cabecera.append(nodo(documento, "p", t("catalogo_sobrelinea")), nodo(documento, "h2", t("catalogo_titulo")), nodo(documento, "p", t("catalogo_ayuda"))); contenedor.append(cabecera);
  if (estado.tipo === "cargando") { const carga = nodo(documento, "p", t("catalogo_cargando")); carga.setAttribute("role", "status"); carga.setAttribute("aria-live", "polite"); carga.setAttribute("tabindex", "-1"); carga.dataset.personalCategoriasFoco = "espera"; contenedor.append(carga, formularioFiltros(documento, estado.consulta, recargar, t, borrador)); return; }
  if (estado.tipo !== "disponible") { const error = nodo(documento, "p", estado.mensaje); error.setAttribute("role", "alert"); contenedor.append(error, formularioFiltros(documento, estado.consulta, recargar, t, borrador)); return; }
  const pagina = estado.pagina;
  const nota = nodo(documento, "p", pagina.fuente.demostracion === true ? t("catalogo_demo") : t("catalogo_publicado")); nota.className = "nota-pendiente"; contenedor.append(nota);
  contenedor.append(nodo(documento, "p", t("catalogo_fuente", { revision: pagina.fuente.revision, catalogo: pagina.catalogo.catalogo_id, version: pagina.catalogo.catalogo_version, aviso: pagina.fuente.aviso })));
  contenedor.append(formularioFiltros(documento, estado.consulta, recargar, t, borrador));
  if (pagina.items.length === 0) { const vacio = nodo(documento, "p", t("catalogo_vacio")); vacio.setAttribute("role", "status"); contenedor.append(vacio); } else {
    const marco = nodo(documento, "div"); marco.className = "marco-tabla-paginado panel"; marco.dataset.personalCategoriasMarco = "";
    const region = nodo(documento, "div"); region.className = "tabla-contenedor";
    region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("catalogo_titulo"));
    region.setAttribute("style", "max-height:min(40vh,360px)");
    const tabla = nodo(documento, "table"); tabla.className = "tabla-datos"; tabla.append(nodo(documento, "caption", t("catalogo_titulo")));
    const cabeceras = nodo(documento, "thead"); const filaCabecera = nodo(documento, "tr"); ["catalogo_cabecera_nombre", "catalogo_cabecera_area", "catalogo_cabecera_estado"].forEach((clave) => { const celda = nodo(documento, "th", t(clave)); celda.setAttribute("scope", "col"); filaCabecera.append(celda); }); cabeceras.append(filaCabecera); tabla.append(cabeceras);
    const cuerpo = nodo(documento, "tbody"); pagina.items.forEach((item) => { const fila = nodo(documento, "tr"); const nombre = nodo(documento, "th", item.name); nombre.setAttribute("scope", "row"); fila.append(nombre, nodo(documento, "td", item.area_etiqueta), nodo(documento, "td", item.state)); cuerpo.append(fila); }); tabla.append(cuerpo); region.append(tabla); marco.append(region); contenedor.append(marco);
  }
  const paginacion = nodo(documento, "nav"); paginacion.className = "paginacion-marco"; paginacion.setAttribute("aria-label", t("catalogo_paginacion"));
  const anterior = nodo(documento, "button", t("catalogo_anterior")); anterior.type = "button"; anterior.dataset.personalCategoriasAnterior = ""; anterior.dataset.personalCategoriasFoco = "anterior"; anterior.disabled = pagina.offset === 0;
  const siguiente = nodo(documento, "button", t("catalogo_siguiente")); siguiente.type = "button"; siguiente.dataset.personalCategoriasSiguiente = ""; siguiente.dataset.personalCategoriasFoco = "siguiente"; siguiente.disabled = pagina.offset + pagina.items.length >= pagina.total;
  paginacion.append(nodo(documento, "span", formatearRecuentoCategorias(pagina.total)), anterior, siguiente);
  (contenedor.querySelector?.("[data-personal-categorias-marco]") || contenedor).append(paginacion);
  anterior.addEventListener("click", () => recargar({ offset: Math.max(0, pagina.offset - pagina.limit) })); siguiente.addEventListener("click", () => recargar({ offset: pagina.offset + pagina.limit }));
}

/** Monta el catálogo profesional real de Personal con cancelación de navegación. */
export async function montarModuloPersonal({ raiz, cliente, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !cliente?.listarCategorias || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("módulo de categorías de Personal no disponible");
  const documento = raiz.ownerDocument; if (!documento?.createElement) throw new TypeError("documento de Personal no disponible");
  const t = crearTraductorPersonal(); const contenedor = nodo(documento, "section"); contenedor.className = "modulo-personal"; contenedor.dataset.personalCategorias = ""; raiz.append(contenedor);
  let activa = true; let controlador = null; let consulta = Object.freeze({ q: "", area: "", limit: 25, offset: 0 });
  const desmontar = () => { if (!activa) return; activa = false; controlador?.abort(); retirar(raiz, contenedor); };
  registrarDesmontar?.(desmontar);
  const recargar = async (cambios = {}) => {
    if (!activa || !sigueMontada(raiz, contenedor)) return;
    const origenFoco = focoActual(documento, contenedor);
    let siguienteConsulta;
    try { siguienteConsulta = validarConsultaCategorias({ ...consulta, ...cambios }); } catch {
      controlador?.abort(); controlador = null;
      const borrador = borradorActual(contenedor);
      pintar(raiz, contenedor, { tipo: "error", mensaje: t("catalogo_filtro_invalido"), consulta }, recargar, t, borrador);
      restaurarFoco(contenedor, origenFoco); return;
    }
    controlador?.abort(); const vuelo = new AbortController(); controlador = vuelo; consulta = siguienteConsulta;
    pintar(raiz, contenedor, { tipo: "cargando", consulta }, recargar, t);
    restaurarFoco(contenedor, origenFoco, true);
    try {
      const pagina = await cliente.listarCategorias(consulta, { signal: vuelo.signal });
      if (activa && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) {
        const activo = focoActual(documento, contenedor); const borrador = borradorActual(contenedor);
        pintar(raiz, contenedor, { tipo: "disponible", pagina, consulta }, recargar, t, borrador);
        restaurarFoco(contenedor, activo === "espera" ? origenFoco : activo);
      }
    } catch (causa) {
      if (activa && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) {
        const estado = clasificarError(causa, t); anunciar(estado.mensaje, "error");
        const activo = focoActual(documento, contenedor); const borrador = borradorActual(contenedor);
        pintar(raiz, contenedor, { ...estado, consulta }, recargar, t, borrador);
        restaurarFoco(contenedor, activo === "espera" ? origenFoco : activo);
      }
    } finally { if (controlador === vuelo) controlador = null; }
  };
  await recargar(); return Object.freeze({ desmontar, capacidad: CAPACIDAD_CONSULTAR_PUESTO });
}
