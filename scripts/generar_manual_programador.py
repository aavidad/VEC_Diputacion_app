#!/usr/bin/env python3
"""Genera docs/manual_programador/ a partir de la documentacion Go del repositorio.

Uso:
    python3 scripts/generar_manual_programador.py

Requiere la toolchain Go del proyecto. Convierte la salida de `go doc -all`
de cada paquete en Markdown: las firmas exportadas quedan en bloques de
codigo y los comentarios de documentacion en texto. Los ficheros generados
no deben editarse a mano; el punto de entrada es docs/manual_programador/LEEME.md.
"""

from __future__ import annotations

import re
import subprocess
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DESTINO = RAIZ / "docs" / "manual_programador"
MODULO = "vec-diputacion-granada"

SECCIONES = {
    "CONSTANTS": "Constantes",
    "VARIABLES": "Variables",
    "FUNCTIONS": "Funciones",
    "TYPES": "Tipos",
}

# Fichero de destino y titulo por area. El orden define el indice.
AREAS = [
    ("cmd_y_configuracion.md", "Arranque, composicion y configuracion"),
    ("vec_dominio.md", "Nucleo VEC: dominio"),
    ("vec_puertos.md", "Nucleo VEC: puertos"),
    ("vec_aplicacion.md", "Nucleo VEC: aplicacion y dobles de prueba"),
    ("vec_adaptadores.md", "Nucleo VEC: adaptadores"),
    ("modulo_bolsa.md", "Modulo Bolsa"),
    ("modulos_personal_cronos_dietas.md", "Modulos Personal, Cronos, Dietas y Administracion"),
    ("nucleo_candidate.md", "Nucleo heredado de candidatos (Bolsa)"),
    ("compartido.md", "Paquetes compartidos"),
]

# Resumen curado para paquetes sin comentario de paquete propio.
RESUMENES = {
    "cmd/vec-server": "Composicion canonica y arranque del servidor HTTP del portal VEC.",
    "cmd/bolsa-server": "Centinela retirado: falla cerrado y no arranca ningun servidor.",
    "config": "Carga y validacion de la configuracion canonica por variables de entorno.",
    "internal/app/bootstrap": "Composicion de la API y montaje de modulos para el arranque.",
    "internal/app/server": "Construccion del servidor HTTP con limites y tiempos canonicos.",
    "internal/candidate/adapters/auth": "Autenticadores del nucleo heredado de Bolsa, incluido el fake local de pruebas.",
    "internal/candidate/adapters/handler": "Handlers HTTP de la API Bolsa heredada.",
    "internal/candidate/adapters/repository": "Repositorios en memoria y durables del nucleo heredado de Bolsa.",
    "internal/candidate/application": "Casos de uso heredados de candidatos de Bolsa.",
    "internal/candidate/domain": "Tipos y reglas puras del dominio heredado de candidatos.",
    "internal/candidate/ports": "Contratos hexagonales del nucleo heredado de Bolsa.",
    "internal/candidate/usecases": "Casos de uso del flujo administrativo heredado.",
    "internal/modules/administracion": "Manifiesto del modulo Administracion para el shell VEC.",
    "internal/modules/bolsa": "Manifiesto del modulo Bolsa: identidad, permisos y menus para el shell VEC.",
    "internal/modules/cronos": "Manifiesto del modulo Cronos: fichajes, permisos, vacaciones y saldos.",
    "internal/modules/cronos/adapters/memory": "Adaptadores en memoria del modulo Cronos.",
    "internal/modules/cronos/application": "Casos de uso del modulo Cronos.",
    "internal/modules/cronos/domain": "Reglas puras del dominio Cronos.",
    "internal/modules/cronos/ports": "Contratos hexagonales del modulo Cronos.",
    "internal/modules/dietas": "Manifiesto del modulo Dietas: comisiones, kilometraje y liquidaciones.",
    "internal/modules/personal": "Manifiesto del modulo Personal/Nominas.",
    "internal/modules/personal/adapters/file": "Adaptador de catalogo Personal sobre fichero local.",
    "internal/modules/personal/adapters/memory": "Adaptador de catalogo Personal en memoria.",
    "internal/modules/personal/application": "Casos de uso del modulo Personal/Nominas.",
    "internal/modules/personal/domain": "Reglas puras del dominio Personal: RPT, puestos y categorias.",
    "internal/modules/personal/ports": "Contratos hexagonales del modulo Personal.",
    "internal/shared/i18n": "Catalogo de internacionalizacion compartido con fallback espanol.",
    "internal/vec/adapters/httpapi": "Adaptador HTTP del shell VEC: rutas publicas y privadas.",
    "internal/vec/adapters/memory": "Adaptadores en memoria del nucleo VEC para pruebas y arranque local.",
    "internal/vec/adapters/seguridad": "Adaptadores criptograficos del nucleo: HMAC, AEAD y atestacion.",
    "internal/vec/application": "Casos de uso del shell VEC: modulos, auditoria, documentos, flujos y cotejo.",
    "internal/vec/domain": "Tipos puros del shell VEC, sin HTTP ni persistencia concreta.",
    "internal/vec/ports": "Contratos hexagonales del nucleo VEC: autorizacion, auditoria, documental y almacen.",
}

