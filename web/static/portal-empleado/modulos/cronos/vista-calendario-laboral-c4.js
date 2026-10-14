import { MENSAJES_CALENDARIO_LABORAL_C4, crearTraductorCalendarioLaboralC4 } from "./i18n-calendario-laboral-c4.js";
import { fechaCalendarioC4Valida, validarCalendarioLaboralC4 } from "./cliente-calendario-laboral-c4.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function fechaVisible(fecha, locale) {
  const [anio, mes, dia] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { dateStyle: "long", timeZone: "UTC" })
    .format(new Date(Date.UTC(anio, mes - 1, dia, 12)));
}

export function renderizarCalendarioLaboralC4({ fecha, estado = "cargando", datos = null,
  mensajes = MENSAJES_CALENDARIO_LABORAL_C4, locale = LOCALIZACION_ACTUAL } = {}) {
  if (!fechaCalendarioC4Valida(fecha)) throw new TypeError("fecha de calendario no válida");
  const t = crearTraductorCalendarioLaboralC4(mensajes);
  if (estado === "listo") validarCalendarioLaboralC4(datos, fecha);
  let contenido;
  if (estado === "cargando") contenido = `<p role="status">${html(t("cargando"))}</p>`;
  else if (estado === "denegado") contenido = `<p role="alert">${html(t("denegado"))}</p>`;
  else if (estado === "error") contenido = `<p role="alert">${html(t("error"))}</p><button type="button" class="boton-secundario" data-cronos-cal-laboral-reintentar>${html(t("reintentar"))}</button>`;
  else if (datos.estado === "no_determinable") contenido = `<p role="status">${html(t("sin_fuente"))}</p>`;
  else {
    const versiones = datos.versiones_calendario.map((v) => `${html(v.denominacion)} · ${html(v.numero)}`).join("; ");
    const jornada = datos.estado === "programado"
      ? `<p><strong>${html(t("programado"))}</strong> · ${html(String(Math.floor(datos.minutos_programados / 60)).padStart(2, "0"))}:${html(String(datos.minutos_programados % 60).padStart(2, "0"))}</p>
        <p>${html(t("turno"))}: <time datetime="${html(datos.inicio_utc)}">${html(new Intl.DateTimeFormat(locale, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(datos.inicio_utc)))}</time>
        – <time datetime="${html(datos.fin_utc)}">${html(new Intl.DateTimeFormat(locale, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(datos.fin_utc)))}</time></p>
        <details><summary>${html(t("fuente_programacion"))}</summary><dl><dt>${html(t("programacion"))}</dt><dd>${html(datos.programacion_ref)}</dd>
        <dt>${html(t("version_programacion"))}</dt><dd>${html(datos.version_programacion)}</dd>
        <dt>${html(t("regimen"))}</dt><dd>${html(datos.regimen_ref)}</dd>
        <dt>${html(t("relacion"))}</dt><dd>${html(datos.relacion_ref)}</dd>
        <dt>${html(t("fuente"))}</dt><dd>${html(datos.fuente_programacion_ref)} · ${html(datos.fuente_version_programacion)}</dd></dl></details>`
      : `<p>${html(t("sin_jornada"))}</p>`;
    contenido = `<p class="cronos-estado cronos-estado-exito">${html(t("publicado"))}</p>
      <p>${html(datos.dia.laborable ? t("laborable") : t("no_laborable"))}</p>
      ${jornada}
      <dl><dt>${html(t("centro"))}</dt><dd>${html(t("centro_identificado"))}</dd></dl>
      <details><summary>${html(t("fuentes"))}</summary><dl>
        <dt>${html(t("centro"))}</dt><dd>${html(datos.centro_ref)}</dd>
        <dt>${html(t("unidad"))}</dt><dd>${html(datos.fuente_unidad_ref)} · ${html(datos.fuente_unidad_version)}</dd>
        <dt>${html(t("organizacion"))}</dt><dd>${html(datos.fuente_organizacion_ref)} · ${html(datos.version_organizacion)}</dd>
        <dt>${html(t("calendarios"))}</dt><dd>${versiones}</dd>
      </dl></details>`;
  }
  return `<section class="panel cronos-panel" data-cronos-cal-laboral-estado="${html(estado)}" aria-busy="${estado === "cargando"}">
    <div class="cabecera-panel"><h3>${html(t("titulo"))}</h3></div>
    <div class="cuerpo-panel"><p><time datetime="${html(fecha)}">${html(fechaVisible(fecha, locale))}</time></p>${contenido}</div>
  </section>`;
}

// El calendario civil puede llamar seleccionar(fecha). Cada cambio cancela
// la petición anterior y descarta respuestas que lleguen fuera de orden.
export function montarCalendarioLaboralC4({ raiz, cliente, fecha, mensajes = MENSAJES_CALENDARIO_LABORAL_C4,
  locale = LOCALIZACION_ACTUAL, registrarDesmontar } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultar !== "function"
    || !fechaCalendarioC4Valida(fecha) || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de calendario laboral no disponible");
  }
  const nodo = raiz.ownerDocument.createElement("div");
  raiz.append(nodo);
  let actual = fecha; let controlador; let secuencia = 0; let activo = true;
  const pintar = (estado, datos) => { nodo.innerHTML = renderizarCalendarioLaboralC4({ fecha: actual, estado, datos, mensajes, locale }); };
  const seleccionar = async (nueva) => {
    if (!activo || !fechaCalendarioC4Valida(nueva)) return;
    controlador?.abort(); controlador = new AbortController();
    actual = nueva; const turno = ++secuencia; pintar("cargando");
    try {
      const datos = await cliente.consultar(nueva, { signal: controlador.signal });
      if (activo && turno === secuencia) pintar("listo", datos);
    } catch (error) {
      if (activo && turno === secuencia && !controlador.signal.aborted) {
        pintar(error?.estado === 401 || error?.estado === 403 ? "denegado" : "error");
      }
    }
  };
  const clic = (evento) => { if (evento.target?.closest?.("[data-cronos-cal-laboral-reintentar]")) void seleccionar(actual); };
  nodo.addEventListener("click", clic);
  void seleccionar(fecha);
  const desmontar = () => { if (!activo) return; activo = false; ++secuencia; controlador?.abort(); nodo.removeEventListener("click", clic); nodo.remove(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ seleccionar, desmontar });
}
