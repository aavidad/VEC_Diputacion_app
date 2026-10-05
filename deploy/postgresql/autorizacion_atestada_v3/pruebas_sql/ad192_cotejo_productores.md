# Cotejo real de recuperación — AD192

Vector para Source con el LOGIN y productores IS/CA reales. Sin INSERT directo
de filas de auditoría ni configuración favorable en la migración.

1. Registrar una operación CA con su evento15 original y confirmar su efecto,
   enlace de dominio y ACK5. Conservar los valores exactos y los recuentos.
2. Abrir READ COMMITTED READ ONLY mediante la fachada CA de reconciliación.
   El cotejo AD192 del mismo frame devuelve ACK5 idéntico; todos los recuentos
   y cabeza de auditoría siguen intactos.
3. Repetir con un evento válido nuevo no existente: cero filas, sin append.
4. Cambiar un campo del frame original, conservando evento_ref: SQLSTATE23505,
   sin datos ni escrituras. Invocar el cotejo IS con acción de CA: 42501.
5. Una consulta funcional nueva usa otro evento y append normal SERIALIZABLE;
   no se presenta como recuperación del COMMIT anterior.
