import { referenciaConsultaValida, validarResultadoConsulta } from "./consulta-contrato.js";
import { crearTextosConsultaMerito } from "./consulta-i18n.js";

const ESTADOS = new Set(["inicial", "cargando", "resultado", "no_encontrada", "denegada", "error"]);
function escapar(valor) { return String(valor).replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]); }
function fila(etiqueta, valor) { return `<div><dt>${escapar(etiqueta)}</dt><dd>${valor}</dd></div>`; }
function dato(etiqueta, valor) { return fila(etiqueta, escapar(valor)); }
function momento(etiqueta, valor, textos) { return fila(etiqueta, `<time datetime="${escapar(valor)}">${escapar(textos.instante(valor))}</time>`); }
function dia(etiqueta, valor, textos) { return fila(etiqueta, `<time datetime="${escapar(valor)}">${escapar(textos.fecha(valor))}</time>`); }

function tecnico(resultado, textos) {
  const { t, numero } = textos; const r = resultado.recibo_consulta; const h = resultado.hecho_actual;
  return `<details class="consulta-merito-tecnico"><summary>${escapar(t("detalle_tecnico"))}</summary><h4>${escapar(t("recibo_consulta"))}</h4><dl class="consulta-merito-datos">
    ${dato(t("recibo_ref"), r.referencia)}${dato(t("hecho_ref"), r.hecho_ref)}${dato(t("version_actual"), numero(r.version_consultada))}
    ${momento(t("consultada_en"), r.consultada_en, textos)}
    ${dato(t("decision_ref"), r.decision_ref)}${dato(t("consumo_huella"), r.consumo_huella_sha256)}
    ${dato(t("auditoria_ref"), r.auditoria_ref)}${dato(t("correlacion_ref"), r.correlacion_ref)}
    ${h ? dato(t("concepto_ref"), h.concepto_ref) : ""}</dl>
    ${h?.revision ? `<h4>${escapar(t("revision"))}</h4><dl class="consulta-merito-datos">${dato(t("revision_ref"), h.revision.referencia)}${dato(t("motivo_ref"), h.revision.motivo_ref)}${momento(t("revision_fecha"), h.revision.fecha, textos)}</dl>` : ""}</details>`;
}

function ficha(resultado, textos) {
  const { t, numero } = textos; const h = resultado.hecho_actual; const p = h.procedencia;
  return `<article class="panel"><div class="cabecera-panel"><h3>${escapar(h.denominacion)}</h3><span class="consulta-merito-estado consulta-merito-estado--${h.estado}">${escapar(t(h.estado))}</span></div>
    <div class="cuerpo-panel consulta-merito-ficha"><dl class="consulta-merito-datos consulta-merito-resumen">${dato(t("tipo"), t(`tipo_${h.tipo}`))}${dato(t("version_actual"), numero(h.version))}${Object.hasOwn(h, "horas") ? dato(t("horas"), numero(h.horas)) : ""}${momento(t("consultada_en"), resultado.recibo_consulta.consultada_en, textos)}</dl>
    <div class="consulta-merito-columnas"><section><h4>${escapar(t("procedencia"))}</h4><dl class="consulta-merito-datos">${dato(t("fuente_ref"), p.fuente_ref)}${dato(t("fuente_version"), p.version)}${dato(t("origen_ref"), p.hecho_origen_ref)}${momento(t("capturada_en"), p.capturada_en, textos)}</dl></section>
    <section><h4>${escapar(t("vigencia"))}</h4><dl class="consulta-merito-datos">${dia(t("desde"), h.vigencia.desde, textos)}${h.vigencia.hasta ? dia(t("hasta"), h.vigencia.hasta, textos) : dato(t("hasta"), t("sin_fin"))}</dl>
    <h4>${escapar(t("evidencias"))}</h4>${h.evidencias.length ? `<p>${escapar(t("evidencias_cuenta", { cuenta: h.evidencias.length }))}</p><ul class="consulta-merito-evidencias">${h.evidencias.map((e) => `<li><span>${escapar(e.id)}</span><span>${escapar(t("evidencia_version", { version: numero(e.version) }))}</span></li>`).join("")}</ul>` : `<p>${escapar(t("sin_evidencias"))}</p>`}</section></div>
    ${tecnico(resultado, textos)}</div></article>`;
}

