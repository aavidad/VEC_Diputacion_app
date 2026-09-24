import {
  validarDatosPreviosAnalisis,
  validarReciboAnalisis,
  validarSolicitudRectificacionAnalisis,
  validarSolicitudRegistroAnalisis,
} from "./contrato-analisis.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_GRUPO = /^[A-Z][A-Z0-9/+.-]{0,19}$/u;
const PATRON_HUELLA = /^[0-9a-f]{64}$/u;
const PATRON_JORNADA = /^(?:[1-9][0-9]{0,3}|10000)$/u;
// La API guarda la jornada en diezmilésimas de la jornada completa (entero
// exacto). La persona la escribe en horas y minutos semanales; la referencia
// de jornada completa es provisional hasta que RRHH la confirme.
export const MINUTOS_JORNADA_COMPLETA = 37 * 60 + 30;
const PATRON_ENTERO_DECIMAL = /^[0-9]+$/u;

export function diezmilesimasDesdeHorasMinutos(horas, minutos) {
  if (!PATRON_ENTERO_DECIMAL.test(horas) || !PATRON_ENTERO_DECIMAL.test(minutos)) return "";
  const cantidadHoras = Number(horas);
  const cantidadMinutos = Number(minutos);
  if (cantidadHoras > 37 || cantidadMinutos > 59) return "";
  const total = cantidadHoras * 60 + cantidadMinutos;
  if (total < 1 || total > MINUTOS_JORNADA_COMPLETA) return "";
  return String(Math.max(1, Math.round(total * 10000 / MINUTOS_JORNADA_COMPLETA)));
}

function jornadaDesdeFormulario(datos) {
  const horas = String(datos.get("jornada_horas") ?? "").trim();
  const minutos = String(datos.get("jornada_minutos") ?? "").trim();
  const original = String(datos.get("jornada_original") ?? "");
  const previa = horasMinutosDesdeDiezmilesimas(original);
  if (PATRON_JORNADA.test(original) && diezmilesimasDesdeHorasMinutos(horas, minutos) !== ""
    && String(Number(horas)) === previa.horas
    && String(Number(minutos)) === previa.minutos && horas !== "" && minutos !== "") return original;
  return diezmilesimasDesdeHorasMinutos(horas, minutos);
}

export function horasMinutosDesdeDiezmilesimas(valor) {
  if (!PATRON_JORNADA.test(valor)) return { horas: "", minutos: "" };
  const total = Math.round(Number(valor) * MINUTOS_JORNADA_COMPLETA / 10000);
  return { horas: String(Math.floor(total / 60)), minutos: String(total % 60) };
}
const MAXIMO_OPCIONES = 100;
const MAXIMO_CATEGORIAS = 1000;
const UUID_PRUEBA = "00000000-0000-4000-8000-000000000001";
const CLAVES_MODALIDADES_RRHH = new Set([
  "sustitucion", "vacante", "acumulacion_tareas", "programa", "relevo",
]);
const CAMPOS_CONFIGURACION = new Set([
  "raiz", "cliente", "contexto", "catalogos", "analisisInicial", "datosPrevios",
  "generarClaveIdempotencia", "mensajes", "locale", "zonaHoraria", "anunciar",
]);
const CLAVES_ETIQUETA = Object.freeze({
  modalidad_clave: "analisis_modalidad",
  categoria_ref: "analisis_categoria",
  grupo_subgrupo: "analisis_grupo",
  causa_clave: "analisis_causa",
  inicio: "analisis_inicio",
  fin: "analisis_fin",
  porcentaje_jornada: "analisis_jornada",
  entrada_rc_referencia: "analisis_entrada_rc",
  motivo_rectificacion_clave: "analisis_motivo_rectificacion",
  observaciones: "analisis_observaciones",
  general: "analisis_errores_titulo",
});

function exigirRegistroExacto(valor, campos, nombre) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)
    || Object.getPrototypeOf(valor) !== Object.prototype
    || Object.getOwnPropertySymbols(valor).length !== 0) {
    throw new TypeError(`${nombre} no válido`);
  }
  const descriptores = Object.getOwnPropertyDescriptors(valor);
  const claves = Object.keys(descriptores);
  if (claves.length !== campos.length || claves.some((clave) => !campos.includes(clave))
    || campos.some((campo) => !Object.hasOwn(descriptores, campo))
    || claves.some((clave) => !Object.hasOwn(descriptores[clave], "value")
      || descriptores[clave].enumerable !== true)) {
    throw new TypeError(`${nombre} no válido`);
  }
}

function valoresListaSimple(lista, nombre, permitirVacia = false, maximo = MAXIMO_OPCIONES) {
  if (!Array.isArray(lista) || Object.getPrototypeOf(lista) !== Array.prototype
    || Object.getOwnPropertySymbols(lista).length !== 0
    || lista.length > maximo || (!permitirVacia && lista.length === 0)) {
    throw new TypeError(`${nombre} no válida`);
  }
  const valores = [];
  for (let indice = 0; indice < lista.length; indice += 1) {
    const descriptor = Object.getOwnPropertyDescriptor(lista, String(indice));
    if (!descriptor || !Object.hasOwn(descriptor, "value")
      || descriptor.enumerable !== true) throw new TypeError(`${nombre} no válida`);
    valores.push(descriptor.value);
  }
  if (Reflect.ownKeys(lista).length !== lista.length + 1) {
    throw new TypeError(`${nombre} no válida`);
  }
  return valores;
}

