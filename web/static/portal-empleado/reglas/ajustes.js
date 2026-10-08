import { cargarTextosAjustes, existeClaveAjustes, fechaAjustes, idiomaAjustes,
  idiomaDatosAjustes, numeroAjustes, reintentarTextosAjustes, traducirAjustes,
  usaIdiomaRespaldoAjustes } from "./ajustes-i18n.js";
import { cargarTextos } from "../../comun/textos.js";

export const API_AJUSTES = "/api/vec/contratacion-temporal/reglas/ajustes";
const ESQUEMA = "vec.contratacion_temporal.reglas.ajustes.v1";
const CLAVE = /^[a-z][a-z0-9._]{2,95}$/u;
const clave = (valor) => typeof valor === "string" && CLAVE.test(valor);
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
const instanteRFC3339 = (valor) => typeof valor === "string"
  && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/u.test(valor)
  && !Number.isNaN(Date.parse(valor)) && new Date(valor).toISOString().replace(".000Z", "Z") === valor;

/** Rechaza las horas inexistentes o ambiguas en los cambios de horario de Madrid. */
export function normalizarFechaMadrid(valor, ahora = Date.now()) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/u.test(valor)) throw new ErrorAjustes("ajustesFechaInvalida");
  const [anio, mes, dia, hora, minuto] = valor.match(/\d+/gu).map(Number);
  const base = Date.UTC(anio, mes - 1, dia, hora, minuto);
  const formato = new Intl.DateTimeFormat(undefined, { timeZone: "Europe/Madrid", numberingSystem: "latn", year: "numeric", month: "2-digit",
    day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23" });
  const candidatos = [1, 2].map((desfase) => base - desfase * 3600000).filter((utc) => {
    const partes = Object.fromEntries(formato.formatToParts(utc).map(({ type, value }) => [type, value]));
    return Number(partes.year) === anio && Number(partes.month) === mes && Number(partes.day) === dia
      && Number(partes.hour) === hora && Number(partes.minute) === minuto;
  });
  if (candidatos.length !== 1 || candidatos[0] <= ahora) throw new ErrorAjustes("ajustesFechaInvalida");
  return new Date(candidatos[0]).toISOString().replace(".000Z", "Z");
}
const CONTRATOS = Object.freeze({
  contratacion_temporal: Object.freeze({ modulo: "contratacion_temporal", ruta: API_AJUSTES,
    esquema: ESQUEMA, catalogoID: "vec.contratacion_temporal.reglas.ajustes" }),
  bolsa: Object.freeze({ modulo: "bolsa", ruta: "/api/vec/bolsa/reglas/ajustes",
    esquema: "vec.bolsa.reglas.ajustes.v1", catalogoID: "vec.bolsa.reglas.ajustes" }),
});
const CONTRATO_CT = CONTRATOS.contratacion_temporal;
function contratoAjustes(entrada = CONTRATO_CT) {
  const modulo = typeof entrada === "string" ? entrada : entrada?.modulo;
  const c = CONTRATOS[modulo];
  if (!c || typeof entrada === "object" && Object.keys(entrada).some((clave) => c[clave] !== entrada[clave])) {
    throw new TypeError("contrato de ajustes no válido");
  }
  return c;
}
const validarEdicion = (edicion) => {
  exigir(edicion && Array.isArray(edicion.campos) && edicion.campos.length > 0
    && edicion.campos.every((campo) => CAMPOS.has(campo))
    && Array.isArray(edicion.opciones_unidad) && edicion.opciones_unidad.length <= 16
    && Array.isArray(edicion.opciones_computo) && edicion.opciones_computo.length <= 16
    && Number.isSafeInteger(edicion.cantidad_minima) && Number.isSafeInteger(edicion.cantidad_maxima)
    && edicion.cantidad_minima > 0 && edicion.cantidad_maxima >= edicion.cantidad_minima);
  for (const opcion of edicion.opciones_unidad) exigir(clave(opcion) && existeClaveReglas(`unidad_${opcion}`));
  for (const opcion of edicion.opciones_computo) exigir(clave(opcion) && existeClaveReglas(`computo_${opcion}`));
};

/** La edición solo se habilita con metadatos recibidos completos y comprobables. */
export function validarLecturaAjustes(respuesta, contrato = CONTRATO_CT) {
  const c = contratoAjustes(contrato);
  const d = respuesta?.data;
  exigir(d?.esquema === c.esquema && d.catalogo_id === c.catalogoID
    && version(d.version_esperada) && typeof d.puede_ajustar === "boolean" && Array.isArray(d.reglas)
    && d.reglas.length <= 64 && Array.isArray(d.historial) && d.historial.length <= 50
    && typeof d.hay_mas === "boolean" && d.activacion
    && ["activa", "inactiva", "sin_publicar"].includes(d.activacion.estado)
    && Object.keys(d.activacion).length === 1);
  const versionPublicada = (v) => v && version(v.version) && v.version > 0
    && typeof v.vigente_desde === "string" && !Number.isNaN(Date.parse(v.vigente_desde))
    && typeof v.publicada_en === "string" && !Number.isNaN(Date.parse(v.publicada_en));
  exigir((d.cabeza === null && d.version_esperada === 0 || versionPublicada(d.cabeza) && d.cabeza.version === d.version_esperada)
    && (d.vigente_hoy === null || versionPublicada(d.vigente_hoy) && d.vigente_hoy.version <= d.version_esperada)
    && Array.isArray(d.programados) && d.programados.length <= 256
    && d.programados.every((p) => versionPublicada(p) && p.version <= d.version_esperada
      && p.version > (d.vigente_hoy?.version ?? 0))
    && d.programados.every((p, i) => i === 0 || Date.parse(d.programados[i - 1].vigente_desde) <= Date.parse(p.vigente_desde)));
  exigir(d.activacion.estado === "activa" || (!d.puede_ajustar && d.reglas.length === 0));
  exigir(d.motivos === undefined || d.motivos === null || (Array.isArray(d.motivos) && d.motivos.length <= 16
    && d.motivos.every((m) => clave(m?.clave) && cadena(m.texto_clave, 100) && existeClaveReglas(m.texto_clave))));
  for (const r of d.reglas) {
    exigir(clave(r?.clave) && cadena(r.etiqueta) && r.etiqueta.length > 0
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
      && clave(h.motivo_clave) && (h.referencia === undefined || cadena(h.referencia, 120))
      && (h.nota === undefined || cadena(h.nota, 500))
      && (h.actor_nombre === undefined || cadena(h.actor_nombre, 160))
      && typeof h.publicada_en === "string" && !Number.isNaN(Date.parse(h.publicada_en))
      && Array.isArray(h.cambios) && h.cambios.length <= 64);
    for (const c of h.cambios) exigir(clave(c?.regla_clave) && CAMPOS.has(c.campo)
      && cadena(c.anterior, 80) && cadena(c.nuevo, 80));
  }
  return { ...d, motivos: d.motivos ?? [] };
}

export function validarReciboAjustes(respuesta, contrato = CONTRATO_CT) {
  const c = contratoAjustes(contrato);
  const r = respuesta?.data?.recibo;
  exigir(respuesta?.data?.esquema === c.esquema && r && cadena(r.recibo_ref) && r.recibo_ref.length > 0 && version(r.version) && r.version > 0
    && cadena(r.vigente_desde, 64) && r.vigente_desde.length > 0
    && cadena(r.publicada_en, 64) && !Number.isNaN(Date.parse(r.publicada_en))
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

export function crearClienteAjustes(fetchImpl = globalThis.fetch, timeoutMs = 10000, contrato = CONTRATO_CT) {
  const c = contratoAjustes(contrato);
  return Object.freeze({
    leer: async ({ antesDeVersion, signal } = {}) => {
      const controlador = new AbortController();
      if (signal?.aborted) controlador.abort();
      signal?.addEventListener("abort", () => controlador.abort(), { once: true });
      const parametros = new URLSearchParams({ limite: "20" });
      if (antesDeVersion !== undefined) parametros.set("antes_de_version", String(antesDeVersion));
      const json = await pedir(fetchImpl, `${c.ruta}?${parametros}`, { method: "GET", headers: { Accept: "application/json" } }, controlador, timeoutMs);
      return validarLecturaAjustes(json, c);
    },
    guardar: async (comando, { signal } = {}) => {
      const campos = new Set(["clave_idempotencia", "version_esperada", "cambios", "motivo_clave", "vigente_desde", "referencia", "nota"]);
      if (!comando || typeof comando !== "object" || Object.keys(comando).some((clave) => !campos.has(clave))
        || !cadena(comando.clave_idempotencia, 80) || !comando.clave_idempotencia
        || !version(comando.version_esperada) || !clave(comando.motivo_clave)
        || !Array.isArray(comando.cambios) || comando.cambios.length < 1 || comando.cambios.length > 256
        || comando.cambios.some((cambio) => !cambio || Object.keys(cambio).some((clave) => !["regla_clave", "campo", "nuevo"].includes(clave))
          || !clave(cambio.regla_clave) || !CAMPOS.has(cambio.campo)
          || !cadena(cambio.nuevo, 80) || !cambio.nuevo)
        || comando.referencia !== undefined && !cadena(comando.referencia, 120)
        || comando.nota !== undefined && !cadena(comando.nota, 500)) throw new ErrorAjustes("ajustesValorInvalido");
      const controlador = new AbortController();
      if (signal?.aborted) controlador.abort();
      signal?.addEventListener("abort", () => controlador.abort(), { once: true });
      if (comando.vigente_desde !== undefined && !instanteRFC3339(comando.vigente_desde)) throw new ErrorAjustes("ajustesFechaInvalida");
      const json = await pedir(fetchImpl, c.ruta, { method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify(comando) }, controlador, timeoutMs);
      const resultado = validarReciboAjustes(json, c);
      exigir(resultado.recibo.clave_idempotencia === comando.clave_idempotencia
        && resultado.recibo.version === comando.version_esperada + 1);
      return resultado;
    },
  });
}

const CLAVES_CAMPO = Object.freeze({
  cantidad: "ajustesCampo_cantidad",
  cantidad_urgente: "ajustesCampo_cantidad_urgente",
  unidad: "ajustesCampo_unidad",
  computo: "ajustesCampo_computo",
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
    const cambios = h.cambios.map((c) => `<li>${esc(etiquetaCampo(c.campo))}: ${esc(valorHistorico(c.campo, c.anterior))} → ${esc(valorHistorico(c.campo, c.nuevo))} <small>${esc(t("ajustesReglaHistorica"))} <code>${esc(c.regla_clave)}</code></small></li>`).join("");
    return `<li><span>${esc(fecha(h.vigente_desde))}</span> · ${esc(h.actor_nombre || t("ajustesAutorNoDisponible"))} · ${esc(t("ajustesVersion", { version: formatearNumero(h.version) }))} · ${esc(motivoTexto)}
      <ul>${cambios}</ul>
      ${h.referencia ? `<p>${esc(t("ajustesReferenciaHistoria", { referencia: h.referencia }))}</p>` : ""}
      ${h.nota ? `<p>${esc(t("ajustesNotaHistoria", { nota: h.nota }))}</p>` : ""}
      ${h.recibo_ref ? `<details><summary>${esc(t("ajustesVerJustificante"))}</summary><code>${esc(h.recibo_ref)}</code></details>` : ""}</li>`;
  }).join("");
  return `<details class="rg-ajuste-historial"><summary>${esc(t("ajustesHistorial"))}</summary><ol>${entradas}</ol></details>`;
}

export function renderizarAjustes(modelo, { reglaActiva = "", borrador = null, fase = "lista", aviso = "", error = false, recibo = null, bloqueado = false,
  modulo = "contratacion_temporal" } = {}) {
  contratoAjustes(modulo);
  const motivos = modelo.motivos ?? [];
  const baseActiva = modelo.activacion.estado === "activa";
  const editable = modelo.puede_ajustar && motivos.length > 0 && !bloqueado;
  const reglas = modelo.reglas.map((r) => {
    const activa = r.clave === reglaActiva;
    const valores = r.ajuste_no_aplicable ? "" : r.edicion.campos.map((campo) => `<div><dt>${esc(etiquetaCampo(campo))}</dt><dd>${esc(presentarValor(r, campo, r.valores[campo]))}</dd></div>`).join("");
    const historial = modelo.historial.filter((h) => h.cambios.some((c) => c.regla_clave === r.clave));
    const listaHistorial = historial.length ? historial.map((h) => `<li><span>${esc(fecha(h.vigente_desde))}</span> · ${esc(h.actor_nombre || t("ajustesAutorNoDisponible"))} · ${esc(motivos.find((m) => m.clave === h.motivo_clave)?.texto_clave ? t(motivos.find((m) => m.clave === h.motivo_clave).texto_clave) : t("ajustesMotivoNoIdentificado"))}
      <ul>${h.cambios.filter((c) => c.regla_clave === r.clave).map((c) => `<li>${esc(etiquetaCampo(c.campo))}: ${esc(valorHistorico(c.campo, c.anterior))} → ${esc(valorHistorico(c.campo, c.nuevo))}</li>`).join("")}</ul>
      ${h.referencia ? `<p>${esc(t("ajustesReferenciaHistoria", { referencia: h.referencia }))}</p>` : ""}
      ${h.nota ? `<p>${esc(t("ajustesNotaHistoria", { nota: h.nota }))}</p>` : ""}
      ${h.recibo_ref ? `<details><summary>${esc(t("ajustesVerJustificante"))}</summary><code>${esc(h.recibo_ref)}</code></details>` : ""}</li>`).join("") : `<li>${esc(t("ajustesSinHistoria"))}</li>`;
    const accion = modelo.puede_ajustar ? `<button type="button" class="rg-secundario" data-ajustes-editar="${esc(r.clave)}" ${editable && !r.ajuste_no_aplicable ? "" : "disabled"}>${esc(t("ajustesCambiar"))}</button>` : "";
    const motivoDesactivado = modelo.puede_ajustar && !motivos.length ? `<p class="rg-aviso">${esc(t("ajustesSinMotivos"))}</p>` : "";
    const revision = r.ajuste_no_aplicable ? `<p class="rg-aviso rg-aviso--error">${esc(t("ajusteRevisionDetalle"))}</p>` : "";
    return `<article class="rg-ajuste-regla" aria-labelledby="rg-ajuste-${esc(r.clave)}"><div class="rg-ajuste-cabecera"><h3 id="rg-ajuste-${esc(r.clave)}"${idiomaAjustes() === idiomaDatosAjustes() ? "" : ` lang="${esc(idiomaDatosAjustes())}"`}>${esc(r.etiqueta)}</h3>${accion}</div>
      ${valores ? `<dl class="rg-ajuste-valores">${valores}</dl>` : ""}${revision}${motivoDesactivado}
      ${activa && editable && !r.ajuste_no_aplicable ? renderizarFormulario(r, motivos, borrador, fase) : ""}
      <details class="rg-ajuste-historial"><summary>${esc(t("ajustesHistorial"))}</summary><ol>${listaHistorial}</ol></details></article>`;
  }).join("");
  const historiaSinBase = !baseActiva ? renderizarHistoriaSinBase(modelo.historial, motivos) : "";
  const mensajeBase = !baseActiva ? t(modelo.activacion.estado === "sin_publicar" ? "ajustesBaseSinPublicar" : "ajustesBaseInactiva") : "";
  const programados = modelo.programados?.length ? `<details class="rg-ajuste-historial"><summary>${esc(t("ajustesProgramados"))}</summary><ol>${modelo.programados.map((p) => `<li>${esc(t("ajustesProgramado", { version: formatearNumero(p.version), fecha: fecha(p.vigente_desde) }))}</li>`).join("")}</ol></details>` : "";
  const versionVigente = baseActiva && modelo.vigente_hoy
    ? t("ajustesVersionVigente", { version: formatearNumero(modelo.vigente_hoy.version) }) : t("ajustesSinVersionVigente");
  const versionCabeza = modelo.cabeza && modelo.cabeza.version !== modelo.vigente_hoy?.version
    ? `<p class="rg-meta">${esc(t("ajustesVersionCabeza", { version: formatearNumero(modelo.cabeza.version) }))}</p>` : "";
  return `<section class="rg-panel rg-ajustes" aria-labelledby="rg-ajustes-titulo"><div class="rg-panel-cabecera"><div><h2 id="rg-ajustes-titulo" tabindex="-1">${esc(t(modulo === "bolsa" ? "ajustesTitulo_bolsa" : "ajustesTitulo"))}</h2><p class="rg-meta">${esc(versionVigente)}</p>${versionCabeza}</div></div>
    ${usaIdiomaRespaldoAjustes() ? `<p class="rg-aviso" role="status">${esc(t("ajustesIdiomaRespaldo"))} <button type="button" class="rg-secundario" data-ajustes-idioma-reintentar>${esc(t("ajustesReintentarIdioma"))}</button></p>` : ""}
    ${mensajeBase ? `<p class="rg-aviso" role="status">${esc(mensajeBase)}</p>` : !modelo.puede_ajustar ? `<p class="rg-aviso">${esc(t("ajustesSoloLectura"))}</p>` : ""}
    ${aviso ? `<p class="rg-aviso${error ? " rg-aviso--error" : ""}" role="status" tabindex="-1" data-ajustes-estado>${esc(aviso)}${error ? ` <button type="button" class="rg-secundario" data-ajustes-reintentar>${esc(t("reintentar"))}</button>` : ""}</p>` : ""}
    ${recibo ? `<div class="rg-ajuste-recibo" role="status" tabindex="-1" data-ajustes-recibo><strong>${esc(t("ajustesGuardado"))}</strong><span>${esc(t("ajustesVersion", { version: formatearNumero(recibo.version) }))}</span><span>${esc(t("ajustesFechaEfectoConfirmada", { fecha: fecha(recibo.vigente_desde) }))}</span><span>${esc(t("ajustesRegistradoEn", { fecha: fecha(recibo.publicada_en) }))}</span><code>${esc(recibo.recibo_ref)}</code>${recibo.auditoria_ref ? `<span>${esc(t("ajustesAuditoria"))} <code>${esc(recibo.auditoria_ref)}</code></span>` : ""}</div>` : ""}
    ${programados}
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
  const inicio = borrador?.inicio_efecto === "futuro" ? "futuro" : "ahora";
  const valorFecha = borrador?.fecha_madrid ?? "";
  const fechaControl = `<fieldset class="rg-ajuste-nota"><legend>${esc(t("ajustesInicioEfecto"))}</legend><label><input type="radio" name="inicio_efecto" value="ahora"${inicio === "ahora" ? " checked" : ""}${bloqueado}>${esc(t("ajustesDesdeAhora"))}</label><label><input type="radio" name="inicio_efecto" value="futuro"${inicio === "futuro" ? " checked" : ""}${bloqueado}>${esc(t("ajustesFechaFutura"))}</label>${inicio === "futuro" ? `<label><span>${esc(t("ajustesFechaHoraMadrid"))}</span><input name="fecha_madrid" type="datetime-local" required value="${esc(valorFecha)}"${bloqueado}></label>` : ""}</fieldset>`;
  const efecto = inicio === "futuro" ? t("ajustesEfectoFuturo", { fecha: fecha(borrador.vigente_desde) }) : t("ajustesEfecto");
  const resumen = fase === "revision" ? `<div class="rg-ajuste-revision" tabindex="-1" data-ajustes-revision><h4>${esc(t("ajustesRevisar"))}</h4><p>${esc(efecto)}</p><dl>${regla.edicion.campos.filter((campo) => borrador[campo] !== regla.valores[campo]).map((campo) => `<div><dt>${esc(etiquetaCampo(campo))}</dt><dd>${esc(presentarValor(regla, campo, regla.valores[campo]))} → ${esc(presentarValor(regla, campo, borrador[campo], borrador.unidad ?? regla.valores.unidad ?? regla.unidad))}</dd></div>`).join("")}
    <div><dt>${esc(t("ajustesMotivo"))}</dt><dd>${esc(motivoElegido ? t(motivoElegido.texto_clave) : t("ajustesMotivoNoIdentificado"))}</dd></div>
    ${borrador.referencia ? `<div><dt>${esc(t("ajustesReferencia"))}</dt><dd>${esc(borrador.referencia)}</dd></div>` : ""}
    ${borrador.nota ? `<div><dt>${esc(t("ajustesNota"))}</dt><dd>${esc(borrador.nota)}</dd></div>` : ""}</dl></div>` : "";
  return `<form class="rg-ajuste-form" data-ajustes-form="${esc(regla.clave)}"><div class="rg-ajuste-campos">${controles}${motivo}${fechaControl}
    <label><span>${esc(t("ajustesReferencia"))}</span><input name="referencia" maxlength="120" value="${esc(borrador?.referencia)}"${bloqueado}></label>
    <label class="rg-ajuste-nota"><span>${esc(t("ajustesNota"))}</span><textarea name="nota" maxlength="500"${bloqueado}>${esc(borrador?.nota)}</textarea></label></div>
    ${resumen}<div class="rg-ajuste-acciones">
    <button type="button" class="rg-secundario" ${fase === "revision" ? "data-ajustes-volver" : "data-ajustes-cancelar"}>${esc(t(fase === "revision" ? "ajustesCorregir" : "ajustesCancelar"))}</button>
    <button type="submit" class="rg-secundario" data-ajustes-enviar${fase === "revision" && !motivoElegido ? " disabled" : ""}>${esc(t(fase === "revision" ? "ajustesGuardar" : "ajustesRevisar"))}</button></div></form>`;
}

/** Panel autocontenido. El cliente se inyecta y ninguna respuesta tardía repinta una vista nueva. */
export async function iniciarAjustes(doc, cliente = crearClienteAjustes(), { modulo = "contratacion_temporal" } = {}) {
  contratoAjustes(modulo);
  const contenedor = doc.getElementById("rg-ajustes");
  if (!contenedor) return () => {};
  try { await cargarTextosAjustes(); }
  catch (causa) {
    let basico;
    try { basico = await cargarTextos("reglas"); }
    catch { throw causa; }
    let desmontado = false;
    let siguiente = null;
    contenedor.innerHTML = `<section class="rg-panel rg-ajustes"><p class="rg-aviso rg-aviso--error" role="alert">${esc(basico.traducir("general.error_servicio_no_disponible"))}</p><button type="button" class="rg-secundario" data-ajustes-catalogo-reintentar>${esc(basico.traducir("general.reintentar"))}</button></section>`;
    contenedor.querySelector("[data-ajustes-catalogo-reintentar]")?.addEventListener("click", () => {
      void iniciarAjustes(doc, cliente, { modulo }).then((deshacer) => {
        if (desmontado) deshacer(); else siguiente = deshacer;
      });
    }, { once: true });
    return () => { desmontado = true; siguiente?.(); };
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
  let escritura = null;
  let desmontado = false;
  let secuencia = 0;
  const enfocar = (selector) => contenedor.querySelector(selector)?.focus();
  const pintar = () => {
    if (!modelo) return;
    contenedor.innerHTML = renderizarAjustes(modelo, { reglaActiva, borrador, fase, aviso, error, recibo, bloqueado, modulo });
  };
  const comunicar = (clave, fallo = false) => {
    aviso = t(clave); error = fallo; pintar();
    enfocar("[data-ajustes-estado]");
  };
  const cargar = async ({ antesDeVersion, conservarBorrador = false } = {}) => {
    const reintentoInicial = !modelo && Boolean(contenedor.querySelector("[data-ajustes-reintentar]"));
    solicitud?.abort();
    solicitud = new AbortController();
    const actual = ++secuencia;
    if (!modelo) contenedor.innerHTML = `<p class="rg-aviso" role="status">${esc(t("ajustesCargando"))}</p>`;
    try {
      const datos = await cliente.leer({ antesDeVersion, signal: solicitud.signal });
      if (actual !== secuencia) return;
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
      if (actual !== secuencia) return false;
      comunicar(e instanceof ErrorAjustes && existeClaveReglas(e.codigo) ? e.codigo : "ajustesNoDisponible", true);
      if (!modelo) {
        contenedor.innerHTML = `<section class="rg-panel rg-ajustes" aria-labelledby="rg-ajustes-titulo"><div class="rg-panel-cabecera"><h2 id="rg-ajustes-titulo">${esc(t("ajustesTitulo"))}</h2></div><p class="rg-aviso rg-aviso--error" role="status" tabindex="-1" data-ajustes-estado>${esc(aviso)}</p><div class="rg-ajuste-mas"><button type="button" class="rg-secundario" data-ajustes-reintentar>${esc(t("reintentar"))}</button></div></section>`;
        enfocar("[data-ajustes-estado]");
      }
      return false;
    }
  };
  const cambios = (regla) => regla.edicion.campos.filter((campo) => borrador?.[campo] !== regla.valores[campo])
    .map((campo) => ({ regla_clave: regla.clave, campo, nuevo: String(borrador[campo]) }));
  const leerFormulario = (form, regla) => {
    const datos = new FormData(form);
    const siguiente = { motivo_clave: String(datos.get("motivo_clave") ?? ""), referencia: String(datos.get("referencia") ?? "").trim(),
      nota: String(datos.get("nota") ?? "").trim(), inicio_efecto: String(datos.get("inicio_efecto") ?? "ahora"),
      fecha_madrid: String(datos.get("fecha_madrid") ?? borrador?.fecha_madrid ?? "") };
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
      ...(borrador.vigente_desde ? { vigente_desde: borrador.vigente_desde } : {}),
      ...(borrador.referencia ? { referencia: borrador.referencia } : {}), ...(borrador.nota ? { nota: borrador.nota } : {}) };
    pendiente = comando;
    enviando = true;
    escritura = new AbortController();
    comunicar("ajustesGuardando");
    try {
      const resultado = await cliente.guardar(comando, { signal: escritura.signal });
      if (desmontado) return;
      recibo = resultado.recibo;
      const leido = await cargar();
      if (desmontado) return;
      aviso = t(leido ? "ajustesGuardado" : "ajustesGuardadoSinLectura"); error = !leido; pintar();
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
    } finally { enviando = false; escritura = null; }
  };
  contenedor.addEventListener("click", (evento) => {
    if (evento.target.closest?.("[data-ajustes-idioma-reintentar]")) {
      void reintentarTextosAjustes().then(() => { if (!desmontado) pintar(); }).catch(() => { if (!desmontado) pintar(); });
      return;
    }
    if (evento.target.closest?.("[data-ajustes-reintentar]")) { void cargar({ conservarBorrador: Boolean(borrador) }); return; }
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
  contenedor.addEventListener("change", (evento) => {
    if (evento.target?.name !== "inicio_efecto" || fase !== "edicion") return;
    const form = evento.target.closest?.("[data-ajustes-form]");
    const regla = modelo?.reglas.find((r) => r.clave === form?.dataset.ajustesForm);
    if (!regla) return;
    borrador = leerFormulario(form, regla);
    pendiente = null;
    pintar();
    contenedor.querySelector(`[data-ajustes-form="${regla.clave}"] [name="inicio_efecto"][value="${borrador.inicio_efecto}"]`)?.focus();
  });
  contenedor.addEventListener("submit", (evento) => {
    const form = evento.target.closest?.("[data-ajustes-form]");
    if (!form) return;
    evento.preventDefault();
    const regla = modelo?.reglas.find((r) => r.clave === form.dataset.ajustesForm);
    if (!regla || bloqueado || !modelo.puede_ajustar || !modelo.motivos?.length || regla.ajuste_no_aplicable) return;
    if (fase !== "revision") {
      if (!form.reportValidity()) return;
      borrador = leerFormulario(form, regla);
      try {
        borrador.vigente_desde = borrador.inicio_efecto === "futuro" ? normalizarFechaMadrid(borrador.fecha_madrid) : undefined;
      } catch (e) { comunicar(e instanceof ErrorAjustes ? e.codigo : "ajustesFechaInvalida", true); return; }
      if (!modelo.motivos.some((m) => m.clave === borrador.motivo_clave)) { comunicar("ajustesMotivoRequerido", true); return; }
      if (!cambios(regla).length) { comunicar("ajustesSinCambios", true); return; }
      fase = "revision"; aviso = ""; pintar();
      enfocar("[data-ajustes-revision]");
    } else if (modelo.motivos.some((m) => m.clave === borrador?.motivo_clave)) void publicar(regla);
    else comunicar("ajustesMotivoRequerido", true);
  });
  void cargar();
  return () => { desmontado = true; ++secuencia; solicitud?.abort(); escritura?.abort(); };
}
