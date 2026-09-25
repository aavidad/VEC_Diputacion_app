// Acciones propias de la persona en «Mi bolsa»: pedir pausa o reactivación y
// responder al llamamiento abierto. El servidor decide con el catálogo de
// reglas (modo, plazo, situaciones admitidas, causas); aquí solo se recogen
// los datos, se calcula la huella del justificante y se muestra el recibo.
import { traducir } from "./i18n.js";
import { escaparAtributo, escaparHTML, listaDatos } from "./vistas/comunes.js";

export const RUTAS_PORTAL_MI_BOLSA = Object.freeze({
  solicitar: "/api/vec/bolsa/mi-bolsa/solicitudes",
  responder: "/api/vec/bolsa/mi-bolsa/respuestas",
});

const RESPALDO = Object.freeze({
  "areaPersonal.portal.titulo": "Disponibilidad y respuesta",
  "areaPersonal.portal.subtitulo": "Por bolsa",
  "areaPersonal.portal.pausa": "Pedir una pausa hasta",
  "areaPersonal.portal.pausaEnviar": "Pedir pausa",
  "areaPersonal.portal.reactivar": "Pedir la reactivación",
  "areaPersonal.portal.pendiente": "Solicitud de {tipo} pendiente de RRHH",
  "areaPersonal.portal.tipo.pausa": "pausa",
  "areaPersonal.portal.tipo.reactivacion": "reactivación",
  "areaPersonal.portal.recibo": "Recibo",
  "areaPersonal.portal.llamamiento": "Llamamiento abierto",
  "areaPersonal.portal.contacto": "Contacto",
  "areaPersonal.portal.vence": "Responder antes de",
  "areaPersonal.portal.venceNoDisponible": "Plazo no disponible",
  "areaPersonal.portal.respuesta": "Mi respuesta",
  "areaPersonal.portal.respuesta.acepta": "Acepto",
  "areaPersonal.portal.respuesta.renuncia": "Renuncio",
  "areaPersonal.portal.respuesta.renuncia_justificada": "Renuncio con causa justificada",
  "areaPersonal.portal.causa": "Causa",
  "areaPersonal.portal.causa.enfermedad": "Enfermedad",
  "areaPersonal.portal.causa.maternidad_paternidad_adopcion": "Maternidad, paternidad o adopción",
  "areaPersonal.portal.causa.alta_seguridad_social": "Alta en la Seguridad Social",
  "areaPersonal.portal.causa.matrimonio_union_hecho": "Matrimonio o unión de hecho",
  "areaPersonal.portal.justificante": "Justificante",
  "areaPersonal.portal.justificanteRef": "Referencia del justificante",
  "areaPersonal.portal.responder": "Enviar respuesta",
  "areaPersonal.portal.modo.propuesta_rrhh": "RRHH confirmará la respuesta",
  "areaPersonal.portal.modo.firme": "Respuesta firme",
  "areaPersonal.portal.ultimaRespuesta": "Respuesta registrada",
  "areaPersonal.portal.enviando": "Enviando…",
  "areaPersonal.portal.hecho": "Registrado. Recibo {recibo}.",
  "areaPersonal.portal.error.solicitud_pendiente": "Ya tiene una solicitud pendiente de RRHH en esta bolsa.",
  "areaPersonal.portal.error.situacion_no_admite": "Su situación actual en la bolsa no admite esta solicitud.",
  "areaPersonal.portal.error.sin_llamamiento_abierto": "No hay un llamamiento abierto que responder.",
  "areaPersonal.portal.error.fuera_de_plazo": "El plazo de respuesta ha terminado.",
  "areaPersonal.portal.error.causa_no_admitida": "La causa indicada no está admitida.",
  "areaPersonal.portal.error.pausa_fuera_de_limite": "La fecha de fin de la pausa no está permitida.",
  "areaPersonal.portal.error.clave_reutilizada": "Esta petición ya se envió con otros datos. Recargue la página.",
  "areaPersonal.portal.error.datos_no_validos": "Revise los datos del formulario.",
  "areaPersonal.portal.error.acceso_denegado": "No tiene permiso para esta acción.",
  "areaPersonal.portal.error.autenticacion_requerida": "Identifíquese de nuevo para continuar.",
  "areaPersonal.portal.error.servicio_no_disponible": "Servicio no disponible. No se ha registrado nada; puede volver a intentarlo.",
});

