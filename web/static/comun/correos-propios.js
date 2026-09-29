/**
 * «Mis correos» (Usuarios 5.08b), común al portal de RRHH y al Área personal.
 *
 * La persona añade direcciones, las confirma con el código de 8 números que
 * recibe en cada una, elige cuál recibe los avisos y quita las que no usa.
 * La identidad la fija el servidor por la ruta; aquí no se guarda nada en el
 * navegador. Los textos viven en `textos/<idioma>/preferencias.json` (sección
 * `correos`).
 */
import { cargarTextos } from "./textos.js";

const MAX_BYTES = 65536;
const LIMITE_MS = 25000;
const PATRON_REF = /^correo:[0-9a-f]{32}$/u;
const PATRON_CLAVE = /^[A-Za-z0-9:_.-]{16,128}$/u;
const CODIGOS_ERROR = new Set(["no_autenticado", "prohibido", "conflicto", "peticion_invalida", "no_disponible",
  "codigo_incorrecto", "codigo_caducado", "ya_registrado", "maximo", "en_uso", "limite"]);
const OPERACIONES = new Set(["anadir", "reenviar", "verificar", "activar", "retirar"]);
export const MAX_CORREOS = 5;

export class ErrorCorreos extends Error {
  constructor(estado = 0, codigo = "", intentosRestantes = null) {
    super(`correos HTTP ${estado}`);
    this.estado = estado;
    this.codigo = codigo;
    this.intentosRestantes = intentosRestantes;
  }
}

function objeto(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor); }
function fechaValida(valor) { return typeof valor === "string" && valor.length <= 40 && !Number.isNaN(Date.parse(valor)); }

