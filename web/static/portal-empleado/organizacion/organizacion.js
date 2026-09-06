export const API_ORGANIZACION = "/api/vec/contratacion-temporal/organizacion";
export const API_CAMBIOS = API_ORGANIZACION + "/cambios";
export const FUENTE_RPT =
  "https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/a-informacion-institucional-y-organizativa/a3-personal/relacion-de-Puestos-de-trabajo/";
export const ESQUEMA_ORGANIZACION = "personal.estructura_organizativa.v1";
export const LIMITE_UNIDADES = 1000;
export const LIMITE_RESPUESTA = 512 * 1024;
export const TEXTOS = Object.freeze({
  eyebrow: "Organización de referencia · Preparación",
  title: "Organización de referencia en preparación",
  intro:
    "Consulta y mantenimiento de la estructura organizativa de referencia para Recursos Humanos.",
  back: "Volver al Portal del Empleado",
  provenanceDetail: "Procedencia y alcance de los datos",
  traceability: "Versión y trazabilidad del catálogo",
  noticeTitle: "Procedencia del catálogo",
  notice:
    "La edición prepara un cambio de catálogo; no acredita ocupantes, dependencia funcional ni permisos de ratificación.",
  catalogId: "Catálogo",
  version: "Versión",
  revision: "Revisión",
  fingerprint: "Huella SHA-256",
  status: "Estado",
  tableTitle: "Unidades organizativas",
  filterText: "Filtrar por texto",
  filterType: "Filtrar por tipo",
  allTypes: "Todos los tipos",
  delegacion: "Agrupación",
  centro: "Centro",
  puesto: "Puesto de responsabilidad",
  tableCaption: "Unidades de la versión consultada",
  type: "Tipo",
  code: "Código / clave",
  name: "Denominación",
  adscription: "Adscripción",
  page: "Página PDF",
  actions: "Acciones",
  newUnit: "Nueva unidad",
  edit: "Editar",
  localChange: "Cambio local",
  sourceLabel: "Fuente declarada:",
  sourceLink: "Consultar transparencia",
  note: "Las claves nuevas son técnicas y no identifican personas ni inventan códigos oficiales.",
  loading: "Cargando organización…",
  error: "No se pudo cargar la organización.",
  retry: "Reintentar",
  empty: "No hay unidades que coincidan con los filtros.",
  count: "{visible} de {total} unidades",
  draft: "Borrador",
  editorTitle: "Preparar alta o edición",
  editorHelper:
    "Este formulario guarda nombre, tipo y adscripción. Revisa el resumen antes de confirmar; no habilita ratificación.",
  label: "Denominación",
  parent: "Adscripción (padre)",
  noParent: "Sin adscripción",
  reason: "Motivo",
  cancel: "Cancelar",
  review: "Revisar cambio",
  confirm: "Confirmar cambio",
  saving: "Guardando…",
  retryExact: "Reintentar el mismo cambio",
  reloadReview: "Recargar y revisar de nuevo",
  conflict:
    "El catálogo cambió. El formulario se conserva; recarga antes de confirmar.",
  uncertain:
    "No se pudo saber si el cambio quedó registrado. Conserva la misma clave y cuerpo para reintentar.",
  saved: "Cambio registrado",
  receiptRef: "Referencia",
  receiptDate: "Fecha",
  reloadError: "El recibo se conserva, pero no se pudo recargar el catálogo.",
  rejected: "Cambio rechazado sin guardar. Revisa los datos, la adscripción y tus permisos antes de volver a confirmar.",
});
const TYPES = new Set(["delegacion", "centro", "puesto_responsabilidad"]);
const esc = (v) =>
  String(v ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
export function validarOrganizacion(payload, bytes = 0) {
  const d = payload?.data;
  const edicionHabilitada = d?.edicion_habilitada === true;
  if (
    bytes > LIMITE_RESPUESTA ||
    !d ||
    d.esquema !== ESQUEMA_ORGANIZACION ||
    d.estado !== "borrador" ||
    !Number.isInteger(d.catalogo_version) ||
    d.catalogo_version < 1 ||
    (edicionHabilitada &&
      (!Number.isInteger(d.catalogo_revision) || d.catalogo_revision < 1)) ||
    typeof d.catalogo_id !== "string" ||
    typeof d.catalogo_huella_sha256 !== "string" ||
    !/^[a-f0-9]{64}$/u.test(d.catalogo_huella_sha256) ||
    typeof d.fuente_ref !== "string" ||
    typeof d.descripcion !== "string" ||
    !Array.isArray(d.unidades) ||
    d.unidades.length > LIMITE_UNIDADES
  )
    throw new Error("respuesta de organización no válida");
  return {
    ...d,
    edicion_habilitada: edicionHabilitada,
    unidades: d.unidades.map((u) => {
      if (
        !u ||
        typeof u.clave !== "string" ||
        typeof u.etiqueta !== "string" ||
        !TYPES.has(u.tipo) ||
        (u.adscripcion_clave !== undefined &&
          typeof u.adscripcion_clave !== "string") ||
        (u.modificada_localmente !== undefined &&
          typeof u.modificada_localmente !== "boolean")
      )
        throw new Error("unidad organizativa no válida");
      return u;
    }),
  };
}
async function leerJSON(r) {
  if (!r.body?.getReader) return r.json();
  const reader = r.body.getReader(),
    chunks = [];
  let total = 0;
  while (true) {
    const p = await reader.read();
    if (p.done) break;
    total += p.value.byteLength;
    if (total > LIMITE_RESPUESTA) {
      await reader.cancel();
      throw Error("respuesta demasiado grande");
    }
    chunks.push(p.value);
  }
  const bytes = new Uint8Array(total);
  let o = 0;
  for (const c of chunks) {
    bytes.set(c, o);
    o += c.length;
  }
  return JSON.parse(new TextDecoder().decode(bytes));
}
export function crearCliente(fetchImpl = globalThis.fetch, timeoutMs = 10000) {
  const conTimeout = async (url, options, procesar) => {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
      return await procesar(await fetchImpl(url, { ...options, signal: controller.signal }));
    } finally {
      clearTimeout(timer);
    }
  };
  return {
    async obtener() {
      return conTimeout(API_ORGANIZACION, {
        credentials: "same-origin",
        redirect: "error",
        cache: "no-store",
      }, async (r) => {
        if (!r.ok) throw Error("HTTP " + r.status);
        return validarOrganizacion(await leerJSON(r));
      });
    },
    async guardar(body) {
      let r, d;
      try {
        ({ r, d } = await conTimeout(API_CAMBIOS, {
          method: "POST",
          credentials: "same-origin",
          redirect: "error",
          cache: "no-store",
          headers: { "content-type": "application/json" },
          body: JSON.stringify(body),
        }, async (r) => ({
          r,
          d: [400, 401, 403, 405, 409].includes(r.status) ? undefined : await leerJSON(r),
        })));
      } catch {
        throw Object.assign(Error("respuesta incierta"), {
          incierto: true,
          body,
        });
      }
      if (r.status === 409)
        throw Object.assign(Error("conflicto"), { conflicto: true });
      if ([400, 401, 403, 405].includes(r.status))
        throw Object.assign(Error("cambio rechazado"), { rechazado: true });
      if (!r.ok)
        throw Object.assign(Error("HTTP " + r.status), {
          incierto: true,
          body,
        });
      const x = d?.data;
      if (
        !x ||
        typeof x.recibo_ref !== "string" ||
        typeof x.registrado_en !== "string" ||
        !Number.isInteger(x.catalogo_version) ||
        !Number.isInteger(x.catalogo_revision) ||
        typeof x.huella_anterior !== "string" ||
        typeof x.huella_posterior !== "string" ||
        x.catalogo_version !== body.catalogo_version ||
        x.catalogo_revision !== body.catalogo_revision + 1 ||
        x.huella_anterior !== body.huella_esperada ||
        !/^[a-f0-9]{64}$/u.test(x.huella_posterior) ||
        !x.recibo_ref.trim() || !Number.isFinite(Date.parse(x.registrado_en)) ||
        !["registrado", "replay_confirmado"].includes(x.estado_local)
      )
        throw Object.assign(Error("respuesta de cambio no válida"), {
          incierto: true,
          body,
        });
      return x;
    },
  };
}
const tipoTexto = (t) =>
  ({
    delegacion: TEXTOS.delegacion,
    centro: TEXTOS.centro,
    puesto_responsabilidad: TEXTOS.puesto,
  })[t];
