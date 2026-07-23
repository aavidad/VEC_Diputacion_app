import { LIMITES_ALTA_CONTRATACION } from "./contrato.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function atributoSeleccionado(actual, valor) {
  return actual === valor ? " selected" : "";
}

function atributoMarcado(marcado) {
  return marcado ? " checked" : "";
}

function mensajeError(t, codigo) {
  try {
    return t(`error_${codigo}`);
  } catch {
    return t("error_generico");
  }
}

function atributosAccesibles(estado, campo, descripcionesAdicionales = []) {
  const error = estado.errores[campo];
  const descritos = [
    `ct-${campo}-ayuda`,
    ...descripcionesAdicionales,
    ...(error ? [`ct-${campo}-error`] : []),
  ];
  return `aria-describedby="${descritos.join(" ")}"${error ? ' aria-invalid="true"' : ""}`;
}

function errorCampo(estado, campo, t) {
  const codigo = estado.errores[campo];
  return codigo
    ? `<span class="ct-error-campo" id="ct-${campo}-error">${escaparHTML(mensajeError(t, codigo))}</span>`
    : "";
}

function ayudaCampo(campo, texto) {
  return `<small id="ct-${campo}-ayuda">${escaparHTML(texto)}</small>`;
}

function opcionesReferencia(opciones, seleccion, t) {
  return [
    `<option value="">${escaparHTML(t("seleccionar"))}</option>`,
    ...opciones.map((opcion) => `<option value="${escaparHTML(opcion.referencia)}"`
      + `${atributoSeleccionado(seleccion, opcion.referencia)}>`
      + `${escaparHTML(opcion.etiqueta)}</option>`),
  ].join("");
}

function opcionesClave(opciones, seleccion, t) {
  return [
    `<option value="">${escaparHTML(t("seleccionar"))}</option>`,
    ...opciones.map((opcion) => `<option value="${escaparHTML(opcion.clave)}"`
      + `${atributoSeleccionado(seleccion, opcion.clave)}>`
      + `${escaparHTML(opcion.etiqueta)}</option>`),
  ].join("");
}

function obtenerCentro(estado) {
  return estado.catalogos.centros.find(
    (opcion) => opcion.referencia === estado.borrador.centro_ref,
  );
}

function obtenerCategoria(estado) {
  return estado.catalogos.categorias.find(
    (opcion) => opcion.referencia === estado.borrador.categoria_ref,
  );
}

function resumenErrores(estado, t) {
  const entradas = Object.entries(estado.errores);
  if (entradas.length === 0) return "";
  return `<section class="ct-resumen-errores" data-ct-error-general role="alert"
    aria-live="assertive" aria-atomic="true" tabindex="-1">
    <h3>${escaparHTML(t("errores_titulo"))}</h3>
    <p>${escaparHTML(t("errores_descripcion"))}</p>
    <ul>${entradas.map(([campo, codigo]) => {
    const etiqueta = campo === "general"
      ? t("errores_titulo")
      : (Object.hasOwn(estado.borrador, campo) ? t(campo) : t("errores_titulo"));
    const contenido = `${escaparHTML(etiqueta)}: ${escaparHTML(mensajeError(t, codigo))}`;
    return campo === "general"
      ? `<li>${contenido}</li>`
      : `<li><button type="button" data-ct-enfocar="${escaparHTML(campo)}">`
        + `${contenido}</button></li>`;
  }).join("")}</ul>
  </section>`;
}

function pasos(estado, t) {
  const actual = estado.fase === "recibo"
    ? "recibo"
    : (estado.fase === "edicion" ? "datos" : "revision");
  return `<ol class="ct-pasos" aria-label="${escaparHTML(t("progreso_etiqueta"))}">
    ${["datos", "revision", "recibo"].map((paso, indice) => `<li`
    + `${paso === actual ? ' aria-current="step"' : ""}`
    + `${paso === "datos" || (paso === "revision" && actual !== "datos") || actual === "recibo"
      ? ' data-completado="true"' : ""}>`
    + `<span aria-hidden="true">${indice + 1}</span>${escaparHTML(t(`progreso_${paso}`))}</li>`).join("")}
  </ol>`;
}

