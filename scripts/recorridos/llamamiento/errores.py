"""Clasificación común para ejecución como módulo o desde la CLI."""


class NoEjecutado(ValueError):
    """Falta una condición previa; no se abre el navegador."""


class FalloRecorrido(RuntimeError):
    """El recorrido empezó y una comprobación falló."""
