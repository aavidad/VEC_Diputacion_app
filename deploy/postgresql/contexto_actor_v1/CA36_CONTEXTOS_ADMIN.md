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

Una tabla CA inmutable enlazará operación y recibo con referencia, versión y
huella de vínculo IS16, autenticación, sesión y evento AD192. Es dato del
dominio de contexto, no otra cadena de auditoría. El resultado favorable
necesita el evento común AD192 confirmado en la misma transacción que el
contexto. Si el núcleo falla después de empezar el efecto,
la escritura se revierte en una subtransacción y se registra el resultado
negativo antes de confirmar. Un COMMIT incierto se consulta con la misma
operación, el mismo recibo y el mismo evento; no se inventa un contexto V2 ni
se registra una segunda operación.

La firma SQL final depende de las fachadas IS16 de sesión y vínculo exactos,
de la ABI15/acuse AD192 y de su cotejo de evento sin escritura. La preimagen
real del núcleo CA ya está medida; el borrador SQL mantiene una parada
explícita de instalación. Estas dependencias aún no acreditan un resultado
favorable, un ensayo PostgreSQL ni un recorrido HTTP.