function cabecera(estado, t) {
  return `<header class="ct-cabecera">
    <div>
      <p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p>
      <h2 id="ct-alta-titulo">${escaparHTML(t("titulo"))}</h2>
      <p>${escaparHTML(t("descripcion"))}</p>
    </div>
    <aside class="ct-alcance" aria-label="${escaparHTML(t("sobrelinea"))}">
      ${escaparHTML(t("alcance"))}
    </aside>
  </header>
  ${pasos(estado, t)}
  <div class="ct-estado ct-estado-${escaparHTML(estado.tipo_mensaje)}"
    data-ct-estado role="status" aria-live="polite" aria-atomic="true" tabindex="-1">
    <strong>${escaparHTML(t(estado.mensaje_clave))}</strong>
    ${estado.disponible ? "" : `<span>${escaparHTML(t("estado_no_disponible_detalle"))}</span>`}
  </div>`;
}

function campoSeleccion({
  estado, t, campo, etiqueta, ayuda, opciones, deshabilitado,
}) {
  return `<div class="ct-campo">
    <label for="ct-${campo}">${escaparHTML(etiqueta)} <b aria-hidden="true">*</b></label>
    <select id="ct-${campo}" name="${campo}" required
      ${atributosAccesibles(estado, campo)}${deshabilitado ? " disabled" : ""}>
      ${opciones}
    </select>
    ${ayudaCampo(campo, ayuda)}
    ${errorCampo(estado, campo, t)}
  </div>`;
}

function camposCentro(estado, t, deshabilitado) {
  const centro = obtenerCentro(estado);
  const categoria = obtenerCategoria(estado);
  return `<fieldset class="ct-bloque">
    <legend>${escaparHTML(t("centro_leyenda"))}</legend>
    <div class="ct-campos">
      ${campoSeleccion({
    estado,
    t,
    campo: "centro_ref",
    etiqueta: t("centro_ref"),
    ayuda: t("centro_ayuda"),
    opciones: opcionesReferencia(estado.catalogos.centros, estado.borrador.centro_ref, t),
    deshabilitado,
  })}
      ${campoSeleccion({
    estado,
    t,
    campo: "contacto_ref",
    etiqueta: t("contacto_ref"),
    ayuda: t("contacto_ayuda"),
    opciones: opcionesReferencia(centro?.contactos ?? [], estado.borrador.contacto_ref, t),
    deshabilitado: deshabilitado || !centro,
  })}
      ${campoSeleccion({
    estado,
    t,
    campo: "categoria_ref",
    etiqueta: t("categoria_ref"),
    ayuda: t("categoria_ayuda"),
    opciones: opcionesReferencia(
      estado.catalogos.categorias,
      estado.borrador.categoria_ref,
      t,
    ),
    deshabilitado,
  })}
      ${campoSeleccion({
    estado,
    t,
    campo: "grupo_subgrupo",
    etiqueta: t("grupo_subgrupo"),
    ayuda: t("grupo_ayuda"),
    opciones: opcionesClave(
      categoria?.grupos_subgrupos ?? [],
      estado.borrador.grupo_subgrupo,
      t,
    ),
    deshabilitado: deshabilitado || !categoria,
  })}
      ${campoSeleccion({
    estado,
    t,
    campo: "motivo_clave",
    etiqueta: t("motivo_clave"),
    ayuda: t("motivo_ayuda"),
    opciones: opcionesClave(estado.catalogos.motivos, estado.borrador.motivo_clave, t),
    deshabilitado,
  })}
    </div>
  </fieldset>`;
}

