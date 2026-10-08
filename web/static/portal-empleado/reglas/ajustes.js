import { cargarTextosAjustes, reintentarTextosAjustes, idiomaAjustes, idiomaDatosAjustes,
  existeClaveAjustes, numeroAjustes, fechaAjustes, traducirAjustes, usaIdiomaRespaldoAjustes } from "./ajustes-i18n.js";
import { cargarTextos } from "../../comun/textos.js";

export const API_AJUSTES = "/api/vec/contratacion-temporal/reglas/ajustes";
const ESQUEMA = "vec.contratacion_temporal.reglas.ajustes.v1";
const CLAVE = /^[a-z][a-z0-9._-]{2,95}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const CAMPOS = new Set(["cantidad", "cantidad_urgente", "unidad", "computo"]);
const LIMITE_RESPUESTA = 1024 * 1024;
const t = traducirAjustes;
const existeClaveReglas = existeClaveAjustes;
const formatearNumero = numeroAjustes;
const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");

export class ErrorAjustes extends Error {
  constructor(codigo) { super(codigo); this.codigo = codigo; }
}
const exigir = (condicion) => { if (!condicion) throw new ErrorAjustes("ajustesRespuestaInvalida"); };
const cadena = (valor, maximo = 400) => typeof valor === "string" && valor.length <= maximo;
const version = (valor) => Number.isSafeInteger(valor) && valor >= 0;
const validarEdicion = (edicion) => {
  exigir(edicion && Array.isArray(edicion.campos) && edicion.campos.length > 0
    && edicion.campos.every((campo) => CAMPOS.has(campo))
    && Array.isArray(edicion.opciones_unidad) && edicion.opciones_unidad.length <= 16
    && Array.isArray(edicion.opciones_computo) && edicion.opciones_computo.length <= 16
    && Number.isSafeInteger(edicion.cantidad_minima) && Number.isSafeInteger(edicion.cantidad_maxima)
    && edicion.cantidad_minima > 0 && edicion.cantidad_maxima >= edicion.cantidad_minima);
  for (const opcion of edicion.opciones_unidad) exigir(CLAVE.test(opcion) && existeClaveReglas(`unidad_${opcion}`));
  for (const opcion of edicion.opciones_computo) exigir(CLAVE.test(opcion) && existeClaveReglas(`computo_${opcion}`));
};

/** La edición solo se habilita con metadatos recibidos completos y comprobables. */
export function validarLecturaAjustes(respuesta) {
  const d = respuesta?.data;
  exigir(d?.esquema === ESQUEMA && d.catalogo_id === "vec.contratacion_temporal.reglas.ajustes"
    && version(d.version_esperada) && typeof d.puede_ajustar === "boolean" && Array.isArray(d.reglas)
    && d.reglas.length <= 64 && Array.isArray(d.historial) && d.historial.length <= 50
    && typeof d.hay_mas === "boolean" && d.activacion
    && ["activa", "inactiva", "sin_publicar"].includes(d.activacion.estado)
    && Object.keys(d.activacion).length === 1);
  exigir(d.activacion.estado === "activa" || (!d.puede_ajustar && d.reglas.length === 0));
  exigir(d.motivos === undefined || d.motivos === null || (Array.isArray(d.motivos) && d.motivos.length <= 16
    && d.motivos.every((m) => CLAVE.test(m?.clave) && cadena(m.texto_clave, 100) && existeClaveReglas(m.texto_clave))));
  for (const r of d.reglas) {
    exigir(CLAVE.test(r?.clave) && cadena(r.etiqueta) && r.etiqueta.length > 0
      && (r.ajuste_no_aplicable === undefined || typeof r.ajuste_no_aplicable === "boolean"));
    if (r.ajuste_no_aplicable) {
      exigir(["valores", "cantidad", "cantidad_urgente", "unidad", "computo"].every((campo) => !Object.hasOwn(r, campo)));
      if (r.edicion != null) validarEdicion(r.edicion);
      continue;
    }
    validarEdicion(r.edicion);
    exigir(r.valores && typeof r.valores === "object" && !Array.isArray(r.valores)
      && existeClaveReglas(`unidad_${r.valores.unidad ?? r.unidad}`));
    for (const campo of r.edicion.campos) exigir(cadena(r.valores[campo], 80)
      && (!["unidad", "computo"].includes(campo) || existeClaveReglas(`${campo}_${r.valores[campo]}`)));
  }
  for (const h of d.historial) {
    exigir(version(h?.version) && h.version > 0 && cadena(h.vigente_desde, 64)
      && CLAVE.test(h.motivo_clave) && (h.referencia === undefined || cadena(h.referencia, 120))
      && (h.nota === undefined || cadena(h.nota, 500))
      && Array.isArray(h.cambios) && h.cambios.length <= 64);
    for (const c of h.cambios) exigir(CLAVE.test(c?.regla_clave) && CAMPOS.has(c.campo)
      && cadena(c.anterior, 80) && cadena(c.nuevo, 80));
  }
  return { ...d, motivos: d.motivos ?? [] };
}

