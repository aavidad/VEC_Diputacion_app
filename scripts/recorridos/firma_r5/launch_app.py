#!/usr/bin/env python3
"""Inicia el binario con el entorno propio; los DSN no pasan por argumentos."""
import json
import os
from pathlib import Path
import sys

kit = Path(sys.argv[1])
variables = json.loads((kit / 'runtime/runtime-config.json').read_text())
environment = {'PATH': '/usr/bin:/bin', 'HOME': str(kit / 'runtime/app/home'), 'TMPDIR': '/tmp', 'TZ': 'UTC'}
environment.update(variables)
binary = str(kit / 'runtime/vec-server')
os.execve(binary, [binary], environment)
