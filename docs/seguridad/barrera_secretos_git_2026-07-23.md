# Barrera de secretos antes de GitHub

Fecha: 23 de julio de 2026.

## Regla

Ninguna credencial, contraseña, clave, token, material criptográfico o dato
personal real puede entrar en Git, aunque aparezca en pruebas, documentación,
commits que se pretendan borrar después o ramas temporales.

Los datos sintéticos de pruebas deben usar los prefijos explícitos
`PRUEBA_NO_SECRETO_` o `SINTETICO_NO_SECRETO_`. Que un fichero termine en
`_test.go` no constituye una excepción.

Las sustituciones `psql` del tipo `PASSWORD :'variable'` no contienen el
valor y quedan fuera de la regla propia. La asignación literal a
`password`, `contraseña`, `clave` o `clave_*`, y los pares directos
`usuario/contraseña` o `usuario:contraseña`, se rechazan.

## Barreras activas

1. GitHub mantiene habilitados `secret scanning` y `push protection`.
2. `.gitleaks.toml` amplía las reglas comunes con vocabulario castellano y
   credenciales explícitas.
3. `.githooks/pre-push` inspecciona exactamente los commits que se pretenden
   enviar y falla cerrado si `gitleaks` no está instalado.
4. La integración continua repite el análisis con
   `gitleaks/gitleaks-action` fijada por SHA y sin persistir credenciales de
   `checkout`.
5. `.gitleaksignore` solo contiene huellas exactas de falsos positivos
   históricos revisados; no admite comodines por ruta, prueba o regla.
6. Toda rama de agente se revisa antes de integrarse. Una rama con un secreto
   se reconstruye desde una base limpia: un commit posterior que lo borre no
   sanea la historia.
7. `scripts/tests/test_gitleaks_config.sh` congela los casos que deben
   bloquearse y las sustituciones o sentinelas sintéticas admitidas.

La activación local del gancho es:

```bash
git config core.hooksPath .githooks
```

## Incidente local O3-02

La revisión independiente de O3-02 detectó una credencial real copiada desde
la conversación a un vector adversarial. La rama no se había integrado ni
publicado y ninguna referencia remota contiene ese commit.

Medidas obligatorias:

- no integrar ni publicar la rama;
- reconstruir el trabajo desde la rama segura, sin reutilizar sus commits;
- rotar o revocar la credencial fuera del repositorio;
- revisar sesiones y accesos asociados;
- eliminar la referencia y objetos locales contaminados cuando la
  reconstrucción segura haya sido verificada.

El valor de la credencial no se reproduce en esta memoria.
