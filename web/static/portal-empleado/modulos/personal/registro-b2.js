import { crearTraductorPersonal } from "./i18n.js?v=20260925-b2-sin-acto-fuente-v1";
import { ErrorRegistroB2 } from "./registro-b2-cliente.js?v=20260925-b2-selector-v1";
import { accionesRegistroB2Disponibles, montarActosRegistroB2 } from "./registro-b2-actos.js?v=20260925-b2-sin-acto-fuente-v1";
import { cargarOpcionesPublicadasCatalogoB2, montarCatalogosRegistroB2 } from "./registro-b2-catalogos.js?v=20260925-b2-sin-acto-fuente-v1";

const BLOQUES = Object.freeze([
  ["relaciones", "registro_b2_relaciones", "registro_b2_tabla_relaciones", [
    ["estado", "registro_b2_estado"], ["periodo", "registro_b2_periodo"], ["regimen_catalogo", "registro_b2_regimen"], ["modalidad_catalogo", "registro_b2_modalidad"], ["unidad", "registro_b2_unidad"],
  ]],
  ["ocupaciones", "registro_b2_ocupaciones", "registro_b2_tabla_ocupaciones", [
    ["puesto", "registro_b2_puesto"], ["plaza", "registro_b2_plaza"], ["modalidad_catalogo", "registro_b2_modalidad"], ["clase", "registro_b2_clase"], ["unidad", "registro_b2_unidad"], ["periodo", "registro_b2_periodo"],
  ]],
  ["situaciones", "registro_b2_situaciones", "registro_b2_tabla_situaciones", [
    ["situacion", "registro_b2_situacion"], ["periodo", "registro_b2_periodo"],
  ]],
  ["servicios", "registro_b2_servicios", "registro_b2_tabla_servicios", [
    ["clase_servicio_catalogo", "registro_b2_clase_servicio"], ["estado", "registro_b2_estado"], ["periodo_servicio", "registro_b2_periodo"], ["dias", "registro_b2_dias_reconocidos"],
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
/** Una referencia es un código interno: nunca se pinta. Solo indica si consta o no el dato sin denominación. */
function sinDenominacion(valor, t) { return textoSeguro(valor, 160) ? t("registro_b2_sin_denominacion") : t("registro_b2_sin_valor"); }
function nombreOReferencia(item, nombre, ref, t) { return textoSeguro(item?.[nombre], 240) || sinDenominacion(item?.[ref], t); }
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
      td.textContent = snapshot && textoSeguro(snapshot.denominacion, 256) ? snapshot.denominacion : presentar(item, campo, t);
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
function validarPaginaEmpleados(respuesta) {
  const p = respuesta?.pagina;
  if (!p || typeof p !== "object" || !p.corte || !fechaValida(p.corte.vigente_en) || !instanteValido(p.corte.conocido_en) ||
      !Array.isArray(p.empleados) || p.empleados.length > 100 || typeof p.cursor_siguiente !== "string" || p.cursor_siguiente.length > 256 ||
      !p.empleados.every((e) => e && typeof e === "object" && /^emp_[A-Za-z0-9_-]{22,128}$/u.test(e.empleado_ref) && !Object.hasOwn(e, "persona_ref") &&
        Array.isArray(e.relaciones) && e.relaciones.length <= 20 &&
        e.relaciones.every((r) => r && typeof r === "object" && ["vigente", "suspendida"].includes(r.estado) && textoSeguro(r.unidad_ref, 160) && r.traza && fechaValida(r.traza.desde)))) throw new TypeError("página de empleados incompatible");
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
export function montarRegistroB2({ raiz, cliente, clienteCatalogos, empleadoRef = "", personaRef = "", catalogos, anunciar = () => {}, registrarDesmontar, reloj = () => new Date() } = {}) {
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
  let seleccionRelacion = "";
  const paginas = { vacantes: { cursor: "", anteriores: [] }, empleados: { cursor: "", anteriores: [] } };
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
      for (const relacion of relacionesUnicas) { const opcion = nodo(d, "option", `${nombreOReferencia(relacion, "unidad_denominacion", "unidad_ref", t)} · ${periodo(relacion.traza, t)}`); opcion.value = relacion.relacion_ref; select.append(opcion); }
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
  function paginacion(clave, pagina, etiqueta) {
    const estadoPagina = paginas[clave];
    const nav = nodo(d, "nav"); nav.className = "personal-registro-b2-paginacion"; nav.setAttribute("aria-label", t(etiqueta));
    const anterior = nodo(d, "button", t("registro_b2_anterior")); anterior.type = "button"; anterior.disabled = estadoPagina.anteriores.length === 0;
    anterior.addEventListener("click", () => { estadoPagina.cursor = estadoPagina.anteriores.pop() || ""; consultar(); });
    const siguiente = nodo(d, "button", t("registro_b2_siguiente")); siguiente.type = "button"; siguiente.disabled = !pagina.cursor_siguiente;
    siguiente.addEventListener("click", () => { estadoPagina.anteriores.push(estadoPagina.cursor); estadoPagina.cursor = pagina.cursor_siguiente; consultar(); });
    nav.append(anterior, siguiente);
    return nav;
  }
  function describirEmpleado(empleado) {
    const r = empleado.relaciones[0];
    if (!r) return t("registro_b2_sin_relacion_vigente");
    return [textoSeguro(r.unidad_denominacion, 300) || sinDenominacion(r.unidad_ref, t), textoSeguro(r.puesto_denominacion, 300)].filter(Boolean).join(" · ");
  }
  function pintarEmpleados(pagina) {
    const p = panel(d, t("registro_b2_empleados"));
    const alta = nodo(d, "button", t("registro_b2_nueva_alta")); alta.type = "button"; alta.disabled = true; alta.setAttribute("aria-disabled", "true");
    alta.className = "personal-registro-b2-alta"; alta.title = t("registro_b2_nueva_alta_motivo");
    const motivo = nodo(d, "span", t("registro_b2_nueva_alta_motivo")); motivo.className = "solo-lectura"; motivo.id = "personal-registro-b2-alta-motivo";
    alta.setAttribute("aria-describedby", motivo.id);
    p.elemento.children[0].append(alta, motivo);
    if (pagina.empleados.length === 0) p.cuerpo.append(estado(d, t("registro_b2_empleados_vacio")));
    else {
      const region = nodo(d, "div"); region.className = "tabla-contenedor"; region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("registro_b2_tabla_empleados"));
      const tab = nodo(d, "table"); tab.className = "tabla-datos personal-registro-b2-empleados"; tab.append(nodo(d, "caption", t("registro_b2_tabla_empleados")));
      const thead = nodo(d, "thead"); const cab = nodo(d, "tr");
      for (const clave of ["registro_b2_unidad", "registro_b2_puesto", "registro_b2_regimen", "registro_b2_modalidad", "registro_b2_estado"]) { const th = nodo(d, "th", t(clave)); th.setAttribute("scope", "col"); cab.append(th); }
      const acciones = nodo(d, "th"); acciones.setAttribute("scope", "col"); acciones.append(Object.assign(nodo(d, "span", t("registro_b2_ver_ficha")), { className: "solo-lectura" })); cab.append(acciones);
      thead.append(cab); tab.append(thead);
      const body = nodo(d, "tbody");
      for (const empleado of pagina.empleados) {
        const r = empleado.relaciones[0]; const tr = nodo(d, "tr");
        const unidad = nodo(d, "td");
        if (r) {
          unidad.append(nodo(d, "span", textoSeguro(r.unidad_denominacion, 300) || sinDenominacion(r.unidad_ref, t)));
          if (empleado.relaciones.length > 1) {
            const mas = empleado.relaciones.length - 1;
            const extra = nodo(d, "small", mas === 1 ? t("registro_b2_mas_relaciones_uno") : t("registro_b2_mas_relaciones_otro", { total: new Intl.NumberFormat("es-ES").format(mas) }));
            extra.className = "personal-registro-b2-secundario"; unidad.append(extra);
          }
        } else unidad.textContent = t("registro_b2_sin_relacion_vigente");
        tr.append(unidad,
          nodo(d, "td", (r && textoSeguro(r.puesto_denominacion, 300)) || t("registro_b2_sin_valor")),
          nodo(d, "td", (r && textoSeguro(r.regimen_denominacion, 300)) || t("registro_b2_sin_valor")),
          nodo(d, "td", (r && textoSeguro(r.modalidad_denominacion, 300)) || t("registro_b2_sin_valor")),
          nodo(d, "td", r ? etiquetaEstado(r.estado, t) : t("registro_b2_sin_valor")));
        if (!r) for (const celda of tr.children.slice?.(1) ?? [...tr.children].slice(1)) celda.dataset.registroB2Vacio = "";
        const celda = nodo(d, "td"); const ver = nodo(d, "button", t("registro_b2_ver_ficha")); ver.type = "button"; ver.className = "personal-registro-b2-ver";
        ver.dataset.registroB2Empleado = empleado.empleado_ref;
        ver.setAttribute("aria-label", t("registro_b2_ver_ficha_de", { descripcion: describirEmpleado(empleado) }));
        ver.addEventListener("click", () => elegirEmpleado(empleado.empleado_ref));
        celda.append(ver); tr.append(celda); body.append(tr);
      }
      tab.append(body); region.append(tab); p.cuerpo.append(region);
    }
    p.cuerpo.append(paginacion("empleados", pagina, "registro_b2_paginacion_empleados"));
    contenido.append(p.elemento);
  }
  function pintarVacantes(pagina) {
    const p = panel(d, t("registro_b2_vacantes"));
    if (pagina.vacantes.length === 0) p.cuerpo.append(estado(d, t("registro_b2_vacio")));
    else p.cuerpo.append(tabla(d, t, "registro_b2_tabla_vacantes", [
      ["puesto", "registro_b2_puesto"], ["plaza", "registro_b2_plaza"], ["unidad", "registro_b2_unidad"], ["cobertura", "registro_b2_estado"],
    ], pagina.vacantes));
    p.cuerpo.append(paginacion("vacantes", pagina, "registro_b2_paginacion"));
    contenido.append(p.elemento);
  }
  async function consultar() {
    if (!activo) return;
    limpiarVuelo();
    const conocidoEn = conocidoUTC(conocidoEnLocal);
    if (!fechaValida(vigenteEn) || !conocidoEn) { contenido.replaceChildren(estado(d, t("registro_b2_fecha_invalida"), true)); return; }
    if (vista === "ficha" && !empleado) {
      const ir = nodo(d, "button", t("registro_b2_elegir_empleado")); ir.type = "button"; ir.className = "personal-registro-b2-ir";
      ir.addEventListener("click", () => { cambiarVista("empleados"); botones.get("empleados")?.focus?.(); });
      contenido.replaceChildren(estado(d, t("registro_b2_sin_empleado")), ir); return;
    }
    const actual = new AbortController(); vuelo = actual; const secuencia = turno;
    contenido.replaceChildren(estado(d, t("registro_b2_cargando")));
    try {
      const cursorVista = paginas[vista]?.cursor ?? "";
      const respuesta = vista === "ficha"
        ? await cliente.consultarFicha({ empleadoRef: empleado, vigenteEn, conocidoEn, signal: actual.signal })
        : vista === "empleados"
          ? await cliente.listarEmpleados({ vigenteEn, conocidoEn, limite: 25, cursor: cursorVista, signal: actual.signal })
          : await cliente.listarVacantes({ vigenteEn, conocidoEn, limite: 25, cursor: cursorVista, signal: actual.signal });
      if (!activo || vuelo !== actual || actual.signal.aborted || secuencia !== turno) return;
      contenido.replaceChildren();
      if (vista === "ficha") {
        const ficha = validarFicha(respuesta);
        if (ficha.empleado_ref !== empleado || ficha.corte.vigente_en !== vigenteEn || Date.parse(ficha.corte.conocido_en) !== Date.parse(conocidoEn)) throw new TypeError("ficha de otro empleado o corte");
        ultimaFicha = ficha; pintarFicha(ficha);
      } else {
        const pagina = vista === "empleados" ? validarPaginaEmpleados(respuesta) : validarPagina(respuesta);
        if (pagina.corte.vigente_en !== vigenteEn || Date.parse(pagina.corte.conocido_en) !== Date.parse(conocidoEn) || pagina.limite !== 25 || pagina.cursor !== cursorVista) throw new TypeError("página de otro corte");
        if (vista === "empleados") pintarEmpleados(pagina); else pintarVacantes(pagina);
      }
    } catch (error) {
      if (!activo || vuelo !== actual || actual.signal.aborted || secuencia !== turno) return;
      const errorCliente = error instanceof ErrorRegistroB2 || error?.name === "ErrorRegistroB2";
      const reintentable = vista === "vacantes" && errorCliente && error.estado === 503;
      const clave = reintentable && error.codigo === "cobertura_no_acreditada" ? "registro_b2_no_determinable" : errorCliente && [401, 403].includes(error.estado) ? "registro_b2_denegado" : "registro_b2_error";
      contenido.replaceChildren(estado(d, t(clave), true));
      if (reintentable) { const reintentar = nodo(d, "button", t("registro_b2_reintentar")); reintentar.type = "button"; reintentar.className = "personal-registro-b2-ir"; reintentar.addEventListener("click", () => consultar()); contenido.append(reintentar); }
      anunciar(t(clave), "error");
    } finally { if (vuelo === actual) vuelo = undefined; }
  }
  const form = nodo(d, "form"); form.className = "personal-registro-b2-toolbar";
  const etiquetaVigente = nodo(d, "label", t("registro_b2_vigente_en")); const campoVigente = nodo(d, "input"); campoVigente.type = "date"; campoVigente.required = true; campoVigente.value = vigenteEn; etiquetaVigente.append(campoVigente);
  const etiquetaConocido = nodo(d, "label", t("registro_b2_conocido_en")); const campoConocido = nodo(d, "input"); campoConocido.type = "datetime-local"; campoConocido.required = true; campoConocido.value = conocidoEnLocal; etiquetaConocido.append(campoConocido);
  const boton = nodo(d, "button", t("registro_b2_consultar")); boton.type = "submit"; form.append(etiquetaVigente, etiquetaConocido, boton);
  form.addEventListener("submit", (evento) => { evento.preventDefault(); vigenteEn = campoVigente.value; conocidoEnLocal = campoConocido.value; for (const p of Object.values(paginas)) { p.cursor = ""; p.anteriores = []; } consultar(); });
  s.replaceChildren(cabecera, pestañas, form, contenido);
  function cambiarVista(nueva) {
    if (!activo || !botones.has(nueva)) return;
    limpiarVuelo(); montajeActos?.desmontar(); montajeActos = undefined; montajeCatalogos?.desmontar(); montajeCatalogos = undefined; vista = nueva; ayuda.open = false;
    textoAyuda.replaceChildren(nodo(d, "p", t(vista === "actos" ? "registro_b2_acto_ayuda" : vista === "catalogos" ? "registro_b2_catalogos_ayuda" : "registro_b2_ayuda")));
    if (vista === "empleados") textoAyuda.append(nodo(d, "p", t("registro_b2_nueva_alta_motivo")));
    form.hidden = vista === "actos" || vista === "catalogos";
    for (const [clave, tab] of botones) { tab.setAttribute("aria-selected", String(clave === vista)); tab.setAttribute("tabindex", clave === vista ? "0" : "-1"); }
    contenido.setAttribute("aria-labelledby", `personal-registro-b2-tab-${vista}`);
    if (vista === "catalogos") {
      contenido.replaceChildren();
      try { montajeCatalogos = montarCatalogosRegistroB2({ raiz: contenido, cliente: clienteCatalogos, anunciar }); }
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
  if (typeof cliente.listarEmpleados === "function") vistas.unshift(["empleados", "registro_b2_empleados"]);
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
  function elegirEmpleado(ref) {
    if (!activo || typeof ref !== "string" || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(ref)) return;
    empleado = ref; ultimaFicha = undefined; seleccionRelacion = "";
    cambiarVista("ficha"); botones.get("ficha")?.focus?.();
  }
  const cambiarEmpleado = (ref) => { if (ref !== "" && (typeof ref !== "string" || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(ref))) throw new TypeError("referencia de empleado no válida"); empleado = ref; ultimaFicha = undefined; seleccionRelacion = ""; if (vista === "ficha") consultar(); else if (vista === "actos") cambiarVista("actos"); };
  cambiarVista(empleado || !botones.has("empleados") ? "ficha" : "empleados");
  return Object.freeze({ desmontar, cambiarEmpleado });
}
