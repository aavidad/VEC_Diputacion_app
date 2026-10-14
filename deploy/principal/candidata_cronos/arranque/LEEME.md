# Activación de Cronos sin detener el portal

Este corte permite preparar Cronos por separado: si su configuración está
ausente, es inválida o falta una dependencia propia, Cronos responde como no
disponible y Contratación y el resto del portal siguen funcionando.

CONFIG NUEVA: ninguna. Conserva los tres selectores de Cronos, el fichero
privado `identidad/cronos-empleado.json` y sus contratos actuales. SQL nueva:
ninguna. No instala ni reaplica migraciones al arrancar.

## Preparación en la copia

1. Restaurar HZ37, sus roles y las ACL de tipos y de la base entregadas por
   dirección. Preparar las conexiones nominales y el material privado de la
   aplicación; comprobar antes que Contratación arranca con esa configuración.
2. Compilar el commit de este corte y conservar las huellas del binario y la
   web. Todas las comprobaciones siguientes usan ese mismo código y la misma
   copia. No cambian servicios ni material de cidonia.
3. Conservar el valor vigente de `VEC_PORTAL_MODULOS_VISIBLES` y añadir `cronos`
   para comprobar su pantalla. El menú no concede permisos.

## Comprobaciones de activación

| Configuración | Resultado esperado |
| --- | --- |
| Tres selectores Cronos apagados | Arranca; no lee el fichero ni sondea conexiones propias de Cronos. |
| `VEC_CRONOS_EMPLEADO_ENABLED=true`, sin fichero privado | Arranca; las rutas internas de Cronos responden 503; el portal y las rutas de Contratación conservan su comportamiento. |
| Fichero incompleto o selector mal escrito | Igual cierre de Cronos; el diagnóstico indica el componente sin exponer la configuración. |
| Fichero completo, una conexión propia no disponible | Igual cierre de Cronos; el resto sigue disponible. |
| Dependencias completas y permisos publicados | Monta las rutas autorizadas de Cronos y conserva los controles de identidad, autorización y auditoría. |

Se comprueban `/livez`, el portal del empleado y una lectura autorizada de
Contratación con Cronos apagado y solicitado. En Chrome, la zona afectada
explica que no se ha podido cargar y conserva la navegación. Después se
reinicia la aplicación de la copia y se repiten las lecturas.

El respaldo HZ37 contiene un fichero Cronos anterior al requisito del motivo
de rechazo y no tiene instaladas las dependencias de resolución. Este corte
no publica motivos ni perfiles y no acredita por sí solo el recorrido completo
de Cronos. La candidata general conserva su inventario y su lista SQL propios.

## Validación del código

Dos revisiones independientes dieron GO a los cinco archivos de montaje de
esta versión. La revisión corrigió un fallo: el plazo de publicación de Cronos
podía vencer el contexto de Contratación. Ahora la publicación opcional queda
después de todas sus operaciones obligatorias y tiene plazo independiente.

Las pruebas focales de selectores, configuración privada, cierre de rutas y
plazo independiente han pasado. `gopls` y Semgrep local pasaron; gosec no
encontró hallazgos en los archivos cambiados. La prueba focal de servidor con
PostgreSQL estaba omitida por no tener configurada esa base en su ejecutor.
En la copia propia, el mismo binario arrancó con Cronos apagado y con su
configuración incompleta: salud, portal, sesión y catálogo de módulos
respondieron 200; las rutas de Cronos quedaron en 404 o 503 según el selector.
La configuración privada se restituyó después de comprobar el cierre.

La revisión independiente en Chrome del sistema dio GO en castellano e inglés,
a 1440 y 390 píxeles: los avisos de saldo y fichaje explican la indisponibilidad;
«Saltar al contenido principal» conserva Cronos y enfoca el contenido. Bolsas
sigue accesible. No hubo desbordamiento ni almacenamiento web. También se
comprobó reflujo a 720 píxeles; el zoom nativo no quedó medido. En móvil, el
aviso de saldo queda bajo la primera pantalla. Esta comprobación no acredita
solicitudes, recibos ni recuperación tras reinicio.

La cadena de caché renueva únicamente las vistas afectadas y sus importadores;
la versión del contrato de caché pública conserva la compatibilidad con el
trabajador de servicio existente. Las pruebas de la cadena comprueban también
que los recursos de Contratación conservan sus versiones propias.

Los cinco archivos de producción y prueba forman una sola responsabilidad:
aislar la activación de Cronos en las dos fases de montaje que comparte con
Contratación. Se reutilizan la frontera, el gobierno y los validadores actuales.

La puerta completa `scripts/verificar_calidad.sh` pasó sobre este árbol: Go,
pruebas con detector de carreras, comprobaciones web y de superficies,
catálogos, vulnerabilidades y tamaños. Las pruebas web se ejecutaron con
concurrencia cuatro para evitar saturar Chrome local; no se omitió ninguna.
La línea base del test de versiones pasa de 1130 a 1132 por dos líneas que
registran las URLs renovadas y conservan su comprobación estricta.
