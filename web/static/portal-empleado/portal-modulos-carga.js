export const CLAVES_CARGA_MODULAR = Object.freeze([
  "contratacion_temporal",
  "personal",
  "cronos",
  "dietas",
]);

export const CLAVES_CARGA_PRESENTACION = Object.freeze([
  ...CLAVES_CARGA_MODULAR,
  "nominas", "solicitudes", "meritos", "comunicaciones",
  "documentos", "aprobaciones", "auditoria", "administracion",
]);

export const LIMITE_CARGA_MODULAR_MS = 2_000;

export function cargarModuloConLimite(cargar, clave, limiteMs, temporizadores) {
  if (typeof temporizadores?.setTimeout !== "function"
    || typeof temporizadores?.clearTimeout !== "function") {
    return Promise.reject(new TypeError("temporizadores modulares no disponibles"));
  }
  return new Promise((resolver, rechazar) => {
    let terminada = false;
    const finalizar = (continuacion, valor) => {
      if (terminada) return;
      terminada = true;
      temporizadores.clearTimeout(temporizador);
      continuacion(valor);
    };
    const temporizador = temporizadores.setTimeout(
      () => finalizar(rechazar, new Error(`tiempo agotado al cargar ${clave}`)),
      limiteMs,
    );
    Promise.resolve()
      .then(cargar)
      .then(
        (recursos) => finalizar(resolver, recursos),
        () => finalizar(rechazar, new Error(`no se pudo cargar ${clave}`)),
      );
  });
}

export function consultarConLimite(consultar, controlador, limiteMs, temporizadores) {
  return new Promise((resolver, rechazar) => {
    let terminada = false;
    const finalizar = (continuacion, valor) => {
      if (terminada) return;
      terminada = true;
      temporizadores.clearTimeout(temporizador);
      continuacion(valor);
    };
    const temporizador = temporizadores.setTimeout(() => {
      controlador.abort();
      finalizar(rechazar, new Error("tiempo agotado al consultar contratación temporal"));
    }, limiteMs);
    Promise.resolve()
      .then(() => consultar({ signal: controlador.signal }))
      .then(
        (resultado) => finalizar(resolver, resultado),
        () => finalizar(rechazar, new Error("no se pudo consultar contratación temporal")),
      );
  });
}

export async function resolverCargasModularesPresentacion(cargadores, {
  claves = CLAVES_CARGA_PRESENTACION,
  limiteMs = LIMITE_CARGA_MODULAR_MS,
  temporizadores = globalThis,
} = {}) {
  if (!Array.isArray(claves)
    || claves.some((clave) => !CLAVES_CARGA_PRESENTACION.includes(clave))
    || new Set(claves).size !== claves.length
    || !Number.isSafeInteger(limiteMs) || limiteMs < 1 || limiteMs > 10_000) {
    throw new TypeError("configuración de carga modular no válida");
  }
  const clavesSolicitadas = new Set(claves);
  const resultados = await Promise.allSettled(CLAVES_CARGA_PRESENTACION.map((clave) => {
    if (!clavesSolicitadas.has(clave)) return Promise.resolve(undefined);
    const cargar = cargadores?.[clave];
    if (typeof cargar !== "function") {
      return Promise.reject(new TypeError(`cargador modular ausente: ${clave}`));
    }
    return cargarModuloConLimite(cargar, clave, limiteMs, temporizadores);
  }));
  return Object.freeze(Object.fromEntries(CLAVES_CARGA_PRESENTACION.map((clave, indice) => {
    if (!clavesSolicitadas.has(clave)) {
      return [clave, Object.freeze({ disponible: false, estado: "denegado" })];
    }
    const resultado = resultados[indice];
    return [clave, resultado.status === "fulfilled"
      ? Object.freeze({ disponible: true, estado: "disponible", recursos: resultado.value })
      : Object.freeze({ disponible: false, estado: "no_disponible" })];
  })));
}
