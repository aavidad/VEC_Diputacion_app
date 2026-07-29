# Revisión de fachada de emisión VEC CT-000047A2.2

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para la fachada VEC-AD-3 de alto nivel.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `d69a744` |
| Commit candidato | `e5c223c` |
| Commit integrado | `a677b14` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Cadena acreditada

La fachada ejecuta en orden:

```text
PDP y concesión durable
→ cotejo de decisión y confirmación
→ atestación
→ verificación de confianza
→ emisión de capacidad
→ recuperación privada de la raíz nominal
→ construcción del material de consumo
```

La operación solo recibe contexto, solicitud V3 y el resultado registrado
exacto. El motivo se deriva de la solicitud. No acepta acción, recurso,
finalidad, audiencia, raíz, prueba, capacidad, clave o reloj como parámetros.

Antes de acreditar una concesión, cualquier fallo devuelve decisión y
confirmación cero y material nulo. Tras una concesión durable válida, una
cancelación o fallo criptográfico conserva decisión y confirmación, pero nunca
entrega material parcial. Los errores arbitrarios se sanean y solo preservan
la identidad de cancelación o vencimiento.

## Evidencia reproducida

La matriz prueba:

- orden de las cuatro fronteras lentas;
- compatibilidad estructural con los servicios VEC existentes;
- éxito, exportación y doble emisión;
- mismo resultado y motivo en toda la cadena;
- nulos y nulos tipados;
- ceros, mezclas, replay y resultados ajenos;
- fallo PDP con y sin concesión acreditada;
- cancelación tras cada frontera;
- atestación o raíz cruzadas;
- ausencia de filtración de errores privados.

Productor, revisor y dirección ejecutaron focales veinte veces, paquete
completo, detector de carreras, `go vet`, formato, revisión del diff, tamaños
y Gitleaks. Todo terminó en verde; producción tiene 194 líneas y la prueba 654,
ambas bajo los límites.

## Límites

La fachada no compone todavía los dos emisores nominales de cuadro y detalle.
El emisor HMAC actual conserva clave en memoria y solo queda autorizado para
pruebas. Producción exige un puerto de MAC con clave no exportable y adaptador
KMS/HSM, firmante COSE V3 y conjunto de confianza gobernado por Sistemas.