/** Dirección con forma nombre@dominio, sin espacios ni nombre visible. */
export function direccionCorreoAdmisible(valor) {
  const texto = String(valor ?? "").trim();
  return texto.length >= 6 && texto.length <= 254 && /^[^\s@<>()[\]",;:\\]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}$/u.test(texto);
}

/** Admite el código con espacios o guiones de agrupación y devuelve 8 dígitos. */
export function normalizarCodigoCorreo(valor) {
  const limpio = String(valor ?? "").replace(/[\s-]/gu, "");
  return /^[0-9]{8}$/u.test(limpio) ? limpio : "";
}

function validarCorreo(correo) {
  if (!objeto(correo) || !PATRON_REF.test(correo.correo_ref) || typeof correo.direccion !== "string"
    || correo.direccion.length > 254 || !["pendiente", "verificado"].includes(correo.estado)
    || typeof correo.activo !== "boolean" || (correo.activo && correo.estado !== "verificado")
    || !fechaValida(correo.creado_utc)) throw new TypeError("correo inválido");
  let codigo = null;
  if (correo.codigo !== undefined && correo.codigo !== null) {
    if (!objeto(correo.codigo) || !fechaValida(correo.codigo.vence_utc)
      || !Number.isSafeInteger(correo.codigo.intentos_restantes) || correo.codigo.intentos_restantes < 1
      || correo.codigo.intentos_restantes > 5 || correo.estado !== "pendiente") throw new TypeError("código inválido");
    codigo = Object.freeze({ vence_utc: correo.codigo.vence_utc, intentos_restantes: correo.codigo.intentos_restantes });
  }
  return Object.freeze({ correo_ref: correo.correo_ref, direccion: correo.direccion, estado: correo.estado,
    activo: correo.activo, creado_utc: correo.creado_utc, codigo });
}

async function contenidoJSON(respuesta) {
  const tipo = respuesta.headers?.get?.("content-type") || "";
  if (!tipo.toLowerCase().includes("application/json")) throw new TypeError("respuesta de correos no JSON");
  if (Number(respuesta.headers?.get?.("content-length") || 0) > MAX_BYTES) throw new TypeError("respuesta de correos demasiado grande");
  const texto = await respuesta.text();
  if (texto.length > MAX_BYTES) throw new TypeError("respuesta de correos demasiado grande");
  return JSON.parse(texto);
}

/** Cliente HTTP de la ruta exacta de cada portal. */
export function crearClienteCorreos({ ruta, fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function" || !["/api/vec/usuarios/mis-correos", "/api/vec/usuarios/area-personal/mis-correos"].includes(ruta)) {
    throw new TypeError("cliente de correos no disponible");
  }
  async function solicitar(metodo, cuerpo, signal) {
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    if (signal?.aborted) abortar();
    else signal?.addEventListener?.("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, LIMITE_MS);
    try {
      const respuesta = await fetchImpl(ruta, {
        method: metodo, credentials: "same-origin", mode: "same-origin", redirect: "error",
        cache: "no-store", referrerPolicy: "no-referrer", signal: controlador.signal,
        headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
        ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
      });
      if (!respuesta.ok) {
        let codigo = "";
        let intentos = null;
        try {
          const error = (await contenidoJSON(respuesta))?.error;
          if (CODIGOS_ERROR.has(error?.codigo) && error.clave_i18n === `api.usuarios.correos.error.${error.codigo}`) codigo = error.codigo;
          if (Number.isSafeInteger(error?.intentos_restantes) && error.intentos_restantes >= 0 && error.intentos_restantes < 5) intentos = error.intentos_restantes;
        } catch { /* El estado HTTP basta para responder con seguridad. */ }
        throw new ErrorCorreos(respuesta.status, codigo, intentos);
      }
      return await contenidoJSON(respuesta);
    } catch (error) {
      if (signal?.aborted) throw error;
      if (error instanceof ErrorCorreos) throw error;
      throw new ErrorCorreos(0);
    } finally {
      clearTimeout(temporizador);
      signal?.removeEventListener?.("abort", abortar);
    }
  }
  return Object.freeze({
    async consultar({ signal } = {}) {
      const datos = (await solicitar("GET", null, signal))?.data;
      if (!objeto(datos) || !Number.isSafeInteger(datos.version) || datos.version < 0 || !Array.isArray(datos.correos)
        || datos.correos.length > MAX_CORREOS) throw new TypeError("lista de correos inválida");
      const correos = datos.correos.map(validarCorreo);
      if (correos.filter((correo) => correo.activo).length > 1) throw new TypeError("lista de correos inválida");
      return Object.freeze({ version: datos.version, correos: Object.freeze(correos) });
    },
    async operar(cuerpo, { signal } = {}) {
      if (!objeto(cuerpo) || !OPERACIONES.has(cuerpo.operacion) || !Number.isSafeInteger(cuerpo.version_esperada)
        || cuerpo.version_esperada < 0 || !PATRON_CLAVE.test(cuerpo.clave_operacion)) throw new TypeError("operación de correos inválida");
      const recibo = (await solicitar("POST", cuerpo, signal))?.data;
      if (!objeto(recibo) || typeof recibo.recibo_ref !== "string" || recibo.recibo_ref.length === 0
        || !PATRON_REF.test(recibo.correo_ref) || !Number.isSafeInteger(recibo.version) || recibo.version < 1
        || !fechaValida(recibo.fecha_utc)) throw new TypeError("recibo de correos inválido");
      return Object.freeze({ recibo_ref: recibo.recibo_ref, correo_ref: recibo.correo_ref, version: recibo.version,
        replay: recibo.replay === true, envio: ["aceptado", "no_enviado"].includes(recibo.envio) ? recibo.envio : "" });
    },
  });
}

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

const MARCO_PREDETERMINADO = Object.freeze({ panel: "panel", cabecera: "div", claseCabecera: "cabecera-panel", cuerpo: "cuerpo-panel" });

/**
 * Superficie de «Mis correos». `marco` adapta las clases del panel a cada
 * portal; el resto del marcado y el comportamiento son comunes.
 */
export function crearSuperficieCorreos({ cliente, textos, marco = MARCO_PREDETERMINADO, aleatorio = globalThis.crypto, zonaHoraria = "Europe/Madrid", cargaAlMostrar = false } = {}) {
  if (!cliente || typeof textos?.traducir !== "function") throw new TypeError("superficie de correos incompleta");
  const t = (clave, variables) => textos.traducir(`correos.${clave}`, variables);
  let datos = null;
  let carga = "sin_cargar";
  let errorCarga = null;
  let mensaje = null;
  let ocupado = null;
  let pendiente = null;
  let confirmarQuitar = "";
  let ayudaVisible = false;
  let campoConError = "";
  let enfocarTras = "";
  const borradores = { direccion: "", codigos: new Map() };
  let controlador = null;
  let generacion = 0;
  let contenedor = null;

  const raiz = () => contenedor?.querySelector?.("[data-correos-raiz]") ?? null;
  function repintar() {
    const elemento = raiz();
    if (!elemento) return;
    elemento.innerHTML = interior();
    const destino = enfocarTras ? elemento.querySelector(enfocarTras) : null;
    enfocarTras = "";
    destino?.focus?.({ preventScroll: false });
  }

  function hora(valor) {
    return textos.fecha(valor, { hour: "2-digit", minute: "2-digit", timeZone: zonaHoraria });
  }
  function direccionDe(ref) { return datos?.correos.find((correo) => correo.correo_ref === ref)?.direccion ?? ""; }

  async function cargar({ conservarMensaje = false } = {}) {
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++generacion;
    carga = "cargando";
    errorCarga = null;
    if (!conservarMensaje) mensaje = null;
    repintar();
    try {
      const nuevos = await cliente.consultar({ signal: controlador.signal });
      if (actual !== generacion) return;
      datos = nuevos;
      carga = "lista";
      for (const ref of borradores.codigos.keys()) if (!nuevos.correos.some((c) => c.correo_ref === ref && c.estado === "pendiente")) borradores.codigos.delete(ref);
    } catch (error) {
      if (actual !== generacion || controlador.signal.aborted) return;
      datos = null;
      carga = "error";
      errorCarga = error;
    }
    repintar();
  }

  function textoError(error, operacion) {
    switch (error?.codigo) {
      case "no_autenticado": return t("error_sesion");
      case "prohibido": return t("error_denegado");
      case "codigo_incorrecto":
        return Number.isSafeInteger(error.intentosRestantes)
          ? t("error_codigo_incorrecto", { cuenta: error.intentosRestantes }) : t("error_codigo_incorrecto_sin_cuenta");
      case "codigo_caducado": return t("error_codigo_caducado");
      case "ya_registrado": return t("error_ya_registrado");
      case "maximo": return t("error_maximo", { maximo: MAX_CORREOS });
      case "en_uso": return t("error_en_uso");
      case "limite": return t("error_limite");
      case "conflicto": return t("error_conflicto");
      case "peticion_invalida": return operacion === "anadir" ? t("error_direccion") : operacion === "verificar" ? t("error_codigo_formato") : t("error_peticion");
      default: return t("error_incierto");
    }
  }

  function textoExito(cuerpo, recibo) {
    const direccion = cuerpo.operacion === "anadir" ? cuerpo.direccion.trim() : direccionDe(cuerpo.correo_ref);
    switch (cuerpo.operacion) {
      case "anadir":
        if (recibo.replay) return t("exito_anadir_repetido", { direccion });
        return recibo.envio === "aceptado" ? t("exito_anadir", { direccion }) : t("exito_anadir_sin_envio", { direccion });
      case "reenviar":
        if (recibo.replay) return t("exito_reenviar_repetido", { direccion });
        return recibo.envio === "aceptado" ? t("exito_reenviar", { direccion }) : t("exito_reenviar_sin_envio");
      case "verificar": return t("exito_verificar", { direccion });
      case "activar": return t("exito_activar", { direccion });
      default: return t("exito_retirar", { direccion });
    }
  }

  async function enviar(cuerpo) {
    if (ocupado) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++generacion;
    ocupado = cuerpo;
    pendiente = null;
    campoConError = "";
    mensaje = { tipo: "estado", texto: t(cuerpo.operacion === "anadir" || cuerpo.operacion === "reenviar" ? "enviando_codigo" : "guardando") };
    repintar();
    try {
      const recibo = await cliente.operar(cuerpo, { signal: controlador.signal });
      if (actual !== generacion) return;
      ocupado = null;
      confirmarQuitar = "";
      if (cuerpo.operacion === "anadir") borradores.direccion = "";
      if (cuerpo.operacion === "verificar") borradores.codigos.delete(cuerpo.correo_ref);
      let texto = textoExito(cuerpo, recibo);
      await cargar({ conservarMensaje: true });
      if (cuerpo.operacion === "verificar") {
        const correo = datos?.correos.find((c) => c.correo_ref === recibo.correo_ref);
        texto = correo?.activo ? t("exito_verificar_activo", { direccion: correo.direccion }) : texto;
      }
      const sinEnvio = recibo.envio === "no_enviado" && (cuerpo.operacion === "anadir" || cuerpo.operacion === "reenviar");
      mensaje = { tipo: sinEnvio ? "aviso" : "exito", texto };
      enfocarTras = !sinEnvio && ["anadir", "reenviar"].includes(cuerpo.operacion) && datos?.correos.some((c) => c.correo_ref === recibo.correo_ref && c.estado === "pendiente")
        ? `#correos-codigo-${recibo.correo_ref.slice(7)}` : "[data-correos-mensaje]";
    } catch (error) {
      if (actual !== generacion || controlador.signal.aborted) return;
      ocupado = null;
      if (!error?.codigo || error.codigo === "no_disponible") pendiente = cuerpo;
      mensaje = { tipo: "error", texto: textoError(error, cuerpo.operacion) };
      if (error?.codigo === "peticion_invalida" && cuerpo.operacion === "anadir") campoConError = "direccion";
      if (["codigo_incorrecto", "peticion_invalida"].includes(error?.codigo) && cuerpo.operacion === "verificar") {
        campoConError = `codigo-${cuerpo.correo_ref}`;
        borradores.codigos.set(cuerpo.correo_ref, "");
      }
      if (["conflicto", "codigo_incorrecto", "codigo_caducado", "en_uso", "ya_registrado", "maximo"].includes(error?.codigo)) {
        await cargar({ conservarMensaje: true });
      }
      enfocarTras = "[data-correos-mensaje]";
    }
    repintar();
  }

  function operar(operacion, extra) {
    if (!datos || ocupado) return;
    const uuid = aleatorio?.randomUUID?.();
    if (!uuid) {
      mensaje = { tipo: "error", texto: t("error_incierto") };
      enfocarTras = "[data-correos-mensaje]";
      repintar();
      return;
    }
    void enviar({ operacion, version_esperada: datos.version, clave_operacion: `web-correo-${uuid}`, ...extra });
  }

  function botonAccion(accion, ref, texto, clase = "boton-secundario", etiqueta = "") {
    const deshabilitado = ocupado ? " disabled" : "";
    const aria = etiqueta ? ` aria-label="${escapar(etiqueta)}"` : "";
    return `<button type="button" class="${clase}" data-correos-accion="${accion}" data-correos-ref="${escapar(ref)}"${aria}${deshabilitado}>${escapar(texto)}</button>`;
  }

  function renderizarCorreo(correo) {
    const id = correo.correo_ref.slice(7);
    const direccion = escapar(correo.direccion);
    const estado = correo.activo
      ? `<span class="correos-estado correos-estado--activo">${escapar(t("estado_activo"))}</span>`
      : correo.estado === "verificado"
        ? `<span class="correos-estado correos-estado--confirmado">${escapar(t("estado_confirmado"))}</span>`
        : `<span class="correos-estado correos-estado--pendiente">${escapar(t("estado_pendiente"))}</span>`;
    let cuerpo = "";
    if (confirmarQuitar === correo.correo_ref) {
      cuerpo = `<div class="correos-confirmar" role="group" aria-labelledby="correos-quitar-${id}"><p id="correos-quitar-${id}">${escapar(t("quitar_pregunta", { direccion: correo.direccion }))}</p><div class="correos-acciones">${botonAccion("retirar", correo.correo_ref, t("quitar_si"), "boton-peligro")}${botonAccion("cancelar-quitar", correo.correo_ref, t("cancelar"))}</div></div>`;
    } else if (correo.estado === "pendiente") {
      const errorCodigo = campoConError === `codigo-${correo.correo_ref}`;
      const nota = correo.codigo
        ? t(correo.codigo.intentos_restantes < 5 ? "codigo_vence_intentos" : "codigo_vence", { hora: hora(correo.codigo.vence_utc), cuenta: correo.codigo.intentos_restantes })
        : t("codigo_sin_vigencia");
      cuerpo = `<form class="correos-verificar" data-correos-verificar="${escapar(correo.correo_ref)}" novalidate><div class="correos-campo"><label for="correos-codigo-${id}">${escapar(t("codigo_etiqueta"))}</label><input id="correos-codigo-${id}" name="codigo" data-correos-ref="${escapar(correo.correo_ref)}" inputmode="numeric" autocomplete="one-time-code" maxlength="11" size="11" spellcheck="false" value="${escapar(borradores.codigos.get(correo.correo_ref) ?? "")}" aria-describedby="correos-nota-${id}"${errorCodigo ? ' aria-invalid="true"' : ""}${ocupado ? " disabled" : ""}></div><button type="submit" class="boton-primario"${ocupado ? " disabled" : ""}>${escapar(t("confirmar"))}</button></form><p id="correos-nota-${id}" class="correos-nota">${escapar(nota)}</p><div class="correos-acciones">${botonAccion("reenviar", correo.correo_ref, t("reenviar"), "boton-secundario", t("reenviar_a", { direccion: correo.direccion }))}${botonAccion("pedir-quitar", correo.correo_ref, t("quitar"), "boton-secundario", t("quitar_a", { direccion: correo.direccion }))}</div>`;
    } else if (!correo.activo) {
      cuerpo = `<div class="correos-acciones">${botonAccion("activar", correo.correo_ref, t("usar_avisos"), "boton-secundario", t("usar_avisos_a", { direccion: correo.direccion }))}${botonAccion("pedir-quitar", correo.correo_ref, t("quitar"), "boton-secundario", t("quitar_a", { direccion: correo.direccion }))}</div>`;
    }
    return `<li class="correos-item"><div class="correos-cabecera-item"><span class="correos-direccion">${direccion}</span>${estado}</div>${cuerpo}</li>`;
  }

  function renderizarMensaje() {
    if (!mensaje) return `<p class="correos-mensaje" data-correos-mensaje tabindex="-1" role="status" hidden></p>`;
    const rol = mensaje.tipo === "error" ? "alert" : "status";
    const reintentar = pendiente ? ` <button type="button" class="boton-secundario" data-correos-accion="reintentar"${ocupado ? " disabled" : ""}>${escapar(t("reintentar"))}</button>` : "";
    return `<p class="correos-mensaje correos-mensaje--${mensaje.tipo}" data-correos-mensaje tabindex="-1" role="${rol}">${escapar(mensaje.texto)}${reintentar}</p>`;
  }

  function renderizarAlta() {
    if (!datos || datos.correos.length >= MAX_CORREOS) return datos ? `<p class="correos-nota">${escapar(t("maximo_alcanzado", { maximo: MAX_CORREOS }))}</p>` : "";
    const error = campoConError === "direccion";
    const etiqueta = datos.correos.length === 0 ? t("anadir_primera") : t("anadir_otra");
    return `<form class="correos-alta" data-correos-anadir novalidate><div class="correos-campo"><label for="correos-nueva">${escapar(etiqueta)}</label><input id="correos-nueva" name="direccion" type="email" autocomplete="email" inputmode="email" maxlength="254" spellcheck="false" value="${escapar(borradores.direccion)}"${error ? ' aria-invalid="true" aria-describedby="correos-nueva-error"' : ""}${ocupado ? " disabled" : ""}>${error ? `<p id="correos-nueva-error" class="correos-error-campo">${escapar(t("error_direccion"))}</p>` : ""}</div><button type="submit" class="boton-secundario"${ocupado ? " disabled" : ""}>${escapar(t("enviar_codigo"))}</button></form>`;
  }

  function interior() {
    const Cabecera = marco.cabecera === "header" ? "header" : "div";
    const ayuda = `<button type="button" class="correos-ayuda-boton" data-correos-accion="ayuda" aria-expanded="${ayudaVisible}" aria-controls="correos-ayuda" aria-label="${escapar(t("ayuda_abrir"))}">?</button>`;
    const cabecera = `<${Cabecera} class="${escapar(marco.claseCabecera)} correos-cabecera"><h2 id="correos-titulo">${escapar(t("titulo"))}</h2>${ayuda}</${Cabecera}>`;
    const textoAyuda = `<div id="correos-ayuda" class="correos-ayuda"${ayudaVisible ? "" : " hidden"}><p>${escapar(t("ayuda"))}</p></div>`;
    let contenido;
    if (carga === "sin_cargar" || (carga === "cargando" && !datos)) {
      contenido = `<p class="correos-nota" role="status" aria-busy="true">${escapar(t("cargando"))}</p>`;
    } else if (!datos) {
      contenido = `<p class="correos-mensaje correos-mensaje--error" role="alert">${escapar(errorCarga?.codigo === "no_autenticado" ? t("error_sesion") : errorCarga?.codigo === "prohibido" ? t("error_denegado") : t("error_carga"))}</p><button type="button" class="boton-secundario" data-correos-accion="recargar">${escapar(t("reintentar"))}</button>`;
    } else {
      const lista = datos.correos.length === 0
        ? `<p class="correos-vacio">${escapar(t("vacio"))}</p>`
        : `<ul class="correos-lista" aria-labelledby="correos-titulo">${datos.correos.map(renderizarCorreo).join("")}</ul>`;
      contenido = `${renderizarMensaje()}${lista}${renderizarAlta()}`;
    }
    return `${cabecera}<div class="${escapar(marco.cuerpo)} correos-cuerpo">${textoAyuda}${contenido}</div>`;
  }

  /**
   * Con `cargaAlMostrar`, la primera vez que se muestra en un contenedor
   * instalado consulta la lista. Sin él, quien la integra decide cuándo: la
   * identidad de desarrollo no admite dos altas de sesión simultáneas de la
   * misma cuenta por rutas distintas, así que el portal la pide después de
   * sus preferencias y nunca a la vez.
   */
  function renderizar() {
    if (cargaAlMostrar && carga === "sin_cargar" && contenedor) queueMicrotask(() => { if (carga === "sin_cargar") void cargar(); });
    return `<section class="${escapar(marco.panel)} correos-panel" data-correos-raiz aria-labelledby="correos-titulo"${ocupado ? ' aria-busy="true"' : ""}>${interior()}</section>`;
  }

  function alHacerClic(evento) {
    const boton = evento.target?.closest?.("[data-correos-accion]");
    if (!boton || !raiz()?.contains(boton)) return;
    const ref = boton.dataset.correosRef ?? "";
    switch (boton.dataset.correosAccion) {
      case "ayuda": ayudaVisible = !ayudaVisible; enfocarTras = ""; repintar(); break;
      case "recargar": void cargar(); break;
      case "reintentar": if (pendiente) void enviar(pendiente); break;
      case "reenviar": if (PATRON_REF.test(ref)) operar("reenviar", { correo_ref: ref }); break;
      case "activar": if (PATRON_REF.test(ref)) operar("activar", { correo_ref: ref }); break;
      case "pedir-quitar":
        if (PATRON_REF.test(ref)) { confirmarQuitar = ref; mensaje = null; enfocarTras = `#correos-quitar-${ref.slice(7)} + .correos-acciones button:last-child`; repintar(); }
        break;
      case "cancelar-quitar": confirmarQuitar = ""; enfocarTras = ""; repintar(); break;
      case "retirar": if (PATRON_REF.test(ref) && confirmarQuitar === ref) operar("retirar", { correo_ref: ref }); break;
      default: break;
    }
  }

  function alEnviar(evento) {
    const formulario = evento.target;
    if (!raiz()?.contains(formulario)) return;
    if (formulario.matches?.("[data-correos-anadir]")) {
      evento.preventDefault();
      const direccion = String(new FormData(formulario).get("direccion") ?? "").trim();
      borradores.direccion = direccion;
      if (!direccionCorreoAdmisible(direccion)) {
        campoConError = "direccion";
        mensaje = { tipo: "error", texto: t("error_direccion") };
        enfocarTras = "#correos-nueva";
        repintar();
        return;
      }
      operar("anadir", { direccion });
    } else if (formulario.matches?.("[data-correos-verificar]")) {
      evento.preventDefault();
      const ref = formulario.dataset.correosVerificar;
      const escrito = String(new FormData(formulario).get("codigo") ?? "");
      borradores.codigos.set(ref, escrito);
      const codigo = normalizarCodigoCorreo(escrito);
      if (!PATRON_REF.test(ref)) return;
      if (!codigo) {
        campoConError = `codigo-${ref}`;
        mensaje = { tipo: "error", texto: t("error_codigo_formato") };
        enfocarTras = `#correos-codigo-${ref.slice(7)}`;
        repintar();
        return;
      }
      operar("verificar", { correo_ref: ref, codigo });
    }
  }

  function alEscribir(evento) {
    const campo = evento.target;
    if (!raiz()?.contains(campo)) return;
    if (campo.name === "direccion") borradores.direccion = campo.value;
    else if (campo.name === "codigo" && PATRON_REF.test(campo.dataset?.correosRef ?? "")) borradores.codigos.set(campo.dataset.correosRef, campo.value);
  }

  function instalar(nuevo) {
    contenedor = nuevo;
    nuevo.addEventListener("click", alHacerClic);
    nuevo.addEventListener("submit", alEnviar);
    nuevo.addEventListener("input", alEscribir);
    return () => {
      nuevo.removeEventListener("click", alHacerClic);
      nuevo.removeEventListener("submit", alEnviar);
      nuevo.removeEventListener("input", alEscribir);
      desmontarPeticion();
      contenedor = null;
    };
  }
  function desmontarPeticion() { controlador?.abort(); ++generacion; ocupado = null; }

  return Object.freeze({ cargar, renderizar, instalar, desmontarPeticion, leer: () => datos, leerCarga: () => carga });
}

/** Textos de «Mis correos» en el idioma actual (con respaldo del idioma por defecto). */
export function cargarTextosCorreos(opciones) {
  return cargarTextos("preferencias", opciones);
}
