import { cargarTextos } from "../../../../comun/textos.js";
import { crearClienteCategorias } from "../../../categorias-rpt/cliente.js?v=20261002-ct-fin-moad-v1";
import { crearClienteCargaConvoca } from "./cliente.js?v=20261005-b1-carga-v1";
import { montarVistaCargaConvoca } from "./vista.js?v=20261005-b1-carga-v1";

const textos = await cargarTextos("bolsa-carga-convoca");
const vista = montarVistaCargaConvoca({
  doc: document,
  cliente: crearClienteCargaConvoca(),
  categorias: crearClienteCategorias(),
  textos,
});
void vista.iniciar();
