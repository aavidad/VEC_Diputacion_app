import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import {
  crearTraductorPersonal,
  formatearFechaEstructuraOrganizativa,
  formatearRecuentoEstructura,
} from "./i18n.js?v=20260925-personal-e10-v1";

function nodo(documento, etiqueta, texto = "") {
  const salida = documento.createElement(etiqueta);
  if (texto) salida.textContent = texto;
  return salida;
}
function sigue(raiz, contenedor) {
  return raiz.querySelector?.("[data-personal-estructura-organizativa-publica]") === contenedor;
}
const FILAS_POR_PAGINA = 10;
function tabla(documento, estructura, unidades, t, pagina) {
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
  unidades.slice(pagina * FILAS_POR_PAGINA, (pagina + 1) * FILAS_POR_PAGINA).forEach((unidad) => {
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
  contenido.append(nodo(documento, "p", t("estructura_ayuda")), nodo(documento, "p", t("estructura_aviso")));
  if (fuente) {
    const huella = nodo(documento, "p", t("estructura_huella", { huella: fuente.huella_sha256 }));
    huella.className = "rpt-huella";
    contenido.append(
      nodo(documento, "p", t("estructura_fuente", {
        ...fuente,
        actualizada_en: formatearFechaEstructuraOrganizativa(fuente.actualizada_en),
      })),
      huella,
    );
  }
  boton.addEventListener("click", () => {
    contenido.hidden = !contenido.hidden;
    boton.setAttribute("aria-expanded", String(!contenido.hidden));
  });
  return { boton, contenido };
}
const normalizarBusqueda = (valor) => valor.normalize("NFD").replace(/[\u0300-\u036f]/gu, "").toLocaleLowerCase(LOCALIZACION_ACTUAL);

function listado(documento, estructura, t, vigente) {
  const salida = nodo(documento, "div");
  const filtros = nodo(documento, "div");
  filtros.className = "barra-filtros";
  const texto = nodo(documento, "input");
  texto.type = "search";
  texto.value = "";
  texto.maxLength = 128;
  texto.autocomplete = "off";
  texto.dataset.personalEstructuraBuscar = "";
  const tipo = nodo(documento, "select");
  tipo.dataset.personalEstructuraTipo = "";
  for (const valor of ["", "delegacion", "centro", "puesto_responsabilidad"]) {
    const opcion = nodo(documento, "option", t(valor ? `estructura_tipo_${valor}` : "organizacion_allTypes"));
    opcion.value = valor;
    tipo.append(opcion);
  }
  tipo.value = "";
  for (const [clave, control] of [["estructura_buscar", texto], ["organizacion_filterType", tipo]]) {
    const etiqueta = nodo(documento, "label");
    etiqueta.className = "campo";
    etiqueta.append(nodo(documento, "span", t(clave)), control);
    filtros.append(etiqueta);
  }
  const limpiar = nodo(documento, "button", t("estructura_limpiar_filtros"));
  limpiar.type = "button";
  limpiar.dataset.personalEstructuraLimpiar = "";
  limpiar.disabled = true;
  filtros.append(limpiar);
  const recuento = nodo(documento, "p");
  recuento.dataset.personalEstructuraRecuento = "";
  recuento.setAttribute("role", "status");
  const marco = nodo(documento, "div");
  marco.className = "marco-tabla-paginado";
  const numero = new Intl.NumberFormat(LOCALIZACION_ACTUAL);
  // Las etiquetas de los padres proceden del catálogo completo, aunque el
  // filtro deje visible solo a sus hijos.
  const padres = new Map(estructura.unidades.map((unidad) => [unidad.clave, unidad.etiqueta]));
  let pagina = 0;
  let unidades = estructura.unidades;
  const actualizarTabla = () => {
    if (!vigente()) return;
    const focoPaginacion = ["anterior", "siguiente"].find((direccion) =>
      documento.activeElement === marco.querySelector(`[data-personal-estructura-${direccion}]`));
    const paginas = Math.ceil(unidades.length / FILAS_POR_PAGINA);
    const navegacion = nodo(documento, "nav");
    navegacion.className = "paginacion-marco";
    navegacion.setAttribute("aria-label", t("estructura_tabla"));
    const anterior = nodo(documento, "button", t("catalogo_anterior"));
    anterior.type = "button";
    anterior.dataset.personalEstructuraAnterior = "";
    anterior.disabled = pagina === 0;
    anterior.addEventListener("click", () => {
      if (!vigente() || pagina === 0) return;
      pagina -= 1;
      actualizarTabla();
    });
    const siguiente = nodo(documento, "button", t("catalogo_siguiente"));
    siguiente.type = "button";
    siguiente.dataset.personalEstructuraSiguiente = "";
    siguiente.disabled = pagina >= paginas - 1;
    siguiente.addEventListener("click", () => {
      if (!vigente() || pagina >= paginas - 1) return;
      pagina += 1;
      actualizarTabla();
    });
    const posicion = nodo(documento, "span", `${numero.format(unidades.length ? pagina + 1 : 0)} / ${numero.format(paginas)}`);
    posicion.setAttribute("aria-live", "polite");
    navegacion.append(anterior, posicion, siguiente);
    if (unidades.length) {
      marco.replaceChildren(tabla(documento, estructura, unidades, t, pagina), navegacion);
    } else {
      const vacio = nodo(documento, "p", t("organizacion_empty"));
      vacio.setAttribute("role", "status");
      marco.replaceChildren(vacio, navegacion);
    }
    recuento.textContent = t("organizacion_count", { visible: numero.format(unidades.length), total: numero.format(estructura.unidades.length) });
    if (focoPaginacion) {
      const mismaDireccion = focoPaginacion === "anterior" ? anterior : siguiente;
      const alternativa = focoPaginacion === "anterior" ? siguiente : anterior;
      (!mismaDireccion.disabled ? mismaDireccion : alternativa).focus?.();
    }
  };
  const filtrar = () => {
    if (!vigente()) return;
    const busqueda = normalizarBusqueda(texto.value.trim());
    limpiar.disabled = !texto.value && !tipo.value;
    unidades = estructura.unidades.filter((unidad) =>
      (!tipo.value || unidad.tipo === tipo.value) && (!busqueda ||
        [unidad.etiqueta, padres.get(unidad.adscripcion_clave) ?? ""].some((valor) => normalizarBusqueda(valor).includes(busqueda))));
    pagina = 0;
    // Solo cambia el resultado: los controles, el foco, la selección y la
    // ayuda siguen siendo los mismos nodos durante la escritura.
    actualizarTabla();
  };
  texto.addEventListener("input", filtrar);
  tipo.addEventListener("change", filtrar);
  limpiar.addEventListener("click", () => {
    if (!vigente()) return;
    texto.value = "";
    tipo.value = "";
    filtrar();
    texto.focus();
  });
  salida.append(filtros, recuento, marco);
  actualizarTabla();
  return salida;
}

function pintar(raiz, contenedor, estado, t, reintentar) {
  if (!sigue(raiz, contenedor)) return;
  const documento = contenedor.ownerDocument;
  const botonAnterior = contenedor.querySelector("[data-personal-estructura-ayuda]");
  const contenidoAnterior = contenedor.querySelector("[data-personal-estructura-ayuda-contenido]");
  const reintentoAnterior = contenedor.querySelector("[data-personal-estructura-reintentar]");
  const cargaAnterior = contenedor.querySelector("[data-personal-estructura-cargando]");
  const ayudaAbierta = contenidoAnterior ? !contenidoAnterior.hidden : false;
  const focoAyuda = documento.activeElement === botonAnterior;
  const focoRecuperacion = documento.activeElement === reintentoAnterior || documento.activeElement === cargaAnterior;
  contenedor.replaceChildren();
  const cabecera = nodo(documento, "header");
  cabecera.className = "cabecera-vista";
  const titulo = nodo(documento, "h2", t("estructura_titulo"));
  const contextual = ayuda(documento, estado.estructura?.fuente, t);
  contextual.contenido.hidden = !ayudaAbierta;
  contextual.boton.setAttribute("aria-expanded", String(ayudaAbierta));
  cabecera.append(
    titulo,
    contextual.boton,
  );
  contenedor.append(cabecera, contextual.contenido);
  const restaurarFoco = () => {
    if (focoAyuda) {
      contextual.boton.focus?.();
      return;
    }
  };
  if (estado.tipo === "cargando") {
    const carga = nodo(documento, "p", t("estructura_cargando"));
    carga.dataset.personalEstructuraCargando = "";
    carga.setAttribute("role", "status");
    carga.setAttribute("aria-live", "polite");
    carga.setAttribute("tabindex", "-1");
    contenedor.append(carga);
    restaurarFoco();
    if (focoRecuperacion) carga.focus?.();
    return;
  }
  if (estado.tipo === "error") {
    const error = nodo(documento, "p", t("estructura_error"));
    error.setAttribute("role", "alert");
    const boton = nodo(documento, "button", t("ficha_reintentar"));
    boton.type = "button";
    boton.className = "boton-secundario";
    boton.dataset.personalEstructuraReintentar = "";
    boton.addEventListener("click", reintentar);
    contenedor.append(error, boton);
    restaurarFoco();
    if (focoRecuperacion) boton.focus?.();
    return;
  }
  const estructura = estado.estructura;
  // La advertencia completa vive tras «?»; en pantalla, solo la pastilla.
  const aviso = nodo(documento, "p");
  const pastilla = nodo(documento, "span", t("estructura_en_preparacion"));
  pastilla.className = "estado-chip";
  pastilla.dataset.personalEstructuraAviso = "";
  aviso.append(pastilla);
  contenedor.append(aviso, nodo(documento, "p", formatearRecuentoEstructura(estructura.unidades.length)),
    listado(documento, estructura, t, () => sigue(raiz, contenedor)));
  restaurarFoco();
  if (focoRecuperacion) {
    titulo.setAttribute("tabindex", "-1");
    titulo.focus?.();
  }
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
  let estado = { tipo: "cargando" };
  let controlador;
  let consulta = 0;
  const repintar = () => pintar(raiz, contenedor, estado, t, consultar);
  const consultar = async () => {
    if (!activa || !sigue(raiz, contenedor) || (controlador && estado.tipo === "cargando")) return;
    const turno = ++consulta;
    controlador?.abort();
    const solicitud = new AbortController();
    controlador = solicitud;
    estado = { tipo: "cargando" };
    repintar();
    try {
      const estructura = await cliente.obtener({ signal: solicitud.signal });
      if (activa && turno === consulta && sigue(raiz, contenedor) && !solicitud.signal.aborted) {
        estado = { tipo: "disponible", estructura };
        repintar();
      }
    } catch {
      if (activa && turno === consulta && sigue(raiz, contenedor) && !solicitud.signal.aborted) {
        anunciar(t("estructura_error"), "error");
        estado = { tipo: "error" };
        repintar();
      }
    }
  };
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    controlador?.abort();
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  await consultar();
  return Object.freeze({ desmontar });
}
