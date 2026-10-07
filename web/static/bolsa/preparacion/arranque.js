import { cargarTextos } from '../../comun/textos.js';
import { montarSelectorIdioma } from '../../comun/idioma.js';
import { validarIdentificador, validarLimites, crearLectorDetalle, seleccionarFicheros, crearResumen } from './modelo.js?v=20261001-preparacion-v1';

export function textoResumen(resumen, textos) {
  const t = textos.traducir;
  const fecha = (valor) => textos.fecha(valor, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Europe/Madrid' });
  return [t('resumen.titulo_descarga'), t('resumen.estado'), t('resumen.limite'),
    t('resumen.fecha', { fecha: fecha(resumen.generada_en) }),
    t('resumen.convocatoria', resumen.convocatoria),
    t('resumen.identificador', { identificador: resumen.convocatoria.identificador_publico }),
    t('pagina.version', resumen.convocatoria), t('pagina.huella', { huella: resumen.convocatoria.huella_sha256 }),
    t(resumen.lectura_confirmada ? 'resumen.lectura' : 'resumen.sin_lectura'),
    '', t('convocatoria.requisitos'), t('resumen.verificacion'),
    ...resumen.requisitos.map((r) => `${r.titulo}: ${r.descripcion} (${t(r.obligatorio ? 'convocatoria.obligatorio' : 'convocatoria.opcional')})`),
    '', t('convocatoria.plazos'),
    ...resumen.plazos.map((p) => `${p.titulo}: ${t('convocatoria.desde_hasta', { desde: fecha(p.abre_en), hasta: fecha(p.cierra_en) })}; ${p.etiqueta_situacion}${p.descripcion ? `; ${p.descripcion}` : ''}`),
    '', t('convocatoria.documentos'), ...resumen.documentos_publicos.map((d) => `${d.titulo}: ${d.url}`),
    '', t('archivos.titulo'), t('archivos.descarga_aviso'),
    ...(resumen.archivos_locales.length ? resumen.archivos_locales.map((a) => `${a.nombre} (${t('archivos.tamano', { tamano: textos.numero(a.tamano) })})`) : [t('archivos.sin_archivos')]), '',
  ].join('\n');
}

export async function montarPreparacion(documento = globalThis.document, ubicacion = globalThis.location) {
  const porId = (id) => documento.getElementById(id);
  if (!porId('preparacion')) return null;
  const textos = await cargarTextos('convoca-preparacion');
  const t = textos.traducir;
  documento.documentElement.lang = textos.idioma;
  for (const elemento of documento.querySelectorAll('[data-texto], [data-aria]')) {
    if (elemento.dataset.texto) elemento.textContent = t(elemento.dataset.texto);
    if (elemento.dataset.aria) elemento.setAttribute('aria-label', t(elemento.dataset.aria));
  }
  montarSelectorIdioma(porId('idioma-preparacion'), ubicacion);
  const limites = validarLimites(textos.seccion('limites'));
  const identificador = new URL(ubicacion.href).searchParams.get('convocatoria');
  const enlace = new URLSearchParams({ lang: textos.idioma });
  porId('inicio-preparacion').href = `/bolsa/?${enlace}`;
  if (validarIdentificador(identificador)) enlace.set('convocatoria', identificador);
  porId('volver-convocatoria').href = `/bolsa/?${enlace}`;
  const lector = crearLectorDetalle();
  let detalle = null;
  let archivos = [];
  let paso = 1;
  let carga = 0;
  let cerrado = false;
  let urlDescarga;
  let resumenActual = null;
  const fecha = (valor) => textos.fecha(valor, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Europe/Madrid' });
  const nodo = (tag, valor, clase) => {
    const elemento = documento.createElement(tag);
    elemento.textContent = valor;
    if (clase) elemento.className = clase;
    return elemento;
  };
  function mostrarLista(id, entradas, vacio, construir) {
    porId(id).replaceChildren(...(entradas.length ? entradas.map(construir) : [nodo('li', t(vacio))]));
  }
  function pintarArchivos() {
    porId('cantidad-archivos').textContent = archivos.length ? textos.plural('archivos.cantidad', archivos.length) : t('archivos.sin_archivos');
    porId('lista-archivos').replaceChildren(...archivos.map((a, indice) => {
      const fila = nodo('li', `${a.nombre} (${t('archivos.tamano', { tamano: textos.numero(a.tamano) })}) `);
      const quitar = nodo('button', t('archivos.quitar', a), 'boton-secundario');
      quitar.type = 'button';
      quitar.addEventListener('click', () => {
        archivos = archivos.filter((_, posicion) => posicion !== indice);
        pintarArchivos();
        const controles = porId('lista-archivos').querySelectorAll('button');
        (controles[Math.min(indice, controles.length - 1)] ?? porId('archivos-locales')).focus();
      });
      fila.append(quitar);
      return fila;
    }));
  }
  function pintarPaso(enfocar = false) {
    const bloques = ['paso-revision', 'paso-archivos', 'paso-resumen'];
    bloques.forEach((id, indice) => { porId(id).hidden = !detalle || paso !== indice + 1; });
    for (const indicador of documento.querySelectorAll('[data-paso]')) {
      const activo = Number(indicador.dataset.paso) === paso;
      indicador.dataset.semantica = activo ? 'informacion' : '';
      if (activo) indicador.setAttribute('aria-current', 'step');
      else indicador.removeAttribute('aria-current');
    }
    porId('numero-paso').textContent = detalle ? t('pagina.paso', { numero: textos.numero(paso) }) : '';
    porId('anterior').hidden = !detalle || paso === 1;
    porId('siguiente').hidden = !detalle || paso === 3;
    porId('descargar').hidden = !detalle || paso !== 3;
    porId('descargar-json').hidden = !detalle || paso !== 3;
    resumenActual = null;
    if (detalle && paso === 3) {
      resumenActual = crearResumen(detalle, { archivos, lectura: porId('lectura-confirmada').checked, limites });
      const elemento = nodo('pre', textoResumen(resumenActual, textos), 'resumen-local');
      porId('revision-resumen').replaceChildren(elemento);
    }
    porId('cuerpo-preparacion').scrollTop = 0;
    if (enfocar) {
      porId(bloques[paso - 1]).setAttribute('tabindex', '-1');
      porId(bloques[paso - 1]).focus();
    }
  }
  function pintarDetalle() {
    porId('titulo-convocatoria').textContent = detalle.convocatoria.titulo;
    porId('descripcion-convocatoria').textContent = detalle.descripcion;
    mostrarLista('requisitos', detalle.requisitos, 'convocatoria.sin_requisitos', (r) => {
      const li = documento.createElement('li');
      li.append(nodo('h3', r.titulo), nodo('p', r.descripcion), nodo('span', t(r.obligatorio ? 'convocatoria.obligatorio' : 'convocatoria.opcional'), 'etiqueta'));
      return li;
    });
    mostrarLista('plazos', detalle.plazos, 'convocatoria.sin_plazos', (p) => {
      const li = documento.createElement('li');
      li.append(nodo('h3', p.titulo), nodo('p', t('convocatoria.desde_hasta', { desde: fecha(p.abre_en), hasta: fecha(p.cierra_en) })), nodo('span', p.etiqueta_situacion, 'etiqueta'));
      if (p.descripcion) li.append(nodo('p', p.descripcion));
      return li;
    });
    mostrarLista('documentos', detalle.documentos, 'convocatoria.sin_documentos', (d) => {
      const li = documento.createElement('li');
      const a = nodo('a', t('convocatoria.documento', d), 'enlace-documento');
      a.href = d.url;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
      li.append(a, nodo('p', d.descripcion));
      return li;
    });
    porId('version').textContent = t('pagina.version', detalle.convocatoria);
    porId('huella').textContent = t('pagina.huella', { huella: detalle.convocatoria.huella_sha256 });
    porId('version-convocatoria').hidden = false;
    pintarPaso();
  }
  async function cargar() {
    const actual = ++carga;
    detalle = null;
    archivos = [];
    paso = 1;
    porId('lectura-confirmada').checked = false;
    porId('lectura-confirmada').setAttribute('aria-invalid', 'false');
    porId('error-lectura').textContent = '';
    porId('version-convocatoria').hidden = true;
    porId('reintentar').hidden = true;
    porId('trabajo-preparacion').setAttribute('aria-busy', 'true');
    pintarPaso();
    if (!validarIdentificador(identificador)) {
      porId('estado-preparacion').textContent = t('pagina.invalida');
      porId('trabajo-preparacion').setAttribute('aria-busy', 'false');
      return;
    }
    porId('estado-preparacion').textContent = t('pagina.cargando');
    try {
      const datos = await lector.leer(identificador, textos.idioma);
      if (cerrado || actual !== carga || !datos) return;
      detalle = datos;
      porId('estado-preparacion').textContent = '';
      pintarDetalle();
    } catch (error) {
      if (cerrado || actual !== carga) return;
      porId('estado-preparacion').textContent = t(error.message === 'no_fuente' ? 'pagina.no_fuente' : 'pagina.error');
      porId('reintentar').hidden = false;
    } finally {
      if (!cerrado && actual === carga) porId('trabajo-preparacion').setAttribute('aria-busy', 'false');
    }
  }
  porId('limites-archivos').textContent = t('archivos.limite', {
    cuenta: textos.numero(limites.maximoArchivos), tamano: textos.numero(limites.maximoBytesArchivo / 1024 / 1024),
  });
  pintarArchivos();
  porId('archivos-locales').addEventListener('change', (evento) => {
    try {
      const nuevos = seleccionarFicheros(evento.target.files, limites);
      archivos = nuevos;
      porId('error-archivos').textContent = '';
      evento.target.setAttribute('aria-invalid', 'false');
      pintarArchivos();
    } catch (error) {
      const clave = ['cantidad_error', 'tamano_error', 'nombre_error', 'datos_error'].includes(error.message) ? error.message : 'datos_error';
      porId('error-archivos').textContent = t(`archivos.${clave}`, { limite: textos.numero(clave === 'cantidad_error' ? limites.maximoArchivos : limites.maximoBytesArchivo / 1024 / 1024) });
      evento.target.setAttribute('aria-invalid', 'true');
    }
    evento.target.value = '';
  });
  porId('lectura-confirmada').addEventListener('change', () => {
    porId('error-lectura').textContent = '';
    porId('lectura-confirmada').setAttribute('aria-invalid', 'false');
  });
  porId('siguiente').addEventListener('click', () => {
    if (!detalle) return;
    if (paso === 1 && !porId('lectura-confirmada').checked) {
      porId('lectura-confirmada').setAttribute('aria-invalid', 'true');
      porId('error-lectura').textContent = t('convocatoria.lectura_error');
      porId('lectura-confirmada').focus();
      return;
    }
    paso = Math.min(3, paso + 1);
    pintarPaso(true);
  });
  porId('anterior').addEventListener('click', () => { if (detalle) { paso = Math.max(1, paso - 1); pintarPaso(true); } });
  porId('reintentar').addEventListener('click', cargar);
  function descargarResumen(json = false) {
    if (!detalle || paso !== 3 || !resumenActual) return;
    if (urlDescarga) URL.revokeObjectURL(urlDescarga);
    const contenido = json ? JSON.stringify(resumenActual, null, 2) : textoResumen(resumenActual, textos);
    const tipo = json ? 'application/json;charset=utf-8' : 'text/plain;charset=utf-8';
    urlDescarga = URL.createObjectURL(new Blob([contenido], { type: tipo }));
    const enlaceDescarga = documento.createElement('a');
    enlaceDescarga.href = urlDescarga;
    enlaceDescarga.download = t(json ? 'resumen.archivo_descarga_json' : 'resumen.archivo_descarga', { identificador });
    enlaceDescarga.click();
    porId('estado-preparacion').textContent = t('pagina.descargado');
  }
  porId('descargar').addEventListener('click', () => descargarResumen());
  porId('descargar-json').addEventListener('click', () => descargarResumen(true));
  const avisarSalida = (evento) => {
    if (!porId('lectura-confirmada').checked && archivos.length === 0) return;
    evento.preventDefault();
    evento.returnValue = '';
  };
  globalThis.addEventListener?.('beforeunload', avisarSalida);
  const desmontar = () => {
    globalThis.removeEventListener?.('beforeunload', avisarSalida);
    cerrado = true;
    ++carga;
    lector.desmontar();
    detalle = null;
    archivos = [];
    porId('archivos-locales').value = '';
    porId('lectura-confirmada').checked = false;
    porId('lectura-confirmada').setAttribute('aria-invalid', 'false');
    for (const id of ['lista-archivos', 'revision-resumen', 'cantidad-archivos', 'error-archivos', 'error-lectura', 'estado-preparacion']) porId(id).replaceChildren();
    pintarPaso();
    if (urlDescarga) URL.revokeObjectURL(urlDescarga);
  };
  globalThis.addEventListener?.('pagehide', desmontar, { once: true });
  globalThis.addEventListener?.('pageshow', (evento) => { if (evento.persisted) ubicacion.reload(); });
  await cargar();
  return { desmontar };
}

if (globalThis.document?.getElementById('preparacion')) await montarPreparacion();
