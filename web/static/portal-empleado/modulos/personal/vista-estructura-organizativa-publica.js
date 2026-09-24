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
function tabla(documento, estructura, t) {
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
  estructura.unidades.forEach((unidad) => {
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
function pintar(raiz, contenedor, estado, t) {
  if (!sigue(raiz, contenedor)) return;
  const documento = contenedor.ownerDocument;
  contenedor.replaceChildren();
  const cabecera = nodo(documento, "header");
  cabecera.className = "cabecera-vista";
  const contextual = ayuda(documento, estado.estructura?.fuente, t);
  cabecera.append(
    nodo(documento, "p", t("estructura_sobrelinea")),
    nodo(documento, "h2", t("estructura_titulo")),
    contextual.boton,
  );
  contenedor.append(cabecera, contextual.contenido);
  if (estado.tipo === "cargando") {
    const carga = nodo(documento, "p", t("estructura_cargando"));
    carga.setAttribute("role", "status");
    carga.setAttribute("aria-live", "polite");
    contenedor.append(carga);
    return;
  }
  if (estado.tipo === "error") {
    const error = nodo(documento, "p", t("estructura_error"));
    error.setAttribute("role", "alert");
    contenedor.append(error);
    return;
  }
  const estructura = estado.estructura;
  const aviso = nodo(documento, "p", t("estructura_aviso"));
  aviso.className = "estado-chip aviso";
  contenedor.append(
    aviso,
    nodo(documento, "p", formatearRecuentoEstructura(estructura.unidades.length)),
    tabla(documento, estructura, t),
  );
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
  const controlador = new AbortController();
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    controlador.abort();
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  pintar(raiz, contenedor, { tipo: "cargando" }, t);
  try {
    const estructura = await cliente.obtener({ signal: controlador.signal });
    if (activa && sigue(raiz, contenedor) && !controlador.signal.aborted) {
      pintar(raiz, contenedor, { tipo: "disponible", estructura }, t);
    }
  } catch {
    if (activa && sigue(raiz, contenedor) && !controlador.signal.aborted) {
      anunciar(t("estructura_error"), "error");
      pintar(raiz, contenedor, { tipo: "error" }, t);
    }
  }
  return Object.freeze({ desmontar });
}
