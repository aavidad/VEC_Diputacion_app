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
