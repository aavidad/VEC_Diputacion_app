import { fechaHistoriaServiciosValida, validarRespuestaHistoriaServicios } from "./cliente-http-historia-servicios-propia.js?v=20261004-personal-historia-v1";

export const CAMPOS_PREPARACION_RECTIFICACION = Object.freeze(["periodo_desde", "periodo_hasta", "dias_reconocidos", "estado", "clase"]);
const fallo = (codigo, errores = {}) => Object.assign(new Error(codigo), { codigo, errores });
const texto = (valor) => typeof valor === "string" && valor.trim().length > 0;
const firma = (historia, servicio) => JSON.stringify(historia.revisiones.filter((r) => r.servicio_ref === servicio));
const diasValidos = (valor) => typeof valor === "string" && /^(0|[1-9][0-9]*)$/u.test(valor) && Number.isSafeInteger(Number(valor));

// Estado y clase usan el validador del DTO recibido: sus códigos y el límite
// técnico de texto siguen teniendo una sola fuente en el cliente de historia.
function valorSegunDTO(validada, original, campo, propuesta, filtros) {
  const revisiones = validada.historia.revisiones.map((revision) => revision === original ? { ...revision, [campo]: propuesta } : revision);
  try {
    validarRespuestaHistoriaServicios({ data: { ...validada, historia: { ...validada.historia, revisiones } } }, filtros);
    return true;
  } catch { return false; }
}

function errorPropuesta(validada, original, campo, propuesta, filtros) {
  if (!texto(propuesta)) return "obligatorio";
  switch (campo) {
    case "periodo_desde":
    case "periodo_hasta": return fechaHistoriaServiciosValida(propuesta) ? "" : "fecha";
    case "dias_reconocidos": return diasValidos(propuesta) ? "" : "dias";
    case "estado": return valorSegunDTO(validada, original, campo, propuesta, filtros) ? "" : "estado";
    case "clase": return valorSegunDTO(validada, original, campo, propuesta, filtros) ? "" : "clase";
    default: return "obligatorio";
  }
}

/** Preparación en memoria; discutir un dato no declara que sea rectificable.
 * La selección tiene que ser una fila del resultado validado que se muestra.
 * Esta pieza no concede permisos, presenta solicitudes ni modifica servicios. */
export function crearPreparacionRectificacionPropia(datos, seleccion) {
  const filtros = { efectosDesde: datos?.historia?.corte?.efectos_desde, efectosHasta: datos?.historia?.corte?.efectos_hasta };
  const validada = validarRespuestaHistoriaServicios({ data: datos }, filtros);
  if (!datos.historia.revisiones.includes(seleccion)) throw fallo("seleccion_no_valida");
  let original = validada.historia.revisiones.find((r) => r.servicio_ref === seleccion.servicio_ref && r.traza.version === seleccion.traza.version);
  let revisiones = firma(validada.historia, original.servicio_ref), borrador;
  let conocidoEn = validada.historia.corte.conocido_en, consultadaEn = validada.consultada_en;
  const campos = Object.freeze(CAMPOS_PREPARACION_RECTIFICACION.filter((campo) => Object.hasOwn(seleccion, campo)));
  return Object.freeze({
    campos,
    valor(campo) { if (!original || !campos.includes(campo)) throw fallo("seleccion_no_valida"); return original[campo]; },
    preparar({ campo, propuesta, motivo, evidencia }) {
      if (!original) throw fallo("seleccion_no_valida");
      borrador = undefined;
      const errores = {};
      if (!campos.includes(campo)) errores.campo = "obligatorio";
      else {
        const codigo = errorPropuesta(validada, original, campo, propuesta, filtros);
        if (codigo) errores.propuesta = codigo;
      }
      if (!texto(motivo)) errores.motivo = "obligatorio";
      if (!texto(evidencia)) errores.evidencia = "obligatorio";
      if (Object.keys(errores).length) throw fallo("borrador_invalido", errores);
      borrador = Object.freeze({ campo, propuesta, motivo, evidencia });
    },
    revisar(respuesta) {
      if (!original || !borrador) throw fallo("borrador_invalido");
      const nueva = validarRespuestaHistoriaServicios({ data: respuesta }, filtros);
      if (nueva.historia.corte.conocido_en < conocidoEn || nueva.consultada_en < consultadaEn || firma(nueva.historia, original.servicio_ref) !== revisiones) throw fallo("revision_sustituida");
      return Object.freeze({ ...borrador, valor_actual: original[borrador.campo], corte: nueva.historia.corte, recibo_ref: nueva.recibo_ref, servicio_ref: original.servicio_ref, traza: original.traza });
    },
    limpiar() { original = undefined; revisiones = undefined; borrador = undefined; conocidoEn = undefined; consultadaEn = undefined; },
  });
}
