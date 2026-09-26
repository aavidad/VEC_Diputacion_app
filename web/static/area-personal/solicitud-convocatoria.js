/**
 * Asistente de solicitud de participación en una convocatoria (área personal).
 *
 * Cuatro pasos: requisitos, datos personales, méritos y revisión. Cada paso
 * guarda el borrador en el servidor con su versión, de modo que se puede
 * retomar desde cualquier dispositivo; nada se guarda en el navegador. El
 * servidor decide plazo, requisitos y puntuación: la vista solo recoge lo que
 * la persona declara y muestra un autobaremo orientativo con el baremo
 * publicado. Firma, registro en sede, tasas y notificación no se ofrecen.
 */
import { crearTraductorSolicitud, textoErrorSolicitud } from "./i18n-solicitud.js?v=20260926-convoca-f1-v1";
import { calcularAutobaremoOrientativo, leerReglasConvocatoria } from "./reglas-convocatoria.js?v=20260926-convoca-f1-v1";
import { botonCopiarReferencia } from "./justificante-copiable.js?v=20260926-convoca-f1-v1";
import { escaparHTML as escapar } from "./vistas/comunes.js";

const TOTAL_PASOS = 4;
const CLAVES_PASO = Object.freeze(["paso_requisitos", "paso_datos", "paso_meritos", "paso_revision"]);
const AYUDA_PASO = Object.freeze(["ayuda_requisitos", "ayuda_requisitos", "ayuda_meritos", "ayuda_revision"]);
const DECLARACIONES = Object.freeze(["cumple", "no_cumple", "pendiente"]);
const CAMPOS_DATOS = Object.freeze([
  ["nombre", "text", "given-name", 80], ["apellidos", "text", "family-name", 120],
  ["documento_identidad", "text", "off", 20], ["fecha_nacimiento", "date", "bday", 10],
  ["nacionalidad", "text", "country-name", 80], ["correo", "email", "email", 160],
  ["telefono", "tel", "tel", 20],
]);
const CAMPOS_DIRECCION = Object.freeze([
  ["via", "street-address", 200], ["codigo_postal", "postal-code", 10],
  ["municipio", "address-level2", 100], ["provincia", "address-level1", 100],
]);
const TRAMITES_NO_DISPONIBLES = Object.freeze(["firma", "registro_sede", "tasas", "notificacion"]);
const PATRON_CANTIDAD = /^\d{1,9}(?:\.\d{1,3})?$/u;
/** Cantidad escrita por la persona como cadena decimal del contrato, o "" si no vale. */
export function cantidadContrato(valor) {
  const normalizada = String(valor ?? "").trim().replace(",", ".");
  return PATRON_CANTIDAD.test(normalizada) && Number(normalizada) > 0 ? normalizada.replace(/^0+(?=\d)/u, "") : "";
}

const formatoNumero = new Intl.NumberFormat("es-ES", { maximumFractionDigits: 2 });
const formatoFecha = new Intl.DateTimeFormat("es-ES", { dateStyle: "long", timeStyle: "short", timeZone: "Europe/Madrid" });
const fecha = (valor) => { const instante = new Date(valor); return Number.isNaN(instante.getTime()) ? "" : formatoFecha.format(instante); };
const claveMerito = (grupo, merito) => `${grupo}|${merito}`;

export function formularioVacio() {
  return {
    requisitos: {}, turno: "", meritos: {},
    datos: { ...Object.fromEntries(CAMPOS_DATOS.map(([campo]) => [campo, ""])), direccion: Object.fromEntries(CAMPOS_DIRECCION.map(([campo]) => [campo, ""])) },
  };
}

/** Rellena el formulario con el borrador devuelto por el servidor. */
export function formularioDesdeBorrador(borrador) {
  const formulario = formularioVacio();
  const datos = borrador?.datos && typeof borrador.datos === "object" ? borrador.datos : {};
  for (const [campo] of CAMPOS_DATOS) if (typeof datos[campo] === "string") formulario.datos[campo] = datos[campo];
  for (const [campo] of CAMPOS_DIRECCION) if (typeof datos.direccion?.[campo] === "string") formulario.datos.direccion[campo] = datos.direccion[campo];
  for (const requisito of Array.isArray(borrador?.requisitos) ? borrador.requisitos : []) {
    if (typeof requisito?.clave === "string" && DECLARACIONES.includes(requisito.estado)) formulario.requisitos[requisito.clave] = requisito.estado;
  }
  for (const merito of Array.isArray(borrador?.meritos) ? borrador.meritos : []) {
    if (typeof merito?.clave_grupo !== "string" || typeof merito?.clave_merito !== "string") continue;
    formulario.meritos[claveMerito(merito.clave_grupo, merito.clave_merito)] = {
      cantidad: typeof merito.cantidad === "string" ? merito.cantidad : "",
      descripcion: typeof merito.descripcion === "string" ? merito.descripcion : "",
    };
  }
  if (typeof borrador?.turno === "string") formulario.turno = borrador.turno;
  return formulario;
}

