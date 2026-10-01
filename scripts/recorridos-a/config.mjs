import fs from 'node:fs';
import path from 'node:path';
import { createHash } from 'node:crypto';

const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const SHA = /^[0-9a-f]{64}$/u;
const ORIGEN_LOCAL = new Set(['localhost', '127.0.0.1', '[::1]']);
export const sha256 = value => createHash('sha256').update(value).digest('hex');
const exigir = (ok, codigo = 'entrada') => { if (!ok) throw new Error(codigo); };

// El inventario, el material mTLS y los resultados viven fuera de Git.
export function rutaPrivada(value, { directorio = false, nueva = false } = {}) {
  exigir(typeof value === 'string' && path.isAbsolute(value) && !value.split('/').includes('..'));
  let actual = '/';
  const partes = value.split('/').filter(Boolean);
  for (let i = 0; i < partes.length; i++) {
    actual = path.join(actual, partes[i]);
    const ultimo = i === partes.length - 1;
    if (ultimo && nueva) { exigir(!fs.existsSync(actual)); break; }
    const st = fs.lstatSync(actual);
    exigir(!st.isSymbolicLink());
    if (!ultimo || directorio) exigir(st.isDirectory());
    else exigir(st.isFile() && st.nlink === 1 && st.size > 0 && st.size <= 2 * 1024 * 1024);
    if (st.isDirectory()) {
      exigir(!fs.existsSync(path.join(actual, '.git')));
      exigir(!(fs.existsSync(path.join(actual, 'HEAD')) && fs.existsSync(path.join(actual, 'objects'))));
    }
    if (ultimo) exigir(st.uid === process.getuid() && (st.mode & 0o077) === 0);
  }
  if (nueva) rutaPrivada(path.dirname(value), { directorio: true });
  return value;
}

export function origen(value, { auxiliar = false } = {}) {
  const u = new URL(value);
  exigir((u.protocol === 'https:' || auxiliar && u.protocol === 'wss:')
    && ORIGEN_LOCAL.has(u.hostname) && u.port && u.pathname === '/' && !u.search && !u.hash
    && !u.username && !u.password);
  return u.origin;
}

export function cargarConfig(file) {
  const c = JSON.parse(fs.readFileSync(rutaPrivada(file), 'utf8'));
  exigir(c?.version === 1 && c?.datos === 'sinteticos' && c?.entorno_controlado === true
    && typeof c.expediente_ref === 'string' && REF.test(c.expediente_ref)
    && typeof c.commit_servido === 'string' && /^[0-9a-f]{40}$/u.test(c.commit_servido)
    && c.identidad && typeof c.identidad === 'object'
    && c.validacion && typeof c.validacion === 'object'
    && c.validacion.b2_instalado === true && c.validacion.e3_instalado === true
    && c.validacion.autofirma_preparada === true && c.validacion.grxfirma_preparada === true);
  c.origen = origen(c.origen);
  exigir(typeof c.identidad.perfil_ref === 'string' && REF.test(c.identidad.perfil_ref));
  exigir(typeof c.identidad.certificado_firma_sha256 === 'string' && SHA.test(c.identidad.certificado_firma_sha256));
  c.identidad.certificado = rutaPrivada(c.identidad.certificado);
  c.identidad.clave = rutaPrivada(c.identidad.clave);
  if (c.capturas !== undefined) c.capturas = rutaPrivada(c.capturas, { directorio: true });
  exigir(sha256(fs.readFileSync(c.identidad.certificado)) !== sha256(fs.readFileSync(c.identidad.clave)));
  c.auxiliares = (c.auxiliares ?? []).map(v => origen(v, { auxiliar: true }));
  exigir(new Set(c.auxiliares).size === c.auxiliares.length && !c.auxiliares.includes(c.origen));
  exigir(c.seleccion && typeof c.seleccion === 'object'
    && ['vacante', 'regimen', 'modalidad', 'clase_ocupacion', 'motivo', 'documento', 'desde', 'hasta'].every(k => Object.hasOwn(c.seleccion, k)));
  const s = c.seleccion;
  const catalogo = v => v && typeof v === 'object' && REF.test(v.ref) && Number.isSafeInteger(v.version) && v.version >= 1;
  exigir(s.vacante && typeof s.vacante === 'object'
    && ['plaza_ref', 'puesto_ref', 'version_plantilla_ref', 'version_rpt_ref'].every(k => REF.test(s.vacante[k]))
    && catalogo(s.regimen) && catalogo(s.modalidad)
    && typeof s.clase_ocupacion === 'string' && s.clase_ocupacion.length < 160
    && typeof s.motivo === 'string' && s.motivo.length < 160
    && s.documento && REF.test(s.documento.documento_ref) && SHA.test(s.documento.documento_sha256)
    && /^\d{4}-\d{2}-\d{2}$/u.test(s.desde) && (s.hasta === '' || /^\d{4}-\d{2}-\d{2}$/u.test(s.hasta)));
  exigir(fs.statSync('/usr/bin/google-chrome').isFile());
  fs.accessSync('/usr/bin/google-chrome', fs.constants.X_OK);
  return c;
}

