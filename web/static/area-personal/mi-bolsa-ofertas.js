// Ofertas publicadas en «Mi bolsa»: la persona ve las ofertas abiertas de sus
// bolsas y manifiesta su disposición (Bolsa 000029). El servidor comprueba
// que la oferta es de una bolsa suya y que sigue abierta; aquí solo se
// recoge la oferta elegida y se muestra el recibo.
import { traducir } from "./i18n.js";
import { escaparAtributo, escaparHTML, listaDatos } from "./vistas/comunes.js";

export const RUTA_DISPOSICION_MI_BOLSA = "/api/vec/bolsa/mi-bolsa/disposiciones";

const ESTADOS = Object.freeze(["abierta", "pendiente_resolucion", "resuelta", "adjudicada_propia"]);
const OFERTA = /^oferta:[0-9a-f]{64}$/u;
const DIA = /^\d{4}-\d{2}-\d{2}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u;

const RESPALDO = Object.freeze({
  "areaPersonal.ofertas.titulo": "Ofertas publicadas",
  "areaPersonal.ofertas.subtitulo": "De mis bolsas",
  "areaPersonal.ofertas.centro": "Centro",
  "areaPersonal.ofertas.inicio": "Inicio",
  "areaPersonal.ofertas.fin": "Fin",
  "areaPersonal.ofertas.sinFin": "Sin fecha de fin",
  "areaPersonal.ofertas.descripcion": "Descripción",
  "areaPersonal.ofertas.vence": "Ofrecerse antes de",
  "areaPersonal.ofertas.manifestar": "Me ofrezco",
  "areaPersonal.ofertas.manifestada": "Se ofreció el {fecha} · Recibo {recibo}",
  "areaPersonal.ofertas.estado.abierta": "Abierta",
  "areaPersonal.ofertas.estado.pendiente_resolucion": "Plazo terminado; pendiente de RRHH",
  "areaPersonal.ofertas.estado.resuelta": "Resuelta",
  "areaPersonal.ofertas.estado.adjudicada_propia": "Adjudicada a usted",
});

export function textoOfertas(clave, variables = {}) {
  const completa = `areaPersonal.ofertas.${clave}`;
  const traducido = traducir(completa, variables);
  if (traducido !== completa) return traducido;
  return (RESPALDO[completa] ?? completa).replace(/\{([a-z_]+)\}/giu, (_, nombre) => String(variables[nombre] ?? ""));
}

function instante(valor, nombre) {
  if (typeof valor !== "string" || !INSTANTE.test(valor) || Number.isNaN(Date.parse(valor))) throw new TypeError(`${nombre} no es un instante válido.`);
}

function texto(valor, nombre) {
  if (typeof valor !== "string" || valor.length < 2 || valor.length > 2000 || valor.trim() !== valor) throw new TypeError(`${nombre} no es válido.`);
}

// validarOfertasMiBolsa comprueba la parte opcional «ofertas» de la
// respuesta. Sin ella no se ofrece la sección.
export function validarOfertasMiBolsa(datos) {
  if (datos.ofertas === undefined) return;
  if (!Array.isArray(datos.ofertas) || datos.ofertas.length > 100) throw new TypeError("Las ofertas de mi bolsa no son válidas.");
  const bolsas = new Set((datos.participaciones || []).map((p) => p.bolsa));
  for (const o of datos.ofertas) {
    if (!o || typeof o !== "object" || !OFERTA.test(o.oferta || "") || !bolsas.has(o.bolsa) || !ESTADOS.includes(o.estado)) {
      throw new TypeError("Oferta no válida o de una bolsa ajena.");
    }
    for (const campo of ["categoria", "centro", "descripcion"]) texto(o[campo], campo);
    if (!DIA.test(o.fecha_inicio || "") || (o.fecha_fin !== null && !DIA.test(o.fecha_fin || ""))) throw new TypeError("Fechas de la oferta no válidas.");
    instante(o.publicada_en, "publicada_en");
    instante(o.vence_antes_de, "vence_antes_de");
    if (o.disposicion !== null) {
      if (!o.disposicion || typeof o.disposicion.recibo !== "string") throw new TypeError("Disposición no válida.");
      instante(o.disposicion.manifestada_en, "manifestada_en");
    }
  }
}

function fecha(valor) {
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

function dia(valor) {
  const [a, m, d] = valor.split("-").map(Number);
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" }).format(new Date(Date.UTC(a, m - 1, d)));
}

function claseEstado(estado) {
  if (estado === "abierta") return "info";
  if (estado === "adjudicada_propia") return "exito";
  return "aviso";
}

// renderizarOfertasMiBolsa pinta cada oferta con su estado y, si sigue
// abierta y la persona no se ha ofrecido, el botón para hacerlo.
export function renderizarOfertasMiBolsa(ofertas) {
  if (!Array.isArray(ofertas) || ofertas.length === 0) return "";
  const t = textoOfertas;
  const tarjetas = ofertas.map((o) => {
    const datos = listaDatos([
      [t("centro"), escaparHTML(o.centro)],
      [t("inicio"), escaparHTML(dia(o.fecha_inicio))],
      [t("fin"), escaparHTML(o.fecha_fin ? dia(o.fecha_fin) : t("sinFin"))],
      [t("descripcion"), escaparHTML(o.descripcion)],
      [t("vence"), escaparHTML(fecha(o.vence_antes_de))],
    ]);
    let accion = "";
    if (o.disposicion) {
      accion = `<p class="nota">${escaparHTML(t("manifestada", { fecha: fecha(o.disposicion.manifestada_en), recibo: o.disposicion.recibo }))}</p>`;
    } else if (o.estado === "abierta") {
      accion = `<form class="portal-mi-bolsa__formulario" data-portal-mi-bolsa="disposicion" data-oferta="${escaparAtributo(o.oferta)}">
        <button type="submit" class="boton-primario">${escaparHTML(t("manifestar"))}</button>
        <p class="nota" role="status" aria-live="polite" data-portal-resultado></p></form>`;
    }
    return `<article class="portal-mi-bolsa__bolsa" data-oferta-mi-bolsa="${escaparAtributo(o.oferta)}">
      <h4>${escaparHTML(o.categoria)} <span class="estado-chip ${claseEstado(o.estado)}">${escaparHTML(t(`estado.${o.estado}`))}</span></h4>
      ${datos}${accion}</article>`;
  }).join("");
  return `<div class="portal-mi-bolsa">${tarjetas}</div>`;
}

// cuerpoDisposicion prepara la petición del formulario de una oferta. La
// clave se guarda en el formulario: un reintento repite la misma petición.
export function cuerpoDisposicion(formulario) {
  const oferta = formulario.dataset.oferta || "";
  if (!OFERTA.test(oferta)) return null;
  formulario.dataset.clave ||= `portal-${globalThis.crypto.randomUUID()}`;
  return { ruta: RUTA_DISPOSICION_MI_BOLSA, cuerpo: { oferta, clave: formulario.dataset.clave } };
}
