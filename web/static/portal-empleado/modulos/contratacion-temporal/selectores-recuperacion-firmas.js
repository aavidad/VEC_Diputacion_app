import { validarSolicitudRecuperacionFirmasV2 } from "./contrato-recuperacion-firmas-v2.js?v=VEC_WEB_VERSION";

const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const SHA256 = /^[0-9a-f]{64}$/u;

export function validarMetadatosRecuperacionFirmas(entrada, expedienteRef, version) {
  const tieneSelectores = Object.hasOwn(entrada, "selectores_recuperacion_firmas");
  const tieneRecibo = Object.hasOwn(entrada, "recibo_selectores_recuperacion_firmas");
  if (!tieneSelectores && !tieneRecibo) return Object.freeze({});
  if (tieneSelectores !== tieneRecibo) throw new TypeError("prueba_selectores_recuperacion_incompleta");
  const recibo = entrada.recibo_selectores_recuperacion_firmas;
  const campos = ["acceso_ref", "sello_detalle_sha256", "sello_selectores_sha256"];
  if (!recibo || Object.getPrototypeOf(recibo) !== Object.prototype
    || Object.keys(recibo).length !== campos.length || campos.some((campo) => !Object.hasOwn(recibo, campo))
    || typeof recibo.acceso_ref !== "string" || !REFERENCIA.test(recibo.acceso_ref)
    || typeof recibo.sello_detalle_sha256 !== "string" || !SHA256.test(recibo.sello_detalle_sha256)
    || typeof recibo.sello_selectores_sha256 !== "string" || !SHA256.test(recibo.sello_selectores_sha256)) {
    throw new TypeError("prueba_selectores_recuperacion_invalida");
  }
  return Object.freeze({
    selectores_recuperacion_firmas: validarSelectoresRecuperacionFirmas(
      entrada.selectores_recuperacion_firmas, expedienteRef, version,
    ),
    recibo_selectores_recuperacion_firmas: Object.freeze({ ...recibo }),
  });
}

// Los selectores son datos de la lectura autorizada. Cada recuperación vuelve
// a comprobar la autoridad vigente; este contrato no concede acceso.
export function validarSelectoresRecuperacionFirmas(entrada, expedienteRef, version) {
  if (typeof expedienteRef !== "string" || !REFERENCIA.test(expedienteRef)
    || !Number.isSafeInteger(version) || version < 1
    || !Array.isArray(entrada) || entrada.length > 128) {
    throw new TypeError("selectores_recuperacion_firmas_invalidos");
  }
  const vistas = new Set();
  return Object.freeze(entrada.map((fila) => {
    if (!fila || Object.getPrototypeOf(fila) !== Object.prototype
      || Object.keys(fila).length !== 2 || !Object.hasOwn(fila, "firma_ref")
      || !Object.hasOwn(fila, "selector") || typeof fila.firma_ref !== "string"
      || !REFERENCIA.test(fila.firma_ref) || vistas.has(fila.firma_ref)) {
      throw new TypeError("selector_recuperacion_firma_invalido");
    }
    const selector = validarSolicitudRecuperacionFirmasV2(fila.selector);
    if (selector.via !== "certificado_vec" || selector.expediente_ref !== expedienteRef
      || selector.version_expediente > version) {
      throw new TypeError("selector_recuperacion_firma_incoherente");
    }
    vistas.add(fila.firma_ref);
    return Object.freeze({ firma_ref: fila.firma_ref, selector });
  }));
}