function textoVisible(valor, nombre) {
  if (typeof valor !== "string" || valor.length === 0 || valor.length > 160
    || valor.trim() !== valor || /[\u0000-\u001f\u007f-\u009f]/u.test(valor)) {
    throw new TypeError(`${nombre} no válido`);
  }
  return valor;
}

function normalizarOpciones(lista, { nombre, campo, patron, permitirVacia = false }) {
  const vistos = new Set();
  return Object.freeze(valoresListaSimple(lista, nombre, permitirVacia).map((opcion) => {
    exigirRegistroExacto(opcion, [campo, "etiqueta"], nombre);
    const valor = opcion[campo];
    if (typeof valor !== "string" || !patron.test(valor) || vistos.has(valor)) {
      throw new TypeError(`${nombre} no válida`);
    }
    vistos.add(valor);
    return Object.freeze({ [campo]: valor, etiqueta: textoVisible(opcion.etiqueta, nombre) });
  }));
}

function normalizarCatalogos(entrada, rectificacion) {
  exigirRegistroExacto(entrada, [
    "modalidades", "categorias", "causas", "entradas_rc", "motivos_rectificacion",
  ], "catálogos del análisis");
  const modalidades = normalizarOpciones(entrada.modalidades, {
    nombre: "modalidades", campo: "clave", patron: PATRON_CLAVE,
  });
  if (modalidades.length !== CLAVES_MODALIDADES_RRHH.size
    || modalidades.some(({ clave }) => !CLAVES_MODALIDADES_RRHH.has(clave))) {
    throw new TypeError("modalidades no válida");
  }
  const causas = normalizarOpciones(entrada.causas, {
    nombre: "causas", campo: "clave", patron: PATRON_CLAVE,
  });
  const motivos = normalizarOpciones(entrada.motivos_rectificacion, {
    nombre: "motivos de rectificación", campo: "clave", patron: PATRON_CLAVE,
    // Un catálogo vacío no se sustituye por una opción local: deja la rectificación
    // en solo lectura hasta que el servidor publique motivos válidos.
    permitirVacia: true,
  });
  const categoriasVistas = new Set();
  const categorias = Object.freeze(valoresListaSimple(
    entrada.categorias,
    "categorías",
    false,
    MAXIMO_CATEGORIAS,
  )
    .map((categoria) => {
      exigirRegistroExacto(
        categoria,
        ["referencia", "etiqueta", "grupos_subgrupos"],
        "categoría",
      );
      if (!PATRON_REFERENCIA.test(categoria.referencia)
        || categoriasVistas.has(categoria.referencia)) throw new TypeError("categoría no válida");
      categoriasVistas.add(categoria.referencia);
      return Object.freeze({
        referencia: categoria.referencia,
        etiqueta: textoVisible(categoria.etiqueta, "categoría"),
        grupos_subgrupos: normalizarOpciones(categoria.grupos_subgrupos, {
          nombre: "grupos o subgrupos", campo: "clave", patron: PATRON_GRUPO,
        }),
      });
    }));
  const entradasVistas = new Set();
  const entradasRC = Object.freeze(valoresListaSimple(entrada.entradas_rc, "entradas RC")
    .map((item) => {
      exigirRegistroExacto(
        item,
        ["referencia", "huella_sha256", "etiqueta"],
        "entrada RC",
      );
      if (!PATRON_REFERENCIA.test(item.referencia) || entradasVistas.has(item.referencia)
        || !PATRON_HUELLA.test(item.huella_sha256) || /^0{64}$/u.test(item.huella_sha256)) {
        throw new TypeError("entrada RC no válida");
      }
      entradasVistas.add(item.referencia);
      return Object.freeze({
        referencia: item.referencia,
        huella_sha256: item.huella_sha256,
        etiqueta: textoVisible(item.etiqueta, "entrada RC"),
      });
    }));
  return Object.freeze({
    modalidades, categorias, causas, entradas_rc: entradasRC, motivos_rectificacion: motivos,
  });
}

function normalizarContexto(contexto) {
  exigirRegistroExacto(
    contexto,
    ["operacion", "expediente_ref", "version_esperada", "artefacto_ref"],
    "contexto del análisis",
  );
  if (!["registrar", "rectificar"].includes(contexto.operacion)
    || !PATRON_REFERENCIA.test(contexto.expediente_ref)
    || !Number.isSafeInteger(contexto.version_esperada)
    || contexto.version_esperada < 1 || contexto.version_esperada >= Number.MAX_SAFE_INTEGER
    || !PATRON_REFERENCIA.test(contexto.artefacto_ref)) {
    throw new TypeError("contexto del análisis no válido");
  }
  return Object.freeze({ ...contexto });
}

