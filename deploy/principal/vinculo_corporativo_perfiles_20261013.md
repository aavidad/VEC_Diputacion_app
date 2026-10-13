# Vínculos corporativos por perfil

CA43 permite mantener los vínculos de C3 y Auditoría en perfiles separados de
la misma cuenta. Cada consulta selecciona su perfil explícitamente y conserva
los cotejos de cuenta, persona, organización, procedencia y vigencia. No suma
permisos entre perfiles ni los concede al publicar un vínculo.

## Instalación y provisión

Aplicar una vez CA43 después de la PR F1: AD258 → CA42 → CA43. Su dependencia
anterior es la lista de Auditoría: AUT82 → CT208 → B109 → P47 → AD251.
`lista_sql_vinculo_corporativo_perfiles.txt` contiene sólo el incremento CA43;
no reaplicar las anteriores. Conservar copia fría y vuelta atrás del kit.

CONFIG NUEVA: ninguna. Se conserva `VEC_VINCULO_CORPORATIVO_DSN`, el LOGIN
exclusivo, la fila de fuente aprobada por DBA y el publicador de la PR F1.

Para el segundo perfil, aprobar una fuente distinta con su propia operación,
contexto y vínculo corporativo. Cuenta, persona y organización deben coincidir
con sus filas actuales de F1; no se copian del primer perfil. Usar las mismas
llamadas `--comprobar` y `--aplicar --huella` del guion F1. El permiso de
Auditoría se publica por ADMIN según su guion, de forma independiente: el
vínculo corporativo no sustituye la asignación de ese permiso.

## Qué conserva

El puntero añade `perfil_ref`, derivado de su versión histórica. Su clave y FK
compuesta incluyen cuenta, perfil, superficie, uso, referencia y versión del
vínculo. Las filas de historia permanecen intactas; se añade sólo el índice
único requerido para esa FK.

CA9, CA40, el acreditador de organización de destino de CA25 y el publicador
CA42 se reconstruyen desde sus cuerpos instalados con selección de perfil.
Conservan firmas, permisos y resultados. Las funciones SECURITY DEFINER
reconstruidas usan `pg_catalog, pg_temp`; la fuente de firmante de CA25 se
comprueba sin modificar su cuerpo. El JOIN del publicador de permiso CT usa
también el perfil explícito. No se editan migraciones anteriores instaladas.

## Evidencia local

CA43 final se instaló una vez sobre la copia posterior a CA42, en PostgreSQL
18, en 0,113 s. El vínculo C3 conserva el plan original y recupera idénticos
recibo y evento: comprobación 40,7 ms y replay 33,7 ms. El segundo perfil se
comprobó en 36,9 ms y se publicó en 40,4 ms; ambos punteros quedan activos.
Son tiempos de CLI y conexión, no de pantalla.

La prueba SQL terminada en ROLLBACK comprueba dos CA40 positivos, perfil
cruzado y contexto caducado denegados, recibos y FK. Un cambio del puntero a
un perfil ajeno falla por FK. Publicador y runtime siguen sin acceso directo
a la tabla. Los cuerpos previos y las ACL se cotejan por huella durante la
instalación; no se reaplicó SQL ni se ejecutó DOWN.

La regresión de replay cambia sólo la referencia de contexto y, después, sólo
la procedencia acreditada en versiones de ensayo. Ambos casos se deniegan y
el ROLLBACK conserva puntero, prueba y recibo original. Se ensayó sobre una
copia limpia con el cuerpo final de CA42 cotejado por su huella, evitando una
versión provisional que carecía de cinco comparaciones de replay.

El runtime legado de CA9 de esta copia no acredita su configuración previa;
no se declara una llamada nominal positiva a CA9 ni se modifica esa guarda.
El cotejo CA40 de las lecturas C3 y Auditoría sí se comprobó con ambos perfiles.

Dirección instala y prueba los GET de C3 y Auditoría en Chrome con sus perfiles
publicados y registra los tiempos de pantalla. Este ensayo no acredita ese
recorrido ni la verificación offline completa de las familias nuevas.
