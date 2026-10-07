import { cargarTextos } from "../../../comun/textos.js";
import { ErrorClienteRPTPublicaV2 } from "./cliente-http-rpt-publica-v2.js";

const textos = await cargarTextos("personal-rpt-v2");
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
  const titulo = n(documento, "h2", t("titulo"));
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
  const resumen = n(documento, "div"); resumen.className = "rejilla-kpi";
  for (const [clave, valor, vista] of [
    ["filas", pagina.resumen.puestos, "puestos"], ["dotaciones", pagina.resumen.dotacion, "puestos"],
    ["categorias_resumen", pagina.resumen.categorias, "categorias"],
  ]) {
    const tarjeta = n(documento, "section"); tarjeta.className = "tarjeta-kpi";
    const abrir = n(documento, "button"); abrir.type = "button"; abrir.dataset.personalRptV2Resumen = clave;
    abrir.append(n(documento, "strong", numero(valor)), n(documento, "span", t(clave)));
    abrir.addEventListener("click", () => recargar({ vista, q: "", categoria_clave: "", centro_codigo: "", offset: 0 }));
    tarjeta.append(abrir);
    resumen.append(tarjeta);
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

export async function montarModuloRPTPublicaV2({ raiz, cliente, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || !cliente?.listar || typeof anunciar !== "function" ||
      registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") throw new TypeError("módulo RPT publicada no disponible");
  const documento = raiz.ownerDocument;
  const contenedor = n(documento, "section"); contenedor.className = "modulo-personal"; contenedor.dataset.personalRptPublicaV2 = "";
  raiz.append(contenedor);
  let activa = true, controlador = null, consulta = Object.freeze({ vista: "categorias", q: "", limit: 25, offset: 0, categoria_clave: "", centro_codigo: "" });
  const sigueMontada = () => raiz.querySelector?.("[data-personal-rpt-publica-v2]") === contenedor;
  const desmontar = () => { if (!activa) return; activa = false; controlador?.abort(); contenedor.remove?.(); };
  registrarDesmontar?.(desmontar);
  const pintar = (tipo, pagina = null, fallo = null) => {
    if (!activa || !sigueMontada()) return;
    const entradaAnterior = contenedor.querySelector?.("[data-personal-rpt-v2-busqueda]");
    const recuperarFoco = documento.activeElement === entradaAnterior;
    const formularioAnterior = contenedor.querySelector?.("[data-personal-rpt-v2-filtros]");
    const valorEntrada = formularioAnterior?.dataset?.personalRptV2Consulta === consulta.q ? (entradaAnterior?.value ?? consulta.q) : consulta.q;
    contenedor.replaceChildren(...cabecera(documento, pagina));
    if (tipo === "disponible") contenedor.append(...estadoFuente(documento, pagina, recargar));
    if (tipo === "cargando") { const aviso = n(documento, "p", t("cargando")); aviso.setAttribute("role", "status"); aviso.setAttribute("aria-live", "polite"); contenedor.append(aviso); }
    if (tipo === "error") {
      const aviso = n(documento, "p", mensajeError(fallo)); aviso.setAttribute("role", "alert"); contenedor.append(aviso);
      const reintentar = n(documento, "button", t("reintentar")); reintentar.type = "button"; reintentar.addEventListener("click", () => recargar()); contenedor.append(reintentar);
    }
    contenedor.append(pestanas(documento, consulta, recargar), filtros(documento, consulta, recargar, valorEntrada));
    if (tipo === "disponible") contenedor.append(tabla(documento, pagina, recargar), paginacion(documento, pagina, recargar));
    if (recuperarFoco) contenedor.querySelector?.("[data-personal-rpt-v2-busqueda]")?.focus?.();
  };
  const recargar = async (cambios = {}) => {
    if (!activa || !sigueMontada()) return;
    const siguiente = { ...consulta, ...cambios };
    if (!["categorias", "puestos"].includes(siguiente.vista) || typeof siguiente.q !== "string" || siguiente.q.length > 100 ||
        !Number.isSafeInteger(siguiente.offset) || siguiente.offset < 0 || siguiente.offset > 100000 ||
        (siguiente.categoria_clave !== "" && !/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/u.test(siguiente.categoria_clave)) ||
        (siguiente.centro_codigo !== "" && !/^[A-Za-z0-9-]{1,64}$/u.test(siguiente.centro_codigo)) ||
        (siguiente.vista !== "puestos" && (siguiente.categoria_clave || siguiente.centro_codigo))) { pintar("error", null, null); return; }
    controlador?.abort(); const vuelo = new AbortController(); controlador = vuelo;
    consulta = Object.freeze(siguiente); pintar("cargando");
    try {
      const pagina = await cliente.listar(consulta, { signal: vuelo.signal });
      if (activa && controlador === vuelo && !vuelo.signal.aborted) pintar("disponible", pagina);
    } catch (err) {
      if (activa && controlador === vuelo && !vuelo.signal.aborted) { anunciar(mensajeError(err), "error"); pintar("error", null, err); }
    } finally { if (controlador === vuelo) controlador = null; }
  };
  await recargar();
  return Object.freeze({ desmontar });
}
