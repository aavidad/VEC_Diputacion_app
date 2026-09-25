import { crearTraductorPersonal } from "./i18n.js?v=20260925-b2-sin-codigos-v1";

const PESTANAS = Object.freeze([
  ["ficha", "ficha_tab_ficha"], ["relaciones", "ficha_tab_relaciones"],
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
const ESTADOS = new Set(["disponible", "vacio", "no_configurado", "denegado"]);
const ETIQUETAS_ESTADO = Object.freeze({
  sin_consulta: "ficha_estado_sin_consulta", cargando: "ficha_estado_cargando",
  disponible: "ficha_estado_disponible", vacio: "ficha_estado_vacio",
  no_configurado: "ficha_estado_no_configurado", denegado: "ficha_estado_denegado",
  error: "ficha_estado_error",
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
  const contexto = nodo(d, "p");
  detalles.append(abrir, nodo(d, "p", t("ficha_ayuda")), contexto);
  return { elemento: detalles, mostrar(clave) { detalles.open = false; contexto.textContent = clave ? t(clave) : ""; } };
}
function accesos(d, t, navegarModulo, destinosDisponibles) {
  const acciones = nodo(d, "div"); acciones.className = "acciones-fila personal-ficha-accesos";
  for (const [destino, etiqueta] of [["dietas", "ficha_ir_dietas"], ["cronos", "ficha_ir_cronos"]]) {
    const boton = nodo(d, "button", t(etiqueta)); boton.type = "button"; boton.dataset.personalFichaDestino = destino;
    // La disponibilidad de una ruta se inyecta por destino; no acredita permiso.
    const disponible = typeof navegarModulo === "function" && Object.hasOwn(destinosDisponibles, destino) && destinosDisponibles[destino] === true;
    boton.disabled = !disponible; boton.setAttribute("aria-disabled", String(!disponible));
    if (!disponible) { boton.title = t("ficha_navegacion_pendiente"); boton.setAttribute("aria-label", `${t(etiqueta)}. ${t("ficha_navegacion_pendiente")}`); }
    boton.addEventListener("click", () => { if (disponible) navegarModulo(destino); }); acciones.append(boton);
  }
  return acciones;
}
function portada(d, t, navegarModulo, destinosDisponibles, estados) {
  const bloques = nodo(d, "div"); bloques.className = "personal-ficha-bloques";
  for (const clave of Object.keys(BLOQUES)) {
    const ficha = nodo(d, "div"); ficha.className = "personal-ficha-bloque";
    ficha.dataset.personalFichaEstado = estados[clave];
    const estado = nodo(d, "span", t(ETIQUETAS_ESTADO[estados[clave]])); estado.className = "personal-ficha-estado";
    ficha.append(nodo(d, "strong", t(BLOQUES[clave].titulo)), estado); bloques.append(ficha);
  }
  const resumen = [mensaje(d, t("ficha_fuente_pendiente"))];
  if (Object.values(estados).every((estado) => estado === "no_configurado")) resumen.push(mensaje(d, t("ficha_sin_datos")));
  resumen.push(bloques);
  return [panel(d, t("ficha_resumen"), resumen, "personal-ficha-panel-ancho"),
    panel(d, t("ficha_accesos_titulo"), [accesos(d, t, navegarModulo, destinosDisponibles)], "personal-ficha-panel-ancho")];
}
function validarResultado(resultado, bloque) {
  if (!resultado || typeof resultado !== "object" || !ESTADOS.has(resultado.estado)) throw new TypeError("respuesta de ficha no válida");
  if (resultado.estado !== "disponible" && resultado.estado !== "vacio") return { estado: resultado.estado };
  if (typeof resultado.fuente !== "string" || !resultado.fuente.trim() || resultado.fuente.length > 160 ||
      typeof resultado.actualizado_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/u.test(resultado.actualizado_en) ||
      !Number.isFinite(Date.parse(resultado.actualizado_en)) ||
      !Array.isArray(resultado.items) || resultado.items.length > 50 ||
      (resultado.estado === "vacio" && resultado.items.length !== 0) || (resultado.estado === "disponible" && resultado.items.length === 0)) throw new TypeError("respuesta de ficha no válida");
  const columnas = BLOQUES[bloque].columnas.map(([campo]) => campo);
  const items = resultado.items.map((item) => {
    if (!item || typeof item !== "object") throw new TypeError("fila de ficha no válida");
    const visible = Object.fromEntries(columnas.map((campo) => {
      const valor = item[campo];
      if (valor !== undefined && (typeof valor !== "string" || valor.length > 240)) throw new TypeError("campo de ficha no válido");
      return [campo, valor ?? ""];
    }));
    if (!Object.values(visible).some((valor) => valor.trim())) throw new TypeError("fila de ficha sin datos visibles");
    return visible;
  });
  return { estado: resultado.estado, fuente: resultado.fuente, actualizado_en: resultado.actualizado_en, items };
}
function formatearFecha(iso) {
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(iso));
}
function mostrarCampo(campo, valor, t) {
  if (!valor) return t("ficha_no_consta");
  if (["desde", "hasta", "fecha"].includes(campo) && /^\d{4}-\d{2}-\d{2}$/u.test(valor)) {
    const fecha = new Date(`${valor}T12:00:00Z`);
    if (Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor) return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(fecha);
  }
  if (campo === "periodo" && /^\d{4}-\d{2}$/u.test(valor)) {
    const fecha = new Date(`${valor}-15T12:00:00Z`);
    if (Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 7) === valor) return new Intl.DateTimeFormat("es-ES", { month: "long", year: "numeric", timeZone: "Europe/Madrid" }).format(fecha);
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
function pintarBloque(d, principal, t, bloque, resultado) {
  const definicion = BLOQUES[bloque]; const piezas = [];
  if (resultado.estado === "disponible" || resultado.estado === "vacio") {
    const metadatos = nodo(d, "p", t("ficha_procedencia", { fuente: resultado.fuente, fecha: formatearFecha(resultado.actualizado_en) }));
    metadatos.className = "personal-ficha-procedencia"; piezas.push(metadatos);
    piezas.push(resultado.estado === "vacio" ? mensaje(d, t("ficha_vacio")) : tabla(d, t, bloque, resultado.items));
  } else {
    const clave = { cargando: "ficha_cargando", no_configurado: "ficha_no_configurado", denegado: "ficha_denegado", error: "ficha_error" }[resultado.estado];
    piezas.push(mensaje(d, t(clave), resultado.estado === "error" ? "alert" : "status"));
  }
  principal.replaceChildren(panel(d, t(definicion.titulo), piezas, "personal-ficha-panel-ancho"));
}

/**
 * Vista de consulta propia. `fuentes[clave].consultarPropios({signal})` debe ser un
 * cliente de servidor que resuelva la identidad y autorice los campos; la vista
 * nunca recibe ni envía referencias de persona o empleado. Bloques sin cliente
 * permanecen no configurados y se consultan solo al abrir su pestaña.
 */
export function montarVistaFichaIntegralPersonal({ raiz, anunciar = () => {}, registrarDesmontar, montarCatalogos, navegarModulo, destinosDisponibles = {}, fuentes = {} } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") ||
      (montarCatalogos !== undefined && typeof montarCatalogos !== "function") ||
      (navegarModulo !== undefined && typeof navegarModulo !== "function") || !destinosDisponibles || typeof destinosDisponibles !== "object" || Array.isArray(destinosDisponibles) ||
      !fuentes || typeof fuentes !== "object") throw new TypeError("vista ficha integral de Personal no disponible");
  const d = raiz.ownerDocument; if (!d?.createElement) throw new TypeError("documento ficha integral de Personal no disponible");
  const t = crearTraductorPersonal(); const contenedor = nodo(d, "section"); contenedor.className = "modulo-personal";
  contenedor.dataset.personalFichaIntegral = ""; raiz.append(contenedor);
  const estados = Object.fromEntries(Object.keys(BLOQUES).map((clave) => [clave,
    Object.hasOwn(fuentes, clave) && typeof fuentes[clave]?.consultarPropios === "function" ? "sin_consulta" : "no_configurado"]));
  let activa = true; let actual = "ficha"; let vuelo; let limpiarCatalogos; let secuencia = 0;
  const limpiar = () => { const fn = limpiarCatalogos; limpiarCatalogos = undefined; fn?.(); };
  const desmontar = () => { if (!activa) return; activa = false; secuencia += 1; vuelo?.abort(); limpiar(); contenedor.remove?.(); };
  registrarDesmontar?.(desmontar);
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-vista";
  const ayudaFicha = ayuda(d, t);
  cabecera.append(nodo(d, "p", t("ficha_sobrelinea")), nodo(d, "h2", t("ficha_titulo")), ayudaFicha.elemento);
  const tabs = nodo(d, "div"); tabs.className = "personal-ficha-pestanas"; tabs.setAttribute("role", "tablist"); tabs.setAttribute("aria-label", t("ficha_navegacion"));
  const principal = nodo(d, "div"); principal.className = "personal-ficha-principal"; principal.id = "personal-ficha-panel";
  principal.setAttribute("role", "tabpanel"); principal.setAttribute("tabindex", "0");
  const pintar = (clave) => {
    if (!activa) return;
    if (actual === "catalogos") limpiar();
    if (vuelo && Object.hasOwn(estados, actual) && estados[actual] === "cargando") estados[actual] = "sin_consulta";
    secuencia += 1; vuelo?.abort(); vuelo = undefined; actual = clave;
    ayudaFicha.mostrar(clave === "ficha" ? "ficha_accesos_ayuda" : BLOQUES[clave]?.ayuda);
    for (const [valor] of PESTANAS) {
      const tab = tabs.querySelector?.(`[data-personal-ficha-tab="${valor}"]`);
      tab?.setAttribute("aria-selected", String(valor === clave)); tab?.setAttribute("tabindex", valor === clave ? "0" : "-1");
    }
    principal.setAttribute("aria-labelledby", `personal-ficha-tab-${clave}`);
    if (clave === "ficha") { principal.replaceChildren(...portada(d, t, navegarModulo, destinosDisponibles, estados)); return; }
    if (clave === "catalogos") {
      const hueco = nodo(d, "div"); hueco.dataset.personalFichaCatalogos = "";
      principal.replaceChildren(panel(d, t("ficha_catalogos_titulo"), [nodo(d, "p", t("ficha_catalogos_completos")), hueco], "personal-ficha-panel-ancho"));
      if (!montarCatalogos) { hueco.append(mensaje(d, t("ficha_catalogos_pendiente"))); return; }
      const turno = secuencia;
      Promise.resolve().then(() => montarCatalogos({ raiz: hueco, anunciar, registrarDesmontar: (fn) => {
        if (typeof fn !== "function") throw new TypeError("limpieza de catálogos de Personal no válida");
        if (activa && actual === "catalogos" && turno === secuencia) limpiarCatalogos = fn; else fn();
      } })).then((montaje) => {
        if (!montaje?.desmontar) return;
        if (!activa || actual !== "catalogos" || turno !== secuencia) { montaje.desmontar(); return; }
        const previo = limpiarCatalogos; limpiarCatalogos = () => { montaje.desmontar(); previo?.(); };
      }).catch(() => { if (activa && actual === "catalogos" && turno === secuencia) hueco.replaceChildren(mensaje(d, t("ficha_catalogos_error"), "alert")); });
      return;
    }
    const consultar = Object.hasOwn(fuentes, clave) ? fuentes[clave]?.consultarPropios : undefined;
    if (typeof consultar !== "function") { pintarBloque(d, principal, t, clave, { estado: "no_configurado" }); return; }
    const turno = secuencia; const controlador = new AbortController(); vuelo = controlador;
    estados[clave] = "cargando";
    pintarBloque(d, principal, t, clave, { estado: "cargando" });
    Promise.resolve().then(() => consultar({ signal: controlador.signal })).then((resultado) => {
      if (!activa || actual !== clave || turno !== secuencia || controlador.signal.aborted) return;
      const validado = validarResultado(resultado, clave); estados[clave] = validado.estado;
      pintarBloque(d, principal, t, clave, validado);
    }).catch(() => {
      if (!activa || actual !== clave || turno !== secuencia || controlador.signal.aborted) return;
      estados[clave] = "error";
      pintarBloque(d, principal, t, clave, { estado: "error" }); anunciar(t("ficha_error"), "error");
    }).finally(() => { if (vuelo === controlador) vuelo = undefined; });
  };
  PESTANAS.forEach(([clave, texto], indice) => {
    const tab = nodo(d, "button", t(texto)); tab.type = "button"; tab.id = `personal-ficha-tab-${clave}`;
    tab.dataset.personalFichaTab = clave; tab.setAttribute("role", "tab"); tab.setAttribute("aria-controls", principal.id);
    tab.addEventListener("click", () => pintar(clave));
    tab.addEventListener("keydown", (evento) => {
      const salto = { ArrowRight: 1, ArrowLeft: -1, Home: -indice, End: PESTANAS.length - 1 - indice }[evento.key];
      if (salto === undefined) return; evento.preventDefault();
      const destino = (indice + salto + PESTANAS.length) % PESTANAS.length;
      pintar(PESTANAS[destino][0]); tabs.querySelector?.(`[data-personal-ficha-tab="${PESTANAS[destino][0]}"]`)?.focus?.();
    }); tabs.append(tab);
  });
  contenedor.append(cabecera, tabs, principal); pintar(actual); return Object.freeze({ desmontar });
}
