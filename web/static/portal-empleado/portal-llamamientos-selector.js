import { textoPortal, traducirPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";

/** La lectura autorizada de Bolsa ofrece las bolsas vigentes; emitir se autoriza al confirmar. */
export function bolsasVigentesParaLlamamiento(estadoBolsas) {
  if (estadoBolsas?.carga !== "listo" || !Array.isArray(estadoBolsas.datos?.bolsas)) return [];
  return estadoBolsas.datos.bolsas.filter((bolsa) => bolsa?.vigente_hasta === null);
}

export function renderizarSelectorLlamamientos({ estadoBolsas, encabezadoVista, escaparHTML }) {
  const cabecera = encabezadoVista("", traducirPortal("txt_nuevo_llamamiento"), "");
  const panel = (contenido, rol = "status") => `${cabecera}<section class="panel" aria-labelledby="selector-llamamientos-titulo">
    <div class="cabecera-panel"><h3 id="selector-llamamientos-titulo">${textoPortal("txt_bolsas_de_trabajo_activas")}</h3></div>
    <div class="cuerpo-panel" role="${rol}">${contenido}</div>
  </section>`;
  if (!estadoBolsas || estadoBolsas.carga === "cargando") {
    return panel(`<p aria-busy="true">${textoPortal("txt_cargando_bolsas_de_trabajo")}</p>`);
  }
  if (estadoBolsas.carga === "denegado") {
    return panel(`<p>${textoPortal("txt_acceso_denegado_a_la_consulta_de_bolsas")}</p>`, "alert");
  }
  if (estadoBolsas.carga !== "listo") {
    return panel(`<p>${textoPortal("txt_no_se_pudieron_consultar_las_bolsas_de_trabajo_http")}</p>
      <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">${textoPortal("txt_reintentar")}</button>`, "alert");
  }
  const bolsas = bolsasVigentesParaLlamamiento(estadoBolsas);
  if (bolsas.length === 0) {
    return panel(`<div class="vacio-controlado"><p><strong>${textoPortal("txt_no_hay_bolsas_de_trabajo_activas")}</strong></p>
      <p>${textoPortal("txt_el_servicio_no_ha_devuelto_bolsas_de_trabajo_reg")}</p>
      <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">${textoPortal("txt_reintentar")}</button></div>`);
  }
  return panel(`<div class="tabla-contenedor"><table class="tabla-datos tabla-apilable">
    <thead><tr><th scope="col">${textoPortal("txt_bolsa")}</th><th scope="col">${textoPortal("txt_accion")}</th></tr></thead>
    <tbody>${bolsas.map((bolsa) => `<tr>
      <th scope="row">${escaparHTML(bolsa.categoria)}</th>
      <td data-etiqueta="${textoPortal("txt_accion")}"><button type="button" class="boton-primario"
        data-elegir-bolsa-llamamiento="${escaparHTML(bolsa.bolsa_ref)}"
        aria-label="${escaparHTML(`${traducirPortal("txt_nuevo_llamamiento")}: ${bolsa.categoria}`)}">${textoPortal("txt_nuevo_llamamiento")}</button></td>
    </tr>`).join("")}</tbody></table></div>`);
}

export function renderizarPantallaLlamamientos({ estado, encabezadoVista, escaparHTML, presentador }) {
  if (!estado.bolsaSeleccionada) {
    return renderizarSelectorLlamamientos({ estadoBolsas: estado.datosBolsas, encabezadoVista, escaparHTML });
  }
  if (!estado.llamamientoDesdeMenu) return null;
  const carga = estado.datosCandidatos?.carga;
  if (carga !== "error" && carga !== "denegado") return presentador.renderizarSoloBolsas("bolsa-candidatos");
  const denegado = carga === "denegado";
  return `${encabezadoVista("", traducirPortal("txt_nuevo_llamamiento"), "")}
    <section class="panel"><div class="cuerpo-panel vacio-controlado" role="alert">
      <p><strong>${textoPortal(denegado ? "txt_acceso_denegado" : "txt_error_al_consultar_candidatos")}</strong></p>
      <p>${textoPortal(denegado ? "txt_la_sesion_no_dispone_de_permisos_para_consultar" : "txt_no_se_pudieron_consultar_los_candidatos_http")}</p>
      <div class="acciones-vista">
        ${denegado ? "" : `<button type="button" class="boton-secundario" data-reintentar-bolsa-llamamiento>${textoPortal("txt_reintentar")}</button>`}
        <button type="button" class="boton-primario" data-cambiar-bolsa-llamamiento>${textoPortal("panel_b7_paso_bolsa")}</button>
      </div>
    </div></section>`;
}

export function instalarSelectorLlamamientos({ documento, estado, controladorBolsas, actualizarVistaBolsa, porId }) {
  function cargarEIniciar(bolsaRef) {
    estado.llamamientoDesdeMenu = true;
    estado.filtrosBolsa = { estado: "", texto: "" };
    void controladorBolsas.cargarCandidatosBolsa(bolsaRef).then(() => {
      if (estado.vista !== "llamamientos" || !estado.llamamientoDesdeMenu
        || estado.bolsaSeleccionada !== bolsaRef || estado.datosCandidatos?.carga !== "listo") return;
      // La acción existente prepara plazo, correo y confirmación B7. No se emite nada aquí.
      porId("espacio-trabajo")?.querySelector('[data-bolsa-accion="iniciar-b7"]')?.click();
    });
  }
  documento.addEventListener("click", (evento) => {
    if (estado.vista !== "llamamientos") return;
    const cancelar = evento.target.closest?.('[data-bolsa-accion="cancelar-b7"], [data-cambiar-bolsa-llamamiento]');
    if (cancelar && estado.llamamientoDesdeMenu) {
      estado.llamamientoDesdeMenu = false;
      estado.bolsaSeleccionada = "";
      actualizarVistaBolsa();
      return;
    }
    if (evento.target.closest?.("[data-reintentar-bolsa-llamamiento]")) {
      evento.preventDefault();
      if (estado.llamamientoDesdeMenu && estado.bolsaSeleccionada) cargarEIniciar(estado.bolsaSeleccionada);
      return;
    }
    const boton = evento.target.closest?.("[data-elegir-bolsa-llamamiento]");
    if (!boton || !boton.closest("#espacio-trabajo")) return;
    evento.preventDefault();
    const bolsaRef = boton.dataset.elegirBolsaLlamamiento;
    if (!bolsasVigentesParaLlamamiento(estado.datosBolsas).some((bolsa) => bolsa.bolsa_ref === bolsaRef)) return;
    cargarEIniciar(bolsaRef);
  });
}
