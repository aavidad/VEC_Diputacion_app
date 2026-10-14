import { crearClienteOriginalC8HTTP } from "./cliente-justificantes-original-c8.js?v=20261014-cronos-c8-original-v1";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const ref = (v) => typeof v === "string" && /^ref:[0-9a-f]{64}$/u.test(v) && v !== `ref:${"0".repeat(64)}`;
const huella = (v) => typeof v === "string" && /^[0-9a-f]{64}$/u.test(v) && v !== "0".repeat(64);
const solicitud = (v) => typeof v === "string" && /^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$/u.test(v);
const entero = (v, minimo = 0) => Number.isSafeInteger(v) && v >= minimo;
const TEXTOS = ["titulo_aporte", "archivo", "aportar", "enviando", "analisis_pendiente", "error_archivo", "error_aporte",
  "recibo", "titulo_revision", "abrir_original", "original_no_disponible", "motivo", "elegir_motivo", "aceptar", "rechazar", "error_revision",
  "titulo_historia", "sin_justificantes", "estado_pendiente_revision", "estado_aceptada", "estado_rechazada", "actualizar",
  "error_consulta", "titulo_bandeja", "sin_pendientes", "persona_sin_nombre", "ver_justificante"];
const textosValidos = (v) => v && TEXTOS.every((k) => typeof v[k] === "string" && v[k].trim());

function claveNueva() {
  const bytes = new Uint8Array(32);
  globalThis.crypto.getRandomValues(bytes);
  return `ref:${Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("")}`;
}

function elemento(doc, etiqueta, clase, texto) {
  const e = doc.createElement(etiqueta);
  if (clase) e.className = clase;
  if (texto !== undefined) e.textContent = texto;
  return e;
}

function resultadoSubidaAdmitida(v) {
  return v && v.estado === "admitido" && ref(v.comprobante_ref) && entero(v.documento_version, 1) &&
    typeof v.recibo_ref === "string" && v.recibo_ref.length > 0;
}

