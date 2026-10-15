# Ajustes de firma y catálogo: F1-37

El corte usa las pantallas de plantillas y la ficha del expediente existentes.
Coteja la publicación del catálogo de su organización y unifica el límite del
PDF de la firma VEC en 2 MiB. La verificación conserva sus comprobaciones.

## Preparación

1. Restaurar una copia propia de la principal, con sus permisos exactos de
   base y tipos y la configuración privada vigente. Conservar el material
   anterior para volver atrás. El conector privado del almacén debe anunciar
   una capacidad de al menos 2 MiB por objeto; si es menor, ajustarla por el
   procedimiento de configuración existente antes del reinicio vigilado.
2. Comprobar CT218 instalada. Instalar CT226 una sola vez según
   `lista_sql_codexh_firma_ajustes_f1_37_20261015.txt`. Su guarda exige la
   definición instalada exacta y rechaza una segunda aplicación. No modificar
   ni reaplicar CT218.
3. Compilar el portal sobre la cabeza congelada. Servir juntos el binario,
   los módulos web, su índice y sus manifiestos. No hay una variable nueva ni
   nuevos actos, cargos, personas o permisos funcionales.

## Comprobación

1. Con el catálogo de plantillas configurado, arrancar y consultar sus
   documentos desde la ficha. La sonda coteja huella, versión y fuente de la
   última publicación de su organización, además de su auditoría y recibo.
   La provisión inicial común se conserva; una publicación ajena no la
   sustituye para otra organización sin publicación propia.
2. Ejecutar la prueba SQL de CT226 en la copia con un LOGIN técnico de CT:
   `psql -v ct_login=LOGIN_CT -f ct226_catalogo_plantillas_por_organizacion.sql`.
   No altera los catálogos ni los permisos. Contrasta la provisión común con
   una publicación corporativa posterior, las dos combinaciones cruzadas y
   la organización ausente.
3. Comprobar los bordes de tamaño en las pruebas focales: 2 MiB admitidos,
   2 MiB más un byte rechazados; JSON y Base64 se calculan aparte. La ruta
   antigua puede contener dos PDF en su manejador. La pasarela conserva
   1 MiB de petición JSON total para `firmas-documento` y `registro-externo`;
   esas rutas no acreditan un PDF de 2 MiB por esa frontera. La excepción
   mayor corresponde solo al POST exacto `registro-vec`. El límite general
   de otras peticiones no aumenta. El rechazo no debe reservar ni custodiar
   el documento.
4. Recuperar un PDF firmado y verificado ya registrado desde la ficha, con
   el certificado autorizado. Reiniciar base y portal y recuperar los mismos
   bytes y el mismo recibo. No se reescriben firmas anteriores. El límite
   ampliado se contrasta además en las pruebas focales del paso anterior.
5. Comprobar también el arranque sin seleccionar la configuración opcional
   de plantillas, conforme al lanzador de la candidata. El resto de las
   funciones del portal debe seguir disponible.

Un cierre de auditoría ausente no tapa el error de montaje. Si existe, se
llama una sola vez y se conserva el orden de limpieza.

## Vuelta atrás y límites

Restaurar de forma vigilada el binario y el conjunto web anteriores junto
con su configuración privada. Conservar CT226 instalada: no ejecutar DOWN.
El contrato anterior de la sonda conserva su firma para la organización
única que admitía CT135. El material histórico mantiene sus bytes y recibos.

La instalación en cidonia corresponde a dirección. Este corte no acredita
el recorrido único completo de aceptación de FLUJO_APP §4.0.