function camposDetalle(estado, t, deshabilitado) {
  const maximo = LIMITES_ALTA_CONTRATACION.texto;
  return `<fieldset class="ct-bloque">
    <legend>${escaparHTML(t("detalle_periodo_leyenda"))}</legend>
    <div class="ct-campos">
      <div class="ct-campo ct-campo-ancho">
        <label for="ct-detalle">${escaparHTML(t("detalle"))} <b aria-hidden="true">*</b></label>
        <textarea id="ct-detalle" name="detalle" required rows="5"
          ${atributosAccesibles(estado, "detalle", ["ct-detalle-contador"])}`
    + `${deshabilitado ? " disabled" : ""}>${escaparHTML(estado.borrador.detalle)}</textarea>
        ${ayudaCampo("detalle", t("detalle_ayuda"))}
        <small class="ct-contador" id="ct-detalle-contador" data-ct-contador="detalle">`
    + `${escaparHTML(t("contador_caracteres", {
      actual: [...estado.borrador.detalle].length,
      maximo,
        }))}</small>
        ${errorCampo(estado, "detalle", t)}
      </div>
      <div class="ct-campo">
        <label for="ct-inicio">${escaparHTML(t("inicio"))} <b aria-hidden="true">*</b></label>
        <input id="ct-inicio" name="inicio" type="date" required
          value="${escaparHTML(estado.borrador.inicio)}"
          ${atributosAccesibles(estado, "inicio")}${deshabilitado ? " disabled" : ""}>
        ${ayudaCampo("inicio", t("inicio"))}
        ${errorCampo(estado, "inicio", t)}
      </div>
      <div class="ct-campo">
        <label for="ct-fin">${escaparHTML(t("fin"))} <b aria-hidden="true">*</b></label>
        <input id="ct-fin" name="fin" type="date" required
          value="${escaparHTML(estado.borrador.fin)}"
          ${atributosAccesibles(estado, "fin")}${deshabilitado ? " disabled" : ""}>
        ${ayudaCampo("fin", t("fin"))}
        ${errorCampo(estado, "fin", t)}
      </div>
      <div class="ct-campo ct-campo-ancho">
        <label for="ct-observaciones">${escaparHTML(t("observaciones"))}</label>
        <textarea id="ct-observaciones" name="observaciones" rows="3"
          ${atributosAccesibles(
    estado,
    "observaciones",
    ["ct-observaciones-contador"],
  )}`
    + `${deshabilitado ? " disabled" : ""}>${escaparHTML(estado.borrador.observaciones)}</textarea>
        ${ayudaCampo("observaciones", t("observaciones_ayuda"))}
        <small class="ct-contador" id="ct-observaciones-contador"
          data-ct-contador="observaciones">`
    + `${escaparHTML(t("contador_caracteres", {
      actual: [...estado.borrador.observaciones].length,
      maximo,
        }))}</small>
        ${errorCampo(estado, "observaciones", t)}
      </div>
    </div>
  </fieldset>`;
}

