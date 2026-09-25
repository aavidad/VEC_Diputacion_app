import { crearTraductorPersonal } from "./i18n.js?v=20260925-b2-registro-v3";
import { calcularHuellaPublicacionCatalogoB2 } from "./registro-b2-catalogos-cliente.js?v=20260925-b2-registro-v3";

const TIPOS = Object.freeze([["regimen", "registro_b2_catalogos_regimen", "regimenes"], ["modalidad", "registro_b2_catalogos_modalidad", "modalidades"], ["situacion", "registro_b2_catalogos_situacion", "situaciones"], ["clase_servicio", "registro_b2_catalogos_clase_servicio", "clasesServicio"]]);
const REF = /^[a-z][a-z0-9_:-]{2,159}$/u;
function nodo(d, etiqueta, texto) { const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; return n; }
function aviso(d, texto, error = false) { const p = nodo(d, "p", texto); p.className = "personal-registro-b2-estado"; p.setAttribute("role", error ? "alert" : "status"); return p; }
function fecha(valor) { const d = new Date(`${valor}T12:00:00Z`); return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(valor) && Number.isFinite(d.getTime()) && d.toISOString().slice(0, 10) === valor; }
function fechaTexto(valor, t) { return fecha(valor) ? new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(`${valor}T12:00:00Z`)) : t("registro_b2_actual"); }
function fechaHora(valor) { return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor)); }

/** Carga completa de las cuatro opciones publicadas; nunca acepta opciones del shell. */
export async function cargarOpcionesPublicadasCatalogoB2(cliente, { signal } = {}) {
  if (typeof cliente?.listar !== "function") throw new TypeError("catálogo de Personal no disponible");
  const consultas = TIPOS.map(async ([tipo, , grupo]) => {
    let cursorRef = "", cursorVersion = 0, organismoRef = ""; const entradas = []; const cursores = new Set();
    for (let pagina = 0; pagina < 20; pagina++) {
      const resultado = await cliente.listar({ tipo, estado: "publicada", cursorRef, cursorVersion, limite: 100, signal });
      if (signal?.aborted || !REF.test(resultado?.organismoRef) || (organismoRef && organismoRef !== resultado.organismoRef) || !Array.isArray(resultado.entradas) ||
          (resultado.entradas.length === 0 && resultado.cursorSiguiente) || resultado.entradas.some((e) => e.estado !== "publicada" || e.tipo !== tipo || e.organismo_ref !== resultado.organismoRef)) throw new TypeError("opciones de catálogo incompatibles");
      organismoRef = resultado.organismoRef; entradas.push(...resultado.entradas);
      if (!resultado.cursorSiguiente) return { grupo, organismoRef, entradas };
      const clave = `${resultado.cursorSiguiente.ref}:${resultado.cursorSiguiente.version}`;
      if (cursores.has(clave)) throw new TypeError("cursor de catálogo repetido");
      cursores.add(clave); cursorRef = resultado.cursorSiguiente.ref; cursorVersion = resultado.cursorSiguiente.version;
    }
    throw new TypeError("catálogo excede el límite de consulta");
  });
  const grupos = await Promise.all(consultas);
  if (!grupos.every((g) => g.organismoRef === grupos[0].organismoRef)) throw new TypeError("catálogos de organismos distintos");
  return Object.freeze({ organismoRef: grupos[0].organismoRef, ...Object.fromEntries(grupos.map(({ grupo, entradas }) => [grupo, Object.freeze(entradas.map((e) => Object.freeze({ ref: e.ref, version: e.version, denominacion: e.denominacion, estado: e.estado })))])) });
}

