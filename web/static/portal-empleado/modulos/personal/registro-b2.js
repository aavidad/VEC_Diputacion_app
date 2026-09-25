import { crearTraductorPersonal } from "./i18n.js?v=20260925-b2-registro-v2";
import { ErrorRegistroB2 } from "./registro-b2-cliente.js?v=20260925-b2-registro-v1";
import { accionesRegistroB2Disponibles, montarActosRegistroB2 } from "./registro-b2-actos.js?v=20260925-b2-registro-v2";
import { cargarOpcionesPublicadasCatalogoB2, montarCatalogosRegistroB2 } from "./registro-b2-catalogos.js?v=20260925-b2-registro-v2";

const BLOQUES = Object.freeze([
  ["relaciones", "registro_b2_relaciones", "registro_b2_tabla_relaciones", [
    ["estado", "registro_b2_estado"], ["periodo", "registro_b2_periodo"], ["regimen_catalogo", "registro_b2_regimen"], ["modalidad_catalogo", "registro_b2_modalidad"], ["unidad", "registro_b2_unidad"],
  ]],
  ["ocupaciones", "registro_b2_ocupaciones", "registro_b2_tabla_ocupaciones", [
    ["puesto", "registro_b2_puesto"], ["plaza", "registro_b2_plaza"], ["modalidad_catalogo", "registro_b2_modalidad"], ["clase", "registro_b2_clase"], ["unidad", "registro_b2_unidad"], ["periodo", "registro_b2_periodo"],
  ]],
  ["situaciones", "registro_b2_situaciones", "registro_b2_tabla_situaciones", [
    ["situacion", "registro_b2_situacion"], ["periodo", "registro_b2_periodo"], ["acto", "registro_b2_acto"],
  ]],
  ["servicios", "registro_b2_servicios", "registro_b2_tabla_servicios", [
    ["clase_servicio_catalogo", "registro_b2_clase_servicio"], ["estado", "registro_b2_estado"], ["periodo_servicio", "registro_b2_periodo"], ["dias", "registro_b2_dias_reconocidos"], ["fuente", "registro_b2_fuente"],
  ]],
]);
const ESTADOS = Object.freeze({
  declarado: "registro_b2_estado_declarado", comprobado: "registro_b2_estado_comprobado",
  reconocido: "registro_b2_estado_reconocido", vigente: "registro_b2_estado_vigente",
  suspendida: "registro_b2_estado_suspendida",
  finalizada: "registro_b2_estado_finalizada", finalizado: "registro_b2_estado_finalizado",
  cese: "registro_b2_estado_cese", disponible: "registro_b2_estado_disponible",
  reservada: "registro_b2_estado_reservada", no_cubrible: "registro_b2_estado_no_cubrible",
  vacante_sin_ocupacion: "registro_b2_estado_vacante_sin_ocupacion",
});
const CLASES = Object.freeze({ titular: "registro_b2_clase_titular", provisional: "registro_b2_clase_provisional", temporal: "registro_b2_clase_temporal", reserva: "registro_b2_clase_reserva" });

