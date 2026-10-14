import { validarConsultaB2 } from "./contrato-incorporacion-personal-b2.js";

const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const HEX64 = /^[0-9a-f]{64}$/u;
const CAMPOS_METADATOS = Object.freeze(["esquema_modelo", "esquema_mapeo", "esquema_carga", "version_expediente", "expediente_ref", "incorporacion_ref", "procedencia_modelo_ref", "correlacion_ref", "idempotencia_ref", "huella_modelo_sha256", "mapeo_ref", "mapeo_version", "procedencia_mapeo_ref", "huella_mapeo_sha256", "huella_carga_sha256"]);
const CAMPOS_B2 = Object.freeze(["actuacion_ref", "expediente_ref", "recibo_ref", "relacion_ref", "seguimiento_ref", "solicitud_personal_ref"]);
function fallo() { throw new TypeError("ficha GINPIX no válida"); }
function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor) || Object.getPrototypeOf(valor) !== Object.prototype || Object.getOwnPropertySymbols(valor).length || Object.keys(valor).length !== campos.length || !campos.every((campo) => Object.hasOwn(valor, campo)) || !Object.values(Object.getOwnPropertyDescriptors(valor)).every((d) => Object.hasOwn(d, "value") && d.enumerable)) fallo();
  return Object.fromEntries(campos.map((campo) => [campo, valor[campo]]));
}
function referencia(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }
function version(valor) { return Number.isSafeInteger(valor) && valor >= 1; }

export function validarConsultaB2ParaFichaGINPIX(consulta) {
  try {
    const c = validarConsultaB2(consulta, consulta?.expediente_ref);
    if (c.estado !== "incorporacion_confirmada" || c.recibo === null || c.plan === null) fallo();
    return c;
  } catch { fallo(); }
}

export function validarFichaGINPIX(contenido, consulta) {
  const c = validarConsultaB2ParaFichaGINPIX(consulta);
  const ficha = registro(contenido, ["esquema", "version", "metadatos", "campos"]);
  const m = registro(ficha.metadatos, CAMPOS_METADATOS);
  if (ficha.esquema !== "vec.dipgra.contratacion-temporal.ginpix.fichero.v1" || ficha.version !== 1
    || m.esquema_modelo !== "vec.dipgra.contratacion-temporal.ginpix.modelo.v1"
    || m.esquema_mapeo !== "vec.dipgra.contratacion-temporal.ginpix.mapeo.v1"
    || m.esquema_carga !== "vec.dipgra.contratacion-temporal.ginpix.carga.v1"
    || !version(m.version_expediente) || m.version_expediente !== c.version_expediente_actual
    || m.expediente_ref !== c.expediente_ref || m.incorporacion_ref !== c.recibo.recibo_ref
    || m.procedencia_modelo_ref !== c.recibo.recibo_ref
    || !["correlacion_ref", "idempotencia_ref"].every((k) => referencia(m[k]))
    || m.mapeo_ref !== "mapeo:ct:ginpix:ficha-personal-b2" || m.procedencia_mapeo_ref !== "configuracion:ct:ginpix:ficha-trazabilidad-personal-b2:v1"
    || !version(m.mapeo_version)
    || !["huella_modelo_sha256", "huella_mapeo_sha256", "huella_carga_sha256"].every((k) => typeof m[k] === "string" && HEX64.test(m[k]))
    || !Array.isArray(ficha.campos) || ficha.campos.length !== CAMPOS_B2.length) fallo();
  const campos = ficha.campos.map((campo, i) => {
    const dato = registro(campo, ["clave", "estado", "valor"]);
    if (dato.clave !== CAMPOS_B2[i] || dato.estado !== "valor" || !referencia(dato.valor)) fallo();
    return Object.freeze(dato);
  });
  const porClave = Object.fromEntries(campos.map((dato) => [dato.clave, dato.valor]));
  if (porClave.actuacion_ref !== c.recibo.recibo_ref || porClave.expediente_ref !== c.expediente_ref
    || porClave.relacion_ref !== c.recibo.relacion_ref || porClave.solicitud_personal_ref !== m.correlacion_ref) fallo();
  return Object.freeze({ esquema: ficha.esquema, version: ficha.version, metadatos: Object.freeze(m), campos: Object.freeze(campos) });
}
