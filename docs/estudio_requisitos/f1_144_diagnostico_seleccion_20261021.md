# F1-144: diagnóstico de una selección indeterminada

El registro de Contratación temporal añade `causa_interna` únicamente cuando
la selección de llamamiento queda indeterminada. Busca el centinela conocido
más interno del módulo mediante `errors.Is`; si no lo reconoce, guarda solo
el tipo Go del error más interno. El marcador del resultado público no desplaza
a una causa interna. No registra mensajes, solicitudes ni referencias nuevas.
La respuesta HTTP conserva su contenido.

## Activación y comprobaciones

No requiere SQL, configuración ni actos nuevos. Se activa al arrancar la
aplicación actualizada. Reutiliza el envoltorio de aplicación existente con
`Unwrap`; no cambia la ejecución de la selección ni sus permisos.

Tres pruebas focales nuevas cubren cadenas simples y múltiples, centinelas
profundos, un envoltorio con `Is`, una cadena de 40 niveles, una causa desconocida
junto a un marcador público más profundo y los criterios pendientes de RRHH.
Comprueban que el texto privado del error no llegue al registro. Las tres pasan.

Semgrep local: cero hallazgos. GoSec advierte G706 en el registro estructurado;
la revisión independiente lo considera falso positivo: el campo nuevo procede
de códigos cerrados o de `%T`, nunca del mensaje. No se ha suprimido la regla.
Los otros dos avisos pertenecen a archivos sin cambios.

Revisión independiente de código y privacidad favorable sobre
`dcca6bc31cc2fcd8b92b8933922d90e97bf305fc`, tras dos correcciones focales.

## Recorrido local del 21 de octubre

La receta pública `deploy/principal/copia_local.py crear --recorrido-completo`
creó una copia nueva desde esa cabeza en 241,24 segundos, con puertos propios y
los ejecutables de verificación y firma declarados por su huella. Los
certificados y actos se generaron dentro de esa copia; no se reutilizó material
de otra instancia. La copia inicial anterior se retiró sin ejecutar selecciones.

Chrome del sistema completó petición del centro, ratificación, alta, análisis,
vía de cobertura, asignación e informe para una solicitud de una persona.
Las dos firmas se confirmaron en sus ventanas nativas: ambos registros
devolvieron 201 con verificación válida. La fiscalización favorable devolvió
201 y versión 6.

A las 05:01:53 de Madrid se ejecutó una sola selección desde la ficha: devolvió
200, estado `confirmado` y recibo. La pantalla mostró «Guardado» y «Persona
llamada». No hubo reintento ni segunda selección. La aplicación se reinició con
la misma cabeza y volvió a cargar la ficha en Chrome; tampoco se seleccionó
otra vez después del reinicio.

El fallo 503 anterior de F1-141 no se reprodujo. No hubo error indeterminado y,
por tanto, no se generó `causa_interna`. Este resultado no explica aquel fallo
ni demuestra la recuperación de una intención pendiente. Dirección recibió el
resultado antes de decidir el siguiente paso. Los registros y capturas quedan
en el archivo privado del equipo, fuera de Git.