function camposRC(estado, t, deshabilitado) {
  const activa = estado.borrador.rc_existe;
  const controlesDeshabilitados = deshabilitado || !activa;
  return `<fieldset class="ct-bloque">
    <legend>${escaparHTML(t("rc_leyenda"))}</legend>
    <p class="ct-aviso" id="ct-rc_existe-ayuda">${escaparHTML(t("rc_aviso"))}</p>
    <fieldset class="ct-radios" id="ct-rc_existe" tabindex="-1"
      ${atributosAccesibles(estado, "rc_existe")}>
      <legend>${escaparHTML(t("rc_existe"))} <b aria-hidden="true">*</b></legend>
      <label><input type="radio" name="rc_existe" value="si" required`
    + `${atributoMarcado(activa)}${deshabilitado ? " disabled" : ""}> ${escaparHTML(t("si"))}</label>
      <label><input type="radio" name="rc_existe" value="no" required`
    + `${atributoMarcado(!activa)}${deshabilitado ? " disabled" : ""}> ${escaparHTML(t("no"))}</label>
      ${errorCampo(estado, "rc_existe", t)}
    </fieldset>
    <div class="ct-campos" data-ct-datos-rc${activa ? "" : " hidden"}>
      <div class="ct-campo">
        <label for="ct-rc_numero">${escaparHTML(t("rc_numero"))} <b aria-hidden="true">*</b></label>
        <input id="ct-rc_numero" name="rc_numero" type="text" maxlength="160"
          value="${escaparHTML(estado.borrador.rc_numero)}" required
          ${atributosAccesibles(estado, "rc_numero")}`
    + `${controlesDeshabilitados ? " disabled" : ""}>
        ${ayudaCampo("rc_numero", t("rc_numero"))}
        ${errorCampo(estado, "rc_numero", t)}
      </div>
      <div class="ct-campo">
        <label for="ct-rc_fecha">${escaparHTML(t("rc_fecha"))} <b aria-hidden="true">*</b></label>
        <input id="ct-rc_fecha" name="rc_fecha" type="date"
          value="${escaparHTML(estado.borrador.rc_fecha)}" required
          ${atributosAccesibles(estado, "rc_fecha")}`
    + `${controlesDeshabilitados ? " disabled" : ""}>
        ${ayudaCampo("rc_fecha", t("rc_fecha"))}
        ${errorCampo(estado, "rc_fecha", t)}
      </div>
      <div class="ct-campo">
        <label for="ct-rc_importe">${escaparHTML(t("rc_importe"))} <b aria-hidden="true">*</b></label>
        <input id="ct-rc_importe" name="rc_importe" type="text" inputmode="decimal"
          value="${escaparHTML(estado.borrador.rc_importe)}"
          placeholder="${escaparHTML(t("rc_importe_placeholder"))}" required
          ${atributosAccesibles(estado, "rc_importe")}`
    + `${controlesDeshabilitados ? " disabled" : ""}>
        ${ayudaCampo("rc_importe", t("rc_importe_ayuda"))}
        ${errorCampo(estado, "rc_importe", t)}
      </div>
      ${campoSeleccion({
    estado,
    t,
    campo: "rc_documento_ref",
    etiqueta: t("rc_documento_ref"),
    ayuda: t("documentos_ayuda"),
    opciones: opcionesReferencia(
      estado.catalogos.documentos,
      estado.borrador.rc_documento_ref,
      t,
    ),
    deshabilitado: controlesDeshabilitados,
  })}
    </div>
  </fieldset>`;
}

function camposDocumentos(estado, t, deshabilitado) {
  const opciones = estado.catalogos.documentos.length === 0
    ? `<p class="ct-vacio">${escaparHTML(t("documentos_vacios"))}</p>`
    : `<div class="ct-documentos">${estado.catalogos.documentos.map((documento) => `
      <label>
        <input type="checkbox" name="documentos_adjuntos"
          value="${escaparHTML(documento.referencia)}"`
      + `${atributoMarcado(estado.borrador.documentos_adjuntos.includes(documento.referencia))}`
      + `${deshabilitado ? " disabled" : ""}>
        <span>${escaparHTML(documento.etiqueta)}</span>
        <code>${escaparHTML(documento.referencia)}</code>
      </label>`).join("")}</div>`;
  return `<fieldset class="ct-bloque" id="ct-documentos_adjuntos" tabindex="-1"
    ${atributosAccesibles(estado, "documentos_adjuntos")}>
    <legend>${escaparHTML(t("documentos_leyenda"))}</legend>
    <p id="ct-documentos_adjuntos-ayuda">${escaparHTML(t("documentos_ayuda"))}</p>
    ${opciones}
    ${errorCampo(estado, "documentos_adjuntos", t)}
  </fieldset>`;
}

