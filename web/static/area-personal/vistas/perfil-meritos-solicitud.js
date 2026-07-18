import {
  barraProgreso, botonOperacion, chip, encabezadoVista, enlaceRuta, escaparAtributo,
  escaparHTML, formatoPuntos, listaDatos, notaDemostracion, panel, tabla,
} from "./comunes.js";

export function renderizarPerfil(datos) {
  const perfil = datos.perfil;
  const formularioContacto = `<form id="formulario-contacto" data-operacion="actualizar_contacto">
    <div class="formulario-rejilla">
      <div class="campo"><label for="perfil-correo">Correo de avisos</label><input id="perfil-correo" name="correo" type="email" autocomplete="email" value="${escaparAtributo(perfil.correo)}" required><small>Los actos que requieran notificación fehaciente usarán el canal administrativo configurado.</small></div>
      <div class="campo"><label for="perfil-telefono">Teléfono de contacto</label><input id="perfil-telefono" name="telefono" type="tel" autocomplete="tel" value="${escaparAtributo(perfil.telefono)}" required></div>
      <div class="campo ancho-completo"><label for="perfil-domicilio">Domicilio a efectos de contacto</label><textarea id="perfil-domicilio" name="domicilio" autocomplete="street-address" required>${escaparHTML(perfil.domicilio)}</textarea></div>
    </div><div class="fila-acciones"><button type="submit" class="boton-primario">Revisar y guardar cambios</button></div></form>`;
  const preferencias = `<form id="formulario-notificaciones" data-operacion="actualizar_notificaciones"><fieldset><legend>Canales de aviso voluntarios</legend><label class="opcion-check"><input type="checkbox" name="correo" checked><span><strong>Correo electrónico</strong><small>Avisos de plazos, cambios y llamamientos.</small></span></label><label class="opcion-check"><input type="checkbox" name="telegram"><span><strong>Telegram</strong><small>Se activará cuando el conector y el consentimiento estén disponibles.</small></span></label><label class="opcion-check"><input type="checkbox" name="interno" checked><span><strong>Bandeja interna</strong><small>Siempre disponible dentro del área personal.</small></span></label></fieldset><p class="nota">Los avisos complementan, pero no sustituyen, una notificación administrativa cuando esta sea exigible.</p><button type="submit" class="boton-secundario">Guardar preferencias</button></form>`;

  return `${encabezadoVista("Perfil, identidad y contacto", "Una identidad común para solicitudes, bolsas, certificados y futuros módulos.")}
    ${datos.meta.presentacion ? notaDemostracion() : ""}
    <div class="rejilla-principal"><div>
      ${panel("Datos de identidad", "La identidad principal procede del sistema de autenticación", listaDatos([
        ["Nombre visible", escaparHTML(perfil.nombre_visible)], ["Identificador", escaparHTML(perfil.identificador_visible)],
        ["Referencia interna", escaparHTML(perfil.referencia)], ["Verificación", chip(perfil.estado_verificacion)],
        ["Provincia", escaparHTML(perfil.provincia)], ["Idioma", escaparHTML(perfil.idioma)],
      ]), { estado: perfil.estado_verificacion })}
      ${panel("Datos de contacto", "Puede proponer cambios; el servidor aplicará las validaciones y permisos", formularioContacto)}
    </div><aside>
      ${panel("Preferencias de comunicación", "Seleccione los avisos que desea recibir", preferencias)}
      ${panel("Privacidad y trazabilidad", "Control sobre el uso de sus datos", `<p>Los accesos y cambios relevantes quedan sujetos a registro de auditoría. Desde esta área podrá consultar el origen de los datos, rectificarlos cuando proceda y conocer qué documentos se reutilizan.</p>${enlaceRuta("ayuda", "Consultar protección de datos", "enlace-boton")}`)}
    </aside></div>`;
}

