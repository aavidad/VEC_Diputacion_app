# Revisión del adaptador PostgreSQL de consultas RRHH CT-000046

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para el alcance CT-000046.

| Elemento | Valor |
| --- | --- |
| Base estable | `caecc48` |
| Candidato integrado | `6c57644` |
| PostgreSQL de las fachadas | 18.4 fijado por digest |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

Este cierre acredita el adaptador Go contra el contrato de las fachadas
CT-000045. No acredita todavía composición raíz, TLS/mTLS viva, interfaz web
conectada, E2E administrativo ni autorización de producción.

## Alcance funcional acreditado

El adaptador `SesionConsultaRRHHPostgreSQL`:

- implementa el puerto `SesionConsultaRRHH` sin cambiar dominio, aplicación,
  HTTP ni autoridad;
- invoca una sola fachada nominal por operación;
- convierte los dieciocho valores escalares del cuadro o los quince del
  detalle en los doce argumentos SQL lógicos contratados;
- escanea, por identidad y orden, las veintiuna columnas del cuadro o las
  veinte del detalle;
- abre `SERIALIZABLE READ WRITE`, valida la salida completa antes de confirmar
  y revierte ante todo fallo posterior a la apertura;
- reconstruye las proyecciones minimizadas y el `ReciboLecturaRRHHV2`;
- valida el resultado contra la orden antes del `COMMIT`;
- conserva cancelación y plazo del contexto;
- normaliza denegación, ausencia y fallos internos sin crear un oráculo;
- no reintenta una capacidad VEC de consumo único.

El analizador acepta únicamente el canon cerrado del contrato, con límite de
256 KiB, UTF-8 válido, cabeceras, bloques, cardinalidad y final exactos. El
cursor se trata como Base64 URL sin relleno y se comprueba sobre sus 32 bytes,
no sobre su representación textual.

La fábrica crea un pool exclusivo, exige un LOGIN nominal explícito y aplica
el endurecimiento TLS ya cerrado. No reutiliza el pool histórico de
recuperación ni admite como identidad el grupo técnico `NOLOGIN`.

## Evidencia reproducida

Quedaron verdes:

```text
go test ./internal/modules/contrataciontemporal/... -count=1
go test -race ./internal/modules/contrataciontemporal/adapters/postgres -count=1
go vet ./internal/modules/contrataciontemporal/...
git diff --check
gitleaks sobre caecc48..6c57644
```

Las pruebas usan centinelas distintos por posición para acreditar los valores
y tipos exactos de las entradas, punteros distintos para acreditar todos los
destinos y una traza de eventos, posterior a la apertura, para probar:

```text
query → scan → validar → commit → rollback diferido inocuo
```

La apertura se cuenta por separado. También se prueban denegación `42501`,
fallo de escaneo, validación, confirmación ambigua sin reintento, canon mal
formado y límites.

## Revisión independiente

Una primera revisión consideró insuficiente contar argumentos y destinos. El
productor añadió pruebas de identidad, tipo y orden, sin modificar el contrato
ni ampliar el alcance. Un revisor distinto examinó el candidato corregido
`6c57644` y emitió `GO` con P0=0, P1=0 y P2=0.

La comprobación integrada contra PostgreSQL se reserva para el E2E de la raíz:
CT-000045 ya acredita las fachadas reales y CT-000046 acredita el adaptador
como contrato unitario. Duplicar ahora la misma instalación no aportaría una
garantía nueva y sí crearía otro arnés que mantener.

## Límites y siguiente puerta

La siguiente tarea es componer, desde la raíz interna productiva:

1. el pool nominal de consultas RRHH;
2. `SesionConsultaRRHHPostgreSQL`;
3. los casos de uso de cuadro y detalle;
4. la frontera corporativa y el PDP reales;
5. la propiedad y cierre inverso de todos los recursos.

La composición permanecerá cerrada ante configuración incompleta y no tendrá
caída a dobles de presentación, cookies ni autoridad aportada por el cliente.
