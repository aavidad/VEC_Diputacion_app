/** Catálogo visual incluido en esta versión del cliente. No concede permisos. */
export const TEMAS_VEC = Object.freeze({
  institucional: Object.freeze({ tema_id: "institucional", revision: 1 }),
  granate: Object.freeze({ tema_id: "granate", revision: 1 }),
});

export class ErrorTemaVec extends Error {
  constructor(codigo, mensaje) {
    super(mensaje);
    this.name = "ErrorTemaVec";
    this.codigo = codigo;
  }
}

/** Valida la selección recibida; nunca interpreta CSS, URLs ni identificadores libres. */
export function validarEstadoTema(estado) {
  if (!estado || typeof estado !== "object" || Array.isArray(estado)) {
    throw new ErrorTemaVec("estado_invalido", "El estado del tema debe ser un objeto.");
  }
  const { tema_id: temaId, revision } = estado;
  if (typeof temaId !== "string" || !Object.hasOwn(TEMAS_VEC, temaId)) {
    throw new ErrorTemaVec("tema_desconocido", "El tema solicitado no pertenece al catálogo aprobado.");
  }
  if (!Number.isInteger(revision) || revision !== TEMAS_VEC[temaId].revision) {
    throw new ErrorTemaVec("revision_no_soportada", "La revisión del tema no está incluida en este cliente.");
  }
  return TEMAS_VEC[temaId];
}

/**
 * Aplica un estado versionado entregado por un consumidor autorizado y permite
 * previsualizarlo sin guardar nada. Este controlador no consulta ni acredita al servidor.
 */
export function crearControladorTema({ documento = globalThis.document } = {}) {
  const raiz = documento?.documentElement;
  const cuerpo = documento?.body;
  if (!raiz?.dataset || !cuerpo?.dataset || typeof raiz.getAttribute !== "function" || typeof raiz.removeAttribute !== "function") {
    throw new ErrorTemaVec("documento_no_disponible", "El documento no permite aplicar el tema.");
  }
  const inicial = raiz.getAttribute("data-tema");
  if (inicial !== null && !Object.hasOwn(TEMAS_VEC, inicial)) {
    throw new ErrorTemaVec("tema_desconocido", "El documento contiene un tema ajeno al catálogo aprobado.");
  }

  let estadoServidor = null;
  let vistaPrevia = null;
  let temaAnterior = null;
  let conflictoExterno = false;

  // Una autoridad externa puede actualizar el atributo mientras hay una vista
  // previa. No restauramos una preimagen que ya no nos pertenece.
  const sincronizarAtributo = () => {
    const actual = raiz.getAttribute("data-tema");
    if (vistaPrevia !== null && actual !== vistaPrevia.tema_id) {
      vistaPrevia = null;
      temaAnterior = null;
      estadoServidor = null;
      conflictoExterno = true;
      return true;
    }
    if (vistaPrevia === null && estadoServidor !== null && actual !== estadoServidor.tema_id) {
      estadoServidor = null;
      conflictoExterno = true;
    }
    return false;
  };

  const leerEstado = () => {
    sincronizarAtributo();
    const visible = vistaPrevia ?? estadoServidor;
    const temaVisible = visible?.tema_id ?? raiz.getAttribute("data-tema") ?? "institucional";
    return Object.freeze({
      tema_id: temaVisible,
      revision: visible?.revision ?? TEMAS_VEC[temaVisible]?.revision ?? null,
      previsualizacion: vistaPrevia !== null,
      estado_servidor: estadoServidor,
      alto_contraste: cuerpo.dataset.contraste === "true",
    });
  };

  return Object.freeze({
    leerEstado,
    aplicarEstadoServidor(estado) {
      const valido = validarEstadoTema(estado);
      estadoServidor = valido;
      vistaPrevia = null;
      temaAnterior = null;
      conflictoExterno = false;
      raiz.dataset.tema = valido.tema_id;
      return leerEstado();
    },
    previsualizar(estado) {
      const valido = validarEstadoTema(estado);
      sincronizarAtributo();
      if (conflictoExterno) {
        throw new ErrorTemaVec("tema_modificado", "El tema visible cambió fuera de este controlador.");
      }
      if (vistaPrevia === null) temaAnterior = raiz.getAttribute("data-tema");
      vistaPrevia = valido;
      raiz.dataset.tema = valido.tema_id;
      return leerEstado();
    },
    cancelarPrevisualizacion() {
      if (sincronizarAtributo()) return leerEstado();
      if (vistaPrevia === null) return leerEstado();
      if (temaAnterior === null) raiz.removeAttribute("data-tema");
      else raiz.dataset.tema = temaAnterior;
      vistaPrevia = null;
      temaAnterior = null;
      return leerEstado();
    },
    establecerAltoContraste(activo) {
      if (typeof activo !== "boolean") {
        throw new ErrorTemaVec("contraste_invalido", "El alto contraste debe indicarse con un valor booleano.");
      }
      cuerpo.dataset.contraste = String(activo);
      return leerEstado();
    },
  });
}

