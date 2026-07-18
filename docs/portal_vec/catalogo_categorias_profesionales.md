# Catalogo gobernado de categorias profesionales

Fecha de referencia: 19-07-2026.

## Estado y alcance

El inventario consolidado contiene 68 categorias profesionales: 5 de
Administracion general, 60 de Administracion especial y 3 de organismos
dependientes. Las 58 entradas recuperadas del arbol historico OPES se han
complementado con diez denominaciones constatadas en bases publicas del BOP de
Granada. No constituye una relacion oficial vigente ni ha sido aprobado por
Recursos Humanos. Por tanto, se usa exclusivamente como contenido de
demostracion y no tiene validez administrativa.

Las huellas de las fuentes revisadas son:

- migracion OPES: `4c94c36a2f024edda8b0c4d7c0cec965b97096f0ffbc64df3e13f64dad568b1b`;
- documento descriptivo OPES: `b276518116527f357dcbbab992929f277fd69018b059de41ba72555812ed9af1`;
- inventario de fuentes publicas BOP:
  `de9af856fea93e91340e77aef6403d607e49b3822e5d8f7856bca4a5d6ad5172`;
- catalogo canonico resultante:
  `b800a7e9c306fa8027709cfb4304cc8ccf8065f888673da71bd73a138c519233`.

No se han incorporado bases de candidatos, DNI, nombres, correos, telefonos ni
otros datos personales encontrados en aplicaciones auxiliares.

## Decision de arquitectura

Las categorias no pertenecen en exclusiva a Bolsa. Las consumen tambien
Personal, RPT, certificados, baremacion y preferencias de avisos. Se adopta
como unica autoridad el catalogo configurable y gobernado del nucleo, fijado
por ID, version y huella SHA-256 exactos. El bootstrap construye una sola
instantanea inmutable y la comparte, mediante puertos y proyecciones
minimizadas, con Bolsa publica y Personal. Ninguno de esos consumidores
resuelve implicitamente «la ultima version» ni mantiene una lista propia.

```text
paquete de importacion revisable
              |
              | importacion administrativa expresa
              v
catalogo gobernado, versionado y publicado
              |
      +-------+---------+------------+----------------+
      v                 v            v                v
    Bolsa            Personal       RPT      certificados/avisos
```

El paquete `data/catalogos/categorias-profesionales/v1.demo.json` es una
instantanea reproducible para desarrollo y demostracion. No se siembra en el
almacen al arrancar ni sustituye al gobierno del catalogo. Las 68 entradas
siguen siendo un inventario historico DEMO pendiente de contraste y aprobacion
por RRHH. En produccion, una version se publicara por el caso de uso
administrativo, con autenticacion alta, doble control, auditoria, aprobacion y
recibo. PostgreSQL y un futuro Oracle deben implementar el mismo puerto.

El antiguo snapshot durable de Personal puede contener una coleccion
`categories`. Se conserva inerte al leer y volver a escribir el fichero para no
destruir datos heredados, pero ya no se carga como autoridad consultable ni
admite altas, cambios o bajas. El workspace tampoco embebe la lista historica.

En este envoltorio DEMO, `actualizada_en` indica cuando se genero la
instantanea tecnica (19 de julio de 2026); no es la fecha de una actualizacion
oficial de RRHH ni de la fuente OPES. Una importacion productiva separara fecha
de fuente, fecha de extraccion y fecha de publicacion gobernada.

## Claves y compatibilidad

Las claves publicas son estables, en minusculas y formato kebab-case. Los 36
procesos publicos de la muestra proceden de 37 publicaciones BOP; una misma
publicacion puede referenciar varias categorias y una categoria puede aparecer
en varios procesos. La muestra expone 35 categorias con al menos un proceso y
el directorio conserva las 68 entradas, incluidas las que no tienen convocatoria
en este corte. La muestra no convierte esos usos en autoridad del catalogo.

Las variantes antiguas con guion bajo no vuelven a publicarse. Los alias de
directorios historicos se resolveran unicamente durante una importacion o
migracion gobernada, con destino existente y sin ciclos; no se exponen como
identificadores alternativos de la API.

## Proyecciones publicas

La consulta de convocatorias y el directorio responden a necesidades distintas
sin crear dos autoridades:

- el filtro muestra solo categorias presentes en procesos que cumplen los
  demas filtros, con su numero de resultados;
- el directorio muestra todas las entradas vigentes y publicables del catalogo,
  incluidas las que no tienen procesos en ese momento;
- los conteos se calculan desde las convocatorias, nunca se guardan en el
  catalogo;
