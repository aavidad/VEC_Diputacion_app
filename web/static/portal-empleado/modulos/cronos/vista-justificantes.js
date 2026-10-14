import { crearClienteJustificantesCronosHTTP } from "./cliente-justificantes-http.js";

const CLAVES = ["titulo", "ayuda", "subir", "enviando", "pendiente", "recibo", "error", "reintentar",
  "abrir_original", "revisar", "aceptar", "rechazar", "motivo", "sin_motivo", "no_disponible"];

function textosValidos(m) { return m && CLAVES.every((k) => typeof m[k] === "string" && m[k].trim()); }
function entradaValida(solicitudRef, versionEsperada) {
  return typeof solicitudRef === "string" && /^permiso:cronos:solicitud:[A-Za-z0-9:._-]{1,128}$/u.test(solicitudRef) &&
    Number.isSafeInteger(versionEsperada) && versionEsperada >= 0;
}
function crearClaveOperacion() {
  const bytes = new Uint8Array(32);
  globalThis.crypto.getRandomValues(bytes);
  return `ref:${Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("")}`;
}

// subirOriginal invoca la carga común de Documentos. Su resultado se enlaza
// solo después de que el servidor vuelva a leer el original admitido.
export function montarAporteJustificanteCronos({ raiz, solicitudRef, versionEsperada, subirOriginal,
  cliente = crearClienteJustificantesCronosHTTP(), mensajes, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.ownerDocument?.createElement || !entradaValida(solicitudRef, versionEsperada) ||
    !textosValidos(mensajes) || typeof subirOriginal !== "function" ||
    typeof cliente?.aportar !== "function" || typeof anunciar !== "function" ||
    (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("aporte no disponible");
  const doc = raiz.ownerDocument;
  const panel = doc.createElement("section"); panel.className = "panel cronos-panel";
  const titulo = doc.createElement("h3"); titulo.textContent = mensajes.titulo;
  const ayuda = doc.createElement("p"); ayuda.textContent = mensajes.ayuda;
  const boton = doc.createElement("button"); boton.type = "button"; boton.className = "boton-primario"; boton.textContent = mensajes.subir;
  const estado = doc.createElement("p"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  panel.append(titulo, ayuda, boton, estado); raiz.append(panel);
  let pendiente = null; let ocupada = false; let viva = true;
  const actuar = async () => {
    if (!viva || ocupada) return;
    ocupada = true; boton.disabled = true; estado.textContent = mensajes.enviando;
    try {
      if (!pendiente) {
        const original = await subirOriginal();
        if (!original || typeof original !== "object") throw new TypeError("subida no acreditada");
        pendiente = { solicitud_ref: solicitudRef, version_esperada: versionEsperada, clave_operacion: crearClaveOperacion(),
          documento_id: original.id, documento_version: original.version, sha256: original.sha256 };
      }
      const resultado = await cliente.aportar(pendiente);
      if (!viva) return;
      if (resultado.enlace_pendiente) {
        estado.textContent = mensajes.pendiente; boton.textContent = mensajes.reintentar;
      } else {
        estado.textContent = `${mensajes.recibo} ${resultado.recibo_ref}`;
        anunciar(estado.textContent); boton.hidden = true; pendiente = null;
      }
    } catch {
      if (viva) { estado.textContent = mensajes.error; anunciar(mensajes.error); boton.textContent = mensajes.reintentar; }
    } finally { ocupada = false; if (viva) boton.disabled = false; }
  };
  boton.addEventListener("click", actuar);
  const desmontar = () => { viva = false; boton.removeEventListener("click", actuar); panel.remove(); pendiente = null; };
  registrarDesmontar?.(desmontar);
  return desmontar;
}

// La bandeja autorizada entrega la referencia y versión actuales. abrirOriginal
// usa la descarga común de Documentos y debe resolver antes de habilitar revisión.
export function montarRevisionJustificanteCronos({ raiz, solicitudRef, versionEsperada, documentoID,
  abrirOriginal, motivos, cliente = crearClienteJustificantesCronosHTTP(), mensajes,
  anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.ownerDocument?.createElement || !entradaValida(solicitudRef, versionEsperada) || versionEsperada < 1 ||
    typeof documentoID !== "string" || !/^ref:[0-9a-f]{64}$/u.test(documentoID) ||
    !textosValidos(mensajes) || typeof abrirOriginal !== "function" ||
    typeof cliente?.revisar !== "function" || !Array.isArray(motivos) || motivos.length === 0 ||
    typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("revisión no disponible");
  }
  const doc = raiz.ownerDocument;
  const panel = doc.createElement("section"); panel.className = "panel cronos-panel";
  const titulo = doc.createElement("h3"); titulo.textContent = mensajes.titulo;
  const ayuda = doc.createElement("p"); ayuda.textContent = mensajes.ayuda;
  const abrir = doc.createElement("button"); abrir.type = "button"; abrir.className = "boton-secundario"; abrir.textContent = mensajes.abrir_original;
  const formulario = doc.createElement("form"); formulario.hidden = true;
  const etiqueta = doc.createElement("label"); etiqueta.textContent = mensajes.motivo;
  const selector = doc.createElement("select"); selector.required = true; selector.className = "control-formulario";
  const vacio = doc.createElement("option"); vacio.value = ""; vacio.textContent = mensajes.sin_motivo; selector.append(vacio);
  for (const motivo of motivos) {
    if (typeof motivo?.ref !== "string" || typeof motivo?.etiqueta !== "string") throw new TypeError("motivo no disponible");
    const opcion = doc.createElement("option"); opcion.value = motivo.ref; opcion.textContent = motivo.etiqueta; selector.append(opcion);
  }
  etiqueta.append(selector);
  const aceptar = doc.createElement("button"); aceptar.type = "submit"; aceptar.value = "aceptada"; aceptar.textContent = mensajes.aceptar;
  const rechazar = doc.createElement("button"); rechazar.type = "submit"; rechazar.value = "rechazada"; rechazar.textContent = mensajes.rechazar;
  formulario.append(etiqueta, aceptar, rechazar);
  const estado = doc.createElement("p"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  panel.append(titulo, ayuda, abrir, formulario, estado); raiz.append(panel);
  let viva = true; let ocupada = false; let pendiente = null;
  const abrirDocumento = async () => {
    if (!viva || ocupada) return;
    ocupada = true; abrir.disabled = true;
    try {
      const original = await abrirOriginal(documentoID);
      if (!viva) return;
      if (!original || original.id !== documentoID || original.estado !== "admitido") throw new TypeError("original no admitido");
      formulario.hidden = false; estado.textContent = ""; selector.focus();
    } catch { if (viva) { estado.textContent = mensajes.no_disponible; anunciar(mensajes.no_disponible); } }
    finally { ocupada = false; if (viva) abrir.disabled = false; }
  };
  const revisar = async (evento) => {
    evento.preventDefault();
    if (!viva || ocupada || formulario.hidden || !selector.value) return;
    const decision = evento.submitter?.value;
    if (!["aceptada", "rechazada"].includes(decision)) return;
    ocupada = true; aceptar.disabled = true; rechazar.disabled = true; estado.textContent = mensajes.enviando;
    try {
      if (!pendiente) pendiente = { solicitud_ref: solicitudRef, clave_operacion: crearClaveOperacion(),
        version_esperada: versionEsperada, decision, motivo_ref: selector.value };
      const recibo = await cliente.revisar(pendiente);
      if (!viva) return;
      estado.textContent = `${mensajes.recibo} ${recibo.recibo_ref}`; anunciar(estado.textContent);
      formulario.hidden = true; pendiente = null;
    } catch { if (viva) { estado.textContent = mensajes.error; anunciar(mensajes.error); } }
    finally { ocupada = false; if (viva) { aceptar.disabled = false; rechazar.disabled = false; } }
  };
  abrir.addEventListener("click", abrirDocumento); formulario.addEventListener("submit", revisar);
  const desmontar = () => { viva = false; abrir.removeEventListener("click", abrirDocumento); formulario.removeEventListener("submit", revisar); panel.remove(); pendiente = null; };
  registrarDesmontar?.(desmontar);
  return desmontar;
}
