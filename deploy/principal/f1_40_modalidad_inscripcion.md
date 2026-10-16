# Modalidad de inscripción en la preparación de bases

Este corte añade la elección de una solicitud por categoría o una para varias
categorías en la preparación existente. La guarda mediante
`ServicioPreparacionBasesV3.Guardar`, por la ruta nominal ya montada de Selección.
El material contiene el mismo `ContenidoPublicableConvocatoria` de Bolsa; no hay
lectura de tablas ajenas ni una segunda autoridad de publicación.

F1-40 permanece abierta: `ConfirmarPublicacion` no dispone de adaptador productivo.
Guardar la preparación no aprueba ni publica bases. La candidata de ensayo carga
la versión con modalidad como dato ficticio; no se presenta como aprobación.
La inscripción conjunta completa corresponde a F1-41.

## Configuración y datos

No hay variable, ruta ni acción nuevas. Se reutilizan los pools, material privado
y capacidad existentes de preparación de bases e inscripción. La lista y la
ficha del aspirante añaden, respectivamente, las hojas
`convocatorias[].modalidad_inscripcion` y `modalidad_inscripcion`. El filtro de
lectura compara todas las hojas del contrato con las de la concesión publicada:
un rol anterior sin esas hojas queda denegado y no recibe una respuesta recortada.
Para habilitar esas lecturas se deben admitir las dos hojas en las acciones
externas existentes del catálogo y publicar una nueva versión del rol de
aspirante con ambas concesiones explícitas, mediante propuesta y cierre de dos
administradores distintos. El versionador externo actual solo añade acciones
ausentes: la vía concreta para ampliar los campos de concesiones ya publicadas
queda pendiente de definir y ensayar. El cambio de código no publica el catálogo
ni concede campos por defecto.

El material debe ser propio de la copia y sus perfiles, certificados y confianzas
de desarrollo deben durar al menos 60 días. Ningún archivo de arranque se modifica
con el proceso en marcha: conservar la configuración anterior y reiniciar vigilado.

La pantalla existente es
`/portal-empleado/modulos/seleccion/preparacion-bases/`. Primero consulta la
preparación conservada. La persona elige la modalidad, pulsa «Revisar cambio» y
«Guardar modalidad». El cliente usa la revisión y huella consultadas, el material
y una clave de operación propia; el servidor resuelve identidad y ámbitos.
Una respuesta incierta conserva esa clave para recuperar el mismo resultado.
Un conflicto obliga a consultar la revisión actual antes de otra modificación.

Para el dato de convocatoria se reutiliza
`deploy/principal/inscripcion_recorrido_20261013/version_certificada_v2/generar.py`:
preparar su SQL y manifiesto fuera de Git y revisar las preimágenes de la copia.
La versión con modalidad es un dato de ensayo, no una publicación por el circuito
de aprobación. No se rodea el adaptador de publicación ausente.

## Instalación SQL

BC12 está reservada para esta rama. Reconstruye tres funciones existentes desde
su definición instalada, con comprobación anterior y posterior de huellas y
metadatos. No crea funciones, tablas, concesiones ni una migración del núcleo de
autorización. Usa `search_path=pg_catalog,pg_temp` en cada función con privilegios.

1. Comprobar en la copia PostgreSQL 18 las dependencias BC9 y B96 y las tres
   prehuellas declaradas en la migración. Si difieren, detener la instalación.
2. Instalar solo las ausentes, en el orden de
   `lista_sql_codexb_f1_40_20261015.txt`. No reaplicar una instalada ni usar DOWN.
3. Ejecutar la prueba focal SQL de BC12: legado y separada, conjunta visible,
   valor desconocido y nulo rechazados. Sus filas se revierten al terminar.
4. Comparar OID, propietario, ACL, configuración, volatilidad y paralelismo.
   Confirmar las poshuellas exactas de las tres funciones.

La modalidad se lee del contenido sellado y se incorpora a los lectores
existentes de lista y detalle. Ausente se interpreta como separada. Una conjunta
se puede consultar, pero el comprobador que B96 llama antes del primer INSERT
la rechaza con B9608. El adaptador devuelve `409 inscripcion_conjunta_pendiente`;
no se registra una solicitud parcial ni se presenta un recibo individual.

## Comprobación de la candidata

- Arrancar con la configuración completa de preparación e inscripción. Consultar
  material real, guardar una modalidad, recuperar el mismo recibo y comprobar
  que una revisión antigua produce conflicto sin sustituir la actual.
- Abrir en Chrome la lista y ficha de aspirante: ambas modalidades tienen nombre
  en el idioma activo. Una presentación individual de conjunta queda cerrada.
- Reiniciar aplicación y PostgreSQL; recuperar la preparación y recibo y volver
  a consultar la convocatoria con la misma modalidad.
- Arrancar sin la configuración opcional de preparación: su función queda
  cerrada y las demás superficies de la aplicación conservan el arranque.

Se reutiliza el recorrido HTTP de
`scripts/recorridos/seleccion/preparacion-bases/ensayar.py` para guardar, recuperar,
conflicto y reinicio. Las comprobaciones finales se consignan en la PR con su
cabeza exacta; esta guía no atribuye un ensayo aún no realizado.
