import { crearTraductorPersonal, formatearRecuentoRPT } from "./i18n.js?v=20260920-personal-rpt-publica-v3";

function nodo(documento, etiqueta, texto = "") {
  const salida = documento.createElement(etiqueta);
  if (texto !== "") salida.textContent = texto;
  return salida;
}

function sigueMontada(raiz, contenedor) {
  return raiz.querySelector?.("[data-personal-rpt-publica]") === contenedor;
}

function retirar(raiz, contenedor) {
  if (!sigueMontada(raiz, contenedor)) return;
  if (typeof contenedor.remove === "function") contenedor.remove();
  else raiz.removeChild?.(contenedor);
}

function formulario(documento, consulta, recargar, t) {
  const salida = nodo(documento, "form");
  salida.dataset.personalRptPublicaFiltros = "";
  const etiqueta = nodo(documento, "label", t("rpt_buscar"));
  const entrada = nodo(documento, "input");
  entrada.name = "q";
  entrada.value = consulta.q;
  entrada.maxLength = 100;
  etiqueta.append(entrada);
  const boton = nodo(documento, "button", t("rpt_accion_buscar"));
  boton.type = "submit";
  salida.append(etiqueta, boton);
  salida.addEventListener("submit", (evento) => {
    evento.preventDefault();
    recargar({ q: entrada.value.trim(), offset: 0 });
  });
  return salida;
}

function tablaRPT(documento, pagina, t) {
  const tabla = nodo(documento, "table");
  tabla.className = "tabla-datos";
  tabla.append(nodo(documento, "caption", t("rpt_titulo")));
  const cabeza = nodo(documento, "thead");
  const filaCabeza = nodo(documento, "tr");
  ["rpt_cabecera_clave", "rpt_cabecera_denominacion", "rpt_cabecera_grupos", "rpt_cabecera_escalas", "rpt_cabecera_puestos", "rpt_cabecera_dotacion"].forEach((clave) => {
    const celda = nodo(documento, "th", t(clave));
    celda.setAttribute("scope", "col");
    filaCabeza.append(celda);
  });
  cabeza.append(filaCabeza);
  tabla.append(cabeza);
  const cuerpo = nodo(documento, "tbody");
  pagina.items.forEach((item) => {
    const fila = nodo(documento, "tr");
    const clave = nodo(documento, "th", item.clave);
    clave.setAttribute("scope", "row");
    fila.append(
      clave,
      nodo(documento, "td", item.denominacion),
      nodo(documento, "td", item.grupos.join(", ")),
      nodo(documento, "td", item.escalas.join(", ") || t("rpt_sin_escalas")),
      nodo(documento, "td", String(item.puestos)),
      nodo(documento, "td", String(item.dotacion)),
    );
    cuerpo.append(fila);
  });
  if (pagina.items.length === 0) {
    const fila = nodo(documento, "tr");
    const celda = nodo(documento, "td", t("rpt_vacio"));
    celda.colSpan = 6;
    fila.append(celda);
    cuerpo.append(fila);
  }
  tabla.append(cuerpo);
  const contenedor = nodo(documento, "div");
  contenedor.className = "tabla-contenedor";
  contenedor.setAttribute("tabindex", "0");
  contenedor.setAttribute("role", "region");
  contenedor.setAttribute("aria-label", t("rpt_tabla"));
  contenedor.append(tabla);
  return contenedor;
}