export function validarReciboAjustes(respuesta) {
  const r = respuesta?.data?.recibo;
  exigir(respuesta?.data?.esquema === ESQUEMA && r && cadena(r.recibo_ref) && r.recibo_ref.length > 0 && version(r.version) && r.version > 0
    && cadena(r.vigente_desde, 64) && r.vigente_desde.length > 0
    && HUELLA.test(r.huella_sha256) && HUELLA.test(r.consumo_huella_sha256)
    && cadena(r.clave_idempotencia, 80) && r.clave_idempotencia.length > 0
    && cadena(r.decision_ref) && r.decision_ref.length > 0
    && cadena(r.auditoria_ref) && r.auditoria_ref.length > 0
    && typeof respuesta.data.replay === "boolean");
  return respuesta.data;
}

async function pedir(fetchImpl, url, opciones, controlador, timeoutMs) {
  const tiempo = setTimeout(() => controlador.abort(), timeoutMs);
  try {
    const respuesta = await fetchImpl(url, { credentials: "same-origin", mode: "same-origin", redirect: "error",
      cache: "no-store", referrerPolicy: "no-referrer", ...opciones, signal: controlador.signal });
    const cuerpo = await respuesta.text();
    if (cuerpo.length > LIMITE_RESPUESTA) throw new ErrorAjustes("ajustesRespuestaInvalida");
    let json;
    try { json = JSON.parse(cuerpo); } catch { throw new ErrorAjustes("ajustesRespuestaInvalida"); }
    if (respuesta.status === 409) throw new ErrorAjustes("ajustesConflicto");
    if (respuesta.status === 422) throw new ErrorAjustes("ajustesValorInvalido");
    if (respuesta.status === 401 || respuesta.status === 403) throw new ErrorAjustes("ajustesSinPermiso");
    if (!respuesta.ok) throw new ErrorAjustes("ajustesNoDisponible");
    return json;
  } catch (error) {
    if (error instanceof ErrorAjustes) throw error;
    throw new ErrorAjustes("ajustesNoDisponible");
  } finally { clearTimeout(tiempo); }
}

export function crearClienteAjustes(fetchImpl = globalThis.fetch, timeoutMs = 10000) {
  return Object.freeze({
    leer: async ({ antesDeVersion, signal } = {}) => {
      const controlador = new AbortController();
      if (signal?.aborted) controlador.abort();
      signal?.addEventListener("abort", () => controlador.abort(), { once: true });
      const parametros = new URLSearchParams({ limite: "20" });
      if (antesDeVersion !== undefined) parametros.set("antes_de_version", String(antesDeVersion));
      const json = await pedir(fetchImpl, `${API_AJUSTES}?${parametros}`, { method: "GET", headers: { Accept: "application/json" } }, controlador, timeoutMs);
      return validarLecturaAjustes(json);
    },
    guardar: async (comando, { signal } = {}) => {
      const controlador = new AbortController();
      if (signal?.aborted) controlador.abort();
      signal?.addEventListener("abort", () => controlador.abort(), { once: true });
      const json = await pedir(fetchImpl, API_AJUSTES, { method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify(comando) }, controlador, timeoutMs);
      const resultado = validarReciboAjustes(json);
      exigir(resultado.recibo.clave_idempotencia === comando.clave_idempotencia
        && resultado.recibo.version === comando.version_esperada + 1);
      return resultado;
    },
  });
}

