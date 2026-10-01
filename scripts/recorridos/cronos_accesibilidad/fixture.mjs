// Se ejecuta en Chrome sobre módulos y catálogos del checkout, sin cliente HTTP.
export async function montarFixture({ pantalla, casos }) {
  const raiz = document.querySelector('#lectura');
  const base = '/portal-empleado/modulos/cronos/';
  const t = casos.textos[document.documentElement.lang];
  window.__fixture = { escrituras: 0 };
  const bloquear = async () => { window.__fixture.escrituras++; throw new Error('escritura_offline'); };
  const copia = valor => structuredClone(valor);
  if (pantalla === 'fichaje') {
    const { montarVistaRemotoCronos } = await import(`${base}vista-remoto.js`);
    window.__vista = montarVistaRemotoCronos({ raiz, cliente: { disponibilidad: async () => copia(casos.fichaje), registrar: bloquear } });
  } else if (pantalla === 'saldo') {
    const { montarVistaSaldoCronos } = await import(`${base}vista-saldo-conectado.js`);
    window.__vista = montarVistaSaldoCronos({ raiz, cliente: { consultar: async consulta => {
      const datos = copia(casos.saldo);
      datos.periodo.tipo = consulta.periodo;
      if (consulta.periodo === 'rango') {
        datos.periodo.desde = consulta.desde; datos.periodo.hasta = consulta.hasta;
        datos.detalle = datos.detalle.filter(d => d.fecha >= consulta.desde && d.fecha <= consulta.hasta);
      }
      return datos;
    } } });
  } else if (pantalla === 'calendario_incidencias') {
    const { montarMovimientosPropiosCronos } = await import(`${base}vista-movimientos-propios.js`);
    const datos = copia(casos.movimientos);
    datos.calendario.dias[0].nombre = t.festivo; datos.absentismos[0].nombre = t.traslado;
    window.__vista = montarMovimientosPropiosCronos({ raiz, anio: Number(datos.periodo.desde.slice(0, 4)), fechaSeleccionada: datos.absentismos[0].desde,
      cliente: { consultarMovimientos: async consulta => ({ ...copia(datos), periodo: { ...datos.periodo, tipo: consulta.periodo } }), solicitarCorreccion: bloquear } });
  } else if (pantalla === 'bandeja') {
    const { montarBandejaPermisosCronos } = await import(`${base}vista-bandeja-permisos.js`);
    const datos = copia(casos.bandeja);
    datos.pendientes[0].nombre = t.vacaciones; datos.pendientes[0].empleado_etiqueta = t.persona;
    window.__vista = montarBandejaPermisosCronos({ raiz,
      cliente: { consultarBandeja: async ({ paso }) => ({ ...copia(datos), paso }), resolver: bloquear } });
  } else throw new Error('pantalla_desconocida');
}
