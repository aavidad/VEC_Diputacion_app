/**
 * Pantalla RRHH para declarar la publicación oficial de las bases de una
 * convocatoria de Bolsa. La pantalla solo recoge los datos del anuncio: la
 * versión vigente, quién declara y el motivo los comprueba o fija el servidor.
 */
import { cargarTextos, reintentarTextos } from "../../../comun/textos.js";
import { localMadrid } from "../../hora-madrid.js";
import { crearClientePublicacionBases, erroresCuerpoPublicacion, MEDIO_SIN_BOLETIN, MEDIOS_PUBLICACION,
  nuevaClavePublicacion } from "./rrhh-publicacion-bases-api.js?v=VEC_WEB_VERSION";

const CATALOGO = "bolsa-publicacion-bases";
const ZONA = "Europe/Madrid";
const CAMPOS = Object.freeze(["medio", "boletin_numero", "anuncio_ref", "fecha_publicacion", "direccion"]);
const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
const camposVacios = () => ({ medio: "", boletin_numero: "", anuncio_ref: "", fecha_publicacion: "", direccion: "" });

/** Fecha de hoy (AAAA-MM-DD) en hora civil de Madrid: tope de la fecha de publicación. */
export function hoyMadrid(ms = Date.now()) { return localMadrid(ms).slice(0, 10); }

/** Traduce los campos del formulario al cuerpo exacto del contrato. */
export function cuerpoDeFormulario(item, campos) {
  const sinBoletin = campos.medio === MEDIO_SIN_BOLETIN;
  const numero = String(campos.boletin_numero ?? "").trim();
  const direccion = String(campos.direccion ?? "").trim();
  return {
    convocatoria_ref: item.convocatoria_ref,
    huella_version_sha256: item.huella_version_sha256,
    medio: campos.medio,
    boletin_numero: sinBoletin ? null : /^[0-9]{1,5}$/u.test(numero) ? Number(numero) : NaN,
    anuncio_ref: String(campos.anuncio_ref ?? "").trim(),
    fecha_publicacion: String(campos.fecha_publicacion ?? ""),
    direccion: direccion === "" ? null : direccion,
    sustituye_ref: item.publicacion?.publicacion_ref ?? null,
  };
}

