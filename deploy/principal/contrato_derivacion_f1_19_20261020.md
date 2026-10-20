# Autorización del expediente hijo (F1-19)

AD305 corrige el contrato
existente y debe integrarse después de AD301. El positivo de la derivación y
la activación conjunta con el consumidor CT247 siguen pendientes.

La decisión firmada vigente contiene la referencia del recurso y la huella
de su contexto. Los ámbitos forman parte de ese contexto; no aparecen como
un campo `ambitos` de la decisión. La rama de derivación de AD291 y su fachada
AD292 exigen ese campo inexistente y rechazan una decisión válida.

El corte elimina solo esa exigencia en la rama
`ct_nombramiento_derivado` y en la fachada existente. Conserva audiencia,
operación, finalidad, tipo de recurso, garantía alta con certificado, recurso
hijo, huella firmada, campos vacíos, obligación de auditoría y consumo en la
misma transacción. No cambia el formato de la decisión ni las otras ramas.

El consumidor de Contratación obtiene organización y unidad
del material canónico inmutable del hijo. Recalcula con esos valores y su
referencia el contexto autorizado, y coteja su huella con la decisión y el
acuse. Una unidad ajena debe quedar denegada, aunque coincida el padre.

## Orden de activación y comprobación pendiente

1. Integrar AD301 y obtener las definiciones instaladas del núcleo y AD292 en
   una copia propia de la principal. Fijar sus huellas y permisos en AD305.
2. Instalar AD305 una sola vez, al final de la lista SQL vigente. No reaplicar
   AD291 ni AD292; no cambia claves, concesiones ni asignaciones de personas.
3. Componer con CT247. Su publicación inicial admite solo v6 con autoridad
   propia. Las versiones siguientes, firmas y una nueva selección conservan
   sus guardas. El rol de RRHH y su catálogo se publican por los actos existentes.
4. Recorrer desde la oferta adjudicada la derivación con recibo de cada hijo,
   la repetición con la misma intención y la denegación de ámbito ajeno.
   Reiniciar y cotejar la misma referencia y el mismo recibo sin duplicados.
5. Conservar el arranque sin la función opcional y con su configuración propia.
   Una preparación parcial no permite presentar el corte como listo.

El ensayo positivo requiere el consumidor CT247 y la configuración admitida;
las pruebas de contrato y de denegación del núcleo por sí solas no lo sustituyen.

## Definiciones y prueba focal

Las preimágenes se extrajeron de PostgreSQL 18.4 propio, con la fuente base
`a8e4f8994` (main y la candidata AD301/BC17) y 133 migraciones instaladas.
La migración incluye las dos definiciones estáticas completas, generadas
desde las instaladas: cinco líneas retiradas y comparación de cuerpos, configuración,
propietarios, permisos y dependencias. La lista de audiencias se conserva.

| Función | Antes | Después |
| --- | --- | --- |
| Núcleo | `49d6559e7aa6e5575c630531cf5c2142a67677e5cf7d3d38da3f04cab7253035` | `58c98915e85aec4ff54d45864df4f733cd013a5a2be5863482cfce96fc0c0c46` |
| Fachada AD292 | `9d1e4fc89863e7fa8a99e1353826be997107571f746c6d3a32b7f29f13b1a1d1` | `2f00e33d4778828eaf25ef2c8d48eabef7d846de60bc86ccb4c885253188b3b5` |

Tras instalar AD305 una vez, ejecutar las pruebas SQL existentes de AD301 y
AD292. La primera conserva su contrato de publicación con la nueva huella
global; la segunda coteja la fachada, la denegación de material ajeno y la
ausencia de consumo en el rechazo. Las dos quedan en la puerta de fase 1.
Estos casos no emiten decisiones falsas ni acreditan un acto permitido.

El archivo SQL generado tiene 3500 líneas; `ad305_cuerpos.diff` muestra las cinco líneas retiradas
en 12 líneas de diferencial. La línea base de tamaño registra ese
cuerpo completo exigido por la guarda estructural, sin ampliar el alcance.

## Perfil de RRHH para derivar

Rol ordinario
`rrhh_derivacion_nombramiento`, versión 1, «Derivación de nombramientos».
La fuente `fuentes/contratacion_temporal_derivacion_nombramiento_v1.json`
contiene una sola concesión de derivación, garantía alta, auditoría y los
ámbitos de organización y unidad. Su vigencia es de 86 días. La clase del
catálogo es `ordinario`; los campos permitidos quedan vacíos.

La fuente por sí sola no publica un rol ni concede acceso. Preparar un paquete
AUT58 con el catálogo vigente y el censo completo; publicar su admisión con el
plan aprobado. Dos personas administradoras distintas proponen y confirman
el rol por AUT60, usando el conjunto 7 de AD304. Obtener las referencias nuevas
del perfil y su vínculo en la preparación de lote de Administración y otorgarlo
por el lote ordinario F1-104, con los dos ámbitos concretos. Conservar los recibos.

El perfil debe pertenecer a la misma cuenta de RRHH y ser distinto de los de
alta y consulta. Su referencia confirmada y la versión y huella del rol publicado
se trasladan a `derivacion` en la configuración de nombramientos de CT,
junto con el motivo admitido y las dos conexiones existentes y distintas.
La admisión, publicación, asignación y el recorrido positivo siguen pendientes.
