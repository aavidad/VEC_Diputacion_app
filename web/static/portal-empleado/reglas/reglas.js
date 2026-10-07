import { IDIOMA_DATOS_REGLAS, IDIOMA_REGLAS, MENSAJES_REGLAS, crearTraductorReglas, existeClaveReglas, formatearNumero, minusculas, textoPresentacionRegla } from "./i18n.js?v=20261007-pantallas-textos-final-v1";
import { icono } from "../../comun/iconos-vec.js?v=20260925-aspecto-v1";

export const API_REGLAS = "/api/vec/reglas/vigentes";
export const ESQUEMA = "vec.reglas.vigentes.v1";
export const LIMITE_RESPUESTA = 1024 * 1024;

const t = MENSAJES_REGLAS ? crearTraductorReglas() : null;
const ESTADOS = new Set(["disponible", "sin_catalogo", "no_disponible"]);
const ORIGENES = new Set(["reglamento", "ejemplo"]);
const CODIGOS_API = new Set(["solicitud_invalida", "servicio_no_disponible", "autenticacion_requerida", "acceso_denegado"]);
const CLAVE = /^[a-z][a-z0-9._:-]{0,95}$/u;

const esc = (v) => String(v ?? "")
  .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;").replaceAll("'", "&#39;");

export class ErrorReglas extends Error {
  constructor(codigo) {
    super(codigo);
    this.codigo = codigo;
  }
}

function exigir(condicion) {
  if (!condicion) throw new ErrorReglas("error_respuesta");
}
const texto = (v, max = 4000) => typeof v === "string" && v.length > 0 && v.length <= max;
const opcional = (v, max = 4000) => v === undefined || (typeof v === "string" && v.length <= max);

/** Valida el contrato vec.reglas.vigentes.v1; lo desconocido se rechaza entero. */
export function validarReglas(payload) {
  const d = payload?.data;
  exigir(d && d.esquema === ESQUEMA && Array.isArray(d.catalogos) && d.catalogos.length <= 8);
  for (const c of d.catalogos) {
    exigir(CLAVE.test(c?.modulo) && CLAVE.test(c?.catalogo_id) && ESTADOS.has(c?.estado)
      && typeof c?.paquete_ejemplo === "boolean" && Array.isArray(c?.reglas) && c.reglas.length <= 1024);
    exigir(c.estado === "disponible" || c.reglas.length === 0);
    for (const r of c.reglas) {
      exigir(CLAVE.test(r?.clave) && texto(r?.etiqueta, 400) && opcional(r?.descripcion) && CLAVE.test(r?.unidad)
        && ORIGENES.has(r?.origen) && texto(r?.norma) && texto(r?.duda) && Number.isInteger(r?.version) && r.version >= 1
        && texto(r?.referencia, 400) && typeof r?.paquete_ejemplo === "boolean"
        && (r.cantidad === undefined || (Number.isInteger(r.cantidad) && r.cantidad > 0))
        && opcional(r.valor) && opcional(r.articulo, 200) && opcional(r.ejemplo_parcial) && opcional(r.computo, 40) && opcional(r.inicio, 200));
      exigir(r.origen === "reglamento" ? texto(r.articulo, 200) : r.articulo === undefined);
    }
  }
  return d;
}

