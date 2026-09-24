import { crearTraductorDocumentos } from "./i18n.js?v=20260924-f2-web2";

const ESTADOS = new Set(["no_configurado", "cargando", "disponible", "vacio", "denegado", "error"]);
const FIRMA = new Set(["borrador", "pendiente_firma", "firmado"]);
const ACCION_DESCARGA = "documentos.descargar_original";
const FORMATOS_DESCARGA = Object.freeze({
  "application/pdf": [".pdf"],
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document": [".docx"],
  "image/png": [".png"],
  "image/jpeg": [".jpg", ".jpeg"],
});
const fechaES = new Intl.DateTimeFormat("es-ES", { day: "2-digit", month: "2-digit", year: "numeric" });

function nodo(documento, etiqueta, texto = "", clase = "") {
  const elemento = documento.createElement(etiqueta);
  if (texto) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}

function panel(documento, titulo, nota, clase = "") {
  const seccion = nodo(documento, "section", "", `panel ${clase}`.trim());
  const cabecera = nodo(documento, "header", "", "cabecera-panel");
  const titulos = nodo(documento, "div");
  titulos.append(nodo(documento, "h3", titulo), nodo(documento, "p", nota));
  cabecera.append(titulos);
  const cuerpo = nodo(documento, "div", "", "cuerpo-panel");
  seccion.append(cabecera, cuerpo);
  return { seccion, cuerpo };
}

function textoLimitado(valor, maximo) {
  return typeof valor === "string" && valor.trim() && valor.length <= maximo ? valor.trim() : null;
}

function fechaValida(valor) {
  if (valor === undefined || valor === null || valor === "") return null;
  const fecha = new Date(valor);
  if (Number.isNaN(fecha.getTime())) throw new TypeError("fecha documental inválida");
  return fecha;
}

function permisoDescargaExacto(permiso, ref, version) {
  return permiso !== null && typeof permiso === "object" && !Array.isArray(permiso)
    && Object.hasOwn(permiso, "accion") && permiso.accion === ACCION_DESCARGA
    && Object.hasOwn(permiso, "recurso_ref") && permiso.recurso_ref === ref
    && Object.hasOwn(permiso, "version") && permiso.version === version
    && Object.hasOwn(permiso, "concedido") && permiso.concedido === true;
}

function permisoDenegado() {
  const error = new Error("permiso de descarga no confirmado");
  error.codigo = "permiso_denegado";
  return error;
}

/**
 * Contrato de lectura: {estado, origen, actualizado_en?, documentos: [{ref,
 * titulo, tipo, version, estado_firma, firma?, huella?, custodia?, antivirus?,
 * descargable?, permiso_descarga?}]}. permiso_descarga lo emite la fuente para
 * acción, referencia y versión exactas. Una marca "firmado" sola no acredita
 * firma; custodia exige recibo y descarga exige antivirus limpio y permiso.
 */