export function prepararEstado(file, anterior = false) {
  if (anterior) return JSON.parse(fs.readFileSync(rutaPrivada(file), 'utf8'));
  rutaPrivada(file, { nueva: true });
  return null;
}

export function guardarEstado(file, data) {
  // Se crea antes de cada efecto; un resultado ambiguo nunca invita a repetir POST.
  const temporal = `${file}.tmp`;
  exigir(!fs.existsSync(temporal));
  const descriptor = fs.openSync(temporal, 'wx', 0o600);
  try { fs.writeFileSync(descriptor, JSON.stringify(data, null, 2)); fs.fsyncSync(descriptor); }
  finally { fs.closeSync(descriptor); }
  fs.renameSync(temporal, file);
  const padre = fs.openSync(path.dirname(file), 'r');
  try { fs.fsyncSync(padre); }
  finally { fs.closeSync(padre); }
}

export function huellaEscenario(c) {
  return sha256(JSON.stringify([c.origen, c.expediente_ref, c.commit_servido, c.seleccion]));
}

export function validarEstadoAnterior(estado, c) {
  exigir(estado?.version === 1 && estado.escenario_sha256 === huellaEscenario(c)
    && Array.isArray(estado.firmas) && estado.b2 && typeof estado.b2 === 'object');
  return estado;
}

export function validarReciboFirma(r, expediente, documento, orden, identidad) {
  exigir(r?.esquema === 'vec.contratacion-temporal.recibo-firma-documento.v1'
    && r.expediente_ref === expediente && r.documento === documento && r.paso_orden === orden
    && r.resultado === 'firmado' && r.perfil_ref === identidad.perfil_ref && r.firma_verificada === true && r.firma_eficaz === false
    && r.verificacion?.estado === 'valida' && r.verificacion?.motivo === 'verificada'
    && r.verificacion?.certificado_sha256 === identidad.certificado_firma_sha256
    && SHA.test(r.verificacion.firmado_sha256) && REF.test(r.recibo_ref)
    && typeof r.registrada_en === 'string' && !Number.isNaN(Date.parse(r.registrada_en)));
  if (r.documento_custodiado !== undefined) exigir(r.documento_custodiado
    && /^ref:[0-9a-f]{64}$/u.test(r.documento_custodiado.expediente_ref)
    && /^ref:[0-9a-f]{64}$/u.test(r.documento_custodiado.documento_ref)
    && Number.isSafeInteger(r.documento_custodiado.version) && r.documento_custodiado.version >= 1
    && r.documento_custodiado.huella_sha256 === r.verificacion.firmado_sha256);
  return { documento, orden, recibo_ref: r.recibo_ref, registrada_en: r.registrada_en,
    firmado_sha256: r.verificacion.firmado_sha256, certificado_sha256: r.verificacion.certificado_sha256,
    custodiado: r.documento_custodiado ?? null };
}
