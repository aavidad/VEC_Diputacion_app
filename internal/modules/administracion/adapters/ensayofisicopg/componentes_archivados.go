package ensayofisicopg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

// PrepararArchivados reutiliza para el ensayo lógico la misma validación física
// de componentes CS05. No recibe PGDATA, sólo binarios/configuración/almacenes.
func PrepararArchivados(ctx context.Context, raiz string, componentes []Componente, limite int64) ([]puertos.Componente, []string, error) {
	if len(componentes) > 128 || limite < 1 || limite > 1<<30 {
		return nil, nil, errEntrada
	}
	if prepararAuxiliar(raiz) != nil {
		return nil, nil, errRuntime
	}
	montajes := []string{"-v", filepath.Join(raiz, "verificador") + ":/verificador:ro", "-v", filepath.Join(raiz, "control") + ":/control:rw"}
	montajes = append(montajes, montajesNSS(raiz)...)
	salida := []puertos.Componente{}
	vistos := map[string]bool{}
	cuenta := &cuentaTar{}
	rutas := make([]string, len(componentes))
	var total int64
	for i, c := range componentes {
		if !idValido.MatchString(c.ID) || !tipoValido.MatchString(c.Tipo) || !huellaValida.MatchString(c.Tar.SHA256) || vistos[c.ID] || c.Tipo == "base_fisica" {
			return nil, nil, errEntrada
		}
		vistos[c.ID] = true
		rutas[i] = filepath.Join(raiz, fmt.Sprintf("archivado-%04d.tar", i))
		if copiarArchivo(ctx, c.Tar, rutas[i], limite-total) != nil {
			return nil, nil, errEntrada
		}
		info, err := os.Stat(rutas[i])
		if err != nil {
			return nil, nil, errEntrada
		}
		total += info.Size()
		if revisarTar(ctx, rutas[i], Configuracion{LimiteExtraidoBytes: limite, LimiteEntradas: 100000}, cuenta) != nil {
			return nil, nil, errEntrada
		}
	}
	for i, c := range componentes {
		destino := filepath.Join(raiz, fmt.Sprintf("archivado-%04d", i))
		if os.Mkdir(destino, 0700) != nil || extraerTar(ctx, rutas[i], destino) != nil {
			return nil, nil, errEntrada
		}
		interna := fmt.Sprintf("/componentes/%04d", i)
		montajes = append(montajes, "-v", destino+":"+interna+":ro")
		hash, bytes, err := huellaContenido(ctx, filepath.Join(destino, "contenido"))
		if err != nil {
			return nil, nil, err
		}
		salida = append(salida, puertos.Componente{ID: c.ID, Tipo: c.Tipo, RutaInterna: interna + "/contenido", SHA256: c.Tar.SHA256, ContenidoSHA256: hash, ContenidoBytes: bytes})
		if prepararMaterial(ctx, destino, Configuracion{LimiteExtraidoBytes: limite, LimiteEntradas: 100000}, cuenta, &salida[len(salida)-1]) != nil {
			return nil, nil, errEntrada
		}
	}
	return salida, montajes, nil
}
