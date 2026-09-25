import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js?v=20260925-d5d6-v1";
import { MENSAJES_RECTIFICACION_ADMIN_ES } from "./i18n-rectificacion-admin.js?v=20260925-tanda-v1";

const nodo = (d, etiqueta, valor = "") => { const n = d.createElement(etiqueta); n.textContent = valor; return n; };
const refSolicitud = (valor) => typeof valor === "string" && /^srd_[0-9a-f]{32}$/u.test(valor);
const fechaCivil = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(valor) && Number.isFinite(Date.parse(`${valor}T00:00:00Z`));
const motivoValido = (valor) => typeof valor === "string" && valor.length >= 3 && new TextEncoder().encode(valor).byteLength <= 500 && !/[\x00-\x1f\x7f]/u.test(valor) && !/per_[A-Za-z0-9_-]{22,128}/u.test(valor);
const claveValida = (valor) => typeof valor === "string" && /^[A-Za-z0-9:_-]{16,128}$/u.test(valor);
const refPersona = (valor) => typeof valor === "string" && /^per_[A-Za-z0-9_-]{22,128}$/u.test(valor);
const textoCatalogo = (valor, maximo = 160) => typeof valor === "string" && valor.length > 0 && valor.length <= maximo &&
  valor.trim() === valor && !/[\x00-\x1f\x7f]/u.test(valor);
