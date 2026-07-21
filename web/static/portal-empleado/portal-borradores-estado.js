export function crearEstadoBorradores() {
  return {
    faseLista: "inicial",
    opciones: null,
    lista: null,
    errorLista: null,
    filtro: { texto: "", categoria: "" },
    cursores: [undefined],
    pagina: 0,
    modoEditor: "ninguno",
    faseEditor: "vacio",
    referenciaSeleccionada: "",
    detalle: null,
    editor: null,
    sucio: false,
    guardando: false,
    errorEditor: null,
    recibo: null,
    claveIdempotencia: "",
    conflictoRemoto: null,
    confirmarReaplicacion: false,
  };
}

/** Elimina toda proyección que pudiera sobrevivir a una revocación. */
export function limpiarEstadoBorradoresRevocado(estado, errorLista) {
  if (!estado || typeof estado !== "object") throw new TypeError("estado de borradores no válido");
  Object.assign(estado, {
    faseLista: "error",
    opciones: null,
    lista: null,
    errorLista,
    filtro: { texto: "", categoria: "" },
    cursores: [undefined],
    pagina: 0,
    modoEditor: "ninguno",
    faseEditor: "vacio",
    referenciaSeleccionada: "",
    detalle: null,
    editor: null,
    sucio: false,
    guardando: false,
    errorEditor: null,
    recibo: null,
    claveIdempotencia: "",
    conflictoRemoto: null,
    confirmarReaplicacion: false,
  });
}
