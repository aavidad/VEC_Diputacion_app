import { validarSelectorFicha, validarLecturaFicha } from "./contrato.js?v=20261001-s1-ficha-v1";

/** Consumidor de lectura: el montaje aporta el lector y el catálogo común. */
export function montarFichaConvocatoria(contenedor, { lector, textos, alVolver } = {}) {
  if (!contenedor?.ownerDocument || typeof lector?.consultarExacta !== "function" || typeof textos?.traducir !== "function") {
    throw new TypeError("montaje_ficha_invalido");
  }
  const doc = contenedor.ownerDocument, t = (clave, vars) => textos.traducir(`ficha.${clave}`, vars);
  let turno = 0, controlador, selectorActual, cerrada = false;
  const nodo = (tag, texto, clase) => {
    const n = doc.createElement(tag); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n;
  };
  const raiz = nodo("article", undefined, "seleccion-ficha"); raiz.lang = textos.idioma;
  raiz.dataset.fichaEstado = "inicio";
  const cabecera = nodo("header", undefined, "cabecera-panel"); cabecera.append(nodo("h2", t("titulo")));
  const acciones = nodo("div", undefined, "seleccion-ficha-acciones");
  const ayuda = nodo("details", undefined, "seleccion-ficha-ayuda"), resumenAyuda = nodo("summary", t("ayuda"));
  ayuda.append(resumenAyuda, nodo("p", t("ayuda_texto")));
  if (typeof alVolver === "function") {
    const volver = nodo("button", t("volver"), "boton-secundario"); volver.type = "button"; volver.addEventListener("click", alVolver); acciones.append(volver);
  }
  const actualizar = nodo("button", t("actualizar"), "boton-secundario"); actualizar.type = "button"; actualizar.hidden = true;
  actualizar.addEventListener("click", () => { if (selectorActual) void consultar(selectorActual); });
  acciones.append(actualizar, ayuda); cabecera.append(acciones);
  const estado = nodo("p", t("inicio"), "seleccion-ficha-estado"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite"); estado.tabIndex = -1;
  const contenido = nodo("div", undefined, "seleccion-ficha-contenido"); contenedor.replaceChildren(raiz); raiz.append(cabecera, estado, contenido);

  function par(lista, clave, valor) { lista.append(nodo("dt", t(clave)), nodo("dd", valor)); }
  function panel(clave) {
    const p = nodo("section", undefined, "panel"), h = nodo("div", undefined, "cabecera-panel"), cuerpo = nodo("div", undefined, "cuerpo-panel");
    h.append(nodo("h3", t(clave))); p.append(h, cuerpo); contenido.append(p); return cuerpo;
  }
  function detalle(padre, filas) {
    const d = nodo("details"), lista = nodo("dl", undefined, "seleccion-ficha-datos");
    d.append(nodo("summary", t("detalle")), lista); for (const [clave, valor] of filas) par(lista, clave, valor); padre.append(d);
  }
  function listo(lectura) {
    const f = lectura.ficha, e = lectura.evidencia;
    const principal = panel("solo_lectura"), resumen = nodo("dl", undefined, "seleccion-ficha-datos seleccion-ficha-resumen");
    par(resumen, "version", textos.numero(f.secuencia)); par(resumen, "revision", textos.numero(f.revision));
    par(resumen, "consultada", textos.fecha(e.consultada_en, { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "Europe/Madrid" }));
    par(resumen, "integridad", f.huella_version_sha256); principal.append(resumen);
    detalle(principal, [["referencia", f.convocatoria_id], ["fuente", f.fuente_ref], ["flujo", f.flujo_proceso.id],
      ["version", textos.numero(f.flujo_proceso.version)], ["huella", f.flujo_proceso.huella_contenido_sha256],
      ["baremo", f.reglas_baremacion.id], ["version", textos.numero(f.reglas_baremacion.version)], ["huella", f.reglas_baremacion.huella_contenido_sha256],
      ["recibo", e.recibo_ref], ["decision", e.decision_ref], ["consumo", e.consumo_huella_sha256], ["auditoria", e.auditoria_ref], ["correlacion", e.correlacion_ref]]);
    const bases = panel("bases"), listaBases = nodo("ol", undefined, "lista-trazabilidad");
    f.bases.forEach((b, i) => {
      const item = nodo("li", undefined, "elemento-trazabilidad"); item.append(nodo("h4", t("base", { numero: textos.numero(i + 1) })));
      const datos = nodo("dl", undefined, "seleccion-ficha-datos"); par(datos, "version_documento", textos.numero(b.version_documento)); par(datos, "huella", b.huella_contenido_sha256); item.append(datos);
      detalle(item, [["rol", b.rol], ["publicacion", b.publicacion_ref], ["documento", b.documento_ref], ["representacion", b.representacion_ref], ["firma", b.firma_validada_ref], ["custodia", b.recibo_custodia_ref]]);
      listaBases.append(item);
    }); bases.append(listaBases);
    const requisitos = panel("requisitos");
    if (!f.requisitos.length) requisitos.append(nodo("p", t("sin_requisitos")));
    else {
      const lista = nodo("ol", undefined, "lista-trazabilidad");
      [...f.requisitos].sort((a, b) => a.orden - b.orden || a.referencia.localeCompare(b.referencia)).forEach((r) => {
        const item = nodo("li", undefined, "elemento-trazabilidad"); item.append(nodo("h4", r.titulo), nodo("span", t(r.obligatorio ? "obligatorio" : "no_obligatorio"), "seleccion-ficha-obligatorio"));
        if (r.descripcion) item.append(nodo("p", r.descripcion)); lista.append(item);
      }); requisitos.append(lista);
    }
    panel("fases").append(nodo("p", t("fases_pendientes")));
    panel("cobertura").append(nodo("p", t("cobertura_pendiente")));
    panel("siguiente").append(nodo("p", t("siguiente_pendiente")));
  }
  function mostrar(nuevoEstado, lectura) {
    raiz.dataset.fichaEstado = nuevoEstado; raiz.setAttribute("aria-busy", String(nuevoEstado === "cargando"));
    estado.setAttribute("role", ["error", "denegado", "invalida"].includes(nuevoEstado) ? "alert" : "status");
    contenido.replaceChildren(); actualizar.hidden = nuevoEstado !== "listo"; actualizar.disabled = nuevoEstado === "cargando";
    if (nuevoEstado === "listo") { estado.textContent = ""; estado.hidden = true; listo(lectura); }
    else {
      estado.hidden = false; estado.textContent = t(nuevoEstado);
      if (nuevoEstado === "error") {
        const reintentar = nodo("button", t("reintentar"), "boton-primario"); reintentar.type = "button";
        reintentar.addEventListener("click", () => void consultar(selectorActual)); contenido.append(reintentar);
      }
    }
  }
  async function consultar(entrada) {
    if (cerrada) return;
    const miTurno = ++turno; controlador?.abort(); controlador = new AbortController(); const miControlador = controlador;
    const trasladarFoco = raiz.contains(doc.activeElement) && doc.activeElement !== resumenAyuda;
    try {
      selectorActual = validarSelectorFicha(entrada); mostrar("cargando");
      if (trasladarFoco) estado.focus();
      const lectura = await lector.consultarExacta(selectorActual, { signal: miControlador.signal });
      if (cerrada || miTurno !== turno || miControlador.signal.aborted) return;
      validarLecturaFicha(lectura, selectorActual); mostrar("listo", lectura);
      if (trasladarFoco) actualizar.focus();
    } catch (error) {
      if (cerrada || miTurno !== turno || miControlador.signal.aborted) return;
      const estados = { acceso_denegado: "denegado", no_encontrada: "vacio", consulta_invalida: "invalida" };
      mostrar(estados[error?.codigo] ?? "error"); if (trasladarFoco) estado.focus();
    }
  }
  return Object.freeze({ consultar, desmontar() { if (cerrada) return; cerrada = true; ++turno; controlador?.abort(); raiz.remove(); } });
}
