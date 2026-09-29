// Resumen operativo de Cronos. No conoce el estado global del portal ni sus
// transportes: recibe una vista ya autorizada y una raiz DOM intercambiable.
import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "../../comun/idioma.js";

const MENSAJES_RESUMEN_CRONOS = Object.freeze({
  es: Object.freeze({
    secciones: "Secciones de Cronos", sin_secciones: "Sin secciones disponibles",
    horas_teoricas: "Horas teóricas", horas_trabajadas: "Horas trabajadas",
    teletrabajo: "Teletrabajo", balance: "Exceso / defecto mes",
    saldos: "Saldos de permisos disponibles", permiso: "Permiso", disponible: "Disponible",
    maximo: "Máximo", solicitado: "Solicitado", restante: "Restante",
    si: "Sí", no: "No", sin_saldos: "Sin saldos de permisos disponibles.",
  }),
  en: Object.freeze({
    secciones: "Cronos sections", sin_secciones: "No sections available",
    horas_teoricas: "Expected hours", horas_trabajadas: "Hours worked",
    teletrabajo: "Remote work", balance: "Monthly surplus / deficit",
    saldos: "Available leave balances", permiso: "Leave type", disponible: "Available",
    maximo: "Maximum", solicitado: "Requested", restante: "Remaining",
    si: "Yes", no: "No", sin_saldos: "No leave balances available.",
  }),
});

export function traducirResumenCronos(clave, idioma = IDIOMA_ACTUAL) {
  const texto = MENSAJES_RESUMEN_CRONOS[idioma]?.[clave] ?? MENSAJES_RESUMEN_CRONOS.es[clave];
  if (texto === undefined) throw new Error(`Missing Cronos summary key: ${clave}`);
  return texto;
}

export function renderizarResumenCronos(vista, documento = document, idioma = IDIOMA_ACTUAL) {
  const destino = documento.querySelector("#cronos-panel");
  if (!destino) return;

  const t = (clave) => traducirResumenCronos(clave, idioma);
  const localizacion = idioma === IDIOMA_ACTUAL ? LOCALIZACION_ACTUAL : (idioma === "en" ? "en-GB" : "es-ES");
  const formatoNumero = new Intl.NumberFormat(localizacion);
  const espacio = vista?.workspace || {};
  const resumen = espacio.cronos_daily_summary || {};
  const permisos = Array.isArray(espacio.cronos_permission_balances) ? espacio.cronos_permission_balances : [];
  const secciones = Array.isArray(espacio.cronos_sections) ? espacio.cronos_sections : [];
  const visible = (valor) => {
    if (valor === null || valor === undefined || valor === "") return "-";
    if (typeof valor === "number" && Number.isFinite(valor)) return formatoNumero.format(valor);
    if (typeof valor === "string" && /^-?\d+(?:\.\d+)?$/.test(valor)) return formatoNumero.format(Number(valor));
    return String(valor);
  };
  const siNo = (valor) => valor === true || /^(si|sí|true|1)$/i.test(String(valor || "")) ? t("si")
    : valor === false || /^(no|false|0)$/i.test(String(valor || "")) ? t("no") : visible(valor);

  const navegacion = documento.createElement("ul");
  navegacion.className = "cronos-nav";
  navegacion.setAttribute("aria-label", t("secciones"));
  const seccionesVisibles = secciones.length ? secciones : [t("sin_secciones")];
  navegacion.replaceChildren(...seccionesVisibles.map((seccion) => {
    const etiqueta = documento.createElement("li");
    etiqueta.textContent = seccion;
    return etiqueta;
  }));

  const cajaResumen = documento.createElement("div");
  cajaResumen.className = "cronos-summary";
  const indicadores = [
    [t("horas_teoricas"), visible(resumen.theoretical)],
    [t("horas_trabajadas"), visible(resumen.worked)],
    [t("teletrabajo"), siNo(resumen.telework)],
    [t("balance"), visible(resumen.period_balance ?? resumen.daily_balance)],
  ];
  cajaResumen.replaceChildren(...indicadores.map(([nombre, valor]) => {
    const indicador = documento.createElement("div");
    const etiqueta = documento.createElement("span");
    const dato = documento.createElement("strong");
    etiqueta.textContent = nombre;
    dato.textContent = valor;
    if (String(valor).startsWith("-")) dato.classList.add("negative");
    indicador.replaceChildren(etiqueta, dato);
    return indicador;
  }));

  const tabla = documento.createElement("table");
  tabla.className = "mini-table";
  tabla.setAttribute("aria-label", t("saldos"));
  const cabecera = documento.createElement("thead");
  const filaCabecera = documento.createElement("tr");
  filaCabecera.replaceChildren(...[t("permiso"), t("disponible"), t("maximo"), t("solicitado"), t("restante")].map((texto) => {
    const celda = documento.createElement("th");
    celda.setAttribute("scope", "col");
    celda.textContent = texto;
    return celda;
  }));
  cabecera.replaceChildren(filaCabecera);
  const cuerpo = documento.createElement("tbody");
  const filas = permisos.slice(0, 8).map((permiso) => {
    const fila = documento.createElement("tr");
    const valores = [
      permiso.name || "-",
      permiso.request ? t("si") : t("no"),
      visible(permiso.max),
      visible(permiso.requested),
      visible(permiso.remaining),
    ];
    fila.replaceChildren(...valores.map((valor, indice) => {
      const celda = documento.createElement(indice === 0 ? "th" : "td");
      if (indice === 0) celda.setAttribute("scope", "row");
      celda.textContent = valor;
      return celda;
    }));
    return fila;
  });
  if (!filas.length) {
    const filaVacia = documento.createElement("tr");
    const celdaVacia = documento.createElement("td");
    celdaVacia.className = "cronos-empty";
    celdaVacia.setAttribute("colspan", "5");
    celdaVacia.textContent = t("sin_saldos");
    filaVacia.replaceChildren(celdaVacia);
    filas.push(filaVacia);
  }
  cuerpo.replaceChildren(...filas);
  tabla.replaceChildren(cabecera, cuerpo);

  const envoltorioTabla = documento.createElement("div");
  envoltorioTabla.className = "cronos-table-wrap";
  envoltorioTabla.replaceChildren(tabla);

  destino.replaceChildren(navegacion, cajaResumen, envoltorioTabla);
}
