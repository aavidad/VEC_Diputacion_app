# Alta directa de RRHH para varias personas

F1-73 incorpora el esquema `vec.ct.alta_necesidad.v2` para las altas nuevas
con necesidad estructurada. El backend está en `b4e0b4d6e`; el emisor web y
su cohorte de caché están preparados. El formulario declara la versión
nueva y el servidor obtiene el número de personas de la necesidad sellada,
con validación de entero y límites. No se admite otra cifra independiente
en la solicitud ni se fabrica un puesto para completar el dato.

El esquema anterior `vec.ct.alta_necesidad.v1` conserva su tratamiento
literal. Un alta antigua cuya necesidad contiene dos personas, pero cuya
solicitud no tenía ese número, se recupera sin añadirlo: misma clave,
bytes originales, folio y recibo. No se recalcula el alta histórica con el
contrato nuevo. CT235 conserva sus comprobaciones de versión, cantidad,
fase y origen; este cambio no las modifica.

## Activación y vuelta atrás

CONFIG NUEVA: ninguna. SQL a instalar: CT239, una sola vez después de
las migraciones de la lista única
[`lista_sql_main_tras_hz37.txt`](lista_sql_main_tras_hz37.txt).

CT239 ajusta cuatro funciones instaladas de Contratación temporal. La
solicitud puede llevar únicamente el número superior si la necesidad sellada
es válida y ambas cantidades coinciden. La validación del análisis aplica
la misma regla; la reconstrucción conserva ese número sin inventar puesto
ni jornada. Una solicitud histórica sin número superior conserva sus bytes.
La migración coteja cuerpos, propietario, permisos, configuración y atributos
antes y después. No añade permisos ni modifica expedientes existentes.

Publicar juntos el binario, los ficheros web y la cohorte de caché del mismo
corte. No activar el emisor nuevo con un servidor que solo admita el esquema
anterior. La fuente opcional de necesidades y las fuentes de RPT siguen la
receta existente de [copia local](LEEME_copia_local.md), apartado 6. Este
corte no acredita su publicación administrativa ni añade otra fuente.

Sin esas fuentes permanece la vía anterior; la función opcional debe
explicar su indisponibilidad sin impedir el arranque. Conservar el binario,
los ficheros web y la configuración anteriores antes de sustituirlos. Hacer
un reinicio vigilado y comprobar arranque y acceso; si falla, restaurar ese
conjunto anterior y volver a arrancar. No reaplicar SQL ni alterar historia
para volver atrás.

## Comprobación realizada y límites

Las pruebas focales Go, `gopls` y las 79 pruebas de interfaz y caché son
favorables para el corte anterior al SQL. La revisión de interfaz de T sobre
`4b5c02707` es favorable. S detectó que el SQL instalado todavía exigía la
terna de puesto, jornada y número: dirección autorizó CT239 para resolverlo.
El SQL pasó el ensayo focal con ROLLBACK: cantidades coherentes y divergentes,
terna anterior, reconstrucción, bytes históricos y análisis. Los tres casos
nuevos de diagnóstico también pasan; la causa SQL se conserva en el registro
técnico sin exponerse en HTTP. La versión final declara las cuatro funciones
de forma explícita y supera las guardas de `search_path` y del preparador de
roles. S y el revisor SQL independiente dieron su conformidad sobre
`bf7691183`: migración SHA-256
`5422bedbbb837f29371f8c1326cf63233f72b916a12a2b2c30e0e32cfa3bf435`.
La unión posterior con main conserva esos bytes y pasa 134 pruebas focales
de interfaz, montaje y caché.

La copia de `bf7691183` instaló una vez las 87 migraciones de la lista y
arrancó en 84,77 segundos. Sin la fuente opcional de necesidades,
`/livez` respondió 200 y el catálogo de alta nuevo devolvió 503 con el error
catalogado de configuración ausente. Tras declarar la fuente propia y
reiniciar, el catálogo respondió 200.

El primer envío, anterior a CT239, había respondido 503 y se había conciliado
sin expediente ni confirmación. Se conservó su cuerpo y su clave. Con CT239,
el reenvío exacto desde Chrome respondió 201: expediente `2026/95073`, una
alta y una confirmación. La solicitud y su necesidad conservan dos personas.
El expediente avanzó desde la ficha hasta análisis, cobertura, asignación e
informe jurídico, con respuesta 201 por cada operación y versión final 5.

Tras reiniciar aplicación y PostgreSQL se recuperaron la cantidad, las cinco
actuaciones y sus fechas. El reintento exacto del alta devolvió el mismo
recibo original y su instante, sin otra alta ni confirmación. Las capturas
de escritorio y móvil están en `docs/operacion/capturas/f1-73/`.

La copia necesitaba declarar el catálogo de firma existente mediante
`VEC_CT_CIRCUITO_FIRMA_SOURCE_PATH`; después del reinicio esa consulta
respondió 200. La ficha sigue indicando que la firma nominal no está
disponible: falta un plan aplicable a la organización y unidad del expediente.
No se firmó, fiscalizó ni reservó con CT235. Esos pasos siguen pendientes de
la preparación común de firma; no se sustituyen por cambios de fase o
permisos mediante SQL. La reserva de varias plazas con CT235 se comprobará
en la copia cuando exista ese plan de firma.