const CLAVES_CAMPO = Object.freeze({
  cantidad: "ajustesCampoCantidad",
  cantidad_urgente: "ajustesCampoCantidadUrgente",
  unidad: "ajustesCampoUnidad",
  computo: "ajustesCampoComputo",
});
const etiquetaCampo = (campo) => t(CLAVES_CAMPO[campo]);
const etiquetaOpcion = (tipo, valor) => t(`${tipo}_${valor}`);
const presentarValor = (regla, campo, valor, unidad = regla.valores.unidad ?? regla.unidad) => ["cantidad", "cantidad_urgente"].includes(campo)
  ? `${formatearNumero(Number(valor))} ${etiquetaOpcion("unidad", unidad)}`
  : etiquetaOpcion(campo, valor);
const valorHistorico = (campo, valor) => {
  if (campo === "unidad" || campo === "computo") {
    const clave = `${campo}_${valor}`;
    return existeClaveReglas(clave) ? t(clave) : t("ajustesValorHistoricoNoDisponible");
  }
  return /^[0-9]+$/u.test(valor) && Number.isSafeInteger(Number(valor))
    ? formatearNumero(Number(valor)) : t("ajustesValorHistoricoNoDisponible");
};
const fecha = (iso) => Number.isNaN(Date.parse(iso)) ? "" : fechaAjustes(iso);

function renderizarHistoriaSinBase(historial, motivos) {
  if (!historial.length) return "";
  const entradas = historial.map((h) => {
    const motivo = motivos.find((m) => m.clave === h.motivo_clave);
    const motivoTexto = motivo ? t(motivo.texto_clave) : t("ajustesMotivoNoIdentificado");
    const cambios = h.cambios.map((c) => {
      const claveNombre = `ajustesRegla${c.regla_clave.replace(/(^|[._])([a-z0-9])/gu, (_valor, _separador, letra) => letra.toUpperCase())}`;
      const nombre = existeClaveReglas(claveNombre) ? t(claveNombre) : t("ajustesReglaAnterior");
      return `<li>${esc(nombre)} · ${esc(etiquetaCampo(c.campo))}: ${esc(valorHistorico(c.campo, c.anterior))} → ${esc(valorHistorico(c.campo, c.nuevo))}</li>`;
    }).join("");
    return `<li><span>${esc(fecha(h.vigente_desde))}</span> · ${esc(h.actor_nombre || t("ajustesPersonaGenerica"))} · ${esc(t("ajustesVersion", { version: formatearNumero(h.version) }))} · ${esc(motivoTexto)}
      <ul>${cambios}</ul>
      ${h.referencia ? `<p>${esc(t("ajustesReferenciaHistoria", { referencia: h.referencia }))}</p>` : ""}
      ${h.nota ? `<p>${esc(t("ajustesNotaHistoria", { nota: h.nota }))}</p>` : ""}
      ${h.recibo_ref ? `<details><summary>${esc(t("ajustesVerJustificante"))}</summary><code>${esc(h.recibo_ref)}</code></details>` : ""}</li>`;
  }).join("");
  return `<details class="rg-ajuste-historial"><summary>${esc(t("ajustesHistorial"))}</summary><ol>${entradas}</ol></details>`;
}