function cantidades(formulario) {
  return new Map(Object.entries(formulario.meritos).map(([clave, valor]) => [clave, Number(cantidadContrato(valor?.cantidad) || 0)]));
}

/** Cuerpo del PUT del borrador según el contrato; solo lleva lo declarado. */
export function cuerpoBorrador({ convocatoriaRef, reglas, formulario, versionEsperada }) {
  const datos = {};
  for (const [campo] of CAMPOS_DATOS) if (formulario.datos[campo]?.trim()) datos[campo] = formulario.datos[campo].trim();
  const direccion = {};
  for (const [campo] of CAMPOS_DIRECCION) if (formulario.datos.direccion[campo]?.trim()) direccion[campo] = formulario.datos.direccion[campo].trim();
  if (Object.keys(direccion).length) datos.direccion = direccion;
  const meritos = [];
  for (const grupo of reglas.baremo?.grupos || []) {
    for (const merito of grupo.meritos) {
      const clave = claveMerito(grupo.clave, merito.clave);
      const cantidad = cantidadContrato(formulario.meritos[clave]?.cantidad);
      if (!cantidad) continue;
      meritos.push({ clave_grupo: grupo.clave, clave_merito: merito.clave, descripcion: (formulario.meritos[clave]?.descripcion || "").trim().slice(0, 300), cantidad });
    }
  }
  return {
    convocatoria_ref: convocatoriaRef,
    version_esperada: versionEsperada,
    ...(reglas.turnos.length ? { turno: formulario.turno } : {}),
    datos,
    requisitos: reglas.requisitos.filter((requisito) => DECLARACIONES.includes(formulario.requisitos[requisito.clave]))
      .map((requisito) => ({ clave: requisito.clave, estado: formulario.requisitos[requisito.clave] })),
    meritos,
  };
}

function botonAyuda(t, clave, tema) {
  return `<button type="button" class="boton-icono boton-ayuda-solicitud" data-ayuda-solicitud="${escapar(clave)}" aria-haspopup="dialog" aria-label="${escapar(t("ayuda_boton", { tema }))}">?</button>`;
}

function pasos(t, actual) {
  return `<ol class="pasos" aria-label="${escapar(t("pasos_aria"))}">${CLAVES_PASO.map((clave, indice) => {
    const numero = indice + 1;
    const clase = numero === actual ? "activo" : numero < actual ? "completo" : "";
    return `<li class="${clase}"${numero === actual ? ' aria-current="step"' : ""}><span>${escapar(t("paso_de", { paso: numero, total: TOTAL_PASOS }))}</span>${escapar(t(clave))}</li>`;
  }).join("")}</ol>`;
}

function pasoRequisitos(t, estado) {
  const { requisitos, turnos } = estado.reglas;
  const lista = requisitos.length ? requisitos.map((requisito, indice) => `<fieldset class="requisito-solicitud"><legend>${escapar(requisito.titulo)}${requisito.obligatorio ? ` <span class="estado-chip aviso">${escapar(t("requisito_obligatorio"))}</span>` : ""}</legend>${requisito.descripcion ? `<p>${escapar(requisito.descripcion)}</p>` : ""}<div class="opciones-requisito">${DECLARACIONES.map((valor) => `<label class="opcion-check"><input type="radio" name="req-${indice}" value="${valor}" required${estado.formulario.requisitos[requisito.clave] === valor ? " checked" : ""}><span>${escapar(t(`requisito_${valor}`))}</span></label>`).join("")}</div></fieldset>`).join("")
    : `<p class="nota">${escapar(t("sin_requisitos"))}</p>`;
  const turno = turnos.length ? `<div class="campo"><label for="solicitud-turno">${escapar(t("turno"))}</label><select id="solicitud-turno" name="turno" required><option value="">${escapar(t("turno_elegir"))}</option>${turnos.map((item) => `<option value="${escapar(item.clave)}"${estado.formulario.turno === item.clave ? " selected" : ""}>${escapar(item.etiqueta)}</option>`).join("")}</select></div>` : "";
  return `<div class="grupo-requisitos" role="group" aria-label="${escapar(t("requisitos_leyenda"))}">${lista}</div>${turno}`;
}

