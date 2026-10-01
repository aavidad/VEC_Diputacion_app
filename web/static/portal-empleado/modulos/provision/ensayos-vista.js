import { formatearPuntos } from './modelo.js?v=20261001-provision-ciclo-v5';
function nodo(d, tag, texto, clase) { const n = d.createElement(tag); if (texto !== undefined) n.textContent = texto; if (clase) n.className = clase; return n; }
function boton(d, texto, accion, clave, deshabilitado = false) { const b = nodo(d, 'button', texto, 'boton-secundario'); b.type = 'button'; b.dataset.foco = clave; b.disabled = deshabilitado; b.addEventListener('click', accion); return b; }
function tabla(d, titulo, cabeceras, filas) { const box = nodo(d, 'div', undefined, 'tabla-contenedor'); box.tabIndex = 0; box.setAttribute('role', 'region'); box.setAttribute('aria-label', titulo); const table = nodo(d, 'table', undefined, 'tabla-datos'); table.append(nodo(d, 'caption', titulo)); const head = nodo(d, 'thead'); const tr = nodo(d, 'tr'); cabeceras.forEach(t => { const th = nodo(d, 'th', t); th.scope = 'col'; tr.append(th); }); head.append(tr); const body = nodo(d, 'tbody'); filas.forEach(f => { const row = nodo(d, 'tr'); f.forEach((texto, i) => { const td = nodo(d, i ? 'td' : 'th'); if (!i) td.scope = 'row'; if (texto?.nodeType) td.append(texto); else td.textContent = texto; row.append(td); }); body.append(row); }); table.append(head, body); box.append(table); return box; }
function textoConCortes(d, tag, valor) { const n = nodo(d, tag); String(valor ?? '').split(/([a-f0-9]{64})/g).forEach(parte => { if (/^[a-f0-9]{64}$/.test(parte)) parte.match(/.{1,8}/g).forEach(fragmento => n.append(nodo(d, 'span', fragmento), nodo(d, 'wbr'))); else n.append(nodo(d, 'span', parte)); }); return n; }
function referencia(d, t, titulo, datos) { const details = nodo(d, 'details', undefined, 'campo'); details.append(nodo(d, 'summary', titulo)); const dl = nodo(d, 'dl', undefined, 'resumen-expediente'); datos.forEach(([clave, valor]) => { dl.append(nodo(d, 'dt', t(`ensayos.${clave}`)), textoConCortes(d, 'dd', valor)); }); details.append(dl); return details; }
export function pintarAdjudicacion({ raiz, estado, textos, acciones }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const pila = nodo(d, 'div', undefined, 'pila');
  const claveRef = ref => typeof ref === 'string' ? ref.replaceAll(':', '_') : '';
  const nombrePersona = ref => Object.hasOwn(textos.mensajes.ensayos.personas, claveRef(ref)) ? t(`ensayos.personas.${claveRef(ref)}`) : t('ensayos.persona_sin_nombre');
  const nombrePuesto = ref => Object.hasOwn(textos.mensajes.ensayos.puestos, claveRef(ref)) ? t(`ensayos.puestos.${claveRef(ref)}`) : t('ensayos.puesto_sin_nombre');
  const puestoVacante = ref => nombrePuesto(estado.resumen?.vacantes.find(v => v.vacante_ref === ref)?.puesto_ref);
  pila.append(nodo(d, 'p', t('ensayos.limite')));
  const aviso = nodo(d, 'p', t(`ensayos.estados.${estado.estado}`)); aviso.dataset.ensayoAviso = ''; aviso.setAttribute('role', ['error', 'validacion', 'conflicto', 'denegado'].includes(estado.estado) ? 'alert' : 'status'); aviso.setAttribute('aria-live', 'polite'); pila.append(aviso);
  if (!estado.configuracion) { if (estado.estado === 'error') pila.append(boton(d, t('reintentar_preparacion'), acciones.cargar, 'ensayo-recargar')); raiz.replaceChildren(pila); return; }
  const selectorLabel = nodo(d, 'label', undefined, 'campo'); selectorLabel.append(nodo(d, 'span', t('ejemplo_selector'))); const selector = nodo(d, 'select'); selector.dataset.foco = 'adjudicacion-ejemplo'; (estado.ejemplos ?? []).forEach((ref, i) => { const opt = nodo(d, 'option', t('ensayos.ejemplo_numero', { numero: textos.numero(i + 1) })); opt.value = String(i); opt.selected = estado.indice === i; selector.append(opt); }); selector.addEventListener('change', () => acciones.elegir(Number(selector.value))); selectorLabel.append(selector); pila.append(selectorLabel, boton(d, t('ensayos.restablecer'), () => acciones.elegir(estado.indice ?? 0), 'adjudicacion-restablecer'));
  const c = estado.configuracion; const config = nodo(d, 'div', undefined, 'rejilla-dos');
  ['politica_ref', 'politica_version', 'bases_ref', 'version'].forEach(campo => {
    const label = nodo(d, 'label', undefined, 'campo'); label.append(nodo(d, 'span', t(`ensayos.${campo}`))); const input = nodo(d, 'input'); input.type = 'text'; input.maxLength = 160; input.value = estado.invalidos?.[campo] ?? c[campo]; input.dataset.foco = `adjudicacion-${campo}`;
    const error = nodo(d, 'p', Object.hasOwn(estado.invalidos ?? {}, campo) ? t('ensayos.configuracion_invalida') : '', 'texto-secundario'); error.id = `adjudicacion-error-${campo}`; error.hidden = !Object.hasOwn(estado.invalidos ?? {}, campo); input.setAttribute('aria-invalid', String(!error.hidden)); input.setAttribute('aria-describedby', error.id);
    input.addEventListener('change', () => { const cambio = acciones.cambiar({ campo, valor: input.value }); raiz.querySelector('[data-ensayo-resultado]')?.replaceChildren(); error.textContent = cambio.valido ? '' : t('ensayos.configuracion_invalida'); error.hidden = cambio.valido; input.setAttribute('aria-invalid', String(!cambio.valido)); aviso.textContent = t(`ensayos.estados.${cambio.estado.estado}`); aviso.setAttribute('role', cambio.estado.estado === 'validacion' ? 'alert' : 'status'); const simular = Array.from(raiz.querySelectorAll('[data-foco]')).find(n => n.dataset.foco === 'adjudicacion-simular'); if (simular) simular.disabled = Object.keys(cambio.estado.invalidos).length > 0; });
    label.append(input, error); config.append(label);
  }); pila.append(config);
  const politica = nodo(d, 'dl', undefined, 'resumen-expediente'); ['metodo', 'prioridad', 'incompatibilidad'].forEach(campo => { politica.append(nodo(d, 'dt', t(`ensayos.${campo}`)), nodo(d, 'dd', Object.hasOwn(textos.mensajes.ensayos.opciones, c[campo]) ? t(`ensayos.opciones.${c[campo]}`) : t('ensayos.estados.no_soportado'))); }); pila.append(politica);
  pila.append(nodo(d, 'h3', t('ensayos.desempates')));
  if (!c.desempates.length) pila.append(nodo(d, 'p', t('ensayos.sin_desempates')));
  const filas = c.desempates.map((regla, i) => {
    const sentidoLabel = nodo(d, 'label', undefined, 'campo'); sentidoLabel.append(nodo(d, 'span', t('ensayos.sentido'))); const sentido = nodo(d, 'select'); sentido.dataset.foco = `desempate-sentido-${i}`;
    ['mayor', 'menor'].forEach(v => { const opt = nodo(d, 'option', t(`ensayos.opciones.${v}`)); opt.value = v; opt.selected = regla.sentido === v; sentido.append(opt); }); sentido.addEventListener('change', () => { acciones.cambiar({ indice: i, sentido: sentido.value }); acciones.repintar(`desempate-sentido-${i}`); }); sentidoLabel.append(sentido);
    const controles = nodo(d, 'div', undefined, 'acciones-vista'); ['arriba', 'abajo', 'quitar'].forEach(accion => { const b = boton(d, t(`acciones.${accion}`), () => { acciones.cambiar({ indice: i, accion }); acciones.repintar('adjudicacion-simular'); }, `desempate-${accion}-${i}`, accion === 'arriba' && i === 0 || accion === 'abajo' && i === c.desempates.length - 1); b.setAttribute('aria-label', t('ensayos.mover_desempate', { accion: t(`acciones.${accion}`), orden: textos.numero(i + 1) })); controles.append(b); });
    return [textos.numero(i + 1), Object.hasOwn(textos.mensajes.ensayos.reglas, claveRef(regla.regla_id)) ? t(`ensayos.reglas.${claveRef(regla.regla_id)}`) : t('ensayos.regla_sin_nombre'), sentidoLabel, controles];
  });
  pila.append(tabla(d, t('ensayos.desempates'), [t('campos.preferencia'), t('campos.regla'), t('ensayos.sentido'), t('campos.acciones')], filas));
  const resumen = estado.resumen.solicitudes.flatMap(s => s.preferencias.map(p => [nombrePersona(s.persona_ref), puestoVacante(p.vacante_ref), textos.numero(p.orden), p.total === null ? t('estados.sin_dato') : formatearPuntos(p.total, textos.localizacion)]));
  pila.append(tabla(d, t('ensayos.preferencias'), [t('ensayos.persona'), t('campos.puesto'), t('campos.preferencia'), t('campos.puntos')], resumen));
  const simular = boton(d, t('ensayos.simular'), acciones.simular, 'adjudicacion-simular', Object.keys(estado.invalidos ?? {}).length > 0 || estado.estado === 'calculando'); simular.className = 'boton-primario'; pila.append(simular);
  const salida = nodo(d, 'div', undefined, 'pila'); salida.dataset.ensayoResultado = ''; pila.append(salida);
  if (estado.resultado) {
    const r = estado.resultado; salida.append(nodo(d, 'h3', t(`ensayos.estados.${r.estado}`)));
    if (r.asignaciones.length) salida.append(tabla(d, t('ensayos.asignaciones'), [t('ensayos.persona'), t('campos.puesto'), t('campos.preferencia')], r.asignaciones.map(a => [nombrePersona(a.persona_ref), nombrePuesto(a.puesto_ref), textos.numero(a.preferencia)])));
    else salida.append(nodo(d, 'p', t('ensayos.sin_asignaciones')));
    const incidencias = nodo(d, 'ul'); r.incidencias.forEach(i => incidencias.append(nodo(d, 'li', Object.hasOwn(textos.mensajes.ensayos.incidencias, i.codigo) ? t(`ensayos.incidencias.${i.codigo}`) : t('ensayos.incidencia_pendiente')))); salida.append(incidencias);
    if (r.personas_sin_asignacion?.length) salida.append(nodo(d, 'p', t('ensayos.personas_pendientes', { personas: r.personas_sin_asignacion.map(nombrePersona).join(', ') })));
    salida.append(referencia(d, t, t('detalle_tecnico'), [['politica_ref', r.politica_ref], ['politica_version', r.politica_version], ['huella_resultado', r.huella_resultado]]));
  }
  const oficial = boton(d, t('acciones.resolver'), () => {}, 'adjudicacion-resolver', true); oficial.setAttribute('aria-describedby', 'adjudicacion-bloqueo'); const bloqueo = nodo(d, 'p', t('ensayos.bloqueo')); bloqueo.id = 'adjudicacion-bloqueo'; pila.append(oficial, bloqueo); raiz.replaceChildren(pila);
}
export function pintarCiclo({ raiz, estado, textos, acciones }) {
  const d = raiz.ownerDocument; const t = textos.traducir; const pila = nodo(d, 'div', undefined, 'pila');
  pila.append(nodo(d, 'p', t('ciclo.limite')));
  const aviso = nodo(d, 'p', t(`ensayos.estados.${estado.estado}`)); aviso.setAttribute('role', ['error', 'validacion', 'conflicto', 'denegado'].includes(estado.estado) ? 'alert' : 'status'); aviso.setAttribute('aria-live', 'polite'); pila.append(aviso);
  if (!estado.casos) { if (estado.estado === 'error') pila.append(boton(d, t('reintentar_preparacion'), acciones.cargar, 'ciclo-recargar')); raiz.replaceChildren(pila); return; }
  const label = nodo(d, 'label', undefined, 'campo'); label.append(nodo(d, 'span', t('ciclo.elegir'))); const selector = nodo(d, 'select'); selector.dataset.foco = 'ciclo-caso';
  estado.casos.forEach(c => { let titulo; try { titulo = t(c.titulo_clave); } catch { titulo = t(`ciclo.casos.${Object.hasOwn(textos.mensajes.ciclo.casos, c.decision) ? c.decision : 'pendiente'}`); } const opt = nodo(d, 'option', titulo); opt.value = c.caso_ref; opt.selected = c.caso_ref === estado.caso_ref; selector.append(opt); });
  selector.addEventListener('change', () => acciones.cambiarCaso(selector.value)); label.append(selector); pila.append(label);
  const dl = nodo(d, 'dl', undefined, 'resumen-expediente'); dl.append(nodo(d, 'dt', t('campos.version')), nodo(d, 'dd', estado.configuracion.version), nodo(d, 'dt', t('ciclo.catalogo')), nodo(d, 'dd', estado.catalogo_causas.version)); pila.append(dl);
  const simular = boton(d, t('ciclo.simular'), acciones.simular, 'ciclo-simular', estado.estado === 'calculando'); simular.className = 'boton-primario'; pila.append(simular);
  if (estado.resultado) {
    const r = estado.resultado;
    const filas = r.valoraciones.map(v => { const decision = r.decisiones.find(q => q.referencia === v.decision_ref); return [textos.numero(v.version), decision ? t(`ciclo.decisiones.${decision.tipo}`) : t('ciclo.inicial'), v.resultado.total === null ? t('estados.sin_dato') : formatearPuntos(v.resultado.total, textos.localizacion), t(`estados.${v.resultado.completo ? 'completo' : 'pendiente'}`)]; });
    pila.append(tabla(d, t('ciclo.cronologia'), [t('campos.version'), t('ciclo.revision'), t('campos.puntos'), t('campos.estado')], filas));
    pila.append(nodo(d, 'h3', t('ciclo.reclamaciones')));
    r.reclamaciones.forEach(q => {
      const causa = Object.hasOwn(textos.mensajes.ciclo.causas, q.causa_codigo) ? t(`ciclo.causas.${q.causa_codigo}`) : t('ciclo.causa_pendiente');
      pila.append(nodo(d, 'p', t('ciclo.reclamacion', { version: textos.numero(q.version_valoracion), causa }))); const decision = r.decisiones.find(e => e.reclamacion_ref === q.referencia);
      pila.append(nodo(d, 'p', decision ? t(`ciclo.decisiones.${decision.tipo}`) : t('ciclo.pendiente_revision')));
      const detalles = nodo(d, 'details', undefined, 'campo'); detalles.append(nodo(d, 'summary', t('detalle_tecnico'))); const refs = nodo(d, 'dl', undefined, 'resumen-expediente');
      [[t('ciclo.evidencia'), q.evidencia_ref], [t('ciclo.motivacion'), decision?.motivacion_ref ?? t('estados.sin_dato')], [t('ciclo.referencia_reclamacion'), q.referencia]].forEach(([titulo, valor]) => refs.append(nodo(d, 'dt', titulo), textoConCortes(d, 'dd', valor))); detalles.append(refs); pila.append(detalles);
    });
    const ultimo = r.valoraciones[r.valoraciones.length - 1]; const detalle = nodo(d, 'details', undefined, 'campo'); detalle.append(nodo(d, 'summary', t('detalle')));
    detalle.append(tabla(d, t('detalle'), [t('campos.regla'), t('campos.estado'), t('campos.puntos')], ultimo.resultado.desglose.map(g => [t(`familias.${g.familia}`), t(`estados.${g.estado === 'calculado' ? 'completo' : 'pendiente'}`), g.estado === 'pendiente_dato' ? t('estados.sin_dato') : formatearPuntos(g.resultado, textos.localizacion)]))); pila.append(detalle);
    pila.append(nodo(d, 'h3', t('ciclo.borrador')), nodo(d, 'p', t('ciclo.sin_efectos')), nodo(d, 'p', t('ciclo.version_resolucion', { version: textos.numero(r.resolucion.version_valoracion) })));
    const pendientes = nodo(d, 'ul'); r.resolucion.pendientes.forEach(codigo => pendientes.append(nodo(d, 'li', Object.hasOwn(textos.mensajes.ciclo.pendientes, codigo) ? t(`ciclo.pendientes.${codigo}`) : t('ciclo.dependencia_pendiente')))); pila.append(pendientes);
    const refs = nodo(d, 'details', undefined, 'campo'); refs.append(nodo(d, 'summary', t('ciclo.huella_versiones'))); r.valoraciones.forEach(v => refs.append(textoConCortes(d, 'p', t('ciclo.version_huella', { version: textos.numero(v.version), huella: v.huella_revision })))); pila.append(refs);
  }
  const presentar = boton(d, t('acciones.alegar'), () => {}, 'ciclo-alegar', true); presentar.setAttribute('aria-describedby', 'ciclo-bloqueo'); const firmar = boton(d, t('acciones.resolver'), () => {}, 'ciclo-resolver', true); firmar.setAttribute('aria-describedby', 'ciclo-bloqueo');
  const bloqueo = nodo(d, 'p', t('ciclo.bloqueo')); bloqueo.id = 'ciclo-bloqueo'; pila.append(presentar, firmar, bloqueo); raiz.replaceChildren(pila);
}
