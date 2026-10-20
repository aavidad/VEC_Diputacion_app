# Derivación de nombramientos por plaza

Tarea F1-19, corte A: ámbito propio, canon y lector nominal del nombramiento
por plaza. El POST de derivación sigue cerrado a 503 hasta el montaje B.
La lectura nominal separa
las actuaciones del padre y el acto propio del hijo. El panel que las muestre
corresponde al corte siguiente. La propuesta, las firmas y la continuación del nombramiento quedan fuera de este corte.

Estado: pendiente de ensayo conjunto con la activación gobernada del perfil.
El recorrido positivo requiere además AD305 y el montaje B; este corte no
publica un rol ni activa la derivación.

## Activación de la copia

Usar una copia propia preparada con la orden completa de
[la guía de copia local](../LEEME_copia_local.md). Primero se instala la lista
de main y después CT247. Para activar el conjunto se necesitan también AD305
y el montaje B. No se reaplica una migración instalada.
CT247 comprueba las definiciones, propietarios, permisos y configuración
previos antes de reconstruir los lectores existentes.

La vía de Administración prepara, con dos administradores,
la admisión, la publicación del rol y su asignación. El perfil activo pertenece
a la misma Persona y Cuenta que la sesión RRHH; tiene una referencia propia,
distinta de los perfiles de alta y consulta. Su rol contiene una única
concesión:

| Campo | Valor |
| --- | --- |
| Rol | `rrhh_derivacion_nombramiento`, versión 1 |
| Acción | `contratacion_temporal.nombramiento.derivar` |
| Módulo | `contratacion_temporal` |
| Tipo de recurso | `nombramiento_derivado_ct` |
| Finalidad | `gestionar_contratacion_temporal` |
| Garantía mínima | `alto` |
| Campos permitidos | Lista vacía |
| Obligaciones | `auditar` |
| Dimensiones | `organizacion_ref`, `unidad_ref` |

Las referencias y huellas se toman de los resultados publicados, nunca se
inventan. Las vigencias del perfil y de las admisiones de desarrollo son de
60 a 90 días.

El material privado de la copia incorpora `identidad/nombramientos-ct.json`,
con estos campos:

- `esquema`: `vec.ct.nombramientos.activacion.v1`.
- `derivacion.perfil_ref`: perfil publicado para derivar.
- `derivacion.rol`: `rol_id`, `version`, `huella_sha256` publicados.
- `derivacion.motivo`: `catalogo_id`, `catalogo_version`,
  `catalogo_huella_sha256`, `entrada_clave` de un motivo vigente.
- `fuente_autorizacion_dsn_file` y `motivos_autorizacion_dsn_file`: rutas
  relativas distintas de conexiones nominales privadas.

Las dos conexiones tienen usuarios separados entre sí y de los consumidores
existentes. Las claves, conexiones, certificados, actos y recibos permanecen
fuera de Git. Después de cambiar la configuración se reinicia la copia y se
comprueba su arranque. Si falta el archivo o alguna dependencia, la derivación
responde 503 y el resto del portal puede arrancar.

## Qué debe comprobar el ensayo

Con la sesión RRHH y su certificado, solicitar por el contrato HTTP existente
la derivación de una plaza adjudicada de una oferta de varias plazas. La respuesta nueva debe ser 201, y repetir el mismo comando debe
devolver 200 con la misma derivación. La lectura nominal del hijo debe contener su número
por plaza, el recibo de derivación y la instrucción conservada, con su único
acto propio separado del historial del padre. Su presentación en la ficha se
comprueba en el corte del panel.

El recurso autorizado es el hijo y su contexto se calcula desde el material
inmutable de la derivación. Se comprueban una organización o unidad propia y
el rechazo 403 de una ajena. El centro copiado del padre no concede lectura
del hijo. Las escrituras y firmas posteriores a la versión inicial siguen
cerradas. Tras reiniciar PostgreSQL y la aplicación deben conservarse el hijo,
el recibo y su lectura nominal. También se comprueba el arranque sin el archivo
de activación, con la derivación cerrada.
