import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import { cargarTextos } from "../../../comun/textos.js";

const TEXTOS = (await cargarTextos("personal-servicios-cer")).seccion("general");
const ESTADOS = new Set(["declarado", "comprobado", "reconocido"]);
const SELECCION = new Set(["incluido", "fuera_corte", "pendiente", "sustituido"]);
const FALTANTES = new Set(["acto", "fuente", "version_fuente", "version_hecho", "clase", "periodo", "registro"]);
const fecha = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(valor) && Number.isFinite(Date.parse(`${valor}T12:00:00Z`)) && new Date(`${valor}T12:00:00Z`).toISOString().slice(0, 10) === valor;
const seguro = (valor, limite = 256) => typeof valor === "string" && valor.length > 0 && valor.length <= limite && !/[\p{Cc}\p{Cf}]/u.test(valor);
const nodo = (d, etiqueta, texto) => { const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; return n; };

export function crearTraductorPreparacionServiciosCER(catalogo = TEXTOS) {
  if (!catalogo || Object.keys(TEXTOS).some((k) => typeof catalogo[k] !== "string" || !catalogo[k])) throw new TypeError("personal.servicios_certificados.catalogo_invalido");
  return (clave) => { if (!Object.hasOwn(TEXTOS, clave)) throw new TypeError("personal.servicios_certificados.clave_invalida"); return catalogo[clave]; };
}

function validarModelo(m) {
  if (!m || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(m.empleado_ref) || m.estado !== "preparacion_sintetica" || m.cobertura !== "no_acreditada" || m.eficacia_administrativa !== false || m.firma_oficial !== false ||
      !Number.isSafeInteger(m.version) || m.version < 1 || !fecha(m.corte?.vigente_en) || !seguro(m.corte?.conocido_en, 40) || !Number.isFinite(Date.parse(m.corte.conocido_en)) ||
      !Array.isArray(m.servicios) || m.servicios.length > 200 || !m.servicios.every((s) => s && seguro(s.servicio_ref, 160) && seguro(s.relacion_ref, 160) &&
        ESTADOS.has(s.estado) && SELECCION.has(s.seleccion_temporal) && typeof s.solapado === "boolean" && typeof s.clase === "string" && s.clase.length <= 256 &&
        Array.isArray(s.faltantes) && s.faltantes.length <= FALTANTES.size && s.faltantes.every((f) => FALTANTES.has(f)) && s.traza && typeof s.traza === "object" &&
        (fecha(s.periodo_desde) || s.faltantes.includes("periodo")) && (fecha(s.periodo_hasta) || s.faltantes.includes("periodo")))) throw new TypeError("personal.servicios_certificados.modelo_invalido");
}

// Solo presenta el modelo ya calculado tras la consulta B2 autorizada.
// El filtro de relación no calcula periodos, reconocimiento ni cobertura.
export function crearPanelPreparacionServiciosCER({ documento: d, modelo, relacionRef = "", t = crearTraductorPreparacionServiciosCER() }) {
  validarModelo(modelo);
  const seccion = nodo(d, "section"); seccion.className = "panel";
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-panel"; cabecera.append(nodo(d, "h3", t("titulo")));
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel";
  const limite = nodo(d, "p", t("limite")); limite.setAttribute("role", "status"); cuerpo.append(limite);
  const formatearFecha = (valor) => fecha(valor) ? new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(`${valor}T12:00:00Z`)) : t("sin_valor");
  const corte = nodo(d, "dl");
  for (const [clave, valor] of [["corte_efectivo", formatearFecha(modelo.corte.vigente_en)], ["corte_conocimiento", new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(modelo.corte.conocido_en))]]) corte.append(nodo(d, "dt", t(clave)), nodo(d, "dd", valor));
  cuerpo.append(corte);
  if (!relacionRef) cuerpo.append(nodo(d, "p", t("seleccionar_relacion")));
  else {
    const filas = modelo.servicios.filter((s) => s.relacion_ref === relacionRef);
    if (filas.length === 0) cuerpo.append(nodo(d, "p", t("vacio")));
    else {
      const detalle = nodo(d, "details"); detalle.append(nodo(d, "summary", t("revisar")));
      const region = nodo(d, "div"); region.className = "tabla-contenedor"; region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("tabla"));
      const tabla = nodo(d, "table"); tabla.className = "tabla-datos"; tabla.append(nodo(d, "caption", t("tabla")));
      const cabeza = nodo(d, "thead"); const cab = nodo(d, "tr");
      for (const clave of ["periodo", "clase", "estado", "revision", "origen"]) { const th = nodo(d, "th", t(clave)); th.setAttribute("scope", "col"); cab.append(th); }
      cabeza.append(cab); tabla.append(cabeza); const tbody = nodo(d, "tbody");
      for (const s of filas) {
        const tr = nodo(d, "tr"); tr.append(nodo(d, "td", `${formatearFecha(s.periodo_desde)} – ${formatearFecha(s.periodo_hasta)}`), nodo(d, "td", seguro(s.clase) ? s.clase : t("sin_valor")), nodo(d, "td", t(s.estado)));
        const revision = nodo(d, "td"); revision.append(nodo(d, "p", t(s.seleccion_temporal)));
        if (s.solapado) revision.append(nodo(d, "p", t("solapado")));
        if (s.faltantes.length) { const lista = nodo(d, "ul"); for (const f of s.faltantes) lista.append(nodo(d, "li", t(`falta_${f}`))); revision.append(lista); }
        const origen = nodo(d, "td"); const procedencia = nodo(d, "details"); procedencia.append(nodo(d, "summary", t("ver_origen")));
        const datos = nodo(d, "dl");
        for (const [clave, valor] of [["version_hecho", s.traza.version], ["version_fuente", s.traza.fuente_version]]) datos.append(nodo(d, "dt", t(clave)), nodo(d, "dd", Number.isSafeInteger(valor) && valor > 0 ? new Intl.NumberFormat(LOCALIZACION_ACTUAL).format(valor) : t("sin_valor")));
        procedencia.append(datos);
        const tecnico = nodo(d, "details"); tecnico.append(nodo(d, "summary", t("referencias"))); const referencias = nodo(d, "dl");
        for (const [clave, valor] of [["acto", s.traza.acto_ref], ["fuente", s.traza.fuente_ref]]) referencias.append(nodo(d, "dt", t(clave)), nodo(d, "dd", seguro(valor, 160) ? valor : t("sin_valor")));
        tecnico.append(referencias); procedencia.append(tecnico); origen.append(procedencia); tr.append(revision, origen); tbody.append(tr);
      }
      tabla.append(tbody); region.append(tabla); detalle.append(region); cuerpo.append(detalle);
    }
  }
  seccion.append(cabecera, cuerpo); return seccion;
}
