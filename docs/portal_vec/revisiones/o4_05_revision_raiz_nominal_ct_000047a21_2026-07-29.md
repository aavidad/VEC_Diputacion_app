# Revisión de raíz nominal VEC CT-000047A2.1

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para la retención y recuperación privada de la
raíz pública VEC-AD-3 verificada.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `192515c` |
| Commit candidato | `0ac3c0d` |
| Commit integrado | `d69a744` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Garantías acreditadas

Cada entrada del servicio de confianza conserva un clon defensivo de su raíz
pública nominal. El recuperador:

- no está exportado;
- exige una prueba nominal válida del mismo catálogo;
- coteja KID, versión, huella DER-SPKI, audiencia, estado y revocación;
- coteja ventanas de raíz y todos los metadatos de configuración;
- selecciona sin ambigüedad la raíz exacta durante una rotación;
- devuelve un nuevo clon defensivo.

No existe getter general, API pública nueva o clave privada almacenada. Valores
cero y receptor nulo fallan cerrados.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron veinte repeticiones del paquete,
detector de carreras, `go vet`, formato, revisión del diff, tamaño y Gitleaks.
Las pruebas incluyen copia defensiva, once cruces de metadatos, rotación
multirraíz y revocación. Todo terminó en verde y no se detectaron secretos.

## Límites

A2.1 no emite decisiones, atestaciones, capacidades o material. Solo
proporciona a la futura fachada A2.2 la raíz nominal exacta que necesita el
constructor de material, sin abrir el catálogo a otros consumidores.
