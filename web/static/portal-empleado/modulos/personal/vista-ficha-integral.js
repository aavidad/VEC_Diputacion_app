import { montarVistaHistoriaRelacionesPropia } from "./vista-historia-relaciones-propia.js?v=20261004-personal-relaciones-v1";
import { montarVistaHistoriaServiciosPropia } from "./vista-historia-servicios-propia.js?v=20261004-b-revision-valor-v1";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import { crearTraductorPersonal } from "./i18n.js?v=20261007-pantallas-textos-final-v1";

import { crearSelectorCorteServicios, esFechaCorteServicios, presentarFechaCorteServicios, traducirCorteServicios } from "./ficha-propia-corte.js?v=20261002-personal-servicios-csv-v1";

import { descargarResumenServicios } from "./servicios-descarga.js?v=20261004-personal-historia-v1";
import { traducirExportacionServicios } from "./i18n-exportacion-servicios.js?v=20261004-personal-historia-v1";
import { referenciaExportacionServiciosValida } from "./cliente-http-exportacion-servicios.js?v=20261004-personal-historia-v1";

const PESTANAS = Object.freeze([
  ["ficha", "ficha_tab_ficha"], ["contacto", "ficha_tab_contacto"], ["relaciones", "ficha_tab_relaciones"],
  ["servicios", "ficha_tab_servicios"], ["tiempo", "ficha_tab_tiempo"],
  ["formacion", "ficha_tab_formacion"], ["economia", "ficha_tab_economia"],
  ["documentos", "ficha_tab_documentos"], ["catalogos", "ficha_tab_catalogos"],
]);
const BLOQUES = Object.freeze({
  relaciones: { titulo: "ficha_relaciones_titulo", ayuda: "ficha_empleo_ayuda", columnas: [["desde", "ficha_cab_inicio"], ["hasta", "ficha_cab_fin"], ["regimen", "ficha_cab_regimen"], ["puesto", "ficha_cab_puesto"], ["unidad", "ficha_cab_unidad"], ["estado", "ficha_cab_estado"]] },
  servicios: { titulo: "ficha_servicios_titulo", ayuda: "ficha_servicios_ayuda", columnas: [["desde", "ficha_cab_inicio"], ["hasta", "ficha_cab_fin"], ["procedencia", "ficha_cab_procedencia"], ["reconocimiento", "ficha_cab_reconocimiento"], ["estado", "ficha_cab_estado"]] },
  tiempo: { titulo: "ficha_tiempo_titulo", ayuda: "ficha_tiempo_ayuda", columnas: [["fecha", "ficha_cab_fecha"], ["tipo", "ficha_cab_tipo"], ["estado", "ficha_cab_estado"]] },
  formacion: { titulo: "ficha_formacion_titulo", ayuda: "ficha_formacion_ayuda", columnas: [["curso", "ficha_cab_curso"], ["entidad", "ficha_cab_entidad"], ["fecha", "ficha_cab_fecha"], ["resultado", "ficha_cab_resultado"], ["acreditacion", "ficha_cab_acreditacion"]] },
  economia: { titulo: "ficha_economia_titulo", ayuda: "ficha_economia_ayuda", columnas: [["documento", "ficha_cab_documento"], ["periodo", "ficha_cab_periodo"], ["estado", "ficha_cab_estado"]] },
  documentos: { titulo: "ficha_documentos_titulo", ayuda: "ficha_documentos_ayuda", columnas: [["documento", "ficha_cab_documento"], ["fecha", "ficha_cab_fecha"], ["estado", "ficha_cab_estado"]] },
});
const ESTADOS = new Set(["disponible", "vacio", "no_configurado", "denegado", "excede_limite", "error"]);
const RESPUESTAS_RPT = new Set(["disponible", "incidencia", "ausente", "denegado"]);
/** Longitud máxima de una celda; la misma que valida el cliente de la ficha propia. */
export const LIMITE_TEXTO_CAMPO_FICHA = 300;
const ETIQUETAS_ESTADO = Object.freeze({
  sin_consulta: "ficha_estado_sin_consulta", cargando: "ficha_estado_cargando",
  disponible: "ficha_estado_disponible", vacio: "ficha_estado_vacio",
  no_configurado: "ficha_estado_no_configurado", denegado: "ficha_estado_denegado",
  excede_limite: "ficha_estado_excede_limite", error: "ficha_estado_error",
});

