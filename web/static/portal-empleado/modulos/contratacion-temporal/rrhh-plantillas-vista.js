/** Alta y edición RRHH de tipos de borrador; la publicación usa otra autorización. */
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-rpt-circular-v6";
import { MENSAJES_RRHH_PLANTILLAS_ES, MENSAJES_RRHH_PLANTILLAS_EN } from "./rrhh-plantillas-i18n.js";
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO } from "../../../comun/idioma.js";
import { crearClientePlantillasRRHH } from "./rrhh-plantillas-cliente.js?v=20261008-alta-rpt-circular-v6";

const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const e = (valor) => String(valor ?? "").replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const texto = (valor, maximo, vacio = false) => typeof valor === "string" && valor.trim() === valor
  && valor.length <= maximo && (vacio || valor.length > 0) && !/[\u0000-\u0008\u000b-\u001f\u007f]/u.test(valor);
const fechaValida = (valor) => FECHA.test(valor) && Number.isFinite(Date.parse(`${valor}T00:00:00Z`))
  && new Date(`${valor}T00:00:00Z`).toISOString().slice(0, 10) === valor;
const claveNueva = () => globalThis.crypto?.randomUUID?.();

function instantaneaOperacion(operacion, solicitud) {
  const entrada = solicitud.entrada && Object.freeze({ ...solicitud.entrada,
    atributos: Object.freeze({ ...solicitud.entrada.atributos }) });
  return Object.freeze({ operacion, clave: solicitud.clave_idempotencia,
    solicitud: Object.freeze({ ...solicitud, ...(entrada ? { entrada } : {}) }) });
}

function parrafosDe(entrada) {
  return Object.keys(entrada?.atributos ?? {}).filter((clave) => /^parrafo\.\d{2}$/u.test(clave))
    .sort().map((clave) => entrada.atributos[clave]);
}

export function prepararEntradaPlantilla(formulario, anterior = null) {
  const datos = formulario instanceof FormData ? formulario : new FormData(formulario);
  const clave = anterior?.clave ?? String(datos.get("clave") ?? "");
  const etiqueta = String(datos.get("etiqueta") ?? "");
  const descripcion = String(datos.get("descripcion") ?? "");
  const titulo = String(datos.get("titulo") ?? "");
  const ordenTexto = String(datos.get("orden") ?? "");
  const orden = Number(ordenTexto);
  const desde = String(datos.get("vigente_desde") ?? "");
  const parrafos = datos.getAll("parrafo").map(String);
  const modalidades = String(datos.get("modalidades") ?? "");
  const firmantes = String(datos.get("firmantes") ?? "");
  const accion = String(datos.get("requiere_accion") ?? "");
  const motivo = String(datos.get("motivo") ?? "");
  const fuente = String(datos.get("fuente_ref") ?? "");
  if (!CLAVE.test(clave) || clave === "etiquetas") throw new TypeError("clave_invalida");
  if (!texto(etiqueta, 256) || !texto(descripcion, 4000, true) || !texto(titulo, 256)
    || !/^\d+$/u.test(ordenTexto) || !Number.isSafeInteger(orden) || orden < 0 || !fechaValida(desde)
    || !texto(motivo, 4000) || !texto(fuente, 160)) throw new TypeError("texto_invalido");
  if (parrafos.length < 1 || parrafos.length > 64 || parrafos.some((p) => !texto(p, 65536))) {
    throw new TypeError("parrafos_invalidos");
  }
  if (!texto(modalidades, 1000) || !texto(firmantes, 2000, true)
    || !texto(accion, 96, true)) throw new TypeError("texto_invalido");
  const atributos = { ...(anterior?.atributos ?? {}), titulo, modalidades };
  for (const claveAnterior of Object.keys(atributos)) {
    if (/^parrafo\.\d{2}$/u.test(claveAnterior)) delete atributos[claveAnterior];
  }
  parrafos.forEach((parrafo, i) => { atributos[`parrafo.${String(i + 1).padStart(2, "0")}`] = parrafo; });
  if (firmantes) atributos.firmantes = firmantes;
  else delete atributos.firmantes;
  if (accion) atributos.requiere_accion = accion;
  else delete atributos.requiere_accion;
  const vigenteDesde = anterior?.vigente_desde?.startsWith(`${desde}T`)
    ? anterior.vigente_desde : `${desde}T00:00:00Z`;
  const entrada = {
    clave, etiqueta, descripcion, orden, vigente_desde: vigenteDesde,
    ...(anterior?.vigente_hasta ? { vigente_hasta: anterior.vigente_hasta } : {}), atributos,
  };
  return { entrada, motivo, fuente_ref: fuente };
}

