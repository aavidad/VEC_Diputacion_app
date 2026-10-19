# Contexto de una comprobación pendiente

F1-09. Consumidor: respuesta HTTP 409 de la selección del llamamiento desde
la ficha de contratación temporal. No añade rutas, lecturas ni configuración.

La propiedad `comprobacion_ofrecimiento.contexto` aporta los 16 campos del
POST de comprobación de Bolsa: ámbito, unidad, organización, expediente,
necesidad, bolsa, categoría, participación, posición, modalidad, causa y
versiones de expediente, bolsa, instantánea y política, junto a su huella.
El ámbito procede de la misma orden interna ligada a la lectura auditada;
el resto procede del expediente y del origen ya cotejados. Solo se publica
el contexto coherente de la primera posición pendiente. No se incluyen hechos,
contactos, documentos ni la referencia de candidato.

Activación: instalar el binario de la candidata con la configuración existente
de selección CT. No requiere SQL ni concesiones adicionales. La ausencia de
configuración conserva el arranque; no se ofrece un contexto alternativo.
La extracción del ámbito se aplica solo a una comprobación pendiente; las
selecciones sin ella conservan su recorrido.

Comprobar con un expediente fiscalizado y una persona pendiente según la regla
publicada: la selección devuelve 409 con el contexto, sin crear llamamiento.
La respuesta no incluye aún el estado del acto ni una versión esperada; la
pantalla debe esperar esa ampliación y no asumir que nunca hubo una comprobación.
Focales de composición y HTTP, incluidas incoherencias y minimización, verdes.
El recorrido con configuración conjunta en Chrome sigue pendiente.

La candidata conjunta `601686060` instaló 98 SQL en dos copias nuevas. Su
Go conserva este corte sin cambios frente a `1ed8b0e42`; ambas aplicaciones
arrancaron y reiniciaron con la comprobación del guion. La selección CT que
devuelve 409 con el contexto no se ha recorrido aún con el caso completo.

Vuelta atrás: recuperar el binario anterior y su configuración conservada.
La ampliación no guarda datos ni altera situaciones o posiciones.
