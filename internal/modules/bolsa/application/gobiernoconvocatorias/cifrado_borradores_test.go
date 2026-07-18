package gobiernoconvocatorias

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestServicioCifraDespuesDeReservarYSellarAntesDeConfirmar(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	if e.perfiles.llamadas != 1 || e.cifrador.llamadas != 1 ||
		e.cifrador.ultima == nil || e.confirmador.ultima == nil {
		t.Fatal("no se completo la secuencia reserva-sellado-cifrado-confirmacion")
	}
	solicitudCifrado := *e.cifrador.ultima
	confirmacion := *e.confirmador.ultima
	if solicitudCifrado.Control.Estado != ResultadoDiarioReservado ||
		!solicitudCifrado.SolicitadaEn.Before(solicitudCifrado.Reserva.ArrendamientoVenceEn) ||
		!confirmacion.Cifrado.validaPara(solicitudCifrado) || confirmacion.Validar() != nil ||
		confirmacion.SolicitadaEn.Before(confirmacion.Cifrado.CifradoEn) {
		t.Fatal("el cifrado no quedo ligado a la reserva y al sellado exactos")
	}
	if _, _, _, materialEnvuelto, _, _, err :=
		confirmacion.Cifrado.EnvolturaClave.DatosParaPersistencia(); err != nil || len(materialEnvuelto) == 0 {
		t.Fatal("la confirmacion no transporta la envoltura opaca")
	}
	if _, _, textoCifrado, _, _, err :=
		confirmacion.Cifrado.SobreCifrado.DatosParaPersistencia(); err != nil || len(textoCifrado) == 0 {
		t.Fatal("la confirmacion no transporta el sobre cifrado")
	}
}

func TestCifradoRechazaCambiosDeAADYBuffers(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	original := *e.confirmador.ultima

	t.Run("revision", func(t *testing.T) {
		alterada := original
		alterada.Control.Revision++
		if alterada.Validar() == nil {
			t.Fatal("un sobre se reutilizo con otra revision")
		}
	})
	t.Run("cercado", func(t *testing.T) {
		alterada := original
		alterada.Control.Cercado++
		if alterada.Validar() == nil {
			t.Fatal("un sobre se reutilizo con otro cercado")
		}
	})
	t.Run("arrendamiento", func(t *testing.T) {
		alterada := original
		alterada.Control.ArrendamientoVenceEn = alterada.Control.ArrendamientoVenceEn.Add(-time.Second)
		if alterada.Validar() == nil {
			t.Fatal("un sobre se reutilizo con otro arrendamiento")
		}
	})
	t.Run("primaria", func(t *testing.T) {
		alterada := original
		alterada.Reserva.IdentidadPrimaria.Localizador.GeneracionClave--
		if alterada.Validar() == nil {
			t.Fatal("un sobre se reutilizo con otra primaria")
		}
	})
	t.Run("perfil gobernado", func(t *testing.T) {
		alterada := original
		perfil, err := NuevoPerfilCifradoBorrador(
			"perfil:cifrado:borradores:degradado", 2, huellaHexPrueba('b'),
			"algoritmo-aead-no-aprobado", "algoritmo-envoltura-no-aprobado",
		)
		if err != nil {
			t.Fatal(err)
		}
		alterada.PerfilCifrado = perfil
		if alterada.Validar() == nil {
			t.Fatal("se acepto un perfil distinto del gobernado para la AAD")
		}
		// Simula un conector que devuelve un resultado internamente coherente,
		// pero con algoritmos distintos de los resueltos por gobierno.
		degradado := original.Cifrado
		degradado.EnvolturaClave.perfil = perfil
		degradado.EnvolturaClave.huellaSHA256 = degradado.EnvolturaClave.calcularHuella()
		degradado.SobreCifrado.perfil = perfil
		degradado.SobreCifrado.huellaSHA256 = degradado.SobreCifrado.calcularHuella()
		degradado.AtestacionKMS.Perfil = perfil
		degradado.AtestacionKMS.HuellaEnvolturaSHA256 = degradado.EnvolturaClave.huellaSHA256
		degradado.AtestacionKMS.HuellaSobreSHA256 = degradado.SobreCifrado.huellaSHA256
		if degradado.validaPara(*e.cifrador.ultima) {
			t.Fatal("el cifrador pudo degradar el perfil gobernado de la solicitud")
		}
	})
	t.Run("texto cifrado", func(t *testing.T) {
		alterada := original
		alterada.Cifrado.SobreCifrado.textoCifrado = append(
			[]byte(nil), alterada.Cifrado.SobreCifrado.textoCifrado...,
		)
		alterada.Cifrado.SobreCifrado.textoCifrado[0] ^= 0xff
		if alterada.Validar() == nil {
			t.Fatal("se acepto texto cifrado alterado")
		}
	})
}

func TestTiposCifradoBloqueanCodecsYNoFiltranBuffers(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	confirmacion := *e.confirmador.ultima
	valores := []any{
		confirmacion.PerfilCifrado, SolicitudResolucionPerfilCifradoBorrador{},
		confirmacion.Cifrado.AAD, confirmacion.Cifrado.EnvolturaClave,
		confirmacion.Cifrado.SobreCifrado, confirmacion.Cifrado.AtestacionKMS,
		confirmacion.Cifrado, *e.cifrador.ultima,
	}
	for _, valor := range valores {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionDiarioProhibida) {
			t.Fatalf("%T permitio JSON: %v", valor, err)
		}
		texto := fmt.Sprintf("%+v", valor)
		if strings.Contains(texto, "A256GCM") || strings.Contains(texto, "clave:kms") {
			t.Fatalf("%T filtro material criptografico: %s", valor, texto)
		}
	}
	versionCanonica, err := e.cifrador.ultima.VersionCanonicaParaCifrado()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(fmt.Sprintf("%+v", *e.cifrador.ultima), string(versionCanonica)) {
		t.Fatal("la solicitud filtro el agregado en claro")
	}
	clear(versionCanonica)
}

func TestPersistenciaDebeRevalidarAtestacionKMSAutoritativa(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	confirmacion := *e.confirmador.ultima
	instante := confirmacion.SolicitadaEn.Add(time.Microsecond)
	solicitud, err := NuevaSolicitudRevalidacionAtestacionKMSBorrador(confirmacion, instante)
	if err != nil {
		t.Fatal(err)
	}
	resultado := ResultadoRevalidacionAtestacionKMSBorrador{
		AtestacionRef:     solicitud.AtestacionKMS.AtestacionRef,
		VersionAtestacion: solicitud.AtestacionKMS.VersionAtestacion,
		Estado:            estadoRevalidacionKMSAutorizada, HuellaAAD: solicitud.HuellaAAD,
		ComprobacionRef:          "comprobacion:kms:persistencia:001",
		HuellaComprobacionSHA256: huellaHexPrueba('f'), ComprobadaEn: instante,
	}
	if err := resultado.ValidarPara(solicitud); err != nil {
		t.Fatal(err)
	}
	resultado.HuellaAAD = huellaHexPrueba('0')
	if !errors.Is(resultado.ValidarPara(solicitud), ErrRevalidacionKMSBorradorFallo) {
		t.Fatal("la persistencia acepto una revalidacion KMS para otra AAD")
	}
}
