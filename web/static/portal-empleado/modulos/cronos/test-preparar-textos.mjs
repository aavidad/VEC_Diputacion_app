import { prepararTextosCronos } from "./preparar-textos.js";

await Promise.all(["jornada", "permisos", "resolucion", "notificaciones"]
  .map((pantalla) => prepararTextosCronos({ pantalla })));
