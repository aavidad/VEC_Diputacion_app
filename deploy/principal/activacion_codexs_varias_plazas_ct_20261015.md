# Ofertas de varias plazas desde Contratación temporal

Estado al 15/10/2026: **candidata preparada, sin ensayo conjunto ni instalación acreditados**. Este guion corresponde a datos ficticios en una copia de la principal. Dirección congela y ensaya la candidata completa antes de decidir una subida. El primer corte de Bolsa y su B117 ya tienen su propio ensayo; no se reaplica B117.

## Dependencias y orden

Comprobar en el registro de migraciones de la copia que están instaladas AD263 y AD265 del relevo HZ37 y AD277 del equipo O, con sus concesiones vigentes. AD277 no forma parte de esta lista. Si falta una, detener la activación y resolverla por su entrega revisada; no crear permisos en el arranque.

Comprobar también que B117 ya está instalada una sola vez y que la bolsa usada
dispone de una política con inicio por publicación. Su activación se describe
en `deploy/principal/activacion_codexs_ofertas_20261015.md`. No cambiar una
política histórica ni reaplicar B117 para preparar este corte.

Sobre una copia que conserve su historia, aplicar **solo si aún no constan instaladas**, y en este orden, las entradas de `deploy/principal/lista_sql_codexs_varias_plazas_c2_20261015.txt`:

1. AD278: autorización del vínculo, el cierre de la intención en Bolsa y el descarte en CT.
2. B120: verificación nominal de la oferta de Bolsa para CT.
3. B121: cierre verificable de una intención sin publicación y protección frente a una publicación tardía.
4. CT220: reserva, vínculo, descarte, historia y recibos de la selección de oferta.

El tercer corte agrega CT223 y después CT225, listadas en `deploy/principal/lista_sql_codexs_varias_plazas_c3_20261015.txt`. CT225 permite consultar por expediente el intento de oferta guardado con una decisión de autorización nueva y su apunte de auditoría. Comparte el bloqueo de cabecera con la reserva. CT223 no cambia la regla del expediente fiscalizado: cerrar una intención sin oferta no autoriza cancelar ese expediente. No mezclar listas ni aplicar una migración con el mismo número de otro esquema.

El cuarto corte añade CT227, en `lista_sql_codexs_varias_plazas_c4_20261015.txt`, después de CT225. Parte de las funciones instaladas y cambia el ámbito de vínculo, consulta y descarte a organización y unidad gestora; conserva los cánones y recibos anteriores. Añade un lector preparatorio nominal que obtiene la unidad del expediente. No se reinstalan CT220, CT223 ni CT225.

Antes de ensayar, conservar la copia anterior y las huellas de los archivos SQL que se van a ejecutar. Revisar las preimágenes, ACL y funciones nominales en el clon; cualquier fallo detiene la instalación de la candidata. Un arranque con una parte de esta lista o con concesiones improvisadas no acredita el corte.

## Configuración privada

La capacidad se selecciona con el perfil de desarrollo de doble llave y un archivo privado **`identidad/ofertas-ct.json`** dentro de `VEC_DEVELOPMENT_MATERIAL_DIR`. No se añade una variable de entorno propia de ofertas. El archivo usa `esquema: vec.ct.ofertas.activacion.v1` y tres apartados distintos: `vinculo`, `cierre_bolsa` y `descarte_ct`. Cada apartado contiene `perfil_ref`, `rol` (`rol_id`, `version`, `huella_sha256`) y `motivo` (`catalogo_id`, `catalogo_version`, `catalogo_huella_sha256`, `entrada_clave`). También declara `fuente_autorizacion_dsn_file` y `motivos_autorizacion_dsn_file`, nombres locales relativos al material privado y diferentes entre sí.

Los tres roles y motivos deben estar publicados antes por las vías de Administración ya existentes. Roles, motivos y perfiles deben ser distintos para cada efecto. La identidad de la persona, su cuenta y su certificado son los mismos en los tres actos; el archivo no concede permisos ni sustituye los actos administrativos. No guardar en Git el JSON privado, DSN, certificados, huellas de identidad ni otros secretos. Comprobar que los archivos privados tienen permisos restringidos y que los roles y concesiones corresponden a las migraciones instaladas.

