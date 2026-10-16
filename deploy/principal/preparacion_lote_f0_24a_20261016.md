# Preparación del lote de perfiles

Corte F0-24A sobre la preparación existente de A7, incorporada en las PR
#729 a #732. Reutiliza la petición, el puerto de unidades, el emisor de
autorización y la auditoría del lote. No añade rutas, permisos ni SQL.

## Activación

Instalar el binario de `vec-admin` con su configuración privada de usuarios y
de lote ya aprobada. El lector, el emisor y el registro de intentos conservan
sus conexiones nominales separadas. El fichero de lote debe declarar la
organización y las unidades que devuelve su fuente propietaria. No reutilizar
confianza vencida ni trasladar material de otra copia sin su publicación.

El cotejo de la unidad se hace al preparar, antes de emitir la autorización y
abrir la transacción de preparación. Su fallo conserva el registro nominal de
intento. La aplicación del lote mantiene sus propias comprobaciones.

## Comprobación

Las pruebas focales fijan los bytes y la huella de una petición válida,
comprueban las respuestas excepcionales del catálogo y rechazan la unidad
incorrecta antes de abrir la transacción. Una respuesta inválida se rechaza
entera, sin recortar duraciones ni ofrecer perfiles parcialmente.

Este corte no acredita un recorrido nuevo en Chrome. La activación de
Administración en la copia H está bloqueada por una confianza vencida; el
recorrido conjunto se completará con F0-23 y los formularios de F0-04/F0-05.
Sin configuración, se conserva el cierre de Administración existente.

Para retirar el cambio, volver al binario anterior con la misma configuración.
No hay datos que restaurar ni migraciones que deshacer.

## Trabajo pendiente

F0-24 no queda cerrada: la relación de AD198 con la auditoría común exige una
migración del equipo propietario del núcleo. La renovación de la configuración
y la retirada del acceso operador requieren su guion operativo. Este corte
no modifica esos contratos.

## Evidencia del corte

Dos revisiones independientes de código y una documental favorables. Pasan
las pruebas focales de dominio, adaptador y aplicación. Semgrep local analiza
los tres Go cambiados sin hallazgos ni errores; GoSec carga los 36 archivos
del paquete adaptador sin errores ni avisos, con la corrección local de carga
por directorio. El binario de Administración compila; sin configuración
rechaza el arranque en la etapa de usuarios, como antes. No se ha probado
el arranque con configuración vigente ni se da por completada la aceptación.
