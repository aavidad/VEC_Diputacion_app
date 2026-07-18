import {
  botonOperacion, chip, encabezadoVista, enlaceRuta, escaparAtributo, escaparHTML,
  formatoPuntos, listaDatos, notaDemostracion, panel, tabla,
} from "./comunes.js";

export function renderizarInicio(datos) {
  const cifras = [
    [datos.resumen.acciones_pendientes, "Acciones pendientes", "Revise plazos y comunicaciones", "aviso"],
    [datos.resumen.convocatorias_abiertas, "Convocatorias abiertas", "Procesos disponibles ahora", ""],
    [datos.resumen.solicitudes_activas, "Solicitudes activas", "En tramitación o revisión", "exito"],
    [formatoPuntos(datos.resumen.puntuacion_provisional), "Puntos provisionales", "Sujeto a revisión técnica", "merito"],
  ].map(([valor, etiqueta, ayuda, clase]) => `<article class="cifra-resumen ${clase}"><span>${escaparHTML(etiqueta)}</span><strong>${escaparHTML(valor)}</strong><small>${escaparHTML(ayuda)}</small></article>`).join("");

  const plazos = datos.plazos.map((plazo) => `<li><span class="fecha-bloque">${escaparHTML(plazo.dia)}<small>${escaparHTML(plazo.mes)}</small></span><span><strong>${escaparHTML(plazo.titulo)}</strong><small>${escaparHTML(plazo.detalle)}</small></span>${enlaceRuta(plazo.ruta, "Abrir", "enlace-boton")}</li>`).join("");
  const acciones = datos.mensajes.filter((mensaje) => mensaje.estado === "No leído").slice(0, 3).map((mensaje) => `<li><span class="fecha-bloque" aria-hidden="true">!</span><span><strong>${escaparHTML(mensaje.asunto)}</strong><small>${escaparHTML(mensaje.resumen)}</small></span>${enlaceRuta(mensaje.ruta, "Atender", "enlace-boton")}</li>`).join("");
  const solicitudes = datos.solicitudes.map((solicitud) => [
    `<strong>${escaparHTML(solicitud.titulo)}</strong><small>${escaparHTML(solicitud.referencia)}</small>`,
    chip(solicitud.estado),
    `<strong>${formatoPuntos(solicitud.puntuacion)} puntos</strong><small>${escaparHTML(solicitud.posicion)}</small>`,
    `<div class="acciones-tabla"><button type="button" class="boton-secundario" data-accion="abrir-expediente" data-id="${escaparAtributo(solicitud.id)}">Ver seguimiento</button></div>`,
  ]);
  const actividad = datos.actividad.slice(0, 4).map((item) => `<li><span class="fecha-bloque" aria-hidden="true">·</span><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.detalle)}</small><small>${escaparHTML(item.actor)}</small></span><small>${escaparHTML(item.fecha)}</small></li>`).join("");

  return `${encabezadoVista("Qué necesita su atención", "Plazos, acciones y estado de sus procesos en un único espacio.", enlaceRuta("convocatorias", "Ver convocatorias", "boton-primario"))}
    ${datos.meta.presentacion ? notaDemostracion() : ""}
    <section class="resumen-cifras" aria-label="Resumen personal">${cifras}</section>
    <div class="rejilla-principal"><div>
      ${panel("Próximos plazos", "Fechas calculadas para sus procesos", `<ul class="lista-plazos">${plazos}</ul>`, { estado: `${datos.plazos.length} plazos` })}
      ${panel("Mis solicitudes", "Estado jurídico y siguiente actuación", tabla({ descripcion: "Solicitudes asociadas a la identidad autenticada", columnas: ["Proceso", "Estado", "Puntuación y posición", "Acción"], filas: solicitudes }))}
    </div><aside>
      ${panel("Requiere atención", "Tareas que pueden tener plazo", acciones ? `<ul class="lista-mensajes">${acciones}</ul>` : `<p>No existen acciones pendientes.</p>`, { estado: `${datos.resumen.acciones_pendientes} pendientes` })}
      ${panel("Actividad reciente", "Trazabilidad visible para la persona interesada", `<ul class="lista-actividad">${actividad}</ul>`)}
    </aside></div>`;
}

