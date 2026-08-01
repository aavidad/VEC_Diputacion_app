# Revisión F0-A3: canon y MAC de capacidad

Fecha: 1 de agosto de 2026.

## Resultado

F0-A3 obtuvo dos revisiones independientes finales `GO`, ambas con
`P0=P1=P2=0`. El commit productor `064f1b4` se integró en la rama estable
como `806691b`.

El corte añade exclusivamente los dos componentes
`030_canon_capacidad_mac.sql`. No publica una función exterior, permiso, rol,
tabla ni autoridad.

## Huellas finales

```text
M030: f9089c7df50843458a13ff36b8a709adc5b62b67f301ebfd2cfd6b2f1d7d9993
      366 líneas; 15.415 bytes
T030: 28c03e6911559e85d84823d921e220db2ae789fed1ffa6023dce42c033e62064
      799 líneas; 34.551 bytes
```

El vector V0 de capacidad mide 1.891 bytes, sin el salto final del fixture, y
conserva la huella:

```text
d3baaa6bf9e8e757d659f42233186a799e3c0b6e9a8e5eab1b5930ca0e7f7e54
```

La preimagen V0 mide 1.280 bytes y conserva la huella:

```text
334ec3d3b1f648cee1a9a9a387d704ed448772f0e03bb4d97c08b933518be3d5
```

## Evidencia reproducida

El productor completó tres ejecuciones finales literales
`probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh --etapa A3` sobre
PostgreSQL 18.4. Ambos revisores volvieron a ejecutar la etapa y dirección la
repitió después de integrar. Todas terminaron con rollback y línea base
exactos.

La matriz acredita:

- canon cerrado de 33 campos y exactamente seis números JSON;
- preimagen de 32 valores con el dominio aprobado;
- HMAC-SHA-256 real y comparación en tiempo constante;
- secreto entre 32 y 4.096 bytes, con ambos límites incluidos;
- igualdad byte a byte con el vector Go/V0;
- cuatro cruces nominales y rechazo de cruces hostiles;
- tipos, orden, claves repetidas, omitidas o sobrantes, UTF-8, instantes,
  referencias, huellas y límites;
- firma, propietario, lenguaje, volatilidad, seguridad, paralelismo,
  configuración, coste, filas, soporte, transformación, binario y ACL exactos;
- huella SHA-256 del cuerpo de las tres funciones;
- denegación predeterminada efectiva de las funciones nuevas y del comparador
  preexistente.

Las revisiones rechazaron candidatos previos que no detectaban sustituir la
comparación constante por `=`, conceder `EXECUTE` a `PUBLIC`, cambiar el coste
o alterar el cuerpo de una función auxiliar. Las regresiones finales matan
cada mutante por separado, restauran la forma exacta y acreditan rollback.

También quedaron verdes Go normal y con detector de carreras, `go vet`, Bash,
ShellCheck, `git diff --check`, límites y Gitleaks. No quedaron contenedores,
procesos ni temporales atribuibles. La comprobación posterior a integración
volvió a terminar verde y Gitleaks no encontró fugas en el commit.

La CI técnica `30714074760` terminó completamente verde sobre `806691b`.

## Continuación

A3 no cambia métricas ni crea todavía una migración `000007` instalable. B1
sigue en revisión de su clausura catalogal; A4 espera sus revisiones finales.
Cuando B1 cierre, quedan desbloqueados B2 y, junto con A2+A3, C1. Producción
continúa en `NO-GO`.