function campo(t, nombre, tipo, autocompletar, maximo, valor, ancho = false) {
  return `<div class="campo${ancho ? " ancho-completo" : ""}"><label for="solicitud-${nombre}">${escapar(t(nombre))}</label><input id="solicitud-${nombre}" name="${nombre}" type="${tipo}" autocomplete="${autocompletar}" maxlength="${maximo}" value="${escapar(valor)}" required></div>`;
}

function pasoDatos(t, estado) {
  const datos = estado.formulario.datos;
  return `<div class="formulario-rejilla">${CAMPOS_DATOS.map(([nombre, tipo, auto, maximo]) => campo(t, nombre, tipo, auto, maximo, datos[nombre])).join("")}${CAMPOS_DIRECCION.map(([nombre, auto, maximo]) => campo(t, nombre, "text", auto, maximo, datos.direccion[nombre], nombre === "via")).join("")}</div>`;
}

function pasoMeritos(t, estado) {
  const baremo = estado.reglas.baremo;
  const documentos = `<dl class="dato-lista"><dt>${escapar(t("documentos"))}</dt><dd><span class="estado-chip info">${escapar(t("documentos_mas_adelante"))}</span></dd></dl>`;
  if (!baremo) return `<p class="nota">${escapar(t("sin_baremo"))}</p>${documentos}`;
  const calculo = calcularAutobaremoOrientativo(baremo, cantidades(estado.formulario));
  const grupos = baremo.grupos.map((grupo, gi) => `<fieldset class="grupo-meritos"><legend>${escapar(grupo.maximo === null ? grupo.titulo : t("grupo_maximo", { grupo: grupo.titulo, maximo: formatoNumero.format(grupo.maximo) }))}</legend><div class="formulario-rejilla">${grupo.meritos.map((merito, mi) => {
    const clave = claveMerito(grupo.clave, merito.clave);
    const valor = estado.formulario.meritos[clave] || {};
    const id = `solicitud-m-${gi}-${mi}`;
    return `<div class="campo"><label for="${id}">${escapar(t("cantidad_merito", { merito: merito.titulo, unidad: merito.unidad }))}</label><input id="${id}" name="m-${gi}-${mi}" type="text" inputmode="decimal" pattern="[0-9]{1,9}([.,][0-9]{1,3})?" maxlength="13" value="${escapar(valor.cantidad || "")}" aria-describedby="${id}-puntos"><small><output id="${id}-puntos" data-puntos-merito="${escapar(clave)}">${escapar(t("puntos_orientativos", { puntos: formatoNumero.format(calculo.porMerito.get(clave) || 0) }))}</output></small></div><div class="campo"><label for="${id}-d">${escapar(t("descripcion_merito", { merito: merito.titulo }))}</label><input id="${id}-d" name="d-${gi}-${mi}" type="text" maxlength="300" value="${escapar(valor.descripcion || "")}"></div>`;
  }).join("")}</div></fieldset>`).join("");
  return `${grupos}<p class="puntuacion-total"><span>${escapar(t("autobaremo_orientativo"))}</span><output data-total-autobaremo aria-live="polite">${escapar(t("autobaremo_total", { puntos: formatoNumero.format(calculo.total) }))}</output></p>${documentos}`;
}

/** Puntuación que calculó el servidor al guardar la última versión. */
function puntuacionServidor(t, estado) {
  if (estado.puntuacionServidor === null || estado.puntuacionServidor === undefined) return "";
  return `<dl class="dato-lista"><dt>${escapar(t("puntuacion_servicio"))}</dt><dd><strong>${escapar(t("autobaremo_total", { puntos: formatoNumero.format(Number(estado.puntuacionServidor)) }))}</strong></dd></dl>`;
}

