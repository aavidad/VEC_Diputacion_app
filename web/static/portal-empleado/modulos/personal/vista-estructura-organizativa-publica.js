import {
  crearTraductorPersonal,
  formatearFechaEstructuraOrganizativa,
  formatearRecuentoEstructura,
} from "./i18n.js?v=20260924-f2-web2";

function nodo(documento, etiqueta, texto = "") {
  const salida = documento.createElement(etiqueta);
  if (texto) salida.textContent = texto;
  return salida;
}
function sigue(raiz, contenedor) {
  return raiz.querySelector?.("[data-personal-estructura-organizativa-publica]") === contenedor;
}
const FILAS_POR_PAGINA = 10;
function tabla(documento, estructura, t, pagina) {
  const tabla = nodo(documento, "table");
  tabla.className = "tabla-datos";
  tabla.append(nodo(documento, "caption", t("estructura_tabla")));
  const thead = nodo(documento, "thead");
  const cabecera = nodo(documento, "tr");
  ["estructura_cabecera_etiqueta", "estructura_cabecera_tipo", "estructura_cabecera_adscripcion", "estructura_cabecera_clave"].forEach((clave) => {
    const celda = nodo(documento, "th", t(clave));
    celda.setAttribute("scope", "col");
    cabecera.append(celda);
  });
  thead.append(cabecera);
  tabla.append(thead);
  const etiquetas = new Map(estructura.unidades.map((unidad) => [unidad.clave, unidad.etiqueta]));
  const cuerpo = nodo(documento, "tbody");
  estructura.unidades.slice(pagina * FILAS_POR_PAGINA, (pagina + 1) * FILAS_POR_PAGINA).forEach((unidad) => {
    const fila = nodo(documento, "tr");
    const etiqueta = nodo(documento, "th", unidad.etiqueta);
    etiqueta.setAttribute("scope", "row");
    const adscripcion = unidad.adscripcion_clave
      ? etiquetas.get(unidad.adscripcion_clave) || unidad.adscripcion_clave
      : t("estructura_sin_adscripcion");
    const clave = nodo(documento, "td");
    clave.append(nodo(documento, "small", unidad.clave));
    fila.append(etiqueta, nodo(documento, "td", t(`estructura_tipo_${unidad.tipo}`)), nodo(documento, "td", adscripcion), clave);
    cuerpo.append(fila);
  });
  tabla.append(cuerpo);
  const caja = nodo(documento, "div");
  caja.className = "tabla-contenedor";
  caja.dataset.personalEstructuraTabla = "";
  caja.setAttribute("tabindex", "0");
  caja.setAttribute("role", "region");
  caja.setAttribute("aria-label", t("estructura_tabla"));
  caja.append(tabla);
  return caja;
}
function ayuda(documento, fuente, t) {
  const boton = nodo(documento, "button", "?");
  boton.type = "button";
  boton.className = "boton-secundario";
  boton.dataset.personalEstructuraAyuda = "";
  boton.setAttribute("aria-label", t("estructura_ayuda"));
  boton.setAttribute("aria-expanded", "false");
  boton.setAttribute("aria-controls", "personal-estructura-ayuda-contenido");
  const contenido = nodo(documento, "div");
  contenido.id = "personal-estructura-ayuda-contenido";
  contenido.dataset.personalEstructuraAyudaContenido = "";
  contenido.hidden = true;
  contenido.append(nodo(documento, "p", t("estructura_ayuda")));
  if (fuente) {
    contenido.append(
      nodo(documento, "p", t("estructura_fuente", {
        ...fuente,
        actualizada_en: formatearFechaEstructuraOrganizativa(fuente.actualizada_en),
      })),
      nodo(documento, "p", t("estructura_huella", { huella: fuente.huella_sha256 })),
    );
  }
  boton.addEventListener("click", () => {
    contenido.hidden = !contenido.hidden;
    boton.setAttribute("aria-expanded", String(!contenido.hidden));
  });
  return { boton, contenido };
}
function pintar(raiz, contenedor, estado, t, pagina, cambiarPagina) {
  if (!sigue(raiz, contenedor)) return;
  const documento = contenedor.ownerDocument;
  const botonAnterior = contenedor.querySelector("[data-personal-estructura-ayuda]");
  const contenidoAnterior = contenedor.querySelector("[data-personal-estructura-ayuda-contenido]");
  const ayudaAbierta = contenidoAnterior ? !contenidoAnterior.hidden : false;
  const focoAyuda = documento.activeElement === botonAnterior;
  const focoPaginacion = ["anterior", "siguiente"].find((direccion) =>
    documento.activeElement === contenedor.querySelector(`[data-personal-estructura-${direccion}]`));
  contenedor.replaceChildren();
  const cabecera = nodo(documento, "header");
  cabecera.className = "cabecera-vista";
  const contextual = ayuda(documento, estado.estructura?.fuente, t);
  contextual.contenido.hidden = !ayudaAbierta;
  contextual.boton.setAttribute("aria-expanded", String(ayudaAbierta));
  cabecera.append(
    nodo(documento, "p", t("estructura_sobrelinea")),
    nodo(documento, "h2", t("estructura_titulo")),
    contextual.boton,
  );
  contenedor.append(cabecera, contextual.contenido);
  const restaurarFoco = () => {
    if (focoAyuda) {
      contextual.boton.focus?.();
      return;
    }
    if (!focoPaginacion) return;
    const mismaDireccion = contenedor.querySelector(`[data-personal-estructura-${focoPaginacion}]`);
    const alternativa = contenedor.querySelector(`[data-personal-estructura-${focoPaginacion === "siguiente" ? "anterior" : "siguiente"}]`);
    const destino = !mismaDireccion?.disabled ? mismaDireccion : alternativa;
    destino?.focus?.();
  };
  if (estado.tipo === "cargando") {
    const carga = nodo(documento, "p", t("estructura_cargando"));
    carga.setAttribute("role", "status");
    carga.setAttribute("aria-live", "polite");
    contenedor.append(carga);
    restaurarFoco();
    return;
  }
  if (estado.tipo === "error") {
    const error = nodo(documento, "p", t("estructura_error"));
    error.setAttribute("role", "alert");
    contenedor.append(error);
    restaurarFoco();
    return;
  }
  const estructura = estado.estructura;
  const aviso = nodo(documento, "p", t("estructura_aviso"));
  aviso.className = "nota-integracion";
  aviso.dataset.personalEstructuraAviso = "";
  aviso.setAttribute("role", "note");
  const paginas = Math.ceil(estructura.unidades.length / FILAS_POR_PAGINA);
  const marco = nodo(documento, "div");
  marco.className = "marco-tabla-paginado";
  const navegacion = nodo(documento, "nav");
  navegacion.className = "paginacion-marco";
  navegacion.setAttribute("aria-label", t("estructura_tabla"));
  const anterior = nodo(documento, "button", t("catalogo_anterior"));
  anterior.type = "button";
  anterior.dataset.personalEstructuraAnterior = "";
  anterior.disabled = pagina === 0;
  anterior.addEventListener("click", () => cambiarPagina(pagina - 1));
  const siguiente = nodo(documento, "button", t("catalogo_siguiente"));
  siguiente.type = "button";
  siguiente.dataset.personalEstructuraSiguiente = "";
  siguiente.disabled = pagina >= paginas - 1;
  siguiente.addEventListener("click", () => cambiarPagina(pagina + 1));
  const numero = new Intl.NumberFormat("es-ES");
  const posicion = nodo(documento, "span", `${numero.format(pagina + 1)} / ${numero.format(paginas)}`);
  posicion.setAttribute("aria-live", "polite");
  navegacion.append(anterior, posicion, siguiente);
  marco.append(tabla(documento, estructura, t, pagina), navegacion);
  contenedor.append(aviso, nodo(documento, "p", formatearRecuentoEstructura(estructura.unidades.length)), marco);
  restaurarFoco();
}
export async function montarModuloEstructuraOrganizativaPublica({ raiz, cliente, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !cliente?.obtener || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("módulo estructura organizativa pública no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento estructura organizativa pública no disponible");
  const t = crearTraductorPersonal();
  const contenedor = nodo(documento, "section");
  contenedor.className = "modulo-personal";
  contenedor.dataset.personalEstructuraOrganizativaPublica = "";
  raiz.append(contenedor);
  let activa = true;
  let pagina = 0;
  let estado = { tipo: "cargando" };
  const cambiarPagina = (siguiente) => {
    if (!activa || estado.tipo !== "disponible" || !sigue(raiz, contenedor)) return;
    const paginas = Math.ceil(estado.estructura.unidades.length / FILAS_POR_PAGINA);
    if (!Number.isSafeInteger(siguiente) || siguiente < 0 || siguiente >= paginas) return;
    pagina = siguiente;
    pintar(raiz, contenedor, estado, t, pagina, cambiarPagina);
  };
  const repintar = () => pintar(raiz, contenedor, estado, t, pagina, cambiarPagina);
  const controlador = new AbortController();
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    controlador.abort();
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  repintar();
  try {
    const estructura = await cliente.obtener({ signal: controlador.signal });
    if (activa && sigue(raiz, contenedor) && !controlador.signal.aborted) {
      estado = { tipo: "disponible", estructura };
      repintar();
    }
  } catch {
    if (activa && sigue(raiz, contenedor) && !controlador.signal.aborted) {
      anunciar(t("estructura_error"), "error");
      estado = { tipo: "error" };
      repintar();
    }
  }
  return Object.freeze({ desmontar });
}