export async function montarPublicacionBasesRRHH({ raiz, cliente = crearClientePublicacionBases(),
  cargarCatalogo = cargarTextos, reintentarCatalogo = reintentarTextos, anunciar = () => {},
  alDenegacion = () => {}, alDisponibilidad = () => {}, ahora = () => Date.now(),
  nuevaClave = nuevaClavePublicacion, signal } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || typeof raiz?.replaceChildren !== "function"
    || typeof cliente?.listar !== "function" || typeof cliente?.declarar !== "function"
    || typeof anunciar !== "function" || typeof alDenegacion !== "function" || typeof alDisponibilidad !== "function") {
    throw new TypeError("vista de publicación de bases no disponible");
  }
  let catalogo;
  try { catalogo = await cargarCatalogo(CATALOGO); }
  catch { catalogo = await reintentarCatalogo(CATALOGO); }
  if (signal?.aborted) return Object.freeze({ desmontar() {} });

  const t = (clave, variables = {}) => catalogo.traducir(`publicacion.${clave}`, variables);
  const et = (clave, variables) => esc(t(clave, variables));
  const fechaHora = (valor) => catalogo.fecha(valor, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA });
  const fechaCivil = (valor) => catalogo.fecha(`${valor}T00:00:00Z`, { dateStyle: "long", timeZone: "UTC" });

  let vivo = true;
  let secuencia = 0;
  let controlador = null;
  let estado = "cargando"; // cargando | lista | error | denegada | no_disponible
  let lista = null;
  let ayuda = false;
  let aviso = "";           // aviso general sobre la lista (conflictos)
  let resguardo = null;
  let abierta = null;       // elemento de la lista cuyo formulario está abierto
  let campos = camposVacios();
  let errores = [];
  let avisoEnvio = "";
  let enviando = false;
  let intento = null;       // { clave, firma }: se reutiliza al reenviar el mismo cuerpo
  let enfocar = "";

  const chip = (item) => item.publicacion
    ? `<span class="estado-chip exito">${et("declarada")}</span>` : `<span class="estado-chip aviso">${et("pendiente")}</span>`;
  function resumenPublicacion(p) {
    const medio = t(`medio_${p.medio}`);
    const fecha = fechaCivil(p.fecha_publicacion);
    const linea = p.boletin_numero === null ? t("sin_boletin", { medio, fecha })
      : t("con_boletin", { medio, boletin: catalogo.numero(p.boletin_numero, { useGrouping: false }), fecha });
    return `<small>${esc(linea)}</small><small>${et("anuncio", { anuncio: p.anuncio_ref })}</small>${p.direccion
      ? `<a href="${esc(p.direccion)}" target="_blank" rel="noopener noreferrer">${et("ver_anuncio")}</a>` : ""}`;
  }
  function filas() {
    return lista.items.map((item) => {
      const plazo = item.plazo_inscripcion_abre_en
        ? `<time datetime="${esc(item.plazo_inscripcion_abre_en)}">${esc(fechaHora(item.plazo_inscripcion_abre_en))}</time>` : et("sin_fecha_plazo");
      const accion = item.publicacion ? "corregir" : "declarar";
      return `<tr${abierta?.convocatoria_ref === item.convocatoria_ref ? ' aria-current="true"' : ""}><th scope="row"><strong>${esc(item.titulo)}</strong><small>${esc(item.identificador_publico)}</small></th>
        <td>${et("version", { secuencia: catalogo.numero(item.secuencia) })}</td><td>${plazo}</td>
        <td><div class="publicacion-bases-estado">${chip(item)}${item.publicacion ? resumenPublicacion(item.publicacion) : ""}</div></td>
        <td><button type="button" class="${item.publicacion ? "boton-secundario" : "boton-primario"}" data-pb-abrir="${esc(item.convocatoria_ref)}" aria-label="${et(`${accion}_etiqueta`, { titulo: item.titulo })}">${et(accion)}</button></td></tr>`;
    }).join("");
  }
  function htmlResguardo() {
    if (!resguardo) return "";
    return `<section class="publicacion-bases-resguardo" aria-labelledby="pb-resguardo-titulo">
      <h3 id="pb-resguardo-titulo" tabindex="-1">${et("resguardo_titulo")}</h3>
      ${resguardo.reutilizada ? `<p>${et("resguardo_repetida")}</p>` : ""}
      <dl><div><dt>${et("resguardo_convocatoria")}</dt><dd>${esc(resguardo.titulo)}</dd></div>
      <div><dt>${et("resguardo_recibo")}</dt><dd><code>${esc(resguardo.recibo_ref)}</code></dd></div>
      <div><dt>${et("resguardo_fecha")}</dt><dd><time datetime="${esc(resguardo.registrada_en)}">${esc(fechaHora(resguardo.registrada_en))}</time></dd></div></dl>
      <button type="button" class="boton-secundario" data-pb-cerrar-resguardo>${et("cerrar_resguardo")}</button></section>`;
  }
  function campo(nombre, control, { opcional = false, oculto = false } = {}) {
    const error = errores.includes(nombre);
    return `<div class="publicacion-bases-campo"${oculto ? " hidden" : ""}><label for="pb-${nombre}">${et(`campo_${nombre === "medio" ? "medio" : nombre === "boletin_numero" ? "boletin" : nombre === "anuncio_ref" ? "anuncio" : nombre === "fecha_publicacion" ? "fecha" : "direccion"}`)}${opcional ? ` <span class="publicacion-bases-opcional">${et("opcional")}</span>` : ""}</label>
      ${control(error ? ` aria-invalid="true" aria-describedby="pb-error-${nombre}"` : "")}
      <span class="error-campo" id="pb-error-${nombre}"${error ? "" : " hidden"}>${error ? et(`error_${nombre}`) : ""}</span></div>`;
  }
  function htmlFormulario() {
    if (!abierta) return "";
    const sinBoletin = campos.medio === MEDIO_SIN_BOLETIN;
    const corregir = !!abierta.publicacion;
    const resumen = errores.length ? `<div class="publicacion-bases-errores" role="alert" id="pb-resumen" tabindex="-1"><p>${et("resumen_errores")}</p><ul>${errores
      .map((nombre) => `<li><a href="#pb-${nombre}" data-pb-ir="${nombre}">${et(`error_${nombre}`)}</a></li>`).join("")}</ul></div>` : "";
    const opciones = MEDIOS_PUBLICACION.map((medio) => `<option value="${medio}"${campos.medio === medio ? " selected" : ""}>${et(`medio_${medio}`)}</option>`).join("");
    const deshabilitar = enviando ? " disabled" : "";
    return `<section class="panel publicacion-bases-formulario" aria-labelledby="pb-form-titulo">
      <div class="cabecera-panel"><div><h3 id="pb-form-titulo" tabindex="-1">${et(corregir ? "formulario_titulo_corregir" : "formulario_titulo")}</h3>
      <p>${et("formulario_convocatoria", { titulo: abierta.titulo, secuencia: catalogo.numero(abierta.secuencia) })}</p></div></div>
      <form class="cuerpo-panel" data-pb-formulario novalidate${enviando ? ' aria-busy="true"' : ""}>
      ${resumen}${avisoEnvio ? `<p class="publicacion-bases-aviso" role="alert" id="pb-aviso-envio" tabindex="-1">${et(avisoEnvio)}</p>` : ""}
      ${corregir ? `<p>${et("formulario_corrige", { fecha: fechaHora(abierta.publicacion.registrada_en) })}</p>` : ""}
      <div class="publicacion-bases-campos">
      ${campo("medio", (a) => `<select id="pb-medio" name="medio" required${a}${deshabilitar}><option value=""${campos.medio ? "" : " selected"}>${et("elegir_medio")}</option>${opciones}</select>`)}
      ${campo("boletin_numero", (a) => `<input id="pb-boletin_numero" name="boletin_numero" type="text" inputmode="numeric" maxlength="5" autocomplete="off" value="${esc(campos.boletin_numero)}"${sinBoletin ? " disabled" : " required"}${a}${deshabilitar}>`, { oculto: sinBoletin })}
      ${campo("anuncio_ref", (a) => `<input id="pb-anuncio_ref" name="anuncio_ref" type="text" maxlength="80" autocomplete="off" spellcheck="false" required value="${esc(campos.anuncio_ref)}"${a}${deshabilitar}>`)}
      ${campo("fecha_publicacion", (a) => `<input id="pb-fecha_publicacion" name="fecha_publicacion" type="date" required max="${hoyMadrid(ahora())}" value="${esc(campos.fecha_publicacion)}"${a}${deshabilitar}>`)}
      ${campo("direccion", (a) => `<input id="pb-direccion" name="direccion" type="url" inputmode="url" maxlength="512" autocomplete="off" spellcheck="false" value="${esc(campos.direccion)}"${a}${deshabilitar}>`, { opcional: true })}
      </div>
      <div class="acciones-vista"><button type="button" class="boton-secundario" data-pb-cancelar${deshabilitar}>${et("cancelar")}</button>
      <button type="submit" class="boton-primario"${deshabilitar}>${et(enviando ? "registrando" : "registrar")}</button></div>
      </form></section>`;
  }
  function htmlCuerpo() {
    if (estado === "cargando" && !lista) return `<p role="status" aria-busy="true">${et("cargando")}</p>`;
    if (estado === "denegada") return `<p role="alert">${et("denegada")}</p>`;
    if (estado === "error") return `<p role="alert">${et("error")}</p><button type="button" class="boton-secundario" data-pb-actualizar>${et("reintentar")}</button>`;
    const avisoLista = aviso ? `<p class="publicacion-bases-aviso" role="alert">${et(aviso)}</p>` : "";
    if (lista.items.length === 0) return `${avisoLista}<p>${et("vacio")}</p>`;
    return `${avisoLista}${lista.completa ? "" : `<p role="status">${et("incompleta")}</p>`}
      <div class="tabla-contenedor publicacion-bases-tabla" role="region" tabindex="0" aria-label="${et("lista")}">
      <table class="tabla-datos"><thead><tr><th scope="col">${et("col_convocatoria")}</th><th scope="col">${et("col_version")}</th>
      <th scope="col">${et("col_plazo")}</th><th scope="col">${et("col_publicacion")}</th><th scope="col">${et("col_accion")}</th></tr></thead>
      <tbody>${filas()}</tbody></table></div>`;
  }
  function pintar() {
    if (!vivo) return;
    // Repintar sustituye el DOM: se devuelve el foco al mismo control si lo tenía.
    const activo = globalThis.document?.activeElement;
    const idActivo = activo && raiz.contains?.(activo) && typeof activo.id === "string" ? activo.id : "";
    if (estado === "no_disponible") {
      raiz.innerHTML = `<section class="panel publicacion-bases" data-publicacion-bases aria-labelledby="pb-no-disponible"><div class="cuerpo-panel vacio-controlado">
        <h2 id="pb-no-disponible" tabindex="-1">${et("no_disponible_titulo")}</h2><p>${et("no_disponible_texto")}</p>
        <div class="acciones-vista"><button type="button" class="boton-secundario" data-vista="portal">${et("volver")}</button></div></div></section>`;
    } else {
      const total = lista ? `<p>${esc(catalogo.plural("publicacion.total", lista.items.length))}</p>` : "";
      const actualizar = lista && estado !== "denegada"
        ? `<button type="button" id="pb-actualizar" class="boton-secundario" data-pb-actualizar${estado === "cargando" ? ' disabled aria-busy="true"' : ""}>${et("actualizar")}</button>` : "";
      raiz.innerHTML = `<div class="publicacion-bases" data-publicacion-bases><section class="panel" aria-labelledby="pb-titulo">
        <header class="cabecera-panel"><div><h2 id="pb-titulo" tabindex="-1">${et("titulo")}</h2>${total}</div>
        <div class="acciones-vista">${actualizar}<button type="button" class="boton-secundario publicacion-bases-ayuda-boton" data-pb-ayuda aria-expanded="${ayuda}" aria-controls="pb-ayuda" aria-label="${et("ayuda_boton")}">?</button></div></header>
        <div class="cuerpo-panel"><p id="pb-ayuda" class="publicacion-bases-ayuda"${ayuda ? "" : " hidden"}>${et("ayuda")}</p>
        ${htmlResguardo()}${htmlCuerpo()}</div></section>${estado === "denegada" ? "" : htmlFormulario()}</div>`;
    }
    const destino = enfocar || idActivo;
    enfocar = "";
    if (destino) raiz.querySelector?.(`[id="${destino}"]`)?.focus?.({ preventScroll: false });
  }

  function denegar() {
    ++secuencia; controlador?.abort();
    lista = null; abierta = null; resguardo = null; intento = null; enviando = false;
    estado = "denegada"; alDenegacion(); alDisponibilidad("oculta"); pintar();
    anunciar(t("denegada"));
  }
  function noDisponible(disponibilidad = "no_habilitada") {
    ++secuencia; controlador?.abort();
    lista = null; abierta = null; enviando = false; estado = "no_disponible";
    alDisponibilidad(disponibilidad); enfocar = "pb-no-disponible"; pintar();
  }
  function nuevoControlador() {
    controlador?.abort();
    controlador = new AbortController();
    return { carga: ++secuencia, senal: controlador.signal };
  }

  async function cargar() {
    const { carga, senal } = nuevoControlador();
    estado = "cargando"; pintar();
    try {
      const recibida = await cliente.listar({ signal: senal });
      if (!vivo || senal.aborted || carga !== secuencia) return;
      lista = recibida; estado = "lista";
      if (abierta) {
        // El formulario solo sigue abierto si la versión que se declara sigue en la lista.
        abierta = lista.items.find((item) => item.convocatoria_ref === abierta.convocatoria_ref
          && item.huella_version_sha256 === abierta.huella_version_sha256) || null;
      }
      alDisponibilidad("disponible"); pintar();
    } catch (error) {
      if (!vivo || senal.aborted || carga !== secuencia) return;
      if (error?.estado === 401 || error?.estado === 403) { denegar(); return; }
      if (error?.estado === 404) { noDisponible("oculta"); return; }
      if (error?.estado === 503 && error?.codigo !== "servicio_no_disponible") { noDisponible(); return; }
      lista = null; estado = "error"; pintar(); anunciar(t("error"));
    }
  }

  function abrir(referencia) {
    const item = lista?.items.find((elemento) => elemento.convocatoria_ref === referencia);
    if (!item || enviando) return;
    abierta = item; errores = []; avisoEnvio = ""; intento = null; resguardo = null; aviso = "";
    campos = item.publicacion ? {
      medio: item.publicacion.medio, boletin_numero: item.publicacion.boletin_numero === null ? "" : String(item.publicacion.boletin_numero),
      anuncio_ref: item.publicacion.anuncio_ref, fecha_publicacion: item.publicacion.fecha_publicacion,
      direccion: item.publicacion.direccion ?? "",
    } : camposVacios();
    enfocar = "pb-form-titulo"; pintar();
  }
  function cerrarFormulario(foco = true) {
    const referencia = abierta?.convocatoria_ref;
    abierta = null; errores = []; avisoEnvio = ""; intento = null; campos = camposVacios();
    pintar();
    if (foco && referencia) raiz.querySelector?.(`[data-pb-abrir="${referencia}"]`)?.focus?.();
  }
  function erroresActuales() {
    if (!abierta) return [];
    return erroresCuerpoPublicacion(cuerpoDeFormulario(abierta, campos), hoyMadrid(ahora()));
  }

  async function enviar() {
    if (!abierta || enviando) return;
    const cuerpo = cuerpoDeFormulario(abierta, campos);
    errores = erroresCuerpoPublicacion(cuerpo, hoyMadrid(ahora()));
    avisoEnvio = "";
    if (errores.length) { enfocar = "pb-resumen"; pintar(); return; }
    const firma = JSON.stringify(cuerpo);
    if (!intento || intento.firma !== firma) intento = { clave: nuevaClave(), firma };
    const { carga, senal } = nuevoControlador();
    const declarada = abierta;
    enviando = true; pintar();
    try {
      const recibido = await cliente.declarar(cuerpo, { claveIdempotencia: intento.clave, signal: senal });
      if (!vivo || senal.aborted || carga !== secuencia) return;
      enviando = false; intento = null; abierta = null; campos = camposVacios(); errores = [];
      resguardo = { ...recibido, titulo: declarada.titulo };
      anunciar(`${t("resguardo_titulo")}. ${t("resguardo_recibo")}: ${recibido.recibo_ref}`);
      enfocar = "pb-resguardo-titulo";
      await cargar();
    } catch (error) {
      if (!vivo || senal.aborted || carga !== secuencia) return;
      enviando = false;
      const codigo = error?.codigo || "";
      if (error?.estado === 401 || error?.estado === 403) { denegar(); return; }
      if (error?.estado === 503 && codigo === "publicacion_oficial_no_habilitada") { noDisponible(); return; }
      if (error?.estado === 409) {
        intento = null; abierta = null; campos = camposVacios();
        aviso = codigo === "version_no_vigente" ? "error_version" : "error_conflicto";
        anunciar(t(aviso)); enfocar = "pb-titulo";
        await cargar();
        return;
      }
      if (error?.estado === 400) { intento = null; avisoEnvio = "error_solicitud"; }
      else if (error?.estado === 503 && codigo === "resultado_no_confirmado") avisoEnvio = "error_no_confirmado";
      else if (error?.estado === 503) avisoEnvio = "error_servicio";
      // Red, tiempo agotado o respuesta ilegible: el resultado es incierto y se
      // conserva la clave para que reenviar los mismos datos no duplique nada.
      else avisoEnvio = "error_envio";
      enfocar = "pb-aviso-envio"; pintar();
    }
  }

  const alPulsar = (evento) => {
    const objetivo = evento.target;
    if (objetivo?.closest?.("[data-pb-ayuda]")) {
      ayuda = !ayuda; enfocar = ""; pintar(); raiz.querySelector?.("[data-pb-ayuda]")?.focus?.(); return;
    }
    if (objetivo?.closest?.("[data-pb-actualizar]")) { aviso = ""; void cargar(); return; }
    if (objetivo?.closest?.("[data-pb-cerrar-resguardo]")) { resguardo = null; enfocar = "pb-titulo"; pintar(); return; }
    if (objetivo?.closest?.("[data-pb-cancelar]")) { cerrarFormulario(); return; }
    const ir = objetivo?.closest?.("[data-pb-ir]");
    if (ir) { evento.preventDefault?.(); raiz.querySelector?.(`[id="pb-${ir.getAttribute("data-pb-ir")}"]`)?.focus?.(); return; }
    const abrirBoton = objetivo?.closest?.("[data-pb-abrir]");
    if (abrirBoton) abrir(abrirBoton.getAttribute("data-pb-abrir"));
  };
  const alEscribir = (evento) => {
    const nombre = evento.target?.name;
    if (!abierta || !CAMPOS.includes(nombre)) return;
    campos[nombre] = String(evento.target.value ?? "");
    if (nombre === "medio" && evento.type === "change") {
      if (campos.medio === MEDIO_SIN_BOLETIN) campos.boletin_numero = "";
      errores = errores.filter((campoError) => campoError !== "boletin_numero" && campoError !== "medio");
      enfocar = "pb-medio"; pintar();
    }
  };
  // Al salir de un campo con datos se valida solo ese campo, sin repintar: el
  // foco sigue donde la persona lo ha llevado.
  const alSalir = (evento) => {
    const nombre = evento.target?.name;
    if (!abierta || !CAMPOS.includes(nombre) || nombre === "medio") return;
    if (campos[nombre] === "" && !errores.includes(nombre)) return;
    const conError = erroresActuales().includes(nombre) && !(campos[nombre] === "" && nombre === "direccion");
    errores = conError ? [...new Set([...errores, nombre])] : errores.filter((campoError) => campoError !== nombre);
    const control = raiz.querySelector?.(`[id="pb-${nombre}"]`);
    const mensaje = raiz.querySelector?.(`[id="pb-error-${nombre}"]`);
    if (control?.setAttribute) {
      if (conError) { control.setAttribute("aria-invalid", "true"); control.setAttribute("aria-describedby", `pb-error-${nombre}`); }
      else { control.removeAttribute("aria-invalid"); control.removeAttribute("aria-describedby"); }
    }
    if (mensaje) { mensaje.textContent = conError ? t(`error_${nombre}`) : ""; mensaje.hidden = !conError; }
  };
  const alEnviar = (evento) => {
    if (!evento.target?.matches?.("[data-pb-formulario]") && !evento.target?.closest?.("[data-pb-formulario]")) return;
    evento.preventDefault?.();
    void enviar();
  };
  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("input", alEscribir);
  raiz.addEventListener("change", alEscribir);
  raiz.addEventListener("focusout", alSalir);
  raiz.addEventListener("submit", alEnviar);
  const desmontar = () => {
    if (!vivo) return;
    vivo = false; ++secuencia; controlador?.abort();
    raiz.removeEventListener("click", alPulsar);
    raiz.removeEventListener("input", alEscribir);
    raiz.removeEventListener("change", alEscribir);
    raiz.removeEventListener("focusout", alSalir);
    raiz.removeEventListener("submit", alEnviar);
    raiz.replaceChildren();
  };
  signal?.addEventListener?.("abort", desmontar, { once: true });
  void cargar();
  return Object.freeze({ desmontar });
}