export function renderizarMeritos(datos) {
  const modoDemo = datos.meta.presentacion;
  const filasMeritos = datos.meritos.map((merito) => [
    `<strong>${escaparHTML(merito.titulo)}</strong><small>${escaparHTML(merito.id)}</small>`,
    escaparHTML(merito.tipo),
    `<span>${escaparHTML(merito.detalle)}</span><small>${formatoPuntos(merito.puntos_estimados)} puntos estimados</small>`,
    chip(merito.estado),
    `<div class="acciones-tabla"><button type="button" class="boton-secundario" data-accion="abrir-documento" data-id="${escaparAtributo(merito.documento_ref)}">Ver evidencia</button></div>`,
  ]);
  const filasDocumentos = datos.documentos.map((documento) => [
    `<strong>${escaparHTML(documento.nombre)}</strong><small>${escaparHTML(documento.id)}</small>`,
    escaparHTML(documento.tipo), escaparHTML(documento.fecha), chip(documento.estado),
    `<div class="acciones-tabla">${botonOperacion("solicitar_descarga", "Descargar", { id: documento.id, clase: "boton-secundario", descripcion: `Preparar descarga de ${documento.nombre}` })}</div>`,
  ]);
  const formulario = `<form id="formulario-merito" data-operacion="incorporar_merito"><div class="formulario-rejilla"><div class="campo"><label for="merito-tipo">Tipo de mérito</label><select id="merito-tipo" name="tipo" required><option value="">Seleccione</option><option>Titulación</option><option>Experiencia</option><option>Formación</option><option>Ejercicio superado</option><option>Otro mérito previsto en bases</option></select></div><div class="campo"><label for="merito-titulo">Denominación</label><input id="merito-titulo" name="titulo" required maxlength="180"></div><div class="campo"><label for="merito-jornada">Jornada, si corresponde</label><select id="merito-jornada" name="jornada"><option>No corresponde</option><option>Completa</option><option>Parcial 50 %</option><option>Parcial 33 %</option><option>Porcentaje distinto</option></select></div><div class="campo"><label for="merito-documento">Documento acreditativo</label><input id="merito-documento" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png"><small>${modoDemo ? "En demostración solo se usa el nombre y nunca se lee ni envía el fichero." : "El documento solo se incorporará cuando el servicio seguro confirme su custodia."}</small></div></div><p class="nota aviso">La aceptación y puntuación dependen de las bases de cada convocatoria. Aportar un mérito al inventario no implica su validación.</p><button type="submit" class="boton-primario">Revisar incorporación</button></form>`;

  return `${encabezadoVista("Méritos, títulos y documentos", "Inventario reutilizable con estado, evidencia y contribución a cada proceso.", `<button type="button" class="boton-primario" data-accion="enfocar-nuevo-merito">Añadir mérito</button>`)}
    <section class="resumen-cifras"><article class="cifra-resumen merito"><span>Méritos inventariados</span><strong>${datos.meritos.length}</strong><small>Todas las categorías</small></article><article class="cifra-resumen exito"><span>Validados</span><strong>${datos.meritos.filter((item) => item.estado === "Validado").length}</strong><small>Confirmados por personal técnico</small></article><article class="cifra-resumen aviso"><span>Pendientes o subsanables</span><strong>${datos.meritos.filter((item) => item.estado !== "Validado").length}</strong><small>Requieren revisión</small></article><article class="cifra-resumen"><span>Documentos</span><strong>${datos.documentos.length}</strong><small>Con versión y trazabilidad</small></article></section>
    ${panel("Inventario de méritos", "La puntuación se recalcula para las bases de cada proceso", tabla({ descripcion: "Méritos asociados al expediente personal", columnas: ["Mérito", "Tipo", "Detalle y estimación", "Estado", "Evidencia"], filas: filasMeritos }))}
    <div class="rejilla-dos"><div>${panel("Incorporar un mérito", "Aporte los datos y la evidencia disponible", formulario, { clase: "panel-nuevo-merito" })}</div><div>${panel("Documentos del expediente", "Descarga y estado de validación", tabla({ descripcion: "Documentos aportados u obtenidos de oficio", columnas: ["Documento", "Tipo", "Fecha", "Estado", "Acción"], filas: filasDocumentos }))}</div></div>`;
}

function pasosSolicitud(paso) {
  const etiquetas = ["Convocatoria", "Datos", "Méritos", "Autobaremo", "Pago, firma y registro"];
  return `<ol class="pasos" aria-label="Pasos de la solicitud">${etiquetas.map((etiqueta, indice) => `<li class="${paso === indice + 1 ? "activo" : paso > indice + 1 ? "completo" : ""}" ${paso === indice + 1 ? 'aria-current="step"' : ""}><span>Paso ${indice + 1}</span>${escaparHTML(etiqueta)}</li>`).join("")}</ol>`;
}

