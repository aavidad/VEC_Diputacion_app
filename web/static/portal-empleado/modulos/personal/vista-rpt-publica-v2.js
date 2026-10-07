import { cargarTextos } from "../../../comun/textos.js";
import { icono } from "../../../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { ErrorClienteRPTPublicaV2 } from "./cliente-http-rpt-publica-v2.js";

const PARAMETROS_RPT = Object.freeze({ vista: "rpt_vista", q: "rpt_q", categoria_clave: "rpt_categoria", centro_codigo: "rpt_centro", limit: "rpt_limite", offset: "rpt_offset" });
const CONSULTA_INICIAL_RPT = Object.freeze({ vista: "categorias", q: "", limit: 25, offset: 0, categoria_clave: "", centro_codigo: "" });

function consultaRPTValida(v) {
  return v && ["categorias", "puestos"].includes(v.vista) && typeof v.q === "string" && v.q === v.q.trim() &&
    [...v.q].length <= 100 && !/[\x00-\x1F\x7F-\x9F]/u.test(v.q) &&
    Number.isSafeInteger(v.limit) && v.limit >= 1 && v.limit <= 100 &&
    Number.isSafeInteger(v.offset) && v.offset >= 0 && v.offset <= 100000 &&
    typeof v.categoria_clave === "string" && (v.categoria_clave === "" || v.categoria_clave.length <= 60 && /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/u.test(v.categoria_clave)) &&
    typeof v.centro_codigo === "string" && (v.centro_codigo === "" || /^[A-Za-z0-9-]{1,64}$/u.test(v.centro_codigo)) &&
    (v.vista === "puestos" || v.categoria_clave === "" && v.centro_codigo === "");
}

function consultaRPTDesdeURL(ventana) {
  if (!ventana?.location?.href) return { ...CONSULTA_INICIAL_RPT };
  try {
    const params = new URL(ventana.location.href).searchParams;
    for (const nombre of Object.values(PARAMETROS_RPT)) if (params.getAll(nombre).length > 1) return null;
    const decimal = (nombre, defecto, minimo, maximo) => {
      const valor = params.get(nombre);
      if (valor === null) return defecto;
      if (!/^(0|[1-9][0-9]*)$/u.test(valor)) return NaN;
      const n = Number(valor);
      return Number.isSafeInteger(n) && n >= minimo && n <= maximo ? n : NaN;
    };
    const consulta = {
      vista: params.get(PARAMETROS_RPT.vista) ?? CONSULTA_INICIAL_RPT.vista,
      q: params.get(PARAMETROS_RPT.q) ?? "",
      categoria_clave: params.get(PARAMETROS_RPT.categoria_clave) ?? "",
      centro_codigo: params.get(PARAMETROS_RPT.centro_codigo) ?? "",
      limit: decimal(PARAMETROS_RPT.limit, CONSULTA_INICIAL_RPT.limit, 1, 100),
      offset: decimal(PARAMETROS_RPT.offset, CONSULTA_INICIAL_RPT.offset, 0, 100000),
    };
    return consultaRPTValida(consulta) ? consulta : null;
  } catch { return null; }
}

function escribirConsultaRPTEnURL(ventana, consulta, modo) {
  if (!ventana?.location?.href || !ventana.history || modo === "none") return;
  const destino = new URL(ventana.location.href);
  for (const [campo, nombre] of Object.entries(PARAMETROS_RPT)) destino.searchParams.set(nombre, String(consulta[campo]));
  if (destino.href === ventana.location.href) return;
  if (modo === "replace") ventana.history.replaceState(ventana.history.state, "", destino.href);
  else ventana.history.pushState(ventana.history.state, "", destino.href);
}

