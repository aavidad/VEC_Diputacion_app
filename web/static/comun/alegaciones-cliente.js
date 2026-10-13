import { consultarJSON, ErrorConsultaJSON } from "./http.js?v=20261012-sonda-frontera-v1";

const PROPIAS = "/api/vec/bolsa/alegaciones";
const RRHH = "/api/vec/bolsa/rrhh/alegaciones";
const REF = /^[A-Za-z0-9][A-Za-z0-9:._-]{2,255}$/u;
const SOLICITUD = /^solicitud_inscripcion_[0-9a-f]{64}$/u;
const CLAVE = /^[A-Za-z0-9_-]{16,128}$/u;
const registro = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const referencia = (v) => typeof v === "string" && REF.test(v);
const instante = (v) => typeof v === "string" && v.length <= 40 && Number.isFinite(Date.parse(v));
const version = (v) => Number.isSafeInteger(v) && v > 0;
const fallo = () => { throw new ErrorConsultaJSON("respuesta_no_valida"); };

function objetoValido(o) {
  if (!registro(o) || !referencia(o.acto_ref) || !version(o.version_acto)) return false;
  if (o.tipo === "decision_inscripcion") return SOLICITUD.test(o.solicitud_ref ?? "") && !o.participacion_ref;
  if (o.tipo === "acto_participacion") return referencia(o.participacion_ref) && !o.solicitud_ref;
  return false;
}

// La selección contiene etiquetas de presentación. El acto recibe sólo los
// atributos cerrados que el servidor volverá a cotejar con su titular.
export function objetoAlegacion(o) {
  if (!objetoValido(o)) throw new TypeError("objeto_alegacion_invalido");
  return { tipo: o.tipo,
    ...(o.tipo === "decision_inscripcion" ? { solicitud_ref: o.solicitud_ref } : { participacion_ref: o.participacion_ref }),
    acto_ref: o.acto_ref, version_acto: o.version_acto };
}

function alegacionValida(a) {
  return registro(a) && referencia(a.alegacion_ref) && objetoValido(a.objeto)
    && typeof a.fundamento === "string" && a.fundamento.trim().length > 0
    && [...a.fundamento].length <= 4000 && !a.fundamento.includes("\0")
    && a.estado === "presentada" && instante(a.presentada_en) && referencia(a.recibo_ref)
    && (a.persona_resumen === undefined || typeof a.persona_resumen === "string" && a.persona_resumen.length <= 512);
}

function pagina(respuesta, esquema, campo, validar, limite) {
  const d = respuesta?.data;
  if (!registro(d) || d.esquema !== esquema || !Array.isArray(d[campo]) || d[campo].length > limite
    || !Number.isSafeInteger(d.total) || d.total < d[campo].length || !d[campo].every(validar)
    || (d.cursor_siguiente !== null && d.cursor_siguiente !== "" && !referencia(d.cursor_siguiente))) fallo();
  return d;
}

function parametros(limite, cursor) {
  if (!Number.isSafeInteger(limite) || limite < 1 || limite > 10 || (cursor && !referencia(cursor))) {
    throw new TypeError("pagina_alegaciones_invalida");
  }
  const q = new URLSearchParams({ limite: String(limite) });
  if (cursor) q.set("cursor", cursor);
  return q;
}

export function crearClienteAlegaciones({ fetchImpl = globalThis.fetch } = {}) {
  const leer = (ruta, signal) => consultarJSON(ruta, { fetchImpl, signal });
  return Object.freeze({
    async objetos({ limite = 10, cursor = "", signal } = {}) {
      const r = await leer(`${PROPIAS}/objetos?${parametros(limite, cursor)}`, signal);
      return pagina(r, "vec.bolsa.alegaciones.objetos.v1", "objetos", (o) => objetoValido(o)
        && typeof o.titulo === "string" && o.titulo.trim() !== "" && o.titulo.length <= 512
        && typeof o.estado === "string" && instante(o.ocurrido_en), limite);
    },
    async propias({ limite = 10, cursor = "", signal } = {}) {
      return pagina(await leer(`${PROPIAS}?${parametros(limite, cursor)}`, signal),
        "vec.bolsa.alegaciones.propias.v1", "alegaciones", alegacionValida, limite);
    },
    async ambitos({ limite = 10, cursor = "", signal } = {}) {
      return pagina(await leer(`${RRHH}/ambitos?${parametros(limite, cursor)}`, signal),
        "vec.bolsa.alegaciones.ambitos.v1", "ambitos", (a) => registro(a)
          && ["convocatoria", "bolsa"].includes(a.tipo) && referencia(a.referencia)
          && typeof a.titulo === "string" && a.titulo.trim() !== "" && a.titulo.length <= 512, limite);
    },
    async rrhh({ tipo, referencia: ref, limite = 10, cursor = "", signal } = {}) {
      if (!["convocatoria", "bolsa"].includes(tipo) || !referencia(ref)) throw new TypeError("ambito_alegaciones_invalido");
      const q = parametros(limite, cursor); q.set(`${tipo}_ref`, ref);
      return pagina(await leer(`${RRHH}?${q}`, signal), "vec.bolsa.alegaciones.rrhh.v1",
        "alegaciones", alegacionValida, limite);
    },
    async detalle({ alegacionRef, signal } = {}) {
      if (!referencia(alegacionRef)) throw new TypeError("alegacion_invalida");
      const d = (await leer(`${RRHH}/${encodeURIComponent(alegacionRef)}`, signal))?.data;
      if (!registro(d) || d.esquema !== "vec.bolsa.alegacion.detalle.v1" || !alegacionValida(d)
        || d.alegacion_ref !== alegacionRef) fallo();
      return d;
    },
    async presentar({ objeto, fundamento, clave_idempotencia, signal } = {}) {
      if (typeof fundamento !== "string" || fundamento.trim() === "" || [...fundamento].length > 4000
        || fundamento.includes("\0") || !CLAVE.test(clave_idempotencia ?? "")) throw new TypeError("alegacion_invalida");
      const cuerpo = { objeto: objetoAlegacion(objeto), fundamento, clave_idempotencia };
      const d = (await consultarJSON(PROPIAS, { metodo: "POST", cuerpo, signal, fetchImpl }))?.data;
      if (!registro(d) || d.esquema !== "vec.bolsa.alegacion.presentada.v1" || !alegacionValida(d)
        || typeof d.repetida !== "boolean" || d.fundamento !== fundamento
        || JSON.stringify(objetoAlegacion(d.objeto)) !== JSON.stringify(cuerpo.objeto)) fallo();
      return d;
    },
  });
}