export function renderizarAjustes(modelo, { reglaActiva = "", borrador = null, fase = "lista", aviso = "", error = false, recibo = null, bloqueado = false } = {}) {
  const motivos = modelo.motivos ?? [];
  const baseActiva = modelo.activacion.estado === "activa";
  const editable = modelo.puede_ajustar && motivos.length > 0 && !bloqueado;
  const reglas = modelo.reglas.map((r) => {
    const activa = r.clave === reglaActiva;
    const claveNombre = `ajustesRegla${r.clave.replace(/(^|[._])([a-z0-9])/gu, (_valor, _separador, letra) => letra.toUpperCase())}`;
    const nombreCatalogado = existeClaveReglas(claveNombre);
    const nombre = nombreCatalogado ? t(claveNombre) : r.etiqueta;
    const valores = r.ajuste_no_aplicable ? "" : r.edicion.campos.map((campo) => `<div><dt>${esc(etiquetaCampo(campo))}</dt><dd>${esc(presentarValor(r, campo, r.valores[campo]))}</dd></div>`).join("");
    const historial = modelo.historial.filter((h) => h.cambios.some((c) => c.regla_clave === r.clave));
    const listaHistorial = historial.length ? historial.map((h) => `<li><span>${esc(fecha(h.vigente_desde))}</span> · ${esc(h.actor_nombre || t("ajustesPersonaGenerica"))} · ${esc(motivos.find((m) => m.clave === h.motivo_clave)?.texto_clave ? t(motivos.find((m) => m.clave === h.motivo_clave).texto_clave) : t("ajustesMotivoNoIdentificado"))}
      <ul>${h.cambios.filter((c) => c.regla_clave === r.clave).map((c) => `<li>${esc(etiquetaCampo(c.campo))}: ${esc(valorHistorico(c.campo, c.anterior))} → ${esc(valorHistorico(c.campo, c.nuevo))}</li>`).join("")}</ul>
      ${h.referencia ? `<p>${esc(t("ajustesReferenciaHistoria", { referencia: h.referencia }))}</p>` : ""}
      ${h.nota ? `<p>${esc(t("ajustesNotaHistoria", { nota: h.nota }))}</p>` : ""}
      ${h.recibo_ref ? `<details><summary>${esc(t("ajustesVerJustificante"))}</summary><code>${esc(h.recibo_ref)}</code></details>` : ""}</li>`).join("") : `<li>${esc(t("ajustesSinHistoria"))}</li>`;
    const accion = modelo.puede_ajustar ? `<button type="button" class="rg-secundario" data-ajustes-editar="${esc(r.clave)}" ${editable && !r.ajuste_no_aplicable ? "" : "disabled"}>${esc(t("ajustesCambiar"))}</button>` : "";
    const motivoDesactivado = modelo.puede_ajustar && !motivos.length ? `<p class="rg-aviso">${esc(t("ajustesSinMotivos"))}</p>` : "";
    const revision = r.ajuste_no_aplicable ? `<p class="rg-aviso rg-aviso--error">${esc(t("ajusteRevisionDetalle"))}</p>` : "";
    return `<article class="rg-ajuste-regla" aria-labelledby="rg-ajuste-${esc(r.clave)}"><div class="rg-ajuste-cabecera"><h3 id="rg-ajuste-${esc(r.clave)}"${nombreCatalogado || idiomaAjustes() === idiomaDatosAjustes() ? "" : ` lang="${esc(idiomaDatosAjustes())}"`}>${esc(nombre)}</h3>${accion}</div>
      ${valores ? `<dl class="rg-ajuste-valores">${valores}</dl>` : ""}${revision}${motivoDesactivado}
      ${activa && editable && !r.ajuste_no_aplicable ? renderizarFormulario(r, motivos, borrador, fase) : ""}
      <details class="rg-ajuste-historial"><summary>${esc(t("ajustesHistorial"))}</summary><ol>${listaHistorial}</ol></details></article>`;
  }).join("");
  const historiaSinBase = !baseActiva ? renderizarHistoriaSinBase(modelo.historial, motivos) : "";
  const mensajeBase = !baseActiva ? t(modelo.activacion.estado === "sin_publicar" ? "ajustesBaseSinPublicar" : "ajustesBaseInactiva") : "";
  return `<section class="rg-panel rg-ajustes" aria-labelledby="rg-ajustes-titulo"><div class="rg-panel-cabecera"><div><h2 id="rg-ajustes-titulo" tabindex="-1">${esc(t("ajustesTitulo"))}</h2><p class="rg-meta">${esc(t("ajustesVersion", { version: formatearNumero(modelo.version_esperada) }))}</p></div></div>
    ${usaIdiomaRespaldoAjustes() ? `<p class="rg-aviso" role="status">${esc(t("ajustesIdiomaRespaldo"))} <button type="button" class="rg-secundario" data-ajustes-idioma-reintentar>${esc(t("ajustesReintentarIdioma"))}</button></p>` : ""}
    ${mensajeBase ? `<p class="rg-aviso" role="status">${esc(mensajeBase)}</p>` : !modelo.puede_ajustar ? `<p class="rg-aviso">${esc(t("ajustesSoloLectura"))}</p>` : ""}
    ${aviso ? `<p class="rg-aviso${error ? " rg-aviso--error" : ""}" role="status" tabindex="-1" data-ajustes-estado>${esc(aviso)}${error ? ` <button type="button" class="rg-secundario" data-ajustes-reintentar>${esc(t("reintentar"))}</button>` : ""}</p>` : ""}
    ${recibo ? `<div class="rg-ajuste-recibo" role="status" tabindex="-1" data-ajustes-recibo><strong>${esc(t("ajustesGuardado"))}</strong><span>${esc(t("ajustesVersion", { version: formatearNumero(recibo.version) }))}</span><span>${esc(fecha(recibo.vigente_desde))}</span><details><summary>${esc(t("ajustesVerJustificante"))}</summary><code>${esc(recibo.recibo_ref)}</code></details></div>` : ""}
    <div class="rg-ajustes-cuerpo">${reglas || (baseActiva ? `<p>${esc(t("ajustesSinReglas"))}</p>` : historiaSinBase)}</div>
    ${modelo.hay_mas ? `<div class="rg-ajuste-mas"><button type="button" class="rg-secundario" data-ajustes-mas>${esc(t("ajustesMasHistoria"))}</button></div>` : ""}</section>`;
}

