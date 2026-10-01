# Provisión: recorrido de ensayo y persistencia pendiente

Fecha: 1 de octubre de 2026. Base inventariada:
`origin/main@960795f3090212257d8df92791bf740e3e663c8a`.

## Lo existente y el corte preparado

La base contiene el motor de Provisión, las seis familias de valoración, el
caso de uso `application.Simular`, el JSON estricto y el consumidor CLI
`vec-baremador --modo concursos`. El motor reutiliza la aritmética exacta
común. Un resultado completo acredita que pudo calcular, no admisión ni
aprobación administrativa.

La candidata web local #250, cabeza
`ca68bd1dffc159a115acdcb06f87f7ef6b85fd40`, permite simular con un ejemplo
sintético elegido por referencia. El servidor fija los hechos. Los assets
situados bajo Bolsa no cambian la propiedad del motor: el procedimiento de
concurso pertenece a Provisión. Este inventario no acredita montaje en el
portal institucional ni publicación de esa candidata.

En esa base faltan proceso de concurso, oferta, solicitud multipuesto,
preferencias, adjudicación global y ciclo de revisión y reclamaciones. Las
piezas que se preparan en paralelo deberán revisarse e integrarse por sus
propios hashes; este documento no las da por integradas.

Este corte añade un ensayo del ciclo sobre la valoración de un solo puesto:

1. Calcula la valoración provisional de ensayo, versión 1, con las reglas y
   la instantánea recibidas.
2. Enlaza cada reclamación con esa versión y su `huella_revision` exacta, una
   causa del catálogo versionado recibido y una referencia opaca de evidencia.
3. Acepta decisiones motivadas de mantener o rectificar. Cada decisión exige
   la versión actual esperada y añade una nueva revisión ligada a la anterior.
4. Para rectificar exige otra instantánea del mismo puesto y ejecuta de nuevo
   `application.Simular`. Mantiene el cálculo, entrada y huella originales.
5. Prepara una resolución con estado `borrador`, versión y huella final de
   ensayo. Señala reclamaciones, revisión inicial o fuentes pendientes.

Las huellas de la nueva revisión incluyen las huellas de la reclamación,
decisión, catálogo y versión anterior. Una motivación diferente cambia la
huella. Mantener una valoración añade una revisión motivada aunque sus puntos
coincidan. Las referencias documentales declaradas no acreditan la existencia
ni custodia de esos documentos.

## Cómo reproducir el ejercicio sintético

```bash
go run ./cmd/vec-ensayar-provision --ejemplo
go run ./cmd/vec-ensayar-provision --ejemplo --emitir-entrada > /tmp/provision-ciclo.sintetico.json
go run ./cmd/vec-ensayar-provision < /tmp/provision-ciclo.sintetico.json
```

La salida conserva dos valoraciones: el curso sintético pierde su condición
de acreditado en la segunda instantánea y el motor recalcula la diferencia.
La resolución sigue siendo un borrador. No se registran datos ni se modifica
Personal o RUM. Cada ejecución vuelve a calcular el ejercicio; esto no es
recuperación de una operación institucional ni idempotencia durable.

El CLI recibe un único JSON por entrada estándar y reutiliza
`simulacion.Decodificar`: límite de 2 MiB, profundidad, claves exactas y
únicas, campos obligatorios y tipos numéricos estrictos. Se limitan además
reclamaciones y revisiones a 32 y méritos acumulados a 10.000. Las causas del
ejemplo son sintéticas y no se aplican automáticamente a ningún concurso.

Se ejecutaron pruebas focales normales y con detector de carreras, `go vet`,
comprobación con `gopls` y el CLI real. Las pruebas cubren recálculo, historia,
copias independientes, causas, versiones, decisiones duplicadas, pendientes y
JSON hostil. Semgrep (`p/golang`, métricas desactivadas) y gosec no encontraron
hallazgos. Esta evidencia no incluye PostgreSQL, navegador ni reinicio, porque
el corte no monta esos adaptadores. La revisión independiente del hash final
corresponde a Dirección antes de integrar.

