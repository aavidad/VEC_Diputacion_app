/* Aviso de arranque: no depende del grafo de módulos ni de sus catálogos. */
(() => {
  const raiz = document.getElementById("espacio-trabajo");
  if (!raiz || raiz.childElementCount > 0) return;

  const aviso = document.createElement("section");
  aviso.className = "panel";
  aviso.setAttribute("role", "status");
  aviso.setAttribute("aria-live", "polite");
  aviso.dataset.portalArranqueAviso = "";
  const cabecera = document.createElement("div");
  cabecera.className = "cabecera-panel";
  const titulo = document.createElement("h2");
  cabecera.append(titulo);
  const cuerpo = document.createElement("div");
  cuerpo.className = "cuerpo-panel";
  const detalle = document.createElement("p");
  const reintentar = document.createElement("button");
  reintentar.type = "button";
  reintentar.className = "boton-primario";
  reintentar.hidden = true;
  reintentar.addEventListener("click", () => location.reload());
  cuerpo.append(detalle, reintentar);
  aviso.append(cabecera, cuerpo);
  raiz.append(aviso);

  let fallo = false;
  let textos = null;
  const controladores = new Set();
  const vigente = () => raiz.contains(aviso);
  const observador = new MutationObserver(() => {
    if (vigente()) return;
    clearTimeout(plazo);
    for (const controlador of controladores) controlador.abort();
    observador.disconnect();
  });
  observador.observe(raiz, { childList: true });

  function pintar() {
    if (!textos || !vigente()) return;
    titulo.textContent = fallo ? textos.titulo_error : textos.titulo;
    detalle.textContent = fallo ? textos.error : textos.cargando;
    reintentar.textContent = textos.reintentar;
    reintentar.hidden = !fallo;
  }

  function mostrarError(codigo) {
    if (fallo || !vigente()) return;
    fallo = true;
    // El fallo puede incluir URL o detalles de identidad: solo sale un código fijo.
    console.error("portal.arranque.fallido", { codigo });
    pintar();
  }

  window.addEventListener("error", (evento) => {
    if (evento.target === window) {
      mostrarError("evaluacion_portal_no_disponible");
      return;
    }
    const script = evento.target;
    if (script?.tagName !== "SCRIPT" || script.type !== "module") return;
    try {
      if (new URL(script.src, location.href).pathname === "/portal-empleado/portal.js") {
        mostrarError("importacion_portal_no_disponible");
      }
    } catch { /* Una URL ajena o defectuosa no se registra. */ }
  }, true);
  window.addEventListener("unhandledrejection", () => mostrarError("promesa_arranque_no_disponible"));
  const plazo = setTimeout(() => mostrarError("arranque_sin_vista"), 12_000);

  const opcionesLectura = Object.freeze({
    method: "GET", credentials: "same-origin", mode: "same-origin",
    redirect: "error", cache: "no-store", referrerPolicy: "no-referrer",
  });
  const patronIdioma = /^[a-z]{2,8}$/u;
  const codigoSeguro = (valor) => {
    const codigo = String(valor ?? "").toLowerCase();
    return patronIdioma.test(codigo) ? codigo : "";
  };

  async function leerJSON(ruta) {
    const controlador = new AbortController();
    controladores.add(controlador);
    let temporizador;
    const vencimiento = new Promise((_, rechazar) => {
      temporizador = setTimeout(() => {
        controlador.abort();
        rechazar(new Error("lectura de arranque agotada"));
      }, 3_000);
    });
    const lectura = async () => {
      const respuesta = await fetch(ruta, { ...opcionesLectura, signal: controlador.signal });
      if (!respuesta.ok) throw new Error("recurso de arranque no disponible");
      const maximo = 128 * 1024;
      if (!respuesta.body?.getReader) {
        const texto = await respuesta.text();
        if (new TextEncoder().encode(texto).byteLength > maximo) throw new Error("catálogo de arranque excesivo");
        return JSON.parse(texto);
      }
      const lector = respuesta.body.getReader();
      const partes = [];
      let tamano = 0;
      while (true) {
        const { done, value } = await lector.read();
        if (done) break;
        tamano += value.byteLength;
        if (tamano > maximo) {
          void lector.cancel().catch(() => {});
          throw new Error("catálogo de arranque excesivo");
        }
        partes.push(value);
      }
      const bytes = new Uint8Array(tamano);
      let posicion = 0;
      for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
      return JSON.parse(new TextDecoder().decode(bytes));
    };
    try { return await Promise.race([lectura(), vencimiento]); }
    finally { clearTimeout(temporizador); controladores.delete(controlador); }
  }

  async function cargarIndice() {
    try {
      const indice = await leerJSON("/textos/idiomas.json");
      const admitidos = indice.idiomas?.map((item) => codigoSeguro(item?.codigo)).filter(Boolean);
      const porDefecto = codigoSeguro(indice.por_defecto);
      if (!Array.isArray(admitidos) || !admitidos.includes(porDefecto)) throw new Error("índice de idiomas no válido");
      const solicitado = codigoSeguro(new URL(location.href).searchParams.get("lang"));
      const preferencias = indice.seguir_navegador === true ? navigator.languages ?? [navigator.language] : [];
      const preferido = preferencias.map((valor) => codigoSeguro(String(valor).split("-", 1)[0]))
        .find((valor) => admitidos.includes(valor));
      return { idioma: admitidos.includes(solicitado) ? solicitado : preferido || porDefecto, porDefecto };
    } catch {
      // Si falla el índice, el idioma declarado por el documento es el respaldo.
      const idioma = codigoSeguro(document.documentElement.lang.split("-", 1)[0]);
      return { idioma, porDefecto: idioma };
    }
  }

  function validarTextos(valores) {
    const claves = ["titulo", "cargando", "titulo_error", "error", "reintentar"];
    if (!valores || typeof valores !== "object" || Array.isArray(valores)
      || claves.some((clave) => typeof valores[clave] !== "string"
        || !valores[clave].trim() || valores[clave].length > 240)) {
      throw new Error("catálogo de arranque no válido");
    }
    return Object.fromEntries(claves.map((clave) => [clave, valores[clave]]));
  }

  async function cargarTextosArranque() {
    const { idioma, porDefecto } = await cargarIndice();
    if (!vigente()) return;
    const disponibles = [...new Set([idioma, porDefecto].filter(Boolean))];
    for (const candidato of disponibles) {
      if (!vigente()) return;
      try {
        const valores = validarTextos(await leerJSON(`/textos/${candidato}/portal-arranque.json`));
        if (!vigente()) return;
        textos = valores;
        document.documentElement.lang = candidato;
        pintar();
        break;
      } catch { /* Se prueba el idioma de respaldo sin conservar el fallo. */ }
    }
    if (!textos) {
      mostrarError("catalogo_arranque_no_disponible");
      return;
    }
    // La traducción del shell es opcional y nunca retrasa el aviso autónomo.
    try {
      const idiomaPortal = await import("./portal-idioma.js?v=20261007-pantallas-textos-final-v1");
      if (vigente()) idiomaPortal.aplicarTextosPortal(document);
    } catch { /* El aviso autónomo conserva su texto. */ }
  }

  void cargarTextosArranque();
})();
