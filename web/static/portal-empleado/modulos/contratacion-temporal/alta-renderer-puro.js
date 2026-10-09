/** Presentación pura del alta CT, compartida con la petición previa del centro. */
import { jornadaVisibleDesdeMinutos, LIMITES_ALTA_CONTRATACION, minutosDesdeJornadaVisible } from "./contrato.js?v=20261009-centro-campos-v1";
import { ESQUEMA_CATALOGOS_NECESIDADES } from "./contrato.js?v=20261009-centro-campos-v1";

const CAMPOS_RPT_PUBLICACION = new Set(["rpt_catalogo_ref", "rpt_catalogo_huella_sha256"]);
const CAMPOS_RPT_INTERNOS = new Set(["puesto_codigo", ...CAMPOS_RPT_PUBLICACION]);
function esNecesidad(estado) { return estado.catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES; }
function etiquetaMotivo(estado, opcion, t) {
  if (esNecesidad(estado)) return t(opcion.etiqueta);
  return opcion.clave === "sustitucion" ? t("motivo_sustitucion") : opcion.etiqueta;
}

export function escaparHTML(valor) {
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
  // Sin textos de ayuda bajo los campos: describen el control solo el contador y el error.
  const descritos = [
    ...descripcionesAdicionales,
    ...(error ? [`ct-${campo}-error`] : []),
  ];
  return `${descritos.length ? `aria-describedby="${descritos.join(" ")}"` : ""}${
    error ? ' aria-invalid="true"' : ""}`;
}