function pasoRevision(t, estado) {
  const { reglas, formulario } = estado;
  const requisitos = reglas.requisitos.map((requisito) => `<li><strong>${escapar(requisito.titulo)}</strong> · ${escapar(DECLARACIONES.includes(formulario.requisitos[requisito.clave]) ? t(`requisito_${formulario.requisitos[requisito.clave]}`) : t("sin_dato"))}</li>`).join("");
  const datos = [...CAMPOS_DATOS.map(([nombre]) => [nombre, formulario.datos[nombre]]), ...CAMPOS_DIRECCION.map(([nombre]) => [nombre, formulario.datos.direccion[nombre]])]
    .map(([nombre, valor]) => `<dt>${escapar(t(nombre))}</dt><dd>${escapar(valor || t("sin_dato"))}</dd>`).join("");
  const cuerpo = cuerpoBorrador({ convocatoriaRef: estado.convocatoriaRef, reglas, formulario, versionEsperada: 0 });
  const publicados = new Map((reglas.baremo?.grupos || []).flatMap((grupo) => grupo.meritos.map((merito) => [claveMerito(grupo.clave, merito.clave), merito])));
  const calculo = calcularAutobaremoOrientativo(reglas.baremo, cantidades(formulario));
  const meritos = cuerpo.meritos.length ? `<ul>${cuerpo.meritos.map((merito) => {
    const clave = claveMerito(merito.clave_grupo, merito.clave_merito);
    return `<li>${escapar(publicados.get(clave)?.titulo || "")} · ${escapar(formatoNumero.format(Number(merito.cantidad)))} ${escapar(publicados.get(clave)?.unidad || "")} · ${escapar(t("puntos_orientativos", { puntos: formatoNumero.format(calculo.porMerito.get(clave) || 0) }))}</li>`;
  }).join("")}</ul>` : `<p>${escapar(t("revision_sin_meritos"))}</p>`;
  const tramites = TRAMITES_NO_DISPONIBLES.map((clave) => `<li><button type="button" class="boton-secundario" disabled aria-disabled="true" aria-describedby="solicitud-motivo-${clave}">${escapar(t(clave))}</button> <small id="solicitud-motivo-${clave}">${escapar(t("motivo_no_concedida"))}</small></li>`).join("");
  const confirmacion = estado.confirmando
    ? `<div class="confirmacion-solicitud" role="group" aria-labelledby="solicitud-confirmacion-texto"><p id="solicitud-confirmacion-texto" tabindex="-1"><strong>${escapar(t("confirmar_presentacion"))}</strong></p><div class="fila-acciones"><button type="button" class="boton-secundario" data-solicitud-cancelar>${escapar(t("cancelar"))}</button><button type="button" class="boton-primario" data-solicitud-confirmar${estado.ocupado ? " disabled" : ""}>${escapar(estado.ocupado ? t("presentando") : t("confirmar"))}</button></div></div>`
    : "";
  return `<section aria-labelledby="rev-conv"><h4 id="rev-conv">${escapar(t("revision_convocatoria"))}</h4><p>${escapar(reglas.titulo)}</p></section>
    <section aria-labelledby="rev-req"><h4 id="rev-req">${escapar(t("revision_requisitos"))}</h4><ul>${requisitos}</ul></section>
    <section aria-labelledby="rev-datos"><h4 id="rev-datos">${escapar(t("revision_datos"))}</h4><dl class="dato-lista">${datos}</dl></section>
    <section aria-labelledby="rev-mer"><h4 id="rev-mer">${escapar(t("revision_meritos"))}</h4>${meritos}${puntuacionServidor(t, estado)}</section>
    ${estado.datosCompletos === false ? `<p class="nota aviso" role="status">${escapar(t("revision_datos_incompletos"))}</p>` : ""}
    <label class="opcion-check"><input type="checkbox" name="declaracion_responsable" value="true" required${estado.declaracion ? " checked" : ""}><span>${escapar(t("declaracion"))}</span></label>
    ${confirmacion}
    <section aria-labelledby="rev-tramites"><h4 id="rev-tramites">${escapar(t("acciones_no_disponibles"))}</h4><ul class="tramites-no-disponibles">${tramites}</ul></section>`;
}

/** Lo que el servidor declara expresamente no hecho (firma, registro en sede…). */
function serviciosNoRealizados(t, servicios = {}) {
  const claves = TRAMITES_NO_DISPONIBLES.filter((clave) => servicios[clave] === "no_disponible");
  if (!claves.length) return "";
  return `<h4>${escapar(t("acciones_no_disponibles"))}</h4><ul class="tramites-no-disponibles">${claves.map((clave) => `<li>${escapar(t(clave))} · ${escapar(t("servicio_no_realizado"))}</li>`).join("")}</ul>`;
}

