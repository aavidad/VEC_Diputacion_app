# Revisión F0-H0a: guardia de la autoprueba sintética

Fecha: 1 de agosto de 2026.

## Resultado

F0-H0a obtuvo dos revisiones independientes `GO`, ambas con
`P0=P1=P2=0`. El commit productor `76fde25` se integró en la rama estable como
`eb21fdd`.

La corrección sustituye una sola línea del runner y limita la autoprueba
sintética a `etapa=H0`. No modifica SQL, auxiliares, capturador, roles,
permisos, API ni datos.

## Huella revisada

```text
runner: f8670131559bddadc3939e52d548def9c0ddf733b1fa5376a12b92e5364c0646
líneas: 550
```

Artefactos inmutables:

```text
auxiliar SQL:       34a5c7b29d4b20eebc9db97d2250a12b5e9f2549f9e6d5732e7db6cabbf42a3e
auxiliar operativo: 8281ac2fe10a2c4609bfb7a87f68f69a1e71189d0d7a3ed946af231b866e2075
capturador Go:      4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902
```

Los dos candidatos A1 usados solo para la integración virtual conservaron:

```text
M010: 7c2e310d293cae87853f5c76c4a95c70eeb28ac1dfaacd38bf7d0316f7fa2c65
T010: c46b47b31a64359b3a39fb98060d3f0329e747e3f0176d3aa5fae314057f8ab3
```

## Evidencia reproducida

El productor ejecutó tres veces H0 sobre PostgreSQL 18.4. Cada revisor volvió
a ejecutar H0; el revisor adversarial repitió tres ciclos. Todos conservaron
la autoprueba nominal y de error de H0.

La integración virtual con los dos componentes A1 terminó con estado cero y
rollback exacto. La variante adversarial creó una tabla y provocó
`SQLSTATE 22012`: el runner terminó no cero, revirtió el objeto y no dejó
contenedores, etiquetas, temporales ni worktrees de ensayo.

También quedaron verdes:

- `bash -n` y ShellCheck;
- `git diff --check`;
- Gitleaks sobre el cambio y el commit;
- comprobación de write-set y tope de 550 líneas;
- huellas byte a byte de los tres artefactos inmutables;
- una ejecución H0 posterior a la integración en la rama estable.

La CI documental `30707669447`, correspondiente a la decisión previa, terminó
completamente verde. La CI técnica `30708210687` se abrió al publicar
`eb21fdd` y seguía en curso al redactar este corte.

## Alcance y continuación

H0a no cambia métricas ni habilita producción. A1 queda desbloqueada y debe
repetir tres ejecuciones literales sobre la rama estable corregida antes de
confirmar sus dos componentes. Producción permanece en `NO-GO`.