function errorCampo(estado, campo, t) {
  const codigo = estado.errores[campo];
  return codigo
    ? `<span class="ct-error-campo" id="ct-${campo}-error">${escaparHTML(mensajeError(t, codigo))}</span>`
    : "";
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
      : (CAMPOS_RPT_INTERNOS.has(campo) ? t("puesto_busqueda")
        : (Object.hasOwn(estado.borrador, campo) ? t(campo) : t("errores_titulo")));
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

export function cabecera(estado, t) {
  return `<header class="ct-cabecera">
    <div>
      <h2 id="ct-alta-titulo">${escaparHTML(t("titulo"))}</h2>
    </div>
    ${esNecesidad(estado) ? `<button type="button" class="boton-secundario" data-ct-accion="ayuda"
      aria-controls="ct-ayuda-necesidad" aria-expanded="false"
      aria-label="${escaparHTML(t("necesidad_ayuda_boton"))}">?</button>` : ""}
  </header>
  ${esNecesidad(estado) ? `<section id="ct-ayuda-necesidad" class="panel ayuda-contextual" data-ct-ayuda hidden>
    <div class="cuerpo-panel"><p>${escaparHTML(t("necesidad_ayuda"))}</p></div>
  </section>` : ""}
  ${pasos(estado, t)}
  <div class="ct-estado ct-estado-${escaparHTML(estado.tipo_mensaje)}"
    data-ct-estado role="status" aria-live="polite" aria-atomic="true" tabindex="-1">
    <strong>${escaparHTML(t(estado.mensaje_clave))}</strong>
    ${estado.disponible ? "" : `<span>${escaparHTML(t("estado_no_disponible_detalle"))}</span>`}
  </div>`;
}

function campoSeleccion({
  estado, t, campo, etiqueta, opciones, deshabilitado,
}) {
  return `<div class="ct-campo">
    <label for="ct-${campo}">${escaparHTML(etiqueta)} <b aria-hidden="true">*</b></label>
    <select id="ct-${campo}" name="${campo}" required
      ${atributosAccesibles(estado, campo)}${deshabilitado ? " disabled" : ""}>
      ${opciones}
    </select>
    ${errorCampo(estado, campo, t)}
  </div>`;
}

function campoNumeroMOAD(estado, t, deshabilitado) {
  if (!Object.hasOwn(estado.borrador, "numero_expediente_moad")) return "";
  const ejemplo = estado.catalogos.numero_expediente_moad?.ejemplo;
  return `<div class="ct-campo">
    <label for="ct-numero_expediente_moad">${escaparHTML(t("numero_expediente_moad"))} <b aria-hidden="true">*</b></label>
    <input id="ct-numero_expediente_moad" name="numero_expediente_moad" type="text" required
      maxlength="${LIMITES_ALTA_CONTRATACION.numeroExpediente}" autocomplete="off"
      value="${escaparHTML(estado.borrador.numero_expediente_moad)}"
      aria-label="${escaparHTML(t("numero_moad_obligatorio"))}"
      ${ejemplo ? `placeholder="${escaparHTML(t("numero_moad_ejemplo", { ejemplo }))}"` : ""}
      ${atributosAccesibles(estado, "numero_expediente_moad")}${deshabilitado ? " disabled" : ""}>
    ${errorCampo(estado, "numero_expediente_moad", t)}
  </div>`;
}

function camposCentro(estado, t, deshabilitado) {
  const centro = obtenerCentro(estado);
  const categoria = obtenerCategoria(estado);
  return `<fieldset class="ct-bloque">
    <legend>${escaparHTML(t("centro_leyenda"))}</legend>
    <div class="ct-campos">
      ${campoNumeroMOAD(estado, t, deshabilitado)}
      ${campoSeleccion({
    estado,
    t,
    campo: "centro_ref",
    etiqueta: t("centro_ref"),
    opciones: opcionesReferencia(estado.catalogos.centros, estado.borrador.centro_ref, t),
    deshabilitado,
  })}
      ${campoSeleccion({
    estado,
    t,
    campo: "contacto_ref",
    etiqueta: t("contacto_ref"),
    opciones: opcionesReferencia(centro?.contactos ?? [], estado.borrador.contacto_ref, t),
    deshabilitado: deshabilitado || !centro,
  })}
      ${campoSeleccion({
    estado,
    t,
    campo: "categoria_ref",
    etiqueta: t("categoria_ref"),
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
    opciones: opcionesClave(estado.catalogos.motivos.map((opcion) => ({
      ...opcion, etiqueta: etiquetaMotivo(estado, opcion, t),
    })), estado.borrador.motivo_clave, t),
    deshabilitado,
  })}
    </div>
  </fieldset>`;
}

function camposNecesidad(estado, t, deshabilitado) {
  if (!esNecesidad(estado)) return "";
  const causa = estado.catalogos.necesidades.causas.find(
    (dato) => dato.clave === estado.borrador.motivo_clave);
  const campos = causa?.campos_permitidos ?? [];
  const obligatorios = causa?.campos_obligatorios ?? [];
  return `<fieldset class="ct-bloque">
    <legend>${escaparHTML(t("necesidad_leyenda"))}</legend>
    <div class="ct-campos">
      <div class="ct-campo">
        <label for="ct-jornada_minutos">${escaparHTML(t("jornada_minutos"))} <b aria-hidden="true">*</b></label>
        <input id="ct-jornada_minutos" name="jornada_horas" type="text" inputmode="decimal" maxlength="8" required
          value="${escaparHTML(jornadaVisibleDesdeMinutos(estado.borrador.jornada_minutos) || estado.borrador.jornada_minutos)}"
          ${atributosAccesibles(estado, "jornada_minutos", ["ct-jornada-formato"])}${deshabilitado ? " disabled" : ""}>
        <small id="ct-jornada-formato">${escaparHTML(t("jornada_formato"))}</small>
        ${errorCampo(estado, "jornada_minutos", t)}
      </div>
      ${[...CAMPOS_RPT_INTERNOS].map((campo) => `<input type="hidden" name="${campo}" value="${escaparHTML(estado.borrador[campo] ?? "")}">`).join("")}
      ${campos.filter((campo) => !CAMPOS_RPT_PUBLICACION.has(campo)).map((campo) => {
    if (campo === "puesto_codigo") return `<div class="ct-campo ct-campo-ancho">
        <label for="ct-puesto_busqueda">${escaparHTML(t("puesto_busqueda"))}${obligatorios.includes(campo) ? ' <b aria-hidden="true">*</b>' : ""}</label>
        <input id="ct-puesto_busqueda" name="puesto_busqueda" type="search" maxlength="64"${obligatorios.includes(campo) ? " required" : ""}
          ${[...CAMPOS_RPT_INTERNOS].some((campo) => estado.errores[campo])
    ? 'aria-invalid="true" aria-describedby="ct-puesto_codigo-error"' : ""}
          value="${escaparHTML(estado.busquedaPuesto ?? "")}"${deshabilitado ? " disabled" : ""}>
        <button type="button" class="boton-secundario" data-ct-accion="buscar-puesto"${deshabilitado ? " disabled" : ""}>${escaparHTML(t("puesto_buscar"))}</button>
        <div data-ct-puesto-resultado role="status" aria-live="polite">${escaparHTML(t(estado.puestoRPT?.mensaje ?? "puesto_sin_seleccion"))}${estado.puestoRPT?.denominacion ? `: ${escaparHTML(estado.puestoRPT.codigo)} · ${escaparHTML(estado.puestoRPT.denominacion)}` : ""}</div>
        ${[...CAMPOS_RPT_INTERNOS].some((campo) => estado.errores[campo])
    ? `<span class="ct-error-campo" id="ct-puesto_codigo-error">${escaparHTML(t("error_puesto_publicacion"))}</span>` : ""}
      </div>`;
    const obligatorio = obligatorios.includes(campo);
    const largo = ["justificacion_temporal", "programa_denominacion"].includes(campo);
    return `<div class="ct-campo">
      <label for="ct-${campo}">${escaparHTML(t(campo === "plaza_codigo" && !obligatorio ? "plaza_codigo_opcional" : campo))}${obligatorio ? ' <b aria-hidden="true">*</b>' : ""}</label>
      ${largo ? `<textarea id="ct-${campo}" name="${campo}" maxlength="4000" rows="3"${obligatorio ? " required" : ""}
        ${atributosAccesibles(estado, campo)}${deshabilitado ? " disabled" : ""}>${escaparHTML(estado.borrador[campo])}</textarea>`
    : `<input id="ct-${campo}" name="${campo}" type="${campo === "programa_fin" ? "date" : ["numero_personas", "porcentaje_financiacion"].includes(campo) ? "number" : "text"}"${campo === "numero_personas" ? ' min="1" max="4294967295" step="1" inputmode="numeric"' : campo === "porcentaje_financiacion" ? ' min="1" max="100" step="1" inputmode="numeric"' : ' maxlength="160"'}${obligatorio ? " required" : ""}
        value="${escaparHTML(estado.borrador[campo])}"
        ${atributosAccesibles(estado, campo)}${deshabilitado ? " disabled" : ""}>`}
      ${errorCampo(estado, campo, t)}
    </div>`;
  }).join("")}
    </div>
  </fieldset>`;
}

function camposPeticionCentro(estado, t, deshabilitado) {
  if (!Object.hasOwn(estado.borrador, "puesto_solicitado")) return "";
  return `<fieldset class="ct-bloque">
    <legend>${escaparHTML(t("peticion_puesto_leyenda"))}</legend>
    <div class="ct-campos">
      <div class="ct-campo">
        <label for="ct-pc-numero-personas">${escaparHTML(t("numero_personas"))} <b aria-hidden="true">*</b></label>
        <input id="ct-pc-numero-personas" name="numero_personas" type="number" min="1" max="4294967295" step="1" required
          value="${escaparHTML(estado.borrador.numero_personas)}"
          ${atributosAccesibles(estado, "numero_personas")}${deshabilitado ? " disabled" : ""}>
        ${errorCampo(estado, "numero_personas", t)}
      </div>
      <div class="ct-campo">
        <label for="ct-pc-jornada">${escaparHTML(t("jornada_minutos"))} <b aria-hidden="true">*</b></label>
        <input id="ct-pc-jornada" name="jornada_horas" type="text" inputmode="decimal" maxlength="8" required
          value="${escaparHTML(jornadaVisibleDesdeMinutos(estado.borrador.jornada_minutos) || estado.borrador.jornada_minutos)}"
          ${atributosAccesibles(estado, "jornada_minutos")}${deshabilitado ? " disabled" : ""}>
        ${errorCampo(estado, "jornada_minutos", t)}
      </div>
      <div class="ct-campo ct-campo-ancho">
        <label for="ct-pc-puesto">${escaparHTML(t("puesto_solicitado"))} <b aria-hidden="true">*</b></label>
        <input id="ct-pc-puesto" name="puesto_solicitado" type="text" maxlength="160" required
          value="${escaparHTML(estado.borrador.puesto_solicitado)}"
          ${atributosAccesibles(estado, "puesto_solicitado")}${deshabilitado ? " disabled" : ""}>
        ${errorCampo(estado, "puesto_solicitado", t)}
      </div>
    </div>
  </fieldset>`;
}

function camposDetalle(estado, t, deshabilitado) {
  const maximo = LIMITES_ALTA_CONTRATACION.texto;
  const motivo = estado.catalogos.motivos.find(({ clave }) => clave === estado.borrador.motivo_clave);
  const reglaFin = motivo?.fecha_fin ?? "obligatoria";
  const causaFin = !estado.borrador.fin && motivo?.causa_fin
    ? t(`causa_fin_${motivo.causa_fin}`) : "";
  return `<fieldset class="ct-bloque">
    <legend>${escaparHTML(t("detalle_periodo_leyenda"))}</legend>
    <div class="ct-campos">
      <div class="ct-campo ct-campo-ancho">
        <label for="ct-detalle">${escaparHTML(t("detalle"))} <b aria-hidden="true">*</b></label>
        <textarea id="ct-detalle" name="detalle" required rows="5"
          ${atributosAccesibles(estado, "detalle", ["ct-detalle-contador"])}`
    + `${deshabilitado ? " disabled" : ""}>${escaparHTML(estado.borrador.detalle)}</textarea>
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
        ${errorCampo(estado, "inicio", t)}
      </div>
      <div class="ct-campo">
        ${reglaFin === "no_aplica" ? `<span>${escaparHTML(t("fin"))}</span>`
          : `<label for="ct-fin">${escaparHTML(t("fin"))}${reglaFin === "obligatoria" ? ' <b aria-hidden="true">*</b>' : ""}</label>`}
        ${reglaFin === "no_aplica" ? "" : `<input id="ct-fin" name="fin" type="date"${reglaFin === "obligatoria" ? " required" : ""}
          value="${escaparHTML(estado.borrador.fin)}"
          ${atributosAccesibles(estado, "fin")}${deshabilitado ? " disabled" : ""}>`}
        ${causaFin ? `<p class="ct-aviso-campo" role="status">${escaparHTML(causaFin)}</p>` : ""}
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
        ${errorCampo(estado, "rc_numero", t)}
      </div>
      <div class="ct-campo">
        <label for="ct-rc_fecha">${escaparHTML(t("rc_fecha"))} <b aria-hidden="true">*</b></label>
        <input id="ct-rc_fecha" name="rc_fecha" type="date"
          value="${escaparHTML(estado.borrador.rc_fecha)}" required
          ${atributosAccesibles(estado, "rc_fecha")}`
    + `${controlesDeshabilitados ? " disabled" : ""}>
        ${errorCampo(estado, "rc_fecha", t)}
      </div>
      <div class="ct-campo">
        <label for="ct-rc_importe">${escaparHTML(t("rc_importe"))} <b aria-hidden="true">*</b></label>
        <input id="ct-rc_importe" name="rc_importe" type="text" inputmode="decimal"
          value="${escaparHTML(estado.borrador.rc_importe)}"
          placeholder="${escaparHTML(t("rc_importe_placeholder"))}" required
          ${atributosAccesibles(estado, "rc_importe")}`
    + `${controlesDeshabilitados ? " disabled" : ""}>
        ${errorCampo(estado, "rc_importe", t)}
      </div>
      ${campoSeleccion({
    estado,
    t,
    campo: "rc_documento_ref",
    etiqueta: t("rc_documento_ref"),
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
      </label>`).join("")}</div>`;
  return `<fieldset class="ct-bloque" id="ct-documentos_adjuntos" tabindex="-1"
    ${atributosAccesibles(estado, "documentos_adjuntos")}>
    <legend>${escaparHTML(t("documentos_leyenda"))}</legend>
    ${opciones}
    ${errorCampo(estado, "documentos_adjuntos", t)}
  </fieldset>`;
}

export function formulario(estado, t) {
  const deshabilitado = !estado.disponible || estado.ocupado;
  return `${resumenErrores(estado, t)}
  <form class="ct-formulario" data-ct-form novalidate>
    ${camposCentro(estado, t, deshabilitado)}
    ${camposPeticionCentro(estado, t, deshabilitado)}
    ${camposDetalle(estado, t, deshabilitado)}
    ${camposNecesidad(estado, t, deshabilitado)}
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