const normalizar = (v) =>
  String(v ?? "")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/gu, "")
    .toLocaleLowerCase("es");
export function filtrarUnidades(us, filtro, tipo) {
  const padres = new Map(us.map((u) => [u.clave, u.etiqueta])),
    n = normalizar(filtro.trim());
  return us.filter(
    (u) =>
      (!tipo || u.tipo === tipo) &&
      (!n ||
        [
          u.clave,
          u.etiqueta,
          u.adscripcion_clave,
          padres.get(u.adscripcion_clave),
        ].some((v) => normalizar(v).includes(n))),
  );
}
const congelarCuerpo = (body) =>
  Object.freeze({
    ...body,
    unidad: Object.freeze({ ...body.unidad }),
  });
export function crearEstadoFormulario() {
  let fase = "borrador",
    bodyPendiente;
  const bloqueado = () => fase === "enviando" || fase === "incierto";
  return Object.freeze({
    preparar(body) {
      if (bloqueado() || fase === "conflicto") return null;
      bodyPendiente = congelarCuerpo(body);
      fase = "revision";
      return bodyPendiente;
    },
    iniciarEnvio() {
      if (
        !bodyPendiente ||
        (fase !== "revision" && fase !== "incierto")
      )
        return null;
      fase = "enviando";
      return bodyPendiente;
    },
    marcarIncierto() {
      if (fase === "enviando") fase = "incierto";
    },
    marcarConflicto() {
      bodyPendiente = undefined;
      fase = "conflicto";
    },
    renovarRevision() {
      if (fase === "conflicto") fase = "borrador";
    },
    marcarExito() {
      bodyPendiente = undefined;
      fase = "borrador";
    },
    marcarRechazo() {
      bodyPendiente = undefined;
      fase = "borrador";
    },
    consultar() {
      return Object.freeze({ fase, bodyPendiente, bloqueado: bloqueado() });
    },
  });
}
function mostrarTextos() {
  document
    .querySelectorAll("[data-i18n]")
    .forEach((e) => (e.textContent = TEXTOS[e.dataset.i18n] ?? ""));
}
function main() {
  mostrarTextos();
  document.querySelector("#source-link").href = FUENTE_RPT;
  const state = document.querySelector("#state"),
    client = crearCliente(),
    editor = document.querySelector("#editor"),
    form = document.querySelector("#unit-form"),
    out = document.querySelector("#save-state"),
    operacion = crearEstadoFormulario();
  let data;
  const actualizarBloqueos = () => {
    const estado = operacion.consultar(),
      bloqueado = estado.bloqueado,
      conflicto = estado.fase === "conflicto";
    form
      .querySelectorAll("input, select, textarea")
      .forEach((control) => (control.disabled = bloqueado));
    form.setAttribute("aria-busy", estado.fase === "enviando" ? "true" : "false");
    document.querySelector("#review-submit").disabled = bloqueado || conflicto;
    document.querySelector("#cancel-edit").disabled = bloqueado;
    document.querySelector("#new-unit").disabled = bloqueado;
    document
      .querySelectorAll(".edit-unit")
      .forEach((button) => (button.disabled = bloqueado));
    const confirmar = document.querySelector("#confirm-change");
    if (confirmar) confirmar.disabled = bloqueado || conflicto;
    const reintentar = document.querySelector("#retry-change");
    if (reintentar) reintentar.disabled = estado.fase === "enviando";
  };
  const actualizarCatalogo = (nuevo) => {
    data = nuevo;
    document.querySelector("#catalog-id").textContent = data.catalogo_id;
    document.querySelector("#catalog-version").textContent =
      data.catalogo_version;
    document.querySelector("#catalog-hash").textContent =
      data.catalogo_huella_sha256;
    document.querySelector("#catalog-status").textContent =
      TEXTOS.draft +
      (Number.isInteger(data.catalogo_revision)
        ? " · " + TEXTOS.revision + " " + data.catalogo_revision
        : "");
    document.querySelector("#provenance").textContent = data.descripcion;
    document.querySelector("#source-ref").textContent = data.fuente_ref;
    document.querySelector("#new-unit").hidden = !data.edicion_habilitada;
  };
  const render = () => {
    const padres = new Map(data.unidades.map((u) => [u.clave, u.etiqueta])),
      vs = filtrarUnidades(
        data.unidades,
        document.querySelector("#filter-text").value,
        document.querySelector("#filter-type").value,
      );
    document.querySelector("#result-count").textContent = TEXTOS.count
      .replace("{visible}", vs.length)
      .replace("{total}", data.unidades.length);
    document.querySelector("#rows").innerHTML = vs
      .map(
        (u) =>
          `<tr><td>${esc(tipoTexto(u.tipo))}</td><td>${esc(u.codigo_fuente ?? u.clave)}</td><td>${esc(u.etiqueta)}${u.modificada_localmente ? `<br><span class="org-local-change">${TEXTOS.localChange}</span>` : ""}</td><td>${esc(padres.get(u.adscripcion_clave) ?? "—")}</td><td>${esc(u.pagina_fuente ?? "—")}</td><td>${data.edicion_habilitada ? `<button type="button" class="edit-unit" data-key="${esc(u.clave)}">${TEXTOS.edit}</button>` : "—"}</td></tr>`,
      )
      .join("");
    document.querySelector("#table-wrap").hidden = false;
    state.hidden = vs.length > 0;
    state.textContent = vs.length ? "" : TEXTOS.empty;
    document
      .querySelectorAll(".edit-unit")
      .forEach((b) => (b.onclick = () => abrir(b.dataset.key)));
    actualizarBloqueos();
  };
  const abrir = (key) => {
    if (operacion.consultar().bloqueado) return;
    const u = data.unidades.find((x) => x.clave === key);
    document.querySelector("#unit-key").value = key;
    document.querySelector("#unit-label").value = u?.etiqueta ?? "";
    document.querySelector("#unit-type").value = u?.tipo ?? "centro";
    document.querySelector("#unit-reason").value = "";
    const p = document.querySelector("#unit-parent");
    p.innerHTML =
      `<option value="">${TEXTOS.noParent}</option>` +
      data.unidades
        .filter((x) => x.clave !== key)
        .map(
          (x) => `<option value="${esc(x.clave)}">${esc(x.etiqueta)}</option>`,
        )
        .join("");
    p.value = u?.adscripcion_clave ?? "";
    document.querySelector("#review").hidden = true;
    editor.hidden = false;
    editor.scrollIntoView({ block: "start" });
    document.querySelector("#unit-label").focus();
  };
  form.onsubmit = (e) => {
    e.preventDefault();
    if (operacion.consultar().bloqueado) return;
    const key =
        document.querySelector("#unit-key").value ||
        "local-" + crypto.randomUUID(),
      u = {
        clave: key,
        etiqueta: document.querySelector("#unit-label").value.trim(),
        tipo: document.querySelector("#unit-type").value,
        adscripcion_clave:
          document.querySelector("#unit-parent").value || undefined,
      },
      motivo = document.querySelector("#unit-reason").value.trim();
    if (!u.etiqueta || !TYPES.has(u.tipo) || !motivo) return;
    const bodyPendiente = operacion.preparar({
      catalogo_version: data.catalogo_version,
      catalogo_revision: data.catalogo_revision,
      huella_esperada: data.catalogo_huella_sha256,
      clave_idempotencia: crypto.randomUUID(),
      unidad: u,
      motivo,
    });
    if (!bodyPendiente) return;
    document.querySelector("#unit-key").value = key;
    const review = document.querySelector("#review");
    review.innerHTML = `<strong>${TEXTOS.confirm}</strong><p>${esc(u.etiqueta)} · ${esc(tipoTexto(u.tipo))} · ${esc(u.adscripcion_clave || TEXTOS.noParent)}</p><p>${esc(motivo)}</p><button type="button" id="confirm-change">${TEXTOS.confirm}</button>`;
    review.hidden = false;
    actualizarBloqueos();
  };
  const mostrarRecibo = (x) => {
    out.className = "org-state success";
    out.innerHTML = `<strong>${TEXTOS.saved}</strong><p class="org-receipt">${TEXTOS.revision} ${esc(x.catalogo_revision)} · ${TEXTOS.receiptDate}: ${esc(x.registrado_en)} · ${TEXTOS.receiptRef}: ${esc(x.recibo_ref)}</p>`;
  };
  const recargarRevision = async () => {
    const button = document.querySelector("#reload-review");
    button.disabled = true;
    try {
      actualizarCatalogo(await client.obtener());
      render();
      operacion.renovarRevision();
      out.hidden = true;
    } catch {
      button.disabled = false;
      out.firstChild.textContent = TEXTOS.error + " ";
    }
    actualizarBloqueos();
  };
  const enviar = async () => {
    const bodyPendiente = operacion.iniciarEnvio();
    if (!bodyPendiente) return;
    out.hidden = false;
    out.className = "org-state";
    out.textContent = TEXTOS.saving;
    actualizarBloqueos();
    try {
      const x = await client.guardar(bodyPendiente);
      operacion.marcarExito();
      document.querySelector("#review").hidden = true;
      form.reset();
      document.querySelector("#unit-key").value = "";
      mostrarRecibo(x);
      actualizarBloqueos();
      try {
        actualizarCatalogo(await client.obtener());
        render();
      } catch {
        out.insertAdjacentText("beforeend", ` — ${TEXTOS.reloadError}`);
      }
    } catch (err) {
      out.className = "org-state error";
      if (err.conflicto) {
        operacion.marcarConflicto();
        document.querySelector("#review").hidden = true;
        out.innerHTML = `${esc(TEXTOS.conflict)} <button type="button" id="reload-review">${TEXTOS.reloadReview}</button>`;
        document.querySelector("#reload-review").onclick = recargarRevision;
      } else if (err.rechazado) {
        operacion.marcarRechazo();
        document.querySelector("#review").hidden = true;
        out.textContent = TEXTOS.rejected;
      } else {
        operacion.marcarIncierto();
        out.innerHTML = `${esc(TEXTOS.uncertain)} <button type="button" id="retry-change">${TEXTOS.retryExact}</button>`;
        document.querySelector("#retry-change").onclick = enviar;
      }
      actualizarBloqueos();
    }
  };
  document.querySelector("#review").onclick = (e) => {
    if (e.target.id === "confirm-change") enviar();
  };
  document.querySelector("#cancel-edit").onclick = () => {
    if (operacion.consultar().bloqueado) return;
    editor.hidden = true;
  };
  document.querySelector("#new-unit").onclick = () => {
    if (operacion.consultar().bloqueado) return;
    form.reset();
    abrir("");
  };
  document.querySelector("#filter-text").oninput = render;
  document.querySelector("#filter-type").onchange = render;
  const cargar = async () => {
    state.hidden = false;
    state.className = "org-state";
    state.textContent = TEXTOS.loading;
    try {
      actualizarCatalogo(await client.obtener());
      editor.hidden = true;
      render();
    } catch (err) {
      state.className = "org-state error";
      state.innerHTML =
        esc(TEXTOS.error) +
        ` <button type="button" id="retry">${TEXTOS.retry}</button>`;
      document.querySelector("#retry").onclick = cargar;
    }
  };
  cargar();
}
if (typeof document !== "undefined") main();
