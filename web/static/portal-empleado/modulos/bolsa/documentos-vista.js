import { cargarTextos } from "../../../comun/textos.js";
import { crearClienteDocumentosBolsa, validarCorrespondenciaReciboDocumento } from "./documentos-api.js?v=20261013-documentos-bolsa-v1";

const PARAMETROS = Object.freeze({ bolsa: "bolsa-doc-bolsa", tipo: "bolsa-doc-tipo", texto: "bolsa-doc-texto" });

function nodo(doc, tag, texto, clase) {
  const elemento = doc.createElement(tag);
  if (texto !== undefined) elemento.textContent = texto;
  if (clase) elemento.className = clase;
  return elemento;
}

function boton(doc, texto, accion, clase = "boton-secundario") {
  const elemento = nodo(doc, "button", texto, clase);
  elemento.type = "button";
  elemento.dataset.accion = accion;
  return elemento;
}

function filtrosDeURL(ventana, bolsas) {
  const parametros = new URLSearchParams(ventana?.location?.search ?? "");
  const bolsaSolicitada = parametros.get(PARAMETROS.bolsa);
  return {
    bolsa: bolsas.some((item) => item.bolsa_ref === bolsaSolicitada) ? bolsaSolicitada : bolsas[0]?.bolsa_ref ?? "",
    tipo: parametros.get(PARAMETROS.tipo) === "constitucion" ? "constitucion" : "",
    texto: (parametros.get(PARAMETROS.texto) ?? "").slice(0, 120),
  };
}

function guardarFiltros(ventana, filtros) {
  if (!ventana?.history?.replaceState || !ventana?.location) return;
  const url = new URL(ventana.location.href);
  for (const clave of Object.values(PARAMETROS)) url.searchParams.delete(clave);
  if (filtros.bolsa) url.searchParams.set(PARAMETROS.bolsa, filtros.bolsa);
  if (filtros.tipo) url.searchParams.set(PARAMETROS.tipo, filtros.tipo);
  if (filtros.texto) url.searchParams.set(PARAMETROS.texto, filtros.texto);
  ventana.history.replaceState(ventana.history.state, "", `${url.pathname}${url.search}${url.hash}`);
}