El vínculo, su consulta y el descarte se autorizan por `organizacion_ref` y
`unidad_ref`: la unidad gestora guardada en el expediente. El centro y la
categoría siguen cotejándose con la necesidad del expediente; no son dimensiones
de la asignación de estos perfiles. El cierre de la intención en Bolsa conserva
`unidad_ref` y `ambito_ref` de la bolsa. No confundir la unidad gestora con la
unidad del puesto, que pertenece a la proyección de Personal.

La asignación de vínculo y descarte debe usar el lote ordinario de Administración
(AUT44), que ya admite organización y unidad. Falta acreditar los actos y sus
recibos junto con la publicación de roles y motivos. No reutilizar el dato antiguo
de la copia que asignaba centro y categoría, ni presentarlo como activación.
Los datos ficticios se preparan solo en la copia propia; no conceden permisos en
cidonia ni sustituyen esos actos.

La ausencia o invalidez de esta configuración deja cerrada la función de oferta desde CT. La aplicación y el llamamiento singular deben arrancar. Un perfil o permiso denegado se corrige mediante Administración y revisión de SQL; no se amplía para desbloquear una prueba.

## Ensayo pendiente en la copia

1. Arrancar primero **sin** `identidad/ofertas-ct.json`. Comprobar salud de la aplicación, apertura de la ficha CT y recorrido singular de una persona. La función nueva debe quedar indisponible de forma comprensible. Registrar versión del binario, configuración no sensible y resultado.
2. Preparar el archivo privado con los tres actos de Administración ya publicados. Reiniciar de forma vigilada; comprobar salud y la misma ficha, además de la capacidad nueva. Ante fallo de arranque, retirar la selección nueva, restaurar la referencia anterior del material y reiniciar. La copia previa de la base permite volver atrás si la instalación SQL falló; no borrar ni reescribir historia de migraciones.
3. Usar la **variante de varias plazas**, separada del caso singular 4.0 de `docs/producto/FLUJO_APP.md`: categoría Administrativo, Antonio y otra persona ficticia existente y disponible en la bolsa. La petición pide dos personas; conservar la necesidad RPT y llegar a fiscalización favorable. No fabricar candidatos desde código. Comprobar también el caso de una persona como regresión singular.
4. En la ficha de CT, revisar y confirmar «Continuar con Bolsa». El POST de selección lleva solo `expediente_ref`, `version_esperada` y `clave_idempotencia`; el servidor deriva bolsa, datos y cantidad. Para la variante de dos personas, Bolsa debe acreditar dos plazas elegibles y una política con inicio por publicación. Verificar el recibo original de Bolsa, el recibo del vínculo CT, la fecha y el plazo; entrar en «Gestionar ofertas» y consultar la misma oferta. Ningún recibo prueba el envío de correo.
5. Perder de forma controlada una respuesta después de guardar, recargar la ficha y comprobar que la consulta `GET /api/vec/contratacion-temporal/llamamientos/seleccion/estado?expediente_ref=…` recupera el intento original sin enviar ningún POST. Si la consulta falla, la pantalla debe impedir preparar otra oferta y permitir reintentarla. Pulsar después la recuperación explícita, conservando la petición y la clave guardadas por el servidor. Debe volver la misma oferta y los mismos recibos. Si el servidor identifica una oferta indeterminada, pulsar «Recuperar operación»: el cuerpo conserva esos tres campos y añade al final `accion: recuperar_oferta`. No crear otra clave durante esta recuperación.
6. Ensayar también la intención sin publicación que supera cinco minutos. Bolsa debe acreditar ausencia y cerrar físicamente la intención mediante B121; CT debe registrar el descarte y mostrar el recibo de cierre de Bolsa y el de descarte CT. Un intento tardío de publicar con la clave cerrada no debe ganar. «Preparar nueva oferta» solo aparece tras ese terminal: confirmar esa decisión, conservar visibles ambos justificantes anteriores y comprobar que el intento nuevo lleva otra clave. Un 409 `seleccion_oferta_sin_reserva` no es un recibo de cierre: repetir la solicitud original.
7. Comprobar la guarda CT223 frente a cancelación concurrente bajo el bloqueo de cabecera y confirmar que la política sigue rechazando cancelar el expediente fiscalizado. Reiniciar aplicación y PostgreSQL, consultar de nuevo recibos e historia, y repetir arranque con el archivo privado retirado. Ensayar en Chrome del sistema a 1440 y 390 px, en castellano e inglés, con teclado y sin desbordamiento horizontal.

