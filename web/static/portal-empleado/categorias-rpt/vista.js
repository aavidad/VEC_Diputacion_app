const SOLO_LECTURA = Object.freeze(["data-i18n", "data-i18n-label"]);
export const TAMANO_PAGINA = 20;

function traducirFijos(doc, t) {
  for (const atributo of SOLO_LECTURA) {
    doc.querySelectorAll(`[${atributo}]`).forEach((elemento) => {
      const texto = t(elemento.getAttribute(atributo));
      if (atributo === "data-i18n") elemento.textContent = texto;
      else elemento.setAttribute("aria-label", texto);
    });
  }
  doc.title = t("tituloDocumento");
}

function textoEn(doc, contenido, idiomaDatos, idiomaUI) {
  const nodo = doc.createElement("span");
  nodo.textContent = contenido;
  if (idiomaDatos !== idiomaUI) nodo.lang = idiomaDatos;
  return nodo;
}

export function filtrarCategorias(categorias, consulta = "", grupo = "", localizacion) {
  const buscado = consulta.trim().toLocaleLowerCase(localizacion);
  return categorias.filter((categoria) =>
    (!grupo || categoria.grupos_subgrupos.some((g) => g.clave === grupo))
    && (!buscado || [categoria.etiqueta, ...categoria.grupos_subgrupos.map((g) => g.etiqueta)]
      .some((valor) => valor.toLocaleLowerCase(localizacion).includes(buscado))));
}

