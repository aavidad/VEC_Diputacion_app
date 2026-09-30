package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func propuestaRegimenTurnoPrueba() DatosPropuestaRegimenTurno {
	return DatosPropuestaRegimenTurno{
		VersionContrato:       VersionContratoRegimenTurnoPropuesto,
		PoliticaRef:           "politica:bolsa:orden:1",
		Version:               1,
		BolsaRef:              "bolsa:1",
		FuenteEvidenciaRef:    "evidencia:regla:1",
		FuenteEvidenciaSHA256: strings.Repeat("a", 64),
		DefinidaEn:            time.Date(2026, 9, 30, 10, 0, 0, 123000000, time.UTC),
		ActorProponenteRef:    "actor:rrhh:1",
		TipoLista:             TipoListaRotatoria,
		ReposicionTrasCese:    ReposicionMismaPosicion,
		OperacionTurno:        OperacionSucesivoDesdeTurnoConsumido,
		ReglaRef:              "regla:sucesivo:1",
		ReglaVersion:          1,
		ReglaHuellaSHA256:     strings.Repeat("b", 64),
	}
}

func TestPropuestaRegimenTurnoSeleccionaUnaOperacionIndependienteDeTipoYReposicion(t *testing.T) {
	base := propuestaRegimenTurnoPrueba()
	for _, elegida := range []OperacionOrdenTurno{OperacionPrelacionInicial, OperacionSucesivoDesdeTurnoConsumido} {
		t.Run(string(elegida), func(t *testing.T) {
			datos := base
			datos.OperacionTurno = elegida
			propuesta, err := NuevaPropuestaRegimenTurnoGobernado(datos)
			if err != nil || propuesta.Validar() != nil || propuesta.Datos().OperacionTurno != elegida {
				t.Fatalf("operacion elegida rechazada: %v", err)
			}
			cerrada := datos
			cerrada.TipoLista = TipoListaCerrada
			otraLista, err := NuevaPropuestaRegimenTurnoGobernado(cerrada)
			if err != nil || otraLista.Datos().OperacionTurno != elegida || otraLista.HuellaContenidoSHA256() == propuesta.HuellaContenidoSHA256() {
				t.Fatal("tipo de lista determino el avance o no quedo fijado")
			}
			otraReposicion := datos
			otraReposicion.ReposicionTrasCese = ReposicionFinLista
			propuestaReposicion, err := NuevaPropuestaRegimenTurnoGobernado(otraReposicion)
			if err != nil || propuestaReposicion.Datos().OperacionTurno != elegida || propuestaReposicion.HuellaContenidoSHA256() == propuesta.HuellaContenidoSHA256() {
				t.Fatal("reposicion tras cese determino el avance o no quedo fijada")
			}
		})
	}
}

func TestPropuestaRegimenTurnoRechazaContratoV1OperacionDesconocidaYDatosInvalidos(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*DatosPropuestaRegimenTurno)
	}{
		{"contrato_b18_v1", func(d *DatosPropuestaRegimenTurno) { d.VersionContrato = 1 }},
		{"contrato_cero", func(d *DatosPropuestaRegimenTurno) { d.VersionContrato = 0 }},
		{"version_cero", func(d *DatosPropuestaRegimenTurno) { d.Version = 0 }},
		{"fuente_sin_huella", func(d *DatosPropuestaRegimenTurno) { d.FuenteEvidenciaSHA256 = "" }},
		{"fecha_local", func(d *DatosPropuestaRegimenTurno) { d.DefinidaEn = d.DefinidaEn.In(time.FixedZone("local", 3600)) }},
		{"fecha_no_canonica", func(d *DatosPropuestaRegimenTurno) { d.DefinidaEn = d.DefinidaEn.Add(time.Nanosecond) }},
		{"politica_ref_invalida", func(d *DatosPropuestaRegimenTurno) { d.PoliticaRef = " politica:1" }},
		{"bolsa_ref_invalida", func(d *DatosPropuestaRegimenTurno) { d.BolsaRef = "bolsa:*" }},
		{"fuente_ref_invalida", func(d *DatosPropuestaRegimenTurno) { d.FuenteEvidenciaRef = "" }},
		{"actor_ref_invalida", func(d *DatosPropuestaRegimenTurno) { d.ActorProponenteRef = "actor con espacio" }},
		{"tipo_lista_abierto", func(d *DatosPropuestaRegimenTurno) { d.TipoLista = "fija" }},
		{"operacion_vacia", func(d *DatosPropuestaRegimenTurno) { d.OperacionTurno = "" }},
		{"operacion_desconocida", func(d *DatosPropuestaRegimenTurno) { d.OperacionTurno = "otro_avance" }},
		{"reposicion_como_avance", func(d *DatosPropuestaRegimenTurno) { d.OperacionTurno = OperacionOrdenTurno(ReposicionFinLista) }},
		{"regla_sin_version", func(d *DatosPropuestaRegimenTurno) { d.ReglaVersion = 0 }},
		{"regla_ref_invalida", func(d *DatosPropuestaRegimenTurno) { d.ReglaRef = "regla:*" }},
		{"regla_huella_invalida", func(d *DatosPropuestaRegimenTurno) { d.ReglaHuellaSHA256 = strings.Repeat("Z", 64) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := propuestaRegimenTurnoPrueba()
			caso.alterar(&datos)
			if _, err := NuevaPropuestaRegimenTurnoGobernado(datos); !errors.Is(err, ErrRegimenTurnoGobernadoInvalido) {
				t.Fatalf("se acepto definicion invalida: %v", err)
			}
		})
	}
}

func TestPropuestaRegimenTurnoHuellaIncluyeEleccionYProcedencia(t *testing.T) {
	base := propuestaRegimenTurnoPrueba()
	original, err := NuevaPropuestaRegimenTurnoGobernado(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(original.HuellaContenidoSHA256()) != 64 {
		t.Fatal("falta huella de contenido")
	}
	for _, caso := range []struct {
		nombre  string
		alterar func(*DatosPropuestaRegimenTurno)
	}{
		{"operacion", func(d *DatosPropuestaRegimenTurno) { d.OperacionTurno = OperacionPrelacionInicial }},
		{"fuente", func(d *DatosPropuestaRegimenTurno) { d.FuenteEvidenciaRef = "evidencia:regla:2" }},
		{"actor", func(d *DatosPropuestaRegimenTurno) { d.ActorProponenteRef = "actor:rrhh:2" }},
		{"version", func(d *DatosPropuestaRegimenTurno) { d.Version = 2 }},
		{"regla", func(d *DatosPropuestaRegimenTurno) { d.ReglaHuellaSHA256 = strings.Repeat("d", 64) }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := propuestaRegimenTurnoPrueba()
			caso.alterar(&datos)
			otra, err := NuevaPropuestaRegimenTurnoGobernado(datos)
			if err != nil || otra.HuellaContenidoSHA256() == original.HuellaContenidoSHA256() {
				t.Fatalf("eleccion o procedencia fuera de la huella: %v", err)
			}
		})
	}
	if (PropuestaRegimenTurnoGobernado{}).Validar() == nil {
		t.Fatal("se admitio el valor cero")
	}
}
