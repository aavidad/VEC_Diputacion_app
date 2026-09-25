/**
 * Cliente del protocolo AutoFirma (afirma://) para firmar un PDF en el equipo
 * de la persona. Lanza la aplicación con «afirma://websocket», se conecta por
 * WSS a 127.0.0.1 y pide una firma PAdES del PDF. Vale para AutoFirma oficial
 * y para AutofirmaV2. No verifica nada: la firma la verifica el servidor, y
 * una firma hecha así no tiene eficacia administrativa hasta el portafirmas
 * corporativo.
 */

export const PUERTO_AUTOFIRMA = 63117;
const VERSION_PROTOCOLO = 4;
const ALFABETO = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
const MAXIMO_PDF = 1024 * 1024;
const MAXIMO_RESPUESTA = 8 * 1024 * 1024;

/** Error con un código cerrado; los textos visibles los pone la vista. */
export class ErrorAutoFirma extends Error {
  constructor(codigo) {
    super(codigo);
    this.name = "ErrorAutoFirma";
    this.codigo = codigo;
  }
}

function base64URL(bytes) {
  let binario = "";
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binario += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  }
  return btoa(binario).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/u, "");
}

function desdeBase64(texto) {
  const limpio = texto.trim().replaceAll("-", "+").replaceAll("_", "/").replace(/=+$/u, "");
  if (!/^[A-Za-z0-9+/]+$/u.test(limpio) || limpio.length % 4 === 1) throw new ErrorAutoFirma("respuesta_no_valida");
  const binario = atob(limpio + "=".repeat((4 - (limpio.length % 4)) % 4));
  const bytes = new Uint8Array(binario.length);
  for (let i = 0; i < binario.length; i += 1) bytes[i] = binario.charCodeAt(i);
  return bytes;
}

function empiezaPorPDF(bytes) {
  return bytes.length > 5 && bytes[0] === 0x25 && bytes[1] === 0x50 && bytes[2] === 0x44 && bytes[3] === 0x46 && bytes[4] === 0x2d;
}

/** Lanza el protocolo con un enlace: no navega fuera del portal. */
export function crearLanzadorProtocolo(documento = globalThis.document) {
  return (url) => {
    if (!documento?.createElement || !documento.body) throw new ErrorAutoFirma("autofirma_no_disponible");
    const enlace = documento.createElement("a");
    enlace.href = url;
    enlace.hidden = true;
    enlace.rel = "noopener noreferrer";
    documento.body.append(enlace);
    try { enlace.click(); } finally { enlace.remove(); }
  };
}

function identificadorSesion(aleatorio) {
  const bytes = aleatorio(20);
  let id = "";
  for (const b of bytes) id += ALFABETO[b % ALFABETO.length];
  return id;
}

function esperar(ms, signal, temporizador) {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) { reject(new ErrorAutoFirma("operacion_abortada")); return; }
    const id = temporizador.setTimeout(() => { signal?.removeEventListener("abort", abortar); resolve(); }, ms);
    function abortar() { temporizador.clearTimeout(id); reject(new ErrorAutoFirma("operacion_abortada")); }
    signal?.addEventListener("abort", abortar, { once: true });
  });
}

/**
 * Canal de mensajes sobre un WebSocket abierto: cada recibir() espera el
 * siguiente mensaje de texto con un plazo, y se cancela con la señal.
 */
