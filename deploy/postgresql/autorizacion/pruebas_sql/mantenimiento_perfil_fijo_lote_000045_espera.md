# Revalidación real tras espera de cadena — AUT45

Vector dinámico pendiente del ensayo autorizado, sobre el fixture vigente
producido por el pipeline. No se retrofecha, cambia config instalada ni se
reescribe una asignación para provocar el caso.

1. Producir un planv2 y configuración DBA con una ventana breve y registrar
   un mantenimiento permitido. Conservar plan, SHA y recibo originales.
2. En otra sesión, bloquear la fila de control de la cadena común. Invocar
   replay con los mismos plan/SHA antes de caducar su ventana. Debe esperar
   en el append del intento permitido; no basta bloquear sólo continuidad.
3. Liberar la fila después de caducar config o plan (dos casos independientes,
   preparados por aprobación/config originales). Confirmar el envelope
   denegado mediante COMMIT, sin sustituir timestamps ni valores de entrada.
4. Comprobar resultado denegado y recibo null; no aparece intento permitido
   de esa llamada, sólo la negativa AD183 nueva. La confirmación del primer
   mantenimiento, Rol6 y sus dos asignaciones originales permanecen idénticos.
5. En un ensayo de alta equivalente, la misma espera caducada revierte todas
   las filas de efecto y confirmación nuevas. La negativa conserva sólo el
   LOGIN real y referencias técnicas, sin Persona/perfil inventados.

Capturar recuentos, huellas y metadatos de los cuatro gates antes/después:
propietario, ACL, configuración y OID permanecen; sólo cambia su source
para la compatibilidad cerrada 5/6. La prueba no reaplica SQL instalada.