export function filaResumen(etiqueta, valor) {
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

export function revision(estado, t, locale) {
  const borrador = estado.borrador;
  const centro = obtenerCentro(estado);
  const categoria = obtenerCategoria(estado);
  const contacto = etiquetaReferencia(centro?.contactos ?? [], borrador.contacto_ref);
  const grupo = etiquetaClave(categoria?.grupos_subgrupos ?? [], borrador.grupo_subgrupo);
  const motivo = etiquetaClave(estado.catalogos.motivos.map((opcion) => ({
    ...opcion, etiqueta: etiquetaMotivo(estado, opcion, t),
  })), borrador.motivo_clave);
  const rc = borrador.rc_existe
    ? `${t("resumen_rc_si")} · ${borrador.rc_numero}`
      + ` · ${formatearFechaCivil(borrador.rc_fecha, locale)}`
      + ` · ${formatearImporteEUR(borrador.rc_importe, locale)}`
      + ` · ${borrador.rc_documento_ref}`
    : t("resumen_rc_no");
  const documentos = borrador.documentos_adjuntos.length === 0
    ? `<p>${escaparHTML(t("resumen_sin_documentos"))}</p>`
    : `<ul>${borrador.documentos_adjuntos.map((referencia) => `<li>`
      + `${escaparHTML(etiquetaReferencia(estado.catalogos.documentos, referencia))}</li>`).join("")}</ul>`;
  const ocupado = estado.ocupado;
  const pendiente = estado.fase === "pendiente";
  const corregirNumeroMOAD = Boolean(estado.errores.numero_expediente_moad);
  return `${resumenErrores(estado, t)}
  <section class="ct-revision" aria-labelledby="ct-revision-titulo"
    ${ocupado ? 'aria-busy="true"' : ""}>
    <p class="sobrelinea">${escaparHTML(t("revision_sobrelinea"))}</p>
    <h3 id="ct-revision-titulo" tabindex="-1">${escaparHTML(t("revision_titulo"))}</h3>
    <p class="ct-aviso">${escaparHTML(t("revision_aviso"))}</p>
    <dl class="ct-resumen">
      ${Object.hasOwn(borrador, "numero_expediente_moad") ? filaResumen(t("numero_expediente_moad"), borrador.numero_expediente_moad) : ""}
      ${filaResumen(t("resumen_centro"), centro?.etiqueta ?? borrador.centro_ref)}
      ${filaResumen(t("resumen_contacto"), contacto)}
      ${filaResumen(t("resumen_categoria"), categoria?.etiqueta ?? borrador.categoria_ref)}
      ${filaResumen(t("resumen_grupo"), grupo)}
      ${filaResumen(t("resumen_motivo"), motivo)}
      ${Object.hasOwn(borrador, "puesto_solicitado") ? filaResumen(t("numero_personas"), borrador.numero_personas)
    + filaResumen(t("jornada_minutos"), jornadaVisibleDesdeMinutos(borrador.jornada_minutos))
    + filaResumen(t("puesto_solicitado"), borrador.puesto_solicitado) : ""}
      ${esNecesidad(estado) ? filaResumen(t("jornada_minutos"), jornadaVisibleDesdeMinutos(borrador.jornada_minutos)) : ""}
      ${esNecesidad(estado) ? estado.catalogos.necesidades.causas.find(
    (dato) => dato.clave === borrador.motivo_clave)?.campos_permitidos
    .filter((campo) => borrador[campo] && !CAMPOS_RPT_PUBLICACION.has(campo))
    .map((campo) => filaResumen(t(campo), borrador[campo])).join("") : ""}
      ${filaResumen(t("resumen_detalle"), borrador.detalle)}
      ${filaResumen(
    t("resumen_periodo"),
    `${formatearFechaCivil(borrador.inicio, locale)} — `
      + (borrador.fin ? formatearFechaCivil(borrador.fin, locale)
        : t(`causa_fin_${estado.catalogos.motivos.find(({ clave }) => clave === borrador.motivo_clave).causa_fin}`)),
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
      ${pendiente
    ? `<p class="ct-aviso" data-ct-operacion-pendiente role="status">
          ${escaparHTML(t("estado_operacion_pendiente_ayuda"))}</p>`
    : ocupado
    ? `<button class="boton-secundario" type="button" data-ct-accion="cancelar">
          ${escaparHTML(t("cancelar_envio"))}</button>`
    : `<button class="boton-secundario" type="button" data-ct-accion="volver">
          ${escaparHTML(t("volver_editar"))}</button>${corregirNumeroMOAD ? "" : `
        <button class="boton-primario" type="button" data-ct-accion="confirmar">
          ${escaparHTML(estado.tipo_mensaje === "error" ? t("reintentar") : t("confirmar"))}
        </button>`}`}
    </div>
  </section>`;
}

export function extraerBorrador(formularioDOM, conNumeroMOAD = true) {
  const datos = new FormData(formularioDOM);
  const peticionCentro = Boolean(formularioDOM.querySelector?.('[name="puesto_solicitado"]'));
  const necesidad = !peticionCentro && Boolean(formularioDOM.querySelector?.('[name="jornada_horas"]'));
  const jornadaEntrada = String(datos.get("jornada_horas") ?? "");
  const minutosJornada = minutosDesdeJornadaVisible(jornadaEntrada);
  const adicionales = necesidad ? {
    jornada_minutos: minutosJornada === null ? jornadaEntrada : String(minutosJornada),
    ...Object.fromEntries([
      "numero_personas", "puesto_codigo", "plaza_codigo", "titular_ref", "vacancia_fuente_ref",
      "rpt_catalogo_ref", "rpt_catalogo_huella_sha256", "organica_codigo", "funcional_codigo",
      "proyecto_gasto_codigo", "porcentaje_financiacion", "justificacion_temporal",
      "programa_denominacion", "programa_fin", "proyecto_codigo", "financiacion_ref", "rc_ref",
      "intervencion_ref",
    ].map((campo) =>
      [campo, String(datos.get(campo) ?? "")])) } : {};
  const adicionalesCentro = peticionCentro ? {
    jornada_minutos: minutosJornada === null ? jornadaEntrada : String(minutosJornada),
    numero_personas: String(datos.get("numero_personas") ?? ""),
    puesto_solicitado: String(datos.get("puesto_solicitado") ?? ""),
  } : {};
  return {
    ...(conNumeroMOAD ? { numero_expediente_moad: String(datos.get("numero_expediente_moad") ?? "") } : {}),
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
    ...adicionalesCentro,
    ...adicionales,
  };
}
