# Recorrido de H — F1-27

En curso, 21 de octubre de 2026. Fuente de ejecución:
`e5601ab663723da8dfc9b7346ff8a59b28c8558e`, main con la propuesta desde Mi
Bolsa y su aceptación reconocida por Personal (F1-127 y F1-134). Se crea con
la receta completa, sin componer ramas pendientes ni modificar G.

Dirección indicó el 21/10 a las 06:17 usar main para los pasos 12 en adelante.
La receta exige otra instancia cuando cambia la revisión. H recorre los
pasos con actos propios para preparar el expediente y continuar
la llamada desde las 09:00 de Madrid, con la franja de 09:00 a 14:00 intacta.

La preparación previa rechazó una carpeta de fuente cuya extracción aún no
había terminado, antes de crear la instancia o aplicar SQL. La ejecución
completa se inició después, con la revisión limpia y subida.

| Tramo | Estado |
| --- | --- |
| Creación completa y carga nativa de Bolsa | Completada en 256,37 segundos |
| Petición, ratificación y alta (1–3) | Confirmadas en Chrome; expediente 2026/95032 |
| Análisis, vía, asignación e informe (4–7) | Confirmados; versión 5 |
| Dos firmas del informe y fiscalización (7b–8) | Primera firma registrada; segunda rechazada; fiscalización pendiente |
| Llamamiento, contactos y aceptación (9–11) | Pendiente de la franja de llamadas |
| Propuesta, resolución, documentos y tres firmas (12–15) | Pendiente |
| Reinicio y recuperación de lecturas y recibos | Pendiente |

La primera firma del informe devolvió 201 con verificación técnica válida.
La segunda, de Jefatura, obtuvo el original y la comprobación previa con 200;
la aplicación de firma completó la firma del PDF, pero el único envío para
registrarla devolvió 403 el 21/10 a las 06:34:12 de Madrid. PostgreSQL rechazó
el registro con 42501, «contexto nominal de firma divergente», en
`registrar_firma_verificada_v3`, llamada desde `registrar_firma_con_plan_v4`.
El expediente conserva una firma registrada. No se repitió el envío ni se
avanzó a fiscalización; la solicitud y el PDF firmado se conservan fuera de Git.

Dirección indicó a las 06:46 diagnosticar en solo lectura y comparar los campos
del contexto calculado por PostgreSQL con el material de la aplicación. La
comparación exacta del intento fallido aún no está acreditada: ese material
interno no está en la petición HTTP y la transacción se deshizo. La definición
instalada y el error se conservaron para continuar el diagnóstico. No hay una
corrección aplicada ni un resultado positivo de la segunda firma en H.

Se reutilizan la [receta pública](../recorrido_copia/LEEME.md), la carga normal
del XLS de Administrativo y el preparador de los cuatro entornos de firma.
H usa Yago y las referencias de su propia carga; no traslada actos, claves ni
recibos de otras copias. Los materiales privados permanecen fuera de Git.

G permanece parada y preparada hasta el paso 8 como respaldo. F se retiró por
su gestor al quedar superada; sus recibos, capturas y documentos se conservan.
Las hojas de [F](recorrido_f_f1_27.md) y [G](recorrido_g_f1_27.md) registran esos
ensayos anteriores. Esta hoja no cierra F1-27 ni F1-28.
