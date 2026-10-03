// Contrato de transporte del CLI; la validación de negocio permanece en Bolsa.
export const MAXIMO_BYTES = 4 * 1024 * 1024;
export const CAMPOS_REFERENCIA = ['fuente_bases', 'catalogos', 'calendario', 'reglas_baremacion', 'flujo_proceso', 'flujo_solicitud', 'plantilla', 'plaza', 'oep', 'rpt'];
const CIRCUITOS = ['documentos_admitidos', 'firma_y_custodia', 'acto_aprobacion', 'publicacion_oficial'];
const CAMPOS = [...CAMPOS_REFERENCIA, ...CIRCUITOS, 'identificador_publico', 'tipo', 'titulo', 'resumen', 'catalogo_categorias', 'categorias', 'plazos', 'documentos_propuestos', 'contenido'];
const CODIGOS = ['referencia_ausente', 'referencia_invalida', 'referencia_no_verificada', 'material_ausente', 'contenido_no_validado', 'circuito_pendiente'];
const objeto = v => v !== null && typeof v === 'object' && !Array.isArray(v);
const cadena = v => typeof v === 'string' && v.length <= 131072;
const entero = v => Number.isSafeInteger(v);
const lista = validar => v => v === null || Array.isArray(v) && v.length <= 1024 && v.every(validar);
export function forma(v, campos, opcionales = []) {
  return objeto(v) && Object.keys(v).every(k => Object.hasOwn(campos, k) || opcionales.includes(k))
    && Object.entries(campos).every(([k, validar]) => Object.hasOwn(v, k) && validar(v[k]));
}
export const ref = v => forma(v, { id: cadena, version: entero, huella_contenido_sha256: cadena });
const catalogo = v => forma(v, { catalogo_id: cadena, catalogo_version: entero, catalogo_huella_sha256: cadena });
const plazo = v => forma(v, { referencia: cadena, tipo: cadena, titulo: cadena, descripcion: cadena, abre_en: cadena, cierra_en: cadena });
const requisito = v => forma(v, { referencia: cadena, orden: entero, titulo: cadena, descripcion: cadena, obligatorio: v => typeof v === 'boolean' });
const documento = v => forma(v, { referencia: cadena, tipo: cadena, orden: entero, titulo: cadena, descripcion: cadena, formato: cadena, url: cadena });
const ayuda = v => forma(v, { referencia: cadena, categoria: cadena, orden: entero, pregunta: cadena, respuesta: cadena });
export const contenido = v => forma(v, {
  identificador_publico: cadena, tipo: cadena, catalogo_categorias: catalogo, categorias: lista(cadena),
  titulo: cadena, resumen: cadena, descripcion: cadena, plazos: lista(plazo), requisitos: lista(requisito), documentos: lista(documento), ayuda: lista(ayuda),
});
export const pendiente = v => forma(v, { campo: v => CAMPOS.includes(v), codigo: v => CODIGOS.includes(v) });
const mensaje = v => forma(v, { campo: cadena, codigo: cadena, mensaje: cadena });

// JSON.parse pierde claves duplicadas. Este recorrido léxico las rechaza antes de interpretar.
export function comprobarClaves(texto) {
  let i = 0;
  const espacio = () => { while (/\s/u.test(texto[i] ?? '') && i < texto.length) ++i; };
  function tokenCadena() {
    const inicio = i++; let escape = false;
    while (i < texto.length) {
      const c = texto[i++];
      if (!escape && c === '"') return JSON.parse(texto.slice(inicio, i));
      if (!escape && c === '\\') escape = true; else escape = false;
    }
    throw new TypeError('formato');
  }
  function valor(nivel) {
    if (nivel > 32) throw new TypeError('formato');
    espacio(); const c = texto[i];
    if (c === '"') { tokenCadena(); return; }
    if (c !== '{' && c !== '[') { while (i < texto.length && !/[\s,}\]]/u.test(texto[i])) ++i; return; }
    ++i; espacio(); const fin = c === '{' ? '}' : ']'; const claves = new Set();
    if (texto[i] === fin) { ++i; return; }
    while (i < texto.length) {
      espacio();
      if (c === '{') {
        if (texto[i] !== '"') throw new TypeError('formato');
        const clave = tokenCadena();
        if (claves.has(clave) || ['__proto__', 'constructor', 'prototype'].includes(clave)) throw new TypeError('formato');
        claves.add(clave); espacio(); if (texto[i++] !== ':') throw new TypeError('formato');
      }
      valor(nivel + 1); espacio();
      if (texto[i] === fin) { ++i; return; }
      if (texto[i++] !== ',') throw new TypeError('formato');
    }
    throw new TypeError('formato');
  }
  valor(0); espacio(); if (i !== texto.length) throw new TypeError('formato');
}

export function leerSalida(bytes) {
  if (!(bytes instanceof Uint8Array) || bytes.byteLength === 0 || bytes.byteLength > MAXIMO_BYTES) throw new TypeError('tamano');
  let dto;
  try {
    const texto = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
    comprobarClaves(texto); dto = JSON.parse(texto);
  } catch { throw new TypeError('formato'); }
  const material = v => forma(v, { alcance: v => v === 'preparacion_sintetica', identidad_material: cadena,
    version_material: v => entero(v) && v >= 1 && v <= 1000000, contenido, referencias: v => forma(v, Object.fromEntries(CAMPOS_REFERENCIA.map(k => [k, ref]))) });
  const preparacion = v => forma(v, { estado: v => v === 'pendiente', material_propuesto: material,
    pendientes: v => Array.isArray(v) && v.length <= 64 && v.every(pendiente) }, ['contenido_canonico_bolsa'])
    && (!Object.hasOwn(v, 'contenido_canonico_bolsa') || contenido(v.contenido_canonico_bolsa));
  if (!forma(dto, { preparacion, limite: cadena, mensajes: v => Array.isArray(v) && v.length <= 64 && v.every(mensaje) })) throw new TypeError('formato');
  const p = dto.preparacion.pendientes; const vistos = new Set();
  if (p.length !== dto.mensajes.length || p.some((item, i) => {
    if (vistos.has(item.campo) || item.campo !== dto.mensajes[i].campo || item.codigo !== dto.mensajes[i].codigo) return true;
    vistos.add(item.campo); return false;
  }) || CAMPOS_REFERENCIA.some(c => !p.some(x => x.campo === c && x.codigo.startsWith('referencia_')))
    || CIRCUITOS.some(c => !p.some(x => x.campo === c && x.codigo === 'circuito_pendiente'))) throw new TypeError('formato');
  return dto;
}

export async function leerArchivo(archivo) {
  if (!archivo || archivo.size <= 0 || archivo.size > MAXIMO_BYTES) throw new TypeError('tamano');
  const bytes = new Uint8Array(await archivo.arrayBuffer());
  return { bytes, dto: leerSalida(bytes) };
}
