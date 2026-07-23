# Encargo — revisión independiente O2-08B

Fecha: 23 de julio de 2026.

## Mandato

Revisar, sin modificar ni integrar, el candidato que unifica la idempotencia
entre la API interna y los clientes web, escritorio, CLI y MCP.

Fuente exacta:

- rama: `agent/ct-o2-08b-idempotencia`;
- SHA: `3e2885c`;
- worktree:
  `/home/alberto/Trabajo/VEC_Diputacion_app/.worktrees/ct-o2-08b-idempotencia`;
- base funcional O2-08A: `a736105`;
- integración objetivo: rama `feature/contratacion-temporal`, desde
  `79cb68e` o el HEAD posterior publicado.

No se revisan cambios sin commit ni se altera el candidato.

## Hipótesis que deben intentar refutarse

1. El sobre JSON contiene exactamente `clave_idempotencia` y `solicitud`.
2. La UUIDv4 es un identificador no confiable de intención, no identidad,
   autenticación, rol, permiso, organización ni capacidad.
3. La autoridad de servidor aporta exclusivamente autenticación, sesión,
   perfil y organización, con clave y solicitud vacías.
4. La misma UUID y solicitud llega sin sustitución al caso de uso, que liga
   contexto y contenido mediante HMAC.
5. Una UUID inválida, nula, duplicada, con caja alternativa o incluida en
   cabecera/query se rechaza antes de autoridad y efecto.
6. La UUID no aparece en respuesta, error, correlación, cabeceras ni estado
   durable de este adaptador.
7. El JSON sigue siendo cerrado frente a campos desconocidos, duplicados,
   `null`, Unicode no canónico, límites, segundo documento y números
   ambiguos.
8. El OpenAPI 3.1 describe el mismo sobre y mantiene cerrados todos los
   objetos.
9. El comando coincide exactamente con O2-09A; no requiere cookies,
   `localStorage`, `sessionStorage`, cabeceras de autoridad ni un adaptador
   distinto por canal.
10. Cancelación, respuesta perdida y recibo válido posterior al efecto no
    inducen una segunda intención automática.

## Puertas

```bash
go test ./internal/modules/contrataciontemporal/adapters/httpinterno -count=50
go test -race ./internal/modules/contrataciontemporal/adapters/httpinterno -count=5
go test ./...
go vet ./...
git diff --check a736105..3e2885c
node --test web/static/portal-empleado/modulos/contratacion-temporal/contratacion-temporal.test.mjs
```

La prueba web se ejecuta en un árbol combinado desechable que contenga
O2-09A; no se modifica ninguna rama. Deben comprobarse también tamaños,
Gitleaks, `git fsck` y una integración virtual contra el HEAD publicado.

## Entrega

Emitir `GO` o `NO-GO` explícito con contraejemplos reproducibles, SHA exacto,
puertas ejecutadas y estado limpio. Un GO solo acredita el adaptador aislado:
no registra ruta, no compone O2-07 y no cierra el E2E O2-10.