PREAMBULO = """# Manual del programador

Referencia de todas las funciones, tipos, constantes y variables exportadas
del portal VEC, organizada por capas. Cada entrada muestra la firma Go y su
comentario de documentacion: para que sirve y como se usa. Los ejemplos de
uso canonicos de cada paquete son sus propios ficheros `*_test.go`.

Este manual se genera con:

```bash
python3 scripts/generar_manual_programador.py
```

No editar a mano los ficheros de este directorio: cualquier correccion debe
hacerse en los comentarios de documentacion del codigo Go (o en el script) y
regenerarse.

## Vision general de la arquitectura

La aplicacion es un shell modular (VEC) que agrega modulos independientes
(Personal/Nominas, Cronos, Dietas, Bolsa, Administracion). Cada capa sigue
arquitectura hexagonal con esta regla de dependencias, siempre hacia dentro:

```
adapters  ->  application  ->  ports  ->  domain
   ^                                        |
   |  (implementan los contratos de ports)  |
   +----------------------------------------+
```

- `domain`: reglas puras, sin HTTP, sin persistencia y sin efectos.
- `ports`: contratos (interfaces y tipos de intercambio). Declaran que se
  necesita, nunca como se implementa.
- `application`: casos de uso probados contra memoria; orquestan dominio y
  puertos y aplican autorizacion por caso de uso.
- `adapters`: implementaciones concretas (HTTP, memoria, PostgreSQL, S3,
  PDF/DOCX, criptografia). Solo aqui hay infraestructura real.
- `cmd/vec-server` y `config` componen todo: es el unico punto de arranque
  soportado.

Convenciones transversales del codigo:

- Fallo cerrado: ante configuracion invalida, dependencia ausente o dato no
  canonico, la operacion se deniega; no hay valores por defecto permisivos.
- Autorizacion de lista positiva sin comodines; las decisiones se versionan
  y dejan recibo auditable.
- Los datos sensibles (HMAC, evidencias, capacidades) no se serializan por
  formatos genericos: los tipos protegidos fallan al formatearse.
- Identificadores y mensajes nuevos en espanol; el codigo heredado de
  candidatos conserva nombres en ingles.
- Puerta de calidad local obligatoria: `go test ./...` (completa:
  `scripts/verificar_calidad.sh`).

Documentos de contexto recomendados antes de tocar codigo:

- [Arquitectura tecnica modular del portal](../portal_vec/arquitectura_tecnica.md)
- [Contrato de modulos VEC](../portal_vec/contrato_modulos_vec.md)
- [Registro de decisiones y mejoras](../portal_vec/registro_decisiones.md)

## Ficheros del manual

"""


def go_list() -> list[str]:
    salida = subprocess.check_output(["go", "list", "./..."], cwd=RAIZ, text=True)
    return [linea.strip() for linea in salida.splitlines() if linea.strip()]


def go_doc(paquete: str) -> str:
    return subprocess.check_output(
        ["go", "doc", "-all", paquete], cwd=RAIZ, text=True,
        stderr=subprocess.DEVNULL,
    )


def area_de(rel: str) -> str:
    if rel.startswith("cmd/") or rel == "config" or rel.startswith("internal/app"):
        return "cmd_y_configuracion.md"
    if rel == "internal/vec/domain":
        return "vec_dominio.md"
    if rel == "internal/vec/ports":
        return "vec_puertos.md"
    if rel in ("internal/vec/application", "internal/vec/pruebas"):
        return "vec_aplicacion.md"
    if rel.startswith("internal/vec/adapters"):
        return "vec_adaptadores.md"
    if rel.startswith("internal/modules/bolsa"):
        return "modulo_bolsa.md"
    if rel.startswith("internal/modules/"):
        return "modulos_personal_cronos_dietas.md"
    if rel.startswith("internal/candidate"):
        return "nucleo_candidate.md"
    return "compartido.md"


def ancla(titulo: str) -> str:
    """Ancla estilo GitHub para un encabezado Markdown."""
    limpio = re.sub(r"[^\w\s-]", "", titulo.lower())
    return re.sub(r"\s+", "-", limpio.strip())