No ejecutar dos POST distintos para averiguar qué pasó con uno incierto. Guardar el resultado observado de cada paso, las huellas de la candidata y el orden de instalación. El ensayo SQL, la configuración conjunta, el recorrido Chrome y el reinicio de C2/C3 **siguen pendientes** a la fecha de este documento; por tanto esta PR no está lista para instalarse en cidonia.

## Fuera del corte

La retirada de una oferta ya publicada y la intención durable de aviso por correo siguen pendientes. El cierre de una intención sin publicación no equivale a retirada ni a notificación. La aceptación de la persona aspirante, su nombramiento y el cese requieren sus propios recibos y comprobaciones; este guion no los da por acreditados.
El ensayo de la variante llega hasta la adjudicación en Bolsa. **Pendiente:** continuar con la propuesta desde una oferta adjudicada; la propuesta singular exige un llamamiento aceptado. Siguen pendientes la consulta de Organización histórica, la correspondencia entre la unidad que tramita y la del puesto y la asignación de los permisos necesarios.

## Corrección de recuperación tras recargar

CT225 se instaló una vez en la copia local con SHA-256 `5c9a6ac291338f2ed87c38d94ada7695c206e41668484cd524f793faeba1a235`, tras revisión SQL y sensible favorables. Cuatro rechazos de material nulo, campos adicionales y tipos inválidos terminaron en rollback. Chrome con certificado obtuvo `200 ausente` al consultar el estado de N2 y `200` con cantidad `1` en el detalle de un expediente legado. No se reaplican CT220, CT223 ni las migraciones de Bolsa para añadirla. La consulta usa el perfil vinculador, una frontera GET propia y la autorización nominal del expediente; no habilita cierre ni descarte con el perfil de consulta. La respuesta minimiza los datos al estado, la intención opaca original y los recibos terminales. El cliente no guarda credenciales ni operaciones en el navegador.

Quedan por acreditar la recuperación de un pendiente después de recargar y el recorrido singular completo. Con la configuración de ofertas retirada, el binario final respondió portal `200`, detalle antiguo `200` con una persona y solo oferta `503`; se restauró la configuración con reinicio vigilado. La ficha N2 ajustada a vacante devolvió `404` también con el binario anterior. Se corrigieron las seis huellas y modalidades desactualizadas de su proyección RRHH mediante un guion privado revisado, sin cambiar actos ni recibos. Después Chrome abrió la ficha: detalle `200` con dos personas y consulta del intento `200 ausente` desde el consumidor real. El `200 ausente` no acredita publicación ni recuperación de una oferta. Las pruebas focales no sustituyen estas comprobaciones del recorrido.

Comprobación local de seguridad: Semgrep sobre los archivos cambiados mantiene un aviso en una línea anterior de firma, sin hallazgos nuevos. gosec detectó una conversión estrecha en la consulta nueva; se sustituyó por comparación en entero de 64 bits. El focal posterior no detecta esa conversión; los avisos restantes pertenecen a código previo. El recorrido completo sigue pendiente.

La lista de comunicaciones devolvió `503` en la primera consulta y `200` al usar «Volver a cargar»; no se eliminó esa guarda para publicar otra oferta. El cliente de Organización histórica heredado se retiró del material propio: se configuró un puerto propio, sin llamar al servicio de otro equipo. Aún falta activar el montaje de consulta histórica y la distinción entre unidad gestora y unidad del puesto antes de publicar desde CT.
