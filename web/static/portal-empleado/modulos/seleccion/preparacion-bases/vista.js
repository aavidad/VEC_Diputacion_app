import { CAMPOS_REFERENCIA } from './modelo.js?v=20261003-s2-consulta-v1';

function nodo(d, etiqueta, texto, clase) {
  const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n;
}
function panel(d, titulo) {
  const n = nodo(d, 'section', undefined, 'panel'); const cabecera = nodo(d, 'header', undefined, 'cabecera-panel');
  cabecera.append(nodo(d, 'h2', titulo)); const cuerpo = nodo(d, 'div', undefined, 'cuerpo-panel pila'); n.append(cabecera, cuerpo); return { n, cuerpo };
}
function datos(d, pares) {
  const dl = nodo(d, 'dl', undefined, 'bases-datos');
  for (const [titulo, valor] of pares) { const div = nodo(d, 'div'); div.append(nodo(d, 'dt', titulo), nodo(d, 'dd', valor)); dl.append(div); }
  return dl;
}
function tabla(d, titulo, columnas, filas, t) {
  if (!filas.length) return nodo(d, 'p', t('sin_elementos'));
  const region = nodo(d, 'div', undefined, 'tabla-contenedor'); region.tabIndex = 0; region.setAttribute('role', 'region'); region.setAttribute('aria-label', titulo);
  const table = nodo(d, 'table', undefined, 'tabla-datos tabla-apilable bases-tabla'); const head = nodo(d, 'thead'); const titulos = nodo(d, 'tr');
  for (const columna of columnas) { const th = nodo(d, 'th', t(`campos.${columna}`)); th.scope = 'col'; titulos.append(th); } head.append(titulos);
  const body = nodo(d, 'tbody');
  for (const fila of filas) {
    const tr = nodo(d, 'tr');
    fila.forEach((valor, i) => { const td = nodo(d, i === 0 ? 'th' : 'td'); if (i === 0) td.scope = 'row'; else td.dataset.etiqueta = t(`campos.${columnas[i]}`); if (valor?.nodeType) td.append(valor); else td.textContent = valor; tr.append(td); }); body.append(tr);
  }
  table.append(head, body); region.append(table); return region;
}

