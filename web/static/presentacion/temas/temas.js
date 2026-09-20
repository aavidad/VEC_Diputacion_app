export const DISENOS_DIETAS = Object.freeze([
  { id: "01", nombre: "Tarea primero", criterio: "El itinerario ocupa el centro; el resto acompaña sin distraer." },
  { id: "02", nombre: "Asistente guiado", criterio: "Un paso cada vez, con resumen estable y siguiente acción inequívoca." },
  { id: "03", nombre: "Bandeja y detalle", criterio: "Las comisiones a la izquierda; la seleccionada se trabaja a la derecha." },
  { id: "04", nombre: "Mapa protagonista", criterio: "La ruta y sus tramos dominan; formulario y resumen quedan asociados." },
  { id: "05", nombre: "Cuadro operativo", criterio: "Situación general primero y acceso directo a lo que requiere atención." },
  { id: "06", nombre: "Agenda de viajes", criterio: "Orden cronológico para quien gestiona varias comisiones próximas." },
  { id: "07", nombre: "Tarjetas por estado", criterio: "Agrupa borradores, revisión y justificación con lectura muy visual." },
  { id: "08", nombre: "Escritorio compacto", criterio: "Más densidad, tabla central y detalle sin abandonar la bandeja." },
  { id: "09", nombre: "Resumen lateral", criterio: "Formulario amplio con decisión y totales siempre visibles a la derecha." },
  { id: "10", nombre: "Proceso vertical", criterio: "Los pasos forman una guía lateral y la tarea activa queda aislada." },
  { id: "11", nombre: "Acción rápida", criterio: "Reduce la página a crear una ruta y continuar; lo histórico pasa detrás." },
  { id: "12", nombre: "Historial útil", criterio: "Prioriza qué ha ocurrido y qué necesita una actuación del empleado." },
  { id: "13", nombre: "Gastos y justificación", criterio: "Pensada para completar importes y justificantes después del viaje." },
  { id: "14", nombre: "Panel progresivo", criterio: "Bloques breves y secuenciales que funcionan especialmente bien en móvil." },
  { id: "15", nombre: "Centro de trabajo", criterio: "Equilibra itinerario, mapa, bandeja y resumen en una sola vista profesional." },
].map((diseno) => Object.freeze(diseno)));

export function montarGaleriaDisenos(documento = globalThis.document) {
  const galeria = documento?.querySelector?.("#galeria-disenos");
  const plantilla = documento?.querySelector?.("#plantilla-diseno");
  const salida = documento?.querySelector?.("#diseno-elegido");
  if (!galeria || !plantilla?.content || !salida) return Object.freeze({ desmontar() {} });
  const escuchas = [];
  DISENOS_DIETAS.forEach((diseno) => {
    const fragmento = plantilla.content.cloneNode(true);
    const propuesta = fragmento.querySelector(".propuesta");
    propuesta.dataset.diseno = diseno.id;
    fragmento.querySelector(".propuesta-numero").textContent = diseno.id;
    fragmento.querySelector(".propuesta-titulo h2").textContent = diseno.nombre;
    fragmento.querySelector(".propuesta-criterio").textContent = diseno.criterio;
    const boton = fragmento.querySelector(".elegir-diseno");
    boton.textContent = `Elegir ${diseno.id} · ${diseno.nombre}`;
    const alElegir = () => {
      galeria.querySelectorAll(".propuesta").forEach((otra) => otra.removeAttribute("data-seleccionado"));
      propuesta.dataset.seleccionado = "true";
      salida.textContent = `Seleccionado: ${diseno.id} · ${diseno.nombre}`;
    };
    boton.addEventListener("click", alElegir);
    escuchas.push([boton, alElegir]);
    galeria.append(fragmento);
  });
  return Object.freeze({
    desmontar() { escuchas.forEach(([boton, escucha]) => boton.removeEventListener("click", escucha)); },
  });
}

if (typeof document !== "undefined") montarGaleriaDisenos(document);