- el area, la etiqueta, la descripcion, el orden y el caracter suscribible se
  proyectan desde la version gobernada;
- rutas locales, fuente interna de cada entrada, actores, motivos y aprobaciones
  no salen por la API publica.

La suscripcion no se simula en el portal publico. Se gestionara desde el area
personal autenticada, con consentimiento, canales configurables y justificante
auditado.

## Proyeccion interna compatible de Personal

`GET /api/vec/personal/categories` y
`GET /api/vec/personal/categories/{slug}` conservan las rutas y los campos de
compatibilidad que necesita el cliente existente (`slug`, `name`, `source`,
`module_key`, `state` y `usage`), pero los proyectan desde la misma autoridad
ID/version/huella que Bolsa. La respuesta incorpora la referencia exacta del
catalogo. No expone `source_path`, rutas locales, alias, actores, motivos ni
aprobaciones.

La consulta sigue admitiendo paginacion y filtros de texto y area. Las
operaciones directas `POST`, `PUT` y `DELETE` permanecen reconocidas por
compatibilidad de transporte, pero responden `409 Conflict` con el codigo
estable `catalogo_gobernado_requiere_borrador`. No mutan el catalogo heredado,
no crean una auditoria de exito y no se reinterpretan como una publicacion.

La futura administracion se implementara como un flujo diferente: nueva
version en borrador, referencia a la version de partida, motivo y fuente,
validaciones, doble aprobacion por personas distintas, firma/publicacion y
recibo auditable. Hasta que ese caso de uso exista y haya contenido aprobado
por RRHH, la pantalla interna es exclusivamente de consulta y no atribuye
validez administrativa a las 68 entradas DEMO.

## Seguridad e integridad

El adaptador local de demostracion debe fallar cerrado ante fichero ausente,
tamano excesivo, claves JSON duplicadas, campos desconocidos, catalogo no
publicado, version distinta, huella invalida, entrada caducada o referencia de
convocatoria desconocida. Mantiene una instantanea inmutable y no realiza
escrituras.

Cada publicacion de convocatoria conserva `catalogo_id`, `catalogo_version` y
`catalogo_huella_sha256` junto a sus claves profesionales. Esos tres valores
forman parte de la huella publica de la convocatoria. Antes de montar las rutas,
el bootstrap obtiene la version seleccionada y coteja anticipadamente todas las
convocatorias: ID o version inexistentes, huella distinta o clave desconocida
impiden el arranque.

La respuesta publica incluye la referencia, version y huella SHA-256 del
catalogo. Esa huella permite identificar la instantanea, pero no equivale a
firma administrativa ni acredita que RRHH haya aprobado su contenido.
`origen_sha256` identifica la fuente historica declarada; no autentica ni firma
por si mismo el paquete DEMO.

## Criterios de aceptacion

- el paquete inicial contiene exactamente 68 claves unicas, distribuidas
  5/60/3 entre Administracion general, Administracion especial y organismos
  dependientes;
- el directorio publico devuelve esas 68 entradas sin metadatos internos;
- Bolsa y Personal devuelven la misma referencia de catalogo, version y huella;
- la proyeccion de Personal mantiene el contrato de lectura sin publicar
  `source_path`;
- las mutaciones directas de Personal responden `409` y no alteran ni el
  catalogo gobernado ni las categorias heredadas del snapshot;
- con la fuente publica DEMO actual, 37 publicaciones BOP se proyectan en 36
  procesos y cero plazos declarados abiertos; el numero de facetas se verificara
  al cerrar la nueva instantanea canonica;
- una convocatoria no puede referenciar una categoria ausente, retirada o no
  vigente;
- publicar una version que incorpore una categoria adicional permite verla sin
  modificar ni recompilar Go, HTML, CSS o JavaScript;
- la nueva categoria solo aparece como faceta cuando un proceso publicado la
  referencia;
- el paquete demo y la API no presentan coincidencias en los patrones
  automatizados definidos para datos personales y secretos.

## Pendiente antes de produccion

1. Obtener de RRHH la relacion oficial, resolucion o fuente maestra vigente.
2. Comparar altas, bajas, denominaciones, areas y equivalencias con las 68
   entradas recuperadas.
3. Generar una propuesta de version y un informe de diferencias.
4. Aprobar y firmar la publicacion mediante doble control.
5. Importarla expresamente al repositorio durable y conservar el recibo.
6. Implantar la gestion administrativa por borrador y doble aprobacion; hasta
   entonces no existe una operacion de escritura productiva sobre categorias.
7. Definir y ejecutar, con informe y respaldo, la retirada futura de las
   categorias heredadas conservadas de forma inerte en snapshots de Personal.
