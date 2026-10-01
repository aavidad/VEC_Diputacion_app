// Package inventariocopias observa bytes locales y contrasta declaraciones
// offline. No autentica los descriptores ni consulta servidores PostgreSQL.
package inventariocopias

import (
	"errors"
	"os"
	"strconv"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type Ruta struct {
	ID           string `json:"id"`
	RutaRelativa string `json:"ruta_relativa"`
}

// Descriptor expresa el contenido previsto del paquete. Nunca acredita por sí
// mismo una instalación. Las rutas pertenecen al adaptador, no al dominio.
type Descriptor struct {
	Inventario copias.Inventario `json:"inventario"`
	Rutas      []Ruta            `json:"rutas"`
}

type Informe struct {
	Alcance               string           `json:"alcance"`
	ProcedenciaPostgreSQL string           `json:"procedencia_postgresql"`
	AutorizaCopia         bool             `json:"autoriza_copia"`
	AutorizaRestauracion  bool             `json:"autoriza_restauracion"`
	Resultado             copias.Resultado `json:"resultado"`
}

// Inventariar lee exclusivamente los bytes declarados bajo raiz. PostgreSQL y
// la relación commit/binario siguen siendo declaraciones proporcionadas por el
// operador, sin atestación; el resultado no habilita efectos de plataforma.
func Inventariar(raiz string, descriptor Descriptor, observado copias.Inventario) Informe {
	informe := Informe{
		Alcance:               "inventario_offline_declarado",
		ProcedenciaPostgreSQL: "declaracion_offline_no_autenticada",
		Resultado:             copias.CompararInventarios(descriptor.Inventario, observado),
	}
	if len(copias.ValidarInventario(descriptor.Inventario)) != 0 || len(copias.ValidarInventario(observado)) != 0 {
		return informe
	}
	esperados := append([]copias.Artefacto(nil), descriptor.Inventario.Release.Binarios...)
	esperados = append(esperados, descriptor.Inventario.Release.Componentes...)
	rutas, razones := reconciliarRutas(esperados, descriptor.Rutas)
	if len(razones) != 0 {
		informe.Resultado.Estado = copias.NoComprobable
		informe.Resultado.Razones = append(informe.Resultado.Razones, razones...)
		return informe
	}
	raizAbierta, err := os.OpenRoot(raiz)
	if err != nil {
		return anadirRazon(&informe, copias.NoComprobable, "raiz_no_disponible", "raiz", "disponible", "no_disponible", "revisar_raiz_autorizada")
	}
	defer raizAbierta.Close()
	medido := observado
	medido.Release.Binarios = append([]copias.Artefacto(nil), observado.Release.Binarios...)
	medido.Release.Componentes = append([]copias.Artefacto(nil), observado.Release.Componentes...)
	archivosMedidos := map[string]*copias.Artefacto{}
	for i := range medido.Release.Binarios {
		archivosMedidos[medido.Release.Binarios[i].ID] = &medido.Release.Binarios[i]
	}
	for i := range medido.Release.Componentes {
		archivosMedidos[medido.Release.Componentes[i].ID] = &medido.Release.Componentes[i]
	}
	var huellasPrivadas []string
	for indice, artefacto := range esperados {
		clave := "archivos." + strconv.Itoa(indice)
		huella, tamano, err := leerHuella(raizAbierta, rutas[artefacto.ID], artefacto.TamanoBytes)
		if actual := archivosMedidos[artefacto.ID]; actual != nil {
			actual.SHA256, actual.TamanoBytes = huella, tamano
		}
		if err != nil {
			medido.Completo = false
			anadirFalloArchivo(&informe, clave, err)
			continue
		}
		if tamano != artefacto.TamanoBytes {
			anadirRazon(&informe, copias.Incompatible, "tamano_distinto", clave+".tamano_bytes", strconv.FormatInt(artefacto.TamanoBytes, 10), strconv.FormatInt(tamano, 10), "reponer_archivo_del_release")
			continue
		}
		if huella != artefacto.SHA256 {
			if artefacto.Tipo == "binario" || artefacto.Tipo == "web" || artefacto.Tipo == "catalogos" {
				anadirRazon(&informe, copias.Incompatible, "huella_archivo_distinta", clave+".sha256", artefacto.SHA256, huella, "reponer_archivo_del_release")
			} else {
				// El contraste de contenido privado se comunica mediante las
				// huellas del inventario entero, nunca el hash aislado del secreto.
				huellasPrivadas = append(huellasPrivadas, clave)
			}
		}
	}
	if len(huellasPrivadas) != 0 {
		esperada, obtenida := copias.HuellaInventario(descriptor.Inventario), copias.HuellaInventario(medido)
		for _, clave := range huellasPrivadas {
			anadirRazon(&informe, copias.Incompatible, "huella_archivo_distinta", clave+".inventario_sha256", esperada, obtenida, "reponer_archivo_del_release")
		}
	}
	return informe
}

func reconciliarRutas(esperados []copias.Artefacto, entradas []Ruta) (map[string]string, []copias.Razon) {
	rutas := make(map[string]string, len(entradas))
	ids := make(map[string]bool, len(esperados))
	usadas := make(map[string]bool, len(entradas))
	for _, artefacto := range esperados {
		if ids[artefacto.ID] {
			return nil, []copias.Razon{{Codigo: "artefacto_repetido", Clave: "release.artefactos", Esperado: "id_unico", Obtenido: "id_repetido", Accion: "corregir_descriptor"}}
		}
		ids[artefacto.ID] = true
	}
	for _, entrada := range entradas {
		if !ids[entrada.ID] || rutas[entrada.ID] != "" || usadas[entrada.RutaRelativa] || !rutaValida(entrada.RutaRelativa) {
			return nil, []copias.Razon{{Codigo: "ruta_no_valida", Clave: "rutas", Esperado: "id_unico_esperado_y_ruta_relativa", Obtenido: "ruta_ambigua_desconocida_o_no_relativa", Accion: "corregir_descriptor"}}
		}
		rutas[entrada.ID] = entrada.RutaRelativa
		usadas[entrada.RutaRelativa] = true
	}
	if len(rutas) != len(ids) {
		return nil, []copias.Razon{{Codigo: "rutas_incompletas", Clave: "rutas.total", Esperado: strconv.Itoa(len(ids)), Obtenido: strconv.Itoa(len(rutas)), Accion: "completar_rutas_del_descriptor"}}
	}
	return rutas, nil
}

func anadirFalloArchivo(informe *Informe, campo string, err error) Informe {
	obtenido := "fallo_lectura_no_clasificado"
	if errors.Is(err, errArchivo) {
		obtenido = "ausente_o_no_legible"
	}
	return anadirRazon(informe, copias.NoComprobable, "archivo_no_comprobable", campo, "regular_legible", obtenido, "completar_paquete_instalado")
}

func anadirRazon(informe *Informe, estado copias.Estado, codigo, clave, esperado, obtenido, accion string) Informe {
	if estado == copias.NoComprobable || informe.Resultado.Estado == copias.Compatible {
		informe.Resultado.Estado = estado
	}
	informe.Resultado.Razones = append(informe.Resultado.Razones, copias.Razon{Codigo: codigo, Clave: clave, Esperado: esperado, Obtenido: obtenido, Accion: accion})
	return *informe
}