/** Cliente de solo lectura: sin cookies propias ni cabeceras de identidad. */
export function crearCliente(fetchImpl = globalThis.fetch, timeoutMs = 10000) {
  return Object.freeze({
    reglas: async () => {
      const controlador = new AbortController();
      const temporizador = setTimeout(() => controlador.abort(), timeoutMs);
      try {
        const respuesta = await fetchImpl(API_REGLAS, {
          method: "GET", credentials: "same-origin", mode: "same-origin", redirect: "error", cache: "no-store", referrerPolicy: "no-referrer",
          headers: { Accept: "application/json" }, signal: controlador.signal,
        });
        const cuerpo = await respuesta.text();
        if (cuerpo.length > LIMITE_RESPUESTA) throw new ErrorReglas("error_respuesta");
        let json = null;
        try { json = JSON.parse(cuerpo); } catch { /* se decide abajo */ }
        if (respuesta.ok) {
          if (json === null) throw new ErrorReglas("error_respuesta");
          return validarReglas(json);
        }
        let codigo = respuesta.status === 401 ? "autenticacion_requerida" : respuesta.status === 403 ? "acceso_denegado" : "servicio_no_disponible";
        if (CODIGOS_API.has(json?.error?.codigo)) codigo = json.error.codigo;
        throw new ErrorReglas(`error_${codigo}`);
      } catch (error) {
        if (error instanceof ErrorReglas) throw error;
        throw new ErrorReglas("error_servicio_no_disponible");
      } finally {
        clearTimeout(temporizador);
      }
    },
  });
}

const etiquetaModulo = (m) => (existeClaveReglas(`modulo_${m}`) ? t(`modulo_${m}`) : m);
const etiquetaUnidad = (u) => (existeClaveReglas(`unidad_${u}`) ? t(`unidad_${u}`) : u);

/** Valor legible: cantidad, franja o lista; «No aplica» si la unidad no lleva valor. */
export function valorRegla(r) {
  if (r.cantidad !== undefined) return formatearNumero(r.cantidad);
  if (r.unidad === "franja_horaria" && r.valor) return r.valor.replace("-", "–");
  if (r.valor) return r.valor.split(",").map((v) => v.trim()).join(", ");
  return t("sinValor");
}

export function origenRegla(r) {
  return r.origen === "reglamento" ? t("origenArticulo", { articulo: r.articulo }) : t("origenEjemplo");
}

export function filtrar(catalogos, { modulo = "", origen = "", texto: consulta = "" } = {}) {
  const q = minusculas(consulta.trim());
  return catalogos.filter((c) => !modulo || c.modulo === modulo).map((c) => ({
    ...c,
    reglas: c.reglas.filter((r) => (!origen || r.origen === origen)
      && (!q || [r.etiqueta, r.clave, r.norma, ...["descripcion", "duda", "ejemplo_parcial"]
        .map((campo) => textoPresentacionRegla(c.modulo, r, campo).texto)].some((v) => minusculas(v).includes(q)))),
  }));
}

const ICONOS_KPI = Object.freeze({ total: "reglas", reglamento: "documento", ejemplo: "pendiente" });

export function renderizarResumen(catalogos) {
  const reglas = catalogos.flatMap((c) => c.reglas);
  const kpi = (clase, clave, valor) => `<div class="rg-kpi rg-kpi--${clase}"><span class="rg-kpi-icono">${icono(ICONOS_KPI[clase])}</span><span class="rg-kpi-rotulo">${esc(t(clave))}</span><strong>${esc(formatearNumero(valor))}</strong></div>`;
  return kpi("total", "kpiTotal", reglas.length)
    + kpi("reglamento", "kpiReglamento", reglas.filter((r) => r.origen === "reglamento").length)
    + kpi("ejemplo", "kpiEjemplo", reglas.filter((r) => r.origen === "ejemplo").length);
}

/** Identificador de fila estable y seguro para `id` y `aria-controls`. */
const idSeguro = (valor) => Array.from(new TextEncoder().encode(valor),
  (byte) => byte.toString(16).padStart(2, "0")).join("");
export const idRegla = (modulo, clave) => `rg-regla-${idSeguro(`${modulo}\0${clave}`)}`;