Las reclamaciones de este ensayo se refieren al provisional inicial. Varias
decisiones se encadenan con sus versiones esperadas; una reclamación de una
publicación posterior necesitará otro comando del proceso durable. No se
simulan plazo, presentación, audiencia, admisión, adjudicación ni cierre legal.

## Propietarios y dependencias

Provisión conserva proceso, bases, solicitudes, preferencias, requisitos,
valoraciones, reclamaciones y adjudicación. Personal conserva relación de
servicio, grado y ocupaciones; RPT, puestos y publicaciones; RUM, méritos y
evidencias. Bolsa conserva su procedimiento. Ninguna puntuación pasa a ser un
hecho universal de Personal o RUM.

`ProcesoProvision.Referencia/Version` y las versiones de reglas son dimensiones
distintas en el contrato del corte de preparación paralelo. El repositorio
futuro deberá enlazarlas de forma exacta y comprobar la solicitud y oferta,
además de las huellas del cálculo. El ensayo actual no acredita esos vínculos
institucionales.

El [contrato PostgreSQL pendiente](../../deploy/postgresql/provision/README.md)
recoge el orden causal: H6 y núcleo, cola D, fuentes y versiones gobernadas,
repositorio autorizado, revisión SQL independiente y ensayo en el clon.
`RepositorioCicloProvision` exige versión esperada, idempotencia, recibo,
historia, auditoría y outbox en una transacción. Los puertos de fuentes,
firma y publicación siguen sin adaptador. No hay SQL ejecutable ni ACL nueva.

## Fuentes y límites

El [estudio de integración del Baremador](integracion_baremador_concursos_provision.md)
define los propietarios y el recorrido completo. Las referencias primarias
consultadas ayudan a contrastar el diseño; sus reglas no se incorporan de
oficio a la Diputación ni al motor:

- La [convocatoria de Granada, expediente 2025/PPT_01/000087](https://www.dipgra.es/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/recursos-humanos/CONVOCATORIA-PARA-LA-PROVISION-DE-PUESTOS-DE-TRABAJO-SINGULARIZADOS-POR-EL-PROCEDIMIENTO-DE-CONCURSO/)
  conserva anuncios de admisión provisional, corrección, rectificación y
  admisión definitiva. Apoya conservar versiones y antecedentes; no determina
  por sí sola el baremo de este ensayo.
- La [Orden de la Junta de Andalucía de 23 de mayo de 2025](https://www.juntadeandalucia.es/boja/2025/100/43)
  regula su concurso abierto y permanente. Su ámbito exige contrastar las
  preferencias y el circuito propios antes de reutilizar reglas.
- La [Resolución estatal de 21 de mayo de 2026](https://boe.es/buscar/doc.php?id=BOE-A-2026-11413)
  regula un concurso de habilitación nacional. Se consulta como otra familia
  de provisión, sin equiparar su procedimiento al concurso singularizado.

La resolución de ensayo mantiene `firmada=false`, `publicada=false` y
`efecto_oficial=false`, incluso con todas las reclamaciones decididas. Bases
y causas quedan sin contraste institucional; firma, publicación y efectos
requieren sus autoridades y recibos reales. Sólo se usan ejemplos públicos
sintéticos; no se copian personas ni archivos del prototipo local.

## Alcance de los ejemplos locales

La preparación multipuesto, la adjudicación global y la revisión de una
valoración usan ejemplos sintéticos independientes. Comparten motor y
contratos, pero no forman un expediente institucional único ni acreditan que
una solicitud presentada haya pasado por todo el procedimiento. Los datos de
Personas/RPT/RUM y las decisiones de RRHH se incorporarán por sus puertos;
conectar esas versiones y conservarlas pertenece al corte durable pendiente.

Las preguntas sobre bases y puestos ofertados, método global y desempates,
fuentes y fecha de corte, revisión y publicación se entregan a dirección para
su numeración en `dudas.md` durante el turno compartido. Las opciones del
ejercicio no constituyen una aprobación de RRHH.