const instante = (valor) => { const d = new Date(valor); return Number.isFinite(d.getTime()) ? new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(d) : ""; };
const civil = (valor) => fechaCivil(valor) ? new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T00:00:00Z`)) : "";
function traductor(traducir) {
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_RECTIFICACION_ADMIN_ES, clave)) return traducir(clave, variables);
    try { const r = traducir(clave, variables); if (typeof r === "string" && r !== clave) return r; } catch { /* Catálogo común todavía sin esta extensión. */ }
    return MENSAJES_RECTIFICACION_ADMIN_ES[clave].replace(/\{([a-z_]+)\}/gu, (_m, k) => String(variables[k] ?? ""));
  };
}
function errorTexto(error, t, ficha = false) {
  if (error?.codigo === "acceso_denegado") return t("ra_denegado");
  if (error?.codigo === "autenticacion_requerida") return t("ra_autenticacion");
  if (error?.codigo === "conflicto") return t("ra_conflicto");
  return t(ficha ? "ra_error_ficha" : "ra_error_lista");
}
function campo(d, t, clave, valor, editable = false) {
  const label = nodo(d, "label", t(clave));
  const control = nodo(d, editable ? "input" : "output");
  if (editable) { control.name = clave; control.value = String(valor ?? ""); }
  else control.textContent = String(valor ?? "");
  label.append(control); return label;
}
function opcionesValidas(lista, clave, comprobarRef) {
  return Array.isArray(lista) && lista.length > 0 && lista.length <= 100 &&
    lista.every((opcion) => opcion && typeof opcion === "object" && !Array.isArray(opcion) &&
      Object.keys(opcion).length === 2 && Object.hasOwn(opcion, clave) && Object.hasOwn(opcion, "etiqueta") &&
      comprobarRef(opcion[clave]) && textoCatalogo(opcion.etiqueta)) &&
    new Set(lista.map((opcion) => opcion[clave])).size === lista.length;
}
function catalogoValido(catalogo, solicitud) {
  return catalogo && typeof catalogo === "object" && !Array.isArray(catalogo) &&
    catalogo.solicitud_ref === solicitud.solicitud_ref && catalogo.asignacion_ref === solicitud.asignacion_ref &&
    catalogo.version === solicitud.version_origen &&
    opcionesValidas(catalogo.centros, "ref", (v) => textoCatalogo(v)) &&
    opcionesValidas(catalogo.administrativos, "persona_ref", refPersona) &&
    opcionesValidas(catalogo.responsables, "persona_ref", refPersona);
}
function copiarCatalogo(c) {
  const copiar = (lista, clave) => Object.freeze(lista.map((opcion) => Object.freeze({ [clave]: opcion[clave], etiqueta: opcion.etiqueta })));
  return Object.freeze({ solicitud_ref: c.solicitud_ref, asignacion_ref: c.asignacion_ref, version: c.version,
    centros: copiar(c.centros, "ref"), administrativos: copiar(c.administrativos, "persona_ref"),
    responsables: copiar(c.responsables, "persona_ref") });
}
function selectCatalogo(d, t, clave, lista, claveRef, actual) {
  const label = nodo(d, "label", t(clave)), select = nodo(d, "select"); select.name = clave; select.required = true;
  const placeholder = nodo(d, "option", t(lista ? "ra_seleccionar_opcion" : "ra_opciones_no_disponibles")); placeholder.value = ""; select.append(placeholder);
  for (const opcion of lista || []) { const option = nodo(d, "option", opcion.etiqueta); option.value = opcion[claveRef]; select.append(option); }
  select.value = lista?.some((opcion) => opcion[claveRef] === actual) ? actual : "";
  select.disabled = !lista; label.append(select); return label;
}

/** La enumeración y ficha proceden de una lectura competente autorizada por Personal. */
export function montarVistaRectificacionAdminDietas(contenedor, {
  cliente,
  clienteCatalogoCompetente,
  traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
  anunciar = () => {},
  registrarDesmontar,
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  confirmarAccion = (mensaje) => globalThis.confirm?.(mensaje) === true,
} = {}) {
  if (!contenedor?.ownerDocument?.createElement || typeof cliente?.listar !== "function" || typeof cliente?.decidir !== "function" ||
      (clienteCatalogoCompetente !== undefined && typeof clienteCatalogoCompetente?.consultar !== "function") ||
      typeof traducir !== "function" || typeof anunciar !== "function" ||
      typeof generarClaveIdempotencia !== "function" || typeof confirmarAccion !== "function")
    throw new TypeError("vista administrativa de rectificación no disponible");
  const d = contenedor.ownerDocument, t = traductor(traducir);
  const raiz = nodo(d, "section"); raiz.className = "modulo-dietas dietas-rectificacion-admin"; raiz.dataset.dietasRectificacionAdmin = "";
  const bandeja = nodo(d, "section"); bandeja.className = "panel dietas-rectificacion-admin-bandeja";
  const cabB = nodo(d, "div"); cabB.className = "cabecera-panel";
  const titB = nodo(d, "div"); titB.append(nodo(d, "h2", t("ra_titulo")), nodo(d, "p", t("ra_subtitulo")));
  const recargar = nodo(d, "button", t("ra_recargar")); recargar.type = "button"; recargar.className = "boton-secundario"; recargar.dataset.dietasRaRecargar = "";
  const ayuda = nodo(d, "details"); ayuda.className = "dietas-recorridos-ayuda";
  const resumen = nodo(d, "summary", "?"); resumen.setAttribute("aria-label", t("ra_ayuda_etiqueta")); ayuda.append(resumen, nodo(d, "p", t("ra_ayuda")));
  cabB.append(titB, recargar, ayuda);
  const estadoLista = nodo(d, "p"); estadoLista.dataset.dietasRaEstadoLista = ""; estadoLista.setAttribute("role", "status");
  const cuerpoB = nodo(d, "div"); cuerpoB.className = "cuerpo-panel"; cuerpoB.dataset.dietasRaLista = "";
  bandeja.append(cabB, estadoLista, cuerpoB);
  const fichaPanel = nodo(d, "section"); fichaPanel.className = "panel dietas-rectificacion-admin-ficha";
  const cabF = nodo(d, "div"); cabF.className = "cabecera-panel"; cabF.append(nodo(d, "h2", t("ra_ficha_titulo")));
  const estadoFicha = nodo(d, "p"); estadoFicha.dataset.dietasRaEstadoFicha = ""; estadoFicha.setAttribute("role", "status");
  const cuerpoF = nodo(d, "div"); cuerpoF.className = "cuerpo-panel"; cuerpoF.dataset.dietasRaFicha = "";
  fichaPanel.append(cabF, estadoFicha, cuerpoF); raiz.append(bandeja, fichaPanel); contenedor.append(raiz);

  let viva = true, lecturaLista, lecturaCatalogo, escritura, secuenciaLista = 0, secuenciaCatalogo = 0;
  let pagina = { items: [] }, ficha, seleccion, pendiente, recibo, catalogo, estadoCatalogo = "ra_catalogo_no_disponible";
  const avisar = (mensaje) => { if (viva) anunciar(mensaje); };
  const estado = (elemento, mensaje, nivel = "") => { elemento.textContent = mensaje; elemento.dataset.estado = nivel; if (mensaje) avisar(mensaje); };
  function pintarLista() {
    cuerpoB.replaceChildren();
    if (!pagina.items.length) { cuerpoB.append(nodo(d, "p", t("ra_vacio"))); return; }
    const envoltura = nodo(d, "div"); envoltura.className = "tabla-contenedor";
    const tabla = nodo(d, "table"); tabla.className = "tabla-admin"; tabla.setAttribute("aria-label", t("ra_tabla_etiqueta"));
    const th = nodo(d, "tr"); ["ra_cab_solicitud", "ra_cab_fecha", "ra_cab_unidad", "ra_cab_estado", "ra_cab_accion"].forEach((k) => th.append(nodo(d, "th", t(k))));
    const cabeza = nodo(d, "thead"); cabeza.append(th); const cuerpo = nodo(d, "tbody");
    for (const item of pagina.items) {
      if (!refSolicitud(item?.solicitud_ref) || item.estado !== "pendiente") continue;
      const tr = nodo(d, "tr"); tr.dataset.dietasRaSolicitud = item.solicitud_ref;
      const ref = nodo(d, "td"); ref.append(nodo(d, "strong", t("ra_solicitud_fecha", { fecha: instante(item.registrada_en) })),
        nodo(d, "small", item.solicitud_ref));
      const celdaAccion = nodo(d, "td"); const ver = nodo(d, "button", t("ra_ver")); ver.type = "button"; ver.className = "boton-secundario";
      ver.dataset.dietasRaVer = item.solicitud_ref; ver.disabled = Boolean(escritura); celdaAccion.append(ver);
      tr.append(ref, nodo(d, "td", instante(item.registrada_en)), nodo(d, "td", String(item.unidad_ref ?? "")), nodo(d, "td", t("ra_estado_pendiente")), celdaAccion); cuerpo.append(tr);
    }
    tabla.append(cabeza, cuerpo); envoltura.append(tabla); cuerpoB.append(envoltura);
  }
  function pintarFicha() {
    const motivoPrevio = cuerpoF.querySelector?.('[name="motivo_decision"]')?.value || "";
    cuerpoF.replaceChildren();
    if (!ficha) { cuerpoF.append(nodo(d, "p", t("ra_ficha_vacia"))); return; }
    const s = ficha.solicitud, a = ficha.asignacion;
    if (!s || !a || s.solicitud_ref !== seleccion || s.estado !== "pendiente" ||
        !Number.isSafeInteger(s.version_origen) || s.version_origen < 1 ||
        !fechaCivil(s.fecha_referencia)) { ficha = null; estado(estadoFicha, t("ra_error_ficha"), "error"); return; }
    const asignacionCambiada = a.version !== s.version_origen || a.asignacion_ref !== s.asignacion_ref;
    const resumenFicha = nodo(d, "dl"); resumenFicha.className = "dietas-rectificacion-admin-resumen";
    for (const [clave, valor] of [["ra_solicitud", s.solicitud_ref], ["ra_relacion", s.relacion_ref], ["ra_unidad", s.unidad_ref], ["ra_fecha_referencia", civil(s.fecha_referencia)], ["ra_motivo_empleado", s.motivo_revision], ["ra_detalle_empleado", s.detalle_solicitado || "—"], ["ra_campos", (s.campos_a_revisar || []).map((c) => t(`ra_campo_${c}`)).join(", ")]]) {
      const par = nodo(d, "div"); par.append(nodo(d, "dt", t(clave)), nodo(d, "dd", String(valor ?? ""))); resumenFicha.append(par);
    }
    const actual = nodo(d, "section"); actual.append(nodo(d, "h3", t("ra_actual")));
    if (asignacionCambiada) { const aviso = nodo(d, "p", t("ra_asignacion_cambiada")); aviso.setAttribute("role", "status"); actual.append(aviso); }
    const etiqueta = (lista, clave, valor) => lista?.find((opcion) => opcion[clave] === valor)?.etiqueta || t("ra_dato_no_disponible");
    const dl = nodo(d, "dl"); for (const [k, v] of [["ra_centro", etiqueta(catalogo?.centros, "ref", a.centro_ref)],
      ["ra_administrativo", etiqueta(catalogo?.administrativos, "persona_ref", a.administrativo_persona_ref)],
      ["ra_responsable", etiqueta(catalogo?.responsables, "persona_ref", a.responsable_persona_ref)],
      ["ra_grupo", a.grupo_dieta], ["ra_vigencia", civil(a.vigente_desde)]]) {
      const p = nodo(d, "div"); p.append(nodo(d, "dt", t(k)), nodo(d, "dd", String(v ?? ""))); dl.append(p);
    } actual.append(dl);
    const form = nodo(d, "form"); form.dataset.dietasRaForm = ""; form.append(nodo(d, "h3", t("ra_nueva")));
    const avisoCatalogo = nodo(d, "p", t(estadoCatalogo)); avisoCatalogo.dataset.dietasRaEstadoCatalogo = ""; avisoCatalogo.setAttribute("role", "status"); form.append(avisoCatalogo);
    form.append(selectCatalogo(d, t, "ra_centro", catalogo?.centros, "ref", a.centro_ref),
      selectCatalogo(d, t, "ra_administrativo", catalogo?.administrativos, "persona_ref", a.administrativo_persona_ref),
      selectCatalogo(d, t, "ra_responsable", catalogo?.responsables, "persona_ref", a.responsable_persona_ref));
    for (const [k, v] of [["ra_grupo", a.grupo_dieta], ["ra_vigencia", a.vigente_desde]]) {
      const label = campo(d, t, k, v, true); const input = label.querySelector?.("input") || label.children?.[0];
      if (k === "ra_grupo") { input.type = "number"; input.min = "1"; input.max = "3"; }
      else if (k === "ra_vigencia") input.type = "date";
      form.append(label);
    }
    const motivo = nodo(d, "textarea"); motivo.name = "motivo_decision"; motivo.required = true; motivo.maxLength = 500; motivo.value = motivoPrevio;
    const labelMotivo = nodo(d, "label", t("ra_motivo_decision")); labelMotivo.append(motivo); form.append(labelMotivo);
    const ayudaMotivo = nodo(d, "details"); const resumenMotivo = nodo(d, "summary", "?"); resumenMotivo.setAttribute("aria-label", t("ra_motivo_decision")); ayudaMotivo.append(resumenMotivo, nodo(d, "p", t("ra_motivo_ayuda"))); form.append(ayudaMotivo);
    const acciones = nodo(d, "div"); acciones.className = "acciones-vista";
    const confirmar = nodo(d, "button", t("ra_confirmar")); confirmar.type = "button"; confirmar.className = "boton-primario"; confirmar.dataset.dietasRaDecision = "confirmar";
    const rechazar = nodo(d, "button", t("ra_rechazar")); rechazar.type = "button"; rechazar.className = "boton-peligro"; rechazar.dataset.dietasRaDecision = "rechazar";
    confirmar.disabled = Boolean(escritura) || Boolean(pendiente?.incierta) || asignacionCambiada || !catalogo;
    rechazar.disabled = Boolean(escritura) || Boolean(pendiente?.incierta); acciones.append(confirmar, rechazar);
    if (pendiente?.incierta && pendiente.solicitud_ref === seleccion) { const retry = nodo(d, "button", t("ra_reintentar")); retry.type = "button"; retry.dataset.dietasRaReintentar = ""; acciones.append(retry); }
    form.append(acciones); cuerpoF.append(resumenFicha, actual, form);
  }
  async function listar() {
    if (escritura || pendiente?.incierta || !viva) return;
    lecturaLista?.abort(); lecturaLista = new AbortController(); const turno = ++secuenciaLista;
    lecturaCatalogo?.abort(); secuenciaCatalogo++; seleccion = null; ficha = null; pendiente = null; catalogo = null;
    estadoCatalogo = "ra_catalogo_no_disponible";
    estado(estadoLista, t("ra_cargando"), "cargando"); pagina = { items: [] }; cuerpoB.replaceChildren(); pintarFicha();
    try {
      const r = await cliente.listar({ signal: lecturaLista.signal });
      if (!viva || turno !== secuenciaLista) return;
      if (!r || !Array.isArray(r.solicitudes) || r.solicitudes.length > 50 || r.cardinalidad !== r.solicitudes.length ||
          r.solicitudes.some((s) => !refSolicitud(s?.solicitud_ref) || s.estado !== "pendiente" || !s.asignacion_actual) ||
          new Set(r.solicitudes.map((s) => s.solicitud_ref)).size !== r.solicitudes.length) throw new TypeError("lista incompatible");
      pagina = { items: r.solicitudes };
      estado(estadoLista, ""); pintarLista();
    } catch (e) { if (!viva || turno !== secuenciaLista || e?.codigo === "operacion_abortada") return; estado(estadoLista, errorTexto(e, t), e?.codigo === "acceso_denegado" || e?.codigo === "autenticacion_requerida" ? "denegado" : "error"); cuerpoB.replaceChildren(); }
  }
  function consultar(ref) {
    if (!refSolicitud(ref) || escritura || pendiente?.incierta) return;
    const s = pagina.items.find((item) => item.solicitud_ref === ref && item.estado === "pendiente");
    if (!s?.asignacion_actual) return;
    lecturaCatalogo?.abort(); secuenciaCatalogo++; seleccion = ref; pendiente = null; catalogo = null;
    estadoCatalogo = clienteCatalogoCompetente ? "ra_catalogo_cargando" : "ra_catalogo_no_disponible";
    cuerpoF.replaceChildren(); ficha = { solicitud: s, asignacion: s.asignacion_actual };
    estado(estadoFicha, t("ra_ficha_pendiente")); pintarFicha();
    if (clienteCatalogoCompetente && s.asignacion_actual.version === s.version_origen && s.asignacion_actual.asignacion_ref === s.asignacion_ref)
      void cargarCatalogo(ref);
    else if (clienteCatalogoCompetente) { estadoCatalogo = "ra_catalogo_no_disponible"; pintarFicha(); }
  }
  async function cargarCatalogo(ref) {
    lecturaCatalogo = new AbortController(); const turno = ++secuenciaCatalogo;
    try {
      const recibido = await clienteCatalogoCompetente.consultar(ref, { signal: lecturaCatalogo.signal });
      if (!viva || turno !== secuenciaCatalogo || seleccion !== ref || !ficha) return;
      if (!catalogoValido(recibido, ficha.solicitud)) throw new TypeError("catálogo competente incompatible");
      catalogo = copiarCatalogo(recibido); estadoCatalogo = "ra_catalogo_disponible"; pintarFicha();
    } catch (e) {
      if (!viva || turno !== secuenciaCatalogo || e?.codigo === "operacion_abortada") return;
      catalogo = null; estadoCatalogo = e?.codigo === "acceso_denegado" ? "ra_catalogo_denegado" : "ra_catalogo_no_disponible";
      pintarFicha();
    }
  }
  async function decidir(decision, reintento = false) {
    if (!viva || escritura || !ficha || !seleccion || !["confirmar", "rechazar"].includes(decision) ||
        (pendiente?.incierta && !reintento)) return;
    if (decision === "confirmar" && (!catalogo || !catalogoValido(catalogo, ficha.solicitud))) {
      estado(estadoFicha, t("ra_catalogo_no_disponible"), "denegado"); return;
    }
    if (decision === "confirmar" && (ficha.asignacion.version !== ficha.solicitud.version_origen ||
        ficha.asignacion.asignacion_ref !== ficha.solicitud.asignacion_ref)) { estado(estadoFicha, t("ra_asignacion_cambiada"), "conflicto"); return; }
    let entrada;
    if (reintento) { if (!pendiente?.incierta || pendiente.solicitud_ref !== seleccion || pendiente.entrada.decision !== decision) return; entrada = pendiente.entrada; }
    else {
      const s = ficha.solicitud, form = cuerpoF.querySelector?.("[data-dietas-ra-form]");
      const leer = (name) => String(form?.querySelector?.(`[name="${name}"]`)?.value ?? "").trim();
      const motivo = leer("motivo_decision"); if (!motivoValido(motivo)) { estado(estadoFicha, t("ra_invalido"), "error"); return; }
      const clave = generarClaveIdempotencia(); if (!claveValida(clave)) { estado(estadoFicha, t("ra_error_ficha"), "error"); return; }
      entrada = { decision, persona_ref: s.persona_ref, empleado_ref: s.empleado_ref, relacion_ref: s.relacion_ref, unidad_ref: s.unidad_ref,
        asignacion_ref: decision === "confirmar" ? s.asignacion_ref : "", version_esperada: decision === "confirmar" ? s.version_origen : 0,
        fecha_referencia: s.fecha_referencia, clave_idempotencia: clave, motivo_revision: motivo };
      if (decision === "confirmar") {
        const grupo = Number(leer("ra_grupo")); const correccion = { centro_ref: leer("ra_centro"), administrativo_persona_ref: leer("ra_administrativo"),
          responsable_persona_ref: leer("ra_responsable"), grupo_dieta: grupo, vigente_desde: leer("ra_vigencia") };
        if (!catalogo.centros.some((opcion) => opcion.ref === correccion.centro_ref) ||
            !catalogo.administrativos.some((opcion) => opcion.persona_ref === correccion.administrativo_persona_ref) ||
            !catalogo.responsables.some((opcion) => opcion.persona_ref === correccion.responsable_persona_ref) ||
            ![1, 2, 3].includes(grupo) || !fechaCivil(correccion.vigente_desde)) {
          estado(estadoFicha, t("ra_invalido"), "error"); return;
        }
        entrada.correccion = correccion;
      }
      if (!confirmarAccion(t(decision === "confirmar" ? "ra_confirmar_aviso" : "ra_rechazar_aviso"))) return;
      pendiente = { solicitud_ref: seleccion, entrada, incierta: false };
    }
    escritura = new AbortController(); estado(estadoFicha, t("ra_procesando"), "cargando"); pintarFicha();
    try { const r = await cliente.decidir(seleccion, entrada, { signal: escritura.signal });
      if (!viva) return;
      if (r?.solicitud_ref !== seleccion || !/^rrd_[0-9a-f]{32}$/u.test(r?.recibo_ref) ||
          ![decision === "confirmar" ? "confirmada" : "rechazada", "replay_confirmado"].includes(r.estado)) throw new TypeError("recibo incompatible");
      recibo = r; pendiente = null; ficha = null; pagina = { ...pagina, items: pagina.items.filter((item) => item.solicitud_ref !== seleccion) };
      estado(estadoFicha, t(r.estado === "replay_confirmado" ? "ra_recibo_recuperado" : "ra_recibo", { recibo: r.recibo_ref, fecha: instante(r.registrada_en) }), "exito");
      cuerpoF.replaceChildren(); pintarLista();
    } catch (e) { if (!viva || e?.codigo === "operacion_abortada") return;
      if (e?.resultadoIndeterminado) { pendiente = { solicitud_ref: seleccion, entrada, incierta: true }; estado(estadoFicha, t("ra_incierto"), "error"); }
      else { pendiente = null; estado(estadoFicha, errorTexto(e, t, true), e?.codigo === "acceso_denegado" ? "denegado" : "error"); }
    } finally { escritura = null; if (viva && ficha) pintarFicha(); }
  }
  function click(e) {
    const b = e.target?.closest?.("[data-dietas-ra-recargar], [data-dietas-ra-ver], [data-dietas-ra-decision], [data-dietas-ra-reintentar]");
    if (!b) return;
    if (b.dataset.dietasRaRecargar !== undefined) void listar();
    else if (b.dataset.dietasRaVer !== undefined) void consultar(b.dataset.dietasRaVer);
    else if (b.dataset.dietasRaReintentar !== undefined) void decidir(pendiente?.entrada.decision, true);
    else if (b.dataset.dietasRaDecision !== undefined) void decidir(b.dataset.dietasRaDecision);
  }
  function desmontar() { if (!viva) return; viva = false; lecturaLista?.abort(); lecturaCatalogo?.abort(); escritura?.abort(); raiz.removeEventListener("click", click); raiz.remove(); }
  raiz.addEventListener("click", click); registrarDesmontar?.(desmontar); pintarFicha(); void listar();
  return Object.freeze({ desmontar, recargar: () => listar(), get recibo() { return recibo; } });
}
