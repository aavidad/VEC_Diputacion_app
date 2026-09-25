import { rotuloTipoOtroGasto } from "./i18n-otros-gastos.js?v=20260925-d5-v1";

/**
 * Líneas de otros medios de transporte y otros gastos (D5) del formulario de
 * comisión. Cada línea lleva un tipo del catálogo versionado que sirve el
 * servidor, la fecha del gasto, una descripción breve, el importe y el
 * justificante por referencia y huella SHA-256. El fichero del justificante
 * no sale del navegador: solo se lee, si la persona lo elige, para calcular
 * su huella. El servidor vuelve a validar todo y calcula los totales.
 */

const REFERENCIA = /^[A-Za-z][A-Za-z0-9:_-]{2,127}$/u;
const HUELLA = /^[a-f0-9]{64}$/u;
const MAXIMO_FICHERO_BYTES = 25 * 1024 * 1024;
const GRUPOS = [["otro_medio", "otros_gastos_grupo_medio"], ["otro_gasto", "otros_gastos_grupo_gasto"]];

function nodo(documento, etiqueta, texto = "") {
  const resultado = documento.createElement(etiqueta);
  if (texto !== "") resultado.textContent = texto;
  return resultado;
}

function fechaCivil(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [ano, mes, dia] = valor.split("-").map(Number);
  const fecha = new Date(Date.UTC(ano, mes - 1, dia));
  return fecha.getUTCFullYear() === ano && fecha.getUTCMonth() === mes - 1 && fecha.getUTCDate() === dia;
}

/** Valida la proyección del catálogo o devuelve null si no es utilizable. */
export function catalogoOtrosGastosValido(catalogo) {
  if (!catalogo || typeof catalogo !== "object" || typeof catalogo.version !== "string" ||
      !/^provisional:[a-z0-9:-]{8,120}$/u.test(catalogo.version) || !Array.isArray(catalogo.tipos) ||
      catalogo.tipos.length < 1 || catalogo.tipos.length > 64 ||
      catalogo.tipos.some((tipo) => typeof tipo?.codigo !== "string" || !/^[a-z][a-z_]{1,40}$/u.test(tipo.codigo) ||
        !GRUPOS.some(([clase]) => clase === tipo.clase)))
    return null;
  return Object.freeze({ version: catalogo.version, tipos: Object.freeze(catalogo.tipos.map((tipo) =>
    Object.freeze({ codigo: tipo.codigo, clase: tipo.clase }))) });
}

/** Rellena el selector de tipo agrupando por apartado y conserva el valor. */
export function pintarTiposOtroGasto(selector, catalogo, traducir, anterior = selector.value) {
  const documento = selector.ownerDocument;
  const inicial = nodo(documento, "option", traducir("otros_gastos_elegir_tipo"));
  inicial.value = "";
  const grupos = GRUPOS.map(([clase, clave]) => {
    const grupo = nodo(documento, "optgroup");
    grupo.label = traducir(clave);
    (catalogo?.tipos || []).filter((tipo) => tipo.clase === clase).forEach((tipo) => {
      const opcion = nodo(documento, "option", rotuloTipoOtroGasto(traducir, tipo.codigo));
      opcion.value = tipo.codigo;
      grupo.append(opcion);
    });
    return grupo;
  }).filter((grupo) => grupo.children.length > 0);
  selector.replaceChildren(inicial, ...grupos);
  selector.value = catalogo?.tipos?.some((tipo) => tipo.codigo === anterior) ? anterior : "";
}