/** Los textos de cada regla llegan del catálogo en su idioma: se marcan si la interfaz usa otro. */
const LANG_DATOS = IDIOMA_REGLAS === IDIOMA_DATOS_REGLAS ? "" : ` lang="${esc(IDIOMA_DATOS_REGLAS)}"`;
const datos = (texto, idioma = IDIOMA_DATOS_REGLAS) => `<span${idioma === IDIOMA_REGLAS ? "" : ` lang="${esc(idioma)}"`}>${esc(texto)}</span>`;
const textoRegla = (modulo, regla, campo) => {
  const { texto, idioma } = textoPresentacionRegla(modulo, regla, campo);
  return datos(texto, idioma);
};
const parteEjemplo = (r, modulo) => {
  const [antes, despues] = MENSAJES_REGLAS.parteEjemplo.split("{texto}");
  return `${esc(antes)}${textoRegla(modulo, r, "ejemplo_parcial")}${esc(despues)}`;
};

function filaRegla(r, modulo, abiertas) {
  const parcial = r.ejemplo_parcial ? `<br><small>${parteEjemplo(r, modulo)}</small>` : "";
  const computo = r.computo && existeClaveReglas(`computo_${r.computo}`) ? `<br><small>${esc(t(`computo_${r.computo}`))}</small>` : "";
  const pastilla = r.origen === "reglamento" ? "rg-pastilla--reglamento" : "rg-pastilla--ejemplo";
  const id = idRegla(modulo, r.clave);
  const abierta = abiertas.has(id);
  return `<tr class="rg-fila rg-fila--${esc(r.origen)}${abierta ? " rg-fila--abierta" : ""}" data-regla="${id}">
    <th scope="row"><button type="button" class="rg-regla-abrir" aria-expanded="${abierta}" aria-controls="${id}"><span class="rg-regla-nombre"${LANG_DATOS}>${esc(r.etiqueta)}</span></button></th>
    <td class="rg-numero">${r.valor ? datos(valorRegla(r)) : esc(valorRegla(r))}</td>
    <td>${esc(etiquetaUnidad(r.unidad))}${computo}</td>
    <td><span class="rg-pastilla ${pastilla}">${esc(origenRegla(r))}</span>${parcial}</td>
    <td>${textoRegla(modulo, r, "duda")}</td>
    <td class="rg-numero">${esc(formatearNumero(r.version))}</td>
  </tr>
  <tr class="rg-detalle rg-fila--${esc(r.origen)}" id="${id}"${abierta ? "" : " hidden"}><td colspan="6">${detalleRegla(r, modulo)}</td></tr>`;
}

/** Texto completo de la regla, sin códigos internos: lo que la fila no deja leer entero. */
export function detalleRegla(r, modulo = "") {
  const bloque = (clave, contenido) => `<div class="rg-detalle-bloque"><dt>${esc(t(clave))}</dt><dd>${contenido}</dd></div>`;
  const partes = [
    bloque("detalleQue", r.descripcion ? textoRegla(modulo, r, "descripcion") : esc(t("detalleSinDescripcion"))),
    bloque("detalleOrigen", esc(origenRegla(r)) + (r.ejemplo_parcial ? `<br>${parteEjemplo(r, modulo)}` : "")),
    bloque("detalleNorma", datos(r.norma)),
    bloque("detalleDuda", textoRegla(modulo, r, "duda")),
  ];
  return `<div class="rg-detalle-cuerpo"><dl>${partes.join("")}</dl></div>`;
}