export function textoPortal(clave, variables = {}) {
  const completa = `areaPersonal.portal.${clave}`;
  const traducido = traducir(completa, variables);
  if (traducido !== completa) return traducido;
  return (RESPALDO[completa] ?? completa).replace(/\{([a-z_]+)\}/giu, (_, nombre) => String(variables[nombre] ?? ""));
}

const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u;
const CAUSA = /^[a-z][a-z0-9_]{0,63}$/u;

function instante(valor, nombre) {
  if (typeof valor !== "string" || !INSTANTE.test(valor) || Number.isNaN(Date.parse(valor))) throw new TypeError(`${nombre} no es un instante válido.`);
}

// validarPortalMiBolsa comprueba las dos partes opcionales de la respuesta de
// «Mi bolsa». Sin ellas el portal no ofrece acciones.
export function validarPortalMiBolsa(datos) {
  if (datos.portal === undefined && datos.acciones_portal === undefined) return;
  const acciones = datos.acciones_portal;
  if (!acciones || typeof acciones !== "object" || !Array.isArray(datos.portal) ||
      !Array.isArray(acciones.causas_renuncia) || !acciones.causas_renuncia.every((c) => typeof c === "string" && CAUSA.test(c)) ||
      !["firme", "propuesta_rrhh"].includes(acciones.modo_respuesta)) {
    throw new TypeError("Las acciones de mi bolsa no son válidas.");
  }
  instante(acciones.pausa_maxima, "pausa_maxima");
  const bolsas = new Set((datos.participaciones || []).map((p) => p.bolsa));
  for (const estado of datos.portal) {
    if (!estado || !bolsas.has(estado.bolsa)) throw new TypeError("El portal cita una bolsa ajena.");
    if (estado.llamamiento_abierto) {
      instante(estado.llamamiento_abierto.contacto_en, "contacto_en");
      if (estado.llamamiento_abierto.vence_antes_de !== null) instante(estado.llamamiento_abierto.vence_antes_de, "vence_antes_de");
    }
    if (estado.solicitud_pendiente) {
      if (!["pausa", "reactivacion"].includes(estado.solicitud_pendiente.tipo) || typeof estado.solicitud_pendiente.recibo !== "string") throw new TypeError("Solicitud pendiente no válida.");
      instante(estado.solicitud_pendiente.registrada_en, "registrada_en");
    }
    if (estado.ultima_respuesta) {
      if (!["acepta", "renuncia", "renuncia_justificada"].includes(estado.ultima_respuesta.respuesta) || typeof estado.ultima_respuesta.recibo !== "string") throw new TypeError("Respuesta registrada no válida.");
      instante(estado.ultima_respuesta.respondida_en, "respondida_en");
    }
  }
}

