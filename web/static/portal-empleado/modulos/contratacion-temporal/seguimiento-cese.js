/** Panel de seguimiento del nombramiento: cese, cierre y modificación. */

import { crearTraductorSeguimientoCese } from "./i18n-seguimiento-cese.js";

const escapar = (valor) => String(valor ?? "").replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

function fechaVisible(valor, locale) {
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.getTime())
    ? new Intl.DateTimeFormat(locale, { dateStyle: "long", timeZone: "UTC" }).format(fecha) : valor;
}

// El contexto sale del detalle ya autorizado: expediente, versión, fase y
// estado. El servidor revalida todo al registrar; la vista solo acota.
export function contextoSeguimientoCeseDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente?.demostracion !== false
    || !Array.isArray(estado.cuadro?.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: ref }) => ref === estado.expediente.expediente_ref);
  if (!resumen || resumen.version !== estado.expediente.version || !["nombramiento"].includes(resumen.fase_clave)) return null;
  return Object.freeze({ expediente_ref: estado.expediente.expediente_ref, version: estado.expediente.version,
    fase_clave: resumen.fase_clave, estado_clave: resumen.estado_clave });
}

async function huellaSHA256(archivo) {
  const contenido = await archivo.arrayBuffer();
  const resumen = await globalThis.crypto.subtle.digest("SHA-256", contenido);
  return [...new Uint8Array(resumen)].map((b) => b.toString(16).padStart(2, "0")).join("");
}

// Un 404 sin sobre de error del módulo («vec route not found») significa que
// el servidor no compone el seguimiento de cese: no se monta el panel, igual
// que la firma cuando su registro no está compuesto. Un 404 con sobre válido
// es un expediente inexistente y sigue siendo un fallo.
export function rutaSeguimientoCeseNoMontada(error) {
  return error?.estado === 404 && error?.envelopeValido !== true;
}