/** La carga común entrega ID y versión; Cronos acredita huella y custodia en servidor. */
export function montarAporteOriginalC8({ raiz, solicitudRef, versionEsperada, versionPermiso, subirOriginal,
  cliente = crearClienteOriginalC8HTTP(), mensajes, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.ownerDocument?.createElement || !solicitud(solicitudRef) || !entero(versionEsperada) ||
    !entero(versionPermiso, 1) ||
    typeof subirOriginal !== "function" || typeof cliente?.aportar !== "function" || !textosValidos(mensajes) ||
    typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("aporte no disponible");
  }
  const doc = raiz.ownerDocument;
  const panel = elemento(doc, "section", "panel cronos-panel");
  const titulo = elemento(doc, "h3", "", mensajes.titulo_aporte);
  const form = elemento(doc, "form", "cronos-notificacion-formulario");
  const etiqueta = elemento(doc, "label", "cronos-campo", mensajes.archivo);
  const archivo = elemento(doc, "input"); archivo.type = "file"; archivo.accept = "application/pdf"; archivo.required = true;
  etiqueta.append(archivo);
  const boton = elemento(doc, "button", "boton-primario", mensajes.aportar); boton.type = "submit";
  const estado = elemento(doc, "p", "cronos-solicitud-aviso"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  form.append(etiqueta, boton); panel.append(titulo, form, estado); raiz.append(panel);
  let viva = true; let ocupada = false; let carga = null; let claveCarga = ""; let claveEnlace = ""; let control = null;
  const publicar = (texto) => { estado.textContent = texto; anunciar(texto); };
  const enviar = async (evento) => {
    evento.preventDefault();
    if (!viva || ocupada || !archivo.files || archivo.files.length !== 1) {
      if (viva) publicar(mensajes.error_archivo);
      return;
    }
    ocupada = true; boton.disabled = true; estado.textContent = mensajes.enviando;
    control = new AbortController();
    try {
      if (!carga) {
        claveCarga ||= claveNueva();
        archivo.disabled = true;
        const respuesta = await subirOriginal(archivo.files[0], { solicitudRef, versionPermiso,
          claveOperacion: claveCarga, signal: control.signal });
        if (!viva || control.signal.aborted) return;
        if (!resultadoSubidaAdmitida(respuesta)) {
          if (respuesta?.estado === "reservada" || respuesta?.estado === "recibido" || respuesta?.estado === "analizando") {
            publicar(mensajes.analisis_pendiente); return;
          }
          if (respuesta?.estado === "rechazado") { claveCarga = ""; archivo.disabled = false; }
          throw new TypeError("original no admitido");
        }
        carga = Object.freeze({ documento_id: respuesta.comprobante_ref, documento_version: respuesta.documento_version });
      }
      claveEnlace ||= claveNueva();
      const recibo = await cliente.aportar({ solicitud_ref: solicitudRef, clave_operacion: claveEnlace,
        version_esperada: versionEsperada, ...carga }, { signal: control.signal });
      if (!viva || control.signal.aborted) return;
      publicar(`${mensajes.recibo} ${recibo.recibo_ref}`); boton.hidden = true;
    } catch {
      if (viva && !control.signal.aborted) publicar(mensajes.error_aporte);
    } finally { ocupada = false; if (viva) boton.disabled = false; }
  };
  form.addEventListener("submit", enviar);
  const desmontar = () => { viva = false; control?.abort(); form.removeEventListener("submit", enviar); panel.remove(); carga = null; };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}

/** RRHH abre el original con su autorización documental antes de revisarlo. */
export function montarRevisionOriginalC8({ raiz, solicitudRef, versionEsperada, documentoID, documentoVersion, documentoSHA256,
  abrirOriginal, motivos, cliente = crearClienteOriginalC8HTTP(), mensajes,
  anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.ownerDocument?.createElement || !solicitud(solicitudRef) || !entero(versionEsperada, 1) ||
    !ref(documentoID) || !entero(documentoVersion, 1) || !huella(documentoSHA256) || typeof abrirOriginal !== "function" ||
    typeof cliente?.revisar !== "function" || !Array.isArray(motivos) || motivos.length === 0 || !textosValidos(mensajes) ||
    typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("revisión no disponible");
  }
  const doc = raiz.ownerDocument;
  const panel = elemento(doc, "section", "panel cronos-panel");
  const titulo = elemento(doc, "h3", "", mensajes.titulo_revision);
  const abrir = elemento(doc, "button", "boton-secundario", mensajes.abrir_original); abrir.type = "button";
  const form = elemento(doc, "form", "cronos-notificacion-formulario"); form.hidden = true;
  const etiqueta = elemento(doc, "label", "cronos-campo", mensajes.motivo);
  const selector = elemento(doc, "select", "control-formulario"); selector.required = true;
  const vacio = elemento(doc, "option", "", mensajes.elegir_motivo); vacio.value = ""; selector.append(vacio);
  for (const motivo of motivos) {
    if (typeof motivo?.ref !== "string" || typeof motivo?.etiqueta !== "string") throw new TypeError("motivo no disponible");
    const opcion = elemento(doc, "option", "", motivo.etiqueta); opcion.value = motivo.ref; selector.append(opcion);
  }
  etiqueta.append(selector);
  const aceptar = elemento(doc, "button", "boton-primario", mensajes.aceptar); aceptar.type = "submit"; aceptar.value = "aceptada";
  const rechazar = elemento(doc, "button", "boton-secundario", mensajes.rechazar); rechazar.type = "submit"; rechazar.value = "rechazada";
  const estado = elemento(doc, "p", "cronos-solicitud-aviso"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  form.append(etiqueta, aceptar, rechazar); panel.append(titulo, abrir, form, estado); raiz.append(panel);
  let viva = true; let ocupada = false; let abierta = false; let pendiente = null; let control = null;
  const publicar = (texto) => { estado.textContent = texto; anunciar(texto); };
  const abrirDocumento = async () => {
    if (!viva || ocupada) return;
    ocupada = true; abrir.disabled = true; control = new AbortController();
    try {
      const visto = await abrirOriginal(documentoID, documentoVersion, { signal: control.signal });
      if (!viva || control.signal.aborted) return;
      if (!visto || visto.documento_id !== documentoID || visto.documento_version !== documentoVersion || visto.sha256 !== documentoSHA256 ||
        visto.estado !== "admitido" || visto.abierto !== true) throw new TypeError("original no abierto");
      abierta = true; form.hidden = false; estado.textContent = ""; selector.focus();
    } catch { if (viva && !control.signal.aborted) publicar(mensajes.original_no_disponible); }
    finally { ocupada = false; if (viva) abrir.disabled = false; }
  };
  const revisar = async (evento) => {
    evento.preventDefault();
    if (!viva || ocupada || !abierta || form.hidden || !selector.value) return;
    const decision = evento.submitter?.value;
    if (!["aceptada", "rechazada"].includes(decision)) return;
    const firma = `${decision}\u0000${selector.value}`;
    if (!pendiente || pendiente.firma !== firma) pendiente = { firma, clave: claveNueva() };
    ocupada = true; aceptar.disabled = true; rechazar.disabled = true; control = new AbortController(); estado.textContent = mensajes.enviando;
    try {
      const recibo = await cliente.revisar({ solicitud_ref: solicitudRef, clave_operacion: pendiente.clave,
        version_esperada: versionEsperada, decision, motivo_ref: selector.value }, { signal: control.signal });
      if (!viva || control.signal.aborted) return;
      publicar(`${mensajes.recibo} ${recibo.recibo_ref}`); form.hidden = true; pendiente = null;
    } catch { if (viva && !control.signal.aborted) publicar(mensajes.error_revision); }
    finally { ocupada = false; if (viva) { aceptar.disabled = false; rechazar.disabled = false; } }
  };
  abrir.addEventListener("click", abrirDocumento); form.addEventListener("submit", revisar);
  const desmontar = () => { viva = false; control?.abort(); abrir.removeEventListener("click", abrirDocumento); form.removeEventListener("submit", revisar); panel.remove(); pendiente = null; };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}

function fechaVisible(instante, locale) {
  return new Intl.DateTimeFormat(locale, { timeZone: "Europe/Madrid", dateStyle: "medium", timeStyle: "short" }).format(new Date(instante));
}

/** La consulta propia recupera estados y recibos originales tras reiniciar. */
export function montarHistoriaOriginalC8({ raiz, solicitudRef, cliente = crearClienteOriginalC8HTTP(),
  mensajes, anunciar = () => {}, registrarDesmontar, locale = LOCALIZACION_ACTUAL } = {}) {
  if (!raiz?.ownerDocument?.createElement || !solicitud(solicitudRef) || typeof cliente?.consultarPropia !== "function" ||
    !textosValidos(mensajes) || typeof anunciar !== "function" ||
    (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("historia no disponible");
  const doc = raiz.ownerDocument;
  const panel = elemento(doc, "section", "panel cronos-panel");
  const titulo = elemento(doc, "h3", "", mensajes.titulo_historia);
  const actualizar = elemento(doc, "button", "boton-secundario", mensajes.actualizar); actualizar.type = "button";
  const estado = elemento(doc, "p", "cronos-solicitud-aviso"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  const lista = elemento(doc, "ul", "cronos-avisos-lista"); panel.append(titulo, actualizar, estado, lista); raiz.append(panel);
  let viva = true; let secuencia = 0; let control = null;
  const cargar = async () => {
    if (!viva) return;
    control?.abort(); control = new AbortController(); const turno = ++secuencia;
    actualizar.disabled = true; lista.replaceChildren(); estado.textContent = mensajes.enviando;
    try {
      const datos = await cliente.consultarPropia(solicitudRef, { signal: control.signal });
      if (!viva || control.signal.aborted || turno !== secuencia) return;
      lista.replaceChildren();
      for (const hecho of datos.historia) {
        const fila = elemento(doc, "li", "cronos-aviso");
        const nombreEstado = mensajes[`estado_${hecho.estado}`];
        fila.append(elemento(doc, "span", "cronos-estado", nombreEstado),
          elemento(doc, "span", "", ` ${fechaVisible(hecho.instante_utc, locale)}`),
          elemento(doc, "span", "", ` ${mensajes.recibo} ${hecho.recibo_ref}`));
        lista.append(fila);
      }
      estado.textContent = datos.historia.length ? "" : mensajes.sin_justificantes;
    } catch {
      if (viva && !control.signal.aborted && turno === secuencia) { estado.textContent = mensajes.error_consulta; anunciar(mensajes.error_consulta); }
    } finally { if (viva && turno === secuencia) actualizar.disabled = false; }
  };
  const pulsar = () => { void cargar(); };
  actualizar.addEventListener("click", pulsar); void cargar();
  const desmontar = () => { viva = false; ++secuencia; control?.abort(); actualizar.removeEventListener("click", pulsar); panel.remove(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}

/** RRHH recibe solo las solicitudes pendientes de su ámbito y abre su detalle. */
export function montarBandejaOriginalC8({ raiz, cliente = crearClienteOriginalC8HTTP(), abrirRevision,
  mensajes, anunciar = () => {}, registrarDesmontar, locale = LOCALIZACION_ACTUAL } = {}) {
  if (!raiz?.ownerDocument?.createElement || typeof cliente?.consultarBandeja !== "function" ||
    typeof abrirRevision !== "function" || !textosValidos(mensajes) || typeof anunciar !== "function" ||
    (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("bandeja no disponible");
  const doc = raiz.ownerDocument;
  const panel = elemento(doc, "section", "panel cronos-panel");
  const titulo = elemento(doc, "h3", "", mensajes.titulo_bandeja);
  const actualizar = elemento(doc, "button", "boton-secundario", mensajes.actualizar); actualizar.type = "button";
  const estado = elemento(doc, "p", "cronos-solicitud-aviso"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  const lista = elemento(doc, "ul", "cronos-avisos-lista"); panel.append(titulo, actualizar, estado, lista); raiz.append(panel);
  let viva = true; let secuencia = 0; let control = null;
  const cargar = async () => {
    if (!viva) return;
    control?.abort(); control = new AbortController(); const turno = ++secuencia;
    actualizar.disabled = true; lista.replaceChildren(); estado.textContent = mensajes.enviando;
    try {
      const datos = await cliente.consultarBandeja({ signal: control.signal });
      if (!viva || control.signal.aborted || turno !== secuencia) return;
      lista.replaceChildren();
      for (const pendiente of datos.pendientes) {
        const fila = elemento(doc, "li", "cronos-aviso");
        const persona = pendiente.empleado_etiqueta || mensajes.persona_sin_nombre;
        const boton = elemento(doc, "button", "boton-secundario", mensajes.ver_justificante);
        boton.type = "button";
        boton.setAttribute("aria-label", `${mensajes.ver_justificante}: ${persona}`);
        boton.addEventListener("click", () => { if (viva) abrirRevision(pendiente); });
        fila.append(elemento(doc, "span", "", persona),
          elemento(doc, "span", "", ` ${fechaVisible(pendiente.registrada_en, locale)}`), boton);
        lista.append(fila);
      }
      estado.textContent = datos.pendientes.length ? "" : mensajes.sin_pendientes;
    } catch {
      if (viva && !control.signal.aborted && turno === secuencia) { estado.textContent = mensajes.error_consulta; anunciar(mensajes.error_consulta); }
    } finally { if (viva && turno === secuencia) actualizar.disabled = false; }
  };
  const pulsar = () => { void cargar(); };
  actualizar.addEventListener("click", pulsar); void cargar();
  const desmontar = () => { viva = false; ++secuencia; control?.abort(); actualizar.removeEventListener("click", pulsar); panel.remove(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
