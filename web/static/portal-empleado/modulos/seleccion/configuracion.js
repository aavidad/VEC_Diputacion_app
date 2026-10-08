export const MODALIDADES = Object.freeze(['oposicion', 'concurso', 'concurso_oposicion']);
export const CAUSAS = Object.freeze(['requisito_no_cumplido', 'requisito_pendiente', 'minimo_pendiente',
  'nota_pendiente', 'nota_fuera_rango', 'minimo_no_superado', 'meritos_pendientes', 'precision_pendiente',
  'clasificacion_pendiente', 'desempate_pendiente']);
const entero = valor => Number.isSafeInteger(valor) && valor >= 0;
const cadena = valor => typeof valor === 'string' && valor.length > 0 && valor.length <= 256;

/** Conversión decimal exacta: nunca redondea una regla escrita por la persona. */
export function aMicropuntos(texto) {
  const valor = String(texto).trim();
  if (!/^\d+(?:[.,]\d{1,6})?$/u.test(valor)) return NaN;
  const [entera, decimal = ''] = valor.replace(',', '.').split('.');
  const puntos = Number(entera) * 1000000 + Number(decimal.padEnd(6, '0'));
  return entero(puntos) ? puntos : NaN;
}
export function decimalPuntos(micro) {
  return String(micro / 1000000);
}

/** Sólo notas de las solicitudes y pruebas que aporta el ejemplo del servidor. */
export function validarNotasEjemplo(ejemplo) {
  const notas = ejemplo.notas_prueba ?? [];
  if (!Array.isArray(notas) || notas.length > 128) throw new Error('seleccion.notas');
  const pares = new Set();
  for (const nota of notas) {
    const fase = ejemplo.configuracion.fases.find(f => f.referencia === nota?.fase_ref);
    const par = `${nota?.solicitud_ref}\0${nota?.fase_ref}`;
    if (!cadena(nota?.solicitud_ref) || !cadena(nota?.nombre) || !fase || fase.tipo !== 'prueba'
      || pares.has(par) || !Object.hasOwn(nota, 'puntos_micropuntos')
      || !(nota.puntos_micropuntos === null || (entero(nota.puntos_micropuntos)
        && nota.puntos_micropuntos <= fase.maximo_micropuntos))) throw new Error('seleccion.notas');
    pares.add(par);
  }
  return notas;
}

export function validarNotasPropuestas(ejemplo, configuracion, propuestas) {
  const notas = validarNotasEjemplo(ejemplo); const errores = []; const pares = new Set();
  for (const nota of propuestas) {
    const indice = notas.findIndex(n => n.solicitud_ref === nota.solicitud_ref && n.fase_ref === nota.fase_ref);
    const fase = configuracion.fases.find(f => f.referencia === nota.fase_ref);
    const par = `${nota.solicitud_ref}\0${nota.fase_ref}`;
    if (indice < 0 || !fase || fase.tipo !== 'prueba' || pares.has(par)
      || !(nota.puntos_micropuntos === null || (entero(nota.puntos_micropuntos)
        && nota.puntos_micropuntos <= fase.maximo_micropuntos))) {
      errores.push({ campo: `seleccion-nota-${Math.max(0, indice)}`, clave: 'validacion.nota' });
    }
    pares.add(par);
  }
  return errores;
}

export function validarConfiguracion(configuracion) {
  const c = configuracion; const errores = [];
  if (!MODALIDADES.includes(c?.modalidad)) errores.push({ campo: 'seleccion-ejemplo', clave: 'validacion.modalidad' });
  if (!entero(c?.plazas) || c.plazas < 1 || c.plazas > 1000) errores.push({ campo: 'seleccion-plazas', clave: 'validacion.plazas' });
  if (!Array.isArray(c?.fases) || !c.fases.length || c.fases.length > 16) return [...errores, { campo: 'seleccion-ejemplo', clave: 'validacion.fases' }];
  c.fases.forEach((fase, indice) => {
    if (fase.minimo_micropuntos !== null && (!entero(fase.minimo_micropuntos) || fase.minimo_micropuntos > fase.maximo_micropuntos)) {
      errores.push({ campo: `seleccion-minimo-${indice}`, clave: 'validacion.minimo' });
    }
    if (!entero(fase.peso) || fase.peso < 1 || fase.peso > 100) errores.push({ campo: `seleccion-peso-${indice}`, clave: 'validacion.peso' });
  });
  if (c.fases.reduce((total, fase) => total + fase.peso, 0) !== 100) errores.push({ campo: 'seleccion-peso-0', clave: 'validacion.suma_pesos' });
  if (!Array.isArray(c.desempates) || new Set(c.desempates).size !== c.desempates.length
    || c.desempates.some(ref => !c.fases.some(fase => fase.referencia === ref))) {
    errores.push({ campo: 'seleccion-desempate-0', clave: 'validacion.desempates' });
  }
  return errores;
}

