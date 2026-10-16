# Confirmación de una aceptación directa de Bolsa

Estado: preparación para revisión; no instalar. La fuente inicial parte del
consumidor real después de AD283. La candidata exige AD279 antes de AD288;
las huellas deben medirse otra vez sobre esa versión y revisarse antes del ensayo.

## Consumidor y alcance

La pantalla «Respuestas pendientes de revisión» del corte de S (#1164) confirma
una aceptación directa de B17. B128 conserva la respuesta, el cambio de situación,
la historia y el recibo en una misma transacción. Este corte aporta su consumidor
nominal de autorización y no crea otra consulta ni otra versión de lectura.

Reutiliza el perfil `situacion_participacion_bolsa`, el ejecutor nominal existente
y el patrón de B100. Solo añade la acción `bolsa.respuesta_revision.resolver`, su
audiencia, el tipo `respuesta_revision_bolsa` y la finalidad
`resolver_respuestas_bolsa_rrhh`, ligados a la participación y a la huella del
efecto. Campos y obligaciones son listas vacías. No crea LOGIN ni permisos
personales y conserva íntegra la lista anterior de audiencias.

Cada petición exige un consumo nuevo. Una decisión usada no se transforma en
autorización vigente: se rechaza y la transacción revierte. Repetir una operación
de negocio exige otra decisión comprobada y conserva el recibo de negocio
original, sin duplicar efectos. La huella de confirmación la construye el servidor;
no procede de atributos que el navegador pueda elegir.

## Activación prevista

Configuración nueva propia: ninguna. La candidata conjunta necesita las fuentes,
permisos y política del corte de S; este borrador no acredita que estén activados.

1. Congelar la cadena completa fijada por dirección, incluidos AD279 y el corte
   de S que contiene B128. No reaplicar migraciones ni usar una preimagen antigua.
2. Exportar el cuerpo instalado del consumidor central y su restricción de
   audiencias. Reconstruir desde ambos, preservando las capacidades anteriores.
3. Actualizar y revisar dos veces la fuente exacta, los permisos y las pruebas.
4. Instalar AD288 una sola vez en una copia nueva y ejecutar las pruebas focales.
5. Arrancar con y sin la activación del corte de S. Su ausencia debe cerrar
   únicamente la resolución; sesión, bolsas y funciones anteriores siguen.
6. Desde la pantalla, confirmar una aceptación válida. Recuperar el mismo recibo
   después de reiniciar. Probar decisión usada, revocada, ámbito ajeno, respuesta
   que no sea aceptación y conflicto de versión: sin efecto ni recibo de negocio.

La vuelta atrás usa la candidata anterior y su copia conservada. No hay DOWN ni
se altera una decisión ya consumida. Dirección integra; no se toca cidonia.

## Evidencia pendiente

La preparación de las huellas se calculó con objetos temporales y rollback sobre
la versión real posterior a AD283. AD288 no está instalada. Faltan AD279, las
pruebas focales completas, las dos revisiones de la fuente final y la aceptación
con el consumidor web de S. Las 3.369 líneas se deben principalmente al cuerpo
instalado del consumidor central, que debe reconstruirse entero y conservarse.