function crearBorrador(analisis = null) {
  return analisis === null ? {
    modalidad_clave: "", categoria_ref: "", grupo_subgrupo: "", causa_clave: "",
    inicio: "", fin: "", porcentaje_jornada: "", entrada_rc_referencia: "",
    motivo_rectificacion_clave: "", observaciones: "",
  } : {
    modalidad_clave: analisis.modalidad_clave,
    categoria_ref: analisis.categoria_ref,
    grupo_subgrupo: analisis.grupo_subgrupo,
    causa_clave: analisis.causa_clave,
    inicio: analisis.periodo.inicio.slice(0, 10),
    fin: analisis.periodo.fin.slice(0, 10),
    porcentaje_jornada: String(analisis.porcentaje_jornada),
    entrada_rc_referencia: analisis.entrada_rc.referencia,
    motivo_rectificacion_clave: "",
    observaciones: analisis.observaciones ?? "",
  };
}

function longitudUnicode(texto) {
  return [...texto].length;
}

function textoValido(valor, maximo, permiteVacio) {
  if (typeof valor !== "string" || valor !== valor.trim()
    || valor.normalize("NFC") !== valor || longitudUnicode(valor) > maximo
    || (!permiteVacio && valor === "")) {
    return false;
  }
  for (const caracter of valor) {
    const codigo = caracter.codePointAt(0);
    if ((codigo < 32 || (codigo >= 127 && codigo <= 159))
      && caracter !== "\n" && caracter !== "\t"
      || (codigo >= 0xD800 && codigo <= 0xDFFF)) {
      return false;
    }
  }
  return true;
}

function fechaCivilValida(valor) {
  if (!/^\d{4}-\d{2}-\d{2}$/u.test(valor) || valor.startsWith("0000-")) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.valueOf()) && fecha.toISOString().slice(0, 10) === valor;
}

function periodoDentroDelMaximo(inicio, fin) {
  if (fin < inicio) return false;
  const limite = new Date(`${inicio}T00:00:00Z`);
  limite.setUTCFullYear(limite.getUTCFullYear() + 100);
  return limite.getUTCFullYear() > 9_999 || fin <= limite.toISOString().slice(0, 10);
}

function validarBorrador(borrador, catalogos, rectificacion) {
  const errores = {};
  const categoria = catalogos.categorias.find(
    ({ referencia }) => referencia === borrador.categoria_ref,
  );
  if (!catalogos.modalidades.some(({ clave }) => clave === borrador.modalidad_clave)) {
    errores.modalidad_clave = "opcion";
  }
  if (!categoria) errores.categoria_ref = "opcion";
  if (!categoria?.grupos_subgrupos.some(({ clave }) => clave === borrador.grupo_subgrupo)) {
    errores.grupo_subgrupo = "opcion";
  }
  if (!catalogos.causas.some(({ clave }) => clave === borrador.causa_clave)) {
    errores.causa_clave = "opcion";
  }
  if (!fechaCivilValida(borrador.inicio)) errores.inicio = "fecha";
  if (!fechaCivilValida(borrador.fin)) errores.fin = "fecha";
  if (!errores.inicio && !errores.fin
    && !periodoDentroDelMaximo(borrador.inicio, borrador.fin)) errores.fin = "periodo";
  if (!PATRON_JORNADA.test(borrador.porcentaje_jornada)) {
    errores.porcentaje_jornada = "jornada";
  }
  if (!catalogos.entradas_rc.some(
    ({ referencia }) => referencia === borrador.entrada_rc_referencia,
  )) errores.entrada_rc_referencia = "opcion";
  if (rectificacion && !catalogos.motivos_rectificacion.some(
    ({ clave }) => clave === borrador.motivo_rectificacion_clave,
  )) errores.motivo_rectificacion_clave = "motivo";
  if (borrador.observaciones) {
    if (!textoValido(borrador.observaciones, 4000, true)) {
      errores.observaciones = "observaciones";
    }
  }
  return errores;
}

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function opciones(lista, campo, seleccion, t) {
  return [`<option value="">${escaparHTML(t("seleccionar"))}</option>`, ...lista.map(
    (opcion) => `<option value="${escaparHTML(opcion[campo])}"${
      opcion[campo] === seleccion ? " selected" : ""}>${escaparHTML(opcion.etiqueta)}</option>`,
  )].join("");
}

function atributosCampo(estado, campo) {
  return `aria-describedby="ct-analisis-${campo}-ayuda${
    estado.errores[campo] ? ` ct-analisis-${campo}-error` : ""}"${
    estado.errores[campo] ? ' aria-invalid="true"' : ""}`;
}

function mensajeCampo(t, codigo) {
  return t(`analisis_error_${codigo}`);
}

function campoSeleccion(estado, t, campo, claveEtiqueta, claveAyuda, lista, propiedad) {
  return `<div class="ct-campo">
    <label for="ct-analisis-${campo}">${escaparHTML(t(claveEtiqueta))} <b aria-hidden="true">*</b></label>
    <select id="ct-analisis-${campo}" name="${campo}" required ${atributosCampo(estado, campo)}>
      ${opciones(lista, propiedad, estado.borrador[campo], t)}
    </select>
    <small id="ct-analisis-${campo}-ayuda">${escaparHTML(t(claveAyuda))}</small>
    ${estado.errores[campo] ? `<span class="ct-error-campo" id="ct-analisis-${campo}-error">${
      escaparHTML(mensajeCampo(t, estado.errores[campo]))}</span>` : ""}
  </div>`;
}

