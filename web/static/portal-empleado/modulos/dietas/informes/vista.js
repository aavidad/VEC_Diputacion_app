const CONCEPTOS = Object.freeze(["manutencion", "kilometraje", "otros_gastos"]);
const TAMANO_PAGINA = 6;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;

function fechaValida(valor) {
  if (!FECHA.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return !Number.isNaN(fecha.valueOf()) && fecha.toISOString().slice(0, 10) === valor;
}

function congelarDatos(datos, localizacion) {
  if (datos?.naturaleza !== "sintetica" || datos.schema !== "dietas-informes-demo"
    || datos.version !== 1 || datos.criterio_periodo !== "fecha_inicio"
    || typeof datos.moneda !== "string" || !/^[A-Z]{3}$/u.test(datos.moneda)
    || !fechaValida(datos.fecha_corte) || !Array.isArray(datos.registros)) throw new TypeError("datos");
  let moneda;
  try { moneda = new Intl.NumberFormat(localizacion, { style: "currency", currency: datos.moneda }); }
  catch { throw new TypeError("datos"); }
  if (moneda.resolvedOptions().minimumFractionDigits !== 2
    || moneda.resolvedOptions().maximumFractionDigits !== 2) throw new TypeError("datos");
  const referencias = new Set();
  const registros = datos.registros.map((entrada) => {
    if (!entrada || !["referencia", "persona_ref", "persona", "unidad_ref", "unidad"].every((campo) =>
      typeof entrada[campo] === "string" && entrada[campo].trim()) || !fechaValida(entrada.fecha_inicio)
      || !Number.isSafeInteger(entrada.version_comision) || entrada.version_comision < 1
      || !["orientativo", "liquidado", "fiscalizado"].includes(entrada.situacion)
      || (entrada.moneda !== undefined && entrada.moneda !== datos.moneda)
      || referencias.has(entrada.referencia)) throw new TypeError("datos");
    referencias.add(entrada.referencia);
    const conceptos = {};
    for (const clave of CONCEPTOS) {
      const valor = entrada.conceptos_centimos?.[clave];
      if (!Number.isSafeInteger(valor) || valor < 0) throw new TypeError("datos");
      conceptos[clave] = valor;
    }
    const total = CONCEPTOS.reduce((suma, clave) => suma + conceptos[clave], 0);
    if (!Number.isSafeInteger(total) || total !== entrada.total_centimos) throw new TypeError("datos");
    return Object.freeze({ referencia: entrada.referencia, version_comision: entrada.version_comision,
      situacion: entrada.situacion, persona_ref: entrada.persona_ref,
      persona: entrada.persona, unidad_ref: entrada.unidad_ref, unidad: entrada.unidad,
      fecha_inicio: entrada.fecha_inicio, conceptos_centimos: Object.freeze(conceptos), total_centimos: entrada.total_centimos });
  });
  return Object.freeze({ registros: Object.freeze(registros), moneda });
}

export function resumirInformesDietas(registros, filtros = {}) {
  const seleccion = registros.filter((fila) => (!filtros.persona || fila.persona_ref === filtros.persona)
    && (!filtros.unidad || fila.unidad_ref === filtros.unidad)
    && (!filtros.desde || fila.fecha_inicio >= filtros.desde)
    && (!filtros.hasta || fila.fecha_inicio <= filtros.hasta));
  const conceptos = Object.fromEntries(CONCEPTOS.map((clave) => [clave, 0]));
  for (const fila of seleccion) for (const clave of CONCEPTOS) {
    conceptos[clave] += fila.conceptos_centimos[clave];
    if (!Number.isSafeInteger(conceptos[clave])) throw new TypeError("datos");
  }
  const total = seleccion.reduce((suma, fila) => suma + fila.total_centimos, 0);
  if (!Number.isSafeInteger(total) || total !== CONCEPTOS.reduce((suma, clave) => suma + conceptos[clave], 0)) throw new TypeError("datos");
  return Object.freeze({ registros: Object.freeze(seleccion), conceptos_centimos: Object.freeze(conceptos),
    total_centimos: total });
}

const nodo = (documento, tipo, texto) => {
  const creado = documento.createElement(tipo);
  if (texto !== undefined) creado.textContent = texto;
  return creado;
};

/** Vista local de datos sintéticos. cargarDatos es la única fuente y se inyecta al montar. */
export function montarInformesDietas(contenedor, { cargarDatos, traducir, documento = contenedor?.ownerDocument,
  localizacion, registrarDesmontar } = {}) {
  if (!contenedor?.append || !documento?.createElement || typeof cargarDatos !== "function"
    || typeof traducir !== "function" || typeof localizacion !== "string" || !localizacion)
    throw new TypeError("informes de Dietas no disponibles");

  const t = (clave, variables) => traducir(clave, variables);
  let moneda;
  const numero = new Intl.NumberFormat(localizacion);
  const fecha = new Intl.DateTimeFormat(localizacion, { dateStyle: "medium", timeZone: "UTC" });
  const importeMonetario = (centimos) => {
    // Intl recibe el entero como BigInt: convertir los céntimos a Number perdería precisión.
    const importe = BigInt(centimos);
    const fraccion = (importe % 100n).toString().padStart(2, "0");
    return moneda.formatToParts(importe / 100n)
      .map((parte) => parte.type === "fraction" ? fraccion : parte.value).join("");
  };

  const raiz = nodo(documento, "section"); raiz.className = "panel"; raiz.dataset.dietasInformes = "";
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-panel";
  const titulo = nodo(documento, "h2", t("titulo"));
  const acciones = nodo(documento, "div");
  const ayuda = nodo(documento, "details");
  const abrirAyuda = nodo(documento, "summary", "?"); abrirAyuda.className = "boton-secundario";
  abrirAyuda.setAttribute("aria-label", t("ayuda"));
  ayuda.append(abrirAyuda, nodo(documento, "p", t("criterio_periodo")));
  acciones.append(ayuda);
  for (const clave of ["exportar", "imprimir"]) {
    const boton = nodo(documento, "button", t(clave)); boton.type = "button";
    boton.className = "boton-secundario"; boton.disabled = true;
    boton.title = t("acciones_pendientes"); acciones.append(boton);
  }
  cabecera.append(titulo, acciones);
  const cuerpo = nodo(documento, "div"); cuerpo.className = "cuerpo-panel";
  const origen = nodo(documento, "p", t("origen_sintetico")); origen.className = "nota-integracion";
  origen.setAttribute("role", "note");
  const limite = nodo(documento, "p", t("acciones_pendientes"));
  const estado = nodo(documento, "p"); estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite"); estado.setAttribute("tabindex", "-1");
  estado.dataset.dietasInformesEstado = "";
  const reintentar = nodo(documento, "button", t("reintentar")); reintentar.type = "button";
  reintentar.className = "boton-secundario"; reintentar.hidden = true;

  const form = nodo(documento, "form"); form.dataset.dietasInformesFiltros = "";
  form.className = "barra-filtros-bolsa";
  const subtitulo = nodo(documento, "h3", t("filtros"));
  const crearSelect = (clave, opcionInicial) => {
    const select = nodo(documento, "select"); select.name = clave;
    const etiqueta = nodo(documento, "label", t(clave)); etiqueta.className = "campo-filtro"; etiqueta.append(select);
    const opcion = nodo(documento, "option", t(opcionInicial)); opcion.value = ""; select.append(opcion);
    form.append(etiqueta); return select;
  };
  const persona = crearSelect("persona", "todas_personas");
  const unidad = crearSelect("unidad", "todas_unidades");
  const crearFecha = (clave) => {
    const campo = nodo(documento, "input"); campo.type = "date"; campo.name = clave;
    const etiqueta = nodo(documento, "label", t(clave)); etiqueta.className = "campo-filtro";
    etiqueta.append(campo); form.append(etiqueta);
    return campo;
  };
  const desde = crearFecha("desde"); const hasta = crearFecha("hasta");
  const aplicar = nodo(documento, "button", t("aplicar")); aplicar.type = "submit"; aplicar.className = "boton-primario";
  const limpiar = nodo(documento, "button", t("limpiar")); limpiar.type = "button"; limpiar.className = "boton-secundario";
  const accionesFiltro = nodo(documento, "div"); accionesFiltro.className = "acciones-filtro";
  accionesFiltro.append(aplicar, limpiar); form.append(accionesFiltro);
  const resumen = nodo(documento, "section"); resumen.dataset.dietasInformesResumen = "";
  resumen.setAttribute("tabindex", "-1");
  const listado = nodo(documento, "section"); listado.dataset.dietasInformesListado = "";
  form.hidden = true; resumen.hidden = true; listado.hidden = true;
  cuerpo.append(origen, estado, reintentar, subtitulo, form, resumen, listado, limite);
  raiz.append(cabecera, cuerpo); contenedor.append(raiz);

  let activa = true; let disponible = false; let controlador; let generacion = 0; let registros = Object.freeze([]);
  let filtros = Object.freeze({}); let pagina = 0;
  const mostrarEstado = (clave, variables) => {
    estado.textContent = clave ? t(clave, variables) : "";
    estado.className = clave === "resultado_filtro" ? "solo-lectura" : "";
  };
  const anunciarResultado = () => mostrarEstado("resultado_filtro", {
    cuenta: numero.format(resumirInformesDietas(registros, filtros).registros.length),
  });
  const limpiarErrorFecha = () => {
    desde.removeAttribute("aria-invalid"); hasta.removeAttribute("aria-invalid");
  };
  const elegir = (select, filas, referencia, nombre) => {
    const anterior = select.value;
    select.replaceChildren();
    const inicial = nodo(documento, "option", t(referencia === "persona_ref" ? "todas_personas" : "todas_unidades"));
    inicial.value = ""; select.append(inicial);
    const opciones = new Map(filas.map((fila) => [fila[referencia], fila[nombre]]));
    for (const [valor, etiqueta] of [...opciones].sort((a, b) => a[1].localeCompare(b[1], localizacion))) {
      const opcion = nodo(documento, "option", etiqueta); opcion.value = valor; select.append(opcion);
    }
    select.value = opciones.has(anterior) ? anterior : "";
  };

  function pintar() {
    if (!activa) return;
    const informe = resumirInformesDietas(registros, filtros);
    const filas = informe.registros; const paginas = Math.max(1, Math.ceil(filas.length / TAMANO_PAGINA));
    pagina = Math.min(pagina, paginas - 1);
    resumen.replaceChildren(); listado.replaceChildren();
    resumen.append(nodo(documento, "h3", t("resumen")));
    const tarjetas = nodo(documento, "div"); tarjetas.className = "rejilla-kpi cuatro";
    for (const [etiqueta, valor] of [["registros", numero.format(filas.length)], ["total", importeMonetario(informe.total_centimos)]]) {
      const tarjeta = nodo(documento, "article"); tarjeta.className = "tarjeta-kpi";
      const icono = nodo(documento, "span", etiqueta === "total"
        ? moneda.formatToParts(0).find((parte) => parte.type === "currency").value : "≡");
      icono.className = "icono-kpi"; icono.setAttribute("aria-hidden", "true");
      const textoTarjeta = nodo(documento, "div");
      const cantidad = nodo(documento, "strong", valor); cantidad.className = "valor-kpi";
      const rotulo = nodo(documento, "span", t(etiqueta)); rotulo.className = "etiqueta-kpi";
      textoTarjeta.append(cantidad, rotulo); tarjeta.append(icono, textoTarjeta); tarjetas.append(tarjeta);
    }
    resumen.append(tarjetas, nodo(documento, "h4", t("desglose")));
    const desglose = nodo(documento, "dl"); desglose.className = "datos-clave";
    for (const clave of CONCEPTOS) {
      const fila = nodo(documento, "div");
      fila.append(nodo(documento, "dt", t(clave)), nodo(documento, "dd", importeMonetario(informe.conceptos_centimos[clave])));
      desglose.append(fila);
    }
    resumen.append(desglose);
    listado.append(nodo(documento, "h3", t("listado")));
    if (!filas.length) { listado.append(nodo(documento, "p", t("vacio"))); return; }
    const marco = nodo(documento, "div"); marco.className = "marco-tabla-paginado";
    const envoltura = nodo(documento, "div"); envoltura.className = "tabla-contenedor";
    envoltura.setAttribute("tabindex", "0");
    const tabla = nodo(documento, "table"); tabla.className = "tabla-datos";
    const caption = nodo(documento, "caption", t("listado")); tabla.append(caption);
    const thead = nodo(documento, "thead"); const cabeceras = nodo(documento, "tr");
    for (const clave of ["referencia", "version_comision", "situacion", "fecha", "persona", "unidad", ...CONCEPTOS, "total"]) {
      const th = nodo(documento, "th", t(clave)); th.scope = "col"; cabeceras.append(th);
    }
    thead.append(cabeceras); tabla.append(thead);
    const tbody = nodo(documento, "tbody");
    for (const fila of filas.slice(pagina * TAMANO_PAGINA, (pagina + 1) * TAMANO_PAGINA)) {
      const tr = nodo(documento, "tr");
      const valores = [fila.referencia, numero.format(fila.version_comision), t(`situacion_${fila.situacion}`),
        fecha.format(new Date(`${fila.fecha_inicio}T00:00:00Z`)), fila.persona,
        fila.unidad, ...CONCEPTOS.map((clave) => importeMonetario(fila.conceptos_centimos[clave])),
        importeMonetario(fila.total_centimos)];
      valores.forEach((valor, indice) => { const td = nodo(documento, "td", valor);
        if (indice === 1 || indice >= 6) td.dataset.tipo = "numero"; tr.append(td); });
      tbody.append(tr);
    }
    tabla.append(tbody); envoltura.append(tabla); marco.append(envoltura);
    const pie = nodo(documento, "div"); pie.className = "paginacion-marco";
    const inicio = pagina * TAMANO_PAGINA + 1; const fin = Math.min(inicio + TAMANO_PAGINA - 1, filas.length);
    const cuenta = nodo(documento, "p", t("mostrando", { inicio: numero.format(inicio), fin: numero.format(fin), total: numero.format(filas.length) }));
    cuenta.dataset.dietasInformesCuenta = ""; cuenta.setAttribute("tabindex", "-1");
    cuenta.setAttribute("aria-live", "polite");
    const nav = nodo(documento, "nav"); nav.className = "paginacion-marco__paginas";
    nav.setAttribute("aria-label", t("paginacion"));
    const botonPagina = (clave, destino, deshabilitado) => { const boton = nodo(documento, "button", t(clave));
      boton.type = "button"; boton.dataset.dietasInformesPagina = String(destino);
      boton.disabled = deshabilitado; nav.append(boton); };
    botonPagina("anterior", pagina - 1, pagina === 0);
    botonPagina("siguiente", pagina + 1, pagina === paginas - 1);
    pie.append(cuenta, nav); marco.append(pie); listado.append(marco);
  }

  async function cargar() {
    if (!activa) return;
    const focoReintento = documento.activeElement === reintentar;
    controlador?.abort(); const propia = ++generacion;
    controlador = new AbortController(); disponible = false; mostrarEstado("cargando"); reintentar.hidden = true;
    if (focoReintento) estado.focus();
    form.hidden = true; resumen.hidden = true; listado.hidden = true;
    try {
      const datos = await cargarDatos({ signal: controlador.signal });
      if (!activa || propia !== generacion) return;
      const fuente = congelarDatos(datos, localizacion); registros = fuente.registros; moneda = fuente.moneda;
      elegir(persona, registros, "persona_ref", "persona");
      elegir(unidad, registros, "unidad_ref", "unidad");
      filtros = Object.freeze({ persona: persona.value, unidad: unidad.value, desde: desde.value, hasta: hasta.value });
      disponible = true; form.hidden = false; resumen.hidden = false; listado.hidden = false;
      pagina = 0; pintar(); anunciarResultado();
      if (focoReintento && documento.activeElement === estado) resumen.focus();
    } catch (error) {
      if (!activa || propia !== generacion || controlador.signal.aborted) return;
      registros = Object.freeze([]); resumen.replaceChildren(); listado.replaceChildren();
      mostrarEstado(error instanceof TypeError && error.message === "datos" ? "datos_error" : "carga_error");
      reintentar.hidden = false;
      if (focoReintento && documento.activeElement === estado) reintentar.focus();
    }
  }

  function enviar(evento) {
    evento.preventDefault(); limpiarErrorFecha();
    if (!disponible) return;
    if ((desde.value && !fechaValida(desde.value)) || (hasta.value && !fechaValida(hasta.value))) {
      desde.setAttribute("aria-invalid", "true"); hasta.setAttribute("aria-invalid", "true");
      mostrarEstado("fecha_error"); estado.focus(); return;
    }
    if (desde.value && hasta.value && desde.value > hasta.value) {
      desde.setAttribute("aria-invalid", "true"); hasta.setAttribute("aria-invalid", "true");
      mostrarEstado("periodo_error"); estado.focus(); return;
    }
    filtros = Object.freeze({ persona: persona.value, unidad: unidad.value, desde: desde.value, hasta: hasta.value });
    pagina = 0; pintar(); anunciarResultado();
  }
  function borrar() {
    if (!disponible) return;
    persona.value = ""; unidad.value = ""; desde.value = ""; hasta.value = "";
    limpiarErrorFecha(); filtros = Object.freeze({}); pagina = 0; pintar(); anunciarResultado();
  }
  function paginar(evento) {
    const destino = evento.target?.dataset?.dietasInformesPagina;
    if (destino === undefined || evento.target.disabled) return;
    const nueva = Number(destino); const paginas = Math.ceil(resumirInformesDietas(registros, filtros).registros.length / TAMANO_PAGINA);
    if (!Number.isInteger(nueva) || nueva < 0 || nueva >= paginas) return;
    pagina = nueva; pintar(); listado.querySelector?.("[data-dietas-informes-cuenta]")?.focus();
  }
  form.addEventListener("submit", enviar); limpiar.addEventListener("click", borrar);
  reintentar.addEventListener("click", cargar); listado.addEventListener("click", paginar);
  cargar();
  function desmontar() {
    if (!activa) return; activa = false; generacion += 1; controlador?.abort();
    form.removeEventListener("submit", enviar); limpiar.removeEventListener("click", borrar);
    reintentar.removeEventListener("click", cargar); listado.removeEventListener("click", paginar); raiz.remove();
  }
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
