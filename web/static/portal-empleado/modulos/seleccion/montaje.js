import { crearClienteSeleccion } from './cliente.js?v=20261004-codexa-s6-notas-v1';
import { validarEjemplos, validarConfiguracion, validarResultado, validarNotasPropuestas, aMicropuntos } from './configuracion.js?v=20261004-codexa-s6-notas-v1';
import { crearEstadoEnsayo } from './estado.js?v=20261004-codexa-s6-notas-v1';
import { nodo, panel, boton } from './dom.js?v=20261004-codexa-s6-notas-v1';
import { pintarFormulario } from './formulario.js?v=20261004-codexa-s6-notas-v1';
import { mostrarResultado } from './resultado.js?v=20261004-codexa-s6-notas-v1';

export function montarSeleccion({ raiz, textos, cliente = crearClienteSeleccion() } = {}) {
  if (!raiz?.ownerDocument || !textos || !cliente?.listar || !cliente?.simular) throw new TypeError('seleccion.montaje');
  const d = raiz.ownerDocument; const t = textos.traducir;
  let activo = true, carga = null, turno = 0, ejemplos = [], ejemplo = null, configuracion = null;
  let notas = [], errores = [], valores = {}, situacion = 'cargando', resultado = null;
  const estado = crearEstadoEnsayo({ cliente, validarResultado, publicar: cambio => {
    situacion = cambio.estado; resultado = cambio.resultado; if (activo) pintar();
  } });
  const notasCambiadas = () => notas.filter((nota, i) => nota.puntos_micropuntos !== ejemplo.notas_prueba[i].puntos_micropuntos)
    .map(({ solicitud_ref, fase_ref, puntos_micropuntos }) => ({ solicitud_ref, fase_ref, puntos_micropuntos }));
  const validar = () => [...validarConfiguracion(configuracion), ...validarNotasPropuestas(ejemplo, configuracion, notas)];
  const entrada = () => ({ ...ejemplo, ejemplo_ref: ejemplo.referencia, configuracion: structuredClone(configuracion), notas_prueba: notasCambiadas() });
  function restablecerDatos() { configuracion = structuredClone(ejemplo.configuracion); notas = structuredClone(ejemplo.notas_prueba ?? []); valores = {}; }
  function invalidar() { estado.invalidar(); errores = []; }
  const paneles = ['seleccion-panel-ejemplo', 'seleccion-panel-configuracion', 'seleccion-panel-resultado'];
  function enfocarVisible(control) {
    if (!control) return;
    control.focus({ preventScroll: true });
    const marco = control.closest?.('.panel');
    if (!marco || marco.scrollHeight <= marco.clientHeight) {
      const pagina = d.scrollingElement; const altura = d.defaultView?.innerHeight;
      if (!pagina || !altura) return;
      const posicion = control.getBoundingClientRect();
      const arriba = Math.max(0, d.querySelector?.('.cabecera-portal')?.getBoundingClientRect().bottom ?? 0) + 4;
      const abajo = altura - 4;
      if (posicion.top < arriba) pagina.scrollTop += posicion.top - arriba;
      else if (posicion.bottom > abajo) pagina.scrollTop += posicion.bottom - abajo;
      return;
    }
    const limite = marco.getBoundingClientRect();
    const posicion = control.getBoundingClientRect();
    const cabecera = marco.children[0]?.clientHeight ?? 0;
    const arriba = limite.top + marco.clientTop + cabecera + 4;
    const abajo = limite.top + marco.clientTop + marco.clientHeight - 4;
    if (posicion.top < arriba) marco.scrollTop += posicion.top - arriba;
    else if (posicion.bottom > abajo) marco.scrollTop += posicion.bottom - abajo;
  }
  function pintar() {
    if (!activo) return;
    const foco = d.activeElement?.id;
    const pagina = d.scrollingElement;
    const posicionPagina = pagina ? { arriba: pagina.scrollTop, izquierda: pagina.scrollLeft } : null;
    const seleccion = d.activeElement?.type === 'text' ? [d.activeElement.selectionStart, d.activeElement.selectionEnd] : null;
    const desplazamientos = paneles.map(id => {
      const marco = d.getElementById(id);
      return { id, arriba: marco?.scrollTop ?? 0, izquierda: marco?.scrollLeft ?? 0 };
    });
    raiz.replaceChildren();
    if (!ejemplos.length) {
      const { elemento, cuerpo } = panel(d, t('titulo'));
      const clave = situacion === 'cargando' ? 'cargando' : situacion === 'vacio' ? 'sin_ejemplos'
        : situacion === 'denegado' ? 'denegado' : 'error_carga';
      const mensaje = nodo(d, 'p', t(clave)); mensaje.setAttribute('role', situacion === 'error' ? 'alert' : 'status');
      cuerpo.append(mensaje);
      if (situacion === 'error') { const reintentar = boton(d, t('reintentar')); reintentar.addEventListener('click', cargar); cuerpo.append(reintentar); }
      raiz.append(elemento); return;
    }
    const selectorPanel = panel(d, t('ejemplo_titulo'));
    selectorPanel.elemento.id = 'seleccion-panel-ejemplo';
    selectorPanel.cuerpo.append(nodo(d, 'p', t('limite'), 'texto-secundario'));
    const etiqueta = nodo(d, 'label', undefined, 'campo'); etiqueta.htmlFor = 'seleccion-ejemplo';
    etiqueta.append(nodo(d, 'span', t('ejemplo_etiqueta')));
    const selector = nodo(d, 'select'); selector.id = 'seleccion-ejemplo';
    for (const item of ejemplos) {
      const opcion = nodo(d, 'option', t(`ejemplo.${item.configuracion.modalidad}`));
      opcion.value = item.referencia; opcion.selected = item.referencia === ejemplo.referencia; selector.append(opcion);
    }
    selector.addEventListener('change', () => {
      const siguiente = ejemplos.find(item => item.referencia === selector.value);
      if (!siguiente) return;
      invalidar(); ejemplo = siguiente; restablecerDatos(); pintar();
      d.getElementById('seleccion-ejemplo')?.focus();
    });
    etiqueta.append(selector); selectorPanel.cuerpo.append(etiqueta);
    const contexto = nodo(d, 'dl', undefined, 'datos-resumen');
    for (const [clave, valor] of [['modalidad', t(`modalidad.${configuracion.modalidad}`)],
      ['turno', t(`turno.${configuracion.turno_acceso}`)], ['destino', t(`destino.${configuracion.destino}`)]]) {
      contexto.append(nodo(d, 'dt', t(`${clave}_etiqueta`)), nodo(d, 'dd', valor));
    }
    selectorPanel.cuerpo.append(contexto); raiz.append(selectorPanel.elemento);
    const formulario = nodo(d, 'div');
    const acciones = {
      plazas: (valor, id) => cambiar(id, valor, () => { configuracion.plazas = Number(valor); }),
      minimo: (indice, valor, id) => cambiar(id, valor, () => { configuracion.fases[indice].minimo_micropuntos = valor.trim() === '' ? null : aMicropuntos(valor); }),
      peso: (indice, valor, id) => cambiar(id, valor, () => { configuracion.fases[indice].peso = Number(valor); }),
      desempate: (ref, activo) => { invalidar(); configuracion.desempates = activo
        ? configuracion.fases.map(f => f.referencia).filter(r => r === ref || configuracion.desempates.includes(r)) : configuracion.desempates.filter(v => v !== ref); pintar(); },
      nota: (indice, valor, id) => cambiar(id, valor, () => { notas[indice].puntos_micropuntos = valor.trim() === '' ? null : aMicropuntos(valor); }),
      restablecer: () => { invalidar(); restablecerDatos(); pintar(); d.getElementById('seleccion-ejemplo')?.focus(); },
      simular: async () => {
        errores = validar();
        if (errores.length) { pintar(); d.getElementById('seleccion-errores')?.focus(); return; }
        await estado.simular(entrada());
      },
    };
    pintarFormulario(formulario, configuracion, textos, acciones, errores, valores, situacion === 'cargando', notas);
    formulario.firstChild.id = 'seleccion-panel-configuracion';
    raiz.append(formulario);
    const resultadoPanel = panel(d, t('resultado_titulo'));
    resultadoPanel.elemento.id = 'seleccion-panel-resultado';
    resultadoPanel.cuerpo.setAttribute('aria-live', 'polite');
    resultadoPanel.cuerpo.setAttribute('aria-busy', situacion === 'cargando' ? 'true' : 'false');
    const mensajeClave = ({ inicial: 'sin_resultado', cargando: 'calculando', error: 'error_simulacion',
      denegado: 'denegado', validacion_servidor: 'validacion_servidor', cancelado: 'sin_resultado' })[situacion];
    if (situacion === 'resultado' && resultado) mostrarResultado(resultadoPanel.cuerpo, resultado, configuracion, textos);
    else resultadoPanel.cuerpo.append(nodo(d, 'p', t(mensajeClave ?? 'sin_resultado')));
    raiz.append(resultadoPanel.elemento);
    for (const { id, arriba, izquierda } of desplazamientos) {
      const marco = d.getElementById(id);
      if (marco) { marco.scrollTop = arriba; marco.scrollLeft = izquierda; }
    }
    if (pagina && posicionPagina) {
      pagina.scrollTop = posicionPagina.arriba; pagina.scrollLeft = posicionPagina.izquierda;
    }
    if (foco) {
      const control = d.getElementById(foco); enfocarVisible(control);
      if (seleccion) control?.setSelectionRange?.(...seleccion);
    }
  }
  function cambiar(id, valor, aplicar) {
    aplicar(); valores[id] = valor; invalidar(); errores = validar();
    pintar(); enfocarVisible(d.getElementById(id));
  }
  async function cargar() {
    carga?.abort(); const actual = ++turno; carga = new AbortController(); situacion = 'cargando'; pintar();
    try {
      const recibidos = validarEjemplos(await cliente.listar({ signal: carga.signal }));
      if (!activo || actual !== turno) return;
      ejemplos = recibidos; ejemplo = ejemplos[0] ?? null;
      if (ejemplo) restablecerDatos(); else { configuracion = null; notas = []; }
      situacion = ejemplo ? 'inicial' : 'vacio'; pintar();
    } catch (error) {
      if (!activo || actual !== turno || carga.signal.aborted) return;
      situacion = error?.codigo === 'denegado' ? 'denegado' : 'error'; pintar();
    }
  }
  function desmontar() { activo = false; turno += 1; carga?.abort(); estado.cerrar(); raiz.replaceChildren(); }
  void cargar();
  return { desmontar, recargar: cargar, obtenerEstado: () => ({ situacion, resultado, configuracion: structuredClone(configuracion), notas_prueba: structuredClone(notas) }) };
}
