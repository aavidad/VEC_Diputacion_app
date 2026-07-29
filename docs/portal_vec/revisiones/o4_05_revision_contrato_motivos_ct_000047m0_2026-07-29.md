# Revisión del contrato de motivos CT-000047M0

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para el puerto nominal de resolución de motivos
de consulta RRHH.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `b372b5a` |
| Commit candidato | `3df7b06` |
| Commit integrado | `c1bb5ec` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Contrato acreditado

`ResolutorMotivoConsultaRRHH` expone exactamente dos métodos nominales, uno
para cuadro y otro para detalle. Cada método recibe únicamente
`context.Context` e instante y devuelve una
`ReferenciaEntradaCatalogo` o el centinela público opaco.

No admite cadenas, selectores, organización, acción, finalidad, HTTP, SQL o
una implementación concreta. El contrato exige UTC canónico, cancelación,
referencia positiva validada y error cerrado. La composición y el futuro
orquestador deberán rechazar implementaciones nulas tipadas, porque una
interfaz Go no puede imponerlo por sí sola.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron focales veinte veces, carrera,
todos los paquetes de Contratación temporal, `go vet`, formato, revisión del
diff, tamaños y Gitleaks. Todo terminó en verde y no se detectaron secretos.

## Límites

M0 no selecciona ni persiste motivos. M1 y M2 deberán implementar dos vínculos
gobernados, uno para cuadro y otro para detalle, una vez que RRHH, DPD y
Jurídico aprueben finalidad, base jurídica y publicaciones. No se codificará
una referencia fija como sustituto.
