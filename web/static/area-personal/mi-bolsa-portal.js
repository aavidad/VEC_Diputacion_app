// Acciones propias de la persona en «Mi bolsa»: responder al llamamiento abierto.
// El servidor decide con el catálogo de
// reglas (modo, plazo, situaciones admitidas, causas); aquí solo se recogen
// los datos, se calcula la huella del justificante y se muestra el recibo.
import { localizacionAreaPersonal, traducir } from "./i18n.js";
import { escaparAtributo, escaparHTML, listaDatos } from "./vistas/comunes.js";
import { cuerpoDisposicion, RUTA_DISPOSICION_MI_BOLSA, validarOfertasMiBolsa } from "./mi-bolsa-ofertas.js";
import { nombreCategoria } from "./mi-bolsa-campos.js";
import { cuerpoConfirmacionContacto, RUTA_CONTACTO_MI_BOLSA, validarContactosMiBolsa } from "./mi-bolsa-contacto.js";
import { crearClienteJustificante, MAX_PDF_JUSTIFICANTE } from "./mi-bolsa-justificante-cliente.js?v=20261015-o-justificante-bolsa-v2";

export const RUTAS_PORTAL_MI_BOLSA = Object.freeze({
  documental: "/api/vec/bolsa/mi-bolsa/solicitudes-documentales",
  responder: "/api/vec/bolsa/mi-bolsa/respuestas",
});


// Códigos de error del servidor con mensaje propio; los demás se muestran como
// servicio no disponible.
const CODIGOS_ERROR = new Set(["solicitud_pendiente", "situacion_no_admite", "sin_llamamiento_abierto", "fuera_de_plazo",
  "causa_no_admitida", "pausa_fuera_de_limite", "oferta_no_abierta", "disposicion_ya_manifestada", "contacto_cambiado",
  "contacto_ya_confirmado", "sin_contacto", "clave_reutilizada", "datos_no_validos", "acceso_denegado",
  "autenticacion_requerida", "servicio_no_disponible"]);

export function textoPortal(clave, variables = {}) {
  return traducir(`areaPersonal.portal.${clave}`, variables);
}

const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u;
const CAUSA = /^[a-z][a-z0-9_]{0,63}$/u;
const REFERENCIA_DOCUMENTAL = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,254}$/u;
const REFERENCIA_PROPIA_SISTEMA = /^[a-z_]+(:[a-z_]+)*:[0-9a-f]{64}$/u;
const DOCUMENTO_IDENTIDAD = /(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]/iu;
const ETIQUETA_IDENTIDAD = /(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)/iu;

function referenciaDocumentalValida(referencia) {
  return REFERENCIA_DOCUMENTAL.test(referencia) && !ETIQUETA_IDENTIDAD.test(referencia) &&
    (REFERENCIA_PROPIA_SISTEMA.test(referencia) || !DOCUMENTO_IDENTIDAD.test(referencia));
}

function instante(valor, nombre) {
  if (typeof valor !== "string" || !INSTANTE.test(valor) || Number.isNaN(Date.parse(valor))
    || new Date(valor).toISOString().slice(0, 19) !== valor.slice(0, 19)) {
    throw new TypeError(`${nombre} no es un instante válido.`);
  }
}

