/**
 * «Mis solicitudes» del área personal: estado, justificante interno y fecha de
 * cada solicitud de la persona, con acceso a continuar un borrador. La
 * referencia de la solicitud no se muestra; se ofrece copiarla.
 */
import { crearTraductorSolicitud, textoErrorSolicitud } from "./i18n-solicitud.js?v=20260926-convoca-f1-v2";
import { botonCopiarReferencia } from "./justificante-copiable.js?v=20260926-convoca-f1-v2";
import { escaparHTML as escapar } from "./vistas/comunes.js";

const formatoFecha = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" });
const formatoNumero = new Intl.NumberFormat("es-ES", { maximumFractionDigits: 2 });
const ESTADOS = Object.freeze({ borrador: ["estado_borrador", "aviso"], presentada: ["estado_presentada", "exito"] });

function fila(t, item) {
  const [claveEstado, clase] = ESTADOS[item.estado] || ["estado_otro", "info"];
  const fecha = item.presentada_en ? `<time datetime="${escapar(item.presentada_en)}">${escapar(formatoFecha.format(new Date(item.presentada_en)))}</time>` : escapar(t("sin_dato"));
  const puntos = item.puntuacion_autobaremo === null ? "" : `<small>${escapar(t("autobaremo_total", { puntos: formatoNumero.format(Number(item.puntuacion_autobaremo)) }))}</small>`;
  const accion = item.estado === "borrador"
    ? `<a class="boton-primario" href="?vista=solicitud&amp;id=${encodeURIComponent(item.convocatoria_ref)}" data-ruta="solicitud" data-id="${escapar(item.convocatoria_ref)}">${escapar(t("continuar"))}</a>`
    : botonCopiarReferencia(item.solicitud_ref, { escapar, copiar: t("justificante_copiar"), copiado: t("justificante_copiado"), descripcion: t("referencia_aria", { convocatoria: item.convocatoria_titulo }) });
  const columnas = [
    ["col_convocatoria", `<strong>${escapar(item.convocatoria_titulo || t("sin_dato"))}</strong>`],
    ["col_estado", `<span class="estado-chip ${clase}">${escapar(t(claveEstado))}</span>`],
    ["col_justificante", `${escapar(item.numero_justificante || t("sin_dato"))}${puntos}`],
    ["col_fecha", fecha],
    ["col_accion", `<div class="acciones-tabla">${accion}</div>`],
  ];
  return `<tr>${columnas.map(([clave, contenido]) => `<td data-etiqueta="${escapar(t(clave))}">${contenido}</td>`).join("")}</tr>`;
}

/** Presentación pura de la lista. */
export function renderizarMisSolicitudes(estado, t = crearTraductorSolicitud()) {
  const ayuda = `<button type="button" class="boton-icono boton-ayuda-solicitud" data-ayuda-solicitud="ayuda_mis" aria-haspopup="dialog" aria-label="${escapar(t("ayuda_boton", { tema: t("mis_titulo") }))}">?</button>`;
  const encabezado = `<header class="encabezado-vista"><div><h2>${escapar(t("mis_titulo"))}</h2><p>${escapar(t("mis_descripcion"))}</p></div><div class="fila-acciones"><a class="boton-secundario" href="?vista=solicitud" data-ruta="solicitud">${escapar(t("convocatorias_abiertas"))}</a>${ayuda}</div></header>`;
  if (estado.fase === "cargando") return `${encabezado}<div class="estado-carga" role="status">${escapar(t("cargando_lista"))}</div>`;
  if (estado.fase === "error") return `${encabezado}<div class="nota error" role="alert"><p>${escapar(textoErrorSolicitud(t, estado.error))}</p><button type="button" class="boton-secundario" data-mis-solicitudes-reintentar>${escapar(t("reintentar"))}</button></div>`;
  if (!estado.solicitudes.length) return `${encabezado}<div class="estado-vacio"><strong>${escapar(t("sin_solicitudes"))}</strong></div>`;
  const cabeceras = ["col_convocatoria", "col_estado", "col_justificante", "col_fecha", "col_accion"];
  return `${encabezado}<section class="panel"><div class="panel-contenido"><div class="tabla-contenedor"><table class="tabla-administrativa"><caption>${escapar(t("tabla_titulo"))}</caption><thead><tr>${cabeceras.map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("")}</tr></thead><tbody>${estado.solicitudes.map((item) => fila(t, item)).join("")}</tbody></table></div></div></section>`;
}

export function montarMisSolicitudes({ raiz, cliente, abrirAyuda = () => {}, t = crearTraductorSolicitud() } = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function") throw new TypeError("raíz de mis solicitudes no válida");
  let activo = true;
  const estado = { fase: "cargando", error: "", solicitudes: [] };
  const pintar = () => { if (activo) raiz.innerHTML = renderizarMisSolicitudes(estado, t); };
  async function cargar() {
    estado.fase = "cargando"; pintar();
    try {
      if (typeof cliente?.listar !== "function") throw Object.assign(new Error("no_disponible"), { codigo: "no_disponible" });
      estado.solicitudes = await cliente.listar();
      estado.fase = "listo";
    } catch (error) {
      estado.fase = "error"; estado.error = error?.codigo || "respuesta_incompatible";
    }
    pintar();
  }
  const alPulsar = (evento) => {
    const ayuda = evento.target?.closest?.("[data-ayuda-solicitud]");
    if (ayuda) { abrirAyuda(t("mis_titulo"), t(ayuda.dataset.ayudaSolicitud)); return; }
    if (evento.target?.closest?.("[data-mis-solicitudes-reintentar]")) void cargar();
  };
  raiz.addEventListener("click", alPulsar);
  void cargar();
  return Object.freeze({ desmontar() { activo = false; raiz.removeEventListener("click", alPulsar); } });
}
