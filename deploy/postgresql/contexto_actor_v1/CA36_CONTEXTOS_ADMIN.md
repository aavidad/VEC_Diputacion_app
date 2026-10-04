# Contexto ADMIN nominal — CA36

CA36 prepara el registro y la recuperación de un contexto V2 para una petición
ADMIN concreta. La cuenta privilegiada, el perfil seleccionado y la sesión
deben seguir vigentes después de las últimas esperas. El vínculo IS procede
de la misma petición y se comprueba por referencia, versión y huella exactas;
buscar la última sesión de una cuenta no acredita esa relación.

La fachada CA tendrá un grupo de ejecución exclusivo. El LOGIN no heredará el
grupo general de ContextoActor ni permisos sobre sus tablas. La fachada
propietaria reutilizará el núcleo V2 para los bytes, la huella, la operación,
el recibo y el CAS. El acceso general conservará sus metadatos y ACL. En ADMIN
el alcance de proyecciones será vacío, el método será certificado o DNIe y la
garantía será alta.

El resultado favorable necesita el evento común AD192 confirmado en la misma
transacción que el contexto. Si el núcleo falla después de empezar el efecto,
la escritura se revierte en una subtransacción y se registra el resultado
negativo antes de confirmar. Un COMMIT incierto se consulta con la misma
operación, el mismo recibo y el mismo evento; no se inventa un contexto V2 ni
se registra una segunda operación.

La firma SQL final depende de las fachadas IS16 de sesión y vínculo exactos,
de la ABI15/acuse AD192 y de la preimagen real del runtime CA. Estas
dependencias aún no acreditan un resultado favorable, un ensayo PostgreSQL ni
un recorrido HTTP.