// validarPortalMiBolsa comprueba las dos partes opcionales de la respuesta de
// «Mi bolsa». Sin ellas el portal no ofrece acciones.
export function validarPortalMiBolsa(datos) {
  validarOfertasMiBolsa(datos);
  validarContactosMiBolsa(datos);
  if (datos.portal === undefined && datos.acciones_portal === undefined) return;
  const acciones = datos.acciones_portal;
  if (!acciones || typeof acciones !== "object" || !Array.isArray(datos.portal) ||
      !Array.isArray(acciones.causas_renuncia) || !acciones.causas_renuncia.every((c) => typeof c === "string" && CAUSA.test(c)) ||
      !["firme", "propuesta_rrhh"].includes(acciones.modo_respuesta)) {
    throw new TypeError("Las acciones de mi bolsa no son válidas.");
  }
  if (acciones.pausa_maxima !== null) instante(acciones.pausa_maxima, "pausa_maxima");
  const bolsas = new Set((datos.participaciones || []).map((p) => p.bolsa));
  for (const estado of datos.portal) {
    if (!estado || !bolsas.has(estado.bolsa)) throw new TypeError("El portal cita una bolsa ajena.");
    if (estado.llamamiento_abierto) {
      instante(estado.llamamiento_abierto.contacto_en, "contacto_en");
      if (estado.llamamiento_abierto.vence_antes_de !== null) instante(estado.llamamiento_abierto.vence_antes_de, "vence_antes_de");
    }
    if (estado.solicitud_pendiente) {
      if (!["pausa", "reactivacion", "documental_rrhh"].includes(estado.solicitud_pendiente.tipo) || typeof estado.solicitud_pendiente.recibo !== "string") throw new TypeError("Solicitud pendiente no válida.");
      instante(estado.solicitud_pendiente.registrada_en, "registrada_en");
    }
    if (estado.solicitud_documental_pendiente !== undefined && typeof estado.solicitud_documental_pendiente !== "boolean") throw new TypeError("Estado documental no válido.");
    if (estado.ultima_solicitud_documental) {
      const solicitud = estado.ultima_solicitud_documental;
      if (solicitud.tipo !== "documental_rrhh" || typeof solicitud.recibo !== "string" ||
          !["pendiente_rrhh", "validada", "rechazada"].includes(solicitud.estado) ||
          (solicitud.estado === "pendiente_rrhh") !== estado.solicitud_documental_pendiente ||
          (solicitud.estado === "pendiente_rrhh" ? solicitud.recibo_resolucion_ref !== null : typeof solicitud.recibo_resolucion_ref !== "string")) throw new TypeError("Solicitud documental no válida.");
      instante(solicitud.registrada_en, "registrada_en");
    } else if (estado.solicitud_documental_pendiente) throw new TypeError("Solicitud documental pendiente sin recibo.");
    if (estado.ultima_respuesta) {
      if (!["acepta", "renuncia", "renuncia_justificada"].includes(estado.ultima_respuesta.respuesta) || typeof estado.ultima_respuesta.recibo !== "string") throw new TypeError("Respuesta registrada no válida.");
      if (estado.ultima_respuesta.en_revision !== undefined && typeof estado.ultima_respuesta.en_revision !== "boolean") throw new TypeError("Revisión de respuesta no válida.");
      instante(estado.ultima_respuesta.respondida_en, "respondida_en");
    }
  }
}

