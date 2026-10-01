import '../contrato-v2.js?v=20260930-codexe-publico-v2-v1';

const contrato = globalThis.VECBolsaContratoV2;
const MAXIMO_RESPUESTA = 2 * 1024 * 1024;

export function validarIdentificador(valor) {
  return typeof valor === 'string' && /^[a-z0-9][a-z0-9-]{2,79}$/u.test(valor);
}

export function rutaDetalle(identificador, idioma) {
  if (!validarIdentificador(identificador)) throw new TypeError('identificador_invalido');
  const parametros = new URLSearchParams({ idioma });
  return `/api/publico/bolsa/convocatorias/${identificador}?${parametros}`;
}

export function validarLimites(datos) {
  const maximoArchivos = Number(datos.maximo_archivos);
  const maximoBytesArchivo = Number(datos.maximo_bytes_archivo);
  if (!Number.isSafeInteger(maximoArchivos) || maximoArchivos < 1 || maximoArchivos > 100 ||
      !Number.isSafeInteger(maximoBytesArchivo) || maximoBytesArchivo < 1 || maximoBytesArchivo > 100 * 1024 * 1024) {
    throw new TypeError('limites_invalidos');
  }
  return Object.freeze({ maximoArchivos, maximoBytesArchivo });
}

// Solo metadatos: no conservamos objetos File, rutas ni bytes del fichero.
export function seleccionarFicheros(ficheros, limites) {
  const lista = Array.from(ficheros ?? []);
  if (lista.length > limites.maximoArchivos) throw new RangeError('cantidad_error');
  return lista.map((fichero) => {
    if (!Number.isSafeInteger(fichero.size) || fichero.size < 0) throw new TypeError('datos_error');
    if (fichero.size > limites.maximoBytesArchivo) throw new RangeError('tamano_error');
    if (typeof fichero.name !== 'string' || !fichero.name.trim() || fichero.name.length > 255 ||
        /[\/\\\u0000-\u001f\u007f]/u.test(fichero.name) || ['.', '..'].includes(fichero.name)) {
      throw new TypeError('nombre_error');
    }
    return Object.freeze({ nombre: fichero.name, tamano: fichero.size });
  });
}

export function crearLectorDetalle({ fetchImpl = globalThis.fetch } = {}) {
  let controlador;
  let secuencia = 0;
  let cerrado = false;
  return {
    async leer(identificador, idioma) {
      if (cerrado) throw new Error('lector_cerrado');
      const ruta = rutaDetalle(identificador, idioma);
      controlador?.abort();
      controlador = new AbortController();
      const signal = controlador.signal;
      const actual = ++secuencia;
      const respuesta = await fetchImpl(ruta, {
        method: 'GET', credentials: 'omit', cache: 'no-store', redirect: 'error',
        referrerPolicy: 'no-referrer', headers: { Accept: 'application/json' }, signal,
      });
      if (!respuesta.ok) throw new Error(respuesta.status === 404 || respuesta.status === 503 ? 'no_fuente' : 'error');
      if (!respuesta.headers.get('content-type')?.includes('application/json')) throw new Error('error');
      if (Number(respuesta.headers.get('content-length')) > MAXIMO_RESPUESTA) throw new Error('error');
      const contenido = await respuesta.text();
      if (contenido.length > MAXIMO_RESPUESTA) throw new Error('error');
      const datos = JSON.parse(contenido);
      if (cerrado || actual !== secuencia || signal.aborted) return null;
      contrato.validarDetalle(datos);
      if (datos.convocatoria.identificador_publico !== identificador) throw new Error('error');
      if (datos.fuente.demostracion) throw new Error('no_fuente');
      return datos;
    },
    desmontar() {
      cerrado = true;
      ++secuencia;
      controlador?.abort();
    },
  };
}

export function crearResumen(detalle, { archivos = [], lectura = false, limites, fecha = new Date() } = {}) {
  contrato.validarDetalle(detalle);
  if (detalle.fuente.demostracion) throw new Error('no_fuente');
  const metadatos = seleccionarFicheros(archivos.map(({ nombre, tamano }) => ({ name: nombre, size: tamano })), limites);
  return {
    esquema: 'vec.bolsa.preparacion-local.v1', estado: 'sin_presentar',
    convocatoria: { identificador_publico: detalle.convocatoria.identificador_publico,
      titulo: detalle.convocatoria.titulo, version: detalle.convocatoria.version,
      huella_sha256: detalle.convocatoria.huella_sha256 },
    generada_en: fecha.toISOString(), lectura_confirmada: lectura === true,
    requisitos: detalle.requisitos.map(({ titulo, descripcion, obligatorio }) => ({ titulo, descripcion, obligatorio,
      cumplimiento: 'pendiente' })),
    plazos: detalle.plazos.map(({ titulo, abre_en, cierra_en, etiqueta_situacion, descripcion }) =>
      ({ titulo, abre_en, cierra_en, etiqueta_situacion, descripcion })),
    documentos_publicos: detalle.documentos.map(({ titulo, url }) => ({ titulo, url })),
    archivos_locales: metadatos,
  };
}