function nodo(d, etiqueta, texto) { const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; return n; }
function textoSeguro(valor, maximo = 256) { return typeof valor === "string" && valor.length <= maximo && !/[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]/u.test(valor) ? valor : ""; }
function fechaValida(valor) { if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false; const fecha = new Date(`${valor}T12:00:00Z`); return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor; }
function instanteValido(valor) { return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/u.test(valor) && Number.isFinite(Date.parse(valor)); }
function formatoFecha(valor, t) { if (!fechaValida(valor)) return t("registro_b2_sin_valor"); return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(`${valor}T12:00:00Z`)); }
function formatoInstante(valor, t) { if (!instanteValido(valor)) return t("registro_b2_sin_valor"); return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor)); }
function periodo(traza, t) { return `${formatoFecha(traza?.desde, t)} – ${traza?.hasta ? formatoFecha(traza.hasta, t) : t("registro_b2_actual")}`; }
function etiquetaEstado(valor, t) { return Object.hasOwn(ESTADOS, valor) ? t(ESTADOS[valor]) : t("registro_b2_estado_desconocido"); }
function referencia(valor, t) { const texto = textoSeguro(valor, 128); return texto ? t("registro_b2_referencia", { referencia: texto }) : t("registro_b2_sin_valor"); }
function nombreOReferencia(item, nombre, ref, t) { return textoSeguro(item?.[nombre], 240) || referencia(item?.[ref], t); }
function presentar(item, campo, t) {
  switch (campo) {
    case "estado": return etiquetaEstado(item.estado, t);
    case "periodo": return periodo(item.traza, t);
    case "periodo_servicio": return `${formatoFecha(item.periodo_desde, t)} – ${formatoFecha(item.periodo_hasta, t)}`;
    case "dias": return Number.isSafeInteger(item.dias_reconocidos) && item.dias_reconocidos >= 0 ? new Intl.NumberFormat("es-ES").format(item.dias_reconocidos) : t("registro_b2_sin_valor");
    case "clase": return Object.hasOwn(CLASES, item.clase) ? t(CLASES[item.clase]) : t("registro_b2_estado_desconocido");
    case "unidad": return nombreOReferencia(item, "unidad_denominacion", "unidad_ref", t);
    case "puesto": return nombreOReferencia(item, "puesto_denominacion", "puesto_ref", t);
    case "plaza": return nombreOReferencia(item, "plaza_denominacion", "plaza_ref", t);
    case "situacion": return textoSeguro(item.catalogo_snapshot?.situacion?.denominacion, 256) || nombreOReferencia(item, "situacion_denominacion", "codigo_ref", t);
    case "acto": return referencia(item.traza?.acto_ref, t);
    case "fuente": return referencia(item.traza?.fuente_ref, t);
    case "cobertura": return etiquetaEstado(item.estado_cobertura, t);
    default: return t("registro_b2_sin_valor");
  }
}
function estado(d, texto, alerta = false) { const p = nodo(d, "p", texto); p.className = "personal-registro-b2-estado"; p.setAttribute("role", alerta ? "alert" : "status"); return p; }
function panel(d, titulo) { const s = nodo(d, "section"); s.className = "panel"; const h = nodo(d, "header"); h.className = "cabecera-panel"; h.append(nodo(d, "h3", titulo)); const c = nodo(d, "div"); c.className = "cuerpo-panel"; s.append(h, c); return { elemento: s, cuerpo: c }; }
function tabla(d, t, titulo, columnas, filas) {
  const region = nodo(d, "div"); region.className = "tabla-contenedor"; region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t(titulo));
  const tab = nodo(d, "table"); tab.className = "tabla-datos"; tab.append(nodo(d, "caption", t(titulo)));
  const thead = nodo(d, "thead"); const cab = nodo(d, "tr");
  for (const [, clave] of columnas) { const th = nodo(d, "th", t(clave)); th.setAttribute("scope", "col"); cab.append(th); }
  thead.append(cab); tab.append(thead);
  const body = nodo(d, "tbody");
  for (const item of filas) {
    const tr = nodo(d, "tr");
    for (const [campo] of columnas) {
      const td = nodo(d, "td");
      const tipoCatalogo = { regimen_catalogo: "regimen", modalidad_catalogo: "modalidad", clase_servicio_catalogo: "clase_servicio", situacion: "situacion" }[campo];
      const snapshot = tipoCatalogo ? item.catalogo_snapshot?.[tipoCatalogo] : undefined;
      if (snapshot && textoSeguro(snapshot.denominacion, 256)) {
        td.append(nodo(d, "span", snapshot.denominacion));
        const secundario = nodo(d, "small", t("registro_b2_catalogo_ref_version", { ref: textoSeguro(snapshot.ref, 160), version: Number.isSafeInteger(snapshot.version) ? new Intl.NumberFormat("es-ES").format(snapshot.version) : "" }));
        secundario.className = "personal-registro-b2-secundario"; td.append(secundario);
      } else td.textContent = presentar(item, campo, t);
      tr.append(td);
    }
    body.append(tr);
  }
  tab.append(body); region.append(tab); return region;
}
function validarFicha(respuesta) {
  const f = respuesta?.ficha;
  if (!f || typeof f !== "object" || !textoSeguro(f.empleado_ref, 128) || !Number.isSafeInteger(f.version) || f.version < 1 ||
      f.eficacia_administrativa !== false || f.firma_oficial !== false || !f.corte || !fechaValida(f.corte.vigente_en) || !instanteValido(f.corte.conocido_en) ||
      !BLOQUES.every(([clave]) => Array.isArray(f[clave]) && f[clave].length <= 200 && f[clave].every((fila) => fila && typeof fila === "object" && fila.traza && fechaValida(fila.traza.desde) && (!fila.traza.hasta || fechaValida(fila.traza.hasta)) && Number.isSafeInteger(fila.traza.version) && fila.traza.version > 0 && (clave !== "servicios" || (fechaValida(fila.periodo_desde) && fechaValida(fila.periodo_hasta)))))) throw new TypeError("ficha de empleado incompatible");
  return f;
}
function validarPagina(respuesta) {
  const p = respuesta?.pagina;
  if (!p || typeof p !== "object" || !p.corte || !fechaValida(p.corte.vigente_en) || !instanteValido(p.corte.conocido_en) ||
      !Array.isArray(p.vacantes) || p.vacantes.length > 100 || p.cobertura !== "completa" ||
      (p.cursor_siguiente !== null && p.cursor_siguiente !== undefined && typeof p.cursor_siguiente !== "string") ||
      !p.vacantes.every((fila) => fila && typeof fila === "object" && fila.traza && fechaValida(fila.traza.desde))) throw new TypeError("página de vacantes incompatible");
  return p;
}
function hoyMadrid(reloj) {
  const partes = new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/Madrid", year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(reloj());
  const dato = (tipo) => partes.find((parte) => parte.type === tipo)?.value;
  return `${dato("year")}-${dato("month")}-${dato("day")}`;
}
function localFechaHora(fecha) {
  const dos = (valor) => String(valor).padStart(2, "0");
  return `${fecha.getFullYear()}-${dos(fecha.getMonth() + 1)}-${dos(fecha.getDate())}T${dos(fecha.getHours())}:${dos(fecha.getMinutes())}`;
}
function conocidoUTC(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/u.test(valor)) return "";
  const fecha = new Date(valor);
  return Number.isFinite(fecha.getTime()) ? fecha.toISOString().replace(/Z$/u, "000Z") : "";
}

