# Confirmación de una aceptación directa de Bolsa

Tarea F1-71. Fuente reconstruida desde el consumidor central instalado tras
AD279, cotejado con main `c3b2fcfe4b`. AD288 se instaló en la copia conjunta
`601686060`; sigue pendiente el ensayo positivo con B128 y la pantalla de S.
No activar fuera de esa candidata.

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
2. Usar la lista canónica `lista_sql_main_tras_hz37.txt`: la lista completa
   de main tras HZ37 más las SQL propias, con AD288 en su posición exacta.
   Cotejar las preimágenes que figuran al final de esta guía;
   la instalación se detiene si difieren. Preserva las capacidades anteriores.
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

La función central previa se extrajo por lectura de la copia propia con AD279
instalada. La restricción nueva se normalizó en una tabla temporal con ROLLBACK;
no se alteró el núcleo para calcularla. Preimagen central:
`a46491328a00b33d5d9276c9fa6b4eec6287ba015d865df14aed807fe2d58fb2`; posterior prevista:
`edfe261f77e3ce275bc1930e16b56b2c73d3bd3b2f73a8412455bb1f3cd57037`. Restricción previa:
`504fd62c39258745ea34f9a7033e8993a2b44d78feed7eab2e3ffa01f0a94b15`; posterior prevista:
`c4888893d1951ec31f7674fb8e145a10af263e94c7d47a76ee749bd82b38c109`.

AD288 está instalada en las dos copias nuevas de `601686060`, cada una con
98 SQL; ambas arrancaron y reiniciaron con la comprobación pública del guion.
Las focales AD291, AD292 y CT238 pasaron allí con ROLLBACK, pero no prueban
la aceptación de F1-71. La fuente tiene dos revisiones favorables y se integró sin cambios de bytes.
Faltan el ensayo positivo conjunto con B128 y la aceptación desde la pantalla
de S. El focal
`ad288_confirmacion_aceptacion_bolsa.sql` comprueba los permisos nominales y
la denegación temprana sin consumo; no sustituye las pruebas firmadas de B128. Las 3397 líneas de SQL incluyen la reconstrucción
literal del cuerpo instalado; el cambio de su lógica se limita a las mismas dos
alternativas nominales de la preparación anterior. No se pierde la consulta de
preferencias incorporada por AD279.
