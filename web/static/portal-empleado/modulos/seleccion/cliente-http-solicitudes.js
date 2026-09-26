/**
 * Adaptador HTTP de la bandeja de solicitudes de Selección para RRHH.
 *
 * Mismo origen (certificado mTLS de RRHH), sin caché ni redirecciones. La
 * autorización y la auditoría del acceso a la ficha las aplica el servidor;
 * aquí solo se validan formas y límites. Los errores son códigos cerrados.
 */

const BASE = "/api/vec/seleccion/solicitudes";
export const RUTAS_SELECCION_RRHH = Object.freeze({
  convocatorias: `${BASE}/convocatorias`,
  consultas: `${BASE}/consultas`,
  detalle: `${BASE}/detalle/consultas`,
});

const MAXIMO_RESPUESTA = 1024 * 1024;
const LIMITE_MS = 15_000;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,159}$/u;
const PATRON_DECIMAL = /^-?\d{1,9}(?:\.\d{1,6})?$/u;
const CODIGOS = new Set(["autenticacion_requerida", "acceso_denegado", "recurso_no_encontrado", "datos_no_validos",
  "peticion_no_permitida", "metodo_no_permitido", "servicio_no_disponible", "convocatoria_no_disponible"]);
const POR_ESTADO = Object.freeze({ 400: "datos_no_validos", 401: "autenticacion_requerida", 403: "acceso_denegado",
  404: "no_disponible", 405: "no_disponible", 503: "servicio_no_disponible" });

export class ErrorSeleccionRRHH extends Error {
  constructor(codigo, estado = 0) {
    super(codigo);
    this.name = "ErrorSeleccionRRHH";
    this.codigo = codigo;
    this.estado = estado;
  }
}

const incompatible = () => new ErrorSeleccionRRHH("respuesta_incompatible");
const objeto = (valor) => { if (!valor || typeof valor !== "object" || Array.isArray(valor)) throw incompatible(); return valor; };
const cadena = (valor, maximo = 300) => typeof valor === "string" ? valor.trim().slice(0, maximo) : "";
const instante = (valor) => typeof valor === "string" && !Number.isNaN(Date.parse(valor)) ? valor : null;
const decimal = (valor) => typeof valor === "string" && PATRON_DECIMAL.test(valor) ? valor : null;
const lista = (valor, maximo) => { if (!Array.isArray(valor) || valor.length > maximo) throw incompatible(); return valor; };

function validarFila(item) {
  objeto(item);
  if (!PATRON_REFERENCIA.test(item.solicitud_ref || "")) throw incompatible();
  return Object.freeze({
    solicitud_ref: item.solicitud_ref, numero_justificante: cadena(item.numero_justificante, 40),
    nombre_visible: cadena(item.nombre_visible, 200), documento_parcial: cadena(item.documento_parcial, 20),
    estado: cadena(item.estado, 60), presentada_en: instante(item.presentada_en),
    puntuacion_autobaremo: decimal(item.puntuacion_autobaremo), turno: cadena(item.turno, 80),
  });
}

function validarConvocatoria(item) {
  objeto(item);
  if (!PATRON_REFERENCIA.test(item.convocatoria_ref || "") || !cadena(item.titulo)) throw incompatible();
  return Object.freeze({ convocatoria_ref: item.convocatoria_ref, titulo: cadena(item.titulo), abierta: item.abierta === true,
    abre_en: instante(item.abre_en), cierra_en: instante(item.cierra_en) });
}

