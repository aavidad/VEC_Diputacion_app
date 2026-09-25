import { crearTraductorDocumentos } from "./i18n.js?v=20260925-documentos-web-v3";

// El servidor devuelve la clave del tipo documental catalogado, nunca su
// referencia opaca; un tipo sin rótulo se muestra como documento genérico.
const TIPOS = Object.freeze({
  "dietas.comision.borrador.v1": "tipo_comision",
  "dietas.justificante.v1": "tipo_justificante",
  "contratacion_temporal.borrador.v1": "tipo_contratacion",
});
const MIME = new Set(["application/pdf", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"]);
const CUSTODIAS = new Set(["vec", "externa"]);
const CONSERVACIONES = new Set(["aprobada", "provisional"]);
const referencia = (v) => typeof v === "string" && (/^ref:[0-9a-f]{64}$/u.test(v) && !/^ref:0{64}$/u.test(v)
  || /^[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u.test(v));
const huellaValida = (v) => typeof v === "string" && /^[0-9a-f]{64}$/iu.test(v);

function elemento(doc, tag, texto, clase) {
  const nodo = doc.createElement(tag);
  if (texto !== undefined) nodo.textContent = texto;
  if (clase) nodo.className = clase;
  return nodo;
}

export function validarRespuestaDocumentos(respuesta) {
  if (!respuesta || !["disponible", "vacio", "denegado"].includes(respuesta.estado)) throw new TypeError("respuesta documental inválida");
  if (respuesta.estado === "denegado") return Object.freeze({ estado: "denegado", documentos: [], siguienteCursor:"" });
  if (!Array.isArray(respuesta.documentos) || respuesta.documentos.length > 100 ||
      (respuesta.estado === "vacio" && respuesta.documentos.length !== 0) ||
      (!respuesta.documentos.length && respuesta.siguiente_cursor) ||
      (respuesta.siguiente_cursor && (typeof respuesta.siguiente_cursor !== "string" || respuesta.siguiente_cursor.length > 512 || /[\s\\/?%*]/u.test(respuesta.siguiente_cursor)))) throw new TypeError("lista documental inválida");
  const vistos = new Set();
  const documentos = respuesta.documentos.map((dato) => {
    const ref = dato?.ref;
    if (!referencia(ref) || vistos.has(ref) || !/^VEC-[0-9]{4}-[0-9]{1,12}$/u.test(dato.numero_vec) ||
        typeof dato.tipo !== "string" || !dato.tipo.trim() || dato.tipo.length > 80 ||
        !Number.isSafeInteger(dato.version) || dato.version < 1 ||
        dato.estado_firma !== "pendiente_firma" || !CUSTODIAS.has(dato.custodia) || !CONSERVACIONES.has(dato.conservacion) ||
        !huellaValida(dato.huella) || typeof dato.mime !== "string" ||
        !(/^[a-z0-9.+-]+\/[a-z0-9.+-]+$/u.test(dato.mime) || (dato.custodia === "externa" && dato.mime === ""))) throw new TypeError("documento inválido");
    vistos.add(ref);
    // B5 no declara firma sin atestación verificable y confirmación durable.
    const firma = dato.estado_firma;
    return Object.freeze({ ref, numero: dato.numero_vec, tipo: dato.tipo.trim(), version: dato.version,
      firma, huella: dato.huella.toLowerCase(), mime: dato.mime, custodia: dato.custodia, conservacion: dato.conservacion,
      // VEC solo entrega bytes que custodia; de un original externo muestra la huella.
      descargable: dato.custodia === "vec" && dato.descargable === true && MIME.has(dato.mime) });
  });
  return Object.freeze({ estado: documentos.length ? "disponible" : "vacio", documentos, siguienteCursor:respuesta.siguiente_cursor || "" });
}

export function validarArchivoDescarga(archivo) {
  if (!(archivo?.contenido instanceof Uint8Array) || archivo.contenido.length === 0 || archivo.contenido.length > 20 * 1024 * 1024 ||
      !MIME.has(archivo.tipo) || typeof archivo.nombre !== "string" ||
      !/^[A-Za-z0-9][A-Za-z0-9._-]{0,119}\.(pdf|docx)$/u.test(archivo.nombre) ||
      (archivo.tipo === "application/pdf" && !archivo.nombre.endsWith(".pdf")) ||
      (archivo.tipo !== "application/pdf" && !archivo.nombre.endsWith(".docx"))) throw new TypeError("original inválido");
  return archivo;
}

export function montarVistaDocumentos({ raiz, fuente, expedienteRef = "", anunciar = () => {}, registrarDesmontar } = {}) {
  const t = crearTraductorDocumentos();
  if (!raiz?.ownerDocument?.createElement || typeof anunciar !== "function" ||
      (fuente !== undefined && (typeof fuente.listar !== "function" || typeof fuente.descargar !== "function" || typeof fuente.seleccionarExpediente !== "function"))) throw new TypeError(t("error_vista"));
  const doc = raiz.ownerDocument;
  const contenedor = elemento(doc, "section", undefined, "modulo-documentos");
  const cabecera = elemento(doc, "header", undefined, "documentos-cabecera");
  const titulo = elemento(doc, "h2", t("titulo"));
  const ayuda = elemento(doc, "details", undefined, "documentos-ayuda");
  const ayudaBoton = elemento(doc, "summary", "?");
  ayudaBoton.setAttribute("aria-label", t("ayuda_etiqueta"));
  ayuda.append(ayudaBoton, elemento(doc, "p", t("aclaracion_firma")));
  cabecera.append(titulo, ayuda);
  const panel = elemento(doc, "section", undefined, "panel documentos-panel");
  // La vista no pide referencias al usuario: solo se abre desde un expediente,
  // que entrega la suya por navegación.
  const expediente = referencia(expedienteRef) ? expedienteRef : "";
  const estado = elemento(doc, "p", "", "documentos-estado-consulta");
  estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite");
  const listado = elemento(doc, "div", undefined, "documentos-listado");
  panel.append(estado, listado);
  contenedor.append(cabecera, panel);
  raiz.append(contenedor);

  let activa = true;
  let secuencia = 0;
  let controlador = null;
  let archivoEnCurso = false;
  let paginaDocumentos = [];
  let siguienteCursor = "";
  const urls = new Set();
  const mostrar = (clave) => { estado.textContent = t(clave); contenedor.dataset.estado = clave; };
  const limpiar = () => listado.replaceChildren();
  const valido = (numero, signal) => activa && numero === secuencia && !signal.aborted;

  async function descargar(item) {
    if (!fuente || archivoEnCurso || !item.descargable || !controlador) return;
    const numero = secuencia;
    const signal = controlador.signal;
    archivoEnCurso = true;
    mostrar("descargando");
    try {
      const archivo = validarArchivoDescarga(await fuente.descargar(item.ref, { version: item.version, mime: item.mime, huella: item.huella, signal }));
      if (!valido(numero, signal)) return;
      if (!doc.defaultView?.Blob || !doc.defaultView.URL?.createObjectURL) throw new TypeError("descarga no disponible");
      const url = doc.defaultView.URL.createObjectURL(new doc.defaultView.Blob([archivo.contenido], { type: archivo.tipo }));
      urls.add(url);
      const enlace = elemento(doc, "a");
      enlace.href = url;
      enlace.download = archivo.nombre;
      enlace.hidden = true;
      doc.body.append(enlace);
      try { enlace.click(); } finally { enlace.remove(); }
      doc.defaultView.setTimeout(() => { doc.defaultView.URL.revokeObjectURL(url); urls.delete(url); }, 0);
      mostrar("descarga_iniciada");
      anunciar(t("descarga_iniciada"), "informacion");
    } catch (error) {
      if (!valido(numero, signal)) return;
      mostrar(error?.codigo === "denegado" ? "denegado" : "descarga_error");
    } finally { archivoEnCurso = false; }
  }

  function pintar(documentos) {
    limpiar();
    if (!documentos.length) { mostrar("vacio"); return; }
    const region = elemento(doc, "div", undefined, "documentos-tabla");
    region.tabIndex = 0;
    region.setAttribute("role", "region");
    region.setAttribute("aria-label", t("tabla_documentos"));
    const tabla = elemento(doc, "table");
    const thead = elemento(doc, "thead");
    const cab = elemento(doc, "tr");
    for (const clave of ["col_documento", "col_tipo", "col_version", "col_firma", "col_accion"]) {
      const th = elemento(doc, "th", t(clave)); th.scope = "col"; cab.append(th);
    }
    thead.append(cab);
    const tbody = elemento(doc, "tbody");
    for (const item of documentos) {
      const tr = elemento(doc, "tr");
      for (const valor of [item.numero, t(Object.hasOwn(TIPOS, item.tipo) ? TIPOS[item.tipo] : "tipo_generico"), String(item.version)]) tr.append(elemento(doc, "td", valor));
      const firma = elemento(doc, "td");
      firma.append(elemento(doc, "span", t(`firma_${item.firma}`), `documentos-estado documentos-estado--${item.firma}`));
      // Plazo de conservación provisional: sin retención fijada en el almacén.
      if (item.conservacion === "provisional") firma.append(elemento(doc, "span", t("conservacion_provisional"), "documentos-conservacion-provisional"));
      tr.append(firma);
      const accion = elemento(doc, "td");
      if (item.custodia === "externa") {
        accion.append(elemento(doc, "span", t("custodia_externa"), "documentos-custodia-externa"), huellaVisible(item.huella));
      } else if (item.descargable) {
        const boton = elemento(doc, "button", t("descargar"), "boton-secundario documentos-descargar");
        boton.type = "button";
        boton.setAttribute("aria-label", t("descargar_de", { numero: item.numero }));
        boton.addEventListener("click", () => descargar(item));
        accion.append(boton);
      }
      tr.append(accion);
      tbody.append(tr);
    }
    tabla.append(thead, tbody);
    region.append(tabla);
    listado.append(region);
    if (siguienteCursor) {
      const siguiente = elemento(doc, "button", t("cargar_mas"), "boton-secundario documentos-mas");
      siguiente.type = "button";
      siguiente.addEventListener("click", () => { void consultar(true); });
      listado.append(siguiente);
    }
    mostrar("disponible");
  }

  // Huella abreviada visible; la completa, en un desplegable accesible por teclado.
  function huellaVisible(huella) {
    const detalle = elemento(doc, "details", undefined, "documentos-huella");
    const resumen = elemento(doc, "summary", t("huella_abreviada", { huella: `${huella.slice(0, 12)}…` }));
    const completa = elemento(doc, "code", huella, "documentos-huella-completa");
    detalle.append(resumen, completa);
    return detalle;
  }

  async function consultar(continuar = false) {
    controlador?.abort();
    controlador = new AbortController();
    const signal = controlador.signal;
    const numero = ++secuencia;
    if (!continuar) { paginaDocumentos = []; siguienteCursor = ""; limpiar(); }
    if (!fuente) { mostrar("no_configurado"); return; }
    if (!expediente) { mostrar("sin_expediente"); return; }
    mostrar("cargando");
    try {
      fuente.seleccionarExpediente(expediente);
      const respuesta = validarRespuestaDocumentos(await fuente.listar({ signal, cursor: continuar ? siguienteCursor : "" }));
      if (!valido(numero, signal)) return;
      if (respuesta.estado === "denegado") { paginaDocumentos = []; siguienteCursor = ""; limpiar(); mostrar("denegado"); return; }
      const vistos = new Set(paginaDocumentos.map((item) => item.ref));
      if (respuesta.documentos.some((item) => vistos.has(item.ref)) ||
          (continuar && respuesta.siguienteCursor === siguienteCursor)) throw new TypeError("paginación contradictoria");
      paginaDocumentos = [...paginaDocumentos, ...respuesta.documentos];
      siguienteCursor = respuesta.siguienteCursor;
      pintar(paginaDocumentos);
    } catch (error) {
      if (!valido(numero, signal)) return;
      paginaDocumentos = []; siguienteCursor = ""; limpiar();
      mostrar(error?.codigo === "denegado" ? "denegado" : "error");
    }
  }
  function desmontar() {
    if (!activa) return;
    activa = false;
    ++secuencia;
    controlador?.abort();
    for (const url of urls) doc.defaultView?.URL?.revokeObjectURL(url);
    urls.clear();
    contenedor.remove();
  }
  registrarDesmontar?.(desmontar);
  mostrar(!fuente ? "no_configurado" : expediente ? "cargando" : "sin_expediente");
  if (expediente && fuente) void consultar();
  return Object.freeze({ consultar: () => consultar(), desmontar });
}