export function renderizarConvocatorias(datos, estado) {
  const termino = estado.filtros?.termino?.toLowerCase() || "";
  const filtroEstado = estado.filtros?.estado || "Todas";
  const filtroCategoria = estado.filtros?.categoria || "Todas";
  const categorias = [...new Set(datos.convocatorias.map((item) => item.categoria))];
  const resultados = datos.convocatorias.filter((item) => {
    const coincideTexto = !termino || `${item.titulo} ${item.referencia} ${item.categoria}`.toLowerCase().includes(termino);
    return coincideTexto && (filtroEstado === "Todas" || item.estado === filtroEstado)
      && (filtroCategoria === "Todas" || item.categoria === filtroCategoria);
  });
  const tarjetas = resultados.map((convocatoria) => `<article class="tarjeta-convocatoria"><div><div class="metadatos"><span>${escaparHTML(convocatoria.referencia)}</span>${chip(convocatoria.estado)}<span>${escaparHTML(convocatoria.categoria)}</span></div><h3>${escaparHTML(convocatoria.titulo)}</h3><p>${escaparHTML(convocatoria.descripcion)}</p><div class="metadatos"><strong>${escaparHTML(convocatoria.plazo)}</strong><span>${escaparHTML(convocatoria.tasa)}</span></div></div><div class="fila-acciones"><button type="button" class="boton-secundario" data-accion="abrir-convocatoria" data-id="${escaparAtributo(convocatoria.id)}">Ver detalle y requisitos</button>${convocatoria.estado === "Plazo abierto" ? `<button type="button" class="boton-primario" data-accion="iniciar-solicitud" data-id="${escaparAtributo(convocatoria.id)}">Iniciar solicitud</button>` : ""}</div></article>`).join("");

  return `${encabezadoVista("Convocatorias y procesos selectivos", "Consulte bases, requisitos, plazos y estado antes de iniciar una solicitud.")}
    <section class="panel"><form class="filtros" id="filtros-convocatorias" data-accion="filtrar-convocatorias"><div class="campo"><label for="filtro-texto">Buscar</label><input id="filtro-texto" name="termino" type="search" value="${escaparAtributo(estado.filtros?.termino || "")}" placeholder="Título, referencia o categoría"></div><div class="campo"><label for="filtro-estado">Estado</label><select id="filtro-estado" name="estado">${["Todas", ...new Set(datos.convocatorias.map((item) => item.estado))].map((valor) => `<option${valor === filtroEstado ? " selected" : ""}>${escaparHTML(valor)}</option>`).join("")}</select></div><div class="campo"><label for="filtro-categoria">Categoría</label><select id="filtro-categoria" name="categoria">${["Todas", ...categorias].map((valor) => `<option${valor === filtroCategoria ? " selected" : ""}>${escaparHTML(valor)}</option>`).join("")}</select></div><button type="submit" class="boton-primario">Aplicar filtros</button></form><div class="panel-contenido">${tarjetas || `<div class="estado-vacio"><strong>Sin resultados</strong>Modifique los filtros para ver otras convocatorias.</div>`}</div></section>`;
}

export function renderizarDetalleConvocatoria(datos, estado) {
  const convocatoria = datos.convocatorias.find((item) => item.id === estado.convocatoriaSeleccionada) || datos.convocatorias[0];
  const requisitos = convocatoria.requisitos.map((requisito) => `<li>${escaparHTML(requisito)}</li>`).join("");
  const documentos = convocatoria.documentos.map((documento, indice) => `<li><span class="fecha-bloque" aria-hidden="true">${indice + 1}</span><span><strong>${escaparHTML(documento)}</strong><small>Documento público de la convocatoria</small></span>${botonOperacion("solicitar_descarga", datos.meta.presentacion ? "Descargar DEMO" : "Descargar", { id: convocatoria.id, clase: "boton-secundario", descripcion: `Preparar descarga de ${documento}` })}</li>`).join("");
  const acciones = `<button type="button" class="boton-secundario" data-accion="volver-convocatorias">Volver al listado</button>${convocatoria.estado === "Plazo abierto" ? `<button type="button" class="boton-primario" data-accion="iniciar-solicitud" data-id="${escaparAtributo(convocatoria.id)}">Iniciar solicitud</button>` : ""}`;
  return `${encabezadoVista(convocatoria.titulo, convocatoria.referencia, acciones)}
    <div class="rejilla-principal"><div>
      ${panel("Resumen de la convocatoria", convocatoria.descripcion, listaDatos([
        ["Estado", chip(convocatoria.estado)], ["Categoría", escaparHTML(convocatoria.categoria)], ["Plazo", escaparHTML(convocatoria.plazo)],
        ["Presentación hasta", escaparHTML(convocatoria.presentacion_hasta)], ["Plazas o bolsa", escaparHTML(convocatoria.plazas)], ["Tasa", escaparHTML(convocatoria.tasa)],
      ]), { estado: convocatoria.estado })}
      ${panel("Requisitos de participación", "La comprobación definitiva se realizará según las bases", `<ul>${requisitos}</ul><p class="nota aviso">Revise las bases completas. Esta lista facilita la consulta, pero no sustituye al texto aprobado.</p>`)}
    </div><aside>
      ${panel("Documentación pública", "Bases, anexos y baremo", `<ul class="lista-documentos">${documentos}</ul>`)}
      ${panel("Antes de empezar", "El asistente comprobará cada paso", `<ol><li>Datos personales y contacto.</li><li>Requisitos y méritos reutilizables.</li><li>Autobaremación.</li><li>Pago, firma y registro.</li></ol>`)}
    </aside></div>`;
}
