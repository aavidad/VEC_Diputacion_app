import { crearClienteCopias, ErrorCopias } from "./cliente-http.js?v=20261001-cs09-copias-ux-v2";
import { TEXTOS_COPIAS, traducirCopias as t, traducirCodigo as tc } from "./i18n.js?v=20261001-cs09-copias-ux-v2";
import { icono } from "../../comun/iconos-vec.js";
import { soporte } from "./vista-soporte.js?v=20261001-cs09-copias-ux-v2";
import { pintarConfiguracion } from "./vista-configuracion.js?v=20261001-cs09-copias-ux-v2";
import { pintarRecuperacion } from "./vista-recuperacion.js?v=20261001-cs09-copias-ux-v2";

let secuencia = 0;
/** Subpágina de la superficie ADMIN; el shell y su frontera permanecen en su propietario. */
export function montarVistaCopias({ raiz, cliente = crearClienteCopias(), anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.ownerDocument?.createElement) throw new ErrorCopias();
  const d = raiz.ownerDocument, s = soporte(d), { crear, boton, panel, dato, chip, fecha, numero } = s;
  const id = "copias-admin-" + ++secuencia;
  const iconoSVG = clave => icono(clave === "valida" ? "correcto" : clave === "no_valida" ? "alerta" : "pendiente");
  const estado = { activa: true, tab: "catalogo", capacidades: {}, pagina: null, seleccion: null, filtro: "", cursores: [undefined], indice: 0, busy: false, recibo: null, lanzamiento: null, configuracion: null, propuestas: null, opciones: null };
  const peticiones = new Map();
  const seccion = crear("section", "", "copias-admin"), cabecera = crear("header", "", "copias-cabecera");
  const titulos = crear("div"), titulo = crear("h2", t("titulo"));
  titulos.append(crear("p", t("miga"), "miga-pan"), titulo);
  const ayuda = crear("details", "", "copias-ayuda"), resumen = crear("summary", "?", "boton-secundario");
  resumen.setAttribute("aria-label", t("ayuda")); ayuda.append(resumen, crear("div", t("ayuda_texto")));
  cabecera.append(titulos, ayuda);
  const nav = crear("nav", "", "copias-pestanas"); nav.setAttribute("aria-label", t("titulo"));
  const aviso = crear("div", "", "copias-aviso"); aviso.setAttribute("role", "status"); aviso.setAttribute("aria-live", "polite");
  const resultado = crear("div", "", "copias-resultado"), cuerpo = crear("div", "", "copias-trabajo");
  seccion.append(cabecera, nav, aviso, resultado, cuerpo); raiz.replaceChildren(seccion);
  seccion.lang = TEXTOS_COPIAS.idioma;

  function informar(clave, error = false) {
    aviso.textContent = error ? tc("errores", clave) : t(clave);
    aviso.setAttribute("role", error ? "alert" : "status");
    aviso.dataset.tipo = error ? "error" : "info";
    anunciar(aviso.textContent, error ? "error" : "info");
  }
  function recibido(recibo) {
    estado.recibo = recibo; aviso.textContent = ""; resultado.replaceChildren();
    const p = crear("details", "", "panel copias-recibo"), c = crear("div", "", "cuerpo-panel"), dl = crear("dl", "", "copias-datos");
    const cabecera = crear("summary", t("recibo"), "cabecera-panel"); p.append(cabecera, c);
    dato(dl, "recibo_fecha", fecha(recibo.registrado_en)); dato(dl, "recibo_estado", tc("estados", recibo.estado));
    const detalle = crear("details"), resumen = crear("summary", t("tecnico"));
    detalle.append(resumen, crear("p", recibo.recibo_ref), crear("p", recibo.operacion_ref)); c.append(dl, detalle); resultado.append(p);
  }
  async function consultar(nombre, fn, aplicar) {
    peticiones.get(nombre)?.abort(); const control = new AbortController(); peticiones.set(nombre, control);
    try {
      const datos = await fn(control.signal);
      if (estado.activa && peticiones.get(nombre) === control && !control.signal.aborted) { aplicar(datos); return datos; }
    } catch (error) {
      if (error?.name !== "AbortError" && estado.activa && peticiones.get(nombre) === control) {
        if (nombre === "catalogo") estado.errorCatalogo = error?.codigo ?? "no_disponible";
        if (error?.codigo === "denegado") {
          estado.denegado = true;
          Object.assign(estado, { capacidades: {}, pagina: null, seleccion: null, configuracion: null, propuestas: null, opciones: null, recibo: null });
          resultado.replaceChildren();
        }
        informar(error?.codigo ?? "no_disponible", true); pintar();
      }
    } finally { if (peticiones.get(nombre) === control) peticiones.delete(nombre); }
  }
  async function operar(fn, aplicar) {
    if (estado.busy) return;
    estado.busy = true; informar("guardando"); pintar();
    const control = new AbortController(); peticiones.set("efecto", control);
    try {
      const datos = await fn(control.signal);
      if (estado.activa && !control.signal.aborted) { aviso.textContent = ""; aplicar(datos); }
    } catch (error) { if (error?.name !== "AbortError" && estado.activa) informar(error?.codigo ?? "no_disponible", true); }
    finally { if (estado.activa) { estado.busy = false; pintar(); } peticiones.delete("efecto"); }
  }
  function pintarNav() {
    if (!nav.children.length) for (const clave of ["catalogo", "calendario", "restauracion"]) {
      const b = boton(clave, () => cambiarTab(clave));
      b.dataset.copiasTab = clave; nav.append(b);
    }
    for (const b of nav.children) {
      if (estado.tab === b.dataset.copiasTab) b.setAttribute("aria-current", "page"); else b.removeAttribute("aria-current");
      b.disabled = estado.busy;
    }
  }
  function cambiarTab(clave) {
    estado.tab = clave; estado.lanzamiento = null; pintar(); cuerpo.scrollTop = 0;
    if (clave === "calendario") cargarConfiguracion();
    if (clave === "restauracion") cargarRestauracion();
    cuerpo.querySelector("h3")?.focus();
  }
  async function cargarCatalogo(cursor) {
    informar("cargando"); cuerpo.setAttribute("aria-busy", "true");
    await consultar("catalogo", signal => cliente.listar({ signal, cursor }), pagina => { estado.pagina = pagina; estado.errorCatalogo = null; if (!estado.denegado) aviso.textContent = ""; pintar(); });
    if (estado.activa) cuerpo.removeAttribute("aria-busy");
  }
  async function cargarConfiguracion() {
    if (typeof cliente.configuracion !== "function") return;
    informar("cargando");
    await consultar("configuracion", signal => cliente.configuracion({ signal }), config => { estado.configuracion = config; aviso.textContent = ""; pintar(); });
  }
  async function cargarRestauracion() {
    if (typeof cliente.propuestas === "function") await consultar("propuestas", signal => cliente.propuestas({ signal }), p => { estado.propuestas = p; pintar(); });
    if (typeof cliente.opcionesRestauracion === "function") await consultar("opciones", signal => cliente.opcionesRestauracion({ signal }), o => { estado.opciones = o; pintar(); });
  }
  function pintarDetalle() {
    const { p, h, c } = panel("detalle_titulo"); h.append(boton("cerrar_detalle", () => { peticiones.get("detalle")?.abort(); estado.seleccion = null; pintar(); }));
    p.classList.add("copias-detalle");
    const copia = estado.seleccion;
    if (!copia) { c.append(crear("p", t("seleccionar"))); return p; }
    const dl = crear("dl", "", "copias-datos");
    dato(dl, "release", copia.release_ref); dato(dl, "retencion_hasta", fecha(copia.retencion_hasta));
    c.append(dl);
    const razones = TEXTOS_COPIAS.seccion("compatibilidad");
    c.append(crear("h4", t("compatibilidad")), chip(copia.compatibilidad.estado), crear("p", razones[copia.compatibilidad.clave_i18n] ?? razones[copia.compatibilidad.estado] ?? t("no_dato"), "copias-nota"));
    if (copia.componentes?.length) {
      const lista = crear("ul", "", "copias-componentes");
      for (const componente of copia.componentes) { const li = crear("li"); li.append(crear("span", tc("componentes", componente.componente)), chip(componente.estado)); lista.append(li); }
      c.append(crear("h4", t("componentes")), lista);
    }
    const tecnico = crear("details"), refs = crear("dl", "", "copias-datos");
    dato(refs, "referencia", copia.copia_ref); if (copia.huella_sha256) dato(refs, "huella", copia.huella_sha256);
    tecnico.append(crear("summary", t("tecnico")), refs); c.append(tecnico);
    const puede = estado.capacidades.proponer && copia.estado === "valida" && copia.compatibilidad.estado === "compatible";
    c.append(boton("proponer", () => cambiarTab("restauracion"), { clase: "boton-peligro", disabled: !puede || estado.busy }));
    if (!puede) c.append(crear("p", t(estado.capacidades.proponer === false ? "sin_permiso" : "copia_no_recuperable"), "copias-nota"));
    return p;
  }
  function pintarCatalogo() {
    const grid = crear("div", "", "copias-rejilla"), zona = crear("div", "", "copias-lista");
    const pagina = estado.pagina;
    const kpis = crear("div", "", "rejilla-kpi copias-kpi");
    for (const [clave, variantes] of [["valida", "exito"], ["no_valida", "peligro"], ["verificando", "advertencia"]]) {
      const kpi = crear("article", "", "tarjeta-kpi kpi--" + variantes), icono = crear("span", "", "icono-kpi"), contenido = crear("div");
      icono.setAttribute("aria-hidden", "true"); icono.innerHTML = iconoSVG(clave);
      contenido.append(crear("span", t(clave), "etiqueta-kpi"), crear("strong", pagina ? numero(pagina.copias.filter(c => c.estado === clave).length) : t("no_dato"), "valor-kpi"), crear("span", t("en_pagina"), "copias-nota"));
      kpi.append(icono, contenido); kpis.append(kpi);
    }
    const { p, h, c } = panel("catalogo");
    const puedeLanzar = estado.capacidades.lanzar && Number.isSafeInteger(pagina?.version);
    h.append(boton("lanzar", () => { estado.lanzamiento = { operacion_ref: "operacion:" + globalThis.crypto.randomUUID(), version_esperada: pagina.version, tipo: "completa" }; pintar(); cuerpo.querySelector("button")?.focus(); }, { clase: "boton-primario", disabled: !puedeLanzar || estado.busy }), boton("actualizar", () => cargarCatalogo(estado.cursores[estado.indice]), { disabled: estado.busy }));
    if (!puedeLanzar) c.append(crear("p", t(estado.capacidades.lanzar === false ? "sin_permiso" : "fuente_pendiente"), "copias-nota"));
    if (!pagina) c.append(crear("p", estado.errorCatalogo ? tc("errores", estado.errorCatalogo) : t("cargando")));
    else if (!pagina.copias.length) c.append(crear("p", t("vacio")));
    else {
      const label = crear("label", t("filtro"), "copias-filtro"), filtro = crear("select");
      filtro.setAttribute("aria-label", t("filtro"));
      for (const clave of ["", "valida", "no_valida", "verificando"]) { const o = crear("option", clave ? t(clave) : t("todas")); o.value = clave; filtro.append(o); }
      filtro.value = estado.filtro; filtro.addEventListener("change", () => { estado.filtro = filtro.value; pintar(); cuerpo.querySelector("select")?.focus(); }); label.append(filtro); c.append(label);
      const marco = crear("div", "", "tabla-contenedor copias-tabla"); marco.tabIndex = 0; marco.setAttribute("aria-label", t("catalogo"));
      const tabla = crear("table", "", "tabla-datos"), head = crear("thead"), tr = crear("tr"), body = crear("tbody");
      tabla.append(crear("caption", t("catalogo")));
      for (const clave of ["fecha", "tamano", "estado", "compatibilidad", "acciones"]) { const th = crear("th", t(clave)); th.scope = "col"; tr.append(th); }
      head.append(tr);
      for (const copia of pagina.copias.filter(c => !estado.filtro || c.estado === estado.filtro)) {
        const fila = crear("tr"); fila.setAttribute("aria-selected", String(estado.seleccion?.copia_ref === copia.copia_ref));
        fila.append(crear("td", fecha(copia.iniciada_en)), crear("td", copia.tamano_bytes === undefined ? t("no_dato") : t("tamano_bytes", { cuenta: numero(copia.tamano_bytes) })));
        fila.children[1].className = "columna-numero";
        const e = crear("td"), compat = crear("td"), acciones = crear("td"); e.append(chip(copia.estado)); compat.append(chip(copia.compatibilidad.estado));
        const abrir = boton("detalle", () => {
          const origen = d.activeElement;
          consultar("detalle", signal => cliente.detalle(copia.copia_ref, { signal }), detalle => {
            const enfocar = d.activeElement === origen; estado.seleccion = detalle; pintar();
            if (enfocar) cuerpo.querySelector(".copias-detalle h3")?.focus();
          });
        }); abrir.dataset.copiasFoco = "detalle:" + copia.copia_ref; acciones.append(abrir);
        fila.append(e, compat, acciones); body.append(fila);
      }
      tabla.append(head, body); marco.append(tabla); c.append(marco);
      const paginas = crear("div", "", "paginacion-marco");
      paginas.append(boton("anterior", () => { estado.indice--; cargarCatalogo(estado.cursores[estado.indice]); }, { disabled: estado.indice === 0 || estado.busy }), boton("siguiente", () => { estado.cursores[++estado.indice] = pagina.cursor_siguiente; cargarCatalogo(pagina.cursor_siguiente); }, { disabled: !pagina.cursor_siguiente || estado.busy })); c.append(paginas);
    }
    zona.append(kpis, p); grid.append(zona, pintarDetalle()); cuerpo.append(grid);
    if (estado.lanzamiento) {
      const { p: revision, c: contenido } = panel("revisar_copia"); contenido.append(crear("p", t("copia_efecto")), boton("confirmar_copia", () => operar(signal => cliente.lanzar(estado.lanzamiento, { signal }), recibo => { recibido(recibo); estado.lanzamiento = null; cargarCatalogo(estado.cursores[estado.indice]); }), { clase: "boton-primario", disabled: estado.busy }), boton("cancelar", () => { estado.lanzamiento = null; pintar(); }, { disabled: estado.busy })); cuerpo.replaceChildren(revision);
    }
  }
  function pintar() {
    if (!estado.activa) return;
    const anterior = d.activeElement;
    estado.propuestasAbiertas ??= new Set();
    for (const n of cuerpo.querySelectorAll("details[data-propuesta-ref]")) {
      if (n.open) estado.propuestasAbiertas.add(n.dataset.propuestaRef); else estado.propuestasAbiertas.delete(n.dataset.propuestaRef);
    }
    const foco = cuerpo.contains(anterior) ? { clave: anterior.dataset?.copiasFoco, nombre: anterior.name, tag: anterior.tagName, texto: anterior.textContent } : null;
    pintarNav(); cuerpo.replaceChildren();
    if (estado.capacidades.consultar !== true) {
      cuerpo.append(crear("p", estado.denegado || estado.capacidades.consultar === false ? t("sin_permiso") : t("cargando")));
      cuerpo.append(boton("actualizar", () => { cargarCapacidades(); cargarCatalogo(estado.cursores[estado.indice]); }));
      if (foco && !anterior.isConnected && d.activeElement === d.body) cuerpo.querySelector("button")?.focus();
      return;
    }
    if (estado.tab === "catalogo") pintarCatalogo();
    else if (estado.tab === "calendario") pintarConfiguracion({ cuerpo, estado, cliente, s, operar, recibido, pintar, cargarConfiguracion });
    else pintarRecuperacion({ cuerpo, estado, cliente, s, operar, recibido, pintar, cargarRestauracion });
    for (const h of cuerpo.querySelectorAll("h3")) h.tabIndex = -1;
    if (foco && !anterior.isConnected && (d.activeElement === d.body || d.activeElement === anterior)) {
      const elementos = [...cuerpo.querySelectorAll("button,input,select,summary,h3,h4")];
      const destino = elementos.find(n => !n.disabled && (foco.clave ? n.dataset.copiasFoco === foco.clave
        : foco.nombre ? n.name === foco.nombre && n.tagName === foco.tag : n.tagName === foco.tag && n.textContent === foco.texto));
      (destino ?? cuerpo.querySelector("h3"))?.focus();
    }
  }
  function cargarCapacidades() {
    return consultar("capacidades", signal => cliente.capacidades({ signal }), capacidades => { estado.capacidades = capacidades; estado.denegado = false; pintar(); });
  }
  pintar();
  cargarCapacidades();
  cargarCatalogo();
  const desmontar = () => { estado.activa = false; for (const c of peticiones.values()) c.abort(); peticiones.clear(); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