function validarDetalle(dato, solicitudRef) {
  objeto(dato);
  if (dato.solicitud_ref !== solicitudRef) throw incompatible();
  const datos = dato.datos && typeof dato.datos === "object" && !Array.isArray(dato.datos) ? dato.datos : {};
  return Object.freeze({
    solicitud_ref: dato.solicitud_ref, convocatoria_titulo: cadena(dato.convocatoria_titulo),
    numero_justificante: cadena(dato.numero_justificante, 40), estado: cadena(dato.estado, 60),
    presentada_en: instante(dato.presentada_en), turno: cadena(dato.turno, 80),
    datos: Object.freeze(structuredClone(datos)),
    requisitos: Object.freeze(lista(dato.requisitos ?? [], 200).map((item) => Object.freeze({
      titulo: cadena(item?.titulo) || cadena(item?.clave), estado: cadena(item?.estado, 30),
      procedencia: cadena(item?.procedencia, 60), fecha_referencia: cadena(item?.fecha_referencia, 10),
    }))),
    meritos: Object.freeze(lista(dato.meritos ?? [], 400).map((item) => Object.freeze({
      titulo: cadena(item?.titulo), descripcion: cadena(item?.descripcion), cantidad: decimal(item?.cantidad), unidad: cadena(item?.unidad, 60),
      puntos: decimal(item?.puntos),
    }))),
    puntuacion_autobaremo: decimal(dato.puntuacion_autobaremo),
    historia: Object.freeze(lista(dato.historia ?? [], 400).map((item) => Object.freeze({
      tipo: cadena(item?.tipo, 60), version: Number.isSafeInteger(item?.version) ? item.version : null, en: instante(item?.en),
    }))),
  });
}

export function crearClienteSolicitudesSeleccion({ fetchImpl = globalThis.fetch } = {}) {
  async function pedir(ruta, { cuerpo, signal } = {}) {
    if (typeof fetchImpl !== "function") throw new ErrorSeleccionRRHH("servicio_no_disponible");
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    signal?.addEventListener?.("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, LIMITE_MS);
    let respuesta;
    try {
      respuesta = await fetchImpl(ruta, {
        method: cuerpo === undefined ? "GET" : "POST", credentials: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
        headers: cuerpo === undefined ? { Accept: "application/json" } : { Accept: "application/json", "Content-Type": "application/json" },
        ...(cuerpo === undefined ? {} : { body: JSON.stringify(cuerpo) }),
      });
    } catch {
      throw new ErrorSeleccionRRHH(signal?.aborted ? "cancelada" : "servicio_no_disponible");
    } finally {
      clearTimeout(temporizador);
      signal?.removeEventListener?.("abort", abortar);
    }
    let datos = null;
    if (/^application\/json(?:\s*;\s*charset=utf-8)?$/iu.test(respuesta?.headers?.get?.("Content-Type") || "")) {
      const texto = await respuesta.text();
      if (new TextEncoder().encode(texto).byteLength > MAXIMO_RESPUESTA) throw incompatible();
      try { datos = JSON.parse(texto); } catch { datos = null; }
    }
    if (respuesta?.status !== 200) {
      const codigo = datos?.error?.codigo;
      throw new ErrorSeleccionRRHH(typeof codigo === "string" && CODIGOS.has(codigo) ? codigo : POR_ESTADO[respuesta?.status] || "respuesta_incompatible", respuesta?.status);
    }
    if (datos === null) throw incompatible();
    return objeto(objeto(datos).data);
  }

  return Object.freeze({
    async convocatorias({ signal } = {}) {
      return Object.freeze(lista((await pedir(RUTAS_SELECCION_RRHH.convocatorias, { signal })).convocatorias, 500).map(validarConvocatoria));
    },
    async consultar({ convocatoriaRef, cursor = "", limite = 25 }, { signal } = {}) {
      if (!PATRON_REFERENCIA.test(convocatoriaRef || "")) throw new ErrorSeleccionRRHH("datos_no_validos");
      const tope = Math.min(100, Math.max(1, Number.isSafeInteger(limite) ? limite : 25));
      const dato = await pedir(RUTAS_SELECCION_RRHH.consultas, { cuerpo: { convocatoria_ref: convocatoriaRef, cursor, limite: tope }, signal });
      const cursorSiguiente = typeof dato.cursor_siguiente === "string" ? dato.cursor_siguiente.slice(0, 512) : "";
      return Object.freeze({ solicitudes: Object.freeze(lista(dato.solicitudes, tope).map(validarFila)), cursorSiguiente });
    },
    async detalle(solicitudRef, { signal } = {}) {
      if (!PATRON_REFERENCIA.test(solicitudRef || "")) throw new ErrorSeleccionRRHH("datos_no_validos");
      return validarDetalle(await pedir(RUTAS_SELECCION_RRHH.detalle, { cuerpo: { solicitud_ref: solicitudRef }, signal }), solicitudRef);
    },
  });
}
