// Acciones propias de la persona en «Mi bolsa»: responder al llamamiento abierto.
// El servidor decide con el catálogo de
// reglas (modo, plazo, situaciones admitidas, causas); aquí solo se recogen
// los datos, se calcula la huella del justificante y se muestra el recibo.
import { localizacionAreaPersonal, traducir } from "./i18n.js";
import { escaparAtributo, escaparHTML, listaDatos } from "./vistas/comunes.js";
import { cuerpoDisposicion, RUTA_DISPOSICION_MI_BOLSA, validarOfertasMiBolsa } from "./mi-bolsa-ofertas.js";
import { nombreCategoria } from "./mi-bolsa-campos.js";
import { cuerpoConfirmacionContacto, RUTA_CONTACTO_MI_BOLSA, validarContactosMiBolsa } from "./mi-bolsa-contacto.js";

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
  const opciones = ["acepta", "renuncia", "renuncia_justificada"].map((valor) =>
    `<label class="opcion-check"><input type="radio" name="respuesta" value="${valor}" required><span>${escaparHTML(textoPortal(`respuesta.${valor}`))}</span></label>`).join("");
  const causas = acciones.causas_renuncia.map((c) => `<option value="${escaparAtributo(c)}">${escaparHTML(textoPortal(`causa.${c}`))}</option>`).join("");
  return `<form class="portal-mi-bolsa__formulario" data-portal-mi-bolsa="responder" data-bolsa="${escaparAtributo(bolsa)}" novalidate>
    <fieldset><legend>${escaparHTML(textoPortal("respuesta"))}</legend>${opciones}</fieldset>
    <div class="formulario-rejilla" data-portal-justificada>
      <div class="campo"><label for="${id("causa")}">${escaparHTML(textoPortal("causa"))}</label><select id="${id("causa")}" name="causa"><option value=""></option>${causas}</select></div>
      <div class="campo"><label for="${id("justificante")}">${escaparHTML(textoPortal("justificante"))}</label><input id="${id("justificante")}" name="justificante" type="file"></div>
      <div class="campo"><label for="${id("referencia")}">${escaparHTML(textoPortal("justificanteRef"))}</label><input id="${id("referencia")}" name="justificante_ref" maxlength="128" pattern="[A-Za-z0-9][A-Za-z0-9._:/#-]*" autocomplete="off"></div>
    </div>
    <button type="submit" class="boton-primario">${escaparHTML(textoPortal("responder"))}</button>
    <p class="nota" role="status" aria-live="polite" data-portal-resultado></p>
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
      partes.push(`<h4>${escaparHTML(textoPortal("llamamiento"))}</h4>${listaDatos([
        [textoPortal("contacto"), escaparHTML(fecha(a.contacto_en))],
        [textoPortal("vence"), escaparHTML(a.vence_antes_de ? fecha(a.vence_antes_de) : textoPortal("venceNoDisponible"))],
      ])}<p><span class="estado-chip info">${escaparHTML(textoPortal(`modo.${acciones.modo_respuesta}`))}</span></p>${a.vence_antes_de ? formularioRespuesta(p.bolsa, acciones) : ""}`);
    }
    if (estado.ultima_respuesta) {
      const r = estado.ultima_respuesta;
      partes.push(listaDatos([[textoPortal("ultimaRespuesta"), `${escaparHTML(textoPortal(`respuesta.${r.respuesta}`))} · ${escaparHTML(fecha(r.respondida_en))} · ${escaparHTML(textoPortal(`modo.${r.modo}`))} · ${escaparHTML(textoPortal("recibo"))} ${escaparHTML(r.recibo)}`]]));
    }
    return `<article class="portal-mi-bolsa__bolsa"><h4>${escaparHTML(nombreCategoria(p))}</h4>${partes.join("")}</article>`;
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
    const referencia = String(datos.get("justificante_ref") || "").trim();
    const causa = String(datos.get("causa") || "");
    if (!causa || !referencia || !(fichero instanceof Blob) || fichero.size === 0) return null;
    Object.assign(cuerpo, { causa, justificante_ref: referencia, justificante_sha256: await huellaSHA256(fichero) });
  } else if (!["acepta", "renuncia"].includes(respuesta)) {
    return null;
  }
  return { ruta: RUTAS_PORTAL_MI_BOLSA.responder, cuerpo };
}

const peticionesInciertas = new WeakMap();
const enviosEnCurso = new WeakSet();

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
export async function enviarPortalMiBolsa(formulario, { fetchImpl = globalThis.fetch, alRegistrar = () => {}, datos } = {}) {
  if (enviosEnCurso.has(formulario)) return false;
  enviosEnCurso.add(formulario);
  const zona = formulario.querySelector("[data-portal-resultado]");
  const boton = formulario.querySelector('button[type="submit"]');
  const mostrar = (texto) => { if (zona) zona.textContent = texto; };
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
    const peticion = peticionesInciertas.get(formulario) || await cuerpoPortalMiBolsa(formulario, datos);
    if (!peticion) {
      mostrar(textoPortal("error.datos_no_validos"));
      return false;
    }
    peticionesInciertas.set(formulario, peticion);
    if (peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental) bloquearDatosDocumentales(formulario, true);
    if (boton) boton.disabled = true;
    mostrar(textoPortal("enviando"));
    const respuesta = await fetchImpl(peticion.ruta, {
      method: "POST", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(peticion.cuerpo),
    });
    const resultado = await respuesta.json().catch(() => null);
    if (respuesta.status === 200 || respuesta.status === 201) {
      const recibo = validarReciboPortal(resultado, peticion, respuesta.status);
      peticionesInciertas.delete(formulario);
      if (peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental) bloquearDatosDocumentales(formulario, false);
      mostrar(textoPortal(peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental ? `documental.resultado.${recibo.estado}` : "hecho", { recibo: recibo.recibo }));
      alRegistrar();
      return true;
    }
    const codigo = String(resultado?.error?.codigo || "servicio_no_disponible");
    if (respuesta.status >= 400 && respuesta.status < 500 && CODIGOS_ERROR.has(codigo)) {
      peticionesInciertas.delete(formulario);
      if (peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental) bloquearDatosDocumentales(formulario, false);
    }
    mostrar(peticionesInciertas.has(formulario) && peticion.ruta === RUTAS_PORTAL_MI_BOLSA.documental
      ? textoPortal("documental.errorIncierto")
      : textoPortal(`error.${CODIGOS_ERROR.has(codigo) ? codigo : "servicio_no_disponible"}`));
    return false;
  } catch {
    mostrar(peticionesInciertas.get(formulario)?.ruta === RUTAS_PORTAL_MI_BOLSA.documental
      ? textoPortal("documental.errorIncierto") : textoPortal("error.servicio_no_disponible"));
    return false;
  } finally {
    enviosEnCurso.delete(formulario);
    if (boton) boton.disabled = false;
  }
}
