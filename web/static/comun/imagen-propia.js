/**
 * «Mi imagen» (Usuarios 5.08c), común al portal de RRHH y al Área personal.
 *
 * La persona elige cómo se la identifica en la cabecera: sus iniciales, un
 * icono de la lista o una foto suya, sobre un color de fondo. La foto la
 * recodifica el servidor (JPEG de 256 px sin datos ocultos) y solo la ve su
 * titular. Aquí no se guarda nada en el navegador; los textos viven en
 * `textos/<idioma>/preferencias.json` (sección `imagen`).
 */
import { cargarTextos } from "./textos.js";

const RUTAS = Object.freeze(["/api/vec/usuarios/mi-imagen", "/api/vec/usuarios/area-personal/mi-imagen"]);
const MAX_RESPUESTA = 512 * 1024;
const LIMITE_MS = 30000;
const PATRON_CLAVE = /^[A-Za-z0-9:_.-]{16,128}$/u;
const PATRON_BASE64 = /^[A-Za-z0-9+/]+={0,2}$/u;
const CODIGOS_ERROR = new Set(["no_autenticado", "prohibido", "conflicto", "peticion_invalida", "no_disponible", "foto_grande", "foto_no_admitida"]);
export const MODOS_IMAGEN = Object.freeze(["iniciales", "icono", "foto"]);
export const PALETAS_IMAGEN = Object.freeze(["azul", "turquesa", "verde", "naranja", "morado", "gris"]);
// Una foto de hasta 20 MB se reduce en el navegador (lado máximo 1024 px)
// antes de enviarla: el servidor admite 1,4 MB, que en base64 caben en el
// límite común de 2 MB por petición, y es quien la recodifica de verdad.
export const TAMANO_MAXIMO_SELECCION = 20 * 1024 * 1024;
export const TAMANO_MAXIMO_ENVIO = 1400 * 1024;
const LADO_ENVIO = 1024;
export const TIPOS_FOTO = Object.freeze(["image/jpeg", "image/png", "image/webp"]);

// Trazos cerrados de los iconos del avatar (24×24, línea con currentColor).
const TRAZOS_ICONO = Object.freeze({
  persona: ["M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8z", "M4 20c1.5-4 4.5-6 8-6s6.5 2 8 6"],
  estrella: ["M12 3l2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1L3.2 9.5l6.1-.9z"],
  hoja: ["M5 19c0-8 5-14 15-15-1 10-7 15-15 15z", "M5 19l7-7"],
  sol: ["M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8z", "M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"],
  corazon: ["M12 20s-7-4.4-9-9a4.5 4.5 0 0 1 9-3 4.5 4.5 0 0 1 9 3c-2 4.6-9 9-9 9z"],
  libro: ["M4 4.5A1.5 1.5 0 0 1 5.5 3H20v15H5.5A1.5 1.5 0 0 0 4 19.5z", "M4 19.5A1.5 1.5 0 0 0 5.5 21H20"],
  cafe: ["M4 8h13v5a5 5 0 0 1-5 5H9a5 5 0 0 1-5-5z", "M17 10h1.5a2.5 2.5 0 0 1 0 5H17", "M8 3v2M12 3v2"],
  montana: ["M3 20l6-10 4 6 3-4 5 8z", "M17 7.5a1.5 1.5 0 1 0-3 0 1.5 1.5 0 0 0 3 0z"],
});
export const ICONOS_IMAGEN = Object.freeze(Object.keys(TRAZOS_ICONO));

export class ErrorImagen extends Error {
  constructor(estado = 0, codigo = "") {
    super(`imagen HTTP ${estado}`);
    this.estado = estado;
    this.codigo = codigo;
  }
}

function objeto(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor); }

/** Comprueba una elección del vocabulario cerrado y devuelve una copia congelada. */
export function validarEleccion(eleccion) {
  if (!objeto(eleccion) || !MODOS_IMAGEN.includes(eleccion.modo) || !PALETAS_IMAGEN.includes(eleccion.paleta)
    || typeof eleccion.icono !== "string" || (eleccion.modo === "icono") !== ICONOS_IMAGEN.includes(eleccion.icono)
    || (eleccion.modo !== "icono" && eleccion.icono !== "")) throw new TypeError("elección de imagen inválida");
  return Object.freeze({ modo: eleccion.modo, paleta: eleccion.paleta, icono: eleccion.icono });
}