export function montarRRHHPlantillas({
  raiz, cliente = crearClientePlantillasRRHH(), mensajes = {}, locale = "es-ES",
  anunciar = () => {}, generarClave = claveNueva,
} = {}) {
  if (!raiz || typeof cliente.consultar !== "function" || typeof cliente.guardar !== "function"
    || typeof cliente.publicar !== "function") {
    throw new TypeError("vista de plantillas RRHH no disponible");
  }
  const t = crearTraductorContratacionTemporal({
    ...(IDIOMA_ACTUAL === IDIOMA_POR_DEFECTO ? MENSAJES_RRHH_PLANTILLAS_ES : MENSAJES_RRHH_PLANTILLAS_EN),
    ...mensajes,
  });
  const controlador = new AbortController();
  let consulta = null;
  let error = null;
  let ocupado = false;
  let bloqueado = false;
  let ayuda = false;
  let seleccionado = null;
  let recibo = null;
  let claveIdempotencia = null;
  let instantaneaPendiente = null;
  let recuperando = false;
  let errorRecuperacion = null;
  const catalogo = () => consulta?.borrador ?? consulta?.publicado ?? null;
  const puedeEditar = () => consulta?.puede_editar === true;
  const puedePublicar = () => consulta?.puede_publicar === true;
  const entradas = () => [...(catalogo()?.entradas ?? [])].filter((entrada) => entrada.clave !== "etiquetas")
    .sort((a, b) => a.orden - b.orden || a.etiqueta.localeCompare(b.etiqueta, locale));
  const entradaActual = () => entradas().find((entrada) => entrada.clave === seleccionado) ?? null;
  const fecha = (valor) => {
    const instante = new Date(valor);
    return Number.isFinite(instante.getTime())
      ? new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(instante)
      : t("plantillas_rrhh_sin_valor");
  };
  const estado = (valor) => t(`plantillas_rrhh_${["borrador", "publicado", "sustituido"].includes(valor) ? valor : "sin_estado"}`);

  function confirmar(operacion, resultado) {
    if (instantaneaPendiente?.operacion !== operacion
      || resultado?.recibo?.clave_idempotencia !== instantaneaPendiente.clave
      || resultado.recibo.operacion !== operacion) {
      throw Object.assign(new TypeError("recibo de plantilla incompatible"), { resultadoIndeterminado: true });
    }
    if (operacion === "editar") {
      consulta = { ...consulta, borrador: resultado.catalogo, puede_publicar: false };
      seleccionado = null;
    } else {
      consulta = { ...consulta, borrador: null, publicado: resultado.catalogo, puede_publicar: false };
    }
    recibo = operacion === "publicar" ? { ...resultado, publicacion: true } : resultado;
    claveIdempotencia = null;
    instantaneaPendiente = null;
    bloqueado = false;
    error = null;
    errorRecuperacion = null;
    anunciar(t(operacion === "publicar" ? "plantillas_rrhh_publicacion_confirmada" : "plantillas_rrhh_guardado"));
  }

  function pintarLista() {
    const filas = entradas();
    return `<section class="panel rrhh-plantillas-panel" aria-labelledby="rrhh-plantillas-lista-titulo">
      <header class="cabecera-panel"><div><h3 id="rrhh-plantillas-lista-titulo">${e(t("plantillas_rrhh_lista_titulo"))}</h3></div>
      ${puedeEditar() ? `<button type="button" class="boton-primario" data-plantillas-accion="nueva" ${ocupado || bloqueado ? "disabled" : ""}>${e(t("plantillas_rrhh_nueva"))}</button>` : ""}</header>
      ${filas.length ? `<div class="tabla-contenedor rrhh-plantillas-tabla" tabindex="0" role="region" aria-label="${e(t("plantillas_rrhh_lista_titulo"))}">
      <table class="tabla-datos"><thead><tr><th scope="col">${e(t("plantillas_rrhh_nombre"))}</th><th scope="col">${e(t("plantillas_rrhh_titulo_documento"))}</th>
      <th scope="col">${e(t("plantillas_rrhh_orden"))}</th><th scope="col">${e(t("plantillas_rrhh_estado"))}</th><th scope="col"></th></tr></thead><tbody>
      ${filas.map((fila) => `<tr><th scope="row"><strong>${e(fila.etiqueta)}</strong></th><td>${e(fila.atributos.titulo ?? t("plantillas_rrhh_sin_valor"))}</td>
      <td class="rrhh-plantillas-numero">${e(new Intl.NumberFormat(locale).format(fila.orden))}</td><td><span class="estado-chip ${catalogo()?.estado === "publicado" ? "info" : "violeta"}">${e(estado(catalogo()?.estado))}</span></td>
      <td>${puedeEditar() ? `<button type="button" class="boton-secundario" data-plantillas-accion="editar" data-clave="${e(fila.clave)}" ${ocupado || bloqueado ? "disabled" : ""}>${e(t("plantillas_rrhh_editar"))}</button>` : ""}</td></tr>`).join("")}</tbody></table></div>`
    : `<div class="cuerpo-panel"><p class="rrhh-plantillas-vacio">${e(t("plantillas_rrhh_vacio"))}</p></div>`}</section>`;
  }

  function pintarFormulario() {
    if (seleccionado === null || !puedeEditar()) return "";
    const anterior = entradaActual();
    const parrafos = anterior ? parrafosDe(anterior) : [""];
    const fuente = catalogo()?.fuente_ref ?? "";
    const campo = (nombre, clave, valor, extra = "") => `<label><span>${e(t(clave))}</span><input name="${nombre}" value="${e(valor)}" ${extra}></label>`;
    return `<section class="panel rrhh-plantillas-panel" aria-labelledby="rrhh-plantillas-form-titulo"><header class="cabecera-panel"><div>
      <h3 id="rrhh-plantillas-form-titulo">${e(t(anterior ? "plantillas_rrhh_form_editar" : "plantillas_rrhh_form_nueva"))}</h3></div></header><div class="cuerpo-panel">
      <form data-plantillas-form novalidate><div class="rrhh-plantillas-campos">
      ${anterior ? "" : campo("clave", "plantillas_rrhh_clave", "", 'required maxlength="80" autocomplete="off"')}
      ${campo("etiqueta", "plantillas_rrhh_nombre", anterior?.etiqueta ?? "", "required maxlength=\"256\"")}
      ${campo("titulo", "plantillas_rrhh_titulo_documento", anterior?.atributos?.titulo ?? "", "required maxlength=\"256\"")}
      ${campo("orden", "plantillas_rrhh_orden", anterior?.orden ?? 0, "required type=\"number\" min=\"0\" step=\"1\"")}
      ${campo("vigente_desde", "plantillas_rrhh_vigente_desde", anterior?.vigente_desde?.slice(0, 10) ?? "", "required type=\"date\"")}
      ${campo("fuente_ref", "plantillas_rrhh_fuente", fuente, "required maxlength=\"160\" autocomplete=\"off\"")}
      <label class="rrhh-plantillas-ancho"><span>${e(t("plantillas_rrhh_descripcion"))}</span><textarea name="descripcion" rows="2" maxlength="4000">${e(anterior?.descripcion ?? "")}</textarea></label>
      ${campo("modalidades", "plantillas_rrhh_modalidades", anterior?.atributos?.modalidades ?? "*", "required maxlength=\"1000\"")}
      ${campo("firmantes", "plantillas_rrhh_firmantes", anterior?.atributos?.firmantes ?? "", "maxlength=\"2000\"")}
      ${campo("requiere_accion", "plantillas_rrhh_accion", anterior?.atributos?.requiere_accion ?? "", "maxlength=\"96\"")}
      <div class="rrhh-plantillas-parrafos rrhh-plantillas-ancho"><div class="rrhh-plantillas-subcabecera"><strong>${e(t("plantillas_rrhh_parrafos"))}</strong>
      <button type="button" class="boton-secundario" data-plantillas-accion="anadir-parrafo">${e(t("plantillas_rrhh_anadir_parrafo"))}</button></div>
      <div data-plantillas-parrafos>${parrafos.map((valor, i) => parrafoHTML(valor, i + 1)).join("")}</div></div>
      <label class="rrhh-plantillas-ancho"><span>${e(t("plantillas_rrhh_motivo"))}</span><textarea name="motivo" rows="2" maxlength="4000" required></textarea></label>
      </div><div class="rrhh-plantillas-acciones"><button type="submit" class="boton-primario" ${ocupado || bloqueado ? "disabled" : ""}>${e(t("plantillas_rrhh_guardar"))}</button>
      <button type="button" class="boton-secundario" data-plantillas-accion="cancelar" ${ocupado || bloqueado ? "disabled" : ""}>${e(t("plantillas_rrhh_cancelar"))}</button></div></form></div></section>`;
  }

  function pintarPublicacion() {
    const borrador = consulta?.borrador;
    if (!borrador || seleccionado !== null || !puedePublicar()) return "";
    return `<section class="panel rrhh-plantillas-panel rrhh-plantillas-publicacion" aria-labelledby="rrhh-plantillas-publicacion-titulo">
      <header class="cabecera-panel"><div><h3 id="rrhh-plantillas-publicacion-titulo">${e(t("plantillas_rrhh_publicacion_titulo"))}</h3></div>
      <span class="estado-chip violeta">${e(t("plantillas_rrhh_borrador"))}</span></header>
      <div class="cuerpo-panel"><form data-plantillas-publicar novalidate><div class="rrhh-plantillas-campos">
      <label><span>${e(t("plantillas_rrhh_publicacion_aprobacion"))}</span><input name="aprobacion_ref" required maxlength="160" autocomplete="off"></label>
      <label class="rrhh-plantillas-ancho"><span>${e(t("plantillas_rrhh_publicacion_motivo"))}</span><textarea name="motivo" rows="2" required maxlength="4000"></textarea></label>
      </div><div class="rrhh-plantillas-acciones"><button type="submit" class="boton-primario" ${ocupado || bloqueado ? "disabled" : ""}>${e(t("plantillas_rrhh_publicacion_enviar"))}</button></div>
      </form></div></section>`;
  }

  function parrafoHTML(valor, numero) {
    return `<label class="rrhh-plantillas-parrafo"><span>${e(t("plantillas_rrhh_parrafo_numero", { numero }))}</span>
      <textarea name="parrafo" rows="3" required>${e(valor)}</textarea><button type="button" class="boton-secundario" data-plantillas-accion="quitar-parrafo">${e(t("plantillas_rrhh_quitar_parrafo"))}</button></label>`;
  }

  function pintar() {
    if (controlador.signal.aborted) return;
    const c = catalogo();
    const mensaje = error ? t(error) : null;
    raiz.innerHTML = `<div class="rrhh-plantillas"><div class="rrhh-plantillas-encabezado"><div><h2>${e(t("plantillas_rrhh_titulo"))}</h2></div><button type="button" class="boton-secundario rrhh-plantillas-ayuda-boton" data-plantillas-accion="ayuda"
      aria-label="${e(t("plantillas_rrhh_ayuda_boton"))}" aria-expanded="${ayuda}" aria-controls="rrhh-plantillas-ayuda">?</button></div>
      <p id="rrhh-plantillas-ayuda" class="rrhh-plantillas-ayuda" ${ayuda ? "" : "hidden"}>${e(t("plantillas_rrhh_ayuda"))}</p>
      ${recibo ? `<section class="rrhh-plantillas-recibo" role="status" tabindex="-1" aria-label="${e(t("plantillas_rrhh_resultado_titulo"))}"><strong>${e(t(recibo.publicacion ? "plantillas_rrhh_publicacion_confirmada" : "plantillas_rrhh_guardado"))}</strong>
      <dl><div><dt>${e(t("plantillas_rrhh_version"))}</dt><dd>${e(String(recibo.catalogo.version))} / ${e(String(recibo.catalogo.revision))}</dd></div>
      <div><dt>${e(t("plantillas_rrhh_recibo"))}</dt><dd>${e(recibo.recibo.recibo_ref)}</dd></div>
      <div><dt>${e(t("plantillas_rrhh_fecha"))}</dt><dd>${e(fecha(recibo.recibo.registrado_en))}</dd></div></dl></section>` : ""}
      ${mensaje ? `<div id="rrhh-plantillas-estado" class="rrhh-plantillas-error" role="${bloqueado || error === "plantillas_rrhh_denegado" ? "alert" : "status"}">${e(mensaje)}</div>` : ""}
      ${errorRecuperacion ? `<div class="rrhh-plantillas-error" role="alert">${e(t(errorRecuperacion))}</div>` : ""}
      ${bloqueado && instantaneaPendiente ? `<div class="rrhh-plantillas-acciones"><button type="button" class="boton-secundario" data-plantillas-accion="recuperar" aria-describedby="rrhh-plantillas-estado" ${ocupado ? "disabled" : ""}>${e(t("plantillas_rrhh_recuperar_resultado"))}</button></div>` : ""}
      ${recuperando ? `<p role="status">${e(t("plantillas_rrhh_recuperando"))}</p>` : ""}
      ${consulta === null && bloqueado ? "" : consulta === null ? `<section class="panel rrhh-plantillas-panel"><div class="cuerpo-panel" aria-busy="${ocupado}"><p>${e(t(ocupado ? "plantillas_rrhh_cargando" : "plantillas_rrhh_error_lectura"))}</p>
      ${!ocupado ? `<button type="button" class="boton-secundario" data-plantillas-accion="recargar">${e(t("plantillas_rrhh_reintentar"))}</button>` : ""}</div></section>`
    : `<section class="panel rrhh-plantillas-panel rrhh-plantillas-meta"><header class="cabecera-panel"><div><h3>${e(t("plantillas_rrhh_version"))}</h3></div><span class="estado-chip ${c?.estado === "publicado" ? "info" : "violeta"}">${e(estado(c?.estado))}</span></header>
      <dl class="cuerpo-panel"><div><dt>${e(t("plantillas_rrhh_version"))}</dt><dd>${c ? `${e(String(c.version))} / ${e(String(c.revision))}` : "—"}</dd></div>
      <div><dt>${e(t("plantillas_rrhh_huella"))}</dt><dd class="rrhh-plantillas-huella">${e(c?.huella_sha256 ?? "—")}</dd></div></dl></section>
      ${!puedeEditar() && !puedePublicar() ? `<p class="rrhh-plantillas-lectura" role="status">${e(t("plantillas_rrhh_no_editable"))}</p>` : ""}
      ${pintarLista()}${pintarFormulario()}${pintarPublicacion()}`}</div>`;
  }

  async function cargar() {
    if (ocupado || bloqueado || controlador.signal.aborted) return;
    ocupado = true; error = null; pintar();
    try {
      consulta = await cliente.consultar({ signal: controlador.signal });
      if (!puedeEditar() || (seleccionado !== "" && !entradaActual())) seleccionado = null;
      anunciar(t("plantillas_rrhh_lista_titulo"));
    } catch (fallo) {
      if (controlador.signal.aborted) return;
      consulta = null;
      error = [401, 403].includes(fallo?.estado) ? "plantillas_rrhh_denegado" : "plantillas_rrhh_error_lectura";
    } finally { ocupado = false; pintar(); }
  }

  async function guardar(formulario) {
    if (ocupado || bloqueado || consulta === null || !puedeEditar()) return;
    let preparado;
    try { preparado = prepararEntradaPlantilla(formulario, entradaActual()); }
    catch (fallo) {
      error = `plantillas_rrhh_${["clave_invalida", "parrafos_invalidos"].includes(fallo?.message) ? fallo.message : "texto_invalido"}`;
      const anterior = formulario.querySelector("[data-plantillas-error]");
      anterior?.remove();
      formulario.insertAdjacentHTML("afterbegin", `<p class="rrhh-plantillas-error" data-plantillas-error role="alert">${e(t(error))}</p>`);
      formulario.querySelector("[data-plantillas-error]")?.focus?.();
      return;
    }
    const vigente = catalogo();
    if (!claveIdempotencia) claveIdempotencia = generarClave();
    if (!claveIdempotencia) { error = "plantillas_rrhh_error_guardar"; pintar(); return; }
    const solicitud = {
      clave_idempotencia: claveIdempotencia,
      version_esperada: vigente?.version ?? 0,
      revision_esperada: consulta.borrador?.revision ?? 0,
      motivo: preparado.motivo,
      fuente_ref: preparado.fuente_ref,
      entrada: preparado.entrada,
    };
    instantaneaPendiente = instantaneaOperacion("editar", solicitud);
    ocupado = true; error = null; recibo = null;
    raiz.querySelector(".rrhh-plantillas-recibo")?.remove();
    formulario.querySelector("[data-plantillas-error]")?.remove();
    formulario.querySelector("[data-plantillas-progreso]")?.remove();
    formulario.insertAdjacentHTML("afterbegin", `<p data-plantillas-progreso role="status">${e(t("plantillas_rrhh_guardando"))}</p>`);
    formulario.querySelectorAll("button, input, textarea").forEach((nodo) => { nodo.disabled = true; });
    let conservarFormulario = false;
    try {
      const resultado = await cliente.guardar(instantaneaPendiente.solicitud, { signal: controlador.signal });
      confirmar("editar", resultado);
    } catch (fallo) {
      if (controlador.signal.aborted) return;
      if (fallo?.resultadoIndeterminado) {
        bloqueado = true;
        error = "plantillas_rrhh_indeterminado";
      } else if (fallo?.estado === 409) {
        error = "plantillas_rrhh_conflicto";
        seleccionado = null;
        claveIdempotencia = null;
        instantaneaPendiente = null;
        try { consulta = await cliente.consultar({ signal: controlador.signal }); } catch { consulta = null; }
      } else {
        error = [401, 403].includes(fallo?.estado) ? "plantillas_rrhh_denegado" : "plantillas_rrhh_error_guardar";
        instantaneaPendiente = null;
        if ([401, 403].includes(fallo?.estado)) { consulta = null; seleccionado = null; }
        else { conservarFormulario = true; claveIdempotencia = null; }
      }
    } finally {
      ocupado = false;
      formulario.querySelector("[data-plantillas-progreso]")?.remove();
      if (conservarFormulario) {
        formulario.querySelectorAll("button, input, textarea").forEach((nodo) => { nodo.disabled = false; });
        formulario.querySelector("[data-plantillas-error]")?.remove();
        formulario.insertAdjacentHTML("afterbegin", `<p class="rrhh-plantillas-error" data-plantillas-error role="alert">${e(t(error))}</p>`);
      } else {
        pintar();
        if (bloqueado) raiz.querySelector?.('[data-plantillas-accion="recuperar"]')?.focus?.({ preventScroll: true });
        else if (recibo) raiz.querySelector?.(".rrhh-plantillas-recibo")?.focus?.({ preventScroll: true });
      }
    }
  }

  async function publicar(formulario) {
    const borrador = consulta?.borrador;
    if (!borrador || ocupado || bloqueado || seleccionado !== null || !puedePublicar()) return;
    const datos = formulario instanceof FormData ? formulario : new FormData(formulario);
    const aprobacion = String(datos.get("aprobacion_ref") ?? "");
    const motivo = String(datos.get("motivo") ?? "");
    if (!texto(aprobacion, 160) || !texto(motivo, 4000)) {
      formulario.querySelector("[data-plantillas-error]")?.remove();
      formulario.insertAdjacentHTML("afterbegin", `<p class="rrhh-plantillas-error" data-plantillas-error role="alert">${e(t("plantillas_rrhh_obligatorio"))}</p>`);
      return;
    }
    if (!claveIdempotencia) claveIdempotencia = generarClave();
    if (!claveIdempotencia) { error = "plantillas_rrhh_publicacion_error"; pintar(); return; }
    const solicitud = {
      clave_idempotencia: claveIdempotencia,
      version_esperada: borrador.version,
      revision_esperada: borrador.revision,
      motivo,
      aprobacion_ref: aprobacion,
    };
    instantaneaPendiente = instantaneaOperacion("publicar", solicitud);
    ocupado = true; error = null; recibo = null;
    raiz.querySelector(".rrhh-plantillas-recibo")?.remove();
    formulario.querySelector("[data-plantillas-error]")?.remove();
    formulario.querySelector("[data-plantillas-progreso]")?.remove();
    formulario.insertAdjacentHTML("afterbegin", `<p data-plantillas-progreso role="status">${e(t("plantillas_rrhh_publicacion_guardando"))}</p>`);
    formulario.querySelectorAll("button, input, textarea").forEach((nodo) => { nodo.disabled = true; });
    let conservarFormulario = false;
    try {
      const resultado = await cliente.publicar(instantaneaPendiente.solicitud, { signal: controlador.signal });
      confirmar("publicar", resultado);
    } catch (fallo) {
      if (controlador.signal.aborted) return;
      if (fallo?.resultadoIndeterminado) {
        bloqueado = true;
        error = "plantillas_rrhh_publicacion_indeterminada";
      } else if (fallo?.estado === 409) {
        claveIdempotencia = null;
        instantaneaPendiente = null;
        error = "plantillas_rrhh_publicacion_conflicto";
        try { consulta = await cliente.consultar({ signal: controlador.signal }); }
        catch { consulta = null; }
      } else if ([401, 403].includes(fallo?.estado)) {
        claveIdempotencia = null;
        instantaneaPendiente = null;
        error = "plantillas_rrhh_publicacion_denegada";
      } else {
        claveIdempotencia = null;
        instantaneaPendiente = null;
        error = "plantillas_rrhh_publicacion_error";
        conservarFormulario = true;
      }
    } finally {
      ocupado = false;
      formulario.querySelector("[data-plantillas-progreso]")?.remove();
      if (conservarFormulario) {
        formulario.querySelectorAll("button, input, textarea").forEach((nodo) => { nodo.disabled = false; });
        formulario.querySelector("[data-plantillas-error]")?.remove();
        formulario.insertAdjacentHTML("afterbegin", `<p class="rrhh-plantillas-error" data-plantillas-error role="alert">${e(t(error))}</p>`);
      } else {
        pintar();
        if (bloqueado) raiz.querySelector?.('[data-plantillas-accion="recuperar"]')?.focus?.({ preventScroll: true });
        else if (recibo) raiz.querySelector?.(".rrhh-plantillas-recibo")?.focus?.({ preventScroll: true });
      }
    }
  }

  async function recuperar() {
    if (!bloqueado || !instantaneaPendiente || ocupado || controlador.signal.aborted) return;
    const instantanea = instantaneaPendiente;
    ocupado = true;
    recuperando = true;
    errorRecuperacion = null;
    pintar();
    try {
      const resultado = await (instantanea.operacion === "editar" ? cliente.guardar : cliente.publicar)(
        instantanea.solicitud, { signal: controlador.signal });
      if (controlador.signal.aborted) return;
      confirmar(instantanea.operacion, resultado);
    } catch (fallo) {
      if (controlador.signal.aborted) return;
      if ([401, 403].includes(fallo?.estado)) {
        consulta = null;
        seleccionado = null;
        errorRecuperacion = "plantillas_rrhh_recuperacion_denegada";
      } else if (fallo?.estado === 409) {
        errorRecuperacion = "plantillas_rrhh_recuperacion_conflicto";
      } else {
        errorRecuperacion = "plantillas_rrhh_recuperacion_pendiente";
      }
    } finally {
      ocupado = false;
      recuperando = false;
      pintar();
      if (bloqueado) raiz.querySelector?.('[data-plantillas-accion="recuperar"]')?.focus?.({ preventScroll: true });
      else if (recibo) raiz.querySelector?.(".rrhh-plantillas-recibo")?.focus?.({ preventScroll: true });
    }
  }

  const click = (evento) => {
    const boton = evento.target.closest?.("[data-plantillas-accion]");
    if (!boton || !raiz.contains(boton)) return;
    const accion = boton.dataset.plantillasAccion;
    if (accion === "ayuda") { ayuda = !ayuda; pintar(); raiz.querySelector?.('[data-plantillas-accion="ayuda"]')?.focus?.({ preventScroll: true }); return; }
    if (accion === "recuperar") { void recuperar(); return; }
    if (ocupado || bloqueado) return;
    if (accion === "recargar") { void cargar(); return; }
    if (accion === "cancelar") { seleccionado = null; claveIdempotencia = null; error = null; pintar(); return; }
    if (["nueva", "editar"].includes(accion) && !puedeEditar()) return;
    if (accion === "nueva") { seleccionado = ""; claveIdempotencia = null; pintar(); raiz.querySelector('[name="clave"]')?.focus(); }
    if (accion === "editar") { seleccionado = boton.dataset.clave; claveIdempotencia = null; pintar(); raiz.querySelector('[name="etiqueta"]')?.focus(); }
    if (accion === "anadir-parrafo") {
      const contenedor = raiz.querySelector("[data-plantillas-parrafos]");
      if (contenedor && contenedor.children.length < 64) contenedor.insertAdjacentHTML("beforeend", parrafoHTML("", contenedor.children.length + 1));
    }
    if (accion === "quitar-parrafo") {
      const contenedor = raiz.querySelector("[data-plantillas-parrafos]");
      if (contenedor?.children.length > 1) boton.closest(".rrhh-plantillas-parrafo")?.remove();
    }
  };
  const submit = (evento) => {
    if (!evento.target.matches?.("[data-plantillas-form], [data-plantillas-publicar]")) return;
    evento.preventDefault();
    if (evento.target.matches("[data-plantillas-form]")) void guardar(evento.target);
    else void publicar(evento.target);
  };
  raiz.addEventListener("click", click);
  raiz.addEventListener("submit", submit);
  void cargar();
  return Object.freeze({
    recargar: cargar,
    desmontar() { controlador.abort(); raiz.removeEventListener("click", click); raiz.removeEventListener("submit", submit); raiz.replaceChildren(); },
  });
}
