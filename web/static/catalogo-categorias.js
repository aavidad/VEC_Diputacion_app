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

export function crearHerramientasCatalogoCategorias(dependencias) {
  const { $, $$, formatCount, stateTone, screenHead, getPersonalCatalog, workTableTextCollator } = dependencias;

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
      .sort((a, b) => (a.orden ?? a.indice) - (b.orden ?? b.indice) || workTableTextCollator.compare(a.etiqueta, b.etiqueta));
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
      administracion_general: "Administración general",
      administracion_especial: "Administración especial",
    };
    return etiquetasConocidas[categoria.area]
      || categoria.area.replaceAll("_", " ").replace(/^./, (letra) => letra.toLocaleUpperCase("es"))
      || "No informada";
  }

  function formatearFechaCatalogo(valor) {
    if (!valor) return "";
    const fecha = new Date(valor);
    if (Number.isNaN(fecha.getTime())) return String(valor);
    return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium" }).format(fecha);
  }

  function vigenciaCategoria(categoria) {
    const desde = formatearFechaCatalogo(categoria.vigenteDesde);
    const hasta = formatearFechaCatalogo(categoria.vigenteHasta);
    if (desde && hasta) return `${desde} - ${hasta}`;
    if (desde) return `Desde ${desde}`;
    if (hasta) return `Hasta ${hasta}`;
    return "No informada";
  }

  function etiquetaBooleanoCatalogo(valor) {
    if (valor === true || /^(si|sí|true|1)$/i.test(String(valor || ""))) return "Sí";
    if (valor === false || /^(no|false|0)$/i.test(String(valor || ""))) return "No";
    return String(valor || "").trim();
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
      ["Categorías recibidas", metadatos.total],
      ["Administración general", general],
      ["Administración especial", especial],
    ];
    const versionCatalogo = catalogo.version ?? catalogo.catalogo_version;
    if (versionCatalogo != null) valores.push(["Versión", versionCatalogo]);
    if (catalogo.revision != null) valores.push(["Revisión", catalogo.revision]);
    if (catalogo.estado || catalogo.state) valores.push(["Estado recibido", catalogo.estado || catalogo.state]);
    valores.forEach(([etiqueta, valor]) => {
      const grupo = document.createElement("div");
      const termino = document.createElement("dt");
      const detalle = document.createElement("dd");
      termino.textContent = etiqueta;
      detalle.textContent = typeof valor === "number" ? formatCount(valor) : String(valor);
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
    summary.textContent = "Metadatos recibidos del catálogo";
    const lista = document.createElement("dl");
    if (catalogo && typeof catalogo === "object") {
      agregarCampoDefinicion(lista, "Identificador", catalogo.id || catalogo.catalogo_id || catalogo.referencia);
      agregarCampoDefinicion(lista, "Nombre", catalogo.nombre || catalogo.name);
      agregarCampoDefinicion(lista, "Versión", catalogo.version ?? catalogo.catalogo_version);
      agregarCampoDefinicion(lista, "Revisión", catalogo.revision);
      agregarCampoDefinicion(lista, "Estado", catalogo.estado || catalogo.state);
      agregarCampoDefinicion(lista, "Huella SHA-256", catalogo.huella_sha256 || catalogo.catalogo_huella_sha256 || catalogo.huella);
    } else {
      agregarCampoDefinicion(lista, "Catálogo", catalogo);
    }
    if (hayFuente) {
      agregarCampoDefinicion(lista, "Revisión de la fuente", fuente.revision);
      agregarCampoDefinicion(lista, "Actualizada en", fuente.actualizada_en || fuente.updated_at);
    }
    details.append(summary, lista);
    return details;
  }

  function renderCatalogoCategoriasGobernado(target, screen, view) {
    const categorias = obtenerCategoriasProfesionales(view);
    const metadatos = metadatosCatalogoCategorias(view);
    const cabeceraPantalla = screenHead(
      "Categorías profesionales",
      "Consulta del catálogo común recibido por Personal, Bolsa, RPT y certificados. Esta pantalla no modifica la fuente.",
      [],
    );
    cabeceraPantalla.classList.add("category-catalog-screen-head");
    target.append(cabeceraPantalla);

    const panel = document.createElement("section");
    panel.className = "governed-category-catalog";
    panel.dataset.governedCategoryCatalog = "true";
    panel.setAttribute("aria-labelledby", "category-catalog-heading");

    const encabezado = document.createElement("div");
    encabezado.className = "category-catalog-heading";
    encabezado.innerHTML = `<div><h3 id="category-catalog-heading">Catálogo recibido</h3><p></p></div>`;
    $("p", encabezado).textContent = "Solo lectura. Las altas, cambios, publicaciones o retiradas deben realizarse mediante el futuro flujo gobernado y auditado.";

    const aviso = document.createElement("div");
    aviso.className = `category-catalog-notice${metadatos.demostracion ? " is-demo" : ""}`;
    aviso.dataset.categorySourceNotice = "true";
    aviso.setAttribute("role", "status");
    const avisoTitulo = document.createElement("strong");
    avisoTitulo.textContent = metadatos.demostracion ? "Catálogo marcado como demostración" : "Fuente de solo lectura";
    const avisoTexto = document.createElement("span");
    const contieneMetadatos = Boolean(metadatos.catalogo || metadatos.fuente);
    avisoTexto.textContent = metadatos.aviso
      || (metadatos.demostracion
        ? "La API ha marcado este contenido como demostración; la interfaz no le atribuye validez administrativa."
        : (contieneMetadatos
          ? "Se muestran exclusivamente los datos y metadatos recibidos; la interfaz no presupone aprobaciones ni vigencias."
          : "La API no ha aportado metadatos de aprobación, versión o revisión; la interfaz no los presupone."));
    aviso.append(avisoTitulo, avisoTexto);
    if (metadatos.total > categorias.length) {
      const paginacion = document.createElement("span");
      paginacion.textContent = `Respuesta incompleta: se han recibido ${formatCount(categorias.length)} de ${formatCount(metadatos.total)} categorías.`;
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
      <label for="category-catalog-search">Buscar por clave o denominación
        <input id="category-catalog-search" type="search" autocomplete="off" data-category-search>
      </label>
      <label for="category-catalog-area">Área
        <select id="category-catalog-area" data-category-area>
          <option value="">Todas las áreas</option>
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
      .sort((a, b) => workTableTextCollator.compare(a[1], b[1]))
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
    tablaRegion.setAttribute("aria-label", "Listado de categorías profesionales");
    tablaRegion.tabIndex = 0;
    const tabla = document.createElement("table");
    tabla.dataset.categoryTable = "true";
    tabla.innerHTML = `
      <caption>Categorías profesionales recibidas de la API</caption>
      <thead><tr>
        <th scope="col">Clave estable</th>
        <th scope="col">Denominación</th>
        <th scope="col">Área</th>
        <th scope="col">Estado</th>
        <th scope="col">Vigencia</th>
        <th scope="col">Acción</th>
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
      <header><div><span class="eyebrow">Detalle de categoría</span><h3 id="category-detail-title" tabindex="-1"></h3></div>
        <button type="button" class="quiet-action" data-category-detail-close aria-label="Cerrar detalle de categoría">Cerrar</button>
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
      titulo.textContent = categoria.etiqueta || "Categoría sin denominación";
      lista.replaceChildren();
      agregarCampoDefinicion(lista, "Clave estable", categoria.clave || "No recibida");
      agregarCampoDefinicion(lista, "Denominación", categoria.etiqueta || "No recibida");
      agregarCampoDefinicion(lista, "Descripción", categoria.descripcion);
      agregarCampoDefinicion(lista, "Área", etiquetaAreaCategoria(categoria));
      agregarCampoDefinicion(lista, "Orden", categoria.orden);
      agregarCampoDefinicion(lista, "Estado", categoria.estado);
      agregarCampoDefinicion(lista, "Vigente desde", formatearFechaCatalogo(categoria.vigenteDesde));
      agregarCampoDefinicion(lista, "Vigente hasta", formatearFechaCatalogo(categoria.vigenteHasta));
      agregarCampoDefinicion(lista, "Publicable", etiquetaBooleanoCatalogo(categoria.publicable));
      agregarCampoDefinicion(lista, "Suscribible", etiquetaBooleanoCatalogo(categoria.suscribible));
      agregarCampoDefinicion(lista, "Semántica", categoria.semantica);
      agregarCampoDefinicion(lista, "Fuente", categoria.fuente);
      agregarCampoDefinicion(lista, "Uso declarado", categoria.uso);
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
      resultado.textContent = `Mostrando ${formatCount(filtradas.length)} de ${formatCount(categorias.length)} categorías recibidas`;
      const tbody = $("tbody", tabla);
      tbody.replaceChildren();
      if (!filtradas.length) {
        const fila = document.createElement("tr");
        const celda = document.createElement("td");
        celda.colSpan = 6;
        celda.className = "empty-state";
        celda.textContent = categorias.length
          ? "No hay categorías que coincidan con los filtros activos."
          : "La API no ha devuelto categorías profesionales.";
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
          ["Clave estable", categoria.clave || "Sin clave recibida"],
          ["Denominación", categoria.etiqueta || "Sin denominación"],
          ["Área", etiquetaAreaCategoria(categoria)],
        ];
        valores.forEach(([etiqueta, valor]) => {
          const celda = document.createElement("td");
          celda.dataset.label = etiqueta;
          celda.textContent = valor;
          fila.append(celda);
        });
        const estadoCelda = document.createElement("td");
        estadoCelda.dataset.label = "Estado";
        const estado = document.createElement("span");
        estado.className = `status-chip ${categoria.estado ? stateTone(categoria.estado) : "chip-slate"}`;
        estado.textContent = categoria.estado || "No informado";
        estadoCelda.append(estado);
        fila.append(estadoCelda);
        const vigenciaCelda = document.createElement("td");
        vigenciaCelda.dataset.label = "Vigencia";
        vigenciaCelda.textContent = vigenciaCategoria(categoria);
        fila.append(vigenciaCelda);
        const accionCelda = document.createElement("td");
        accionCelda.dataset.label = "Acción";
        const boton = document.createElement("button");
        boton.type = "button";
        boton.className = "row-action";
        boton.dataset.categoryView = "true";
        boton.textContent = "Ver detalle";
        boton.setAttribute("aria-label", `Ver detalle de ${categoria.etiqueta || categoria.clave || "categoría sin identificar"}`);
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
