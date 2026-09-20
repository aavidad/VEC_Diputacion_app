import { CAPACIDAD_CONSULTAR_PUESTO, validarConsultaCategorias } from "./contrato.js?v=20260920-personal-catalogo-v1";
import { crearTraductorPersonal, formatearRecuentoCategorias } from "./i18n.js?v=20260920-personal-catalogo-v1";

function nodo(documento, etiqueta, texto = "") { const salida = documento.createElement(etiqueta); if (texto !== "") salida.textContent = texto; return salida; }
function sigueMontada(raiz, contenedor) { return raiz.querySelector?.("[data-personal-categorias]") === contenedor; }
function retirar(raiz, contenedor) { if (!sigueMontada(raiz, contenedor)) return; if (typeof contenedor.remove === "function") contenedor.remove(); else raiz.removeChild?.(contenedor); }

function formularioFiltros(documento, consulta, recargar, t) {
  const formulario = nodo(documento, "form"); formulario.dataset.personalCategoriasFiltros = "";
  const etiquetaQ = nodo(documento, "label", t("catalogo_buscar")); const entradaQ = nodo(documento, "input"); entradaQ.name = "q"; entradaQ.value = consulta.q; entradaQ.maxLength = 100; etiquetaQ.append(entradaQ);
  const etiquetaArea = nodo(documento, "label", t("catalogo_area")); const entradaArea = nodo(documento, "input"); entradaArea.name = "area"; entradaArea.value = consulta.area; entradaArea.maxLength = 80; etiquetaArea.append(entradaArea);
  const buscar = nodo(documento, "button", t("catalogo_accion_buscar")); buscar.type = "submit"; formulario.append(etiquetaQ, etiquetaArea, buscar);
  formulario.addEventListener("submit", (evento) => { evento.preventDefault(); recargar({ q: entradaQ.value.trim(), area: entradaArea.value.trim(), offset: 0 }); });
  return formulario;
}

function pintar(raiz, contenedor, estado, recargar, t) {
  if (!sigueMontada(raiz, contenedor)) return;
  const documento = contenedor.ownerDocument; contenedor.replaceChildren();
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-vista";
  cabecera.append(nodo(documento, "p", t("catalogo_sobrelinea")), nodo(documento, "h2", t("catalogo_titulo")), nodo(documento, "p", t("catalogo_ayuda"))); contenedor.append(cabecera);
  if (estado.tipo === "cargando") { const carga = nodo(documento, "p", t("catalogo_cargando")); carga.setAttribute("role", "status"); carga.setAttribute("aria-live", "polite"); contenedor.append(carga); return; }
  if (estado.tipo === "error") { const error = nodo(documento, "p", estado.mensaje); error.setAttribute("role", "alert"); contenedor.append(error, formularioFiltros(documento, estado.consulta, recargar, t)); return; }
  const pagina = estado.pagina;
  const nota = nodo(documento, "p", pagina.fuente.demostracion === true ? t("catalogo_demo") : t("catalogo_publicado")); nota.className = "nota-pendiente"; contenedor.append(nota);
  contenedor.append(nodo(documento, "p", t("catalogo_fuente", { revision: pagina.fuente.revision, catalogo: pagina.catalogo.catalogo_id, version: pagina.catalogo.catalogo_version, aviso: pagina.fuente.aviso })));
  contenedor.append(formularioFiltros(documento, estado.consulta, recargar, t));
  if (pagina.items.length === 0) { const vacio = nodo(documento, "p", t("catalogo_vacio")); vacio.setAttribute("role", "status"); contenedor.append(vacio); } else {
    const tabla = nodo(documento, "table"); tabla.className = "tabla-datos"; tabla.append(nodo(documento, "caption", t("catalogo_titulo")));
    const cabeceras = nodo(documento, "thead"); const filaCabecera = nodo(documento, "tr"); ["catalogo_cabecera_nombre", "catalogo_cabecera_area", "catalogo_cabecera_estado"].forEach((clave) => { const celda = nodo(documento, "th", t(clave)); celda.setAttribute("scope", "col"); filaCabecera.append(celda); }); cabeceras.append(filaCabecera); tabla.append(cabeceras);
    const cuerpo = nodo(documento, "tbody"); pagina.items.forEach((item) => { const fila = nodo(documento, "tr"); const nombre = nodo(documento, "th", item.name); nombre.setAttribute("scope", "row"); fila.append(nombre, nodo(documento, "td", item.area_etiqueta), nodo(documento, "td", item.state)); cuerpo.append(fila); }); tabla.append(cuerpo); contenedor.append(tabla);
  }
  const paginacion = nodo(documento, "nav"); paginacion.setAttribute("aria-label", t("catalogo_paginacion"));
  const anterior = nodo(documento, "button", t("catalogo_anterior")); anterior.type = "button"; anterior.dataset.personalCategoriasAnterior = ""; anterior.disabled = pagina.offset === 0;
  const siguiente = nodo(documento, "button", t("catalogo_siguiente")); siguiente.type = "button"; siguiente.dataset.personalCategoriasSiguiente = ""; siguiente.disabled = pagina.offset + pagina.items.length >= pagina.total;
  paginacion.append(nodo(documento, "span", formatearRecuentoCategorias(pagina.total)), anterior, siguiente); contenedor.append(paginacion);
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
    let siguienteConsulta;
    try { siguienteConsulta = validarConsultaCategorias({ ...consulta, ...cambios }); } catch { pintar(raiz, contenedor, { tipo: "error", mensaje: t("catalogo_filtro_invalido"), consulta }, recargar, t); return; }
    controlador?.abort(); const vuelo = new AbortController(); controlador = vuelo; consulta = siguienteConsulta; pintar(raiz, contenedor, { tipo: "cargando" }, recargar, t);
    try {
      const pagina = await cliente.listarCategorias(consulta, { signal: vuelo.signal });
      if (activa && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) pintar(raiz, contenedor, { tipo: "disponible", pagina, consulta }, recargar, t);
    } catch {
      if (activa && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) { const mensaje = t("catalogo_error"); anunciar(mensaje, "error"); pintar(raiz, contenedor, { tipo: "error", mensaje, consulta }, recargar, t); }
    } finally { if (controlador === vuelo) controlador = null; }
  };
  await recargar(); return Object.freeze({ desmontar, capacidad: CAPACIDAD_CONSULTAR_PUESTO });
}
