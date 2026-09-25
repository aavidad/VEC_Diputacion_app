export const CLAVES_CARGA_MODULAR = Object.freeze([
  "contratacion_temporal",
  "personal",
  "cronos",
  "dietas",
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

export function consultarConLimite(consultar, controlador, limiteMs, temporizadores, operacion = "consultar contratación temporal") {
  return new Promise((resolver, rechazar) => {
    let terminada = false;
    const cancelada = () => finalizar(rechazar, new Error(`carga cancelada al ${operacion}`));
    const finalizar = (continuacion, valor) => {
      if (terminada) return;
      terminada = true;
      temporizadores.clearTimeout(temporizador);
      controlador.signal.removeEventListener("abort", cancelada);
      continuacion(valor);
    };
    const temporizador = temporizadores.setTimeout(() => {
      finalizar(rechazar, new Error(`tiempo agotado al ${operacion}`));
      controlador.abort();
    }, limiteMs);
    controlador.signal.addEventListener("abort", cancelada, { once: true });
    if (controlador.signal.aborted) {
      cancelada();
      return;
    }
    Promise.resolve()
      .then(() => {
        if (controlador.signal.aborted) throw new Error("consulta cancelada");
        return consultar({ signal: controlador.signal });
      })
      .then(
        (resultado) => finalizar(resolver, resultado),
        () => finalizar(rechazar, new Error(`no se pudo ${operacion}`)),
      );
  });
}
