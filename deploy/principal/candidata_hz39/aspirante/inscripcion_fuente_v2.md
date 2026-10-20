# Fuente v2 de inscripción del aspirante (F1-101)

Desde F1-40 y F1-41 la aplicación pide dos datos más en la lista de
convocatorias abiertas y en su ficha: `modalidad_inscripcion` y
`solicitudes_propias[]` (categoría, solicitud y estado). La autorización de
lectura exige que la concesión publicada tenga exactamente los campos que pide
la aplicación. Con la fuente v1, que no los tiene, la lista y la ficha
respondían 403 aunque el aspirante ya tuviera las acciones de inscripción.

## Qué cambia

- **Fuente v2**: [`inscripcion_bolsa_actos_v2.json`](../../fuentes/inscripcion_bolsa_actos_v2.json).
  Misma referencia (`fuente:bolsa:inscripcion_recorrido_20261013`) y las mismas
  ocho acciones en el mismo orden, todas en versión 2. Solo
  `bolsa.inscripcion.convocatorias.listar` y
  `bolsa.inscripcion.convocatoria.consultar` ganan las cuatro hojas nuevas; las
  otras seis, incluidas las tres de RRHH, conservan los campos de la v1.
  Vigente desde el **20 de octubre de 2026 a las 00:00 UTC** hasta antes del
  **13 de octubre de 2027 a las 00:00 UTC**, la misma fecha final que la v1.
  Huella del contenido canónico:
  `3185be5620b770381f77189e561dc0f3e5b90ee1c9a9e7094742c2c29cc688e0`; SHA-256
  del archivo: `c832315ad79869b9dd24a5375c2d63f5cb9cfe37e421c81d0cd02e149a1a8507`.
  La v1 no se toca.
- **Dominio**: el plan de la versión del rol acepta para cada acción de
  inscripción los campos de la v1 o los de la v2 completos; una proyección a
  medias se rechaza, y la lista y la ficha de convocatorias tienen que salir
  de la misma versión.
- **AUT105** (`autorizacion/000105_fuente_inscripcion_v2.up.sql`): la misma
  regla en `validar_plan_version_rol_bolsa_externa_v1`. Las cinco altas del
  aspirante salen de la fuente v1 (huella `07bf…`) o de la v2 (huella
  `3185…`), nunca mezcladas. No crea acciones, audiencias, transiciones ni
  permisos. Coteja la preimagen de AUT104 (`41b606db…`) y deja la postimagen
  `c2a42fc4…` con el mismo OID, propietario, configuración y ACL. Va al final
  de `lista_sql_main_tras_hz37.txt`, después de AUT104. Solo UP.

## Activación (dos actos de Administración)

Nada se escribe por SQL directo ni se rebaja la coincidencia exacta de campos.

0. **Solo en una instalación desde cero: el rol de RRHH primero.** Publicar
   la versión del rol de RRHH con motivos, decidir e incorporar desde el
   catálogo v6, que trae la fuente v1, antes de admitir el v7.
   `validar_plan_version_rol_bolsa_v2` (AUT84) sigue exigiendo la huella de la
   v1 y con el v7 ya no podría. Las copias locales ya traen ese rol en v10.
1. **Admitir la fuente v2.** Se publica una versión nueva del catálogo
   `catalogo:bolsa:carga_convoca:b1` con todo lo vigente y la fuente v2 en el
   sitio de la v1 (un catálogo no admite dos entradas con la misma acción). La
   aprobación técnica del DBA liga la huella del plan; después
   `vec-catalogo-acciones-admitir --fase aplicar`. La repetición con el mismo
   plan devuelve el mismo recibo.
2. **Publicar la versión del rol del aspirante**
   (`candidato_bolsa_portal_historial_propio_desarrollo`) con las cinco
   acciones del aspirante del catálogo nuevo, en el orden de
   [`gobierno_inscripcion.md`](../../inscripcion_recorrido_20261013/gobierno_inscripcion.md).
   El administrador 1 propone en
   `/api/admin/perfiles/v1/gobierno-version-bolsa-externa/propuestas` y el
   administrador 2 cierra en `…/cierres`, cada uno desde su Chrome con su
   certificado y el perfil «Administración de la aplicación».

Con el rol en su versión base (siete concesiones), el plan añade las cinco
acciones con los campos v2 y el rol pasa a v2. Si el rol ya tiene las
preferencias, pasa a v3 igual que antes.

## Evidencia en copia local (20/10/2026)

Copia con `--modulos --portal-externo --administracion` y AUT105 en la lista:

- Catálogo v6 → v7: 22 entradas, 67 perfiles. Admisión aplicada y repetida
  con el mismo recibo y la misma fecha.
- Propuesta del administrador 1: 201; repetición 200 con la misma referencia.
  Cierre del administrador 2: 200; repetición 200 con el mismo recibo y fecha.
  El rol del aspirante pasa a v2 con doce concesiones y la asignación se mueve
  a v2.
- El aspirante, desde Chrome con su certificado: convocatorias abiertas 200 con
  `modalidad_inscripcion`; sus solicitudes 200.

El recorrido completo del aspirante en pantalla, con reinicio, es F1-29.

## Límites

- **Rol ya publicado con la fuente v1.** Un entorno cuyo rol del aspirante ya
  lleve las cinco acciones de la v1 (por ejemplo, una copia en la que se hizo
  el acto de F0-28) no puede pasar a la v2 con este plan: haría falta una
  transición de sustitución como la de modalidades de preferencias. Las copias
  nuevas y la instalación de producción, que empiezan en la versión base, no
  lo necesitan.
- **Rol de RRHH.** `validar_plan_version_rol_bolsa_v2` (AUT84) sigue exigiendo
  la fuente v1 para añadir motivos, decidir e incorporar; por eso el paso 0 va
  antes de admitir el v7. Añadirlas después desde un catálogo con la v2
  necesitaría el mismo cambio en esa función.
- Las herramientas que preparan el plan del catálogo y el de la versión del rol
  son privadas y no están en el repositorio.