function abrirCanal(WebSocketImpl, url, signal, temporizador, plazoApertura) {
  return new Promise((resolve, reject) => {
    let socket;
    try { socket = new WebSocketImpl(url); } catch { reject(new ErrorAutoFirma("autofirma_no_disponible")); return; }
    const cola = [];
    const esperas = [];
    let cerrado = false;
    const fallarEsperas = (codigo) => { while (esperas.length) esperas.shift().reject(new ErrorAutoFirma(codigo)); };
    const plazo = temporizador.setTimeout(() => { try { socket.close(); } catch { /* sin efecto */ } reject(new ErrorAutoFirma("autofirma_no_disponible")); }, plazoApertura);
    socket.onmessage = (evento) => {
      const texto = typeof evento.data === "string" ? evento.data : null;
      if (texto === null || texto.length > MAXIMO_RESPUESTA) { fallarEsperas("respuesta_no_valida"); return; }
      if (esperas.length) esperas.shift().resolve(texto); else cola.push(texto);
    };
    socket.onclose = () => { cerrado = true; fallarEsperas("autofirma_no_disponible"); };
    socket.onerror = () => { temporizador.clearTimeout(plazo); reject(new ErrorAutoFirma("autofirma_no_disponible")); };
    socket.onopen = () => {
      temporizador.clearTimeout(plazo);
      resolve(Object.freeze({
        enviar(texto) {
          if (cerrado) throw new ErrorAutoFirma("autofirma_no_disponible");
          socket.send(texto);
        },
        recibir(ms) {
          if (cola.length) return Promise.resolve(cola.shift());
          if (cerrado) return Promise.reject(new ErrorAutoFirma("autofirma_no_disponible"));
          return new Promise((ok, ko) => {
            const entrada = { resolve: ok, reject: ko };
            const limite = temporizador.setTimeout(() => {
              const i = esperas.indexOf(entrada);
              if (i >= 0) esperas.splice(i, 1);
              ko(new ErrorAutoFirma("plazo_agotado"));
            }, ms);
            const abortar = () => { temporizador.clearTimeout(limite); ko(new ErrorAutoFirma("operacion_abortada")); };
            signal?.addEventListener("abort", abortar, { once: true });
            entrada.resolve = (v) => { temporizador.clearTimeout(limite); signal?.removeEventListener("abort", abortar); ok(v); };
            entrada.reject = (e) => { temporizador.clearTimeout(limite); signal?.removeEventListener("abort", abortar); ko(e); };
            esperas.push(entrada);
          });
        },
        cerrar() { try { socket.close(); } catch { /* sin efecto */ } },
      }));
    };
  });
}

/**
 * Cliente de firma. Todas las dependencias se pueden sustituir en pruebas;
 * en el navegador usa WebSocket, el reloj y el generador aleatorio del
 * propio navegador.
 */
export function crearClienteAutoFirma({
  WebSocketImpl = globalThis.WebSocket,
  lanzar = crearLanzadorProtocolo(),
  aleatorio = (n) => globalThis.crypto.getRandomValues(new Uint8Array(n)),
  temporizador = { setTimeout: globalThis.setTimeout.bind(globalThis), clearTimeout: globalThis.clearTimeout.bind(globalThis) },
  puerto = PUERTO_AUTOFIRMA,
  intentos = 40,
  esperaReintentoMs = 1000,
  plazoFirmaMs = 5 * 60 * 1000,
} = {}) {
  return Object.freeze({
    /** Devuelve el PDF firmado (PAdES) por AutoFirma. */
    async firmarPDF(pdf, { signal } = {}) {
      if (!(pdf instanceof Uint8Array) || !empiezaPorPDF(pdf) || pdf.length > MAXIMO_PDF) throw new ErrorAutoFirma("documento_no_valido");
      if (typeof WebSocketImpl !== "function" || typeof lanzar !== "function") throw new ErrorAutoFirma("autofirma_no_disponible");
      const id = identificadorSesion(aleatorio);
      lanzar(`afirma://websocket?v=${VERSION_PROTOCOLO}&idsession=${id}&ports=${puerto}`);
      let canal = null;
      try {
        for (let intento = 0; intento < intentos && !canal; intento += 1) {
          if (signal?.aborted) throw new ErrorAutoFirma("operacion_abortada");
          try {
            const candidato = await abrirCanal(WebSocketImpl, `wss://127.0.0.1:${puerto}`, signal, temporizador, esperaReintentoMs * 5);
            candidato.enviar(`echo=-idsession=${id}@EOF`);
            if ((await candidato.recibir(esperaReintentoMs * 5)).trim() === "OK") canal = candidato;
            else candidato.cerrar();
          } catch (error) {
            if (error?.codigo === "operacion_abortada") throw error;
            await esperar(esperaReintentoMs, signal, temporizador);
          }
        }
        if (!canal) throw new ErrorAutoFirma("autofirma_no_disponible");
        canal.enviar(`afirma://sign?op=sign&id=${id}&idsession=${id}&format=PAdES&algorithm=SHA256withRSA&dat=${base64URL(pdf)}`);
        const respuesta = (await canal.recibir(plazoFirmaMs)).trim();
        if (/^CANCEL/iu.test(respuesta)) throw new ErrorAutoFirma("firma_cancelada");
        if (/^(SAF_|ERR-)/iu.test(respuesta)) throw new ErrorAutoFirma("firma_fallida");
        const partes = respuesta.split("|");
        if (partes.length < 2 || partes.length > 3) throw new ErrorAutoFirma("respuesta_no_valida");
        const firmado = desdeBase64(partes[1]);
        if (!empiezaPorPDF(firmado) || firmado.length <= pdf.length) throw new ErrorAutoFirma("respuesta_no_valida");
        return firmado;
      } finally {
        canal?.cerrar();
      }
    },
  });
}

export const paraPruebas = Object.freeze({ base64URL, desdeBase64 });
