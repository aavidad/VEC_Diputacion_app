#!/usr/bin/env python3
"""Renueva las versiones de caché (?v=) de los JavaScript modificados.

Los .js con ?v= se sirven con caché inmutable: cuando uno cambia, todos sus
importadores deben pedirlo con una versión nueva, y como eso cambia a los
importadores, la renovación sube por el grafo. Este script:

1. toma los .js de web/static modificados frente a la rama base (por
   defecto origin/main), confirmados o no, sin contar pruebas;
2. en cada .js/.html de web/static que referencie uno de ellos con ?v=,
   sustituye la versión por la indicada, y repite con los importadores así
   modificados hasta cerrar el grafo;
3. no toca los ficheros protegidos (por defecto portal.js e index.html del
   portal, que tienen otras ramas abiertas): solo avisa de que habría que
   renovarlos al integrar.

Uso: scripts/renovar_versiones_cache.py 20260929-mi-cambio-v1 [rama_base]
"""
import os
import re
import subprocess
import sys

RAIZ = "web/static"
PROTEGIDOS = {
    "web/static/portal-empleado/portal.js",
    "web/static/portal-empleado/index.html",
}
REFERENCIA = re.compile(r"((?:\.\.?/|/)[\w./-]+\.js)\?v=([\w.-]+)")


def es_prueba(ruta):
    return ".test." in ruta or "test-helper" in ruta


def modificados(base):
    salida = subprocess.check_output(
        ["git", "diff", "--name-only", base, "--", RAIZ], text=True
    ).splitlines()
    nuevos = subprocess.check_output(
        ["git", "ls-files", "--others", "--exclude-standard", "--", RAIZ], text=True
    ).splitlines()
    return {r for r in salida + nuevos if r.endswith(".js") and not es_prueba(r) and os.path.exists(r)}


def fuentes():
    for carpeta, _, ficheros in os.walk(RAIZ):
        for nombre in ficheros:
            ruta = os.path.join(carpeta, nombre)
            if nombre.endswith((".js", ".html")) and not es_prueba(ruta):
                yield ruta


def destino(importador, ruta):
    if ruta.startswith("/"):
        return os.path.normpath(os.path.join(RAIZ, ruta.lstrip("/")))
    return os.path.normpath(os.path.join(os.path.dirname(importador), ruta))


def main():
    if len(sys.argv) not in (2, 3) or not re.fullmatch(r"[\w.-]+", sys.argv[1]):
        sys.exit(__doc__)
    version, base = sys.argv[1], (sys.argv[2] if len(sys.argv) == 3 else "origin/main")
    todas = list(fuentes())
    pendientes = sorted(modificados(base))
    vistos, avisos = set(), set()
    while pendientes:
        cambiado = pendientes.pop()
        if cambiado in vistos:
            continue
        vistos.add(cambiado)
        for importador in todas:
            if importador == cambiado:
                continue
            with open(importador, encoding="utf-8") as fichero:
                texto = fichero.read()

            def sustituir(coincidencia):
                if destino(importador, coincidencia.group(1)) != cambiado or coincidencia.group(2) == version:
                    return coincidencia.group(0)
                return f"{coincidencia.group(1)}?v={version}"

            nuevo = REFERENCIA.sub(sustituir, texto)
            if nuevo == texto:
                continue
            if importador in PROTEGIDOS:
                avisos.add(importador)
                continue
            with open(importador, "w", encoding="utf-8") as fichero:
                fichero.write(nuevo)
            print(f"{importador}: renovada la referencia a {cambiado}")
            pendientes.append(importador)
    for protegido in sorted(avisos):
        print(f"AVISO: {protegido} no se ha tocado (protegido); renovarlo al integrar.")


if __name__ == "__main__":
    main()