function justificante(t, estado) {
  const dato = estado.justificante;
  return `<section class="panel justificante-solicitud"><header><div><h3 id="solicitud-justificante-titulo" tabindex="-1">${escapar(t("justificante_titulo"))}</h3><p>${escapar(estado.reglas?.titulo || "")}</p></div><span class="estado-chip exito">${escapar(t("estado_presentada"))}</span></header><div class="panel-contenido"><dl class="dato-lista"><dt>${escapar(t("justificante_numero"))}</dt><dd><strong>${escapar(dato.numero_justificante)}</strong></dd><dt>${escapar(t("justificante_fecha"))}</dt><dd><time datetime="${escapar(dato.presentada_en)}">${escapar(fecha(dato.presentada_en))}</time></dd><dt>${escapar(t("justificante_puntuacion"))}</dt><dd>${escapar(dato.puntuacion_autobaremo === null ? t("justificante_sin_puntuacion") : t("autobaremo_total", { puntos: formatoNumero.format(Number(dato.puntuacion_autobaremo)) }))}</dd></dl>${serviciosNoRealizados(t, dato.servicios)}<div class="fila-acciones">${botonCopiarReferencia(dato.recibo_ref, { escapar, copiar: t("justificante_copiar"), copiado: t("justificante_copiado") })}<a class="boton-primario" href="?vista=mis-solicitudes" data-ruta="mis-solicitudes">${escapar(t("ver_mis_solicitudes"))}</a>${botonAyuda(t, "ayuda_revision", t("justificante_titulo"))}</div></div></section>`;
}

function avisoError(t, estado) {
  if (!estado.error) return "";
  const recargar = ["version_obsoleta", "clave_reutilizada", "convocatoria_actualizada", "conflicto"].includes(estado.error)
    ? `<button type="button" class="boton-secundario" data-solicitud-recargar>${escapar(t("recargar_borrador"))}</button>` : "";
  const mensaje = estado.error === "__campos" ? estado.errorDetalle || t("error_campos") : textoErrorSolicitud(t, estado.error);
  return `<div class="nota error" role="alert" tabindex="-1" data-solicitud-error><p>${escapar(mensaje)}</p>${recargar}</div>`;
}

/** Sin convocatoria elegida: las convocatorias con plazo que publica el servidor. */
function eleccionConvocatoria(t, estado) {
  if (estado.fase === "error") return `${avisoError(t, estado)}<div class="fila-acciones"><button type="button" class="boton-primario" data-solicitud-reintentar>${escapar(t("reintentar"))}</button></div>`;
  if (!Array.isArray(estado.convocatorias)) return `<div class="estado-carga" role="status">${escapar(t("cargando"))}</div>`;
  const abiertas = estado.convocatorias.filter((item) => item.abierta);
  if (!abiertas.length) return `<div class="estado-vacio"><strong>${escapar(t("sin_convocatorias_abiertas"))}</strong><a class="boton-secundario" href="?vista=mis-solicitudes" data-ruta="mis-solicitudes">${escapar(t("ver_mis_solicitudes"))}</a></div>`;
  return `<section class="panel"><header><div><h3>${escapar(t("convocatorias_abiertas"))}</h3></div></header><div class="panel-contenido"><ul class="lista-plazos">${abiertas.map((item) => `<li><span><strong>${escapar(item.titulo)}</strong>${item.cierra_en ? `<small>${escapar(t("plazo_hasta", { fecha: fecha(item.cierra_en) }))}</small>` : ""}</span><a class="boton-primario" href="?vista=solicitud&amp;id=${encodeURIComponent(item.convocatoria_ref)}" data-ruta="solicitud" data-id="${escapar(item.convocatoria_ref)}">${escapar(t("solicitar"))}</a></li>`).join("")}</ul></div></section>`;
}