function validarOpciones(lista, permitidos) {
  if (!Array.isArray(lista) || lista.length === 0 || lista.length > permitidos.length
    || lista.some((o) => !objeto(o) || !permitidos.includes(o.codigo))) throw new TypeError("catálogo de imagen inválido");
  return Object.freeze(lista.map((o) => o.codigo));
}

function validarVista(datos) {
  if (!objeto(datos) || !objeto(datos.catalogo) || !objeto(datos.estado) || typeof datos.catalogo.version_ref !== "string") throw new TypeError("imagen inválida");
  const catalogo = Object.freeze({ version_ref: datos.catalogo.version_ref,
    paletas: validarOpciones(datos.catalogo.paletas, PALETAS_IMAGEN), iconos: validarOpciones(datos.catalogo.iconos, ICONOS_IMAGEN) });
  const { version, catalogo_version_ref: versionCatalogo } = datos.estado;
  if (!Number.isSafeInteger(version) || version < 0 || typeof versionCatalogo !== "string") throw new TypeError("estado de imagen inválido");
  const eleccion = validarEleccion(datos.estado.eleccion);
  let foto = null;
  if (datos.foto !== null && datos.foto !== undefined) {
    if (!objeto(datos.foto) || datos.foto.tipo !== "image/jpeg" || typeof datos.foto.datos !== "string"
      || datos.foto.datos.length > 400000 || !PATRON_BASE64.test(datos.foto.datos) || eleccion.modo !== "foto") throw new TypeError("foto inválida");
    foto = Object.freeze({ tipo: "image/jpeg", datos: datos.foto.datos });
  }
  return Object.freeze({ catalogo, estado: Object.freeze({ version, catalogo_version_ref: versionCatalogo, eleccion }), foto });
}

async function contenidoJSON(respuesta) {
  const tipo = respuesta.headers?.get?.("content-type") || "";
  if (!tipo.toLowerCase().includes("application/json")) throw new TypeError("respuesta de imagen no JSON");
  if (Number(respuesta.headers?.get?.("content-length") || 0) > MAX_RESPUESTA) throw new TypeError("respuesta de imagen demasiado grande");
  const texto = await respuesta.text();
  if (texto.length > MAX_RESPUESTA) throw new TypeError("respuesta de imagen demasiado grande");
  return JSON.parse(texto);
}

/**
 * Envuelve `fetch` para que las peticiones a las rutas de Usuarios salgan de
 * una en una: la identidad de desarrollo no admite dos altas de sesión
 * simultáneas de la misma cuenta. Lee el cuerpo con el límite de su ruta antes
 * de dar paso a la siguiente; tampoco retiene una respuesta sin longitud.
 */
export function peticionesEnSerie(fetchImpl = globalThis.fetch) {
  if (typeof fetchImpl !== "function") throw new TypeError("fetch no disponible");
  const limites = new Map([
    ["/api/vec/usuarios/mi-imagen", MAX_RESPUESTA],
    ["/api/vec/usuarios/area-personal/mi-imagen", MAX_RESPUESTA],
    ["/api/vec/usuarios/mis-preferencias", 65536],
    ["/api/vec/usuarios/mis-correos", 65536],
    ["/api/vec/usuarios/area-personal/mis-correos", 65536],
  ]);
  let cola = Promise.resolve();
  return function fetchEnSerie(recurso, opciones) {
    const turno = cola.then(async () => {
      const respuesta = await fetchImpl(recurso, opciones);
      const sinCuerpo = [101, 204, 205, 304].includes(respuesta.status);
      if (sinCuerpo) return new Response(null, { status: respuesta.status, statusText: respuesta.statusText, headers: respuesta.headers });
      const limite = limites.get(recurso) ?? 65536;
      const longitud = respuesta.headers?.get?.("content-length");
      if (longitud != null && (!/^\d+$/u.test(longitud) || Number(longitud) > limite)) {
        try { await respuesta.body?.cancel?.(); } catch { /* La respuesta no se usará. */ }
        throw new TypeError("respuesta_serie_demasiado_grande");
      }
      if (!respuesta.body?.getReader) throw new TypeError("respuesta_serie_sin_flujo");
      const lector = respuesta.body.getReader();
      const partes = [];
      let total = 0;
      let abortar;
      const cancelada = new Promise((_resolve, reject) => {
        abortar = () => reject(new TypeError("respuesta_serie_cancelada"));
        opciones?.signal?.addEventListener?.("abort", abortar, { once: true });
      });
      const temporizador = setTimeout(abortar, LIMITE_MS);
      try {
        if (opciones?.signal?.aborted) throw new TypeError("respuesta_serie_cancelada");
        for (;;) {
          const { done, value } = await Promise.race([lector.read(), cancelada]);
          if (done) break;
          total += value.byteLength;
          if (total > limite) throw new TypeError("respuesta_serie_demasiado_grande");
          partes.push(value);
        }
      } finally {
        clearTimeout(temporizador);
        opciones?.signal?.removeEventListener?.("abort", abortar);
        try { await lector.cancel(); } catch { /* La cola debe liberarse tras el descarte. */ }
      }
      const cuerpo = new Uint8Array(total);
      let posicion = 0;
      for (const parte of partes) { cuerpo.set(parte, posicion); posicion += parte.byteLength; }
      return new Response(cuerpo, { status: respuesta.status, statusText: respuesta.statusText, headers: respuesta.headers });
    });
    cola = turno.then(() => undefined, () => undefined);
    return turno;
  };
}