function renderizarFormulario(regla, motivos, borrador, fase) {
  const bloqueado = fase === "revision" ? " disabled" : "";
  const motivoElegido = motivos.find((m) => m.clave === borrador?.motivo_clave);
  const controles = regla.edicion.campos.map((campo) => {
    const valor = borrador?.[campo] ?? regla.valores[campo];
    const opciones = campo === "unidad" ? regla.edicion.opciones_unidad : campo === "computo" ? regla.edicion.opciones_computo : null;
    const control = opciones ? `<select name="${campo}" required${bloqueado}>${opciones.map((opcion) => `<option value="${esc(opcion)}"${opcion === valor ? " selected" : ""}>${esc(etiquetaOpcion(campo, opcion))}</option>`).join("")}</select>`
      : `<input name="${campo}" type="number" inputmode="numeric" required min="${regla.edicion.cantidad_minima}" max="${regla.edicion.cantidad_maxima}" step="1" value="${esc(valor)}"${bloqueado}>`;
    return `<label><span>${esc(etiquetaCampo(campo))}</span>${control}</label>`;
  }).join("");
  const motivo = `<label><span>${esc(t("ajustesMotivo"))}</span><select name="motivo_clave" required${bloqueado}><option value="">${esc(t("ajustesElegirMotivo"))}</option>${motivos.map((m) => `<option value="${esc(m.clave)}"${m.clave === borrador?.motivo_clave ? " selected" : ""}>${esc(t(m.texto_clave))}</option>`).join("")}</select></label>`;
  const resumen = fase === "revision" ? `<div class="rg-ajuste-revision" tabindex="-1" data-ajustes-revision><h4>${esc(t("ajustesRevisar"))}</h4><p>${esc(t("ajustesEfecto"))}</p><dl>${regla.edicion.campos.filter((campo) => borrador[campo] !== regla.valores[campo]).map((campo) => `<div><dt>${esc(etiquetaCampo(campo))}</dt><dd>${esc(presentarValor(regla, campo, regla.valores[campo]))} → ${esc(presentarValor(regla, campo, borrador[campo], borrador.unidad ?? regla.valores.unidad ?? regla.unidad))}</dd></div>`).join("")}
    <div><dt>${esc(t("ajustesMotivo"))}</dt><dd>${esc(motivoElegido ? t(motivoElegido.texto_clave) : t("ajustesMotivoNoIdentificado"))}</dd></div>
    ${borrador.referencia ? `<div><dt>${esc(t("ajustesReferencia"))}</dt><dd>${esc(borrador.referencia)}</dd></div>` : ""}
    ${borrador.nota ? `<div><dt>${esc(t("ajustesNota"))}</dt><dd>${esc(borrador.nota)}</dd></div>` : ""}</dl></div>` : "";
  return `<form class="rg-ajuste-form" data-ajustes-form="${esc(regla.clave)}"><div class="rg-ajuste-campos">${controles}${motivo}
    <label><span>${esc(t("ajustesReferencia"))}</span><input name="referencia" maxlength="120" value="${esc(borrador?.referencia)}"${bloqueado}></label>
    <label class="rg-ajuste-nota"><span>${esc(t("ajustesNota"))}</span><textarea name="nota" maxlength="500"${bloqueado}>${esc(borrador?.nota)}</textarea></label></div>
    ${resumen}<div class="rg-ajuste-acciones">
    <button type="button" class="rg-secundario" ${fase === "revision" ? "data-ajustes-volver" : "data-ajustes-cancelar"}>${esc(t(fase === "revision" ? "ajustesCorregir" : "ajustesCancelar"))}</button>
    <button type="submit" class="rg-secundario" data-ajustes-enviar${fase === "revision" && !motivoElegido ? " disabled" : ""}>${esc(t(fase === "revision" ? "ajustesGuardar" : "ajustesRevisar"))}</button></div></form>`;
}

