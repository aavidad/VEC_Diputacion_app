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

CONFIG NUEVA: ninguna. SQL a instalar: ninguna.

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
favorables. Una revisión independiente del backend también es favorable;
quedan pendientes la revisión de S y la revisión de usabilidad de T.

El recorrido HTTP en Chrome, la fiscalización y la reserva real con CT235
todavía no están acreditados. La aceptación debe conservar el reintento
antiguo descrito arriba y comprobar un alta nueva de dos personas, con
cantidad sellada transmitida hasta la reserva; también debe rechazar las
divergencias. Un resultado focal de backend no sustituye ese recorrido.
