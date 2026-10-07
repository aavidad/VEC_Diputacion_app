const objeto = v => v !== null && typeof v === 'object' && !Array.isArray(v);
const cadena = v => typeof v === 'string' && v.length > 0 && v.length <= 240;
const clave = v => cadena(v) && /^formacion\.[a-z0-9_.]+$/u.test(v);
const pendientes = v => Array.isArray(v) && v.length <= 100 && v.every(clave);
const ausente = v => v === undefined || v === null || v === '';
const opcional = validar => v => ausente(v) || validar(v);
const cantidad = v => v == null || Number.isSafeInteger(v) && v >= 0;
const fecha = v => ausente(v) || typeof v === 'string' && /^\d{4}-\d{2}-\d{2}$/u.test(v) && new Date(`${v}T12:00:00Z`).toISOString().slice(0, 10) === v;
const catalogo = v => v == null || Array.isArray(v) && v.length <= 16 && v.every(clave);

export function validarEscenario(dto) {
  if (!objeto(dto) || dto.alcance !== 'preparacion_sintetica' || dto.version !== 1
    || !objeto(dto.fuente) || dto.fuente.escenario !== 'sintetico' || !cadena(dto.fuente.referencia) || !cadena(dto.fuente.version)
    || !objeto(dto.plan) || !cadena(dto.plan.referencia) || !clave(dto.plan.titulo_clave)
    || !fecha(dto.plan.desde) || !fecha(dto.plan.hasta) || !cantidad(dto.plan.presupuesto_centimos) || !cantidad(dto.plan.plazas)
    || !objeto(dto.plan.configuracion) || !opcional(cadena)(dto.plan.configuracion.version)
    || !catalogo(dto.plan.configuracion.modalidades) || !catalogo(dto.plan.configuracion.prioridades)
    || !Array.isArray(dto.ediciones) || dto.ediciones.length > 64 || !pendientes(dto.pendientes)
    || !Array.isArray(dto.checklist) || dto.checklist.length > 1024) throw new TypeError('formacion.contrato');
  const refs = new Set();
  for (const e of dto.ediciones) {
    if (!objeto(e) || !cadena(e.referencia) || refs.has(e.referencia) || !opcional(cadena)(e.accion_referencia)
      || !clave(e.titulo_clave) || ![e.necesidad_clave, e.prioridad_clave, e.modalidad_clave].every(opcional(clave))
      || !fecha(e.desde) || !fecha(e.hasta) || ![e.plazas, e.horas, e.presupuesto_centimos].every(cantidad)
      || !pendientes(e.pendientes)) throw new TypeError('formacion.edicion');
    refs.add(e.referencia);
  }
  for (const item of dto.checklist) {
    if (!objeto(item) || !clave(item.clave) || item.estado !== 'pendiente' || !cadena(item.referencia)) throw new TypeError('formacion.checklist');
  }
  // La copia de vista completa únicamente ausencias; la descarga conserva los bytes originales.
  return {
    ...dto,
    plan: { ...dto.plan, desde: dto.plan.desde ?? '', hasta: dto.plan.hasta ?? '',
      plazas: dto.plan.plazas ?? null, presupuesto_centimos: dto.plan.presupuesto_centimos ?? null,
      configuracion: { ...dto.plan.configuracion, version: dto.plan.configuracion.version ?? '',
        modalidades: dto.plan.configuracion.modalidades ?? [], prioridades: dto.plan.configuracion.prioridades ?? [] } },
    ediciones: dto.ediciones.map(e => ({ ...e, accion_referencia: e.accion_referencia ?? '',
      necesidad_clave: e.necesidad_clave ?? '', prioridad_clave: e.prioridad_clave ?? '', modalidad_clave: e.modalidad_clave ?? '',
      desde: e.desde ?? '', hasta: e.hasta ?? '', plazas: e.plazas ?? null, horas: e.horas ?? null,
      presupuesto_centimos: e.presupuesto_centimos ?? null })),
  };
}

export function filtrarEdiciones(escenario, modalidad = '') {
  return escenario.ediciones.filter(e => modalidad === '' || e.modalidad_clave === modalidad);
}

export function enlaceOficial(valor, permitidos) {
  if (typeof valor !== 'string' || !Array.isArray(permitidos) || !permitidos.includes(valor)) return null;
  try { const url = new URL(valor); return url.protocol === 'https:' && !url.username && !url.password ? url.href : null; }
  catch { return null; }
}