export function validarRespuestaDocumentos(respuesta) {
  if (!respuesta || typeof respuesta !== "object" || !["disponible", "vacio", "denegado"].includes(respuesta.estado)) {
    throw new TypeError("respuesta documental inválida");
  }
  if (respuesta.estado === "denegado") return { estado: "denegado", documentos: [], origen: "", actualizado: null };
  const origen = textoLimitado(respuesta.origen, 120);
  if (!origen || !Array.isArray(respuesta.documentos) || respuesta.documentos.length > 100) throw new TypeError("fuente documental inválida");
  const referencias = new Set();
  const documentos = respuesta.documentos.map((item) => {
    const ref = textoLimitado(item?.ref, 120);
    const titulo = textoLimitado(item?.titulo, 180);
    const tipo = textoLimitado(item?.tipo, 80);
    if (!ref || referencias.has(ref) || !titulo || !tipo || !Number.isSafeInteger(item.version) || item.version < 1 || !FIRMA.has(item.estado_firma)) {
      throw new TypeError("documento de consulta inválido");
    }
    referencias.add(ref);
    const firmaValidada = item.estado_firma === "firmado" && item.firma?.validada === true && !!textoLimitado(item.firma.referencia, 120);
    const custodiaConfirmada = item.custodia?.confirmada === true && !!textoLimitado(item.custodia.recibo_ref, 120);
    const huella = typeof item.huella === "string" && /^[0-9a-f]{64}$/iu.test(item.huella) ? item.huella.toLowerCase() : null;
    return Object.freeze({
      ref, titulo, tipo, version: item.version,
      fecha: fechaValida(item.fecha),
      firma: firmaValidada ? "firmado" : item.estado_firma === "firmado" ? "sin_acreditar" : item.estado_firma,
      firmaRef: firmaValidada ? item.firma.referencia.trim() : null,
      custodia: custodiaConfirmada ? item.custodia.recibo_ref.trim() : null,
      huella,
      descargable: item.descargable === true && item.antivirus === "limpio"
        && permisoDescargaExacto(item.permiso_descarga, ref, item.version),
    });
  });
  if (respuesta.estado === "vacio" && documentos.length) throw new TypeError("consulta documental contradictoria");
  return Object.freeze({
    estado: documentos.length ? "disponible" : "vacio",
    documentos,
    origen,
    actualizado: fechaValida(respuesta.actualizado_en),
  });
}

/** El conector entrega bytes del original; no se aceptan URL ni HTML inyectados. */
export function validarArchivoDescarga(archivo) {
  const nombre = textoLimitado(archivo?.nombre, 160);
  const extensiones = Object.hasOwn(FORMATOS_DESCARGA, archivo?.tipo) ? FORMATOS_DESCARGA[archivo.tipo] : [];
  if (!archivo || typeof archivo !== "object" || !(archivo.contenido instanceof Uint8Array)
    || archivo.contenido.byteLength < 1 || archivo.contenido.byteLength > 20 * 1024 * 1024
    || !nombre || /[/\\\x00-\x1f\u202a-\u202e\u2066-\u2069]/iu.test(nombre)
    || !extensiones.some((extension) => nombre.toLocaleLowerCase("es").endsWith(extension))) {
    throw new TypeError("archivo documental inválido");
  }
  return { contenido: archivo.contenido, nombre, tipo: archivo.tipo };
}

/** Revalida en la fuente antes de pedir bytes; el conector debe hacerlo otra vez al servirlos. */
export async function obtenerArchivoDescargaAutorizada(fuente, item, { signal, vigente = () => true } = {}) {
  if (!item?.descargable || typeof fuente?.confirmarPermisoDescarga !== "function"
    || typeof fuente?.descargar !== "function" || typeof vigente !== "function" || signal?.aborted || !vigente()) {
    throw permisoDenegado();
  }
  const permiso = await fuente.confirmarPermisoDescarga(item.ref, { version: item.version, signal });
  if (signal?.aborted || !vigente() || !permisoDescargaExacto(permiso, item.ref, item.version)) throw permisoDenegado();
  return validarArchivoDescarga(await fuente.descargar(item.ref, { version: item.version, signal }));
}

function filaDato(documento, titulo, valor) {
  const fila = nodo(documento, "div");
  fila.append(nodo(documento, "dt", titulo), nodo(documento, "dd", valor));
  return fila;
}

/**
 * fuente.listar({signal}) consulta documentos ya autorizados para el actor.
 * fuente.confirmarPermisoDescarga(ref,{version,signal}) obtiene decisión fresca.
 * fuente.descargar(ref, {version,signal}) devuelve {contenido: Uint8Array, nombre, tipo}
 * del original autorizado. El módulo crea una descarga local de esos bytes.
 * El conector debe revalidar autorización al servir el original; esta vista
 * nunca crea permisos, originales, firmas ni recibos de custodia.
 */
