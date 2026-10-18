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

## Comprobación pendiente

Las pruebas focales Go, `gopls` y las 79 pruebas de interfaz y caché son
favorables para el corte anterior al SQL. La revisión de interfaz de T sobre
`4b5c02707` es favorable. S detectó que el SQL instalado todavía exigía la
terna de puesto, jornada y número: dirección autorizó CT239 para resolverlo.
El SQL y los ajustes de diagnóstico guardados durante la parada siguen
pendientes de ensayo y revisión sobre la cabeza final.

El recorrido HTTP en Chrome, la fiscalización y la reserva real con CT235
todavía no están acreditados. La aceptación debe conservar el reintento
antiguo descrito arriba y comprobar un alta nueva de dos personas, con
cantidad sellada transmitida hasta la reserva; también debe rechazar las
divergencias. Un resultado focal de backend no sustituye ese recorrido.

El único envío real del contrato nuevo devolvió 503. La conciliación anterior
a la parada encontró cero expedientes y confirmaciones y una candidatura
técnica preparatoria. Se conserva el cuerpo y su clave; antes de repetir la
operación se recupera su estado por la vía existente. Ese envío no acredita
un alta ni una reserva de plazas.
