# Encargo — revisión independiente de la quinta corrección O4-02

Fecha: 23 de julio de 2026.

## Mandato

Revisar, sin modificar ni integrar, la corrección de las consultas minimizadas
a las fuentes de cobertura. El revisor no puede ser autor de ninguno de los
cuatro commits examinados.

Fuente exacta:

- rama: `agent/ct-o4-02-rework2`;
- candidato: `5bb4aaf20ac9b848dc67346d161092d0f8ef21bd`;
- base anterior rechazada: `a0c7ecf`;
- worktree:
  `/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o4-02-rework2`;
- integración objetivo: el `HEAD` publicado de
  `feature/contratacion-temporal`.

El árbol candidato debe estar limpio. No se revisan cambios posteriores sin
commit.

## Hipótesis que se deben intentar refutar

1. Cada presentación a una fuente se verifica con una lectura autoritativa de
   reloj posterior a la propia presentación y el extremo de caducidad es
   exclusivo.
2. Una cancelación, plazo vencido, credencial caducada o autoridad actual no
   confiable impide cualquier consumo.
3. Un replay posterior a una rotación legítima K1→K2 autentica primero la
   operación actual con K2 y verifica por separado el recibo histórico con la
   evidencia pública K1 original.
4. La evidencia histórica no contiene secretos, no concede autoridad actual y
   está ligada a la petición, respuesta, atestación, confirmación y efecto
   originales.
5. Una revocación de K1 aplicable a su ventana histórica invalida el recibo;
   una rotación posterior no reescribe la confianza que correspondía a esa
   ventana.
6. Una confianza actual inválida nunca queda enmascarada por un recibo K1
   válido.
7. Desafío, credencial, presentación, verificación y coordinación de fuentes
   se ejecutan en `application`; `ports` conserva contratos y tipos opacos,
   no un caso de uso multipuerto.
8. Timeout, cancelación y concurrencia se prueban mediante colaboradores o
   relojes deterministas, sin esperas de 5 ms ni dependencia del planificador.
9. Replay exacto converge al recibo original; petición, actor, perfil,
   organización, fuente o efecto distintos no lo recuperan.
10. Errores, formatos, trazas y copias defensivas no exponen material de
    autenticación, referencias privadas ni autoridad fabricable por el
    cliente.

## Puertas mínimas

```bash
go test ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/seguridad -count=50
go test -race ./internal/modules/contrataciontemporal/application \
  ./internal/modules/contrataciontemporal/adapters/seguridad -count=5
go test ./...
go vet ./...
go mod verify
git diff --check a0c7ecf..5bb4aaf
```

También se revisan tamaños, secretos en el rango, neutralidad de canal,
castellano e i18n, denegación por defecto y una integración virtual contra el
`HEAD` publicado. Una puerta omitida se declara y no se considera superada.

## Entrega

Emitir `GO` o `NO-GO` explícito, con severidad, contraejemplos reproducibles,
líneas afectadas, SHA exacto, puertas ejecutadas y riesgos residuales. Un
`GO` acredita O4-02 aislado; no acredita todavía persistencia, API, web ni E2E
de O4.
