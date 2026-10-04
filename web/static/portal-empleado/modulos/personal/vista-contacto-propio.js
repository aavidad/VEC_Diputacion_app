import { crearClienteCorreos, direccionCorreoAdmisible, MAX_CORREOS } from "../../../comun/correos-propios.js?v=20260929-correos-508b-v1";
import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import { ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20261001-ct-a-i18n-v1";

const textos = await cargarTextos("personal-contacto");
const t = (clave) => textos.traducir(clave);
const nodo = (d, tipo, texto) => { const n = d.createElement(tipo); if (texto !== undefined) n.textContent = texto; return n; };

function validar(datos) {
  if (!Number.isSafeInteger(datos?.version) || datos.version < 0 || !Array.isArray(datos.correos) || datos.correos.length > MAX_CORREOS) throw new TypeError("contacto_propio_no_valido");
  const referencias = new Set(); let activos = 0;
  for (const correo of datos.correos) {
    if (!direccionCorreoAdmisible(correo.direccion) || !/^correo:[0-9a-f]{32}$/u.test(correo.correo_ref) || referencias.has(correo.correo_ref) ||
      !["pendiente", "verificado"].includes(correo.estado) || typeof correo.activo !== "boolean" || (correo.activo && correo.estado !== "verificado") ||
      typeof correo.creado_utc !== "string" || !Number.isFinite(Date.parse(correo.creado_utc))) throw new TypeError("contacto_propio_no_valido");
    referencias.add(correo.correo_ref); if (correo.activo) activos += 1;
  }
  if (activos > 1) throw new TypeError("contacto_propio_no_valido");
}

/** Usuarios resuelve el titular y audita cada consulta. Personal sólo presenta su respuesta. */
export function montarVistaContactoPropio({ raiz, fetchImpl, cliente, abrirCorreos, anunciar = () => {}, registrarDesmontar, alCaducarSesion = () => {} }) {
  const d = raiz.ownerDocument;
  const fuente = cliente ?? crearClienteCorreos({ ruta: "/api/vec/usuarios/mis-correos", fetchImpl });
  let activa = true, secuencia = 0, controlador;
  const panel = nodo(d, "section"); panel.className = "panel personal-ficha-panel personal-ficha-panel-ancho";
  const cabecera = nodo(d, "header"); cabecera.className = "cabecera-panel"; cabecera.append(nodo(d, "h3", t("titulo")));
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel";
  const contenido = nodo(d, "div");
  const estado = nodo(d, "p"); estado.setAttribute("role", "status"); estado.dataset.personalContactoEstado = "";
  const acciones = nodo(d, "div"); acciones.className = "acciones-fila";
  const actualizar = nodo(d, "button", t("actualizar")); actualizar.type = "button"; actualizar.className = "boton-secundario"; actualizar.dataset.personalContactoActualizar = "";
  acciones.append(actualizar);
  if (typeof abrirCorreos === "function") {
    const gestionar = nodo(d, "button", t("gestionar")); gestionar.type = "button"; gestionar.className = "boton-primario";
    gestionar.addEventListener("click", abrirCorreos); acciones.append(gestionar);
  }
  cuerpo.append(estado, contenido, acciones); panel.append(cabecera, cuerpo); raiz.append(panel);
  const desmontar = () => { if (!activa) return; activa = false; secuencia += 1; controlador?.abort(); contenido.replaceChildren(); panel.replaceChildren(); panel.remove?.(); };
  registrarDesmontar?.(desmontar);
  async function cargar() {
    if (!activa) return;
    const restaurarFoco = d.activeElement === actualizar;
    controlador?.abort(); const vuelo = new AbortController(); controlador = vuelo; const turno = ++secuencia;
    contenido.replaceChildren(); estado.setAttribute("role", "status"); estado.textContent = t("cargando"); actualizar.disabled = true; panel.setAttribute("aria-busy", "true");
    try {
      const datos = await fuente.consultar({ signal: vuelo.signal });
      if (!activa || turno !== secuencia || vuelo.signal.aborted) return;
      validar(datos);
      estado.textContent = t(datos.correos.length ? "fuente" : "vacio");
      if (datos.correos.length) {
        const region = nodo(d, "div"); region.className = "tabla-contenedor personal-ficha-tabla";
        region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("titulo"));
        const tabla = nodo(d, "table"); tabla.className = "tabla-datos"; tabla.append(nodo(d, "caption", t("titulo")));
        const encabezado = nodo(d, "thead"), fila = nodo(d, "tr");
        for (const clave of ["direccion", "estado", "alta"]) { const th = nodo(d, "th", t(clave)); th.setAttribute("scope", "col"); fila.append(th); }
        encabezado.append(fila); const filas = nodo(d, "tbody");
        for (const correo of datos.correos) {
          const fila = nodo(d, "tr");
          const fecha = new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeZone: ZONA_HORARIA_PORTAL }).format(new Date(correo.creado_utc));
          fila.append(nodo(d, "td", correo.direccion), nodo(d, "td", t(correo.activo ? "activo" : correo.estado)), nodo(d, "td", fecha)); filas.append(fila);
        }
        tabla.append(encabezado, filas); region.append(tabla); contenido.append(region);
      }
    } catch (error) {
      if (!activa || turno !== secuencia || vuelo.signal.aborted) return;
      contenido.replaceChildren(); estado.setAttribute("role", "alert");
      const clave = error?.estado === 401 ? "sesion" : error?.estado === 403 ? "denegado" : error?.estado === 404 ? "no_configurado" : "error";
      estado.textContent = t(clave); anunciar(estado.textContent, "error");
      if (error?.estado === 401) alCaducarSesion();
    } finally {
      if (activa && turno === secuencia) {
        actualizar.disabled = false; panel.setAttribute("aria-busy", "false");
        if (restaurarFoco && (d.activeElement === actualizar || d.activeElement === d.body) && (typeof d.hasFocus !== "function" || d.hasFocus())) actualizar.focus?.();
      }
    }
  }
  actualizar.addEventListener("click", cargar); void cargar();
  return Object.freeze({ desmontar });
}