/** Cliente HTTP de la ruta exacta de cada portal. */
export function crearClienteImagen({ ruta, fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function" || !RUTAS.includes(ruta)) throw new TypeError("cliente de imagen no disponible");
  async function solicitar(metodo, cuerpo, signal) {
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    if (signal?.aborted) abortar();
    else signal?.addEventListener?.("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, LIMITE_MS);
    try {
      const respuesta = await fetchImpl(ruta, {
        method: metodo, credentials: "same-origin", mode: "same-origin", redirect: "error",
        cache: "no-store", referrerPolicy: "no-referrer", signal: controlador.signal,
        headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
        ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
      });
      if (!respuesta.ok) {
        let codigo = "";
        try {
          const error = (await contenidoJSON(respuesta))?.error;
          if (CODIGOS_ERROR.has(error?.codigo) && error.clave_i18n === `api.usuarios.imagen.error.${error.codigo}`) codigo = error.codigo;
        } catch { /* El estado HTTP basta para responder con seguridad. */ }
        // El límite común del servidor responde 413 sin cuerpo JSON.
        if (!codigo && respuesta.status === 413) codigo = "foto_grande";
        throw new ErrorImagen(respuesta.status, codigo);
      }
      return await contenidoJSON(respuesta);
    } catch (error) {
      if (signal?.aborted) throw error;
      if (error instanceof ErrorImagen) throw error;
      throw new ErrorImagen(0);
    } finally {
      clearTimeout(temporizador);
      signal?.removeEventListener?.("abort", abortar);
    }
  }
  return Object.freeze({
    async consultar({ signal } = {}) {
      return validarVista((await solicitar("GET", null, signal))?.data);
    },
    async guardar(cuerpo, { signal } = {}) {
      if (!objeto(cuerpo) || !["elegir", "subir_foto"].includes(cuerpo.operacion) || !Number.isSafeInteger(cuerpo.version_esperada)
        || cuerpo.version_esperada < 0 || !PATRON_CLAVE.test(cuerpo.clave_operacion) || typeof cuerpo.catalogo_version_ref !== "string"
        || (cuerpo.operacion === "subir_foto") !== (typeof cuerpo.foto_base64 === "string")) throw new TypeError("operación de imagen inválida");
      validarEleccion(cuerpo.eleccion);
      const recibo = (await solicitar("POST", cuerpo, signal))?.data;
      if (!objeto(recibo) || typeof recibo.recibo_ref !== "string" || recibo.recibo_ref.length === 0
        || !Number.isSafeInteger(recibo.version) || recibo.version < 1) throw new TypeError("recibo de imagen inválido");
      return Object.freeze({ recibo_ref: recibo.recibo_ref, version: recibo.version, replay: recibo.replay === true,
        foto_nueva: recibo.foto_nueva === true, foto_retirada: recibo.foto_retirada === true });
    },
  });
}

const SVG = "http://www.w3.org/2000/svg";

/** SVG decorativo de un icono del catálogo; nunca procede del servidor ni del cliente. */
export function crearIconoAvatar(documento, codigo) {
  const svg = documento.createElementNS(SVG, "svg");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("aria-hidden", "true");
  svg.setAttribute("focusable", "false");
  svg.setAttribute("class", "avatar-imagen-icono");
  for (const d of TRAZOS_ICONO[codigo] ?? TRAZOS_ICONO.persona) {
    const trazo = documento.createElementNS(SVG, "path");
    trazo.setAttribute("d", d);
    svg.append(trazo);
  }
  return svg;
}

function iconoHTML(codigo) {
  const trazos = (TRAZOS_ICONO[codigo] ?? TRAZOS_ICONO.persona).map((d) => `<path d="${d}"/>`).join("");
  return `<svg class="avatar-imagen-icono" viewBox="0 0 24 24" aria-hidden="true" focusable="false">${trazos}</svg>`;
}

/**
 * Pinta un avatar según la elección. `iniciales` vacías conservan el
 * contenido que ya tenga el elemento (por ejemplo, el icono genérico).
 */
export function pintarAvatar(elemento, { eleccion, foto = null, iniciales = "" } = {}) {
  if (!elemento?.ownerDocument || !eleccion) return;
  const documento = elemento.ownerDocument;
  for (const paleta of PALETAS_IMAGEN) elemento.classList.remove(`avatar-imagen--${paleta}`);
  elemento.classList.add("avatar-imagen", `avatar-imagen--${eleccion.paleta}`);
  elemento.dataset.imagenModo = eleccion.modo;
  if (eleccion.modo === "foto" && foto?.datos) {
    const img = documento.createElement("img");
    img.alt = "";
    img.decoding = "async";
    img.src = `data:image/jpeg;base64,${foto.datos}`;
    elemento.replaceChildren(img);
  } else if (eleccion.modo === "icono") {
    elemento.replaceChildren(crearIconoAvatar(documento, eleccion.icono));
  } else if (iniciales) {
    elemento.replaceChildren(documento.createTextNode(iniciales));
  }
}

/** Controla el avatar de la cabecera: se repinta al llegar las iniciales o la imagen. */
export function crearAvatarCabecera(elemento) {
  let iniciales = "";
  let vista = null;
  function repintar() {
    if (!elemento) return;
    if (vista) pintarAvatar(elemento, { eleccion: vista.estado.eleccion, foto: vista.foto, iniciales });
    else if (iniciales) elemento.textContent = iniciales;
  }
  return Object.freeze({
    fijarIniciales(texto) {
      const nuevas = String(texto ?? "").slice(0, 3);
      if (nuevas === iniciales) return;
      iniciales = nuevas;
      repintar();
    },
    fijarImagen(nueva) { vista = nueva ?? null; repintar(); },
  });
}

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function base64DeBytes(bytes) {
  let binario = "";
  for (let i = 0; i < bytes.length; i += 0x8000) binario += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(binario);
}

/**
 * Prepara el fichero elegido para enviarlo: si el navegador sabe, lo reduce
 * (respetando la orientación) y lo pasa a JPEG; si no, lo envía tal cual
 * cuando cabe. Rechaza por tipo declarado o tamaño con el código de la API.
 */
export async function prepararFoto(fichero, entorno = globalThis) {
  if (!fichero || !TIPOS_FOTO.includes(fichero.type)) throw new ErrorImagen(0, "foto_no_admitida");
  if (fichero.size > TAMANO_MAXIMO_SELECCION) throw new ErrorImagen(0, "foto_grande");
  if (typeof entorno.createImageBitmap === "function") {
    // Si el navegador sabe leer fotos y no puede con esta, no es una foto válida.
    let mapa;
    try { mapa = await entorno.createImageBitmap(fichero, { imageOrientation: "from-image" }); }
    catch { throw new ErrorImagen(0, "foto_no_admitida"); }
    try {
      const escala = Math.min(1, LADO_ENVIO / Math.max(mapa.width, mapa.height));
      const ancho = Math.max(1, Math.round(mapa.width * escala));
      const alto = Math.max(1, Math.round(mapa.height * escala));
      const lienzo = typeof entorno.OffscreenCanvas === "function" ? new entorno.OffscreenCanvas(ancho, alto)
        : Object.assign(entorno.document.createElement("canvas"), { width: ancho, height: alto });
      const pincel = lienzo.getContext("2d");
      pincel.fillStyle = "white";
      pincel.fillRect(0, 0, ancho, alto);
      pincel.drawImage(mapa, 0, 0, ancho, alto);
      mapa.close?.();
      const blob = typeof lienzo.convertToBlob === "function" ? await lienzo.convertToBlob({ type: "image/jpeg", quality: 0.9 })
        : await new Promise((resolver) => lienzo.toBlob(resolver, "image/jpeg", 0.9));
      if (blob && blob.size > 0 && blob.size <= TAMANO_MAXIMO_ENVIO) {
        return Object.freeze({ tipo: "image/jpeg", base64: base64DeBytes(new Uint8Array(await blob.arrayBuffer())), nombre: nombreFichero(fichero) });
      }
    } catch { /* Sin lienzo disponible se intenta con el original. */ }
  }
  if (fichero.size > TAMANO_MAXIMO_ENVIO) throw new ErrorImagen(0, "foto_grande");
  return Object.freeze({ tipo: fichero.type, base64: base64DeBytes(new Uint8Array(await fichero.arrayBuffer())), nombre: nombreFichero(fichero) });
}

function nombreFichero(fichero) {
  const nombre = String(fichero?.name ?? "").replace(/[\p{Cc}\p{Cf}]/gu, "").trim();
  return nombre.length > 60 ? `${nombre.slice(0, 57)}…` : nombre;
}

const MARCO_PREDETERMINADO = Object.freeze({ panel: "panel", cabecera: "div", claseCabecera: "cabecera-panel", cuerpo: "cuerpo-panel" });

/**
 * Superficie de «Mi imagen» dentro de Mis preferencias. `marco` adapta las
 * clases del panel a cada portal; `alCambiar` recibe cada vista cargada para
 * repintar la cabecera; `iniciales` da el texto del modo iniciales.
 */
export function crearSuperficieImagen({ cliente, textos, marco = MARCO_PREDETERMINADO, aleatorio = globalThis.crypto,
  alCambiar = () => {}, iniciales = () => "", cargaAlMostrar = false, entorno = globalThis } = {}) {
  if (!cliente || typeof textos?.traducir !== "function") throw new TypeError("superficie de imagen incompleta");
  const t = (clave, variables) => textos.traducir(`imagen.${clave}`, variables);
  let datos = null;
  let carga = "sin_cargar";
  let errorCarga = null;
  let mensaje = null;
  let ocupado = false;
  let pendiente = null;
  let ayudaVisible = false;
  let borrador = null;
  let archivo = null;
  let errorFoto = false;
  let enfocarTras = "";
  let controlador = null;
  let generacion = 0;
  let contenedor = null;

  const raiz = () => contenedor?.querySelector?.("[data-imagen-raiz]") ?? null;
  function repintar() {
    const elemento = raiz();
    if (!elemento) return;
    elemento.innerHTML = interior();
    elemento.toggleAttribute?.("aria-busy", ocupado);
    const destino = enfocarTras ? elemento.querySelector(enfocarTras) : null;
    enfocarTras = "";
    destino?.focus?.({ preventScroll: false });
  }

  async function cargar({ conservarMensaje = false } = {}) {
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++generacion;
    carga = "cargando";
    errorCarga = null;
    if (!conservarMensaje) mensaje = null;
    repintar();
    try {
      const nuevos = await cliente.consultar({ signal: controlador.signal });
      if (actual !== generacion) return;
      datos = nuevos;
      carga = "lista";
      borrador = null;
      archivo = null;
      alCambiar(nuevos);
    } catch (error) {
      if (actual !== generacion || controlador.signal.aborted) return;
      datos = null;
      carga = "error";
      errorCarga = error;
    }
    repintar();
  }

  function eleccionActual() {
    return borrador ?? datos?.estado.eleccion ?? null;
  }

  function textoError(error) {
    switch (error?.codigo) {
      case "no_autenticado": return t("error_sesion");
      case "prohibido": return t("error_denegado");
      case "conflicto": return t("error_conflicto");
      case "foto_grande": return t("error_foto_grande");
      case "foto_no_admitida": return t("error_foto_no_admitida");
      case "peticion_invalida": return t("error_peticion");
      default: return t("error_incierto");
    }
  }

  async function enviar(cuerpo) {
    if (ocupado) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++generacion;
    ocupado = true;
    pendiente = null;
    mensaje = { tipo: "estado", texto: t(cuerpo.operacion === "subir_foto" ? "subiendo" : "guardando") };
    repintar();
    try {
      const recibo = await cliente.guardar(cuerpo, { signal: controlador.signal });
      if (actual !== generacion) return;
      ocupado = false;
      await cargar({ conservarMensaje: true });
      mensaje = { tipo: "exito", texto: recibo.foto_retirada && cuerpo.eleccion.modo !== "foto" ? t("exito_sin_foto") : t("exito") };
      enfocarTras = "[data-imagen-mensaje]";
    } catch (error) {
      if (actual !== generacion || controlador.signal.aborted) return;
      ocupado = false;
      if (!error?.codigo || error.codigo === "no_disponible") pendiente = cuerpo;
      // Una foto rechazada no se queda como vista previa.
      if (["foto_no_admitida", "foto_grande"].includes(error?.codigo)) archivo = null;
      mensaje = { tipo: "error", texto: textoError(error) };
      enfocarTras = "[data-imagen-mensaje]";
      if (error?.codigo === "conflicto") await cargar({ conservarMensaje: true });
    }
    repintar();
  }

  function guardar() {
    const eleccion = eleccionActual();
    if (!datos || !eleccion || ocupado) return;
    if (eleccion.modo === "foto" && !archivo && !datos.foto) {
      mensaje = { tipo: "error", texto: t("error_sin_foto") };
      errorFoto = true;
      enfocarTras = "#imagen-archivo";
      repintar();
      return;
    }
    const uuid = aleatorio?.randomUUID?.();
    if (!uuid) {
      mensaje = { tipo: "error", texto: t("error_incierto") };
      enfocarTras = "[data-imagen-mensaje]";
      repintar();
      return;
    }
    const cuerpo = { operacion: archivo && eleccion.modo === "foto" ? "subir_foto" : "elegir", version_esperada: datos.estado.version,
      catalogo_version_ref: datos.catalogo.version_ref, clave_operacion: `web-imagen-${uuid}`, eleccion };
    if (cuerpo.operacion === "subir_foto") cuerpo.foto_base64 = archivo.base64;
    void enviar(cuerpo);
  }

  async function elegirArchivo(fichero) {
    if (!fichero || ocupado) return;
    ocupado = true;
    errorFoto = false;
    mensaje = { tipo: "estado", texto: t("preparando") };
    repintar();
    try {
      archivo = await prepararFoto(fichero, entorno);
      mensaje = { tipo: "estado", texto: t("foto_lista") };
    } catch (error) {
      archivo = null;
      mensaje = { tipo: "error", texto: textoError(error?.codigo ? error : { codigo: "foto_no_admitida" }) };
    }
    ocupado = false;
    enfocarTras = "[data-imagen-mensaje]";
    repintar();
  }

  function vistaPrevia(eleccion) {
    const clase = `avatar-imagen avatar-imagen--grande avatar-imagen--${escapar(eleccion.paleta)}`;
    let contenido = escapar(iniciales() || "");
    if (eleccion.modo === "foto" && (archivo || datos?.foto)) {
      const fuente = archivo ? `data:${archivo.tipo};base64,${archivo.base64}` : `data:image/jpeg;base64,${datos.foto.datos}`;
      contenido = `<img alt="" src="${escapar(fuente)}">`;
    } else if (eleccion.modo === "icono") contenido = iconoHTML(eleccion.icono);
    else if (!contenido) contenido = iconoHTML("persona");
    const color = t(`paleta_${eleccion.paleta}`).toLocaleLowerCase();
    const descripcion = eleccion.modo === "icono" ? t("vista_accesible_icono", { icono: t(`icono_${eleccion.icono}`).toLocaleLowerCase(), color })
      : eleccion.modo === "foto" && (archivo || datos?.foto) ? t("vista_accesible_foto") : t("vista_accesible_iniciales", { color });
    return `<div class="imagen-vista"><span class="${clase}" data-imagen-modo="${escapar(eleccion.modo)}" aria-hidden="true">${contenido}</span><p class="imagen-vista-texto"><span class="imagen-solo-lectura">${escapar(descripcion)} </span>${escapar(t(archivo || borrador ? "vista_nueva" : "vista"))}</p></div>`;
  }

  function radio(nombre, valor, marcado, etiqueta, extra = "") {
    const id = `imagen-${nombre}-${valor}`;
    return `<label class="imagen-opcion imagen-opcion--${nombre}" for="${id}"><input type="radio" id="${id}" name="imagen-${nombre}" value="${escapar(valor)}"${marcado ? " checked" : ""}${ocupado ? " disabled" : ""}>${extra}<span>${escapar(etiqueta)}</span></label>`;
  }

  function formulario() {
    const e = eleccionActual();
    const modos = MODOS_IMAGEN.map((m) => radio("modo", m, e.modo === m, t(`modo_${m}`))).join("");
    let detalle = "";
    if (e.modo === "icono") {
      detalle = `<fieldset class="imagen-grupo imagen-grupo--iconos"><legend>${escapar(t("icono_titulo"))}</legend><div class="imagen-rejilla">${datos.catalogo.iconos.map((i) => radio("icono", i, e.icono === i, t(`icono_${i}`), `<span class="avatar-imagen avatar-imagen--muestra avatar-imagen--${escapar(e.paleta)}" aria-hidden="true">${iconoHTML(i)}</span>`)).join("")}</div></fieldset>`;
    } else if (e.modo === "foto") {
      const estadoFoto = archivo ? t("foto_elegida", { nombre: archivo.nombre || "—" }) : datos.foto ? t("foto_actual") : t("foto_ninguna");
      const error = errorFoto ? `<p id="imagen-foto-error" class="imagen-error-campo">${escapar(t("error_sin_foto"))}</p>` : "";
      const descrita = `imagen-foto-estado imagen-foto-limites${errorFoto ? " imagen-foto-error" : ""}`;
      detalle = `<div class="imagen-foto"><input type="file" id="imagen-archivo" class="imagen-archivo-oculto" name="foto" accept="${TIPOS_FOTO.join(",")}" aria-describedby="${descrita}"${errorFoto ? ' aria-invalid="true"' : ""}${ocupado ? " disabled" : ""}><label for="imagen-archivo" class="boton-secundario imagen-archivo-boton">${escapar(t(datos.foto ? "foto_cambiar" : "foto_elegir"))}</label><p id="imagen-foto-estado" class="imagen-nota">${escapar(estadoFoto)}</p><p id="imagen-foto-limites" class="imagen-nota">${escapar(t("foto_limites"))}</p>${error}</div>`;
    }
    const paletas = datos.catalogo.paletas.map((p) => radio("paleta", p, e.paleta === p, t(`paleta_${p}`), `<span class="imagen-muestra avatar-imagen--${escapar(p)}" aria-hidden="true"></span>`)).join("");
    const borrara = datos.foto && e.modo !== "foto" ? `<p class="imagen-aviso" role="note">${escapar(t("aviso_borrar_foto"))}</p>` : "";
    return `<form class="imagen-formulario" data-imagen-form novalidate><fieldset class="imagen-grupo imagen-grupo--modo"><legend>${escapar(t("modo_titulo"))}</legend><div class="imagen-modos">${modos}</div></fieldset>${detalle}<fieldset class="imagen-grupo"><legend>${escapar(t("paleta_titulo"))}</legend><div class="imagen-rejilla">${paletas}</div></fieldset>${borrara}<div class="imagen-acciones"><button type="submit" class="boton-primario"${ocupado ? " disabled" : ""}>${escapar(t(ocupado ? "guardando" : "guardar"))}</button></div></form>`;
  }

  function renderizarMensaje() {
    if (!mensaje) return `<p class="imagen-mensaje" data-imagen-mensaje tabindex="-1" role="status" hidden></p>`;
    const rol = mensaje.tipo === "error" ? "alert" : "status";
    const reintentar = pendiente ? ` <button type="button" class="boton-secundario" data-imagen-accion="reintentar"${ocupado ? " disabled" : ""}>${escapar(t("reintentar"))}</button>` : "";
    return `<p class="imagen-mensaje imagen-mensaje--${mensaje.tipo}" data-imagen-mensaje tabindex="-1" role="${rol}">${escapar(mensaje.texto)}${reintentar}</p>`;
  }

  function interior() {
    const Cabecera = marco.cabecera === "header" ? "header" : "div";
    const ayuda = `<button type="button" class="imagen-ayuda-boton" data-imagen-accion="ayuda" aria-expanded="${ayudaVisible}" aria-controls="imagen-ayuda" aria-label="${escapar(t("ayuda_abrir"))}">?</button>`;
    const cabecera = `<${Cabecera} class="${escapar(marco.claseCabecera)} imagen-cabecera"><h2 id="imagen-titulo">${escapar(t("titulo"))}</h2>${ayuda}</${Cabecera}>`;
    const textoAyuda = `<div id="imagen-ayuda" class="imagen-ayuda"${ayudaVisible ? "" : " hidden"}><p>${escapar(t("ayuda"))}</p></div>`;
    let contenido;
    if (carga === "sin_cargar" || (carga === "cargando" && !datos)) {
      contenido = `<p class="imagen-nota" role="status" aria-busy="true">${escapar(t("cargando"))}</p>`;
    } else if (!datos) {
      const texto = errorCarga?.codigo === "no_autenticado" ? t("error_sesion") : errorCarga?.codigo === "prohibido" ? t("error_denegado") : t("error_carga");
      contenido = `<p class="imagen-mensaje imagen-mensaje--error" role="alert">${escapar(texto)}</p><button type="button" class="boton-secundario" data-imagen-accion="recargar">${escapar(t("reintentar"))}</button>`;
    } else {
      contenido = `${renderizarMensaje()}${vistaPrevia(eleccionActual())}${formulario()}`;
    }
    return `${cabecera}<div class="${escapar(marco.cuerpo)} imagen-cuerpo">${textoAyuda}${contenido}</div>`;
  }

  function renderizar() {
    if (cargaAlMostrar && carga === "sin_cargar" && contenedor) queueMicrotask(() => { if (carga === "sin_cargar") void cargar(); });
    return `<section class="${escapar(marco.panel)} imagen-panel" data-imagen-raiz aria-labelledby="imagen-titulo"${ocupado ? ' aria-busy="true"' : ""}>${interior()}</section>`;
  }

  function alHacerClic(evento) {
    const boton = evento.target?.closest?.("[data-imagen-accion]");
    if (!boton || !raiz()?.contains(boton)) return;
    switch (boton.dataset.imagenAccion) {
      case "ayuda": ayudaVisible = !ayudaVisible; enfocarTras = "[data-imagen-accion=ayuda]"; repintar(); break;
      case "recargar": void cargar(); break;
      case "reintentar": if (pendiente) void enviar(pendiente); break;
      default: break;
    }
  }

  function alCambiarCampo(evento) {
    const campo = evento.target;
    if (!raiz()?.contains(campo) || !datos) return;
    const e = { ...eleccionActual() };
    if (campo.name === "foto") { void elegirArchivo(campo.files?.[0]); return; }
    if (campo.name === "imagen-modo" && MODOS_IMAGEN.includes(campo.value)) {
      e.modo = campo.value;
      e.icono = e.modo === "icono" ? (datos.catalogo.iconos.includes(e.icono) ? e.icono : datos.catalogo.iconos[0]) : "";
      enfocarTras = `#imagen-modo-${campo.value}`;
    } else if (campo.name === "imagen-paleta" && datos.catalogo.paletas.includes(campo.value)) {
      e.paleta = campo.value;
      enfocarTras = `#imagen-paleta-${campo.value}`;
    } else if (campo.name === "imagen-icono" && datos.catalogo.iconos.includes(campo.value)) {
      e.icono = campo.value;
      enfocarTras = `#imagen-icono-${campo.value}`;
    } else return;
    borrador = validarEleccion(e);
    errorFoto = false;
    mensaje = null;
    repintar();
  }

  function alEnviar(evento) {
    if (!evento.target?.matches?.("[data-imagen-form]") || !raiz()?.contains(evento.target)) return;
    evento.preventDefault();
    guardar();
  }

  function instalar(nuevo) {
    contenedor = nuevo;
    nuevo.addEventListener("click", alHacerClic);
    nuevo.addEventListener("change", alCambiarCampo);
    nuevo.addEventListener("submit", alEnviar);
    return () => {
      nuevo.removeEventListener("click", alHacerClic);
      nuevo.removeEventListener("change", alCambiarCampo);
      nuevo.removeEventListener("submit", alEnviar);
      desmontarPeticion();
      contenedor = null;
    };
  }
  function desmontarPeticion() { controlador?.abort(); ++generacion; ocupado = false; }

  return Object.freeze({ cargar, renderizar, instalar, desmontarPeticion, leer: () => datos, leerCarga: () => carga });
}

/** Textos de «Mi imagen» en el idioma actual (con respaldo del idioma por defecto). */
export function cargarTextosImagen(opciones) {
  return cargarTextos("preferencias", opciones);
}
