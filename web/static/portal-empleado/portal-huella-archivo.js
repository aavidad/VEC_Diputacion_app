/**
 * Huella de un justificante calculada en el propio equipo.
 *
 * Los formularios que registran un documento que permanece en custodia de la
 * persona (Bolsa: operaciones y sanciones; Contratación temporal: cese) solo
 * envían su referencia legible y su huella SHA-256. Nadie puede teclear una
 * huella: se elige el archivo, el navegador la calcula con
 * `crypto.subtle.digest` y la deja en un campo oculto. El contenido nunca sale
 * del equipo ni se conserva: los bytes leídos se ponen a cero al terminar.
 *
 * Uso:
 *  - `renderizarCampoHuellaArchivo(...)` pinta el control de archivo, el campo
 *    oculto con el nombre que espera el servidor y la línea de estado;
 *  - `instalarHuellaArchivo(raiz)` escucha los cambios de archivo y, en fase de
 *    captura, bloquea el envío de un formulario cuya huella obligatoria falta
 *    o aún se está calculando, con un mensaje legible y el foco en el control.
 */

/**
 * Tamaño máximo que se lee en memoria para calcular la huella: 20 MiB.
 * Constante técnica, no regla funcional: cubre un documento administrativo
 * escaneado y evita reservar memoria desproporcionada en el navegador. Se
 * comprueba con `File.size` antes de leer un solo byte.
 */
export const LIMITE_HUELLA_ARCHIVO_BYTES = 20 * 1024 * 1024;

export const MENSAJES_HUELLA_ARCHIVO_ES = Object.freeze({
  etiqueta: "Archivo del justificante",
  comprobado: "Documento comprobado en este equipo; su contenido no se guarda.",
  calculando: "Comprobando el documento en este equipo…",
  sin_archivo: "Elija el archivo del justificante para poder continuar.",
  esperando: "Espere a que termine la comprobación del documento.",
  demasiado_grande: "El archivo supera el tamaño máximo de {limite} MB. Elija una copia más ligera del documento.",
  vacio: "El archivo está vacío. Elija el documento correcto.",
  no_disponible: "Este navegador no permite comprobar el documento. Use un navegador actualizado.",
  error_lectura: "No se ha podido leer el archivo. Vuelva a elegirlo.",
  ayuda_aria: "Ayuda sobre el justificante",
  ayuda: "El archivo del justificante se lee solo en este equipo para calcular su huella digital, que es lo único que se registra junto a su referencia. El documento no se envía ni se guarda en VEC y sigue en su custodia. Tamaño máximo: {limite} MB.",
});

export function crearTraductorHuellaArchivo(catalogo = MENSAJES_HUELLA_ARCHIVO_ES) {
  return (clave, valores = {}) => {
    const plantilla = typeof catalogo?.[clave] === "string" ? catalogo[clave] : MENSAJES_HUELLA_ARCHIVO_ES[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, nombre) => (Object.hasOwn(valores, nombre) ? String(valores[nombre]) : `{${nombre}}`));
  };
}

export const traducirHuellaArchivo = crearTraductorHuellaArchivo();

const HEX_SHA256 = /^[0-9a-f]{64}$/u;
const megas = (bytes) => new Intl.NumberFormat("es-ES", { maximumFractionDigits: 0 }).format(bytes / (1024 * 1024));

/** Texto de ayuda de la pantalla que usa el control, con el límite vigente. */
export function ayudaHuellaArchivo(t = traducirHuellaArchivo, limite = LIMITE_HUELLA_ARCHIVO_BYTES) {
  return t("ayuda", { limite: megas(limite) });
}

/**
 * Calcula la huella SHA-256 en hexadecimal de un `File`/`Blob`.
 * Devuelve `{ ok: true, huella }` o `{ ok: false, motivo }` con motivo
 * `sin_archivo`, `vacio`, `demasiado_grande`, `no_disponible` o `error_lectura`.
 */
export async function calcularHuellaArchivo(archivo, { limite = LIMITE_HUELLA_ARCHIVO_BYTES, criptografia = globalThis.crypto } = {}) {
  if (!archivo || typeof archivo.arrayBuffer !== "function") return { ok: false, motivo: "sin_archivo" };
  if (!Number.isSafeInteger(archivo.size) || archivo.size < 1) return { ok: false, motivo: "vacio" };
  if (archivo.size > limite) return { ok: false, motivo: "demasiado_grande" };
  if (typeof criptografia?.subtle?.digest !== "function") return { ok: false, motivo: "no_disponible" };
  let bytes;
  try {
    bytes = new Uint8Array(await archivo.arrayBuffer());
    if (bytes.byteLength !== archivo.size) return { ok: false, motivo: "error_lectura" };
    const resumen = new Uint8Array(await criptografia.subtle.digest("SHA-256", bytes));
    const huella = Array.from(resumen, (b) => b.toString(16).padStart(2, "0")).join("");
    return resumen.length === 32 && HEX_SHA256.test(huella) ? { ok: true, huella } : { ok: false, motivo: "error_lectura" };
  } catch {
    return { ok: false, motivo: "error_lectura" };
  } finally {
    bytes?.fill(0);
  }
}

