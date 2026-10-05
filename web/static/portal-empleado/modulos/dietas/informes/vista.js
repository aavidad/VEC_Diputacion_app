// Campos del DTO de comisión que puede usar el informe de ejemplo, no una regla de período.
const CAMPOS_FECHA = Object.freeze(["fecha_inicio", "fecha_liquidacion", "fecha_fiscalizacion"]);
const TAMANO_PAGINA = 6;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const INSTANTE_UTC = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/u;
const CLAVE = /^[a-z][a-z0-9_]*$/u;

function fechaValida(valor) {
  if (!FECHA.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return !Number.isNaN(fecha.valueOf()) && fecha.toISOString().slice(0, 10) === valor;
}

function instanteValido(valor) {
  if (typeof valor !== "string" || !INSTANTE_UTC.test(valor)) return false;
  const instante = new Date(valor);
  return Number.isFinite(instante.valueOf()) && instante.toISOString().replace(".000Z", "Z") === valor;
}

function congelarConfiguracion(configuracion, datos) {
  const criterio = configuracion?.criterio;
  const lista = (valores) => Array.isArray(valores) && valores.length > 0 && valores.length <= 20
    && valores.every((valor) => typeof valor === "string" && CLAVE.test(valor))
    && new Set(valores).size === valores.length;
  if (configuracion?.schema !== "dietas-informes-config-demo" || configuracion.naturaleza !== "sintetica"
    || typeof configuracion.referencia !== "string" || !configuracion.referencia
    || !Number.isSafeInteger(configuracion.version) || configuracion.version < 1
    || configuracion.referencia !== datos?.configuracion_ref
    || configuracion.version !== datos.configuracion_version
    || !criterio || !CAMPOS_FECHA.includes(criterio.campo_fecha)
    || !lista(criterio.estados_incluidos) || !lista(criterio.conceptos_incluidos)
    || !Array.isArray(configuracion.historia) || configuracion.historia.length !== configuracion.version)
    throw new TypeError("datos");
  const historia = configuracion.historia.map((entrada, indice) => {
    if (entrada?.version !== indice + 1 || typeof entrada.actor_ref !== "string"
      || !entrada.actor_ref.startsWith("actor:ejemplo:") || !instanteValido(entrada.fecha)
      || typeof entrada.actor_nombre !== "string" || !entrada.actor_nombre.trim()
      || typeof entrada.idioma_motivo !== "string"
      || !/^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/u.test(entrada.idioma_motivo)
      || typeof entrada.motivo !== "string"
      || !entrada.motivo.trim()) throw new TypeError("datos");
    try {
      if (Intl.getCanonicalLocales(entrada.idioma_motivo).length !== 1) throw new TypeError("datos");
    } catch { throw new TypeError("datos"); }
    return Object.freeze({ version: entrada.version, actor_ref: entrada.actor_ref,
      actor_nombre: entrada.actor_nombre, fecha: entrada.fecha,
      idioma_motivo: entrada.idioma_motivo, motivo: entrada.motivo });
  });
  return Object.freeze({ referencia: configuracion.referencia, version: configuracion.version,
    campo_fecha: criterio.campo_fecha,
    estados_incluidos: Object.freeze([...criterio.estados_incluidos]),
    conceptos_incluidos: Object.freeze([...criterio.conceptos_incluidos]),
    historia: Object.freeze(historia) });
}

function congelarDatos(datos, configuracion, localizacion) {
  if (datos?.naturaleza !== "sintetica" || datos.schema !== "dietas-informes-demo"
    || datos.version !== 1
    || typeof datos.moneda !== "string" || !/^[A-Z]{3}$/u.test(datos.moneda)
    || typeof Intl.supportedValuesOf !== "function" || !Intl.supportedValuesOf("currency").includes(datos.moneda)
    || !fechaValida(datos.fecha_corte) || !Array.isArray(datos.registros)) throw new TypeError("datos");
  let moneda;
  try { moneda = new Intl.NumberFormat(localizacion, { style: "currency", currency: datos.moneda }); }
  catch { throw new TypeError("datos"); }
  if (moneda.resolvedOptions().minimumFractionDigits !== 2
    || moneda.resolvedOptions().maximumFractionDigits !== 2) throw new TypeError("datos");
  const referencias = new Set();
  const registros = datos.registros.map((entrada) => {
    if (!entrada || !["referencia", "persona_ref", "persona", "unidad_ref", "unidad"].every((campo) =>
      typeof entrada[campo] === "string" && entrada[campo].trim())
      || !fechaValida(entrada.fecha_inicio)
      || CAMPOS_FECHA.some((campo) => !Object.hasOwn(entrada, campo)
        || (entrada[campo] !== null && !fechaValida(entrada[campo])))
      || !Number.isSafeInteger(entrada.version_comision) || entrada.version_comision < 1
      || typeof entrada.situacion !== "string" || !CLAVE.test(entrada.situacion)
      || (configuracion.estados_incluidos.includes(entrada.situacion)
        && !fechaValida(entrada[configuracion.campo_fecha]))
      || (entrada.moneda !== undefined && entrada.moneda !== datos.moneda)
      || referencias.has(entrada.referencia)) throw new TypeError("datos");
    referencias.add(entrada.referencia);
    if (!entrada.conceptos_centimos || typeof entrada.conceptos_centimos !== "object"
      || Array.isArray(entrada.conceptos_centimos)) throw new TypeError("datos");
    const pares = Object.entries(entrada.conceptos_centimos);
    if (!pares.length || pares.length > 20 || pares.some(([clave, valor]) => !CLAVE.test(clave)
      || !Number.isSafeInteger(valor) || valor < 0)
      || configuracion.conceptos_incluidos.some((clave) => !Object.hasOwn(entrada.conceptos_centimos, clave)))
      throw new TypeError("datos");
    const conceptos = Object.fromEntries(pares);
    const total = pares.reduce((suma, [, valor]) => suma + valor, 0);
    if (!Number.isSafeInteger(total) || total !== entrada.total_centimos) throw new TypeError("datos");
    return Object.freeze({ referencia: entrada.referencia, version_comision: entrada.version_comision,
      situacion: entrada.situacion, persona_ref: entrada.persona_ref,
      persona: entrada.persona, unidad_ref: entrada.unidad_ref, unidad: entrada.unidad,
      fecha_inicio: entrada.fecha_inicio, fecha_liquidacion: entrada.fecha_liquidacion,
      fecha_fiscalizacion: entrada.fecha_fiscalizacion,
      conceptos_centimos: Object.freeze(conceptos), total_centimos: entrada.total_centimos });
  });
  return Object.freeze({ registros: Object.freeze(registros), moneda });
}

export function resumirInformesDietas(registros, configuracion, filtros = {}) {
  const estados = new Set(configuracion.estados_incluidos);
  const campo = configuracion.campo_fecha;
  if (registros.some((fila) => estados.has(fila.situacion) && !fechaValida(fila[campo]))) throw new TypeError("datos");
  const seleccion = registros.filter((fila) => estados.has(fila.situacion)
    && (!filtros.situacion || fila.situacion === filtros.situacion)
    && (!filtros.persona || fila.persona_ref === filtros.persona)
    && (!filtros.unidad || fila.unidad_ref === filtros.unidad)
    && (!filtros.desde || fila[campo] >= filtros.desde)
    && (!filtros.hasta || fila[campo] <= filtros.hasta));
  const conceptos = Object.fromEntries(configuracion.conceptos_incluidos.map((clave) => [clave, 0]));
  for (const fila of seleccion) for (const clave of configuracion.conceptos_incluidos) {
    conceptos[clave] += fila.conceptos_centimos[clave];
    if (!Number.isSafeInteger(conceptos[clave])) throw new TypeError("datos");
  }
  const total = configuracion.conceptos_incluidos.reduce((suma, clave) => suma + conceptos[clave], 0);
  if (!Number.isSafeInteger(total)) throw new TypeError("datos");
  const filas = seleccion.map((fila) => {
    const incluido = configuracion.conceptos_incluidos.reduce((suma, clave) => suma + fila.conceptos_centimos[clave], 0);
    if (!Number.isSafeInteger(incluido)) throw new TypeError("datos");
    return Object.freeze({ ...fila, importe_incluido_centimos: incluido });
  });
  return Object.freeze({ registros: Object.freeze(filas), conceptos_centimos: Object.freeze(conceptos),
    total_centimos: total });
}

const nodo = (documento, tipo, texto) => {
  const creado = documento.createElement(tipo);
  if (texto !== undefined) creado.textContent = texto;
  return creado;
};

/** Vista local de datos sintéticos. Fuente y configuración se inyectan al montar. */
export function montarInformesDietas(contenedor, { cargarDatos, cargarConfiguracion, traducir, documento = contenedor?.ownerDocument,
  localizacion, registrarDesmontar } = {}) {
  if (!contenedor?.append || !documento?.createElement || typeof cargarDatos !== "function"
    || typeof cargarConfiguracion !== "function"
    || typeof traducir !== "function" || typeof localizacion !== "string" || !localizacion)
    throw new TypeError("informes de Dietas no disponibles");

  const t = (clave, variables) => traducir(clave, variables);
  let moneda;
  const numero = new Intl.NumberFormat(localizacion);
  const listaTextos = new Intl.ListFormat(localizacion, { style: "long", type: "conjunction" });
  const fecha = new Intl.DateTimeFormat(localizacion, { dateStyle: "medium", timeZone: "UTC" });
  const fechaHistoria = new Intl.DateTimeFormat(localizacion,
    { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" });
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
  const ayuda = nodo(documento, "section"); ayuda.hidden = true; ayuda.className = "cuerpo-panel";
  ayuda.dataset.dietasInformesAyuda = "";
  const abrirAyuda = nodo(documento, "button", "?"); abrirAyuda.className = "boton-secundario";
  abrirAyuda.type = "button"; abrirAyuda.hidden = true;
  abrirAyuda.setAttribute("aria-label", t("ayuda"));
  abrirAyuda.setAttribute("aria-expanded", "false"); acciones.append(abrirAyuda);
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
  const situacion = crearSelect("situacion", "todas_situaciones");
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
  const filtrosVisibles = nodo(documento, "section"); filtrosVisibles.className = "filtros-activos";
  filtrosVisibles.dataset.dietasInformesAplicados = "";
  filtrosVisibles.setAttribute("aria-label", t("filtros_aplicados"));
  const avisoEdicion = nodo(documento, "span", t("cambios_sin_aplicar"));
  avisoEdicion.className = "estado-chip info"; avisoEdicion.hidden = true;
  avisoEdicion.setAttribute("role", "status");
  const resumen = nodo(documento, "section"); resumen.dataset.dietasInformesResumen = "";
  resumen.setAttribute("tabindex", "-1");
  const listado = nodo(documento, "section"); listado.dataset.dietasInformesListado = "";
  form.hidden = true; filtrosVisibles.hidden = true; resumen.hidden = true; listado.hidden = true;
  cuerpo.append(origen, estado, reintentar, subtitulo, form, filtrosVisibles, resumen, listado, limite);
  raiz.append(cabecera, ayuda, cuerpo); contenedor.append(raiz);

  let activa = true; let disponible = false; let controlador; let generacion = 0; let registros = Object.freeze([]);
  let configuracion;
  let filtros = Object.freeze({}); let pagina = 0;
  const mostrarEstado = (clave, variables) => {
    estado.textContent = clave ? t(clave, variables) : "";
    estado.className = clave === "resultado_filtro" ? "solo-lectura" : "";
  };
  const anunciarResultado = () => mostrarEstado("resultado_filtro", {
    cuenta: numero.format(resumirInformesDietas(registros, configuracion, filtros).registros.length),
  });
  const limpiarErrorFecha = () => {
    desde.removeAttribute("aria-invalid"); hasta.removeAttribute("aria-invalid");
  };
  const elegir = (select, filas, referencia, nombre, aplicado) => {
    const opcionAnterior = [...select.children].find((opcion) => opcion.value === aplicado);
    select.replaceChildren();
    const inicial = nodo(documento, "option", t(referencia === "persona_ref" ? "todas_personas"
      : referencia === "situacion" ? "todas_situaciones" : "todas_unidades"));
    inicial.value = ""; select.append(inicial);
    const opciones = new Map(filas.map((fila) => [fila[referencia], fila[nombre]]));
    for (const [valor, etiqueta] of [...opciones].sort((a, b) => a[1].localeCompare(b[1], localizacion))) {
      const opcion = nodo(documento, "option", etiqueta); opcion.value = valor; select.append(opcion);
    }
    // Una selección aplicada que desaparece sigue filtrando: quitarla mostraría otros informes.
    if (aplicado && !opciones.has(aplicado) && opcionAnterior) select.append(opcionAnterior);
    select.value = aplicado;
  };

  const nombreSeleccionado = (select, valor) => [...select.children]
    .find((opcion) => opcion.value === valor)?.textContent;
  function actualizarAvisoEdicion() {
    if (!disponible) return;
    avisoEdicion.hidden = persona.value === (filtros.persona || "")
      && unidad.value === (filtros.unidad || "")
      && situacion.value === (filtros.situacion || "")
      && desde.value === (filtros.desde || "")
      && hasta.value === (filtros.hasta || "");
  }
  function pintarFiltrosAplicados() {
    const valores = [
      ["persona", filtros.persona && nombreSeleccionado(persona, filtros.persona)],
      ["unidad", filtros.unidad && nombreSeleccionado(unidad, filtros.unidad)],
      ["situacion", filtros.situacion && nombreSeleccionado(situacion, filtros.situacion)],
      ["desde", filtros.desde && fecha.format(new Date(`${filtros.desde}T00:00:00Z`))],
      ["hasta", filtros.hasta && fecha.format(new Date(`${filtros.hasta}T00:00:00Z`))],
    ].filter(([, valor]) => valor);
    filtrosVisibles.replaceChildren(nodo(documento, "strong", t("filtros_aplicados")));
    if (!valores.length) filtrosVisibles.append(nodo(documento, "span", t("sin_filtros")));
    for (const [campo, valor] of valores) {
      const etiqueta = nodo(documento, "span", t("filtro_valor", { campo: t(campo), valor }));
      etiqueta.className = "estado-chip neutro"; filtrosVisibles.append(etiqueta);
    }
    filtrosVisibles.append(avisoEdicion);
    actualizarAvisoEdicion();
  }

  function pintarAyuda(criterio) {
    const dato = (lista, clave, valor, idioma) => {
      const fila = nodo(documento, "div");
      const valorNodo = nodo(documento, "dd", valor);
      if (idioma) valorNodo.setAttribute("lang", idioma);
      fila.append(nodo(documento, "dt", t(clave)), valorNodo); lista.append(fila);
    };
    const etiquetaFecha = t(`fecha_${criterio.campo_fecha}`);
    const resumenCriterio = nodo(documento, "dl"); resumenCriterio.className = "datos-clave";
    dato(resumenCriterio, "version_configuracion", numero.format(criterio.version));
    dato(resumenCriterio, "fecha_configuracion", etiquetaFecha);
    dato(resumenCriterio, "estados_incluidos", listaTextos.format(
      criterio.estados_incluidos.map((clave) => t(`situacion_${clave}`))));
    dato(resumenCriterio, "conceptos_incluidos", listaTextos.format(
      criterio.conceptos_incluidos.map((clave) => t(clave))));
    const historia = nodo(documento, "ol");
    for (const entrada of criterio.historia) {
      const linea = nodo(documento, "li");
      const datosHistoria = nodo(documento, "dl"); datosHistoria.className = "datos-clave";
      dato(datosHistoria, "version_comision", numero.format(entrada.version));
      dato(datosHistoria, "preparado_por", entrada.actor_nombre);
      dato(datosHistoria, "fecha_historia", fechaHistoria.format(new Date(entrada.fecha)));
      dato(datosHistoria, "motivo_historia", entrada.motivo, entrada.idioma_motivo);
      const referencias = nodo(documento, "details");
      referencias.append(nodo(documento, "summary", t("detalle_referencias")));
      const datosReferencia = nodo(documento, "dl"); datosReferencia.className = "datos-clave";
      dato(datosReferencia, "referencia_configuracion", criterio.referencia);
      dato(datosReferencia, "referencia_preparador", entrada.actor_ref);
      referencias.append(datosReferencia); linea.append(datosHistoria, referencias); historia.append(linea);
    }
    ayuda.replaceChildren(nodo(documento, "h3", t("configuracion")),
      nodo(documento, "p", t("criterio_periodo", { fecha: etiquetaFecha })), resumenCriterio,
      nodo(documento, "h4", t("historia_configuracion")), historia,
      nodo(documento, "p", t("limite_configuracion")));
  }

  function cambiarAyuda() {
    if (abrirAyuda.hidden) return;
    ayuda.hidden = !ayuda.hidden;
    abrirAyuda.setAttribute("aria-expanded", String(!ayuda.hidden));
  }

  function pintar() {
    if (!activa) return;
    const informe = resumirInformesDietas(registros, configuracion, filtros);
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
    for (const clave of configuracion.conceptos_incluidos) {
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
    for (const clave of ["referencia", "version_comision", "situacion", "fecha_configuracion", "persona", "unidad",
      ...configuracion.conceptos_incluidos, "total"]) {
      const etiqueta = clave === "fecha_configuracion" ? t(`fecha_${configuracion.campo_fecha}`) : t(clave);
      const th = nodo(documento, "th", clave === "fecha_configuracion"
        ? etiqueta.charAt(0).toLocaleUpperCase(localizacion) + etiqueta.slice(1) : etiqueta);
      th.scope = "col"; cabeceras.append(th);
    }
    thead.append(cabeceras); tabla.append(thead);
    const tbody = nodo(documento, "tbody");
    for (const fila of filas.slice(pagina * TAMANO_PAGINA, (pagina + 1) * TAMANO_PAGINA)) {
      const tr = nodo(documento, "tr");
      const valores = [fila.referencia, numero.format(fila.version_comision), t(`situacion_${fila.situacion}`),
        fecha.format(new Date(`${fila[configuracion.campo_fecha]}T00:00:00Z`)), fila.persona,
        fila.unidad, ...configuracion.conceptos_incluidos.map((clave) => importeMonetario(fila.conceptos_centimos[clave])),
        importeMonetario(fila.importe_incluido_centimos)];
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
    form.hidden = true; filtrosVisibles.hidden = true; resumen.hidden = true; listado.hidden = true; ayuda.hidden = true;
    abrirAyuda.hidden = true; abrirAyuda.setAttribute("aria-expanded", "false");
    try {
      const [datos, catalogo] = await Promise.all([
        cargarDatos({ signal: controlador.signal }), cargarConfiguracion({ signal: controlador.signal }),
      ]);
      if (!activa || propia !== generacion) return;
      const criterio = congelarConfiguracion(catalogo, datos);
      const fuente = congelarDatos(datos, criterio, localizacion);
      registros = fuente.registros; moneda = fuente.moneda; configuracion = criterio;
      pintarAyuda(criterio);
      elegir(persona, registros, "persona_ref", "persona", filtros.persona || "");
      elegir(unidad, registros, "unidad_ref", "unidad", filtros.unidad || "");
      elegir(situacion, criterio.estados_incluidos.map((clave) => ({ situacion: clave, etiqueta: t(`situacion_${clave}`) })),
        "situacion", "etiqueta", filtros.situacion || "");
      desde.value = filtros.desde || ""; hasta.value = filtros.hasta || "";
      limpiarErrorFecha();
      disponible = true; form.hidden = false; filtrosVisibles.hidden = false;
      resumen.hidden = false; listado.hidden = false; abrirAyuda.hidden = false;
      pintarFiltrosAplicados();
      pagina = 0; pintar(); anunciarResultado();
      if (focoReintento && documento.activeElement === estado) resumen.focus();
    } catch (error) {
      if (!activa || propia !== generacion || controlador.signal.aborted) return;
      registros = Object.freeze([]); configuracion = undefined; resumen.replaceChildren(); listado.replaceChildren();
      mostrarEstado(error instanceof TypeError && error.message === "datos" ? "datos_error" : "carga_error");
      reintentar.hidden = false;
      if (focoReintento && documento.activeElement === estado) reintentar.focus();
    }
  }

  function enviar(evento) {
    evento.preventDefault(); limpiarErrorFecha();
    if (!disponible) return;
    const fechasInvalidas = [desde, hasta].filter((campo) => campo.value && !fechaValida(campo.value));
    if (fechasInvalidas.length) {
      for (const campo of fechasInvalidas) campo.setAttribute("aria-invalid", "true");
      actualizarAvisoEdicion(); mostrarEstado("fecha_error"); estado.focus(); return;
    }
    if (desde.value && hasta.value && desde.value > hasta.value) {
      desde.setAttribute("aria-invalid", "true"); hasta.setAttribute("aria-invalid", "true");
      actualizarAvisoEdicion(); mostrarEstado("periodo_error"); estado.focus(); return;
    }
    filtros = Object.freeze({ persona: persona.value, unidad: unidad.value, situacion: situacion.value,
      desde: desde.value, hasta: hasta.value });
    pagina = 0; pintarFiltrosAplicados(); pintar(); anunciarResultado();
  }
  function borrar() {
    if (!disponible) return;
    persona.value = ""; unidad.value = ""; situacion.value = ""; desde.value = ""; hasta.value = "";
    limpiarErrorFecha(); filtros = Object.freeze({}); pagina = 0;
    pintarFiltrosAplicados(); pintar(); anunciarResultado();
  }
  function paginar(evento) {
    const destino = evento.target?.dataset?.dietasInformesPagina;
    if (destino === undefined || evento.target.disabled) return;
    const nueva = Number(destino); const paginas = Math.ceil(resumirInformesDietas(registros, configuracion, filtros).registros.length / TAMANO_PAGINA);
    if (!Number.isInteger(nueva) || nueva < 0 || nueva >= paginas) return;
    pagina = nueva; pintar(); listado.querySelector?.("[data-dietas-informes-cuenta]")?.focus();
  }
  form.addEventListener("submit", enviar); limpiar.addEventListener("click", borrar);
  form.addEventListener("input", actualizarAvisoEdicion); form.addEventListener("change", actualizarAvisoEdicion);
  abrirAyuda.addEventListener("click", cambiarAyuda);
  reintentar.addEventListener("click", cargar); listado.addEventListener("click", paginar);
  cargar();
  function desmontar() {
    if (!activa) return; activa = false; generacion += 1; controlador?.abort();
    form.removeEventListener("submit", enviar); limpiar.removeEventListener("click", borrar);
    form.removeEventListener("input", actualizarAvisoEdicion); form.removeEventListener("change", actualizarAvisoEdicion);
    abrirAyuda.removeEventListener("click", cambiarAyuda);
    reintentar.removeEventListener("click", cargar); listado.removeEventListener("click", paginar); raiz.remove();
  }
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
