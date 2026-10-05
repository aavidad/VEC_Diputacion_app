# Replay que caduca mientras espera la barrera

Vector focal para el clon coordinado. No requiere DOWN, alteración de funciones
ni permisos nuevos. El publicador y el LOGIN técnico usan su configuración
positiva real, fijada fuera de Git antes de la primera publicación.

Ejecutar dos casos independientes: configuración caducada y plan caducado.
En el primero, el plan cubre toda la prueba; en el segundo, la configuración
cubre toda la prueba. El vencimiento pertenece a la configuración/plan originales:
no se cambia después de confirmar el recibo, pues cambiaría su compromiso.

1. Publicar la unidad mientras ambas ventanas son válidas y conservar el recibo
   confirmado. Cotejar antes de la espera que el mismo plan/LOGIN/configuración
   recupera exactamente ese recibo.
2. Una conexión DBA del clon abre transacción y bloquea la fila existente:

   ```sql
   SELECT generacion
   FROM vec_personal.control_unidad_bootstrap_admin_v1
   WHERE singleton
   FOR UPDATE;
   ```

   No modifica la generación, configuración, historia ni recibos.
3. La conexión real del operador, en SERIALIZABLE/UTC y sin SET ROLE, llama al
   inicializador con los tres argumentos originales antes del vencimiento.
   Dirección comprueba que la llamada ha alcanzado la espera de esa fila;
   una llamada que comienza ya caducada no acredita este caso.
4. Liberar la barrera después del vencimiento y antes del timeout de la llamada.
   La espera debe ser breve; usar la ventana de configuración con microsegundos
   facilita el primer caso. No añadir esperas largas ni excepciones a guardas.
5. Confirmar el sobre por COMMIT: `estado=denegado`,
   `codigo=unidad_rechazada`, `recibo=null`, `replay=false`, con coordenadas
   nuevas de `auditoria_intento`.
6. Desde la conexión DBA, cotejar que la fila, generación y recibo original son
   idénticos y que sólo hay un intento común nuevo de la acción de Personal.
   No debe aparecer otra confirmación, fuente, unidad ni referencia revertida.

La prueba distingue la validación inicial de la revalidación después de la
espera. El cambio exige en el camino de replay las mismas comprobaciones finales
de configuración y caducidad del plan que ya tenía el alta. La primera lectura
no basta para autorizar un resultado después de un bloqueo.

Este vector está preparado, no ejecutado por el productor. Dirección ejecuta
el ensayo en su único clon desechable y comunica el resultado sobre el SHA final.