function accionesPaso(paso) {
  return `<div class="fila-acciones">${paso > 1 ? `<button type="button" class="boton-secundario" data-accion="paso-anterior">Anterior</button>` : ""}${paso < 5 ? `<button type="button" class="boton-primario" data-accion="paso-siguiente">Guardar y continuar</button>` : ""}</div>`;
}

function contenidoPaso(datos, estado, convocatoria) {
  const paso = estado.pasoSolicitud;
  if (paso === 1) {
    return `<fieldset><legend>Convocatoria seleccionada</legend>${datos.convocatorias.map((item) => `<label class="opcion-check"><input type="radio" name="convocatoria" value="${escaparAtributo(item.id)}" ${item.id === convocatoria.id ? "checked" : ""} data-accion="seleccionar-convocatoria"><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.referencia)} · ${escaparHTML(item.plazo)}</small></span></label>`).join("")}</fieldset><p class="nota aviso">Al continuar declara haber consultado las bases. La comprobación de requisitos se efectuará también en el servidor.</p>`;
  }
  if (paso === 2) {
    return `${listaDatos([["Identidad", escaparHTML(datos.perfil.nombre_visible)], ["Identificador", escaparHTML(datos.perfil.identificador_visible)], ["Correo", escaparHTML(datos.perfil.correo)], ["Teléfono", escaparHTML(datos.perfil.telefono)], ["Verificación", chip(datos.perfil.estado_verificacion)]])}<label class="opcion-check"><input type="checkbox" checked data-requisito-paso><span><strong>Confirmo que los datos de contacto están actualizados</strong><small>Puede modificarlos desde Perfil y contacto.</small></span></label>`;
  }
  if (paso === 3) {
    return `<fieldset><legend>Méritos que desea asociar</legend>${datos.meritos.map((item) => `<label class="opcion-check"><input type="checkbox" name="meritos" value="${escaparAtributo(item.id)}" checked><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.estado)} · ${formatoPuntos(item.puntos_estimados)} puntos estimados</small></span></label>`).join("")}</fieldset><p class="nota">Los documentos se reutilizan sin duplicar el fichero. La solicitud conserva la referencia y versión exactas.</p>`;
  }
  if (paso === 4) {
    const total = datos.baremo.reduce((suma, item) => suma + Number(item.puntos), 0);
    return `<div class="rejilla-dos"><div>${datos.baremo.map((item) => `<div class="criterio-baremo"><span><strong>${escaparHTML(item.nombre)}</strong><small>${escaparHTML(item.detalle)}</small></span>${barraProgreso(item.puntos, item.maximo)}<output>${formatoPuntos(item.puntos)}</output></div>`).join("")}</div><aside class="puntuacion-total"><span>Total autobaremado</span><output>${formatoPuntos(total)}</output><span>puntos provisionales</span></aside></div><p class="nota aviso">El cálculo no vincula a RRHH. Cada mérito será revisado conforme a las bases y quedará constancia de la decisión.</p>`;
  }
  const operaciones = estado.operacionesSolicitud || {};
  const demo = datos.meta.presentacion;
  return `<div class="rejilla-dos"><section><h3>1. Tasa o exención</h3><p>Importe mostrado: <strong>${escaparHTML(convocatoria.tasa)}</strong></p>${operaciones.iniciar_pago ? chip(demo ? "Pago DEMO confirmado" : "Pago confirmado") : botonOperacion("iniciar_pago", demo ? "Simular pago o exención" : "Pagar o acreditar exención", { id: convocatoria.id, descripcion: demo ? "Confirmar la simulación de pago o exención" : "Confirmar el pago o la acreditación de exención" })}</section><section><h3>2. Firma electrónica</h3><p>Se firmará la representación exacta de la solicitud.</p>${operaciones.firmar_solicitud ? chip(demo ? "Firma DEMO confirmada" : "Firma confirmada") : botonOperacion("firmar_solicitud", demo ? "Simular firma" : "Firmar solicitud", { id: convocatoria.id, descripcion: demo ? "Confirmar la simulación de firma electrónica" : "Firmar electrónicamente la solicitud" })}</section></div><section class="panel separacion-superior"><div class="panel-contenido"><h3>3. Registro</h3><p>Revise convocatoria, méritos, puntuación, tasa y firma. El registro es el acto que presenta la solicitud.</p>${operaciones.registrar_solicitud ? `<p class="nota ${demo ? "demo" : ""}"><strong>${demo ? "Registro DEMO completado." : "Solicitud registrada."}</strong> ${demo ? "Consulte el recibo mostrado; no existe asiento administrativo real." : "Conserve el recibo y el asiento devueltos por el servicio."}</p>` : botonOperacion("registrar_solicitud", demo ? "Simular firma y registro final" : "Registrar solicitud", { id: convocatoria.id, descripcion: demo ? "Registrar la solicitud de demostración sin efectos administrativos" : "Presentar la solicitud en el registro autorizado" })}</div></section>`;
}

