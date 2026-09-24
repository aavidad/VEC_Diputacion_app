import { crearTraductorCronosC4, MENSAJES_CRONOS_C4_ES } from "./i18n-c4.js?v=20260924-web-paradas-periodos-v1";

const ANIO_MINIMO = 1900;
const ANIO_MAXIMO = 2100;

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function fechaISO(anio, mes, dia) {
  return `${anio}-${String(mes).padStart(2, "0")}-${String(dia).padStart(2, "0")}`;
}

function fechaValida(fecha) {
  if (!/^\d{4}-\d{2}-\d{2}$/u.test(String(fecha))) return false;
  const [anio, mes, dia] = fecha.split("-").map(Number);
  if (anio < ANIO_MINIMO || anio > ANIO_MAXIMO || mes < 1 || mes > 12) return false;
  return dia >= 1 && dia <= new Date(Date.UTC(anio, mes, 0)).getUTCDate();
}

function fechaActualMadrid() {
  const partes = new Intl.DateTimeFormat("en-US", {
    timeZone: "Europe/Madrid", year: "numeric", month: "2-digit", day: "2-digit",
  }).formatToParts(new Date());
  const valor = (tipo) => partes.find((parte) => parte.type === tipo)?.value;
  return `${valor("year")}-${valor("month")}-${valor("day")}`;
}

function formatoFecha(fecha, locale) {
  const [anio, mes, dia] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: "full" })
    .format(new Date(Date.UTC(anio, mes - 1, dia)));
}

function nombreMes(anio, mes, locale) {
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", month: "long" })
    .format(new Date(Date.UTC(anio, mes - 1, 1)));
}

function diasSemana(locale) {
  const corto = new Intl.DateTimeFormat(locale, { timeZone: "UTC", weekday: "short" });
  const largo = new Intl.DateTimeFormat(locale, { timeZone: "UTC", weekday: "long" });
  return Array.from({ length: 7 }, (_, indice) => {
    const fecha = new Date(Date.UTC(2024, 0, 1 + indice));
    return `<th scope="col"><abbr title="${escaparHTML(largo.format(fecha))}">${escaparHTML(corto.format(fecha))}</abbr></th>`;
  }).join("");
}

function mesHTML(anio, mes, seleccionada, locale, t) {
  const dias = new Date(Date.UTC(anio, mes, 0)).getUTCDate();
  const huecos = (new Date(Date.UTC(anio, mes - 1, 1)).getUTCDay() + 6) % 7;
  const celdas = Array.from({ length: huecos }, () => '<td aria-hidden="true"></td>');
  for (let dia = 1; dia <= dias; dia++) {
    const fecha = fechaISO(anio, mes, dia);
    const finSemana = [0, 6].includes(new Date(Date.UTC(anio, mes - 1, dia)).getUTCDay());
    const etiqueta = t("calendario_dia", { fecha: formatoFecha(fecha, locale) });
    celdas.push(`<td><button type="button" data-cronos-cal-dia="${fecha}" aria-label="${escaparHTML(etiqueta)}" aria-pressed="${fecha === seleccionada}" tabindex="${fecha === seleccionada ? "0" : "-1"}"${finSemana ? ' data-dia-civil="fin-semana"' : ""}>${dia}</button></td>`);
  }
  while (celdas.length % 7) celdas.push('<td aria-hidden="true"></td>');
  const filas = [];
  for (let indice = 0; indice < celdas.length; indice += 7) filas.push(`<tr>${celdas.slice(indice, indice + 7).join("")}</tr>`);
  const mesVisible = nombreMes(anio, mes, locale);
  return `<section class="cronos-calendario-mes" aria-label="${escaparHTML(t("calendario_mes", { mes: mesVisible, anio }))}">
    <h4>${escaparHTML(mesVisible)}</h4><table><thead><tr>${diasSemana(locale)}</tr></thead><tbody>${filas.join("")}</tbody></table>
  </section>`;
}

