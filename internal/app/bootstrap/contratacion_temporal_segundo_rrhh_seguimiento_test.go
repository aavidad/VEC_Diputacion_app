package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestSeleccionSoporteSeguimientoRRHHUsaHojaVerificadaYSujeto(t *testing.T) {
	sello := &selloConsultasContratacionTemporalDesarrollo{}
	original := identidadRRHHOriginalPrueba()
	segundo := original
	segundo.huella = sha256.Sum256([]byte("hoja-segunda"))
	segundo.principal = clonarPrincipalDesarrollo(original.principal)
	segundo.principal.ID = "desarrollo:segunda-rrhh"
	segundo.principal.Attributes["certificate_sha256"] = hex.EncodeToString(segundo.huella[:])
	base := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: original.principal.ID, certificadoSHA256: original.principal.Attributes["certificate_sha256"]}
	adicional := &soporteAltaContratacionTemporalDesarrollo{sello: sello, principalID: segundo.principal.ID, certificadoSHA256: segundo.principal.Attributes["certificate_sha256"]}
	selector := seleccionSoporteSeguimientoRRHH{original: base, adicional: adicional}
	capacidad := func(i identidadCertificadoDesarrollo, ruta string) context.Context {
		return context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{},
			capacidadConsultaContratacionTemporalDesarrollo{sello: sello, ruta: ruta, metodo: http.MethodPost, principal: i.principal})
	}
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		esperado *soporteAltaContratacionTemporalDesarrollo
	}{
		{"original", capacidad(original, httpinterno.RutaNoIncorporaciones), base},
		{"adicional", capacidad(segundo, httpinterno.RutaNoIncorporaciones), adicional},
		{"adicional en cese", capacidad(segundo, httpinterno.RutaCesesNombramiento), nil},
		{"adicional en GINPIX", capacidad(segundo, httpinterno.RutaConfirmacionesGINPIX), nil},
		{"sin capacidad", context.Background(), nil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			got, ok := selector.elegir(caso.ctx)
			if got != caso.esperado || ok != (caso.esperado != nil) {
				t.Fatalf("soporte=%p válido=%v", got, ok)
			}
		})
	}
	autoridadOriginal := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: base}}
	autoridadAdicional := &autoridadSeguimientoCeseDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: adicional}}
	autoridadOriginal.adicional = autoridadAdicional
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaNoIncorporaciones)) != autoridadAdicional {
		t.Fatal("el consumidor c22 no selecciona el soporte de la segunda hoja")
	}
	if autoridadOriginal.paraPeticion(capacidad(segundo, httpinterno.RutaCesesNombramiento)) != autoridadOriginal {
		t.Fatal("la segunda hoja selecciona el soporte en una ruta distinta de c22")
	}
	falsa := segundo
	falsa.principal = clonarPrincipalDesarrollo(segundo.principal)
	falsa.principal.ID = original.principal.ID
	if _, ok := selector.elegir(capacidad(falsa, httpinterno.RutaNoIncorporaciones)); ok {
		t.Fatal("sujeto original con hoja segunda aceptado")
	}
	falsa = segundo
	falsa.principal = clonarPrincipalDesarrollo(segundo.principal)
	falsa.principal.Attributes["certificate_sha256"] = original.principal.Attributes["certificate_sha256"]
	if _, ok := selector.elegir(capacidad(falsa, httpinterno.RutaNoIncorporaciones)); ok {
		t.Fatal("hoja original con sujeto segundo aceptada")
	}
}

// La asignación del segundo técnico vale tras reiniciar tanto en su forma
// sembrada (solo organización) como ceñida a un expediente por una operación
// anterior; cualquier otra dimensión, organización o concesión se rechaza.
func TestInstantaneaSegundoRRHHAceptaSemillaYAsignacionCenida(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	const principal, perfil = "per_0123456789abcdef0123456789abcdef", "prf_0123456789abcdef0123456789abcdef"
	org := vecdomain.AmbitoPerfil{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}
	exp := vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:ct:prueba"}}
	fase := vecdomain.AmbitoPerfil{Clave: "fase_previa", Valores: []string{string(domain.FaseNombramiento)}}
	estado := vecdomain.AmbitoPerfil{Clave: "estado_previo", Valores: []string{string(domain.EstadoEnCurso)}}
	concesion := vecdomain.ConcesionRol{Accion: string(domain.AccionRegistrarNoIncorporacion), ModuloID: ports.ModuloContratacion,
		TipoRecurso: ports.TipoRecursoNoIncorporacion, Finalidades: []string{ports.FinalidadRegistrarNoIncorporacion},
		GarantiaMinima: vecdomain.AuthAssuranceHigh}
	nueva := func(ambitos []vecdomain.AmbitoPerfil, concesiones ...vecdomain.ConcesionRol) vecdomain.InstantaneaAutorizacion {
		t.Helper()
		i, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principal, perfil, ahora,
			"seguimiento_prueba", "Seguimiento de prueba", "seguimiento_prueba", concesiones, ambitos)
		if err != nil {
			t.Fatal(err)
		}
		return i
	}
	otraOrg := vecdomain.AmbitoPerfil{Clave: "organizacion_ref", Valores: []string{"organizacion:otra"}}
	otraFase := vecdomain.AmbitoPerfil{Clave: "fase_previa", Valores: []string{string(domain.FaseFiscalizacion)}}
	unidad := vecdomain.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{"unidad:prueba"}}
	otra := concesion
	otra.Accion = string(domain.AccionRechazarNoIncorporacion)
	for _, caso := range []struct {
		nombre string
		i      vecdomain.InstantaneaAutorizacion
		valida bool
	}{
		{"semilla", nueva([]vecdomain.AmbitoPerfil{org}, concesion), true},
		{"ceñida", nueva([]vecdomain.AmbitoPerfil{org, exp, fase, estado}, concesion), true},
		{"otra organización", nueva([]vecdomain.AmbitoPerfil{otraOrg}, concesion), false},
		{"otra fase", nueva([]vecdomain.AmbitoPerfil{org, exp, otraFase, estado}, concesion), false},
		{"dimensión ajena", nueva([]vecdomain.AmbitoPerfil{org, unidad}, concesion), false},
		{"sin organización", nueva([]vecdomain.AmbitoPerfil{exp}, concesion), false},
		{"sin concesión c22", nueva([]vecdomain.AmbitoPerfil{org}, otra), false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := instantaneaSegundoRRHHNoIncorporacionValida(caso.i, principal, perfil, ahora); got != caso.valida {
				t.Fatalf("válida=%v, esperada %v", got, caso.valida)
			}
		})
	}
	if instantaneaSegundoRRHHNoIncorporacionValida(nueva([]vecdomain.AmbitoPerfil{org}, concesion), "per_otro0000000000000000000000000000", perfil, ahora) {
		t.Fatal("otro actor aceptado")
	}
	base := baseSegundoRRHH(nueva([]vecdomain.AmbitoPerfil{org, exp, fase, estado}, concesion))
	if len(base.AsignacionPerfil.Ambitos) != 1 || base.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" {
		t.Fatalf("la base conserva el ámbito de una operación anterior: %+v", base.AsignacionPerfil.Ambitos)
	}
}