export function renderizarCatalogo(c, abiertas = new Set()) {
  const id = `rg-cat-${idSeguro(c.modulo)}`;
  const pastilla = c.paquete_ejemplo ? `<span class="rg-pastilla rg-pastilla--ejemplo">${esc(t("paqueteEjemplo"))}</span>` : "";
  const meta = c.estado === "disponible" && c.version
    ? `<p class="rg-meta">${esc(t("catalogoVersion", { version: formatearNumero(c.version) }))}</p>`
    : "";
  let cuerpo;
  if (c.estado !== "disponible") {
    cuerpo = `<p class="rg-aviso ${c.estado === "no_disponible" ? "rg-aviso--error" : ""}">${esc(t(`estado_${c.estado}`))}</p>`;
  } else if (!c.reglas.length) {
    cuerpo = `<p class="rg-aviso">${esc(t("sinResultados"))}</p>`;
  } else {
    cuerpo = `<div class="rg-tabla" tabindex="0" role="region" aria-labelledby="${id}"><table>
      <thead><tr><th scope="col">${esc(t("colRegla"))}</th><th scope="col" class="rg-numero">${esc(t("colValor"))}</th><th scope="col">${esc(t("colUnidad"))}</th><th scope="col">${esc(t("colOrigen"))}</th><th scope="col">${esc(t("colDuda"))}</th><th scope="col" class="rg-numero">${esc(t("colVersion"))}</th></tr></thead>
      <tbody>${c.reglas.map((r) => filaRegla(r, c.modulo, abiertas)).join("")}</tbody></table></div>`;
  }
  return `<section class="rg-panel" aria-labelledby="${id}">
    <div class="rg-panel-cabecera"><div><h2 id="${id}">${esc(etiquetaModulo(c.modulo))}</h2>${meta}</div>
      <div class="rg-panel-acciones">${pastilla}<span class="rg-contador">${esc(t("contadorReglas", { cantidad: formatearNumero(c.reglas.length) }))}</span></div></div>
    ${cuerpo}
  </section>`;
}

export function mensajeError(error) {
  const clave = error instanceof ErrorReglas && existeClaveReglas(error.codigo) ? error.codigo : "error_servicio_no_disponible";
  return t(clave);
}

function traducirDocumento(doc) {
  doc.documentElement.lang = IDIOMA_REGLAS;
  doc.title = t("documentTitle");
  doc.querySelectorAll("[data-i18n]").forEach((el) => { el.textContent = t(el.dataset.i18n); });
  doc.querySelectorAll("[data-i18n-label]").forEach((el) => { el.setAttribute("aria-label", t(el.dataset.i18nLabel)); });
}

