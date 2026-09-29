import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "./comun/idioma.js";

const MENSAJES_CATEGORIAS = Object.freeze({
  es: Object.freeze({
    administracion_general: "Administración general", administracion_especial: "Administración especial",
    no_informada: "No informada", desde: "Desde {fecha}", hasta: "Hasta {fecha}", si: "Sí", no: "No",
    categorias_recibidas: "Categorías recibidas", version: "Versión", revision: "Revisión", estado_recibido: "Estado recibido",
    metadatos: "Metadatos recibidos del catálogo", identificador: "Identificador", nombre: "Nombre", estado: "Estado",
    huella: "Huella SHA-256", catalogo: "Catálogo", revision_fuente: "Revisión de la fuente", actualizada: "Actualizada en",
    titulo: "Categorías profesionales", descripcion: "Consulta del catálogo común recibido por Personal, Bolsa, RPT y certificados. Esta pantalla no modifica la fuente.",
    catalogo_recibido: "Catálogo recibido", solo_lectura: "Solo lectura. Las altas, cambios, publicaciones o retiradas deben realizarse mediante el futuro flujo gobernado y auditado.",
    catalogo_demo: "Catálogo marcado como demostración", fuente_lectura: "Fuente de solo lectura",
    aviso_demo: "La API ha marcado este contenido como demostración; la interfaz no le atribuye validez administrativa.",
    aviso_metadatos: "Se muestran exclusivamente los datos y metadatos recibidos; la interfaz no presupone aprobaciones ni vigencias.",
    aviso_sin_metadatos: "La API no ha aportado metadatos de aprobación, versión o revisión; la interfaz no los presupone.",
    respuesta_incompleta: "Respuesta incompleta: se han recibido {recibidas} de {total} categorías.",
    respuesta_incompleta_una: "Respuesta incompleta: se ha recibido {recibidas} de {total} categoría.",
    buscar: "Buscar por clave o denominación", area: "Área", todas_areas: "Todas las áreas",
    listado: "Listado de categorías profesionales", caption: "Categorías profesionales recibidas de la API",
    clave_estable: "Clave estable", denominacion: "Denominación", vigencia: "Vigencia", accion: "Acción",
    detalle_categoria: "Detalle de categoría", cerrar_detalle: "Cerrar detalle de categoría", cerrar: "Cerrar",
    sin_denominacion_categoria: "Categoría sin denominación", no_recibida: "No recibida", descripcion_campo: "Descripción",
    orden: "Orden", vigente_desde: "Vigente desde", vigente_hasta: "Vigente hasta", publicable: "Publicable",
    suscribible: "Suscribible", semantica: "Semántica", fuente: "Fuente", uso_declarado: "Uso declarado",
    mostrando: "Mostrando {mostradas} de {total} categorías recibidas", sin_coincidencias: "No hay categorías que coincidan con los filtros activos.",
    mostrando_una: "Mostrando {mostradas} de {total} categoría recibida",
    sin_categorias: "La API no ha devuelto categorías profesionales.", sin_clave: "Sin clave recibida", sin_denominacion: "Sin denominación",
    no_informado: "No informado", ver_detalle: "Ver detalle", ver_detalle_de: "Ver detalle de {categoria}", categoria_sin_identificar: "categoría sin identificar",
    estado_vigente: "Vigente", estado_activo: "Activo", estado_borrador: "Borrador", estado_retirado: "Retirado", estado_no_vigente: "No vigente",
  }),
  en: Object.freeze({
    administracion_general: "General administration", administracion_especial: "Specialist administration",
    no_informada: "Not provided", desde: "From {fecha}", hasta: "Until {fecha}", si: "Yes", no: "No",
    categorias_recibidas: "Categories received", version: "Version", revision: "Revision", estado_recibido: "Status received",
    metadatos: "Catalogue metadata received", identificador: "Identifier", nombre: "Name", estado: "Status",
    huella: "SHA-256 digest", catalogo: "Catalogue", revision_fuente: "Source revision", actualizada: "Updated on",
    titulo: "Job categories", descripcion: "View the shared catalogue received by Personnel, Bolsa, RPT and certificates. This screen does not change the source.",
    catalogo_recibido: "Catalogue received", solo_lectura: "Read only. New entries, changes, publication and withdrawal require the future governed, audited process.",
    catalogo_demo: "Catalogue marked as a demonstration", fuente_lectura: "Read-only source",
    aviso_demo: "The API has marked this content as a demonstration; the interface does not treat it as administratively valid.",
    aviso_metadatos: "Only received data and metadata are shown; the interface makes no assumptions about approval or validity.",
    aviso_sin_metadatos: "The API did not provide approval, version or revision metadata; the interface makes no assumptions about them.",
    respuesta_incompleta: "Incomplete response: {recibidas} of {total} categories received.",
    respuesta_incompleta_una: "Incomplete response: {recibidas} of {total} category received.",
    buscar: "Search by key or name", area: "Area", todas_areas: "All areas",
    listado: "List of job categories", caption: "Job categories received from the API",
    clave_estable: "Stable key", denominacion: "Name", vigencia: "Validity", accion: "Action",
    detalle_categoria: "Category details", cerrar_detalle: "Close category details", cerrar: "Close",
    sin_denominacion_categoria: "Category without a name", no_recibida: "Not received", descripcion_campo: "Description",
    orden: "Order", vigente_desde: "Valid from", vigente_hasta: "Valid until", publicable: "Publishable",
    suscribible: "Subscribable", semantica: "Meaning", fuente: "Source", uso_declarado: "Declared use",
    mostrando: "Showing {mostradas} of {total} categories received", sin_coincidencias: "No categories match the active filters.",
    mostrando_una: "Showing {mostradas} of {total} category received",
    sin_categorias: "The API returned no job categories.", sin_clave: "No key received", sin_denominacion: "No name received",
    no_informado: "Not provided", ver_detalle: "View details", ver_detalle_de: "View details for {categoria}", categoria_sin_identificar: "unidentified category",
    estado_vigente: "Valid", estado_activo: "Active", estado_borrador: "Draft", estado_retirado: "Withdrawn", estado_no_vigente: "No longer valid",
  }),
});

