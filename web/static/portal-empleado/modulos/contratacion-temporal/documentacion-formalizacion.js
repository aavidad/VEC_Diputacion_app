/**
 * Documentación para la formalización tras aceptar una oferta (dudas 9 y 18).
 *
 * Muestra a RRHH los documentos exigidos por el catálogo de reglas, el plazo
 * calculado desde la aceptación, el estado de cada documento (pendiente o
 * aportado) y el aviso de vencimiento. RRHH anota cada documento aportado con
 * la referencia del original y su huella SHA-256 mediante el registro común de
 * referencias externas de Documentos; el fichero no sale de este equipo.
 * El panel se pinta dentro del formulario de llamamiento, que repinta su HTML
 * completo: por eso conserva su estado aquí y se vuelve a pintar en el nuevo
 * contenedor cada vez. No guarda nada en el navegador.
 */
import { escaparHTML as e } from "./componentes-expedientes.js";

const MAX_FICHERO = 20 * 1024 * 1024;
const REFERENCIA = /^[!-~]{3,128}$/u;
const REFERENCIA_PROHIBIDA = /[/\\*?%]|\.\./u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const EXPEDIENTE_CT = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const OPACA = /^(?:ref:[0-9a-f]{64}|[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$/u;

function fechaCivil(locale, valor) {
  const [anio, mes, dia] = String(valor).split("-").map(Number);
  if (!anio || !mes || !dia) return String(valor);
  return new Intl.DateTimeFormat(locale, { dateStyle: "long", timeZone: "UTC" }).format(Date.UTC(anio, mes - 1, dia));
}

export function crearPanelDocumentacionFormalizacion({
  fuente, t, locale = "es-ES", zonaHoraria = "Europe/Madrid", criptografia = globalThis.crypto,
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(), anunciar = () => {},
} = {}) {
  if (typeof fuente?.consultar !== "function" || typeof fuente.anotados !== "function"
    || typeof fuente.registrar !== "function" || typeof t !== "function"
    || typeof generarClaveIdempotencia !== "function" || typeof anunciar !== "function") {
    throw new TypeError("dependencias del panel de documentación no válidas");
  }
  const instante = new Intl.DateTimeFormat(locale, { dateStyle: "long", timeStyle: "short", timeZone: zonaHoraria });
  let activo = true;
  let contenedor = null;
  let estado = null;

  function nombreDocumento(clave) {
    try { return t(`ct_formalizacion_documento_${clave}`); } catch { return t("ct_formalizacion_documento_sin_etiqueta", { clave }); }
  }

  function nuevoEstado(aceptadaEn, expedienteRef) {
    return { aceptadaEn, expedienteRef, carga: "cargando", error: "", datos: null, anotados: null,
      anotadosError: false, edicion: null, mensaje: "", tono: "informacion", controlador: new AbortController() };
  }

  async function cargar() {
    const actual = estado;
    actual.carga = "cargando";
    actual.error = "";
    repintar();
    try {
      const datos = await fuente.consultar({ expedienteRef: actual.expedienteRef, aceptadaEn: actual.aceptadaEn, signal: actual.controlador.signal });
      if (!activo || estado !== actual) return;
      actual.datos = datos;
      actual.carga = "listo";
      try {
        actual.anotados = await fuente.anotados({ expedienteRef: datos.expediente_documental_ref, signal: actual.controlador.signal });
        actual.anotadosError = false;
      } catch {
        actual.anotados = null;
        actual.anotadosError = true;
      }
    } catch (error) {
      if (!activo || estado !== actual || error?.codigo === "cancelado") return;
      actual.carga = "error";
      actual.error = error?.codigo === "sin_reglas" ? "ct_formalizacion_sin_reglas"
        : error?.codigo === "denegado" ? "ct_formalizacion_denegado" : "ct_formalizacion_no_disponible";
    }
    if (activo && estado === actual) repintar();
  }

  function pendientes() {
    if (!estado?.datos || !estado.anotados) return null;
    return estado.datos.documentos.filter((d) => !estado.anotados.has(d.tipo_documental)).length;
  }

  function chip(clase, texto) {
    return `<span class="estado-chip ${clase}">${e(texto)}</span>`;
  }

  function procedencia(regla) {
    return `<p class="ct-ayuda" data-ct-formalizacion-procedencia="${e(regla.clave)}">
      ${chip(regla.ejemplo ? "violeta" : "neutro", t(regla.ejemplo ? "ct_formalizacion_regla_ejemplo" : "ct_formalizacion_regla_reglamento"))}
      ${e(t("ct_formalizacion_procedencia", { norma: regla.norma, articulo: regla.articulo ?? "", referencia: regla.referencia }))}
      ${regla.ejemplo ? ` · ${e(t("ct_formalizacion_duda", { duda: regla.duda }))}` : ""}</p>`;
  }

  function plazos() {
    const { plazo_documentacion: doc, plazo_incorporacion: inc, aceptada_en: aceptada } = estado.datos;
    const claseDoc = { en_curso: "info", ultimo_dia: "", vencido: "peligro" }[doc.estado];
    const filas = [`<div><dt>${e(t("ct_formalizacion_aceptada_en"))}</dt><dd>${e(instante.format(new Date(aceptada)))}</dd></div>`,
      `<div data-ct-formalizacion-plazo="documentacion"><dt>${e(t("ct_formalizacion_plazo_documentacion"))}</dt>
        <dd>${e(t("ct_formalizacion_plazo_documentacion_valor", { cantidad: doc.regla.cantidad ?? "", fecha: fechaCivil(locale, doc.ultimo_dia) }))}
        ${chip(claseDoc, t(`ct_formalizacion_estado_${doc.estado}`))}
        ${doc.prorrogado ? ` ${e(t("ct_formalizacion_prorrogado"))}` : ""}</dd></div>`];
    if (inc) {
      const cumplido = inc.estado === "vencido";
      filas.push(`<div data-ct-formalizacion-plazo="incorporacion"><dt>${e(t("ct_formalizacion_plazo_incorporacion"))}</dt>
        <dd>${e(t("ct_formalizacion_plazo_incorporacion_valor", { cantidad: inc.regla.cantidad ?? "", fecha: fechaCivil(locale, inc.ultimo_dia) }))}
        ${chip(cumplido ? "exito" : "neutro", t(cumplido ? "ct_formalizacion_incorporacion_cumplido" : "ct_formalizacion_incorporacion_en_curso"))}</dd></div>`);
    }
    const reglas = [doc.regla, estado.datos.regla_documentos, ...(inc ? [inc.regla] : [])]
      .filter((regla, indice, lista) => lista.findIndex((otra) => otra.clave === regla.clave) === indice);
    return `<dl class="ct-resumen">${filas.join("")}</dl>${reglas.map(procedencia).join("")}`;
  }

  function aviso() {
    const quedan = pendientes();
    const doc = estado.datos.plazo_documentacion;
    if (quedan === 0) return `<p class="ct-exp-mensaje ct-tono-informacion" role="status" data-ct-formalizacion-aviso="completa">${e(t("ct_formalizacion_completa"))}</p>`;
    if (quedan === null || doc.estado === "en_curso") return "";
    const clave = doc.estado === "vencido" ? "ct_formalizacion_aviso_vencido" : "ct_formalizacion_aviso_ultimo_dia";
    return `<p class="ct-exp-mensaje ${doc.estado === "vencido" ? "ct-tono-error" : "ct-tono-aviso"}" role="alert"
      data-ct-formalizacion-aviso="${e(doc.estado)}">${e(t(clave, { pendientes: quedan, fecha: fechaCivil(locale, doc.ultimo_dia) }))}</p>`;
  }

  function fila(documento) {
    const anotacion = estado.anotados?.get(documento.tipo_documental);
    const estadoDoc = estado.anotados === null ? chip("neutro", t("ct_formalizacion_estado_no_consultado"))
      : anotacion ? chip("exito", t("ct_formalizacion_aportado")) : chip("", t("ct_formalizacion_pendiente"));
    let accion = "";
    if (anotacion) {
      accion = `<code>${e(t("ct_formalizacion_anotacion_valor", { numero: anotacion.numero_vec, huella: anotacion.huella.slice(0, 12) }))}</code>`;
    } else if (!documento.registrable) {
      accion = `<span class="ct-ayuda">${e(t("ct_formalizacion_no_registrable"))}</span>`;
    } else if (estado.anotados !== null) {
      accion = `<button class="boton-secundario" type="button" data-ct-formalizacion-anotar="${e(documento.clave)}"
        aria-label="${e(t("ct_formalizacion_anotar_titulo", { documento: nombreDocumento(documento.clave) }))}"
        ${estado.edicion ? "disabled" : ""}>${e(t("ct_formalizacion_anotar"))}</button>`;
    }
    return `<tr data-ct-formalizacion-documento="${e(documento.clave)}"><th scope="row">${e(nombreDocumento(documento.clave))}</th>
      <td>${estadoDoc}</td><td>${accion}</td></tr>`;
  }

  function formularioEdicion() {
    const edicion = estado.edicion;
    if (!edicion) return "";
    const nombre = nombreDocumento(edicion.clave);
    return `<form class="ct-llamamiento-paso" data-ct-formalizacion-form novalidate aria-busy="${edicion.ocupado || edicion.calculando}">
      <fieldset${edicion.ocupado ? " disabled" : ""}><legend id="ct-formalizacion-edicion-titulo" tabindex="-1">${e(t("ct_formalizacion_anotar_titulo", { documento: nombre }))}</legend>
        <div class="ct-campos">
          <div class="ct-campo"><label for="ct-formalizacion-referencia">${e(t("ct_formalizacion_referencia"))} *</label>
            <input id="ct-formalizacion-referencia" name="referencia" type="text" maxlength="128" autocomplete="off" spellcheck="false"
              required value="${e(edicion.referencia)}" data-ct-formalizacion-referencia></div>
          <div class="ct-campo"><label for="ct-formalizacion-archivo">${e(t("ct_formalizacion_archivo"))} *</label>
            <input id="ct-formalizacion-archivo" type="file" required aria-describedby="ct-formalizacion-huella" data-ct-formalizacion-archivo></div>
        </div>
        <p id="ct-formalizacion-huella" class="ct-ayuda" role="status" aria-live="polite">${edicion.calculando ? e(t("ct_formalizacion_huella_calculando"))
          : edicion.huella ? e(t("ct_formalizacion_huella_calculada", { huella: edicion.huella })) : ""}</p>
        ${edicion.mensaje ? `<p class="ct-exp-mensaje ct-tono-${e(edicion.tono)}" role="${edicion.tono === "error" ? "alert" : "status"}"
          data-ct-formalizacion-edicion-mensaje>${e(t(edicion.mensaje))}</p>` : ""}
        <p class="ct-acciones"><button class="boton-primario" type="submit" ${edicion.calculando ? "disabled" : ""}>${e(t("ct_formalizacion_confirmar"))}</button>
          <button class="boton-secundario" type="button" data-ct-formalizacion-cancelar>${e(t("ct_formalizacion_cancelar"))}</button></p>
      </fieldset></form>`;
  }

  function cuerpo() {
    if (estado.carga === "cargando") return `<p role="status" aria-live="polite">${e(t("ct_formalizacion_cargando"))}</p>`;
    if (estado.carga === "error") {
      return `<p class="ct-exp-mensaje ct-tono-aviso" role="alert" data-ct-formalizacion-error>${e(t(estado.error))}</p>
        <p class="ct-acciones"><button class="boton-secundario" type="button" data-ct-formalizacion-reintentar>${e(t("ct_formalizacion_reintentar"))}</button></p>`;
    }
    return `${plazos()}${aviso()}
      ${estado.anotadosError ? `<p class="ct-exp-mensaje ct-tono-aviso" role="status">${e(t("ct_formalizacion_documentos_no_disponibles"))}</p>` : ""}
      ${estado.mensaje ? `<p class="ct-exp-mensaje ct-tono-${e(estado.tono)}" role="status" data-ct-formalizacion-mensaje>${e(estado.mensaje)}</p>` : ""}
      <div class="tabla-contenedor"><table class="tabla-datos"><caption>${e(t("ct_formalizacion_tabla"))}</caption>
        <thead><tr><th scope="col">${e(t("ct_formalizacion_columna_documento"))}</th><th scope="col">${e(t("ct_formalizacion_columna_estado"))}</th>
          <th scope="col">${e(t("ct_formalizacion_columna_anotacion"))}</th></tr></thead>
        <tbody>${estado.datos.documentos.map(fila).join("")}</tbody></table></div>
      ${formularioEdicion()}`;
  }

  function repintar(foco = "") {
    if (!activo || !contenedor || !estado) return;
    contenedor.innerHTML = `<section class="ct-llamamiento-paso" aria-labelledby="ct-formalizacion-titulo" data-ct-formalizacion
      aria-busy="${estado.carga === "cargando"}">
      <div class="ct-acciones"><h3 id="ct-formalizacion-titulo">${e(t("ct_formalizacion_titulo"))}</h3>
        <details data-ct-formalizacion-ayuda><summary aria-label="${e(t("ct_formalizacion_ayuda_abrir"))}"><span aria-hidden="true">?</span></summary>
          <p>${e(t("ct_formalizacion_ayuda"))}</p></details></div>
      ${cuerpo()}</section>`;
    if (foco) contenedor.querySelector?.(foco)?.focus?.();
  }

  function documentoEditable(clave) {
    return estado?.datos?.documentos.find((d) => d.clave === clave && d.registrable && !estado.anotados?.has(d.tipo_documental));
  }

  async function alCambiar(evento) {
    const control = evento.target;
    const edicion = estado?.edicion;
    if (!edicion || edicion.ocupado) return;
    if (control?.dataset?.ctFormalizacionReferencia !== undefined) {
      edicion.referencia = String(control.value ?? "");
      return;
    }
    if (control?.dataset?.ctFormalizacionArchivo === undefined || evento.type !== "change") return;
    const lectura = ++edicion.lectura;
    const archivo = control.files?.length === 1 ? control.files[0] : null;
    edicion.huella = "";
    edicion.calculando = true;
    edicion.mensaje = "";
    repintar();
    let bytes;
    try {
      // El fichero se lee solo para calcular su huella: ni su nombre ni su
      // contenido se muestran, se guardan ni se envían.
      if (!archivo || archivo.size < 1 || archivo.size > MAX_FICHERO || !criptografia?.subtle?.digest) throw new TypeError();
      bytes = new Uint8Array(await archivo.arrayBuffer());
      if (bytes.byteLength !== archivo.size) throw new TypeError();
      const digest = new Uint8Array(await criptografia.subtle.digest("SHA-256", bytes));
      const huella = Array.from(digest, (byte) => byte.toString(16).padStart(2, "0")).join("");
      if (digest.length !== 32 || huella === "0".repeat(64)) throw new TypeError();
      if (!activo || estado?.edicion !== edicion || lectura !== edicion.lectura) return;
      edicion.huella = huella;
    } catch {
      if (!activo || estado?.edicion !== edicion || lectura !== edicion.lectura) return;
      edicion.mensaje = "ct_formalizacion_huella_error";
      edicion.tono = "error";
    } finally {
      bytes?.fill(0);
      if (activo && estado?.edicion === edicion && lectura === edicion.lectura) {
        edicion.calculando = false;
        repintar("#ct-formalizacion-huella");
      }
    }
  }

  async function alEnviar(evento) {
    if (evento.target?.dataset?.ctFormalizacionForm === undefined) return;
    evento.preventDefault();
    const actual = estado;
    const edicion = actual?.edicion;
    if (!edicion || edicion.ocupado || edicion.calculando) return;
    const documento = documentoEditable(edicion.clave);
    const referencia = edicion.referencia.trim();
    if (!documento || !REFERENCIA.test(referencia) || REFERENCIA_PROHIBIDA.test(referencia) || !/^[0-9a-f]{64}$/u.test(edicion.huella)) {
      edicion.mensaje = "ct_formalizacion_error_validacion";
      edicion.tono = "error";
      repintar("[data-ct-formalizacion-edicion-mensaje]");
      return;
    }
    // La clave se fija en el primer intento y se conserva en los reintentos
    // del mismo documento: el servidor reconoce la anotación y no la duplica.
    if (!edicion.claveIdempotencia || edicion.referenciaEnviada !== referencia || edicion.huellaEnviada !== edicion.huella) {
      let clave = "";
      try { clave = `clave:${generarClaveIdempotencia() ?? ""}`; } catch { clave = ""; }
      if (!OPACA.test(clave)) {
        edicion.mensaje = "ct_formalizacion_error_registro";
        edicion.tono = "error";
        repintar();
        return;
      }
      edicion.claveIdempotencia = clave;
      edicion.referenciaEnviada = referencia;
      edicion.huellaEnviada = edicion.huella;
    }
    edicion.ocupado = true;
    edicion.mensaje = "ct_formalizacion_registrando";
    edicion.tono = "informacion";
    repintar();
    try {
      const anotacion = await fuente.registrar({ expedienteRef: actual.datos.expediente_documental_ref, tipo: documento.tipo_documental,
        referencia, huella: edicion.huella, claveIdempotencia: edicion.claveIdempotencia, signal: actual.controlador.signal });
      if (!activo || estado !== actual) return;
      actual.anotados = new Map(actual.anotados ?? []);
      actual.anotados.set(documento.tipo_documental, anotacion);
      actual.edicion = null;
      actual.mensaje = t("ct_formalizacion_registrado", { numero: anotacion.numero_vec });
      actual.tono = "informacion";
      repintar(`[data-ct-formalizacion-documento="${documento.clave}"] th`);
      try { anunciar(actual.mensaje, "informacion"); } catch { /* La región viva del panel permanece. */ }
    } catch (error) {
      if (!activo || estado !== actual || actual.edicion !== edicion) return;
      edicion.ocupado = false;
      edicion.mensaje = error?.codigo === "conflicto" ? "ct_formalizacion_error_conflicto"
        : error?.codigo === "rechazado" ? "ct_formalizacion_error_rechazado"
          : error?.codigo === "denegado" ? "ct_formalizacion_error_denegado" : "ct_formalizacion_error_registro";
      edicion.tono = "error";
      repintar("[data-ct-formalizacion-edicion-mensaje]");
    }
  }

  function alPulsar(evento) {
    const objetivo = evento.target?.closest?.("[data-ct-formalizacion-anotar],[data-ct-formalizacion-cancelar],[data-ct-formalizacion-reintentar]");
    if (!objetivo || !contenedor?.contains?.(objetivo) || !estado) return;
    evento.preventDefault();
    if (objetivo.dataset.ctFormalizacionReintentar !== undefined) {
      if (estado.carga === "error") cargar();
      return;
    }
    if (objetivo.dataset.ctFormalizacionCancelar !== undefined) {
      const clave = estado.edicion?.clave;
      if (estado.edicion?.ocupado) return;
      estado.edicion = null;
      repintar(clave ? `[data-ct-formalizacion-anotar="${clave}"]` : "");
      return;
    }
    const clave = objetivo.dataset.ctFormalizacionAnotar;
    if (estado.edicion || !documentoEditable(clave)) return;
    estado.mensaje = "";
    estado.edicion = { clave, referencia: "", huella: "", calculando: false, ocupado: false, lectura: 0,
      mensaje: "", tono: "informacion", claveIdempotencia: "", referenciaEnviada: "", huellaEnviada: "" };
    repintar("#ct-formalizacion-edicion-titulo");
  }

  function enlazar(nuevo) {
    if (contenedor === nuevo) return;
    contenedor?.removeEventListener?.("click", alPulsar);
    contenedor?.removeEventListener?.("submit", alEnviar);
    contenedor?.removeEventListener?.("change", alCambiar);
    contenedor?.removeEventListener?.("input", alCambiar);
    contenedor = nuevo;
    contenedor?.addEventListener?.("click", alPulsar);
    contenedor?.addEventListener?.("submit", alEnviar);
    contenedor?.addEventListener?.("change", alCambiar);
    contenedor?.addEventListener?.("input", alCambiar);
  }

  return Object.freeze({
    /**
     * Pinta el panel en `nuevo` para la aceptación confirmada. Sin aceptación
     * válida no muestra nada ni consulta el servidor.
     */
    pintar(nuevo, { aceptadaEn, expedienteRef } = {}) {
      if (!activo) return;
      if (!nuevo || typeof aceptadaEn !== "string" || !INSTANTE.test(aceptadaEn) || !EXPEDIENTE_CT.test(expedienteRef ?? "")) {
        enlazar(nuevo ?? null);
        return;
      }
      enlazar(nuevo);
      if (estado?.aceptadaEn !== aceptadaEn || estado?.expedienteRef !== expedienteRef) {
        estado?.controlador.abort();
        estado = nuevoEstado(aceptadaEn, expedienteRef);
        cargar();
        return;
      }
      repintar();
    },
    desmontar() {
      if (!activo) return;
      activo = false;
      estado?.controlador.abort();
      enlazar(null);
    },
  });
}
