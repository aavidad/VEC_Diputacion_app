# Contexto ADMIN nominal — CA36

CA36 prepara el registro y la recuperación de un contexto V2 para una petición
ADMIN concreta. La cuenta privilegiada, el perfil seleccionado y la sesión
deben seguir vigentes después de las últimas esperas. El vínculo IS procede
de la misma petición y se comprueba por referencia, versión y huella exactas;
buscar la última sesión de una cuenta no acredita esa relación.

La fachada CA tendrá un grupo de ejecución exclusivo. El LOGIN no heredará el
grupo general de ContextoActor ni permisos sobre sus tablas. El cuerpo medido
del núcleo V2 se extrae a dos helpers privados del propietario CA. Las entradas
generales conservan su acreditación, sus metadatos y sus ACL; ADMIN sólo entra
por fachadas que comprueban su grupo propio y el vínculo IS16 exacto antes de
llamar a los helpers. El núcleo conserva los bytes, la huella, la operación,
el recibo y el CAS. En ADMIN el alcance de proyecciones será vacío, el método
será certificado o DNIe y la garantía será alta.

El adaptador Go exige `NuevoContextoRegistradoADMINPostgreSQL` con pool CA
propio y proceso privado validado para AD192. El constructor anterior sigue
compilando, pero permanece cerrado. La composición del pool nuevo queda fuera
de este corte.

`registrar_contexto_admin_v1` usa `SERIALIZABLE`. IS16 coteja el método real
y, en ese aislamiento, comprueba el perfil de Aplicación por CA31. CA36
verifica además la selección y las versiones actuales de su propio contexto.
La recuperación de un COMMIT incierto usa `READ COMMITTED` con la operación,
recibo, vínculo y evento originales. Sólo lee el enlace CA y coteja el evento
existente en AD192. Una consulta funcional posterior usa `SERIALIZABLE` y un
evento nuevo; no se confunde con la recuperación.

Una tabla CA inmutable enlazará operación y recibo con referencia, versión y
huella de vínculo IS16, autenticación, sesión y evento AD192. Es dato del
dominio de contexto, no otra cadena de auditoría. El resultado favorable
necesita el evento común AD192 confirmado en la misma transacción que el
contexto. Si el núcleo falla después de empezar el efecto,
la escritura se revierte en una subtransacción y se registra el resultado
negativo antes de confirmar. El adaptador retiene el evento original sólo
durante esa llamada, sin enviarlo a HTTP ni escribirlo en logs. Si la
respuesta auditada fue negativa, la recuperación devuelve ese estado sin
contexto V2; no afirma que no haya otros registros históricos.

IS16 y AD192 han publicado candidatos de sus fachadas propietarias. La
preimagen real del núcleo CA está medida. CA36 mantiene una parada explícita
de instalación hasta dos revisiones independientes y el ensayo PostgreSQL
del escritor único. Las pruebas Go usan transporte sintético: comprueban la
ligadura del acuse y la recuperación de un COMMIT incierto, pero no acreditan
firma, efecto PostgreSQL, montaje ni recorrido HTTP.