export function traducirCatalogoCategorias(clave, variables = {}, idioma = IDIOMA_ACTUAL) {
  const plantilla = MENSAJES_CATEGORIAS[idioma]?.[clave] ?? MENSAJES_CATEGORIAS.es[clave];
  if (plantilla === undefined) throw new Error(`Missing category catalogue key: ${clave}`);
  return plantilla.replace(/\{(\w+)\}/g, (_, nombre) => String(variables[nombre] ?? ""));
}

export function normalizarPaginaCategoriasProfesionales(valor) {
  const listaDirecta = Array.isArray(valor) ? valor : null;
  const envoltorio = valor && typeof valor === "object" && !Array.isArray(valor) ? valor : {};
  const pagina = envoltorio.categories && typeof envoltorio.categories === "object" && !Array.isArray(envoltorio.categories)
    ? envoltorio.categories
    : envoltorio;
  const items = listaDirecta || (Array.isArray(pagina.items)
    ? pagina.items
    : (Array.isArray(pagina.entradas) ? pagina.entradas : (Array.isArray(pagina.categorias) ? pagina.categorias : [])));
  const catalogo = envoltorio.catalogo || envoltorio.catalog || pagina.catalogo || pagina.catalog || null;
  const fuente = envoltorio.fuente || pagina.fuente || null;
  const totalRecibido = Number(pagina.total ?? catalogo?.total ?? items.length);
  return {
    items,
    total: Number.isFinite(totalRecibido) ? totalRecibido : items.length,
    limit: Number(pagina.limit || 0),
    offset: Number(pagina.offset || 0),
    catalogo,
    catalog: catalogo,
    fuente,
    aviso: String(envoltorio.aviso || pagina.aviso || fuente?.aviso || "").trim(),
    demostracion: envoltorio.demostracion === true || pagina.demostracion === true || fuente?.demostracion === true,
  };
}

