import { traducirConvocatoriasS1 } from "./portal-i18n-convocatorias.js?v=20260924-f2-web2";

/**
 * Consulta S1 de solo lectura. El montaje aporta funciones autorizadas; esta
 * vista no conoce rutas HTTP, credenciales, permisos ni datos de presentación.
 *
 * consultarLista({ signal }) -> [{ referencia, titulo, categoria, estado,
 *   version_actual, cierre_plazo }]
 * consultarDetalle(referencia, { signal }) -> { referencia, titulo, resumen,
 *   categoria, estado, identificador_publico, version_actual,
 *   versiones: [{ codigo, estado, publicada_en }],
 *   bases: { codigo, resumen, requisitos: [{ referencia, titulo, descripcion,
 *     obligatorio, hito_exigibilidad }], hitos: [{ titulo, fecha }],
 *     documentos: [{ titulo, referencia }] } }
 * La capa que consulta valida el contrato y la autorización antes de entregar
 * la proyección. Aquí se exige una forma mínima y se escapa cada dato visible.
 */
export function crearSuperficieConvocatoriasS1({
  contenedor, consultarLista, consultarDetalle, escaparHTML = escaparHTMLConvocatorias,
  traducir = traducirConvocatoriasS1,
} = {}) {
  const e = (valor) => escaparHTML(String(valor ?? ""));
  const t = (clave, variables) => traducir(clave, variables);
  let activo = false;
  let generacion = 0;
  let controlador = null;
  let estado = { carga: "no_configurado", convocatorias: [], detalle: null, seleccionada: "" };

  function fecha(valor) {
    if (!valor) return t("sin_fecha");
    if (!/^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}(?::\d{2}(?:\.\d+)?)?(?:Z|[+-]\d{2}:\d{2}))?$/.test(String(valor))) return t("sin_fecha");
    const instante = new Date(valor);
    if (Number.isNaN(instante.getTime())) return t("sin_fecha");
    return new Intl.DateTimeFormat("es-ES", {
      dateStyle: "medium", ...(String(valor).includes("T") ? { timeStyle: "short" } : {}),
      timeZone: "Europe/Madrid",
    }).format(instante);
  }

  function texto(valor) { return valor ? e(valor) : e(t("sin_dato")); }
  function lista(valor) { return Array.isArray(valor) ? valor : []; }
  function validaLista(valor) {
    return Array.isArray(valor) && valor.length <= 100
      && new Set(valor.map((item) => item?.referencia)).size === valor.length
      && valor.every((item) => item
      && typeof item.referencia === "string" && item.referencia.length > 0
      && typeof item.titulo === "string" && item.titulo.length > 0);
  }
  function validaDetalle(valor, referencia) {
    return valor && typeof valor === "object" && valor.referencia === referencia
      && typeof valor.titulo === "string" && valor.titulo.length > 0
      && valor.bases && typeof valor.bases === "object"
      && Array.isArray(valor.versiones) && Array.isArray(valor.bases.requisitos)
      && Array.isArray(valor.bases.hitos) && Array.isArray(valor.bases.documentos)
      && valor.versiones.length <= 100 && valor.bases.requisitos.length <= 256
      && valor.bases.hitos.length <= 100 && valor.bases.documentos.length <= 100
      && valor.versiones.every((item) => item && typeof item === "object")
      && valor.bases.requisitos.every((item) => item && typeof item === "object")
      && valor.bases.hitos.every((item) => item && typeof item === "object")
      && valor.bases.documentos.every((item) => item && typeof item === "object");
  }
  function aviso(carga) {
    const alerta = carga === "denegado" || carga === "error" || carga === "error_detalle";
    return `<section class="panel s1-estado" ${alerta ? 'role="alert"' : 'role="status"'} aria-live="polite" tabindex="-1"><div class="cabecera-panel"><h3>${e(t(`estado_${carga}_titulo`))}</h3></div><div class="cuerpo-panel"><p>${e(t(`estado_${carga}_detalle`))}</p></div></section>`;
  }
  function controlNoConectado(clave) {
    return `<button type="button" class="boton-secundario" disabled aria-disabled="true" title="${e(t("accion_no_conectada"))}">${e(t(clave))}</button>`;
  }
  function ficha(item) {
    const seleccionada = item.referencia === estado.seleccionada;
    return `<li class="s1-elemento${seleccionada ? " s1-elemento-activo" : ""}"><button type="button" data-s1-convocatoria="${e(item.referencia)}" aria-current="${seleccionada ? "true" : "false"}"><strong>${e(item.titulo)}</strong><span>${texto(item.categoria)}</span><span class="s1-meta">${e(t("version"))}: ${texto(item.version_actual)} · ${e(t("cierre"))}: ${e(fecha(item.cierre_plazo))}</span><span class="estado-chip info">${texto(item.estado)}</span></button></li>`;
  }
  const apartados = Object.freeze([
    ["resumen", "resumen"], ["bases", "bases_apartado"], ["versiones", "navegacion_versiones"],
    ["requisitos", "navegacion_requisitos"], ["hitos", "navegacion_hitos"], ["documentos", "navegacion_documentos"],
  ]);
  function navegacionDetalle() {
    return `<nav class="s1-navegacion" aria-label="${e(t("navegacion_detalle"))}">${apartados.map(([id, clave]) => `<button type="button" data-s1-seccion="${id}" aria-controls="s1-seccion-${id}">${e(t(clave))}</button>`).join("")}</nav>`;
  }
  function grupo(id, titulo, contenido, vacio) {
    return `<section class="panel" id="s1-seccion-${id}" tabindex="-1"><div class="cabecera-panel"><h3>${e(t(titulo))}</h3></div><div class="cuerpo-panel">${contenido || `<p class="s1-vacio">${e(t(vacio))}</p>`}</div></section>`;
  }
  function detalleHTML(detalle) {
    if (!detalle) return aviso("sin_seleccion");
    const bases = detalle.bases;
    const versiones = lista(detalle.versiones).map((item) => `<li class="s1-fila"><strong>${texto(item.codigo)}</strong><span>${texto(item.estado)}</span><time>${e(fecha(item.publicada_en))}</time></li>`).join("");
    const requisitos = lista(bases.requisitos).map((item) => `<li class="s1-fila s1-requisito"><div><strong>${texto(item.titulo)}</strong><p>${texto(item.descripcion)}</p></div><div><span class="estado-chip ${item.obligatorio === true ? "aviso" : "info"}">${e(t(item.obligatorio === true ? "obligatorio" : "no_obligatorio"))}</span><small>${e(t("hito_exigibilidad"))}: ${texto(item.hito_exigibilidad)}</small></div></li>`).join("");
    const hitos = lista(bases.hitos).map((item) => `<li class="s1-fila"><strong>${texto(item.titulo)}</strong><time>${e(fecha(item.fecha))}</time></li>`).join("");
    const documentos = lista(bases.documentos).map((item) => `<li class="s1-fila"><strong>${texto(item.titulo)}</strong><span>${texto(item.referencia)}</span></li>`).join("");
    return `<div class="s1-detalle">
      ${navegacionDetalle()}
      <section class="panel s1-resumen" id="s1-seccion-resumen" tabindex="-1"><div class="cabecera-panel"><div><h3>${e(detalle.titulo)}</h3><p>${texto(detalle.categoria)}</p></div><span class="estado-chip info">${texto(detalle.estado)}</span></div><div class="cuerpo-panel"><p>${texto(detalle.resumen)}</p><dl class="resumen-expediente"><div class="fila-resumen"><dt>${e(t("version_actual"))}</dt><dd>${texto(detalle.version_actual)}</dd></div><div class="fila-resumen"><dt>${e(t("identificador_publico"))}</dt><dd>${texto(detalle.identificador_publico)}</dd></div></dl></div></section>
      ${grupo("bases", "bases_apartado", `<dl class="s1-bases-datos"><div><dt>${e(t("version"))}</dt><dd>${texto(bases.codigo)}</dd></div></dl>${bases.resumen ? `<p>${texto(bases.resumen)}</p>` : `<p class="s1-vacio">${e(t("bases_vacias"))}</p>`}`, "bases_vacias")}
      ${grupo("versiones", "versiones", versiones ? `<ul class="s1-lista">${versiones}</ul>` : "", "versiones_vacias")}
      ${grupo("requisitos", "requisitos", requisitos ? `<ul class="s1-lista">${requisitos}</ul>` : "", "requisitos_vacios")}
      ${grupo("hitos", "hitos", hitos ? `<ul class="s1-lista">${hitos}</ul>` : "", "hitos_vacios")}
      ${grupo("documentos", "documentos", documentos ? `<ul class="s1-lista">${documentos}</ul>` : "", "documentos_vacios")}
      <section class="panel"><div class="cabecera-panel"><h3>${e(t("acciones"))}</h3></div><div class="cuerpo-panel s1-acciones">${controlNoConectado("editar_bases")}${controlNoConectado("enviar_firma")}${controlNoConectado("publicar")}</div></section>
    </div>`;
  }
  function renderizar() {
    const cabecera = `<header class="s1-cabecera"><div><p class="sobrelinea">${e(t("sobrelinea"))}</p><h2>${e(t("titulo"))}</h2><p>${e(t("descripcion"))}</p></div><details class="s1-ayuda"><summary aria-label="${e(t("ayuda_aria"))}">?</summary><p>${e(t("ayuda_detalle"))}</p></details></header>`;
    if (estado.carga !== "disponible") return `<div class="consulta-convocatorias-s1">${cabecera}${aviso(estado.carga)}</div>`;
    if (estado.convocatorias.length === 0) return `<div class="consulta-convocatorias-s1">${cabecera}${aviso("vacio")}</div>`;
    return `<div class="consulta-convocatorias-s1">${cabecera}<div class="s1-rejilla"><section class="panel s1-listado"><div class="cabecera-panel"><div><h3>${e(t("listado"))}</h3><p>${e(t("cantidad", { numero: new Intl.NumberFormat("es-ES").format(estado.convocatorias.length) }))}</p></div><span class="estado-chip info">${e(t("solo_lectura"))}</span></div><ul class="s1-lista">${estado.convocatorias.map(ficha).join("")}</ul></section>${estado.cargaDetalle === "cargando" ? aviso("cargando_detalle") : estado.cargaDetalle === "no_configurado" ? aviso("no_configurado") : estado.cargaDetalle === "denegado" ? aviso("denegado") : estado.cargaDetalle === "error" ? aviso("error_detalle") : detalleHTML(estado.detalle)}</div></div>`;
  }
  function pintar() { if (activo && contenedor) contenedor.innerHTML = renderizar(); }
  async function cargarDetalle(referencia, { enfocarDetalle = false } = {}) {
    if (!activo || !estado.convocatorias.some((item) => item.referencia === referencia)) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++generacion;
    estado = { ...estado, seleccionada: referencia, detalle: null, cargaDetalle: "cargando" };
    pintar();
    if (enfocarDetalle) contenedor.querySelector?.(".s1-rejilla > .s1-estado")?.focus();
    if (typeof consultarDetalle !== "function") {
      estado = { ...estado, cargaDetalle: "no_configurado" };
      pintar();
      if (enfocarDetalle) contenedor.querySelector?.(".s1-rejilla > .s1-estado")?.focus();
      return;
    }
    try {
      const detalle = await consultarDetalle(referencia, { signal: controlador.signal });
      if (!activo || actual !== generacion) return;
      if (!validaDetalle(detalle, referencia)) throw new Error("contrato de detalle inválido");
      estado = { ...estado, detalle, cargaDetalle: "disponible" };
    } catch (error) {
      if (!activo || actual !== generacion) return;
      estado = { ...estado, detalle: null, cargaDetalle: error?.status === 401 || error?.status === 403 ? "denegado" : "error" };
    }
    pintar();
    if (enfocarDetalle) {
      contenedor.querySelector?.(estado.cargaDetalle === "disponible" ? "#s1-seccion-resumen" : ".s1-rejilla > .s1-estado")?.focus();
    }
  }
  function seleccionar(evento) {
    const boton = evento.target?.closest?.("[data-s1-convocatoria]");
    if (boton && contenedor?.contains(boton)) {
      void cargarDetalle(boton.dataset.s1Convocatoria, { enfocarDetalle: true });
      return;
    }
    const enlace = evento.target?.closest?.("[data-s1-seccion]");
    if (!enlace || !contenedor?.contains(enlace)) return;
    const seccion = apartados.find(([id]) => id === enlace.dataset.s1Seccion);
    if (!seccion) return;
    const destino = contenedor.querySelector?.(`#s1-seccion-${seccion[0]}`);
    destino?.scrollIntoView?.({ block: "start", behavior: "instant" });
    destino?.focus?.({ preventScroll: true });
  }
  async function montar() {
    if (!contenedor || typeof contenedor.addEventListener !== "function") throw new TypeError("contenedor S1 inválido");
    if (activo) return;
    activo = true;
    contenedor.addEventListener("click", seleccionar);
    if (typeof consultarLista !== "function") { pintar(); return; }
    estado = { carga: "cargando", convocatorias: [], detalle: null, seleccionada: "" };
    pintar();
    controlador = new AbortController();
    const actual = ++generacion;
    try {
      const convocatorias = await consultarLista({ signal: controlador.signal });
      if (!activo || actual !== generacion) return;
      if (!validaLista(convocatorias)) throw new Error("contrato de lista inválido");
      estado = { carga: "disponible", convocatorias, detalle: null, seleccionada: "" };
      pintar();
      if (convocatorias.length > 0) await cargarDetalle(convocatorias[0].referencia);
    } catch (error) {
      if (!activo || actual !== generacion) return;
      estado = { carga: error?.status === 401 || error?.status === 403 ? "denegado" : "error", convocatorias: [], detalle: null, seleccionada: "" };
      pintar();
    }
  }
  function desmontar() {
    if (!activo) return;
    activo = false;
    ++generacion;
    controlador?.abort();
    contenedor.removeEventListener("click", seleccionar);
    contenedor.replaceChildren();
  }
  return Object.freeze({ montar, desmontar, renderizar, cargarDetalle });
}

function escaparHTMLConvocatorias(valor) {
  return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
