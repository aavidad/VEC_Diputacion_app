package recibomaterial

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func atestacionPrueba(t *testing.T, dominio string, mensaje []byte) (DatosSolicitudAtestacion, DatosAtestacion) {
	t.Helper()
	solicitud, err := PrepararSolicitudAtestacion(dominio, mensaje)
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := NuevaAtestacionNominal(solicitud, DatosAtestacion{
		Algoritmo: AlgoritmoHMACSHA256, ClaveRef: "clave:atestacion:material:v2",
		ClaveVersion: 7, Dominio: dominio, Huella: solicitud.Huella,
		Codigo: make([]byte, sha256.Size),
	})
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, atestacion
}

func TestCapacidadesAtestacionMantienenCopiasDefensivas(t *testing.T) {
	mensaje := []byte("mensaje material estable")
	solicitud, atestacion := atestacionPrueba(t, DominioRecibo, mensaje)
	mensaje[0] ^= 1
	if bytes.Equal(mensaje, solicitud.Mensaje) {
		t.Fatal("la solicitud comparte el mensaje de entrada")
	}

	revelada, firma, err := RevelarVerificacionAtestacion(solicitud, atestacion)
	if err != nil {
		t.Fatal(err)
	}
	revelada.Mensaje[0] ^= 1
	firma.Codigo[0] ^= 1
	segundaSolicitud, segundaFirma, err := RevelarVerificacionAtestacion(solicitud, atestacion)
	if err != nil || bytes.Equal(revelada.Mensaje, segundaSolicitud.Mensaje) ||
		bytes.Equal(firma.Codigo, segundaFirma.Codigo) {
		t.Fatal("una revelacion altero la capacidad opaca")
	}
	for nombre, capacidad := range map[string]any{
		"solicitud":  solicitud,
		"atestacion": atestacion,
	} {
		if _, err := json.Marshal(capacidad); !errors.Is(err, ErrSerializacionProhibida) {
			t.Fatalf("%s permitio JSON: %v", nombre, err)
		}
		texto := fmt.Sprintf("%v|%+v|%#v", capacidad, capacidad, capacidad)
		if strings.Contains(texto, "mensaje material") || strings.Contains(texto, atestacion.ClaveRef) {
			t.Fatalf("%s filtro material sensible", nombre)
		}
	}
}

func TestCapacidadesPlanPerfilYReferenciaFallanCerradas(t *testing.T) {
	vinculo := VinculoPlan{
		Seleccion:        SeleccionPlan{Referencia: "plan:material:v2:001", Version: 5},
		ConectorLogicoID: "conector:almacen:corporativo",
		Hechos:           reciboPrueba().Hechos,
	}
	huellaVinculo, err := HuellaVinculoPlan(vinculo)
	if err != nil || !SolicitudPlanValida(vinculo, huellaVinculo) {
		t.Fatalf("vinculo valido rechazado: %v", err)
	}
	huellaPlan := [sha256.Size]byte{9}
	if !ResultadoPlanValido(vinculo, huellaVinculo, huellaPlan, huellaVinculo) {
		t.Fatal("resultado ligado rechazado")
	}
	huellaPlan = huellaVinculo
	if ResultadoPlanValido(vinculo, huellaVinculo, huellaPlan, huellaVinculo) {
		t.Fatal("reutilizacion del vinculo como plan aceptada")
	}

	perfil := perfilPrueba()
	canonico, _ := CanonicoPerfil(perfil)
	huellaPerfil := sha256.Sum256(canonico)
	_, atestacionPerfil := atestacionPrueba(t, DominioPerfil, canonico)
	if !PerfilSelladoNominalValido(perfil, huellaPerfil, atestacionPerfil) {
		t.Fatal("perfil sellado valido rechazado")
	}
	publicado := DatosPerfilPublicado{
		Referencia: perfil.Referencia, Version: perfil.Version,
		ConectorLogicoID: perfil.ConectorLogicoID, Huella: huellaPerfil, Canonico: canonico,
	}
	revelado, err := RevelarPerfilPublicado(publicado)
	if err != nil {
		t.Fatal(err)
	}
	revelado.Canonico[0] ^= 1
	if !PerfilPublicadoValido(publicado) {
		t.Fatal("la revelacion compartio el perfil publicado")
	}
	if _, err := json.Marshal(publicado); !errors.Is(err, ErrSerializacionProhibida) {
		t.Fatalf("el perfil publicado permitio JSON: %v", err)
	}

	huellaIdentidad, err := NuevaHuellaIdentidad([]byte("identidad durable"))
	resultadoReferencia := DatosResultadoReferencia{
		Referencia: "recibo:material:durable:001", HuellaIdentidad: huellaIdentidad,
	}
	if err != nil || !ResultadoReferenciaValido(huellaIdentidad, resultadoReferencia) {
		t.Fatal("referencia durable valida rechazada")
	}
	if _, err := json.Marshal(resultadoReferencia); !errors.Is(err, ErrSerializacionProhibida) {
		t.Fatalf("el resultado de referencia permitio JSON: %v", err)
	}
	texto := fmt.Sprintf("%v|%+v|%#v", resultadoReferencia, resultadoReferencia, resultadoReferencia)
	if strings.Contains(texto, resultadoReferencia.Referencia) {
		t.Fatal("el resultado de referencia filtro la referencia durable")
	}
}

func TestCapacidadReciboSelladoCubreTodosLosDominios(t *testing.T) {
	recibo := reciboPrueba()
	canonico, err := CanonicoRecibo(recibo)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(canonico)
	_, atestacion := atestacionPrueba(t, DominioRecibo, canonico)
	if !ReciboSelladoNominalValido(recibo, huella, atestacion) {
		t.Fatal("recibo sellado valido rechazado")
	}
	recibo.HuellaPlan.Suma = huella
	if ReciboSelladoNominalValido(recibo, huella, atestacion) {
		t.Fatal("huella de recibo reutilizada como plan aceptada")
	}
}
