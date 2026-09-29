package bootstrap

import (
	"testing"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestPerfilCoberturaFijadoPorAccionDeComposicion(t *testing.T) {
	const alta, cobertura = "prf_ct_alta_prueba", "prf_ct_cobertura_prueba"
	originales := descriptoresFronterasContratacionTemporalDesarrollo(alta, []string{alta})
	resultado, err := asignarPerfilCoberturaCTDesarrollo(originales, alta, cobertura)
	if err != nil {
		t.Fatalf("asignar perfil de cobertura: %v", err)
	}
	for i, descriptor := range resultado {
		perfil := alta
		if rutaPerfilCoberturaCTDesarrollo(descriptor.Ruta) {
			perfil = cobertura
		}
		if descriptor.Ruta == cthttp.RutaConsultaCuadroRRHH || descriptor.Ruta == cthttp.RutaConsultaDetalleRRHH {
			continue
		}
		if len(descriptor.PerfilesActivosRef) != 1 || descriptor.PerfilesActivosRef[0] != perfil {
			t.Fatalf("ruta %s usa perfil ajeno: %v", descriptor.Ruta, descriptor.PerfilesActivosRef)
		}
		if originales[i].Ruta == cthttp.RutaPropuestaCobertura && originales[i].PerfilesActivosRef[0] != alta {
			t.Fatal("la composición alteró el catálogo de entrada")
		}
	}
	if rutaPerfilCoberturaCTDesarrollo(cthttp.RutaAltaSolicitudes) {
		t.Fatal("el alta no pertenece al perfil de cobertura")
	}
}

func TestPerfilCoberturaRechazaCatalogoIncompletoOAdulterado(t *testing.T) {
	const alta, cobertura = "prf_ct_alta_prueba", "prf_ct_cobertura_prueba"
	originales := descriptoresFronterasContratacionTemporalDesarrollo(alta, []string{alta})
	for _, caso := range []struct {
		nombre string
		mutar  func([]descriptorFronteraComunDesarrollo) []descriptorFronteraComunDesarrollo
	}{
		{"acción alterada", func(d []descriptorFronteraComunDesarrollo) []descriptorFronteraComunDesarrollo {
			for i := range d {
				if d[i].Ruta == cthttp.RutaPropuestaCobertura {
					d[i].ClaveCapacidad = "accion:cliente"
				}
			}
			return d
		}},
		{"ruta ausente", func(d []descriptorFronteraComunDesarrollo) []descriptorFronteraComunDesarrollo {
			for i := range d {
				if d[i].Ruta == cthttp.RutaPropuestaCobertura {
					return append(d[:i], d[i+1:]...)
				}
			}
			return d
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := append([]descriptorFronteraComunDesarrollo(nil), originales...)
			if _, err := asignarPerfilCoberturaCTDesarrollo(caso.mutar(copia), alta, cobertura); err == nil {
				t.Fatal("la frontera alterada debe cerrar")
			}
		})
	}
	if _, err := asignarPerfilCoberturaCTDesarrollo(originales, alta, alta); err == nil {
		t.Fatal("alta y cobertura no pueden compartir asignación")
	}
}

func TestPerfilesFijosSeparanAltaDeLegadoYCobertura(t *testing.T) {
	const legado, alta, cobertura = "prf_ct_legado_prueba", "prf_ct_alta_prueba", "prf_ct_cobertura_prueba"
	originales := descriptoresFronterasContratacionTemporalDesarrollo(legado, []string{legado})
	resultado, err := asignarPerfilesFijosCTDesarrollo(originales, legado, alta, cobertura)
	if err != nil {
		t.Fatal(err)
	}
	for _, descriptor := range resultado {
		perfil := legado
		if rutaPerfilAltaFijaCTDesarrollo(descriptor.Ruta) {
			perfil = alta
		} else if rutaPerfilCoberturaCTDesarrollo(descriptor.Ruta) {
			perfil = cobertura
		}
		if len(descriptor.PerfilesActivosRef) != 1 || descriptor.PerfilesActivosRef[0] != perfil {
			t.Fatalf("ruta %s cambió de autoridad: %v", descriptor.Ruta, descriptor.PerfilesActivosRef)
		}
	}
	if rutaPerfilAltaFijaCTDesarrollo(cthttp.RutaPropuestaCobertura) ||
		rutaPerfilAltaFijaCTDesarrollo(cthttp.RutaRegistroAnalisisRRHH) ||
		rutaPerfilCoberturaCTDesarrollo(cthttp.RutaAltaSolicitudes) {
		t.Fatal("la selección de perfiles superpone acciones")
	}
	if _, err := asignarPerfilesFijosCTDesarrollo(originales, legado, legado, cobertura); err == nil {
		t.Fatal("alta fija no puede compartir el perfil legado")
	}
}
