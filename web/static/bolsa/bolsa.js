"use strict";

(() => {
  const API = "/api/publico/bolsa/convocatorias";
  const TAMANO = 12;
  const porId = (id) => document.getElementById(id);
  const elementos = {
    formulario: porId("filtros-convocatorias"), texto: porId("filtro-texto"), tipo: porId("filtro-tipo"),
    categoria: porId("filtro-categoria"), estado: porId("filtro-estado"), plazo: porId("filtro-plazo"),
    limpiar: porId("limpiar-filtros"), reintentar: porId("reintentar-consulta"), listado: porId("lista-convocatorias"),
    panelListado: porId("panel-listado"), cargando: porId("estado-cargando"), error: porId("estado-error"), vacio: porId("estado-vacio"),
    estadoConsulta: porId("estado-consulta"), revision: porId("revision-fuente"), avisoContenedor: porId("aviso-demostracion"), aviso: porId("texto-aviso-demostracion"),
    paginacion: porId("paginacion"), anterior: porId("pagina-anterior"), siguiente: porId("pagina-siguiente"), pagina: porId("pagina-actual"),
    panelDetalle: porId("panel-detalle"), tituloDetalle: porId("titulo-detalle"), cerrarDetalle: porId("cerrar-detalle"),
    detalleEspera: porId("detalle-espera"), detalleCargando: porId("detalle-cargando"), detalleError: porId("detalle-error"),
    contenidoDetalle: porId("contenido-detalle"), detalleEtiquetas: porId("detalle-etiquetas"), detalleResumen: porId("detalle-resumen"),
    detalleDescripcion: porId("detalle-descripcion"), detallePlazos: porId("detalle-plazos"), detalleRequisitos: porId("detalle-requisitos"),
    detalleDocumentos: porId("detalle-documentos"), detalleAyuda: porId("detalle-ayuda"), detalleIntegridad: porId("detalle-integridad"),
  };
  const estado = { pagina: 1, paginas: 0, convocatoria: "", controladorListado: null, controladorDetalle: null };
  const formatoFecha = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" });

  function texto(tag, contenido, clase) {
    const nodo = document.createElement(tag);
    if (clase) nodo.className = clase;
    nodo.textContent = contenido;
    return nodo;
  }

  function vaciar(nodo) {
    while (nodo.firstChild) nodo.removeChild(nodo.firstChild);
  }

  function etiqueta(valor) {
    const nodo = texto("span", valor.etiqueta, "etiqueta");
    nodo.dataset.semantica = valor.semantica;
    if (valor.descripcion) nodo.title = valor.descripcion;
    return nodo;
  }

  function fecha(instante) {
    const nodo = document.createElement("time");
    nodo.dateTime = instante;
    const valor = new Date(instante);
    nodo.textContent = Number.isNaN(valor.getTime()) ? instante : formatoFecha.format(valor);
    return nodo;
  }

  function intervalo(plazo) {
    const contenedor = document.createElement("span");
    contenedor.append("Desde ", fecha(plazo.abre_en), " hasta ", fecha(plazo.cierra_en));
    return contenedor;
  }

  function configurarPreferencia(idBoton, clase, clave) {
    const boton = porId(idBoton);
    let activa = false;
    try { activa = localStorage.getItem(clave) === "true"; } catch (_) { activa = false; }
    document.body.classList.toggle(clase, activa);
    boton.setAttribute("aria-pressed", String(activa));
    boton.addEventListener("click", () => {
      activa = !document.body.classList.contains(clase);
      document.body.classList.toggle(clase, activa);
      boton.setAttribute("aria-pressed", String(activa));
      try { localStorage.setItem(clave, String(activa)); } catch (_) { /* preferencia no esencial */ }
    });
  }

  function cargarFiltrosDesdeURL() {
    const parametros = new URLSearchParams(window.location.search);
    elementos.texto.value = parametros.get("texto") || "";
    elementos.tipo.dataset.valorInicial = parametros.get("tipo") || "";
    elementos.categoria.dataset.valorInicial = parametros.get("categoria") || "";
    elementos.estado.dataset.valorInicial = parametros.get("estado") || "";
    elementos.plazo.checked = parametros.get("plazo") === "abierto";
    const pagina = Number.parseInt(parametros.get("pagina") || "1", 10);
    estado.pagina = Number.isInteger(pagina) && pagina > 0 ? pagina : 1;
    estado.convocatoria = parametros.get("convocatoria") || "";
  }

  function parametrosConsulta() {
    const parametros = new URLSearchParams();
    const valores = [
      ["texto", elementos.texto.value.trim()], ["tipo", elementos.tipo.value],
      ["categoria", elementos.categoria.value], ["estado", elementos.estado.value],
    ];
    valores.forEach(([clave, valor]) => { if (valor) parametros.set(clave, valor); });
    if (elementos.plazo.checked) parametros.set("plazo", "abierto");
    parametros.set("pagina", String(estado.pagina));
    parametros.set("tamano", String(TAMANO));
    return parametros;
  }

  function actualizarURL() {
    const parametros = parametrosConsulta();
    parametros.delete("tamano");
    if (estado.pagina === 1) parametros.delete("pagina");
    if (estado.convocatoria) parametros.set("convocatoria", estado.convocatoria);
    const consulta = parametros.toString();
    history.replaceState(null, "", `/bolsa/${consulta ? `?${consulta}` : ""}`);
  }

  async function obtenerJSON(url, signal) {
    const respuesta = await fetch(url, { method: "GET", headers: { Accept: "application/json" }, credentials: "same-origin", signal });
    const tipo = respuesta.headers.get("content-type") || "";
    if (!tipo.includes("application/json")) throw new Error("respuesta no JSON");
    const contenido = await respuesta.json();
    if (!respuesta.ok) throw new Error(contenido?.error?.codigo || "consulta fallida");
    return contenido;
  }

  function estadoListado(nombre) {
    elementos.panelListado.setAttribute("aria-busy", String(nombre === "cargando"));
    elementos.cargando.hidden = nombre !== "cargando";
    elementos.error.hidden = nombre !== "error";
    elementos.vacio.hidden = nombre !== "vacio";
    elementos.listado.hidden = nombre !== "listo";
    if (nombre !== "listo") elementos.paginacion.hidden = true;
  }

  function completarSelect(select, valores, textoTodos) {
    const seleccionado = select.value || select.dataset.valorInicial || "";
    vaciar(select);
    const todos = document.createElement("option");
    todos.value = "";
    todos.textContent = textoTodos;
    select.appendChild(todos);
    valores.forEach((valor) => {
      const opcion = document.createElement("option");
      opcion.value = valor.clave;
      opcion.textContent = valor.etiqueta;
      select.appendChild(opcion);
    });
    if ([...select.options].some((opcion) => opcion.value === seleccionado)) select.value = seleccionado;
    delete select.dataset.valorInicial;
  }

  function renderizarFacetas(facetas) {
    completarSelect(elementos.tipo, facetas.tipos, "Todos los tipos");
    completarSelect(elementos.categoria, facetas.categorias, "Todas las categorías");
    completarSelect(elementos.estado, facetas.estados, "Todos los estados");
  }

  function tarjetaConvocatoria(convocatoria) {
    const item = document.createElement("li");
    const articulo = document.createElement("article");
    articulo.className = "tarjeta-convocatoria";
    articulo.dataset.identificador = convocatoria.identificador_publico;
    articulo.setAttribute("aria-current", String(convocatoria.identificador_publico === estado.convocatoria));
    const etiquetas = document.createElement("div");
    etiquetas.className = "grupo-etiquetas";
    etiquetas.append(etiqueta(convocatoria.tipo), etiqueta(convocatoria.estado));
    convocatoria.categorias.forEach((categoria) => etiquetas.appendChild(etiqueta(categoria)));
    articulo.appendChild(etiquetas);
    articulo.appendChild(texto("h3", convocatoria.titulo));
    articulo.appendChild(texto("p", convocatoria.resumen));
    if (convocatoria.plazo_destacado) {
      const plazo = document.createElement("div");
      plazo.className = "plazo-meta";
      plazo.append(etiqueta({ etiqueta: convocatoria.plazo_destacado.etiqueta_situacion, semantica: convocatoria.plazo_destacado.semantica_situacion }));
      plazo.appendChild(intervalo(convocatoria.plazo_destacado));
      articulo.appendChild(plazo);
    }
    articulo.appendChild(texto("p", `${convocatoria.numero_requisitos} requisitos · ${convocatoria.numero_documentos} documentos · ${convocatoria.numero_ayudas} ayudas`, "tarjeta-meta"));
    const enlace = texto("a", "Consultar ficha pública", "enlace-detalle");
    enlace.href = `/bolsa/?convocatoria=${encodeURIComponent(convocatoria.identificador_publico)}`;
    enlace.addEventListener("click", (evento) => {
      evento.preventDefault();
      abrirDetalle(convocatoria.identificador_publico, true);
    });
    articulo.appendChild(enlace);
    item.appendChild(articulo);
    return item;
  }

  function renderizarListado(datos) {
    if (datos.esquema !== "vec.bolsa.publico.convocatorias.v1" || !Array.isArray(datos.convocatorias)) throw new Error("esquema público inesperado");
    renderizarFacetas(datos.facetas);
    elementos.avisoContenedor.hidden = datos.fuente.demostracion !== true;
    if (datos.fuente.demostracion === true) elementos.aviso.textContent = datos.fuente.aviso || "DEMOSTRACIÓN sin validez administrativa.";
    elementos.revision.textContent = `Fuente ${datos.fuente.revision} · actualizada ${formatoFecha.format(new Date(datos.fuente.actualizada_en))}`;
    estado.paginas = datos.paginacion.paginas;
    estado.pagina = datos.paginacion.pagina;
    elementos.estadoConsulta.textContent = `${datos.paginacion.total} convocatorias encontradas`;
    vaciar(elementos.listado);
    datos.convocatorias.forEach((convocatoria) => elementos.listado.appendChild(tarjetaConvocatoria(convocatoria)));
    if (datos.convocatorias.length === 0) {
      estadoListado("vacio");
      return;
    }
    estadoListado("listo");
    elementos.paginacion.hidden = estado.paginas <= 1;
    elementos.pagina.textContent = `Página ${estado.pagina} de ${estado.paginas}`;
    elementos.anterior.disabled = estado.pagina <= 1;
    elementos.siguiente.disabled = estado.pagina >= estado.paginas;
  }

  async function cargarListado() {
    if (estado.controladorListado) estado.controladorListado.abort();
    estado.controladorListado = new AbortController();
    estadoListado("cargando");
    elementos.estadoConsulta.textContent = "Cargando convocatorias…";
    actualizarURL();
    try {
      const datos = await obtenerJSON(`${API}?${parametrosConsulta().toString()}`, estado.controladorListado.signal);
      renderizarListado(datos);
      actualizarURL();
      if (estado.convocatoria) await abrirDetalle(estado.convocatoria, false);
    } catch (error) {
      if (error.name === "AbortError") return;
      estadoListado("error");
      elementos.estadoConsulta.textContent = "La consulta no está disponible.";
    }
  }

  function estadoDetalle(nombre) {
    elementos.panelDetalle.setAttribute("aria-busy", String(nombre === "cargando"));
    elementos.detalleEspera.hidden = nombre !== "espera";
    elementos.detalleCargando.hidden = nombre !== "cargando";
    elementos.detalleError.hidden = nombre !== "error";
    elementos.contenidoDetalle.hidden = nombre !== "listo";
    elementos.cerrarDetalle.hidden = nombre === "espera";
  }

  function renderizarPlazos(plazos) {
    vaciar(elementos.detallePlazos);
    plazos.forEach((plazo) => {
      const bloque = document.createElement("article");
      bloque.className = "bloque-plazo";
      bloque.appendChild(texto("h4", plazo.titulo));
      const marcas = document.createElement("div");
      marcas.className = "grupo-etiquetas";
      marcas.append(etiqueta(plazo.tipo), etiqueta({ etiqueta: plazo.etiqueta_situacion, semantica: plazo.semantica_situacion }));
      bloque.appendChild(marcas);
      bloque.appendChild(intervalo(plazo));
      if (plazo.descripcion) bloque.appendChild(texto("p", plazo.descripcion));
      elementos.detallePlazos.appendChild(bloque);
    });
  }

  function renderizarRequisitos(requisitos) {
    vaciar(elementos.detalleRequisitos);
    requisitos.forEach((requisito) => {
      const item = document.createElement("li");
      item.appendChild(texto("h4", requisito.titulo));
      item.appendChild(texto("p", requisito.descripcion));
      item.appendChild(texto("span", requisito.obligatorio ? "Obligatorio" : "No obligatorio", "etiqueta"));
      elementos.detalleRequisitos.appendChild(item);
    });
  }

  function renderizarDocumentos(documentos) {
    vaciar(elementos.detalleDocumentos);
    documentos.forEach((documento) => {
      const item = document.createElement("li");
      item.append(etiqueta(documento.tipo), texto("h4", documento.titulo), texto("p", documento.descripcion));
      const enlace = texto("a", `Abrir ${documento.formato.toUpperCase()}: ${documento.titulo}`, "enlace-documento");
      enlace.href = documento.url;
      item.appendChild(enlace);
      elementos.detalleDocumentos.appendChild(item);
    });
  }

  function renderizarAyuda(ayudas) {
    vaciar(elementos.detalleAyuda);
    ayudas.forEach((ayuda) => {
      const bloque = document.createElement("details");
      bloque.className = "ayuda-item";
      bloque.appendChild(texto("summary", ayuda.pregunta));
      bloque.append(etiqueta(ayuda.categoria), texto("p", ayuda.respuesta));
      elementos.detalleAyuda.appendChild(bloque);
    });
  }

  function renderizarDetalle(datos) {
    if (datos.esquema !== "vec.bolsa.publico.convocatoria.v1") throw new Error("esquema de detalle inesperado");
    const convocatoria = datos.convocatoria;
    elementos.tituloDetalle.textContent = convocatoria.titulo;
    vaciar(elementos.detalleEtiquetas);
    elementos.detalleEtiquetas.append(etiqueta(convocatoria.tipo), etiqueta(convocatoria.estado));
    convocatoria.categorias.forEach((categoria) => elementos.detalleEtiquetas.appendChild(etiqueta(categoria)));
    elementos.detalleResumen.textContent = convocatoria.resumen;
    elementos.detalleDescripcion.textContent = datos.descripcion;
    renderizarPlazos(datos.plazos);
    renderizarRequisitos(datos.requisitos);
    renderizarDocumentos(datos.documentos);
    renderizarAyuda(datos.ayuda);
    elementos.detalleIntegridad.textContent = `Versión ${convocatoria.version} · huella SHA-256 ${convocatoria.huella_sha256}`;
    estadoDetalle("listo");
    document.querySelectorAll(".tarjeta-convocatoria").forEach((tarjeta) => tarjeta.setAttribute("aria-current", String(tarjeta.dataset.identificador === estado.convocatoria)));
  }

  async function abrirDetalle(identificador, moverFoco) {
    if (estado.controladorDetalle) estado.controladorDetalle.abort();
    estado.controladorDetalle = new AbortController();
    estado.convocatoria = identificador;
    actualizarURL();
    estadoDetalle("cargando");
    try {
      const datos = await obtenerJSON(`${API}/${encodeURIComponent(identificador)}`, estado.controladorDetalle.signal);
      renderizarDetalle(datos);
      if (moverFoco && window.matchMedia("(max-width: 800px)").matches) elementos.tituloDetalle.focus?.();
    } catch (error) {
      if (error.name === "AbortError") return;
      elementos.tituloDetalle.textContent = "Ficha no disponible";
      estadoDetalle("error");
    }
  }

  function cerrarDetalle() {
    if (estado.controladorDetalle) estado.controladorDetalle.abort();
    estado.convocatoria = "";
    elementos.tituloDetalle.textContent = "Seleccione una convocatoria";
    estadoDetalle("espera");
    document.querySelectorAll(".tarjeta-convocatoria").forEach((tarjeta) => tarjeta.setAttribute("aria-current", "false"));
    actualizarURL();
    porId("titulo-resultados").focus?.();
  }

  elementos.formulario.addEventListener("submit", (evento) => { evento.preventDefault(); estado.pagina = 1; estado.convocatoria = ""; cerrarDetalle(); cargarListado(); });
  elementos.limpiar.addEventListener("click", () => { elementos.formulario.reset(); estado.pagina = 1; estado.convocatoria = ""; cerrarDetalle(); cargarListado(); });
  elementos.reintentar.addEventListener("click", cargarListado);
  elementos.anterior.addEventListener("click", () => { if (estado.pagina > 1) { estado.pagina -= 1; cargarListado(); } });
  elementos.siguiente.addEventListener("click", () => { if (estado.pagina < estado.paginas) { estado.pagina += 1; cargarListado(); } });
  elementos.cerrarDetalle.addEventListener("click", cerrarDetalle);

  configurarPreferencia("alternar-texto", "texto-grande", "vec.bolsa.texto_grande");
  configurarPreferencia("alternar-contraste", "alto-contraste", "vec.bolsa.alto_contraste");
  cargarFiltrosDesdeURL();
  cargarListado();
})();
