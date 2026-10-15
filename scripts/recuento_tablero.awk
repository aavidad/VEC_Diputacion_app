# Recuento del tablero de tareas de VEC.
# Uso: awk -F'|' -f scripts/recuento_tablero.awk docs/producto/TABLERO_TAREAS.md
# Comprueba estados e identificadores, saca el recuento por fase y estado
# y lista las tareas pendientes cuyas dependencias están hechas o fusionadas.
# Columnas: 2 ID, 5 Depende de, 8 Equipo, 10 Estado. Sale con 1 si hay errores.
function limpia(s) { gsub(/^ +| +$/, "", s); return s }
$2 ~ /^ F[0-7]-[0-9][0-9] $/ {
  id = limpia($2); fase = substr(id, 2, 1); estado = limpia($10)
  if (id in est) { print "ID repetido: " id; mal = 1 }
  dep[id] = limpia($5); est[id] = estado; eq[id] = limpia($8); orden[++n] = id
  cuenta[fase "," estado]++; por_fase[fase]++; por_estado[estado]++
  if (estado !~ /^(Pendiente|Bloqueada|En curso|En revisión|Fusionada|Hecha|Aparcada|Cancelada)$/) {
    print "Estado no válido en " id ": " estado; mal = 1
  }
}
END {
  split("Pendiente,Bloqueada,En curso,En revisión,Fusionada,Hecha,Aparcada,Cancelada", E, ",")
  printf "| Fase"; for (i = 1; i <= 8; i++) printf " | %s", E[i]; print " | Total |"
  for (f = 0; f <= 7; f++) {
    printf "| %d", f; for (i = 1; i <= 8; i++) printf " | %d", cuenta[f "," E[i]] + 0
    print " | " por_fase[f] + 0 " |"
  }
  printf "| Total"; for (i = 1; i <= 8; i++) printf " | %d", por_estado[E[i]] + 0; print " | " n " |"
  print ""; print "Empezables (pendientes con dependencias hechas o fusionadas):"
  for (k = 1; k <= n; k++) {
    id = orden[k]; if (est[id] != "Pendiente") continue
    d = dep[id]; if (d ~ /fase/) continue; listo = 1
    while (match(d, /F[0-7]-[0-9][0-9]/)) {
      x = substr(d, RSTART, RLENGTH); d = substr(d, RSTART + RLENGTH)
      if (!(x in est)) { print "Dependencia desconocida en " id ": " x; mal = 1 }
      if (est[x] != "Hecha" && est[x] != "Fusionada") listo = 0
    }
    if (listo) { print "  " id " (" eq[id] ")"; total++; fases[substr(id, 2, 1)]++ }
  }
  print "Total: " total + 0
  for (f = 0; f <= 7; f++) if (fases[f]) print "  fase " f ": " fases[f]
  exit mal
}