/** Consulta y gobierno RRHH de vocabularios versionados. */
export function montarCatalogosRegistroB2({ raiz, cliente, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.listar !== "function" || typeof cliente?.cambiar !== "function" || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("catálogos de Registro de Personal no disponibles");
  const d = raiz.ownerDocument; const t = crearTraductorPersonal(); const s = nodo(d, "section"); s.className = "panel personal-registro-b2-catalogos"; s.dataset.personalRegistroB2Catalogos = ""; raiz.append(s);
  let activo = true, tipo = "regimen", filtro = "publicada", cursorRef = "", cursorVersion = 0, previos = [], paginaActual, pendiente, formulario, estado = "cargando", errorTexto = "", ultimoRecibo, vuelo, secuencia = 0;
  const desmontar = () => { if (!activo) return; activo = false; secuencia++; vuelo?.abort(); s.remove?.(); };
  registrarDesmontar?.(desmontar);
  const limpiarVuelo = () => { secuencia++; vuelo?.abort(); vuelo = undefined; };
  const opcion = (valor, etiqueta) => { const o = nodo(d, "option", t(etiqueta)); o.value = valor; return o; };
  const etiquetaCampo = (clave, texto, control) => { const label = nodo(d, "label", t(texto)); control.dataset.registroB2CatalogoCampo = clave; label.append(control); return label; };

  function pintar() {
    if (!activo) return;
    s.replaceChildren();
    const cab = nodo(d, "header"); cab.className = "cabecera-panel"; cab.append(nodo(d, "h3", t("registro_b2_catalogos"))); s.append(cab);
    const filtros = nodo(d, "div"); filtros.className = "personal-registro-b2-toolbar";
    const tipoControl = nodo(d, "select"); for (const [valor, texto] of TIPOS) tipoControl.append(opcion(valor, texto)); tipoControl.value = tipo;
    tipoControl.addEventListener("change", () => { tipo = tipoControl.value; cursorRef = ""; cursorVersion = 0; previos = []; formulario = undefined; consultar(); });
    const estadoControl = nodo(d, "select"); for (const [valor, texto] of [["", "registro_b2_catalogos_todos"], ["publicada", "registro_b2_catalogos_publicada"], ["retirada", "registro_b2_catalogos_retirada"]]) estadoControl.append(opcion(valor, texto)); estadoControl.value = filtro;
    estadoControl.addEventListener("change", () => { filtro = estadoControl.value; cursorRef = ""; cursorVersion = 0; previos = []; formulario = undefined; consultar(); });
    filtros.append(etiquetaCampo("tipo", "registro_b2_catalogos_tipo", tipoControl), etiquetaCampo("estado", "registro_b2_catalogos_estado", estadoControl)); s.append(filtros);
    const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel personal-registro-b2-catalogos-cuerpo"; s.append(cuerpo);
    if (ultimoRecibo) cuerpo.append(aviso(d, t(ultimoRecibo.accesoActual.estado_replay === "replay" ? "registro_b2_catalogos_replay" : "registro_b2_catalogos_guardado") + ". " + t("registro_b2_catalogos_recibo", { fecha: fechaHora(ultimoRecibo.recibo.registrado_en) })));
    if (estado === "cargando") { cuerpo.append(aviso(d, t("registro_b2_catalogos_cargando"))); return; }
    if (estado === "error") { cuerpo.append(aviso(d, errorTexto, true)); const retry = nodo(d, "button", t("registro_b2_reintentar")); retry.type = "button"; retry.addEventListener("click", consultar); cuerpo.append(retry); return; }
    if (estado === "conflicto") { cuerpo.append(aviso(d, t("registro_b2_catalogos_conflicto"), true)); const actualizar = nodo(d, "button", t("registro_b2_reintentar")); actualizar.type = "button"; actualizar.addEventListener("click", () => { formulario = undefined; consultar(); }); cuerpo.append(actualizar); return; }
    if (estado === "incierto") { cuerpo.append(aviso(d, t("registro_b2_catalogos_incierto"), true)); const exacto = nodo(d, "button", t("registro_b2_reintentar_exacto")); exacto.type = "button"; exacto.addEventListener("click", confirmar); cuerpo.append(exacto); return; }
    if (estado === "enviando") { cuerpo.append(aviso(d, t("registro_b2_guardando"))); return; }
    if (estado === "revision") { pintarRevision(cuerpo); return; }
    if (formulario) { pintarFormulario(cuerpo); return; }
    if (!paginaActual?.entradas?.length) cuerpo.append(aviso(d, t("registro_b2_catalogos_vacio")));
    else cuerpo.append(tabla(cuerpo));
    const acciones = nodo(d, "div"); acciones.className = "personal-registro-b2-paginacion";
    const anterior = nodo(d, "button", t("registro_b2_catalogos_anterior")); anterior.type = "button"; anterior.disabled = previos.length === 0; anterior.addEventListener("click", () => { const c = previos.pop(); cursorRef = c.ref; cursorVersion = c.version; consultar(); });
    const siguiente = nodo(d, "button", t("registro_b2_catalogos_siguiente")); siguiente.type = "button"; siguiente.disabled = !paginaActual?.cursorSiguiente; siguiente.addEventListener("click", () => { previos.push({ ref: cursorRef, version: cursorVersion }); cursorRef = paginaActual.cursorSiguiente.ref; cursorVersion = paginaActual.cursorSiguiente.version; consultar(); });
    const publicar = nodo(d, "button", t("registro_b2_catalogos_publicar")); publicar.type = "button"; publicar.disabled = !paginaActual?.organismoRef;
    publicar.addEventListener("click", () => { if (!paginaActual?.organismoRef) return; formulario = { operacion: "publicar" }; pintar(); });
    acciones.append(anterior, siguiente, publicar); cuerpo.append(acciones);
  }
  function tabla() {
    const region = nodo(d, "div"); region.className = "tabla-contenedor"; region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("registro_b2_catalogos_tabla"));
    const tab = nodo(d, "table"); tab.className = "tabla-datos"; tab.append(nodo(d, "caption", t("registro_b2_catalogos_tabla")));
    const th = nodo(d, "thead"); const filaCab = nodo(d, "tr"); for (const texto of ["registro_b2_catalogos_denominacion", "registro_b2_catalogos_version", "registro_b2_catalogos_estado", "registro_b2_catalogos_vigencia", "registro_b2_catalogos_revision", "registro_b2_catalogos_acciones"]) { const celda = nodo(d, "th", t(texto)); celda.setAttribute("scope", "col"); filaCab.append(celda); } th.append(filaCab); tab.append(th);
    const body = nodo(d, "tbody");
    for (const e of paginaActual.entradas) {
      const tr = nodo(d, "tr"); const nombre = nodo(d, "td"); nombre.append(nodo(d, "span", e.denominacion)); const sub = nodo(d, "small", e.ref); sub.className = "personal-registro-b2-secundario"; nombre.append(sub);
      tr.append(nombre, nodo(d, "td", new Intl.NumberFormat("es-ES").format(e.version)), nodo(d, "td", t(e.estado === "publicada" ? "registro_b2_catalogos_publicada" : "registro_b2_catalogos_retirada")), nodo(d, "td", `${fechaTexto(e.vigente_desde, t)} – ${fechaTexto(e.vigente_hasta, t)}`), nodo(d, "td", new Intl.NumberFormat("es-ES").format(e.revision)));
      const celda = nodo(d, "td");
      if (e.estado === "publicada") { const retirar = nodo(d, "button", t("registro_b2_catalogos_retirar")); retirar.type = "button"; retirar.addEventListener("click", () => { formulario = { operacion: "retirar", entrada: e }; pintar(); }); celda.append(retirar); }
      tr.append(celda);
      body.append(tr);
    }
    tab.append(body); region.append(tab); return region;
  }
  function pintarFormulario(cuerpo) {
    const form = nodo(d, "form"); form.className = "personal-registro-b2-catalogos-formulario"; const campos = new Map();
    const agregar = (clave, etiqueta, tipoCampo, valor = "", requerido = true) => { const control = nodo(d, "input"); control.type = tipoCampo; control.value = valor; control.required = requerido; if (tipoCampo === "number") { control.min = "1"; control.step = "1"; } campos.set(clave, control); form.append(etiquetaCampo(clave, etiqueta, control)); };
    if (formulario.operacion === "publicar") { agregar("ref", "registro_b2_catalogos_ref", "text"); agregar("version", "registro_b2_catalogos_version", "number", "1"); agregar("denominacion", "registro_b2_catalogos_denominacion", "text"); agregar("vigente_desde", "registro_b2_catalogos_desde", "date"); agregar("vigente_hasta", "registro_b2_catalogos_hasta", "date", "", false); }
    else { const e = formulario.entrada; form.append(aviso(d, `${e.denominacion} · ${e.ref} · ${t("registro_b2_catalogos_version")} ${new Intl.NumberFormat("es-ES").format(e.version)}`)); }
    const revisar = nodo(d, "button", t("registro_b2_catalogos_revisar")); revisar.type = "submit";
    const cancelar = nodo(d, "button", t("registro_b2_catalogos_cancelar")); cancelar.type = "button"; cancelar.addEventListener("click", () => { formulario = undefined; pintar(); });
    form.append(revisar, cancelar); form.addEventListener("submit", async (evento) => {
      evento.preventDefault(); if (!activo || pendiente || formulario.revisando) return; formulario.revisando = true; const formularioActual = formulario;
      const valor = (clave) => campos.get(clave)?.value.trim() || "";
      try {
        if (!paginaActual?.organismoRef || !globalThis.crypto?.randomUUID) throw new TypeError();
        let body;
        if (formulario.operacion === "publicar") {
          const version = Number(valor("version")); const desde = valor("vigente_desde"), hasta = valor("vigente_hasta");
          const huella = await calcularHuellaPublicacionCatalogoB2({ organismoRef: paginaActual.organismoRef, tipo, ref: valor("ref"), version, revision: 1, denominacion: valor("denominacion"), vigenteDesde: desde, vigenteHasta: hasta });
          if (!activo || formulario !== formularioActual) return;
          body = { operacion: "publicar", tipo, ref: valor("ref"), version, revision: 1, denominacion: valor("denominacion"), huella_sha256: huella, vigente_desde: desde, vigente_hasta: hasta };
        } else { const e = formulario.entrada; body = { operacion: "retirar", tipo, ref: e.ref, version: e.version, revision: e.revision + 1, denominacion: e.denominacion, huella_sha256: e.huella_sha256, vigente_desde: e.vigente_desde, vigente_hasta: e.vigente_hasta }; }
        pendiente = Object.freeze({ cuerpo: Object.freeze(body), clave: globalThis.crypto.randomUUID() }); estado = "revision"; pintar();
      } catch { cuerpo.append(aviso(d, t("registro_b2_catalogos_invalido"), true)); }
      finally { if (formulario) formulario.revisando = false; }
    }); cuerpo.append(form);
  }
  function pintarRevision(cuerpo) {
    cuerpo.append(aviso(d, t(pendiente.cuerpo.operacion === "publicar" ? "registro_b2_catalogos_publicar" : "registro_b2_catalogos_retirar")));
    const datos = nodo(d, "dl"); datos.className = "personal-registro-b2-catalogos-revision";
    for (const [etiqueta, valor] of [["registro_b2_catalogos_tipo", t(TIPOS.find(([v]) => v === tipo)[1])], ["registro_b2_catalogos_ref", pendiente.cuerpo.ref], ["registro_b2_catalogos_version", new Intl.NumberFormat("es-ES").format(pendiente.cuerpo.version)], ["registro_b2_catalogos_denominacion", pendiente.cuerpo.denominacion], ["registro_b2_catalogos_desde", fechaTexto(pendiente.cuerpo.vigente_desde, t)]]) { datos.append(nodo(d, "dt", t(etiqueta)), nodo(d, "dd", valor)); }
    cuerpo.append(datos);
    const acciones = nodo(d, "div"); acciones.className = "personal-registro-b2-paginacion";
    const cancelar = nodo(d, "button", t("registro_b2_catalogos_cancelar")); cancelar.type = "button"; cancelar.addEventListener("click", () => { pendiente = undefined; estado = "lista"; pintar(); });
    const confirmarBoton = nodo(d, "button", t("registro_b2_catalogos_confirmar")); confirmarBoton.type = "button"; confirmarBoton.addEventListener("click", confirmar); acciones.append(cancelar, confirmarBoton); cuerpo.append(acciones);
  }
  async function confirmar() {
    if (!activo || !pendiente || (estado !== "revision" && estado !== "incierto")) return;
    const exacto = pendiente; estado = "enviando"; pintar();
    try {
      const resultado = await cliente.cambiar(exacto.cuerpo, { claveIdempotencia: exacto.clave });
      if (!activo || pendiente !== exacto) return;
      pendiente = undefined; formulario = undefined; ultimoRecibo = resultado; cursorRef = ""; cursorVersion = 0; previos = []; consultar();
    } catch (error) {
      if (!activo || pendiente !== exacto) return;
      if (error?.codigo === "resultado_incierto") estado = "incierto";
      else if (error?.estado === 409) { pendiente = undefined; formulario = undefined; estado = "conflicto"; }
      else { pendiente = undefined; estado = "error"; errorTexto = error?.estado === 403 ? t("registro_b2_acto_denegado") : t("registro_b2_acto_error"); }
      anunciar(estado === "conflicto" ? t("registro_b2_catalogos_conflicto") : estado === "incierto" ? t("registro_b2_catalogos_incierto") : errorTexto, "error"); pintar();
    }
  }
  async function consultar() {
    if (!activo) return; limpiarVuelo(); const actual = new AbortController(); vuelo = actual; const turno = secuencia; estado = "cargando"; pintar();
    try {
      const pagina = await cliente.listar({ tipo, estado: filtro, cursorRef, cursorVersion, limite: 50, signal: actual.signal });
      if (!activo || vuelo !== actual || actual.signal.aborted || turno !== secuencia) return;
      paginaActual = pagina; estado = "lista"; pintar();
    } catch (error) {
      if (!activo || vuelo !== actual || actual.signal.aborted || turno !== secuencia) return;
      estado = "error"; errorTexto = error?.estado === 403 ? t("registro_b2_catalogos_denegado") : t("registro_b2_catalogos_error"); anunciar(errorTexto, "error"); pintar();
    } finally { if (vuelo === actual) vuelo = undefined; }
  }
  consultar();
  return Object.freeze({ desmontar });
}
