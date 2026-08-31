# VEC-DOC-CONS-01 — Política documental gobernada V1

Fecha: 31 de agosto de 2026.

Base exacta: `b2390a5179ec5cb807707766d90a8188fc3f4c66`.

Corrección arquitectónica: parte de
`57676b316c10c7ce4531d0246ee3dd38794e2cde` y atiende exclusivamente el P1
que situaba la coordinación ejecutable dentro de `ports`.

Estado técnico local: implementado y con puertas focales verdes; pendiente de
revisión independiente. No habilita datos reales, conservación operativa,
borrado ni expurgo.

## Decisión

La autoridad de conservación y bloqueo pertenece a la capacidad documental
común de VEC. Contratación temporal no define plazos, bases jurídicas, series,
tipos documentales ni bloqueos. Su siguiente corte solo podrá consumir una
política exacta resuelta por este puerto.

`internal/vec/ports` incorpora el contrato neutral para solicitar y representar
la única política aprobada que coincide con todos estos vínculos. La capa
`internal/vec/application` coordina su resolución:

- procedimiento;
- serie documental;
- tipo documental;
- expediente;
- referencia, versión y huella SHA-256 de la política;
- referencia de la base jurídica;
- vigencia exacta semiabierta `[desde, hasta)`.

Todos los identificadores son referencias opacas con el formato técnico
`ref:` seguido de 64 caracteres hexadecimales minúsculos no nulos. No hay
campos de texto libre, identidad, contacto, credencial, ruta, producto de
almacenamiento ni contenido documental.

## Contrato

`SolicitudPoliticaConservacionDocumental` es un valor cerrado construido por
`NuevaSolicitudPoliticaConservacionDocumental`. Conserva internamente la
huella en 32 bytes y entrega siempre una copia. El valor cero, una referencia
legible o repetida, una huella nula, una versión cero o una vigencia no
canónica se rechazan.

`ResolutorPoliticaConservacionDocumental` devuelve todas las coincidencias de
la publicación exacta. No expone una operación «última política» y no autoriza
al coordinador a elegir la primera. El contrato, los valores y sus invariantes
permanecen en `ports`; `application.ResolverPoliticaConservacionDocumental`
exige cardinalidad exactamente uno y vuelve a comprobar cada ligadura mediante
el constructor neutral del resultado.

El reloj neutral ya existente en `ports` se inyecta en el coordinador de
`application` y se consulta después de la resolución. La política solo es
aplicable cuando está aprobada y el instante UTC de microsegundo se encuentra
dentro de su vigencia semiabierta. Un contexto cancelado antes o durante la
frontera lenta deniega sin producir resultado.

`PoliticaConservacionDocumental` contiene exclusivamente:

- la solicitud exacta inmovilizada;
- el fin de conservación ordinaria resuelto por la autoridad;
- la protección efectiva `conservacion` o `bloqueo`;
- una referencia opaca de bloqueo cuando este está activo;
- el estado `aprobada` o `retirada` y el instante exacto de retirada cuando
  corresponda.

El bloqueo es la protección efectiva prevalente aunque exista una fecha de
fin de conservación ordinaria. Esa fecha nunca significa permiso para borrar,
eliminar o expurgar. El contrato no define ninguna operación, capacidad,
booleano ni estado que autorice esos efectos.

## Denegación predeterminada

La función de resolución de `application` devuelve el valor cero y el único
error público `ErrPoliticaConservacionDocumentalNoResuelta` ante cualquiera de
estas condiciones:

- solicitud o resultado inválidos;
- política ausente o más de una coincidencia;
- ligadura distinta en cualquiera de los diez ejes exactos;
- política retirada, aún no vigente o vencida;
- contexto nulo o cancelado;
- resolutor o reloj nulos, incluidos nulos tipados;
- resultado acompañado de error;
- error interno de la dependencia.

La respuesta no distingue catálogo, proveedor, causa interna ni existencia de
un expediente. Un resultado parcial acompañado de error nunca se aprovecha.

## Copias e inmutabilidad

Solicitud, política y resultado tienen estado privado y solo nacen mediante
constructores. Los `time.Time` son valores UTC canónicos y la única colección
mutable, la huella SHA-256, se copia al entrar y al salir. Sus representaciones
de texto están redactadas y no incluyen referencias.

## Autoridades y límites

La decisión aplica las siguientes autoridades leídas para este corte:

- `capacidad_documental.md`: Documentos es la autoridad transversal, la
  coordinación reside en `application` y los módulos consumidores no generan
  reglas propias;
- `almacen_documental_seguro.md`: retención, bloqueo y archivo se coordinan
  por política de serie, sin que una referencia o un permiso anterior conceda
  eliminación;
- `firma_csv_qr_y_cotejo.md`: conservación y acceso dependen de política
  aprobada y no del CSV, QR o conocimiento de una referencia;
- `cumplimiento_y_seguridad.md` y la matriz normativa de Contratación
  temporal: Archivo y DPD deben aprobar calendario, base, bloqueo y acceso;
- O8-01 y el tablero O8: este contrato prepara O8-03, no O8-04.

CT-CUM-06 y las aprobaciones de Archivo, DPD, Secretaría y demás responsables
siguen abiertas. No se incorpora ninguna política real, plazo normativo real,
base jurídica real ni dato personal. Las pruebas usan únicamente referencias,
fechas y huellas sintéticas.

## Alcance negativo

Este corte no añade adaptadores, SQL, HTTP, composición, almacenamiento,
firma, cotejo, auditoría durable, outbox, API, interfaz, datos reales, borrado
ni expurgo. Tampoco cambia porcentajes, estados transversales o autorizaciones
de producción.

## Evidencia focal

La familia `TestPoliticaConservacionDocumental`, separada entre las invariantes
de `ports` y la coordinación de `application`, cubre:

- resolución válida y prevalencia del bloqueo;
- valores cero y entradas que intentan transportar identidad o credenciales;
- variación individual de todas las ligaduras;
- ausencia, ambigüedad, retirada, vigencia futura y vencimiento;
- cancelación previa y durante la resolución;
- dependencias nulas y nulas tipadas;
- resultado simultáneo con error y opacidad del diagnóstico;
- copias defensivas de la huella.

Puertas del corte:

```text
gofmt
GOPROXY=off go test ./internal/vec/ports ./internal/vec/application -run '^TestPoliticaConservacionDocumental' -count=1
GOPROXY=off go test -race ./internal/vec/ports ./internal/vec/application -run '^TestPoliticaConservacionDocumental' -count=1
GOPROXY=off go vet ./internal/vec/ports ./internal/vec/application
git diff --check
```

Resultado local sobre el árbol correctivo:

- prueba focal normal: verde;
- prueba focal con detector de carreras: verde;
- `go vet` focal: verde;
- formato y `git diff --check`: verdes;
- tamaños: `ports` 332 líneas de producción y 223 de pruebas; `application`
  55 líneas de producción y 393 de pruebas; decisión 167 líneas. Todos
  permanecen por debajo del máximo de 800.

Esta evidencia es del productor y no sustituye la reproducción ni el `GO` de
un revisor independiente.

## Siguiente corte

`CT-LITE-O8-03B` consumirá esta política exacta desde Contratación temporal.
No implementará ni autorizará expurgo.
