# F1-141: recuperar un llamamiento pendiente

El corte recupera una intención singular pendiente con la clave y versión
originales cuando está montada la consulta de estado de ofertas existente.
RRHH revisa y confirma «Reanudar llamamiento». Abrir la ficha solo consulta:
no envía el acto ni guarda la intención en el navegador.

La selección singular nueva sin ofertas se ha comprobado en Chrome, con recibo,
recarga y reinicio. La recuperación de una intención pendiente con autorización
real y la causa de los 503 anteriores siguen sin acreditarse. La integración y
el cierre corresponden a dirección.

## Alcance y activación

Se reutilizan el GET `llamamientos/seleccion/estado`, su autorización y auditoría,
el registro O6 y el POST singular. CT257 reconstruye CT225 desde el cuerpo
instalado; conserva propietario, permisos, autorización positiva y auditoría.
Añade la rama singular después de consumir la decisión y conserva el contrato
plural. Su huella es
`b8834e208a4bd0d42c77d6086bd9eec7077a76121bb8fb783beeb39333437654`.

No hay configuración nueva. CT257 va después de las migraciones de main en
`deploy/principal/lista_sql_main_tras_hz37.txt`; su prueba está en fase 1.
El arranque y los actos de la copia usan la receta pública de
`deploy/principal/LEEME_copia_local.md`, con `crear --recorrido-completo`, puertos
propios y ejecutables de verificación y firma declarados por huella.

Para las dos firmas se usó el manejador declarado por
`89a7ecb8bcddd1501873a5debb4fb777f535ca7a62f0cc73be62c927a0403b36`.
La receta, apartado 11, explica el requisito de cofirma; el manejador instalado
0.0.105 no sirve para ese recorrido. No se acredita distribución pública ni
compilador actualizado del ejecutable cedido. Solo se usaron identidades
ficticias; las firmas conservan `firma_eficaz: false`.

Sin configuración de ofertas, el GET no se monta. Su 404 catalogado conserva
la selección singular nueva. Un error de una ruta instalada, una respuesta
inválida o una intención ya retenida mantienen el bloqueo. La recuperación
tras recargar sin ofertas pasó a F1-142 por decisión de dirección de las 02:46.

El router exterior reutiliza el error catalogado para rutas ausentes. Conserva
404 y los 404/405 de las rutas montadas; aplica no-store, nosniff, CSP cerrada y
retira cabeceras de cookies, CORS y redirección. Dirección aprobó esta diferencia
el 21 de octubre a las 03:06.

## Recorrido real de la candidata 483931b0c

La copia nueva arrancó en 240,44 segundos. Chrome del sistema completó el
expediente ficticio `2026/95045`: petición, ratificación, alta, análisis, vía,
asignación e informe. Las dos firmas nativas devolvieron 201 con verificación
válida; la fiscalización favorable devolvió 201 y versión 6.

A las 06:24:14 de Madrid se ejecutó una sola selección. Devolvió 200, estado
`confirmado`, justificante y versión de llamamiento 1. No hubo reintento ni otra
selección. El GET de estado devolvió el 404 catalogado de ruta no montada.

Tras una recarga real y después del reinicio por el gestor, la ficha volvió a
mostrar «Selección confirmada, pendiente de aviso», el mismo llamamiento y el
justificante de las 06:24:14. Las respuestas GET contienen las mismas referencias
de llamamiento y recibo que el POST original; el registro conserva un solo POST.
Una primera navegación de comprobación falló con `ERR_NETWORK_CHANGED`; se
repitió únicamente la carga, sin volver a seleccionar.

Las capturas del bloque a 1440 y 390 píxeles tienen revisión visual independiente
favorable. Los anchos medidos fueron 1440/1440 y 375/390, respectivamente. Estado,
justificante, fecha y siguiente acción se leen sin truncamiento. Las capturas y
los registros permanecen en el archivo privado del equipo, fuera de Git.

## Límites del diagnóstico

| Candidata ensayada | Resultado de selección |
| --- | --- |
| a5d78c211 | 503 y un reintento 503 con el mismo cuerpo y clave |
| dcca6bc31, control de main con F1-144, sin F1-141 | Un único 200 confirmado |
| c8ef0d2c2, F1-141 con F1-144 | Un único 503; causa interna de tipo `*errors.errorString` |
| 483931b0c, F1-141 y diagnóstico ampliado | Un único 200 confirmado, con recarga y reinicio positivos |

La comparación no ha identificado una causa concreta. Entre c8 y 483 se
integraron cambios de main y tres códigos de diagnóstico; no se hizo un arreglo
causal de selección ni se alteró CT257. El cotejo Go no encontró QueryRow/Scan
nuevos en el POST. CT257 solo reconstruye la función del GET de estado; no se
encontró un llamador SQL del POST y aquel GET no estaba montado. No hubo una
prueba roja causal ni un parche preventivo de tratamiento de ausencias.

El registro reconoce ahora `pgx.ErrNoRows`, cancelación y plazo agotado por
identidad y con códigos cerrados. Si no reconoce la causa, conserva solo su tipo
Go. Su prueba focal comprueba que los mensajes no se registren. El resultado 200
no generó un diagnóstico de error y no explica retrospectivamente los 503.

## Pruebas y revisiones

Las focales cubren recarga de intención pendiente sin POST automático, misma
clave y versión al reanudar, respuestas ambiguas, denegación y distinción entre
404 de router y de expediente. Las pruebas de interfaz con respuestas
controladas no acreditan recuperación pendiente con autorización real.

Las dos revisiones independientes sobre `2c1a585dd` ensayaron cada una 140
migraciones, 24 casos antes y después de reiniciar y dos concurrencias. CT257
conserva los mismos bytes. Main añadió CT253 y CT256; ahora se registran 27 casos.
Pasan el contador focal y cinco casos web de integración de aceptación y
recuperación. La línea base refleja las 1388 líneas reales del formulario.

El diagnóstico tiene prueba focal de privacidad y revisión sensible favorable
sobre `483931b0c`; la integración visual conserva su revisión sobre `3411607f6`.
Semgrep local no detecta hallazgos. GoSec señala G706 en el registro estructurado,
revisado como falso positivo por usar códigos cerrados o tipos Go, sin suprimir
la regla; los otros dos avisos pertenecen a archivos sin cambios. La CI de la
cabeza documental final debe completarse antes de dar la entrega por lista.
