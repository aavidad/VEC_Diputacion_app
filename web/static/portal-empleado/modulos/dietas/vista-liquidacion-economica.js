import { cargarTextos } from "../../../comun/textos.js";
import { ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20261007-pantallas-textos-final-v1";

const nodo = (doc, tag, valor = "") => { const n = doc.createElement(tag); n.textContent = valor; return n; };
const centimos = (valor) => Number.isSafeInteger(valor) && valor >= 0 && valor <= 1000000000000;
const codigo = /^[a-z][a-z0-9_]{2,79}$/u;
const motivos = ["revision_documental", "revision_justificante", "gasto_no_admitido"];
const MOTIVO_DIFERENCIA = "diferencia_tarifa_publicada";

export function montarVistaLiquidacionEconomica(contenedor, { cliente, referencia, version, generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  registrarDesmontar, alRegistrar = () => {}, alCambiarBloqueo = () => {}, cargarTextosImpl = cargarTextos,
  entornoDescarga = globalThis, modoConsulta = false } = {}) {
  if (!contenedor?.ownerDocument || !cliente?.leer || (!modoConsulta && (!cliente?.propuesta || !cliente?.registrar))
    || !/^dco_[A-Za-z0-9_-]{22,128}$/u.test(referencia)
    || (modoConsulta ? version !== "actual" && (!Number.isSafeInteger(version) || version < 2)
      : !Number.isSafeInteger(version) || version < 1)) throw new TypeError("vista de liquidación no disponible");
  const doc = contenedor.ownerDocument; const raiz = nodo(doc, "section"); raiz.dataset.dietasLiquidacion = ""; raiz.className = "panel";
  const titulo = nodo(doc, "h4"); const estado = nodo(doc, "p"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite"); estado.setAttribute("tabindex", "-1"); estado.dataset.dietasLiquidacionEstado = "";
  const contenido = nodo(doc, "div"); contenido.className = "cuerpo-panel"; raiz.append(titulo, estado, contenido); contenedor.append(raiz);
  let viva = true; let controlador; let generacion = 0; let textos; let propuesta; let revisiones = []; let decisiones = []; let revisionVisible = false; let revisionConfirmada; let pendiente; let liquidacion; let denegada = false;
  let descargaControlador; let urlDocumento; let revocacion; let errorCampo;
  const liberarURL = () => { if (revocacion) clearTimeout(revocacion); revocacion = undefined;
    if (urlDocumento) entornoDescarga.URL?.revokeObjectURL?.(urlDocumento); urlDocumento = undefined; };
  const t = (clave, variables) => textos?.traducir(`liquidacion_economica.${clave}`, variables) ?? "";
  const euro = (n) => new Intl.NumberFormat(textos.localizacion, { style: "currency", currency: "EUR" }).format(n / 100);
  const fecha = (valor) => new Intl.DateTimeFormat(textos.localizacion, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(new Date(valor));
  const fechaCivil = (valor) => new Intl.DateTimeFormat(textos.localizacion, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T00:00:00Z`));
  const ponerEstado = (clave) => { estado.textContent = t(clave); };
  const limpiarErrorCampo = () => {
    if (!errorCampo) return;
    errorCampo.control.removeAttribute("aria-invalid"); errorCampo.control.removeAttribute("aria-describedby");
    errorCampo.mensaje.remove(); errorCampo = undefined;
  };
  const mostrarErrorCampo = ({ error, control }) => {
    limpiarErrorCampo(); ponerEstado(error);
    if (control) {
      const mensaje = nodo(doc, "p", t(error));
      const indice = control.dataset.dietasLiquidacionDecision ?? control.dataset.dietasLiquidacionIndice ?? control.dataset.dietasLiquidacionMotivo;
      const tipo = control.dataset.dietasLiquidacionDecision !== undefined ? "decision"
        : control.dataset.dietasLiquidacionIndice !== undefined ? "importe" : "motivo";
      mensaje.id = `dietas-liquidacion-error-${referencia}-${indice}-${tipo}`;
      mensaje.dataset.dietasLiquidacionErrorCampo = "";
      control.setAttribute("aria-invalid", "true"); control.setAttribute("aria-describedby", mensaje.id);
      control.closest("td")?.append(mensaje); errorCampo = { control, mensaje };
      control.focus();
    }
  };
  const boton = (clave, accion, clase = "boton-secundario") => { const b = nodo(doc, "button", t(clave)); b.type = "button"; b.className = clase; b.dataset.dietasLiquidacionAccion = accion; return b; };
  const par = (clave, valor) => { const grupo = nodo(doc, "div"); grupo.append(nodo(doc, "dt", t(clave)), nodo(doc, "dd", String(valor ?? ""))); return grupo; };
  const calculado = (linea) => linea.calculado_publicado_centimos ?? 0;
  const diferencia = (linea) => linea.diferencia_centimos ?? 0;
  const eurosConSigno = (valor) => `${valor > 0 ? "+" : ""}${euro(valor)}`;
  function capturar() {
    for (let i = 0; i < propuesta.lineas.length; i++) {
      const linea = propuesta.lineas[i]; const eleccion = contenido.querySelector?.(`[data-dietas-liquidacion-decision="${i}"]`);
      const decision = eleccion?.value || ""; decisiones[i] = decision;
      if (!decision) return { error: "decision_obligatoria", control: eleccion };
      const revision = revisiones[i];
      if (decision === "aceptar") {
        revision.reconocido_propuesto_centimos = calculado(linea);
        revision.motivo_codigo = diferencia(linea) ? MOTIVO_DIFERENCIA : "";
      } else if (decision === "orientativo" && calculado(linea) > linea.original_centimos) {
        revision.reconocido_propuesto_centimos = linea.original_centimos; revision.motivo_codigo = "";
      } else if (decision === "ajustar") {
        const control = contenido.querySelector?.(`[data-dietas-liquidacion-indice="${i}"]`);
        const valor = String(control?.value || "");
        if (!/^\d+(?:[.,]\d{1,2})?$/u.test(valor)) return { error: "importe_invalido", control };
        const [euros, decimales = ""] = valor.replace(",", ".").split(".");
        const importe = Number(euros) * 100 + Number(decimales.padEnd(2, "0"));
        if (!centimos(importe) || importe > calculado(linea)) return { error: "importe_invalido", control };
        if (importe === calculado(linea)) return { error: "ajuste_sin_cambio", control };
        if (importe === linea.original_centimos && calculado(linea) > linea.original_centimos) return { error: "mantener_orientativo", control };
        const motivo = contenido.querySelector?.(`[data-dietas-liquidacion-motivo="${i}"]`);
        if (!motivos.includes(motivo?.value) || !codigo.test(motivo.value)) return { error: "motivo_obligatorio", control: motivo };
        revision.reconocido_propuesto_centimos = importe; revision.motivo_codigo = motivo.value;
      } else return { error: "decision_obligatoria", control: eleccion };
    }
    return {};
  }
  function pintar() {
    if (!viva || !textos) return;
    titulo.textContent = t(modoConsulta ? "consulta_titulo" : "titulo"); limpiarErrorCampo(); contenido.replaceChildren();
    if (liquidacion) {
      const c = liquidacion.constancia;
      const resumen = nodo(doc, "dl"); resumen.append(par("numero_documento", c.numero_documento), par("recibo", c.recibo_ref), par("fecha_registro", fecha(c.fecha)),
        par("version", c.version_resultante), par("total_reconocido", euro(liquidacion.economia.totales.reconocido_propuesto_centimos)));
      contenido.append(nodo(doc, "h5", t("registrada")), nodo(doc, "p", t("pendiente_firma")), resumen);
      if (typeof cliente.descargar === "function" && /^VEC-D-\d{4}-\d{6,18}$/u.test(c.numero_documento)) {
        const acciones = nodo(doc, "div"); acciones.className = "dietas-circuito-botones";
        for (const formato of ["pdf", "docx"]) {
          const control = boton(formato === "pdf" ? "descargar_pdf" : "descargar_word", `descargar_${formato}`);
          control.disabled = Boolean(descargaControlador); acciones.append(control);
        }
        contenido.append(acciones);
      }
      return;
    }
    if (!propuesta) {
      if (modoConsulta && estado.dataset.nivel !== "cargando") contenido.append(boton("reintentar_consulta", "recargar"));
      else if (estado.dataset.nivel === "error" && !denegada) contenido.append(boton("reintentar_consulta", "recargar"));
      return;
    }
    const catalogo = propuesta.catalogo; const ficha = nodo(doc, "dl"); ficha.className = "dietas-circuito-resumen";
    ficha.append(par("catalogo", `${catalogo.referencia} · ${catalogo.version}`), par("publicacion", catalogo.publicacion_ref),
      par("fecha_publicacion", fechaCivil(catalogo.fecha_publicacion)), par("vigencia", `${fechaCivil(catalogo.vigente_desde)} – ${fechaCivil(catalogo.vigente_hasta)}`),
      par("tarifa", catalogo.version_tarifa_ref), par("ambito", catalogo.ambito), par("grupo", catalogo.grupo));
    const normas = nodo(doc, "ul");
    catalogo.normas.forEach((norma) => { const li = nodo(doc, "li"); const a = nodo(doc, "a", `${norma.referencia} · ${norma.articulo}`); a.href = norma.url;
      a.rel = "noopener noreferrer"; a.target = "_blank"; li.append(a); normas.append(li); });
    contenido.append(nodo(doc, "h5", t("fuente_aplicable")), ficha, nodo(doc, "h5", t("normas")), normas);
    const marco = nodo(doc, "div"); marco.className = "tabla-contenedor"; marco.setAttribute("tabindex", "0");
    marco.setAttribute("role", "region"); marco.setAttribute("aria-label", t("lineas"));
    const tabla = nodo(doc, "table"); tabla.className = "tabla-datos";
    tabla.append(nodo(doc, "caption", t("lineas")));
    const thead = nodo(doc, "thead"); const trh = nodo(doc, "tr");
    ["concepto", "original", "calculado_publicado", "diferencia", "decision", "admitido", "rechazado", "motivo"].forEach((k) => { const th = nodo(doc, "th", t(k)); th.setAttribute("scope", "col"); trh.append(th); }); thead.append(trh); tabla.append(thead);
    const tbody = nodo(doc, "tbody");
    propuesta.lineas.forEach((l, i) => {
      const fila = nodo(doc, "tr"); const regla = catalogo.reglas.find((r) => r.referencia === l.regla_ref);
      const concepto = regla ? t(`tipo_${regla.tipo}`) : t("concepto");
      const th = nodo(doc, "th", t("linea_numero", { concepto, numero: textos.numero(i + 1) })); th.setAttribute("scope", "row");
      const eleccion = nodo(doc, "select"); eleccion.dataset.dietasLiquidacionDecision = String(i);
      eleccion.setAttribute("aria-label", t("decision_linea", { concepto })); eleccion.disabled = Boolean(pendiente || revisionVisible || denegada);
      const opciones = ["", "aceptar", ...(calculado(l) > l.original_centimos ? ["orientativo"] : []), "ajustar"];
      opciones.forEach((valor) => { const opcion = nodo(doc, "option", t(`decision_${valor || "pendiente"}`)); opcion.value = valor; eleccion.append(opcion); });
      eleccion.value = decisiones[i];
      const campo = nodo(doc, "input"); campo.type = "number"; campo.min = "0"; campo.max = String(calculado(l) / 100); campo.step = "0.01";
      campo.value = (revisiones[i].reconocido_propuesto_centimos / 100).toFixed(2); campo.dataset.dietasLiquidacionIndice = String(i);
      campo.setAttribute("aria-label", t("admitido_linea", { concepto })); campo.disabled = Boolean(pendiente || revisionVisible || denegada || decisiones[i] !== "ajustar");
      const select = nodo(doc, "select"); select.dataset.dietasLiquidacionMotivo = String(i); select.setAttribute("aria-label", t("motivo_linea", { concepto })); select.disabled = Boolean(pendiente || revisionVisible || denegada || decisiones[i] !== "ajustar");
      ["", ...motivos].forEach((k) => { const option = nodo(doc, "option", k ? t(`motivo_${k}`) : t("sin_reduccion")); option.value = k; option.selected = revisiones[i].motivo_codigo === k; select.append(option); }); select.value = revisiones[i].motivo_codigo;
      const importe = nodo(doc, "td"); importe.append(campo); const motivo = nodo(doc, "td"); motivo.append(select);
      if (decisiones[i] === "aceptar" && diferencia(l)) motivo.replaceChildren(nodo(doc, "span", t("motivo_diferencia_tarifa_publicada")));
      if (decisiones[i] === "orientativo") motivo.replaceChildren(nodo(doc, "span", t("sin_reduccion")));
      const celdaDecision = nodo(doc, "td"); celdaDecision.append(eleccion);
      fila.append(th, nodo(doc, "td", euro(l.original_centimos)), nodo(doc, "td", euro(calculado(l))),
        nodo(doc, "td", eurosConSigno(diferencia(l))), celdaDecision, importe,
        nodo(doc, "td", decisiones[i] ? euro(calculado(l) - revisiones[i].reconocido_propuesto_centimos) : t("decision_pendiente")), motivo); tbody.append(fila);
    });
    tabla.append(tbody); marco.append(tabla); contenido.append(marco);
    const total = revisiones.reduce((n, r) => n + r.reconocido_propuesto_centimos, 0);
    const resumen = nodo(doc, "dl"); resumen.className = "dietas-circuito-resumen";
    const publicado = propuesta.totales.calculado_publicado_centimos ?? 0;
    resumen.append(par("total_original", euro(propuesta.totales.original_centimos)), par("total_calculado", euro(publicado)),
      par("total_diferencia", eurosConSigno(propuesta.totales.diferencia_centimos ?? 0)),
      par("total_admitido", decisiones.every(Boolean) ? euro(total) : t("decision_pendiente")),
      par("total_rechazado", decisiones.every(Boolean) ? euro(publicado - total) : t("decision_pendiente"))); contenido.append(resumen);
    if (denegada) return;
    if (pendiente?.incierta) contenido.append(boton("reintentar_misma", "reintentar"));
    else if (revisionVisible) contenido.append(nodo(doc, "p", t("revisar_aviso")), boton("corregir", "corregir"), boton("confirmar", "confirmar", "boton-primario"));
    else contenido.append(boton("revisar", "revisar", "boton-primario"));
  }
  async function consultar() {
    controlador?.abort(); controlador = new AbortController(); const actual = ++generacion;
    propuesta = undefined; if (modoConsulta) liquidacion = undefined;
    denegada = false; ponerEstado("cargando"); estado.dataset.nivel = "cargando"; pintar();
    try {
      if (modoConsulta) {
        const leida = await cliente.leer(referencia, version, { signal: controlador.signal });
        if (!viva || actual !== generacion) return;
        liquidacion = leida; ponerEstado("consulta_lista"); estado.dataset.nivel = "exito"; pintar(); return;
      }
      const p = await cliente.propuesta(referencia, version, { signal: controlador.signal });
      if (!viva || actual !== generacion) return;
      propuesta = p; revisiones = p.lineas.map((l) => ({ indice: l.indice, regla_ref: l.regla_ref,
        reconocido_propuesto_centimos: l.reconocido_propuesto_centimos, motivo_codigo: l.motivo_codigo }));
      decisiones = p.lineas.map((l) => p.decision_pendiente && diferencia(l) ? ""
        : l.reconocido_propuesto_centimos === calculado(l) ? "aceptar"
          : calculado(l) > l.original_centimos && l.reconocido_propuesto_centimos === l.original_centimos && !l.motivo_codigo ? "orientativo" : "ajustar");
      estado.textContent = ""; estado.dataset.nivel = ""; pintar();
    } catch (e) {
      if (!viva || actual !== generacion || e?.codigo === "operacion_abortada") return;
      denegada = !modoConsulta && ["acceso_denegado", "competencia_sin_fuente"].includes(e?.codigo);
      ponerEstado(modoConsulta ? e?.codigo === "no_encontrada" ? "consulta_vacia"
        : ["acceso_denegado", "competencia_sin_fuente"].includes(e?.codigo) ? "consulta_denegada" : "consulta_error"
        : denegada ? "denegado" : e?.codigo === "conflicto_estado" ? "conflicto" : "error_consulta");
      estado.dataset.nivel = e?.codigo === "no_encontrada" && modoConsulta ? "" : "error"; pintar();
    } finally { if (actual === generacion) controlador = undefined; }
  }
  async function registrar(reintento = false) {
    if (controlador || !propuesta || liquidacion || denegada || (pendiente && !reintento) || (reintento && !pendiente?.incierta)) return;
    if (!reintento) {
      const comprobacion = capturar();
      if (comprobacion.error) { mostrarErrorCampo(comprobacion); return; }
      if (!revisionVisible || JSON.stringify(revisiones) !== revisionConfirmada) { revisionVisible = false; ponerEstado("revisar_actualizado"); pintar(); return; }
      const clave = generarClaveIdempotencia();
      if (typeof clave !== "string" || !/^[A-Za-z0-9:_-]{16,128}$/u.test(clave)) { ponerEstado("error_envio"); return; }
      pendiente = { entrada: { version_esperada: version, clave_idempotencia: clave, revisiones: revisiones.map((r) => ({ ...r })) }, incierta: false };
      alCambiarBloqueo(true);
    }
    if (reintento) { pendiente.conflicto = false; alCambiarBloqueo(true); }
    controlador = new AbortController(); const actual = ++generacion; ponerEstado("registrando"); estado.dataset.nivel = "cargando"; pintar();
    try {
      const resultado = await cliente.registrar(referencia, pendiente.entrada, { signal: controlador.signal });
      if (!viva || actual !== generacion) return;
      liquidacion = resultado; pendiente = undefined; alCambiarBloqueo(false); ponerEstado("registrada"); estado.dataset.nivel = "exito"; pintar(); alRegistrar(resultado);
    } catch (e) {
      if (!viva || actual !== generacion || e?.codigo === "operacion_abortada") return;
      if (["conflicto_estado", "conflicto_version", "conflicto_idempotencia"].includes(e?.codigo)) {
        pendiente = { ...pendiente, incierta: true, conflicto: true };
        alCambiarBloqueo(false); ponerEstado("conflicto_reintento");
      } else if (e?.resultadoIndeterminado) { pendiente.incierta = true; ponerEstado("incierto"); }
      else { pendiente = undefined; alCambiarBloqueo(false); denegada = ["acceso_denegado", "competencia_sin_fuente"].includes(e?.codigo); ponerEstado(denegada ? "denegado"
        : ["conflicto_estado", "conflicto_version", "conflicto_idempotencia", "documento_no_disponible"].includes(e?.codigo) ? "conflicto" : "error_envio"); }
      estado.dataset.nivel = "error"; pintar();
    } finally { if (actual === generacion) controlador = undefined; }
  }
  async function descargar(formato) {
    if (!viva || descargaControlador || !liquidacion || typeof cliente.descargar !== "function") return;
    const c = liquidacion.constancia;
    if (!/^VEC-D-\d{4}-\d{6,18}$/u.test(c.numero_documento)
      || !Number.isSafeInteger(c.version_resultante) || c.version_resultante < 2) return;
    descargaControlador = new AbortController(); const actual = ++generacion;
    alCambiarBloqueo(true); ponerEstado("descargando"); estado.dataset.nivel = "cargando"; pintar();
    try {
      const archivo = await cliente.descargar(referencia, c.version_resultante, c.numero_documento, formato, textos.idioma,
        { signal: descargaControlador.signal });
      if (!viva || actual !== generacion || descargaControlador.signal.aborted) return;
      if (archivo.nombre !== `${c.numero_documento}.${formato}` || !(archivo.bytes instanceof Uint8Array)
        || !archivo.bytes.byteLength || typeof entornoDescarga.URL?.createObjectURL !== "function"
        || typeof entornoDescarga.URL?.revokeObjectURL !== "function"
        || !entornoDescarga.document?.body || typeof entornoDescarga.Blob !== "function") throw new TypeError("descarga no disponible");
      liberarURL(); urlDocumento = entornoDescarga.URL.createObjectURL(new entornoDescarga.Blob([archivo.bytes], { type: archivo.mime }));
      const enlace = entornoDescarga.document.createElement("a");
      try { enlace.href = urlDocumento; enlace.download = archivo.nombre; enlace.hidden = true;
        entornoDescarga.document.body.append(enlace); enlace.click(); }
      finally { enlace.remove(); revocacion = setTimeout(liberarURL, 0); }
      ponerEstado("descarga_lista"); estado.dataset.nivel = "exito";
    } catch (error) {
      if (!viva || actual !== generacion || error?.codigo === "operacion_abortada") return;
      ponerEstado(["acceso_denegado", "competencia_sin_fuente"].includes(error?.codigo) ? "descarga_denegada"
        : ["conflicto_estado", "conflicto_version", "no_encontrada"].includes(error?.codigo) ? "descarga_conflicto" : "descarga_error");
      estado.dataset.nivel = "error";
    } finally {
      if (actual === generacion) { descargaControlador = undefined; alCambiarBloqueo(false); pintar(); }
    }
  }
  async function clic(evento) {
    const accion = evento.target?.closest?.("[data-dietas-liquidacion-accion]")?.dataset.dietasLiquidacionAccion;
    if (!accion || !viva) return;
    if (accion === "recargar") return consultar();
    if (accion === "reintentar") return registrar(true);
    if (accion === "descargar_pdf" || accion === "descargar_docx") return descargar(accion.slice("descargar_".length));
    if (accion === "corregir") { revisionVisible = false; revisionConfirmada = undefined; pintar(); return; }
    if (accion === "revisar") {
      const resultado = capturar(); if (resultado.error) { mostrarErrorCampo(resultado); return; }
      revisionVisible = true; revisionConfirmada = JSON.stringify(revisiones); ponerEstado("revisar_aviso"); pintar(); estado.focus?.(); return;
    }
    if (accion === "confirmar" && revisionVisible) return registrar();
  }
  async function cambio(evento) {
    const eleccion = evento.target?.closest?.("[data-dietas-liquidacion-decision]");
    if (eleccion) {
      limpiarErrorCampo();
      const i = Number(eleccion.dataset.dietasLiquidacionDecision); decisiones[i] = eleccion.value;
      const linea = propuesta.lineas[i];
      if (eleccion.value === "aceptar") { revisiones[i].reconocido_propuesto_centimos = calculado(linea);
        revisiones[i].motivo_codigo = diferencia(linea) ? MOTIVO_DIFERENCIA : ""; }
      else if (eleccion.value === "orientativo") { revisiones[i].reconocido_propuesto_centimos = linea.original_centimos; revisiones[i].motivo_codigo = ""; }
      revisionVisible = false; revisionConfirmada = undefined; estado.textContent = ""; pintar();
      const selectorFoco = eleccion.value === "ajustar" ? `[data-dietas-liquidacion-indice="${i}"]` : `[data-dietas-liquidacion-decision="${i}"]`;
      contenido.querySelector?.(selectorFoco)?.focus?.();
      return;
    }
    if (evento.target?.closest?.("[data-dietas-liquidacion-indice]") || evento.target?.closest?.("[data-dietas-liquidacion-motivo]")) {
      if (errorCampo?.control === evento.target) limpiarErrorCampo();
      revisionVisible = false; revisionConfirmada = undefined; estado.textContent = "";
    }
  }
  function editarCampo(evento) {
    if (errorCampo?.control !== evento.target) return;
    limpiarErrorCampo(); revisionVisible = false; revisionConfirmada = undefined; estado.textContent = "";
  }
  function desmontar() { if (!viva) return; viva = false; ++generacion; controlador?.abort(); descargaControlador?.abort(); liberarURL(); raiz.removeEventListener("click", clic); raiz.removeEventListener("change", cambio); raiz.removeEventListener("input", editarCampo); raiz.remove(); }
  raiz.addEventListener("click", clic); raiz.addEventListener("change", cambio); raiz.addEventListener("input", editarCampo); registrarDesmontar?.(desmontar);
  cargarTextosImpl("dietas").then((c) => { if (!viva) return; textos = c; consultar(); }).catch(() => { if (viva) { estado.textContent = ""; estado.dataset.nivel = "error"; } });
  return Object.freeze({ desmontar, recargar: consultar, bloqueaSalida: () => Boolean((pendiente && !pendiente.conflicto) || descargaControlador) });
}