/** Montaje de lectura RRHH. El empleado se recibe de una selección autorizada del shell. */
export function montarRegistroB2({ raiz, cliente, clienteCatalogos, fuenteActos, empleadoRef = "", personaRef = "", catalogos, anunciar = () => {}, registrarDesmontar, reloj = () => new Date() } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || !cliente?.consultarFicha || !cliente?.listarVacantes ||
      (empleadoRef !== "" && !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(empleadoRef)) ||
      (personaRef !== "" && !/^per_[A-Za-z0-9_-]{22,128}$/u.test(personaRef)) ||
      typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") || typeof reloj !== "function") throw new TypeError("Registro de Personal no disponible");
  const d = raiz.ownerDocument; const t = crearTraductorPersonal(); const s = nodo(d, "section"); s.className = "modulo-personal personal-registro-b2"; s.dataset.personalRegistroB2 = "";
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-vista"; cabecera.append(nodo(d, "h2", t("registro_b2_titulo")));
  const ayuda = nodo(d, "details"); ayuda.className = "personal-registro-b2-ayuda"; const resumenAyuda = nodo(d, "summary", "?"); resumenAyuda.setAttribute("aria-label", t("registro_b2_ayuda_abrir")); const textoAyuda = nodo(d, "div"); textoAyuda.append(nodo(d, "p", t("registro_b2_ayuda"))); ayuda.append(resumenAyuda, textoAyuda); cabecera.append(ayuda);
  const pestañas = nodo(d, "div"); pestañas.className = "personal-registro-b2-pestanas"; pestañas.setAttribute("role", "tablist"); pestañas.setAttribute("aria-label", t("registro_b2_pestanas"));
  const contenido = nodo(d, "div"); contenido.className = "personal-registro-b2-contenido"; contenido.setAttribute("role", "tabpanel"); contenido.setAttribute("tabindex", "0"); contenido.id = "personal-registro-b2-panel";
  s.append(cabecera, pestañas, contenido); raiz.append(s);
  let activo = true; let vista = "ficha"; let vuelo; let empleado = empleadoRef; let turno = 0; let ultimaFicha; let montajeActos; let montajeCatalogos;
  let vigenteEn = hoyMadrid(reloj); let conocidoEnLocal = localFechaHora(reloj());
  let seleccionRelacion = ""; let cursor = ""; let anteriores = [];
  const botones = new Map();
  const limpiarVuelo = () => { turno += 1; vuelo?.abort(); vuelo = undefined; };
  const desmontar = () => { if (!activo) return; activo = false; limpiarVuelo(); montajeActos?.desmontar(); montajeCatalogos?.desmontar(); s.remove?.(); };
  registrarDesmontar?.(desmontar);

  function pintarFicha(ficha) {
    const ultimas = new Map();
    for (const relacion of ficha.relaciones) if (!ultimas.has(relacion.relacion_ref) || ultimas.get(relacion.relacion_ref).traza.version < relacion.traza.version) ultimas.set(relacion.relacion_ref, relacion);
    const relacionesUnicas = [...ultimas.values()];
    const resumen = nodo(d, "div"); resumen.className = "panel";
    const cuerpoResumen = nodo(d, "div"); cuerpoResumen.className = "cuerpo-panel personal-registro-b2-resumen";
    cuerpoResumen.append(nodo(d, "h3", t("registro_b2_ficha")), nodo(d, "p", t("registro_b2_version", { version: new Intl.NumberFormat("es-ES").format(ficha.version) })));
    resumen.append(cuerpoResumen); contenido.append(resumen);
    if (relacionesUnicas.length > 1) {
      const form = nodo(d, "div"); form.className = "personal-registro-b2-toolbar";
      const label = nodo(d, "label", t("registro_b2_elegir_relacion")); const select = nodo(d, "select"); select.dataset.registroB2Relacion = "";
      const opcionVacia = nodo(d, "option", t("registro_b2_elegir_relacion")); opcionVacia.value = ""; select.append(opcionVacia);
      for (const relacion of relacionesUnicas) { const opcion = nodo(d, "option", nombreOReferencia(relacion, "unidad_denominacion", "relacion_ref", t)); opcion.value = relacion.relacion_ref; select.append(opcion); }
      select.value = ficha.relaciones.some((r) => r.relacion_ref === seleccionRelacion) ? seleccionRelacion : "";
      select.addEventListener("change", () => { seleccionRelacion = select.value; contenido.replaceChildren(); pintarFicha(ficha); });
      label.append(select); form.append(label); contenido.append(form);
    } else seleccionRelacion = relacionesUnicas[0]?.relacion_ref || "";
    const rejilla = nodo(d, "div"); rejilla.className = "personal-registro-b2-grupos";
    for (const [clave, titulo, nombreTabla, columnas] of BLOQUES) {
      const bloque = panel(d, t(titulo));
      const filas = clave === "relaciones" ? ficha[clave] : ficha[clave].filter((fila) => fila.relacion_ref === seleccionRelacion);
      if (clave !== "relaciones" && relacionesUnicas.length > 1 && !seleccionRelacion) bloque.cuerpo.append(estado(d, t("registro_b2_relacion_pendiente")));
      else if (filas.length === 0) bloque.cuerpo.append(estado(d, t("registro_b2_vacio")));
      else bloque.cuerpo.append(tabla(d, t, nombreTabla, columnas, filas));
      rejilla.append(bloque.elemento);
    }
    contenido.append(rejilla);
  }
  function pintarVacantes(pagina) {
    const p = panel(d, t("registro_b2_vacantes"));
    if (pagina.vacantes.length === 0) p.cuerpo.append(estado(d, t("registro_b2_vacio")));
    else p.cuerpo.append(tabla(d, t, "registro_b2_tabla_vacantes", [
      ["puesto", "registro_b2_puesto"], ["plaza", "registro_b2_plaza"], ["unidad", "registro_b2_unidad"], ["cobertura", "registro_b2_estado"],
    ], pagina.vacantes));
    const nav = nodo(d, "nav"); nav.className = "personal-registro-b2-paginacion"; nav.setAttribute("aria-label", t("registro_b2_paginacion"));
    const anterior = nodo(d, "button", t("registro_b2_anterior")); anterior.type = "button"; anterior.disabled = anteriores.length === 0;
    anterior.addEventListener("click", () => { cursor = anteriores.pop() || ""; consultar(); });
    const siguiente = nodo(d, "button", t("registro_b2_siguiente")); siguiente.type = "button"; siguiente.disabled = !pagina.cursor_siguiente;
    siguiente.addEventListener("click", () => { anteriores.push(cursor); cursor = pagina.cursor_siguiente; consultar(); });
    nav.append(anterior, siguiente); p.elemento.append(nav);
    contenido.append(p.elemento);
  }
  async function consultar() {
    if (!activo) return;
    limpiarVuelo();
    const conocidoEn = conocidoUTC(conocidoEnLocal);
    if (!fechaValida(vigenteEn) || !conocidoEn) { contenido.replaceChildren(estado(d, t("registro_b2_fecha_invalida"), true)); return; }
    if (vista === "ficha" && !empleado) { contenido.replaceChildren(estado(d, t("registro_b2_sin_empleado"))); return; }
    const actual = new AbortController(); vuelo = actual; const secuencia = turno;
    contenido.replaceChildren(estado(d, t("registro_b2_cargando")));
    try {
      const respuesta = vista === "ficha"
        ? await cliente.consultarFicha({ empleadoRef: empleado, vigenteEn, conocidoEn, signal: actual.signal })
        : await cliente.listarVacantes({ vigenteEn, conocidoEn, limite: 25, cursor, signal: actual.signal });
      if (!activo || vuelo !== actual || actual.signal.aborted || secuencia !== turno) return;
      contenido.replaceChildren();
      if (vista === "ficha") {
        const ficha = validarFicha(respuesta);
        if (ficha.empleado_ref !== empleado || ficha.corte.vigente_en !== vigenteEn || Date.parse(ficha.corte.conocido_en) !== Date.parse(conocidoEn)) throw new TypeError("ficha de otro empleado o corte");
        ultimaFicha = ficha; pintarFicha(ficha);
      } else {
        const pagina = validarPagina(respuesta);
        if (pagina.corte.vigente_en !== vigenteEn || Date.parse(pagina.corte.conocido_en) !== Date.parse(conocidoEn) || pagina.limite !== 25 || pagina.cursor !== cursor) throw new TypeError("página de otro corte");
        pintarVacantes(pagina);
      }
    } catch (error) {
      if (!activo || vuelo !== actual || actual.signal.aborted || secuencia !== turno) return;
      const errorCliente = error instanceof ErrorRegistroB2 || error?.name === "ErrorRegistroB2";
      const reintentable = vista === "vacantes" && errorCliente && error.estado === 503;
      const clave = reintentable && error.codigo === "cobertura_no_acreditada" ? "registro_b2_no_determinable" : errorCliente && [401, 403].includes(error.estado) ? "registro_b2_denegado" : "registro_b2_error";
      contenido.replaceChildren(estado(d, t(clave), true));
      if (reintentable) { const reintentar = nodo(d, "button", t("registro_b2_reintentar")); reintentar.type = "button"; reintentar.addEventListener("click", () => consultar()); contenido.append(reintentar); }
      anunciar(t(clave), "error");
    } finally { if (vuelo === actual) vuelo = undefined; }
  }
  const form = nodo(d, "form"); form.className = "personal-registro-b2-toolbar";
  const etiquetaVigente = nodo(d, "label", t("registro_b2_vigente_en")); const campoVigente = nodo(d, "input"); campoVigente.type = "date"; campoVigente.required = true; campoVigente.value = vigenteEn; etiquetaVigente.append(campoVigente);
  const etiquetaConocido = nodo(d, "label", t("registro_b2_conocido_en")); const campoConocido = nodo(d, "input"); campoConocido.type = "datetime-local"; campoConocido.required = true; campoConocido.value = conocidoEnLocal; etiquetaConocido.append(campoConocido);
  const boton = nodo(d, "button", t("registro_b2_consultar")); boton.type = "submit"; form.append(etiquetaVigente, etiquetaConocido, boton);
  form.addEventListener("submit", (evento) => { evento.preventDefault(); vigenteEn = campoVigente.value; conocidoEnLocal = campoConocido.value; cursor = ""; anteriores = []; consultar(); });
  s.replaceChildren(cabecera, pestañas, form, contenido);
  function cambiarVista(nueva) {
    if (!activo || !botones.has(nueva)) return;
    limpiarVuelo(); montajeActos?.desmontar(); montajeActos = undefined; montajeCatalogos?.desmontar(); montajeCatalogos = undefined; vista = nueva; ayuda.open = false;
    textoAyuda.replaceChildren(nodo(d, "p", t(vista === "actos" ? "registro_b2_acto_ayuda" : vista === "catalogos" ? "registro_b2_catalogos_ayuda" : "registro_b2_ayuda")));
    form.hidden = vista === "actos" || vista === "catalogos";
    for (const [clave, tab] of botones) { tab.setAttribute("aria-selected", String(clave === vista)); tab.setAttribute("tabindex", clave === vista ? "0" : "-1"); }
    contenido.setAttribute("aria-labelledby", `personal-registro-b2-tab-${vista}`);
    if (vista === "catalogos") {
      contenido.replaceChildren();
      try { montajeCatalogos = montarCatalogosRegistroB2({ raiz: contenido, cliente: clienteCatalogos, fuenteActos, anunciar }); }
      catch { contenido.append(estado(d, t("registro_b2_catalogos_error"), true)); }
      return;
    }
    if (vista === "actos") {
      contenido.replaceChildren(estado(d, t("registro_b2_catalogos_cargando")));
      const actual = new AbortController(); vuelo = actual; const secuencia = turno;
      cargarOpcionesPublicadasCatalogoB2(clienteCatalogos, { signal: actual.signal }).then((publicadas) => {
        if (!activo || vista !== "actos" || vuelo !== actual || actual.signal.aborted || secuencia !== turno) return;
        const opciones = { ...catalogos, regimenes: publicadas.regimenes, modalidades: publicadas.modalidades, situaciones: publicadas.situaciones, clasesServicio: publicadas.clasesServicio };
        contenido.replaceChildren();
        if (!accionesRegistroB2Disponibles({ cliente, catalogos: opciones, personaRef, empleadoRef: empleado })) { contenido.append(estado(d, t("registro_b2_actos_no_disponibles"))); return; }
        montajeActos = montarActosRegistroB2({ raiz: contenido, cliente, catalogos: opciones, personaRef, empleadoRef: empleado, ficha: ultimaFicha, anunciar, alRegistrar: (recibo) => { empleado = recibo.empleado_ref; ultimaFicha = undefined; seleccionRelacion = recibo.relacion_ref; } });
      }).catch(() => {
        if (!activo || vista !== "actos" || vuelo !== actual || actual.signal.aborted || secuencia !== turno) return;
        contenido.replaceChildren(estado(d, t("registro_b2_catalogos_error"), true));
      }).finally(() => { if (vuelo === actual) vuelo = undefined; });
      return;
    }
    consultar();
  }
  const vistas = [["ficha", "registro_b2_ficha"], ["vacantes", "registro_b2_vacantes"]];
  if (typeof clienteCatalogos?.listar === "function" && typeof clienteCatalogos?.cambiar === "function") {
    vistas.push(["catalogos", "registro_b2_catalogos"]);
    const base = ["organismos", "unidades", "plazas", "puestos", "actos", "fuentes"].every((clave) => Array.isArray(catalogos?.[clave]));
    if (base && typeof cliente?.registrarAlta === "function" && typeof cliente?.registrarHecho === "function" && (personaRef || empleadoRef)) vistas.push(["actos", "registro_b2_actuaciones"]);
  }
  vistas.forEach(([clave, etiqueta], indice) => {
    const tab = nodo(d, "button", t(etiqueta)); tab.type = "button"; tab.id = `personal-registro-b2-tab-${clave}`; tab.setAttribute("role", "tab"); tab.setAttribute("aria-controls", contenido.id);
    tab.dataset.registroB2Tab = clave; tab.addEventListener("click", () => cambiarVista(clave));
    tab.addEventListener("keydown", (evento) => { if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(evento.key)) return; evento.preventDefault(); const destino = evento.key === "Home" ? 0 : evento.key === "End" ? vistas.length - 1 : (indice + (evento.key === "ArrowRight" ? 1 : -1) + vistas.length) % vistas.length; const elegido = vistas[destino][0]; cambiarVista(elegido); botones.get(elegido)?.focus?.(); });
    botones.set(clave, tab); pestañas.append(tab);
  });
  const cambiarEmpleado = (ref) => { if (ref !== "" && (typeof ref !== "string" || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(ref))) throw new TypeError("referencia de empleado no válida"); empleado = ref; ultimaFicha = undefined; seleccionRelacion = ""; if (vista === "ficha") consultar(); else if (vista === "actos") cambiarVista("actos"); };
  cambiarVista("ficha");
  return Object.freeze({ desmontar, cambiarEmpleado });
}