function formulario(estado, t) {
  const deshabilitado = !estado.disponible || estado.ocupado;
  return `${resumenErrores(estado, t)}
  <form class="ct-formulario" data-ct-form novalidate>
    <p class="ct-obligatorios">${escaparHTML(t("campos_obligatorios"))}</p>
    ${camposCentro(estado, t, deshabilitado)}
    ${camposDetalle(estado, t, deshabilitado)}
    ${camposRC(estado, t, deshabilitado)}
    ${camposDocumentos(estado, t, deshabilitado)}
    <div class="ct-acciones">
      <button class="boton-primario" type="submit"${deshabilitado ? " disabled" : ""}>
        ${escaparHTML(t("revisar"))}
      </button>
    </div>
  </form>`;
}

function etiquetaReferencia(opciones, referencia) {
  return opciones.find((opcion) => opcion.referencia === referencia)?.etiqueta ?? referencia;
}

function etiquetaClave(opciones, clave) {
  return opciones.find((opcion) => opcion.clave === clave)?.etiqueta ?? clave;
}

function filaResumen(etiqueta, valor) {
  return `<div><dt>${escaparHTML(etiqueta)}</dt><dd>${escaparHTML(valor)}</dd></div>`;
}

function formatearFechaCivil(valor, locale) {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "long",
    timeZone: "UTC",
  }).format(new Date(`${valor}T00:00:00Z`));
}

function formatearImporteEUR(valor, locale) {
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency: "EUR",
  }).format(valor.replace(",", "."));
}

function revision(estado, t, locale) {
  const borrador = estado.borrador;
  const centro = obtenerCentro(estado);
  const categoria = obtenerCategoria(estado);
  const contacto = etiquetaReferencia(centro?.contactos ?? [], borrador.contacto_ref);
  const grupo = etiquetaClave(categoria?.grupos_subgrupos ?? [], borrador.grupo_subgrupo);
  const motivo = etiquetaClave(estado.catalogos.motivos, borrador.motivo_clave);
  const rc = borrador.rc_existe
    ? `${t("resumen_rc_si")} · ${borrador.rc_numero}`
      + ` · ${formatearFechaCivil(borrador.rc_fecha, locale)}`
      + ` · ${formatearImporteEUR(borrador.rc_importe, locale)}`
      + ` · ${borrador.rc_documento_ref}`
    : t("resumen_rc_no");
  const documentos = borrador.documentos_adjuntos.length === 0
    ? `<p>${escaparHTML(t("resumen_sin_documentos"))}</p>`
    : `<ul>${borrador.documentos_adjuntos.map((referencia) => `<li>`
      + `${escaparHTML(etiquetaReferencia(estado.catalogos.documentos, referencia))} `
      + `<code>${escaparHTML(referencia)}</code></li>`).join("")}</ul>`;
  const ocupado = estado.ocupado;
  return `${resumenErrores(estado, t)}
  <section class="ct-revision" aria-labelledby="ct-revision-titulo"
    ${ocupado ? 'aria-busy="true"' : ""}>
    <p class="sobrelinea">${escaparHTML(t("revision_sobrelinea"))}</p>
    <h3 id="ct-revision-titulo" tabindex="-1">${escaparHTML(t("revision_titulo"))}</h3>
    <p class="ct-aviso">${escaparHTML(t("revision_aviso"))}</p>
    <dl class="ct-resumen">
      ${filaResumen(t("resumen_centro"), centro?.etiqueta ?? borrador.centro_ref)}
      ${filaResumen(t("resumen_contacto"), contacto)}
      ${filaResumen(t("resumen_categoria"), categoria?.etiqueta ?? borrador.categoria_ref)}
      ${filaResumen(t("resumen_grupo"), grupo)}
      ${filaResumen(t("resumen_motivo"), motivo)}
      ${filaResumen(t("resumen_detalle"), borrador.detalle)}
      ${filaResumen(
    t("resumen_periodo"),
    `${formatearFechaCivil(borrador.inicio, locale)} — `
      + `${formatearFechaCivil(borrador.fin, locale)}`,
  )}
      ${filaResumen(t("resumen_rc"), rc)}
      ${filaResumen(
    t("resumen_observaciones"),
    borrador.observaciones || t("resumen_sin_observaciones"),
  )}
    </dl>
    <section class="ct-resumen-documentos" aria-labelledby="ct-resumen-documentos">
      <h4 id="ct-resumen-documentos">${escaparHTML(t("resumen_documentos"))}</h4>
      ${documentos}
    </section>
    <div class="ct-acciones">
      ${ocupado
    ? `<button class="boton-secundario" type="button" data-ct-accion="cancelar">
          ${escaparHTML(t("cancelar_envio"))}</button>`
    : `<button class="boton-secundario" type="button" data-ct-accion="volver">
          ${escaparHTML(t("volver_editar"))}</button>
        <button class="boton-primario" type="button" data-ct-accion="confirmar">
          ${escaparHTML(estado.tipo_mensaje === "error" ? t("reintentar") : t("confirmar"))}
        </button>`}
    </div>
  </section>`;
}