export function pintarSalida({ raiz, dto, textos, institucional = false }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const material = institucional ? dto.preparacion.material : dto.preparacion.material_propuesto;
  const contenido = material.contenido; const pendientesOriginales = institucional ? dto.pendientes : dto.preparacion.pendientes;
  const valor = v => v === '' ? t('sin_dato') : v;
  const descripcion = item => { const div = nodo(d, 'div'); div.append(nodo(d, 'strong', valor(item.titulo)), nodo(d, 'p', valor(item.descripcion))); return div; };
  const fecha = v => {
    if (!/^\d{4}-\d{2}-\d{2}T/u.test(v) || v.startsWith('0001-') || !Number.isFinite(Date.parse(v))) return valor(v);
    return textos.fecha(v, { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Europe/Madrid' });
  };
  const fragmento = d.createDocumentFragment(); const resumen = nodo(d, 'div', undefined, 'bases-rejilla');
  const pendientes = panel(d, t('pendientes')); pendientes.n.classList.add('bases-pendiente');
  pendientes.cuerpo.append(nodo(d, 'p', textos.plural('recuento_pendientes', pendientesOriginales.length), 'texto-secundario'));
  const lista = nodo(d, 'ul', undefined, 'lista-comprobacion'); const comprobaciones = nodo(d, 'ul', undefined, 'lista-comprobacion');
  for (const p of pendientesOriginales) {
    const li = nodo(d, 'li', undefined, 'pendiente'); li.append(nodo(d, 'strong', t(`campos.${p.campo}`)), nodo(d, 'p', t(`motivos.${p.codigo}`)));
    (p.codigo === 'circuito_pendiente' ? lista : comprobaciones).append(li);
  }
  pendientes.cuerpo.append(lista);
  const resto = nodo(d, 'details'); resto.append(nodo(d, 'summary', textos.plural('comprobaciones_material', comprobaciones.children.length)), comprobaciones); pendientes.cuerpo.append(resto); resumen.append(pendientes.n);
  const propuesta = panel(d, contenido.titulo || t('propuesta')); propuesta.cuerpo.append(datos(d, [
    [t('campos.estado'), t('pendiente')], [t('campos.tipo'), Object.hasOwn(textos.mensajes.modalidades, contenido.tipo) ? t(`modalidades.${contenido.tipo}`) : valor(contenido.tipo)],
    [t('campos.resumen'), valor(contenido.resumen)], [t('campos.descripcion'), valor(contenido.descripcion)], [t('campos.categorias'), (contenido.categorias ?? []).join(', ') || t('sin_dato')],
  ])); resumen.append(propuesta.n); fragmento.append(resumen);
  const requisitos = panel(d, t('requisitos')); requisitos.cuerpo.append(tabla(d, t('requisitos'), ['requisito', 'obligatorio'], (contenido.requisitos ?? []).map(r => [descripcion(r), t(r.obligatorio ? 'si' : 'no')]), t)); fragmento.append(requisitos.n);
  const plazos = panel(d, t('plazos')); plazos.cuerpo.append(nodo(d, 'p', t('plazos_limite'), 'texto-secundario'), tabla(d, t('plazos'), ['plazo', 'desde', 'hasta'], (contenido.plazos ?? []).map(p => [descripcion(p), fecha(p.abre_en), fecha(p.cierra_en)]), t)); fragmento.append(plazos.n);
  const docs = panel(d, t('documentos')); docs.cuerpo.append(nodo(d, 'p', t('documentos_limite'), 'texto-secundario'), tabla(d, t('documentos'), ['documento', 'formato', 'ruta_propuesta'], (contenido.documentos ?? []).map(item => [descripcion(item), valor(item.formato), valor(item.url)]), t)); fragmento.append(docs.n);
  const refs = panel(d, t('referencias')); const r = institucional ? Object.fromEntries(material.referencias.map(x => [x.campo, x.referencia])) : material.referencias;
  const referencia = c => r[c] ?? { id: '', version: 0, huella_contenido_sha256: '' };
  refs.cuerpo.append(nodo(d, 'p', t('referencias_limite'), 'texto-secundario'), tabla(d, t('referencias'), ['referencia', 'identificador', 'version', 'comprobacion'], CAMPOS_REFERENCIA.map(c => [t(`campos.${c}`), valor(referencia(c).id), textos.numero(referencia(c).version), t(`motivos.${pendientesOriginales.find(p => p.campo === c).codigo}`)]), t));
  const huellas = nodo(d, 'details'); huellas.append(nodo(d, 'summary', t('huellas'))); huellas.append(datos(d, CAMPOS_REFERENCIA.map(c => [t(`campos.${c}`), valor(referencia(c).huella_contenido_sha256)]))); refs.cuerpo.append(huellas); fragmento.append(refs.n);
  const tecnico = panel(d, t(institucional ? 'detalle_consulta' : 'detalle_archivo')); const detalle = nodo(d, 'details'); detalle.append(nodo(d, 'summary', t(institucional ? 'ver_detalle_consulta' : 'ver_detalle_archivo')));
  if (institucional) {
    const e = dto.preparacion.estado;
    tecnico.cuerpo.append(nodo(d, 'p', t('origen_consulta')));
    detalle.append(datos(d, [[t('consulta_referencia'), e.preparacion_ref], [t('consulta_revision'), textos.numero(e.revision)], [t('consulta_huella'), e.huella_material_sha256]]));
    for (const [clave, evidencia, fechaClave] of [['recibo_preparacion', dto.recibo, 'confirmada_en'], ['acceso_consulta', dto.acceso, 'accedida_en']]) {
      detalle.append(nodo(d, 'h3', t(clave)), datos(d, [[t('recibo'), evidencia.recibo_ref], [t('fecha'), fecha(evidencia[fechaClave])]]));
    }
  } else detalle.append(datos(d, [[t('campos.identidad_material'), material.identidad_material], [t('campos.version_material'), textos.numero(material.version_material)], [t('campos.identificador_publico'), valor(contenido.identificador_publico)], [t('limite_original'), dto.limite]]));
  const original = nodo(d, 'pre', JSON.stringify(dto, null, 2)); original.className = 'bases-original'; detalle.append(original); tecnico.cuerpo.append(detalle); fragmento.append(tecnico.n);
  raiz.replaceChildren(fragmento);
}