export function montarPanelSeguimientoCese({
  contenedor, cliente, contexto, mensajes = {}, locale = "es-ES", anunciar = () => {},
  confirmarOperacion = () => false, alConfirmar = () => {},
  generarClave = () => globalThis.crypto?.randomUUID?.(),
} = {}) {
  if (!contenedor || typeof cliente?.consultarSeguimientoCese !== "function" || !contexto) throw new TypeError("panel de seguimiento no disponible");
  const t = crearTraductorSeguimientoCese(mensajes);
  const controlador = new AbortController();
  const claves = new Map();
  let datos = null;
  let ocupado = false;
  let ayudaAbierta = false;
  let aviso = null;

  const etiquetaOpcion = (o) => (o.clave_i18n && typeof mensajes[o.clave_i18n] === "string" ? mensajes[o.clave_i18n] : o.etiqueta);
  const claveDe = (operacion) => {
    if (!claves.has(operacion)) claves.set(operacion, generarClave());
    return claves.get(operacion);
  };

  function cabecera(estado) {
    const chip = estado.cierre ? ["estado_cerrado", "ct-fase-completado"] : estado.cese ? ["estado_cesado", "ct-fase-espera"] : ["estado_vigente", "ct-fase-en_curso"];
    return `<header class="ct-exp-fase-panel-cabecera"><div><h3 id="ct-seg-cese-titulo">${escapar(t("titulo"))}</h3><p>${escapar(t("subtitulo"))}</p></div>
      <div class="ct-exp-fase-panel-acciones"><span class="ct-exp-chip ${chip[1]}">${escapar(t(chip[0]))}</span>
      <button type="button" class="boton-secundario ct-seg-cese-ayuda" data-ct-seg-ayuda aria-expanded="${ayudaAbierta}" aria-controls="ct-seg-cese-ayuda" aria-label="${escapar(t("ayuda_boton"))}">?</button></div></header>
      <p id="ct-seg-cese-ayuda" class="ct-seg-cese-texto-ayuda" ${ayudaAbierta ? "" : "hidden"}>${escapar(t("ayuda"))}</p>`;
  }

  function resumen(estado, opciones) {
    const causa = estado.cese ? opciones.causas_cese.find((c) => c.clave === estado.cese.causa_clave) : null;
    const filas = [
      [t("incorporacion"), estado.incorporacion ? t("incorporacion_inicio", { fecha: fechaVisible(estado.incorporacion.inicio, locale) }) : t("sin_incorporacion")],
      [t("cese"), estado.cese ? t("cese_registrado", { causa: causa ? etiquetaOpcion(causa) : estado.cese.causa_clave, fecha: fechaVisible(estado.cese.fecha_efecto, locale) }) : t("pendiente")],
      [t("cierre"), estado.cierre ? t("cierre_registrado", { ginpix: estado.cierre.ginpix_numero || t("cierre_sin_ginpix") }) : t("pendiente")],
    ];
    return `<dl class="ct-exp-fase-datos">${filas.map(([a, b]) => `<div><dt>${escapar(a)}</dt><dd>${escapar(b)}</dd></div>`).join("")}</dl>`;
  }

  const deshabilitado = () => (ocupado ? "disabled" : "");

  function formularioCese(opciones, estado) {
    const sinIncorporacion = !estado.incorporacion;
    const opcionesCausa = opciones.causas_cese.map((c) => `<option value="${escapar(c.clave)}" data-justificante="${escapar(c.justificante_tipo)}">${escapar(etiquetaOpcion(c))}</option>`).join("");
    return `<form class="ct-seg-cese-form" data-ct-seg-form="cese" aria-labelledby="ct-seg-cese-form-titulo" novalidate>
      <h4 id="ct-seg-cese-form-titulo">${escapar(t("cese_titulo"))}</h4>
      ${sinIncorporacion ? `<p class="ct-seg-cese-motivo" role="note">${escapar(t("cese_requiere_incorporacion"))}</p>` : ""}
      <div class="ct-seg-cese-rejilla">
      <label class="ct-campo"><span>${escapar(t("cese_causa"))}</span><select name="causa_clave" required ${deshabilitado()} ${sinIncorporacion ? "disabled" : ""}>${opcionesCausa}</select><small data-ct-seg-justificante-tipo></small></label>
      <label class="ct-campo"><span>${escapar(t("cese_fecha"))}</span><input type="date" name="fecha_efecto" required ${deshabilitado()} ${sinIncorporacion ? "disabled" : ""}></label>
      <label class="ct-campo"><span>${escapar(t("cese_justificante_ref"))}</span><input type="text" name="justificante_ref" required maxlength="160" autocomplete="off" ${deshabilitado()} ${sinIncorporacion ? "disabled" : ""}></label>
      <label class="ct-campo"><span>${escapar(t("cese_justificante_archivo"))}</span><input type="file" data-ct-seg-justificante ${deshabilitado()} ${sinIncorporacion ? "disabled" : ""}></label>
      <label class="ct-campo"><span>${escapar(t("cese_justificante_huella"))}</span><input type="text" name="justificante_sha256" required pattern="[0-9a-f]{64}" maxlength="64" spellcheck="false" autocomplete="off" ${deshabilitado()} ${sinIncorporacion ? "disabled" : ""}></label>
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("cese_observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" ${deshabilitado()} ${sinIncorporacion ? "disabled" : ""}></textarea></label>
      </div>
      <div class="ct-acciones"><button type="submit" class="boton-primario" ${deshabilitado()} ${sinIncorporacion ? "disabled" : ""}>${escapar(t("cese_enviar"))}</button></div></form>`;
  }

  function formularioCierre(opciones) {
    const exigeGINPIX = opciones.condiciones_cierre.includes("ginpix_confirmado");
    return `<form class="ct-seg-cese-form" data-ct-seg-form="cierre" aria-labelledby="ct-seg-cierre-titulo" novalidate>
      <h4 id="ct-seg-cierre-titulo">${escapar(t("cierre_titulo"))}</h4>
      <p class="ct-seg-cese-condiciones"><span>${escapar(t("cierre_condiciones"))}:</span> ${opciones.condiciones_cierre.map((c) => `<span class="ct-exp-chip ct-fase-completado">${escapar(t(`condicion_${c}`))}</span>`).join(" ")}</p>
      <div class="ct-seg-cese-rejilla">
      ${exigeGINPIX ? `<label class="ct-campo"><span>${escapar(t("cierre_ginpix_numero"))}</span><input type="text" name="ginpix_numero" required maxlength="64" autocomplete="off" ${deshabilitado()}></label>
      <label class="ct-campo"><span>${escapar(t("cierre_ginpix_fecha"))}</span><input type="date" name="ginpix_confirmada_en" required ${deshabilitado()}></label>` : ""}
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("cierre_observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" ${deshabilitado()}></textarea></label>
      </div>
      <div class="ct-acciones"><button type="submit" class="boton-primario" ${deshabilitado()}>${escapar(t("cierre_enviar"))}</button></div></form>`;
  }

  function formularioModificacion(opciones) {
    const motivos = opciones.motivos_modificacion.map((m) => `<option value="${escapar(m.clave)}">${escapar(etiquetaOpcion(m))}</option>`).join("");
    const fase = t(`fase_${opciones.fase_retorno_modificacion}`);
    return `<form class="ct-seg-cese-form" data-ct-seg-form="modificacion" aria-labelledby="ct-seg-mod-titulo" novalidate>
      <h4 id="ct-seg-mod-titulo">${escapar(t("modificacion_titulo"))}</h4>
      <p class="ct-seg-cese-condiciones"><span class="ct-exp-chip ct-fase-espera">${escapar(t("modificacion_vuelta", { fase }))}</span></p>
      <div class="ct-seg-cese-rejilla">
      <label class="ct-campo"><span>${escapar(t("modificacion_motivo"))}</span><select name="motivo_clave" required ${deshabilitado()}>${motivos}</select></label>
      <label class="ct-campo"><span>${escapar(t("modificacion_inicio"))}</span><input type="date" name="periodo_inicio" required ${deshabilitado()}></label>
      <label class="ct-campo"><span>${escapar(t("modificacion_fin"))}</span><input type="date" name="periodo_fin" required ${deshabilitado()}></label>
      <label class="ct-campo"><span>${escapar(t("modificacion_jornada"))}</span><input type="number" name="porcentaje_jornada" min="0.01" max="100" step="0.01" required inputmode="decimal" ${deshabilitado()}></label>
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("modificacion_observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" required ${deshabilitado()}></textarea></label>
      </div>
      <div class="ct-acciones"><button type="submit" class="boton-primario" ${deshabilitado()}>${escapar(t("modificacion_enviar"))}</button></div></form>`;
  }

  function pintar() {
    if (controlador.signal.aborted) return;
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
      contenedor.innerHTML = `<section class="ct-exp-fase-panel ct-seg-cese" role="alert"><p>${escapar(t("no_disponible"))}</p><button type="button" class="boton-secundario" data-ct-seg-reintentar>${escapar(t("reintentar"))}</button></section>`;
      return;
    }
    const { opciones, estado } = datos;
    const vigente = contexto.estado_clave === "en_curso";
    const formularios = [];
    if (vigente && !estado.cese) formularios.push(formularioCese(opciones, estado), formularioModificacion(opciones));
    if (vigente && estado.cese && !estado.cierre) formularios.push(formularioCierre(opciones));
    contenedor.innerHTML = `<section class="ct-exp-fase-panel ct-seg-cese" data-ct-seg-cese aria-labelledby="ct-seg-cese-titulo" ${ocupado ? 'aria-busy="true"' : ""}>
      ${cabecera(estado)}${resumen(estado, opciones)}
      ${aviso ? `<p class="ct-exp-mensaje ct-tono-${aviso.tono}" role="${aviso.tono === "peligro" ? "alert" : "status"}" data-ct-seg-aviso tabindex="-1">${escapar(aviso.texto)}</p>` : ""}
      ${formularios.join("")}</section>`;
    actualizarJustificante();
  }

  function actualizarJustificante() {
    const select = contenedor.querySelector('[data-ct-seg-form="cese"] select[name="causa_clave"]');
    const salida = contenedor.querySelector("[data-ct-seg-justificante-tipo]");
    if (!select || !salida) return;
    const tipo = select.selectedOptions?.[0]?.dataset?.justificante ?? "";
    salida.textContent = tipo ? t("cese_justificante_tipo", { tipo }) : "";
  }

  async function cargar() {
    datos = null;
    pintar();
    try {
      datos = await cliente.consultarSeguimientoCese(contexto.expediente_ref, { signal: controlador.signal });
    } catch (error) {
      if (controlador.signal.aborted) return;
      datos = rutaSeguimientoCeseNoMontada(error) ? { ausente: true } : { error: true };
    }
    pintar();
  }

  function leerFormulario(formulario) {
    const campos = Object.fromEntries(new FormData(formulario).entries());
    for (const clave of Object.keys(campos)) campos[clave] = String(campos[clave]).trim();
    return campos;
  }

  function solicitudPara(tipo, campos) {
    const base = { expediente_ref: contexto.expediente_ref, version_esperada: contexto.version, clave_idempotencia: claveDe(`${tipo}:${contexto.version}`) };
    if (tipo === "cese") {
      return ["registrarCese", { ...base, causa_clave: campos.causa_clave ?? "", fecha_efecto: campos.fecha_efecto ?? "",
        justificante_ref: campos.justificante_ref ?? "", justificante_sha256: (campos.justificante_sha256 ?? "").toLowerCase(), observaciones: campos.observaciones ?? "" }];
    }
    if (tipo === "cierre") {
      return ["cerrarExpediente", { ...base, ginpix_numero: campos.ginpix_numero ?? "", ginpix_confirmada_en: campos.ginpix_confirmada_en ?? "", observaciones: campos.observaciones ?? "" }];
    }
    const jornada = Math.round(Number(String(campos.porcentaje_jornada ?? "").replace(",", ".")) * 100);
    return ["modificarTrasNombramiento", { ...base, motivo_clave: campos.motivo_clave ?? "", periodo_inicio: campos.periodo_inicio ?? "",
      periodo_fin: campos.periodo_fin ?? "", porcentaje_jornada: Number.isSafeInteger(jornada) ? jornada : 0, observaciones: campos.observaciones ?? "" }];
  }

  async function enviar(formulario) {
    if (ocupado) return;
    const tipo = formulario.dataset.ctSegForm;
    const [metodo, solicitud] = solicitudPara(tipo, leerFormulario(formulario));
    let confirmada = false;
    const titulos = { cese: "cese_titulo", cierre: "cierre_titulo", modificacion: "modificacion_titulo" };
    try { confirmada = confirmarOperacion({ titulo: t(titulos[tipo]), advertencia: t("confirmar"), referencia: contexto.expediente_ref }) === true; } catch {}
    if (!confirmada) return;
    ocupado = true;
    aviso = { tono: "info", texto: t("enviando") };
    pintar();
    try {
      const recibo = await cliente[metodo](solicitud, { signal: controlador.signal });
      claves.delete(`${tipo}:${contexto.version}`);
      let texto = t("recibo", { recibo: recibo.recibo_ref, version: recibo.version_resultante });
      if (recibo.coste_centimos) {
        texto += ` ${t("recibo_coste", { coste: new Intl.NumberFormat(locale, { style: "currency", currency: "EUR" }).format(recibo.coste_centimos / 100) })}`;
      }
      aviso = { tono: "exito", texto };
      anunciar(texto, "exito");
      ocupado = false;
      alConfirmar(recibo);
    } catch (error) {
      ocupado = false;
      if (controlador.signal.aborted) return;
      if (error?.resultadoIndeterminado) aviso = { tono: "aviso", texto: t("error_indeterminado") };
      else {
        if (error?.envelopeValido) claves.delete(`${tipo}:${contexto.version}`);
        const conocido = typeof error?.codigo === "string" && t(`error_${error.codigo}`) !== `error_${error.codigo}`;
        aviso = { tono: "peligro", texto: conocido ? t(`error_${error.codigo}`) : (error instanceof TypeError ? t("error_contenido_no_valido") : t("error_general")) };
      }
      pintar();
      contenedor.querySelector("[data-ct-seg-aviso]")?.focus?.();
    }
  }

  async function alCambiar(evento) {
    const objetivo = evento.target;
    if (objetivo?.matches?.('select[name="causa_clave"]')) actualizarJustificante();
    if (objetivo?.matches?.("[data-ct-seg-justificante]") && objetivo.files?.[0]) {
      const campo = contenedor.querySelector('input[name="justificante_sha256"]');
      if (campo) campo.value = await huellaSHA256(objetivo.files[0]);
    }
  }

  function alPulsar(evento) {
    const boton = evento.target?.closest?.("button");
    if (!boton) return;
    if (boton.matches("[data-ct-seg-ayuda]")) {
      ayudaAbierta = !ayudaAbierta;
      boton.setAttribute("aria-expanded", String(ayudaAbierta));
      const texto = contenedor.querySelector("#ct-seg-cese-ayuda");
      if (texto) texto.hidden = !ayudaAbierta;
    } else if (boton.matches("[data-ct-seg-reintentar]")) {
      cargar();
    }
  }

  function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-seg-form]");
    if (!formulario) return;
    evento.preventDefault();
    enviar(formulario);
  }

  contenedor.addEventListener("click", alPulsar);
  contenedor.addEventListener("change", alCambiar);
  contenedor.addEventListener("submit", alEnviar);
  cargar();
  return () => {
    controlador.abort();
    contenedor.removeEventListener("click", alPulsar);
    contenedor.removeEventListener("change", alCambiar);
    contenedor.removeEventListener("submit", alEnviar);
  };
}
