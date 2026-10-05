import { nodo, panel, boton } from './dom.js?v=20261004-codexa-s6-notas-v1';
import { decimalPuntos } from './configuracion.js?v=20261004-codexa-s6-notas-v1';

function campo(documento, id, texto, valor, tipo = 'text') {
  const etiqueta = nodo(documento, 'label', undefined, 'campo');
  etiqueta.htmlFor = id;
  const nombre = nodo(documento, 'span', texto);
  const entrada = nodo(documento, 'input');
  entrada.id = id; entrada.name = id; entrada.type = tipo; entrada.value = String(valor);
  if (tipo === 'number') { entrada.min = '1'; entrada.step = '1'; entrada.required = true; entrada.max = id === 'seleccion-plazas' ? '1000' : '100'; }
  else entrada.inputMode = 'decimal';
  entrada.setAttribute('aria-describedby', `${id}-error`);
  const error = nodo(documento, 'span', undefined, 'error-campo');
  error.id = `${id}-error`; error.hidden = true;
  etiqueta.append(nombre, entrada, error);
  return { etiqueta, entrada, error };
}

/** Controles declarativos; cada edición deja de mostrar el cálculo anterior. */
export function pintarFormulario(raiz, configuracion, textos, acciones, errores = [], valores = {}, cargando = false, notas = []) {
  const d = raiz.ownerDocument; const t = textos.traducir;
  const { elemento, cuerpo } = panel(d, t('configuracion_titulo'));
  const contexto = nodo(d, 'p', t('configuracion_contexto'));
  cuerpo.append(contexto);
  const form = nodo(d, 'form', undefined, 'pila'); form.noValidate = true;
  const resumen = nodo(d, 'div', undefined, 'aviso-error'); resumen.id = 'seleccion-errores';
  resumen.tabIndex = -1; resumen.hidden = !errores.length;
  if (errores.length) {
    resumen.append(nodo(d, 'p', t('validacion_titulo')));
    const lista = nodo(d, 'ul');
    for (const error of errores) {
      const li = nodo(d, 'li'); const enlace = nodo(d, 'a', t(error.clave));
      enlace.href = `#${error.campo}`;
      enlace.addEventListener('click', evento => { evento.preventDefault(); d.getElementById(error.campo)?.focus(); });
      li.append(enlace); lista.append(li);
    }
    resumen.append(lista);
  }
  form.append(resumen);
  const controles = [];
  const agregar = (id, titulo, valor, tipo, actualizar) => {
    const c = campo(d, id, titulo, valores[id] ?? valor, tipo);
    const error = errores.find(e => e.campo === id);
    if (error) { c.entrada.setAttribute('aria-invalid', 'true'); c.error.textContent = t(error.clave); c.error.hidden = false; }
    c.entrada.addEventListener('input', () => actualizar(c.entrada.value, id));
    controles.push(c.entrada); return c.etiqueta;
  };
  form.append(agregar('seleccion-plazas', t('plazas'), configuracion.plazas, 'number', acciones.plazas));
  const faseLista = nodo(d, 'div', undefined, 'pila');
  configuracion.fases.forEach((fase, indice) => {
    const bloque = nodo(d, 'fieldset', undefined, 'grupo-campos');
    bloque.append(nodo(d, 'legend', t(`fase.${fase.tipo}`, { numero: textos.numero(indice + 1) })));
    const rejilla = nodo(d, 'div', undefined, 'rejilla-campos');
    rejilla.append(
      agregar(`seleccion-minimo-${indice}`, t('minimo'), fase.minimo_micropuntos === null ? '' : decimalPuntos(fase.minimo_micropuntos), 'text', (valor, id) => acciones.minimo(indice, valor, id)),
      agregar(`seleccion-peso-${indice}`, t('peso'), fase.peso, 'number', (valor, id) => acciones.peso(indice, valor, id)));
    const dato = nodo(d, 'p', t('maximo_fase', { puntos: textos.numero(fase.maximo_micropuntos / 1000000, { maximumFractionDigits: 6 }) }), 'texto-secundario');
    bloque.append(rejilla, dato); faseLista.append(bloque);
  });
  form.append(faseLista);
  if (notas.length) {
    const bloqueNotas = nodo(d, 'fieldset', undefined, 'grupo-campos');
    bloqueNotas.append(nodo(d, 'legend', t('notas_titulo')));
    const ayudaNotas = nodo(d, 'p', t('notas_pendientes'), 'texto-secundario');
    ayudaNotas.id = 'seleccion-notas-ayuda'; bloqueNotas.append(ayudaNotas);
    const rejillaNotas = nodo(d, 'div', undefined, 'rejilla-campos');
    notas.forEach((nota, indice) => {
      const faseIndice = configuracion.fases.findIndex(f => f.referencia === nota.fase_ref);
      const fase = configuracion.fases[faseIndice];
      const titulo = t('nota_etiqueta', { nombre: nota.nombre, fase: t('fase.prueba', { numero: textos.numero(faseIndice + 1) }) });
      const etiqueta = agregar(`seleccion-nota-${indice}`, titulo,
        nota.puntos_micropuntos === null ? '' : decimalPuntos(nota.puntos_micropuntos), 'text',
        (valor, id) => acciones.nota(indice, valor, id));
      const pista = nodo(d, 'span', t('maximo_fase', { puntos: textos.numero(fase.maximo_micropuntos / 1000000, { maximumFractionDigits: 6 }) }), 'texto-secundario');
      pista.id = `seleccion-nota-${indice}-maximo`; etiqueta.append(pista);
      controles.at(-1).setAttribute('aria-describedby', `seleccion-notas-ayuda ${pista.id} seleccion-nota-${indice}-error`);
      rejillaNotas.append(etiqueta);
    });
    bloqueNotas.append(rejillaNotas); form.append(bloqueNotas);
  }
  const desempate = nodo(d, 'fieldset', undefined, 'grupo-campos');
  desempate.append(nodo(d, 'legend', t('desempate')), nodo(d, 'p', t('desempate_ayuda'), 'texto-secundario'));
  configuracion.fases.forEach((fase, indice) => {
    const id = `seleccion-desempate-${indice}`;
    const etiqueta = nodo(d, 'label', undefined, 'opcion-casilla');
    const casilla = nodo(d, 'input'); casilla.type = 'checkbox'; casilla.id = id;
    casilla.checked = configuracion.desempates.includes(fase.referencia);
    casilla.addEventListener('change', () => acciones.desempate(fase.referencia, casilla.checked));
    etiqueta.append(casilla, nodo(d, 'span', t(`fase.${fase.tipo}`, { numero: textos.numero(indice + 1) })));
    desempate.append(etiqueta);
  });
  form.append(desempate);
  const accionesFila = nodo(d, 'div', undefined, 'acciones-formulario');
  const restablecer = boton(d, t('restablecer')); restablecer.addEventListener('click', acciones.restablecer);
  const simular = boton(d, t('simular'), 'boton-primario'); simular.type = 'submit'; simular.disabled = cargando;
  accionesFila.append(restablecer, simular); form.append(accionesFila);
  form.addEventListener('submit', evento => { evento.preventDefault(); acciones.simular(); });
  cuerpo.append(form); raiz.replaceChildren(elemento);
  return { controles, resumen };
}
