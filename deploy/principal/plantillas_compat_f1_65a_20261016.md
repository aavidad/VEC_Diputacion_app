# Retirada de la comprobación antigua de plantillas

CT232 elimina la entrada de compatibilidad V1 de la comprobación del catálogo de
plantillas. El portal ya usa V2, que recibe la organización del expediente. No
cambia las plantillas, su publicación, los recibos ni los permisos de V2.

## Activación en una copia local

1. Partir de una copia de la principal con CT218 y CT226 instaladas. Conservar
   la copia anterior para la vuelta atrás. CT232 exige las definiciones y los
   permisos exactos de ambas funciones y que V1 no tenga dependencias; si
   divergen, detiene la instalación antes de borrar nada.
2. Instalar una sola vez la lista
   `lista_sql_codexh_plantillas_compat_f1_65a_20261016.txt`. No hay variables,
   actos, personas ni datos nuevos. No reaplicar CT218 ni CT226.
3. Ejecutar las pruebas de CT226 y CT232 con un LOGIN técnico de CT:
   `psql -v ct_login=LOGIN_CT -f deploy/postgresql/contratacion_temporal/pruebas_sql/ct226_catalogo_plantillas_por_organizacion.sql`
   y el mismo comando con `ct232_retirada_sonda_plantillas_v1.sql`. La primera
   conserva su comprobación histórica cuando V1 está instalada y, tras CT232,
   comprueba V2. La segunda exige V1 ausente y V2 intacta; usa las publicaciones
   sintéticas existentes, sin escribir historia.
   La antigua prueba de CT218 se retira porque invocaba V1, que ya no existe.
   CT226 comprueba que la provisión común se acepta para una organización sin
   publicación propia, que la publicación vigente se acepta para su organización
   y que una publicación ajena o anterior se rechaza. CT232 comprueba además
   la ausencia de V1 y que V2 conserva su definición y permisos; sus casos
   rechazan huella, versión y fuente distintas de las publicadas y una
   organización ausente. Así permanecen cubiertos los cotejos de la prueba
   retirada.
4. Con el catálogo publicado y su fichero configurado, arrancar el portal y
   recuperar un documento desde la ficha. Comprobar que el PDF y los recibos
   originales siguen disponibles tras reiniciar aplicación y PostgreSQL.
   Este corte no necesita activar la gestión administrativa opcional de
   plantillas ni cambiar sus permisos.

## Comprobación realizada

CT232 se instaló una vez en la copia H sobre CT226. Dos revisiones
independientes fueron favorables; CT226 y CT232 pasan, con dos positivos y
seis rechazos en la prueba nueva. V2 conserva definición y permisos exactos;
no cambian los cinco registros de firma, sus cinco custodias ni el catálogo.
Semgrep local sobre los tres SQL no encuentra incidencias con las reglas de
retirada de tablas y concesiones públicas; la revisión SQL cubre el resto.

El binario conjunto congelado anterior a la corrección de CT228, con su web
del mismo origen, arrancó antes y después de retirar V1. El fichero documental
de plantillas estaba seleccionado. Chrome recuperó desde la ficha el PDF de
56.399 bytes antes y después de reiniciar PostgreSQL y la aplicación, con la
misma huella. Las tres consultas posteriores devuelven 200 y conservan los
recibos y fechas de la comprobación anterior de F1-64. No hubo errores de
página, cookies ni desbordamiento horizontal a 390 píxeles.

La gestión administrativa de plantillas no estaba montada: el perfil de
firma obtuvo 401 y el perfil general, en otro contexto, 404. No se probó esa
pantalla ni se cambiaron permisos para abrirla. No se generó un borrador nuevo.
La prueba usa la composición de A, B, C y F1-64 anterior a la minimización;
no acredita la aceptación final de CT228 ni el expediente único de fase 1.

## Vuelta atrás y límite

Si falla el arranque o el recorrido, detener la candidata y restaurar la copia
anterior de base y binario mediante el procedimiento local vigilado. No ejecutar
DOWN ni recrear V1 a mano. La ampliación de plantillas a otra organización sigue
pendiente de un contrato de autorización propio; CT232 no la activa.