/** Crea una línea editable. `valor` puede venir de una línea ya guardada. */
export function crearLineaOtroGasto(documento, { traducir, catalogo, valor = {}, fechaInicio = "", fechaFin = "" }) {
  const fila = nodo(documento, "div");
  fila.className = "dietas-comision-otro-linea";
  fila.dataset.dietasOtroLinea = "";
  const campo = (nombre, clave, tipo = "text", ajustes = {}) => {
    const etiqueta = nodo(documento, "label", traducir(clave));
    const entrada = nodo(documento, "input");
    entrada.name = nombre;
    entrada.type = tipo;
    entrada.required = true;
    entrada.value = valor[nombre] ?? "";
    Object.assign(entrada, ajustes);
    etiqueta.append(entrada);
    return etiqueta;
  };
  const etiquetaTipo = nodo(documento, "label", traducir("otros_gastos_tipo"));
  const tipo = nodo(documento, "select");
  tipo.name = "tipo_gasto";
  tipo.required = true;
  pintarTiposOtroGasto(tipo, catalogo, traducir, valor.tipo_gasto || "");
  etiquetaTipo.append(tipo);
  const fecha = campo("fecha", "otros_gastos_fecha", "date");
  const entradaFecha = fecha.querySelector("input");
  if (fechaCivil(fechaInicio)) entradaFecha.min = fechaInicio;
  if (fechaCivil(fechaFin)) entradaFecha.max = fechaFin;
  const importe = campo("importe", "comision_otros_importe", "text", { inputMode: "decimal" });
  const referencia = campo("justificante_ref", "otros_gastos_justificante_ref", "text",
    { maxLength: 128, pattern: "[A-Za-z][A-Za-z0-9:_\\-]{2,127}", autocomplete: "off", spellcheck: false });
  const huella = campo("justificante_sha256", "otros_gastos_justificante_sha256", "text",
    { minLength: 64, maxLength: 64, pattern: "[a-fA-F0-9]{64}", autocomplete: "off", spellcheck: false });
  huella.className = "dietas-comision-otro-huella";
  const etiquetaFichero = nodo(documento, "label", traducir("otros_gastos_calcular_huella"));
  etiquetaFichero.className = "dietas-comision-otro-fichero";
  const fichero = nodo(documento, "input");
  fichero.type = "file";
  fichero.dataset.dietasOtroFichero = "";
  etiquetaFichero.append(fichero);
  const quitar = nodo(documento, "button", traducir("comision_otros_quitar"));
  quitar.type = "button";
  quitar.className = "boton-secundario";
  quitar.dataset.dietasOtroQuitar = "";
  fila.append(etiquetaTipo, fecha, campo("concepto", "otros_gastos_descripcion", "text", { maxLength: 500, minLength: 3 }),
    importe, referencia, huella, etiquetaFichero, quitar);
  return fila;
}

/**
 * Lee las líneas del formulario en la forma que exige el servidor. Lanza
 * TypeError si alguna está incompleta; el servidor valida de nuevo.
 */
export function leerOtrosGastos(form, catalogo, { fechaInicio, fechaFin }) {
  const filas = form.querySelectorAll("[data-dietas-otro-linea]");
  if (filas.length > 0 && !catalogo) throw new TypeError("catálogo de otros gastos no disponible");
  return Array.from(filas).map((fila) => {
    const valor = (nombre) => String([...fila.querySelectorAll("input"), ...fila.querySelectorAll("select")]
      .find((entrada) => entrada.name === nombre)?.value || "").trim();
    const eurosTexto = valor("importe").replace(",", ".");
    if (!/^(?:0|[1-9]\d{0,5})(?:\.\d{1,2})?$/u.test(eurosTexto)) throw new TypeError("importe no válido");
    const [enteros, decimales = ""] = eurosTexto.split(".");
    const tipo = catalogo.tipos.find((entrada) => entrada.codigo === valor("tipo_gasto"));
    const fecha = valor("fecha");
    const linea = {
      tipo: tipo?.clase,
      tipo_gasto: tipo?.codigo,
      catalogo_version: catalogo.version,
      fecha,
      concepto: valor("concepto"),
      importe_centimos: Number(enteros) * 100 + Number(decimales.padEnd(2, "0")),
      justificante_ref: valor("justificante_ref"),
      justificante_sha256: valor("justificante_sha256").toLowerCase(),
    };
    if (!tipo || !fechaCivil(fecha) || fecha < fechaInicio || fecha > fechaFin || linea.concepto.length < 3 ||
        linea.importe_centimos < 1 || !REFERENCIA.test(linea.justificante_ref) || !HUELLA.test(linea.justificante_sha256))
      throw new TypeError("línea de otros gastos incompleta");
    return linea;
  });
}

/** Huella SHA-256 en hexadecimal de un fichero local, sin enviarlo. */
export async function calcularHuellaFichero(fichero, cripto = globalThis.crypto) {
  if (!fichero || typeof fichero.arrayBuffer !== "function" || !Number.isSafeInteger(fichero.size) ||
      fichero.size < 1 || fichero.size > MAXIMO_FICHERO_BYTES || typeof cripto?.subtle?.digest !== "function")
    throw new TypeError("fichero no válido");
  const resumen = await cripto.subtle.digest("SHA-256", await fichero.arrayBuffer());
  return Array.from(new Uint8Array(resumen), (octeto) => octeto.toString(16).padStart(2, "0")).join("");
}

/** Texto de una línea guardada: tipo, fecha y descripción, sin códigos. */
export function describirOtroGasto(linea, traducir, formatoFecha) {
  if (typeof linea?.tipo_gasto !== "string") return String(linea?.concepto ?? "");
  return traducir("otros_gastos_linea", { tipo: rotuloTipoOtroGasto(traducir, linea.tipo_gasto),
    fecha: formatoFecha(linea.fecha), descripcion: linea.concepto });
}