export async function montarVistaDocumentosBolsa({ raiz, bolsas = [], cliente = crearClienteDocumentosBolsa(),
  anunciar = () => {}, registrarDesmontar, abrirCandidatos } = {}) {
  if (!raiz?.ownerDocument || !Array.isArray(bolsas) || typeof cliente.listar !== "function"
      || typeof cliente.recibir !== "function" || typeof anunciar !== "function") throw new TypeError("montaje documental no válido");
  const doc = raiz.ownerDocument;
  const ventana = doc.defaultView;
  let activo = true;
  let consulta = null;
  let recibo = null;
  let secuencia = 0;
  let secuenciaRecibo = 0;
  let textos;
  let filtros = filtrosDeURL(ventana, bolsas);
  let cursores = [""];
  let pagina = 0;
  let resultado = null;
  let botonOrigenRecibo = null;
  const contenedor = nodo(doc, "section", undefined, "panel");
  contenedor.dataset.bolsaDocumentos = "";
  raiz.append(contenedor);

  const desmontar = () => {
    if (!activo) return;
    activo = false;
    secuencia++;
    secuenciaRecibo++;
    consulta?.abort();
    recibo?.abort();
    contenedor.remove();
  };
  registrarDesmontar?.(desmontar);

  async function cargarCatalogo() {
    // El cargador común aplica el idioma de respaldo; el shell ofrece Reintentar si falla también.
    textos = await cargarTextos("bolsa-documentos");
    if (!activo) return;
    montarEstructura();
    if (filtros.bolsa) void cargarPagina();
    else estado("sin_bolsas");
  }

  const t = (clave, variables) => textos.traducir(`general.${clave}`, variables);
  function estado(clave, variables) {
    const zona = contenedor.querySelector("[data-documentos-estado]");
    if (zona) zona.textContent = t(clave, variables);
  }

  function montarEstructura() {
    const cabecera = nodo(doc, "header", undefined, "cabecera-panel");
    const titulo = nodo(doc, "h2", t("titulo"));
    const ayuda = boton(doc, "?", "ayuda");
    ayuda.setAttribute("aria-label", t("ayuda_abrir"));
    ayuda.setAttribute("aria-expanded", "false");
    ayuda.setAttribute("aria-controls", "bolsa-documentos-ayuda");
    cabecera.append(titulo, ayuda);
    const cuerpo = nodo(doc, "div", undefined, "cuerpo-panel");
    const textoAyuda = nodo(doc, "div", undefined, "panel");
    textoAyuda.id = "bolsa-documentos-ayuda";
    textoAyuda.hidden = true;
    const contenidoAyuda = nodo(doc, "div", undefined, "cuerpo-panel");
    contenidoAyuda.append(nodo(doc, "p", t("ayuda_1")), nodo(doc, "p", t("ayuda_2")));
    textoAyuda.append(contenidoAyuda);
    const formulario = nodo(doc, "form", undefined, "barra-filtros-bolsa");
    formulario.dataset.documentosFiltros = "";
    const campoBolsa = nodo(doc, "div", undefined, "campo-filtro");
    const etiquetaBolsa = nodo(doc, "label", t("bolsa"));
    etiquetaBolsa.htmlFor = "bolsa-doc-bolsa";
    const selectBolsa = nodo(doc, "select");
    selectBolsa.id = "bolsa-doc-bolsa";
    selectBolsa.name = "bolsa";
    for (const bolsa of bolsas) {
      const opcion = nodo(doc, "option", bolsa.categoria);
      opcion.value = bolsa.bolsa_ref;
      selectBolsa.append(opcion);
    }
    selectBolsa.value = filtros.bolsa;
    selectBolsa.disabled = bolsas.length === 0;
    campoBolsa.append(etiquetaBolsa, selectBolsa);
    const campoTipo = nodo(doc, "div", undefined, "campo-filtro");
    const etiquetaTipo = nodo(doc, "label", t("tipo"));
    etiquetaTipo.htmlFor = "bolsa-doc-tipo";
    const selectTipo = nodo(doc, "select");
    selectTipo.id = "bolsa-doc-tipo";
    selectTipo.name = "tipo";
    for (const [valor, clave] of [["", "todos_tipos"], ["constitucion", "constitucion"]]) {
      const opcion = nodo(doc, "option", t(clave)); opcion.value = valor; selectTipo.append(opcion);
    }
    selectTipo.value = filtros.tipo;
    campoTipo.append(etiquetaTipo, selectTipo);
    const campoTexto = nodo(doc, "div", undefined, "campo-filtro");
    const etiquetaTexto = nodo(doc, "label", t("buscar"));
    etiquetaTexto.htmlFor = "bolsa-doc-texto";
    const input = nodo(doc, "input");
    input.id = "bolsa-doc-texto";
    input.name = "texto";
    input.type = "search";
    input.maxLength = 120;
    input.value = filtros.texto;
    campoTexto.append(etiquetaTexto, input);
    const acciones = nodo(doc, "div", undefined, "acciones-filtro");
    const buscar = boton(doc, t("buscar_accion"), "buscar", "boton-primario");
    buscar.type = "submit";
    acciones.append(buscar, boton(doc, t("limpiar"), "limpiar"));
    formulario.append(campoBolsa, campoTipo, campoTexto, acciones);
    const zonaEstado = nodo(doc, "p", "", "");
    zonaEstado.dataset.documentosEstado = "";
    zonaEstado.setAttribute("role", "status");
    zonaEstado.setAttribute("aria-live", "polite");
    const resultadoNodo = nodo(doc, "div");
    resultadoNodo.dataset.documentosResultado = "";
    cuerpo.append(textoAyuda, formulario, zonaEstado, resultadoNodo);
    contenedor.replaceChildren(cabecera, cuerpo);
    contenedor.addEventListener("click", alPulsar);
    formulario.addEventListener("submit", alBuscar);
    selectBolsa.addEventListener("change", () => aplicarFormulario(formulario));
  }

  function reiniciar() {
    secuencia++;
    consulta?.abort();
    secuenciaRecibo++;
    recibo?.abort();
    cursores = [""];
    pagina = 0;
    resultado = null;
    guardarFiltros(ventana, filtros);
    void cargarPagina();
  }

  function alBuscar(evento) {
    evento.preventDefault();
    aplicarFormulario(evento.currentTarget);
  }

  function aplicarFormulario(formulario) {
    filtros = { bolsa: formulario.elements.bolsa.value, tipo: formulario.elements.tipo.value,
      texto: formulario.elements.texto.value.trim().slice(0, 120) };
    reiniciar();
  }

  function pintarLista() {
    const zona = contenedor.querySelector("[data-documentos-resultado]");
    zona.replaceChildren();
    if (!resultado?.documentos.length) { estado("vacio"); return; }
    const region = nodo(doc, "div", undefined, "tabla-contenedor");
    region.setAttribute("role", "region");
    region.setAttribute("aria-label", t("tabla"));
    region.tabIndex = 0;
    const tabla = nodo(doc, "table", undefined, "tabla-datos");
    const cab = nodo(doc, "thead");
    const filaCab = nodo(doc, "tr");
    for (const clave of ["col_documento", "col_fecha", "col_version", "col_estado"]) {
      const th = nodo(doc, "th", t(clave)); th.scope = "col"; filaCab.append(th);
    }
    cab.append(filaCab);
    const cuerpo = nodo(doc, "tbody");
    resultado.documentos.forEach((item, indice) => {
      const tr = nodo(doc, "tr");
      const documento = nodo(doc, "th");
      documento.scope = "row";
      documento.style.whiteSpace = "normal";
      const acciones = nodo(doc, "div", undefined, "acciones-filtro");
      acciones.style.display = "flex";
      acciones.style.flexWrap = "wrap";
      acciones.style.gap = "8px";
      const titulo = t(item.titulo_clave);
      const ver = boton(doc, t("titulo_con_recibo", { titulo }), "recibo");
      ver.dataset.indice = String(indice);
      ver.dataset.documentoRef = item.documento_ref;
      acciones.append(ver);
      if (typeof abrirCandidatos === "function") {
        const ficha = boton(doc, t("ver_lista"), "lista");
        ficha.dataset.indice = String(indice);
        acciones.append(ficha);
      }
      documento.append(acciones);
      tr.append(documento);
      tr.append(nodo(doc, "td", textos.fecha(item.fecha)));
      tr.append(nodo(doc, "td", textos.numero(item.version_bolsa)));
      tr.append(nodo(doc, "td", t("estado_constitucion")));
      cuerpo.append(tr);
    });
    tabla.append(cab, cuerpo);
    region.append(tabla);
    zona.append(region);
    const pie = nodo(doc, "nav", undefined, "paginacion-bolsa");
    pie.setAttribute("aria-label", t("paginacion"));
    pie.append(boton(doc, t("total", { cuenta: resultado.paginacion.total }), "primera"));
    if (pagina > 0) pie.append(boton(doc, t("anterior"), "anterior"));
    if (resultado.paginacion.cursor_siguiente) pie.append(boton(doc, t("siguiente"), "siguiente"));
    zona.append(pie);
    estado("mostrando", { cuenta: resultado.documentos.length });
  }

  async function cargarPagina() {
    if (!activo || !filtros.bolsa) return;
    const numero = ++secuencia;
    consulta?.abort();
    recibo?.abort();
    secuenciaRecibo++;
    consulta = new AbortController();
    const signal = consulta.signal;
    estado("cargando");
    contenedor.querySelector("[data-documentos-resultado]")?.replaceChildren();
    try {
      const datos = await cliente.listar({ bolsaRef: filtros.bolsa, tipo: filtros.tipo,
        texto: filtros.texto, cursor: cursores[pagina], signal });
      if (!activo || signal.aborted || numero !== secuencia) return;
      resultado = datos;
      pintarLista();
    } catch (error) {
      if (!activo || signal.aborted || numero !== secuencia) return;
      resultado = null;
      const zona = contenedor.querySelector("[data-documentos-resultado]");
      const clave = error?.status === 401 || error?.status === 403 ? "denegado"
        : error?.status === 404 ? "bolsa_no_disponible" : "error_consulta";
      estado(clave);
      zona.append(boton(doc, t("reintentar"), "reintentar"));
    }
  }

  async function verRecibo(item, origen = null) {
    if (origen) botonOrigenRecibo = origen;
    const numero = ++secuenciaRecibo;
    recibo?.abort();
    recibo = new AbortController();
    const signal = recibo.signal;
    const zona = contenedor.querySelector("[data-documentos-resultado]");
    zona.querySelector("[data-documentos-recibo]")?.remove();
    zona.querySelectorAll('[data-accion="reintentar_recibo"]').forEach((control) => control.remove());
    zona.querySelectorAll('[data-accion="recibo"]').forEach((control) => { control.disabled = true; });
    estado("cargando_recibo");
    try {
      const respuesta = await cliente.recibir({ bolsaRef: filtros.bolsa, documentoRef: item.documento_ref, signal });
      if (!activo || signal.aborted || numero !== secuenciaRecibo) return;
      validarCorrespondenciaReciboDocumento(respuesta.datos, item);
      const panel = nodo(doc, "section", undefined, "panel");
      panel.dataset.documentosRecibo = "";
      panel.setAttribute("aria-label", t("recibo_titulo"));
      const cabecera = nodo(doc, "header", undefined, "cabecera-panel");
      const tituloRecibo = nodo(doc, "h3", t("recibo_titulo"));
      tituloRecibo.tabIndex = -1;
      cabecera.append(tituloRecibo, boton(doc, t("cerrar"), "cerrar_recibo"));
      const cuerpo = nodo(doc, "div", undefined, "cuerpo-panel");
      const lista = nodo(doc, "dl");
      for (const [clave, valor] of [["recibo_bolsa", bolsas.find((b) => b.bolsa_ref === filtros.bolsa)?.categoria ?? filtros.bolsa],
        ["recibo_fecha", textos.fecha(respuesta.datos.confirmada_en)],
        ["recibo_version", textos.numero(respuesta.datos.version_bolsa)]]) {
        lista.append(nodo(doc, "dt", t(clave)), nodo(doc, "dd", valor));
      }
      const participaciones = nodo(doc, "dd");
      const cuenta = textos.numero(respuesta.datos.total_participaciones);
      if (typeof abrirCandidatos === "function") {
        participaciones.append(boton(doc, `${cuenta} · ${t("ver_lista")}`, "lista_recibo"));
      } else participaciones.textContent = cuenta;
      lista.append(nodo(doc, "dt", t("recibo_participaciones")), participaciones);
      const detalle = nodo(doc, "details");
      const campos = nodo(doc, "dl");
      for (const [clave, valor] of [["detalle_acta", respuesta.datos.acta_ref],
        ["detalle_listado", respuesta.datos.listado_definitivo_ref],
        ["detalle_version_listado", textos.numero(respuesta.datos.version_listado)]]) {
        const dato = nodo(doc, "dd", valor);
        dato.style.overflowWrap = "anywhere";
        campos.append(nodo(doc, "dt", t(clave)), dato);
      }
      detalle.append(nodo(doc, "summary", t("detalle_recibo")), campos);
      const descargar = boton(doc, t("descargar_recibo"), "descargar");
      descargar.dataset.documentoRef = item.documento_ref;
      descargar.addEventListener("click", () => descargarRecibo(respuesta.bytes));
      cuerpo.append(lista, detalle, descargar);
      panel.append(cabecera, cuerpo);
      zona.append(panel);
      estado("recibo_abierto");
      tituloRecibo.focus({ preventScroll: true });
    } catch (error) {
      if (!activo || signal.aborted || numero !== secuenciaRecibo) return;
      estado(error?.status === 401 || error?.status === 403 ? "denegado" : "error_recibo");
      zona.append(boton(doc, t("reintentar"), "reintentar_recibo"));
      zona.querySelector('[data-accion="reintentar_recibo"]').dataset.documentoRef = item.documento_ref;
    } finally {
      if (activo && numero === secuenciaRecibo) {
        zona.querySelectorAll('[data-accion="recibo"]').forEach((control) => { control.disabled = false; });
      }
    }
  }

  function descargarRecibo(bytes) {
    if (!ventana?.Blob || !ventana.URL?.createObjectURL) { estado("error_descarga"); return; }
    const url = ventana.URL.createObjectURL(new ventana.Blob([bytes], { type: "application/json" }));
    const enlace = nodo(doc, "a");
    enlace.href = url;
    enlace.download = "recibo-constitucion-bolsa.json";
    enlace.hidden = true;
    doc.body.append(enlace);
    try { enlace.click(); } finally { enlace.remove(); ventana.setTimeout(() => ventana.URL.revokeObjectURL(url), 0); }
  }

  function alPulsar(evento) {
    const control = evento.target.closest("[data-accion]");
    if (!control || !contenedor.contains(control)) return;
    const accion = control.dataset.accion;
    if (accion === "ayuda") {
      const ayuda = contenedor.querySelector("#bolsa-documentos-ayuda");
      ayuda.hidden = !ayuda.hidden;
      control.setAttribute("aria-expanded", String(!ayuda.hidden));
    } else if (accion === "limpiar") {
      const form = contenedor.querySelector("[data-documentos-filtros]");
      form.elements.tipo.value = ""; form.elements.texto.value = "";
      filtros = { bolsa: form.elements.bolsa.value, tipo: "", texto: "" };
      reiniciar();
    } else if (accion === "reintentar") void cargarPagina();
    else if (accion === "primera") { pagina = 0; void cargarPagina(); }
    else if (accion === "anterior" && pagina > 0) { pagina--; void cargarPagina(); }
    else if (accion === "siguiente" && resultado?.paginacion.cursor_siguiente) {
      cursores[pagina + 1] = resultado.paginacion.cursor_siguiente;
      pagina++; void cargarPagina();
    } else if (accion === "recibo") {
      const item = resultado?.documentos[Number(control.dataset.indice)];
      if (item && !control.disabled) void verRecibo(item, control);
    } else if (accion === "reintentar_recibo") {
      control.remove();
      const item = resultado?.documentos.find((d) => d.documento_ref === control.dataset.documentoRef);
      if (item) void verRecibo(item);
    } else if (accion === "cerrar_recibo") {
      recibo?.abort(); secuenciaRecibo++;
      contenedor.querySelector("[data-documentos-recibo]")?.remove();
      estado("mostrando", { cuenta: resultado.documentos.length });
      if (botonOrigenRecibo?.isConnected) botonOrigenRecibo.focus({ preventScroll: true });
      botonOrigenRecibo = null;
    } else if (accion === "lista_recibo" && typeof abrirCandidatos === "function") {
      abrirCandidatos(filtros.bolsa);
    } else if (accion === "lista" && typeof abrirCandidatos === "function") {
      const item = resultado?.documentos[Number(control.dataset.indice)];
      if (item) abrirCandidatos(filtros.bolsa);
    }
  }

  await cargarCatalogo();
  return Object.freeze({ desmontar });
}