function crearMontadorRPTPublicaV2(textos) {
const t = (clave, variables = {}) => textos.traducir(`general.${clave}`, variables);
const n = (documento, etiqueta, contenido = "") => {
  const elemento = documento.createElement(etiqueta);
  if (contenido !== "") elemento.textContent = contenido;
  return elemento;
};
const numero = (valor) => textos.numero(valor, { useGrouping: "always" });
const dinero = (centimos) => new Intl.NumberFormat(textos.localizacion, { style: "currency", currency: "EUR" }).format(centimos / 100);

function cabecera(documento, pagina) {
  const salida = n(documento, "header"); salida.className = "cabecera-vista";
  const titulo = n(documento, "h2", t("titulo")); titulo.tabIndex = -1; titulo.dataset.personalRptV2Titulo = "";
  const boton = n(documento, "button", "?"); boton.type = "button";
  boton.setAttribute("aria-label", t("ayuda_boton")); boton.setAttribute("aria-controls", "personal-rpt-v2-ayuda"); boton.setAttribute("aria-expanded", "false");
  boton.dataset.personalRptV2AyudaBoton = "";
  salida.append(titulo, boton);
  const ayuda = n(documento, "section"); ayuda.id = "personal-rpt-v2-ayuda"; ayuda.className = "panel ayuda-contextual"; ayuda.hidden = true;
  const cuerpo = n(documento, "div"); cuerpo.className = "cuerpo-panel";
  cuerpo.append(n(documento, "p", t("ayuda")));
  if (pagina) {
    const fecha = textos.fecha(new Date(`${pagina.corte}T12:00:00Z`), { dateStyle: "long", timeZone: "Europe/Madrid" });
    cuerpo.append(n(documento, "p", t("fecha_documento", { fecha })), n(documento, "p", t("origen_documento", { nombre: pagina.fuente.documento })));
    const huella = n(documento, "p", t("huella", { huella: pagina.huella_sha256 })); huella.className = "rpt-huella";
    const recibo = n(documento, "p", t("recibo", { recibo: pagina.evidencia.recibo_ref })); recibo.className = "rpt-huella";
    cuerpo.append(huella, recibo);
  }
  ayuda.append(cuerpo);
  boton.addEventListener("click", () => { ayuda.hidden = !ayuda.hidden; boton.setAttribute("aria-expanded", String(!ayuda.hidden)); });
  boton.addEventListener("keydown", (evento) => { if (evento.key === "Escape") { evento.preventDefault(); ayuda.hidden = true; boton.setAttribute("aria-expanded", "false"); boton.focus?.(); } });
  return [salida, ayuda];
}

function estadoFuente(documento, pagina, recargar) {
  const panel = n(documento, "section"); panel.className = "panel"; panel.dataset.personalRptV2Estado = "";
  const cuerpo = n(documento, "div"); cuerpo.className = "cuerpo-panel";
  const etiqueta = n(documento, "strong", t("estado")); etiqueta.className = "estado-avisado";
  cuerpo.append(etiqueta); panel.append(cuerpo);
  const resumen = n(documento, "div"); resumen.className = "rejilla-kpi rejilla-kpi--compacta";
  for (const [clave, valor, vista, marca] of [
    ["filas", pagina.resumen.puestos, "puestos", "documento"],
    ["dotaciones", pagina.resumen.dotacion, "puestos", "grafico"],
    ["categorias_resumen", pagina.resumen.categorias, "categorias", "reglas"],
  ]) {
    const abrir = n(documento, "button"); abrir.type = "button"; abrir.className = "tarjeta-kpi"; abrir.dataset.personalRptV2Resumen = clave;
    const pictograma = n(documento, "span"); pictograma.className = "icono-kpi"; pictograma.setAttribute("aria-hidden", "true");
    pictograma.innerHTML = icono(marca);
    const cifras = n(documento, "span");
    const valorVisible = n(documento, "strong", numero(valor)); valorVisible.className = "valor-kpi";
    const etiquetaVisible = n(documento, "span", t(clave)); etiquetaVisible.className = "etiqueta-kpi";
    cifras.append(valorVisible, etiquetaVisible); abrir.append(pictograma, cifras);
    abrir.addEventListener("click", () => recargar({ vista, q: "", categoria_clave: "", centro_codigo: "", offset: 0 }));
    resumen.append(abrir);
  }
  return [panel, resumen];
}

function pestanas(documento, consulta, recargar) {
  const nav = n(documento, "nav"); nav.setAttribute("aria-label", t("titulo"));
  for (const [vista, clave] of [["categorias", "pestana_categorias"], ["puestos", "pestana_puestos"]]) {
    const boton = n(documento, "button", t(clave)); boton.type = "button"; boton.dataset.personalRptV2Vista = vista;
    boton.setAttribute("aria-pressed", String(consulta.vista === vista)); boton.disabled = consulta.vista === vista;
    boton.addEventListener("click", () => recargar({ vista, categoria_clave: "", centro_codigo: "", offset: 0 })); nav.append(boton);
  }
  return nav;
}

function filtros(documento, consulta, recargar, valorEntrada = consulta.q) {
  const form = n(documento, "form"); form.className = "panel filtros-quitables"; form.dataset.personalRptV2Filtros = "";
  form.dataset.personalRptV2Consulta = consulta.q;
  const label = n(documento, "label"); label.append(n(documento, "span", t("buscar")));
  const entrada = n(documento, "input"); entrada.type = "search"; entrada.name = "q"; entrada.value = valorEntrada; entrada.maxLength = 100; entrada.dataset.personalRptV2Busqueda = "";
  label.append(entrada);
  const buscar = n(documento, "button", t("accion_buscar")); buscar.type = "submit"; buscar.className = "boton-primario";
  form.append(label, buscar);
  if (consulta.q) {
    const activo = n(documento, "div"); activo.className = "filtros-activos";
    activo.append(n(documento, "span", t("busqueda_aplicada", { busqueda: consulta.q })));
    const quitar = n(documento, "button", t("quitar_busqueda")); quitar.type = "button"; quitar.className = "boton-secundario"; quitar.dataset.personalRptV2QuitarBusqueda = "";
    quitar.addEventListener("click", () => recargar({ q: "", offset: 0 })); activo.append(quitar); form.append(activo);
  }
  if (consulta.categoria_clave || consulta.centro_codigo) {
    const activos = n(documento, "div"); activos.className = "filtros-activos";
    if (consulta.categoria_clave) {
      activos.append(n(documento, "span", t("categoria_filtro")));
      const quitar = n(documento, "button", t("quitar_categoria")); quitar.type = "button"; quitar.className = "boton-secundario";
      quitar.addEventListener("click", () => recargar({ categoria_clave: "", offset: 0 })); activos.append(quitar);
    }
    if (consulta.centro_codigo) {
      activos.append(n(documento, "span", t("centro_filtro", { codigo: consulta.centro_codigo })));
      const quitar = n(documento, "button", t("quitar_centro")); quitar.type = "button"; quitar.className = "boton-secundario";
      quitar.addEventListener("click", () => recargar({ centro_codigo: "", offset: 0 })); activos.append(quitar);
    }
    form.append(activos);
  }
  form.addEventListener("submit", (evento) => { evento.preventDefault(); recargar({ q: entrada.value.trim(), offset: 0 }); });
  return form;
}

function origen(item) { return item.origen === "categoria" ? t("categoria_columna") : t("categoria_denominacion"); }
function valorGrupo(item) { return item.grupos.length ? item.grupos.join(", ") : t("sin_grupo"); }

function detallePuesto(documento, item, cerrar) {
  const contenedor = n(documento, "section");
  const botonCerrar = n(documento, "button", t("cerrar_detalle")); botonCerrar.type = "button";
  botonCerrar.addEventListener("click", cerrar); contenedor.append(botonCerrar);
  const lista = n(documento, "dl");
  const datos = [
    ["delegacion", item.delegacion], ["escala", item.escala || t("sin_dato")],
    ["nivel", String(item.nivel_destino)], ["complemento", dinero(item.complemento_especifico_anual_centimos)],
    ["tipo", item.tipo], ["provision", item.provision],
    ["categoria_referencias", item.categorias_claves.length ? item.categorias_claves.join(", ") : t("categoria_sin_referencia")],
  ];
  for (const [clave, valor] of datos) {
    if (clave === "categoria_referencias") lista.append(n(documento, "dt", t("categoria_pendiente")), n(documento, "dd", valor));
    else lista.append(n(documento, "dt", t(clave)), n(documento, "dd", valor));
  }
  if (item.categorias_pendientes.length) {
    lista.append(n(documento, "dt", t("categoria_pendiente")), n(documento, "dd", t("categoria_alternativas", { nombres: item.categorias_pendientes.map((p) => p.denominacion).join(", ") })));
  }
  contenedor.append(lista);
  return contenedor;
}

function tabla(documento, pagina, recargar) {
  const contenedor = n(documento, "div"); contenedor.className = "tabla-contenedor"; contenedor.setAttribute("role", "region"); contenedor.setAttribute("tabindex", "0");
  contenedor.setAttribute("aria-label", t(pagina.vista === "puestos" ? "tabla_puestos" : "tabla_categorias"));
  const tabla = n(documento, "table"); tabla.className = "tabla-datos";
  tabla.append(n(documento, "caption", t(pagina.vista === "puestos" ? "tabla_puestos" : "tabla_categorias")));
  const columnas = pagina.vista === "puestos" ? ["denominacion", "codigo", "centro", "grupos", "dotacion"] : ["denominacion", "origen", "grupos"];
  const thead = n(documento, "thead"), tr = n(documento, "tr");
  for (const clave of columnas) { const th = n(documento, "th", t(clave)); th.setAttribute("scope", "col"); tr.append(th); }
  thead.append(tr); tabla.append(thead);
  const tbody = n(documento, "tbody");
  if (!pagina.items.length) {
    const fila = n(documento, "tr"), celda = n(documento, "td", t("vacio")); celda.colSpan = columnas.length; fila.append(celda); tbody.append(fila);
  }
  pagina.items.forEach((item, indice) => {
    const fila = n(documento, "tr"), esPuesto = pagina.vista === "puestos";
    const valores = esPuesto ? [item.denominacion, item.codigo, item.centro, valorGrupo(item), numero(item.dotacion)] :
      [item.denominacion, origen(item), valorGrupo(item)];
    valores.forEach((valor, columna) => {
      const celda = n(documento, columna === 0 ? "th" : "td");
      if (columna === 0) celda.setAttribute("scope", "row");
      if (esPuesto && columna === 0) {
        const boton = n(documento, "button", valor); boton.type = "button"; boton.dataset.personalRptV2DetalleBoton = "";
        boton.setAttribute("aria-label", t("ver_detalle", { nombre: valor })); boton.setAttribute("aria-expanded", "false");
        boton.setAttribute("aria-controls", `personal-rpt-v2-detalle-${pagina.offset}-${indice}`);
        boton.addEventListener("click", () => {
          const actual = tbody.querySelector?.(`[data-personal-rpt-v2-detalle="${indice}"]`);
          if (actual) { actual.remove(); boton.setAttribute("aria-expanded", "false"); return; }
          const detalle = n(documento, "tr"); detalle.id = `personal-rpt-v2-detalle-${pagina.offset}-${indice}`; detalle.dataset.personalRptV2Detalle = String(indice);
          const cerrar = () => { detalle.remove(); boton.setAttribute("aria-expanded", "false"); boton.focus?.(); };
          const c = n(documento, "td"); c.colSpan = columnas.length; c.append(detallePuesto(documento, item, cerrar)); detalle.append(c);
          detalle.addEventListener("keydown", (evento) => { if (evento.key === "Escape") { evento.preventDefault(); cerrar(); } });
          fila.after?.(detalle); boton.setAttribute("aria-expanded", "true");
        }); celda.append(boton);
      } else if (!esPuesto && columna === 0) {
        const boton = n(documento, "button", valor); boton.type = "button"; boton.dataset.personalRptV2Categoria = item.clave;
        boton.setAttribute("aria-label", t("ver_puestos_categoria", { nombre: valor }));
        boton.addEventListener("click", () => recargar({ vista: "puestos", q: "", categoria_clave: item.clave, centro_codigo: "", offset: 0 }));
        celda.append(boton);
      } else if (esPuesto && columna === 2) {
        const boton = n(documento, "button", valor); boton.type = "button"; boton.dataset.personalRptV2Centro = item.centro_codigo;
        boton.setAttribute("aria-label", t("ver_puestos_centro", { nombre: valor }));
        boton.addEventListener("click", () => recargar({ vista: "puestos", q: "", categoria_clave: "", centro_codigo: item.centro_codigo, offset: 0 }));
        celda.append(boton);
      } else celda.textContent = valor;
      fila.append(celda);
    }); tbody.append(fila);
  });
  tabla.append(tbody); contenedor.append(tabla); return contenedor;
}

function paginacion(documento, pagina, recargar) {
  const nav = n(documento, "nav"); nav.setAttribute("aria-label", t("paginacion"));
  const desde = pagina.offset < pagina.total ? pagina.offset + 1 : 0;
  const hasta = pagina.items.length ? pagina.offset + pagina.items.length : 0;
  nav.append(n(documento, "span", t("mostrando", { desde: numero(desde), hasta: numero(hasta), total: numero(pagina.total) })));
  const anterior = n(documento, "button", t("anterior")); anterior.type = "button"; anterior.disabled = pagina.offset === 0;
  anterior.addEventListener("click", () => recargar({ offset: Math.max(0, pagina.offset - pagina.limit) }));
  const siguiente = n(documento, "button", t("siguiente")); siguiente.type = "button"; siguiente.disabled = pagina.offset + pagina.items.length >= pagina.total;
  siguiente.addEventListener("click", () => recargar({ offset: pagina.offset + pagina.limit }));
  nav.append(anterior, siguiente); return nav;
}

function mensajeError(error) {
  if (error instanceof ErrorClienteRPTPublicaV2 && error.codigo === "autenticacion_requerida") return t("error_auth");
  if (error instanceof ErrorClienteRPTPublicaV2 && error.codigo === "acceso_denegado") return t("error_denied");
  return t("error_unavailable");
}

async function montarConTextos({ raiz, cliente, anunciar = () => {}, registrarDesmontar, ventana = raiz?.ownerDocument?.defaultView ?? globalThis.window } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || !cliente?.listar || typeof anunciar !== "function" ||
      registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") throw new TypeError("módulo RPT publicada no disponible");
  const documento = raiz.ownerDocument;
  const contenedor = n(documento, "section"); contenedor.className = "modulo-personal"; contenedor.dataset.personalRptPublicaV2 = "";
  raiz.append(contenedor);
  let activa = true, controlador = null, consulta = Object.freeze(consultaRPTDesdeURL(ventana) ?? { ...CONSULTA_INICIAL_RPT });
  const sigueMontada = () => raiz.querySelector?.("[data-personal-rpt-publica-v2]") === contenedor;
  const alVolver = () => queueMicrotask(() => {
    if (!activa || !sigueMontada()) return;
    const desdeURL = consultaRPTDesdeURL(ventana);
    void recargar(desdeURL ?? { ...CONSULTA_INICIAL_RPT }, desdeURL ? "none" : "replace");
  });
  ventana?.addEventListener?.("popstate", alVolver);
  const desmontar = () => { if (!activa) return; activa = false; controlador?.abort(); ventana?.removeEventListener?.("popstate", alVolver); contenedor.remove?.(); };
  registrarDesmontar?.(desmontar);
  const pintar = (tipo, pagina = null, fallo = null) => {
    if (!activa || !sigueMontada()) return;
    const entradaAnterior = contenedor.querySelector?.("[data-personal-rpt-v2-busqueda]");
    const recuperarFoco = Boolean(entradaAnterior && documento.activeElement === entradaAnterior);
    const retryAnterior = contenedor.querySelector?.("[data-personal-rpt-v2-reintentar]");
    const cargaAnterior = contenedor.querySelector?.("[data-personal-rpt-v2-carga]");
    const recuperarTrasRetry = Boolean(retryAnterior && documento.activeElement === retryAnterior || cargaAnterior && documento.activeElement === cargaAnterior);
    const formularioAnterior = contenedor.querySelector?.("[data-personal-rpt-v2-filtros]");
    const valorEntrada = formularioAnterior?.dataset?.personalRptV2Consulta === consulta.q ? (entradaAnterior?.value ?? consulta.q) : consulta.q;
    contenedor.replaceChildren(...cabecera(documento, pagina));
    if (tipo === "disponible") contenedor.append(...estadoFuente(documento, pagina, recargar));
    if (tipo === "cargando") { const aviso = n(documento, "p", t("cargando")); aviso.setAttribute("role", "status"); aviso.setAttribute("aria-live", "polite"); aviso.tabIndex = -1; aviso.dataset.personalRptV2Carga = ""; contenedor.append(aviso); }
    if (tipo === "error") {
      const aviso = n(documento, "p", mensajeError(fallo)); aviso.setAttribute("role", "alert"); contenedor.append(aviso);
      const reintentar = n(documento, "button", t("reintentar")); reintentar.type = "button"; reintentar.dataset.personalRptV2Reintentar = ""; reintentar.addEventListener("click", () => recargar()); contenedor.append(reintentar);
    }
    contenedor.append(pestanas(documento, consulta, recargar), filtros(documento, consulta, recargar, valorEntrada));
    if (tipo === "disponible") contenedor.append(tabla(documento, pagina, recargar), paginacion(documento, pagina, recargar));
    if (recuperarTrasRetry) {
      const destino = tipo === "cargando" ? "[data-personal-rpt-v2-carga]" :
        tipo === "error" ? "[data-personal-rpt-v2-reintentar]" : "[data-personal-rpt-v2-titulo]";
      contenedor.querySelector?.(destino)?.focus?.();
    } else if (recuperarFoco) contenedor.querySelector?.("[data-personal-rpt-v2-busqueda]")?.focus?.();
  };
  const recargar = async (cambios = {}, historial = "push") => {
    if (!activa || !sigueMontada()) return;
    const siguiente = { ...consulta, ...cambios };
    if (!consultaRPTValida(siguiente)) { pintar("error", null, null); return; }
    controlador?.abort(); const vuelo = new AbortController(); controlador = vuelo;
    consulta = Object.freeze(siguiente);
    escribirConsultaRPTEnURL(ventana, consulta, historial);
    pintar("cargando");
    try {
      const pagina = await cliente.listar(consulta, { signal: vuelo.signal });
      if (activa && controlador === vuelo && !vuelo.signal.aborted) pintar("disponible", pagina);
    } catch (err) {
      if (activa && controlador === vuelo && !vuelo.signal.aborted) { anunciar(mensajeError(err), "error"); pintar("error", null, err); }
    } finally { if (controlador === vuelo) controlador = null; }
  };
  await recargar({}, "replace");
  return Object.freeze({ desmontar });
}
return montarConTextos;
}

// El módulo se puede importar aunque falte el catálogo. Una lectura fallida
// deja reintentar el montaje con el mismo URL cuando vuelva el lector común.
export async function montarModuloRPTPublicaV2({ cargarCatalogo = cargarTextos, ...opciones } = {}) {
  if (typeof cargarCatalogo !== "function") throw new TypeError("lector de textos no disponible");
  const textos = await cargarCatalogo("personal-rpt-v2");
  if (!textos || typeof textos.traducir !== "function" || typeof textos.numero !== "function" || typeof textos.fecha !== "function") {
    throw new TypeError("catálogo de textos no disponible");
  }
  return crearMontadorRPTPublicaV2(textos)(opciones);
}
