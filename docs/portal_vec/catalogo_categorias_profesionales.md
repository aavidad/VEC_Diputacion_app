# Catalogo gobernado de categorias profesionales

Fecha de referencia: 16-07-2026.

## Estado y alcance

El inventario recuperado contiene 58 categorias profesionales: 5 de
Administracion general y 53 de Administracion especial. Procede del arbol
historico OPES y coincide con la migracion local revisada, pero no se ha
contrastado todavia con una resolucion oficial vigente ni ha sido aprobado por
Recursos Humanos. Por tanto, se usa exclusivamente como contenido de
demostracion y no tiene validez administrativa.

Las huellas de las fuentes revisadas son:

- migracion OPES: `4c94c36a2f024edda8b0c4d7c0cec965b97096f0ffbc64df3e13f64dad568b1b`;
- documento descriptivo OPES: `b276518116527f357dcbbab992929f277fd69018b059de41ba72555812ed9af1`.

No se han incorporado bases de candidatos, DNI, nombres, correos, telefonos ni
otros datos personales encontrados en aplicaciones auxiliares.

## Decision de arquitectura

Las categorias no pertenecen en exclusiva a Bolsa. Las consumen tambien
Personal, RPT, certificados, baremacion y preferencias de avisos. Se adopta
como autoridad objetivo el catalogo configurable y gobernado del nucleo, con
version y huella exactas. La superficie publica de Bolsa ya consume una
proyeccion minimizada mediante un puerto. Personal y el workspace conservan
representaciones heredadas que deben migrarse antes de afirmar unicidad en
todo el portal.

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
almacen al arrancar ni sustituye al gobierno del catalogo. En produccion se
publicara por el caso de uso administrativo, con autenticacion alta, doble
control, auditoria, aprobacion y recibo. PostgreSQL y un futuro Oracle deben
implementar el mismo puerto.

En este envoltorio DEMO, `actualizada_en` indica cuando se genero la
instantanea tecnica (16 de julio de 2026); no es la fecha de una actualizacion
oficial de RRHH ni de la fuente OPES. Una importacion productiva separara fecha
de fuente, fecha de extraccion y fecha de publicacion gobernada.

## Claves y compatibilidad

Las claves publicas son estables, en minusculas y formato kebab-case. Las dos
convocatorias sinteticas usan:

- `auxiliar-administrativo`;
- `tecnico-de-gestion`.

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

- el paquete inicial contiene exactamente 58 claves unicas, distribuidas 5/53;
- el directorio publico devuelve esas 58 entradas sin metadatos internos;
- con la fuente sintetica actual, el filtro devuelve solo 2 categorias y un
  resultado para cada una;
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
2. Comparar altas, bajas, denominaciones, areas y equivalencias con las 58
   entradas recuperadas.
3. Generar una propuesta de version y un informe de diferencias.
4. Aprobar y firmar la publicacion mediante doble control.
5. Importarla expresamente al repositorio durable y conservar el recibo.
6. Migrar los consumidores heredados de Personal y retirar sus operaciones de
   alta, modificacion o borrado directo sobre un segundo almacen.