/** Render puro. Los estados sin datos nunca reutilizan una ficha anterior. */
export function renderizarConsultaMerito({ estado = "inicial", resultado, hechoRef } = {}, textos = crearTextosConsultaMerito()) {
  if (!ESTADOS.has(estado)) throw new TypeError("meritos.consulta.estado_invalido");
  const { t } = textos;
  let contenido;
  if (["resultado", "no_encontrada"].includes(estado)) {
    resultado = validarResultadoConsulta(resultado, hechoRef);
    if ((estado === "resultado") !== (resultado.codigo === "obtenida")) throw new TypeError("meritos.consulta.estado_invalido");
  }
  if (estado === "resultado") contenido = ficha(resultado, textos);
  else contenido = `<section class="panel consulta-merito-aviso" role="${["error", "denegada"].includes(estado) ? "alert" : "status"}"><div class="cuerpo-panel"><h3>${escapar(t(estado))}</h3><p>${escapar(t(`${estado}_texto`))}</p>${estado === "no_encontrada" ? tecnico(resultado, textos) : ""}</div></section>`;
  const accion = ["resultado", "no_encontrada", "error"].includes(estado) && referenciaConsultaValida(hechoRef)
    ? `<button class="boton-secundario" type="button" data-consulta-merito-actualizar>${escapar(t(estado === "resultado" ? "actualizar" : "reintentar"))}</button>` : "";
  return `<header class="consulta-merito-cabecera"><h2>${escapar(t("titulo"))}</h2><div class="consulta-merito-acciones">${accion}<details class="consulta-merito-ayuda"><summary aria-label="${escapar(t("ayuda"))}">?</summary><p>${escapar(t("ayuda_texto"))}</p></details></div></header><div class="consulta-merito-contenido" aria-busy="${estado === "cargando"}">${contenido}</div>`;
}

/** El montaje recibe la referencia desde el contexto autorizado, nunca desde un input. */
export function montarConsultaMeritoPropio({ raiz, lector, hechoRef, textos = crearTextosConsultaMerito(), anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof lector?.consultar !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")
    || (hechoRef !== undefined && !referenciaConsultaValida(hechoRef))) throw new TypeError("meritos.consulta.montaje_invalido");
  const contenedor = raiz.ownerDocument.createElement("section");
  contenedor.className = "consulta-merito"; contenedor.lang = textos.idioma;
  raiz.append(contenedor);
  let activa = true; let revision = 0; let controlador; let seleccion = hechoRef;
  let modelo = { estado: "inicial" };
  const pintar = () => { if (activa) contenedor.innerHTML = renderizarConsultaMerito(modelo, textos); };
  const anunciarEstado = () => anunciar(textos.t(modelo.estado), ["error", "denegada"].includes(modelo.estado) ? "error" : "informacion");
  const cargar = async (recuperarFoco = false) => {
    if (!activa || !seleccion) return;
    controlador?.abort(); controlador = new AbortController();
    const signal = controlador.signal; const vigente = ++revision; const consultada = seleccion;
    modelo = { estado: "cargando", hechoRef: consultada }; pintar(); anunciarEstado();
    if (recuperarFoco) { contenedor.tabIndex = -1; contenedor.focus?.({ preventScroll: true }); }
    try {
      const datos = await lector.consultar({ hechoRef: consultada, signal });
      if (!activa || signal.aborted || vigente !== revision) return;
      const resultado = validarResultadoConsulta(datos, consultada);
      modelo = { estado: resultado.codigo === "obtenida" ? "resultado" : "no_encontrada", resultado, hechoRef: consultada };
    } catch (causa) {
      if (!activa || signal.aborted || vigente !== revision) return;
      modelo = { estado: causa?.codigo === "denegada" ? "denegada" : "error", hechoRef: consultada };
    }
    pintar(); anunciarEstado();
    if (recuperarFoco) {
      const destino = contenedor.querySelector?.("[data-consulta-merito-actualizar]") ?? contenedor;
      if (destino === contenedor) contenedor.tabIndex = -1;
      destino.focus?.({ preventScroll: true });
    }
  };
  const alClick = (evento) => {
    const boton = evento.target?.closest?.("[data-consulta-merito-actualizar]");
    if (boton && contenedor.contains(boton) && modelo.estado !== "cargando") void cargar(true);
  };
  contenedor.addEventListener("click", alClick); pintar();
  const seleccionar = (referencia) => {
    if (!activa) return Promise.resolve();
    if (referencia !== undefined && !referenciaConsultaValida(referencia)) throw new TypeError("meritos.consulta.seleccion_invalida");
    controlador?.abort(); revision += 1; seleccion = referencia;
    modelo = { estado: "inicial" }; pintar();
    return cargar();
  };
  const desmontar = () => { if (!activa) return; activa = false; revision += 1; controlador?.abort(); modelo = { estado: "inicial" }; seleccion = undefined; contenedor.removeEventListener("click", alClick); contenedor.remove(); };
  registrarDesmontar?.(desmontar);
  const preparada = cargar();
  return Object.freeze({ preparada, seleccionar, actualizar: () => cargar(), desmontar });
}
