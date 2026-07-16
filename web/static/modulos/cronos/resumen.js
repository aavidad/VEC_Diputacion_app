// Resumen operativo de Cronos. No conoce el estado global del portal ni sus
// transportes: recibe una vista ya autorizada y una raiz DOM intercambiable.
export function renderizarResumenCronos(vista, documento = document) {
  const destino = documento.querySelector("#cronos-panel");
  if (!destino) return;

  const espacio = vista?.workspace || {};
  const resumen = espacio.cronos_daily_summary || {};
  const permisos = Array.isArray(espacio.cronos_permission_balances) ? espacio.cronos_permission_balances : [];
  const secciones = Array.isArray(espacio.cronos_sections) ? espacio.cronos_sections : [];
  const visible = (valor) => valor === null || valor === undefined || valor === "" ? "-" : String(valor);

  const navegacion = documento.createElement("ul");
  navegacion.className = "cronos-nav";
  navegacion.setAttribute("aria-label", "Secciones de Cronos");
  const seccionesVisibles = secciones.length ? secciones : ["Sin secciones disponibles"];
  navegacion.replaceChildren(...seccionesVisibles.map((seccion) => {
    const etiqueta = documento.createElement("li");
    etiqueta.textContent = seccion;
    return etiqueta;
  }));

  const cajaResumen = documento.createElement("div");
  cajaResumen.className = "cronos-summary";
  const indicadores = [
    ["Horas teóricas", visible(resumen.theoretical)],
    ["Horas trabajadas", visible(resumen.worked)],
    ["Teletrabajo", visible(resumen.telework)],
    ["Exceso / defecto mes", visible(resumen.period_balance ?? resumen.daily_balance)],
  ];
  cajaResumen.replaceChildren(...indicadores.map(([nombre, valor]) => {
    const indicador = documento.createElement("div");
    const etiqueta = documento.createElement("span");
    const dato = documento.createElement("strong");
    etiqueta.textContent = nombre;
    dato.textContent = valor;
    if (String(valor).startsWith("-")) dato.classList.add("negative");
    indicador.replaceChildren(etiqueta, dato);
    return indicador;
  }));

  const tabla = documento.createElement("table");
  tabla.className = "mini-table";
  tabla.setAttribute("aria-label", "Saldos de permisos disponibles");
  const cabecera = documento.createElement("thead");
  const filaCabecera = documento.createElement("tr");
  filaCabecera.replaceChildren(...["Permiso", "Disponible", "Máximo", "Solicitado", "Restante"].map((texto) => {
    const celda = documento.createElement("th");
    celda.setAttribute("scope", "col");
    celda.textContent = texto;
    return celda;
  }));
  cabecera.replaceChildren(filaCabecera);
  const cuerpo = documento.createElement("tbody");
  const filas = permisos.slice(0, 8).map((permiso) => {
    const fila = documento.createElement("tr");
    const valores = [
      permiso.name || "-",
      permiso.request ? "Sí" : "No",
      visible(permiso.max),
      visible(permiso.requested),
      visible(permiso.remaining),
    ];
    fila.replaceChildren(...valores.map((valor, indice) => {
      const celda = documento.createElement(indice === 0 ? "th" : "td");
      if (indice === 0) celda.setAttribute("scope", "row");
      celda.textContent = valor;
      return celda;
    }));
    return fila;
  });
  if (!filas.length) {
    const filaVacia = documento.createElement("tr");
    const celdaVacia = documento.createElement("td");
    celdaVacia.className = "cronos-empty";
    celdaVacia.setAttribute("colspan", "5");
    celdaVacia.textContent = "Sin saldos de permisos disponibles.";
    filaVacia.replaceChildren(celdaVacia);
    filas.push(filaVacia);
  }
  cuerpo.replaceChildren(...filas);
  tabla.replaceChildren(cabecera, cuerpo);

  const envoltorioTabla = documento.createElement("div");
  envoltorioTabla.className = "cronos-table-wrap";
  envoltorioTabla.replaceChildren(tabla);

  destino.replaceChildren(navegacion, cajaResumen, envoltorioTabla);
}
