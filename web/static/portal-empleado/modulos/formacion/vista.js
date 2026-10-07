import { filtrarEdiciones, enlaceOficial } from './modelo.js?v=20261001-formacion-preparacion-v1';

function nodo(d, tag, texto, clase) { const n = d.createElement(tag); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n; }
function panel(d, titulo) { const n = nodo(d, 'section', undefined, 'panel'); const head = nodo(d, 'header', undefined, 'cabecera-panel'); head.append(nodo(d, 'h2', titulo)); const cuerpo = nodo(d, 'div', undefined, 'cuerpo-panel pila'); n.append(head, cuerpo); return { n, cuerpo }; }
function boton(d, texto, accion, foco, disabled = false) { const b = nodo(d, 'button', texto, 'boton-secundario'); b.type = 'button'; b.disabled = disabled; b.dataset.foco = foco; b.addEventListener('click', accion); return b; }
function lista(d, claves, t, dato) { const ul = nodo(d, 'ul', undefined, 'lista-comprobacion'); claves.forEach(clave => ul.append(nodo(d, 'li', t('pendiente_item', { tarea: dato(clave) }), 'pendiente'))); return ul; }
function datos(d, pares) { const dl = nodo(d, 'dl', undefined, 'resumen-expediente'); pares.forEach(([etiqueta, valor]) => { const div = nodo(d, 'div', undefined, 'fila-resumen'); div.append(nodo(d, 'dt', etiqueta), nodo(d, 'dd', valor)); dl.append(div); }); return dl; }

