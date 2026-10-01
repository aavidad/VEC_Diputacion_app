import { crearClienteSeleccion } from './cliente.js?v=20261001-seleccion-v1';
import { validarEjemplos, validarConfiguracion, validarResultado, aMicropuntos } from './configuracion.js?v=20261001-seleccion-v1';
import { crearEstadoEnsayo } from './estado.js?v=20261001-seleccion-v1';
import { nodo, panel, boton } from './dom.js?v=20261001-seleccion-v1';
import { pintarFormulario } from './formulario.js?v=20261001-seleccion-v1';
import { mostrarResultado } from './resultado.js?v=20261001-seleccion-v1';

export function montarSeleccion({ raiz, textos, cliente = crearClienteSeleccion() } = {}) {
  if (!raiz?.ownerDocument || !textos || !cliente?.listar || !cliente?.simular) throw new TypeError('seleccion.montaje');
  const d = raiz.ownerDocument; const t = textos.traducir;
  let activo = true, carga = null, turno = 0, ejemplos = [], ejemplo = null, configuracion = null;
  let errores = [], valores = {}, situacion = 'cargando', resultado = null;
  const estado = crearEstadoEnsayo({ cliente, validarResultado, publicar: cambio => {
    situacion = cambio.estado; resultado = cambio.resultado; if (activo) pintar();
  } });
  const entrada = () => ({ ...ejemplo, configuracion: structuredClone(configuracion) });
  function invalidar() { estado.invalidar(); errores = []; }
  function pintar() {
    if (!activo) return;
    const foco = d.activeElement?.id;
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
    const etiqueta = nodo(d, 'label', undefined, 'campo'); etiqueta.htmlFor = 'seleccion-ejemplo';
    etiqueta.append(nodo(d, 'span', t('ejemplo')));
    const selector = nodo(d, 'select'); selector.id = 'seleccion-ejemplo';
    for (const item of ejemplos) {
      const opcion = nodo(d, 'option', t(`ejemplo.${item.configuracion.modalidad}`));
      opcion.value = item.referencia; opcion.selected = item.referencia === ejemplo.referencia; selector.append(opcion);
    }
    selector.addEventListener('change', () => {
      const siguiente = ejemplos.find(item => item.referencia === selector.value);
      if (!siguiente) return;
      invalidar(); ejemplo = siguiente; configuracion = structuredClone(siguiente.configuracion); valores = {}; pintar();
      d.getElementById('seleccion-ejemplo')?.focus();
    });
    etiqueta.append(selector); selectorPanel.cuerpo.append(etiqueta);
    const contexto = nodo(d, 'dl', undefined, 'datos-resumen');
    for (const [clave, valor] of [['modalidad', t(`modalidad.${configuracion.modalidad}`)],
      ['turno', t(`turno.${configuracion.turno_acceso}`)], ['destino', t(`destino.${configuracion.destino}`)]]) {
      contexto.append(nodo(d, 'dt', t(clave)), nodo(d, 'dd', valor));
    }
    selectorPanel.cuerpo.append(contexto); raiz.append(selectorPanel.elemento);
    const formulario = nodo(d, 'div');
    const acciones = {
      plazas: (valor, id) => cambiar(id, valor, () => { configuracion.plazas = Number(valor); }),
      minimo: (indice, valor, id) => cambiar(id, valor, () => { configuracion.fases[indice].minimo_micropuntos = valor.trim() === '' ? null : aMicropuntos(valor); }),
      peso: (indice, valor, id) => cambiar(id, valor, () => { configuracion.fases[indice].peso = Number(valor); }),
      desempate: (ref, activo) => { invalidar(); configuracion.desempates = activo
        ? [...configuracion.desempates, ref] : configuracion.desempates.filter(v => v !== ref); pintar(); },
      restablecer: () => { invalidar(); configuracion = structuredClone(ejemplo.configuracion); valores = {}; pintar(); d.getElementById('seleccion-ejemplo')?.focus(); },
      simular: async () => {
        errores = validarConfiguracion(configuracion);
        if (errores.length) { pintar(); d.getElementById('seleccion-errores')?.focus(); return; }
        await estado.simular(entrada());
      },
    };
    pintarFormulario(formulario, configuracion, textos, acciones, errores, valores);
    raiz.append(formulario);
    const resultadoPanel = panel(d, t('resultado_titulo'));
    resultadoPanel.cuerpo.setAttribute('aria-live', 'polite');
    resultadoPanel.cuerpo.setAttribute('aria-busy', situacion === 'cargando' ? 'true' : 'false');
    const mensajeClave = ({ inicial: 'sin_resultado', cargando: 'calculando', error: 'error_simulacion',
      denegado: 'denegado', validacion_servidor: 'validacion_servidor', cancelado: 'sin_resultado' })[situacion];
    if (situacion === 'resultado' && resultado) mostrarResultado(resultadoPanel.cuerpo, resultado, configuracion, textos);
    else resultadoPanel.cuerpo.append(nodo(d, 'p', t(mensajeClave ?? 'sin_resultado')));
    resultadoPanel.cuerpo.append(nodo(d, 'p', t('limite'), 'texto-secundario'));
    raiz.append(resultadoPanel.elemento);
    if (foco && !errores.length) d.getElementById(foco)?.focus({ preventScroll: true });
  }
  function cambiar(id, valor, aplicar) {
    aplicar(); valores[id] = valor; invalidar(); errores = validarConfiguracion(configuracion);
    pintar(); d.getElementById(id)?.focus({ preventScroll: true });
  }
  async function cargar() {
    carga?.abort(); const actual = ++turno; carga = new AbortController(); situacion = 'cargando'; pintar();
    try {
      const recibidos = validarEjemplos(await cliente.listar({ signal: carga.signal }));
      if (!activo || actual !== turno) return;
      ejemplos = recibidos; ejemplo = ejemplos[0] ?? null;
      configuracion = ejemplo ? structuredClone(ejemplo.configuracion) : null;
      situacion = ejemplo ? 'inicial' : 'vacio'; pintar();
    } catch (error) {
      if (!activo || actual !== turno || carga.signal.aborted) return;
      situacion = error?.codigo === 'denegado' ? 'denegado' : 'error'; pintar();
    }
  }
  function desmontar() { activo = false; turno += 1; carga?.abort(); estado.cerrar(); raiz.replaceChildren(); }
  void cargar();
  return { desmontar, recargar: cargar, obtenerEstado: () => ({ situacion, resultado, configuracion: structuredClone(configuracion) }) };
}
