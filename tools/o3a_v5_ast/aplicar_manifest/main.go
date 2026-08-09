package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type mutante struct{ ID, Familia, Ruta, Anterior, Posterior, Oraculo string }
type manifiesto struct{ Mutantes []mutante }

var archivo = map[string]string{
	"G1":  "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go",
	"G4":  "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go",
	"G6a": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go",
	"G6b": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go",
	"G6c": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go",
	"G7a": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go",
	"G7b": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go",
}

func fatal(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "aplicar_manifest=fallo "+f+"\n", a...)
	os.Exit(1)
}
func main() {
	if len(os.Args) != 4 {
		fatal("uso: aplicar_manifest MANIFEST ID DIRECTORIO")
	}
	b, err := os.ReadFile(os.Args[1])
	if err != nil {
		fatal("manifest: %v", err)
	}
	var m manifiesto
	if err = json.Unmarshal(b, &m); err != nil {
		fatal("json: %v", err)
	}
	for _, x := range m.Mutantes {
		if x.ID == os.Args[2] {
			n, ok := archivo[x.Ruta]
			if !ok {
				fatal("ruta %s", x.Ruta)
			}
			ruta := filepath.Join(os.Args[3], n)
			s, err := os.ReadFile(ruta)
			if err != nil {
				fatal("fuente: %v", err)
			}
			if c := bytes.Count(s, []byte(x.Anterior)); c != 1 {
				fatal("patrón cardinalidad=%d", c)
			}
			s = bytes.Replace(s, []byte(x.Anterior), []byte(x.Posterior), 1)
			if err = os.WriteFile(ruta, s, 0600); err != nil {
				fatal("escritura: %v", err)
			}
			fmt.Printf("aplicar_manifest=ok id=%s familia=%s oraculo=%s\n", x.ID, x.Familia, x.Oraculo)
			return
		}
	}
	fatal("ID ausente: %s", os.Args[2])
}