function nodo(d, etiqueta, texto) { const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; return n; }
function panel(d, titulo, contenido, clase = "") {
  const seccion = nodo(d, "section"); seccion.className = `panel personal-ficha-panel ${clase}`.trim();
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-panel"; cabecera.append(nodo(d, "h3", titulo));
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel"; cuerpo.append(...contenido); seccion.append(cabecera, cuerpo); return seccion;
}
function mensaje(d, texto, tipo = "status") { const p = nodo(d, "p", texto); p.className = "personal-ficha-mensaje"; p.setAttribute("role", tipo); return p; }
function ayuda(d, t) {
  const detalles = nodo(d, "details"); detalles.className = "ayuda-contextual"; detalles.dataset.personalFichaAyuda = "";
  const abrir = nodo(d, "summary", "?"); abrir.setAttribute("aria-label", t("ficha_abrir_ayuda"));
  abrir.setAttribute("tabindex", "0");
  const contexto = nodo(d, "p"); const corte = nodo(d, "p");
  detalles.append(abrir, nodo(d, "p", t("ficha_ayuda")), contexto, corte);
  return { elemento: detalles, mostrar(clave) { detalles.open = false; contexto.textContent = clave ? t(clave) : ""; corte.textContent = clave === "ficha_servicios_ayuda" ? traducirCorteServicios("ayuda") : ""; } };
}
function accesos(d, t, navegarModulo, destinosDisponibles, abrirCorreos) {
  const acciones = nodo(d, "div"); acciones.className = "acciones-fila personal-ficha-accesos";
  for (const [destino, etiqueta] of [["dietas", "ficha_ir_dietas"], ["cronos", "ficha_ir_cronos"]]) {
    // Un destino que no figura en la disponibilidad inyectada es un módulo que
    // este despliegue no muestra (VEC_PORTAL_MODULOS_VISIBLES): no se ofrece.
    if (!Object.hasOwn(destinosDisponibles, destino)) continue;
    const boton = nodo(d, "button", t(etiqueta)); boton.type = "button"; boton.dataset.personalFichaDestino = destino;
    // La disponibilidad de una ruta se inyecta por destino; no acredita permiso.
    const disponible = typeof navegarModulo === "function" && Object.hasOwn(destinosDisponibles, destino) && destinosDisponibles[destino] === true;
    boton.disabled = !disponible; boton.setAttribute("aria-disabled", String(!disponible));
    if (!disponible) { boton.title = t("ficha_navegacion_pendiente"); boton.setAttribute("aria-label", `${t(etiqueta)}. ${t("ficha_navegacion_pendiente")}`); }
    boton.addEventListener("click", () => { if (disponible) navegarModulo(destino); }); acciones.append(boton);
  }
  if (typeof abrirCorreos === "function") {
    const boton = nodo(d, "button", t("ficha_ir_mis_correos")); boton.type = "button";
    boton.className = "boton-secundario"; boton.dataset.personalFichaCorreos = "";
    boton.title = t("ficha_mis_correos_destino");
    boton.addEventListener("click", abrirCorreos); acciones.append(boton);
  }
  return acciones;
}
function portada(d, t, navegarModulo, destinosDisponibles, estados, visibles, ocultarSinFuente, abrirCorreos) {
  const accesosPanel = panel(d, t("ficha_accesos_titulo"), [accesos(d, t, navegarModulo, destinosDisponibles, abrirCorreos)], "personal-ficha-panel-ancho");
  if (ocultarSinFuente && visibles.length === 0) return [accesosPanel];
  const bloques = nodo(d, "div"); bloques.className = "personal-ficha-bloques";
  for (const clave of visibles) {
    const ficha = nodo(d, "div"); ficha.className = "personal-ficha-bloque";
    ficha.dataset.personalFichaEstado = estados[clave];
    const estado = nodo(d, "span", t(ETIQUETAS_ESTADO[estados[clave]])); estado.className = "personal-ficha-estado";
    ficha.append(nodo(d, "strong", t(BLOQUES[clave].titulo)), estado); bloques.append(ficha);
  }
  const resumen = ocultarSinFuente ? [] : [mensaje(d, t("ficha_fuente_pendiente"))];
  if (!ocultarSinFuente && Object.values(estados).every((estado) => estado === "no_configurado")) resumen.push(mensaje(d, t("ficha_sin_datos")));
  resumen.push(bloques);
  return [panel(d, t("ficha_resumen"), resumen, "personal-ficha-panel-ancho"), accesosPanel];
}
function validarResultado(resultado, bloque) {
  if (!resultado || typeof resultado !== "object" || !ESTADOS.has(resultado.estado)) throw new TypeError("respuesta de ficha no válida");
  if (resultado.estado !== "disponible" && resultado.estado !== "vacio") return { estado: resultado.estado, ...(resultado.estado === "denegado" && resultado.aviso_exportacion === "sesion_caducada" ? { aviso_exportacion: "sesion_caducada" } : {}) };
  if (typeof resultado.fuente !== "string" || !resultado.fuente.trim() || resultado.fuente.length > 160 ||
      typeof resultado.actualizado_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/u.test(resultado.actualizado_en) ||
      !Number.isFinite(Date.parse(resultado.actualizado_en)) ||
      !Array.isArray(resultado.items) || resultado.items.length > 200 ||
      (resultado.estado === "vacio" && resultado.items.length !== 0) || (resultado.estado === "disponible" && resultado.items.length === 0)) throw new TypeError("respuesta de ficha no válida");
  const columnas = BLOQUES[bloque].columnas.map(([campo]) => campo);
  const items = resultado.items.map((item) => {
    if (!item || typeof item !== "object") throw new TypeError("fila de ficha no válida");
    const visible = Object.fromEntries(columnas.map((campo) => {
      const valor = item[campo];
      if (valor !== undefined && (typeof valor !== "string" || valor.length > LIMITE_TEXTO_CAMPO_FICHA)) throw new TypeError("campo de ficha no válido");
      return [campo, valor ?? ""];
    }));
    if (!Object.values(visible).some((valor) => valor.trim())) throw new TypeError("fila de ficha sin datos visibles");
    return visible;
  });
  if (resultado.exportacion_servicios_disponible !== undefined && typeof resultado.exportacion_servicios_disponible !== "boolean") throw new TypeError("disponibilidad_exportacion_no_valida");
  if (resultado.fecha_referencia !== undefined && (bloque !== "servicios" || !esFechaCorteServicios(resultado.fecha_referencia))) throw new TypeError("fecha de referencia no válida");
  return { estado: resultado.estado, fuente: resultado.fuente, actualizado_en: resultado.actualizado_en, items,
    ...(resultado.fecha_referencia ? { fecha_referencia: resultado.fecha_referencia } : {}),
    ...(bloque === "relaciones" ? { historia_relaciones_disponible: resultado.historia_relaciones_disponible === true } : {}),
    ...(bloque === "servicios" ? { exportacion_servicios_disponible: resultado.exportacion_servicios_disponible === true, historia_servicios_disponible: resultado.historia_servicios_disponible === true } : {}),
    ...(bloque === "servicios" && referenciaExportacionServiciosValida(resultado.recibo_ref, resultado.corte) && resultado.corte.vigente_en === resultado.fecha_referencia ? { recibo_ref: resultado.recibo_ref, corte: Object.freeze({ ...resultado.corte }) } : {}) };
}
function formatearFecha(iso) {
  return new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(iso));
}
function mostrarCampo(campo, valor, t) {
  if (!valor) return t("ficha_no_consta");
  if (["desde", "hasta", "fecha"].includes(campo) && /^\d{4}-\d{2}-\d{2}$/u.test(valor)) {
    const fecha = new Date(`${valor}T12:00:00Z`);
    if (Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor) return new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(fecha);
  }
  if (campo === "periodo" && /^\d{4}-\d{2}$/u.test(valor)) {
    const fecha = new Date(`${valor}-15T12:00:00Z`);
    if (Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 7) === valor) return new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { month: "long", year: "numeric", timeZone: "Europe/Madrid" }).format(fecha);
  }
  return valor;
}
function tabla(d, t, bloque, items) {
  const definicion = BLOQUES[bloque]; const region = nodo(d, "div"); region.className = "tabla-contenedor personal-ficha-tabla";
  region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t(definicion.titulo));
  const tablaDatos = nodo(d, "table"); tablaDatos.className = "tabla-datos"; tablaDatos.append(nodo(d, "caption", t(definicion.titulo)));
  const cabecera = nodo(d, "thead"); const filaCabecera = nodo(d, "tr");
  for (const [, etiqueta] of definicion.columnas) { const th = nodo(d, "th", t(etiqueta)); th.setAttribute("scope", "col"); filaCabecera.append(th); }
  cabecera.append(filaCabecera); const cuerpo = nodo(d, "tbody");
  for (const item of items) {
    const fila = nodo(d, "tr");
    for (const [campo] of definicion.columnas) fila.append(nodo(d, "td", mostrarCampo(campo, item[campo], t)));
    cuerpo.append(fila);
  }
  tablaDatos.append(cabecera, cuerpo); region.append(tablaDatos);
  const conjunto = nodo(d, "div"); conjunto.className = "personal-ficha-tabla-conjunto";
  const indicacion = nodo(d, "p", t("ficha_desplazar_tabla")); indicacion.className = "personal-ficha-desplazar";
  conjunto.append(indicacion, region); return conjunto;
}
function pintarBloque(d, principal, t, bloque, resultado, actualizar, corte, descargar, abrirHistoria) {
  const definicion = BLOQUES[bloque]; const piezas = [];
  if (corte && !["denegado", "no_configurado"].includes(resultado.estado)) {
    piezas.push(crearSelectorCorteServicios(d, corte.referencia(), corte.consultar));
    piezas.push(mensaje(d, traducirCorteServicios("alcance")));
  }
  if (resultado.fecha_referencia) piezas.push(mensaje(d, traducirCorteServicios("al_corte", { fecha: presentarFechaCorteServicios(resultado.fecha_referencia) })));
  if (resultado.estado === "disponible" || resultado.estado === "vacio") {
    const metadatos = nodo(d, "p", t("ficha_procedencia", { fuente: resultado.fuente, fecha: formatearFecha(resultado.actualizado_en) }));
    metadatos.className = "personal-ficha-procedencia"; piezas.push(metadatos);
    piezas.push(resultado.estado === "vacio" ? mensaje(d, t("ficha_vacio")) : tabla(d, t, bloque, resultado.items));
  } else {
    const clave = { cargando: "ficha_cargando", no_configurado: "ficha_no_configurado", denegado: "ficha_denegado", excede_limite: "ficha_excede_limite", error: "ficha_error" }[resultado.estado];
    piezas.push(mensaje(d, resultado.aviso_exportacion === "sesion_caducada" ? traducirExportacionServicios("sesion_caducada") : t(clave), resultado.estado === "error" ? "alert" : "status"));
  }
  if (bloque === "servicios" && typeof descargar === "function" && ["disponible", "vacio"].includes(resultado.estado)) {
    const resumen = nodo(d, "div"); resumen.className = "acciones-fila";
    const boton = nodo(d, "button", traducirExportacionServicios("descargar"));
    boton.type = "button"; boton.className = "boton-secundario";
    boton.dataset.personalServiciosDescargar = "";
    boton.addEventListener("click", descargar); resumen.append(boton);
    const estado = mensaje(d, ""); estado.dataset.personalServiciosExportacionEstado = ""; estado.setAttribute("aria-live", "polite"); resumen.append(estado);
    piezas.push(mensaje(d, traducirExportacionServicios("alcance")), resumen);
  }
  if (["servicios", "relaciones"].includes(bloque) && typeof abrirHistoria === "function" && ["disponible", "vacio"].includes(resultado.estado)) {
    const boton=nodo(d,"button",t(bloque === "servicios" ? "ficha_ver_historia_servicios" : "ficha_ver_historia_relaciones"));boton.type="button";boton.className="boton-secundario";boton.dataset.personalHistoriaAbrir="";
    boton.addEventListener("click",abrirHistoria);piezas.push(boton);
  }
  if (typeof actualizar === "function" && (["disponible", "vacio", "excede_limite", "error"].includes(resultado.estado) || resultado.aviso_exportacion === "sesion_caducada")) {
    const accion = nodo(d, "button", t(resultado.estado === "error" ? "ficha_reintentar" : "ficha_actualizar"));
    accion.type = "button"; accion.className = "boton-secundario"; accion.dataset.personalFichaActualizar = bloque;
    accion.addEventListener("click", actualizar); piezas.push(accion);
  }
  principal.replaceChildren(panel(d, t(definicion.titulo), piezas, "personal-ficha-panel-ancho"));
}

