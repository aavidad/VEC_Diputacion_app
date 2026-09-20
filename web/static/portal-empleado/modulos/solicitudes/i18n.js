export const MENSAJES_SOLICITUDES_ES = Object.freeze({
  sobrelinea: "Portal del empleado · demostración RRHH",
  titulo: "Solicitudes y certificados",
  descripcion: "Consulta una bandeja de trámites y prepara la información necesaria. Esta pantalla no registra ni envía solicitudes.",
  demostracion: "Datos sintéticos · recorrido visual sin conexión",
  buscar: "Buscar por trámite o referencia",
  filtrar: "Estado",
  todos: "Todos los estados",
  nueva: "Preparar solicitud",
  ver: "Ver detalle",
  seguimiento: "Seguimiento",
  certificados: "Certificados",
});

export function crearTraductorSolicitudes(mensajes = MENSAJES_SOLICITUDES_ES) {
  return (clave) => mensajes[clave] || clave;
}
