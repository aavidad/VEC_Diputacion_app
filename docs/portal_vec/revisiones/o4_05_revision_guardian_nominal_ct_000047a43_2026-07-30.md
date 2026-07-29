# Revisión del guardián nominal CT-000047A4.3

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente** para el emisor de material de consultas RRHH.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `841f90a` |
| Commit candidato | `ba4902b` |
| Commit integrado | `fa304c4` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Garantías acreditadas

`EmisorMaterialConsultaRRHH` solo se construye de forma atómica con:

- el resolutor nominal M0;
- un generador estrecho de correlaciones;
- un reloj;
- dos emisores A2.2 de identidad física distinta, uno para cuadro y otro para
  detalle.

Los métodos públicos son nominales y separados. Cada recorrido:

1. comprueba contexto, cancelación y reloj UTC canónico;
2. resuelve una sola vez el motivo correspondiente;
3. genera una sola correlación opaca;
4. reutiliza A4.2 para derivar D1 o D2 y retener el resultado exacto;
5. invoca exclusivamente el emisor A2.2 de su operación;
6. vuelve a leer el reloj, rechaza retrocesos y revalida la preparación;
7. construye y coteja el material VEC-AD-3 final.

No existe getter del `ResultadoContextoActorRegistradoV2`, segunda resolución
de identidad ni selector libre de acción, finalidad, recurso, motivo,
organización o audiencia.

La cancelación y el vencimiento se conservan; el resto de errores se sanea a
`ErrConsultaRRHHNoDisponible`. Cualquier error, incluido uno posterior a una
decisión y confirmación durables, devuelve material cero y nunca fabrica un
éxito parcial.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron focales veinte veces, detector de
carreras tres veces, el paquete completo de puertos de Contratación temporal,
`go vet`, formato, revisión del diff, tamaños y Gitleaks. Todo terminó en
verde.

## Límite

A4.3 cierra el guardián de emisión, no la migración de los casos de uso
existentes. A5 debe sustituir los dos puertos copiables
`AutoridadContextoConsultaRRHH` y `AutorizadorConsultaRRHH` antes de la
composición raíz. M1/M2, organización corporativa, KMS/COSE y TLS/mTLS real
siguen siendo dependencias separadas.