function recibo(estado, t, locale, zonaHoraria) {
  const dato = estado.recibo;
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "long",
    timeStyle: "medium",
    timeZone: zonaHoraria,
  }).format(new Date(dato.confirmada_en));
  return `<section class="ct-recibo" data-ct-recibo role="status" aria-live="polite"
    aria-atomic="true" tabindex="-1" aria-labelledby="ct-recibo-titulo">
    <p class="sobrelinea">${escaparHTML(t("recibo_sobrelinea"))}</p>
    <h3 id="ct-recibo-titulo">${escaparHTML(t("recibo_titulo"))}</h3>
    <p>${escaparHTML(t("recibo_descripcion"))}</p>
    <dl>
      ${filaResumen(t("recibo_expediente_ref"), dato.expediente_ref)}
      ${filaResumen(t("recibo_numero_visible"), dato.numero_visible)}
      ${filaResumen(t("recibo_version"), dato.version)}
      ${filaResumen(t("recibo_ref"), dato.recibo_ref)}
      ${filaResumen(t("recibo_fecha"), fecha)}
    </dl>
  </section>`;
}

export function renderizarAltaContratacionTemporal(estado, {
  mensajes = {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
} = {}) {
  const t = crearTraductorContratacionTemporal(mensajes);
  const contenido = estado.fase === "edicion"
    ? formulario(estado, t)
    : (estado.fase === "recibo"
      ? recibo(estado, t, locale, zonaHoraria)
      : revision(estado, t, locale));
  return `<section class="ct-alta" data-modulo="contratacion-temporal"
    aria-labelledby="ct-alta-titulo">
    ${cabecera(estado, t)}
    ${contenido}
  </section>`;
}

function extraerBorrador(formularioDOM) {
  const datos = new FormData(formularioDOM);
  return {
    centro_ref: String(datos.get("centro_ref") ?? ""),
    contacto_ref: String(datos.get("contacto_ref") ?? ""),
    categoria_ref: String(datos.get("categoria_ref") ?? ""),
    grupo_subgrupo: String(datos.get("grupo_subgrupo") ?? ""),
    motivo_clave: String(datos.get("motivo_clave") ?? ""),
    detalle: String(datos.get("detalle") ?? ""),
    inicio: String(datos.get("inicio") ?? ""),
    fin: String(datos.get("fin") ?? ""),
    rc_existe: datos.get("rc_existe") === "si",
    rc_numero: String(datos.get("rc_numero") ?? ""),
    rc_fecha: String(datos.get("rc_fecha") ?? ""),
    rc_importe: String(datos.get("rc_importe") ?? ""),
    rc_documento_ref: String(datos.get("rc_documento_ref") ?? ""),
    documentos_adjuntos: datos.getAll("documentos_adjuntos").map(String),
    observaciones: String(datos.get("observaciones") ?? ""),
  };
}

function enfocarVisible(elemento) {
  elemento?.focus?.();
  elemento?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
}

export function montarAltaContratacionTemporal({
  raiz,
  presentador,
  mensajes = {},
  anunciar = () => {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.querySelector !== "function"
    || typeof presentador?.obtenerEstado !== "function"
    || typeof anunciar !== "function") {
    throw new TypeError("dependencias DOM del alta no válidas");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  let montada = true;

  function repintar(selectorFoco = "") {
    if (!montada) return;
    const estado = presentador.obtenerEstado();
    raiz.innerHTML = renderizarAltaContratacionTemporal(estado, {
      mensajes,
      locale,
      zonaHoraria,
    });
    if (selectorFoco) enfocarVisible(raiz.querySelector(selectorFoco));
    anunciar(t(estado.mensaje_clave), estado.tipo_mensaje);
  }

  function enfocarTrasValidacion() {
    const estado = presentador.obtenerEstado();
    if (Object.keys(estado.errores).length > 0) {
      repintar("[data-ct-error-general]");
      return;
    }
    repintar("#ct-revision-titulo");
  }

  async function alPulsar(evento) {
    const enfocar = evento.target?.closest?.("[data-ct-enfocar]");
    if (enfocar && raiz.contains(enfocar)) {
      evento.preventDefault();
      const campo = enfocar.dataset.ctEnfocar;
      if (Object.hasOwn(presentador.obtenerEstado().borrador, campo)) {
        enfocarVisible(raiz.querySelector(`#ct-${campo}`));
      }
      return;
    }
    const control = evento.target?.closest?.("[data-ct-accion]");
    if (!control || !raiz.contains(control)) return;
    evento.preventDefault();
    if (control.dataset.ctAccion === "volver") {
      presentador.volverAEdicion();
      repintar("#ct-centro_ref");
      return;
    }
    if (control.dataset.ctAccion === "cancelar") {
      presentador.cancelarEnvio();
      repintar("[data-ct-estado]");
      return;
    }
    if (control.dataset.ctAccion === "confirmar") {
      const tarea = presentador.enviar();
      repintar("[data-ct-accion='cancelar']");
      await tarea;
      const estado = presentador.obtenerEstado();
      repintar(estado.fase === "recibo" ? "[data-ct-recibo]" : "[data-ct-accion='confirmar']");
    }
  }

  function alEnviar(evento) {
    const formularioDOM = evento.target?.closest?.("[data-ct-form]");
    if (!formularioDOM || !raiz.contains(formularioDOM)) return;
    evento.preventDefault();
    presentador.prepararRevision(extraerBorrador(formularioDOM));
    enfocarTrasValidacion();
  }

  function alCambiar(evento) {
    const campo = evento.target?.name;
    if (!["centro_ref", "categoria_ref", "rc_existe"].includes(campo)) return;
    const formularioDOM = evento.target.closest?.("[data-ct-form]");
    if (!formularioDOM || !raiz.contains(formularioDOM)) return;
    const borrador = extraerBorrador(formularioDOM);
    presentador.actualizarBorrador(borrador);
    const selectorFoco = campo === "rc_existe"
      ? `[name="rc_existe"][value="${borrador.rc_existe ? "si" : "no"}"]`
      : `#ct-${campo}`;
    repintar(selectorFoco);
  }

  function alIntroducir(evento) {
    const campo = evento.target?.name;
    if (!["detalle", "observaciones"].includes(campo)) return;
    const contador = raiz.querySelector(`[data-ct-contador="${campo}"]`);
    if (contador) {
      contador.textContent = t("contador_caracteres", {
        actual: [...evento.target.value].length,
        maximo: LIMITES_ALTA_CONTRATACION.texto,
      });
    }
  }

  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("change", alCambiar);
  raiz.addEventListener("input", alIntroducir);
  repintar();

  return () => {
    montada = false;
    raiz.removeEventListener("click", alPulsar);
    raiz.removeEventListener("submit", alEnviar);
    raiz.removeEventListener("change", alCambiar);
    raiz.removeEventListener("input", alIntroducir);
    presentador.desmontar?.();
  };
}