function campoEntrada(estado, t, campo, tipo, claveEtiqueta, claveAyuda, atributos = "") {
  return `<div class="ct-campo">
    <label for="ct-analisis-${campo}">${escaparHTML(t(claveEtiqueta))} <b aria-hidden="true">*</b></label>
    <input id="ct-analisis-${campo}" name="${campo}" type="${tipo}" required value="${
      escaparHTML(estado.borrador[campo])}" ${atributosCampo(estado, campo)} ${atributos}>
    <small id="ct-analisis-${campo}-ayuda">${escaparHTML(t(claveAyuda))}</small>
    ${estado.errores[campo] ? `<span class="ct-error-campo" id="ct-analisis-${campo}-error">${
      escaparHTML(mensajeCampo(t, estado.errores[campo]))}</span>` : ""}
  </div>`;
}

function campoJornada(estado, t, formateadorJornada) {
  const valor = estado.borrador.porcentaje_jornada;
  const derivada = horasMinutosDesdeDiezmilesimas(valor);
  const horas = estado.borrador.jornada_horas ?? derivada.horas;
  const minutos = estado.borrador.jornada_minutos ?? derivada.minutos;
  const equivalencia = PATRON_JORNADA.test(valor) ? t("analisis_jornada_equivalencia", {
    porcentaje: formateadorJornada.format(Number(valor) / 10000),
  }) : "";
  const atributos = atributosCampo(estado, "porcentaje_jornada");
  // jornada_original conserva el valor exacto en diezmilésimas: si horas y
  // minutos no se tocan, se reenvía tal cual y no se redondea al minuto.
  const original = PATRON_JORNADA.test(valor)
    ? `<input type="hidden" name="jornada_original" value="${escaparHTML(valor)}">` : "";
  return `<fieldset class="ct-campo ct-campo-jornada">${original}
    <legend>${escaparHTML(t("analisis_jornada"))} <b aria-hidden="true">*</b></legend>
    <div class="ct-jornada-entradas">
      <label for="ct-analisis-porcentaje_jornada">${escaparHTML(t("analisis_jornada_horas"))}</label>
      <input id="ct-analisis-porcentaje_jornada" name="jornada_horas" type="number" required value="${escaparHTML(horas)}" ${atributos} min="0" max="37" step="1" inputmode="numeric">
      <label for="ct-analisis-jornada_minutos">${escaparHTML(t("analisis_jornada_minutos"))}</label>
      <input id="ct-analisis-jornada_minutos" name="jornada_minutos" type="number" required value="${escaparHTML(minutos)}" ${atributos} min="0" max="59" step="1" inputmode="numeric">
    </div>
    <small id="ct-analisis-porcentaje_jornada-ayuda">${escaparHTML(t("analisis_jornada_ayuda"))}</small>
    <small id="ct-analisis-porcentaje_jornada-equivalencia" aria-live="polite" aria-atomic="true">${escaparHTML(equivalencia)}</small>
    ${estado.errores.porcentaje_jornada ? `<span class="ct-error-campo" id="ct-analisis-porcentaje_jornada-error">${escaparHTML(mensajeCampo(t, estado.errores.porcentaje_jornada))}</span>` : ""}
  </fieldset>`;
}

function campoAreaTexto(estado, t, campo, id, claveEtiqueta, claveAyuda, maxlength = 4000) {
  return `<div class="ct-campo">
    <label for="${id}">${escaparHTML(t(claveEtiqueta))}</label>
    <textarea id="${id}" name="${campo}" maxlength="${maxlength}" ${atributosCampo(estado, campo)}>${escaparHTML(estado.borrador[campo] ?? "")}</textarea>
    <small id="ct-analisis-${campo}-ayuda">${escaparHTML(t(claveAyuda))}</small>
    ${estado.errores[campo] ? `<span class="ct-error-campo" id="ct-analisis-${campo}-error">${
      escaparHTML(mensajeCampo(t, estado.errores[campo]))}</span>` : ""}
  </div>`;
}

function resumenErrores(estado, t) {
  const entradas = Object.entries(estado.errores);
  if (entradas.length === 0) return "";
  return `<section class="ct-resumen-errores" data-ct-analisis-error-general role="alert"
    aria-live="assertive" aria-atomic="true" tabindex="-1">
    <h3>${escaparHTML(t("analisis_errores_titulo"))}</h3>
    <p>${escaparHTML(t("analisis_errores_descripcion"))}</p>
    <ul>${entradas.map(([campo, codigo]) => `<li>${campo === "general" ? "" :
    `<button type="button" data-ct-analisis-enfocar="${escaparHTML(campo)}">`}${
    escaparHTML(t(CLAVES_ETIQUETA[campo]))}: ${escaparHTML(mensajeCampo(t, codigo))}${
    campo === "general" ? "" : "</button>"}</li>`).join("")}</ul>
  </section>`;
}

