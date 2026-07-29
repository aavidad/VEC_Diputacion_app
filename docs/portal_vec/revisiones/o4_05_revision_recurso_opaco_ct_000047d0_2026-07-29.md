# Revisión del recurso opaco CT-000047D0

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para el envoltorio opaco común de los recursos de
consulta RRHH.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `83ceea4` |
| Commit candidato | `bb3bc2e` |
| Commit integrado | `6a18b43` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

## Superficie acreditada

`RecursosConsultaRRHH` contiene exclusivamente:

- el bloqueador común de serialización y registro;
- un único `RecursoAutorizable` privado, sin etiqueta.

No existe getter, exportador, constructor genérico ni acceso a sus mapas. El
valor cero falla cerrado. El validador común solo comprueba la estructura del
recurso y el módulo exacto; las reglas de cuadro y detalle pertenecen a las
fábricas nominales D1 y D2.

JSON, XML, Text, Binary, Gob, CBOR y YAML quedan bloqueados para escritura y
reconstrucción. `fmt` y `slog` redactan el valor sin filtrar referencia,
ámbitos o atributos.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron:

```text
pruebas focales repetidas veinte veces
go test del paquete ports
go test -race del paquete ports
go vet del paquete
gofmt
git diff --check
verificación de tamaños
Gitleaks sobre el commit
```

Todas las puertas terminaron en verde. Los dos archivos tienen 23 y 240
líneas, y el análisis de secretos del commit no encontró filtraciones.

## Límites

D0 no construye recursos de cuadro o detalle, no fija una decisión PDP, no
emite material VEC-AD-3 y no toca aplicación, HTTP, PostgreSQL, composición o
web. D1 y D2 deben seguir impidiendo que organización, mapas, acción,
finalidad, audiencia, dominio o huella procedan del cliente.