/** Montaje de solo lectura. Los dos controles de cambio permanecen inertes en HTML. */
export function montarVistaCategorias({ doc, puerto, t, idiomaUI, idiomaDatos, localizacion }) {
  if (!doc || typeof puerto?.listarOpciones !== "function" || typeof t !== "function") {
    throw new TypeError("montaje de categorías no disponible");
  }
  doc.documentElement.lang = idiomaUI;
  doc.querySelectorAll('a[href="/portal-empleado/"]').forEach((enlace) =>
    enlace.setAttribute("href", `/portal-empleado/?lang=${encodeURIComponent(idiomaUI)}`));
  traducirFijos(doc, t);

  const buscar = doc.getElementById("buscar");
  const grupo = doc.getElementById("grupo");
  const estado = doc.getElementById("estado");
  const reintentar = doc.getElementById("reintentar");
  const tabla = doc.getElementById("tabla-contenedor");
  const filas = doc.getElementById("filas");
  const cuenta = doc.getElementById("cuenta");
  const paginacion = doc.getElementById("paginacion");
  const paginaEstado = doc.getElementById("pagina-estado");
  const anterior = doc.getElementById("anterior");
  const siguiente = doc.getElementById("siguiente");
  const detalle = doc.getElementById("detalle");
  const nombreDetalle = doc.getElementById("detalle-nombre");
  const gruposDetalle = doc.getElementById("detalle-grupos");
  let categorias = [];
  let seleccionada = "";
  let pagina = 1;
  let controlador = null;
  let secuencia = 0;

  function ocultarDetalle() {
    seleccionada = "";
    detalle.hidden = true;
  }

  function mostrarDetalle(categoria) {
    seleccionada = categoria.referencia;
    nombreDetalle.replaceChildren(textoEn(doc, categoria.etiqueta, idiomaDatos, idiomaUI));
    gruposDetalle.replaceChildren(textoEn(doc,
      categoria.grupos_subgrupos.length
        ? categoria.grupos_subgrupos.map((g) => g.etiqueta).join(", ")
        : t("sinGrupo"),
      categoria.grupos_subgrupos.length ? idiomaDatos : idiomaUI, idiomaUI));
    detalle.hidden = false;
    for (const boton of filas.querySelectorAll("button[data-referencia]")) {
      boton.setAttribute("aria-pressed", String(boton.dataset.referencia === seleccionada));
    }
    detalle.focus();
  }

  function pintar() {
    const visibles = filtrarCategorias(categorias, buscar.value, grupo.value, localizacion);
    const totalPaginas = Math.max(1, Math.ceil(visibles.length / TAMANO_PAGINA));
    pagina = Math.min(pagina, totalPaginas);
    const mostradas = visibles.slice((pagina - 1) * TAMANO_PAGINA, pagina * TAMANO_PAGINA);
    filas.replaceChildren();
    tabla.hidden = visibles.length === 0;
    paginacion.hidden = visibles.length === 0;
    paginaEstado.textContent = t("paginaEstado", {
      pagina: new Intl.NumberFormat(localizacion).format(pagina),
      total: new Intl.NumberFormat(localizacion).format(totalPaginas),
    });
    anterior.disabled = pagina <= 1;
    siguiente.disabled = pagina >= totalPaginas;
    cuenta.hidden = false;
    cuenta.textContent = t("cuenta", { cuenta: new Intl.NumberFormat(localizacion).format(visibles.length) });
    estado.textContent = visibles.length ? "" : t(categorias.length ? "sinCoincidencias" : "sinCategorias");
    if (!mostradas.some((c) => c.referencia === seleccionada)) ocultarDetalle();

    for (const categoria of mostradas) {
      const fila = doc.createElement("tr");
      const celdaNombre = doc.createElement("th");
      celdaNombre.scope = "row";
      celdaNombre.append(textoEn(doc, categoria.etiqueta, idiomaDatos, idiomaUI));
      const celdaGrupo = doc.createElement("td");
      celdaGrupo.append(textoEn(doc,
        categoria.grupos_subgrupos.length
          ? categoria.grupos_subgrupos.map((g) => g.etiqueta).join(", ")
          : t("sinGrupo"),
        categoria.grupos_subgrupos.length ? idiomaDatos : idiomaUI, idiomaUI));
      const celdaAccion = doc.createElement("td");
      const boton = doc.createElement("button");
      boton.type = "button";
      boton.className = "boton-terciario";
      boton.dataset.referencia = categoria.referencia;
      boton.setAttribute("aria-pressed", String(seleccionada === categoria.referencia));
      boton.textContent = t("verDetalle");
      boton.addEventListener("click", () => mostrarDetalle(categoria));
      celdaAccion.append(boton);
      fila.append(celdaNombre, celdaGrupo, celdaAccion);
      filas.append(fila);
    }
  }

  async function cargar() {
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++secuencia;
    categorias = [];
    ocultarDetalle();
    filas.replaceChildren();
    tabla.hidden = true;
    paginacion.hidden = true;
    cuenta.hidden = true;
    buscar.disabled = true;
    grupo.disabled = true;
    reintentar.hidden = true;
    estado.textContent = t("cargando");
    try {
      const opciones = await puerto.listarOpciones({ signal: controlador.signal });
      if (actual !== secuencia || controlador.signal.aborted) return;
      if (!Array.isArray(opciones)) throw new TypeError("respuesta no válida");
      categorias = opciones;
      const grupos = new Map(opciones.flatMap((c) => c.grupos_subgrupos.map((g) => [g.clave, g.etiqueta])));
      grupo.replaceChildren();
      const todos = doc.createElement("option");
      todos.value = "";
      todos.textContent = t("todos");
      grupo.append(todos);
      for (const [clave, etiqueta] of grupos) {
        const opcion = doc.createElement("option");
        opcion.value = clave;
        opcion.textContent = etiqueta;
        grupo.append(opcion);
      }
      buscar.value = "";
      grupo.value = "";
      buscar.disabled = false;
      grupo.disabled = false;
      pintar();
    } catch (error) {
      if (actual !== secuencia || controlador.signal.aborted) return;
      const codigo = error?.codigo;
      const sinIdentidad = error?.estado === 401 || codigo === "autenticacion_requerida";
      const sinPermiso = error?.estado === 403 || codigo === "acceso_denegado";
      estado.textContent = t(sinIdentidad ? "sinIdentidad" : sinPermiso ? "sinPermiso" : "errorConsulta");
      reintentar.hidden = sinIdentidad || sinPermiso;
    }
  }

  doc.getElementById("filtros").addEventListener("submit", (evento) => evento.preventDefault());
  buscar.addEventListener("input", () => { pagina = 1; pintar(); });
  grupo.addEventListener("change", () => { pagina = 1; pintar(); });
  anterior.addEventListener("click", () => { pagina -= 1; pintar(); });
  siguiente.addEventListener("click", () => { pagina += 1; pintar(); });
  reintentar.addEventListener("click", cargar);
  doc.defaultView?.addEventListener("pagehide", () => controlador?.abort());
  doc.defaultView?.addEventListener("pageshow", (evento) => {
    if (evento.persisted) void cargar();
  });
  return Object.freeze({ cargar, cancelar: () => controlador?.abort() });
}