/** Presentación pura del asistente a partir de su estado. */
export function renderizarSolicitudConvocatoria(estado, t = crearTraductorSolicitud()) {
  const encabezado = (subtitulo, extra = "") => `<header class="encabezado-vista"><div><h2>${escapar(t("titulo"))}</h2>${subtitulo ? `<p>${escapar(subtitulo)}</p>` : ""}</div>${extra ? `<div class="fila-acciones">${extra}</div>` : ""}</header>`;
  if (!estado.convocatoriaRef) return `${encabezado("")}${eleccionConvocatoria(t, estado)}`;
  if (estado.fase === "cargando") return `${encabezado("")}<div class="estado-carga" role="status">${escapar(t("cargando"))}</div>`;
  if (estado.fase === "error") return `${encabezado("")}${avisoError(t, estado)}<div class="fila-acciones"><button type="button" class="boton-primario" data-solicitud-reintentar>${escapar(t("reintentar"))}</button></div>`;
  if (estado.fase === "presentada") return `${encabezado(estado.reglas?.titulo || "")}${justificante(t, estado)}`;
  if (estado.fase === "ya_presentada") return `${encabezado(estado.reglas?.titulo || "")}<p class="nota" role="status">${escapar(t("ya_presentada"))}</p><a class="boton-primario" href="?vista=mis-solicitudes" data-ruta="mis-solicitudes">${escapar(t("ver_mis_solicitudes"))}</a>`;
  const chip = `<span class="estado-chip ${estado.borrador ? "aviso" : "info"}">${escapar(estado.borrador ? t("borrador_version", { version: estado.borrador.version }) : t("borrador_sin_guardar"))}</span>${estado.reglas.ejemplo ? `<span class="estado-chip info">${escapar(t("reglas_ejemplo"))}</span>` : ""}`;
  if (!estado.reglas.abierta) return `${encabezado(estado.reglas.titulo, chip)}<p class="nota aviso" role="status">${escapar(t("inscripcion_cerrada"))}</p><a class="boton-secundario" href="?vista=mis-solicitudes" data-ruta="mis-solicitudes">${escapar(t("ver_mis_solicitudes"))}</a>`;
  const paso = estado.paso;
  const contenido = [pasoRequisitos, pasoDatos, pasoMeritos, pasoRevision][paso - 1](t, estado);
  const siguiente = paso < TOTAL_PASOS
    ? `<button type="submit" class="boton-primario"${estado.ocupado ? " disabled" : ""}>${escapar(estado.ocupado ? t("guardando") : t("guardar_continuar"))}</button>`
    : `<button type="submit" class="boton-primario"${estado.ocupado || estado.confirmando ? " disabled" : ""}>${escapar(t("presentar"))}</button>`;
  const anterior = paso > 1 ? `<button type="button" class="boton-secundario" data-solicitud-anterior>${escapar(t("anterior"))}</button>` : "";
  const guardado = estado.avisoGuardado ? `<p class="nota" role="status">${escapar(t("borrador_guardado", { version: estado.borrador?.version ?? "" }))}</p>` : "";
  return `${encabezado(estado.reglas.titulo, chip)}${pasos(t, paso)}${avisoError(t, estado)}${guardado}
    <form class="panel" data-solicitud-paso="${paso}" novalidate><header><div><h3 id="solicitud-paso-titulo" tabindex="-1">${escapar(t("paso_de", { paso, total: TOTAL_PASOS }))} · ${escapar(t(CLAVES_PASO[paso - 1]))}</h3></div>${botonAyuda(t, AYUDA_PASO[paso - 1], t(CLAVES_PASO[paso - 1]))}</header><div class="panel-contenido">${contenido}</div><div class="fila-acciones panel-contenido">${anterior}${siguiente}</div></form>`;
}

function leerPaso(estado, formulario) {
  const datos = new FormData(formulario);
  const siguiente = structuredClone(estado.formulario);
  if (estado.paso === 1) {
    estado.reglas.requisitos.forEach((requisito, indice) => {
      const valor = datos.get(`req-${indice}`);
      if (DECLARACIONES.includes(valor)) siguiente.requisitos[requisito.clave] = valor;
    });
    if (estado.reglas.turnos.length) siguiente.turno = String(datos.get("turno") || "");
  } else if (estado.paso === 2) {
    for (const [campo] of CAMPOS_DATOS) siguiente.datos[campo] = String(datos.get(campo) || "");
    for (const [campo] of CAMPOS_DIRECCION) siguiente.datos.direccion[campo] = String(datos.get(campo) || "");
  } else if (estado.paso === 3) {
    (estado.reglas.baremo?.grupos || []).forEach((grupo, gi) => grupo.meritos.forEach((merito, mi) => {
      siguiente.meritos[claveMerito(grupo.clave, merito.clave)] = {
        cantidad: String(datos.get(`m-${gi}-${mi}`) || ""), descripcion: String(datos.get(`d-${gi}-${mi}`) || ""),
      };
    }));
  }
  return siguiente;
}

/** Comprobación de forma del paso (campos rellenos); las reglas las aplica el servidor. */
function faltaEnPaso(t, estado, formulario, formularioHTML) {
  if (estado.paso === 1) {
    if (estado.reglas.requisitos.some((requisito) => !DECLARACIONES.includes(formulario.requisitos[requisito.clave]))) return t("error_requisito_sin_declarar");
    if (estado.reglas.turnos.length && !formulario.turno) return t("error_turno");
  }
  if (estado.paso === 3 && Object.values(formulario.meritos).some((merito) => String(merito?.cantidad || "").trim() !== "" && !cantidadContrato(merito.cantidad))) return t("error_cantidad");
  if (estado.paso === 4 && !formularioHTML.elements.declaracion_responsable?.checked) return t("error_declaracion");
  if (typeof formularioHTML.checkValidity === "function" && !formularioHTML.checkValidity()) return t("error_campos");
  return "";
}

/**
 * Monta el asistente. `cliente` es el adaptador de cliente-http-solicitudes.js.
 * `abrirAyuda(titulo, texto)` muestra la ayuda del botón «?».
 */