function escaparBasico(valor) {
  return String(valor ?? "").replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
}

/**
 * Control de archivo + campo oculto `nombre` + estado. Si `huella` ya es
 * válida (se eligió antes y el formulario se volvió a pintar), se conserva y
 * el estado dice que el documento está comprobado.
 */
export function renderizarCampoHuellaArchivo({
  id, nombre = "sha256", huella = "", obligatorio = true, etiqueta, deshabilitado = false,
  clase = "campo", escapar = escaparBasico, t = traducirHuellaArchivo,
} = {}) {
  if (typeof id !== "string" || !/^[A-Za-z][\w-]*$/u.test(id)) throw new TypeError("identificador de campo no válido");
  const valida = HEX_SHA256.test(String(huella || ""));
  const estado = `${id}-estado`;
  return `<div class="campo-huella-archivo" data-huella-campo>`
    + `<label class="${escapar(clase)}" for="${escapar(id)}"><span>${escapar(etiqueta || t("etiqueta"))}</span>`
    + `<input id="${escapar(id)}" type="file" data-huella-archivo aria-describedby="${escapar(estado)}"`
    + `${obligatorio ? ' aria-required="true" data-huella-obligatoria' : ""}${deshabilitado ? " disabled" : ""}></label>`
    + `<input type="hidden" name="${escapar(nombre)}" value="${valida ? escapar(huella) : ""}" data-huella-valor>`
    + `<small id="${escapar(estado)}" class="estado-huella-archivo" role="status" data-huella-estado>${valida ? escapar(t("comprobado")) : ""}</small>`
    + "</div>";
}

function partesDe(control) {
  const campo = control?.closest?.("[data-huella-campo]");
  return { oculto: campo?.querySelector?.("[data-huella-valor]") ?? null, estado: campo?.querySelector?.("[data-huella-estado]") ?? null };
}

function marcarError(control, estado, texto) {
  control.setAttribute("aria-invalid", "true");
  if (estado) estado.textContent = texto;
}

/**
 * Revisa los controles de huella de un formulario antes de enviarlo. Devuelve
 * el primer mensaje de error (y deja el foco en su control) o "" si puede
 * enviarse. Un control deshabilitado no bloquea.
 */
export function comprobarHuellasFormulario(formulario, t = traducirHuellaArchivo) {
  for (const control of formulario?.querySelectorAll?.("[data-huella-archivo]") ?? []) {
    if (control.disabled) continue;
    const { oculto, estado } = partesDe(control);
    let motivo = "";
    if (control.dataset.huellaCalculando === "true") motivo = "esperando";
    else if (control.hasAttribute("data-huella-obligatoria") && !HEX_SHA256.test(oculto?.value || "")) motivo = "sin_archivo";
    if (!motivo) continue;
    const texto = t(motivo);
    marcarError(control, estado, texto);
    control.focus?.();
    return texto;
  }
  return "";
}

const RAICES = new WeakSet();

/**
 * Instala, una sola vez por raíz, el cálculo al elegir archivo y el bloqueo de
 * envío. Devuelve la función que lo retira.
 */
export function instalarHuellaArchivo(raiz = globalThis.document, {
  t = traducirHuellaArchivo, limite = LIMITE_HUELLA_ARCHIVO_BYTES, criptografia = globalThis.crypto,
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function" || RAICES.has(raiz)) return () => {};
  RAICES.add(raiz);

  async function alCambiar(evento) {
    const control = evento.target?.closest?.("[data-huella-archivo]");
    if (!control) return;
    const { oculto, estado } = partesDe(control);
    if (!oculto) return;
    const lectura = String((Number(control.dataset.huellaLectura) || 0) + 1);
    control.dataset.huellaLectura = lectura;
    oculto.value = "";
    control.removeAttribute("aria-invalid");
    const archivo = control.files?.length === 1 ? control.files[0] : null;
    if (!archivo) {
      delete control.dataset.huellaCalculando;
      if (estado) estado.textContent = "";
      return;
    }
    control.dataset.huellaCalculando = "true";
    if (estado) estado.textContent = t("calculando");
    const resultado = await calcularHuellaArchivo(archivo, { limite, criptografia });
    if (control.dataset.huellaLectura !== lectura) return;
    delete control.dataset.huellaCalculando;
    if (resultado.ok) {
      oculto.value = resultado.huella;
      if (estado) estado.textContent = t("comprobado");
      return;
    }
    control.value = "";
    marcarError(control, estado, t(resultado.motivo, { limite: megas(limite) }));
  }

  function alEnviar(evento) {
    const formulario = evento.target;
    if (!formulario?.querySelectorAll || !comprobarHuellasFormulario(formulario, t)) return;
    evento.preventDefault();
    evento.stopImmediatePropagation();
  }

  raiz.addEventListener("change", alCambiar);
  raiz.addEventListener("submit", alEnviar, true);
  return () => {
    raiz.removeEventListener("change", alCambiar);
    raiz.removeEventListener("submit", alEnviar, true);
    RAICES.delete(raiz);
  };
}