function fecha(valor) {
  return new Intl.DateTimeFormat(localizacionAreaPersonal(), { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

function idSeguro(bolsa, sufijo) {
  return `portal-${String(bolsa).replace(/[^A-Za-z0-9_-]/gu, "-")}-${sufijo}`;
}

function formularioRespuesta(bolsa, acciones) {
  const id = (s) => idSeguro(bolsa, s);
  const obligatorio = `<span>${escaparHTML(traducir("areaPersonal.vista.comun.campoObligatorio"))}</span>`;
  const opciones = ["acepta", "renuncia", "renuncia_justificada"].map((valor) =>
    `<label class="opcion-check"><input type="radio" name="respuesta" value="${valor}" required><span>${escaparHTML(textoPortal(`respuesta.${valor}`))}</span></label>`).join("");
  const causas = acciones.causas_renuncia.map((c) => `<option value="${escaparAtributo(c)}">${escaparHTML(textoPortal(`causa.${c}`))}</option>`).join("");
  return `<form class="portal-mi-bolsa__formulario" data-portal-mi-bolsa="responder" data-bolsa="${escaparAtributo(bolsa)}" novalidate>
    <div data-portal-edicion><fieldset><legend>${escaparHTML(textoPortal("respuesta"))}</legend>${opciones}</fieldset>
    <div class="formulario-rejilla" data-portal-justificada hidden>
      <div class="campo"><label for="${id("causa")}">${escaparHTML(textoPortal("causa"))} ${obligatorio}</label><select id="${id("causa")}" name="causa"><option value=""></option>${causas}</select></div>
      <div class="campo"><label for="${id("justificante")}">${escaparHTML(textoPortal("justificante"))} ${obligatorio}</label><input id="${id("justificante")}" name="justificante" type="file" accept="application/pdf,.pdf"></div>
    </div></div>
    <section class="panel" data-portal-revision hidden tabindex="-1" aria-label="${escaparAtributo(textoPortal("revision.titulo"))}">
      <div class="cabecera-panel"><h4>${escaparHTML(textoPortal("revision.titulo"))}</h4></div>
      <div class="cuerpo-panel"><p>${escaparHTML(textoPortal("revision.aviso"))}</p><dl class="resumen-expediente" data-portal-revision-datos></dl>
      <button type="button" class="boton-secundario" data-portal-revision-volver>${escaparHTML(textoPortal("revision.volver"))}</button>
      <button type="submit" class="boton-primario">${escaparHTML(textoPortal("revision.confirmar"))}</button></div>
    </section>
    <button type="submit" class="boton-primario" data-portal-iniciar-revision>${escaparHTML(textoPortal("responder"))}</button>
    <p class="nota" role="status" aria-live="polite" tabindex="-1" data-portal-resultado></p>
    <button type="button" class="boton-secundario" data-portal-reintentar hidden>${escaparHTML(textoPortal("revision.reintentar"))}</button>
  </form>`;
}

function formularioDocumental(bolsa) {
  const id = (s) => idSeguro(bolsa, s);
  const obligatorio = `<span>${escaparHTML(traducir("areaPersonal.vista.comun.campoObligatorio"))}</span>`;
  const opcional = `<span>${escaparHTML(traducir("areaPersonal.vista.comun.campoOpcional"))}</span>`;
  return `<form class="portal-mi-bolsa__formulario" data-portal-mi-bolsa="documental" data-bolsa="${escaparAtributo(bolsa)}">
    <div class="campo"><label for="${id("fin-causa")}">${escaparHTML(textoPortal("documental.fechaFinCausa"))} ${opcional}</label><input id="${id("fin-causa")}" name="fecha_fin_causa" type="date"></div>
    <div class="campo"><label for="${id("documento-ref")}">${escaparHTML(textoPortal("documental.documentoRef"))} ${obligatorio}</label><input id="${id("documento-ref")}" name="documento_ref" maxlength="255" pattern="[A-Za-z0-9][A-Za-z0-9._:/#-]*" required aria-describedby="${id("documento-ref-ayuda")}"><small id="${id("documento-ref-ayuda")}">${escaparHTML(textoPortal("documental.documentoRefAyuda"))}</small></div>
    <div class="campo"><label for="${id("documento")}">${escaparHTML(textoPortal("documental.documento"))} ${obligatorio}</label><input id="${id("documento")}" name="documento" type="file" required aria-describedby="${id("documento-aviso")}"><small id="${id("documento-aviso")}">${escaparHTML(textoPortal("documental.documentoAviso"))}</small></div>
    <button type="submit" class="boton-primario">${escaparHTML(textoPortal("documental.enviar"))}</button>
    <p class="nota" role="status" aria-live="polite" data-portal-resultado></p>
  </form>`;
}

// renderizarPortalMiBolsa pinta, por bolsa, lo que la persona puede hacer.
export function renderizarPortalMiBolsa(participaciones, portal, acciones) {
  const porBolsa = new Map((portal || []).map((e) => [e.bolsa, e]));
  const bloques = (participaciones || []).map((p) => {
    const estado = porBolsa.get(p.bolsa) || {};
    const partes = [];
    if (estado.solicitud_pendiente) {
      const s = estado.solicitud_pendiente;
      partes.push(`<p class="nota aviso">${escaparHTML(textoPortal("pendiente", { tipo: textoPortal(`tipo.${s.tipo}`) }))} · ${escaparHTML(textoPortal("recibo"))} ${escaparHTML(s.recibo)}</p>`);
    }
    if (estado.ultima_solicitud_documental) {
      const s = estado.ultima_solicitud_documental;
      partes.push(listaDatos([[textoPortal("documental.ultima"), `${escaparHTML(textoPortal(`documental.estado.${s.estado}`))} · ${escaparHTML(fecha(s.registrada_en))} · ${escaparHTML(textoPortal("recibo"))} ${escaparHTML(s.recibo)}`],
        ...(s.recibo_resolucion_ref ? [[textoPortal("documental.reciboResolucion"), escaparHTML(s.recibo_resolucion_ref)]] : [])]));
    }
    if ((!p.vigente_hasta || Date.parse(p.vigente_hasta) > Date.now()) && !estado.solicitud_pendiente && !estado.solicitud_documental_pendiente) {
      partes.push(formularioDocumental(p.bolsa));
    }
    if (estado.llamamiento_abierto) {
      const a = estado.llamamiento_abierto;
      partes.push(`<section class="panel"><header><h4>${escaparHTML(textoPortal("llamamiento"))}</h4></header><div class="panel-contenido">${listaDatos([
        [textoPortal("contacto"), escaparHTML(fecha(a.contacto_en))],
        [textoPortal("vence"), escaparHTML(a.vence_antes_de ? fecha(a.vence_antes_de) : textoPortal("venceNoDisponible"))],
      ])}<p><span class="estado-chip info">${escaparHTML(textoPortal(`modo.${acciones.modo_respuesta}`))}</span></p>${a.vence_antes_de ? formularioRespuesta(p.bolsa, acciones) : ""}</div></section>`);
    }
    if (estado.ultima_respuesta) {
      const r = estado.ultima_respuesta;
      const titulo = textoPortal(estado.llamamiento_abierto ? "respuestaAnterior" : "ultimaRespuestaBolsa");
      partes.push(`<section class="panel"><header><h4>${escaparHTML(titulo)}</h4></header><div class="panel-contenido">${listaDatos([
        [textoPortal("respuesta"), escaparHTML(textoPortal(`respuesta.${r.respuesta}`))],
        [textoPortal("fechaRespuesta"), escaparHTML(fecha(r.respondida_en))],
        [textoPortal("modoRespuesta"), escaparHTML(textoPortal(`modo.${r.modo}`))],
        [textoPortal("recibo"), escaparHTML(r.recibo)],
      ])}</div></section>`);
    }
    return `<article class="portal-mi-bolsa__bolsa"><h3>${escaparHTML(nombreCategoria(p))}</h3>${partes.join("")}</article>`;
  }).join("");
  const gestion = `<p class="nota">${escaparHTML(textoPortal("documental.limite"))}</p>`;
  return `<div class="portal-mi-bolsa">${bloques}${gestion}</div>`;
}

async function huellaSHA256(fichero) {
  const bytes = await fichero.arrayBuffer();
  const resumen = await globalThis.crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(resumen), (b) => b.toString(16).padStart(2, "0")).join("");
}

function claveIdempotencia(formulario) {
  // La clave se conserva en el formulario: un reintento tras un corte repite
  // la misma petición y el servidor devuelve el recibo original.
  formulario.dataset.clave ||= `portal-${globalThis.crypto.randomUUID()}`;
  return formulario.dataset.clave;
}

export async function cuerpoPortalMiBolsa(formulario, datos = new FormData(formulario)) {
  if (formulario.dataset.portalMiBolsa === "disposicion") return cuerpoDisposicion(formulario);
  if (formulario.dataset.portalMiBolsa === "contacto") return cuerpoConfirmacionContacto(formulario);
  const bolsa = formulario.dataset.bolsa;
  if (formulario.dataset.portalMiBolsa === "documental") {
    const documento = datos.get("documento");
    const documentoRef = String(datos.get("documento_ref") || "").trim();
    const fechaFinCausa = String(datos.get("fecha_fin_causa") || "");
    if (!referenciaDocumentalValida(documentoRef) || (fechaFinCausa && (!/^\d{4}-\d{2}-\d{2}$/u.test(fechaFinCausa) ||
        Number.isNaN(Date.parse(`${fechaFinCausa}T00:00:00Z`)) || new Date(`${fechaFinCausa}T00:00:00Z`).toISOString().slice(0, 10) !== fechaFinCausa)) ||
        !(documento instanceof Blob) || documento.size === 0) return null;
    const cuerpo = {
      tipo: "documental_rrhh", bolsa, documento_ref: documentoRef,
      documento_sha256: await huellaSHA256(documento),
      clave: claveIdempotencia(formulario),
    };
    if (fechaFinCausa) cuerpo.fecha_fin_causa = fechaFinCausa;
    return { ruta: RUTAS_PORTAL_MI_BOLSA.documental, cuerpo };
  }
  if (formulario.dataset.portalMiBolsa !== "responder") return null;
  const respuesta = String(datos.get("respuesta") || "");
  const cuerpo = { bolsa, respuesta, clave: claveIdempotencia(formulario) };
  if (respuesta === "renuncia_justificada") {
    const fichero = datos.get("justificante");
    const causa = String(datos.get("causa") || "");
    if (!CAUSA.test(causa) || !(fichero instanceof Blob) || fichero.size === 0
      || !/\.pdf$/iu.test(fichero.name || "") || (fichero.type && fichero.type !== "application/pdf")) return null;
    if (fichero.size > MAX_PDF_JUSTIFICANTE) return null;
    Object.assign(cuerpo, { causa, justificante_sha256: await huellaSHA256(fichero) });
    return { ruta: RUTAS_PORTAL_MI_BOLSA.responder, cuerpo, fichero, claveSubida: `${cuerpo.clave}:justificante` };
  } else if (!["acepta", "renuncia"].includes(respuesta)) {
    return null;
  }
  return { ruta: RUTAS_PORTAL_MI_BOLSA.responder, cuerpo };
}

const peticionesInciertas = new WeakMap();
const enviosEnCurso = new WeakSet();

const respuestasRevisadas = new WeakMap();
const revisionesEnCurso = new WeakSet();

export function actualizarCamposRenunciaJustificada(formulario) {
  if (formulario?.dataset?.portalMiBolsa !== "responder" || peticionesInciertas.has(formulario)
    || respuestasRevisadas.has(formulario)) return;
  const grupo = formulario.querySelector("[data-portal-justificada]");
  if (!grupo) return;
  const justificada = formulario.querySelector('[name="respuesta"]:checked')?.value === "renuncia_justificada";
  grupo.hidden = !justificada;
  for (const nombre of ["causa", "justificante"]) {
    const campo = formulario.querySelector(`[name="${nombre}"]`);
    if (campo) campo.required = justificada;
  }
}

// La operación revisada queda fijada antes de mostrar el resumen: una edición
// posterior del DOM no puede transformar una renuncia confirmada en aceptación.
export async function revisarRespuestaPortalMiBolsa(formulario, datos = new FormData(formulario)) {
  if (formulario.dataset.portalMiBolsa !== "responder") return true;
  const revision = formulario.querySelector("[data-portal-revision]");
  if (!revision || !revision.hidden) return true;
  if (revisionesEnCurso.has(formulario)) return false;
  const respuesta = String(datos.get("respuesta") || "");
  if (respuesta === "acepta") return true;
  if (!["renuncia", "renuncia_justificada"].includes(respuesta)) {
    formulario.querySelector("[data-portal-resultado]").textContent = textoPortal("error.datos_no_validos");
    return false;
  }
  const filas = [[textoPortal("respuesta"), textoPortal(`respuesta.${respuesta}`)]];
  if (respuesta === "renuncia_justificada") {
    const causa = String(datos.get("causa") || "");
    const fichero = datos.get("justificante");
    if (!CAUSA.test(causa)) {
      formulario.querySelector("[data-portal-resultado]").textContent = textoPortal("error.datos_no_validos");
      return false;
    }
    if (!(fichero instanceof Blob) || fichero.size === 0
      || !/\.pdf$/iu.test(fichero.name || "") || (fichero.type && fichero.type !== "application/pdf")) {
      formulario.querySelector("[data-portal-resultado]").textContent = textoPortal("justificanteNoValido");
      return false;
    }
    if (fichero.size > MAX_PDF_JUSTIFICANTE) {
      formulario.querySelector("[data-portal-resultado]").textContent = textoPortal("justificanteDemasiadoGrande");
      return false;
    }
    filas.push([textoPortal("causa"), textoPortal(`causa.${causa}`)],
      [textoPortal("revision.nombreArchivo"), fichero.name || textoPortal("revision.archivoSeleccionado")]);
  }
  revisionesEnCurso.add(formulario);
  try {
    const copia = new FormData();
    for (const [nombre, valor] of datos) copia.append(nombre, valor);
    const peticion = await cuerpoPortalMiBolsa(formulario, copia);
    if (!peticion) return false;
    respuestasRevisadas.set(formulario, peticion);
  } finally {
    revisionesEnCurso.delete(formulario);
  }
  formulario.querySelector("[data-portal-resultado]").textContent = "";
  formulario.querySelector("[data-portal-revision-datos]").innerHTML = filas.map(([etiqueta, valor]) =>
    `<div class="fila-resumen"><dt>${escaparHTML(etiqueta)}</dt><dd>${escaparHTML(valor)}</dd></div>`).join("");
  formulario.querySelector("[data-portal-edicion]").hidden = true;
  formulario.querySelector("[data-portal-iniciar-revision]").hidden = true;
  revision.hidden = false;
  revision.focus();
  return false;
}

export function volverRespuestaPortalMiBolsa(formulario) {
  if (peticionesInciertas.has(formulario) || enviosEnCurso.has(formulario)) return false;
  respuestasRevisadas.delete(formulario);
  delete formulario.dataset.clave;
  formulario.querySelector("[data-portal-revision]").hidden = true;
  formulario.querySelector("[data-portal-edicion]").hidden = false;
  formulario.querySelector("[data-portal-iniciar-revision]").hidden = false;
  actualizarCamposRenunciaJustificada(formulario);
  formulario.querySelector('[name="respuesta"]:checked')?.focus();
  return true;
}

function bloquearDatosDocumentales(formulario, bloqueados) {
  for (const campo of formulario.querySelectorAll?.('input[name="fecha_fin_causa"], input[name="documento_ref"], input[name="documento"]') || []) {
    campo.disabled = bloqueados;
  }
}

function instanteRecibo(valor, nombre) {
  instante(valor, nombre);
  // Date.parse normaliza días inexistentes y 24:00. El recibo exige la fecha
  // civil exacta; comparar hasta el segundo conserva los microsegundos recibidos.
  if (new Date(valor).toISOString().slice(0, 19) !== valor.slice(0, 19)) throw new TypeError();
}

// Este recibo es el de httppersonal/reciboPortal; el recibo del panel del área
// personal usa otro esquema. Solo contacto y disposición devuelven el recurso
// de la petición; las respuestas devuelven una referencia opaca nueva.
function validarReciboPortal(entrada, peticion, estadoHTTP) {
  const contratos = {
    [RUTAS_PORTAL_MI_BOLSA.documental]: ["solicitud-documental", /^solicitud-documental:[a-f0-9]{64}$/u, "solicitud-documental", ["pendiente_rrhh", "validada", "rechazada"]],
    [RUTAS_PORTAL_MI_BOLSA.responder]: ["respuesta", /^respuesta-portal:[a-f0-9]{64}$/u, "respuesta-portal", ["firme", "propuesta_rrhh"]],
    [RUTA_DISPOSICION_MI_BOLSA]: ["disposicion", peticion.cuerpo.oferta, "disposicion", ["manifestada"]],
    [RUTA_CONTACTO_MI_BOLSA]: ["contacto", peticion.cuerpo.bolsa, "confirmacion-contacto", ["confirmado"]],
  };
  const contrato = contratos[peticion.ruta];
  const recibo = entrada?.data;
  if (!contrato || !recibo || typeof recibo !== "object" || Array.isArray(recibo)) throw new TypeError();
  const [tipo, referencia, prefijo, estados] = contrato;
  if (recibo.esquema !== `vec.bolsa.mi-bolsa.${tipo}.v1` ||
      typeof recibo.referencia !== "string" ||
      !(referencia instanceof RegExp ? referencia.test(recibo.referencia) : recibo.referencia === referencia) ||
      typeof recibo.recibo !== "string" || !new RegExp(`^recibo:${prefijo}:[a-f0-9]{64}$`, "u").test(recibo.recibo) ||
      !estados.includes(recibo.estado) || typeof recibo.repetida !== "boolean" || recibo.repetida !== (estadoHTTP === 200)) throw new TypeError();
  instanteRecibo(recibo.registrada_en, "registrada_en");
  if (tipo === "solicitud-documental") {
    if (!/^[a-f0-9]{64}$/u.test(recibo.contenido_sha256 || "") || recibo.version !== 1 ||
        (!recibo.repetida && recibo.estado !== "pendiente_rrhh")) throw new TypeError();
  }
  if (recibo.vence_antes_de !== undefined) {
    if (tipo !== "respuesta") throw new TypeError();
    instanteRecibo(recibo.vence_antes_de, "vence_antes_de");
  }
  return recibo;
}

// enviarPortalMiBolsa envía el formulario y deja el resultado en su zona de
// estado; tras un registro correcto pide recargar «Mi bolsa».
export async function enviarPortalMiBolsa(formulario, { fetchImpl = globalThis.fetch, alRegistrar = () => {}, datos, reintentoExplicito = false } = {}) {
  if (peticionesInciertas.has(formulario) && respuestasRevisadas.has(formulario) && !reintentoExplicito) return false;
  if (enviosEnCurso.has(formulario)) return false;
  enviosEnCurso.add(formulario);
  const zona = formulario.querySelector("[data-portal-resultado]");
  const boton = formulario.querySelector("[data-portal-revision]")?.hidden
    ? formulario.querySelector("[data-portal-iniciar-revision]") : formulario.querySelector('button[type="submit"]');
  const volver = formulario.querySelector("[data-portal-revision-volver]");
  const reintentar = formulario.querySelector("[data-portal-reintentar]");
  const mostrar = (texto) => { if (zona) zona.textContent = texto; };
  let seEnvio = false;
  let confirmado = false;
  let rechazoDefinitivo = false;
  let mensajeSubida = "";
  try {
    // Un resultado incierto puede haber persistido: un reintento explícito usa
    // el mismo cuerpo, incluida la versión, aunque se hayan editado los campos.
    if (!peticionesInciertas.has(formulario) && formulario.dataset.portalMiBolsa === "documental") {
      datos ||= new FormData(formulario);
      if (!referenciaDocumentalValida(String(datos.get("documento_ref") || "").trim())) {
        const mensaje = textoPortal("documental.referenciaNoValida");
        const campo = formulario.querySelector('[name="documento_ref"]');
        campo?.setCustomValidity?.(mensaje);
        campo?.reportValidity?.();
        campo?.focus?.();
        campo?.addEventListener?.("input", () => campo.setCustomValidity(""), { once: true });
        mostrar(mensaje);
        return false;
      }
    }
    const peticion = peticionesInciertas.get(formulario) || respuestasRevisadas.get(formulario)
      || await cuerpoPortalMiBolsa(formulario, datos);
    if (!peticion) {
      mostrar(textoPortal("error.datos_no_validos"));
      return false;
    }
    peticionesInciertas.set(formulario, peticion);
    if (peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental) bloquearDatosDocumentales(formulario, true);
    if (boton) boton.disabled = true;
    if (reintentar) reintentar.disabled = true;
    mostrar(textoPortal("enviando"));
    seEnvio = true;
    if (peticion.fichero && !peticion.cuerpo.justificante_ref) {
      const cliente = crearClienteJustificante({ fetchImpl });
      const encontrada = await cliente.consultar(peticion.cuerpo.bolsa, peticion.claveSubida);
      const subida = encontrada?.estado === "admitido" || encontrada?.estado === "rechazado"
        ? encontrada : await cliente.subir(peticion.cuerpo.bolsa, peticion.claveSubida, peticion.fichero);
      if (subida.estado === "rechazado") {
        peticionesInciertas.delete(formulario);
        respuestasRevisadas.delete(formulario);
        rechazoDefinitivo = true;
        mensajeSubida = textoPortal("justificanteRechazado");
        mostrar(mensajeSubida);
        return false;
      }
      if (subida.estado !== "admitido") {
        mensajeSubida = textoPortal("justificantePendiente");
        mostrar(mensajeSubida);
        return false;
      }
      peticion.cuerpo.justificante_ref = subida.comprobante_ref;
    }
    const respuesta = await fetchImpl(peticion.ruta, {
      method: "POST", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(peticion.cuerpo),
    });
    const resultado = await respuesta.json().catch(() => null);
    if (respuesta.status === 200 || respuesta.status === 201) {
      const recibo = validarReciboPortal(resultado, peticion, respuesta.status);
      peticionesInciertas.delete(formulario);
      respuestasRevisadas.delete(formulario);
      if (peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental) bloquearDatosDocumentales(formulario, false);
      mostrar(textoPortal(peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental ? `documental.resultado.${recibo.estado}` : "hecho", { recibo: recibo.recibo }));
      alRegistrar();
      confirmado = true;
      return true;
    }
    const codigo = String(resultado?.error?.codigo || "servicio_no_disponible");
    if (respuesta.status >= 400 && respuesta.status < 500 && CODIGOS_ERROR.has(codigo)) {
      peticionesInciertas.delete(formulario);
      rechazoDefinitivo = true;
      if (peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental) bloquearDatosDocumentales(formulario, false);
    }
    mostrar(peticionesInciertas.has(formulario) && [RUTAS_PORTAL_MI_BOLSA.documental, RUTAS_PORTAL_MI_BOLSA.responder].includes(peticion.ruta)
      ? textoPortal(peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental ? "documental.errorIncierto" : "error.servicio_no_disponible")
      : textoPortal(`error.${CODIGOS_ERROR.has(codigo) ? codigo : "servicio_no_disponible"}`));
    return false;
  } catch (error) {
    const ruta = peticionesInciertas.get(formulario)?.ruta;
    if (error?.estado === 413) {
      peticionesInciertas.delete(formulario);
      respuestasRevisadas.delete(formulario);
      rechazoDefinitivo = true;
    }
    mostrar(error?.estado === 413 ? textoPortal("justificanteDemasiadoGrande") : mensajeSubida || textoPortal(ruta === RUTAS_PORTAL_MI_BOLSA.documental ? "documental.errorIncierto"
      : peticionesInciertas.get(formulario)?.fichero && !peticionesInciertas.get(formulario)?.cuerpo.justificante_ref
        ? "justificanteSinConfirmar" : "error.servicio_no_disponible"));
    return false;
  } finally {
    enviosEnCurso.delete(formulario);
    const requiereReintento = seEnvio && !confirmado && respuestasRevisadas.has(formulario)
      && peticionesInciertas.has(formulario);
    const volverParaCorregir = rechazoDefinitivo && respuestasRevisadas.has(formulario);
    if (boton) {
      boton.disabled = requiereReintento || volverParaCorregir;
      if (requiereReintento || volverParaCorregir) boton.hidden = true;
    }
    if (reintentar) { reintentar.hidden = !requiereReintento; reintentar.disabled = !requiereReintento; }
    if (volver) volver.disabled = peticionesInciertas.has(formulario);
    if (requiereReintento || volverParaCorregir) zona?.focus?.();
  }
}
