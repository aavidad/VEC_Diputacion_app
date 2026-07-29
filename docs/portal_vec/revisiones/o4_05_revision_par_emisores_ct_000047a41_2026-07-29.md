# Revisión del par de emisores CT-000047A4.1

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para la separación nominal privada de los
emisores VEC de cuadro y detalle.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `a677b14` |
| Commit candidato | `873426e` |
| Commit integrado | `05d6767` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Garantías acreditadas

La interfaz privada reproduce exactamente la firma estructural de A2.2. Dos
envoltorios nominales distintos conservan los emisores de cuadro y detalle en
campos separados.

El constructor privado rechaza:

- dependencias nulas y nulas tipadas;
- implementaciones por valor sin identidad física estable;
- la misma instancia física en ambas posiciones.

Acepta dos punteros distintos aunque tengan igual contenido y no los
intercambia. La comprobación por reflexión valida el tipo antes de consultar
el puntero y no abre una ruta de pánico.

No se publica API, audiencia, motivo, recurso, llamada de emisión, PDP o
composición.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron focales veinte veces, paquete
`ports`, detector de carreras, `go vet`, revisión del diff, tamaños y Gitleaks.
Todo terminó en verde; no se detectaron secretos.

## Límite

La identidad física acredita que hay dos instancias, no su configuración
interna. La futura composición debe demostrar que una está preligada a la
audiencia de cuadro y la otra a detalle.
