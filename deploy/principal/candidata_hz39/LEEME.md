# Candidata HZ39

Versión candidata congelada para la presentación. Se ensaya entera en una copia
de cidonia y, si sale bien, se instala en cidonia exactamente igual (regla 1 de
`AGENTS.md`, «Entrega y subida»). Caso de aceptación: `docs/producto/FLUJO_APP.md`,
apartado 4.0.

- **Base de partida:** lo instalado en cidonia (HZ37 + web HZ38, código
  `44218865a`; catálogo de acciones v6).
- **Código de la candidata:** el commit de `origin/main` que se fije al congelar
  (se anota aquí antes de instalar).
- **Nada se instala desde fuera de esta carpeta**: si falta algo, se añade aquí
  en una PR y se vuelve a ensayar.

## Orden de instalación

1. **SQL**: `lista_sql.txt`, en ese orden, cada una una vez y con
   `ON_ERROR_STOP`. Comprobar el SHA-256 de cada fichero antes de aplicarlo.
2. **Datos del recorrido**: `datos/` (guion repetible de K: centro 520, puesto
   9/01, plaza 2026/0033, bolsa de Administrativo).
3. **Entrada del aspirante**: Mi Bolsa interna (HZ37) por defecto. La confianza
   externa (#1120, `deploy/principal/confianza_externa_hz38.md`) solo si su
   ensayo valida el acceso y el recorrido; decisión de corte el 13/10 a las
   17:00 y se anota aquí la vía elegida.
4. **Actos de administración**, por la vía de gobierno existente y con dos
   administradores distintos:
   - inscripción RRHH (lector técnico de Carmen) cuando la vía del paso 3 lo
     permita, con reinicio vigilado;
   - plan de firma v2 de cinco pasos (`scripts/operacion/firma_ct_tres_pasos/`):
     crear y publicar; después `identidad/firma-vec.json` con los valores
     resultantes y reinicio.
5. **Arranque vigilado** con vuelta atrás automática.
6. **Recorrido** con el guion de P (`deploy/principal/ensayo_candidata/`):
   e1–e5 del aspirante y los pasos del caso, reinicio de app y PostgreSQL y
   repetición de lecturas; mismos recibos y sin duplicados.

## Estado del ensayo

| Pieza | Estado |
| --- | --- |
| SQL (11) | Aplicadas sin error en la copia de P |
| Datos del recorrido | Pendiente de `datos/` (K) |
| Vía del aspirante | Pendiente del corte de las 17:00 |
| Plan de firma y cinco firmas | En curso (H) |
| Recorrido completo y reinicio | En curso (P) |

Lo que no se haya ensayado en la copia no se instala en cidonia.