/** Panel autocontenido. El cliente se inyecta y ninguna respuesta tardía repinta una vista nueva. */
export async function iniciarAjustes(doc, cliente = crearClienteAjustes()) {
  const contenedor = doc.getElementById("rg-ajustes");
  if (!contenedor) return () => {};
  try { await cargarTextosAjustes(); }
  catch {
    let mensaje = "";
    let reintentar = "";
    try {
      const textos = (await cargarTextos("portal")).seccion("textos");
      mensaje = textos.txt_el_servicio_no_esta_disponible_ahora_puede_reint;
      reintentar = textos.txt_reintentar;
    } catch { /* El catálogo de apoyo también está temporalmente ausente. */ }
    contenedor.innerHTML = `<section class="rg-panel rg-ajustes"><p class="rg-aviso rg-aviso--error" role="alert">${esc(mensaje)}</p><button type="button" class="rg-secundario" data-ajustes-catalogo-reintentar>${esc(reintentar)}</button></section>`;
    let desmontado = false;
    contenedor.querySelector("[data-ajustes-catalogo-reintentar]")?.addEventListener("click", () => {
      void reintentarTextosAjustes().then(() => { if (!desmontado) void iniciarAjustes(doc, cliente); })
        .catch(() => { if (!desmontado) void iniciarAjustes(doc, cliente); });
    }, { once: true });
    return () => { desmontado = true; };
  }
  let modelo = null;
  let reglaActiva = "";
  let borrador = null;
  let fase = "lista";
  let aviso = "";
  let error = false;
  let recibo = null;
  let pendiente = null;
  let enviando = false;
  let bloqueado = false;
  let solicitud = null;
  let lecturaEnCurso = false;
  let desmontado = false;
  let secuencia = 0;
  const enfocar = (selector) => contenedor.querySelector(selector)?.focus();
  const pintar = () => {
    if (!modelo) return;
    contenedor.innerHTML = renderizarAjustes(modelo, { reglaActiva, borrador, fase, aviso, error, recibo, bloqueado });
  };
  const comunicar = (clave, fallo = false) => {
    aviso = t(clave); error = fallo; pintar();
    enfocar("[data-ajustes-estado]");
  };
  const cargar = async ({ antesDeVersion, conservarBorrador = false } = {}) => {
    if (lecturaEnCurso || desmontado) return false;
    lecturaEnCurso = true;
    const reintentoInicial = !modelo && Boolean(contenedor.querySelector("[data-ajustes-reintentar]"));
    solicitud = new AbortController();
    const actual = ++secuencia;
    if (!modelo) contenedor.innerHTML = `<p class="rg-aviso" role="status">${esc(t("ajustesCargando"))}</p>`;
    try {
      const datos = await cliente.leer({ antesDeVersion, signal: solicitud.signal });
      if (actual !== secuencia || desmontado) return false;
      if (antesDeVersion !== undefined && modelo) {
        modelo = { ...datos, historial: [...modelo.historial, ...datos.historial] };
      } else modelo = datos;
      if (!conservarBorrador) { reglaActiva = ""; borrador = null; fase = "lista"; pendiente = null; }
      bloqueado = false;
      aviso = ""; error = false;
      pintar();
      if (conservarBorrador && borrador) enfocar("[data-ajustes-form] input, [data-ajustes-form] select, #rg-ajustes-titulo");
      else if (antesDeVersion !== undefined) enfocar("[data-ajustes-mas], #rg-ajustes-titulo");
      else if (reintentoInicial) enfocar("#rg-ajustes-titulo");
      return true;
    } catch (e) {
      if (actual !== secuencia || desmontado) return false;
      comunicar(e instanceof ErrorAjustes && existeClaveReglas(e.codigo) ? e.codigo : "ajustesNoDisponible", true);
      if (!modelo) {
        contenedor.innerHTML = `<section class="rg-panel rg-ajustes" aria-labelledby="rg-ajustes-titulo"><div class="rg-panel-cabecera"><h2 id="rg-ajustes-titulo">${esc(t("ajustesTitulo"))}</h2></div><p class="rg-aviso rg-aviso--error" role="status" tabindex="-1" data-ajustes-estado>${esc(aviso)}</p><div class="rg-ajuste-mas"><button type="button" class="rg-secundario" data-ajustes-reintentar>${esc(t("reintentar"))}</button></div></section>`;
        enfocar("[data-ajustes-estado]");
      }
      return false;
    } finally {
      lecturaEnCurso = false;
      solicitud = null;
    }
  };
  const cambios = (regla) => regla.edicion.campos.filter((campo) => borrador?.[campo] !== regla.valores[campo])
    .map((campo) => ({ regla_clave: regla.clave, campo, nuevo: String(borrador[campo]) }));
  const leerFormulario = (form, regla) => {
    const datos = new FormData(form);
    const siguiente = { motivo_clave: String(datos.get("motivo_clave") ?? ""), referencia: String(datos.get("referencia") ?? "").trim(),
      nota: String(datos.get("nota") ?? "").trim() };
    for (const campo of regla.edicion.campos) siguiente[campo] = String(datos.get(campo) ?? "");
    return siguiente;
  };
  const publicar = async (regla) => {
    if (enviando || desmontado) return;
    if (!globalThis.crypto?.randomUUID) { comunicar("ajustesNoDisponible", true); return; }
    const nuevosCambios = cambios(regla);
    if (!nuevosCambios.length) { comunicar("ajustesSinCambios", true); return; }
    const comando = pendiente ?? { clave_idempotencia: globalThis.crypto.randomUUID(), version_esperada: modelo.version_esperada,
      cambios: nuevosCambios, motivo_clave: borrador.motivo_clave,
      ...(borrador.referencia ? { referencia: borrador.referencia } : {}), ...(borrador.nota ? { nota: borrador.nota } : {}) };
    pendiente = comando;
    enviando = true;
    comunicar("ajustesGuardando");
    try {
      const resultado = await cliente.guardar(comando);
      if (desmontado) return;
      recibo = resultado.recibo;
      pendiente = null;
      const recuperado = await cargar();
      if (desmontado) return;
      aviso = t(recuperado ? "ajustesGuardado" : "ajustesGuardadoSinLectura"); error = !recuperado; pintar();
      enfocar("[data-ajustes-recibo]");
    } catch (e) {
      if (desmontado) return;
      if (e instanceof ErrorAjustes && e.codigo === "ajustesConflicto") {
        pendiente = null;
        fase = "edicion";
        const renovado = await cargar({ conservarBorrador: true });
        if (!renovado) { bloqueado = true; comunicar("ajustesConflictoSinLectura", true); return; }
        comunicar("ajustesConflicto", true);
      } else comunicar(e instanceof ErrorAjustes && existeClaveReglas(e.codigo) ? e.codigo : "ajustesNoDisponible", true);
    } finally { enviando = false; }
  };
  contenedor.addEventListener("click", (evento) => {
    if (evento.target.closest?.("[data-ajustes-idioma-reintentar]")) {
      void reintentarTextosAjustes().then(pintar).catch(() => {});
      return;
    }
    if (evento.target.closest?.("[data-ajustes-reintentar]")) { void cargar({ conservarBorrador: !recibo && Boolean(borrador) }); return; }
    if (evento.target.closest?.("[data-ajustes-mas]")) {
      const menor = modelo?.historial.at(-1)?.version;
      if (Number.isInteger(menor) && menor > 1) void cargar({ antesDeVersion: menor });
      return;
    }
    if (evento.target.closest?.("[data-ajustes-cancelar]")) {
      const clave = reglaActiva;
      reglaActiva = ""; borrador = null; pendiente = null; fase = "lista"; aviso = ""; pintar();
      [...contenedor.querySelectorAll("[data-ajustes-editar]")].find((boton) => boton.dataset.ajustesEditar === clave)?.focus();
      return;
    }
    if (evento.target.closest?.("[data-ajustes-volver]")) {
      pendiente = null; fase = "edicion"; aviso = ""; pintar();
      enfocar("[data-ajustes-form] input, [data-ajustes-form] select");
      return;
    }
    const boton = evento.target.closest?.("[data-ajustes-editar]");
    if (boton && !boton.disabled && modelo?.puede_ajustar && modelo.motivos?.length) {
      reglaActiva = boton.dataset.ajustesEditar; borrador = null; pendiente = null; fase = "edicion"; recibo = null; aviso = ""; pintar();
      contenedor.querySelector("[data-ajustes-form] input, [data-ajustes-form] select")?.focus();
    }
  });
  contenedor.addEventListener("submit", (evento) => {
    const form = evento.target.closest?.("[data-ajustes-form]");
    if (!form) return;
    evento.preventDefault();
    const regla = modelo?.reglas.find((r) => r.clave === form.dataset.ajustesForm);
    if (!regla || bloqueado || !modelo.puede_ajustar || !modelo.motivos?.length || regla.ajuste_no_aplicable) return;
    if (fase !== "revision") {
      borrador = leerFormulario(form, regla);
      if (!modelo.motivos.some((m) => m.clave === borrador.motivo_clave)) { comunicar("ajustesMotivoRequerido", true); return; }
      if (!cambios(regla).length) { comunicar("ajustesSinCambios", true); return; }
      fase = "revision"; aviso = ""; pintar();
      enfocar("[data-ajustes-revision]");
    } else if (modelo.motivos.some((m) => m.clave === borrador?.motivo_clave)) void publicar(regla);
    else comunicar("ajustesMotivoRequerido", true);
  });
  void cargar();
  return () => { desmontado = true; ++secuencia; solicitud?.abort(); };
}
