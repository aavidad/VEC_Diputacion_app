// Contacto propio en «Mi bolsa» (duda 45): la persona ve si el contacto que
// tiene RRHH procede de CONVOCA y lo confirma. Se confirma la versión que se
// muestra; si RRHH registró otra entre medias, el servidor lo rechaza. El
// correo y los teléfonos nunca llegan a esta pantalla.
import { traducir } from "./i18n.js";
import { escaparAtributo, escaparHTML } from "./vistas/comunes.js";

export const RUTA_CONTACTO_MI_BOLSA = "/api/vec/bolsa/mi-bolsa/contacto";

const DIA = /^\d{4}-\d{2}-\d{2}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u;

const RESPALDO = Object.freeze({
  "areaPersonal.contacto.titulo": "Mi contacto",
  "areaPersonal.contacto.subtitulo": "Por bolsa",
  "areaPersonal.contacto.convoca": "Contacto de origen CONVOCA sin confirmar",
  "areaPersonal.contacto.vencido": "Contacto de origen CONVOCA; confirmación pendiente desde el {fecha}",
  "areaPersonal.contacto.confirmado": "Contacto confirmado el {fecha}",
  "areaPersonal.contacto.rrhh": "Contacto registrado por RRHH",
  "areaPersonal.contacto.confirmar": "Confirmo que mi contacto es correcto",
});

export function textoContacto(clave, variables = {}) {
  const completa = `areaPersonal.contacto.${clave}`;
  const traducido = traducir(completa, variables);
  if (traducido !== completa) return traducido;
  return (RESPALDO[completa] ?? completa).replace(/\{([a-z_]+)\}/giu, (_, nombre) => String(variables[nombre] ?? ""));
}

// validarContactosMiBolsa comprueba la parte opcional «contactos».
export function validarContactosMiBolsa(datos) {
  if (datos.contactos === undefined) return;
  if (!Array.isArray(datos.contactos)) throw new TypeError("Los contactos de mi bolsa no son válidos.");
  const bolsas = new Set((datos.participaciones || []).map((p) => p.bolsa));
  for (const c of datos.contactos) {
    if (!c || !bolsas.has(c.bolsa) || !Number.isSafeInteger(c.version) || c.version < 1) throw new TypeError("Contacto no válido o de una bolsa ajena.");
    if (c.origen !== null && (!c.origen || !["vigente", "vencido"].includes(c.origen.estado) || !DIA.test(c.origen.ultimo_dia || ""))) {
      throw new TypeError("Origen del contacto no válido.");
    }
    if (c.confirmada_en !== null && (typeof c.confirmada_en !== "string" || !INSTANTE.test(c.confirmada_en))) throw new TypeError("Confirmación no válida.");
    if (Object.keys(c).sort().join() !== "bolsa,confirmada_en,origen,version") throw new TypeError("El contacto contiene campos no autorizados.");
  }
}

function fecha(valor) {
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

function dia(valor) {
  const [a, m, d] = valor.split("-").map(Number);
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" }).format(new Date(Date.UTC(a, m - 1, d)));
}

// renderizarContactoMiBolsa pinta el estado del contacto de cada bolsa y, si
// procede de CONVOCA sin confirmar, el botón para confirmarlo.
export function renderizarContactoMiBolsa(participaciones, contactos) {
  if (!Array.isArray(contactos) || contactos.length === 0) return "";
  const categorias = new Map((participaciones || []).map((p) => [p.bolsa, p.categoria]));
  const t = textoContacto;
  const bloques = contactos.map((c) => {
    let estado;
    let accion = "";
    if (c.confirmada_en) {
      estado = `<span class="estado-chip exito">${escaparHTML(t("confirmado", { fecha: fecha(c.confirmada_en) }))}</span>`;
    } else if (c.origen) {
      const texto = c.origen.estado === "vencido" ? t("vencido", { fecha: dia(c.origen.ultimo_dia) }) : t("convoca");
      estado = `<span class="estado-chip aviso">${escaparHTML(texto)}</span>`;
      accion = `<form class="portal-mi-bolsa__formulario" data-portal-mi-bolsa="contacto" data-bolsa="${escaparAtributo(c.bolsa)}" data-version="${escaparAtributo(String(c.version))}">
        <button type="submit" class="boton-secundario">${escaparHTML(t("confirmar"))}</button>
        <p class="nota" role="status" aria-live="polite" data-portal-resultado></p></form>`;
    } else {
      estado = `<span class="estado-chip info">${escaparHTML(t("rrhh"))}</span>`;
    }
    return `<article class="portal-mi-bolsa__bolsa"><h4>${escaparHTML(categorias.get(c.bolsa) || c.bolsa)}</h4><p>${estado}</p>${accion}</article>`;
  }).join("");
  return `<div class="portal-mi-bolsa">${bloques}</div>`;
}

// cuerpoConfirmacionContacto prepara la petición; la clave se guarda en el
// formulario para que un reintento repita la misma petición.
export function cuerpoConfirmacionContacto(formulario) {
  const bolsa = formulario.dataset.bolsa || "";
  const version = Number(formulario.dataset.version);
  if (!bolsa || !Number.isSafeInteger(version) || version < 1) return null;
  formulario.dataset.clave ||= `portal-${globalThis.crypto.randomUUID()}`;
  return { ruta: RUTA_CONTACTO_MI_BOLSA, cuerpo: { bolsa, version, clave: formulario.dataset.clave } };
}
