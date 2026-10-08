export const TAMANO_PAGINA = 20;

const PLANTILLA = `
  <section class="panel" aria-labelledby="rpt-lista-titulo">
    <div class="cabecera-panel">
      <h2 id="rpt-lista-titulo" data-rpt-texto="listaTitulo"></h2>
    </div>
    <div class="cuerpo-panel">
      <form class="barra-filtros-bolsa" role="search" data-rpt="filtros" aria-labelledby="rpt-lista-titulo">
        <div class="campo-filtro">
          <label for="rpt-buscar" data-rpt-texto="buscar"></label>
          <input id="rpt-buscar" data-rpt="buscar" type="search" maxlength="100" autocomplete="off" disabled />
        </div>
        <div class="campo-filtro">
          <label for="rpt-grupo" data-rpt-texto="grupoFiltro"></label>
          <select id="rpt-grupo" data-rpt="grupo" disabled></select>
        </div>
      </form>
      <p data-rpt="estado" role="status" aria-live="polite"></p>
      <button type="button" data-rpt="reintentar" class="boton-secundario" hidden data-rpt-texto="reintentar"></button>
      <p data-rpt="cuenta" aria-live="polite" hidden></p>
    </div>
    <div class="tabla-contenedor" data-rpt="tabla" tabindex="0" hidden>
      <table class="tabla-datos" aria-labelledby="rpt-lista-titulo">
        <thead><tr>
          <th scope="col" data-rpt-texto="colCategoria"></th>
          <th scope="col" data-rpt-texto="colGrupo"></th>
        </tr></thead>
        <tbody data-rpt="filas"></tbody>
      </table>
    </div>
    <nav class="paginacion-marco" data-rpt="paginacion" hidden>
      <span data-rpt="pagina-estado" aria-live="polite"></span>
      <div class="paginacion-marco__paginas">
        <button type="button" data-rpt="anterior" data-rpt-texto="anterior"></button>
        <button type="button" data-rpt="siguiente" data-rpt-texto="siguiente"></button>
      </div>
    </nav>
  </section>`;

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

/** Consulta de categorías montada dentro de la raíz que entrega el portal. */
export function montarVistaCategorias({ raiz, puerto, t, idiomaUI, idiomaDatos, localizacion }) {
  if (!raiz?.ownerDocument || typeof puerto?.listarOpciones !== "function" || typeof t !== "function") {
    throw new TypeError("montaje de categorías no disponible");
  }
  const doc = raiz.ownerDocument;
  raiz.innerHTML = PLANTILLA;
  raiz.querySelectorAll("[data-rpt-texto]").forEach((nodo) => { nodo.textContent = t(nodo.dataset.rptTexto); });
  const parte = (nombre) => raiz.querySelector(`[data-rpt="${nombre}"]`);
  const paginacion = parte("paginacion");
  paginacion.setAttribute("aria-label", t("paginacionLabel"));
  const buscar = parte("buscar");
  const grupo = parte("grupo");
  const estado = parte("estado");
  const reintentar = parte("reintentar");
  const tabla = parte("tabla");
  const filas = parte("filas");
  const cuenta = parte("cuenta");
  const paginaEstado = parte("pagina-estado");
  const anterior = parte("anterior");
  const siguiente = parte("siguiente");
  const numero = new Intl.NumberFormat(localizacion);
  let categorias = [];
  let pagina = 1;
  let controlador = null;
  let secuencia = 0;

  function pintar() {
    const visibles = filtrarCategorias(categorias, buscar.value, grupo.value, localizacion);
    const totalPaginas = Math.max(1, Math.ceil(visibles.length / TAMANO_PAGINA));
    pagina = Math.min(pagina, totalPaginas);
    const mostradas = visibles.slice((pagina - 1) * TAMANO_PAGINA, pagina * TAMANO_PAGINA);
    filas.replaceChildren();
    tabla.hidden = visibles.length === 0;
    paginacion.hidden = totalPaginas <= 1;
    paginaEstado.textContent = t("paginaEstado", { pagina: numero.format(pagina), total: numero.format(totalPaginas) });
    anterior.disabled = pagina <= 1;
    siguiente.disabled = pagina >= totalPaginas;
    cuenta.hidden = false;
    cuenta.textContent = t("cuenta", { cuenta: numero.format(visibles.length) });
    estado.textContent = visibles.length ? "" : t(categorias.length ? "sinCoincidencias" : "sinCategorias");

    for (const categoria of mostradas) {
      const fila = doc.createElement("tr");
      const celdaNombre = doc.createElement("th");
      celdaNombre.scope = "row";
      celdaNombre.append(textoEn(doc, categoria.etiqueta, idiomaDatos, idiomaUI));
      const celdaGrupo = doc.createElement("td");
      if (!categoria.grupos_subgrupos.length) celdaGrupo.textContent = t("sinGrupo");
      categoria.grupos_subgrupos.forEach((g, i) => {
        if (i > 0) celdaGrupo.append(", ");
        // Cada grupo lleva a la lista filtrada por él.
        const boton = doc.createElement("button");
        boton.type = "button";
        boton.className = "boton-terciario";
        boton.dataset.grupo = g.clave;
        boton.setAttribute("aria-label", t("filtrarGrupo", { grupo: g.etiqueta }));
        boton.append(textoEn(doc, g.etiqueta, idiomaDatos, idiomaUI));
        celdaGrupo.append(boton);
      });
      fila.append(celdaNombre, celdaGrupo);
      filas.append(fila);
    }
  }

  async function cargar() {
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++secuencia;
    categorias = [];
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
      const todos = doc.createElement("option");
      todos.value = "";
      todos.textContent = t("todos");
      grupo.replaceChildren(todos);
      for (const [clave, etiqueta] of [...grupos].sort((a, b) => a[1].localeCompare(b[1], localizacion))) {
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

  parte("filtros").addEventListener("submit", (evento) => evento.preventDefault());
  buscar.addEventListener("input", () => { pagina = 1; pintar(); });
  grupo.addEventListener("change", () => { pagina = 1; pintar(); });
  filas.addEventListener("click", (evento) => {
    const boton = evento.target.closest?.("button[data-grupo]");
    if (!boton) return;
    grupo.value = boton.dataset.grupo;
    pagina = 1;
    pintar();
    grupo.focus();
  });
  anterior.addEventListener("click", () => { pagina -= 1; pintar(); tabla.focus(); });
  siguiente.addEventListener("click", () => { pagina += 1; pintar(); tabla.focus(); });
  reintentar.addEventListener("click", cargar);
  return Object.freeze({ cargar, desmontar: () => { secuencia += 1; controlador?.abort(); } });
}
