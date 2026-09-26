/** Panel de cancelación del expediente antes de la fiscalización (RRHH). */

import { crearTraductorCancelacion } from "./i18n-cancelacion.js?v=20260926-cancelacion-v1";
import { renderizarJustificante } from "../../portal-justificante.js";

const escapar = (valor) => String(valor ?? "").replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

function instanteVisible(valor, locale) {
  const fecha = new Date(valor);
  return Number.isFinite(fecha.getTime())
    ? new Intl.DateTimeFormat(locale, { dateStyle: "long", timeStyle: "short", timeZone: "Europe/Madrid" }).format(fecha) : valor;
}

// El contexto sale del detalle ya autorizado y de su fila en el cuadro:
// expediente, versión, fase y estado. El servidor decide si admite la
// cancelación (fases del catálogo y barrera de la fiscalización); la vista
// solo evita ofrecerla cuando no procede.
export function contextoCancelacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente?.demostracion !== false
    || !Array.isArray(estado.cuadro?.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: ref }) => ref === estado.expediente.expediente_ref);
  if (!resumen || resumen.version !== estado.expediente.version || typeof resumen.fase_clave !== "string") return null;
  return Object.freeze({ expediente_ref: estado.expediente.expediente_ref, version: estado.expediente.version,
    fase_clave: resumen.fase_clave, estado_clave: resumen.estado_clave });
}

// Un 404 sin sobre de error del módulo significa que el servidor no compone
// la cancelación: el panel no se monta. Un 404 con sobre válido es un fallo.
export function rutaCancelacionNoMontada(error) {
  return error?.estado === 404 && error?.envelopeValido !== true;
}