export function montarSolicitudConvocatoria({ raiz, cliente, convocatoriaRef = "", anunciar = () => {}, abrirAyuda = () => {}, t = crearTraductorSolicitud() } = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function") throw new TypeError("raíz del asistente no válida");
  let activo = true;
  const estado = {
    convocatoriaRef, fase: "cargando", error: "", errorDetalle: "", reglas: null, paso: 1, borrador: null,
    formulario: formularioVacio(), ocupado: false, avisoGuardado: false, confirmando: false, declaracion: false,
    justificante: null, huellaGuardada: "", intentoGuardado: null, intentoPresentacion: null,
    puntuacionServidor: null, convocatorias: null, datosCompletos: null,
  };
  const pintar = (enfocar = "") => {
    if (!activo) return;
    raiz.innerHTML = renderizarSolicitudConvocatoria(estado, t);
    const destino = enfocar ? raiz.querySelector(enfocar) : null;
    destino?.focus?.({ preventScroll: false });
  };
  const fallar = (error, enfocar = true) => {
    estado.error = error?.codigo || "respuesta_incompatible";
    estado.ocupado = false;
    pintar(enfocar ? "[data-solicitud-error]" : "");
  };

  function aplicarBorrador(borrador) {
    estado.borrador = borrador ? { solicitud_ref: borrador.solicitud_ref, version: borrador.version } : null;
    estado.formulario = borrador ? formularioDesdeBorrador(borrador) : formularioVacio();
    estado.puntuacionServidor = typeof borrador?.puntuacion_autobaremo === "string" ? borrador.puntuacion_autobaremo : null;
    estado.datosCompletos = typeof borrador?.datos_completos === "boolean" ? borrador.datos_completos : null;
    // Lo recién leído ya está guardado: no se vuelve a enviar si no cambia.
    estado.huellaGuardada = borrador && estado.reglas ? JSON.stringify(cuerpoBorrador({ convocatoriaRef, reglas: estado.reglas, formulario: estado.formulario, versionEsperada: borrador.version })) : "";
    estado.intentoGuardado = null; estado.intentoPresentacion = null;
    if (borrador?.estado === "presentada") estado.fase = "ya_presentada";
  }

  async function cargar() {
    estado.fase = "cargando"; estado.error = ""; pintar();
    try {
      if (typeof cliente?.convocatoria !== "function") throw Object.assign(new Error("no_disponible"), { codigo: "no_disponible" });
      if (!convocatoriaRef) {
        estado.convocatorias = await cliente.convocatorias();
        if (!activo) return;
        estado.fase = "listo"; pintar();
        return;
      }
      const [reglas, borrador] = await Promise.all([cliente.convocatoria(convocatoriaRef), cliente.obtenerBorrador(convocatoriaRef)]);
      if (!activo) return;
      estado.reglas = leerReglasConvocatoria(reglas);
      estado.fase = "listo";
      aplicarBorrador(borrador);
      pintar();
    } catch (error) {
      if (!activo) return;
      estado.fase = "error";
      fallar(error, false);
    }
  }

  /** Tras un conflicto: vuelve a leer reglas y última versión guardada. */
  async function recargarBorrador() {
    estado.ocupado = true; pintar();
    try {
      const [reglas, borrador] = await Promise.all([cliente.convocatoria(convocatoriaRef), cliente.obtenerBorrador(convocatoriaRef)]);
      if (!activo) return;
      estado.reglas = leerReglasConvocatoria(reglas);
      aplicarBorrador(borrador);
      estado.error = ""; estado.ocupado = false; estado.confirmando = false;
      pintar("#solicitud-paso-titulo");
    } catch (error) { if (activo) fallar(error); }
  }

  async function guardar() {
    const cuerpo = cuerpoBorrador({ convocatoriaRef, reglas: estado.reglas, formulario: estado.formulario, versionEsperada: estado.borrador?.version ?? 0 });
    const huella = JSON.stringify(cuerpo);
    if (estado.borrador && huella === estado.huellaGuardada) return true;
    // El mismo material y versión reutilizan la clave: un reintento no duplica.
    if (estado.intentoGuardado?.huella !== huella) estado.intentoGuardado = { huella, clave: cliente.nuevaClaveIdempotencia() };
    const resultado = await cliente.guardarBorrador(cuerpo, estado.intentoGuardado.clave);
    estado.borrador = { solicitud_ref: resultado.solicitud_ref, version: resultado.version };
    estado.puntuacionServidor = resultado.puntuacion_autobaremo;
    estado.datosCompletos = resultado.datos_completos;
    estado.huellaGuardada = JSON.stringify({ ...cuerpo, version_esperada: resultado.version });
    estado.intentoGuardado = null;
    return true;
  }

  async function presentar() {
    estado.ocupado = true; pintar();
    try {
      await guardar();
      const huella = `${estado.borrador.solicitud_ref}|${estado.borrador.version}`;
      if (estado.intentoPresentacion?.huella !== huella) estado.intentoPresentacion = { huella, clave: cliente.nuevaClaveIdempotencia() };
      const resultado = await cliente.presentar({ solicitudRef: estado.borrador.solicitud_ref, versionEsperada: estado.borrador.version }, estado.intentoPresentacion.clave);
      if (!activo) return;
      estado.justificante = resultado; estado.fase = "presentada"; estado.ocupado = false; estado.confirmando = false; estado.error = "";
      pintar("#solicitud-justificante-titulo");
      anunciar(t("anuncio_presentada", { numero: resultado.numero_justificante }));
    } catch (error) {
      if (!activo) return;
      estado.confirmando = false;
      fallar(error);
    }
  }

  const alEnviar = async (evento) => {
    const formularioHTML = evento.target?.closest?.("[data-solicitud-paso]");
    if (!formularioHTML) return;
    evento.preventDefault();
    if (estado.ocupado) return;
    estado.formulario = leerPaso(estado, formularioHTML);
    if (estado.paso === 4) estado.declaracion = formularioHTML.elements.declaracion_responsable?.checked === true;
    const falta = faltaEnPaso(t, estado, estado.formulario, formularioHTML);
    if (falta) { estado.error = "__campos"; estado.errorDetalle = falta; pintar("[data-solicitud-error]"); return; }
    estado.error = "";
    if (estado.paso === 4) { estado.confirmando = true; pintar("#solicitud-confirmacion-texto"); return; }
    estado.ocupado = true; pintar();
    try {
      await guardar();
      if (!activo) return;
      estado.ocupado = false; estado.avisoGuardado = true; estado.paso += 1;
      pintar("#solicitud-paso-titulo");
      anunciar(t("anuncio_paso", { paso: estado.paso, titulo: t(CLAVES_PASO[estado.paso - 1]) }));
    } catch (error) { if (activo) fallar(error); }
  };

  const alPulsar = (evento) => {
    const objetivo = evento.target;
    const ayuda = objetivo?.closest?.("[data-ayuda-solicitud]");
    if (ayuda) { abrirAyuda(t("titulo"), t(ayuda.dataset.ayudaSolicitud)); return; }
    if (objetivo?.closest?.("[data-solicitud-anterior]")) {
      const formularioHTML = raiz.querySelector("[data-solicitud-paso]");
      if (formularioHTML) estado.formulario = leerPaso(estado, formularioHTML);
      estado.paso = Math.max(1, estado.paso - 1); estado.error = ""; estado.avisoGuardado = false; estado.confirmando = false;
      pintar("#solicitud-paso-titulo");
      return;
    }
    if (objetivo?.closest?.("[data-solicitud-reintentar]")) { void cargar(); return; }
    if (objetivo?.closest?.("[data-solicitud-recargar]")) { void recargarBorrador(); return; }
    if (objetivo?.closest?.("[data-solicitud-cancelar]")) { estado.confirmando = false; pintar("[data-solicitud-paso] button[type=submit]"); return; }
    if (objetivo?.closest?.("[data-solicitud-confirmar]")) void presentar();
  };

  // Recalcula en vivo el autobaremo orientativo del paso de méritos.
  const alEscribir = (evento) => {
    const formularioHTML = evento.target?.closest?.("[data-solicitud-paso='3']");
    if (!formularioHTML || !estado.reglas?.baremo) return;
    const provisional = leerPaso(estado, formularioHTML);
    const calculo = calcularAutobaremoOrientativo(estado.reglas.baremo, cantidades(provisional));
    formularioHTML.querySelectorAll("[data-puntos-merito]").forEach((salida) => {
      salida.textContent = t("puntos_orientativos", { puntos: formatoNumero.format(calculo.porMerito.get(salida.dataset.puntosMerito) || 0) });
    });
    const total = formularioHTML.querySelector("[data-total-autobaremo]");
    if (total) total.textContent = t("autobaremo_total", { puntos: formatoNumero.format(calculo.total) });
  };

  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("input", alEscribir);
  void cargar();
  return Object.freeze({
    desmontar() {
      if (!activo) return;
      activo = false;
      raiz.removeEventListener("submit", alEnviar);
      raiz.removeEventListener("click", alPulsar);
      raiz.removeEventListener("input", alEscribir);
    },
  });
}
