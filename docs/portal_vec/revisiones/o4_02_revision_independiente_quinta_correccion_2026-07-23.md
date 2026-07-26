# Revisión independiente O4-02 — quinta corrección

Fecha: 23 de julio de 2026.

## Candidato

- SHA exacto: `5bb4aaf20ac9b848dc67346d161092d0f8ef21bd`.
- Árbol limpio durante la revisión.
- Revisor distinto del productor.

## Veredicto

**NO-GO**, con un hallazgo bloqueante de severidad alta.

## Hallazgo bloqueante

La ruta nueva de O4-02 en `application` presenta, relee el reloj autoritativo
y verifica en el orden correcto. Sin embargo, el paquete `ports` conserva dos
casos de uso multipuerto completos:

- `ValidarRCConFuente`;
- `CalcularCosteConFuente`.

Ambos capturan `reloj.Ahora()` antes de obtener la presentación y entregan ese
instante anterior a la verificación. Por tanto, sigue existiendo una ruta que
puede aceptar una credencial que ya no cubra el horizonte mínimo después del
tiempo consumido por el proveedor.

Contraejemplo reproducible:

1. el reloj devuelve `t0`;
2. la credencial vence en `t0 + 6 s`;
3. el presentador avanza el reloj hasta `t0 + 2 s`;
4. el horizonte exigido es de cinco segundos;
5. la ruta residual verifica contra `t0` y acepta, aunque debería verificar
   contra `t0 + 2 s` y rechazar.

La corrección debe eliminar la coordinación multipuerto de `ports` y garantizar
que toda ruta relee el tiempo autoritativo después de cada presentación.

## Evidencia favorable que no levanta el bloqueo

- Focales repetidas cincuenta veces: correctas.
- Focales con detector de carreras, cinco repeticiones: correctas.
- Regresiones de tiempo, K1, K2, revocación y estado actual repetidas cien
  veces: correctas.
- Pruebas globales frescas: correctas.
- `go vet ./...` y `go mod verify`: correctos.
- Formato, diff y tamaños: correctos.
- Gitleaks: cuatro commits, cero fugas.
- Integración virtual sobre `764fd52`: sin conflictos de código.

Estas pruebas acreditan la ruta nueva, pero no alcanzaban la ruta heredada que
mantiene el defecto estructural.

## Requisito para nueva revisión

El productor debe aportar una regresión determinista del contraejemplo, retirar
la coordinación residual de `ports` sin abrir un ciclo de dependencias y
repetir las puertas focales, carrera, globales, estáticas y de secretos. La
sexta corrección necesitará una revisión independiente nueva antes de
integrarse.
