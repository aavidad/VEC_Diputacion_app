# Revisión de retirada de vías crudas CT-000047A5.2

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `402452e` |
| Commit candidato | `fe89040` |
| Commit integrado | `12aaaf6` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Alcance verificado

La superficie productiva ya no exporta:

- `AutorizadorConsultaRRHH`;
- `NuevoMaterialAutorizacionConsultaRRHH`.

El constructor de material es privado y A4.3 conserva su única llamada. No se
añade otra interfaz, guardián o ayudante productivo para las pruebas.

Los fixtures de aplicación y puertos atraviesan el emisor A4.3 real. Se
retiraron dobles, campos y asignaciones muertos; los ficheros quedaron en 369
y 460 líneas, cerrando el P2 de A5.1.

El ámbito de organización es el único ejercitado porque es el único autorizado
por la decisión vigente. Centro y unidad permanecen modelados, pero no se
habilitan sin una autoridad corporativa gobernada.

## Cobertura conservada

La eliminación de andamiaje no retiró las pruebas negativas:

- D1/D2 cubren tipo, referencia, dominio, huella, clase y cruces;
- A4.2 cubre contexto, solicitud, recurso, motivo y correlación;
- A4.3 cubre audiencia, nulos, cronología, cancelación y error posterior a
  evidencia durable;
- las suites de material, capacidad, codecs, cánones, recibos y cursores
  permanecen verdes.

Los vectores PostgreSQL conservan sus constantes byte a byte. El test calcula
`SHA-256(canon)` por una segunda ruta y lo coteja con la constante; no copia la
salida del código probado.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron focales veinte veces, detector de
carreras tres veces, los paquetes completos de aplicación y puertos, `go vet`,
formato, revisión del diff, tamaños, guardas de símbolos y Gitleaks. Todo
terminó en verde.