export async function iniciar(doc, cliente) {
  if (!t) {
    doc.documentElement.lang = IDIOMA_REGLAS;
    const aviso = doc.getElementById("rg-estado");
    aviso.hidden = false;
    aviso.classList.add("rg-aviso--error");
    doc.getElementById("rg-ayuda-abrir").hidden = true;
    try {
      const { cargarTextos } = await import("../../comun/textos.js");
      const textos = (await cargarTextos("portal")).seccion("textos");
      doc.title = textos.txt_reglas_vigentes;
      doc.querySelector(".rg-cabecera h1").textContent = textos.txt_reglas_vigentes;
      doc.querySelector(".rg-volver").textContent = textos.txt_volver_al_cuadro_de_mando;
      aviso.textContent = textos.txt_el_servicio_no_esta_disponible_ahora_puede_reint;
      const reintentar = doc.createElement("button");
      reintentar.type = "button";
      reintentar.className = "rg-secundario";
      reintentar.textContent = textos.txt_reintentar;
      reintentar.addEventListener("click", () => doc.defaultView.location.reload());
      aviso.append(" ", reintentar);
    } catch { /* Sin catálogo común tampoco se muestra contenido de reglas. */ }
    return;
  }
  traducirDocumento(doc);
  const $ = (id) => doc.getElementById(id);
  const ayuda = $("rg-ayuda");
  $("rg-ayuda-abrir").addEventListener("click", () => ayuda.showModal?.());
  $("rg-ayuda-cerrar").addEventListener("click", () => ayuda.close?.());
  const avisar = (mensaje, error = false) => {
    const el = $("rg-estado");
    el.textContent = mensaje;
    el.classList.toggle("rg-aviso--error", error);
    el.hidden = mensaje === "";
  };
  const cargar = async (disparador) => {
    const aviso = $("rg-estado");
    const conservarFoco = disparador && doc.activeElement === disparador;
    avisar(t("cargando"));
    if (conservarFoco) aviso.focus({ preventScroll: true });
    let datos;
    try {
      datos = await cliente.reglas();
    } catch (error) {
      avisar(mensajeError(error), true);
      const reintentar = doc.createElement?.("button");
      if (reintentar) {
        reintentar.type = "button";
        reintentar.className = "rg-secundario";
        reintentar.textContent = t("reintentar");
        reintentar.addEventListener("click", () => cargar(reintentar));
        aviso.append(" ", reintentar);
        if (conservarFoco && doc.activeElement === aviso) reintentar.focus({ preventScroll: true });
      }
      return;
    }
    const modulos = [...new Set(datos.catalogos.map((c) => c.modulo))];
    $("rg-modulo").innerHTML = `<option value="">${esc(t("todos"))}</option>` + modulos.map((m) => `<option value="${esc(m)}">${esc(etiquetaModulo(m))}</option>`).join("");
    $("rg-origen").innerHTML = `<option value="">${esc(t("todos"))}</option>` + [...ORIGENES].map((o) => `<option value="${o}">${esc(t(`origen_${o}`))}</option>`).join("");
    // Reglas abiertas: sobreviven al filtrar y, la última, a recargar (ancla).
    const abiertas = new Set();
    let ancla = "";
    try { ancla = decodeURIComponent(String(doc.defaultView?.location?.hash ?? "").slice(1)); }
    catch { /* Un ancla inválida no debe impedir mostrar las reglas. */ }
    if (/^rg-regla-[a-z0-9-]+$/u.test(ancla)) abiertas.add(ancla);
    const pintar = () => {
      const visibles = filtrar(datos.catalogos, { modulo: $("rg-modulo").value, origen: $("rg-origen").value, texto: $("rg-texto").value });
      $("rg-kpis").innerHTML = renderizarResumen(visibles);
      $("rg-catalogos").innerHTML = visibles.map((c) => renderizarCatalogo(c, abiertas)).join("");
    };
    const alternar = (fila) => {
      const id = fila.dataset.regla;
      const abrir = !abiertas.has(id);
      if (abrir) abiertas.add(id); else abiertas.delete(id);
      fila.classList.toggle("rg-fila--abierta", abrir);
      fila.querySelector(".rg-regla-abrir")?.setAttribute("aria-expanded", String(abrir));
      const detalle = doc.getElementById(id);
      if (detalle) detalle.hidden = !abrir;
      const vista = doc.defaultView;
      // El ancla guarda la última regla abierta que sigue abierta, para recargar sin perderla.
      const ultima = abrir ? id : [...abiertas].at(-1);
      try { vista?.history?.replaceState(null, "", ultima ? `#${ultima}` : `${vista.location.pathname}${vista.location.search}`); } catch { /* sin historial */ }
    };
    // Se abre con el botón del nombre o pulsando en cualquier punto de la fila.
    $("rg-catalogos").addEventListener("click", (evento) => {
      const fila = evento.target.closest?.("tr.rg-fila");
      if (!fila || (evento.target.closest("a") && !evento.target.closest(".rg-regla-abrir"))) return;
      if (!evento.target.closest(".rg-regla-abrir") && doc.getSelection?.()?.toString()) return;
      alternar(fila);
    });
    $("rg-filtros").addEventListener("submit", (evento) => evento.preventDefault());
    for (const id of ["rg-modulo", "rg-origen"]) $(id).addEventListener("change", pintar);
    $("rg-texto").addEventListener("input", pintar);
    pintar();
    const devolverFoco = conservarFoco && doc.activeElement === aviso;
    avisar("");
    $("rg-resultado").hidden = false;
    if (devolverFoco) $("rg-modulo").focus({ preventScroll: true });
    if (abiertas.size) doc.getElementById([...abiertas][0])?.previousElementSibling?.scrollIntoView?.({ block: "center" });
  };
  await cargar();
}

if (typeof document !== "undefined" && document.getElementById("reglas")) {
  iniciar(document, crearCliente());
}
