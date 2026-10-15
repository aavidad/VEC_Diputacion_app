# Activación del justificante de renuncia de Bolsa

Esta guía corresponde a la entrega en preparación de Codex-O. Su alcance es
únicamente el PDF de renuncia justificada que la persona aspirante aporta desde
Mi Bolsa con recibo y que una técnica de RRHH con permiso vigente consulta y
descarga desde la ficha del candidato. El seguimiento muestra la actuación.
La persona aspirante conserva el original: Mi Bolsa no ofrece una descarga
posterior de su PDF. El límite es de 16 MiB en el cliente; la API rechaza con
413 un cuerpo demasiado grande. El documento debe superar el antivirus antes
de quedar disponible.

## Estado de la entrega

El código y las migraciones de esta rama no acreditan una instalación. Están
pendientes la revisión SQL independiente, la revisión de seguridad, el ensayo
en una copia de la principal, el arranque con y sin configuración, el recorrido
en Chrome y la recuperación después de reiniciar. Dirección incorporará las
huellas, recibos y el resultado de esas comprobaciones a la entrega final.
Ningún paso de esta guía autoriza a probar ficheros sueltos en cidonia.

## Preparar una copia y fijar el orden

Dirección congela el árbol fuente y ensaya la misma candidata que instalará.
Comprueba el estado de migraciones en la copia y usa la futura lista
`deploy/principal/lista_sql_codexo_justificante_limpio_20261015.txt` para
documentar cada archivo, orden y huella antes de ejecutarlo. No se reaplica
una migración ya instalada.

El orden de dependencia es AD277 → AUT95 → D20 → M7 → creación de los LOGIN
documentales sin membresías → D22 → membresías gobernadas → B119 →
CT222. AUT99, AD281 y D23 corrigen la autoridad externa de los actos propios y se intercalan según las
precondiciones y el orden exacto que confirme la lista SQL revisada. Antes de
instalar se cotejan los números reservados, los roles y las funciones presentes
en la copia. La creación de LOGIN técnicos y sus credenciales queda en el
circuito de DBA; los secretos no entran en Git.

La base documental conserva metadatos, historia y recibos; el almacén local
custodia los bytes. B119 vincula el documento con la respuesta de renuncia.
La descarga de RRHH exige una decisión de autorización para esa operación,
consumida por su función nominal, y deja constancia de la lectura. CT222 acota
la frontera con el expediente de Contratación temporal. Las migraciones nuevas
AUT99, AD281 y D23 siguen pendientes de revisión y ensayo.

## Configuración y gobierno

Se preparan fuera de Git los materiales privados que lee la composición
externa de justificantes y el almacén: `externo/documentos/justificantes-bolsa.json`,
sus claves, la conexión documental, el directorio de custodia y el servicio de
antivirus con su confianza TLS. Se usan ficheros privados con permisos `0600`
y LOGIN distintos según la función. Los valores concretos se derivan del
inventario de la copia; no se copian credenciales ni rutas de otra instalación.

Para la lectura interna se prepara
`identidad/bolsa-justificantes-rrhh.json` dentro del material privado de
desarrollo, con **solo** estas tres claves:

```json
{
  "version": 1,
  "superficie": "interna_corporativa",
  "dsn_documentos": "<DSN privado del LOGIN documental de RRHH>"
}
```

El valor de `superficie` coincide con el contrato Go vigente. El DSN corresponde al LOGIN técnico
`vec_interno_documentos_bolsa_rrhh`, miembro únicamente del grupo documental
previsto. El fichero no contiene cuenta, certificado, persona ni perfil
asignado para elegir qué técnica puede consultar. Cada petición identifica a
la persona y comprueba su permiso vigente por rol.

La publicación de acciones, versiones de rol y membresías personales se hace
por los actos de Administración gobernados, con dos administradores distintos
cuando corresponda. Se conservan el catálogo y las concesiones existentes; se
añade la acción específica para consultar o descargar el justificante. No se
insertan ni modifican directamente filas de datos, asignaciones o permisos.
Los datos ficticios necesarios para el recorrido se preparan con la semilla o
el guion privado de datos, fuera de esta activación.

## Ensayo de la candidata completa

1. En la copia, instalar SQL en el orden revisado, preparar configuración y
   ejecutar los actos ADMIN. Guardar acuses, versiones y huellas sin secretos.
2. Arrancar sin el material propio de justificantes. Comprobar que VEC atiende
   las demás funciones y que solo estas rutas responden 503. El registro debe
   indicar etapa y clase segura, sin DSN, claves ni datos personales.
3. Añadir los ficheros privados y reiniciar de forma vigilada. Comprobar que la
   función queda disponible y que el resto conserva su estado.
4. En Chrome del sistema, entrar como aspirante ficticio, subir un PDF de hasta
   16 MiB desde Mi Bolsa, confirmar una renuncia justificada y conservar el
   recibo. Comprobar que no aparece una descarga propia del PDF. Probar el
   rechazo de un archivo mayor y el 413 de la API.
5. Entrar como Carmen y como otra técnica ficticia con el mismo permiso
   vigente por rol. Desde la ficha del candidato, consultar y descargar el
   mismo PDF y comprobar el seguimiento y la auditoría de lectura. Retirar el
   permiso a una de ellas por el acto ADMIN y verificar la denegación.
6. Reiniciar aplicación y PostgreSQL en la copia. Repetir la consulta del
   recibo, la lectura autorizada y la descarga. Verificar también el rechazo
   seguro de un PDF que el antivirus no admita.

Una incidencia se corrige en la candidata, se congela de nuevo y se repite el
ensayo completo. Solo Dirección instala en cidonia el contenido exacto que
superó estas comprobaciones, con reinicio vigilado y vuelta atrás preparada.