function fecha(valor) {
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

function fechaISO(valor) {
  // Fecha civil peninsular (AAAA-MM-DD) para los límites del selector.
  const partes = new Intl.DateTimeFormat("en-CA", { timeZone: "Europe/Madrid", year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(new Date(valor));
  const parte = (tipo) => partes.find((p) => p.type === tipo)?.value;
  return `${parte("year")}-${parte("month")}-${parte("day")}`;
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

// renderizarPortalMiBolsa pinta, por bolsa, lo que la persona puede hacer.
export function renderizarPortalMiBolsa(participaciones, portal, acciones) {
  const porBolsa = new Map((portal || []).map((e) => [e.bolsa, e]));
  const maxima = fechaISO(acciones.pausa_maxima);
  const manana = fechaISO(Date.now() + 24 * 3600 * 1000);
  const bloques = (participaciones || []).map((p) => {
    const estado = porBolsa.get(p.bolsa) || {};
    const id = (s) => idSeguro(p.bolsa, s);
    const partes = [];
    if (estado.solicitud_pendiente) {
      const s = estado.solicitud_pendiente;
      partes.push(`<p class="nota aviso">${escaparHTML(textoPortal("pendiente", { tipo: textoPortal(`tipo.${s.tipo}`) }))} · ${escaparHTML(textoPortal("recibo"))} ${escaparHTML(s.recibo)}</p>`);
    } else {
      partes.push(`<form class="portal-mi-bolsa__formulario" data-portal-mi-bolsa="solicitar" data-tipo="pausa" data-bolsa="${escaparAtributo(p.bolsa)}">
        <div class="campo"><label for="${id("pausa")}">${escaparHTML(textoPortal("pausa"))}</label><input id="${id("pausa")}" name="hasta" type="date" required min="${manana}" max="${maxima}"></div>
        <button type="submit" class="boton-secundario">${escaparHTML(textoPortal("pausaEnviar"))}</button>
        <p class="nota" role="status" aria-live="polite" data-portal-resultado></p></form>
        <form class="portal-mi-bolsa__formulario" data-portal-mi-bolsa="solicitar" data-tipo="reactivacion" data-bolsa="${escaparAtributo(p.bolsa)}">
        <button type="submit" class="boton-secundario">${escaparHTML(textoPortal("reactivar"))}</button>
        <p class="nota" role="status" aria-live="polite" data-portal-resultado></p></form>`);
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
    return `<article class="portal-mi-bolsa__bolsa"><h4>${escaparHTML(p.categoria)}</h4>${partes.join("")}</article>`;
  }).join("");
  return `<div class="portal-mi-bolsa">${bloques}</div>`;
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
  const bolsa = formulario.dataset.bolsa;
  if (formulario.dataset.portalMiBolsa === "solicitar") {
    const tipo = formulario.dataset.tipo;
    const cuerpo = { tipo, bolsa, clave: claveIdempotencia(formulario) };
    if (tipo === "pausa") {
      const dia = String(datos.get("hasta") || "");
      if (!/^\d{4}-\d{2}-\d{2}$/u.test(dia)) return null;
      // Hasta el final del día elegido, hora local del navegador.
      cuerpo.pausa_hasta = new Date(`${dia}T23:59:59`).toISOString();
    }
    return { ruta: RUTAS_PORTAL_MI_BOLSA.solicitar, cuerpo };
  }
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

// enviarPortalMiBolsa envía el formulario y deja el resultado en su zona de
// estado; tras un registro correcto pide recargar «Mi bolsa».
export async function enviarPortalMiBolsa(formulario, { fetchImpl = globalThis.fetch, alRegistrar = () => {}, datos } = {}) {
  const zona = formulario.querySelector("[data-portal-resultado]");
  const boton = formulario.querySelector('button[type="submit"]');
  const mostrar = (texto) => { if (zona) zona.textContent = texto; };
  const peticion = await cuerpoPortalMiBolsa(formulario, datos);
  if (!peticion) {
    mostrar(textoPortal("error.datos_no_validos"));
    return false;
  }
  if (boton) boton.disabled = true;
  mostrar(textoPortal("enviando"));
  try {
    const respuesta = await fetchImpl(peticion.ruta, {
      method: "POST", credentials: "omit", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", "Content-Type": "application/json" }, body: JSON.stringify(peticion.cuerpo),
    });
    const datos = await respuesta.json().catch(() => null);
    if (respuesta.status === 200 || respuesta.status === 201) {
      mostrar(textoPortal("hecho", { recibo: String(datos?.data?.recibo || "") }));
      alRegistrar();
      return true;
    }
    const codigo = String(datos?.error?.codigo || "servicio_no_disponible");
    mostrar(textoPortal(`error.${Object.hasOwn(RESPALDO, `areaPersonal.portal.error.${codigo}`) ? codigo : "servicio_no_disponible"}`));
    return false;
  } catch {
    mostrar(textoPortal("error.servicio_no_disponible"));
    return false;
  } finally {
    if (boton) boton.disabled = false;
  }
}