export function renderizarSolicitud(datos, estado) {
  const convocatoria = datos.convocatorias.find((item) => item.id === estado.convocatoriaSolicitud) || datos.convocatorias.find((item) => item.estado === "Plazo abierto") || datos.convocatorias[0];
  return `${encabezadoVista("Nueva solicitud", `${convocatoria.titulo} · ${convocatoria.referencia}`, botonOperacion("guardar_borrador", datos.meta.presentacion ? "Guardar borrador DEMO" : "Guardar borrador", { id: convocatoria.id, clase: "boton-secundario", descripcion: datos.meta.presentacion ? "Guardar un borrador efímero de la solicitud" : "Guardar el borrador de forma segura" }))}
    ${pasosSolicitud(estado.pasoSolicitud)}
    <section class="panel"><header><div><h3>Paso ${estado.pasoSolicitud} de 5</h3><p>Los controles definitivos se reutilizan con el cliente productivo</p></div>${chip(estado.pasoSolicitud === 5 ? "Revisión final" : "En preparación")}</header><div class="panel-contenido">${contenidoPaso(datos, estado, convocatoria)}</div></section>
    ${accionesPaso(estado.pasoSolicitud)}`;
}

export function renderizarAutobaremacion(datos) {
  const total = datos.baremo.reduce((suma, criterio) => suma + Number(criterio.puntos), 0);
  const maximo = datos.baremo.reduce((suma, criterio) => suma + Number(criterio.maximo), 0);
  const criterios = datos.baremo.map((criterio) => `<article class="criterio-baremo"><span><strong>${escaparHTML(criterio.nombre)}</strong><small>${escaparHTML(criterio.detalle)} · ${escaparHTML(criterio.estado)}</small></span>${barraProgreso(criterio.puntos, criterio.maximo)}<output>${formatoPuntos(criterio.puntos)}</output></article>`).join("");
  const convocatoria = datos.convocatorias[0];
  return `${encabezadoVista("Autobaremación desglosada", "Estimación trazable aplicada a la versión de bases de la convocatoria.", botonOperacion("calcular_autobaremo", datos.meta.presentacion ? "Recalcular autobaremo DEMO" : "Recalcular autobaremo", { id: convocatoria?.id || "", descripcion: "Recalcular la autobaremación con los méritos seleccionados" }))}
    <div class="rejilla-principal"><div>${panel("Criterios aplicados", `${convocatoria?.titulo || "Convocatoria seleccionada"} · ${datos.meta.presentacion ? "versión sintética de bases" : "versión vigente de bases"}`, `<div class="desglose-baremo">${criterios}</div>`)}</div><aside>${panel("Resultado provisional", "Pendiente de revisión técnica", `<div class="puntuacion-total"><span>Puntuación estimada</span><output>${formatoPuntos(total)}</output><span>de ${formatoPuntos(maximo)} posibles</span></div><p>${barraProgreso(total, maximo)}</p><p class="nota aviso">No constituye puntuación oficial. RRHH aceptará, rechazará o ajustará cada concepto con motivación y trazabilidad.</p>`, { estado: "Provisional" })}${panel("Qué se tendrá en cuenta", "Reglas parametrizadas desde las bases", `<ul><li>Periodos exactos y solapamientos.</li><li>Porcentaje de jornada y reducciones.</li><li>Administración, categoría y rama.</li><li>Topes por bloque y puntuación máxima.</li><li>Documentos de oficio y aportados.</li></ul>`)}</aside></div>`;
}
