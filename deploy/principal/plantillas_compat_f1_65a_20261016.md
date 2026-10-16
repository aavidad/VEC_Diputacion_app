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
4. Con el catálogo publicado y su fichero configurado, arrancar el portal y
   abrir «Plantillas» en RRHH. Descargar un borrador desde la ficha y comprobar
   que su documento y recibo siguen disponibles tras reiniciar. Repetir el
   arranque sin seleccionar la configuración opcional de plantillas: las demás
   funciones deben abrir. Usar el mismo binario y los mismos datos ficticios.

## Vuelta atrás y límite

Si falla el arranque o el recorrido, detener la candidata y restaurar la copia
anterior de base y binario mediante el procedimiento local vigilado. No ejecutar
DOWN ni recrear V1 a mano. La ampliación de plantillas a otra organización sigue
pendiente de un contrato de autorización propio; CT232 no la activa.