function renderizarContenido(estado, contexto, catalogos, t, formateador, formateadorJornada) {
  if (estado.recibo) {
    const recibo = estado.recibo;
    return `<section class="ct-recibo" data-ct-analisis-recibo role="status" aria-live="polite"
      aria-atomic="true" tabindex="-1" aria-labelledby="ct-analisis-recibo-titulo">
      <p class="sobrelinea">${escaparHTML(t("analisis_recibo_sobrelinea"))}</p>
      <h3 id="ct-analisis-recibo-titulo">${escaparHTML(t(recibo.operacion === "rectificar" ? "analisis_recibo_rectificacion_titulo" : "analisis_recibo_titulo"))}</h3>
      <p>${escaparHTML(t("analisis_recibo_descripcion"))}</p>
      <dl><div><dt>${escaparHTML(t("analisis_recibo_expediente"))}</dt><dd><code>${escaparHTML(recibo.expediente_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("analisis_recibo_version"))}</dt><dd>${recibo.version_resultante}</dd></div>
      <div><dt>${escaparHTML(t("analisis_recibo_referencia"))}</dt><dd><code>${escaparHTML(recibo.recibo_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("analisis_recibo_fecha"))}</dt><dd>${escaparHTML(formateador.format(new Date(recibo.confirmada_en)))}</dd></div></dl>
    </section>`;
  }
  if (estado.bloqueado) {
    return `<section class="ct-alcance" data-ct-analisis-indeterminado role="status"
      aria-live="assertive" aria-atomic="true" tabindex="-1">
      <h3>${escaparHTML(t("analisis_indeterminado_titulo"))}</h3>
      <p>${escaparHTML(t("analisis_indeterminado_descripcion"))}</p>
    </section>`;
  }
  if (estado.solo_lectura) {
    return `<section class="ct-alcance" data-ct-analisis-solo-lectura role="status"
      aria-live="polite" aria-atomic="true" tabindex="-1">
      <h3>${escaparHTML(t("analisis_rectificacion_solo_lectura_titulo"))}</h3>
      <p>${escaparHTML(t("analisis_rectificacion_solo_lectura_descripcion"))}</p>
    </section>`;
  }
  const categoria = catalogos.categorias.find(
    ({ referencia }) => referencia === estado.borrador.categoria_ref,
  );
  const rectificacion = contexto.operacion === "rectificar";
  return `${resumenErrores(estado, t)}
  <form data-ct-analisis-form novalidate>
    <fieldset class="ct-bloque"${estado.ocupado ? " disabled" : ""}>
      <legend>${escaparHTML(t("analisis_campos_leyenda"))}</legend>
      <p>${escaparHTML(t("campos_obligatorios"))}</p>
      <div class="ct-campos">
        ${campoSeleccion(estado, t, "modalidad_clave", "analisis_modalidad", "analisis_modalidad_ayuda", catalogos.modalidades, "clave")}
        ${campoSeleccion(estado, t, "categoria_ref", "analisis_categoria", "analisis_categoria_ayuda", catalogos.categorias, "referencia")}
        ${campoSeleccion(estado, t, "grupo_subgrupo", "analisis_grupo", "analisis_grupo_ayuda", categoria?.grupos_subgrupos ?? [], "clave")}
        ${campoSeleccion(estado, t, "causa_clave", "analisis_causa", "analisis_causa_ayuda", catalogos.causas, "clave")}
        ${campoEntrada(estado, t, "inicio", "date", "analisis_inicio", "analisis_periodo_ayuda")}
        ${campoEntrada(estado, t, "fin", "date", "analisis_fin", "analisis_periodo_ayuda")}
        ${campoJornada(estado, t, formateadorJornada)}
        ${campoSeleccion(estado, t, "entrada_rc_referencia", "analisis_entrada_rc", "analisis_entrada_rc_ayuda", catalogos.entradas_rc, "referencia")}
        ${rectificacion ? campoSeleccion(estado, t, "motivo_rectificacion_clave", "analisis_motivo_rectificacion", "analisis_motivo_rectificacion_ayuda", catalogos.motivos_rectificacion, "clave") : ""}
        ${campoAreaTexto(estado, t, "observaciones", "analisis_observaciones", "analisis_observaciones", "analisis_observaciones_ayuda", 4000)}
      </div>
    </fieldset>
    <div class="ct-acciones">${estado.ocupado
    ? `<button class="boton-secundario" type="button" data-ct-analisis-accion="cancelar">${escaparHTML(t("analisis_cancelar"))}</button>`
    : `<button class="boton-primario" type="submit">${escaparHTML(t(rectificacion ? "analisis_rectificar" : "analisis_registrar"))}</button>`}</div>
  </form>`;
}

function extraerBorrador(formulario) {
  const datos = new FormData(formulario);
  return {
    modalidad_clave: String(datos.get("modalidad_clave") ?? ""),
    categoria_ref: String(datos.get("categoria_ref") ?? ""),
    grupo_subgrupo: String(datos.get("grupo_subgrupo") ?? ""),
    causa_clave: String(datos.get("causa_clave") ?? ""),
    inicio: String(datos.get("inicio") ?? ""), fin: String(datos.get("fin") ?? ""),
    jornada_horas: String(datos.get("jornada_horas") ?? "").trim(),
    jornada_minutos: String(datos.get("jornada_minutos") ?? "").trim(),
    porcentaje_jornada: jornadaDesdeFormulario(datos),
    entrada_rc_referencia: String(datos.get("entrada_rc_referencia") ?? ""),
    motivo_rectificacion_clave: String(datos.get("motivo_rectificacion_clave") ?? ""),
    observaciones: String(datos.get("observaciones") ?? "").trim(),
  };
}