const MODOS_COLOR = new Set([
  "sistema", "claro", "oscuro", "diputacion_granada", "arena", "salvia",
  "lavanda", "azul_sereno", "noche_suave",
]);
const TAMANOS_TEXTO = new Set(["normal", "grande", "muy_grande"]);

/** Acepta exclusivamente las opciones visuales del catálogo de Usuarios. */
export function validarPreferenciasVisuales(preferencias) {
  if (!preferencias || typeof preferencias !== "object" || Array.isArray(preferencias)) {
    throw new ErrorTemaVec("preferencias_invalidas", "Las preferencias visuales deben ser un objeto.");
  }
  const { tema, alto_contraste: altoContraste, tamano_texto: tamanoTexto } = preferencias;
  if (!MODOS_COLOR.has(tema)) {
    throw new ErrorTemaVec("modo_color_invalido", "El modo de color no pertenece al catálogo.");
  }
  if (typeof altoContraste !== "boolean") {
    throw new ErrorTemaVec("contraste_invalido", "El alto contraste debe ser booleano.");
  }
  if (!TAMANOS_TEXTO.has(tamanoTexto)) {
    throw new ErrorTemaVec("tamano_texto_invalido", "El tamaño de texto no pertenece al catálogo.");
  }
  return Object.freeze({ tema, alto_contraste: altoContraste, tamano_texto: tamanoTexto });
}

/** Aplica respuestas del servidor sin HTTP ni persistencia local. */
export function crearControladorPreferenciasVisuales({ documento = globalThis.document, ventana = globalThis.window } = {}) {
  const raiz = documento?.documentElement;
  const cuerpo = documento?.body;
  if (!raiz?.dataset || !cuerpo?.dataset) {
    throw new ErrorTemaVec("documento_no_disponible", "El documento no permite aplicar preferencias.");
  }

  let preferenciasServidor = null;
  let consultaSistema = null;
  const modoSistema = () => consultaSistema?.matches ? "oscuro" : "claro";
  const actualizarSistema = () => {
    if (preferenciasServidor?.tema === "sistema") cuerpo.dataset.modoColor = modoSistema();
  };
  const desconectarSistema = () => {
    consultaSistema?.removeEventListener?.("change", actualizarSistema);
    consultaSistema?.removeListener?.(actualizarSistema);
    consultaSistema = null;
  };
  const leerEstado = () => Object.freeze({
    preferencias_servidor: preferenciasServidor,
    modo_color: cuerpo.dataset.modoColor ?? null,
    alto_contraste: cuerpo.dataset.contraste === "true",
    tamano_texto: raiz.dataset.tamanoTexto ?? null,
  });

  return Object.freeze({
    leerEstado,
    aplicarPreferenciasServidor(preferencias) {
      const validas = validarPreferenciasVisuales(preferencias);
      desconectarSistema();
      preferenciasServidor = validas;
      if (validas.tema === "sistema") {
        consultaSistema = ventana?.matchMedia?.("(prefers-color-scheme: dark)") ?? null;
        consultaSistema?.addEventListener?.("change", actualizarSistema);
        if (!consultaSistema?.addEventListener) consultaSistema?.addListener?.(actualizarSistema);
      }
      cuerpo.dataset.modoColor = validas.tema === "sistema" ? modoSistema() : validas.tema;
      cuerpo.dataset.contraste = String(validas.alto_contraste);
      raiz.dataset.tamanoTexto = validas.tamano_texto;
      return leerEstado();
    },
    desmontar() { desconectarSistema(); },
  });
}

/** Atajo para una respuesta del servidor; devuelve controlador para desmontar al salir. */
export function aplicarPreferenciasVisuales(preferencias, opciones = {}) {
  const controlador = crearControladorPreferenciasVisuales(opciones);
  controlador.aplicarPreferenciasServidor(preferencias);
  return controlador;
}
