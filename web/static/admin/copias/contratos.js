import { ErrorCopias, objeto, entero, fecha, codigo, lista, referencia } from "./cliente-http.js?v=20261001-cs09-copias-v1";

const HUELLA = /^[a-f0-9]{64}$/u;
export function huella(valor) { if (typeof valor !== "string" || !HUELLA.test(valor)) throw new ErrorCopias(); return valor; }
function dia(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d\d-\d\d$/u.test(valor) || new Date(valor).toISOString().slice(0, 10) !== valor) throw new ErrorCopias();
  return valor;
}
function hora(valor) { if (typeof valor !== "string" || !/^(?:[01]\d|2[0-3]):[0-5]\d$/u.test(valor)) throw new ErrorCopias(); return valor; }
function positiva(valor) { const n = entero(valor); if (n === 0) throw new ErrorCopias(); return n; }
function bool(valor) { if (typeof valor !== "boolean") throw new ErrorCopias(); return valor; }
export function normalizarConfiguracion(valor) {
  const c = objeto(valor), p = objeto(c.politica), v = objeto(p.ventana), r = objeto(p.retencion);
  if (p.formato !== 1 || typeof p.zona_horaria !== "string" || p.zona_horaria.length > 80) throw new ErrorCopias();
  try { new Intl.DateTimeFormat(undefined, { timeZone: p.zona_horaria }); } catch { throw new ErrorCopias(); }
  const cada = positiva(p.cada_dias); if (cada > 366) throw new ErrorCopias();
  const dias = lista(p.dias_semana ?? []).map(v => { const n = entero(v); if (n > 6) throw new ErrorCopias(); return n; });
  if (new Set(dias).size !== dias.length || hora(v.inicio) >= hora(v.fin)) throw new ErrorCopias();
  return Object.freeze({ version: entero(c.version), politica: Object.freeze({ formato: 1,
    referencia: referencia(p.referencia), destino: referencia(p.destino), zona_horaria: p.zona_horaria,
    fecha_inicial: dia(p.fecha_inicial), cada_dias: cada, dias_semana: Object.freeze(dias),
    ventana: Object.freeze({ inicio: v.inicio, fin: v.fin }),
    retencion: Object.freeze({ conservar_minimo: positiva(r.conservar_minimo), edad_maxima_dias: positiva(r.edad_maxima_dias),
      protegidas: Object.freeze(lista(r.protegidas ?? []).map(referencia)), borrado_permitido: bool(r.borrado_permitido),
      doble_control: r.doble_control === undefined ? true : bool(r.doble_control) }) }) });
}
export function normalizarPropuesta(valor) {
  const p = objeto(valor);
  return Object.freeze({ propuesta_ref: referencia(p.propuesta_ref), conjunto_ref: referencia(p.conjunto_ref),
    conjunto_huella_sha256: huella(p.conjunto_huella_sha256), destino_ref: referencia(p.destino_ref), motivo_ref: referencia(p.motivo_ref), ventana_ref: referencia(p.ventana_ref),
    politica_ref: referencia(p.politica_ref), preimagen_sha256: huella(p.preimagen_sha256), politica_huella_sha256: huella(p.politica_huella_sha256),
    estado: codigo(p.estado), version: entero(p.version), huella_sha256: huella(p.huella_sha256),
    caduca_en: fecha(p.caduca_en), ventana_inicio: fecha(p.ventana_inicio), ventana_fin: fecha(p.ventana_fin),
    doble_control: bool(p.doble_control), copia_previa_requerida: bool(p.copia_previa_requerida),
    ...(p.perdida_desde ? { perdida_desde: fecha(p.perdida_desde) } : {}),
    ...(p.perdida_hasta ? { perdida_hasta: fecha(p.perdida_hasta) } : {}),
    ...(p.alcance_perdida_clave_i18n ? { alcance_perdida_clave_i18n: clave(p.alcance_perdida_clave_i18n) } : {}) });
}
function clave(valor) { if (typeof valor !== "string" || !/^[a-z][a-z0-9_.-]{0,179}$/u.test(valor)) throw new ErrorCopias(); return valor; }
export function normalizarOpciones(valor) {
  const o = objeto(valor);
  return Object.freeze(Object.fromEntries(["destinos", "motivos", "ventanas"].map(k => [k, Object.freeze(lista(o[k]).map(v => {
    const op = objeto(v); return Object.freeze({ ref: referencia(op.ref), clave_i18n: clave(op.clave_i18n) });
  }))])));
}
export function solicitudPropuesta(valor) {
  const p = objeto(valor);
  return { operacion_ref: referencia(p.operacion_ref), conjunto_ref: referencia(p.conjunto_ref), destino_ref: referencia(p.destino_ref),
    motivo_ref: referencia(p.motivo_ref), ventana_ref: referencia(p.ventana_ref), ventana_inicio: fecha(p.ventana_inicio), ventana_fin: fecha(p.ventana_fin), caduca_en: fecha(p.caduca_en) };
}
export function solicitudControl(valor) {
  const p = objeto(valor); return { operacion_ref: referencia(p.operacion_ref), destino_ref: referencia(p.destino_ref),
    propuesta_huella_sha256: huella(p.propuesta_huella_sha256), version_esperada: entero(p.version_esperada) };
}
