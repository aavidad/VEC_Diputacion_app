# CT-LITE-O8-03B — Consumo de política documental gobernada

Fecha: 31 de agosto de 2026.

Base exacta: `3f72adc33d8a44203ae5785e14b033668975b266`.

Estado: candidato del productor pendiente de revisión independiente. No cambia
el estado de O8-03, CT-CUM-06 ni las autorizaciones de producción.

## Capability e invariante

Contratación temporal puede obtener el resultado exacto de una
`SolicitudPoliticaConservacionDocumental` mediante el resolutor y el reloj
neutrales de `internal/vec/ports`. La única coordinación de cardinalidad,
vigencia, aprobación y coincidencia permanece en
`internal/vec/application.ResolverPoliticaConservacionDocumental`.

El consumidor de Contratación temporal no selecciona políticas, no calcula
plazos, no interpreta `ConservacionHasta` como permiso y no degrada un bloqueo
a conservación ordinaria. Solicitud, contexto o dependencia inválidos,
ausencia, ambigüedad, retirada, falta de vigencia, cancelación o error interno
devuelven el valor cero y el error público opaco
`ErrPoliticaConservacionDocumentalNoResuelta`.

## Frontera y dependencia

El corte añade una fachada mínima en la capa de aplicación de Contratación
temporal. La fachada recibe directamente los valores cerrados de VEC y delega
en su coordinador común. No existe un puerto local equivalente, un segundo
coordinador, un catálogo de CT ni una traducción que pueda perder ligaduras.

La dirección de dependencia es:

```text
contrataciontemporal/application
        |
        +--> vec/application (coordinación única)
                  |
                  +--> vec/ports (contratos neutrales)
```

## Evidencia focal

Las pruebas sintéticas cubren:

- solicitud exacta y una única resolución aprobada;
- preservación del bloqueo prevalente y de la fecha ordinaria sin convertirla
  en autorización;
- ausencia, ambigüedad, retirada y borde exclusivo de vigencia;
- resultado parcial acompañado de error privado sin filtración;
- contexto nulo, cancelación previa y durante la frontera;
- solicitud cero, receptor nulo y dependencias nulas o nulas tipadas.

Puertas del productor:

```text
gofmt
GOPROXY=off go test ./internal/modules/contrataciontemporal/application -run '^TestConsumoPoliticaConservacionDocumental' -count=1
GOPROXY=off go test -race ./internal/modules/contrataciontemporal/application -run '^TestConsumoPoliticaConservacionDocumental' -count=1
GOPROXY=off go vet ./internal/modules/contrataciontemporal/application
git diff --check
```

El preflight comprobó la base exacta, el árbol limpio, la ausencia previa de
los tres ficheros del write-set y la regresión VEC-DOC-CONS-01. Las puertas
focales normal y `-race`, `go vet`, `gofmt` y `git diff --check` terminaron
verdes en el candidato. También terminó verde la suite completa del paquete
`internal/modules/contrataciontemporal/application`, tanto normal como con el
detector de carreras.

## Alcance negativo y aprobaciones

No se añaden eliminación, borrado, expurgo, política real, plazos normativos
reales, base jurídica real, datos personales, SQL, adaptadores, composición,
HTTP, API, almacenamiento ni activación productiva. Las pruebas solo usan
referencias, huellas e instantes sintéticos.

CT-CUM-06 continúa abierta. Archivo y DPD deben aprobar la serie, el calendario,
la base jurídica, los bloqueos, el acceso, la transferencia y la eliminación;
este corte no sustituye esas decisiones. O8-04 permanece bloqueada y será otro
contrato con doble autorización y evidencia de eliminación, si llega a ser
aprobado.