def primera_frase(texto: str) -> str:
    plano = " ".join(texto.split())
    if not plano:
        return ""
    corte = plano.find(". ")
    return plano[: corte + 1] if corte != -1 else plano


def paquete_a_markdown(rel: str, salida: str) -> tuple[str, str]:
    """Convierte la salida de go doc -all en Markdown.

    Devuelve (markdown, resumen para el indice).
    """
    lineas = salida.splitlines()
    md: list[str] = [f"## Paquete `{rel}`", ""]
    codigo: list[str] = []

    def volcar_codigo() -> None:
        while codigo and not codigo[-1].strip():
            codigo.pop()
        if codigo:
            md.append("```go")
            md.extend(codigo)
            md.append("```")
            md.append("")
            codigo.clear()

    # Comentario de paquete: texto sin sangria hasta la primera seccion.
    # Los comandos no imprimen la clausula "package"; solo se salta si existe.
    indice = 1 if lineas and lineas[0].startswith("package ") else 0
    doc_paquete: list[str] = []
    while indice < len(lineas) and lineas[indice].strip() not in SECCIONES:
        doc_paquete.append(lineas[indice])
        indice += 1
    texto_paquete = "\n".join(doc_paquete).strip("\n")

    resumen = RESUMENES.get(rel) or primera_frase(texto_paquete)
    if resumen:
        md.append(f"> {resumen}")
        md.append("")
    if texto_paquete.strip():
        md.append(texto_paquete)
        md.append("")

    while indice < len(lineas):
        linea = lineas[indice]
        contenido = linea.strip()
        if contenido in SECCIONES and not linea.startswith((" ", "\t")):
            volcar_codigo()
            md.append(f"### {SECCIONES[contenido]}")
            md.append("")
        elif not contenido:
            if codigo:
                codigo.append("")
            else:
                md.append("")
        elif linea.startswith("    ") and not linea.startswith("\t"):
            # Documentacion del simbolo: go doc la sangra con 4 espacios.
            volcar_codigo()
            md.append(linea[4:])
        else:
            codigo.append(linea)
        indice += 1
    volcar_codigo()

    # Compactar lineas en blanco repetidas.
    compacto: list[str] = []
    for linea in md:
        if linea == "" and compacto and compacto[-1] == "":
            continue
        compacto.append(linea)
    return "\n".join(compacto).rstrip() + "\n", resumen


def main() -> int:
    paquetes = go_list()
    DESTINO.mkdir(parents=True, exist_ok=True)

    titulos = dict(AREAS)
    cuerpos: dict[str, list[str]] = {fichero: [] for fichero, _ in AREAS}
    indice_global: list[tuple[str, str, str]] = []  # (rel, fichero, resumen)

    for paquete in paquetes:
        rel = paquete.removeprefix(MODULO + "/")
        if rel == MODULO:
            rel = "."
        fichero = area_de(rel)
        try:
            salida = go_doc(paquete)
        except subprocess.CalledProcessError:
            continue
        markdown, resumen = paquete_a_markdown(rel, salida)
        cuerpos[fichero].append(markdown)
        indice_global.append((rel, fichero, resumen))

    for fichero, titulo in AREAS:
        secciones = cuerpos[fichero]
        if not secciones:
            continue
        contenido = [
            f"# {titulo}",
            "",
            "Parte del [Manual del programador](LEEME.md). Fichero generado con",
            "`scripts/generar_manual_programador.py`; no editar a mano.",
            "",
        ]
        contenido.extend(secciones)
        (DESTINO / fichero).write_text(
            "\n".join(contenido).rstrip() + "\n", encoding="utf-8"
        )

    lineas_leeme = [PREAMBULO]
    for fichero, titulo in AREAS:
        if cuerpos[fichero]:
            lineas_leeme.append(f"- [{titulo}]({fichero})")
    lineas_leeme.append("")
    lineas_leeme.append("## Indice de paquetes")
    lineas_leeme.append("")
    lineas_leeme.append("| Paquete | Area | Para que sirve |")
    lineas_leeme.append("| --- | --- | --- |")
    for rel, fichero, resumen in indice_global:
        enlace = f"{fichero}#{ancla(f'Paquete `{rel}`')}"
        lineas_leeme.append(f"| [`{rel}`]({enlace}) | {titulos[fichero]} | {resumen} |")
    (DESTINO / "LEEME.md").write_text(
        "\n".join(lineas_leeme).rstrip() + "\n", encoding="utf-8"
    )

    total = sum(len(v) for v in cuerpos.values())
    print(f"Manual generado en {DESTINO.relative_to(RAIZ)}: {total} paquetes.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
