# Instruccion de direccion — 18 de julio de 2026

Cambio de prioridades. Este documento manda sobre cualquier instruccion
anterior en lo que la contradiga. Lee primero, en este orden:

1. [Backlog Txx](autoprogramacion_orquesta_pendientes_2026-07-16.md) — el
   bloque «Orden de ataque vigente» y las entradas **T20** y **T21**.
2. [Registro de decisiones](portal_vec/registro_decisiones.md) — **DEC-092**
   y **DEC-093**.

Esos dos documentos son la fuente de verdad. Lo que sigue es el resumen
ejecutivo, no un sustituto de leerlos.

## Que hacer ahora: T21 y T20, juntas y en cabeza de cola

**T21 primero, porque habilita lo demas**: perfil de ejecucion `desarrollo`
con credenciales propias — CA local, certificados mTLS, KMS respaldado por
fichero, identidad de alta garantia simulada, TSA determinista y TLS
autofirmado. Todo detras de las interfaces que **ya existen**, extendiendo el
`AuthMode` de `config`. No crees un mecanismo paralelo.

**T20 encima**: adaptador Go PostgreSQL del diario y del agregado cifrado de
borradores de convocatorias, restaurando la identidad primaria y todos sus
alias HMAC multigeneracion, y revalidando la atestacion KMS dentro de la
transaccion de confirmacion. Corrige antes el defecto alto de idempotencia
multigeneracion que tu propia revision cruzada dejo en NO-GO (ventanas de
rotacion solapadas tipo `[g3,g2]` y `[g2,g1]`).

**Criterio de terminado**: la bandeja y el editor de borradores del Portal
del Empleado **funcionan de punta a punta** bajo el perfil `desarrollo`, sin
fallar cerrada. Alta y actualizacion. **No** publicacion ni retirada: esas
tienen la barrera de DEC-091, que es de diseno juridico y no la levanta
ninguna credencial.

## Que ha cambiado y por que

Ya no esperas a Sistemas. La identidad, el KMS y el TLS autoritativos
llegaran como **cambio de configuracion, no de diseno**, porque todos entran
inyectados tras interfaz. **S-03 sigue bloqueando produccion; ya no bloquea
desarrollo.** Se aparca el despliegue real, no el trabajo.

El motivo del cambio: se habia acumulado contrato de nucleo probado en
memoria porque el camino a lo operativo estaba cerrado. Eso produce paquetes
verdes sin flujo utilizable.

## Guardarrailes: son condicion, no recomendacion

El riesgo de usar credenciales de pega no es tecnico, es de **trasvase**: que
ese material, o los actos firmados con el, acaben tratados como
autoritativos. Se contiene asi:

1. **Ni una clave ni un certificado en Git.** El repositorio es **publico**.
   Genera el material con script, en local. Los patrones ya estan en
   `.gitignore`. Esto incluye los de pega: un certificado en el historico
   publico es un hallazgo de seguridad aunque no valga nada.
2. El perfil `desarrollo` **no** es el valor por defecto ni es seleccionable
   desde el perfil productivo. Replica el patron anti-fuga que ya usas para
   los datos sinteticos de demostracion.
3. Todo acto producido bajo `desarrollo` va **marcado de forma estructural e
   imborrable como no autoritativo, en el dato persistido** y no solo en la
   configuracion. Los datos de desarrollo **no se migran**: se descartan al
   cambiar de perfil. Un sello de la TSA de desarrollo no debe poder
   confundirse jamas con uno cualificado.
4. **Arranque ruidoso**: el log declara en cada arranque que se corre con
   credenciales no autoritativas, y cuales.
5. **Prueba de conmutacion**: el perfil productivo **rechaza arrancar** si
   algun proveedor de desarrollo esta compuesto. El fallo cerrado se
   conserva; lo que cambia es que en `desarrollo` hay un proveedor que
   responde, no que desaparezca la exigencia.

## Que NO debes hacer

- **Ni una tanda mas de T02.** Aparcado hasta la v2 (DEC-092). Pero sus
  reglas de contencion siguen vigentes: **ningun fichero nuevo en `*/ports`**,
  DEC-051 sin excepciones, y `internal/vec/canonico` se conserva y se usa.
- **No sustituyas un camino bloqueado por mas contrato de nucleo probado en
  memoria.** Ante un bloqueo: informa y pasa a la siguiente tarea
  desbloqueada de la cola. No profundices en la bloqueada.
- **Si una tarea necesaria no esta en la cola, senalalo a direccion** en vez
  de asumir que no es prioritaria. T20 estuvo dias descrito como «trabajo
  propuesto» dentro de un analisis de brecha sin entrada numerada, y por eso
  nunca se abordo. Esa omision fue de direccion, pero se corrige antes si la
  senalas.
- **No bajes el liston** en lo que si se programa: mismo rigor de pruebas,
  `go vet`, carrera donde toque y evidencia medida en cada entrega. Lo que se
  aparca es un refactor estructural, no la disciplina.

## Pendiente inmediato en tu arbol de trabajo

`web/static/portal-empleado/portal.js` esta en **818 lineas** sin commitear
(794 en HEAD). Supera el tope duro de 800 y la puerta de tamano te va a
rechazar el commit. Trocealo antes de entregar.