export function montarVistaDocumentos({ raiz, anunciar = () => {}, registrarDesmontar, fuente } = {}) {
  const t = crearTraductorDocumentos();
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof anunciar !== "function"
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")
    || (fuente !== undefined && typeof fuente?.listar !== "function")) throw new TypeError(t("error_vista"));
  const documento = raiz.ownerDocument;
  let activa = true;
  let controlador = null;
  let secuencia = 0;
  let estado = "no_configurado";
  let documentos = [];
  let seleccionado = null;
  let filtro = "";
  let origen = "";
  let actualizado = null;
  let descargando = false;
  const urls = new Set();

  const contenedor = nodo(documento, "section", "", "modulo-documentos");
  contenedor.dataset.documentos = "";
  const cabecera = nodo(documento, "header", "", "documentos-cabecera");
  cabecera.append(nodo(documento, "p", t("sobrelinea"), "sobrelinea"), nodo(documento, "h2", t("titulo")), nodo(documento, "p", t("descripcion")));
  const estadoVisible = nodo(documento, "div", "", "documentos-aviso");
  estadoVisible.setAttribute("role", "status");
  estadoVisible.setAttribute("aria-live", "polite");
  cabecera.append(estadoVisible);
  const indicadores = nodo(documento, "div", "", "documentos-indicadores");
  const principal = nodo(documento, "div", "", "documentos-principal");
  const listado = panel(documento, t("repositorio_titulo"), t("repositorio_nota"), "documentos-listado");
  const ficha = panel(documento, t("ficha"), t("ficha_nota"), "documentos-ficha");
  principal.append(listado.seccion, ficha.seccion);

  const filtros = nodo(documento, "form", "", "documentos-filtros");
  const etiqueta = nodo(documento, "label", t("filtro"));
  const entrada = nodo(documento, "input");
  entrada.type = "search";
  entrada.name = "filtro";
  entrada.placeholder = t("filtro_placeholder");
  etiqueta.append(entrada);
  const aplicar = nodo(documento, "button", t("aplicar_filtro"), "boton-primario");
  aplicar.type = "submit";
  filtros.append(etiqueta, aplicar);
  const listadoContenido = nodo(documento, "div", "", "documentos-listado-contenido");
  const tablaAyuda = nodo(documento, "p", t("tabla_desplazar"), "documentos-tabla-ayuda");
  listado.cuerpo.append(filtros, tablaAyuda, listadoContenido);

  const fichaTitulo = nodo(documento, "h4", "", "documentos-titulo-ficha");
  fichaTitulo.tabIndex = -1;
  const fichaDescripcion = nodo(documento, "p", "", "documentos-ficha-aviso");
  const metadatos = nodo(documento, "dl", "", "documentos-metadatos");
  const acciones = nodo(documento, "div", "", "documentos-acciones");
  const accionesNoDisponibles = [["generar_version", "generar_motivo"], ["subir_original", "subir_motivo"], ["firmar", "firmar_motivo"], ["verificar", "verificar_motivo"], ["enviar", "enviar_motivo"]];
  accionesNoDisponibles.forEach(([clave, motivo]) => {
    const boton = nodo(documento, "button", t(clave), "documentos-accion");
    boton.type = "button";
    boton.disabled = true;
    boton.title = t(motivo);
    boton.setAttribute("aria-label", `${t(clave)}. ${t(motivo)}`);
    acciones.append(boton);
  });
  const descargar = nodo(documento, "button", t("descargar"), "documentos-accion documentos-descargar");
  descargar.type = "button";
  acciones.append(descargar);
  const descargaEstado = nodo(documento, "p", "", "documentos-descarga-estado");
  descargaEstado.setAttribute("role", "status");
  descargaEstado.setAttribute("aria-live", "polite");
  ficha.cuerpo.append(fichaTitulo, fichaDescripcion, metadatos, nodo(documento, "h5", t("acciones_titulo")), acciones, descargaEstado);

  const ayuda = nodo(documento, "details", "", "panel documentos-ayuda");
  const pregunta = nodo(documento, "summary", t("ayuda_titulo"));
  pregunta.setAttribute("aria-label", t("ayuda_etiqueta"));
  const ayudaContenido = nodo(documento, "div", "", "cuerpo-panel");
  ayudaContenido.append(nodo(documento, "p", t("aclaracion_firma")), nodo(documento, "h4", t("tipos_previstos")));
  const tipos = nodo(documento, "ul", "", "documentos-tipos");
  ["tipo_informe", "tipo_resolucion", "tipo_rc", "tipo_diligencia"].forEach((clave) => tipos.append(nodo(documento, "li", t(clave))));
  ayudaContenido.append(tipos, nodo(documento, "h4", t("circuito_titulo")));
  const pasos = nodo(documento, "ol", "", "documentos-pasos");
  ["paso_1", "paso_2", "paso_3", "paso_4", "paso_5"].forEach((clave) => pasos.append(nodo(documento, "li", t(clave))));
  ayudaContenido.append(pasos);
  ayuda.append(pregunta, ayudaContenido);
  contenedor.append(cabecera, indicadores, principal, ayuda);
  raiz.append(contenedor);

  function estadoFirma(item) { return t(`firma_${item.firma}`); }
  function disponibles() {
    return documentos.filter((item) => !filtro || `${item.titulo} ${item.tipo}`.toLocaleLowerCase("es").includes(filtro.toLocaleLowerCase("es")));
  }
  function pintarEstado() {
    contenedor.dataset.estado = estado;
    estadoVisible.replaceChildren(nodo(documento, "strong", t(`estado_${estado}`)), nodo(documento, "span", t(`explicacion_${estado}`)));
    if (origen && ["disponible", "vacio"].includes(estado)) {
      estadoVisible.append(nodo(documento, "small", `${t("fuente")}: ${origen}${actualizado ? ` · ${t("actualizado")}: ${fechaES.format(actualizado)}` : ""}`));
    }
  }
  function pintarIndicadores() {
    const consultar = estado === "disponible" || estado === "vacio";
    const resumen = [
      ["borrador", consultar ? String(documentos.filter((item) => item.firma === "borrador").length) : t("valor_sin_fuente"), "nota_borrador"],
      ["firmado", consultar ? String(documentos.filter((item) => item.firma === "firmado").length) : t("valor_sin_fuente"), "nota_firmado"],
      ["descarga", consultar ? String(documentos.filter((item) => item.descargable && typeof fuente?.confirmarPermisoDescarga === "function" && typeof fuente?.descargar === "function").length) : t("valor_sin_fuente"), "nota_descarga"],
      ["custodia", consultar ? String(documentos.filter((item) => item.custodia).length) : t("valor_sin_fuente"), "nota_custodia"],
    ];
    indicadores.replaceChildren(...resumen.map(([titulo, valor, nota]) => {
      const tarjeta = nodo(documento, "article", "", "documentos-indicador");
      tarjeta.append(nodo(documento, "span", t(titulo)), nodo(documento, "strong", valor), nodo(documento, "small", t(nota)));
      return tarjeta;
    }));
  }
  function pintarListado() {
    entrada.disabled = estado !== "disponible";
    aplicar.disabled = estado !== "disponible";
    listadoContenido.replaceChildren();
    if (estado === "error" && fuente) {
      const reintentar = nodo(documento, "button", t("reintentar"), "boton-secundario documentos-reintentar");
      reintentar.type = "button";
      reintentar.addEventListener("click", consultar);
      listadoContenido.append(reintentar);
    }
    if (estado !== "disponible" || !disponibles().length) {
      const vacio = nodo(documento, "div", "", "documentos-vacio");
      vacio.setAttribute("role", "status");
      vacio.append(nodo(documento, "strong", t(estado === "disponible" ? "sin_resultados" : `estado_${estado}`)), nodo(documento, "p", t(estado === "disponible" ? "filtro_sin_resultados" : `explicacion_${estado}`)));
      listadoContenido.append(vacio);
      return;
    }
    const region = nodo(documento, "div", "", "documentos-tabla");
    region.tabIndex = 0;
    region.setAttribute("role", "region");
    region.setAttribute("aria-label", t("tabla_documentos"));
    const tabla = nodo(documento, "table");
    tabla.append(nodo(documento, "caption", t("tabla_documentos")));
    const cabecera = nodo(documento, "thead");
    const encabezado = nodo(documento, "tr");
    ["col_documento", "col_tipo", "col_version", "col_firma", "col_accion"].forEach((clave) => {
      const th = nodo(documento, "th", t(clave));
      th.scope = "col";
      encabezado.append(th);
    });
    cabecera.append(encabezado);
    const cuerpo = nodo(documento, "tbody");
    disponibles().forEach((item) => {
      const fila = nodo(documento, "tr");
      if (seleccionado?.ref === item.ref) fila.dataset.seleccionada = "true";
      fila.append(nodo(documento, "td", item.titulo), nodo(documento, "td", item.tipo), nodo(documento, "td", String(item.version)));
      const firma = nodo(documento, "td");
      firma.append(nodo(documento, "span", estadoFirma(item), `documentos-estado documentos-estado--${item.firma}`));
      fila.append(firma);
      const accion = nodo(documento, "td");
      const ver = nodo(documento, "button", t("ver_ficha"), "documentos-ver");
      ver.type = "button";
      ver.setAttribute("aria-label", t("ver_ficha_de", { titulo: item.titulo }));
      ver.addEventListener("click", () => {
        seleccionado = item;
        descargaEstado.textContent = "";
        pintarListado();
        pintarFicha();
        fichaTitulo.focus();
        anunciar(t("ficha_seleccionada", { titulo: item.titulo }), "informacion");
      });
      accion.append(ver);
      fila.append(accion);
      cuerpo.append(fila);
    });
    tabla.append(cabecera, cuerpo);
    region.append(tabla);
    listadoContenido.append(region);
  }
  function pintarFicha() {
    fichaTitulo.textContent = seleccionado ? seleccionado.titulo : t("ficha_vacia");
    fichaDescripcion.textContent = seleccionado ? t("ficha_origen") : estado === "disponible" ? t("ficha_sin_seleccion") : t("ficha_sin_original");
    metadatos.replaceChildren();
    if (seleccionado) {
      [
        ["tipo", seleccionado.tipo], ["version", String(seleccionado.version)],
        ["fecha", seleccionado.fecha ? fechaES.format(seleccionado.fecha) : t("dato_no_disponible")],
        ["firma", estadoFirma(seleccionado)],
        ["firma_ref", seleccionado.firmaRef || t("dato_no_disponible")],
        ["huella", seleccionado.huella || t("dato_no_disponible")],
        ["conservacion", seleccionado.custodia ? t("custodia_confirmada") : t("custodia_sin_fuente")],
        ["custodia_ref", seleccionado.custodia || t("dato_no_disponible")],
      ].forEach(([clave, valor]) => metadatos.append(filaDato(documento, t(clave), valor)));
    } else {
      [["version", "version_sin_fuente"], ["firma", "firma_sin_evidencia"], ["huella", "huella_sin_fuente"], ["conservacion", "custodia_sin_fuente"]].forEach(([clave, valor]) => metadatos.append(filaDato(documento, t(clave), t(valor))));
    }
    descargar.disabled = !seleccionado?.descargable || typeof fuente?.confirmarPermisoDescarga !== "function"
      || typeof fuente?.descargar !== "function" || descargando;
    const motivo = seleccionado?.descargable ? t("descarga_sin_conector") : t("descargar_motivo");
    descargar.title = descargar.disabled ? motivo : t("descarga_autorizada");
    descargar.setAttribute("aria-label", `${t("descargar")}. ${descargar.title}`);
  }
  function pintar() {
    if (!activa || !ESTADOS.has(estado)) return;
    pintarEstado();
    pintarIndicadores();
    pintarListado();
    pintarFicha();
  }
  async function consultar() {
    if (!activa || !fuente) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++secuencia;
    estado = "cargando";
    documentos = [];
    seleccionado = null;
    filtro = "";
    entrada.value = "";
    origen = "";
    actualizado = null;
    descargaEstado.textContent = "";
    pintar();
    try {
      const respuesta = validarRespuestaDocumentos(await fuente.listar({ signal: controlador.signal }));
      if (!activa || actual !== secuencia || controlador.signal.aborted) return;
      ({ estado, documentos, origen, actualizado } = respuesta);
      seleccionado = documentos[0] ?? null;
    } catch {
      if (!activa || actual !== secuencia || controlador.signal.aborted) return;
      estado = "error";
    }
    pintar();
  }
  filtros.addEventListener("submit", (evento) => {
    evento.preventDefault();
    if (estado !== "disponible") return;
    filtro = entrada.value.trim();
    const visibles = disponibles();
    if (!visibles.includes(seleccionado)) {
      seleccionado = visibles[0] ?? null;
      descargaEstado.textContent = "";
    }
    pintarListado();
    pintarFicha();
    anunciar(filtro ? t("filtro_aplicado") : t("filtro_eliminado"), "informacion");
  });
  descargar.addEventListener("click", async () => {
    const item = seleccionado;
    if (!activa || descargar.disabled || !item?.descargable
      || typeof fuente?.confirmarPermisoDescarga !== "function" || typeof fuente?.descargar !== "function") return;
    const ref = item.ref;
    const actual = secuencia;
    const signal = controlador?.signal;
    const vigente = () => activa && actual === secuencia && !signal?.aborted
      && seleccionado?.ref === ref && seleccionado?.version === item.version;
    descargando = true;
    descargar.disabled = true;
    descargaEstado.textContent = t("descarga_en_curso");
    try {
      const archivo = await obtenerArchivoDescargaAutorizada(fuente, item, { signal, vigente });
      if (!vigente()) return;
      const entorno = documento.defaultView;
      if (!documento.body || !entorno?.Blob || !entorno.URL?.createObjectURL) throw new TypeError("descarga no disponible");
      const url = entorno.URL.createObjectURL(new entorno.Blob([archivo.contenido], { type: archivo.tipo }));
      urls.add(url);
      const enlace = nodo(documento, "a");
      enlace.href = url;
      enlace.download = archivo.nombre;
      enlace.hidden = true;
      documento.body.append(enlace);
      try { enlace.click(); } finally { enlace.remove(); }
      entorno.setTimeout(() => { entorno.URL.revokeObjectURL(url); urls.delete(url); }, 0);
      descargaEstado.textContent = t("descarga_iniciada");
      anunciar(t("descarga_iniciada"), "informacion");
    } catch (error) {
      if (!vigente()) return;
      const mensaje = t(error?.codigo === "permiso_denegado" ? "descargar_motivo" : "descarga_error");
      descargaEstado.textContent = mensaje;
      anunciar(mensaje, "error");
    } finally {
      descargando = false;
      if (activa) pintarFicha();
    }
  });
  function desmontar() {
    if (!activa) return;
    activa = false;
    ++secuencia;
    controlador?.abort();
    for (const url of urls) documento.defaultView?.URL?.revokeObjectURL(url);
    urls.clear();
    contenedor.remove();
  }
  registrarDesmontar?.(desmontar);
  pintar();
  if (fuente) void consultar();
  return Object.freeze({ desmontar, consultar });
}