export function montarPanelCancelacion({
  contenedor, cliente, contexto, mensajes = {}, locale = "es-ES", anunciar = () => {},
  confirmarOperacion = () => false, alConfirmar = () => {},
  generarClave = () => globalThis.crypto?.randomUUID?.(), avisoInicial = null,
} = {}) {
  if (!contenedor || typeof cliente?.consultarCancelacion !== "function" || !contexto) throw new TypeError("panel de cancelación no disponible");
  const t = crearTraductorCancelacion(mensajes);
  const controlador = new AbortController();
  let datos = null;
  let ocupado = false;
  let abierto = false;
  let clave = null;
  let ayudaAbierta = false;
  // Al volver a montar el panel tras cancelar, el justificante sigue a la vista.
  let aviso = avisoInicial?.tono === "exito" && typeof avisoInicial.texto === "string"
    ? { tono: "exito", texto: avisoInicial.texto, recibo: typeof avisoInicial.recibo === "string" ? avisoInicial.recibo : "" } : null;

  const etiquetaMotivo = (clave18n, etiqueta) => (clave18n && typeof mensajes[clave18n] === "string" ? mensajes[clave18n] : etiqueta);
  const etiquetaFase = (fase) => (t(`fase_${fase}`) === `fase_${fase}` ? fase : t(`fase_${fase}`));

  function justificante(referencia) {
    return renderizarJustificante(referencia, { escapar, etiqueta: t("justificante_registrado"), copiar: t("justificante_copiar"), copiado: t("justificante_copiado") });
  }

  function htmlAviso() {
    if (!aviso) return "";
    const recibo = aviso.recibo ? ` ${justificante(aviso.recibo)}` : "";
    return `<p class="ct-exp-mensaje ct-tono-${aviso.tono}" role="${aviso.tono === "peligro" ? "alert" : "status"}" data-ct-cancelacion-aviso tabindex="-1">${escapar(aviso.texto)}${recibo}</p>`;
  }

  function botonAyuda() {
    return `<button type="button" class="boton-secundario ct-seg-cese-ayuda" data-ct-cancelacion-ayuda aria-expanded="${ayudaAbierta}" aria-controls="ct-cancelacion-ayuda" aria-label="${escapar(t("ayuda_boton"))}">?</button>`;
  }

  function textoAyuda() {
    return `<p id="ct-cancelacion-ayuda" class="ct-seg-cese-texto-ayuda" ${ayudaAbierta ? "" : "hidden"}>${escapar(t("ayuda"))}</p>`;
  }

  function registrada(c) {
    const filas = [
      [t("cancelado_fecha"), instanteVisible(c.registrada_en, locale)],
      [t("cancelado_fase"), etiquetaFase(c.fase_previa)],
      ...(c.observaciones ? [[t("cancelado_observaciones"), c.observaciones]] : []),
    ];
    return `<section class="ct-exp-fase-panel ct-seg-cese" data-ct-cancelacion aria-labelledby="ct-cancelacion-titulo">
      <header class="ct-exp-fase-panel-cabecera"><div><h3 id="ct-cancelacion-titulo">${escapar(t("titulo"))}</h3>
      <p>${escapar(t("cancelado_resumen", { quien: t(`quien_${c.canal}`), motivo: etiquetaMotivo(c.motivo_clave_i18n, c.motivo_etiqueta) }))}</p></div>
      <div class="ct-exp-fase-panel-acciones"><span class="ct-exp-chip ct-fase-cancelado">${escapar(t("estado_cancelado"))}</span>${botonAyuda()}</div></header>${textoAyuda()}
      <dl class="ct-exp-fase-datos">${filas.map(([a, b]) => `<div><dt>${escapar(a)}</dt><dd>${escapar(b)}</dd></div>`).join("")}
      <div><dt>${escapar(t("justificante_registrado"))}</dt><dd>${renderizarJustificante(c.recibo_ref, { escapar, etiqueta: "", copiar: t("justificante_copiar"), copiado: t("justificante_copiado") })}</dd></div></dl>
      ${htmlAviso()}</section>`;
  }

  function formulario(motivos) {
    const deshabilitado = ocupado ? "disabled" : "";
    const opciones = motivos.map((m) => `<option value="${escapar(m.clave)}">${escapar(etiquetaMotivo(m.clave_i18n, m.etiqueta))}</option>`).join("");
    return `<form class="ct-seg-cese-form" data-ct-cancelacion-form aria-labelledby="ct-cancelacion-titulo" novalidate>
      <div class="ct-seg-cese-rejilla">
      <label class="ct-campo"><span>${escapar(t("motivo"))}</span><select name="motivo_clave" required ${deshabilitado}><option value=""></option>${opciones}</select></label>
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" ${deshabilitado}></textarea></label>
      </div>
      <div class="ct-acciones"><button type="submit" class="boton-peligro" ${deshabilitado}>${escapar(t("enviar"))}</button>
      <button type="button" class="boton-secundario" data-ct-cancelacion-cerrar ${deshabilitado}>${escapar(t("cerrar_formulario"))}</button></div></form>`;
  }

  function pintar() {
    if (controlador.signal.aborted) return;
    contenedor.hidden = false;
    if (datos?.ausente) {
      contenedor.innerHTML = "";
      contenedor.hidden = true;
      return;
    }
    if (!datos) {
      contenedor.innerHTML = `<section class="ct-exp-fase-panel ct-seg-cese" aria-busy="true"><p role="status">${escapar(t("cargando"))}</p></section>`;
      return;
    }
    if (datos.error) {
      contenedor.innerHTML = `<section class="ct-exp-fase-panel ct-seg-cese" role="alert"><p>${escapar(t("no_disponible"))}</p><button type="button" class="boton-secundario" data-ct-cancelacion-reintentar>${escapar(t("reintentar"))}</button></section>`;
      return;
    }
    if (datos.cancelacion) {
      contenedor.innerHTML = registrada(datos.cancelacion);
      return;
    }
    // Solo se ofrece en curso y en una fase que admita la regla vigente.
    const admitida = contexto.estado_clave === "en_curso" && datos.fases_admitidas.includes(contexto.fase_clave);
    if (!admitida && !aviso) {
      contenedor.innerHTML = "";
      contenedor.hidden = true;
      return;
    }
    contenedor.innerHTML = `<section class="ct-exp-fase-panel ct-seg-cese" data-ct-cancelacion aria-labelledby="ct-cancelacion-titulo" ${ocupado ? 'aria-busy="true"' : ""}>
      <header class="ct-exp-fase-panel-cabecera"><div><h3 id="ct-cancelacion-titulo">${escapar(t("titulo"))}</h3></div>
      <div class="ct-exp-fase-panel-acciones">${admitida ? `<button type="button" class="boton-secundario" data-ct-cancelacion-abrir aria-expanded="${abierto}" ${ocupado ? "disabled" : ""}>${escapar(t("abrir"))}</button>` : ""}${botonAyuda()}</div></header>${textoAyuda()}
      ${htmlAviso()}${admitida && abierto ? formulario(datos.motivos) : ""}</section>`;
  }

  async function cargar() {
    datos = null;
    pintar();
    try {
      datos = await cliente.consultarCancelacion(contexto.expediente_ref, { signal: controlador.signal });
    } catch (error) {
      if (controlador.signal.aborted) return;
      datos = rutaCancelacionNoMontada(error) ? { ausente: true } : { error: true };
    }
    pintar();
  }

  function enfocar(selector) {
    contenedor.querySelector?.(selector)?.focus?.();
  }

  async function enviar(formularioHTML) {
    if (ocupado) return;
    const campos = Object.fromEntries(new FormData(formularioHTML).entries());
    if (!clave) clave = generarClave();
    const solicitud = { expediente_ref: contexto.expediente_ref, version_esperada: contexto.version, clave_idempotencia: clave,
      motivo_clave: String(campos.motivo_clave ?? "").trim(), observaciones: String(campos.observaciones ?? "").trim() };
    if (solicitud.motivo_clave === "") {
      aviso = { tono: "peligro", texto: t("error_contenido_no_valido") };
      pintar();
      enfocar("[data-ct-cancelacion-aviso]");
      return;
    }
    let confirmada = false;
    try { confirmada = confirmarOperacion({ titulo: t("confirmar_titulo"), advertencia: t("confirmar"), referencia: contexto.expediente_ref }) === true; } catch {}
    if (!confirmada) return;
    ocupado = true;
    aviso = { tono: "info", texto: t("enviando") };
    pintar();
    try {
      const recibo = await cliente.cancelarExpediente(solicitud, { signal: controlador.signal });
      clave = null;
      aviso = { tono: "exito", texto: t("recibo"), recibo: recibo.recibo_ref };
      anunciar(t("recibo"), "exito");
      ocupado = false;
      abierto = false;
      pintar();
      enfocar("[data-ct-cancelacion-aviso]");
      alConfirmar(recibo, aviso);
    } catch (error) {
      ocupado = false;
      if (controlador.signal.aborted) return;
      if (error?.resultadoIndeterminado) aviso = { tono: "aviso", texto: t("error_indeterminado") };
      else {
        if (error?.envelopeValido) clave = null;
        const conocido = typeof error?.codigo === "string" && t(`error_${error.codigo}`) !== `error_${error.codigo}`;
        aviso = { tono: "peligro", texto: conocido ? t(`error_${error.codigo}`) : (error instanceof TypeError ? t("error_contenido_no_valido") : t("error_general")) };
      }
      pintar();
      enfocar("[data-ct-cancelacion-aviso]");
    }
  }

  function alPulsar(evento) {
    const boton = evento.target?.closest?.("button");
    if (!boton) return;
    if (boton.matches("[data-ct-cancelacion-ayuda]")) {
      ayudaAbierta = !ayudaAbierta;
      boton.setAttribute("aria-expanded", String(ayudaAbierta));
      const texto = contenedor.querySelector?.("#ct-cancelacion-ayuda");
      if (texto) texto.hidden = !ayudaAbierta;
      return;
    }
    if (ocupado) return;
    if (boton.matches("[data-ct-cancelacion-abrir]")) {
      abierto = !abierto;
      if (!abierto) aviso = null;
      pintar();
      enfocar(abierto ? 'select[name="motivo_clave"]' : "[data-ct-cancelacion-abrir]");
    } else if (boton.matches("[data-ct-cancelacion-cerrar]")) {
      abierto = false;
      aviso = null;
      pintar();
      enfocar("[data-ct-cancelacion-abrir]");
    } else if (boton.matches("[data-ct-cancelacion-reintentar]")) {
      cargar();
    }
  }

  function alEnviar(evento) {
    const formularioHTML = evento.target?.closest?.("[data-ct-cancelacion-form]");
    if (!formularioHTML) return;
    evento.preventDefault();
    enviar(formularioHTML);
  }

  contenedor.addEventListener("click", alPulsar);
  contenedor.addEventListener("submit", alEnviar);
  cargar();
  return () => {
    controlador.abort();
    contenedor.removeEventListener("click", alPulsar);
    contenedor.removeEventListener("submit", alEnviar);
  };
}
