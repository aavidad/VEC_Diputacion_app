# O4-05: revisión de los cánones de resultado RRHH

Fecha: 28 de julio de 2026

Ámbito: parte Go del contrato probatorio CT `000041`

Estado: `GO` técnico independiente; CT `000041` y producción siguen abiertas

## Alcance verificado

Los commits `c9f1849`, `297e2bb` y `69aac05` incorporan:

- canon del contenido de una página del cuadro RRHH;
- canon del contenido minimizado del detalle de expediente;
- canon común de evidencia de resultado compatible con PostgreSQL;
- huella SHA-256 de los 32 bytes crudos del cursor, nunca de su representación
  Base64URL;
- validación previa al recibo, orden estable, rechazo de duplicados y copias
  defensivas;
- límite estricto de 256 KiB sin anexado parcial;
- tipos nominales opacos que impiden serializar o registrar accidentalmente el
  material interno.

El detalle se reconstruye desde
`EntradaDetalleExpedienteRRHHMinimizada`; no se ha creado un DTO paralelo ni
una vía alternativa de autorización.

## Comparación real Go y PostgreSQL

La revisión independiente ejecutó PostgreSQL 18.4 mediante la imagen fijada:

```text
postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296
```

El resultado de cuadro vacío coincidió byte a byte en ambos extremos y produjo:

```text
cb8ad45d7c31faa5100a249a840e66671b0de2319d23fc1c8878e56da7076ee0
```

El resultado del detalle mínimo también coincidió byte a byte y produjo:

```text
0b7d78f6d34cd87f3da98fc32a0830ba953c83f36c2a7423fba9810baff78e31
```

Para el vector de cursor formado por 32 octetos `FF`, la huella del material
decodificado fue:

```text
af9613760f72635fbdb44a5a0a63c39f12af30f950a6ee5c971be188e89c4051
```

Las pruebas confirman que el token Base64URL no aparece en los bytes
canónicos, en `fmt`, en `slog` ni en los serializadores bloqueados.

## Puertas reproducidas

Quedaron verdes y sin caché:

- `go test -count=1 ./internal/modules/contrataciontemporal/...`;
- `go test -count=1 -race` para `ports` y `domain`;
- `go vet ./internal/modules/contrataciontemporal/...`;
- `git diff --check`;
- Gitleaks sobre los tres commits: 46,82 KiB, cero hallazgos.

Los ocho ficheros del corte permanecen por debajo de 800 líneas.

## Dictamen y límites

La parte Go de cánones de contenido y resultado obtiene `GO`. El dictamen no
cierra CT `000041`: faltan el Recibo V2 y el contrato PostgreSQL que recalculará
y ligará esos cánones en la misma transacción que el acceso.

La primera versión de la migración de vocabulario de estados obtuvo
`NO-GO` adversarial por manifiestos estructurales insuficientes y no se ha
integrado ni publicado. Está en corrección. Por tanto se mantienen:

- Contratación temporal: `20/46`, 43 %;
- O4-05: `3/5`;
- Bolsa: `1/14`, 7 %;
- producción: `NO-GO`.
