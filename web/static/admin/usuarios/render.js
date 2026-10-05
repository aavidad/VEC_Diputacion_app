import { etiquetaNombreMetadatos } from "./metadatos.js?v=20261004-admin-usuarios-metadata-v1";
export const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
export function crearRender({ root, id, textos, metadatos = false }) {
  const t = textos.traducir;
  const tx = (clave, variables) => escapar(t(clave, variables));
  const el = (clave) => root.querySelector(`#${id(clave)}`);
  const fecha = (valor) => valor ? textos.fecha(new Date(valor), { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }) : t("detalle.sin_fecha");
  const campo = (clave, valor) => `<div class="fila-resumen"><dt>${tx(clave)}</dt><dd>${escapar(valor)}</dd></div>`;
  const boton = (accion, clave, extra = "", clase = "boton-secundario") => `<button type="button" class="${clase}" data-accion="${accion}" ${extra}>${tx(clave)}</button>`;
  const estado = (valor) => `<span class="estado-chip ${["activo", "vigente"].includes(valor) ? "exito" : "neutro"}">${tx(`estados.${valor === "vigente" ? "activo" : valor}`)}</span>`;
  const nombre = (p) => p.proyeccion === "metadatos_v1" ? etiquetaNombreMetadatos(p, t) : p.nombre;
  const unidadNombre = (p) => p.proyeccion === "metadatos_v1" ? t("metadatos.unidad_no_consultada") : p.unidad_nombre;
  const tecnico = (ref) => `<details><summary>${tx("general.tecnico")}</summary><code>${escapar(ref)}</code></details>`;
  function pantalla() {
    root.innerHTML = `<section class="usuarios" aria-labelledby="${id("titulo")}">
      <header class="cabeza-pagina"><div><h2 id="${id("titulo")}">${tx("general.titulo")}</h2></div>
        ${boton("recargar", "general.recargar", `id="${id("recargar")}"`)}<details class="usuarios-ayuda"><summary class="boton-secundario" aria-label="${tx("general.ayuda_nombre")}">${tx("general.ayuda")}</summary><p>${tx(metadatos ? "metadatos.ayuda" : "general.ayuda_contenido")}</p></details></header>
      <div class="usuarios-pestanas" role="tablist" aria-label="${tx("general.pestanas")}">
        <button class="boton-secundario" type="button" role="tab" aria-selected="true" aria-controls="${id("panel-usuarios")}" id="${id("tab-usuarios")}" data-accion="usuarios">${tx("general.usuarios")}</button>
        <button class="boton-secundario" type="button" role="tab" aria-selected="false" tabindex="-1" aria-controls="${id("panel-perfiles")}" id="${id("tab-perfiles")}" data-accion="perfiles">${tx("general.perfiles")}</button>
        <button class="boton-secundario" type="button" role="tab" aria-selected="false" tabindex="-1" aria-controls="${id("panel-propuestas")}" id="${id("tab-propuestas")}" data-accion="propuestas">${tx("propuestas.titulo")}</button></div>
      <p id="${id("estado")}" role="status" aria-live="polite" class="usuarios-estado"></p>
      <div class="usuarios-trabajo"><section role="tabpanel" id="${id("panel-usuarios")}" aria-labelledby="${id("tab-usuarios")}">
        <section class="panel" id="${id("listado")}" aria-labelledby="${id("lista-titulo")}"><div class="cabecera-panel"><h3 id="${id("lista-titulo")}">${tx(metadatos ? "metadatos.titulo_lista" : "busqueda.titulo")}</h3></div>
          ${metadatos ? `<p class="cuerpo-panel texto-secundario">${tx("metadatos.filtros_no_disponibles")}</p>` : ""}
          <form id="${id("buscar")}" class="cuerpo-panel usuarios-filtros">
            <label class="campo" for="${id("consulta")}"><span>${tx("busqueda.texto")}</span><input type="search" id="${id("consulta")}" maxlength="132" autocomplete="off"></label>
            <label class="campo" for="${id("perfil")}"><span>${tx("busqueda.perfil")}</span><select id="${id("perfil")}"></select></label>
            <label class="campo" for="${id("unidad")}"><span>${tx("busqueda.unidad")}</span><select id="${id("unidad")}"></select></label>
            <label class="campo" for="${id("vigencia")}"><span>${tx("busqueda.vigencia")}</span><select id="${id("vigencia")}"><option value="">${tx("busqueda.todos_estados")}</option>
              <option value="vigente">${tx("estados.activo")}</option><option value="caducado">${tx("estados.caducado")}</option></select></label>
            <div class="acciones-paso"><button type="submit" class="boton-primario" id="${id("buscar-boton")}">${tx("busqueda.buscar")}</button>${boton("limpiar", "busqueda.limpiar", `id="${id("limpiar")}"`)}</div>
          </form><div id="${id("filtros-activos")}" class="usuarios-filtros-activos"></div><div id="${id("resultados")}"></div></section>
        <section class="panel" id="${id("detalle")}" tabindex="-1" hidden></section>
        <section class="panel" id="${id("revision")}" tabindex="-1" hidden></section>
      </section><section class="panel" role="tabpanel" id="${id("panel-perfiles")}" aria-labelledby="${id("tab-perfiles")}" hidden></section>
      <section class="panel" role="tabpanel" id="${id("panel-propuestas")}" aria-labelledby="${id("tab-propuestas")}" hidden></section></div>
    </section>`;
    el("perfil").innerHTML = `<option value="">${tx("busqueda.todos")}</option>`;
    el("unidad").innerHTML = `<option value="">${tx("busqueda.todas")}</option>`;
  }
  function filtros(roles, unidades = []) {
    const previo = el("perfil").value;
    el("perfil").innerHTML = `<option value="">${tx("busqueda.todos")}</option>` + roles.map((r) => `<option value="${escapar(r.version_ref)}">${escapar(r.etiqueta)}</option>`).join("");
    el("perfil").value = previo;
    const unidad = el("unidad").value;
    el("unidad").innerHTML = `<option value="">${tx("busqueda.todas")}</option>` + unidades.map((u) => `<option value="${escapar(u.unidad_ref)}">${escapar(u.nombre)}</option>`).join("");
    el("unidad").value = unidad;
    el("unidad").disabled = unidades.length === 0;
    el("unidad").setAttribute("aria-describedby", id("estado"));
  }
  function activos(valores, roles, unidades) {
    const etiquetas = { busqueda: valores.busqueda, perfil_ref: roles.find((r) => r.version_ref === valores.perfil_ref)?.etiqueta,
      unidad_ref: unidades.find((u) => u.unidad_ref === valores.unidad_ref)?.nombre, estado: valores.estado ? t(`estados.${valores.estado === "vigente" ? "activo" : valores.estado}`) : "" };
    el("filtros-activos").innerHTML = Object.entries(etiquetas).filter(([, v]) => v).map(([c, v]) =>
      `<button type="button" class="filtro-activo" data-accion="quitar-filtro" data-campo="${c}" aria-label="${tx("busqueda.quitar", { filtro: v })}">${escapar(v)} <span aria-hidden="true">${tx("busqueda.quitar_simbolo")}</span></button>`).join("");
  }
  function tabla(pagina, roles) {
    el("resultados").innerHTML = pagina.personas.length ? `<div class="tabla-contenedor" tabindex="0" aria-label="${tx("busqueda.tabla")}"><table class="tabla-datos"><caption>${tx("busqueda.cuenta", { cuenta: textos.numero(pagina.personas.length) })}</caption><thead><tr>
      <th scope="col">${tx("busqueda.nombre")}</th><th scope="col">${tx("busqueda.unidad")}</th><th scope="col">${tx("busqueda.perfiles")}</th></tr></thead><tbody>${pagina.personas.map((p, n) => `<tr><th scope="row"><button type="button" class="enlace-fila" data-accion="persona" data-ref="${escapar(p.persona_ref)}">${escapar(p.proyeccion === "metadatos_v1" && p.nombre_estado !== "consultado" ? t("metadatos.persona_en_lista", { numero: textos.numero(n + 1), nombre: nombre(p) }) : nombre(p))}</button></th>
      <td>${escapar(unidadNombre(p))}</td><td>${p.perfiles?.length ? p.perfiles.map((perfil, i) => `<div>${escapar(roles.find((r) => r.version_ref === perfil.rol_version_ref)?.etiqueta || (p.proyeccion === "metadatos_v1" ? t("metadatos.perfil_en_lista", { numero: textos.numero(i + 1) }) : t("detalle.nombre_pendiente")))} ${estado(perfil.estado)}</div>`).join("") : tx(p.perfiles ? "detalle.sin_perfiles" : "detalle.resumen_pendiente")}</td></tr>`).join("")}</tbody></table></div>`
      : `<p class="cuerpo-panel">${tx("busqueda.vacia")}</p>`;
    if (pagina.siguiente_cursor) el("resultados").innerHTML += `<div class="pie-panel">${boton("mas", "busqueda.mas")}</div>`;
  }
  function catalogo(roles) {
    el("panel-perfiles").innerHTML = `<div class="cabecera-panel"><div><h3>${tx("catalogo.titulo")}</h3><p>${tx("catalogo.descripcion")}</p></div><span class="estado-chip neutro">${tx("catalogo.lectura")}</span></div>
      <div class="tabla-contenedor" tabindex="0" aria-label="${tx("catalogo.tabla")}"><table class="tabla-datos"><caption>${tx("catalogo.tabla")}</caption><thead><tr><th scope="col">${tx("catalogo.nombre")}</th><th scope="col">${tx("catalogo.circuito")}</th><th scope="col">${tx("catalogo.definicion")}</th></tr></thead><tbody>
      ${roles.map((r) => `<tr><th scope="row">${escapar(r.etiqueta)}</th><td>${tx(r.clase === "ordinario" ? "catalogo.ordinario" : "catalogo.doble")}</td><td>${tx(r.fijo ? "catalogo.fijo" : "catalogo.gestionado")}</td></tr>`).join("")}</tbody></table></div>`;
  }
  function ficha(datos, roles, disponibles) {
    if (datos.proyeccion === "metadatos_v1") {
      el("detalle").innerHTML = `<div class="cabecera-panel"><div><h3>${escapar(nombre(datos))}</h3><p>${tx("metadatos.unidad_no_consultada")}</p></div>${boton("volver", "general.volver")}</div><div class="cuerpo-panel">
        <p class="texto-secundario">${tx("metadatos.alcance")}</p><h4>${tx("detalle.perfiles")}</h4>
        ${datos.perfiles.length ? `<ul class="usuarios-perfiles">${datos.perfiles.map((p, i) => `<li><div class="usuarios-fila-perfil"><strong>${tx("metadatos.perfil_en_lista", { numero: textos.numero(i + 1) })}</strong>${estado(p.estado)}</div>
          <dl class="resumen-expediente">${campo("detalle.desde", fecha(p.vigente_desde))}${campo("detalle.hasta", fecha(p.vigente_hasta))}</dl>${tecnico(p.perfil_ref)}</li>`).join("")}</ul>` : `<p>${tx("detalle.sin_perfiles")}</p>`}
        <p>${tx("metadatos.actos_no_consultados")}</p><p>${tx("metadatos.historia_no_consultada")}</p>${tecnico(datos.persona_ref)}</div>`;
      return;
    }
    rolesHistoria = roles;
    el("detalle").innerHTML = `<div class="cabecera-panel"><div><h3>${escapar(datos.nombre)}</h3><p>${escapar(datos.unidad_nombre)}</p></div>${boton("volver", "general.volver")}</div><div class="cuerpo-panel">
      <h4>${tx("detalle.perfiles")}</h4>${datos.perfiles.length ? `<ul class="usuarios-perfiles">${datos.perfiles.map((p) => `<li><div class="usuarios-fila-perfil"><strong>${escapar(roles.find((r) => r.version_ref === p.rol_version_ref)?.etiqueta || t("detalle.nombre_pendiente"))}</strong>${estado(p.estado)}</div>
        <dl class="resumen-expediente">${campo("detalle.ambito", ambito(p))}${campo("detalle.desde", fecha(p.vigente_desde))}${campo("detalle.hasta", fecha(p.vigente_hasta))}</dl></li>`).join("")}</ul>` : `<p>${tx("detalle.sin_perfiles")}</p>`}
      <form id="${id("seleccion")}"><fieldset class="usuarios-seleccion"><legend>${tx("detalle.cambiar")}</legend>
        <label class="campo" for="${id("operacion")}"><span>${tx("detalle.operacion")}</span><select id="${id("operacion")}"><option value="otorgar">${tx("operaciones.otorgar")}</option><option value="revocar">${tx("operaciones.revocar")}</option></select></label>
        <div id="${id("opciones")}">${opciones(disponibles)}</div><div id="${id("error-seleccion")}" role="alert" tabindex="-1" hidden></div>
        <div class="acciones-paso"><button type="submit" class="boton-primario" id="${id("revisar")}" ${disponibles.length ? "" : "disabled"}>${tx("detalle.revisar")}</button></div></fieldset></form>
      <div class="usuarios-acceso"><button type="button" class="boton-peligro" disabled aria-describedby="${id("limite-acceso")}">${tx("detalle.retirar_acceso")}</button><p id="${id("limite-acceso")}" class="texto-secundario">${tx("detalle.sin_revocacion_acceso")}</p></div>
      <details class="usuarios-historia"><summary>${tx("detalle.historia")}</summary>${historia(datos.historia)}</details></div>`;
  }
  function opciones(disponibles) {
    return disponibles.length ? disponibles.map(({ acto, rol, indice }) => `<div class="usuarios-opcion"><label class="usuarios-checkbox"><input type="checkbox" data-seleccion="${indice}"><span>${escapar(rol.etiqueta)}</span></label>
      <p class="texto-secundario">${tx(rol.clase === "ordinario" ? "catalogo.ordinario" : "catalogo.doble")}</p>
      <label class="campo" for="${id(`motivo-${indice}`)}"><span>${tx("detalle.motivo")}</span><select data-motivo="${indice}" id="${id(`motivo-${indice}`)}"><option value="">${tx("detalle.elegir_motivo")}</option>${acto.motivos.map((m, n) => `<option value="${n}">${escapar(m.etiqueta)}</option>`).join("")}</select></label></div>`).join("") : `<p>${tx("detalle.sin_actos")}</p>`;
  }
  const ambito = (item) => item.ambito_etiqueta || item.ambitos?.map((a) => a.nombre).filter(Boolean).join(t("general.separador")) || t("detalle.ambito_pendiente");
  function identidad(actor) {
    if (!actor?.nombre) return t("detalle.identidad_pendiente");
    return t("historia.identidad", { persona: actor.nombre, perfil: actor.perfil_activo_nombre || t("historia.perfil_pendiente") });
  }
  function historia(items) {
    return items.length ? `<ol class="usuarios-historial">${items.map((h) => {
      const efectiva = /^recibo_admin:[a-f0-9]{32}$/u.test(h.recibo_ref || "") && ((h.operacion === "otorgar" && h.estado === "activo") || (h.operacion === "revocar" && h.estado === "revocado"));
      const rol = h.rol_etiqueta || rolesHistoria.find((r) => r.version_ref === h.rol_version_ref)?.etiqueta || t("detalle.nombre_pendiente");
      return `<li><time datetime="${escapar(h.confirmado_en)}">${escapar(fecha(h.confirmado_en))}</time>
        <p>${tx("historia.actuacion", { actor: identidad(h.actor), operacion: t(efectiva ? `historia.${h.operacion}` : "historia.registro"), perfil: rol, persona: h.objetivo_nombre || t("detalle.identidad_pendiente") })}</p>
        ${h.proponente ? `<p>${tx("historia.proponente", { persona: identidad(h.proponente) })}</p>` : ""}
        ${h.aprobador ? `<p>${tx("historia.aprobador", { persona: identidad(h.aprobador) })}</p>` : ""}
        <dl class="resumen-expediente">${campo("detalle.ambito", ambito(h))}${campo("detalle.desde", fecha(h.vigente_desde))}${campo("detalle.hasta", fecha(h.vigente_hasta))}
          ${campo("detalle.motivo_corto", h.motivo?.etiqueta || t("historia.motivo_pendiente"))}${h.referencia_acto ? campo("historia.acto", h.referencia_acto) : ""}</dl>${tecnico(h.acto_ref)}</li>`;
    }).join("")}</ol>` : `<p>${tx("detalle.sin_historia")}</p>`;
  }
  let rolesHistoria = [];
  function revision(decision, datos, motivos, disponible, unidades = []) {
    el("revision").innerHTML = `<div class="cabecera-panel"><h3>${tx("revision.titulo")}</h3><span class="estado-chip">${tx("revision.paso")}</span></div><div class="cuerpo-panel"><dl class="resumen-expediente">${campo("revision.persona", datos.nombre)}${campo("busqueda.unidad", datos.unidad_nombre)}</dl>
      <ul class="usuarios-perfiles">${decision.seleccion.map(({ acto, rol, indice }) => {
        const vigencia = acto.operacion === "revocar" ? datos.perfiles.find((p) => p.perfil_ref === acto.objetivo.perfil_ref) : acto.objetivo;
        const alcance = acto.ambitos ? ambito(acto) : acto.operacion === "revocar" ? ambito(vigencia)
          : unidades.find((u) => u.unidad_ref === acto.objetivo.unidad_ref)?.nombre || t("detalle.ambito_pendiente");
        return `<li><h4>${escapar(rol.etiqueta)}</h4><dl class="resumen-expediente">${campo("detalle.operacion", t(`operaciones.${acto.operacion}`))}
        ${campo("detalle.ambito", alcance)}${campo("detalle.desde", fecha(vigencia?.vigente_desde))}${campo("detalle.hasta", fecha(vigencia?.vigente_hasta))}${campo("detalle.motivo_corto", acto.motivos[motivos[indice]].etiqueta)}</dl></li>`;
      }).join("")}</ul>
      <p>${tx(decision.sensible ? "revision.doble" : "revision.efecto")}</p><p id="${id("dependencia")}" class="texto-secundario">${tx(disponible ? "revision.verificar" : decision.seleccion.length > 1 && !decision.motivoComun ? "revision.motivo_lote" : decision.seleccion.length > 1 ? "revision.sin_lote" : "revision.sin_conexion")}</p>
      <div id="${id("resultado")}" tabindex="-1" role="status" aria-live="polite"></div><div class="acciones-paso">${boton("corregir", "revision.corregir", `id="${id("corregir")}"`)}${boton("confirmar", decision.sensible ? "revision.proponer" : "revision.confirmar", `id="${id("confirmar")}" aria-describedby="${id("dependencia")}" ${disponible ? "" : "disabled"}`, "boton-primario")}</div></div>`;
  }
  function resultado(resultado) {
    el("resultado").innerHTML = `<h4>${tx(resultado.tipo === "propuesta" ? "resultado.propuesta" : "resultado.confirmado")}</h4><p>${tx(resultado.tipo === "propuesta" ? "resultado.espera" : "resultado.efecto")}</p>
      <dl class="resumen-expediente">${campo(resultado.tipo === "propuesta" ? "resultado.caduca" : "resultado.fecha", fecha(resultado.fecha))}</dl>${tecnico(resultado.referencia)}`;
  }
  return { pantalla, filtros, activos, tabla, catalogo, ficha, opciones, revision, resultado, el, t, tx };
}