function esIndeterminado(error) {
  try {
    return error?.resultadoIndeterminado === true;
  } catch {
    return false;
  }
}

function claveErrorPublico(error) {
  let codigo;
  try { codigo = error?.codigo; } catch { return "analisis_estado_error"; }
  if (["autenticacion_requerida", "acceso_denegado"].includes(codigo)) {
    return "analisis_estado_acceso_denegado";
  }
  if (["conflicto", "clave_idempotencia_reutilizada"].includes(codigo)) {
    return "analisis_estado_conflicto";
  }
  if (["peticion_no_valida", "peticion_no_permitida", "contenido_no_valido"].includes(codigo)) {
    return "analisis_estado_rechazado";
  }
  return "analisis_estado_error";
}

function enfocar(raiz, selector) {
  const elemento = raiz?.querySelector?.(selector);
  elemento?.focus?.();
  elemento?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
}

/** Monta la vista/controlador sin resolver identidad, perfil ni autorización. */
export function montarFormularioAnalisisRRHH(configuracion = {}) {
  const descriptoresConfiguracion = configuracion !== null
    && typeof configuracion === "object" && !Array.isArray(configuracion)
    ? Object.getOwnPropertyDescriptors(configuracion) : {};
  if (configuracion === null || typeof configuracion !== "object" || Array.isArray(configuracion)
    || Object.getPrototypeOf(configuracion) !== Object.prototype
    || Object.getOwnPropertySymbols(configuracion).length !== 0
    || Object.keys(descriptoresConfiguracion).some((campo) => !CAMPOS_CONFIGURACION.has(campo)
      || !Object.hasOwn(descriptoresConfiguracion[campo], "value")
      || descriptoresConfiguracion[campo].enumerable !== true)) {
    throw new TypeError("configuración del formulario de análisis no válida");
  }
  let {
    raiz, cliente, contexto: contextoEntrada, catalogos: catalogosEntrada,
    analisisInicial = null, datosPrevios = null, generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
    mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid", anunciar = () => {},
  } = configuracion;
  configuracion = null;
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function" || typeof raiz.querySelector !== "function"
    || typeof raiz.contains !== "function" || typeof generarClaveIdempotencia !== "function"
    || typeof anunciar !== "function" || typeof AbortController !== "function"
    || typeof FormData !== "function") throw new TypeError("dependencias del formulario no válidas");
  let contexto = normalizarContexto(contextoEntrada);
  const rectificacion = contexto.operacion === "rectificar";
  let catalogos = normalizarCatalogos(catalogosEntrada, rectificacion);
  const soloLectura = rectificacion && catalogos.motivos_rectificacion.length === 0;
  let metodo = rectificacion ? cliente?.rectificarAnalisis : cliente?.registrarAnalisis;
  if (!soloLectura && typeof metodo !== "function") throw new TypeError("cliente de análisis no válido");
  let t = crearTraductorContratacionTemporal(mensajes);
  let formateador = new Intl.DateTimeFormat(locale, {
    dateStyle: "long", timeStyle: "medium", timeZone: zonaHoraria,
  });
  let formateadorJornada = new Intl.NumberFormat(locale, {
    style: "percent", minimumFractionDigits: 2, maximumFractionDigits: 2,
  });
  let analisisValidado = null;
  if (analisisInicial !== null) {
    try {
      analisisValidado = validarSolicitudRegistroAnalisis({
        expediente_ref: contexto.expediente_ref, version_esperada: contexto.version_esperada,
        clave_idempotencia: UUID_PRUEBA, artefacto_ref: contexto.artefacto_ref,
        analisis: analisisInicial,
      }).analisis;
    } catch {
      throw new TypeError("análisis inicial no válido");
    }
  }
  let borrador = crearBorrador(analisisValidado);
  if (analisisValidado !== null) {
    if (Object.keys(validarBorrador(borrador, catalogos, false)).length !== 0) {
      throw new TypeError("análisis inicial ajeno a los catálogos");
    }
  }
  if (datosPrevios !== null) {
    if (!rectificacion || analisisInicial !== null) throw new TypeError("datos previos incompatibles");
    const previo = validarDatosPreviosAnalisis(datosPrevios);
    borrador = {
      ...borrador,
      modalidad_clave: previo.modalidad_clave,
      categoria_ref: previo.categoria_ref,
      causa_clave: previo.causa_clave,
      inicio: previo.periodo.inicio.slice(0, 10),
      fin: previo.periodo.fin.slice(0, 10),
      porcentaje_jornada: String(previo.porcentaje_jornada),
      observaciones: previo.observaciones ?? "",
    };
    // Una opción histórica retirada debe seleccionarse de nuevo en el catálogo vigente.
    const erroresPrevios = validarBorrador(borrador, catalogos, false);
    for (const campo of ["modalidad_clave", "categoria_ref", "causa_clave"]) {
      if (erroresPrevios[campo]) borrador[campo] = "";
    }
  }
  datosPrevios = null;
  let raizActual = raiz;
  let clienteActual = cliente;
  let anunciarActual = anunciar;
  let generarClaveActual = generarClaveIdempotencia;
  let montado = true;
  let envioActual = null;
  let controlador = null;
  let cancelacionSolicitada = false;
  let intento = null;
  let estado = {
    borrador, errores: {}, ocupado: false, bloqueado: false, solo_lectura: soloLectura, recibo: null,
    mensaje_clave: soloLectura ? "analisis_estado_solo_lectura" : "analisis_estado_listo",
    tipo_mensaje: "informacion",
  };
  raiz = null;
  cliente = null;
  contextoEntrada = null;
  catalogosEntrada = null;
  analisisInicial = null;
  generarClaveIdempotencia = null;
  mensajes = null;
  locale = null;
  zonaHoraria = null;
  anunciar = null;

  function repintar(selectorFoco = "") {
    if (!montado) return;
    const titulo = rectificacion ? "analisis_titulo_rectificar" : "analisis_titulo_registrar";
    raizActual.innerHTML = `<section class="ct-alta" data-ct-analisis
      aria-labelledby="ct-analisis-titulo">
      <header class="ct-cabecera"><div><p class="sobrelinea">${escaparHTML(t("analisis_sobrelinea"))}</p>
      <h2 id="ct-analisis-titulo">${escaparHTML(t(titulo))}</h2>
      <p>${escaparHTML(t("analisis_descripcion"))}</p></div>
      <aside class="ct-alcance" aria-label="${escaparHTML(t("analisis_alcance_etiqueta"))}">${escaparHTML(t("analisis_alcance"))}</aside></header>
      <div class="ct-estado ct-estado-${escaparHTML(estado.tipo_mensaje)}" data-ct-analisis-estado
        role="status" aria-live="polite" aria-atomic="true" tabindex="-1">
        <strong>${escaparHTML(t(estado.mensaje_clave))}</strong></div>
      ${renderizarContenido(estado, contexto, catalogos, t, formateador, formateadorJornada)}
    </section>`;
    if (selectorFoco) enfocar(raizActual, selectorFoco);
    try { anunciarActual(t(estado.mensaje_clave), estado.tipo_mensaje); } catch {
      // El anunciador auxiliar no sustituye la región viva ni rompe la vista.
    }
  }

  function construirSolicitud(entrada) {
    const errores = validarBorrador(entrada, catalogos, rectificacion);
    if (Object.keys(errores).length !== 0) return { errores };
    const rc = catalogos.entradas_rc.find(
      ({ referencia }) => referencia === entrada.entrada_rc_referencia,
    );
    const analisis = {
      modalidad_clave: entrada.modalidad_clave,
      categoria_ref: entrada.categoria_ref,
      grupo_subgrupo: entrada.grupo_subgrupo,
      causa_clave: entrada.causa_clave,
      periodo: { inicio: `${entrada.inicio}T00:00:00Z`, fin: `${entrada.fin}T00:00:00Z` },
      porcentaje_jornada: Number(entrada.porcentaje_jornada),
      entrada_rc: { referencia: rc.referencia, huella_sha256: rc.huella_sha256 },
    };
    if (typeof entrada.observaciones === "string" && entrada.observaciones.trim() !== "") {
      analisis.observaciones = entrada.observaciones.trim();
    }
    const semantica = JSON.stringify([contexto, analisis,
      rectificacion ? entrada.motivo_rectificacion_clave : null]);
    const clave = intento?.semantica === semantica
      ? intento.clave : generarClaveActual();
    const solicitud = {
      expediente_ref: contexto.expediente_ref,
      version_esperada: contexto.version_esperada,
      clave_idempotencia: clave,
      artefacto_ref: contexto.artefacto_ref,
      analisis,
    };
    if (rectificacion) {
      solicitud.motivo_rectificacion_clave = entrada.motivo_rectificacion_clave;
    }
    try {
      const validada = rectificacion
        ? validarSolicitudRectificacionAnalisis(solicitud)
        : validarSolicitudRegistroAnalisis(solicitud);
      intento = Object.freeze({ semantica, clave: validada.clave_idempotencia });
      return { solicitud: validada };
    } catch {
      return { errores: { general: "contrato" } };
    }
  }

  function bloquearIndeterminado() {
    estado = {
      ...estado, ocupado: false, bloqueado: true, recibo: null, errores: {},
      mensaje_clave: "analisis_estado_indeterminado", tipo_mensaje: "aviso",
    };
  }

  function enviar(entrada) {
    if (envioActual !== null) return envioActual;
    if (!montado || estado.bloqueado || estado.solo_lectura || estado.recibo) return Promise.resolve(null);
    estado = { ...estado, borrador: entrada, errores: {} };
    let preparada;
    try { preparada = construirSolicitud(entrada); } catch { preparada = { errores: { general: "contrato" } }; }
    if (preparada.errores) {
      estado = {
        ...estado, errores: preparada.errores,
        mensaje_clave: "analisis_estado_validacion", tipo_mensaje: "error",
      };
      repintar("[data-ct-analisis-error-general]");
      return Promise.resolve(null);
    }
    controlador = new AbortController();
    cancelacionSolicitada = false;
    estado = {
      ...estado, ocupado: true, mensaje_clave: "analisis_estado_enviando",
      tipo_mensaje: "informacion",
    };
    repintar("[data-ct-analisis-accion='cancelar']");
    const solicitud = preparada.solicitud;
    const tarea = (async () => {
      await Promise.resolve();
      try {
        const respuesta = await metodo.call(
          clienteActual,
          solicitud,
          Object.freeze({ signal: controlador.signal }),
        );
        let recibo;
        try {
          recibo = validarReciboAnalisis(respuesta);
          if (recibo.operacion !== contexto.operacion
            || recibo.expediente_ref !== contexto.expediente_ref
            || recibo.version_resultante !== contexto.version_esperada + 1) {
            throw new TypeError("recibo no ligado");
          }
        } catch {
          if (montado) bloquearIndeterminado();
          return null;
        }
        if (!montado) return null;
        estado = {
          ...estado, borrador: crearBorrador(), ocupado: false, recibo,
          errores: {}, mensaje_clave: "analisis_estado_confirmado", tipo_mensaje: "exito",
        };
        intento = null;
        return recibo;
      } catch (errorPrivado) {
        if (!montado) return null;
        if (cancelacionSolicitada || esIndeterminado(errorPrivado)) bloquearIndeterminado();
        else {
          estado = {
            ...estado, ocupado: false, recibo: null,
            mensaje_clave: claveErrorPublico(errorPrivado), tipo_mensaje: "error",
          };
        }
        return null;
      } finally {
        controlador = null;
        cancelacionSolicitada = false;
        envioActual = null;
        if (montado) repintar(estado.recibo ? "[data-ct-analisis-recibo]"
          : estado.bloqueado ? "[data-ct-analisis-indeterminado]" : "[data-ct-analisis-estado]");
      }
    })();
    envioActual = tarea;
    return tarea;
  }

  function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-analisis-form]");
    if (!formulario || !raizActual.contains(formulario)) return undefined;
    evento.preventDefault();
    return enviar(extraerBorrador(formulario));
  }

  function alCambiar(evento) {
    if (evento.target?.name !== "categoria_ref" || estado.ocupado) return;
    const formulario = evento.target.closest?.("[data-ct-analisis-form]");
    if (!formulario || !raizActual.contains(formulario)) return;
    const siguiente = extraerBorrador(formulario);
    siguiente.grupo_subgrupo = "";
    estado = { ...estado, borrador: siguiente, errores: {} };
    repintar("#ct-analisis-categoria_ref");
  }

  function alEscribir(evento) {
    const entrada = evento.target;
    if ((entrada?.name !== "jornada_horas" && entrada?.name !== "jornada_minutos") || estado.ocupado) return;
    const formulario = entrada.closest?.("[data-ct-analisis-form]");
    if (!formulario || !raizActual.contains(formulario)) return;
    const ayuda = raizActual.querySelector("#ct-analisis-porcentaje_jornada-equivalencia");
    if (!ayuda) return;
    const horas = formulario.querySelector?.("[name=jornada_horas]")?.value ?? "";
    const minutos = formulario.querySelector?.("[name=jornada_minutos]")?.value ?? "";
    const valor = diezmilesimasDesdeHorasMinutos(String(horas).trim(), String(minutos).trim());
    ayuda.textContent = valor
      ? t("analisis_jornada_equivalencia", { porcentaje: formateadorJornada.format(Number(valor) / 10000) })
      : "";
  }

  function alPulsar(evento) {
    const enlaceError = evento.target?.closest?.("[data-ct-analisis-enfocar]");
    if (enlaceError && raizActual.contains(enlaceError)) {
      evento.preventDefault();
      const campo = enlaceError.dataset.ctAnalisisEnfocar;
      const selector = campo === "observaciones" ? "#analisis_observaciones" : `#ct-analisis-${campo}`;
      enfocar(raizActual, selector);
      return;
    }
    const accion = evento.target?.closest?.("[data-ct-analisis-accion]");
    if (!accion || !raizActual.contains(accion)
      || accion.dataset.ctAnalisisAccion !== "cancelar") return;
    evento.preventDefault();
    if (controlador && estado.ocupado) {
      cancelacionSolicitada = true;
      estado = {
        ...estado, mensaje_clave: "analisis_estado_cancelando", tipo_mensaje: "aviso",
      };
      controlador.abort();
      repintar("[data-ct-analisis-estado]");
    }
  }

  raizActual.addEventListener("submit", alEnviar);
  raizActual.addEventListener("change", alCambiar);
  raizActual.addEventListener("input", alEscribir);
  raizActual.addEventListener("click", alPulsar);
  repintar();

  return function desmontarFormularioAnalisis() {
    if (!montado) return;
    montado = false;
    controlador?.abort();
    raizActual.removeEventListener("submit", alEnviar);
    raizActual.removeEventListener("change", alCambiar);
    raizActual.removeEventListener("input", alEscribir);
    raizActual.removeEventListener("click", alPulsar);
    if (typeof raizActual.replaceChildren === "function") raizActual.replaceChildren();
    else raizActual.innerHTML = "";
    estado = null;
    intento = null;
    envioActual = null;
    controlador = null;
    catalogos = null;
    clienteActual = null;
    anunciarActual = null;
    generarClaveActual = null;
    formateador = null;
    formateadorJornada = null;
    contexto = null;
    metodo = null;
    analisisValidado = null;
    t = null;
    raizActual = null;
    borrador = null;
  };
}
