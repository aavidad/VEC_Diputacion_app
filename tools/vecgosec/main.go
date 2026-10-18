// El adaptador oficial carga paquetes completos y exige el análisis SSA.
package main

import (
	"github.com/securego/gosec/v2/goanalysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(goanalysis.Analyzer)
}
