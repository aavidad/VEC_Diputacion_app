"use strict";

(() => {
  const API = "/api/publico/bolsa/convocatorias";
  const API_CATEGORIAS = "/api/publico/bolsa/categorias";
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
    reintentarDetalle: porId("reintentar-detalle"),
    contenidoDetalle: porId("contenido-detalle"), detalleEtiquetas: porId("detalle-etiquetas"), detalleResumen: porId("detalle-resumen"),
    detallePublicacion: porId("detalle-publicacion"),
    detalleDescripcion: porId("detalle-descripcion"), detallePlazos: porId("detalle-plazos"), detalleRequisitos: porId("detalle-requisitos"),
    detalleDocumentos: porId("detalle-documentos"), detalleAyuda: porId("detalle-ayuda"), detalleIntegridad: porId("detalle-integridad"),
    directorio: porId("directorio-categorias"), buscarCategoria: porId("buscar-categoria"), areaCategoria: porId("filtrar-area-categoria"),
    estadoDirectorio: porId("estado-directorio-categorias"), cargandoDirectorio: porId("cargando-directorio-categorias"),
    errorDirectorio: porId("error-directorio-categorias"), vacioDirectorio: porId("vacio-directorio-categorias"),
    gruposDirectorio: porId("grupos-directorio-categorias"), reintentarDirectorio: porId("reintentar-directorio-categorias"),
    integridadCategorias: porId("integridad-catalogo-categorias"),
    inicioInstitucional: porId("enlace-inicio-institucional"),
  };
  const estado = {
    pagina: 1, paginas: 0, convocatoria: "", categorias: [], controladorListado: null, controladorDetalle: null,
    controladorCategorias: null, etiquetasArea: new Map(), fuentesDemostracion: { convocatorias: null, categorias: null }, avisosDemostracion: {},
    facetas: null,
  };
  const formatoFecha = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" });
  const formatoDia = new Intl.DateTimeFormat("es-ES", { dateStyle: "long", timeZone: "Europe/Madrid" });

  function texto(tag, contenido, clase) {
    const nodo = document.createElement(tag);
    if (clase) nodo.className = clase;
    nodo.textContent = contenido;
    return nodo;
  }

  function vaciar(nodo) {
    while (nodo.firstChild) nodo.removeChild(nodo.firstChild);
  }

  function normalizarBusqueda(valor) {
    return String(valor || "").normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLocaleLowerCase("es").trim();
  }

  function etiquetaArea(valor) {
    if (estado.etiquetasArea.has(valor)) return estado.etiquetasArea.get(valor);
    const limpio = String(valor || "").replace(/[_-]+/g, " ").trim();
    return limpio ? limpio.charAt(0).toLocaleUpperCase("es") + limpio.slice(1) : "Área no indicada";
  }

  function actualizarAvisoDemostracion(origen, fuente) {
    estado.fuentesDemostracion[origen] = fuente?.demostracion === true;
    const aviso = String(fuente?.aviso || "").replace(/^DEMOSTRACI[ÓO]N\s*:?\s*/i, "").trim();
    if (aviso) estado.avisosDemostracion[origen] = aviso;
    const valores = Object.values(estado.fuentesDemostracion);
    const esDemostracion = valores.some(Boolean);
    elementos.avisoContenedor.hidden = valores.every((valor) => valor !== null) && !esDemostracion;
    elementos.inicioInstitucional.href = esDemostracion ? "/presentacion/" : "/bolsa/";
    elementos.inicioInstitucional.setAttribute("aria-label", esDemostracion
      ? "Volver al selector de recorridos de la presentación"
      : "Inicio de Bolsa y procesos selectivos");
    const avisos = [...new Set(Object.values(estado.avisosDemostracion))];
    if (esDemostracion) {
      elementos.aviso.textContent = "Datos públicos reales de referencia; plazos y actuaciones rotulados DEMO son sintéticos y carecen de validez administrativa.";
      elementos.aviso.title = avisos.join(" ");
    }
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

  function fechaPublicacion(instante) {
    const valor = new Date(instante);
    return Number.isNaN(valor.getTime()) ? instante : formatoDia.format(valor);
  }

  function configurarPreferencia(idBoton, clase) {
    const boton = porId(idBoton);
    boton.setAttribute("aria-pressed", "false");
    boton.addEventListener("click", () => {
      const activa = !document.body.classList.contains(clase);
      document.body.classList.toggle(clase, activa);
      boton.setAttribute("aria-pressed", String(activa));
    });
  }

  function cargarFiltrosDesdeURL() {
    const parametros = new URLSearchParams(window.location.search);
    elementos.texto.value = parametros.get("texto") || "";
    elementos.tipo.value = "";
    elementos.categoria.value = "";
    elementos.estado.value = "";
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
      ["texto", elementos.texto.value.trim()], ["tipo", elementos.tipo.value || elementos.tipo.dataset.valorInicial || ""],
      ["categoria", elementos.categoria.value || elementos.categoria.dataset.valorInicial || ""],
      ["estado", elementos.estado.value || elementos.estado.dataset.valorInicial || ""],
    ];
    valores.forEach(([clave, valor]) => { if (valor) parametros.set(clave, valor); });
    if (elementos.plazo.checked) parametros.set("plazo", "abierto");
    parametros.set("pagina", String(estado.pagina));
    parametros.set("tamano", String(TAMANO));
    return parametros;
  }

  function actualizarURL(modo = "replace", conservarAncla = true, datosHistorial = undefined) {
    if (modo === "none") return;
    const parametros = parametrosConsulta();
    parametros.delete("tamano");
    if (estado.pagina === 1) parametros.delete("pagina");
    if (estado.convocatoria) parametros.set("convocatoria", estado.convocatoria);
    const consulta = parametros.toString();
    const ancla = conservarAncla ? window.location.hash : "";
    const destino = `/bolsa/${consulta ? `?${consulta}` : ""}${ancla}`;
    const datos = datosHistorial === undefined ? (modo === "replace" ? history.state : null) : datosHistorial;
    if (modo === "push") history.pushState(datos, "", destino);
    else history.replaceState(datos, "", destino);
  }

  function activacionSimpleEnlace(evento) {
    return !evento.defaultPrevented && evento.button === 0 && !evento.metaKey && !evento.ctrlKey && !evento.shiftKey && !evento.altKey;
  }

  async function obtenerJSON(url, signal) {
    const respuesta = await fetch(url, { method: "GET", headers: { Accept: "application/json" }, credentials: "omit", signal });
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
      opcion.textContent = Number.isInteger(valor.numero_resultados) && valor.numero_resultados > 0
        ? `${valor.etiqueta} (${valor.numero_resultados})`
        : valor.etiqueta;
      select.appendChild(opcion);
    });
    if (seleccionado && ![...select.options].some((opcion) => opcion.value === seleccionado)) {
      const categoria = select === elementos.categoria ? estado.categorias.find((valor) => valor.clave === seleccionado) : null;
      const opcionSeleccionada = document.createElement("option");
      opcionSeleccionada.value = seleccionado;
      opcionSeleccionada.textContent = categoria
        ? `${categoria.etiqueta} (sin procesos publicados)`
        : `Selección sin resultados: ${seleccionado}`;
      select.appendChild(opcionSeleccionada);
    }
    if ([...select.options].some((opcion) => opcion.value === seleccionado)) select.value = seleccionado;
    delete select.dataset.valorInicial;
  }

  function renderizarFacetas(facetas) {
    estado.facetas = facetas;
    completarSelect(elementos.tipo, facetas.tipos, "Todos los tipos");
    completarSelect(elementos.categoria, facetas.categorias, "Todas con procesos");
    completarSelect(elementos.estado, facetas.estados, "Todos los estados");
  }

  function cantidad(total, singular, plural) {
    return `${total} ${total === 1 ? singular : plural}`;
  }

  function hrefDetalle(identificador) {
    const parametros = parametrosConsulta();
    parametros.delete("tamano");
    if (estado.pagina === 1) parametros.delete("pagina");
    parametros.set("convocatoria", identificador);
    return `/bolsa/?${parametros.toString()}`;
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
    const publicacion = texto("p", `Publicada el ${fechaPublicacion(convocatoria.publicada_en)}`, "tarjeta-meta");
    const publicacionTiempo = document.createElement("time");
    publicacionTiempo.dateTime = convocatoria.publicada_en;
    publicacionTiempo.textContent = fechaPublicacion(convocatoria.publicada_en);
    publicacion.replaceChildren("Publicada el ", publicacionTiempo);
    articulo.appendChild(publicacion);
    if (convocatoria.plazo_destacado) {
      const plazo = document.createElement("div");
      plazo.className = "plazo-meta";
      plazo.append(etiqueta({ etiqueta: convocatoria.plazo_destacado.etiqueta_situacion, semantica: convocatoria.plazo_destacado.semantica_situacion }));
      plazo.appendChild(intervalo(convocatoria.plazo_destacado));
      articulo.appendChild(plazo);
    }
    articulo.appendChild(texto("p", [
      cantidad(convocatoria.numero_requisitos, "requisito", "requisitos"),
      cantidad(convocatoria.numero_documentos, "documento", "documentos"),
      cantidad(convocatoria.numero_ayudas, "ayuda", "ayudas"),
    ].join(" · "), "tarjeta-meta"));
    const enlace = texto("a", "Consultar ficha pública", "enlace-detalle");
    enlace.href = hrefDetalle(convocatoria.identificador_publico);
    enlace.addEventListener("click", (evento) => {
      if (!activacionSimpleEnlace(evento)) return;
      evento.preventDefault();
      abrirDetalle(convocatoria.identificador_publico, true, "push", false);
    });
    articulo.appendChild(enlace);
    item.appendChild(articulo);
    return item;
  }

  function renderizarListado(datos) {
    if (datos.esquema !== "vec.bolsa.publico.convocatorias.v1" || !Array.isArray(datos.convocatorias)) throw new Error("esquema público inesperado");
    renderizarFacetas(datos.facetas);
    actualizarAvisoDemostracion("convocatorias", datos.fuente);
    elementos.revision.textContent = `Fuente ${datos.fuente.revision} · actualizada ${formatoFecha.format(new Date(datos.fuente.actualizada_en))}`;
    estado.paginas = datos.paginacion.paginas;
    estado.pagina = datos.paginacion.pagina;
    elementos.estadoConsulta.textContent = cantidad(datos.paginacion.total, "convocatoria encontrada", "convocatorias encontradas");
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

  async function cargarListado(modoHistoria = "replace", conservarAncla = true) {
    if (estado.controladorListado) estado.controladorListado.abort();
    estado.controladorListado = new AbortController();
    estadoListado("cargando");
    elementos.estadoConsulta.textContent = "Cargando convocatorias…";
    actualizarURL(modoHistoria, conservarAncla);
    try {
      let datos = await obtenerJSON(`${API}?${parametrosConsulta().toString()}`, estado.controladorListado.signal);
      const paginaValida = datos.paginacion.paginas > 0 ? Math.min(estado.pagina, datos.paginacion.paginas) : 1;
      if (paginaValida !== estado.pagina) {
        estado.pagina = paginaValida;
        actualizarURL("replace", conservarAncla);
        datos = await obtenerJSON(`${API}?${parametrosConsulta().toString()}`, estado.controladorListado.signal);
      }
      renderizarListado(datos);
      actualizarURL(modoHistoria === "none" ? "none" : "replace", conservarAncla);
      if (estado.convocatoria) await abrirDetalle(estado.convocatoria, false, "none", conservarAncla);
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
    if (plazos.length === 0) {
      elementos.detallePlazos.appendChild(texto("p", "No hay plazos públicos asociados.", "detalle-vacio"));
      return;
    }
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
    if (requisitos.length === 0) {
      elementos.detalleRequisitos.appendChild(texto("li", "No hay requisitos públicos asociados.", "detalle-vacio"));
      return;
    }
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
    if (documentos.length === 0) {
      elementos.detalleDocumentos.appendChild(texto("li", "No hay documentos públicos asociados.", "detalle-vacio"));
      return;
    }
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
    if (ayudas.length === 0) {
      elementos.detalleAyuda.appendChild(texto("p", "No hay respuestas de ayuda asociadas.", "detalle-vacio"));
      return;
    }
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
    const publicacionTiempo = document.createElement("time");
    publicacionTiempo.dateTime = convocatoria.publicada_en;
    publicacionTiempo.textContent = fechaPublicacion(convocatoria.publicada_en);
    elementos.detallePublicacion.replaceChildren("Bases publicadas el ", publicacionTiempo);
    elementos.detalleDescripcion.textContent = datos.descripcion;
    renderizarPlazos(datos.plazos);
    renderizarRequisitos(datos.requisitos);
    renderizarDocumentos(datos.documentos);
    renderizarAyuda(datos.ayuda);
    elementos.detalleIntegridad.textContent = `Versión ${convocatoria.version} · huella SHA-256 ${convocatoria.huella_sha256}`;
    estadoDetalle("listo");
    document.querySelectorAll(".tarjeta-convocatoria").forEach((tarjeta) => tarjeta.setAttribute("aria-current", String(tarjeta.dataset.identificador === estado.convocatoria)));
  }

  async function abrirDetalle(identificador, moverFoco, modoHistoria = "replace", conservarAncla = true) {
    if (estado.controladorDetalle) estado.controladorDetalle.abort();
    estado.controladorDetalle = new AbortController();
    estado.convocatoria = identificador;
    let modoEfectivo = modoHistoria;
    if (modoHistoria === "push" && history.state?.vecBolsaDetalle === true) modoEfectivo = "replace";
    actualizarURL(modoEfectivo, conservarAncla, modoHistoria === "push" ? { vecBolsaDetalle: true } : undefined);
    estadoDetalle("cargando");
    try {
      const datos = await obtenerJSON(`${API}/${encodeURIComponent(identificador)}`, estado.controladorDetalle.signal);
      renderizarDetalle(datos);
      if (moverFoco) elementos.tituloDetalle.focus?.();
    } catch (error) {
      if (error.name === "AbortError") return;
      elementos.tituloDetalle.textContent = "Ficha no disponible";
      estadoDetalle("error");
    }
  }

  function reiniciarDetalleVisual() {
    if (estado.controladorDetalle) estado.controladorDetalle.abort();
    elementos.tituloDetalle.textContent = "Seleccione una convocatoria";
    estadoDetalle("espera");
    document.querySelectorAll(".tarjeta-convocatoria").forEach((tarjeta) => tarjeta.setAttribute("aria-current", "false"));
  }

  function cerrarDetalle(modoHistoria = "replace", moverFoco = true, conservarAncla = false) {
    reiniciarDetalleVisual();
    estado.convocatoria = "";
    if (modoHistoria === "replace" && history.state?.vecBolsaDetalle === true) {
      history.back();
      if (moverFoco) porId("titulo-resultados").focus?.();
      return;
    }
    actualizarURL(modoHistoria, conservarAncla);
    if (moverFoco) porId("titulo-resultados").focus?.();
  }

  function mostrarEstadoDirectorio(nombre) {
    elementos.directorio.setAttribute("aria-busy", String(nombre === "cargando"));
    elementos.cargandoDirectorio.hidden = nombre !== "cargando";
    elementos.errorDirectorio.hidden = nombre !== "error";
    elementos.vacioDirectorio.hidden = nombre !== "vacio";
    elementos.gruposDirectorio.hidden = nombre !== "listo";
  }

  function configurarAreasDirectorio(categorias) {
    const seleccionada = elementos.areaCategoria.value;
    const areas = [...new Set(categorias.map((categoria) => categoria.area).filter(Boolean))];
    vaciar(elementos.areaCategoria);
    const todas = document.createElement("option");
    todas.value = "";
    todas.textContent = "Todas las áreas";
    elementos.areaCategoria.appendChild(todas);
    areas.forEach((area) => {
      const opcion = document.createElement("option");
      opcion.value = area;
      opcion.textContent = etiquetaArea(area);
      elementos.areaCategoria.appendChild(opcion);
    });
    if (areas.includes(seleccionada)) elementos.areaCategoria.value = seleccionada;
  }

  function resumenConteo(categoria) {
    const procesos = Number(categoria.numero_convocatorias) || 0;
    const abiertos = Number(categoria.numero_plazos_abiertos) || 0;
    const textoProcesos = `${procesos} ${procesos === 1 ? "proceso publicado" : "procesos publicados"}`;
    const textoAbiertos = `${abiertos} ${abiertos === 1 ? "plazo abierto" : "plazos abiertos"}`;
    return `${textoProcesos} · ${textoAbiertos}`;
  }

  async function aplicarCategoriaDesdeDirectorio(categoria) {
    elementos.formulario.reset();
    elementos.categoria.dataset.valorInicial = categoria.clave;
    estado.pagina = 1;
    cerrarDetalle("none", false);
    await cargarListado("push", false);
    porId("titulo-resultados").focus?.();
  }

  function filaCategoria(categoria) {
    const item = document.createElement("li");
    item.className = "categoria-directorio";
    const contenido = document.createElement("div");
    contenido.className = "categoria-directorio__contenido";
    contenido.appendChild(texto("h4", categoria.etiqueta));
    if (categoria.descripcion) contenido.appendChild(texto("p", categoria.descripcion));
    contenido.appendChild(texto("p", resumenConteo(categoria), "categoria-directorio__conteo"));
    item.appendChild(contenido);

    if (Number(categoria.numero_convocatorias) > 0) {
      const enlace = texto("a", "Ver procesos", "enlace-detalle");
      enlace.setAttribute("aria-label", `Ver procesos de ${categoria.etiqueta}`);
      enlace.href = `/bolsa/?categoria=${encodeURIComponent(categoria.clave)}`;
      enlace.addEventListener("click", (evento) => {
        if (!activacionSimpleEnlace(evento)) return;
        evento.preventDefault();
        aplicarCategoriaDesdeDirectorio(categoria);
      });
      item.appendChild(enlace);
    } else {
      item.appendChild(texto("span", "Sin convocatorias publicadas actualmente", "categoria-directorio__sin-procesos"));
    }
    return item;
  }

  function renderizarDirectorioFiltrado() {
    const consulta = normalizarBusqueda(elementos.buscarCategoria.value);
    const area = elementos.areaCategoria.value;
    const filtradas = estado.categorias.filter((categoria) => {
      if (area && categoria.area !== area) return false;
      if (!consulta) return true;
      return normalizarBusqueda([categoria.etiqueta, categoria.descripcion, categoria.clave].join(" ")).includes(consulta);
    });
    elementos.estadoDirectorio.textContent = `${filtradas.length} de ${estado.categorias.length} categorías mostradas`;
    vaciar(elementos.gruposDirectorio);
    if (filtradas.length === 0) {
      mostrarEstadoDirectorio("vacio");
      return;
    }

    const grupos = new Map();
    filtradas.forEach((categoria) => {
      const areaCategoria = categoria.area || "Área no indicada";
      if (!grupos.has(areaCategoria)) grupos.set(areaCategoria, []);
      grupos.get(areaCategoria).push(categoria);
    });
    [...grupos.entries()].forEach(([nombreArea, categorias]) => {
      const grupo = document.createElement("section");
      grupo.className = "grupo-directorio";
      const titulo = texto("h3", etiquetaArea(nombreArea));
      titulo.id = `area-${normalizarBusqueda(nombreArea).replace(/[^a-z0-9]+/g, "-")}`;
      grupo.setAttribute("aria-labelledby", titulo.id);
      grupo.appendChild(titulo);
      const lista = document.createElement("ul");
      lista.className = "lista-directorio";
      categorias.forEach((categoria) => lista.appendChild(filaCategoria(categoria)));
      grupo.appendChild(lista);
      elementos.gruposDirectorio.appendChild(grupo);
    });
    mostrarEstadoDirectorio("listo");
  }

  function renderizarDirectorio(datos) {
    if (datos.esquema !== "vec.bolsa.publico.categorias.v1" || !datos.catalogo || !Array.isArray(datos.categorias)) {
      throw new Error("esquema de categorías inesperado");
    }
    if (datos.catalogo.total !== datos.categorias.length || !/^[a-f0-9]{64}$/.test(datos.catalogo.huella_sha256 || "")) {
      throw new Error("integridad de categorías incoherente");
    }
    estado.categorias = datos.categorias.slice().sort((a, b) => (a.orden - b.orden) || a.etiqueta.localeCompare(b.etiqueta, "es"));
    estado.etiquetasArea = new Map(estado.categorias.map((categoria) => [categoria.area, categoria.area_etiqueta]));
    actualizarAvisoDemostracion("categorias", datos.fuente);
    configurarAreasDirectorio(estado.categorias);
    const huella = String(datos.catalogo.huella_sha256 || "");
    elementos.integridadCategorias.textContent = `Catálogo ${datos.catalogo.referencia} · versión ${datos.catalogo.version} · ${datos.catalogo.total} categorías · huella ${huella.slice(0, 16)}…`;
    elementos.integridadCategorias.setAttribute("aria-label", `Catálogo ${datos.catalogo.referencia}, versión ${datos.catalogo.version}, ${datos.catalogo.total} categorías, huella SHA-256 ${huella}`);
    elementos.integridadCategorias.title = `SHA-256 ${huella}`;
    if (estado.facetas) renderizarFacetas(estado.facetas);
    renderizarDirectorioFiltrado();
  }

  async function cargarDirectorioCategorias() {
    if (estado.controladorCategorias) estado.controladorCategorias.abort();
    estado.controladorCategorias = new AbortController();
    mostrarEstadoDirectorio("cargando");
    elementos.estadoDirectorio.textContent = "Cargando el catálogo profesional…";
    try {
      const datos = await obtenerJSON(API_CATEGORIAS, estado.controladorCategorias.signal);
      renderizarDirectorio(datos);
    } catch (error) {
      if (error.name === "AbortError") return;
      estado.categorias = [];
      elementos.estadoDirectorio.textContent = "El directorio no está disponible.";
      mostrarEstadoDirectorio("error");
    }
  }

  elementos.formulario.addEventListener("submit", (evento) => {
    evento.preventDefault();
    estado.pagina = 1;
    cerrarDetalle("none", false);
    cargarListado("push", false);
  });
  elementos.limpiar.addEventListener("click", () => {
    elementos.formulario.reset();
    estado.pagina = 1;
    cerrarDetalle("none", false);
    cargarListado("push", false);
  });
  elementos.reintentar.addEventListener("click", () => cargarListado("replace", true));
  elementos.anterior.addEventListener("click", () => {
    if (estado.pagina > 1) {
      estado.pagina -= 1;
      cargarListado("push", false);
    }
  });
  elementos.siguiente.addEventListener("click", () => {
    if (estado.pagina < estado.paginas) {
      estado.pagina += 1;
      cargarListado("push", false);
    }
  });
  elementos.cerrarDetalle.addEventListener("click", () => cerrarDetalle());
  elementos.reintentarDetalle.addEventListener("click", () => abrirDetalle(estado.convocatoria, true, "replace", false));
  elementos.buscarCategoria.addEventListener("input", renderizarDirectorioFiltrado);
  elementos.areaCategoria.addEventListener("change", renderizarDirectorioFiltrado);
  elementos.reintentarDirectorio.addEventListener("click", cargarDirectorioCategorias);
  window.addEventListener("popstate", () => {
    cargarFiltrosDesdeURL();
    if (!estado.convocatoria) reiniciarDetalleVisual();
    cargarListado("none", true);
  });

  configurarPreferencia("alternar-texto", "texto-grande");
  configurarPreferencia("alternar-contraste", "alto-contraste");
  cargarFiltrosDesdeURL();
  cargarListado("replace", true);
  cargarDirectorioCategorias();
})();