export function ofertaPerteneceAlCatalogoActual(oferta, clavesVigentes, referencia) {
  const claves = clavesVigentes instanceof Set ? clavesVigentes : new Set(clavesVigentes || []);
  const huellaEsperada = String(referencia?.huellaSHA256 || "").toLowerCase();
  return Boolean(
    referencia?.id
    && Number(referencia.version) > 0
    && /^[a-f0-9]{64}$/.test(huellaEsperada)
    && claves.has(oferta?.categoryKey)
    && oferta.categoryCatalogID === referencia.id
    && Number(oferta.categoryCatalogVersion) === Number(referencia.version)
    && String(oferta.categoryCatalogSHA256 || "").toLowerCase() === huellaEsperada
  );
}

export function crearHerramientasCatalogoCategorias(dependencias, idioma = IDIOMA_ACTUAL) {
  const { $, $$, stateTone, screenHead, getPersonalCatalog, workTableTextCollator } = dependencias;
  const localizacion = idioma === IDIOMA_ACTUAL ? LOCALIZACION_ACTUAL : (idioma === "en" ? "en-GB" : "es-ES");
  const t = (clave, variables) => traducirCatalogoCategorias(clave, variables, idioma);
  const formatoNumero = new Intl.NumberFormat(localizacion);
  const numero = (valor) => formatoNumero.format(valor);
  const collator = idioma === "en" ? new Intl.Collator(localizacion, { sensitivity: "base" }) : workTableTextCollator;

  function normalizarTextoBusquedaCatalogo(valor) {
    return String(valor || "")
      .normalize("NFD")
      .replace(/[\u0300-\u036f]/g, "")
      .toLocaleLowerCase("es")
      .trim();
  }

  function normalizarCategoriaProfesional(valor, indice = 0) {
    const categoria = valor && typeof valor === "object" && !Array.isArray(valor) ? valor : {};
    const atributos = categoria.atributos && typeof categoria.atributos === "object" && !Array.isArray(categoria.atributos)
      ? categoria.atributos
      : {};
    const clave = String(categoria.clave ?? categoria.slug ?? "").trim();
    const etiqueta = String(categoria.etiqueta ?? categoria.name ?? categoria.label ?? clave).trim();
    const ordenRecibido = categoria.orden == null ? Number.NaN : Number(categoria.orden);
    return {
      clave,
      etiqueta,
      descripcion: String(categoria.descripcion ?? categoria.description ?? "").trim(),
      orden: Number.isFinite(ordenRecibido) ? ordenRecibido : null,
      area: String(categoria.area ?? atributos.area ?? "").trim(),
      areaEtiqueta: String(categoria.area_etiqueta ?? atributos.area_etiqueta ?? "").trim(),
      estado: String(categoria.estado ?? categoria.state ?? "").trim(),
      fuente: String(categoria.fuente ?? categoria.source ?? "").trim(),
      uso: String(categoria.uso ?? categoria.usage ?? "").trim(),
      vigenteDesde: String(categoria.vigente_desde ?? categoria.valid_from ?? "").trim(),
      vigenteHasta: String(categoria.vigente_hasta ?? categoria.valid_until ?? "").trim(),
      publicable: categoria.publicable ?? atributos.publicable,
      suscribible: categoria.suscribible ?? atributos.suscribible,
      semantica: String(categoria.semantica ?? atributos.semantica ?? "").trim(),
      indice,
    };
  }

  function obtenerCategoriasProfesionales(view) {
    const items = getPersonalCatalog(view).categories?.items || [];
    return items
      .map(normalizarCategoriaProfesional)
      .sort((a, b) => (a.orden ?? a.indice) - (b.orden ?? b.indice) || collator.compare(a.etiqueta, b.etiqueta));
  }

  function metadatosCatalogoCategorias(view) {
    const pagina = getPersonalCatalog(view).categories || {};
    const totalRecibido = Number(pagina.total ?? pagina.items?.length ?? 0);
    return {
      total: Number.isFinite(totalRecibido) ? totalRecibido : (pagina.items?.length || 0),
      catalogo: pagina.catalogo || pagina.catalog || null,
      fuente: pagina.fuente || null,
      aviso: String(pagina.aviso || pagina.fuente?.aviso || "").trim(),
      demostracion: pagina.demostracion === true || pagina.fuente?.demostracion === true,
    };
  }

  function referenciaCatalogoCategorias(view) {
    const catalogo = metadatosCatalogoCategorias(view).catalogo || {};
    return {
      id: String(catalogo.catalogo_id ?? catalogo.id ?? "").trim(),
      version: Number(catalogo.catalogo_version ?? catalogo.version ?? 0),
      huellaSHA256: String(catalogo.catalogo_huella_sha256 ?? catalogo.huella_sha256 ?? "").trim(),
    };
  }

  function etiquetaAreaCategoria(categoria) {
    if (categoria.areaEtiqueta) return categoria.areaEtiqueta;
    const etiquetasConocidas = {
      administracion_general: t("administracion_general"),
      administracion_especial: t("administracion_especial"),
    };
    return etiquetasConocidas[categoria.area]
      || categoria.area.replaceAll("_", " ").replace(/^./, (letra) => letra.toLocaleUpperCase("es"))
      || t("no_informada");
  }

  function formatearFechaCatalogo(valor) {
    if (!valor) return "";
    const fecha = new Date(valor);
    if (Number.isNaN(fecha.getTime())) return String(valor);
    return new Intl.DateTimeFormat(localizacion, { dateStyle: "medium" }).format(fecha);
  }

  function vigenciaCategoria(categoria) {
    const desde = formatearFechaCatalogo(categoria.vigenteDesde);
    const hasta = formatearFechaCatalogo(categoria.vigenteHasta);
    if (desde && hasta) return `${desde} - ${hasta}`;
    if (desde) return t("desde", { fecha: desde });
    if (hasta) return t("hasta", { fecha: hasta });
    return t("no_informada");
  }

  function etiquetaBooleanoCatalogo(valor) {
    if (valor === true || /^(si|sí|true|1)$/i.test(String(valor || ""))) return t("si");
    if (valor === false || /^(no|false|0)$/i.test(String(valor || ""))) return t("no");
    return String(valor || "").trim();
  }

  function estadoCategoria(valor) {
    const clave = `estado_${String(valor || "").toLocaleLowerCase("es").trim().replace(/[\s-]+/g, "_")}`;
    return MENSAJES_CATEGORIAS.es[clave] ? t(clave) : valor;
  }

  function agregarCampoDefinicion(lista, etiqueta, valor) {
    if (valor == null || String(valor).trim() === "") return;
    const grupo = document.createElement("div");
    const termino = document.createElement("dt");
    const detalle = document.createElement("dd");
    termino.textContent = etiqueta;
    detalle.textContent = String(valor);
    grupo.append(termino, detalle);
    lista.append(grupo);
  }

  function resumenCatalogoCategorias(panel, categorias, metadatos) {
    const catalogo = metadatos.catalogo && typeof metadatos.catalogo === "object" ? metadatos.catalogo : {};
    const general = categorias.filter((categoria) => categoria.area === "administracion_general").length;
    const especial = categorias.filter((categoria) => categoria.area === "administracion_especial").length;
    const valores = [
      [t("categorias_recibidas"), metadatos.total],
      [t("administracion_general"), general],
      [t("administracion_especial"), especial],
    ];
    const versionCatalogo = catalogo.version ?? catalogo.catalogo_version;
    if (versionCatalogo != null) valores.push([t("version"), versionCatalogo]);
    if (catalogo.revision != null) valores.push([t("revision"), catalogo.revision]);
    if (catalogo.estado || catalogo.state) valores.push([t("estado_recibido"), estadoCategoria(catalogo.estado || catalogo.state)]);
    valores.forEach(([etiqueta, valor]) => {
      const grupo = document.createElement("div");
      const termino = document.createElement("dt");
      const detalle = document.createElement("dd");
      termino.textContent = etiqueta;
      detalle.textContent = typeof valor === "number" ? numero(valor) : String(valor);
      grupo.append(termino, detalle);
      panel.append(grupo);
    });
  }

  function metadatosRecibidosCatalogo(metadatos) {
    const catalogo = metadatos.catalogo;
    const fuente = metadatos.fuente;
    const hayCatalogo = catalogo && (typeof catalogo !== "object" || Object.keys(catalogo).length > 0);
    const hayFuente = fuente && typeof fuente === "object" && Object.keys(fuente).length > 0;
    if (!hayCatalogo && !hayFuente) return null;
    const details = document.createElement("details");
    details.className = "category-catalog-metadata";
    const summary = document.createElement("summary");
    summary.textContent = t("metadatos");
    const lista = document.createElement("dl");
    if (catalogo && typeof catalogo === "object") {
      agregarCampoDefinicion(lista, t("identificador"), catalogo.id || catalogo.catalogo_id || catalogo.referencia);
      agregarCampoDefinicion(lista, t("nombre"), catalogo.nombre || catalogo.name);
      agregarCampoDefinicion(lista, t("version"), catalogo.version ?? catalogo.catalogo_version);
      agregarCampoDefinicion(lista, t("revision"), catalogo.revision);
      agregarCampoDefinicion(lista, t("estado"), estadoCategoria(catalogo.estado || catalogo.state));
      agregarCampoDefinicion(lista, t("huella"), catalogo.huella_sha256 || catalogo.catalogo_huella_sha256 || catalogo.huella);
    } else {
      agregarCampoDefinicion(lista, t("catalogo"), catalogo);
    }
    if (hayFuente) {
      agregarCampoDefinicion(lista, t("revision_fuente"), fuente.revision);
      agregarCampoDefinicion(lista, t("actualizada"), formatearFechaCatalogo(fuente.actualizada_en || fuente.updated_at));
    }
    details.append(summary, lista);
    return details;
  }

  function renderCatalogoCategoriasGobernado(target, screen, view) {
    const categorias = obtenerCategoriasProfesionales(view);
    const metadatos = metadatosCatalogoCategorias(view);
    const cabeceraPantalla = screenHead(t("titulo"), t("descripcion"), []);
    cabeceraPantalla.classList.add("category-catalog-screen-head");
    target.append(cabeceraPantalla);

    const panel = document.createElement("section");
    panel.className = "governed-category-catalog";
    panel.dataset.governedCategoryCatalog = "true";
    panel.setAttribute("aria-labelledby", "category-catalog-heading");

    const encabezado = document.createElement("div");
    encabezado.className = "category-catalog-heading";
    encabezado.innerHTML = `<div><h3 id="category-catalog-heading"></h3><p></p></div>`;
    $("h3", encabezado).textContent = t("catalogo_recibido");
    $("p", encabezado).textContent = t("solo_lectura");

    const aviso = document.createElement("div");
    aviso.className = `category-catalog-notice${metadatos.demostracion ? " is-demo" : ""}`;
    aviso.dataset.categorySourceNotice = "true";
    aviso.setAttribute("role", "status");
    const avisoTitulo = document.createElement("strong");
    avisoTitulo.textContent = metadatos.demostracion ? t("catalogo_demo") : t("fuente_lectura");
    const avisoTexto = document.createElement("span");
    const contieneMetadatos = Boolean(metadatos.catalogo || metadatos.fuente);
    avisoTexto.textContent = metadatos.aviso
      || (metadatos.demostracion
        ? t("aviso_demo")
        : (contieneMetadatos
          ? t("aviso_metadatos")
          : t("aviso_sin_metadatos")));
    aviso.append(avisoTitulo, avisoTexto);
    if (metadatos.total > categorias.length) {
      const paginacion = document.createElement("span");
      paginacion.textContent = t(metadatos.total === 1 ? "respuesta_incompleta_una" : "respuesta_incompleta", { recibidas: numero(categorias.length), total: numero(metadatos.total) });
      aviso.append(paginacion);
    }

    const resumen = document.createElement("dl");
    resumen.className = "category-catalog-summary";
    resumen.dataset.categorySummary = "true";
    resumenCatalogoCategorias(resumen, categorias, metadatos);

    const filtros = document.createElement("form");
    filtros.className = "category-catalog-filters";
    filtros.dataset.categoryFilters = "true";
    filtros.setAttribute("role", "search");
    filtros.innerHTML = `
      <label for="category-catalog-search">${t("buscar")}
        <input id="category-catalog-search" type="search" autocomplete="off" data-category-search>
      </label>
      <label for="category-catalog-area">${t("area")}
        <select id="category-catalog-area" data-category-area>
          <option value="">${t("todas_areas")}</option>
        </select>
      </label>
      <p data-category-results role="status" aria-live="polite"></p>
    `;
    filtros.addEventListener("submit", (event) => event.preventDefault());
    const buscador = $("[data-category-search]", filtros);
    const selectorArea = $("[data-category-area]", filtros);
    const resultado = $("[data-category-results]", filtros);
    const areas = new Map();
    categorias.forEach((categoria) => {
      if (categoria.area && !areas.has(categoria.area)) areas.set(categoria.area, etiquetaAreaCategoria(categoria));
    });
    Array.from(areas.entries())
      .sort((a, b) => collator.compare(a[1], b[1]))
      .forEach(([valor, etiqueta]) => {
        const option = document.createElement("option");
        option.value = valor;
        option.textContent = etiqueta;
        selectorArea.append(option);
      });

    const cuerpo = document.createElement("div");
    cuerpo.className = "category-catalog-body";
    const tablaRegion = document.createElement("div");
    tablaRegion.className = "category-catalog-table-wrap";
    tablaRegion.setAttribute("role", "region");
    tablaRegion.setAttribute("aria-label", t("listado"));
    tablaRegion.tabIndex = 0;
    const tabla = document.createElement("table");
    tabla.dataset.categoryTable = "true";
    tabla.innerHTML = `
      <caption>${t("caption")}</caption>
      <thead><tr>
        <th scope="col">${t("clave_estable")}</th>
        <th scope="col">${t("denominacion")}</th>
        <th scope="col">${t("area")}</th>
        <th scope="col">${t("estado")}</th>
        <th scope="col">${t("vigencia")}</th>
        <th scope="col">${t("accion")}</th>
      </tr></thead>
      <tbody></tbody>
    `;
    tablaRegion.append(tabla);

    const detalle = document.createElement("aside");
    detalle.className = "category-catalog-detail";
    detalle.dataset.categoryDetail = "true";
    detalle.hidden = true;
    detalle.setAttribute("aria-labelledby", "category-detail-title");
    detalle.innerHTML = `
      <header><div><span class="eyebrow">${t("detalle_categoria")}</span><h3 id="category-detail-title" tabindex="-1"></h3></div>
        <button type="button" class="quiet-action" data-category-detail-close aria-label="${t("cerrar_detalle")}">${t("cerrar")}</button>
      </header>
      <dl></dl>
    `;
    cuerpo.append(tablaRegion, detalle);

    let ultimoDisparador = null;
    const cerrarDetalle = (restaurarFoco = true) => {
      if (detalle.hidden) return;
      detalle.hidden = true;
      cuerpo.classList.remove("has-detail");
      $$(`tr[aria-selected="true"]`, tabla).forEach((fila) => fila.setAttribute("aria-selected", "false"));
      if (restaurarFoco && ultimoDisparador?.isConnected) ultimoDisparador.focus();
    };
    const abrirDetalle = (categoria, boton, fila) => {
      ultimoDisparador = boton;
      const titulo = $("#category-detail-title", detalle);
      const lista = $("dl", detalle);
      titulo.textContent = categoria.etiqueta || t("sin_denominacion_categoria");
      lista.replaceChildren();
      agregarCampoDefinicion(lista, t("clave_estable"), categoria.clave || t("no_recibida"));
      agregarCampoDefinicion(lista, t("denominacion"), categoria.etiqueta || t("no_recibida"));
      agregarCampoDefinicion(lista, t("descripcion_campo"), categoria.descripcion);
      agregarCampoDefinicion(lista, t("area"), etiquetaAreaCategoria(categoria));
      agregarCampoDefinicion(lista, t("orden"), categoria.orden);
      agregarCampoDefinicion(lista, t("estado"), estadoCategoria(categoria.estado));
      agregarCampoDefinicion(lista, t("vigente_desde"), formatearFechaCatalogo(categoria.vigenteDesde));
      agregarCampoDefinicion(lista, t("vigente_hasta"), formatearFechaCatalogo(categoria.vigenteHasta));
      agregarCampoDefinicion(lista, t("publicable"), etiquetaBooleanoCatalogo(categoria.publicable));
      agregarCampoDefinicion(lista, t("suscribible"), etiquetaBooleanoCatalogo(categoria.suscribible));
      agregarCampoDefinicion(lista, t("semantica"), categoria.semantica);
      agregarCampoDefinicion(lista, t("fuente"), categoria.fuente);
      agregarCampoDefinicion(lista, t("uso_declarado"), categoria.uso);
      detalle.hidden = false;
      cuerpo.classList.add("has-detail");
      $$(`tr[aria-selected="true"]`, tabla).forEach((actual) => actual.setAttribute("aria-selected", "false"));
      fila.setAttribute("aria-selected", "true");
      titulo.focus();
    };
    $("[data-category-detail-close]", detalle).addEventListener("click", () => cerrarDetalle());
    panel.addEventListener("keydown", (event) => {
      if (event.key === "Escape" && !detalle.hidden) {
        event.preventDefault();
        cerrarDetalle();
      }
    });

    const renderFilas = () => {
      cerrarDetalle(false);
      const consulta = normalizarTextoBusquedaCatalogo(buscador.value);
      const area = selectorArea.value;
      const filtradas = categorias.filter((categoria) => {
        if (area && categoria.area !== area) return false;
        if (!consulta) return true;
        const texto = normalizarTextoBusquedaCatalogo([
          categoria.clave,
          categoria.etiqueta,
          categoria.descripcion,
          etiquetaAreaCategoria(categoria),
        ].join(" "));
        return texto.includes(consulta);
      });
      resultado.textContent = t(categorias.length === 1 ? "mostrando_una" : "mostrando", { mostradas: numero(filtradas.length), total: numero(categorias.length) });
      const tbody = $("tbody", tabla);
      tbody.replaceChildren();
      if (!filtradas.length) {
        const fila = document.createElement("tr");
        const celda = document.createElement("td");
        celda.colSpan = 6;
        celda.className = "empty-state";
        celda.textContent = categorias.length
          ? t("sin_coincidencias")
          : t("sin_categorias");
        fila.append(celda);
        tbody.append(fila);
        return;
      }
      filtradas.forEach((categoria) => {
        const fila = document.createElement("tr");
        fila.dataset.categoryRow = "true";
        if (categoria.clave) fila.dataset.categoryKey = categoria.clave;
        fila.setAttribute("aria-selected", "false");
        const valores = [
          [t("clave_estable"), categoria.clave || t("sin_clave")],
          [t("denominacion"), categoria.etiqueta || t("sin_denominacion")],
          [t("area"), etiquetaAreaCategoria(categoria)],
        ];
        valores.forEach(([etiqueta, valor]) => {
          const celda = document.createElement("td");
          celda.dataset.label = etiqueta;
          celda.textContent = valor;
          fila.append(celda);
        });
        const estadoCelda = document.createElement("td");
        estadoCelda.dataset.label = t("estado");
        const estado = document.createElement("span");
        estado.className = `status-chip ${categoria.estado ? stateTone(categoria.estado) : "chip-slate"}`;
        estado.textContent = categoria.estado ? estadoCategoria(categoria.estado) : t("no_informado");
        estadoCelda.append(estado);
        fila.append(estadoCelda);
        const vigenciaCelda = document.createElement("td");
        vigenciaCelda.dataset.label = t("vigencia");
        vigenciaCelda.textContent = vigenciaCategoria(categoria);
        fila.append(vigenciaCelda);
        const accionCelda = document.createElement("td");
        accionCelda.dataset.label = t("accion");
        const boton = document.createElement("button");
        boton.type = "button";
        boton.className = "row-action";
        boton.dataset.categoryView = "true";
        boton.textContent = t("ver_detalle");
        boton.setAttribute("aria-label", t("ver_detalle_de", { categoria: categoria.etiqueta || categoria.clave || t("categoria_sin_identificar") }));
        boton.addEventListener("click", () => abrirDetalle(categoria, boton, fila));
        accionCelda.append(boton);
        fila.append(accionCelda);
        tbody.append(fila);
      });
    };
    buscador.addEventListener("input", renderFilas);
    selectorArea.addEventListener("change", renderFilas);
    renderFilas();

    panel.append(encabezado, aviso, resumen, filtros, cuerpo);
    const metadatosPanel = metadatosRecibidosCatalogo(metadatos);
    if (metadatosPanel) panel.append(metadatosPanel);
    target.append(panel);
  }

  return {
    obtenerCategoriasProfesionales,
    referenciaCatalogoCategorias,
    renderCatalogoCategoriasGobernado,
  };
}
