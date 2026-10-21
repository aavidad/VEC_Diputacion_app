# F1-145: consulta de documentos ante conflictos de serialización

La ficha del expediente de Contratación temporal usa
`POST /api/vec/documentos/expedientes/consultas` para listar sus documentos.
El repositorio repite la transacción completa si PostgreSQL devuelve 40001
o 40P01. Reutiliza `internal/shared/postgresql`: hasta 30 intentos, con espera
aleatoria y límite por el contexto de la petición.

Cada intento abre una transacción SERIALIZABLE nueva, aplica los ajustes,
ejecuta la misma fachada con la misma preimagen y autoridad, y confirma.
El intento abortado se cierra antes del siguiente. El consumo de autoridad
y la auditoría quedan dentro de la transacción. Una denegación, un bloqueo
55P03 o una confirmación con resultado ambiguo no provocan otro intento.
El agotamiento devuelve `ErrRepositorioNoDisponible`, que conserva
`ports.ErrCapacidadNoDisponible`; la frontera responde 503 catalogado.
Ningún error PostgreSQL llega al consumidor.

El registro de denegación de frontera repite su llamada completa, que es una
transacción implícita. Conserva la orden y la correlación originales. Solo
repite abortos acreditados por PostgreSQL; nunca una pérdida de conexión.
Si no puede registrar, devuelve `ErrFronteraNoDisponible` y falla cerrado.

El registro común de decisiones denegadas de contexto de actor ya usaba esa
política. La prueba añadida confirma la recuperación y el agotamiento con la
misma decisión, motivo y versiones. No se modifica el núcleo de autorización.
No se atribuye el 503 observado en la copia de T exclusivamente a los huecos
documentales: el registro común ya estaba protegido y sigue limitado por el
contexto de cada petición.

## Activación

CONFIG NUEVA: ninguna. SQL a instalar: ninguno.
Compilar y arrancar la candidata con la configuración documental vigente.
No hay datos, actos ni ajustes de arranque nuevos. El montaje opcional de
Documentos conserva su comportamiento cuando falta su configuración.

Comprobación pendiente en la copia de dirección: abrir la ficha de un
expediente y su apartado Documentos con un perfil autorizado, consultar,
reiniciar por la fábrica y repetir la lectura. Los permisos siguen siendo
necesarios; este cambio no habilita accesos denegados.

## Evidencia focal

Se han ejecutado con éxito, también con `-race`, los casos
`TestConsultaDocumentos*`, `TestDenegacionFronteraDocumentos*`,
`TestDenegacionContextoActorV3*`, `TestTransaccion*`,
`TestClasificarErrorSQL*` y las pruebas de traducción HTTP relacionadas.
40001 en el primer intento se recupera en el segundo. El persistente agota
los intentos y produce el 503 catalogado, sin cuerpo SQL ni un 500.
`go vet -p 6` de los tres paquetes afectados termina sin avisos.

Semgrep local: 42 reglas Go, tres archivos de producción cambiados, cero
hallazgos. GoSec del paquete documental: cero hallazgos en los archivos
cambiados; ocho avisos G115 en archivos ajenos al cambio. No se han añadido
exclusiones. La revisión focal del cambio comprueba aborto antes de repetir,
conservación de autoridad, ausencia de detalle SQL y fallo cerrado al agotar.

No se ha creado otra copia ni repetido el recorrido completo. Las pruebas
de PostgreSQL que requieren una base configurada no se han ejecutado en
local. La CI ejecuta la puerta completa; dirección hace la revisión
independiente de la cabeza final antes de integrar.