/** Solo calendario gregoriano civil. Nunca atribuye efectos laborales o administrativos. */
export function renderizarCalendarioCivilCronos({ anio, fechaSeleccionada, locale = "es-ES", mensajes = MENSAJES_CRONOS_C4_ES } = {}) {
  const seleccionada = fechaSeleccionada ?? fechaActualMadrid();
  if (!fechaValida(seleccionada)) throw new RangeError("fecha civil de Cronos no válida");
  const anioVisible = anio ?? Number(seleccionada.slice(0, 4));
  if (!Number.isInteger(anioVisible) || anioVisible < ANIO_MINIMO || anioVisible > ANIO_MAXIMO || Number(seleccionada.slice(0, 4)) !== anioVisible) {
    throw new RangeError("año civil de Cronos no válido");
  }
  const t = crearTraductorCronosC4(mensajes);
  const diaSemana = new Date(`${seleccionada}T00:00:00Z`).getUTCDay();
  return `<section class="panel cronos-calendario" data-cronos-calendario data-estado="no_configurado" aria-labelledby="cronos-calendario-titulo">
    <header class="cabecera-panel"><div><h3 id="cronos-calendario-titulo">${escaparHTML(t("calendario_civil_titulo"))}</h3><p>${escaparHTML(t("calendario_civil_descripcion"))}</p></div><span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("calendario_estado"))}</span></header>
    <div class="cuerpo-panel cronos-calendario-cuerpo">
      <nav class="cronos-calendario-navegacion" aria-label="${escaparHTML(t("calendario_civil_titulo"))}"><button type="button" class="boton-secundario" data-cronos-cal-anio="-1" aria-label="${escaparHTML(t("calendario_anterior"))}"${anioVisible === ANIO_MINIMO ? " disabled" : ""}>‹</button><strong aria-live="polite">${anioVisible}</strong><button type="button" class="boton-secundario" data-cronos-cal-anio="1" aria-label="${escaparHTML(t("calendario_siguiente"))}"${anioVisible === ANIO_MAXIMO ? " disabled" : ""}>›</button></nav>
      <div class="cronos-calendario-meses">${Array.from({ length: 12 }, (_, indice) => mesHTML(anioVisible, indice + 1, seleccionada, locale, t)).join("")}</div>
      <div class="cronos-calendario-pie"><div class="cronos-calendario-seleccion" role="status"><span>${escaparHTML(t("calendario_seleccion"))}</span><strong>${escaparHTML(formatoFecha(seleccionada, locale))}</strong><span>${escaparHTML(t("calendario_natural"))}${[0, 6].includes(diaSemana) ? ` · ${escaparHTML(t("calendario_fin_semana"))}` : ""}</span></div>
      <div class="cronos-calendario-leyenda" aria-label="${escaparHTML(t("calendario_leyenda"))}"><strong>${escaparHTML(t("calendario_leyenda"))}</strong><span>${escaparHTML(t("calendario_leyenda_dia"))}</span><span>${escaparHTML(t("calendario_leyenda_fin_semana"))}</span></div></div>
      <p class="cronos-calendario-aviso" role="note"><strong>${escaparHTML(t("calendario_estado"))}.</strong> ${escaparHTML(t("calendario_estado_detalle"))} ${escaparHTML(t("calendario_fuente"))}</p>
    </div>
  </section>`;
}

/** Montaje independiente: el integrador registra CSS y lo inserta en Jornada. */
export function montarCalendarioCivilCronos({ raiz, fechaSeleccionada, locale = "es-ES", mensajes = MENSAJES_CRONOS_C4_ES, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof anunciar !== "function"
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("raíz de calendario civil de Cronos no válida");
  const contenedor = raiz.ownerDocument.createElement("div");
  let seleccionada = fechaSeleccionada ?? fechaActualMadrid();
  let anio = Number(seleccionada.slice(0, 4));
  const pintar = () => { contenedor.innerHTML = renderizarCalendarioCivilCronos({ anio, fechaSeleccionada: seleccionada, locale, mensajes }); };
  pintar();
  raiz.append(contenedor);
  const t = crearTraductorCronosC4(mensajes);
  const seleccionar = (fecha, enfocar = false) => {
    if (!fechaValida(fecha)) return;
    seleccionada = fecha;
    anio = Number(fecha.slice(0, 4));
    pintar();
    if (enfocar) contenedor.querySelector?.(`[data-cronos-cal-dia="${fecha}"]`)?.focus?.();
    anunciar(t("calendario_dia", { fecha: formatoFecha(fecha, locale) }));
  };
  const clic = (evento) => {
    const dia = evento.target?.closest?.("[data-cronos-cal-dia]");
    if (dia) { seleccionar(dia.dataset.cronosCalDia, true); return; }
    const control = evento.target?.closest?.("[data-cronos-cal-anio]");
    if (!control || control.disabled) return;
    const siguiente = anio + Number(control.dataset.cronosCalAnio);
    if (siguiente < ANIO_MINIMO || siguiente > ANIO_MAXIMO) return;
    const [, mes, diaNumero] = seleccionada.split("-").map(Number);
    seleccionada = fechaISO(siguiente, mes, Math.min(diaNumero, new Date(Date.UTC(siguiente, mes, 0)).getUTCDate()));
    anio = siguiente;
    pintar();
    contenedor.querySelector?.(`[data-cronos-cal-anio="${control.dataset.cronosCalAnio}"]`)?.focus?.();
    anunciar(String(anio));
  };
  const teclado = (evento) => {
    const dia = evento.target?.closest?.("[data-cronos-cal-dia]");
    if (!dia) return;
    const desplazamiento = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -7, ArrowDown: 7 }[evento.key];
    if (desplazamiento === undefined && !["Home", "End"].includes(evento.key)) return;
    evento.preventDefault();
    const [anioDia, mes, numero] = dia.dataset.cronosCalDia.split("-").map(Number);
    const nueva = evento.key === "Home" ? fechaISO(anioDia, mes, 1)
      : evento.key === "End" ? fechaISO(anioDia, mes, new Date(Date.UTC(anioDia, mes, 0)).getUTCDate())
        : (() => { const fecha = new Date(Date.UTC(anioDia, mes - 1, numero + desplazamiento)); return fechaISO(fecha.getUTCFullYear(), fecha.getUTCMonth() + 1, fecha.getUTCDate()); })();
    seleccionar(nueva, true);
  };
  contenedor.addEventListener("click", clic);
  contenedor.addEventListener("keydown", teclado);
  let activo = true;
  const desmontar = () => {
    if (!activo) return;
    activo = false;
    contenedor.removeEventListener("click", clic);
    contenedor.removeEventListener("keydown", teclado);
    contenedor.remove();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