export function validarEjemplos(datos) {
  if (!Array.isArray(datos?.ejemplos) || datos.ejemplos.length > 32) throw new Error('seleccion.catalogo');
  const referencias = new Set();
  for (const ejemplo of datos.ejemplos) {
    const c = ejemplo?.configuracion;
    if (!cadena(ejemplo.referencia) || referencias.has(ejemplo.referencia)
      || !cadena(ejemplo.titulo_clave) || !cadena(ejemplo.convocatoria_ref) || !entero(ejemplo.bases_version) || ejemplo.bases_version < 1
      || !entero(c?.version) || c.version < 1 || !['libre', 'promocion_interna', 'discapacidad'].includes(c.turno_acceso)
      || !['bolsa', 'plaza'].includes(c.destino) || validarConfiguracion(c).length
      || c.fases.some(f => !cadena(f.referencia) || !['prueba', 'meritos'].includes(f.tipo) || !entero(f.maximo_micropuntos) || f.maximo_micropuntos < 1 || f.maximo_micropuntos > 1000000000)
      || new Set(c.fases.map(f => f.referencia)).size !== c.fases.length) throw new Error('seleccion.catalogo');
    referencias.add(ejemplo.referencia);
    validarNotasEjemplo(ejemplo);
  }
  return datos.ejemplos;
}

export function validarResultado(r, datos) {
  const c = datos.configuracion;
  const causas = valores => Array.isArray(valores) && valores.length <= 32 && valores.every(v => CAUSAS.includes(v));
  if (!r || r.version !== c.version || r.convocatoria_ref !== datos.convocatoria_ref
    || r.bases_version !== datos.bases_version || r.plazas !== c.plazas || r.alcance !== 'ensayo_sintetico'
    || r.modalidad !== c.modalidad || r.turno_acceso !== c.turno_acceso
    || r.destino !== c.destino || !['provisional', 'indeterminado'].includes(r.estado) || !causas(r.causas)
    || !Array.isArray(r.solicitudes) || r.solicitudes.length > 1000) throw new Error('seleccion.resultado');
  const referencias = new Set();
  for (const s of r.solicitudes) {
    if (!cadena(s.referencia) || referencias.has(s.referencia) || !cadena(s.nombre)
      || !['cumple', 'no_cumple', 'pendiente'].includes(s.acceso)
      || !['apta', 'no_apta', 'pendiente', 'empate_pendiente'].includes(s.estado)
      || !['propuesta_provisional', 'sin_propuesta', 'pendiente'].includes(s.propuesta)
      || !(s.total_micropuntos === null || entero(s.total_micropuntos))
      || !(s.orden === null || (entero(s.orden) && s.orden > 0))
      || (s.estado === 'empate_pendiente' && s.orden !== null) || !causas(s.causas)
      || !Array.isArray(s.acceso_detalle) || s.acceso_detalle.length > 32
      || s.acceso_detalle.some(a => !cadena(a.referencia)
        || !['cumple', 'no_cumple', 'pendiente'].includes(a.estado)
        || (a.causa && !CAUSAS.includes(a.causa)))
      || !Array.isArray(s.fases) || s.fases.length !== c.fases.length
      || s.fases.some((f, i) => f.referencia !== c.fases[i].referencia
        || f.tipo !== c.fases[i].tipo || f.peso !== c.fases[i].peso
        || f.minimo_micropuntos !== c.fases[i].minimo_micropuntos || f.maximo_micropuntos !== c.fases[i].maximo_micropuntos
        || !(f.puntos_micropuntos === null || entero(f.puntos_micropuntos))
        || !['superada', 'no_superada', 'pendiente'].includes(f.estado)
        || !['prueba_embebida', 'prueba_editada', 'motor_bolsa'].includes(f.origen)
        || !Array.isArray(f.reglas) || f.reglas.length > 256
        || f.reglas.some(regla => !cadena(regla.referencia) || !entero(regla.puntos_micropuntos)))) throw new Error('seleccion.resultado');
    referencias.add(s.referencia);
  }
  return r;
}