export function pintarFormacion({ raiz, escenario, textos, modalidad = '', seleccion = null, acciones }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const p = nodo(d, 'div', undefined, 'pila');
  const dato = clave => { if (!clave) return t('sin_dato'); try { return t(clave); } catch { return t('sin_dato'); } };
  const numero = v => v === null ? t('sin_dato') : textos.numero(v);
  const moneda = v => v === null ? t('sin_dato') : textos.numero(v / 100, { style: 'currency', currency: 'EUR' });
  const fecha = v => v === '' ? t('sin_dato') : textos.fecha(`${v}T12:00:00Z`, { dateStyle: 'medium', timeZone: 'Europe/Madrid' });
  const aviso = nodo(d, 'section', undefined, 'siguiente-paso en-espera'); const contenidoAviso = nodo(d, 'div'); contenidoAviso.append(nodo(d, 'h2', t('preparacion')), nodo(d, 'p', t('limite'))); aviso.append(contenidoAviso); p.append(aviso);
  const plan = panel(d, dato(escenario.plan.titulo_clave));
  plan.cuerpo.append(datos(d, [[t('campos.periodo'), t('periodo', { desde: fecha(escenario.plan.desde), hasta: fecha(escenario.plan.hasta) })], [t('campos.presupuesto'), moneda(escenario.plan.presupuesto_centimos)], [t('campos.plazas'), numero(escenario.plan.plazas)], [t('campos.version'), escenario.plan.configuracion.version || t('sin_dato')]]));
  const accionesPlan = nodo(d, 'div', undefined, 'acciones-vista'); const exportar = boton(d, t('exportar'), acciones.exportar, 'exportar'); exportar.className = 'boton-primario'; accionesPlan.append(exportar);
  for (const [campo, clave] of [['plan_url', 'ver_fuente'], ['plataforma_url', 'ver_plataforma']]) {
    const url = enlaceOficial(escenario.fuente_corporativa?.[campo], Object.values(textos.mensajes.fuentes_permitidas));
    if (url) { const a = nodo(d, 'a', t(clave), 'boton-secundario'); a.href = url; a.rel = 'noreferrer noopener'; a.referrerPolicy = 'no-referrer'; accionesPlan.append(a); }
  }
  plan.cuerpo.append(accionesPlan); p.append(plan.n);
  const ediciones = filtrarEdiciones(escenario, modalidad); const ed = panel(d, t('ediciones'));
  const controles = nodo(d, 'div', undefined, 'acciones-vista'); const campo = nodo(d, 'label', undefined, 'campo'); campo.append(nodo(d, 'span', t('campos.modalidad'))); const select = nodo(d, 'select'); select.dataset.foco = 'modalidad';
  for (const valor of ['', ...new Set([...escenario.plan.configuracion.modalidades, ...escenario.ediciones.map(e => e.modalidad_clave).filter(Boolean)])]) { const opt = nodo(d, 'option', valor === '' ? t('todas_modalidades') : dato(valor)); opt.value = valor; opt.selected = valor === modalidad; select.append(opt); }
  select.addEventListener('change', () => acciones.filtrar(select.value)); campo.append(select); controles.append(campo);
  if (modalidad) controles.append(boton(d, t('quitar_filtro'), () => acciones.filtrar(''), 'quitar-filtro')); ed.cuerpo.append(controles, nodo(d, 'p', textos.plural('recuento', ediciones.length), 'texto-secundario'));
  if (!ediciones.length) ed.cuerpo.append(nodo(d, 'p', t('sin_ediciones')));
  else {
    const region = nodo(d, 'div', undefined, 'tabla-contenedor'); region.tabIndex = 0; region.setAttribute('role', 'region'); region.setAttribute('aria-label', t('ediciones'));
    const table = nodo(d, 'table', undefined, 'tabla-datos tabla-apilable'); const head = nodo(d, 'thead'); const tr = nodo(d, 'tr'); const columnas = ['accion', 'modalidad', 'periodo', 'plazas', 'horas', 'detalle'];
    columnas.forEach(k => { const th = nodo(d, 'th', t(`campos.${k}`)); th.scope = 'col'; tr.append(th); }); head.append(tr); const body = nodo(d, 'tbody');
    ediciones.forEach(e => { const row = nodo(d, 'tr'); const b = boton(d, t('ver_detalle'), () => acciones.elegir(e.referencia), `edicion-${e.referencia}`); b.setAttribute('aria-label', t('abrir_edicion', { titulo: dato(e.titulo_clave), modalidad: dato(e.modalidad_clave) })); b.setAttribute('aria-expanded', String(e.referencia === seleccion));
      const valores = [dato(e.titulo_clave), dato(e.modalidad_clave), t('periodo', { desde: fecha(e.desde), hasta: fecha(e.hasta) }), numero(e.plazas), numero(e.horas), b]; valores.forEach((v, i) => { const cell = nodo(d, i === 0 ? 'th' : 'td'); if (i === 0) cell.scope = 'row'; else cell.dataset.etiqueta = t(`campos.${columnas[i]}`); if (v?.nodeType) cell.append(v); else cell.textContent = v; row.append(cell); }); body.append(row); }); table.append(head, body); region.append(table); ed.cuerpo.append(region);
  }
  p.append(ed.n);
  const elegida = ediciones.find(e => e.referencia === seleccion);
  if (elegida) {
    const detail = panel(d, t('detalle_edicion', { titulo: dato(elegida.titulo_clave) })); detail.n.id = 'formacion-detalle'; detail.n.tabIndex = -1; detail.n.dataset.foco = 'detalle';
    detail.cuerpo.append(datos(d, [[t('campos.necesidad'), dato(elegida.necesidad_clave)], [t('campos.modalidad'), dato(elegida.modalidad_clave)], [t('campos.prioridad'), dato(elegida.prioridad_clave)], [t('campos.presupuesto'), moneda(elegida.presupuesto_centimos)]]), lista(d, elegida.pendientes, t, dato));
    const bloqueo = nodo(d, 'p', t('bloqueo_inscripcion')); bloqueo.id = 'formacion-bloqueo'; const botones = nodo(d, 'div', undefined, 'acciones-vista');
    for (const accion of ['presentar', 'certificado']) { const b = boton(d, t(accion), () => {}, accion, true); b.setAttribute('aria-describedby', bloqueo.id); botones.append(b); }
    detail.cuerpo.append(botones, bloqueo, boton(d, t('cerrar_detalle'), acciones.cerrar, 'cerrar-detalle')); p.append(detail.n);
  }
  const dependencias = panel(d, t('siguiente')); dependencias.cuerpo.append(lista(d, escenario.pendientes, t, dato));
  const checklist = nodo(d, 'details'); checklist.append(nodo(d, 'summary', t('comprobaciones_ediciones')));
  for (const e of escenario.ediciones) {
    const claves = [...new Set(escenario.checklist.filter(item => item.referencia === e.referencia).map(item => item.clave))];
    if (claves.length) checklist.append(nodo(d, 'h3', dato(e.titulo_clave)), lista(d, claves, t, dato));
  }
  dependencias.cuerpo.append(checklist); p.append(dependencias.n);
  const tecnico = nodo(d, 'details'); tecnico.append(nodo(d, 'summary', t('detalle_tecnico')), datos(d, [[t('campos.fuente'), escenario.fuente.referencia], [t('campos.version_fuente'), escenario.fuente.version], [t('campos.version'), textos.numero(escenario.version)], [t('campos.referencia'), escenario.plan.referencia]])); p.append(tecnico);
  raiz.replaceChildren(p);
}
