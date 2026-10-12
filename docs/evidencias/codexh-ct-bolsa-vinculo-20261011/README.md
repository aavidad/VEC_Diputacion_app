# Vínculo del llamamiento de Bolsa con el expediente CT

Al abrir un llamamiento desde la ficha CT, el enlace conserva la versión del expediente. Cuando Bolsa devuelve la emisión confirmada, el portal registra el vínculo con el expediente usando el cliente CT existente. Si ese segundo paso falla, conserva el recibo y ofrece reintentar el vínculo sin emitir otro llamamiento.

La identidad de la sesión se vuelve a comprobar antes de emitir y antes de vincular. El reintento conserva el comando, la clave y la identidad iniciales. Salir o cancelar invalida el intento: una respuesta tardía conserva el recibo y puede actualizar la vista del mismo flujo expresamente reabierto, pero no inicia por sí sola otro acto. Mientras queda una petición de vínculo en curso, la pantalla muestra espera y no ofrece un botón que no pueda actuar.

## Código reutilizado

Se reutilizan el cliente `cliente-http-vinculo-bolsa.js`, las rutas CT de vínculos, la emisión de Bolsa, la consulta de sesión del portal y sus recibos. Este corte añade el montaje y la continuidad del flujo. No crea SQL, API, permisos ni otra implementación de llamamientos. Las dependencias son #1019 y la cadena de fichas CT que termina en #999.

## Comprobaciones

El productor verificó 105 pruebas focales; la corrección de recuperación tiene 21 casos de integración, incluidos emisión tardía, cancelación, reapertura y respuesta tardía del vínculo. Tras incorporar las dependencias, la suite web completa pasó 3820/3820 en 5,82 s. Las guardas de catálogos, dependencias de superficies y manifiestos comprobaron el árbol integrado; la cartografía pública se preparó con el script existente y su SHA256 fijado.

La revisión de seguridad de 977040194 fue favorable. La revisión independiente de usabilidad detectó dos fallos de recuperación en Chrome a 1440 y 390 px; se corrigieron en 503c3930e. El dictamen sobre la cabeza final se registra en la PR, junto con la CI exacta. Las comprobaciones de navegador usan el presentador y controlador reales con transporte controlado: no acreditan instalación SQL ni un recorrido nominal en PostgreSQL.

Las versiones de caché se renuevan por el grafo CT/Bolsa con el script existente. Las pruebas conservan las comprobaciones de URL única, módulos alcanzables y catálogos; sólo se actualizan sus cohortes esperadas. La huella de textos se calcula sobre todos los catálogos integrados. La línea base de tamaños refleja los cinco archivos que crecieron por el montaje, la recuperación y las pruebas del grafo, sin alterar el verificador.

## Recorrido pendiente de integración

1. Abrir desde CT un expediente con categoría y bolsa vinculadas y elegir «Abrir llamamiento en Bolsa».
2. Revisar y confirmar la emisión con una sesión nominal habilitada para ambos actos.
3. Comprobar el recibo Bolsa y el vínculo CT. Ante un fallo del vínculo, reintentar y verificar que no hay otra emisión.
4. Cancelar durante la emisión o el vínculo y volver al mismo flujo; comprobar espera mientras hay petición pendiente y recuperación del recibo después.
5. Comparar recibos, auditoría e historia tras reiniciar aplicación y PostgreSQL.

Este recorrido conectado requiere las dependencias integradas y la provisión privada autorizada en un clon aislado. No se ha ejecutado sobre cidonia.

CONFIG NUEVA: ninguna propia de este corte.
