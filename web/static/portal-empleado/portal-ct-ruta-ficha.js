/** Estado de navegación de la ficha CT. La URL nunca acredita acceso ni identidad. */
const REFERENCIA = /^[A-Za-z0-9:_.-]{1,200}$/u;
const VERSION_MAXIMA = 1_000_000_000;
const PARAMETROS = Object.freeze(["expediente", "expediente_version"]);

export function leerFichaCTDeRuta(busqueda) {
  const parametros = new URLSearchParams(busqueda);
  if (!parametros.has("expediente")) return null;
  if (PARAMETROS.some((clave) => parametros.getAll(clave).length > 1)) return null;
  const referencia = parametros.get("expediente") ?? "";
  const versionTexto = parametros.get("expediente_version");
  const version = versionTexto === null ? null : Number(versionTexto);
  if (!REFERENCIA.test(referencia) || (versionTexto !== null
    && (!/^[1-9][0-9]{0,9}$/u.test(versionTexto) || !Number.isSafeInteger(version)
      || version > VERSION_MAXIMA))) return null;
  return Object.freeze({ expedienteRef: referencia, version });
}

/** El enlace antiguo puede abrirse desde la primera página; fuera de ella necesita una indicación. */
export function enlaceCTSinVersionFueraDePagina(ficha, cuadro) {
  return ficha !== null && typeof ficha === "object" && REFERENCIA.test(ficha.expedienteRef)
    && ficha.version === null && cuadro?.demostracion === false
    && Array.isArray(cuadro.expedientes)
    && !cuadro.expedientes.some((expediente) => expediente?.expediente_ref === ficha.expedienteRef);
}

export function rutaConFichaCT(ubicacion, ficha = null) {
  if (!ubicacion || typeof ubicacion.pathname !== "string" || !ubicacion.pathname.startsWith("/")
    || ubicacion.pathname.startsWith("//") || /[\\?#]/u.test(ubicacion.pathname)
    || typeof ubicacion.search !== "string" || typeof ubicacion.hash !== "string"
    || !/^#[a-z][a-z0-9/-]*$/u.test(ubicacion.hash)) throw new TypeError("ruta CT no válida");
  const parametros = new URLSearchParams(ubicacion.search);
  for (const clave of PARAMETROS) parametros.delete(clave);
  if (ficha !== null) {
    const { expedienteRef, version } = ficha;
    if (ubicacion.hash !== "#contratacion-temporal" || !REFERENCIA.test(expedienteRef)
      || !Number.isSafeInteger(version) || version < 1 || version > VERSION_MAXIMA) {
      throw new TypeError("ficha CT no válida");
    }
    parametros.set("expediente", expedienteRef);
    parametros.set("expediente_version", String(version));
  }
  const busqueda = parametros.toString();
  return `${ubicacion.pathname}${busqueda ? `?${busqueda}` : ""}${ubicacion.hash}`;
}
