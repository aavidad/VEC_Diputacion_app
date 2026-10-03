/** Adaptador del puerto ADMIN existente. No abre transporte ni crea permisos. */
export function crearClienteUsuarios({ administracion, buscarUsuarios } = {}) {
  if (!administracion) return Object.freeze({});
  const cliente = {};
  for (const metodo of ["capacidades", "roles", "persona", "propuestas", "cerrarPropuesta", "aplicar", "proponer", "aplicarLote", "proponerLote"]) {
    if (typeof administracion[metodo] === "function") cliente[metodo] = (...args) => administracion[metodo](...args);
  }
  if (typeof buscarUsuarios === "function") cliente.buscar = buscarUsuarios;
  else if (typeof administracion.buscar === "function") cliente.buscar = (filtros, signal) => {
    if (filtros.perfil_ref || filtros.unidad_ref || filtros.estado) {
      throw Object.assign(new Error("filtros_no_disponibles"), { codigo: "filtros_no_disponibles" });
    }
    return administracion.buscar(filtros.busqueda, signal);
  };
  return Object.freeze(cliente);
}