function pintar(raiz, contenedor, estado, recargar, t) {
  if (!sigueMontada(raiz, contenedor)) return;
  const documento = contenedor.ownerDocument;
  contenedor.replaceChildren();
  const cabecera = nodo(documento, "header");
  cabecera.className = "cabecera-vista";
  cabecera.append(
    nodo(documento, "p", t("rpt_sobrelinea")),
    nodo(documento, "h2", t("rpt_titulo")),
    nodo(documento, "p", t("rpt_ayuda")),
  );
  contenedor.append(cabecera);
  if (estado.tipo === "cargando") {
    const carga = nodo(documento, "p", t("rpt_cargando"));
    carga.setAttribute("role", "status");
    carga.setAttribute("aria-live", "polite");
    contenedor.append(carga);
    return;
  }
  if (estado.tipo === "error") {
    const error = nodo(documento, "p", estado.mensaje);
    error.setAttribute("role", "alert");
    contenedor.append(error, formulario(documento, estado.consulta, recargar, t));
    return;
  }
  const { pagina } = estado;
  const huella = nodo(documento, "p", t("rpt_huella", {
    importacion: pagina.fuente.importacion,
    huella: pagina.fuente.huella_sha256,
  }));
  huella.className = "rpt-huella";
  contenedor.append(
    nodo(documento, "p", t("rpt_fuente", pagina.fuente)),
    huella,
    formulario(documento, estado.consulta, recargar, t),
    tablaRPT(documento, pagina, t),
  );
  const navegacion = nodo(documento, "nav");
  navegacion.setAttribute("aria-label", t("rpt_paginacion"));
  const anterior = nodo(documento, "button", t("rpt_anterior"));
  anterior.type = "button";
  anterior.dataset.personalRptPublicaAnterior = "";
  anterior.disabled = pagina.offset === 0;
  const siguiente = nodo(documento, "button", t("rpt_siguiente"));
  siguiente.type = "button";
  siguiente.dataset.personalRptPublicaSiguiente = "";
  siguiente.disabled = pagina.offset + pagina.items.length >= pagina.total;
  navegacion.append(nodo(documento, "span", formatearRecuentoRPT(pagina.total)), anterior, siguiente);
  anterior.addEventListener("click", () => recargar({ offset: Math.max(0, pagina.offset - pagina.limit) }));
  siguiente.addEventListener("click", () => recargar({ offset: pagina.offset + pagina.limit }));
  contenedor.append(navegacion);
}

export async function montarModuloRPTPublica({ raiz, cliente, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !cliente?.listar || typeof anunciar !== "function"
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("módulo RPT pública no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento RPT pública no disponible");
  const t = crearTraductorPersonal();
  const contenedor = nodo(documento, "section");
  contenedor.className = "modulo-personal";
  contenedor.dataset.personalRptPublica = "";
  raiz.append(contenedor);
  let activa = true;
  let controlador = null;
  let consulta = Object.freeze({ q: "", limit: 25, offset: 0 });
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    controlador?.abort();
    retirar(raiz, contenedor);
  };
  registrarDesmontar?.(desmontar);
  const recargar = async (cambios = {}) => {
    if (!activa || !sigueMontada(raiz, contenedor)) return;
    const siguienteConsulta = { ...consulta, ...cambios };
    if (typeof siguienteConsulta.q !== "string" || siguienteConsulta.q !== siguienteConsulta.q.trim()
      || siguienteConsulta.q.length > 100 || !Number.isSafeInteger(siguienteConsulta.limit)
      || siguienteConsulta.limit < 1 || siguienteConsulta.limit > 100
      || !Number.isSafeInteger(siguienteConsulta.offset) || siguienteConsulta.offset < 0) {
      pintar(raiz, contenedor, { tipo: "error", mensaje: t("rpt_error"), consulta }, recargar, t);
      return;
    }
    controlador?.abort();
    const vuelo = new AbortController();
    controlador = vuelo;
    consulta = Object.freeze(siguienteConsulta);
    pintar(raiz, contenedor, { tipo: "cargando", consulta }, recargar, t);
    try {
      const pagina = await cliente.listar(consulta, { signal: vuelo.signal });
      if (activa && controlador === vuelo && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) {
        pintar(raiz, contenedor, { tipo: "disponible", pagina, consulta }, recargar, t);
      }
    } catch {
      if (activa && controlador === vuelo && sigueMontada(raiz, contenedor) && !vuelo.signal.aborted) {
        const mensaje = t("rpt_error");
        anunciar(mensaje, "error");
        pintar(raiz, contenedor, { tipo: "error", mensaje, consulta }, recargar, t);
      }
    } finally {
      if (controlador === vuelo) controlador = null;
    }
  };
  await recargar();
  return Object.freeze({ desmontar });
}