/**
 * Vista de consulta propia. `fuentes[clave].consultarPropios({signal})` debe ser un
 * cliente de servidor que resuelva la identidad y autorice los campos; la vista
 * nunca recibe ni envía referencias de persona o empleado. Bloques sin cliente
 * permanecen no configurados y se consultan solo al abrir su pestaña.
 * Con `ocultarSinFuente` (portal real) los apartados sin cliente no se ofrecen,
 * no se muestran textos explicativos y, si no queda ninguno, se abre Catálogos.
 */
export function montarVistaFichaIntegralPersonal({ raiz, anunciar = () => {}, registrarDesmontar, montarCatalogos, montarContacto, navegarModulo, abrirCorreos, destinosDisponibles = {}, fuentes = {}, ocultarSinFuente = false, rptDisponible = false, rptIncidencia = false, reintentarRPT } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") ||
      (montarCatalogos !== undefined && typeof montarCatalogos !== "function") ||
      (montarContacto !== undefined && typeof montarContacto !== "function") ||
      (abrirCorreos !== undefined && typeof abrirCorreos !== "function") ||
      (navegarModulo !== undefined && typeof navegarModulo !== "function") || !destinosDisponibles || typeof destinosDisponibles !== "object" || Array.isArray(destinosDisponibles) ||
      !fuentes || typeof fuentes !== "object" || typeof ocultarSinFuente !== "boolean" ||
      typeof rptDisponible !== "boolean" || typeof rptIncidencia !== "boolean" ||
      (reintentarRPT !== undefined && typeof reintentarRPT !== "function") ||
      (rptDisponible && (rptIncidencia || !montarCatalogos)) ||
      (rptIncidencia && (!reintentarRPT || !montarCatalogos))) throw new TypeError("vista ficha integral de Personal no disponible");
  const d = raiz.ownerDocument; if (!d?.createElement) throw new TypeError("documento ficha integral de Personal no disponible");
  const t = crearTraductorPersonal(); const contenedor = nodo(d, "section"); contenedor.className = "modulo-personal";
  contenedor.dataset.personalFichaIntegral = ""; raiz.append(contenedor);
  const estados = Object.fromEntries(Object.keys(BLOQUES).map((clave) => [clave,
    Object.hasOwn(fuentes, clave) && typeof fuentes[clave]?.consultarPropios === "function"
      ? (fuentes[clave].estadoInicial === "error" ? "error" : "sin_consulta") : "no_configurado"]));
  const visibles = Object.keys(BLOQUES).filter((clave) => !ocultarSinFuente || estados[clave] !== "no_configurado");
  const pestanas = PESTANAS.filter(([clave]) => clave === "ficha" || visibles.includes(clave) || (clave === "contacto" && montarContacto) || (clave === "catalogos" && (!ocultarSinFuente || montarCatalogos)));
  let activa = true; let actual = !rptIncidencia && ocultarSinFuente && visibles.length === 0 && montarCatalogos ? "catalogos" : "ficha";
  let vuelo; let vueloExportacion; let limpiarHistoria; let limpiarCatalogos; let secuencia = 0; let referenciaServicios = ""; let serviciosDescargables;
  let estadoRPT = rptIncidencia ? "incidencia" : "sin_aviso", vueloRPT = null, controlRPT = null;
  const limpiar = () => { limpiarHistoria?.(); limpiarHistoria = undefined; const fn = limpiarCatalogos; limpiarCatalogos = undefined; fn?.(); };
  const desmontar = () => { if (!activa) return; activa = false; serviciosDescargables = undefined; secuencia += 1; vuelo?.abort(); vueloExportacion?.abort(); limpiar(); contenedor.remove?.(); };
  const caducarSesion = () => {
    try { for (const fuente of Object.values(fuentes)) fuente?.actualizar?.(); }
    finally {
      desmontar();
      const aviso = mensaje(d, t("ficha_sesion_caducada"), "alert"); aviso.setAttribute("tabindex", "-1");
      raiz.append(aviso); aviso.focus?.(); anunciar(aviso.textContent, "error");
    }
  };
  registrarDesmontar?.(desmontar);
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-vista";
  const ayudaFicha = ayuda(d, t);
  cabecera.append(nodo(d, "p", t("ficha_sobrelinea")), nodo(d, "h2", t("ficha_titulo")), ayudaFicha.elemento);
  const tabs = nodo(d, "div"); tabs.className = "personal-ficha-pestanas"; tabs.setAttribute("role", "tablist"); tabs.setAttribute("aria-label", t("ficha_navegacion"));
  const avisoRPT = nodo(d, "div"); avisoRPT.dataset.personalFichaRptAviso = ""; avisoRPT.setAttribute("aria-live", "polite");
  const pintarAvisoRPT = () => {
    const conservarFoco = d.activeElement === controlRPT;
    avisoRPT.hidden = estadoRPT === "sin_aviso";
    if (avisoRPT.hidden) { avisoRPT.replaceChildren(); controlRPT = null; return; }
    const clave = { incidencia: "ficha_rpt_incidencia", cargando: "ficha_rpt_cargando", disponible: "ficha_rpt_disponible",
      ausente: "ficha_rpt_ausente", denegado: "ficha_rpt_denegado" }[estadoRPT];
    const estado = mensaje(d, t(clave), estadoRPT === "incidencia" ? "alert" : "status");
    estado.setAttribute("tabindex", "-1"); avisoRPT.replaceChildren(estado); controlRPT = estado;
    if (estadoRPT === "incidencia" || estadoRPT === "disponible") {
      const boton = nodo(d, "button", t(estadoRPT === "incidencia" ? "ficha_rpt_reintentar" : "ficha_rpt_abrir"));
      boton.type = "button"; boton.className = "boton-secundario";
      boton.dataset[estadoRPT === "incidencia" ? "personalFichaRptReintentar" : "personalFichaRptAbrir"] = "";
      boton.addEventListener("click", estadoRPT === "incidencia" ? reintentarRPTVisible : () => {
        if (activa && estadoRPT === "disponible" && actual !== "catalogos") pintar("catalogos");
      });
      avisoRPT.append(boton); controlRPT = boton;
    }
    if (conservarFoco) controlRPT?.focus?.({ preventScroll: true });
  };
  const reintentarRPTVisible = async () => {
    if (!activa || estadoRPT !== "incidencia" || vueloRPT) return;
    estadoRPT = "cargando"; pintarAvisoRPT();
    const intento = Promise.resolve().then(() => {
      const respuesta = reintentarRPT();
      if (!respuesta || typeof respuesta.then !== "function") throw new TypeError("reintento RPT no disponible");
      return respuesta;
    }); vueloRPT = intento;
    try {
      const resultado = await intento;
      if (!RESPUESTAS_RPT.has(resultado)) throw new TypeError("respuesta RPT no válida");
      if (!activa || vueloRPT !== intento) return;
      estadoRPT = resultado;
    } catch {
      if (!activa || vueloRPT !== intento) return;
      estadoRPT = "incidencia";
    } finally {
      if (vueloRPT === intento) vueloRPT = null;
      if (activa) pintarAvisoRPT();
    }
  };
  const principal = nodo(d, "div"); principal.className = "personal-ficha-principal"; principal.id = "personal-ficha-panel";
  principal.setAttribute("role", "tabpanel"); principal.setAttribute("tabindex", "0");
  const pintar = (clave, enfocarAccion = false) => {
    if (!activa) return;
    limpiarHistoria?.(); limpiarHistoria = undefined;
    if (actual === "catalogos" || actual === "contacto") limpiar();
    if (vuelo && Object.hasOwn(estados, actual) && estados[actual] === "cargando") estados[actual] = "sin_consulta";
    serviciosDescargables = undefined;
    secuencia += 1; vuelo?.abort(); vueloExportacion?.abort(); vueloExportacion = undefined; vuelo = undefined; actual = clave;
    ayudaFicha.mostrar(clave === "ficha" ? "ficha_accesos_ayuda" : clave === "contacto" ? "ficha_contacto_ayuda" : BLOQUES[clave]?.ayuda);
    for (const [valor] of pestanas) {
      const tab = tabs.querySelector?.(`[data-personal-ficha-tab="${valor}"]`);
      tab?.setAttribute("aria-selected", String(valor === clave)); tab?.setAttribute("tabindex", valor === clave ? "0" : "-1");
    }
    principal.setAttribute("aria-labelledby", `personal-ficha-tab-${clave}`);
    if (clave === "ficha") { principal.replaceChildren(...portada(d, t, navegarModulo, destinosDisponibles, estados, visibles, ocultarSinFuente, abrirCorreos)); return; }
    if (clave === "catalogos" || clave === "contacto") {
      const contacto = clave === "contacto", montarComplemento = contacto ? montarContacto : montarCatalogos;
      const hueco = nodo(d, "div"); hueco.dataset[contacto ? "personalFichaContacto" : "personalFichaCatalogos"] = "";
      if (contacto) hueco.className = "personal-ficha-panel-ancho";
      principal.replaceChildren(contacto ? hueco : panel(d, t("ficha_catalogos_titulo"), ocultarSinFuente ? [hueco] : [nodo(d, "p", t("ficha_catalogos_completos")), hueco], "personal-ficha-panel-ancho"));
      if (!montarComplemento) { hueco.append(mensaje(d, t("ficha_catalogos_pendiente"))); return; }
      const turno = secuencia;
      Promise.resolve().then(() => {
        if (!activa || actual !== clave || turno !== secuencia) return;
        return montarComplemento({ raiz: hueco, anunciar, ...(contacto ? { abrirCorreos, alCaducarSesion: caducarSesion } : {}), registrarDesmontar: (fn) => {
        if (typeof fn !== "function") throw new TypeError("limpieza de catálogos de Personal no válida");
        if (activa && actual === clave && turno === secuencia) limpiarCatalogos = fn; else fn();
      } }); }).then((montaje) => {
        if (!montaje?.desmontar) return;
        if (!activa || actual !== clave || turno !== secuencia) { montaje.desmontar(); return; }
        const previo = limpiarCatalogos; limpiarCatalogos = previo === montaje.desmontar ? previo : () => { montaje.desmontar(); previo?.(); };
      }).catch(() => { if (activa && actual === clave && turno === secuencia) hueco.replaceChildren(mensaje(d, t("ficha_catalogos_error"), "alert")); });
      return;
    }
    const consultar = Object.hasOwn(fuentes, clave) ? fuentes[clave]?.consultarPropios : undefined;
    if (typeof consultar !== "function") { pintarBloque(d, principal, t, clave, { estado: "no_configurado" }); return; }
    const actualizar = typeof fuentes[clave].actualizar === "function" ? () => {
      fuentes[clave].actualizar();
      const grupo = fuentes[clave].grupoActualizacion;
      if (grupo) for (const otra of visibles) {
        if (fuentes[otra]?.grupoActualizacion === grupo) estados[otra] = "sin_consulta";
      }
      pintar(clave, true);
    } : undefined;
    const corte = clave === "servicios" && fuentes[clave]?.seleccionarFecha === true && actualizar ? {
      referencia: () => referenciaServicios || fuentes[clave].fechaReferencia || "",
      consultar: (referencia) => { referenciaServicios = referencia; actualizar(); },
    } : undefined;
    const enfocar = () => {
      if (!enfocarAccion || d.activeElement !== principal || (typeof d.hasFocus === "function" && !d.hasFocus())) return;
      (principal.querySelector?.(corte ? "[data-personal-ficha-consultar-corte]" : `[data-personal-ficha-actualizar="${clave}"]`) || principal).focus?.();
    };
    const pintarResultado = (resultado) => {
      const previo = corte ? principal.querySelector?.("[data-personal-ficha-fecha]") : undefined;
      const borrador = previo?.value; const teniaFoco = previo && d.activeElement === previo;
      const exportar = Object.hasOwn(fuentes[clave], "exportarPropios") ? fuentes[clave].exportarPropios : undefined;
      const descargar = clave === "servicios" && resultado.exportacion_servicios_disponible === true && typeof exportar === "function" && resultado.recibo_ref ? async () => {
        if (!activa || actual !== "servicios" || turno !== secuencia || serviciosDescargables !== resultado || vueloExportacion || principal.querySelector("[data-personal-servicios-descargar]")?.disabled) return;
        const boton = principal.querySelector("[data-personal-servicios-descargar]");
        const estado = principal.querySelector("[data-personal-servicios-exportacion-estado]");
        const controladorExportacion = new AbortController(); vueloExportacion = controladorExportacion;
        const conservaFoco = d.activeElement === boton; let requiereActualizar = false;
        estado.setAttribute("role", "status");
        boton.disabled = true; boton.setAttribute("aria-disabled", "true"); boton.setAttribute("aria-busy", "true");
        estado.textContent = traducirExportacionServicios("preparando");
        const vigente = () => activa && actual === "servicios" && turno === secuencia && serviciosDescargables === resultado && !controladorExportacion.signal.aborted;
        try {
          const archivo = await exportar({ reciboRef: resultado.recibo_ref, corte: resultado.corte, signal: controladorExportacion.signal });
          if (!vigente()) return;
          descargarResumenServicios(d, archivo);
          estado.textContent = traducirExportacionServicios("preparada");
          anunciar(estado.textContent, "status");
        } catch (causa) {
          if (!vigente()) return;
          if (causa?.estado === 401) {
            serviciosDescargables = undefined;
            pintar("servicios", true);
            anunciar(traducirExportacionServicios("sesion_caducada"), "error");
            return;
          }
          const codigo = ["denegado", "sin_consulta", "no_disponible", "respuesta_no_valida"].includes(causa?.codigo) ? causa.codigo : "error";
          requiereActualizar = codigo === "denegado" || codigo === "sin_consulta";
          estado.textContent = traducirExportacionServicios(codigo); estado.setAttribute("role", "alert");
          anunciar(estado.textContent, "error");
        } finally {
          if (vueloExportacion === controladorExportacion) vueloExportacion = undefined;
          if (vigente()) {
            boton.disabled = requiereActualizar; boton.setAttribute("aria-disabled", String(requiereActualizar)); boton.setAttribute("aria-busy", "false");
            if (conservaFoco && (d.activeElement === boton || d.activeElement === d.body) && (typeof d.hasFocus !== "function" || d.hasFocus())) {
              const destino = requiereActualizar ? principal.querySelector('[data-personal-ficha-actualizar="servicios"]') : boton;
              destino?.focus?.();
            }
          }
        }
      } : undefined;
      const historiaDisponible = clave === "servicios" ? resultado.historia_servicios_disponible === true : clave === "relaciones" && resultado.historia_relaciones_disponible === true;
      const abrirHistoria = historiaDisponible && typeof fuentes[clave]?.clienteHistoria?.consultar === "function" ? () => {
        if (!activa || turno!==secuencia || limpiarHistoria) return;
        const hueco=nodo(d,"div");hueco.className="personal-ficha-panel-ancho personal-ficha-tabla-conjunto";hueco.dataset.personalHistoriaHueco="";principal.append(hueco);
        (clave === "servicios" ? montarVistaHistoriaServiciosPropia : montarVistaHistoriaRelacionesPropia)({raiz:hueco,cliente:fuentes[clave].clienteHistoria,anunciar,alCaducarSesion:()=>{serviciosDescargables=undefined;pintar(clave,true);anunciar(traducirExportacionServicios("sesion_caducada"),"error");},registrarDesmontar:(fn)=>{limpiarHistoria=()=>{fn();hueco.remove?.();};}});
        hueco.querySelector?.('[data-personal-historia-fecha="desde"]')?.focus?.();
        const boton=principal.querySelector?.('[data-personal-historia-abrir]');if(boton)boton.disabled=true;
      } : undefined;
      pintarBloque(d, principal, t, clave, resultado, actualizar, corte, descargar, abrirHistoria);
      const nuevo = corte ? principal.querySelector?.("[data-personal-ficha-fecha]") : undefined;
      if (nuevo && teniaFoco) { nuevo.value = borrador; nuevo.focus?.(); }
      enfocar();
    };
    const turno = secuencia; const controlador = new AbortController(); vuelo = controlador;
    estados[clave] = "cargando";
    pintarBloque(d, principal, t, clave, { estado: "cargando" }, undefined, corte);
    if (enfocarAccion) principal.focus?.();
    Promise.resolve().then(() => consultar({ signal: controlador.signal, ...(corte ? { fechaReferencia: referenciaServicios } : {}) })).then((resultado) => {
      if (!activa || actual !== clave || turno !== secuencia || controlador.signal.aborted) return;
      const validado = validarResultado(resultado, clave); estados[clave] = validado.estado;
      serviciosDescargables = clave === "servicios" && ["disponible", "vacio"].includes(validado.estado) ? validado : undefined;
      pintarResultado(validado);
      if (validado.estado === "error") anunciar(t("ficha_error"), "error");
    }).catch(() => {
      if (!activa || actual !== clave || turno !== secuencia || controlador.signal.aborted) return;
      serviciosDescargables = undefined;
      estados[clave] = "error";
      pintarResultado({ estado: "error" }); anunciar(t("ficha_error"), "error");
    }).finally(() => { if (vuelo === controlador) vuelo = undefined; });
  };
  pestanas.forEach(([clave, texto], indice) => {
    const tab = nodo(d, "button", t(texto)); tab.type = "button"; tab.id = `personal-ficha-tab-${clave}`;
    tab.dataset.personalFichaTab = clave; tab.setAttribute("role", "tab"); tab.setAttribute("aria-controls", principal.id);
    tab.addEventListener("click", () => pintar(clave));
    tab.addEventListener("keydown", (evento) => {
      const salto = { ArrowRight: 1, ArrowLeft: -1, Home: -indice, End: pestanas.length - 1 - indice }[evento.key];
      if (salto === undefined) return; evento.preventDefault();
      const destino = (indice + salto + pestanas.length) % pestanas.length;
      pintar(pestanas[destino][0]); tabs.querySelector?.(`[data-personal-ficha-tab="${pestanas[destino][0]}"]`)?.focus?.();
    }); tabs.append(tab);
  });
  contenedor.append(cabecera, tabs, avisoRPT, principal); pintarAvisoRPT(); pintar(actual); return Object.freeze({ desmontar });
}
