import { cargarTextos } from "../comun/textos.js";
import { icono } from "../comun/iconos-vec.js?v=20260925-aspecto-v1";

const TEXTOS = await cargarTextos("accesos-empleado");

export function crearTraductorAccesosEmpleado(textos = TEXTOS) {
  return (clave, variables) => textos.traducir(`accesos.${clave}`, variables);
}

export const traducirAccesosEmpleado = crearTraductorAccesosEmpleado();

// La composición entrega navegación propia, no concesiones de lectura. Cada
// destino conserva la autorización del servidor y la propiedad de sus datos.
const DESTINOS = Object.freeze([
  { vista: "mis-tramites", texto: "mis_tramites", icono: "expediente" },
  { vista: "personal", icono: "expediente" },
  { vista: "cronos", icono: "reloj" },
  { vista: "dietas", icono: "euro" },
]);
const ESTADOS_NAVEGABLES = new Set(["diferido", "disponible", "cargando"]);

/** Conserva el resumen de gestión y sólo aclara dónde está la navegación propia. */
export function crearTraductorResumenAccesosEmpleado({ accesos, traducir, textos = TEXTOS }) {
  const propios = DESTINOS.some(({ vista }) => accesos && Object.hasOwn(accesos, vista)
    && ESTADOS_NAVEGABLES.has(accesos[vista]?.estado));
  return (clave, variables) => clave === "resumen_modulos_ninguno" && propios
    ? textos.traducir("accesos.consultar_accesos") : traducir(clave, variables);
}

/**
 * accesos: { personal?, cronos?, dietas? }, cada uno con un estado de navegación
 * de composición. Ausentes, denegados, no disponibles y desconocidos se omiten.
 * El renderer solo produce HTML; no consulta datos ni monta módulos de destino.
 */
export function renderizarAccesosEmpleado({ accesos, escaparHTML, traducir = traducirAccesosEmpleado }) {
  const t = (clave) => escaparHTML(traducir(clave));
  const filas = DESTINOS.flatMap(({ vista, texto = vista, icono: nombreIcono }) => {
    if (!accesos || !Object.hasOwn(accesos, vista)) return [];
    const estado = accesos[vista]?.estado;
    const navegable = ESTADOS_NAVEGABLES.has(estado);
    if (!navegable && estado !== "error") return [];
    const etiqueta = t(texto);
    const accion = navegable
      ? `<a class="boton-terciario" href="#${vista}" data-vista="${vista}"${estado === "cargando" ? ' aria-busy="true"' : ""}>${icono(nombreIcono)}${etiqueta}</a>`
      : `<strong>${etiqueta}</strong>`;
    const mensaje = estado === "cargando"
      ? `<span role="status">${t("cargando")}</span>`
      : (estado === "error" ? `<span role="alert">${t("error")}</span>` : "");
    return [`<li class="elemento-actividad"><span class="marca-actividad" aria-hidden="true"></span><div class="acciones-fila">${accion} ${mensaje}</div></li>`];
  });
  if (filas.length === 0) return "";
  return `<section class="panel" aria-labelledby="portal-accesos-empleado-titulo">
    <div class="cabecera-panel"><h3 id="portal-accesos-empleado-titulo">${t("titulo")}</h3></div>
    <div class="cuerpo-panel"><ul class="lista-actividad" role="list">${filas.join("")}</ul></div>
  </section>`;
}
