import { validarPreparacionIncorporacionEjercicio } from "./contrato-incorporacion-ejercicio.js";

const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const HEX64 = /^[0-9a-f]{64}$/u;
const CAMPOS_METADATOS = Object.freeze(["esquema_modelo", "esquema_mapeo", "esquema_carga", "version_expediente", "expediente_ref", "incorporacion_ref", "procedencia_modelo_ref", "correlacion_ref", "idempotencia_ref", "huella_modelo_sha256", "mapeo_ref", "mapeo_version", "procedencia_mapeo_ref", "huella_mapeo_sha256", "huella_carga_sha256"]);
function fallo() { throw new TypeError("ficha GINPIX no válida"); }
function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor) || Object.getPrototypeOf(valor) !== Object.prototype || Object.getOwnPropertySymbols(valor).length || Object.keys(valor).length !== campos.length || !campos.every((campo) => Object.hasOwn(valor, campo)) || !Object.values(Object.getOwnPropertyDescriptors(valor)).every((d) => Object.hasOwn(d, "value") && d.enumerable)) fallo();
  return Object.fromEntries(campos.map((campo) => [campo, valor[campo]]));
}
function referencia(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }
function version(valor) { return Number.isSafeInteger(valor) && valor >= 1; }
export function validarReciboV2ParaFichaGINPIX(recibo) {
  if (!recibo || typeof recibo !== "object" || Array.isArray(recibo) || !referencia(recibo.expediente_ref)) fallo();
  try {
    return validarPreparacionIncorporacionEjercicio({ esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", expediente_ref: recibo.expediente_ref, version_actual_expediente: recibo.version_actual_expediente, preparacion: null, recibo }, recibo.expediente_ref).recibo;
  } catch { fallo(); }
}
export function validarFichaGINPIX(contenido, recibo) {
  const r = validarReciboV2ParaFichaGINPIX(recibo); const ficha = registro(contenido, ["esquema", "version", "metadatos", "campos"]); const m = registro(ficha.metadatos, CAMPOS_METADATOS);
  if (ficha.esquema !== "vec.dipgra.contratacion-temporal.ginpix.fichero.v1" || ficha.version !== 1 || m.esquema_modelo !== "vec.dipgra.contratacion-temporal.ginpix.modelo.v1" || m.esquema_mapeo !== "vec.dipgra.contratacion-temporal.ginpix.mapeo.v1" || m.esquema_carga !== "vec.dipgra.contratacion-temporal.ginpix.carga.v1" || !version(m.version_expediente) || m.version_expediente !== r.version_actual_expediente || m.expediente_ref !== r.expediente_ref || m.incorporacion_ref !== r.actuacion_ref || m.procedencia_modelo_ref !== r.recibo_ref || !["correlacion_ref", "idempotencia_ref", "mapeo_ref", "procedencia_mapeo_ref"].every((k) => referencia(m[k])) || !version(m.mapeo_version) || !["huella_modelo_sha256", "huella_mapeo_sha256", "huella_carga_sha256"].every((k) => typeof m[k] === "string" && HEX64.test(m[k])) || !Array.isArray(ficha.campos) || ficha.campos.length < 1 || ficha.campos.length > 128) fallo();
  let anterior = ""; const campos = ficha.campos.map((campo) => { const c = registro(campo, ["clave", "estado", "valor"]); if (typeof c.clave !== "string" || !/^[a-z][a-z0-9._-]{1,79}$/u.test(c.clave) || c.clave <= anterior || !["ausente", "nulo", "valor"].includes(c.estado) || typeof c.valor !== "string") fallo(); anterior = c.clave; return Object.freeze(c); });
  return Object.freeze({ esquema: ficha.esquema, version: ficha.version, metadatos: Object.freeze(m), campos: Object.freeze(campos) });
}
