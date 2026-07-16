# Versiones gobernadas de convocatoria de Bolsa

## Estado del corte

Fecha de referencia: 16 de julio de 2026.

El dominio de este documento está implementado en
`internal/modules/bolsa/domain` desde el commit `dde04bd`. La persistencia
PostgreSQL, la composición productiva y las superficies internas todavía no
forman parte del corte terminado. Hasta que esas barreras estén verificadas,
la aplicación solo puede publicar las convocatorias sintéticas marcadas como
demostración y debe fallar cerrada ante cualquier intento administrativo real.

## Problema que resuelve

Una solicitud, una autobaremación o una revisión técnica no pueden depender de
«las reglas actuales». Deben conservar exactamente las bases, calendarios,
catálogos, reglas, documentos y flujos que estaban vigentes para la versión de
la convocatoria en la que se registraron.

La solución separa tres conceptos que antes podían confundirse:

1. **Contenido semántico**: lo aprobado para una versión concreta.
2. **Gobierno administrativo**: borrador, publicada, sustituida o retirada.
3. **Fase del procedimiento**: inscripción, subsanación, alegaciones, cierre u
   otra clave administrable, obtenida de una instancia del flujo del proceso.

Una convocatoria cerrada no está necesariamente retirada. Cerrar es una fase
del procedimiento; retirar impide seguir usando una publicación administrativa
por una decisión expresa y trazada.

## Composición de la proyección pública

La representación pública compatible con `Convocatoria` se deriva de:

```text
versión publicada exacta
        +
instancia del flujo del procedimiento
        +
instante fiable de actualización
        =
proyección pública de la convocatoria
```

El JSON público no expone actores internos, expedientes, aprobaciones,
evidencias de autorización ni referencias de custodia.

## Contenido de una versión

`VersionConvocatoriaGobernada` conserva:

- identidad estable del procedimiento;
- secuencia técnica monotónica;
- código de versión que se muestra al público;
- referencia exacta a la versión anterior;
- contenido publicable sin fechas inventadas;
- expediente administrativo de procedencia;
- configuración exacta;
- revisión optimista del borrador;
- metadatos y evidencias de gobierno.

La secuencia técnica evita ramas aunque el código público sea `v2`, `2026.1` o
cualquier otra etiqueta válida. No se elige nunca una versión por ser «la más
alta» ni «la última».

## Dependencias exactas

Cada referencia de configuración contiene identidad, versión y huella SHA-256
del contenido. La versión fija:

- paquete de catálogos aplicable;
- calendario;
- conjunto completo de reglas de baremación;
- flujo del procedimiento;
- flujo de cada solicitud;
- documentos oficiales.

Antes de publicar, un verificador de dependencias debe demostrar que todas las
referencias existen, están publicadas, pertenecen al módulo y tipo correctos y
coinciden con sus huellas. Su evidencia queda vinculada a la huella semántica
de la convocatoria. La comprobación tiene una vigencia corta y se repetirá
dentro de la transacción durable.

Mientras catálogos, reglas y flujos no dispongan de persistencia durable o de
un paquete canónico autocontenido, se permiten borradores, pero no publicación
productiva.

## Documentos oficiales

Una URL no acredita un documento. Cada elemento público se vincula uno a uno
con:

- rol documental;
- referencia pública;
- documento lógico y versión;
- representación exacta;
- huella SHA-256 de los bytes;
- evidencia de firma validada;
- recibo de custodia.

Una referencia ausente, repetida o perteneciente a otro documento invalida la
versión. La custodia de los bytes sigue correspondiendo al conector documental;
la convocatoria solo conserva referencias opacas y verificables.

## Transiciones de gobierno

El ciclo admitido es:

```text
borrador -> publicada -> retirada
                     +-> sustituida por la siguiente versión
```

Reglas principales:

- solo los borradores se editan;
- cada actualización usa revisión y huella esperadas;
- una actualización sin cambio semántico se rechaza;
- creador y último modificador no pueden publicar;
- quien publicó no puede retirar;
- aprobación y verificación de dependencias se vinculan a la referencia y
  huella exactas;
- una retirada no se reactiva;
- una corrección crea otra versión;
- publicar una sucesora y sustituir la anterior debe ser una sola transacción;
- solo puede existir una versión activa por procedimiento e identificador
  público.

## Huellas diferentes

El modelo mantiene tres huellas con finalidades distintas:

| Huella | Incluye | Uso |
|---|---|---|
| `HuellaContenidoSHA256` | identidad, secuencia, predecesora, contenido, expediente y dependencias exactas | vincular solicitudes, aprobaciones y reproducción histórica |
| `HuellaSHA256` | instantánea completa, incluido gobierno | CAS, auditoría y evidencia de estado |
| `Convocatoria.HuellaPublicaSHA256` | proyección pública con fase y fechas | caché, integridad y publicación |

La huella de contenido permanece idéntica al publicar, sustituir o retirar. Un
cambio de regla, calendario, flujo, catálogo o documento produce otra huella.

## Persistencia prevista

El adaptador inicial será PostgreSQL y el puerto seguirá siendo sustituible por
Oracle u otro conector. El esquema `vec_bolsa` debe ofrecer:

- instantáneas de solo adición;
- puntero actual con CAS;
- índice de publicación activa;
- idempotencia semántica estable;
- consumo único de la decisión autorizativa;
- revalidación de sesión, perfil, rol y políticas dentro de la transacción;
- auditoría encadenada;
- bandeja transaccional de eventos;
- publicación nueva y sustitución anterior en un único `COMMIT`;
- pools y roles técnicos distintos para gestión interna y lectura pública.

No se reutilizan directamente las tablas de baremación porque sus restricciones,
roles y funciones están cerrados nominalmente a ese agregado. Sí se reutilizan
sus patrones probados de transacción serializable, privilegio mínimo, seguridad
por filas, funciones nominales y recuperación tras respuesta perdida.

## Pruebas del dominio ya presentes

Las pruebas cubren, entre otros casos:

- borrador sin marcas de publicación ficticias;
- separación entre gobierno y fase;
- publicación con doble control;
- estabilidad de la huella semántica;
- proyección de distintas fases sin mutar la versión;
- documento firmado y custodiado uno a uno;
- conflicto de revisión;
- rechazo de actualización sin cambios;
- comprobación de dependencias caducada;
- retirada con separación de funciones;
- nueva versión y sustitución exacta;
- clonación profunda.

La batería completa `go test ./...` pasó tras el corte. Esto acredita el
dominio, no la persistencia real: las pruebas PostgreSQL deberán ejecutarse sin
saltos sobre una instancia limpia y con sus roles efectivos.

## Siguiente secuencia de implantación

1. Cerrar puertos y servicio de aplicación con órdenes opacas, autorización
   previa, idempotencia y CAS.
2. Implementar memoria únicamente como doble contractual de pruebas.
3. Crear el esquema durable `vec_bolsa`, migraciones, roles y pruebas reales.
4. Componer gestión interna sin exponerla en la superficie exterior.
5. Derivar la consulta pública desde versión e instancia de flujo.
6. Crear la solicitud ciudadana vinculada a referencia y huella exactas.
