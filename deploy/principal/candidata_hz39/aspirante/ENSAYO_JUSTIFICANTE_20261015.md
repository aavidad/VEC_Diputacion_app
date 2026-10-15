# Ensayo del justificante de renuncia, 15 de octubre de 2026

La custodia y su activación están preparadas. Sigue pendiente completar la subida desde Mi Bolsa, registrar la respuesta, descargar el mismo original desde RRHH y comprobarlo tras reiniciar. Mi Bolsa ya devuelve 200 tras admitir los tres llamantes técnicos en Identidad. El GET documental previo a subir rechaza el ámbito: la asignación candidata usa `candidato_ref` y el recurso documental pide `persona_ref`. La corrección de este contrato tiene revisión de arquitectura pendiente. Este documento no acredita el recorrido de aceptación ni una instalación en cidonia.

## Base y orden

Se restauró una copia propia y desechable de PostgreSQL 18 desde HZ37, con sus permisos de base y tipos. Se instalaron una vez AD263 y AD265, y después las seis migraciones de la lista `deploy/principal/lista_sql_codexo_justificante_renuncia_20261015.txt`. Las funciones reconstruidas están completas en el fichero y fijan `search_path` a `pg_catalog,pg_temp`. Dos revisiones independientes comprobaron el contenido exacto.

| Migración | SHA-256 de la fuente ensayada |
| --- | --- |
| `000277_justificante_bolsa_documental.up.sql` | `620a1319ea4e2cc67ebffd4f281d3b53327e2abf9e5af7f543e20f89d3e0489c` |
| `000095_autorizacion_subida_documental_externa.up.sql` | `f5ceff6d900ba5fb4110195faefa315840a53103cb2742a7e977a35041923a1d` |
| `000020_subida_documental_comun.up.sql` | `d30d7ea1d565874172535662927034617fde81ce2797923f3c0e5ef6871943df` |
| `000022_justificante_renuncia_bolsa.up.sql` | `7fcae3a5d21d1b86b812271bb51fe0149dc2b06f81b44b4617acc514c4dcda0f` |
| `000119_justificante_renuncia_documental.up.sql` | `f51e877cdff8435ed60c23c36e7b7bc60cedeed4910b6137f9dd20722d969d56` |
| `000222_frontera_justificante_renuncia_bolsa.up.sql` | `5b7d7dc636e4ba5368a1a0f2dec6c86b524c04f44904de12266e77cb96bc8e80` |

Las seis pruebas focales SQL terminaron correctamente en esa copia. La preparación de confianza externa publicó las 28 claves mediante el publicador nominal existente. Las claves de lectura de RRHH se publicaron mediante el gobierno interno, sin crear perfiles ni conceder permisos de negocio al arrancar.

El guion de datos requiere una copia HZ37 recién restaurada y comprueba los LOGIN, sus membresías exclusivas, las contraseñas SCRAM y la ausencia de permisos temporales. La repetición del guion terminó con `justificantes=preparados candidato=asignado`. La transición de una asignación documental antigua de Elena tiene revisión estática; no se ha ensayado por separado. Para este ensayo se cargó un llamamiento ficticio abierto de Antonio: es una precondición de datos, no una prueba de emisión o notificación de un llamamiento.

## Configuración opcional

Cada cambio se hizo con la aplicación detenida y se verificó el reinicio con TLS y certificado de cliente. Después se restauraron los dos ficheros originales.

| Configuración documental | Preferencias del usuario | Justificante propio | Descarga RRHH |
| --- | --- | --- | --- |
| Ausente | 200 | 503 | 503 |
| JSON incorrecto | 200 | 503 | 503 |
| Restaurada | 200 | 503; denegación por desajuste de ámbito en revisión | 403 para el aspirante sin permiso de RRHH |

Las dos primeras variantes mantienen disponible el resto de la aplicación comprobado. La última acredita que RRHH rechaza a un aspirante; aún falta su descarga positiva con la técnica autorizada.

## Comprobaciones de código

Las pruebas web focales terminaron con 367 casos correctos. También pasaron las pruebas focales de subida, origen, límite HTTP, recuperación por clave, descarga y selección de confianza. Las guardas de superficies, manifiestos, traducciones y `search_path` pasan.

Semgrep local con reglas de Go y JavaScript no encontró problemas nuevos: señaló el uso previo de `path.Clean` en la comprobación de ruta canónica, que compara la ruta original y rechaza diferencias. Gosec se ejecutó sobre los paquetes afectados; no señaló los ficheros nuevos de justificantes, pero informó problemas anteriores y errores de análisis de tipos, por lo que su cobertura es parcial. Las revisiones independientes del código conservan su propio alcance.

La prueba Go global local detectó cuatro fallos de inventario y expectativas de publicación/rutas. Se corrigieron con pruebas focales; no se repitió la global local por orden de dirección. La puerta completa de calidad corresponde a la CI y debe terminar correctamente antes de pedir la integración.

## Corte congelado para revisión de dirección

AUT98 añade el catálogo documental a la autoridad externa, ligado solo a los dos actos propios de justificante. Su SHA es `8f594c850c9f6a683b340a1c330ae209e37f04367d9df88636623ca92b6cbc05`; instalación única y prueba estructural correctas, con dos revisiones independientes.

La copia incorpora AD278 de S (SHA `819835d73824a4815635413d8ef180f4d10c8e6663205fcb84705fd15b3b6fc1`) y después AUT96 → AD279 → B116 → AD275, una vez y en ese orden. AD279 tiene SHA `9309bf189c8d8a899845460fa7d1d0f6215864f7b773f848ea01249931c9de2d`; su prueba estructural pasa y conserva los perfiles de S. Las cuatro fuentes tienen dos revisiones sensibles favorables. El material de preferencias internas está preparado pero sigue sin activar: faltan el LOGIN y los datos revisados de concesión/asignación, y el recorrido GET/PUT con reinicio.

La admisión de llamantes es un dato ficticio de la copia; para instalar en otro entorno se necesitan las aprobaciones formales de Identidad. La causa del antiguo 503 de Mi Bolsa quedó confirmada por una prueba nominal del registro en transacción revertida y por el GET 200 tras completar ese dato. El envío documental permanece pendiente; no se fabricó una concesión contraria al PDP ni un recibo de subida.
